# Phase 141: Redesign Implementation — Research

**Researched:** 2026-06-20
**Domain:** CSS token migration, ARIA refactor, copy fix (frontend-only)
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 Hub scope:** Phase 141 Hub work is ARIA + copy only. Hub already
  consumes `--hub-*` tokens; no recolor is required. Do not touch Hub card
  layout or visuals.
- **D-02 Editor chrome:** Editor (CodeMirror 6) chrome — breadcrumb, save
  controls, preview header — is restyled by matching the File Browser surface
  (same `--hub-*` token migration). CodeMirror's internal theme (`--cm-*`) is
  out of scope.
- **D-13 Recolor-only:** Phase 141 applies the Refined Native visual language
  (color only) to existing component structures. No layout or interaction
  changes to any surface.
- **D-08 / D-09 / D-10:** Sessions page, Remote page, and `+ New Session`
  sidebar item must NOT be reintroduced.
- **D-11 Copy fix:** All "Sessions tab" copy strings must be replaced or removed.
- **Accent locked:** Dark `--hub-accent: #7aa2f7`; light `--hub-accent: #3d6fe8`.
  The standalone HTML's `#7C8CFF` is REJECTED.
- **CARRY-01 resolution chosen:** Drop `listbox`/`option` roles from GroupSidebar;
  use plain `<ul>` + `<li>` + `<button type="button" aria-pressed>`. Full ARIA
  contract is in 141-UI-SPEC.md §CARRY-01.
- **D-03 migration boundaries:** Agent badge colors (`.tab__agent-badge--*`) and
  semantic status colors (Running/Idle/Waiting/Errored/Stopped) must NOT be
  migrated to theme tokens.

### Claude's Discretion

- Per-surface migration ordering and plan-splitting are the planner's call.
- Exact new `--hub-*` token names (where genuinely needed) follow the planner's
  judgement within existing naming convention.

### Deferred Ideas (OUT OF SCOPE)

- None — all deferred items remain from the Phase 140 triage; no new items
  were pulled into Phase 141.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RDS-02 | Chosen redesign applied across all surviving surfaces (Welcome→Home, Hub, terminal/session, File Browser, Editor, Settings) | Per-surface hex inventory (§ below) maps every hardcoded value to its `--hub-*` token; S-07 gap identified |
| RDS-03 | Reconciled with Hub-first structure; no Sessions/Remote pages reintroduced | D-08/09/10 fences verified; D-11 copy locations found (StatusBar.tsx:49) |
| RDS-04 | `prefers-reduced-motion` honored; colorblind-safe semantics in both light and dark themes | Motion pattern documented; status dot hex confirmed as reinforcement-only; colorblind verification procedure specified |
| CARRY-01 | Hub GroupSidebar ARIA model made internally consistent (#97) | GroupSidebar.tsx fully audited; current mismatched pattern documented; correct button+aria-pressed fix specified |
</phase_requirements>

---

## Summary

Phase 141 is a **pure frontend CSS + ARIA + copy change** with no backend
involvement. The scope breaks into four concrete work items:

1. **Token migration** (S-01, S-03, S-04, S-05, S-06, and the sidebar):
   Replace every hardcoded hex constant in six non-Hub surface CSS sections of
   `frontend/src/style.css` with the corresponding `--hub-*` token. Each
   migrated selector needs a matching `[data-ui-theme="light"]` override if the
   token's light value differs from the dark-theme `:root` value. Two new tokens
   are needed (see §New Tokens Required below).

2. **Share Modal CSS gap** (S-07): `SessionShareModal.tsx` references
   `.hub-share-modal`, `.hub-share-modal--{phase}`, `.hub-share-modal__header`,
   `.hub-share-modal__title`, `.hub-share-modal__body`, and
   `.hub-share-modal__lan-creds` — none have CSS rules in `style.css`.
   The modal also has two inline `style={{...}}` attributes with hardcoded hex
   that must be lifted into CSS rules. Add these rules modelled on the
   `.hub-modal` pattern (header/body/footer, tokens throughout).

3. **ARIA fix** (CARRY-01): `GroupSidebar.tsx` has `<ul role="listbox">` with
   `<li role="option" tabIndex={0} onClick/onKeyDown>`. Replace with plain
   `<ul>` + `<li>` + `<button type="button" aria-pressed>` per the contract in
   141-UI-SPEC.md §CARRY-01.

4. **Copy fix** (D-11): `StatusBar.tsx` line 49 renders `"Share links are on
   the Sessions tab"` — must be reworded. Its test at
   `StatusBar.test.tsx:70` asserts the old string — must be updated.
   `App.tsx:710` contains a comment referencing "Sessions tab" — comment-only,
   can be reworded. `StatusBar.tsx:15` has a JSDoc reference — can be reworded.

**Primary recommendation:** Migrate surfaces in dependency order — sidebar
first (affects all surfaces visually), then S-01 Welcome, S-03 Terminal/Tab,
S-06 Settings, S-04/S-05 File Browser/Editor together (same CSS section),
S-07 Share Modal new rules last. ARIA and copy fixes are independent of CSS
order and can be bundled into any wave.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CSS token migration | Browser/Client (style.css) | — | Pure CSS; no server state |
| Light/dark theme switching | Browser/Client (`[data-ui-theme="light"]` attribute on `<html>`) | — | Attribute-conditioned CSS, already established |
| ARIA fix (GroupSidebar) | Browser/Client (GroupSidebar.tsx) | — | DOM/ARIA only, no backend touch |
| Copy fix (StatusBar) | Browser/Client (StatusBar.tsx + its test) | — | String change + test update |
| New CSS tokens | Browser/Client (style.css `:root` + `[data-ui-theme="light"]`) | — | Must land in both blocks |
| Share Modal CSS | Browser/Client (style.css `.hub-share-modal*`) | — | New rules; no backend |

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| CSS custom properties (`--hub-*`) | n/a | Token system for theme-aware colors | Already in use; the entire Hub surface runs on it |
| `@heroicons/react` 24/outline | Already installed | Colorblind-safe status/origin icons | Project standard; no new install |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| vitest | ^4.1.0 (already installed) | Unit test runner | All automated assertions |

**Installation:** No new packages needed. This phase is CSS + TSX edits only.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `--hub-*` CSS tokens | CSS Modules, Tailwind | Tokens are the established pattern; switching would regress the existing Hub surface |
| `aria-pressed` toggle button | Full `listbox`/`option` with roving tabindex | Listbox needs roving tabindex + aria-activedescendant; this is overkill for a nav filter — plain buttons are simpler and correct |

---

## Package Legitimacy Audit

No new packages installed. Not applicable.

---

## Architecture Patterns

### System Architecture Diagram

```
style.css (:root dark tokens)
  └── [data-ui-theme="light"] overrides
        └── consumed by → .tab-bar / .tab* / .tab-status-bar*
                          .sidebar / .sidebar__item*
                          .welcome-tab*
                          .file-browser* / .file-browser__preview*
                          .settings-panel* / .settings-jump-bar* / .settings-search*
                          .hub-share-modal* (NEW — S-07)
        └── no change → .hub-* / .hub__* (already tokenized)
                         .tab__agent-badge--* (semantic colors, D-03 fence)
                         .hub-card__status-dot--* (semantic colors, D-03 fence)
                         --cm-* (CodeMirror internal, out of scope)

GroupSidebar.tsx
  <ul role="listbox">            →  <ul>
    <li role="option" tabIndex=0>  →    <li>
      (onClick / onKeyDown)            <button type="button" aria-pressed>

StatusBar.tsx line 49
  "Share links are on the Sessions tab"  →  "Share link — open the Hub card"
  (or equivalent per planner)
```

### Recommended Project Structure

No structural changes to the file tree. All edits are in:

```
frontend/src/
├── style.css                              # token migration + S-07 new rules
├── components/
│   ├── Hub/
│   │   ├── GroupSidebar.tsx               # CARRY-01 ARIA fix
│   │   └── SessionShareModal.tsx          # remove inline hex styles → CSS classes
│   └── StatusBar.tsx                      # D-11 copy fix
└── components/__tests__/
    ├── StatusBar.test.tsx                 # update D-11 string assertion
    ├── Hub/GroupSidebar.test.tsx          # update role/aria assertions
    └── SessionShareModal.test.tsx         # add .hub-share-modal render smoke
```

### Pattern 1: Token Migration (non-Hub CSS surfaces)

**What:** Replace each hardcoded hex constant with the appropriate `--hub-*`
CSS custom property using `var()`. For selectors that have no per-surface
light-theme override yet, add `[data-ui-theme="light"] .selector { ... }`
blocks (or nest inside the existing `[data-ui-theme="light"]` block if one
exists).

**When to use:** Every hex value in the surfaces listed under §Per-Surface Hex
Inventory below.

**Example (tab bar):**
```css
/* BEFORE */
.tab-bar {
  background-color: #16161e;
  border-bottom: 1px solid #292e42;
}

/* AFTER */
.tab-bar {
  background-color: var(--hub-surface);
  border-bottom: 1px solid var(--hub-border);
}
/* No [data-ui-theme="light"] block needed — the token already changes */
```

**Example (new light-theme override when token value shifts):**
```css
/* For surfaces not in the existing [data-ui-theme="light"] :root block,
   add per-selector light overrides only when a token needs a different
   semantic value in context — e.g., sidebar active bg uses rgba alpha: */
[data-ui-theme="light"] .sidebar__item--active {
  background: var(--hub-sidebar-item-active-bg);
  color: var(--hub-accent);
}
```

### Pattern 2: New `--hub-*` Tokens

**What:** When no existing `--hub-*` token covers the semantic need exactly,
add new tokens to BOTH the `:root` (dark) block and the
`[data-ui-theme="light"]` block.

**When to use:** Only when confirmed gap exists (see §New Tokens Required).

**Example:**
```css
/* In :root dark block */
--hub-sidebar-active-border: #7aa2f7;   /* accent blue left border */

/* In [data-ui-theme="light"] block */
--hub-sidebar-active-border: #3d6fe8;
```

### Pattern 3: Hub-Modal Header/Body Layout (S-07 reference)

**What:** The `.hub-share-modal` panel reuses the structural layout of
`.hub-modal` but is narrower (share modal dimensions ≈ 480px wide, not 1100px).

**Reference (verified from style.css lines 4988–5100):**
```css
.hub-share-modal {
  position: relative;
  width: min(480px, calc(100vw - 48px));
  background: var(--hub-surface-elevated);
  border: 1px solid var(--hub-border);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.hub-share-modal__header {
  height: 48px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  border-bottom: 1px solid var(--hub-border);
  flex-shrink: 0;
}

.hub-share-modal__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--hub-text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hub-share-modal__body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 16px;
  overflow-y: auto;
  gap: 12px;
}

.hub-share-modal__lan-creds {
  font-size: 12px;
  color: var(--hub-text-secondary);
}

.hub-share-modal__lan-creds code {
  background: var(--hub-surface);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
```

**Animation:** Same entering/exiting phase machine as `hub-modal` but without
`hub-modal-grow` (no `transformOrigin`). Use `hub-modal-overlay-in/out` keyframes
for the overlay (already declared). Add share-modal-specific slide-in or
just fade if grow is unnecessary.

### Pattern 4: CARRY-01 ARIA Contract

**What:** Replace the mismatched `role="listbox"` + `role="option"` + all-tabIndex-0
pattern with plain `<ul>` + `<li>` + `<button type="button" aria-pressed>`.

**Verified current markup in GroupSidebar.tsx (lines 264–297):**
```tsx
// CURRENT (broken)
<ul id={SIDEBAR_LIST_ID} role="listbox">
  <li role="option" aria-selected={isActive} tabIndex={0}
      onClick={() => onGroupSelect(id)}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') ... }}>
    ...
  </li>
</ul>
```

**Target markup (per 141-UI-SPEC.md §CARRY-01):**
```tsx
// AFTER
<aside aria-label="Session groups" className={...}>
  <button
    type="button"
    aria-label={collapsed ? 'Expand group sidebar' : 'Collapse group sidebar'}
    aria-expanded={!collapsed}
    aria-controls={SIDEBAR_LIST_ID}
  />
  {!collapsed && (
    <span id="hub-group-sidebar-heading">Groups</span>
  )}
  <ul
    id={SIDEBAR_LIST_ID}
    aria-labelledby="hub-group-sidebar-heading"
    className="hub__group-sidebar-list"
    // NO role="listbox"
  >
    <li>
      <button
        type="button"
        aria-pressed={isActive}
        aria-label={`${label} group, ${counts.running}/${counts.total} sessions`}
        className={itemClass}
        onClick={() => onGroupSelect(id)}
        // onKeyDown NOT needed — native button handles Enter/Space
        onDragOver={...}
        onDragLeave={...}
        onDrop={...}
      >
        ...
      </button>
    </li>
  </ul>
</aside>
```

Key changes:
1. Remove `role="listbox"` from `<ul>`
2. Remove `role="option"` and `aria-selected` from `<li>`
3. Remove `tabIndex={0}` from `<li>` (make `<li>` inert)
4. Add inner `<button type="button">` as the interactive element
5. Add `aria-pressed={isActive}` on the button
6. Add descriptive `aria-label` including count on the button
7. Add `aria-label="Session groups"` on `<aside>` (already has the class; just add the attr)
8. Note: `onDragOver/onDragLeave/onDrop` move from `<li>` to `<button>` — drag
   drop on group items still works via the button

**GroupSidebar.test.tsx impact:** The test file asserts `role="option"` and
`aria-selected` patterns — these assertions must be updated to `role="button"`
and `aria-pressed`.

### Anti-Patterns to Avoid

- **Light-theme orphans:** Adding a new `--hub-*` token to `:root` but
  forgetting to add it to `[data-ui-theme="light"]`. Result: light theme shows
  the dark value. Prevention: always add to both blocks in the same diff.
- **Inline style survival:** `SessionShareModal.tsx` lines 292–295 have
  `style={{ color: '#a9b1d6' }}` and `style={{ background: '#16161e' }}` —
  these must be lifted to CSS classes, not left as inline styles.
- **Status hex migration:** Migrating `.tab-status-bar__state--on { color: #9ece6a }`
  to `--hub-success`. **Do not do this.** These are semantic status colors (D-03
  fence) — they stay as hardcoded hex. They are colorblind-safe via the "WEB ON"
  text label (reinforcement-only color per the colorblind contract).
- **Agent badge migration:** `.tab__agent-badge--claude { background: #7aa2f7 }`
  looks like accent blue but is a per-agent semantic identifier. Do NOT replace
  with `--hub-accent`. D-03 fence.
- **`#7C8CFF` accent:** The standalone HTML's accent is rejected. Do not use it.
- **Motion outside guard:** Any new `transition` or `animation` must be inside
  `@media (prefers-reduced-motion: no-preference)`. Static fallback goes in
  `@media (prefers-reduced-motion: reduce)`.

---

## Per-Surface Hex Inventory

This is the primary planning artifact. Every hex constant listed must be
replaced with the indicated `--hub-*` token. Sources verified against
`frontend/src/style.css`.

### Sidebar (`.sidebar`, `.sidebar__toggle`, `.sidebar__item`, lines 218–293)

[VERIFIED: source code grep]

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.sidebar` | `background-color` | `#16161e` | `--hub-surface` |
| `.sidebar` | `border-right` | `#292e42` | `--hub-border` |
| `.sidebar__toggle` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.sidebar__toggle:hover` | `background-color` | `#1e2030` | `--hub-surface-elevated` |
| `.sidebar__toggle:hover` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.sidebar__item` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.sidebar__item:hover` | `background-color` | `#1e2030` | `--hub-surface-elevated` |
| `.sidebar__item:hover` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.sidebar__item--active` | `background` | `rgba(122,162,247,0.12)` | `--hub-sidebar-item-active-bg` |
| `.sidebar__item--active` | `color` | `#7aa2f7` (via `var(--hub-accent, #7aa2f7)`) | Already uses token — verify |

**Note:** `.sidebar__item--active` at line 4537 already uses
`var(--hub-accent, #7aa2f7)` for color and `rgba(122,162,247,0.12)` as a raw
rgba literal. Replace the rgba literal with `var(--hub-sidebar-item-active-bg)`
(token already declared for both themes).

**Light theme:** The `--hub-sidebar-item-active-bg` token is already declared in
`[data-ui-theme="light"]` block (`rgba(61,111,232,0.10)`). No new light override
needed for `--hub-accent` (already overridden globally). The `background-color`
and `border-right` on `.sidebar` will auto-flip via token.

---

### S-01: Welcome Tab (`.welcome-tab*`, lines 1302–1404)

[VERIFIED: source code grep]

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.welcome-tab` | `background-color` | `#1a1b26` | `--hub-bg` |
| `.welcome-tab__tagline` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.welcome-tab__version` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.welcome-tab__heading` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.welcome-tab__text` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.welcome-tab__text strong` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.welcome-tab__install-label` | `color` | `#7aa2f7` | `--hub-accent` |
| `.welcome-tab__code` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.welcome-tab__code` | `background` | `#16161e` | `--hub-surface` |
| `.welcome-tab__code` | `border` | `#292e42` | `--hub-border` |
| `.welcome-tab__link` | `color` | `#7aa2f7` | `--hub-accent` |

**Light theme:** All covered by global `[data-ui-theme="light"]` token overrides.
No per-selector light override needed unless a semantic gap exists. Planner
should verify after migration that light theme renders correctly.

---

### S-03: Terminal / Tab Bar / Status Bar (lines 82–368)

[VERIFIED: source code grep]

#### Tab Bar (`.tab-bar`, `.tab`, `.tab-bar__chevron`)

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.tab-bar` | `background-color` | `#16161e` | `--hub-surface` |
| `.tab-bar` | `border-bottom` | `#292e42` | `--hub-border` |
| `.tab` | `border-right` | `#292e42` | `--hub-border` |
| `.tab` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.tab:hover` | `background-color` | `#1e2030` | `--hub-surface-elevated` |
| `.tab:hover` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.tab--active` | `background-color` | `#1a1b26` | `--hub-bg` |
| `.tab--active` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.tab--active` | `border-bottom` | `#7aa2f7` | `--hub-accent` |
| `.tab__rename-input` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.tab__rename-input` | `border` | `#7aa2f7` | `--hub-accent` |
| `.tab__rename-input` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.tab__close` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.tab__close:hover` | `background-color` | `#3b4261` | `--hub-border-hover` |
| `.tab__close:hover` | `color` | `#f7768e` | `--hub-destructive` |
| `.tab-bar__chevron` | `background` | `#16161e` | `--hub-surface` |
| `.tab-bar__chevron` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.tab-bar__chevron` | `border-right` | `#292e42` | `--hub-border` |
| `.tab-bar__chevron--right` | `border-left` | `#292e42` | `--hub-border` |
| `.tab-bar__chevron:hover` | `color` | `#c0caf5` | `--hub-text-primary` |

**Do NOT migrate (D-03 fence):**
```
.tab__agent-badge--claude   { background: #7aa2f7; }   ← semantic per-agent ID
.tab__agent-badge--opencode { background: #9ece6a; }   ← semantic per-agent ID
.tab__agent-badge--codex    { background: #bb9af7; }   ← semantic per-agent ID
.tab__agent-badge--gemini   { background: #2ac3de; }   ← semantic per-agent ID
.tab__agent-badge--cursor   { background: #e0af68; }   ← semantic per-agent ID
.tab__agent-badge--aider    { background: #f7768e; }   ← semantic per-agent ID
.tab__agent-badge--shell    { background: #89ddff; }   ← semantic per-agent ID
```

#### Status Bar (`.tab-status-bar*`, lines 312–368)

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.tab-status-bar` | `background-color` | `#16161e` | `--hub-surface` |
| `.tab-status-bar` | `border-top` | `#292e42` | `--hub-border` |
| `.tab-status-bar` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.tab-status-bar__hint` | `color` | `#545c7e` | NEW TOKEN: `--hub-text-dim` (see §New Tokens Required) |
| `.tab-status-bar__btn` | `border` | `#292e42` | `--hub-border` |
| `.tab-status-bar__btn` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.tab-status-bar__btn:hover` | `background-color` | `#1e2030` | `--hub-surface-elevated` |
| `.tab-status-bar__btn:hover` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.tab-status-bar__btn--active` | `border-color` | `#7aa2f7` | `--hub-accent` |
| `.tab-status-bar__btn--active` | `color` | `#7aa2f7` | `--hub-accent` |

**Do NOT migrate (D-03 fence + colorblind contract):**
```
.tab-status-bar__state--on     { color: #9ece6a; }   ← semantic status (WEB ON text is primary)
.tab-status-bar__state--off    { color: #9aa5ce; }   ← semantic status (WEB OFF text is primary)
.tab-status-bar__state--inactive { color: #414868; } ← semantic status (WEB SERVER NOT RUNNING text is primary)
```

#### Terminal Container

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.terminal-container` | `background-color` | `#1a1b26` | `--hub-bg` |

---

### S-04 + S-05: File Browser + Editor Chrome (`.file-browser*`, lines 2646–3380+)

[VERIFIED: source code grep]

#### Breadcrumb Bar (`.file-browser__breadcrumb*`)

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.file-browser__breadcrumb` | `background` | `#16161e` | `--hub-surface` |
| `.file-browser__breadcrumb` | `border-bottom` | `#292e42` | `--hub-border` |
| `.file-browser__breadcrumb` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__breadcrumb-item::before` | `color` | `#565f89` | NEW TOKEN: `--hub-text-dim` (see §New Tokens Required) |
| `.file-browser__breadcrumb-root` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.file-browser__breadcrumb-root--clickable:hover` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__breadcrumb-root--clickable:hover` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__breadcrumb-root--clickable:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__breadcrumb-segment` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.file-browser__breadcrumb-segment:hover` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__breadcrumb-segment:hover` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__breadcrumb-segment:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__breadcrumb-current` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__breadcrumb-refreshed` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__breadcrumb-refresh` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.file-browser__breadcrumb-refresh:hover` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__breadcrumb-refresh:hover` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__breadcrumb-refresh:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |

#### Status Line (`.file-browser__status*`)

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.file-browser__status` | `background` | `#16161e` | `--hub-surface` |
| `.file-browser__status` | `border-top` | `#292e42` | `--hub-border` |
| `.file-browser__status` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__status-filter` | `background` | `#1a1b26` | `--hub-bg` |
| `.file-browser__status-filter` | `border` | `#292e42` | `--hub-border` |
| `.file-browser__status-filter` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__status-filter:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__status-filter-info` | `color` | `#a9b1d6` | `--hub-text-secondary` |

#### File List (`.file-browser`, `.file-browser__list*`, `.file-browser__col*`)

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.file-browser` | `background` | `#1a1b26` | `--hub-bg` |
| `.file-browser__list-container` | `background` | `#1a1b26` | `--hub-bg` |
| `.file-browser__list` | `background` | `#1a1b26` | `--hub-bg` |
| `.file-browser__divider` | `background` | `#292e42` | `--hub-border` |
| `.file-browser__divider:hover` | `background` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__divider:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__list:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__truncated-banner` | `border-left` | `#f59e0b` | `--hub-warning` |
| `.file-browser__truncated-banner` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__truncated-banner` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__list-header` | `background` | `#16161e` | `--hub-surface` |
| `.file-browser__list-header` | `border-bottom` | `#292e42` | `--hub-border` |
| `.file-browser__col-divider` | `background` | `#292e42` | `--hub-border` |
| `.file-browser__col-divider:hover` | `background` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__col` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__col--active` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__col:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__col-chevron` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__list-row` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__list-row:hover` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__list-row--selected` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__list-row--selected` | `border-left-color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__row-icon` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.file-browser__row-icon-overlay` | `color` | `#f7768e` | `--hub-destructive` |
| `.file-browser__row-size` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.file-browser__row-mtime` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.file-browser__list-empty` | `color` | `#9aa5ce` | `--hub-text-muted` |

#### Preview / Editor Chrome (`.file-browser__preview*`, `.file-browser__btn*`)

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.file-browser__preview` | `background` | `#16161e` | `--hub-surface` |
| `.file-browser__preview` | `border-left` | `#292e42` | `--hub-border` |
| `.file-browser__preview-header` | `background` | `#16161e` | `--hub-surface` |
| `.file-browser__preview-header` | `border-bottom` | `#292e42` | `--hub-border` |
| `.file-browser__preview-header` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__preview-size` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__preview-body-muted` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__preview-heading` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__preview--idle` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__preview--loading` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.file-browser__preview--text` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__preview--markdown` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__preview--markdown a` | `color` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__preview--markdown code` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__preview--markdown pre` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__preview--markdown th/td` | `border` | `#292e42` | `--hub-border` |
| `.file-browser__preview--unsupported` (and variants) | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__btn` | `border` | `#292e42` | `--hub-border` |
| `.file-browser__btn` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.file-browser__btn:hover` | `background` | `#1e2030` | `--hub-surface-elevated` |
| `.file-browser__btn--primary` | `background` | `#7aa2f7` | `--hub-accent` |
| `.file-browser__btn--primary` | `color` | `#1a1b26` | `--hub-bg` |
| `.file-browser__btn--primary` | `border-color` | `#7aa2f7` | `--hub-accent` |

---

### S-06: Settings (`.settings-panel*`, `.settings-jump-bar*`, `.settings-search*`, lines 370–771)

[VERIFIED: source code grep]

| Selector | Property | Current Hex | `--hub-*` Token |
|----------|----------|-------------|-----------------|
| `.settings-panel` | `background-color` | `#1e2030` | `--hub-surface-elevated` |
| `.settings-panel` | `border` | `#292e42` | `--hub-border` |
| `.settings-panel__body` scrollbar | `#3b4261` | `--hub-scrollbar` |
| `.settings-panel__body` scrollbar thumb:hover | `#565f89` | `--hub-scrollbar-hover` |
| `.settings-panel__body h3` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__body h3` | `border-top` | `#292e42` | `--hub-border` |
| `.settings-jump-bar` | `background-color` | `#1a1b26` | `--hub-bg` |
| `.settings-jump-bar` | `border-bottom` | `#292e42` | `--hub-border` |
| `.settings-jump-bar__link` | `color` | `#7aa2f7` | `--hub-accent` |
| `.settings-jump-bar__link:hover` | `background-color` | `#292e42` | `--hub-border` |
| `.settings-jump-bar__link:hover` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-jump-bar__link:focus-visible` | `outline` | `#7aa2f7` | `--hub-accent` |
| `.settings-search__input` | `background-color` | `#16161e` | `--hub-surface` |
| `.settings-search__input` | `border` | `#292e42` | `--hub-border` |
| `.settings-search__input` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-search__input:focus` | `border-color` | `#7aa2f7` | `--hub-accent` |
| `.settings-search__results` | `background-color` | `#1a1b26` | `--hub-bg` |
| `.settings-search__results` | `border` | `#292e42` | `--hub-border` |
| `.settings-search__result` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-search__result:hover` | `background-color` | `#292e42` | `--hub-border` |
| `.settings-search__result:hover` | `color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__empty` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__table th` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__table th` | `border-bottom` | `#292e42` | `--hub-border` |
| `.settings-panel__table td` | `border-bottom` | `#1a1b26` | `--hub-bg` |
| `.settings-panel__cli-name` | `color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__path-input` | `background-color` | `#16161e` | `--hub-surface` |
| `.settings-panel__path-input` | `border` | `#292e42` | `--hub-border` |
| `.settings-panel__path-input` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-panel__path-input:focus` | `border-color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__error` | `color` | `#f7768e` | `--hub-destructive` |
| `.settings-panel__btn--cancel` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__btn--cancel` | `border` | `#292e42` | `--hub-border` |
| `.settings-panel__btn--cancel:hover` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-panel__btn--cancel:hover` | `border-color` | `#3b4261` | `--hub-border-hover` |
| `.settings-panel__btn--save` | `background-color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__btn--save` | `color` | `#1a1b26` | `--hub-bg` |
| `.settings-panel__btn--save:hover` | `background-color` | `#89b4fa` | `--hub-accent-hover` |
| `.settings-panel__btn--saved` | `background-color` | `#9ece6a` | `--hub-success` |
| `.settings-panel__btn--saved` | `color` | `#1a1b26` | `--hub-bg` |
| `.settings-panel__browse-btn` | `border` | `#292e42` | `--hub-border` |
| `.settings-panel__browse-btn` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__browse-btn:hover` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-panel__browse-btn:hover` | `border-color` | `#3b4261` | `--hub-border-hover` |
| `.settings-panel__label` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.settings-panel__description` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__select` | `background-color` | `#16161e` | `--hub-surface` |
| `.settings-panel__select` | `border` | `#292e42` | `--hub-border` |
| `.settings-panel__select` | `color` | `#c0caf5` | `--hub-text-primary` |
| `.settings-panel__select:focus` | `border-color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__url` | `color` | `#9aa5ce` | `--hub-text-muted` |
| `.settings-panel__url a` | `color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__check` | `color` | `#9ece6a` | `--hub-success` |
| `.settings-panel__code` | `color` | `#a9b1d6` | `--hub-text-secondary` |
| `.settings-panel__code` | `background-color` | `#16161e` | `--hub-surface` |
| `.settings-panel__details summary` | `color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__details summary:hover` | `color` | `#89b4fa` | `--hub-accent-hover` |
| `.settings-panel__toggle-track` | `background-color` | `#16161e` | `--hub-surface` |
| `.settings-panel__toggle-track` | `border` | `#292e42` | `--hub-border` |
| `.settings-panel__toggle-thumb` | `background-color` | `#565f89` | `--hub-scrollbar-hover` (or new `--hub-toggle-thumb-off`) |
| `.settings-panel__toggle-row--checked .toggle-track` | `background-color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__toggle-row--checked .toggle-track` | `border-color` | `#7aa2f7` | `--hub-accent` |
| `.settings-panel__toggle-row--checked .toggle-thumb` | `background-color` | `#1a1b26` | `--hub-bg` |
| `.settings-panel__toggle-label` | `color` | `#c0caf5` | `--hub-text-primary` |

**Note on toggle thumb off-state:** `#565f89` is `--hub-scrollbar-hover` semantically
(both are dim slate). If the planner prefers precision, introduce
`--hub-toggle-thumb-off` as a new token (`#565f89` dark / `#9999b0` light).
Otherwise reusing `--hub-scrollbar-hover` is acceptable as a shared dim-slate token.

---

### S-07: Share Modal (New CSS — `.hub-share-modal*`)

[VERIFIED: source code grep — confirmed zero existing CSS rules for these selectors]

Classes referenced in `SessionShareModal.tsx` that need CSS rules added:

| Class | Used In TSX | Rule to Add |
|-------|------------|-------------|
| `.hub-share-modal` | `className={hub-share-modal hub-share-modal--${phase}}` | Panel container (width, background, border, border-radius, flex column) |
| `.hub-share-modal--entering` | Phase machine | Fade-in animation (inside `prefers-reduced-motion: no-preference`) |
| `.hub-share-modal--exiting` | Phase machine | Fade-out animation (inside `prefers-reduced-motion: no-preference`) |
| `.hub-share-modal--open` | Phase machine | Visible state (opacity: 1, no animation) |
| `.hub-share-modal__header` | Header div | Height 48px, flex, border-bottom |
| `.hub-share-modal__title` | Title span | Font size 14px font-weight 600, text-primary |
| `.hub-share-modal__body` | Body div | flex-column, padding 16px, gap 12px |
| `.hub-share-modal__lan-creds` | Inline div (currently has `style={{...}}`) | Font size 12px, color text-secondary |

**Inline styles to lift in SessionShareModal.tsx:**
- Line 292: `style={{ margin: '8px 0', fontSize: 12, color: '#a9b1d6' }}`
  → remove `color: '#a9b1d6'`; add `color: var(--hub-text-secondary)` to `.hub-share-modal__lan-creds`
- Line 294: `style={{ background: '#16161e', padding: '2px 6px', borderRadius: 3, fontFamily: 'monospace' }}`
  → lift to `.hub-share-modal__lan-creds code` rule

**Note:** The modal already uses `hub-modal-overlay` for the overlay scrim
(class is correct and has CSS). The animation keyframes (`hub-modal-overlay-in/out`)
are already declared. Only the share-modal panel itself (`hub-share-modal`) and
its children need new rules. Use a simpler enter/exit (fade-in/out, no grow)
since this modal is smaller and less dramatic than HubModal.

---

## New Tokens Required

Two new `--hub-*` tokens are needed. Both must be added to `:root` (dark) and
`[data-ui-theme="light"]` blocks.

[VERIFIED: source code grep confirmed these hex values exist in non-Hub surfaces
but have no corresponding token]

### `--hub-text-dim`
Used by: `.tab-status-bar__hint` (`#545c7e`) and `.file-browser__breadcrumb-item::before`
separator (`#565f89`). These are the same semantic role — very faint hint text
dimmer than `--hub-text-muted`. The two hex values are close; consolidate to one
token.

| Theme | Value |
|-------|-------|
| Dark (`:root`) | `#565f89` |
| Light (`[data-ui-theme="light"]`) | `#9999b0` (matches `--hub-text-placeholder` light value — same tier of dimness) |

### `--hub-toggle-thumb-off` (optional — planner's discretion per D-03 latitude)

Used by: `.settings-panel__toggle-thumb` off state (`#565f89`). If the planner
decides to reuse `--hub-scrollbar-hover` (`#565f89` / `#9999b0`) for this, no
new token is needed. If named precision is preferred, introduce this token.

| Theme | Value |
|-------|-------|
| Dark (`:root`) | `#565f89` |
| Light (`[data-ui-theme="light"]`) | `#9999b0` |

---

## D-03 Boundary Confirmation

The following hex values appear in the CSS but must **NOT** be migrated to
`--hub-*` tokens. All confirmed via source code grep.

[VERIFIED: source code — confirmed D-03 boundary]

**Agent badge colors** (`.tab__agent-badge--*`, lines 1097–1103):
Each badge is a semantic per-agent visual identifier. Users identify agents by
their badge color. These must remain as per-agent hardcoded values.
```
#7aa2f7  claude        DO NOT MIGRATE
#9ece6a  opencode      DO NOT MIGRATE
#bb9af7  codex         DO NOT MIGRATE
#2ac3de  gemini        DO NOT MIGRATE
#e0af68  cursor        DO NOT MIGRATE
#f7768e  aider         DO NOT MIGRATE
#89ddff  shell         DO NOT MIGRATE
```

**Semantic status colors** (`.tab-status-bar__state--*`, lines 334–336 and
Hub card status dot lines 4217–4228):
```
#9ece6a  WEB ON state       colorblind-safe: "WEB ON" text is primary
#9aa5ce  WEB OFF state      colorblind-safe: "WEB OFF" text is primary
#414868  server-not-running colorblind-safe: "WEB SERVER NOT RUNNING" text is primary
#3b82f6  running (hub dot)  colorblind-safe: ArrowPathIcon + "Running" label
#22c55e  idle (hub dot)     colorblind-safe: CheckCircleIcon + "Idle" label
#f59e0b  waiting (hub dot)  colorblind-safe: PauseCircleIcon + "Waiting" label
#ef4444  errored (hub dot)  colorblind-safe: ExclamationCircleIcon + "Errored" label
#565f89  stopped (hub dot)  colorblind-safe: StopCircleIcon + "Stopped" label
```

**CodeMirror internal theme (`--cm-*`):** Not present in the codebase as CSS
custom properties (searched — no matches). CodeMirror 6 themes are applied via
JS `theme` extension in Editor.tsx. This is confirmed out of scope for Phase 141.

---

## D-11 Copy Fix — Exact Locations

[VERIFIED: source code grep — found all instances]

| File | Line | Current Text | Action |
|------|------|-------------|--------|
| `frontend/src/components/StatusBar.tsx` | 49 | `Share links are on the Sessions tab` | **Change to:** `Share — open the Hub card` (or equivalent reword per planner) |
| `frontend/src/components/StatusBar.tsx` | 15 | JSDoc: `is the Sessions tab's job` | Reword comment (non-functional, editorial) |
| `frontend/src/components/__tests__/StatusBar.test.tsx` | 66 | test description: `'shows hint pointing to Sessions tab'` | Update test description |
| `frontend/src/components/__tests__/StatusBar.test.tsx` | 70 | `expect(hint?.textContent).toBe('Share links are on the Sessions tab')` | **Update assertion** to match new copy string |
| `frontend/src/App.tsx` | 710 | Code comment: `the Sessions tab show bogus "WEB ON" state` | Reword comment (non-functional, editorial) |

The only **functional** change is `StatusBar.tsx:49`. The test at
`StatusBar.test.tsx:70` is a **blocking** change — it will fail if the text
changes but the assertion does not. The JSDoc and App.tsx comment are editorial.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Light/dark token switching | Per-surface `@media` or JS theme check | `[data-ui-theme="light"]` attribute on `<html>` + CSS custom property overrides | Already established pattern; zero JS needed |
| ARIA navigation filter | Custom `role="listbox"` with roving tabindex | `<button type="button" aria-pressed>` (native button) | Listbox needs roving tabindex + aria-activedescendant; plain buttons are correct for navigation filters and require no custom key handling |
| Motion guards | `window.matchMedia` JS check | `@media (prefers-reduced-motion: reduce)` CSS | Established pattern in this codebase; already used correctly by hub-modal, find-bar, banner |
| Colorblind verification | Visual inspection (impossible for colorblind user) | `grep` for hex constants in modified CSS files | User is colorblind; grep is the only reliable method |

---

## Motion Contract

[VERIFIED: source code — confirmed existing pattern across `.find-bar`, `.banner`,
`.webgl-recovery-banner`, `.hub-card--attention`, `.hub-modal`]

The established pattern in this codebase uses **static-first** transitions:
transitions are declared on the element unconditionally, then overridden with
`transition: none` inside `@media (prefers-reduced-motion: reduce)`.

```css
/* Example from .find-bar (lines 2421–2549) */
.find-bar {
  transition: transform 200ms ease, opacity 200ms ease;  /* always present */
}
@media (prefers-reduced-motion: reduce) {
  .find-bar,
  .find-bar--entering,
  .find-bar--exiting {
    transition: none;  /* static override */
  }
}
```

For the hub-modal pattern (what S-07 should follow):
```css
/* Keyframes declared at root scope (outside media query) */
@keyframes hub-share-modal-in { from { opacity: 0; } to { opacity: 1; } }
@keyframes hub-share-modal-out { from { opacity: 1; } to { opacity: 0; } }

/* Animation assignment — inside no-preference guard */
@media (prefers-reduced-motion: no-preference) {
  .hub-share-modal--entering {
    animation: hub-share-modal-in 150ms ease forwards;
  }
  .hub-share-modal--exiting {
    animation: hub-share-modal-out 120ms ease forwards;
  }
}

/* Reduced-motion: instant appear/disappear */
@media (prefers-reduced-motion: reduce) {
  .hub-share-modal {
    animation: none;
    transition: none;
    opacity: 1;
  }
}
```

**For migrated non-Hub surfaces:** These surfaces have `transition:
background-color 0.1s` / `color 0.1s` on hover states. These hover transitions
must be wrapped in `@media (prefers-reduced-motion: no-preference)` blocks, with
`transition: none` fallbacks in the reduce block. Follow the `.banner` and
`.find-bar` pattern exactly.

---

## Common Pitfalls

### Pitfall 1: Forgetting the Light Theme Override

**What goes wrong:** A hex value is replaced with `--hub-*` in the `:root`
(dark) rules, but no `[data-ui-theme="light"]` rule is added for selectors
that had previously hardcoded a value that only existed for dark theme.

**Why it happens:** The non-Hub surfaces currently have NO light-theme rules at
all. After migration, the `--hub-*` tokens auto-apply the light values globally
— but only for tokens that are declared in both `:root` and
`[data-ui-theme="light"]`. Custom tokens introduced in Phase 141 (e.g.,
`--hub-text-dim`) require explicit addition to both blocks.

**How to avoid:** For each new token added to `:root`, immediately add the
corresponding value to `[data-ui-theme="light"]` in the same diff.

**Warning signs:** If a surface looks correct in dark mode but has dark colors
visible in light mode, a light token is missing.

### Pitfall 2: Inline Style Survival in SessionShareModal

**What goes wrong:** The `hub-share-modal__lan-creds` div (line 292) has
`style={{ color: '#a9b1d6' }}` and the `<code>` inside has
`style={{ background: '#16161e' }}`. If only CSS rules are added but these
inline styles are not removed, the inline styles override the CSS, breaking
light-theme support (inline styles have highest specificity short of `!important`).

**Why it happens:** The original code used inline styles as a quick-ship shortcut.

**How to avoid:** Remove the inline `color` and `background` properties from the
TSX; they are replaced by the new `.hub-share-modal__lan-creds` and
`.hub-share-modal__lan-creds code` CSS rules.

**Warning signs:** Light theme shows a dark background code block in the share
modal despite correct CSS rules.

### Pitfall 3: ARIA Test File Mismatch

**What goes wrong:** `GroupSidebar.test.tsx` asserts `role="option"` and
`aria-selected` on items. After the CARRY-01 fix, the test will fail because
those attributes no longer exist.

**Why it happens:** The test was written to verify the old (broken) ARIA pattern.

**How to avoid:** Update `GroupSidebar.test.tsx` simultaneously with the
component change. The new assertions should check `role="button"` and
`aria-pressed` on items.

**Warning signs:** `vitest run` fails on `GroupSidebar.test.tsx` after the
component change but before the test update.

### Pitfall 4: D-03 Badge Hex Accidentally Migrated

**What goes wrong:** The token for accent blue is `--hub-accent: #7aa2f7`.
Several agent badge backgrounds also use `#7aa2f7` (claude badge). A naive
find-and-replace of `#7aa2f7` → `var(--hub-accent)` would also change agent
badges, breaking their semantic-identifier purpose.

**Why it happens:** Claude's badge intentionally uses the same blue, but for a
different semantic purpose (agent identity, not theme accent).

**How to avoid:** Migrate hex values selector-by-selector, not via global find-
and-replace. Only touch `.tab__agent-badge--*` rules if explicitly listed in
§Per-Surface Hex Inventory (they are NOT listed — intentionally excluded).

**Warning signs:** All agent tabs look the same color; badge differentiation is
lost in light theme.

### Pitfall 5: StatusBar Test String Mismatch

**What goes wrong:** D-11 fix changes `StatusBar.tsx:49` copy, but
`StatusBar.test.tsx:70` still asserts the old string.

**Why it happens:** The test is an exact string match.

**How to avoid:** The StatusBar.tsx change and StatusBar.test.tsx change MUST
be in the same commit or the test suite will fail.

---

## Code Examples

### Verified Motion Pattern (from `.hub-modal`, lines 5244–5273)

```css
/* Source: frontend/src/style.css lines 5244–5273 */
@media (prefers-reduced-motion: no-preference) {
  .hub-modal--entering {
    animation: hub-modal-grow 220ms cubic-bezier(0.25, 0.46, 0.45, 0.94) forwards;
  }
  .hub-modal--exiting {
    animation: hub-modal-shrink 180ms cubic-bezier(0.55, 0, 1, 0.45) forwards;
  }
  .hub-modal-overlay--entering {
    animation: hub-modal-overlay-in 220ms ease forwards;
  }
  .hub-modal-overlay--exiting {
    animation: hub-modal-overlay-out 180ms ease forwards;
  }
}
@media (prefers-reduced-motion: reduce) {
  .hub-modal {
    animation: none;
    transition: none;
    opacity: 1;
    transform: none;
  }
  .hub-modal-overlay {
    animation: none;
    transition: none;
    opacity: 1;
  }
}
```

### Verified Token System — Sidebar Active State (from style.css line 4537)

```css
/* Source: frontend/src/style.css line 4537 */
.sidebar__item--active {
  background: rgba(122, 162, 247, 0.12);  /* → var(--hub-sidebar-item-active-bg) */
  color: var(--hub-accent, #7aa2f7);       /* already uses token — keep; verify fallback */
}
```

### CARRY-01 ARIA Target Pattern

```tsx
/* Source: 141-UI-SPEC.md §CARRY-01 */
<aside aria-label="Session groups" className={`hub__group-sidebar${collapsed ? ' hub__group-sidebar--collapsed' : ''}`}>
  <button
    type="button"
    className="hub__group-sidebar-toggle"
    onClick={onToggle}
    aria-label={collapsed ? 'Expand group sidebar' : 'Collapse group sidebar'}
    aria-expanded={!collapsed}
    aria-controls={SIDEBAR_LIST_ID}
  >
    {collapsed ? <ChevronRightIcon aria-hidden="true" /> : <ChevronLeftIcon aria-hidden="true" />}
  </button>

  {!collapsed && (
    <span id="hub-group-sidebar-heading" className="hub__group-sidebar-heading">Groups</span>
  )}

  <ul
    id={SIDEBAR_LIST_ID}
    className="hub__group-sidebar-list"
    aria-labelledby="hub-group-sidebar-heading"
    /* NO role="listbox" */
  >
    <li>
      <button
        type="button"
        className={itemClass}
        aria-pressed={isActive}
        aria-label={`${label} group, ${counts.running}/${counts.total} sessions`}
        onClick={() => onGroupSelect(id)}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        {/* children unchanged */}
      </button>
    </li>
  </ul>
</aside>
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `role="listbox"` for navigation filters | `<button aria-pressed>` for navigation filters | Phase 141 (CARRY-01) | Correct ARIA semantics for filter controls; no roving tabindex complexity |
| Hardcoded TokyoNight hex in non-Hub surfaces | `--hub-*` CSS tokens | Phase 141 | Light/dark theme toggle works uniformly across all surfaces |
| Inline `style={{...}}` with hardcoded colors in SessionShareModal | CSS class rules with tokens | Phase 141 | Light theme support in share modal |
| "Sessions tab" copy in StatusBar | "Hub card" reference | Phase 141 (D-11) | Accurate — Sessions page removed in Phase 138 |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `--cm-*` variables do not exist in style.css (CodeMirror theme is pure JS, not CSS custom properties) | D-03 Boundary Confirmation | Low risk — if `--cm-*` rules were found, they are still out of scope per D-02; no plan impact |
| A2 | `#545c7e` (status bar hint) and `#565f89` (breadcrumb separator) are close enough to consolidate into one `--hub-text-dim` token | New Tokens Required | If color precision matters for these two uses, planner may introduce two separate tokens instead |
| A3 | `--hub-toggle-thumb-off` can reuse `--hub-scrollbar-hover` (`#565f89`/`#9999b0`) without introducing a new token | Settings toggle | Semantic clarity suffers slightly; visual result is identical |

---

## Open Questions

1. **S-07 Share Modal animation style**
   - What we know: HubModal uses a grow+fade animation; SessionShareModal uses no grow (comment in TSX says "without grow animation").
   - What's unclear: Whether a plain fade-in/out is the right choice or if no animation (always `open` phase) is preferable for a smaller modal.
   - Recommendation: Planner can choose — fade is safer and consistent with hub-modal-overlay behavior.

2. **Sidebar light-theme transition timing**
   - What we know: `.sidebar` has `transition: width 0.15s ease` for collapse. This is a layout transition, not a color transition, so it does not need a motion guard by the letter of the spec.
   - What's unclear: Whether the planner wants to add a `prefers-reduced-motion: reduce` guard for the width transition anyway (defensive quality).
   - Recommendation: Add the guard for defensive correctness — it is low-cost.

---

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — this phase is CSS + TSX + test file edits only).

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0 |
| Config file | `frontend/vite.config.ts` (vitest embedded) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite + type-check | `cd frontend && pnpm exec tsc --noEmit && pnpm test` |

**Critical note from project memory:** `tsc && vite build` rejects TS errors
that vitest tolerates. The post-merge gate MUST run `tsc --noEmit` (not just
vitest) to confirm the app builds.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RDS-02 | All non-Hub surface CSS uses `--hub-*` tokens | Grep assertion | `grep -rn '#[0-9a-fA-F]\{3,6\}' frontend/src/style.css` — no hex should appear in migrated selectors | Manual / CI grep |
| RDS-02 | `.hub-share-modal` renders with correct DOM structure | Unit — smoke | `pnpm test -- SessionShareModal` | ✅ exists |
| RDS-03 | StatusBar shows updated D-11 copy (no "Sessions tab" text) | Unit | `pnpm test -- StatusBar` | ✅ exists |
| RDS-04 | All transitions in `prefers-reduced-motion: no-preference` guards | Code review / grep | `grep -n 'transition\|animation' frontend/src/style.css` cross-checked against media query blocks | Manual |
| RDS-04 | Theme toggle: light theme applies to all migrated surfaces | Visual (grep-backed) | Grep `[data-ui-theme="light"]` — confirm migrated selectors covered by token | Manual |
| CARRY-01 | GroupSidebar items have `aria-pressed` attribute, not `role="option"` | Unit | `pnpm test -- GroupSidebar` | ✅ exists (needs update) |
| CARRY-01 | GroupSidebar `<ul>` has no `role="listbox"` | Unit | same GroupSidebar test suite | ✅ (assertion to add) |

### Colorblind Verification (grep-based — user is colorblind)

```bash
# Verify accent tokens at hex-constant level:
grep -rn '#7aa2f7' frontend/src/style.css  # dark accent — should only appear in agent badge + token declaration + status-color annotations
grep -rn '#3d6fe8' frontend/src/style.css  # light accent — should only appear in token declaration

# Confirm no hardcoded hex remains in migrated CSS sections (post-migration):
# Tab bar section (lines ~82–368):
sed -n '82,368p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'
# Welcome tab section (lines ~1302–1404):
sed -n '1302,1404p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'
# Settings section (lines ~370–771):
sed -n '370,771p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'
# File browser section (lines ~2646–3380):
sed -n '2646,3380p' frontend/src/style.css | grep -E '#[0-9a-fA-F]{3,6}'
```

Permitted residual hex in migrated sections (DO NOT flag these):
- Agent badge colors (`.tab__agent-badge--*`) — D-03 fence
- Status state colors (`.tab-status-bar__state--*`) — D-03 fence + colorblind contract
- The `box-shadow: 0 8px 32px rgba(0,0,0,0.5)` style rgba values — these are shadow
  opacity, not theme colors

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm test -- <surface-test-file>`
- **Per wave merge:** `cd frontend && pnpm exec tsc --noEmit && pnpm test`
- **Phase gate:** Full suite green + no hardcoded hex in migrated sections (grep) before `/gsd:verify-work`

### Wave 0 Gaps

The following test assertions need to be added (no new test files needed — only
updates to existing files):

- [ ] `GroupSidebar.test.tsx` — update assertions to check `role="button"` on items (not `role="option"`), `aria-pressed` attribute, no `role="listbox"` on `<ul>` (CARRY-01)
- [ ] `StatusBar.test.tsx` line 70 — update expected string to match new D-11 copy
- [ ] `StatusBar.test.tsx` line 66 — update test description string
- [ ] `SessionShareModal.test.tsx` — add smoke test asserting `.hub-share-modal__header` and `.hub-share-modal__body` render (S-07)

---

## Security Domain

This phase makes no backend changes, no authentication changes, and no capability
changes. The share modal interaction model is preserved as-is (D-13). ASVS
categories are not applicable to a CSS token migration + ARIA fix.

---

## Sources

### Primary (HIGH confidence)
- `frontend/src/style.css` — directly grepped and read; all hex values verified by direct file inspection
- `frontend/src/components/Hub/GroupSidebar.tsx` — directly read; current ARIA pattern confirmed
- `frontend/src/components/Hub/SessionShareModal.tsx` — directly read; inline styles confirmed, class names confirmed
- `frontend/src/components/StatusBar.tsx` — directly read; D-11 copy string confirmed at line 49
- `.planning/phases/141-redesign-implementation/141-UI-SPEC.md` — locked design contract, CARRY-01 ARIA resolution, phase constraints
- `.planning/phases/141-redesign-implementation/141-CONTEXT.md` — micro-decisions D-01/D-02/D-03

### Secondary (MEDIUM confidence)
- `.planning/phases/140-ui-spec-gate/140-UI-SPEC.md` — upstream gate for D-05 accent rejection, D-08/09/10 structural fences, D-11, D-13

### Tertiary
None — all findings are verified from direct source code inspection.

---

## Metadata

**Confidence breakdown:**
- Per-surface hex inventory: HIGH — directly read from source code
- ARIA fix target markup: HIGH — verified from GroupSidebar.tsx + 141-UI-SPEC.md CARRY-01
- D-11 copy locations: HIGH — grep-verified
- New tokens required: HIGH — verified by absence in existing token blocks
- Motion contract pattern: HIGH — verified from existing working examples in style.css

**Research date:** 2026-06-20
**Valid until:** 2026-07-20 (stable CSS codebase; changes only with frontend edits)
