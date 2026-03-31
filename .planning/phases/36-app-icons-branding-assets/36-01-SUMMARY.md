---
phase: 36-app-icons-branding-assets
plan: 01
subsystem: ui
tags: [branding, icons, icns, ico, png, imagemagick, iconutil, sips, wails]

# Dependency graph
requires: []
provides:
  - "1024x1024 branded appicon.png (transparent background, A logomark)"
  - "10-entry macOS iconfile.icns (590KB, all 5 sizes x 2 densities)"
  - "6-frame Windows icon.ico (16/32/48/64/128/256px)"
  - "6 Linux PNGs (16/32/48/128/256/512px) in build/linux/"
  - "agenthub-title-logo.png copied to frontend/src/assets/ for Phase 37"
  - "scripts/verify-icons.sh: 13-check automated icon asset verification"
  - "Production .app bundle with branded ICNS injected (590KB vs Wails 3-size 361KB)"
affects:
  - "37-splash-screen (uses frontend/src/assets/agenthub-title-logo.png)"
  - "41-tray-icon (uses build/appicon.png as source)"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "iconutil compile pattern: sips resize all 10 iconset entries -> iconutil -c icns"
    - "Wails ICNS injection pattern: wails build -> cp pre-built iconfile.icns into bundle Resources"
    - "ImageMagick multi-frame ICO: magick src.png ( -clone 0 -resize NxN ) ... -delete 0 out.ico"

key-files:
  created:
    - build/AppIcon.iconset/icon_16x16.png
    - build/AppIcon.iconset/icon_16x16@2x.png
    - build/AppIcon.iconset/icon_32x32.png
    - build/AppIcon.iconset/icon_32x32@2x.png
    - build/AppIcon.iconset/icon_128x128.png
    - build/AppIcon.iconset/icon_128x128@2x.png
    - build/AppIcon.iconset/icon_256x256.png
    - build/AppIcon.iconset/icon_256x256@2x.png
    - build/AppIcon.iconset/icon_512x512.png
    - build/AppIcon.iconset/icon_512x512@2x.png
    - build/darwin/iconfile.icns
    - build/linux/16x16.png
    - build/linux/32x32.png
    - build/linux/48x48.png
    - build/linux/128x128.png
    - build/linux/256x256.png
    - build/linux/512x512.png
    - frontend/src/assets/agenthub-title-logo.png
    - scripts/verify-icons.sh
  modified:
    - build/appicon.png
    - build/windows/icon.ico
    - .gitignore

key-decisions:
  - "Extract A logomark from title logo via ImageMagick crop (185x185+5+10) + fuzz 30% background removal + trim, then center on 1024x1024 transparent canvas with 700px mark size (15% padding)"
  - "Use iconutil (not jackmordaunt/icns) for ICNS generation: Wails auto-gen only produces 3 sizes; iconutil produces all 10 required by Apple HIG"
  - "Post-build ICNS injection: run wails build then cp build/darwin/iconfile.icns into bundle Resources (Wails always overwrites; pre-placement not supported)"
  - "Transparent background for appicon.png: macOS applies standardized rounded corners + drop shadow to transparent icons, producing most native appearance"
  - "Added !build/darwin/iconfile.icns exception to .gitignore: it is a source asset (pre-built branded ICNS), not a Wails build output"

patterns-established:
  - "Pattern 1: Icon source pipeline - docs/agenthub-title-logo.png -> crop -> fuzz transparent -> trim -> 1024x1024 center -> sips iconset -> iconutil icns"
  - "Pattern 2: Post-build ICNS injection - always run after wails build to get 10-entry ICNS vs Wails 3-entry auto-gen"

requirements-completed: [BRND-01]

# Metrics
duration: 37min
completed: 2026-03-31
---

# Phase 36 Plan 01: App Icons & Branding Assets Summary

**1024x1024 branded A logomark extracted from title logo, compiled into full 10-entry macOS ICNS (590KB), 6-frame Windows ICO, and 6 Linux PNGs via sips+iconutil+ImageMagick pipeline with post-build bundle injection**

## Performance

- **Duration:** ~37 min
- **Started:** 2026-03-31T15:10:00Z
- **Completed:** 2026-03-31T15:47:00Z
- **Tasks:** 2 auto-tasks complete, 1 checkpoint awaiting human visual verify
- **Files modified:** 22 (19 created, 3 modified)

## Accomplishments

- Extracted AgentHub "A" logomark from 805x208 title logo (crop + background removal + trim) and produced clean 1024x1024 transparent PNG
- Generated complete macOS AppIcon.iconset (10 entries) and compiled to iconfile.icns (590KB) via iconutil — replacing Wails auto-gen which only produces 3 sizes
- Ran production `wails build -tags wailsassets` successfully; injected full 10-entry ICNS into app bundle (590KB vs 361KB Wails auto-gen)
- Generated 6-frame Windows ICO and 6 Linux PNGs for all required desktop integration sizes
- Created 13-check automated verification script (scripts/verify-icons.sh) — all checks pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Create square logomark and generate all icon format assets** - `31c548b` (feat)
2. **Task 1 gitignore fix: track iconfile.icns** - `c569219` (chore)
3. **Task 2: Production build + ICNS injection verification** - `e4f5104` (chore)
4. **Task 3: Visual verification checkpoint** - awaiting human approval

## Files Created/Modified

- `build/appicon.png` - 1024x1024 branded A logomark (transparent background, replaces 256x256 placeholder)
- `build/AppIcon.iconset/` - All 10 macOS iconset PNGs (16x16 through 512x512@2x)
- `build/darwin/iconfile.icns` - Full 10-entry ICNS (590KB), matches CFBundleIconFile="iconfile" in Info.plist
- `build/windows/icon.ico` - 6-frame ICO (16/32/48/64/128/256px) from branded logomark
- `build/linux/{16,32,48,128,256,512}x{...}.png` - Linux desktop integration PNGs
- `frontend/src/assets/agenthub-title-logo.png` - Title logo copy for Phase 37 splash screen
- `scripts/verify-icons.sh` - 13-check automated verification script (all pass)
- `.gitignore` - Added !build/darwin/iconfile.icns exception

## Decisions Made

- **Logomark extraction approach:** Crop 185x185 from x=5,y=10 of title logo, apply 30% fuzz transparent background removal, trim auto-detect, then resize to 700px centered on 1024x1024 transparent canvas with 15% padding
- **ICNS tool choice:** iconutil (macOS built-in) over jackmordaunt/icns — Wails auto-gen produces only 3 sizes (ic10/ic14/ic13); success criterion requires 10
- **Post-build injection:** Required because Wails always regenerates ICNS from appicon.png into bundle Resources during build; pre-placement in build/darwin/ is NOT used by Wails
- **Transparent background:** macOS applies standardized rounded corners + drop shadow to transparent PNG icons for most native appearance
- **gitignore exception:** iconfile.icns added to !-exception list because it is a pre-built source asset, not a runtime build output

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed verify-icons.sh crashing due to ((FAIL++)) with set -e**
- **Found during:** Task 1 (verification script creation)
- **Issue:** `((FAIL++))` returns exit code 1 when result is 0 (false in bash arithmetic), triggering `set -e` to abort
- **Fix:** Changed `((PASS++))` and `((FAIL++))` to `PASS=$((PASS+1))` and `FAIL=$((FAIL+1))` (POSIX arithmetic)
- **Files modified:** scripts/verify-icons.sh
- **Verification:** Script runs all 13 checks without aborting
- **Committed in:** 31c548b (Task 1 commit)

**2. [Rule 2 - Missing Critical] Added !build/darwin/iconfile.icns to .gitignore**
- **Found during:** Task 1 git commit
- **Issue:** `build/darwin/*` gitignore rule prevented tracking iconfile.icns; force-add required
- **Fix:** Added `!build/darwin/iconfile.icns` exception so it's tracked as source asset
- **Files modified:** .gitignore
- **Verification:** `git status` shows iconfile.icns tracked in c569219
- **Committed in:** c569219

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both fixes required for correctness. No scope creep.

## Issues Encountered

- Wails auto-generated ICNS was 361KB (3-size: 1024/512/256); post-build injection of our 590KB 10-entry ICNS required (as anticipated in research/plan)

## Known Stubs

None - all assets are fully generated from the real branded logomark.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 37 (splash screen): `frontend/src/assets/agenthub-title-logo.png` is ready
- Phase 41 (tray icon): `build/appicon.png` 1024x1024 source is ready
- Visual verification (Task 3 checkpoint) still pending — user should inspect `build/bin/agenthub.app` icon in Finder and Dock

---
*Phase: 36-app-icons-branding-assets*
*Completed: 2026-03-31*
