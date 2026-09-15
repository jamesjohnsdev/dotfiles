package stages

import (
	"strings"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestCategorizeSettingsFile(t *testing.T) {
	cases := map[string]FileCategory{
		"/etc/skel/.config/nvim/init.lua":                          CategorySkip,
		"/etc/skel/.local/state/omarchy/toggles/x.lua":             CategorySkip,
		"/usr/lib/systemd/user/omarchy-migrate-notify.service":     CategoryUserServiceDrop,
		"/usr/lib/systemd/user/omarchy-update-user-notify.service": CategoryUserServiceDrop,
		"/usr/lib/systemd/user/omarchy-fcitx5.service":             CategoryUserServiceKeep,
		"/usr/lib/systemd/user/omarchy-sleep-lock.service":         CategoryUserServiceKeep,
		"/usr/lib/systemd/user/bt-agent.service":                   CategoryAdopt, // not an omarchy- unit
		"/etc/sysctl.d/99-omarchy-sysctl.conf":                     CategoryAdopt,
		"/etc/systemd/logind.conf.d/10-ignore-power-button.conf":   CategoryAdopt,
		"/usr/share/sddm/themes/omarchy/theme.conf":                CategoryAdopt,
		"/etc/sudoers.d/omarchy-dns":                               CategoryAdopt,
	}
	for path, want := range cases {
		if got := CategorizeSettingsFile(path); got != want {
			t.Errorf("CategorizeSettingsFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestStage4Plan_NoopWhenNotInstalled(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 1}, "pacman", "-Qi", "omarchy-settings")

	actions, err := Stage4Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected no actions, got %d", len(actions))
	}
}

func TestStage4Plan_BackupHappensBeforeRemovalAndRestoreAfter(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{Home: "/home/james"}

	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy-settings\nRequired By : None\n"}, "pacman", "-Qi", "omarchy-settings")
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: strings.Join([]string{
		"omarchy-settings /etc/sysctl.d/99-omarchy-sysctl.conf",
		"omarchy-settings /usr/lib/systemd/user/omarchy-fcitx5.service",
		"omarchy-settings /usr/lib/systemd/user/omarchy-migrate-notify.service",
		"omarchy-settings /etc/skel/.config/nvim/init.lua",
		"",
	}, "\n")}, "pacman", "-Ql", "omarchy-settings")

	actions, err := Stage4Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backupIdx, removeIdx, restoreIdx := -1, -1, -1
	for i, a := range actions {
		switch {
		case strings.HasPrefix(a.Description, "back up"):
			backupIdx = i
		case strings.HasPrefix(a.Description, "remove package omarchy-settings"):
			removeIdx = i
		case strings.HasPrefix(a.Description, "restore"):
			restoreIdx = i
		}
	}
	if backupIdx == -1 || removeIdx == -1 || restoreIdx == -1 {
		t.Fatalf("expected backup, remove, and restore actions all present; got:\n%s", describeActions(actions))
	}
	if !(backupIdx < removeIdx && removeIdx < restoreIdx) {
		t.Fatalf("expected order backup < remove < restore, got indices %d %d %d:\n%s", backupIdx, removeIdx, restoreIdx, describeActions(actions))
	}

	// Simulate: backup runs, file physically disappears (package removal),
	// restore must bring it back from the backup copy, not the original.
	fake.Files["/etc/sysctl.d/99-omarchy-sysctl.conf"] = "tuning=1\n"
	if err := actions[backupIdx].Apply(fake); err != nil {
		t.Fatalf("backup apply: %v", err)
	}
	delete(fake.Files, "/etc/sysctl.d/99-omarchy-sysctl.conf")

	if err := actions[restoreIdx].Apply(fake); err != nil {
		t.Fatalf("restore apply: %v", err)
	}
	if fake.Files["/etc/sysctl.d/99-omarchy-sysctl.conf"] != "tuning=1\n" {
		t.Errorf("file not restored correctly, got %q", fake.Files["/etc/sysctl.d/99-omarchy-sysctl.conf"])
	}

	// Both the backup (source may be unreadable to the invoking user, e.g.
	// mode-0440 sudoers.d files) and the restore (destination is a
	// root-owned /etc path) must go through the privileged path - confirmed
	// on a real machine that the plain CopyFile variant fails here.
	sawSudoBackup, sawSudoRestore := false, false
	for _, r := range fake.Ran {
		if r == "sudo:copy-file /etc/sysctl.d/99-omarchy-sysctl.conf /home/james/.deomarchify/settings-backup/etc/sysctl.d/99-omarchy-sysctl.conf" {
			sawSudoBackup = true
		}
		if r == "sudo:copy-file /home/james/.deomarchify/settings-backup/etc/sysctl.d/99-omarchy-sysctl.conf /etc/sysctl.d/99-omarchy-sysctl.conf" {
			sawSudoRestore = true
		}
	}
	if !sawSudoBackup {
		t.Errorf("expected a privileged (sudo) backup copy, got Ran=%v", fake.Ran)
	}
	if !sawSudoRestore {
		t.Errorf("expected a privileged (sudo) restore copy, got Ran=%v", fake.Ran)
	}

	// /etc/skel/... must never appear in the adopt set.
	for _, a := range actions {
		if strings.Contains(a.Description, "etc/skel") {
			t.Errorf("skel path leaked into an action description: %s", a.Description)
		}
	}
}

func TestRewriteUnitExecPaths(t *testing.T) {
	content := `[Unit]
Description=Lock Omarchy before suspend

[Service]
ExecStart=/usr/bin/omarchy-system-sleep-monitor
Restart=always
`
	got := RewriteUnitExecPaths(content, "/home/james/dotfiles-omarchy")
	want := `[Unit]
Description=Lock Omarchy before suspend

[Service]
ExecStart=/home/james/dotfiles-omarchy/bin/omarchy-system-sleep-monitor
Restart=always
`
	if got != want {
		t.Errorf("RewriteUnitExecPaths =\n%s\nwant\n%s", got, want)
	}
}

// This reproduces the exact real-world failure: a kept systemd --user unit
// copied verbatim still points ExecStart at /usr/bin/omarchy-crash-watch,
// which Stage 3 already deleted (owned by the omarchy package) - the unit
// crash-loops (203/EXEC) the instant it starts. The copy action must rewrite
// ExecStart to the vendored path, not just relocate the unit file itself.
func TestStage4Plan_KeptUnitExecStartPointsAtVendoredPath(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{Home: "/home/james", VendorDir: "/home/james/dotfiles-omarchy"}

	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy-settings\nRequired By : None\n"}, "pacman", "-Qi", "omarchy-settings")
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "omarchy-settings /usr/lib/systemd/user/omarchy-crash-watch.service\n"}, "pacman", "-Ql", "omarchy-settings")
	fake.Files["/usr/lib/systemd/user/omarchy-crash-watch.service"] = "[Service]\nExecStart=/usr/bin/omarchy-crash-watch\n"
	fake.SetCommand(system.CommandResult{ExitCode: 0}, "systemctl", "--user", "daemon-reload")
	fake.SetCommand(system.CommandResult{ExitCode: 0}, "systemctl", "--user", "enable", "--now", "omarchy-crash-watch.service")

	actions, err := Stage4Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var copyAction *plan.Action
	for i := range actions {
		if strings.Contains(actions[i].Description, "systemd --user units") {
			copyAction = &actions[i]
		}
	}
	if copyAction == nil {
		t.Fatalf("no systemd unit copy action found among: %+v", actions)
	}
	if err := copyAction.Apply(fake); err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := fake.Files["/home/james/.config/systemd/user/omarchy-crash-watch.service"]
	if strings.Contains(got, "/usr/bin/omarchy-crash-watch") {
		t.Errorf("copied unit still points at the deleted package path, got: %s", got)
	}
	if !strings.Contains(got, "/home/james/dotfiles-omarchy/bin/omarchy-crash-watch") {
		t.Errorf("copied unit doesn't point at the vendored path, got: %s", got)
	}
}
