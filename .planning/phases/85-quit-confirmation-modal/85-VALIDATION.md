---
phase: 85
slug: quit-confirmation-modal
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-19
validated: 2026-04-19
---

# Phase 85 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest + jsdom (frontend); Go testing (backend) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && pnpm test --run` |
| **Full suite command** | `cd frontend && pnpm test --run && cd .. && go test -tags wailsassets ./... -run 'TestBeforeClose\|TestTrayQuit\|TestHideWindow\|TestQuit'` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test --run`
- **After every plan wave:** Run `cd frontend && pnpm test --run && cd .. && go test -tags wailsassets ./... -run 'TestBeforeClose\|TestTrayQuit\|TestHideWindow\|TestQuit'`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 85-01-01 | 01 | 1 | APP-01 | — | N/A | unit | `cd frontend && npx vitest run` | ✅ | ✅ green |
| 85-01-02 | 01 | 1 | APP-02 | — | N/A | unit | `cd frontend && npx vitest run` | ✅ | ✅ green |
| 85-01-03 | 01 | 1 | APP-03 | — | N/A | unit | `cd frontend && npx vitest run` | ✅ | ✅ green |
| 85-02-01 | 02 | 1 | APP-01 | — | N/A | unit (Go) | `go test -tags wailsassets -run TestBeforeCloseEmitsEvent -v .` | ✅ | ✅ green |
| 85-02-02 | 02 | 1 | APP-02 | — | N/A | unit (Go) | `go test -tags wailsassets -run TestQuitAll -v .` | ✅ | ✅ green |
| 85-regression-01 | — | — | APP-01 | — | N/A | unit (Go, existing) | `go test -tags wailsassets -run TestBeforeCloseReturnsTrue -v .` | ✅ | ✅ green |
| 85-regression-02 | — | — | APP-01 | — | N/A | unit (Go, existing) | `go test -tags wailsassets -run TestHideWindowSessionsAlive -v .` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `frontend/src/components/__tests__/QuitConfirmModal.test.tsx` — 25 source-inspection tests for APP-01, APP-02, APP-03 (created during plan 02 execution)
- [x] Go test additions to `tray_test.go`: `TestBeforeCloseEmitsEvent`, `TestQuitAll` (created during Nyquist validation audit)

*Existing `TestBeforeCloseReturnsTrue` and `TestHideWindowSessionsAlive` continue to pass — regression guards for the beforeClose refactor.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| macOS notification appears after "Quit GUI Only" | D-11 | Requires macOS Notification Center interaction; cannot be tested headlessly | 1. Click "Quit GUI Only" in modal. 2. Check macOS Notification Center for "AgentHub is still running in the background" message. |
| Window auto-shows when hidden + tray Quit | D-08 | Requires actual window visibility state on macOS | 1. Hide window to tray. 2. Click tray "Quit". 3. Verify window becomes visible with modal. |

---

## Validation Audit 2026-04-19

| Metric | Count |
|--------|-------|
| Gaps found | 2 |
| Resolved | 2 |
| Escalated | 0 |

**Tests added:** `TestBeforeCloseEmitsEvent` (APP-01 — beforeClose emits event, no WindowHide), `TestQuitAll` (APP-02 — sets quitting flag, calls ShutdownDaemon, guards nil ctx/client). Both in `tray_test.go`.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete (2026-04-19)
