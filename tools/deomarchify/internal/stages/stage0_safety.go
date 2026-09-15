// Stage 0: safety net. Take a snapper snapshot and mark every real
// dependency of the omarchy packages as explicitly installed, so a later
// `pacman -R` of the omarchy packages can never sweep up Hyprland, SDDM,
// pipewire etc as "orphaned".
package stages

import (
	"fmt"
	"sort"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/pacman"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// DependenciesToProtect returns the deduplicated, sorted set of packages
// depended on by any of the given omarchy packages, excluding the omarchy
// packages themselves. These are the packages that must be marked
// explicitly-installed before removing any omarchy package, or pacman's
// orphan sweep (`-Rns`) - or even an unrelated later `pacman -Rs` - could
// remove them as no-longer-needed.
//
// Pure function: takes already-fetched Info values so it's testable without
// a System.
func DependenciesToProtect(infos []pacman.Info, omarchyPackages []string) []string {
	omarchy := map[string]bool{}
	for _, p := range omarchyPackages {
		omarchy[p] = true
	}

	seen := map[string]bool{}
	for _, info := range infos {
		for _, dep := range info.Depends {
			if omarchy[dep] {
				continue
			}
			seen[dep] = true
		}
	}

	out := make([]string, 0, len(seen))
	for dep := range seen {
		out = append(out, dep)
	}
	sort.Strings(out)
	return out
}

// Stage0Plan builds the safety-net actions: snapshot + mark-explicit.
func Stage0Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	var actions []plan.Action

	var infos []pacman.Info
	for _, pkg := range cfg.OmarchyPackages {
		info, ok, err := pacman.QueryInfo(sys, pkg)
		if err != nil {
			return nil, fmt.Errorf("querying %s: %w", pkg, err)
		}
		if ok {
			infos = append(infos, info)
		}
	}

	protect := DependenciesToProtect(infos, cfg.OmarchyPackages)

	actions = append(actions, plan.Action{
		Description: "snapshot root filesystem via snapper before any changes",
		Apply: func(sys system.System) error {
			res, err := sys.Run("sudo", "snapper", "create", "--description", "pre-deomarchify")
			if err != nil {
				return err
			}
			if res.ExitCode != 0 {
				return fmt.Errorf("snapper create failed: %s", res.Stderr)
			}
			return nil
		},
	})

	if len(protect) > 0 {
		desc := fmt.Sprintf("mark %d real OS/DE packages as explicitly installed (protects them from orphan removal): %v", len(protect), protect)
		actions = append(actions, plan.Action{
			Description: desc,
			Apply: func(sys system.System) error {
				args := append([]string{"-D", "--asexplicit"}, protect...)
				res, err := sys.Run("sudo", append([]string{"pacman"}, args...)...)
				if err != nil {
					return err
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("pacman -D --asexplicit failed: %s", res.Stderr)
				}
				return nil
			},
		})
	}

	return actions, nil
}
