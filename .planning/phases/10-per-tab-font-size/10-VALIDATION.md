---
phase: 10
slug: per-tab-font-size
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-19
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` (test: { environment: 'jsdom' }) |
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
| 10-01-01 | 01 | 0 | TERM-02, TERM-03, TERM-04 | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ (new cases needed) | ⬜ pending |
| 10-01-02 | 01 | 0 | TERM-04 | unit (source inspection) | `pnpm test -- App` | ❌ W0 | ⬜ pending |
| 10-02-01 | 02 | 1 | TERM-04 | unit (source inspection) | `pnpm test -- App` | ❌ W0 | ⬜ pending |
| 10-02-02 | 02 | 1 | TERM-02, TERM-03, TERM-04 | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ (new cases needed) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] New test cases in `frontend/src/components/__tests__/TerminalPanel.test.tsx` — stubs for TERM-02, TERM-03, TERM-04 via `?raw` source inspection
- [ ] New test file `frontend/src/components/__tests__/App.test.tsx` — stubs for TERM-04 (fontSizes state in App.tsx) via `?raw` source inspection

*Vitest, jsdom, and React testing infrastructure already configured. No new framework setup needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| SHIFT+= visibly increases font size in active terminal | TERM-02 | Requires real canvas rendering + visual confirmation | Press SHIFT+= in terminal tab, observe font size increase |
| SHIFT+- visibly decreases font size in active terminal | TERM-03 | Requires real canvas rendering + visual confirmation | Press SHIFT+- in terminal tab, observe font size decrease |
| Holding SHIFT+= does not inject '+' characters | TERM-04 | Requires real PTY + xterm canvas; jsdom cannot simulate | Hold SHIFT+= for 3 seconds, check no '+' in shell |
| Terminal reflows correctly after font change | TERM-02, TERM-03 | Requires real canvas to verify layout | Change font size, run `ls`, verify no clipping/garbling |
| Each tab retains independent font size | TERM-04 | Requires multi-tab UI with real terminal instances | Change font in tab 1, switch to tab 2, switch back, verify tab 1 retained size |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
