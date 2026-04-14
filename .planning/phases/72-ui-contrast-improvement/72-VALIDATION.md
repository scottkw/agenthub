---
phase: 72
slug: ui-contrast-improvement
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
---

# Phase 72 — Validation Strategy

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
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 72-01-01 | 01 | 0 | UI-01 | — | N/A | unit | `cd frontend && pnpm test -- style.contrast` | ❌ W0 | ⬜ pending |
| 72-02-01 | 02 | 1 | UI-01 | — | N/A | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ W0 | ⬜ pending |
| 72-02-02 | 02 | 1 | UI-01 | — | N/A | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ W0 | ⬜ pending |
| 72-02-03 | 02 | 1 | UI-01 | — | N/A | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ W0 | ⬜ pending |
| 72-02-04 | 02 | 1 | UI-01 | — | N/A | unit (CSS inspection) | `cd frontend && pnpm test -- style.contrast` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/style.contrast.test.ts` — stubs for UI-01 contrast assertions
  - Embeds WCAG formula, asserts `contrastRatio('#9aa5ce', bg) >= 4.5` for each background
  - Asserts failing-selector regex patterns do NOT still contain `#565f89` as text color

*Existing infrastructure covers test runner and framework.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual appearance of fixed colors | UI-01 | Automated tests verify contrast ratios but not subjective legibility | Launch app, check sidebar, tabs, settings, welcome screen for comfortable readability |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
