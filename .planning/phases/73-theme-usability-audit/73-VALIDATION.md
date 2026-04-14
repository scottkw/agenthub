---
phase: 73
slug: theme-usability-audit
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
---

# Phase 73 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.0 |
| **Config file** | `frontend/vite.config.ts` (test block present) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsTab` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test -- SettingsTab`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 73-01-01 | 01 | 1 | THM-04 | — | N/A | unit | `pnpm test -- SettingsTab` | ✅ exists — needs update | ⬜ pending |
| 73-01-02 | 01 | 1 | THM-04 | — | N/A | unit | `pnpm test -- SettingsTab` | ❌ W0 | ⬜ pending |
| 73-01-03 | 01 | 1 | THM-04 | — | N/A | unit | `pnpm test -- SettingsTab` | ❌ W0 | ⬜ pending |
| 73-01-04 | 01 | 1 | THM-04 | — | N/A | unit | `pnpm test -- SettingsTab` | ❌ W0 | ⬜ pending |
| 73-01-05 | 01 | 1 | THM-04 | — | N/A | unit | `pnpm test -- SettingsTab` | ❌ W0 | ⬜ pending |
| 73-01-06 | 01 | 1 | THM-04 | — | N/A | unit | `pnpm test -- SettingsTab` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/__tests__/SettingsTab.test.tsx` — update 2 existing assertions that reference `Object.keys(xtermThemes).sort()` pattern
- [ ] `frontend/src/__tests__/SettingsTab.test.tsx` — add THM-04 describe block with 5 new assertions
- [ ] Locate `selectedTheme` localStorage initialization — add fallback guard test

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual readability of surviving themes in each agent | THM-04 | Requires human judgment on contrast/readability | Spot-check 3-5 themes in each of Claude Code, Codex, Gemini CLI, OpenCode |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
