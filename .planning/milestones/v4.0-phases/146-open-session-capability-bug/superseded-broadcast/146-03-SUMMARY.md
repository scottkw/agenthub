---
phase: 146-open-session-capability-bug
plan: "03"
subsystem: regression-suite
tags: [testing, traceability, TESTING.md, FIX-03]
dependency_graph:
  requires: [146-00, 146-01, 146-02]
  provides: [FIX-03-traceability, M-13-manual-checklist]
  affects: [TESTING.md]
tech_stack:
  added: []
  patterns: [regression-convention-compliance]
key_files:
  modified:
    - TESTING.md
decisions:
  - "Go count already at 348 from Wave 1; only vitest +1 and Total fix needed here"
  - "Total corrected to 465 (348+109+7+1) — Wave 1 had incremented Go but not Total"
  - "Added Category G to Manual Regression Checklist for live tailnet UAT (M-13)"
metrics:
  duration: "~10 minutes"
  completed: "2026-06-22"
  tasks_completed: 1
  tasks_total: 1
  files_changed: 1
---

# Phase 146 Plan 03: TESTING.md Regression Convention Compliance Summary

TESTING.md updated for Phase 146 per the standing convention: vitest count incremented to 109, Total corrected to 465 (internally consistent), FIX-03 traceability row added for `App.open-remote.test.tsx`, and M-13 manual checklist item added for the live two-Mac tailnet "Open in browser" UAT.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Update TESTING.md Suite Manifest, Traceability, and Manual checklist | 961c88f6 | TESTING.md |
| 1 (fix) | Correct Total count to 465 (internally consistent) | 7e268f24 | TESTING.md |

## What Was Built

**Section 2 (Suite Manifest):**
- Incremented vitest count 108 → 109 (new `App.open-remote.test.tsx`, Phase 146 Wave 0)
- Corrected Total from 462 → 465 (348 Go + 109 vitest + 7 Playwright + 1 build-script = 465); Wave 1 had incremented Go without updating Total
- Updated manifest footnote to note Phase 146 vitest addition

Wave 1 (commit f76d7b0c) already set Go = 348 (was 346) and added the 2 Go FIX-03 traceability rows.

**Section 4 (Traceability Map):** Added the third FIX-03 row:
```
| FIX-03 | frontend/src/components/__tests__/App.open-remote.test.tsx | vitest |
  handleOpenRemoteSession exchange-then-open (D-03/D-05/D-06 selection: RW only when
  peer=self, else RO); no-code informative UI; expired-banner on exchange failure |
```

**Section 5 (Manual Checklist):** Added Category G and M-13 — the live two-Mac tailnet "Open in browser" UAT for FIX-03 (#98), transcribed from `146-VALIDATION.md`. Covers three scenarios: (1) RO+RW shared session opened from remote Mac shows live terminal, (2) RO-only share opens RO mode, (3) owner re-attach gets RW per D-06. Documents why-not-automatable (two real Macs on one tailnet, :34115 wails-dev bridge has no real peer, web-share WS blocks automated input).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Total count inconsistency**
- **Found during:** Task 1 verification
- **Issue:** Wave 1 incremented Go 346→348 but did not update Total (still 462). This plan initially set Total=463 (+1 for vitest), leaving it off by 2 from the sum.
- **Fix:** Corrected Total to 465 (348+109+7+1=465) in a separate commit.
- **Files modified:** TESTING.md
- **Commit:** 7e268f24

## Verification

- `bash tests/check-traceability-paths.sh` exits 0
- `grep -q "FIX-03" TESTING.md` passes (3 FIX-03 rows present)
- `grep -q "App.open-remote.test.tsx" TESTING.md` passes
- All 3 FIX-03 test files confirmed on disk
- M-13 present in Section 5 under Category G
- Suite Manifest counts internally consistent: 348+109+7+1=465

## Known Stubs

None — documentation-only change, no runtime stubs.

## Threat Flags

None — documentation-only change, no new runtime trust boundaries.

## Self-Check: PASSED

- [x] TESTING.md committed (961c88f6, 7e268f24)
- [x] `frontend/src/components/__tests__/App.open-remote.test.tsx` exists on disk
- [x] `internal/webserver/sessions_meta_embed_test.go` exists on disk
- [x] `internal/daemon/mint_join_codes_test.go` exists on disk
- [x] Commits 961c88f6 and 7e268f24 present in git log
- [x] Suite Manifest counts: Go=348, vitest=109, Playwright=7, build-script=1, Total=465
