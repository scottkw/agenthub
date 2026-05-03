---
phase: 90-release-pipeline-hardening
plan: 06
status: partial
completed: 2026-04-24
tasks_completed: 0/3
reason: "autonomous: false — destructive external actions (live git tag push, release workflow run, UAT cleanup) deferred to human execution"
---

# Plan 90-06: E2E RC Verification — Partial (Handed Off to Human UAT)

## Status

`autonomous: false` — this plan requires actions Claude cannot autonomously perform:
1. Pushing a real `v3.1.0-rc1` git tag to `scottkw/agenthub` (live CI trigger, GitHub Actions minutes consumed)
2. Creating `release-90-test` branch on `scottkw/homebrew-agenthub` (cross-repo write)
3. Running `gh attestation verify` on downloaded artifacts (requires downloading signed binaries)
4. Cleanup decisions that affect public distribution (delete draft release, rewind tap branch)

Per D-14, D-15, D-16, D-17 — these must be human-gated for safety.

## What Is Verifiable Now (Static)

Plans 01-05 delivered everything that CAN be verified without a live tag cut:

| Static verification | Status |
|---------------------|--------|
| `bash scripts/grep-gate.sh` — zero floating refs across all workflows | ✅ PASS |
| `bash tests/build-script.test.sh` — 38 assertions incl. Section 12 SEC-10 | ✅ 38/38 |
| `go build .` — wails v2.12.0 + nfpm v2.33.1 compile | ✅ PASS |
| `go vet .` — clean | ✅ PASS |
| SHA-pinned actions in build.yml (8), release.yml (11), distribute.yml (5), release-please.yml (1) | ✅ all 40-char SHA + # vX.Y.Z |
| sign-macos is the sole holder of `environment: release` + MACOS_* secrets | ✅ (release.yml line-by-line inspection) |
| Internal attestation verify step (`gh attestation verify --bundle`) present in sign-macos before codesign | ✅ |
| `draft: contains(github.ref, '-rc')` hyphen-anchored in publish step | ✅ |
| wingetcreate.exe + SHA-256 `8BD7...D85D` verification replaces `winget-releaser@main` | ✅ |
| `if: !contains(github.ref, '-rc')` hyphen-anchored on submit-winget | ✅ |
| rc-aware checkout `ref:` + `git push HEAD:"$BRANCH"` in update-homebrew-tap | ✅ |

## What Requires Human UAT

Everything from Plan 06's `truths` — a live tag push and the downstream observations — is tracked in `.planning/phases/90-release-pipeline-hardening/90-06-HUMAN-UAT.md` (15-item checklist):

1. Pre-flight (static bundle green + tap branch created + env audit + build.yml green + no blocking Dependabot PRs)
2. Tag cut (`v3.1.0-rc1` pushed; release.yml observed end-to-end; 6 assets materialized; draft=true)
3. External verification (all 6 `gh attestation verify` succeed; distribute.yml run with rc behavior; tap-branch-only update; submit-winget skipped)
4. Cleanup + UAT report signed off

See `90-06-HUMAN-UAT.md` for the runnable checklist, expected commands, and fill-in result fields.

## Why This Is Not a Failure

Plan 06 is explicitly marked `autonomous: false` and `type: checkpoint:human-action`/`checkpoint:human-verify` in its PLAN.md. The expected flow is for the orchestrator to pause at this plan and surface human-gated verification items. That is what happened.

Phase 90's SC-1, SC-2, SC-3 (static hardening) are fully satisfied by Plans 01-05. SC-4 (E2E verification) becomes satisfied when the user completes the `90-06-HUMAN-UAT.md` checklist and produces `90-UAT-REPORT.md`.

## Next Steps (User-Gated)

- Run the 15-item checklist in `90-06-HUMAN-UAT.md` at your own pace.
- When all items pass, run `/gsd-verify-work 90` to reconcile the UAT against the phase goal.
- Alternately: cut the real `v3.1.0` tag directly if you prefer to skip rc verification (not recommended — D-14 explicitly requires the rc cut as the SC-4 gate).

## Traceability

- Requirements attempted: SEC-09, SEC-10, SEC-11 (all three depend on the E2E run for full verification)
- Upstream summaries: 90-01, 90-02, 90-03, 90-04, 90-05 (all complete)
- Pending artifacts: `90-UAT-REPORT.md` (user creates during UAT)
