---
phase: 128-remote-write-parity-cross-surface-integration
reviewed: 2026-06-15
depth: focused (orchestrator direct inspection)
files_reviewed: 4
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Phase 128: Code Review

Net-new code is two small cross-surface error mappings (RMW-04 405, RMW-05 401) plus a test-fixture extension and the parity test. Orchestrator directly inspected the deltas.

## Assessment — CLEAN

- **Go (`internal/tui/remote_files_client.go`):** 405 → `ErrRemotePeerNoWriteSupport` (verbatim `remotePeerOutdatedMessage`), 401 → `ErrRemoteCapExpired` (verbatim `remoteCapExpiredMessage`), mapped in all 4 write methods before the generic status block. Cap-leak invariant holds: `redactCapFromURL` present; sentinel errors carry only the user-facing message (no URL/cap); the parity + client tests assert `assertNoCapInError`.
- **TS (`filesApi.ts` / `useFilesWrite.ts`):** `WriteOutcome` extended additively to `'peer-outdated'` / `'expired'`; `isMethodNotAllowed()` / `isUnauthorized()` predicates mirror `isConflict()`; both branches set the verbatim message and explicitly DO NOT clear the buffer (T-125-08 preserved — regression-guarded by tests).
- **Cross-surface parity (release-blocking):** the 405 message is a single byte-identical const in Go and TS, grep-gated in 128-01.
- **Parity harness (`remote_files_write_parity_test.go`):** `package daemon_test` (cycle-safe), persisting `files.Sandbox` fixture (real write-then-read byte-equivalence, not canned), 3 observers, cap-leak asserted.
- **Coverage:** Phase 122 remote-read suite passes zero regressions; full v3.5 Go surface (files/daemon/webserver/tui) green under `-race`.

**Verdict:** No findings. The error-mapping deltas are minimal, well-patterned, parity-gated, and cap-leak-safe. The bulk of remote-write plumbing was consumed as-is from Phases 123-127 (Chesterton's Fence honored).
