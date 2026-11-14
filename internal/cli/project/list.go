package project

import (
	"fmt"
	"strings"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/spf13/cobra"
)

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

	cfg, _, configLoadErr := core.LoadConfig()

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

	selectedProjects := core.FilterProjects(projects, all, tag)

	for _, selectedProject := range selectedProjects {
		_, projectDirErr := c.ProjectDir(selectedProject)

		if projectDirErr != nil {
			return projectDirErr
		}
	}

	if len(projects) == 0 {
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
