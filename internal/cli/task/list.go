/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/checklist"
	"github.com/cedricahenkorah/trail/internal/cli/log"
	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Browse project tasks and mark them done or open",
	Long: `Interactively browse Markdown files in a project's tasks directory,
	showing open and total task counts for each file.

	Choose a file to view its tasks in an interactive checklist. Completed
	tasks start checked. Use the arrow keys to move between tasks and Space
	to mark them done or reopen them. Press Enter to save your changes and
	return to the file picker with updated counts.

	Only changed checkbox markers are updated; task text, formatting, and
	due dates are preserved. If nothing changed, no file is written.
	Choose Done in the file picker to exit.

	Use --project to select an active project by name. Otherwise, Trail uses
	the single active project linked to the current directory, or prompts
	you to choose an active project if there is no unique match.`,
	RunE: runTaskListCommand,
	Args: cobra.NoArgs,
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "select an active project by name")
}

func runTaskListCommand(cmd *cobra.Command, args []string) error {
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

	projectFilter, projectFilterErr := cmd.Flags().GetString("project")

	if projectFilterErr != nil {
		return fmt.Errorf("error reading the project filter: %w", projectFilterErr)
	}

	projectFilter = strings.TrimSpace(projectFilter)

	if cmd.Flags().Changed("project") && projectFilter == "" {
		return fmt.Errorf("--project requires a project name")
	}

	var matchedProject core.Project

	if projectFilter != "" {
		foundProject, findErr := c.Project(projectFilter)

		if findErr != nil {
			return findErr
		}

		if foundProject.Archived {
			return fmt.Errorf("project %q is archived; restore it before filtering by this project", foundProject.Name)
		}

		matchedProject = foundProject
	}

	if !cmd.Flags().Changed("project") {
		matchedProjects, matchedProjectsErr := log.FindProjectsLinkedToCurrentDirectory(projects)

		if matchedProjectsErr != nil {
			return matchedProjectsErr
		}

		activeProjects := core.ActiveProjects(projects)

		if len(activeProjects) == 0 {
			return fmt.Errorf("no active projects available; add one with trail project add")
		}

		activeMatches := core.ActiveProjects(matchedProjects)

		switch len(activeMatches) {
		case 1:
			matchedProject = activeMatches[0]

		default:
			activeProject, activeProjectErr := log.ChooseActiveProject(activeProjects)

			if activeProjectErr != nil {
				return activeProjectErr
			}

			matchedProject = activeProject
		}
	}

	matchedProjectDir, matchedProjectDirErr := c.ProjectDir(matchedProject)

	if matchedProjectDirErr != nil {
		return matchedProjectDirErr
	}

	taskDir, taskDirErr := checklist.ResolveDir(matchedProjectDir, checklist.Tasks)

	if taskDirErr != nil {
		return taskDirErr
	}

	for {
		summaries, listErr := checklist.ListFiles(taskDir)

		if listErr != nil {
			return listErr
		}

		if len(summaries) == 0 {
			cmd.Printf(
				"No task files yet for %s. Add one with trail task add --project %s.\n",
				matchedProject.Name,
				matchedProject.Name,
			)
			return nil
		}

		var options []huh.Option[string]

		for _, summary := range summaries {
			label := fmt.Sprintf(
				"%s — %d open / %d total",
				summary.Name,
				summary.Open,
				summary.Total,
			)

			options = append(options, huh.NewOption(label, summary.Name))
		}

		options = append(options, huh.NewOption("Done", ""))

		chosenFile := summaries[0].Name

		cmd.Printf("Tasks for %s\n\n", matchedProject.Name)

		selectFileErr := prompt.Run(huh.NewSelect[string]().
			Title("Select a task file").
			Value(&chosenFile).
			Options(options...))

		if selectFileErr != nil {
			return fmt.Errorf("choose file: %w", selectFileErr)
		}

		if chosenFile == "" {
			return nil
		}

		chosenFilePath := filepath.Join(taskDir, chosenFile)

		chosenFileContent, chosenFileContentErr := os.ReadFile(chosenFilePath)

		if chosenFileContentErr != nil {
			return fmt.Errorf("read task file: %w", chosenFileContentErr)
		}

		parsedChosenFileTaskItems := checklist.ParseItems(chosenFileContent)

		cmd.Printf("%s / %s\n\n", matchedProject.Name, chosenFile)

		if len(parsedChosenFileTaskItems) == 0 {
			cmd.Printf("No checkbox tasks in %s.\n", chosenFile)

			var input string

			backErr := prompt.Run(huh.NewInput().
				Title("Press Enter to go back").
				Validate(func(value string) error {
					if value != "" {
						return fmt.Errorf("press Enter without typing anything")
					}
					return nil
				}).
				Value(&input))

			if backErr != nil {
				return fmt.Errorf("return to file picker: %w", backErr)
			}

			continue
		}

		var taskOptions []huh.Option[int]
		var completedIndexes []int

		for i, item := range parsedChosenFileTaskItems {
			taskOptions = append(taskOptions, huh.NewOption(item.Text, i))

			if item.Done {
				completedIndexes = append(completedIndexes, i)
			}
		}

		toggleTasksErr := prompt.Run(huh.NewMultiSelect[int]().
			Title("Tasks").
			Description("↑/↓ to move • Space to check/uncheck • Enter to save and go back").
			Options(taskOptions...).
			Value(&completedIndexes))

		if toggleTasksErr != nil {
			return fmt.Errorf("select tasks: %w", toggleTasksErr)
		}

		changed, saveErr := checklist.SetDone(chosenFilePath, chosenFileContent, parsedChosenFileTaskItems, completedIndexes)

		if saveErr != nil {
			return saveErr
		}

		if !changed {
			continue
		}

		cmd.Printf("Saved tasks in %s.\n", chosenFile)
	}
}
