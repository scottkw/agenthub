package tui

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/textinput"
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
	// No sessions — attach shows "Session not available"
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.toast != "Session not available" {
		t.Errorf("expected 'Session not available' toast for Enter with no sessions, got %q", result.toast)
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

func TestUpdate_KeyReassignment(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "test", CLI: "claude", Status: "running"},
	}
	m.selected = 0

	// r should enter rename mode, not refresh
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	result := updated.(Model)
	if !result.editing {
		t.Error("expected editing=true after r key (rename)")
	}
	if result.editSessionID != "1" {
		t.Errorf("expected editSessionID='1', got %q", result.editSessionID)
	}

	// R (shift+r) should trigger refresh (returns a cmd)
	m2 := testModel()
	_, cmd := m2.Update(tea.KeyPressMsg{Code: 'R'})
	if cmd == nil {
		t.Error("expected non-nil cmd from R key (refresh)")
	}
}

func TestUpdate_KillConfirmOpen(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "test", CLI: "claude", Status: "running"},
	}
	m.selected = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	result := updated.(Model)
	if result.modal != modalKillConfirm {
		t.Errorf("expected modal=modalKillConfirm, got %d", result.modal)
	}
	if result.killTarget == nil {
		t.Fatal("expected killTarget to be set")
	}
	if result.killTarget.ID != "1" {
		t.Errorf("expected killTarget.ID='1', got %q", result.killTarget.ID)
	}
	if result.killFocusYes {
		t.Error("expected killFocusYes=false (default No)")
	}
}

func TestKill_QuickYes(t *testing.T) {
	m := testModel()
	m.modal = modalKillConfirm
	m.killTarget = &daemon.SessionInfo{ID: "1", Name: "test"}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
	result := updated.(Model)
	if result.modal != modalNone {
		t.Error("expected modal closed after y")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (killSession)")
	}
}

func TestKill_Cancel(t *testing.T) {
	m := testModel()
	m.modal = modalKillConfirm
	m.killTarget = &daemon.SessionInfo{ID: "1", Name: "test"}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	result := updated.(Model)
	if result.modal != modalNone {
		t.Error("expected modal closed after n")
	}
	if result.killTarget != nil {
		t.Error("expected killTarget=nil after cancel")
	}
}

func TestKill_ToggleFocus(t *testing.T) {
	m := testModel()
	m.modal = modalKillConfirm
	m.killTarget = &daemon.SessionInfo{ID: "1", Name: "test"}
	m.killFocusYes = false

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	result := updated.(Model)
	if !result.killFocusYes {
		t.Error("expected killFocusYes=true after right arrow")
	}
}

func TestRename_SubmitAndCancel(t *testing.T) {
	m := testModel()
	m.editing = true
	m.editSessionID = "1"
	m.editOriginal = "old-name"
	m.editInput = textinput.New()
	m.editInput.SetValue("new-name")

	// Submit
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.editing {
		t.Error("expected editing=false after enter")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (renameSession)")
	}

	// Cancel
	m2 := testModel()
	m2.editing = true
	m2.editOriginal = "old-name"
	m2.editInput = textinput.New()
	m2.editInput.SetValue("changed")

	updated2, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result2 := updated2.(Model)
	if result2.editing {
		t.Error("expected editing=false after esc")
	}
}

func TestRename_EmptyRejected(t *testing.T) {
	m := testModel()
	m.editing = true
	m.editSessionID = "1"
	m.editOriginal = "old-name"
	m.editInput = textinput.New()
	m.editInput.SetValue("")

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if !result.editing {
		t.Error("expected editing=true (empty name rejected)")
	}
	if result.toast != "Name cannot be empty" {
		t.Errorf("expected empty name toast, got %q", result.toast)
	}
}

func TestRename_SameNameNoOp(t *testing.T) {
	m := testModel()
	m.editing = true
	m.editSessionID = "1"
	m.editOriginal = "same-name"
	m.editInput = textinput.New()
	m.editInput.SetValue("same-name")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.editing {
		t.Error("expected editing=false")
	}
	if cmd != nil {
		t.Error("expected nil cmd for same-name (no API call)")
	}
}

func TestUpdate_KillSessionMsg(t *testing.T) {
	m := testModel()

	// Success
	updated, cmd := m.Update(killSessionMsg{err: nil})
	result := updated.(Model)
	if result.toast != "Session killed" {
		t.Errorf("expected 'Session killed' toast, got %q", result.toast)
	}
	if result.toastKind != toastSuccess {
		t.Errorf("expected toastSuccess, got %d", result.toastKind)
	}
	if cmd == nil {
		t.Error("expected refresh cmd after kill")
	}

	// Error
	updated2, _ := m.Update(killSessionMsg{err: fmt.Errorf("not found")})
	result2 := updated2.(Model)
	if result2.toast != "Kill failed: not found" {
		t.Errorf("expected error toast, got %q", result2.toast)
	}
	if result2.toastKind != toastError {
		t.Errorf("expected toastError, got %d", result2.toastKind)
	}
}

func TestUpdate_RenameSessionMsg(t *testing.T) {
	m := testModel()

	// Success — refresh cmd returned
	updated, cmd := m.Update(renameSessionMsg{err: nil})
	result := updated.(Model)
	if cmd == nil {
		t.Error("expected refresh cmd after rename")
	}
	_ = result

	// Error
	updated2, _ := m.Update(renameSessionMsg{err: fmt.Errorf("daemon error")})
	result2 := updated2.(Model)
	if result2.toast != "Rename failed: daemon error" {
		t.Errorf("expected error toast, got %q", result2.toast)
	}
}

func TestUpdate_NewSessionModalOpen(t *testing.T) {
	m := testModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	result := updated.(Model)
	if result.modal != modalNewSession {
		t.Errorf("expected modal=modalNewSession after n key, got %d", result.modal)
	}
}

func TestUpdate_RenameStart(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "my-session", CLI: "claude", Status: "running"},
	}
	m.selected = 0

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r'})
	result := updated.(Model)
	if !result.editing {
		t.Error("expected editing=true after r")
	}
	if result.editSessionID != "1" {
		t.Errorf("expected editSessionID='1', got %q", result.editSessionID)
	}
	if result.editOriginal != "my-session" {
		t.Errorf("expected editOriginal='my-session', got %q", result.editOriginal)
	}
	if result.editInput.Value() != "my-session" {
		t.Errorf("expected editInput prefilled with 'my-session', got %q", result.editInput.Value())
	}
	if cmd == nil {
		t.Error("expected Focus cmd from textinput")
	}
}

func TestRename_NavigationSuppressed(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{{ID: "1"}, {ID: "2"}}
	m.selected = 0
	m.editing = true
	m.editInput = textinput.New()

	// j key should be captured by textinput, not move selection
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := updated.(Model)
	if result.selected != 0 {
		t.Errorf("expected selected=0 (j captured by rename), got %d", result.selected)
	}
}
