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
	qrcode "github.com/skip2/go-qrcode"
)

// sidebarTabs maps sidebar item indices to their corresponding tab IDs.
// This decouples the visual sidebar ordering from the tabID iota values.
var sidebarTabs = [...]tabID{tabHome, tabSessions, tabRemote, tabSettings}

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
		m.rebuildUnifiedList()
		return m, nil

	case remoteSessionsMsg:
		m.remoteSessions = msg.groups
		m.rebuildUnifiedList()
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
			fetchRemoteSessions(m.fetchRemoteFn),
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
	// Priority 4: QR overlay (Phase 78)
	if m.qrSession != nil {
		return m.handleQRKey(msg)
	}
	// Priority 5: Help overlay
	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}
	// Priority 6: Tab cycling (safe here — no modal uses [ or ])
	if key.Matches(msg, m.keys.PrevTab) {
		m.cycleTab(-1)
		return m, nil
	}
	if key.Matches(msg, m.keys.NextTab) {
		m.cycleTab(+1)
		return m, nil
	}
	// Priority 7: Pane-focus-aware dispatch
	if m.panesFocus == focusSidebar {
		return m.handleSidebarKey(msg)
	}
	return m.handleContentKey(msg)
}

// handleQRKey handles keys when the QR overlay is open.
// Esc or q closes the overlay; Q or ctrl+c quits; all other keys are swallowed.
func (m Model) handleQRKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "esc", s == "q":
		m.qrSession = nil
		m.qrContent = ""
		m.qrURL = ""
		return m, nil
	case s == "Q", s == "ctrl+c":
		return m, tea.Quit
	}
	// Swallow all other keys while QR overlay is open
	return m, nil
}

// entryID returns a unique string identifier for a list entry (used to restore selection).
func entryID(e listEntry) string {
	switch e.kind {
	case entryLocal:
		if e.session != nil {
			return e.session.ID
		}
	case entryRemote:
		if e.remote != nil {
			return e.remote.ID
		}
	}
	return ""
}

// handleContentKey handles key presses in the main content pane.
func (m Model) handleContentKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab toggles focus to sidebar (placed here, not in handleKey, to avoid modal conflict)
	if key.Matches(msg, m.keys.TabFocus) {
		m.panesFocus = focusSidebar
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, m.keys.Down):
		for i := m.selected + 1; i < len(m.unifiedList); i++ {
			if m.unifiedList[i].kind != entryDivider {
				m.selected = i
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		for i := m.selected - 1; i >= 0; i-- {
			if m.unifiedList[i].kind != entryDivider {
				m.selected = i
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Top):
		m.selected = m.firstSelectableIndex()
		return m, nil

	case key.Matches(msg, m.keys.Bottom):
		for i := len(m.unifiedList) - 1; i >= 0; i-- {
			if m.unifiedList[i].kind != entryDivider {
				m.selected = i
				break
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, tea.Batch(fetchSessions(m.client), fetchWebStatus(m.client))

	case key.Matches(msg, m.keys.Attach):
		if len(m.unifiedList) == 0 {
			m.toast = "Session not available"
			m.toastKind = toastError
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		entry := m.unifiedList[m.selected]
		switch entry.kind {
		case entryLocal:
			s := entry.session
			switch s.Status {
			case "running", "idle", "waiting":
				// OK to attach
			default:
				m.toast = "Session not available"
				m.toastKind = toastError
				m.toastExp = time.Now().Add(2 * time.Second)
				return m, nil
			}
			cmd := &attachCmd{client: m.client, sessionID: s.ID}
			return m, tea.Exec(cmd, func(err error) tea.Msg {
				return attachDoneMsg{err: err}
			})
		case entryRemote:
			// Remote attach deferred: display only in Phase 78 scope
			m.toast = "Remote attach not yet supported"
			m.toastKind = toastInfo
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		default:
			return m, nil
		}

	case key.Matches(msg, m.keys.New):
		return m.openNewSessionModal()

	case key.Matches(msg, m.keys.Kill):
		if len(m.unifiedList) == 0 {
			return m, nil
		}
		entry := m.unifiedList[m.selected]
		if entry.kind == entryRemote {
			m.toast = "Cannot kill remote session"
			m.toastKind = toastInfo
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		if entry.kind != entryLocal || entry.session == nil {
			return m, nil
		}
		s := *entry.session
		m.modal = modalKillConfirm
		m.killTarget = &s
		m.killFocusYes = false
		return m, nil

	case key.Matches(msg, m.keys.Rename):
		if len(m.unifiedList) == 0 {
			return m, nil
		}
		entry := m.unifiedList[m.selected]
		if entry.kind == entryRemote {
			m.toast = "Cannot rename remote session"
			m.toastKind = toastInfo
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		if entry.kind != entryLocal || entry.session == nil {
			return m, nil
		}
		s := entry.session
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

	case key.Matches(msg, m.keys.QR):
		if len(m.unifiedList) == 0 {
			return m, nil
		}
		entry := m.unifiedList[m.selected]
		url := m.sessionURL(entry)
		if url == "" {
			m.toast = "Web serving not enabled for this session"
			m.toastKind = toastInfo
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		// Check terminal size (55x25 minimum for QR overlay per UI-SPEC)
		if m.width < 55 || m.height < 25 {
			m.toast = "Terminal too small to display QR code"
			m.toastKind = toastInfo
			m.toastExp = time.Now().Add(3 * time.Second)
			return m, nil
		}
		q, err := qrcode.New(url, qrcode.Medium)
		if err != nil {
			m.toast = fmt.Sprintf("QR code generation failed: %s", err)
			m.toastKind = toastError
			m.toastExp = time.Now().Add(3 * time.Second)
			return m, nil
		}
		var name string
		isRemote := false
		switch entry.kind {
		case entryLocal:
			name = entry.session.Name
		case entryRemote:
			name = entry.remote.Name
			isRemote = true
		}
		m.qrSession = &sessionRef{
			ID:       entryID(entry),
			Name:     name,
			IsRemote: isRemote,
			URL:      url,
		}
		m.qrContent = q.ToSmallString(false)
		m.qrURL = url
		return m, nil
	}
	return m, nil
}

// handleSidebarKey handles keys when the sidebar pane has focus.
func (m Model) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab toggles focus back to content
	if key.Matches(msg, m.keys.TabFocus) {
		m.panesFocus = focusContent
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.sidebarFocus > 0 {
			m.sidebarFocus--
		}
	case "down", "j":
		if m.sidebarFocus < len(sidebarTabs)-1 {
			m.sidebarFocus++
		}
	case "enter":
		m.openTab(sidebarTabs[m.sidebarFocus])
		m.panesFocus = focusContent
	case "Q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	}
	return m, nil
}

// rebuildUnifiedList constructs the unified list from local sessions and remote groups.
// Restores selection by entry identity to prevent cursor jump on refresh.
func (m *Model) rebuildUnifiedList() {
	// Remember current selection identity
	var selID string
	var selKind listEntryKind
	if m.selected >= 0 && m.selected < len(m.unifiedList) {
		cur := m.unifiedList[m.selected]
		selKind = cur.kind
		switch cur.kind {
		case entryLocal:
			if cur.session != nil {
				selID = cur.session.ID
			}
		case entryRemote:
			if cur.remote != nil {
				selID = cur.remote.ID + ":" + cur.remote.Hostname
			}
		}
	}

	var list []listEntry
	// Local sessions first
	for i := range m.sessions {
		list = append(list, listEntry{kind: entryLocal, session: &m.sessions[i]})
	}
	// Remote groups (already sorted alphabetically by hostname from fetchRemoteFn)
	for _, g := range m.remoteSessions {
		if len(g.Sessions) == 0 {
			continue
		}
		list = append(list, listEntry{kind: entryDivider, divider: &peerDivider{
			Hostname:     g.Hostname,
			SessionCount: len(g.Sessions),
		}})
		for i := range g.Sessions {
			list = append(list, listEntry{kind: entryRemote, remote: &g.Sessions[i]})
		}
	}
	m.unifiedList = list

	// Restore selection by identity
	restored := false
	if selID != "" {
		for i, e := range m.unifiedList {
			switch {
			case e.kind == selKind && e.kind == entryLocal && e.session != nil && e.session.ID == selID:
				m.selected = i
				restored = true
			case e.kind == selKind && e.kind == entryRemote && e.remote != nil && (e.remote.ID+":"+e.remote.Hostname) == selID:
				m.selected = i
				restored = true
			}
			if restored {
				break
			}
		}
	}
	if !restored {
		// Clamp to first selectable entry
		m.selected = m.firstSelectableIndex()
	}
}

// firstSelectableIndex returns the index of the first non-divider entry, or 0 if empty.
func (m Model) firstSelectableIndex() int {
	for i, e := range m.unifiedList {
		if e.kind != entryDivider {
			return i
		}
	}
	return 0
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
//
// Phase 101 SHELL-03: agent selection reads from the unified agentEntries
// slice (AI CLIs + shells). For shell entries (cli ∈ {shell, bash, zsh, pwsh,
// powershell}) the args field is intentionally dropped before dispatch —
// mirrors Phase 100 Anti-Pattern A6: shell sessions never accept caller-
// supplied argv.
func (m Model) submitNewSession() (tea.Model, tea.Cmd) {
	entries := m.agentEntries()
	if len(entries) == 0 {
		m.toast = "Agent is required"
		m.toastKind = toastError
		m.toastExp = time.Now().Add(2 * time.Second)
		return m, nil
	}
	idx := m.agentIdx
	if idx < 0 || idx >= len(entries) {
		idx = 0
	}

	// Validate directory
	workDir := strings.TrimSpace(m.dirInput.Value())
	if workDir == "" {
		m.toast = "Directory is required"
		m.toastKind = toastError
		m.toastExp = time.Now().Add(2 * time.Second)
		return m, nil
	}

	cli := entries[idx].cliKey
	name := filepath.Base(workDir)

	// Parse arguments (split on spaces, simple). For shell sessions args is
	// dropped per Phase 100 A6.
	var args []string
	if !isShellCLI(cli) {
		argsStr := strings.TrimSpace(m.argsInput.Value())
		if argsStr != "" {
			args = strings.Fields(argsStr)
		}
	}

	m.modal = modalNone
	m.toast = "Creating session..."
	m.toastKind = toastInfo
	m.toastExp = time.Now().Add(10 * time.Second)
	return m, createSession(m.client, cli, name, workDir, args)
}

// isShellCLI reports whether the given cli identifier represents a raw shell
// session. Mirrors the agentBadgeColor shell case and the cmdNewShell allowlist.
func isShellCLI(cli string) bool {
	switch cli {
	case "shell", "bash", "zsh", "pwsh", "powershell":
		return true
	}
	return false
}
