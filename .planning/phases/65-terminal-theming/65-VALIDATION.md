---
phase: 65
slug: terminal-theming
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 65 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest ^4.1.0 |
| **Config file** | `frontend/vite.config.ts` (test.environment: jsdom) |
| **Quick run command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Full suite command** | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **After every plan wave:** Run `cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 65-01-01 | 01 | 1 | THM-01 | — | N/A | unit | `pnpm test` | ✅ SettingsTab.test.tsx | ⬜ pending |
| 65-01-02 | 01 | 1 | THM-02 | — | N/A | unit | `pnpm test` | ✅ App.test.tsx | ⬜ pending |
| 65-01-03 | 01 | 1 | THM-03 | — | N/A | unit | `pnpm test` | ✅ TerminalPanel.test.tsx | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test files needed; assertions are added to existing test files.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Theme colors render visually correct | THM-01 | Visual appearance cannot be verified by unit tests | Open Settings > Appearance, select a theme, verify terminal colors change visually |
| Theme persists after full app restart | THM-02 | Requires actual app restart cycle | Select a theme, quit app fully, reopen, verify same theme is active |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
