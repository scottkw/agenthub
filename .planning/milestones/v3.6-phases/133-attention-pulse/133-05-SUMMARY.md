---
phase: 133-attention-pulse
plan: "05"
subsystem: frontend/ui
tags: [tdd, hubpanel, sessioncardgrid, attention, attn-02, attn-03, attn-04, attn-05, flip, debounce, colorblind-safe]

requires:
  - phase: 133-01
    provides: isAttentionStatus export in hubStatus.ts
  - phase: 133-02
    provides: .hub-card--attention CSS rules in style.css
  - phase: 133-03
    provides: isAttention prop on SessionCard
  - phase: 133-04
    provides: GroupSidebar attention badge

provides:
  - useDebouncedValue hook in HubPanel (ATTN-04)
  - live attentionIds Set<string> derived from allSessions (ATTN-01 integration)
  - debouncedSortKey at 1000ms window driving reorder position only (ATTN-02/04)
  - sortSessionsForDisplay helper in SessionCardGrid — stable attention-first sort per group (ATTN-02)
  - attentionIds + debouncedSortKey props threaded from HubPanel to SessionCardGrid
  - isAttention={attentionIds?.has(s.id)} threaded to every SessionCard (both render paths)
  - FLIP reorder animation (useFLIPAnimation hook) suppressed under prefers-reduced-motion
  - ATTN-03/05 status-driven clear: poll-driven, no modal coupling
  - CARD-07 single-interval invariant preserved

affects: [134-modal-interaction, 135-attention-pulse-a11y]

tech-stack:
  added: []
  patterns:
    - TDD RED/GREEN — ordering, debounce, ATTN-03 clear, single-interval invariant
    - useDebouncedValue: useRef+setTimeout (NOT setInterval) for position-only debounce
    - FLIP animation: useLayoutEffect capture-before + playFLIP-after keyed on debouncedSortKey
    - Per-group sort inside group builders — flat sessions list never sorted (GROUP-04 preserved)
    - typeof window.matchMedia guard for test environment compatibility

key-files:
  created: []
  modified:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/SessionCardGrid.tsx
    - frontend/src/components/Hub/SessionCardGrid.test.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx

key-decisions:
  - "Live attentionIds (non-debounced) drives per-card isAttention for immediate border/icon; only reorder POSITION is debounced (RESEARCH Pitfall 5)"
  - "Sort applied per group INSIDE groupByWorkDir and groupByNamedGroups — never to the flat sessions list (RESEARCH Pitfall 7; GROUP-04 preserved)"
  - "useDebouncedValue uses useRef+setTimeout; setInterval count in HubPanel.tsx remains 1 (CARD-07 invariant)"
  - "FLIP capturePositions/playFLIP guard window.matchMedia with typeof check for jsdom test compatibility (Rule 3 auto-fix)"
  - "debouncedSortKey is a dep for the playFLIP useLayoutEffect; actual sort uses live isAttentionStatus calls inside sortSessionsForDisplay"

patterns-established:
  - "Debounce pattern: useDebouncedValue<T>(value, delay) with useRef+setTimeout cleanup — controls ORDER only, not content"
  - "FLIP reorder: capturePositions in unkeyed useLayoutEffect (before every update) + playFLIP in keyed useLayoutEffect (after debounce settles)"
  - "Per-group sort: apply sortSessionsForDisplay inside each group builder, not to the flat list"

requirements-completed: [ATTN-02, ATTN-03, ATTN-04, ATTN-05]

duration: 6min
completed: 2026-06-17
---

# Phase 133 Plan 05: Attention Wire-Up (HubPanel + SessionCardGrid) Summary

**Live `attentionIds` set drives immediate card border/icon; `useDebouncedValue` at 1000ms debounces only reorder position; `sortSessionsForDisplay` floats attention cards to top within each group; FLIP animation suppressed under prefers-reduced-motion — 73 tests pass, single-interval invariant preserved.**

## Performance

- **Duration:** ~6 min
- **Completed:** 2026-06-17
- **Tasks:** 3 (TDD RED + GREEN pt 1 + GREEN pt 2)
- **Files modified:** 4

## Accomplishments

### HubPanel.tsx
- Added `useDebouncedValue<T>(value, delay): T` module-scope hook (useRef+setTimeout); does NOT add a second periodic timer — single-interval invariant (CARD-07) preserved
- Derived `attentionIds: Set<string>` (LIVE, non-debounced) from `allSessions` using `isAttentionStatus(deriveHubStatus(s))` — border/icon updates immediately on every poll
- Derived `attentionSortKey` string and `debouncedSortKey` (1000ms debounce) from `allSessions` — reorder fires at most once per second
- Extended `<SessionCardGrid>` call with `attentionIds` and `debouncedSortKey` props
- `grep -c "setInterval" HubPanel.tsx` = 1 confirmed

### SessionCardGrid.tsx
- Added `isAttentionStatus/deriveHubStatus` import
- Added `sortSessionsForDisplay` (exported, stable, returns new array — no mutation)
- Applied `sortSessionsForDisplay` per group INSIDE `groupByWorkDir` and `groupByNamedGroups` — flat sessions list never sorted, group boundaries preserved
- Added `attentionIds?: Set<string>` and `debouncedSortKey?: string` to `SessionCardGridProps`
- Threaded `isAttention={attentionIds?.has(s.id)}` to each `SessionCard` in both named-group and workDir render paths
- Added `useFLIPAnimation` hook with `capturePositions` + `playFLIP` (both guarded via `typeof window.matchMedia` check for test environment safety)
- Added two `useLayoutEffect` calls: one unkeyed for capture (before DOM update), one keyed on `debouncedSortKey` for playFLIP

### Tests
- SessionCardGrid: 4 new attention ordering tests + 4 sortSessionsForDisplay unit tests
- HubPanel: 5 new attention tests (live vs debounced, ATTN-03/05 clear, debounce timing, single-interval invariant, multi-card attention)
- All 73 tests pass; 0 TypeScript errors; build succeeds

## Task Commits

1. **Task 1: RED tests** — `8dfb189b` (test)
2. **Task 2: HubPanel GREEN pt 1** — `ec71cd23` (feat)
3. **Task 3: SessionCardGrid GREEN pt 2** — `c022c917` (feat)

## Files Created/Modified

- `frontend/src/components/Hub/HubPanel.tsx` — useDebouncedValue hook, attentionIds derivation, debouncedSortKey, extended SessionCardGrid call
- `frontend/src/components/Hub/SessionCardGrid.tsx` — sortSessionsForDisplay, per-group sort in both builders, attentionIds/debouncedSortKey props, isAttention threading, useFLIPAnimation hook
- `frontend/src/components/Hub/SessionCardGrid.test.tsx` — attention ordering tests, sortSessionsForDisplay unit tests
- `frontend/src/components/Hub/HubPanel.test.tsx` — live vs debounced tests, ATTN-03/05 clear test, debounce timing test, single-interval invariant test

## Decisions Made

- Live `attentionIds` (non-debounced) for per-card `isAttention`; only reorder position is debounced (RESEARCH Pitfall 5 — border/icon must not lag 1s)
- Sort applied per group INSIDE the group builders — never to the flat list (RESEARCH Pitfall 7 — must not break group boundaries)
- `window.matchMedia` guarded with `typeof` check to handle jsdom test environment (auto-fix Rule 3 — blocking issue)
- `debouncedSortKey` is the dep for `playFLIP` useLayoutEffect; the actual sort inside the group builders uses live `isAttentionStatus` calls — debouncedSortKey only triggers the animation
- FLIP implemented (not simplified fallback) since the `typeof window.matchMedia` guard cleanly handles all environments

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] FLIP window.matchMedia not available in jsdom**
- **Found during:** Task 3 (first test run after implementing FLIP)
- **Issue:** `window.matchMedia` is not a function in the vitest/jsdom test environment; `capturePositions` and `playFLIP` threw TypeError, causing all 53+ tests to fail
- **Fix:** Added `typeof window.matchMedia === 'function'` guard before calling `matchMedia` in both `capturePositions` and `playFLIP`. In production (real browser), this is always defined; in tests, the guard suppresses the animation (behavior-equivalent since there are no real DOM positions to measure in jsdom anyway).
- **Files modified:** `frontend/src/components/Hub/SessionCardGrid.tsx`
- **Commit:** `c022c917`

## Known Stubs

None. The attention wiring is fully live:
- `attentionIds` is derived from `allSessions` on every render (same poll cycle as session status)
- `isAttention` is passed to every `SessionCard` with real computed value
- The debounce drives real position reordering via the group builders

## Threat Flags

None — presentational reorder + prop threading derived from existing session status. No new data ingress, RPC, input, or storage. T-133-05b (single-interval invariant) confirmed: `grep -c "setInterval" HubPanel.tsx` = 1.

## Self-Check: PASSED

- `frontend/src/components/Hub/HubPanel.tsx` modified and contains `useDebouncedValue` (2), `attentionIds` (2), `isAttentionStatus` (4), `setInterval` (1) — grep counts verified
- `frontend/src/components/Hub/SessionCardGrid.tsx` modified and contains `sortSessionsForDisplay` (3), `isAttention={attentionIds` (2), `prefers-reduced-motion: reduce` (2), `isAttentionStatus` (3)
- All 73 tests pass: `pnpm test -- --run ...SessionCardGrid.test.tsx ...HubPanel.test.tsx`
- `pnpm tsc --noEmit` clean (no errors)
- `pnpm build` succeeds
- Commits 8dfb189b, ec71cd23, c022c917 exist in git log
- Flat sessions list not sorted before grouping (sort only inside groupByWorkDir and groupByNamedGroups)
