---
phase: 14
slug: tailscale-health-check-infrastructure
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none — `go test ./...` from module root |
| **Quick run command** | `go test ./internal/webserver/ -run TestCheckHealth -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/ ./`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | HEALTH-01 | unit | `go test ./internal/webserver/ -run TestCheckHealth_NotRunning -v` | ❌ W0 | ⬜ pending |
| 14-01-02 | 01 | 1 | HEALTH-02 | unit | `go test ./internal/webserver/ -run TestCheckHealth_BackendState -v` | ❌ W0 | ⬜ pending |
| 14-01-03 | 01 | 1 | HEALTH-03 | unit | `go test ./internal/webserver/ -run TestCheckHealth_CertDomains -v` | ❌ W0 | ⬜ pending |
| 14-02-01 | 02 | 1 | HEALTH-06 | unit | `go test . -run TestGetTailscaleStatus -v` | ❌ W0 | ⬜ pending |
| 14-02-02 | 02 | 1 | HEALTH-06 | unit | `go test . -run TestHealthPollerStops -v -race` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/tailscale_test.go` — stubs for HEALTH-01, HEALTH-02, HEALTH-03 with fake `statusFn`
- [ ] `app_test.go` additions — stubs for HEALTH-06 (`TestGetTailscaleStatus`, `TestHealthPollerStops`)

*No new test framework needed — existing `go test` infrastructure covers everything.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Health poller emits `tailscale:health` events visible in Wails frontend | HEALTH-06 | Requires running Wails app with live frontend | Start app → open DevTools → verify `tailscale:health` events fire every 10s when Tailscale state changes |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
