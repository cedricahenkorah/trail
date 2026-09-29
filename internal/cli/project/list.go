/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects tracked by Trail",
	Long: `List projects from Trail's configured data directory, showing their
	names and tags. By default, only active projects are shown. Use --all
	to include archived projects, or --tag to filter by a single tag.
	Tag matching ignores case and requires a complete tag match.`,
	RunE: runProjectListCommand,
	Args: cobra.NoArgs,
}

func init() {
	listCmd.Flags().Bool("all", false, "include archived projects")
	listCmd.Flags().StringP("tag", "t", "", "filter projects by tag")
}

func runProjectListCommand(cmd *cobra.Command, args []string) error {

	cfg, _, configLoadErr := config.Load()

	if configLoadErr != nil {
		return configLoadErr
	}

	dataDirErr := config.ValidateDataDirectory(cfg.DataDirectory)

	if dataDirErr != nil {
		return dataDirErr
	}

	projectRegistry, projectRegistryErr := LoadProjectRegistry(cfg.DataDirectory)

	if projectRegistryErr != nil {
		return projectRegistryErr
	}

	all, allErr := cmd.Flags().GetBool("all")

	if allErr != nil {
		return fmt.Errorf("error reading the all flag: %w", allErr)
	}

	tag, tagErr := cmd.Flags().GetString("tag")

	if tagErr != nil {
		return fmt.Errorf("error reading the tag flag: %w", tagErr)
	}

	tag = strings.TrimSpace(tag)

	hasTagFilter := cmd.Flags().Changed("tag")

	if hasTagFilter && tag == "" {
		return fmt.Errorf("tag cannot be omitted if the flag is passed")
	}

	var selectedProjects []ProjectRecord

	for _, registeredProject := range projectRegistry.Projects {
		if registeredProject.Archived && !all {
			continue
		}

		if hasTagFilter {
			matchesTag := false

			for _, projectTag := range registeredProject.Tags {
				if strings.EqualFold(projectTag, tag) {
					matchesTag = true
					break
				}
			}

			if !matchesTag {
				continue
			}
		}

		selectedProjects = append(selectedProjects, registeredProject)
	}

	for _, selectedProject := range selectedProjects {
		projectDir := filepath.Join(cfg.DataDirectory, "projects", selectedProject.Name)

		projectDirInfo, projectDirInfoErr := os.Stat(projectDir)

		if os.IsNotExist(projectDirInfoErr) {
			return fmt.Errorf("project %q is registered but its folder is missing: %s", selectedProject.Name, projectDir)
		}

		if projectDirInfoErr != nil {
			return fmt.Errorf("check folder for project %q: %w", selectedProject.Name, projectDirInfoErr)
		}

		if !projectDirInfo.IsDir() {
			return fmt.Errorf("project %q has a path that is not a directory: %s", selectedProject.Name, projectDir)
		}
	}

	if len(projectRegistry.Projects) == 0 {
		cmd.Println("No projects yet. Add one with trail project add.")
		return nil
	}

	if len(selectedProjects) == 0 {
		cmd.Println("No projects match the current filters.")

		if !all {
			cmd.Println("Try trail project list --all to include archived projects.")
		}

		return nil
	}

	heading := "Projects"

	if hasTagFilter && all {
		heading += fmt.Sprintf(" (tag: %s; including archived)", tag)
	} else if hasTagFilter {
		heading += fmt.Sprintf(" (tag: %s)", tag)
	} else if all {
		heading += " (including archived)"
	}

	cmd.Println(heading)
	cmd.Println()

	for _, project := range selectedProjects {
		line := project.Name

		if len(project.Tags) > 0 {
			line += "  [" + strings.Join(project.Tags, ", ") + "]"
		}

		if project.Archived {
			line += "  (archived)"
		}

		cmd.Println(line)
	}

	return nil
}
