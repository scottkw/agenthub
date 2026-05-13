package tui

import (
	"context"
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

// listEntryKind identifies the type of row in the unified session list.
type listEntryKind int

const (
	entryLocal listEntryKind = iota
	entryRemote
	entryDivider
)

// panesFocusState tracks which pane has keyboard focus.
type panesFocusState bool

const (
	focusContent panesFocusState = false
	focusSidebar panesFocusState = true
)

// tabID identifies a content tab.
type tabID int

const (
	tabHome     tabID = iota
	tabSessions tabID = iota
	tabRemote   tabID = iota
	tabSettings tabID = iota
)

// listEntry represents one row in the unified session list (local session, remote session, or divider).
type listEntry struct {
	kind    listEntryKind
	session *daemon.SessionInfo // non-nil when kind == entryLocal
	remote  *RemoteSessionEntry // non-nil when kind == entryRemote
	divider *peerDivider        // non-nil when kind == entryDivider
}

// RemoteSessionEntry holds fields for a remote peer session displayed in the TUI.
// Exported so cmd_tui.go (package main) can construct values for the callback.
type RemoteSessionEntry struct {
	ID       string
	Name     string
	CLIType  string
	Status   string
	Hostname string
	FQDN     string
	URL      string // pre-built: https://{fqdn}:{port}/sessions/{id}
}

// peerDivider holds metadata for a section divider row between peer groups.
type peerDivider struct {
	Hostname     string
	SessionCount int
}

// sessionRef captures identity of a session for QR overlay and selection restoration.
type sessionRef struct {
	ID       string
	Name     string
	IsRemote bool
	URL      string
}

// ListRemoteGroup groups remote sessions by peer hostname.
// Exported so cmd_tui.go (package main) can construct values for the callback.
type ListRemoteGroup struct {
	Hostname string
	Sessions []RemoteSessionEntry
}

// FetchRemoteFn is a callback that fetches remote session groups from tailnet peers.
// Injected from cmd_tui.go to avoid package-main import cycle.
type FetchRemoteFn func(ctx context.Context) []ListRemoteGroup

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
	agentIdx     int               // current agent picker index (covers AI CLIs + shells)
	dirInput     textinput.Model   // directory field in new-session modal
	argsInput    textinput.Model   // arguments field in new-session modal
	focusedField int               // 0=agent, 1=directory, 2=arguments
	detectedCLIs []pty.DetectedCLI // cached on first modal open

	// Phase 101 SHELL-03: shell entries surfaced in the agent picker. Populated
	// in-process via pty.DiscoverShells() (RESEARCH A1 — TUI calls pty directly,
	// not via DaemonClient). Order is preserved as supplied; the picker
	// applies sortShellsForPicker each render so display order is deterministic
	// (shell → bash → zsh → pwsh → powershell).
	detectedShells []pty.DetectedShell

	// Kill confirmation state (Phase 77)
	killTarget   *daemon.SessionInfo
	killFocusYes bool

	// Inline rename state (Phase 77)
	editing       bool
	editInput     textinput.Model
	editOriginal  string
	editSessionID string

	// Remote sessions and unified list (Phase 78)
	remoteSessions []ListRemoteGroup // last fetched remote session groups
	unifiedList    []listEntry       // computed from sessions + remoteSessions on each update
	fetchRemoteFn  FetchRemoteFn     // injected callback for remote fetch

	// QR overlay state (Phase 78)
	qrSession *sessionRef // nil = no QR overlay; non-nil = QR overlay open
	qrContent string      // pre-rendered QR string from ToSmallString(false)
	qrURL     string      // URL shown below QR

	// Tab navigation state (Phase 86)
	panesFocus   panesFocusState // focusContent or focusSidebar
	openTabs     []tabID         // ordered list of open tabs
	activeTab    int             // index into openTabs
	sidebarFocus int             // 0=Home, 1=Sessions, 2=Remote, 3=Settings
	version      string          // app version string for Home tab
}

// activeTabID returns the tabID of the currently active tab, defaulting to tabSessions.
func (m Model) activeTabID() tabID {
	if m.activeTab >= 0 && m.activeTab < len(m.openTabs) {
		return m.openTabs[m.activeTab]
	}
	return tabSessions
}

// tabName returns the display name for a tab ID.
func tabName(id tabID) string {
	switch id {
	case tabHome:
		return "Home"
	case tabSessions:
		return "Sessions"
	case tabRemote:
		return "Remote"
	case tabSettings:
		return "Settings"
	default:
		return "?"
	}
}

// openTab opens the given tab (or activates it if already open).
func (m *Model) openTab(id tabID) {
	for i, t := range m.openTabs {
		if t == id {
			m.activeTab = i
			return
		}
	}
	m.openTabs = append(m.openTabs, id)
	m.activeTab = len(m.openTabs) - 1
}

// cycleTab cycles through open tabs by the given direction (+1 or -1).
func (m *Model) cycleTab(dir int) {
	if len(m.openTabs) == 0 {
		return
	}
	m.activeTab = (m.activeTab + dir + len(m.openTabs)) % len(m.openTabs)
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

type remoteSessionsMsg struct {
	groups []ListRemoteGroup
}
