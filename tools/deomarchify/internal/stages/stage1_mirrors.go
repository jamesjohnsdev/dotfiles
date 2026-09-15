// Stage 1: point pacman at a real Arch mirror instead of Omarchy's.
package stages

import (
	"fmt"
	"strings"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

const mirrorlistPath = "/etc/pacman.d/mirrorlist"

// officialGeoMirror is Arch's own GeoIP-redirecting mirror
// (https://wiki.archlinux.org/title/Mirrors - listed as the always-working
// fallback), used instead of a country-specific reflector run so this stays
// correct on any machine without needing network-based mirror ranking.
const officialGeoMirror = "Server = https://geo.mirror.pkgbuild.com/$repo/os/$arch\n"

// NeedsMirrorFix reports whether the given mirrorlist content still points
// at an omarchy-controlled host.
func NeedsMirrorFix(mirrorlistContent string) bool {
	return strings.Contains(mirrorlistContent, "omarchy.org")
}

func Stage1Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	content, err := sys.ReadFile(mirrorlistPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", mirrorlistPath, err)
	}

	if !NeedsMirrorFix(content) {
		return nil, nil
	}

	backupPath := mirrorlistPath + ".pre-deomarchify.bak"

	return []plan.Action{
		{
			Description: fmt.Sprintf("back up %s to %s, then replace with the official Arch geo-mirror", mirrorlistPath, backupPath),
			Destructive: true,
			Apply: func(sys system.System) error {
				// /etc/pacman.d/mirrorlist is root-owned: both the backup
				// and the replacement need the privileged Sudo* variants,
				// not the plain ones (which run as the invoking user and
				// fail with a plain permission error here).
				if err := sys.SudoCopyFile(mirrorlistPath, backupPath); err != nil {
					return fmt.Errorf("backing up mirrorlist: %w", err)
				}
				if err := sys.SudoWriteFile(mirrorlistPath, officialGeoMirror, 0o644); err != nil {
					return fmt.Errorf("writing new mirrorlist: %w", err)
				}
				return nil
			},
		},
		{
			Description: "refresh pacman package databases against the new mirror (pacman -Syy)",
			Apply: func(sys system.System) error {
				res, err := sys.Run("sudo", "pacman", "-Syy", "--noconfirm")
				if err != nil {
					return err
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("pacman -Syy failed: %s", res.Stderr)
				}
				return nil
			},
		},
	}, nil
}
