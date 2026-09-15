package stages

import (
	"reflect"
	"strings"
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

// canAllInstalled registers the canned response for pacman.AllInstalled's
// `pacman -Qi` (no args) call, built from real Info fixtures so tests read
// like "these packages are installed" rather than raw pacman text.
func canAllInstalled(fake *system.Fake, infos []pacman.Info) {
	var blocks []string
	for _, info := range infos {
		block := "Name            : " + info.Name + "\n"
		if len(info.Depends) > 0 {
			block += "Depends On      : "
			for i, d := range info.Depends {
				if i > 0 {
					block += "  "
				}
				block += d
			}
			block += "\n"
		}
		if len(info.Provides) > 0 {
			block += "Provides        : "
			for i, p := range info.Provides {
				if i > 0 {
					block += "  "
				}
				block += p
			}
			block += "\n"
		}
		blocks = append(blocks, block)
	}
	output := ""
	for i, b := range blocks {
		if i > 0 {
			output += "\n"
		}
		output += b
	}
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: output}, "pacman", "-Qi")
}

func TestStage0Plan_SkipsMarkExplicitWhenNothingToProtect(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{OmarchyPackages: []string{"omarchy"}}
	fake.SetCommand(system.CommandResult{ExitCode: 1}, "pacman", "-Qi", "omarchy")
	canAllInstalled(fake, nil)

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
	canAllInstalled(fake, []pacman.Info{{Name: "hyprland"}, {Name: "sddm"}})

	actions, err := Stage0Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions (snapshot + mark-explicit), got %d: %+v", len(actions), actions)
	}
}

// This reproduces the exact real-world failure: `pacman -D --asexplicit`
// bundled all 24 dependency names into one command, and it failed entirely
// because "quickshell" (from omarchy's Depends On) isn't installed under
// that literal name on this machine - "quickshell-git" is, and it provides
// "quickshell". The fix must resolve to the real installed name, not drop
// protection for it.
func TestStage0Plan_ResolvesProviderPackageInsteadOfDroppingProtection(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{OmarchyPackages: []string{"omarchy"}}
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy\nDepends On : hyprland  quickshell\n"}, "pacman", "-Qi", "omarchy")
	canAllInstalled(fake, []pacman.Info{
		{Name: "hyprland"},
		{Name: "quickshell-git", Provides: []string{"quickshell"}},
	})

	actions, err := Stage0Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var markExplicit *string
	for i := range actions {
		if len(actions[i].Description) > 4 && actions[i].Description[:4] == "mark" {
			markExplicit = &actions[i].Description
		}
	}
	if markExplicit == nil {
		t.Fatalf("no mark-explicit action found among: %+v", actions)
	}
	if !strings.Contains(*markExplicit, "quickshell-git") {
		t.Errorf("expected resolved provider quickshell-git in description, got: %s", *markExplicit)
	}
	if strings.Contains(*markExplicit, "[quickshell ") || strings.Contains(*markExplicit, "quickshell]") {
		t.Errorf("literal unresolved 'quickshell' name should not appear (only the real installed quickshell-git), got: %s", *markExplicit)
	}
}

func TestStage0Plan_SurfacesTrulyUnresolvedDependencies(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{OmarchyPackages: []string{"omarchy"}}
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy\nDepends On : hyprland  totally-missing-thing\n"}, "pacman", "-Qi", "omarchy")
	canAllInstalled(fake, []pacman.Info{{Name: "hyprland"}})

	actions, err := Stage0Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, a := range actions {
		if strings.Contains(a.Description, "WARNING") && strings.Contains(a.Description, "totally-missing-thing") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning action naming the unresolved dependency, got: %+v", actions)
	}
}
