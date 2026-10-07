package task

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/cedricahenkorah/trail/internal/cli/log"
	"github.com/cedricahenkorah/trail/internal/cli/project"
	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a task to a project",
	Long: `Add an open task to a project's Markdown task file.

	Supply task text to add it without prompting. The file defaults to today's
	date; use --file to choose an existing file or name a new one. Use --due
	to set a due date in YYYY-MM-DD format.

	Omit the text to choose a file and enter a task and optional due date
	interactively. Supplied --file and --due flags skip those prompts.

	Trail uses the project linked to the current directory when there is one
	clear match. Otherwise, interactive mode asks you to choose a project.
	Use --project to select a project explicitly.`,
	RunE: runTaskAddCommand,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	addCmd.Flags().StringP("due", "d", "", "sets a due date for the task")
	addCmd.Flags().StringP("project", "p", "", "project to add the note to")
	addCmd.Flags().StringP("file", "f", "", "name of the file a task gets saved to")
}

func runTaskAddCommand(cmd *cobra.Command, args []string) error {
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

	dueDate, dueDateErr := cmd.Flags().GetString("due")

	if dueDateErr != nil {
		return fmt.Errorf("error reading the due date value: %w", dueDateErr)
	}

	dueDate = strings.TrimSpace(dueDate)

	if cmd.Flags().Changed("due") && dueDate == "" {
		return fmt.Errorf("--due requires a due date in YYYY-MM-DD format")
	}

	if cmd.Flags().Changed("due") && dueDate != "" {
		validatedDate, validatedDateErr := log.ValidateDate(dueDate)

		if validatedDateErr != nil {
			return validatedDateErr
		}

		dueDate = validatedDate
	}

	fileName, fileNameErr := cmd.Flags().GetString("file")

	if fileNameErr != nil {
		return fmt.Errorf("error reading the passed file name: %w", fileNameErr)
	}

	fileName = strings.TrimSpace(fileName)

	if cmd.Flags().Changed("file") && fileName == "" {
		return fmt.Errorf("--file requires a file name")
	}

	if cmd.Flags().Changed("file") && fileName != "" {
		normalizedFileName, normalizedFileNameErr := normalizeTaskFilename(fileName)

		if normalizedFileNameErr != nil {
			return normalizedFileNameErr
		}

		fileName = normalizedFileName
	}

	var matchedProject project.ProjectRecord

	if projectFilter != "" {
		for _, registeredProject := range projectRegistry.Projects {
			if !strings.EqualFold(registeredProject.Name, projectFilter) {
				continue
			}

			if registeredProject.Archived {
				return fmt.Errorf("project %q is archived; restore it before adding a task entry", registeredProject.Name)
			}

			matchedProject = registeredProject
			break
		}

		if matchedProject.Name == "" {
			return fmt.Errorf("project %q is not registered; run trail project list to see available projects", projectFilter)
		}
	}

	today := time.Now().Format("2006-01-02")

	if len(args) == 1 {
		task := strings.TrimSpace(args[0])

		if task == "" {
			return fmt.Errorf("task cannot be blank")
		}

		if matchedProject.Name == "" {
			matchedProjects, matchedProjectsErr := log.FindProjectsLinkedToCurrentDirectory(projectRegistry)

			if matchedProjectsErr != nil {
				return matchedProjectsErr
			}

			switch len(matchedProjects) {
			case 0:
				return fmt.Errorf("no project is linked to the current directory; pass --project <name>")
			case 1:
				if matchedProjects[0].Archived {
					return fmt.Errorf("project %q is archived; restore it before adding a task", matchedProjects[0].Name)
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

		matchedProjectDir, matchedProjectDirErr := resolveMatchedProjectDirectory(cfg.DataDirectory, matchedProject)

		if matchedProjectDirErr != nil {
			return matchedProjectDirErr
		}

		if fileName == "" {
			fileName = today + ".md"
		}

		taskFilePath, saveTaskErr := saveTaskEntry(matchedProjectDir, fileName, today, task, dueDate)

		if saveTaskErr != nil {
			return saveTaskErr
		}

		cmd.Printf("Added task to %s\n", matchedProject.Name)
		cmd.Printf("File: %s\n", taskFilePath)
		if dueDate != "" {
			cmd.Printf("Due date: %s\n", dueDate)
		}
		return nil
	} else {
		if matchedProject.Name == "" {
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

			switch len(matchedProjects) {
			case 0:
				activeProject, activeProjectErr := log.ChooseActiveProject(activeProjects)

				if activeProjectErr != nil {
					return activeProjectErr
				}

				matchedProject = activeProject

			case 1:
				if matchedProjects[0].Archived {
					activeProject, activeProjectErr := log.ChooseActiveProject(activeProjects)

					if activeProjectErr != nil {
						return activeProjectErr
					}

					matchedProject = activeProject
				} else {
					matchedProject = matchedProjects[0]
				}

			default:
				activeProject, activeProjectErr := log.ChooseActiveProject(activeProjects)

				if activeProjectErr != nil {
					return activeProjectErr
				}

				matchedProject = activeProject
			}
		}

		matchedProjectDir, matchedProjectDirErr := resolveMatchedProjectDirectory(cfg.DataDirectory, matchedProject)

		if matchedProjectDirErr != nil {
			return matchedProjectDirErr
		}

		taskDir, taskDirErr := resolveTaskDirectory(matchedProjectDir)

		if taskDirErr != nil {
			return taskDirErr
		}

		if fileName == "" {
			entries, entriesErr := os.ReadDir(taskDir)

			if entriesErr != nil && !os.IsNotExist(entriesErr) {
				return fmt.Errorf("read task directory %s: %w", taskDir, entriesErr)
			}

			var taskFiles []string

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				if !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}

				taskFiles = append(taskFiles, entry.Name())
			}

			todayFile := today + ".md"

			todayExists := slices.Contains(taskFiles, todayFile)

			todayLabel := todayFile + " (today; create new)"

			if todayExists {
				todayLabel = todayFile + " (today; append)"
			}

			options := []huh.Option[string]{
				huh.NewOption(todayLabel, todayFile),
			}

			for _, taskFile := range taskFiles {
				if taskFile == todayFile {
					continue
				}

				options = append(options, huh.NewOption(taskFile, taskFile))
			}

			const customFileChoice = "create-custom"

			options = append(options, huh.NewOption("Create a custom-named file", customFileChoice))

			chosenFileOption := todayFile

			selectFileErr := prompt.Run(huh.NewSelect[string]().
				Title("Choose a file to save the task to").
				Options(options...).
				Value(&chosenFileOption))

			if selectFileErr != nil {
				return fmt.Errorf("choose file: %w", selectFileErr)
			}

			fileName = chosenFileOption

			if chosenFileOption == customFileChoice {
				var custom string

				for {
					customFileErr := prompt.Run(huh.NewInput().
						Title("Enter the file name of your choice").
						Validate(func(custom string) error {
							custom = strings.TrimSpace(custom)

							_, normalizedFileNameErr := normalizeTaskFilename(custom)

							if normalizedFileNameErr != nil {
								return normalizedFileNameErr
							}

							return nil
						}).
						Value(&custom))

					if customFileErr != nil {
						return fmt.Errorf("read file name: %w", customFileErr)
					}

					normalizedFileName, normalizedFileNameErr := normalizeTaskFilename(strings.TrimSpace(custom))

					if normalizedFileNameErr != nil {
						return normalizedFileNameErr
					}

					customPath := filepath.Join(taskDir, normalizedFileName)

					_, checkErr := os.Lstat(customPath)

					if os.IsNotExist(checkErr) {
						fileName = normalizedFileName
						break
					}

					if checkErr != nil {
						return fmt.Errorf("check task file %s: %w", customPath, checkErr)
					}

					var collisionChoice string

					collisionErr := prompt.Run(huh.NewSelect[string]().
						Title(normalizedFileName+" already exists").
						Options(
							huh.NewOption("Append to it", "append"),
							huh.NewOption("Choose another name", "rename"),
						).
						Value(&collisionChoice))

					if collisionErr != nil {
						return fmt.Errorf("choose how to handle existing file: %w", collisionErr)
					}

					if collisionChoice == "append" {
						fileName = normalizedFileName
						break
					}
				}

			}
		}

		var task string

		inputTaskErr := prompt.Run(huh.NewInput().
			Title("Enter the task").
			Validate(func(s string) error {
				s = strings.TrimSpace(s)

				if s == "" {
					return fmt.Errorf("Task cannot be left blank")
				}

				return nil
			}).
			Value(&task))

		if inputTaskErr != nil {
			return fmt.Errorf("read task: %w", inputTaskErr)
		}

		task = strings.TrimSpace(task)

		if dueDate == "" {
			inputDueDateErr := prompt.Run(huh.NewInput().
				Title("Enter a Due date (YYYY-MM-DD, optional)").
				Validate(func(s string) error {
					s = strings.TrimSpace(s)

					if s == "" {
						return nil
					}

					_, validatedDateErr := log.ValidateDate(s)

					if validatedDateErr != nil {
						return validatedDateErr
					}

					return nil
				}).
				Value(&dueDate))

			if inputDueDateErr != nil {
				return fmt.Errorf("read due date: %w", inputDueDateErr)
			}

			dueDate = strings.TrimSpace(dueDate)
		}

		taskFilePath, saveTaskErr := saveTaskEntry(matchedProjectDir, fileName, today, task, dueDate)

		if saveTaskErr != nil {
			return saveTaskErr
		}

		cmd.Printf("Added task to %s\n", matchedProject.Name)
		cmd.Printf("File: %s\n", taskFilePath)
		if dueDate != "" {
			cmd.Printf("Due date: %s\n", dueDate)
		}
		return nil
	}
}

func resolveTaskDirectory(
	matchedProjectDir string,
) (string, error) {
	taskDir := filepath.Join(matchedProjectDir, "tasks")

	taskDirInfo, taskDirInfoErr := os.Lstat(taskDir)

	if taskDirInfoErr != nil && !os.IsNotExist(taskDirInfoErr) {
		return "", fmt.Errorf("check task dir: %w", taskDirInfoErr)
	}

	if taskDirInfoErr == nil && !taskDirInfo.IsDir() {
		return "", fmt.Errorf("project task folder is not a directory: %s", taskDir)
	}

	return taskDir, nil
}

func resolveMatchedProjectDirectory(
	dataDir string,
	matchedProject project.ProjectRecord,
) (string, error) {
	matchedProjectDir := filepath.Join(dataDir, "projects", matchedProject.Name)

	matchedProjectInfo, matchedProjectInfoErr := os.Lstat(matchedProjectDir)

	if os.IsNotExist(matchedProjectInfoErr) {
		return "", fmt.Errorf("project %q is registered but its folder is missing: %s", matchedProject.Name, matchedProjectDir)
	}

	if matchedProjectInfoErr != nil {
		return "", fmt.Errorf("check project folder %s: %w", matchedProjectDir, matchedProjectInfoErr)
	}

	if !matchedProjectInfo.IsDir() {
		return "", fmt.Errorf("project folder is not a directory: %s", matchedProjectDir)
	}

	return matchedProjectDir, nil
}

func normalizeTaskFilename(fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("filename cannot be left blank")
	}

	if strings.HasPrefix(fileName, ".") || strings.ContainsAny(fileName, `/\`) {
		return "", fmt.Errorf("filename contains invalid characters")
	}

	fileNameExt := filepath.Ext(fileName)

	if fileNameExt != "" && fileNameExt != ".md" {
		return "", fmt.Errorf("filename must have no extension or end in .md")
	}

	if fileNameExt == "" {
		fileName += ".md"
	}

	return fileName, nil
}

func saveTaskEntry(
	matchedProjectDir string,
	fileName string,
	today string,
	task string,
	dueDate string,
) (string, error) {
	taskDir, taskDirErr := resolveTaskDirectory(matchedProjectDir)

	if taskDirErr != nil {
		return "", taskDirErr
	}

	createTaskDirErr := os.MkdirAll(taskDir, 0700)

	if createTaskDirErr != nil {
		return "", fmt.Errorf("create task dir: %w", createTaskDirErr)
	}

	taskFilePath := filepath.Join(taskDir, fileName)

	taskEntry := fmt.Sprintf("- [ ] %s", task)

	if dueDate != "" {
		taskEntry += fmt.Sprintf(" @due(%s)", dueDate)
	}

	taskEntry += "\n"

	openTaskFile, openTaskFileErr := os.OpenFile(taskFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

	if os.IsExist(openTaskFileErr) {
		existingTaskFile, existingTaskFileErr := os.ReadFile(taskFilePath)

		if existingTaskFileErr != nil {
			return "", fmt.Errorf("read task file %s: %w", taskFilePath, existingTaskFileErr)
		}

		if len(existingTaskFile) > 0 && existingTaskFile[len(existingTaskFile)-1] != '\n' {
			taskEntry = "\n" + taskEntry
		}

		openTaskFile, openTaskFileErr := os.OpenFile(taskFilePath, os.O_WRONLY|os.O_APPEND, 0)

		if openTaskFileErr != nil {
			return "", fmt.Errorf("open task file %s: %w", taskFilePath, openTaskFileErr)
		}

		_, writeErr := openTaskFile.Write([]byte(taskEntry))
		closeErr := openTaskFile.Close()

		if writeErr != nil {
			return "", fmt.Errorf("append task file %s: %w", taskFilePath, writeErr)
		}

		if closeErr != nil {
			return "", fmt.Errorf("close task file %s: %w", taskFilePath, closeErr)
		}

		return taskFilePath, nil
	} else if openTaskFileErr != nil {
		return "", fmt.Errorf("create task file: %w", openTaskFileErr)
	} else {
		// create the file
		trimmedFileName := strings.TrimSuffix(fileName, ".md")

		content := fmt.Sprintf("---\ndate: %s\n---\n\n# %s\n\n%s", today, trimmedFileName, taskEntry)

		_, writeTaskErr := openTaskFile.Write([]byte(content))
		closeTaskErr := openTaskFile.Close()

		if writeTaskErr != nil {
			return "", fmt.Errorf("write task %s: %w", taskFilePath, writeTaskErr)
		}

		if closeTaskErr != nil {
			return "", fmt.Errorf("close task %s: %w", taskFilePath, closeTaskErr)
		}

		return taskFilePath, nil
	}

}
