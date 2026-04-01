---
phase: 38
slug: remote-session-metadata
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-01
---

# Phase 38 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./internal/daemon/... -run TestAPIListSessionsHostname\|TestEngineListSessionsHostname` |
| **Full suite command** | `go test ./internal/daemon/...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... -run TestAPIListSessionsHostname\|TestEngineListSessionsHostname`
- **After every plan wave:** Run `go test ./internal/daemon/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 38-01-01 | 01 | 1 | RMTE-03 | unit | `go test ./internal/daemon/... -run TestEngineListSessionsHostname` | ❌ W0 | ⬜ pending |
| 38-01-02 | 01 | 1 | RMTE-03 | integration | `go test ./internal/daemon/... -run TestAPIListSessionsHostname` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/engine_test.go` — add `TestEngineListSessionsHostname` (inline, no new file)
- [ ] `internal/daemon/api_test.go` — add `TestAPIListSessionsHostname` (inline, no new file)

*Existing infrastructure covers all phase requirements — no new test framework or fixtures needed.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
