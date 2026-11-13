package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
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

	projectFilter, projectFilterErr := cmd.Flags().GetString("project")

	if projectFilterErr != nil {
		return fmt.Errorf("error reading the project filter: %w", projectFilterErr)
	}

	projectFilter = strings.TrimSpace(projectFilter)

	if cmd.Flags().Changed("project") && projectFilter == "" {
		return fmt.Errorf("--project requires a project name")
	}

	dateFilter, dateFilterErr := cmd.Flags().GetString("date")

	if dateFilterErr != nil {
		return fmt.Errorf("error reading the date filter: %w", dateFilterErr)
	}

	dateFilter = strings.TrimSpace(dateFilter)

	if cmd.Flags().Changed("date") && dateFilter == "" {
		return fmt.Errorf("--date requires a date in YYYY-MM-DD format")
	}

	var matchedProject core.Project

	if projectFilter != "" {
		foundProject, findErr := c.Project(projectFilter)

		if findErr != nil {
			return findErr
		}

		if foundProject.Archived {
			return fmt.Errorf("project %q is archived; restore it before adding a log entry", foundProject.Name)
		}

		matchedProject = foundProject
	}

	date := time.Now().Format("2006-01-02")

	if dateFilter != "" {
		validatedDate, validatedDateErr := ValidateDate(dateFilter)

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

		if matchedProject.Name == "" {
			resolvedProject, resolveErr := prompt.ResolveProject(projects, false)

			if resolveErr != nil {
				return resolveErr
			}

			matchedProject = resolvedProject
		}

		logFilePath, logFilePathErr := saveLogEntry(c.DataDir(), matchedProject, date, note)

		if logFilePathErr != nil {
			return logFilePathErr
		}

		cmd.Printf("Added log entry to %s for %s\n", matchedProject.Name, date)
		cmd.Printf("Log: %s\n", logFilePath)
		return nil
	} else {
		if matchedProject.Name == "" {
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

		logFilePath, logFilePathErr := saveLogEntry(c.DataDir(), matchedProject, date, note)

		if logFilePathErr != nil {
			return logFilePathErr
		}

		cmd.Printf("Added log entry to %s for %s\n", matchedProject.Name, date)
		cmd.Printf("Log: %s\n", logFilePath)

		return nil
	}

}

func ValidateDate(value string) (string, error) {
	parsedDate, parsedDateErr := time.Parse("2006-01-02", value)

	if parsedDateErr != nil {
		return "", fmt.Errorf("invalid date %q: use YYYY-MM-DD", value)
	}

	value = parsedDate.Format("2006-01-02")

	return value, nil
}

func saveLogEntry(
	dataDir string,
	selectedProject core.Project,
	date string,
	note string,
) (string, error) {
	matchedProjectDir := filepath.Join(dataDir, "projects", selectedProject.Name)

	matchedProjectInfo, matchedProjectInfoErr := os.Stat(matchedProjectDir)

	if os.IsNotExist(matchedProjectInfoErr) {
		return "", fmt.Errorf("project %q is registered but its folder is missing: %s", selectedProject.Name, matchedProjectDir)
	}

	if matchedProjectInfoErr != nil {
		return "", fmt.Errorf("check project folder %s: %w", matchedProjectDir, matchedProjectInfoErr)
	}

	if !matchedProjectInfo.IsDir() {
		return "", fmt.Errorf("project folder is not a directory: %s", matchedProjectDir)
	}

	logDir := filepath.Join(matchedProjectDir, "log")

	logDirInfo, logDirInfoErr := os.Stat(logDir)

	if logDirInfoErr != nil && !os.IsNotExist(logDirInfoErr) {
		return "", fmt.Errorf("check log dir: %w", logDirInfoErr)
	}

	if os.IsNotExist(logDirInfoErr) {
		createLogDirErr := os.MkdirAll(logDir, 0700)

		if createLogDirErr != nil {
			return "", fmt.Errorf("create log dir: %w", createLogDirErr)
		}
	}

	if logDirInfoErr == nil && !logDirInfo.IsDir() {
		return "", fmt.Errorf("project log folder is not a directory: %s", logDir)
	}

	logFilePath := filepath.Join(logDir, date+".md")

	logFile, logFileErr := os.OpenFile(logFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

	if os.IsExist(logFileErr) {
		// append the contents
		existingLog, existingLogErr := os.ReadFile(logFilePath)

		if existingLogErr != nil {
			return "", fmt.Errorf("read daily log %s: %w", logFilePath, existingLogErr)
		}

		entry := fmt.Sprintf("- %s\n", note)

		if len(existingLog) > 0 && existingLog[len(existingLog)-1] != '\n' {
			entry = "\n" + entry
		}

		logFile, logFileErr := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND, 0)

		if logFileErr != nil {
			return "", fmt.Errorf("open daily log %s: %w", logFilePath, logFileErr)
		}

		_, writeErr := logFile.Write([]byte(entry))
		closeErr := logFile.Close()

		if writeErr != nil {
			return "", fmt.Errorf("append daily log %s: %w", logFilePath, writeErr)
		}

		if closeErr != nil {
			return "", fmt.Errorf("close daily log %s: %w", logFilePath, closeErr)
		}

	} else if logFileErr != nil {
		return "", fmt.Errorf("create daily log %s: %w", logFilePath, logFileErr)
	} else {
		// write to the new file
		logContent := fmt.Sprintf("# %s\n\n- %s\n", date, note)

		_, writeLogErr := logFile.Write([]byte(logContent))
		closeLogErr := logFile.Close()

		if writeLogErr != nil {
			return "", fmt.Errorf("write daily log %s: %w", logFilePath, writeLogErr)
		}

		if closeLogErr != nil {
			return "", fmt.Errorf("close daily log %s: %w", logFilePath, closeLogErr)
		}
	}

	return logFilePath, nil
}
