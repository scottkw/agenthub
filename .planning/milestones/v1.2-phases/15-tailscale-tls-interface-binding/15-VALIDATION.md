---
phase: 15
slug: tailscale-tls-interface-binding
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 15 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib) |
| **Config file** | none |
| **Quick run command** | `go test ./internal/webserver/... -count=1` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... -count=1`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 15-01-01 | 01 | 1 | TLS-01 | unit | `go test ./internal/webserver/... -run TestStart_UsesTailscaleCert -count=1` | ❌ W0 | ⬜ pending |
| 15-01-02 | 01 | 1 | TLS-02 | unit | `go test ./internal/webserver/... -run TestBaseURL_UsesFQDN -count=1` | ❌ W0 | ⬜ pending |
| 15-01-03 | 01 | 1 | TLS-03 | unit | `go test . -run TestStartWebServer_NoTailscale -count=1` | ❌ W0 | ⬜ pending |
| 15-01-04 | 01 | 1 | TLS-03 | unit | `go test ./internal/webserver/... -run TestStart_BindsToTailscaleIP -count=1` | ❌ W0 | ⬜ pending |
| 15-01-05 | 01 | 1 | TLS-04 | unit | `go test . -run TestCTDisclosure -count=1` | ❌ W0 | ⬜ pending |
| 15-01-06 | 01 | 1 | TLS-05 | compile-time | `go build ./...` | via impl | ⬜ pending |
| 15-01-07 | 01 | 1 | TLS-05 | unit | `go test ./internal/webserver/... -run TestNewWebServer_NoCertFiles -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/server_test.go` — update `testServer()` helper to use `TLSConfig` override; add `TestBaseURL_UsesFQDN`, `TestStart_BindsToTailscaleIP`, `TestNewWebServer_NoCertFiles`
- [ ] `app_test.go` — add `TestStartWebServer_NoTailscale`, `TestCTDisclosure_*`
- [ ] `internal/webserver/tls_test.go` — move CA/leaf test helpers to test-internal use; `tls.go` itself is deleted so test file must be removed/rewritten

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Browser opens `.ts.net` HTTPS URL without cert warning | TLS-01 | Requires live Tailscale daemon and browser | 1. Start Tailscale, 2. Start web server, 3. Open `https://<fqdn>:<port>` in browser, 4. Verify no cert warning |
| FQDN in QR code resolves correctly | TLS-02 | Requires live Tailscale network | 1. Start server, 2. Check QR code URL uses FQDN not IP |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
