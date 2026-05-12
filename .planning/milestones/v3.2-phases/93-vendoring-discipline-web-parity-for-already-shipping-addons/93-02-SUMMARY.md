---
phase: 93
plan: 02
subsystem: web-vendor
tags: [vendoring, web, xterm, csp, embed]
requires:
  - frontend/node_modules/@xterm/addon-webgl/lib/addon-webgl.js
  - frontend/node_modules/@xterm/addon-unicode11/lib/addon-unicode11.js
  - frontend/node_modules/@xterm/addon-clipboard/lib/addon-clipboard.js
  - frontend/pnpm-lock.yaml
provides:
  - web/vendor/xterm/addons/addon-webgl.js (UMD; window.WebglAddon.WebglAddon)
  - web/vendor/xterm/addons/addon-unicode11.js (UMD; window.Unicode11Addon.Unicode11Addon)
  - web/vendor/xterm/addons/addon-clipboard.js (UMD; window.ClipboardAddon.ClipboardAddon)
  - web/vendor/xterm/VERSION (5-line manifest; aligned with pnpm-lock)
  - web/embed.go (//go:embed directive extended to addons/)
  - web/terminal.html (3 new <script> tags between addon-fit.js and terminal.js)
affects:
  - Plan 93-01 vendor_drift_test (now exercises 5 packages instead of 2)
  - Plan 93-04 (will consume the verified UMD globals at runtime)
tech-stack:
  added: []
  patterns:
    - Verbatim-copy vendoring (cmp -s parity vs node_modules)
    - Explicit //go:embed paths (no globs — per 93-RESEARCH Pitfall #4)
    - External-script-only HTML (no inline blocks — preserves CSP script-src 'self')
key-files:
  created:
    - web/vendor/xterm/addons/addon-webgl.js
    - web/vendor/xterm/addons/addon-unicode11.js
    - web/vendor/xterm/addons/addon-clipboard.js
  modified:
    - web/vendor/xterm/VERSION
    - web/embed.go
    - web/terminal.html
decisions:
  - Vendored UMD `.js` (not `.mjs` ESM) — `<script src=...>` without `type="module"` is the simplest CSP-compatible loader and matches the existing addon-fit.js pattern.
  - Verified UMD global names by reading the wrapper in each bundle's first 800 bytes. Plan 93-04 must use `new window.<Namespace>.<ClassName>()` form — the exports object is the namespace, the class lives at `<Namespace>.<ClassName>`.
  - clipboard addon's UMD wrapper uses `self` (not `globalThis`) as root; safe in a window context (`self === window`), but worth noting if the page ever loads in a Worker.
  - Explicit per-file `//go:embed` paths (not a `vendor/xterm/addons/*.js` glob) — keeps the embed surface auditable and matches existing project convention.
metrics:
  duration: ~9 min (estimate; very small mechanical plan)
  completed: 2026-05-04
  tasks_completed: 2
  files_changed: 6 (3 created, 3 modified)
  commits: 2
---

# Phase 93 Plan 02: Vendor Three Already-Shipping Web Addons Summary

Vendored `@xterm/addon-webgl@0.19.0`, `@xterm/addon-unicode11@0.9.0`, and `@xterm/addon-clipboard@0.2.0` into `web/vendor/xterm/addons/` and wired them into the Go embed.FS + the web terminal page so the v3.1 strict CSP (`script-src 'self'`) is honored when Plan 93-04 turns them on at runtime.

## What Shipped

- **3 vendored UMD bundles** (verbatim copies, byte-identical to source per `cmp -s`):
  - `web/vendor/xterm/addons/addon-webgl.js` — 247,535 bytes — global `window.WebglAddon` (class at `window.WebglAddon.WebglAddon`)
  - `web/vendor/xterm/addons/addon-unicode11.js` — 52,489 bytes — global `window.Unicode11Addon` (class at `window.Unicode11Addon.Unicode11Addon`)
  - `web/vendor/xterm/addons/addon-clipboard.js` — 6,384 bytes — global `window.ClipboardAddon` (class at `window.ClipboardAddon.ClipboardAddon`); UMD root is `self`, harmless in window context
- **`web/vendor/xterm/VERSION`** — extended from 2 lines to 5 lines, listing each `@xterm/*` package at the exact version resolved in `frontend/pnpm-lock.yaml`. Plan 93-01's generalized `vendor_drift_test` now passes for all five entries.
- **`web/embed.go`** — added a second `//go:embed` line covering `vendor/xterm/addons/addon-{webgl,unicode11,clipboard}.js`. `go build ./web` succeeds.
- **`web/terminal.html`** — inserted three external `<script src="/assets/xterm/addons/addon-*.js"></script>` tags between the existing `addon-fit.js` line and the existing `terminal.js` line. No inline scripts; load order guarantees `terminal.js` sees all three globals when it runs.

## UMD Global Names (for Plan 93-04 consumption)

Verified by reading the UMD wrapper in each bundle:

| Vendored file               | UMD root         | Namespace export        | Constructor path                         |
| --------------------------- | ---------------- | ----------------------- | ---------------------------------------- |
| addon-webgl.js              | `globalThis`     | `WebglAddon`            | `new window.WebglAddon.WebglAddon()`     |
| addon-unicode11.js          | `globalThis`     | `Unicode11Addon`        | `new window.Unicode11Addon.Unicode11Addon()` |
| addon-clipboard.js          | `self` (=window) | `ClipboardAddon`        | `new window.ClipboardAddon.ClipboardAddon()` |

Pattern in each wrapper (paraphrased): `module.exports = factory()`, where `factory` returns `{ <Name>Addon: <ClassDefinition> }`. Plan 93-04 should NOT do `new window.WebglAddon()` — that returns the namespace object, not a constructor.

## Verification

| Gate                                               | Result          |
| -------------------------------------------------- | --------------- |
| `cmp -s` byte equality vs `frontend/node_modules`  | OK (3/3)        |
| `wc -l web/vendor/xterm/VERSION` returns 5         | OK              |
| Per-package grep counts (`grep -c` returns 1)      | OK (3 manifest entries, 3 embed lines, 3 script tags) |
| `awk` order check (webgl AFTER addon-fit, BEFORE terminal.js) | OK    |
| `go build ./web`                                   | exit 0          |
| `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock` | PASS |
| `go test ./internal/webserver/... -run TestSecurity_NoInlineScriptOrStyleInHTML` | PASS |
| `go test ./internal/webserver/... -run TestSecurity_NoCDNReferencesInWebAssets` | PASS |

Plan 93-01's generalized vendor-drift gate is now green with all 5 entries — confirms the parity contract between `web/vendor/xterm/VERSION` and `frontend/pnpm-lock.yaml` for the entire xterm vendoring surface.

## Commits

| Task | Description                                                                                  | Hash      |
| ---- | -------------------------------------------------------------------------------------------- | --------- |
| 1    | Vendor three xterm addon UMD bundles into `web/vendor/xterm/addons/`                         | `8c153ae` |
| 2    | Wire vendored addons into VERSION manifest, embed.go, and terminal.html                      | `c3d2ad8` |

## Deviations from Plan

None — plan executed exactly as written. Versions in pnpm-lock matched the planner's expected values (0.19.0 / 0.9.0 / 0.2.0), all source files were present at the documented `lib/` paths, and all acceptance grep checks plus the three test gates passed on the first run.

## Threat Flags

None new. The plan's `<threat_model>` is fully addressed:
- T-93-WEB-02 (inline-script injection): no inline `<script>` blocks introduced; `TestSecurity_NoInlineScriptOrStyleInHTML` PASS.
- T-93-CLIP-01 (OSC 52 clipboard): vendoring only ships the bundle; instantiation gate (read-only viewers do not get clipboard write) is owned by Plan 93-04.
- T-93-WEB-01 (supply-chain drift): `cmp -s` byte parity asserted in Task 1 acceptance; `TestXtermVendorVersionsMatchPnpmLock` PASS.

No new security-relevant surfaces were introduced beyond the three vendored static `.js` files served from the binary under the existing CSP — these are byte-for-byte the same bundles already shipped to desktop users via the Wails frontend.

## Known Stubs

None. This plan is mechanical vendoring — no UI components, no data flows. Plan 93-04 will consume the vendored bundles at runtime to instantiate addons.

## Self-Check: PASSED

- `web/vendor/xterm/addons/addon-webgl.js` — FOUND
- `web/vendor/xterm/addons/addon-unicode11.js` — FOUND
- `web/vendor/xterm/addons/addon-clipboard.js` — FOUND
- `web/vendor/xterm/VERSION` (5 lines) — FOUND
- `web/embed.go` (extended directive) — FOUND
- `web/terminal.html` (3 new script tags) — FOUND
- Commit `8c153ae` — FOUND in `git log --oneline`
- Commit `c3d2ad8` — FOUND in `git log --oneline`
