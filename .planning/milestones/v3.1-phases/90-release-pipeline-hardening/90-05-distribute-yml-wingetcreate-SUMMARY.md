---
phase: 90-release-pipeline-hardening
plan: "05"
subsystem: ci
tags: [ci, hardening, distribute-yml, wingetcreate, rc-guards, sha-pin, sec-09, sec-11]

requires:
  - phase: 90-release-pipeline-hardening plan 04
    provides: release.yml split into build/sign/publish with SLSA L2 attestations

provides:
  - distribute.yml fully SHA-pinned end-to-end (zero floating refs in entire repo)
  - submit-winget on windows-latest with SHA-256-verified wingetcreate.exe (D-08)
  - rc-aware tap branch routing: release-90-test on rc tags, main on real tags (D-16)
  - rc skip for WinGet submission: if !contains(github.ref, '-rc') (D-17)
  - grep-gate passes for first time in Phase 90 (SEC-09 static-analysis objective complete)

affects:
  - 90-06-e2e-rc-verification (plan 06 — all code ready, proceeds with rc tag E2E test)

tech-stack:
  added:
    - wingetcreate.exe v1.12.8.0 (Microsoft first-party WinGet manifest submission tool)
  patterns:
    - SHA-256-verified binary download before execution (tamper-evident supply chain)
    - Hyphen-anchored rc-guard: contains(github.ref, '-rc') throughout
    - rc-branch ternary: release-90-test if rc, main if not (both checkout ref and git push target)

key-files:
  created: []
  modified:
    - .github/workflows/distribute.yml
    - scripts/grep-gate.sh

key-decisions:
  - "D-02 confirmed: TAP_DEPLOY_TOKEN belongs only in distribute.yml (checkout + push to homebrew-agenthub tap); removed from release.yml in Plan 04"
  - "D-08: wingetcreate.exe requires windows-latest runner — .NET binary, does not run on Linux"
  - "D-16: rc-aware tap branch routing — checkout ref: and git push target both use contains(github.ref, '-rc') && 'release-90-test' || 'main'"
  - "D-17: submit-winget skipped entirely on rc tags via if: !contains(github.ref, '-rc') — prevents test submissions entering Microsoft WinGet review queue"
  - "grep-gate Check 2 uses \\w@latest pattern to distinguish real @latest tool-install refs from @latest as a quoted string in test assertion code"

patterns-established:
  - "SHA-256 verified binary download: Invoke-WebRequest + Get-FileHash + exit 1 on mismatch before execution"
  - "rc-ternary pattern: ${{ contains(github.ref, '-rc') && 'release-90-test' || 'main' }} used consistently for both checkout ref and push target"
  - "Chesterton's Fence: continue-on-error: true + comment preserved on submit-winget (WinGet first submission known-flaky)"

requirements-completed: [SEC-09, SEC-11]

duration: 3min
completed: "2026-04-24"
---

# Phase 90 Plan 05: distribute.yml wingetcreate Summary

**First-party wingetcreate.exe (SHA-256 verified) replaces floating vedantmgoyal9/winget-releaser@main; distribute.yml is fully SHA-pinned; grep-gate passes end-to-end for the first time in Phase 90**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-04-24T13:21:03Z
- **Completed:** 2026-04-24T13:24:09Z
- **Tasks:** 2
- **Files modified:** 2 (.github/workflows/distribute.yml, scripts/grep-gate.sh)

## Accomplishments

- Eliminated the last floating third-party ref in the repo (`vedantmgoyal9/winget-releaser@main`) — SEC-09 static-analysis objective complete
- Replaced with inline PowerShell recipe on windows-latest: downloads wingetcreate.exe v1.12.8.0, verifies SHA-256 = `8BD738851B524885410112678E3771B341C5C716DE60FBBECB88AB0A363ED85D`, then submits to WinGet
- Added rc-aware tap branch routing (D-16): checkout `ref:` and `git push origin HEAD:"$BRANCH"` both derive branch from `-rc` ternary
- Added WinGet rc skip (D-17): `if: ${{ !contains(github.ref, '-rc') }}` gates entire submit-winget job
- SHA-pinned `nick-fields/retry@v3` → `@ad984534 # v4.0.0` and `actions/checkout@v4` → `@de0fac2e # v6.0.2`
- `bash scripts/grep-gate.sh` exits 0 — first GREEN for Phase 90 Wave 0 gate

## Task Commits

Each task was committed atomically:

1. **Task 1: update-homebrew-tap SHA-pin + rc-aware branch routing** - `fe8dff9` (ci)
2. **Task 2: submit-winget wingetcreate replacement + grep-gate fix** - `a457b5e` (ci)

**Plan metadata:** _(added in final commit)_

## Files Created/Modified

- `.github/workflows/distribute.yml` — SHA-pinned both actions; added rc-aware checkout `ref:` and push `HEAD:"$BRANCH"`; replaced submit-winget job body with wingetcreate recipe on windows-latest
- `scripts/grep-gate.sh` — Check 2 pattern tightened from `@latest` to `\w@latest` to exclude quoted-string mentions in test assertion code (Rule 1 auto-fix)

## Decisions Made

- D-02 confirmed: TAP_DEPLOY_TOKEN is correctly scoped to distribute.yml only; no change needed here
- D-08 applied: windows-latest runner for submit-winget (wingetcreate.exe is .NET, Linux-incompatible)
- D-16 applied: rc-branch ternary on both checkout `ref:` and `git push origin HEAD:` target — consistent routing
- D-17 applied: hyphen-anchored `!contains(github.ref, '-rc')` on if: gate for submit-winget
- grep-gate Check 2 narrowed to `\w@latest` regex: `build-script.test.sh` Section 12 contains `@latest` as a grep pattern string in assertion code; the word-boundary prefix distinguishes real tool-install refs from string literals

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] grep-gate Check 2 false-positive on test assertion strings**
- **Found during:** Task 2 verification (running `bash scripts/grep-gate.sh`)
- **Issue:** Check 2 used bare `@latest` grep across `tests/` — the test file `tests/build-script.test.sh` Section 12 contains `@latest` as a quoted grep pattern string in assertion code (`grep -c '@latest'`, `echo "no @latest refs"`, etc.). After Task 2 eliminated the real floating refs, Check 2 started failing on these string literals.
- **Fix:** Changed Check 2 pattern from `@latest` to `\w@latest` — requires a word character immediately before `@latest`, matching real tool-install refs (`tool@latest`) but not quoted string mentions (`'@latest'`)
- **Verification:** `go install tool@latest` still caught; `grep -c '@latest'` in test file not caught; `bash scripts/grep-gate.sh` exits 0
- **Files modified:** `scripts/grep-gate.sh`
- **Committed in:** `a457b5e` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug in grep-gate false-positive)
**Impact on plan:** Required to satisfy the plan's "grep-gate passes" success criterion. No scope creep — grep-gate was Phase 90 infrastructure, the fix makes it correctly detect real violations without false-positiving on test infrastructure.

## Issues Encountered

None beyond the Rule 1 auto-fix above.

## User Setup Required

None — no external service configuration required.

**Pre-flight for Plan 06 (E2E rc verification):**
1. Manually create `release-90-test` branch on `scottkw/homebrew-agenthub` (see `90-TAP-BRANCH-SETUP.md`) — required before rc tag triggers tap checkout step
2. `gh api /repos/scottkw/agenthub/environments/release` — audit environment protection rules (RESEARCH Open Q4)

## Next Phase Readiness

All code changes are in place for Phase 90 Plan 06 end-to-end rc verification:
- distribute.yml: rc-aware tap routing + wingetcreate on windows-latest
- release.yml (Plan 04): three-stage build/sign/publish with SLSA L2 attestations
- grep-gate: GREEN across all four workflow files
- Remaining Plan 06 blockers: (1) `release-90-test` branch precreation on homebrew-agenthub tap, (2) real `v3.1.0-rc1` tag push to trigger the full pipeline

---
*Phase: 90-release-pipeline-hardening*
*Completed: 2026-04-24*
