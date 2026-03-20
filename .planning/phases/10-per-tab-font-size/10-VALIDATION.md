---
phase: 10
slug: per-tab-font-size
status: complete
nyquist_compliant: true
wave_0_complete: true
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
| 10-01-01 | 01 | 0 | TERM-02, TERM-03, TERM-04 | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ | ✅ green |
| 10-01-02 | 01 | 0 | TERM-04 | unit (source inspection) | `pnpm test -- App` | ✅ | ✅ green |
| 10-02-01 | 02 | 1 | TERM-04 | unit (source inspection) | `pnpm test -- App` | ✅ | ✅ green |
| 10-02-02 | 02 | 1 | TERM-02, TERM-03, TERM-04 | unit (source inspection) | `pnpm test -- TerminalPanel` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] New test cases in `frontend/src/components/__tests__/TerminalPanel.test.tsx` — 10 source-inspection tests for TERM-02, TERM-03, TERM-04
- [x] New test file `frontend/src/components/__tests__/App.test.tsx` — 7 source-inspection tests for TERM-04 (fontSizes state in App.tsx)

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved

## Validation Audit 2026-03-20

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
