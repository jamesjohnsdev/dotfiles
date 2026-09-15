package stages

import (
	"reflect"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/pacman"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestDependenciesToProtect_ExcludesOmarchyPackagesAndDedups(t *testing.T) {
	omarchyPkgs := []string{"omarchy", "omarchy-settings", "omarchy-nvim", "omarchy-keyring"}
	infos := []pacman.Info{
		{Name: "omarchy", Depends: []string{"omarchy-keyring", "omarchy-settings", "hyprland", "uwsm", "sddm"}},
		{Name: "omarchy-settings", Depends: []string{"hyprland", "wireplumber", "docker"}},
	}

	got := DependenciesToProtect(infos, omarchyPkgs)
	want := []string{"docker", "hyprland", "sddm", "uwsm", "wireplumber"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("DependenciesToProtect = %v, want %v", got, want)
	}
}

func TestDependenciesToProtect_NoDeps(t *testing.T) {
	got := DependenciesToProtect(nil, []string{"omarchy"})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestStage0Plan_SkipsMarkExplicitWhenNothingToProtect(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{OmarchyPackages: []string{"omarchy"}}
	fake.SetCommand(system.CommandResult{ExitCode: 1}, "pacman", "-Qi", "omarchy")

	actions, err := Stage0Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the snapshot action, no mark-explicit action, since omarchy
	// isn't even installed (nothing to query deps from).
	if len(actions) != 1 {
		t.Fatalf("expected 1 action (snapshot only), got %d: %+v", len(actions), actions)
	}
}

func TestStage0Plan_IncludesMarkExplicitWhenDepsFound(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{OmarchyPackages: []string{"omarchy"}}
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy\nDepends On : hyprland  sddm\n"}, "pacman", "-Qi", "omarchy")

	actions, err := Stage0Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions (snapshot + mark-explicit), got %d: %+v", len(actions), actions)
	}
}
