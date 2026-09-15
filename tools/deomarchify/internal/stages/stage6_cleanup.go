// Stage 6: remove leftover cruft from the pre-quattro upgrade and the
// package cache. Deliberately does NOT touch ~/.local/state/omarchy - that
// holds live state (current theme, clipboard history, toggles) the vendored
// scripts keep reading and writing, not package cruft.
package stages

import (
	"fmt"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// CleanupGlobs are the glob patterns swept in Stage 6. Kept as data so the
// "what gets touched" set is table-tested, and so ~/.local/state/omarchy
// staying absent from this list is enforced by a test rather than a comment.
func CleanupGlobs(home string) []string {
	return []string{
		home + "/.config/*.omarchy-upgrade-to-quattro.*.bak",
		home + "/.bashrc.omarchy-upgrade-to-quattro.*.bak",
		"/etc/pacman.d/mirrorlist.omarchy-upgrade-to-quattro.*.bak",
	}
}

func Stage6Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	var actions []plan.Action

	for _, pattern := range CleanupGlobs(cfg.Home) {
		matches, err := sys.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("globbing %s: %w", pattern, err)
		}
		for _, path := range matches {
			p := path
			actions = append(actions, plan.Action{
				Description: fmt.Sprintf("remove leftover upgrade-backup file %s", p),
				Apply: func(sys system.System) error {
					return sys.Remove(p)
				},
			})
		}
	}

	actions = append(actions, plan.Action{
		Description: fmt.Sprintf("remove regenerable cache dir %s/.cache/omarchy", cfg.Home),
		Apply: func(sys system.System) error {
			res, err := sys.Run("rm", "-rf", cfg.Home+"/.cache/omarchy")
			if err != nil {
				return err
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("rm -rf failed: %s", res.Stderr)
			}
			return nil
		},
	})

	return actions, nil
}
