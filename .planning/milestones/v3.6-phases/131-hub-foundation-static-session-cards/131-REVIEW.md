---
phase: 131-hub-foundation-static-session-cards
reviewed: 2026-06-16T00:00:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - internal/daemon/types.go
  - internal/daemon/engine.go
  - app.go
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/components/Hub/InlineSessionName.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/Hub/HubFilterBar.tsx
  - frontend/src/components/Hub/HubEmptyState.tsx
  - frontend/src/components/Hub/SessionCardGrid.tsx
  - frontend/src/components/Hub/HubPanel.tsx
  - frontend/src/components/Sidebar.tsx
  - frontend/src/components/TabBar.tsx
  - frontend/src/App.tsx
  - frontend/src/style.css
findings:
  critical: 2
  warning: 3
  info: 2
  total: 7
status: issues_found
---

# Phase 131: Code Review Report

**Reviewed:** 2026-06-16
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

This phase introduces the Hub UI surface: a React/TS session-card grid over local session data plus a Go backend `WorkDir` field threaded through the daemon → engine → app.go → Wails RPC chain. The Go backend changes are clean. The XSS concern is a non-issue: no `dangerouslySetInnerHTML` is used anywhere; all session-name, workDir, hostname, and CLI values are rendered as React text nodes (default JSX escaping). The Hub coexistence with the Sessions panel (HUB-02) is correctly implemented — Hub gets its own tab and its own poll loop. The 3s poll cleanup is correct (cancelled flag + `clearInterval`).

Two blockers were found: a widespread CSS class name mismatch that breaks all Hub card and filter-bar layout, and a double RPC call on every Hub rename. Three warnings cover code duplication, a stale `hubSessions` display, and an unused destructured variable used in a misleading way.

---

## Critical Issues

### CR-01: CSS class names in Hub components do not match style.css definitions

**Files:**
- `frontend/src/components/Hub/SessionCard.tsx:183-239`
- `frontend/src/components/Hub/HubFilterBar.tsx:109-149`
- `frontend/src/style.css:4277-4412`

**Issue:** The CSS and the TSX components use completely different BEM class names for card rows and the filter bar. The components were written against one naming scheme; the CSS was written against another. None of the mismatched classes have any fallback styling, so the Hub renders unstyled at runtime.

Mismatch table:

| TSX uses (no CSS definition) | CSS defines |
|---|---|
| `hub-card__row` | `hub-card__row1` |
| `hub-card__row--primary` | `hub-card__row1` |
| `hub-card__row--origin` | `hub-card__row2` |
| `hub-card__row--meta` | `hub-card__row3` |
| `hub-card__row--exit` | `hub-card__row4` |
| `hub-card__status-indicator` | _(no CSS rule)_ |
| `hub-card__status-label` | _(no CSS rule)_ |
| `hub-card__time` | `hub-card__uptime` |
| `hub-filter` (wrapper) | `hub__filter-bar` |
| `hub-filter__pills` | _(no CSS rule)_ |
| `hub-filter__new-session` | _(no CSS rule)_ |

Additionally, the CLI badge in `SessionCard.tsx` is rendered with `tab__agent-badge tab__agent-badge--{cli}` (the 8×8 dot used in TabBar), but the CSS defines a separate text-style badge `hub-card__badge` for Hub cards. The dot badge class works functionally but renders the wrong visual shape for the Hub spec.

**Fix — Option A (align TSX to CSS):** Rename the TSX class names to match the CSS definitions. This is the lower-risk option since it requires no CSS changes.

In `SessionCard.tsx`:
```tsx
// Replace all four hub-card__row BEM variants:
<div className="hub-card__row1">           // was hub-card__row hub-card__row--primary
<div className="hub-card__row2">           // was hub-card__row hub-card__row--origin
<div className="hub-card__row3">           // was hub-card__row hub-card__row--meta
<div className="hub-card__row4">           // was hub-card__row hub-card__row--exit

// Replace missing classes:
<span className="hub-card__status-indicator"> → wrap inline or use an unstyled span; add CSS rule
<span className="hub-card__status-label">    → add CSS rule, or inline
<span className="hub-card__time">            // was hub-card__time — rename to hub-card__uptime

// CLI badge: use hub-card__badge instead of tab__agent-badge
<span className="hub-card__badge">{cli}</span>
```

In `HubFilterBar.tsx`:
```tsx
<div className="hub__filter-bar">          // was hub-filter
// hub-filter__pills and hub-filter__new-session need CSS rules added
```

**Fix — Option B (align CSS to TSX):** Rename the CSS rules to use `hub-card__row` + BEM modifiers and `hub-filter` wrapper. Requires updating style.css. Either option is acceptable; the mismatch must be resolved before the Hub renders correctly.

---

### CR-02: Double RenameSession RPC call on every Hub card rename

**Files:**
- `frontend/src/components/Hub/InlineSessionName.tsx:40-50`
- `frontend/src/App.tsx:776-785`

**Issue:** `InlineSessionName.commitEdit` calls `RenameSession(id, trimmed)` directly (line 43), then calls `onRenamed?.(trimmed)` (line 46). The `onRenamed` callback propagates up the component tree:

```
InlineSessionName.onRenamed(trimmed)
  → SessionCard.onRename?.(id, newName)       [SessionCard.tsx:195]
  → HubPanel.onRename(id, newName)             [HubPanel.tsx:136]
  → App.handleRenameTab(id, name)              [App.tsx:1331]
  → RenameSession(id, name)                    [App.tsx:778]  ← second call
```

Every Hub rename therefore calls `RenameSession` twice: once at the leaf component and once at App root. Both calls send the same ID and name so the result is idempotent, but the double RPC doubles daemon socket round-trips and can generate a visible race if the session is renamed externally between the two calls (the second call overwrites the external rename).

The root cause is that `InlineSessionName` was designed to call the RPC itself (mirroring `TabBar`'s `commitEdit` which calls `onRename` only once, delegating the RPC to `App.handleRenameTab`). The correct pattern in the Hub context is to have `InlineSessionName` call `onRenamed` only and let the App-level handler own the RPC.

**Fix:** Remove the direct `RenameSession` call from `InlineSessionName.commitEdit` and let the existing callback chain handle it:

```tsx
// InlineSessionName.tsx — commitEdit
async function commitEdit(): Promise<void> {
  const trimmed = editValue.trim()
  if (trimmed.length > 0 && trimmed !== name) {
    // Do NOT call RenameSession here — the parent (App.handleRenameTab)
    // already calls RenameSession via the onRename → onRenamed callback chain.
    onRenamed?.(trimmed)
  } else {
    setEditValue(name)
  }
  setEditing(false)
}
```

If `InlineSessionName` is also used in contexts where it must own the RPC call itself (i.e., no parent callback), the RPC should be kept and `onRenamed` should be a post-success notification rather than a trigger for a second save.

---

## Warnings

### WR-01: `deriveStatus` logic triplicated across three Hub files

**Files:**
- `frontend/src/components/Hub/SessionCard.tsx:56-61`
- `frontend/src/components/Hub/HubFilterBar.tsx:36-41`
- `frontend/src/components/Hub/HubPanel.tsx:15-20`

**Issue:** Identical `deriveStatus` / `deriveFilterStatus` logic appears three times with only naming differences. All three implement:
```ts
if (s.state === 'stopped') return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
return s.status as HubStatus
```
Comments in the code acknowledge the duplication. A future change to the stopped-state logic (e.g., adding a `cancelled` exit state) requires three simultaneous edits; missing one silently creates a cross-component status inconsistency.

**Fix:** Extract to a shared utility, e.g., `frontend/src/lib/hubStatus.ts`:
```ts
export type HubStatus = 'running' | 'idle' | 'waiting' | 'errored' | 'stopped-ok' | 'stopped-err'
export function deriveHubStatus(s: SessionInfo): HubStatus {
  if (s.state === 'stopped') return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
  return s.status as HubStatus
}
```
Import from all three components. This is the third call-site for this pattern (after `SessionCard` and `HubFilterBar`), meeting the project's 3-example abstraction threshold.

---

### WR-02: `hubSessions` stale on Hub tab re-open; Hub error state never clears on tab close

**File:** `frontend/src/App.tsx:904-919`

**Issue:** The Hub polling effect (lines 904–919) sets `hubSessions` and `hubError` while the Hub tab is active, then stops the interval on cleanup. When the user navigates away and returns to the Hub tab, the previous `hubSessions` array is still in state and renders momentarily before the first fresh poll completes. This creates a flash of stale data — particularly noticeable if a session was killed while the Hub was hidden.

More critically, if the Hub tab is active and `ListSessions()` throws (setting `hubError = true`), then the user navigates away, the error state is left set. On returning to the Hub, the error state renders immediately even though the daemon may be healthy again.

```ts
// App.tsx line 904
useEffect(() => {
  if (mode === 'web') return
  if (activeId !== HUB_TAB.id) return
  // ...
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])
```

**Fix:** Reset `hubSessions` and `hubError` when the Hub tab is activated (at effect start, before the first poll completes), mirroring the pattern used in the daemon-manager panel:

```ts
useEffect(() => {
  if (mode === 'web') return
  if (activeId !== HUB_TAB.id) return
  // Reset stale state immediately so we don't flash old data on re-open.
  setHubError(false)
  // Do NOT reset hubSessions here — that would cause a flicker to empty state.
  // The first refresh() call below runs synchronously before the next render.
  let cancelled = false
  async function refresh() { /* ... */ }
  void refresh()
  const interval = setInterval(() => void refresh(), 3000)
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])
```

At minimum, `setHubError(false)` should be called at the start of the effect so a stale error banner does not appear when the user returns to a healthy Hub.

---

### WR-03: `SessionCard` renders `tab__agent-badge` (8px color dot) instead of text badge `hub-card__badge`

**File:** `frontend/src/components/Hub/SessionCard.tsx:162-165, 199`

**Issue:** `SessionCard` computes `badgeClass` using `tab__agent-badge tab__agent-badge--{cli}` (the 8×8 pixel colored dot defined for the tab bar). The Hub spec and the CSS define a separate text-style badge `.hub-card__badge` that renders the CLI name as a small monospace chip. The component renders a color-only dot and then separately renders `{cli}` text inside the badge span (line 200), resulting in a hybrid that matches neither the tab-bar pattern (dot only, no text, `aria-hidden`) nor the Hub-card badge pattern (text chip).

```tsx
// Current (line 162-165, 199-201)
const modifier = agentBadgeModifier(cli)
const badgeClass = modifier
  ? `tab__agent-badge tab__agent-badge--${modifier}`
  : 'tab__agent-badge'
// ...
<span className={badgeClass} aria-hidden="true">
  {cli}   {/* tab badge is normally empty — text inside is unintended */}
</span>
```

**Fix:** Use the Hub-specific badge class instead:
```tsx
<span className="hub-card__badge">
  {cli}
</span>
```
The color-dot reinforcement (the `.tab__agent-badge--{cli}` palette) is redundant in the Hub card because the CLI name text is already visible. If a color dot is desired, add it as a separate `span` before the text badge.

---

## Info

### IN-01: `_workDir` unused destructure in `SessionCard` serves no structural purpose

**File:** `frontend/src/components/Hub/SessionCard.tsx:147`

**Issue:** `workDir` is destructured as `_workDir` (convention for intentional discard). The comment context implies it was extracted to prevent accidental spread, but `session` is never spread into another object in this component. The destructure adds noise without value.

**Fix:** Remove `workDir: _workDir` from the destructure. `session.workDir` is passed through to `SessionCardGrid` grouping one level up; `SessionCard` itself has no need to reference or exclude it.

---

### IN-02: `SessionCard.tsx` aria-label for `stopped-err` status includes both icon `aria-label` and exit-code chip text, creating duplicate screen-reader announcement

**File:** `frontend/src/components/Hub/SessionCard.tsx:173-175, 186-188, 237-239`

**Issue:** The card `aria-label` is built as `${name}, ${displayLabel}, ${cli}, ${originText}` (line 174). For `stopped-err`, `displayLabel` is `"Exited 1"`. The card also contains a visible exit-code chip `<span className="hub-card__exit-chip">Exited {exitCode}</span>` (line 238) whose text is read by screen readers because it is not `aria-hidden`. A screen reader on a non-zero-exit card will announce: "session name, Exited 1, claude, Local" (from aria-label) then separately read the chip "Exited 1". The exit code is announced twice.

**Fix:** Add `aria-hidden="true"` to the exit-code chip since the exit code is already communicated in the card's `aria-label`:
```tsx
<span className="hub-card__exit-chip" aria-hidden="true">Exited {exitCode}</span>
```

---

_Reviewed: 2026-06-16_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
