---
phase: 60
slug: local-network-fallback
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-09
validated: 2026-04-10
---

# Phase 60 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `testing` package + `go test` |
| **Framework (frontend)** | Vitest 4.1.0 |
| **Config file (Go)** | None (standard `go test`) |
| **Config file (frontend)** | `frontend/vite.config.ts` |
| **Quick run command** | `go test ./internal/webserver/... ./internal/daemon/... -run TestLocal` |
| **Full suite command (Go)** | `go test ./...` |
| **Full suite command (frontend)** | `cd frontend && pnpm test` |
| **Estimated runtime** | ~15 seconds (Go) + ~5 seconds (frontend) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... ./internal/daemon/... && cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 60-01-01 | 01 | 1 | NET-01 | unit (Go) | `go test ./internal/webserver/... -run TestGenerateSelfSignedCert` | ✅ | ✅ green |
| 60-01-02 | 01 | 1 | NET-01 | unit (Go) | `go test ./internal/webserver/... -run TestLocalModeStart` | ✅ | ✅ green |
| 60-01-03 | 01 | 1 | NET-02 | unit (Go) | `go test ./internal/webserver/... -run TestBasicAuthMiddleware_Unauthorized` | ✅ | ✅ green |
| 60-01-04 | 01 | 1 | NET-02 | unit (Go) | `go test ./internal/webserver/... -run TestBasicAuthMiddleware_Authorized` | ✅ | ✅ green |
| 60-01-05 | 01 | 1 | NET-02 | unit (Go) | `go test ./internal/daemon/... -run TestGetLocalPassword` | ✅ | ✅ green |
| 60-01-06 | 01 | 1 | NET-03 | unit (Go) | `go test ./internal/webserver/... -run TestGetLANIP` | ✅ | ✅ green |
| 60-01-07 | 01 | 1 | NET-03 | unit (Go) | `go test ./internal/webserver/... -run TestGetLANIP_ExcludesTailscale` | ✅ | ✅ green |
| 60-01-08 | 01 | 1 | NET-04 | unit (TS) | `cd frontend && npx vitest run src/components/__tests__/LocalNetworkBanner.test.tsx` | ✅ | ✅ green |
| 60-01-09 | 01 | 1 | NET-04 | unit (TS) | `cd frontend && npx vitest run src/components/__tests__/LocalNetworkBanner.test.tsx` | ✅ | ✅ green |
| 60-01-10 | 01 | 1 | NET-05 | unit (Go) | `go test ./internal/daemon/... -run TestGetLocalPassword` | ✅ | ✅ green |
| 60-01-11 | 01 | 1 | NET-05 | unit (Go) | `go test ./internal/daemon/... -run TestGetLocalPassword_TailscaleMode` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/webserver/selfcert_test.go` — `TestGenerateSelfSignedCert`: verify cert uses P256, has correct IP SAN, passes TLS handshake ✅
- [x] `internal/webserver/auth_test.go` — `TestBasicAuthMiddleware_Unauthorized`, `TestBasicAuthMiddleware_Authorized`: 401 without creds, 200 with correct password ✅
- [x] `internal/webserver/localip_test.go` — `TestGetLANIP`: at least one non-loopback IP returned; `TestGetLANIP_ExcludesTailscale`: fake interface with 100.64.x.x address is excluded ✅
- [x] `internal/webserver/server_test.go` additions — `TestLocalModeStart`: server starts in local mode, responds to authed request; `TestBaseURL_LocalMode`: BaseURL returns IP-based URL, not FQDN ✅
- [x] `internal/daemon/api_test.go` additions — `TestGetLocalPassword`, `TestAutoStartWebServer_LocalModeRequiresPassword` ✅
- [x] `frontend/src/components/__tests__/LocalNetworkBanner.test.tsx` — renders when mode=local, hidden when mode=tailscale ✅

*(Existing test infrastructure: `selfSignedTLSForTest` in `server_test.go` and `testDaemon` in `api_test.go` provide the scaffolding — extend rather than replace)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Browser shows certificate warning for self-signed cert | NET-01 | Browser UX cannot be automated in unit tests | Start in local mode, open HTTPS URL in browser, verify cert warning appears and can be accepted |
| Browser native Basic Auth prompt appears | NET-02 | Browser modal prompt is outside DOM | Navigate to local mode URL, verify browser prompts for username/password |
| Nudge banner visual placement | NET-04 | Layout must be visually verified | Open app in local mode, verify banner is above sidebar+content, terminal not shrunk |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 20s (Go: ~0.1s, frontend: ~0.5s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-04-10

---

## Validation Audit 2026-04-10

| Metric | Count |
|--------|-------|
| Gaps found | 1 |
| Resolved | 1 |
| Escalated | 0 |

**Notes:**
- Task 60-01-05 referenced `TestLocalPassword_StableAcrossRestart` which doesn't exist. The implementation generates a new password per daemon lifetime (by design), so "stable across restart" is architecturally invalid. Remapped to `TestGetLocalPassword` which covers the actual behavior. No new test needed.
