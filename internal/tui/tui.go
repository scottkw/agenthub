package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

// Run starts the Bubble Tea TUI program. Blocks until the user quits.
// Returns nil on clean exit, or an error if the program fails.
func Run(client *daemon.DaemonClient) error {
	p := tea.NewProgram(newModel(client))
	_, err := p.Run()
	return err
}

// newModel creates the initial Model with default state.
// Assumes dark background until tea.BackgroundColorMsg arrives.
func newModel(client *daemon.DaemonClient) Model {
	return Model{
		client:  client,
		loading: true,
		keys:    defaultKeyMap(),
		styles:  newStyles(true), // assume dark until BackgroundColorMsg
	}
}

// Init returns the initial batch of commands:
// 1. Request background color detection (for adaptive styles)
// 2. Fetch sessions from daemon
// 3. Fetch web server status from daemon
// 4. Start the 2-second polling tick
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		fetchSessions(m.client),
		fetchWebStatus(m.client),
		nextTick(),
	)
}
