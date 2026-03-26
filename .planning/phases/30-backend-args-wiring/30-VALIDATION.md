---
phase: 30
slug: backend-args-wiring
status: draft
nyquist_compliant: true
wave_0_complete: true
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
| **Linter command** | `golangci-lint run ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... -v`
- **After every plan wave:** Run `go test ./...` and `golangci-lint run ./...`
- **Before `/gsd:verify-work`:** Full suite and linter must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 30-01-01 | 01 | 1 | ARGS-03 | unit | `go test ./internal/daemon/... -run TestAPICreateSessionWithArgs -v` | Created in T2 | ⬜ pending |
| 30-01-02 | 01 | 1 | ARGS-03 | integration | `go test ./internal/daemon/... -run TestClientCreateSessionWithArgs -v` | Created in T2 | ⬜ pending |
| 30-01-03 | 01 | 1 | ARGS-03 | unit | `go test ./internal/daemon/... -run TestEngineCreateSessionWithArgs -v` | Created in T2 | ⬜ pending |
| 30-01-04 | 01 | 1 | ARGS-03 | regression | `go test ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

All Wave 0 tests are created within Task 2 of Plan 01:

- [x] `internal/daemon/api_test.go` — `TestAPICreateSessionWithArgs` (created in Task 2)
- [x] `internal/daemon/api_test.go` — `TestClientCreateSessionWithArgs` (created in Task 2)
- [x] `internal/daemon/engine_test.go` — `TestEngineCreateSessionWithArgs` (created in Task 2)

*All three test functions are created as part of Task 2, which is a TDD task — tests are written before confirming they pass.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** ready
