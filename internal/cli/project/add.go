package project

import (
	"fmt"
	"os"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a project to Trail",
	Long: `Create a project in Trail's data directory. Supply a name to add one
	project directly, or omit it to enter project details interactively
	and optionally add more projects. You can link a local directory and
	assign tags.`,
	RunE: runProjectAddCommand,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	addCmd.Flags().StringSliceP("tag", "t", nil, "adds a tag to the project")
	addCmd.Flags().StringP("path", "p", "", "links the project to a directory")
}

func runProjectAddCommand(cmd *cobra.Command, args []string) error {
	cfg, _, configLoadErr := core.LoadConfig()

	if configLoadErr != nil {
		return configLoadErr
	}

	c, openErr := core.Open(cfg.DataDirectory)

	if openErr != nil {
		return openErr
	}

	// Load the registry up front so a broken one fails before any prompts.
	_, projectsErr := c.Projects()

	if projectsErr != nil {
		return projectsErr
	}

	if len(args) == 1 {
		projectPath, projectPathErr := cmd.Flags().GetString("path")

		if projectPathErr != nil {
			return fmt.Errorf("error reading the path: %w", projectPathErr)
		}

		tags, tagsErr := cmd.Flags().GetStringSlice("tag")

		if tagsErr != nil {
			return fmt.Errorf("error reading the tags: %w", tagsErr)
		}

		return addProject(cmd, c, core.NewProject{
			Name: args[0],
			Tags: tags,
			Path: projectPath,
		})
	} else {
		if cmd.Flags().Changed("path") || cmd.Flags().Changed("tag") {
			return fmt.Errorf("--path and --tag require a project name")
		}

		for {
			var name string
			var linkedDir string
			var tags []string

			pickProjectNameErr := prompt.Run(huh.NewInput().
				Title("How do you want to name this project?").
				Validate(func(value string) error {
					projectNameValidationErr := core.ValidateProjectName(value)

					if projectNameValidationErr != nil {
						return projectNameValidationErr
					}

					return c.CheckNameAvailable(strings.TrimSpace(value))
				}).
				Value(&name))

			if pickProjectNameErr != nil {
				return fmt.Errorf("read project name: %w", pickProjectNameErr)
			}

			cwd, cwdErr := os.Getwd()

			if cwdErr != nil {
				return fmt.Errorf("find current directory %w", cwdErr)
			}

			var directoryPathLinkChoice string

			linkDirectoryErr := prompt.Run(huh.NewSelect[string]().
				Title("Link a working directory?").
				Options(
					huh.NewOption(fmt.Sprintf("In this current directory (%s)", cwd), "current"),
					huh.NewOption("At another directory path", "custom"),
					huh.NewOption("Skip", "skip"),
				).
				Value(&directoryPathLinkChoice))

			if linkDirectoryErr != nil {
				return fmt.Errorf("choose linked directory: %w", linkDirectoryErr)
			}

			switch directoryPathLinkChoice {
			case "custom":
				var custom string

				customDirPathErr := prompt.Run(huh.NewInput().
					Title("Enter the directory path").
					Value(&custom))

				if customDirPathErr != nil {
					return fmt.Errorf("read linked directory path: %w", customDirPathErr)
				}

				linkedDir = strings.TrimSpace(custom)

				if linkedDir == "" {
					return fmt.Errorf("linked directory path cannot be empty")
				}

			case "current":
				linkedDir = cwd

			case "skip":
				linkedDir = ""

			default:
				return fmt.Errorf("unknown linked directory choice: %s", directoryPathLinkChoice)
			}

			var tagsInput string

			tagsErr := prompt.Run(huh.NewInput().
				Title("Tags (optional, separated by commas)").
				Placeholder("personal, work").
				Value(&tagsInput))

			if tagsErr != nil {
				return fmt.Errorf("read project tags: %w", tagsErr)
			}

			if strings.TrimSpace(tagsInput) != "" {
				tags = strings.Split(tagsInput, ",")
			}

			addProjectErr := addProject(cmd, c, core.NewProject{
				Name: name,
				Tags: tags,
				Path: linkedDir,
			})

			if addProjectErr != nil {
				return addProjectErr
			}

			var anotherProject bool

			confirmAddAnotherProjectErr := prompt.Run(huh.NewConfirm().
				Title("Add another project?").
				Value(&anotherProject))

			if confirmAddAnotherProjectErr != nil {
				return fmt.Errorf("confirm add another project: %w", confirmAddAnotherProjectErr)
			}

			if !anotherProject {
				return nil
			}
		}
	}
}

func addProject(cmd *cobra.Command, c *core.Core, newProject core.NewProject) error {
	addedProject, addErr := c.AddProject(newProject)

	if addErr != nil {
		return addErr
	}

	projectDir, projectDirErr := c.ProjectDir(addedProject)

	if projectDirErr != nil {
		return projectDirErr
	}

	linkedLine := ""
	if addedProject.LinkedPath != "" {
		linkedLine = fmt.Sprintf("Linked directory: %s\n", addedProject.LinkedPath)
	}

	tagsLine := ""
	if len(addedProject.Tags) > 0 {
		tagsLine = fmt.Sprintf("Tags: %s\n", strings.Join(addedProject.Tags, ", "))
	}

	cmd.Printf(
		"Created project %s\nData: %s\n%s%s",
		addedProject.Name,
		projectDir,
		linkedLine,
		tagsLine,
	)

	return nil
}
