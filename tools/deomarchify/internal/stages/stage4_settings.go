// Stage 4: omarchy-settings owns real system tuning (sysctl, systemd
// drop-ins, docker/mkinitcpio/limine config, SDDM/Plymouth theme, fonts,
// sudoers) in addition to desktop glue. Removing the package would silently
// revert all of it to upstream Arch defaults at once. Instead: adopt every
// file as a plain, non-package-tracked copy at the same path first, so
// current behavior survives the removal untouched; only after that succeeds
// does the package actually get removed.
package stages

import (
	"fmt"
	"strings"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/pacman"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

type FileCategory int

const (
	// CategorySkip: irrelevant once not package-managed (first-login
	// templates under /etc/skel - only affect newly created users).
	CategorySkip FileCategory = iota
	// CategoryAdopt: copy forward as a plain file at the same path, so its
	// effect on the running system is unchanged after package removal.
	CategoryAdopt
	// CategoryUserServiceKeep: a systemd --user unit worth keeping; copied
	// to ~/.config/systemd/user/ instead of the package's
	// /usr/lib/systemd/user/, since it still calls the now-vendored scripts.
	CategoryUserServiceKeep
	// CategoryUserServiceDrop: a systemd --user unit tied to Omarchy's
	// update/migration machinery, with nothing to keep once that's gone.
	CategoryUserServiceDrop
)

// dropUserServices are unit basenames with no purpose once this system is
// no longer package-managed and auto-updating.
var dropUserServices = map[string]bool{
	"omarchy-migrate-notify.service":     true,
	"omarchy-update-user-notify.service": true,
}

// CategorizeSettingsFile decides what should happen to one file owned by
// the omarchy-settings package. Pure function, table-tested against the
// real `pacman -Ql omarchy-settings` output captured during investigation.
func CategorizeSettingsFile(path string) FileCategory {
	if strings.HasPrefix(path, "/etc/skel/") {
		return CategorySkip
	}
	if strings.HasPrefix(path, "/usr/lib/systemd/user/omarchy-") {
		base := path[strings.LastIndex(path, "/")+1:]
		if dropUserServices[base] {
			return CategoryUserServiceDrop
		}
		return CategoryUserServiceKeep
	}
	return CategoryAdopt
}

func Stage4Plan(sys system.System, cfg config.Config) ([]plan.Action, error) {
	_, ok, err := pacman.QueryInfo(sys, "omarchy-settings")
	if err != nil {
		return nil, fmt.Errorf("querying omarchy-settings: %w", err)
	}
	if !ok {
		return nil, nil // already removed
	}

	files, err := pacman.ListFiles(sys, "omarchy-settings")
	if err != nil {
		return nil, fmt.Errorf("listing omarchy-settings files: %w", err)
	}

	var adopt, keepUnits []string
	for _, f := range files {
		switch CategorizeSettingsFile(f) {
		case CategoryAdopt:
			adopt = append(adopt, f)
		case CategoryUserServiceKeep:
			keepUnits = append(keepUnits, f)
		}
	}

	var actions []plan.Action
	backupDir := cfg.Home + "/.deomarchify/settings-backup"

	// pacman -R deletes every package-tracked file unconditionally
	// (regardless of content) unless it's declared a %BACKUP% file in the
	// PKGBUILD, in which case a locally-modified copy gets renamed to
	// .pacsave instead of deleted - not something to rely on here without
	// checking every file individually. So: back up first, remove the
	// package (original paths disappear either way), then restore from the
	// backup to the original paths afterward. Rewriting the file in place
	// *before* removal (skipped here on purpose) would accomplish nothing,
	// since removal deletes it regardless of content.
	if len(adopt) > 0 {
		actions = append(actions, plan.Action{
			Description: fmt.Sprintf("back up %d omarchy-settings files (sysctl, systemd drop-ins, docker/mkinitcpio/limine/NetworkManager config, SDDM/Plymouth theme, fonts, sudoers, etc.) to %s before removal", len(adopt), backupDir),
			Apply: func(sys system.System) error {
				for _, f := range adopt {
					if err := sys.CopyFile(f, backupDir+f, 0o644); err != nil {
						return fmt.Errorf("backing up %s: %w", f, err)
					}
				}
				return nil
			},
		})
	}

	if len(keepUnits) > 0 {
		userServiceDir := cfg.Home + "/.config/systemd/user"
		actions = append(actions, plan.Action{
			Description: fmt.Sprintf("copy %d omarchy systemd --user units to %s (still call the now-vendored scripts)", len(keepUnits), userServiceDir),
			Apply: func(sys system.System) error {
				for _, f := range keepUnits {
					base := f[strings.LastIndex(f, "/")+1:]
					if err := sys.CopyFile(f, userServiceDir+"/"+base, 0o644); err != nil {
						return fmt.Errorf("copying %s: %w", f, err)
					}
				}
				return nil
			},
		})
		actions = append(actions, plan.Action{
			Description: "systemctl --user daemon-reload and re-enable the copied units",
			Apply: func(sys system.System) error {
				if res, err := sys.Run("systemctl", "--user", "daemon-reload"); err != nil || res.ExitCode != 0 {
					if err != nil {
						return err
					}
					return fmt.Errorf("daemon-reload failed: %s", res.Stderr)
				}
				for _, f := range keepUnits {
					base := f[strings.LastIndex(f, "/")+1:]
					res, err := sys.Run("systemctl", "--user", "enable", "--now", base)
					if err != nil {
						return err
					}
					if res.ExitCode != 0 {
						return fmt.Errorf("enabling %s failed: %s", base, res.Stderr)
					}
				}
				return nil
			},
		})
	}

	removal, err := removePackageSafely(sys, "omarchy-settings")
	if err != nil {
		return nil, err
	}
	actions = append(actions, removal...)

	if len(adopt) > 0 {
		actions = append(actions, plan.Action{
			Description: fmt.Sprintf("restore %d backed-up files to their original paths as plain, non-package-tracked files - preserves current system behavior exactly", len(adopt)),
			Destructive: true,
			Apply: func(sys system.System) error {
				for _, f := range adopt {
					if err := sys.CopyFile(backupDir+f, f, 0o644); err != nil {
						return fmt.Errorf("restoring %s: %w", f, err)
					}
				}
				return nil
			},
		})
	}

	return actions, nil
}
