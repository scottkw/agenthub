---
phase: 148
slug: session-tab-chevron
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-23
validated: 2026-06-23
note: Authored retroactively at v4.0 milestone close — phase shipped TDD without a VALIDATION.md; this documents the coverage that already exists and passes.
---

# Phase 148 — Validation Strategy

> Per-phase validation contract. Authored retroactively at milestone close to fill the
> Nyquist-coverage gap; the phase was executed TDD (RED → GREEN) and all coverage below is green.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (jsdom) — frontend only (no backend change) |
| **Config file** | `frontend/vitest.config.ts` |
| **Quick run command** | `cd frontend && pnpm test -- TabBar` |
| **Full suite + type check** | `cd frontend && npx tsc --noEmit && pnpm test` |
| **Estimated runtime** | ~5 seconds (TabBar suite) |

> Project rule (memory: run-tsc-in-the-frontend-gate): vitest tolerates TS errors that
> `tsc && vite build` rejects — `tsc --noEmit` is part of the gate.

---

## Per-Task Verification Map

| Task | Plan | Requirement | Test Type | Automated Command | Status |
|------|------|-------------|-----------|-------------------|--------|
| Chevron button renders on session tabs (truthy sessionId) | 148-01 | TAB-04 | unit (TDD RED→GREEN) | `pnpm test -- TabBar` | ✅ green |
| Special tabs (Welcome/Settings/Hub/Help/File-browser) render NO chevron (sessionId gate, D-05) | 148-01 | TAB-04 | unit | `pnpm test -- TabBar` | ✅ green |
| Chevron click opens menu anchored below via `getBoundingClientRect` (D-01), not cursor coords | 148-01 | TAB-04 | unit | `pnpm test -- TabBar` | ✅ green |
| Chevron inserted between countdown span and close button (D-03 DOM order) | 148-01 | TAB-04 | unit | `pnpm test -- TabBar` | ✅ green |
| Right-click on tab name still opens menu at cursor (unchanged) | 148-01 | TAB-04 | unit | `pnpm test -- TabBar` | ✅ green |
| `.tab__context-menu` tokenized to `--hub-*` for light/dark | 148-01 | TAB-04 | unit (CSS gate) | `pnpm test -- style.hub` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

None. TAB-04 extends the existing `TabBar` context menu (already present); the RED test
(commit `720ef346`) was written against existing stable structure before the chevron was
added (commit `5250e79d`). No scaffolding wave required.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Chevron dropdown opens at the correct on-screen position in the live native webview; theme correctness light/dark | TAB-04 | Exact pixel anchoring + live theme repaint are visual; covered structurally by unit tests | Build/`wails dev`; click ▾ on a session tab; confirm the Rename / Save Terminal As… / Browse files menu drops below the chevron and is readable in both themes. (Covered by TESTING.md visual regression where registered.) |

---

## Validation Sign-Off

- [x] All tasks have automated verify
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none required)
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-23 (retroactive)

---

## Post-Execution Validation (2026-06-23)

Authored at v4.0 milestone close to fill the only Nyquist-coverage gap (148 had no
VALIDATION.md). The phase shipped TDD and its coverage is green today:

- `pnpm test -- TabBar` → **39 tests passed** (chevron render, sessionId gate, rect-anchored
  open, DOM order, right-click-unchanged).
- `.tab__context-menu` tokenization covered by `style.hub.test.ts`.
- `npx tsc --noEmit` → exit 0.
- Phase 148 VERIFICATION passed; 148-REVIEW.md clean.
