---
phase: 70
slug: sidebar-icon-position-stability
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-13
---

# Phase 70 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.0 + jsdom |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && pnpm test`
- **Before `/gsd-verify-work`:** Full suite must be green + manual visual verification of sidebar toggle
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 70-01-XX | 01 | — | SBR-02 | — | N/A | CSS unit | `cd frontend && pnpm test -- Sidebar` | ✅ | ⬜ pending |

*Populated by planner; executor updates Status during runs.*

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] New test case(s) in `frontend/src/components/__tests__/Sidebar.test.tsx` — assert `.sidebar__icon` margin is `0 14px` (structural position-stability contract)
- [ ] Anti-regression: assert no `justify-content: center` override on `.sidebar--collapsed .sidebar__item`

*Existing Vitest + jsdom infrastructure covers the CSS-level contract; no framework installs required.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| No perceived horizontal jump on toggle | SBR-02 (crit. 2) | True pixel-diff requires real browser; jsdom has no layout engine | Launch `wails dev`, click hamburger toggle ~10 times, observe icons stay at x=24px center |
| Smooth transition, no reflow flicker | SBR-02 (crit. 3) | Animation quality is visual-only | Launch `wails dev`, toggle rapidly, confirm 0.15s width ease with no flicker/jitter |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
