---
phase: 04-web-serving-tls-auth
plan: 03
subsystem: webserver
tags: [go, https, tls, websocket, auth, embed, xterm, dashboard]
dependency_graph:
  requires:
    - 04-01 (tls.go: LoadOrCreateCA, GenerateLeafCert, BuildTLSConfig)
    - 04-02 (auth.go: AuthManager; tokens.go: TokenStore)
    - internal/relay (HubManager, Hub, Subscriber, ParseFrame, MsgOutput/MsgInput/MsgResize2/MsgPing)
  provides:
    - WebServer struct with Start/Stop, route registration, auth middleware
    - web/embed.go: go:embed directive for dashboard.html and terminal.html
    - web/dashboard.html: login form, session list, copy-token-link, CA cert install guidance
    - web/terminal.html: xterm.js 6 CDN terminal with binary framing protocol
  affects:
    - Phase 5: CLI status indicators — web serving can be toggled per session from App.go
    - Phase 6: Cross-platform — CA cert install path used in dashboard CA guidance section
tech_stack:
  added: []
  patterns:
    - "go:embed in separate web/ package — embed prohibits ../ paths; web/ is at project root"
    - "subscribe-before-snapshot pattern in WSS relay — same as relay/server.go handleSession"
    - "EADDRINUSE fallback to :0 — prevents startup failure if preferred port is taken"
    - "simpleCookieJar in server.go — minimal CookieJar for test cookie persistence without import cycle"
    - "sessionAuth checks token first, cookie second — token access does not require dashboard login"
key_files:
  created:
    - web/embed.go
    - web/dashboard.html
    - web/terminal.html
    - internal/webserver/server.go
    - internal/webserver/server_test.go
  modified: []
key_decisions:
  - "OriginPatterns: []string{\"*\"} in websocket.Accept — sessionAuth middleware already validated the request; no additional origin check needed"
  - "simpleCookieJar lives in server.go (not a test file) — avoids _test.go package boundary for TestClient method"
  - "sessionAuth returns 401 if token is provided but invalid (not 404) — prevents token enumeration via 404 vs 401 difference on disabled vs bad-token"
  - "webEnabled map controls /sessions/{id} route separately from HubManager — session can be enabled before hub is created"
metrics:
  duration: 4min
  completed_date: "2026-03-18"
  tasks_completed: 2
  files_created: 5
requirements_satisfied: [WEB-01, WEB-03, WEB-06]
---

# Phase 4 Plan 03: HTTPS Web Server with Auth and WSS Relay Summary

**One-liner:** HTTPS WebServer with embedded dashboard/terminal HTML, bcrypt-auth cookie sessions, per-session token links, and subscribe-before-snapshot WSS relay using CA-signed TLS.

## What Was Built

### Task 1: Embedded Web Assets

- `web/embed.go` — `//go:embed dashboard.html terminal.html` directive; exports `WebFS embed.FS`
- `web/dashboard.html` — single-page dashboard with:
  - Login form (fetch API POST /login, stays on page)
  - Session list from GET /api/sessions with "Open" and "Copy Token Link" per session
  - CA cert installation guidance for macOS (Keychain/security), Linux (update-ca-certificates/update-ca-trust), Windows (certutil) — satisfies WEB-03
  - "Download CA Certificate" link to `/ca.crt`
  - Auto-login-state detection on page load
- `web/terminal.html` — xterm.js 6 terminal page with:
  - CDN loads for @xterm/xterm@6 and @xterm/addon-fit@0.11
  - Session ID extracted from URL path `/sessions/{id}`
  - Optional token from `?token=xxx` query param
  - WSS connection to `wss://${location.host}/sessions/${sessionID}/ws`
  - Binary framing: MsgOutput(0x01), MsgInput(0x10), MsgResize2(0x11)
  - fitAddon.fit() on load and window resize

### Task 2: WebServer Struct with Routes, Auth, and WSS Relay

**Config struct:** BindIP, Port (default 7443 with :0 fallback), ConfigDir

**WebServer struct:** Config, AuthManager, TokenStore, HubManager, webEnabled map, listener, mux, caKey, caCert, caDER, tlsCfg

**Constructor:** `NewWebServer(cfg Config, manager *relay.HubManager)` — calls LoadOrCreateCA, initializes all stores, registers routes

**Routes registered:**
- `GET /` → redirect to /dashboard
- `GET /dashboard` — dashboardAuth → serves dashboard.html
- `POST /login` — JSON {password} → bcrypt validate → Set-Cookie agenthub_session
- `GET /api/sessions` — dashboardAuth → JSON array of web-enabled session IDs
- `GET /sessions/{id}` — sessionAuth → serves terminal.html
- `GET /sessions/{id}/ws` — sessionAuth → WebSocket upgrade + relay
- `POST /api/sessions/{id}/token` — dashboardAuth → creates token, returns {token, url}
- `GET /ca.crt` — no auth → PEM CA cert with Content-Type application/x-pem-file

**Auth middleware:**
- `dashboardAuth`: validates agenthub_session cookie via AuthManager.IsAuthenticated
- `sessionAuth`: checks webEnabled map (404 if disabled), token param first, cookie fallback

**Toggle:** EnableSession/DisableSession update webEnabled map under RWMutex

**Start:** GenerateLeafCert → BuildTLSConfig → tls.Listen (EADDRINUSE fallback to :0) → go http.Serve

**WSS relay:** identical subscribe-before-snapshot pattern from relay/server.go — Subscribe, ScrollbackSnapshot replay, read pump (MsgInput/MsgResize2/MsgPing), write pump from hub.Msgs channel

**TestClient():** returns *http.Client with CA-trusting TLS transport and simpleCookieJar for cookie persistence in tests

## Verification

All 32 tests pass, full project compiles:

```
go test ./internal/webserver/... -v -timeout 30s  # PASS (32 tests)
go build ./...                                     # OK
```

## Task Commits

| Task | Description | Commit |
|------|-------------|--------|
| 1 | feat: embedded web assets (dashboard + terminal HTML) | d9dcd1d |
| 2 RED | test: failing tests for WebServer | ce8abf6 |
| 2 GREEN | feat: WebServer implementation | 7266c6c |

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

One minor fix during GREEN: added `net/url` import (forgotten) and removed unused `context` import in server.go — caught by compiler immediately and resolved in the same edit before commit.

## Self-Check: PASSED

- FOUND: web/embed.go
- FOUND: web/dashboard.html
- FOUND: web/terminal.html
- FOUND: internal/webserver/server.go
- FOUND: internal/webserver/server_test.go
- FOUND: commit d9dcd1d (feat - web assets)
- FOUND: commit ce8abf6 (test RED - server tests)
- FOUND: commit 7266c6c (feat GREEN - server implementation)
