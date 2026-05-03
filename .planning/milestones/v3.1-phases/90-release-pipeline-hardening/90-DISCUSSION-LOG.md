# Phase 90: Release Pipeline Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-23
**Phase:** 90-release-pipeline-hardening
**Areas discussed:** Pipeline split architecture, SHA pinning approach, Build tool pinning strategy, Release verification mechanism

---

## Pipeline split architecture

### Q1: How should the signed-publish trust boundary be structured?

| Option | Description | Selected |
|--------|-------------|----------|
| Build-per-platform → sign-macos → publish | Three stages. Unsigned build jobs hold no secrets. `sign-macos` holds only MACOS_*. `publish` holds only the GH release token. Windows/Linux flow straight through. | ✓ |
| Build-per-platform → merged sign+publish | Same unsigned builds, but one downstream job holds signing AND publish tokens. | |
| Full per-platform sign job + separate publish | Add `sign-windows`/`sign-linux` placeholders now even though only macOS signs today. | |

**User's choice:** Build-per-platform → sign-macos → publish (recommended)

### Q2: How should artifacts cross the unsigned→signed boundary?

| Option | Description | Selected |
|--------|-------------|----------|
| upload-artifact / download-artifact as-is | Standard GitHub-hosted artifact transfer between jobs. | |
| + SHA256 manifest check in sign job | Build emits SHA256 as job output; sign job verifies before signing. | |
| + internal attest-build-provenance | Build attests unsigned artifact; sign job verifies attestation. SLSA-aligned. | ✓ |

**User's choice:** Add internal build-provenance attestation

### Q3: Release attestation on final published artifacts?

| Option | Description | Selected |
|--------|-------------|----------|
| Defer to its own phase | Keep Phase 90 focused on SEC-09/10/11. | |
| Include — add to publish job | SLSA L2 provenance on final .dmg/.exe/.tar.gz/.deb. Verifiable via `gh attestation verify`. | ✓ |

**User's choice:** Include — add to publish job now
**Notes:** User asked for explanation of what build-provenance attestation actually protects against. After clarifying that release attestation lets downstream consumers (users, Homebrew, corporate scanners) cryptographically verify artifacts came from the actual workflow, user opted to include it. Same logic extended to internal boundary attestation.

---

## SHA pinning approach

### Q1: How should Actions be pinned and kept current?

| Option | Description | Selected |
|--------|-------------|----------|
| Manual pin with version comment | `@<sha> # v4.2.2` — official GitHub guidance. Dependabot-compatible. | ✓ |
| Tool-driven (pin-github-action or ratchet) | Auto-convert tag refs to SHA+comment via CLI tool. | |
| Dependabot-only | Skip initial pinning; let Dependabot eventually convert. | |

**User's choice:** Manual pin with version comment (recommended)

### Q2: How should Dependabot handle pinned Actions updates?

| Option | Description | Selected |
|--------|-------------|----------|
| Weekly schedule, manual merge | PRs open; human reviews and merges. | ✓ |
| Weekly + auto-merge patch | Auto-merge patch bumps on CI pass; major/minor manual. | |
| No Dependabot | Manual updates only. | |

**User's choice:** Enable with weekly schedule, manual merge (recommended)

### Q3: How to handle `vedantmgoyal9/winget-releaser@main`?

| Option | Description | Selected |
|--------|-------------|----------|
| Pin to SHA like the others | Same treatment as other Actions; keep `continue-on-error`. | |
| Pin to SHA + remove continue-on-error | Pin + hard-fail on WinGet failures. | |
| Replace with Microsoft's wingetcreate CLI | Ditch third-party action; call `wingetcreate submit` directly. First-party Microsoft tool. | ✓ |

**User's choice:** Replace with wingetcreate CLI
**Notes:** Meaningful scope — this replaces a third-party action with direct first-party CLI invocation. Research will need to cover wingetcreate semantics and auth (still needs WINGET_TOKEN for fork/PR push).

---

## Build tool pinning strategy

### Q1: Where should pinned build-tool versions live as source of truth?

| Option | Description | Selected |
|--------|-------------|----------|
| tools.go + go.mod | Standard Go convention; versions in go.mod. Dependabot covers via Go ecosystem. | ✓ |
| Inline env vars at top of each workflow | `WAILS_VERSION: v2.9.1` at workflow top. Duplicates. | |
| Dedicated tools.env sourced by all | `build/tools.env` with version constants. | |

**User's choice:** tools.go + go.mod (recommended)

### Q2: How should CI actually install the pinned tools?

| Option | Description | Selected |
|--------|-------------|----------|
| `go install $(go list -m -f ...)` | Derive version from go.mod at install time. Zero drift. | ✓ |
| Hardcoded version string | `go install wails@v2.9.1`. Multiple files to update. | |
| Shell helper script | `scripts/install-build-tools.sh` called by CI and build.sh. | |

**User's choice:** go install $(go list -m -f ...) (recommended)

---

## Release verification mechanism

### Q1: How should the split pipeline be verified end-to-end?

| Option | Description | Selected |
|--------|-------------|----------|
| Pre-release tag (vX.Y.Z-rc1) on branch | Real workflow end-to-end. Real signing, real notarization, draft release, test tap branch. | ✓ |
| workflow_dispatch dry-run mode | Build+sign but skip actual publish. Doesn't exercise publish path. | |
| Mock-secrets test workflow on PR | Fake secrets/self-signed. Doesn't satisfy SC-4 alone. | |

**User's choice:** Pre-release tag (vX.Y.Z-rc1) on a branch (recommended)

### Q2: How should the Homebrew tap side be verified?

| Option | Description | Selected |
|--------|-------------|----------|
| Test branch in homebrew-agenthub | `release-90-test` branch, merged to main only on success. | ✓ |
| Separate homebrew-agenthub-test repo | Dedicated test tap repo. | |
| Skip tap verification for rc tags | `if: !contains(github.ref, '-rc')` guard. | |

**User's choice:** Test branch in homebrew-agenthub tap (recommended)

### Q3: Should the verification tag produce a public GitHub release or a draft?

| Option | Description | Selected |
|--------|-------------|----------|
| Draft release, cleaned up after | `draft: true` when tag matches v*-rc*. No users see it. | ✓ |
| Public prerelease | Visible on GH releases page as prerelease. | |
| No actual release | Abort before upload; only confirms workflow structure. | |

**User's choice:** Draft release, cleaned up after (recommended)

---

## Claude's Discretion

Areas where the user deferred implementation detail to planning:
- Exact format of version comments (`# v4.2.2` vs `# v4.2.2 (2026-02-14)`)
- Whether the grep gate is a job in an existing workflow or a dedicated `hardening-check.yml`
- Exact file location for `tools.go` (repo root vs `internal/tools/` vs `build/tools/`)
- Whether `sign-macos` uses a composite action or keeps steps inline
- Whether rc-draft-cleanup is automated (workflow) or documented manual step

## Deferred Ideas

Ideas surfaced during discussion that are noted for future phases, not acted on in Phase 90:

- Windows code signing (EV cert + signtool) — future phase, slots into the split architecture built here
- Linux .deb GPG signing + apt repo — future phase
- Reproducible builds — separate, larger phase
- Runner hardening (step-security/harden-runner, egress controls) — future phase
- Secret rotation playbook — process work, future phase
