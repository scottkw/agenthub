---
phase: 5
slug: qr-codes-status-indicators
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-18
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (existing) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test ./internal/status/... ./internal/webserver/... -v` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/status/... ./internal/webserver/... -v`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | QR-01 | unit | `go test . -run TestGetSessionQRCode -v` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | QR-01 | unit | `go test ./internal/... -run TestQREncode` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 1 | QR-02 | integration | `go test ./internal/webserver/... -run TestQREndpoint` | ❌ W0 | ⬜ pending |
| 05-01-04 | 01 | 1 | QR-02 | integration | `go test ./internal/webserver/... -run TestQREndpointNotEnabled` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 1 | STAT-01 | integration | `go test . -run TestStatusEventEmit` | ❌ W0 | ⬜ pending |
| 05-02-02 | 02 | 1 | STAT-02 | unit | `go test ./internal/status/... -run TestDetector` | ❌ W0 | ⬜ pending |
| 05-02-03 | 02 | 1 | STAT-02 | unit | `go test ./internal/status/... -run TestANSIStrip` | ❌ W0 | ⬜ pending |
| 05-02-04 | 02 | 1 | STAT-02 | unit | `go test ./internal/status/... -run TestDetectorShutdown` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/status/detector_test.go` — stubs for STAT-02 patterns + ANSI stripping + shutdown
- [ ] `internal/webserver/server_test.go` additions — QR endpoint tests (TestQREndpoint, TestQREndpointNotEnabled)
- [ ] `app_test.go` additions — TestGetSessionQRCode against testApp helper

*Existing `go test` infrastructure covers framework setup.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| QR code scans correctly on phone | QR-01 | Physical device scanning cannot be automated | 1. Enable web serving for a session 2. Open QR in desktop UI 3. Scan with phone camera 4. Verify URL loads session |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
