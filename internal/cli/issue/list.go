package issue

import (
	"fmt"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Browse project issues and close or reopen them",
	Long: `Interactively browse Markdown files in a project's issues directory,
	showing unresolved and total issue counts for each file.

	Choose a file to view its issues in an interactive checklist. Closed
	issues start checked. Use the arrow keys to move between issues and Space
	to close or reopen them. Press Enter to save your changes and return to
	the file picker with updated counts.

	Only changed checkbox markers are updated; issue text and formatting are
	preserved. If nothing changed, no file is written. Choose Done in the
	file picker to exit.

	Use --project to select an active project by name. Otherwise, Trail uses
	the single active project linked to the current directory, or prompts
	you to choose an active project if there is no unique match.`,
	RunE: runIssueListCommand,
	Args: cobra.NoArgs,
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "select an active project by name")
}

func runIssueListCommand(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("project %q is archived; restore it before listing its issues", foundProject.Name)
		}

		matchedProject = foundProject
	} else {
		resolvedProject, resolveErr := prompt.ResolveProject(projects, true)

		if resolveErr != nil {
			return resolveErr
		}

		matchedProject = resolvedProject
	}

	for {
		summaries, listErr := c.Files(matchedProject, core.Issues)

		if listErr != nil {
			return listErr
		}

		if len(summaries) == 0 {
			cmd.Printf(
				"No issue files yet for %s. Add one with trail issue add --project %s.\n",
				matchedProject.Name,
				matchedProject.Name,
			)
			return nil
		}

		var options []huh.Option[string]

		for _, summary := range summaries {
			label := fmt.Sprintf("%s — %d unresolved / %d total", summary.Name, summary.Open, summary.Total)
			options = append(options, huh.NewOption(label, summary.Name))
		}

		options = append(options, huh.NewOption("Done", ""))

		chosenFile := summaries[0].Name

		cmd.Printf("Issues for %s\n\n", matchedProject.Name)

		selectFileErr := prompt.Run(huh.NewSelect[string]().
			Title("Select an issue file").
			Value(&chosenFile).
			Options(options...))

		if selectFileErr != nil {
			return fmt.Errorf("choose file: %w", selectFileErr)
		}

		if chosenFile == "" {
			return nil
		}

		chosenFilePath, chosenFilePathErr := c.FilePath(matchedProject, core.Issues, chosenFile)

		if chosenFilePathErr != nil {
			return chosenFilePathErr
		}

		content, readErr := c.ReadFile(matchedProject, core.Issues, chosenFile)

		if readErr != nil {
			return readErr
		}

		items := core.ParseItems(content)

		cmd.Printf("%s / %s\n\n", matchedProject.Name, chosenFile)

		if len(items) == 0 {
			cmd.Printf("No checkbox issues in %s.\n", chosenFile)

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

		var issueOptions []huh.Option[int]
		var closedIndexes []int

		for i, item := range items {
			issueOptions = append(issueOptions, huh.NewOption(item.Text, i))

			if item.Done {
				closedIndexes = append(closedIndexes, i)
			}
		}

		toggleErr := prompt.Run(huh.NewMultiSelect[int]().
			Title("Issues").
			Description("↑/↓ to move • Space to close/reopen • Enter to save and go back").
			Options(issueOptions...).
			Value(&closedIndexes))

		if toggleErr != nil {
			return fmt.Errorf("select issues: %w", toggleErr)
		}

		changed, saveErr := core.SetDone(chosenFilePath, content, items, closedIndexes)

		if saveErr != nil {
			return saveErr
		}

		if changed {
			cmd.Printf("Saved issues in %s.\n", chosenFile)
		}
	}
}
