---
phase: 37
slug: splash-screen
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-31
---

# Phase 37 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 2.x + jsdom |
| **Config file** | `frontend/vite.config.ts` (test section present) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 37-01-01 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- SplashScreen` | ❌ W0 | ⬜ pending |
| 37-01-02 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- App` | ✅ (extend) | ⬜ pending |
| 37-01-03 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- App` | ✅ (extend) | ⬜ pending |
| 37-01-04 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- App` | ✅ (extend) | ⬜ pending |
| 37-01-05 | 01 | 1 | BRND-02 | manual/visual | `wails build` production test | manual | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/SplashScreen.test.tsx` — stubs for BRND-02 (component structure, img src, visible/hidden state)

*Existing `App.test.tsx` will be extended with new test cases — not a gap.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| No white flash before splash on app launch | BRND-02 | Requires visual inspection of OS-level window show timing | Run `wails build`, launch built app, observe startup |
| Splash shows title logo immediately | BRND-02 | Visual rendering verification | Launch app, verify logo displays before main UI |
| Splash dismisses when daemon connects | BRND-02 | Requires running daemon | Launch app with daemon running, verify splash transitions to main UI |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
