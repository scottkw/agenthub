package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
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
// handler ALWAYS batches tea.ClearScreen + editWriteBackCmd + loadDirCmd,
// regardless of whether exitErr is nil or non-nil.
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

	// assertBatchContains checks that the batch contains at least one cmd whose
	// invocation returns a message of the expected type (by format string).
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

	// Create a temporary file to act as tmpPath (editWriteBackCmd reads it).
	tmpFile, err := os.CreateTemp("", "test-edit-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Write([]byte("hello"))
	tmpFile.Close()
	// Leave the file in place — editWriteBackCmd will remove it (defer os.Remove).

	t.Run("exitErr==nil: batch contains ClearScreen + write-back + loadDir", func(t *testing.T) {
		// Recreate the temp file since the first sub-test may have removed it.
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
		// loadDirCmd with nil client returns filesListMsg.
		assertBatchContainsMsgType(t, cmds, "tui.filesListMsg")
	})

	t.Run("exitErr!=nil: STILL batches write-back + ClearScreen + loadDir (unconditional)", func(t *testing.T) {
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
		assertBatchContainsMsgType(t, cmds, "tui.filesOpMsg")   // editWriteBackCmd
		assertBatchContainsMsgType(t, cmds, "tui.filesListMsg") // loadDirCmd
	})

	// Clean up — both sub-tests may or may not have removed the file.
	os.Remove(tmpPath)
}
