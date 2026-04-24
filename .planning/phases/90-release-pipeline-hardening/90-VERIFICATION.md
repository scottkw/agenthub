---
phase: 90-release-pipeline-hardening
verified: 2026-04-24T14:00:00Z
status: human_needed
score: 14/15 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Push v3.1.0-rc1 tag and observe release.yml end-to-end: validate → build-macos/build-windows/build-linux → sign-macos → publish"
    expected: "All jobs green; sign-macos 'Verify internal attestation' step shows Sigstore signature verified before codesign runs; draft release created with isDraft=true; 6 assets (DMG, installer.exe, .exe, tar.gz, .deb, checksums.txt) present"
    why_human: "Requires live git tag push, GitHub Actions runner execution with real signing secrets, and visual inspection of job logs — cannot be verified statically"
  - test: "Confirm gh attestation verify succeeds for all 6 release assets from a clean client"
    expected: "Each file outputs 'Loaded N attestation(s) ... verification succeeded' when run against the draft release assets"
    why_human: "Requires downloading live binaries from a draft GitHub release and invoking Sigstore's public-good instance — not automatable without executing a live release"
  - test: "Trigger distribute.yml (via workflow_dispatch) for the rc tag and confirm rc semantics"
    expected: "update-homebrew-tap checks out release-90-test branch (not main); git push targets release-90-test; submit-winget job shows conclusion=skipped"
    why_human: "Requires a live workflow_dispatch trigger, cross-repo tap observation, and WinGet job inspection — cannot be confirmed without a running Actions environment"
  - test: "Confirm scottkw/homebrew-agenthub main branch is untouched after distribute.yml rc run"
    expected: "'git log origin/main..origin/release-90-test --oneline' shows 1 new commit on release-90-test; main's last commit predates the rc push"
    why_human: "Requires access to scottkw/homebrew-agenthub repository state after a live workflow run"
  - test: "Create and sign off 90-UAT-REPORT.md with human reviewer name, date, and PASS/FAIL"
    expected: ".planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md exists and contains signed human sign-off"
    why_human: "Artifact produced by human reviewer after completing all UAT steps — cannot be generated autonomously"
---

# Phase 90: Release Pipeline Hardening Verification Report

**Phase Goal:** Security hardening of the release pipeline — SHA-pin all GitHub Actions, pin build tools via tools.go + go.mod, split release.yml to scope secrets, replace third-party @main action with first-party wingetcreate, and establish E2E RC verification path.
**Verified:** 2026-04-24T14:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A grep-gate script exists that fails CI if any workflow contains @main, @master, @latest, or non-40-char-SHA action refs | VERIFIED | `bash scripts/grep-gate.sh` exits 0; script is executable at `scripts/grep-gate.sh` |
| 2 | The build.sh test suite has 38 passing assertions including Section 12 SEC-10 compliance (no @latest, go list pattern, WAILS_PINNED_VER gate) | VERIFIED | `bash tests/build-script.test.sh` outputs "Results: 38 passed, 0 failed" |
| 3 | A documented manual step exists for pre-creating release-90-test branch in scottkw/homebrew-agenthub before E2E verification | VERIFIED | `90-TAP-BRANCH-SETUP.md` exists with 6 occurrences of "release-90-test" and `gh repo clone scottkw/homebrew-agenthub` instructions |
| 4 | The project declares CI build tools (wails, nfpm) in go.mod via tools.go blank-import pattern; both v2.12.0 and v2.33.1 appear in go.mod | VERIFIED | `tools.go` has `//go:build tools`, imports `goreleaser/nfpm/v2` and `wailsapp/wails/v2`; go.mod has `wailsapp/wails/v2 v2.12.0` and `goreleaser/nfpm/v2 v2.33.1` |
| 5 | Dependabot is configured with weekly PRs for github-actions and gomod ecosystems, no auto-merge | VERIFIED | `.github/dependabot.yml` has 2 package-ecosystem entries; `grep -c "auto-merge"` returns 0 |
| 6 | build.yml has zero @vN/@main/@master/@latest refs — every uses: is SHA-pinned with # vX.Y.Z comment | VERIFIED | 8 SHA-pin comments in build.yml; grep-gate exits 0 for build.yml; no @latest |
| 7 | release-please.yml uses googleapis/release-please-action pinned to SHA 45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0 | VERIFIED | Plan 03 committed SHA; grep-gate passes across all workflows |
| 8 | build.sh no longer references @latest; install hint uses go list -m pattern with WAILS_PINNED_VER gate | VERIFIED | `grep -c "@latest" build.sh` returns 0; `WAILS_PINNED_VER` appears 6 times; Section 12 all green |
| 9 | release.yml build-macos/build-windows/build-linux jobs have zero MACOS_*, WINGET_TOKEN, TAP_DEPLOY_TOKEN, or environment: release — secret-free build stage | VERIFIED | MACOS_ first appears at line 292 (sign-macos job starts at line 285); build-macos at line 43 has no env block; TAP_DEPLOY_TOKEN: 0 matches in release.yml |
| 10 | release.yml has sign-macos job with environment: release holding ONLY MACOS_* secrets, verifying attestation before signing | VERIFIED | `environment: release` appears exactly once (line 288, sign-macos); `gh attestation verify artifacts/AgentHub.app.tar.gz` present; 7 MACOS_* secrets in env block |
| 11 | release.yml publish job depends on sign-macos, uses GITHUB_TOKEN (not TAP_DEPLOY_TOKEN), attests every asset via subject-checksums, and marks release draft when tag matches v*-rc* | VERIFIED | `needs: [build-macos, build-windows, build-linux, sign-macos]` at line 391; `token: secrets.GITHUB_TOKEN`; `subject-checksums: artifacts/checksums.txt` at line 412; `draft: contains(github.ref, '-rc')` hyphen-anchored at line 424 |
| 12 | macOS .app bundle is tar -czf'd before upload and untar'd before sign — symlinks and +x bits survive cross-job handoff | VERIFIED | `tar -czf build/bin/AgentHub.app.tar.gz` at release.yml line 94; `tar -xzf artifacts/AgentHub.app.tar.gz` at sign-macos untar step |
| 13 | distribute.yml has zero floating refs; vedantmgoyal9/winget-releaser@main replaced by inline wingetcreate.exe with SHA-256 verification | VERIFIED | `grep -E 'uses:.*@' distribute.yml \| grep -Ev '@[a-f0-9]{40}'` returns empty; wingetcreate SHA-256 `8BD738851B...` present; `vedantmgoyal9` count is 0 |
| 14 | distribute.yml submit-winget skips on rc tags; update-homebrew-tap routes to release-90-test on rc | VERIFIED | `if: !contains(github.ref, '-rc')` on submit-winget; checkout `ref:` and `git push origin HEAD:"$BRANCH"` both use rc ternary; `release-90-test` appears in 2 ternary expressions |
| 15 | A real v3.1.0-rc1 tag has been pushed, release workflow ran with sign-macos verifying attestation, all artifacts are externally gh-attestation-verifiable, distribute.yml targeted release-90-test only, 90-UAT-REPORT.md signed off | HUMAN NEEDED | Plan 06 is autonomous: false; requires live tag push, GitHub Actions runner execution, and human UAT sign-off. See 90-06-HUMAN-UAT.md for 15-item checklist. |

**Score:** 14/15 truths verified (Truth 15 is Plan 06's human-gated E2E verification)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `scripts/grep-gate.sh` | SEC-09+SEC-10 regression guard | VERIFIED | Executable; exits 0 against current pinned repo; 3 independent grep checks |
| `tests/build-script.test.sh` | Section 12 SEC-10 assertions | VERIFIED | Section 12 present; 38/38 passing including all 3 Section 12 assertions |
| `.planning/phases/90-release-pipeline-hardening/90-TAP-BRANCH-SETUP.md` | D-16 manual prerequisite runbook | VERIFIED | File exists; contains release-90-test branch creation commands |
| `tools.go` | Build-tool dependency manifest | VERIFIED | `//go:build tools`, `// +build tools`, blank imports for nfpm+wails |
| `go.mod` | Pinned wails v2.12.0 + nfpm v2.33.1 | VERIFIED | Both versions confirmed present |
| `go.sum` | Cryptographic pins | VERIFIED | go.sum updated by `go mod tidy`; `go build .` exits 0 |
| `.github/dependabot.yml` | Weekly PRs for github-actions + gomod | VERIFIED | 2 ecosystems, weekly schedule, no auto-merge, ungrouped |
| `.github/workflows/build.yml` | SHA-pinned; go-list wails install | VERIFIED | 8 SHA-pin comments; no @latest; WAILS_VER go list pattern present |
| `.github/workflows/release-please.yml` | SHA-pinned to googleapis@45996ed | VERIFIED | grep-gate passes; no floating refs |
| `build.sh` | go list pattern; WAILS_PINNED_VER gate | VERIFIED | 0 @latest; 6 WAILS_PINNED_VER occurrences; syntax valid |
| `.github/workflows/release.yml` | Split pipeline; SLSA L2 attestations; SHA-pinned | VERIFIED | 27 SHA-pin comments; 4x attest-build-provenance; environment: release on sign-macos only; 0 TAP_DEPLOY_TOKEN |
| `.github/workflows/distribute.yml` | wingetcreate; rc-aware routing; SHA-pinned | VERIFIED | 2 uses: lines both SHA-pinned; wingetcreate SHA-256 verified; 3 rc-ternary expressions |
| `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md` | Signed E2E verification record | MISSING (expected) | Requires human Plan 06 execution; tracked in 90-06-HUMAN-UAT.md |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `scripts/grep-gate.sh` | `.github/workflows/*.yml + build.sh` | grep -rE pattern matching | WIRED | Exit 0 confirmed; `\w@latest` pattern excludes test string literals |
| `tests/build-script.test.sh Section 12` | `build.sh` | grep assertions | WIRED | All 3 assertions pass: @latest absent, go list pattern present, WAILS_PINNED_VER present |
| `tools.go` | `go.mod` | blank imports trigger go mod tidy | WIRED | nfpm v2.33.1 and wails v2.12.0 in go.mod require block |
| `.github/dependabot.yml` | `go.mod + .github/workflows/*` | gomod + github-actions ecosystems | WIRED | Both ecosystems confirmed in file |
| `build.yml wails install step` | `go.mod via go list -m` | shell substitution | WIRED | `go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2` present in build.yml |
| `build.sh WAILS_PINNED_VER block` | `go.mod` | go list -m sanity gate | WIRED | Pattern present; empty-string check aborts if wails not in go.mod |
| `build-macos tar step` | `sign-macos untar step` | actions/upload-artifact v7 + download-artifact v8 with tarball | WIRED | `tar -czf` at line 94; `tar -xzf` in sign-macos; artifact name `agenthub-darwin-universal-unsigned` consistent |
| `build-* attest step` | `sign-macos gh attestation verify step` | bundle-path output uploaded + gh attestation verify --bundle | WIRED | 4x attest-build-provenance in uses: lines; `gh attestation verify artifacts/AgentHub.app.tar.gz` in sign-macos |
| `publish job` | `softprops/action-gh-release@v3.0.0` | `token: secrets.GITHUB_TOKEN` + `draft: contains(github.ref, '-rc')` | WIRED | SHA `b4309332981a82ec1c5618f44dd2e27cc8bfbfda` confirmed; draft guard hyphen-anchored |
| `submit-winget job` | `microsoft/winget-create v1.12.8.0` | `Invoke-WebRequest + Get-FileHash SHA-256 verification` | WIRED | SHA-256 `8BD738851B...D85D` present; `Invoke-WebRequest` confirmed; `if: !contains(github.ref, '-rc')` present |
| `update-homebrew-tap checkout + push` | `scottkw/homebrew-agenthub branch selection` | contains(github.ref, '-rc') ternary | WIRED | checkout `ref:` ternary at distribute.yml:52; `git push origin HEAD:"$BRANCH"` at distribute.yml:66 |
| `v3.1.0-rc1 tag push` | `release.yml full run → distribute.yml rc behavior` | live GitHub Actions trigger | NOT YET WIRED | Requires human Plan 06 execution |

### Data-Flow Trace (Level 4)

Not applicable — this phase delivers CI workflow YAML and Go module tooling configuration, not components that render dynamic data. No UI data-flow tracing required.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| grep-gate exits 0 across all workflows | `bash scripts/grep-gate.sh; echo EXIT:$?` | EXIT:0 | PASS |
| Build test suite 38/38 | `bash tests/build-script.test.sh 2>&1 \| tail -5` | Results: 38 passed, 0 failed | PASS |
| Go binary builds cleanly | `go build .; echo EXIT:$?` | EXIT:0 | PASS |
| Go vet passes | `go vet .; echo EXIT:$?` | EXIT:0 | PASS |
| build.yml has 8+ SHA-pin comments | `grep -c "# v" build.yml` | 8 | PASS |
| release.yml has 11+ SHA-pin comments | `grep -c "# v" release.yml` | 27 | PASS |
| distribute.yml has 2 SHA-pinned uses: lines (all uses:) | `grep -c "# v" distribute.yml` | 2 | PASS (wingetcreate is inline shell — no further uses: lines needed) |
| environment: release exactly once (sign-macos) | `grep "environment: release" release.yml` | 1 match at line 288 | PASS |
| MACOS_ refs absent outside sign-macos | `grep "MACOS_" release.yml \| grep -v sign-macos \| head` | empty (all at line 292+ which is inside sign-macos) | PASS |
| wingetcreate present | `grep "wingetcreate" distribute.yml \| head -3` | 3 matching lines | PASS |
| wingetcreate SHA-256 hash | `grep "8BD738851B..." distribute.yml` | match | PASS |
| tools.go exists with build tag | `test -f tools.go && head -5 tools.go` | `//go:build tools` present | PASS |
| wails v2.12.0 in go.mod | `grep "wailsapp/wails/v2" go.mod` | `v2.12.0` | PASS |
| nfpm v2.33.1 in go.mod | `grep "goreleaser/nfpm/v2" go.mod` | `v2.33.1` | PASS |
| E2E rc verification run | requires live tag push | N/A | SKIP (human_needed) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SEC-09 | Plans 01, 02, 03, 04, 05 | All third-party GitHub Actions pinned to immutable commit SHAs | SATISFIED (static) | grep-gate exits 0; all 4 workflow files pass 40-char SHA check; vedantmgoyal9@main eliminated |
| SEC-10 | Plans 01, 02, 03 | Go build tools pinned to exact versions, no @latest | SATISFIED (static) | build.sh has 0 @latest; build.yml has 0 @latest; tools.go + go.mod pin wails v2.12.0 and nfpm v2.33.1 |
| SEC-11 | Plan 04 | Release pipeline restructured so unsigned build cannot access signing/publish secrets | SATISFIED (static) | build-macos: 0 MACOS_* refs, 0 environment: declarations; environment: release exclusively on sign-macos; TAP_DEPLOY_TOKEN: 0 in release.yml |
| SEC-09/SEC-10/SEC-11 E2E | Plan 06 | Live rc tag run proves the restructured pipeline works end-to-end | NEEDS HUMAN | 90-06-HUMAN-UAT.md contains 15-item checklist; all static preconditions satisfied; awaiting live tag cut |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | — | — | — | Plans 01-05 produced well-formed YAML, valid bash, and properly-tagged Go. No placeholder patterns, empty handlers, or stub implementations detected. |

### Human Verification Required

#### 1. E2E RC Verification (Plan 06 — all 5 truths from must_haves)

**Pre-flight:**
- Create `release-90-test` branch on `scottkw/homebrew-agenthub` (see `90-TAP-BRANCH-SETUP.md`)
- Audit `release` environment: `gh api /repos/scottkw/agenthub/environments/release`
- Confirm build.yml is green on current HEAD

**Tag cut and pipeline observation:**
- `git tag -a v3.1.0-rc1 -m "Phase 90 rc verification..."` + `git push origin v3.1.0-rc1`
- Watch `gh run watch --workflow release.yml --exit-status`
- In sign-macos logs: confirm "Verify internal attestation" step shows Sigstore verification BEFORE codesign runs
- `gh release view v3.1.0-rc1 --json isDraft --jq .isDraft` must return `true`
- `gh release view v3.1.0-rc1 --json assets --jq '.assets[].name'` must return 6 files

**External attestation:**
- Download all 6 assets to `/tmp/rc-verify/`
- Run `gh attestation verify <file> --owner scottkw` for each — all must succeed

**distribute.yml rc semantics:**
- `gh workflow run distribute.yml --field tag=v3.1.0-rc1`
- update-homebrew-tap: confirm logs show `ref: release-90-test` and `git push origin HEAD:release-90-test`
- submit-winget: confirm `conclusion: skipped`
- Confirm `git log origin/main..origin/release-90-test --oneline` shows 1 commit; main untouched

**Completion:**
- Create `.planning/phases/90-release-pipeline-hardening/90-UAT-REPORT.md` using template in Plan 06 `<interfaces>` block
- Sign off with human reviewer name, date, PASS/FAIL

**All steps tracked in:** `.planning/phases/90-release-pipeline-hardening/90-06-HUMAN-UAT.md`

### Gaps Summary

No gaps in Plans 01-05. All static deliverables are present, substantive, and wired. The sole remaining item is Plan 06's end-to-end human verification, which is **by design** deferred to human execution (`autonomous: false` in the plan frontmatter). The phase's SC-1 (SHA-pin), SC-2 (tool-pin), and SC-3 (secret scope split) are fully satisfied. SC-4 (E2E verification) requires the human UAT described above.

**Requirements SEC-09, SEC-10, SEC-11 are all marked `Complete` in REQUIREMENTS.md** — the static analysis evidence is sufficient for the requirement, with the E2E run serving as operational confirmation rather than a gate.

---

_Verified: 2026-04-24T14:00:00Z_
_Verifier: Claude (gsd-verifier)_
