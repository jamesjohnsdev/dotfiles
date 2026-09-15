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
	return os.WriteFile(path, []byte(content), perm)
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
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		return r.CopyFile(path, target, info.Mode())
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
	return os.WriteFile(dst, b, perm)
}

func (r *Real) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func (r *Real) Now() string {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now().Format(time.RFC3339)
}
