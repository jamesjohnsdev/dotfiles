package envfiles

import (
	"strings"
	"testing"
)

func TestUWSMEnvFile_ContainsOmarchyPath(t *testing.T) {
	out := UWSMEnvFile("/home/james/dotfiles-omarchy")
	want := `export OMARCHY_PATH="/home/james/dotfiles-omarchy"`
	if !strings.Contains(out, want) {
		t.Errorf("output missing %q, got:\n%s", want, out)
	}
	if !strings.Contains(out, "/home/james/dotfiles-omarchy/bin:$PATH") {
		t.Errorf("output missing PATH prepend, got:\n%s", out)
	}
}

func TestPatchBashrc_ReplacesHardcodedPath(t *testing.T) {
	original := "# Omarchy environment (OMARCHY_PATH + PATH), needed even for non-interactive shells\n" +
		"[[ -r /usr/share/omarchy/default/bash/env-bootstrap ]] && source /usr/share/omarchy/default/bash/env-bootstrap\n\n" +
		"[[ $- != *i* ]] && return\n"

	patched, changed := PatchBashrc(original, "/home/james/dotfiles-omarchy")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if strings.Contains(patched, "/usr/share/omarchy") {
		t.Errorf("hardcoded /usr/share/omarchy still present:\n%s", patched)
	}
	if !strings.Contains(patched, "/home/james/dotfiles-omarchy/default/bash/env-bootstrap") {
		t.Errorf("vendored path not present:\n%s", patched)
	}
	if !strings.Contains(patched, "[[ $- != *i* ]] && return") {
		t.Errorf("unrelated lines were dropped:\n%s", patched)
	}
}

func TestPatchBashrc_NoMatchReturnsUnchanged(t *testing.T) {
	original := "echo hello\n"
	patched, changed := PatchBashrc(original, "/x")
	if changed {
		t.Errorf("expected changed=false when line absent")
	}
	if patched != original {
		t.Errorf("expected content unchanged")
	}
}

func TestPatchZshrc_ReplacesExportLine(t *testing.T) {
	original := "export OMARCHY_PATH=$HOME/.local/share/omarchy\n" +
		"export PATH=$OMARCHY_PATH/bin:$PATH:$HOME/.local/bin\n"

	patched, changed := PatchZshrc(original, "/home/james/dotfiles-omarchy")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if !strings.Contains(patched, "export OMARCHY_PATH=/home/james/dotfiles-omarchy") {
		t.Errorf("export line not patched:\n%s", patched)
	}
	// The PATH line references $OMARCHY_PATH indirectly and should be left alone.
	if !strings.Contains(patched, "export PATH=$OMARCHY_PATH/bin:$PATH:$HOME/.local/bin") {
		t.Errorf("unrelated PATH line was changed:\n%s", patched)
	}
}

func TestPatchZshrc_NoMatchReturnsUnchanged(t *testing.T) {
	original := "echo hello\n"
	patched, changed := PatchZshrc(original, "/x")
	if changed {
		t.Errorf("expected changed=false when line absent")
	}
	if patched != original {
		t.Errorf("expected content unchanged")
	}
}
