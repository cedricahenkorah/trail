package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the TUI's key bindings. It implements help.KeyMap. Moving
// within a list (j/k, g/G, /) is handled by the list itself, and scrolling
// the content (j/k, pgup/pgdn) by its viewport.
type keyMap struct {
	NextPane key.Binding
	PrevPane key.Binding
	GoToTab  key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding
	Open     key.Binding
	Back     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Reload   key.Binding
	Quit     key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous pane"),
		),
		GoToTab: key.NewBinding(
			key.WithKeys("1", "2", "3", "4"),
			key.WithHelp("1-4", "go to tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous tab"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextPane, k.Open, k.GoToTab, k.Reload, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPane, k.PrevPane},
		{k.Open, k.Back, k.Top, k.Bottom},
		{k.GoToTab, k.NextTab, k.PrevTab},
		{k.Reload, k.Quit},
	}
}
