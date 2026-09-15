package stages

import (
	"github.com/hamst/dotfiles/tools/deomarchify/internal/config"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/plan"
	"github.com/hamst/dotfiles/tools/deomarchify/internal/system"
)

// Stage bundles a stage's number, name, and planner for the CLI driver.
type Stage struct {
	Number int
	Name   string
	Plan   func(sys system.System, cfg config.Config) ([]plan.Action, error)
}

// All returns every stage in execution order.
func All() []Stage {
	return []Stage{
		{0, "safety-net", Stage0Plan},
		{1, "mirrors", Stage1Plan},
		{2, "remove-omarchy-nvim", Stage2Plan},
		{3, "vendor-omarchy", Stage3Plan},
		{4, "adopt-omarchy-settings", Stage4Plan},
		{5, "remove-omarchy-keyring", Stage5Plan},
		{6, "cleanup", Stage6Plan},
	}
}
