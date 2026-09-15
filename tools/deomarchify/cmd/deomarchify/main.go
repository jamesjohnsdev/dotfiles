// deomarchify converts an Omarchy install to a plain Arch + Hyprland
// system: vendors the omarchy tree into a self-owned checkout, adopts
// omarchy-settings' system config as plain files, and removes the
// omarchy/omarchy-settings/omarchy-nvim/omarchy-keyring packages - staged,
// idempotent, defaulting to a dry-run so nothing happens by accident.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hamst/dotfiles/tools/deomarchify/internal/checkpoint"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/driver"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/stages"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("deomarchify", flag.ContinueOnError)
	apply := fs.Bool("apply", false, "actually make changes (default: dry-run / plan-only)")
	stageFlag := fs.Int("stage", -1, "run only this stage number (default: all stages in order)")
	force := fs.Bool("force", false, "re-run a stage even if the checkpoint says it already completed")
	listStages := fs.Bool("list", false, "list all stages and exit")
	status := fs.Bool("status", false, "show checkpoint status and exit")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "deomarchify: convert an Omarchy install to plain Arch, staged and idempotent.")
		fmt.Fprintln(os.Stderr, "\nBy default this only PLANS and prints what it would do. Pass -apply to actually run it.")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := config.Default()
	sys := system.NewReal()

	if *listStages {
		for _, s := range stages.All() {
			fmt.Printf("%d  %s\n", s.Number, s.Name)
		}
		return 0
	}

	if *status {
		return printStatus(sys, cfg)
	}

	if !*apply {
		fmt.Println("Dry-run (no -apply passed): showing what would happen, changing nothing.")
	}

	outcomes, err := driver.Run(sys, cfg, stages.All(), driver.Options{
		DryRun:      !*apply,
		TargetStage: *stageFlag,
		Force:       *force,
		Log:         func(line string) { fmt.Println(line) },
	})

	printSummary(outcomes)

	if err != nil {
		fmt.Fprintln(os.Stderr, "\nFAILED:", err)
		fmt.Fprintln(os.Stderr, "Fix the underlying issue and re-run - stages are idempotent and completed stages are skipped automatically.")
		fmt.Fprintln(os.Stderr, "For anything more serious: sudo snapper rollback (see Stage 0's snapshot).")
		return 1
	}

	return 0
}

func printStatus(sys system.System, cfg config.Config) int {
	cp, err := checkpoint.Load(sys, cfg.CheckpointPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "loading checkpoint:", err)
		return 1
	}
	for _, s := range stages.All() {
		state := "pending"
		if cp.IsDone(s.Number) {
			state = "done (" + cp.Completed[s.Number] + ")"
		}
		fmt.Printf("%d  %-24s %s\n", s.Number, s.Name, state)
	}
	return 0
}

func printSummary(outcomes []driver.StageOutcome) {
	fmt.Println("\n--- summary ---")
	for _, o := range outcomes {
		switch {
		case o.Skipped:
			fmt.Printf("%d  %-24s skipped (already done)\n", o.Stage.Number, o.Stage.Name)
		case o.NoOp:
			fmt.Printf("%d  %-24s nothing to do\n", o.Stage.Number, o.Stage.Name)
		case o.Err != nil:
			fmt.Printf("%d  %-24s FAILED: %v\n", o.Stage.Number, o.Stage.Name, o.Err)
		default:
			fmt.Printf("%d  %-24s %d action(s)\n", o.Stage.Number, o.Stage.Name, len(o.Results))
		}
	}
}
