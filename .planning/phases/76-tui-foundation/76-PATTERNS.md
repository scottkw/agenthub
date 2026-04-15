# Phase 76: TUI Foundation - Pattern Map

**Mapped:** 2026-04-15
**Files analyzed:** 13 new/modified files
**Analogs found:** 13 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/tui/tui.go` | controller (entry point) | request-response | `internal/statusbar/bar.go` | role-match |
| `internal/tui/model.go` | model (state + messages) | event-driven | `internal/daemon/types.go` | role-match |
| `internal/tui/update.go` | controller (message dispatch) | event-driven | `internal/statusbar/bar.go` | role-match |
| `internal/tui/view.go` | utility (renderer) | transform | `internal/statusbar/bar.go` | role-match |
| `internal/tui/styles.go` | config (style tokens) | transform | `internal/statusbar/bar.go` (ANSI constants) | partial |
| `internal/tui/keys.go` | config (keybinding definitions) | event-driven | -- (new pattern, Charm-specific) | no-analog |
| `internal/tui/help.go` | utility (overlay renderer) | transform | `internal/statusbar/bar.go` (format method) | partial |
| `internal/tui/cmds.go` | service (async data fetch) | request-response | `internal/daemon/client.go` | role-match |
| `internal/tui/tui_test.go` | test | -- | `internal/statusbar/bar_test.go` | exact |
| `cmd_tui.go` | utility (CLI command) | request-response | `cmd_attach.go` | exact |
| `main.go` | controller (CLI dispatch) | request-response | `main.go` (self) | exact -- additive |
| `internal/daemon/types.go` | model (types) | -- | `internal/daemon/types.go` (self) | exact -- additive |
| `internal/daemon/engine.go` | service (session listing) | CRUD | `internal/daemon/engine.go` (self) | exact -- additive |

## Pattern Assignments

---

### `cmd_tui.go` (utility, CLI command -- NEW FILE)

**Analog:** `cmd_attach.go`

**Package + imports pattern** (`cmd_attach.go` lines 1-23):
```go
package main

import (
	"fmt"
	"os"

	"github.com/scottkw/agenthub/internal/daemon"
	"github.com/scottkw/agenthub/internal/tui"
	"golang.org/x/term"
)
```

**TTY detection pattern** (`cmd_attach.go` lines 77-79):
```go
// Must be run in an interactive terminal.
if !term.IsTerminal(int(os.Stdin.Fd())) {
    return fmt.Errorf("attach: stdin is not a terminal")
}
```

For `cmd_tui.go`, check stdout (not stdin) per UI-SPEC:
```go
if !term.IsTerminal(int(os.Stdout.Fd())) {
    return fmt.Errorf("agenthub tui requires a terminal. Redirect to a TTY or use 'agenthub list' instead")
}
```

**Function signature pattern** (`cmd_attach.go` line 41):
```go
func cmdAttach(client *daemon.DaemonClient, args []string) error {
```

For `cmd_tui.go`, use the simpler form (no args needed):
```go
func cmdTUI(client *daemon.DaemonClient) error {
```

**Health check before operation** (from RESEARCH.md CLI Wiring section):
```go
if err := client.Health(); err != nil {
    return fmt.Errorf("cannot connect to daemon: %w", err)
}
return tui.Run(client)
```

---

### `main.go` (additive modification -- CLI dispatch)

**Analog:** `main.go` (self)

**Switch-case extension pattern** (`main.go` lines 168-197):
```go
switch cmd {
case "new":
    err = cmdNew(client, cmdArgs, extraArgs, os.Stdout)
case "list":
    err = cmdList(client, cmdArgs, os.Stdout)
// ... existing cases ...
case "daemon":
    err = cmdDaemonStatus(client, cmdArgs[1:], os.Stdout)
default:
    fmt.Fprintf(os.Stderr, "agenthub: unknown command %q\nRun 'agenthub --help' for usage.\n", cmd)
    os.Exit(1)
}
```

Add before `default`:
```go
case "tui":
    err = cmdTUI(client)
```

**Usage string update** (`cmd_cli.go` lines 23-53) -- add to Commands section:
```
  tui                                       Launch interactive terminal UI
```

---

### `internal/tui/tui.go` (controller, entry point -- NEW FILE)

**Analog:** `internal/statusbar/bar.go` (constructor + lifecycle pattern)

**Package declaration + imports** (Bubble Tea v2 pattern from RESEARCH.md):
```go
package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/scottkw/agenthub/internal/daemon"
)
```

**Exported Run() entry point pattern** (modeled on `statusbar.New` + `statusbar.Start` combined into a single call):

The `statusbar/bar.go` constructor pattern (`bar.go` lines 69-74):
```go
func New(w io.Writer, opts Options) *Bar {
    return &Bar{
        w:    w,
        opts: opts,
    }
}
```

For TUI, combine constructor and runner:
```go
func Run(client *daemon.DaemonClient) error {
    p := tea.NewProgram(newModel(client))
    _, err := p.Run()
    return err
}
```

**newModel constructor** (mirrors `statusbar.New` -- struct with config fields + initial state):
```go
func newModel(client *daemon.DaemonClient) Model {
    return Model{
        client:  client,
        loading: true,
        keys:    defaultKeyMap(),
        styles:  newStyles(true), // assume dark until BackgroundColorMsg
    }
}
```

---

### `internal/tui/model.go` (model, state + message types -- NEW FILE)

**Analog:** `internal/daemon/types.go` (type definition patterns)

**Type definition pattern** (`internal/daemon/types.go` lines 4-13):
```go
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"`
}
```

For `model.go`, define the Model struct (no JSON tags -- internal only):
```go
package tui

import (
    "time"

    "github.com/scottkw/agenthub/internal/daemon"
)

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

**Message type pattern** (follow Go convention of small unexported structs):
```go
type sessionsMsg struct {
    sessions []daemon.SessionInfo
    err      error
}

type webStatusMsg struct {
    status daemon.WebServerStatusResponse
    err    error
}

type tickMsg time.Time
```

---

### `internal/tui/update.go` (controller, message dispatch -- NEW FILE)

**Analog:** `internal/statusbar/bar.go` (event handling in tickLoop + draw)

**Bubble Tea Update signature** (from RESEARCH.md verified v2 pattern):
```go
package tui

import (
    tea "charm.land/bubbletea/v2"
    "charm.land/bubbles/v2/key"
)

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
        // ...
    case webStatusMsg:
        // ...
    case tickMsg:
        // ...
    }
    return m, nil
}
```

**Key dispatch pattern** (mirrors `cmd_attach.go` switch on message type, lines 106-122 of `relay/server.go`):
```go
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    if m.showHelp {
        if key.Matches(msg, m.keys.Help) || msg.String() == "esc" {
            m.showHelp = false
            return m, nil
        }
        return m, nil
    }
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
    // ... etc
    }
    return m, nil
}
```

**Init method** (returns batch of initial commands):
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

---

### `internal/tui/view.go` (utility, renderer -- NEW FILE)

**Analog:** `internal/statusbar/bar.go` (format method, lines 187-232)

**String formatting/truncation pattern** (`bar.go` lines 211-224):
```go
runeLen := utf8.RuneCountInString(text)
if runeLen > cols {
    if cols > 1 {
        text = string([]rune(text)[:cols-1]) + "\u2026"
    } else {
        text = "\u2026"
    }
    runeLen = utf8.RuneCountInString(text)
}
```

For TUI column truncation (truncate to `max-3` and append `...`):
```go
func truncate(s string, maxWidth int) string {
    if utf8.RuneCountInString(s) <= maxWidth {
        return s
    }
    if maxWidth <= 3 {
        return s[:maxWidth]
    }
    return string([]rune(s)[:maxWidth-3]) + "..."
}
```

**View method pattern** (Bubble Tea v2 -- returns `tea.View`, NOT string):
```go
func (m Model) View() tea.View {
    var v tea.View
    v.AltScreen = true
    // NO mouse mode per REQUIREMENTS.md

    if m.width < 60 || m.height < 10 {
        v.SetContent(lipgloss.Place(m.width, m.height,
            lipgloss.Center, lipgloss.Center,
            "Terminal too small (need 60x10)"))
        return v
    }

    if m.loading && m.width == 0 {
        v.SetContent("Loading sessions...")
        return v
    }

    v.SetContent(m.renderFull())
    return v
}
```

**Render composition** (using `lipgloss.JoinVertical` or string concatenation):
```go
func (m Model) renderFull() string {
    header := m.renderHeader()
    colHeaders := m.renderColumnHeaders()
    list := m.renderSessionList()
    separator := ""
    footer := m.renderFooter()

    content := lipgloss.JoinVertical(lipgloss.Left,
        header, colHeaders, list, separator, footer)

    if m.showHelp {
        overlay := m.renderHelpOverlay()
        return lipgloss.Place(m.width, m.height,
            lipgloss.Center, lipgloss.Center, overlay,
            lipgloss.WithWhitespaceChars(" "),
            lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}))
    }
    return content
}
```

**Session row rendering** (column layout from UI-SPEC):
```go
func (m Model) renderSessionRow(s daemon.SessionInfo, idx int) string {
    isSelected := idx == m.selected

    cursor := "  "
    if isSelected {
        cursor = "> "
    }

    nameWidth := m.width - 53 // fixed: cursor(2)+status(2)+agent(12)+host(20)+viewers(7)+gaps(10)
    if nameWidth < 8 {
        nameWidth = 8
    }

    // ... truncate columns, format with fmt.Sprintf ...
}
```

**Status glyph mapping** (from UI-SPEC status indicators table):
```go
func statusGlyph(status string, s Styles) (string, lipgloss.TerminalColor) {
    switch status {
    case "idle":
        return "\u25CB", s.StatusIdle     // WHITE CIRCLE
    case "waiting":
        return "\u25CF", s.StatusWaiting  // BLACK CIRCLE (yellow)
    case "errored":
        return "\u2716", s.StatusErrored  // HEAVY MULTIPLICATION X
    default: // "running" or unrecognized
        return "\u25CF", s.StatusRunning  // BLACK CIRCLE (green)
    }
}
```

---

### `internal/tui/styles.go` (config, style tokens -- NEW FILE)

**Analog:** `internal/statusbar/bar.go` (ANSI constant block, lines 16-25)

**Constant block pattern** (`bar.go` lines 16-25):
```go
const (
    setScrollRegion   = "\033[%d;%dr"
    resetScrollRegion = "\033[r"
    cursorSave        = "\033[s"
    cursorRestore     = "\033[u"
    moveCursor        = "\033[%d;1H"
    eraseLineEntire   = "\033[2K"
    reverseVideo      = "\033[7m"
    sgrReset          = "\033[m"
)
```

For `styles.go`, use Lip Gloss instead of raw ANSI (UI-SPEC mandates no raw ANSI in TUI):
```go
package tui

import (
    "charm.land/lipgloss/v2"
)

type Styles struct {
    FgNormal      lipgloss.TerminalColor
    FgMuted       lipgloss.TerminalColor
    FgAccent      lipgloss.TerminalColor
    BgSelected    lipgloss.TerminalColor
    FgSelected    lipgloss.TerminalColor
    StatusRunning lipgloss.TerminalColor
    StatusIdle    lipgloss.TerminalColor
    StatusWaiting lipgloss.TerminalColor
    StatusErrored lipgloss.TerminalColor
    WebOn         lipgloss.TerminalColor
    WebOff        lipgloss.TerminalColor
    BorderNormal  lipgloss.TerminalColor
    BorderAccent  lipgloss.TerminalColor
}

func newStyles(hasDark bool) Styles {
    ld := lipgloss.LightDark(hasDark)
    return Styles{
        FgNormal:      ld(lipgloss.Color("#303030"), lipgloss.Color("#C6C6C6")),
        FgMuted:       ld(lipgloss.Color("#949494"), lipgloss.Color("#626262")),
        FgAccent:      ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
        BgSelected:    ld(lipgloss.Color("#E4E4E4"), lipgloss.Color("#303030")),
        FgSelected:    ld(lipgloss.Color("#000000"), lipgloss.Color("#FFFFFF")),
        StatusRunning: ld(lipgloss.Color("#008700"), lipgloss.Color("#5FAF5F")),
        StatusIdle:    ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
        StatusWaiting: ld(lipgloss.Color("#AF8700"), lipgloss.Color("#FFAF00")),
        StatusErrored: ld(lipgloss.Color("#D70000"), lipgloss.Color("#FF5F5F")),
        WebOn:         ld(lipgloss.Color("#008700"), lipgloss.Color("#5FAF5F")),
        WebOff:        ld(lipgloss.Color("#949494"), lipgloss.Color("#626262")),
        BorderNormal:  ld(lipgloss.Color("#BCBCBC"), lipgloss.Color("#444444")),
        BorderAccent:  ld(lipgloss.Color("#005FD7"), lipgloss.Color("#5F87FF")),
    }
}
```

---

### `internal/tui/keys.go` (config, keybinding definitions -- NEW FILE)

**Analog:** None in codebase (new Charm-specific pattern). Pattern comes from RESEARCH.md Pattern 6.

**Key binding definition pattern** (from `charm.land/bubbles/v2/key` verified API):
```go
package tui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
    Quit    key.Binding
    Help    key.Binding
    Up      key.Binding
    Down    key.Binding
    Refresh key.Binding
    Top     key.Binding
    Bottom  key.Binding
    Attach  key.Binding // reserved -- Phase 77
    New     key.Binding // reserved -- Phase 77
}

func defaultKeyMap() KeyMap {
    return KeyMap{
        Quit: key.NewBinding(
            key.WithKeys("q", "ctrl+c"),
            key.WithHelp("q", "quit"),
        ),
        Help: key.NewBinding(
            key.WithKeys("?"),
            key.WithHelp("?", "toggle help"),
        ),
        Up: key.NewBinding(
            key.WithKeys("up", "k"),
            key.WithHelp("\u2191/k", "move up"),
        ),
        Down: key.NewBinding(
            key.WithKeys("down", "j"),
            key.WithHelp("\u2193/j", "move down"),
        ),
        // ... remaining bindings ...
    }
}
```

---

### `internal/tui/help.go` (utility, overlay renderer -- NEW FILE)

**Analog:** `internal/statusbar/bar.go` `format()` method (lines 187-232) for string assembly pattern.

**String assembly with width constraints** (`bar.go` lines 187-232):
```go
func (b *Bar) format(viewerCount int, connState string, cols int) string {
    // ... build parts ...
    text := " " + strings.Join(parts, " \u2502 ") + " "
    // truncate to terminal width
    runeLen := utf8.RuneCountInString(text)
    if runeLen > cols {
        // ...
    }
    // pad to full width
    if runeLen < cols {
        text = text + strings.Repeat(" ", cols-runeLen)
    }
    return reverseVideo + text + sgrReset
}
```

For help overlay, use Lip Gloss bordered modal (per UI-SPEC and RESEARCH.md Pattern 5):
```go
func (m Model) renderHelpOverlay() string {
    content := m.buildHelpContent()

    overlayWidth := max(40, min(60, m.width-10))
    bordered := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(m.styles.BorderNormal).
        Width(overlayWidth - 2).
        Render(content)

    return lipgloss.Place(m.width, m.height,
        lipgloss.Center, lipgloss.Center, bordered)
}
```

**Help content structure** (groups: Navigation, Actions, General per UI-SPEC):
```go
func (m Model) buildHelpContent() string {
    // Group 1: Navigation
    // Group 2: Actions (includes reserved keys shown as available)
    // Group 3: General
    // Close hint: "Press ? or Esc to close"
}
```

---

### `internal/tui/cmds.go` (service, async data fetch -- NEW FILE)

**Analog:** `internal/daemon/client.go` (API call patterns)

**DaemonClient call pattern** (`client.go` lines 45-54):
```go
func (c *DaemonClient) ListSessions() ([]SessionInfo, error) {
    var sessions []SessionInfo
    if err := c.doJSON(http.MethodGet, "/sessions", nil, &sessions); err != nil {
        return nil, err
    }
    if sessions == nil {
        sessions = []SessionInfo{}
    }
    return sessions, nil
}
```

Wrap in `tea.Cmd` (returns a closure that runs in a goroutine):
```go
package tui

import (
    "time"

    tea "charm.land/bubbletea/v2"
    "github.com/scottkw/agenthub/internal/daemon"
)

func fetchSessions(client *daemon.DaemonClient) tea.Cmd {
    return func() tea.Msg {
        sessions, err := client.ListSessions()
        return sessionsMsg{sessions: sessions, err: err}
    }
}

func fetchWebStatus(client *daemon.DaemonClient) tea.Cmd {
    return func() tea.Msg {
        status, err := client.GetWebServerStatus()
        return webStatusMsg{status: status, err: err}
    }
}

func nextTick() tea.Cmd {
    return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}
```

---

### `internal/tui/tui_test.go` (and other `*_test.go` -- NEW FILES)

**Analog:** `internal/statusbar/bar_test.go`

**External test package pattern** (`bar_test.go` lines 1-12):
```go
package statusbar_test

import (
    "bytes"
    "os"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/scottkw/agenthub/internal/statusbar"
)
```

For TUI tests, use `package tui` (internal test package) since Update/View need access to unexported Model fields and message types:
```go
package tui

import (
    "strings"
    "testing"

    "github.com/scottkw/agenthub/internal/daemon"
)
```

**Test helper pattern** (`bar_test.go` lines 33-44):
```go
func testOpts() statusbar.Options {
    return statusbar.Options{
        SessionName:  "test",
        AgentType:    "claude",
        Hostname:     "host",
        CreatedAt:    time.Now(),
        Position:     statusbar.Bottom,
        Fd:           os.Stdout.Fd(),
        FallbackCols: 120,
        FallbackRows: 24,
    }
}
```

For TUI tests, create a test model factory:
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

**strings.Contains assertion pattern** (`bar_test.go` lines 64-79):
```go
checks := []string{
    "my-session",
    "claude",
    "macbook-pro",
}
for _, want := range checks {
    if !strings.Contains(output, want) {
        t.Errorf("bar output missing %q; got: %q", want, output)
    }
}
```

**Update state transition test pattern** (from RESEARCH.md testing strategy):
```go
func TestUpdate_SessionsMsg(t *testing.T) {
    m := testModel()
    m.loading = true

    sessions := []daemon.SessionInfo{
        {ID: "1", Name: "test", CLI: "claude", Hostname: "macbook", Status: "running"},
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

**Test naming convention** (from codebase -- `TestType_Behavior` pattern):
- `TestBar_FormatContainsRequiredFields` (`bar_test.go` line 49)
- `TestBar_StopIdempotent` (`bar_test.go` line 175)
- `TestCmdNew_Success` (`cmd_cli_test.go` line 40)

For TUI tests:
- `TestUpdate_SessionsMsg`
- `TestUpdate_KeyQuit`
- `TestUpdate_KeyHelp`
- `TestView_ContainsSessionName`
- `TestView_EmptyState`
- `TestView_TerminalTooSmall`
- `TestHelpOverlay_ContainsGroups`

---

### `internal/daemon/types.go` (additive modification)

**Analog:** `internal/daemon/types.go` (self)

**Existing SessionInfo struct** (`types.go` lines 4-13):
```go
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"`
}
```

Add `Status` field after `State`:
```go
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    Status      string `json:"status"` // heuristic status: running/idle/waiting/errored
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"`
}
```

---

### `internal/daemon/engine.go` (additive modification)

**Analog:** `internal/daemon/engine.go` (self)

**Existing ListSessions with status read pattern** (`engine.go` lines 134-167):
```go
func (e *SessionEngine) ListSessions() []SessionInfo {
    sessions := e.registry.List()
    result := make([]SessionInfo, 0, len(sessions))

    e.mu.RLock()
    defer e.mu.RUnlock()

    for _, s := range sessions {
        state := "running"
        if s.State == pty.StateStopped {
            state = "stopped"
        }
        name := e.tabNames[s.ID]

        viewerCount := 0
        if hub, ok := e.manager.Get(s.ID); ok {
            viewerCount = hub.SubscriberCount()
        }

        result = append(result, SessionInfo{
            ID:          s.ID,
            CLI:         s.CLI,
            Name:        name,
            State:       state,
            CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
            Hostname:    e.hostname,
            ViewerCount: viewerCount,
        })
    }
    return result
}
```

**Existing statusMu read pattern** (`engine.go` lines 202-209):
```go
func (e *SessionEngine) GetSessionStatus(sessionID string) string {
    e.statusMu.RLock()
    s, ok := e.sessionStatuses[sessionID]
    e.statusMu.RUnlock()
    if !ok {
        return string(status.StatusRunning)
    }
    return string(s)
}
```

Add heuristic status lookup inside `ListSessions()` for-loop, before `result = append(...)`. Note: `e.mu.RLock()` is already held; `e.statusMu.RLock()` is a separate lock -- safe to acquire (no ordering conflict since `GetSessionStatus` acquires only `statusMu`):
```go
        // Heuristic status from detector (running/idle/waiting/errored).
        heuristicStatus := string(status.StatusRunning) // conservative default
        e.statusMu.RLock()
        if hs, ok := e.sessionStatuses[s.ID]; ok {
            heuristicStatus = string(hs)
        }
        e.statusMu.RUnlock()

        result = append(result, SessionInfo{
            // ... existing fields ...
            Status:      heuristicStatus,
        })
```

---

## Shared Patterns

### DaemonClient API Consumption
**Source:** `internal/daemon/client.go` lines 45-54 (`ListSessions`), lines 137-142 (`GetWebServerStatus`)
**Apply to:** `internal/tui/cmds.go` -- all `tea.Cmd` functions wrap these calls

```go
func (c *DaemonClient) ListSessions() ([]SessionInfo, error) {
    var sessions []SessionInfo
    if err := c.doJSON(http.MethodGet, "/sessions", nil, &sessions); err != nil {
        return nil, err
    }
    if sessions == nil {
        sessions = []SessionInfo{}
    }
    return sessions, nil
}
```

### TTY Detection Pre-Check
**Source:** `cmd_attach.go` lines 77-79
**Apply to:** `cmd_tui.go`

```go
if !term.IsTerminal(int(os.Stdin.Fd())) {
    return fmt.Errorf("attach: stdin is not a terminal")
}
```

For TUI: check `os.Stdout.Fd()` instead.

### CLI Dispatch Pattern
**Source:** `main.go` lines 168-197
**Apply to:** `main.go` (add `case "tui":`)

### External Test Package with safeBuf
**Source:** `internal/statusbar/bar_test.go` lines 1-29
**Apply to:** Integration tests that need to capture output

```go
type safeBuf struct {
    mu  sync.Mutex
    buf bytes.Buffer
}
func (s *safeBuf) Write(p []byte) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.buf.Write(p)
}
func (s *safeBuf) String() string {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.buf.String()
}
```

### Test Setup with Real Daemon
**Source:** `cmd_cli_test.go` lines 19-36
**Apply to:** `cmd_tui.go` integration tests (if needed)

```go
func testSetup(t *testing.T) *daemon.DaemonClient {
    t.Helper()
    engine := daemon.NewSessionEngine()
    api := daemon.NewAPI(engine)
    socketPath := fmt.Sprintf("/tmp/aht%d_%d.sock", os.Getpid(), time.Now().UnixNano()%10000)
    t.Cleanup(func() { os.Remove(socketPath) })
    if err := api.Start(socketPath); err != nil {
        t.Fatalf("api.Start: %v", err)
    }
    t.Cleanup(func() { api.Stop() })
    time.Sleep(10 * time.Millisecond)
    return daemon.NewDaemonClient(socketPath)
}
```

### Mutex-Protected State (Lock-Copy-Unlock-Use)
**Source:** `internal/statusbar/bar.go` lines 149-164 (`draw` method)
**Apply to:** Not directly needed for TUI (Bubble Tea's single-threaded Update/View eliminates mutex need), but relevant if any goroutine-based pattern is added.

```go
b.mu.Lock()
viewerCount := b.viewerCount
connState := b.connState
localCols := b.cols
localRows := b.rows
b.mu.Unlock()
```

### Error Handling in CLI Commands
**Source:** `cmd_cli.go` lines 58-69 (`cmdNew`)
**Apply to:** `cmd_tui.go`

```go
func cmdNew(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error {
    if len(args) < 2 {
        return fmt.Errorf("usage: agenthub new <agent> <path>")
    }
    // ... operation ...
    if err != nil {
        return fmt.Errorf("agenthub new: %w", err)
    }
    return nil
}
```

Pattern: validate args, perform operation, wrap errors with command prefix.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/tui/keys.go` | config | event-driven | No Charm `key.Binding` patterns exist in codebase yet. Use RESEARCH.md Pattern 6 (verified v2 API from Context7). |
| `internal/tui/styles.go` (partial) | config | transform | No Lip Gloss `LightDark` adaptive color patterns exist in codebase. `statusbar/bar.go` uses raw ANSI, not Lip Gloss. Use RESEARCH.md Pattern 2. |

---

## Import Path Reference

All new files in `internal/tui/` must use these exact import paths:

| Import | Package | Purpose |
|--------|---------|---------|
| `tea "charm.land/bubbletea/v2"` | Bubble Tea v2 | Model-View-Update framework |
| `"charm.land/lipgloss/v2"` | Lip Gloss v2 | Declarative terminal styling |
| `"charm.land/bubbles/v2/key"` | Bubbles v2 key | Key binding definitions + matching |
| `"github.com/scottkw/agenthub/internal/daemon"` | Daemon client + types | `DaemonClient`, `SessionInfo`, `WebServerStatusResponse` |
| `"github.com/scottkw/agenthub/internal/status"` | Status constants | `SessionStatus` type (for reference only -- TUI maps string values from JSON) |
| `"golang.org/x/term"` | Term | `IsTerminal()` for TTY pre-check in `cmd_tui.go` |

**go.mod additions** (run before any Go code compiles):
```bash
cd /Users/ken/dev/agenthub && go get charm.land/bubbletea/v2@v2.0.5 charm.land/lipgloss/v2@v2.0.3 charm.land/bubbles/v2@v2.1.0
```

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub` -- all `.go` files
**Files scanned:** 97 Go source files across 9 packages
**Key files read:** `main.go`, `cmd_attach.go`, `cmd_cli.go`, `cmd_cli_test.go`, `dispatch_test.go`, `internal/statusbar/bar.go`, `internal/statusbar/bar_test.go`, `internal/daemon/types.go`, `internal/daemon/engine.go`, `internal/daemon/engine_test.go`, `internal/daemon/client.go`, `internal/daemon/api.go`, `internal/status/detector.go`
**Pattern extraction date:** 2026-04-15
