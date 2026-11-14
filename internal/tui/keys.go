package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the TUI's key bindings. The bindings under "Help only" are
// handled by the lists and the viewport themselves; they are here so the
// help bar can show them.
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
	Help     key.Binding
	Quit     key.Binding

	// Help only.
	Move         key.Binding
	Scroll       key.Binding
	Page         key.Binding
	Filter       key.Binding
	ApplyFilter  key.Binding
	CancelFilter key.Binding
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
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more keys"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),

		Move: key.NewBinding(
			key.WithKeys("j", "k", "up", "down"),
			key.WithHelp("j/k", "move"),
		),
		Scroll: key.NewBinding(
			key.WithKeys("j", "k", "up", "down"),
			key.WithHelp("j/k", "scroll"),
		),
		Page: key.NewBinding(
			key.WithKeys("pgup", "pgdown"),
			key.WithHelp("pgup/pgdn", "page"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ApplyFilter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
		CancelFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel filter"),
		),
	}
}

// paneHelp picks the help for the focused pane. It implements help.KeyMap.
type paneHelp struct {
	keys      keyMap
	focus     pane
	filtering bool
}

func (h paneHelp) ShortHelp() []key.Binding {
	k := h.keys

	switch {
	case h.filtering:
		return []key.Binding{k.ApplyFilter, k.CancelFilter}
	case h.focus == contentPane:
		return []key.Binding{k.Scroll, k.Top, k.Bottom, k.Back, k.NextPane, k.Help, k.Quit}
	default:
		return []key.Binding{k.Move, k.Filter, k.Open, k.NextPane, k.GoToTab, k.Help, k.Quit}
	}
}

func (h paneHelp) FullHelp() [][]key.Binding {
	k := h.keys

	if h.filtering {
		return [][]key.Binding{{k.ApplyFilter, k.CancelFilter}}
	}

	moving := []key.Binding{k.Move, k.Top, k.Bottom, k.Page, k.Filter}
	panes := []key.Binding{k.NextPane, k.PrevPane, k.Open}

	if h.focus == contentPane {
		moving = []key.Binding{k.Scroll, k.Top, k.Bottom, k.Page}
		panes = []key.Binding{k.NextPane, k.PrevPane, k.Back}
	}

	// The full help is showing, so ? now hides it.
	k.Help.SetHelp("?", "fewer keys")

	return [][]key.Binding{
		moving,
		panes,
		{k.GoToTab, k.NextTab, k.PrevTab},
		{k.Reload, k.Help, k.Quit},
	}
}
