---
phase: 90
slug: release-pipeline-hardening
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-23
---

# Phase 90 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bats / bash (build.sh tests) + GitHub Actions workflow dispatch (CI-side verification) |
| **Config file** | `tests/build-script.test.sh` (existing); grep-gate script location — Claude's discretion |
| **Quick run command** | `bash tests/build-script.test.sh && bash scripts/grep-gate.sh` (script name TBD by planner) |
| **Full suite command** | Quick run + `gh workflow run build.yml` on phase branch (integration) |
| **Estimated runtime** | Quick ~5s; full suite ~10–15m (CI dispatch) |

---

## Sampling Rate

- **After every task commit:** Run quick run command (grep gate + build-script tests)
- **After every plan wave:** Run full suite (quick + workflow dispatch)
- **Before `/gsd-verify-work`:** E2E tag cut (`v3.1.0-rc1`) must produce all final artifacts + attestations
- **Max feedback latency:** ~5s (quick); ~15m (integration)

---

## Per-Task Verification Map

Populated by `gsd-planner` — every task must list an `<automated>` check or flag a Wave 0 dependency.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 90-XX-YY | ?? | ? | SEC-09 / SEC-10 / SEC-11 | T-90-XX | {behavior} | unit/integration/e2e | `{command}` | ✅ / ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `scripts/grep-gate.sh` (or equivalent) — asserts zero `@main` / `@master` / `@latest` in `.github/workflows/` and `build.sh`
- [ ] `tests/build-script.test.sh` extensions — cover the new `go install X@$(go list -m ...)` pattern (install hint at build.sh:65 + real install path)
- [ ] `release-90-test` branch pre-created in `scottkw/homebrew-agenthub` for rc tap verification (D-16)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Signed + notarized DMG opens without Gatekeeper prompt | SEC-11 (E2E) | Requires actual macOS install | Download `v3.1.0-rc1` DMG, open on clean Mac, confirm no `unidentified developer` dialog |
| Homebrew tap install resolves against `release-90-test` branch | SC-4 | Requires `brew` workflow against tap branch | `brew tap scottkw/agenthub --custom-remote https://github.com/scottkw/homebrew-agenthub.git --branch release-90-test && brew install agenthub` |
| `gh attestation verify <file> --owner scottkw` passes for every release asset | SEC-11 (D-05) | External verification from clean client | Download every `v3.1.0-rc1` asset; run `gh attestation verify` on each; confirm "Verification succeeded" |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (grep-gate, build-script.test extensions, tap test branch)
- [ ] No watch-mode flags
- [ ] Feedback latency < 15m (integration ceiling)
- [ ] `nyquist_compliant: true` set in frontmatter (planner will update after per-task map is populated)

**Approval:** pending
