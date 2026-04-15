package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the TUI main view.
type KeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Up      key.Binding
	Down    key.Binding
	Refresh key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Attach  key.Binding // reserved -- Phase 77
	New     key.Binding // reserved -- Phase 77
}

func defaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("\u2191/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("\u2193/j", "move down"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh list"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/Home", "jump to first"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/End", "jump to last"),
		),
		Attach: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "attach to session"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new session"),
		),
	}
}
