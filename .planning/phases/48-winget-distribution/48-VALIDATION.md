---
phase: 48
slug: winget-distribution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-05
---

# Phase 48 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Shell commands (winget validate, grep, file existence checks) |
| **Config file** | none — no test framework needed |
| **Quick run command** | `test -f .github/workflows/distribute.yml && grep -q 'winget-releaser' .github/workflows/distribute.yml && echo "PASS"` |
| **Full suite command** | `grep -q 'winget-releaser' .github/workflows/distribute.yml && grep -q 'WINGET_TOKEN' .github/workflows/distribute.yml && echo "ALL PASS"` |
| **Estimated runtime** | ~1 second |

---

## Sampling Rate

- **After every task commit:** Run quick run command
- **After every plan wave:** Run full suite command
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 1 second

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 48-01-01 | 01 | 1 | DIST-03 | integration | `grep -q 'winget-releaser' .github/workflows/distribute.yml` | ❌ W0 | ⬜ pending |
| 48-02-01 | 02 | 1 | DIST-03 | manual | Manual: fork repo + submit PR to microsoft/winget-pkgs | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| First WinGet submission accepted | DIST-03 | Requires human PR to microsoft/winget-pkgs and moderator approval | Fork winget-pkgs, create manifest PR, wait for merge |
| winget install works | DIST-03 | Requires Windows machine with winget CLI | Run `winget install scottkw.agenthub` on Windows |
| distribute.yml WinGet job triggers | DIST-03 | Requires actual GitHub Release publish event | Publish a release, check Actions run |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 1s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
