---
phase: 90
slug: release-pipeline-hardening
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-23
updated: 2026-04-23
---

# Phase 90 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bash (tests/build-script.test.sh) + grep-gate script (Wave 0) + YAML validation + GitHub Actions workflow dispatch (integration + E2E) |
| **Config file** | `tests/build-script.test.sh` (existing; extended in Plan 01) + `scripts/grep-gate.sh` (new in Plan 01) |
| **Quick run command** | `bash tests/build-script.test.sh && bash scripts/grep-gate.sh` |
| **Full suite command** | Quick + `go build -tags tools ./... && go test -race -short ./... && for f in .github/workflows/*.yml .github/dependabot.yml; do python3 -c "import yaml; yaml.safe_load(open('$f'))" || exit 1; done` |
| **Phase gate** | Full suite green + v3.1.0-rc1 tag cut succeeds end-to-end per Plan 06 |
| **Estimated runtime** | Quick ~5s; full suite ~30s local; E2E ~15-20m CI |

---

## Sampling Rate

- **After every task commit:** Run quick run command (grep-gate + build-script tests)
- **After every plan wave:** Run full suite (quick + YAML validation + `go build -tags tools`)
- **Before `/gsd-verify-work`:** E2E tag cut (`v3.1.0-rc1`) must produce all final artifacts + external `gh attestation verify` passes on all 6 assets
- **Max feedback latency:** ~5s (quick); ~30s (full local); ~15-20m (CI E2E)

---

## Per-Task Verification Map

Populated by `gsd-planner`. Every task has an automated command (or explicit Wave 0 dependency).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 90-01-01 | 01 | 0 | SEC-09, SEC-10 | T-90-01, T-90-02 | grep-gate script rejects floating refs, @latest, non-SHA refs | static | `test -x scripts/grep-gate.sh && bash -n scripts/grep-gate.sh && grep -c 'grep -rE' scripts/grep-gate.sh | awk '{exit $1 < 2}'` | ✅ in-plan | ⬜ pending |
| 90-01-02 | 01 | 0 | SEC-10 | T-90-03 | build-script.test.sh asserts @latest absent + WAILS_PINNED_VER present; tap branch runbook documented | static + runbook | `bash tests/build-script.test.sh 2>&1 | grep 'Section 12' && test -f .planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` | ✅ in-plan | ⬜ pending |
| 90-02-01 | 02 | 1 | SEC-10 | T-90-05, T-90-08 | tools.go + go.mod with wails v2.12.0 + nfpm; go build -tags tools green | integration | `test -f tools.go && grep -F 'github.com/wailsapp/wails/v2 v2.12.0' go.mod && grep -F 'github.com/goreleaser/nfpm/v2' go.mod && go build -tags tools ./...` | ✅ in-plan | ⬜ pending |
| 90-02-02 | 02 | 1 | SEC-09 | T-90-07 | dependabot.yml has github-actions + gomod ecosystems, no auto-merge field | static | `grep -F 'package-ecosystem: "github-actions"' .github/dependabot.yml && grep -F 'package-ecosystem: "gomod"' .github/dependabot.yml && ! grep -q 'auto-merge' .github/dependabot.yml` | ✅ in-plan | ⬜ pending |
| 90-03-01 | 03 | 2 | SEC-09, SEC-10 | T-90-11, T-90-13 | build.yml fully SHA-pinned; wails install via go-list pattern with Pitfall-5 gate | static | `grep -c '@latest' .github/workflows/build.yml | awk '{exit $1 > 0}' && grep -E 'uses:\s*[^ ]+@' .github/workflows/build.yml | grep -Ev '@[a-f0-9]{40}(\s\|$)' | wc -l | awk '{exit $1 > 0}'` | ✅ in-plan | ⬜ pending |
| 90-03-02 | 03 | 2 | SEC-09 | T-90-11 | release-please.yml SHA-pinned to release-please-action v5.0.0 | static | `grep -c 'googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0' .github/workflows/release-please.yml | awk '{exit $1 != 1}'` | ✅ in-plan | ⬜ pending |
| 90-03-03 | 03 | 2 | SEC-10 | T-90-15, T-90-16 | build.sh has zero @latest + WAILS_PINNED_VER gate; build-script tests green | behavioral | `grep -c '@latest' build.sh | awk '{exit $1 > 0}' && grep -F 'WAILS_PINNED_VER' build.sh && bash tests/build-script.test.sh` | ✅ in-plan | ⬜ pending |
| 90-04-01 | 04 | 3 | SEC-09, SEC-10, SEC-11 | T-90-17, T-90-18, T-90-19 | build-* jobs secret-free; SHA-pinned; attestations present; .app tarred | static | `! grep -A1 'build-macos:' .github/workflows/release.yml | head -30 | grep -q 'MACOS_CERTIFICATE' && grep -F 'tar -czf build/bin/AgentHub.app.tar.gz' .github/workflows/release.yml && grep -c 'attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32' .github/workflows/release.yml | awk '{exit $1 < 3}'` | ✅ in-plan | ⬜ pending |
| 90-04-02 | 04 | 3 | SEC-11 | T-90-18, T-90-21 | sign-macos job exists with environment: release, MACOS_* env only, attestation verify + untar | static | `grep -F 'sign-macos:' .github/workflows/release.yml && grep -c 'environment: release' .github/workflows/release.yml | awk '{exit $1 != 1}' && grep -F 'gh attestation verify artifacts/AgentHub.app.tar.gz' .github/workflows/release.yml` | ✅ in-plan | ⬜ pending |
| 90-04-03 | 04 | 3 | SEC-09, SEC-11 | T-90-22, T-90-25 | publish uses GITHUB_TOKEN (not TAP_DEPLOY_TOKEN), release attestation, hyphen-anchored rc draft | static | `! grep -q 'TAP_DEPLOY_TOKEN' .github/workflows/release.yml && grep -F 'token: ${{ secrets.GITHUB_TOKEN }}' .github/workflows/release.yml && grep -F "draft: \${{ contains(github.ref, '-rc') }}" .github/workflows/release.yml && grep -F 'subject-checksums: artifacts/checksums.txt' .github/workflows/release.yml` | ✅ in-plan | ⬜ pending |
| 90-05-01 | 05 | 4 | SEC-09 | T-90-28, T-90-30 | update-homebrew-tap SHA-pinned; rc-aware branch selection in checkout + git push | static | `grep -F 'nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0' .github/workflows/distribute.yml && grep -c "contains(github.ref, '-rc') && 'release-90-test' || 'main'" .github/workflows/distribute.yml | awk '{exit $1 < 2}'` | ✅ in-plan | ⬜ pending |
| 90-05-02 | 05 | 4 | SEC-09, SEC-11 | T-90-26, T-90-27, T-90-29 | submit-winget on windows-latest with SHA-256-verified wingetcreate; rc skip; zero floating refs | static | `! grep -q 'vedantmgoyal9/winget-releaser' .github/workflows/distribute.yml && grep -F '8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D' .github/workflows/distribute.yml && grep -F "if: \${{ !contains(github.ref, '-rc') }}" .github/workflows/distribute.yml && bash scripts/grep-gate.sh` | ✅ in-plan | ⬜ pending |
| 90-06-01 | 06 | 5 | SEC-09, SEC-10, SEC-11 | T-90-34, T-90-38 | Pre-flight: scripts green, tap branch created, env rules audited, build green | checkpoint:human-action | manual verification per plan instructions | ⚠️ human | ⬜ pending |
| 90-06-02 | 06 | 5 | SEC-11 | T-90-18, T-90-34 | v3.1.0-rc1 tag cut produces 6 assets in a DRAFT release; sign-macos internal verification green | checkpoint:human-action + observation | `gh release view v3.1.0-rc1 --json isDraft --jq .isDraft` returns `true` AND `gh release view v3.1.0-rc1 --json assets --jq '.assets \| length'` returns 6 | ⚠️ requires tag cut | ⬜ pending |
| 90-06-03 | 06 | 5 | SEC-11 | T-90-34, T-90-40 | External gh attestation verify green for all 6 assets; distribute.yml targets release-90-test branch + skips submit-winget | checkpoint:human-verify | `for f in agenthub-*.dmg agenthub-*.exe agenthub-*.tar.gz agenthub-*.deb checksums.txt; do gh attestation verify "$f" --owner scottkw; done` exits 0 for all + `gh run view <distribute-run-id> --json jobs --jq '.jobs[] \| select(.name=="submit-winget").conclusion'` returns `"skipped"` | ⚠️ requires Steps 2-7 | ⬜ pending |

*Status legend: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky or requires human*

---

## Wave 0 Requirements

- [ ] `scripts/grep-gate.sh` — asserts zero `@main` / `@master` / `@latest` in `.github/workflows/`, `build.sh`, and `tests/` (Plan 01 Task 1)
- [ ] `tests/build-script.test.sh` — Section 12 extensions for SEC-10 compliance + WAILS_PINNED_VER pattern (Plan 01 Task 2)
- [ ] `.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` — runbook for D-16 prerequisite (Plan 01 Task 2)
- [ ] `release-90-test` branch pre-created in `scottkw/homebrew-agenthub` — **MANUAL** step per runbook; executed during Plan 06 Task 1 pre-flight

---

## Manual-Only Verifications

These require human observation or operations on external systems.

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Signed + notarized DMG opens without Gatekeeper prompt | SEC-11 E2E | Requires clean macOS Gatekeeper state | Download `v3.1.0-rc1` DMG from draft release; open on a Mac that has never installed agenthub; confirm no "unidentified developer" dialog; confirm app launches cleanly. |
| Homebrew tap install resolves against `release-90-test` branch | SC-4 / D-16 | Requires `brew` workflow against tap branch | `brew tap scottkw/agenthub --custom-remote https://github.com/scottkw/homebrew-agenthub.git --branch release-90-test && brew install agenthub` — expect successful install of the rc build. |
| `gh attestation verify <file> --owner scottkw` passes for every release asset | SEC-11 D-05 | External verification from clean client | In a fresh shell on any machine with `gh` CLI: download every v3.1.0-rc1 asset; run `gh attestation verify` on each; expect "verification succeeded" per asset. Plan 06 Task 3 Step 2. |
| `submit-winget` job concluded "skipped" on rc run | SEC-09 D-17 | Requires workflow run history inspection | `gh run view <distribute-run-id> --json jobs --jq '.jobs[] \| select(.name=="submit-winget").conclusion'` returns `"skipped"`. |
| Repository environment `release` protection rules preserved after job re-scope | SC-3 / Pitfall 4 | GitHub UI / API-only inspection | `gh api /repos/scottkw/agenthub/environments/release` output before vs after — any required-reviewer / wait-timer rule must now gate `sign-macos` specifically. Plan 06 Task 1 Step 3. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — populated in Per-Task Verification Map
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (every task has an automated check; the three human-checkpoint tasks in Plan 06 are the correct exceptions for E2E verification)
- [x] Wave 0 covers all MISSING references (grep-gate, build-script.test extensions; tap-branch doc is Wave 0 artifact, branch creation is Plan 06 pre-flight)
- [x] No watch-mode flags in CI commands
- [x] Feedback latency < 15m for integration; < 20m for E2E (rc cut)
- [x] `nyquist_compliant: true` set in frontmatter (updated 2026-04-23 after plan completion)

**Approval:** planner-approved 2026-04-23; pending human sign-off at end of Plan 06.
