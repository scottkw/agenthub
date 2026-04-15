# Phase 78: TUI Remote & QR - Pattern Map

**Mapped:** 2026-04-15
**Files analyzed:** 12 (modified) + 1 (potentially created)
**Analogs found:** 13 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/tui/model.go` | model | state-container | `internal/tui/model.go` (self -- extend) | exact |
| `internal/tui/cmds.go` | service | request-response | `internal/tui/cmds.go` (self -- extend) | exact |
| `internal/tui/update.go` | controller | event-driven | `internal/tui/update.go` (self -- extend) | exact |
| `internal/tui/view.go` | component | transform | `internal/tui/view.go` (self -- extend) | exact |
| `internal/tui/keys.go` | config | static | `internal/tui/keys.go` (self -- modify) | exact |
| `internal/tui/help.go` | component | transform | `internal/tui/help.go` (self -- modify) | exact |
| `internal/tui/modal.go` | component | transform | `internal/tui/modal.go` (self -- extend) | exact |
| `internal/tui/tui.go` | provider | request-response | `internal/tui/tui.go` (self -- extend) | exact |
| `cmd_tui.go` | controller | request-response | `cmd_tui.go` (self -- extend) | exact |
| `internal/tui/update_test.go` | test | unit | `internal/tui/update_test.go` (self -- extend) | exact |
| `internal/tui/view_test.go` | test | unit | `internal/tui/view_test.go` (self -- extend) | exact |
| `internal/tui/help_test.go` | test | unit | `internal/tui/help_test.go` (self -- modify) | exact |
| `internal/tui/qr.go` (new) | component | transform | `internal/tui/modal.go` `renderKillConfirmModal` | exact |

## Pattern Assignments

### `internal/tui/model.go` (model, state-container)

**Analog:** `internal/tui/model.go` (self -- extend existing types)

**Existing type pattern** (lines 12-18 -- iota-based enums):
```go
type modalState int

const (
	modalNone modalState = iota
	modalNewSession
	modalKillConfirm
)
```
New types must follow the same iota-enum pattern for `listEntryKind`.

**Existing struct pattern** (lines 30-66 -- Model with grouped fields and phase comments):
```go
type Model struct {
	client    *daemon.DaemonClient
	sessions  []daemon.SessionInfo
	webStatus daemon.WebServerStatusResponse
	selected  int
	// ...existing fields...

	// Modal state (Phase 77)
	modal        modalState
	// ...

	// Kill confirmation state (Phase 77)
	killTarget   *daemon.SessionInfo
	killFocusYes bool
}
```
New fields (`remoteSessions`, `unifiedList`, `qrSession`, `qrContent`, `qrURL`, `fetchRemoteFn`) must be grouped under a `// Phase 78` comment block, same as Phase 77 fields.

**Existing message type pattern** (lines 70-91):
```go
type sessionsMsg struct {
	sessions []daemon.SessionInfo
	err      error
}
```
New `remoteSessionsMsg` must follow the same struct pattern with typed payload and `err error` field.

**New types to add** (from UI-SPEC Data Model section):
```go
type listEntryKind int

const (
	entryLocal listEntryKind = iota
	entryRemote
	entryDivider
)

type listEntry struct {
	kind    listEntryKind
	session *daemon.SessionInfo
	remote  *remoteSessionEntry
	divider *peerDivider
}

type remoteSessionEntry struct {
	ID       string
	Name     string
	CLIType  string
	Status   string
	Hostname string
	FQDN     string
	URL      string
}

type peerDivider struct {
	Hostname     string
	SessionCount int
}

type sessionRef struct {
	ID       string
	Name     string
	IsRemote bool
	URL      string
}
```

**Callback injection field** (RESEARCH.md Option 3):
```go
// Remote session fetching (Phase 78)
fetchRemoteFn func(ctx context.Context) []listRemoteGroup
```
Where `listRemoteGroup` mirrors the type from `cmd_cli.go` line 80:
```go
type listRemoteGroup struct {
	Hostname string
	Sessions []remoteSessionEntry
}
```

---

### `internal/tui/cmds.go` (service, request-response)

**Analog:** `internal/tui/cmds.go` (self -- extend with one new tea.Cmd)

**Existing cmd pattern** (lines 12-17 -- function returning tea.Cmd):
```go
func fetchSessions(client *daemon.DaemonClient) tea.Cmd {
	return func() tea.Msg {
		sessions, err := client.ListSessions()
		return sessionsMsg{sessions: sessions, err: err}
	}
}
```
New `fetchRemoteSessions` must follow the identical closure pattern. The key difference: it calls a callback function rather than a client method directly (to avoid the package-main import cycle).

**New cmd to add:**
```go
func fetchRemoteSessions(fn func(ctx context.Context) []listRemoteGroup) tea.Cmd {
	return func() tea.Msg {
		if fn == nil {
			return remoteSessionsMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		groups := fn(ctx)
		return remoteSessionsMsg{groups: groups}
	}
}
```

---

### `internal/tui/update.go` (controller, event-driven)

**Analog:** `internal/tui/update.go` (self -- extend Update, handleKey, handleMainKey)

**Message handler pattern** (lines 32-45 -- sessionsMsg as template for remoteSessionsMsg):
```go
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
```
The `remoteSessionsMsg` handler must follow the same pattern but: (1) store remote groups, (2) call `rebuildUnifiedList()`, (3) restore selection by entry identity (not just clamp).

**Key dispatch priority pattern** (lines 106-129):
```go
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
		// ...
	}
	// Priority 5: Main view
	return m.handleMainKey(msg)
}
```
QR overlay check (`m.qrSession != nil`) inserts at **Priority 4** (between new-session modal and help), bumping help to Priority 5 and main view to Priority 6. Must follow the exact same `if` + `return m.handleQRKey(msg)` pattern.

**Kill confirm key handler pattern** (lines 254-275 -- template for handleQRKey):
```go
func (m Model) handleKillConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "y":
		return m.executeKill()
	case s == "n", s == "esc":
		m.modal = modalNone
		m.killTarget = nil
		return m, nil
	// ...
	}
	return m, nil
}
```
`handleQRKey` follows the same shape: match specific keys (`esc`, `q` to close; `Q`, `ctrl+c` to quit), swallow everything else.

**Toast pattern** (used throughout -- e.g., lines 168-170):
```go
m.toast = "Session not available"
m.toastKind = toastError
m.toastExp = time.Now().Add(2 * time.Second)
return m, nil
```
Remote session blocked actions (kill/rename) and QR error states use this exact 4-line toast pattern.

**Navigation pattern -- current** (lines 141-148):
```go
case key.Matches(msg, m.keys.Down):
	if m.selected < len(m.sessions)-1 {
		m.selected++
	}
	return m, nil
```
This must be replaced with divider-skipping logic that operates on `m.unifiedList` instead of `m.sessions`. The RESEARCH.md Pattern 2 provides the algorithm:
```go
func (m Model) moveDown() Model {
	for i := m.selected + 1; i < len(m.unifiedList); i++ {
		if m.unifiedList[i].kind != entryDivider {
			m.selected = i
			return m
		}
	}
	return m
}
```

**Tick batch pattern** (lines 52-57):
```go
case tickMsg:
	return m, tea.Batch(
		fetchSessions(m.client),
		fetchWebStatus(m.client),
		nextTick(),
	)
```
Extend this batch to include `fetchRemoteSessions(m.fetchRemoteFn)`.

**Kill/Rename on selected session pattern** (lines 194-201):
```go
case key.Matches(msg, m.keys.Kill):
	if len(m.sessions) == 0 {
		return m, nil
	}
	s := m.sessions[m.selected]
	m.modal = modalKillConfirm
	m.killTarget = &s
	m.killFocusYes = false
	return m, nil
```
All operations that access `m.sessions[m.selected]` must be converted to access `m.unifiedList[m.selected]` and check `entry.kind` before acting. For remote sessions, show a blocking toast instead.

---

### `internal/tui/view.go` (component, transform)

**Analog:** `internal/tui/view.go` (self -- extend renderSessionList, renderHeader, renderFooter)

**Session row rendering pattern** (lines 182-227 -- renderSessionRow):
```go
func (m Model) renderSessionRow(s daemon.SessionInfo, idx int) string {
	isSelected := idx == m.selected

	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	glyph, glyphColor := statusGlyph(s.Status, m.styles)
	styledGlyph := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph)

	nameWidth := m.nameColWidth()
	name := truncate(s.Name, nameWidth)
	agent := truncate(s.CLI, 12)
	host := truncate(s.Hostname, 20)
	viewers := ""
	if s.ViewerCount > 0 {
		viewers = fmt.Sprintf("%d", s.ViewerCount)
	}

	row := fmt.Sprintf("%s%s %-*s  %-12s  %-20s  %7s",
		cursor, styledGlyph, nameWidth, name, agent, host, viewers)

	if isSelected {
		return lipgloss.NewStyle().
			Background(m.styles.BgSelected).
			Foreground(m.styles.FgSelected).
			Width(m.width).
			Render(row)
	}
	return lipgloss.NewStyle().
		Foreground(m.styles.FgNormal).
		Width(m.width).
		Render(row)
}
```
Remote session rows use this **exact same pattern**, substituting `remoteSessionEntry` fields for `SessionInfo` fields. The only difference: `viewers` is always blank for remote sessions.

**Session list iteration pattern** (lines 110-159 -- renderSessionList):
```go
func (m Model) renderSessionList() string {
	listHeight := m.height - 5
	// ...empty/error/loading states...
	start, end := m.visibleRange(listHeight)
	var rows []string
	for i := start; i < end; i++ {
		rows = append(rows, m.renderSessionRow(m.sessions[i], i))
	}
	// pad remaining lines...
}
```
Must be refactored to iterate `m.unifiedList` instead of `m.sessions`, dispatching per `entry.kind`:
- `entryLocal` -> `renderSessionRow` (existing)
- `entryRemote` -> `renderRemoteSessionRow` (new, mirrors renderSessionRow)
- `entryDivider` -> `renderDividerRow` (new)

**Divider row rendering** -- new, but follows the row width pattern from renderColumnHeaders (lines 97-107):
```go
func (m Model) renderColumnHeaders() string {
	style := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgMuted)
	row := fmt.Sprintf("    %-*s  %-12s  %-20s  %7s",
		nameWidth, "NAME", "AGENT", "HOST", "VIEWERS")
	return style.Render(row)
}
```
Divider row uses full terminal width with `fg.accent` for hostname and `fg.muted` for fill chars.

**Header pattern** (lines 74-94):
```go
func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().Bold(true).
		Foreground(m.styles.FgNormal).
		Render("AgentHub")
	count := fmt.Sprintf("%d sessions", len(m.sessions))
	if len(m.sessions) == 1 {
		count = "1 session"
	}
	// ...right-align count...
}
```
Must be updated: when remote sessions exist, show `"{N} local, {M} remote"` per UI-SPEC.

**Hint bar pattern** (lines 276-280):
```go
func (m Model) renderHintBar() string {
	hint := "j/k Up/Down  Enter Attach  n New  d Kill  r Rename  ? Help  q Quit"
	return lipgloss.NewStyle().Foreground(m.styles.FgMuted).
		Width(m.width).Render(hint)
}
```
Update to: `"j/k Up/Down  Enter Attach  q QR  n New  d Kill  r Rename  ? Help  Q Quit"`.

**Web status footer -- quit hint** (lines 248-249):
```go
quitHint := lipgloss.NewStyle().Foreground(m.styles.FgAccent).Render("q Quit")
```
Update to `"Q Quit"`.

**visibleRange pattern** (lines 162-179):
```go
func (m Model) visibleRange(listHeight int) (int, int) {
	total := len(m.sessions)
	// ...scrolling logic...
}
```
Must be updated to use `len(m.unifiedList)` instead of `len(m.sessions)`.

**renderFull modal overlay dispatch** (lines 62-70):
```go
if m.modal == modalNewSession {
	return m.renderNewSessionModal()
}
if m.modal == modalKillConfirm {
	return m.renderKillConfirmModal()
}
```
Add QR overlay check: `if m.qrSession != nil { return m.renderQROverlay() }` -- placed after modal checks but before returning `content`.

---

### `internal/tui/qr.go` (new file) or `internal/tui/modal.go` (extend)

**Analog:** `internal/tui/modal.go` `renderKillConfirmModal` (lines 98-163)

**Modal overlay construction pattern** (the exact template to copy):
```go
func (m Model) renderKillConfirmModal() string {
	overlayWidth := max(40, min(55, m.width-20))
	innerWidth := overlayWidth - 6 // subtract border (2) + padding (4)

	// ... compose content string ...

	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderNormal).
		Width(overlayWidth-2).
		Padding(1, 2).
		Background(m.styles.BgModal).
		Render(content)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.FgDanger).  // QR uses BorderAccent instead of FgDanger
		Render(" Kill Session ")

	bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}
```
QR overlay follows this pattern exactly, with these substitutions:
- `overlayWidth` calculation uses `max(qr_cols + 6, url_len + 6, 50)` clamped to `min(terminal_cols - 4, 80)` per UI-SPEC
- Title uses `m.styles.BorderAccent` (not `FgDanger`)
- Title text: `" QR: {session-name} "`
- Content includes: QR code block (unstyled), URL line (`fg.accent`), hint line (`fg.muted`)
- **Critical:** QR code string (`m.qrContent`) must NOT be wrapped in any lipgloss color style -- use `lipgloss.Place()` for centering only

**QR generation pattern** from `cmd_cli.go` (lines 317-334):
```go
func cmdQR(client *daemon.DaemonClient, args []string, out io.Writer) error {
	resp, err := client.GetWebServerStatus()
	if err != nil || !resp.Running {
		return fmt.Errorf("web server not running")
	}
	url := fmt.Sprintf("%s/sessions/%s", resp.URL, args[0])
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("agenthub qr: %w", err)
	}
	fmt.Fprint(out, q.ToSmallString(false))
	fmt.Fprintln(out, url)
	return nil
}
```
TUI reuses the same `qrcode.New(url, qrcode.Medium)` + `q.ToSmallString(false)` call, but stores the result in `m.qrContent` instead of writing to stdout.

---

### `internal/tui/keys.go` (config, static)

**Analog:** `internal/tui/keys.go` (self -- modify)

**Key binding pattern** (lines 6-18, 20-67):
```go
type KeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	// ...
	Kill    key.Binding // Phase 77
	Rename  key.Binding // Phase 77
}

func defaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		// ...
	}
}
```
Changes:
1. Add `QR key.Binding` field to `KeyMap` struct (with `// Phase 78` comment)
2. Modify `Quit` binding: `key.WithKeys("Q", "ctrl+c")`, `key.WithHelp("Q", "quit")`
3. Add `QR` binding: `key.WithKeys("q")`, `key.WithHelp("q", "QR code / URL")`

---

### `internal/tui/help.go` (component, transform)

**Analog:** `internal/tui/help.go` (self -- modify)

**Help content binding pattern** (lines 59-67 -- Sessions group):
```go
sections = append(sections, groupStyle.Render("Sessions"))
sections = append(sections,
	formatBinding("Enter", "Attach to session"),
	formatBinding("n", "New session"),
	formatBinding("d", "Kill session"),
	formatBinding("r", "Rename session"),
)
```
Add `formatBinding("q", "QR code / URL")` after `"Enter"` line.

**Help content quit pattern** (lines 71-75 -- General group):
```go
sections = append(sections, groupStyle.Render("General"))
sections = append(sections,
	formatBinding("?", "Toggle help"),
	formatBinding("q, Ctrl+C", "Quit"),
)
```
Change to `formatBinding("Q, Ctrl+C", "Quit")`.

---

### `internal/tui/tui.go` (provider, request-response)

**Analog:** `internal/tui/tui.go` (self -- extend)

**Run function signature pattern** (lines 11-15):
```go
func Run(client *daemon.DaemonClient) error {
	p := tea.NewProgram(newModel(client))
	_, err := p.Run()
	return err
}
```
Must accept `fetchRemoteFn` parameter and pass it to `newModel`:
```go
func Run(client *daemon.DaemonClient, fetchRemoteFn func(ctx context.Context) []listRemoteGroup) error {
	p := tea.NewProgram(newModel(client, fetchRemoteFn))
	_, err := p.Run()
	return err
}
```

**newModel constructor pattern** (lines 19-27):
```go
func newModel(client *daemon.DaemonClient) Model {
	return Model{
		client:       client,
		loading:      true,
		keys:         defaultKeyMap(),
		styles:       newStyles(true),
		detectedCLIs: pty.DetectCLIs(),
	}
}
```
Add `fetchRemoteFn` parameter and store it in the returned Model.

**Init batch pattern** (lines 34-41):
```go
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		fetchSessions(m.client),
		fetchWebStatus(m.client),
		nextTick(),
	)
}
```
Add `fetchRemoteSessions(m.fetchRemoteFn)` to the batch.

---

### `cmd_tui.go` (controller, request-response)

**Analog:** `cmd_tui.go` (self -- extend)

**Current call pattern** (lines 13-25):
```go
func cmdTUI(client *daemon.DaemonClient) error {
	// ...TTY check...
	// ...health check...
	return tui.Run(client)
}
```
Must construct the `fetchRemoteFn` callback wrapping the package-main functions:
```go
func cmdTUI(client *daemon.DaemonClient) error {
	// ...existing checks...

	fetchRemoteFn := func(ctx context.Context) []tui.ListRemoteGroup {
		peers, _ := client.ListTailnetPeers()
		if len(peers) == 0 {
			return nil
		}
		// Same grouping pattern as cmdList (cmd_cli.go lines 103-128)
		groupMap := make(map[string][]tui.RemoteSessionEntry)
		for _, p := range peers {
			fqdn := strings.TrimSuffix(p.DNSName, ".")
			peerSessions, _ := fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)
			for _, s := range peerSessions {
				groupMap[p.Hostname] = append(groupMap[p.Hostname], tui.RemoteSessionEntry{
					ID: s.ID, Name: s.Name, CLIType: s.CLIType, Status: s.Status,
					Hostname: p.Hostname, FQDN: fqdn,
					URL: fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, tailnet.DefaultProbePort, s.ID),
				})
			}
		}
		var groups []tui.ListRemoteGroup
		for hostname, sess := range groupMap {
			groups = append(groups, tui.ListRemoteGroup{Hostname: hostname, Sessions: sess})
		}
		sort.Slice(groups, func(i, j int) bool { return groups[i].Hostname < groups[j].Hostname })
		return groups
	}
	return tui.Run(client, fetchRemoteFn)
}
```
Note: types used in the callback (`ListRemoteGroup`, `RemoteSessionEntry`) must be exported from `internal/tui` for the callback signature. Alternatively, use unexported types with a `func(ctx) []listRemoteGroup` closure that cmd_tui.go provides -- see RESEARCH.md Option 3 details.

**Exact remote fetch pattern from cmd_cli.go** (lines 103-128 -- the template for the callback body):
```go
peers, _ := client.ListTailnetPeers()
if len(peers) > 0 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	groupMap := make(map[string][]CLIRemoteSession)
	for _, p := range peers {
		fqdn := strings.TrimSuffix(p.DNSName, ".")
		peerSessions, _ := fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)
		for i := range peerSessions {
			peerSessions[i].Hostname = p.Hostname
			peerSessions[i].FQDN = fqdn
		}
		if len(peerSessions) > 0 {
			groupMap[p.Hostname] = append(groupMap[p.Hostname], peerSessions...)
		}
	}
}
```

---

### `internal/tui/update_test.go` (test, unit)

**Analog:** `internal/tui/update_test.go` (self -- extend)

**Test helper pattern** (lines 13-21):
```go
func testModel() Model {
	m := newModel(nil) // nil client -- tests don't make HTTP calls
	m.width = 120
	m.height = 24
	m.hasDark = true
	m.styles = newStyles(true)
	m.loading = false
	return m
}
```
Must be updated to accept/set `fetchRemoteFn` (nil for most tests; mock function for remote tests).

**Message test pattern** (lines 23-42 -- TestUpdate_SessionsMsg):
```go
func TestUpdate_SessionsMsg(t *testing.T) {
	m := testModel()
	m.loading = true

	sessions := []daemon.SessionInfo{
		{ID: "1", Name: "test-session", CLI: "claude", Hostname: "macbook", Status: "running"},
	}
	updated, _ := m.Update(sessionsMsg{sessions: sessions})
	result := updated.(Model)

	if result.loading {
		t.Error("expected loading=false after sessionsMsg")
	}
	if len(result.sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(result.sessions))
	}
}
```
New `TestUpdate_RemoteSessionsMsg` follows this exact shape: construct message, call `m.Update()`, assert on result fields.

**Key press test pattern** (lines 88-100 -- TestUpdate_KeyQuit):
```go
func TestUpdate_KeyQuit(t *testing.T) {
	m := testModel()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}
```
This test must be updated: `'q'` no longer quits. New test `TestUpdate_QuitKeyReassignment` verifies `'Q'` quits and `'q'` does NOT quit.

**Kill confirm test pattern** (lines 230-251 -- template for QR overlay tests):
```go
func TestUpdate_KillConfirmOpen(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "test", CLI: "claude", Status: "running"},
	}
	m.selected = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	result := updated.(Model)
	if result.modal != modalKillConfirm {
		t.Errorf("expected modal=modalKillConfirm, got %d", result.modal)
	}
}
```
QR overlay tests follow this same pattern: set up model with sessions + webStatus, press `'q'`, assert `result.qrSession != nil`.

**Toast assertion pattern** (lines 181-189):
```go
func TestUpdate_ReservedKeysShowToast(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := updated.(Model)
	if result.toast != "Session not available" {
		t.Errorf("expected 'Session not available' toast, got %q", result.toast)
	}
}
```
Use for: QR-no-URL toast, remote-kill-blocked toast, remote-rename-blocked toast tests.

---

### `internal/tui/view_test.go` (test, unit)

**Analog:** `internal/tui/view_test.go` (self -- extend)

**View content assertion pattern** (lines 14-48 -- TestView_SessionList):
```go
func TestView_SessionList(t *testing.T) {
	m := testModel()
	m.sessions = []daemon.SessionInfo{
		{ID: "1", Name: "my-session", CLI: "claude", Hostname: "macbook-pro", Status: "running", ViewerCount: 2},
	}

	v := m.View()
	content := v.Content

	checks := []string{"my-session", "claude", "macbook-pro"}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("view output missing %q", want)
		}
	}
}
```
New tests for divider rows, remote session rows, and QR overlay content follow this exact pattern: set up model state, call `m.View()`, assert `strings.Contains(content, expected)`.

**Hint bar test pattern** (lines 172-181):
```go
func TestView_HintBar(t *testing.T) {
	m := testModel()
	hint := m.renderHintBar()
	required := []string{"Enter Attach", "n New", "d Kill", "r Rename", "? Help", "q Quit"}
	for _, want := range required {
		if !strings.Contains(hint, want) {
			t.Errorf("hint bar missing %q", want)
		}
	}
}
```
Must be updated: `"q Quit"` -> `"Q Quit"`, add `"q QR"`.

---

### `internal/tui/help_test.go` (test, unit)

**Analog:** `internal/tui/help_test.go` (self -- modify)

**Help content test pattern** (lines 23-45):
```go
func TestHelpOverlay_ContainsBindings(t *testing.T) {
	m := testModel()
	content := m.buildHelpContent()

	bindings := []string{
		"Move up/down",
		"Attach to session",
		"New session",
		"Kill session",
		"Rename session",
		"Toggle help",
		"Quit",
	}
	for _, binding := range bindings {
		if !strings.Contains(content, binding) {
			t.Errorf("help content missing binding description %q", binding)
		}
	}
}
```
Add `"QR code / URL"` to the bindings list. Existing `"Quit"` check still passes since `"Q, Ctrl+C  Quit"` contains `"Quit"`.

---

## Shared Patterns

### Modal Overlay Construction
**Source:** `internal/tui/modal.go` lines 98-163 (`renderKillConfirmModal`)
**Apply to:** QR overlay (`renderQROverlay`)
```go
// Standard modal construction (all 3 overlays use this):
overlayWidth := max(W_MIN, min(W_MAX, m.width-MARGIN))

bordered := lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(m.styles.BorderNormal).
	Width(overlayWidth-2).
	Padding(1, 2).
	Background(m.styles.BgModal).
	Render(content)

title := lipgloss.NewStyle().
	Bold(true).
	Foreground(TITLE_COLOR).
	Render(" TITLE_TEXT ")

bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)

return lipgloss.Place(m.width, m.height,
	lipgloss.Center, lipgloss.Center, bordered)
```

### Toast Feedback
**Source:** `internal/tui/update.go` (used in ~15 places)
**Apply to:** Remote kill/rename blocked, QR error states
```go
m.toast = "message text"
m.toastKind = toastInfo // or toastError
m.toastExp = time.Now().Add(2 * time.Second) // 2s info, 3s error
return m, nil
```

### tea.Cmd Async Pattern
**Source:** `internal/tui/cmds.go` lines 12-17 (`fetchSessions`)
**Apply to:** `fetchRemoteSessions`
```go
func fetchXxx(args...) tea.Cmd {
	return func() tea.Msg {
		result, err := doWork(args...)
		return xxxMsg{result: result, err: err}
	}
}
```

### Key Binding Definition
**Source:** `internal/tui/keys.go` lines 20-67 (`defaultKeyMap`)
**Apply to:** New QR binding, modified Quit binding
```go
FieldName: key.NewBinding(
	key.WithKeys("keychar"),
	key.WithHelp("display", "description"),
),
```

### Test Model Setup
**Source:** `internal/tui/update_test.go` lines 13-21 (`testModel`)
**Apply to:** All new test functions
```go
m := testModel()
m.sessions = []daemon.SessionInfo{...}
m.selected = 0
// set additional state...
updated, cmd := m.Update(msgOrKeyPress)
result := updated.(Model)
// assert on result fields...
```

### Callback Injection for Package-Main Functions
**Source:** New pattern for Phase 78 (no existing analog -- this is the first cross-package callback)
**Apply to:** `fetchRemoteFn` in Model, constructed in `cmd_tui.go`
**Pattern:**
```go
// In internal/tui/model.go:
type FetchRemoteFn func(ctx context.Context) []ListRemoteGroup

// In internal/tui/tui.go:
func Run(client *daemon.DaemonClient, fetchFn FetchRemoteFn) error { ... }

// In cmd_tui.go (package main):
fetchFn := func(ctx context.Context) []tui.ListRemoteGroup {
    // wraps client.ListTailnetPeers() + fetchPeerSessions()
}
tui.Run(client, fetchFn)
```

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (none) | -- | -- | All files have exact analogs within the existing TUI package or cmd_cli.go |

The callback injection pattern (`fetchRemoteFn`) is the only truly new architectural pattern. It has no existing analog in the codebase because `internal/tui` has never needed to call package-main functions before. However, the closure pattern itself is idiomatic Go and the RESEARCH.md provides a concrete implementation.

## Metadata

**Analog search scope:** `internal/tui/`, `cmd_tui.go`, `cmd_cli.go`, `cmd_remote.go`, `internal/tailnet/tailnet.go`, `internal/daemon/types.go`, `internal/daemon/client.go`
**Files scanned:** 22
**Pattern extraction date:** 2026-04-15
