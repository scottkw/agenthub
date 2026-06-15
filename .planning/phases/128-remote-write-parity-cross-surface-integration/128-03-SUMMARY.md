---
phase: 128-remote-write-parity-cross-surface-integration
plan: "03"
subsystem: daemon/tui/e2e
tags: [parity, write, daemon-proxy, RemoteFilesClient, playwright, RMW-01, RMW-02, RMW-03]
dependency_graph:
  requires: [128-01, 128-02]
  provides: [write-parity-proof, persisting-fixture-peer, observer-C-e2e]
  affects: [cmd/playwright-fixture, internal/daemon, frontend/e2e]
tech_stack:
  added: []
  patterns:
    - "3-observer write-then-read parity (daemon-proxy Go + RemoteFilesClient Go + Playwright HTTPS)"
    - "files.Sandbox backed write persistence in playwright fixture (Pitfall 2 mitigation)"
    - "APIRequestContext contract-not-DOM pattern (Observer C)"
key_files:
  created:
    - internal/daemon/remote_files_write_parity_test.go
  modified:
    - cmd/playwright-fixture/main.go
    - frontend/e2e/files-browser.spec.ts
    - frontend/e2e/fixtures/remote-peer-setup.ts
decisions:
  - "newFixtureRemoteWritePeer created as a separate helper (not modifying newFixtureRemotePeer) to avoid breaking existing read-parity tests"
  - "Observer C Playwright test added as scenario 18 in files-browser.spec.ts, mirroring the contract-not-DOM pattern of scenarios 16+17"
  - "remoteFilesWriteURL added to remote-peer-setup.ts to keep URL building consistent with existing pattern"
metrics:
  duration: "~68 minutes"
  completed: "2026-06-15"
  tasks_completed: 3
  files_created: 1
  files_modified: 3
requirements: [RMW-01, RMW-02, RMW-03]
---

# Phase 128 Plan 03: 3-Observer Write Parity Harness Summary

3-observer write-then-read byte-equivalence proof (daemon-proxy Go + tui.RemoteFilesClient Go + Playwright HTTPS) against a persisting fixture peer backed by a real files.Sandbox.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Extend startRemotePeerFixture with write persistence | 6af654f | cmd/playwright-fixture/main.go |
| 2 | New package daemon_test write-then-read 3-observer parity test | 7c7fb6f | internal/daemon/remote_files_write_parity_test.go |
| 3 | Playwright HTTPS Observer C write-then-read scenario | 57f7f51 | frontend/e2e/files-browser.spec.ts, frontend/e2e/fixtures/remote-peer-setup.ts |

## What Was Built

### Task 1: Persisting Fixture Peer
Extended `startRemotePeerFixture` in `cmd/playwright-fixture/main.go` to back write verbs with a real `files.Sandbox` rooted at `os.MkdirTemp`. The `GET /api/files/read` handler now serves from the sandbox first (persisted writes), then falls back to canned "hello world" for paths never written (backward-compat with scenarios 16+17). Write routes added: `PUT /api/files/write`, `DELETE /api/files/delete`, `POST /api/files/rename`, `POST /api/files/mkdir`. The sandbox temp dir is cleaned up in the stop closure.

### Task 2: Go Write Parity Test
Created `internal/daemon/remote_files_write_parity_test.go` in `package daemon_test` (avoids tui→daemon import cycle). Tests:
- Observer A (daemon proxy) write-then-read: PUT via `/api/files/remote/sid1/write`, GET via proxy read
- Observer B (direct RemoteFilesClient) write-then-read: `WriteFile` then `ReadFile`
- Cross-observer A→B: A writes via proxy, B reads directly — byte-identical (proves shared persisted state)
- Cross-observer B→A: B writes directly, A reads via proxy — byte-identical
- `assertNoCapInError` on all direct-client error paths (T-128-08 CAP-LEAK invariant)

### Task 3: Playwright HTTPS Observer C
Added scenario 18 to `frontend/e2e/files-browser.spec.ts`: `APIRequestContext` PUT to `remoteFilesWriteURL` then GET `remoteFilesReadURL`, asserts read-back bytes equal written bytes (not canned "hello world"). Added `remoteFilesWriteURL` builder to `remote-peer-setup.ts`.

## Verification Results

- `go test ./internal/daemon/ -run TestRemoteFilesWrite_CrossSurface -race -count=1`: PASS
- `go test ./internal/daemon/ -race -count=1` (full suite): PASS (11.4s)
- `go build -tags playwrightfixture ./cmd/playwright-fixture/`: PASS
- Playwright scenario 18 chromium: PASS (182ms)
- `grep -q '^package daemon_test' internal/daemon/remote_files_write_parity_test.go`: PASS
- `grep -q 'WriteFileAtomic' cmd/playwright-fixture/main.go`: PASS

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree branch was behind main (missing 128-01+128-02 writes)**
- **Found during:** Task 1 build verification
- **Issue:** Worktree was at d725107 (v3.4.2 patch) and lacked `internal/files/write.go` (added in Phases 118-126), causing `sandbox.WriteFileAtomic` to be undefined
- **Fix:** Merged `main` (5e0bf13) into the worktree branch, bringing in all 128-01+128-02 changes
- **Files modified:** All files from the merge
- **Commit:** Merge via `git merge main --no-edit`

**2. [Rule 2 - Design] newFixtureRemoteWritePeer rather than modifying newFixtureRemotePeer**
- **Found during:** Task 2 implementation
- **Issue:** Modifying `newFixtureRemotePeer` to persist writes would change the fixture for existing read-parity tests (scenarios 16+17) which assert on canned "hello world" content
- **Fix:** Created separate `newFixtureRemoteWritePeer` helper that uses a real sandbox, preserving backward-compat

## Known Stubs

None. All write-then-read assertions use actual written bytes, not placeholders.

## Threat Flags

No new threat surface beyond the plan's threat model.

## Self-Check: PASSED

- `internal/daemon/remote_files_write_parity_test.go` exists: FOUND
- `cmd/playwright-fixture/main.go` modified (WriteFileAtomic): FOUND
- `frontend/e2e/files-browser.spec.ts` scenario 18 present: FOUND
- Task commits 6af654f, 7c7fb6f, 57f7f51: FOUND (git log verified)
