package log

import (
	"fmt"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [note]",
	Short: "Add a note to a project's daily log",
	Long: `Add a note to a project's daily Markdown log. Supply a note to add it
	directly, or omit it to enter a note interactively. Trail uses the project
	linked to the current directory when there is one clear match; otherwise,
	interactive mode asks you to choose a project. The date defaults to today.
	Use --project or --date to select either explicitly.`,
	RunE: runLogAddCmd,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	addCmd.Flags().StringP("project", "p", "", "project to add the note to")
	addCmd.Flags().StringP("date", "d", "", "log date in YYYY-MM-DD format (default: today)")
}

func runLogAddCmd(cmd *cobra.Command, args []string) error {
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

	matchedProject, projectGiven, projectErr := prompt.ProjectFlag(cmd, c, "adding a log entry")

	if projectErr != nil {
		return projectErr
	}

	dateFilter, dateFilterErr := cmd.Flags().GetString("date")

	if dateFilterErr != nil {
		return fmt.Errorf("error reading the date filter: %w", dateFilterErr)
	}

	dateFilter = strings.TrimSpace(dateFilter)

	if cmd.Flags().Changed("date") && dateFilter == "" {
		return fmt.Errorf("--date requires a date in YYYY-MM-DD format")
	}

	date := time.Now().Format("2006-01-02")

	if dateFilter != "" {
		validatedDate, validatedDateErr := core.ValidateDate(dateFilter)

		if validatedDateErr != nil {
			return validatedDateErr
		}

		date = validatedDate
	}

	if len(args) == 1 {
		note := strings.TrimSpace(args[0])

		if note == "" {
			return fmt.Errorf("note cannot be blank")
		}

		if !projectGiven {
			resolvedProject, resolveErr := prompt.ResolveProject(projects, false)

			if resolveErr != nil {
				return resolveErr
			}

			matchedProject = resolvedProject
		}

		logFilePath, logFilePathErr := c.AddLogEntry(matchedProject, date, note)

		if logFilePathErr != nil {
			return logFilePathErr
		}

		cmd.Printf("Added log entry to %s for %s\n", matchedProject.Name, date)
		cmd.Printf("Log: %s\n", logFilePath)
		return nil
	} else {
		if !projectGiven {
			resolvedProject, resolveErr := prompt.ResolveProject(projects, true)

			if resolveErr != nil {
				return resolveErr
			}

			matchedProject = resolvedProject
		}

		var note string

		notePromptErr := prompt.Run(huh.NewInput().
			Title("Enter your log for " + date).
			Value(&note).Validate(func(value string) error {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("note cannot be blank")
			}

			return nil
		}))

		if notePromptErr != nil {
			return fmt.Errorf("note prompt: %w", notePromptErr)
		}

		note = strings.TrimSpace(note)

		logFilePath, logFilePathErr := c.AddLogEntry(matchedProject, date, note)

		if logFilePathErr != nil {
			return logFilePathErr
		}

		cmd.Printf("Added log entry to %s for %s\n", matchedProject.Name, date)
		cmd.Printf("Log: %s\n", logFilePath)

		return nil
	}

}
