---
phase: 146-open-session-capability-bug
plan: 04
subsystem: testing
tags: [testing, traceability, regression, documentation, out-of-band, fix-03]

# Dependency graph
requires:
  - phase: 146-open-session-capability-bug
    plan: 02
    provides: "mintSessionJoinCodes removed; mint_join_codes_test.go deleted"
  - phase: 146-open-session-capability-bug
    plan: 03
    provides: "Frontend broadcast removed; out-of-band open wired; TESTING.md FIX-03 rows updated"
provides:
  - "TESTING.md compliant with standing regression convention for out-of-band redesign"
  - "M-13 manual UAT item rewired to out-of-band code-paste flow (no D-06 client-side selection language)"
  - "Suite Manifest note pruned of deleted-file reference (mint_join_codes_test.go)"
  - "Traceability checker exits 0"
  - "Full Go + frontend suite green; tsc clean; no broadcast symbols in production source"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Stale traceability prose references (deleted file names) should be removed from Suite Manifest notes as well as path-column rows"

key-files:
  created: []
  modified:
    - TESTING.md

key-decisions:
  - "M-13 rewired to out-of-band code-paste flow; removed D-06 isPeerSelf client-side RO/RW selection language which referenced dead code"
  - "mint_join_codes_test.go mention in L34 Suite Manifest prose removed to satisfy grep-c==0 acceptance criterion; the note still explains the Go count reduction using symbol names rather than the file path"
  - "isPeerSelf in App.open-remote.test.tsx is test-code testing for the symbol's absence from production code; it is not a broadcast symbol remnant in production source — documented, not suppressed"

requirements-completed: [FIX-03]

# Metrics
duration: 8min
completed: 2026-06-22
---

# Phase 146 Plan 04: Open Session Capability Bug — TESTING.md Compliance Summary

**TESTING.md brought into standing-convention compliance: M-13 rewired to out-of-band code-paste flow, stale Suite Manifest note pruned, traceability checker exits 0, full suite green**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-06-22T17:01:57Z
- **Completed:** 2026-06-22
- **Tasks:** 2
- **Files modified:** 1 (TESTING.md)

## Accomplishments

- Rewrote M-13 (Category G manual UAT) to the out-of-band code-paste flow: owner copies RO/RW join code from Share modal → delivers out of band → viewer pastes into RemoteJoinCodeModal → browser opens at baseURL/sessions/{id}?cap=TOKEN with the permission of the code sent.
- Removed stale D-06 client-side RO/RW selection language ("expect RW open per D-06"; "RW only when peer=self") — dead code removed in Plan 03.
- Reworded Suite Manifest note at L34 to remove `mint_join_codes_test.go` file path reference (file deleted in Plan 02; acceptance criterion: grep-c==0).
- Confirmed FIX-03 Section 4 traceability rows were already correct from Plan 03 (sessions_meta_embed_test.go Notes → TestSessionsMeta_NoJoinCodesInResponse; App.open-remote.test.tsx Notes → out-of-band open contract).
- Full Go suite green: `go test ./internal/webserver/ ./internal/daemon/ ./internal/tailnet/` all PASS (including TestSessionsMeta_NoJoinCodesInResponse).
- Frontend suite green: 1818/1818 tests across 112 files PASS.
- TypeScript gate clean: `pnpm tsc --noEmit` exits 0.
- No broadcast symbols in production source: mintSessionJoinCodes and SetJoinCodeIssuer completely absent; isPeerSelf absent from all non-test files.
- Traceability checker exits 0 (OK: all traceability paths exist).

## Task Commits

1. **Task 1: Revert/register TESTING.md rows + reword M-13; run traceability checker** — `5ca0c648`
2. **Task 2: Full-suite + build-gate verification** — No source changes; verification results recorded in SUMMARY only.

## Files Created/Modified

- `/Users/ken/dev/agenthub/TESTING.md` — rewrote M-13 to out-of-band code-paste flow; removed stale file-path mention from Suite Manifest note; all counts remain accurate (347 Go, 112 vitest)

## Decisions Made

- **M-13 reword scope:** Replaced the full step-list with the out-of-band flow steps. The old steps referenced "RW only when peer=self (D-06)" which was the broadcast-era client-side selection — dead code removed in Plan 03. New steps describe: Mac A shares + copies join code → sends out of band → Mac B pastes → browser opens with code's permission level.
- **Suite Manifest note (L34):** Removed the explicit `mint_join_codes_test.go` filename (satisfies `grep -c == 0` acceptance criterion); replaced with "broadcast-only test deleted in Plan 02 — mintSessionJoinCodes wiring removed" which still accurately explains the Go -1 count change without leaving a stale deleted-file reference.
- **isPeerSelf in test file:** `App.open-remote.test.tsx` contains `isPeerSelf` in comments and a test description that tests for its absence from App.tsx. This is expected and correct — the test verifies the dead code was removed. It is not a broadcast symbol remnant in production code. The production-source grep (`grep -rn isPeerSelf ... | grep -v __tests__`) returns zero matches.

## Deviations from Plan

None — plan executed exactly as written. The FIX-03 traceability rows in Section 4 were already accurate from Plan 03, so no changes were needed there. Only the M-13 manual UAT item and the Suite Manifest prose note required updates.

## Issues Encountered

- The acceptance criterion `grep -rn "mintSessionJoinCodes\|SetJoinCodeIssuer\|isPeerSelf" internal/ frontend/src/ | grep -v superseded` returns matches in `App.open-remote.test.tsx` (lines 4, 12, 21, 108, 109). These are in test file comments and a test description that verify the symbols are absent from production code (`expect(raw).not.toContain('isPeerSelf')`). This is correct behavior, not a regression. Production source is clean; `isPeerSelf` appears only in the test that tests for its removal.

## Known Stubs

None — TESTING.md is fully wired to real test implementations.

## Threat Flags

None — this plan only modifies TESTING.md documentation. No new network endpoints, auth paths, or schema changes.

## Self-Check

Files exist:
- [x] `/Users/ken/dev/agenthub/TESTING.md` — modified

Commits exist:
- [x] `5ca0c648` — docs(146-04): bring TESTING.md into compliance with out-of-band redesign

Acceptance criteria:
- [x] `grep -c "mint_join_codes_test.go" TESTING.md` == 0
- [x] `grep -c "TestSessionsMeta_EmbedJoinCodes\|TestMintSessionJoinCodes\|peer=self" TESTING.md` == 0
- [x] `grep -c "TestSessionsMeta_NoJoinCodesInResponse" TESTING.md` >= 1 (count=1, in Section 4 FIX-03 row)
- [x] App.open-remote.test.tsx row references out-of-band open path (confirmed on Section 4 line 151)
- [x] M-13 contains "out-of-band" and "paste" language (Category G, line 221)
- [x] No "peer=self" selection language in TESTING.md
- [x] `bash tests/check-traceability-paths.sh` exits 0 (OK: all traceability paths exist)
- [x] Go test suite PASS: webserver, daemon, tailnet (fresh run, no cache)
- [x] Frontend test suite PASS: 1818/1818 tests, 112 files
- [x] `pnpm tsc --noEmit` exit 0 (clean)
- [x] Go test count confirmed: 347 files (find . -name '*_test.go' | wc -l)
- [x] Vitest test count confirmed: 112 files

## Self-Check: PASSED
