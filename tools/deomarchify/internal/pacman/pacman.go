// Package pacman queries and parses pacman output. Parsing is pure
// (string in, struct out) so it's unit tested with fixture text, independent
// of the system.System abstraction used for the actual command execution.
package pacman

import (
	"strings"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// Info is the subset of `pacman -Qi <pkg>` this tool cares about.
type Info struct {
	Name        string
	Repository  string // empty when not from a synced repo (AUR/local build)
	Depends     []string
	RequiredBy  []string
	OptionalFor []string
}

// Installed reports whether a package is currently installed.
func Installed(sys system.System, pkg string) (bool, error) {
	res, err := sys.Run("pacman", "-Qi", pkg)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// QueryInfo runs and parses `pacman -Qi <pkg>`. Returns ok=false if the
// package isn't installed (exit code non-zero), which is not an error.
func QueryInfo(sys system.System, pkg string) (Info, bool, error) {
	res, err := sys.Run("pacman", "-Qi", pkg)
	if err != nil {
		return Info{}, false, err
	}
	if res.ExitCode != 0 {
		return Info{}, false, nil
	}
	return ParseQi(res.Stdout), true, nil
}

// ListFiles runs and parses `pacman -Ql <pkg>`, returning absolute file
// paths (directories excluded).
func ListFiles(sys system.System, pkg string) ([]string, error) {
	res, err := sys.Run("pacman", "-Ql", pkg)
	if err != nil {
		return nil, err
	}
	return ParseQl(res.Stdout), nil
}

// ParseQi parses the field: value block format of `pacman -Qi` output.
// Multi-value fields (Depends On, Required By, Optional For) are
// space-separated on one line, "None" meaning empty.
func ParseQi(output string) Info {
	info := Info{}
	for _, line := range strings.Split(output, "\n") {
		field, value, ok := splitField(line)
		if !ok {
			continue
		}
		switch field {
		case "Name":
			info.Name = strings.TrimSpace(value)
		case "Repository":
			info.Repository = strings.TrimSpace(value)
		case "Depends On":
			info.Depends = parseList(value)
		case "Required By":
			info.RequiredBy = parseList(value)
		case "Optional For":
			info.OptionalFor = parseList(value)
		}
	}
	return info
}

// splitField splits a "Field       : value" line. pacman's field column is
// fixed-width padded with spaces before the ":".
func splitField(line string) (field, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	f := strings.TrimSpace(line[:idx])
	if f == "" {
		return "", "", false
	}
	return f, line[idx+1:], true
}

func parseList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "None" {
		return nil
	}
	fields := strings.Fields(value)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		// Depends On entries can carry a version constraint like
		// "omarchy-settings=4.0.2" - strip it, callers want bare pkg names.
		if i := strings.IndexAny(f, "=<>"); i >= 0 {
			f = f[:i]
		}
		out = append(out, f)
	}
	return out
}

// ParseQl parses `pacman -Ql` output ("pkgname /path/to/file") into a plain
// list of paths, dropping directory entries (paths ending in "/").
func ParseQl(output string) []string {
	var out []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		path := strings.TrimSpace(parts[1])
		if strings.HasSuffix(path, "/") {
			continue
		}
		out = append(out, path)
	}
	return out
}
