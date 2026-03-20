---
phase: 15-tailscale-tls-interface-binding
plan: "02"
subsystem: app-layer
tags: [tailscale, tls, ct-disclosure, frontend, wails, go]
dependency_graph:
  requires: [tailscale-tls-backend]
  provides: [tailscale-gated-web-server, ct-disclosure-flow, settings-ui-updated]
  affects: [app.go, frontend/src/components/SettingsPanel.tsx, frontend/src/style.css]
tech_stack:
  added: []
  patterns: [Tailscale health gate before WebServer creation, CT disclosure persistence via sentinel file]
key_files:
  created: []
  modified:
    - app.go
    - app_test.go
    - frontend/src/components/SettingsPanel.tsx
    - frontend/src/style.css
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - .gitignore
decisions:
  - "StartWebServer(port int) gates on Connected, IP != '', HasCerts — Tailscale health provides all bind parameters"
  - "CT disclosure persisted as sentinel file ct_disclosed in configDir(); HasCTDisclosure reads os.Stat"
  - "Wails JS stubs (App.d.ts, App.js) updated manually since they are hand-maintained stubs, not auto-generated"
  - "TestGetSessionQRCode bypasses StartWebServer Tailscale gate via direct WebServer construction with in-memory TLS"
metrics:
  duration: 480s
  completed: "2026-03-20"
  tasks_completed: 2
  files_modified: 7
---

# Phase 15 Plan 02: App-Layer Wiring — Tailscale-Gated WebServer and CT Disclosure Summary

**One-liner:** Wired Tailscale health check into StartWebServer (removing bindIP param, deriving IP+FQDN from daemon), added CT disclosure persistence methods, and rewrote SettingsPanel to show CT banner and remove interface dropdown and CA cert section.

## What Was Built

- **StartWebServer(port int)**: Removed `bindIP string` parameter; derives `BindIP` and `FQDN` from `GetTailscaleStatus()`. Gates on `h.Connected`, `h.IP != ""`, `h.HasCerts` before creating WebServer. Uses `webserver.Config{BindIP: h.IP, FQDN: h.Domain}`.
- **HasCTDisclosure() / AcknowledgeCTDisclosure()**: Wails-bound methods using a sentinel file (`~/.config/agenthub/ct_disclosed`) for persistence across restarts.
- **GetCACertPath() removed**: Deleted from app.go — called `webserver.ExportCACertPath` which no longer exists after Plan 01 deleted `tls.go`.
- **SetWebPassword lazy init fixed**: Removed `ConfigDir: configDir()` from `webserver.Config` literal (field deleted in Plan 01).
- **SettingsPanel.tsx rewritten**: CT disclosure banner replaces interface dropdown; Security tab loses CA Certificate section; `StartWebServer(selectedPort)` replaces `StartWebServer(selectedInterface, selectedPort)`.
- **style.css**: Added `.ct-disclosure`, `.ct-disclosure--acknowledged`, `.ct-disclosure__text`, `.ct-disclosure__btn` classes.
- **.gitignore**: Added `*.ts.net.crt` and `*.ts.net.key`.

## Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Update app.go, gitignore | d703be1 | app.go, .gitignore |
| 2 | Update frontend, fix tests | 565aa4d | SettingsPanel.tsx, style.css, App.d.ts, App.js, app_test.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Wails JS stubs (App.d.ts, App.js) needed manual update**
- **Found during:** Task 2 TypeScript verification
- **Issue:** `App.d.ts` and `App.js` still exported `GetCACertPath`, `GetNetworkInterfaces`, and `StartWebServer(bindIP, port)` — causing tsc errors for new imports
- **Fix:** Updated both stubs to match new Go API: removed `GetCACertPath`, updated `StartWebServer` to `(port: number)`, added `HasCTDisclosure` and `AcknowledgeCTDisclosure`
- **Files modified:** `frontend/src/wailsjs/go/main/App.d.ts`, `frontend/src/wailsjs/go/main/App.js`
- **Commit:** 565aa4d

**2. [Rule 1 - Bug] app_test.go called StartWebServer with old 2-arg signature**
- **Found during:** Task 2 go test run
- **Issue:** `TestStartWebServerErrorsWhenPasswordNotSet` called `app.StartWebServer("127.0.0.1", 0)` — compile error with new signature
- **Fix:** Updated to `app.StartWebServer(0)` — password check is still the first gate so test logic is unchanged
- **Files modified:** `app_test.go`
- **Commit:** 565aa4d

**3. [Rule 1 - Bug] TestGetSessionQRCode called StartWebServer which now requires Tailscale**
- **Found during:** Task 2 go test run
- **Issue:** `TestGetSessionQRCode` called `app.StartWebServer("127.0.0.1", 0)` — now fails because Tailscale is not available in CI/test environments
- **Fix:** Rewrote test to directly construct a `webserver.WebServer` with in-memory self-signed TLS config (using `webserver.Config{TLSConfig: ...}`) and assign to `app.webServer`, bypassing the Tailscale health gate. Added `selfSignedTLSForAppTest` helper in `app_test.go`.
- **Files modified:** `app_test.go`
- **Commit:** 565aa4d

## Success Criteria Verification

- [x] `go build ./...` passes
- [x] `go test ./... -count=1` passes (all packages green)
- [x] Frontend TypeScript compiles without errors (`npx tsc --noEmit` exits 0)
- [x] `StartWebServer` in app.go accepts `(port int)` only
- [x] `StartWebServer` gates on `h.Connected`, `h.IP != ""`, `h.HasCerts`
- [x] `HasCTDisclosure` and `AcknowledgeCTDisclosure` exist in app.go
- [x] `GetCACertPath` removed from app.go
- [x] No `ConfigDir:` in any `webserver.Config` literal
- [x] SettingsPanel has CT disclosure banner, no interface dropdown, no CA cert section
- [x] `.gitignore` excludes `*.ts.net.crt` and `*.ts.net.key`

## Self-Check: PASSED
