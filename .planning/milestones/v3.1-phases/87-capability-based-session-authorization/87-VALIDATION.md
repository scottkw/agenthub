---
phase: 87
slug: capability-based-session-authorization
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-20
updated: 2026-04-20
---

# Phase 87 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution of capability-based session authorization.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (backend). No new frontend test suite is added by this phase — UI is validated via Manual-Only browser flows + backend acceptance-criteria greps on the HTML files. |
| **Config file** | Go: repo-root `go.mod` |
| **Quick run command** | `go test ./internal/capability/... ./internal/webserver/... ./internal/daemon/... -count=1` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && pnpm build` |
| **Estimated runtime** | ~30 seconds quick / ~90 seconds full (fuzz smoke excluded) |

---

## Sampling Rate

- **After every task commit:** Run quick command (capability + webserver + daemon packages)
- **After every plan wave:** Run full suite (`go test ./... && pnpm build`)
- **Before `/gsd-verify-work`:** Full suite must be green + Manual-Only browser tests executed per Success Criteria 1
- **Max feedback latency:** 30 seconds for quick, 90 seconds for full

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 87-01-01 | 01 | 0 | SEC-01..05 | T-87-03,06,08 | Wave 0 test skeletons for capability package (Sign/Verify/Keystore/JoinCode/FuzzVerify) compile build-tag-gated | scaffold (unit) | `ls internal/capability/{capability,keystore,joincode,capability_fuzz}_test.go && go test ./internal/capability/... -count=1` exits 0 with "no matching test files" | ✅ creates | ⬜ pending |
| 87-01-02 | 01 | 0 | SEC-02..05 | T-87-01,02,04,07 | Wave 0 webserver capability test skeletons + relocated helpers compile build-tag-gated | scaffold (integration) | `ls internal/webserver/capability_test{,_helpers}.go && go test ./internal/webserver/... -count=1` | ✅ creates | ⬜ pending |
| 87-02-01 | 02 | 1 | SEC-01..05 | T-87-03,08 | capability.Sign / Verify / WithClaims / ClaimsFromContext + sentinel errors; FuzzVerify 30s | unit + fuzz | `go test ./internal/capability/ -run 'TestSign\|TestVerify\|TestClaims' -count=1 && go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s` | ✅ (87-01-01) | ⬜ pending |
| 87-02-02 | 02 | 1 | SEC-01,06(avail) | T-87-06,07 | FileKeyStore round-trip (0600); JoinCodeManager single-use, 5-min TTL, TOCTOU-safe | unit | `go test ./internal/capability/ -count=1 -v` (all 21 unit tests PASS) | ✅ (87-01-01) | ⬜ pending |
| 87-03-01 | 03 | 2 | SEC-02..05 | — (scaffold) | WebServer struct gains signingKey + grants + joinCodes; SetSigningKey/SetJoinCodes/AddGrant/ClearGrants/isGrantActive/currentSigningKey | unit (compile gate) | `go build ./internal/webserver/... && go test ./internal/webserver/ -count=1` | ✅ existing | ⬜ pending |
| 87-03-02 | 03 | 2 | SEC-02,03,04,05 | T-87-01,02,04,07,08 | requireCapability middleware; handleListSessions single-session (D-18); handleWSSRelay reads readonly from claims.Perms (D-24); 9 Wave 0 SEC tests GREEN | integration | `go test ./internal/webserver/ -count=1 -v -run 'TestCapability\|TestSecurity'` | ✅ (87-01-02) | ⬜ pending |
| 87-04-01 | 04 | 3 | SEC-01 | T-87-05,06,07 | Auto-enable removed; onExit calls ClearGrants (Pitfall 1); startup LoadOrGenerate + SetSigningKey + SetJoinCodes; toggle-off ClearGrants | integration | `go test ./internal/daemon/ -count=1 -v -run 'TestHandleCreateSession_NoAutoEnable\|TestHandleWebServe_Toggle\|TestOnExit_ClearsGrants\|TestStartup_LoadsOrGeneratesSigningKey'` | ✅ creates `internal/daemon/api_test.go` rows | ⬜ pending |
| 87-04-02 | 04 | 3 | SEC-01..05 | T-87-03,08 | IssueCapabilities / ExchangeJoinCode (410/404/500) / RegenerateSigningKey IPC handlers + typed client methods + Wails bindings | integration | `go test ./internal/daemon/ -count=1 -v -run 'TestIPCHandlers'` | ✅ (87-04-01) | ⬜ pending |
| 87-05-01 | 05 | 4 | SEC-01,04 | T-87-05,07 | SessionSharePanel renders only when sessionShares populated; QR encodes join-code URL (D-09); Copy/Open/QR buttons functional | build + grep | `cd frontend && pnpm build && grep -q "Read-Only Link\|Full Access Link\|GetCapabilityQRCode\|/join?code=" frontend/src/components/SessionSharePanel.tsx` | ✅ creates | ⬜ pending |
| 87-05-02 | 05 | 4 | D-16 | T-87-03 (rotation) | RegenerateKeyModal + Settings Security section; verbatim Copywriting Contract strings | build + grep | `cd frontend && pnpm build && grep -q "Regenerate Signing Key?\|Invalidate All Links\|Keep Links\|Invalidating…" frontend/src/components/RegenerateKeyModal.tsx` | ✅ creates | ⬜ pending |
| 87-06-01 | 06 | 5 | SEC-01 (D-17) | T-87-01,03,07 | Dashboard landing page (no session list); 5-state /join page; POST /join/exchange 303→/sessions/{id}?cap= (410/404/session-gone redirects) | integration | `go test ./internal/webserver/ -count=1 && grep -q 'Join a Shared Session' web/dashboard.html && ! grep -q 'session-list\|renderSessions' web/dashboard.html` | ✅ existing + creates web/join.html | ⬜ pending |
| 87-06-02 | 06 | 5 | SEC-04 (D-23) | T-87-04 | terminal.html reads perms from /api/sessions/{id}/info (fail-safe default read); READ ONLY badge; no params.get('readonly'); E2E integration test | integration (E2E) | `go test ./internal/webserver/ -count=1 -v -run TestEndToEnd_CapabilityFlow && ! grep -qE "get\\s*\\(\\s*['\\\"]readonly['\\\"]" web/terminal.html` | ✅ (87-01-02 base) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Test infrastructure that must exist before Wave 1 tasks run. All items are created by Plan 01 (tasks 87-01-01 and 87-01-02).

- [x] `internal/capability/capability_test.go` — Sign/Verify round-trip, tamper detection, wrong-key rejection, malformed input, ConstantTimeComparison, Claims context round-trip (build-tag-gated until Plan 02)
- [x] `internal/capability/keystore_test.go` — FileKeyStore round-trip (0600), missing-file-is-not-error, corrupt-length, GenerateKey length, LoadOrGenerate round-trip (build-tag-gated)
- [x] `internal/capability/joincode_test.go` — Issue format regex, single-use Exchange, double-use rejection, TTL expiry, TOCTOU atomicity, unknown-code rejection (build-tag-gated)
- [x] `internal/capability/capability_fuzz_test.go` — FuzzVerify harness with one seed token (build-tag-gated)
- [x] `internal/webserver/capability_test_helpers.go` — selfSignedTLSForTest, testServer, testServerWithHub, dialWebServerWS, readPipeWithTimeout (build-tag-gated until Plan 03)
- [x] `internal/webserver/capability_test.go` — 9 SEC test skeletons: UnauthenticatedClientCannotEnumerateSessions, WrongSessionCapRejected, ReadOnlyParamCannotGrantWrite, ReadOnlyCapabilityBlocksMsgInput, ReconnectWithoutReadonlyStillBlocked, MissingCapReturns401, InvalidSignatureReturns401, RevokedGrantReturns403, ValidCapReturnsSession (build-tag-gated)
- [x] `internal/daemon/api_test.go` — additions live inside Plan 04 task 87-04-01 (NOT a Wave 0 file). The daemon package already has an existing api_test.go; Plan 04 appends 5 new TestHandle* / TestOnExit_* / TestStartup_* tests and 3 TestIPCHandlers_* tests. This file is therefore marked ✅ existing in the map above.

Notes on scope clarifications:
- No Go file lives under `internal/web/` — the correct package path is `internal/webserver/`. Previous VALIDATION scaffolding that referenced `internal/web/` or `internal/session/` was stale.
- No frontend Vitest suite is created by this phase. Frontend coverage is: `pnpm build` clean + literal-string greps on component files (acceptance_criteria in each Plan 05 / Plan 06 task) + Manual-Only browser flows below.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Un-granted tailnet peer cannot enumerate sessions | SEC-01 / SC-1 | Requires second tailnet machine; automated tests use in-process HTTP | From peer machine: `curl http://<host>.tailnet.ts.net:<port>/api/sessions` must return 401 |
| Capability-bearing URL opens in external browser | SEC-02 / SC-3 | End-to-end manual browser test | Grant share from GUI (toggle web-serve + Copy Read-Only Link), paste URL in browser on second device, confirm session loads and READ ONLY badge is visible |
| Daemon restart preserves existing shared URLs | SEC-05 / SC-5 | Multi-process lifecycle test | Share session → note URL → kill+restart daemon → reload URL → must still authenticate (signing key persists via capability.key 0600) |
| QR scan join flow | D-09 / SC-3 | Requires phone camera | From a toggle-on session, click QR for Read-Only Link → scan QR on phone → phone browser lands on /join?code=XXXX-XXXX → tap "Join Session" → confirm redirect to /sessions/{id}?cap=<token> and read-only session view |
| Regenerate signing key invalidates all links | D-16 / T-87-03 | End-to-end destructive action | Share a session, note URL, open in second browser → return to GUI → Settings → Security → Regenerate Signing Key → confirm → reload second browser → must show "capability required" 401 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (see per-task map above)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (every task has one)
- [x] Wave 0 covers all MISSING references (Plan 01 creates all 6 test-scaffold files; daemon additions live inside Plan 04)
- [x] No watch-mode flags (`-watch`, `--watchAll`, etc.)
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter
- [x] `wave_0_complete: true` set in frontmatter (Plan 01 = Wave 0)

**Approval:** approved 2026-04-20
