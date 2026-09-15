package plan

import (
	"errors"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func TestRun_DryRunNeverApplies(t *testing.T) {
	applied := false
	actions := []Action{
		{Description: "do a thing", Apply: func(sys system.System) error {
			applied = true
			return nil
		}},
	}

	results := Run(actions, system.NewFake(), Options{DryRun: true})

	if applied {
		t.Errorf("Apply was called during dry-run")
	}
	if len(results) != 1 || results[0].Ran {
		t.Errorf("expected one unRan result, got %+v", results)
	}
}

func TestRun_StopsAtFirstError(t *testing.T) {
	var order []string
	actions := []Action{
		{Description: "one", Apply: func(sys system.System) error {
			order = append(order, "one")
			return nil
		}},
		{Description: "two", Apply: func(sys system.System) error {
			order = append(order, "two")
			return errors.New("boom")
		}},
		{Description: "three", Apply: func(sys system.System) error {
			order = append(order, "three")
			return nil
		}},
	}

	results := Run(actions, system.NewFake(), Options{})

	if len(order) != 2 {
		t.Fatalf("expected 2 actions to run before stopping, got %v", order)
	}
	if err := FirstError(results); err == nil || err.Error() != "boom" {
		t.Errorf("FirstError = %v, want boom", err)
	}
	if len(results) != 2 {
		t.Errorf("expected results for the 2 actions attempted, got %d", len(results))
	}
}

func TestRun_AllSucceed(t *testing.T) {
	actions := []Action{
		{Description: "one", Apply: func(sys system.System) error { return nil }},
		{Description: "two", Apply: func(sys system.System) error { return nil }},
	}

	results := Run(actions, system.NewFake(), Options{})

	if err := FirstError(results); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}
