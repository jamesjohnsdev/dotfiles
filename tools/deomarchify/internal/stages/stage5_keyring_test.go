package stages

import (
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func installedFake(installed map[string]bool) *system.Fake {
	fake := system.NewFake()
	for _, pkg := range []string{"omarchy", "omarchy-settings", "omarchy-nvim", "omarchy-keyring"} {
		if installed[pkg] {
			fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : " + pkg + "\nRequired By : None\n"}, "pacman", "-Qi", pkg)
		} else {
			fake.SetCommand(system.CommandResult{ExitCode: 1}, "pacman", "-Qi", pkg)
		}
	}
	return fake
}

func TestStage5Plan_RefusesIfOtherPackagesStillInstalled(t *testing.T) {
	fake := installedFake(map[string]bool{"omarchy": true, "omarchy-keyring": true})

	_, err := Stage5Plan(fake, config.Config{})
	if err == nil {
		t.Fatalf("expected error when omarchy is still installed")
	}
}

func TestStage5Plan_ProceedsOnceOthersGone(t *testing.T) {
	fake := installedFake(map[string]bool{"omarchy-keyring": true})

	actions, err := Stage5Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
}

func TestStage5Plan_NoopWhenAlreadyRemoved(t *testing.T) {
	fake := installedFake(map[string]bool{})

	actions, err := Stage5Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected no actions, got %d", len(actions))
	}
}
