---
phase: 165-funnel-backend
verified: 2026-06-30T20:15:00Z
status: passed
status_note: "Code-verified 9/9; all live-only UAT (M-34/M-35/M-36) PASSED on a real Funnel tailnet after the 165-05 loopback-HTTP fix. See Live UAT Addendum at EOF. NOTE: the human_verification block below was written for the 165-04 https+insecure target, which live UAT then proved insufficient (502 via TLS SNI) — superseded by 165-05; the addendum is authoritative."
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 5/5
  gaps_closed:
    - "FNL-03 502 (GAP 1): Funnel reverse-proxy target was https://localhost:<port> (unreachable when listener binds to ws.config.BindIP) — fixed to https+insecure://<bindIP>:<port> (Option A). TestEnableFunnel_ProxyTargetReachable asserts exact proxy string AND real HTTPS reachability (GET returns 200, not connection-refused). server.go:632 confirmed live."
    - "FNL-05 kill path (GAP 2): handleDeleteSession returned 204 without any cleanup, leaving Funnel exposed and a stale funnelSessions[id] ref-count entry after explicit kill. handleDeleteSession now calls a.runSessionExitCleanup(id) synchronously after KillSession (no 10s grace). TestFunnelTeardown_KillPath (2 sub-cases) confirms teardown + sibling ref-count integrity. api.go:742 confirmed live."
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "M-34: External-tailnet HTTP request gets 200 on the no-port Funnel URL"
    expected: "A machine with no Tailscale installed opens https://<hostname>.ts.net/sessions/<id>?cap=TOKEN and receives 200 (not 502, not 403). The 165-04 fix changes the proxy target from localhost (502) to the real bind IP (https+insecure). This item is now expected to PASS live — the automated guard TestEnableFunnel_ProxyTargetReachable confirms the proxy target string and real reachability in CI."
    why_human: "Requires a live Tailscale Funnel-granted tailnet node and an external machine off the tailnet. The unit test confirms the proxy target and reachability in loopback; the full end-to-end Funnel TCP routing and real tailscaled DNS require live infrastructure."
  - test: "M-35: tailscale serve status shows empty config after each teardown trigger"
    expected: "After each of the four production teardown paths (toggle-off, web-share-off, session natural end, daemon clean stop), running `tailscale serve status` on the host reports no serve entries or an empty config."
    why_human: "TestFunnelTeardown_AllTriggers (5 sub-tests) verifies via daemonFakeFunnelClient that SetServeConfig is called with the cleared config. The live tailscaled response and the output of `tailscale serve status` require a real tailscaled process."
  - test: "M-36: Local-network fallback web-share continues unaffected with tailscaled stopped"
    expected: "With tailscaled not running (or Tailscale not installed), an existing web-share session remains accessible on the local tailnet URL. funnelActive stays false; attempting to enable Funnel returns an error. The web-share guest connection is not disrupted."
    why_human: "TestEnableFunnel_FallbackModeSafe proves the error guard via fake StatusWithoutPeers. Actual service continuity of an existing web-share session while Funnel is unavailable requires a live environment."
---

# Phase 165: Funnel Backend Verification Report

**Phase Goal:** The daemon can activate and fully tear down Tailscale Funnel with correct Origin/BaseURL awareness, so an internet-external guest can reach a Funnel-enabled session.
**Verified:** 2026-06-30T20:15:00Z
**Status:** human_needed
**Re-verification:** Yes — after 165-04 gap-closure (FNL-03 502 proxy target + FNL-05 kill-path teardown)

## Gap-Closure Summary (165-04)

Two live-UAT defects found on a real Funnel-granted tailnet (fake-based unit tests could not catch them) were closed:

| Gap | Root Cause | Fix | Automated Guard |
|-----|-----------|-----|-----------------|
| GAP 1 / FNL-03 (BLOCKER) | Funnel proxy target hard-coded to `https://localhost:<port>`; listener binds to `ws.config.BindIP` (tailnet IP), not localhost — every external guest got HTTP 502 | `server.go:632` — target changed to `https+insecure://` + `net.JoinHostPort(ws.config.BindIP, localPort)` (Option A) | `TestEnableFunnel_ProxyTargetReachable` asserts exact string + real HTTPS reachability |
| GAP 2 / FNL-05 (MAJOR) | `handleDeleteSession` returned 204 with no cleanup — Funnel stayed exposed after explicit kill; stale `funnelSessions[id]` entry blocked sibling teardown | `api.go:742` — `a.runSessionExitCleanup(id)` called synchronously after `KillSession` (no 10s grace) | `TestFunnelTeardown_KillPath` (2 sub-cases: single teardown + ref-count guard) |

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A Funnel share URL emitted by the daemon has no port suffix (`https://hostname.ts.net/sessions/id?cap=TOKEN`, not `:7443`) | VERIFIED | `server.go:633` — `ws.funnelBaseURL = "https://" + hostname` (no port). URL builders (`api.go:1336-1340`, `api.go:1445-1449`) swap base to `ws.FunnelBaseURL()`. `TestEnableFunnelCallsSetServeConfig` and `TestIssueCapabilities_FunnelURL` confirm (PASS). |
| 2 | An HTTP request carrying `Origin: https://hostname.ts.net` receives 200, not 403 (after EnableFunnel) | VERIFIED | `origin_mw.go:62-64` — exact-match secondary branch on `ws.FunnelBaseURL()`. `TestRequireAllowedOrigin_FunnelOrigin` (4 sub-tests: FunnelOrigin_Active_200, FunnelOrigin_Inactive_403, TailnetOrigin_Unaffected_200, UnrelatedOrigin_403) — all PASS. Fail-closed when `FunnelBaseURL()==""`. |
| 3 | tailscale serve config is empty after each of the four teardown triggers (toggle-off, web-share-off, session ends naturally, daemon stops cleanly) | VERIFIED | All five teardown sites wired through `disableFunnelForSession` (`api.go:1520`, `api.go:1245`, `api.go:698`, `server.go:511`, expiry timer `api.go:1550`). `TestFunnelTeardown_AllTriggers` (5 sub-tests) and `TestWebServerStop_DisablesFunnel` — all PASS. |
| 4 | When Funnel prerequisites are not met, `EnableFunnel` returns a human-readable error and never calls `SetServeConfig` | VERIFIED | `server.go:579` — `ipn.CheckFunnelAccess` called before lock and before any `SetServeConfig`. `TestEnableFunnel_PrereqCheckPreventsSetServeConfig` asserts error contains `"Funnel not available"` and `setServeConfigCalled==false` (PASS). Surfaces as HTTP 400. |
| 5 | Web-share in local-network fallback mode (Tailscale not running) continues unaffected and `funnelActive` remains false | VERIFIED | `server.go:572-577` — `StatusWithoutPeers` error path returns early without acquiring lock or calling `SetServeConfig`. `TestEnableFunnel_FallbackModeSafe` asserts `FunnelBaseURL()==""` and `setServeConfigCalled==false` (PASS). |
| 6 | After EnableFunnel(ctx,443) the Funnel serve-config reverse-proxy target equals exactly `https+insecure://<ws.config.BindIP>:<listenerPort>` — NOT `https://localhost:<port>` (FNL-03 502 root cause, GAP 1 closed) | VERIFIED | `server.go:632` — `"https+insecure://" + net.JoinHostPort(ws.config.BindIP, localPort)`. `TestEnableFunnel_ProxyTargetReachable` asserts `handler.Proxy == "https+insecure://127.0.0.1:<port>"` (exact string — the old `localhost` host fails this assertion). Test PASS. |
| 7 | An HTTPS client with InsecureSkipVerify can connect to that proxy target and receive a real HTTP response (reachable — not connection-refused; the exact 502 condition) | VERIFIED | `TestEnableFunnel_ProxyTargetReachable` — reachability leg dials `https://127.0.0.1:<port>` with `InsecureSkipVerify`, receives HTTP 200 from the running mux (`funnel_test.go:372: "reachability: GET → 200"`). Connection-refused or timeout fails the test. PASS. |
| 8 | After an explicit kill (`DELETE /sessions/{id}`) of a Funnel-enabled session, the serve config is empty and `funnelSessions[id]` is removed — no leaked exposure, no stale ref-count entry (FNL-05 kill path, GAP 2 closed) | VERIFIED | `api.go:742` — `a.runSessionExitCleanup(id)` called synchronously after `KillSession`. `TestFunnelTeardown_KillPath/single_session_killed` drives real DELETE handler, asserts `fake.IsFunnelOn()==false` and GET /sessions reports `FunnelActive=false`. PASS. |
| 9 | Killing one of two Funnel-enabled sessions leaves the other session's Funnel up; the killed session's ref-count entry is removed so the sibling's later teardown succeeds (stale-ref-count regression gone) | VERIFIED | `TestFunnelTeardown_KillPath/refcount_killing_a_keeps_b_up` — kills A, asserts `fake.IsFunnelOn()==true` (B still up); then runs B's natural-exit cleanup, asserts `fake.IsFunnelOn()==false` (B's teardown cleared config). PASS. Old GAP 2 root cause: A's stale entry would have left `len==1 → DisableFunnel never called → config stays active`. |

**Score:** 9/9 truths verified (0 present, behavior-unverified)

### Regression Check — Original 5 Truths

All 5 truths from the initial (pre-gap-closure) verification are unaffected by the 165-04 changes. Full re-run confirmed:

| Suite | Command | Result |
|-------|---------|--------|
| Webserver funnel (original 7 functions) | `go test ./internal/webserver/... -run 'TestFunnelClient\|TestEnableFunnel\|TestWebServerStop_DisablesFunnel\|TestRequireAllowedOrigin\|TestOriginAllowedForWrite'` | 8 tests PASS (includes new ProxyTargetReachable) |
| Daemon funnel (original 13 functions) | `go test ./internal/daemon/... -run 'TestFunnelSessionsMap\|TestHandleSetSessionFunnel\|TestFunnelAutoExpiry\|TestFunnelTeardown_AllTriggers\|TestFunnelTeardown_RefCountKeepsSiblingUp\|TestIssueCapabilities_FunnelURL\|TestExchangeJoinCode_FunnelURL\|TestStartupClearsLingeringFunnel'` | 13 tests PASS (5 AllTriggers sub-tests included) |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/webserver/funnel_client.go` | funnelClient interface + FunnelClientForTest alias + SetFunnelClientForTest | VERIFIED | Exists. Defines `funnelClient` (3 methods). Production wired: `NewWebServer` sets `ws.funnelClient = &ws.lc`. |
| `internal/webserver/funnel_test.go` | 8 test functions (7 original + TestEnableFunnel_ProxyTargetReachable) | VERIFIED | Exists. New test at line 286 with proxy-string assertion and reachability leg. All 8 functions PASS. |
| `internal/daemon/funnel_test.go` | 15 test functions (13 original + TestFunnelTeardown_KillPath with 2 sub-cases) | VERIFIED | Exists. New test at line 678 with two sub-cases. All 15 functions PASS. |
| `internal/webserver/server.go` (modified) | EnableFunnel Proxy target: `https+insecure://<bindIP>:<port>` with Option A comment | VERIFIED | `server.go:615-632` — 8-line Option A comment + target at line 632: `"https+insecure://" + net.JoinHostPort(ws.config.BindIP, localPort)`. All other EnableFunnel steps unchanged. |
| `internal/daemon/api.go` (modified) | handleDeleteSession calls `a.runSessionExitCleanup(id)` synchronously after KillSession with WHY comment | VERIFIED | `api.go:727-742` — 9-line WHY comment (no double-cleanup race, ref-count authority) + `a.runSessionExitCleanup(id)` at line 742. |

### Key Link Verification — Gap-Closure Additions

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `EnableFunnel` Proxy target | `ws.config.BindIP` (real bind address) | `net.JoinHostPort(ws.config.BindIP, localPort)` at `server.go:632` | VERIFIED | Host is BindIP (not `localhost`). Scheme is `https+insecure`. HostPort map key (hostname:funnelPort) unchanged. |
| `handleDeleteSession` | `a.runSessionExitCleanup(id)` | Synchronous call at `api.go:742` after `engine.KillSession(id)` succeeds | VERIFIED | No `time.AfterFunc`, no 10s grace. KillSession sets `sess.IsKilled()` so natural-exit goroutine returns early — no double-cleanup race. `disableFunnelForSession` remains the ref-count authority. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| GAP 1: proxy target string + real reachability | `go test ./internal/webserver/... -run 'TestEnableFunnel_ProxyTargetReachable' -count=1 -v` | `reachability: GET "https://127.0.0.1:<port>" → 200` — PASS in 0.02s | PASS |
| GAP 2: kill-path teardown, single session | `go test ./internal/daemon/... -run 'TestFunnelTeardown_KillPath/single_session_killed' -count=1 -v` | `--- PASS: TestFunnelTeardown_KillPath/single_session_killed (0.02s)` | PASS |
| GAP 2: kill-path teardown, ref-count guard | `go test ./internal/daemon/... -run 'TestFunnelTeardown_KillPath/refcount_killing_a_keeps_b_up' -count=1 -v` | `--- PASS: TestFunnelTeardown_KillPath/refcount_killing_a_keeps_b_up (0.02s)` | PASS |
| Full webserver suite | `go test ./internal/webserver/... -count=1` | `ok github.com/scottkw/agenthub/internal/webserver 9.904s` | PASS |
| Full daemon suite | `go test ./internal/daemon/... -count=1` | `ok github.com/scottkw/agenthub/internal/daemon 25.030s` | PASS |
| Traceability check | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |
| Full build | `go build ./...` | No output (success — confirmed by full suite green) | PASS |

### TESTING.md Wiring Verification

| Section | Update | Status |
|---------|--------|--------|
| Section 2 — Suite Manifest | Phase 165-04 gap-closure note: `TestEnableFunnel_ProxyTargetReachable` and `TestFunnelTeardown_KillPath` named; file counts unchanged (no new files) | VERIFIED — line 38 |
| Section 4 — Traceability | New FNL-03 row (`internal/webserver/funnel_test.go`): `TestEnableFunnel_ProxyTargetReachable` described. Extended FNL-05 row: `TestFunnelTeardown_KillPath` (kill + ref-count guard) added | VERIFIED — lines 275, 279 |
| Section 5 — M-34 | Updated to "expected to PASS live after 165-04 fix"; 165-04 root cause + `TestEnableFunnel_ProxyTargetReachable` automated guard cited | VERIFIED — line 534-540 |
| Section 5 — M-35, M-36 | Retained unchanged | VERIFIED |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| FNL-01 | 165-02, 165-03 | Session owner can enable Funnel; off by default | SATISFIED | `funnelSessions` starts empty. `POST /sessions/{id}/funnel` enables per-session. `App.SetSessionFunnel` + `DaemonClient.SetSessionFunnel` complete Wails on-ramp. `TestFunnelSessionsMap` + `TestListSessions_PropagatesFunnelActive` verify. |
| FNL-02 | 165-01 | Uses embedded Tailscale LocalClient; no admin API token | SATISFIED | `funnelClient` interface wraps `local.Client`. Production wired to `&ws.lc`. No admin API calls. `TestFunnelClient_CompileSmoke` verifies seam. |
| FNL-03 | 165-02, **165-04** | External recipient uses Funnel share URL gated by join code + cap token; proxy target reachable (no 502) | SATISFIED | URL builders use `ws.FunnelBaseURL()` (no-port). **165-04 fix:** proxy target `https+insecure://<bindIP>:<port>` (not localhost). `TestEnableFunnel_ProxyTargetReachable` asserts exact string + real reachability. `TestIssueCapabilities_FunnelURL` + `TestExchangeJoinCode_FunnelURL_GateIntact` verify URL builders. |
| FNL-04 | 165-01 | Origin allowlist uses Funnel hostname; external join-code exchange succeeds without 403 | SATISFIED | `requireAllowedOrigin`, `allowedOrigins`, `originAllowedForWrite` all include secondary exact-match Funnel URL branch. Fail-closed when `FunnelBaseURL()==""`. `TestRequireAllowedOrigin_FunnelOrigin` (4 sub-tests) + `TestOriginAllowedForWrite_FunnelOrigin` verify. |
| FNL-05 | 165-01, 165-02, **165-04** | Funnel torn down on toggle-off, web-share-off, session end, daemon shutdown; also on explicit kill with no ref-count leak | SATISFIED | Five natural teardown sites wired through `disableFunnelForSession`. `TestFunnelTeardown_AllTriggers` (5 sub-tests). **165-04 fix:** `handleDeleteSession` now calls `runSessionExitCleanup(id)` synchronously. `TestFunnelTeardown_KillPath` (2 sub-cases) verifies kill-path teardown + sibling ref-count integrity. |
| FNL-06 | 165-01, 165-02 | Prerequisite failure yields clear human-readable error; never fails opaquely | SATISFIED | `ipn.CheckFunnelAccess` called before lock or SetServeConfig; surfaces as HTTP 400. `TestEnableFunnel_PrereqCheckPreventsSetServeConfig` verifies. |
| FNL-07 | 165-02 | Auto-expiry after user-chosen duration; enforced server-side | SATISFIED | `time.AfterFunc` per-session in `funnelExpiry[id]`. Re-enable cancels prior timer. `TestFunnelAutoExpiry` verifies. |

### Anti-Patterns Found

Scanned all 5 files modified by 165-04 (`server.go`, `funnel_test.go` [webserver], `api.go`, `funnel_test.go` [daemon], `TESTING.md`).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | No TBD/FIXME/XXX/HACK/PLACEHOLDER markers found | — | — |

No stubs, no orphaned code, no hardcoded empty returns in data paths. The Option A decision (https+insecure) is documented with a WHY comment at `server.go:615-632`.

### Human Verification Required

These three items cannot be auto-verified (require a live Tailscale Funnel-granted tailnet). They are unchanged from initial verification, except M-34 now carries an explicit "expected to PASS live" note because the proxy-target fix closes the 502 root cause.

#### 1. External-tailnet HTTP request receives 200 (M-34) — Expected to PASS live after 165-04 fix

**Test:** On a machine that is NOT on the tailnet and has no Tailscale installed, open `https://<hostname>.ts.net/sessions/<id>?cap=TOKEN` in a browser.

**Expected:** 200 OK — the Funnel URL is served without a 502 or 403. The 165-04 fix changes the proxy target from `https://localhost:<port>` (unreachable) to `https+insecure://<bindIP>:<port>` (the address the listener actually binds to). `TestEnableFunnel_ProxyTargetReachable` confirms the exact proxy string and real reachability in loopback CI.

**Why human:** Requires a live Tailscale Funnel-granted node and an external machine off the tailnet. The automated guard proves the proxy target in loopback; the actual Funnel TCP routing via tailscaled and real-world DNS resolution require live infrastructure.

#### 2. tailscale serve status empty after each teardown trigger (M-35)

**Test:** On a live Tailscale node, enable Funnel for a session, then exercise each of the four production teardown paths one at a time: toggle-off, web-share-off, session natural end, daemon clean stop. After each, run `tailscale serve status`.

**Expected:** `tailscale serve status` shows empty config after each trigger. No lingering TCP proxy entry or `AllowFunnel` annotation remains.

**Why human:** `TestFunnelTeardown_AllTriggers` (5 sub-tests) verifies via `daemonFakeFunnelClient.IsFunnelOn()` that SetServeConfig is called with the cleared config. Whether the real tailscaled clears the serve status requires a live tailscaled process.

#### 3. Fallback-mode web-share unaffected with tailscaled stopped (M-36)

**Test:** With tailscaled stopped (or Tailscale not installed), establish a web-share session and confirm: session accessible on local tailnet URL, `POST /sessions/{id}/funnel {enabled:true}` returns an error (not panic), existing guest connection unaffected, `funnelActive` remains false in GET /sessions.

**Expected:** Web-share works normally. Funnel enable returns a clear error. No disruption to existing web-share.

**Why human:** `TestEnableFunnel_FallbackModeSafe` proves the error guard via fake StatusWithoutPeers. Actual service continuity requires a live environment test.

---

## Summary

Phase 165 gap-closure (165-04) is fully verified in the codebase.

**What changed (165-04):**
- `server.go:632` — EnableFunnel Proxy target `https://localhost:<port>` → `https+insecure://<ws.config.BindIP>:<port>`. Closes external-guest HTTP 502 (FNL-03). Rationale comment at lines 615-631 (Option A, same-host hop, cert-hostname skip safe because hop never leaves host).
- `api.go:742` — `handleDeleteSession` calls `a.runSessionExitCleanup(id)` synchronously after `KillSession`. Closes Funnel-exposure-after-kill and stale ref-count regression (FNL-05 kill path). WHY comment at lines 727-741.
- `internal/webserver/funnel_test.go:286` — `TestEnableFunnel_ProxyTargetReachable` (FNL-03 CI guard): asserts exact proxy string + real HTTPS reachability. Old `localhost` host fails the string assertion.
- `internal/daemon/funnel_test.go:678` — `TestFunnelTeardown_KillPath` (FNL-05 CI guard): 2 sub-cases — single-session DELETE clears fake config, two-session kill-A-leaves-B ref-count guard.
- `TESTING.md` — Section 2 gap-closure note, Section 4 FNL-03/FNL-05 rows extended, Section 5 M-34 updated (expected PASS live, automated guard cited).

**All 7 FNL requirements are SATISFIED.** The 3 human-verification items (M-34/M-35/M-36) require a real Tailscale Funnel-granted tailnet and cannot be auto-verified. M-34 is now expected to PASS live (the proxy-target fix closes the 502 root cause that made it fail).

---

_Verified: 2026-06-30T20:15:00Z_
_Verifier: Claude (gsd-verifier)_
_Re-verification after: 165-04 gap-closure (FNL-03 502 + FNL-05 kill-path teardown)_

---

## Live UAT Addendum — 2026-06-30 (post-165-05, authoritative for the human-verification items)

Live UAT on a real Funnel-granted tailnet (`kens-personal-macbook-air.tail46d69a.ts.net`) found that the 165-04 `https+insecure://<bindIP>:443` proxy target was itself insufficient end-to-end, and a follow-on gap-closure (**165-05**) closed it for real. All live-only items now PASS:

- **M-34 — external-tailnet 200: ✅ PASS.** Off-tailnet device (no Tailscale) `curl -L` to the Funnel URL → **HTTP 200** on `/app/` (followed the /sessions→/app redirect), ~0.44s, against a production build (`wails build -tags wailsassets`). 502/hang gone.
  - Root cause the unit tests missed (twice): hop 2 (tailscaled→AgentHub) needs a TLS handshake because the listener is HTTPS-only with an SNI-driven cert. `https+insecure://<bindIP>` (165-04) → tailscaled sends SNI=IP → no cert for an IP literal → TLS internal_error → 502. `https+insecure://<FQDN>` → public DNS resolves the FQDN to the Funnel ingress → Funnel→ingress→Funnel loop → hang. **165-05 fix:** AgentHub adds a plain-HTTP loopback listener (127.0.0.1, ephemeral) in startTailscale serving the same mux; `EnableFunnel` proxies hop 2 to `http://127.0.0.1:<loopbackPort>` — no TLS/SNI/cert on hop 2 (tailscaled owns the only public TLS on hop 1). Loopback plaintext is safe because it never leaves the host (co-location; documented in code + threat model T-165-18).
  - BUILD NOTE: `/app/` serves only from a production build's embedded SPA; `wails dev`/`go build` (no `wailsassets` tag) returns 503 "app bundle not configured" (server.go:252-257) — expected, not a Funnel bug.
- **M-35 — serve config empty after every teardown trigger: ✅ PASS (a/b/c/d).** Funnel-off, web-share-off, explicit-kill (DELETE — the 165-04 GAP 2 kill-path), and daemon-stop each emptied `tailscale serve status` immediately.
- **M-36 — fallback mode (Tailscale stopped): ✅ PASS.** AgentHub auto-fell-back to local-mode web-share (`https://<LAN-IP>:7443`, Basic Auth: correct pw → 302, wrong pw → 401). Enabling Funnel failed CLOSED (HTTP 400, no serve config written, local web-share unaffected). The 400 body is `funnel: loopback listener not started` (the 165-05 loopback nil-guard; `startLocal` makes no loopback listener; `CheckFunnelAccess` passes on cached node capability while tailscaled is stopped). Behavior is correct/fail-closed; user-facing wording + disabling the toggle in fallback is Phase 166 (Share modal) work.

**Verdict: Phase 165 (funnel-backend) goal achieved and verified live — Funnel reaches an external internet guest end-to-end, tears down on every trigger, and fails closed in fallback. FNL-01..07 closed.**
