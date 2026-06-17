---
phase: 133-attention-pulse
verified: 2026-06-16T21:40:00Z
status: human_needed
score: 7/7 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Pulse animation visible on a waiting/errored card; static amber border shown under prefers-reduced-motion: reduce"
    expected: "Card shows animated amber-gold pulsing border + glow (2s loop) under normal motion settings; under reduced-motion the border is static amber with no glow or animation"
    why_human: "CSS animation and media-query behavior cannot be verified by grep or test suite; requires live rendering"
  - test: "Debounced float-to-top + FLIP animation timing: card holds position for ~1s then smoothly slides to top"
    expected: "When a session status changes to waiting/errored, the BellAlertIcon and border appear immediately; the card stays in its original grid slot for approximately 1 second, then animates to the top of its group using the FLIP transform"
    why_human: "WR-03 fix was flagged by the code reviewer as requiring a live UAT pass; the debounce-gate + FLIP timing is a correctness change that automated tests can only partially verify (memo boundary confirmed in tests, visual timing must be observed)"
  - test: "Collapsed group sidebar shows attention badge (BellAlertIcon + count) when group contains waiting/errored session"
    expected: "Collapsing a group that contains an attention session shows the amber badge with BellAlertIcon and the count number; expanding the group hides the badge"
    why_human: "DOM assertions pass in tests; visual appearance, badge rendering scale, and BellAlertIcon color/size require live inspection in the Wails app"
  - test: "ATTN-05: Resolving a waiting session inside its modal clears the card's pulse and attention icon without a page reload"
    expected: "After the Phase 134 modal resolves the waiting session (status transitions to running), the card in the grid loses its amber border, BellAlertIcon, and .hub-card--attention class on the next poll tick, with no full reload"
    why_human: "Phase 134 modal does not exist yet; the clearing mechanism is status-driven (proven by ATTN-03/05 test in HubPanel.test.tsx), but the end-to-end user flow requires Phase 134 to be present to test fully. The underlying status-driven clear IS verified by automated test."
---

# Phase 133: Attention + Pulse Verification Report

**Phase Goal:** Sessions needing attention float to the top and pulse visibly, with debounced non-jarring reordering, so users can identify blocked or errored sessions at a glance without relying on color
**Verified:** 2026-06-16T21:40:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `isAttentionStatus` returns true for `waiting`, `errored`, `stopped-err`; false for `running`, `idle`, `stopped-ok` | VERIFIED | `hubStatus.ts` line 31-33; 6/6 unit tests pass (`hubStatus.test.ts`) |
| 2 | A single canonical attention predicate exists, imported by all Hub components | VERIFIED | `hubStatus.ts` exports `isAttentionStatus`; imported in `SessionCard.tsx`, `GroupSidebar.tsx`, `SessionCardGrid.tsx`, `HubPanel.tsx` |
| 3 | Attention card shows amber pulsing border + BellAlertIcon; gated by `prefers-reduced-motion: no-preference`; static fallback under `reduce` | VERIFIED (code) / HUMAN for visual | CSS rules confirmed: `.hub-card--attention` at line 4921, animation at line 4927-4938, reduce-fallback at line 4950-4955, keyframe at 4943-4947; `opacity: 1` guard present (CR-02 fix) |
| 4 | Attention cards sort to top within each group (stable, never cross-group); reorder debounced ~1000ms | VERIFIED | `sortSessionsForDisplay` exported from `SessionCardGrid.tsx`; `sortedOrder` memoized on `debouncedSortKey` (WR-03 fix); applied via `sortedSessions` before grouping; attention-first ordering test passes (74 tests in SessionCardGrid + HubPanel suites) |
| 5 | Per-card `isAttention` is LIVE (immediate border/icon); only position is debounced | VERIFIED | `attentionIds` derived live from `allSessions` in `HubPanel.tsx` (line 251-253); threaded as `isAttention={attentionIds?.has(s.id)}` in both render paths of `SessionCardGrid.tsx` (lines 246, 288); `debouncedSortKey` gates sort order only |
| 6 | Status-leaving-attention clears the card's attention state without a page reload (ATTN-03/ATTN-05) | VERIFIED (automated) / HUMAN (full UX) | HubPanel test `ATTN-03/05` simulates `waiting → running` status change; asserts `.hub-card--attention` disappears without remount; 74 tests pass |
| 7 | Collapsed group sidebar shows attention badge (BellAlertIcon + count) when attention > 0; no badge when expanded | VERIFIED (code) / HUMAN for visual | `GroupSidebar.tsx` lines 144-156; badge condition `collapsed && counts.attention > 0`; `!collapsed && counts.waiting` inversion bug FIXED (zero grep hits for old pattern); `hub__group-sidebar-item__attn-badge--count` CSS rule added (CR-03 fix); 33 GroupSidebar tests pass |

**Score:** 7/7 truths verified (4 require human visual confirmation)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/lib/hubStatus.ts` | `isAttentionStatus(status: HubStatus): boolean` export | VERIFIED | Line 31-33; ATTN-01 comment; unchanged HubStatus type union |
| `frontend/src/lib/hubStatus.test.ts` | 6-case unit test covering full HubStatus truth table | VERIFIED | 6 tests, all pass |
| `frontend/src/style.css` | Attention tokens, pulse modifier, keyframe, reduced-motion guards, icon/badge sizing | VERIFIED | Lines 4136-4142 (dark), 4188-4194 (light); `.hub-card--attention` at 4921; keyframe at 4943; guards at 4927, 4950; `hub-card__attn-icon svg` at 4748; `hub__group-sidebar-item__attn-badge` at 4757; `--count` rule at 4779 |
| `frontend/src/components/Hub/SessionCard.tsx` | `isAttention` prop, BellAlertIcon in ROW 1, `.hub-card--attention` class, aria-label suffix | VERIFIED | Lines 100 (prop), 129 (destructure), 165-166 (aria), 225 (class), 296-300 (icon); no Tailwind on icon |
| `frontend/src/components/Hub/SessionCard.test.tsx` | Attention rendering on/off, aria-label suffix | VERIFIED | 48 tests pass; includes attention describe block |
| `frontend/src/components/Hub/GroupSidebar.tsx` | `GroupCounts.attention`, both count functions, badge render, corrected collapsed condition | VERIFIED | Lines 22 (interface), 29/37/39 (computeCounts), 45/50/52 (computeGlobalCounts), 144-157 (badge render); `!collapsed && counts.waiting` gone |
| `frontend/src/components/Hub/GroupSidebar.test.tsx` | Attention count, badge rules, collapsed/expanded, attention-over-needs-input priority | VERIFIED | 33 tests pass; ATTN-06 describe block present |
| `frontend/src/components/Hub/HubPanel.tsx` | `useDebouncedValue` hook, `attentionIds` (live), `debouncedSortKey`, prop threading, localStorage try/catch (WR-01), `attentionIds` memoized (IN-02) | VERIFIED | Lines 117-127 (hook), 242-254 (keys + memoized set), 320-321 (grid props), 202-210 (localStorage guard), 251-254 (useMemo); `setInterval` count = 1 |
| `frontend/src/components/Hub/HubPanel.test.tsx` | Live vs debounced, ATTN-03/05 clear, single-interval invariant | VERIFIED | Tests at lines 566-681; 74 tests pass (combined with SessionCardGrid suite) |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | `sortSessionsForDisplay`, per-group sort via `sortedSessions`, `isAttention` to both render paths, FLIP animation, CR-01 fix (single layoutEffect), WR-02 fix (capture always) | VERIFIED | Lines 76-81 (sort), 202-215 (memos), 246/288 (isAttention props), 84-125 (FLIP hook), 187-192 (single layoutEffect on debouncedSortKey) |
| `frontend/src/components/Hub/SessionCardGrid.test.tsx` | Attention-first ordering, stable sort, per-group boundary, WR-03 debounce gate | VERIFIED | 74 tests pass; ATTN-02 describe block at line 464 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `hubStatus.ts` | `HubStatus` type | `isAttentionStatus(status: HubStatus): boolean` | VERIFIED | Predicate over existing enum; no type change |
| `SessionCard.tsx` | `hubStatus.ts isAttentionStatus` | `isAttention` prop (computed by grid, not card) | VERIFIED | Card consumes prop only; grid/panel compute via `isAttentionStatus` |
| `.hub-card--attention` class | `style.css` rule | Applied when `isAttention` is true | VERIFIED | Line 225 of `SessionCard.tsx`; rule at line 4921 |
| `HubPanel.tsx` | `SessionCardGrid.tsx` | `attentionIds` + `debouncedSortKey` props | VERIFIED | Lines 320-321 of `HubPanel.tsx` |
| `SessionCardGrid.tsx` | `SessionCard.tsx` | `isAttention={attentionIds?.has(s.id)}` | VERIFIED | Lines 246, 288 — both named-group and workDir render paths |
| `sortSessionsForDisplay` | `hubStatus.ts isAttentionStatus` | Comparator calls `isAttentionStatus(deriveHubStatus(s))` | VERIFIED | Lines 78-79 of `SessionCardGrid.tsx` |
| `GroupSidebar.tsx` | `hubStatus.ts isAttentionStatus` | `computeCounts` + `computeGlobalCounts` | VERIFIED | Lines 37, 50 — both count functions |
| `.hub__group-sidebar-item__attn-badge` | `style.css` attn-badge rule | Class on collapsed badge span | VERIFIED | Line 148 of `GroupSidebar.tsx`; rule at line 4757 |
| `animation: hub-attn-pulse` | `@media (prefers-reduced-motion: no-preference)` | Animation declaration gated inside guard | VERIFIED | Line 4927-4929; keyframe declared outside guard at line 4943 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `HubPanel.tsx` | `attentionIds` | `allSessions.filter(isAttentionStatus)` — derived from live session poll | Yes — status from real daemon sessions | FLOWING |
| `HubPanel.tsx` | `debouncedSortKey` | `attentionSortKey` string derived from `allSessions`, debounced via `useDebouncedValue` | Yes — encodes real session id:bit pairs | FLOWING |
| `SessionCardGrid.tsx` | `sortedOrder` | `useMemo([debouncedSortKey])` over `sortSessionsForDisplay(sessions)` | Yes — memoized attention-first order | FLOWING |
| `SessionCard.tsx` | `isAttention` | `attentionIds?.has(s.id)` — live Set lookup | Yes — populated from real session status | FLOWING |
| `GroupSidebar.tsx` | `counts.attention` | `computeCounts` calls `isAttentionStatus(deriveHubStatus(s))` over real sessions | Yes — real session status | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| hubStatus predicate truth table | `pnpm test -- --run src/lib/hubStatus.test.ts` | 6/6 pass | PASS |
| SessionCard attention rendering | `pnpm test -- --run src/components/Hub/SessionCard.test.tsx` | 48/48 pass | PASS |
| GroupSidebar attention badge | `pnpm test -- --run src/components/Hub/GroupSidebar.test.tsx` | 33/33 pass | PASS |
| Grid sort + HubPanel debounce | `pnpm test -- --run src/components/Hub/SessionCardGrid.test.tsx src/components/Hub/HubPanel.test.tsx` | 74/74 pass | PASS |
| Full test suite | `pnpm test -- --run` | 1638/1638 pass | PASS |
| TypeScript types | `pnpm tsc --noEmit` | No errors | PASS |
| Production build | `pnpm build` | Bundle built (285ms) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| ATTN-01 | 133-01, 133-03 | Session with `waiting`, `errored`, non-zero exit is flagged as needing attention | SATISFIED | `isAttentionStatus` in `hubStatus.ts`; used by SessionCard, GroupSidebar, SessionCardGrid, HubPanel |
| ATTN-02 | 133-02, 133-05 | Attention card shows pulsing animated highlighted border plus attention icon | SATISFIED (code) / HUMAN (visual) | `.hub-card--attention` CSS + `hub-attn-pulse` keyframe; BellAlertIcon in SessionCard ROW 1 |
| ATTN-03 | 133-05 | Attention cards sort to top when cards overflow viewport | SATISFIED | `sortSessionsForDisplay` + `sortedOrder` memoized on `debouncedSortKey` in SessionCardGrid; ordering test passes |
| ATTN-04 | 133-05 | Reordering debounced and position changes animated (non-jarring) | SATISFIED (code) / HUMAN (visual timing) | `useDebouncedValue(1000ms)` in HubPanel; FLIP hook in SessionCardGrid; CR-01 + WR-03 fixes confirmed |
| ATTN-05 | 133-05 | Resolving a `waiting` session clears attention state | SATISFIED (status-driven clear proven) / HUMAN (end-to-end modal UX pending Phase 134) | HubPanel ATTN-03/05 test: `waiting → running` removes `.hub-card--attention`; no modal dependency in code |
| ATTN-06 | 133-04 | Collapsed group with attention card shows attention badge | SATISFIED (code) / HUMAN (visual) | GroupSidebar lines 144-156; `collapsed && counts.attention > 0` condition; badge CSS at line 4757 |

All six ATTN-xx requirements from PLAN frontmatter are covered. REQUIREMENTS.md marks all six as Phase 133 / Complete. No orphaned requirements found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `GroupSidebar.tsx` | 293 | `placeholder="Group name…"` | Info | HTML input placeholder attribute — not a stub indicator; live input for group creation |

No TBD, FIXME, XXX, or unresolved debt markers in any phase-modified file.

### Code Review Fixes Verified

All 9 findings from `133-REVIEW.md` were fixed (documented in `133-REVIEW-FIX.md`):

- **CR-01 (BLOCKER-grade fix):** Single `useLayoutEffect([debouncedSortKey])` with capture in cleanup; standalone always-running effect removed. VERIFIED at `SessionCardGrid.tsx` lines 183-192.
- **CR-02:** `opacity: 1` on `.hub-card--attention` prevents dim-modifier suppression. VERIFIED at `style.css` line 4923.
- **CR-03:** `.hub__group-sidebar-item__attn-badge--count` CSS rule added. VERIFIED at `style.css` lines 4779-4784.
- **WR-01:** `localStorage.setItem` wrapped in try/catch. VERIFIED at `HubPanel.tsx` lines 205-210.
- **WR-02:** `capturePositions` now always updates regardless of reduced-motion; only `playFLIP` guards. VERIFIED at `SessionCardGrid.tsx` lines 95-103.
- **WR-03:** Two-memo `sortedOrder` + `sortedSessions` approach decouples sort from live renders. VERIFIED at `SessionCardGrid.tsx` lines 202-215.
- **WR-04:** Dead Tailwind `w-3 h-3` / `w-4 h-4` classes removed from `GroupSidebar.tsx`. VERIFIED: zero grep hits for `w-3 h-3` in that file.
- **IN-01:** `position: relative` added to `.hub-card`. VERIFIED at `style.css` line 4304.
- **IN-02:** `attentionIds` memoized via `useMemo([attentionSortKey])`. VERIFIED at `HubPanel.tsx` lines 251-254.

### Colorblind Safety (Release-Blocking)

Verified at source level per user memory (user is colorblind; never verify by eye):

- `COLORBLIND-SAFE` comments present on dark hex `#e0af68` and light hex `#b45309` in both token blocks (`style.css` lines 4136, 4188)
- BellAlertIcon (shape carrier) is the primary signal — `aria-hidden="false"` equivalent via `aria-label="Needs attention"` on wrapper span (SessionCard line 298)
- Pulse border is reinforcement only — icon shape + motion carry the state
- Reduced-motion: static amber border (no motion); icon still present — colorblind-safe without motion
- All six `--hub-attn-*` tokens defined in both `:root` (dark) and `[data-ui-theme="light"]` blocks

### CARD-07 / Phase-132 Invariant

`grep -c "setInterval" frontend/src/components/Hub/HubPanel.tsx` = **1** (only the preview poller at line 107). `useDebouncedValue` uses `setTimeout` (line 123) — not a second periodic timer. Single-interval invariant preserved. Test at `HubPanel.test.tsx` line 650 confirms with a spy.

### Human Verification Required

#### 1. Pulse Animation Visual Fidelity

**Test:** Open the Wails app, create or observe a session in `waiting` or `errored` state
**Expected:** Card shows a 2-second amber-gold pulsing border with glow (dark: `#e0af68`; light: `#b45309`); the BellAlertIcon appears in ROW 1 to the left of the status indicator; under system `prefers-reduced-motion: reduce`, the border is static amber with no animation or glow
**Why human:** CSS `@keyframes` and media-query gating cannot be verified without live rendering

#### 2. Debounced Float-to-Top with FLIP Animation

**Test:** Trigger a session status change (e.g., session transitions to `waiting`); observe the card behavior over the next ~2 seconds
**Expected:** BellAlertIcon and amber border appear immediately; the card stays in its current grid position for approximately 1 second; after the debounce settles, the card smoothly slides (FLIP transform) to the top of its group; on second and subsequent changes the same 1s-hold-then-animate pattern holds
**Why human:** The WR-03 code-review fix changed the debounce-gate logic in a way that automated tests can only partially verify (memo boundary confirmed); the visual debounce timing and FLIP smoothness require live observation

#### 3. Collapsed Group Attention Badge

**Test:** Collapse a group in the sidebar that contains a `waiting` or `errored` session; then observe a group that contains only non-attention sessions
**Expected:** Attention group: amber badge with BellAlertIcon icon + count number appears on the collapsed sidebar item; expanding the group hides the badge; non-attention group: no badge when collapsed (if no sessions are waiting), or needs-input badge if `waiting > 0` and `attention === 0` (not currently reachable via normal status flow)
**Why human:** DOM assertions pass; visual size/color of badge and BellAlertIcon require live inspection

#### 4. ATTN-05 End-to-End Modal Clear (Phase 134 dependency)

**Test:** Open a `waiting` session card modal (Phase 134), interact to resolve the waiting state, observe the card in the grid after closing the modal
**Expected:** The card's amber border and BellAlertIcon clear on the next poll tick after status transitions to `running` — no page reload required
**Why human:** Phase 134 modal does not yet exist; the status-driven clear mechanism IS proven by automated test (`HubPanel.test.tsx` ATTN-03/05 case); this item is for end-to-end confirmation once Phase 134 ships

### Gaps Summary

No automated gaps found. All 7 must-have truths are VERIFIED at the code level. The 4 human verification items are inherently visual or involve a not-yet-built dependency (Phase 134 modal for ATTN-05 end-to-end). The WR-03 debounce-gate fix was specifically called out by the code reviewer as warranting a live UAT pass.

---

_Verified: 2026-06-16T21:40:00Z_
_Verifier: Claude (gsd-verifier)_
