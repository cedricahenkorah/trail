package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Project is a project entry in the registry.
type Project struct {
	Name       string   `json:"name"`
	Tags       []string `json:"tags"`
	LinkedPath string   `json:"linked_path"`
	Archived   bool     `json:"archived"`
}

// NewProject holds the details for [Core.AddProject].
type NewProject struct {
	Name string
	Tags []string
	Path string
}

// registry is the contents of projects.json.
type registry struct {
	Projects []Project `json:"projects"`
}

func (c *Core) registryPath() string {
	return filepath.Join(c.dataDir, "projects.json")
}

func (c *Core) projectsDir() string {
	return filepath.Join(c.dataDir, "projects")
}

func (c *Core) loadRegistry() (registry, error) {
	var projectsRegistry registry

	registryPath := c.registryPath()

	registryData, readErr := os.ReadFile(registryPath)

	if os.IsNotExist(readErr) {
		projectDirEntries, projectDirEntriesErr := os.ReadDir(c.projectsDir())

		if projectDirEntriesErr != nil && !os.IsNotExist(projectDirEntriesErr) {
			return registry{}, fmt.Errorf("read projects directory: %w", projectDirEntriesErr)
		}

		// todo: add a helpful command in the error msg so a user can repair the drift / offer to repair in this path
		if len(projectDirEntries) > 0 {
			return registry{}, fmt.Errorf("project registry is missing but projects directory is not empty")
		}

		projectsRegistry.Projects = []Project{}

		return projectsRegistry, nil
	}

	if readErr != nil {
		return registry{}, fmt.Errorf("read project registry at %s: %w", registryPath, readErr)
	}

	parseErr := json.Unmarshal(registryData, &projectsRegistry)

	if parseErr != nil {
		return registry{}, fmt.Errorf("parse projects json at %s: %w", registryPath, parseErr)
	}

	return projectsRegistry, nil
}

// Projects returns every registered project, including archived ones.
func (c *Core) Projects() ([]Project, error) {
	projectsRegistry, loadErr := c.loadRegistry()

	return projectsRegistry.Projects, loadErr
}

// Project returns the project named name, ignoring case, or [ErrProjectNotFound].
func (c *Core) Project(name string) (Project, error) {
	projects, projectsErr := c.Projects()

	if projectsErr != nil {
		return Project{}, projectsErr
	}

	name = strings.TrimSpace(name)

	for _, registeredProject := range projects {
		if strings.EqualFold(registeredProject.Name, name) {
			return registeredProject, nil
		}
	}

	return Project{}, fmt.Errorf("%w: %s", ErrProjectNotFound, name)
}

// ProjectDir returns the folder holding p's files, or an error if it is missing.
func (c *Core) ProjectDir(p Project) (string, error) {
	dir := filepath.Join(c.projectsDir(), p.Name)

	exists, dirErr := dirExists(dir, "project folder")

	if dirErr != nil {
		return "", dirErr
	}

	if !exists {
		return "", fmt.Errorf("project %q is registered but its folder is missing: %s", p.Name, dir)
	}

	return dir, nil
}

// ActiveProjects returns the projects that are not archived.
func ActiveProjects(projects []Project) []Project {
	var activeProjects []Project

	for _, registeredProject := range projects {
		if !registeredProject.Archived {
			activeProjects = append(activeProjects, registeredProject)
		}
	}

	return activeProjects
}

// FilterProjects drops archived projects unless includeArchived is set, and
// keeps only projects tagged tag, ignoring case, when tag is not empty.
func FilterProjects(projects []Project, includeArchived bool, tag string) []Project {
	var selectedProjects []Project

	for _, registeredProject := range projects {
		if registeredProject.Archived && !includeArchived {
			continue
		}

		if tag != "" && !hasTag(registeredProject, tag) {
			continue
		}

		selectedProjects = append(selectedProjects, registeredProject)
	}

	return selectedProjects
}

func hasTag(p Project, tag string) bool {
	for _, projectTag := range p.Tags {
		if strings.EqualFold(projectTag, tag) {
			return true
		}
	}

	return false
}

// ValidateProjectName returns an error if name is blank or contains path separators or dots.
func ValidateProjectName(name string) error {
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

// CheckNameAvailable returns an error if a project named name already exists.
func (c *Core) CheckNameAvailable(name string) error {
	projectsRegistry, loadErr := c.loadRegistry()

	if loadErr != nil {
		return loadErr
	}

	return c.checkNameAvailable(name, projectsRegistry)
}

func (c *Core) checkNameAvailable(name string, projectsRegistry registry) error {
	projectDir := filepath.Join(c.projectsDir(), name)

	_, projectDirInfoErr := os.Stat(projectDir)

	if projectDirInfoErr == nil {
		return fmt.Errorf("a project with this name already exists: %s", name)
	}

	if !os.IsNotExist(projectDirInfoErr) {
		return fmt.Errorf("check project directory: %w", projectDirInfoErr)
	}

	for _, existingProject := range projectsRegistry.Projects {
		if strings.EqualFold(existingProject.Name, name) {
			return fmt.Errorf("a project with this name is already registered: %s", existingProject.Name)
		}
	}

	return nil
}

// validateNewProject checks newProject and returns it as a Project with a trimmed
// name, an absolute linked path, and de-duplicated tags.
func (c *Core) validateNewProject(newProject NewProject, projectsRegistry registry) (Project, error) {
	nameValidationErr := ValidateProjectName(newProject.Name)

	if nameValidationErr != nil {
		return Project{}, nameValidationErr
	}

	name := strings.TrimSpace(newProject.Name)

	nameAvailableErr := c.checkNameAvailable(name, projectsRegistry)

	if nameAvailableErr != nil {
		return Project{}, nameAvailableErr
	}

	path := strings.TrimSpace(newProject.Path)

	if path != "" {
		absolutePath, expandErr := ExpandPath(path)

		if expandErr != nil {
			return Project{}, fmt.Errorf("resolve linked directory: %w", expandErr)
		}

		absPathInfo, absPathInfoErr := os.Stat(absolutePath)

		if absPathInfoErr != nil {
			return Project{}, fmt.Errorf("resolve linked directory: %w", absPathInfoErr)
		}

		if !absPathInfo.IsDir() {
			return Project{}, fmt.Errorf("linked path is not a directory: %s", absolutePath)
		}

		path = absolutePath
	}

	var tags []string

	seen := make(map[string]bool)

	for _, tag := range newProject.Tags {
		tag = strings.TrimSpace(tag)

		if tag == "" {
			return Project{}, fmt.Errorf("tag cannot be blank")
		}

		key := strings.ToLower(tag)

		if seen[key] {
			continue
		}

		seen[key] = true

		tags = append(tags, tag)
	}

	return Project{Name: name, Tags: tags, LinkedPath: path}, nil
}

// AddProject validates newProject, creates the project's folder and project.md,
// and adds it to the registry.
func (c *Core) AddProject(newProject NewProject) (Project, error) {
	projectsRegistry, loadErr := c.loadRegistry()

	if loadErr != nil {
		return Project{}, loadErr
	}

	project, validateErr := c.validateNewProject(newProject, projectsRegistry)

	if validateErr != nil {
		return Project{}, validateErr
	}

	createProjectsDirErr := os.MkdirAll(c.projectsDir(), 0700)

	if createProjectsDirErr != nil {
		return Project{}, fmt.Errorf("create project directory: %w", createProjectsDirErr)
	}

	newProjectDir := filepath.Join(c.projectsDir(), project.Name)

	newProjectDirErr := os.Mkdir(newProjectDir, 0700)

	if newProjectDirErr != nil {
		return Project{}, fmt.Errorf("create project directory: %w", newProjectDirErr)
	}

	projectMdPath := filepath.Join(newProjectDir, "project.md")

	writeProjectMdErr := os.WriteFile(projectMdPath, []byte("# "+project.Name+"\n"), 0600)

	if writeProjectMdErr != nil {
		return Project{}, removeNewProject(newProjectDir, fmt.Errorf("write project md file: %w", writeProjectMdErr))
	}

	projectsRegistry.Projects = append(projectsRegistry.Projects, project)

	registryContents, encodeErr := json.MarshalIndent(projectsRegistry, "", "  ")

	if encodeErr != nil {
		return Project{}, removeNewProject(newProjectDir, fmt.Errorf("encode project registry: %w", encodeErr))
	}

	registryContents = append(registryContents, '\n')

	writeRegistryErr := writeFileAtomic(c.registryPath(), registryContents, 0600)

	if writeRegistryErr != nil {
		return Project{}, removeNewProject(newProjectDir, fmt.Errorf("write project registry: %w", writeRegistryErr))
	}

	return project, nil
}

// removeNewProject deletes a folder AddProject just created and returns cause.
func removeNewProject(newProjectDir string, cause error) error {
	removeErr := os.RemoveAll(newProjectDir)

	if removeErr != nil {
		return fmt.Errorf("%w; could not remove %s: %v", cause, newProjectDir, removeErr)
	}

	return cause
}
