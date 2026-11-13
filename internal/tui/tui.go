// Package tui is Trail's terminal UI for browsing a data directory.
package tui

import (
	"fmt"

	"github.com/cedricahenkorah/trail/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// Options configures [Run].
type Options struct {
	// Project names the project to select first. If empty, the first one is selected.
	Project string
}

// Run opens the TUI on c and blocks until the user quits.
func Run(c *core.Core, opts Options) error {
	program := tea.NewProgram(newModel(c, opts), tea.WithAltScreen())

	_, runErr := program.Run()

	if runErr != nil {
		return fmt.Errorf("run TUI: %w", runErr)
	}

	return nil
}
