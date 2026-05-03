# Phase 90: Release Pipeline Hardening - Context

**Gathered:** 2026-04-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Harden the release pipeline so (1) every third-party GitHub Action is pinned to an immutable commit SHA, (2) every Go build tool is pinned to an exact version, and (3) the job that runs untrusted build code cannot reach macOS signing, notarization, or publish secrets. Covers `.github/workflows/*.yml` and `build.sh`. Runtime code is out of scope — this is CI/CD surface only.

Requirements locked: **SEC-09**, **SEC-10**, **SEC-11**.

</domain>

<decisions>
## Implementation Decisions

### Pipeline split architecture (SEC-11)

- **D-01:** Three-stage split: `build-<platform>` → `sign-macos` → `publish`.
  - `build-macos` / `build-windows` / `build-linux`: run `wails build` and any other untrusted code. **No access** to `MACOS_CERTIFICATE*`, `MACOS_NOTARIZATION_*`, `WINGET_TOKEN`, `TAP_DEPLOY_TOKEN`, `RELEASE_PLEASE_TOKEN`, or the GH release upload token.
  - `sign-macos`: macos-latest runner. Downloads the unsigned `.app` artifact, holds ONLY `MACOS_*` secrets, performs codesign + notarytool + stapler + DMG creation, re-uploads the signed DMG. Does not publish.
  - `publish`: ubuntu-latest. Downloads all final artifacts, holds ONLY the GH release upload token, generates checksums, calls `softprops/action-gh-release`. Does not build or sign.
  - Windows/Linux flow straight from their build job to `publish` (no signing today).
- **D-02:** Fix misuse of `TAP_DEPLOY_TOKEN` in the current `publish` job — it's being used for GH release upload, which should use `${{ secrets.GITHUB_TOKEN }}` instead. The tap token belongs only to `distribute.yml` where it pushes to the tap repo.

### Artifact transfer across the trust boundary

- **D-03:** Use `actions/upload-artifact` / `actions/download-artifact` (SHA-pinned per D-06) between stages.
- **D-04:** **Internal build-provenance attestation** on unsigned artifacts: `build-<platform>` job calls `actions/attest-build-provenance` on the unsigned `.app` / `.exe` / binary. `sign-macos` verifies the attestation before signing. Prevents a compromised downstream action from swapping artifacts between jobs.
- **D-05:** **Release build-provenance attestation** on final published artifacts: `publish` job calls `actions/attest-build-provenance` on every file uploaded to the GH release (`.dmg`, `.exe`, `.tar.gz`, `.deb`, `checksums.txt`). Published attestations verifiable via `gh attestation verify <file> --owner scottkw`.

### SHA pinning (SEC-09)

- **D-06:** All third-party Actions pinned to 40-char commit SHAs with a trailing version comment: `uses: actions/checkout@<sha> # v4.2.2`. Official GitHub supply-chain guidance; Dependabot-compatible format.
- **D-07:** Enable Dependabot for the GitHub Actions ecosystem. Weekly schedule. **Manual merge** — no auto-merge. PRs open against `.github/workflows/*` with SHA + comment bumps.
- **D-08:** Replace `vedantmgoyal9/winget-releaser@main` entirely with Microsoft's first-party `wingetcreate` CLI invoked in a `run:` step. The `wingetcreate` binary is downloaded and SHA-verified (pinned version) within the step. Eliminates the one floating-branch ref and moves WinGet submission to a first-party tool.
- **D-09:** Grep gate: a CI step (or pre-commit hook — planner's call) asserts zero matches for `@main`, `@master`, or unpinned branch refs in `.github/workflows/`. Becomes part of the verification bundle for SC-1.

### Build tool pinning (SEC-10)

- **D-10:** Create `tools.go` with `//go:build tools` build tag and blank imports for wails and nfpm. Versions live in `go.mod` alongside runtime deps. Dependabot covers them via the Go ecosystem.
- **D-11:** Install tools in CI using `go list`-derived versions (zero drift between `tools.go` and install command):
  ```
  go install github.com/wailsapp/wails/v2/cmd/wails@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
  go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)
  ```
- **D-12:** `build.sh` updated to use the same pattern so local and CI builds run identical tool versions. The install-suggestion error message (`build.sh:65`) also updated.
- **D-13:** Any other `go install ...@latest` in workflows or scripts is removed as part of this phase (grep gate for `@latest` becomes part of SC-2 verification).

### Release verification (SC-4)

- **D-14:** Verification is performed by cutting a real **`vX.Y.Z-rc1`** pre-release tag on the phase branch. Triggers the real split pipeline end-to-end. Required outputs: signed + notarized macOS DMG, Windows installer, Linux tar.gz + deb, checksums, release + internal attestations.
- **D-15:** `release.yml` upload step passes `draft: true` when the tag matches `v*-rc*`. Artifacts are uploaded, checksums computed, attestations signed — proving the publish path works — but no non-collaborators see the release. Delete the draft after verification.
- **D-16:** `distribute.yml` Homebrew tap step runs against a **`release-90-test` branch** of `scottkw/homebrew-agenthub` when the tag is an rc. Merged to main only if the real release (not the rc) succeeds. Prevents test artifacts from appearing to tap users.
- **D-17:** `distribute.yml` WinGet submission is guarded with `if: !contains(github.ref, '-rc')` so rc tags don't submit to the WinGet repo (avoids polluting their review queue with test submissions).

### Claude's Discretion

- Exact number and format of version comments (e.g., `# v4.2.2` vs `# v4.2.2 (2026-02-14)`) — planner picks.
- Whether the grep gate is a job in an existing workflow or a dedicated `hardening-check.yml` — planner picks based on what's cleaner.
- Exact file location for `tools.go` (repo root vs `internal/tools/` vs `build/tools/`) — planner picks based on Go conventions.
- Whether `sign-macos` uses a composite action or keeps steps inline — stylistic.
- Whether the rc-draft-cleanup is automated (workflow) or documented manual step — planner's call.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` — SEC-09, SEC-10, SEC-11 definitions
- `.planning/ROADMAP.md` §"Phase 90" — goal and 4 success criteria

### Current pipeline (files being modified)
- `.github/workflows/build.yml` — PR/push build matrix. Has `wails@latest` at line 81; Actions at @v4/@v5
- `.github/workflows/release.yml` — tag-triggered release. `build-macos` job holds MACOS_* secrets AND runs wails build (the SEC-11 violation). `wails@latest` at lines 79/177/243; `nfpm@latest` at line 264. `TAP_DEPLOY_TOKEN` misused as release-upload token at line 316
- `.github/workflows/distribute.yml` — runs on release:published. Uses `nick-fields/retry@v3`, `actions/checkout@v4`, and the floating `vedantmgoyal9/winget-releaser@main`
- `.github/workflows/release-please.yml` — uses `googleapis/release-please-action@v4`; holds `RELEASE_PLEASE_TOKEN`
- `build.sh` — local build script. References `wails@latest` in install hint (line 65). Also needs the go.mod-derived install pattern for dev/CI parity

### External references for research
- GitHub hardening guidance: https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions
- `actions/attest-build-provenance` docs (SLSA L2 attestation): https://github.com/actions/attest-build-provenance
- Dependabot for GitHub Actions: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/keeping-your-actions-up-to-date-with-dependabot
- `wingetcreate` CLI: https://github.com/microsoft/winget-create
- Go `tools.go` convention: https://github.com/golang/go/issues/25922

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable assets
- Existing `build-macos` / `build-windows` / `build-linux` job skeletons in `release.yml` are the right starting shape for the split — they already exist as separate jobs, just with secrets attached. The signing/notarize steps (lines 93–136) extract cleanly into a new `sign-macos` job.
- Existing `publish` job (lines 288–316) already uses `actions/download-artifact` + checksum generation — ~80% of the final publish structure.
- `distribute.yml` is already cleanly separated (triggered on `release:published`, different workflow file) — the trust-boundary work there is primarily "replace the @main action with wingetcreate" + "guard rc tags".

### Established patterns
- Workflows already use `actions/upload-artifact@v4` and `actions/download-artifact@v4` between jobs — the artifact-handoff pattern needed for the split is already in use.
- Secrets are accessed via `env:` block at job level, not inline in steps — easy to audit who has what.
- `continue-on-error: true` is used deliberately (WinGet submission); don't remove it unless replacing the step.

### Integration points
- `build.sh` tests live at `tests/build-script.test.sh` — any script changes need corresponding test updates (mentioned in build.yml:62).
- `go.mod` currently has no `tools` section. Adding `tools.go` is net-new structure but idiomatic.
- Dependabot config file (`.github/dependabot.yml`) does not currently exist — Phase 90 creates it.

</code_context>

<specifics>
## Specific Ideas

- Trust posture bias: user picked the stricter option on BOTH attestation questions (internal + release) rather than deferring — signal that Phase 90 should end at SLSA L2 end-to-end, not just minimum SEC-11 compliance.
- User prefers first-party tools over third-party when the trust cost is meaningful — hence wingetcreate over the third-party action.
- Verification should be real, not simulated — draft release + test tap branch, not mock secrets.

</specifics>

<deferred>
## Deferred Ideas

- **Phase 9X — Windows code signing (EV cert + signtool)**: placeholder for when a Windows signing cert is acquired. Current pipeline leaves Windows unsigned; Phase 90 builds the split architecture that makes future Windows signing a clean add (new `sign-windows` job slots in next to `sign-macos`).
- **Phase 9X — Linux .deb GPG signing + apt repo**: same structure as Windows placeholder.
- **Phase 9X — Reproducible builds**: attestation proves provenance, but not that the binary can be rebuilt byte-for-byte. Separate, larger piece of work.
- **Phase 9X — Runner hardening**: disable outbound network egress to unknown hosts during build steps, use `step-security/harden-runner`, etc. Bigger scope than SEC-09/10/11.
- **Phase 9X — Secret rotation playbook**: document + automate periodic rotation of `MACOS_*`, `WINGET_TOKEN`, `TAP_DEPLOY_TOKEN`, `RELEASE_PLEASE_TOKEN`. Process work, not code work.

</deferred>

---

*Phase: 90-release-pipeline-hardening*
*Context gathered: 2026-04-23*
