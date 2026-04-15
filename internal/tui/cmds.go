package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

// fetchSessions returns a tea.Cmd that fetches sessions from the daemon.
// Runs in a goroutine -- never blocks the Update loop.
func fetchSessions(client *daemon.DaemonClient) tea.Cmd {
	return func() tea.Msg {
		sessions, err := client.ListSessions()
		return sessionsMsg{sessions: sessions, err: err}
	}
}

// fetchWebStatus returns a tea.Cmd that fetches web server status from the daemon.
func fetchWebStatus(client *daemon.DaemonClient) tea.Cmd {
	return func() tea.Msg {
		status, err := client.GetWebServerStatus()
		return webStatusMsg{status: status, err: err}
	}
}

// nextTick returns a tea.Cmd that fires a tickMsg after 2 seconds.
// The tick triggers a re-fetch of sessions and web status.
func nextTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
