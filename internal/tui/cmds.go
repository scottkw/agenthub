package tui

import (
	"context"
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

// createSession returns a tea.Cmd that creates a new session via the daemon.
func createSession(client *daemon.DaemonClient, cli, name, workDir string, args []string) tea.Cmd {
	return func() tea.Msg {
		id, err := client.CreateSession(cli, name, workDir, args, 0, 0)
		return createSessionMsg{id: id, err: err}
	}
}

// killSession returns a tea.Cmd that kills the given session via the daemon.
func killSession(client *daemon.DaemonClient, id string) tea.Cmd {
	return func() tea.Msg {
		err := client.KillSession(id)
		return killSessionMsg{err: err}
	}
}

// renameSession returns a tea.Cmd that renames the given session via the daemon.
func renameSession(client *daemon.DaemonClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.RenameSession(id, name)
		return renameSessionMsg{err: err}
	}
}

// fetchRemoteSessions returns a tea.Cmd that fetches remote sessions via the injected callback.
// Returns empty remoteSessionsMsg if callback is nil (no tailnet configured).
func fetchRemoteSessions(fn FetchRemoteFn) tea.Cmd {
	return func() tea.Msg {
		if fn == nil {
			return remoteSessionsMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		groups := fn(ctx)
		return remoteSessionsMsg{groups: groups}
	}
}
