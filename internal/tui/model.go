package tui

import (
	"fmt"
	"time"

	"github.com/cedricahenkorah/trail/internal/core"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
	projectList    list.Model
	// project is the selected project, or the zero Project if there are none.
	project core.Project

	fileList list.Model
	fileName string
	content  contentView

	// err is the last load error, shown above the help bar until a reload.
	err error
}

func newModel(c *core.Core, opts Options) model {
	m := model{
		core:        c,
		opts:        opts,
		keys:        defaultKeyMap(),
		help:        help.New(),
		projectList: newProjectList(),
		fileList:    newFileList(),
		content:     newContentView(opts.Style),
	}

	m.setFocus(projectsPane)

	return m
}

func (m model) Init() tea.Cmd {
	return loadProjects(m.core)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := m.update(msg)

	m.resize()

	return m, cmd
}

func (m model) update(msg tea.Msg) (model, tea.Cmd) {
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

	case list.FilterMatchesMsg:
		return m.updateFocused(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if l := m.focusedList(); l != nil && l.FilterState() == list.Filtering {
		return m.updateFocused(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.NextPane):
		m.setFocus(m.nextPane(1))
	case key.Matches(msg, m.keys.PrevPane):
		m.setFocus(m.nextPane(-1))
	case key.Matches(msg, m.keys.GoToTab):
		return m.switchTab(tab(msg.String()[0] - '1'))
	case key.Matches(msg, m.keys.NextTab):
		return m.switchTab((m.activeTab + 1) % tab(len(tabs)))
	case key.Matches(msg, m.keys.PrevTab):
		return m.switchTab((m.activeTab + tab(len(tabs)) - 1) % tab(len(tabs)))
	case key.Matches(msg, m.keys.Open):
		m.open()
	case key.Matches(msg, m.keys.Back) && m.focus == contentPane:
		m.back()
	case key.Matches(msg, m.keys.Reload):
		m.err = nil
		return m, loadProjects(m.core)
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
	default:
		return m.updateFocused(msg)
	}

	return m, nil
}

// focusedList returns the list in the focused pane, or nil if it has none.
func (m *model) focusedList() *list.Model {
	switch {
	case m.focus == projectsPane:
		return &m.projectList
	case m.focus == filesPane && m.showFiles():
		return &m.fileList
	default:
		return nil
	}
}

// open moves focus one pane right: from a project to its files, and from
// a file to its content.
func (m *model) open() {
	switch {
	case m.focus == projectsPane && m.showFiles():
		m.setFocus(filesPane)
	case m.focus == projectsPane, m.focus == filesPane && m.fileName != "":
		m.setFocus(contentPane)
	}
}

// back moves focus from the content to the pane it was opened from.
func (m *model) back() {
	if m.showFiles() {
		m.setFocus(filesPane)
		return
	}

	m.setFocus(projectsPane)
}

// updateFocused passes msg to the focused pane. For a list, it then loads
// whatever is under its cursor if that changed. While a filter is being
// typed the cursor jumps around, so loading waits until it is accepted.
func (m model) updateFocused(msg tea.Msg) (model, tea.Cmd) {
	var listCmd, loadCmd tea.Cmd

	switch m.focus {
	case projectsPane:
		m.projectList, listCmd = m.projectList.Update(msg)

		if m.projectList.FilterState() != list.Filtering {
			m, loadCmd = m.projectCursorMoved()
		}
	case filesPane:
		m.fileList, listCmd = m.fileList.Update(msg)

		if m.fileList.FilterState() != list.Filtering {
			m, loadCmd = m.fileCursorMoved()
		}
	case contentPane:
		m.content, listCmd = m.content.update(msg, m.keys)
	}

	return m, tea.Batch(listCmd, loadCmd)
}

// projectCursorMoved selects the project under the cursor and loads the active tab.
func (m model) projectCursorMoved() (model, tea.Cmd) {
	p, ok := selectedProject(m.projectList)

	if !ok || p.Name == m.project.Name {
		return m, nil
	}

	m.project = p
	m.err = nil
	m.clearTab()

	return m, m.loadTab()
}

// fileCursorMoved selects the file under the cursor and loads it.
func (m model) fileCursorMoved() (model, tea.Cmd) {
	name, ok := selectedFileName(m.fileList)

	if !ok || name == m.fileName {
		return m, nil
	}

	m.fileName = name
	m.err = nil
	m.content.clear()

	section, _ := m.activeTab.section()

	return m, loadFile(m.core, m.project, section, name)
}

// switchTab shows t and loads it, moving focus off the file list if t hides it.
func (m model) switchTab(t tab) (model, tea.Cmd) {
	if t == m.activeTab {
		return m, nil
	}

	m.activeTab = t
	m.err = nil
	m.clearTab()

	if m.focus == filesPane && !m.showFiles() {
		m.setFocus(contentPane)
	}

	return m, m.loadTab()
}

// clearTab drops everything loaded for the active tab.
func (m *model) clearTab() {
	m.fileList.ResetFilter()
	m.fileList.SetItems(nil)
	m.fileName = ""
	m.content.clear()
}

// setFocus focuses p and highlights the selected row only in its list.
func (m *model) setFocus(p pane) {
	m.focus = p
	setListFocus(&m.projectList, p == projectsPane)
	setListFocus(&m.fileList, p == filesPane)
}

// handleProjects keeps the selected project if it still exists, else
// selects opts.Project or the first one, then loads the active tab.
func (m model) handleProjects(msg projectsLoadedMsg) (model, tea.Cmd) {
	m.projectsLoaded = true

	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	preferredName := m.project.Name

	if preferredName == "" {
		preferredName = m.opts.Project
	}

	m.projectList.ResetFilter()
	listCmd := m.projectList.SetItems(projectItems(msg.projects))
	m.projectList.Select(max(projectIndex(m.projectList, preferredName), 0))

	selected, ok := selectedProject(m.projectList)

	if !ok {
		selected = core.Project{}
	}

	if selected.Name != m.project.Name {
		m.clearTab()
	}

	m.project = selected

	return m, tea.Batch(listCmd, m.loadTab())
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
		m.clearTab()
		return m, nil
	}

	m.fileList.ResetFilter()
	listCmd := m.fileList.SetItems(fileItems(msg.files, section))
	m.fileList.Select(max(fileIndex(m.fileList, m.fileName), 0))

	name, ok := selectedFileName(m.fileList)

	if !ok {
		name = ""
	}

	if name != m.fileName {
		m.content.clear()
	}

	m.fileName = name

	if name == "" {
		return m, listCmd
	}

	return m, tea.Batch(listCmd, loadFile(m.core, m.project, section, name))
}

func (m model) handleFile(msg fileLoadedMsg) (model, tea.Cmd) {
	section, ok := m.activeTab.section()

	if msg.project != m.project.Name || !ok || msg.section != section || msg.name != m.fileName {
		return m, nil
	}

	if msg.err != nil {
		m.err = msg.err
		m.content.clear()
		return m, nil
	}

	_, body := core.SplitFrontMatter(msg.content)

	m.content.setMarkdown(msg.project+"/"+string(msg.section)+"/"+msg.name, string(body))

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

	m.content.setMarkdown(msg.project+"/today", todayMarkdown(msg.project, msg.summary))

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

// footer returns the help bar for the focused pane, with the error above it if there is one.
func (m model) footer() string {
	l := m.focusedList()

	footer := m.help.View(paneHelp{
		keys:      m.keys,
		focus:     m.focus,
		filtering: l != nil && l.FilterState() == list.Filtering,
	})

	if m.err != nil {
		status := errorStyle.Inline(true).MaxWidth(m.width).Render("Error: " + m.err.Error())
		footer = lipgloss.JoinVertical(lipgloss.Left, status, footer)
	}

	return footer
}

func (m model) layout() layout {
	return newLayout(m.width, m.height, lipgloss.Height(m.footer()), m.showFiles())
}

// resize fits the lists and content inside their panes.
func (m *model) resize() {
	l := m.layout()

	if l.tooSmall {
		return
	}

	showFilterWhenUsed(&m.projectList)
	showFilterWhenUsed(&m.fileList)

	frameWidth, frameHeight := paneStyle.GetFrameSize()
	bodyHeight := l.paneHeight - frameHeight - paneHeaderHeight

	m.projectList.SetSize(l.sidebarWidth-frameWidth, bodyHeight)
	m.fileList.SetSize(filesWidth-frameWidth, bodyHeight)
	m.content.setSize(l.contentWidth-frameWidth, bodyHeight)
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}

	l := m.layout()

	if l.tooSmall {
		return tooSmallView(m.width, m.height)
	}

	panes := []string{
		renderPane("Projects", m.projectsBody(), l.sidebarWidth, l.paneHeight, m.focus == projectsPane),
	}

	if m.showFiles() {
		panes = append(panes, renderPane(tabBar(m.activeTab, m.focus == filesPane), m.filesBody(), l.filesWidth, l.paneHeight, m.focus == filesPane))
	}

	panes = append(panes, renderPane(m.contentTitle(), m.contentBody(), l.contentWidth, l.paneHeight, m.focus == contentPane))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, panes...),
		m.footer(),
	)
}

func (m model) projectsBody() string {
	if !m.projectsLoaded {
		return mutedStyle.Render("Loading…")
	}

	if len(m.projectList.Items()) == 0 {
		return mutedStyle.Render("No projects yet.\nAdd one with\ntrail project add")
	}

	return m.projectList.View()
}

func (m model) filesBody() string {
	if m.project.Name == "" {
		return ""
	}

	if len(m.fileList.Items()) == 0 {
		return mutedStyle.Render("No files")
	}

	return m.fileList.View()
}

// contentTitle is the file name, or the tab bar on the Today tab, where
// the file list and its tab bar are hidden.
func (m model) contentTitle() string {
	if m.activeTab == todayTab {
		return tabBar(m.activeTab, m.focus == contentPane)
	}

	if m.fileName == "" {
		return "Content"
	}

	if percent := m.content.scrollPercent(); percent != "" {
		return m.fileName + "  " + mutedStyle.Render(percent)
	}

	return m.fileName
}

// contentBody is the rendered file or today summary.
func (m model) contentBody() string {
	if m.project.Name == "" {
		return ""
	}

	if m.content.empty() && m.activeTab == todayTab && m.err != nil {
		return mutedStyle.Render("Couldn't load today's summary")
	}

	if m.content.empty() && m.activeTab == todayTab {
		return mutedStyle.Render("Loading…")
	}

	if m.content.empty() {
		return mutedStyle.Render("No file selected")
	}

	return m.content.view()
}
