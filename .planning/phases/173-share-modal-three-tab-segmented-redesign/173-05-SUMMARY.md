---
phase: 173-share-modal-three-tab-segmented-redesign
plan: 05
subsystem: ui
tags: [react, typescript, share-modal, funnel, tailscale, vitest]

requires:
  - phase: 173-share-modal-three-tab-segmented-redesign (plans 01-02-04)
    provides: "CSS classes (.hub-share-modal__tabpanel etc.), hoisted CodeDisplay/HoldToConfirmButton in shared.tsx, and the reusable ShareLinkCard component"
provides:
  - "TailnetTab.tsx — private tailnet-only link tier (Read-Only + Full Access ShareLinkCards)"
  - "InternetReadOnlyTab.tsx — public read-only Funnel link tier (ShareLinkCard + reusable code + Disable + warmup/timeout sub-states)"
  - "InternetFullAccessTab.tsx — the entire public-write/command-execution flow (Idle -> Gate-open -> Armed), walled off in one component"
  - "25 new vitest tests across 3 files, including the SM-04 negative wall-off assertion in both non-danger tabs"
affects: [173-06-shell-wiring, 173-07-testing-reconciliation]

tech-stack:
  added: []
  patterns:
    - "Idle -> Gate-open -> Armed local state machine for dangerous actions (danger explainer always visible; explicit second click required before the hold-to-confirm control is even reachable)"
    - "ShareLinkCard's url prop always receives the pre-computed join-code exchange URL (never a raw capability link), so Copy/Open/QR can never leak an ephemeral cap token"

key-files:
  created:
    - frontend/src/components/SessionShare/TailnetTab.tsx
    - frontend/src/components/SessionShare/InternetReadOnlyTab.tsx
    - frontend/src/components/SessionShare/InternetFullAccessTab.tsx
    - frontend/src/components/__tests__/TailnetTab.test.tsx
    - frontend/src/components/__tests__/InternetReadOnlyTab.test.tsx
    - frontend/src/components/__tests__/InternetFullAccessTab.test.tsx
  modified: []

key-decisions:
  - "InternetFullAccessTab implements the Idle/Gate-open/Armed 3-state flow described in the plan's <behavior>/must_haves (danger explainer -> 'Enable public write access…' -> expiry+hold+Cancel -> armed summary) rather than the simpler 2-state flow in the pre-existing SessionSharePanel.tsx — the plan's own truths/behavior/DESIGN code sketch all specify this explicitly, and it's an additive local-state change only (no TTL/capability/teardown semantics touched)"
  - "gateOpen resets to false automatically when writeGateUrl/writeGateCode transition from truthy to falsy (i.e. after Disable public write), so re-entering the tab always starts from Idle rather than a stale gate-open form"
  - "InternetReadOnlyTab's ShareLinkCard receives the pre-computed publicEntryUrl (the /join?code=... exchange URL) as its url prop, never the raw funnelUrl cap link — Copy/Open/QR all operate on the reusable link, closing the same leak class FNL-08 fixed"
  - "Preserved the degenerate no-publicReadCode fallback (bare /join page, plain markup, no ShareLinkCard) exactly as SessionSharePanel.tsx had it, since ShareLinkCard requires a non-empty code prop"

patterns-established:
  - "Three-tab decomposition: TailnetTab / InternetReadOnlyTab / InternetFullAccessTab as pure prop-driven renderers with all RPC handlers staying in the shell (enforced again in 173-06)"

requirements-completed: [SM-04, SM-06]

coverage:
  - id: D1
    description: "TailnetTab renders exactly two ShareLinkCards (Read-Only Link, Full Access Link) with scope descriptions attached beneath, and contains no public-write markup"
    requirement: "SM-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/TailnetTab.test.tsx#renders exactly two ShareLinkCards: Read-Only Link and Full Access Link"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/TailnetTab.test.tsx#SM-04: the public-write gate markup and hold-to-confirm button are ABSENT"
        status: pass
    human_judgment: false
  - id: D2
    description: "InternetReadOnlyTab renders the public URL (ShareLinkCard) + reusable public read code + Disable button, with warming/timed-out sub-states, and no public-write markup"
    requirement: "SM-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/InternetReadOnlyTab.test.tsx#funnelActive + publicReadCode renders a ShareLinkCard with the reusable /join entry URL (not the raw cap link)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/InternetReadOnlyTab.test.tsx#SM-04: the public-write gate markup and hold-to-confirm button are ABSENT"
        status: pass
    human_judgment: false
  - id: D3
    description: "InternetFullAccessTab hosts the entire public-write flow (danger explainer -> Enable public write access -> hold-to-confirm gate -> armed summary) with focus-on-arm and the fixed-enum expiry select preserved"
    requirement: "SM-04"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/InternetFullAccessTab.test.tsx#Gate open: clicking \"Enable public write access…\" reveals the expiry select + HoldToConfirmButton + Cancel"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/InternetFullAccessTab.test.tsx#focus management: focus moves to the Disable-public-write button the instant the gate arms"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-07-08
status: complete
---

# Phase 173 Plan 05: Three Tab-Body Renderers Summary

**Decomposed `SessionSharePanel.tsx` into `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab`, walling the public-write/command-execution flow off inside one component with a new Idle → Gate-open → Armed local state machine, plus 25 new vitest tests proving the SM-04 negative wall-off.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-07-08T07:04:00-05:00 (approx.)
- **Completed:** 2026-07-08T07:23:20-05:00
- **Tasks:** 3
- **Files modified:** 6 (all new)

## Accomplishments
- `TailnetTab.tsx` — renders two `ShareLinkCard` instances (Read-Only Link, Full Access Link), each with its scope description beneath (SM-06)
- `InternetReadOnlyTab.tsx` — public `/join` entry URL as a `ShareLinkCard` + reusable public join code + Disable button + warming/timed-out sub-states, keyed strictly off `funnelActive`/`warmingUp`/`warmupTimedOut` (never the shell's `funnelOn`, per RESEARCH Pitfall 3)
- `InternetFullAccessTab.tsx` — the entire public-write flow, now a 3-state Idle → Gate-open → Armed machine: danger explainer (always visible) → "Enable public write access…" → fixed-enum expiry select + `HoldToConfirmButton` (disabled until `funnelActive && !warmingUp`) + Cancel → armed summary (write URL / single-use code / live countdown / Disable public write) with focus moving to the Disable button the instant it arms
- 25 new vitest tests across 3 files; the SM-04/T-173-01 negative wall-off assertion (`hub-funnel-write-gate` class + hold-to-confirm button ABSENT) present in both `TailnetTab.test.tsx` and `InternetReadOnlyTab.test.tsx`
- Full project `tsc --noEmit`, `pnpm build`, and full `pnpm vitest run` (148 files / 2406 tests) all green after the change

## Task Commits

Each task was committed atomically:

1. **Task 1: Create TailnetTab + InternetReadOnlyTab** - `ff167e38` (feat)
2. **Task 2: Create InternetFullAccessTab** - `de8fd76e` (feat)
3. **Task 3: Per-tab tests incl. the SM-04 wall-off negative test** - `c18d563a` (test)

_Note: no separate refactor commit was needed — each task's implementation was correct on first pass._

## Files Created/Modified
- `frontend/src/components/SessionShare/TailnetTab.tsx` - Read-Only + Full Access ShareLinkCards, no public-write UI
- `frontend/src/components/SessionShare/InternetReadOnlyTab.tsx` - Public read-only Funnel URL card + reusable code + Disable + warmup/timeout states
- `frontend/src/components/SessionShare/InternetFullAccessTab.tsx` - Idle/Gate-open/Armed public-write consent gate, focus-management + countdown preserved
- `frontend/src/components/__tests__/TailnetTab.test.tsx` - 5 tests incl. SM-04 negative wall-off
- `frontend/src/components/__tests__/InternetReadOnlyTab.test.tsx` - 8 tests incl. SM-04 negative wall-off
- `frontend/src/components/__tests__/InternetFullAccessTab.test.tsx` - 13 tests covering the full Idle/Gate-open/Armed lifecycle

## Decisions Made
- Implemented the Idle → Gate-open → Armed 3-state flow for `InternetFullAccessTab` as explicitly specified in the plan's `<behavior>`/`must_haves` and the DESIGN doc's code sketch (`armed ? Summary : gateOpen ? Controls : IdleButton`), rather than the simpler always-visible-hold-control flow the pre-existing `SessionSharePanel.tsx` has today. This is additive local UI state only — it does not touch TTL/capability/teardown semantics (D-08 preserved) and reduces accidental public-write exposure by requiring a deliberate second click before the hold-to-confirm control is even reachable.
- `gateOpen` resets to `false` automatically when the armed result clears (`writeGateUrl`/`writeGateCode` go from truthy back to falsy after Disable), so the tab always re-enters at Idle rather than a stale gate-open form.
- `InternetReadOnlyTab`'s `ShareLinkCard` receives the pre-computed `publicEntryUrl` (the `/join?code=...` exchange URL) as its `url` prop — never the raw `funnelUrl` capability link — so Copy/Open/QR can never leak the ephemeral cap token (mirrors the FNL-08 fix).
- Preserved the degenerate no-`publicReadCode` fallback (bare `/join` page, plain markup, no `ShareLinkCard`) verbatim from `SessionSharePanel.tsx`, since `ShareLinkCard` requires a non-empty `code` prop and can't represent "no code yet" cleanly.

## Deviations from Plan

None affecting scope or files — one interpretive decision (documented above under Decisions Made) resolved an apparent tension in the plan between "extract verbatim" (`<action>`) and the explicit 3-state Idle/Gate-open/Armed flow described in `<behavior>`/`must_haves`/DESIGN. Followed the more detailed and repeated specification (`<behavior>`, `must_haves.truths`, and the DESIGN code sketch all independently describe the same 3-state machine).

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 06 (shell wiring) can now import `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab` and dispatch to them from `SessionShareModal.tsx`'s `.hub-share-modal__tabpanel` region; all RPC handlers/props remain in the shell per design.
- `SessionSharePanel.tsx` and its test are intentionally left untouched — plan 06 deletes both once the modal stops importing the panel, avoiding a red suite mid-wave.
- Plan 07's `TESTING.md` Suite Manifest/Traceability reconciliation should include the 3 new test files added here (`TailnetTab.test.tsx`, `InternetReadOnlyTab.test.tsx`, `InternetFullAccessTab.test.tsx`) alongside the other phase-173 test files, per its own frontmatter (`key_links` already lists them).

---
*Phase: 173-share-modal-three-tab-segmented-redesign*
*Completed: 2026-07-08*

## Self-Check: PASSED

All 6 created files found on disk; all 3 task commit hashes (`ff167e38`, `de8fd76e`, `c18d563a`) found in git log.
