---
phase: 59
slug: auto-serve-sessions
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-09
---

# Phase 59 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `testing` package + `go test` |
| **Framework (frontend)** | Vitest 4.1.0 |
| **Config file (Go)** | None (standard `go test`) |
| **Config file (frontend)** | `frontend/vite.config.ts` |
| **Quick run command** | `go test ./internal/daemon/... -run TestAutoServe && cd frontend && pnpm test` |
| **Full suite command** | `go test ./... && cd frontend && pnpm test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/daemon/... && cd frontend && pnpm test`
- **After every plan wave:** Run `go test ./... && cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 59-01-01 | 01 | 0 | SERVE-01 | unit (Go) | `go test ./internal/daemon/... -run TestAutoStartWebServer_CreatesNewServer` | ✅ | ✅ green |
| 59-01-02 | 01 | 0 | SERVE-01 | unit (Go) | `go test ./internal/daemon/... -run TestAutoStartWebServer_LocalModeRequiresPassword` | ✅ | ✅ green |
| 59-01-03 | 01 | 0 | SERVE-02 | unit (Go) | `go test ./internal/daemon/... -run TestCreateSession_AutoWebEnable` | ✅ | ✅ green |
| 59-01-04 | 01 | 0 | SERVE-02 | unit (Go) | `go test ./internal/daemon/... -run TestCreateSession_NoAutoEnable` | ✅ | ✅ green |
| 59-01-05 | 01 | 1 | SERVE-02 | unit (TS) | `cd frontend && pnpm test -- --run` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/daemon/api_test.go` — `TestAutoStartWebServer_CreatesNewServer`: verify AutoStartWebServer creates and starts a web server in local mode when none exists
- [x] `internal/daemon/api_test.go` — `TestAutoStartWebServer_LocalModeRequiresPassword`: verify local mode with empty password returns error (guards unauthenticated access)
- [x] `internal/daemon/api_test.go` — `TestAutoStartWebServer_AlreadyRunning`: verify idempotent no-op when web server already set
- [x] `internal/daemon/api_test.go` — `TestCreateSession_AutoWebEnable`: create session with web server pre-set; verify session auto-enabled
- [x] `internal/daemon/api_test.go` — `TestCreateSession_NoAutoEnable`: create session without web server; verify no auto-enable

*Frontend tests use existing App.test.tsx infrastructure — 246 tests pass including webEnabled props.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Daemon restart re-starts web server | SERVE-01 | Requires full daemon lifecycle (stop+start) | 1. Start daemon 2. Verify web server running 3. Stop daemon 4. Start daemon again 5. Verify web server running again |
| Session list shows web toggle ON | SERVE-02 | Visual UI verification | 1. Create new session while web server running 2. Open daemon panel 3. Verify toggle is ON |

---

## Validation Audit 2026-04-10

| Metric | Count |
|--------|-------|
| Gaps found | 2 |
| Resolved | 2 |
| Escalated | 0 |

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete
