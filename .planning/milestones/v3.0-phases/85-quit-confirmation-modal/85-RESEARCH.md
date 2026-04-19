# Phase 85: Quit Confirmation Modal - Research

**Researched:** 2026-04-19
**Domain:** Wails v2 event system, macOS Objective-C notifications, React modal patterns (BEM/custom CSS)
**Confidence:** HIGH

## Summary

Phase 85 intercepts all quit actions in a Wails v2 macOS desktop app and routes them through a
React confirmation modal. The work spans three tiers: Go backend (event emission, new bound
methods, optional macOS notification via ObjC cgo), Wails event bus (Go → frontend event
bridging), and React frontend (new QuitConfirmModal component wired into App.tsx).

The codebase already has all primitives needed: `runtime.EventsEmit` for Go→frontend events,
`EventsOn` subscriptions in App.tsx, `beforeClose` for window-close interception, `onTrayQuit`
for tray interception, `ListSessions` for session data, and two existing modal components
(NewSessionModal, QRModal) whose patterns define the exact visual and structural contract. The
UI-SPEC (85-UI-SPEC.md) is fully approved and specifies every CSS class, color, and copy string —
the planner should treat that document as locked implementation detail.

The only open technical question is the OS notification implementation approach. Wails v2.10.2
does NOT ship a first-party notification API. The established cgo pattern in `tray_objc_darwin.m`
(which already imports `<Cocoa/Cocoa.h>` and uses `dispatch_async`) is the natural extension
point: add a `sendNotification` C function alongside the existing tray ObjC code using
`NSUserNotificationCenter` (macOS 10.14+) or `UNUserNotificationCenter` (10.14+, preferred).
This is fully consistent with how the project already does native macOS work.

**Primary recommendation:** Implement in 4 discrete units — (1) `QuitConfirmModal` React
component, (2) new Go bound methods (`QuitGUIOnly`, `QuitAll`), (3) `beforeClose` + `onTrayQuit`
refactor to emit `app:quit-requested`, (4) macOS notification via new ObjC helper.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Always show the quit confirmation modal regardless of session count — even when 0 sessions are active
- **D-02:** When session count is 0, display an adjusted message ("No active sessions") instead of a count. The two exit buttons remain available since daemon state is still relevant.
- **D-03:** Informational tone with session context — modal lists each active session by name and current status (running/idle/waiting/errored)
- **D-04:** "Quit Everything" button styled with destructive red accent to differentiate from "Quit GUI Only" (the safe default)
- **D-05:** Match existing modal patterns (dark overlay, Escape to close, Cancel button returns to app)
- **D-06:** Tray Quit emits a Wails event (e.g., `app:quit-requested`) instead of calling ShutdownDaemon + runtime.Quit directly
- **D-07:** Frontend listens for the quit-requested event and shows the modal
- **D-08:** When the quit event arrives and the window is hidden, auto-show the window so the modal is visible to the user
- **D-09:** User's modal choice calls back to Go to execute the selected quit behavior
- **D-10:** "Quit GUI Only" hides the window to tray (same as current close behavior) — daemon and sessions keep running
- **D-11:** After hiding, send an OS-level notification (macOS Notification Center) confirming "AgentHub is still running in the background. N sessions active."
- **D-12:** Window close (red button) triggers the same quit confirmation modal instead of silently hiding — consistent with tray Quit behavior

### Claude's Discretion
- Wails event naming convention for the quit-requested event
- Modal component structure (new component vs. extending existing patterns)
- OS notification implementation approach (Wails notification API or Go-native)
- Session list rendering within the modal (truncation strategy if many sessions)
- Cancel button placement and keyboard shortcut handling
- Animation/transition for modal appearance

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| APP-01 | Quit action (GUI close / tray Quit) shows a confirmation modal | `beforeClose` returns `true` to block quit; `onTrayQuit` can emit event instead of directly quitting. Both interception points identified and understood. |
| APP-02 | Modal offers two choices: quit GUI only (daemon stays running) or quit both GUI and daemon | New bound methods `QuitGUIOnly` (calls `runtime.WindowHide` + `setDockVisible(false)` + notification) and `QuitAll` (calls `ShutdownDaemon` + sets `quitting=true` + `runtime.Quit`) are the correct pattern. |
| APP-03 | Modal displays count of currently active sessions as context for the decision | `ListSessions()` already returns `[]SessionInfo` with `Name` and `Status` fields. Frontend calls this on modal open; data flows into `QuitConfirmModal` via props. |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Intercept window close button | Go / Wails OnBeforeClose | — | `beforeClose` callback is the only hook Wails provides for the OS-level close signal |
| Intercept tray Quit menu item | Go / cgo ObjC callback | — | `onTrayQuit` is a cgo-exported function called from ObjC `quitClicked:` |
| Emit quit-requested event | Go (runtime.EventsEmit) | — | Event must originate in Go since both intercept points are Go-side |
| Auto-show window when hidden | Go (runtime.WindowShow + setDockVisible) | — | Window visibility is owned by Go; frontend cannot show the Wails window directly |
| Quit confirmation modal UI | Frontend / React | — | Visual decision dialog belongs in the React layer per existing pattern (NewSessionModal, QRModal) |
| Session list data for modal | Frontend calls Go (ListSessions) | — | Data fetch happens at modal-open time on the frontend; Go owns the source of truth |
| Execute quit-GUI-only | Go bound method (QuitGUIOnly) | — | Needs to call runtime.WindowHide, setDockVisible(false), send OS notification |
| Execute quit-everything | Go bound method (QuitAll) | — | Must call ShutdownDaemon, set quitting=true, then runtime.Quit |
| OS notification (D-11) | Go / cgo ObjC | — | Wails v2 has no notification API; project already uses ObjC cgo in tray_objc_darwin.m |

---

## Standard Stack

### Core (already in project — no new dependencies needed)
[VERIFIED: codebase grep]

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Wails v2 | v2.10.2 | Desktop app shell, event bus, bound methods | Already the project runtime |
| React + TypeScript | (from package.json) | Frontend modal component | Existing stack |
| Vitest + jsdom | (from vitest.config.ts) | Frontend unit tests | Existing test setup |
| cgo + Cocoa | (darwin-native) | ObjC notification in tray_objc_darwin.m | Existing pattern for macOS native |

### No New Dependencies
This phase requires zero new npm packages and zero new Go modules. All primitives are already in place.

**Installation:** none required.

---

## Architecture Patterns

### System Architecture Diagram

```
[User action]
     |
     +-- Window close button (macOS red button)
     |        |
     |        v
     |   beforeClose(ctx) [app.go]
     |        |
     |        +-- currently: hides window
     |        +-- NEW: emit "app:quit-requested" event + window show if hidden
     |
     +-- Tray menu "Quit"
              |
              v
         onTrayQuit() [tray.go cgo export]
              |
              +-- currently: ShutdownDaemon + quitting=true + runtime.Quit
              +-- NEW: emit "app:quit-requested" event + window show if hidden

                    |
                    v (Wails event bus Go → frontend)
             EventsOn('app:quit-requested') [App.tsx]
                    |
                    v
        setShowQuitModal(true) + fetch ListSessions()
                    |
                    v
            <QuitConfirmModal>
           /         |          \
    onCancel    onQuitGUI     onQuitAll
       |             |             |
   (no-op)    QuitGUIOnly()   QuitAll()
              [Go bound]     [Go bound]
                   |               |
            WindowHide +    ShutdownDaemon +
            setDock(false)+ quitting=true +
            sendNotif()    runtime.Quit()
```

### Recommended File Changes / New Files

```
agenthub/
├── app.go                              # modify beforeClose + add QuitGUIOnly/QuitAll methods
├── tray.go (darwin)                    # modify onTrayQuit — emit event, don't directly quit
├── tray_objc_darwin.m                  # add sendNotification() C function
├── notification_darwin.go              # (optional) cgo wrapper for sendNotification
└── frontend/src/
    ├── App.tsx                         # add EventsOn('app:quit-requested') + showQuitModal state
    ├── components/
    │   └── QuitConfirmModal.tsx        # NEW component
    └── index.css / style.css           # add .quit-modal-* CSS (BEM classes from UI-SPEC)
```

### Pattern 1: Emitting a Wails Event from Go
**What:** Go calls `runtime.EventsEmit` to push data to the frontend's event subscription.
**When to use:** Any time a Go-side event (user action, system state change) needs to trigger frontend UI.

```go
// Source: app.go lines 255 (existing session:exit pattern)
runtime.EventsEmit(a.ctx, "app:quit-requested", nil)
```

The frontend registers with `EventsOn` in the `useEffect([], [])` block in App.tsx (same pattern as `session:status`, `session:exit`, `tray:focus-session`).

### Pattern 2: Bound Go Method Called from Frontend
**What:** A Go method on `*App` annotated with no special tag is automatically available in
`wailsjs/go/main/App.ts` after `wails generate module`.
**When to use:** Frontend needs to trigger a Go-side action with a return value or error.

```go
// Source: pattern from existing ListSessions, CreateSession, etc.
// New methods to add:
func (a *App) QuitGUIOnly() {
    // hide window, set dock invisible, send notification
}

func (a *App) QuitAll() {
    if a.client != nil {
        _ = a.client.ShutdownDaemon()
    }
    a.quitting = true
    runtime.Quit(a.ctx)
}
```

After adding these methods, run `wails generate module` (or build) to regenerate the TypeScript bindings.

### Pattern 3: beforeClose Refactor (D-12)
**What:** `beforeClose` currently silently hides the window. Under D-12, it must instead emit `app:quit-requested` (which brings up the modal), and always return `true` to prevent Wails from quitting.

```go
// Source: app.go lines 162-177 (current implementation)
func (a *App) beforeClose(ctx context.Context) bool {
    if a.quitting {
        return false // QuitAll already set this flag — allow quit
    }
    // NEW: emit event for modal instead of silently hiding
    if ctx.Value("frontend") != nil {
        runtime.WindowShow(ctx)       // ensure window visible (D-08)
        a.setDockVisible(true)
        runtime.EventsEmit(ctx, "app:quit-requested", nil)
    }
    return true // always prevent default quit — modal handles next step
}
```

**Critical invariant:** `beforeClose` must return `true` in all non-quitting paths. The existing test `TestBeforeCloseReturnsTrue` validates this — it must continue to pass after the refactor.

### Pattern 4: onTrayQuit Refactor (D-06)
**What:** `onTrayQuit` currently calls ShutdownDaemon + runtime.Quit directly. Under D-06 it must emit `app:quit-requested` and show the window.

```go
// Source: tray.go lines 47-63 (current implementation)
//export onTrayQuit
func onTrayQuit() {
    app := trayCallbackApp
    go func() {
        if app != nil && app.ctx != nil {
            runtime.WindowShow(app.ctx)    // D-08: auto-show if hidden
            app.setDockVisible(true)
            runtime.EventsEmit(app.ctx, "app:quit-requested", nil)
        }
    }()
}
```

### Pattern 5: macOS Notification via Existing cgo Pattern
**What:** Add a `sendNotification` C function to `tray_objc_darwin.m` using `UNUserNotificationCenter`. Called from a new `notification_darwin.go` file (same build tag as tray.go).
**When to use:** After "Quit GUI Only" completes (D-11).

```objc
// To add to tray_objc_darwin.m (alongside existing functions)
#import <UserNotifications/UserNotifications.h>

void sendNotification(const char *title, const char *body) {
    NSString *nsTitle = [NSString stringWithUTF8String:title];
    NSString *nsBody  = [NSString stringWithUTF8String:body];
    dispatch_async(dispatch_get_main_queue(), ^{
        UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
        UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
        content.title = nsTitle;
        content.body  = nsBody;
        UNNotificationRequest *req = [UNNotificationRequest
            requestWithIdentifier:@"agenthub.quit-gui-only"
            content:content
            trigger:nil];
        [center addNotificationRequest:req withCompletionHandler:nil];
    });
}
```

**Permission requirement:** UNUserNotificationCenter requires the user to grant notification
permission on first use. The entitlement `com.apple.developer.usernotifications.critical-alerts`
is NOT needed for standard notifications. However, the app must request authorization if it hasn't
already. For a clean first-use experience, request permission lazily in `QuitGUIOnly` (only when
the user first selects that option). [ASSUMED: project does not already request notification
permission — verify in entitlements / Info.plist]

**Alternative (simpler, no permission needed):** NSUserNotificationCenter was the legacy API
(deprecated macOS 11+) and works without requesting permission for local notifications in many
app configurations. If the notification entitlement is problematic, the notification step could
be skipped or deferred as a post-v3.0 enhancement. This is flagged for planner discretion.

**LDFLAGS addition needed:** `tray.go` already has `-framework Cocoa`. UNUserNotificationCenter
requires also linking `-framework UserNotifications`:
```go
//#cgo darwin LDFLAGS: -framework Cocoa -framework UserNotifications
```
[VERIFIED: codebase — tray.go line 7 shows current LDFLAGS; UserNotifications framework must be added]

### Pattern 6: Frontend Modal Integration (App.tsx)
**What:** App.tsx subscribes to `app:quit-requested` in the existing `useEffect([], [])` block, sets `showQuitModal` state, and renders `<QuitConfirmModal>`.

```typescript
// Source: App.tsx lines 263-410 (existing EventsOn subscription block)
const [showQuitModal, setShowQuitModal] = useState(false)
const [quitSessions, setQuitSessions] = useState<SessionInfo[]>([])

// Inside the [] useEffect:
const offQuit = EventsOn('app:quit-requested', () => {
    ListSessions().then(sessions => {
        setQuitSessions(sessions)
        setShowQuitModal(true)
    }).catch(() => {
        setQuitSessions([])
        setShowQuitModal(true)
    })
})
// ... add offQuit() to cleanup return

// In JSX, alongside existing modal renders:
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

### Pattern 7: QuitConfirmModal Component Structure
**What:** New component following the exact BEM class structure from 85-UI-SPEC.md.
**Key structural notes:**
- Props: `{ isOpen, sessions, onQuitGUI, onQuitAll, onCancel }` (from UI-SPEC)
- Overlay: `div.quit-modal-overlay` with onClick→onCancel and stopPropagation on inner panel
- Escape key: `useEffect` with `keydown` listener (pattern from QRModal)
- Focus: On open, focus the "Keep Running" button (safe default per UI-SPEC interaction contract)
- Button disabled state: after clicking Quit GUI Only or Quit Everything, disable both buttons (opacity 0.5) to prevent double-fire (UI-SPEC States table)
- Session truncation: show first 5, then "...and N more" if count > 5 (UI-SPEC decision)

### Anti-Patterns to Avoid
- **Calling ShutdownDaemon + runtime.Quit inside `beforeClose`:** `beforeClose` is a synchronous callback; calling `runtime.Quit` from inside it causes undefined Wails behavior. Execute quit actions from bound methods called by the frontend instead. [VERIFIED: existing `onTrayQuit` already uses goroutine to avoid this]
- **Setting `quitting = true` without calling `runtime.Quit`:** If `QuitAll` sets `quitting` but something prevents `runtime.Quit` from executing, the next window-close will fall through `beforeClose` and immediately exit — bypassing any state cleanup. The flag and the Quit call must be atomic from the user's perspective.
- **Forgetting `stopPropagation` on modal panel:** Overlay click-to-dismiss must not fire when clicking inside the modal body. Both existing modals demonstrate this pattern.
- **Generating wailsjs bindings manually:** New Go bound methods on `*App` are auto-generated by `wails generate module` or `wails build`. The planner must include a task to run this command (or confirm it runs as part of `wails build`). [VERIFIED: wailsjs/go/main/App.ts is auto-generated — do not hand-edit]
- **Not handling the case where ctx is nil in new Go methods:** Both `QuitGUIOnly` and `QuitAll` should guard against `a.ctx == nil` (same defensive pattern as `emitExitEvent`).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Event bus Go→frontend | Custom WebSocket or polling | `runtime.EventsEmit` / `EventsOn` | Already in every event subscription in the codebase |
| Modal overlay + Escape | New pattern | Copy QRModal/NewSessionModal patterns verbatim | Deviating breaks visual consistency (D-05) |
| Session list data | Separate IPC endpoint | `ListSessions()` Go bound method | Already exists, already returns Name+Status+State |
| macOS notification | shell out to `osascript` | cgo `UNUserNotificationCenter` in tray_objc_darwin.m | Project already uses cgo for macOS native; no subprocess overhead |
| TypeScript bindings | Manually write Go method signatures | `wails generate module` | Auto-generation prevents drift between Go and TypeScript types |

**Key insight:** Every mechanism this phase needs already exists in the codebase. The work is
wiring, not invention.

---

## Common Pitfalls

### Pitfall 1: beforeClose Called Before Frontend Is Ready
**What goes wrong:** If the user closes the window immediately after launch (before `domReady`), `runtime.EventsEmit` is called but the frontend EventsOn listener hasn't registered yet — the event fires into a void and no modal appears.
**Why it happens:** Wails calls `beforeClose` as soon as the OS close signal arrives, which can precede frontend initialization.
**How to avoid:** Guard with a `ctx.Value("frontend") != nil` check (already present in current `beforeClose`). Since `domReady` calls `runtime.WindowShow`, the window is only visible after the frontend is ready — the user can't click the close button before the frontend registers its listeners.
**Warning signs:** Modal never appears during rapid launch-then-close cycles.

### Pitfall 2: Double-Fire on Rapid Close Attempts
**What goes wrong:** User clicks the close button twice quickly, causing two `app:quit-requested` events. The second event opens a second modal (or re-opens after cancel).
**Why it happens:** Each `beforeClose` call emits an independent event.
**How to avoid:** Gate in App.tsx: only emit/show modal if `showQuitModal` is already `false`. The `EventsOn` callback checks current state via the setter pattern: `setShowQuitModal(prev => { if (!prev) { ...; return true } return prev })`.

### Pitfall 3: runtime.Quit Called Inside beforeClose
**What goes wrong:** Wails undefined behavior — `runtime.Quit` triggers another `beforeClose` call recursively.
**Why it happens:** Developers may think `beforeClose` is the right place to call Quit after confirming.
**How to avoid:** `runtime.Quit` must only be called from a bound method (`QuitAll`) invoked by the frontend, not from within `beforeClose`.

### Pitfall 4: Tray-Initiated Quit When Window Already Hidden
**What goes wrong:** User hides window to tray, then selects Quit from tray. `app:quit-requested` fires but window is hidden — modal is invisible.
**Why it happens:** The window stays hidden until explicitly shown.
**How to avoid:** In the refactored `onTrayQuit` (and in `beforeClose` for the tray path), call `runtime.WindowShow` + `setDockVisible(true)` before emitting the event (D-08). The existing `onTrayShow` function demonstrates this exact sequence.

### Pitfall 5: UserNotifications Authorization Not Requested
**What goes wrong:** Notification silently fails on first "Quit GUI Only" click because the app hasn't requested notification permission.
**Why it happens:** `UNUserNotificationCenter` requires explicit authorization request before delivering notifications.
**How to avoid:** Call `[center requestAuthorizationWithOptions:...]` once before adding the notification request. Request lazily (first time user selects "Quit GUI Only") or during app startup. Handle the async authorization result gracefully — if denied, skip the notification silently.
**Warning signs:** No notification appears on first "Quit GUI Only" click; subsequent clicks work after permission granted via Settings.

### Pitfall 6: Missing -framework UserNotifications in LDFLAGS
**What goes wrong:** Build error: `ld: framework not found UserNotifications`.
**Why it happens:** tray.go LDFLAGS only includes `-framework Cocoa` currently.
**How to avoid:** Add `-framework UserNotifications` to the `#cgo darwin LDFLAGS` line in `tray.go` (or in a new `notification_darwin.go`).

### Pitfall 7: Wails Bindings Not Regenerated
**What goes wrong:** TypeScript still shows old `QuitGUIOnly`/`QuitAll` as undefined.
**Why it happens:** `wailsjs/go/main/App.ts` is auto-generated and not updated until `wails generate module` or `wails build` runs.
**How to avoid:** Include regeneration as an explicit task step. The Go implementation must compile successfully before bindings generate.

---

## Code Examples

### Verified: EventsOn subscription teardown pattern
```typescript
// Source: App.tsx lines 396-410 — existing cleanup pattern
return () => {
    offStatus()
    offHealth()
    offDaemonError()
    cancelTrayFocus()
    offExit()
    // ... add offQuit() here
}
```

### Verified: QRModal Escape key handler
```typescript
// Source: frontend/src/components/QRModal.tsx lines 29-35
useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
        if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
}, [onClose])
```

### Verified: NewSessionModal overlay + stopPropagation
```typescript
// Source: frontend/src/components/NewSessionModal.tsx lines 66-67
<div className="new-session-overlay" onClick={onClose}>
    <div className="new-session-modal" onClick={(e) => e.stopPropagation()}>
```

### Verified: runtime.EventsEmit call (session:exit pattern)
```go
// Source: app.go lines 255-263
runtime.EventsEmit(a.ctx, "session:exit", map[string]any{
    "sessionId": sessionID,
    // ...
})
```

### Verified: beforeClose current implementation
```go
// Source: app.go lines 166-177
func (a *App) beforeClose(ctx context.Context) bool {
    if a.quitting {
        return false // allow quit
    }
    if ctx.Value("frontend") != nil {
        a.setDockVisible(false)
        runtime.WindowHide(ctx)
    }
    return true
}
```

### Verified: onTrayShow pattern (window auto-show on tray action)
```go
// Source: tray.go lines 40-45
func onTrayShow() {
    if trayCallbackApp != nil && trayCallbackApp.ctx != nil {
        runtime.WindowShow(trayCallbackApp.ctx)
        trayCallbackApp.setDockVisible(true)
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| NSUserNotificationCenter | UNUserNotificationCenter (macOS 10.14+) | Required for future macOS compatibility; requires permission request |
| Wails v2 OnBeforeClose returns false to allow quit | Returns true to prevent quit, emits event for modal | Architectural shift: modal owns the quit decision |

**Deprecated/outdated:**
- `NSUserNotificationCenter`: Deprecated macOS 11+. Use `UNUserNotificationCenter` from UserNotifications.framework. [ASSUMED: based on Apple documentation knowledge — verify against macOS deployment target in project]

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Project does not already request UNUserNotificationCenter authorization | Architecture Patterns - Pattern 5 | If authorization was already requested elsewhere, the implementation approach is still correct but the lazy-request logic in QuitGUIOnly is redundant (harmless) |
| A2 | macOS deployment target supports UNUserNotificationCenter (macOS 10.14+) | Architecture Patterns - Pattern 5 | If min target is < 10.14, need to use NSUserNotificationCenter or skip notification entirely |
| A3 | wails generate module is a separate step from wails build (or build triggers it) | Don't Hand-Roll | If bindings are generated during build, a separate generate step is unnecessary |

**All other claims in this research are VERIFIED via codebase inspection.**

---

## Open Questions

1. **UNUserNotificationCenter authorization flow**
   - What we know: First notification attempt silently fails without prior authorization
   - What's unclear: Best time to request (startup vs. first Quit GUI Only click)
   - Recommendation: Request lazily on first "Quit GUI Only" invocation. If denied, skip notification silently. This is zero-friction for users who never use "Quit GUI Only" and surfaces permission UI only when relevant.

2. **D-11 notification on macOS App Store vs. direct distribution**
   - What we know: Project uses macOS code signing (MEMORY.md) — has a .p12 cert
   - What's unclear: Whether the app targets App Store (requires stricter entitlements) or direct distribution
   - Recommendation: For direct distribution, `UNUserNotificationCenter` works with a `com.apple.security.app-sandbox` entitlement + `com.apple.security.user-notification.alerts` (if sandboxed). If not sandboxed, no special entitlement needed. [ASSUMED]

---

## Environment Availability

All implementation is code-only changes to existing files plus one new `.tsx` component and one
new `.go` file. No external tools, databases, or services are required beyond the existing
build toolchain.

Step 2.6: SKIPPED (no new external dependencies — changes are pure code/ObjC additions to existing build targets)

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest + jsdom (frontend); Go testing (backend) |
| Config file | `frontend/vitest.config.ts` |
| Quick run command | `cd frontend && pnpm test --run` |
| Full suite command | `cd frontend && pnpm test --run && cd .. && go test -tags wailsassets ./... -run 'TestBeforeClose\|TestTrayQuit\|TestHideWindow'` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| APP-01 | Window close and tray Quit both show modal (emit event) | unit (source inspection) | `cd frontend && pnpm test --run -- QuitConfirmModal` | ❌ Wave 0 |
| APP-01 | `beforeClose` emits `app:quit-requested` event instead of hiding | unit (Go) | `go test -tags wailsassets -run TestBeforeCloseEmitsEvent ./...` | ❌ Wave 0 |
| APP-02 | Modal renders "Quit GUI Only" and "Quit Everything" buttons | unit (source inspection) | `cd frontend && pnpm test --run -- QuitConfirmModal` | ❌ Wave 0 |
| APP-02 | `QuitAll` calls ShutdownDaemon + sets quitting + calls Quit | unit (Go) | `go test -tags wailsassets -run TestQuitAll ./...` | ❌ Wave 0 |
| APP-03 | Modal receives sessions prop with name + status | unit (source inspection) | `cd frontend && pnpm test --run -- QuitConfirmModal` | ❌ Wave 0 |
| APP-01 (regression) | `beforeClose` still returns true (no quit bypassed) | unit (Go, existing) | `go test -tags wailsassets -run TestBeforeCloseReturnsTrue ./...` | ✅ exists |
| APP-01 (regression) | `TestHideWindowSessionsAlive` sessions survive beforeClose | unit (Go, existing) | `go test -tags wailsassets -run TestHideWindowSessionsAlive ./...` | ✅ exists |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test --run`
- **Per wave merge:** Full suite (frontend + Go)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/QuitConfirmModal.test.tsx` — covers APP-01, APP-02, APP-03 (source inspection style, matching existing test patterns)
- [ ] Go test additions to `tray_test.go`: `TestBeforeCloseEmitsEvent`, `TestQuitAll`
- [ ] Note: existing `TestBeforeCloseReturnsTrue` and `TestHideWindowSessionsAlive` must continue to pass — they are regression guards for the beforeClose refactor

---

## Security Domain

This phase adds no authentication, session management, cryptography, or network endpoints.
The only user-data-touching operation is reading the session list (already exposed) and
sending a local OS notification (no network transmission, no secrets).

ASVS V5 (Input Validation) does not apply — no user input is accepted in this flow; the
modal only offers button choices. No security domain research required.

---

## Sources

### Primary (HIGH confidence — codebase verified)
- `/Users/ken/dev/agenthub/app.go` lines 55-64, 162-177, 265-288 — App struct, beforeClose, ListSessions
- `/Users/ken/dev/agenthub/tray.go` lines 40-63 — onTrayShow, onTrayQuit patterns
- `/Users/ken/dev/agenthub/tray_objc_darwin.m` — Full ObjC cgo implementation reference
- `/Users/ken/dev/agenthub/main.go` lines 62-86 — Wails app options (HideWindowOnClose, OnBeforeClose)
- `/Users/ken/dev/agenthub/frontend/src/components/NewSessionModal.tsx` — Modal overlay + stopPropagation pattern
- `/Users/ken/dev/agenthub/frontend/src/components/QRModal.tsx` — Escape key handler pattern
- `/Users/ken/dev/agenthub/frontend/src/App.tsx` lines 263-410 — EventsOn subscriptions and cleanup
- `/Users/ken/dev/agenthub/.planning/phases/85-quit-confirmation-modal/85-UI-SPEC.md` — Full visual/interaction contract
- `/Users/ken/dev/agenthub/internal/daemon/types.go` — SessionInfo struct fields

### Secondary (MEDIUM confidence)
- Wails v2 documentation patterns (runtime.EventsEmit, EventsOn, bound methods) — consistent with codebase usage [ASSUMED based on training knowledge consistent with v2.10.2 observed in go.mod]

### Tertiary (LOW confidence)
- UNUserNotificationCenter permission requirement and LDFLAGS — [ASSUMED from Apple platform knowledge, not verified against actual project entitlements]

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, all libraries already in use
- Architecture: HIGH — patterns directly extracted from existing code files
- Frontend component: HIGH — UI-SPEC fully specifies every BEM class, color, and interaction
- OS notification: MEDIUM — approach is correct but entitlement/permission details need verification
- Pitfalls: HIGH — derived from direct code inspection of the exact files being modified

**Research date:** 2026-04-19
**Valid until:** 2026-05-19 (stable codebase — no fast-moving dependencies)
