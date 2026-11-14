package tui

import (
	"fmt"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pane is one of the TUI's three columns.
type pane int

const (
	projectsPane pane = iota
	filesPane
	contentPane
	paneCount // number of panes; keep last
)

// tab is one of the file list's tabs.
type tab int

const (
	logTab tab = iota
	tasksTab
	issuesTab
	todayTab
)

func (t tab) String() string {
	switch t {
	case logTab:
		return "Log"
	case tasksTab:
		return "Tasks"
	case issuesTab:
		return "Issues"
	case todayTab:
		return "Today"
	default:
		return fmt.Sprintf("tab(%d)", int(t))
	}
}

// model is the root Bubble Tea model.
type model struct {
	core *core.Core
	opts Options
	keys keyMap
	help help.Model

	focus     pane
	activeTab tab

	width  int
	height int
}

func newModel(c *core.Core, opts Options) model {
	return model{
		core: c,
		opts: opts,
		keys: defaultKeyMap(),
		help: help.New(),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextPane):
			m.focus = m.nextPane(1)
		case key.Matches(msg, m.keys.PrevPane):
			m.focus = m.nextPane(-1)
		}
	}

	return m, nil
}

// showFiles reports whether the file list is visible; the Today tab hides it.
func (m model) showFiles() bool {
	return m.activeTab != todayTab
}

// nextPane returns the pane step places from the focused one, skipping
// the file list when it's hidden.
func (m model) nextPane(step int) pane {
	next := m.focus

	for {
		next = (next + pane(step) + paneCount) % paneCount

		if next != filesPane || m.showFiles() {
			return next
		}
	}
}

func (m model) View() string {
	// Bubble Tea sends the window size right after start; draw nothing until then.
	if m.width == 0 {
		return ""
	}

	l := newLayout(m.width, m.height, m.showFiles())

	if l.tooSmall {
		return tooSmallView(m.width, m.height)
	}

	projectName := m.opts.Project

	if projectName == "" {
		projectName = mutedStyle.Render("no project selected")
	}

	panes := []string{
		renderPane("Projects", projectName, l.sidebarWidth, l.paneHeight, m.focus == projectsPane),
	}

	if m.showFiles() {
		panes = append(panes, renderPane(m.activeTab.String(), mutedStyle.Render("No files yet"), l.filesWidth, l.paneHeight, m.focus == filesPane))
	}

	panes = append(panes, renderPane("Content", mutedStyle.Render("Select a file"), l.contentWidth, l.paneHeight, m.focus == contentPane))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, panes...),
		m.help.View(m.keys),
	)
}
