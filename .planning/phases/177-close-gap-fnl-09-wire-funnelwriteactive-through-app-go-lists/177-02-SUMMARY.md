---
phase: 177-close-gap-fnl-09-wire-funnelwriteactive-through-app-go-lists
plan: 02
subsystem: api
tags: [go, testing, tdd, funnel, sharing, regression-guard]

# Dependency graph
requires:
  - phase: 177-01
    provides: app.go SessionInfo.FunnelWriteActive field + ListSessions() propagation (the seam this plan guards)
provides:
  - "TestSessionInfo_MirrorsDaemonFunnelFields (app_test.go): reflection-based daemon<->app.go funnel-field parity guard, future-proofed against any new daemon Funnel* field"
  - "TestListSessions_PropagatesFunnelWriteActive (app_test.go): real ListSessions + json.Marshal round-trip guard on the actual serialized JSON"
  - "funnelBinding.contract.test.tsx: optional D-05b assertion that App.d.ts declares funnelWriteActive: boolean"
  - "TESTING.md Suite Manifest note + FNL-09 traceability row for the new guard"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reflection-based struct-parity guard: enumerate daemon.SessionInfo fields with a name prefix (Funnel), assert app.SessionInfo carries a field with the identical json tag — auto-covers future sibling fields without a new test"
    - "Serialization round-trip guard: drive the real App.ListSessions binding through a live in-process daemon + webserver + fake funnel client, then assert against json.Marshal output (not just the Go bool) — protects the exact bytes the raw Call() passthrough hands the frontend"

key-files:
  created: []
  modified:
    - app_test.go
    - frontend/src/components/__tests__/funnelBinding.contract.test.tsx
    - TESTING.md

key-decisions:
  - "Implemented BOTH D-05 variants (reflection parity + ListSessions/json.Marshal round-trip) rather than picking one — reflection auto-covers future funnel fields; the round-trip proves the actual runtime JSON bytes, which is what App.js passes through raw"
  - "Applied the optional D-05b assertion (App.d.ts funnelWriteActive: boolean) since it was cheap and non-load-bearing, mirroring the existing funnelActive assertion"
  - "Round-trip test bootstraps real capability state (api.BootstrapCapabilityState() + ws.SetSigningKey/SetJoinCodes) and drives app.SetSessionFunnelWrite end-to-end rather than constructing a SessionInfo by hand — this exercises the real handleSetSessionFunnelWrite mint path, not just a fabricated struct"

requirements-completed: [FNL-09]

coverage:
  - id: D5
    description: "A genuine Go regression test fails if app.go drops a daemon funnel-exposure field (FunnelActive or FunnelWriteActive) or its json tag diverges — exercising the real app.go struct + ListSessions serialization, never the App.d.ts stub text"
    requirement: "FNL-09"
    verification:
      - kind: unit
        ref: "go test -race -short . -run 'FunnelWrite|MirrorsDaemonFunnel' -v (PASS on fixed code); manually proven RED three ways against reintroduced pre-fix app.go: (1) removed the ListSessions copy line -> TestListSessions_PropagatesFunnelWriteActive FAILs with a clear message, (2) removed the struct field entirely -> compile failure (app_test.go references to .FunnelWriteActive undefined), (3) restored the field but mismatched its json tag -> TestSessionInfo_MirrorsDaemonFunnelFields FAILs with a clear message. All three reverted; full go test -race -short . green afterward."
        status: pass
    human_judgment: false
  - id: D5b
    description: "Optional: funnelBinding.contract.test.tsx asserts App.d.ts contains funnelWriteActive: boolean, mirroring the existing funnelActive assertion, without regressing prior assertions"
    requirement: "FNL-09"
    verification:
      - kind: unit
        ref: "cd frontend && pnpm exec vitest run src/components/__tests__/funnelBinding.contract.test.tsx -> 4/4 pass"
        status: pass
    human_judgment: false
  - id: docs
    description: "TESTING.md Suite Manifest note + FNL-09 traceability row added; counts unchanged (139 Go / 148 vitest / 298 total); check-traceability-paths.sh passes"
    requirement: "FNL-09"
    verification:
      - kind: unit
        ref: "bash tests/check-traceability-paths.sh -> 'OK: all traceability paths exist'; cross-checked manually with a Python script (macOS BSD grep -oP in the script is known-unreliable per Phase 173-07 precedent) confirming every Section 4 path-column entry resolves to a real repo-relative file"
        status: pass
    human_judgment: false

# Metrics
duration: 18min
completed: 2026-07-09
status: complete
---

# Phase 177 Plan 02: Go struct-parity / serialization regression guard for funnelWriteActive Summary

**Added a genuine Go regression test suite (reflection-based field parity + a real ListSessions/json.Marshal round trip) that fails if app.go ever again drops a daemon funnel-exposure field — proven RED against a temporarily reintroduced pre-fix app.go and GREEN on the fixed code — then reconciled TESTING.md's Suite Manifest and FNL-09 traceability map.**

## Performance

- **Duration:** 18 min
- **Completed:** 2026-07-09
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments

- `app_test.go` gained `TestSessionInfo_MirrorsDaemonFunnelFields`: a reflection-based test that enumerates every `daemon.SessionInfo` field whose Go name starts with `Funnel` (currently `FunnelActive`, `FunnelWriteActive`) and asserts `main.SessionInfo` carries a field with the identical `json` tag. Future daemon funnel fields are auto-covered without a new test.
- `app_test.go` gained `TestListSessions_PropagatesFunnelWriteActive`: drives a real in-process daemon + webserver (with a fake Funnel client, real capability bootstrap via `api.BootstrapCapabilityState()` + `ws.SetSigningKey`/`SetJoinCodes`) through `app.SetSessionFunnelWrite` and `app.ListSessions()`, then asserts BOTH the Go bool and the `json.Marshal`-produced bytes contain `"funnelWriteActive":false` (initial) and `"funnelWriteActive":true` (post-mint) — proving the exact JSON `App.js`'s raw `Call()` passthrough hands the frontend, not just an in-memory struct field.
- Proved genuineness three ways by temporarily reintroducing pre-fix app.go states, running the tests, observing the failure, then restoring: (1) commented out the `ListSessions` copy line -> the round-trip test FAILed with a clear message; (2) removed the struct field entirely -> the package failed to COMPILE (an even stronger signal); (3) restored the field with a deliberately mismatched json tag -> the reflection test FAILed with a clear message naming the missing/mismatched field. All three states were reverted and the full `go test -race -short .` suite confirmed green afterward.
- `funnelBinding.contract.test.tsx` gained the optional D-05b assertion: `App.d.ts SessionInfo interface contains funnelWriteActive: boolean`, mirroring the existing `funnelActive` assertion, with all 4 tests in the file (2 prior + 1 unrelated + this new one) passing.
- `TESTING.md` Section 2 (Suite Manifest) gained a dated Phase 177-02 reconciliation note: counts unchanged (139 Go / 148 vitest / 298 total) since both new Go test functions and the D-05b assertion were added in place to existing files, no new file. Section 4 (Traceability Map) gained a new `FNL-09` row pointing at `app_test.go` naming both new test functions.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the daemon<->app.go funnel-field parity + serialization regression guard (D-05); optional imported-stub assertion (D-05b)** - `22198744` (test)
2. **Task 2: Reconcile TESTING.md — Suite Manifest note + FNL-09 traceability row** - `fc39444c` (docs)

**Plan metadata:** (pending — final docs commit follows this SUMMARY)

## Files Created/Modified

- `app_test.go` - Added `reflect` and `encoding/json` imports; added `TestSessionInfo_MirrorsDaemonFunnelFields` and `TestListSessions_PropagatesFunnelWriteActive`. This is the load-bearing regression guard for the seam Plan 01 fixed.
- `frontend/src/components/__tests__/funnelBinding.contract.test.tsx` - Added the optional D-05b assertion (`funnelWriteActive: boolean` on the imported `App.d.ts` stub).
- `TESTING.md` - Section 2 dated reconciliation note (counts unchanged, 298 total) + Section 4 new FNL-09 traceability row for `app_test.go`.

## Decisions Made

- Implemented BOTH D-05 variants (reflection parity + ListSessions/json.Marshal round-trip) rather than choosing one, since each catches a different class of regression: reflection auto-covers a future sibling field that a developer forgets to add to the ListSessions copy loop; the round-trip proves the actual runtime JSON bytes are correct even if the struct/tag pairing looks right on paper.
- Applied the optional D-05b assertion since it was cheap and low-risk.
- The round-trip test bootstraps real daemon capability state (`api.BootstrapCapabilityState()`, `ws.SetSigningKey`, `ws.SetJoinCodes`) and drives the actual `handleSetSessionFunnelWrite` HTTP mint path via `app.SetSessionFunnelWrite`, rather than fabricating a `SessionInfo{FunnelWriteActive: true}` by hand — this exercises the full real seam (daemon mint -> daemon ListSessions -> app.go ListSessions -> json.Marshal), the strongest form of the guard the CONTEXT's D-05 decision asked for.

## Deviations from Plan

None — plan executed exactly as written. Both the reflection-based parity test and the ListSessions/json.Marshal round-trip test were implemented (plan allowed picking either; both were included since "Strongly consider including" language in the plan's `<action>` favored the round-trip as the strongest guard, and the reflection variant was equally cheap to add). The RED->GREEN proof was performed via three separate pre-fix reintroductions (missing copy line, missing struct field, mismatched json tag) rather than just one, for stronger evidence that both new tests are genuine rather than tautological.

## Issues Encountered

None. `bash tests/check-traceability-paths.sh` printed `OK` but per the standing [[reference_gsd_decision_coverage_bullet_format]]-adjacent lesson from Phase 173-07 (macOS BSD `grep -oP` in that script is known-unreliable and can false-OK with zero paths actually checked), a manual Python cross-check of every Section 4 path-column entry was run to independently confirm all paths (including the new `app_test.go` row) resolve to real repo-relative files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 177 (both plans) is now complete: Plan 01 wired `FunnelWriteActive` through the app.go Wails bridge (the runtime fix), and this plan (177-02) added the durable Go regression guard plus TESTING.md reconciliation the audit demanded. Live UAT of the FULL ACCESS badge/tab-icon/share-modal-resync rendering a real gate-minted write cap on a live Tailscale tailnet remains the one deferred human-judgment item (documented in 177-01-SUMMARY.md coverage D3), consistent with the M-46/M-47 live-UAT precedent for this milestone. Ready for `/gsd-verify-work 177`.

---
*Phase: 177-close-gap-fnl-09-wire-funnelwriteactive-through-app-go-lists*
*Completed: 2026-07-09*

## Self-Check: PASSED

- FOUND: app_test.go
- FOUND: frontend/src/components/__tests__/funnelBinding.contract.test.tsx
- FOUND: TESTING.md
- FOUND: 22198744 (Task 1 commit)
- FOUND: fc39444c (Task 2 commit)
