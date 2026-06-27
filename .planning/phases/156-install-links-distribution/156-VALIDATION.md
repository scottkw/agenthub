---
phase: 156
slug: install-links-distribution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-26
---

# Phase 156 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) + shellcheck/bash (scripts) |
| **Config file** | frontend/vitest.config.ts; TESTING.md suite manifest |
| **Quick run command** | `cd frontend && pnpm vitest run src/components/__tests__/WelcomeTab.install.test.tsx` |
| **Full suite command** | `bash tests/install-sh.test.sh && cd frontend && pnpm vitest run` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command for the touched surface
- **After every plan wave:** Run full suite command
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | INSTALL-XX | — | {expected behavior} | {unit/shell} | `{command}` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · Planner fills this map from the final PLAN.md task IDs.*

---

## Wave 0 Requirements

- [ ] `tests/install-sh.test.sh` — shellcheck + behavior stubs for `scripts/install.sh` (INSTALL-01)
- [ ] `frontend/src/components/__tests__/WelcomeTab.install.test.tsx` — asserts corrected install strings + repo link (INSTALL-01/02)

*Planner confirms whether existing infrastructure covers these or Wave 0 creates them.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Linux `curl … \| sh` installs runnable `agenthub` on a clean box | INSTALL-01 | Needs a clean Linux env (physical or `ubuntu:22.04` Docker); real GitHub-release fetch + SHA256 verify | TESTING.md M-25 |
| winget first-submission dry-run + operator runbook | INSTALL-03 | Live submission gated on external `microsoft/winget-pkgs` review | TESTING.md M-26 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
