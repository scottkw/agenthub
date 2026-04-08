---
phase: 56
slug: navigation-wiring-tab-bar-cleanup
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-08
---

# Phase 56 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | frontend/vite.config.ts |
| **Quick run command** | `cd frontend && pnpm test` |
| **Full suite command** | `cd frontend && pnpm test` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm test`
- **After every plan wave:** Run `cd frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 56-01-01 | 01 | 1 | NAV-01 | unit (source inspection) | `cd frontend && pnpm test -- App.nav` | ❌ W0 | ⬜ pending |
| 56-01-02 | 01 | 1 | NAV-02 | unit (source inspection) | `cd frontend && pnpm test -- App.nav` | ❌ W0 | ⬜ pending |
| 56-01-03 | 01 | 1 | NAV-03 | unit (source inspection) | `cd frontend && pnpm test -- App.nav` | ❌ W0 | ⬜ pending |
| 56-01-04 | 01 | 1 | NAV-04 | unit (source inspection) | `cd frontend && pnpm test -- App.nav` | ❌ W0 | ⬜ pending |
| 56-01-05 | 01 | 1 | NAV-05 | unit (source inspection) | `cd frontend && pnpm test -- App.nav` | ❌ W0 | ⬜ pending |
| 56-02-01 | 02 | 1 | TAB-01 | unit (source inspection + CSS check) | `cd frontend && pnpm test -- App.nav` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/App.nav.test.tsx` — stubs for NAV-01, NAV-02, NAV-03, NAV-04, NAV-05, TAB-01
- [ ] Remove UILAY-01 describe block from `frontend/src/components/__tests__/TabBar.test.tsx` — 4 tests that assert `.tab-bar__btn` CSS (will break when dead CSS is deleted for TAB-01)

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
