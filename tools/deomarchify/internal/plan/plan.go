// Package plan defines the Action type stages produce and a small executor
// that runs them with dry-run support. Stages only ever *decide* what to do
// (pure-ish, given a System to query); the Executor is the only thing that
// actually applies changes, which keeps "what would happen" trivially
// testable separately from "did it happen".
package plan

import (
	"fmt"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// Action is one planned, describable, individually-applicable step.
type Action struct {
	// Description is a one-line human-readable summary, printed in both
	// dry-run and real runs.
	Description string
	// Destructive marks actions that remove/overwrite something, for extra
	// logging emphasis; purely cosmetic, does not change execution.
	Destructive bool
	// Apply performs the action for real. Must be idempotent: safe to call
	// again after a previous partial failure.
	Apply func(sys system.System) error
}

// Result records what happened to one action during Run.
type Result struct {
	Action Action
	Ran    bool // false when DryRun was set
	Err    error
}

// Options controls how a plan is executed.
type Options struct {
	DryRun bool
	// Log receives one line per action (description, and prefix on apply).
	Log func(line string)
}

// Run executes actions in order, stopping at the first error (nil Log means
// silent). In dry-run mode nothing is applied; every action is reported.
func Run(actions []Action, sys system.System, opts Options) []Result {
	logf := opts.Log
	if logf == nil {
		logf = func(string) {}
	}

	var results []Result
	for _, a := range actions {
		prefix := "[plan]"
		if !opts.DryRun {
			prefix = "[apply]"
		}
		marker := ""
		if a.Destructive {
			marker = " (destructive)"
		}
		logf(fmt.Sprintf("%s%s %s", prefix, marker, a.Description))

		if opts.DryRun {
			results = append(results, Result{Action: a, Ran: false})
			continue
		}

		err := a.Apply(sys)
		results = append(results, Result{Action: a, Ran: true, Err: err})
		if err != nil {
			logf(fmt.Sprintf("[error] %s: %v", a.Description, err))
			return results
		}
	}
	return results
}

// FirstError returns the first non-nil error among results, or nil.
func FirstError(results []Result) error {
	for _, r := range results {
		if r.Err != nil {
			return r.Err
		}
	}
	return nil
}
