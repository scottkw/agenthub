# Project Research Summary

**Project:** AgentHub v1.1 Polish & Build
**Domain:** Cross-platform desktop terminal multiplexer (Go/Wails + React/xterm.js) for AI coding CLIs
**Researched:** 2026-03-19
**Confidence:** HIGH

## Executive Summary

AgentHub v1.1 is a polish-and-build milestone on top of a fully functional v1.0. The app is a Wails v2 desktop app (Go backend + React/xterm.js frontend) that multiplexes AI coding CLI sessions with optional web serving for remote access. All v1.1 features are incremental improvements to the existing architecture — zero new dependencies are required. Every stack decision has been verified against local binaries and official documentation, and every architecture integration point has been confirmed through direct codebase inspection of all affected files.

The recommended approach is to work inside the existing architecture without introducing new abstractions. The 8 v1.1 features fall into three natural groups: pure CSS/layout fixes (terminal fill, toolbar sizing, status bar, settings modal), React component additions with no backend changes (per-tab font size, new-session modal frontend), and backend-touching changes (new-session modal Go binding, tab rename web propagation, build script). This ordering matters — layout fixes must come first because the status bar and font size features depend on a stable, correctly filling terminal container. The `min-height: 0` flex trap is the single most likely cause of failed integration tests if the build sequence is violated.

The primary risks are concentrated in two areas: the macOS signing pipeline (notarytool exit-0 trap, `--deep` flag producing invalid signatures that silently pass local verification) and per-tab font size key event handling (xterm.js consuming SHIFT+= before the app handler fires, and the mandatory `requestAnimationFrame(() => fitAddon.fit())` call after every font size change). Both risks are well-understood with specific mitigations documented in PITFALLS.md. The build script is the only feature with HIGH integration risk; all UI features are LOW to MEDIUM risk.

## Key Findings

### Recommended Stack

The v1.1 stack is the v1.0 stack with zero additions. No new npm packages and no new Go modules are required. All new capabilities (native OS folder picker, per-tab font size, local build/sign automation) are built with APIs already present in the installed dependencies: `runtime.OpenDirectoryDialog` from `github.com/wailsapp/wails/v2` (already in `go.mod`), `terminal.options.fontSize` and `fitAddon.fit()` from `@xterm/xterm` and `@xterm/addon-fit` (already installed), and `wails build` CLI with macOS system tools (`codesign`, `xcrun notarytool`, `ditto`, `xcrun stapler`) available on any macOS dev machine with Xcode.

**Core technologies:**
- **Go 1.26.1 + Wails v2.10.2:** Desktop app host — all new backend methods go on the `App` struct in `app.go`; `runtime.OpenDirectoryDialog` already available for folder browser; no version change
- **React 19 + TypeScript 5.9:** Frontend — two new components (`StatusBar`, `NewSessionModal`) replace inline JSX; `App.tsx` stays as thin state hub; no new libraries
- **@xterm/xterm 6 + @xterm/addon-fit:** Terminal rendering — `terminal.options.fontSize` is a live-mutable property; `fitAddon.fit()` is mandatory after every font size change; `@xterm/addon-fit ^0.11.0` must match xterm 6.x
- **Bash + Xcode CLI tools:** Build script — `wails build -platform <target>` + bottom-up `codesign` (no `--deep`) + `xcrun notarytool` with explicit JSON status parsing

### Expected Features

**Must have (table stakes) — all P1:**
- `build.sh` — one-command local cross-platform build with macOS signing; env-gated so unsigned local builds work without certs
- Terminal full-fill fix — CSS `min-height: 0` at every flex level in the column chain; prerequisite for status bar and font size features
- Per-tab status bar — fixed-height strip replacing the `web-serving-bar` overlay; always rendered; shows web controls + session status indicator
- Larger toolbar buttons — 36–44px hit targets; CSS only
- Settings modal declutter — tabbed "CLI Paths" / "Web Serving" sections; single "Close" footer
- Tab renaming propagation to web dashboard — `GET /api/sessions` must return `[]SessionSummary{id, name, cli_type}` not `[]string`
- New-session modal with agent picker + folder browser — replaces bare CLI picker overlay; remembers last folder via `localStorage`
- Per-tab SHIFT+/SHIFT- font size — 8–32px bounds; `attachCustomKeyEventHandler` to intercept before PTY delivery; `requestAnimationFrame(() => fitAddon.fit())` after each change

**Should have (stretch goal for v1.1):**
- Web dashboard visual refresh — card layout, status color dots, CLI badge; coordinated with `GET /api/sessions` shape change (P2)

**Defer (v1.2+):**
- Tab color coding per CLI type
- Status heuristics for non-Claude CLIs
- Font size persistence via backend (localStorage is sufficient)

### Architecture Approach

All 8 features integrate cleanly into the existing Wails IPC architecture (App struct methods → TypeScript bindings → React components) without restructuring. Two new React components (`StatusBar`, `NewSessionModal`) replace inline JSX blocks in `App.tsx`. Two new Go methods (`BrowseFolder`, updated `CreateSession` with `workDir`) extend `app.go`. One internal package change propagates `workDir` through `CreateRequest` to `cmd.Dir` at PTY spawn. One webserver change (`handleListSessions` returning `[]SessionSummary` via a `NameFunc` callback) avoids a circular import while enabling web dashboard name propagation.

**Major components and their v1.1 changes:**
1. **`App.tsx`** — adds `tabFontSizes` state, `showNewSessionModal` state, window-level `keydown` handler for font size, replaces inline CLI picker JSX with `<NewSessionModal>`, wires `StatusBar` props
2. **`StatusBar.tsx` (NEW)** — fixed-height per-tab bar with web controls + session status indicator; always rendered; `flex-shrink: 0`
3. **`NewSessionModal.tsx` (NEW)** — agent picker + folder browser + session name input; calls `BrowseFolder()` Go binding; persists last folder in `localStorage`
4. **`TerminalPanel.tsx`** — adds `fontSize` prop; `useEffect([fontSize])` mutates `term.options.fontSize` + calls `fitAddon.fit()`
5. **`app.go`** — adds `BrowseFolder()` method, updates `CreateSession` signature with `workDir string`, adds `getTabName()` helper, wires `NameFunc` into webserver config
6. **`internal/webserver/server.go`** — `handleListSessions` returns `[]SessionSummary`; `Config` gains `NameFunc` callback to avoid circular import
7. **`internal/pty/backend.go` + `native.go`** — `CreateRequest.WorkDir` field added; `cmd.Dir = req.WorkDir` at PTY spawn
8. **`build.sh` (NEW)** — local wrapper for `wails build`; explicit bottom-up codesign; `notarytool` JSON output parsed for `"Accepted"` status

### Critical Pitfalls

1. **Font size change without fit() leaves terminal garbled** — always call `requestAnimationFrame(() => fitAddonRef.current?.fit())` after setting `terminal.options.fontSize`; without the RAF defer, `fit()` reads stale glyph metrics and computes wrong cols/rows; PTY receives incorrect terminal dimensions (xterm.js issue #4886)

2. **SHIFT+= consumed by xterm.js before app handler fires** — use `term.attachCustomKeyEventHandler()` which runs before PTY delivery; return `false` to suppress PTY delivery; do NOT use `onKey` or `window.addEventListener` for this specific handler; register all custom shortcuts in a single function since `attachCustomKeyEventHandler` overwrites any previous registration

3. **macOS `codesign --deep` produces invalid signatures** — sign bottom-up explicitly (binary first, then app bundle) without `--deep`; use `ditto -c -k --keepParent` (not `zip -r`) for notarization archive; parse `notarytool` JSON output for `"status": "Accepted"` since exit code is always 0 even on rejection

4. **Wails `OpenDirectoryDialog` panics on Windows with deleted DefaultDirectory** — validate path with `os.Stat()` before passing; fall back to `""` if path no longer exists (Wails issues #1052, #1381)

5. **Flex `min-height: 0` trap** — every flex column container in the ancestor chain (`.app`, `.terminal-container`, `.terminal-wrapper`, `TerminalPanel` outer div) must have `min-height: 0`; adding the status bar changes the flex hierarchy and can cause terminals to stop filling remaining height

6. **Stale Wails TypeScript bindings** — after adding any new Go method to `App`, run `wails dev` to regenerate `wailsjs/go/main/App.ts` before writing frontend code that calls it; stale bindings compile cleanly but fail silently at runtime

## Implications for Roadmap

Based on combined research, the recommended 7-phase build sequence follows the dependency graph established in ARCHITECTURE.md and explicitly addresses pitfall prevention ordering.

### Phase 1: Layout Baseline (Terminal Fill + Toolbar Sizing)

**Rationale:** All subsequent UI features depend on the terminal correctly filling its container. The flex `min-height: 0` trap (Pitfall 5) must be resolved before the status bar is added, or every layout test will be unreliable. This is pure CSS — fastest to verify, zero regression risk to existing functionality.
**Delivers:** Terminals fill all available vertical space on all tabs; toolbar buttons meet 36–44px hit target
**Addresses:** Terminal full-fill fix, larger toolbar buttons (both P1 table stakes)
**Avoids:** Flex min-height trap; establishes stable layout baseline for Phases 2–4

### Phase 2: Per-Tab Status Bar

**Rationale:** Depends on Phase 1 (stable container height). Extracting the existing inline `web-serving-bar` JSX into a proper `StatusBar` component is prerequisite for the Phase 3 font size feature, since font size changes trigger `fitAddon.fit()` and the status bar height must be stable first. StatusBar is always rendered — terminal height is now deterministic.
**Delivers:** Permanent single-line status strip below tab bar; web controls always accessible; session status indicator always visible
**Addresses:** Per-tab status bar (P1 table stake)
**Avoids:** Flex min-height trap at each new flex level in the StatusBar chain; StatusBar must use `flex-shrink: 0`

### Phase 3: Settings Modal Overhaul

**Rationale:** Isolated frontend refactor with no dependencies on Phases 1–2 and no backend changes. Best done early before adding more settings (new-session modal, font size). Reduces cognitive load on SettingsPanel before touching it again in later phases.
**Delivers:** Tabbed settings modal (CLI Paths / Web Serving); single "Close" footer; no API surface changes
**Addresses:** Settings modal declutter (P1 table stake)
**Avoids:** No significant pitfalls; pure layout work; all existing Wails bindings unchanged

### Phase 4: Per-Tab Font Size (SHIFT+/SHIFT-)

**Rationale:** Depends on Phase 1 (stable container) and Phase 2 (stable status bar height). The font size change → fit() call cycle requires a correct container to measure accurately. This phase has two non-trivial interlocking pitfalls (Pitfalls 1 and 2) that require explicit manual verification.
**Delivers:** Per-tab font size adjustment via keyboard; bounds 8–32px; persisted in `localStorage` per sessionId; terminal cols/rows correctly updated after each change
**Addresses:** Per-tab SHIFT+/SHIFT- font size (P1 table stake)
**Avoids:** Pitfall 1 (always `requestAnimationFrame(() => fitAddon.fit())` after fontSize); Pitfall 2 (use `attachCustomKeyEventHandler`, return `false`, single handler)

### Phase 5: New-Session Modal + Folder Browser

**Rationale:** Largest scope — requires both Go backend changes (`BrowseFolder` method, `CreateSession` workDir param, PTY struct and spawn changes) and a new React component. Placed after frontend patterns are stable. Wails binding regeneration is a required step when Go signatures change (Pitfall 6).
**Delivers:** Full new-session modal with agent picker, folder browser (native OS dialog), session naming; remembers last folder across restarts
**Addresses:** New-session modal with agent picker + folder browser (P1 differentiator)
**Avoids:** Pitfall 4 (validate DefaultDirectory with `os.Stat` before `OpenDirectoryDialog`); Pitfall 6 (regenerate Wails bindings via `wails dev` after adding `BrowseFolder`)

### Phase 6: Tab Rename Web Propagation + Dashboard Refresh

**Rationale:** Requires webserver API shape change (`[]string` → `[]SessionSummary`) and `NameFunc` callback wiring. Dashboard visual refresh is coordinated with this change — they touch the same file (`dashboard.html`) and the same API response shape. Must be deployed together to avoid version mismatch between API and dashboard renderer.
**Delivers:** Web dashboard reflects desktop tab renames; card-based dashboard layout with status color dots and CLI badge
**Addresses:** Tab renaming propagation to web dashboard (P1); web dashboard visual refresh (P2 stretch goal)
**Avoids:** Pitfall 5 (verify web dashboard shows renamed session name — not just React state); backward-compatible response shape since `dashboard.html` already handles `s.name` for both string and object array

### Phase 7: Build Script

**Rationale:** Written last against a stable, fully-featured binary. Build script complexity (macOS signing, notarization) is independent of all code changes and is best validated after the binary is final. macOS signing pitfalls (Pitfalls 3 and the notarytool exit-0 trap) are highest-risk and require end-to-end testing with real Apple Developer credentials.
**Delivers:** `build.sh` with per-platform and all-platform modes; macOS signing + notarization + staple; env-gated for unsigned local builds; verified with `spctl --assess`
**Addresses:** `build.sh` (P1 table stake)
**Avoids:** Invalid signatures from `--deep` (sign bottom-up explicitly); silent CI shipping of unnotarized builds (parse `notarytool` JSON, fail if `status != "Accepted"`)

### Phase Ordering Rationale

- **Layout first, then features:** Phases 1–2 establish the CSS foundation that all interactive features require. Skipping this order risks false-positive testing — font size may look correct visually but terminal cols/rows are wrong.
- **Frontend before backend:** Phases 3–4 are pure frontend and can be verified immediately in `wails dev`. Phase 5 introduces Go changes requiring binary rebuild and binding regeneration — fewer moving parts until patterns are proven.
- **Coordinated backend changes sequentially:** Phases 5 and 6 each require Go changes; placing them sequentially avoids merge conflicts in `app.go`.
- **Build script last:** It is a developer artifact, not a product feature. It validates against the final binary and is the one feature with HIGH integration risk that benefits from a stable compilation target.

### Research Flags

Phases that need explicit manual verification during execution (implementation guidance is complete — no additional research needed):
- **Phase 4 (Font size):** Two interlocking pitfalls require explicit manual verification: hold SHIFT+= for 2 seconds and verify no `+` characters appear in the shell prompt; check `term.cols` value via browser console after resize to confirm fit() computed correct dimensions
- **Phase 5 (New-session modal):** Cross-platform `OpenDirectoryDialog` behavior on Windows with a deleted "last folder" path; Wails binding regeneration step is easy to miss — verify `wailsjs/go/main/App.ts` lists `BrowseFolder` before writing the React caller
- **Phase 7 (Build script):** Full sign + notarize + staple + `spctl --assess` pipeline requires Apple Developer credentials and real-world test; the notarytool exit-0 trap cannot be detected without running actual notarization

Phases with standard, well-documented patterns (execution is straightforward):
- **Phase 1:** Pure CSS flexbox; min-height:0 pattern is well-established
- **Phase 2:** Extracting existing inline JSX into a component; no new data flows
- **Phase 3:** Pure layout refactor; all existing Wails bindings unchanged

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All flags verified against local `wails v2.10.2` binary and official docs; zero new dependencies confirmed by exhaustive search through each feature's requirements |
| Features | HIGH | v1.0 codebase read directly; all feature gaps identified by code inspection, not inference; dependency graph verified against actual source files |
| Architecture | HIGH | Integration points confirmed by direct inspection of all affected files (`app.go`, `App.tsx`, `TerminalPanel.tsx`, `internal/webserver/server.go`, etc.); data flows traced end-to-end |
| Pitfalls | HIGH | Critical pitfalls sourced from official xterm.js GitHub issues (#4886, #4841), Wails issues (#1052, #1381), and Apple notarization post-mortems with documented recovery steps; all verified against the actual codebase structure |

**Overall confidence:** HIGH

### Gaps to Address

- **OpenDirectoryDialog on Linux (GTK file chooser):** The `os.Stat` guard prevents the Windows panic, but dialog UX and behavior on Linux (GTK file chooser) has not been explicitly verified. Test on Linux runner during Phase 5 verification.

- **Font size `localStorage` persistence vs. session ID stability:** Research recommends `localStorage` keyed by `sessionId`. If session IDs are regenerated on each app launch, font size preferences will be lost on restart. Verify session ID lifecycle (persistent UUID vs. ephemeral) before implementing persistence in Phase 4.

- **Status bar rendering on Windows WebView2:** Flex rendering differences between WebKit (macOS) and WebView2 (Windows) can produce subtle layout bugs. PITFALLS.md explicitly flags this. Test on all three platforms during Phase 2 verification before proceeding to Phase 3.

## Sources

### Primary (HIGH confidence)
- `wails build --help` (local binary, v2.10.2) — all build flags verified against installed binary
- [Wails runtime OpenDirectoryDialog](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/runtime) — official Go pkg.go.dev documentation
- [Wails OpenDialogOptions struct](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/dialog) — all struct fields verified
- [xterm.js Terminal.options API](https://xtermjs.org/docs/api/terminal/classes/terminal/) — `fontSize` read-write, `attachCustomKeyEventHandler` documented
- [xterm.js Issue #4886](https://github.com/xtermjs/xterm.js/issues/4886) — fontSize without fit() causes abnormal display
- [xterm.js Issue #3346](https://github.com/xtermjs/xterm.js/issues/3346) — FitAddon height zero when flex container lacks min-height: 0
- [Wails Issue #1052](https://github.com/wailsapp/wails/issues/1052) + [#1381](https://github.com/wailsapp/wails/issues/1381) — OpenDirectoryDialog panic on Windows with invalid DefaultDirectory
- Direct codebase inspection: `app.go`, `App.tsx`, `TerminalPanel.tsx`, `TabBar.tsx`, `SettingsPanel.tsx`, `style.css`, `relayClient.ts`, `internal/relay/hub.go`, `internal/webserver/server.go`, `web/dashboard.html`, `.github/workflows/build.yml`, `wails.json`

### Secondary (MEDIUM confidence)
- [Wails code signing guide](https://wails.io/docs/guides/signing/) — official but terse; supplemented by community post-mortems
- [macOS code signing gist (rsms)](https://gist.github.com/rsms/929c9c2fec231f0cf843a1a746a416f5) — bottom-up signing rationale, `--deep` pitfall documented
- [Federico Terzi notarytool CI post](https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/) — exit-0 trap documented with fix pattern
- [ddev signing_tools](https://github.com/ddev/signing_tools) — reference implementation for CI signing
- [Wails cross-platform build guide](https://wails.io/docs/guides/crossplatform-build/) — macOS CGo cross-compile constraint confirmed
- [CSS-Tricks flex fill height](https://css-tricks.com/boxes-fill-height-dont-squish/) — min-height: 0 pattern

---
*Research completed: 2026-03-19*
*Ready for roadmap: yes*
