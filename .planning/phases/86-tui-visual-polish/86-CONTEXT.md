# Phase 86: TUI Visual Polish - Context

**Gathered:** 2026-04-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Transform the TUI from a flat session list into a polished, tabbed interface with a vertical sidebar, bordered frames, styled session rows, per-agent colored badges, and a TokyoNight-derived color palette — matching the GUI's visual structure and navigation model.

</domain>

<decisions>
## Implementation Decisions

### Tab Navigation & Sidebar
- **D-01:** Vertical narrow sidebar on the left mirroring the GUI sidebar layout — short labels for Home, Sessions, Remote, Settings
- **D-02:** Arrow keys (Up/Down) move focus within the sidebar, Enter selects a section
- **D-03:** Tab key toggles focus between the sidebar pane and the content pane, with a clear visual indicator (accent border or highlight) showing which pane is active
- **D-04:** Sidebar items open as persistent tabs in a horizontal tab bar at the top of the content area — multiple tabs can be open simultaneously (Home, Settings, session terminals), matching the GUI's tabbed model
- **D-05:** `[` and `]` keys cycle through open tabs (previous/next)

### Session Attach Behavior
- **D-06:** Attaching to a session uses existing full-screen `tea.Exec` approach (suspends Bubble Tea, hands terminal to PTY). On detach (Ctrl-\), returns to TUI with all tab state preserved — the session's tab remains highlighted/open.
- **D-07:** Inline PTY rendering within tabs (embedding a terminal emulator in the Bubble Tea pane) is deferred to a future version.

### Session Framing
- **D-08:** Single outer bordered frame (rounded corners via lipgloss) around the entire session list, with the section name as a border title using `injectBorderTitle`
- **D-09:** Thin horizontal dividers between local and remote session groups inside the frame
- **D-10:** Column headers (NAME, AGENT, HOST, VIEWERS) render as the first row inside the frame, with a separator line below them

### Session Row Styling
- **D-11:** Enhanced column layout: keep current column structure but add colored agent badges (short tags like `[claude]`, `[codex]`), bolder status glyphs, and subtle background highlight on selected row
- **D-12:** Per-agent distinct badge colors — each CLI gets its own accent color for quick visual scanning (derived from agent brand or assigned from TokyoNight palette)

### Home Tab Content
- **D-13:** Combined branding + dashboard: AgentHub title, version, and tagline at the top; bordered stats section below with session counts (by status), web server status, Tailscale connection state, and quick-action hints
- **D-14:** Branding anchors the top of the view, live stats provide utility below a separator

### Color Palette (TokyoNight)
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### TUI codebase
- `internal/tui/view.go` — Current rendering: header, column headers, session list, footer, modal overlays, `injectBorderTitle` helper, `statusGlyph` mapping
- `internal/tui/styles.go` — Adaptive Styles struct with light/dark color tokens (needs TokyoNight alignment)
- `internal/tui/model.go` — Model struct, list entry types (local/remote/divider), modal states, message types
- `internal/tui/update.go` — Update loop, key handling, `tea.Exec` for attach
- `internal/tui/keys.go` — KeyMap with current keybindings (needs `[`, `]`, Tab additions)
- `internal/tui/help.go` — Help overlay with rounded border + `injectBorderTitle` pattern (reusable for frames)
- `internal/tui/modal.go` — New session and kill confirm modal patterns
- `internal/tui/tui.go` — Entry point, `newModel`, `Init` command batch
- `internal/tui/attach.go` — `attachCmd` implementing `tea.ExecCommand` for full-screen PTY attach

### GUI reference (visual targets)
- `frontend/src/components/Sidebar.tsx` — GUI sidebar: Home, Remote, Sessions, New Session, Settings
- `frontend/src/style.css` — TokyoNight colors: `#1a1b26` (bg), `#7aa2f7` (accent), `#565f89` (muted)

### Requirements
- `.planning/REQUIREMENTS.md` — TUI-01, TUI-02, TUI-03, TUI-04

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `injectBorderTitle()` in view.go — Splices styled title into lipgloss border top line; reuse for all section frames
- `lipgloss.RoundedBorder()` pattern from help overlay — consistent border style
- `statusGlyph()` in view.go — Status-to-glyph+color mapping; keep and enhance with bolder styling
- `renderDividerRow()` in view.go — Remote section divider with box-drawing chars; adapt for inline frame dividers
- `Styles` struct in styles.go — Adaptive light/dark tokens; extend with TokyoNight-specific values and per-agent colors
- `attachCmd` in attach.go — Full-screen PTY attach via `tea.Exec`; reuse as-is with tab state restoration on return

### Established Patterns
- Bubble Tea v2 with `tea.View` (alt-screen, no mouse)
- `lipgloss.LightDark(hasDark)` for adaptive colors via `tea.BackgroundColorMsg`
- Modal overlays rendered on top of content in `renderFull()`
- Unified list with `listEntry` kind discriminator (local/remote/divider)

### Integration Points
- `Model` struct needs: active tab tracking, sidebar state (focused item, collapsed), tab bar state (open tabs, active tab)
- `KeyMap` needs: `[`, `]` for tab cycling, `Tab` for focus toggle
- `renderFull()` needs restructuring: sidebar pane + content pane side-by-side, tab bar above content
- `handleKey()` needs focus-aware routing: sidebar keys vs content keys vs tab keys
- Footer (web status + hint bar) placement — below the content pane or spanning full width

</code_context>

<specifics>
## Specific Ideas

- Tab bar should show open tab names with the active tab visually distinguished (bold, underline, or accent color)
- When detaching from a session, the TUI should restore to the exact same tab that was active — no jarring reset to Home
- Sidebar should show the currently active section highlighted even when focus is on the content pane
- Per-agent colors should be visually distinct at a glance — avoid colors that are too similar in terminal rendering
- The Home tab branding should use the same tagline as the GUI splash screen: "AI coding terminal sessions"

</specifics>

<deferred>
## Deferred Ideas

- **Inline PTY tab rendering** — Embed a terminal emulator (VT100 state machine) inside Bubble Tea to render live PTY output in a content pane tab, enabling true tabbed terminal sessions without full-screen attach. Significant technical lift (raw mode conflict, sub-terminal rendering, per-tab virtual dimensions). Deferred to a future version.

</deferred>

---

*Phase: 86-tui-visual-polish*
*Context gathered: 2026-04-19*
