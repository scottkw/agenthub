# Phase 86: TUI Visual Polish - Pattern Map

**Mapped:** 2026-04-19
**Files analyzed:** 6 (all modifications to existing files in `internal/tui/`)
**Analogs found:** 6 / 6

---

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------|------|-----------|----------------|---------------|
| `internal/tui/styles.go` | config/style | transform | `internal/tui/styles.go` (self — full replacement) | exact |
| `internal/tui/model.go` | model | event-driven | `internal/tui/model.go` (self — additive) | exact |
| `internal/tui/keys.go` | config/binding | event-driven | `internal/tui/keys.go` (self — additive) | exact |
| `internal/tui/view.go` | component | request-response | `internal/tui/help.go` + `internal/tui/modal.go` | exact |
| `internal/tui/update.go` | controller | event-driven | `internal/tui/update.go` (self — additive) | exact |
| `internal/tui/tui.go` | entry-point | request-response | `internal/tui/tui.go` (self — signature change) | exact |

**New test files (Wave 0):**

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `internal/tui/styles_test.go` | test | transform | `internal/tui/view_test.go` | role-match |
| `internal/tui/view_test.go` (extended) | test | request-response | `internal/tui/view_test.go` (self — additive) | exact |
| `internal/tui/update_test.go` (extended) | test | event-driven | `internal/tui/update_test.go` (self — additive) | exact |

---

## Pattern Assignments

### `internal/tui/styles.go` (config, transform)

**Analog:** `internal/tui/styles.go` (current file — full replacement of color tokens)

**Current struct + factory pattern** (lines 1-62 — full file):
```go
// Existing Styles struct pattern — extend with new tokens, same factory shape
type Styles struct {
    FgNormal      color.Color
    FgMuted       color.Color
    FgAccent      color.Color
    BgSelected    color.Color
    FgSelected    color.Color
    // ... existing tokens
    BorderNormal  color.Color
    BorderAccent  color.Color
    // ADD THESE new tokens (D-15 + UI-SPEC color table):
    BgSurface     color.Color  // #1a1b26 dark / #e9e9ec light
    BgSidebar     color.Color  // #16161e dark / #f0f0f4 light
    // Per-agent badge colors (D-12):
    BadgeClaude   color.Color  // #7aa2f7 dark / #2e7de9 light
    BadgeOpencode color.Color  // #9ece6a dark / #485e30 light
    BadgeCodex    color.Color  // #bb9af7 dark / #7847bd light
    BadgeGemini   color.Color  // #2ac3de dark / #118c9e light
    BadgeCursor   color.Color  // #e0af68 dark / #8c6c3e light
    BadgeAider    color.Color  // #f7768e dark / #c64343 light
}

// Adaptive color factory pattern (lines 37-62) — COPY THIS EXACTLY, replace values:
func newStyles(hasDark bool) Styles {
    ld := lipgloss.LightDark(hasDark)
    return Styles{
        // Replace ALL existing values with TokyoNight equivalents:
        FgNormal:      ld(lipgloss.Color("#3b4261"), lipgloss.Color("#c0caf5")),
        FgMuted:       ld(lipgloss.Color("#9699b0"), lipgloss.Color("#565f89")),
        FgAccent:      ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),
        BgSelected:    ld(lipgloss.Color("#d0d5e3"), lipgloss.Color("#283457")),
        FgSelected:    ld(lipgloss.Color("#3b4261"), lipgloss.Color("#c0caf5")),
        StatusRunning: ld(lipgloss.Color("#485e30"), lipgloss.Color("#9ece6a")),
        StatusIdle:    ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),
        StatusWaiting: ld(lipgloss.Color("#8c6c3e"), lipgloss.Color("#e0af68")),
        StatusErrored: ld(lipgloss.Color("#c64343"), lipgloss.Color("#f7768e")),
        // ... remaining tokens per UI-SPEC color table
    }
}
```

**Critical constraint:** `ld(light, dark)` — light value is FIRST argument. Verified at `styles.go` line 38: `ld := lipgloss.LightDark(hasDark)` where `LightDark` returns `func(light, dark)`.

**Anti-pattern to avoid:** Never write `lipgloss.Color("#7aa2f7")` without wrapping in `ld(...)`. All new tokens must use the adaptive pattern.

---

### `internal/tui/model.go` (model, event-driven)

**Analog:** `internal/tui/model.go` — additive changes to existing Model struct and new type constants.

**Existing modal state const pattern** (lines 13-19) — copy to add new focus/tab constants:
```go
// COPY THIS PATTERN for new pane focus and tab type constants
type modalState int
const (
    modalNone modalState = iota
    modalNewSession
    modalKillConfirm
)

// NEW — add analogous pane focus type:
type panesFocusState bool
const (
    focusContent panesFocusState = false
    focusSidebar panesFocusState = true
)

// NEW — add tab type:
type tabID int
const (
    tabHome     tabID = iota
    tabSessions
    tabRemote
    tabSettings
)
```

**Existing Model struct pattern** (lines 84-131) — additive fields to append:
```go
// EXISTING Model struct — append these fields after qrURL (line 130):
// Tab navigation state (Phase 86)
panesFocus   panesFocusState    // focusContent | focusSidebar
openTabs     []tabID            // ordered list of open tabs
activeTab    int                // index into openTabs (not a tabID)
sidebarFocus int                // 0=Home 1=Sessions 2=Remote 3=Settings
version      string             // passed from daemon.BuildVersion via Run()
```

**Existing listEntry kind pattern** (lines 30-37) — copy for new tab struct if needed:
```go
// Tab struct (simple, no interface needed):
type openTab struct {
    id   tabID
    name string
}
```

**Message type pattern** (lines 133-160) — no new message types needed for Phase 86 tab/sidebar state (it is pure local Model state, not async).

---

### `internal/tui/keys.go` (config, event-driven)

**Analog:** `internal/tui/keys.go` — additive bindings following exact existing pattern.

**Existing KeyMap struct + factory** (lines 6-72) — copy binding pattern:
```go
// EXISTING pattern — copy for 3 new bindings:
type KeyMap struct {
    // ... existing bindings ...
    QR      key.Binding // Phase 78
    // ADD (Phase 86):
    TabFocus key.Binding // toggle sidebar/content focus
    PrevTab  key.Binding // [ cycle tabs backward
    NextTab  key.Binding // ] cycle tabs forward
}

func defaultKeyMap() KeyMap {
    return KeyMap{
        // ... existing bindings ...
        // ADD (same pattern as existing QR binding at lines 67-70):
        TabFocus: key.NewBinding(
            key.WithKeys("tab"),
            key.WithHelp("Tab", "toggle sidebar/content focus"),
        ),
        PrevTab: key.NewBinding(
            key.WithKeys("["),
            key.WithHelp("[", "previous tab"),
        ),
        NextTab: key.NewBinding(
            key.WithKeys("]"),
            key.WithHelp("]", "next tab"),
        ),
    }
}
```

**Critical constraint:** `TabFocus` must NOT be dispatched at the top of `handleKey()`. It must only be reachable after the modal priority checks (priorities 1-5 in `update.go`) to avoid conflicting with the existing `tab` key in `handleNewSessionKey` (line 503 of `update.go`).

---

### `internal/tui/view.go` (component, request-response)

**Analog:** `internal/tui/help.go` for bordered frame pattern; `internal/tui/modal.go` for `injectBorderTitle` pattern; existing `view.go` for row/column layout patterns.

**`renderFull()` refactor pattern** (existing lines 42-75) — restructure to two-pane:
```go
// EXISTING flat structure (lines 42-75) — replace body with JoinHorizontal:
func (m Model) renderFull() string {
    if m.showHelp {
        return m.renderHelpOverlay()
    }
    // Modal overlays remain at top (same priority as before)
    if m.modal == modalNewSession { return m.renderNewSessionModal() }
    if m.modal == modalKillConfirm { return m.renderKillConfirmModal() }
    if m.qrSession != nil { return m.renderQROverlay() }

    // NEW two-pane layout:
    sidebar := m.renderSidebar()
    tabBar := m.renderTabBar()
    content := m.renderContentPane()
    footer := m.renderFooter()

    right := lipgloss.JoinVertical(lipgloss.Left, tabBar, content, footer)
    return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
}
```

**Bordered frame pattern** — copy from `help.go` lines 16-33:
```go
// SOURCE: internal/tui/help.go renderHelpOverlay() lines 16-33
bordered := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(m.styles.BorderNormal).  // use BorderAccent when content pane is focused
    Width(contentWidth - 2).
    Padding(0, 2).
    Render(innerContent)

title := lipgloss.NewStyle().
    Bold(true).
    Foreground(m.styles.FgAccent).
    Render(" Sessions ")

bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)
// NOTE: pass BorderAccent instead of BorderNormal when pane is focused
```

**`injectBorderTitle` — existing function** (lines 396-425 of view.go) — reuse as-is, no changes. Signature:
```go
func injectBorderTitle(bordered string, title string, borderColor color.Color) string
```

**`renderSessionRow` width fix** (lines 212-257) — after sidebar added, replace `m.width` with `m.contentWidth()`:
```go
// EXISTING lines 247-252 — change m.width to m.contentWidth():
return lipgloss.NewStyle().
    Background(m.styles.BgSelected).
    Foreground(m.styles.FgSelected).
    Width(m.contentWidth()).   // WAS: m.width
    Render(row)
```

**`nameColWidth()` fix** (lines 371-379) — subtract sidebar from available width:
```go
// EXISTING lines 371-379 — add contentWidth() helper and update formula:
func (m Model) contentWidth() int {
    w := m.width - m.sidebarWidth() - 1 // -1 for separator char
    if w < 20 {
        return 20
    }
    return w
}

func (m Model) sidebarWidth() int {
    return 16 // exact value is executor discretion per CONTEXT.md
}

func (m Model) nameColWidth() int {
    w := m.contentWidth() - 53  // WAS: m.width - 53
    if w < 8 {
        return 8
    }
    return w
}
```

**Sidebar render pattern** — no exact analog, use styled text rows (NOT per-item border boxes per RESEARCH.md anti-pattern):
```go
// NEW renderSidebar() — modeled on existing lipgloss.NewStyle() row pattern:
func (m Model) renderSidebar() string {
    items := []struct{ id int; label string }{
        {0, "Home"},
        {1, "Sessions"},
        {2, "Remote"},
        {3, "Settings"},
    }
    var rows []string
    for _, item := range items {
        style := lipgloss.NewStyle().Foreground(m.styles.FgMuted).Width(m.sidebarWidth())
        if item.id == m.sidebarFocus {
            style = style.Bold(true).Foreground(m.styles.FgAccent)
            if m.panesFocus == focusSidebar {
                style = style.Background(m.styles.BgSelected)
            }
        }
        rows = append(rows, style.Render("  "+item.label))
    }
    return lipgloss.NewStyle().
        Background(m.styles.BgSidebar).
        Width(m.sidebarWidth()).
        Height(m.height - 2).  // -2 for footer
        Render(strings.Join(rows, "\n"))
}
```

**Tab bar render pattern** — based on RESEARCH.md Pattern 5, using existing lipgloss style conventions:
```go
// NEW renderTabBar() — copy Bold/Foreground style pattern from renderColumnHeaders() (lines 115-125):
func (m Model) renderTabBar() string {
    var parts []string
    for i, tab := range m.openTabs {
        label := "  " + tabName(tab) + "  "
        if i == m.activeTab {
            parts = append(parts, lipgloss.NewStyle().
                Bold(true).Foreground(m.styles.FgAccent).Underline(true).
                Render(label))
        } else {
            parts = append(parts, lipgloss.NewStyle().
                Foreground(m.styles.FgMuted).Render(label))
        }
    }
    return lipgloss.NewStyle().
        Width(m.contentWidth()).
        Foreground(m.styles.BorderNormal).
        Render(strings.Join(parts, ""))
}
```

**Agent badge render pattern** — modify `renderSessionRow` to render badge inline (D-11, D-12):
```go
// In renderSessionRow() — replace plain agent text with colored badge:
// EXISTING line 237: agent := truncate(s.CLI, 12)
// REPLACE WITH:
badgeColor := agentBadgeColor(s.CLI, m.styles)
badge := "[" + truncate(strings.ToLower(s.CLI), 10) + "]"
agentBadge := lipgloss.NewStyle().Foreground(badgeColor).Render(badge)
// Then use agentBadge in the row format string (note: lipgloss width for alignment)
```

**`agentBadgeColor()` function** — new helper, modeled on `statusGlyph()` (lines 427-439):
```go
// NEW agentBadgeColor() — copy switch pattern from statusGlyph() lines 427-439:
func agentBadgeColor(cli string, s Styles) color.Color {
    switch strings.ToLower(cli) {
    case "claude":   return s.BadgeClaude
    case "opencode": return s.BadgeOpencode
    case "codex":    return s.BadgeCodex
    case "gemini":   return s.BadgeGemini
    case "cursor":   return s.BadgeCursor
    case "aider":    return s.BadgeAider
    default:         return s.FgMuted
    }
}
```

**Home tab render pattern** — uses existing `renderWebStatus()` data + new branding:
```go
// NEW renderHomeTab() — branding modeled on renderHeader() (lines 78-112):
// Title: Bold FgNormal "AgentHub" — same style as renderHeader() line 79
// Tagline: FgMuted "AI coding terminal sessions" — same style as renderHintBar()
// Stats block: uses injectBorderTitle bordered frame (copy from help.go pattern)
// Session counts: derived from m.sessions (same loop as renderHeader() lines 84-99)
// Web status: reuse m.webStatus fields (same as renderWebStatus() lines 326-335)
// Tailscale: derive from len(m.remoteSessions) > 0 (no new API per RESEARCH.md open question A)
```

---

### `internal/tui/update.go` (controller, event-driven)

**Analog:** `internal/tui/update.go` — additive routing in `handleKey()` following established priority chain.

**Priority chain pattern** (lines 110-137) — insert new pane-focus routing at priority 6, replacing existing `handleMainKey` call:
```go
// EXISTING handleKey() lines 110-137:
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    // Priority 1: editing (line 112)
    if m.editing { return m.handleRenameKey(msg) }
    // Priority 2: kill confirm (line 116)
    if m.modal == modalKillConfirm { return m.handleKillConfirmKey(msg) }
    // Priority 3: new session modal (line 120) — intercepts "tab" key
    if m.modal == modalNewSession { return m.handleNewSessionKey(msg) }
    // Priority 4: QR (line 124)
    if m.qrSession != nil { return m.handleQRKey(msg) }
    // Priority 5: help (line 128)
    if m.showHelp { /* ... */ }

    // Priority 6: NEW — tab cycling (safe here because no modal uses [ or ])
    if key.Matches(msg, m.keys.PrevTab) { m.cycleTab(-1); return m, nil }
    if key.Matches(msg, m.keys.NextTab) { m.cycleTab(+1); return m, nil }

    // Priority 7: NEW — pane focus dispatch (TabFocus is INSIDE these handlers, not here)
    if m.panesFocus == focusSidebar {
        return m.handleSidebarKey(msg)
    }
    return m.handleContentKey(msg)  // replaces handleMainKey
}
```

**`handleMainKey` becomes `handleContentKey`** — rename and add TabFocus toggle inside:
```go
// handleContentKey wraps existing handleMainKey logic, adds TabFocus at top:
func (m Model) handleContentKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    // TabFocus placed HERE (not at top of handleKey) to avoid modal conflict:
    if key.Matches(msg, m.keys.TabFocus) {
        m.panesFocus = focusSidebar
        return m, nil
    }
    // ... rest of existing handleMainKey() cases unchanged ...
}
```

**`handleSidebarKey` pattern** — modeled on `handleKillConfirmKey` switch structure (lines 451-472):
```go
// NEW handleSidebarKey() — copy switch-on-string pattern from handleKillConfirmKey():
func (m Model) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    if key.Matches(msg, m.keys.TabFocus) {
        m.panesFocus = focusContent
        return m, nil
    }
    switch msg.String() {
    case "up", "k":
        if m.sidebarFocus > 0 { m.sidebarFocus-- }
    case "down", "j":
        if m.sidebarFocus < 3 { m.sidebarFocus++ }
    case "enter":
        m.openTab(tabID(m.sidebarFocus))
        m.panesFocus = focusContent
    case "Q", "ctrl+c":
        return m, tea.Quit
    }
    return m, nil
}
```

**`cycleTab` helper** — modeled on `cycleAgent()` (modal.go lines 86-96):
```go
// NEW cycleTab() — copy modular arithmetic pattern from cycleAgent() in modal.go:
func (m *Model) cycleTab(dir int) {
    if len(m.openTabs) == 0 { return }
    m.activeTab = (m.activeTab + dir + len(m.openTabs)) % len(m.openTabs)
}
```

**`attachDoneMsg` handler** (lines 63-70) — tab state is automatically preserved because `Model` is a value type. The `openTabs`, `activeTab`, `panesFocus` fields survive `tea.Exec` unchanged. No code change needed in the `attachDoneMsg` case.

---

### `internal/tui/tui.go` (entry-point, request-response)

**Analog:** `internal/tui/tui.go` (self — signature change only).

**Current `Run` + `newModel` signatures** (lines 13-30):
```go
// EXISTING (lines 13-17):
func Run(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn) error {
    p := tea.NewProgram(newModel(client, fetchRemoteFn))
    _, err := p.Run()
    return err
}

// EXISTING (lines 21-30):
func newModel(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn) Model {
    return Model{
        client:        client,
        loading:       true,
        keys:          defaultKeyMap(),
        styles:        newStyles(true),
        detectedCLIs:  pty.DetectCLIs(),
        fetchRemoteFn: fetchRemoteFn,
    }
}
```

**Required change** — add `version string` parameter + initialize tab state:
```go
// REPLACE WITH (same structure, adds version param and initial tab state):
func Run(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn, version string) error {
    p := tea.NewProgram(newModel(client, fetchRemoteFn, version))
    _, err := p.Run()
    return err
}

func newModel(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn, version string) Model {
    return Model{
        client:        client,
        loading:       true,
        keys:          defaultKeyMap(),
        styles:        newStyles(true),
        detectedCLIs:  pty.DetectCLIs(),
        fetchRemoteFn: fetchRemoteFn,
        version:       version,
        // Initial tab state — Sessions open by default (matches current UX):
        openTabs:    []tabID{tabSessions},
        activeTab:   0,
        panesFocus:  focusContent,
        sidebarFocus: 1, // 1 = Sessions
    }
}
```

**Call site update required** — `cmd_tui.go` line 53:
```go
// EXISTING (cmd_tui.go line 53):
return tui.Run(client, fetchRemoteFn)

// REPLACE WITH:
return tui.Run(client, fetchRemoteFn, daemon.BuildVersion)
```

`daemon.BuildVersion` is confirmed as the correct package-level var (RESEARCH.md A2, verified in `main.go` and `internal/daemon/process.go`).

---

## New Test Files (Wave 0)

### `internal/tui/styles_test.go` (new file)

**Analog:** `internal/tui/view_test.go` — copy `package tui`, imports, and `testModel()` reference (defined in `update_test.go` lines 13-21).

**Test pattern** — copy table-driven test style from `TestStatusGlyph` (view_test.go lines 154-173):
```go
// SOURCE: view_test.go TestStatusGlyph lines 154-173
func TestAgentBadgeColor(t *testing.T) {
    s := newStyles(true) // same as TestStatusGlyph line 155
    tests := []struct {
        cli  string
        want color.Color
    }{
        {"claude", s.BadgeClaude},
        {"opencode", s.BadgeOpencode},
        // ...
    }
    for _, tt := range tests {
        got := agentBadgeColor(tt.cli, s)
        if got != tt.want {
            t.Errorf("agentBadgeColor(%q) = %v, want %v", tt.cli, got, tt.want)
        }
    }
}
```

**Styles palette test** — assert token values are distinct from old 256-color approximations:
```go
func TestStyles_TokyoNight(t *testing.T) {
    s := newStyles(true) // dark
    // Spot-check key TokyoNight hex values per UI-SPEC
    // FgAccent dark must be #7aa2f7 (lipgloss.Color is a string type)
    if string(s.FgAccent.(lipgloss.Color)) != "#7aa2f7" {
        t.Errorf("FgAccent dark = %v, want #7aa2f7", s.FgAccent)
    }
}
```

### `internal/tui/view_test.go` extensions

**Analog:** `internal/tui/view_test.go` — additive test functions following existing naming convention `TestView_*`.

**Test fixture pattern** — `testModel()` is defined in `update_test.go` lines 13-21 (same package, so accessible in view_test.go):
```go
// All new view tests use testModel() — same as all existing view tests:
func TestView_SessionFrame(t *testing.T) {
    m := testModel()
    m.sessions = []daemon.SessionInfo{...}
    m.rebuildUnifiedList()
    // call m.renderSessionFrame() or check v.Content for border chars
    if !strings.Contains(content, "╭") { ... }  // RoundedBorder top-left
}

func TestView_Sidebar(t *testing.T) {
    m := testModel()
    sidebar := m.renderSidebar()
    for _, label := range []string{"Home", "Sessions", "Remote", "Settings"} {
        if !strings.Contains(sidebar, label) {
            t.Errorf("sidebar missing %q", label)
        }
    }
}

func TestView_AgentBadge(t *testing.T) {
    m := testModel()
    m.sessions = []daemon.SessionInfo{
        {ID: "1", Name: "test", CLI: "claude", Status: "running"},
    }
    m.rebuildUnifiedList()
    row := m.renderSessionRow(m.sessions[0], 0)
    if !strings.Contains(row, "[claude]") {
        t.Errorf("session row missing agent badge [claude], got: %q", row)
    }
}
```

### `internal/tui/update_test.go` extensions

**Analog:** `internal/tui/update_test.go` — additive `TestUpdate_*` functions, same package structure.

**Key press test pattern** — copy from existing `TestUpdate_*` that sends `tea.KeyPressMsg` (lines 80+ in update_test.go):
```go
func TestUpdate_TabFocusToggle(t *testing.T) {
    m := testModel()
    m.panesFocus = focusContent

    // Send Tab key — goes to handleContentKey which dispatches TabFocus
    updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
    result := updated.(Model)

    if result.panesFocus != focusSidebar {
        t.Error("Tab key should toggle focus to sidebar")
    }
}

func TestUpdate_TabCycle(t *testing.T) {
    m := testModel()
    m.openTabs = []tabID{tabHome, tabSessions, tabRemote}
    m.activeTab = 0

    updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRunes, Runes: []rune{']'}})
    result := updated.(Model)
    if result.activeTab != 1 {
        t.Errorf("expected activeTab=1 after ], got %d", result.activeTab)
    }
}
```

---

## Shared Patterns

### Adaptive Color Pattern
**Source:** `internal/tui/styles.go` lines 37-62
**Apply to:** All new color token additions in `styles.go`
```go
ld := lipgloss.LightDark(hasDark)
// ALWAYS: ld(lightValue, darkValue) — light FIRST
SomeToken: ld(lipgloss.Color("#lightHex"), lipgloss.Color("#darkHex")),
```

### Bordered Frame with Title
**Source:** `internal/tui/help.go` lines 16-33 (help overlay) and `internal/tui/modal.go` lines 16-29 (new session modal)
**Apply to:** `renderSessionFrame()`, `renderRemoteTab()`, Home tab stats section
```go
bordered := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(borderColor).  // pass as parameter, not global state
    Width(width - 2).
    Padding(0, 2).
    Render(innerContent)
title := lipgloss.NewStyle().Bold(true).Foreground(m.styles.FgAccent).Render(" Label ")
bordered = injectBorderTitle(bordered, title, borderColor)
```

### Width Measurement
**Source:** `internal/tui/view.go` lines 106-108 (renderHeader gap calc)
**Apply to:** All new layout computations
```go
// ALWAYS lipgloss.Width() for ANSI-aware measurement, NEVER len()
gap := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
```

### Modal Overlay Placement
**Source:** `internal/tui/help.go` line 32 / `internal/tui/modal.go` lines 31-32
**Apply to:** Any new overlay rendered on top of TUI content (currently no new overlays in Phase 86)
```go
return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, bordered)
```

### Key Priority Chain
**Source:** `internal/tui/update.go` lines 110-137
**Apply to:** New `handleSidebarKey()` and `handleContentKey()` — modal checks must remain at priorities 1-5 before any new routing.

### Test Fixture
**Source:** `internal/tui/update_test.go` lines 13-21 (`testModel()`)
**Apply to:** All new test functions in `view_test.go`, `update_test.go`, and new `styles_test.go`
```go
// testModel() is defined ONCE in update_test.go — available to all files in package tui tests
m := testModel()  // nil client/fetchRemoteFn, width=120, height=24, dark styles
```

---

## No Analog Found

All Phase 86 files are modifications to existing `internal/tui/` package files. No genuinely novel file role exists. The rendering patterns for `renderSidebar()`, `renderTabBar()`, `renderHomeTab()`, `renderSettingsTab()`, and `renderRemoteTab()` have no direct code analog in the codebase but are composable from the shared patterns above. The planner should use RESEARCH.md Pattern 1 (two-pane layout) and Pattern 5 (tab bar) for these functions.

---

## Metadata

**Analog search scope:** `internal/tui/` (all 14 files)
**Files scanned:** 14 source files + 5 test files
**Pattern extraction date:** 2026-04-19
