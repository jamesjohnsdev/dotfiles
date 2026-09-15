package driver

import (
	"errors"
	"testing"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/checkpoint"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/stages"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// These tests exercise driver.Run's loop/skip/checkpoint logic directly
// against fake stages built in-line, without going through the real stage
// planners in internal/stages (those have their own tests). That way a
// driver-loop bug can't hide behind stage-planning complexity, and vice
// versa - Run takes the stage list as a parameter specifically to allow this.

func testCfg() config.Config {
	return config.Config{CheckpointPath: "/home/james/.deomarchify/checkpoint.json"}
}

func fakeStage(number int, name string, actions []plan.Action) stages.Stage {
	return stages.Stage{
		Number: number,
		Name:   name,
		Plan: func(sys system.System, cfg config.Config) ([]plan.Action, error) {
			return actions, nil
		},
	}
}

func failingStage(number int, name string, err error) stages.Stage {
	return stages.Stage{
		Number: number,
		Name:   name,
		Plan: func(sys system.System, cfg config.Config) ([]plan.Action, error) {
			return nil, err
		},
	}
}

func TestRun_DryRunNeverWritesCheckpoint(t *testing.T) {
	fake := system.NewFake()
	applied := false
	stageList := []stages.Stage{
		fakeStage(0, "one", []plan.Action{{Description: "x", Apply: func(system.System) error { applied = true; return nil }}}),
	}

	_, err := Run(fake, testCfg(), stageList, Options{DryRun: true, TargetStage: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied {
		t.Errorf("dry-run should never call Apply")
	}
	cp, _ := checkpoint.Load(fake, testCfg().CheckpointPath)
	if cp.IsDone(0) {
		t.Errorf("dry-run should never mark a stage done")
	}
}

func TestRun_SkipsAlreadyCompletedStage(t *testing.T) {
	fake := system.NewFake()
	runCount := 0
	stageList := []stages.Stage{
		fakeStage(0, "one", []plan.Action{{Description: "x", Apply: func(system.System) error { runCount++; return nil }}}),
	}

	cfg := testCfg()
	cp, _ := checkpoint.Load(fake, cfg.CheckpointPath)
	checkpoint.MarkDone(fake, cfg.CheckpointPath, cp, 0, "t0")

	outcomes, err := Run(fake, cfg, stageList, Options{TargetStage: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runCount != 0 {
		t.Errorf("expected stage to be skipped, but it ran %d times", runCount)
	}
	if len(outcomes) != 1 || !outcomes[0].Skipped {
		t.Errorf("expected a Skipped outcome, got %+v", outcomes)
	}
}

func TestRun_ForceReRunsCompletedStage(t *testing.T) {
	fake := system.NewFake()
	runCount := 0
	stageList := []stages.Stage{
		fakeStage(0, "one", []plan.Action{{Description: "x", Apply: func(system.System) error { runCount++; return nil }}}),
	}

	cfg := testCfg()
	cp, _ := checkpoint.Load(fake, cfg.CheckpointPath)
	checkpoint.MarkDone(fake, cfg.CheckpointPath, cp, 0, "t0")

	_, err := Run(fake, cfg, stageList, Options{TargetStage: -1, Force: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runCount != 1 {
		t.Errorf("expected forced re-run to call Apply once, got %d", runCount)
	}
}

func TestRun_StopsAtFirstFailingStageAndDoesNotMarkItDone(t *testing.T) {
	fake := system.NewFake()
	secondRan := false
	stageList := []stages.Stage{
		fakeStage(0, "fails", []plan.Action{{Description: "x", Apply: func(system.System) error { return errors.New("boom") }}}),
		fakeStage(1, "never runs", []plan.Action{{Description: "y", Apply: func(system.System) error { secondRan = true; return nil }}}),
	}

	cfg := testCfg()
	_, err := Run(fake, cfg, stageList, Options{TargetStage: -1})
	if err == nil {
		t.Fatalf("expected error")
	}
	if secondRan {
		t.Errorf("stage after the failing one should not have run")
	}
	cp, _ := checkpoint.Load(fake, cfg.CheckpointPath)
	if cp.IsDone(0) {
		t.Errorf("failed stage must not be marked done")
	}
}

func TestRun_PlanningErrorStopsTheRun(t *testing.T) {
	fake := system.NewFake()
	stageList := []stages.Stage{
		failingStage(0, "bad plan", errors.New("cannot query pacman")),
	}

	_, err := Run(fake, testCfg(), stageList, Options{TargetStage: -1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRun_TargetStageRunsOnlyThatOne(t *testing.T) {
	fake := system.NewFake()
	var ran []int
	stageList := []stages.Stage{
		fakeStage(0, "a", []plan.Action{{Description: "x", Apply: func(system.System) error { ran = append(ran, 0); return nil }}}),
		fakeStage(1, "b", []plan.Action{{Description: "y", Apply: func(system.System) error { ran = append(ran, 1); return nil }}}),
	}

	_, err := Run(fake, testCfg(), stageList, Options{TargetStage: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ran) != 1 || ran[0] != 1 {
		t.Errorf("expected only stage 1 to run, got %v", ran)
	}
}

func TestRun_NoOpStageStillGetsMarkedDone(t *testing.T) {
	fake := system.NewFake()
	stageList := []stages.Stage{
		fakeStage(0, "empty", nil),
	}

	cfg := testCfg()
	outcomes, err := Run(fake, cfg, stageList, Options{TargetStage: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outcomes) != 1 || !outcomes[0].NoOp {
		t.Fatalf("expected a NoOp outcome, got %+v", outcomes)
	}
	cp, _ := checkpoint.Load(fake, cfg.CheckpointPath)
	if !cp.IsDone(0) {
		t.Errorf("no-op stage should still be marked done so it's not re-checked forever")
	}
}
