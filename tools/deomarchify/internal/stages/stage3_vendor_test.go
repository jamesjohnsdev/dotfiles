package stages

import (
	"strings"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestIsSelfUpdateScript(t *testing.T) {
	cases := map[string]bool{
		"omarchy-update":           true,
		"omarchy-update-status":    true,
		"omarchy-migrate":          true,
		"omarchy-migrate-notify":   true,
		"omarchy-refresh-pacman":   true,
		"omarchy-reinstall":        true,
		"omarchy-reinstall-pkgs":   true,
		"omarchy-menu":             false,
		"omarchy-theme-set":        false,
		"omarchy-refresh-hyprland": false,
	}
	for name, want := range cases {
		if got := IsSelfUpdateScript(name); got != want {
			t.Errorf("IsSelfUpdateScript(%q) = %v, want %v", name, got, want)
		}
	}
}

func baseCfg(home string) config.Config {
	return config.Config{
		Home:            home,
		VendorDir:       home + "/dotfiles-omarchy",
		OmarchyPackages: []string{"omarchy-nvim", "omarchy", "omarchy-settings", "omarchy-keyring"},
	}
}

func TestStage3Plan_IncludesExpectedActionsAndOrder(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy\nRequired By : None\n"}, "pacman", "-Qi", "omarchy")

	cfg := baseCfg("/home/james")
	actions, err := Stage3Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// len(VendorSubdirs) copy actions + neutralize + uwsm env + bashrc +
	// zshrc + symlink + package removal.
	want := len(VendorSubdirs) + 6
	if len(actions) != want {
		t.Fatalf("expected %d actions, got %d:\n%s", want, len(actions), describeActions(actions))
	}

	last := actions[len(actions)-1]
	if !strings.Contains(last.Description, "remove package omarchy") {
		t.Errorf("expected last action to be package removal, got: %s", last.Description)
	}
}

func TestStage3Plan_RefusesWhenOmarchyStillHasDependents(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy\nRequired By : something-unexpected\n"}, "pacman", "-Qi", "omarchy")

	_, err := Stage3Plan(fake, baseCfg("/home/james"))
	if err == nil {
		t.Fatalf("expected error when omarchy has unexpected dependents")
	}
}

func TestStage3Plan_BashrcPatchFailsLoudlyIfLineMissing(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 1}, "pacman", "-Qi", "omarchy")
	fake.Files[bashrcPath("/home/james")] = "echo already customized, no omarchy line here\n"

	actions, err := Stage3Plan(fake, baseCfg("/home/james"))
	if err != nil {
		t.Fatalf("unexpected error building plan: %v", err)
	}

	var bashrcAction *plan.Action
	for i := range actions {
		if strings.Contains(actions[i].Description, ".bashrc") {
			bashrcAction = &actions[i]
			break
		}
	}
	if bashrcAction == nil {
		t.Fatalf("no bashrc action found")
	}
	if err := bashrcAction.Apply(fake); err == nil {
		t.Errorf("expected Apply to fail when the expected line is missing, got nil")
	}
}

func describeActions(actions []plan.Action) string {
	var b strings.Builder
	for _, a := range actions {
		b.WriteString("- " + a.Description + "\n")
	}
	return b.String()
}
