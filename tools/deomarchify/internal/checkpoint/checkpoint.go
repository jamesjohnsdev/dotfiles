// Package checkpoint tracks which stages have completed, so re-running the
// driver after fixing a problem skips work already done instead of
// re-applying it (each stage's own actions are also idempotent, this is the
// coarser "don't even re-check" fast path).
package checkpoint

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// State is the on-disk checkpoint format.
type State struct {
	Completed map[int]string `json:"completed"` // stage number -> timestamp finished
}

// Load reads the checkpoint file. A missing file is not an error: it means
// no stage has run yet.
func Load(sys system.System, path string) (State, error) {
	content, err := sys.ReadFile(path)
	if err != nil {
		return State{Completed: map[int]string{}}, nil
	}
	var s State
	if err := json.Unmarshal([]byte(content), &s); err != nil {
		return State{}, fmt.Errorf("checkpoint file %s is corrupt: %w", path, err)
	}
	if s.Completed == nil {
		s.Completed = map[int]string{}
	}
	return s, nil
}

// Save writes the checkpoint file.
func Save(sys system.System, path string, s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return sys.WriteFile(path, string(b)+"\n", 0o644)
}

// MarkDone records stage as completed and saves.
func MarkDone(sys system.System, path string, s State, stage int, timestamp string) (State, error) {
	next := State{Completed: map[int]string{}}
	for k, v := range s.Completed {
		next.Completed[k] = v
	}
	next.Completed[stage] = timestamp
	if err := Save(sys, path, next); err != nil {
		return s, err
	}
	return next, nil
}

// IsDone reports whether stage has already completed.
func (s State) IsDone(stage int) bool {
	_, ok := s.Completed[stage]
	return ok
}

// CompletedStages returns completed stage numbers, sorted ascending.
func (s State) CompletedStages() []int {
	out := make([]int, 0, len(s.Completed))
	for k := range s.Completed {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
