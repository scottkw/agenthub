---
phase: 172-hub-card-layout-badge-refinement
plan: 01
subsystem: ui
tags: [react, css-tokens, hub-card, chips, badges, colorblind, sketch-driven]

# Dependency graph
requires:
  - phase: 171-fnl-09-full-access-badge
    provides: ".hub-fullaccess-badge (notched clip-path, LockOpenIcon, colorblind-guarantee) reused verbatim"
provides:
  - "Consolidated Hub session-card metadata: one .hub-card__chiprow (agent · origin quiet chips) + .hub-card__exposure (INTERNET/FULL ACCESS filled badges, own line) + .hub-card__meta (uptime · viewers · Connected)"
  - "Origin marker fully muted (no green-local/blue-remote color coding)"
affects: [hub-card-visual-polish, colorblind-safe-badge-review]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Quiet outlined chip family (.hub-card__chip, 7px rounded-rect, transparent bg) vs filled exposure badges — structural contrast, not just color, per Sketch 001 Variant B"
    - "Exposure cluster forced onto its own flex-basis:100% line so long hostnames / coexisting badges never wrap unpredictably or clip"

key-files:
  created: []
  modified:
    - frontend/src/style.css
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/SessionCard.test.tsx
    - frontend/src/components/__tests__/SessionCard.share.test.tsx
    - TESTING.md

key-decisions:
  - "Kept .hub-card__badge CSS in style.css unchanged (not superseded) because HubModal.tsx's session-picker chip still consumes it — confirmed via grep before removal, per the plan's own explicit instruction"
  - "Origin chip resolves to --hub-text-muted only, dropping the green-local/blue-remote color modifiers entirely (Sketch 001 D-01 pin — color reinforcement now lives only on the agent chip tint + filled exposure badges)"
  - "Exposure cluster (.hub-card__exposure) only renders when session.funnelActive || session.funnelWriteActive is true — avoids an empty wrapper span on cards with no exposure"

patterns-established:
  - "New Hub-card chip classes must resolve through --hub-* tokens with no per-theme override (agent-tint hexes are the sole intentional exception, matching the pre-existing .hub-card__badge palette)"

requirements-completed: [D-01, D-02, D-03, D-04, D-05, D-06, D-07]

coverage:
  - id: D1
    description: "Status stays the primary top-line signal above the chip row, unchipified, keeping spin + attention pulse (D-03)"
    requirement: "D-03"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#running card icon has hub-card__status-icon--spin class"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#isAttention=true: renders .hub-card__attn-icon element with aria-label \"Needs attention\""
        status: pass
    human_judgment: false
  - id: D2
    description: "Agent + origin render as one consolidated row of outlined quiet chips (7px rounded-rect, 1px border, transparent bg, muted text); origin chip is fully muted with no color coding (D-01)"
    requirement: "D-01"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > renders an agent chip (.hub-card__chip--agent) with the cli text"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > local session renders an origin chip (.hub-card__chip--origin) with \"Local\""
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > remote session renders an origin chip (.hub-card__chip--origin) with the peer hostname"
        status: pass
      - kind: other
        ref: "grep -Eq 'border-radius:\\s*7px' frontend/src/style.css"
        status: pass
    human_judgment: false
  - id: D3
    description: "INTERNET and FULL ACCESS are the only filled chips, on their own dedicated right-aligned exposure line; both coexist when funnelActive AND funnelWriteActive are true (D-04/D-05); FULL ACCESS keeps its notched clip-path, 700 weight, LockOpenIcon (Phase-171 colorblind guarantee)"
    requirement: "D-04, D-05"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > FUI-03/D-04: funnelActive=true renders .hub-internet-badge inside .hub-card__exposure"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > D-05: funnelActive AND funnelWriteActive both true renders BOTH exposure badges (coexist, no supersede)"
        status: pass
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionCard.share.test.tsx#FNL-09: FULL ACCESS badge — distinct indicator + coexistence with INTERNET > coexistence: both badges render (read INTERNET first, FULL ACCESS second) when both are active"
        status: pass
      - kind: other
        ref: "grep -q 'clip-path: polygon' frontend/src/style.css"
        status: pass
    human_judgment: false
  - id: D4
    description: "Uptime, viewer count, and remote Connected/Available render together on one muted meta line below the chip row (D-06)"
    requirement: "D-06"
    verification:
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > D-06: muted .hub-card__meta line renders the viewer count when viewerCount > 0"
        status: pass
      - kind: unit
        ref: "frontend/src/components/Hub/SessionCard.test.tsx#SessionCard chip row (Phase 172) > D-06: connected remote card renders the Connected item inside .hub-card__meta"
        status: pass
    human_judgment: false
  - id: D5
    description: "Both dark (:root) and light ([data-ui-theme=\"light\"]) themes resolve the new chip/meta rules correctly through --hub-* tokens (D-02) — visual/computed-style parity"
    requirement: "D-02"
    verification: []
    human_judgment: true
    rationale: "jsdom does not compute layout/CSS cascade against real stylesheet application — token resolution across themes needs a rendered-browser visual check, not a DOM assertion. Sketch 001 Variant B (the built-against contract) already renders both themes toggleable for comparison."

# Metrics
duration: 6min
completed: 2026-07-07
status: complete
---

# Phase 172 Plan 01: Hub-card chip-row consolidation Summary

**Consolidated the Hub session card's three inconsistent metadata treatments into one `agent · origin · exposure` chip row (Sketch 001 Variant B): 7px rounded-rect quiet outlined chips for agent/origin, INTERNET/FULL ACCESS as the sole filled badges on their own right-aligned line, and a muted uptime/viewers/Connected meta line below.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-08T02:04:00Z
- **Completed:** 2026-07-08T02:10:12Z
- **Tasks:** 3
- **Files modified:** 5 (4 declared in plan + 1 deviation)

## Accomplishments
- New `.hub-card__chiprow` / `.hub-card__chip` / `.hub-card__chip--agent` / `.hub-card__chip--origin` / `.hub-card__chip-icon` / `.hub-card__exposure` / `.hub-card__meta` CSS rules, all resolving through `--hub-*` tokens, matching the Sketch 001 Variant B pinned values (7px radius, 8px gap, 2px 8px padding, exposure `flex-basis:100%` own-line)
- SessionCard.tsx restructured: status row (unchanged, D-03) → exit chip → new chip row (agent chip, origin chip, exposure cluster wrapping the existing `.hub-internet-badge`/`.hub-fullaccess-badge` verbatim) → new muted meta line (uptime · viewers · Connected/Available)
- Origin marker is now fully muted — the green-local/blue-remote color coding is gone (Sketch pin resolving D-01 discretion)
- Extended `SessionCard.test.tsx` with a `describe('SessionCard chip row (Phase 172)')` block covering the decisive D-05 both-badges-coexist case, and reconciled `TESTING.md`'s Suite Manifest with a dated note

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the consolidated chip-row + exposure-line + meta-line CSS (both themes)** - `63f6fff1` (feat)
2. **Task 2: Restructure SessionCard.tsx into status → chip row → exposure line → meta line** - `6bace8bc` (feat)
3. **Task 3: Extend SessionCard structure tests + reconcile TESTING.md** - `ca3caf2b` (test)

**Plan metadata:** (recorded after this SUMMARY commit)

## Files Created/Modified
- `frontend/src/style.css` - New chip/exposure/meta CSS classes; removed dead `.hub-card__row2`/`.hub-card__row2-meta`/`.hub-card__origin*` rules; kept `.hub-card__badge` (still consumed by HubModal.tsx)
- `frontend/src/components/Hub/SessionCard.tsx` - Restructured render order into status → exit chip → chiprow → meta line; updated Layout JSDoc
- `frontend/src/components/Hub/SessionCard.test.tsx` - Updated 3 pre-existing DOM assertions to the new class names (Rule 3 fix); added the new Phase-172 chip-row describe block
- `frontend/src/components/__tests__/SessionCard.share.test.tsx` - Updated 2 pre-existing origin-indicator assertions to `.hub-card__chip--origin` (Rule 3 fix)
- `TESTING.md` - Dated Phase 172 Suite Manifest note (extended in place, no new files, vitest count unchanged 142/529)

## Decisions Made
- Kept `.hub-card__badge` CSS unchanged (not removed) since `HubModal.tsx`'s session-picker chip still consumes it — confirmed by grep before touching it, per the plan's own explicit instruction to check for other consumers first.
- Reused the existing `.hub-card__conn`/`.hub-card__conn--connected` classes for the Connected/Available meta-line item rather than inventing new classes, since the plan's artifact list marks `.hub-card__conn` as "moves into the meta line" (restyled/moved, not new).
- Built the meta line's dot separators via a `metaItems` array + `React.Fragment` mapping so a separator only ever appears between two items that both actually rendered (no dangling leading/trailing dot for cards with only one meta item).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated pre-existing DOM assertions broken by the class-name restructuring**
- **Found during:** Task 2 (SessionCard.tsx restructuring)
- **Issue:** Task 2's DOM restructuring supersedes `.hub-card__origin` (now `.hub-card__chip--origin`) and repoints the agent chip from `.hub-card__badge` to `.hub-card__chip--agent`. This broke 3 pre-existing assertions in `SessionCard.test.tsx` (origin ×2, agent badge ×1) and 2 in `frontend/src/components/__tests__/SessionCard.share.test.tsx`'s `CARD-02: Local vs remote origin indicator` block — a file not listed in the plan's `files_modified`, but directly broken by the plan's own Task 2 instructions.
- **Fix:** Updated the 5 assertions to query the new class names (`.hub-card__chip--origin`, `.hub-card__chip--agent`), preserving their original intent and text-content checks.
- **Files modified:** `frontend/src/components/Hub/SessionCard.test.tsx`, `frontend/src/components/__tests__/SessionCard.share.test.tsx`
- **Verification:** `pnpm test -- src/components/Hub/SessionCard.test.tsx src/components/__tests__/SessionCard.share.test.tsx src/components/__tests__/style.hub.test.ts` → 3 files, 172 tests passed. Full suite re-run: 142 files / 2360 tests passed.
- **Committed in:** `6bace8bc` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 - blocking, touching a file outside the plan's declared `files_modified` list)
**Impact on plan:** Necessary consequence of the plan's own DOM restructuring instructions; no scope creep beyond fixing tests the plan's changes broke. `.hub-card__badge` and `.hub-card__origin`'s CSS-content assertions in `style.hub.test.ts` needed no changes (they test the still-present `.hub-card__badge` rule, untouched).

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The Hub card now renders the consolidated `agent · origin · exposure` chip row per Sketch 001 Variant B; `pnpm build` (tsc + vite) and the full vitest suite (142 files / 2360 tests) both pass.
- Dark/light theme token resolution (D-02) is verified at source (all new rules resolve through `--hub-*` tokens, no per-theme override needed) but not re-verified by an actual browser render in this plan — flagged as `human_judgment: true` (D5 in coverage) for the verifier's visual/UAT pass, consistent with the plan's own verification section noting the Wails-coupled app shell needs an isolated component harness or the sketch file itself for a live dark/light comparison.
- No blockers for closing out Phase 172.

---
*Phase: 172-hub-card-layout-badge-refinement*
*Completed: 2026-07-07*

## Self-Check: PASSED

- FOUND: frontend/src/style.css
- FOUND: frontend/src/components/Hub/SessionCard.tsx
- FOUND: frontend/src/components/Hub/SessionCard.test.tsx
- FOUND: frontend/src/components/__tests__/SessionCard.share.test.tsx
- FOUND: TESTING.md
- FOUND: commit 63f6fff1 (Task 1)
- FOUND: commit 6bace8bc (Task 2)
- FOUND: commit ca3caf2b (Task 3)
