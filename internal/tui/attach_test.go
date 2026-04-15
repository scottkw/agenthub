package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)

func TestAttachCmd_ImplementsExecCommand(t *testing.T) {
	cmd := &attachCmd{}
	// Verify it satisfies tea.ExecCommand interface at compile time.
	var _ tea.ExecCommand = cmd
	_ = cmd // use cmd to satisfy go vet
}

func TestAttachCmd_SetStdinStdout(t *testing.T) {
	cmd := &attachCmd{}
	// SetStdin/SetStdout/SetStderr should not panic with nil values.
	cmd.SetStdin(nil)
	cmd.SetStdout(nil)
	cmd.SetStderr(nil)
}

func TestUpdate_AttachDispatch(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "test", CLI: "claude", Status: "running"},
	}
	m.selected = 0

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from Enter key (tea.Exec)")
	}
}

func TestUpdate_AttachErroredSession(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "test", CLI: "claude", Status: "errored"},
	}
	m.selected = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.toast != "Session not available" {
		t.Errorf("expected 'Session not available' toast, got %q", result.toast)
	}
}

func TestUpdate_AttachDone(t *testing.T) {
	m := testModel()

	// Success path — should return refresh command, no error toast.
	updated, cmd := m.Update(attachDoneMsg{err: nil})
	_ = updated.(Model)
	if cmd == nil {
		t.Error("expected refresh cmd after attachDone")
	}

	// Error path — should set error toast.
	updated2, _ := m.Update(attachDoneMsg{err: fmt.Errorf("connection lost")})
	result2 := updated2.(Model)
	if result2.toast != "Attach error: connection lost" {
		t.Errorf("expected error toast, got %q", result2.toast)
	}
	if result2.toastKind != toastError {
		t.Errorf("expected toastError kind")
	}
}
