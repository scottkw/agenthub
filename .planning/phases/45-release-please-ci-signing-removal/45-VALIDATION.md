---
phase: 45
slug: release-please-ci-signing-removal
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-04
---

# Phase 45 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bash (tests/build-script.test.sh) + go test |
| **Config file** | none — invoked directly |
| **Quick run command** | `python3 -m json.tool release-please-config.json && python3 -m json.tool .release-please-manifest.json` |
| **Full suite command** | `go test -race ./... && bash tests/build-script.test.sh` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `python3 -m json.tool release-please-config.json && python3 -m json.tool .release-please-manifest.json`
- **After every plan wave:** Run `go test -race ./... && bash tests/build-script.test.sh`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 45-01-01 | 01 | 1 | REL-01 | smoke | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release-please.yml'))"` | ❌ W0 | ⬜ pending |
| 45-01-02 | 01 | 1 | REL-01 | smoke | `python3 -m json.tool release-please-config.json` | ❌ W0 | ⬜ pending |
| 45-01-03 | 01 | 1 | REL-01 | smoke | `python3 -c "import json; d=json.load(open('.release-please-manifest.json')); assert d['.']=='1.7.0'"` | ❌ W0 | ⬜ pending |
| 45-02-01 | 02 | 1 | REL-03 | smoke | `grep -c "MACOS_" .github/workflows/build.yml; test $? -eq 1` | ✅ | ⬜ pending |
| 45-02-02 | 02 | 1 | REL-03 | smoke | `grep -c "codesign\|notarytool" .github/workflows/build.yml; test $? -eq 1` | ✅ | ⬜ pending |
| 45-02-03 | 02 | 1 | REL-03 | unit | `go test -race ./...` | ✅ | ⬜ pending |
| 45-02-04 | 02 | 1 | REL-03 | unit | `bash tests/build-script.test.sh` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] No new test files needed — validation is structural (JSON/YAML lint) and runtime (CI trigger test)
- [ ] Smoke commands for YAML validity use PyYAML (`python3 -c "import yaml"`) — verify available in venv

*Existing infrastructure covers most phase requirements. New files (release-please configs) validated via JSON/YAML linting.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Release PR appears after conventional-commit push to main | REL-01 | Requires real GitHub Actions trigger — not simulatable locally | Push a `fix: test release-please` commit to main; verify Release PR appears within ~60s |
| Release PR shows correct SemVer (v1.8.0) | REL-01 | Requires release-please state machine on GitHub | Verify Release PR title contains `1.8.0` |
| RELEASE_PLEASE_TOKEN PAT created and stored | REL-01 | Requires GitHub web UI (PAT creation) | Verify `gh api repos/scottkw/agenthub/actions/secrets` shows RELEASE_PLEASE_TOKEN |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
