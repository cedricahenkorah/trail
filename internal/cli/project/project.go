package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects tracked by Trail",
	Long: `Create and manage the projects you track with Trail. Project notes,
	tasks, and issues are stored in Trail's data directory. A project can
	optionally link to a local working directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("project called")
	},
}

func NewCommand() *cobra.Command {
	projectCmd.AddCommand(addCmd)
	projectCmd.AddCommand(listCmd)
	return projectCmd
}

func LoadProjectRegistry(dataDir string) (ProjectsRegistry, error) {
	var projectsRegistry ProjectsRegistry

	projectJsonPath := filepath.Join(dataDir, "projects.json")

	projectJsonData, projectJsonDataReadErr := os.ReadFile(projectJsonPath)

	if os.IsNotExist(projectJsonDataReadErr) {
		projectsDir := filepath.Join(dataDir, "projects")

		projectDirEntries, projectDirEntriesErr := os.ReadDir(projectsDir)

		if projectDirEntriesErr != nil && !os.IsNotExist(projectDirEntriesErr) {
			return ProjectsRegistry{}, fmt.Errorf("read projects directory: %w", projectDirEntriesErr)
		}

		if len(projectDirEntries) > 0 {
			return ProjectsRegistry{}, fmt.Errorf("project registry is missing but projects directory is not empty")
		}

		projectsRegistry.Projects = []ProjectRecord{}
	} else if projectJsonDataReadErr != nil {
		return ProjectsRegistry{}, fmt.Errorf("read project registry at %s: %w", projectJsonPath, projectJsonDataReadErr)
	} else {
		parseErr := json.Unmarshal(projectJsonData, &projectsRegistry)

		if parseErr != nil {
			return ProjectsRegistry{}, fmt.Errorf("parse projects json at %s: %w", projectJsonPath, parseErr)
		}
	}

	return projectsRegistry, nil
}
