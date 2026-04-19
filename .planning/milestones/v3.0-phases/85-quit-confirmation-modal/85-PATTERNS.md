# Phase 85: Quit Confirmation Modal - Pattern Map

**Mapped:** 2026-04-19
**Files analyzed:** 7 new/modified files
**Analogs found:** 7 / 7

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/QuitConfirmModal.tsx` | component | request-response | `frontend/src/components/NewSessionModal.tsx` | exact (same modal structure, BEM classes, overlay pattern) |
| `frontend/src/App.tsx` | component | event-driven | `frontend/src/App.tsx` (self — add to existing EventsOn block) | exact (same pattern already repeated 5×) |
| `app.go` — `beforeClose` refactor + `QuitGUIOnly`/`QuitAll` methods | service | request-response | `app.go` — `emitExitEvent`, `ListSessions`, existing bound methods | exact (same App receiver method pattern) |
| `tray.go` — `onTrayQuit` refactor | middleware | event-driven | `tray.go` — `onTrayShow` (exact same window-show-then-emit pattern) | exact |
| `tray_objc_darwin.m` — add `sendNotification` C function | utility | request-response | `tray_objc_darwin.m` — `updateTrayTooltip` / `setDockVisible` (dispatch_async + ObjC API pattern) | exact |
| `notification_darwin.go` — cgo wrapper for sendNotification | utility | request-response | `tray.go` — cgo header declarations and C function call pattern | exact |
| `frontend/src/components/__tests__/QuitConfirmModal.test.tsx` | test | — | `frontend/src/components/__tests__/NewSessionModal.test.tsx` + `App.exit.test.tsx` | exact (source-inspection style) |

---

## Pattern Assignments

### `frontend/src/components/QuitConfirmModal.tsx` (component, request-response)

**Analog:** `frontend/src/components/NewSessionModal.tsx` (overlay + BEM) and `frontend/src/components/QRModal.tsx` (Escape key)

**Imports pattern** (`NewSessionModal.tsx` lines 1-3; `QRModal.tsx` line 1):
```typescript
import React, { useEffect, useRef } from 'react'
// Bound methods — added after wails generate module:
import { QuitGUIOnly, QuitAll } from '../wailsjs/go/main/App'
import type { SessionInfo } from '../wailsjs/go/main/App'
```

**Props interface** (from UI-SPEC `85-UI-SPEC.md` Component Inventory):
```typescript
interface QuitConfirmModalProps {
  isOpen: boolean
  sessions: Array<{ id: string; name: string; status: string }>
  onQuitGUI: () => void
  onQuitAll: () => void
  onCancel: () => void
}
```

**Guard pattern — early return when not open** (`NewSessionModal.tsx` line 29):
```typescript
if (!isOpen) return null
```

**Overlay + stopPropagation pattern** (`NewSessionModal.tsx` lines 66-67):
```typescript
<div className="quit-modal-overlay" onClick={onCancel}>
  <div className="quit-modal" onClick={(e) => e.stopPropagation()}>
```

**Escape key handler pattern** (`QRModal.tsx` lines 29-35):
```typescript
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') onCancel()
  }
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [onCancel])
```

**Focus on open pattern** (new; use `useRef` + `useEffect`):
```typescript
const cancelBtnRef = useRef<HTMLButtonElement>(null)
useEffect(() => {
  if (isOpen) cancelBtnRef.current?.focus()
}, [isOpen])
```

**Disabled-after-click pattern** (from `NewSessionModal.tsx` lines 54-56 `creating` state; `UI-SPEC States` table):
```typescript
const [acting, setActing] = useState(false)
// On button click: setActing(true) before calling onQuitGUI/onQuitAll
<button disabled={acting} style={acting ? { opacity: 0.5 } : undefined} ...>
```

**Session truncation pattern** (UI-SPEC decision — show first 5, then overflow line):
```typescript
const visible = sessions.slice(0, 5)
const overflow = sessions.length - 5
// render visible rows, then:
{overflow > 0 && <span className="quit-modal__overflow">...and {overflow} more</span>}
```

**BEM class structure** (from `85-UI-SPEC.md` Component Inventory — all required classes):
```
.quit-modal-overlay
  .quit-modal [role="dialog" aria-modal="true" aria-labelledby="quit-modal-title"]
    .quit-modal__header
      h2.quit-modal__title [id="quit-modal-title"]
      button.quit-modal__close [aria-label="Close"]
    .quit-modal__body
      p.quit-modal__subtitle
      .quit-modal__session-list (when sessions.length > 0)
        .quit-modal__session-item
          span.quit-modal__session-dot [aria-hidden="true"]
          span.quit-modal__session-name
          span.quit-modal__session-status
      p.quit-modal__no-sessions (when sessions.length === 0)
    .quit-modal__footer
      button.quit-modal__btn--cancel [ref={cancelBtnRef}]
      button.quit-modal__btn--quit-gui
      button.quit-modal__btn--quit-all
```

---

### `frontend/src/App.tsx` — add EventsOn subscription + modal render (component, event-driven)

**Analog:** `frontend/src/App.tsx` existing EventsOn block (lines 263-410) — add to same `useEffect([], [])` block

**State declarations pattern** (copy from existing `useState` declarations, e.g. lines ~100-120 in App.tsx):
```typescript
const [showQuitModal, setShowQuitModal] = useState(false)
const [quitSessions, setQuitSessions] = useState<Array<{ id: string; name: string; status: string }>>([])
```

**EventsOn subscription pattern** (`App.tsx` lines 263-268, 336-394 — exact shape to copy):
```typescript
const offQuit = EventsOn('app:quit-requested', () => {
  setShowQuitModal(prev => {
    if (prev) return prev  // D-pitfall-2: ignore if already showing
    ListSessions().then(sessions => {
      setQuitSessions(sessions)
      setShowQuitModal(true)
    }).catch(() => {
      setQuitSessions([])
      setShowQuitModal(true)
    })
    return prev
  })
})
```

**Cleanup return pattern** (`App.tsx` lines 396-410):
```typescript
return () => {
  offStatus()
  offHealth()
  offDaemonError()
  cancelTrayFocus()
  offExit()
  offQuit()   // ADD THIS LINE
  // ...existing cleanup...
}
```

**Import additions** (copy import pattern from `App.tsx` lines 8-29):
```typescript
import { QuitConfirmModal } from './components/QuitConfirmModal'
// Bound methods added after wails generate module:
// QuitGUIOnly, QuitAll added to existing import from './wailsjs/go/main/App'
```

**JSX render — alongside existing modal renders** (copy from existing `{showNewSessionModal && ...}` pattern in App.tsx JSX):
```typescript
{showQuitModal && (
  <QuitConfirmModal
    isOpen={showQuitModal}
    sessions={quitSessions}
    onQuitGUI={() => { setShowQuitModal(false); void QuitGUIOnly() }}
    onQuitAll={() => { setShowQuitModal(false); void QuitAll() }}
    onCancel={() => setShowQuitModal(false)}
  />
)}
```

---

### `app.go` — `beforeClose` refactor + `QuitGUIOnly` + `QuitAll` bound methods (service, request-response)

**Analog:** `app.go` — `beforeClose` (lines 166-177), `emitExitEvent` (lines 243-263), `ListSessions` (lines 266-282)

**beforeClose refactor pattern** (lines 166-177 — current implementation to replace):
```go
// Source: app.go lines 166-177 (current)
func (a *App) beforeClose(ctx context.Context) bool {
    if a.quitting {
        return false // allow quit — tray Quit was clicked
    }
    if ctx.Value("frontend") != nil {
        a.setDockVisible(false)
        runtime.WindowHide(ctx)
    }
    return true
}
// REFACTOR TO:
func (a *App) beforeClose(ctx context.Context) bool {
    if a.quitting {
        return false // QuitAll already set flag — allow quit
    }
    if ctx.Value("frontend") != nil {
        runtime.WindowShow(ctx)       // D-08: ensure window visible
        a.setDockVisible(true)
        runtime.EventsEmit(ctx, "app:quit-requested", nil)
    }
    return true // always prevent default quit — modal owns the decision
}
```

**ctx nil guard pattern** (`emitExitEvent` lines 243-246):
```go
func (a *App) emitExitEvent(...) {
    if a.ctx == nil {
        return
    }
    // ...
}
// COPY THIS GUARD into QuitGUIOnly and QuitAll:
func (a *App) QuitGUIOnly() {
    if a.ctx == nil {
        return
    }
    // ...
}
```

**EventsEmit pattern** (`app.go` lines 255-263 — `session:exit` emission):
```go
runtime.EventsEmit(a.ctx, "session:exit", map[string]any{...})
// For beforeClose refactor:
runtime.EventsEmit(ctx, "app:quit-requested", nil)
```

**QuitAll pattern** (mirrors existing `onTrayQuit` logic from `tray.go` lines 47-63):
```go
func (a *App) QuitAll() {
    if a.ctx == nil {
        return
    }
    if a.client != nil {
        _ = a.client.ShutdownDaemon()
    }
    a.quitting = true
    runtime.Quit(a.ctx)
}
```

**QuitGUIOnly pattern** (mirrors `onTrayShow` from `tray.go` lines 40-45 but inverted):
```go
func (a *App) QuitGUIOnly() {
    if a.ctx == nil {
        return
    }
    a.setDockVisible(false)
    runtime.WindowHide(a.ctx)
    // sendNotification call goes here (after notification_darwin.go is added)
}
```

---

### `tray.go` — `onTrayQuit` refactor (middleware, event-driven)

**Analog:** `tray.go` — `onTrayShow` (lines 40-45) — exact window-show-then-act pattern

**Current onTrayQuit** (lines 47-63 — to be replaced):
```go
//export onTrayQuit
func onTrayQuit() {
    app := trayCallbackApp
    go func() {
        if app != nil && app.client != nil {
            _ = app.client.ShutdownDaemon()
        }
        if app != nil {
            app.quitting = true
            if app.ctx != nil {
                runtime.Quit(app.ctx)
            }
        }
    }()
}
```

**onTrayShow pattern to copy** (lines 40-45):
```go
//export onTrayShow
func onTrayShow() {
    if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
        runtime.WindowShow(trayCallbackApp.ctx)
        trayCallbackApp.setDockVisible(true)
    }
}
```

**Refactored onTrayQuit** (D-06, D-07, D-08 — copies goroutine safety from current + show pattern from onTrayShow):
```go
//export onTrayQuit
func onTrayQuit() {
    app := trayCallbackApp
    go func() {
        if app != nil && app.ctx != nil {
            runtime.WindowShow(app.ctx)      // D-08: auto-show if hidden
            app.setDockVisible(true)
            runtime.EventsEmit(app.ctx, "app:quit-requested", nil)
        }
    }()
}
```

---

### `tray_objc_darwin.m` — add `sendNotification` C function (utility, request-response)

**Analog:** `tray_objc_darwin.m` — `updateTrayTooltip` (lines 116-123) and `setDockVisible` (lines 128-137) — both use `dispatch_async(dispatch_get_main_queue(), ^{...})` pattern with ObjC API

**dispatch_async pattern to copy** (`tray_objc_darwin.m` lines 116-123):
```objc
void updateTrayTooltip(const char *tooltip) {
    NSString *tip = [NSString stringWithUTF8String:tooltip];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            statusItem.button.toolTip = tip;
        }
    });
}
```

**C string → NSString pattern** (used in every existing function taking `const char *`):
```objc
NSString *nsTitle = [NSString stringWithUTF8String:title];
NSString *nsBody  = [NSString stringWithUTF8String:body];
```

**New sendNotification function** (add after existing functions, before the `@interface NSStatusItem` section):
```objc
#import <UserNotifications/UserNotifications.h>

void sendNotification(const char *title, const char *body) {
    NSString *nsTitle = [NSString stringWithUTF8String:title];
    NSString *nsBody  = [NSString stringWithUTF8String:body];
    dispatch_async(dispatch_get_main_queue(), ^{
        UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
        // Request auth lazily — first call only
        [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
            completionHandler:^(BOOL granted, NSError *error) {
            if (!granted) return;
            dispatch_async(dispatch_get_main_queue(), ^{
                UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
                content.title = nsTitle;
                content.body  = nsBody;
                UNNotificationRequest *req = [UNNotificationRequest
                    requestWithIdentifier:@"agenthub.quit-gui-only"
                    content:content
                    trigger:nil];
                [center addNotificationRequest:req withCompletionHandler:nil];
            });
        }];
    });
}
```

**C function declaration** — must also be added to the `tray.go` cgo header block alongside existing declarations:
```c
void sendNotification(const char *title, const char *body);
```

---

### `notification_darwin.go` — cgo wrapper for sendNotification (utility, request-response)

**Analog:** `tray.go` — cgo declaration block (lines 1-27) and `setDockVisible` Go wrapper (lines 84-90)

**Build tag + cgo header pattern** (`tray.go` lines 1-20):
```go
//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa -framework UserNotifications

#include <stdlib.h>

void sendNotification(const char *title, const char *body);
*/
import "C"

import "unsafe"
```

**LDFLAGS note:** `-framework UserNotifications` must be added. It can go in `notification_darwin.go`'s own cgo LDFLAGS or appended to `tray.go`'s existing line. Keeping it in `notification_darwin.go` avoids modifying tray.go's header.

**C string + free pattern** (`tray.go` lines 105-107 — `updateTrayTooltip` call pattern):
```go
ctip := C.CString(tip)
C.updateTrayTooltip(ctip)
C.free(unsafe.Pointer(ctip))
```

**Go wrapper function** (mirrors `setDockVisible` wrapper at lines 84-90):
```go
func sendNotification(title, body string) {
    ctitle := C.CString(title)
    cbody := C.CString(body)
    C.sendNotification(ctitle, cbody)
    C.free(unsafe.Pointer(ctitle))
    C.free(unsafe.Pointer(cbody))
}
```

---

### `frontend/src/components/__tests__/QuitConfirmModal.test.tsx` (test)

**Analog:** `frontend/src/components/__tests__/NewSessionModal.test.tsx` (source-inspection style, `?raw` import, `describe`/`it`/`expect().toContain`) and `frontend/src/components/__tests__/App.exit.test.tsx` (App.tsx wiring assertions)

**Source-inspection import pattern** (`NewSessionModal.test.tsx` lines 1-3):
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../QuitConfirmModal.tsx?raw'
```

**Test structure pattern** (`NewSessionModal.test.tsx` — grouped by requirement ID):
```typescript
describe('QuitConfirmModal source inspection', () => {
  describe('APP-01: modal structure', () => {
    it('uses quit-modal-overlay class', () => {
      expect(raw).toContain('quit-modal-overlay')
    })
    it('uses quit-modal class with stopPropagation', () => {
      expect(raw).toContain('stopPropagation')
    })
    it('handles Escape key via keydown listener', () => {
      expect(raw).toContain('Escape')
    })
  })
  describe('APP-02: exit buttons', () => {
    it('renders quit-modal__btn--quit-gui', () => {
      expect(raw).toContain('quit-modal__btn--quit-gui')
    })
    it('renders quit-modal__btn--quit-all', () => {
      expect(raw).toContain('quit-modal__btn--quit-all')
    })
    it('renders quit-modal__btn--cancel', () => {
      expect(raw).toContain('quit-modal__btn--cancel')
    })
  })
  describe('APP-03: session list', () => {
    it('accepts sessions prop', () => {
      expect(raw).toContain('sessions')
    })
    it('renders session name', () => {
      expect(raw).toContain('session-name')
    })
    it('renders session status', () => {
      expect(raw).toContain('session-status')
    })
  })
})
```

**App.tsx wiring test** (separate describe block, using `?raw` on App.tsx — copy `App.exit.test.tsx` pattern):
```typescript
import appRaw from '../../App.tsx?raw'

describe('App.tsx quit-modal wiring (Phase 85)', () => {
  it('subscribes to app:quit-requested event', () => {
    expect(appRaw).toContain("'app:quit-requested'")
  })
  it('defines showQuitModal state', () => {
    expect(appRaw).toContain('showQuitModal')
  })
  it('imports QuitConfirmModal component', () => {
    expect(appRaw).toContain("from './components/QuitConfirmModal'")
  })
  it('renders QuitConfirmModal in JSX', () => {
    expect(appRaw).toContain('<QuitConfirmModal')
  })
  it('unsubscribes via offQuit in cleanup', () => {
    expect(appRaw).toContain('offQuit')
  })
})
```

---

## Shared Patterns

### EventsEmit / EventsOn event bus
**Source:** `app.go` lines 255-263 (emit) + `frontend/src/App.tsx` lines 263-268 (subscribe)
**Apply to:** `beforeClose` refactor, `onTrayQuit` refactor, `App.tsx` new subscription
```go
// Go side — emit:
runtime.EventsEmit(a.ctx, "app:quit-requested", nil)
```
```typescript
// Frontend side — subscribe (inside [] useEffect):
const offQuit = EventsOn('app:quit-requested', () => { ... })
// cleanup:
return () => { offQuit() }
```

### ctx nil guard
**Source:** `app.go` lines 243-246 (`emitExitEvent`)
**Apply to:** `QuitGUIOnly`, `QuitAll`, and `beforeClose` (already has `quitting` guard + `ctx.Value("frontend")`)
```go
if a.ctx == nil {
    return
}
```

### cgo C string memory management
**Source:** `tray.go` lines 105-107
**Apply to:** `notification_darwin.go` `sendNotification` wrapper
```go
cstr := C.CString(goStr)
C.someFunc(cstr)
C.free(unsafe.Pointer(cstr))
```

### dispatch_async main queue (ObjC)
**Source:** `tray_objc_darwin.m` lines 116-123, 128-137
**Apply to:** `sendNotification` in `tray_objc_darwin.m`
```objc
dispatch_async(dispatch_get_main_queue(), ^{
    // all UI / AppKit / UserNotifications API calls here
});
```

### BEM modal CSS structure
**Source:** `frontend/src/style.css` lines 655-848 (`.new-session-modal__*` patterns)
**Apply to:** new `.quit-modal-*` classes in `frontend/src/style.css`

Key values to replicate exactly (D-05 visual consistency):
- Overlay: `rgba(0,0,0,0.6)` backdrop, `z-index: 1000`, `position: fixed`, `inset: 0`
- Modal panel: `background: #1e2030`, `border: 1px solid #292e42`, `border-radius: 8px`, `box-shadow: 0 8px 32px rgba(0,0,0,0.5)`, `width: 420px`, `max-width: 95vw`
- Header/footer: `padding: 12px 20px`, `border-bottom/top: 1px solid #292e42`
- Header h2: `font-size: 16px`, `font-weight: 600`, `color: #c0caf5`
- Body: `padding: 20px`
- Cancel button: ghost (`transparent` bg, `border: 1px solid #292e42`, `color: #9aa5ce`, `padding: 8px 16px`, `border-radius: 4px`)
- Quit GUI Only button: accent (`background: #7aa2f7`, `color: #1a1b26`, `font-weight: 600`, `padding: 8px 16px`)
- Quit Everything button: destructive (`background: #f7768e`, `color: #1a1b26`, `font-weight: 600`, `padding: 8px 16px`)
- Disabled state: `opacity: 0.5`, `cursor: not-allowed`

### Source-inspection test style
**Source:** `frontend/src/components/__tests__/NewSessionModal.test.tsx` + `App.exit.test.tsx`
**Apply to:** `QuitConfirmModal.test.tsx`
Pattern: `import raw from '../Component.tsx?raw'` + `expect(raw).toContain(...)` — no component mounting, no Wails runtime mock needed.

---

## No Analog Found

None — all 7 files have close or exact analogs in the codebase.

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub/` (Go root), `frontend/src/components/`, `frontend/src/components/__tests__/`
**Files scanned:** 12 source files read directly
**Pattern extraction date:** 2026-04-19

**Critical implementation notes for planner:**
1. `wails generate module` (or `wails build`) must run after adding `QuitGUIOnly`/`QuitAll` to `app.go` — `wailsjs/go/main/App.ts` is auto-generated and cannot be hand-edited (RESEARCH.md Pitfall 7).
2. `TestBeforeCloseReturnsTrue` and `TestHideWindowSessionsAlive` are existing regression tests that must continue to pass after the `beforeClose` refactor — the new `beforeClose` still returns `true` in all non-quitting paths.
3. `tray_objc_darwin.m` filename in repo is `tray_objc_darwin.m` (confirmed by Read) — not `tray_objc.m` as the file header comment says. Use the actual path.
4. `-framework UserNotifications` LDFLAGS must be added; placing it in `notification_darwin.go` avoids modifying `tray.go`'s existing cgo block.
