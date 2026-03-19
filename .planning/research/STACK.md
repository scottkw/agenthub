# Technology Stack

**Project:** AgentHub
**Milestone:** v1.1 Polish & Build
**Researched:** 2026-03-19
**Confidence:** HIGH (all core claims verified against official docs or local binary)

---

## Scope of This Document

This is a **delta document** for v1.1. It covers only what is NEW or CHANGED relative to the validated v1.0 stack. The existing stack (Go/Wails v2, React 19, xterm.js 6, go-pty, coder/websocket, skip2/go-qrcode) is unchanged and already installed.

The new features requiring stack decisions:
1. Build script (`build.sh`) — local cross-platform compilation with macOS signing
2. Settings modal / UI redesign — styling only, no new libraries
3. New-session modal with folder browser — native directory picker via Wails runtime
4. Per-tab status bar — layout/CSS only, no new libraries
5. Tab renaming — already wired in v1.0 (`RenameSession` Wails binding exists)
6. Per-tab SHIFT+/SHIFT- font size adjustment — xterm.js `terminal.options.fontSize` + `fitAddon.fit()`
7. Terminal full-fill fix — CSS flexbox correction + `fitAddon.fit()` timing

---

## Recommended Stack

### Core Technologies (Unchanged)

Already installed. No version changes required.

| Technology | Current Version | Status |
|------------|----------------|--------|
| Go | 1.26.1 | No change |
| Wails v2 | v2.10.2 | No change |
| React | ^19.2.4 | No change |
| TypeScript | ^5.9.3 | No change |
| @xterm/xterm | ^6.0.0 | No change |
| @xterm/addon-fit | ^0.11.0 | No change (critical for font size fix) |

### New Capabilities and Their Stack Implications

#### 1. Build Script — `build.sh`

**What it does:** Local developer script that wraps `wails build` for per-platform and all-platform compilation. Handles the macOS signing + notarization sequence already proven in CI.

**Stack decision:** Pure bash script using `wails build` CLI directly. No new Go or npm dependencies.

`wails build` flags used (verified against local `wails v2.10.2`):

| Flag | Purpose |
|------|---------|
| `-platform darwin/universal` | macOS universal binary (arm64 + amd64) |
| `-platform linux/amd64` | Linux 64-bit |
| `-platform windows/amd64` | Windows 64-bit |
| `-nsis` | Windows NSIS installer (requires NSIS installed) |
| `-clean` | Delete bin dir before build (prevents stale artifact confusion) |
| `-ldflags "-s -w"` | Strip debug symbols for smaller binary |

**macOS signing sequence** (bash, runs on macOS only):

```bash
# Already proven in .github/workflows/build.yml — same commands for local use
codesign --force -s "$DEVELOPER_ID" --options runtime \
  --entitlements build/entitlements.plist \
  build/bin/agenthub.app

# Notarization (requires Apple ID credentials in environment)
ditto -c -k --keepParent build/bin/agenthub.app notarization.zip
xcrun notarytool submit notarization.zip \
  --apple-id "$APPLE_ID" \
  --team-id "$APPLE_TEAM_ID" \
  --password "$APPLE_APP_PASSWORD" \
  --wait
xcrun stapler staple build/bin/agenthub.app
```

**No new Go dependencies.** No new npm dependencies. The signing tools (`codesign`, `xcrun notarytool`, `ditto`, `xcrun stapler`) are Xcode CLI tools already present on any macOS dev machine with Xcode installed.

**Linux cross-compile note:** Linux builds require native Linux runners (`gtk`, `webkit2gtk`). The build script should detect the OS and skip Linux builds when run on macOS, or use Docker if cross-compile is needed. Simplest: build.sh builds current platform by default, with `--all` flag documented as "requires running per platform."

#### 2. Folder Browser (New-Session Modal)

**What it does:** Opens a native OS folder picker dialog when user clicks "Browse" in the new-session modal. Returns the selected directory path to the frontend.

**Stack decision:** Wails v2 runtime `OpenDirectoryDialog` — already available in the Wails runtime, no new dependency.

Go backend binding (verified against `pkg.go.dev/github.com/wailsapp/wails/v2/pkg/runtime`):

```go
import "github.com/wailsapp/wails/v2/pkg/runtime"

func (a *App) BrowseDirectory(defaultDir string) (string, error) {
    return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        DefaultDirectory: defaultDir,
        Title:            "Choose working directory",
    })
}
```

`OpenDialogOptions` struct fields (verified via `pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/dialog`):

| Field | Type | Use |
|-------|------|-----|
| `DefaultDirectory` | `string` | Pre-open to this path (use last remembered folder) |
| `Title` | `string` | Dialog window title |
| `AllowFiles` | `bool` | Set `false` — directories only |
| `AllowDirectories` | `bool` | Set `true` |
| `ShowHiddenFiles` | `bool` | Optional, default `false` |
| `CanCreateDirectories` | `bool` | Allow user to create new dirs in dialog |

**Wails JS runtime note:** Dialog is NOT callable from the JS runtime. Must be exposed as a Go-bound method via `//go:bind` pattern (standard Wails App struct method). The frontend calls `BrowseDirectory(lastDir)` via the generated Wails binding.

**Last-folder persistence:** Store in-memory in the Go App struct (for session lifetime) OR persist to a config file using `os.UserConfigDir()` + JSON marshal. No external config library needed — encoding/json stdlib.

**No new dependencies.** `runtime.OpenDirectoryDialog` is in `github.com/wailsapp/wails/v2` which is already in `go.mod`.

#### 3. Per-Tab Font Size (SHIFT+/SHIFT-)

**What it does:** Each tab has an independent font size. SHIFT+= (alias SHIFT+Plus) increases font size, SHIFT+Minus decreases. The terminal reflows after the change.

**Stack decision:** xterm.js `terminal.options.fontSize` is a writable property at runtime (verified against official xterm.js API docs). After changing `fontSize`, call `fitAddon.fit()` to reflow cols/rows for the new character dimensions.

Implementation pattern (no new library needed):

```typescript
// In TerminalPanel component — expose a changeFontSize(delta: number) method
// called by the parent via a ref or a prop

function changeFontSize(delta: number) {
  if (!termRef.current || !fitAddonRef.current) return
  const current = termRef.current.options.fontSize ?? 14
  const next = Math.max(8, Math.min(32, current + delta))
  termRef.current.options.fontSize = next
  // Must call fit() after font size change so cols/rows recalculate
  // Defer one frame so the browser has painted the new char dimensions
  requestAnimationFrame(() => {
    fitAddonRef.current?.fit()
  })
}
```

**Keyboard event handling:** Listen for `keydown` on the active terminal's container div (or document), filter for `shiftKey + Key=` and `shiftKey + Minus`. Use the browser's `KeyboardEvent.key` values: `"+"` (shift+=) and `"_"` (shift+-). Note that xterm.js captures all key events inside the terminal viewport — use a `keydown` listener on the outer wrapper div BEFORE xterm.js receives the event, or use `term.attachCustomKeyEventHandler()` which runs before xterm processes the key.

The recommended approach is `term.attachCustomKeyEventHandler()`:

```typescript
term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
  if (event.type !== 'keydown') return false
  if (event.shiftKey && event.key === '+') {
    changeFontSize(+1)
    return false  // prevent xterm from also processing the key
  }
  if (event.shiftKey && event.key === '_') {
    changeFontSize(-1)
    return false
  }
  return true  // let xterm handle everything else
})
```

**Per-tab state:** Store `fontSize` per tab in the App component state (`Record<string, number>`) and pass it down to `TerminalPanel` as a prop. Or let each `TerminalPanel` own its font size in local state and expose a ref-based imperative handle. Both work; local state in `TerminalPanel` is simpler since font size doesn't need to be shared.

**No new dependencies.**

#### 4. Terminal Full-Fill Fix

**What it does:** Fix terminals not expanding to fill the full available container space.

**Root cause** (diagnosed from existing code): The `terminal-container` uses `flex: 1` and `terminal-wrapper` uses `height: 100%`, which requires all ancestors to have explicit heights in a flex column. The current layout chain is:

```
.app (flex column, height: 100%)
  .tab-bar (flex-shrink: 0, height: 36px)
  .terminal-container (flex: 1, overflow: hidden)
    .terminal-wrapper (display: flex, flex-direction: column, width: 100%, height: 100%)
      .web-serving-bar (flex-shrink: 0) -- optional
      TerminalPanel div (flex: 1, width: 100%, minHeight: 0)
```

The `xterm.js` viewport div inside the TerminalPanel container has `position: relative` and the xterm canvas is absolutely positioned. The fit addon measures `containerElement.clientWidth / clientHeight` to calculate cols/rows. If the container's height is 0 at the time `fit()` runs (e.g., before the display:none→flex transition completes), the terminal renders with 0 rows.

**Fix:** The `requestAnimationFrame(() => fitAddonRef.current?.fit())` pattern already exists in the code and is the correct approach. The remaining issue is likely that the `terminal-wrapper` with `height: 100%` inside a `flex: 1` parent requires `min-height: 0` on the flex container to prevent overflow. Add `min-height: 0` to `.terminal-container`.

**CSS fix (no new library):**

```css
.terminal-container {
  flex: 1;
  min-height: 0;  /* CRITICAL: prevents flex overflow in column layout */
  overflow: hidden;
  position: relative;
}

.terminal-wrapper {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-height: 0;  /* Also needed here for nested flex */
}
```

**No new dependencies.**

#### 5. Per-Tab Status Bar

**What it does:** Replace the `web-serving-bar` floating overlay with a proper status bar docked below the tab bar (or above the terminal). Shows web status, URL, and controls per active tab.

**Stack decision:** CSS layout only. Move the existing `web-serving-bar` content from `App.tsx` into a dedicated `StatusBar` React component. Render it between the tab bar and the terminal. No new library needed.

```css
.status-bar {
  flex-shrink: 0;
  height: 28px;
  display: flex;
  align-items: center;
  padding: 0 8px;
  gap: 8px;
  background-color: #16161e;
  border-bottom: 1px solid #292e42;
  font-size: 12px;
}
```

**No new dependencies.**

#### 6. UI/UX Overhaul (Settings Modal, Toolbar, New-Session Modal)

**What it does:** Visual redesign of existing components. Larger toolbar buttons, cleaner settings modal layout, new-session modal replacing the CLI picker dropdown.

**Stack decision:** CSS-only changes. No new UI component library needed. The existing custom CSS approach (plain CSS classes in `style.css`) is sufficient for this scale of UI. The existing modal pattern (`settings-overlay` + `settings-panel`) works well and just needs visual polish.

**No new dependencies.**

---

## Supporting Libraries — Summary for v1.1

| Library | Status | Why |
|---------|--------|-----|
| `wails/v2/pkg/runtime` | Already in `go.mod` | `OpenDirectoryDialog` for folder browser |
| `@xterm/addon-fit` | Already installed | `fit()` after font size changes |
| bash / xcrun / codesign | macOS system tools | Build script signing — no install needed |

**Zero new npm packages.** **Zero new Go modules.**

---

## Alternatives Considered

| Feature | Recommended | Alternative | Why Not |
|---------|-------------|-------------|---------|
| Folder browser | Wails `OpenDirectoryDialog` | Custom JS file picker (html `<input type="file" webkitdirectory>`) | Wails webview may not support `webkitdirectory` reliably across platforms. Native dialog is the correct Wails pattern. |
| Font size shortcuts | `term.attachCustomKeyEventHandler()` | `document.addEventListener('keydown')` on parent | xterm.js captures all keyboard events inside its viewport. `attachCustomKeyEventHandler` runs BEFORE xterm processes the key, enabling clean interception without event propagation fights. |
| Font size state | Local state in `TerminalPanel` | Global state in `App` | Font size is per-terminal, not shared — local state avoids prop drilling and is cleaner. |
| Build script | bash `build.sh` | Go-based build tool (Mage/Task) | The signing/notarization commands are macOS CLI tools invoked via shell. Bash is the natural fit. The CI already uses these commands in YAML; `build.sh` is a local mirror of that. |
| Config persistence (last folder) | `os.UserConfigDir()` + JSON | SQLite / BoltDB | JSON config file is sufficient for a handful of preference values. No query needs; no reason for a database. |

---

## What NOT to Add

| Temptation | Why to Resist |
|------------|--------------|
| A UI component library (Radix, shadcn, MUI) | Scope is UI polish on existing components, not a full redesign. Adding a component library mid-project requires migrating existing components and resolving style conflicts. The current custom CSS is intentional and sufficient. |
| electron-builder or tauri for packaging | Already using Wails. Do not introduce a second desktop framework. |
| A cross-platform build tool (GoReleaser, Mage) | The `wails build` CLI handles the Go+JS compilation. GoReleaser is for Go-only binaries — it doesn't know how to invoke the frontend Vite build. A simple `build.sh` that shells out to `wails build` is the right scope. |
| Font size stored in localStorage | The Wails webview is a webview, not a persistent browser context. `localStorage` may work but its persistence across app restarts on Wails webview is not guaranteed cross-platform. Prefer Go-side config if persistence is required. |
| `@xterm/addon-search` | Not requested in this milestone. Defer. |
| React portals for modals | The existing `position: fixed` modal pattern works correctly in a Wails webview (no parent overflow clip issues). Portals add complexity with no benefit here. |

---

## Version Compatibility

| Package | Version | Compatible With | Notes |
|---------|---------|----------------|-------|
| `@xterm/addon-fit` | ^0.11.0 | `@xterm/xterm ^6.0.0` | Must use matching major version. 0.11.x is the addon version for xterm 6.x. |
| `wails/v2/pkg/runtime` | v2.10.2 | Go 1.26.1 | `OpenDirectoryDialog` available since Wails v2.0. No version concern. |
| `wails build` CLI | v2.10.2 | Verified locally | All flags documented above verified against `wails build --help` on v2.10.2. |

---

## Sources

- `wails build --help` (local) — HIGH confidence. All build flags verified against Wails v2.10.2 installed locally.
- [Wails runtime OpenDirectoryDialog](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/runtime) — HIGH confidence. Official pkg.go.dev documentation.
- [Wails OpenDialogOptions struct](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/dialog) — HIGH confidence. Official pkg.go.dev documentation showing all struct fields.
- [xterm.js Terminal.options (readable/writable)](https://xtermjs.org/docs/api/terminal/classes/terminal/) — HIGH confidence. Official xterm.js API docs confirm `options` is a read-write property. `terminal.options.fontSize = N` is the documented API.
- [xterm.js 6.0.0 release notes](https://github.com/xtermjs/xterm.js/releases) — HIGH confidence. fontSize API is unchanged in v6 (breaking changes were windowsMode, canvas renderer, overviewRulerWidth only).
- [xterm.js attachCustomKeyEventHandler](https://xtermjs.org/docs/api/terminal/classes/terminal/) — HIGH confidence. Official docs show this method runs before xterm's own key handler.
- [CSS flex min-height: 0 pattern](https://css-tricks.com/boxes-fill-height-dont-squish/) — HIGH confidence. Well-established CSS flexbox behavior, not a library-specific concern.
- [Wails Dialog JS unsupported](https://wails.io/docs/reference/runtime/dialog/) — MEDIUM confidence. Confirmed via search results that dialog must be called from Go side.

---

*Stack research for: AgentHub v1.1 Polish & Build milestone*
*Researched: 2026-03-19*
