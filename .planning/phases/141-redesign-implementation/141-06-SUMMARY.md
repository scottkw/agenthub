---
phase: 141-redesign-implementation
plan: "06"
subsystem: frontend-css
tags: [fonts, design-tokens, wcag, gap-closure]
dependency_graph:
  requires: []
  provides: [vendored-fonts, comp-dark-palette-tokens, font-radii-type-tokens]
  affects: [frontend/src/style.css, frontend/src/assets/fonts/]
tech_stack:
  added: [Plus Jakarta Sans woff2, JetBrains Mono woff2, CSS @font-face]
  patterns: [vendored fonts, CSS custom properties, WCAG AA contrast verification]
key_files:
  created:
    - frontend/src/assets/fonts/PlusJakartaSans-Regular.woff2
    - frontend/src/assets/fonts/PlusJakartaSans-Medium.woff2
    - frontend/src/assets/fonts/PlusJakartaSans-SemiBold.woff2
    - frontend/src/assets/fonts/JetBrainsMono-Regular.woff2
    - frontend/src/assets/fonts/JetBrainsMono-Medium.woff2
    - frontend/src/assets/fonts/LICENSES.md
  modified:
    - frontend/src/style.css
    - frontend/src/components/__tests__/style.hub.test.ts
    - frontend/src/components/__tests__/style.contrast.test.ts
decisions:
  - "Font paths use ./assets/fonts/ (not ../assets/fonts/) relative to style.css at src/ — Vite resolves CSS-relative paths from the source file location; correct path fingerprints all 5 woff2 into dist/assets/"
  - "Contrast test re-pointed to #9398a8 (comp muted) on comp backgrounds — minimum ratio 5.77:1 on #1c1e28, all pairs well above WCAG AA 4.5 threshold"
  - "--hub-text-dim updated from #565f89 to #6e7180 (comp dim tier); #565f89 still present as D-03 status-dot fence hex in xterm area (out of scope)"
  - "Light block gets font/radii/type-scale/semantic tokens only — color tokens preserved unchanged per plan decision"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-21"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 9
---

# Phase 141 Plan 06: Font Vendoring + Token Re-value Summary

**One-liner:** Vendored Plus Jakarta Sans (3 weights) and JetBrains Mono (2 weights) as committed woff2 binaries; re-valued all dark `--hub-*` color tokens to the comp palette (#14151b bg, comp grays, AA-verified) while preserving the locked blue accent and light palette unchanged.

## Tasks Completed

| Task | Commit | Description |
|------|--------|-------------|
| Task 1: Vendor fonts | 2782d150 | 5 woff2 binaries committed to frontend/src/assets/fonts/; LICENSES.md with OFL-1.1 attribution |
| Task 2: @font-face + token re-value | 608ee0e9 | 5 @font-face rules, font/radii/type-scale tokens in both blocks, dark palette re-valued to comp, chrome gate tests re-pointed |

## What Was Built

**Font vendoring (Task 1):**
- Plus Jakarta Sans weights 400/500/600 as latin-only woff2 binaries (11–12 KB each, verified WOFF2 format)
- JetBrains Mono weights 400/500 as latin-only woff2 binaries (~21 KB each, verified WOFF2 format)
- LICENSES.md recording OFL-1.1 attribution for both families with source URLs
- No CDN/Google Fonts URL added to index.html or style.css

**CSS changes (Task 2):**
- 5 `@font-face` rules added at top of style.css using `./assets/fonts/` relative URLs
- `--hub-font-ui` / `--hub-font-mono` tokens declared in both `:root` and `[data-ui-theme=light]`
- Radii tokens (`--hub-radius-sm/md/lg/pill`) declared in both blocks
- Type-scale tokens (`--hub-font-size-base/sm/heading`, `--hub-font-weight-emphasis: 600`) in both
- `--hub-teal: #43ddb2` and `--hub-orange: #e08a66` added in both blocks
- Dark `:root` color tokens re-valued to comp palette:
  - `--hub-bg: #14151b` (was TokyoNight `#1a1b26`)
  - `--hub-surface: #16181f` (was `#16161e`)
  - `--hub-surface-elevated: #1c1e28` (was `#1e2030`)
  - `--hub-border: #41454f` / `--hub-border-hover: #54586a`
  - `--hub-text-primary: #f4f5f8` / `--hub-text-secondary: #c7cad6` / `--hub-text-muted: #9398a8`
  - `--hub-text-placeholder: #54586a` / `--hub-text-dim: #6e7180`
  - `--hub-success: #4ade80` / `--hub-warning: #fbbf24`
- Locked unchanged: `--hub-accent: #7aa2f7`, `--hub-accent-hover: #89b4fa`, `--hub-destructive: #f7768e`
- Light palette color tokens PRESERVED (only new theme-independent tokens added to light block)

**Chrome gate tests:**
- `style.hub.test.ts`: updated assertion from `--hub-bg: #1a1b26` to `--hub-bg: #14151b` (comp dark surface); 85 tests pass
- `style.contrast.test.ts`: re-pointed 3 background/text pairings from old muted `#9aa5ce` on TokyoNight bgs to `#9398a8` on comp bgs (`#16181f`, `#14151b`, `#1c1e28`); min contrast 5.77:1 (well above 4.5 WCAG AA)

## Verification Gates

- `VENDOR_OK`: 5 woff2 files present, LICENSES.md present, no CDN URLs — PASSED
- `TOKENS_OK`: @font-face count >= 5, --hub-font-ui present, --hub-bg: #14151b, accent unchanged, no #7C8CFF, font-mono count == 2, radius-pill count == 2, light bg preserved — PASSED
- `pnpm test -- --run style.hub.test.ts style.contrast.test.ts`: 85/85 tests — PASSED
- `cd frontend && pnpm build`: exit 0, all 5 woff2 fingerprinted in dist/assets/ — PASSED

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed @font-face relative path from `../assets/fonts/` to `./assets/fonts/`**
- **Found during:** Task 2 build verification
- **Issue:** Plan specified `url("../assets/fonts/...")` but style.css is at `frontend/src/style.css` and fonts are at `frontend/src/assets/fonts/`. Going up one level from `src/` reaches `frontend/`, not `src/assets/`. The wrong path caused Vite to emit "didn't resolve at build time" warnings and skip fingerprinting the woff2 files.
- **Fix:** Changed all 5 @font-face src URLs to `url("./assets/fonts/<name>.woff2")` — correct CSS-relative path resolves Vite fingerprinting (all 5 woff2 appear in dist/assets/ output).
- **Files modified:** frontend/src/style.css (@font-face src paths)
- **Commit:** 608ee0e9

## Known Stubs

None. This plan changes token DEFINITIONS only — no surface selectors, no component rendering. The font and color tokens are wired but not yet applied to `.hub-*` surface selectors (that is 141-07's job).

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: font-CDN-ban-enforced | frontend/src/style.css | 5 @font-face rules reference same-origin relative paths only; no CDN URLs in index.html or style.css; woff2 binaries committed to repo |

## Self-Check: PASSED

- [x] `frontend/src/assets/fonts/PlusJakartaSans-Regular.woff2` — FOUND
- [x] `frontend/src/assets/fonts/JetBrainsMono-Medium.woff2` — FOUND
- [x] `frontend/src/assets/fonts/LICENSES.md` — FOUND
- [x] `frontend/src/style.css` — modified, @font-face rules present
- [x] Commit 2782d150 — FOUND (font vendoring)
- [x] Commit 608ee0e9 — FOUND (CSS token re-value)
