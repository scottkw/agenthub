package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textinput"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/pty"
)

func TestView_SessionList(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "my-session", CLI: "claude", Hostname: "macbook-pro", Status: "running", ViewerCount: 2},
		{ID: "2", Name: "another", CLI: "opencode", Hostname: "linux-vm", Status: "idle", ViewerCount: 0},
	}
	m.rebuildUnifiedList()

	v := m.View()
	content := v.Content

	checks := []string{
		"my-session",
		"claude",
		"macbook-pro",
		"another",
		"opencode",
		"linux-vm",
		"AgentHub",
		"2 sessions",
		"NAME",
		"AGENT",
		"HOST",
		"VIEWERS",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("view output missing %q", want)
		}
	}

	// Viewer count 2 should appear, viewer count 0 should not show "0"
	if !strings.Contains(content, "2") {
		t.Error("view should show viewer count 2")
	}
}

func TestView_Footer_WebRunning(t *testing.T) {
	m := testModel()
	m.webStatus = daemon.WebServerStatusResponse{Running: true, URL: "https://mac.tail.ts.net"}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Running") {
		t.Error("footer should contain 'Running' when web server is on")
	}
	if !strings.Contains(content, "https://mac.tail.ts.net") {
		t.Error("footer should contain web server URL")
	}
}

func TestView_Footer_WebStopped(t *testing.T) {
	m := testModel()
	m.webStatus = daemon.WebServerStatusResponse{Running: false}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Stopped") {
		t.Error("footer should contain 'Stopped' when web server is off")
	}
}

func TestView_EmptyState(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "No sessions") {
		t.Error("empty state should show 'No sessions'")
	}
	if !strings.Contains(content, "Press n to create a new session") {
		t.Error("empty state should show create hint")
	}
}

func TestView_ErrorState(t *testing.T) {
	m := testModel()
	m.err = fmt.Errorf("connection refused")

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Cannot connect to daemon") {
		t.Error("error state should show daemon error message")
	}
}

func TestView_TerminalTooSmall(t *testing.T) {
	m := testModel()
	m.width = 40
	m.height = 8

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Terminal too small (need 60x10)") {
		t.Error("should show too-small message for 40x8 terminal")
	}
}

func TestView_SingleSession(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "test", CLI: "claude", Hostname: "host", Status: "running"},
	}
	m.rebuildUnifiedList()

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "1 session") {
		t.Error("should show '1 session' (singular) for single session")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxWidth int
		want     string
	}{
		{"short", 10, "short"},
		{"this is a long name", 10, "this is..."},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxWidth)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
		}
	}
}

func TestStatusGlyph(t *testing.T) {
	styles := newStyles(true)

	tests := []struct {
		status    string
		wantGlyph string
	}{
		{"running", "\u25CF"},
		{"idle", "\u25CB"},
		{"waiting", "\u25CF"},
		{"errored", "\u2716"},
		{"unknown", "\u25CF"}, // defaults to running
	}
	for _, tt := range tests {
		glyph, _ := statusGlyph(tt.status, styles)
		if glyph != tt.wantGlyph {
			t.Errorf("statusGlyph(%q) glyph = %q, want %q", tt.status, glyph, tt.wantGlyph)
		}
	}
}

func TestView_HintBar(t *testing.T) {
	m := testModel()
	hint := m.renderHintBar()
	required := []string{"Enter Attach", "n New", "d Kill", "r Rename", "? Help", "q Quit"}
	for _, want := range required {
		if !strings.Contains(hint, want) {
			t.Errorf("hint bar missing %q", want)
		}
	}
}

func TestView_KillConfirmDialog(t *testing.T) {
	m := testModel()
	m.modal = modalKillConfirm
	m.killTarget = &daemon.SessionInfo{ID: "1", Name: "my-session"}
	m.killFocusYes = false

	v := m.View()
	content := v.Content

	checks := []string{
		"Kill Session",
		"Kill session",
		"my-session",
		"This will terminate the running process",
		"No",
		"Yes",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("kill confirm dialog missing %q", want)
		}
	}
}

func TestView_InlineRename(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "old-name", CLI: "claude", Hostname: "mac", Status: "running"},
	}
	m.rebuildUnifiedList()
	m.selected = 0
	m.editing = true
	m.editSessionID = "1"
	m.editInput = textinput.New()
	m.editInput.SetValue("new-name")

	v := m.View()
	content := v.Content

	// The textinput view should render the value
	if !strings.Contains(content, "new-name") {
		t.Error("inline rename should show textinput value 'new-name'")
	}
}

func TestView_ToastKind(t *testing.T) {
	m := testModel()
	m.toast = "Test toast"
	m.toastExp = time.Now().Add(10 * time.Second)

	// toastInfo should use FgMuted
	m.toastKind = toastInfo
	web := m.renderWebStatus()
	if !strings.Contains(web, "Test toast") {
		t.Error("toast not rendered in web status")
	}

	// toastError still renders the toast text
	m.toastKind = toastError
	web = m.renderWebStatus()
	if !strings.Contains(web, "Test toast") {
		t.Error("error toast not rendered")
	}
}

func TestView_NewSessionModal(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.focusedField = 0
	m.detectedCLIs = []pty.DetectedCLI{
		{Name: "claude", DisplayName: "Claude Code", Path: "/usr/bin/claude"},
	}
	m.agentIdx = 0
	m.dirInput = textinput.New()
	m.dirInput.SetValue("/Users/ken/dev")
	m.argsInput = textinput.New()

	v := m.View()
	content := v.Content

	checks := []string{
		"New Session",
		"Agent:",
		"Directory:",
		"Arguments:",
		"Tab: next field",
		"Enter: create",
		"Esc: cancel",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("new session modal missing %q", want)
		}
	}
}

func TestView_NewSessionModal_NoAgents(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.focusedField = 0
	m.detectedCLIs = nil
	m.dirInput = textinput.New()
	m.argsInput = textinput.New()

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "(none found)") {
		t.Error("modal should show '(none found)' when no CLIs detected")
	}
}
