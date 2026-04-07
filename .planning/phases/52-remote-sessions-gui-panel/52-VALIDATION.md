---
phase: 52
slug: remote-sessions-gui-panel
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-07
---

# Phase 52 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest + jsdom (frontend), Go testing stdlib (backend) |
| **Config file** | `frontend/vite.config.ts` (test: { environment: 'jsdom', globals: true }) |
| **Quick run command** | `cd frontend && pnpm test run` |
| **Full suite command** | `cd frontend && pnpm test run && cd /Users/ken/dev/agenthub && go test ./... -race` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test run`
- **After every plan wave:** Run `cd frontend && pnpm test run && cd /Users/ken/dev/agenthub && go test ./... -race`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 52-01-01 | 01 | 0 | REM-02 | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ W0 | ⬜ pending |
| 52-01-02 | 01 | 0 | REM-02 | unit (Go) | `go test ./... -race -run TestGetRemoteSessions` | ❌ W0 | ⬜ pending |
| 52-02-01 | 02 | 1 | REM-02 | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ W0 | ⬜ pending |
| 52-02-02 | 02 | 1 | REM-02 | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ W0 | ⬜ pending |
| 52-03-01 | 03 | 2 | REM-03 | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ W0 | ⬜ pending |
| 52-03-02 | 03 | 2 | REM-03 | unit (source) | `pnpm test run RemoteSessionsPanel` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/RemoteSessionsPanel.tsx` — component stub (does not exist yet)
- [ ] `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` — test file for source inspection + DOM tests
- [ ] `app_test.go` additions — test for `GetRemoteSessions` nil-safety (returns `[]RemotePeerSessions{}` not nil)
- No new framework install needed — Vitest and Go testing tooling confirmed present

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Open button launches system browser | REM-03 | BrowserOpenURL requires Wails runtime | Click Open on a session row; verify system browser opens with correct URL |
| 30s auto-refresh visible | REM-02 | Timing-based UI behavior | Wait 30s; confirm panel re-fetches and updates |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
