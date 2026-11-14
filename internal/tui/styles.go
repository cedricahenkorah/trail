package tui

import "github.com/charmbracelet/lipgloss"

var (
	borderColor        = lipgloss.AdaptiveColor{Light: "250", Dark: "240"}
	focusedBorderColor = lipgloss.AdaptiveColor{Light: "63", Dark: "62"}

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	focusedPaneStyle = paneStyle.BorderForeground(focusedBorderColor)

	titleStyle = lipgloss.NewStyle().Bold(true)

	mutedStyle = lipgloss.NewStyle().Faint(true)
)

// renderPane draws title and body in a bordered box of outer size width × height.
func renderPane(title, body string, width, height int, focused bool) string {
	style := paneStyle

	if focused {
		style = focusedPaneStyle
	}

	frameWidth, frameHeight := style.GetFrameSize()
	innerWidth := width - frameWidth
	innerHeight := height - frameHeight

	content := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(title), "", body)
	content = lipgloss.NewStyle().MaxWidth(innerWidth).MaxHeight(innerHeight).Render(content)

	// Width and Height include padding but not the border.
	return style.
		Width(width - style.GetHorizontalBorderSize()).
		Height(innerHeight).
		Render(content)
}
