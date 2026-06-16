---
phase: 129-write-concurrency-fix-dns-error-ux
plan: "02"
subsystem: files
tags: [sync, mutex, concurrency, race, toctou, atomic-write, sandbox]
dependency_graph:
  requires:
    - phase: 129-01
      provides: "TestWrite_TwoWritersIfMatchRace (RED) + TestRemoteFiles_TwoWriterRace_RelaySurface (RED)"
  provides:
    - "Package-level per-path keyed lock (sync.Map of *sync.Mutex) in WriteFileAtomic — single-winner guarantee (RACE-01, RACE-03)"
    - "Zero 'last-writer-wins' language in remote-write proxy (RACE-02)"
  affects:
    - internal/files/sandbox.go
    - internal/daemon/remote_files.go
tech_stack:
  added: ["sync (stdlib, no new deps)"]
  patterns:
    - "Package-level sync.Map keyed on absolute path for per-path mutex (not struct field — Sandbox is per-request/stateless)"
    - "LoadOrStore for atomic first-use mutex init — concurrent callers for same path always get same mutex"
    - "defer mu.Unlock() immediately after mu.Lock() so ALL early-return error paths release the lock"
key_files:
  created: []
  modified:
    - internal/files/sandbox.go (pathLocks sync.Map + perPathLock helper + lock in WriteFileAtomic + updated doc comment)
    - internal/daemon/remote_files.go (WR-02 comments corrected from last-writer-wins to single-winner)
key-decisions:
  - "Lock key is filepath.Join(s.rootPath, cleaned) — absolute path, not relative — so two Sandbox roots with the same relative filename do NOT contend with each other"
  - "Lock acquired AFTER denylistCheck (cheap reject before touching the lock map) and BEFORE os.OpenRoot (so entire temp-create→stat-check→rename is inside critical section)"
  - "pathLocks entries never pruned — one *sync.Mutex pointer per distinct absolute path is negligible memory; bounded by sandbox tree size (T-129-04 accepted)"
requirements-completed: [RACE-01, RACE-02, RACE-03]
duration: ~10min
completed: "2026-06-15"
---

# Phase 129 Plan 02: WriteFileAtomic Per-Path Lock + WR-02 Comment Corrections Summary

**Package-level per-path sync.Map mutex in WriteFileAtomic closes the stat→rename TOCTOU window — deterministic single-winner guarantee (RACE-01/RACE-03) — and zero "last-writer-wins" language remains in the remote-write proxy (RACE-02).**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-06-15T23:59:00Z
- **Completed:** 2026-06-15T23:59:48Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `WriteFileAtomic` now serializes concurrent same-path writers via a package-level `pathLocks sync.Map` keyed on the absolute path — the loser acquires the lock only after the winner's rename completes, sees a changed validator, and returns `ErrPreconditionFailed` cleanly
- `TestWrite_TwoWritersIfMatchRace` passes 100/100 with `-count=100 -race` (was 0/100 RED before the fix)
- `TestRemoteFiles_TwoWriterRace_RelaySurface` passes — single-winner guarantee holds end-to-end through the GUI relay loopback
- Both WR-02 comment blocks in `proxyRemoteFiles` corrected: "last-writer-wins" language replaced with accurate single-winner description; `grep -c "last-writer-wins"` = 0

## Task Commits

1. **Task 1: Per-path keyed lock in WriteFileAtomic** - `5825b92` (feat)
2. **Task 2: WR-02 comment corrections in remote_files.go** - `70e982f` (fix)

## Files Created/Modified

- `/Users/ken/dev/agenthub/internal/files/sandbox.go` — Added `sync` import, `pathLocks sync.Map`, `perPathLock()` helper, lock acquisition in `WriteFileAtomic` after `denylistCheck` / before `os.OpenRoot`, updated doc comment to single-winner language
- `/Users/ken/dev/agenthub/internal/daemon/remote_files.go` — Replaced "last-writer-wins" in both WR-02 comment blocks with accurate single-winner / peer-enforces-preconditions language

## Decisions Made

- Lock key uses `filepath.Join(s.rootPath, cleaned)` (absolute path) so two different Sandbox instances with the same relative filename do NOT contend — consistent with RESEARCH §Pattern 1 and the plan spec.
- `defer mu.Unlock()` placed immediately after `mu.Lock()` so every early-return (rand.Read fail, temp-create fail, sync/close fail, denylist) releases the lock without any code path audit needed.
- No change to the lock-pruning decision: entries are never evicted. One `*sync.Mutex` pointer per distinct absolute path is ~8 bytes; growth bounded by the finite sandbox tree (T-129-04 accepted).

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None. The three pre-existing RED test failures from Plan 01 (`TestCheckHealth_AcceptDNS`, `TestProxyRemoteFiles_AcceptDNSMessage` sub-case A, `TestSER03_NoAutoSavePatterns`) remain RED as expected — Plan 03 addresses the first two; the release-package test is an unrelated pre-existing environment artifact.

## Known Stubs

None. The per-path lock is live production behavior, not a stub.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes. Changes are purely internal locking and comment corrections.

T-129-03 (Tampering — stat→rename TOCTOU silent overwrite): **MITIGATED** — per-path mutex held across entire critical section.
T-129-04 (DoS — unbounded lock map growth): **ACCEPTED** — bounded by sandbox tree; one pointer per distinct path.
T-129-05 (Repudiation — doc/behavior mismatch): **MITIGATED** — grep gate = 0, comments now match code.

## Next Phase Readiness

- Plan 03 (DNS error UX) can proceed — RACE-01/02/03 gates are fully green
- `TestCheckHealth_AcceptDNS` and `TestProxyRemoteFiles_AcceptDNSMessage` sub-case A remain RED (Plan 03's responsibility)
- Full suite is green except those two pre-existing RED gates and the unrelated `TestSER03_NoAutoSavePatterns` env artifact

## Self-Check: PASSED

Files modified exist and contain expected content:
- `internal/files/sandbox.go`: contains `pathLocks sync.Map` — CONFIRMED
- `internal/files/sandbox.go`: contains `perPathLock` helper — CONFIRMED
- `internal/files/sandbox.go`: `"sync"` in imports — CONFIRMED
- `internal/daemon/remote_files.go`: `grep -c "last-writer-wins"` = 0 — CONFIRMED

Commits verified:
- 5825b92: feat(129-02): add per-path keyed lock to WriteFileAtomic for single-winner guarantee
- 70e982f: fix(129-02): correct WR-02 comments from last-writer-wins to single-winner (RACE-02)
