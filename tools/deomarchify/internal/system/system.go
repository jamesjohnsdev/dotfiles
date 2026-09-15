// Package system abstracts every OS interaction (running commands, touching
// files) behind an interface so stage planning logic can be unit tested
// against a fake implementation instead of a real machine.
package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandResult holds the outcome of a command run via System.Run.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// System is everything a stage needs from the outside world. Implement it
// once for real (Real) and once in-memory for tests (see system/fake).
type System interface {
	// Run executes name with args and returns combined result. It does not
	// return an error for a non-zero exit; callers check ExitCode/inspect
	// output, since many checks here are "does this command fail" checks.
	Run(name string, args ...string) (CommandResult, error)

	ReadFile(path string) (string, error)
	WriteFile(path string, content string, perm os.FileMode) error
	FileExists(path string) bool
	IsSymlink(path string) bool
	Readlink(path string) (string, error)
	Symlink(target, linkPath string) error
	RemoveSymlink(path string) error
	Remove(path string) error
	MkdirAll(path string, perm os.FileMode) error
	CopyTree(src, dst string) error
	CopyFile(src, dst string, perm os.FileMode) error
	// Glob returns paths matching a filepath.Match-style pattern.
	Glob(pattern string) ([]string, error)

	// Sudo* variants run as root via `sudo`, for paths the invoking user
	// can't write (or, for SudoCopyFile, can't necessarily even read - e.g.
	// mode-0440 sudoers.d files) - most of /etc and /usr. Every other
	// method above operates as the invoking user; using the wrong one for
	// a root-owned path fails with a plain permission error rather than
	// prompting for a password, which is what surfaced this split in the
	// first place. SudoCopyFile uses `cp -a` specifically to preserve the
	// source's mode/ownership rather than imposing a fixed one - some
	// adopted system files (sudoers.d) are silently ignored by their
	// consumer if their permissions change.
	SudoWriteFile(path string, content string, perm os.FileMode) error
	SudoCopyFile(src, dst string) error
	SudoSymlink(target, linkPath string) error
	SudoRemove(path string) error
	SudoMkdirAll(path string) error

	// Now returns the current time as an RFC3339 string, for log/checkpoint
	// timestamps. Abstracted so tests get deterministic output.
	Now() string
}

// Real is the production System backed by the actual OS and exec.Command.
type Real struct {
	Clock func() string
}

func NewReal() *Real {
	return &Real{}
}

func (r *Real) Run(name string, args ...string) (CommandResult, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := CommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.ExitCode = exitErr.ExitCode()
		return res, nil
	}
	if err != nil {
		return res, err
	}
	return res, nil
}

func (r *Real) ReadFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *Real) WriteFile(path string, content string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeFileForce(path, []byte(content), perm)
}

// writeFileForce writes path even when it already exists with a mode
// lacking the owner-write bit - which is exactly what a file copied out of
// a package directory looks like (installed scripts are commonly mode 0555,
// no write bit at all). os.WriteFile ignores its perm argument for a file
// that already exists (perm only applies at creation), so a plain
// os.WriteFile fails with a permission error in that case despite the
// caller owning the file. Chmod first (best-effort: an ENOENT here just
// means the file doesn't exist yet, which is fine) so the write - and any
// later write to the same path - succeeds regardless of what mode the
// source file had.
func writeFileForce(path string, data []byte, perm os.FileMode) error {
	// If path is currently a symlink (e.g. left over from an older,
	// pre-fix vendoring attempt that recreated symlinks instead of
	// dereferencing them), os.Chmod would follow it and try to chmod
	// whatever it points at - which, for a symlink into /usr/bin/..., is a
	// root-owned file the invoking user can't chmod (EPERM, confirmed on a
	// real machine). Remove it first so what follows always creates a real
	// file at path, never chmods through a stale symlink.
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(path); err != nil {
			return err
		}
	} else if err := os.Chmod(path, perm); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, data, perm)
}

func (r *Real) FileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func (r *Real) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func (r *Real) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

func (r *Real) Symlink(target, linkPath string) error {
	return os.Symlink(target, linkPath)
}

func (r *Real) RemoveSymlink(path string) error {
	if !r.IsSymlink(path) {
		return fmt.Errorf("refusing to remove %s: not a symlink", path)
	}
	return os.Remove(path)
}

func (r *Real) Remove(path string) error {
	return os.Remove(path)
}

func (r *Real) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *Real) CopyTree(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil // optional subtree not present on this install, fine
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		// Dereference symlinks into real copies rather than recreating
		// them: confirmed on a real machine that every one of
		// /usr/share/omarchy/bin's 428 entries is a symlink into
		// /usr/bin/omarchy-* (owned by the very package this tree is being
		// vendored away from). Preserving the symlink as-is would leave
		// the "vendored" checkout still silently dependent on the package,
		// and dangling the moment it's removed - the opposite of the
		// point of this step. os.ReadFile below (inside CopyFile) follows
		// symlinks transparently, so this just resolves the real mode and
		// falls through to the regular-file copy path.
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			resolved, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("resolving symlink %s: %w", path, err)
			}
			mode = resolved.Mode()
		}
		// Add the owner-write bit rather than mirroring the source mode
		// exactly: installed package files are commonly 0555 (no write bit
		// at all), and the whole point of vendoring is handing over a
		// checkout the user can actually edit, not a faithful read-only
		// replica of the package.
		return r.CopyFile(path, target, mode|0o200)
	})
}

func (r *Real) CopyFile(src, dst string, perm os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return writeFileForce(dst, b, perm)
}

func (r *Real) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

// runSudo is Run with a "sudo " prefix and a uniform error-or-nonzero-exit
// check, since every Sudo* method below needs exactly that.
func (r *Real) runSudo(args ...string) error {
	res, err := r.Run("sudo", args...)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("sudo %s failed: %s", strings.Join(args, " "), res.Stderr)
	}
	return nil
}

func (r *Real) SudoWriteFile(path string, content string, perm os.FileMode) error {
	tmp, err := os.CreateTemp("", "deomarchify-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := r.runSudo("mkdir", "-p", filepath.Dir(path)); err != nil {
		return err
	}
	return r.runSudo("install", "-m", fmt.Sprintf("%03o", perm), tmpPath, path)
}

func (r *Real) SudoCopyFile(src, dst string) error {
	if err := r.runSudo("mkdir", "-p", filepath.Dir(dst)); err != nil {
		return err
	}
	// -a preserves mode/ownership/timestamps from src, which matters for
	// files like /etc/sudoers.d/* that sudo refuses to read at all if
	// their permissions change.
	return r.runSudo("cp", "-a", src, dst)
}

func (r *Real) SudoSymlink(target, linkPath string) error {
	return r.runSudo("ln", "-sfn", target, linkPath)
}

func (r *Real) SudoRemove(path string) error {
	// -r as well as -f: some of what this removes turns out to be a
	// directory, not a file (confirmed on a real machine - an
	// omarchy-upgrade-to-quattro backup of a directory-based app config,
	// e.g. mako's), and plain `rm -f` refuses a directory outright.
	return r.runSudo("rm", "-rf", path)
}

func (r *Real) SudoMkdirAll(path string) error {
	return r.runSudo("mkdir", "-p", path)
}

func (r *Real) Now() string {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now().Format(time.RFC3339)
}
