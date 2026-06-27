---
phase: 157-terminal-screen-share-semantics-issue-109
plan: "05"
subsystem: testing
tags: [testing, traceability, regression, documentation, view-01, view-02, view-03, view-04, view-05]

requires:
  - phase: 157-01
    provides: "Hub host-authority arbiter + hub_test.go 6-test suite (VIEW-01/02)"
  - phase: 157-02
    provides: "Server join-push + web-drop tests in server_test.go + webserver/server_test.go (VIEW-02/03)"
  - phase: 157-03
    provides: "terminal.js viewer rewrite (vendored asset, structural gates only)"
  - phase: 157-04
    provides: "Desktop viewer parity + vitest new files + partial TESTING.md update (VIEW-04/05 rows)"

provides:
  - "TESTING.md Section 2: comprehensive Phase 157 delta note covering Go extensions (Plans 01/02) + new vitest files (Plan 04) + terminal.js structural gate note"
  - "TESTING.md Section 4: VIEW-01..03 traceability rows (VIEW-04/05 rows already present from 157-04)"
  - "TESTING.md Section 5: Category P (M-27/M-28) — Issue #109 garble check + desktop cross-surface parity check"
  - "bash tests/check-traceability-paths.sh exits 0 — ROADMAP success criterion 6 satisfied"

affects: []

tech-stack:
  added: []
  patterns:
    - "Vendored-asset structural gate note: terminal.js is outside vitest; document as node --check + grep + Category P manual UAT"

key-files:
  created: []
  modified:
    - TESTING.md

key-decisions:
  - "Section 2 note updated in-place (Phase 157 already had a partial note from 157-04); extended to cover Go test extensions from Plans 01 and 02 — no file-count change since those plans extended existing files"
  - "VIEW-01..03 rows inserted immediately before the existing VIEW-04/05 rows in Section 4 — logical ordering by requirement number"
  - "Category P placed as new Section 5 sub-section (M-27/M-28) before the Section 6 boundary — consistent with Category N/O convention"
  - "terminal.js (Plan 03) noted as vendored/structural in both Section 2 and the Category P introduction — correctly explains why it has no vitest coverage"

requirements-completed: [VIEW-01, VIEW-02, VIEW-03, VIEW-04, VIEW-05]

coverage:
  - id: D1
    description: "TESTING.md Section 2 comprehensive Phase 157 delta note (Go extensions + vitest new files + terminal.js structural note)"
    requirement: VIEW-01
    verification:
      - kind: other
        ref: "grep 'Phase 157' TESTING.md — note present and covers Go extensions from Plans 01/02"
        status: pass
    human_judgment: false
  - id: D2
    description: "TESTING.md Section 4 VIEW-01..05 traceability rows with repo-relative path-only columns"
    requirement: VIEW-02
    verification:
      - kind: automated
        ref: "bash tests/check-traceability-paths.sh — OK: all traceability paths exist"
        status: pass
    human_judgment: false
  - id: D3
    description: "TESTING.md Section 5 Category P (M-27/M-28) manual checklist for Issue #109 garble and cross-surface parity"
    requirement: VIEW-05
    verification:
      - kind: other
        ref: "grep 'Category P' TESTING.md — M-27/M-28 present with behavior description and why-not-automatable"
        status: pass
    human_judgment: false

duration: 3min
completed: 2026-06-27
status: complete
---

# Phase 157 Plan 05: TESTING.md Regression Convention Update Summary

**TESTING.md reconciled for full Phase 157 (VIEW-01..05): Section 2 note extended with Go test extensions, Section 4 VIEW-01..03 traceability rows added, Section 5 Category P manual UAT items (M-27/M-28) for the Issue #109 garble + cross-surface parity scenarios; path-check green**

## Performance

- **Duration:** 3 min
- **Started:** 2026-06-27T13:42:13Z
- **Completed:** 2026-06-27T13:45:12Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Extended the Phase 157 delta note in Section 2 to cover the Go test extensions from Plans 01 and 02 (hub_test.go MC-06 → 6 host-authority tests; server_test.go relay ordering test; webserver/server_test.go join-push + drop tests). No file-count change — Plans 01/02 extended existing files; Plans 01–05 net delta is vitest +2, Go unchanged at 366, total 505.
- Added a terminal.js vendored-asset note to the Section 2 delta: Plan 03's web viewer rewrite is outside the vitest suite; its correctness is gated by `node --check` + grep structural assertions, with behavioral proof deferred to Category P manual UAT.
- Added VIEW-01, VIEW-02, VIEW-03 traceability rows to Section 4 (VIEW-04/05 already present from 157-04 commit 7b0870ed). All five VIEW requirements now have at least one traceability row pointing at a repo-relative file path.
- Added Category P (M-27/M-28) to Section 5: M-27 covers the Issue #109 two-surface garble check (host PTY + smaller-windowed web guest, no overlapping chars, downscale cap s≤1.0); M-28 covers the desktop guest cross-surface parity check (isGuest gate, no sendResize back, CSS scale anchor). Both cite 157-VALIDATION.md as source.
- `bash tests/check-traceability-paths.sh` exits 0 on both macOS (grep -E fallback) and confirmed via Linux-compatible path extraction.

## Task Commits

1. **Task 1: Update TESTING.md manifest + traceability + manual checklist** - `d0621e64` (docs)

## Files Created/Modified

- `TESTING.md` — Section 2 note extended; VIEW-01..03 rows added to Section 4; Category P (M-27/M-28) added to Section 5

## Decisions Made

- Phase 157 partial TESTING.md update from 157-04 (VIEW-04/05 rows, vitest count 127→129) was preserved in-place; this plan reconciled and completed it by adding the missing VIEW-01..03 rows and the Section 2 Go extension note.
- VIEW-01..03 traceability maps to Go test files (relay/hub_test.go, relay/server_test.go, webserver/server_test.go) — not to terminal.js, which is a vendored asset without a unit harness.
- M-27 covers both the web guest garble scenario and the downscale-cap visual check (VIEW-05 criterion 4) as a single item since they require the same test setup (live host + smaller browser window).
- M-28 covers the desktop parity check separately (VIEW-04/05 criterion 5) since it requires the native Wails WebView, which Playwright cannot access.

## Deviations from Plan

None — plan executed exactly as written. The objective included a CONTEXT note that 157-04 had already added VIEW-04/05 rows and bumped the vitest count; this plan reconciled the remaining work (VIEW-01..03, Section 2 Go note, Category P) without duplicating anything.

## Issues Encountered

None.

## Known Stubs

None — all TESTING.md entries point at concrete, committed test files. The Category P manual items are not stubs; they are intentional human-UAT items for behaviors that cannot be automated (live multi-window xterm.js pixel render, native Wails WebView).

## Threat Flags

No new security surfaces — this plan edits TESTING.md only (documentation). T-157-08 (repudiation / coverage gap) is closed by the addition of VIEW-01..03 traceability rows and the Category P manual checklist.

## Phase 157 Final State

All five VIEW requirements satisfied and traceable:

| Requirement | Test Coverage | Manual UAT |
|-------------|---------------|------------|
| VIEW-01 | `internal/relay/hub_test.go` (broadcastResize) | M-27 (garble check) |
| VIEW-02 | `internal/relay/hub_test.go` + `internal/webserver/server_test.go` | M-27 |
| VIEW-03 | `internal/relay/server_test.go` + `internal/webserver/server_test.go` | M-27 |
| VIEW-04 | `frontend/src/lib/relayClient.test.ts` + `TerminalPanel.scale.test.tsx` | M-28 |
| VIEW-05 | `frontend/src/lib/terminalScale.test.ts` + `TerminalPanel.scale.test.tsx` | M-27/M-28 |

---
*Phase: 157-terminal-screen-share-semantics-issue-109*
*Completed: 2026-06-27*

## Self-Check: PASSED

- `TESTING.md` — FOUND (modified)
- Commit `d0621e64` — verified (docs: Task 1)
- `bash tests/check-traceability-paths.sh` — OK: all traceability paths exist
- `grep -c 'VIEW-0' TESTING.md` — 14 matches (VIEW-01..05 rows + note references)
- VIEW-01 row — FOUND in Section 4
- VIEW-02 rows (×2) — FOUND in Section 4
- VIEW-03 rows (×2) — FOUND in Section 4
- VIEW-04 rows (×2) — FOUND in Section 4 (from 157-04, preserved)
- VIEW-05 rows (×2) — FOUND in Section 4 (from 157-04, preserved)
- Category P (M-27/M-28) — FOUND in Section 5
