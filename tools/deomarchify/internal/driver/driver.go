// Package driver runs the stage list against a checkpoint file, the piece
// that turns individual stage Plan()s into "run everything not yet done, in
// order, stop at the first failure, skip what's already complete unless
// forced". Kept separate from main() so the loop/skip/checkpoint logic is
// unit tested against a fake System instead of only exercised by hand.
package driver

import (
	"fmt"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/checkpoint"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/stages"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// Options controls one driver run.
type Options struct {
	DryRun bool
	// TargetStage, if >= 0, runs only that stage number. Negative means run
	// every stage in order.
	TargetStage int
	// Force re-runs a stage even if the checkpoint says it already completed.
	Force bool
	Log   func(line string)
}

// StageOutcome summarizes what happened to one stage during a run.
type StageOutcome struct {
	Stage   stages.Stage
	Skipped bool // already done per checkpoint, and Force wasn't set
	NoOp    bool // ran, but Plan() produced zero actions
	Results []plan.Result
	Err     error
}

// Run executes stageList in order per opts, updating the checkpoint after
// each one that completes with no error (dry-run never updates the
// checkpoint). Stops at the first stage that errors. stageList is a
// parameter rather than always stages.All() so driver-loop logic
// (skip/force/checkpoint/stop-on-error) can be tested with fake stages,
// independent of the real stage-planning logic tested in internal/stages.
func Run(sys system.System, cfg config.Config, stageList []stages.Stage, opts Options) ([]StageOutcome, error) {
	logf := opts.Log
	if logf == nil {
		logf = func(string) {}
	}

	cp, err := checkpoint.Load(sys, cfg.CheckpointPath)
	if err != nil {
		return nil, fmt.Errorf("loading checkpoint: %w", err)
	}

	var outcomes []StageOutcome
	for _, stage := range stageList {
		if opts.TargetStage >= 0 && stage.Number != opts.TargetStage {
			continue
		}

		if cp.IsDone(stage.Number) && !opts.Force {
			logf(fmt.Sprintf("== stage %d (%s): already completed, skipping ==", stage.Number, stage.Name))
			outcomes = append(outcomes, StageOutcome{Stage: stage, Skipped: true})
			continue
		}

		logf(fmt.Sprintf("== stage %d (%s) ==", stage.Number, stage.Name))
		actions, err := stage.Plan(sys, cfg)
		if err != nil {
			outcome := StageOutcome{Stage: stage, Err: fmt.Errorf("planning stage %d (%s): %w", stage.Number, stage.Name, err)}
			outcomes = append(outcomes, outcome)
			return outcomes, outcome.Err
		}
		if len(actions) == 0 {
			logf(fmt.Sprintf("stage %d (%s): nothing to do", stage.Number, stage.Name))
			outcomes = append(outcomes, StageOutcome{Stage: stage, NoOp: true})
			if !opts.DryRun {
				cp, err = checkpoint.MarkDone(sys, cfg.CheckpointPath, cp, stage.Number, sys.Now())
				if err != nil {
					return outcomes, fmt.Errorf("saving checkpoint after stage %d: %w", stage.Number, err)
				}
			}
			continue
		}

		results := plan.Run(actions, sys, plan.Options{DryRun: opts.DryRun, Log: logf})
		outcome := StageOutcome{Stage: stage, Results: results, Err: plan.FirstError(results)}
		outcomes = append(outcomes, outcome)
		if outcome.Err != nil {
			return outcomes, fmt.Errorf("stage %d (%s) failed: %w", stage.Number, stage.Name, outcome.Err)
		}

		if !opts.DryRun {
			cp, err = checkpoint.MarkDone(sys, cfg.CheckpointPath, cp, stage.Number, sys.Now())
			if err != nil {
				return outcomes, fmt.Errorf("saving checkpoint after stage %d: %w", stage.Number, err)
			}
		}
	}

	return outcomes, nil
}
