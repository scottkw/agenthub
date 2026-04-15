# Phase 76: TUI Foundation - Research

**Researched:** 2026-04-15
**Domain:** Bubble Tea v2 terminal UI framework, Lip Gloss v2 styling, Bubbles v2 components, Go CLI integration
**Confidence:** HIGH

## Summary

Phase 76 adds the `agenthub tui` subcommand that launches a full-screen Bubble Tea v2 terminal UI. The TUI displays a scrollable session list with status indicators, a footer showing web server status, and a help overlay modal. The entire UI contract is already defined in the approved `76-UI-SPEC.md` -- this research focuses on the technical implementation patterns needed to execute that contract.

The Charm ecosystem (Bubble Tea v2, Lip Gloss v2, Bubbles v2) has stabilized with GA releases as of early 2025. Bubble Tea v2 introduces breaking changes from v1: the import path moves to `charm.land/bubbletea/v2`, `View()` returns `tea.View` instead of `string`, terminal features (alt-screen, mouse mode) are declarative fields on `tea.View` rather than program options, and key events use `tea.KeyPressMsg` instead of `tea.KeyMsg`. These v2 patterns are well-documented and all code examples in this research reflect the v2 API.

The primary technical challenge is the data source: the existing `GET /sessions` endpoint returns PTY process state ("running"/"stopped") but not the heuristic status from `internal/status` (running/idle/waiting/errored). The UI-SPEC requires 4 distinct status indicators. The cleanest solution is to enrich `SessionInfo.State` with the heuristic status in `ListSessions()` -- a one-line change in `engine.go` that reads from the already-populated `sessionStatuses` map. This avoids N+1 API calls per refresh tick.

**Primary recommendation:** Build `internal/tui/` package with Bubble Tea v2 Model-View-Update pattern, enrich the daemon session list API with heuristic status, use `lipgloss.LightDark` for adaptive colors via `tea.BackgroundColorMsg`, and implement the help overlay as a custom centered modal (not `bubbles/help` which renders inline, not as a bordered overlay).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUI-01 | `agenthub tui` command launches a Bubble Tea terminal UI | CLI wiring pattern documented (Section: CLI Wiring); `tea.NewProgram` + `tea.View.AltScreen = true`; TTY pre-check pattern from Phase 75 |
| TUI-02 | Session list panel shows all sessions with status indicators, agent type, hostname, and viewer count | Data source: `DaemonClient.ListSessions()` returns `[]SessionInfo` with all fields; heuristic status enrichment documented (Section: Data Source - Session Status) |
| TUI-08 | Web server status displayed in footer/status area | Data source: `DaemonClient.GetWebServerStatus()` returns `WebServerStatusResponse{Running, URL}`; polling via `tea.Tick` every 2s |
| TUI-09 | Help overlay shows all keybindings for current view | Custom overlay implementation (not `bubbles/help`); centered bordered modal using `lipgloss.Place` + `lipgloss.RoundedBorder()`; dismiss on `?` or `Esc` |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TUI rendering (View) | `internal/tui` | -- | Bubble Tea Model-View-Update owns all rendering |
| Session data fetching | `internal/daemon.DaemonClient` | `internal/tui` (tea.Cmd) | DaemonClient provides typed HTTP client; TUI wraps calls in tea.Cmd goroutines |
| Heuristic status detection | `internal/status.Detector` | `internal/daemon.SessionEngine` | Detector classifies PTY output; Engine stores results; API serves them |
| Web server status | `internal/daemon.DaemonClient` | `internal/tui` (tea.Cmd) | Same pattern as session data |
| Adaptive color theming | `internal/tui/styles.go` | Lip Gloss v2 | styles.go defines 13 semantic tokens using `lipgloss.LightDark` |
| Keybinding management | `internal/tui/keys.go` | Bubbles v2 `key` package | `key.NewBinding` for each action; `key.Matches` in Update |
| Help overlay rendering | `internal/tui/help.go` | Lip Gloss v2 | Custom modal, not `bubbles/help` (which renders inline) |
| CLI command dispatch | `main.go` (runCLI) | `internal/tui` | `case "tui":` added to switch in main.go |
| TTY detection | `cmd_tui.go` | `golang.org/x/term` | Pre-check before starting tea.Program |
| Status indicator glyphs | `internal/tui/view.go` | `internal/status` | Maps `SessionStatus` values to Unicode glyphs + color tokens |

## Standard Stack

### Core (NEW dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.5 | Model-View-Update TUI framework | [VERIFIED: `go list -m -versions`] GA release, Charm's flagship |
| `charm.land/lipgloss/v2` | v2.0.3 | Declarative terminal styling (colors, borders, padding) | [VERIFIED: `go list -m -versions`] Charm's styling companion |
| `charm.land/bubbles/v2` | v2.1.0 | Pre-built components: `key`, `help` (keybinding infrastructure) | [VERIFIED: `go list -m -versions`] Standard Charm component library |

### Supporting (already in go.mod)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/term` | v0.41.0 | `IsTerminal(fd)` for TTY pre-check | [VERIFIED: go.mod] Pre-check before launching tea.Program |
| `internal/daemon` | -- | `DaemonClient` for session/webserver API calls | [VERIFIED: codebase] Existing typed HTTP client over Unix socket |
| `internal/status` | -- | `SessionStatus` type constants (running/idle/waiting/errored) | [VERIFIED: codebase] Status indicator mapping |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `bubbles/help` for help overlay | Custom Lip Gloss overlay | `bubbles/help` renders inline (horizontal/vertical lists), not as a centered bordered modal. UI-SPEC requires a centered bordered overlay. Custom is required. |
| `bubbles/list` for session list | Custom rendered table | `bubbles/list` has its own filtering/searching UI that conflicts with our keybindings and column layout. A custom rendered list gives exact control over column widths and truncation. Recommended: custom. |
| `bubbles/viewport` for scrolling | Manual scroll math | With a simple list (not free-form text), `selectedIdx` + visible window offset is simpler than a viewport. Viewport adds unnecessary complexity for a row-based list. |
| Per-session status API calls | Enrich `ListSessions` response | N+1 API calls per tick is wasteful. Engine already has `sessionStatuses` map -- enrich in `ListSessions()` with one line. |

**Installation:**
```bash
cd /Users/ken/dev/agenthub && go get charm.land/bubbletea/v2@v2.0.5 charm.land/lipgloss/v2@v2.0.3 charm.land/bubbles/v2@v2.1.0
```

**Version verification:**
- `charm.land/bubbletea/v2` latest: v2.0.5 [VERIFIED: `go list -m -versions` 2026-04-15]
- `charm.land/lipgloss/v2` latest: v2.0.3 [VERIFIED: `go list -m -versions` 2026-04-15]
- `charm.land/bubbles/v2` latest: v2.1.0 [VERIFIED: `go list -m -versions` 2026-04-15]

## Architecture Patterns

### System Architecture Diagram

```
cmd_tui.go                    internal/tui/               internal/daemon/
───────────                   ───────────────             ────────────────

agenthub tui
  │
  ├─ term.IsTerminal(stdout) ─── false ──► print error + exit(1)
  │
  ├─ daemon.NewDaemonClient(socketPath)
  │
  ├─ daemon.EnsureDaemon(socketPath) ─── fail ──► print error + exit(1)
  │
  └─ tui.Run(client) ────────────────────────────────────────────────────
       │
       ├─ tea.NewProgram(newModel(client))
       │    │
       │    ├─ Init() ──► tea.Batch(fetchSessions, fetchWebStatus)
       │    │                │
       │    │                ├─ fetchSessions ──► client.ListSessions()
       │    │                │                      return sessionsMsg{sessions, err}
       │    │                │
       │    │                └─ fetchWebStatus ──► client.GetWebServerStatus()
       │    │                                      return webStatusMsg{status, err}
       │    │
       │    ├─ Update(msg) ◄───────────────────────────────────────────
       │    │    │
       │    │    ├─ tea.WindowSizeMsg ──► m.width, m.height = msg.Width, msg.Height
       │    │    │
       │    │    ├─ tea.BackgroundColorMsg ──► m.hasDark = msg.IsDark()
       │    │    │                              m.styles = newStyles(m.hasDark)
       │    │    │
       │    │    ├─ tea.KeyPressMsg ──► dispatch: q/Ctrl+C quit, ?/Esc help,
       │    │    │                       j/k/Up/Down navigate, r refresh, g/G jump,
       │    │    │                       Enter/n reserved toast
       │    │    │
       │    │    ├─ sessionsMsg ──► m.sessions = msg.sessions, m.loading = false
       │    │    │
       │    │    ├─ webStatusMsg ──► m.webStatus = msg.status
       │    │    │
       │    │    ├─ tickMsg ──► tea.Batch(fetchSessions, fetchWebStatus, nextTick)
       │    │    │
       │    │    └─ errMsg ──► m.err = msg.err
       │    │
       │    └─ View() ──► tea.View{
       │                    AltScreen: true,
       │                    Content: renderHeader + renderColHeaders +
       │                             renderSessionList + renderSeparator +
       │                             renderFooter [+ renderHelpOverlay if showHelp]
       │                  }
       │
       └─ p.Run() ──► blocks until quit ──► return
```

### Recommended Project Structure

```
internal/tui/
├── tui.go           # Run() entry point, newModel(), Init()
├── model.go         # Model struct, message types (sessionsMsg, webStatusMsg, tickMsg, errMsg)
├── update.go        # Update() method — all message dispatch
├── view.go          # View() method — header, column headers, session list, separator, footer
├── help.go          # Help overlay rendering (custom bordered modal)
├── styles.go        # Semantic color tokens, adaptive styles via LightDark
├── keys.go          # Key bindings using bubbles/key
├── cmds.go          # tea.Cmd functions: fetchSessions, fetchWebStatus, nextTick
├── tui_test.go      # Unit tests for Update() state transitions, View() rendering
cmd_tui.go           # cmdTUI() function: TTY check, create client, call tui.Run()
```

### Pattern 1: Bubble Tea v2 Model-View-Update

**What:** The core application pattern for Bubble Tea v2. All state lives in a Model struct. `Update()` receives messages and returns a new model + optional commands. `View()` returns a `tea.View` struct (not a string).

**When to use:** Every Bubble Tea v2 app.

**Example:**
```go
// Source: charm.land/bubbletea/v2 UPGRADE_GUIDE_V2.md [VERIFIED: Context7]
import tea "charm.land/bubbletea/v2"

type Model struct {
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
    client    *daemon.DaemonClient
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        tea.RequestBackgroundColor,  // detect light/dark
        fetchSessions(m.client),
        fetchWebStatus(m.client),
        nextTick(),
    )
}

func (m Model) View() tea.View {
    var v tea.View
    v.AltScreen = true
    // NO mouse mode (per REQUIREMENTS.md)
    if m.width < 60 || m.height < 10 {
        v.SetContent(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
            "Terminal too small (need 60x10)"))
        return v
    }
    v.SetContent(m.renderFull())
    return v
}
```

### Pattern 2: Adaptive Colors via BackgroundColorMsg

**What:** Bubble Tea v2 integrates with Lip Gloss v2 for adaptive light/dark color detection. Request the background color in `Init()`, receive `tea.BackgroundColorMsg` in `Update()`, then build styles using `lipgloss.LightDark()`.

**When to use:** Any TUI that must work on both light and dark terminal themes.

**Example:**
```go
// Source: Lip Gloss v2 README.md [VERIFIED: Context7]
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.BackgroundColorMsg:
        m.hasDark = msg.IsDark()
        m.styles = newStyles(m.hasDark)
        return m, nil
    }
    // ...
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

### Pattern 3: tea.Tick Polling

**What:** Periodic data refresh using `tea.Tick`. Returns a `tickMsg` that triggers re-fetch of sessions and web status.

**When to use:** Any polled data source with a fixed interval.

**Example:**
```go
// Source: charm.land/bubbletea/v2 docs [VERIFIED: Context7]
type tickMsg time.Time

func nextTick() tea.Cmd {
    return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case tickMsg:
        return m, tea.Batch(
            fetchSessions(m.client),
            fetchWebStatus(m.client),
            nextTick(),
        )
    }
    // ...
}
```

### Pattern 4: Async tea.Cmd for Data Fetching

**What:** Wrap daemon API calls in `tea.Cmd` functions that run in goroutines and return result messages. Never block `Update()`.

**Example:**
```go
// Source: Bubble Tea v2 pattern [VERIFIED: Context7]
type sessionsMsg struct {
    sessions []daemon.SessionInfo
    err      error
}

type webStatusMsg struct {
    status daemon.WebServerStatusResponse
    err    error
}

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
```

### Pattern 5: Custom Help Overlay (Centered Bordered Modal)

**What:** The UI-SPEC requires a centered bordered modal for the help overlay, not the inline help view that `bubbles/help` provides. Render the overlay content with `lipgloss.Place()` to center it over the existing view.

**Example:**
```go
// Source: Lip Gloss v2 Place API [VERIFIED: Context7]
func (m Model) renderHelpOverlay() string {
    // Build help content with keybinding groups
    content := m.buildHelpContent()

    // Apply border
    overlayWidth := max(40, min(60, m.width-10))
    bordered := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(m.styles.BorderNormal).
        BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true).
        Width(overlayWidth - 2). // subtract border width
        Render(content)

    // Center the overlay over the full screen
    return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, bordered)
}
```

### Pattern 6: Key Bindings with bubbles/key

**What:** Define all keybindings using `key.NewBinding` with `key.WithKeys` and `key.WithHelp`. Match in `Update()` with `key.Matches()`.

**Example:**
```go
// Source: charm.land/bubbles/v2/key [VERIFIED: Context7]
import "charm.land/bubbles/v2/key"

type KeyMap struct {
    Quit    key.Binding
    Help    key.Binding
    Up      key.Binding
    Down    key.Binding
    Refresh key.Binding
    Top     key.Binding
    Bottom  key.Binding
    Attach  key.Binding  // reserved
    New     key.Binding  // reserved
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
            key.WithHelp("↑/k", "move up"),
        ),
        Down: key.NewBinding(
            key.WithKeys("down", "j"),
            key.WithHelp("↓/j", "move down"),
        ),
        Refresh: key.NewBinding(
            key.WithKeys("r"),
            key.WithHelp("r", "refresh list"),
        ),
        Top: key.NewBinding(
            key.WithKeys("g", "home"),
            key.WithHelp("g/Home", "jump to first"),
        ),
        Bottom: key.NewBinding(
            key.WithKeys("G", "end"),
            key.WithHelp("G/End", "jump to last"),
        ),
        Attach: key.NewBinding(
            key.WithKeys("enter"),
            key.WithHelp("Enter", "attach to session"),
        ),
        New: key.NewBinding(
            key.WithKeys("n"),
            key.WithHelp("n", "new session"),
        ),
    }
}
```

### Anti-Patterns to Avoid

- **Enable mouse mode:** `tea.View.MouseMode` must remain unset. REQUIREMENTS.md explicitly excludes mouse-driven TUI navigation. [CITED: REQUIREMENTS.md Out of Scope]
- **Block in Update():** All daemon API calls must be `tea.Cmd` functions that return messages. Never call `client.ListSessions()` synchronously in Update(). [CITED: 76-UI-SPEC.md Anti-Patterns]
- **Hard-code terminal dimensions:** Always derive layout from `model.width` / `model.height` updated via `tea.WindowSizeMsg`. [CITED: 76-UI-SPEC.md Anti-Patterns]
- **Use fmt.Print/Println for rendering:** All output goes through `tea.View`'s content. Never write directly to stdout. [CITED: 76-UI-SPEC.md Anti-Patterns]
- **Poll faster than 2 seconds:** 2-second tick is the contract. [CITED: 76-UI-SPEC.md]
- **Use `bubbles/help` for the overlay:** It renders inline, not as a centered bordered modal. Build custom. [VERIFIED: Context7 -- bubbles/help renders as horizontal/vertical inline help bar]
- **Use `bubbles/list` for session list:** Its built-in filtering/pagination/search UI conflicts with our column layout and keybindings. Build custom. [ASSUMED]
- **Use `tea.WithAltScreen()` program option:** This is v1. In v2, set `v.AltScreen = true` in `View()`. [VERIFIED: Context7 UPGRADE_GUIDE_V2.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal styling | Raw ANSI escape sequences | Lip Gloss v2 `lipgloss.NewStyle()` | Lip Gloss handles color profiles, adaptive colors, borders, alignment. Phase 75 used raw ANSI for statusbar (different surface); TUI uses Lip Gloss exclusively. |
| Key binding matching | Manual `switch msg.String()` | `bubbles/key` + `key.Matches()` | Key bindings integrate with help text, can be enabled/disabled, and avoid string typos. |
| Periodic timer | `time.Ticker` + goroutine | `tea.Tick(2*time.Second, ...)` | `tea.Tick` integrates with the Bubble Tea message loop. Manual timers would need channels and risk race conditions. |
| Light/dark detection | `lipgloss.HasDarkBackground()` manually | `tea.RequestBackgroundColor` + `tea.BackgroundColorMsg` | Bubble Tea v2 integrates this detection into the message loop. Manual detection races with program startup. |
| Text centering in terminal | Manual padding math | `lipgloss.Place(w, h, hAlign, vAlign, content)` | Place handles all the whitespace padding and alignment math. |
| Box drawing borders | Manual Unicode box chars | `lipgloss.RoundedBorder()` + `.Border()` style | Lip Gloss provides predefined border sets and handles corners correctly. |

## Common Pitfalls

### Pitfall 1: v1/v2 API Confusion

**What goes wrong:** Using v1 patterns (e.g., `tea.WithAltScreen()`, `View() string`, `tea.KeyMsg`) that don't compile with v2.
**Why it happens:** Most online examples and training data are v1. The import path changed from `github.com/charmbracelet/bubbletea` to `charm.land/bubbletea/v2`.
**How to avoid:** Always use `charm.land/bubbletea/v2` imports. `View()` returns `tea.View`, not `string`. Key events are `tea.KeyPressMsg`, not `tea.KeyMsg`. Alt-screen is `v.AltScreen = true` in `View()`, not a program option.
**Warning signs:** Import path contains `github.com/charmbracelet/bubbletea` (v1). `View()` returns `string`.

### Pitfall 2: Blocking Update() with HTTP Calls

**What goes wrong:** Calling `client.ListSessions()` directly in `Update()` freezes the UI for the duration of the HTTP round-trip.
**Why it happens:** Natural instinct is to fetch data where you need it. Bubble Tea's single-threaded message loop makes this a hard freeze.
**How to avoid:** All I/O goes in `tea.Cmd` functions. `Update()` only dispatches commands and processes results.
**Warning signs:** UI hangs on tick. Daemon not running causes permanent freeze instead of error message.

### Pitfall 3: Missing Initial WindowSizeMsg

**What goes wrong:** Rendering with zero width/height on first frame causes empty or panicked output.
**Why it happens:** `tea.WindowSizeMsg` is delivered asynchronously after program start. The first `View()` call may happen before it arrives.
**How to avoid:** Initialize `width` and `height` to 0 in the model. In `View()`, if `width == 0 || height == 0`, return a minimal view (empty or "Loading..."). Alternatively, the `loading` flag serves this purpose.
**Warning signs:** Panic on first render. Division by zero in column width calculations.

### Pitfall 4: Session Status Data Gap

**What goes wrong:** `SessionInfo.State` returns "running"/"stopped" (PTY process state), but the UI-SPEC requires 4 heuristic states (running/idle/waiting/errored) from `internal/status`.
**Why it happens:** The `ListSessions()` engine method populates `State` from `pty.StateStopped`, not from the heuristic detector.
**How to avoid:** Enrich `SessionInfo` with a `Status` field populated from `engine.sessionStatuses[id]` in `ListSessions()`. This is a one-line addition to `engine.go`. Alternatively, map "stopped" to "errored" in the TUI, but that loses the nuance of waiting/idle.
**Warning signs:** All sessions show as "running" regardless of actual state.

### Pitfall 5: Color Profile Mismatch

**What goes wrong:** Hex colors display as wrong colors on terminals with limited color support (ANSI 256 only, or ANSI 16 only).
**Why it happens:** Not all terminals support TrueColor. The UI-SPEC specifies both hex and ANSI 256 values.
**How to avoid:** Lip Gloss v2 automatically downgrades colors to the terminal's detected color profile. Using `lipgloss.Color("#5F87FF")` works on TrueColor terminals and Lip Gloss will approximate on 256-color terminals. No manual `compat.CompleteColor` needed for our use case since all our hex values are exact ANSI 256 matches.
**Warning signs:** Colors look wrong on certain terminals (e.g., macOS Terminal.app in 256-color mode).

### Pitfall 6: Help Overlay Z-Order

**What goes wrong:** Help overlay renders behind or adjacent to the session list instead of on top.
**Why it happens:** Bubble Tea renders a single string per frame. There's no z-index. "On top" means the overlay string replaces the underlying content at those character positions.
**How to avoid:** When `showHelp` is true, render the full background (header + list + footer), then use `lipgloss.Place()` to center the overlay on top. The Place function fills the surrounding area with spaces, effectively covering the background.
**Warning signs:** Help overlay appears at the bottom of the screen or next to the list.

### Pitfall 7: Lip Gloss v2 AdaptiveColor Migration

**What goes wrong:** Using `lipgloss.AdaptiveColor{Light: "#xxx", Dark: "#yyy"}` (v1 syntax) fails to compile.
**Why it happens:** Lip Gloss v2 removed the `AdaptiveColor` type. The replacement is `lipgloss.LightDark(hasDarkBG)(lightColor, darkColor)`.
**How to avoid:** Use the `lipgloss.LightDark` helper with `tea.BackgroundColorMsg.IsDark()` result.
**Warning signs:** Import of `lipgloss.AdaptiveColor` doesn't compile.

## Code Examples

### Complete Model Initialization

```go
// Source: Pattern synthesis from Bubble Tea v2 docs [VERIFIED: Context7]
package tui

import (
    "time"

    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
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
    toast     string       // reserved-key feedback
    toastExp  time.Time    // toast expiry
}

func newModel(client *daemon.DaemonClient) Model {
    return Model{
        client:  client,
        loading: true,
        keys:    defaultKeyMap(),
        styles:  newStyles(true), // assume dark until BackgroundColorMsg
    }
}

func Run(client *daemon.DaemonClient) error {
    p := tea.NewProgram(newModel(client))
    _, err := p.Run()
    return err
}
```

### Complete tea.Cmd Data Fetching

```go
// Source: Bubble Tea v2 Cmd pattern [VERIFIED: Context7]
type sessionsMsg struct {
    sessions []daemon.SessionInfo
    err      error
}

type webStatusMsg struct {
    status daemon.WebServerStatusResponse
    err    error
}

type tickMsg time.Time

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

### Session Row Rendering with Status Glyph

```go
// Source: UI-SPEC status indicators table [CITED: 76-UI-SPEC.md]
func (m Model) renderSessionRow(s daemon.SessionInfo, idx int) string {
    isSelected := idx == m.selected

    // Cursor column (2 chars)
    cursor := "  "
    if isSelected {
        cursor = "> "
    }

    // Status glyph (2 chars)
    glyph, glyphColor := statusGlyph(s.Status, m.styles)
    styledGlyph := lipgloss.NewStyle().Foreground(glyphColor).Render(glyph) + " "

    // Column widths per UI-SPEC
    nameWidth := m.width - 53 // 53 = cursor(2) + status(2) + agent(12) + host(20) + viewers(7) + gaps(5*2)
    if nameWidth < 8 {
        nameWidth = 8
    }

    name := truncate(s.Name, nameWidth)
    agent := truncate(s.CLI, 12)
    host := truncate(s.Hostname, 20)
    viewers := ""
    if s.ViewerCount > 0 {
        viewers = fmt.Sprintf("%d", s.ViewerCount)
    }

    row := fmt.Sprintf("%s%s%-*s  %-12s  %-20s  %7s",
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

func statusGlyph(status string, s Styles) (string, lipgloss.TerminalColor) {
    switch status {
    case "idle":
        return "○", s.StatusIdle
    case "waiting":
        return "●", s.StatusWaiting
    case "errored":
        return "✖", s.StatusErrored
    default: // "running" or unrecognized
        return "●", s.StatusRunning
    }
}
```

## Data Source Details

### Data Source: Session List

**Endpoint:** `GET /sessions` via `DaemonClient.ListSessions()`
**Response type:** `[]daemon.SessionInfo`
**Fields used by TUI:**
- `ID` -- for future attach/kill operations (Phase 77)
- `Name` -- session name column
- `CLI` -- agent type column
- `Hostname` -- host column
- `ViewerCount` -- viewers column (MC-04, already populated)
- `State` -- currently "running"/"stopped" (PTY state), NOT the heuristic status
- `Status` -- **MUST BE ADDED**: heuristic status from `internal/status` (running/idle/waiting/errored)

**Required daemon change:** In `internal/daemon/engine.go` `ListSessions()`, add:
```go
// After existing state assignment (line 142-145):
heuristicStatus := "running" // conservative default
e.statusMu.RLock()
if hs, ok := e.sessionStatuses[s.ID]; ok {
    heuristicStatus = string(hs)
}
e.statusMu.RUnlock()
// Include in SessionInfo:
result = append(result, SessionInfo{
    // ... existing fields ...
    Status: heuristicStatus,  // NEW field
})
```

And in `internal/daemon/types.go`, add `Status string \`json:"status"\`` field to `SessionInfo`.

**Rationale:** This avoids N+1 API calls per tick. The engine already maintains `sessionStatuses` -- exposing it through the list endpoint is the natural location. [VERIFIED: engine.go lines 34, 122-124, 200-209]

### Data Source: Web Server Status

**Endpoint:** `GET /webserver/status` via `DaemonClient.GetWebServerStatus()`
**Response type:** `daemon.WebServerStatusResponse`
**Fields used by TUI:**
- `Running` (bool) -- determines glyph and color
- `URL` (string) -- displayed when running

[VERIFIED: internal/daemon/types.go lines 72-78, client.go lines 137-142]

### Data Source: Daemon Health

**Endpoint:** `GET /health` via `DaemonClient.Health()`
**Used:** At TUI launch to validate daemon is running before starting the Bubble Tea program.
**Error handling:** If health check fails, display "Cannot connect to daemon. Is it running?" per UI-SPEC copywriting contract.

[VERIFIED: internal/daemon/client.go lines 39-42]

## CLI Wiring

### Adding the `tui` Subcommand

The TUI command hooks into the existing CLI dispatch in `main.go` `runCLI()`. The pattern follows existing commands exactly.

**File: `cmd_tui.go`** (new file, same pattern as `cmd_attach.go`):
```go
package main

import (
    "fmt"
    "os"

    "github.com/scottkw/agenthub/internal/daemon"
    "github.com/scottkw/agenthub/internal/tui"
    "golang.org/x/term"
)

func cmdTUI(client *daemon.DaemonClient) error {
    // TTY pre-check (UI-SPEC TTY Detection)
    if !term.IsTerminal(int(os.Stdout.Fd())) {
        return fmt.Errorf("agenthub tui requires a terminal. Redirect to a TTY or use 'agenthub list' instead")
    }

    // Validate daemon is reachable
    if err := client.Health(); err != nil {
        return fmt.Errorf("cannot connect to daemon: %w", err)
    }

    return tui.Run(client)
}
```

**File: `main.go`** (add to switch in `runCLI`):
```go
case "tui":
    err = cmdTUI(client)
```

**File: `cmd_cli.go`** (add to usage string):
```
  tui                                       Launch interactive terminal UI
```

[VERIFIED: main.go dispatch pattern lines 168-196; cmd_cli.go usage() lines 23-53]

## TTY Detection

Bubble Tea v2 handles TTY detection internally -- it opens `/dev/tty` for input automatically (the `tea.WithInputTTY()` option was removed in v2). However, the UI-SPEC explicitly requires a pre-check that prints an error and exits if stdout is not a TTY:

```
Error: agenthub tui requires a terminal. Redirect to a TTY or use 'agenthub list' instead.
```

This pre-check uses `term.IsTerminal(int(os.Stdout.Fd()))` -- the same pattern Phase 75 uses for status bar suppression.

[VERIFIED: golang.org/x/term already in go.mod; Pattern used in cmd_attach.go]

## Testing Strategy

### Unit Tests for Update() State Transitions

Test the model's `Update()` method directly by sending messages and asserting the resulting model state. This is the primary testing approach.

```go
func TestUpdate_SessionsMsg(t *testing.T) {
    m := newModel(nil)
    m.loading = true

    sessions := []daemon.SessionInfo{
        {ID: "1", Name: "test", CLI: "claude"},
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

### Unit Tests for View() Rendering

Test the model's `View()` method by setting model state and asserting the rendered content contains expected strings.

```go
func TestView_ContainsSessionName(t *testing.T) {
    m := newModel(nil)
    m.width = 120
    m.height = 24
    m.hasDark = true
    m.styles = newStyles(true)
    m.sessions = []daemon.SessionInfo{
        {ID: "1", Name: "my-project", CLI: "claude", Hostname: "macbook"},
    }
    m.loading = false

    view := m.View()
    content := view.Content() // or however tea.View exposes content
    if !strings.Contains(content, "my-project") {
        t.Errorf("view should contain session name")
    }
}
```

### Golden Frame Snapshot Tests (teatest)

For more comprehensive rendering tests, use `teatest` from `github.com/charmbracelet/x/exp/teatest`. This captures the full rendered output and compares against golden files.

**Note:** The `teatest` package is experimental and may have breaking changes. For Phase 76, prioritize direct Update()/View() unit tests. Golden frame tests are a nice-to-have.

```go
// Source: charm.land/blog/teatest/ [CITED: https://charm.land/blog/teatest/]
import "github.com/charmbracelet/x/exp/teatest"

func TestGolden_InitialView(t *testing.T) {
    m := newModel(nil)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 24))
    // Send initial data
    tm.Send(sessionsMsg{sessions: testSessions})
    tm.Send(tea.KeyPressMsg{Code: 'q'})
    tm.WaitFinished(t)
    out, _ := io.ReadAll(tm.FinalOutput(t))
    teatest.RequireEqualOutput(t, out)
}
// Run with: go test ./internal/tui/... -update  (to create/update golden files)
```

### Manual UAT for Terminal Rendering

Some aspects cannot be tested headlessly:
- Actual color rendering on light vs dark themes
- Status glyph Unicode rendering on different terminals
- Scroll behavior with many sessions
- Terminal resize responsiveness
- Help overlay visual positioning

These require manual testing on at least macOS Terminal.app and iTerm2.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bubble Tea v1: `View() string` | v2: `View() tea.View` | v2.0.0 (2025) | Must use `tea.NewView()` or `v.SetContent()` |
| v1: `tea.WithAltScreen()` option | v2: `v.AltScreen = true` in View() | v2.0.0 | All terminal features are declarative in View |
| v1: `tea.KeyMsg` | v2: `tea.KeyPressMsg` | v2.0.0 | Type assertion in Update() must change |
| v1: `github.com/charmbracelet/bubbletea` | v2: `charm.land/bubbletea/v2` | v2.0.0 | New vanity domain import path |
| v1: `lipgloss.AdaptiveColor{}` | v2: `lipgloss.LightDark(isDark)(light, dark)` | lipgloss v2.0.0 | AdaptiveColor struct removed; use LightDark helper |
| v1: `msg.String() == " "` for space | v2: `msg.String() == "space"` | v2.0.0 | Space bar returns "space" not " " |

**Deprecated/outdated:**
- `tea.WithAltScreen()`: removed in v2 [VERIFIED: Context7 UPGRADE_GUIDE_V2.md]
- `tea.WithInputTTY()`: removed in v2 (always opens TTY automatically) [VERIFIED: Context7]
- `tea.WithANSICompressor()`: removed in v2 (renderer handles optimization automatically) [VERIFIED: Context7]
- `lipgloss.AdaptiveColor`: removed in v2 (use `LightDark` or `compat.AdaptiveColor`) [VERIFIED: Context7]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `bubbles/list` conflicts with our column layout and keybindings | Anti-Patterns | LOW -- if wrong, we could use it but custom is still preferred for exact column control |
| A2 | `tea.View.Content()` method exists to extract string content for testing | Testing Strategy | MEDIUM -- if View() doesn't expose content as a string, tests need to use teatest or different approach |
| A3 | Lip Gloss hex values that exactly match ANSI 256 palette will render identically on 256-color terminals | Pitfall 5 | LOW -- Lip Gloss auto-downgrades; UI-SPEC provides both hex and ANSI 256 values as fallback |
| A4 | Adding `Status` field to `SessionInfo` won't break existing API consumers (GUI, CLI list) | Data Source | LOW -- new field, not a rename; JSON clients ignore unknown fields |

## Open Questions

1. **Session Status Enrichment Approach**
   - What we know: `ListSessions()` returns PTY state ("running"/"stopped"), but TUI needs heuristic status (running/idle/waiting/errored). Engine already has `sessionStatuses` map.
   - What's unclear: Should we add a new `Status` field to `SessionInfo` (preferred), or have the TUI make per-session status calls?
   - Recommendation: Add `Status string` field to `SessionInfo` and populate in `ListSessions()`. This is the cleanest approach and benefits future API consumers too.

2. **tea.View Content Access for Testing**
   - What we know: `tea.View` is returned from `View()` in v2. We need to extract the content string for unit tests.
   - What's unclear: The exact API to get the string content from a `tea.View` struct.
   - Recommendation: `tea.NewView(content)` suggests `tea.View` stores content internally. Check v2 source or use `v.String()` if available. Fallback: test `renderFull()` directly (a method that returns string, called by `View()`).

3. **teatest v2 Stability**
   - What we know: `github.com/charmbracelet/x/exp/teatest` exists but is experimental.
   - What's unclear: Whether a v2-compatible version is released and stable.
   - Recommendation: Use direct Update()/View() unit tests as primary strategy. Add teatest golden tests only if the package is stable. Don't block phase execution on teatest availability.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go runtime | All code | Yes | go1.26.2 | -- |
| Unix socket (daemon) | Data fetching | Yes | -- | -- |
| Terminal (TTY) | TUI rendering | Yes (dev) | -- | Error + exit for non-TTY |
| charm.land/bubbletea/v2 | TUI framework | Yes (go get) | v2.0.5 | -- |
| charm.land/lipgloss/v2 | Styling | Yes (go get) | v2.0.3 | -- |
| charm.land/bubbles/v2 | Key bindings | Yes (go get) | v2.1.0 | -- |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package (go1.26.2) |
| Config file | none -- Go's built-in test runner |
| Quick run command | `go test ./internal/tui/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -timeout 120s` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUI-01 | `agenthub tui` launches Bubble Tea UI | unit (Init returns commands) + manual UAT | `go test ./internal/tui/... -run TestInit -count=1` | Wave 0 |
| TUI-02 | Session list shows name, status, agent, host, viewers | unit (View rendering) | `go test ./internal/tui/... -run TestView_SessionList -count=1` | Wave 0 |
| TUI-08 | Footer shows web server status + URL | unit (View rendering) | `go test ./internal/tui/... -run TestView_Footer -count=1` | Wave 0 |
| TUI-09 | Help overlay on ? key | unit (Update state + View rendering) | `go test ./internal/tui/... -run TestHelpOverlay -count=1` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/tui/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -timeout 120s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/tui/tui_test.go` -- covers TUI-01 (Init returns expected commands)
- [ ] `internal/tui/update_test.go` -- covers TUI-01/02/08/09 (Update state transitions)
- [ ] `internal/tui/view_test.go` -- covers TUI-02/08/09 (View rendering assertions)
- [ ] `internal/tui/help_test.go` -- covers TUI-09 (help overlay content)
- [ ] Framework install: `go get charm.land/bubbletea/v2@v2.0.5 charm.land/lipgloss/v2@v2.0.3 charm.land/bubbles/v2@v2.1.0`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Not applicable -- TUI connects to local daemon via Unix socket (no auth) |
| V3 Session Management | no | Not applicable -- no user sessions in TUI |
| V4 Access Control | no | Unix socket permissions control access |
| V5 Input Validation | yes | Session names/hostnames displayed in TUI come from daemon API. Lip Gloss handles rendering safely (no raw ANSI injection through lipgloss.Render). However, format strings in `fmt.Sprintf` with `%s` are safe. |
| V6 Cryptography | no | Not applicable |

### Known Threat Patterns for Go TUI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Terminal injection via session names | Tampering | Lip Gloss `Render()` escapes content safely. Additionally, `internal/statusbar.sanitize()` pattern can be reused if rendering raw strings outside Lip Gloss. |
| Daemon API unavailability | Denial of Service | Display "Cannot connect to daemon" error state. Don't crash or hang. Error is a tea.Msg, not a panic. |
| Rapid terminal resize | Denial of Service | Bubble Tea handles SIGWINCH and delivers `WindowSizeMsg`. View() is called on next frame. No extra handling needed. |

## Sources

### Primary (HIGH confidence)
- Context7 `/charmbracelet/bubbletea` -- v2 Model/View/Update pattern, UPGRADE_GUIDE_V2.md, import paths, KeyPressMsg, Tick, Suspend, declarative View fields
- Context7 `/charmbracelet/lipgloss` -- v2 styling, LightDark adaptive colors, Place/JoinVertical/JoinHorizontal layout, Border, table rendering
- Context7 `/charmbracelet/bubbles` -- v2 key bindings, help component API, key.Matches pattern
- Codebase: `internal/daemon/client.go` -- DaemonClient API (ListSessions, GetWebServerStatus, Health)
- Codebase: `internal/daemon/types.go` -- SessionInfo, WebServerStatusResponse types
- Codebase: `internal/daemon/engine.go` -- ListSessions implementation, sessionStatuses map
- Codebase: `internal/status/detector.go` -- SessionStatus type, status constants
- Codebase: `main.go` -- CLI dispatch pattern
- `76-UI-SPEC.md` -- Approved UI contract (layout, colors, keybindings, anti-patterns)

### Secondary (MEDIUM confidence)
- [Writing Bubble Tea Tests - Charm](https://charm.land/blog/teatest/) -- teatest golden file testing approach
- [Bubble Tea v2 Discussion](https://github.com/charmbracelet/bubbletea/discussions/1374) -- v2 changelog and migration notes

### Tertiary (LOW confidence)
- teatest v2 package availability and API stability [not verified -- experimental package]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- versions verified via `go list -m -versions`, APIs verified via Context7
- Architecture: HIGH -- patterns from Bubble Tea v2 official docs, codebase analysis confirms data source availability
- Pitfalls: HIGH -- v1/v2 migration pitfalls from official UPGRADE_GUIDE_V2.md; data gap identified via codebase inspection
- Testing: MEDIUM -- direct unit testing is solid; teatest golden testing is experimental

**Research date:** 2026-04-15
**Valid until:** 2026-05-15 (Charm ecosystem is stable post-v2 GA)
