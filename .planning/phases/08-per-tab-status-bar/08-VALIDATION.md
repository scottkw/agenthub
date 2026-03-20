---
phase: 8
slug: per-tab-status-bar
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-19
updated: 2026-03-20
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` (vitest configured inline) |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test` |
| **Estimated runtime** | ~1.2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 8-01-01 | 01 | 1 | UILAY-02 | unit | `cd frontend && pnpm test -- --reporter=verbose` | ✅ StatusBar.test.tsx | ✅ green |
| 8-01-02 | 01 | 1 | UILAY-02 | unit | `cd frontend && pnpm test -- --reporter=verbose` | ✅ StatusBar.test.tsx | ✅ green |
| 8-02-01 | 02 | 1 | UILAY-02 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ✅ App.test.tsx | ✅ green |
| 8-02-02 | 02 | 1 | UILAY-03 | source-inspection | `cd frontend && pnpm test -- --reporter=verbose` | ✅ App.test.tsx | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Automated Test Coverage

### UILAY-02 — StatusBar component (9 tests in StatusBar.test.tsx)
- renders .tab-status-bar root element
- shows inactive state with "WEB SERVER NOT RUNNING" when webServerRunning=false
- shows off state with "WEB OFF" when webServerRunning=true, webEnabled=false
- shows "Enable Web" button in off state
- shows on state with "WEB ON" when web enabled
- shows URL link with .tab-status-bar__url
- shows "Disable Web", "Copy Link", and "QR" buttons when web enabled
- calls onToggleWeb when Enable Web button clicked
- calls onToggleWeb when Disable Web button clicked

### UILAY-02 — App.tsx integration (3 tests in App.test.tsx)
- imports StatusBar from components/StatusBar
- renders `<StatusBar` inside terminal-wrapper
- passes required props to StatusBar (webServerRunning, webEnabled, sessionURL)

### UILAY-03 — Old overlay removed (4 tests in App.test.tsx)
- does not contain web-serving-bar class
- does not contain web-toggle-btn class
- does not contain web-session-url class
- does not contain copy-token-btn class

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Status bar is at bottom of tab (position/layout) | UILAY-02 | CSS layout positioning not unit-testable | Run app, verify status bar appears below terminal in each tab |
| Terminal fills full height with no dead space above status bar | UILAY-03 | CSS layout rendering not unit-testable | Run app, verify no gap between terminal and status bar |
| Status bar layout correct on macOS, Linux, and Windows | UILAY-02 | Cross-platform rendering | Test on each platform |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 2 |
| Resolved | 2 |
| Escalated | 0 |
