---
phase: 165-funnel-backend
verified: 2026-06-30T15:57:01Z
status: human_needed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "M-34: External-tailnet HTTP request gets 200 on the no-port Funnel URL"
    expected: "A machine with no Tailscale installed opens https://<hostname>.ts.net/sessions/<id>?cap=TOKEN and receives 200 (not 403). The Origin: https://<hostname>.ts.net header must be present."
    why_human: "Requires a live Tailscale Funnel-enabled tailnet node and an external machine off the tailnet. The unit test (TestRequireAllowedOrigin_FunnelOrigin) verifies the Origin check logic with a fake; the actual Funnel TCP routing and real-world DNS resolution require live infrastructure."
  - test: "M-35: tailscale serve status shows empty config after each teardown trigger"
    expected: "After each of the four production teardown paths (toggle-off, web-share-off, session natural end, daemon clean stop), running `tailscale serve status` on the host reports no serve entries or an empty config."
    why_human: "The unit tests (TestFunnelTeardown_AllTriggers) verify via a fake funnelClient that SetServeConfig is called with the cleared config. The live tailscaled response to those calls and the visible output of `tailscale serve status` require a real tailscaled process."
  - test: "M-36: Local-network fallback web-share continues unaffected with tailscaled stopped"
    expected: "With tailscaled not running (or Tailscale not installed), an existing web-share session remains accessible on the local tailnet URL. funnelActive stays false; attempting to enable Funnel returns an error. The web-share guest connection is not disrupted."
    why_human: "Requires stopping tailscaled and running the daemon against a live session. The unit test (TestEnableFunnel_FallbackModeSafe) proves the code guard; the actual service continuity requires a live environment."
---

# Phase 165: Funnel Backend Verification Report

**Phase Goal:** The daemon can activate and fully tear down Tailscale Funnel with correct Origin/BaseURL awareness, so an internet-external guest can reach a Funnel-enabled session.
**Verified:** 2026-06-30T15:57:01Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A Funnel share URL emitted by the daemon has no port suffix (`https://hostname.ts.net/sessions/id?cap=TOKEN`, not `:7443`) | VERIFIED | `server.go:633` — `ws.funnelBaseURL = "https://" + hostname // no port — 443 is default HTTPS`. URL builders in `api.go:1336-1340` and `api.go:1445-1449` swap base to `ws.FunnelBaseURL()` (no port). `TestEnableFunnelCallsSetServeConfig` asserts `FunnelBaseURL()=="https://mynode.ts.net"` (PASS). `TestIssueCapabilities_FunnelURL` asserts no port in the emitted URL (PASS). |
| 2 | An HTTP request carrying `Origin: https://hostname.ts.net` receives 200, not 403 (after EnableFunnel) | VERIFIED | `origin_mw.go:62-64` — exact-match secondary branch: `if funnelURL := ws.FunnelBaseURL(); funnelURL != "" && origin == funnelURL`. `TestRequireAllowedOrigin_FunnelOrigin/FunnelOrigin_Active_200` (PASS), `FunnelOrigin_Inactive_403` (PASS), `TailnetOrigin_Unaffected_200` (PASS), `UnrelatedOrigin_403` (PASS). Fail-closed when `FunnelBaseURL()==""`. |
| 3 | tailscale serve config is empty after each of the four teardown triggers (toggle-off, web-share-off, session ends naturally, daemon stops cleanly) | VERIFIED | All five teardown sites wired through `disableFunnelForSession` (`api.go:1520`, `api.go:1245`, `api.go:698`, `server.go:511`). `TestFunnelTeardown_AllTriggers` (5 sub-tests, each driving its real entry point, asserting `fake.IsFunnelOn()==false`) — all PASS. `TestWebServerStop_DisablesFunnel` (PASS). Teardown assertions via `daemonFakeFunnelClient.IsFunnelOn()` on stored serve config — not struct-field read. |
| 4 | When Funnel prerequisites are not met, `EnableFunnel` returns a human-readable error matching `ipn.CheckFunnelAccess` output and never calls `SetServeConfig` | VERIFIED | `server.go:579` — `ipn.CheckFunnelAccess(funnelPort, st.Self)` called before `ws.mu.Lock()` and before any `SetServeConfig`. `TestEnableFunnel_PrereqCheckPreventsSetServeConfig` asserts error contains `"Funnel not available"` (verbatim Tailscale text) and `setServeConfigCalled==false` (PASS). `handleSetSessionFunnel` surfaces the error as HTTP 400 so the text reaches the API caller (FNL-06). |
| 5 | Web-share in local-network fallback mode (Tailscale not running) continues unaffected and `funnelActive` remains false | VERIFIED | `server.go:572-577` — `StatusWithoutPeers` error path returns early without acquiring the lock or calling `SetServeConfig`. `TestEnableFunnel_FallbackModeSafe` asserts `FunnelBaseURL()==""` (funnelActive stays false) and `setServeConfigCalled==false` (PASS). |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/webserver/funnel_client.go` | funnelClient interface + FunnelClientForTest alias + SetFunnelClientForTest | VERIFIED | Exists (2,249 bytes). Defines `funnelClient` (3 methods: GetServeConfig, SetServeConfig, StatusWithoutPeers). Exported alias `FunnelClientForTest = funnelClient`. Method `SetFunnelClientForTest` takes lock and assigns. Production wired: `NewWebServer` sets `ws.funnelClient = &ws.lc`. |
| `internal/webserver/funnel_test.go` | 7 test functions exercising Enable/Disable/Origin via fake | VERIFIED | Exists (14,776 bytes). Contains `fakeFunnelClient`, `validFunnelStatus`, `TestFunnelClient_CompileSmoke`, `TestEnableFunnelCallsSetServeConfig`, `TestEnableFunnel_PrereqCheckPreventsSetServeConfig`, `TestEnableFunnel_FallbackModeSafe`, `TestWebServerStop_DisablesFunnel`, `TestRequireAllowedOrigin_FunnelOrigin` (4 sub-tests), `TestOriginAllowedForWrite_FunnelOrigin`. All 7 functions pass. |
| `internal/daemon/funnel_test.go` | 9 daemon-level funnel tests + daemonFakeFunnelClient | VERIFIED | Exists (25,722 bytes). Contains `daemonFakeFunnelClient` (stateful, thread-safe, ETag-aware), plus `TestFunnelSessionsMap`, `TestHandleSetSessionFunnel_Enable`, `TestHandleSetSessionFunnel_DisableTeardown`, `TestFunnelAutoExpiry`, `TestFunnelTeardown_AllTriggers` (5 sub-tests), `TestFunnelTeardown_RefCountKeepsSiblingUp`, `TestIssueCapabilities_FunnelURL`, `TestExchangeJoinCode_FunnelURL_GateIntact`, `TestStartupClearsLingeringFunnel`. All 13 tests pass. |
| `internal/webserver/server.go` (modified) | EnableFunnel, DisableFunnel, FunnelBaseURL, ClearLingeringFunnel + struct fields | VERIFIED | Fields at lines 87-97: `lc local.Client`, `funnelClient funnelClient`, `funnelActive bool`, `funnelBaseURL string`, `funnelPort uint16`. Methods at lines 557-724: full implementations. `Stop()` at line 511 calls `ws.DisableFunnel`. `NewWebServer` at line 196 sets `ws.funnelClient = &ws.lc`. |
| `internal/daemon/api.go` (modified) | funnelSessions/funnelExpiry maps, POST /sessions/{id}/funnel route, handleSetSessionFunnel, disableFunnelForSession, FunnelActive population, URL builders, ClearLingeringFunnel startup | VERIFIED | Maps at lines 71-80. Route at line 159-160. Handler at line 1503. Helper at line 1567. Teardown sites at 1245 (web-share-off), 698 (session-end), expiry callback at 1550. FunnelActive population at 635-638. URL builders at 1328-1340 and 1437-1449. Startup clear at 513. |
| `internal/daemon/types.go` (modified) | SetSessionFunnelRequest, SetSessionFunnelResponse, SessionInfo.FunnelActive | VERIFIED | `SetSessionFunnelRequest` at line 154-159. `SetSessionFunnelResponse` at line 161-165. `SessionInfo.FunnelActive bool` at line 34 with `json:"funnelActive"` (no omitempty). |
| `app.go` (modified) | App.SetSessionFunnel, SessionInfo.FunnelActive field, ListSessions propagation | VERIFIED | `SessionInfo.FunnelActive bool` at line 50. `App.SetSessionFunnel` at lines 894-902 with nil-guard. `FunnelActive: s.FunnelActive` copy in ListSessions at line 382. |
| `internal/daemon/client.go` (modified) | DaemonClient.SetSessionFunnel | VERIFIED | Method at lines 361-369, POSTs `SetSessionFunnelRequest{Enabled: enabled, ExpiresIn: expiresIn}` to `/sessions/`+sessionID+`/funnel`. Mirrors ToggleWebServing/SetSessionBrowse pattern exactly. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `EnableFunnel` sets `ws.funnelActive + ws.funnelBaseURL` | `requireAllowedOrigin` / `allowedOrigins` / `originAllowedForWrite` read for secondary origin check | `ws.FunnelBaseURL()` return value | VERIFIED | `origin_mw.go:63` and `capability_mw.go:206` call `ws.FunnelBaseURL()` for secondary check. Fail-closed when empty. |
| `NewWebServer` | `ws.funnelClient = &ws.lc` | Direct assignment at `server.go:196` | VERIFIED | Production path wires concrete `*local.Client`; tests override via `ws.funnelClient = fake`. |
| `handleSetSessionFunnel` | `ws.EnableFunnel` | `server.go` EnableFunnel called at `api.go:1526-1528`; `funnelSessions[id]=true` at `api.go:1536` | VERIFIED | `handleSetSessionFunnel` calls real `ws.EnableFunnel(r.Context(), 443)` and only records `funnelSessions[id]=true` on success. |
| All 5 teardown triggers | Single `disableFunnelForSession` | Toggle-off (1520), web-share-off (1245), session-end (698), ws.Stop→DisableFunnel (server.go:511), expiry-timer (1550) | VERIFIED | Five distinct call sites all converge on the same helper. Ref-count gate at `len(funnelSessions)==0`. |
| `issueCapabilitiesForSession` + `handleExchangeJoinCode` | `ws.FunnelBaseURL()` | Snapshot `isFunnelSession := a.funnelSessions[sessionID]` then swap base if non-empty | VERIFIED | Both builders: `if isFunnelSession { if fb := ws.FunnelBaseURL(); fb != "" { base = fb } }`. Fail-safe: keeps tailnet base if FunnelBaseURL is empty. |
| `App.SetSessionFunnel` | `DaemonClient.SetSessionFunnel` | Nil-guard + delegate in `app.go:898-902` | VERIFIED | `TestApp_SetSessionFunnel_NilClient` confirms nil-guard returns "daemon not connected". `TestDaemonClient_SetSessionFunnel` confirms POST to correct path with correct body. |
| `App.ListSessions` | `SessionInfo.FunnelActive` propagation | `FunnelActive: s.FunnelActive` copy at `app.go:382` | VERIFIED | `TestListSessions_PropagatesFunnelActive` confirms true and false both propagate. |

### Data-Flow Trace (Level 4)

All artifacts that render dynamic data are API routes and daemon maps (not UI components). Data flow traced:

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `handleSetSessionFunnel` response | `FunnelURL string` | `ws.FunnelBaseURL()` ← `ws.funnelBaseURL` ← `EnableFunnel` ← `funnelClient.StatusWithoutPeers` + `SetServeConfig` | Yes — set from real Tailscale hostname or fake in tests | FLOWING |
| `handleListSessions` `FunnelActive` | `funnelSessions[id]` snapshot | `funnelSessions map[string]bool` mutated by `handleSetSessionFunnel` and `disableFunnelForSession` | Yes — boolean reflects real enable/disable calls | FLOWING |
| `App.ListSessions` `FunnelActive` | `s.FunnelActive` | daemon `SessionInfo.FunnelActive` from `funnelSessions` snapshot | Yes — propagated without omitempty | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Webserver funnel tests (interface seam, Enable/Disable lifecycle, Origin allowlist) | `go test ./internal/webserver/... -run 'TestFunnelClient\|TestEnableFunnel\|TestWebServerStop_DisablesFunnel\|TestRequireAllowedOrigin_FunnelOrigin\|TestOriginAllowedForWrite' -count=1` | 7 tests PASS, ok in 0.022s | PASS |
| Daemon funnel tests (all 5 teardown triggers, ref-count, URL builders, startup clear, auto-expiry) | `go test ./internal/daemon/... -run 'TestFunnelSessionsMap\|TestHandleSetSessionFunnel\|TestFunnelAutoExpiry\|TestFunnelTeardown\|TestIssueCapabilities_FunnelURL\|TestExchangeJoinCode_FunnelURL\|TestStartupClearsLingeringFunnel' -count=1` | 13 tests PASS (including 5 sub-tests of AllTriggers), ok in 4.328s | PASS |
| Wails bridge tests (nil-guard, client POST, FunnelActive propagation) | `go test . ./internal/daemon/... -run 'TestApp_SetSessionFunnel_NilClient\|TestDaemonClient_SetSessionFunnel\|TestListSessions_PropagatesFunnelActive' -count=1` | 4 tests PASS | PASS |
| Full build | `go build ./...` | No output (success) | PASS |
| Full webserver + daemon test suites | `go test ./internal/webserver/... ./internal/daemon/... -count=1` | Both packages ok | PASS |
| Traceability check | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| FNL-01 | 165-02, 165-03 | Session owner can enable Funnel; off by default | SATISFIED | `funnelSessions` starts empty (off by default). `POST /sessions/{id}/funnel` enables per-session. `App.SetSessionFunnel` + `DaemonClient.SetSessionFunnel` complete the Wails on-ramp. `TestFunnelSessionsMap` + `TestListSessions_PropagatesFunnelActive` verify. |
| FNL-02 | 165-01 | Uses embedded Tailscale LocalClient (SetServeConfig/AllowFunnel); no admin API token | SATISFIED | `funnelClient` interface wraps `local.Client.GetServeConfig` / `SetServeConfig` / `StatusWithoutPeers`. Production wired to `&ws.lc` (concrete `local.Client`). No admin API calls. `TestFunnelClient_CompileSmoke` verifies seam. |
| FNL-03 | 165-02 | External recipient uses Funnel share URL gated by join code + cap token | SATISFIED | Both URL builders swap to `ws.FunnelBaseURL()` (no-port) for Funnel sessions. Join-code consumption and cap token are unchanged. `TestIssueCapabilities_FunnelURL` + `TestExchangeJoinCode_FunnelURL_GateIntact` (reused code → 404, unknown code → 404) verify. |
| FNL-04 | 165-01 | Origin allowlist uses Funnel hostname; external join-code exchange succeeds without 403 | SATISFIED | `requireAllowedOrigin`, `allowedOrigins`, `originAllowedForWrite` all include secondary exact-match Funnel URL branch. Fail-closed when `FunnelBaseURL()==""`. `TestRequireAllowedOrigin_FunnelOrigin` (4 sub-tests) + `TestOriginAllowedForWrite_FunnelOrigin` verify. |
| FNL-05 | 165-01, 165-02 | Funnel torn down on toggle-off, web-share-off, session end, daemon shutdown | SATISFIED | All 5 teardown sites wired through `disableFunnelForSession`. Ref-count protects sibling sessions. `ws.Stop()` calls `DisableFunnel` (site 4). `TestFunnelTeardown_AllTriggers` (5 sub-tests) + `TestFunnelTeardown_RefCountKeepsSiblingUp` verify. |
| FNL-06 | 165-01, 165-02 | Prerequisite failure yields clear human-readable error; never fails opaquely | SATISFIED | `ipn.CheckFunnelAccess` called before lock or SetServeConfig; error returned verbatim. `handleSetSessionFunnel` surfaces as HTTP 400 body. `TestEnableFunnel_PrereqCheckPreventsSetServeConfig` verifies error contains `"Funnel not available"` and SetServeConfig not called. |
| FNL-07 | 165-02 | Auto-expiry after user-chosen duration; enforced server-side | SATISFIED | `time.AfterFunc` per-session in `funnelExpiry[id]`. Early teardown calls `Stop()+delete`. Re-enable cancels prior timer. `TestFunnelAutoExpiry` verifies timer fires, config clears with no HTTP call, re-enable cancels prior timer. `TestFunnelTeardown_AllTriggers/5_expiry_timer` also verifies. |

### Anti-Patterns Found

Scanned all 8 files modified/created in this phase (`funnel_client.go`, `funnel_test.go`, `server.go`, `origin_mw.go`, `capability_mw.go`, `api.go`, `types.go`, `app.go`, `client.go`, `daemon/funnel_test.go`).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | No TBD/FIXME/XXX/HACK/PLACEHOLDER markers found | — | — |

No stubs, no orphaned code, no hardcoded empty returns in data paths.

### Human Verification Required

#### 1. External-tailnet HTTP request receives 200 (M-34)

**Test:** On a machine that is NOT on the tailnet and has no Tailscale installed, open `https://<hostname>.ts.net/sessions/<id>?cap=TOKEN` in a browser. The Origin header will be `https://<hostname>.ts.net`. The request must receive a 200 response (not 403).

**Expected:** 200 OK — the Funnel URL is served without a "Forbidden" response. The terminal WebSocket connects successfully.

**Why human:** Requires a live Tailscale Funnel-enabled node (activated via `SetServeConfig`) and an external machine off the tailnet. The unit test `TestRequireAllowedOrigin_FunnelOrigin` proves the Origin check logic via fake; end-to-end Funnel TCP routing and real DNS resolution require live infrastructure.

#### 2. tailscale serve status empty after each teardown trigger (M-35)

**Test:** On a live Tailscale node, enable Funnel for a session, then exercise each of the four production teardown paths one at a time:
1. Toggle-off via the API (`POST /sessions/{id}/funnel {enabled:false}`)
2. Web-share-off (`POST /sessions/{id}/web-serve {enabled:false}`)
3. Session natural end (let the session process exit normally)
4. Daemon clean stop (stop the agenthub daemon)

After each, run `tailscale serve status` and confirm the output shows no serve entries (or reports "no serve config").

**Expected:** `tailscale serve status` shows empty config after each trigger. No lingering TCP proxy entry or `AllowFunnel` annotation remains.

**Why human:** Unit tests (TestFunnelTeardown_AllTriggers, all 5 sub-tests) verify via `daemonFakeFunnelClient.IsFunnelOn()` that SetServeConfig is called with the cleared config. Whether the real tailscaled clears the serve status as observed by `tailscale serve status` requires a live tailscaled process and cannot be verified by code inspection.

#### 3. Fallback-mode web-share unaffected with tailscaled stopped (M-36)

**Test:** With tailscaled stopped (or Tailscale not installed on the host), establish a web-share session and confirm:
- The session is accessible on its local tailnet URL
- Attempting `POST /sessions/{id}/funnel {enabled:true}` returns an error (not a panic, not a silent success)
- The existing web-share guest connection is unaffected by the failed Funnel enable
- `funnelActive` remains false in GET /sessions

**Expected:** Web-share works normally. Funnel enable returns an error (`"tailscaled not running"` or similar). No disruption to the existing web-share.

**Why human:** Unit test `TestEnableFunnel_FallbackModeSafe` proves the error guard via fake StatusWithoutPeers. The actual service continuity of an existing web-share session while Funnel is unavailable requires a live environment test.

---

## Summary

Phase 165 delivers a complete, multi-layer Tailscale Funnel backend:

**webserver layer (165-01):** Injectable `funnelClient` interface seam with `EnableFunnel`/`DisableFunnel`/`FunnelBaseURL`/`ClearLingeringFunnel` on `WebServer`. Dual-origin allowlist (exact-match, fail-closed) in `requireAllowedOrigin`, `allowedOrigins`, and `originAllowedForWrite`. CheckFunnelAccess prerequisites checked before lock and before SetServeConfig; fallback-mode guard prevents funnelActive from going true when tailscaled is unreachable. Stop() tears down Funnel (site 4).

**daemon layer (165-02):** Per-session `funnelSessions` reference-count map and `funnelExpiry` auto-expiry timers. `POST /sessions/{id}/funnel` endpoint wired to real `ws.EnableFunnel`. Single `disableFunnelForSession` helper used by all five teardown sites; ref-count protects sibling sessions. Funnel-aware URL builders in both `issueCapabilitiesForSession` and `handleExchangeJoinCode` (host-only swap; single-use join-code gate intact). Daemon-restart clears lingering serve config via `ws.ClearLingeringFunnel`. `SessionInfo.FunnelActive` for frontend polling.

**Wails bridge (165-03):** `App.SetSessionFunnel` nil-guarded bound method + `DaemonClient.SetSessionFunnel` client delegation. `SessionInfo.FunnelActive` mirror in `app.go` with copy-loop propagation (no omitempty).

All 5 phase success criteria are VERIFIED by passing unit tests. The three human-verification items (M-34, M-35, M-36) are end-to-end live-Tailscale checks that cannot be automated; they are already registered in TESTING.md with full descriptions.

---

_Verified: 2026-06-30T15:57:01Z_
_Verifier: Claude (gsd-verifier)_
