# Phase 86: TUI Visual Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-19
**Phase:** 86-tui-visual-polish
**Areas discussed:** Tab navigation layout, Session framing, Session row styling, Home tab content, Tab model (persistent vs section switcher)

---

## Tab Navigation Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Horizontal tab bar at top | Row of tab labels below header, Tab/Shift-Tab or number keys to switch | |
| Vertical sidebar (narrow) | Narrow left column with short labels mirroring GUI sidebar | ✓ |
| Bottom status-bar tabs | Tab labels in footer area like tmux window list | |

**User's choice:** Vertical sidebar (narrow)
**Notes:** User wants to mirror the GUI sidebar as closely as possible.

### Follow-up: Switching mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Number keys 1-4 | Press 1=Home, 2=Sessions, 3=Remote, 4=Settings | |
| Tab/Shift-Tab cycling | Tab cycles forward, Shift-Tab goes back | |
| Arrow keys + Enter | Up/Down move sidebar focus, Enter selects | ✓ |

**User's choice:** Arrow keys + Enter

### Follow-up: Focus model

| Option | Description | Selected |
|--------|-------------|----------|
| Tab key toggles focus | Tab switches between sidebar and content pane | ✓ |
| Left arrow returns to sidebar | Directional movement handles focus naturally | |
| You decide | Claude picks | |

**User's choice:** Tab key toggles focus

---

## Session Framing

| Option | Description | Selected |
|--------|-------------|----------|
| Sectioned frame with border title | One bordered frame per section with section name as border title | |
| Single outer frame with inline dividers | One big bordered frame with thin horizontal dividers between sections | ✓ |
| No outer frame, just section headers | Bold styled section headers with no border box | |

**User's choice:** Single outer frame with inline dividers

### Follow-up: Column header placement

| Option | Description | Selected |
|--------|-------------|----------|
| Inside frame, below border title | Column headers as first row inside frame with separator line | ✓ |
| Above the frame | Column headers float above the frame | |
| You decide | Claude picks | |

**User's choice:** Inside frame, below border title

---

## Session Row Styling

| Option | Description | Selected |
|--------|-------------|----------|
| Enhanced columns | Keep column layout, add colored agent badges, bold glyphs, selected highlight | ✓ |
| Card-like bordered rows | Each session in its own mini bordered box (3 rows per session) | |
| Current layout, just add color | Minimal change — only update colors to TokyoNight | |

**User's choice:** Enhanced columns

### Follow-up: Agent badge colors

| Option | Description | Selected |
|--------|-------------|----------|
| Per-agent colors | Each CLI gets a distinct accent color for its badge | ✓ |
| Uniform accent color | All badges use the same #7aa2f7 blue | |
| You decide | Claude picks | |

**User's choice:** Per-agent colors

---

## Home Tab Content

**User's choice:** Combined branding + dashboard (user clarified by combining two presented options)
**Notes:** User wanted both the dashboard overview (live session stats, web/Tailscale status, quick actions) AND the branding elements (logo/title, version, tagline). Layout: branding at top, stats below.

### Follow-up: Layout priority

| Option | Description | Selected |
|--------|-------------|----------|
| Branding top, stats below | Title/version/tagline at top, bordered stats section below | ✓ |
| Stats top, branding bottom | Live stats prominent, branding as footer | |
| You decide | Claude picks | |

**User's choice:** Branding top, stats below

---

## Tab Model (Persistent vs Section Switcher)

**User raised this area** asking whether the TUI should have tabs like the GUI.

| Option | Description | Selected |
|--------|-------------|----------|
| Persistent tabs (match GUI) | Sidebar opens items as tabs in a tab bar, multiple open at once | ✓ |
| Section switcher (simpler) | Sidebar just swaps content area, only one view at a time | |

**User's choice:** Persistent tabs (match GUI)

### Follow-up: Tab switching keys

| Option | Description | Selected |
|--------|-------------|----------|
| [ and ] keys cycle tabs | [ previous, ] next — common in multiplexers | ✓ |
| Number keys for first 9 tabs | 1-9 jump to specific tabs | |
| Ctrl+Left / Ctrl+Right | Arrow keys with Ctrl modifier | |

**User's choice:** [ and ] keys cycle tabs

### Follow-up: Session attach behavior

**User's choice:** Full-screen attach (keep current tea.Exec behavior), tab state preserved on return
**Notes:** User wanted inline PTY rendering but accepted the simpler approach after technical discussion of the hurdles (raw mode conflict, no sub-terminal rendering in Bubble Tea, virtual screen buffer requirement). Inline PTY tabs deferred to a future version.

---

## Claude's Discretion

- Sidebar width, per-agent badge color assignments, tab bar rendering style
- Separator line style, Settings/Remote tab content layout
- Focus indicator styling, quick-action hint formatting

## Deferred Ideas

- Inline PTY tab rendering (embed terminal emulator in Bubble Tea pane) — future version
