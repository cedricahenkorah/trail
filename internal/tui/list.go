package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// row is a list item drawn on one line: a title with an optional detail on the right.
type row interface {
	list.Item
	title() string
	detail() string
}

// newList returns a list with its own title, status bar, help, and quit
// keys turned off, since the TUI draws and handles those itself.
func newList(singular, plural string) list.Model {
	l := list.New(nil, rowDelegate{}, 0, 0)

	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetStatusBarItemName(singular, plural)
	l.DisableQuitKeybindings()
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.SetShowFilter(false)

	return l
}

// showFilterWhenUsed shows l's filter bar only while a filter is typed,
// since an empty bar leaves a blank line above the list.
func showFilterWhenUsed(l *list.Model) {
	l.SetShowFilter(l.FilterState() == list.Filtering)
}

// rowDelegate draws each row on one line. focused highlights the selected
// row in the focus color.
type rowDelegate struct {
	focused bool
}

func (d rowDelegate) Height() int                             { return 1 }
func (d rowDelegate) Spacing() int                            { return 0 }
func (d rowDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	r, ok := item.(row)

	if !ok {
		return
	}

	selected := index == m.Index()

	prefix := "  "

	if selected {
		prefix = "> "
	}

	detail := r.detail()
	titleWidth := m.Width() - lipgloss.Width(prefix)

	if detail != "" {
		titleWidth -= lipgloss.Width(detail) + 1
	}

	title := ansi.Truncate(r.title(), max(titleWidth, 0), "…")
	gap := strings.Repeat(" ", max(titleWidth-lipgloss.Width(title), 0))

	line := prefix + title

	switch {
	case selected && d.focused:
		line = selectedStyle.Render(line)
	case selected:
		line = titleStyle.Render(line)
	}

	if detail != "" {
		line += gap + " " + mutedStyle.Render(detail)
	}

	fmt.Fprint(w, line)
}

// setListFocus redraws l's selected row for whether its pane is focused.
func setListFocus(l *list.Model, focused bool) {
	l.SetDelegate(rowDelegate{focused: focused})
}
