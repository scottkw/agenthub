---
phase: 173-share-modal-three-tab-segmented-redesign
plan: 06
subsystem: ui
tags: [react, typescript, vitest, hub-share-modal, funnel]

requires:
  - phase: 173-01
    provides: CSS tokens/classes (.hub-share-modal__tabpanel, .share-segbar/.share-seg*, widened 520px modal)
  - phase: 173-03
    provides: ShareSegmentedControl component + ShareTab type
  - phase: 173-05
    provides: TailnetTab / InternetReadOnlyTab / InternetFullAccessTab tab-body renderers
provides:
  - Wired shell — SessionShareModal dispatches to the three-tab segmented panel instead of stacking SessionSharePanel inline
  - Tab state machine (funnelOn-transition effects: default-to-Internet-RO-on-confirm, reset-to-Tailnet-on-disable)
  - Transient confirm view (FunnelRiskPanel repositioned to replace, not precede, the segmented panel)
  - On/Off/N-A colorblind-safe toggle state labels on all three control-strip toggles
  - Deletion of the now-obsolete SessionSharePanel.tsx + its test
affects: [173-07]

tech-stack:
  added: []
  patterns:
    - "Transition-only effects (prevFunnelOnRef) drive the tab default/reset — no new parallel confirm/tab state beyond the ShareTab itself"
    - "React.act() wrapping for async-RPC-then-passive-effect test sequences, not bare flushSync+setTimeout(0)"

key-files:
  created: []
  modified:
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/style.css
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - TESTING.md
  deleted:
    - frontend/src/components/SessionSharePanel.tsx
    - frontend/src/components/__tests__/SessionSharePanel.test.tsx

key-decisions:
  - "funnelError renders in BOTH the transient confirm view (RESEARCH A3) and inside .hub-share-modal__tabpanel — the full-access gate-confirm failure (handleGateConfirm) sets the same funnelError slot but fires once the confirm view is no longer reachable, so it needed a second surface to avoid silently dropping the error"
  - "Dropped the redundant `!funnelDisabled` guard around the confirm-view condition — riskPanelOpen can never be true while funnelDisabled (handleFunnelToggle early-returns), so the plan's literal `shareEnabled && riskPanelOpen` condition is sufficient"
  - "Tab defaults via lazy useState initializer (session.funnelActive ? 'internet-ro' : 'tailnet') on mount, and via a transition-only effect (prevFunnelOnRef) thereafter — matches RESEARCH Pitfall 4 (effects react to transitions, never fire on mount)"

requirements-completed: [SM-01, SM-05, SM-07, SM-08]

coverage:
  - id: D1
    description: "Fixed control strip (3 toggles) never reflows when a toggle is flipped"
    requirement: SM-01
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — SM-01: fixed control strip does not reflow on toggle"
        status: pass
    human_judgment: false
  - id: D2
    description: "Single scroll region (.hub-share-modal__tabpanel), not the whole dialog body"
    requirement: SM-02
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — SM-02: single scroll region"
        status: pass
    human_judgment: false
  - id: D3
    description: "ShareSegmentedControl + active tab body render in the tabpanel; real tablist/tab roles"
    requirement: SM-03
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — SM-03: segmented control is a real tablist"
        status: pass
    human_judgment: false
  - id: D4
    description: "Tab state machine — transient confirm view replaces the panel; Internet tabs aria-disabled until confirmed; default tab = Internet·Read-only on confirm; reset to Tailnet on disable"
    requirement: SM-05
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — FUI-01: risk panel on every enable (Tailscale mode)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — SM-05: tab availability, default, and reset"
        status: pass
    human_judgment: false
  - id: D5
    description: "On/Off/N-A colorblind-safe toggle state text labels on all three control-strip toggles"
    requirement: SM-07
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — SM-07: On/Off/N-A toggle state labels"
        status: pass
    human_judgment: false
  - id: D6
    description: "Modal width source string (520px clamp, already bumped in 173-01) plus test-suite update confirming it"
    requirement: SM-08
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionShareModal.test.tsx#SessionShareModal — SM-08: modal width source string"
        status: pass
    human_judgment: false
  - id: D7
    description: "SessionSharePanel.tsx + its test deleted; no dangling imports; project compiles"
    verification:
      - kind: unit
        ref: "test ! -f frontend/src/components/SessionSharePanel.tsx && test ! -f frontend/src/components/__tests__/SessionSharePanel.test.tsx"
        status: pass
      - kind: other
        ref: "pnpm exec tsc --noEmit (clean) + pnpm build (vite build succeeds)"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-07-08
status: complete
---

# Phase 173 Plan 06: Wire the shell — three-tab segmented dispatch Summary

**SessionShareModal now a fixed control strip + swappable three-tab panel (ShareSegmentedControl + TailnetTab/InternetReadOnlyTab/InternetFullAccessTab) with a transient FunnelRiskPanel confirm view and On/Off/N-A toggle labels, replacing the old single-column SessionSharePanel stack.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 3/3 completed
- **Files modified:** 5 (2 deleted)

## Accomplishments

- Wired the tab state machine: `tab` state (`ShareTab`) with a transition-only effect (`prevFunnelOnRef`) that defaults to Internet·Read-only when `funnelOn` transitions false→true (confirm) and resets to Tailnet when it transitions true→false (disable) — reusing existing `riskPanelOpen`/`funnelOn`, no new parallel state, per D-05/RESEARCH.
- Repositioned `FunnelRiskPanel` to render as the transient confirm view that REPLACES the segmented-control/tab-body region (case `shareEnabled && riskPanelOpen`) instead of being injected inline above the links — the exact "disheveled/reflow" defect the phase's DESIGN spec named.
- Redistributed all former `SessionSharePanel` props (17 total) across `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab`, dispatched from a single `.hub-share-modal__tabpanel` scroll region; kept every RPC handler in the shell.
- Added colorblind-safe On/Off/N-A text state labels (`toggleStateLabel` helper + `.settings-panel__toggle-state` span) to all three control-strip toggles, and migrated their inline `style={}` layout (cursor/opacity/pointerEvents/margin/fontWeight/fontSize/color) to CSS classes driven by `aria-disabled` attribute selectors.
- Deleted `SessionSharePanel.tsx` + its test; verified no remaining production import.
- Updated `SessionShareModal.test.tsx` (51 tests total, up from 42) with new SM-01/02/03/05/07/08 structural assertions and repaired the FUI-01/02/06 tests to seed `shareEnabled` (a real, spec-locked D-05 behavior change — the confirm view now requires sharing to be ON, which the pre-173 code didn't enforce). `SessionShareModal.disconnect.test.tsx` needed no changes.
- Corrected 4 stale `TESTING.md` traceability rows (FUI-04, FUI-05, FNL-08, FNL-09) that pointed at the deleted `SessionSharePanel.test.tsx`, redirecting them to `InternetReadOnlyTab.test.tsx`/`InternetFullAccessTab.test.tsx` where that coverage now lives (Plan 05). Full Suite Manifest count reconciliation (new-file rows, vitest total) remains 173-07's dedicated task per 173-05-SUMMARY.md.

## Task Commits

1. **Task 1: Add tab state machine + ShareSegmentedControl + tab dispatch + transient confirm view** - `525af848` (feat)
2. **Task 2: On/Off/N-A toggle labels + migrate inline style={} to classes + delete SessionSharePanel** - `bb6a9121` (feat)
3. **Task 3: Update SessionShareModal.test.tsx + disconnect test to the new structure (non-hue)** - `62685291` (test)

**Plan metadata:** (this commit, docs)

## Files Created/Modified

- `frontend/src/components/Hub/SessionShareModal.tsx` - Tab state machine, ShareSegmentedControl + tab dispatch, transient confirm view, On/Off/N-A labels, deleted SessionSharePanel import
- `frontend/src/style.css` - `.hub-share-modal__rule` (divider), `.hub-share-modal__empty-hint`, `.settings-panel__toggle-state`, `.hub-share-modal__toggle-wrap[aria-disabled]`, `.hub-funnel-toggle-section[aria-disabled]`, `.hub-funnel-toggle-section__note`, `.hub-share-modal__lan-creds-label` + margin fold-in
- `frontend/src/components/__tests__/SessionShareModal.test.tsx` - Updated FUI-01/02/06 gating, hardened async-effect timing with `React.act()`, added SM-01/02/03/05/07/08 describe blocks (9 new tests)
- `TESTING.md` - 4 traceability path corrections (SessionSharePanel.test.tsx → InternetReadOnlyTab.test.tsx / InternetFullAccessTab.test.tsx)
- `frontend/src/components/SessionSharePanel.tsx` - **deleted**
- `frontend/src/components/__tests__/SessionSharePanel.test.tsx` - **deleted**

## Decisions Made

- **funnelError dual-render:** the plan text only specified rendering `funnelError` alongside the transient confirm view (case b). But `handleGateConfirm` (the full-access write-gate confirm handler) sets the SAME `funnelError` state, and that failure can only happen once past the confirm view (tab === 'internet-fa', riskPanelOpen already false). Rendering it only in case (b) would silently swallow that error — a regression vs. the pre-173 code, which showed `funnelError` unconditionally. Resolved by rendering it in both places (mutually exclusive at any given time, so no duplicate display); documented as a Rule 1 (bug-prevention) deviation.
- **Dropped the `!funnelDisabled` guard** on the confirm-view render condition: `riskPanelOpen` can never become `true` while `funnelDisabled` (guarded in `handleFunnelToggle`), so the extra check was dead code. Simplified to match the plan's literal condition list.
- **Mount-time tab default via lazy `useState`** (`session.funnelActive ? 'internet-ro' : 'tailnet'`) rather than an effect firing on mount — keeps the transition effect (`prevFunnelOnRef`) strictly reacting to *changes*, per RESEARCH Pitfall 4, and lands a reopened already-Funnel-active modal on the safer Read-only tab immediately rather than requiring an extra render cycle.
- **React.act() over bare tick() for async-then-effect test sequences:** discovered via stress-testing (20 consecutive full-suite runs) that a single `setTimeout(0)` tick after an async RPC click does not reliably flush the SM-05 tab-transition passive effect — timing depends on the surrounding test run, not just the click itself. `enableFunnel` and the disable-flow helpers now wrap the click+resolve in `React.act(async () => {...})`, which flushes pending effects deterministically. Verified flake-free across 20+ repeated runs of both modal test files (147/2384 full-suite files/tests also pass).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug prevention] funnelError would have been silently dropped for full-access gate-confirm failures**
- **Found during:** Task 1 (render-region design)
- **Issue:** The plan's case (b) only specifies rendering `funnelError` alongside the transient confirm view. `handleGateConfirm`'s failure path sets the same state but fires when the confirm view is unreachable (tab already switched to `internet-fa`), which would make that error invisible — a regression versus the pre-refactor code where `funnelError` rendered unconditionally.
- **Fix:** Render `funnelError` in both the confirm-view branch (case b) and inside `.hub-share-modal__tabpanel` (case c) — mutually exclusive states, no duplicate display.
- **Files modified:** `frontend/src/components/Hub/SessionShareModal.tsx`
- **Commit:** `525af848`

**2. [Rule 1 - Bug] Async-effect test flakiness under repeated runs**
- **Found during:** Task 3 (stress-testing the updated test suite)
- **Issue:** `enableFunnel`'s single `await tick()` after the async CTA click did not reliably flush the new funnelOn-transition passive effect (tab default-to-Internet-RO); observed as an intermittent failure (~1/12–20 runs) in "shows the Starting up… state", "re-issues IssueCapabilities…", "Disable internet share", and the new SM-07 "Share the session" toggle test.
- **Fix:** Wrapped the async click+resolve sequences in `React.act(async () => {...})`, which deterministically flushes pending passive effects before the assertion runs.
- **Files modified:** `frontend/src/components/__tests__/SessionShareModal.test.tsx`
- **Verification:** 20 consecutive full runs of both modal test files, zero failures; full frontend suite (147 files / 2384 tests) green.
- **Commit:** `62685291`

**3. [Rule 1 - Bug prevention] Corrected 4 dangling TESTING.md traceability rows**
- **Found during:** Task 2 (post-deletion check)
- **Issue:** The Traceability map had 4 rows (FUI-04, FUI-05, FNL-08, FNL-09) pointing at `frontend/src/components/__tests__/SessionSharePanel.test.tsx`, which this plan deletes.
- **Fix:** Repointed the path column to `InternetReadOnlyTab.test.tsx` (FUI-04, FUI-05, FNL-08) / `InternetFullAccessTab.test.tsx` (FNL-09), where 173-05 relocated that coverage. Full new-file rows + Suite Manifest count are 173-07's dedicated task.
- **Files modified:** `TESTING.md`
- **Verification:** `bash tests/check-traceability-paths.sh` exits 0.
- **Commit:** `62685291`

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bug-prevention, 1 Rule 1 bug-fix in tests).
**Impact on plan:** All auto-fixes necessary for correctness (no silently-dropped errors, no flaky tests, no dangling docs). No scope creep — the Suite Manifest count reconciliation was explicitly left to 173-07.

## Issues Encountered

The FUI-01/02/06 tests originally rendered the modal without `webEnabled: true` before opening the Funnel risk panel, which worked under the pre-173 code (the risk panel rendered regardless of `shareEnabled`). Under the new D-05 state machine, the transient confirm view is correctly gated on `shareEnabled && riskPanelOpen` — a real, spec-locked behavior change, not a bug. Repaired by seeding `webEnabled: true` and awaiting the seeding tick in those tests (Task 3).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 173-07 (TESTING.md Suite Manifest reconciliation + full-suite/build gate) is unblocked: `tsc --noEmit` clean, `pnpm build` succeeds, full vitest suite (147 files / 2384 tests) green, `check-traceability-paths.sh` exits 0. Remaining 173-07 work: add Traceability rows for the net-new phase test files (`ShareSegmentedControl.test.tsx`, `ShareLinkCard.test.tsx`, `TailnetTab.test.tsx`, `HoldToConfirmButton.test.tsx`) and correct the vitest Suite Manifest count for the net file delta (+N new − 1 deleted `SessionSharePanel.test.tsx`).

---
*Phase: 173-share-modal-three-tab-segmented-redesign*
*Completed: 2026-07-08*

## Self-Check: PASSED
