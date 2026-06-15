package tui

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	qrcode "github.com/skip2/go-qrcode"
)

// remoteBaseURLFromSessionURL strips the /sessions/{id} suffix off a
// RemoteSessionEntry.URL of shape "https://{fqdn}:{port}/sessions/{id}",
// returning the bare https://{fqdn}:{port} base. Uses net/url so query
// strings and ports are preserved correctly (Phase 122 RESEARCH
// §Don't Hand-Roll).
func remoteBaseURLFromSessionURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

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

	case filesListMsg:
		return m.applyFilesListMsg(msg)

	case filesHeadMsg:
		return m.applyFilesHeadMsg(msg)

	case filesReadMsg:
		return m.applyFilesReadMsg(msg)

	case filesEditReadyMsg:
		return m.applyFilesEditReadyMsg(msg)

	case editorExitMsg:
		return m.applyEditorExitMsg(msg)

	case filesOpMsg:
		return m.applyFilesOpMsg(msg)

	case joinCodeResultMsg:
		return m.applyJoinCodeResultMsg(msg)

	case cancelJoinCodeMsg:
		m.modal = modalNone
		return m, nil
	}

	return m, nil
}

// handleJoinCodePromptKey routes keypresses inside the join-code prompt
// modal to the prompt sub-model. The prompt's Update returns either a
// no-op, the exchange Cmd, or the cancel Cmd.
func (m Model) handleJoinCodePromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C is the universal "quit" contract for this TUI and must not be
	// swallowed by the textinput inside the prompt. Mirrors the equivalent
	// guard in handleFilesKey's filter-mode (files.go CR-01).
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.joinCodePrompt, cmd = m.joinCodePrompt.Update(msg)
	return m, cmd
}

// applyJoinCodeResultMsg consumes the outcome of an exchangeJoinCodeCmd. On
// success it caches the cap, closes the modal, opens tabFiles against a
// fresh RemoteFilesClient, and dispatches loadDirCmd. On failure it
// transitions the prompt to error state so the user can retry.
//
// Staleness guard: msg.sessionID is compared against the prompt's
// sessionID. If the user dismissed the modal and triggered a different
// remote session between Cmd dispatch and result delivery, we drop the
// stale result silently (the new flow has already overwritten
// m.joinCodePrompt).
func (m Model) applyJoinCodeResultMsg(msg joinCodeResultMsg) (Model, tea.Cmd) {
	if m.modal != modalJoinCodePrompt || msg.sessionID != m.joinCodePrompt.sessionID {
		return m, nil
	}
	if msg.err != nil {
		m.joinCodePrompt.state = joinCodePromptError
		m.joinCodePrompt.errMsg = friendlyJoinCodeError(msg.err)
		return m, nil
	}

	// Success: cache the cap, close the modal, build the remote files
	// client, open the Files tab.
	sid := m.joinCodePrompt.sessionID
	baseURL := m.joinCodePrompt.remoteBaseURL
	if m.remoteCapStore == nil {
		m.remoteCapStore = make(map[string]remoteCapEntry)
	}
	m.remoteCapStore[sid] = remoteCapEntry{baseURL: baseURL, capToken: msg.cap}
	m.modal = modalNone

	// Recompute pane dimensions in case the window resized while the modal
	// was open (parity with the cap-cached fast path).
	cw := m.contentWidth()
	listW := cw * 40 / 100
	previewW := cw - listW - 1
	if previewW < 1 {
		previewW = 1
	}
	paneH := m.height - 3
	if paneH < 1 {
		paneH = 1
	}

	client := NewRemoteFilesClient(baseURL, msg.cap)
	m.files = newFilesModelWithClient(sid, client, listW, paneH-2, previewW, paneH-2)
	m.openTab(tabFiles)
	return m, loadDirCmd(m.files.client, sid, ".", m.files.generation)
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
	// Priority 2.5: File delete confirmation dialog (Phase 126 / TUIW-05).
	// Sits immediately after kill-confirm so the "y" key consumed by the
	// confirm dialog cannot leak to handleFilesKey (Tab cycling, etc.).
	// Priority sub-test: TestFilesDelete_DispatchPriority.
	if m.modal == modalFileDeleteConfirm {
		return m.handleFileDeleteConfirmKey(msg)
	}
	// Priority 2.6: Files inline name-input (rename / mkdir) (Phase 126 / TUIW-05).
	// Sits above the Files-tab dispatcher (5.5) so Enter and Esc are consumed
	// by the input handler and never reach handleFilesKey (which would navigate).
	if m.files.nameInputActive {
		return m.handleFilesNameInputKey(msg)
	}
	// Priority 3: New session modal
	if m.modal == modalNewSession {
		return m.handleNewSessionKey(msg)
	}
	// Priority 3.5: Join-code prompt modal (Phase 122).
	// Sits between kill-confirm and QR overlay because the join-code prompt
	// is initiated from the Sessions view via `f` (same flow surface as
	// kill-confirm) and must capture keypresses before they reach the
	// Files-tab dispatcher below.
	if m.modal == modalJoinCodePrompt {
		return m.handleJoinCodePromptKey(msg)
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
	// Priority 5.5: Files tab active (Phase 121).
	// Sits BELOW the modal/help priorities (1-5) so kill-confirm, new-session,
	// QR, and help overlays still capture keys; ABOVE the tab-cycling check
	// so the Plan-02 navigation dispatch (h/j/k/l, /, Enter, Backspace) is
	// not intercepted by the main view's handlers.
	if m.activeTabID() == tabFiles {
		return m.handleFilesKey(msg)
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

	case key.Matches(msg, m.keys.FilesOpen):
		// Phase 121: open Files tab from a selected Sessions-list entry.
		// Phase 122 (REMOTE-03): branch on entryLocal vs entryRemote;
		// remote-with-cached-cap takes the fast path; remote-without-cap
		// opens the join-code prompt modal.
		if len(m.unifiedList) == 0 {
			return m, nil
		}
		entry := m.unifiedList[m.selected]

		cw := m.contentWidth()
		listW := cw * 40 / 100
		previewW := cw - listW - 1
		if previewW < 1 {
			previewW = 1
		}
		paneH := m.height - 3
		if paneH < 1 {
			paneH = 1
		}

		switch entry.kind {
		case entryLocal:
			if entry.session == nil {
				return m, nil
			}
			sid := entry.session.ID
			// RESEARCH.md Pitfall TUI-PITFALL-6: always reset on `f`.
			m.files = newFilesModelWithClient(sid, m.client, listW, paneH-2, previewW, paneH-2)
			m.openTab(tabFiles)
			return m, loadDirCmd(m.files.client, sid, ".", m.files.generation)

		case entryRemote:
			if entry.remote == nil {
				return m, nil
			}
			sid := entry.remote.ID

			// Cap-cached fast path (REMOTE-03 + D-03).
			if cached, ok := m.remoteCapStore[sid]; ok {
				client := NewRemoteFilesClient(cached.baseURL, cached.capToken)
				m.files = newFilesModelWithClient(sid, client, listW, paneH-2, previewW, paneH-2)
				m.openTab(tabFiles)
				return m, loadDirCmd(m.files.client, sid, ".", m.files.generation)
			}

			// Cap not cached — open join-code prompt modal (D-01 parity).
			baseURL := remoteBaseURLFromSessionURL(entry.remote.URL)
			if baseURL == "" {
				m.toast = "Cannot parse remote session URL"
				m.toastKind = toastError
				m.toastExp = time.Now().Add(3 * time.Second)
				return m, nil
			}
			m.joinCodePrompt = newJoinCodePromptModel(sid, entry.remote.Name, entry.remote.Hostname, baseURL)
			m.modal = modalJoinCodePrompt
			return m, nil
		}
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

// handleFileDeleteConfirmKey handles keys when the file-delete confirmation
// dialog is open. Clones handleKillConfirmKey with delete semantics.
//
// y / Enter-on-Yes → dispatch deleteCmd + close modal
// n / esc           → close modal without dispatch (T-126-09)
// Enter-on-No       → close modal without dispatch
// left/right/h/l/tab → toggle focus between No (default) and Yes (Delete)
func (m Model) handleFileDeleteConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "y":
		return m.executeFileDelete()
	case s == "n", s == "esc":
		m.modal = modalNone
		m.fileDeleteTarget = nil
		return m, nil
	case s == "enter":
		if m.fileDeleteFocusYes {
			return m.executeFileDelete()
		}
		m.modal = modalNone
		m.fileDeleteTarget = nil
		return m, nil
	case s == "left", s == "right", s == "h", s == "l", s == "tab":
		m.fileDeleteFocusYes = !m.fileDeleteFocusYes
		return m, nil
	}
	return m, nil
}

// executeFileDelete sends the deleteCmd and cleans up modal state.
//
// WR-03 (capture-at-keypress contract): The delete target (relPath, isDir,
// name) is captured when the user presses 'd' and stored in m.fileDeleteTarget.
// Between capture and confirm, an in-flight loadDirCmd may land and refresh
// m.files.entries, moving the list cursor. The modal text shows the CAPTURED
// name (not the current cursor entry), so the user always confirms the correct
// target regardless of cursor drift.
//
// IMPORTANT for future maintainers: do NOT change this to re-read the current
// selection at confirm time — that would reintroduce a TOCTOU race where a
// background refresh moves the cursor between 'd' and 'y', causing the user to
// delete a different file than they intended.
//
// The deleteCmd is dispatched with target.relPath — the stable captured path —
// not any derived value from the refreshed entries list.
func (m Model) executeFileDelete() (tea.Model, tea.Cmd) {
	if m.fileDeleteTarget == nil {
		m.modal = modalNone
		return m, nil
	}
	// Use the captured target explicitly — never re-read from m.files.entries.
	target := m.fileDeleteTarget
	m.modal = modalNone
	m.fileDeleteTarget = nil
	m.toast = "Deleting..."
	m.toastKind = toastInfo
	m.toastExp = time.Now().Add(10 * time.Second)
	m.files.generation++ // supersede any in-flight request
	return m, deleteCmd(m.files.client, m.files.sessionID, target.relPath, m.files.generation)
}

// handleFilesNameInputKey handles keys while the inline name-input (rename or
// mkdir) is active. Clones handleRenameKey with files semantics.
//
// enter → trim value; dispatch renameCmd or mkdirCmd; clear input state
// esc   → cancel; clear input state
// default → forward to nameInput.Update (textinput handles typing + backspace)
func (m Model) handleFilesNameInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "enter":
		name := strings.TrimSpace(m.files.nameInput.Value())
		if name == "" {
			m.toast = "Name cannot be empty"
			m.toastKind = toastError
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		// WR-02: reject names containing path separators or the special
		// relative-navigation tokens "." and "..". The server is the real
		// security boundary, but accepting these at the input layer would
		// silently target a different directory than displayed (path.Join
		// collapses ".." segments) — a correctness/UX defect, not an escape.
		// Only plain filenames within the current directory are accepted.
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			m.toast = "Name cannot contain path separators"
			m.toastKind = toastError
			m.toastExp = time.Now().Add(2 * time.Second)
			return m, nil
		}
		m.files.nameInputActive = false
		m.files.nameInput.Blur()

		switch m.files.nameInputMode {
		case "rename":
			if name == m.files.nameInputOriginal {
				// No-op guard: user pressed Enter without changing the name.
				return m, nil
			}
			oldRel := joinDir(m.files.cwd, m.files.nameInputOriginal)
			newRel := joinDir(m.files.cwd, name)
			m.files.generation++ // WR-03
			return m, renameCmd(m.files.client, m.files.sessionID, oldRel, newRel, m.files.generation)
		case "mkdir":
			newRel := joinDir(m.files.cwd, name)
			m.files.generation++ // WR-03
			return m, mkdirCmd(m.files.client, m.files.sessionID, newRel, m.files.generation)
		}
		return m, nil
	case s == "esc":
		m.files.nameInputActive = false
		m.files.nameInput.Blur()
		return m, nil
	default:
		var cmd tea.Cmd
		m.files.nameInput, cmd = m.files.nameInput.Update(msg)
		return m, cmd
	}
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
// Phase 108 PARITY-TUI-01/02: agent selection reads from the unified
// agentEntries slice, which post-collapse emits AI CLI keys plus a single
// static "shell" entry (no per-shell variants). For shell entries the args
// field is intentionally dropped before dispatch — mirrors Phase 100
// Anti-Pattern A6: shell sessions never accept caller-supplied argv.
//
// isShellCLI still accepts the legacy keys {bash, zsh, pwsh, powershell}
// as defense-in-depth for any non-picker caller (e.g. restored sessions);
// the picker itself can only produce "shell" post-Phase-108.
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
// session. Post-Phase-108 the TUI picker emits only "shell"; the legacy
// per-shell keys {bash, zsh, pwsh, powershell} are kept here as defense-
// in-depth for non-picker callers (e.g. session restoration with a stored
// pre-collapse cliKey) and mirror the agentBadgeColor shell case.
func isShellCLI(cli string) bool {
	switch cli {
	case "shell", "bash", "zsh", "pwsh", "powershell":
		return true
	}
	return false
}

// applyFilesEditReadyMsg handles the result of editFetchCmd: the file bytes
// have been written to a host-local temp file. If fetching failed, set an
// error toast and refresh the listing. On success, suspend the TUI and hand
// the terminal to the editor process. The editor path + temp file path arrive
// in msg — no need to call resolveEditor() again (T-126-04: argv, not shell).
func (m Model) applyFilesEditReadyMsg(msg filesEditReadyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.toast = fmt.Sprintf("Edit: %s", msg.err)
		m.toastKind = toastError
		m.toastExp = time.Now().Add(3 * time.Second)
		m.files.generation++
		return m, loadDirCmd(m.files.client, msg.sessionID, m.files.cwd, m.files.generation)
	}
	// T-126-04: pass tmpPath as a separate argv element, never via shell string.
	cmd := exec.Command(msg.editor, msg.tmpPath) //nolint:gosec // editor resolved via LookPath; tmpPath is a controlled os.CreateTemp path
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorExitMsg{
			sessionID:  msg.sessionID,
			generation: msg.generation,
			tmpPath:    msg.tmpPath,
			relPath:    msg.relPath,
			exitErr:    err,
		}
	})
}

// applyEditorExitMsg handles editorExitMsg: the editor process has exited.
// Per TUIW-04, the write-back is UNCONDITIONAL — it happens regardless of the
// editor's exit code. A toast is shown on non-zero exit, but the write-back
// still runs.
//
// CR-01 FIX: Do NOT bump generation here. The generation is left at its
// current value so that the editWriteBackCmd result (stamped with msg.generation)
// is NOT discarded as stale by applyFilesOpMsg's staleness guard. Only after
// applyFilesOpMsg processes the write-back result (success or error) does it
// bump generation and dispatch the listing refresh. This ensures a failed
// write-back error is always surfaced to the user.
//
// Batch order: tea.ClearScreen, editWriteBackCmd (TUIW-04).
func (m Model) applyEditorExitMsg(msg editorExitMsg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{tea.ClearScreen}
	if msg.exitErr != nil {
		m.toast = fmt.Sprintf("Editor exited with error: %s", msg.exitErr)
		m.toastKind = toastError
		m.toastExp = time.Now().Add(3 * time.Second)
	}
	// UNCONDITIONAL write-back — NOT gated on exitErr==nil (TUIW-04).
	// Generation is NOT bumped here — see CR-01 fix comment above.
	cmds = append(cmds, editWriteBackCmd(m.files.client, msg.sessionID, msg.relPath, msg.tmpPath, msg.generation))
	return m, tea.Batch(cmds...)
}

// applyFilesOpMsg handles the result of a write operation (edit write-back,
// delete, rename, mkdir). Stale messages (generation behind current) are
// silently discarded. On error, a toast is shown and the listing is refreshed
// to show current state. On success, all ops (including edit write-back)
// refresh the listing unconditionally (TUIW-05).
//
// CR-01 FIX: The edit write-back (op=="edit") is no longer excluded from the
// success-refresh path. applyEditorExitMsg no longer pre-bumps the generation
// or pre-dispatches loadDirCmd, so this function is now solely responsible for
// the post-write-back listing refresh. This also means the error path is never
// skipped by the staleness guard for edit write-backs.
func (m Model) applyFilesOpMsg(msg filesOpMsg) (tea.Model, tea.Cmd) {
	if msg.generation < m.files.generation {
		return m, nil // stale — discard
	}
	if msg.err != nil {
		// RMW-04: v3.4 peer has no write routes — surface the verbatim "older version"
		// message instead of the generic "<op> failed: ..." copy.
		if errors.Is(msg.err, ErrRemotePeerNoWriteSupport) {
			m.toast = remotePeerOutdatedMessage
		} else {
			m.toast = fmt.Sprintf("%s failed: %s", msg.op, msg.err)
		}
		m.toastKind = toastError
		m.toastExp = time.Now().Add(3 * time.Second)
		// Re-run a fresh listing so the UI shows the current state.
		m.files.generation++
		return m, loadDirCmd(m.files.client, msg.sessionID, m.files.cwd, m.files.generation)
	}
	// Success: refresh listing for all ops including edit write-back (TUIW-05).
	m.files.generation++
	return m, loadDirCmd(m.files.client, msg.sessionID, m.files.cwd, m.files.generation)
}
