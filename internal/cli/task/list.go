package task

import (
	"fmt"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

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

	matchedProject, projectGiven, projectErr := prompt.ProjectFlag(cmd, c, "filtering by this project")

	if projectErr != nil {
		return projectErr
	}

	if !projectGiven {
		resolvedProject, resolveErr := prompt.ResolveProject(projects, true)

		if resolveErr != nil {
			return resolveErr
		}

		matchedProject = resolvedProject
	}

	for {
		summaries, listErr := c.Files(matchedProject, core.Tasks)

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

		chosenFilePath, chosenFilePathErr := c.FilePath(matchedProject, core.Tasks, chosenFile)

		if chosenFilePathErr != nil {
			return chosenFilePathErr
		}

		chosenFileContent, chosenFileContentErr := c.ReadFile(matchedProject, core.Tasks, chosenFile)

		if chosenFileContentErr != nil {
			return chosenFileContentErr
		}

		parsedChosenFileTaskItems := core.ParseItems(chosenFileContent)

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

		changed, saveErr := core.SetDone(chosenFilePath, chosenFileContent, parsedChosenFileTaskItems, completedIndexes)

		if saveErr != nil {
			return saveErr
		}

		if !changed {
			continue
		}

		cmd.Printf("Saved tasks in %s.\n", chosenFile)
	}
}
