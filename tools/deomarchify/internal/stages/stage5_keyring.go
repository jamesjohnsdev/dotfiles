// Stage 5: remove omarchy-keyring. It's only relevant while pulling
// packages built against Omarchy's PKGBUILDs (validpgpkeys during an AUR
// build) or from an omarchy-controlled repo; once the other 3 packages are
// gone and pacman isn't pointed at an omarchy repo, it's dead weight.
package stages

import (
	"fmt"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/pacman"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func Stage5Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	for _, pkg := range []string{"omarchy", "omarchy-settings", "omarchy-nvim"} {
		installed, err := pacman.Installed(sys, pkg)
		if err != nil {
			return nil, fmt.Errorf("checking %s: %w", pkg, err)
		}
		if installed {
			return nil, fmt.Errorf("refusing to remove omarchy-keyring while %s is still installed - run earlier stages first", pkg)
		}
	}

	return removePackageSafely(sys, "omarchy-keyring")
}
