# Phase 86: TUI Visual Polish - Research

**Researched:** 2026-04-19
**Domain:** Go / Bubble Tea v2 / lipgloss v2 TUI layout and styling
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Vertical narrow sidebar on the left mirroring the GUI sidebar layout — short labels for Home, Sessions, Remote, Settings
- **D-02:** Arrow keys (Up/Down) move focus within the sidebar, Enter selects a section
- **D-03:** Tab key toggles focus between the sidebar pane and the content pane, with a clear visual indicator (accent border or highlight) showing which pane is active
- **D-04:** Sidebar items open as persistent tabs in a horizontal tab bar at the top of the content area — multiple tabs can be open simultaneously (Home, Settings, session terminals), matching the GUI's tabbed model
- **D-05:** `[` and `]` keys cycle through open tabs (previous/next)
- **D-06:** Attaching to a session uses existing full-screen `tea.Exec` approach (suspends Bubble Tea, hands terminal to PTY). On detach (Ctrl-\\), returns to TUI with all tab state preserved — the session's tab remains highlighted/open.
- **D-07:** Inline PTY rendering within tabs (embedding a terminal emulator in the Bubble Tea pane) is deferred to a future version.
- **D-08:** Single outer bordered frame (rounded corners via lipgloss) around the entire session list, with the section name as a border title using `injectBorderTitle`
- **D-09:** Thin horizontal dividers between local and remote session groups inside the frame
- **D-10:** Column headers (NAME, AGENT, HOST, VIEWERS) render as the first row inside the frame, with a separator line below them
- **D-11:** Enhanced column layout: keep current column structure but add colored agent badges (short tags like `[claude]`, `[codex]`), bolder status glyphs, and subtle background highlight on selected row
- **D-12:** Per-agent distinct badge colors — each CLI gets its own accent color for quick visual scanning (derived from agent brand or assigned from TokyoNight palette)
- **D-13:** Combined branding + dashboard: AgentHub title, version, and tagline at the top; bordered stats section below with session counts (by status), web server status, Tailscale connection state, and quick-action hints
- **D-14:** Branding anchors the top of the view, live stats provide utility below a separator
- **D-15:** Update TUI color tokens in `styles.go` to align with the GUI's TokyoNight palette — primary accent `#7aa2f7`, background `#1a1b26`, muted `#565f89`, and status colors derived from TokyoNight variants

### Claude's Discretion

- Exact sidebar width (characters) — balance between label readability and content space
- Per-agent badge color assignments (specific hex values per CLI)
- Tab bar rendering style (underline, bracket, highlight)
- Separator line style inside frames (thin horizontal rule vs box-drawing chars)
- Settings tab content (read-only display of current settings, or pointer to `agenthub settings`)
- Remote tab content layout (reuse session list frame pattern or custom layout)
- Focus indicator styling (accent border, background color, or cursor glyph)
- Quick-action hint formatting on Home tab

### Deferred Ideas (OUT OF SCOPE)

- **Inline PTY tab rendering** — Embed a terminal emulator (VT100 state machine) inside Bubble Tea to render live PTY output in a content pane tab. Deferred to a future version.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUI-01 | Session list displayed in bordered lipgloss frames with section headers | `injectBorderTitle()` + `lipgloss.RoundedBorder()` already exist in `help.go` and `modal.go` — reuse the exact pattern |
| TUI-02 | Tabbed navigation mimicking GUI sidebar (Home, Sessions, Remote, Settings) | New `sidebarState` + `tabBarState` structs in `model.go`; focus-aware routing in `handleKey()` |
| TUI-03 | Styled session rows with agent type, status glyphs, hostname, and viewer count matching GUI aesthetic | Badge column replaces plain agent text; per-agent colors added to `Styles` struct |
| TUI-04 | Consistent color palette and typography between TUI and GUI (TokyoNight-derived) | Full `styles.go` replacement per UI-SPEC color table; new tokens `BgSidebar`, `BgSurface` |
</phase_requirements>

---

## Summary

Phase 86 transforms the flat Bubble Tea v2 TUI into a structured two-pane layout: a narrow vertical sidebar on the left (Home / Sessions / Remote / Settings) and a content pane on the right with a horizontal tab bar at the top. The existing session list is wrapped in a lipgloss rounded-border frame. Session rows gain colored agent badges and the entire color palette migrates from the current 256-color approximations to exact TokyoNight hex values matching the GUI.

The codebase is already well-structured for this refactor. All the building blocks exist — `injectBorderTitle()`, `lipgloss.RoundedBorder()`, `statusGlyph()`, `renderDividerRow()`, adaptive `Styles`, and `tea.Exec` attach — and none need to be replaced. The work is additive state in `model.go`, a restructured `renderFull()` in `view.go`, focus-aware key routing in `update.go`, three new key bindings in `keys.go`, and a palette replacement in `styles.go`.

The UI-SPEC (86-UI-SPEC.md) is the authoritative visual contract. All color values, glyph choices, and layout proportions in this research derive from it. The planner should treat the UI-SPEC as a locked downstream constraint.

**Primary recommendation:** Refactor `renderFull()` to emit `sidebar | tabBar + contentPane + footer` using `lipgloss.JoinHorizontal`, with focus state carried in `Model`. All existing render functions survive as content suppliers to the new pane structure.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Color palette / Styles | `styles.go` | — | Single source of truth for all color tokens; rebuilt on `BackgroundColorMsg` |
| Sidebar rendering | `view.go` (new `renderSidebar()`) | — | Pure render function, reads `sidebarFocus` + `activeSidebarItem` from Model |
| Tab bar rendering | `view.go` (new `renderTabBar()`) | — | Renders open tabs, marks active tab |
| Content pane dispatch | `view.go` (refactored `renderContentPane()`) | — | Routes to Home / Sessions / Remote / Settings renderer based on active tab |
| Session frame | `view.go` (refactored `renderSessionList()`) | `view.go` (injectBorderTitle) | Wraps existing list in lipgloss border |
| Home tab | `view.go` (new `renderHomeTab()`) | `model.go` (version field) | Branding + live stats |
| Settings tab | `view.go` (new `renderSettingsTab()`) | `model.go` (settings snapshot) | Read-only display |
| Remote tab | `view.go` (new `renderRemoteTab()`) | — | Reuse session frame pattern with remote-only rows |
| Focus model / key routing | `update.go` (refactored `handleKey()`) | `model.go` (panesFocus field) | Tab toggles pane focus; Up/Down context-sensitive |
| Key bindings | `keys.go` | — | Add Tab, `[`, `]` bindings |
| Version display | `model.go` (new `version string` field) | `tui.go` (Run/newModel signature) | Pass from `daemon.BuildVersion` at startup |

---

## Standard Stack

### Core (already in go.mod — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.5 | Event loop, alt-screen, tea.Exec | Project standard; v2 API used throughout |
| `charm.land/lipgloss/v2` | v2.0.3 | Styles, borders, placement, JoinHorizontal/Vertical | Project standard; all layout uses this |
| `charm.land/bubbles/v2` | v2.1.0 | textinput component (existing modals) | Project standard |
| `github.com/charmbracelet/x/ansi` | v0.11.7 | ANSI strip for `injectBorderTitle` | Already imported in `view.go` |

**No new Go package dependencies are introduced in this phase.** [VERIFIED: go.mod read]

### Installation
No installation required — all packages already present in `go.mod` / `go.sum`.

---

## Architecture Patterns

### System Architecture Diagram

```
Key press / Resize / Tick / Fetch msgs
         │
         ▼
   Update() — tea.Model
         │
   handleKey()
         │
   ┌─────┴──────────────────────────────────┐
   │             Focus Router               │
   │  editing? → handleRenameKey()          │
   │  modal?   → handleKillConfirmKey()     │
   │             handleNewSessionKey()      │
   │  qrOpen?  → handleQRKey()             │
   │  help?    → dismiss help              │
   │  else:                                 │
   │    panesFocus==sidebar                 │
   │      → handleSidebarKey()  (NEW)       │
   │    panesFocus==content                 │
   │      → handleContentKey()  (NEW)       │
   │    tab key → toggle panesFocus (NEW)   │
   │    [ / ]   → cycle tabs    (NEW)       │
   └────────────────────────────────────────┘
         │
         ▼
    View() — renders
         │
   renderFull()  (REFACTORED)
         │
   ┌─────┴─────────────────────────────────────────┐
   │  lipgloss.JoinHorizontal(                      │
   │    renderSidebar(),         (NEW)              │
   │    lipgloss.JoinVertical(                      │
   │      renderTabBar(),        (NEW)              │
   │      renderContentPane(),   (NEW dispatcher)  │
   │      renderFooter(),        (existing)         │
   │    )                                           │
   │  )                                             │
   └────────────────────────────────────────────────┘

renderContentPane() dispatches to:
  activeTab==tabHome     → renderHomeTab()     (NEW)
  activeTab==tabSessions → renderSessionFrame() (REFACTORED from renderSessionList)
  activeTab==tabRemote   → renderRemoteTab()   (NEW)
  activeTab==tabSettings → renderSettingsTab() (NEW)
```

### Recommended Project Structure

No new files are required. All changes happen within the existing `internal/tui/` package:

```
internal/tui/
├── model.go     — add: sidebarState, tabBarState, panesFocus, version, agentBadgeColor()
├── styles.go    — replace: all color tokens with TokyoNight palette; add BgSidebar, BgSurface
├── keys.go      — add: TabFocus, PrevTab, NextTab key bindings
├── view.go      — refactor renderFull(); add renderSidebar(), renderTabBar(),
│                   renderContentPane(), renderHomeTab(), renderSettingsTab(),
│                   renderRemoteTab(), renderSessionFrame()
├── update.go    — refactor handleKey(); add handleSidebarKey(), handleContentKey()
└── tui.go       — update Run() and newModel() to accept version string
```

### Pattern 1: Two-Pane Layout with lipgloss.JoinHorizontal

**What:** Sidebar and content area rendered as separate lipgloss strings, joined horizontally.
**When to use:** Any time you need a fixed-width left column beside a flexible right content area.

```go
// Source: lipgloss v2 layout API — verified in existing view.go usage
func (m Model) renderFull() string {
    sidebarWidth := 16 // or m.sidebarWidth()

    sidebar := lipgloss.NewStyle().
        Width(sidebarWidth).
        Height(m.height - 2). // minus footer
        Background(m.styles.BgSidebar).
        Render(m.renderSidebarItems())

    contentWidth := m.width - sidebarWidth - 1 // -1 for separator char

    tabBar := m.renderTabBar()
    contentPane := m.renderContentPane(contentWidth, m.height-2-1) // -tabbar row
    footer := m.renderFooter()

    right := lipgloss.JoinVertical(lipgloss.Left, tabBar, contentPane, footer)

    separator := lipgloss.NewStyle().
        Foreground(m.styles.BorderNormal).
        Render(strings.Repeat("│\n", m.height))

    return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, separator, right)
}
```

**Note:** `lipgloss.JoinHorizontal` aligns by top edge (`lipgloss.Top`). The sidebar must be padded to full height or the join will be ragged. Use `.Height(n)` on the sidebar style. [VERIFIED: existing `lipgloss.Place()` call in view.go confirms lipgloss layout API usage]

### Pattern 2: Bordered Frame with injectBorderTitle

**What:** A lipgloss RoundedBorder box with a styled title injected into the top border edge.
**When to use:** Session list frame, Remote tab frame — any section needing a labeled border.

```go
// Source: existing help.go renderHelpOverlay() — VERIFIED in codebase
bordered := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(m.styles.BorderNormal). // or BorderAccent when content pane focused
    Width(contentWidth - 2).
    Render(innerContent)

title := lipgloss.NewStyle().
    Bold(true).
    Foreground(m.styles.FgAccent).
    Render(" Sessions ")

bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)
```

**The border color must be passed to `injectBorderTitle`** as the third argument — it re-applies color to the non-title portions of the border top line. If the content pane has focus, pass `m.styles.BorderAccent`; otherwise pass `m.styles.BorderNormal`.

### Pattern 3: Focus-Aware Key Routing

**What:** A two-level dispatch: first check which pane has focus, then dispatch to pane-specific handler.
**When to use:** Any time the same key (Up/Down/Enter) means different things in different panes.

```go
// Source: existing handleKey() priority chain in update.go — VERIFIED
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
    // ... existing priority 1-5 (editing, modals, qr, help) ...

    // Tab: toggle pane focus (new, fires before sidebar/content dispatch)
    if key.Matches(msg, m.keys.TabFocus) {
        m.panesFocus = !m.panesFocus // false=content, true=sidebar
        return m, nil
    }
    // [ and ] cycle open tabs regardless of pane focus
    if key.Matches(msg, m.keys.PrevTab) {
        m.cycleTab(-1)
        return m, nil
    }
    if key.Matches(msg, m.keys.NextTab) {
        m.cycleTab(+1)
        return m, nil
    }

    if m.panesFocus == focusSidebar {
        return m.handleSidebarKey(msg)
    }
    return m.handleContentKey(msg) // routes to existing handleMainKey logic
}
```

### Pattern 4: Per-Agent Badge Colors

**What:** Map agent CLI name to a lipgloss color from the TokyoNight palette.
**When to use:** Rendering the `[claude]`-style badge in session rows.

```go
// Source: UI-SPEC.md per-agent badge color table — VERIFIED against styles.go pattern
func agentBadgeColor(cli string, s Styles) color.Color {
    switch strings.ToLower(cli) {
    case "claude":
        return s.BadgeClaude   // #7aa2f7 dark / #2e7de9 light
    case "opencode":
        return s.BadgeOpencode // #9ece6a dark / #485e30 light
    case "codex":
        return s.BadgeCodex    // #bb9af7 dark / #7847bd light
    case "gemini":
        return s.BadgeGemini   // #2ac3de dark / #118c9e light
    case "cursor":
        return s.BadgeCursor   // #e0af68 dark / #8c6c3e light
    case "aider":
        return s.BadgeAider    // #f7768e dark / #c64343 light
    default:
        return s.FgMuted       // #565f89 dark / #9699b0 light
    }
}
```

Badge tokens must be added to the `Styles` struct and populated in `newStyles()`.

### Pattern 5: Tab Bar Rendering

**What:** A single horizontal line showing open tab names; active tab bold + underline indicator.
**When to use:** Tab bar above the content pane.

```go
// Source: UI-SPEC.md Tab Bar section — based on existing lipgloss style patterns
func (m Model) renderTabBar() string {
    var parts []string
    for i, tab := range m.openTabs {
        label := "  " + tab.Name + "  "
        if i == m.activeTab {
            parts = append(parts, lipgloss.NewStyle().
                Bold(true).
                Foreground(m.styles.FgAccent).
                Underline(true).
                Render(label))
        } else {
            parts = append(parts, lipgloss.NewStyle().
                Foreground(m.styles.FgMuted).
                Render(label))
        }
    }
    return lipgloss.NewStyle().
        Width(m.width - m.sidebarWidth() - 1).
        Render(strings.Join(parts, ""))
}
```

### Pattern 6: Version String Injection

**What:** Pass app version string from `main.go`'s `Version` var into the TUI model for display in the Home tab.
**When to use:** `tui.Run()` and `newModel()` — add a `version string` parameter.

```go
// Source: main.go Version variable — VERIFIED
// In tui.go:
func Run(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn, version string) error { ... }
func newModel(client *daemon.DaemonClient, fetchRemoteFn FetchRemoteFn, version string) Model {
    return Model{ ..., version: version }
}
// In cmd_tui.go:
return tui.Run(client, fetchRemoteFn, daemon.BuildVersion)
```

### Anti-Patterns to Avoid

- **Separate lipgloss box per sidebar item:** Do NOT render each sidebar item in its own border box — this wastes vertical space. Use a single background-colored column with styled text rows.
- **Hardcoded dark-only colors:** The adaptive `ld := lipgloss.LightDark(hasDark)` pattern is mandatory. Every new color token must use `ld(lightValue, darkValue)`, not a bare `lipgloss.Color("#...")`.
- **Recalculating column widths in sidebar:** The existing `nameColWidth()` formula (`width - 53`) must subtract the sidebar width from total terminal width. The sidebar is not free space — update the formula to `width - sidebarWidth - 1 - 53`.
- **Using `Tab` key in modal forms:** The new `TabFocus` binding conflicts with the existing modal form `Tab` (cycles fields). The modal handlers (`handleNewSessionKey`, `handleKillConfirmKey`) already intercept `Tab` before the main pane focus router — this is safe because modals have higher priority in `handleKey()`.
- **Storing tab content as rendered strings:** Render tab content on each `View()` call from live Model state. Don't cache rendered strings — cached strings go stale when sessions or web status change between ticks.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Horizontal two-pane layout | Custom string column math | `lipgloss.JoinHorizontal` | Handles ANSI width correctly across multi-byte runes |
| Border with injected title | String index slicing | `injectBorderTitle()` (existing) | Already handles ANSI stripping safely — lives in view.go |
| Center/place overlay | `strings.Repeat` + manual padding | `lipgloss.Place()` | Handles terminal width, already used for all modals |
| ANSI-safe string width | `len(string)` | `lipgloss.Width()` | Counts visual cells, not bytes |
| Adaptive colors | `if hasDark { color1 } else { color2 }` | `lipgloss.LightDark(hasDark)(light, dark)` | Project standard, already in styles.go |

**Key insight:** The biggest risk in TUI layout is confusing byte-length with visual width. Every width calculation must use `lipgloss.Width()` on styled strings, never `len()`.

---

## Common Pitfalls

### Pitfall 1: Sidebar Width Steals From Content Column Widths

**What goes wrong:** `nameColWidth()` currently assumes the full terminal width is available for content. After adding a sidebar, the name column will be too wide and the row will wrap.

**Why it happens:** `nameColWidth = m.width - 53`. After adding a 16-char sidebar + 1-char separator, the available content width is `m.width - 17`, so `nameColWidth` must use `m.contentWidth() - 53`.

**How to avoid:** Add a `contentWidth()` helper that returns `m.width - m.sidebarWidth() - 1`, and replace all uses of `m.width` in column calculations with `m.contentWidth()`.

**Warning signs:** Session rows that wrap to a second line, or VIEWERS column appearing past the right edge.

### Pitfall 2: Tab Key Conflict Between New-Session Modal and Pane Focus

**What goes wrong:** The new `TabFocus` key binding conflicts with the existing Tab key used to cycle form fields in the new-session modal.

**Why it happens:** Both handlers respond to the `tab` key string.

**How to avoid:** The modal key handler (`handleNewSessionKey`) has priority 3 in `handleKey()`. The pane focus toggle must be added at priority 6 (main view), AFTER the modal check. The existing priority chain in `handleKey()` already guarantees this if you add `TabFocus` inside `handleContentKey()` / `handleSidebarKey()` rather than at the top of `handleKey()`. Confirm: `[` and `]` can safely be at the top-level (priority 6) since no existing modal uses these keys.

**Warning signs:** Tab key in new-session modal switches pane focus instead of cycling fields.

### Pitfall 3: JoinHorizontal Height Mismatch Causes Ragged Output

**What goes wrong:** `lipgloss.JoinHorizontal` aligns columns by their top edge. If the sidebar string is shorter (fewer lines) than the content string, the columns appear ragged.

**Why it happens:** The sidebar has fixed items (4 labels) but the content pane height varies.

**How to avoid:** Apply `.Height(targetHeight)` to the sidebar `lipgloss.Style` before rendering. Target height = `m.height - 2` (footer rows). This pads the sidebar background to the correct height.

**Warning signs:** Sidebar background cuts off midway, content extends below sidebar.

### Pitfall 4: Focus Border Color Not Reset After Pane Switch

**What goes wrong:** The session frame border stays `BorderAccent` color after switching to the Home tab (which has no frame), then switching back to Sessions — the border flickers or stays the wrong color.

**Why it happens:** Border color is determined at render time based on `panesFocus`. As long as `renderFull()` re-evaluates border color each frame, this is safe.

**How to avoid:** Pass border color as a parameter to `renderSessionFrame(contentWidth, panesFocused bool)` rather than reading global state. `renderFull()` passes `m.panesFocus == focusContent` as the `panesFocused` argument.

**Warning signs:** Border remains accent-colored after switching to a non-framed tab.

### Pitfall 5: `injectBorderTitle` insertPos Off-By-One at Narrow Widths

**What goes wrong:** `injectBorderTitle` has an early return when `borderWidth <= titleWidth+4`. At very narrow content pane widths (e.g., minimum terminal of 60 columns minus 17 = 43 content cols) with long section titles, the title may be silently dropped.

**Why it happens:** The guard in `injectBorderTitle()` (line 406 in view.go) returns the un-titled bordered string if the border is too narrow to fit the title.

**How to avoid:** Keep section names short (≤ 10 chars): "Sessions", "Remote", "Settings", "Home". At 43 columns, `lipgloss.Width(" Sessions ")` is 10 — the check `43 <= 10+4 = 14` is false, so it will render. No special handling needed for these names.

**Warning signs:** Section frame renders without a visible title at narrow terminals.

### Pitfall 6: Version Field Requires Run() Signature Change

**What goes wrong:** Adding `version string` to `tui.Run()` requires updating the call site in `cmd_tui.go` — easily overlooked if this is not flagged.

**Why it happens:** `tui.Run()` is a public function called from `cmd_tui.go` in package main.

**How to avoid:** Update both `tui.go` (Run + newModel) and `cmd_tui.go` in the same task. Pass `daemon.BuildVersion` (the package-level var set in `main.go`).

---

## Code Examples

### Verified Pattern: RoundedBorder Frame from help.go

```go
// Source: internal/tui/help.go renderHelpOverlay() — VERIFIED
bordered := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(m.styles.BorderNormal).
    Width(overlayWidth-2).
    Padding(1, 2).
    Render(content)

title := lipgloss.NewStyle().
    Bold(true).
    Foreground(m.styles.BorderAccent).
    Render(" Keybindings ")

bordered = injectBorderTitle(bordered, title, m.styles.BorderNormal)
```

### Verified Pattern: Adaptive Colors from styles.go

```go
// Source: internal/tui/styles.go newStyles() — VERIFIED
ld := lipgloss.LightDark(hasDark)
FgAccent: ld(lipgloss.Color("#2e7de9"), lipgloss.Color("#7aa2f7")),
// note: LightDark(hasDark) returns func(light, dark) — light value FIRST
```

### Verified Pattern: Width-Padded Row from renderSessionRow()

```go
// Source: internal/tui/view.go renderSessionRow() — VERIFIED
return lipgloss.NewStyle().
    Background(m.styles.BgSelected).
    Foreground(m.styles.FgSelected).
    Width(m.width).   // use contentWidth() after sidebar refactor
    Render(row)
```

### Verified Pattern: tea.Exec for Full-Screen Attach

```go
// Source: internal/tui/update.go handleMainKey() — VERIFIED
cmd := &attachCmd{client: m.client, sessionID: s.ID}
return m, tea.Exec(cmd, func(err error) tea.Msg {
    return attachDoneMsg{err: err}
})
// On return: attachDoneMsg arrives in Update(), session tab state is preserved
// because Model fields (activeTab, sidebarFocus) are value types on Model
```

### Verified Pattern: Tailscale Detection for Home Tab

```go
// Source: cmd_tui.go fetchRemoteFn + daemon.DaemonClient — VERIFIED
// Tailscale connectivity: check if remoteSessions fetch returned groups OR
// use client.ListTailnetPeers() to test connectivity.
// For Home tab display: show "Connected" if m.remoteSessions != nil && len > 0
// or m.tailscaleStatus field (new, fetched via daemon).
// Simpler: derive from existing m.remoteSessions — no new API call needed.
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global flat session list with header | Two-pane sidebar + content with tab bar | Phase 86 (this phase) | Navigation now matches GUI model |
| 256-color ANSI approximations | TokyoNight exact hex values | Phase 86 (this phase) | Visual parity with GUI |
| Plain text agent column | Colored `[agent]` badge | Phase 86 (this phase) | Per-agent quick scan |
| Single-mode view (sessions only) | Tabbed content: Home, Sessions, Remote, Settings | Phase 86 (this phase) | Extensible content model |

**Deprecated/outdated after this phase:**
- `renderHeader()` — replaced by Home tab branding section and tab bar
- `renderColumnHeaders()` — moves inside session frame, below border title
- Flat `renderFull()` structure — replaced by sidebar + content dispatch

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `lipgloss.JoinHorizontal` pads shorter strings to match the taller column's height | Architecture Patterns — Pattern 1 | If it does NOT pad, the sidebar and content join will produce ragged output. Use `.Height()` on the sidebar style as insurance regardless. |
| A2 | `daemon.BuildVersion` is the correct package-level variable to pass as version string to TUI | Code Examples — Version Pattern | If the version field is elsewhere (e.g., a struct method), `cmd_tui.go` call site changes. Confirmed: main.go line 43 sets `daemon.BuildVersion = Version`, and `daemon.BuildVersion` is a package-level var in `process.go`. [VERIFIED] |

**A2 is effectively verified** — confirmed in both `main.go` and `internal/daemon/process.go` reads above.

---

## Open Questions

1. **Tailscale status on Home tab**
   - What we know: The Home tab should show "Tailscale: Connected" or "not detected" (UI-SPEC line 207)
   - What's unclear: The cheapest signal available. Options: (a) derive from `m.remoteSessions` being non-empty, (b) add a new `tailscaleStatus` field fetched separately, (c) call `client.ListTailnetPeers()` which already exists.
   - Recommendation: Option (a) — derive from existing `m.remoteSessions` being non-nil. If `fetchRemoteFn != nil` and at least one group was returned, show "Connected". No new daemon API needed. If `fetchRemoteFn == nil`, show "not configured".

2. **Settings tab data source**
   - What we know: UI-SPEC specifies read-only display of current settings (Claude path, web port, auto-close).
   - What's unclear: Whether a settings query exists on `daemon.DaemonClient`, or if settings must be read from a config file directly.
   - Recommendation: For this phase, show a pointer message ("run 'agenthub settings' to edit") as specified in the UI-SPEC copywriting contract, with only the fields retrievable from the already-fetched `webStatus` (port, running state). Avoids new daemon API for Phase 86.

3. **Session count stats on Home tab**
   - What we know: Home tab shows `{N} running  {N} idle` counts.
   - What's unclear: Whether "running" and "idle" are the only two states to count, or all four (running, idle, waiting, errored).
   - Recommendation: Derive from `m.sessions` — count all four status values using existing `statusGlyph` mappings. Show non-zero counts only to keep the line concise.

---

## Environment Availability

Step 2.6: SKIPPED — phase is purely Go code changes within the existing project, no external CLI tools, services, or runtimes beyond what already compiles.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing package (stdlib) |
| Config file | none — standard `go test` |
| Quick run command | `cd /Users/ken/dev/agenthub && go test ./internal/tui/... -count=1` |
| Full suite command | `cd /Users/ken/dev/agenthub && go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUI-01 | Session list renders inside bordered lipgloss frame with section header | unit | `go test ./internal/tui/... -run TestView_SessionFrame -count=1` | ❌ Wave 0 |
| TUI-01 | `injectBorderTitle` places title in frame border | unit | `go test ./internal/tui/... -run TestInjectBorderTitle -count=1` | ❌ Wave 0 |
| TUI-02 | Tab key toggles sidebar/content focus | unit | `go test ./internal/tui/... -run TestUpdate_TabFocusToggle -count=1` | ❌ Wave 0 |
| TUI-02 | `[`/`]` cycles through open tabs | unit | `go test ./internal/tui/... -run TestUpdate_TabCycle -count=1` | ❌ Wave 0 |
| TUI-02 | Sidebar Up/Down navigates sidebar items | unit | `go test ./internal/tui/... -run TestUpdate_SidebarNavigation -count=1` | ❌ Wave 0 |
| TUI-02 | View renders sidebar with all 4 section labels | unit | `go test ./internal/tui/... -run TestView_Sidebar -count=1` | ❌ Wave 0 |
| TUI-03 | Session row renders `[agent]` badge text | unit | `go test ./internal/tui/... -run TestView_AgentBadge -count=1` | ❌ Wave 0 |
| TUI-03 | Status glyphs use updated TokyoNight colors | unit | `go test ./internal/tui/... -run TestStatusGlyph -count=1` | ✅ view_test.go (extend) |
| TUI-04 | Dark palette uses TokyoNight hex values | unit | `go test ./internal/tui/... -run TestStyles_TokyoNight -count=1` | ❌ Wave 0 |
| TUI-04 | Per-agent badge color returns distinct color per agent | unit | `go test ./internal/tui/... -run TestAgentBadgeColor -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/tui/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/tui/view_test.go` — add `TestView_SessionFrame`, `TestView_Sidebar`, `TestView_AgentBadge`, `TestInjectBorderTitle`
- [ ] `internal/tui/update_test.go` — add `TestUpdate_TabFocusToggle`, `TestUpdate_TabCycle`, `TestUpdate_SidebarNavigation`
- [ ] `internal/tui/styles_test.go` — add `TestStyles_TokyoNight`, `TestAgentBadgeColor` (new file)

---

## Security Domain

Security enforcement is not applicable to this phase — all changes are pure TUI rendering and local model state. No network connections, no auth, no user-supplied input beyond keyboard navigation within an already-running session.

---

## Sources

### Primary (HIGH confidence)
- `internal/tui/view.go` — verified `injectBorderTitle`, `renderSessionRow`, `renderDividerRow`, `renderFull`, layout patterns
- `internal/tui/styles.go` — verified current color token structure and `LightDark` adaptive pattern
- `internal/tui/model.go` — verified Model struct fields, existing state for modals, remote sessions, unified list
- `internal/tui/update.go` — verified `handleKey` priority chain, `rebuildUnifiedList`, `attachCmd` flow
- `internal/tui/keys.go` — verified existing `KeyMap` bindings to avoid conflicts
- `internal/tui/help.go` — verified `RoundedBorder + injectBorderTitle` reuse pattern
- `internal/tui/modal.go` — verified modal overlay pattern
- `.planning/phases/86-tui-visual-polish/86-UI-SPEC.md` — authoritative visual contract: colors, layout, glyphs
- `.planning/phases/86-tui-visual-polish/86-CONTEXT.md` — locked decisions D-01 through D-15
- `frontend/src/style.css` — verified TokyoNight hex values: `#1a1b26`, `#7aa2f7`, `#565f89`, `#16161e`, `#292e42`, `#9ece6a`, `#e0af68`, `#f7768e`
- `go.mod` — verified library versions: bubbletea v2.0.5, lipgloss v2.0.3, bubbles v2.1.0
- `main.go`, `internal/daemon/process.go` — verified `daemon.BuildVersion` as version source

### Secondary (MEDIUM confidence)
- `internal/tui/view_test.go`, `update_test.go`, `integration_test.go` — verified existing test patterns and `testModel()` fixture for new tests

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in go.mod, versions verified
- Architecture: HIGH — all building blocks verified in existing source; pattern reuse is straightforward
- Pitfalls: HIGH — derived from direct code inspection of the exact files being modified
- Color palette: HIGH — verified against both UI-SPEC and frontend/src/style.css

**Research date:** 2026-04-19
**Valid until:** 2026-05-19 (stable libraries, in-repo implementation — not affected by upstream changes)
