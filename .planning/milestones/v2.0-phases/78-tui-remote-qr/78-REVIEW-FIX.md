---
phase: 78-tui-remote-qr
fixed_at: 2026-04-15T14:30:00Z
review_path: .planning/phases/78-tui-remote-qr/78-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 78: Code Review Fix Report

**Fixed at:** 2026-04-15T14:30:00Z
**Source review:** .planning/phases/78-tui-remote-qr/78-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: Silent error discarding in fetchRemoteFn callback (cmd_tui.go)

**Files modified:** `cmd_tui.go`
**Commit:** b46fa47
**Applied fix:** Replaced silent error discarding (`_, _`) on both `client.ListTailnetPeers()` and `fetchPeerSessions()` with explicit error handling. Errors are now logged at warn level using `log.Printf` (consistent with the project's existing logging convention). `ListTailnetPeers` failure returns nil early; `fetchPeerSessions` failure logs and continues to the next peer. Added `"log"` import.

### WR-02: Potential nil pointer dereference if entryLocal has nil session (view.go)

**Files modified:** `internal/tui/view.go`
**Commit:** a740ba0
**Applied fix:** Added defensive nil guards for `entry.session` (entryLocal) and `entry.remote` (entryRemote) in the render loop at `renderSessionList`, consistent with the nil checks already present in `handleMainKey` at `update.go:263-264`. If either pointer is nil, the row is silently skipped rather than panicking.

## Skipped Issues

None -- all in-scope findings were fixed.

---

_Fixed: 2026-04-15T14:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
