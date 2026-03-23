---
phase: 16-auth-layer-removal
plan: "01"
subsystem: webserver
tags: [auth-removal, security, tailscale, go, refactor]
dependency_graph:
  requires: [Phase 15 - TLS/IP binding complete]
  provides: [Clean Go backend with no auth infrastructure]
  affects: [app.go, internal/webserver/server.go, internal/webserver/server_test.go, app_test.go]
tech_stack:
  added: []
  patterns: [Tailscale network-level access control, open HTTP routes]
key_files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/server_test.go
    - app.go
    - app_test.go
    - go.mod
  deleted:
    - internal/webserver/auth.go
    - internal/webserver/auth_test.go
    - internal/webserver/tokens.go
    - internal/webserver/tokens_test.go
decisions:
  - "Auth removal: All app-layer auth removed; Tailscale provides network-level access control"
  - "golang.org/x/crypto moved from direct to indirect dependency after go mod tidy"
  - "TestStartWebServerNoPasswordRequired adapted to handle both connected and disconnected Tailscale environments"
metrics:
  duration: "7m 25s"
  completed_date: "2026-03-20"
  tasks_completed: 2
  files_changed: 9
---

# Phase 16 Plan 01: Auth Layer Removal Summary

**One-liner:** Deleted auth.go/tokens.go, removed dashboardAuth/sessionAuth middleware and all auth methods from WebServer and App; all routes now open with Tailscale providing network-level access control.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Delete auth files, strip auth from WebServer struct and routes | 345a46c | auth.go (deleted), tokens.go (deleted), server.go, server_test.go |
| 2 | Update app.go, rewrite all Go tests, run go mod tidy | 796de23 | app.go, app_test.go, go.mod |

## What Was Built

Removed all application-layer authentication infrastructure from the AgentHub backend:

**Deleted files:**
- `internal/webserver/auth.go` — AuthManager with bcrypt password hashing, session cookie management
- `internal/webserver/auth_test.go` — Tests for auth manager
- `internal/webserver/tokens.go` — TokenStore for one-time session sharing tokens
- `internal/webserver/tokens_test.go` — Tests for token store

**Modified files:**
- `internal/webserver/server.go` — Removed `auth *AuthManager` and `tokens *TokenStore` fields from WebServer struct; removed `SetPassword`, `LoadPasswordHash`, `IsPasswordSet`, `CreateToken` methods; removed `dashboardAuth` and `sessionAuth` middleware; removed `handleLogin` and `handleCreateToken` handlers; removed `simpleCookieJar` dead code; removed `net/url` import; rewrote `setupRoutes` to serve all endpoints without auth wrapping; updated WebSocket origin comment
- `internal/webserver/server_test.go` — Removed `testCookieJar`, `login` helper, auth-testing tests; rewrote `testServer` helper without SetPassword; added `TestLoginRouteNotRegistered`, `TestTokenRouteNotRegistered`, `TestSessionAccessWithoutAuth`
- `app.go` — Removed `SetWebPassword`, `IsWebPasswordSet`, `GenerateSessionToken`, `webPasswordPath`; removed password gate and `LoadPasswordHash` call from `StartWebServer`
- `app_test.go` — Removed `TestSetWebPasswordPersistsAndReloads`, `TestStartWebServerErrorsWhenPasswordNotSet`, `testAppWithConfigDir`; added `TestStartWebServerNoPasswordRequired`; fixed `TestGetSessionQRCode` to not call `ws.SetPassword`
- `go.mod` — `golang.org/x/crypto` moved from direct to indirect dependency

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] server_test.go rewritten in Task 1 instead of Task 2**
- **Found during:** Task 1 verification (`go build ./internal/webserver/...`)
- **Issue:** Test file in the same package still referenced `ws.SetPassword` which was removed in Task 1. Build fails with `ws.SetPassword undefined`.
- **Fix:** Rewrote server_test.go as part of Task 1 to unblock the build verification. Task 2 then only needed to handle app.go and app_test.go.
- **Files modified:** `internal/webserver/server_test.go`
- **Commit:** 345a46c

**2. [Rule 1 - Bug] TestStartWebServerNoPasswordRequired adapted for real Tailscale environment**
- **Found during:** Task 2 test run
- **Issue:** The plan's test assumed Tailscale is NOT connected in the test environment, but the machine running tests has a live Tailscale connection with HTTPS certs enabled. The original test body `t.Fatal("expected error (Tailscale not connected), got nil")` would fail.
- **Fix:** Rewrote test to handle both cases: if StartWebServer succeeds (Tailscale connected), clean up and pass; if it errors, assert the error doesn't mention "password". The core assertion — no password requirement — is preserved in both branches.
- **Files modified:** `app_test.go`
- **Commit:** 796de23

## Test Results

```
ok  github.com/agenthub/agenthub                    1.607s
ok  github.com/agenthub/agenthub/internal/pty       2.789s
ok  github.com/agenthub/agenthub/internal/relay     2.808s
ok  github.com/agenthub/agenthub/internal/status    1.857s
ok  github.com/agenthub/agenthub/internal/webserver 2.655s
```

All tests pass with `-race` flag.

## Self-Check: PASSED

- auth.go deleted: confirmed
- auth_test.go deleted: confirmed
- tokens.go deleted: confirmed
- tokens_test.go deleted: confirmed
- server.go has no auth references: confirmed
- app.go has no password methods: confirmed
- Commits 345a46c and 796de23 exist: confirmed
- Full test suite green with -race: confirmed
