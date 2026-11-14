package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// contentView shows Markdown rendered by Glamour in a scrollable viewport.
// It keeps the source so a resize can render it again at the new width.
type contentView struct {
	viewport viewport.Model
	// style is a Glamour standard style, "dark" or "light".
	style string

	renderer      *glamour.TermRenderer
	rendererWidth int

	// key names what is shown, so new content scrolls to the top but a
	// reload of the same file keeps its place.
	key      string
	markdown string
}

func newContentView(style string) contentView {
	if style == "" {
		style = "dark"
	}

	return contentView{viewport: viewport.New(0, 0), style: style}
}

// empty reports whether nothing is shown, as opposed to an empty file.
func (v contentView) empty() bool {
	return v.key == ""
}

// setSize fits the view to width × height, rendering again if the width changed.
func (v *contentView) setSize(width, height int) {
	v.viewport.Height = height

	if width == v.viewport.Width {
		return
	}

	v.viewport.Width = width
	v.render()
}

// setMarkdown shows markdown. key identifies it; a new key scrolls to the top.
func (v *contentView) setMarkdown(key, markdown string) {
	isNew := key != v.key

	v.key = key
	v.markdown = markdown
	v.render()

	if isNew {
		v.viewport.GotoTop()
	}
}

// clear empties the view.
func (v *contentView) clear() {
	v.setMarkdown("", "")
}

// render draws the Markdown at the viewport's width. If Glamour fails, it
// shows the source as plain text instead.
func (v *contentView) render() {
	if v.markdown == "" || v.viewport.Width <= 0 {
		v.viewport.SetContent("")
		return
	}

	rendered, renderErr := v.renderMarkdown()

	if renderErr != nil {
		v.viewport.SetContent(errorStyle.Render(renderErr.Error()) + "\n\n" + v.markdown)
		return
	}

	v.viewport.SetContent(rendered)
}

// renderMarkdown renders with a renderer for the current width, making a new
// one only when the width changes.
func (v *contentView) renderMarkdown() (string, error) {
	if v.renderer == nil || v.rendererWidth != v.viewport.Width {
		renderer, rendererErr := glamour.NewTermRenderer(
			glamour.WithStandardStyle(v.style),
			glamour.WithWordWrap(v.viewport.Width),
		)

		if rendererErr != nil {
			return "", fmt.Errorf("create Markdown renderer: %w", rendererErr)
		}

		v.renderer = renderer
		v.rendererWidth = v.viewport.Width
	}

	rendered, renderErr := v.renderer.Render(v.markdown)

	if renderErr != nil {
		return "", fmt.Errorf("render Markdown: %w", renderErr)
	}

	return strings.Trim(rendered, "\n"), nil
}

// update scrolls the view. g and G jump to the top and bottom; the
// viewport handles the rest (j/k, pgup/pgdn, u/d).
func (v contentView) update(msg tea.Msg, keys keyMap) (contentView, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Top):
			v.viewport.GotoTop()
			return v, nil
		case key.Matches(keyMsg, keys.Bottom):
			v.viewport.GotoBottom()
			return v, nil
		}
	}

	var cmd tea.Cmd

	v.viewport, cmd = v.viewport.Update(msg)

	return v, cmd
}

func (v contentView) view() string {
	return v.viewport.View()
}

// scrollPercent returns how far down the view is, e.g. "40%", or "" if it all fits.
func (v contentView) scrollPercent() string {
	if v.viewport.TotalLineCount() <= v.viewport.Height {
		return ""
	}

	return fmt.Sprintf("%d%%", int(v.viewport.ScrollPercent()*100))
}
