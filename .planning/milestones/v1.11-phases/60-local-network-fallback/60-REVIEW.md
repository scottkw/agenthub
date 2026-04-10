---
phase: 60-local-network-fallback
reviewed: 2026-04-09T00:00:00Z
depth: standard
files_reviewed: 23
files_reviewed_list:
  - app.go
  - cmd_cli.go
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/LocalNetworkBanner.test.tsx
  - frontend/src/components/HealthModal.tsx
  - frontend/src/components/LocalNetworkBanner.tsx
  - frontend/src/components/SettingsPanel.tsx
  - frontend/src/style.css
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - internal/daemon/api_test.go
  - internal/daemon/api.go
  - internal/daemon/client.go
  - internal/daemon/process.go
  - internal/daemon/types.go
  - internal/webserver/auth_test.go
  - internal/webserver/auth.go
  - internal/webserver/localip_test.go
  - internal/webserver/localip.go
  - internal/webserver/selfcert_test.go
  - internal/webserver/selfcert.go
  - internal/webserver/server_test.go
  - internal/webserver/server.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 60: Code Review Report

**Reviewed:** 2026-04-09
**Depth:** standard
**Files Reviewed:** 23
**Status:** issues_found

## Summary

This phase adds local-network fallback mode for the web server: when Tailscale is unavailable the daemon auto-starts the web server on the LAN using a self-signed TLS cert and HTTP Basic Auth. The implementation covers password generation, IP discovery, cert creation, Basic Auth middleware, daemon auto-start logic, a new `GetLocalNetworkPassword` / `GetWebServerMode` RPC pair, and a React banner that nudges users toward Tailscale.

The cryptographic and networking foundations are solid (P256 ECDSA, `crypto/rand`, proper CGNAT exclusion). The most significant issue is a missing password-validation guard in `handleWebServerStart` that allows local mode to start with an empty password, bypassing authentication entirely. There are also a handful of logic/correctness issues in the frontend and daemon auto-start path worth addressing before shipping.

## Critical Issues

### CR-01: Local-mode web server can start with empty password

**File:** `internal/daemon/api.go:316-365`

**Issue:** `handleWebServerStart` does not validate that `req.Password` is non-empty when `req.Mode == "local"`. A caller that omits the password field (or passes an empty string) will produce a `Config.Password == ""`. `BasicAuthMiddleware` will then accept any request whose password field is also empty — a standard browser will send `Authorization: Basic dXNlcjoK` (user with empty password) and gain access.

The `Config` struct documents `Password` as "Must be non-empty when Mode is 'local'"`, but that contract is not enforced in the handler, leaving it open to misconfiguration.

**Fix:**
```go
// in handleWebServerStart, after resolving LAN IP:
if req.Mode == "local" && req.Password == "" {
    http.Error(w, "local mode requires a non-empty password", http.StatusBadRequest)
    return
}
```

The same guard should be added to `AutoStartWebServer` in `api.go:152`:
```go
func (a *API) AutoStartWebServer(ip string, port int, fqdn, mode, password string) error {
    if mode == "local" && password == "" {
        return fmt.Errorf("AutoStartWebServer: local mode requires a non-empty password")
    }
    // ...
}
```

---

## Warnings

### WR-01: `webServerMode` state in App.tsx not updated when web server stops

**File:** `frontend/src/App.tsx:300-306`

**Issue:** `handleSettingsClose` re-checks `IsWebServerRunning()` when the settings panel closes (to detect start/stop), but it does not re-fetch `GetWebServerMode()`. If the user stops the web server, `webServerRunning` becomes `false` and the `LocalNetworkBanner` conditional (`webServerMode === 'local'`) is no longer sufficient to hide it — `webServerMode` still holds `'local'` from the previous start, so the banner stays visible even though no server is running.

**Fix:**
```tsx
const handleSettingsClose = useCallback(async () => {
  setShowSettings(false)
  try {
    const running = await IsWebServerRunning()
    setWebServerRunning(running)
    if (!running) {
      setWebServerMode(null)          // clear stale mode
    } else {
      const mode = await GetWebServerMode()
      setWebServerMode(mode === 'tailscale' || mode === 'local' ? mode : null)
    }
  } catch (_) { /* ignore */ }
}, [])
```

Alternatively, drive the banner from `webServerRunning && webServerMode === 'local'` (which already happens to be the current render condition in JSX line 470, since the banner only renders when `webServerMode === 'local'`) — but the root fix is ensuring `webServerMode` is cleared on stop.

---

### WR-02: `handleWebServerStart` does not stop a previously running server before starting a new one

**File:** `internal/daemon/api.go:316-365`

**Issue:** `handleWebServerStart` does not acquire the write lock or check whether `a.webServer` is already non-nil before creating and starting a new `WebServer`. If two start requests arrive concurrently (or the user calls `StartWebServer` twice from the GUI), a second `WebServer` will start on a new port, the first one will be orphaned (listener left open, no reference to close it), and `a.webServer` will be overwritten with the second instance. The first server's TLS listener leaks.

Compare with `AutoStartWebServer` (line 152) which correctly returns early when `a.webServer != nil`.

**Fix:**
```go
func (a *API) handleWebServerStart(w http.ResponseWriter, r *http.Request) {
    // ... decode req ...

    a.mu.Lock()
    if a.webServer != nil {
        // Stop existing server before replacing it.
        _ = a.webServer.Stop()
        a.webServer = nil
    }
    a.mu.Unlock()

    // ... create and start new ws ...
}
```

---

### WR-03: `pollSessionStatus` goroutine in `app.go` never stops for sessions that reach `StatusRunning`

**File:** `app.go:197-219`

**Issue:** `pollSessionStatus` exits only when status reaches `StatusErrored` or the 60-second deadline expires. A session that remains in `"running"` indefinitely keeps the goroutine alive for a full 60 seconds, polling every 500 ms (120 HTTP round-trips to the daemon). This is not a leak (it exits eventually), but for long-running sessions the polling period fires well past the time the frontend has already received a `"running"` status, which causes unnecessary daemon load and may cause spurious UI re-renders.

**Fix:** Add `StatusRunning` as an exit condition to match the intent of the comment ("Stops when status reaches 'errored' or after 60 seconds"):
```go
switch s {
case string(status.StatusErrored), string(status.StatusRunning):
    return
}
```
Or, more conservatively, reduce the deadline to 10s once a stable non-waiting status is received.

---

### WR-04: `GetLocalNetworkPassword` on the daemon API is unauthenticated

**File:** `internal/daemon/api.go:396-401`

**Issue:** `GET /webserver/local-password` returns the plaintext LAN password over the Unix socket without any check. The daemon API has no authentication at all (it relies on filesystem permissions on the socket), which is acceptable for local IPC. However, any process running as the same user can retrieve the password and use it to access the web server. This is documented-by-omission rather than explicitly acknowledged.

This is worth noting because the daemon socket path is `~/.local/share/agenthub/daemon.sock` (default), which is user-owned, so the threat is limited to same-UID processes. No immediate fix is required, but it should be noted in documentation or a comment near `handleGetLocalPassword` so the threat model is explicit.

**Fix (informational):** Add an inline comment:
```go
// handleGetLocalPassword returns the local-mode password over the Unix socket.
// The socket is owned by the current user (0600) so only same-UID processes
// can reach this endpoint. This is the intended access-control model.
func (a *API) handleGetLocalPassword(w http.ResponseWriter, r *http.Request) {
```

---

## Info

### IN-01: Duplicate init logic between `init()` and `retryInit()` in App.tsx

**File:** `frontend/src/App.tsx:91-143, 419-466`

**Issue:** The `init()` function inside the mount `useEffect` and `retryInit` are near-identical: both call the same `Promise.all`, both call `GetWebServerMode().then(...)`, both restore sessions. Any change to one needs to be mirrored in the other. This is a maintenance risk rather than a current bug.

**Fix:** Extract the shared logic into a helper:
```tsx
async function loadAppState() { /* shared Promise.all block */ }
// then:  void loadAppState() in init() and retryInit()
```

---

### IN-02: `cmd_cli.go` `cmdWebStart` does not support local-mode fallback

**File:** `cmd_cli.go:192-212`

**Issue:** The CLI `web start` subcommand only supports Tailscale mode — it returns an error if Tailscale is not connected. The daemon now auto-starts in local mode as a fallback, but `cmdWebStart` cannot exercise or report the local mode URL. This is a usability gap rather than a bug: the daemon handles local mode automatically, but the CLI user cannot trigger or inspect it via `agenthub web start`.

**Fix:** Consider adding a `--local` flag or updating the command to reflect the current server state via `cmdWebStatus` when Tailscale checks fail.

---

### IN-03: `selfcert.go` leaf cert does not include `SubjectKeyId` / `AuthorityKeyIdentifier`

**File:** `internal/webserver/selfcert.go:66-75`

**Issue:** The leaf cert template omits `SubjectKeyIdentifier` and `AuthorityKeyIdentifier`. Some TLS stacks and security scanners flag their absence, though Go's `crypto/tls` and all major browsers accept the cert without them. Adding them is a small improvement to standards conformance:
```go
leafTmpl := &x509.Certificate{
    // ... existing fields ...
    SubjectKeyId: leafKey.PublicKey.X.Bytes(), // advisory
}
```
This is a low-priority improvement with no current behavioural impact.

---

_Reviewed: 2026-04-09_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
