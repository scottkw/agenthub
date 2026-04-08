---
phase: 55
slug: sidebar-component-icons
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-08
---

# Phase 55 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest |
| **Config file** | frontend/vitest.config.ts |
| **Quick run command** | `cd frontend && pnpm test -- --run` |
| **Full suite command** | `cd frontend && pnpm test -- --run` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test -- --run`
- **After every plan wave:** Run `cd frontend && pnpm test -- --run`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 55-01-01 | 01 | 0 | UILAY-01 | unit | `cd frontend && pnpm test -- --run TabBar.test` | ✅ | ⬜ pending |
| 55-02-01 | 02 | 1 | ICON-01, ICON-02 | unit | `cd frontend && pnpm test -- --run` | ❌ W0 | ⬜ pending |
| 55-02-02 | 02 | 1 | SIDE-01, SIDE-02 | unit | `cd frontend && pnpm test -- --run Sidebar.test` | ❌ W0 | ⬜ pending |
| 55-02-03 | 02 | 1 | SIDE-03 | unit | `cd frontend && pnpm test -- --run Sidebar.test` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Fix `TabBar.test.tsx` UILAY-01 stale assertion (font-size 18px → 20px)
- [ ] `frontend/src/components/__tests__/Sidebar.test.tsx` — stubs for SIDE-01, SIDE-02, SIDE-03
- [ ] `pnpm add @heroicons/react` — install icon library

*Existing vitest infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Sidebar toggle animation smooth | SIDE-02 | CSS transitions need visual verification | Toggle sidebar, observe no jank or layout shift |
| Terminal refits on sidebar toggle | SIDE-02 | ResizeObserver + xterm.js refit timing | Toggle sidebar while terminal has content, verify no truncation |
| Sidebar state persists across app restart | SIDE-03 | Requires full Wails app lifecycle | Expand sidebar, close app, reopen — sidebar should be expanded |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
