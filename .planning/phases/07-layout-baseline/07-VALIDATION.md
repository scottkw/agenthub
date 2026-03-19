---
phase: 7
slug: layout-baseline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-19
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
| **Estimated runtime** | ~5 seconds |

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
| 7-01-01 | 01 | 1 | TERM-01 | unit | `pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 7-01-02 | 01 | 1 | TERM-01 | unit | `pnpm test -- --reporter=verbose` | ❌ W0 | ⬜ pending |
| 7-02-01 | 02 | 1 | UILAY-01 | manual-only | Visual inspection | N/A | ⬜ pending |
| 7-02-02 | 02 | 1 | UILAY-01 | manual-only | Visual inspection | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — stubs for TERM-01 (verifies inline styles: flex:1, minHeight:0, width:100%)
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — stubs for UILAY-01 structure (classnames present)

*Existing vitest infrastructure is in place — only test files are missing.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tab bar buttons are >= 36px in both dimensions | UILAY-01 | CSS pixel dimensions require layout engine; jsdom has none | Run app, inspect `.tab-bar__btn` computed styles in DevTools |
| Tab bar height accommodates larger buttons without overflow | UILAY-01 | Visual layout property | Run app, verify no overflow/clipping on tab bar |
| Terminal fills full vertical height with no dead space | TERM-01 | Full layout behavior requires real browser | Run app, check no blank space below terminal output in any tab |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
