package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/files"
)

// TestResolveEditor verifies the $EDITOR→$VISUAL→nano→vim→vi resolution chain.
//
// Case 1 (resolved): $EDITOR set to a known-present binary ("go") — expects
// non-empty path that ends with "go".
// Case 2 (empty): all env vars unset AND PATH stripped to a directory that
// has no editor — expects "".
func TestResolveEditor(t *testing.T) {
	t.Run("resolves $EDITOR when set to known binary", func(t *testing.T) {
		// "go" is always present on PATH in this repo's dev environment.
		goPath, err := exec.LookPath("go")
		if err != nil {
			t.Skip("go not on PATH — skipping EDITOR resolve test")
		}

		t.Setenv("EDITOR", "go")
		t.Setenv("VISUAL", "")

		got := resolveEditor()
		if got == "" {
			t.Fatal("resolveEditor: expected non-empty path with EDITOR=go, got empty string")
		}
		if got != goPath {
			t.Errorf("resolveEditor: got %q, want %q", got, goPath)
		}
	})

	t.Run("returns empty string when no editor resolves", func(t *testing.T) {
		// Unset both env vars and point PATH at an empty temp directory so
		// none of the fallback binaries (nano, vim, vi) can be found.
		emptyDir := t.TempDir()

		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")
		t.Setenv("PATH", emptyDir)

		got := resolveEditor()
		if got != "" {
			t.Errorf("resolveEditor: expected empty string with no editors on PATH, got %q", got)
		}
	})

	t.Run("resolution order is EDITOR VISUAL nano vim vi", func(t *testing.T) {
		// Create a fake "myeditor" in a temp dir to verify order: $EDITOR wins
		// over $VISUAL wins over fallbacks.
		dir := t.TempDir()
		for _, name := range []string{"myeditor", "myvisual", "nano", "vim", "vi"} {
			path := dir + "/" + name
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
				t.Fatalf("create fake %s: %v", name, err)
			}
		}

		t.Setenv("EDITOR", "myeditor")
		t.Setenv("VISUAL", "myvisual")
		oldPath := os.Getenv("PATH")
		t.Setenv("PATH", dir+":"+oldPath)

		got := resolveEditor()
		if !strings.HasSuffix(got, "/myeditor") {
			t.Errorf("resolveEditor: $EDITOR should win, got %q", got)
		}

		// Now unset EDITOR — $VISUAL should win.
		t.Setenv("EDITOR", "")
		got2 := resolveEditor()
		if !strings.HasSuffix(got2, "/myvisual") {
			t.Errorf("resolveEditor: $VISUAL should win when $EDITOR unset, got %q", got2)
		}
	})
}

// TestHandleFilesKey_Edit verifies the 'e' key branch behavior in handleFilesKey.
//
// Covered behaviors (TUIW-02, TUIW-03):
//  1. Pressing 'e' on a regular file with a resolvable editor bumps generation
//     and returns a non-nil cmd (editFetchCmd).
//  2. Pressing 'e' on a directory is a no-op.
//  3. Pressing 'e' with no resolvable editor (PATH stripped) sets m.files.err
//     to the exact locked error copy and returns no cmd.
func TestHandleFilesKey_Edit(t *testing.T) {
	t.Run("e on a file with editor set dispatches editFetchCmd", func(t *testing.T) {
		// Set up a fake editor on PATH.
		dir := t.TempDir()
		fakePath := dir + "/fakeeditor"
		if err := os.WriteFile(fakePath, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("create fake editor: %v", err)
		}
		t.Setenv("EDITOR", "fakeeditor")
		t.Setenv("VISUAL", "")
		t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "main.go", IsDir: false}}
		m.files.selected = 0
		gen := m.files.generation

		updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'e'})
		r := updated.(Model)

		if cmd == nil {
			t.Fatal("expected non-nil cmd from 'e' on a file with editor set")
		}
		if r.files.generation <= gen {
			t.Errorf("expected generation to bump; before=%d after=%d", gen, r.files.generation)
		}
	})

	t.Run("e on a directory is a no-op", func(t *testing.T) {
		dir := t.TempDir()
		fakePath := dir + "/fakeeditor"
		if err := os.WriteFile(fakePath, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatalf("create fake editor: %v", err)
		}
		t.Setenv("EDITOR", "fakeeditor")
		t.Setenv("VISUAL", "")
		t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "subdir", IsDir: true}}
		m.files.selected = 0

		_, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'e'})
		if cmd != nil {
			t.Error("expected nil cmd from 'e' on a directory")
		}
	})

	t.Run("e with no editor sets exact locked error and returns nil cmd", func(t *testing.T) {
		emptyDir := t.TempDir()
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")
		t.Setenv("PATH", emptyDir)

		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "README.md", IsDir: false}}
		m.files.selected = 0

		updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'e'})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd when no editor resolves")
		}
		if r.files.err == nil {
			t.Fatal("expected m.files.err to be set when no editor resolves")
		}
		const wantErr = "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)."
		if r.files.err.Error() != wantErr {
			t.Errorf("exact locked error copy mismatch:\ngot:  %q\nwant: %q", r.files.err.Error(), wantErr)
		}
	})
}

// TestEditorExit_RefreshesUnconditionally verifies TUIW-04: the editorExitMsg
// handler ALWAYS batches tea.ClearScreen + editWriteBackCmd, regardless of
// whether exitErr is nil or non-nil.
//
// CR-01 FIX: The batch no longer contains loadDirCmd. applyEditorExitMsg now
// only dispatches the write-back; applyFilesOpMsg is responsible for the
// subsequent listing refresh (success or error path). This ensures write-back
// errors are never discarded as stale.
//
// This test specifically asserts that write-back is present even when the
// editor exits non-zero (exitErr != nil). A regression that gates write-back
// on exitErr==nil would fail this test.
func TestEditorExit_RefreshesUnconditionally(t *testing.T) {
	// unpackBatch calls the returned cmd and asserts it is a tea.BatchMsg,
	// then returns the list of cmds inside the batch.
	unpackBatch := func(t *testing.T, cmd tea.Cmd) []tea.Cmd {
		t.Helper()
		if cmd == nil {
			t.Fatal("expected non-nil cmd from editorExitMsg handler")
		}
		msg := cmd()
		batch, ok := msg.(tea.BatchMsg)
		if !ok {
			t.Fatalf("expected tea.BatchMsg from editorExitMsg handler, got %T", msg)
		}
		return []tea.Cmd(batch)
	}

	// assertBatchContainsMsgType checks that the batch contains at least one cmd
	// whose invocation returns a message of the expected type (by format string).
	assertBatchContainsMsgType := func(t *testing.T, cmds []tea.Cmd, typeName string) {
		t.Helper()
		for _, c := range cmds {
			if c == nil {
				continue
			}
			msg := c()
			if fmt.Sprintf("%T", msg) == typeName {
				return
			}
		}
		t.Errorf("batch does not contain a cmd yielding %s; got types: %v",
			typeName, func() []string {
				types := make([]string, 0, len(cmds))
				for _, c := range cmds {
					if c != nil {
						types = append(types, fmt.Sprintf("%T", c()))
					}
				}
				return types
			}())
	}

	// assertBatchNotContainsMsgType checks that the batch does NOT contain a cmd
	// of the named type (to guard the CR-01 contract: loadDir is NOT in the
	// first batch, only in the second after write-back resolves).
	assertBatchNotContainsMsgType := func(t *testing.T, cmds []tea.Cmd, typeName string) {
		t.Helper()
		for _, c := range cmds {
			if c == nil {
				continue
			}
			msg := c()
			if fmt.Sprintf("%T", msg) == typeName {
				t.Errorf("batch unexpectedly contains %s (CR-01: loadDir must be driven by applyFilesOpMsg, not applyEditorExitMsg)", typeName)
				return
			}
		}
	}

	// Create a temporary file to act as tmpPath (editWriteBackCmd reads it).
	tmpFile, err := os.CreateTemp("", "test-edit-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Write([]byte("hello"))
	tmpFile.Close()
	// Leave the file in place — editWriteBackCmd will remove it (defer os.Remove).

	t.Run("exitErr==nil: batch contains ClearScreen + write-back (no premature loadDir)", func(t *testing.T) {
		// Recreate the temp file since a prior sub-test may have removed it.
		if err := os.WriteFile(tmpPath, []byte("hello"), 0600); err != nil {
			t.Fatalf("recreate temp file: %v", err)
		}

		m := testModel()
		m.files.sessionID = "s1"
		m.files.cwd = "."
		m.files.generation = 5

		msg := editorExitMsg{
			sessionID:  "s1",
			generation: 5,
			tmpPath:    tmpPath,
			relPath:    "README.md",
			exitErr:    nil,
		}
		_, cmd := m.Update(msg)
		cmds := unpackBatch(t, cmd)

		// ClearScreen is a func that returns tea.clearScreenMsg (unexported type
		// in the bubbletea package). Identify it by its package-qualified type name.
		assertBatchContainsMsgType(t, cmds, "tea.clearScreenMsg")
		// editWriteBackCmd with nil client returns filesOpMsg.
		assertBatchContainsMsgType(t, cmds, "tui.filesOpMsg")
		// CR-01: loadDirCmd must NOT be in the first batch — it is driven by
		// applyFilesOpMsg after the write-back result lands.
		assertBatchNotContainsMsgType(t, cmds, "tui.filesListMsg")
	})

	t.Run("exitErr!=nil: STILL batches write-back + ClearScreen (unconditional)", func(t *testing.T) {
		// Recreate the temp file since the prior sub-test removed it.
		if err := os.WriteFile(tmpPath, []byte("hello"), 0600); err != nil {
			t.Fatalf("recreate temp file: %v", err)
		}

		m := testModel()
		m.files.sessionID = "s1"
		m.files.cwd = "."
		m.files.generation = 5

		msg := editorExitMsg{
			sessionID:  "s1",
			generation: 5,
			tmpPath:    tmpPath,
			relPath:    "README.md",
			exitErr:    errors.New("editor exited with status 1"),
		}
		_, cmd := m.Update(msg)
		cmds := unpackBatch(t, cmd)

		// KEY ASSERTION: write-back must be present even when exitErr != nil.
		// A regression that gates write-back on exitErr==nil would fail here.
		assertBatchContainsMsgType(t, cmds, "tea.clearScreenMsg")
		assertBatchContainsMsgType(t, cmds, "tui.filesOpMsg") // editWriteBackCmd
		// CR-01: loadDirCmd is NOT in this batch either.
		assertBatchNotContainsMsgType(t, cmds, "tui.filesListMsg")
	})

	// Clean up — both sub-tests may or may not have removed the file.
	os.Remove(tmpPath)
}

// TestEditWriteBack_ErrorSurfaces verifies CR-01: a failed write-back from the
// daemon sets m.files.err (visible to the user) and is NOT silently discarded
// by the staleness guard.
//
// This is the regression test that would have caught the original bug: the
// generation was bumped before the write-back result arrived, making the
// staleness guard always discard it. With the fix, generation is not bumped
// in applyEditorExitMsg; it is only bumped in applyFilesOpMsg after the
// write-back result lands.
func TestEditWriteBack_ErrorSurfaces(t *testing.T) {
	// Build a mock FilesClient whose WriteFile always returns an error.
	writeErr := errors.New("write cap exceeded")
	mock := &mockFilesClient{writeFileErr: writeErr}

	m := testModel()
	m.files.client = mock
	m.files.sessionID = "s1"
	m.files.cwd = "."
	m.files.generation = 5

	// Write a temp file so editWriteBackCmd can read it.
	tmpFile, err := os.CreateTemp("", "cr01-test-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Write([]byte("edited content"))
	tmpFile.Close()

	// Step 1: dispatch the editorExitMsg — this bumps nothing and returns
	// editWriteBackCmd in the batch.
	exitMsg := editorExitMsg{
		sessionID:  "s1",
		generation: 5,
		tmpPath:    tmpPath,
		relPath:    "README.md",
		exitErr:    nil,
	}
	m1, cmd1 := m.Update(exitMsg)
	if cmd1 == nil {
		t.Fatal("expected non-nil cmd after editorExitMsg")
	}

	// Step 2: run the batch to collect the write-back cmd.
	batchMsg := cmd1()
	batch, ok := batchMsg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected BatchMsg, got %T", batchMsg)
	}

	// Find and execute the editWriteBackCmd (it returns filesOpMsg).
	var writeBackMsg tea.Msg
	for _, c := range []tea.Cmd(batch) {
		if c == nil {
			continue
		}
		msg := c()
		if _, isOp := msg.(filesOpMsg); isOp {
			writeBackMsg = msg
			break
		}
	}
	if writeBackMsg == nil {
		t.Fatal("batch did not contain a filesOpMsg (editWriteBackCmd)")
	}

	// Step 3: deliver the write-back result to the model.
	m2, _ := m1.(Model).Update(writeBackMsg)
	r := m2.(Model)

	// CRITICAL ASSERTION (CR-01): the error must be visible to the user.
	if r.toast == "" {
		t.Error("expected m.toast to be set after failed write-back; got empty string")
	}
	if r.toastKind != toastError {
		t.Errorf("expected toastKind=toastError, got %v", r.toastKind)
	}
	if !strings.Contains(r.toast, "write cap exceeded") {
		t.Errorf("toast should mention the write error; got %q", r.toast)
	}
}

// mockFilesClient is a minimal FilesClient implementation for testing write-back
// error surfacing (CR-01). Only WriteFile is exercised; all other methods return
// zero values so the interface is satisfied.
type mockFilesClient struct {
	writeFileErr error
}

func (m *mockFilesClient) ListFiles(_ context.Context, _, _ string) ([]files.FileEntry, bool, error) {
	return nil, false, nil
}
func (m *mockFilesClient) StatFile(_ context.Context, _, _ string) (files.FileEntry, error) {
	return files.FileEntry{}, nil
}
func (m *mockFilesClient) ReadFile(_ context.Context, _, _ string) ([]byte, string, error) {
	return nil, "", nil
}
func (m *mockFilesClient) HeadFile(_ context.Context, _, _ string) (int64, string, time.Time, error) {
	return 0, "", time.Time{}, nil
}
func (m *mockFilesClient) WriteFile(_ context.Context, _, _ string, _ []byte) (files.FileWriteResponse, error) {
	return files.FileWriteResponse{}, m.writeFileErr
}
func (m *mockFilesClient) DeleteFile(_ context.Context, _, _ string) (files.FileOpResponse, error) {
	return files.FileOpResponse{}, nil
}
func (m *mockFilesClient) RenameFile(_ context.Context, _, _, _ string) (files.FileOpResponse, error) {
	return files.FileOpResponse{}, nil
}
func (m *mockFilesClient) MkdirFile(_ context.Context, _, _ string) (files.FileOpResponse, error) {
	return files.FileOpResponse{}, nil
}
