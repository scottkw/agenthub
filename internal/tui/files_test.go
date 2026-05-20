package tui

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/scottkw/agenthub/internal/daemon"
)

// TestTruncateLeft covers the documented behavior for the new
// left-truncating helper used by the Files status line.
func TestTruncateLeft(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
		// optional predicate checks
		startsWith string
		endsWith   string
		runeWidth  int // expected []rune width when > 0
	}{
		{name: "shorter than maxWidth returns unchanged", input: "short", maxWidth: 20, want: "short"},
		{name: "empty string returns empty", input: "", maxWidth: 10, want: ""},
		{
			// Width may be <= maxWidth: when a path-segment boundary lies
			// inside the kept window, snapping to it drops a few runes so
			// the result reads as "…/<dir>/<leaf>" instead of a mid-
			// segment fragment. The behavior contract is "prefix='…/',
			// preserve the leaf-end, width <= maxWidth" — not a hard
			// exact-width guarantee.
			name:       "long path is truncated with ellipsis prefix",
			input:      "a/b/c/d/utils/helper.ts",
			maxWidth:   18,
			startsWith: "…/",
			endsWith:   "helper.ts",
		},
		{name: "tiny maxWidth returns rightmost runes without panic", input: "abc", maxWidth: 2, want: "bc"},
		{name: "zero maxWidth returns empty", input: "abc", maxWidth: 0, want: ""},
		{name: "negative maxWidth returns empty", input: "abc", maxWidth: -3, want: ""},
		{name: "multibyte runes counted by rune not byte", input: "αβγδεζηθικ", maxWidth: 4, runeWidth: 4, startsWith: "…/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateLeft(tc.input, tc.maxWidth)
			if tc.want != "" || (tc.startsWith == "" && tc.endsWith == "" && tc.runeWidth == 0) {
				if got != tc.want {
					t.Errorf("truncateLeft(%q, %d) = %q, want %q", tc.input, tc.maxWidth, got, tc.want)
				}
			}
			if tc.startsWith != "" && !strings.HasPrefix(got, tc.startsWith) {
				t.Errorf("truncateLeft(%q, %d) = %q, expected prefix %q", tc.input, tc.maxWidth, got, tc.startsWith)
			}
			if tc.endsWith != "" && !strings.HasSuffix(got, tc.endsWith) {
				t.Errorf("truncateLeft(%q, %d) = %q, expected suffix %q", tc.input, tc.maxWidth, got, tc.endsWith)
			}
			if tc.runeWidth > 0 {
				if w := len([]rune(got)); w != tc.runeWidth {
					t.Errorf("truncateLeft(%q, %d) = %q, rune-width %d, want %d", tc.input, tc.maxWidth, got, w, tc.runeWidth)
				}
			}
		})
	}
}

// TestFiles_TabID_Distinct asserts the new tabFiles iota value is distinct from
// every preceding tab ID.
func TestFiles_TabID_Distinct(t *testing.T) {
	if tabFiles == tabHome {
		t.Error("tabFiles must differ from tabHome")
	}
	if tabFiles == tabSessions {
		t.Error("tabFiles must differ from tabSessions")
	}
	if tabFiles == tabRemote {
		t.Error("tabFiles must differ from tabRemote")
	}
	if tabFiles == tabSettings {
		t.Error("tabFiles must differ from tabSettings")
	}
}

// TestFiles_KeyMap_BindsF asserts the FilesOpen binding catches the 'f' key.
func TestFiles_KeyMap_BindsF(t *testing.T) {
	km := defaultKeyMap()
	if !key.Matches(tea.KeyPressMsg{Code: 'f'}, km.FilesOpen) {
		t.Error("FilesOpen binding does not match 'f' key")
	}
	if !key.Matches(tea.KeyPressMsg{Code: '/'}, km.FilterStart) {
		t.Error("FilterStart binding does not match '/' key")
	}
	if len(km.FilesFocusToggle.Keys()) == 0 {
		t.Error("FilesFocusToggle binding has no keys configured")
	}
	if len(km.FilterEsc.Keys()) == 0 {
		t.Error("FilterEsc binding has no keys configured")
	}
}

// silence unused-import warnings if no daemon-using test exists in this file yet
var _ = daemon.SessionInfo{}

// TestLoadDirCmd_NilClient_ReturnsErrSentinel asserts the nil-client guard in
// loadDirCmd returns a typed filesListMsg carrying errNilClient and the echo
// fields populated, so the test-only paths (e.g. testModel() with nil client)
// don't panic.
func TestLoadDirCmd_NilClient_ReturnsErrSentinel(t *testing.T) {
	cmd := loadDirCmd(nil, "sid", ".")
	if cmd == nil {
		t.Fatal("loadDirCmd returned nil command")
	}
	msg, ok := cmd().(filesListMsg)
	if !ok {
		t.Fatalf("expected filesListMsg, got %T", cmd())
	}
	if msg.err != errNilClient {
		t.Errorf("expected errNilClient, got %v", msg.err)
	}
	if msg.sessionID != "sid" {
		t.Errorf("expected echo sessionID=sid, got %q", msg.sessionID)
	}
	if msg.relPath != "." {
		t.Errorf("expected echo relPath=., got %q", msg.relPath)
	}
}

func TestReadFileCmd_NilClient_ReturnsErrSentinel(t *testing.T) {
	cmd := readFileCmd(nil, "sid", "a.txt")
	if cmd == nil {
		t.Fatal("readFileCmd returned nil command")
	}
	msg, ok := cmd().(filesReadMsg)
	if !ok {
		t.Fatalf("expected filesReadMsg, got %T", cmd())
	}
	if msg.err != errNilClient {
		t.Errorf("expected errNilClient, got %v", msg.err)
	}
	if msg.sessionID != "sid" || msg.relPath != "a.txt" {
		t.Errorf("expected echo sessionID=sid relPath=a.txt, got %q %q", msg.sessionID, msg.relPath)
	}
}

func TestHeadFileCmd_NilClient_ReturnsErrSentinel(t *testing.T) {
	cmd := headFileCmd(nil, "sid", "a.txt")
	if cmd == nil {
		t.Fatal("headFileCmd returned nil command")
	}
	msg, ok := cmd().(filesHeadMsg)
	if !ok {
		t.Fatalf("expected filesHeadMsg, got %T", cmd())
	}
	if msg.err != errNilClient {
		t.Errorf("expected errNilClient, got %v", msg.err)
	}
	if msg.sessionID != "sid" || msg.relPath != "a.txt" {
		t.Errorf("expected echo sessionID=sid relPath=a.txt, got %q %q", msg.sessionID, msg.relPath)
	}
}

// TestLoadDirCmd_DispatchesAsync proves the factory returns a non-nil tea.Cmd
// without executing the I/O synchronously — the closure must contain the work.
func TestLoadDirCmd_DispatchesAsync(t *testing.T) {
	cmd := loadDirCmd(nil, "sid", ".")
	if cmd == nil {
		t.Fatal("loadDirCmd returned nil command — I/O must be deferred to the closure")
	}
}

// --- Task 3: priority dispatch + open-from-Sessions ---

// TestFiles_HandleKey_DispatchPriority asserts that when the active tab is
// tabFiles, handleKey routes the press to handleFilesKey BEFORE the tab-
// cycling check at Priority 6 — the Plan 01 stub swallows unrecognized keys
// and the active tab must remain tabFiles afterward.
func TestFiles_HandleKey_DispatchPriority(t *testing.T) {
	m := testModel()
	m.openTabs = []tabID{tabSessions, tabFiles}
	m.activeTab = 1
	if m.activeTabID() != tabFiles {
		t.Fatalf("setup: expected activeTabID=tabFiles, got %v", m.activeTabID())
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	r := updated.(Model)
	if r.activeTabID() != tabFiles {
		t.Errorf("expected activeTab to remain tabFiles after unrecognized key, got %v", r.activeTabID())
	}
}

// TestFiles_HandleKey_DispatchPriority_BelowKillConfirm asserts modal
// priority 2 still beats the new Priority 5.5 — pressing 'n' on a
// killConfirm modal cancels the kill, even if the active tab is tabFiles.
func TestFiles_HandleKey_DispatchPriority_BelowKillConfirm(t *testing.T) {
	m := testModel()
	m.modal = modalKillConfirm
	m.killTarget = &daemon.SessionInfo{ID: "x", Name: "y"}
	m.openTabs = []tabID{tabFiles}
	m.activeTab = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	r := updated.(Model)
	if r.modal != modalNone {
		t.Errorf("expected modal=modalNone after 'n' (kill cancel) — kill-confirm priority must beat Files priority. got modal=%v", r.modal)
	}
}

// TestFiles_OpenFromSessions_LocalEntry asserts pressing 'f' on a local
// session opens tabFiles, dispatches loadDirCmd, and stores the session ID.
func TestFiles_OpenFromSessions_LocalEntry(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "sess-1", Name: "x", CLI: "claude", Status: "running"},
	}
	m.rebuildUnifiedList()
	m.selected = 0

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'f'})
	r := updated.(Model)
	if r.activeTabID() != tabFiles {
		t.Fatalf("expected activeTab=tabFiles after 'f' on local session, got %v", r.activeTabID())
	}
	if cmd == nil {
		t.Error("expected loadDirCmd to be returned (non-nil cmd)")
	}
	if r.files.sessionID != "sess-1" {
		t.Errorf("expected files.sessionID=sess-1, got %q", r.files.sessionID)
	}
}

// TestFiles_OpenFromSessions_RemoteEntry_ShowsToast asserts pressing 'f' on
// a remote session surfaces the documented toast and does NOT open tabFiles.
func TestFiles_OpenFromSessions_RemoteEntry_ShowsToast(t *testing.T) {
	m := testModel()
	m.remoteSessions = []ListRemoteGroup{
		{Hostname: "peer", Sessions: []RemoteSessionEntry{
			{ID: "r1", Name: "rem", CLIType: "claude", Status: "running", Hostname: "peer"},
		}},
	}
	m.rebuildUnifiedList()
	// First entry is the divider; advance to the remote entry.
	for i, e := range m.unifiedList {
		if e.kind == entryRemote {
			m.selected = i
			break
		}
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	r := updated.(Model)
	if r.activeTabID() == tabFiles {
		t.Error("expected NOT to open tabFiles on remote-session 'f' press")
	}
	want := "File browser not available for remote sessions in v3.4"
	if r.toast != want {
		t.Errorf("expected toast %q, got %q", want, r.toast)
	}
	if r.toastKind != toastInfo {
		t.Errorf("expected toastKind=toastInfo, got %v", r.toastKind)
	}
}

// TestFiles_OpenFromSessions_EmptyList_NoOp asserts pressing 'f' with an
// empty unifiedList is a no-op (no toast, no tab open, no cmd).
func TestFiles_OpenFromSessions_EmptyList_NoOp(t *testing.T) {
	m := testModel()
	// rebuildUnifiedList with no sessions/remote leaves the list empty.
	m.rebuildUnifiedList()
	if len(m.unifiedList) != 0 {
		t.Fatalf("setup: expected empty unifiedList, got %d entries", len(m.unifiedList))
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'f'})
	r := updated.(Model)
	if r.activeTabID() == tabFiles {
		t.Error("expected NOT to open tabFiles on empty list")
	}
	if r.toast != "" {
		t.Errorf("expected no toast on empty list, got %q", r.toast)
	}
	if cmd != nil {
		t.Error("expected nil cmd on empty list")
	}
}

// TestFiles_OpenFromSessions_ResetsModel asserts every 'f' keypress RESETS
// the embedded filesModel — per RESEARCH.md Pitfall TUI-PITFALL-6 the TUI
// must NOT carry stale state from a prior session.
func TestFiles_OpenFromSessions_ResetsModel(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "sess-1", Name: "x", CLI: "claude", Status: "running"},
	}
	m.rebuildUnifiedList()
	m.selected = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'f'})
	m = updated.(Model)
	m.files.cwd = "subdir" // simulate Plan-02 navigation

	// Switch back to Sessions tab so the next 'f' goes through
	// handleContentKey (NOT the in-tab handleFilesKey stub).
	for i, t := range m.openTabs {
		if t == tabSessions {
			m.activeTab = i
			break
		}
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'f'})
	r := updated.(Model)
	if r.files.cwd != "" {
		t.Errorf("expected files.cwd reset to \"\" after re-press, got %q (Pitfall TUI-PITFALL-6)", r.files.cwd)
	}
}

// --- Plan 02 Task 1: applyFilesListMsg / applyFilesHeadMsg / applyFilesReadMsg ---

// filesTestModel returns a Model whose embedded filesModel is fully initialised
// against a known session ID — used by the Plan 02 apply-helper tests.
func filesTestModel(sid string) Model {
	m := testModel()
	m.files = newFilesModel(sid, 30, 20, 50, 20)
	return m
}

func TestApplyFilesListMsg_HappyPath(t *testing.T) {
	m := filesTestModel("s1")
	updated, cmd := m.applyFilesListMsg(filesListMsg{
		sessionID: "s1",
		relPath:   "sub",
		entries:   []daemon.FileEntry{{Name: "a"}, {Name: "b"}},
		truncated: false,
	})
	if cmd != nil {
		t.Errorf("expected nil cmd, got non-nil")
	}
	if updated.files.cwd != "sub" {
		t.Errorf("expected cwd=sub, got %q", updated.files.cwd)
	}
	if len(updated.files.entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(updated.files.entries))
	}
	if updated.files.selected != 0 {
		t.Errorf("expected selected=0 after listing, got %d", updated.files.selected)
	}
	if updated.files.loading {
		t.Error("expected loading=false")
	}
	if updated.files.err != nil {
		t.Errorf("expected err=nil, got %v", updated.files.err)
	}
}

func TestApplyFilesListMsg_StaleDiscarded(t *testing.T) {
	m := filesTestModel("s1")
	m.files.cwd = "original"
	m.files.entries = []daemon.FileEntry{{Name: "original"}}
	updated, _ := m.applyFilesListMsg(filesListMsg{
		sessionID: "s2", // different session
		relPath:   "sub",
		entries:   []daemon.FileEntry{{Name: "a"}, {Name: "b"}},
	})
	if updated.files.cwd != "original" {
		t.Errorf("stale msg should not mutate cwd; got %q", updated.files.cwd)
	}
	if len(updated.files.entries) != 1 || updated.files.entries[0].Name != "original" {
		t.Errorf("stale msg should not mutate entries; got %+v", updated.files.entries)
	}
}

func TestApplyFilesListMsg_SessionNotFound_FriendlyMessage(t *testing.T) {
	m := filesTestModel("s1")
	updated, _ := m.applyFilesListMsg(filesListMsg{
		sessionID: "s1",
		relPath:   ".",
		err:       errors.New("files list: 404 session not found or has no working directory"),
	})
	if updated.files.err == nil {
		t.Fatal("expected err to be set")
	}
	if !strings.Contains(updated.files.err.Error(), "Session no longer running") {
		t.Errorf("expected friendly 'Session no longer running' message, got %q", updated.files.err.Error())
	}
}

func TestApplyFilesHeadMsg_OverCap_RefusalMessage(t *testing.T) {
	m := filesTestModel("s1")
	updated, cmd := m.applyFilesHeadMsg(filesHeadMsg{
		sessionID: "s1",
		relPath:   "big.bin",
		size:      6 * 1024 * 1024,
		mime:      "text/plain", // over-cap beats text/* check
	})
	if updated.files.previewKind != previewOverCap {
		t.Errorf("expected previewOverCap, got %v", updated.files.previewKind)
	}
	if cmd != nil {
		t.Error("expected nil cmd on over-cap (no read dispatched)")
	}
	if !strings.Contains(updated.files.preview.GetContent(), "Too large") {
		t.Errorf("expected 'Too large…' content, got %q", updated.files.preview.GetContent())
	}
}

func TestApplyFilesHeadMsg_Binary_RefusalMessage(t *testing.T) {
	m := filesTestModel("s1")
	updated, cmd := m.applyFilesHeadMsg(filesHeadMsg{
		sessionID: "s1",
		relPath:   "image.png",
		size:      100,
		mime:      "image/png",
	})
	if updated.files.previewKind != previewBinary {
		t.Errorf("expected previewBinary, got %v", updated.files.previewKind)
	}
	if cmd != nil {
		t.Error("expected nil cmd on binary (no read dispatched)")
	}
	if !strings.Contains(updated.files.preview.GetContent(), "Use desktop or web to preview") {
		t.Errorf("expected binary refusal content, got %q", updated.files.preview.GetContent())
	}
}

func TestApplyFilesHeadMsg_Text_DispatchesRead(t *testing.T) {
	m := filesTestModel("s1")
	updated, cmd := m.applyFilesHeadMsg(filesHeadMsg{
		sessionID: "s1",
		relPath:   "a.txt",
		size:      100,
		mime:      "text/plain",
	})
	if cmd == nil {
		t.Error("expected non-nil readFileCmd to be dispatched")
	}
	if !updated.files.previewLoading {
		t.Error("expected previewLoading=true while read is in flight")
	}
}

func TestApplyFilesReadMsg_TextSetsContent(t *testing.T) {
	m := filesTestModel("s1")
	updated, _ := m.applyFilesReadMsg(filesReadMsg{
		sessionID: "s1",
		relPath:   "a.txt",
		data:      []byte("hello"),
		mime:      "text/plain",
	})
	if updated.files.previewKind != previewText {
		t.Errorf("expected previewText, got %v", updated.files.previewKind)
	}
	if !strings.Contains(updated.files.preview.GetContent(), "hello") {
		t.Errorf("expected preview to contain 'hello', got %q", updated.files.preview.GetContent())
	}
}

func TestApplyFilesReadMsg_MarkdownSuffix_UsesGlamour(t *testing.T) {
	m := filesTestModel("s1")
	updated, _ := m.applyFilesReadMsg(filesReadMsg{
		sessionID: "s1",
		relPath:   "README.md",
		data:      []byte("# Hello\n\nWorld\n"),
		mime:      "text/markdown",
	})
	if updated.files.previewKind != previewMarkdown {
		t.Errorf("expected previewMarkdown, got %v", updated.files.previewKind)
	}
	// Content must be non-empty after glamour render.
	if updated.files.preview.GetContent() == "" {
		t.Error("expected glamour-rendered content to be non-empty")
	}
}

func TestApplyFilesReadMsg_StaleSessionID_Discarded(t *testing.T) {
	m := filesTestModel("s1")
	updated, _ := m.applyFilesReadMsg(filesReadMsg{
		sessionID: "s2",
		relPath:   "a.txt",
		data:      []byte("hello"),
		mime:      "text/plain",
	})
	if updated.files.previewKind != previewEmpty {
		t.Errorf("stale read should not change previewKind; expected previewEmpty, got %v", updated.files.previewKind)
	}
}

// keep ansi import live for later Plan 02 tests
var _ = ansi.Strip
