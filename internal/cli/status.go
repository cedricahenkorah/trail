package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Trail's setup and current directory context",
	Long: `Show Trail's data and config locations, plus the project linked
	to the current directory, if any. Trail can contain multiple projects;
	this command only identifies the one relevant to where you are.`,
	RunE: runStatusCmd,
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatusCmd(cmd *cobra.Command, args []string) error {
	cfg, cfgFilePath, configLoadErr := core.LoadConfig()

	if configLoadErr != nil {
		return configLoadErr
	}

	c, openErr := core.Open(cfg.DataDirectory)

	if openErr != nil {
		return openErr
	}

	projects, projectsErr := c.Projects()

	if projectsErr != nil {
		return projectsErr
	}

	cwd, cwdErr := os.Getwd()

	if cwdErr != nil {
		return fmt.Errorf("find current directory %w", cwdErr)
	}

	currentProject := "none"

	resolvedProject, resolveErr := core.ResolveProject(projects, cwd)

	switch {
	case resolveErr == nil:
		currentProject = resolvedProject.Name
	case errors.Is(resolveErr, core.ErrNoLinkedProject):
	case errors.Is(resolveErr, core.ErrProjectArchived):
		linkedProjects, linkedErr := core.LinkedTo(projects, cwd)

		if linkedErr != nil {
			return linkedErr
		}

		currentProject = linkedProjects[0].Name + " (archived)"
	default:
		return resolveErr
	}

	cmd.Printf(
		"Trail status\n\nData directory:   %s\nConfig file:      %s\nCurrent project:  %s\n",
		cfg.DataDirectory,
		cfgFilePath,
		currentProject,
	)
	return nil
}
