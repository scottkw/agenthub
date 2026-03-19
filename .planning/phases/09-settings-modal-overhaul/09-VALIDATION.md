---
phase: 9
slug: settings-modal-overhaul
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-19
---

# Phase 9 — Validation Strategy

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
| 09-01-01 | 01 | 1 | SETT-01 | unit | `pnpm test -- SettingsPanel` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | SETT-01 | unit | `pnpm test -- SettingsPanel` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | SETT-01 | unit | `pnpm test -- SettingsPanel` | ❌ W0 | ⬜ pending |
| 09-01-04 | 01 | 1 | SETT-02 | unit | `pnpm test -- SettingsPanel` | ❌ W0 | ⬜ pending |
| 09-01-05 | 01 | 1 | SETT-02 | unit | `pnpm test -- SettingsPanel` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/SettingsPanel.test.tsx` — stubs for SETT-01 and SETT-02

*Vitest, jsdom, and React testing infrastructure already configured. Only the test file itself is missing.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tab visual styling matches design tokens | SETT-02 | CSS appearance cannot be verified in jsdom | Visual inspection: tab underline is accent blue (#7aa2f7), inactive text is dim (#565f89) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
