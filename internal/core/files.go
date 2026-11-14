package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Section is a folder of Markdown files inside a project.
type Section string

const (
	Log    Section = "log"
	Tasks  Section = "tasks"
	Issues Section = "issues"
)

// FileSummary describes one file in a section. For Tasks and Issues, Open
// and Total count checklist items; for Log, Total counts entries.
type FileSummary struct {
	Name  string
	Open  int
	Total int
}

// sectionDir returns the path of section inside projectDir, which may not exist yet.
func sectionDir(projectDir string, section Section) (string, error) {
	dir := filepath.Join(projectDir, string(section))

	_, dirErr := dirExists(dir, "project "+string(section)+" folder")

	if dirErr != nil {
		return "", dirErr
	}

	return dir, nil
}

// projectSectionDir returns the path of p's section folder, which may not exist yet.
func (c *Core) projectSectionDir(p Project, section Section) (string, error) {
	projectDir, projectDirErr := c.ProjectDir(p)

	if projectDirErr != nil {
		return "", projectDirErr
	}

	return sectionDir(projectDir, section)
}

// markdownFiles returns the names of the visible .md files in dir, or none if dir is missing.
func markdownFiles(dir string) ([]string, error) {
	entries, readErr := os.ReadDir(dir)

	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("read directory %s: %w", dir, readErr)
	}

	var names []string

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".md") {
			continue
		}

		names = append(names, name)
	}

	return names, nil
}

// Files lists the Markdown files in p's section, sorted by name.
func (c *Core) Files(p Project, section Section) ([]FileSummary, error) {
	dir, dirErr := c.projectSectionDir(p, section)

	if dirErr != nil {
		return nil, dirErr
	}

	if section == Tasks || section == Issues {
		return listChecklistFiles(dir)
	}

	names, namesErr := markdownFiles(dir)

	if namesErr != nil {
		return nil, namesErr
	}

	var summaries []FileSummary

	for _, name := range names {
		path := filepath.Join(dir, name)

		content, readErr := os.ReadFile(path)

		if readErr != nil {
			return nil, fmt.Errorf("read file %s: %w", path, readErr)
		}

		summaries = append(summaries, FileSummary{Name: name, Total: len(ParseLogEntries(content))})
	}

	return summaries, nil
}

// FilePath returns the path of the file name in p's section.
func (c *Core) FilePath(p Project, section Section, name string) (string, error) {
	dir, dirErr := c.projectSectionDir(p, section)

	if dirErr != nil {
		return "", dirErr
	}

	return filepath.Join(dir, name), nil
}

// ReadFile returns the contents of the file name in p's section.
func (c *Core) ReadFile(p Project, section Section, name string) ([]byte, error) {
	path, pathErr := c.FilePath(p, section, name)

	if pathErr != nil {
		return nil, pathErr
	}

	content, readErr := os.ReadFile(path)

	if readErr != nil {
		return nil, fmt.Errorf("read file %s: %w", path, readErr)
	}

	return content, nil
}

// NormalizeFilename checks a user-supplied file name and adds .md if it has no extension.
func NormalizeFilename(fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("filename cannot be left blank")
	}

	if strings.HasPrefix(fileName, ".") || strings.ContainsAny(fileName, `/\`) {
		return "", fmt.Errorf("filename contains invalid characters")
	}

	ext := filepath.Ext(fileName)

	if ext != "" && ext != ".md" {
		return "", fmt.Errorf("filename must have no extension or end in .md")
	}

	if ext == "" {
		fileName += ".md"
	}

	return fileName, nil
}
