package pacman

import (
	"reflect"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

const qiFixture = `Name            : omarchy
Version         : 4.0.2-1
Description     : An opinionated take on setting up Linux desktops
Architecture    : any
URL             : https://omarchy.org
Licenses        : MIT
Groups          : None
Provides        : None
Depends On      : omarchy-keyring  omarchy-settings=4.0.2  limine  limine-mkinitcpio-hook  hyprland  uwsm  sddm
Optional Deps   : None
Required By     : None
Optional For    : omarchy-settings
Conflicts With  : None
Replaces        : None
Installed Size  : 2.50 MiB
Packager        : Unknown Packager
Build Date      : Mon 31 Aug 2026 01:11:00 PM UTC
Install Date    : Fri 29 Aug 2026 03:58:32 PM UTC
Install Reason  : Explicitly installed
Install Script  : Yes
Validated By    : None
`

const qlFixture = `omarchy /etc/
omarchy /etc/skel/
omarchy /usr/bin/omarchy
omarchy /usr/bin/omarchy-menu
omarchy /usr/share/omarchy/
omarchy /usr/share/omarchy/bin/
omarchy /usr/share/omarchy/bin/omarchy-menu
`

func TestParseQi(t *testing.T) {
	info := ParseQi(qiFixture)

	if info.Name != "omarchy" {
		t.Errorf("Name = %q, want omarchy", info.Name)
	}

	wantDepends := []string{"omarchy-keyring", "omarchy-settings", "limine", "limine-mkinitcpio-hook", "hyprland", "uwsm", "sddm"}
	if !reflect.DeepEqual(info.Depends, wantDepends) {
		t.Errorf("Depends = %v, want %v", info.Depends, wantDepends)
	}

	if info.RequiredBy != nil {
		t.Errorf("RequiredBy = %v, want nil (None)", info.RequiredBy)
	}

	wantOptFor := []string{"omarchy-settings"}
	if !reflect.DeepEqual(info.OptionalFor, wantOptFor) {
		t.Errorf("OptionalFor = %v, want %v", info.OptionalFor, wantOptFor)
	}
}

func TestParseQi_VersionConstraintStripped(t *testing.T) {
	info := ParseQi(qiFixture)
	for _, d := range info.Depends {
		if d == "omarchy-settings=4.0.2" {
			t.Fatalf("version constraint not stripped from Depends: %v", info.Depends)
		}
	}
}

func TestParseQl_DropsDirectories(t *testing.T) {
	files := ParseQl(qlFixture)
	want := []string{"/usr/bin/omarchy", "/usr/bin/omarchy-menu", "/usr/share/omarchy/bin/omarchy-menu"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("ParseQl = %v, want %v", files, want)
	}
}

func TestQueryInfo_NotInstalled(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 1, Stderr: "package 'nope' was not found"}, "pacman", "-Qi", "nope")

	_, ok, err := QueryInfo(fake, "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected ok=false for uninstalled package")
	}
}

func TestQueryInfo_Installed(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: qiFixture}, "pacman", "-Qi", "omarchy")

	info, ok, err := QueryInfo(fake, "omarchy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if info.Name != "omarchy" {
		t.Errorf("Name = %q, want omarchy", info.Name)
	}
}

func TestListFiles(t *testing.T) {
	fake := system.NewFake()
	fake.SetCommand(system.CommandResult{ExitCode: 0, Stdout: qlFixture}, "pacman", "-Ql", "omarchy")

	files, err := ListFiles(fake, "omarchy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("got %d files, want 3: %v", len(files), files)
	}
}
