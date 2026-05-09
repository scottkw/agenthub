---
phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate
plan: "04"
subsystem: testing
tags: [playwright, e2e, csp, github-actions, firefox, webkit, chromium, cross-browser]

# Dependency graph
requires:
  - phase: 93-vendoring-discipline-web-parity-for-already-shipping-addons
    provides: web-csp.spec.ts and playwright.config.ts baseline (single chromium project)
  - phase: 96-image-addon-csp-audit
    provides: final CSP header (script-src 'self' 'wasm-unsafe-eval')
provides:
  - "Three Playwright browser projects: chromium, firefox, webkit in playwright.config.ts"
  - "Verified zero CSP violations on all three browser engines against Phase 96 CSP header"
  - "GitHub Actions workflow .github/workflows/e2e.yml for CI enforcement of cross-browser CSP gate"
affects:
  - 99-verification
  - future-phases-touching-csp-or-e2e

# Tech tracking
tech-stack:
  added:
    - "Playwright Firefox 148.0.2 (playwright firefox v1511)"
    - "Playwright WebKit 26.4 (playwright webkit v2272)"
  patterns:
    - "Three-project Playwright config: chromium -> firefox -> webkit (fastest-sanity-check ordering)"
    - "workers: 1 / fullyParallel: false preserved for single-instance Go fixture"
    - "SHA-pinned GitHub Actions per Phase 90 SEC-09 (all pins copied verbatim from build.yml)"

key-files:
  created:
    - .github/workflows/e2e.yml
  modified:
    - frontend/playwright.config.ts

key-decisions:
  - "SHA pins in e2e.yml copied verbatim from build.yml/release.yml (SEC-09 uniformity): checkout v6.0.2, setup-go v6.4.0, pnpm/action-setup v6.0.3, setup-node v6.4.0, upload-artifact v7.0.1"
  - "Project ordering chromium -> firefox -> webkit: chromium first as fastest sanity check, webkit last as most-similar-to-iPad-Safari proxy"
  - "HUMAN REVIEW REQUIRED comment added to e2e.yml header listing all four review items (triggers, runner, CI cost, Option 2 deferral)"
  - "No per-project use override added: Desktop Chrome/Firefox/Safari devices supply relevant defaults"

patterns-established:
  - "Multi-browser Playwright: extend projects[] array; no per-project global-use override needed"
  - "e2e.yml Go setup required: global-setup.ts runs go build for playwright-fixture binary"

requirements-completed: []

# Metrics
duration: 25min
completed: "2026-05-08"
---

# Phase 99 Plan 04: Cross-Browser CSP e2e + CI Workflow Summary

**Playwright extended to chromium + firefox + webkit with zero CSP violations on all three engines; GitHub Actions e2e.yml committed (flagged for human review per autonomous: false)**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-05-08T18:53:00Z
- **Completed:** 2026-05-08T19:18:17Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Extended `frontend/playwright.config.ts` with firefox (Desktop Firefox) and webkit (Desktop Safari) projects, preserving chromium first and keeping `workers: 1` / `fullyParallel: false` for the single-instance Go fixture.
- Ran `pnpm exec playwright test web-csp.spec.ts` locally — all 3 tests passed (chromium 4.0s, firefox 4.8s, webkit 7.2s), confirming zero CSP violations across all three browser engines.
- Created `.github/workflows/e2e.yml` with SHA pins copied verbatim from build.yml (Phase 90 SEC-09 uniformity), triggering on push/PR to main, running `playwright install --with-deps chromium firefox webkit` then `pnpm exec playwright test`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend playwright.config.ts with firefox + webkit projects** - `2d7ec8c` (feat)
2. **Task 2: Create .github/workflows/e2e.yml + human review** - `8533c69` (feat)

## Files Created/Modified

- `frontend/playwright.config.ts` - Extended projects[] array: chromium -> firefox -> webkit; all other config keys preserved verbatim
- `.github/workflows/e2e.yml` - New GitHub Actions workflow: cross-browser Playwright e2e on push/PR to main, SHA-pinned per SEC-09, HUMAN REVIEW REQUIRED comment

## Local Install Command

```bash
cd frontend && pnpm exec playwright install --with-deps firefox webkit
```

(chromium is already installed as part of the existing Playwright setup)

## e2e.yml Workflow Shape

| Step | Action | SHA Pin | Version |
|------|--------|---------|---------|
| Checkout | actions/checkout | de0fac2e4500dabe0009e67214ff5f5447ce83dd | v6.0.2 |
| Set up Go | actions/setup-go | 4a3601121dd01d1626a1e23e37211e3254c1c06c | v6.4.0 |
| Set up pnpm | pnpm/action-setup | 903f9c1a6ebcba6cf41d87230be49611ac97822e | v6.0.3 |
| Set up Node.js | actions/setup-node | 48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e | v6.4.0 |
| Upload report | actions/upload-artifact | 043fb46d1a93c77aae656e7c1c64a875d1fc6a0a | v7.0.1 |

All SHAs copied verbatim from `.github/workflows/build.yml` — zero drift from Phase 90 pins.

## Test Result Evidence

```
Running 3 tests using 1 worker

  ✓  1 [chromium] › e2e/web-csp.spec.ts:18:7 › Phase 93 WEB-02 web-csp zero-violation › no CSP violations during attach/scroll session (4.0s)
  ✓  2 [firefox] › e2e/web-csp.spec.ts:18:7 › Phase 93 WEB-02 web-csp zero-violation › no CSP violations during attach/scroll session (4.8s)
  ✓  3 [webkit] › e2e/web-csp.spec.ts:18:7 › Phase 93 WEB-02 web-csp zero-violation › no CSP violations during attach/scroll session (7.2s)

  3 passed (22.1s)
```

Zero CSP violations on Chromium, Firefox, and WebKit against the Phase 96 header (`script-src 'self' 'wasm-unsafe-eval'`).

## Decisions Made

- SHA pins in e2e.yml copied verbatim from build.yml/release.yml per Phase 90 SEC-09 (uniform SHA-pin convention across all workflows). Plan skeleton had older v4 pins — replaced with repo's actual v6 pins.
- Set up Go step included in e2e.yml because global-setup.ts calls `go build -tags=playwrightfixture` to build the playwright-fixture binary. The workflow would fail without Go available.
- Node version set to 22 to match build.yml/release.yml (plan skeleton had 20).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Accuracy] SHA pins updated from plan skeleton to repo-actual versions**
- **Found during:** Task 2 (creating e2e.yml)
- **Issue:** Plan skeleton used older v4 SHA pins (e.g., `actions/checkout@b4ffde65...` for v4.1.1). The repo uses v6 pins per build.yml/release.yml. The threat model T-99-11 explicitly mandates copying pins from build.yml verbatim.
- **Fix:** Used build.yml/release.yml SHA pins verbatim: checkout v6.0.2, setup-go v6.4.0, pnpm v6.0.3, setup-node v6.4.0, upload-artifact v7.0.1.
- **Files modified:** .github/workflows/e2e.yml
- **Verification:** `diff <(grep -oE 'uses: actions/checkout@[a-f0-9]{40}' build.yml) <(grep ...)` returns empty for all shared actions.
- **Committed in:** 8533c69 (Task 2 commit)

**2. [Rule 2 - Missing Critical] Node version updated from 20 to 22**
- **Found during:** Task 2 (creating e2e.yml)
- **Issue:** Plan skeleton specified `node-version: '20'` but build.yml/release.yml use `node-version: 22` for consistency.
- **Fix:** Changed to 22 to match existing workflows.
- **Files modified:** .github/workflows/e2e.yml
- **Committed in:** 8533c69 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 accuracy/bug, 1 missing critical for consistency)
**Impact on plan:** Both fixes required for SEC-09 compliance and workflow consistency. No scope creep.

## Issues Encountered

- `pnpm exec playwright install` failed initially because `pnpm install --frozen-lockfile` had not been run in the worktree. Installed deps first, then browsers downloaded successfully.
- Playwright fixture binary not present in worktree `.playwright/` — copied from main repo's `.playwright/playwright-fixture` (already built there from Phase 93 runs).

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes. The only new surface is the GitHub Actions workflow file — covered by T-99-11 (SHA pin uniformity) and T-99-12 (artifact upload on failure only, 7-day retention).

## Known Stubs

None. The three browser projects are fully wired and the CSP spec runs green on all three.

## Next Phase Readiness

- SC-4 cross-browser CSP portion satisfied: zero violations on Chromium + Firefox + WebKit (the `script-src 'self' 'wasm-unsafe-eval'` header is universally compliant).
- `.github/workflows/e2e.yml` committed and flagged for human review — human should verify SHA pins, triggers, and CI cost before merge.
- SC-4 iPad Safari Tailscale UAT portion is owned by Plan 99-05.

---
*Phase: 99-settings-ui-polish-migration-final-csp-audit-release-gate*
*Completed: 2026-05-08*
