# Architecture Research

**Domain:** Cross-platform desktop terminal app (Go/Wails + React/xterm.js) — v1.1 feature integration
**Researched:** 2026-03-19
**Confidence:** HIGH — based on direct codebase inspection of all affected files

---

## Existing System Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                     Wails Desktop Window                           │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  React Frontend  (frontend/src/)                             │  │
│  │  ┌──────────┐  ┌───────────────┐  ┌────────────────────────┐│  │
│  │  │  TabBar  │  │TerminalPanel  │  │    SettingsPanel       ││  │
│  │  │(toolbar+ │  │(xterm.js +    │  │    (modal overlay)     ││  │
│  │  │ status   │  │ FitAddon +    │  └────────────────────────┘│  │
│  │  │  dots)   │  │ RelayClient)  │  ┌──────────────┐          │  │
│  │  └──────────┘  └───────────────┘  │  QRModal     │          │  │
│  │                                   └──────────────┘          │  │
│  │  App.tsx (root state: tabs, statuses, webEnabled, URLs)      │  │
│  └──────────────────────────┬───────────────────────────────────┘  │
│                             │ Wails IPC (wailsjs bindings)          │
│  ┌──────────────────────────┴───────────────────────────────────┐  │
│  │  Go Backend  (app.go + internal/)                            │  │
│  │  ┌───────────┐  ┌──────────────┐  ┌──────────────────────┐  │  │
│  │  │  pty/     │  │  relay/      │  │  webserver/          │  │  │
│  │  │ SessionReg│  │  HubManager  │  │  WebServer (HTTPS)   │  │  │
│  │  │ NativePTY │  │  Hub (fan-   │  │  dashboard.html      │  │  │
│  │  │ Backend   │  │  out, scroll │  │  terminal.html       │  │  │
│  │  │ detector  │  │  back)       │  │  AuthManager+Tokens  │  │  │
│  │  └───────────┘  └──────────────┘  └──────────────────────┘  │  │
│  │  App struct (tabNames map, cliPaths map, sessionStatuses map) │  │
│  │  HTTP relay server on 127.0.0.1:RANDOM_PORT (WS relay)       │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  macOS tray (NSStatusBar), linux/windows stubs                      │
└────────────────────────────────────────────────────────────────────┘
```

### Existing Key Data Flows

**PTY output → desktop terminal:**
PTY process → Hub.Run() reads 32 KiB chunks → wraps in MsgOutput frame → broadcasts to all Hub.Subscribers → RelayClient (WS to 127.0.0.1:RELAY_PORT) → xterm.js.write()

**PTY output → web browser:**
Same Hub.Subscribers → WebServer.handleWSSRelay pump → browser xterm.js in terminal.html

**Session status:**
status.Watch goroutine subscribes to Hub output → pattern-matches → runtime.EventsEmit("session:status") → App.tsx EventsOn handler → setSessionStatuses state

**Tab rename:**
App.tsx handleRenameTab → RenameSession(id, name) Wails call → app.go writes to tabNames map → ListSessions reads tabNames → returned to frontend on next ListSessions call

---

## New Features: Integration Points

### 1. Build Script (`build.sh`)

**Nature:** New file — no existing code modified.

**Integration points:**
- Wraps `wails build -platform {target}` — the same invocation the CI YAML already uses
- macOS signing: replicates the exact codesign/notarytool sequence in `.github/workflows/build.yml`:
  keychain create → import certificate.p12 → codesign with `--options runtime` + `build/entitlements.plist` → notarytool submit → stapler staple → keychain delete
- Build output lands in `build/bin/` (Wails convention, already used by CI artifact upload steps)
- No Go or React code changes required
- Must handle: `darwin/universal`, `linux/amd64`, `windows/amd64`
- macOS signing is conditional on env vars being set (same pattern CI uses for `MACOS_CERTIFICATE`)

**Critical constraint:** The script is a developer convenience wrapper only. It must not duplicate configuration that belongs in `wails.json` or the CI YAML — those remain authoritative.

---

### 2. Terminal Full-Fill Fix

**Existing problem:** `TerminalPanel` returns a div with `flex: 1; width: 100%; minHeight: 0`. The `.terminal-wrapper` has `display: flex; flex-direction: column; height: 100%`. The `.web-serving-bar` is `flex-shrink: 0` with no explicit height. The bug is a layout race: when `display: none → display: flex` happens on tab switch (or initial mount), the browser has not yet committed the final layout, so `fitAddon.fit()` fires against a zero or stale container size.

**Integration points:**
- Modify: `frontend/src/style.css` — ensure `.terminal-container` has `position: absolute; inset: 0` or equivalent to give the `terminal-wrapper` a concrete pixel height
- Modify: `frontend/src/components/TerminalPanel.tsx` — the single `requestAnimationFrame(() => fitAddon.fit())` in the init effect may need a second pass; consider using a `MutationObserver` or a second RAF after the first resolves
- The `ResizeObserver` in the `isActive` effect already handles ongoing size changes correctly — the bug is only at initial render/tab-switch
- Do not restructure the `display: none` tab-hiding pattern — that pattern preserves xterm buffer state and is correct

---

### 3. Per-Tab Status Bar (replacing inline web-serving controls)

**Existing state:** In `App.tsx`, a `.web-serving-bar` div is rendered inline inside `.terminal-wrapper` for each tab. It contains: web toggle button, session URL link, copy token button, QR button. This is unstyled (no explicit height, no separator line).

**New component:** `StatusBar` — a fixed-height bar at the bottom of each terminal panel that shows:
- Web on/off toggle button
- Session URL (when web is on and web server is running)
- Copy token link button
- QR button
- Session status indicator (running / waiting / idle / errored)

**Integration points:**
- New file: `frontend/src/components/StatusBar.tsx`
- Modify: `frontend/src/App.tsx` — replace the inline `web-serving-bar` JSX block with `<StatusBar />`. Props: `sessionId`, `webEnabled`, `sessionURL`, `webServerRunning`, `onToggleWeb`, `onCopyToken`, `onShowQR`, `status`
- Modify: `frontend/src/style.css` — add `.status-bar` rules (height, border-top, flex layout); repurpose or remove `.web-serving-bar` rules
- StatusBar is always rendered (not gated on `webServerRunning`) so the status indicator is always visible; web controls appear conditionally inside it

**Data flow:** All StatusBar props come from existing App.tsx state — no new Go backend calls required.

---

### 4. Settings Modal Overhaul

**Existing state:** `SettingsPanel` is ~334 LOC mixing CLI paths, web serving controls, and CA cert instructions in a single scrollable body div. Three `<h3>` sections exist but lack visual separation.

**Integration points:**
- Modify: `frontend/src/components/SettingsPanel.tsx` — restructure body layout into visually separated sections (e.g. tabs or accordion); no change to Wails bindings or internal state
- Modify: `frontend/src/style.css` — update `.settings-panel*` rules for new layout

**No new Go backend methods required.** All existing Wails bindings (`UpdateCLIPath`, `SetWebPassword`, `GetNetworkInterfaces`, `StartWebServer`, `StopWebServer`, `GetWebServerURL`, `GetCACertPath`, `IsWebServerRunning`) remain unchanged.

---

### 5. Toolbar Buttons (larger hit targets)

**Existing state:** Tab bar control buttons (`+` and gear) are 28×28px in `.tab-bar__btn`. They use plain character glyphs (`+` and `&#9881;`).

**Integration points:**
- Modify: `frontend/src/style.css` — increase `.tab-bar__btn` to at least 32×32px and bump `font-size`
- Optionally modify: `frontend/src/components/TabBar.tsx` — replace character glyphs with inline SVG icons for crisper rendering at larger sizes

No backend changes.

---

### 6. New-Session Modal with Agent Picker + Folder Browser

**Existing state:** When `detectedCLIs.length > 1`, `App.tsx` renders a `.cli-picker-overlay` — absolute-positioned divs. When only one CLI exists, `CreateSession(cli, name)` is called immediately. The `CreateSession` Wails method takes `(cli, name string)` — no working directory.

**New component:** `NewSessionModal` — a proper modal dialog with:
- Agent/CLI picker (list of detected CLIs)
- Folder browser (path input + "Browse" button triggering Wails `OpenDirectoryDialog`)
- Session name input (optional, auto-generated default)
- "Remember last folder" via `localStorage`

**Integration points — Frontend:**
- New file: `frontend/src/components/NewSessionModal.tsx`
- Modify: `frontend/src/App.tsx` — replace `showCLIPicker` state + inline picker JSX with `showNewSessionModal` + `<NewSessionModal />`. The modal calls `createTab(cli, name, workDir)` on confirm.
- Modify: `frontend/src/style.css` — remove `.cli-picker*` rules; add `.new-session-modal*` rules

**Integration points — Go backend:**
- Modify: `app.go` `CreateSession` method — add `workDir string` parameter:
  ```go
  func (a *App) CreateSession(cli, name, workDir string) (string, error)
  ```
- Add: `app.go` `BrowseFolder` method — wraps `runtime.OpenDirectoryDialog`:
  ```go
  func (a *App) BrowseFolder() (string, error) {
      return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
          Title: "Select working folder",
      })
  }
  ```
- Modify: `internal/pty/backend.go` — add `WorkDir string` field to `CreateRequest` struct
- Modify: `internal/pty/native.go` — set `cmd.Dir = req.WorkDir` when `WorkDir` is non-empty at PTY spawn time

**Persist last folder:** `localStorage.setItem('agenthub.lastFolder', workDir)` inside the modal component — no backend needed.

**Web interface:** The web-served `terminal.html` has no new-session UI. This feature is desktop-only — no changes to the web layer.

---

### 7. Tab Renaming with Web Dashboard Propagation

**Existing state:** Tab renaming already works end-to-end for the desktop: `RenameSession(id, name)` writes to `app.tabNames[id]`; `ListSessions()` reads it. The web dashboard's `GET /api/sessions` currently returns `[]string` (just session IDs). The `dashboard.html` `renderSessions` function already handles a `session.name` field — it has: `const name = (typeof s === 'object' && s.name) ? s.name : id` — so names already display correctly if the API returns objects.

**Gap:** The API returns `[]string`, not objects. The dashboard shows IDs instead of names.

**Integration points:**
- Modify: `internal/webserver/server.go` — change `handleListSessions` response from `[]string` to `[]SessionSummary`:
  ```go
  type SessionSummary struct {
      ID      string `json:"id"`
      Name    string `json:"name"`
      CLIType string `json:"cli_type"`
  }
  ```
- Modify: `internal/webserver/server.go` `Config` struct — add a `NameFunc func(id string) (name, cliType string)` callback field. The webserver does not import the main package; a callback avoids a circular import.
- Modify: `app.go` — add `getTabName(id string) (string, string)` helper method and wire it to `Config.NameFunc` when calling `NewWebServer`.
- Modify: `web/dashboard.html` — no structural change needed (already handles `s.name`); polish the CSS and layout as part of the dashboard visual refresh.

**No change to relay protocol, Hub, or auth layers required.**

---

### 8. Per-Tab Font Size (SHIFT+ / SHIFT-)

**Existing state:** `TerminalPanel` creates its `Terminal` with hardcoded `fontSize: 14`. The terminal instance is fully encapsulated in `termRef.current` and not accessible from `App.tsx`.

**Integration points:**
- Modify: `frontend/src/App.tsx` — add `tabFontSizes: Record<string, number>` state (default 14 per tab); register a `window` keydown listener for `SHIFT+=` (SHIFT+) and `SHIFT+-` on the active tab:
  ```typescript
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (!activeId) return
      if (e.shiftKey && e.key === '=') {
        setTabFontSizes(prev => ({ ...prev, [activeId]: Math.min((prev[activeId] ?? 14) + 1, 32) }))
      }
      if (e.shiftKey && e.key === '-') {
        setTabFontSizes(prev => ({ ...prev, [activeId]: Math.max((prev[activeId] ?? 14) - 1, 8) }))
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [activeId])
  ```
- Modify: `frontend/src/components/TerminalPanel.tsx` — add `fontSize` prop; add a `useEffect([fontSize])` that mutates `term.options.fontSize` and calls `fitAddonRef.current?.fit()`. Do not recreate the terminal — xterm.js supports live `options.fontSize` mutation.

**Note on keyboard capture:** xterm.js captures keyboard events inside its canvas before they bubble. `window.addEventListener` fires for global shortcuts. The `window`-level listener in App.tsx is the correct approach — `attachCustomKeyEventHandler` is for events xterm processes (printable chars, navigation), not app-level shortcuts.

**No Go backend changes required.**

---

### 9. Web Dashboard Visual Refresh

**Existing state:** `web/dashboard.html` is a single self-contained HTML file with all styles and JS inline. Embedded via `//go:embed` in `web/embed.go`. No build step.

**Integration points:**
- Modify: `web/dashboard.html` — update inline CSS; improve session list layout to show names (see §7 above); this change is coordinated with the `GET /api/sessions` shape change
- The dashboard visual refresh and the session name propagation (§7) must be deployed together — they are the same file

---

## Revised Component Map

```
frontend/src/
├── App.tsx                  MODIFIED — new state: tabFontSizes, showNewSessionModal;
│                                        replaces showCLIPicker + inline picker JSX with
│                                        NewSessionModal; font-size keydown listener;
│                                        StatusBar props wiring
├── style.css                MODIFIED — StatusBar rules, toolbar sizing,
│                                        terminal fill CSS, NewSessionModal rules,
│                                        remove .cli-picker* rules
├── components/
│   ├── TabBar.tsx           MODIFIED — larger buttons (CSS or SVG glyph swap)
│   ├── TerminalPanel.tsx    MODIFIED — adds fontSize prop; useEffect mutates
│   │                                   term.options.fontSize + re-fit on change
│   ├── SettingsPanel.tsx    MODIFIED — layout overhaul; no API changes
│   ├── QRModal.tsx          UNCHANGED
│   ├── StatusBar.tsx        NEW — per-tab bar: web controls + status indicator
│   └── NewSessionModal.tsx  NEW — agent picker + folder browser + name input
└── lib/
    └── relayClient.ts       UNCHANGED

app.go                       MODIFIED — CreateSession adds workDir param;
│                                        new BrowseFolder() method;
│                                        new getTabName() helper;
│                                        NameFunc wired into webserver Config
internal/
├── pty/
│   ├── backend.go           MODIFIED — CreateRequest.WorkDir field added
│   └── native.go            MODIFIED — cmd.Dir = req.WorkDir when non-empty
│   └── [others]             UNCHANGED
├── relay/                   UNCHANGED
└── webserver/
    └── server.go            MODIFIED — Config gains NameFunc callback;
                                         handleListSessions returns []SessionSummary
    └── [others]             UNCHANGED
web/
└── dashboard.html           MODIFIED — CSS polish + session name display
                                         (coordinated with server.go API shape change)

build.sh                     NEW — local build wrapper: wails build + macOS signing
```

---

## Data Flow Changes

### Session Name Propagation to Web Dashboard

```
Desktop: RenameSession(id, name)
    → app.go: tabNames[id] = name          (existing)
    → setTabs() in App.tsx                 (existing — desktop tab updates immediately)

Web request: GET /api/sessions
    → webserver.handleListSessions()
    → calls cfg.NameFunc(id) for each web-enabled session    (NEW)
    → cfg.NameFunc = app.getTabName                          (NEW wiring in app.go)
    → returns []SessionSummary{id, name, cli_type}           (NEW response shape)
    → dashboard.html renderSessions() reads s.name           (already handles this)
```

### Working Directory for New Sessions

```
NewSessionModal (React)
    → user picks folder: BrowseFolder() Wails call           (NEW backend method)
    → result stored in localStorage as 'agenthub.lastFolder' (NEW, frontend-only)
    → CreateSession(cli, name, workDir) Wails call           (MODIFIED signature)
    → app.go resolves CLI path, passes workDir to CreateRequest
    → pty.CreateRequest{CLI, Cols, Rows, WorkDir}            (MODIFIED struct)
    → native.go: cmd.Dir = req.WorkDir if non-empty          (MODIFIED PTY spawn)
```

### Font Size State Flow

```
Window keydown (SHIFT+/SHIFT-)
    → App.tsx handler updates tabFontSizes[activeId]
    → TerminalPanel receives fontSize prop change
    → useEffect([fontSize]): term.options.fontSize = fontSize; fitAddon.fit()
    → xterm reflows columns/rows to new cell size
    → term.onResize fires → RelayClient.sendResize(cols, rows)
    → Hub.Resize() → PTY resized to new dimensions           (existing resize path)
```

---

## Architectural Patterns to Follow

### Pattern: Self-Contained Terminal Panel

`TerminalPanel` creates, owns, and destroys its xterm instance. Switching tabs uses `display: none` — the panel is never re-mounted for the same `sessionId`. This pattern preserves the xterm buffer and relay connection.

**For font size:** Add a `fontSize` prop and react to changes with `useEffect([fontSize])`. Do not lift terminal instance state out of `TerminalPanel` into App.tsx.

### Pattern: App.tsx as Thin State Hub

All cross-cutting state (tabs, webEnabled, sessionStatuses, tabFontSizes) lives in `App.tsx` and flows down as props. Child components do not call Wails bindings directly (except `SettingsPanel`, which manages its own local state for web-serving controls). New state for font sizes and the new-session modal follows this pattern.

### Pattern: Wails IPC Boundary Is Go Methods on App

All Wails-exposed methods live on the `App` struct in `app.go`. They return simple JSON-serializable types. Adding `BrowseFolder()` and modifying `CreateSession()` follows the existing convention. The TypeScript bindings in `wailsjs/` regenerate automatically on `wails dev`.

### Pattern: Embedded Web Assets with No Build Step

`web/dashboard.html` and `web/terminal.html` are embedded via `//go:embed`. No build tooling — vanilla HTML/CSS/JS only. The dashboard refresh must remain self-contained in a single HTML file. Do not introduce a CDN dependency beyond the existing xterm.js CDN references in `terminal.html`.

### Pattern: NameFunc Callback for Webserver Decoupling

The `webserver` package must not import the `main` package (circular import). All cross-boundary data (like session names) flows via callback fields in `webserver.Config`. This is the existing pattern for `HubManager` (passed in as a concrete type) extended to name resolution.

---

## Anti-Patterns to Avoid

### Anti-Pattern: Recreating the Terminal for Font Size

**What:** Disposing and re-creating the `Terminal` instance when font size changes.
**Why wrong:** Destroys the scrollback buffer, disconnects the relay, causes a visible flash.
**Do this instead:** Mutate `term.options.fontSize` in-place, then call `fitAddon.fit()`. xterm.js supports this — viewport reflows without losing buffer state.

### Anti-Pattern: Working Directory in the Binary Framing Protocol

**What:** Adding `workDir` as a new frame type in the relay binary protocol.
**Why wrong:** `workDir` is a one-time session creation parameter, not a runtime I/O message.
**Do this instead:** Pass `workDir` through `CreateSession` → `CreateRequest` → `cmd.Dir` at PTY spawn time. The relay layer never sees it.

### Anti-Pattern: Polling Session Names in the Web Dashboard

**What:** Using a timer or WebSocket push to keep the dashboard session list current.
**Why wrong:** Unnecessary complexity — the list is already manually refreshed.
**Do this instead:** Serve names correctly in `GET /api/sessions`. Dashboard fetches on load and on the existing "Refresh" button. That is sufficient.

### Anti-Pattern: Global Keyboard Shortcuts via xterm attachCustomKeyEventHandler

**What:** Using xterm's `attachCustomKeyEventHandler` for the SHIFT+/SHIFT- font size shortcuts.
**Why wrong:** That handler intercepts events before xterm acts on them — it is for overriding terminal key behavior, not app-level shortcuts.
**Do this instead:** `window.addEventListener('keydown', ...)` in App.tsx, checked for `e.shiftKey`. This captures events from all sources including the xterm canvas.

---

## Build Order for v1.1 Features

The features have these dependencies:

```
Terminal fill fix           — CSS + TerminalPanel only; no deps
Toolbar larger buttons      — CSS only; independent
StatusBar component         — needs App.tsx web-serving state (already exists); no new backend
Settings modal overhaul     — pure frontend; independent
Per-tab font size           — needs TerminalPanel fontSize prop; no backend
New-session modal           — needs BrowseFolder() Go method + CreateSession workDir
Tab rename → web prop.      — needs webserver API shape change + NameFunc callback
Build script                — independent of all code; written last against stable binary
```

**Recommended build sequence:**

| Step | Features | Reason |
|------|----------|--------|
| 1 | Terminal fill + toolbar sizing | Pure CSS; validates layout baseline for all subsequent UI work |
| 2 | StatusBar component | Extracts existing inline JSX; tests the status indicator placement |
| 3 | Settings modal overhaul | Isolated component refactor; reduces clutter before adding more settings |
| 4 | Per-tab font size | Adds TerminalPanel `fontSize` prop; no backend; validates the prop+effect pattern |
| 5 | New-session modal + folder browser | Requires backend changes; largest scope; implement after frontend patterns are stable |
| 6 | Tab rename web propagation | Requires backend + webserver API shape change; coordinate with dashboard.html refresh |
| 7 | Build script | Written after all code is stable; tested against the final binary |

---

## Integration Risk Assessment

| Feature | Risk | Reason |
|---------|------|--------|
| Terminal fill fix | LOW | Pure CSS change; ResizeObserver already handles ongoing resizes |
| Toolbar sizing | LOW | CSS only |
| StatusBar | LOW | Extracts existing inline JSX; no new data flows |
| Settings overhaul | LOW | No API surface changes |
| Per-tab font size | MEDIUM | `window` keydown listener interaction with xterm canvas needs verification; `term.options.fontSize` mutation is stable in xterm 5.x/6.x but requires fit() call |
| New-session modal + workDir | MEDIUM | Requires Go backend changes; `runtime.OpenDirectoryDialog` behavior on all three platforms (macOS/Linux/Windows) needs verification |
| Tab rename web propagation | MEDIUM | Changes `GET /api/sessions` response shape; backward-compatible since `dashboard.html` already handles both string and object array formats |
| Build script macOS signing | HIGH | Requires local Developer ID certificate + notarization credentials; codesign + notarytool behavior differs between local and CI keychain setups |

---

## Sources

- Direct codebase inspection: `app.go`, `App.tsx`, `TerminalPanel.tsx`, `TabBar.tsx`, `SettingsPanel.tsx`, `style.css`, `relayClient.ts`, `internal/relay/hub.go`, `internal/relay/server.go`, `internal/webserver/server.go`, `web/dashboard.html`, `web/terminal.html`, `.github/workflows/build.yml`, `internal/pty/detect.go`, `internal/pty/registry.go`, `wails.json`
- xterm.js `terminal.options.fontSize` live mutation: supported since xterm 4.x, present in xterm 5.x/6.x API
- Wails v2 `runtime.OpenDirectoryDialog`: documented in Wails v2 runtime API
- `//go:embed` pattern: `web/embed.go` confirmed in codebase

---

*Architecture research for: AgentHub v1.1 Polish & Build milestone*
*Researched: 2026-03-19*
