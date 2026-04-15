package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

// TestTUIRemoteAndQR_FullFlow exercises the complete lifecycle:
// local+remote session loading, unified list count, view content,
// divider-skip navigation, QR open on remote, QR close on Esc,
// QR open on local, QR close on q, and Q quit.
func TestTUIRemoteAndQR_FullFlow(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 30
	m.webStatus = daemon.WebServerStatusResponse{Running: true, URL: "https://test.ts.net"}

	// Step 1: Load local sessions. s1 has WebEnabled:true because step 6 opens QR on it;
	// s2 stays unserved which also guards that QR is a per-session concern.
	localSessions := []daemon.SessionInfo{
		{ID: "s1", Name: "my-session", CLI: "claude", Hostname: "macbook-pro", Status: "running", WebEnabled: true},
		{ID: "s2", Name: "docs-check", CLI: "opencode", Hostname: "macbook-pro", Status: "idle"},
	}
	updated, _ := m.Update(sessionsMsg{sessions: localSessions})
	m = updated.(Model)

	// Step 2: Load remote sessions
	remoteGroups := []ListRemoteGroup{
		{
			Hostname: "laptop-work",
			Sessions: []RemoteSessionEntry{
				{ID: "r1", Name: "their-proj", CLIType: "claude", Status: "running",
					Hostname: "laptop-work", FQDN: "laptop-work.tail.ts.net",
					URL: "https://laptop-work.tail.ts.net:7443/sessions/r1"},
				{ID: "r2", Name: "qa-review", CLIType: "claude", Status: "idle",
					Hostname: "laptop-work", FQDN: "laptop-work.tail.ts.net",
					URL: "https://laptop-work.tail.ts.net:7443/sessions/r2"},
			},
		},
	}
	updated, _ = m.Update(remoteSessionsMsg{groups: remoteGroups})
	m = updated.(Model)

	// Verify unified list: 2 local + 1 divider + 2 remote = 5
	if len(m.unifiedList) != 5 {
		t.Fatalf("expected 5 unified entries, got %d", len(m.unifiedList))
	}
	if m.unifiedList[2].kind != entryDivider {
		t.Errorf("expected entryDivider at index 2, got kind %d", m.unifiedList[2].kind)
	}

	// Verify view output
	v := m.View()
	content := v.Content
	for _, want := range []string{"2 local, 2 remote", "my-session", "docs-check", "Remote:", "laptop-work", "their-proj", "qa-review"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q", want)
		}
	}

	// Step 3: Navigate to first remote session (j, j should skip divider)
	m.selected = 0
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j'}) // 0 -> 1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'j'}) // 1 -> 3 (skip divider at 2)
	m = updated.(Model)
	if m.selected != 3 {
		t.Errorf("expected selected=3 (first remote), got %d", m.selected)
	}

	// Step 4: Press q on remote session -- QR overlay should open
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q'})
	m = updated.(Model)
	if m.qrSession == nil {
		t.Fatal("expected QR overlay to open on remote session")
	}
	if m.qrURL != "https://laptop-work.tail.ts.net:7443/sessions/r1" {
		t.Errorf("expected remote URL, got %q", m.qrURL)
	}
	if m.qrContent == "" {
		t.Error("expected non-empty QR content")
	}

	// Verify overlay view contains session name in the title and the remote URL domain.
	// Lipgloss may word-wrap the URL at hyphens across lines, so we check reliable substrings.
	// m.qrURL already confirmed the full URL above.
	overlay := m.renderQROverlay()
	if !strings.Contains(overlay, "their-proj") {
		t.Error("QR overlay missing session name in title")
	}
	// "tail.ts.net" appears on the same line as "work" after lipgloss wraps at "laptop-"
	if !strings.Contains(overlay, "tail.ts.net") {
		t.Error("QR overlay missing URL domain portion")
	}

	// Step 5: Press Esc to close overlay
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.qrSession != nil {
		t.Error("expected QR overlay closed after Esc")
	}

	// Step 6: Navigate to first local session and open QR
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'g'}) // jump to top
	m = updated.(Model)
	if m.selected != 0 {
		t.Errorf("expected selected=0 after g, got %d", m.selected)
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q'})
	m = updated.(Model)
	if m.qrSession == nil {
		t.Fatal("expected QR overlay to open on local session")
	}
	if !strings.Contains(m.qrURL, "test.ts.net/sessions/s1") {
		t.Errorf("expected local URL containing 'test.ts.net/sessions/s1', got %q", m.qrURL)
	}

	// Step 7: Press q to close overlay (q also closes per UI-SPEC)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q'})
	m = updated.(Model)
	if m.qrSession != nil {
		t.Error("expected QR overlay closed after q while in overlay")
	}

	// Step 8: Verify Q quits
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'Q'})
	if cmd == nil {
		t.Fatal("expected quit command from Q")
	}
}

// TestTUIRemoteAndQR_BlockedOperations tests that kill/rename on remote sessions
// are blocked, and QR on non-web-served session shows toast.
func TestTUIRemoteAndQR_BlockedOperations(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 30
	m.sessions = []daemon.SessionInfo{
		{ID: "s1", Name: "local-session", CLI: "claude", Status: "running"},
	}
	m.remoteSessions = []ListRemoteGroup{
		{Hostname: "peer1", Sessions: []RemoteSessionEntry{
			{ID: "r1", Name: "remote-session", CLIType: "claude", Status: "running",
				Hostname: "peer1", URL: "https://peer1.tail.ts.net:7443/sessions/r1"},
		}},
	}
	m.rebuildUnifiedList()

	// Navigate to remote session (index 2, after local at 0 and divider at 1)
	m.selected = 2
	if m.unifiedList[2].kind != entryRemote {
		t.Fatalf("expected remote at index 2, got kind %d", m.unifiedList[2].kind)
	}

	// Kill blocked
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	result := updated.(Model)
	if result.toast != "Cannot kill remote session" {
		t.Errorf("expected kill-blocked toast, got %q", result.toast)
	}
	if result.modal != modalNone {
		t.Error("expected no modal opened for remote kill")
	}

	// Rename blocked
	updated, _ = result.Update(tea.KeyPressMsg{Code: 'r'})
	result = updated.(Model)
	if result.toast != "Cannot rename remote session" {
		t.Errorf("expected rename-blocked toast, got %q", result.toast)
	}
	if result.editing {
		t.Error("expected editing=false for remote rename")
	}

	// QR on local session without web server
	result.webStatus = daemon.WebServerStatusResponse{Running: false}
	result.selected = 0
	updated, _ = result.Update(tea.KeyPressMsg{Code: 'q'})
	result = updated.(Model)
	if result.toast != "Web serving not enabled for this session" {
		t.Errorf("expected web-not-enabled toast, got %q", result.toast)
	}
	if result.qrSession != nil {
		t.Error("expected no QR overlay when web not running")
	}
}
