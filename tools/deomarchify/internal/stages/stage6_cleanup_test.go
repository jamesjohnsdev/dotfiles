package stages

import (
	"strings"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestCleanupGlobs_NeverTouchesLiveState(t *testing.T) {
	for _, g := range CleanupGlobs("/home/james") {
		if strings.Contains(g, ".local/state/omarchy") {
			t.Errorf("cleanup glob touches live state dir: %s", g)
		}
	}
}

func TestStage6Plan_RemovesMatchingBackupFiles(t *testing.T) {
	fake := system.NewFake()
	cfg := config.Config{Home: "/home/james"}

	fake.Files["/home/james/.config/mimeapps.list.omarchy-upgrade-to-quattro.20260829155833.bak"] = "x"
	fake.Files["/home/james/.config/waybar.omarchy-upgrade-to-quattro.20260829155833.bak"] = "x"
	fake.Files["/home/james/.config/waybar/config.jsonc"] = "keep me"
	fake.SetCommand(system.CommandResult{ExitCode: 0}, "rm", "-rf", "/home/james/.cache/omarchy")

	actions, err := Stage6Plan(fake, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, a := range actions {
		if err := a.Apply(fake); err != nil {
			t.Fatalf("apply %q: %v", a.Description, err)
		}
	}

	if _, ok := fake.Files["/home/james/.config/mimeapps.list.omarchy-upgrade-to-quattro.20260829155833.bak"]; ok {
		t.Errorf("bak file not removed")
	}
	if _, ok := fake.Files["/home/james/.config/waybar/config.jsonc"]; !ok {
		t.Errorf("unrelated file was removed")
	}
}
