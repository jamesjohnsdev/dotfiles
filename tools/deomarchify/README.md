# deomarchify

Converts an Omarchy install to plain Arch + Hyprland: vendors
`/usr/share/omarchy` into a self-owned checkout you control, points your own
env bootstrap (uwsm `env.d`, `.bashrc`, `.zshrc`) at it instead of the
package, adopts `omarchy-settings`'s system config (sysctl, systemd
drop-ins, docker/mkinitcpio/limine, SDDM/Plymouth theme, fonts, sudoers...)
as plain files so removing the package doesn't silently revert any of it,
then removes `omarchy`, `omarchy-settings`, `omarchy-nvim` and
`omarchy-keyring`.

Full rationale per stage: see `../../.claude/plans/transient-meandering-island.md`
in the session this was built from, or read the doc comment at the top of
each `internal/stages/stageN_*.go` file - they're kept in sync with the
actual logic, the plan file is not.

## Design

- **Staged, idempotent, checkpointed.** Each stage only plans and applies
  what isn't already done; re-running after fixing a problem skips
  completed stages and safely re-attempts the failed one.
- **Dry-run by default.** Nothing happens without `-apply`. Read the plan
  output first.
- **No custom rollback.** Recovery is either "fix it and re-run" (stages
  are idempotent) or `sudo snapper rollback` to the Stage 0 snapshot for
  anything more serious - see the discussion in the plan doc for why a
  hand-rolled undo of a completed `pacman -R` would be false confidence.
- **Portable across Omarchy installs.** Nothing here is hardcoded to one
  machine: the set of packages to protect from orphan removal is read live
  from `pacman -Qi omarchy`'s Depends line, and the omarchy-settings files
  to adopt come from `pacman -Ql omarchy-settings`, not a fixed list.
- **Testable by construction.** Every OS interaction (running commands,
  touching files) goes through the `system.System` interface
  (`internal/system`); stage `Plan()` functions are pure given a `System`,
  so decision logic is tested against `system.Fake` with canned pacman
  output instead of a real machine. Only the thin `system.Real` wrapper and
  `main.go`'s flag plumbing are untested by unit tests - exercised instead
  by running the built binary in dry-run mode (see below), which is safe:
  every stage's `Plan()` only ever *reads* (pacman queries, file reads,
  globs); all mutation happens in `Apply`, which dry-run never calls.

## Usage

```sh
go build -o deomarchify ./cmd/deomarchify

./deomarchify                  # dry-run everything, in order
./deomarchify -stage 3         # dry-run just stage 3
./deomarchify -apply           # actually run everything, in order
./deomarchify -apply -stage 3  # actually run just stage 3
./deomarchify -apply -force -stage 2   # re-run a stage even if already done
./deomarchify -status          # show which stages have completed
./deomarchify -list            # list stage numbers/names
```

**Run this stage by stage, rebooting between stages 3, 4, and 5** - env/uwsm/
systemd-unit changes need a real session restart to take effect, not just a
Hyprland reload. Read the plan output before passing `-apply`.

## Testing

```sh
go test ./...          # unit tests, no real machine touched
go test ./... -cover   # coverage per package
```

`internal/system`, `internal/config`, and `cmd/deomarchify` show 0% unit
coverage by design (real OS/exec wrappers and flag glue) - they're
validated by actually running the built binary without `-apply`, which is
read-only end to end.
