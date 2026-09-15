// Package config holds the run-time configuration discovered or provided
// for a deomarchify run - deliberately not hardcoded, since this tool is
// meant to run on any Omarchy machine, not just the one it was written on.
package config

import (
	"os"
	"path/filepath"
)

// Config is everything stage Plan() functions need beyond the System.
type Config struct {
	Home string // $HOME
	// DotfilesDir is where the user's own dotfiles repo lives (used to
	// audit which configs reference omarchy-*). Defaults to $HOME/dotfiles.
	DotfilesDir string
	// VendorDir is where the omarchy source tree gets copied to, and what
	// OMARCHY_PATH points at from Stage 3 onward.
	VendorDir string
	// CheckpointPath is where stage-completion state is recorded.
	CheckpointPath string
	// LogDir is where per-stage logs are written.
	LogDir string

	// OmarchyPackages are the package names this tool knows how to remove,
	// in removal order (nvim first / safest, keyring last). Fixed by
	// upstream Omarchy naming, safe to hardcode across installs.
	OmarchyPackages []string
}

// Default builds a Config from $HOME with the tool's standard layout.
// Individual fields can be overridden (tests do this; so can CLI flags).
func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	base := filepath.Join(home, ".deomarchify")
	return Config{
		Home:           home,
		DotfilesDir:    filepath.Join(home, "dotfiles"),
		VendorDir:      filepath.Join(home, "dotfiles-omarchy"),
		CheckpointPath: filepath.Join(base, "checkpoint.json"),
		LogDir:         filepath.Join(base, "log"),
		OmarchyPackages: []string{
			"omarchy-nvim",
			"omarchy",
			"omarchy-settings",
			"omarchy-keyring",
		},
	}
}
