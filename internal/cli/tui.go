package cli

import (
	"os"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/cedricahenkorah/trail/internal/tui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().StringP("project", "p", "", "open the TUI on an active project by name")
}

// runTUI opens the TUI, or prints help if stdout isn't a terminal.
func runTUI(cmd *cobra.Command, args []string) error {
	if !isTerminal(os.Stdout) {
		return cmd.Help()
	}

	cfg, _, configLoadErr := core.LoadConfig()

	if configLoadErr != nil {
		return configLoadErr
	}

	c, openErr := core.Open(cfg.DataDirectory)

	if openErr != nil {
		return openErr
	}

	projectName, projectErr := startingProject(cmd, c)

	if projectErr != nil {
		return projectErr
	}

	return tui.Run(c, tui.Options{Project: projectName})
}

// startingProject returns the --project name, else the active project linked
// to the current directory, else "" so the TUI picks one.
func startingProject(cmd *cobra.Command, c *core.Core) (string, error) {
	flagProject, projectGiven, projectErr := prompt.ProjectFlag(cmd, c, "opening it")

	if projectErr != nil {
		return "", projectErr
	}

	if projectGiven {
		return flagProject.Name, nil
	}

	projects, projectsErr := c.Projects()

	if projectsErr != nil {
		return "", projectsErr
	}

	linkedProject, linkedErr := prompt.LinkedProject(projects)

	if prompt.Unresolved(linkedErr) {
		return "", nil
	}

	if linkedErr != nil {
		return "", linkedErr
	}

	return linkedProject.Name, nil
}

// isTerminal reports whether f is a terminal.
func isTerminal(f *os.File) bool {
	info, statErr := f.Stat()

	return statErr == nil && info.Mode()&os.ModeCharDevice != 0
}
