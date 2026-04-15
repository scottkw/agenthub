package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// Update processes messages and returns the updated model and any commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		m.hasDark = msg.IsDark()
		m.styles = newStyles(m.hasDark)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case sessionsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.sessions = msg.sessions
		// Clamp selection if sessions list shrank
		if m.selected >= len(m.sessions) {
			m.selected = max(0, len(m.sessions)-1)
		}
		return m, nil

	case webStatusMsg:
		if msg.err == nil {
			m.webStatus = msg.status
		}
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			fetchSessions(m.client),
			fetchWebStatus(m.client),
			nextTick(),
		)

	case attachDoneMsg:
		cmds := []tea.Cmd{fetchSessions(m.client), fetchWebStatus(m.client)}
		if msg.err != nil {
			m.toast = fmt.Sprintf("Attach error: %s", msg.err)
			m.toastKind = toastError
			m.toastExp = time.Now().Add(3 * time.Second)
		}
		return m, tea.Batch(cmds...)

	case createSessionMsg:
		if msg.err != nil {
			m.toast = fmt.Sprintf("Create failed: %s", msg.err)
			m.toastKind = toastError
			m.toastExp = time.Now().Add(3 * time.Second)
		} else {
			m.toast = "Session created"
			m.toastKind = toastSuccess
			m.toastExp = time.Now().Add(2 * time.Second)
		}
		return m, tea.Batch(fetchSessions(m.client), fetchWebStatus(m.client))

	case killSessionMsg:
		if msg.err != nil {
			m.toast = fmt.Sprintf("Kill failed: %s", msg.err)
			m.toastKind = toastError
			m.toastExp = time.Now().Add(3 * time.Second)
		} else {
			m.toast = "Session killed"
			m.toastKind = toastSuccess
			m.toastExp = time.Now().Add(2 * time.Second)
		}
		return m, tea.Batch(fetchSessions(m.client), fetchWebStatus(m.client))

	case renameSessionMsg:
		if msg.err != nil {
			m.toast = fmt.Sprintf("Rename failed: %s", msg.err)
			m.toastKind = toastError
			m.toastExp = time.Now().Add(3 * time.Second)
		}
		return m, fetchSessions(m.client)
	}

	return m, nil
}

// handleKey dispatches key presses with priority-based routing.
// Priority: editing > kill confirm > new session modal > help > main view.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Priority 1: Inline rename captures all keys
	if m.editing {
		return m.handleRenameKey(msg)
	}
	// Priority 2: Kill confirmation dialog
	if m.modal == modalKillConfirm {
		return m.handleKillConfirmKey(msg)
	}
	// Priority 3: New session modal
	if m.modal == modalNewSession {
		return m.handleNewSessionKey(msg)
	}
	// Priority 4: Help overlay
	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}
	// Priority 5: Main view
	return m.handleMainKey(msg)
}

// handleMainKey handles key presses in the main session list view.
func (m Model) handleMainKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.selected < len(m.sessions)-1 {
			m.selected++
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.selected > 0 {
			m.selected--
		}
		return m, nil

	case key.Matches(msg, m.keys.Top):
		m.selected = 0
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		if len(m.sessions) > 0 {
			m.selected = len(m.sessions) - 1
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, tea.Batch(fetchSessions(m.client), fetchWebStatus(m.client))

	case key.Matches(msg, m.keys.Attach):
		if len(m.sessions) == 0 || m.sessions[m.selected].Status == "errored" {
			m.toast = "Session not available"
			m.toastKind = toastError
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		cmd := &attachCmd{
			client:    m.client,
			sessionID: m.sessions[m.selected].ID,
		}
		return m, tea.Exec(cmd, func(err error) tea.Msg {
			return attachDoneMsg{err: err}
		})

	case key.Matches(msg, m.keys.New):
		return m.openNewSessionModal()

	case key.Matches(msg, m.keys.Kill):
		if len(m.sessions) == 0 {
			return m, nil
		}
		s := m.sessions[m.selected]
		m.modal = modalKillConfirm
		m.killTarget = &s
		m.killFocusYes = false
		return m, nil

	case key.Matches(msg, m.keys.Rename):
		if len(m.sessions) == 0 {
			return m, nil
		}
		s := m.sessions[m.selected]
		m.editing = true
		m.editSessionID = s.ID
		m.editOriginal = s.Name
		m.editInput = textinput.New()
		m.editInput.Prompt = ""
		m.editInput.SetValue(s.Name)
		m.editInput.SetWidth(m.nameColWidth())
		m.editInput.CursorEnd()
		cmd := m.editInput.Focus()
		return m, cmd
	}
	return m, nil
}

// handleRenameKey handles keys when inline rename is active.
func (m Model) handleRenameKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "enter":
		name := strings.TrimSpace(m.editInput.Value())
		if name == "" {
			m.toast = "Name cannot be empty"
			m.toastKind = toastError
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		m.editing = false
		if name == m.editOriginal {
			return m, nil
		}
		m.toast = "Renaming..."
		m.toastKind = toastInfo
		m.toastExp = time.Now().Add(10 * time.Second)
		return m, renameSession(m.client, m.editSessionID, name)
	case s == "esc":
		m.editing = false
		return m, nil
	default:
		var cmd tea.Cmd
		m.editInput, cmd = m.editInput.Update(msg)
		return m, cmd
	}
}

// handleKillConfirmKey handles keys when kill confirmation dialog is open.
func (m Model) handleKillConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "y":
		return m.executeKill()
	case s == "n", s == "esc":
		m.modal = modalNone
		m.killTarget = nil
		return m, nil
	case s == "enter":
		if m.killFocusYes {
			return m.executeKill()
		}
		m.modal = modalNone
		m.killTarget = nil
		return m, nil
	case s == "left", s == "right", s == "h", s == "l", s == "tab":
		m.killFocusYes = !m.killFocusYes
		return m, nil
	}
	return m, nil
}

// executeKill sends the kill command and cleans up modal state.
func (m Model) executeKill() (tea.Model, tea.Cmd) {
	if m.killTarget == nil {
		m.modal = modalNone
		return m, nil
	}
	id := m.killTarget.ID
	m.modal = modalNone
	m.killTarget = nil
	m.toast = "Killing session..."
	m.toastKind = toastInfo
	m.toastExp = time.Now().Add(10 * time.Second)
	return m, killSession(m.client, id)
}

// handleNewSessionKey handles keys when new-session modal is open.
func (m Model) handleNewSessionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	// Intercept modal-level keys BEFORE delegating to textinput
	switch {
	case s == "esc":
		m.modal = modalNone
		return m, nil

	case s == "enter":
		return m.submitNewSession()

	case s == "tab":
		return m.cycleFocus(true)

	case s == "shift+tab":
		return m.cycleFocus(false)

	case (s == "left" || s == "right") && m.focusedField == 0:
		// Agent picker cycling
		m = m.cycleAgent(s == "right")
		return m, nil
	}

	// Delegate to focused textinput (if text field is focused)
	switch m.focusedField {
	case 1:
		var cmd tea.Cmd
		m.dirInput, cmd = m.dirInput.Update(msg)
		return m, cmd
	case 2:
		var cmd tea.Cmd
		m.argsInput, cmd = m.argsInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// openNewSessionModal initializes the new-session modal state.
func (m Model) openNewSessionModal() (tea.Model, tea.Cmd) {
	m.modal = modalNewSession
	m.focusedField = 0
	m.agentIdx = 0

	// Initialize directory input with current working directory
	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "/"
	}

	modalInnerWidth := max(50, min(70, m.width-10)) - 6 // border(2) + padding(4)
	labelWidth := 14                                    // "  Arguments:  " is the widest label

	m.dirInput = textinput.New()
	m.dirInput.Placeholder = cwd
	m.dirInput.SetValue(cwd)
	m.dirInput.SetWidth(modalInnerWidth - labelWidth)
	m.dirInput.Prompt = ""
	m.dirInput.CharLimit = 256

	m.argsInput = textinput.New()
	m.argsInput.Placeholder = "--model opus (optional)"
	m.argsInput.SetWidth(modalInnerWidth - labelWidth)
	m.argsInput.Prompt = ""
	m.argsInput.CharLimit = 256

	// Agent field starts focused (no textinput to focus)
	// Blur both text inputs initially
	m.dirInput.Blur()
	m.argsInput.Blur()

	return m, nil
}

// cycleFocus moves focus between modal form fields (agent/directory/arguments).
func (m Model) cycleFocus(forward bool) (tea.Model, tea.Cmd) {
	// Blur current field
	switch m.focusedField {
	case 1:
		m.dirInput.Blur()
	case 2:
		m.argsInput.Blur()
	}

	if forward {
		m.focusedField = (m.focusedField + 1) % 3
	} else {
		m.focusedField = (m.focusedField + 2) % 3
	}

	// Focus new field
	var cmd tea.Cmd
	switch m.focusedField {
	case 1:
		cmd = m.dirInput.Focus()
	case 2:
		cmd = m.argsInput.Focus()
	}
	return m, cmd
}

// submitNewSession validates form fields and dispatches session creation.
func (m Model) submitNewSession() (tea.Model, tea.Cmd) {
	// Validate agent
	if len(m.detectedCLIs) == 0 {
		m.toast = "Agent is required"
		m.toastKind = toastError
		m.toastExp = time.Now().Add(2 * time.Second)
		return m, nil
	}

	// Validate directory
	workDir := strings.TrimSpace(m.dirInput.Value())
	if workDir == "" {
		m.toast = "Directory is required"
		m.toastKind = toastError
		m.toastExp = time.Now().Add(2 * time.Second)
		return m, nil
	}

	cli := m.detectedCLIs[m.agentIdx].Name
	name := filepath.Base(workDir)

	// Parse arguments (split on spaces, simple)
	argsStr := strings.TrimSpace(m.argsInput.Value())
	var args []string
	if argsStr != "" {
		args = strings.Fields(argsStr)
	}

	m.modal = modalNone
	m.toast = "Creating session..."
	m.toastKind = toastInfo
	m.toastExp = time.Now().Add(10 * time.Second)
	return m, createSession(m.client, cli, name, workDir, args)
}
