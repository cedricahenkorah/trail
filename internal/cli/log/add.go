package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/cedricahenkorah/trail/internal/cli/project"
	"github.com/cedricahenkorah/trail/internal/cli/prompt"
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

	dateFilter, dateFilterErr := cmd.Flags().GetString("date")

	if dateFilterErr != nil {
		return fmt.Errorf("error reading the date filter: %w", dateFilterErr)
	}

	dateFilter = strings.TrimSpace(dateFilter)

	if cmd.Flags().Changed("date") && dateFilter == "" {
		return fmt.Errorf("--date requires a date in YYYY-MM-DD format")
	}

	var matchedProject project.ProjectRecord

	if projectFilter != "" {
		for _, registeredProject := range projectRegistry.Projects {
			if !strings.EqualFold(registeredProject.Name, projectFilter) {
				continue
			}

			if registeredProject.Archived {
				return fmt.Errorf("project %q is archived; restore it before adding a log entry", registeredProject.Name)
			}

			matchedProject = registeredProject
			break
		}

		if matchedProject.Name == "" {
			return fmt.Errorf("project %q is not registered; run trail project list to see available projects", projectFilter)
		}
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
			matchedProjects, matchedProjectsErr := FindProjectsLinkedToCurrentDirectory(projectRegistry)

			if matchedProjectsErr != nil {
				return matchedProjectsErr
			}

			switch len(matchedProjects) {
			case 0:
				return fmt.Errorf("no project is linked to the current directory; pass --project <name>")
			case 1:
				if matchedProjects[0].Archived {
					return fmt.Errorf("project %q is archived; restore it before adding a log entry", matchedProjects[0].Name)
				}

				matchedProject = matchedProjects[0]
			default:
				names := make([]string, 0, len(matchedProjects))

				for _, matched := range matchedProjects {
					names = append(names, matched.Name)
				}

				return fmt.Errorf(
					"multiple projects are linked to the current directory: %s; pass --project <name>",
					strings.Join(names, ", "),
				)
			}
		}

		logFilePath, logFilePathErr := saveLogEntry(cfg.DataDirectory, matchedProject, date, note)

		if logFilePathErr != nil {
			return logFilePathErr
		}

		cmd.Printf("Added log entry to %s for %s\n", matchedProject.Name, date)
		cmd.Printf("Log: %s\n", logFilePath)
		return nil
	} else {
		if matchedProject.Name == "" {
			matchedProjects, matchedProjectsErr := FindProjectsLinkedToCurrentDirectory(projectRegistry)

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

			switch len(matchedProjects) {
			case 0:
				activeProject, activeProjectErr := ChooseActiveProject(activeProjects)

				if activeProjectErr != nil {
					return activeProjectErr
				}

				matchedProject = activeProject

			case 1:
				if matchedProjects[0].Archived {
					activeProject, activeProjectErr := ChooseActiveProject(activeProjects)

					if activeProjectErr != nil {
						return activeProjectErr
					}

					matchedProject = activeProject
				} else {
					matchedProject = matchedProjects[0]
				}

			default:
				activeProject, activeProjectErr := ChooseActiveProject(activeProjects)

				if activeProjectErr != nil {
					return activeProjectErr
				}

				matchedProject = activeProject
			}
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

		logFilePath, logFilePathErr := saveLogEntry(cfg.DataDirectory, matchedProject, date, note)

		if logFilePathErr != nil {
			return logFilePathErr
		}

		cmd.Printf("Added log entry to %s for %s\n", matchedProject.Name, date)
		cmd.Printf("Log: %s\n", logFilePath)

		return nil
	}

}

func FindProjectsLinkedToCurrentDirectory(
	registry project.ProjectsRegistry,
) ([]project.ProjectRecord, error) {
	var matchedProjects []project.ProjectRecord

	cwd, cwdErr := os.Getwd()

	if cwdErr != nil {
		return nil, fmt.Errorf("find current directory %w", cwdErr)
	}

	for _, registeredProject := range registry.Projects {
		if registeredProject.LinkedPath == "" {
			continue
		}

		rel, relErr := filepath.Rel(registeredProject.LinkedPath, cwd)

		if relErr != nil {
			return nil, fmt.Errorf("compare current directory with project %q: %w",
				registeredProject.Name, relErr)
		}

		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			matchedProjects = append(matchedProjects, registeredProject)
		}
	}

	return matchedProjects, nil
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
	selectedProject project.ProjectRecord,
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

func ChooseActiveProject(activeProjects []project.ProjectRecord) (project.ProjectRecord, error) {
	var chosenProjectName string
	var options []huh.Option[string]

	for _, p := range activeProjects {
		options = append(options, huh.NewOption(p.Name, p.Name))
	}

	selectProjectErr := prompt.Run(huh.NewSelect[string]().
		Title("Select a project").
		Options(options...).
		Value(&chosenProjectName))

	if selectProjectErr != nil {
		return project.ProjectRecord{}, fmt.Errorf("choose project: %w", selectProjectErr)
	}

	for _, p := range activeProjects {
		if p.Name == chosenProjectName {
			return p, nil
		}
	}

	return project.ProjectRecord{}, fmt.Errorf("selected project %q was not found", chosenProjectName)
}
