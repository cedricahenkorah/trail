package tui

import (
	"fmt"

	"github.com/cedricahenkorah/trail/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// model is the root Bubble Tea model.
type model struct {
	core *core.Core
	opts Options
}

func newModel(c *core.Core, opts Options) model {
	return model{core: c, opts: opts}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	project := m.opts.Project

	if project == "" {
		project = "no project selected"
	}

	return fmt.Sprintf("Trail · %s\n\nq to quit\n", project)
}
