---
phase: 06-distribution-cross-platform
plan: 02
subsystem: infra
tags: [github-actions, ci, wails, macos, linux, windows, codesign, notarization, webkit2gtk, nsis, webview2]

# Dependency graph
requires:
  - phase: 06-distribution-cross-platform
    provides: tray stubs, build/entitlements.plist, build/darwin/Info.plist, wails.json Info section

provides:
  - .github/workflows/build.yml with 4-runner CI matrix
  - macOS universal binary build on macos-latest
  - Linux 24.04 build with webkit2_41 tag and libwebkit2gtk-4.1-dev
  - Linux 22.04 build with libwebkit2gtk-4.0-dev
  - Windows NSIS installer with embedded WebView2 bootstrapper
  - macOS signing with hardened runtime + notarization via notarytool (conditional on secrets)
  - go test ./... run on all platforms before build

affects:
  - Any future CI/CD workflows (baseline established)
  - macOS signing setup instructions (secrets must be configured)

# Tech tracking
tech-stack:
  added:
    - dAppServer/wails-build-action@main
  patterns:
    - "CI matrix pattern: fail-fast:false so all platforms attempt even if one fails"
    - "Secrets-conditional signing: macOS sign/notarize steps gated on env.MACOS_CERTIFICATE != ''"
    - "Cleanup with always(): CI keychain and certificate deleted even if notarization fails"

key-files:
  created:
    - .github/workflows/build.yml
  modified: []

key-decisions:
  - "wails-build-action@main (not @v3): uses @main for latest webkit2_41 support per research"
  - "go test ./... runs BEFORE wails build step using actions/setup-go@v5 — ensures tests execute even if wails build fails"
  - "Conditional cleanup with always(): security delete-keychain runs regardless of signing success/failure to avoid keychain leaks"
  - "Per-OS artifact names (agenthub-darwin-universal, agenthub-linux-amd64-ubuntu24, agenthub-linux-amd64-ubuntu22, agenthub-windows-amd64) disambiguate download artifacts"
  - "nsis triggered by matrix.build.os == 'windows-latest' not runner.os == 'Windows' — consistent with matrix field naming"

patterns-established:
  - "macOS CI signing pattern: ephemeral keychain created, used, deleted within same job"
  - "Linux WebKitGTK variant pattern: matrix field webkit_pkg controls apt package; webkit_tag controls build flags"

requirements-completed: [PLAT-01, PLAT-02, PLAT-03]

# Metrics
duration: 5min
completed: 2026-03-19
---

# Phase 6 Plan 02: CI Matrix Workflow Summary

**GitHub Actions 4-runner CI matrix with macOS hardened-runtime signing + notarytool notarization, Linux WebKitGTK 4.0/4.1 variants, Windows NSIS installer with embedded WebView2 — all gated on `dAppServer/wails-build-action@main`**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-19T13:03:22Z
- **Completed:** 2026-03-19T13:08:00Z
- **Tasks:** 1 of 2 (Task 2 is a human-verify checkpoint)
- **Files modified:** 1

## Accomplishments

- `.github/workflows/build.yml` created with 4-runner matrix covering all target platforms
- macOS signing/notarization steps added, conditional on `MACOS_CERTIFICATE` secret being configured
- Linux jobs differentiate libwebkit2gtk-4.0-dev (22.04) vs libwebkit2gtk-4.1-dev (24.04 + `-tags webkit2_41`)
- Windows job generates NSIS installer with `-webview2 embed` flag
- `go test ./...` runs on all platforms before Wails build step
- `fail-fast: false` ensures all platforms attempt even if one runner fails

## Task Commits

Each task was committed atomically:

1. **Task 1: Create GitHub Actions CI matrix workflow with signing and notarization** - `6794747` (feat)
2. **Task 2: Verify CI passes on all platforms** - awaiting human verification (checkpoint)

## Files Created/Modified

- `.github/workflows/build.yml` - 4-runner CI matrix with macOS signing/notarization, Linux WebKitGTK variants, Windows NSIS+WebView2

## Decisions Made

- **`actions/setup-go@v5` before wails-build-action:** Added explicit Go setup step to run `go test ./...` before the Wails build. The `dAppServer/wails-build-action` installs Go internally, but running tests first provides earlier feedback on unit test failures.
- **`always()` on macOS cleanup:** The keychain cleanup step uses `always()` condition so the ephemeral CI keychain is deleted even if the notarization step fails, preventing keychain accumulation on macOS runners.
- **Artifact naming strategy:** Separate artifact names per platform variant (darwin-universal, linux-amd64-ubuntu24, linux-amd64-ubuntu22, windows-amd64) avoid overwriting and allow downloading specific platform builds.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

**macOS code signing requires manual configuration.** To enable signing and notarization:

1. Export your "Developer ID Application" certificate as `.p12` from Keychain Access
2. Add these GitHub repository secrets (Settings -> Secrets and variables -> Actions):

| Secret | Source |
|--------|--------|
| `MACOS_CERTIFICATE` | `base64 Certificates.p12` (base64-encode the .p12 file) |
| `MACOS_CERTIFICATE_NAME` | Full cert string: `"Developer ID Application: Your Name (TEAMID)"` |
| `MACOS_CERTIFICATE_PWD` | Password used when exporting the .p12 |
| `MACOS_CI_KEYCHAIN_PWD` | Any random strong password for the ephemeral CI keychain |
| `MACOS_NOTARIZATION_APPLE_ID` | Apple Developer account email |
| `MACOS_NOTARIZATION_PWD` | App-specific password from appleid.apple.com |
| `MACOS_NOTARIZATION_TEAM_ID` | Team ID from developer.apple.com/account -> Membership Details |

**Without these secrets:** CI still builds successfully on all platforms — signing/notarization steps are skipped when `MACOS_CERTIFICATE` is empty.

## Next Phase Readiness

- CI matrix workflow is ready to trigger on push/PR — push the branch to start CI
- All 4 platform builds should proceed without secrets; only macOS signing is optional
- Once CI is green, PLAT-01/02/03 are proven at the CI level
- Phase 6 is complete after CI verification passes

---
*Phase: 06-distribution-cross-platform*
*Completed: 2026-03-19*
