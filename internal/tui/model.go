package tui

import (
	"time"

	"github.com/scottkw/agenthub/internal/daemon"
)

// Model holds all state for the Bubble Tea TUI.
type Model struct {
	client    *daemon.DaemonClient
	sessions  []daemon.SessionInfo
	webStatus daemon.WebServerStatusResponse
	selected  int
	width     int
	height    int
	showHelp  bool
	loading   bool
	err       error
	hasDark   bool
	styles    Styles
	keys      KeyMap
	toast     string    // reserved-key feedback message
	toastExp  time.Time // toast expiry timestamp
}

// Message types for Bubble Tea Update loop.

type sessionsMsg struct {
	sessions []daemon.SessionInfo
	err      error
}

type webStatusMsg struct {
	status daemon.WebServerStatusResponse
	err    error
}

type tickMsg time.Time
