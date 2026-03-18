---
phase: 2
slug: session-registry-websocket-relay
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-18
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib) |
| **Config file** | none — go test discovers by convention |
| **Quick run command** | `go test ./internal/relay/... -v -timeout 30s -race` |
| **Full suite command** | `go test ./... -v -timeout 60s -race` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/relay/... -v -timeout 30s -race`
- **After every plan wave:** Run `go test ./... -v -timeout 60s -race`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 0 | SESS-03 | unit | `go test ./internal/relay/... -run TestFrame -v` | Wave 0 | ⬜ pending |
| 02-01-02 | 01 | 1 | SESS-03 (criterion 1) | integration | `go test ./internal/relay/... -run TestHub_TwoClientsFanOut -v` | Wave 0 | ⬜ pending |
| 02-01-03 | 01 | 1 | SESS-03 (criterion 2) | integration | `go test ./internal/relay/... -run TestHub_ReconnectScrollback -v` | Wave 0 | ⬜ pending |
| 02-01-04 | 01 | 1 | SESS-03 (criterion 3) | integration | `go test ./internal/relay/... -run TestHub_InputFanOut -v` | Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/relay/hub.go` — Hub struct, Subscribe/Unsubscribe, broadcast, drain goroutine
- [ ] `internal/relay/manager.go` — HubManager: Create/Get/Remove per session
- [ ] `internal/relay/protocol.go` — MsgOutput, MsgInput, MsgResize constants + frame helpers
- [ ] `internal/relay/scrollback.go` — Scrollback struct with Append/Snapshot/size cap
- [ ] `internal/relay/server.go` — HTTP server with ServeMux routes and WebSocket handler
- [ ] `internal/relay/hub_test.go` — TestHub_TwoClientsFanOut, TestHub_ReconnectScrollback, TestHub_InputFanOut, TestHub_SlowClientDisconnected
- [ ] `internal/relay/protocol_test.go` — Frame encode/decode round-trip tests
- [ ] `internal/relay/scrollback_test.go` — Append/Snapshot, truncation boundary test

*(Framework install: none — Go stdlib `testing`; `coder/websocket` added to go.mod)*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
