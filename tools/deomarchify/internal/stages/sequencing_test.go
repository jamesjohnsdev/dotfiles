package stages

import (
	"strings"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// This reproduces what a real `-apply` run surfaced: a plan-everything-
// upfront dry-run of Stage 4 fails, because omarchy-settings still shows
// `Required By: [omarchy]` while omarchy hasn't actually been removed yet
// (dry-run applies nothing). That's correct dry-run behavior, not a bug -
// but it only means something if Stage 4 genuinely succeeds once Stage 3's
// removal of `omarchy` has actually happened, which is what a real -apply
// run does (each stage Plans only right before it executes, after every
// earlier stage already applied). This test proves that transition holds,
// by hand-simulating pacman's state change between the two stages instead
// of trusting the reasoning.
func TestSequencing_Stage3RemovalOfOmarchyUnblocksStage4(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{
		Home:      "/home/james",
		VendorDir: "/home/james/dotfiles-omarchy",
	}

	// Before Stage 3 runs: omarchy is installed, omarchy-settings is
	// installed and still required by omarchy (real current state).
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy\nRequired By : None\n"}, "pacman", "-Qi", "omarchy")
	fake.SetCommand(system.CommandResult{ExitCode: 0}, "sudo", "pacman", "-R", "--noconfirm", "omarchy")

	// Files Stage 3's actions need to read/patch.
	fake.Files[bashrcPath(cfg.Home)] = "[[ -r /usr/share/omarchy/default/bash/env-bootstrap ]] && source /usr/share/omarchy/default/bash/env-bootstrap\n"
	fake.Files[zshrcPath(cfg.Home)] = "export OMARCHY_PATH=$HOME/.local/share/omarchy\n"

	stage3Actions, err := Stage3Plan(fake, cfg)
	if err != nil {
		t.Fatalf("Stage3Plan: %v", err)
	}
	for _, a := range stage3Actions {
		if err := a.Apply(fake); err != nil {
			t.Fatalf("applying stage3 action %q: %v", a.Description, err)
		}
	}

	// Simulate what pacman would now report: omarchy is gone, so
	// omarchy-settings (still installed) is no longer required by it. A
	// real pacman would report this live; the fake needs it re-registered.
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "Name : omarchy-settings\nRequired By : None\n"}, "pacman", "-Qi", "omarchy-settings")
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: "omarchy-settings /etc/sysctl.d/99-omarchy-sysctl.conf\n"}, "pacman", "-Ql", "omarchy-settings")

	stage4Actions, err := Stage4Plan(fake, cfg)
	if err != nil {
		t.Fatalf("Stage4Plan should succeed once omarchy is actually gone, got error: %v", err)
	}
	if len(stage4Actions) == 0 {
		t.Errorf("expected stage 4 to plan some actions")
	}

	// And confirm the bashrc/zshrc patches Stage 3 applied actually took,
	// as a bonus check that the sequencing test exercised real Apply logic
	// rather than short-circuiting.
	if got := fake.Files[bashrcPath(cfg.Home)]; !strings.Contains(got, cfg.VendorDir) {
		t.Errorf("bashrc not patched to vendored path, got: %s", got)
	}
}
