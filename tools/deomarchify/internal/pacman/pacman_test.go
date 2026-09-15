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

const qiAllFixture = `Name            : bash
Version         : 5.2.037-1

Name            : quickshell-git
Version         : 0.2.0.r120-1
Provides        : quickshell=0.2.0

Name            : hyprland
Version         : 0.45.0-1
`

func TestParseQiAll_SplitsOnBlankLines(t *testing.T) {
	infos := ParseQiAll(qiAllFixture)
	if len(infos) != 3 {
		t.Fatalf("expected 3 packages, got %d: %+v", len(infos), infos)
	}
	names := []string{infos[0].Name, infos[1].Name, infos[2].Name}
	want := []string{"bash", "quickshell-git", "hyprland"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestParseQiAll_ParsesProvidesWithVersionStripped(t *testing.T) {
	infos := ParseQiAll(qiAllFixture)
	var quickshell Info
	for _, i := range infos {
		if i.Name == "quickshell-git" {
			quickshell = i
		}
	}
	want := []string{"quickshell"}
	if !reflect.DeepEqual(quickshell.Provides, want) {
		t.Errorf("Provides = %v, want %v", quickshell.Provides, want)
	}
}

func TestResolveInstalledNames_LiteralNameMatch(t *testing.T) {
	installed := []Info{{Name: "hyprland"}, {Name: "sddm"}}
	resolved, unresolved := ResolveInstalledNames([]string{"hyprland", "sddm"}, installed)

	if !reflect.DeepEqual(resolved, []string{"hyprland", "sddm"}) {
		t.Errorf("resolved = %v", resolved)
	}
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

func TestResolveInstalledNames_ResolvesViaProvides(t *testing.T) {
	installed := []Info{
		{Name: "hyprland"},
		{Name: "quickshell-git", Provides: []string{"quickshell"}},
	}
	resolved, unresolved := ResolveInstalledNames([]string{"hyprland", "quickshell"}, installed)

	want := []string{"hyprland", "quickshell-git"}
	if !reflect.DeepEqual(resolved, want) {
		t.Errorf("resolved = %v, want %v", resolved, want)
	}
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

func TestResolveInstalledNames_TrulyMissingGoesToUnresolved(t *testing.T) {
	installed := []Info{{Name: "hyprland"}}
	resolved, unresolved := ResolveInstalledNames([]string{"hyprland", "nothing-provides-this"}, installed)

	if !reflect.DeepEqual(resolved, []string{"hyprland"}) {
		t.Errorf("resolved = %v", resolved)
	}
	if !reflect.DeepEqual(unresolved, []string{"nothing-provides-this"}) {
		t.Errorf("unresolved = %v", unresolved)
	}
}

func TestResolveInstalledNames_DedupsWhenTwoCandidatesResolveToSameRealPackage(t *testing.T) {
	installed := []Info{{Name: "quickshell-git", Provides: []string{"quickshell", "quickshell-lib"}}}
	resolved, _ := ResolveInstalledNames([]string{"quickshell", "quickshell-lib"}, installed)

	if !reflect.DeepEqual(resolved, []string{"quickshell-git"}) {
		t.Errorf("resolved = %v, want deduped to just quickshell-git", resolved)
	}
}
