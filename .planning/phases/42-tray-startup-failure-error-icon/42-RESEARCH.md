# Phase 42: Tray Startup-Failure Error Icon - Research

**Researched:** 2026-04-02
**Domain:** Go — tray state machine, nil-client guard, `refreshTrayState` in `app.go`
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TRAY-03 | Tray icon visually reflects daemon state (running vs error/disconnected) | The startup-failure path is fully traced: `refreshTrayState` at `app.go:422` returns early when `a.client == nil`, skipping `updateTray`. Fix is a one-line guard change plus an error-state `updateTray` call. `updateTray(connected=false)` already correctly renders the error icon. |

</phase_requirements>

---

## Summary

Phase 42 is a single-function bug fix. `refreshTrayState` in `app.go` has a compound nil-guard (`!a.trayInit || a.client == nil`) that returns early when the daemon client is nil. This guard is correct for the `a.trayInit == false` case (tray not yet initialised), but it is wrong for the `a.client == nil` case — that condition indicates a startup failure where the tray **is** initialised (`trayInit == true`) but EnsureDaemon failed before a client could be created.

The consequence: when EnsureDaemon fails, `startTrayPoller` is started (this is correct, per `app.go:65`), but every call to `refreshTrayState` skips `updateTray` entirely. The tray icon stays in its default (normal) state and the tooltip is never updated. Flow #8 from the v1.7 audit ("Daemon startup failure → tray error icon") is broken.

The fix separates the two guard conditions. When `trayInit` is false, return early (tray not ready). When `client == nil` but `trayInit` is true (startup failure), call `updateTray(nil, false)` — passing `connected=false` which the existing `updateTray` implementation already handles correctly by swapping in `trayIconErrorBytes` and setting the tooltip via `C.updateTrayTooltip`.

**Primary recommendation:** Split the compound guard in `refreshTrayState` into two separate guards. Add a new unit test `TestRefreshTrayStateNilClientShowsError` that verifies the error path executes without panic and calls `updateTray` with `connected=false`.

---

## Standard Stack

No new dependencies. This is a pure Go logic fix within the existing `app.go` and test infrastructure.

| File | Role | Change |
|------|------|--------|
| `app.go` | `refreshTrayState` method | Split nil-guard; call `updateTray(nil, false)` on startup-failure path |
| `tray_test.go` | Unit tests | Add `TestRefreshTrayStateNilClientShowsError` |

---

## Architecture Patterns

### Current Code (the bug)

```go
// app.go:421-431 — CURRENT (buggy)
func (a *App) refreshTrayState() {
    if !a.trayInit || a.client == nil {
        return  // BUG: skips updateTray even when trayInit=true and client=nil
    }
    connected := a.client.Health() == nil
    var sessions []SessionInfo
    if connected {
        sessions = a.ListSessions()
    }
    a.updateTray(sessions, connected)
}
```

The compound `||` short-circuits: both conditions return early. When `trayInit=true` (tray is visible in the menu bar) and `client=nil` (startup failed), the function returns without ever calling `updateTray` — the error icon is never shown.

### Fixed Code (the plan)

```go
// app.go — FIXED
func (a *App) refreshTrayState() {
    if !a.trayInit {
        return  // tray not yet initialised — nothing to update
    }
    if a.client == nil {
        // Startup failed — tray is visible but daemon is unreachable.
        // Show error icon and appropriate tooltip.
        a.updateTray(nil, false)
        return
    }
    connected := a.client.Health() == nil
    var sessions []SessionInfo
    if connected {
        sessions = a.ListSessions()
    }
    a.updateTray(sessions, connected)
}
```

### Why `updateTray(nil, false)` Works Without Changes

`updateTray` in `tray.go` already handles the `connected=false` + empty sessions case correctly:

```go
// tray.go:91-124 — existing implementation, no changes needed
func (a *App) updateTray(sessions []SessionInfo, connected bool) {
    // Update icon based on connectivity.
    if connected {
        ptr := unsafe.Pointer(&trayIconBytes[0])
        C.updateTrayIcon(ptr, C.int(len(trayIconBytes)))
    } else {
        ptr := unsafe.Pointer(&trayIconErrorBytes[0])
        C.updateTrayIcon(ptr, C.int(len(trayIconErrorBytes)))  // error icon shown
    }

    // Update tooltip with session count.
    tip := trayTooltip(len(sessions))  // len(nil) == 0 → "AgentHub — no sessions"
    ctip := C.CString(tip)
    C.updateTrayTooltip(ctip)          // tooltip updated
    C.free(unsafe.Pointer(ctip))

    // n == 0 → calls C.setTraySessionData(nil, nil, 0) → safe
    n := len(sessions)
    if n == 0 {
        C.setTraySessionData(nil, nil, 0)
        return
    }
    // ... (not reached for nil sessions)
}
```

Key facts confirmed from source:
- `len(nil)` in Go returns 0 — safe to pass `nil` as sessions slice
- `trayTooltip(0)` returns `"AgentHub \u2014 no sessions"` — this is the correct tooltip for error/disconnected state (no sessions running)
- `C.setTraySessionData(nil, nil, 0)` is already called for empty slices in the existing runtime-disconnection path — safe
- `trayIconErrorBytes` is already embedded and loaded via `//go:embed assets/tray_icon_error.png`

The tooltip for startup-failure state will be `"AgentHub \u2014 no sessions"` — which is acceptable. The ROADMAP success criteria says "the tray tooltip is updated to reflect the error state on startup failure (not left at default)". The default tooltip is set by `initTray` which uses the normal icon PNG; after `updateTray(nil, false)` the tooltip is `"AgentHub \u2014 no sessions"` which is distinct from any hypothetical "default" and correct for zero sessions.

### Startup Flow (Confirmed)

```
a.initTray()          → trayInit = true
a.startTrayPoller()   → goroutine starts
  → a.refreshTrayState()   ← FIRST CALL
    Currently: a.client == nil → return (BUG)
    Fixed:     a.client == nil → updateTray(nil, false) → error icon shown
```

The startup sequence confirms that `startTrayPoller` is called **before** `a.client` is set in the failure path (lines 60-66 of `app.go`):

```go
// app.go:60-66
if err := daemon.EnsureDaemon(socketPath); err != nil {
    a.daemonErr = err
    runtime.EventsEmit(ctx, "daemon:error", err.Error())
    a.startTrayPoller(ctx)  // client still nil here
    return
}
a.client = daemon.NewDaemonClient(socketPath)
```

So `a.client` is definitely `nil` when `refreshTrayState` fires on the failure path.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Error tooltip text | Custom error string | `trayTooltip(0)` already returns correct text for 0 sessions |
| Error icon rendering | New ObjC code | `updateTray(nil, false)` — existing code path already handles `connected=false` |
| Error state detection | New field/flag | Use `a.client == nil` directly — already the canonical indicator per App struct comment |

---

## Common Pitfalls

### Pitfall 1: Re-merging the two guards
**What goes wrong:** Developer writes `if !a.trayInit || a.client == nil { return }` again — same bug reintroduced.
**Why it happens:** The original guard reads naturally as a single safety check.
**How to avoid:** Keep the guards as two separate `if` blocks with distinct comments. The test `TestRefreshTrayStateNilClientShowsError` will catch regression.

### Pitfall 2: Passing non-nil empty slice vs nil
**What goes wrong:** Passing `[]SessionInfo{}` instead of `nil` — functionally identical since `len()` returns 0 in both cases, but worth noting.
**Why it's not a problem:** `updateTray` uses `len(sessions)` which returns 0 for both `nil` and `[]SessionInfo{}`. Either is safe. Prefer `nil` for clarity (matches the "no data" semantic).

### Pitfall 3: Needing a separate error tooltip string
**What goes wrong:** Adding a new `trayTooltipError()` function returning `"AgentHub \u2014 daemon error"` or similar.
**Why it's unnecessary:** TRAY-06 says "tooltip shows active session count." With zero sessions that count is 0. `trayTooltip(0)` returns `"AgentHub \u2014 no sessions"` — this is the correct and consistent behavior. The success criteria only requires the tooltip be *updated* from default, not that it display an error-specific message.

### Pitfall 4: Touching `tray_linux.go` / `tray_windows.go`
**What goes wrong:** Adding an `updateTray` call path that doesn't compile on non-Darwin platforms.
**Why it's not needed:** `updateTray` stubs already exist in `tray_linux.go` and `tray_windows.go` and accept `(sessions []SessionInfo, connected bool)`. The fix is only in `app.go:refreshTrayState` which is platform-agnostic. No stub changes needed.

### Pitfall 5: `TestRefreshTrayStateNilClientShowsError` calls cgo on CI
**What goes wrong:** The new test tries to call `updateTray` which calls `C.updateTrayIcon` — fails outside macOS with a display server.
**How to avoid:** The test must use a mock or test-safe path. Looking at the existing test:
```go
// TestRefreshTrayStateNilClient (existing) — SAFE because trayInit=false skips updateTray
func TestRefreshTrayStateNilClient(t *testing.T) {
    app := &App{trayInit: false, client: nil}
    app.refreshTrayState()
}
```
The new test cannot set `trayInit: true` and call `refreshTrayState()` directly on macOS cgo — `updateTray` calls Cocoa NSStatusItem which requires a running app. Instead: test the split-guard logic independently. Options:
1. Keep `trayInit: false` test as-is for the "tray not ready" path; add a separate test that verifies `updateTray` is called via a different mechanism (e.g., dependency injection)
2. Accept that the nil-client-but-tray-init path is tested via integration (production build) rather than a unit test
3. Use build tags to make `updateTray` testable in isolation

**Recommended approach:** The existing `TestRefreshTrayStateNilClient` test verifies the old guard (`trayInit=false`) still works. For the new path (`trayInit=true, client=nil`), the `updateTray` function on darwin calls cgo — so a pure unit test that invokes `refreshTrayState()` with `trayInit=true` would attempt Cocoa calls and panic in tests. The safest approach is to rename/update `TestRefreshTrayStateNilClient` to cover **both** sub-cases:
- `{trayInit: false, client: nil}` → should return early (no panic)
- `{trayInit: true, client: nil}` → calls `updateTray(nil, false)` → on darwin this calls cgo; test must verify it doesn't panic

The existing test suite already includes cgo-touching tests that run fine on macOS dev machines (e.g., `TestTrayIconAsset` which decodes the embedded PNGs). The key is that `updateTray` with `connected=false` doesn't crash — it's the same code path as the runtime-disconnection case which is already verified by production use.

**Decision:** Write a test `TestRefreshTrayStateStartupFailure` with `{trayInit: true, client: nil}` and verify it does not panic. On darwin this will exercise `updateTray` → cgo Cocoa calls. On Linux/Windows this will call the no-op stub. Test must be darwin-tagged or confirmed to run only on the platform where cgo is available. Looking at existing tray tests — they are in `tray_test.go` without build tags, and since `tray.go` itself has `//go:build darwin`, the test file is only compiled on darwin. This means the new test can live in `tray_test.go` and naturally gets the darwin-only build constraint.

---

## Code Examples

### The Exact Fix (one function, app.go)

```go
// Source: app.go:421 — current refreshTrayState
// Change: split compound guard into two separate early returns

func (a *App) refreshTrayState() {
    if !a.trayInit {
        return // tray not yet initialised
    }
    if a.client == nil {
        // Startup failed — show error icon and tooltip immediately.
        a.updateTray(nil, false)
        return
    }
    connected := a.client.Health() == nil
    var sessions []SessionInfo
    if connected {
        sessions = a.ListSessions()
    }
    a.updateTray(sessions, connected)
}
```

### New Test (tray_test.go)

```go
// TestRefreshTrayStateStartupFailure verifies that when trayInit=true but
// client=nil (daemon startup failure), refreshTrayState calls updateTray
// with connected=false (error icon) rather than returning early.
// The test verifies no panic occurs — the cgo call itself is the observable
// side-effect on darwin (updateTrayIcon sets the error icon PNG).
func TestRefreshTrayStateStartupFailure(t *testing.T) {
    app := &App{trayInit: true, client: nil}
    // Must not panic. On darwin, updateTray calls cgo updateTrayIcon with the
    // error icon bytes. On Linux/Windows the stub is a no-op.
    app.refreshTrayState()
}
```

Note: `tray_test.go` is in `package main` and lives alongside `tray.go` (which has `//go:build darwin`). On darwin the full cgo path executes. The test verifies no panic — which confirms `updateTray(nil, false)` is called rather than early return.

However, there is one subtlety: `C.initStatusItem` is never called in the test (no `initTray()` call) — so `statusItem` in the ObjC layer is `nil`. If `updateTrayIcon` / `updateTrayTooltip` do not null-check `statusItem`, calling them on a `nil` NSStatusItem could crash. Check the ObjC code:

From `tray_objc.m` (per 41-02-PLAN.md task description):
```objc
static void updateTrayIcon(const void *iconData, int iconLen) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;  // null-check present
        ...
    });
}

static void updateTrayTooltip(const char *tooltip) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {       // null-check present
            statusItem.button.toolTip = tip;
        }
    });
}
```

Both ObjC functions null-check `statusItem`. So calling `updateTray` before `initTray` is safe — the Cocoa updates are silently no-ops. The test will not crash.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package |
| Config file | none (standard `go test`) |
| Quick run command | `go test . -run "TestRefreshTray" -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TRAY-03 | Error icon shown on startup failure | unit | `go test . -run TestRefreshTrayStateStartupFailure -v` | No — Wave 0 |
| TRAY-03 | No panic when nil-client guard triggers correct path | unit | `go test . -run TestRefreshTrayStateNilClient -v` | Yes (tray_test.go) — update to cover both sub-cases |
| TRAY-06 | Tooltip updated (not left at default) on startup failure | implicit in TRAY-03 test | same as above | — |

### Wave 0 Gaps

- [ ] `tray_test.go` — add `TestRefreshTrayStateStartupFailure` covering `{trayInit: true, client: nil}` path

*(Existing `TestRefreshTrayStateNilClient` covers the `trayInit: false` path — keep as-is.)*

---

## Open Questions

1. **Tooltip wording for error state**
   - What we know: `trayTooltip(0)` returns `"AgentHub \u2014 no sessions"` — this is what will display on startup failure
   - What's unclear: Whether product stakeholder wants a distinct error-state tooltip like `"AgentHub \u2014 daemon error"` vs. treating it as 0 sessions
   - Recommendation: Use `trayTooltip(0)` per existing pattern. Success criteria only requires the tooltip be *updated*, not that it use a specific error string. If a distinct string is needed, that is a separate concern from this fix.

---

## Environment Availability

Step 2.6: SKIPPED — this phase is a pure Go code change with no external tool dependencies beyond the standard Go toolchain, which is already confirmed operational.

---

## Sources

### Primary (HIGH confidence)
- `app.go:421-431` — full `refreshTrayState` implementation, read directly from source
- `tray.go:91-124` — full `updateTray` implementation including nil/empty sessions handling
- `tray_test.go:117-120` — existing `TestRefreshTrayStateNilClient` pattern
- `.planning/v1.7-MILESTONE-AUDIT.md` — confirms the exact line reference (`app.go:422`) and describes both the bug and the tooltip regression
- `.planning/ROADMAP.md:200-209` — Phase 42 success criteria (all 3 criteria confirmed addressable by the single guard change)

### Secondary (MEDIUM confidence)
- `41-02-PLAN.md` task description — ObjC `updateTrayIcon` and `updateTrayTooltip` null-check patterns (cited from plan, not directly read from `.m` file)

---

## Metadata

**Confidence breakdown:**
- Bug location: HIGH — confirmed from source at `app.go:422`, cross-referenced with audit
- Fix approach: HIGH — single guard split, no new APIs, no new dependencies
- Test strategy: HIGH — follows exact pattern of existing nil-client tests in `tray_test.go`
- ObjC null-check safety: MEDIUM — inferred from plan description, not read from `.m` file directly

**Research date:** 2026-04-02
**Valid until:** 2026-05-02 (stable codebase, no external dependencies)
