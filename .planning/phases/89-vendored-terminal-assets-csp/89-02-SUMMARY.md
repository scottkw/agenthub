---
phase: 89-vendored-terminal-assets-csp
plan: "02"
subsystem: web
tags: [html, csp, refactor, security]
dependency_graph:
  requires: []
  provides: [web/assets/terminal.js, web/assets/terminal.css, web/assets/dashboard.js, web/assets/dashboard.css, web/assets/join.js, web/assets/join.css]
  affects: [web/terminal.html, web/dashboard.html, web/join.html]
tech_stack:
  added: []
  patterns: [lift-and-shift extraction, external asset references, CSP-compatible HTML]
key_files:
  created:
    - web/assets/terminal.js
    - web/assets/terminal.css
    - web/assets/dashboard.js
    - web/assets/dashboard.css
    - web/assets/join.js
    - web/assets/join.css
  modified:
    - web/terminal.html
    - web/dashboard.html
    - web/join.html
decisions:
  - "Byte-for-byte content migration — no logic changes, no defer/async, no 'use strict' additions"
  - "Tag order in terminal.html preserved: xterm.js -> addon-fit.js -> terminal.js (global dependency order)"
  - "CDN xterm links replaced with /assets/xterm/* paths — actual file serving wired in Plan 04"
metrics:
  duration: "~8 minutes"
  completed: "2026-04-22"
  tasks_completed: 2
  files_changed: 9
---

# Phase 89 Plan 02: Extract Inline Blocks to External Assets Summary

Extracted every inline `<script>` and `<style>` block from `web/terminal.html`, `web/dashboard.html`, and `web/join.html` into six external companion files under `web/assets/`, and replaced three `cdn.jsdelivr.net` references in `terminal.html` with `/assets/xterm/*` paths. Pure lift-and-shift — zero logic changes, zero behavior changes.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Extract terminal.html inline blocks + swap CDN URLs | 2e79f50 | web/terminal.html, web/assets/terminal.js, web/assets/terminal.css |
| 2 | Extract dashboard.html and join.html inline blocks | 7d06a89 | web/dashboard.html, web/join.html, web/assets/dashboard.js, web/assets/dashboard.css, web/assets/join.js, web/assets/join.css |

## Before/After Byte Counts

| File | Before (bytes) | After (bytes) | Delta |
|------|---------------|---------------|-------|
| web/terminal.html | 9,309 | 674 | -8,635 |
| web/dashboard.html | 3,674 | 999 | -2,675 |
| web/join.html | 6,942 | 2,424 | -4,518 |

## New Companion File Line Counts

| File | Lines | Minimum Required | Status |
|------|-------|-----------------|--------|
| web/assets/terminal.js | 208 | 180 | PASS |
| web/assets/terminal.css | 48 | 40 | PASS |
| web/assets/dashboard.js | 31 | 25 | PASS |
| web/assets/dashboard.css | 43 | 35 | PASS |
| web/assets/join.js | 54 | 45 | PASS |
| web/assets/join.css | 93 | 80 | PASS |

## CSP Compatibility Confirmation

All three HTML files are now strict-CSP-compatible:

- **Zero inline `<script>` blocks** — `grep -cE '<script[^>]*>[^<]*[a-zA-Z]'` returns 0 for all three files
- **Zero inline `<style>` blocks** — `grep -cE '<style[^>]*>[^<]*[a-zA-Z]'` returns 0 for all three files
- **Zero `cdn.jsdelivr.net` references** — `grep -r "cdn\.jsdelivr"` finds nothing under `web/`
- All style references are `<link rel="stylesheet" href="/assets/...">` (same-origin)
- All script references are `<script src="/assets/...">` (same-origin)

A `default-src 'none'; script-src 'self'; style-src 'self'` CSP header (added in Plan 04) will not block any resource referenced by these pages.

## Tag Load Order (terminal.html)

The three xterm-related script tags load in the required order:
1. `/assets/xterm/xterm.js` (line 16) — defines `Terminal` global
2. `/assets/xterm/addon-fit.js` (line 17) — defines `FitAddon` global
3. `/assets/terminal.js` (line 18) — consumes both globals

## Content Preservation

- `web/assets/terminal.js` contains `Binary framing protocol constants` comment and `async function initTerminal` IIFE — preserved verbatim
- `web/assets/terminal.css` contains `#web-status-bar` selector and Tokyo Night palette (`#1a1b26`, `#16161e`, `#9ece6a`, `#f7768e`, `#e0af68`)
- `web/assets/dashboard.js` contains `formatCode` function with Base32 alphabet filter (A-Z, 2-7)
- `web/assets/join.js` contains `showState`, `formatCode`, and `wireCodeInput` functions plus query-string state dispatcher
- `web/assets/join.css` contains all 5-state UI styles (`.state`, `.state.active`, `.join-btn`, `.back-btn`, `.join-perm-badge--readonly`, `.join-perm-badge--readwrite`)

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- [x] web/assets/terminal.js exists — FOUND
- [x] web/assets/terminal.css exists — FOUND
- [x] web/assets/dashboard.js exists — FOUND
- [x] web/assets/dashboard.css exists — FOUND
- [x] web/assets/join.js exists — FOUND
- [x] web/assets/join.css exists — FOUND
- [x] Commit 2e79f50 exists — FOUND (Task 1)
- [x] Commit 7d06a89 exists — FOUND (Task 2)
- [x] Zero inline blocks across all three HTML files — VERIFIED
- [x] Zero CDN references — VERIFIED
