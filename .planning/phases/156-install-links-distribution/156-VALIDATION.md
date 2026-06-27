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
| 156-01-01 | 01 | 1 | INSTALL-01/02 | T-156-05/06 | Source-gate fails against unfixed strings (RED) | vitest | `cd frontend && pnpm vitest run src/components/__tests__/WelcomeTab.install.test.tsx` | ❌ W0 | ⬜ pending |
| 156-01-02 | 01 | 1 | INSTALL-01/02 | T-156-05/06 | Correct curl URL, scottkw.agenthub id, scottkw repo link; line 54 stays span | vitest | `cd frontend && pnpm vitest run src/components/__tests__/WelcomeTab.install.test.tsx` | ❌ W0 | ⬜ pending |
| 156-01-03 | 01 | 1 | INSTALL-02 | — | TESTING.md vitest 127 / Total 502 + traceability row | shell | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |
| 156-02-01 | 02 | 2 | INSTALL-01 | T-156-01/02/03 | shellcheck + static gate fails against missing script (RED) | shell | `bash tests/install-sh.test.sh` | ❌ W0 | ⬜ pending |
| 156-02-02 | 02 | 2 | INSTALL-01 | T-156-01/02/03 | POSIX installer SHA256-verifies before extract; hard-abort on mismatch/missing | shell | `bash tests/install-sh.test.sh && sh -n scripts/install.sh` | ❌ W0 | ⬜ pending |
| 156-02-03 | 02 | 2 | INSTALL-01 | — | CI gate wired ubuntu-latest; TESTING.md build-script 2 / Total 503 / M-25 | shell | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |
| 156-03-01 | 03 | 3 | INSTALL-03 | T-156-09 | Dry-run produces valid YAML with scottkw.agenthub + installer URL | shell | `bash packaging/winget/dry-run-first-submission.sh` | ❌ W0 | ⬜ pending |
| 156-03-02 | 03 | 3 | INSTALL-03 | T-156-07/08 | Operator runbook (least-priv PAT, post-acceptance reset) + M-26 | shell | `bash tests/check-traceability-paths.sh` | ✅ | ⬜ pending |

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
