---
phase: 4
slug: web-serving-tls-auth
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-18
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `net/http/httptest` |
| **Config file** | none — Go tests use `go test ./...` |
| **Quick run command** | `go test ./internal/webserver/... -timeout 30s` |
| **Full suite command** | `go test ./... -timeout 60s` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... -timeout 30s`
- **After every plan wave:** Run `go test ./... -timeout 60s`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | WEB-02 | unit | `go test ./internal/webserver/... -run TestTLSCertGeneration` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | WEB-03 | unit | `go test ./internal/webserver/... -run TestCAExport` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | WEB-04 | unit | `go test ./internal/webserver/... -run TestDashboardAuth` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | WEB-05 | unit | `go test ./internal/webserver/... -run TestTokenAuth` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 1 | NET-01 | unit | `go test ./internal/webserver/... -run TestBind` | ❌ W0 | ⬜ pending |
| 04-03-02 | 03 | 1 | NET-02 | unit | `go test ./internal/webserver/... -run TestTailscaleDetection` | ❌ W0 | ⬜ pending |
| 04-03-03 | 03 | 1 | NET-03 | unit | `go test ./internal/webserver/... -run TestInterfaceList` | ❌ W0 | ⬜ pending |
| 04-04-01 | 04 | 2 | WEB-01 | unit | `go test ./internal/webserver/... -run TestToggleWebServing` | ❌ W0 | ⬜ pending |
| 04-04-02 | 04 | 2 | WEB-06 | integration | `go test ./internal/webserver/... -run TestWebSocketRelay` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/tls_test.go` — stubs for WEB-02, WEB-03
- [ ] `internal/webserver/auth_test.go` — stubs for WEB-04, WEB-05
- [ ] `internal/webserver/network_test.go` — stubs for NET-01, NET-02, NET-03
- [ ] `internal/webserver/server_test.go` — stubs for WEB-01, WEB-06

*Existing infrastructure covers Go testing framework — no new framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CA cert installation in OS trust store | WEB-03 | Requires OS-level keychain/trust store interaction | Export CA cert, follow in-app instructions, verify browser trusts HTTPS URL |
| Browser loads xterm.js terminal page | WEB-06 | Requires real browser rendering + CDN fetch | Open HTTPS URL in browser, verify terminal renders and accepts input |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
