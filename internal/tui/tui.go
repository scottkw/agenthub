package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/pty"
)

// Run starts the Bubble Tea TUI program. Blocks until the user quits.
// Returns nil on clean exit, or an error if the program fails.
// fetchRemoteFn is an optional callback for fetching remote tailnet sessions;
// pass nil if tailnet is not configured.
// version is the app build version string displayed on the Home tab.
func Run(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn, version string) error {
	p := tea.NewProgram(newModel(client, fetchRemoteFn, version))
	_, err := p.Run()
	return err
}

// newModel creates the initial Model with default state.
// Assumes dark background until tea.BackgroundColorMsg arrives.
//
// Phase 108 PARITY-TUI-04: shell discovery removed from the TUI startup
// path. The single "Shell" picker entry is static — the daemon resolves the
// binary via shellPath (engine.go:500-530) when cli=="shell".
func newModel(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn, version string) Model {
	return Model{
		client:        client,
		loading:       true,
		keys:          defaultKeyMap(),
		styles:        newStyles(true), // assume dark until BackgroundColorMsg
		detectedCLIs:  pty.DetectCLIs(),
		fetchRemoteFn: fetchRemoteFn,
		version:       version,
		// Initial tab state: Sessions open by default (matches current UX)
		openTabs:     []tabID{tabSessions},
		activeTab:    0,
		panesFocus:   focusContent,
		sidebarFocus: 1, // 1 = Sessions
	}
}

// Init returns the initial batch of commands:
// 1. Request background color detection (for adaptive styles)
// 2. Fetch sessions from daemon
// 3. Fetch web server status from daemon
// 4. Fetch remote tailnet sessions (if configured)
// 5. Start the 2-second polling tick
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		fetchSessions(m.client),
		fetchWebStatus(m.client),
		fetchRemoteSessions(m.fetchRemoteFn),
		nextTick(),
	)
}
