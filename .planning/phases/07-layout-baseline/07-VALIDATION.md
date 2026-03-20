---
phase: 7
slug: layout-baseline
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-03-19
updated: 2026-03-20
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` (vitest configured inline) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Estimated runtime** | ~1 second |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 7-01-01 | 01 | 1 | TERM-01 | unit (source inspection) | `pnpm test -- --reporter=verbose` | ✅ | ✅ green |
| 7-01-02 | 01 | 1 | TERM-01 | unit (CSS inspection) | `pnpm test -- --reporter=verbose` | ✅ | ✅ green |
| 7-02-01 | 01 | 1 | UILAY-01 | unit (DOM structure) | `pnpm test -- --reporter=verbose` | ✅ | ✅ green |
| 7-02-02 | 01 | 1 | UILAY-01 | unit (CSS inspection) | `pnpm test -- --reporter=verbose` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — verifies inline styles (flex:1, minHeight:0, width:100%) AND parent `.terminal-container { min-height: 0 }` in style.css
- [x] `frontend/src/components/__tests__/TabBar.test.tsx` — verifies structural classnames AND CSS dimensions (42px tab-bar, 38x38px buttons, 18px font-size)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Terminal fills full vertical height with no dead space | TERM-01 | Full layout behavior requires real browser rendering | Run app, check no blank space below terminal output in any tab |
| Initial-paint fill (known xterm.js FitAddon timing race) | TERM-01 | Cannot test timing race statically; user tabled this gap | Run app, observe first render — may show brief dead space before first resize |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** complete

---

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 2 |
| Resolved | 2 |
| Escalated | 0 |

**Details:**
- Gap 1 (TERM-01): Added CSS source-inspection test for `.terminal-container { min-height: 0 }` via `readFileSync` on style.css
- Gap 2 (UILAY-01): Added CSS source-inspection tests for `.tab-bar { height: 42px }` and `.tab-bar__btn { width: 38px; height: 38px; font-size: 18px }`
- Both gaps resolved by adding tests to existing test files (TerminalPanel.test.tsx, TabBar.test.tsx)
- Added `@types/node` dev dependency to support `readFileSync` imports in test files
