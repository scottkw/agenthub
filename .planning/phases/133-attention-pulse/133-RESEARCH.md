# Phase 133: Attention + Pulse - Research

**Researched:** 2026-06-16
**Domain:** React/TypeScript frontend UI, CSS animation (FLIP, keyframes), BEM CSS
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting.
Extend the Phase 131/132 Hub components (SessionCard attention treatment, SessionCardGrid float-to-top
ordering, GroupSidebar attention badge on collapsed groups, hubStatus derivation). Reuse the
established `--hub-*` tokens, reduced-motion guard, and colorblind-safe icon+text pattern.

### Critical Locked Constraints (from CONTEXT.md)
- User is COLORBLIND (release-blocking): attention conveyed by pulsing border + distinct icon + position, NEVER color alone.
- Pulse animation MUST respect prefers-reduced-motion (static border fallback, icon+position still active).
- Float-to-top reordering MUST be debounced (not on every poll tick) and position changes animate smoothly.
- ATTN-03: clearing is STATUS-DRIVEN via poll — no modal coupling. Status change → pulse clears automatically. No Phase 134 dependency.

### Claude's Discretion
All implementation choices.

### Deferred Ideas (OUT OF SCOPE)
None — discuss skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ATTN-01 | A session with status `waiting`, `errored`, or non-zero exit is flagged as needing attention | Add `isAttentionStatus()` to `hubStatus.ts`; returns true for `waiting \| errored \| stopped-err` |
| ATTN-02 | An attention card shows a pulsing animated highlighted border plus an attention icon | `.hub-card--attention` modifier + `@keyframes hub-attn-pulse` in CSS; `BellAlertIcon` in ROW 1 left of status icon |
| ATTN-03 | When cards overflow the viewport, attention cards sort to the top (within group) | Debounced sort in HubPanel; FLIP animation in SessionCardGrid; clearing is poll-driven via `isAttentionStatus()` |
| ATTN-04 | Reordering on status change is debounced and position changes are animated (non-jarring) | `useDebouncedValue` hook (useRef+setTimeout); FLIP animation pattern; 1s debounce, 300ms ease |
| ATTN-05 | Resolving a `waiting` session inside its modal clears that card's attention state | Poll-driven: when next poll returns non-attention status, `isAttentionStatus()` returns false → treatment clears. No modal coupling. |
| ATTN-06 | A collapsed group containing an attention card shows an attention badge on its header | Extend `GroupCounts` + `computeCounts` in GroupSidebar; replace needs-input badge with attention badge when `attnCount > 0` and group is collapsed |
</phase_requirements>

---

## Summary

Phase 133 is a pure frontend change — no backend gap. The attention state derives entirely from
`deriveHubStatus()` which already returns `'waiting'`, `'errored'`, and `'stopped-err'` — all three
statuses that define the attention set. No new backend RPC, HTTP route, or daemon field is required.

The phase adds four cooperating pieces:
1. **`isAttentionStatus()`** helper added to `frontend/src/lib/hubStatus.ts` — single canonical predicate
2. **`SessionCard`** attention treatment: `BellAlertIcon` inline in ROW 1 + `.hub-card--attention` CSS modifier (pulsing border + glow)
3. **`SessionCardGrid` / `HubPanel`** debounced float-to-top sort within each group, animated via FLIP
4. **`GroupSidebar`** collapsed-group attention badge replacing the needs-input badge when `attnCount > 0`

The most implementation-critical discovery is how to do FLIP reorder animation in React without any
third-party library. The pattern uses `useLayoutEffect` + `getBoundingClientRect()` before/after the
sort, then injects `transform: translateY()` with a short `transition`, then removes the transform to
play the animation. This is zero-dependency and matches the established codebase pattern (native DOM,
no animation library).

The existing `.hub-card` CSS already has `transition: border-color 100ms ease, background 100ms ease`
— the attention clear animation MUST extend this to `400ms ease` (per UI-SPEC) under
`prefers-reduced-motion: no-preference`. The attention pulse keyframe must be wrapped in
`@media (prefers-reduced-motion: no-preference)` per the established Phase 131/132 motion guard pattern.

**No backend gap this phase.** `deriveHubStatus()` already returns all three attention statuses from
fields populated by Phase 131/132. Attention state is fully derivable from the existing poll.

**Primary recommendation:** Add `isAttentionStatus()` to hubStatus.ts first; use it everywhere so
attention logic never diverges from a single source of truth.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Attention predicate | Frontend (`hubStatus.ts`) | — | Pure derivation from existing `HubStatus` enum — no backend data needed |
| Pulse animation | Frontend CSS (`style.css`) | — | CSS keyframe on `.hub-card--attention` modifier; driven by class presence |
| BellAlertIcon rendering | Frontend (`SessionCard.tsx`) | — | Presentational; receives `isAttention` prop |
| Debounced sort key | Frontend (`HubPanel.tsx`) | — | HubPanel owns the poll; debounce lives alongside the poll logic |
| Float-to-top sort | Frontend (`HubPanel.tsx` + `SessionCardGrid.tsx`) | — | Sort applied before passing to grid; FLIP measured in grid |
| FLIP animation | Frontend (`SessionCardGrid.tsx`) | — | useLayoutEffect + getBoundingClientRect; requires DOM access to measure |
| Collapsed-group badge | Frontend (`GroupSidebar.tsx`) | — | Extends `computeCounts` to include `attention` count |
| Status-driven clear | Frontend (`HubPanel.tsx` poll) | — | Poll result drives `isAttentionStatus()` re-derivation per tick |

---

## Standard Stack

### Core (all already installed — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | ^19.2.4 | Component rendering | Already the app's UI framework [VERIFIED: frontend/package.json] |
| TypeScript | 5.9.x | Type safety | Already the project language [VERIFIED: frontend/package.json] |
| @heroicons/react | ^2.2.0 | `BellAlertIcon` (attention icon, 16px in ROW 1; 12px in badge) | Already installed; `BellAlertIcon` confirmed present in `@heroicons/react/24/outline` [VERIFIED: node_modules inspection] |
| CSS Custom Properties | native | `--hub-attn-*` tokens | Hand-rolled BEM + CSS custom property pattern established in Phase 131/132 [VERIFIED: style.css] |

### No New Packages Required

No new npm packages are needed. FLIP animation is implemented with native DOM APIs
(`getBoundingClientRect`, `useLayoutEffect`, `requestAnimationFrame`). Debounce is implemented
with `useRef + setTimeout`. Both are zero-dependency approaches consistent with the codebase style.

---

## Package Legitimacy Audit

No external packages are installed in Phase 133. All capabilities use existing infrastructure.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Backend Gap Confirmation

**There is NO backend gap for Phase 133.** [VERIFIED: codebase inspection]

| Status | Source Field | Already in `deriveHubStatus()`? |
|--------|-------------|--------------------------------|
| `waiting` | `session.status === 'waiting'` | YES — returned directly |
| `errored` | `session.status === 'errored'` | YES — returned directly |
| `stopped-err` | `session.state === 'stopped' && (session.exitCode ?? 0) !== 0` | YES — explicit branch |

`isAttentionStatus()` is a pure predicate over the existing `HubStatus` enum. The poll already
delivers all required data. The Phase 134 modal (when it resolves a `waiting` session) will cause
a status transition that the next poll detects — no special signal or new endpoint is needed.

**Why this matters for the plan:** Wave 0 has NO Go backend tasks. All work is frontend-only.

---

## Architecture Patterns

### System Architecture Diagram

```
HubPanel (owns poll every 3s)
    │
    ├─ derives isAttention per session from isAttentionStatus(deriveHubStatus(s))
    │
    ├─ builds debouncedSortKey (1s debounce on attention membership change)
    │   └─ useDebouncedValue(sessions.map(s => `${s.id}:${isAttentionStatus(...)}`).join(','), 1000)
    │
    ├─ passes visibleSessions (already attention-sorted within each group) to SessionCardGrid
    │   └─ sort is applied in HubPanel (or SessionCardGrid — planner decides; research recommends HubPanel)
    │
    ├─ passes isAttention prop per card through SessionCardGrid → SessionCard
    │   └─ SessionCard renders BellAlertIcon + .hub-card--attention when isAttention=true
    │
    └─ passes allSessions to GroupSidebar
        └─ GroupSidebar.computeCounts() adds attention count
            └─ GroupSidebarItem: collapsed + attnCount > 0 → attention badge replaces needs-input badge

CSS layer (style.css):
    └─ .hub-card--attention: border-color var(--hub-attn-border)
    └─ @keyframes hub-attn-pulse: box-shadow expand/contract 2s ease-in-out infinite
    └─ @media (prefers-reduced-motion: reduce) .hub-card--attention: static border, no animation
    └─ .hub-card: transition extended to border-color 400ms ease for attention clear
    └─ new --hub-attn-* tokens in :root and [data-ui-theme="light"]

FLIP animation (SessionCardGrid):
    └─ Before sort: measure each card's getBoundingClientRect().top
    └─ After sort: measure again; compute delta; apply transform: translateY(delta)
    └─ requestAnimationFrame: remove transform (triggering transition play)
    └─ On reduce-motion: skip FLIP, allow instant snap
```

### Recommended Project Structure

No new files. Phase 133 modifies:

```
frontend/src/
├─ lib/
│   └─ hubStatus.ts             (add isAttentionStatus() export)
├─ components/Hub/
│   ├─ SessionCard.tsx          (add isAttention prop + BellAlertIcon + .hub-card--attention modifier + aria-label suffix)
│   ├─ SessionCardGrid.tsx      (add FLIP animation hook; receive sorted sessions per group)
│   ├─ GroupSidebar.tsx         (extend GroupCounts + computeCounts; add attention badge; replace needs-input badge when attnCount>0)
│   └─ HubPanel.tsx             (derive isAttention per session; debounced sort key; pass isAttention props)
└─ style.css                    (append --hub-attn-* tokens; hub-card--attention rules; keyframe; reduced-motion guard)
```

### Pattern 1: `isAttentionStatus()` — Single Predicate, All Consumers

**What:** One exported function in `hubStatus.ts` is the canonical definition of attention state.
**When to use:** Every place in the codebase that needs to know if a session needs attention.
**Why:** Prevents attention logic from diverging across SessionCard, SessionCardGrid, GroupSidebar, HubPanel.

```typescript
// Source: 133-UI-SPEC.md Attention Detection Contract [VERIFIED]
// Add to frontend/src/lib/hubStatus.ts after deriveHubStatus()

/* ATTN-01: canonical attention predicate — waiting, errored, or non-zero-exit sessions need attention */
export function isAttentionStatus(status: HubStatus): boolean {
  return status === 'waiting' || status === 'errored' || status === 'stopped-err'
}
```

All consumers call `isAttentionStatus(deriveHubStatus(session))`. No inline status checks.

### Pattern 2: Debounced Sort Key (HubPanel)

**What:** Debounce the ORDER in which cards appear, but NOT the card content (status icon, border, icon
all update immediately from live poll; only position changes are debounced).
**When to use:** Any time a reactive value changes on every poll tick but should not trigger a layout
change on every tick.

```typescript
// Source: 133-UI-SPEC.md ATTN-02 Debounce Window [VERIFIED]
// Pattern: useRef + setTimeout, returns debounced value.
// Add as a local hook inside HubPanel.tsx.

/* ATTN-02: float-to-top sort is within-group; debounce window 1000ms */
function useDebouncedValue<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = React.useState<T>(value)
  const timerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  React.useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)
    return () => {
      if (timerRef.current !== null) clearTimeout(timerRef.current)
    }
  }, [value, delay])

  return debouncedValue
}

// Usage in HubPanel:
const attentionSortKey = allSessions
  .map((s) => `${s.id}:${isAttentionStatus(deriveHubStatus(s)) ? '1' : '0'}`)
  .join(',')
const debouncedSortKey = useDebouncedValue(attentionSortKey, 1000)
```

The sort itself uses `debouncedSortKey` to determine ORDER; the individual card's `isAttention` prop
uses the live (non-debounced) status so the border and icon update immediately on the next poll.

### Pattern 3: Float-to-Top Sort (applied within each group)

**What:** Sort attention sessions before non-attention sessions within each group's session array.
Stable sort: sessions with equal attention weight preserve their original order.
**When to use:** Applied after the debounce settles, before rendering.

```typescript
// Source: 133-UI-SPEC.md ATTN-02 Sort Rule [VERIFIED]
/* ATTN-02: float-to-top sort within each group; stable: equal attention → original order preserved */
function sortSessionsForDisplay(sessions: SessionInfo[]): SessionInfo[] {
  return [...sessions].sort((a, b) => {
    const aAttn = isAttentionStatus(deriveHubStatus(a)) ? 0 : 1
    const bAttn = isAttentionStatus(deriveHubStatus(b)) ? 0 : 1
    return aAttn - bAttn
  })
}
```

**Implementation location decision for planner:** The sort can live in HubPanel (sort each group's
sessions before passing to the grid) OR in SessionCardGrid (sort each group's sessions before
rendering). Research recommends putting the sort in HubPanel alongside the debounce key, so
SessionCardGrid receives pre-sorted groups and only needs to implement the FLIP animation. This
separation of concerns makes testing easier.

### Pattern 4: FLIP Animation (SessionCardGrid)

**What:** FLIP (First/Last/Invert/Play) is the standard pattern for animating DOM reorders without
a third-party library. It works by capturing position BEFORE the sort, applying the sort, capturing
position AFTER, computing the delta, and animating from old to new using CSS `transform`.

**Why FLIP and not CSS order transition:** CSS `order` property changes are not animatable in
any browser. `transition: order` has no effect. FLIP is the correct solution.

**Implementation with useLayoutEffect:**

```typescript
// Source: FLIP animation pattern — no library, native React DOM APIs [ASSUMED — well-known pattern]
// Located in SessionCardGrid.tsx

/* ATTN-02: reorder animation: FLIP pattern, 300ms ease; suppressed under prefers-reduced-motion */
function useFLIPAnimation(sortedIds: string[], enabled: boolean) {
  // Map from session id → DOM element ref
  const nodeMap = React.useRef<Map<string, HTMLElement>>(new Map())

  // Snapshot positions BEFORE render (called before React's DOM update)
  const prevPositions = React.useRef<Map<string, DOMRect>>(new Map())

  // Register a node (called in the list item's ref callback)
  const registerNode = React.useCallback((id: string, el: HTMLElement | null) => {
    if (el) nodeMap.current.set(id, el)
    else nodeMap.current.delete(id)
  }, [])

  // Capture positions before the sort updates the DOM
  const capturePositions = React.useCallback(() => {
    if (!enabled) return
    const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (prefersReduced) return
    const snapshot = new Map<string, DOMRect>()
    for (const [id, el] of nodeMap.current) {
      snapshot.set(id, el.getBoundingClientRect())
    }
    prevPositions.current = snapshot
  }, [enabled])

  // Play the FLIP animation after the DOM update
  const playFLIP = React.useCallback(() => {
    if (!enabled) return
    const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (prefersReduced) return
    for (const [id, el] of nodeMap.current) {
      const prev = prevPositions.current.get(id)
      if (!prev) continue
      const next = el.getBoundingClientRect()
      const deltaY = prev.top - next.top
      if (Math.abs(deltaY) < 1) continue  // no perceptible movement

      // Invert: jump to old position
      el.style.transform = `translateY(${deltaY}px)`
      el.style.transition = 'none'

      // Play: animate to natural position
      requestAnimationFrame(() => {
        el.style.transform = ''
        el.style.transition = 'transform 300ms ease'
        // Clean up after animation completes
        const onEnd = () => {
          el.style.transition = ''
          el.removeEventListener('transitionend', onEnd)
        }
        el.addEventListener('transitionend', onEnd, { once: true })
      })
    }
  }, [enabled])

  return { registerNode, capturePositions, playFLIP }
}
```

**Key callsites in SessionCardGrid:**
- Before React re-renders with new sort: `capturePositions()` (called via `useLayoutEffect` or
  `useEffect` that runs synchronously before the next paint when the sort key changes)
- After React applies the new DOM order: `playFLIP()` (in the same effect, after state update)

**Simpler alternative for the planner:** If the FLIP hook adds too much complexity, an acceptable
fallback is to skip position animation entirely and rely on the CSS `transition: border-color 400ms`
for the attention CLEAR, while the REORDER snaps instantly. The UI-SPEC says FLIP is recommended
but does not make it release-blocking — the non-jarring requirement is primarily satisfied by the
1-second debounce (preventing reorders on every 3s poll).

**CRITICAL — Reduced-motion gate:** The FLIP animation MUST be suppressed when
`window.matchMedia('(prefers-reduced-motion: reduce)').matches` returns true. Under reduced motion,
the sort still happens (cards still float to top); only the translate animation is skipped.

### Pattern 5: Attention Card Treatment (SessionCard + CSS)

**What:** When `isAttention={true}`, SessionCard adds `.hub-card--attention` to the article element,
renders `BellAlertIcon` inline in ROW 1 to the left of the status icon, and appends ", needs attention"
to the card's `aria-label`.

```tsx
// Source: 133-UI-SPEC.md ATTN-01 + Accessibility Contract [VERIFIED]

/* ATTN-01: attention card — BellAlertIcon inline in ROW 1, left of status icon */
/* COLORBLIND-SAFE: attn border dark hex #e0af68 — reinforcement only; BellAlertIcon + pulse shape carries state */
import { BellAlertIcon } from '@heroicons/react/24/outline'

// In SessionCard props:
export interface SessionCardProps {
  // ... existing props ...
  /** ATTN-01: true when isAttentionStatus(deriveHubStatus(session)) is true */
  isAttention?: boolean
}

// In the article element's className:
className={[
  'hub-card',
  hubStatus === 'stopped-ok' ? 'hub-card--dim' : '',
  isDragging ? 'hub-card--dragging' : '',
  isAttention ? 'hub-card--attention' : '',
].filter(Boolean).join(' ')}

// In the aria-label:
const cardAriaLabel = `${name}, ${displayLabel}, ${cli}, ${originText}${isAttention ? ', needs attention' : ''}`

// ROW 1 — attention icon inserted LEFT of status icon:
<div className="hub-card__row1">
  {isAttention && (
    <span className="hub-card__attn-icon" aria-label="Needs attention">
      {/* COLORBLIND-SAFE: attn icon dark hex #e0af68 — BellAlertIcon is primary differentiator, not color */}
      <BellAlertIcon style={{ width: '16px', height: '16px' }} aria-hidden="true" />
    </span>
  )}
  <span className="hub-card__status-indicator">
    ...existing status icon + label...
  </span>
  ...
</div>
```

**CRITICAL — No Tailwind for icon sizing.** The codebase has NO Tailwind. `w-4 h-4` class names are
NO-OPS in this project (Phase 132 UAT bug). Size the BellAlertIcon with explicit CSS:
`style={{ width: '16px', height: '16px' }}` OR via the `.hub-card__attn-icon svg` rule in style.css.
[VERIFIED: project has no Tailwind; Phase 132 bug confirmed in CONTEXT.md]

### Pattern 6: Collapsed-Group Attention Badge (GroupSidebar)

**What:** Extend `GroupCounts` to include an `attention` count. When a group is collapsed AND
`attention > 0`, show the attention badge INSTEAD OF the needs-input badge.

```typescript
// Source: 133-UI-SPEC.md ATTN-06 Counts Computation [VERIFIED]
// Extend GroupSidebar.tsx

/* ATTN-06: collapsed-group attention badge uses BellAlertIcon (12px) + count; replaces needs-input badge */
interface GroupCounts {
  running: number
  total: number
  waiting: number    // existing — Phase 132 needs-input badge
  attention: number  // NEW — Phase 133 (superset: waiting | errored | stopped-err)
}

function computeCounts(sessions: SessionInfo[], memberKeys: Set<string>): GroupCounts {
  let running = 0, total = 0, waiting = 0, attention = 0
  for (const s of sessions) {
    const key = memberKey(s.name, s.workDir)
    if (!memberKeys.has(key)) continue
    total++
    const st = deriveHubStatus(s)
    if (st === 'running' || st === 'idle' || st === 'waiting') running++
    if (st === 'waiting') waiting++
    if (isAttentionStatus(st)) attention++
  }
  return { running, total, waiting, attention }
}
```

Badge render rule in `GroupSidebarItem` (when collapsed):
- `collapsed && counts.attention > 0` → render attention badge (replaces needs-input badge)
- `collapsed && counts.attention === 0 && counts.waiting > 0` → render needs-input badge (Phase 132 unchanged)

```tsx
// Source: 133-UI-SPEC.md ATTN-06 Badge Display Rules [VERIFIED]
/* ATTN-06: attention badge only shown when sidebar item is collapsed AND attnCount > 0 */
/* COLORBLIND-SAFE: attn badge dark hex #e0af68 — reinforcement only; BellAlertIcon carries state */
{collapsed && counts.attention > 0 && (
  <span
    className="hub__group-sidebar-item__attn-badge"
    aria-label={counts.attention === 1
      ? '1 session needs attention'
      : `${counts.attention} sessions need attention`}
  >
    <BellAlertIcon style={{ width: '12px', height: '12px' }} aria-hidden="true" />
    <span className="hub__group-sidebar-item__attn-badge--count">{counts.attention}</span>
  </span>
)}
{collapsed && counts.attention === 0 && counts.waiting > 0 && (
  <NeedsInputBadge count={counts.waiting} />
)}
```

### Anti-Patterns to Avoid

- **`w-4 h-4` / `w-3 h-3` Tailwind classes on icons:** These are NO-OPS — the project has no Tailwind.
  Use `style={{ width: '16px', height: '16px' }}` or a CSS rule on the wrapper class.
- **Coupling ATTN-03 clear to Phase 134 modal:** The attention treatment clears when
  `isAttentionStatus(deriveHubStatus(session))` returns false. The Phase 134 modal is not imported or
  referenced. Testing is done by simulating a status change (e.g., changing a session from `waiting` to
  `idle` in the mock data).
- **Debouncing card CONTENT (status icon, border) instead of only ORDER:** The border color and
  BellAlertIcon update on every poll (live). Only the card's POSITION within the grid is debounced.
- **Global attention lane across groups:** Forbidden per UI-SPEC. Float-to-top is WITHIN each group
  to preserve group context.
- **Per-card animation timers:** The FLIP animation is measured and played by the grid, not individual
  cards. Cards do not manage their own position transitions.
- **`animation:` declarations outside `prefers-reduced-motion: no-preference` guard:** All new
  animations (pulse keyframe, FLIP transition) MUST be wrapped. Phase 131 established this pattern with
  the `hub-spin` keyframe.
- **Inline `border-color` override on `.hub-card:hover`:** The `.hub-card:hover` rule sets
  `border-color: var(--hub-border-hover)`. The `.hub-card--attention` rule must have higher specificity
  OR the hover rule must NOT override the attention border. Solution: use `.hub-card--attention:hover`
  to keep `--hub-attn-border` even on hover (attention takes precedence over hover state). [ASSUMED
  based on CSS specificity rules — verify at implementation]

---

## CSS Additions Required

### New Tokens (append to existing `:root` and `[data-ui-theme="light"]` blocks)

```css
/* === Phase 133: Attention + Pulse tokens === */
/* ATTN-01: attention card — pulsing border + glow (colorblind-safe: BellAlertIcon carries state) */
/* COLORBLIND-SAFE: attn border dark hex #e0af68 — reinforcement only; BellAlertIcon + pulse shape carries state */
/* COLORBLIND-SAFE: attn border light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
/* Append to :root block: */
--hub-attn-border: #e0af68;
--hub-attn-border-glow: rgba(224,175,104,0.35);
--hub-attn-icon-color: #e0af68;
--hub-attn-badge-bg: rgba(224,175,104,0.18);
--hub-attn-badge-text: #e0af68;
--hub-attn-static-border: #e0af68;

/* Append to [data-ui-theme="light"] block: */
--hub-attn-border: #b45309;
--hub-attn-border-glow: rgba(180,83,9,0.25);
--hub-attn-icon-color: #b45309;
--hub-attn-badge-bg: rgba(180,83,9,0.14);
--hub-attn-badge-text: #b45309;
--hub-attn-static-border: #b45309;
```

### New Animation Rules (append after existing Hub rules)

```css
/* ATTN-01: attention card — pulsing border + glow */
/* COLORBLIND-SAFE: attn border dark hex #e0af68 — reinforcement only; BellAlertIcon + pulse carries state */
/* COLORBLIND-SAFE: attn border light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
.hub-card--attention {
  border-color: var(--hub-attn-border);
}

/* ATTN-03: attention clear — fade border back to normal over 400ms */
/* A11Y-03: pulse wrapped in @media (prefers-reduced-motion: no-preference); static border fallback */
@media (prefers-reduced-motion: no-preference) {
  .hub-card--attention {
    animation: hub-attn-pulse 2s ease-in-out infinite;
  }
  /* Extend .hub-card transition for smooth clear when attention class is removed */
  /* NOTE: This overrides the existing 100ms transition on .hub-card — must be in same media query */
  .hub-card {
    transition: border-color 400ms ease, box-shadow 400ms ease, background 100ms ease;
  }
  /* Attention card should keep amber border even on hover */
  .hub-card--attention:hover {
    border-color: var(--hub-attn-border);
  }
}

@keyframes hub-attn-pulse {
  0%   { border-color: var(--hub-attn-border); box-shadow: 0 0 0 0 var(--hub-attn-border-glow); }
  50%  { border-color: var(--hub-attn-border); box-shadow: 0 0 0 4px var(--hub-attn-border-glow); }
  100% { border-color: var(--hub-attn-border); box-shadow: 0 0 0 0 var(--hub-attn-border-glow); }
}

/* Reduced-motion: static attention border only — no animation, no glow */
@media (prefers-reduced-motion: reduce) {
  .hub-card--attention {
    border-color: var(--hub-attn-static-border);
    box-shadow: none;
    animation: none;
  }
}

/* ATTN-01: attention icon wrapper */
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
}

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
}
/* CRITICAL: size badge icon with explicit CSS — no Tailwind */
.hub__group-sidebar-item__attn-badge svg {
  width: 12px;
  height: 12px;
}
```

### Existing Transition Conflict

The existing `.hub-card` rule has `transition: border-color 100ms ease, background 100ms ease`.
The ATTN-03 clear animation needs `transition: border-color 400ms ease, box-shadow 400ms ease`.
Resolution: override the transition on `.hub-card` inside the `prefers-reduced-motion: no-preference`
block so it only applies to users who have motion enabled. The 100ms transition on `.hub-card` (outside
the media query) remains as the base for reduced-motion users. [VERIFIED: style.css line 4288 shows
existing 100ms transition]

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Attention predicate | Inline `status === 'waiting' \|\| ...` per component | `isAttentionStatus()` from `hubStatus.ts` | Prevents divergence; already has deriveHubStatus() as companion |
| CSS reorder animation | react-spring, framer-motion, @formkit/auto-animate | FLIP with `useLayoutEffect` + `getBoundingClientRect` | Zero new dependencies; full control; established in this codebase |
| Debounce | lodash.debounce | `useRef + setTimeout` hook | No dependency; controllable cleanup; already established pattern |
| Pulse animation | JS timer-driven opacity changes | CSS `@keyframes` + class toggle | CSS keyframes are GPU-accelerated; automatically respect prefers-reduced-motion when gated in media query |
| Badge icon sizing | `w-3 h-3` Tailwind class | `style={{ width: '12px', height: '12px' }}` or `.hub__group-sidebar-item__attn-badge svg { width: 12px; height: 12px }` | No Tailwind in this project — utility classes are no-ops |

**Key insight:** Zero new npm packages needed. The full animation contract (pulse + FLIP + debounce)
is achievable with native browser APIs and the existing heroicons/CSS infrastructure.

---

## Common Pitfalls

### Pitfall 1: Tailwind Icon Size Classes Are No-Ops
**What goes wrong:** `BellAlertIcon` renders at its natural SVG size (24px from heroicons default),
overflowing ROW 1 or pushing the status icon off-screen.
**Why it happens:** `w-4 h-4` / `w-3 h-3` appear in existing Hub code (e.g., drag handle, menu button)
and in GroupSidebar's NeedsInputBadge — but ALL of those are no-ops. The icons only appear small because
the heroicons SVG has `width="1.5em"` by default and the inline font-size context limits them.
**How to avoid:** Use `style={{ width: '16px', height: '16px' }}` on the `BellAlertIcon` element,
OR add a CSS rule targeting `.hub-card__attn-icon svg { width: 16px; height: 16px }`.
**Warning signs:** BellAlertIcon renders visually large (≥20px) in ROW 1. [VERIFIED: Phase 132 UAT bug]

### Pitfall 2: Border Hover Rule Overrides Attention Border
**What goes wrong:** When the user hovers over an attention card, the amber border changes to
`--hub-border-hover` (blue-grey), losing the amber attention signal.
**Why it happens:** `.hub-card:hover { border-color: var(--hub-border-hover) }` has higher specificity
precedence than `.hub-card--attention` when both apply (hover + attention class).
**How to avoid:** Add `.hub-card--attention:hover { border-color: var(--hub-attn-border) }` to maintain
the attention border color on hover. [VERIFIED: style.css lines 4291-4294]
**Warning signs:** Card border color changes when mouse enters an attention card.

### Pitfall 3: Pulse Keyframe Missing Reduced-Motion Guard
**What goes wrong:** Animation plays for users with `prefers-reduced-motion: reduce`.
**Why it happens:** `@keyframes hub-attn-pulse` declared or referenced outside the media query.
**How to avoid:** Wrap the `animation:` property on `.hub-card--attention` inside
`@media (prefers-reduced-motion: no-preference)`. The `@keyframes` block itself can be outside the
query (unused keyframes don't run), but the `animation:` declaration must be guarded. Pattern from
Phase 131 `hub-spin` at style.css line 4359.
**Warning signs:** `animation: hub-attn-pulse` outside a `prefers-reduced-motion` block.

### Pitfall 4: FLIP Measures Positions After React Re-renders Instead of Before
**What goes wrong:** FLIP "First" snapshot is taken after React has already moved the DOM elements —
delta is always zero and no animation plays.
**Why it happens:** `useEffect` runs AFTER the DOM is painted; if `capturePositions()` is called inside
a regular `useEffect`, React has already updated the DOM.
**How to avoid:** Use `useLayoutEffect` for the pre-render snapshot. `useLayoutEffect` fires
synchronously after React applies DOM mutations but before the browser paints — this is the correct
hook for FLIP's "First" phase. The "Play" phase still goes through `requestAnimationFrame` to let
the browser set up the transition.
**Warning signs:** Cards snap to new positions without any sliding animation.

### Pitfall 5: Debounce Applies to Card Content (Not Just Order)
**What goes wrong:** A card's attention border and BellAlertIcon only appear 1 second after the status
change, making the UI feel sluggish.
**Why it happens:** The `isAttention` prop passed to SessionCard is derived from the debounced value
instead of the live poll result.
**How to avoid:** The `isAttention` prop per card is derived from `isAttentionStatus(deriveHubStatus(s))`
using the LIVE session data (not the debounced key). Only the POSITION of the card within the grid is
debounced. The `debouncedSortKey` drives the sort order; individual `isAttention` props drive the
immediate visual treatment.
**Warning signs:** Border and BellAlertIcon appear with a 1-second delay after status change.

### Pitfall 6: Attention Badge Missing from Global "All" Count
**What goes wrong:** The "All" sidebar item shows no attention badge even when the grid has attention
sessions.
**Why it happens:** `computeGlobalCounts()` is not updated to include the `attention` count.
**How to avoid:** Update BOTH `computeCounts()` (per named group) AND `computeGlobalCounts()` (for the
"All" item) to include the `attention` field. [VERIFIED: GroupSidebar.tsx line 38-47 shows
computeGlobalCounts() as a separate function from computeCounts()]
**Warning signs:** "All" sidebar item never shows an attention badge.

### Pitfall 7: Attention Sort Breaks Named-Group Grouping
**What goes wrong:** Cards float to the top ACROSS groups, breaking the group boundary.
**Why it happens:** Sort is applied to `allSessions` before grouping, instead of per-group AFTER grouping.
**How to avoid:** Apply `sortSessionsForDisplay()` to each group's session array AFTER the group
assignment step (inside `groupByNamedGroups` or `groupByWorkDir`), not to the flat session list.
**Warning signs:** An attention card from Group A appears at the top of Group B's render.

---

## Code Examples

### `isAttentionStatus` addition to hubStatus.ts

```typescript
// Source: 133-UI-SPEC.md Attention Detection Contract [VERIFIED]
// Append to frontend/src/lib/hubStatus.ts after deriveHubStatus()

/* ATTN-01: canonical attention predicate — waiting, errored, or non-zero-exit sessions need attention */
export function isAttentionStatus(status: HubStatus): boolean {
  return status === 'waiting' || status === 'errored' || status === 'stopped-err'
}
```

### Attention Sort with Within-Group Application

```typescript
// Source: 133-UI-SPEC.md ATTN-02 Sort Rule [VERIFIED]
// Applied per-group inside groupByNamedGroups() or groupByWorkDir() return values

/* ATTN-02: float-to-top sort is within-group, not a global lane; debounce window 1000ms */
function sortSessionsForDisplay(sessions: SessionInfo[]): SessionInfo[] {
  return [...sessions].sort((a, b) => {
    const aAttn = isAttentionStatus(deriveHubStatus(a)) ? 0 : 1
    const bAttn = isAttentionStatus(deriveHubStatus(b)) ? 0 : 1
    return aAttn - bAttn
    // Stable sort: equal attention weight → original order preserved
  })
}
```

### Pulse CSS (full block — ready to append to style.css)

```css
/* Source: 133-UI-SPEC.md ATTN-01 Pulsing Border [VERIFIED] */

/* ATTN-01: attention card — pulsing border + glow (colorblind-safe: BellAlertIcon carries state) */
/* COLORBLIND-SAFE: attn border dark hex #e0af68 — reinforcement only; BellAlertIcon + pulse shape carries state */
/* COLORBLIND-SAFE: attn border light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
.hub-card--attention {
  border-color: var(--hub-attn-border);
}

/* A11Y-03: pulse wrapped in @media (prefers-reduced-motion: no-preference); static border fallback */
@media (prefers-reduced-motion: no-preference) {
  .hub-card--attention {
    animation: hub-attn-pulse 2s ease-in-out infinite;
  }
  /* ATTN-03: attention clear — fade border back to normal over 400ms */
  /* Override base .hub-card 100ms transition (motion-enabled users only) */
  .hub-card {
    transition: border-color 400ms ease, box-shadow 400ms ease, background 100ms ease;
  }
  /* Keep amber border even when hovering an attention card */
  .hub-card--attention:hover {
    border-color: var(--hub-attn-border);
  }
}

@keyframes hub-attn-pulse {
  0%   { border-color: var(--hub-attn-border); box-shadow: 0 0 0 0 var(--hub-attn-border-glow); }
  50%  { border-color: var(--hub-attn-border); box-shadow: 0 0 0 4px var(--hub-attn-border-glow); }
  100% { border-color: var(--hub-attn-border); box-shadow: 0 0 0 0 var(--hub-attn-border-glow); }
}

/* Reduced-motion: static attention border only — no animation, no glow */
@media (prefers-reduced-motion: reduce) {
  .hub-card--attention {
    border-color: var(--hub-attn-static-border);
    box-shadow: none;
    animation: none;
  }
}
```

---

## Existing Code — Critical Notes for Planner

These are exact observations from reading the Phase 131/132 code. The planner MUST account for them.

### `hubStatus.ts` — current state
- File: `frontend/src/lib/hubStatus.ts` [VERIFIED: codebase]
- Exports: `HubStatus` type + `deriveHubStatus(s: SessionInfo): HubStatus`
- Does NOT export `isAttentionStatus` (to be added in this phase)
- `HubStatus` already includes `'waiting' | 'errored' | 'stopped-err'` — no type changes needed

### `SessionCard.tsx` — current state
- File: `frontend/src/components/Hub/SessionCard.tsx` [VERIFIED: codebase]
- ROW 1 structure: `hub-card__status-indicator` | `InlineSessionName` | `hub-card__badge`
- Props do NOT include `isAttention` — must be added
- `className` on `article`: `hub-card`, optionally `hub-card--dim`, optionally `hub-card--dragging`
- `aria-label` on `article`: `${name}, ${displayLabel}, ${cli}, ${originText}` — must append `, needs attention` when `isAttention=true`
- Icons imported: ArrowPathIcon, CheckCircleIcon, PauseCircleIcon, ExclamationCircleIcon, StopCircleIcon, ComputerDesktopIcon, GlobeAltIcon, EyeIcon, Bars3Icon, EllipsisHorizontalIcon — `BellAlertIcon` NOT yet imported

### `SessionCardGrid.tsx` — current state
- File: `frontend/src/components/Hub/SessionCardGrid.tsx` [VERIFIED: codebase]
- Two render paths: named-group (Phase 132) and workDir-fallback (Phase 131)
- Props do NOT include `isAttention` per session — must thread through from HubPanel
- No animation hooks — FLIP must be added here (or in a helper hook)
- `SessionCard` props currently: `session`, `onRename`, `onOpenSession`, `previewLines`, `groupDefs`, `onAssignGroup`
- Must add `isAttention` prop to `SessionCard` and thread it from the grid

### `GroupSidebar.tsx` — current state
- File: `frontend/src/components/Hub/GroupSidebar.tsx` [VERIFIED: codebase]
- `GroupCounts` interface: `{ running, total, waiting }` — must add `attention: number`
- `computeCounts()` does NOT call `isAttentionStatus()` — must add
- `computeGlobalCounts()` also needs `attention` field
- `NeedsInputBadge` renders when `!collapsed && counts.waiting > 0` — must change to `collapsed` check + attention priority logic
- `GroupSidebarItem` currently renders the needs-input badge when `!collapsed` — the UI-SPEC says badges show when COLLAPSED; the current code appears to show the badge only when EXPANDED. **Investigate:** The current render in `GroupSidebarItem` is:
  ```tsx
  {!collapsed && counts.waiting > 0 && <NeedsInputBadge count={counts.waiting} />}
  ```
  This shows the badge when EXPANDED, not collapsed. The UI-SPEC and Phase 132 UI-SPEC agree the badge
  should show when COLLAPSED. This may be a Phase 132 implementation discrepancy that Phase 133 should
  correct. [VERIFIED: GroupSidebar.tsx line 137 — condition is `!collapsed` which means badge shows when
  EXPANDED. The UI-SPEC says badges show on COLLAPSED sidebar items.]

### `HubPanel.tsx` — current state
- File: `frontend/src/components/Hub/HubPanel.tsx` [VERIFIED: codebase]
- Passes `sessions={visibleSessions}` to `SessionCardGrid` — no `isAttention` threading yet
- Contains `usePreviewPoller` hook (established debounce pattern with `sessionIdKey`)
- The `useDebouncedValue` hook does NOT exist in this file — must be added
- No sort logic — attention sort must be added here or in SessionCardGrid

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No attention indicator | `isAttentionStatus()` + pulse border + BellAlertIcon | Phase 133 | Session status drives visual urgency without color-only signal |
| CSS `order` transition (not possible) | FLIP animation via `useLayoutEffect` + `getBoundingClientRect` | Phase 133 | Correct cross-browser solution for animating DOM reorders |
| Immediate reorder on every poll | 1s debounced reorder | Phase 133 | Prevents jarring jumps on every 3-second poll tick |
| Needs-input badge always visible | Collapsed-group attention badge replaces needs-input when `attnCount > 0` | Phase 133 | Superset signal avoids two competing badges |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Hover specificity conflict (`.hub-card:hover` overrides `.hub-card--attention` border) requires explicit `.hub-card--attention:hover` rule | Common Pitfalls / CSS Additions | If specificity is already handled by cascade order, the extra rule is redundant but harmless |
| A2 | The Phase 132 `NeedsInputBadge` render condition `!collapsed` is a bug (badge shows when EXPANDED, not COLLAPSED per UI-SPEC) | Existing Code Critical Notes | If intentional, Phase 133 attention badge logic needs different placement; research says the badge should show when COLLAPSED per 132-UI-SPEC.md and 133-UI-SPEC.md |
| A3 | `Array.prototype.sort` is stable in the React 19 / modern V8 runtime used by Wails | Pattern 3: Float-to-Top Sort | V8 has had stable sort since 7.0 (Node 12+). Wails uses Chromium WebView; all modern Chromium builds use V8 ≥ 7.0. Risk: very low |
| A4 | FLIP animation implemented with `useLayoutEffect` + `getBoundingClientRect` works correctly inside Wails WebView (Chromium) | Pattern 4: FLIP Animation | DOM measurement APIs are standard and work in Chromium. Risk: very low |

---

## Open Questions

1. **NeedsInputBadge render condition in Phase 132 code**
   - What we know: `GroupSidebar.tsx` line 137 renders `<NeedsInputBadge>` when `!collapsed` (expanded), but the 132-UI-SPEC and 133-UI-SPEC both specify the badge shows when COLLAPSED.
   - What's unclear: Is this an intentional Phase 132 inversion (showing the badge when expanded where cards are visible) or a bug?
   - Recommendation: Treat as a bug to be corrected in Phase 133. The Phase 133 attention badge MUST show when collapsed. Fix both badges (needs-input and attention) to show when collapsed, per 133-UI-SPEC.md §ATTN-06 Badge Display Rules.

2. **Sort location: HubPanel vs SessionCardGrid**
   - What we know: The sort must happen within each group. HubPanel computes groups implicitly via visibleSessions + groupDefs; SessionCardGrid does the actual grouping.
   - What's unclear: Whether it's cleaner to sort in HubPanel (before passing to the grid) or inside `groupByNamedGroups`/`groupByWorkDir` helper functions in SessionCardGrid.
   - Recommendation: Add `sortSessionsForDisplay()` inside `groupByNamedGroups` and `groupByWorkDir` — at the point where group session arrays are built. This avoids HubPanel needing to know about the internal group structure. The FLIP animation (which needs DOM measurement) still lives in SessionCardGrid.

---

## Environment Availability

Step 2.6: SKIPPED — Phase 133 is a pure frontend change (TypeScript + CSS). No new external tools,
services, runtimes, or CLI utilities are required beyond what Phase 131/132 established. No new npm
packages. No new Go code.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0 |
| Config file | `frontend/vite.config.ts` (test block) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test:coverage` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ATTN-01 | `isAttentionStatus` returns true for `waiting`, `errored`, `stopped-err`; false for others | unit | `pnpm test` | ❌ Wave 0 — `lib/hubStatus.test.ts` (new file) |
| ATTN-01 | `SessionCard` renders `BellAlertIcon` when `isAttention=true`; not rendered when `false` | unit | `pnpm test` | ✅ Extend `SessionCard.test.tsx` |
| ATTN-01 | `SessionCard` applies `.hub-card--attention` class when `isAttention=true` | unit | `pnpm test` | ✅ Extend `SessionCard.test.tsx` |
| ATTN-01 | `SessionCard` aria-label includes "needs attention" when `isAttention=true` | unit | `pnpm test` | ✅ Extend `SessionCard.test.tsx` |
| ATTN-02 | Within each group, attention cards appear before non-attention cards in rendered order | unit | `pnpm test` | ✅ Extend `SessionCardGrid.test.tsx` |
| ATTN-03 | When `isAttention` prop flips false, `.hub-card--attention` class is removed | unit | `pnpm test` | ✅ Extend `SessionCard.test.tsx` |
| ATTN-04 | `sortSessionsForDisplay` — attention sessions sort before non-attention; stable sort | unit | `pnpm test` | ✅ Extend `SessionCardGrid.test.tsx` |
| ATTN-06 | Collapsed `GroupSidebarItem` shows attention badge when `attnCount > 0` | unit | `pnpm test` | ✅ Extend `GroupSidebar.test.tsx` |
| ATTN-06 | Collapsed `GroupSidebarItem` shows needs-input badge (not attention) when `attnCount === 0 && waiting > 0` | unit | `pnpm test` | ✅ Extend `GroupSidebar.test.tsx` |
| ATTN-06 | Expanded `GroupSidebarItem` shows no attention badge (cards are visible) | unit | `pnpm test` | ✅ Extend `GroupSidebar.test.tsx` |
| ATTN-06 | `computeCounts` includes `attention` count; superset of `waiting` | unit | `pnpm test` | ✅ Extend `GroupSidebar.test.tsx` |
| Colorblind | `BellAlertIcon` has `aria-label="Needs attention"` (not `aria-hidden`) | unit | `pnpm test` | ✅ Extend `SessionCard.test.tsx` |
| Reduced-motion | Visual only — no CSS animation test in vitest | manual | — | N/A (verify in browser) |

### Wave 0 Gaps

- [ ] `frontend/src/lib/hubStatus.test.ts` (new file) — covers `isAttentionStatus` for all 6 HubStatus values
- [ ] Additions to `frontend/src/components/Hub/SessionCard.test.tsx` — ATTN-01, ATTN-03, colorblind aria-label
- [ ] Additions to `frontend/src/components/Hub/SessionCardGrid.test.tsx` — ATTN-02, ATTN-04 sort
- [ ] Additions to `frontend/src/components/Hub/GroupSidebar.test.tsx` — ATTN-06 badge logic, computeCounts attention field

*(All existing test files are in place from Phase 131/132 — only additions needed, no new infrastructure)*

---

## Security Domain

No new authentication, authorization, capability, or input handling is introduced. Phase 133 is
purely presentational: it toggles CSS classes and renders a Heroicons SVG based on existing session
status data. No new RPCs, no new user input, no new storage.

ASVS V5 input validation: not applicable — no new inputs.

---

## Project Constraints (from CLAUDE.md)

- **React/TypeScript frontend** (confirmed from codebase — Hub components are React, not Vue)
- **`pnpm` preferred** as package manager
- **TypeScript:** `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types
- **No Tailwind:** The project uses hand-rolled BEM CSS with CSS custom properties — `w-N h-N` utility classes are NO-OPS
- **Testing:** vitest (installed at ^4.1.0); extend existing test files in Hub/
- **No new global installs** — Phase 133 adds zero new npm packages
- **Go not involved** — pure frontend phase (no backend gap to fill)

---

## Sources

### Primary (HIGH confidence)
- `frontend/src/lib/hubStatus.ts` — Confirmed exports: `HubStatus` type + `deriveHubStatus()`; no `isAttentionStatus` yet [VERIFIED: codebase]
- `frontend/src/components/Hub/SessionCard.tsx` — Full ROW structure, props interface, className pattern, existing icon imports [VERIFIED: codebase]
- `frontend/src/components/Hub/SessionCardGrid.tsx` — `groupByWorkDir`, `groupByNamedGroups`, render paths [VERIFIED: codebase]
- `frontend/src/components/Hub/GroupSidebar.tsx` — `GroupCounts` interface, `computeCounts()`, `NeedsInputBadge`, `GroupSidebarItem` render [VERIFIED: codebase]
- `frontend/src/components/Hub/HubPanel.tsx` — `usePreviewPoller` hook pattern, `filterSessions`, prop interface [VERIFIED: codebase]
- `frontend/src/style.css` — All Hub CSS rules: `.hub-card` (line 4280), existing transition (line 4288), `:root` tokens (line 4096), light-theme tokens (line 4138), `hub-spin` keyframe pattern (line 4359-4365), `NeedsInputBadge` svg rule (line 4716) [VERIFIED: codebase]
- `frontend/package.json` — `@heroicons/react: ^2.2.0`; no animation libraries installed [VERIFIED: codebase]
- `frontend/node_modules/@heroicons/react/24/outline/BellAlertIcon.js` — BellAlertIcon confirmed present [VERIFIED: npm package]
- `.planning/phases/133-attention-pulse/133-UI-SPEC.md` — Locked design contract: tokens, keyframes, BEM classes, badge rules, sort/debounce specification [VERIFIED: planning docs]
- `.planning/phases/133-attention-pulse/133-CONTEXT.md` — Locked constraints: colorblind mandate, reduced-motion requirement, ATTN-03 modal-free clearing [VERIFIED: planning docs]

### Secondary (MEDIUM confidence)
- `.planning/phases/131-hub-foundation-static-session-cards/131-RESEARCH.md` — Hub architecture grounding, CSS pattern reference
- `.planning/phases/132-unified-grid-mini-preview-named-groups/132-RESEARCH.md` — GroupSidebar, GroupCounts patterns, Phase 132 bug notes
- `.planning/REQUIREMENTS.md` — ATTN-01..06 requirement definitions

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — confirmed from package.json and node_modules
- Backend gap analysis: HIGH — read all four layers (hubStatus.ts, SessionCard, GroupSidebar, HubPanel); confirmed no backend work needed
- Architecture: HIGH — derived from reading actual Phase 131/132 implementation files
- FLIP animation pattern: MEDIUM — pattern is well-known; specific React hook implementation is assumed (standard FLIP technique, not verified from an authoritative doc)
- CSS pitfalls: HIGH — derived directly from reading style.css and identifying the existing transition + hover conflict
- Existing code discrepancy (NeedsInputBadge condition): HIGH — read directly from GroupSidebar.tsx line 137

**Research date:** 2026-06-16
**Valid until:** 2026-07-16 (stable codebase — no external dependency changes expected)
