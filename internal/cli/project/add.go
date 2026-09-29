/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cedricahenkorah/trail/internal/cli/config"
	"github.com/cedricahenkorah/trail/internal/cli/prompt"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

type ProjectsRegistry struct {
	Projects []ProjectRecord `json:"projects"`
}

type ProjectRecord struct {
	Name       string   `json:"name"`
	Tags       []string `json:"tags"`
	LinkedPath string   `json:"linked_path"`
	Archived   bool     `json:"archived"`
}

type ProjectDetails struct {
	Name string
	Tags []string
	Path string
}

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
	var projectsRegistry ProjectsRegistry

	cfg, _, configLoadErr := config.Load()

	if configLoadErr != nil {
		return configLoadErr
	}

	dataDirErr := config.ValidateDataDirectory(cfg.DataDirectory)

	if dataDirErr != nil {
		return dataDirErr
	}

	projectJsonPath := filepath.Join(cfg.DataDirectory, "projects.json")

	projectJsonData, projectJsonDataReadErr := os.ReadFile(projectJsonPath)

	if os.IsNotExist(projectJsonDataReadErr) {
		projectsDir := filepath.Join(cfg.DataDirectory, "projects")

		projectDirEntries, projectDirEntriesErr := os.ReadDir(projectsDir)

		if projectDirEntriesErr != nil && !os.IsNotExist(projectDirEntriesErr) {
			return fmt.Errorf("read projects directory: %w", projectDirEntriesErr)
		}

		// todo: add a helpful command in the error msg so a user can repair the drift / offer to repair in this path
		if len(projectDirEntries) > 0 {
			return fmt.Errorf("project registry is missing but projects directory is not empty")
		}

		projectsRegistry.Projects = []ProjectRecord{}
	} else if projectJsonDataReadErr != nil {
		return fmt.Errorf("read project registry: %w", projectJsonDataReadErr)
	} else {
		parseErr := json.Unmarshal(projectJsonData, &projectsRegistry)

		if parseErr != nil {
			return fmt.Errorf("parse projects json at %s: %w", projectJsonPath, parseErr)
		}
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

		projectDetails := ProjectDetails{
			Name: args[0],
			Tags: tags,
			Path: projectPath,
		}

		return addAProject(cmd, &projectDetails, cfg.DataDirectory, projectJsonPath, &projectsRegistry)
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
					projectNameValidationErr := validateProjectName(value)

					if projectNameValidationErr != nil {
						return projectNameValidationErr
					}

					return checkProjectNameAvailable(
						strings.TrimSpace(value),
						cfg.DataDirectory,
						&projectsRegistry,
					)
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

			projectDetails := ProjectDetails{
				Name: name,
				Tags: tags,
				Path: linkedDir,
			}

			addProjectErr := addAProject(cmd, &projectDetails, cfg.DataDirectory, projectJsonPath, &projectsRegistry)

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

func validateProjectDetails(details *ProjectDetails, dataDir string, registry *ProjectsRegistry) error {
	var name string
	var path string
	var tags []string

	nameValidationErr := validateProjectName(details.Name)

	if nameValidationErr != nil {
		return nameValidationErr
	}

	name = strings.TrimSpace(details.Name)

	projectNameAvailableErr := checkProjectNameAvailable(name, dataDir, registry)

	if projectNameAvailableErr != nil {
		return projectNameAvailableErr
	}

	path = strings.TrimSpace(details.Path)

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, homeDirErr := os.UserHomeDir()

		if homeDirErr != nil {
			return fmt.Errorf("find home directory: %w", homeDirErr)
		}

		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	if path != "" {
		absolutePath, absErr := filepath.Abs(path)

		if absErr != nil {
			return fmt.Errorf("resolve linked directory: %w", absErr)
		}

		absPathInfo, absPathInfoErr := os.Stat(absolutePath)

		if absPathInfoErr != nil {
			return fmt.Errorf("resolve linked directory: %w", absPathInfoErr)
		}

		if !absPathInfo.IsDir() {
			return fmt.Errorf("linked path is not a directory: %s", absolutePath)
		}

		path = absolutePath
	}

	seen := make(map[string]bool)

	for _, tag := range details.Tags {
		tag = strings.TrimSpace(tag)

		if tag == "" {
			return fmt.Errorf("tag cannot be blank")
		}

		key := strings.ToLower(tag)

		if seen[key] {
			continue
		}

		seen[key] = true

		tags = append(tags, tag)
	}

	details.Name = name
	details.Tags = tags
	details.Path = path

	return nil
}

func validateProjectName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("Project name cannot be blank")
	}

	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("project name cannot contain path separators")
	}

	if strings.Contains(name, ".") {
		return fmt.Errorf("project name cannot contain dots")
	}

	return nil
}

func checkProjectNameAvailable(name, dataDir string, registry *ProjectsRegistry) error {
	projectDir := filepath.Join(dataDir, "projects", name)

	_, projectDirInfoErr := os.Stat(projectDir)

	if projectDirInfoErr == nil {
		return fmt.Errorf("a project with this name already exists: %s", name)
	}

	if !os.IsNotExist(projectDirInfoErr) {
		return fmt.Errorf("check project directory: %w", projectDirInfoErr)
	}

	for _, existingProject := range registry.Projects {
		if strings.EqualFold(existingProject.Name, name) {
			return fmt.Errorf("a project with this name is already registered: %s", existingProject.Name)
		}
	}

	return nil
}

func cleanUpProject(projectsMdPath, newProjectDir string, cause error) error {
	if err := os.Remove(projectsMdPath); err != nil {
		return fmt.Errorf("%w; could not remove %s: %v", cause, projectsMdPath, err)
	}
	if err := os.Remove(newProjectDir); err != nil {
		return fmt.Errorf("%w; could not remove %s: %v", cause, newProjectDir, err)
	}
	return cause
}

func addAProject(cmd *cobra.Command, projectDetails *ProjectDetails, dataDir, projectJsonPath string, projectsRegistry *ProjectsRegistry) error {
	projectDetailsValidationErr := validateProjectDetails(projectDetails, dataDir, projectsRegistry)

	if projectDetailsValidationErr != nil {
		return projectDetailsValidationErr
	}

	projectsDir := filepath.Join(dataDir, "projects")

	createProjectsDirErr := os.MkdirAll(projectsDir, 0700)

	if createProjectsDirErr != nil {
		return fmt.Errorf("create project directory: %w", createProjectsDirErr)
	}

	newProjectDir := filepath.Join(projectsDir, projectDetails.Name)

	newProjectDirErr := os.Mkdir(newProjectDir, 0700)

	if newProjectDirErr != nil {
		return fmt.Errorf("create project directory: %w", newProjectDirErr)
	}

	projectsMdPath := filepath.Join(newProjectDir, "project.md")

	projectMdContent := []byte("# " + projectDetails.Name + "\n")

	projectMd, openProjectMdErr := os.OpenFile(projectsMdPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)

	if openProjectMdErr != nil {
		removeNewProjectDirErr := os.Remove(newProjectDir)

		if removeNewProjectDirErr != nil {
			return fmt.Errorf(
				"create project md file: %w; could not remove project directory %s: %v",
				openProjectMdErr, newProjectDir, removeNewProjectDirErr,
			)
		}

		return fmt.Errorf("create project md file: %w", openProjectMdErr)
	}

	_, writeProjectMdErr := projectMd.Write(projectMdContent)
	closeProjectMdErr := projectMd.Close()

	if writeProjectMdErr != nil {
		removeProjectMdErr := os.Remove(projectsMdPath)

		if removeProjectMdErr != nil {
			return fmt.Errorf(
				"write projects md file: %w; could not remove partial md file at %s: %v",
				writeProjectMdErr, projectsMdPath, removeProjectMdErr,
			)
		}

		removeNewProjectDirErr := os.Remove(newProjectDir)

		if removeNewProjectDirErr != nil {
			return fmt.Errorf(
				"write project md file: %w; could not remove project directory %s: %v",
				writeProjectMdErr, newProjectDir, removeNewProjectDirErr,
			)
		}

		return fmt.Errorf("write project md file: %w", writeProjectMdErr)
	}

	if closeProjectMdErr != nil {
		removeProjectMdErr := os.Remove(projectsMdPath)

		if removeProjectMdErr != nil {
			return fmt.Errorf(
				"close project md file: %w; could not remove partial project md at %s: %v",
				closeProjectMdErr, projectsMdPath, removeProjectMdErr,
			)
		}

		removeNewProjectDirErr := os.Remove(newProjectDir)

		if removeNewProjectDirErr != nil {
			return fmt.Errorf(
				"close project md file: %w; could not remove project directory %s: %v",
				closeProjectMdErr, newProjectDir, removeNewProjectDirErr,
			)
		}

		return fmt.Errorf("close project md file: %w", closeProjectMdErr)
	}

	projectRecord := ProjectRecord{
		Name:       projectDetails.Name,
		Tags:       projectDetails.Tags,
		LinkedPath: projectDetails.Path,
		Archived:   false,
	}

	projectsRegistry.Projects = append(projectsRegistry.Projects, projectRecord)

	projectRegistrycontents, projectRegistryIndentErr := json.MarshalIndent(projectsRegistry, "", "  ")

	if projectRegistryIndentErr != nil {
		return cleanUpProject(
			projectsMdPath,
			newProjectDir,
			fmt.Errorf("encode project registry: %w", projectRegistryIndentErr),
		)
	}

	projectRegistrycontents = append(projectRegistrycontents, '\n')

	tempFile, createTempFileErr := os.CreateTemp(filepath.Dir(projectJsonPath), "projects-*.tmp")

	if createTempFileErr != nil {
		return cleanUpProject(
			projectsMdPath,
			newProjectDir,
			fmt.Errorf("create temporary project json file: %w", createTempFileErr),
		)
	}

	tempPath := tempFile.Name()

	defer os.Remove(tempPath)

	_, tempFileWriteErr := tempFile.Write(projectRegistrycontents)

	if tempFileWriteErr != nil {
		tempFile.Close()

		return cleanUpProject(
			projectsMdPath,
			newProjectDir,
			fmt.Errorf("write temporary project json file: %w", tempFileWriteErr))
	}

	tempFileCloseErr := tempFile.Close()

	if tempFileCloseErr != nil {
		return cleanUpProject(
			projectsMdPath,
			newProjectDir,
			fmt.Errorf("close temporary project json file: %w", tempFileCloseErr))
	}

	renameTempFile := os.Rename(tempPath, projectJsonPath)

	if renameTempFile != nil {
		return cleanUpProject(
			projectsMdPath,
			newProjectDir,
			fmt.Errorf("replace project json file: %w", renameTempFile))
	}

	linkedLine := ""
	if projectDetails.Path != "" {
		linkedLine = fmt.Sprintf("Linked directory: %s\n", projectDetails.Path)
	}

	tagsLine := ""
	if len(projectDetails.Tags) > 0 {
		tagsLine = fmt.Sprintf("Tags: %s\n", strings.Join(projectDetails.Tags, ", "))
	}

	cmd.Printf(
		"Created project %s\nData: %s\n%s%s",
		projectDetails.Name,
		newProjectDir,
		linkedLine,
		tagsLine,
	)

	return nil
}
