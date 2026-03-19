---
phase: 3
slug: wails-desktop-ui
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-18
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (backend) + Vitest (React frontend) |
| **Config file** | `frontend/vite.config.ts` — `test` section (Wave 0 installs if needed) |
| **Quick run command** | `go test ./... -short` |
| **Full suite command** | `go test ./... && cd frontend && pnpm test --run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -short`
- **After every plan wave:** Run `go test ./... && cd frontend && pnpm test --run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 0 | TERM-02 | unit | `go test ./cmd/agenthub/... -run TestRenameSession` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 0 | CLI-03 | unit | `go test ./cmd/agenthub/... -run TestUpdateCLIPath` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 0 | SESS-02 | integration | `go test ./cmd/agenthub/... -run TestHideWindowSessionsAlive` | ❌ W0 | ⬜ pending |
| 03-01-04 | 01 | 0 | TERM-01 | unit | `cd frontend && pnpm test --run -- relayClient` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 1 | TERM-01 | integration | `go test ./cmd/agenthub/... -run TestCreateSession` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 1 | TERM-03 | manual-only | N/A — visual ANSI verification | manual | ⬜ pending |
| 03-02-03 | 02 | 1 | TERM-04 | unit | `go test ./internal/relay/... -run TestScrollback` | ✅ | ⬜ pending |
| 03-02-04 | 02 | 1 | TERM-05 | manual-only | N/A — clipboard requires browser context | manual | ⬜ pending |
| 03-03-01 | 03 | 2 | SESS-02 | integration | `go test ./cmd/agenthub/... -run TestTrayIntegration` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/agenthub/app_test.go` — unit tests for `CreateSession`, `RenameSession`, `KillSession`, `GetRelayPort`, `UpdateCLIPath`
- [ ] `cmd/agenthub/tray_test.go` — stub test confirming `HideWindowOnClose` behavior (session count unchanged after hide)
- [ ] `frontend/src/lib/relayClient.test.ts` — unit tests for binary framing encode/decode (MSG_OUTPUT, MSG_INPUT, MSG_RESIZE)
- [ ] Framework install: `pnpm add -D vitest @vitest/coverage-v8` in `frontend/` — if frontend dir is new

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ANSI color rendering | TERM-03 | Visual verification — ANSI pass-through is architectural guarantee, rendering correctness is browser-side | Run Claude Code in a tab, verify 256-color output, emoji, and box-drawing characters render without corruption |
| Copy/paste | TERM-05 | Clipboard API requires browser context and user interaction | Select text in terminal, Cmd+C, paste into external editor; paste external text into terminal via Cmd+V |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
