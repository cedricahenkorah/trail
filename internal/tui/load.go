package tui

import (
	"github.com/cedricahenkorah/trail/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// Each load runs as a tea.Cmd so file reads stay off the UI loop. Results
// carry what was asked for, so the model can drop ones that arrive late.

type projectsLoadedMsg struct {
	projects []core.Project
	err      error
}

type filesLoadedMsg struct {
	project string
	section core.Section
	files   []core.FileSummary
	err     error
}

type fileLoadedMsg struct {
	project string
	section core.Section
	name    string
	content []byte
	err     error
}

type todayLoadedMsg struct {
	project string
	summary core.TodaySummary
	err     error
}

// loadProjects lists the active projects.
func loadProjects(c *core.Core) tea.Cmd {
	return func() tea.Msg {
		projects, err := c.Projects()

		return projectsLoadedMsg{projects: core.ActiveProjects(projects), err: err}
	}
}

// loadFiles lists the files in p's section.
func loadFiles(c *core.Core, p core.Project, section core.Section) tea.Cmd {
	return func() tea.Msg {
		files, err := c.Files(p, section)

		return filesLoadedMsg{project: p.Name, section: section, files: files, err: err}
	}
}

// loadFile reads the file name in p's section.
func loadFile(c *core.Core, p core.Project, section core.Section, name string) tea.Cmd {
	return func() tea.Msg {
		content, err := c.ReadFile(p, section, name)

		return fileLoadedMsg{project: p.Name, section: section, name: name, content: content, err: err}
	}
}

// loadToday builds p's summary for date, a YYYY-MM-DD string.
func loadToday(c *core.Core, p core.Project, date string) tea.Cmd {
	return func() tea.Msg {
		summary, err := c.Today(p, date)

		return todayLoadedMsg{project: p.Name, summary: summary, err: err}
	}
}
