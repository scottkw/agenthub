---
phase: 37
slug: splash-screen
status: audited
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-31
audited: 2026-04-02
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
| **Estimated runtime** | ~7 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 7 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 37-01-01 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- WelcomeTab` | ✅ WelcomeTab.test.tsx | ✅ green |
| 37-01-02 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- App` | ✅ App.test.tsx BRND-02 block | ✅ green |
| 37-01-03 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- App` | ✅ App.test.tsx (splash-static) | ✅ green |
| 37-01-04 | 01 | 1 | BRND-02 | unit (raw source) | `pnpm test -- App` | ✅ App.test.tsx (WelcomeTab render) | ✅ green |
| 37-01-05 | 01 | 1 | BRND-02 | manual/visual | `wails build -tags wailsassets` | manual | ✅ verified |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test framework or config needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| No white flash before splash on app launch | BRND-02 | Requires Wails runtime + OS window manager (StartHidden/OnDomReady lifecycle) | Run `wails build -tags wailsassets`, launch built app, observe startup |
| Splash shows title logo immediately | BRND-02 | Visual rendering verification requiring Wails runtime | Launch app, verify logo displays before main UI |
| Go StartHidden: true lifecycle | BRND-02 | Wails window lifecycle not unit-testable | Verify `grep -q 'StartHidden' main.go` — confirmed present |
| Go OnDomReady -> WindowShow lifecycle | BRND-02 | Wails runtime required for window show | Verify `grep -q 'domReady' app.go` — confirmed present |

---

## Test Coverage Summary

| Test File | Tests | Scope |
|-----------|-------|-------|
| `frontend/src/components/__tests__/WelcomeTab.test.tsx` | 9 | Component structure, logo, tagline, version, install instructions, links, BEM classes |
| `frontend/src/components/__tests__/App.test.tsx` (BRND-02 block) | 6 | WelcomeTab import, tab type, initial state, activeId, render, splash-static cleanup |
| **Total** | **15** | BRND-02 fully covered (frontend) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 7s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-02

---

## Validation Audit 2026-04-02

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

**Notes:** Phase implementation evolved from plan — `SplashScreen.tsx` was replaced by `WelcomeTab.tsx` during execution (welcome tab pattern instead of overlay splash). All BRND-02 requirements remain fully covered. Go lifecycle hooks (StartHidden, OnDomReady, WindowShow) are inherently manual-only — Wails runtime required. 15 automated tests cover all frontend behaviors. All 177 project tests pass.
