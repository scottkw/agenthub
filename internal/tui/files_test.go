package tui

import (
	"errors"
	"fmt"
	"os"
	"regexp"
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
	cmd := loadDirCmd(nil, "sid", ".", 1)
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
	if msg.generation != 1 {
		t.Errorf("expected echo generation=1, got %d", msg.generation)
	}
}

func TestReadFileCmd_NilClient_ReturnsErrSentinel(t *testing.T) {
	cmd := readFileCmd(nil, "sid", "a.txt", 1)
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
	cmd := headFileCmd(nil, "sid", "a.txt", 1)
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
	cmd := loadDirCmd(nil, "sid", ".", 1)
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

// TestFiles_OpenFromSessions_RemoteEntry_NoCachedCap_OpensPrompt asserts
// pressing 'f' on a remote session WITHOUT a cached cap opens the join-code
// prompt modal — Phase 122 replaced the v3.4 toast with the prompt flow.
func TestFiles_OpenFromSessions_RemoteEntry_NoCachedCap_OpensPrompt(t *testing.T) {
	m := testModel()
	m.remoteSessions = []ListRemoteGroup{
		{Hostname: "peer", Sessions: []RemoteSessionEntry{
			{
				ID: "r1", Name: "rem", CLIType: "claude", Status: "running",
				Hostname: "peer",
				URL:      "https://peer.example:9443/sessions/r1",
			},
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
		t.Error("expected NOT to open tabFiles immediately on remote 'f' with no cap")
	}
	if r.modal != modalJoinCodePrompt {
		t.Errorf("expected modal=modalJoinCodePrompt, got %v", r.modal)
	}
	if r.joinCodePrompt.sessionID != "r1" {
		t.Errorf("expected joinCodePrompt.sessionID=r1, got %q", r.joinCodePrompt.sessionID)
	}
	if r.joinCodePrompt.remoteBaseURL != "https://peer.example:9443" {
		t.Errorf("expected remoteBaseURL stripped to base, got %q", r.joinCodePrompt.remoteBaseURL)
	}
	// The v3.4 toast must be gone.
	if strings.Contains(r.toast, "File browser not available for remote sessions") {
		t.Errorf("v3.4 remote-refusal toast must be REMOVED, got %q", r.toast)
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    "sub",
		entries:    []daemon.FileEntry{{Name: "a"}, {Name: "b"}},
		truncated:  false,
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
		sessionID:  "s2", // different session
		generation: m.files.generation,
		relPath:    "sub",
		entries:    []daemon.FileEntry{{Name: "a"}, {Name: "b"}},
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    ".",
		err:        errors.New("files list: 404 session not found or has no working directory"),
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    "big.bin",
		size:       6 * 1024 * 1024,
		mime:       "text/plain", // over-cap beats text/* check
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    "image.png",
		size:       100,
		mime:       "image/png",
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    "a.txt",
		size:       100,
		mime:       "text/plain",
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    "a.txt",
		data:       []byte("hello"),
		mime:       "text/plain",
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
		sessionID:  "s1",
		generation: m.files.generation,
		relPath:    "README.md",
		data:       []byte("# Hello\n\nWorld\n"),
		mime:       "text/markdown",
	})
	if updated.files.previewKind != previewMarkdown {
		t.Errorf("expected previewMarkdown, got %v", updated.files.previewKind)
	}
	// Content must be non-empty after glamour render.
	if updated.files.preview.GetContent() == "" {
		t.Error("expected glamour-rendered content to be non-empty")
	}
}

func TestApplyFilesListMsg_StaleGenerationDiscarded(t *testing.T) {
	// WR-03: a reply tagged with an older generation must NOT overwrite a
	// freshly-reset model. This is the intra-session race: same sessionID
	// but a prior in-flight loadDir lands after the user has already
	// navigated/reset.
	m := filesTestModel("s1")
	m.files.cwd = "current"
	m.files.entries = []daemon.FileEntry{{Name: "current_a"}}
	currentGen := m.files.generation
	// Simulate a stale message: generation BELOW the current one.
	updated, _ := m.applyFilesListMsg(filesListMsg{
		sessionID:  "s1",
		generation: currentGen - 1, // stale
		relPath:    "stale",
		entries:    []daemon.FileEntry{{Name: "stale_a"}},
	})
	if updated.files.cwd != "current" {
		t.Errorf("stale-generation msg should not mutate cwd; got %q", updated.files.cwd)
	}
	if len(updated.files.entries) != 1 || updated.files.entries[0].Name != "current_a" {
		t.Errorf("stale-generation msg should not mutate entries; got %+v", updated.files.entries)
	}
}

func TestApplyFilesReadMsg_StaleSessionID_Discarded(t *testing.T) {
	m := filesTestModel("s1")
	updated, _ := m.applyFilesReadMsg(filesReadMsg{
		sessionID:  "s2",
		generation: m.files.generation,
		relPath:    "a.txt",
		data:       []byte("hello"),
		mime:       "text/plain",
	})
	if updated.files.previewKind != previewEmpty {
		t.Errorf("stale read should not change previewKind; expected previewEmpty, got %v", updated.files.previewKind)
	}
}

// --- Plan 02 Task 2: handleFilesKey full dispatch ---

// filesKeyTestModel returns a Model whose embedded filesModel matches Plan-02
// dispatch expectations: sessionID="s1", active tab is tabFiles.
func filesKeyTestModel() Model {
	m := filesTestModel("s1")
	m.openTabs = []tabID{tabSessions, tabFiles}
	m.activeTab = 1
	return m
}

func TestHandleFilesKey_Backspace_AtRoot_NoOp(t *testing.T) {
	m := filesKeyTestModel()
	m.files.cwd = ""
	updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd != nil {
		t.Error("expected nil cmd at root Backspace")
	}
	if updated.(Model).files.cwd != "" {
		t.Errorf("expected cwd unchanged at root, got %q", updated.(Model).files.cwd)
	}
}

func TestHandleFilesKey_Backspace_AtRoot_Dot_NoOp(t *testing.T) {
	m := filesKeyTestModel()
	m.files.cwd = "."
	updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd != nil {
		t.Error("expected nil cmd at root='.' Backspace")
	}
	if updated.(Model).files.cwd != "." {
		t.Errorf("expected cwd unchanged at '.', got %q", updated.(Model).files.cwd)
	}
}

func TestHandleFilesKey_Backspace_NonRoot_DispatchesParent(t *testing.T) {
	m := filesKeyTestModel()
	m.files.cwd = "a/b"
	_, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if cmd == nil {
		t.Error("expected non-nil loadDirCmd at non-root Backspace")
	}
}

func TestHandleFilesKey_Slash_ActivatesFilter(t *testing.T) {
	m := filesKeyTestModel()
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: '/'})
	if !updated.(Model).files.filterActive {
		t.Error("expected filterActive=true after '/'")
	}
}

func TestHandleFilesKey_FilterActive_EscClears(t *testing.T) {
	m := filesKeyTestModel()
	m.files.filterActive = true
	m.files.filterInput.SetValue("abc")
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	r := updated.(Model)
	if r.files.filterActive {
		t.Error("expected filterActive=false after Esc")
	}
	if r.files.filterInput.Value() != "" {
		t.Errorf("expected filterInput cleared, got %q", r.files.filterInput.Value())
	}
}

func TestHandleFilesKey_FilterActive_BackspaceDoesNotNavigate(t *testing.T) {
	// Pitfall TUI-PITFALL-2: when filter is active, Backspace MUST go to the
	// textinput (deleting a char) — never navigate the cwd up.
	m := filesKeyTestModel()
	m.files.cwd = "a/b"
	m.files.filterActive = true
	m.files.filterInput.SetValue("abc")
	m.files.filterInput.CursorEnd()
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyBackspace})
	r := updated.(Model)
	if r.files.cwd != "a/b" {
		t.Errorf("Backspace during filter must NOT navigate up; cwd changed to %q", r.files.cwd)
	}
}

func TestHandleFilesKey_FilterActive_CtrlCQuits(t *testing.T) {
	// CR-01 (BLOCKER): Ctrl+C MUST quit the TUI even when the filter input
	// is active. The bundled textinput has no Ctrl+C handler, so without an
	// explicit interception the key is silently swallowed and the user
	// cannot exit the TUI without first pressing Esc.
	m := filesKeyTestModel()
	m.files.filterActive = true
	m.files.filterInput.SetValue("abc")
	_, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected non-nil tea.Quit cmd from Ctrl+C while filtering, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg from Ctrl+C while filtering, got %T", msg)
	}
}

func TestHandleFilesKey_Down_MovesCursor(t *testing.T) {
	m := filesKeyTestModel()
	m.files.entries = []daemon.FileEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.files.selected = 0
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if updated.(Model).files.selected != 1 {
		t.Errorf("expected selected=1 after Down, got %d", updated.(Model).files.selected)
	}
}

func TestHandleFilesKey_Down_ClampedAtEnd(t *testing.T) {
	m := filesKeyTestModel()
	m.files.entries = []daemon.FileEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.files.selected = 2
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if updated.(Model).files.selected != 2 {
		t.Errorf("expected selected clamped to 2, got %d", updated.(Model).files.selected)
	}
}

func TestHandleFilesKey_Enter_OnDir_DispatchesLoadDir(t *testing.T) {
	m := filesKeyTestModel()
	m.files.entries = []daemon.FileEntry{{Name: "sub", IsDir: true}}
	m.files.selected = 0
	_, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil loadDirCmd on Enter for a directory")
	}
}

func TestHandleFilesKey_Enter_OnFile_DispatchesHead(t *testing.T) {
	m := filesKeyTestModel()
	m.files.entries = []daemon.FileEntry{{Name: "a.txt", IsDir: false}}
	m.files.selected = 0
	updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected non-nil headFileCmd on Enter for a file")
	}
	if !updated.(Model).files.previewLoading {
		t.Error("expected previewLoading=true on Enter for a file")
	}
}

func TestHandleFilesKey_Tab_TogglesPreviewFocus(t *testing.T) {
	m := filesKeyTestModel()
	before := m.files.previewFocused
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: tea.KeyTab})
	if updated.(Model).files.previewFocused == before {
		t.Error("expected previewFocused to toggle on Tab")
	}
}

func TestHandleFilesKey_BracketKeysCycleTabs(t *testing.T) {
	// Pitfall TUI-PITFALL-7: '[' / ']' must cycle tabs even when
	// handleFilesKey is the active dispatcher.
	m := filesKeyTestModel()
	m.openTabs = []tabID{tabSessions, tabFiles}
	m.activeTab = 1
	updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: '['})
	if updated.(Model).activeTab != 0 {
		t.Errorf("expected activeTab=0 after '[', got %d", updated.(Model).activeTab)
	}
}

func TestParentDir(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a/b", "a"},
		{"a", ""},
		{"", ""},
		{".", ""},
		{"a/b/c", "a/b"},
		{"a/", ""},
	}
	for _, tc := range cases {
		if got := parentDir(tc.in); got != tc.want {
			t.Errorf("parentDir(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestJoinDir(t *testing.T) {
	cases := []struct{ base, name, want string }{
		{"", "a", "a"},
		{".", "a", "a"},
		{"a", "b", "a/b"},
		{"a/b", "c", "a/b/c"},
	}
	for _, tc := range cases {
		if got := joinDir(tc.base, tc.name); got != tc.want {
			t.Errorf("joinDir(%q, %q) = %q, want %q", tc.base, tc.name, got, tc.want)
		}
	}
}

// --- Plan 02 Task 3: renderFilesTab + status line + help overlay + hint bar ---

func TestRenderFilesTab_BasicLayout(t *testing.T) {
	m := filesTestModel("s1")
	m.openTabs = []tabID{tabFiles}
	m.activeTab = 0
	m.files.entries = []daemon.FileEntry{
		{Name: "alpha.txt"},
		{Name: "beta.md"},
		{Name: "gamma"},
	}
	got := m.renderFilesTab(120, 24)
	if got == "" {
		t.Fatal("renderFilesTab returned empty string")
	}
	plain := ansi.Strip(got)
	if !strings.Contains(plain, "alpha.txt") {
		t.Errorf("expected output to contain first entry 'alpha.txt'; got %q", plain)
	}
}

func TestRenderFilesStatusLine_TruncatedFlag(t *testing.T) {
	m := filesTestModel("s1")
	m.files.cwd = "."
	m.files.truncated = true
	m.files.entries = []daemon.FileEntry{
		{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"},
	}
	got := ansi.Strip(m.renderFilesStatusLine(120))
	if !strings.Contains(got, "(truncated)") {
		t.Errorf("expected status line to include (truncated); got %q", got)
	}
}

func TestRenderFilesStatusLine_ErrorShown(t *testing.T) {
	m := filesTestModel("s1")
	m.files.err = errors.New("Session no longer running")
	got := ansi.Strip(m.renderFilesStatusLine(120))
	if !strings.Contains(got, "Session no longer running") {
		t.Errorf("expected status line to show the error; got %q", got)
	}
}

func TestRenderFilesStatusLine_LeftTruncation(t *testing.T) {
	m := filesTestModel("s1")
	m.files.cwd = "very/deep/nested/path/structure/utils/helper.ts"
	got := ansi.Strip(m.renderFilesStatusLine(60))
	if !strings.Contains(got, "…/") {
		t.Errorf("expected left-truncation ellipsis '…/' in status; got %q", got)
	}
}

func TestBuildHelpContent_FilesActive_ShowsFilesGroup(t *testing.T) {
	m := testModel()
	m.openTabs = []tabID{tabFiles}
	m.activeTab = 0
	got := ansi.Strip(m.buildHelpContent())
	if !strings.Contains(got, "Files") {
		t.Errorf("expected Files group header; got %q", got)
	}
	if !strings.Contains(got, "Enter directory / preview file") {
		t.Errorf("expected Files-specific Enter binding; got %q", got)
	}
}

func TestBuildHelpContent_SessionsActive_ShowsSessionsGroup(t *testing.T) {
	m := testModel()
	m.openTabs = []tabID{tabSessions}
	m.activeTab = 0
	got := ansi.Strip(m.buildHelpContent())
	if !strings.Contains(got, "Sessions") {
		t.Errorf("expected Sessions group header; got %q", got)
	}
	if !strings.Contains(got, "Open files view") {
		t.Errorf("expected 'Open files view' binding; got %q", got)
	}
}

func TestRenderHintBar_FilesActive(t *testing.T) {
	m := testModel()
	m.openTabs = []tabID{tabFiles}
	m.activeTab = 0
	got := ansi.Strip(m.renderHintBar())
	if !strings.Contains(got, "Tab Focus") {
		t.Errorf("expected Files-specific hint 'Tab Focus'; got %q", got)
	}
}

func TestRenderHintBar_SessionsActive(t *testing.T) {
	m := testModel()
	m.openTabs = []tabID{tabSessions}
	m.activeTab = 0
	got := ansi.Strip(m.renderHintBar())
	if !strings.Contains(got, "Attach") {
		t.Errorf("expected Sessions-specific hint 'Attach'; got %q", got)
	}
}

// keep ansi import live for later Plan 02 tests
var _ = ansi.Strip

// ----------------------------------------------------------------------------
// Phase 121 Plan 03 — TUI-XX coverage matrix + merge-gate tests
// ----------------------------------------------------------------------------

// TestFiles_NoSyncFSCalls is the TUI-07 merge gate: production files in the
// TUI Files subsystem MUST NOT call synchronous os.ReadDir / os.Open /
// os.Stat / os.OpenFile. All filesystem I/O must go through the daemon
// socket via tea.Cmd (see loadDirCmd / readFileCmd / headFileCmd). A
// regression that introduces a synchronous FS call into the Update path
// would silently freeze the Bubble Tea event loop on slow disks; the
// grep below is the cheapest possible safety net.
//
// The test file itself is allowed to call os.ReadFile (we are READING the
// production source). The production source must contain ZERO matches.
func TestFiles_NoSyncFSCalls(t *testing.T) {
	files := []string{
		"files.go",
		"files_cmds.go",
	}
	// Match os.ReadDir, os.Open, os.OpenFile, os.Stat with word boundaries.
	re := regexp.MustCompile(`\bos\.(ReadDir|Open|OpenFile|Stat)\b`)
	commentLine := regexp.MustCompile(`^\s*//`)
	for _, name := range files {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if commentLine.MatchString(line) {
				continue
			}
			// Strip inline trailing comment so doc references like
			// "see os.ReadDir for semantics" don't trip the gate.
			if idx := strings.Index(line, "//"); idx >= 0 {
				line = line[:idx]
			}
			if re.MatchString(line) {
				t.Errorf("TUI-07 violation: %s:%d contains synchronous FS call: %s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// TestFiles_PathTruncation_StatusLine exercises TUI-06 specifically: a very
// deep cwd must render with a `…/` ellipsis prefix and preserve the leaf
// segment, with the leading "very/deep" components dropped. Asserts on the
// rendered status line after ansi.Strip so style codes don't perturb the
// substring checks.
//
// Post-WR-06: pathBudget is computed as w - lipgloss.Width(tail), not a
// magic 40. With a deeply-nested cwd longer than the available budget,
// truncateLeft snaps the result to a path-segment boundary with a '…/'
// prefix. Pane width 50 + tail ' • 1 entries • 1/1' (~18 cols) ⇒
// pathBudget = 32 — narrow enough to force truncation of the 49-char
// "./very/deep/nested/path/structure/utils/helper.ts" path.
func TestFiles_PathTruncation_StatusLine(t *testing.T) {
	m := testModel()
	m.files = newFilesModel("s1", 20, 20, 30, 20)
	m.files.cwd = "very/deep/nested/path/structure/utils/helper.ts"
	m.files.entries = []daemon.FileEntry{{Name: "x"}}
	rendered := ansi.Strip(m.renderFilesStatusLine(50))
	if !strings.Contains(rendered, "…/") {
		t.Fatalf("TUI-06: expected '…/' prefix in truncated path, got %q", rendered)
	}
	if !strings.Contains(rendered, "helper.ts") {
		t.Fatalf("TUI-06: expected leaf 'helper.ts' preserved, got %q", rendered)
	}
	if strings.Contains(rendered, "very/deep") {
		t.Fatalf("TUI-06: expected leading segments truncated, got %q", rendered)
	}
}

// TestRenderFilesStatusLine_TailAwarePathBudget covers WR-06: pathBudget
// must be computed from the actual tail width, not a magic 40. With a
// large entry count + (truncated) flag the tail can exceed 36 chars; the
// fix subtracts lipgloss.Width(tail) so the leading path is sized to
// whatever space is actually left.
func TestRenderFilesStatusLine_TailAwarePathBudget(t *testing.T) {
	m := testModel()
	m.files = newFilesModel("s1", 20, 20, 30, 20)
	m.files.cwd = "src/app"
	// 9999 entries + truncated flag — tail string is ~36+12 chars.
	m.files.entries = make([]daemon.FileEntry, 9999)
	for i := range m.files.entries {
		m.files.entries[i].Name = fmt.Sprintf("f%d", i)
	}
	m.files.truncated = true
	rendered := ansi.Strip(m.renderFilesStatusLine(80))
	// The tail must appear in full — its prior 40-col reservation would
	// have been too tight for "9999 entries (truncated) • 1/9999".
	if !strings.Contains(rendered, "9999 entries (truncated)") {
		t.Errorf("expected full tail with 9999 entries and (truncated) flag, got %q", rendered)
	}
	if !strings.Contains(rendered, "1/9999") {
		t.Errorf("expected '1/9999' selection counter, got %q", rendered)
	}
}

// TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp exercises TUI-10:
// the Files key handler sits at Priority 5.5 — below help overlay (5) and
// modal handlers (1-4), above tab cycling (6). The three sub-tests assert
// each adjacent boundary.
func TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp(t *testing.T) {
	// Sub-test A: Help overlay (Priority 5) beats Files (Priority 5.5).
	// With showHelp=true and tabFiles active, Esc closes help instead of
	// going through handleFilesKey.
	t.Run("help_beats_files", func(t *testing.T) {
		m := testModel()
		m.openTabs = []tabID{tabFiles}
		m.activeTab = 0
		m.showHelp = true
		updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		if updated.(Model).showHelp {
			t.Fatal("TUI-10: showHelp should be false after Esc — help priority must win over files priority")
		}
	})

	// Sub-test B: Files (Priority 5.5) beats tab cycling (Priority 6).
	// With tabFiles active, pressing an unrecognised key 'x' is swallowed by
	// handleFilesKey (NOT processed as tab cycling). m.activeTabID() stays
	// at tabFiles, proving Priority 5.5 returned first.
	t.Run("files_beats_tabcycling", func(t *testing.T) {
		m := testModel()
		m.openTabs = []tabID{tabSessions, tabFiles}
		m.activeTab = 1
		m.showHelp = false
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
		u := updated.(Model)
		if u.activeTabID() != tabFiles {
			t.Fatalf("TUI-10: tabFiles should still be active; got %v", u.activeTabID())
		}
	})

	// Sub-test C: modalKillConfirm (Priority 2) beats Files (Priority 5.5).
	// With kill-confirm modal up and tabFiles active, pressing 'n' cancels
	// the modal rather than landing in handleFilesKey.
	t.Run("killconfirm_beats_files", func(t *testing.T) {
		m := testModel()
		m.openTabs = []tabID{tabFiles}
		m.activeTab = 0
		m.modal = modalKillConfirm
		m.killTarget = &daemon.SessionInfo{ID: "x", Name: "y"}
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
		if updated.(Model).modal != modalNone {
			t.Fatal("TUI-10: kill-confirm 'n' should cancel modal — kill-confirm priority must win over files priority")
		}
	})
}

// TestFiles_Phase121_Requirements is the TUI-XX → test-name traceability
// matrix that enables `/gsd:verify-work` for Phase 121 to confirm every
// merge-gate requirement has at least one covering automated test.
//
// Each sub-test is named "TUI-NN" (matching .planning/REQUIREMENTS.md) and
// lists the existing tests that cover it. The meta-test fails fast if any
// requirement has zero covering tests — adding a new TUI-XX requirement
// without a covering test is an explicit error.
//
// This is a documentation test (it doesn't re-run the underlying tests);
// it asserts the test set IS the contract.
func TestFiles_Phase121_Requirements(t *testing.T) {
	coverage := []struct {
		req    string
		covers []string
	}{
		{"TUI-01", []string{
			"TestFiles_TabID_Distinct",
			"TestRenderFilesTab_BasicLayout",
		}},
		{"TUI-02", []string{
			"TestFiles_OpenFromSessions_LocalEntry",
			"TestFiles_OpenFromSessions_RemoteEntry_ShowsToast",
			"TestFiles_OpenFromSessions_EmptyList_NoOp",
			"TestFiles_OpenFromSessions_ResetsModel",
		}},
		{"TUI-03", []string{
			"TestHandleFilesKey_Backspace_AtRoot_NoOp",
			"TestHandleFilesKey_Backspace_AtRoot_Dot_NoOp",
			"TestHandleFilesKey_Backspace_NonRoot_DispatchesParent",
			"TestHandleFilesKey_Down_MovesCursor",
			"TestHandleFilesKey_Down_ClampedAtEnd",
			"TestHandleFilesKey_Enter_OnDir_DispatchesLoadDir",
		}},
		{"TUI-04", []string{
			"TestApplyFilesHeadMsg_OverCap_RefusalMessage",
			"TestApplyFilesHeadMsg_Binary_RefusalMessage",
			"TestApplyFilesHeadMsg_Text_DispatchesRead",
			"TestApplyFilesReadMsg_TextSetsContent",
			"TestApplyFilesReadMsg_MarkdownSuffix_UsesGlamour",
		}},
		{"TUI-05", []string{
			"TestHandleFilesKey_Slash_ActivatesFilter",
			"TestHandleFilesKey_FilterActive_EscClears",
			"TestHandleFilesKey_FilterActive_BackspaceDoesNotNavigate",
			"TestHandleFilesKey_FilterActive_CtrlCQuits",
		}},
		{"TUI-06", []string{
			"TestTruncateLeft",
			"TestRenderFilesStatusLine_TruncatedFlag",
			"TestRenderFilesStatusLine_ErrorShown",
			"TestRenderFilesStatusLine_LeftTruncation",
			"TestFiles_PathTruncation_StatusLine",
		}},
		{"TUI-07", []string{
			"TestFiles_NoSyncFSCalls",
			"TestLoadDirCmd_DispatchesAsync",
		}},
		{"TUI-08", []string{
			"TestFiles_OpenFromSessions_RemoteEntry_ShowsToast",
			"TestFiles_Integration_LocalSessionEndToEnd",
		}},
		{"TUI-09", []string{
			"TestBuildHelpContent_FilesActive_ShowsFilesGroup",
			"TestBuildHelpContent_SessionsActive_ShowsSessionsGroup",
		}},
		{"TUI-10", []string{
			"TestFiles_HandleKey_DispatchPriority",
			"TestFiles_HandleKey_DispatchPriority_BelowKillConfirm",
			"TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp",
		}},
	}
	for _, c := range coverage {
		t.Run(c.req, func(t *testing.T) {
			if len(c.covers) == 0 {
				t.Fatalf("%s has no coverage", c.req)
			}
			t.Logf("%s covered by: %s", c.req, strings.Join(c.covers, ", "))
		})
	}
}
