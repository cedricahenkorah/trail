package tui

import "github.com/charmbracelet/lipgloss"

const (
	sidebarWidth = 22
	filesWidth   = 32
	helpHeight   = 1
	minWidth     = 70
	minHeight    = 12
)

// layout holds the outer size of each pane, borders included.
type layout struct {
	sidebarWidth int
	filesWidth   int
	contentWidth int
	paneHeight   int
	tooSmall     bool
}

// newLayout splits a width × height window into panes. filesWidth is 0
// when showFiles is false, and content takes its space.
func newLayout(width, height int, showFiles bool) layout {
	if width < minWidth || height < minHeight {
		return layout{tooSmall: true}
	}

	l := layout{
		sidebarWidth: sidebarWidth,
		paneHeight:   height - helpHeight,
	}

	if showFiles {
		l.filesWidth = filesWidth
	}

	l.contentWidth = width - l.sidebarWidth - l.filesWidth

	return l
}

// tooSmallView asks for a bigger window, centered in width × height.
func tooSmallView(width, height int) string {
	message := mutedStyle.Render("Terminal too small\nResize to at least 70×12")

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, message)
}
