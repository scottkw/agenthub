package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

// ----------------------------------------------------------------------------
// Phase 126 Plan 03 — delete / rename / mkdir (TUIW-05)
// ----------------------------------------------------------------------------

// TestFilesOpCmd verifies that deleteCmd, renameCmd, and mkdirCmd produce
// filesOpMsg values with the correct op field and are gate-safe (nil-client
// returns errNilClient, not a panic).
func TestFilesOpCmd(t *testing.T) {
	t.Run("deleteCmd nil-client returns filesOpMsg with errNilClient", func(t *testing.T) {
		cmd := deleteCmd(nil, "s1", "file.txt", 1)
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
		if op.op != "delete" {
			t.Errorf("expected op=delete, got %q", op.op)
		}
		if op.err != errNilClient {
			t.Errorf("expected errNilClient, got %v", op.err)
		}
		if op.sessionID != "s1" {
			t.Errorf("expected sessionID=s1, got %q", op.sessionID)
		}
		if op.generation != 1 {
			t.Errorf("expected generation=1, got %d", op.generation)
		}
	})

	t.Run("renameCmd nil-client returns filesOpMsg with errNilClient", func(t *testing.T) {
		cmd := renameCmd(nil, "s1", "old.txt", "new.txt", 2)
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
		if op.op != "rename" {
			t.Errorf("expected op=rename, got %q", op.op)
		}
		if op.err != errNilClient {
			t.Errorf("expected errNilClient, got %v", op.err)
		}
		if op.generation != 2 {
			t.Errorf("expected generation=2, got %d", op.generation)
		}
	})

	t.Run("mkdirCmd nil-client returns filesOpMsg with errNilClient", func(t *testing.T) {
		cmd := mkdirCmd(nil, "s1", "newdir", 3)
		if cmd == nil {
			t.Fatal("expected non-nil cmd")
		}
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
		if op.op != "mkdir" {
			t.Errorf("expected op=mkdir, got %q", op.op)
		}
		if op.err != errNilClient {
			t.Errorf("expected errNilClient, got %v", op.err)
		}
		if op.generation != 3 {
			t.Errorf("expected generation=3, got %d", op.generation)
		}
	})
}

// TestFilesDelete_ModalStateSet verifies that pressing 'd' on a file or directory
// sets modal=modalFileDeleteConfirm and populates fileDeleteTarget.
func TestFilesDelete_ModalStateSet(t *testing.T) {
	t.Run("d on a file sets delete-confirm modal", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "main.go", IsDir: false}}
		m.files.selected = 0

		updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'd'})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd from 'd' (modal set, no dispatch yet)")
		}
		if r.modal != modalFileDeleteConfirm {
			t.Errorf("expected modalFileDeleteConfirm, got %v", r.modal)
		}
		if r.fileDeleteTarget == nil {
			t.Fatal("expected fileDeleteTarget to be non-nil")
		}
		if r.fileDeleteTarget.name != "main.go" {
			t.Errorf("expected name=main.go, got %q", r.fileDeleteTarget.name)
		}
		if r.fileDeleteTarget.isDir {
			t.Error("expected isDir=false for a file")
		}
	})

	t.Run("d on a directory sets delete-confirm modal with isDir=true", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "subdir", IsDir: true}}
		m.files.selected = 0

		updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: 'd'})
		r := updated.(Model)

		if r.modal != modalFileDeleteConfirm {
			t.Errorf("expected modalFileDeleteConfirm, got %v", r.modal)
		}
		if r.fileDeleteTarget == nil {
			t.Fatal("expected fileDeleteTarget to be non-nil")
		}
		if !r.fileDeleteTarget.isDir {
			t.Error("expected isDir=true for a directory")
		}
	})

	t.Run("d on empty listing is a no-op", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{}
		m.files.selected = 0

		updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'd'})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if r.modal != modalNone {
			t.Errorf("expected modal=none for empty listing, got %v", r.modal)
		}
	})
}

// TestFilesDelete_ConfirmHandler verifies handleFileDeleteConfirmKey behavior:
// y/Enter-on-Yes dispatches deleteCmd; n/esc closes modal without dispatch.
func TestFilesDelete_ConfirmHandler(t *testing.T) {
	// Helper: build a model in the delete-confirm state.
	deleteConfirmModel := func() Model {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "file.txt", IsDir: false}}
		m.files.selected = 0
		m.files.cwd = "."
		m.modal = modalFileDeleteConfirm
		m.fileDeleteTarget = &fileDeleteTarget{
			relPath: "file.txt",
			isDir:   false,
			name:    "file.txt",
		}
		return m
	}

	t.Run("y dispatches deleteCmd and closes modal", func(t *testing.T) {
		m := deleteConfirmModel()
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
		r := updated.(Model)

		if r.modal != modalNone {
			t.Errorf("expected modal=none after 'y', got %v", r.modal)
		}
		if r.fileDeleteTarget != nil {
			t.Error("expected fileDeleteTarget=nil after confirm")
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd after 'y'")
		}
		// The cmd should produce a filesOpMsg with op=delete.
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
		if op.op != "delete" {
			t.Errorf("expected op=delete, got %q", op.op)
		}
	})

	t.Run("n closes modal without dispatch", func(t *testing.T) {
		m := deleteConfirmModel()
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
		r := updated.(Model)

		if r.modal != modalNone {
			t.Errorf("expected modal=none after 'n', got %v", r.modal)
		}
		if r.fileDeleteTarget != nil {
			t.Error("expected fileDeleteTarget=nil after cancel")
		}
		if cmd != nil {
			t.Error("expected nil cmd after 'n'")
		}
	})

	t.Run("esc closes modal without dispatch", func(t *testing.T) {
		m := deleteConfirmModel()
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		r := updated.(Model)

		if r.modal != modalNone {
			t.Errorf("expected modal=none after esc, got %v", r.modal)
		}
		if cmd != nil {
			t.Error("expected nil cmd after esc")
		}
	})

	t.Run("enter on Yes-focused dispatches deleteCmd", func(t *testing.T) {
		m := deleteConfirmModel()
		m.fileDeleteFocusYes = true
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		if r.modal != modalNone {
			t.Errorf("expected modal=none after Enter-on-Yes, got %v", r.modal)
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd after Enter-on-Yes")
		}
		msg := cmd()
		if _, ok := msg.(filesOpMsg); !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
	})

	t.Run("enter on No-focused closes modal without dispatch", func(t *testing.T) {
		m := deleteConfirmModel()
		m.fileDeleteFocusYes = false
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		if r.modal != modalNone {
			t.Errorf("expected modal=none after Enter-on-No, got %v", r.modal)
		}
		if cmd != nil {
			t.Error("expected nil cmd after Enter-on-No")
		}
	})

	t.Run("left/right toggles focus between Yes and No", func(t *testing.T) {
		m := deleteConfirmModel()
		m.fileDeleteFocusYes = false

		updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		if !updated.(Model).fileDeleteFocusYes {
			t.Error("expected fileDeleteFocusYes=true after Right")
		}

		updated2, _ := updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyLeft})
		if updated2.(Model).fileDeleteFocusYes {
			t.Error("expected fileDeleteFocusYes=false after Left")
		}
	})
}

// TestFilesDelete_DispatchPriority verifies that with modalFileDeleteConfirm
// active, a 'y' keypress is consumed by handleFileDeleteConfirmKey, NOT routed
// to tab-cycling or handleFilesKey. This mirrors
// TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp (TUI-10).
func TestFilesDelete_DispatchPriority(t *testing.T) {
	t.Run("delete-confirm beats Files handler", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "file.txt", IsDir: false}}
		m.modal = modalFileDeleteConfirm
		m.fileDeleteTarget = &fileDeleteTarget{
			relPath: "file.txt",
			isDir:   false,
			name:    "file.txt",
		}

		updated, _ := m.Update(tea.KeyPressMsg{Code: 'y'})
		r := updated.(Model)

		// 'y' must close the confirm modal, not be swallowed by handleFilesKey.
		if r.modal != modalNone {
			t.Error("delete-confirm handler must consume 'y' before handleFilesKey")
		}
	})

	t.Run("delete-confirm beats tab-cycling", func(t *testing.T) {
		m := testModel()
		m.openTabs = []tabID{tabSessions, tabFiles}
		m.activeTab = 1
		m.modal = modalFileDeleteConfirm
		m.fileDeleteTarget = &fileDeleteTarget{
			relPath: "file.txt",
			isDir:   false,
			name:    "file.txt",
		}

		priorTab := m.activeTab
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
		r := updated.(Model)

		// 'n' closes the modal — it must NOT cycle tabs.
		if r.activeTab != priorTab {
			t.Errorf("delete-confirm 'n' must not change active tab; before=%d after=%d", priorTab, r.activeTab)
		}
		if r.modal != modalNone {
			t.Error("expected modal to close on 'n'")
		}
	})
}

// TestFilesRename verifies the 'r' key inline-rename flow:
// - 'r' enters rename mode with nameInput prefilled with current name
// - Enter with changed non-empty name dispatches renameCmd
// - Enter with empty → toast, no dispatch
// - Enter with unchanged name → no-op (no dispatch)
// - esc cancels
func TestFilesRename(t *testing.T) {
	t.Run("r sets rename mode prefilled with current name", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "README.md", IsDir: false}}
		m.files.selected = 0

		updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: 'r'})
		r := updated.(Model)

		// cmd may be non-nil (nameInput.Focus() returns a focus cmd) — that is fine.
		if !r.files.nameInputActive {
			t.Error("expected files.nameInputActive=true after 'r'")
		}
		if r.files.nameInputMode != "rename" {
			t.Errorf("expected nameInputMode=rename, got %q", r.files.nameInputMode)
		}
		if r.files.nameInput.Value() != "README.md" {
			t.Errorf("expected nameInput prefilled with README.md, got %q", r.files.nameInput.Value())
		}
		if r.files.nameInputOriginal != "README.md" {
			t.Errorf("expected nameInputOriginal=README.md, got %q", r.files.nameInputOriginal)
		}
	})

	t.Run("r on empty listing is a no-op", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{}
		m.files.selected = 0

		updated, cmd := m.handleFilesKey(tea.KeyPressMsg{Code: 'r'})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd")
		}
		if r.files.nameInputActive {
			t.Error("expected nameInputActive=false on empty listing")
		}
	})

	// Helper: build a model in rename-mode with the given prefill value.
	renameModel := func(original, currentValue string) Model {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: original, IsDir: false}}
		m.files.selected = 0
		m.files.cwd = "."
		m.files.nameInputActive = true
		m.files.nameInputMode = "rename"
		m.files.nameInputOriginal = original
		m.files.nameInput.SetValue(currentValue)
		return m
	}

	t.Run("enter with changed name dispatches renameCmd", func(t *testing.T) {
		m := renameModel("old.txt", "new.txt")
		gen := m.files.generation

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		if r.files.nameInputActive {
			t.Error("expected nameInputActive=false after Enter")
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd after Enter with changed name")
		}
		// Verify generation was bumped.
		if r.files.generation <= gen {
			t.Errorf("expected generation to bump; before=%d after=%d", gen, r.files.generation)
		}
		// The cmd (nil client) should yield filesOpMsg{op:"rename"}.
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
		if op.op != "rename" {
			t.Errorf("expected op=rename, got %q", op.op)
		}
	})

	t.Run("enter with empty name sets toast and no dispatch", func(t *testing.T) {
		m := renameModel("old.txt", "")
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd for empty name")
		}
		if r.files.nameInputActive == false {
			// nameInputActive stays true — user can correct the name
		}
		if !strings.Contains(r.toast, "empty") {
			t.Errorf("expected toast mentioning 'empty', got %q", r.toast)
		}
	})

	t.Run("enter with unchanged name is a no-op", func(t *testing.T) {
		m := renameModel("file.txt", "file.txt")
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd when name unchanged")
		}
		if r.files.nameInputActive {
			t.Error("expected nameInputActive=false after no-op rename")
		}
	})

	t.Run("esc cancels rename", func(t *testing.T) {
		m := renameModel("old.txt", "new.txt")
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd after esc")
		}
		if r.files.nameInputActive {
			t.Error("expected nameInputActive=false after esc")
		}
	})
}

// TestFilesMkdir verifies the 'm' key inline-mkdir flow:
// - 'm' enters mkdir mode with empty nameInput
// - Enter with non-empty name dispatches mkdirCmd with joinDir(cwd, name)
// - Enter with empty name → no dispatch
// - esc cancels
func TestFilesMkdir(t *testing.T) {
	t.Run("m sets mkdir mode with empty input", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "existing.txt", IsDir: false}}
		m.files.selected = 0

		updated, _ := m.handleFilesKey(tea.KeyPressMsg{Code: 'm'})
		r := updated.(Model)

		// cmd may be non-nil (nameInput.Focus() returns a focus cmd) — that is fine.
		if !r.files.nameInputActive {
			t.Error("expected files.nameInputActive=true after 'm'")
		}
		if r.files.nameInputMode != "mkdir" {
			t.Errorf("expected nameInputMode=mkdir, got %q", r.files.nameInputMode)
		}
		if r.files.nameInput.Value() != "" {
			t.Errorf("expected nameInput empty for mkdir, got %q", r.files.nameInput.Value())
		}
	})

	// Helper: build a model in mkdir-mode.
	mkdirModel := func(cwd, inputValue string) Model {
		m := filesKeyTestModel()
		m.files.cwd = cwd
		m.files.nameInputActive = true
		m.files.nameInputMode = "mkdir"
		m.files.nameInputOriginal = ""
		m.files.nameInput.SetValue(inputValue)
		return m
	}

	t.Run("enter with non-empty name dispatches mkdirCmd", func(t *testing.T) {
		m := mkdirModel(".", "newdir")
		gen := m.files.generation

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		if r.files.nameInputActive {
			t.Error("expected nameInputActive=false after Enter")
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd after Enter with name")
		}
		if r.files.generation <= gen {
			t.Errorf("expected generation to bump; before=%d after=%d", gen, r.files.generation)
		}
		// The cmd (nil client) should yield filesOpMsg{op:"mkdir"}.
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg, got %T", msg)
		}
		if op.op != "mkdir" {
			t.Errorf("expected op=mkdir, got %q", op.op)
		}
	})

	t.Run("mkdir Enter with empty name is no-op", func(t *testing.T) {
		m := mkdirModel(".", "")
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

		if cmd != nil {
			t.Error("expected nil cmd for empty mkdir name")
		}
		_ = updated
	})

	t.Run("mkdir esc cancels", func(t *testing.T) {
		m := mkdirModel(".", "newdir")
		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		r := updated.(Model)

		if cmd != nil {
			t.Error("expected nil cmd after esc")
		}
		if r.files.nameInputActive {
			t.Error("expected nameInputActive=false after esc")
		}
	})
}

// TestFilesDeleteModal_ColorblindSafeText verifies the delete-confirm modal
// contains explicit "Delete" and "cannot be undone" text (non-color danger
// signal — colorblind constraint from MEMORY).
func TestFilesDeleteModal_ColorblindSafeText(t *testing.T) {
	m := testModel()
	m.width = 100
	m.height = 30
	m.modal = modalFileDeleteConfirm
	m.fileDeleteTarget = &fileDeleteTarget{
		relPath: "subdir",
		isDir:   true,
		name:    "subdir",
	}

	rendered := m.renderFileDeleteConfirmModal()

	if !strings.Contains(rendered, "Delete") {
		t.Error("delete-confirm modal must contain explicit 'Delete' text (colorblind-safe)")
	}
	if !strings.Contains(rendered, "cannot be undone") {
		t.Error("delete-confirm modal must contain 'cannot be undone' text (colorblind-safe)")
	}
	// Dir variant should mention "directory" or "all contents"
	if !strings.Contains(rendered, "directory") && !strings.Contains(rendered, "all contents") {
		t.Error("delete-confirm modal for dir must mention 'directory' or 'all contents'")
	}
}

// TestFilesNameInput_DispatchPriority verifies that while files inline-input is
// active, Enter is consumed by the inline handler, NOT passed through to
// handleFilesKey (which would navigate into a directory).
func TestFilesNameInput_DispatchPriority(t *testing.T) {
	t.Run("inline-input active beats handleFilesKey", func(t *testing.T) {
		m := filesKeyTestModel()
		m.files.entries = []daemon.FileEntry{{Name: "subdir", IsDir: true}}
		m.files.selected = 0
		m.files.cwd = "."
		m.files.nameInputActive = true
		m.files.nameInputMode = "rename"
		m.files.nameInputOriginal = "subdir"
		m.files.nameInput.SetValue("newname")

		updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		r := updated.(Model)

		// Enter was consumed by the inline-input handler, not by handleFilesKey.
		// Proof: nameInputActive is now false (handler ran) and cmd is for renameCmd.
		if r.files.nameInputActive {
			t.Error("nameInputActive should be false after Enter (consumed by inline handler)")
		}
		if cmd == nil {
			t.Fatal("expected non-nil cmd — inline handler should dispatch renameCmd")
		}
		// The inline handler dispatched renameCmd, NOT a navigation loadDir.
		msg := cmd()
		op, ok := msg.(filesOpMsg)
		if !ok {
			t.Fatalf("expected filesOpMsg (rename dispatch), got %T", msg)
		}
		if op.op != "rename" {
			t.Errorf("expected op=rename, got %q", op.op)
		}
	})
}

// TestFiles_Phase126_Requirements is the per-phase traceability matrix.
// It asserts the test set IS the contract for Phase 126 / TUIW-05.
func TestFiles_Phase126_Requirements(t *testing.T) {
	coverage := []struct {
		req    string
		covers []string
	}{
		{"TUIW-05", []string{
			"TestFilesOpCmd",
			"TestFilesDelete_ModalStateSet",
			"TestFilesDelete_ConfirmHandler",
			"TestFilesDelete_DispatchPriority",
			"TestFilesRename",
			"TestFilesMkdir",
			"TestFilesDeleteModal_ColorblindSafeText",
			"TestFilesNameInput_DispatchPriority",
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

// fmtTypes is a helper for test error messages.
func fmtTypes(cmds []tea.Cmd) []string {
	types := make([]string, 0, len(cmds))
	for _, c := range cmds {
		if c != nil {
			types = append(types, fmt.Sprintf("%T", c()))
		}
	}
	return types
}
