---
phase: 167-native-notifications
reviewed: 2026-07-01T13:59:34Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - app.go
  - frontend/src/components/SettingsTab.tsx
  - notification_darwin.go
  - notification_linux.go
  - notification_windows.go
  - tray_objc_darwin.m
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 167: Code Review Report

**Reviewed:** 2026-07-01T13:59:34Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Reviewed the Phase 167-06 / 167-07 gap-closure changes: the native macOS
proactive-authorization path (`tray_objc_darwin.m`, `notification_darwin.go`),
the cross-platform no-op stubs (`notification_linux.go`,
`notification_windows.go`), the `app.go` edge-detection / auth wiring, and the
`SettingsTab.tsx` `notification:permission-denied` subscription.

The native path is solid on the details that usually break CGO/UN work: the
`hasValidBundleIdentifier` guard is applied synchronously before every entry
point (`sendNotification`, `requestNotificationAuthorization`,
`registerNotificationDelegate`); the C strings are converted to `NSString`
*before* `dispatch_async`, so the Go-side `C.free` in
`notification_darwin.go:52-54` cannot cause a use-after-free; the
foreground-presentation delegate is correctly registered to stop macOS
swallowing banners for the frontmost app; and the completion-handler thread
hand-off in `onNotificationAuthResult` follows the existing `onTraySession`
pattern. `EventsOn` returns `() => void`, so the `return off` cleanup in the
new `useEffect` is correct.

One real defect stands out: the denied-permission hint the UI shows is sticky
and its own remediation text does not work. The remaining items are edge-case
observability gaps rather than crashes.

## Warnings

### WR-01: Permission-denied hint is sticky and its own remediation instructions never clear it

**File:** `frontend/src/components/SettingsTab.tsx:128,233,540-544` and `notification_darwin.go:67-80`
**Issue:** `notifyPermissionDenied` is only ever set to `true` (line 233); no
code path resets it to `false`. The hint text it renders (lines 541-543)
instructs the user to "Enable it in System Settings > Notifications >
AgentHub, then toggle this setting off and on again." Following those exact
instructions does not clear the message:

1. `handleToggleNotifyOnWaiting` (lines 390-402) never calls
   `setNotifyPermissionDenied(false)`, so toggling off then on leaves the
   error banner rendered.
2. Even if it did, the backend never emits a "granted" / "cleared" event.
   `onNotificationAuthResult` returns early on grant
   (`notification_darwin.go:70-72`) and only emits
   `notification:permission-denied` on denial. So after the user grants
   permission in System Settings and re-toggles, `requestNotificationAuth`
   fires, the OS reports granted, the callback returns silently, and the
   stale banner persists for the rest of the session.

The net effect: once the banner appears it is permanent for the app session,
and the guidance it gives is misleading. This degrades the exact
recover-from-denial UX the 167-07 change was added to provide.
**Fix:** Reset the flag at the start of the toggle handler and have the
backend signal success so the UI can self-heal. Frontend:
```tsx
async function handleToggleNotifyOnWaiting() {
  const next = !notifyOnWaiting
  setNotifyOnWaitingSaving(true)
  setNotifyOnWaitingError(null)
  if (next) setNotifyPermissionDenied(false) // clear stale hint before re-requesting
  ...
}
```
And subscribe to a granted signal so a successful re-auth clears the banner:
```tsx
useEffect(() => {
  const offDenied = EventsOn('notification:permission-denied', () => setNotifyPermissionDenied(true))
  const offGranted = EventsOn('notification:permission-granted', () => setNotifyPermissionDenied(false))
  return () => { offDenied(); offGranted() }
}, [])
```
Backend (`notification_darwin.go:70-72`): emit a granted event instead of a
bare `return` so the UI is told when permission is (re)acquired.

## Info

### IN-01: A denied result is dropped whenever SettingsTab is unmounted or `appInstance.ctx` is nil

**File:** `notification_darwin.go:75-79` and `frontend/src/components/SettingsTab.tsx:231-234`
**Issue:** The `notification:permission-denied` event is fire-and-forget. The
`EventsOn` subscription only exists while `SettingsTab` is mounted. The OS
authorization prompt is modal and the user can dismiss it after navigating
away, or the async result can arrive after unmount — in which case the denial
is never surfaced. Likewise, if `appInstance.ctx` is nil when the result
returns (`notification_darwin.go:76`), the event is silently discarded. In
practice the toggle can only be flipped from the Settings tab and `ctx` is set
before the UI can render, so this is an edge case rather than a live bug.
**Fix:** Persist the denied state on the backend (e.g. an atomic) and expose it
via a `GetNotificationPermissionDenied()` bound method the tab reads on mount,
so the hint survives navigation instead of relying solely on a live event.

### IN-02: Denial discovered at notification-send time is never surfaced to the frontend

**File:** `tray_objc_darwin.m:269-272`
**Issue:** `sendNotification`'s auth completion handler logs
`authorization NOT granted` and returns, but (unlike
`requestNotificationAuthorization`) does not call `onNotificationAuthResult`.
If the user revokes permission in System Settings after enabling the toggle,
subsequent waiting notifications silently no-op with no UI feedback — the
Settings hint only appears on the toggle-time path. Design intent is that the
prompt happens at toggle time, so this is a known gap, but worth recording.
**Fix:** Route both completion handlers through `onNotificationAuthResult` so a
send-time denial re-raises the same UI hint.

### IN-03: `onNotificationAuthResult` reads the `appInstance` global from the UN completion-handler thread without synchronization

**File:** `notification_darwin.go:75-79`
**Issue:** The exported callback runs on the UserNotifications completion
thread and reads the package global `appInstance` (and `appInstance.ctx`),
which is written on the Wails startup goroutine (`app.go:222`). This is an
unsynchronized cross-goroutine access that Go's `-race` detector would flag.
It is benign in practice (set once, early) and mirrors the established
`onTraySession`/`appInstance` pattern already in the codebase, so it is noted
for completeness rather than as a new defect.
**Fix:** If tightening later, publish the context via an `atomic.Pointer` or
guard `appInstance`/`ctx` access behind the existing mutex used elsewhere.

---

_Reviewed: 2026-07-01T13:59:34Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
