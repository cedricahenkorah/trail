package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
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

	dueDate, dueDateErr := cmd.Flags().GetString("due")

	if dueDateErr != nil {
		return fmt.Errorf("error reading the due date value: %w", dueDateErr)
	}

	dueDate = strings.TrimSpace(dueDate)

	if cmd.Flags().Changed("due") && dueDate == "" {
		return fmt.Errorf("--due requires a due date in YYYY-MM-DD format")
	}

	if cmd.Flags().Changed("due") && dueDate != "" {
		validatedDate, validatedDateErr := core.ValidateDate(dueDate)

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
		normalizedFileName, normalizedFileNameErr := core.NormalizeFilename(fileName)

		if normalizedFileNameErr != nil {
			return normalizedFileNameErr
		}

		fileName = normalizedFileName
	}

	var matchedProject core.Project

	if projectFilter != "" {
		foundProject, findErr := c.Project(projectFilter)

		if findErr != nil {
			return findErr
		}

		if foundProject.Archived {
			return fmt.Errorf("project %q is archived; restore it before adding a task entry", foundProject.Name)
		}

		matchedProject = foundProject
	}

	today := time.Now().Format("2006-01-02")

	if len(args) == 1 {
		task := strings.TrimSpace(args[0])

		if task == "" {
			return fmt.Errorf("task cannot be blank")
		}

		if matchedProject.Name == "" {
			resolvedProject, resolveErr := prompt.ResolveProject(projects, false)

			if resolveErr != nil {
				return resolveErr
			}

			matchedProject = resolvedProject
		}

		if fileName == "" {
			fileName = today + ".md"
		}

		taskFilePath, saveTaskErr := c.AddItem(matchedProject, core.Tasks, fileName, today, core.WithDue(task, dueDate))

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
			resolvedProject, resolveErr := prompt.ResolveProject(projects, true)

			if resolveErr != nil {
				return resolveErr
			}

			matchedProject = resolvedProject
		}

		if fileName == "" {
			taskFiles, taskFilesErr := c.Files(matchedProject, core.Tasks)

			if taskFilesErr != nil {
				return taskFilesErr
			}

			chosenFile, chooseFileErr := prompt.ChooseFile(taskFiles, today)

			if chooseFileErr != nil {
				return chooseFileErr
			}

			fileName = chosenFile
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

					_, validatedDateErr := core.ValidateDate(s)

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

		taskFilePath, saveTaskErr := c.AddItem(matchedProject, core.Tasks, fileName, today, core.WithDue(task, dueDate))

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
