---
phase: 80
slug: tailscale-detection
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-16
---

# Phase 80 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Go test is built-in |
| **Quick run command** | `go test ./internal/webserver/... ./internal/pty/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/... ./internal/pty/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 80-01-01 | 01 | 1 | TS-01 | — | N/A | unit | `go test ./internal/webserver/ -run TestCheckHealth` | ✅ | ⬜ pending |
| 80-01-02 | 01 | 1 | TS-01 | — | N/A | unit | `go test ./internal/webserver/ -run TestDetectBinary` | ❌ W0 | ⬜ pending |
| 80-01-03 | 01 | 1 | TS-02 | — | N/A | unit | `go test ./internal/webserver/ -run TestCheckHealth_DaemonStopped` | ❌ W0 | ⬜ pending |
| 80-02-01 | 02 | 2 | TS-02 | — | N/A | integration | `go test ./... -run TestHealthPoller` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/tailscale_test.go` — extend with 4-state health check tests
- [ ] `internal/webserver/tailscale_paths_test.go` — binary detection path tests

*Existing infrastructure covers test framework requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tailscale status displays correctly in Settings UI | TS-02 | Requires visual browser verification | Open Settings > Web Server; verify dot color and text match state |
| Platform-specific instructions render correctly | TS-02 | Platform-dependent UI text | Check "Show diagnostics" checklist on each platform |
| Tailscale path override works via Browse button | TS-01 | Requires file dialog interaction | Settings > Paths > Browse for tailscale binary |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
