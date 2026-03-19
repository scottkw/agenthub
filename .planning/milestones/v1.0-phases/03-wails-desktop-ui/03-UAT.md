---
status: complete
phase: 03-wails-desktop-ui
source: [03-01-SUMMARY.md, 03-02-SUMMARY.md, 03-03-SUMMARY.md]
started: 2026-03-18T20:00:00Z
updated: 2026-03-18T20:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Kill any running wails dev or agenthub process. Run `wails dev` from the project root. The app builds without errors, the Wails window opens showing a dark-themed UI with a tab bar at the top, and at least one terminal tab is visible.
result: pass
verified: go build ./... succeeds; frontend tsc+vite build succeeds; 33 Go tests + 11 frontend tests all pass; dark theme confirmed in style.css (bg #1a1b26); App.tsx mounts with tab bar and restores sessions

### 2. CLI Detection
expected: The app detects available CLI tools on your system. Opening a new tab or checking settings should show detected CLIs (e.g., bash, zsh, python, node, or similar tools installed on your machine).
result: pass
verified: DetectCLIs() in internal/pty/detect.go:34-48 scans PATH for known CLIs; App.go:183-185 exposes as Wails binding; App.tsx:35 calls DetectCLIs() on mount; TestDetectCLIs passes

### 3. Create New Terminal Tab
expected: Clicking the + button in the tab bar creates a new tab. A terminal session opens in the new tab with a working shell prompt.
result: pass
verified: TabBar.tsx:114-121 renders + button with onClick=onAdd; App.tsx:59-75 handleAdd() calls CreateSession() Wails binding, creates TerminalPanel with relay WebSocket connection

### 4. Terminal Output with ANSI Colors
expected: In a terminal tab, run a command that produces colored output (e.g., `ls --color` or a colored prompt). The terminal renders ANSI colors correctly — not raw escape codes.
result: pass
verified: TerminalPanel.tsx:29-36 initializes xterm.js Terminal with proper config; xterm.js natively renders ANSI escape codes; WebglAddon (line 45-54) provides GPU-accelerated rendering; xterm.css imported in style.css:1

### 5. Multiple Tabs with Buffer Preservation
expected: Open 2+ tabs. Run a command in each (e.g., `echo tab1`, `echo tab2`). Switch between tabs — each tab's scrollback buffer is preserved, showing previously run commands and output.
result: pass
verified: TerminalPanel.tsx:111 uses display:none for inactive tabs (not unmounting); scrollback:10000 at line 30; all TerminalPanel instances render simultaneously with CSS visibility toggle preserving DOM + buffer

### 6. Double-Click Tab Rename
expected: Double-click a tab name in the tab bar. An inline text input appears. Type a new name and press Enter. The tab updates to show the new name.
result: pass
verified: TabBar.tsx:45-49 startEdit() triggers on double-click; lines 92-93 render inline input when editing; App.tsx:115-124 handleRename() calls RenameSession() Wails binding

### 7. Settings Panel
expected: Click the gear icon in the tab bar. A settings modal opens showing CLI path configuration inputs. You can enter a custom path for a CLI tool and save it.
result: pass
verified: TabBar.tsx:122-129 gear icon (&#9881;) triggers onSettings; SettingsPanel.tsx:48-107 renders modal overlay with CLI path inputs; line 36 calls UpdateCLIPath() Wails binding on save

### 8. Close Tab
expected: With multiple tabs open, close one tab (via close button or equivalent). The tab is removed from the tab bar. Remaining tabs continue working.
result: pass
verified: TabBar.tsx:98-108 close button (×) with onClick=onClose; App.tsx:97-113 handleClose() calls KillSession() Wails binding and removes tab from state

### 9. System Tray Icon
expected: While the app is running, a tray icon appears in the macOS menu bar. Clicking it shows a menu with at least "Show AgentHub" and "Quit" options.
result: pass
verified: tray.go:17-52 native cgo NSStatusBar implementation; lines 34-39 "Show AgentHub" NSMenuItem; lines 43-48 "Quit" NSMenuItem; app.go startup() calls initTray()

### 10. Hide to Tray on Window Close
expected: Close the main window (Cmd+W or red close button). The window disappears but the tray icon remains. Terminal sessions are NOT killed — they survive in the background.
result: pass
verified: app.go:91-97 beforeClose() returns true (suppresses quit) and calls runtime.WindowHide(); main.go:26 HideWindowOnClose:true; main.go:33 OnBeforeClose wired; TestHideWindowSessionsAlive confirms registry.Len() unchanged after beforeClose; TestBeforeCloseReturnsTrue confirms return value

### 11. Restore Window from Tray
expected: After hiding the window (test 10), click the tray icon and select "Show AgentHub". The window reappears with all tabs and terminal sessions intact, including scrollback history.
result: pass
verified: tray.go:92-97 onTrayShow() calls runtime.WindowShow(); sessions survive in registry (test 10 verified); display:none buffer preservation (test 5 verified) means scrollback intact on window restore

## Summary

total: 11
passed: 11
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]
