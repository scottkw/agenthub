package tui

import (
	"time"

	"charm.land/bubbles/v2/textinput"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/pty"
)

// modalState represents the currently active modal overlay.
type modalState int

const (
	modalNone modalState = iota
	modalNewSession
	modalKillConfirm
)

// toastKind controls the color of toast messages.
type toastKind int

const (
	toastInfo toastKind = iota
	toastSuccess
	toastError
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

	// Toast enhancement (Phase 77)
	toastKind toastKind

	// Modal state (Phase 77)
	modal        modalState
	agentIdx     int               // current agent picker index
	dirInput     textinput.Model   // directory field in new-session modal
	argsInput    textinput.Model   // arguments field in new-session modal
	focusedField int               // 0=agent, 1=directory, 2=arguments
	detectedCLIs []pty.DetectedCLI // cached on first modal open

	// Kill confirmation state (Phase 77)
	killTarget   *daemon.SessionInfo
	killFocusYes bool

	// Inline rename state (Phase 77)
	editing       bool
	editInput     textinput.Model
	editOriginal  string
	editSessionID string
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

type attachDoneMsg struct{ err error }

type createSessionMsg struct {
	id  string
	err error
}

type killSessionMsg struct{ err error }

type renameSessionMsg struct{ err error }
