---
phase: 46
slug: release-build-pipeline
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-04
---

# Phase 46 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bash smoke tests + python yaml validation |
| **Config file** | none — Wave 0 installs |
| **Quick run command** | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` |
| **Full suite command** | `go test -race ./... && python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green + merge v1.8.0 Release PR for e2e validation
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 46-01-01 | 01 | 1 | REL-02 | smoke | `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` | ❌ W0 | ⬜ pending |
| 46-01-02 | 01 | 1 | REL-02 | smoke | `python3 -c "import yaml; d=yaml.safe_load(open('.github/workflows/release.yml')); assert 'v*' in str(d['on'])"` | ❌ W0 | ⬜ pending |
| 46-01-03 | 01 | 1 | REL-02 | smoke | `grep -c 'environment: release' .github/workflows/release.yml` | ❌ W0 | ⬜ pending |
| 46-01-04 | 01 | 1 | REL-02 | smoke | `grep -c 'MACOS_' .github/workflows/release.yml` (expect >= 7) | ❌ W0 | ⬜ pending |
| 46-01-05 | 01 | 1 | REL-04 | smoke | `grep -c 'checksums.txt' .github/workflows/release.yml` | ❌ W0 | ⬜ pending |
| 46-01-06 | 01 | 1 | REL-04 | smoke | `grep -c 'sha256sum' .github/workflows/release.yml` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `.github/workflows/release.yml` — new file, covers REL-02 and REL-04
- [ ] PyYAML available for YAML validation: `pip install pyyaml` or use `ruby -ryaml -e "YAML.load_file(...)"`

*Existing infrastructure covers Go test requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| End-to-end: merge Release PR triggers release.yml and produces artifacts | REL-02 | Requires actual GitHub Actions run with secrets | Merge the open v1.8.0 Release PR; verify all 6 artifacts + checksums.txt appear on the GitHub Release page |
| macOS DMG is signed and notarized — Gatekeeper allows installation | REL-02 | Requires downloading and running on macOS | Download DMG from GitHub Release; open it; verify macOS does not show security override prompt |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
