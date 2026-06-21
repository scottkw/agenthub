# Phase 142: Hub & Settings Redesign Polish — Research

**Researched:** 2026-06-21
**Domain:** React/TypeScript UI polish — xterm.js WebGL repaint, Hub IA restructure, card layout, settings controls
**Confidence:** HIGH (all findings from direct codebase inspection)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Move group navigation into the **main left sidebar** as an **expandable sub-list nested under "Hub"**. Selecting a group opens the Hub filtered to that group; the session grid spans full width. The `GroupSidebar` side-panel is removed.
- **D-02:** The comp does not depict a Groups concept at all — the nested-sidebar pattern is agreed; apply Direction 01 visual language to it.
- **D-03 (researcher flag):** Preserve `onDropOnGroup` → `assignToGroup`/`removeFromGroup` drag-to-assign behavior. With groups in the main sidebar, the preferred path is dropping a card onto a sidebar group item. If infeasible, fall back to a per-card "Add to group…" menu.
- **D-04:** Give mini-preview a **taller fixed height** (~6 lines), and **reserve a dedicated header gutter** so ⋮ and ☰ never overlap session name/status/preview at any width.
- **D-05:** **Harden the full repaint path** (theme/tab/font coordination), not a minimal patch. Confirm exact 141-08 regression source first.
- **D-06:** Light/Dark toggle is a **single switch** with a sun/moon icon + text label on/with the knob — colorblind-safe. Keep existing `uiTheme` persistence + `[data-ui-theme]` wiring.
- **D-07:** Both "New session" buttons (`HubFilterBar` top-right + `HubEmptyState`) restyled to match the comp's sidebar "+ New Session" affordance (`c-sessions.png` left sidebar). Existing `onClick` wiring unchanged.

### Claude's Discretion

- Exact preview pixel height and grid reflow breakpoints (bounded by D-04 "~6 lines" intent).
- Plan-splitting across the five POL items, and migration ordering.
- Exact CSS token / class names, following `--hub-*` convention and light-theme override discipline.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope. Formal regression-test program (TEST-01..05) is Phase 143.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| POL-01 | Hub card header icons (⋮ menu, ☰ handle) do not overlap other card elements at any width; in-card preview sized to be legible/useful | Card CSS at lines 5018–5089; `position: absolute` overlap mechanism confirmed; preview height 56px too short (≈4 lines at 11px/1.3 lh); gutter approach via `padding-top` |
| POL-02 | Settings Light/Dark control is a single toggle switch, retaining persistence and colorblind-safe state | `SettingsTab.tsx` ~438–461 shows current two-button pattern; `App.tsx` uiTheme wiring at ~279–294 is clean to reuse; `@heroicons/react` SunIcon/MoonIcon confirmed installed |
| POL-03 | Both Hub "New session" buttons styled to match comp's sidebar "+ New Session" affordance | `HubFilterBar.tsx` `hub-filter__new-session` CSS is a bordered surface button; `HubEmptyState.tsx` `.hub__empty-cta` is similar; comp shows a plain text link with `+` prefix |
| POL-04 | Terminal session repaints correctly (no garble) after theme or tab switch; 141-08 regression confirmed/fixed | Theme effect at lines 696–704 confirmed NO `isActive` guard and no `fitTerminal()` call; tab-switch display:none→flex gating at App.tsx ~1492 confirmed; full repaint-path hardening design documented below |
| POL-05 | Hub group navigation restructured out of secondary side-by-side panel; groups in main sidebar under Hub | `GroupSidebar.tsx` + `Sidebar.tsx` fully read; group data model (`lib/hubGroups.ts`) confirmed clean for in-place reuse; drag-to-sidebar feasibility assessed below |
</phase_requirements>

---

## Summary

Phase 142 is a pure-frontend polish phase on top of Phase 141's landed redesign. There are no new external packages, no backend changes, and no new routes. All five POL items are isolated within `frontend/src/` — CSS and TSX edits only.

**POL-04 (terminal repaint garble) is the highest-risk item.** The root cause is confirmed in `TerminalPanel.tsx`: the theme-change effect runs unconditionally on every panel, even hidden ones with zero cell dimensions. The fix must coordinate three sequences — theme change, tab visibility switch, and font-size change — through a single hardened repaint path. Verification requires the native `wails dev` window (the `:34115` bridge has no PTY).

**POL-05 (group navigation restructure) is the largest structural change.** `GroupSidebar.tsx` (339 lines) is deleted as a side-panel; its group-select and create-group logic is absorbed into `Sidebar.tsx` (81 lines currently). Group state (`groupDefs`, `activeGroupId`) lifts from `HubPanel.tsx` to `App.tsx`/`Sidebar.tsx` coordination. The `lib/hubGroups.ts` CRUD functions and localStorage format (`agenthub:hubGroups:v1`) are unchanged.

**POL-01, POL-02, POL-03 are lower-risk surgical changes** — CSS adjustments, a control replacement, and button restyle respectively.

**Primary recommendation:** Execute in dependency order: POL-04 first (highest-risk, can be verified independently), then POL-05 (structural, must not break activeGroupId filtering), then POL-01/POL-02/POL-03 (surgical).

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Terminal repaint coordination (POL-04) | Browser/Client (TerminalPanel.tsx) | — | xterm.js + WebGL addon are client-only; all repaint logic is in React effects |
| Group navigation IA (POL-05) | Browser/Client (Sidebar.tsx + App.tsx) | HubPanel.tsx (activeGroupId filter) | Group state must be shared between Sidebar (render) and HubPanel (filtering) — App.tsx is the natural owner |
| Card header gutter + preview height (POL-01) | Browser/Client (SessionCard.tsx + style.css) | — | Pure CSS layout fix |
| Light/Dark toggle control (POL-02) | Browser/Client (SettingsTab.tsx + style.css) | App.tsx (uiTheme state owner) | Control is in SettingsTab; state and DOM wiring already in App.tsx |
| New session button restyle (POL-03) | Browser/Client (HubFilterBar.tsx + HubEmptyState.tsx + style.css) | — | Pure CSS restyle; onClick wiring unchanged |

---

## Standard Stack

### Core (no new packages — all existing)

| Library | Installed Version | Purpose in Phase 142 |
|---------|------------------|----------------------|
| `@xterm/xterm` | pinned in go.mod vendor | Terminal emulator — repaint path fix (POL-04) |
| `@xterm/addon-fit` | pinned | FitAddon.proposeDimensions() / FitTerminal() — used in repaint path |
| `@xterm/addon-webgl` | pinned | clearTextureAtlas() — used in theme repaint |
| `@heroicons/react` | 24/outline, installed | SunIcon + MoonIcon for POL-02 toggle knob |
| React 18 | installed | Effects + useCallback for group state lift |

**No new npm installs required for this phase.** [VERIFIED: direct codebase inspection]

### Installation

```bash
# No installs needed — all dependencies already present
```

---

## Package Legitimacy Audit

> No new packages in this phase — audit not required.

**Packages installed by this phase:** none.

---

## Architecture Patterns

### System Architecture Diagram

```
App.tsx (state owner)
  │
  ├── uiTheme ──────────────────────────► SettingsTab.tsx (POL-02: replace control)
  │                                           └── LightDarkToggle (new single-switch)
  │
  ├── groupDefs ──────────────────────►  Sidebar.tsx (POL-05: add nested group sub-list)
  ├── activeGroupId ──┬──────────────►  Sidebar.tsx (group selected → highlight)
  │                   └──────────────►  HubPanel.tsx (visibleSessions filter unchanged)
  │
  └── HubPanel.tsx
        ├── HubFilterBar.tsx (POL-03: restyle new-session button)
        ├── HubEmptyState.tsx (POL-03: restyle empty-cta button)
        ├── SessionCardGrid → SessionCard.tsx (POL-01: gutter + preview height)
        │       └── MiniPreview.tsx (POL-01: taller height)
        └── [GroupSidebar REMOVED — POL-05]

Terminal tab (App.tsx):
  display:none/flex ──────────────────► TerminalPanel.tsx (POL-04: repaint path)
                                              ├── isActive effect (fit rAF loop)
                                              ├── theme effect (NO guard currently)
                                              └── fontSize effect (fitTerminal)
```

### Recommended Project Structure

No new files/folders needed. Changes are in-place edits to:

```
frontend/src/
├── components/
│   ├── Sidebar.tsx                  # POL-05: add expandable group sub-list
│   ├── SettingsTab.tsx              # POL-02: replace two-button with single toggle
│   ├── TerminalPanel.tsx            # POL-04: harden repaint path
│   ├── Hub/
│   │   ├── HubPanel.tsx             # POL-05: remove GroupSidebar; lift group state
│   │   ├── GroupSidebar.tsx         # POL-05: DELETE (group logic absorbed into Sidebar)
│   │   ├── HubFilterBar.tsx         # POL-03: restyle new-session button
│   │   ├── HubEmptyState.tsx        # POL-03: restyle empty-cta
│   │   ├── SessionCard.tsx          # POL-01: add padding-top gutter for icons
│   │   └── MiniPreview.tsx          # POL-01: height increase (no TSX change needed)
│   └── __tests__/
│       └── SettingsTab.appearance-theme.test.tsx  # POL-02: update to match new toggle
├── lib/
│   └── hubGroups.ts                 # UNCHANGED — CRUD and localStorage format stable
├── App.tsx                          # POL-05: lift groupDefs/activeGroupId state
└── style.css                        # POL-01/02/03/05: CSS changes throughout
```

---

## POL-04: Terminal Repaint — Root Cause Confirmed

### Confirmed suspects (all verified by direct file inspection)

**Suspect 1: Theme-change effect has no `isActive` guard** [CONFIRMED — HIGH confidence]

`TerminalPanel.tsx` lines 696–704:
```tsx
useEffect(() => {
  if (!termRef.current) return
  termRef.current.options.theme = theme      // runs on ALL panels, even display:none
  termRef.current.clearTextureAtlas()        // clears WebGL glyph cache
  termRef.current.refresh(0, termRef.current.rows - 1)  // refresh with zero-dim rows
}, [theme])
```

This effect fires on every mounted `TerminalPanel` whenever the theme prop changes, regardless of whether that panel is the active one. A panel at `display:none` has `rows = 0` (CharSizeService hasn't measured font on a hidden element), so `refresh(0, -1)` is a no-op or worse — it corrupts the WebGL atlas on a panel that hasn't had a fit run yet. When that panel later becomes visible, the stale/corrupt atlas produces garbled output.

**Suspect 2: No `fitTerminal()` call after theme repaint** [CONFIRMED — HIGH confidence]

The theme effect calls `clearTextureAtlas()` then `refresh()` but never calls `fitTerminal()`. If a font metric changed with the theme (even a minor subpixel rounding difference), the stale cell layout will misrender the first frame after the atlas rebuild.

**Suspect 3: Tab-switch ↔ theme-change race** [CONFIRMED — HIGH confidence]

The `isActive`-gated fit effect (lines 647–687) uses `requestAnimationFrame()` to wait for `display:none → flex` layout commit. If a theme change fires *during* this rAF window (e.g., user switches theme while tab-switching), the theme effect clears the atlas on a panel that is mid-fit — the fit rAF loop then calls `fitTerminal()` after the garbled atlas frame has already been drawn.

**Suspect 4: WebGL `clearTextureAtlas()` on `display:none → flex` transitions** [CONFIRMED — MEDIUM confidence]

App.tsx line ~1492: `style={{ display: isActive ? 'flex' : 'none' }}`. All TerminalPanel instances are always mounted; visibility is CSS-only. When a previously-hidden tab becomes active, the isActive fit effect fires. But if the theme-change effect *also* ran while the panel was hidden (clearing atlas on zero-dim panel), the first `fitTerminal()` call on tab-switch re-renders into an already-corrupted state.

### Hardened Repaint Path Design

The fix must enforce a strict ordering invariant:
> **A terminal panel must only rebuild its WebGL atlas when it is visible (isActive) AND has valid cell dimensions.**

**Recommended approach:**

```tsx
// Theme effect — add isActive guard and fit call
useEffect(() => {
  if (!termRef.current) return
  if (!isActive) {
    // Mark that a theme update is pending; apply when panel next becomes active
    pendingThemeRef.current = theme
    return
  }
  applyThemeAndRefit(termRef.current, theme, fitAddonRef.current)
}, [theme, isActive])

// isActive effect — apply pending theme before fitting
useEffect(() => {
  if (!isActive || !containerRef.current) return
  // Apply any theme that arrived while this panel was hidden
  if (pendingThemeRef.current) {
    applyThemeAndRefit(termRef.current!, pendingThemeRef.current, fitAddonRef.current)
    pendingThemeRef.current = null
  }
  // ... existing rAF fit loop ...
}, [isActive])

function applyThemeAndRefit(term: Terminal, theme: ITheme, fitAddon: FitAddon | null) {
  term.options.theme = theme
  term.clearTextureAtlas()
  if (fitAddon) fitTerminal(term)  // refit after atlas clear, not just refresh
}
```

Key changes:
1. Add `pendingThemeRef = useRef<ITheme | null>(null)` for deferred theme application
2. Theme effect checks `isActive`; if not active, stores theme in `pendingThemeRef` and returns
3. isActive effect drains `pendingThemeRef` before starting the fit rAF loop
4. `applyThemeAndRefit()` helper calls `fitTerminal()` after `clearTextureAtlas()` — not `refresh(0, rows-1)` — ensuring cell dims are recalculated after the atlas rebuild
5. Font-size effect (lines 690–694) already calls `fitTerminal()` — no change needed

**Verification:** Must run in native `wails dev` window — the `:34115` bridge has no PTY. Steps: open session, switch theme, verify no garble; switch away and back, verify; switch theme while on a different tab, switch back, verify.

---

## POL-05: Group Navigation IA — Feasibility Analysis

### Current state (confirmed by direct inspection)

- `GroupSidebar.tsx` (339 lines) is a self-contained `<aside>` rendered as the left column of `hub__body` flex row
- `HubPanel.tsx` owns `groupDefs`, `activeGroupId`, and `sidebarCollapsed` state
- `Sidebar.tsx` (81 lines) is a simple nav with 3 buttons (Home, Hub, Settings) — no group awareness
- Group state is not visible to `App.tsx` — it lives entirely inside `HubPanel.tsx`

### What moves where

| Currently in | Moves to | Notes |
|---|---|---|
| `HubPanel.tsx` `groupDefs` state | App.tsx (or Sidebar.tsx via prop) | Must be shared so Sidebar can render the sub-list and HubPanel can filter |
| `HubPanel.tsx` `activeGroupId` state | App.tsx | Filtering of `visibleSessions` in HubPanel needs this |
| `HubPanel.tsx` `handleCreateGroup` | App.tsx or Sidebar.tsx | Inline input for "New group" |
| `GroupSidebar.tsx` counts computation (`computeCounts`, `computeGlobalCounts`) | New file `lib/hubGroupCounts.ts` or inline in Sidebar | Counts need `allSessions` which is in HubPanel — must be passed down |
| `GroupSidebar.tsx` drag-drop handlers | Sidebar.tsx group sub-list items | See drag feasibility below |

### State lift design

App.tsx needs to own `groupDefs` and `activeGroupId` so it can pass both down to Sidebar (for rendering) and HubPanel (for filtering):

```tsx
// App.tsx additions
const [groupDefs, setGroupDefs] = useState<HubGroupDef[]>(() => loadGroups())
const [activeGroupId, setActiveGroupId] = useState<string | null>(null)

// Pass to Sidebar:
<Sidebar
  groupDefs={groupDefs}
  activeGroupId={activeGroupId}
  onGroupSelect={setActiveGroupId}
  onCreateGroup={(name) => setGroupDefs(createGroup(groupDefs, name))}
  // counts need sessions — pass allSessions from HubPanel or compute here
  // ...
/>

// Pass to HubPanel (activeGroupId already used there for filtering):
<HubPanel
  activeGroupId={activeGroupId}
  // groupDefs still needed for per-card "Move to group" menu (GROUP-02)
  groupDefs={groupDefs}
  onDropOnGroup={(groupId, key) => setGroupDefs(assignToGroup(groupDefs, groupId, key))}
  // ...
/>
```

**Count computation challenge:** Running/total/attention counts require iterating `allSessions` (local + remote). `allSessions` is computed inside `HubPanel.tsx`. Two options:
- Pass `allSessions` up to App.tsx (requires lifting session poll data, invasive)
- Pass a `counts` prop down from HubPanel to App.tsx/Sidebar (lighter — one callback or computed memo)

Recommendation: Add `onGroupCountsChange: (counts: Record<string|'__all__', GroupCounts>) => void` callback on HubPanel — HubPanel computes counts and passes them up; Sidebar receives them via App.tsx. This is the minimal-lift path and avoids moving session polling logic.

### Drag-to-sidebar feasibility

**Verdict: Feasible but requires drop zone expansion.** [ASSESSED — MEDIUM confidence]

Current drag source: `SessionCard.tsx` sets `e.dataTransfer.setData('text/plain', memberKeyForSession)` on `draggable="true"` article.

Current drop target: `GroupSidebarItem` in `GroupSidebar.tsx` handles `onDragOver`/`onDrop` on `<li>`.

After POL-05, the drop targets move to sidebar group sub-list items in `Sidebar.tsx`. The HTML5 drag API works across any DOM elements, so dropping onto a sidebar `<li>` or `<button>` is technically identical to dropping onto the current panel items.

**Potential UX issue:** The main sidebar is 200px wide (collapsed: 48px). Dragging a card from the main grid area (right side of screen) across to the sidebar (far left) is a long drag distance. However, this is the same UX that the old `GroupSidebar` side-panel required (left column of hub__body). The nested sidebar is at the same horizontal position, so distance is equivalent.

**Collapsed sidebar problem:** When the sidebar is collapsed (48px), group sub-items are not visible. The drag must trigger the sidebar to expand, or the group sub-list must remain accessible in collapsed state (collapsed icon-only mode similar to GroupSidebar's collapsed icon state).

**Recommendation:** Implement drag-to-sidebar as the primary path. Add auto-expand on drag-enter the sidebar (set `collapsed = false` during drag, restore on drag-leave). The per-card "Add to group…" menu (already in `hub-card__menu` via `handleAssign`) is retained as the keyboard/accessible fallback — it already exists and works.

### GroupSidebar sub-list ARIA contract (inherited from CARRY-01 / 141)

The same `aria-pressed` pattern from the CARRY-01 fix applies directly to the new sidebar sub-list items:
```tsx
<button
  type="button"
  aria-pressed={activeGroupId === g.id}
  aria-label={`${g.name} group, ${counts.running}/${counts.total} sessions`}
  onClick={() => onGroupSelect(g.id)}
>
```

The `hub-group-sidebar-heading` + `aria-labelledby` pattern can be simplified since the items are now children of a nested `<ul>` under the "Hub" `<li>`.

---

## POL-01: Card Header Gutter + Preview Height

### Current state (confirmed)

**Icons:** `hub-card__drag-handle` and `hub-card__menu-btn` are both `position: absolute` with `top: 8px` — drag-handle at `left: 8px`, menu-btn at `right: 8px`. The card body starts at `padding: 12px 16px`. When the drag handle and menu button appear (on hover/focus-within), they overlay the card's top row content (ROW 1: status + name + CLI badge).

**Fix:** Reserve top padding for the icon row. Add `padding-top: 32px` to `.hub-card` (or introduce a dedicated header gutter element). The absolute-positioned icons sit in this gutter. Alternatively, change the icons from `position: absolute` to part of a fixed-height header row.

Recommended approach — fixed header row:
```css
/* Replace current absolute approach */
.hub-card {
  padding: 0;  /* reset — let header + body handle own padding */
}

.hub-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 32px;       /* dedicated gutter */
  padding: 0 8px;
  opacity: 0;         /* hidden when not hover/focus */
}

.hub-card:hover .hub-card__header,
.hub-card:focus-within .hub-card__header {
  opacity: 1;
}
```

Or simpler: keep `position: absolute` but ensure card has enough `padding-top` to prevent overlap:
```css
.hub-card {
  padding: 36px 16px 12px;  /* 36px top = 8px icon-top + 20px icon-height + 8px gap */
}
```

The simpler `padding-top` approach avoids TSX changes. The planner can choose.

**Preview height:** Current `.hub-card__preview` height is `56px`. Line height is `11px * 1.3 = ~14.3px` per line. `56px / 14.3px ≈ 3.9 lines`. For 6 lines: `6 * 14.3px = 85.8px → 88px` (nearest 4px multiple). At 7 lines: `7 * 14.3px = 100px` — may feel too tall for grid density.

**Recommendation:** `height: 88px` (≈6 lines). Planner confirms against grid density. No TSX changes to `MiniPreview.tsx` needed — height is CSS-only.

**Narrow-width reflow:** Card has `min-width: 240px`, `max-width: 360px`. At 240px, the header row has 240px - 16px = 224px available. With the dedicated 32px-tall gutter, icons are always in their own row above content — no overlap possible at any grid column count.

---

## POL-02: Light/Dark Toggle — Current Control + Replacement

### Current control (confirmed)

`SettingsTab.tsx` lines 438–461: Two `<button>` elements in a `role="group"` div, each with `aria-pressed`. The existing `uiTheme` state, `handleUiThemeChange` callback, `localStorage` persistence, and `document.documentElement.setAttribute` effect are all in `App.tsx` and are correct — they do not change.

**Existing tests** (`SettingsTab.appearance-theme.test.tsx`) assert the `aria-pressed` interface, button text ("Light"/"Dark"), and `onUiThemeChange` calls. These tests will need updating when the control becomes a toggle switch.

### Replacement design (D-06)

Replace the two-button group with a single `<button role="switch">` or styled `<div>` + hidden checkbox. Colorblind-safe: state must not rely on knob position or color alone — icon + text label on the knob.

```tsx
// Sun = light mode; Moon = dark mode
// The knob shows the CURRENT state icon + text label
import { SunIcon, MoonIcon } from '@heroicons/react/24/outline'

<div
  className={`settings-panel__theme-toggle${uiTheme === 'light' ? ' settings-panel__theme-toggle--light' : ''}`}
  role="switch"
  aria-checked={uiTheme === 'light'}
  aria-label="Light mode"
  tabIndex={0}
  onClick={() => onUiThemeChange(uiTheme === 'light' ? 'dark' : 'light')}
  onKeyDown={(e) => {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      onUiThemeChange(uiTheme === 'light' ? 'dark' : 'light')
    }
  }}
>
  <span className="settings-panel__theme-toggle-track">
    <span className="settings-panel__theme-toggle-knob">
      {uiTheme === 'light'
        ? <><SunIcon aria-hidden="true" /><span>Light</span></>
        : <><MoonIcon aria-hidden="true" /><span>Dark</span></>
      }
    </span>
  </span>
</div>
```

`role="switch"` + `aria-checked` is the correct ARIA pattern for a two-state toggle (on/off). The icon shape + text label satisfies the colorblind-safe contract (D-06). No position or color alone carries the state.

**SunIcon / MoonIcon:** Both are in `@heroicons/react/24/outline` which is already installed. [VERIFIED: `import` of other heroicons icons confirmed in SessionCard.tsx]

**CSS pattern (light-theme override discipline):**
```css
/* dark: knob on left side (dark mode default) */
.settings-panel__theme-toggle-track { ... }
.settings-panel__theme-toggle-knob { ... }

/* light: knob on right side */
[data-ui-theme="light"] .settings-panel__theme-toggle-track { ... }
```

The knob slide animation follows the existing motion contract — wrap in `@media (prefers-reduced-motion: no-preference)`.

**Existing test file must be updated** to assert `role="switch"` / `aria-checked` instead of `button[aria-pressed]` with text "Light"/"Dark".

---

## POL-03: "New Session" Button Restyle

### Current state (confirmed)

**HubFilterBar:** `.hub-filter__new-session` is a bordered surface button (height 28px, `border: 1px solid var(--hub-border)`, `background: var(--hub-surface-elevated)`, `color: var(--hub-text-secondary)`).

**HubEmptyState:** `.hub__empty-cta` is a similar bordered button (`border: 1px solid var(--hub-border)`, `color: var(--hub-text-primary)`, `padding: 6px 16px`).

### Comp target (c-sessions.png — confirmed by visual inspection)

The comp's sidebar "+ New Session" item is a **plain text link affordance** — no background, no border. It has a `+` prefix, uses the sidebar text color, and is indented at the same level as "Sessions". It's understated — not a CTA button.

### Restyle approach

Both buttons should become the same visual affordance: minimal weight, accent-colored `+` prefix, text label, no filled background. This matches the comp's sidebar aesthetic while working in the Hub context:

```css
/* FilterBar new-session — restyle to comp affordance */
.hub-filter__new-session {
  background: transparent;
  border: none;
  color: var(--hub-text-secondary);
  font-size: 12px;
  padding: 0 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  margin-left: auto;
}

.hub-filter__new-session::before {
  content: '+';
  color: var(--hub-accent);
  font-size: 16px;
  line-height: 1;
}

.hub-filter__new-session:hover {
  color: var(--hub-text-primary);
}
```

Or: add a `+` icon from heroicons (`PlusIcon`) into the TSX instead of CSS `::before`. TSX approach is cleaner (no pseudo-element) and keeps the accent glyph screen-reader-invisible via `aria-hidden`.

The `hub__empty-cta` gets the same visual treatment but centered (empty state is centered). Both light-theme overrides needed for `color` values.

**TSX changes needed:** Both `HubFilterBar.tsx` and `HubEmptyState.tsx` need a `PlusIcon` added from `@heroicons/react/24/outline` (already installed). Existing `onClick` wiring is unchanged.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Light/Dark toggle switch | Custom JS state machine + CSS | React `role="switch"` + CSS | ARIA switch pattern is 2 attributes; the toggle is purely stateful CSS |
| WebGL atlas management | Custom atlas rebuild logic | `clearTextureAtlas()` from `@xterm/addon-webgl` | Already in codebase; just needs correct guard |
| Group counts computation | New service/hook | Inline `computeCounts()` from GroupSidebar.tsx (copy to new location) | It's 20 lines of pure iteration — no library needed |
| Icon for toggle knob | Custom SVG | `@heroicons/react` SunIcon / MoonIcon | Already installed, already used elsewhere |

---

## Common Pitfalls

### Pitfall 1: Theme effect running on inactive panels
**What goes wrong:** `clearTextureAtlas()` + `refresh()` on a `display:none` panel corrupts the WebGL state for when the panel next becomes visible.
**Why it happens:** The effect has no `isActive` guard — it fires on every mounted panel.
**How to avoid:** Add `if (!isActive) { pendingThemeRef.current = theme; return }` at the top of the theme effect.
**Warning signs:** After switching themes with multiple tabs open, the non-active terminals show garbled characters when switched to.

### Pitfall 2: fitTerminal() not called after clearTextureAtlas()
**What goes wrong:** After theme change, cell dimensions may have changed subtly (subpixel rounding). `refresh()` redraws with the old layout.
**Why it happens:** The original code calls `refresh()` not `fitTerminal()` after clearing the atlas.
**How to avoid:** Replace `termRef.current.refresh(0, termRef.current.rows - 1)` with `fitTerminal(termRef.current)` in the hardened theme path.

### Pitfall 3: Race between isActive rAF loop and theme pending drain
**What goes wrong:** The isActive effect drains `pendingThemeRef` but the rAF loop starts before the drain; first frame uses old theme.
**How to avoid:** Drain `pendingThemeRef` synchronously at the start of the isActive effect, *before* starting the rAF loop.

### Pitfall 4: group state lift breaking HubPanel card "Move to group" menu
**What goes wrong:** `groupDefs` and `onDropOnGroup` are still needed inside HubPanel for the per-card overflow menu (GROUP-02). If they are removed from HubPanel props rather than passed down, the "Move to group" menu breaks.
**How to avoid:** Keep `groupDefs` and `onDropOnGroup` as HubPanel props even after POL-05 — they now come from App.tsx instead of being owned by HubPanel.

### Pitfall 5: Missing `[data-ui-theme="light"]` counterpart for new CSS
**What goes wrong:** New toggle knob CSS, new button affordance CSS, new gutter CSS — only dark-theme rules added. Light theme falls back to incorrect styles.
**How to avoid:** Every new CSS rule that touches color or background must have a corresponding `[data-ui-theme="light"]` override. Follow existing `style.css` pattern (dark in `:root`, light in `[data-ui-theme="light"]` blocks around lines 4110+).

### Pitfall 6: activeGroupId not reset when Hub is unfocused
**What goes wrong:** User selects a group in the sidebar, switches to Home, returns to Hub — group filter is still active but the sidebar may not visually indicate it.
**How to avoid:** The sidebar always reflects `activeGroupId` from App.tsx state. No reset needed — persistent group filter is correct UX. Just ensure visual active state is rendered.

### Pitfall 7: SettingsTab.appearance-theme tests fail after POL-02
**What goes wrong:** Existing tests assert `button[aria-pressed]` with text "Light"/"Dark". After replacing with `role="switch"`, those DOM queries return nothing.
**How to avoid:** Update `SettingsTab.appearance-theme.test.tsx` as part of the POL-02 task to assert `[role="switch"][aria-checked]` instead.

### Pitfall 8: Drag source must still be readable after GroupSidebar is removed
**What goes wrong:** `GroupSidebar.tsx` currently reads `e.dataTransfer.getData('text/plain')` as the memberKey. If the file is deleted and the new drop target in Sidebar.tsx doesn't implement the same `getData` call, drops silently fail.
**How to avoid:** Copy the `onDragOver`/`onDragLeave`/`onDrop` handlers from `GroupSidebarItem` into the new Sidebar group sub-list items. The drag data format (`memberKey` string) is unchanged.

---

## Code Examples

### Pattern 1: Hardened theme effect (POL-04)

```tsx
// Source: direct inspection of TerminalPanel.tsx + xterm.js best practices [ASSUMED]
const pendingThemeRef = useRef<ITheme | null>(null)

// Theme effect — defers application on hidden panels
useEffect(() => {
  if (!termRef.current) return
  if (!isActive) {
    pendingThemeRef.current = theme
    return
  }
  termRef.current.options.theme = theme
  termRef.current.clearTextureAtlas()
  fitTerminal(termRef.current)
}, [theme, isActive])

// isActive effect — drain pending theme before fit loop
useEffect(() => {
  if (!isActive || !containerRef.current) return
  if (pendingThemeRef.current && termRef.current) {
    termRef.current.options.theme = pendingThemeRef.current
    termRef.current.clearTextureAtlas()
    pendingThemeRef.current = null
    // fitTerminal called by the rAF loop below
  }
  // ... existing rAF fit loop unchanged ...
}, [isActive])
```

### Pattern 2: Sidebar group sub-list (POL-05)

```tsx
// In Sidebar.tsx — expandable sub-list under Hub button
const [groupsExpanded, setGroupsExpanded] = useState(true)

// Hub button becomes expandable item when groups exist
{!collapsed && groupDefs.length > 0 && (
  <ul className="sidebar__group-list" aria-label="Hub groups">
    <li>
      <button
        className={`sidebar__group-item${activeGroupId === null ? ' sidebar__group-item--active' : ''}`}
        type="button"
        aria-pressed={activeGroupId === null}
        onClick={() => { onGroupSelect(null); onOpenHub() }}
      >
        All
        <span className="sidebar__group-count">{globalCounts.running}/{globalCounts.total}</span>
      </button>
    </li>
    {groupDefs.map(g => (
      <li key={g.id}>
        <button
          className={`sidebar__group-item${activeGroupId === g.id ? ' sidebar__group-item--active' : ''}`}
          type="button"
          aria-pressed={activeGroupId === g.id}
          aria-label={`${g.name} group, ${counts[g.id]?.running ?? 0}/${counts[g.id]?.total ?? 0} sessions`}
          onDragOver={(e) => { e.preventDefault(); setDragOverGroupId(g.id) }}
          onDragLeave={() => setDragOverGroupId(null)}
          onDrop={(e) => {
            e.preventDefault()
            const key = e.dataTransfer.getData('text/plain')
            if (key) onDropOnGroup(g.id, key)
            setDragOverGroupId(null)
          }}
          onClick={() => { onGroupSelect(g.id); onOpenHub() }}
        >
          {g.name}
        </button>
      </li>
    ))}
  </ul>
)}
```

### Pattern 3: Single toggle switch (POL-02)

```tsx
// Source: ARIA authoring practices (role="switch") + CONTEXT.md D-06 [ASSUMED for ARIA, VERIFIED for D-06]
import { SunIcon, MoonIcon } from '@heroicons/react/24/outline'

<label className="settings-panel__theme-toggle-label">Interface Theme</label>
<button
  type="button"
  role="switch"
  aria-checked={uiTheme === 'light'}
  aria-label={uiTheme === 'light' ? 'Light mode — click to switch to dark' : 'Dark mode — click to switch to light'}
  className={`settings-panel__theme-toggle${uiTheme === 'light' ? ' settings-panel__theme-toggle--light' : ''}`}
  onClick={() => onUiThemeChange(uiTheme === 'light' ? 'dark' : 'light')}
>
  <span className="settings-panel__theme-toggle-track">
    <span className="settings-panel__theme-toggle-knob" aria-hidden="true">
      {uiTheme === 'light'
        ? <><SunIcon /><span>Light</span></>
        : <><MoonIcon /><span>Dark</span></>
      }
    </span>
  </span>
</button>
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|---|---|---|
| `refresh(0, rows-1)` after theme change | `fitTerminal()` after atlas clear | Ensures cell dimensions are recalculated — prevents layout stale |
| Two segmented buttons for Light/Dark | Single `role="switch"` toggle | ARIA switch is the correct semantic; knob provides colorblind-safe state |
| GroupSidebar as separate panel | Nested sub-list in main sidebar | Eliminates two-panel layout; reduces navigation surface from 2 panels to 1 |

**Deprecated in this phase:**
- `GroupSidebar.tsx`: entire file removed
- `hub__body` flex-row layout: grid becomes single column (`hub__grid-scroll` spans full width)
- `hub__group-sidebar*` CSS rules: removed or repurposed (may keep `hub__group-sidebar-item` pattern renamed to `sidebar__group-item`)

---

## Environment Availability

> This phase is purely frontend code/CSS — no external services, CLIs, or runtimes beyond the project's existing build chain.

| Dependency | Required By | Available | Version | Fallback |
|-----------|-------------|-----------|---------|----------|
| Node.js + pnpm | `pnpm test` / `pnpm build` | Confirmed (project builds) | — | — |
| `wails dev` native window | POL-04 terminal UAT (PTY required) | Confirmed (wails installed) | — | No fallback — PTY-only |
| Playwright / browser bridge | POL-02/03 visual UAT | `:34115` bridge available | — | headless for non-PTY surfaces |

**Blocking:** POL-04 terminal garble verification requires the native `wails dev` window — the `:34115` bridge has no PTY. This is not a blocker to implementation, only to verification.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest (jsdom environment) |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `cd frontend && pnpm test --reporter=verbose 2>&1 \| tail -20` |
| Full suite command | `cd frontend && pnpm test` |
| Type gate | `cd frontend && pnpm build` (runs `tsc && vite build`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| POL-01 | Card icon/preview CSS — no TSX logic change | manual (visual, wails dev) | n/a | n/a — CSS-only |
| POL-01 | Preview height CSS token | source-inspection | grep `hub-card__preview` style.css | ✅ style.css |
| POL-02 | Toggle renders as `role="switch"` with `aria-checked` | unit | `pnpm test --reporter=verbose SettingsTab.appearance-theme` | ✅ needs update |
| POL-02 | Toggle calls `onUiThemeChange` on click | unit | same test file | ✅ needs update |
| POL-03 | New session buttons have accent `+` prefix | source-inspection | grep `PlusIcon` HubFilterBar HubEmptyState | ❌ Wave 0 |
| POL-04 | Theme effect has `isActive` guard | source-inspection | grep `pendingThemeRef` TerminalPanel | ❌ Wave 0 |
| POL-04 | Theme change does not garble active terminal | manual (wails dev, PTY required) | n/a | n/a |
| POL-05 | Group sub-list renders in Sidebar | unit | `pnpm test Sidebar` or `pnpm test HubPanel` | ❌ Wave 0 |
| POL-05 | Drop on sidebar group item calls onDropOnGroup | unit | `pnpm test Sidebar` | ❌ Wave 0 |
| POL-05 | `hub__body` no longer renders GroupSidebar | unit | `pnpm test HubPanel` — assert `.hub__group-sidebar` absent | ✅ HubPanel.test.tsx update |

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm test 2>&1 | tail -5`
- **Per wave merge:** `cd frontend && pnpm build` (tsc + vite — catches type errors vitest misses)
- **Phase gate:** Full suite green + `pnpm build` clean before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/components/Sidebar.test.tsx` — covers POL-05 group sub-list render, group select, drag-drop
- [ ] `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx` — update assertions from `button[aria-pressed]` to `[role="switch"][aria-checked]`
- [ ] `frontend/src/components/Hub/HubPanel.test.tsx` — update to assert GroupSidebar is absent (`hub__group-sidebar` not in DOM)

*(Existing `GroupSidebar.test.tsx` is deleted when `GroupSidebar.tsx` is deleted.)*

---

## Security Domain

> This phase makes no authentication, session, network, or cryptographic changes. All changes are UI layout and CSS only. ASVS categories V2, V3, V4, V6 do not apply.

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | partial | Group name input (inline `<input>` in new Sidebar group creator) — existing `inputValue.trim()` guard retained; no server round-trip |
| V6 Cryptography | no | — |

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| Group name input (client-side) | Tampering | `inputValue.trim()` + localStorage — client-only, no server |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `pendingThemeRef` approach is compatible with xterm.js WebGL addon — deferring `clearTextureAtlas()` to activation does not cause intermediate rendering issues | POL-04 repaint design | Low — worst case is a brief stale-palette frame on activation, which is invisible to user |
| A2 | `fitTerminal()` after `clearTextureAtlas()` is safe (not double-fit) — xterm.js processes them as separate operations | POL-04 code example | Low — the existing font-size effect already calls `fitTerminal()` after `options.fontSize` write with no issues |
| A3 | Drag-from-card-to-main-sidebar works cross-DOM with the HTML5 dataTransfer API (no shadow DOM boundary in Wails webview) | POL-05 drag feasibility | Medium — if Wails webview has restrictions on cross-element drag, must fall back to per-card menu only |
| A4 | `role="switch"` ARIA semantics are supported in the Wails WebView (based on Chromium) | POL-02 toggle control | Low — Chromium supports ARIA switch; Wails desktop uses system WebView (macOS = WKWebView, also supports it) |

---

## Open Questions

1. **Group count data path**
   - What we know: `allSessions` (local + remote) is computed inside `HubPanel.tsx`; Sidebar needs counts to display running/total per group
   - What's unclear: Cleanest lift path — pass allSessions up to App.tsx or pass computed counts from HubPanel via callback
   - Recommendation: Computed-counts callback is lighter; planner to decide based on whether HubPanel already has a suitable prop slot

2. **Collapsed sidebar + group items**
   - What we know: The main sidebar collapses to 48px (icons only); group sub-list items are text-heavy
   - What's unclear: Should collapsed sidebar hide all group items (sub-list collapsed), or show icon-only group items with tooltips?
   - Recommendation: Hide group sub-list when sidebar is collapsed (matches existing behavior — GroupSidebar's icon-only state was a separate column anyway); add group-count attention badge on Hub icon when collapsed + a group is filtering

3. **`GroupSidebar.test.tsx` disposition**
   - What we know: The file tests the GroupSidebar component which is being deleted
   - What's unclear: Whether to delete the test file or migrate tests to Sidebar.test.tsx
   - Recommendation: Migrate drag-drop and count tests to new `Sidebar.test.tsx`; delete `GroupSidebar.test.tsx`

---

## Sources

### Primary (HIGH confidence — direct codebase inspection)

- `frontend/src/components/TerminalPanel.tsx` lines 647–704 — isActive fit effect + theme effect, confirmed no guard and no fitTerminal call
- `frontend/src/App.tsx` lines 89, 279–294, 1102–1110, 1480–1519 — HUB_TAB, uiTheme wiring, handleOpenHub, terminal tab display:none/flex gating
- `frontend/src/components/Hub/HubPanel.tsx` lines 259–286, 509–519 — group state, hub__body flex layout with GroupSidebar
- `frontend/src/components/Hub/GroupSidebar.tsx` — full file, 339 lines — group item ARIA, drag handlers, count computation
- `frontend/src/components/Sidebar.tsx` — full file, 81 lines — current 3-button nav, localStorage key
- `frontend/src/components/Hub/SessionCard.tsx` lines 338–554 — drag handle / menu-btn overlay, MiniPreview render
- `frontend/src/style.css` lines 5018–5089 — absolute positioning of drag/menu icons, preview height 56px
- `frontend/src/style.css` lines 4533–4641 — hub-filter__new-session, hub__empty-cta current styles
- `frontend/src/components/SettingsTab.tsx` lines 438–461 — current two-button Light/Dark control
- `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx` — existing test structure confirmed
- `frontend/src/lib/hubGroups.ts` — full file, CRUD + localStorage format
- `agenthub-v4.0-redesign/AgentHub UI redesign/c-sessions.png` — comp sidebar "+ New Session" affordance (visual reference, inspected)
- `agenthub-v4.0-redesign/AgentHub UI redesign/screenshots/141-09/02-hub-dark.png` — current hub with side-by-side GROUPS panel (before state confirmed)

### Secondary (MEDIUM confidence)

- `.planning/phases/141-redesign-implementation/141-UI-SPEC.md` — Direction 01 lock, token system, colorblind contract
- `.planning/phases/141-redesign-implementation/141-RENDER-COMPARE.md` — UAT findings source, headless limitation documented
- `.planning/phases/142-hub-settings-redesign-polish/142-CONTEXT.md` — locked decisions D-01..D-07

### Tertiary (LOW confidence / ASSUMED)

- xterm.js best practice for `clearTextureAtlas()` + `fitTerminal()` ordering — derived from code behavior, not official docs [ASSUMED]
- ARIA `role="switch"` + `aria-checked` semantics for WKWebView (macOS Wails) — standard Chromium-based support assumed [ASSUMED]

---

## Metadata

**Confidence breakdown:**
- POL-04 root cause: HIGH — confirmed by direct line inspection of TerminalPanel.tsx
- POL-05 architecture: HIGH — confirmed by full read of all five relevant files
- POL-01 layout fix: HIGH — CSS values confirmed by inspection; pixel math is deterministic
- POL-02 control: HIGH — existing control and wiring fully inspected; new control pattern is standard ARIA
- POL-03 restyle: HIGH — both button files fully inspected; comp target visually confirmed
- Repaint fix correctness: MEDIUM — logic is sound but xterm.js internals not verified via official docs

**Research date:** 2026-06-21
**Valid until:** 2026-07-21 (stable codebase; no fast-moving external deps in this phase)
