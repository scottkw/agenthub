---
phase: 52
slug: remote-sessions-gui-panel
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-07
audited: 2026-04-07
---

# Phase 52 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest + jsdom (frontend), Go testing stdlib (backend) |
| **Config file** | `frontend/vite.config.ts` (test: { environment: 'jsdom', globals: true }) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && npx vitest run` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && npx vitest run && cd /Users/ken/dev/agenthub && go test ./... -race` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && npx vitest run`
- **After every plan wave:** Run `cd frontend && npx vitest run && cd /Users/ken/dev/agenthub && go test ./... -race`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File | Status |
|---------|------|------|-------------|-----------|-------------------|------|--------|
| 52-01-01 | 01 | 0 | REM-02 | unit (Go) | `go test -race -run TestNilClientGetRemoteSessions -v` | `app_test.go` | green |
| 52-01-02 | 01 | 0 | REM-02 | unit (DOM) | `npx vitest run RemoteSessionsPanel` | `RemoteSessionsPanel.test.tsx` | green |
| 52-02-01 | 02 | 1 | REM-02 | unit (DOM) | `npx vitest run RemoteSessionsPanel` | `RemoteSessionsPanel.test.tsx` | green |
| 52-02-02 | 02 | 1 | REM-02 | unit (DOM) | `npx vitest run RemoteSessionsPanel` | `RemoteSessionsPanel.test.tsx` | green |
| 52-03-01 | 03 | 2 | REM-03 | unit (DOM) | `npx vitest run TabBar` | `TabBar.test.tsx` | green |
| 52-03-02 | 03 | 2 | REM-03 | unit (source) | `npx vitest run App.wiring` | `App.wiring.test.tsx` | green |

*Status: ⬜ pending · green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `frontend/src/components/RemoteSessionsPanel.tsx` — component exists
- [x] `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` — 19 tests passing
- [x] `app_test.go` — `TestNilClientGetRemoteSessions` passes, race-clean

---

## Gap Fills (Nyquist Audit 2026-04-07)

| Gap | Task ID | Test Added | File | Status |
|-----|---------|------------|------|--------|
| Globe button renders and fires callback | 52-03-01 | `TabBar globe button (52-03-01)` describe block (2 tests) | `TabBar.test.tsx` | green |
| App.tsx wiring source inspection | 52-03-02 | `App.tsx remote-sessions wiring (52-03-02)` describe block (14 tests) | `App.wiring.test.tsx` | green |

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Open button launches system browser | REM-03 | BrowserOpenURL requires Wails runtime | Click Open on a session row; verify system browser opens with correct URL |
| 30s auto-refresh visible | REM-02 | Timing-based UI behavior | Wait 30s; confirm panel re-fetches and updates |

---

## Validation Sign-Off

- [x] All tasks have automated verify
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 all requirements met
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** Nyquist audit complete — 248 frontend tests + Go nil-client test all green
