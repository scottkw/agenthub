package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

func testModel() Model {
	m := newModel(nil) // nil client -- tests don't make HTTP calls
	m.width = 120
	m.height = 24
	m.hasDark = true
	m.styles = newStyles(true)
	m.loading = false
	return m
}

func TestUpdate_SessionsMsg(t *testing.T) {
	m := testModel()
	m.loading = true

	sessions := []daemon.SessionInfo{
		{ID: "1", Name: "test-session", CLI: "claude", Hostname: "macbook", Status: "running"},
	}
	updated, _ := m.Update(sessionsMsg{sessions: sessions})
	result := updated.(Model)

	if result.loading {
		t.Error("expected loading=false after sessionsMsg")
	}
	if len(result.sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(result.sessions))
	}
	if result.sessions[0].Name != "test-session" {
		t.Errorf("expected session name 'test-session', got %q", result.sessions[0].Name)
	}
}

func TestUpdate_SessionsMsg_Error(t *testing.T) {
	m := testModel()
	m.loading = true

	updated, _ := m.Update(sessionsMsg{err: fmt.Errorf("connection refused")})
	result := updated.(Model)

	if result.loading {
		t.Error("expected loading=false after error sessionsMsg")
	}
	if result.err == nil {
		t.Error("expected err to be set after error sessionsMsg")
	}
}

func TestUpdate_WebStatusMsg(t *testing.T) {
	m := testModel()

	status := daemon.WebServerStatusResponse{Running: true, URL: "https://test.ts.net"}
	updated, _ := m.Update(webStatusMsg{status: status})
	result := updated.(Model)

	if !result.webStatus.Running {
		t.Error("expected webStatus.Running=true")
	}
	if result.webStatus.URL != "https://test.ts.net" {
		t.Errorf("expected URL 'https://test.ts.net', got %q", result.webStatus.URL)
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	result := updated.(Model)

	if result.width != 200 {
		t.Errorf("expected width=200, got %d", result.width)
	}
	if result.height != 50 {
		t.Errorf("expected height=50, got %d", result.height)
	}
}

func TestUpdate_KeyQuit(t *testing.T) {
	m := testModel()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	// tea.Quit returns a special command -- executing it produces tea.QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdate_KeyHelp(t *testing.T) {
	m := testModel()
	m.showHelp = false

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	result := updated.(Model)

	if !result.showHelp {
		t.Error("expected showHelp=true after ? key")
	}

	// Press ? again to close
	updated, _ = result.Update(tea.KeyPressMsg{Code: '?'})
	result = updated.(Model)

	if result.showHelp {
		t.Error("expected showHelp=false after second ? key")
	}
}

func TestUpdate_KeyNavigation(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "s1"},
		{ID: "2", Name: "s2"},
		{ID: "3", Name: "s3"},
	}
	m.selected = 0

	// Move down
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := updated.(Model)
	if result.selected != 1 {
		t.Errorf("expected selected=1 after j, got %d", result.selected)
	}

	// Move down again
	updated, _ = result.Update(tea.KeyPressMsg{Code: 'j'})
	result = updated.(Model)
	if result.selected != 2 {
		t.Errorf("expected selected=2 after second j, got %d", result.selected)
	}

	// Move up
	updated, _ = result.Update(tea.KeyPressMsg{Code: 'k'})
	result = updated.(Model)
	if result.selected != 1 {
		t.Errorf("expected selected=1 after k, got %d", result.selected)
	}

	// Jump to last
	updated, _ = result.Update(tea.KeyPressMsg{Code: 'G'})
	result = updated.(Model)
	if result.selected != 2 {
		t.Errorf("expected selected=2 after G, got %d", result.selected)
	}

	// Jump to first
	updated, _ = result.Update(tea.KeyPressMsg{Code: 'g'})
	result = updated.(Model)
	if result.selected != 0 {
		t.Errorf("expected selected=0 after g, got %d", result.selected)
	}
}

func TestUpdate_SelectionClampOnShrink(t *testing.T) {
	m := testModel()
	m.selected = 2
	m.sessions = []daemon.SessionInfo{{ID: "1"}, {ID: "2"}, {ID: "3"}}

	// Sessions shrink to 1
	updated, _ := m.Update(sessionsMsg{sessions: []daemon.SessionInfo{{ID: "1"}}})
	result := updated.(Model)

	if result.selected != 0 {
		t.Errorf("expected selected clamped to 0, got %d", result.selected)
	}
}

func TestUpdate_ReservedKeysShowToast(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{{ID: "1", Name: "s1"}}

	// Enter key (reserved)
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.toast != "Coming in next update" {
		t.Errorf("expected toast for Enter key, got %q", result.toast)
	}
}

func TestUpdate_HelpSwallowsKeys(t *testing.T) {
	m := testModel()
	m.showHelp = true
	m.sessions = []daemon.SessionInfo{{ID: "1"}, {ID: "2"}}
	m.selected = 0

	// j key should be swallowed when help is open
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := updated.(Model)
	if result.selected != 0 {
		t.Errorf("expected selected=0 (j swallowed), got %d", result.selected)
	}
}
