package issue

import (
	"fmt"
	"strings"
	"time"

	"github.com/cedricahenkorah/trail/internal/cli/checklist"
	"github.com/cedricahenkorah/trail/internal/cli/log"
	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add an issue to a project",
	Long: `Add an unresolved issue to a project's Markdown issue file.

	Supply issue text to add it without prompting. The file defaults to today's
	date; use --file to choose an existing file or name a new one.

	Omit the text to choose a file and enter the issue interactively. A
	supplied --file flag skips the file prompt.

	Trail uses the project linked to the current directory when there is one
	clear match. Otherwise, interactive mode asks you to choose a project.
	Use --project to select a project explicitly.`,
	RunE: runIssueAddCommand,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	addCmd.Flags().StringP("project", "p", "", "project to add the issue to")
	addCmd.Flags().StringP("file", "f", "", "name of the file the issue gets saved to")
}

func runIssueAddCommand(cmd *cobra.Command, args []string) error {
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

	fileName, fileNameErr := cmd.Flags().GetString("file")

	if fileNameErr != nil {
		return fmt.Errorf("error reading the passed file name: %w", fileNameErr)
	}

	fileName = strings.TrimSpace(fileName)

	if cmd.Flags().Changed("file") {
		normalizedFileName, normalizeErr := checklist.NormalizeFilename(fileName)

		if normalizeErr != nil {
			if fileName == "" {
				return fmt.Errorf("--file requires a file name")
			}

			return normalizeErr
		}

		fileName = normalizedFileName
	}

	var issueText string
	interactive := len(args) == 0

	if !interactive {
		issueText = strings.TrimSpace(args[0])

		if issueText == "" {
			return fmt.Errorf("issue cannot be blank")
		}

		if strings.ContainsAny(issueText, "\r\n") {
			return fmt.Errorf("issue must be a single line")
		}
	}

	var matchedProject core.Project

	if projectFilter != "" {
		foundProject, findErr := c.Project(projectFilter)

		if findErr != nil {
			return findErr
		}

		if foundProject.Archived {
			return fmt.Errorf("project %q is archived; restore it before adding an issue", foundProject.Name)
		}

		matchedProject = foundProject
	}

	if matchedProject.Name == "" {
		resolvedProject, resolveErr := log.ResolveProject(projects, interactive)

		if resolveErr != nil {
			return resolveErr
		}

		matchedProject = resolvedProject
	}

	today := time.Now().Format("2006-01-02")

	projectDir, projectDirErr := c.ProjectDir(matchedProject)

	if projectDirErr != nil {
		return projectDirErr
	}

	issueDir, issueDirErr := checklist.ResolveDir(projectDir, checklist.Issues)

	if issueDirErr != nil {
		return issueDirErr
	}

	if fileName == "" {
		if interactive {
			chosenFile, chooseFileErr := checklist.ChooseFile(issueDir, checklist.Issues, today)

			if chooseFileErr != nil {
				return chooseFileErr
			}

			fileName = chosenFile
		} else {
			fileName = today + ".md"
		}
	}

	if interactive {
		inputErr := prompt.Run(huh.NewInput().
			Title("Enter the issue").
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("issue cannot be left blank")
				}

				return nil
			}).
			Value(&issueText))

		if inputErr != nil {
			return fmt.Errorf("read issue: %w", inputErr)
		}

		issueText = strings.TrimSpace(issueText)
	}

	issueFilePath, saveErr := checklist.AddItem(projectDir, checklist.Issues, fileName, today, issueText)

	if saveErr != nil {
		return saveErr
	}

	cmd.Printf("Added issue to %s\n", matchedProject.Name)
	cmd.Printf("File: %s\n", issueFilePath)
	return nil
}
