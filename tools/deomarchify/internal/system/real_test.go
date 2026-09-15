package system

import (
	"os"
	"path/filepath"
	"testing"
)

// Real is otherwise untested by design (thin OS/exec wrapper, exercised by
// running the built binary instead - see the package doc). This file is the
// one exception: it covers the exact bug a real -apply run hit, where
// overwriting a file that already exists with a no-write-bit mode (0555,
// typical for installed package scripts) failed even though the caller
// owned it and passed a writable perm - os.WriteFile ignores its perm
// argument for files that already exist. Uses only a real temp dir; no
// sudo, no real system paths.

func TestReal_WriteFile_OverwritesReadOnlyExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script")

	if err := os.WriteFile(path, []byte("old"), 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := NewReal()
	if err := r.WriteFile(path, "new", 0o755); err != nil {
		t.Fatalf("WriteFile over a 0555 existing file: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
}

func TestReal_CopyFile_OverwritesReadOnlyExistingDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	if err := os.WriteFile(src, []byte("from source"), 0o644); err != nil {
		t.Fatalf("setup src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("stale"), 0o555); err != nil {
		t.Fatalf("setup dst: %v", err)
	}

	r := NewReal()
	if err := r.CopyFile(src, dst, 0o755); err != nil {
		t.Fatalf("CopyFile over a 0555 existing destination: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(got) != "from source" {
		t.Errorf("content = %q, want %q", got, "from source")
	}
}

func TestReal_CopyTree_RetryAfterPartialFailureSucceeds(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "src")
	dstDir := filepath.Join(t.TempDir(), "dst")
	targetDir := t.TempDir()

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	realTarget := filepath.Join(targetDir, "omarchy-migrate")
	if err := os.WriteFile(realTarget, []byte("#!/bin/sh\necho real\n"), 0o555); err != nil {
		t.Fatalf("setup target: %v", err)
	}
	if err := os.Symlink(realTarget, filepath.Join(srcDir, "omarchy")); err != nil {
		t.Fatalf("setup symlink: %v", err)
	}

	r := NewReal()
	if err := r.CopyTree(srcDir, dstDir); err != nil {
		t.Fatalf("first CopyTree: %v", err)
	}

	// Confirms the exact real failure: a retry re-walks the whole source
	// tree, including entries already copied by the first (partial) run.
	if err := r.CopyTree(srcDir, dstDir); err != nil {
		t.Fatalf("retry CopyTree over already-copied entry: %v", err)
	}

	vendoredPath := filepath.Join(dstDir, "omarchy")
	info, err := os.Lstat(vendoredPath)
	if err != nil {
		t.Fatalf("stat vendored copy: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("vendored copy is still a symlink - should be a dereferenced real file")
	}
}

// This is the more important correctness case: /usr/share/omarchy/bin's
// real entries are symlinks into /usr/bin/omarchy-* (owned by the package
// being vendored away from). If CopyTree preserved that symlink instead of
// dereferencing it, the "vendored" checkout would still silently depend on
// the package and go dangling the moment it's removed - confirmed on a real
// machine (all 428 bin/ entries there are symlinks of exactly this shape).
func TestReal_CopyTree_DereferencesSymlinkContentSurvivesTargetRemoval(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "src")
	dstDir := filepath.Join(t.TempDir(), "dst")
	targetDir := t.TempDir()

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	realTarget := filepath.Join(targetDir, "omarchy-migrate")
	if err := os.WriteFile(realTarget, []byte("#!/bin/sh\necho real content\n"), 0o555); err != nil {
		t.Fatalf("setup target: %v", err)
	}
	if err := os.Symlink(realTarget, filepath.Join(srcDir, "omarchy-migrate")); err != nil {
		t.Fatalf("setup symlink: %v", err)
	}

	r := NewReal()
	if err := r.CopyTree(srcDir, dstDir); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	// Simulate the package being removed: the symlink's target disappears.
	if err := os.Remove(realTarget); err != nil {
		t.Fatalf("simulating target removal: %v", err)
	}

	vendoredPath := filepath.Join(dstDir, "omarchy-migrate")
	info, err := os.Lstat(vendoredPath)
	if err != nil {
		t.Fatalf("vendored copy vanished along with the original target - not actually vendored: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("vendored copy is a symlink, not a dereferenced copy - would now be dangling")
	}
	got, err := os.ReadFile(vendoredPath)
	if err != nil {
		t.Fatalf("reading vendored copy after target removal: %v", err)
	}
	if string(got) != "#!/bin/sh\necho real content\n" {
		t.Errorf("vendored content = %q, want the real script content", got)
	}
}

// A stale symlink already sitting at the destination (left over from an
// earlier, pre-fix vendoring attempt that preserved symlinks into
// /usr/bin/... instead of dereferencing them) must not make writeFileForce
// try to chmod through it into a root-owned target - confirmed on a real
// machine as the exact failure that followed fixing the dereferencing bug.
func TestReal_WriteFile_ReplacesStaleSymlinkDestination(t *testing.T) {
	dir := t.TempDir()
	rootOwnedLookingTarget := filepath.Join(t.TempDir(), "not-actually-writable")
	if err := os.WriteFile(rootOwnedLookingTarget, []byte("original"), 0o444); err != nil {
		t.Fatalf("setup: %v", err)
	}
	path := filepath.Join(dir, "omarchy-migrate")
	if err := os.Symlink(rootOwnedLookingTarget, path); err != nil {
		t.Fatalf("setup symlink: %v", err)
	}

	r := NewReal()
	if err := r.WriteFile(path, "stub", 0o755); err != nil {
		t.Fatalf("WriteFile over a stale symlink destination: %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("path is still a symlink after WriteFile - should be a real file now")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(got) != "stub" {
		t.Errorf("content = %q, want %q", got, "stub")
	}

	// And the thing the stale symlink pointed at must be untouched.
	unchanged, err := os.ReadFile(rootOwnedLookingTarget)
	if err != nil || string(unchanged) != "original" {
		t.Errorf("symlink target was modified, got %q, err %v", unchanged, err)
	}
}

func TestReal_CopyTree_VendoredCopyIsOwnerWritable(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "src")
	dstDir := filepath.Join(t.TempDir(), "dst")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	scriptPath := filepath.Join(srcDir, "omarchy-migrate")
	// 0555: no write bit at all, same as a real installed package script.
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho old\n"), 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}

	r := NewReal()
	if err := r.CopyTree(srcDir, dstDir); err != nil {
		t.Fatalf("CopyTree: %v", err)
	}

	vendoredPath := filepath.Join(dstDir, "omarchy-migrate")
	info, err := os.Stat(vendoredPath)
	if err != nil {
		t.Fatalf("stat vendored copy: %v", err)
	}
	if info.Mode()&0o200 == 0 {
		t.Fatalf("vendored copy mode %o has no owner-write bit - matches the exact bug that broke a real -apply run", info.Mode())
	}

	// And confirm it's actually writable in practice, not just the bit set
	// (belt and suspenders - this is what the real neutralize step does).
	if err := r.WriteFile(vendoredPath, "#!/bin/sh\necho disabled\n", 0o755); err != nil {
		t.Fatalf("writing over the vendored copy: %v", err)
	}
}
