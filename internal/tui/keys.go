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
	Attach  key.Binding
	New     key.Binding
	Kill    key.Binding // Phase 77
	Rename  key.Binding // Phase 77
	QR      key.Binding // Phase 78
	TabFocus key.Binding // Phase 86: toggle sidebar/content focus
	PrevTab  key.Binding // Phase 86: previous tab
	NextTab  key.Binding // Phase 86: next tab
}

func defaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("Q", "ctrl+c"),
			key.WithHelp("Q", "quit"),
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
			key.WithKeys("R"),
			key.WithHelp("R", "refresh list"),
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
		Kill: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "kill session"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename session"),
		),
		QR: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "QR code / URL"),
		),
		TabFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "toggle focus"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next tab"),
		),
	}
}
