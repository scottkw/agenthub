---
phase: 26
slug: graceful-gui-startup-failure
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-24
---

# Phase 26 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest |
| **Config file** | `vite.config.ts` (frontend), Go test files (backend) |
| **Quick run command** | `go test ./... -run TestGraceful -count=1` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && npx vitest run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -run TestGraceful -count=1`
- **After every plan wave:** Run `go test ./... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 26-01-01 | 01 | 1 | DAEMON-05 | unit | `go test ./... -run TestStartupError -count=1` | ❌ W0 | ⬜ pending |
| 26-01-02 | 01 | 1 | DAEMON-05 | unit | `go test ./... -run TestRetryDaemon -count=1` | ❌ W0 | ⬜ pending |
| 26-02-01 | 02 | 1 | DAEMON-05 | component | `cd frontend && npx vitest run --reporter=verbose` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `app_startup_test.go` — stubs for startup error handling tests
- [ ] Frontend test file for error banner component — stubs for daemon error UI tests

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GUI does not crash when daemon binary missing | DAEMON-05 | Requires removing binary and launching app | 1. Remove daemon binary 2. Launch GUI 3. Verify error banner appears 4. Click retry after restoring binary |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
