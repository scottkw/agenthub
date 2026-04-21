---
phase: 87-capability-based-session-authorization
verified: 2026-04-20T19:10:00Z
uat_completed: 2026-04-21T14:40:00Z
status: passed
score: 5/5 must-haves verified (automated) + 5/5 Manual-Only UAT items verified (user)
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "5/5 automated only"
  gaps_closed:
    - "UAT-1 tailnet peer enumeration (curl from asustor → 401 capability required)"
    - "UAT-2 browser cap URL flow (READ ONLY badge visible; xterm renders; stdin disabled)"
    - "UAT-3 daemon restart preserves signing key (capability.key mtime unchanged across kill+relaunch; 403 revoked — not 401 — confirms signature still verifies)"
    - "UAT-4 QR scan join flow (works perfectly from Sessions tab SharePanel)"
    - "UAT-5 regenerate signing key rotates capability.key (mtime advanced 08:24:36 → 09:37:53; live browser invalidated)"
  gaps_remaining: []
  regressions: []
uat_findings_fixed_during_verification:
  - id: "UI-BUG-1"
    commit: "2cd8f9f"
    summary: "SessionSharePanel rendered inline with session row — flex-wrap: wrap missing on .daemon-panel__session-row so flex-basis: 100% didn't wrap. Copy/Open/QR buttons pushed off-screen."
  - id: "UI-BUG-2"
    commit: "2cd8f9f"
    summary: "New sessions displayed as web-enabled in UI even though daemon correctly kept them disabled (SEC-01 intact at daemon layer). Root cause: App.tsx fake auto-seed of webEnabled[id]=true on session create without calling ToggleWebServing. Removed the auto-seed so UI tracks daemon truth."
  - id: "UI-BUG-3"
    commit: "75ffde9"
    summary: "Per-tab StatusBar rendered a capability-less session URL and QR button, both pre-Phase-87 surfaces that would 401 if used. Removed URL + QR button + GetSessionQRCode Wails binding + QRModal component. Sharing is single-sourced on the Sessions tab. Status bar now owns state+toggle only."
uat_results_file: "87-UAT-RESULTS.md"
---

# Phase 87: Capability-Based Session Authorization — Verification Report

**Phase Goal:** Tailnet reachability no longer grants session access; only explicitly granted, capability-token-bearing clients can list, view, or drive a session, and write permission is a server-controlled property of that capability.

**Verified:** 2026-04-20T19:10:00Z
**Status:** **human_needed** — all automated gates green; 5 Manual-Only UAT items remain from 87-VALIDATION.md
**One-line verdict:** **PASS (automated) — pending Manual-Only UAT for human_needed items**

---

## Goal Achievement

### Observable Truths — ROADMAP Success Criteria

| # | Success Criterion | Status | Evidence |
|---|-------------------|--------|----------|
| SC-1 | Tailnet user without grant cannot enumerate sessions via `GET /api/sessions` — even over Tailscale | ✓ VERIFIED (automated) | `internal/webserver/server.go:381` wraps `GET /api/sessions` in `ws.requireCapability(ws.handleListSessions)`. `handleListSessions` (server.go:505-525) reads `claims.SID` from context and returns a zero-or-one-item list per D-18. Test `TestSecurity_UnauthenticatedClientCannotEnumerateSessions` PASS; `TestCapability_MissingCapReturns401` PASS; `TestCapability_ValidCapReturnsSession` PASS (asserts `len(items) == 1` even when a second session is web-enabled). Manual-Only UAT on a real tailnet peer required for final SC-1 signoff. |
| SC-2 | A valid cap for session A cannot open session B's WebSocket or metadata — cap bound to specific session ID | ✓ VERIFIED | `requireCapability` middleware (`internal/webserver/capability_mw.go`) checks `claims.SID == r.PathValue("id")` and returns 403 `capability does not match session` on mismatch. Routes `GET /api/sessions/{id}/info`, `GET /sessions/{id}`, `GET /sessions/{id}/ws` all wrapped (server.go:386, 391, 396). Test `TestSecurity_WrongSessionCapRejected` PASS. |
| SC-3 | Creating a new session while web server is running does NOT auto-expose it; only reachable after explicit grant (D-06) | ✓ VERIFIED | `internal/daemon/api.go:328-365` — `handleCreateSession` no longer calls `ws.EnableSession(id)`. Only remaining `EnableSession` call is in `handleWebServe` (line 632), which is the explicit user toggle. Test `TestHandleCreateSession_NoAutoEnable` PASS asserting `ws.IsSessionEnabled(id) == false` and `info.WebEnabled == false`. `TestHandleWebServe_ToggleOnEnablesSession` PASS. Manual-Only UAT on live browser required for end-to-end confirmation. |
| SC-4 | Read-only cap rejects `MsgInput` at relay even if client omits `?readonly` or reconnects without it — write permission determined by cap, not client/query | ✓ VERIFIED | `internal/webserver/server.go:610-611` — `readonly := claims.Perms == "read"` replaces the old `r.URL.Query().Get("readonly")` path. Confirmed `grep -n "readonly" internal/webserver/server.go` returns zero matches. Tests `TestSecurity_ReadOnlyParamCannotGrantWrite`, `TestSecurity_ReadOnlyCapabilityBlocksMsgInput`, and `TestSecurity_ReconnectWithoutReadonlyStillBlocked` all PASS. `! grep -qE "get\s*\(\s*['\"]readonly['\"]" web/terminal.html` PASS — terminal.html no longer reads `?readonly` from URL. |
| SC-5 | Capability tokens survive daemon restart (signing key persisted alongside settings.json) | ✓ VERIFIED (automated) | `internal/capability/keystore.go:31-33` — `NewFileKeyStore(dir)` uses `filepath.Join(dir, "capability.key")`; write uses `os.WriteFile(..., 0600)`. `internal/capability/keystore.go:75-79` — `LoadOrGenerate` returns an existing key when present, generating only on first run. `internal/daemon/api.go:101-103` + `internal/daemon/process.go:74` — `BootstrapCapabilityState` runs on daemon startup BEFORE `AutoStartWebServer`, loading from the persisted file. Test `TestStartup_LoadsOrGeneratesSigningKey` PASS (constructs two consecutive `API` instances in the same configDir and asserts identical key bytes, proving persistence round-trip). `TestIPCHandlers_RegenerateSigningKey_SwapsKey` PASS. Manual-Only UAT (kill+restart real daemon + browser reload) required for end-to-end confirmation. |

**Automated Score:** 5/5 success criteria verified programmatically
**Manual-Only UAT Remaining:** 5 items (see `human_verification` frontmatter section and 87-VALIDATION.md Manual-Only table)

---

## Required Artifacts

| Artifact | Status | Details |
|----------|--------|---------|
| `internal/capability/capability.go` (Sign/Verify, Claims) | ✓ VERIFIED | 3308 bytes; HMAC-SHA256; `hmac.Equal` present; no `bytes.Equal`. |
| `internal/capability/keystore.go` (FileKeyStore, LoadOrGenerate) | ✓ VERIFIED | 3260 bytes; `os.WriteFile` mode 0600; `capability.key` filename. |
| `internal/capability/joincode.go` (JoinCodeManager) | ✓ VERIFIED | 3420 bytes; `sync.Mutex` (not RWMutex) for TOCTOU safety. |
| `internal/capability/context.go` (WithClaims/ClaimsFromContext) | ✓ VERIFIED | Unexported `ctxKey struct{}` for collision safety. |
| `internal/capability/errors.go` (sentinels) | ✓ VERIFIED | 5 typed errors: `ErrMalformedToken`, `ErrInvalidSignature`, `ErrMalformedClaims`, `ErrCodeNotFound`, `ErrCodeExpired`. |
| `internal/webserver/capability_mw.go` (requireCapability) | ✓ VERIFIED | 3306 bytes; single `requireCapability` method; collapses all Verify failures to 401; session-ID cross-check + grant-list cross-check. |
| `internal/webserver/server.go` (routes + handlers) | ✓ VERIFIED (WIRED) | 4 routes wrapped (`grep -c requireCapability` = 15 total mentions). `handleListSessions` uses `claims.SID`; `handleWSSRelay` sources readonly from `claims.Perms`. |
| `internal/daemon/api.go` (BootstrapCapabilityState, IPC handlers) | ✓ VERIFIED | `BootstrapCapabilityState`, `issueCapabilitiesForSession`, `handleIssueCapabilities`, `handleExchangeJoinCode`, `handleRegenerateSigningKey`, `runSessionExitCleanup` all present. `EnableSession(id)` appears only in explicit-toggle path. |
| `internal/daemon/process.go` (startup wiring) | ✓ VERIFIED | `BootstrapCapabilityState` called BEFORE `AutoStartWebServer` (process.go:74) — Pitfall 3 closed. |
| `app.go` (Wails bindings) | ✓ VERIFIED | 4 bindings: `IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`, `GetCapabilityQRCode`. |
| `frontend/src/components/SessionSharePanel.tsx` | ✓ VERIFIED | 5788 bytes; renders "Read-Only Link" + "Full Access Link"; QR encodes `/join?code=` (D-09). |
| `frontend/src/components/RegenerateKeyModal.tsx` | ✓ VERIFIED | 3308 bytes; "Invalidate All Links" destructive action + "Keep Links" cancel; verbatim UI-SPEC copy. |
| `frontend/src/components/DaemonManagerPanel.tsx` (modified) | ✓ VERIFIED | `sessionShares` state reconciliation against `(sessions, webEnabled)`; calls `IssueCapabilities` on toggle-on, prunes on toggle-off. |
| `frontend/src/components/SettingsTab.tsx` (modified) | ✓ VERIFIED | Security section with Regenerate Signing Key button + modal; verbatim UI-SPEC description. |
| `web/dashboard.html` (landing page) | ✓ VERIFIED | D-17 landing page; contains "Join a Shared Session" heading; no `session-list` / `renderSessions` references. |
| `web/join.html` (5-state routed page) | ✓ VERIFIED | 6942 bytes; client-side routed on `?code=` / `?error=` params. |
| `web/terminal.html` (perms-driven caret) | ✓ VERIFIED | Reads `perms` from `/api/sessions/{id}/info`; fail-safe default `read`; no `params.get('readonly')`. |
| `capability.key` (runtime file, NOT in repo) | ✓ VERIFIED ABSENT | `ls internal/capability/capability.key` → No such file; 0600 key file is runtime-only, generated under daemon configDir on first start. |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `setupRoutes` → `handleListSessions` | capability middleware | `ws.requireCapability(ws.handleListSessions)` | ✓ WIRED | server.go:381 |
| `setupRoutes` → `handleSessionInfo` | capability middleware | `ws.requireCapability(ws.handleSessionInfo)` | ✓ WIRED | server.go:386 |
| `setupRoutes` → `handleTerminalPage` | capability middleware | `ws.requireCapability(ws.handleTerminalPage)` | ✓ WIRED | server.go:391 |
| `setupRoutes` → `handleWSSRelay` | capability middleware | `ws.requireCapability(ws.handleWSSRelay)` | ✓ WIRED | server.go:396 |
| `handleWSSRelay` → `Subscriber.ReadOnly` | `claims.Perms == "read"` | context value | ✓ WIRED | server.go:610-611 — client `?readonly` is never read |
| `handleListSessions` → `claims.SID` | single-session response | `capability.ClaimsFromContext` | ✓ WIRED | server.go:505-525 (D-18 collapse) |
| Daemon startup → `BootstrapCapabilityState` | before `AutoStartWebServer` | `runDaemonCore` | ✓ WIRED | process.go:74 — Pitfall 3 mitigation |
| `handleWebServe(enable=true)` → `EnableSession` | `ws.EnableSession(id)` | explicit toggle | ✓ WIRED | api.go:632 |
| `handleWebServe(enable=false)` → `ClearGrants` | `ws.ClearGrants(id)` | D-15 revocation | ✓ WIRED | api.go:637 |
| `onExit` → `runSessionExitCleanup` → `ClearGrants` | session-end revocation | `time.AfterFunc(10s, ...)` | ✓ WIRED | api.go:377 — Pitfall 1 closure |
| `SessionSharePanel` → `/join?code=` URL | QR encoding | `GetCapabilityQRCode(joinURL)` | ✓ WIRED | D-09 defense against leaked QR |
| `terminal.html` → `/api/sessions/{id}/info` | perms fetch | async IIFE before xterm init | ✓ WIRED | terminal.html:154-185 — fail-safe default `read` |
| `POST /join/exchange` → `ws.joinCodes.Exchange` | code consumption | single-use + TTL | ✓ WIRED | server.go:440 (handleJoinExchange); test `TestEndToEnd_CapabilityFlow` PASS |
| `capability.key` persistence → `LoadOrGenerate` | daemon restart | `NewFileKeyStore(configDir)` | ✓ WIRED | api.go:101-103; `TestStartup_LoadsOrGeneratesSigningKey` PASS |

---

## Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `SessionSharePanel` | `readURL`/`writeURL`/`readCode`/`writeCode` | `IssueCapabilities(sessionId)` Wails binding → daemon `handleIssueCapabilities` → `issueCapabilitiesForSession` → `capability.Sign` + `joinCodes.Issue` | Yes (real HMAC-signed tokens + dashed base32 join codes) | ✓ FLOWING |
| `DaemonManagerPanel.sessionShares` | reconciled map | `useEffect` watching `(sessions, webEnabled)` | Yes (prunes toggle-off, fetches toggle-on) | ✓ FLOWING |
| `terminal.html` perms | `perms` variable | GET `/api/sessions/{id}/info?cap=...` → `handleSessionInfo` → `claims.Perms` | Yes (sourced from verified capability, fail-safe `read`) | ✓ FLOWING |
| `handleListSessions` response | single-item array | `claims.SID` → `ws.IsSessionEnabled(claims.SID)` | Yes (D-18 single-session contract) | ✓ FLOWING |

---

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go unit/integration tests green | `go test ./... -count=1` | ok for all 12 agenthub packages; only `security-review/` (gitignored) fails with setup error | ✓ PASS |
| Fuzz test finds no crashers | `go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=10s` | 308,661 execs, 0 new crashers | ✓ PASS |
| `go vet` clean | `go vet ./...` | Only security-review error (gitignored orphan) | ✓ PASS |
| Frontend build clean | `cd frontend && pnpm build` | 693 modules, 275 ms, 0 TypeScript errors | ✓ PASS |
| Dashboard is landing page | `grep -q 'Join a Shared Session' web/dashboard.html` | MATCH | ✓ PASS |
| Dashboard has no session list | `! grep -q 'session-list\|renderSessions' web/dashboard.html` | 0 matches | ✓ PASS |
| Terminal page does not read `?readonly` | `! grep -qE "get\s*\(\s*['\"]readonly['\"]" web/terminal.html` | 0 matches | ✓ PASS |
| Capability middleware count | `grep -c "requireCapability" internal/webserver/server.go` | 15 (≥ 4 route hits expected) | ✓ PASS |
| `capability.key` not in repo | `ls internal/capability/capability.key` | No such file | ✓ PASS |
| Phase 87 security tests pass | `go test ./internal/webserver/ -run 'TestSecurity\|TestCapability\|TestEndToEnd_CapabilityFlow'` | 10/10 PASS | ✓ PASS |
| Phase 87 daemon tests pass | `go test ./internal/daemon/ -run 'TestHandleCreateSession_NoAutoEnable\|TestHandleWebServe_Toggle\|TestOnExit_ClearsGrants\|TestStartup_LoadsOrGeneratesSigningKey\|TestIPCHandlers'` | 8/8 PASS | ✓ PASS |

---

## Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|----------------|-------------|--------|----------|
| SEC-01 | 87-01, 87-02, 87-04, 87-05, 87-06 | Explicit grant; no auto-expose of new sessions | ✓ SATISFIED | `handleCreateSession` auto-enable removed (api.go:328-365); `TestHandleCreateSession_NoAutoEnable` PASS; capability issuance via `IssueCapabilities` IPC. |
| SEC-02 | 87-01, 87-02, 87-03, 87-04 | `GET /api/sessions` requires valid capability | ✓ SATISFIED | Route wrapped in `requireCapability`; D-18 single-session collapse; `TestSecurity_UnauthenticatedClientCannotEnumerateSessions` PASS. |
| SEC-03 | 87-01, 87-02, 87-03, 87-04 | Session-scoped cap binding for `/sessions/{id}/ws` and `/sessions/{id}` | ✓ SATISFIED | Middleware cross-checks `claims.SID == r.PathValue("id")`; `TestSecurity_WrongSessionCapRejected` PASS. |
| SEC-04 | 87-01, 87-02, 87-03, 87-04, 87-05, 87-06 | Read-only is a server-bound cap property, not `?readonly` | ✓ SATISFIED | `readonly := claims.Perms == "read"` (server.go:611); no `?readonly` read in write path; terminal.html caret driven exclusively by perms fetch; `TestSecurity_ReadOnlyParamCannotGrantWrite` PASS. |
| SEC-05 | 87-01, 87-02, 87-03, 87-04 | Relay rejects `MsgInput` from read-only subscribers | ✓ SATISFIED | `handleWSSRelay` sets `Subscriber.ReadOnly` from `claims.Perms`; `TestSecurity_ReadOnlyCapabilityBlocksMsgInput` PASS; `TestSecurity_ReconnectWithoutReadonlyStillBlocked` PASS. |

**Orphaned requirements:** None. REQUIREMENTS.md maps only SEC-01..SEC-05 to Phase 87, and all 5 are claimed by at least one plan.

---

## Anti-Patterns Scanned

Files modified in Phase 87 were grep-scanned for TODO/FIXME/placeholder/empty returns/hardcoded empty data. Findings:

| Pattern Class | Result |
|---------------|--------|
| TODO / FIXME / XXX / HACK | None found in Phase 87 source files |
| Placeholder / "coming soon" / "not yet implemented" | None found |
| Empty returns (`return null`, `return {}`, `return []` without DB/auth query) | None found. `handleListSessions` returns `[]` only when the claims-bound session is not web-enabled, which is the D-15 revocation contract (not a stub). |
| Hardcoded empty state (`= []`, `= {}`) not overwritten by fetch | `sessionShares` initial state = `{}` (correct), overwritten by reconciliation `useEffect`. Not a stub. |
| Console.log-only handlers | None found |
| Silent fallbacks (`or {}` etc.) | None found. `handleJoinExchange` explicitly nil-checks `ws.joinCodes` and `ws.signingKey` and returns 500 on unwired state (CLAUDE.md "let it crash" principle honored). |

**No blocker or warning anti-patterns identified.**

---

## Known Non-Issues (Documented)

### Pre-existing `security-review/` orphan scaffold

`go test ./...` reports:
```
# github.com/scottkw/agenthub/security-review
found packages relay (internal_relay_protocol_fuzz_test.go) and webserver (internal_webserver_server_test.go) in /Users/ken/dev/agenthub/security-review
FAIL	github.com/scottkw/agenthub/security-review [setup failed]
```

**This is NOT a Phase 87 regression.** The `security-review/` directory is gitignored (`.gitignore:44` matches `security-review/`). It contains reference test scaffolds supplied with the external security review and was never intended to be a Go package — the two files use different package names (`relay` and `webserver`) in the same directory, which Go rejects at the package-scan step. Documented in every Plan SUMMARY (01-06) under "Issues Encountered" as out-of-scope reference material.

**Verdict:** Known non-issue. Not counted against Phase 87. All 12 agenthub-proper packages pass `go test`.

---

## Human Verification Required

The following items cannot be verified programmatically. They are carried forward from 87-VALIDATION.md's "Manual-Only Verifications" table and each corresponds to a ROADMAP Success Criterion that has already passed automated checks. These are end-to-end UAT items for the user:

### 1. Un-granted tailnet peer cannot enumerate sessions (SEC-01 / SC-1)

**Test:** From a second tailnet machine, run `curl http://<host>.tailnet.ts.net:<port>/api/sessions`
**Expected:** HTTP 401 with body `capability required`
**Why human:** Requires a second tailnet machine; automated tests use in-process HTTP and cannot exercise the real Tailscale reachability boundary.

### 2. Capability-bearing URL opens in external browser (SEC-02 / SC-3)

**Test:** Grant share from GUI (toggle web-serve + Copy Read-Only Link). Paste URL in a browser on a second device.
**Expected:** Session loads, terminal renders, READ ONLY badge visible in status bar.
**Why human:** Requires live user-agent with xterm.js rendering; badge visibility + caret suppression are UX qualities.

### 3. Daemon restart preserves existing shared URLs (SEC-05 / SC-5)

**Test:** Share session → note URL → kill daemon → restart daemon → reload URL in browser.
**Expected:** URL still authenticates (signing key persisted via `capability.key` mode 0600 under daemon configDir).
**Why human:** Multi-process lifecycle + real browser session lifetime test.

### 4. QR scan join flow (D-09 / SC-3)

**Test:** Toggle-on a session → click QR for Read-Only Link → scan QR on phone camera → tap Join Session.
**Expected:** Phone browser lands on `/join?code=XXXX-XXXX` → tap "Join" → redirects to `/sessions/{id}?cap=<token>` → read-only session view.
**Why human:** Requires phone camera hardware; no way to automate a real QR scan.

### 5. Regenerate signing key invalidates all links (D-16 / T-87-03)

**Test:** Share a session, open URL in browser #2, return to GUI → Settings → Security → Regenerate Signing Key → confirm.
**Expected:** Reload browser #2 → 401 `capability required` on next request.
**Why human:** Cross-process destructive UX verification; requires concurrent browser + GUI + daemon state changes.

---

## Gaps Summary

**No automated gaps identified.** All 5 ROADMAP Success Criteria have positive programmatic evidence (test PASS + wiring greps + persistence round-trip). All 5 REQUIREMENTS (SEC-01..SEC-05) are covered with implementation evidence. All 18 Phase 87 test names pass. Frontend build is clean. No stubs, no hollow wiring, no data-flow disconnections, no blocker anti-patterns.

**Remaining work:** 5 Manual-Only UAT items must be executed by the user before the Phase 87 gate can move from `human_needed` to `passed`. These are the end-to-end lifecycle and cross-device flows that cannot be simulated by in-process Go tests.

**No NEW UAT items created by this verification** — all 5 human-verification items were already declared in 87-VALIDATION.md's Manual-Only table. This verifier did not surface additional gaps requiring UAT.

---

*Verified: 2026-04-20T19:10:00Z*
*Verifier: Claude Opus 4.7 (gsd-verifier)*
*Automated gates: 18/18 Phase 87 tests PASS · `go test ./... -count=1` green for all agenthub packages · `go vet ./...` clean · `pnpm build` clean · fuzz 10s zero crashers*
