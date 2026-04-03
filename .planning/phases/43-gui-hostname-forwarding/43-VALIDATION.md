---
phase: 43
slug: gui-hostname-forwarding
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-03
---

# Phase 43 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend), Go testing package (backend) |
| **Config file** | `frontend/vite.config.ts` (vitest inline) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test --run` |
| **Full suite command** | `go test ./... && cd frontend && pnpm test --run` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test --run`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub && go test ./... && cd frontend && pnpm test --run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 43-01-01 | 01 | 1 | RMTE-03 | unit (Go) | `go test -run TestListSessions ./...` | Partial — needs hostname assertion | ⬜ pending |
| 43-01-02 | 01 | 1 | RMTE-03 | unit (Go) | `go test -run TestListSessions ./...` | Partial — needs hostname assertion | ⬜ pending |
| 43-01-03 | 01 | 1 | DMGR-03 | unit (vitest DOM) | `cd frontend && pnpm test --run DaemonManagerPanel` | Partial — needs hostname assertion | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `app_test.go` — extend existing list test to assert `Hostname != ""`
- [ ] `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx` — update `mockSessions` to include `hostname`, add DOM assertion

*Existing test infrastructure covers the phase — no new files needed, only additions to existing tests.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Hostname visually renders in DaemonManagerPanel | DMGR-03 | Visual layout/styling check | Run app, open DaemonManagerPanel, verify hostname appears per session row |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
