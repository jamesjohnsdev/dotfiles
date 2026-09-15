package stages

import (
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestNeedsMirrorFix(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"omarchy mirror", "Server = https://stable-mirror.omarchy.org/$repo/os/$arch\n", true},
		{"already fixed", "Server = https://geo.mirror.pkgbuild.com/$repo/os/$arch\n", false},
		{"other real mirror", "Server = https://mirror.example.com/archlinux/$repo/os/$arch\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NeedsMirrorFix(c.content); got != c.want {
				t.Errorf("NeedsMirrorFix(%q) = %v, want %v", c.content, got, c.want)
			}
		})
	}
}

func TestStage1Plan_NoActionWhenAlreadyFixed(t *testing.T) {
	fake := system.NewFake()
	fake.Files[mirrorlistPath] = officialGeoMirror

	actions, err := Stage1Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected no actions when mirrorlist already fixed, got %d", len(actions))
	}
}

func TestStage1Plan_PlansBackupAndReplaceWhenOmarchyMirror(t *testing.T) {
	fake := system.NewFake()
	fake.Files[mirrorlistPath] = "Server = https://stable-mirror.omarchy.org/$repo/os/$arch\n"

	actions, err := Stage1Plan(fake, config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions (backup+replace, then -Syy), got %d", len(actions))
	}

	fake.SetCommand(system.CommandResult{ExitCode: 0}, "sudo", "pacman", "-Syy", "--noconfirm")
	if err := actions[0].Apply(fake); err != nil {
		t.Fatalf("apply backup+replace: %v", err)
	}
	if fake.Files[mirrorlistPath] != officialGeoMirror {
		t.Errorf("mirrorlist not replaced, got %q", fake.Files[mirrorlistPath])
	}
	if fake.Files[mirrorlistPath+".pre-deomarchify.bak"] == "" {
		t.Errorf("backup not written")
	}

	// /etc/pacman.d/mirrorlist is root-owned: both the backup and the
	// replacement must go through the privileged Sudo* path, not the plain
	// one (which runs as the invoking user and fails with a permission
	// error against a real /etc path - confirmed on a real machine).
	sawSudoCopy, sawSudoWrite := false, false
	for _, r := range fake.Ran {
		if r == "sudo:copy-file "+mirrorlistPath+" "+mirrorlistPath+".pre-deomarchify.bak" {
			sawSudoCopy = true
		}
		if r == "sudo:write-file "+mirrorlistPath {
			sawSudoWrite = true
		}
	}
	if !sawSudoCopy {
		t.Errorf("expected a privileged (sudo) copy of the mirrorlist backup, got Ran=%v", fake.Ran)
	}
	if !sawSudoWrite {
		t.Errorf("expected a privileged (sudo) write of the new mirrorlist, got Ran=%v", fake.Ran)
	}

	if err := actions[1].Apply(fake); err != nil {
		t.Fatalf("apply pacman -Syy: %v", err)
	}
}
