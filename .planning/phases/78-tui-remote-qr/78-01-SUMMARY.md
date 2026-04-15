---
phase: 78
plan: "01"
subsystem: tui
tags: [tui, remote-sessions, unified-list, bubble-tea, tailnet]
dependency_graph:
  requires: []
  provides: [unified-session-list, remote-session-fetch, divider-navigation, remote-blocked-toasts]
  affects: [internal/tui, cmd_tui.go]
tech_stack:
  added: []
  patterns: [callback-injection-for-import-cycle, unified-list-model, divider-skip-navigation]
key_files:
  created: []
  modified:
    - internal/tui/model.go
    - internal/tui/cmds.go
    - internal/tui/tui.go
    - internal/tui/update.go
    - internal/tui/view.go
    - internal/tui/update_test.go
    - internal/tui/view_test.go
    - internal/tui/attach_test.go
    - cmd_tui.go
decisions:
  - "Use callback injection (FetchRemoteFn) to avoid package-main import cycle; cmd_tui.go provides the closure that wraps ListTailnetPeers + fetchPeerSessions"
  - "Exported RemoteSessionEntry and ListRemoteGroup types so cmd_tui.go (package main) can construct callback return values"
  - "Remote attach deferred to future phase with toast 'Remote attach not yet supported' per plan scope decision"
  - "rebuildUnifiedList() called by both sessionsMsg and remoteSessionsMsg handlers to keep cursor identity-stable across refreshes"
metrics:
  duration: "~20 minutes"
  completed: "2026-04-15"
  tasks: 3
  files: 9
---

# Phase 78 Plan 01: Unified Session List with Remote Tailnet Peers Summary

**One-liner:** Unified TUI session list model with async remote tailnet peer fetching, divider rows grouped by hostname, divider-skip j/k navigation, and remote-blocked kill/rename toasts via FetchRemoteFn callback injection.

## What Was Built

Plan 01 satisfies TUI-07 by integrating remote tailnet peer sessions into the Bubble Tea TUI session list. The flat `[]daemon.SessionInfo` list is replaced by a unified `[]listEntry` slice that interleaves local sessions, per-peer divider rows, and remote sessions. Remote data is fetched asynchronously on each 2-second tick without blocking local session refresh.

### Core Data Model (model.go)

New types added:
- `listEntry` / `listEntryKind` — unified row type with `entryLocal`, `entryRemote`, `entryDivider` variants
- `RemoteSessionEntry` — exported struct mirroring `CLIRemoteSession` fields plus pre-built URL
- `ListRemoteGroup` — exported struct grouping remote sessions by peer hostname
- `FetchRemoteFn` — exported function type for the callback injected from `cmd_tui.go`
- `peerDivider` — metadata for section divider rows (hostname, session count)
- `sessionRef` — identity capture for QR overlay (Plan 02) and selection restoration
- `remoteSessionsMsg` — Bubble Tea message carrying fetched remote groups

New Model fields: `remoteSessions`, `unifiedList`, `fetchRemoteFn`, `qrSession`, `qrContent`, `qrURL`

### Async Fetch (cmds.go + tui.go)

`fetchRemoteSessions(fn FetchRemoteFn) tea.Cmd` fires the callback in a goroutine with a 10-second context timeout. Returns empty `remoteSessionsMsg` when callback is nil (no tailnet configured). Added to both `Init()` and `tickMsg` batches alongside `fetchSessions` and `fetchWebStatus`.

### Callback Injection (cmd_tui.go)

`cmdTUI` now constructs a `fetchRemoteFn` closure that:
1. Calls `client.ListTailnetPeers()` to get tailnet peers
2. For each peer, calls `fetchPeerSessions(ctx, fqdn, tailnet.DefaultProbePort)`
3. Builds `tui.RemoteSessionEntry` values with pre-constructed HTTPS URLs
4. Groups by hostname (alphabetically sorted) into `[]tui.ListRemoteGroup`

This avoids the `internal/tui` → `package main` import cycle.

### Unified List Logic (update.go)

`rebuildUnifiedList()` rebuilds `m.unifiedList` from `m.sessions` + `m.remoteSessions` on every `sessionsMsg` or `remoteSessionsMsg`. Selection is restored by entry identity (session ID + kind) to prevent cursor jump on refresh. Falls back to first selectable entry if identity not found.

Navigation handlers updated:
- `j`/`k`: iterate `m.unifiedList` skipping `entryDivider` entries
- `g`/`G`: jump to first/last selectable entry (non-divider)
- Attach: handles `entryLocal` (existing flow) and `entryRemote` (toast: "Remote attach not yet supported")
- Kill: blocks on `entryRemote` with toast "Cannot kill remote session"
- Rename: blocks on `entryRemote` with toast "Cannot rename remote session"

### View Updates (view.go)

- `renderHeader`: shows "N local, M remote" when remote sessions present; unchanged "N sessions" otherwise
- `renderSessionList`: dispatches per `entry.kind` to `renderSessionRow`, `renderRemoteSessionRow`, or `renderDividerRow`
- `renderRemoteSessionRow`: same column layout as local rows; viewers column always blank
- `renderDividerRow`: `── Remote: {hostname} ({N} session/sessions) ──────...` with FgAccent label and FgMuted fill chars (U+2500)
- `visibleRange`: updated to use `len(m.unifiedList)`
- `renderFull`: comment placeholder for QR overlay (Plan 02)

### Tests (10 new + 7 updated)

New tests:
- `TestUpdate_RemoteSessionsMsg` — unified list has 4 entries for 1 local + 2 remote
- `TestUpdate_NavigationSkipsDividers` — j skips divider, lands on remote; k skips back
- `TestUpdate_KillRemoteBlocked` — toast shown, no kill modal
- `TestUpdate_RenameRemoteBlocked` — toast shown, editing=false
- `TestUpdate_SelectionRestoredAfterRebuild` — remote cursor survives local session refresh
- `TestUpdate_UnifiedListEmpty` — navigation is panic-free with empty list
- `TestView_HeaderRemoteCount` — "2 local, 3 remote" header
- `TestView_DividerRow` — box-drawing chars, singular/plural session count
- `TestView_RemoteSessionRow` — name/agent/hostname fields render correctly
- `TestView_SessionListWithRemotes` — full view contains local, remote, and divider content

Updated existing tests to call `m.rebuildUnifiedList()` after directly setting `m.sessions` (required because navigation/operations now go through `m.unifiedList` instead of `m.sessions`):
- `TestUpdate_KeyNavigation`, `TestUpdate_KeyReassignment`, `TestUpdate_KillConfirmOpen`
- `TestUpdate_RenameStart`, `TestView_SessionList`, `TestView_SingleSession`, `TestView_InlineRename`
- `TestUpdate_AttachDispatch`, `TestUpdate_AttachErroredSession`

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 657cbe6 | feat(78-01): add unified list types, FetchRemoteFn callback, fetchRemoteSessions cmd |
| 2 | 61a60b1 | feat(78-01): implement unified list rebuild, divider-skip navigation, remote-blocked toasts |
| 3 | 51b5900 | test(78-01): add comprehensive unit tests for remote sessions, unified list, navigation |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fix existing tests broken by unifiedList migration**
- **Found during:** Task 2 verification
- **Issue:** 9 existing tests set `m.sessions` directly without calling `rebuildUnifiedList()`. After navigation/operations moved from `m.sessions` to `m.unifiedList`, these tests produced wrong results (selected stayed at 0, kill modal didn't open, rename didn't start).
- **Fix:** Added `m.rebuildUnifiedList()` call after `m.sessions =` assignment in: `attach_test.go` (2 tests), `update_test.go` (4 tests), `view_test.go` (3 tests).
- **Files modified:** `internal/tui/attach_test.go`, `internal/tui/update_test.go`, `internal/tui/view_test.go`
- **Commit:** 61a60b1

## Known Stubs

- **Remote attach** (`internal/tui/update.go`, Attach handler `entryRemote` case): shows toast "Remote attach not yet supported". Per plan scope decision (RESEARCH.md Open Questions item 1, RESOLVED: deferred). A future phase will implement WSS attach to peer FQDN.

## Threat Flags

None — all security-relevant surfaces were in-scope and analyzed by the plan's threat model. Remote sessions are read-only from TUI; kill/rename blocked with toasts per ASVS V4 Access Control.

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| internal/tui/model.go exists | FOUND |
| internal/tui/cmds.go exists | FOUND |
| internal/tui/tui.go exists | FOUND |
| internal/tui/update.go exists | FOUND |
| internal/tui/view.go exists | FOUND |
| cmd_tui.go exists | FOUND |
| Commit 657cbe6 exists | FOUND |
| Commit 61a60b1 exists | FOUND |
| Commit 51b5900 exists | FOUND |
| `go test ./internal/tui/... -count=1` | PASS (66 tests) |
| `go build ./...` | PASS |
