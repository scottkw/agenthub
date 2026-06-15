---
phase: 128-remote-write-parity-cross-surface-integration
plan: "01"
subsystem: remote-write-parity
tags: [rmw-04, version-gate, 405, cross-surface-parity, tdd, go, typescript]
dependency_graph:
  requires:
    - 126-01 (RemoteFilesClient write methods — WriteFile/DeleteFile/RenameFile/MkdirFile)
    - 125-03 (useFilesWrite hook — WriteOutcome, write() catch block shape)
  provides:
    - ErrRemotePeerNoWriteSupport sentinel (Go, internal/tui/remote_files_client.go)
    - remotePeerOutdatedMessage const (Go, byte-identical to TS const)
    - REMOTE_PEER_OUTDATED_MESSAGE const (TS, frontend/src/lib/filesApi.ts)
    - isMethodNotAllowed() predicate (TS, FilesApiError)
    - WriteOutcome 'peer-outdated' (TS, useFilesWrite.ts)
    - TUI toast mapping for ErrRemotePeerNoWriteSupport (update.go applyFilesOpMsg)
  affects:
    - internal/tui/update.go (applyFilesOpMsg error toast dispatch)
    - frontend/src/components/FileBrowserTab.tsx (outcome dispatch comments)
tech_stack:
  added: []
  patterns:
    - "grep-gated cross-surface parity const (byte-identical Go+TS, one const per language)"
    - "TDD RED/GREEN per language (Go test first, then TS)"
    - "errors.Is sentinel (Go) + instanceof predicate (TS) for typed error dispatch"
key_files:
  created:
    - frontend/src/lib/__tests__/useFilesWrite.test.tsx
  modified:
    - internal/tui/remote_files_client.go
    - internal/tui/update.go
    - internal/tui/remote_files_client_test.go
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesWrite.ts
    - frontend/src/components/FileBrowserTab.tsx
    - frontend/src/lib/__tests__/filesApi.test.ts
decisions:
  - "TUI surface mapping placed in update.go applyFilesOpMsg (where all op errors render as toasts) rather than files.go — that is the actual render site"
  - "FileBrowserTab 'peer-outdated' dispatch: saveError already surfaces REMOTE_PEER_OUTDATED_MESSAGE via the hook; comment added to dispatch; no nav on peer-outdated"
  - "Worktree lacked phases 123-127 code — merged local main before execution (phases 123-127 were on local main, not origin/main)"
metrics:
  duration: "~20 minutes"
  completed: "2026-06-14"
  tasks_completed: 3
  files_changed: 7
---

# Phase 128 Plan 01: Remote Write Parity — v3.4 Peer 405 Version Gate Summary

**One-liner:** JWT-free 405 version-gate with byte-identical Go+TS consts surfacing "older version" message on write against a v3.4 remote peer (RMW-04).

## What Was Built

RMW-04 satisfied: when a write verb (PUT/DELETE/POST rename/POST mkdir) hits a v3.4 remote peer that has no write routes and returns HTTP 405, both the Go TUI and TS GUI/web surface now show the verbatim message "The remote session is running an older version of AgentHub that does not support file writes." instead of a raw status code or generic network error.

### Go Surface (TUI)

- `internal/tui/remote_files_client.go`: added `remotePeerOutdatedMessage` const (verbatim SC3) and `ErrRemotePeerNoWriteSupport` sentinel (`errors.New(remotePeerOutdatedMessage)`). All 4 write methods (WriteFile, DeleteFile, RenameFile, MkdirFile) now check `resp.StatusCode == http.StatusMethodNotAllowed` BEFORE the generic `!= http.StatusOK` block and return the sentinel. Read methods (ListFiles, StatFile, ReadFile, HeadFile) are unchanged — write-only scope.
- `internal/tui/update.go`: `applyFilesOpMsg` maps `errors.Is(msg.err, ErrRemotePeerNoWriteSupport)` → `m.toast = remotePeerOutdatedMessage` (verbatim copy, no `op + " failed: "` prefix).

### TypeScript Surface (GUI/web)

- `frontend/src/lib/filesApi.ts`: added `export const REMOTE_PEER_OUTDATED_MESSAGE` (byte-identical to Go const) and `isMethodNotAllowed(): boolean { return this.status === 405 }` predicate with RMW-04 JSDoc.
- `frontend/src/lib/useFilesWrite.ts`: extended `WriteOutcome` to `'saved' | 'conflict' | 'error' | 'peer-outdated'`. Added 405 catch branch BEFORE the generic branch: `setSaveError(REMOTE_PEER_OUTDATED_MESSAGE); setSaveState('idle'); return 'peer-outdated'`. Buffer NOT cleared (T-125-08 locked).
- `frontend/src/components/FileBrowserTab.tsx`: documented `'peer-outdated'` in outcome dispatch comment.

### Cross-Surface Parity Gate (Task 3)

The verbatim string is present in exactly 1 Go const (`remote_files_client.go:1`) and 1 TS const (`filesApi.ts:1`). Grep gate passes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree missing phases 123-127 code**
- **Found during:** Pre-execution setup
- **Issue:** Worktree was forked from `3e62b81` (docs commit) which was not yet on `origin/main`. Local `main` had all phase 123-127 code; `origin/main` did not. The write methods referenced in the plan (WriteFile at line 221, etc.) were absent.
- **Fix:** `git merge main` from within the worktree — merged all phase 123-127 implementation commits cleanly with no conflicts.
- **Commit:** merge commit (no hash, fast-forward merge)

**2. [Rule 2 - Scope] TUI surface mapping moved to update.go**
- **Found during:** Task 1 implementation
- **Issue:** Plan said "In internal/tui/files.go, where remote write errors render…" but the actual render site is `update.go:applyFilesOpMsg` (filesOpMsg handler), not `files.go`.
- **Fix:** Placed `errors.Is` dispatch in `update.go:applyFilesOpMsg` — the correct render site. No behavior change vs. plan intent.
- **Files modified:** internal/tui/update.go (not files.go)

## Test Results

- `go test ./internal/tui/ -run TestRemoteFilesClient -race -count=1`: PASS
- `pnpm exec vitest run src/lib/__tests__/filesApi.test.ts src/lib/__tests__/useFilesWrite.test.tsx`: 27/27 PASS
- `pnpm exec tsc --noEmit`: clean
- `gofmt -l internal/tui/remote_files_client.go internal/tui/update.go`: no output (clean)
- `go vet ./internal/tui/`: clean

## TDD Gate Compliance

Both languages followed RED → GREEN:
- Go RED: `test(128-01)` commit `b35956e` — `TestRemoteFilesClient_405_ErrRemotePeerNoWriteSupport` (build failure, undefined `ErrRemotePeerNoWriteSupport`)
- Go GREEN: `feat(128-01)` commit `f98557c` — sentinel + const + 4 write method guards
- TS RED: `test(128-01)` commit `47b1c7d` — 5 test failures (missing predicate/const/outcome)
- TS GREEN: `feat(128-01)` commit `ab587e9` — predicate + const + WriteOutcome extension

## Known Stubs

None. All behavior is wired end-to-end:
- Go: ErrRemotePeerNoWriteSupport flows from write methods → applyFilesOpMsg → toast
- TS: FilesApiError(405) flows from catch → saveError → saveError display in FileBrowserTab

## Threat Flags

No new threat surface beyond the plan's threat model. The new message is a fixed string — no interpolated URL/cap (T-128-01 mitigated). The 405 mapping is consumer-side only — the proxy is not modified (T-128-02 mitigated).

## Self-Check

Files created/modified exist:

- [FOUND] internal/tui/remote_files_client.go
- [FOUND] internal/tui/update.go
- [FOUND] internal/tui/remote_files_client_test.go
- [FOUND] frontend/src/lib/filesApi.ts
- [FOUND] frontend/src/lib/useFilesWrite.ts
- [FOUND] frontend/src/components/FileBrowserTab.tsx
- [FOUND] frontend/src/lib/__tests__/filesApi.test.ts
- [FOUND] frontend/src/lib/__tests__/useFilesWrite.test.tsx

Commits:

- b35956e test(128-01): RED — 405 ErrRemotePeerNoWriteSupport sentinel
- f98557c feat(128-01): Go 405 sentinel + const + TUI surface copy
- 47b1c7d test(128-01): RED — isMethodNotAllowed + REMOTE_PEER_OUTDATED_MESSAGE
- ab587e9 feat(128-01): TS isMethodNotAllowed + const + peer-outdated WriteOutcome

## Self-Check: PASSED
