package stages

import (
	"strings"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestStage2Plan_NoopWhenNotInstalled(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 1}, "pacman", "-Qi", "omarchy-nvim")

	actions, err := Stage2Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected no actions, got %d", len(actions))
	}
}

func TestStage2Plan_PlansRemovalWhenInstalledAndUnused(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy-nvim\nRequired By : None\n"}, "pacman", "-Qi", "omarchy-nvim")

	actions, err := Stage2Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	fake.SetCommand(system.CommandResult{ExitCode: 0}, "sudo", "pacman", "-R", "--noconfirm", "omarchy-nvim")
	if err := actions[0].Apply(fake); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestStage2Plan_RefusesWhenSomethingElseDependsOnIt(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy-nvim\nRequired By : some-other-pkg\n"}, "pacman", "-Qi", "omarchy-nvim")

	_, err := Stage2Plan(fake, config.Config{})
	if err == nil {
		t.Fatalf("expected error when a dependent exists")
	}
	if !strings.Contains(err.Error(), "some-other-pkg") {
		t.Errorf("error should name the dependent, got: %v", err)
	}
}
