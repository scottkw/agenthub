# Phase 176: Platform & Hardening Bug Fixes - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-08
**Phase:** 176-platform-hardening-bug-fixes
**Areas discussed:** BUG-07 approach, BUG-06 CSP policy, BUG-05 verify strategy, BUG-05 menu scope

---

## BUG-07 (#127) — mini-preview approach

| Option | Description | Selected |
|--------|-------------|----------|
| Repro-first, clip target | Reproduce live BEFORE any change; the hypothesized CSS fix is already present. Find true cause if still broken; target = single clipped row. Close as already-fixed if not reproducible. | ✓ |
| Repro-first, column-wrap target | Same repro discipline, but target = terminal-style multi-row wrap at card column width. | |
| Just re-apply/harden CSS | Skip live repro, defensively harden CSS. (Risks certifying an unconfirmed cause.) | |

**User's choice:** Repro-first, clip target
**Notes:** Scout confirmed `.hub-card__preview-line` already has `white-space:nowrap; overflow:hidden; text-overflow:ellipsis` (`style.css:6020-6028`) — the issue's suspected fix. So repro-first is essential; the real cause is unknown or already resolved.

---

## BUG-06 (#123) — CSP policy on /app/

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse existing, verify vs bundle | Apply existing `cspHeaders` middleware to `/app/` as-is; verify against a prod Vite build; relax only for concrete violations. | ✓ |
| Separate SPA-tailored policy | Author a distinct `appCspHeaders` up front anticipating looser SPA needs. | |

**User's choice:** Reuse existing, verify vs bundle
**Notes:** Vite content-hashes assets (external `script-src 'self'`), so the existing static policy (`csp_mw.go:93-127`) is a strong fit. Verification requires a `wails build -tags wailsassets` prod build since `/app/` 503s under `wails dev`.

---

## BUG-05 (#124) — verification strategy

| Option | Description | Selected |
|--------|-------------|----------|
| M-NN item + accept reporter verify | Add manual M-NN to TESTING.md; treat reporter's from-source verification as sufficient to ship; run M-NN opportunistically on a Linux box. | ✓ |
| Hard-block ship on live Linux UAT | Do not ship until verified live on real Linux/Wayland ourselves. | |
| Fix only, no manual item | Apply fix, rely on reporter, add no manual item. | |

**User's choice:** M-NN item + accept reporter verify
**Notes:** Fix cannot be verified on the macOS dev box. Reporter verified both patches on Pop!_OS 24.04/COSMIC/Wayland from source. Preserves the repo's standing M-NN regression convention without blocking ship on hardware availability.

---

## BUG-05 (#124) — menu scope on Linux

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal darwin-guard | Wrap only AppMenu/EditMenu/WindowMenu in `if darwin`; keep existing File/Help; verify Linux copy/paste via WebKitGTK native shortcuts; add Edit items only if broken. | ✓ |
| Add explicit Linux Edit menu | Proactively build a plain Edit submenu with real callbacks for Linux. | |

**User's choice:** Minimal darwin-guard
**Notes:** File/Help submenus already exist in `appMenu()`. On Linux the clipboard path is WebKitGTK-native (Ctrl+C/V in xterm.js), not the app menu, so the removed EditMenu roles should not break copy/paste — verify rather than assume.

## Claude's Discretion

- `runtime` stdlib import aliasing in `main.go` (name collision with the Wails `runtime` alias).
- Whether the `/app/` CSP presence is asserted via a new Go test or folded into an existing `internal/webserver` test.
- The precise dev-browser harness setup for BUG-07 live repro.

## Deferred Ideas

- Terminal-style column-wrap for the mini-preview (rejected in favor of single clipped row).
- Broader SPA-bundle hardening (SRI, nonce-based CSP) beyond the header itself.
- Tailscale admin-API device-share/ACL automation (already milestone-out-of-scope).
