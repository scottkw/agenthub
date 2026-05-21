package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// modelWithRemoteSession constructs a testModel with a single remote
// session pre-loaded and selected. Shared helper for the Phase 122
// FilesOpen-remote tests.
func modelWithRemoteSession(t *testing.T, sid, url, cap string) Model {
	t.Helper()
	m := testModel()
	m.remoteSessions = []ListRemoteGroup{
		{Hostname: "peer-host", Sessions: []RemoteSessionEntry{
			{
				ID: sid, Name: "rem-name", CLIType: "claude", Status: "running",
				Hostname: "peer-host",
				URL:      url,
			},
		}},
	}
	m.rebuildUnifiedList()
	for i, e := range m.unifiedList {
		if e.kind == entryRemote {
			m.selected = i
			break
		}
	}
	if cap != "" {
		m.remoteCapStore = map[string]remoteCapEntry{
			sid: {baseURL: "https://peer.example:9443", capToken: cap},
		}
	}
	return m
}

// TestFilesOpen_RemoteCached_FastPath_NoModal — Behavior 1.
// Remote entry WITH a cached cap → opens tabFiles directly against a
// RemoteFilesClient. No modal opens.
func TestFilesOpen_RemoteCached_FastPath_NoModal(t *testing.T) {
	m := modelWithRemoteSession(t, "r1", "https://peer.example:9443/sessions/r1", "cached-cap-1")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'f'})
	r := updated.(Model)
	if r.modal != modalNone {
		t.Errorf("expected no modal on cached-cap remote 'f', got modal=%v", r.modal)
	}
	if r.activeTabID() != tabFiles {
		t.Errorf("expected activeTabID=tabFiles, got %v", r.activeTabID())
	}
	if !r.files.isRemoteClient() {
		t.Error("expected filesModel.client to be *RemoteFilesClient")
	}
	if r.files.sessionID != "r1" {
		t.Errorf("expected files.sessionID=r1, got %q", r.files.sessionID)
	}
	if cmd == nil {
		t.Error("expected loadDirCmd dispatched on cached-cap fast path")
	}
}

// TestFilesOpen_RemoteNoCached_OpensJoinCodePrompt — Behavior 2.
// Remote entry WITHOUT cached cap → opens modalJoinCodePrompt with session
// context.
func TestFilesOpen_RemoteNoCached_OpensJoinCodePrompt(t *testing.T) {
	m := modelWithRemoteSession(t, "r2", "https://peer.example:9443/sessions/r2", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	r := updated.(Model)
	if r.modal != modalJoinCodePrompt {
		t.Errorf("expected modalJoinCodePrompt, got %v", r.modal)
	}
	if r.joinCodePrompt.sessionID != "r2" {
		t.Errorf("expected joinCodePrompt.sessionID=r2, got %q", r.joinCodePrompt.sessionID)
	}
	if r.joinCodePrompt.sessionName != "rem-name" {
		t.Errorf("expected joinCodePrompt.sessionName=rem-name, got %q", r.joinCodePrompt.sessionName)
	}
	if r.joinCodePrompt.remoteBaseURL != "https://peer.example:9443" {
		t.Errorf("expected baseURL stripped, got %q", r.joinCodePrompt.remoteBaseURL)
	}
}

// TestJoinCodePromptModal_TypingRoutesToInput — Behavior 3.
// In modalJoinCodePrompt mode, typing keys routes to the prompt's input,
// not to the underlying tab's handleFilesKey.
func TestJoinCodePromptModal_TypingRoutesToInput(t *testing.T) {
	m := modelWithRemoteSession(t, "r3", "https://peer.example:9443/sessions/r3", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)

	for _, r := range "WXY" {
		updated, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(Model)
	}
	if m.joinCodePrompt.input.Value() != "WXY" {
		t.Errorf("expected input='WXY', got %q", m.joinCodePrompt.input.Value())
	}
	if m.activeTabID() == tabFiles {
		t.Error("must NOT open tabFiles while modal is up")
	}
}

// TestJoinCodePromptModal_EscClosesNoTab — Behavior 4.
func TestJoinCodePromptModal_EscClosesNoTab(t *testing.T) {
	m := modelWithRemoteSession(t, "r4", "https://peer.example:9443/sessions/r4", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	// Pressing Esc should dispatch a cancelJoinCodeMsg.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Esc must dispatch a cmd that emits cancelJoinCodeMsg")
	}
	// Run the cancel cmd and feed the result back through Update.
	cancelMsg := cmd()
	updated, _ = m.Update(cancelMsg)
	m = updated.(Model)
	if m.modal != modalNone {
		t.Errorf("expected modalNone after Esc + cancel cmd, got %v", m.modal)
	}
	if m.activeTabID() == tabFiles {
		t.Error("Esc must NOT open tabFiles")
	}
}

// TestJoinCodePromptModal_SuccessOpensTabAndCachesCap — Behavior 5.
// On joinCodeResultMsg success: cap stored in remoteCapStore, modal closed,
// tabFiles opens with a RemoteFilesClient, loadDirCmd dispatched.
func TestJoinCodePromptModal_SuccessOpensTabAndCachesCap(t *testing.T) {
	m := modelWithRemoteSession(t, "r5", "https://peer.example:9443/sessions/r5", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	if m.modal != modalJoinCodePrompt {
		t.Fatalf("setup: expected modalJoinCodePrompt, got %v", m.modal)
	}
	const wantCap = "TOK-r5"
	updated, cmd := m.Update(joinCodeResultMsg{sessionID: "r5", cap: wantCap})
	r := updated.(Model)
	if r.modal != modalNone {
		t.Errorf("expected modalNone after success, got %v", r.modal)
	}
	if r.activeTabID() != tabFiles {
		t.Errorf("expected tabFiles open, got %v", r.activeTabID())
	}
	if !r.files.isRemoteClient() {
		t.Error("expected filesModel.client to be *RemoteFilesClient after success")
	}
	got, ok := r.remoteCapStore["r5"]
	if !ok {
		t.Fatal("expected remoteCapStore[r5] populated")
	}
	if got.capToken != wantCap {
		t.Errorf("expected cached capToken=%q, got %q", wantCap, got.capToken)
	}
	if got.baseURL != "https://peer.example:9443" {
		t.Errorf("expected cached baseURL=https://peer.example:9443, got %q", got.baseURL)
	}
	if cmd == nil {
		t.Error("expected loadDirCmd dispatched after success")
	}
}

// TestJoinCodePromptModal_FailurePreservesModal — Behavior 6.
// On joinCodeResultMsg{err: ...}: prompt transitions to error state,
// modal stays open so the user can retry.
func TestJoinCodePromptModal_FailurePreservesModal(t *testing.T) {
	m := modelWithRemoteSession(t, "r6", "https://peer.example:9443/sessions/r6", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	updated, _ = m.Update(joinCodeResultMsg{sessionID: "r6", err: errors.New("join exchange: expired")})
	r := updated.(Model)
	if r.modal != modalJoinCodePrompt {
		t.Errorf("expected modal to STAY open on error, got %v", r.modal)
	}
	if r.joinCodePrompt.state != joinCodePromptError {
		t.Errorf("expected joinCodePromptError state, got %v", r.joinCodePrompt.state)
	}
	if !strings.Contains(r.joinCodePrompt.errMsg, "Code expired") {
		t.Errorf("expected friendly 'Code expired' message, got %q", r.joinCodePrompt.errMsg)
	}
}

// TestRemote401_ForgetsCapAndShowsUnsharedMessage — Behavior 7.
// When tabFiles is open against a remote session and the first loadDirCmd
// returns an error containing "401": status line shows the
// "Remote session must be web-shared…" copy AND remoteCapStore[sid] is
// DELETED so the next 'f' re-prompts.
func TestRemote401_ForgetsCapAndShowsUnsharedMessage(t *testing.T) {
	m := modelWithRemoteSession(t, "r7", "https://peer.example:9443/sessions/r7", "stale-cap")
	// Open the tab via cached-cap fast path.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	if m.activeTabID() != tabFiles {
		t.Fatalf("setup: expected tabFiles open, got %v", m.activeTabID())
	}
	if _, ok := m.remoteCapStore["r7"]; !ok {
		t.Fatal("setup: cap must be in store before 401")
	}

	// Simulate the loadDir round-trip returning a 401 from the remote.
	updated, _ = m.Update(filesListMsg{
		sessionID:  "r7",
		generation: m.files.generation,
		relPath:    ".",
		err:        errors.New("remote files list: 401 unauthorized"),
	})
	r := updated.(Model)
	if r.files.err == nil {
		t.Fatal("expected files.err set after 401")
	}
	if r.files.err.Error() != remoteUnsharedMsg {
		t.Errorf("expected unshared-message copy, got %q", r.files.err.Error())
	}
	if _, ok := r.remoteCapStore["r7"]; ok {
		t.Error("expected remoteCapStore[r7] DELETED after 401")
	}
}

// TestUpdateGo_NoV34Toast — Behavior 8.
// Source-grep gate: the v3.4 "File browser not available for remote
// sessions" toast must be GONE from update.go.
func TestUpdateGo_NoV34Toast(t *testing.T) {
	data, err := os.ReadFile("update.go")
	if err != nil {
		t.Fatalf("read update.go: %v", err)
	}
	if strings.Contains(string(data), "File browser not available for remote sessions") {
		t.Error("v3.4 remote-refusal toast must be REMOVED from internal/tui/update.go")
	}
}

// TestRemoteBaseURLFromSessionURL covers the URL-stripping helper.
func TestRemoteBaseURLFromSessionURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://peer.example:9443/sessions/r1", "https://peer.example:9443"},
		{"https://peer.example/sessions/r1?x=y", "https://peer.example"},
		{"", ""},
		{"not a url", ""},
	}
	for _, tc := range cases {
		if got := remoteBaseURLFromSessionURL(tc.in); got != tc.want {
			t.Errorf("remoteBaseURLFromSessionURL(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestJoinCodeResultMsg_StalenessGuard verifies a joinCodeResultMsg whose
// sessionID does NOT match the currently-open prompt is silently dropped.
// Defense against late-arriving result after the user dismissed the modal
// and triggered a different remote session.
func TestJoinCodeResultMsg_StalenessGuard(t *testing.T) {
	m := modelWithRemoteSession(t, "r8", "https://peer.example:9443/sessions/r8", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	// Late-arriving message for a DIFFERENT session — must be ignored.
	updated, _ = m.Update(joinCodeResultMsg{sessionID: "different-sid", cap: "abc"})
	r := updated.(Model)
	if r.modal != modalJoinCodePrompt {
		t.Errorf("stale result must NOT close modal, got modal=%v", r.modal)
	}
	if _, ok := r.remoteCapStore["different-sid"]; ok {
		t.Error("stale result must NOT cache cap for the unrelated sid")
	}
}
