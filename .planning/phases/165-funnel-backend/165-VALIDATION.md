---
phase: 165
slug: funnel-backend
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-30
audited: 2026-06-30
---

# Phase 165 — Validation Strategy

> Per-phase validation contract. Audited retroactively: every FNL requirement has
> automated `go test` coverage; the three inherently live-only behaviors (M-34/M-35/M-36)
> are manual-only and were live-verified PASS on a real Funnel-granted tailnet.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./internal/webserver/... ./internal/daemon/...` |
| **Full suite command** | `go test ./...` |
| **Funnel-only run** | `go test ./internal/webserver/... -run 'TestFunnel\|TestEnableFunnel\|TestWebServerStop_DisablesFunnel\|TestRequireAllowedOrigin_FunnelOrigin\|TestOriginAllowedForWrite_FunnelOrigin' && go test ./internal/daemon/... -run 'TestFunnel\|TestHandleSetSessionFunnel\|TestIssueCapabilities_FunnelURL\|TestExchangeJoinCode_FunnelURL_GateIntact\|TestStartupClearsLingeringFunnel'` |
| **Estimated runtime** | webserver ~10s full / ~0.03s funnel-only · daemon ~25s full / ~4.4s funnel-only |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~25 s (daemon full suite, the slower of the two packages)

---

## Per-Task Verification Map

Mapped by requirement (FNL-01..07). Every requirement resolves to ≥1 automated `go test`
function; CI test seam is the injectable `funnelClient` interface (`internal/webserver/funnel_client.go`),
which lets the suite run without a live `tailscaled` daemon — mirroring the existing
`statusFunc`/`prefsFunc` injection idiom in `tailscale.go`.

| Requirement | Plan(s) | Test Type | File | Automated Test(s) | File Exists | Status |
|-------------|---------|-----------|------|-------------------|-------------|--------|
| FNL-01 | 165-02, 165-03 | unit | `internal/daemon/funnel_test.go` | `TestFunnelSessionsMap`, `TestHandleSetSessionFunnel_Enable`, `TestHandleSetSessionFunnel_DisableTeardown` | ✅ | ✅ green |
| FNL-01 | 165-03 | unit | `app_test.go` | `TestApp_SetSessionFunnel_NilClient`, `TestListSessions_PropagatesFunnelActive` | ✅ | ✅ green |
| FNL-01 | 165-03 | unit | `internal/daemon/client_test.go` | `TestDaemonClient_SetSessionFunnel`, `TestDaemonClient_SetSessionFunnel_ErrorStatus` | ✅ | ✅ green |
| FNL-02 | 165-01 | compile-smoke | `internal/webserver/funnel_test.go` | `TestFunnelClient_CompileSmoke` | ✅ | ✅ green |
| FNL-03 | 165-02, 165-04, 165-05 | unit + reachability | `internal/webserver/funnel_test.go` | `TestEnableFunnel_ProxyTargetReachable` (loopback-HTTP: asserts `http://127.0.0.1:<loopbackPort>`, rejects 165-04 `https+insecure` regression, dials with plain http client) | ✅ | ✅ green |
| FNL-03 | 165-02 | unit | `internal/daemon/funnel_test.go` | `TestIssueCapabilities_FunnelURL`, `TestExchangeJoinCode_FunnelURL_GateIntact` | ✅ | ✅ green |
| FNL-04 | 165-01 | unit | `internal/webserver/funnel_test.go` | `TestRequireAllowedOrigin_FunnelOrigin` (4 sub-tests), `TestOriginAllowedForWrite_FunnelOrigin` | ✅ | ✅ green |
| FNL-05 | 165-01 | unit | `internal/webserver/funnel_test.go` | `TestEnableFunnelCallsSetServeConfig`, `TestWebServerStop_DisablesFunnel`, `TestEnableFunnel_FallbackModeSafe` | ✅ | ✅ green |
| FNL-05 | 165-02, 165-04 | unit | `internal/daemon/funnel_test.go` | `TestFunnelTeardown_AllTriggers` (5 sub-tests), `TestFunnelTeardown_RefCountKeepsSiblingUp`, `TestFunnelTeardown_KillPath` (2 sub-cases: single-kill + ref-count guard), `TestStartupClearsLingeringFunnel` | ✅ | ✅ green |
| FNL-06 | 165-01, 165-02 | unit | `internal/webserver/funnel_test.go` | `TestEnableFunnel_PrereqCheckPreventsSetServeConfig` | ✅ | ✅ green |
| FNL-07 | 165-02 | unit (timer) | `internal/daemon/funnel_test.go` | `TestFunnelAutoExpiry`, `TestFunnelTeardown_AllTriggers/5_expiry_timer` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `funnelClient` interface seam (3 methods) so CI tests run without a live `tailscaled` daemon — implemented in `internal/webserver/funnel_client.go`; production wired to `&ws.lc` in `NewWebServer`; mirrors the `statusFunc`/`prefsFunc` injection pattern in `tailscale.go`.
- [x] Test files registered in TESTING.md per the standing regression-test convention — Suite Manifest (Section 2) notes for 165-01..05; Traceability rows (Section 4) for FNL-01..07; Manual checklist (Section 5) M-34/M-35/M-36. `bash tests/check-traceability-paths.sh` → `OK: all traceability paths exist`.

---

## Manual-Only Verifications

All three are inherently live-only (require a real Tailscale Funnel-granted tailnet + an
off-tailnet machine). Each was **live-verified PASS on 2026-06-30** against
`kens-personal-macbook-air.tail46d69a.ts.net` (see 165-VERIFICATION.md Live UAT Addendum).

| Behavior | Requirement | Why Manual | Test Instructions | Live Result |
|----------|-------------|------------|-------------------|-------------|
| External-tailnet 200 (not 502/403) on the no-port Funnel URL | FNL-03/FNL-04 | Requires a real off-tailnet machine (no Tailscale installed) hitting a live Funnel URL | From a non-tailnet machine, `curl -L` the emitted Funnel share URL `https://<host>.ts.net/sessions/<id>?cap=TOKEN`; confirm 200 (not 502/403) | ✅ PASS — HTTP 200 on `/app/` (~0.44s), production build (`wails build -tags wailsassets`) |
| `tailscale serve status` empty after all teardown triggers | FNL-05 | Asserts live `tailscaled` serve-config state | After each of: toggle-off, web-share-off, explicit kill (DELETE), daemon stop — run `tailscale serve status`; confirm empty | ✅ PASS — a/b/c/d each emptied config immediately |
| Fallback-mode web-share unaffected (Tailscale stopped) | FNL-06 | Requires Tailscale stopped on host | Stop Tailscale, start web-share; confirm local URL works, Funnel-enable fails CLOSED (HTTP 400, no serve config), `funnelActive` stays false | ✅ PASS — local web-share unaffected; enable failed closed (400) |

*Automated companions (loopback / fake-based) for these behaviors: `TestEnableFunnel_ProxyTargetReachable` (FNL-03 shape+reachability), `TestFunnelTeardown_AllTriggers`/`TestFunnelTeardown_KillPath` (FNL-05 fake-verified), `TestEnableFunnel_FallbackModeSafe` + `TestEnableFunnel_PrereqCheckPreventsSetServeConfig` (FNL-06).*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none remained)
- [x] No watch-mode flags
- [x] Feedback latency < 25 s (daemon full suite)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** nyquist-compliant — all 7 FNL requirements have automated `go test` coverage (18 funnel test functions, all green); 3 inherently live-only behaviors are manual-only and live-verified PASS.

---

## Validation Audit 2026-06-30

| Metric | Count |
|--------|-------|
| Requirements audited | 7 (FNL-01..07) |
| Automated test functions | 18 (8 webserver + 10 daemon) + Wails/client coverage in `app_test.go` / `client_test.go` |
| Gaps found | 0 |
| Resolved | 0 (none needed — coverage pre-existed; auditor not spawned) |
| Escalated to manual-only | 0 (M-34/M-35/M-36 were already manual-only by design) |

**Note:** No `gsd-nyquist-auditor` was spawned — gap analysis found zero MISSING/PARTIAL
requirements. This audit only populated the VALIDATION.md draft skeleton (left unfilled at
planning time) to reflect the comprehensive coverage that already existed and is green.
