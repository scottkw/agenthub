# Phase 133: Attention + Pulse - Pattern Map

**Mapped:** 2026-06-16
**Files analyzed:** 6 (5 modified source files + 1 test file gap)
**Analogs found:** 6 / 6 (all modifications have close analogs in the same files)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/lib/hubStatus.ts` | utility | transform | `deriveHubStatus()` in the same file (lines 23-28) | exact |
| `frontend/src/components/Hub/SessionCard.tsx` | component | request-response | `hub-card--dim` / `hub-card--dragging` modifier pattern + STATUS_CONFIG icon rendering (lines 216, 283-304) | exact |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | component | transform | `groupByNamedGroups` / `groupByWorkDir` sort loops + listitem render (lines 14-53, 117-148) | exact |
| `frontend/src/components/Hub/GroupSidebar.tsx` | component | CRUD | `GroupCounts` interface + `computeCounts` + `NeedsInputBadge` + badge render condition (lines 17-47, 55-68, 137-139) | exact |
| `frontend/src/components/Hub/HubPanel.tsx` | component | event-driven | `usePreviewPoller` hook (useRef + setInterval debounce-adjacent pattern, lines 58-112) + `allSessions` merge + prop threading (lines 214-284) | role-match |
| `frontend/src/style.css` | config | — | `@keyframes hub-spin` + `prefers-reduced-motion` guard (lines 4359-4370); `--hub-needs-input-badge-*` tokens (lines 4129-4131); `.hub-card--dim` / `.hub-card--dragging` modifiers (lines 4303-4306, 4848-4852); `.hub__group-sidebar-item__needs-input-badge` (lines 4794-4805) | exact |

---

## Pattern Assignments

### `frontend/src/lib/hubStatus.ts` (utility, transform)

**Analog:** `deriveHubStatus()` in the same file

**Existing export pattern** (lines 1-28 — full file; it is 28 lines):
```typescript
import type { SessionInfo } from '../wailsjs/go/main/App'

export type HubStatus = 'running' | 'idle' | 'waiting' | 'errored' | 'stopped-ok' | 'stopped-err'

export function deriveHubStatus(s: SessionInfo): HubStatus {
  if (s.state === 'stopped') {
    return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
  }
  return s.status as HubStatus
}
```

**Core addition pattern** — append after line 28, copying the same exported-function style:
```typescript
/* ATTN-01: canonical attention predicate — waiting, errored, or non-zero-exit sessions need attention */
export function isAttentionStatus(status: HubStatus): boolean {
  return status === 'waiting' || status === 'errored' || status === 'stopped-err'
}
```

**Key constraint:** `HubStatus` type already includes all three attention statuses. No type change needed. The function accepts a `HubStatus`, not a `SessionInfo` — callers must call `deriveHubStatus(s)` first.

---

### `frontend/src/components/Hub/SessionCard.tsx` (component, request-response)

**Analog:** The existing `hub-card--dim` / `hub-card--dragging` modifier pattern + STATUS_CONFIG icon + ROW 1 render

**Imports pattern — add `BellAlertIcon`** (lines 1-14, add to the existing heroicons import block):
```typescript
import {
  ArrowPathIcon,
  CheckCircleIcon,
  PauseCircleIcon,
  ExclamationCircleIcon,
  StopCircleIcon,
  ComputerDesktopIcon,
  GlobeAltIcon,
  EyeIcon,
  Bars3Icon,
  EllipsisHorizontalIcon,
  BellAlertIcon,          // ADD — ATTN-01 attention icon
} from '@heroicons/react/24/outline'
```

Also add `isAttentionStatus` to the hubStatus import (line 17-18):
```typescript
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'
import type { HubStatus } from '../../lib/hubStatus'
```

**Props addition pattern** (lines 83-99 — copy the optional-prop style of `onRename`, `onOpenSession`):
```typescript
export interface SessionCardProps {
  session: SessionInfo
  onRename?: (id: string, name: string) => void
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  previewLines?: string[]
  groupDefs?: HubGroupDef[]
  onAssignGroup?: (memberKey: string, groupId: string) => void
  /** ATTN-01: true when isAttentionStatus(deriveHubStatus(session)) is true */
  isAttention?: boolean   // ADD
}
```

**className modifier pattern** (line 216 — copy the exact multi-modifier pattern):
```typescript
// EXISTING:
className={`hub-card${hubStatus === 'stopped-ok' ? ' hub-card--dim' : ''}${isDragging ? ' hub-card--dragging' : ''}`}

// NEW (extend to array-filter pattern for readability with three modifiers):
className={[
  'hub-card',
  hubStatus === 'stopped-ok' ? 'hub-card--dim' : '',
  isDragging ? 'hub-card--dragging' : '',
  isAttention ? 'hub-card--attention' : '',
].filter(Boolean).join(' ')}
```

**aria-label extension** (line 161):
```typescript
// EXISTING:
const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}`

// NEW:
const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}${isAttention ? ', needs attention' : ''}`
```

**ROW 1 BellAlertIcon insertion** (lines 285-304 — insert left of existing `hub-card__status-indicator`):
```tsx
<div className="hub-card__row1">
  {/* ATTN-01: attention icon — inline left of status icon; COLORBLIND-SAFE: BellAlertIcon carries state */}
  {isAttention && (
    <span className="hub-card__attn-icon" aria-label="Needs attention">
      {/* CRITICAL: NO Tailwind w-4 h-4 — size via CSS rule .hub-card__attn-icon svg { width:16px } */}
      <BellAlertIcon aria-hidden="true" />
    </span>
  )}
  <span className="hub-card__status-indicator">
    <Icon
      className={`hub-card__status-icon${spin ? ' hub-card__status-icon--spin' : ''}`}
      aria-label={displayLabel}
    />
    <span className="hub-card__status-label">{displayLabel}</span>
  </span>
  ...existing InlineSessionName and badge...
</div>
```

**Pitfall:** `BellAlertIcon` renders at 24px by default. Do NOT add `style={{ width:'16px', height:'16px' }}` inline — instead add the CSS rule `.hub-card__attn-icon svg { width: 16px; height: 16px }` to style.css, matching the established pattern at style.css line 4710-4714 (`.hub__group-sidebar-toggle svg`) and line 4716-4720 (`.hub__group-sidebar-item__needs-input-badge svg`).

---

### `frontend/src/components/Hub/SessionCardGrid.tsx` (component, transform)

**Analog:** `groupByNamedGroups` + `groupByWorkDir` group-building loops (lines 14-53); listitem render (lines 117-148, 156-184)

**Import addition** (lines 1-4 — add isAttentionStatus + deriveHubStatus):
```typescript
import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { SessionCard } from './SessionCard'
import { memberKey, type HubGroupDef } from '../../lib/hubGroups'
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'  // ADD
```

**Props addition** (lines 72-85 — add `attentionIds` set or thread per-session `isAttention` via existing session loop):
```typescript
export interface SessionCardGridProps {
  sessions: SessionInfo[]
  onRename: (id: string, name: string) => void
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  groupDefs?: HubGroupDef[]
  previewTails?: Map<string, string[]>
  onAssignGroup?: (memberKey: string, groupId: string) => void
  /** ATTN-02: set of session IDs currently in attention state (live, not debounced) */
  attentionIds?: Set<string>   // ADD (planner may prefer deriving inline from session status)
}
```

**sortSessionsForDisplay helper** — add after line 68 (after `basename`), before the Props block:
```typescript
/* ATTN-02: float-to-top sort within each group — stable; attention before non-attention */
function sortSessionsForDisplay(sessions: SessionInfo[]): SessionInfo[] {
  return [...sessions].sort((a, b) => {
    const aAttn = isAttentionStatus(deriveHubStatus(a)) ? 0 : 1
    const bAttn = isAttentionStatus(deriveHubStatus(b)) ? 0 : 1
    return aAttn - bAttn
  })
}
```

**Application inside groupByNamedGroups** (lines 37-53) — apply sort at the point where each group's session array is assembled:
```typescript
// In the final loop of groupByNamedGroups, after all sessions are distributed:
// Sort each group's sessions: attention first, stable
for (const entry of result.values()) {
  entry.sessions = sortSessionsForDisplay(entry.sessions)
}
return result
```

**Application inside groupByWorkDir** (lines 14-23) — same: apply `sortSessionsForDisplay` to each group's array before returning.

**FLIP animation hook** — add as a module-scope function before `SessionCardGrid` component (no direct analog in codebase; pattern is standard FLIP via `useLayoutEffect`):
```typescript
/* ATTN-02: FLIP animation hook — measures positions before/after sort, animates with transform */
/* ATTN-02: reorder animation: FLIP pattern, 300ms ease; suppressed under prefers-reduced-motion */
function useFLIPAnimation(enabled: boolean) {
  const nodeMap = React.useRef<Map<string, HTMLElement>>(new Map())
  const prevPositions = React.useRef<Map<string, DOMRect>>(new Map())

  const registerNode = React.useCallback((id: string, el: HTMLElement | null) => {
    if (el) nodeMap.current.set(id, el)
    else nodeMap.current.delete(id)
  }, [])

  const capturePositions = React.useCallback(() => {
    if (!enabled) return
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    const snap = new Map<string, DOMRect>()
    for (const [id, el] of nodeMap.current) snap.set(id, el.getBoundingClientRect())
    prevPositions.current = snap
  }, [enabled])

  const playFLIP = React.useCallback(() => {
    if (!enabled) return
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    for (const [id, el] of nodeMap.current) {
      const prev = prevPositions.current.get(id)
      if (!prev) continue
      const next = el.getBoundingClientRect()
      const deltaY = prev.top - next.top
      if (Math.abs(deltaY) < 1) continue
      el.style.transform = `translateY(${deltaY}px)`
      el.style.transition = 'none'
      requestAnimationFrame(() => {
        el.style.transform = ''
        el.style.transition = 'transform 300ms ease'
        el.addEventListener('transitionend', () => { el.style.transition = '' }, { once: true })
      })
    }
  }, [enabled])

  return { registerNode, capturePositions, playFLIP }
}
```

**useLayoutEffect callsite** in `SessionCardGrid` — the FLIP hook integrates with the component via `useLayoutEffect`. The `capturePositions` call must happen BEFORE React re-renders (in a `useLayoutEffect` that runs when the debounced sort key changes), and `playFLIP` AFTER the DOM is updated. See RESEARCH.md Pitfall 4 for the reason `useEffect` is wrong for capture. The `listitem` div receives a `ref` callback from `registerNode`.

**Simplified fallback:** If FLIP complexity is high-risk, omit it. The 1-second debounce (in HubPanel) is the primary "non-jarring" mechanism. FLIP is enhancement, not release-blocking.

---

### `frontend/src/components/Hub/GroupSidebar.tsx` (component, CRUD)

**Analog:** `GroupCounts` interface (lines 17-21), `computeCounts` (lines 23-36), `computeGlobalCounts` (lines 38-47), `NeedsInputBadge` component (lines 55-68), badge render condition (line 137-139)

**Import addition** (lines 1-13 — add `BellAlertIcon` and `isAttentionStatus`):
```typescript
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  PauseCircleIcon,
  BellAlertIcon,          // ADD — ATTN-06
} from '@heroicons/react/24/outline'
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'  // extend existing import
```

**GroupCounts extension** (lines 17-21 — add `attention` field):
```typescript
// EXISTING:
interface GroupCounts {
  running: number
  total: number
  waiting: number
}

// NEW:
interface GroupCounts {
  running: number
  total: number
  waiting: number
  attention: number  // ATTN-06: superset of waiting (waiting | errored | stopped-err)
}
```

**computeCounts extension** (lines 23-36 — add attention counter, copy the existing `waiting` counter pattern):
```typescript
function computeCounts(sessions: SessionInfo[], memberKeys: Set<string>): GroupCounts {
  let running = 0, total = 0, waiting = 0, attention = 0  // ADD attention
  for (const s of sessions) {
    const key = memberKey(s.name, s.workDir)
    if (!memberKeys.has(key)) continue
    total++
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
    if (isAttentionStatus(st)) attention++             // ADD
  }
  return { running, total, waiting, attention }        // ADD attention
}
```

**computeGlobalCounts extension** (lines 38-47 — same pattern for the "All" item):
```typescript
function computeGlobalCounts(sessions: SessionInfo[]): GroupCounts {
  let running = 0, waiting = 0, attention = 0          // ADD attention
  for (const s of sessions) {
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
    if (isAttentionStatus(st)) attention++             // ADD
  }
  return { running, total: sessions.length, waiting, attention }  // ADD attention
}
```

**NeedsInputBadge condition inversion BUG — CONFIRMED** (line 137):
```tsx
// EXISTING (line 137) — WRONG: renders when NOT collapsed (i.e. expanded), but per UI-SPEC
// the badge should show when COLLAPSED (when cards are hidden and badge is the only signal):
{!collapsed && counts.waiting > 0 && (
  <NeedsInputBadge count={counts.waiting} />
)}

// FIX in Phase 133 — invert the condition AND add attention badge priority:
{collapsed && counts.attention > 0 && (
  <span
    className="hub__group-sidebar-item__attn-badge"
    aria-label={counts.attention === 1 ? '1 session needs attention' : `${counts.attention} sessions need attention`}
  >
    {/* COLORBLIND-SAFE: attn badge dark hex #e0af68 — reinforcement only; BellAlertIcon carries state */}
    {/* CRITICAL: NO Tailwind — size via CSS rule .hub__group-sidebar-item__attn-badge svg */}
    <BellAlertIcon aria-hidden="true" />
    <span className="hub__group-sidebar-item__attn-badge--count">{counts.attention}</span>
  </span>
)}
{collapsed && counts.attention === 0 && counts.waiting > 0 && (
  <NeedsInputBadge count={counts.waiting} />
)}
```

**Flag:** The `NeedsInputBadge` component in `GroupSidebarItem` also renders name and count at lines 129-135. Those render when `!collapsed` too. The group name and count only show in expanded state — that's correct. Only the badge condition is inverted.

**GroupSidebarProps passthrough** — `GroupSidebarItem` receives `counts: GroupCounts` (line 74). Because `GroupCounts` now includes `attention`, no new prop threading is needed — the item already receives the full counts struct. The parent `GroupSidebar` calls `computeCounts` and `computeGlobalCounts`, which now return `attention`.

---

### `frontend/src/components/Hub/HubPanel.tsx` (component, event-driven)

**Analog:** `usePreviewPoller` hook (lines 58-112) for the `useRef + setInterval + cleanup` debounce-adjacent pattern; `allSessions` merge (line 214) for threading props; `SessionCardGrid` call (lines 274-283) for prop addition

**Import addition** (lines 1-19 — add `isAttentionStatus`):
```typescript
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'  // extend existing import
```

**useDebouncedValue hook** — add as module-scope function after `usePreviewPoller` (after line 112), before Props block. Copies the `useRef + clearTimeout + setTimeout` pattern established in `usePreviewPoller`'s `setInterval`/cleanup structure:
```typescript
/* ATTN-04: debounce hook — useRef + setTimeout; controls sort ORDER only, not card content */
function useDebouncedValue<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = React.useState<T>(value)
  const timerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  React.useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setDebouncedValue(value), delay)
    return () => { if (timerRef.current !== null) clearTimeout(timerRef.current) }
  }, [value, delay])

  return debouncedValue
}
```

**Debounced sort key** — add after `visibleSessions` derivation (after line 238):
```typescript
// ATTN-04: attention sort key — debounced so position changes don't fire on every 3s poll
/* ATTN-02: float-to-top sort is within-group, not a global lane; debounce window 1000ms */
const attentionSortKey = allSessions
  .map((s) => `${s.id}:${isAttentionStatus(deriveHubStatus(s)) ? '1' : '0'}`)
  .join(',')
const debouncedSortKey = useDebouncedValue(attentionSortKey, 1000)
// debouncedSortKey is passed to SessionCardGrid to trigger sort only after debounce settles.
// Per-card isAttention uses LIVE (non-debounced) status — border and icon update immediately.
```

**attentionIds derivation** — live (not debounced), for threading to SessionCardGrid:
```typescript
// ATTN-01: live attention set — NOT debounced; card border/icon updates immediately
const attentionIds = new Set(
  allSessions
    .filter((s) => isAttentionStatus(deriveHubStatus(s)))
    .map((s) => s.id)
)
```

**SessionCardGrid call extension** (lines 274-283 — add new props alongside existing props):
```tsx
<SessionCardGrid
  sessions={visibleSessions}
  onRename={onRename}
  onOpenSession={onOpenSession}
  groupDefs={groupDefs.length > 0 ? groupDefs : undefined}
  previewTails={previewTails}
  onAssignGroup={handleAssignGroup}
  attentionIds={attentionIds}          // ADD — live attention set
  debouncedSortKey={debouncedSortKey}  // ADD — triggers debounced reorder in grid
/>
```

**Pitfall:** `debouncedSortKey` is passed as a trigger for the FLIP animation's `useLayoutEffect`. The grid does NOT use the string value directly — it only uses it as a dep for the effect. The actual sort uses live `isAttentionStatus` calls inside `sortSessionsForDisplay`.

---

### `frontend/src/style.css` (config)

**Analog 1 — Token block pattern** (lines 4095-4135 `:root`, lines 4138-4179 `[data-ui-theme="light"]`)
Copy the exact inline-comment format for new tokens — append at end of each block before the closing `}`:
```css
/* === Phase 133: Attention + Pulse tokens === */
/* COLORBLIND-SAFE: attn border dark hex #e0af68 — reinforcement only; BellAlertIcon + pulse shape carries state */
--hub-attn-border: #e0af68;
--hub-attn-border-glow: rgba(224,175,104,0.35);
--hub-attn-icon-color: #e0af68;
--hub-attn-badge-bg: rgba(224,175,104,0.18);
--hub-attn-badge-text: #e0af68;
--hub-attn-static-border: #e0af68;
```

Light theme (append to `[data-ui-theme="light"]` before its closing `}`):
```css
/* COLORBLIND-SAFE: attn border light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
--hub-attn-border: #b45309;
--hub-attn-border-glow: rgba(180,83,9,0.25);
--hub-attn-icon-color: #b45309;
--hub-attn-badge-bg: rgba(180,83,9,0.14);
--hub-attn-badge-text: #b45309;
--hub-attn-static-border: #b45309;
```

**Analog 2 — `hub-card--dim` / `hub-card--dragging` modifier pattern** (lines 4301-4306, 4848-4852):
Copy the BEM modifier pattern for `.hub-card--attention`. Add after `.hub-card--dragging` (line 4852):
```css
/* ATTN-01: attention card — pulsing border + glow (colorblind-safe: BellAlertIcon carries state) */
/* COLORBLIND-SAFE: attn border dark hex #e0af68 — reinforcement only; BellAlertIcon + pulse shape carries state */
/* COLORBLIND-SAFE: attn border light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
.hub-card--attention {
  border-color: var(--hub-attn-border);
}
```

**Analog 3 — `@media (prefers-reduced-motion: no-preference)` guard** (lines 4359-4363):
```css
/* Running: spin animation — MUST be inside prefers-reduced-motion guard (Motion Contract) */
@media (prefers-reduced-motion: no-preference) {
  .hub-card__status-icon--spin {
    animation: hub-spin 0.8s linear infinite;
  }
}
```
Copy this exact guard structure for the pulse animation. The `animation:` declaration on `.hub-card--attention` MUST be inside this guard. Also extend `.hub-card` transition inside the guard (overriding the line 4288 100ms base):
```css
/* A11Y-03: pulse wrapped in @media (prefers-reduced-motion: no-preference); static border fallback */
@media (prefers-reduced-motion: no-preference) {
  .hub-card--attention {
    animation: hub-attn-pulse 2s ease-in-out infinite;
  }
  /* ATTN-03: attention clear — override 100ms base transition for motion-enabled users only */
  .hub-card {
    transition: border-color 400ms ease, box-shadow 400ms ease, background 100ms ease;
  }
  /* Keep amber border on hover — prevents .hub-card:hover (line 4292) from overriding attention border */
  .hub-card--attention:hover {
    border-color: var(--hub-attn-border);
  }
}
```

**Analog 4 — `@keyframes hub-spin`** (lines 4367-4370):
```css
@keyframes hub-spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}
```
Copy this keyframe declaration pattern (outside any media query — keyframe is declared globally, the `animation:` property reference is gated):
```css
@keyframes hub-attn-pulse {
  0%   { border-color: var(--hub-attn-border); box-shadow: 0 0 0 0 var(--hub-attn-border-glow); }
  50%  { border-color: var(--hub-attn-border); box-shadow: 0 0 0 4px var(--hub-attn-border-glow); }
  100% { border-color: var(--hub-attn-border); box-shadow: 0 0 0 0 var(--hub-attn-border-glow); }
}
```

**Analog 5 — `prefers-reduced-motion: reduce` static fallback** (lines 1743-1748 and multiple later instances):
```css
@media (prefers-reduced-motion: reduce) {
  .hub-card--attention {
    border-color: var(--hub-attn-static-border);
    box-shadow: none;
    animation: none;
  }
}
```

**Analog 6 — `.hub__group-sidebar-item__needs-input-badge` + svg sizing** (lines 4792-4805, 4716-4720):
```css
/* COLORBLIND-SAFE: needs-input badge dark hex #f59e0b — reinforcement only; PauseCircleIcon carries the state */
.hub__group-sidebar-item__needs-input-badge {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  background: var(--hub-needs-input-badge-bg);
  color: var(--hub-needs-input-badge-text);
  border-radius: 8px;
  padding: 0 4px;
  height: 16px;
  font-size: 11px;
  flex-shrink: 0;
}
.hub__group-sidebar-item__needs-input-badge svg {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}
```
Copy this exact shape for `.hub__group-sidebar-item__attn-badge` — same dimensions, different color tokens, different gap (4px vs 2px per spec):
```css
/* ATTN-06: collapsed-group attention badge */
/* COLORBLIND-SAFE: attn badge dark hex #e0af68 — reinforcement only; BellAlertIcon carries state */
/* COLORBLIND-SAFE: attn badge light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
.hub__group-sidebar-item__attn-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;
  height: 16px;
  border-radius: 8px;
  background: var(--hub-attn-badge-bg);
  color: var(--hub-attn-badge-text);
  font-size: 11px;
  line-height: 1.0;
  font-family: system-ui, -apple-system, "Segoe UI", sans-serif;
  flex-shrink: 0;
}
/* CRITICAL: explicit CSS sizing — no Tailwind in this project */
.hub__group-sidebar-item__attn-badge svg {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}
```

**Analog 7 — icon wrapper pattern** (lines 4710-4714, `.hub__group-sidebar-toggle svg`):
Copy for `.hub-card__attn-icon` and its svg rule:
```css
.hub-card__attn-icon {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  color: var(--hub-attn-icon-color);
}
/* CRITICAL: size the icon with explicit CSS — no Tailwind in this project */
.hub-card__attn-icon svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
```

---

## Shared Patterns

### Colorblind-Safe Icon + Comment Convention
**Source:** `frontend/src/components/Hub/SessionCard.tsx` lines 29-49 (STATUS_CONFIG comments); `frontend/src/style.css` lines 4792-4793
**Apply to:** Every new CSS rule touching color, every new TSX element that renders an icon
```
/* COLORBLIND-SAFE: <element> dark hex #<hex> — reinforcement only; <Icon>/<text> carries the state */
/* COLORBLIND-SAFE: <element> light hex #<hex> — WCAG AA X.X:1 on white; icon carries state */
```

### No-Tailwind Icon Sizing Rule
**Source:** `frontend/src/style.css` lines 4707-4714 (Phase 132 UAT fix comment + `.hub__group-sidebar-toggle svg`)
**Apply to:** Every new Heroicons SVG (`BellAlertIcon` in card ROW 1 + sidebar badge)
```css
/* Phase 132 UAT fix: this project has NO Tailwind — `w-N h-N` utility classes are no-ops */
.<wrapper> svg {
  width: <Npx>;
  height: <Npx>;
  flex-shrink: 0;
}
```
Never use `className="w-4 h-4"` or `style={{ width:'16px' }}` on the icon element directly — use the CSS wrapper rule.

### reduced-motion Guard Pattern
**Source:** `frontend/src/style.css` lines 4358-4363 (`hub-spin` guard)
**Apply to:** Every new `animation:` or FLIP `transition:` property in Phase 133 CSS
```css
@media (prefers-reduced-motion: no-preference) {
  /* animation/transition declarations here */
}
/* Keyframe declarations go OUTSIDE the query (unused keyframes don't run) */
@keyframes <name> { ... }
@media (prefers-reduced-motion: reduce) {
  /* static fallback here */
}
```

### Phase-Tagged CSS Comment Convention
**Source:** `frontend/src/style.css` lines 4119-4121, 4163-4164
**Apply to:** New token blocks and animation rules
```css
/* === Phase 133: Attention + Pulse <description> === */
```

### Prop Threading Pattern (HubPanel → SessionCardGrid → SessionCard)
**Source:** `frontend/src/components/Hub/HubPanel.tsx` lines 274-283; `frontend/src/components/Hub/SessionCardGrid.tsx` lines 133-142
**Apply to:** `isAttention` / `attentionIds` prop threading
Each layer destructures its own props and passes relevant data down. `HubPanel` computes `attentionIds: Set<string>`. `SessionCardGrid` receives it and passes `isAttention={attentionIds?.has(s.id)}` to each `SessionCard`. `SessionCard` uses `isAttention` for className + BellAlertIcon render.

---

## No Analog Found

No files are completely without analog. All six files modify existing components where the same file contains the closest pattern.

The FLIP animation hook (`useFLIPAnimation`) has no prior instance in this codebase — it is a net-new pattern. The closest analog is the `usePreviewPoller` hook's `useRef + cleanup` structure. The FLIP implementation follows the well-known FLIP technique (zero-dependency, native DOM APIs). See RESEARCH.md Pattern 4 for the full implementation.

---

## Critical Flag: NeedsInputBadge Condition Inversion

**Location:** `frontend/src/components/Hub/GroupSidebar.tsx` line 137

**Current code:**
```tsx
{!collapsed && counts.waiting > 0 && (
  <NeedsInputBadge count={counts.waiting} />
)}
```

**Bug:** `!collapsed` means the badge shows when EXPANDED (cards are visible). The UI-SPEC for both Phase 132 and Phase 133 specify the badge shows when COLLAPSED (cards are hidden, badge is the only signal). This must be corrected in Phase 133 as part of adding the attention badge. Both badges (needs-input and attention) must flip to `collapsed` condition.

---

## Metadata

**Analog search scope:** `frontend/src/components/Hub/`, `frontend/src/lib/`, `frontend/src/style.css`
**Files read:** 6 source files (hubStatus.ts, SessionCard.tsx, SessionCardGrid.tsx, GroupSidebar.tsx, HubPanel.tsx, style.css)
**Pattern extraction date:** 2026-06-16
