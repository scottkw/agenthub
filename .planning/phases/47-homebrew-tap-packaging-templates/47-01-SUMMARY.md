---
phase: 47-homebrew-tap-packaging-templates
plan: "01"
subsystem: infra
tags: [homebrew, winget, packaging, distribution, cask, ruby]

# Dependency graph
requires:
  - phase: 46-release-build-pipeline
    provides: "DMG and NSIS artifact naming conventions (agenthub-${VERSION}-darwin-universal.dmg, agenthub-${VERSION}-windows-amd64-installer.exe)"
provides:
  - "packaging/homebrew/agenthub.rb.template: Homebrew cask formula template with {{VERSION}} and {{SHA256}} placeholders"
  - "packaging/winget/manifests/: Three WinGet manifest files at schema 1.12.0 (version, installer, locale)"
affects:
  - 47-02 (distribute.yml uses Homebrew template as source-of-truth)
  - 48-winget-submission (WinGet manifests are base templates for Phase 48 submission)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Homebrew cask template pattern: {{VERSION}}/{{SHA256}} sentinel tokens sed-replaced by distribute.yml; Ruby #{version} interpolation in URL stanza"
    - "WinGet three-file manifest structure at schema 1.12.0: version + installer + locale YAMLs"

key-files:
  created:
    - packaging/homebrew/agenthub.rb.template
    - packaging/winget/manifests/scottkw.agenthub.yaml
    - packaging/winget/manifests/scottkw.agenthub.installer.yaml
    - packaging/winget/manifests/scottkw.agenthub.locale.en-US.yaml
  modified: []

key-decisions:
  - "InstallerType: nullsoft (not exe) — Phase 46 produces NSIS installer; nullsoft enables automatic /S silent install in WinGet"
  - "License: Proprietary — no LICENSE file exists in repo; update to SPDX identifier if LICENSE added before Phase 48"
  - "depends_on macos: '>= :ventura' — safe baseline; Tailscale Go client library targets macOS 12+, Ventura is oldest commonly-supported macOS in 2026"
  - "Homebrew template uses Ruby #{version} interpolation in URL, not {{VERSION}} — sed replaces only the version/sha256 stanza values; Homebrew DSL evaluates #{version} at install time"

patterns-established:
  - "Packaging templates live in packaging/ at repo root, not in .github/ — source of truth for distribution scripts"
  - "WinGet manifests use three-file split (version + installer + locale) for schema 1.12.0 compliance"

requirements-completed: [DIST-01, DIST-04]

# Metrics
duration: 5min
completed: 2026-04-05
---

# Phase 47 Plan 01: Packaging Templates Summary

**Homebrew cask template and three-file WinGet manifests (schema 1.12.0) with sed-replaceable {{VERSION}} tokens matching Phase 46 artifact naming exactly**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-04-05T00:25:00Z
- **Completed:** 2026-04-05T00:30:00Z
- **Tasks:** 1
- **Files modified:** 4

## Accomplishments

- Created Homebrew cask formula template with `{{VERSION}}`/`{{SHA256}}` placeholders; URL pattern matches Phase 46 DMG naming (`agenthub-v#{version}-darwin-universal.dmg`); valid Ruby syntax confirmed via `ruby -c`
- Created three WinGet manifests at schema 1.12.0 (version, installer, locale) with `InstallerType: nullsoft` matching Phase 46 NSIS build; `{{WINDOWS_SHA256}}` placeholder distinct from Homebrew SHA256
- All URL patterns verified against Phase 46 release.yml artifact naming convention

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Homebrew cask template and WinGet manifest templates** - `937565c` (feat)

**Plan metadata:** (pending docs commit)

## Files Created/Modified

- `packaging/homebrew/agenthub.rb.template` - Homebrew cask formula template; `{{VERSION}}`/`{{SHA256}}` replaced by distribute.yml sed; `#{version}` in URL evaluated by Homebrew at install time
- `packaging/winget/manifests/scottkw.agenthub.yaml` - WinGet version manifest (ManifestType: version, schema 1.12.0)
- `packaging/winget/manifests/scottkw.agenthub.installer.yaml` - WinGet installer manifest (nullsoft/NSIS, x64, URL matches Phase 46)
- `packaging/winget/manifests/scottkw.agenthub.locale.en-US.yaml` - WinGet default locale manifest (License: Proprietary pending LICENSE file)

## Decisions Made

- `InstallerType: nullsoft` chosen because Phase 46 uses `nsis: true` in wails-build-action, producing NSIS installer; `nullsoft` enables automatic `/S` silent install inference in WinGet without explicit InstallerSwitches
- `License: Proprietary` used as placeholder; no LICENSE file exists in repo as of 2026-04-04; must update to SPDX identifier before Phase 48 submission
- `depends_on macos: ">= :ventura"` (macOS 13.0) chosen as safe minimum; Tailscale Go client library targets macOS 12+; Ventura is oldest commonly-supported macOS in 2026

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required for template files themselves.

Note: Phase 47 Plan 02 (distribute.yml) requires `TAP_DEPLOY_TOKEN` classic PAT and `scottkw/homebrew-agenthub` GitHub repo to be created before the workflow can run.

## Next Phase Readiness

- Homebrew cask template ready for use by distribute.yml (Plan 02)
- WinGet manifests ready as base templates for Phase 48 manual submission to microsoft/winget-pkgs
- Both template sets use correct URL patterns matching Phase 46 artifact naming

---
*Phase: 47-homebrew-tap-packaging-templates*
*Completed: 2026-04-05*
