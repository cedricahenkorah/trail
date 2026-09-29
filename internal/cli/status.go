/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/cedricahenkorah/trail/internal/cli/project"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runStatusCmd(cmd *cobra.Command, args []string) error {
	cfg, cfgFilePath, configLoadErr := config.Load()

	if configLoadErr != nil {
		return configLoadErr
	}

	dataDirErr := config.ValidateDataDirectory(cfg.DataDirectory)

	if dataDirErr != nil {
		return dataDirErr
	}

	projectRegistry, projectRegistryErr := project.LoadProjectRegistry(cfg.DataDirectory)

	if projectRegistryErr != nil {
		return projectRegistryErr
	}

	cwd, cwdErr := os.Getwd()

	if cwdErr != nil {
		return fmt.Errorf("find current directory %w", cwdErr)
	}

	var matchedProjects []project.ProjectRecord

	for _, registeredProject := range projectRegistry.Projects {
		if registeredProject.LinkedPath == "" {
			continue
		}

		rel, relErr := filepath.Rel(registeredProject.LinkedPath, cwd)

		if relErr != nil {
			return fmt.Errorf("compare current directory with project %q: %w",
				registeredProject.Name, relErr)
		}

		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			matchedProjects = append(matchedProjects, registeredProject)
		}
	}

	currentProject := "none"

	switch len(matchedProjects) {
	case 0:
	case 1:
		currentProject = matchedProjects[0].Name
	default:
		names := make([]string, 0, len(matchedProjects))
		for _, matched := range matchedProjects {
			names = append(names, matched.Name)
		}
		return fmt.Errorf(
			"more than one project is linked to the current directory: %s",
			strings.Join(names, ", "),
		)
	}

	cmd.Printf(
		"Trail status\n\nData directory:   %s\nConfig file:      %s\nCurrent project:  %s\n",
		cfg.DataDirectory,
		cfgFilePath,
		currentProject,
	)
	return nil
}
