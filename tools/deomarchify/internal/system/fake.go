package system

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Fake is an in-memory System for tests: no real command execution, no real
// filesystem access. Stage Plan() logic is tested against this instead of
// the live machine.
type Fake struct {
	// Commands maps "name arg1 arg2" -> canned result. Set these up in tests
	// to simulate pacman/systemctl/etc output.
	Commands map[string]CommandResult
	// Ran records every command actually invoked, in order, for assertions.
	Ran []string

	Files    map[string]string      // path -> content
	Symlinks map[string]string      // path -> target
	Perms    map[string]os.FileMode // path -> perm, informational

	clock string
}

func NewFake() *Fake {
	return &Fake{
		Commands: map[string]CommandResult{},
		Files:    map[string]string{},
		Symlinks: map[string]string{},
		Perms:    map[string]os.FileMode{},
		clock:    "2026-09-15T00:00:00Z",
	}
}

func key(name string, args ...string) string {
	return strings.TrimSpace(name + " " + strings.Join(args, " "))
}

// SetCommand registers the canned response for a given command invocation.
func (f *Fake) SetCommand(result CommandResult, name string, args ...string) {
	f.Commands[key(name, args...)] = result
}

func (f *Fake) Run(name string, args ...string) (CommandResult, error) {
	k := key(name, args...)
	f.Ran = append(f.Ran, k)
	if res, ok := f.Commands[k]; ok {
		return res, nil
	}
	return CommandResult{}, fmt.Errorf("fake system: no canned response for %q", k)
}

func (f *Fake) ReadFile(path string) (string, error) {
	if c, ok := f.Files[path]; ok {
		return c, nil
	}
	return "", os.ErrNotExist
}

func (f *Fake) WriteFile(path string, content string, perm os.FileMode) error {
	f.Files[path] = content
	f.Perms[path] = perm
	return nil
}

func (f *Fake) FileExists(path string) bool {
	if _, ok := f.Files[path]; ok {
		return true
	}
	_, ok := f.Symlinks[path]
	return ok
}

func (f *Fake) IsSymlink(path string) bool {
	_, ok := f.Symlinks[path]
	return ok
}

func (f *Fake) Readlink(path string) (string, error) {
	if t, ok := f.Symlinks[path]; ok {
		return t, nil
	}
	return "", fmt.Errorf("not a symlink: %s", path)
}

func (f *Fake) Symlink(target, linkPath string) error {
	f.Symlinks[linkPath] = target
	return nil
}

func (f *Fake) RemoveSymlink(path string) error {
	if _, ok := f.Symlinks[path]; !ok {
		return fmt.Errorf("refusing to remove %s: not a symlink", path)
	}
	delete(f.Symlinks, path)
	return nil
}

func (f *Fake) Remove(path string) error {
	if _, ok := f.Files[path]; !ok {
		return os.ErrNotExist
	}
	delete(f.Files, path)
	return nil
}

func (f *Fake) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (f *Fake) CopyTree(src, dst string) error {
	prefix := src
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for path, content := range f.copySourceFiles(prefix) {
		rel := strings.TrimPrefix(path, prefix)
		f.Files[dst+"/"+rel] = content
	}
	return nil
}

func (f *Fake) copySourceFiles(prefix string) map[string]string {
	out := map[string]string{}
	for path, content := range f.Files {
		if strings.HasPrefix(path, prefix) {
			out[path] = content
		}
	}
	return out
}

func (f *Fake) CopyFile(src, dst string, perm os.FileMode) error {
	c, ok := f.Files[src]
	if !ok {
		return fmt.Errorf("fake system: copy source missing: %s", src)
	}
	f.Files[dst] = c
	f.Perms[dst] = perm
	return nil
}

// Glob does simple filepath.Match against every known path (Files and
// Symlinks), sufficient for the glob patterns this tool actually uses.
func (f *Fake) Glob(pattern string) ([]string, error) {
	seen := map[string]bool{}
	for path := range f.Files {
		seen[path] = true
	}
	for path := range f.Symlinks {
		seen[path] = true
	}
	var out []string
	for path := range seen {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			return nil, err
		}
		if matched {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (f *Fake) Now() string {
	return f.clock
}

// SortedRan returns Ran sorted, handy for order-independent assertions.
func (f *Fake) SortedRan() []string {
	out := append([]string{}, f.Ran...)
	sort.Strings(out)
	return out
}
