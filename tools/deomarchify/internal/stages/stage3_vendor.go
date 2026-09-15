// Stage 3: vendor /usr/share/omarchy into a self-owned checkout, point the
// user's own env bootstrap (uwsm env.d, bashrc, zshrc) at it instead of the
// package, then remove the omarchy package itself.
package stages

import (
	"fmt"
	"strings"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/envfiles"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

const packagedOmarchyPath = "/usr/share/omarchy"

// VendorSubdirs are the parts of /usr/share/omarchy worth vendoring.
// install/ and migrations/ are deliberately excluded: they're package
// lifecycle machinery (build/update/migrate tooling) with no purpose once
// this stops being a package - keeping them would just carry forward the
// self-update chain we're trying to get away from.
var VendorSubdirs = []string{"bin", "default", "themes", "config", "applications", "shell", "etc-overrides"}

func uwsmEnvDPath(home string) string {
	return home + "/.config/uwsm/env.d/10-own-omarchy"
}

func bashrcPath(home string) string            { return home + "/.bashrc" }
func zshrcPath(home string) string             { return home + "/.zshrc" }
func localShareOmarchyLink(home string) string { return home + "/.local/share/omarchy" }

func Stage3Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	var actions []plan.Action

	for _, sub := range VendorSubdirs {
		src := packagedOmarchyPath + "/" + sub
		dst := cfg.VendorDir + "/" + sub
		actions = append(actions, plan.Action{
			Description: fmt.Sprintf("copy %s -> %s", src, dst),
			Apply: func(sys system.System) error {
				return sys.CopyTree(src, dst)
			},
		})
	}

	actions = append(actions, plan.Action{
		Description: "neutralize self-update/migrate scripts in the vendored checkout (omarchy-update*, omarchy-migrate*, omarchy-refresh-pacman, omarchy-reinstall*) so it never tries to pacman-update or migrate itself",
		Apply: func(sys system.System) error {
			matches, err := sys.Glob(cfg.VendorDir + "/bin/omarchy-*")
			if err != nil {
				return err
			}
			for _, path := range matches {
				name := path[strings.LastIndex(path, "/")+1:]
				if !IsSelfUpdateScript(name) {
					continue
				}
				stub := fmt.Sprintf("#!/bin/sh\necho %q >&2\nexit 1\n", name+": disabled by deomarchify - this checkout is no longer package-managed, this command has no meaning here")
				if err := sys.WriteFile(path, stub, 0o755); err != nil {
					return fmt.Errorf("neutralizing %s: %w", path, err)
				}
			}
			return nil
		},
	})

	uwsmPath := uwsmEnvDPath(cfg.Home)
	actions = append(actions, plan.Action{
		Description: fmt.Sprintf("write %s (own OMARCHY_PATH/PATH bootstrap for the uwsm/Hyprland session, replaces package-owned /usr/share/uwsm/env.d/10-omarchy)", uwsmPath),
		Apply: func(sys system.System) error {
			return sys.WriteFile(uwsmPath, envfiles.UWSMEnvFile(cfg.VendorDir), 0o644)
		},
	})

	bp := bashrcPath(cfg.Home)
	actions = append(actions, plan.Action{
		Description: fmt.Sprintf("patch %s to source the vendored env-bootstrap instead of the hardcoded /usr/share/omarchy path", bp),
		Apply: func(sys system.System) error {
			original, err := sys.ReadFile(bp)
			if err != nil {
				return err
			}
			patched, changed := envfiles.PatchBashrc(original, cfg.VendorDir)
			if !changed {
				return fmt.Errorf("expected line not found in %s - bashrc may have already been edited; check manually before re-running", bp)
			}
			return sys.WriteFile(bp, patched, 0o644)
		},
	})

	zp := zshrcPath(cfg.Home)
	actions = append(actions, plan.Action{
		Description: fmt.Sprintf("patch %s: export OMARCHY_PATH to point at the vendored checkout", zp),
		Apply: func(sys system.System) error {
			original, err := sys.ReadFile(zp)
			if err != nil {
				return err
			}
			patched, changed := envfiles.PatchZshrc(original, cfg.VendorDir)
			if !changed {
				return fmt.Errorf("expected export line not found in %s - check manually before re-running", zp)
			}
			return sys.WriteFile(zp, patched, 0o644)
		},
	})

	linkPath := localShareOmarchyLink(cfg.Home)
	actions = append(actions, plan.Action{
		Description: fmt.Sprintf("repoint symlink %s -> %s (was -> %s)", linkPath, cfg.VendorDir, packagedOmarchyPath),
		Apply: func(sys system.System) error {
			if sys.IsSymlink(linkPath) {
				if err := sys.RemoveSymlink(linkPath); err != nil {
					return err
				}
			}
			return sys.Symlink(cfg.VendorDir, linkPath)
		},
	})

	removal, err := removePackageSafely(sys, "omarchy")
	if err != nil {
		return nil, err
	}
	actions = append(actions, removal...)

	return actions, nil
}

// neutralizeSelfUpdateScripts lists the vendored bin/ scripts that should be
// replaced with a stub refusing to run, so the checkout never tries to
// pacman-update or migrate itself (that machinery assumes it's still the
// package). Exposed as data so it's covered by a table test rather than
// buried in Apply closures.
var SelfUpdateScriptPrefixes = []string{
	"omarchy-update",
	"omarchy-migrate",
	"omarchy-refresh-pacman",
	"omarchy-reinstall",
}

// IsSelfUpdateScript reports whether a vendored bin/ filename should be
// neutralized per SelfUpdateScriptPrefixes.
func IsSelfUpdateScript(filename string) bool {
	for _, prefix := range SelfUpdateScriptPrefixes {
		if strings.HasPrefix(filename, prefix) {
			return true
		}
	}
	return false
}
