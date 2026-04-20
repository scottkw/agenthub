---
phase: 87
plan: 04
type: execute
wave: 3
depends_on: [1, 2, 3]
files_modified:
  - internal/daemon/api.go
  - internal/daemon/types.go
  - internal/daemon/client.go
  - internal/daemon/engine.go
  - internal/daemon/api_test.go
  - app.go
autonomous: true
requirements:
  - SEC-01
  - SEC-02
  - SEC-03
  - SEC-04
  - SEC-05
tags:
  - security
  - daemon
  - ipc
  - wails

must_haves:
  truths:
    - "Creating a new session while the web server is running does NOT auto-enable web serving (SEC-01 / D-06)"
    - "Toggling web-serving ON for a session issues two capabilities (read + read,write) and returns URLs with ?cap=<token>"
    - "Toggling web-serving OFF clears the session's grant list permanently (D-15)"
    - "Session onExit calls both DisableSession AND ClearGrants (Pitfall 1)"
    - "Daemon startup loads or generates capability.key and calls ws.SetSigningKey before accepting any HTTP request"
    - "RegenerateSigningKey replaces capability.key on disk and updates WS.signingKey atomically"
    - "Wails bindings IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey, GetCapabilityQRCode are exported to the frontend"
  artifacts:
    - path: internal/daemon/engine.go
      provides: "Loads/generates signing key via capability.FileKeyStore at startup; passes key to WebServer via SetSigningKey"
      contains: "capability.LoadOrGenerate\\|NewFileKeyStore"
    - path: internal/daemon/api.go
      provides: "Auto-enable removed from handleCreateSession; handleWebServe enable path issues caps; onExit clears grants; 3 new IPC handlers"
      contains: "IssueCapabilities\\|ExchangeJoinCode\\|RegenerateSigningKey"
    - path: internal/daemon/types.go
      provides: "IssueCapabilitiesResponse, ExchangeJoinCodeRequest/Response, RegenerateSigningKeyResponse"
      contains: "IssueCapabilitiesResponse"
    - path: internal/daemon/client.go
      provides: "Typed IPC client methods: IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey"
    - path: app.go
      provides: "Wails bindings: IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey, GetCapabilityQRCode"
  key_links:
    - from: internal/daemon/engine.go
      to: internal/capability/keystore.go
      via: "LoadOrGenerate(NewFileKeyStore(configDir))"
      pattern: "capability\\.(LoadOrGenerate|NewFileKeyStore)"
    - from: internal/daemon/api.go handleWebServe enable path
      to: "ws.AddGrant + capability.Sign"
      via: "issue two caps per toggle-on"
      pattern: "AddGrant\\(.*claims\\.GrantID"
    - from: internal/daemon/api.go onExit
      to: "ws.ClearGrants"
      via: "Pitfall 1 fix — clear grants alongside DisableSession"
      pattern: "ClearGrants\\("
---

<objective>
Wire the daemon IPC and engine to the capability subsystem. Remove the SEC-01 auto-enable bug (api.go:287-293). Extend `handleWebServe` enable path to issue two capabilities (read + read,write) and return shareable URLs; extend the disable path to clear the grant list. Fix the RESEARCH Pitfall 1 grant leak by adding `ClearGrants` into the onExit callback. Load or generate the signing key at daemon startup. Add three new IPC handlers (`IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`) and their Wails bindings. Add a `GetCapabilityQRCode` binding that encodes the join-code exchange URL (D-09) rather than the capability token directly.

Purpose: Deliver SEC-01 (no auto-expose), complete the grant-issuance integration (Plan 03 provides the state, this plan populates it), and surface the entire flow to the frontend via Wails.

Output: Daemon integration complete. `go test ./internal/daemon/ -run TestHandleCreateSession_NoAutoEnable` green; `app.go` Wails bindings regenerated. SEC-01 behavioral test passes.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md
@.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md
@internal/daemon/api.go
@internal/daemon/types.go
@internal/daemon/client.go
@internal/daemon/engine.go
@app.go
@internal/capability/capability.go
@internal/capability/keystore.go
@internal/capability/joincode.go

<interfaces>
Consumes from Plan 02:
```go
capability.Claims, Sign, Verify
capability.FileKeyStore, NewFileKeyStore, LoadOrGenerate, GenerateKey
capability.JoinCodeManager, NewJoinCodeManager
capability.ErrCodeNotFound, ErrCodeExpired
```

Consumes from Plan 03:
```go
ws.AddGrant(sessionID, grantID string)
ws.ClearGrants(sessionID string)
ws.SetSigningKey(key []byte)
ws.EnableSession(sessionID string)  // existing
ws.DisableSession(sessionID string) // existing
```

New IPC endpoints registered in api.go:
```
POST /sessions/{id}/capabilities     -> IssueCapabilitiesResponse
POST /join/exchange                  -> ExchangeJoinCodeResponse
POST /capability/regenerate-key      -> 200 OK
```

New daemon types (types.go additions):
```go
type IssueCapabilitiesResponse struct {
    ReadURL   string `json:"readUrl"`
    WriteURL  string `json:"writeUrl"`
    ReadCode  string `json:"readCode"`
    WriteCode string `json:"writeCode"`
}
type ExchangeJoinCodeRequest struct { Code string `json:"code"` }
type ExchangeJoinCodeResponse struct { URL string `json:"url"` }
```

New client methods:
```go
func (c *DaemonClient) IssueCapabilities(sessionID string) (IssueCapabilitiesResponse, error)
func (c *DaemonClient) ExchangeJoinCode(code string) (string, error)
func (c *DaemonClient) RegenerateSigningKey() error
```

New Wails bindings on *App:
```go
func (a *App) IssueCapabilities(sessionID string) (daemon.IssueCapabilitiesResponse, error)
func (a *App) ExchangeJoinCode(code string) (string, error)
func (a *App) RegenerateSigningKey() error
func (a *App) GetCapabilityQRCode(joinURL string) (string, error) // base64-encoded PNG per app.go:582 pattern
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <id>87-04-01</id>
  <name>Task 1: Remove auto-enable, add signing key startup, wire grant issuance/clearance on toggle, fix onExit grant cleanup</name>
  <files>internal/daemon/engine.go, internal/daemon/api.go, internal/daemon/api_test.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/internal/daemon/api.go (full, especially lines 257-295 handleCreateSession and 515-525 handleToggleWebServing / handleWebServe)
    - /Users/ken/dev/agenthub/internal/daemon/engine.go (full, especially lines 136-154 NewSessionEngine and startup sequence)
    - /Users/ken/dev/agenthub/internal/daemon/api_test.go (full — existing test patterns)
    - /Users/ken/dev/agenthub/internal/daemon/engine_exit_test.go (full — onExit test pattern)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 511-526 Pattern 7 auto-enable removal; lines 569-575 Pitfall 1 grant cleanup on exit; lines 626-655 token flow; Open Question 2 grant persistence)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 472-544 api.go edits; lines 616-634 engine.go startup wiring)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md (D-14 grant persistence, D-15 toggle-off clears, D-06 toggle is grant gesture)
  </read_first>
  <behavior>
    GREEN test (add to api_test.go):
    - TestHandleCreateSession_NoAutoEnable (SEC-01): create a session while ws is running; assert that `ws.IsSessionEnabled(sessionID)` returns false immediately after create
    - TestHandleWebServe_ToggleOnIssuesCaps: toggle web-serving ON; assert response body contains ReadURL and WriteURL; assert `ws.isGrantActive(sid, claims.GrantID)` returns true for each cap
    - TestHandleWebServe_ToggleOffClearsGrants: toggle ON then OFF; assert `ws.isGrantActive(sid, anyPriorGrantID)` returns false
    - TestOnExit_ClearsGrants (RESEARCH Pitfall 1): simulate session exit; after the 10-second grace (use a 0-second grace for test if possible via injected delay, or use existing test helper pattern), assert grants map is empty for that sid
    - TestStartup_LoadsOrGeneratesSigningKey: start engine with empty config dir; assert capability.key file now exists and WS.signingKey is non-nil
  </behavior>
  <action>
    1. In `internal/daemon/engine.go`, near the startup sequence (NewSessionEngine or NewAPI — pick the location where `a.webServer` is constructed):
       - Import `github.com/scottkw/agenthub/internal/capability`.
       - After config dir is known, call: `key, err := capability.LoadOrGenerate(capability.NewFileKeyStore(configDir))` — return error if it fails.
       - Store `signingKey []byte` on the engine or API struct alongside existing fields.
       - Add a `JoinCodeManager` to the engine/API: `a.joinCodes = capability.NewJoinCodeManager(5 * time.Minute)` (D-11).
       - Where WebServer is constructed / `AutoStartWebServer` is called, call `ws.SetSigningKey(key)` BEFORE `ws.Serve()` / listener.Accept — Pitfall 3. Place as a new statement immediately after WebServer construction, before ListenAndServe.

    2. In `internal/daemon/api.go`, in `handleCreateSession` (lines 257-295 per PATTERNS), REMOVE the auto-enable block:
       ```go
       // DELETE lines approximately 287-293:
       // a.mu.RLock()
       // ws := a.webServer
       // a.mu.RUnlock()
       // if ws != nil {
       //     ws.EnableSession(id)
       // }
       ```
       Leave the rest of handleCreateSession intact (session creation, registration, onExit setup).

    3. In `handleCreateSession`, locate the `onExit` callback definition (approximately lines 268-277 per PATTERNS). Currently it calls `ws.DisableSession(sessionID)` inside the 10-second grace. Add a sibling `ws.ClearGrants(sessionID)` call after DisableSession. Fixes RESEARCH Pitfall 1.
       ```go
       onExit := func(sessionID string, exitCode int) {
           time.AfterFunc(10*time.Second, func() {
               a.mu.RLock()
               ws := a.webServer
               a.mu.RUnlock()
               if ws != nil {
                   ws.DisableSession(sessionID)
                   ws.ClearGrants(sessionID)  // D-15 also applies on natural exit (Pitfall 1)
               }
           })
       }
       ```

    4. In `internal/daemon/api.go` `handleWebServe` (lines 501-524 per PATTERNS). Split the behavior for Enable vs Disable:
       ```go
       // After decoding req.Enabled and resolving ws:
       if req.Enabled {
           ws.EnableSession(id)
           readURL, writeURL, readCode, writeCode, err := a.issueCapabilitiesForSession(id)
           if err != nil {
               http.Error(w, err.Error(), http.StatusInternalServerError)
               return
           }
           writeJSON(w, http.StatusOK, IssueCapabilitiesResponse{
               ReadURL: readURL, WriteURL: writeURL,
               ReadCode: readCode, WriteCode: writeCode,
           })
           return
       }
       ws.DisableSession(id)
       ws.ClearGrants(id)  // D-15 permanent grant list clear
       w.WriteHeader(http.StatusNoContent)
       ```

    5. Add private helper `issueCapabilitiesForSession` on *API (mirrors RESEARCH Code Example at lines 626-655):
       ```go
       func (a *API) issueCapabilitiesForSession(sessionID string) (readURL, writeURL, readCode, writeCode string, err error) {
           key := a.signingKey // set at startup
           // Generate two 128-bit grant IDs.
           var rgid, wgid [16]byte
           if _, err := rand.Read(rgid[:]); err != nil { return "", "", "", "", err }
           if _, err := rand.Read(wgid[:]); err != nil { return "", "", "", "", err }

           now := time.Now().Unix()
           rClaims := capability.Claims{SID: sessionID, Perms: "read",       IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
           wClaims := capability.Claims{SID: sessionID, Perms: "read,write", IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}

           rTok, err := capability.Sign(rClaims, key); if err != nil { return "", "", "", "", err }
           wTok, err := capability.Sign(wClaims, key); if err != nil { return "", "", "", "", err }

           ws := a.webServer
           ws.AddGrant(sessionID, rClaims.GrantID)
           ws.AddGrant(sessionID, wClaims.GrantID)

           base := ws.BaseURL()
           readURL  = base + "/sessions/" + sessionID + "?cap=" + rTok
           writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok

           // Issue join codes for each (D-09).
           readCode, err  = a.joinCodes.Issue(rTok); if err != nil { return "", "", "", "", err }
           writeCode, err = a.joinCodes.Issue(wTok); if err != nil { return "", "", "", "", err }
           return
       }
       ```
       Imports needed: `crypto/rand`, `encoding/hex`, `time`, `github.com/scottkw/agenthub/internal/capability`.

    6. Add the 5 tests listed under `<behavior>` to `internal/daemon/api_test.go`. Use existing test helpers (`startTestAPI`, etc. — read api_test.go for the idiom). For TestOnExit_ClearsGrants, simulate exit through the existing engine_exit_test.go pattern (injecting a faster grace period if available, or polling for up to 11 seconds).

    7. Run: `go test ./internal/daemon/ -count=1 -v`. All new tests + all existing tests PASS.

    Anti-patterns:
    - DO NOT keep the auto-enable block "just in case" — delete it (SEC-01 requires removal)
    - DO NOT forget ClearGrants in onExit (Pitfall 1)
    - DO NOT call ws.SetSigningKey AFTER the server starts accepting connections (Pitfall 3)
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon/ -count=1 -v -run 'TestHandleCreateSession_NoAutoEnable|TestHandleWebServe_Toggle|TestOnExit_ClearsGrants|TestStartup_LoadsOrGeneratesSigningKey' 2>&1 | tee /tmp/daemon-cap.log ; ! grep -q FAIL /tmp/daemon-cap.log && ! grep -qE 'EnableSession\(id\)$' internal/daemon/api.go && grep -q 'ClearGrants' internal/daemon/api.go && grep -q 'capability.LoadOrGenerate\|capability.NewFileKeyStore' internal/daemon/engine.go && grep -q 'SetSigningKey' internal/daemon/engine.go && go test ./internal/daemon/ -count=1 2>&1 | tee /tmp/daemon-all.log ; ! grep -q FAIL /tmp/daemon-all.log</automated>
  </verify>
  <acceptance_criteria>
    - `grep -q "capability.LoadOrGenerate" internal/daemon/engine.go` OR `grep -q "capability.NewFileKeyStore" internal/daemon/engine.go` succeeds
    - `grep -q "SetSigningKey" internal/daemon/engine.go` succeeds
    - `grep -qE "ws.EnableSession\(id\)\s*$" internal/daemon/api.go` fails (auto-enable block removed — no standalone EnableSession call in handleCreateSession)
    - `grep -c "ClearGrants" internal/daemon/api.go` returns 2 or more (one in onExit, one in handleWebServe disable path)
    - `grep -q "issueCapabilitiesForSession" internal/daemon/api.go` succeeds
    - `grep -q "AddGrant" internal/daemon/api.go` succeeds
    - `grep -q '"read,write"' internal/daemon/api.go` succeeds (D-07 two caps)
    - TestHandleCreateSession_NoAutoEnable passes
    - TestHandleWebServe_ToggleOnIssuesCaps passes
    - TestHandleWebServe_ToggleOffClearsGrants passes
    - TestOnExit_ClearsGrants passes
    - TestStartup_LoadsOrGeneratesSigningKey passes
    - Full daemon suite `go test ./internal/daemon/ -count=1` passes (no regression)
  </acceptance_criteria>
  <done>SEC-01 closed: no auto-enable. Toggle-on issues two caps and registers grants. Toggle-off clears grants. onExit clears grants. Daemon startup loads or generates capability.key and hands key to WebServer before serving.</done>
</task>

<task type="auto" tdd="true">
  <id>87-04-02</id>
  <name>Task 2: Add IssueCapabilities / ExchangeJoinCode / RegenerateSigningKey IPC handlers, types, client methods, and Wails bindings</name>
  <files>internal/daemon/types.go, internal/daemon/api.go, internal/daemon/client.go, app.go</files>
  <read_first>
    - /Users/ken/dev/agenthub/internal/daemon/types.go (full — struct-per-endpoint pattern at lines 62-88)
    - /Users/ken/dev/agenthub/internal/daemon/client.go (full — three-line method template at lines 204-207)
    - /Users/ken/dev/agenthub/internal/daemon/api.go (full — handler routing at lines 43-70, handler template at lines 501-524)
    - /Users/ken/dev/agenthub/app.go (lines 469-613 — Wails binding shapes: ToggleWebServing, GetSessionQRCode, GetWebServerQRCode)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-PATTERNS.md (lines 548-612 types.go + client.go templates; lines 636-669 Wails binding shell)
    - /Users/ken/dev/agenthub/.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-RESEARCH.md (lines 658-694 Join Code Exchange handler)
  </read_first>
  <behavior>
    - New types.go structs: IssueCapabilitiesResponse (ReadURL, WriteURL, ReadCode, WriteCode), ExchangeJoinCodeRequest (Code), ExchangeJoinCodeResponse (URL). JSON tags: camelCase per existing convention.
    - New api.go handlers:
      - POST /sessions/{id}/capabilities -> calls issueCapabilitiesForSession(id), writeJSON IssueCapabilitiesResponse
      - POST /join/exchange -> decodes ExchangeJoinCodeRequest, calls a.joinCodes.Exchange(code), maps ErrCodeExpired->410 Gone, ErrCodeNotFound->404, other errors->500. Verifies token via capability.Verify to extract claims.SID, returns ExchangeJoinCodeResponse with URL = base + "/sessions/" + SID + "?cap=" + token
      - POST /capability/regenerate-key -> generates new key via capability.GenerateKey, saves via FileKeyStore, calls ws.SetSigningKey(newKey), no response body (200 OK)
    - New client methods: IssueCapabilities, ExchangeJoinCode (returns URL string), RegenerateSigningKey.
    - New Wails bindings on *App:
      - IssueCapabilities(sessionID) -> (daemon.IssueCapabilitiesResponse, error)
      - ExchangeJoinCode(code) -> (string, error)
      - RegenerateSigningKey() -> error
      - GetCapabilityQRCode(joinURL) -> (string, error) — base64 PNG, mirrors GetWebServerQRCode at app.go:582; accepts the join-code exchange URL (D-09), not the capability token
  </behavior>
  <action>
    1. Append to `internal/daemon/types.go`:
       ```go
       // IssueCapabilitiesResponse is the response body for POST /sessions/{id}/capabilities
       // and the toggle-on path of POST /sessions/{id}/web-serve.
       type IssueCapabilitiesResponse struct {
           ReadURL   string `json:"readUrl"`
           WriteURL  string `json:"writeUrl"`
           ReadCode  string `json:"readCode"`
           WriteCode string `json:"writeCode"`
       }

       // ExchangeJoinCodeRequest is the body for POST /join/exchange.
       type ExchangeJoinCodeRequest struct {
           Code string `json:"code"`
       }

       // ExchangeJoinCodeResponse returns the capability-bearing URL the client should follow.
       type ExchangeJoinCodeResponse struct {
           URL string `json:"url"`
       }
       ```

    2. In `internal/daemon/api.go` route registration (lines 43-70 per PATTERNS), add:
       ```go
       a.mux.HandleFunc("POST /sessions/{id}/capabilities", a.handleIssueCapabilities)
       a.mux.HandleFunc("POST /join/exchange",              a.handleExchangeJoinCode)
       a.mux.HandleFunc("POST /capability/regenerate-key",  a.handleRegenerateSigningKey)
       ```

    3. Add the three handlers (mirror handleWebServe pattern at lines 501-524):
       ```go
       func (a *API) handleIssueCapabilities(w http.ResponseWriter, r *http.Request) {
           id := r.PathValue("id")
           a.mu.RLock(); ws := a.webServer; a.mu.RUnlock()
           if ws == nil { http.Error(w, "web server not running", http.StatusBadRequest); return }
           if !ws.IsSessionEnabled(id) { http.Error(w, "session not web-enabled", http.StatusBadRequest); return }
           readURL, writeURL, readCode, writeCode, err := a.issueCapabilitiesForSession(id)
           if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
           writeJSON(w, http.StatusOK, IssueCapabilitiesResponse{ReadURL: readURL, WriteURL: writeURL, ReadCode: readCode, WriteCode: writeCode})
       }

       func (a *API) handleExchangeJoinCode(w http.ResponseWriter, r *http.Request) {
           var req ExchangeJoinCodeRequest
           if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "invalid request body", http.StatusBadRequest); return }
           if req.Code == "" { http.Error(w, "code required", http.StatusBadRequest); return }
           token, err := a.joinCodes.Exchange(req.Code)
           switch {
           case errors.Is(err, capability.ErrCodeExpired): http.Error(w, "code expired", http.StatusGone); return
           case errors.Is(err, capability.ErrCodeNotFound): http.Error(w, "invalid code", http.StatusNotFound); return
           case err != nil: http.Error(w, "internal error", http.StatusInternalServerError); return
           }
           claims, err := capability.Verify(token, a.signingKey)
           if err != nil { http.Error(w, "internal error", http.StatusInternalServerError); return }
           a.mu.RLock(); ws := a.webServer; a.mu.RUnlock()
           if ws == nil { http.Error(w, "web server not running", http.StatusBadRequest); return }
           url := ws.BaseURL() + "/sessions/" + claims.SID + "?cap=" + token
           writeJSON(w, http.StatusOK, ExchangeJoinCodeResponse{URL: url})
       }

       func (a *API) handleRegenerateSigningKey(w http.ResponseWriter, r *http.Request) {
           newKey, err := capability.GenerateKey()
           if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
           store := capability.NewFileKeyStore(a.configDir)
           if err := store.Save(newKey); err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
           a.mu.Lock()
           a.signingKey = newKey
           ws := a.webServer
           a.mu.Unlock()
           if ws != nil { ws.SetSigningKey(newKey) }
           // Also clear all grants — all outstanding caps are now invalid anyway,
           // and clearing the maps prevents stale entries from accumulating.
           // Note: this is a belt-and-suspenders measure; signature check already fails.
           // (Left as a comment — no call — to avoid tight coupling to WebServer internals beyond SetSigningKey.)
           w.WriteHeader(http.StatusOK)
       }
       ```

    4. Append to `internal/daemon/client.go` (mirror ToggleWebServing at lines 204-207):
       ```go
       func (c *DaemonClient) IssueCapabilities(sessionID string) (IssueCapabilitiesResponse, error) {
           var resp IssueCapabilitiesResponse
           err := c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/capabilities", nil, &resp)
           return resp, err
       }

       func (c *DaemonClient) ExchangeJoinCode(code string) (string, error) {
           var resp ExchangeJoinCodeResponse
           if err := c.doJSON(http.MethodPost, "/join/exchange", ExchangeJoinCodeRequest{Code: code}, &resp); err != nil {
               return "", err
           }
           return resp.URL, nil
       }

       func (c *DaemonClient) RegenerateSigningKey() error {
           return c.doJSON(http.MethodPost, "/capability/regenerate-key", nil, nil)
       }
       ```

    5. Append to `app.go` (mirror ToggleWebServing at app.go:520-525 and GetWebServerQRCode at app.go:582-596):
       ```go
       func (a *App) IssueCapabilities(sessionID string) (daemon.IssueCapabilitiesResponse, error) {
           if a.client == nil { return daemon.IssueCapabilitiesResponse{}, fmt.Errorf("daemon not connected") }
           return a.client.IssueCapabilities(sessionID)
       }

       func (a *App) ExchangeJoinCode(code string) (string, error) {
           if a.client == nil { return "", fmt.Errorf("daemon not connected") }
           return a.client.ExchangeJoinCode(code)
       }

       func (a *App) RegenerateSigningKey() error {
           if a.client == nil { return fmt.Errorf("daemon not connected") }
           return a.client.RegenerateSigningKey()
       }

       func (a *App) GetCapabilityQRCode(joinURL string) (string, error) {
           // joinURL is the join-code exchange URL per D-09 (e.g. https://host/join?code=A7K-4P2N),
           // NOT the capability token URL. The encoder call is identical to GetWebServerQRCode.
           png, err := qrcode.Encode(joinURL, qrcode.Medium, 256)
           if err != nil { return "", err }
           return base64.StdEncoding.EncodeToString(png), nil
       }
       ```
       Imports: `fmt`, `encoding/base64`, `github.com/skip2/go-qrcode`, `github.com/scottkw/agenthub/internal/daemon` — most already imported.

    6. Run Wails binding regeneration. The Wails v2 build system regenerates `frontend/src/wailsjs/go/main/App.d.ts` automatically on build. For this task, just run `go build ./...` to confirm everything compiles. A note in the done line: frontend Plan 05 will run `wails generate module` or equivalent to refresh bindings.

    7. Add a single integration test to api_test.go:
       - TestIPCHandlers_CapabilityRoundTrip: POST /sessions/{sid}/capabilities -> extract ReadURL and ReadCode -> POST /join/exchange {Code: ReadCode} -> assert URL contains the same ?cap= token as ReadURL
       - TestIPCHandlers_ExpiredCodeReturns410: Issue code, advance joinCodes now func past TTL, POST /join/exchange -> 410 Gone
       - TestIPCHandlers_RegenerateSigningKey_SwapsKey: record current signingKey; POST /capability/regenerate-key; assert a.signingKey changed and capability.key file has new contents

    8. Run `go test ./internal/daemon/ -count=1 -v -run 'TestIPCHandlers'` — all three PASS.

    Anti-patterns:
    - DO NOT wire the regenerate-key handler to auto-re-issue all caps (blast radius is intentional per D-16)
    - DO NOT accept GET for /join/exchange (must be POST per RESEARCH anti-patterns — prevents browser prefetch replay)
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build ./... && go test ./internal/daemon/ -count=1 -v -run 'TestIPCHandlers' 2>&1 | tee /tmp/ipc.log ; ! grep -q FAIL /tmp/ipc.log && grep -q 'IssueCapabilitiesResponse' internal/daemon/types.go && grep -q 'handleIssueCapabilities\|handleExchangeJoinCode\|handleRegenerateSigningKey' internal/daemon/api.go && grep -q 'func (c \*DaemonClient) IssueCapabilities\|func (c \*DaemonClient) ExchangeJoinCode\|func (c \*DaemonClient) RegenerateSigningKey' internal/daemon/client.go && grep -q 'func (a \*App) IssueCapabilities\|func (a \*App) ExchangeJoinCode\|func (a \*App) RegenerateSigningKey\|func (a \*App) GetCapabilityQRCode' app.go && go test ./... -count=1 2>&1 | tee /tmp/all.log ; ! grep -q FAIL /tmp/all.log</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./...` succeeds
    - `grep -q "IssueCapabilitiesResponse" internal/daemon/types.go` succeeds
    - `grep -q "ExchangeJoinCodeRequest" internal/daemon/types.go` succeeds
    - `grep -q "handleIssueCapabilities" internal/daemon/api.go` succeeds
    - `grep -q "handleExchangeJoinCode" internal/daemon/api.go` succeeds
    - `grep -q "handleRegenerateSigningKey" internal/daemon/api.go` succeeds
    - `grep -q "POST /sessions/{id}/capabilities" internal/daemon/api.go` succeeds
    - `grep -q "POST /join/exchange" internal/daemon/api.go` succeeds
    - `grep -q "POST /capability/regenerate-key" internal/daemon/api.go` succeeds
    - Client has 3 new methods (grep -q for each)
    - App has 4 new Wails bindings (IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey, GetCapabilityQRCode)
    - Integration tests TestIPCHandlers_CapabilityRoundTrip, TestIPCHandlers_ExpiredCodeReturns410, TestIPCHandlers_RegenerateSigningKey_SwapsKey all PASS
    - `go test ./... -count=1` passes (full suite no regression)
  </acceptance_criteria>
  <done>Three IPC endpoints, three typed client methods, four Wails bindings. End-to-end token round-trip works: toggle-on issues two caps with join codes; join-code POST /join/exchange returns the capability URL; regenerate-key swaps the signing key in place.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| process start → disk | Daemon reads or generates capability.key (mode 0600) before accepting any HTTP/IPC request |
| Unix socket → daemon | IPC endpoints for capability issuance, exchange, rotation — trusted (socket boundary is the existing trust line) |
| daemon memory → WS | SetSigningKey crosses process-internal trust boundary; protected by ws.mu |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-87-05 | Information Disclosure | auto-share on session create (SEC-01) | mitigate | Auto-enable block deleted from handleCreateSession; TestHandleCreateSession_NoAutoEnable in task 87-04-01 |
| T-87-06 | Availability | signing key lost across restart | mitigate | engine.go startup calls LoadOrGenerate; TestStartup_LoadsOrGeneratesSigningKey in task 87-04-01 |
| T-87-07 | Elevation of Privilege | revoked grant / toggle-off persistence leak | mitigate | handleWebServe disable path calls ClearGrants; onExit ClearGrants (Pitfall 1); TestHandleWebServe_ToggleOffClearsGrants + TestOnExit_ClearsGrants in task 87-04-01 |
| T-87-03 | Spoofing | forged token passed through /join/exchange | mitigate | Exchange verifies token via capability.Verify before returning URL; otherwise 500 internal error |
| T-87-08 | Information Disclosure | RegenerateSigningKey race | mitigate | a.mu.Lock held while writing a.signingKey and fetching ws; ws.SetSigningKey uses its own ws.mu for atomic swap |
</threat_model>

<verification>
Phase-level gate after this plan:
- `go test ./... -count=1` green (capability + webserver + daemon + all)
- Full SEC-01..SEC-05 test coverage is active: daemon-level SEC-01 (auto-enable removal), webserver-level SEC-02..SEC-05 (from Plan 03)
- No handler accepts GET for /join/exchange
- Wails binding file compiles (a.client invocation shape matches existing patterns exactly)
</verification>

<success_criteria>
- 5 behavioral tests from task 87-04-01 PASS
- 3 integration tests from task 87-04-02 PASS
- `go test ./... -count=1` full suite passes
- All 4 Wails bindings present in app.go
- capability.key is created in the daemon config dir on first run and reloaded on restart
</success_criteria>

<output>
After completion, create `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-04-SUMMARY.md` documenting: the SEC-01 auto-enable removal diff, the onExit Pitfall 1 fix, the 3 IPC endpoints + 4 Wails bindings, and the exact error->HTTP-status mapping for /join/exchange (expired=410, not-found=404, bad-body=400, verify-fail=500).
</output>
