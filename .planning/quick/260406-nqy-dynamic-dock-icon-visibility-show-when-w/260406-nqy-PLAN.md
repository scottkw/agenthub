---
phase: quick
plan: 260406-nqy
type: execute
wave: 1
depends_on: []
files_modified:
  - tray_objc.m
  - tray.go
  - tray_linux.go
  - tray_windows.go
  - App.go
autonomous: true
requirements: [DOCK-VISIBILITY]

must_haves:
  truths:
    - "Dock icon is visible when the main window is shown"
    - "Dock icon is hidden when the window is minimized or closed to tray"
    - "Toggling show/hide cycles dock visibility correctly on macOS"
    - "Windows and Linux builds compile without errors (stubs)"
  artifacts:
    - path: "tray_objc.m"
      provides: "setDockVisible C function for macOS activation policy toggling"
      contains: "setDockVisible"
    - path: "tray.go"
      provides: "Go wrapper calling setDockVisible from cgo"
      contains: "setDockVisible"
    - path: "App.go"
      provides: "Dock visibility calls wired into window show/hide lifecycle"
      contains: "setDockVisible"
  key_links:
    - from: "App.go (beforeClose, domReady, onTrayShow)"
      to: "tray.go setDockVisible"
      via: "direct function call"
      pattern: "setDockVisible"
    - from: "tray.go"
      to: "tray_objc.m C.setDockVisible"
      via: "cgo bridge"
      pattern: "C\\.setDockVisible"
---

<objective>
Implement dynamic dock icon visibility that tracks window state: dock icon appears when the window is visible, disappears when the window hides to tray.

Purpose: Users expect the dock icon to reflect app visibility. Currently it is permanently hidden via NSApplicationActivationPolicyAccessory set at tray init time.
Output: Modified tray and app lifecycle files across all three platforms.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@tray_objc.m
@tray.go
@tray_linux.go
@tray_windows.go
@App.go
@main.go

<interfaces>
From tray_objc.m (current C functions exposed to Go):
```c
void initStatusItem(const void *iconData, int iconLen);
void removeStatusItem(void);
void updateTrayIcon(const void *iconData, int iconLen);
void updateTrayTooltip(const char *tooltip);
void setTraySessionData(const char **names, const char **ids, int count);
```

From tray.go (current Go exports/callbacks):
```go
//export onTrayShow
func onTrayShow()
//export onTrayQuit
func onTrayQuit()
//export onTraySession
func onTraySession(idx C.int)
func (a *App) updateTray(sessions []SessionInfo, connected bool)
func (a *App) initTray()
func (a *App) cleanupTray()
```

From App.go (window lifecycle methods):
```go
func (a *App) domReady(ctx context.Context)    // shows window
func (a *App) beforeClose(ctx context.Context) bool  // hides window to tray
func (a *App) startup(ctx context.Context)
```

From main.go (Wails options):
```go
HideWindowOnClose: true
StartHidden: true
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add setDockVisible C function and Go wrappers for all platforms</name>
  <files>tray_objc.m, tray.go, tray_linux.go, tray_windows.go</files>
  <action>
1. In tray_objc.m, add a new C function `setDockVisible(int visible)`:
   - When `visible` is truthy: call `[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular]` then `[NSApp activateIgnoringOtherApps:YES]` to bring app forward and show dock icon.
   - When `visible` is falsy: call `[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory]`.
   - Wrap in `dispatch_async(dispatch_get_main_queue(), ^{ ... })` like other functions in this file.

2. In tray_objc.m `initStatusItem`, REMOVE the existing line `[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory]` (line 87). The dock will start hidden because `StartHidden: true` means domReady calls WindowShow which will trigger dock-show, and beforeClose triggers dock-hide. Actually: since the app starts hidden, dock should start hidden. So keep the accessory policy set in initStatusItem but note that domReady will flip it to Regular.

3. In tray.go, add the C function declaration in the cgo preamble:
   ```
   void setDockVisible(int visible);
   ```
   Then add a Go function:
   ```go
   func (a *App) setDockVisible(visible bool) {
       if visible {
           C.setDockVisible(1)
       } else {
           C.setDockVisible(0)
       }
   }
   ```

4. In tray_linux.go and tray_windows.go, add no-op stubs:
   ```go
   func (a *App) setDockVisible(visible bool) {}
   ```

This ensures all platforms compile. The macOS implementation toggles between NSApplicationActivationPolicyRegular and NSApplicationActivationPolicyAccessory.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build -tags dev -o /dev/null ./... 2>&1 | head -20</automated>
  </verify>
  <done>setDockVisible function exists on all platforms; macOS implementation toggles activation policy; project compiles on current platform.</done>
</task>

<task type="auto">
  <name>Task 2: Wire dock visibility into window lifecycle events</name>
  <files>App.go, tray.go</files>
  <action>
1. In App.go `domReady` method (called when window becomes visible after startup), add `a.setDockVisible(true)` after the `runtime.WindowShow(ctx)` call. This shows the dock icon when the window first appears.

2. In App.go `beforeClose` method, add `a.setDockVisible(false)` just before `runtime.WindowHide(ctx)` (inside the `ctx.Value("frontend") != nil` block). This hides the dock icon when the window closes to tray.

3. In tray.go `onTrayShow` callback (called when user clicks "Open AgentHub" in tray menu), add dock visibility:
   ```go
   func onTrayShow() {
       if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
           runtime.WindowShow(trayCallbackApp.ctx)
           trayCallbackApp.setDockVisible(true)
       }
   }
   ```

4. In tray.go `onTraySession` callback (called when user clicks a session in tray menu), the window is already shown via `runtime.WindowShow`. Add `trayCallbackApp.setDockVisible(true)` right after that WindowShow call inside the goroutine.

This covers all window show/hide transitions:
- Window shows (domReady, tray Show, tray Session click) -> dock visible
- Window hides (beforeClose) -> dock hidden
- App quits (onTrayQuit sets quitting=true, beforeClose returns false) -> no dock toggle needed, app exits
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build -tags dev -o /dev/null ./... 2>&1 | head -20</automated>
  </verify>
  <done>All window show paths call setDockVisible(true); all window hide paths call setDockVisible(false); dock icon tracks window visibility on macOS; Windows/Linux compile with no-op stubs.</done>
</task>

</tasks>

<verification>
1. Build succeeds: `go build -tags dev -o /dev/null ./...`
2. Manual test on macOS: launch app, confirm dock icon appears when window shows, closes to tray hides dock icon, tray "Open AgentHub" restores dock icon.
3. Cross-compile check (if available): ensure no build errors on darwin target.
</verification>

<success_criteria>
- macOS: Dock icon visible when window is shown, hidden when window minimizes/closes to tray
- All three platforms (macOS, Windows, Linux) compile without errors
- No change to existing tray menu behavior (sessions, quit, tooltips still work)
- Window lifecycle (startup hidden, show on DOM ready, hide on close, quit from tray) unchanged
</success_criteria>

<output>
After completion, create `.planning/quick/260406-nqy-dynamic-dock-icon-visibility-show-when-w/260406-nqy-SUMMARY.md`
</output>
