---
phase: 8
slug: per-tab-status-bar
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-19
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
| **Estimated runtime** | ~5 seconds |

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
| 8-01-01 | 01 | 1 | UILAY-02 | unit | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 8-01-02 | 01 | 1 | UILAY-02 | unit | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 8-01-03 | 01 | 1 | UILAY-02 | unit | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 8-02-01 | 02 | 1 | UILAY-03 | unit | `cd frontend && pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 8-02-02 | 02 | 1 | UILAY-02 | manual | Visual inspection | N/A | ⬜ pending |
| 8-02-03 | 02 | 1 | UILAY-03 | manual | Visual inspection | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/StatusBar.test.tsx` — stubs for UILAY-02 (rendering, state variants)
- [ ] `frontend/src/components/StatusBar.tsx` — component extracted from App.tsx

*Existing vitest infrastructure covers all tooling needs — only new source + test files are missing.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Status bar is at bottom of tab (position/layout) | UILAY-02 | CSS layout positioning not unit-testable | Run app, verify status bar appears below terminal in each tab |
| Terminal fills full height with no dead space above status bar | UILAY-03 | CSS layout rendering not unit-testable | Run app, verify no gap between terminal and status bar |
| Status bar layout correct on macOS, Linux, and Windows | UILAY-02 | Cross-platform rendering | Test on each platform |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
