package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the TUI's key bindings. It implements help.KeyMap.
type keyMap struct {
	NextPane key.Binding
	PrevPane key.Binding
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
	return []key.Binding{k.NextPane, k.Reload, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.NextPane, k.PrevPane}, {k.Reload, k.Quit}}
}
