// Package tui is Trail's terminal UI for browsing a data directory.
package tui

import (
	"fmt"

	"github.com/cedricahenkorah/trail/internal/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Options configures [Run].
type Options struct {
	// Project names the project to select first. If empty, the first one is selected.
	Project string
	// Style is the Glamour style for file contents, "dark" or "light". If
	// empty, it follows the terminal's background.
	Style string
}

// Run opens the TUI on c and blocks until the user quits.
func Run(c *core.Core, opts Options) error {
	// Ask the terminal for its background before Bubble Tea takes over input.
	if opts.Style == "" {
		opts.Style = "light"

		if lipgloss.HasDarkBackground() {
			opts.Style = "dark"
		}
	}

	program := tea.NewProgram(newModel(c, opts), tea.WithAltScreen())

	_, runErr := program.Run()

	if runErr != nil {
		return fmt.Errorf("run TUI: %w", runErr)
	}

	return nil
}
