# Phase 176: Platform & Hardening Bug Fixes - Context

**Gathered:** 2026-07-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Close the three remaining cross-cutting platform/hardening bugs before v4.2 ships — the last phase of the milestone:

- **BUG-05 (#124)** — Linux desktop GUI is dead-on-arrival: (1) segfault on launch from macOS role-based menu items (`AppMenu`/`EditMenu`/`WindowMenu`) whose `nil` SubMenu is dereferenced by Wails' GTK backend, and (2) WebKit2GTK DMABUF GPU-renderer freeze on first interaction under Wayland. Both fixable in `main.go`.
- **BUG-06 (#123)** — the `/app/` route (web-share guest SPA surface) serves **no** Content-Security-Policy header; add CSP defense-in-depth. Carried over from the Phase 168-02 #123 follow-up (deliberately scoped out then).
- **BUG-07 (#127)** — Hub session-card mini-preview stacks long output lines one/two characters per row instead of rendering each line as one clipped horizontal row.

**Fixed scope; no new capabilities.** All three are pre-triaged bugs with root causes documented in the issues. This phase is independent of Phases 174/175.
</domain>

<decisions>
## Implementation Decisions

### BUG-07 (#127) — mini-preview line rendering
- **D-01 — Reproduce live BEFORE any code change.** The CSS the issue hypothesizes as the fix (`white-space: nowrap; overflow: hidden; text-overflow: ellipsis`) is **already present** on `.hub-card__preview-line` (`frontend/src/style.css:6020-6028`). So the issue's suspected root cause is already patched. The phase must first reproduce the char-per-row stacking live on the Hub (long-line session, e.g. `sh -c 'i=0; while true; do i=$((i+1)); printf "%s  live tick #%d\n" "$(date +%T)" "$i"; sleep 1; done'`) via a dev-browser component harness. Do NOT re-apply CSS that already exists.
- **D-02 — If reproducible, find the TRUE root cause.** With `nowrap` already on the line div, the real cause is elsewhere: the per-character `<span>` runs in `MiniPreview.tsx` (each styled run is its own inline span), a global/reset rule forcing spans to `display:block`, a `flex-direction:column` leaking onto the line/spans, or the styled-tail data itself being one-char-per-run. Confirm the mechanism before fixing. This directly honors the standing lesson that a green/"looks-fixed" state can certify a broken feature — drive it live. See [[feedback_tests_encoding_same_wrong_assumption]] and [[feedback_verify_ui_against_design_comp]].
- **D-03 — Target render = single clipped row.** Each output line renders on ONE horizontal row, clipped/truncated (clip or ellipsis, matching the existing CSS intent) to the card width — a normal terminal tail. NOT terminal-style column-wrap to multiple rows.
- **D-04 — If NOT reproducible, close as already-fixed with evidence.** If the current build already renders long lines correctly (a prior hub-card layout effort may have landed the nowrap CSS), record the live evidence (screenshot / computed styles) and close #127 as already-fixed — no code change. Colorblind-safe rendering is unaffected (this is layout, not color) but verify at source per [[user_colorblind]] if any color spans are touched.

### BUG-06 (#123) — CSP header on /app/
- **D-05 — Reuse the existing `cspHeaders` middleware on `/app/` as-is.** Apply the existing static middleware (`internal/webserver/csp_mw.go:93-127`) to the `/app/` route registration (`server.go:1034`), the same wrapper already used on `/dashboard`, `/join`, `/sessions/{id}`. Do NOT author a separate SPA-tailored policy up front. Vite content-hashes every asset (external files under `script-src 'self'`), so the existing policy is a strong candidate.
- **D-06 — Verify against a production Vite build.** Build with `wails build -tags "webkit2_41,wailsassets"` (SPA embedded — the daemon has no embedded SPA under `wails dev`/plain `go build`, so `/app/` returns 503 there; a prod build is required to exercise it). Load `/app/` in a browser and check the console for CSP violations across: inline scripts, `wasm-unsafe-eval` (xterm/wasm), `connect-src` for the SSE `/api/plugin-config/stream` + WS relay, `font-src`, and `img-src data:`. React inline styles are `style=` attributes, allowed by the existing `style-src 'unsafe-inline'`.
- **D-07 — Relax the policy ONLY for a concrete violation the bundle actually hits.** No speculative loosening. The existing policy is `default-src 'none'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' wss://<base>; img-src 'self' data:; font-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'`.

### BUG-05 (#124) — Linux GUI segfault + DMABUF freeze
- **D-08 — Minimal darwin-guard for the menu.** Wrap only the three role menus (`menu.AppMenu()`, `menu.EditMenu()`, `menu.WindowMenu()`) in `if runtime.GOOS == "darwin"` in `appMenu()` (`main.go:98-116`). The File and Help submenus already exist with real callbacks and stay on all platforms. Import the stdlib `runtime` package (currently only the Wails `runtime` alias is imported — avoid the name clash by aliasing/qualifying appropriately). This matches the reporter's verified fix.
- **D-09 — Verify Linux copy/paste works via WebKitGTK native shortcuts.** Dropping `EditMenu` on Linux removes the app-menu Cmd/Ctrl+C/V/X/Z roles, but the clipboard path on Linux is WebKitGTK's native Ctrl+C/V in the xterm.js terminal, not the app menu. Add explicit Linux Edit items ONLY if verification shows copy/paste is actually broken. Do not proactively build a Linux Edit submenu.
- **D-10 — DMABUF renderer fix.** Set `WEBKIT_DISABLE_DMABUF_RENDERER=1` at the top of `runGUI()` (`main.go:65`, before `wails.Run`), only on Linux and only if the user hasn't already set it (`os.LookupEnv` guard — `os` is already imported). This resolves the WebKit2GTK GPU-renderer hang on Wayland compositors.

### BUG-05 (#124) — verification strategy
- **D-11 — Add a manual M-NN checklist item + accept reporter's from-source verification for ship.** The Linux GUI fix cannot be verified on this macOS dev box. Add a manual M-NN item to TESTING.md Section 5 (Linux/Wayland: GUI launches without segfault, File/Help menu bar works, hamburger/UI interactions do not freeze). Treat the reporter's from-source verification (Pop!_OS 24.04 / COSMIC / Wayland, both patches, `wails build -tags "webkit2_41,wailsassets"`) as sufficient evidence to ship v4.2. Run the live M-NN opportunistically if a Linux box becomes available. Do NOT hard-block ship on live Linux UAT.

### Regression-test convention (standing rule)
- **D-12 — Update TESTING.md per the repo convention.** BUG-05's Linux GUI check → new manual M-NN item (Section 5). BUG-06 → a Go/automated test if the CSP header on `/app/` is unit-assertable (assert the header is present on an `/app/` response), plus traceability rows (Section 4) for any new test files, `.go`/`.ts`/`.tsx`/`.sh` paths only. Run `bash tests/check-traceability-paths.sh` before committing. BUG-07's fix (if code changes) → vitest/component coverage if the true cause is CSS/render-layer. See CLAUDE.md "Regression Test Convention" + TESTING.md Section 6.

### Claude's Discretion
- Exact `runtime` import aliasing in `main.go` (stdlib `runtime` vs the Wails `runtime` alias name collision) — planner/executor's call.
- Whether BUG-06's CSP presence is asserted via a new Go test vs folded into an existing `internal/webserver` test file.
- The precise dev-browser harness setup for BUG-07 live repro (component-mount pattern per [[reference_browser_uat_help_harness.md]]).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Issue tracker (authoritative root-cause + prescribed fixes)
- GitHub `scottkw/agenthub#124` — Linux GUI segfault + DMABUF; contains the verified `appMenu()` darwin-guard rewrite and the `WEBKIT_DISABLE_DMABUF_RENDERER=1` `runGUI()` patch. Reporter verified both from source on Pop!_OS 24.04/COSMIC/Wayland.
- GitHub `scottkw/agenthub#123` — `/app/` CSP header follow-up; documents why 168-02 (threat model T-168-03) scoped it out and what to verify against Vite hashes.
- GitHub `scottkw/agenthub#127` — Hub mini-preview char-per-row; lists `MiniPreview.tsx` + `.hub-card__preview-line` CSS as the suspected area (now known already-patched).

### Source files (current state confirmed via scout)
- `main.go` §`appMenu()` (98-116) — role menus appended unconditionally; File/Help submenus already exist. §`runGUI()` (65-93) — no env setup, `wails.Run` at line 69. `os` imported, stdlib `runtime` NOT imported.
- `internal/webserver/csp_mw.go` (93-127) — the static `cspHeaders` middleware to reuse on `/app/`.
- `internal/webserver/server.go` — cspHeaders applied at lines 869/874/979; `/app/` route registered at 1034-1067 with NO CSP; Vite-hash caching note at 1024-1032.
- `frontend/src/components/Hub/MiniPreview.tsx` (40-64) — per-styled-run `<span>` rendering.
- `frontend/src/style.css` (6012-6038) — `.hub-card__preview` / `.hub-card__preview-line` (already has `nowrap`/`overflow:hidden`/`text-overflow:ellipsis`) + empty/loading `display:flex` override.

### Project conventions
- `TESTING.md` §4 (traceability), §5 (manual M-NN checklist), §6 (standing convention); `tests/check-traceability-paths.sh`.
- `/Users/ken/dev/agenthub/CLAUDE.md` — Regression Test Convention (standing rule).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cspHeaders` middleware (`csp_mw.go:93`) — reuse verbatim on `/app/`; already battle-tested on 3 routes.
- Existing File/Help submenus in `appMenu()` — keep; only the role menus need guarding.
- dev-browser component-harness pattern (`[[reference_browser_uat_help_harness.md]]`) — for BUG-07 live repro without the Wails app shell.

### Established Patterns
- `runtime.GOOS == "..."` platform guards — none exist in `main.go` yet; BUG-05 introduces the first (stdlib `runtime` import needed, mind the Wails `runtime` alias collision).
- Vite content-hashed assets are the cache key (`server.go:1024-1032`) — external `script-src 'self'` friendly; the reason the existing CSP is a strong fit for `/app/`.
- Production-build gate: `/app/` only serves the SPA under `wails build -tags "webkit2_41,wailsassets"`; returns 503 under `wails dev`/plain `go build`. BUG-06 verification requires a prod build.

### Integration Points
- `main.go` `appMenu()` + `runGUI()` — the two BUG-05 fix sites (both single-file).
- `server.go:1034` `/app/` route registration — wrap with `ws.cspHeaders(...)`.
- `MiniPreview.tsx` + `.hub-card__preview*` CSS — only if BUG-07 reproduces and the true cause lands there.

</code_context>

<specifics>
## Specific Ideas

- BUG-07 repro command (from #127): `sh -c 'i=0; while true; do i=$((i+1)); printf "%s  live tick #%d\n" "$(date +%T)" "$i"; sleep 1; done'` — long unbroken lines to trigger the stacking.
- BUG-06 build/verify recipe: `wails build -tags "webkit2_41,wailsassets"` then load `/app/` and read the browser console for CSP violations.
- BUG-05 reporter environment for the M-NN item: Pop!_OS 24.04, COSMIC, Wayland, x86_64, WebKit2GTK 4.1, `.deb` install.

</specifics>

<deferred>
## Deferred Ideas

- Terminal-style column-wrap for the mini-preview (multi-row wrapped tail) — rejected in favor of single clipped row (D-03); could be a future polish idea but not this phase.
- Any broader SPA-bundle hardening beyond the CSP header (e.g. SRI, nonce-based CSP) — out of scope; D-05 reuses the existing static policy.
- Automating device-share / ACL edits via Tailscale admin API — already milestone-out-of-scope (PROJECT.md), unrelated here.

None beyond the above — discussion stayed within phase scope.

</deferred>

---

*Phase: 176-platform-hardening-bug-fixes*
*Context gathered: 2026-07-08*
