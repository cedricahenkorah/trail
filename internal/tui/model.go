package tui

import (
	"fmt"
	"strings"
	"time"

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

// section returns the core section t lists. ok is false for the Today tab.
func (t tab) section() (section core.Section, ok bool) {
	switch t {
	case logTab:
		return core.Log, true
	case tasksTab:
		return core.Tasks, true
	case issuesTab:
		return core.Issues, true
	default:
		return "", false
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

	projectsLoaded bool
	projects       []core.Project
	// project is the selected project, or the zero Project if there are none.
	project  core.Project
	files    []core.FileSummary
	fileName string
	content  []byte
	today    core.TodaySummary
	// err is the last load error, shown above the help bar until a reload.
	err error
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
	return loadProjects(m.core)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case projectsLoadedMsg:
		return m.handleProjects(msg)

	case filesLoadedMsg:
		return m.handleFiles(msg)

	case fileLoadedMsg:
		return m.handleFile(msg)

	case todayLoadedMsg:
		return m.handleToday(msg)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextPane):
			m.focus = m.nextPane(1)
		case key.Matches(msg, m.keys.PrevPane):
			m.focus = m.nextPane(-1)
		case key.Matches(msg, m.keys.Reload):
			m.err = nil
			return m, loadProjects(m.core)
		}
	}

	return m, nil
}

// handleProjects keeps the selected project if it still exists, else
// selects opts.Project or the first one, then loads the active tab.
func (m model) handleProjects(msg projectsLoadedMsg) (model, tea.Cmd) {
	m.projectsLoaded = true

	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	m.projects = msg.projects

	preferredName := m.project.Name

	if preferredName == "" {
		preferredName = m.opts.Project
	}

	selected := core.Project{}

	for _, p := range m.projects {
		if p.Name == preferredName {
			selected = p
		}
	}

	if selected.Name == "" && len(m.projects) > 0 {
		selected = m.projects[0]
	}

	if selected.Name != m.project.Name {
		m.files = nil
		m.fileName = ""
		m.content = nil
		m.today = core.TodaySummary{}
	}

	m.project = selected

	return m, m.loadTab()
}

// loadTab loads the active tab for the selected project.
func (m model) loadTab() tea.Cmd {
	if m.project.Name == "" {
		return nil
	}

	section, ok := m.activeTab.section()

	if !ok {
		return loadToday(m.core, m.project, time.Now().Format("2006-01-02"))
	}

	return loadFiles(m.core, m.project, section)
}

// handleFiles keeps the selected file if it still exists, else selects
// the first one, then loads it.
func (m model) handleFiles(msg filesLoadedMsg) (model, tea.Cmd) {
	section, ok := m.activeTab.section()

	if msg.project != m.project.Name || !ok || msg.section != section {
		return m, nil
	}

	if msg.err != nil {
		m.err = msg.err
		m.files = nil
		m.fileName = ""
		m.content = nil
		return m, nil
	}

	m.files = msg.files

	previous := m.fileName
	m.fileName = ""

	for _, file := range m.files {
		if file.Name == previous {
			m.fileName = previous
		}
	}

	if m.fileName == "" && len(m.files) > 0 {
		m.fileName = m.files[0].Name
	}

	if m.fileName != previous {
		m.content = nil
	}

	if m.fileName == "" {
		return m, nil
	}

	return m, loadFile(m.core, m.project, section, m.fileName)
}

func (m model) handleFile(msg fileLoadedMsg) (model, tea.Cmd) {
	section, ok := m.activeTab.section()

	if msg.project != m.project.Name || !ok || msg.section != section || msg.name != m.fileName {
		return m, nil
	}

	if msg.err != nil {
		m.err = msg.err
		m.content = nil
		return m, nil
	}

	m.content = msg.content

	return m, nil
}

func (m model) handleToday(msg todayLoadedMsg) (model, tea.Cmd) {
	if msg.project != m.project.Name || m.activeTab != todayTab {
		return m, nil
	}

	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	m.today = msg.summary

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

	footer := m.help.View(m.keys)

	if m.err != nil {
		status := errorStyle.Inline(true).MaxWidth(m.width).Render("Error: " + m.err.Error())
		footer = lipgloss.JoinVertical(lipgloss.Left, status, footer)
	}

	l := newLayout(m.width, m.height, lipgloss.Height(footer), m.showFiles())

	if l.tooSmall {
		return tooSmallView(m.width, m.height)
	}

	panes := []string{
		renderPane("Projects", m.projectsBody(), l.sidebarWidth, l.paneHeight, m.focus == projectsPane),
	}

	if m.showFiles() {
		panes = append(panes, renderPane(m.activeTab.String(), m.filesBody(), l.filesWidth, l.paneHeight, m.focus == filesPane))
	}

	panes = append(panes, renderPane(m.contentTitle(), m.contentBody(), l.contentWidth, l.paneHeight, m.focus == contentPane))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, panes...),
		footer,
	)
}

// The pane bodies below are plain placeholders; steps 5 to 8 replace them.

func (m model) projectsBody() string {
	if !m.projectsLoaded {
		return mutedStyle.Render("Loading…")
	}

	if len(m.projects) == 0 {
		return mutedStyle.Render("No projects yet.\nAdd one with\ntrail project add")
	}

	var lines []string

	for _, p := range m.projects {
		lines = append(lines, marker(p.Name == m.project.Name)+p.Name)
	}

	return strings.Join(lines, "\n")
}

func (m model) filesBody() string {
	if m.project.Name == "" {
		return ""
	}

	if len(m.files) == 0 {
		return mutedStyle.Render("No files")
	}

	var lines []string

	for _, file := range m.files {
		count := fmt.Sprint(file.Total)

		if m.activeTab != logTab {
			count = fmt.Sprintf("%d/%d", file.Open, file.Total)
		}

		lines = append(lines, marker(file.Name == m.fileName)+file.Name+"  "+mutedStyle.Render(count))
	}

	return strings.Join(lines, "\n")
}

func (m model) contentTitle() string {
	if m.activeTab == todayTab {
		return "Today"
	}

	if m.fileName != "" {
		return m.fileName
	}

	return "Content"
}

func (m model) contentBody() string {
	if m.project.Name == "" {
		return ""
	}

	if m.activeTab == todayTab {
		return fmt.Sprintf(
			"Devlog: %d entries\nOpen tasks: %d\nUnresolved issues: %d",
			len(m.today.Devlog),
			countItems(m.today.Tasks),
			countItems(m.today.Issues),
		)
	}

	if m.content == nil {
		return mutedStyle.Render("No file selected")
	}

	_, body := core.SplitFrontMatter(m.content)

	return string(body)
}

// marker returns the prefix for a list row, "> " if it is selected.
func marker(selected bool) string {
	if selected {
		return "> "
	}

	return "  "
}

// countItems returns the number of items across files.
func countItems(files []core.ChecklistFile) int {
	count := 0

	for _, file := range files {
		count += len(file.Items)
	}

	return count
}
