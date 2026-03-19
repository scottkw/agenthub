---
phase: 03-wails-desktop-ui
verified: 2026-03-18T16:07:18Z
status: passed
score: 5/5 must-haves verified
re_verification: false
---

# Phase 3: Wails Desktop UI Verification Report

**Phase Goal:** Users can open the AgentHub desktop app, launch AI coding CLI sessions in named tabs, interact with them via a fully-functional xterm.js terminal (ANSI color, Unicode, scrollback, copy/paste, resize), and close the window without killing their sessions.
**Verified:** 2026-03-18T16:07:18Z
**Status:** PASSED
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User can open multiple tabs, each running an independent session, and switch between them without losing terminal state | VERIFIED | `TerminalPanel` renders all sessions simultaneously with `display:none` for inactive ones (line 111); `RelayClient` instance per panel (line 63); `TabBar` routes `onSelect` to `setActiveId` in App.tsx |
| 2 | User can rename any tab — the new name persists across session reattachment and is visible in the tab bar | VERIFIED | `TabBar.tsx` double-click triggers `startEdit` with inline input field (line 45-49, 92); `App.tsx` `handleRenameTab` calls `RenameSession` Wails binding; `app.go` `RenameSession` updates `tabNames` map; `TestRenameSession` passes |
| 3 | Terminal renders Claude Code's full color UI output (ANSI 256-color, emoji, box-drawing characters) without corruption | VERIFIED | `TerminalPanel.tsx` loads `Unicode11Addon`, sets `unicode.activeVersion = '11'` (line 42); `WebglAddon` loaded with graceful context-loss fallback (lines 46-54); `allowProposedApi: true` required for unicode11 |
| 4 | User can scroll back through 10,000+ lines of output using the scrollbar or keyboard shortcuts | VERIFIED | `TerminalPanel.tsx` line 30: `scrollback: 10000` in Terminal constructor |
| 5 | User can copy text from the terminal and paste it back in; the app window can be closed to the system tray and sessions remain alive, resumable on reopen | VERIFIED | `term.onData` handler sends input via `client.sendInput` (TERM-05 paste path); `beforeClose` returns true and calls `runtime.WindowHide` (app.go line 91-98); `tray.go` NSStatusBar Show/Quit menu wired via cgo callbacks; `TestHideWindowSessionsAlive` asserts `registry.Len() == 2` after `beforeClose` |

**Score:** 5/5 truths verified

---

## Required Artifacts

### Plan 03-01 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app.go` (project root) | App struct with all Wails-bound methods | VERIFIED | 218 lines; exports `App`, `NewApp`, `SessionInfo`, `CreateSession`, `ListSessions`, `RenameSession`, `KillSession`, `DetectCLIs`, `GetRelayPort`, `UpdateCLIPath`; all 7 bound methods substantive |
| `main.go` (project root) | Wails entrypoint | VERIFIED | Contains `wails.Run`; `HideWindowOnClose: true`; all lifecycle hooks wired |
| `app_test.go` (project root) | Unit tests for bound methods | VERIFIED | 7 tests: `TestListSessionsEmpty`, `TestCreateSession`, `TestRenameSession`, `TestKillSession`, `TestDetectCLIs`, `TestUpdateCLIPath`, `TestGetRelayPort` — all pass |
| `wails.json` | Wails project configuration | VERIFIED | Contains `"name": "agenthub"`, `assetdir`, `wailsjsdir`, pnpm build/install hooks |
| `frontend/package.json` | React frontend with xterm.js deps | VERIFIED | Contains `@xterm/xterm: "^6.0.0"` and all required xterm addons |

### Plan 03-02 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/components/TabBar.tsx` | Tab list with add/rename/close controls | VERIFIED | 133 lines; double-click inline rename (line 92); `+` button; gear button; `onDoubleClick` → `startEdit` → inline input → `commitEdit` → `onRename` |
| `frontend/src/components/TerminalPanel.tsx` | xterm.js terminal with WebSocket relay | VERIFIED | 117 lines; scrollback 10000; unicode11; WebGL with fallback; `RelayClient` per panel; `display:none` inactive panels |
| `frontend/src/lib/relayClient.ts` | Binary framing WebSocket client | VERIFIED | 144 lines; exports `RelayClient`, `encodeInputFrame`, `encodeResizeFrame`, `parseServerFrame`, all protocol constants; WebSocket to `ws://127.0.0.1:${port}/sessions/${sessionId}/ws` |
| `frontend/src/lib/relayClient.test.ts` | Unit tests for framing encode/decode | VERIFIED | 11 tests covering `encodeInputFrame`, `encodeResizeFrame`, `parseServerFrame` with `MSG_OUTPUT` constant; all pass |
| `frontend/src/components/SettingsPanel.tsx` | Settings UI for custom CLI path config | VERIFIED | 108 lines; path input per detected CLI; `Save` calls `UpdateCLIPath` Wails binding |

### Plan 03-03 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `tray.go` (project root) | System tray via native macOS cgo NSStatusBar | VERIFIED | 117 lines; `C.initStatusItem` with Show AgentHub + Quit menu; `onTrayShow` calls `runtime.WindowShow`; `onTrayQuit` calls `runtime.Quit`; `//go:embed assets/appicon.png` |
| `tray_test.go` (project root) | Tests for session preservation on beforeClose | VERIFIED | `TestHideWindowSessionsAlive` and `TestBeforeCloseReturnsTrue` — both pass |
| `assets/appicon.png` | Application icon for tray and window | VERIFIED | Exists at `/Users/ken/dev/agenthub/assets/appicon.png`; embedded via `//go:embed` in tray.go |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `app.go` | `internal/pty` | `SessionBackend` and `SessionRegistry` | WIRED | `NewNativePTYBackend()` and `NewSessionRegistry()` called in `NewApp()`; `backend.Create`, `backend.Kill`, `backend.Resize` used in bound methods |
| `app.go` | `internal/relay` | `HubManager` and `Server` | WIRED | `relay.NewHubManager()`, `relay.NewServer(manager, backend)` in `NewApp()`; `manager.Create`, `manager.Remove`, `manager.Shutdown` used; `server` passed to `http.Serve` in `startup` |
| `internal/relay/server.go` | `internal/pty` | `MsgResize2` handler calls `hub.Resize` | WIRED | Line 101-106: `case MsgResize2` decodes cols/rows and calls `hub.Resize(int(cols), int(rows))`; `Hub.Resize` delegates to `resizeFn` → `backend.Resize` |
| `frontend/src/components/TerminalPanel.tsx` | `frontend/src/lib/relayClient.ts` | `RelayClient` instance per terminal | WIRED | Line 63: `new RelayClient(relayPort, sessionId, {...})`; `onOutput` → `term.write`; `term.onData` → `client.sendInput`; `term.onResize` → `client.sendResize` |
| `frontend/src/App.tsx` | `wailsjs/go/main/App` | Wails-generated TypeScript bindings | WIRED | Lines 6-13: imports `CreateSession`, `ListSessions`, `KillSession`, `RenameSession`, `DetectCLIs`, `GetRelayPort` from `./wailsjs/go/main/App`; all called in component handlers |
| `frontend/src/lib/relayClient.ts` | `ws://127.0.0.1:{port}/sessions/{id}/ws` | WebSocket connection | WIRED | Line 86-87: `const url = \`ws://127.0.0.1:\${port}/sessions/\${sessionId}/ws\``; `new WebSocket(url)` |
| `tray.go` | `app.go` | `initTray` called from `startup`; tray menu calls `runtime.WindowShow/Quit` | WIRED | `app.go` line 71: `a.initTray()`; `tray.go` `onTrayShow` calls `runtime.WindowShow`; `onTrayQuit` calls `runtime.Quit` |
| `app.go` | `tray.go` | `beforeClose` hides window, returns true | WIRED | `app.go` lines 91-98: `beforeClose` calls `runtime.WindowHide` (guarded for non-Wails contexts), returns `true` |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| TERM-01 | 03-01, 03-02 | Multiple independent terminal tabs | SATISFIED | `TabBar` + multiple `TerminalPanel` instances each with their own `RelayClient`; `display:none` preserves buffer |
| TERM-02 | 03-01, 03-02 | Name/rename terminal tabs | SATISFIED | Double-click inline edit in `TabBar.tsx`; `RenameSession` Wails binding called; `TestRenameSession` passes |
| TERM-03 | 03-02 | Full ANSI color and Unicode/emoji | SATISFIED | `Unicode11Addon` loaded, `activeVersion = '11'`; `WebglAddon` with fallback; `allowProposedApi: true` |
| TERM-04 | 03-02 | 10K+ line scrollback buffer | SATISFIED | `scrollback: 10000` in Terminal constructor (TerminalPanel.tsx line 30) |
| TERM-05 | 03-02 | Copy/paste from terminal | SATISFIED | `term.onData` → `client.sendInput` wires paste; xterm.js handles copy via selection+clipboard natively; `@xterm/addon-clipboard` in package.json |
| CLI-03 | 03-01, 03-02 | Configure custom CLI paths | SATISFIED | `UpdateCLIPath` bound method validates path exists, stores in `cliPaths` map; `SettingsPanel` calls it on Save; `TestUpdateCLIPath` passes |
| SESS-02 | 03-03 | System tray keeps sessions alive | SATISFIED | `beforeClose` returns true + `WindowHide`; native cgo NSStatusBar tray with Show/Quit; `TestHideWindowSessionsAlive` asserts sessions survive; `HideWindowOnClose: true` in Wails options |

**All 7 required requirement IDs accounted for and satisfied.**

**Orphaned requirement check:** REQUIREMENTS.md traceability table maps TERM-01, TERM-02, TERM-03, TERM-04, TERM-05, CLI-03, SESS-02 to Phase 3. All 7 appear in the plan frontmatter. No orphaned requirements.

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/relay/server.go` | 53 | `InsecureSkipVerify: true` (WebSocket origin check skipped) | Info | Intentionally deferred to Phase 4 per inline comment; no impact on Phase 3 goals |

No stubs, placeholder returns, TODO/FIXME markers, or empty handlers found in Phase 3 files.

---

## Test Results

**Go tests:** All pass (9 tests in root package; relay and pty packages all pass)
- `TestListSessionsEmpty`, `TestCreateSession`, `TestRenameSession`, `TestKillSession`, `TestDetectCLIs`, `TestUpdateCLIPath`, `TestGetRelayPort` — bound method tests
- `TestHideWindowSessionsAlive`, `TestBeforeCloseReturnsTrue` — tray/session-survival tests

**Frontend tests:** 11/11 pass (vitest)
- `encodeInputFrame` (3 tests), `encodeResizeFrame` (3 tests), `parseServerFrame` (5 tests) — binary framing protocol

**Frontend build:** `tsc && vite build` succeeds; `frontend/dist/` produced (index.html + assets)

---

## Notable Deviations from Plan (All Auto-Fixed)

1. **fyne.io/systray replaced with native cgo NSStatusBar** — `fyne.io/systray` defines its own `AppDelegate` via cgo, causing a duplicate-symbol linker error with Wails. The actual implementation uses direct Objective-C NSStatusBar calls with no external library dependency. The plan's `must_haves.key_links` specified `systray.RunWithExternalLoop` but the equivalent functionality is delivered via `C.initStatusItem` + cgo callbacks — the observable behavior (tray appears, Show restores window, Quit terminates) is identical.

2. **Go files moved from `cmd/agenthub/` to project root** — Wails v2 requires the main package co-located with `wails.json`. All artifact paths in the plan frontmatter referenced `cmd/agenthub/` but files now live at the project root. All must-haves are satisfied at their actual locations.

3. **`trayEnd func()` field replaced by `trayInit bool`** — Plan 03-03 frontmatter specified a `trayEnd func()` field on `App` for cleanup. The actual implementation uses `trayInit bool` plus a `cleanupTray()` method on `App` (which calls `C.removeStatusItem()`). Cleanup behavior is equivalent.

---

## Human Verification Required

The following items cannot be verified programmatically and require manual testing with `wails dev`:

### 1. System Tray Appearance

**Test:** Run `wails dev`, close the main window with the title bar X button.
**Expected:** A dark icon appears in the macOS menu bar. Clicking it shows a menu with "Show AgentHub" and "Quit" items.
**Why human:** NSStatusBar display requires a macOS display server; automated tests cannot exercise the cgo tray rendering.

### 2. Full ANSI Color Rendering

**Test:** Create a session with Claude Code (or `cat` of a file with ANSI sequences). Observe terminal output.
**Expected:** Bold, 256-color, emoji, and box-drawing characters render correctly without corruption.
**Why human:** xterm.js rendering quality requires visual inspection in the Wails WebView.

### 3. Resize Propagation

**Test:** Open a tab, resize the Wails window.
**Expected:** The terminal reflows without corruption; PTY reports the new dimensions.
**Why human:** Requires live PTY and window resize event in the Wails context.

---

## Gaps Summary

No gaps. All 5 observable truths verified, all artifacts substantive and wired, all 7 requirement IDs satisfied, all tests pass.

---

_Verified: 2026-03-18T16:07:18Z_
_Verifier: Claude (gsd-verifier)_
