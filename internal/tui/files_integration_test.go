//go:build !windows

package tui

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/files"
)

// socketCounter mints unique short socket paths under /tmp so we stay
// under the macOS ~104-byte sun_path limit. t.TempDir() lives under
// $TMPDIR which on macOS resolves to /var/folders/<hash>/.../ — easily
// 100+ bytes before the socket filename. Phase 118 daemon tests use the
// same approach (see internal/daemon/socket_test.go::shortSocketPath).
var socketCounter atomic.Uint64

// ----------------------------------------------------------------------------
// Phase 121 Plan 03 — End-to-end integration test
//
// Spins up an in-process HTTP server on a Unix domain socket that serves the
// `internal/files` Handler routes (list/stat/read) at the same paths the
// daemon exposes (/api/files/list, /api/files/stat, /api/files/read +
// HEAD /api/files/read). Builds a real *daemon.DaemonClient against that
// socket, then drives the TUI Update loop via tea.KeyPressMsg messages.
//
// We bypass internal/daemon's mux entirely (the plan explicitly authorises
// this — internal/daemon's test-only helpers like newFilesAPI / spyBackend
// are package-private and not reusable from internal/tui). The TUI →
// DaemonClient → /api/files/* HTTP transport remains EXACTLY the
// production path; only the daemon's session-engine bookkeeping is
// short-circuited with a closure-based files.Sandbox resolver.
//
// This test is gated `!windows` because the DaemonClient transport on
// Windows uses named pipes (not Unix sockets); the parallel Windows
// integration test would belong in a separate file with a `windows` build
// tag and use the named-pipe API. Phase 117 PAPER-01 tests use the same
// build constraint mechanism.
// ----------------------------------------------------------------------------

// integrationServer holds the lifecycle state of a one-off file-routes HTTP
// server bound to a Unix socket. Closing the listener races the server's
// goroutine; we accept the bounded shutdown window because t.Cleanup runs
// at test exit, not concurrently with the test body.
type integrationServer struct {
	socketPath string
	listener   net.Listener
	server     *http.Server
}

// setupDaemonWithSession creates a tempdir-backed session, an in-process HTTP
// server on a Unix socket serving the file routes, and a *daemon.DaemonClient
// bound to that socket.
//
// Tempdir layout:
//
//	<tempdir>/a.txt          ("alpha")
//	<tempdir>/b.md           ("# beta\n")
//	<tempdir>/sub/nested.txt ("nested-body")
//
// The returned cleanup function closes the listener + server; tempdir is
// reclaimed by t.TempDir() at test exit.
func setupDaemonWithSession(t *testing.T) (*daemon.DaemonClient, string, func()) {
	t.Helper()
	tmp := t.TempDir()

	mustWrite := func(rel, body string) {
		full := filepath.Join(tmp, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	mustWrite("a.txt", "alpha")
	mustWrite("b.md", "# beta\n")
	mustWrite("sub/nested.txt", "nested-body")

	// Build a Sandbox once and capture it in a closure. The closure form
	// matches the unexported sandboxResolver type expected by
	// files.NewHandler — Go accepts any matching function literal.
	sb, err := files.NewSandbox(tmp)
	if err != nil {
		t.Fatalf("NewSandbox(%q): %v", tmp, err)
	}
	const sessionID = "test-session"
	resolver := func(sid string) (*files.Sandbox, error) {
		if sid != sessionID {
			return nil, errors.New("session not found")
		}
		return sb, nil
	}
	handler := files.NewHandler(resolver)

	// Mount on a fresh ServeMux at exactly the paths the DaemonClient hits.
	// Both GET and HEAD route to handler.Read because files.Handler.Read
	// supports HEAD via http.ServeContent (no separate Head method).
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/files/list", handler.List)
	mux.HandleFunc("GET /api/files/stat", handler.Stat)
	mux.HandleFunc("GET /api/files/read", handler.Read)
	mux.HandleFunc("HEAD /api/files/read", handler.Read)

	// macOS imposes a 104-byte limit on sun_path; t.TempDir() under
	// $TMPDIR easily exceeds that. Use /tmp directly with a unique
	// numeric suffix so the full path stays under ~30 bytes. Mirrors
	// internal/daemon/socket_test.go::shortSocketPath.
	socketPath := fmt.Sprintf("/tmp/tuif%d.sock", socketCounter.Add(1))
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("net.Listen unix %s: %v", socketPath, err)
	}
	t.Cleanup(func() { _ = os.Remove(socketPath) })

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		// Serve returns http.ErrServerClosed on graceful shutdown; ignore
		// that one. Any other error is logged because there's no t in scope
		// — we only fail the test via assertions in the body.
		_ = srv.Serve(ln)
	}()

	// Tiny readiness probe — http.Server.Serve starts accepting almost
	// immediately, but a 1ms sleep yields the goroutine and avoids the
	// race where Dial happens before Listen has fully registered.
	time.Sleep(10 * time.Millisecond)

	client := daemon.NewDaemonClient(socketPath)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = ln.Close()
	}
	return client, sessionID, cleanup
}

// drainCmd executes cmd() synchronously, feeds the resulting message back
// through m.Update, and returns the updated model + any follow-up cmd. The
// caller decides whether to chain further drains. nil cmd is a no-op.
func drainCmd(t *testing.T, m Model, cmd tea.Cmd) (Model, tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return m, nil
	}
	msg := cmd()
	if msg == nil {
		return m, nil
	}
	updated, next := m.Update(msg)
	return updated.(Model), next
}

// drainAll repeatedly drains cmd through m.Update until no further cmd is
// queued OR the iteration cap is hit. The cap protects against an
// accidental infinite cmd→cmd loop in a future refactor.
func drainAll(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for i := 0; i < 8 && cmd != nil; i++ {
		m, cmd = drainCmd(t, m, cmd)
	}
	if cmd != nil {
		t.Fatalf("drainAll: cmd chain did not terminate within 8 iterations")
	}
	return m
}

// TestFiles_Integration_LocalSessionEndToEnd is the Phase 121 end-to-end
// merge gate. It drives a real *daemon.DaemonClient against a tempdir-
// backed in-process HTTP server, exercising the complete open → list →
// navigate → backspace → filter → preview flow.
//
// Step-by-step contract:
//
//  1. Press 'f' on a local session row → tabFiles opens, m.files.sessionID
//     is wired, a loadDirCmd is dispatched.
//  2. Drain the loadDirCmd → m.files.entries contains a.txt, b.md, sub/.
//  3. Move cursor to sub/, press Enter → loadDirCmd("sub") dispatched.
//     Drain → m.files.cwd == "sub", entries contains nested.txt.
//  4. Press Backspace → loadDirCmd("") dispatched. Drain → cwd back at
//     root ("" or "." accepted; parentDir returns "" for non-root).
//  5. Press '/' → filter activates. Type 'a' → filterInput value becomes
//     "a", filteredEntries() includes a.txt and excludes b.md.
//  6. Press Esc → filter clears.
//  7. Move cursor to a.txt, press Enter → headFileCmd dispatched. Drain
//     head, which dispatches readFileCmd; drain read → previewKind is
//     previewText AND preview.View() contains "alpha".
func TestFiles_Integration_LocalSessionEndToEnd(t *testing.T) {
	client, sessionID, cleanup := setupDaemonWithSession(t)
	defer cleanup()

	// Sanity check the transport before exercising the TUI — if the daemon
	// route is unreachable, we want a precise failure here rather than a
	// confusing TUI-state assertion failure downstream.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		entries, _, err := client.ListFiles(ctx, sessionID, ".")
		if err != nil {
			t.Fatalf("transport sanity check: ListFiles error: %v", err)
		}
		if len(entries) < 3 {
			t.Fatalf("transport sanity check: expected >= 3 entries, got %d", len(entries))
		}
	}

	// Build a Model wired to the real client, with a single local session.
	m := newModel(client, nil, "test")
	m.width = 120
	m.height = 30
	m.hasDark = true
	m.styles = newStyles(true)
	m.loading = false
	m.sessions = []daemon.SessionInfo{{
		ID: sessionID, Name: "test", CLI: "claude", Hostname: "local",
		Status: "running",
	}}
	m.rebuildUnifiedList()
	m.selected = 0
	m.panesFocus = focusContent

	// Step 1: press 'f' to open Files view from the Sessions tab.
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	if m.activeTabID() != tabFiles {
		t.Fatalf("after 'f': expected activeTabID=tabFiles, got %v", m.activeTabID())
	}
	if cmd == nil {
		t.Fatal("after 'f': expected loadDirCmd to be dispatched, got nil")
	}
	if m.files.sessionID != sessionID {
		t.Fatalf("after 'f': expected files.sessionID=%q, got %q", sessionID, m.files.sessionID)
	}

	// Step 2: drain the loadDirCmd for root.
	m, follow := drainCmd(t, m, cmd)
	if follow != nil {
		t.Fatalf("applyFilesListMsg unexpectedly returned a follow-up cmd: %T", follow)
	}
	if m.files.err != nil {
		t.Fatalf("root listing returned err: %v", m.files.err)
	}
	if len(m.files.entries) < 3 {
		t.Fatalf("expected >= 3 entries at root, got %d (%+v)", len(m.files.entries), m.files.entries)
	}

	// Step 3: find the 'sub' directory; move cursor there; press Enter.
	subIdx := -1
	for i, e := range m.files.entries {
		if ansi.Strip(e.Name) == "sub" && e.IsDir {
			subIdx = i
			break
		}
	}
	if subIdx < 0 {
		t.Fatalf("expected 'sub' directory in root listing, entries=%+v", m.files.entries)
	}
	m.files.selected = subIdx
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter on directory: expected loadDirCmd dispatch, got nil")
	}
	m, follow = drainCmd(t, m, cmd)
	if follow != nil {
		t.Fatalf("subdir loadDir returned a follow-up cmd: %T", follow)
	}
	if m.files.cwd != "sub" {
		t.Fatalf("after Enter on sub: expected cwd='sub', got %q", m.files.cwd)
	}
	hasNested := false
	for _, e := range m.files.entries {
		if ansi.Strip(e.Name) == "nested.txt" {
			hasNested = true
			break
		}
	}
	if !hasNested {
		t.Fatalf("expected nested.txt inside sub/, got %+v", m.files.entries)
	}

	// Step 4: Backspace → parent (root).
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Backspace from sub: expected loadDirCmd(parent) dispatch, got nil")
	}
	m, follow = drainCmd(t, m, cmd)
	if follow != nil {
		t.Fatalf("Backspace loadDir returned a follow-up cmd: %T", follow)
	}
	if m.files.cwd != "" && m.files.cwd != "." {
		t.Fatalf("after Backspace: expected cwd at root ('' or '.'), got %q", m.files.cwd)
	}

	// Step 5: '/' to activate filter, then 'a' to filter for a.txt.
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/'})
	m = updated.(Model)
	if !m.files.filterActive {
		t.Fatal("after '/': expected filterActive=true")
	}
	// textinput.Update reads msg.Text (not msg.Code) when inserting
	// runes — see charm.land/bubbles/v2/textinput.Update default branch.
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = updated.(Model)
	if v := m.files.filterInput.Value(); v != "a" {
		t.Fatalf("after typing 'a': expected filterInput value='a', got %q", v)
	}
	filtered := m.files.filteredEntries()
	hasA, hasB := false, false
	for _, e := range filtered {
		n := ansi.Strip(e.Name)
		if n == "a.txt" {
			hasA = true
		}
		if n == "b.md" {
			hasB = true
		}
	}
	if !hasA || hasB {
		t.Fatalf("filter 'a': expected hasA=true hasB=false, got hasA=%v hasB=%v (filtered=%+v)",
			hasA, hasB, filtered)
	}

	// Step 6: Esc clears filter.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = updated.(Model)
	if m.files.filterActive {
		t.Fatal("after Esc: expected filterActive=false")
	}
	if v := m.files.filterInput.Value(); v != "" {
		t.Fatalf("after Esc: expected filterInput value to be cleared, got %q", v)
	}

	// Step 7: select a.txt and Enter → head → read → previewKind=previewText.
	aIdx := -1
	for i, e := range m.files.entries {
		if ansi.Strip(e.Name) == "a.txt" {
			aIdx = i
			break
		}
	}
	if aIdx < 0 {
		t.Fatalf("expected a.txt in root listing after filter clear, got %+v", m.files.entries)
	}
	m.files.selected = aIdx
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter on a.txt: expected headFileCmd dispatch, got nil")
	}
	// Drain head → applyFilesHeadMsg returns readFileCmd → drain → previewText.
	m = drainAll(t, m, cmd)

	if m.files.previewKind != previewText {
		t.Fatalf("after head+read for a.txt: expected previewKind=previewText, got %v (previewErr=%v)",
			m.files.previewKind, m.files.previewErr)
	}
	if !strings.Contains(m.files.preview.View(), "alpha") {
		t.Errorf("expected preview to contain 'alpha'; got %q", m.files.preview.View())
	}
}
