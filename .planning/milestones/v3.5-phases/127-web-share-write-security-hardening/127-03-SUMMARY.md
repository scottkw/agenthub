---
phase: 127-web-share-write-security-hardening
plan: "03"
subsystem: security-audit
tags: [security, capability, audit, data-integrity, testing]
dependency_graph:
  requires: ["127-01"]
  provides: ["127-SECURITY.md capability-escalation audit", "SEC-05 two-writer race test", "SEC-05 interrupted-write preservation test"]
  affects: ["internal/files/write_test.go"]
tech_stack:
  added: []
  patterns: ["STRIDE threat register", "ASVS L1 mapping", "If-Match optimistic concurrency"]
key_files:
  created:
    - .planning/phases/127-web-share-write-security-hardening/127-SECURITY.md
  modified:
    - internal/files/write_test.go
decisions:
  - "SEC-04: documented daemon loopback socket as ACCEPTED RISK (WEB-01 loopback trust), not a finding — only in-process GUI/TUI reach it"
  - "SEC-04: documented stat→rename TOCTOU residual as ACCEPTED (AR-127-02) — not eliminable in Go stdlib without renameat2"
  - "SEC-04: documented NFC/NFD as LOW residual (AR-127-03) — all protected names ASCII; case-fold sufficient"
  - "SEC-05: used stale-validator path (ErrPreconditionFailed) to drive the interrupted-write test, exercising the same temp+Remove cleanup branch"
  - "Shadowing built-in min with a local test helper: intentional, avoids import; vet and tests clean"
metrics:
  duration: "~20 minutes"
  completed: "2026-06-15"
  tasks_completed: 2
  files_created: 2
  files_modified: 1
---

# Phase 127 Plan 03: Capability-Escalation Audit + SEC-05 Data-Integrity Tests

**One-liner:** HMAC+HasPerm+SID+Origin per-surface audit committed as 127-SECURITY.md (SEC-04); two concurrent-writer If-Match race and interrupted-write-preserves-original tests added and passing under -race (SEC-05).

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Author and commit 127-SECURITY.md (SEC-04 audit artifact) | 461ab09 | `.planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` (created) |
| 2 | SEC-05 data-integrity tests — two-writer race + interrupted write | 2d8e4a8 | `internal/files/write_test.go` (2 tests added: `TestWrite_TwoWritersIfMatchRace`, `TestWrite_InterruptedWritePreservesOriginal`) |

## What Was Built

### Task 1 — 127-SECURITY.md (SEC-04)

The capability-escalation audit artifact for Phase 127. Structure follows the 83/74/60-SECURITY.md precedent.

**Sections:**
- **Trust Boundaries:** 5 boundaries documented (tailnet browser, desktop Wails, daemon socket, remote proxy, two concurrent writers).
- **Per-surface capability-escalation matrix:** 7 rows covering every surface that could theoretically reach a write endpoint.  Key findings:
  - All webserver write routes (PUT/POST/DELETE/rename/mkdir + HEAD canWrite probe) enforce `requireFilesWrite` → `requireCapability` (HMAC+SID+grant+session-enabled) → `HasPerm` whole-token → `originAllowedForWrite` CSRF check.
  - Remote proxy strips caller `?cap` and re-mints via the deposited token; remote peer enforces independently.
  - Daemon loopback socket is the only surface reachable without `files.write` — by design (WEB-01), documented as ACCEPTED RISK.
  - All sessions are SID-scoped; cross-session use returns 403.
  - `HasPerm` uses `strings.Split` whole-token comparison, not `strings.Contains` — no substring bypass.
  - Viewer token is bare `"read"`; `files.write` is per-session opt-in only.
- **CSRF Origin inversion** explicitly documented: `originAllowedForWrite` is the INVERSE of `requireAllowedOrigin`. Absent Origin passes vacuously (desktop Wails); present must byte-match `BaseURL()`.
- **Denylist threat model:** Phase 127-01 fixes (macOS config-dir + case-fold) documented with test citations.
- **STRIDE threat register:** T-127-01..T-127-10, all closed (10 threats, 0 open).
- **Accepted Risks Log:** AR-127-01 (daemon socket), AR-127-02 (stat→rename TOCTOU), AR-127-03 (NFC/NFD LOW residual).
- **ASVS L1 mapping:** V1/V2/V4/V5/V6/V12/V13.
- 170 lines, 121 substantive lines (requirement: ≥ 40).

### Task 2 — SEC-05 Data-Integrity Tests

Two new tests in `internal/files/write_test.go`:

**`TestWrite_TwoWritersIfMatchRace`:** Launches two goroutines each calling `WriteFileAtomic` with the same `If-Match` validator captured before either write commits. Asserts:
- Exactly one returns `nil`.
- Exactly one returns `ErrPreconditionFailed` (via `errors.Is`).
- Final file content is entirely writer-A's payload or entirely writer-B's — never a byte-interleaved mix.
- No `.agenthub-tmp-*` sibling remains after both goroutines complete (via `filepath.Glob`).

**`TestWrite_InterruptedWritePreservesOriginal`:** Captures a validator, modifies the file externally (simulating a concurrent writer landing first), then calls `WriteFileAtomic` with the now-stale validator. Asserts:
- Returns `ErrPreconditionFailed`.
- On-disk content equals the interleaved update (the file is unchanged from before our call).
- No `.agenthub-tmp-*` sibling remains (CR-01 cleanup verified).

Both pass under `go test -race ./internal/files/ -count=1`.

## Verification

```
go test -race ./internal/files/ -run 'TestWrite_TwoWritersIfMatchRace|TestWrite_InterruptedWritePreservesOriginal' -count=1 -v
=== RUN   TestWrite_TwoWritersIfMatchRace
--- PASS: TestWrite_TwoWritersIfMatchRace (0.06s)
=== RUN   TestWrite_InterruptedWritePreservesOriginal
--- PASS: TestWrite_InterruptedWritePreservesOriginal (0.01s)
PASS
ok      github.com/scottkw/agenthub/internal/files      1.110s

go test -race ./internal/files/ -count=1
ok      github.com/scottkw/agenthub/internal/files      6.297s
```

```
test -f .planning/phases/127-web-share-write-security-hardening/127-SECURITY.md  → 0
grep -qi "capability-escalation" 127-SECURITY.md  → match
grep -qi "accepted" 127-SECURITY.md  → match
grep -qi "loopback" 127-SECURITY.md  → match
substantive lines: 121 (requirement ≥ 40)
```

## Deviations from Plan

None — plan executed exactly as written.

- SECURITY.md was created with the exact structure prescribed (per-surface matrix + CSRF inversion + denylist threat model + STRIDE register + accepted risks log + ASVS mapping).
- SEC-05 tests use the validator-mismatch path for the interrupted-write test as the plan explicitly permits and recommends, with the choice documented in a test comment.
- The `min` helper added at the end of write_test.go shadows the Go 1.21+ built-in; `go vet` is clean and this is intentional for test readability.

## Known Stubs

None. The SECURITY artifact is a complete audit of existing code; no data is deferred or stubbed.

## Threat Flags

None. Task 1 is a documentation-only change. Task 2 is test-only (no production source modified).

## Self-Check: PASSED

- `test -f .planning/phases/127-web-share-write-security-hardening/127-SECURITY.md` → exists
- `test -f internal/files/write_test.go` → exists (modified)
- Commit `461ab09` exists: `git log --oneline --all | grep 461ab09` → confirmed
- Commit `2d8e4a8` exists: `git log --oneline --all | grep 2d8e4a8` → confirmed
- No production source modified (only `.planning/` doc + `_test.go` file)
