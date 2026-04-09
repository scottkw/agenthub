---
phase: 60-local-network-fallback
verified: 2026-04-09T21:11:21Z
status: human_needed
score: 17/17 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Open the app without Tailscale running. Check that the amber nudge banner appears above the sidebar+content row."
    expected: "Banner shows 'Local network mode active' text, amber left border, and 'Install Tailscale' button. Terminal area is not shrunk or displaced."
    why_human: "Visual layout requires physical inspection in the running Wails app. Cannot verify amber border rendering or flex layout correctness programmatically."
  - test: "Navigate to Settings > Web Server tab while in local mode. Verify password field appears."
    expected: "A 'LAN Access Password' field shows the generated password (~22 chars base64url). Clicking copies it to clipboard with 'Copied!' feedback."
    why_human: "Clipboard interaction and visual password display require manual interaction in a running app."
  - test: "From a browser on the same LAN, navigate to https://<LAN-IP>:7443. Verify credential prompt appears and password is accepted."
    expected: "Browser shows HTTP Basic Auth credential prompt. Entering any username and the generated password grants access. Entering wrong password returns 401."
    why_human: "Requires a real LAN connection, a second device or browser, and a self-signed TLS cert that must be bypassed. Cannot simulate in tests."
  - test: "Start the app with Tailscale connected and fully configured. Verify the nudge banner does NOT appear."
    expected: "No LocalNetworkBanner in the DOM. HealthModal shows normally if Tailscale needs attention."
    why_human: "Requires real Tailscale connectivity state to test the tailscale-mode branch."
---

# Phase 60: Local Network Fallback Verification Report

**Phase Goal:** Users without Tailscale can serve sessions over the local network using self-signed TLS and a randomly generated password, with the password visible in the Settings tab and a persistent nudge banner encouraging Tailscale installation
**Verified:** 2026-04-09T21:11:21Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
|---|-------------------|--------|----------|
| 1 | Web server starts and serves sessions over HTTPS on the local network IP when Tailscale is not available | VERIFIED | `startLocal()` in server.go calls `GenerateSelfSignedCert(ws.config.BindIP)` + binds TLS listener. `runDaemonCore` calls `GetLANIP()` + `AutoStartWebServer(lanIP, 7443, "", "local", localPassword)` in the else branch. `TestLocalModeStart` passes. |
| 2 | Browser connection in local mode prompts for credentials and accepts the generated password (HTTP Basic Auth) | VERIFIED | `basicAuthMiddleware` wraps the mux in `startLocal()`. Returns 401 + `WWW-Authenticate: Basic realm="AgentHub"` on no/wrong creds. `TestBasicAuthMiddleware_Unauthorized` and `TestBasicAuthMiddleware_Authorized` pass. |
| 3 | User can see the generated password in the Settings tab under a clearly labeled field | VERIFIED | `SettingsPanel.tsx` renders `LAN Access Password` label, password field, and `(click to copy)` hint when `webServerMode === 'local' && isServerRunning`. `GetLocalNetworkPassword()` Wails binding is imported and called. |
| 4 | A nudge banner appears in the app on each launch when running in local network mode, recommending Tailscale installation | VERIFIED | `LocalNetworkBanner` rendered in `App.tsx` when `webServerMode === 'local'`. `GetWebServerMode()` polled during `init()` and `retryInit()`. Banner contains "Install Tailscale" CTA with `tailscale.com/download`. |
| 5 | Nudge banner does not shrink or displace the terminal area (renders outside the terminal flex container) | VERIFIED (code) | `LocalNetworkBanner` is rendered before `<div className="app__row">`, which wraps Sidebar + `app__content`. The `.app` div is `flex-direction: column` so the banner stacks above the row. `app__row` has `flex: 1` to consume remaining height. Rendering confirmed in DOM position; actual visual layout needs human check. |

**Score:** 5/5 roadmap success criteria satisfied in code

### Observable Truths (Plan Must-Haves)

#### Plan 01 — Webserver Infrastructure

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Self-signed TLS cert uses P256 curve and includes IP SAN for the bind address | VERIFIED | `selfcert.go` line 30: `elliptic.P256()` for CA; line 56: `elliptic.P256()` for leaf. Line 69: `IPAddresses: []net.IP{parsedIP}`. `TestGenerateSelfSignedCert` verifies both P256 and IP SAN. |
| 2 | Basic Auth middleware returns 401 with WWW-Authenticate header when no credentials provided | VERIFIED | `auth.go` line 21: `w.Header().Set("WWW-Authenticate", \`Basic realm="AgentHub"\`)`. Tests `TestBasicAuthMiddleware_Unauthorized` and `TestBasicAuthMiddleware_WrongPassword` pass. |
| 3 | Basic Auth middleware returns 200 when correct password provided | VERIFIED | `auth.go` line 24: `next.ServeHTTP(w, r)` called when `pass == password`. `TestBasicAuthMiddleware_Authorized` passes. |
| 4 | GetLANIP returns a non-loopback private IPv4 address and excludes Tailscale CGNAT range | VERIFIED | `localip.go` — `isTailscaleIP` checks `ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127`. `firstPrivateIPv4` skips loopback, link-local, Tailscale IPs. `TestGetLANIP` and `TestGetLANIP_ExcludesTailscale` pass. |
| 5 | WebServer starts in local mode with self-signed cert and auth middleware | VERIFIED | `server.go` `startLocal()` calls `GenerateSelfSignedCert` then wraps mux with `basicAuthMiddleware`. `TestLocalModeStart` confirms 401 without auth and 200 with correct auth. |
| 6 | BaseURL returns IP-based URL in local mode, FQDN-based URL in tailscale mode | VERIFIED | `server.go` `BaseURL()` checks `ws.config.Mode == "local"` — returns `https://<BindIP>:<port>` vs `https://<FQDN>:<port>`. `TestBaseURL_LocalMode` and `TestBaseURL_TailscaleMode` pass. |

#### Plan 02 — Daemon Integration

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 7 | Password is generated once per daemon lifetime using crypto/rand | VERIFIED | `process.go` `generateLocalPassword()` uses `rand.Read(b)` with 16 bytes, encoded as `base64.RawURLEncoding`. Called once in `runDaemonCore` before any server start. |
| 8 | AutoStartWebServer accepts mode and password parameters | VERIFIED | `api.go` signature: `func (a *API) AutoStartWebServer(ip string, port int, fqdn, mode, password string) error`. Passes Mode and Password to `webserver.Config`. |
| 9 | Daemon falls back to local mode when Tailscale is not available | VERIFIED | `process.go` lines 66-84: `if h.Connected && h.HasCerts && h.IP != ""` → tailscale mode; else → `GetLANIP()` + local mode with `localPassword`. |
| 10 | GET /webserver/status returns Mode field | VERIFIED | `api.go` `handleWebServerStatus`: `writeJSON(w, http.StatusOK, WebServerStatusResponse{Running: true, URL: ws.BaseURL(), Addr: ws.Addr(), Mode: ws.Mode()})`. `types.go` `WebServerStatusResponse` has `Mode string`. |
| 11 | GET /webserver/local-password returns the generated password in local mode | VERIFIED | `api.go` route: `"GET /webserver/local-password"` → `handleGetLocalPassword` returns `{"password": pwd}`. `TestGetLocalPassword` passes. |
| 12 | Wails binding GetLocalNetworkPassword exposes password to frontend | VERIFIED | `app.go` `GetLocalNetworkPassword() string` delegates to `a.client.GetLocalNetworkPassword()`. Wails TS stubs: `App.d.ts` declares `GetLocalNetworkPassword(): Promise<string>`, `App.js` exports the Call. |
| 13 | GetWebServerMode Wails binding exposes mode to frontend | VERIFIED | `app.go` `GetWebServerMode() string` delegates to `a.client.GetWebServerStatus().Mode`. TS stubs present in `App.d.ts` and `App.js`. |

#### Plan 03 — Frontend

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 14 | Nudge banner appears when webServerMode is 'local' and disappears in tailscale mode | VERIFIED | `App.tsx` line 470-475: `{webServerMode === 'local' && <LocalNetworkBanner visible={true} onOpenURL={BrowserOpenURL} />}`. Component returns null when `visible=false`. |
| 15 | Banner renders above the sidebar+content row, never inside terminal-container | VERIFIED | `App.tsx` banner is rendered before `<div className="app__row">` which contains Sidebar + `app__content`. `.app` is `flex-direction: column`, `.app__row` is `flex: 1`. Banner is structurally outside terminal area. |
| 16 | Banner has warning amber left border, Install Tailscale CTA button | VERIFIED | `style.css` line 1642: `border-left: 3px solid #f59e0b`. `LocalNetworkBanner.tsx` has `<button>Install Tailscale</button>` that calls `onOpenURL('https://tailscale.com/download')`. |
| 17 | User can see the LAN access password in Settings > Web Server when in local mode | VERIFIED | `SettingsPanel.tsx` lines 321-340: conditional block `{webServerMode === 'local' && isServerRunning && ...}` renders `LAN Access Password` label + password field calling `GetLocalNetworkPassword()`. |

**Score:** 17/17 must-haves verified

### Required Artifacts

| Artifact | Status | Evidence |
|----------|--------|----------|
| `internal/webserver/selfcert.go` | VERIFIED | Exports `GenerateSelfSignedCert(ip string) (*tls.Config, error)`, uses P256, IP SAN, 365-day expiry |
| `internal/webserver/auth.go` | VERIFIED | Exports `BasicAuthMiddleware` and unexported `basicAuthMiddleware` alias. `WWW-Authenticate` header present. |
| `internal/webserver/localip.go` | VERIFIED | Exports `GetLANIP()`, `IsTailscaleIP()`. CGNAT check: `ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127` |
| `internal/webserver/server.go` | VERIFIED | `Config.Mode`, `Config.Password`, `Mode()` accessor, `Start()` dispatches to `startLocal()`/`startTailscale()`, mode-aware `BaseURL()` |
| `internal/daemon/types.go` | VERIFIED | `WebServerStartRequest` has `Mode` and `Password`. `WebServerStatusResponse` has `Mode`. |
| `internal/daemon/api.go` | VERIFIED | `localPassword` field, `SetLocalPassword()`, `handleGetLocalPassword`, `GET /webserver/local-password` route, `AutoStartWebServer` 5-arg signature |
| `internal/daemon/client.go` | VERIFIED | `StartWebServer` 5-arg signature, `GetLocalNetworkPassword()` method |
| `internal/daemon/process.go` | VERIFIED | `generateLocalPassword()` with crypto/rand, `api.SetLocalPassword()`, `webserver.GetLANIP()`, mode-aware `AutoStartWebServer` call |
| `app.go` | VERIFIED | `GetLocalNetworkPassword() string`, `GetWebServerMode() string`, updated `StartWebServer` with local mode fallback |
| `frontend/src/components/LocalNetworkBanner.tsx` | VERIFIED | Exports `LocalNetworkBanner`, `role="status"`, amber border, CTA button with tailscale.com/download |
| `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` | VERIFIED | 5 tests: visible/not-visible, onOpenURL call, role="status", warning icon. All pass. |
| `frontend/src/style.css` | VERIFIED | `.local-network-banner`, `border-left: 3px solid #f59e0b`, `.app__row`, `.settings-web-server__password-field` |

### Key Link Verification

| From | To | Via | Status | Evidence |
|------|----|-----|--------|----------|
| `server.go` | `selfcert.go` | `startLocal()` calls `GenerateSelfSignedCert` | WIRED | Line 171: `tlsCfg, err := GenerateSelfSignedCert(ws.config.BindIP)` |
| `server.go` | `auth.go` | `startLocal()` wraps mux with `basicAuthMiddleware` | WIRED | Line 201: `handler := basicAuthMiddleware(ws.config.Password)(ws.mux)` |
| `server.go` | `localip.go` | consumer uses `GetLANIP` | WIRED | `process.go` line 74: `webserver.GetLANIP()` |
| `process.go` | `api.go` | `runDaemonCore` calls `api.SetLocalPassword` and `api.AutoStartWebServer` | WIRED | Lines 59, 67, 78 in process.go |
| `process.go` | `localip.go` | `runDaemonCore` calls `webserver.GetLANIP()` for fallback | WIRED | Line 74 in process.go |
| `app.go` | `client.go` | `GetLocalNetworkPassword` calls `client.GetLocalNetworkPassword` | WIRED | `app.go` line 347: `a.client.GetLocalNetworkPassword()` |
| `App.tsx` | `LocalNetworkBanner.tsx` | conditional render when `webServerMode === 'local'` | WIRED | App.tsx line 470: `{webServerMode === 'local' && <LocalNetworkBanner ...>}` |
| `App.tsx` | `app.go` | `GetWebServerMode` Wails binding polled for mode state | WIRED | App.tsx lines 114, 443: `GetWebServerMode().then(mode => setWebServerMode(...))` |
| `SettingsPanel.tsx` | `app.go` | `GetLocalNetworkPassword` Wails binding for password display | WIRED | SettingsPanel.tsx line 84: `GetLocalNetworkPassword().then(setLocalPassword)` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `LocalNetworkBanner.tsx` | `visible` (from `webServerMode`) | `GetWebServerMode()` → `client.GetWebServerStatus()` → `ws.Mode()` → `ws.config.Mode` | Yes — mode set from real daemon state in `startLocal()`/`startTailscale()` | FLOWING |
| `SettingsPanel.tsx` | `localPassword` | `GetLocalNetworkPassword()` → `client.GetLocalNetworkPassword()` → `GET /webserver/local-password` → `a.localPassword` | Yes — set by `generateLocalPassword()` using `crypto/rand` in `runDaemonCore` | FLOWING |
| `HealthModal.tsx` | `webServerRunning` | `IsWebServerRunning()` → `client.GetWebServerStatus()` → `ws != nil` | Yes — reflects real webserver state from daemon | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All webserver tests pass | `go test ./internal/webserver/...` | 30 tests PASS, 0 FAIL | PASS |
| All daemon tests pass | `go test ./internal/daemon/...` | All tests PASS, 0 FAIL | PASS |
| Go build succeeds (Wails bindings compile) | `go build ./...` | EXIT 0 | PASS |
| Frontend LocalNetworkBanner tests pass | `pnpm test` | 5/5 LocalNetworkBanner tests PASS | PASS |
| Pre-existing Sidebar.test.tsx failures | `pnpm test` | 14 failures in Sidebar.test.tsx only (localStorage jsdom issue, pre-existing, out of scope) | INFO |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|-------------|--------|----------|
| NET-01 | 60-01, 60-02 | User can serve sessions over the local network with self-signed TLS when Tailscale is not available | SATISFIED | `selfcert.go` + `startLocal()` + `runDaemonCore` local fallback |
| NET-02 | 60-01, 60-02 | Local network mode generates a random password for all web connections via HTTP Basic Auth | SATISFIED | `generateLocalPassword()` with crypto/rand + `basicAuthMiddleware` wrapping mux |
| NET-03 | 60-01, 60-02 | Web server binds to a local network interface when operating in local mode | SATISFIED | `GetLANIP()` returns private IPv4 (not loopback, not Tailscale CGNAT); `startLocal()` binds to `ws.config.BindIP` |
| NET-04 | 60-03 | User sees a persistent nudge banner on each launch recommending Tailscale installation when in local mode | SATISFIED | `LocalNetworkBanner` rendered in `App.tsx` on `webServerMode === 'local'`; polled during init |
| NET-05 | 60-02, 60-03 | User can view the generated password in the UI (settings/status area) | SATISFIED | `SettingsPanel.tsx` shows "LAN Access Password" field with click-to-copy when in local mode |

All 5 requirements covered. No orphaned requirements found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `frontend/src/components/SettingsPanel.tsx` | 216 | `placeholder={cli.Path \|\| \`Path to ${cli.Name}\`}` | INFO | Pre-existing input placeholder for CLI path overrides, not related to Phase 60. Not a stub — this is valid HTML placeholder text for an input field. |

No blockers found. No Phase 60 stubs.

### Human Verification Required

#### 1. Banner Visual Layout

**Test:** Launch the app without Tailscale. Navigate to any tab (Welcome, Sessions, etc.).
**Expected:** Amber-bordered banner "Local network mode active — your sessions are accessible on your LAN." appears above the sidebar and content area. The terminal/content area below is not squeezed or displaced.
**Why human:** CSS flex layout correctness with `flex-direction: column` on `.app` and `flex: 1` on `.app__row` cannot be verified without rendering in the actual Wails WebView.

#### 2. Settings Password Display and Copy

**Test:** Launch app without Tailscale. Open Settings > Web Server tab.
**Expected:** A "LAN Access Password" label with a ~22-character base64url password is shown. Clicking the field shows "Copied!" and clipboard contains the password.
**Why human:** Clipboard API behavior and visual password rendering require a live app with a running daemon that generated the password.

#### 3. Browser HTTP Basic Auth Prompt

**Test:** From a device on the same LAN, open a browser and navigate to `https://<LAN-IP>:7443` (IP shown in Settings > Web Server).
**Expected:** Browser shows an HTTP Basic Auth credential dialog. Entering any username and the password from the Settings tab grants access. Wrong password returns 401.
**Why human:** Requires real LAN access, a second device or browser, and accepting the self-signed certificate warning. Cannot simulate network-level auth in tests.

#### 4. No Banner in Tailscale Mode

**Test:** Launch the app with Tailscale connected and HTTPS certs enabled.
**Expected:** No LocalNetworkBanner visible. HealthModal behaves normally (shown when Tailscale not fully configured, hidden when running and connected).
**Why human:** Requires real Tailscale connectivity. The tailscale mode branch in `runDaemonCore` and `GetWebServerMode` returning `"tailscale"` needs live verification.

### Gaps Summary

No automated gaps found. All 17 must-haves verified. All 5 requirements satisfied. The 4 human verification items are standard UAT for networking, visual layout, and clipboard behavior — all require a running Wails app with actual network conditions.

---

_Verified: 2026-04-09T21:11:21Z_
_Verifier: Claude (gsd-verifier)_
