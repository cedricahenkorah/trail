package cli

import (
	"errors"
	"fmt"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/spf13/cobra"
)

// addHints wraps the RunE of cmd and its subcommands so core errors get CLI hints.
func addHints(cmd *cobra.Command) {
	if run := cmd.RunE; run != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return withHint(run(cmd, args))
		}
	}

	for _, subcommand := range cmd.Commands() {
		addHints(subcommand)
	}
}

// withHint adds the trail command that fixes err, if there is one.
func withHint(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, core.ErrNotInitialized):
		return fmt.Errorf("%w; run trail init", err)
	case errors.Is(err, core.ErrEmptyConfig), errors.Is(err, core.ErrNoDataDirectory):
		return fmt.Errorf("%w; run trail init to repair it", err)
	case errors.Is(err, core.ErrProjectNotFound):
		return fmt.Errorf("%w; run trail project list to see available projects", err)
	default:
		return err
	}
}
