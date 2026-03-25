---
phase: 19
slug: daemon-core-engine-ipc
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-23
---

# Phase 19 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (go1.26.1) |
| **Config file** | None — standard `go test` |
| **Quick run command** | `go test -race ./internal/daemon/...` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race ./internal/daemon/... ./...`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 19-01-01 | 01 | 1 | DAEMON-02 | unit | `go test -race ./internal/daemon/... -run TestAPI` | ❌ W0 | ⬜ pending |
| 19-01-02 | 01 | 1 | DAEMON-02 | unit | `go test -race ./internal/daemon/... -run TestSessionCRUD` | ❌ W0 | ⬜ pending |
| 19-01-03 | 01 | 1 | DAEMON-02 | unit | `go test -race ./internal/daemon/... -run TestClient` | ❌ W0 | ⬜ pending |
| 19-01-04 | 01 | 1 | DAEMON-02 | unit | `go test -race ./internal/daemon/... -run TestStaleSocket` | ❌ W0 | ⬜ pending |
| 19-01-05 | 01 | 1 | DAEMON-02 | unit | `go test -race ./internal/daemon/... -run TestSocketPathLength` | ❌ W0 | ⬜ pending |
| 19-01-06 | 01 | 1 | DAEMON-02 | integration | `go test -race . -run TestCreate` | ✅ existing | ⬜ pending |
| 19-01-07 | 01 | 1 | DAEMON-02 | integration | `go test -race . -run Test` (all app_test.go) | ✅ existing | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/daemon/engine_test.go` — unit tests for SessionEngine CRUD, covers DAEMON-02
- [ ] `internal/daemon/api_test.go` — HTTP handler tests over real Unix socket, covers DAEMON-02
- [ ] `internal/daemon/client_test.go` — DaemonClient round-trip tests, covers DAEMON-02
- [ ] `internal/daemon/socket_test.go` — stale socket + path length tests, covers DAEMON-02

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GUI behavior identical to v1.2 | DAEMON-02 | Visual parity requires manual inspection | Launch app, create/rename/kill sessions, verify web UI loads correctly |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
