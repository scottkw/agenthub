---
phase: 13-build-script
plan: 02
subsystem: infra
tags: [bash, codesign, notarytool, notarization, macos, apple-developer, gatekeeper]

# Dependency graph
requires:
  - phase: 13-build-script/13-01
    provides: build.sh with sign_and_notarize() stub, codesign and notarization pipeline skeleton
provides:
  - Full macOS code-signing and notarization pipeline in build.sh sign_and_notarize()
  - Env var validation with setup instructions for all four Apple credentials
  - ditto-based notarization archive (preserves macOS extended attributes)
  - notarytool submit --wait invocation (avoids exit-0 false-success trap)
  - spctl --assess Gatekeeper final verification
affects: [release-workflow, ci-cd, distribution]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "codesign --deep --force --options runtime --timestamp --entitlements for hardened runtime"
    - "ditto -c -k --keepParent instead of zip -r for notarization archive"
    - "notarytool store-credentials + submit --wait for safe async notarization"
    - "xcrun stapler staple to attach ticket before distribution"
    - "spctl --assess --type install as final Gatekeeper acceptance check"

key-files:
  created: []
  modified:
    - build.sh

key-decisions:
  - "ditto -c -k --keepParent used (NOT zip -r) — preserves macOS extended attributes required by notarytool"
  - "notarytool submit --wait is mandatory — without --wait, exit 0 does not mean notarization succeeded"
  - "--options runtime enables hardened runtime, required for notarization since macOS 10.14.5"
  - "--timestamp uses Apple timestamp server, required for notarization"
  - "All four env vars (MACOS_SIGNING_IDENTITY, MACOS_APPLE_ID, MACOS_TEAM_ID, MACOS_APP_PASSWORD) checked upfront before any signing begins"
  - "notarytool store-credentials saves to keychain profile 'agenthub-notarize' for secure credential handling"
  - "Temp zip (build/bin/agenthub-notarize.zip) cleaned up after notarization regardless of outcome"

patterns-established:
  - "Upfront env var validation pattern: collect MISSING array, print clear error with setup instructions, exit 1"
  - "7-step macOS signing pipeline: codesign -> verify -> ditto zip -> store-credentials -> notarytool submit --wait -> stapler staple -> spctl assess"

requirements-completed: [BUILD-05]

# Metrics
duration: ~10min
completed: 2026-03-20
---

# Phase 13 Plan 02: macOS Code Signing and Notarization Pipeline Summary

**Full 7-step macOS signing pipeline in build.sh: codesign with hardened runtime + ditto archive + notarytool --wait + stapler staple + spctl Gatekeeper verify**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-20T13:30:12Z
- **Completed:** 2026-03-20
- **Tasks:** 2 (1 auto + 1 checkpoint:human-verify)
- **Files modified:** 1

## Accomplishments

- Replaced sign_and_notarize() stub with full production-ready implementation
- Upfront validation of all four Apple Developer credentials with clear setup instructions on missing
- Correct ditto-based zip (not zip -r) to preserve macOS extended attributes for notarytool
- notarytool submit --wait avoids silent exit-0 false-success trap documented in research
- Gatekeeper verification via spctl --assess as final step confirms distribution readiness
- Human-verified: missing credentials path exits 1 with all four var names and setup instructions; build without --sign proceeds without signing

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement sign_and_notarize() function in build.sh** - `4a2e0a4` (feat)

**Plan metadata:** (docs: complete plan — this commit)

## Files Created/Modified

- `build.sh` - sign_and_notarize() function fully implemented with 7-step codesign/notarize/staple/verify pipeline

## Decisions Made

- `ditto -c -k --keepParent` used instead of `zip -r` — zip strips macOS extended attributes (com.apple.quarantine etc.) that notarytool requires
- `notarytool submit --wait` is non-negotiable — without it the command exits 0 even when notarization fails asynchronously
- `--options runtime` + `--timestamp` are both required by Apple for notarization (hardened runtime + trusted timestamp)
- Entitlements file `build/entitlements.plist` always passed to codesign (network.client + network.server needed for the app to function after notarization)
- Credentials stored in keychain profile "agenthub-notarize" via `store-credentials` rather than passing on CLI (avoids exposing password in shell history/process list)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

**External services require manual configuration.** Apple Developer credentials needed to use `--sign`:

| Variable | Source |
|----------|--------|
| `MACOS_SIGNING_IDENTITY` | `security find-identity -v -p codesigning` — look for "Developer ID Application: Name (TEAMID)" |
| `MACOS_APPLE_ID` | Your Apple ID email used for developer account |
| `MACOS_TEAM_ID` | Apple Developer portal -> Membership -> Team ID |
| `MACOS_APP_PASSWORD` | appleid.apple.com -> Sign In & Security -> App-Specific Passwords |

Running `bash build.sh --platform macos --sign` without these set will print a clear error listing all four with setup instructions.

## Next Phase Readiness

- Phase 13 (build-script) is now complete — both plans executed
- build.sh supports `--platform macos|windows|linux`, `--all`, and `--sign` for macOS notarized builds
- The signing pipeline requires real Apple Developer credentials to test the full end-to-end flow; missing credentials path verified manually
- No further phases planned for v1.1 milestone

---
*Phase: 13-build-script*
*Completed: 2026-03-20*
