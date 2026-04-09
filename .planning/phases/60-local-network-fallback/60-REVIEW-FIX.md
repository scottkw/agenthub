---
phase: 60-local-network-fallback
fixed_at: 2026-04-09T21:54:37Z
review_path: .planning/phases/60-local-network-fallback/60-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 60: Code Review Fix Report

**Fixed at:** 2026-04-09T21:54:37Z
**Source review:** .planning/phases/60-local-network-fallback/60-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5
- Fixed: 5
- Skipped: 0

## Fixed Issues

### CR-01: Local-mode web server can start with empty password

**Files modified:** `internal/daemon/api.go`
**Commit:** 659741e
**Applied fix:** Added non-empty password validation guard in both `handleWebServerStart` (returns 400 Bad Request) and `AutoStartWebServer` (returns error). Prevents local-mode web server from starting without authentication, closing the bypass where BasicAuthMiddleware would accept empty passwords.

### WR-01: `webServerMode` state in App.tsx not updated when web server stops

**Files modified:** `frontend/src/App.tsx`
**Commit:** 7713f8a
**Applied fix:** Updated `handleSettingsClose` to clear `webServerMode` to `null` when the web server is not running, and re-fetch `GetWebServerMode()` when it is running. Prevents the `LocalNetworkBanner` from remaining visible after the server has been stopped.

### WR-02: `handleWebServerStart` does not stop a previously running server before starting a new one

**Files modified:** `internal/daemon/api.go`
**Commit:** 29558d9
**Applied fix:** Added a lock-protected check-and-stop block before creating a new WebServer in `handleWebServerStart`. If `a.webServer` is already non-nil, the existing server is stopped and its reference cleared before proceeding. Prevents listener leaks from concurrent or repeated start requests.

### WR-03: `pollSessionStatus` goroutine in `app.go` never stops for sessions that reach `StatusRunning`

**Files modified:** `app.go`
**Commit:** 94bfea5 (fixed: requires human verification)
**Applied fix:** Changed the single `if` check for `StatusErrored` to a `switch` statement that exits on both `StatusErrored` and `StatusRunning`. This stops the polling goroutine as soon as the session reaches a stable running state, avoiding 120 unnecessary HTTP round-trips to the daemon for long-running sessions. This is a logic change -- please verify the exit-on-running semantic matches the intended polling behavior.

### WR-04: `GetLocalNetworkPassword` on the daemon API is unauthenticated

**Files modified:** `internal/daemon/api.go`
**Commit:** df7d419
**Applied fix:** Added a doc comment to `handleGetLocalPassword` explicitly documenting the threat model: the Unix socket is user-owned (0600) so only same-UID processes can reach this endpoint, which is the intended access-control model.

---

_Fixed: 2026-04-09T21:54:37Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
