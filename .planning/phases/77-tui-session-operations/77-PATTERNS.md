# Phase 77: TUI Session Operations - Pattern Map

**Mapped:** 2026-04-15
**Files analyzed:** 14 new/modified files
**Analogs found:** 14 / 14

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/tui/attach.go` | controller (ExecCommand) | streaming | `cmd_attach.go` | role-match |
| `internal/tui/modal.go` | component (modal renderer) | transform | `internal/tui/help.go` | exact |
| `internal/tui/model.go` | model (state + messages) | event-driven | `internal/tui/model.go` (self) | exact -- additive |
| `internal/tui/update.go` | controller (message dispatch) | event-driven | `internal/tui/update.go` (self) | exact -- additive |
| `internal/tui/view.go` | utility (renderer) | transform | `internal/tui/view.go` (self) | exact -- additive |
| `internal/tui/keys.go` | config (keybinding definitions) | event-driven | `internal/tui/keys.go` (self) | exact -- additive |
| `internal/tui/styles.go` | config (style tokens) | transform | `internal/tui/styles.go` (self) | exact -- additive |
| `internal/tui/help.go` | utility (overlay renderer) | transform | `internal/tui/help.go` (self) | exact -- additive |
| `internal/tui/cmds.go` | service (async data fetch) | request-response | `internal/tui/cmds.go` (self) | exact -- additive |
| `internal/tui/tui.go` | controller (entry point) | request-response | `internal/tui/tui.go` (self) | exact -- additive |
| `internal/tui/update_test.go` | test | -- | `internal/tui/update_test.go` (self) | exact -- additive |
| `internal/tui/view_test.go` | test | -- | `internal/tui/view_test.go` (self) | exact -- additive |
| `internal/tui/help_test.go` | test | -- | `internal/tui/help_test.go` (self) | exact -- additive |
| `internal/tui/attach_test.go` | test | -- | `internal/tui/update_test.go` | role-match |

## Pattern Assignments

---

### `internal/tui/attach.go` (controller, ExecCommand -- NEW FILE)

**Analog:** `cmd_attach.go` (lines 1-489)

**Package + imports pattern** (new `internal/tui` file, follows existing package convention in `tui.go` line 1):
```go
package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/relay"
	"github.com/scottkw/agenthub/internal/statusbar"
	"golang.org/x/term"
)
```

**lockedWriter pattern** (copy from `cmd_attach.go` lines 26-36):
```go
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (lw *lockedWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}
```

**ExecCommand struct pattern** (implements `tea.ExecCommand` -- see RESEARCH.md Pattern 1):
```go
type attachCmd struct {
	client    *daemon.DaemonClient
	sessionID string
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
}

func (a *attachCmd) SetStdin(r io.Reader)  { a.stdin = r }
func (a *attachCmd) SetStdout(w io.Writer) { a.stdout = w }
func (a *attachCmd) SetStderr(w io.Writer) { a.stderr = w }
```

**Run() method -- session lookup pattern** (adapted from `cmd_attach.go` lines 82-103):
```go
func (a *attachCmd) Run() error {
	port, err := a.client.GetRelayPort()
	if err != nil {
		return err
	}

	sessions, err := a.client.ListSessions()
	if err != nil {
		return err
	}
	var session *daemon.SessionInfo
	for _, s := range sessions {
		s := s
		if s.ID == a.sessionID {
			session = &s
			break
		}
	}
	if session == nil {
		return fmt.Errorf("session %q not found", a.sessionID)
	}
	// ...
}
```

**WebSocket dial pattern** (from `cmd_attach.go` lines 106-132):
```go
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws", port, a.sessionID)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
```

**Status bar + raw mode pattern** (from `cmd_attach.go` lines 134-175, adapted for io.Reader/io.Writer from tea.Exec):
```go
	lw := &lockedWriter{w: a.stdout}

	// Status bar: only if stdin is *os.File and a terminal
	var bar *statusbar.Bar
	if f, ok := a.stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		createdAt, _ := time.Parse(time.RFC3339, session.CreatedAt)
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		bar = statusbar.New(lw, statusbar.Options{
			SessionName: session.Name,
			AgentType:   session.CLI,
			Hostname:    session.Hostname,
			CreatedAt:   createdAt,
			Position:    statusbar.Bottom,
			Fd:          f.Fd(),
		})
		bar.Start()
		defer bar.Stop()
	}

	// Raw mode: only if stdin is *os.File
	if f, ok := a.stdin.(*os.File); ok {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err != nil {
			return err
		}
		defer term.Restore(int(f.Fd()), oldState)

		if cols, rows, err := term.GetSize(int(f.Fd())); err == nil {
			frame := makeClientResizeFrame(uint16(cols), uint16(rows))
			_ = conn.Write(ctx, websocket.MessageBinary, frame)
		}
	}
```

**I/O pump delegation** (reuses `attachSession` -- but it's in `package main`, so must be duplicated or extracted; see Pitfall 5 in RESEARCH.md):

The functions `attachSession`, `stdinPump`, `wsOutputPump`, and `makeClientResizeFrame` currently live in `cmd_attach.go` (package main, lines 348-488). They CANNOT be imported by `internal/tui`. Two options per RESEARCH.md:

- **Option A (recommended):** Extract into `internal/attach/attach.go` -- import from both `cmd_attach.go` and `internal/tui/attach.go`
- **Option B (fallback):** Duplicate the 5 functions (~140 lines) directly in `internal/tui/attach.go`

If Option A, the `cmd_attach.go` lines 348-488 become thin wrappers. If Option B, copy these verbatim:
- `attachSession` (lines 348-369)
- `stdinPump` (lines 373-415)
- `wsOutputPump` (lines 422-451)
- `makeClientResizeFrame` (lines 482-488)

**watchResize platform-specific pattern** (from `cmd_attach_unix.go` lines 18-37):
```go
func watchResize(ctx context.Context, conn *websocket.Conn) {
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	go func() {
		defer signal.Stop(winchCh)
		for {
			select {
			case <-winchCh:
				cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
				// ...
			case <-ctx.Done():
				return
			}
		}
	}()
}
```

If extracted to `internal/attach`, needs build tags `//go:build !windows` / `//go:build windows`.

---

### `internal/tui/modal.go` (component, modal renderer -- NEW FILE)

**Analog:** `internal/tui/help.go` (lines 1-96)

**Package + imports pattern** (follows `help.go` line 1):
```go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)
```

**Modal overlay pattern** (copy from `help.go` lines 11-47 -- `renderHelpOverlay`):
```go
func (m Model) renderNewSessionModal() string {
	content := m.buildNewSessionContent()

	overlayWidth := max(50, min(70, m.width-10))

	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.BorderNormal).
		Width(overlayWidth - 2).
		Padding(1, 2).
		Background(m.styles.BgModal).
		Render(content)

	// Add title to top border (same technique as help.go lines 24-43)
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.BorderAccent).
		Render(" New Session ")

	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		topBorder := lines[0]
		titleWidth := lipgloss.Width(title)
		borderWidth := lipgloss.Width(topBorder)
		if borderWidth > titleWidth+4 {
			insertPos := 3
			runes := []rune(topBorder)
			titleRunes := []rune(title)
			copy(runes[insertPos:insertPos+len(titleRunes)], titleRunes)
			lines[0] = string(runes)
		}
		bordered = strings.Join(lines, "\n")
	}

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center, bordered)
}
```

**Kill confirmation modal** (smaller variant of the same pattern, per UI-SPEC Surface 3):
```go
func (m Model) renderKillConfirmModal() string {
	overlayWidth := max(40, min(55, m.width-20))
	// Same lipgloss.RoundedBorder + lipgloss.Place pattern
	// Title: "Kill Session" in fg.danger (NOT border.accent)
}
```

**Help content builder pattern** (from `help.go` lines 50-96 -- `buildHelpContent`):
```go
func (m Model) buildNewSessionContent() string {
	labelStyle := lipgloss.NewStyle().Foreground(m.styles.FgNormal)
	focusedLabelStyle := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgAccent)

	var sections []string

	// Agent picker row
	lbl := labelStyle
	if m.focusedField == 0 {
		lbl = focusedLabelStyle
	}
	sections = append(sections, fmt.Sprintf("  %s  %s",
		lbl.Render("Agent:"), m.renderAgentPicker()))

	// Directory field
	// Arguments field
	// Hint row

	return strings.Join(sections, "\n")
}
```

---

### `internal/tui/model.go` (model, state + messages -- MODIFY)

**Analog:** `internal/tui/model.go` (self -- lines 1-39)

**Existing struct** (lines 10-25):
```go
type Model struct {
	client    *daemon.DaemonClient
	sessions  []daemon.SessionInfo
	webStatus daemon.WebServerStatusResponse
	selected  int
	width     int
	height    int
	showHelp  bool
	loading   bool
	err       error
	hasDark   bool
	styles    Styles
	keys      KeyMap
	toast     string
	toastExp  time.Time
}
```

**Add these fields** (per UI-SPEC Data Flow Contract and RESEARCH.md Pattern 2):
```go
	// Modal state (Phase 77)
	modal         modalState
	agentIdx      int                // current agent picker index
	dirInput      textinput.Model    // directory field in new-session modal
	argsInput     textinput.Model    // arguments field in new-session modal
	focusedField  int                // 0=agent, 1=directory, 2=arguments
	detectedCLIs  []pty.DetectedCLI  // cached on first modal open

	// Kill confirmation state (Phase 77)
	killTarget    *daemon.SessionInfo
	killFocusYes  bool

	// Inline rename state (Phase 77)
	editing       bool
	editInput     textinput.Model
	editOriginal  string
	editSessionID string

	// Toast enhancement (Phase 77)
	toastKind     toastKind
```

**New type definitions** (add before or after Model struct, following Go convention):
```go
type modalState int
const (
	modalNone modalState = iota
	modalNewSession
	modalKillConfirm
)

type toastKind int
const (
	toastInfo toastKind = iota
	toastSuccess
	toastError
)
```

**New message types** (add after existing `tickMsg` at line 39):
```go
type attachDoneMsg struct {
	err error
}

type createSessionMsg struct {
	id  string
	err error
}

type killSessionMsg struct {
	err error
}

type renameSessionMsg struct {
	err error
}
```

**New import needed**: `"charm.land/bubbles/v2/textinput"` and `"github.com/scottkw/agenthub/internal/pty"`.

---

### `internal/tui/update.go` (controller, message dispatch -- MODIFY)

**Analog:** `internal/tui/update.go` (self -- lines 1-127)

**Existing Update switch** (lines 11-56) -- add new message cases after `tickMsg`:
```go
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
```

**Existing handleKey** (lines 60-127) -- replace with priority-based dispatch per RESEARCH.md Pattern 3:
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
		if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}
	// Priority 5: Main view
	return m.handleMainKey(msg)
}
```

**Existing main view key handler** (lines 71-127 -- rename to `handleMainKey` and modify):

Change `case key.Matches(msg, m.keys.Attach):` (lines 107-111) from toast to actual dispatch:
```go
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
```

Change `case key.Matches(msg, m.keys.New):` (lines 113-117) from toast to modal open:
```go
case key.Matches(msg, m.keys.New):
	return m.openNewSessionModal()
```

Replace the silent `d`/`e` consumption (lines 120-124) with new actions:
```go
case key.Matches(msg, m.keys.Kill):
	if len(m.sessions) == 0 {
		return m, nil
	}
	s := m.sessions[m.selected]
	m.modal = modalKillConfirm
	m.killTarget = &s
	m.killFocusYes = false // default No
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
```

**New handler methods** -- add `handleRenameKey`, `handleKillConfirmKey`, `handleNewSessionKey` (patterns from RESEARCH.md Examples 4 and 5):

`handleRenameKey` (from RESEARCH.md Example 5, lines 622-649):
```go
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
```

`handleKillConfirmKey` (from RESEARCH.md Example 4, lines 564-598):
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
```

**New import needed**: `"charm.land/bubbles/v2/textinput"`, `"strings"`.

---

### `internal/tui/view.go` (utility, renderer -- MODIFY)

**Analog:** `internal/tui/view.go` (self -- lines 1-290)

**renderFull -- add modal rendering** (modify lines 41-62):

After the help overlay check (line 42-44), add modal rendering before or after content composition:
```go
func (m Model) renderFull() string {
	if m.showHelp {
		return m.renderHelpOverlay()
	}

	header := m.renderHeader()
	colHeaders := m.renderColumnHeaders()
	list := m.renderSessionList()
	separator := ""
	footer := m.renderFooter()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header, colHeaders, list, separator, footer)

	// Modal overlays (rendered on top of content)
	if m.modal == modalNewSession {
		return m.renderNewSessionModal()
	}
	if m.modal == modalKillConfirm {
		return m.renderKillConfirmModal()
	}

	return content
}
```

**renderSessionRow -- inline edit support** (modify lines 173-211):

When `m.editing && s.ID == m.editSessionID`, replace the Name column with textinput view:
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

	// Inline rename: replace name with textinput view
	var name string
	if m.editing && s.ID == m.editSessionID {
		name = m.editInput.View()
	} else {
		name = truncate(s.Name, nameWidth)
	}

	// ... rest unchanged from lines 192-211 ...
}
```

**renderWebStatus -- toast kind coloring** (modify lines 237-239):

Replace the single-color toast with kind-based styling:
```go
	if m.toast != "" && time.Now().Before(m.toastExp) {
		var toastColor color.Color
		switch m.toastKind {
		case toastSuccess:
			toastColor = m.styles.StatusRunning
		case toastError:
			toastColor = m.styles.StatusErrored
		default: // toastInfo
			toastColor = m.styles.FgMuted
		}
		webPart = lipgloss.NewStyle().Foreground(toastColor).Render(m.toast)
	}
```

**renderHintBar -- update** (modify line 252):

Change from Phase 76 hints to Phase 77 full hint bar (per UI-SPEC):
```go
func (m Model) renderHintBar() string {
	hint := "j/k Up/Down  Enter Attach  n New  d Kill  r Rename  ? Help  q Quit"
	return lipgloss.NewStyle().Foreground(m.styles.FgMuted).
		Width(m.width).Render(hint)
}
```

---

### `internal/tui/keys.go` (config, keybinding definitions -- MODIFY)

**Analog:** `internal/tui/keys.go` (self -- lines 1-57)

**Existing KeyMap struct** (lines 6-16) -- add new bindings:
```go
type KeyMap struct {
	Quit    key.Binding
	Help    key.Binding
	Up      key.Binding
	Down    key.Binding
	Refresh key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Attach  key.Binding
	New     key.Binding
	Kill    key.Binding   // NEW -- Phase 77
	Rename  key.Binding   // NEW -- Phase 77
}
```

**Modify `defaultKeyMap()`** (lines 18-57):

Reassign `r` from Refresh to Rename, `R` to Refresh:
```go
	Refresh: key.NewBinding(
		key.WithKeys("R"),              // was "r"
		key.WithHelp("R", "refresh list"),
	),
	// ...
	Kill: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "kill session"),
	),
	Rename: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "rename session"),
	),
```

Remove `// reserved -- Phase 77` comments from Attach and New.

---

### `internal/tui/styles.go` (config, style tokens -- MODIFY)

**Analog:** `internal/tui/styles.go` (self -- lines 1-46)

**Existing Styles struct** (lines 11-25) -- add new tokens per UI-SPEC:
```go
type Styles struct {
	// ... existing 13 fields ...
	BgModal       color.Color  // NEW -- Phase 77
	FgDanger      color.Color  // NEW -- Phase 77
	FgInput       color.Color  // NEW -- Phase 77
	BgInput       color.Color  // NEW -- Phase 77
	FgPlaceholder color.Color  // NEW -- Phase 77
	FgFocusedLabel color.Color // NEW -- Phase 77
}
```

**Extend `newStyles()`** (after line 44, following existing pattern):
```go
	BgModal:        ld(lipgloss.Color("#F5F5F5"), lipgloss.Color("#1C1C1C")),
	FgDanger:       ld(lipgloss.Color("#D70000"), lipgloss.Color("#FF5F5F")),
	FgInput:        ld(lipgloss.Color("#000000"), lipgloss.Color("#FFFFFF")),
	BgInput:        ld(lipgloss.Color("#E4E4E4"), lipgloss.Color("#303030")),
	FgPlaceholder:  ld(lipgloss.Color("#949494"), lipgloss.Color("#626262")),
	FgFocusedLabel: ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
```

---

### `internal/tui/help.go` (utility, overlay renderer -- MODIFY)

**Analog:** `internal/tui/help.go` (self -- lines 1-96)

**Modify `buildHelpContent()`** (lines 50-96):

Change group name "Actions" to "Sessions" (line 74), add new bindings, update `r` to `R` for refresh:
```go
	// Group 1: Navigation
	sections = append(sections, groupStyle.Render("Navigation"))
	sections = append(sections,
		formatBinding("j/k, Up/Down", "Move up/down"),
		formatBinding("g/Home", "Jump to first"),
		formatBinding("G/End", "Jump to last"),
		formatBinding("R", "Refresh list"),  // was "r"
	)

	// Group 2: Sessions (was "Actions")
	sections = append(sections, "")
	sections = append(sections, groupStyle.Render("Sessions"))
	sections = append(sections,
		formatBinding("Enter", "Attach to session"),
		formatBinding("n", "New session"),
		formatBinding("d", "Kill session"),      // NEW
		formatBinding("r", "Rename session"),     // NEW
	)

	// Group 3: General (unchanged)
```

---

### `internal/tui/cmds.go` (service, async data fetch -- MODIFY)

**Analog:** `internal/tui/cmds.go` (self -- lines 1-33)

**Existing tea.Cmd pattern** (lines 12-17, `fetchSessions`) -- add new commands following same pattern:
```go
func createSession(client *daemon.DaemonClient, cli, name, workDir string, args []string) tea.Cmd {
	return func() tea.Msg {
		id, err := client.CreateSession(cli, name, workDir, args, 0, 0)
		return createSessionMsg{id: id, err: err}
	}
}

func killSession(client *daemon.DaemonClient, id string) tea.Cmd {
	return func() tea.Msg {
		err := client.KillSession(id)
		return killSessionMsg{err: err}
	}
}

func renameSession(client *daemon.DaemonClient, id, name string) tea.Cmd {
	return func() tea.Msg {
		err := client.RenameSession(id, name)
		return renameSessionMsg{err: err}
	}
}
```

The DaemonClient method signatures (from `internal/daemon/client.go`):
- `CreateSession(cli, name, workDir string, args []string, cols, rows int) (string, error)` -- line 59
- `KillSession(id string) error` -- line 69
- `RenameSession(id, name string) error` -- line 74

---

### `internal/tui/tui.go` (controller, entry point -- MODIFY)

**Analog:** `internal/tui/tui.go` (self -- lines 1-39)

**Modify `newModel()`** (lines 18-25) to cache DetectCLIs:

Per RESEARCH.md anti-pattern: call `pty.DetectCLIs()` once and store in model. Either call in `newModel` or lazily on first modal open. The UI-SPEC says cache for TUI lifetime:
```go
func newModel(client *daemon.DaemonClient) Model {
	return Model{
		client:       client,
		loading:      true,
		keys:         defaultKeyMap(),
		styles:       newStyles(true),
		detectedCLIs: pty.DetectCLIs(), // cache for TUI lifetime
	}
}
```

**New import needed**: `"github.com/scottkw/agenthub/internal/pty"`

---

### Test Files (MODIFY existing, NEW attach_test.go)

**Analog:** `internal/tui/update_test.go` (lines 1-204), `view_test.go` (lines 1-168), `help_test.go` (lines 1-83)

**testModel() factory** (from `update_test.go` lines 11-19 -- extend with Phase 77 fields):
```go
func testModel() Model {
	m := newModel(nil)
	m.width = 120
	m.height = 24
	m.hasDark = true
	m.styles = newStyles(true)
	m.loading = false
	m.detectedCLIs = []pty.DetectedCLI{
		{Name: "claude", DisplayName: "Claude Code", Path: "/usr/bin/claude"},
	}
	return m
}
```

**State transition test pattern** (from `update_test.go` lines 21-40 -- `TestUpdate_SessionsMsg`):
```go
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
```

**View content assertion pattern** (from `view_test.go` lines 11-45 -- `TestView_SessionList`):
```go
func TestView_NewSessionModal(t *testing.T) {
	m := testModel()
	m.modal = modalNewSession
	m.focusedField = 0

	v := m.View()
	content := v.Content

	checks := []string{"New Session", "Agent:", "Directory:", "Arguments:"}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("modal view missing %q", want)
		}
	}
}
```

**Help content update test** (modify `help_test.go` lines 55-68):

The `TestHelpOverlay_NoReservedHiddenKeys` test must be updated -- it currently asserts that "Kill" and "Rename" are NOT present. In Phase 77, they MUST be present. Update the assertion:
```go
func TestHelp_UpdatedBindings(t *testing.T) {
	m := testModel()
	content := m.buildHelpContent()

	// Phase 77: Kill and Rename should now appear in help
	required := []string{"Kill session", "Rename session", "Sessions"}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("help content missing %q", want)
		}
	}
	// "Actions" group renamed to "Sessions"
	if strings.Contains(content, "Actions") {
		t.Error("help should use 'Sessions' group, not 'Actions'")
	}
}
```

**Test for reserved key toast** (modify `update_test.go` line 179-189):

`TestUpdate_ReservedKeysShowToast` currently expects "Coming in next update" from Enter key. Remove or update this test since Enter now dispatches `tea.Exec`.

---

## Shared Patterns

### DaemonClient API Consumption (tea.Cmd wrapper)
**Source:** `internal/tui/cmds.go` lines 12-17 (`fetchSessions`)
**Apply to:** All new `tea.Cmd` functions: `createSession`, `killSession`, `renameSession`

```go
func fetchSessions(client *daemon.DaemonClient) tea.Cmd {
	return func() tea.Msg {
		sessions, err := client.ListSessions()
		return sessionsMsg{sessions: sessions, err: err}
	}
}
```

### Modal Overlay Rendering (lipgloss.Place + RoundedBorder)
**Source:** `internal/tui/help.go` lines 11-47 (`renderHelpOverlay`)
**Apply to:** `renderNewSessionModal()`, `renderKillConfirmModal()` in `modal.go`

```go
bordered := lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(m.styles.BorderNormal).
	Width(overlayWidth - 2).
	Padding(1, 2).
	Render(content)

return lipgloss.Place(m.width, m.height,
	lipgloss.Center, lipgloss.Center, bordered)
```

### Border Title Insertion (rune copy into top border)
**Source:** `internal/tui/help.go` lines 24-43
**Apply to:** All modals with titles

```go
lines := strings.Split(bordered, "\n")
if len(lines) > 0 {
	topBorder := lines[0]
	titleWidth := lipgloss.Width(title)
	borderWidth := lipgloss.Width(topBorder)
	if borderWidth > titleWidth+4 {
		insertPos := 3
		runes := []rune(topBorder)
		titleRunes := []rune(title)
		copy(runes[insertPos:insertPos+len(titleRunes)], titleRunes)
		lines[0] = string(runes)
	}
	bordered = strings.Join(lines, "\n")
}
```

### Priority-Based Key Dispatch
**Source:** New pattern for Phase 77 (documented in RESEARCH.md Pattern 3)
**Apply to:** `internal/tui/update.go` `handleKey` method

```go
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.editing { return m.handleRenameKey(msg) }
	if m.modal == modalKillConfirm { return m.handleKillConfirmKey(msg) }
	if m.modal == modalNewSession { return m.handleNewSessionKey(msg) }
	if m.showHelp { /* help dismiss only */ }
	return m.handleMainKey(msg)
}
```

### textinput Delegation (intercept first, then delegate)
**Source:** New pattern for Phase 77 (documented in RESEARCH.md Pitfall 3)
**Apply to:** `handleRenameKey`, `handleNewSessionKey`

```go
// Intercept Tab, Shift+Tab, Enter, Esc BEFORE textinput gets them
s := msg.String()
switch {
case s == "enter":
	// handle submit
case s == "esc":
	// handle cancel
case s == "tab", s == "shift+tab":
	// handle focus cycling
default:
	// Delegate to textinput.Update
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}
```

### Test Model Factory
**Source:** `internal/tui/update_test.go` lines 11-19 (`testModel`)
**Apply to:** All `*_test.go` files in `internal/tui/`

```go
func testModel() Model {
	m := newModel(nil)
	m.width = 120
	m.height = 24
	m.hasDark = true
	m.styles = newStyles(true)
	m.loading = false
	return m
}
```

### Error Handling in Message Handlers (toast + refresh)
**Source:** Pattern established by existing `sessionsMsg` handler (`update.go` lines 27-39)
**Apply to:** `attachDoneMsg`, `createSessionMsg`, `killSessionMsg`, `renameSessionMsg`

```go
case someMsg:
	if msg.err != nil {
		m.toast = fmt.Sprintf("Operation failed: %s", msg.err)
		m.toastKind = toastError
		m.toastExp = time.Now().Add(3 * time.Second)
	} else {
		m.toast = "Operation succeeded"
		m.toastKind = toastSuccess
		m.toastExp = time.Now().Add(2 * time.Second)
	}
	return m, fetchSessions(m.client)
```

### Adaptive Color Token (LightDark)
**Source:** `internal/tui/styles.go` lines 29-46 (`newStyles`)
**Apply to:** All new color tokens in `styles.go`

```go
ld := lipgloss.LightDark(hasDark)
BgModal: ld(lipgloss.Color("#F5F5F5"), lipgloss.Color("#1C1C1C")),
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/tui/attach.go` (partial) | controller | streaming | No `tea.ExecCommand` implementation exists in codebase. The ExecCommand interface pattern comes from RESEARCH.md Pattern 1 (verified from bubbletea v2.0.5 source). The attach *logic* analog is `cmd_attach.go`. |

**Note:** All other files have exact or role-match analogs. The `textinput.Model` usage has no codebase analog but is well-documented in RESEARCH.md Pattern 2 with verified API signatures from bubbles v2.1.0 source.

---

## Import Path Reference

All files in `internal/tui/` use these import paths (extends Phase 76 table):

| Import | Package | Purpose |
|--------|---------|---------|
| `tea "charm.land/bubbletea/v2"` | Bubble Tea v2 | MVU framework, `tea.Exec` for attach |
| `"charm.land/lipgloss/v2"` | Lip Gloss v2 | Modal borders, adaptive colors, layout |
| `"charm.land/bubbles/v2/key"` | Bubbles v2 key | Key binding definitions + matching |
| `"charm.land/bubbles/v2/textinput"` | Bubbles v2 textinput | **NEW** -- Rename inline edit, modal form fields |
| `"github.com/scottkw/agenthub/internal/daemon"` | Daemon client + types | DaemonClient methods, SessionInfo |
| `"github.com/scottkw/agenthub/internal/pty"` | **NEW** -- PTY detect | `DetectCLIs()`, `DetectedCLI` struct |
| `"github.com/scottkw/agenthub/internal/relay"` | **NEW** -- Relay protocol | `MakeInputFrame`, `ParseFrame` (attach I/O) |
| `"github.com/scottkw/agenthub/internal/statusbar"` | **NEW** -- Status bar | `statusbar.New`, `statusbar.Bar` (during attach) |
| `"github.com/coder/websocket"` | **NEW** -- WebSocket | `websocket.Dial` (attach) |
| `"golang.org/x/term"` | **NEW** -- Terminal | `term.MakeRaw`, `term.Restore`, `term.GetSize` (attach) |

No new `go get` needed -- all dependencies already in go.mod from Phase 76 and earlier phases.

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub` -- `internal/tui/`, `cmd_attach*.go`, `internal/daemon/client.go`, `internal/pty/detect.go`
**Key files read:** `internal/tui/model.go`, `internal/tui/update.go`, `internal/tui/view.go`, `internal/tui/keys.go`, `internal/tui/styles.go`, `internal/tui/help.go`, `internal/tui/cmds.go`, `internal/tui/tui.go`, `internal/tui/update_test.go`, `internal/tui/view_test.go`, `internal/tui/help_test.go`, `cmd_attach.go`, `cmd_attach_unix.go`, `internal/daemon/client.go`, `internal/pty/detect.go`
**Pattern extraction date:** 2026-04-15
