---
phase: 30
slug: backend-args-wiring
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-25
---

# Phase 30 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test ./internal/daemon/... -run TestArgs -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 30-01-01 | 01 | 1 | ARGS-03 | unit | `go test ./internal/daemon/... -run TestArgsRoundTrip -v` | ❌ W0 | ⬜ pending |
| 30-01-02 | 01 | 1 | ARGS-03 | integration | `go test ./internal/daemon/... -run TestClientCreateSessionWithArgs -v` | ❌ W0 | ⬜ pending |
| 30-01-03 | 01 | 1 | ARGS-03 | unit | `go test ./internal/daemon/... -run TestEngineCreateSessionWithArgs -v` | ❌ W0 | ⬜ pending |
| 30-01-04 | 01 | 1 | ARGS-03 | regression | `go test ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/engine_test.go` — add `TestEngineCreateSessionWithArgs` stub
- [ ] `internal/daemon/client_test.go` — add `TestClientCreateSessionWithArgs` stub using `testDaemon` helper

*No new test files required — add functions to existing test files.*

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
