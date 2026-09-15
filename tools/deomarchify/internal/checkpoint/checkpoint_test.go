package checkpoint

import (
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

const path = "/home/james/.deomarchify/checkpoint.json"

func TestLoad_MissingFileIsEmptyState(t *testing.T) {
	fake := system.NewFake()
	s, err := Load(fake, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Completed) != 0 {
		t.Errorf("expected empty state, got %v", s.Completed)
	}
	if s.IsDone(0) {
		t.Errorf("stage 0 should not be done")
	}
}

func TestMarkDone_ThenLoadRoundTrips(t *testing.T) {
	fake := system.NewFake()
	s, _ := Load(fake, path)

	s, err := MarkDone(fake, path, s, 0, "2026-09-15T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.IsDone(0) {
		t.Errorf("stage 0 should be done after MarkDone")
	}

	reloaded, err := Load(fake, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reloaded.IsDone(0) {
		t.Errorf("reloaded state should still show stage 0 done")
	}
}

func TestMarkDone_PreservesEarlierStages(t *testing.T) {
	fake := system.NewFake()
	s, _ := Load(fake, path)
	s, _ = MarkDone(fake, path, s, 0, "t0")
	s, _ = MarkDone(fake, path, s, 1, "t1")

	if !s.IsDone(0) || !s.IsDone(1) {
		t.Fatalf("expected stages 0 and 1 both done, got %v", s.Completed)
	}
	if got := s.CompletedStages(); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("CompletedStages = %v, want [0 1]", got)
	}
}

func TestLoad_CorruptFileIsError(t *testing.T) {
	fake := system.NewFake()
	fake.Files[path] = "{not json"

	_, err := Load(fake, path)
	if err == nil {
		t.Fatalf("expected error for corrupt checkpoint file")
	}
}
