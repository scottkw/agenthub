---
phase: 47
slug: homebrew-tap-packaging-templates
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-04
---

# Phase 47 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Shell scripts + brew audit (no test framework — YAML/Ruby config files) |
| **Config file** | none |
| **Quick run command** | `ruby -c packaging/homebrew/agenthub.rb.template` |
| **Full suite command** | `brew tap scottkw/agenthub && brew install --cask agenthub` (acceptance on macOS) |
| **Estimated runtime** | ~5 seconds (syntax check); ~60 seconds (full install) |

---

## Sampling Rate

- **After every task commit:** Run `ruby -c packaging/homebrew/agenthub.rb.template`
- **After every plan wave:** Verify distribute.yml syntax with `actionlint .github/workflows/distribute.yml` if available, else manual review
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 47-01-01 | 01 | 1 | DIST-04 | unit | `ruby -c packaging/homebrew/agenthub.rb.template` | ❌ W0 | ⬜ pending |
| 47-01-02 | 01 | 1 | DIST-04 | unit | `test -f packaging/winget/manifests/scottkw.agenthub.yaml` | ❌ W0 | ⬜ pending |
| 47-02-01 | 02 | 1 | DIST-02 | integration | `grep 'nick-fields/retry' .github/workflows/distribute.yml` | ❌ W0 | ⬜ pending |
| 47-02-02 | 02 | 1 | DIST-01 | smoke | `brew tap scottkw/agenthub && brew install --cask agenthub` | ❌ (tap repo) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `scottkw/homebrew-agenthub` repository created on GitHub (public, initialized with README)
- [ ] `TAP_DEPLOY_TOKEN` classic PAT created and stored as repository secret in scottkw/agenthub
- [ ] `packaging/homebrew/` and `packaging/winget/manifests/` directories created in main repo

*These are manual prerequisites that must exist before automated verification can run.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `brew install --cask agenthub` installs working app | DIST-01 | Requires macOS + published release + tap repo | 1. `brew tap scottkw/agenthub` 2. `brew install --cask agenthub` 3. Open agenthub.app 4. Verify app launches |
| distribute.yml auto-updates tap on release | DIST-02 | Requires actual GitHub Release event | 1. Create test release 2. Check Actions tab for distribute.yml run 3. Verify Casks/agenthub.rb updated in tap repo |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
