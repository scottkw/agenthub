# Project Research Summary

**Project:** AgentHub v1.7 — Daemon UX & Branding
**Domain:** Desktop daemon manager (Wails/Go/React) — system tray, remote session indicators, app icons, splash screen
**Researched:** 2026-03-31
**Confidence:** HIGH (codebase inspected directly; Wails platform constraints verified against open GitHub issues; platform icon requirements from Apple/Microsoft official docs)

## Executive Summary

AgentHub v1.7 adds the UX layer that transforms a functional daemon manager into a polished desktop product: a persistent tray icon with session management, remote session status indicators for web and CLI clients, proper platform icons, and a branded splash screen. The existing architecture (Wails v2 + Go daemon + React frontend) is already the right foundation, and all four v1.7 areas can be implemented without introducing new architectural paradigms. Three of four feature areas require no new Go dependencies at all — only the system tray icon (macOS change) and icon generation tooling require new library work.

The single hardest constraint for this milestone is well-documented and already encountered in this project: `fyne.io/systray` and every mainstream systray library conflict with Wails' own `AppDelegate` at the macOS linker level, producing an unresolvable duplicate symbol. The project already resolved this by implementing a custom cgo `NSStatusItem` wrapper for macOS. This pattern must be preserved and extended; it must not be replaced with a systray library. The Linux and Windows implementations remain as no-op stubs for v1.7, which is the right call — Linux GNOME has no guaranteed tray support regardless of library used. Wails v2 is also single-window only (confirmed in issue #1480), which means the "daemon mini management window" from the milestone description is a React panel inside the existing window, not a second OS window.

The build order is cleanly dependency-driven: icons and branding assets have no code dependencies and unblock all visual work; splash screen needs the logo in the frontend asset tree; remote status bars (web and CLI) are self-contained; the DaemonPanel React component must exist before the tray "Daemon Manager" menu item can be wired up to it. No new Wails bindings are needed for v1.7 — all required operations (`ListSessions`, `GetSessionStatus`, `KillSession`, `RenameSession`) are already bound in `app.go`.

## Key Findings

### Recommended Stack

v1.7 adds minimal new dependencies. The system tray is already implemented as a custom cgo NSStatusBar on macOS (confirmed working in codebase). The icon generation pipeline requires `sips`+`iconutil` (macOS-native, zero-install) or `icnsify` (pure Go, cross-platform CI) for ICNS generation; ImageMagick `magick convert` for ICO on Windows CI; and `golang.org/x/image/draw` (already in project as indirect dep via Wails/Tailscale) for PNG resizing. The splash screen and web status bar are frontend-only changes with no new npm packages.

**Core technologies:**
- Custom cgo NSStatusItem (darwin only): system tray — the only safe pattern given Wails AppDelegate conflict; confirmed in project KEY DECISION
- `systray-on-wails` (`github.com/ra1phdd/systray-on-wails`): Linux/Windows tray (optional v1.7) — uses `Register()` not `Run()`, non-blocking, coexists with Wails event loop
- `sips` + `iconutil` (macOS system tools): ICNS generation — zero-install on macOS build hosts; use `icnsify` for CI/Linux
- `magick convert` (ImageMagick): ICO generation — already available in CI; multi-size via `icon:auto-resize=256,48,32,16`
- `golang.org/x/image/draw`: PNG resizing — already in project via indirect deps

**What NOT to add:**
- `fyne.io/systray`, `github.com/energye/systray`, `github.com/getlantern/systray`, `github.com/cratonica/trayhost` — all cause duplicate `_OBJC_CLASS_$_AppDelegate` linker error with Wails on macOS (project already hit this)
- Wails v3 upgrade — alpha, unquantified risk; stay on v2.10.2 for this milestone

### Expected Features

**Must have (table stakes for v1.7 launch):**
- App icon set (ICNS, ICO, platform PNGs) — required for professional distribution; existing `appicon.png` is a programmatic placeholder
- Tray icon in system tray (not dock) with right-click Open/Quit — daemon apps must live in the tray; absence makes the app feel like a toy
- GUI hides (not quits) on window close — tray lifecycle requirement; current Wails default terminates the process on close
- macOS dock icon hidden (`LSUIElement` in Info.plist) — daemon should not appear in dock or Cmd+Tab
- Daemon state reflected in tray icon (running vs. error; two icons minimum)
- Web terminal session status bar — session name, agent, connection state, host name
- CLI attach session banner — printed to stderr before raw mode; confirms remote session and detach key

**Should have (add after core tray is stable):**
- Splash screen using title logo — masks 200-800ms WebKit init latency; adds brand quality signal
- Tray tooltip with session count — depends on tray event loop being stable
- Tray menu lists active session names (capped at 5-10) — depends on dynamic menu update support

**Defer (v2+):**
- Mini management window from tray — Wails v2 is single-window; a real second window requires Wails v3 (alpha) or native Cocoa cgo; complexity far outweighs the value at this stage
- Session count badge overlay on tray icon — platform support inconsistent; low priority
- Full-color dark/light adaptive tray icon template — polish detail; ship monochrome first
- Linux/Windows real tray implementation — Linux GNOME tray is best-effort regardless of library; stubs acceptable for v1.7

### Architecture Approach

v1.7 modifies 8 existing files and adds 6 new files. No new daemon IPC routes are needed — the daemon API already exposes all required operations. The web terminal status bar uses a polling REST endpoint (`GET /api/sessions/{id}/status`, 3-second interval) rather than a new relay frame type; this avoids touching the hot binary I/O path across 5 files/packages. The DaemonPanel is a React modal inside the existing Wails single window, not a second OS window; tray-to-panel wiring uses the established `runtime.EventsEmit` → `EventsOn` pattern already used for `session:status`, `tailscale:health`, and `daemon:error`. The splash screen is a React state gate on `initComplete` in `App.tsx`, with `defer setInitComplete(true)` required on all code paths including error branches.

**Major components:**
1. `tray.go` (darwin cgo, MODIFIED) — add "Daemon Manager" NSMenuItem; emit `daemon:show-manager` Wails event via `onTrayDaemonMgr()` cgo callback
2. `tray_linux.go` / `tray_windows.go` (MODIFIED or keep stubs) — optionally implement `systray-on-wails`; no-op stubs are acceptable for v1.7
3. `SplashScreen.tsx` (NEW) — full-screen branding overlay; dismisses when `initComplete = true` in App.tsx
4. `DaemonPanel.tsx` (NEW) — session list with status/kill/rename using only existing Wails bindings (no new bindings needed)
5. `web/terminal.html` (MODIFIED) — status bar div + 3-second polling fetch + flex CSS layout fix
6. `cmd_attach.go` (MODIFIED) — ANSI banner to stderr before `term.MakeRaw` entry
7. `internal/webserver/server.go` (MODIFIED) — add `/api/sessions/{id}/status` route (3 lines of Go)
8. `build.sh` + `assets/appicon.png` + `build/darwin/AppIcon.icns` + `build/darwin/Info.plist` — icon generation pipeline and LSUIElement

### Critical Pitfalls

1. **fyne.io/systray and all mainstream systray libraries produce duplicate AppDelegate symbol on macOS** — Never add any of these as a Go dependency. The project already hit this exact failure. Keep the custom cgo NSStatusItem wrapper for macOS exclusively; never call `systray.Run()` from the Wails process on macOS.

2. **LSUIElement must be in Info.plist before app launch, not set at runtime** — Setting `NSApplicationActivationPolicyAccessory` at runtime causes a dock icon flash (~10ms) on every startup. Add `<key>LSUIElement</key><true/>` to `build/darwin/Info.plist` ONLY (not `Info.dev.plist` — this would make the app invisible during debugging). Validate by inspecting the built `.app/Contents/Info.plist` after `wails build`.

3. **Wails v2 is single-window only — no second OS window for Daemon Manager** — Any attempt to open a second OS window (second `wails.Run()`, second webview, native NSWindow) will either crash, produce a blank non-functional window, or require a full Cocoa UI in cgo. The "daemon mini management window" is a React panel inside the existing window shown via `runtime.WindowShow()` + Wails events.

4. **Splash screen white flash before React renders** — A React-rendered splash shows the OS white WebView background for ~100-300ms before React paints. Use `StartHidden: true` in Wails options + static HTML splash in `index.html` (no JS required) + `runtime.WindowShow()` from `OnDomReady`. The static HTML splash renders before React hydration.

5. **Terminal CSS height breaks when status bar is injected above xterm.js** — FitAddon `proposeDimensions()` measures container `clientHeight`. Status bar must be a flex sibling with fixed pixel height; terminal container must use `flex: 1 1 0; min-height: 0; overflow: hidden`. Verify `proposeDimensions()` row count is unchanged after adding the bar — this is the v1.6 layout regression in a new form.

6. **macOS ICNS missing @2x layers** — `iconutil` requires all 10 named files (5 sizes × 2 densities). Missing `@2x` variants look correct on non-Retina displays but blurry on Retina. Verify with `sips -g all AppIcon.icns` — must show 10 entries including 1024x1024.

7. **initComplete not set on daemon error path** — If the daemon fails during `init()`, the splash must still dismiss so the error banner can render. Use `defer setInitComplete(true)` at the top of the `init()` function; never guard it behind only the happy path.

## Implications for Roadmap

Based on the dependency graph and risk profile from combined research, 6 phases are suggested. Phases 1-4 are independent and can proceed in parallel. Phases 5-6 are sequentially coupled.

### Phase 1: App Icons & Branding Assets
**Rationale:** Zero code dependencies; unblocks every other visual feature. The existing `appicon.png` is a programmatic placeholder — all icon-dependent work validated against a placeholder is misleading. Complete this before any tray or splash work.
**Delivers:** Branded 1024x1024 `appicon.png`, `AppIcon.icns` (all 10 sizes for macOS Retina), `icon.ico` (4+ sizes for Windows), `tray_icon.png` (18x18 monochrome template), updated `build.sh` icon generation steps, title logo copied into `frontend/src/assets/`.
**Addresses:** FEATURES table-stakes "App has proper platform icons"
**Avoids:** Pitfall 6 (missing @2x ICNS layers), Pitfall 8 (single-size ICO produces blurry taskbar icon on Windows)

### Phase 2: Splash Screen
**Rationale:** Depends only on the title logo being in `frontend/src/assets/` (Phase 1 output). Pure frontend work; no Go changes. Closes the first-impression UX gap visible on every app launch.
**Delivers:** `SplashScreen.tsx`, `initComplete` state in `App.tsx` with `defer setInitComplete(true)`, `StartHidden: true` in Wails options, static HTML splash fallback in `index.html`, `OnDomReady` → `runtime.WindowShow()` wiring.
**Addresses:** FEATURES "Splash screen using title logo"
**Avoids:** Pitfall 4 (white flash before splash), Pitfall 7 (initComplete not set on error path)

### Phase 3: Remote Status Bar — Web Terminal
**Rationale:** Fully self-contained; touches only `internal/webserver/server.go` and `web/terminal.html`. No dependency on tray or splash work. Immediately improves UX for all remote web session users.
**Delivers:** `GET /api/sessions/{id}/status` JSON endpoint, flex CSS layout in `terminal.html`, 3-second `setInterval` polling, status bar showing session name/agent/connection state/host machine name.
**Addresses:** FEATURES "Remote session status bar in web terminal", "Web session status bar shows machine/host name"
**Avoids:** Pitfall 3 (adding new relay frame type to hot I/O path — use REST polling instead), Pitfall 5 (terminal CSS height break from status bar injection)

### Phase 4: Remote Status Banner — CLI Attach
**Rationale:** Standalone; modifies only `cmd_attach.go`. Lowest-complexity item in the milestone. Print one ANSI `fmt.Fprintf` to stderr before `term.MakeRaw` entry. No protocol changes, no new bindings.
**Delivers:** ANSI status banner: `[AgentHub] Connected to "<name>" (<agent>) on <hostname>` + Ctrl-\ detach hint; `[AgentHub] Detached.` message on exit.
**Addresses:** FEATURES "Remote session indicator in CLI attach"
**Avoids:** No specific pitfall; confirmed clean pattern from Architecture research

### Phase 5: DaemonPanel React Component
**Rationale:** Must precede tray wiring (Phase 6). All four required Wails bindings already exist at known locations in `app.go`. Can be validated independently by temporarily setting `showDaemonPanel = true` in `App.tsx` without any tray work.
**Delivers:** `DaemonPanel.tsx` (session list, per-session status, kill and rename controls), `App.tsx` `EventsOn("daemon:show-manager")` listener, `showDaemonPanel` state toggle.
**Addresses:** FEATURES "Daemon mini management window from tray" (scoped as an in-window React panel, not a second OS window)
**Avoids:** Pitfall 3/Architecture Anti-Pattern 1 (second OS window attempt in Wails v2)

### Phase 6: System Tray — macOS + LSUIElement
**Rationale:** Depends on DaemonPanel (Phase 5). Most platform-constrained phase; placed last after all other features are validated in isolation. Requires `wails build` production testing — cannot be fully validated in `wails dev` due to Info.plist and template image behavior differences.
**Delivers:** "Daemon Manager" NSMenuItem added to `tray.go`, `onTrayDaemonMgr()` cgo callback emitting `daemon:show-manager` event, `LSUIElement` in `build/darwin/Info.plist`, window hide-on-close behavior (override default Wails quit), daemon state icon switching (running/error), tray tooltip with session count.
**Addresses:** FEATURES "Tray icon", "Right-click tray menu", "GUI hides on window close", "macOS dock icon hidden", "Tray icon reflects daemon state"
**Avoids:** Pitfall 1 (systray library AppDelegate conflict), Pitfall 2 (LSUIElement at runtime = dock flash), Pitfall 6 (daemon process owns tray icon), Pitfall 10 (Linux tray silent failure — keep stubs for v1.7)

### Phase Ordering Rationale

- Phases 1-4 have no mutual dependencies and can proceed in parallel with multiple workstreams, or sequentially in the order listed.
- Phase 5 (DaemonPanel) must precede Phase 6 (tray "Daemon Manager" item) because the tray callback emits `daemon:show-manager` and the React handler must exist to be testable.
- Phase 6 (tray) is last because it is the most constrained by platform behavior, requires production build testing (not dev mode), and depends on the panel it opens (Phase 5) being complete.
- Phase 1 (icons) is first because validating visual quality against a placeholder icon is misleading and creates rework.

### Research Flags

Phases needing targeted review before or during planning:

- **Phase 6 (System Tray):** Review the existing `tray.go` cgo implementation before writing new code — the KEY DECISION in PROJECT.md documents which approaches failed. Determine at planning time whether Linux/Windows tray gets real `systray-on-wails` implementation or remains as stubs; this scope decision directly affects the Phase 6 work estimate and should be resolved before sprint planning.
- **Phase 1 (Icons):** Requires a design decision before coding can start. The existing `docs/agenthub-title-logo.png` is a horizontal wordmark (805x208px); icon sizes below 64px require a standalone square logomark, not a wordmark. Determine whether to extract the mark from the existing logo (image editing) or commission new artwork. This is a blocker for Phase 1.

Phases with standard patterns (no additional research needed):

- **Phase 2 (Splash):** Architecture research provides the exact implementation pattern including the `defer setInitComplete(true)` requirement and `StartHidden: true` + `OnDomReady` wiring.
- **Phase 3 (Web Status Bar):** Architecture research specifies the exact route signature, CSS structure, and `setInterval` polling pattern. This is a solved problem.
- **Phase 4 (CLI Banner):** One `fmt.Fprintf` before `term.MakeRaw`. Fully specified.
- **Phase 5 (DaemonPanel):** Architecture research lists the exact Wails binding names and their line numbers in `app.go`. No new research needed.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Critical tray constraint confirmed from project KEY DECISION (direct project failure, not inference) + Wails GitHub issues. Icon tools are pure Go or macOS-native and well-documented. |
| Features | MEDIUM-HIGH | Feature set derived from clear product context. Platform-specific behavior (Linux GNOME tray) has known limitations documented by the GNOME project itself. |
| Architecture | HIGH | Based on direct codebase inspection of 15+ files. Wails single-window constraint confirmed from GitHub issue #1480. All data flows drawn from actual source code, not docs. |
| Pitfalls | HIGH | Most critical pitfalls derive from failures already experienced in this project (systray linker error documented in KEY DECISION; Wails single-window constraint; terminal layout break from v1.6). Not theoretical. |

**Overall confidence:** HIGH

### Gaps to Address

- **Square logomark asset:** The project has a horizontal wordmark (`docs/agenthub-title-logo.png`, 805x208) but no standalone square icon at 1024x1024 for ICNS/ICO generation. Phase 1 cannot complete until a square logomark exists. Resolve before Phase 1 sprint: extract the mark from the existing logo, redraw it, or letterbox with padding.
- **Linux/Windows tray scope decision:** Architecture recommends `systray-on-wails` for Linux/Windows but flags it as optional. Whether Phase 6 includes real Linux/Windows tray or keeps stubs is a scope call that affects the work estimate by roughly 1-2 days. Decide before Phase 6 planning.
- **LSUIElement UX tradeoff:** `LSUIElement = YES` removes the app from Cmd+Tab (app switcher) entirely — not just from the Dock. This is the correct behavior for a pure background daemon tray app but is a significant behavioral change from current behavior. Confirm this is acceptable product behavior before Phase 6 implementation.
- **Production build test cycle:** Phase 6 (tray) and Phase 1 (ICNS) cannot be fully validated in `wails dev` — they require `wails build` + running the `.app` bundle. Plan for this slower test cycle in the sprint estimate for those phases.

## Sources

### Primary (HIGH confidence)
- AgentHub codebase (direct inspection): `tray.go`, `tray_linux.go`, `tray_windows.go`, `app.go`, `main.go`, `cmd_attach.go`, `internal/daemon/api.go`, `internal/relay/protocol.go`, `internal/webserver/server.go`, `web/terminal.html`, `frontend/src/App.tsx`, `build/darwin/Info.plist`, `build/gen_icon.go`, `go.mod`, `wails.json`
- `.planning/PROJECT.md` KEY DECISION — "fyne.io/systray conflicts with Wails AppDelegate (duplicate symbol)" — confirms pitfall was already encountered in this project
- [Wails issue #1480](https://github.com/wailsapp/wails/issues/1480) — single-window constraint confirmed (driver of v3 rewrite)
- [Wails issue #3700](https://github.com/wailsapp/wails/issues/3700) — dock icon hiding; LSUIElement must be in Info.plist
- [Wails discussion #4514](https://github.com/wailsapp/wails/discussions/4514) — systray subprocess/IPC pattern; NSStatusBar cgo as macOS-only alternative confirmed by maintainer
- [Apple LSUIElement documentation](https://developer.apple.com/documentation/bundleresources/information-property-list/lsuielement) — agent app behavior; dock and Cmd+Tab exclusion

### Secondary (MEDIUM confidence)
- [fyne.io/systray pkg.go.dev](https://pkg.go.dev/fyne.io/systray) v1.12.0 — Linux DBus requirement, CGO requirement, API
- [systray-on-wails](https://pkg.go.dev/github.com/ra1phdd/systray-on-wails) — cross-platform tray for Wails v2, `Register()` API, published Nov 2024
- [jackmordaunt/icns GitHub](https://github.com/JackMordaunt/icns) — pure Go, `icnsify` CLI, cross-platform, v2.2.7
- [macOS ICNS required sizes and naming convention](https://gist.github.com/jamieweavis/b4c394607641e1280d447deed5fc85fc) — all 10 files required for full Retina support
- [Windows .ico required sizes (Microsoft)](https://learn.microsoft.com/en-us/windows/apps/design/iconography/app-icon-construction) — 16, 32, 48, 256 minimum

### Tertiary (LOW confidence, noted for context)
- [GNOME AppIndicator GNOME 48 compatibility issue](https://bbs.archlinux.org/viewtopic.php?id=304357) — Linux tray support degrading further on modern GNOME; reinforces best-effort position
- [systray-on-wails adoption](https://pkg.go.dev/github.com/ra1phdd/systray-on-wails) — pre-release v0.0.0 with limited production evidence; acceptable risk for Linux/Windows stub replacement, not for macOS

---
*Research completed: 2026-03-31*
*Ready for roadmap: yes*
