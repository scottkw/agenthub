---
phase: 160
slug: v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-27
---

# Phase 160 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) + Go testing (backend) + bash/shellcheck (scripts) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `go test -race -short ./internal/relay/... && cd frontend && pnpm vitest run src/components/Hub/` |
| **Full suite command** | `go test -race -short ./... && bash tests/install-sh.test.sh && cd frontend && pnpm test` |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm vitest run src/components/Hub/ --passWithNoTests` (frontend tasks) or `go test -race -short ./internal/relay/...` (Go tasks) or `bash tests/install-sh.test.sh` (install.sh tasks)
- **After every plan wave:** Run `go test -race -short ./... && cd frontend && pnpm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 160-01 | 01 | 1 | NOTIF-01 (background WS source) | — | Read-only loopback WS; no frames sent to server | unit | `cd frontend && pnpm vitest run src/components/Hub/useChatUnreadListeners` | ❌ W0 | ⬜ pending |
| 160-02 | 02 | 2 | NOTIF-01 (prop threading + reset) | — | N/A | unit | `cd frontend && pnpm vitest run src/components/Hub/` | ❌ W0 | ⬜ pending |
| 160-03 | 03 | 1 | IN-02 | T-IN-02 (control-only inject) | `HandleInject("\x1b[2J")` → no PTY write, no Enter | unit | `go test -run TestInject_ControlOnly ./internal/relay/...` | ❌ W0 | ⬜ pending |
| 160-04 | 04 | 1 | WR-01 / WR-03 | — | Checksum match exact; INSTALL_DIR created | static/behavioral | `bash tests/install-sh.test.sh` | ✅ (extend) | ⬜ pending |
| 160-05 | 05 | 1 | IN-04 / NOTIF-02 / WR-02 | — | N/A (doc accuracy) | doc-only | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/Hub/useChatUnreadListeners.test.tsx` — new vitest file for the background unread hook (NOTIF-01 background WS)
- [ ] `frontend/src/components/Hub/HubPanel` unread-threading test — new or extended vitest file (NOTIF-01 prop threading + reset semantics)
- [ ] `frontend/src/components/Hub/SessionCardGrid.test.tsx` — extend to assert `unreadBySessionId` threads to `SessionCard` (NOTIF-01 card badge)
- [ ] `TestInject_ControlOnly*` in `internal/relay/server_inject_test.go` (or `hub` test) — IN-02 regression
- [ ] `tests/install-sh.test.sh` — extend with WR-01 (grep -F exactness) + WR-03 (root mkdir) assertions

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Background unread badge lights on a live closed-modal session when a real chat message arrives over the relay | NOTIF-01 | Requires a live daemon + a second chat participant over a real relay WS; unit tests prove the accrual/threading logic but not end-to-end relay delivery to a backgrounded card | Start daemon, open Hub, leave a session's modal CLOSED, send a chat message to that session from another surface (web-share or second client), confirm the Hub card badge increments and that opening the modal resets it to 0 |

*Unit/static automation covers all other phase behaviors.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
