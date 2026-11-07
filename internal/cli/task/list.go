/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package task

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/checklist"
	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/cedricahenkorah/trail/internal/cli/log"
	"github.com/cedricahenkorah/trail/internal/cli/project"
	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

type taskItem struct {
	Text         string
	Done         bool
	MarkerOffset int
}

type taskFileSummary struct {
	Name  string
	Open  int
	Total int
}

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
	cfg, _, configLoadErr := config.Load()

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

	projectFilter, projectFilterErr := cmd.Flags().GetString("project")

	if projectFilterErr != nil {
		return fmt.Errorf("error reading the project filter: %w", projectFilterErr)
	}

	projectFilter = strings.TrimSpace(projectFilter)

	if cmd.Flags().Changed("project") && projectFilter == "" {
		return fmt.Errorf("--project requires a project name")
	}

	var matchedProject project.ProjectRecord

	if projectFilter != "" {
		for _, registeredProject := range projectRegistry.Projects {
			if !strings.EqualFold(registeredProject.Name, projectFilter) {
				continue
			}

			if registeredProject.Archived {
				return fmt.Errorf("project %q is archived; restore it before filtering by this project", registeredProject.Name)
			}

			matchedProject = registeredProject
			break
		}

		if matchedProject.Name == "" {
			return fmt.Errorf("project %q is not registered; run trail project list to see available projects", projectFilter)
		}
	}

	if !cmd.Flags().Changed("project") {
		matchedProjects, matchedProjectsErr := log.FindProjectsLinkedToCurrentDirectory(projectRegistry)

		if matchedProjectsErr != nil {
			return matchedProjectsErr
		}

		var activeProjects []project.ProjectRecord

		for _, registeredProject := range projectRegistry.Projects {
			if registeredProject.Archived {
				continue
			}

			activeProjects = append(activeProjects, registeredProject)
		}

		if len(activeProjects) == 0 {
			return fmt.Errorf("no active projects available; add one with trail project add")
		}

		var activeMatches []project.ProjectRecord

		for _, matched := range matchedProjects {
			if matched.Archived {
				continue
			}

			activeMatches = append(activeMatches, matched)
		}

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

	matchedProjectDir, matchedProjectDirErr := project.ResolveFolder(cfg.DataDirectory, matchedProject)

	if matchedProjectDirErr != nil {
		return matchedProjectDirErr
	}

	taskDir, taskDirErr := checklist.ResolveDir(matchedProjectDir, checklist.Tasks)

	if taskDirErr != nil {
		return taskDirErr
	}

	for {
		entries, entriesErr := os.ReadDir(taskDir)

		if entriesErr != nil && !os.IsNotExist(entriesErr) {
			return fmt.Errorf("read task directory %s: %w", taskDir, entriesErr)
		}

		var taskFiles []string

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			taskFiles = append(taskFiles, entry.Name())
		}

		if len(taskFiles) == 0 {
			cmd.Printf(
				"No task files yet for %s. Add one with trail task add --project %s.\n",
				matchedProject.Name,
				matchedProject.Name,
			)
			return nil
		}

		var summaries []taskFileSummary

		for _, file := range taskFiles {
			path := filepath.Join(taskDir, file)

			content, readErr := os.ReadFile(path)

			if readErr != nil {
				return fmt.Errorf("read task file %s: %w", path, readErr)
			}

			items := parseTaskItems(content)

			openCount := 0

			for _, item := range items {
				if !item.Done {
					openCount++
				}
			}

			totalCount := len(items)

			summaries = append(summaries, taskFileSummary{
				Name:  file,
				Open:  openCount,
				Total: totalCount,
			})
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

		chosenFile := taskFiles[0]

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

		parsedChosenFileTaskItems := parseTaskItems(chosenFileContent)

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

		updatedContent := bytes.Clone(chosenFileContent)
		hasChanges := false

		for i, item := range parsedChosenFileTaskItems {
			nowDone := slices.Contains(completedIndexes, i)

			if nowDone == item.Done {
				continue
			}

			if nowDone {
				updatedContent[item.MarkerOffset] = 'x'
			} else {
				updatedContent[item.MarkerOffset] = ' '
			}

			hasChanges = true
		}

		if !hasChanges {
			continue
		}

		saveErr := os.WriteFile(chosenFilePath, updatedContent, 0600)
		if saveErr != nil {
			return fmt.Errorf("save task file %s: %w", chosenFilePath, saveErr)
		}

		cmd.Printf("Saved tasks in %s.\n", chosenFile)
	}
}

func parseTaskItems(content []byte) []taskItem {
	var items []taskItem
	offset := 0

	for _, rawLine := range bytes.SplitAfter(content, []byte("\n")) {
		lineOffset := offset
		offset += len(rawLine)

		line := strings.TrimSuffix(string(rawLine), "\n")
		line = strings.TrimSuffix(line, "\r")

		isTask := strings.HasPrefix(line, "- [ ] ") ||
			strings.HasPrefix(line, "- [x] ") ||
			strings.HasPrefix(line, "- [X] ")

		if !isTask {
			continue
		}

		text := line[6:]

		if strings.TrimSpace(text) == "" {
			continue
		}

		items = append(items, taskItem{
			Text:         text,
			Done:         line[3] != ' ',
			MarkerOffset: lineOffset + 3,
		})
	}

	return items
}
