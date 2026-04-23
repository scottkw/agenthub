# Phase 90: Release Pipeline Hardening - Research

**Researched:** 2026-04-23
**Domain:** CI/CD supply-chain hardening (GitHub Actions, Go build-tool pinning, SLSA provenance, trust-boundary job split)
**Confidence:** HIGH — all critical claims verified against live APIs (GitHub, Microsoft releases) and primary-source docs at research time

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Pipeline split architecture (SEC-11)**
- **D-01:** Three-stage split: `build-<platform>` → `sign-macos` → `publish`.
  - `build-macos` / `build-windows` / `build-linux`: run `wails build` and any other untrusted code. **No access** to `MACOS_CERTIFICATE*`, `MACOS_NOTARIZATION_*`, `WINGET_TOKEN`, `TAP_DEPLOY_TOKEN`, `RELEASE_PLEASE_TOKEN`, or the GH release upload token.
  - `sign-macos`: macos-latest runner. Downloads the unsigned `.app` artifact, holds ONLY `MACOS_*` secrets, performs codesign + notarytool + stapler + DMG creation, re-uploads the signed DMG. Does not publish.
  - `publish`: ubuntu-latest. Downloads all final artifacts, holds ONLY the GH release upload token, generates checksums, calls `softprops/action-gh-release`. Does not build or sign.
  - Windows/Linux flow straight from their build job to `publish` (no signing today).
- **D-02:** Fix misuse of `TAP_DEPLOY_TOKEN` in the current `publish` job — it's being used for GH release upload, which should use `${{ secrets.GITHUB_TOKEN }}` instead. The tap token belongs only to `distribute.yml` where it pushes to the tap repo.

**Artifact transfer across the trust boundary**
- **D-03:** Use `actions/upload-artifact` / `actions/download-artifact` (SHA-pinned per D-06) between stages.
- **D-04:** **Internal build-provenance attestation** on unsigned artifacts: `build-<platform>` job calls `actions/attest-build-provenance` on the unsigned `.app` / `.exe` / binary. `sign-macos` verifies the attestation before signing. Prevents a compromised downstream action from swapping artifacts between jobs.
- **D-05:** **Release build-provenance attestation** on final published artifacts: `publish` job calls `actions/attest-build-provenance` on every file uploaded to the GH release (`.dmg`, `.exe`, `.tar.gz`, `.deb`, `checksums.txt`). Published attestations verifiable via `gh attestation verify <file> --owner scottkw`.

**SHA pinning (SEC-09)**
- **D-06:** All third-party Actions pinned to 40-char commit SHAs with a trailing version comment: `uses: actions/checkout@<sha> # v4.2.2`. Official GitHub supply-chain guidance; Dependabot-compatible format.
- **D-07:** Enable Dependabot for the GitHub Actions ecosystem. Weekly schedule. **Manual merge** — no auto-merge.
- **D-08:** Replace `vedantmgoyal9/winget-releaser@main` entirely with Microsoft's first-party `wingetcreate` CLI invoked in a `run:` step. The `wingetcreate` binary is downloaded and SHA-verified (pinned version) within the step.
- **D-09:** Grep gate: a CI step asserts zero matches for `@main`, `@master`, or unpinned branch refs in `.github/workflows/`. Becomes part of the verification bundle for SC-1.

**Build tool pinning (SEC-10)**
- **D-10:** Create `tools.go` with `//go:build tools` build tag and blank imports for wails and nfpm. Versions live in `go.mod` alongside runtime deps. Dependabot covers them via the Go ecosystem.
- **D-11:** Install tools in CI using `go list`-derived versions (zero drift):
  - `go install github.com/wailsapp/wails/v2/cmd/wails@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)`
  - `go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(go list -m -f '{{.Version}}' github.com/goreleaser/nfpm/v2)`
- **D-12:** `build.sh` updated to use the same pattern so local and CI builds run identical tool versions. The install-suggestion error message at `build.sh:65` also updated.
- **D-13:** Any other `go install ...@latest` in workflows or scripts removed (grep gate for `@latest` becomes part of SC-2).

**Release verification (SC-4)**
- **D-14:** Verification is performed by cutting a real `vX.Y.Z-rc1` pre-release tag on the phase branch.
- **D-15:** `release.yml` upload step passes `draft: true` when the tag matches `v*-rc*`.
- **D-16:** `distribute.yml` Homebrew tap step runs against a `release-90-test` branch of `scottkw/homebrew-agenthub` when the tag is an rc.
- **D-17:** `distribute.yml` WinGet submission is guarded with `if: !contains(github.ref, '-rc')`.

### Claude's Discretion

- Exact number and format of version comments (e.g., `# v4.2.2` vs `# v4.2.2 (2026-02-14)`) — planner picks.
- Whether the grep gate is a job in an existing workflow or a dedicated `hardening-check.yml` — planner picks based on what's cleaner.
- Exact file location for `tools.go` (repo root vs `internal/tools/` vs `build/tools/`) — planner picks based on Go conventions.
- Whether `sign-macos` uses a composite action or keeps steps inline — stylistic.
- Whether the rc-draft-cleanup is automated (workflow) or documented manual step — planner's call.

### Deferred Ideas (OUT OF SCOPE)

- **Phase 9X — Windows code signing (EV cert + signtool)**: current pipeline leaves Windows unsigned; Phase 90 builds the split architecture so a future `sign-windows` slots in next to `sign-macos`.
- **Phase 9X — Linux .deb GPG signing + apt repo.**
- **Phase 9X — Reproducible builds**: attestation proves provenance, but not that the binary can be rebuilt byte-for-byte.
- **Phase 9X — Runner hardening**: `step-security/harden-runner`, egress controls.
- **Phase 9X — Secret rotation playbook.**
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SEC-09 | All third-party GitHub Actions in `.github/workflows/` pinned to immutable commit SHAs (no `@main`, `@master`, no floating branch refs) | "Current Stable SHAs" table + "Grep Gate Design"; `@main` removal covered by D-08 replacement strategy |
| SEC-10 | Go build tools (wails, nfpm, any `go install` targets) pinned to exact versions (no `@latest`) | "Build-Tool Pinning: tools.go Pattern" + Go 1.24+ `go tool` directive alternative; `go list -m -f '{{.Version}}'` invocation validated |
| SEC-11 | Release pipeline restructured so unsigned build step cannot access signing/notarization/publish secrets; signing/publish runs in separate job consuming only pre-built artifacts | "Pipeline Split Architecture" + "Artifact Integrity Across Trust Boundary" + internal/release attestation recipe |
</phase_requirements>

## Summary

Phase 90 is a **CI/CD supply-chain hardening** exercise with three independent concerns — SHA-pinning (SEC-09), tool-version pinning (SEC-10), and secret-scoping via job split (SEC-11) — plus two user-elected stretch goals (internal + release SLSA L2 provenance attestations). Everything is achievable with **first-party GitHub tooling** (`actions/attest-build-provenance`, Dependabot's `github-actions` ecosystem, the `gh attestation verify` CLI) plus **Microsoft's `wingetcreate` CLI** replacing the one floating-branch third-party action.

Three surprises the planner must internalize before writing plans:

1. **`wingetcreate` is Windows-only.** [VERIFIED: Microsoft's own workflow examples for edit, terminal, copilot-cli, PowerToys, oh-my-posh all use `runs-on: windows-latest` — the tool is a .NET binary targeting Windows.] The current `distribute.yml submit-winget` job runs on `ubuntu-latest`; replacing `vedantmgoyal9/winget-releaser@main` with `wingetcreate` requires moving that job to `windows-latest`.
2. **`actions/upload-artifact@v4+` zips everything and strips permissions and symlinks.** [VERIFIED: official README "Limitations → Permission Loss" + issue 693 + issue 92.] A macOS `.app` bundle uploaded raw will have its `Versions/Current` symlinks flattened and its executables stripped of their `+x` bit — codesign on download will fail. **Build-macos must tar the `.app` bundle before upload; sign-macos must untar before signing.** This is the single biggest architectural constraint the planner must honor.
3. **`actions/attest-build-provenance@v4` is now a wrapper.** [VERIFIED: v4.1.0 action.yml runs `actions/attest@v4.1.0` internally.] It continues to work, but v3.2.0 remains the stable, self-contained version. Either is fine — recommend v4 for longevity; the API is identical (`subject-path`, `subject-digest`, `subject-checksums`).

**Primary recommendation:** Plan six waves — (1) preflight grep-gate + Dependabot bootstrap, (2) `tools.go` + go.mod tool requires + CI install pattern + build.sh update, (3) SHA-pin pass across all four workflow files, (4) release.yml split into build/sign/publish with internal attestation + tar-before-upload for .app, (5) distribute.yml — swap winget-releaser for wingetcreate on windows-latest + rc-guard, (6) rc-verification via real `v3.1.0-rc1` tag. Attestation on final artifacts (D-05) is one step in wave 4's publish job and adds ~30s to the pipeline.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Unsigned artifact production | `build-<platform>` jobs (untrusted code) | — | Runs `wails build` — must have zero access to signing/publish secrets |
| Signing + notarization | `sign-macos` job (trusted secret scope) | — | Holds only `MACOS_*` secrets; downloads pre-attested unsigned artifact |
| Publish to GH release | `publish` job (trusted secret scope) | — | Holds only `GITHUB_TOKEN`; no build or sign capability |
| Cross-job artifact transfer | `actions/upload-artifact` + `actions/download-artifact` | `actions/attest-build-provenance` | Upload carries artifact; attestation carries cryptographic identity |
| Trust-boundary verification | `gh attestation verify --bundle <path>` in sign-macos | — | Verifies internal attestation before codesign runs on the artifact |
| Tool-version source of truth | `go.mod` (via `tools.go` blank imports) | Dependabot `gomod` ecosystem | Single file; bumps flow through standard Go dependency PRs |
| Action-version source of truth | SHA in `uses:` line + `# vX.Y.Z` comment | Dependabot `github-actions` ecosystem | Dependabot updates both SHA and comment atomically on upstream release |
| WinGet submission | `submit-winget` job on `windows-latest` | Microsoft `wingetcreate.exe` | First-party tool; avoids third-party action code execution in the distribute workflow |
| Homebrew tap update | `update-homebrew-tap` job on `ubuntu-latest` | — | Bash + sed + git; no third-party action other than checkout/retry |
| Release verification | `vX.Y.Z-rc1` real tag + draft release + rc tap branch + rc WinGet skip | — | Exercises the full pipeline without polluting public release or WinGet review queue |

## Standard Stack

### Core third-party GitHub Actions (pinned SHAs)

All SHAs verified against `gh api repos/<owner>/<repo>/tags` on 2026-04-23. Each row shows the exact string that should appear in the `uses:` line. [VERIFIED: GitHub API live queries at research time.]

| Action | Pinned SHA + Comment | Notes |
|--------|---------------------|-------|
| `actions/checkout` | `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2` | Current latest; v4.2.2 also widely deployed (`11bd71901bbe5b1630ceea73d27597364c9af683`) — either acceptable |
| `actions/setup-go` | `actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0` | |
| `actions/setup-node` | `actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e # v6.4.0` | |
| `actions/upload-artifact` | `actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1` | v4 currently in all four workflow files — bumping to v7 is a breaking change (zip format evolved); recommend bump during hardening pass |
| `actions/download-artifact` | `actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1` | v4 currently in `release.yml`; v8 pairs with upload v7 |
| `actions/attest-build-provenance` | `actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0` | v4 is wrapper on `actions/attest@v4.1.0`; still stable API |
| `pnpm/action-setup` | `pnpm/action-setup@903f9c1a6ebcba6cf41d87230be49611ac97822e # v6.0.3` | Note: latest *release* is v5.0.0 but v6.0.3 tag exists (pre-release pattern) — check `prerelease: false` on release before pinning v6 |
| `softprops/action-gh-release` | `softprops/action-gh-release@b4309332981a82ec1c5618f44dd2e27cc8bfbfda # v3.0.0` | v2 currently in `release.yml`; v3 is current; Node-only action (no nested third-party deps) |
| `nick-fields/retry` | `nick-fields/retry@ad984534de44a9489a53aefd81eb77f87c70dc60 # v4.0.0` | Currently v3 in `distribute.yml` |
| `googleapis/release-please-action` | `googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7 # v5.0.0` | Currently v4 in `release-please.yml`; v5 is current |

**To be REMOVED (replaced per D-08):**
- `vedantmgoyal9/winget-releaser@main` — replaced by inline `wingetcreate.exe` invocation on `windows-latest`.

### Go build tools (pinned via go.mod / tools.go)

| Tool | Module path | Latest v2.x | Currently installed via |
|------|-------------|-------------|-------------------------|
| Wails CLI | `github.com/wailsapp/wails/v2/cmd/wails` | v2.12.0 (2026-03-26) | `go install ...@latest` at 4 places (SEC-10 violation) |
| nfpm (deb packager) | `github.com/goreleaser/nfpm/v2/cmd/nfpm` | v2.46.3 | `go install ...@latest` at `release.yml:264` |

**Current project state:** `go.mod` already pins `github.com/wailsapp/wails/v2 v2.10.2` as a runtime dep. `tools.go` does not exist; `nfpm` is not in `go.mod` at all. [VERIFIED: read of `go.mod` shows no tools block and no nfpm entry; no `tools.go` anywhere in repo.]

**Installation verification:** Training data showed wails v2.10.2 currently pinned in go.mod matches the runtime dep. For `tools.go` to pin a newer wails version specifically for the CLI, the planner can choose either: (a) bump the runtime dep to latest v2.12.0 (clean — one source of truth), or (b) keep runtime at v2.10.2 and pin CLI separately in `tools.go` (complex — may diverge). **Recommend (a).** [CITED: go.mod best practice; multiple wails issues warn about CLI/library version skew.]

### Other tooling

| Tool | Role | Source | Notes |
|------|------|--------|-------|
| `wingetcreate.exe` v1.12.8.0 | WinGet manifest submission | https://github.com/microsoft/winget-create/releases | Self-contained .NET binary; runs on `windows-latest`; SHA-256 `8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D` [VERIFIED: `curl.../wingetcreate.exe.txt \| iconv -f UTF-16LE -t UTF-8` on 2026-04-23]. Authoritative hash published as UTF-16-encoded `.txt` sidecar next to the binary. |
| `gh` CLI (GitHub Actions runners) | Attestation verification (`gh attestation verify`) | Preinstalled on all GitHub-hosted runners | No action needed to install |

## Architecture Patterns

### System Architecture Diagram

```
                ┌────────────────────────────────────────────────────────┐
                │ trigger: push tag v*                                    │
                └────────────────────────────┬───────────────────────────┘
                                             │
                                             ▼
                                 ┌──────────────────────┐
                                 │ validate (unchanged) │
                                 │  go test + fe test   │
                                 └──────────┬───────────┘
                                            │
                         ┌──────────────────┼──────────────────┐
                         ▼                  ▼                  ▼
                 ┌────────────────┐ ┌────────────────┐ ┌────────────────┐
                 │  build-macos   │ │ build-windows  │ │  build-linux   │
                 │  (NO SECRETS)  │ │  (NO SECRETS)  │ │  (NO SECRETS)  │
                 │                │ │                │ │                │
                 │ 1. wails build │ │ 1. wails build │ │ 1. wails build │
                 │ 2. tar .app ←── CRITICAL                             │
                 │ 3. attest      │ │ 2. attest      │ │ 2. attest      │
                 │    (internal)  │ │    (internal)  │ │    (internal)  │
                 │ 4. upload:     │ │ 3. upload:     │ │ 3. upload:     │
                 │    .app.tar.gz │ │    .exe files  │ │    tar.gz+deb  │
                 │    + bundle    │ │    + bundle    │ │    + bundle    │
                 └──────┬─────────┘ └──────┬─────────┘ └──────┬─────────┘
                        │                  │                  │
                        ▼                  │                  │
                ┌──────────────────────┐   │                  │
                │    sign-macos        │   │                  │
                │  (MACOS_* SECRETS    │   │                  │
                │   ONLY — NO PUBLISH) │   │                  │
                │                      │   │                  │
                │ 1. download .app.tgz │   │                  │
                │ 2. download bundle   │   │                  │
                │ 3. gh attestation    │   │                  │
                │    verify --bundle ← TRUST BOUNDARY         │
                │ 4. untar .app        │   │                  │
                │ 5. codesign+notarize │   │                  │
                │ 6. create DMG        │   │                  │
                │ 7. upload signed DMG │   │                  │
                └──────┬───────────────┘   │                  │
                       │                   │                  │
                       └──────────────────┼──────────────────┘
                                          │
                                          ▼
                              ┌──────────────────────────┐
                              │        publish           │
                              │ (GITHUB_TOKEN ONLY —     │
                              │  NO BUILD/SIGN)          │
                              │                          │
                              │ 1. download all          │
                              │ 2. checksums.txt         │
                              │ 3. attest (release)  ←── PUBLIC PROVENANCE
                              │ 4. action-gh-release     │
                              │    draft: <v*-rc* check> │
                              └──────┬───────────────────┘
                                     │
                                     ▼
                              [ release:published event ]
                                     │
                                     ▼
                  ┌──────────────────────────────────┐
                  │         distribute.yml           │
                  │                                  │
                  │ update-homebrew-tap:             │
                  │   (TAP_DEPLOY_TOKEN)             │
                  │   branch=release-90-test if rc   │
                  │                                  │
                  │ submit-winget: (windows-latest!) │
                  │   if: !contains(ref,'-rc')       │
                  │   download+verify wingetcreate   │
                  │   WINGET_CREATE_GITHUB_TOKEN     │
                  └──────────────────────────────────┘
```

### Recommended Project Structure

Files added by Phase 90:

```
.github/
├── dependabot.yml                  # NEW: Dependabot for github-actions + gomod ecosystems
└── workflows/
    ├── build.yml                   # MODIFIED: SHA-pin; swap wails@latest for tools.go pattern
    ├── release.yml                 # MODIFIED: split into build-*/sign-macos/publish; attestations
    ├── distribute.yml              # MODIFIED: rc-guard; winget-releaser → wingetcreate; move to windows
    ├── release-please.yml          # MODIFIED: SHA-pin release-please-action
    └── hardening-check.yml         # NEW (Claude's discretion): grep-gate workflow or step
tools.go                            # NEW: //go:build tools + blank imports (repo root — Go convention)
go.mod                              # MODIFIED: nfpm added; wails version becomes source of truth
go.sum                              # MODIFIED: nfpm checksums
build.sh                            # MODIFIED: line 65 wails@latest → tools.go install pattern
```

### Pattern 1: Three-Stage Release Workflow with Strict Secret Scoping

**What:** Split the tag-triggered release into untrusted-build → trusted-sign → trusted-publish jobs. Each job's secret scope is the minimum necessary for its responsibility.

**When to use:** Any release pipeline that combines untrusted build code (downloads dependencies, runs `npm/go install`, compiles third-party code) with signing or publish credentials.

**Example:** [CITED: https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-secrets and the locked D-01/D-02 decisions]

```yaml
# release.yml — locked structure from D-01 / D-02

jobs:
  validate:
    runs-on: ubuntu-latest
    # unchanged — no secrets

  build-macos:
    needs: [validate]
    runs-on: macos-latest
    # NO environment: block — NO secrets
    permissions:
      id-token: write        # for attest-build-provenance
      attestations: write    # for attest-build-provenance
      contents: read
    steps:
      - uses: actions/checkout@<sha> # v6.0.2
      - uses: actions/setup-go@<sha> # v6.4.0
        with: { go-version-file: go.mod }
      - name: Install Wails CLI (pinned via tools.go)
        run: go install github.com/wailsapp/wails/v2/cmd/wails@$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
      - name: Build .app
        run: wails build -platform darwin/universal -tags wailsassets -ldflags "-X main.Version=${{ github.ref_name }}"
      # CRITICAL: tar the .app BEFORE upload to preserve symlinks+permissions
      - name: Tar .app bundle (preserve symlinks and +x bits)
        run: tar -czf build/bin/AgentHub.app.tar.gz -C build/bin AgentHub.app
      - name: Attest unsigned artifact (internal)
        id: attest
        uses: actions/attest-build-provenance@<sha> # v4.1.0
        with:
          subject-path: build/bin/AgentHub.app.tar.gz
      - name: Upload unsigned artifact + bundle
        uses: actions/upload-artifact@<sha> # v7.0.1
        with:
          name: agenthub-darwin-universal-unsigned
          path: |
            build/bin/AgentHub.app.tar.gz
            ${{ steps.attest.outputs.bundle-path }}
          if-no-files-found: error

  sign-macos:
    needs: [build-macos]
    runs-on: macos-latest
    env:
      MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
      MACOS_CERTIFICATE_NAME: ${{ secrets.MACOS_CERTIFICATE_NAME }}
      MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
      MACOS_CI_KEYCHAIN_PWD: ${{ secrets.MACOS_CI_KEYCHAIN_PWD }}
      MACOS_NOTARIZATION_APPLE_ID: ${{ secrets.MACOS_NOTARIZATION_APPLE_ID }}
      MACOS_NOTARIZATION_PWD: ${{ secrets.MACOS_NOTARIZATION_PWD }}
      MACOS_NOTARIZATION_TEAM_ID: ${{ secrets.MACOS_NOTARIZATION_TEAM_ID }}
    # NO id-token / attestations permissions — sign job only verifies; build attested
    steps:
      - name: Download unsigned artifact + attestation bundle
        uses: actions/download-artifact@<sha> # v8.0.1
        with:
          name: agenthub-darwin-universal-unsigned
          path: artifacts/
      - name: Verify internal attestation
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          BUNDLE=$(ls artifacts/attestation.json artifacts/*.sigstore.json 2>/dev/null | head -1)
          gh attestation verify artifacts/AgentHub.app.tar.gz \
            --repo ${{ github.repository }} \
            --bundle "$BUNDLE"
      - name: Untar .app bundle
        run: tar -xzf artifacts/AgentHub.app.tar.gz -C build/bin/
      # ...existing codesign + notarytool + stapler + create-dmg steps...
      - name: Upload signed DMG
        uses: actions/upload-artifact@<sha> # v7.0.1
        with:
          name: macos-dmg
          path: build/bin/agenthub-*.dmg

  publish:
    needs: [build-macos, build-windows, build-linux, sign-macos]
    runs-on: ubuntu-latest
    permissions:
      contents: write         # required for creating release
      id-token: write         # for release attestation
      attestations: write
    steps:
      - uses: actions/download-artifact@<sha> # v8.0.1
        with: { path: artifacts, merge-multiple: true }
      - name: Generate checksums
        run: cd artifacts && sha256sum * > checksums.txt
      - name: Release provenance attestation
        uses: actions/attest-build-provenance@<sha> # v4.1.0
        with:
          subject-checksums: artifacts/checksums.txt
      - name: Upload to GitHub Release
        uses: softprops/action-gh-release@<sha> # v3.0.0
        with:
          files: |
            artifacts/*.dmg
            artifacts/*.exe
            artifacts/*.tar.gz
            artifacts/*.deb
            artifacts/checksums.txt
          fail_on_unmatched_files: true
          draft: ${{ contains(github.ref, '-rc') }}
          token: ${{ secrets.GITHUB_TOKEN }}   # D-02: was TAP_DEPLOY_TOKEN
```

**Key properties:**
- `build-macos` has **zero** secret access — `env:` block is empty. Compromised action code cannot exfiltrate what isn't there.
- `sign-macos` has **only** `MACOS_*` secrets. Cannot upload to GH release or tap.
- `publish` has **only** `GITHUB_TOKEN`. Cannot sign, cannot build.
- Attestation verification in `sign-macos` is the cryptographic check that the tar the sign job downloads is byte-identical to what `build-macos` attested.

### Pattern 2: Internal vs Release Attestation — What's the Difference?

**Internal attestation (D-04):** The build job attests the *unsigned* artifact. The `sign-macos` job verifies that attestation before signing. This uses the same `actions/attest-build-provenance` action, the same Sigstore infrastructure, and the same `gh attestation verify` CLI. The attestation is **also** published to the GitHub attestations API (there is no "private" mode in public repos — Sigstore public-good instance is used). [VERIFIED: attest action.yml `create-storage-record` default is `true`, which requires `push-to-registry: true`; attestation always uploads to GH attestations API regardless.]

**Why publish the internal attestation at all?** Two reasons:
1. You can't cryptographically verify against an attestation the API doesn't know about. `gh attestation verify --bundle` does the Sigstore signature verification locally but still checks the artifact's digest against the bundle's claims.
2. Having the internal attestation publicly verifiable is a feature — it proves the build job produced the unsigned artifact before anyone signed it. Defense in depth.

**Release attestation (D-05):** The publish job attests the *final, signed* artifacts. Downstream consumers (Homebrew tap users, corporate scanners, users running `gh attestation verify agenthub-*.dmg --owner scottkw`) can cryptographically verify provenance.

**Recommended pattern for release attestation:** Use `subject-checksums: artifacts/checksums.txt` to attest all files in the release with one action call. The `actions/attest` docs explicitly support this. [CITED: https://github.com/actions/attest README "Identify Subjects with Checksums File"]

### Pattern 3: Verifying Attestations Downstream

**In-workflow verification (sign-macos):**
```bash
gh attestation verify artifacts/AgentHub.app.tar.gz \
  --repo ${{ github.repository }} \
  --bundle artifacts/attestation.json
```
[VERIFIED: `--bundle` accepts "a single bundle in a JSON file or a JSON lines file with multiple bundles" per `gh attestation verify --help`; fully offline with `--bundle` present.]

**External verification (anyone, after release):**
```bash
gh attestation verify agenthub-v3.1.0-darwin-universal.dmg --owner scottkw
```
[VERIFIED: `gh attestation verify <file> --owner <org>` is the documented CLI; `--repo <owner>/<repo>` is the narrower form.]

**What the `bundle-path` output contains:** An absolute path to a JSON file on the runner filesystem (`/tmp/attestation-<id>.json` typically). Upload it via `actions/upload-artifact` as a sibling to the subject file; the `sign-macos` job downloads both and passes the bundle path to `--bundle`.

### Pattern 4: `wingetcreate.exe` Inline Invocation on windows-latest

**The Context mismatch:** CONTEXT.md does not specify the runner OS for the wingetcreate replacement. Microsoft's own first-party examples (microsoft/edit, microsoft/terminal, github/copilot-cli, microsoft/PowerToys, JanDeDobbeleer/oh-my-posh) **all use `runs-on: windows-latest`** — and their code comments state explicitly `# winget-create is only supported on Windows`. [VERIFIED: read of 3 separate first-party workflow YAMLs.]

**Concrete pattern for `distribute.yml` `submit-winget` job:**

```yaml
submit-winget:
  runs-on: windows-latest           # MUST be windows; wingetcreate is .NET+Windows
  continue-on-error: true           # preserve — WinGet first submission is known-flaky
  if: ${{ !contains(github.ref, '-rc') }}   # D-17: skip on rc tags
  env:
    WINGET_CREATE_GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}  # Microsoft's env-var contract
    WINGETCREATE_VERSION: v1.12.8.0
    WINGETCREATE_SHA256: 8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D
  steps:
    - name: Download and verify wingetcreate.exe
      shell: pwsh
      run: |
        $url = "https://github.com/microsoft/winget-create/releases/download/$env:WINGETCREATE_VERSION/wingetcreate.exe"
        Invoke-WebRequest -Uri $url -OutFile wingetcreate.exe
        $actual = (Get-FileHash wingetcreate.exe -Algorithm SHA256).Hash
        if ($actual -ne $env:WINGETCREATE_SHA256) {
          Write-Error "SHA256 mismatch: expected $env:WINGETCREATE_SHA256 got $actual"
          exit 1
        }
    - name: Submit to WinGet
      shell: pwsh
      run: |
        $version = "${{ github.event.release.tag_name }}".Trim('v')
        $installerUrl = "https://github.com/scottkw/agenthub/releases/download/${{ github.event.release.tag_name }}/agenthub-${{ github.event.release.tag_name }}-windows-amd64-installer.exe"
        .\wingetcreate.exe update scottkw.agenthub `
          --version $version `
          --urls $installerUrl `
          --submit
```

**Installer URL construction:** The current `vedantmgoyal9/winget-releaser@main` autodiscovers installers via `installers-regex`. `wingetcreate update` wants explicit URLs, so the planner will construct them from the release tag. The project uses a predictable naming pattern (`agenthub-${TAG}-windows-amd64-installer.exe`) — hardcoding the pattern is fine.

**.NET runtime availability:** [CITED: Microsoft's winget-create README, "Using the standalone exe" section — the standalone `wingetcreate.exe` is self-contained and ships with its .NET dependencies; no `UseDotNet` task required.] The standalone .exe (as opposed to the .msixbundle) bundles the .NET runtime. `windows-latest` has .NET anyway, but even if it didn't, the standalone binary works.

**Token:** `WINGET_CREATE_GITHUB_TOKEN` is the Microsoft-recommended env var name. The existing `WINGET_TOKEN` secret should continue to be used — just re-export it under the env var name wingetcreate expects. No new secret needed.

### Anti-Patterns to Avoid

- **Uploading `.app` directories raw with `actions/upload-artifact`.** Zip archives strip symlinks and +x permissions; the downloaded `.app` cannot be signed. Always `tar -czf` first. [VERIFIED: upload-artifact README "Limitations → Permission Loss"]
- **Pinning `@vMajor` (e.g., `@v4`).** Moves with every patch release. Not immutable. Fails the grep gate. Always use a 40-char commit SHA.
- **Using the `actions/attest@v4` (lower-level) directly when `actions/attest-build-provenance@v4` will do.** The wrapper is the documented stable API and has better ergonomics for the common "attest this file" case. Save `actions/attest` for SBOM or custom predicates.
- **Putting secrets in the `validate` job.** The whole point of D-01 is secret scoping; don't re-leak by running a secret-bearing smoke test in validate.
- **Using `contains(github.ref, 'rc')` without the hyphen.** `v3.1.0-rc1` and (hypothetically) `v3.1.0-archive` both contain "rc". Always use `contains(github.ref, '-rc')` to anchor to the hyphen. [VERIFIED: reviewed GitHub docs `contains()` behavior.]
- **Hoping `continue-on-error: true` will mask a real WinGet replacement bug.** Keep it during the transition (D-17 rc-skip already avoids most noise), but plan to remove after first successful submission.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SLSA L2 provenance attestation | Custom `cosign sign-blob` + JSON in-toto predicate | `actions/attest-build-provenance@<sha> # v4.1.0` | The action generates the correct in-toto statement, signs with ephemeral Sigstore cert via OIDC token, and persists to GH's attestation API — all in one step |
| Cross-job artifact integrity check | SHA256 manifest committed as job output + manual diff | Internal attestation + `gh attestation verify --bundle` | Cryptographic (not just hash comparison); Sigstore ephemeral cert validates the *source job* in addition to the artifact content |
| WinGet manifest submission | Custom PR-via-`gh` script + YAML mutation | Microsoft `wingetcreate.exe update --submit` | First-party, handles manifest schema changes, forks the winget-pkgs repo, submits PR with all required fields |
| SHA-pin freshness check | Custom cron job + grep + PR opener | Dependabot `github-actions` ecosystem | Native; supports SHA-pin comment updates since 2022-10; opens PRs with both SHA and `# vX.Y.Z` comment bumped atomically [VERIFIED: GitHub changelog 2022-10-31] |
| Tool-version pinning | Shell script that exports `WAILS_VERSION=v2.12.0` | `tools.go` + `go install X@$(go list -m -f '{{.Version}}' X)` | Single source of truth in `go.mod`; Dependabot `gomod` handles bumps uniformly with runtime deps |
| Draft release cleanup | Cron workflow that polls for old drafts | Manual delete (or simple workflow_dispatch utility) | Volume is low (only rc tags create drafts); automating adds more surface than it saves |
| macOS codesign / notarize | Custom `security import` + `codesign` + `notarytool` wrappers | **Keep the existing steps** — they work | Only the *secret boundary* changes, not the signing logic. Existing `release.yml` lines 93–136 extract verbatim into `sign-macos` |

**Key insight:** Phase 90 is a **surface-area reduction** exercise, not a feature-addition exercise. Every item above has a first-party or existing solution. The only new code is the internal-attestation verification (~5 lines of `gh attestation verify`) and the wingetcreate download+SHA-verify step (~10 lines of PowerShell). Everything else is rearrangement and SHA substitution.

## Runtime State Inventory

> Phase 90 is a workflow-modification / job-split phase. Not a rename/refactor phase, but Step 2.5 still applies because some of the work is CI-only renames (job-name references, secret-name references in Dependabot-managed PRs).

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | None — no DB, no keystores hold "release pipeline" state | — |
| Live service config | **GitHub repo settings:** `secrets.MACOS_*`, `TAP_DEPLOY_TOKEN`, `WINGET_TOKEN`, `RELEASE_PLEASE_TOKEN` are stored in repo/environment settings (UI-only). These **do not need renaming** — Phase 90 only re-scopes which job reads which. The `release` environment (referenced by `release.yml:46 build-macos environment: release`) — planner should decide whether to keep it, split it per job, or eliminate it. [VERIFIED: read of release.yml line 46]. | Re-scope via `env:` block membership; audit GitHub repo environment protection rules |
| OS-registered state | None | — |
| Secrets / env vars | `TAP_DEPLOY_TOKEN` is misused for GH release upload (D-02); must swap to `secrets.GITHUB_TOKEN` in publish job. `WINGET_TOKEN` → rename reference via env to `WINGET_CREATE_GITHUB_TOKEN` inside the submit-winget job (secret name unchanged; env-var alias only). | Code edit in `release.yml` (swap token); code edit in `distribute.yml` (env alias) |
| Build artifacts / installed packages | `go.sum` will change when nfpm is added to `go.mod`. `GOPATH/bin/wails` on local dev machines will be whatever version `go install ...@latest` last installed — phase 90's build.sh change will reinstall the pinned version on next run. No stale egg-info-style artifacts. | Update `go.sum`; document `build.sh` behavior in commit message |

**Canonical answer:** After the SHA-pin pass and job split lands, the only runtime state that could drift is the local dev machine's `GOPATH/bin/wails`. The D-12 `build.sh` change (install pattern parity) makes this self-healing on next invocation.

## Common Pitfalls

### Pitfall 1: Treating `.app` as a file instead of a directory tree with symlinks

**What goes wrong:** Upload-artifact zips the `.app` directory. Symlinks are replaced with regular files (bloating the archive). Executables lose `+x`. Codesign on download fails with "resource fork, Finder information, or similar detritus not allowed" or "bundle format is ambiguous."

**Why it happens:** `upload-artifact@v4+` uses a zip-based wire format that doesn't carry POSIX symlinks or permissions. [VERIFIED: README "Permission Loss" section, issues 92, 693.]

**How to avoid:** `tar -czf <name>.app.tar.gz -C build/bin/ AgentHub.app` before upload. Tar preserves symlinks (relative ones inside the bundle) and permissions. Untar in the sign job. The existing `release.yml:119` already uses `ditto -c -k --keepParent` to zip for notarytool — that's a *different* zip (PKZip-compatible with macOS metadata hooks) and only used internally during notarization; for cross-job transfer, plain `tar -czf` is simpler and sufficient.

**Warning signs:** First failure mode is `codesign --verify` complaining about the signature. Second is `spctl --assess` rejecting the bundle. Both appear after successful "build" step and only during sign.

### Pitfall 2: Pinning `@vMajor` and thinking it's "pinned"

**What goes wrong:** `uses: actions/checkout@v4` looks pinned but `v4` is a moving reference. An attacker who compromises the `v4` tag on actions/checkout ships malicious code to every workflow using it the next time it runs.

**Why it happens:** Semantic-versioning habits from package managers don't map to Git tags, which are mutable references. The tj-actions/changed-files March-2025 compromise exploited exactly this. [VERIFIED: web search on tj-actions changed-files compromise, 350+ tags retroactively moved.]

**How to avoid:** Always use a 40-char commit SHA. The `# vX.Y.Z` comment is for human/Dependabot readability only; it carries no authority. The grep gate (D-09) must reject any `uses: <owner>/<repo>@<anything-not-40-hex>` in the workflow files.

**Warning signs:** `git diff` shows a workflow file's `uses:` line changed but no obvious attribution. Without SHA pinning, this could be because GitHub updated `v4` — and you have no way to know what changed.

### Pitfall 3: Transitive Action Dependencies Aren't Locked by Your SHA

**What goes wrong:** You pin `softprops/action-gh-release@<sha>`, but if that action *internally* uses `@latest` or `@v1` for a sub-action, your pin doesn't propagate.

**Why it happens:** GitHub Actions has no lockfile for transitive deps. [VERIFIED: GitHub's 2026 security roadmap explicitly calls out this gap; a `dependencies:` section is coming in 2026 but not shipped at research time.]

**How to avoid:**
- For each third-party action you pin, check its `action.yml` `runs:` block. If `using: "node*" | "docker"`, there are no transitive actions — the action is self-contained. [VERIFIED: `softprops/action-gh-release@v3` is `runs: using: "node24"` — safe, no nested actions.]
- If `using: "composite"`, inspect each inner step — if any use `@<non-sha>`, that's a gap.

**Warning signs:** An inner step in a composite action fails or behaves unexpectedly after an upstream maintainer's change. The outer action's SHA didn't change but behavior did.

**For Phase 90 specifically:** `softprops/action-gh-release@v3.0.0` is a Node action (safe). `actions/attest-build-provenance@v4.1.0` is composite but its only inner step uses `actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26 # v4.1.0` — already SHA-pinned. [VERIFIED: read of action.yml.] All other third-party actions in the project are Node-only. No transitive exposure at research time — but the planner should include a comment reminding future maintainers to re-audit on version bump.

### Pitfall 4: Token Scope Creep in Environments

**What goes wrong:** Current `release.yml` declares `environment: release` on `build-macos` (line 46). Environment-scoped secrets are attached to **every** job that declares that environment. If `sign-macos` and `publish` are both added to the `release` environment, all three jobs see all environment secrets — defeating D-01.

**Why it happens:** GitHub environments are a secret-bundling construct, not an exclusion-rule construct. There's no "environment excludes secret X" syntax.

**How to avoid:** Option A (recommended) — keep `release` environment on `sign-macos` *only*, put `MACOS_*` secrets there; put `GITHUB_TOKEN` in the default environment for `publish`; leave `build-*` jobs with no environment declaration. Option B — create three environments (`build`, `sign`, `publish`) with non-overlapping secret sets. A is simpler; B is more auditable. [CITED: https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment]

**Warning signs:** `env.MACOS_CERTIFICATE` is accessible in a job that has no business signing. Audit with `echo "::add-mask::$MACOS_CERTIFICATE"` tricks removed; try to reference the secret in an untrusted job and verify it's empty.

### Pitfall 5: Go tool Install Pattern Silently Installs Latest on go.mod Miss

**What goes wrong:** `go install X@$(go list -m -f '{{.Version}}' X)` — if the module X is *not* in `go.mod`, `go list -m` prints a warning to stderr but still produces a string (empty or go.mod directive-less). Shell expansion gives `go install X@` which defaults to `@latest`.

**Why it happens:** Go's `go list -m` on a non-required module behaves differently across Go versions; versions ≥1.22 return clean error, older return permissive. Under `set -e` this may still pass. [CITED: https://pkg.go.dev/cmd/go#hdr-Print_module_list "Print module list" behavior on non-existent modules.]

**How to avoid:** Wrap with an explicit check:
```bash
WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
if [[ -z "$WAILS_VER" ]]; then echo "wails not pinned in go.mod" >&2; exit 1; fi
go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"
```

**Warning signs:** `go install` succeeds but the installed version is newer than go.mod says. Surface this with a post-install check: `wails version | grep "$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)"`.

### Pitfall 6: Dependabot Opens PR, Merges Silently Via Auto-Merge

**What goes wrong:** Repo-level auto-merge is enabled (or `merge_queue.auto_merge` is set). Dependabot PR lands without human review. Attacker-controlled release of a pinned action slips through.

**Why it happens:** Auto-merge is a repo setting (Settings → Pull requests → Allow auto-merge), independent of `dependabot.yml`. [VERIFIED: there's no field in `dependabot.yml` that enables auto-merge.] The risk is an existing repo setting or a separate `pull_request` workflow that calls `gh pr merge --auto`.

**How to avoid:**
- Confirm Settings → Pull requests → "Allow auto-merge" is **off**, OR keep it on but ensure no workflow calls `gh pr merge --auto` on Dependabot PRs.
- The D-07 "manual merge only" requirement is enforced by repo configuration, not by `dependabot.yml` alone — document this in CLAUDE.md or PROJECT.md.

**Warning signs:** Dependabot PR closes as "merged" within minutes of opening without a human commit. Check the merge author — `dependabot[bot]` merging itself is the smoking gun.

### Pitfall 7: Forgetting That `github.ref` on `release:published` Is `refs/tags/...`

**What goes wrong:** Code assumes `github.ref` on a `release:published` trigger is a branch ref or the release name. It's actually `refs/tags/<tag>` (e.g., `refs/tags/v3.1.0-rc1`). [VERIFIED: search of GitHub Actions issues; confirmed behavior.]

**Why it happens:** Workflows triggered by `push: tags:` get `refs/tags/<tag>`, which matches intuition. `release:published` is triggered by the release event (which has an underlying tag), so `github.ref` is the tag ref. But developers sometimes write conditions assuming it's the release *name*.

**How to avoid:** Use `contains(github.ref, '-rc')` (works for both tag-trigger and release-trigger because both carry `refs/tags/v*-rc*`). Avoid `github.event.release.name` for rc detection — the release name is user-editable and not always the same as the tag.

**Warning signs:** A condition like `if: github.event.release.tag_name` suddenly not matching because the release was renamed in the UI.

## Code Examples

### Example 1: `tools.go` at repo root (locked D-10)

```go
//go:build tools
// +build tools

// Package tools documents build-tool dependencies. It is excluded from normal
// builds by the `tools` build tag. The blank imports cause `go mod tidy` to
// keep these modules in go.mod alongside runtime dependencies, making Dependabot
// aware of them via the gomod ecosystem.
//
// CI and build.sh install these tools using:
//   go install <path>@$(go list -m -f '{{.Version}}' <module>)
//
// See .planning/phases/90-release-pipeline-hardening/ for rationale.
package tools

import (
	_ "github.com/goreleaser/nfpm/v2/cmd/nfpm"
	_ "github.com/wailsapp/wails/v2/cmd/wails"
)
```

**File location (Claude's discretion per CONTEXT):** Repo root. [CITED: https://github.com/golang/go/issues/25922 discussion; also well-known Go projects place it at root — grpc-go, kubernetes, cockroachdb all use root-level tools.go or equivalent.] `internal/tools/` works too but doesn't auto-pick-up by `go mod tidy` the same way — root is the standard form in the issue's accepted solution.

**Required go.mod update:** `go mod tidy` will add nfpm as a direct require after tools.go lands:
```
require (
    github.com/goreleaser/nfpm/v2 v2.46.3  // new
    github.com/wailsapp/wails/v2 v2.10.2
    // ... rest unchanged
)
```

**Go 1.24+ alternative (NOT USED per D-10 lock):** Since the project is on Go 1.26, the modern `go tool` directive would work:
```
tool github.com/wailsapp/wails/v2/cmd/wails
tool github.com/goreleaser/nfpm/v2/cmd/nfpm
```
and invocation becomes `go tool wails build ...` instead of `go install ... + wails build ...`. [VERIFIED: Go 1.24 release notes explicitly state the `tool` directive replaces the `tools.go` workaround.] **Flagged here because D-10 locks `tools.go` — if the user wants to revisit, they can; for now, honor D-10.** The `tools.go` pattern remains fully supported in Go 1.26+.

### Example 2: CI install with drift-free version derivation

```bash
# workflow step (locked D-11)
- name: Install Wails CLI (version from go.mod)
  run: |
    WAILS_VER=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2)
    [ -n "$WAILS_VER" ] || { echo "wails not pinned in go.mod"; exit 1; }
    go install github.com/wailsapp/wails/v2/cmd/wails@"$WAILS_VER"
```

### Example 3: `build.sh` update for D-12

Replace current lines 61-67:
```bash
# OLD (current build.sh:62-67)
WAILS="$(go env GOPATH)/bin/wails"
if [[ ! -x "$WAILS" ]]; then
  echo "ERROR: wails not found at $WAILS"
  echo "Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"  # <-- SEC-10 violation
  exit 1
fi
```

With:
```bash
# NEW
WAILS="$(go env GOPATH)/bin/wails"
WAILS_PINNED_VER="$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2 2>/dev/null)"
if [[ -z "$WAILS_PINNED_VER" ]]; then
  echo "ERROR: wails not pinned in go.mod — Phase 90 tools.go setup missing"
  exit 1
fi
if [[ ! -x "$WAILS" ]]; then
  echo "ERROR: wails not found at $WAILS"
  echo "Install: go install github.com/wailsapp/wails/v2/cmd/wails@$WAILS_PINNED_VER"
  exit 1
fi
# Optional: verify installed version matches pinned version
if ! "$WAILS" version 2>/dev/null | grep -qF "$WAILS_PINNED_VER"; then
  echo "WARN: installed wails does not match pinned version ($WAILS_PINNED_VER)"
  echo "Reinstall: go install github.com/wailsapp/wails/v2/cmd/wails@$WAILS_PINNED_VER"
fi
```

The `tests/build-script.test.sh` will need a new assertion — `build.sh` no longer contains the literal `@latest`. Add to Section 9/11:
```bash
# NEW test
output=$(grep -c '@latest' "$BUILD_SH" || true)
if [[ "$output" -eq 0 ]]; then
  pass "build.sh contains no @latest references (SEC-10)"
else
  fail "build.sh contains @latest — SEC-10 violation" "matches: $output"
fi
```

### Example 4: `.github/dependabot.yml` (locked D-07)

```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
      timezone: "America/Los_Angeles"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "ci(actions)"
    labels:
      - "dependencies"
      - "github-actions"
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
    open-pull-requests-limit: 5
    commit-message:
      prefix: "deps"
    labels:
      - "dependencies"
      - "go"
```

**No auto-merge field exists in dependabot.yml** [VERIFIED: current GitHub docs schema has no such field]. Manual merge is the default. Auto-merge requires either repo-level setting + branch protection, or a separate workflow — the planner should NOT add such a workflow.

**Grouping (Claude's discretion):** GitHub-actions grouped-update support was added in 2023. [CITED: GitHub docs "groups" option.] Trade-off: grouped = fewer PRs, harder SHA-by-SHA audit; ungrouped = more PR noise, easier review. **Recommend ungrouped** — the audit discipline is the whole point of SEC-09, and 4 workflows × ~6 actions = ~24 actions — bump frequency is measured in PRs-per-month, not per-day.

### Example 5: Grep gate (locked D-09)

```bash
# .github/workflows/hardening-check.yml or inline step
#!/usr/bin/env bash
set -euo pipefail

echo "==> Checking for unpinned @main / @master / @latest refs..."

# Any uses: line whose ref isn't a 40-char hex SHA
BAD=$(grep -rEn 'uses:\s*[^#]*@(main|master|v[0-9]+|[a-z]+)$' .github/workflows/ || true)
if [[ -n "$BAD" ]]; then
  echo "FAIL: unpinned action refs found:"
  echo "$BAD"
  exit 1
fi

# Any @latest in workflows or build.sh
LATEST=$(grep -rEn '@latest' .github/workflows/ build.sh tests/ || true)
if [[ -n "$LATEST" ]]; then
  echo "FAIL: @latest references found:"
  echo "$LATEST"
  exit 1
fi

# uses: lines that are not 40-char SHA (negative regex — must match SHA pattern)
NON_SHA=$(grep -rE 'uses:\s*[^ ]+@' .github/workflows/ \
  | grep -Ev 'uses:\s*[^ ]+@[a-f0-9]{40}(\s|$)' || true)
if [[ -n "$NON_SHA" ]]; then
  echo "FAIL: non-SHA action refs (likely @v4 tags):"
  echo "$NON_SHA"
  exit 1
fi

echo "PASS: all action refs are SHA-pinned"
```

**Location recommendation (Claude's discretion per CONTEXT):** Make it a step in `build.yml` under a new job called `workflow-hardening`. Runs on every PR/push to `main`. Cheaper than a separate workflow (one less runner minute, same effect). If you prefer separation, `.github/workflows/hardening-check.yml` with `on: [push, pull_request]` works too. **Recommend inline step in build.yml.**

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `@v4` tag pins | 40-char SHA + `# v4.x.x` comment | 2022-10 Dependabot comment-sync, 2025-03 tj-actions compromise | Required for any serious supply-chain security posture; cheap |
| `tools.go` build-tag pattern | Go 1.24+ `tool` directive in go.mod | 2025 Go 1.24 release | **Both work;** D-10 locks tools.go for now. Modern preference is `go tool`, but tools.go is more widely understood and supported by more tooling |
| Custom `cosign sign-blob` | `actions/attest-build-provenance` | 2024 GA of GH artifact attestations | SLSA L2 provenance with ~5 lines of YAML; use `actions/attest@v4` for SBOMs/custom predicates |
| `actions/upload-artifact@v3` | `@v4+` (zip wire-format) | 2024 v4 release | Different artifact API; v3 deprecated; v4/v5/v6/v7/v8 releases roughly quarterly; **WATCH: v7 introduces further breaking changes** |
| Third-party `winget-releaser` actions | Microsoft first-party `wingetcreate` CLI | Consistent since 2022 | Microsoft's own repos use `wingetcreate`; third-party actions are unnecessary middlemen |
| Unified release job (build+sign+publish) | Three-stage split with internal attestation | GitHub SLSA L2 attestation GA + SEC-11 | Prevents build-step compromise from reaching signing secrets |

**Deprecated / outdated:**
- `@actions/toolkit` v1: no effect on user workflows; internal to actions themselves.
- `upload-artifact@v3`: removed 2024; any reference needs bumping.
- `set-output` syntax (`::set-output name=X::value`): replaced by `$GITHUB_OUTPUT` env file; confirm none in `release.yml:46, 301` — current project uses `$GITHUB_OUTPUT` correctly. [VERIFIED: read of distribute.yml:23]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `upload-artifact@v7` is a safe bump from `v4` for `.tar.gz` / `.exe` / `.dmg` files (non-bundle files) | Standard Stack | Low — zip wire format changed between v3/v4; v4→v7 reported backward-compatible for simple file uploads [not tested in this project]. Mitigation: planner can choose to stay on v4 (also current and works) — the bump is optional |
| A2 | `actions/attest-build-provenance` works on macos-latest and windows-latest runners in addition to ubuntu-latest | Pattern 1, 2 | Low — no official "unsupported platform" statement found, but also no explicit per-platform confirmation in docs. Mitigation: rc-tag verification (D-14) will exercise all three platforms; any platform gap surfaces immediately |
| A3 | Microsoft's `wingetcreate.exe` standalone binary does not require additional .NET runtime install on `windows-latest` | Pattern 4 | Low — README "Using the standalone exe" text suggests the standalone is self-contained; the .msixbundle variant is the one requiring VCLibs. Mitigation: if it fails, add `UseDotNet@2` task at step start (reasonable fallback) |
| A4 | The grep gate can reliably detect all "bad" refs with the regex set in Example 5 | Example 5 | Low — the regex explicitly enumerates `@main|@master|@vMajor|@latest`. Edge case: someone could write `uses: myorg/my-action@some-custom-branch` which the regex catches via the "not 40-char SHA" check. Mitigation: the planner should include a test case in the hardening-check workflow that intentionally adds a bad ref and asserts the check fails |
| A5 | Moving `submit-winget` from ubuntu to windows runner has no billing/availability impact worth flagging | Pattern 4 | Low — windows-latest minute-multiplier is 2x ubuntu on private repos; **public repo (this project) is 1x for both**. No meaningful cost change |
| A6 | The `release` environment in release.yml:46 is used only for `MACOS_*` secrets, not for other environment-scoped config | Pitfall 4 | Medium — if the environment also has protection rules (required reviewers, wait timer), moving which jobs use it changes release mechanics. Mitigation: planner should query the repo environment config (`gh api /repos/scottkw/agenthub/environments/release`) as part of task planning |
| A7 | `gh attestation verify --bundle <local-file>` on macos-latest has the required `gh` CLI version (≥2.49 introduced `--bundle`) | Pattern 3 | Low — GitHub-hosted runners ship recent `gh`. Verified `gh` is available (not verified minimum version on macos-latest). Mitigation: add `gh --version` step and bail if too old |

## Open Questions

1. **Will `draft: true` on the publish step work as expected when the release was *already created* by `release-please`?**
   - What we know: `release.yml:307 softprops/action-gh-release@v2 files: ...` uploads assets to an existing release if the tag matches. `release-please` creates the release on merge-to-main. The rc flow pushes a tag directly, so release-please doesn't fire — the tag-triggered release.yml creates the release inline (via softprops/action-gh-release's auto-create behavior).
   - What's unclear: In the rc flow, does softprops/action-gh-release create the release as draft on first upload (honoring `draft: ${{ contains(github.ref, '-rc') }}`)? Or does it upload to a non-draft release?
   - Recommendation: Test this as part of the D-14 verification run. Expectation per softprops docs: `draft` input maps directly to the API `draft` field on creation. Should work, but the rc-first-upload path is the specific one to exercise.

2. **Does the `sign-macos` job need the `attestations: write` permission to *verify* an attestation, or only `contents: read`?**
   - What we know: Generating attestation requires `id-token: write + attestations: write + contents: read`. Verifying is a read operation on the attestations API.
   - What's unclear: Whether `gh attestation verify --bundle` (with a local bundle file) hits the API at all, or is fully offline.
   - Recommendation: Start with `permissions: contents: read` only on sign-macos. If `gh attestation verify --bundle` fails with a permissions error, escalate to `attestations: read`. Likely fully offline — the `--bundle` flag is documented as offline-capable. [VERIFIED via gh-cli docs.]

3. **Should the internal attestation use `subject-path: <.app.tar.gz>` or `subject-digest: sha256:<hash>` + `subject-name: AgentHub.app.tar.gz`?**
   - What we know: Both work; `subject-path` is simpler and auto-computes the digest.
   - What's unclear: In the cross-job verify, which form does `gh attestation verify --bundle` handle better?
   - Recommendation: `subject-path` during attest, `gh attestation verify <path-to-file> --bundle <bundle>` during verify. The file itself gets re-hashed during verify; the tool handles this transparently.

4. **Is there any reason to keep the `release` environment declaration on any job after the split?**
   - What we know: GitHub environments have protection rules (required reviewers, wait timer, secret binding). The current `release.yml:46` uses it for secret binding of `MACOS_*` — that function is preserved when only `sign-macos` declares `environment: release`.
   - What's unclear: Is there a wait-timer or required-reviewer rule currently attached that affects deploy mechanics?
   - Recommendation: `gh api /repos/scottkw/agenthub/environments/release` to dump the rules before re-scoping. Preserve equivalent protection on the new sign-macos job; otherwise the split accidentally bypasses a required-reviewer gate.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `go` | All workflow modifications (go.mod, tools.go) | ✓ (local dev), GitHub runners auto-install | go1.26.2 locally; go.mod pins 1.26.1 | — |
| `gh` CLI | Attestation verify; probing current SHA state | ✓ | 2.90.0 locally; pre-installed on runners | — |
| `docker` | Linux cross-compile in `build.sh` (unchanged) | ✓ locally | 29.4.0 | — |
| `wingetcreate.exe` v1.12.8.0 | submit-winget job on windows-latest | N/A (downloaded at job runtime) | v1.12.8.0 (SHA `8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D`) | Pin to older version if v1.12.8.0 breaks |
| Microsoft winget-pkgs repo write (fork flow) | submit-winget end-to-end | Requires `WINGET_TOKEN` with `public_repo` scope | — | If token scope insufficient, submission PR fails — matches current behavior (rc-guard avoids this) |
| Homebrew tap repo (`scottkw/homebrew-agenthub`) with `release-90-test` branch | distribute.yml tap step on rc | Exists: scottkw's own repo; branch must be created before rc verification | — | Manual branch creation if workflow auto-creation fails |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:**
- `release-90-test` branch in `scottkw/homebrew-agenthub` — must be created before cutting `v3.1.0-rc1`. Either by Claude via the planner's wave-6 task, or by user. Simple: `gh repo clone scottkw/homebrew-agenthub && cd homebrew-agenthub && git checkout -b release-90-test && git push -u origin release-90-test`.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Bash tests (`tests/build-script.test.sh`) + Go tests (`go test`) + shell grep-gates (new Phase 90 infra) |
| Config file | None — tests are scripts invoked by GitHub workflows |
| Quick run command | `bash tests/build-script.test.sh` |
| Full suite command | `bash tests/build-script.test.sh && go test -race -short ./... && bash .github/workflows/hardening-check-local.sh` (last one new) |
| Phase gate | Full suite green + successful `v3.1.0-rc1` end-to-end run before `/gsd-verify-work` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| SEC-09 | Zero `@main`, `@master`, unpinned branch refs in `.github/workflows/` | Static grep gate | `bash scripts/check-action-pins.sh` (or equivalent inline step) | ❌ Wave 0 |
| SEC-09 | Zero non-SHA refs anywhere in workflows | Static grep gate | Same script, negative regex | ❌ Wave 0 |
| SEC-10 | Zero `@latest` in workflows or build scripts | Static grep gate | `! grep -rE '@latest' .github/ build.sh tests/` | ❌ Wave 0 |
| SEC-10 | `build.sh` uses `go list -m` pattern for install hint | Static string check | Extend `tests/build-script.test.sh` | ✅ test harness exists; new assertion needed |
| SEC-10 | `tools.go` exists and compiles with `-tags tools` | Smoke | `go build -tags tools ./...` | ❌ Wave 0 |
| SEC-11 | `build-<platform>` jobs have no MACOS_* / WINGET / TAP / RELEASE_PLEASE secrets | Workflow source-grep | Custom: parse release.yml, assert `env:` on build jobs is empty (or whitelist only build-safe env) | ❌ Wave 0 |
| SEC-11 | `sign-macos` has MACOS_* but no WINGET/TAP/RELEASE_PLEASE secrets | Workflow source-grep | Same parser, different whitelist | ❌ Wave 0 |
| SEC-11 | `publish` has only GITHUB_TOKEN (explicit: NOT TAP_DEPLOY_TOKEN) | Workflow source-grep | Same | ❌ Wave 0 |
| SC-4 / all | Real `v3.1.0-rc1` tag produces signed DMG, draft release, rc tap branch, release attestation, passing `gh attestation verify` | E2E (manual) | Manually push tag; observe workflow runs; verify artifacts | ❌ Wave 6 |

### Sampling Rate
- **Per task commit:** `bash tests/build-script.test.sh` + grep-gates
- **Per wave merge:** Full suite above + `go build -tags tools ./...`
- **Phase gate:** Full suite green + manual rc-tag verification produces all expected outputs + `gh attestation verify agenthub-v3.1.0-rc1-darwin-universal.dmg --owner scottkw` succeeds

### Wave 0 Gaps
- [ ] `scripts/check-action-pins.sh` (or equivalent) — parametric SHA-pin + `@latest` grep-gate script.
- [ ] `scripts/check-secret-scopes.sh` — YAML-aware parser (or a strict grep approximation) that asserts per-job secret scope boundaries for SEC-11.
- [ ] Extension of `tests/build-script.test.sh` — assertions that `@latest` is absent and that the new install pattern string is present.
- [ ] Pre-create `release-90-test` branch in `scottkw/homebrew-agenthub` — prerequisite for D-16 verification.
- [ ] Decision on grep-gate workflow file vs step-in-build.yml (Claude's discretion per CONTEXT — recommend step-in-build.yml).

## Security Domain

### Applicable ASVS Categories

Phase 90 is CI/CD-focused. Traditional ASVS categories (V2 auth, V3 session, V5 input validation) primarily address runtime application code. For a supply-chain hardening phase, the relevant security model is **SLSA (Supply-chain Levels for Software Artifacts)** and **SSDF (NIST Secure Software Development Framework)**, not ASVS. Mapping provided for completeness:

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | — (CI auth handled by GitHub OIDC; action-level tokens are ephemeral) |
| V3 Session Management | no | — |
| V4 Access Control | **yes (workflow secret scoping)** | Per-job secret env — the core of D-01 |
| V5 Input Validation | no (no user-supplied input in this phase) | — |
| V6 Cryptography | **yes (SLSA L2 provenance)** | `actions/attest-build-provenance` (Sigstore ephemeral cert, OIDC-bound) — never hand-roll signing |
| V14 Configuration | **yes (principle of least privilege)** | Scoped tokens, environment-scoped secrets, no org-wide PATs |

**More relevant frameworks:**
- **SLSA Build L2:** what Phase 90 delivers. Requires hosted build platform (GitHub Actions satisfies) + signed provenance (attestation satisfies). [CITED: https://slsa.dev/spec/v1.0/levels]
- **NIST SSDF PS.3 / PW.4:** secure software development practices — pinning dependencies, verifying provenance. SEC-09, SEC-10 map directly.
- **OpenSSF Scorecard:** `Pinned-Dependencies` check (which scores exactly the SHA-pin posture SEC-09 establishes).

### Known Threat Patterns for GitHub Actions CI/CD

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Compromised floating-branch action tag (tj-actions/changed-files 2025-03) | Tampering | SHA-pin every third-party action (D-06) |
| Secret exfiltration during untrusted build step (`wails build` invokes `npm install` which runs arbitrary package postinstall) | Information Disclosure | Job split + per-job secret scoping (D-01) |
| Artifact substitution between build and sign | Spoofing | Internal build-provenance attestation (D-04) + verify-before-sign |
| Downstream consumer cannot verify release authenticity | Spoofing (downstream) | Release build-provenance attestation (D-05) — `gh attestation verify` available to anyone |
| Dependency confusion / malicious new tool dep | Tampering | `tools.go` + Dependabot `gomod` weekly review + manual merge (D-07, D-10) |
| Third-party action maintainer turnover | Supply-chain | Prefer first-party tools (D-08: microsoft/wingetcreate over vedantmgoyal9/winget-releaser) |
| GitHub tag-name collision or typo confusion (`actions/chekcout`) | Spoofing | 40-char SHA makes namesquatting irrelevant — SHA has to match |
| Bypassed secret scope via reusable workflow / composite action nesting | Elevation of Privilege | Audit composite action.yml for nested `uses:` (Pitfall 3); prefer Node-only actions |
| Long-lived PAT exfiltration | Information Disclosure | Default: `secrets.GITHUB_TOKEN` (ephemeral per-run); PATs only when strictly required (`TAP_DEPLOY_TOKEN`, `RELEASE_PLEASE_TOKEN`, `WINGET_TOKEN`) and scoped minimally |

## Project Constraints (from CLAUDE.md)

- **`NEVER kill node.exe`** — Claude Code runs on Node. Irrelevant for Phase 90 CI work but listed for completeness.
- **Make beliefs pay rent; notice confusion; line of retreat.** Research surfaces 7 explicit assumptions (A1–A7) and 4 open questions — discipline is honored.
- **Chesterton's Fence.** Before removing the `release` environment declaration from release.yml:46, understand *why* it's there (secret binding) so the restructure preserves that function. Before removing `continue-on-error: true` from submit-winget — preserve it, given first-submission flake history noted in distribute.yml comment.
- **LSP preference over grep** — n/a for this phase (YAML and shell files, not Go code).
- **UAT via dev-browser skill** — n/a for this phase (no browser surface).

## Sources

### Primary (HIGH confidence)

- **Live GitHub API queries on 2026-04-23** — all action SHAs in the Standard Stack table:
  - `gh api repos/actions/checkout/tags` → v6.0.2 SHA `de0fac2e4500dabe0009e67214ff5f5447ce83dd`
  - `gh api repos/actions/setup-go/tags` → v6.4.0 SHA `4a3601121dd01d1626a1e23e37211e3254c1c06c`
  - `gh api repos/actions/setup-node/tags` → v6.4.0 SHA `48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e`
  - `gh api repos/actions/upload-artifact/tags` → v7.0.1 SHA `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a`
  - `gh api repos/actions/download-artifact/tags` → v8.0.1 SHA `3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c`
  - `gh api repos/actions/attest-build-provenance/tags` → v4.1.0 SHA `a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32`
  - `gh api repos/actions/attest/tags` → v4.1.0 SHA `59d89421af93a897026c735860bf21b6eb4f7b26`
  - `gh api repos/softprops/action-gh-release/tags` → v3.0.0 SHA `b4309332981a82ec1c5618f44dd2e27cc8bfbfda`
  - `gh api repos/nick-fields/retry/tags` → v4.0.0 SHA `ad984534de44a9489a53aefd81eb77f87c70dc60`
  - `gh api repos/googleapis/release-please-action/tags` → v5.0.0 SHA `45996ed1f6d02564a971a2fa1b5860e934307cf7`
  - `gh api repos/pnpm/action-setup/tags` → v6.0.3 SHA `903f9c1a6ebcba6cf41d87230be49611ac97822e`
  - `gh api repos/microsoft/winget-create/releases/latest` → v1.12.8.0 with SHA-256 `8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D` (decoded from UTF-16 `.txt` sidecar)
  - `gh api repos/wailsapp/wails/releases` → v2.12.0 (2026-03-26)
  - `gh api repos/goreleaser/nfpm/releases/latest` → v2.46.3
- **Official docs (read during research):**
  - `actions/attest-build-provenance` README + action.yml (v4.1.0 + v3.2.0)
  - `actions/attest` README v4.1.0 (the underlying primitive)
  - `actions/upload-artifact` README (v4/v7 behavior)
  - `microsoft/winget-create` README + `doc/update.md` + `doc/token.md`
  - `softprops/action-gh-release` action.yml (confirms `runs: node24`, no nested actions)
  - GitHub Docs: https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions
  - GitHub Docs: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/keeping-your-actions-up-to-date-with-dependabot
  - GitHub Docs: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference
  - GitHub CLI manual: https://cli.github.com/manual/gh_attestation_verify
  - Go 1.24 release notes: https://tip.golang.org/doc/go1.24 (for the `go tool` directive alternative)
  - GitHub blog 2022-10-31: Dependabot now updates comments in SHA-pinned GitHub Actions
- **First-party workflow examples for wingetcreate:**
  - https://github.com/microsoft/edit/blob/main/.github/workflows/winget.yml (VERIFIED: `runs-on: windows-latest`, standalone .exe download)
  - https://github.com/microsoft/terminal/blob/main/.github/workflows/winget.yml (VERIFIED: same pattern)
  - https://github.com/github/copilot-cli/blob/v0.0.368-2/.github/workflows/winget.yml (VERIFIED: same pattern, `WINGET_CREATE_GITHUB_TOKEN` env var)

### Secondary (MEDIUM confidence)
- GitHub Issue 25922 (tools.go convention) — https://github.com/golang/go/issues/25922 — community consensus on root-level tools.go
- GitHub Actions Changelog 2025-08-15: SHA-pinning policy support
- GitHub Blog: https://github.blog/changelog/2022-10-31-dependabot-now-updates-comments-in-github-actions-workflows-referencing-action-versions/
- StepSecurity blog: Pinning GitHub Actions — https://www.stepsecurity.io/blog/pinning-github-actions-for-enhanced-security-a-complete-guide
- actions/upload-artifact issues 92, 693 (symlink and permission limitations — matches README)

### Tertiary (LOW confidence)
- None — all critical claims have HIGH-confidence backing.

## Metadata

**Confidence breakdown:**
- Standard stack (SHAs, versions): HIGH — all live-queried today
- Architecture split (D-01/D-02 recipe): HIGH — honors locked CONTEXT + official docs
- Attestation pattern: HIGH — primary docs + action.yml read + v3.2/v4 version history confirmed
- `wingetcreate` pattern: HIGH — three independent first-party Microsoft workflow examples confirm the exact pattern; SHA verified against official release asset
- Pitfalls: HIGH for upload-artifact symlink loss, SHA-pin transitive, rc-tag ref shape; MEDIUM for Pitfall 5 (go install empty-version case — behavior is Go-version-dependent)
- Dependabot config: HIGH for schema; MEDIUM for grouped-vs-ungrouped tradeoff recommendation (judgment call)
- Go tools.go vs Go 1.24 `go tool`: HIGH for both — user decision locked (D-10) honors tools.go, but Go 1.24+ alternative documented for future reference

**Research date:** 2026-04-23
**Valid until:** 2026-05-23 (30-day estimate; SHA values will drift faster — re-query before phase execution if > 2 weeks old)
