---
phase: 17
slug: dead-code-cleanup
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-20
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `go test` (stdlib) |
| **Framework (Frontend)** | vitest v4.1.0 |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run (Go)** | `go test ./...` |
| **Quick run (Frontend)** | `pnpm --prefix frontend test -- --run` |
| **Full suite** | Both above sequentially |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./...`
- **After every plan wave:** Run full suite (Go + Frontend)
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 17-01-01 | 01 | 1 | CLEAN-01 | build verification | `go build ./...` | ✅ | ⬜ pending |
| 17-01-02 | 01 | 1 | CLEAN-01 | source inspection | `grep -r GetNetworkInterfaces --include="*.go"` | ✅ | ⬜ pending |
| 17-01-03 | 01 | 2 | CLEAN-02 | existing test | `go test ./internal/webserver/... -run TestLoginRouteNotRegistered\|TestTokenRouteNotRegistered` | ✅ | ⬜ pending |
| 17-01-04 | 01 | 3 | CLEAN-03 | existing test | `pnpm --prefix frontend test -- --run` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test files needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
