// Stage 2: remove omarchy-nvim. It only ships /etc/skel/.config/nvim (a
// first-login template) - a real dotfiles-managed ~/.config/nvim is
// untouched by this, so removal is a no-op at runtime. Still guarded by a
// RequiredBy check like every other removal in this tool.
package stages

import (
	"fmt"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/pacman"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func Stage2Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	return removePackageSafely(sys, "omarchy-nvim")
}

// removePackageSafely is shared by every stage that removes one omarchy
// package: skip if not installed, refuse if something unexpected depends on
// it (a real dependent would mean this machine differs from what this tool
// assumes), otherwise plan a plain `pacman -R` (no -s, so it can never
// sweep up unrelated packages regardless of what Stage 0 already protected).
func removePackageSafely(sys system.System, pkg string) ([]plan.Action, error) {
	info, ok, err := pacman.QueryInfo(sys, pkg)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", pkg, err)
	}
	if !ok {
		return nil, nil // already removed, nothing to do
	}
	if len(info.RequiredBy) > 0 {
		return nil, fmt.Errorf("refusing to remove %s: still required by %v (this machine differs from what this tool expects - investigate before removing manually)", pkg, info.RequiredBy)
	}

	return []plan.Action{
		{
			Description: fmt.Sprintf("remove package %s (pacman -R, no dependency sweep)", pkg),
			Destructive: true,
			Apply: func(sys system.System) error {
				res, err := sys.Run("sudo", "pacman", "-R", "--noconfirm", pkg)
				if err != nil {
					return err
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("pacman -R %s failed: %s", pkg, res.Stderr)
				}
				return nil
			},
		},
	}, nil
}
