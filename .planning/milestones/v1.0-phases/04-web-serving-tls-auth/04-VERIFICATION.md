---
phase: 04-web-serving-tls-auth
verified: 2026-03-18T19:00:00Z
status: human_needed
score: 17/17 must-haves verified
human_verification:
  - test: "Open desktop app, set password in Settings, start web server, enable web serving for a session, open HTTPS URL in browser — verify full auth + terminal interaction flow"
    expected: "Browser reaches dashboard with login form, login succeeds, session terminal renders xterm.js, keystrokes reach PTY, output appears in browser"
    why_human: "End-to-end flow requires a real running app with a PTY session, a browser, and human keyboard/visual confirmation of bidirectional terminal I/O over WSS"
  - test: "Generate token link from desktop app, open in incognito window"
    expected: "Page connects directly to session without password prompt"
    why_human: "Token auth bypass requires live session, live WebSocket connection, and visual confirmation of no-password access"
  - test: "Download /ca.crt from browser, install into OS trust store, reload page"
    expected: "Browser no longer shows TLS warning after CA cert is installed"
    why_human: "OS trust store installation and browser trust behavior cannot be verified programmatically"
---

# Phase 4: Web Serving, TLS, and Auth Verification Report

**Phase Goal:** Web serving with TLS and authentication for remote browser access to terminal sessions
**Verified:** 2026-03-18T19:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

All automated checks passed across all four plans. The infrastructure (TLS/CA, auth, token store, WebServer routes, WSS relay) is fully implemented, tested, and wired into the desktop app. Three items require human visual verification to confirm the end-to-end user experience.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | CA cert is generated once and persisted to ~/.config/agenthub/ | VERIFIED | `LoadOrCreateCA` writes `ca.crt`/`ca.key` to `dir` under `0700`; `TestLoadOrCreateCA` passes |
| 2 | Leaf cert is CA-signed with SAN IPAddresses and trusted by x509.Verify | VERIFIED | `GenerateLeafCert` sets `IPAddresses: []net.IP{bindIP}`; `TestGenerateLeafCert` passes with `x509.Verify` |
| 3 | CA cert and key are loaded from disk on subsequent launches | VERIFIED | `LoadOrCreateCA` reads `ca.crt`/`ca.key` when they exist; `TestLoadOrCreateCA` validates identity |
| 4 | Network interfaces are enumerated excluding loopback and link-local | VERIFIED | `ListInterfaces` skips `FlagLoopback`, `IsLoopback()`, `IsLinkLocalUnicast()` IPs; `TestListInterfaces` passes |
| 5 | Tailscale interfaces are detected via 100.64.0.0/10 CGNAT range | VERIFIED | `IsTailscaleIP` uses package-level `tailscaleCIDR`; `TestIsTailscaleIP` passes all four test IPs |
| 6 | Dashboard password is bcrypt-hashed, never stored in plaintext | VERIFIED | `HashPassword` uses `bcrypt.GenerateFromPassword`; `SetWebPassword` persists hash (not plaintext) to disk |
| 7 | Correct password sets an httpOnly session cookie | VERIFIED | `Login` → `MakeSessionCookie` sets `HttpOnly: true, Secure: true, SameSite: Strict`; `TestAuthManagerLogin` and `TestSessionCookieProperties` pass |
| 8 | Invalid password returns 401 without setting a cookie | VERIFIED | `handleLogin` returns 401 on `ErrInvalidPassword`; `TestWebServerLoginBadPassword` passes |
| 9 | Per-session opaque token grants access to exactly one session | VERIFIED | `sessionAuth` calls `tokens.Lookup(tok)` and checks `sid == sessionID`; `TestWebServerTokenAccess` passes |
| 10 | Invalid or missing token returns 401 | VERIFIED | `sessionAuth` returns 401 when token provided but invalid; `TestWebServerTokenAccessInvalid` passes |
| 11 | Tokens are cryptographically random 32-byte values | VERIFIED | `GenerateToken` uses `crypto/rand.Read(buf)` with 32-byte buf; `TestTokenLength` verifies 43-char base64url |
| 12 | WebServer starts an HTTPS listener on the specified bind IP | VERIFIED | `Start()` calls `tls.Listen("tcp", bindIP+":"+port, tlsCfg)`; `TestWebServerDashboardRequiresAuth` exercises a live TLS server |
| 13 | Dashboard page requires password authentication to access | VERIFIED | `dashboardAuth` middleware checks `agenthub_session` cookie; `TestWebServerDashboardRequiresAuth` passes |
| 14 | Dashboard lists all web-served sessions | VERIFIED | `handleListSessions` returns `webEnabledSessions()`; `TestWebServerSessionListAPI` passes |
| 15 | Per-session token URL bypasses dashboard auth and connects to exactly that session | VERIFIED | `sessionAuth` checks token first via `tokens.Lookup`; `TestWebServerTokenAccess` passes |
| 16 | Remote browser connects via WSS and receives MsgOutput frames | VERIFIED | `handleWSSRelay` sends `ScrollbackSnapshot()` then relays `sub.Msgs`; `TestWebServerWSS` passes |
| 17 | Remote browser can send MsgInput frames back to the PTY | VERIFIED | WSS read pump parses `MsgInput` and calls `hub.WriteInput(payload)`; verified in `TestWebServerWSS` and code review of `handleWSSRelay` |

**Score:** 17/17 truths verified (automated)

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/webserver/tls.go` | CA generation, persistence, leaf cert generation, tls.Config builder | VERIFIED | 163 lines; exports `GenerateCA`, `LoadOrCreateCA`, `GenerateLeafCert`, `BuildTLSConfig`, `ExportCACertPath` |
| `internal/webserver/network.go` | Network interface enumeration, Tailscale detection | VERIFIED | 77 lines; exports `NetworkInterface`, `ListInterfaces`, `IsTailscaleIP` |
| `internal/webserver/auth.go` | Password hashing, cookie session management, auth middleware | VERIFIED | 114 lines; exports `HashPassword`, `CheckPassword`, `AuthManager`, `NewAuthManager`, `ErrInvalidPassword` |
| `internal/webserver/tokens.go` | Opaque token generation and lookup | VERIFIED | 109 lines; exports `TokenStore`, `GenerateToken`, `NewTokenStore` |
| `internal/webserver/server.go` | WebServer struct with Start/Stop, route registration, auth middleware | VERIFIED | 485 lines; exports `WebServer`, `NewWebServer`, `Config`; all 8 routes registered |
| `web/dashboard.html` | Embedded HTML dashboard page with login form and CA guidance | VERIFIED | Login form present; GET /api/sessions fetches sessions; CA guidance section with macOS/Linux/Windows tabs; `/ca.crt` download link |
| `web/terminal.html` | Embedded HTML terminal page with xterm.js from CDN | VERIFIED | xterm.js 6 loaded from jsDelivr; WSS connection to `wss://${location.host}/sessions/${sessionID}/ws`; binary framing MsgOutput/MsgInput/MsgResize2 implemented |
| `web/embed.go` | go:embed directive for web/ directory | VERIFIED | `//go:embed dashboard.html terminal.html` present; exports `WebFS` |
| `app.go` | Wails-bound methods for web server control | VERIFIED | 10 bound methods: `SetWebPassword`, `IsWebPasswordSet`, `GetNetworkInterfaces`, `StartWebServer`, `StopWebServer`, `ToggleWebServing`, `GenerateSessionToken`, `GetWebServerURL`, `GetCACertPath`, `IsWebServerRunning` |
| `frontend/src/components/SettingsPanel.tsx` | Web serving settings UI (password, interface selector, CA guidance) | VERIFIED | Password field with checkmark, network interface dropdown with Tailscale auto-detection, Start/Stop button with URL display, CA cert path and platform-specific instructions |
| `frontend/src/App.tsx` | Per-tab web serving toggle, URL display, token copy | VERIFIED | `web-serving-bar` per tab, `handleToggleWeb` calls `ToggleWebServing`, URL displayed, "Copy Token Link" calls `GenerateSessionToken` |
| `frontend/src/wailsjs/go/main/App.d.ts` | TypeScript declarations for all 10 new bound methods | VERIFIED | All 10 method signatures present with correct parameter/return types |
| `frontend/src/wailsjs/go/main/App.js` | JS stubs for all 10 new bound methods | VERIFIED | All 10 exports via `Call('main.App.MethodName', [...])` |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/webserver/tls.go` | `crypto/x509` | `x509.CreateCertificate` for CA and leaf | WIRED | Line 36 (CA), line 118 (leaf) |
| `internal/webserver/network.go` | `net.Interfaces` | stdlib interface enumeration | WIRED | Line 31 |
| `internal/webserver/auth.go` | `golang.org/x/crypto/bcrypt` | `bcrypt.CompareHashAndPassword` + `bcrypt.GenerateFromPassword` | WIRED | Lines 19, 24 |
| `internal/webserver/tokens.go` | `crypto/rand` | `rand.Read` for token generation | WIRED | Line 13 |
| `internal/webserver/server.go` | `internal/webserver/tls.go` | `BuildTLSConfig` for HTTPS listener | WIRED | Line 140 |
| `internal/webserver/server.go` | `internal/webserver/auth.go` | `ws.auth.IsAuthenticated` / `ws.auth.Login` | WIRED | Lines 258, 291, 320, 326 |
| `internal/webserver/server.go` | `internal/webserver/tokens.go` | `ws.tokens.Lookup` for session token auth | WIRED | Line 279 |
| `internal/webserver/server.go` | `internal/relay` | `ws.manager.Get` for session WebSocket relay | WIRED | Line 383 |
| `web/terminal.html` | `internal/webserver/server.go` | WSS connection to `/sessions/{id}/ws` | WIRED | `wss://${location.host}/sessions/${sessionID}/ws` at line 57 |
| `app.go` | `internal/webserver/server.go` | `a.webServer.Start/Stop/EnableSession/DisableSession` | WIRED | `a.webServer` field accessed in all 10 bound methods; `ws.Start()`, `ws.Stop()`, `ws.EnableSession()`, `ws.DisableSession()` called |
| `frontend/src/App.tsx` | `app.go` | Wails bindings for web serving toggle | WIRED | `ToggleWebServing` and `GetWebServerURL` imported and called in `handleToggleWeb` |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| WEB-01 | 04-03, 04-04 | User can toggle web serving on/off per session | SATISFIED | `EnableSession`/`DisableSession` in server.go; `ToggleWebServing` in app.go; per-tab toggle in App.tsx |
| WEB-02 | 04-01 | Web-served sessions use self-signed TLS with local CA cert pattern | SATISFIED | `LoadOrCreateCA` + `GenerateLeafCert` + `BuildTLSConfig`; TLS 1.2 min; leaf cert CA-signed with SAN IPAddresses |
| WEB-03 | 04-03, 04-04 | App provides in-app guidance for installing CA cert in OS trust store | SATISFIED | CA guidance in `dashboard.html` (macOS/Linux/Windows tabs); CA guidance in `SettingsPanel.tsx` with platform detection; `/ca.crt` download endpoint |
| WEB-04 | 04-02, 04-03 | Web dashboard lists all web-served sessions with password authentication | SATISFIED | `dashboardAuth` middleware; `POST /login` with bcrypt; `GET /api/sessions` lists web-enabled sessions |
| WEB-05 | 04-02, 04-03 | User can generate shareable token links for specific sessions | SATISFIED | `POST /api/sessions/{id}/token`; `TokenStore` with crypto/rand; `GenerateSessionToken` in app.go; "Copy Token Link" in App.tsx |
| WEB-06 | 04-03 | Remote browser connects to session via xterm.js over WebSocket (full interaction, not read-only) | SATISFIED | `terminal.html` with xterm.js 6 CDN; WSS relay in `handleWSSRelay`; MsgInput write pump sends to PTY; `TestWebServerWSS` passes |
| NET-01 | 04-01, 04-04 | User can bind web server to a specific network interface | SATISFIED | `Config.BindIP` field; `StartWebServer(bindIP, port)` in app.go; interface dropdown in `SettingsPanel.tsx` |
| NET-02 | 04-01 | App auto-detects Tailscale interface via CGNAT range (100.64.0.0/10) | SATISFIED | `IsTailscaleIP` with `tailscaleCIDR`; `SettingsPanel.tsx` auto-selects Tailscale interface |
| NET-03 | 04-01, 04-04 | User can select other VPN interfaces (WireGuard, etc.) from a dropdown | SATISFIED | `ListInterfaces` enumerates all IPv4 non-loopback interfaces; dropdown in `SettingsPanel.tsx` |

All 9 requirements satisfied. No orphaned requirements detected.

---

### Anti-Patterns Found

No blocker or warning anti-patterns found. The three HTML/JSX `placeholder` attribute occurrences are UI hint text, not code stubs. No `TODO`/`FIXME`/empty implementations detected in any phase file.

---

### Human Verification Required

#### 1. End-to-end web serving flow

**Test:** Launch `wails dev`, set password in Settings, start web server, create a terminal session, toggle "Web On" for it, open the displayed HTTPS URL in a browser. Log in with the password, click the session link, interact with the xterm.js terminal.
**Expected:** Login succeeds, dashboard shows the session, terminal renders PTY output, keystrokes typed in browser reach the CLI process and produce output.
**Why human:** Requires a live running app with an actual PTY session, a browser, and confirmation of bidirectional keyboard I/O. The `TestWebServerWSS` test validates the Go relay machinery, but cannot verify the end-to-end browser experience.

#### 2. Token link bypass

**Test:** Generate a token link using the "Copy Token Link" button in the desktop app. Open the link in an incognito browser window (no session cookie).
**Expected:** Page loads the xterm.js terminal and connects to the session without prompting for a password.
**Why human:** Token-based access bypass cannot be confirmed without a live app, live session, and visual confirmation that no login prompt appears.

#### 3. CA cert installation and trust

**Test:** Download `/ca.crt` from the running web server, install it into the OS trust store using the instructions shown in the dashboard or SettingsPanel, restart the browser and reload the HTTPS page.
**Expected:** Browser no longer shows a TLS security warning. The page loads with the padlock icon indicating trusted TLS.
**Why human:** OS trust store behavior and browser TLS UI cannot be verified programmatically. Platform-specific commands (security, certutil, update-ca-certificates) require real OS execution and visual browser confirmation.

---

### Test Results Summary

All automated test suites pass:

```
go test ./internal/webserver/... -timeout 60s   # 29 tests — PASS
go test -run "TestWeb|TestSetWeb|TestGetNetwork|TestToggleWeb|TestStartWeb|TestIsWeb" -timeout 30s   # 5 tests — PASS
go build ./...   # OK (no compile errors)
```

No TypeScript compilation was run due to tooling permissions, but the binding files were verified by direct code review — all 10 method declarations are present in `App.d.ts` and `App.js` with correct signatures.

---

*Verified: 2026-03-18T19:00:00Z*
*Verifier: Claude (gsd-verifier)*
