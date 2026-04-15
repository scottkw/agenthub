package tui

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/pty"
)

func testModel() Model {
	m := newModel(nil, nil) // nil client, nil fetchRemoteFn -- tests don't make HTTP calls
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

	// Phase 78: quit is now Q (Shift+Q), not lowercase q
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'Q'})
	if cmd == nil {
		t.Fatal("expected quit command from Q, got nil")
	}
	// tea.Quit returns a special command -- executing it produces tea.QuitMsg
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg from Q, got %T", msg)
	}
}

func TestUpdate_QuitKeyReassignment(t *testing.T) {
	m := testModel()
	// q should NOT quit anymore -- it's the QR trigger now
	// With no sessions, q should be a no-op (empty list guard)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("q should not quit -- it should trigger QR now")
		}
	}
	// Q (Shift) should still quit
	_, cmd2 := m.Update(tea.KeyPressMsg{Code: 'Q'})
	if cmd2 == nil {
		t.Fatal("Q should produce a quit command")
	}
	msg2 := cmd2()
	if _, ok := msg2.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg from Q, got %T", msg2)
	}
}

func TestUpdate_QROpen(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "s1", Name: "test-session", CLI: "claude", Status: "running"},
	}
	m.webStatus = daemon.WebServerStatusResponse{Running: true, URL: "https://test.ts.net"}
	m.width = 80
	m.height = 30
	m.rebuildUnifiedList()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	result := updated.(Model)
	if result.qrSession == nil {
		t.Fatal("expected qrSession to be set after q on web-served session")
	}
	if result.qrContent == "" {
		t.Error("expected non-empty qrContent")
	}
	if result.qrURL != "https://test.ts.net/sessions/s1" {
		t.Errorf("expected URL 'https://test.ts.net/sessions/s1', got %q", result.qrURL)
	}
}

func TestUpdate_QRClose(t *testing.T) {
	m := testModel()
	m.qrSession = &sessionRef{ID: "s1", Name: "test", URL: "https://test.ts.net/sessions/s1"}
	m.qrContent = "qr-data"
	m.qrURL = "https://test.ts.net/sessions/s1"

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := updated.(Model)
	if result.qrSession != nil {
		t.Error("expected qrSession=nil after Esc")
	}
	if result.qrContent != "" {
		t.Error("expected qrContent cleared after Esc")
	}
}

func TestUpdate_QRNoURL(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "s1", Name: "test-session", CLI: "claude", Status: "running"},
	}
	m.webStatus = daemon.WebServerStatusResponse{Running: false}
	m.width = 80
	m.height = 30
	m.rebuildUnifiedList()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	result := updated.(Model)
	if result.qrSession != nil {
		t.Error("expected qrSession=nil when web server not running")
	}
	if result.toast != "Web serving not enabled for this session" {
		t.Errorf("expected web-not-enabled toast, got %q", result.toast)
	}
}

func TestUpdate_QRTerminalTooSmall(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "s1", Name: "test", CLI: "claude", Status: "running"},
	}
	m.webStatus = daemon.WebServerStatusResponse{Running: true, URL: "https://test.ts.net"}
	m.width = 50  // below 55 minimum
	m.height = 20 // below 25 minimum
	m.rebuildUnifiedList()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	result := updated.(Model)
	if result.qrSession != nil {
		t.Error("expected qrSession=nil when terminal too small")
	}
	if result.toast != "Terminal too small to display QR code" {
		t.Errorf("expected too-small toast, got %q", result.toast)
	}
}

func TestUpdate_QRSwallowsKeys(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{{ID: "1", Name: "s1"}, {ID: "2", Name: "s2"}}
	m.rebuildUnifiedList()
	m.selected = 0
	m.qrSession = &sessionRef{ID: "s1", Name: "test", URL: "https://test.ts.net/sessions/s1"}
	m.qrContent = "qr-data"
	m.qrURL = "https://test.ts.net/sessions/s1"

	// j key should be swallowed while QR overlay is open
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := updated.(Model)
	if result.selected != 0 {
		t.Errorf("expected selected=0 (j swallowed by QR overlay), got %d", result.selected)
	}
	if result.qrSession == nil {
		t.Error("expected QR overlay to remain open after j")
	}
}

func TestUpdate_QRQuitFromOverlay(t *testing.T) {
	m := testModel()
	m.qrSession = &sessionRef{ID: "s1", Name: "test", URL: "https://test.ts.net/sessions/s1"}
	m.qrContent = "qr-data"
	m.qrURL = "https://test.ts.net/sessions/s1"

	// Q should quit even from QR overlay
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'Q'})
	if cmd == nil {
		t.Fatal("expected quit command from Q while QR overlay is open")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg from Q in QR overlay, got %T", msg)
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
	m.rebuildUnifiedList()
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
	m.rebuildUnifiedList()
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
	m.rebuildUnifiedList()
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
	m.rebuildUnifiedList()
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

func TestModal_FocusCycle(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.focusedField = 0
	m.dirInput = textinput.New()
	m.argsInput = textinput.New()

	// Tab: agent(0) -> dir(1)
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	result := updated.(Model)
	if result.focusedField != 1 {
		t.Errorf("expected focusedField=1 after Tab, got %d", result.focusedField)
	}

	// Tab: dir(1) -> args(2)
	updated, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	result = updated.(Model)
	if result.focusedField != 2 {
		t.Errorf("expected focusedField=2 after Tab, got %d", result.focusedField)
	}

	// Tab: args(2) -> agent(0) (wraps)
	updated, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	result = updated.(Model)
	if result.focusedField != 0 {
		t.Errorf("expected focusedField=0 after Tab wrap, got %d", result.focusedField)
	}
}

func TestModal_AgentCycle(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.focusedField = 0
	m.detectedCLIs = []pty.DetectedCLI{
		{Name: "claude", DisplayName: "Claude Code", Path: "/usr/bin/claude"},
		{Name: "opencode", DisplayName: "OpenCode", Path: "/usr/bin/opencode"},
	}
	m.agentIdx = 0
	m.dirInput = textinput.New()
	m.argsInput = textinput.New()

	// Right arrow: claude(0) -> opencode(1)
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	result := updated.(Model)
	if result.agentIdx != 1 {
		t.Errorf("expected agentIdx=1 after Right, got %d", result.agentIdx)
	}

	// Right arrow: opencode(1) -> claude(0) (wraps)
	updated, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	result = updated.(Model)
	if result.agentIdx != 0 {
		t.Errorf("expected agentIdx=0 after Right wrap, got %d", result.agentIdx)
	}

	// Left arrow: claude(0) -> opencode(1) (wraps backward)
	updated, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	result = updated.(Model)
	if result.agentIdx != 1 {
		t.Errorf("expected agentIdx=1 after Left wrap, got %d", result.agentIdx)
	}
}

func TestModal_SubmitValidation(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.focusedField = 1
	m.detectedCLIs = []pty.DetectedCLI{
		{Name: "claude", DisplayName: "Claude Code", Path: "/usr/bin/claude"},
	}
	m.agentIdx = 0
	m.dirInput = textinput.New()
	m.dirInput.SetValue("") // empty directory
	m.argsInput = textinput.New()

	// Submit with empty directory
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.toast != "Directory is required" {
		t.Errorf("expected 'Directory is required' toast, got %q", result.toast)
	}
	if result.modal != modalNewSession {
		t.Error("modal should stay open on validation failure")
	}
}

func TestModal_SubmitNoAgents(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.detectedCLIs = nil // no agents
	m.dirInput = textinput.New()
	m.dirInput.SetValue("/some/dir")
	m.argsInput = textinput.New()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.toast != "Agent is required" {
		t.Errorf("expected 'Agent is required' toast, got %q", result.toast)
	}
}

func TestModal_SubmitSuccess(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.detectedCLIs = []pty.DetectedCLI{
		{Name: "claude", DisplayName: "Claude Code", Path: "/usr/bin/claude"},
	}
	m.agentIdx = 0
	m.dirInput = textinput.New()
	m.dirInput.SetValue("/Users/ken/dev/project")
	m.argsInput = textinput.New()
	m.argsInput.SetValue("--model opus")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.modal != modalNone {
		t.Error("modal should close on successful submit")
	}
	if result.toast != "Creating session..." {
		t.Errorf("expected 'Creating session...' toast, got %q", result.toast)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (createSession)")
	}
}

func TestModal_Cancel(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.dirInput = textinput.New()
	m.argsInput = textinput.New()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := updated.(Model)
	if result.modal != modalNone {
		t.Error("modal should close on Esc")
	}
}

func TestUpdate_CreateSessionMsg(t *testing.T) {
	m := testModel()

	// Success
	updated, cmd := m.Update(createSessionMsg{id: "new-id", err: nil})
	result := updated.(Model)
	if result.toast != "Session created" {
		t.Errorf("expected 'Session created' toast, got %q", result.toast)
	}
	if result.toastKind != toastSuccess {
		t.Errorf("expected toastSuccess, got %d", result.toastKind)
	}
	if cmd == nil {
		t.Error("expected refresh cmd after create")
	}

	// Error
	updated2, _ := m.Update(createSessionMsg{err: fmt.Errorf("port in use")})
	result2 := updated2.(Model)
	if result2.toast != "Create failed: port in use" {
		t.Errorf("expected error toast, got %q", result2.toast)
	}
}

// --- Phase 78: Remote Sessions Tests ---

func testRemoteGroups() []ListRemoteGroup {
	return []ListRemoteGroup{
		{
			Hostname: "laptop-work",
			Sessions: []RemoteSessionEntry{
				{ID: "r1", Name: "their-proj", CLIType: "claude", Status: "running", Hostname: "laptop-work", FQDN: "laptop-work.tail.ts.net", URL: "https://laptop-work.tail.ts.net:7443/sessions/r1"},
				{ID: "r2", Name: "qa-review", CLIType: "opencode", Status: "idle", Hostname: "laptop-work", FQDN: "laptop-work.tail.ts.net", URL: "https://laptop-work.tail.ts.net:7443/sessions/r2"},
			},
		},
	}
}

func TestUpdate_RemoteSessionsMsg(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "local-session", CLI: "claude", Status: "running"},
	}
	m.rebuildUnifiedList()

	// Send a remoteSessionsMsg with 1 group of 2 remote sessions
	groups := testRemoteGroups()
	updated, _ := m.Update(remoteSessionsMsg{groups: groups})
	result := updated.(Model)

	// Expect: 1 local + 1 divider + 2 remote = 4 entries
	if len(result.unifiedList) != 4 {
		t.Errorf("expected 4 unified list entries (1 local + 1 divider + 2 remote), got %d", len(result.unifiedList))
	}
	if result.unifiedList[0].kind != entryLocal {
		t.Errorf("expected entry 0 to be entryLocal, got %d", result.unifiedList[0].kind)
	}
	if result.unifiedList[1].kind != entryDivider {
		t.Errorf("expected entry 1 to be entryDivider, got %d", result.unifiedList[1].kind)
	}
	if result.unifiedList[2].kind != entryRemote {
		t.Errorf("expected entry 2 to be entryRemote, got %d", result.unifiedList[2].kind)
	}
	if result.unifiedList[3].kind != entryRemote {
		t.Errorf("expected entry 3 to be entryRemote, got %d", result.unifiedList[3].kind)
	}
	if len(result.remoteSessions) != 1 {
		t.Errorf("expected 1 remote group, got %d", len(result.remoteSessions))
	}
}

func TestUpdate_NavigationSkipsDividers(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "local", CLI: "claude", Status: "running"},
	}
	m.remoteSessions = testRemoteGroups()
	m.rebuildUnifiedList()
	// List is: [local(0), divider(1), remote(2), remote(3)]
	m.selected = 0

	// Press j: should skip divider at index 1 and land on remote at index 2
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := updated.(Model)
	if result.selected != 2 {
		t.Errorf("expected selected=2 (skip divider at 1), got %d", result.selected)
	}

	// Press k: should skip divider at index 1 and land back on local at index 0
	updated2, _ := result.Update(tea.KeyPressMsg{Code: 'k'})
	result2 := updated2.(Model)
	if result2.selected != 0 {
		t.Errorf("expected selected=0 (skip divider at 1 going up), got %d", result2.selected)
	}
}

func TestUpdate_KillRemoteBlocked(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "local", CLI: "claude", Status: "running"},
	}
	m.remoteSessions = testRemoteGroups()
	m.rebuildUnifiedList()
	// List: [local(0), divider(1), remote(2), remote(3)]
	m.selected = 2 // remote session

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	result := updated.(Model)

	if result.toast != "Cannot kill remote session" {
		t.Errorf("expected 'Cannot kill remote session' toast, got %q", result.toast)
	}
	if result.modal != modalNone {
		t.Errorf("expected modal=modalNone (no kill dialog for remote), got %d", result.modal)
	}
}

func TestUpdate_RenameRemoteBlocked(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "local", CLI: "claude", Status: "running"},
	}
	m.remoteSessions = testRemoteGroups()
	m.rebuildUnifiedList()
	// List: [local(0), divider(1), remote(2), remote(3)]
	m.selected = 2 // remote session

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r'})
	result := updated.(Model)

	if result.toast != "Cannot rename remote session" {
		t.Errorf("expected 'Cannot rename remote session' toast, got %q", result.toast)
	}
	if result.editing {
		t.Error("expected editing=false (rename blocked on remote session)")
	}
}

func TestUpdate_SelectionRestoredAfterRebuild(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "local", CLI: "claude", Status: "running"},
	}
	m.remoteSessions = testRemoteGroups()
	m.rebuildUnifiedList()
	// List: [local(0), divider(1), remote-r1(2), remote-r2(3)]
	m.selected = 2 // select remote r1

	// Send a new sessionsMsg with the same local sessions -- remote stays the same
	updated, _ := m.Update(sessionsMsg{sessions: []daemon.SessionInfo{
		{ID: "1", Name: "local", CLI: "claude", Status: "running"},
	}})
	result := updated.(Model)

	// Selection should be restored to the remote session at index 2 (identity: r1:laptop-work)
	if result.selected != 2 {
		t.Errorf("expected selected=2 (remote r1 restored), got %d", result.selected)
	}
}

func TestUpdate_UnifiedListEmpty(t *testing.T) {
	m := testModel()
	// No sessions, no remote groups
	m.rebuildUnifiedList()

	if len(m.unifiedList) != 0 {
		t.Errorf("expected empty unifiedList, got %d entries", len(m.unifiedList))
	}

	// Navigation keys must not panic with empty list
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := updated.(Model)
	if result.selected != 0 {
		t.Errorf("expected selected=0 with empty list, got %d", result.selected)
	}

	updated2, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	result2 := updated2.(Model)
	if result2.selected != 0 {
		t.Errorf("expected selected=0 with empty list, got %d", result2.selected)
	}
}
