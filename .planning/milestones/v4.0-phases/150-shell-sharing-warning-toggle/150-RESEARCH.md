# Phase 150: Shell-Sharing Warning Toggle — Research

**Researched:** 2026-06-23
**Domain:** Daemon-backed settings toggle + cross-surface warning interception (Go daemon + React frontend)
**Confidence:** HIGH — all cited refs verified against current source

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Introduce a NEW daemon-backed setting (the "warning enabled" master switch). Separate from the existing one-time `shellWebShareWarned` flag. Both coexist.
- **D-02:** Shell warning fires iff: session is a shell AND `warningEnabled == true` AND `shellWebShareWarned == false`.
- **D-03:** Re-arm semantics — flipping the Settings toggle OFF→ON MUST reset `shellWebShareWarned` to `false`, so the warning shows again on next shell web-share.
- **D-04:** Behavior matrix: OFF→ON re-arms; ON+never-acked shows once; ON+already-acked stays suppressed; OFF never warns.
- **D-05:** Toggle lives in the Session Behavior section of Settings (`SettingsTab.tsx` line 413), directly below the Auto-close-on-exit toggle.
- **D-06:** Label: "Warn before web-sharing a shell session." Reuse `role=switch` colorblind-safe toggle pattern.
- **D-07:** Turning the toggle OFF prompts a confirmation dialog ("Disable the shell web-share security warning?" → Cancel / Disable). Turning it ON is instant.
- **D-08:** Default ON on fresh install.
- **D-09:** Wire the shell warning into the Hub Share modal ON-path (`SessionShareModal.handleShareToggle`) for shell sessions. Both surfaces respect the same `warningEnabled && !warned` gate.
- **D-10:** Reuse existing `ShellWebShareBanner` component and interception/race-mitigation logic from `App.tsx`. Don't fork the warning UI.

### Claude's Discretion

- Exact confirm-dialog component/wording (reuse existing modal/dialog primitives).
- The new setting's exact field/RPC name (mirror the `ShellWebShareWarned` naming convention — `Get/SetShellWebShareWarningEnabled` is the natural analog).
- Whether the Share-modal warning renders as the existing banner inline or as a modal-appropriate variant.

### Deferred Ideas (OUT OF SCOPE)

- Retire/consolidate the legacy StatusBar per-tab web toggle.
- Web-surface behavior for the warning.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SET-01 | A Settings toggle enables/disables the shell-session web-sharing warning, and can re-enable it after the first acknowledgment; the warning fires consistently across both share surfaces — the Hub Share modal and the per-tab StatusBar toggle (#51). | Full plumbing chain verified; both share surfaces located; interception pattern documented. |
</phase_requirements>

---

## Summary

Phase 150 wires a new daemon-backed boolean setting (`shellWebShareWarningEnabled`) through the full stack: Go engine struct field + JSON tag → save/load → HTTP API handler → daemon client → Wails app binding → frontend state. It mirrors the Phase 101 `shellWebShareWarned` plumbing exactly at every layer. The existing `ShellWebShareBanner` component and its race-mitigation pattern (synchronous local state set before awaiting RPCs) are reused unchanged.

The primary complexity is cross-surface wiring: the warning already fires on the StatusBar/`handleToggleWeb` path (App.tsx) but does NOT fire on the Hub Share modal path (`SessionShareModal.handleShareToggle`). Wiring the modal requires adding `cli` to the `ShareSession` interface and threading `shellWebShareWarned`/`warningEnabled` state into `SessionShareModal` as props, or handling the interception at the HubPanel level where `SessionInfo` (which includes `cli`) is already available. The latter is cleaner because `shellWebShareWarned` and `warningEnabled` already live in App.tsx where `HubPanel` is rendered.

Re-arm semantics (D-03) require that `SetShellWebShareWarningEnabled(true)` atomically resets `shellWebShareWarned = false` and saves both fields in a single disk write. This is a two-field save in the engine, not two separate RPC calls.

**Primary recommendation:** Mirror `shellWebShareWarned` plumbing verbatim for `shellWebShareWarningEnabled`; handle Share-modal interception at the HubPanel level via new props threaded from App.tsx; use `RegenerateKeyModal` CSS class pattern (`.quit-modal*`) for the confirm-on-disable dialog.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Warning-enabled persistence | Backend (daemon engine) | — | Both `shellWebShareWarningEnabled` and the re-arm of `shellWebShareWarned` must be atomic and survive restarts |
| HTTP API endpoints | Backend (daemon API) | — | GET/PATCH `/settings/shell-web-share-warning-enabled` mirrors the existing warned endpoint |
| Daemon client RPC | Backend (daemon client) | — | Typed Go client wraps HTTP; called by Wails App binding |
| Wails binding | Go app layer (app.go) | — | `GetShellWebShareWarningEnabled` / `SetShellWebShareWarningEnabled` exposed to frontend |
| Frontend state | Browser (App.tsx) | — | `shellWebShareWarningEnabled` loaded on mount; governs both share surfaces |
| StatusBar warning interception | Browser (App.tsx `handleToggleWeb`) | — | Already wired; add `warningEnabled` gate alongside existing `!shellWebShareWarned` check |
| Share-modal warning interception | Browser (HubPanel or SessionShareModal) | App.tsx | `cli` available in HubPanel via `SessionInfo`; intercept there OR add props to SessionShareModal |
| Settings toggle UI | Browser (SettingsTab.tsx) | — | Session Behavior section, below auto-close toggle (line 413+) |
| Confirm-on-disable dialog | Browser (new inline or modal component) | — | Reuse `.quit-modal*` CSS; RegenerateKeyModal is the closest analog |
| SettingsSearch index | Browser (SettingsSearch.tsx) | — | Add label entry; target `settings-session-behavior` |

---

## Verified Line References

All CONTEXT.md citations verified against current source. Findings below show actual current lines.

### engine.go — `shellWebShareWarned` plumbing chain

| CONTEXT.md Cited | Actual Content | Current Lines | Drift? |
|-----------------|----------------|---------------|--------|
| `engine.go:45` | `shellWebShareWarned bool` struct field | **45** | None |
| `engine.go:110` | `ShellWebShareWarned bool \`json:"shellWebShareWarned,omitempty"\`` in `daemonSettings` | **110** | None |
| `engine.go:195` | `e.shellWebShareWarned = s.ShellWebShareWarned` in `loadSettingsFromDisk` | **195** | None |
| `engine.go:222` | `ShellWebShareWarned: e.shellWebShareWarned,` in `saveSettingsToDisk` | **222** | None |
| `engine.go:993-1001` | `GetShellWebShareWarned()` + `SetShellWebShareWarned()` | **993–1011** | Minor: Set function body runs to 1011, not 1001 |

`GetShellWebShareWarned` (engine.go:995–999): reads `e.mu.RLock`, returns `e.shellWebShareWarned`.
`SetShellWebShareWarned` (engine.go:1001–1011): acquires `e.mu.Lock`, sets field, calls `saveSettingsToDisk`, unlocks. Returns `error` (always nil today).

### api.go — HTTP handlers

| CONTEXT.md Cited | Actual Content | Current Lines | Drift? |
|-----------------|----------------|---------------|--------|
| `api.go:112-113` | Route registrations for GET/PATCH `/settings/shell-web-share-warned` | **112–113** | None |
| `api.go:725-741` | `handleGetShellWebShareWarned` + `handleUpdateShellWebShareWarned` | **725–746** | Minor: PATCH handler body runs to 746 |

GET handler (api.go:727–729): `writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarned()})`.
PATCH handler (api.go:733–746): decodes `{"value": bool}`, calls `a.engine.SetShellWebShareWarned(req.Value)`, 204 on success.

### client.go — daemon client methods

| CONTEXT.md Cited | Actual Content | Current Lines | Drift? |
|-----------------|----------------|---------------|--------|
| `client.go:166-178` | `GetShellWebShareWarned` + `SetShellWebShareWarned` | **166–181** | Minor: Set method runs to 181 |

`GetShellWebShareWarned` (client.go:168–174): GET `/settings/shell-web-share-warned` → `resp["value"]`.
`SetShellWebShareWarned` (client.go:178–181): PATCH `/settings/shell-web-share-warned` with `{"value": val}`.

### App.tsx — frontend warning gate

| CONTEXT.md Cited | Actual Content | Current Lines | Drift? |
|-----------------|----------------|---------------|--------|
| `App.tsx:853-910` | `handleToggleWeb` + `handleShellWebShareConfirm` + `handleShellWebShareCancel` | **853–910** | None |
| `App.tsx:886-890` | Race-mitigation: `setShellWebShareWarned(true)` before `await Promise.all([...])` | **890** (single line) | Minor: The synchronous set is at line 890; the await block is 892–895 |

`handleToggleWeb` (App.tsx:853–876): intercepts ON-toggles for shell sessions when `!shellWebShareWarned`. Checks `SHELL_CLIS.has(tab.cli)`. Sets `pendingShellWebToggle`.

`handleShellWebShareConfirm` (App.tsx:883–906): race mitigation pattern — `setShellWebShareWarned(true)` at line 890 (synchronous, no await), then `await Promise.all([SetShellWebShareWarned(true), ToggleWebServing(sessionId, true)])`. On error: rolls back `setShellWebShareWarned(false)`.

`SHELL_CLIS` (App.tsx:89): `new Set(['shell', 'bash', 'zsh', 'pwsh', 'powershell'])` — matches `engine.go isShellSession()`.

`shellWebShareWarned` state declared at App.tsx:138. Hydrated via `GetShellWebShareWarned()` at mount (line 536–541). Default false on error (safe degradation).

### SettingsTab.tsx — Session Behavior section and auto-close pattern

| CONTEXT.md Cited | Actual Content | Current Lines | Drift? |
|-----------------|----------------|---------------|--------|
| `SettingsTab.tsx:413` | `<h3 id="settings-session-behavior">Session Behavior</h3>` | **413** | None |
| `SettingsTab.tsx:106-110` | Auto-close state declarations | **107–110** | Minor: starts at 107 |
| `SettingsTab.tsx:180-183` | Auto-close `useEffect` load pattern | **181–186** | Minor: ends at 186 |
| `SettingsTab.tsx:312` | `handleToggleAutoClose` function | **326** | DRIFT: function is at line 326, not 312 |

Auto-close pattern for new toggle to mirror:
- State: `const [autoCloseSession, setAutoCloseSession] = useState(true)` + `autoCloseLoaded`, `autoCloseSaving`, `autoCloseError` (lines 107–110)
- Load: `useEffect(() => { GetAutoCloseSession().then(val => { setAutoCloseSession(val); setAutoCloseLoaded(true) }).catch(...) }, [])` (lines 181–186)
- Save: `handleToggleAutoClose` (lines 326–338): sets saving, calls `SetAutoCloseSession(next)`, updates state, clears saving in `finally`
- JSX: `autoCloseLoaded &&` guard, `role` not set (uses `type="checkbox"` not `role="switch"`) — NOTE: SettingsTab uses `<input type="checkbox">` styled as a toggle, not `role="switch"` button. CONTEXT.md D-06 says to reuse `role=switch` — the Appearance section toggle uses `role="switch"` on a `<button>` (line 446–450), but the auto-close toggle at lines 415–432 uses a checkbox input styled visually as a toggle.

### SessionShareModal.tsx — handleShareToggle (current state, NO warning)

| CONTEXT.md Cited | Actual Content | Current Lines | Drift? |
|-----------------|----------------|---------------|--------|
| `SessionShareModal.tsx:183-199` | `handleShareToggle` | **186–199** | Minor: function declaration at 186, not 183 |

`handleShareToggle` (SessionShareModal.tsx:186–199): calls `await ToggleWebServing(session.id, next)` with NO shell-warning interception. No `cli` field in `ShareSession` interface; no `SHELL_CLIS` reference.

**Critical finding:** `SessionShareModal` receives `session: ShareSession` where `ShareSession` (lines 14–20) has: `id`, `name`, `webEnabled`, `homeDir`, `browseEnabled` — NO `cli` field. The modal does not currently know whether a session is a shell.

In `HubPanel.tsx` (line 262): `shareModalSession` is typed `SessionInfo | null`. `SessionInfo` (from Wails binding `App.d.ts` line 6–24) DOES include `cli: string`. So `cli` is available in HubPanel but not passed to SessionShareModal.

### SettingsSearch.tsx

Current `SEARCH_INDEX` (lines 24–38): includes `{ label: 'Auto-close tab on exit', target: 'settings-session-behavior' }` at line 29. New entry must be added: `{ label: 'Warn before web-sharing a shell session.', target: 'settings-session-behavior' }`.

---

## Standard Stack

No new external packages. This phase is pure Go + React using existing project dependencies.

| Layer | Technology | Existing Analog |
|-------|-----------|-----------------|
| Go daemon field | `bool` in `daemonSettings` struct | `ShellWebShareWarned bool` (engine.go:110) |
| Go HTTP endpoint | `GET/PATCH /settings/shell-web-share-warning-enabled` | `GET/PATCH /settings/shell-web-share-warned` (api.go:112–113) |
| Go client method | `GetShellWebShareWarningEnabled / SetShellWebShareWarningEnabled` | `Get/SetShellWebShareWarned` (client.go:166–181) |
| Wails binding | `app.go` exported methods + App.js/App.d.ts manual update | `GetShellWebShareWarned / SetShellWebShareWarned` (app.go:501–519) |
| Frontend state | `useState(true)` + loaded/saving/error trio | `autoCloseSession` pattern (SettingsTab.tsx:107–110) |
| Confirm dialog | Inline JSX using `.quit-modal*` CSS classes | `RegenerateKeyModal.tsx` (same CSS) |

---

## Package Legitimacy Audit

No new packages are installed in this phase.

---

## Architecture Patterns

### Full Plumbing Chain — New Setting

The `shellWebShareWarningEnabled` setting (default `true`) must traverse every layer below. Each layer has a verified existing analog.

**Layer 1 — engine.go: struct field + daemonSettings JSON tag**

```go
// In SessionEngine struct (around line 45, after shellWebShareWarned):
shellWebShareWarningEnabled bool  // Phase 150 SET-01: master warning switch (default ON)

// In daemonSettings struct (around line 110, after ShellWebShareWarned):
ShellWebShareWarningEnabled bool `json:"shellWebShareWarningEnabled,omitempty"`
```

**Important:** Because the default is `true` but `omitempty` omits zero-values (`false`), a missing key from an old settings.json file will deserialize as `false`. This would default NEW users to OFF — wrong (D-08 says default ON). Two options:
- Use `*bool` pointer (like `AutoCloseSession`) — nil pointer means "not set, use default true"
- Or: use a `bool` with inverted semantics ("shellWebShareWarningDisabled") so zero = not disabled = warning ON

**Recommendation (mirrors `AutoCloseSession` pattern):** Use `*bool` pointer in `daemonSettings` with `omitempty`. `loadSettingsFromDisk` checks for nil and defaults to `true`. This is the cleanest analog to how `AutoCloseSession` handles its `true` default.

**Layer 2 — engine.go: loadSettingsFromDisk and saveSettingsToDisk**

In `loadSettingsFromDisk` (around line 197):
```go
// After: e.shellWebShareWarned = s.ShellWebShareWarned
if s.ShellWebShareWarningEnabled != nil {
    e.shellWebShareWarningEnabled = *s.ShellWebShareWarningEnabled
} else {
    e.shellWebShareWarningEnabled = true // D-08: default ON
}
```

In `saveSettingsToDisk` (around line 222):
```go
// Add to daemonSettings literal:
ShellWebShareWarningEnabled: &e.shellWebShareWarningEnabled,
```

**Layer 3 — engine.go: Get/Set accessors (re-arm semantics)**

```go
// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
func (e *SessionEngine) GetShellWebShareWarningEnabled() bool {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.shellWebShareWarningEnabled
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// D-03 re-arm: when enabling (val=true), also resets shellWebShareWarned to false
// so the one-time banner fires again on the next shell web-share.
func (e *SessionEngine) SetShellWebShareWarningEnabled(val bool) error {
    e.mu.Lock()
    e.shellWebShareWarningEnabled = val
    if val {
        // Re-arm: reset the one-time acknowledged flag so the banner shows again.
        e.shellWebShareWarned = false
    }
    e.saveSettingsToDisk()
    e.mu.Unlock()
    return nil
}
```

**Key insight:** Re-arm (D-03) happens atomically inside one `saveSettingsToDisk` call. Both `shellWebShareWarningEnabled = true` and `shellWebShareWarned = false` are written together. No second RPC call is needed from the frontend.

**Layer 4 — api.go: route registration + handlers**

```go
// Route registrations (around line 120, after auto-close-session):
a.mux.HandleFunc("GET /settings/shell-web-share-warning-enabled", a.handleGetShellWebShareWarningEnabled)
a.mux.HandleFunc("PATCH /settings/shell-web-share-warning-enabled", a.handleSetShellWebShareWarningEnabled)

// Handlers (mirror handleGet/UpdateShellWebShareWarned at lines 727-746):
func (a *API) handleGetShellWebShareWarningEnabled(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]bool{"value": a.engine.GetShellWebShareWarningEnabled()})
}

func (a *API) handleSetShellWebShareWarningEnabled(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Value bool `json:"value"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    if err := a.engine.SetShellWebShareWarningEnabled(req.Value); err != nil {
        http.Error(w, fmt.Sprintf("persist: %v", err), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

**Layer 5 — client.go: typed HTTP client methods**

```go
// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
func (c *DaemonClient) GetShellWebShareWarningEnabled() (bool, error) {
    var resp map[string]bool
    if err := c.doJSON(http.MethodGet, "/settings/shell-web-share-warning-enabled", nil, &resp); err != nil {
        return false, err
    }
    return resp["value"], nil
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
func (c *DaemonClient) SetShellWebShareWarningEnabled(val bool) error {
    return c.doJSON(http.MethodPatch, "/settings/shell-web-share-warning-enabled",
        map[string]bool{"value": val}, nil)
}
```

**Layer 6 — app.go: Wails binding (GetShellWebShareWarned analog at lines 501–519)**

```go
// GetShellWebShareWarningEnabled returns the warning-enabled master switch.
// Default true when daemon is not connected (preserve safe behavior per D-08).
func (a *App) GetShellWebShareWarningEnabled() bool {
    if a.client == nil {
        return true
    }
    val, err := a.client.GetShellWebShareWarningEnabled()
    if err != nil {
        return true // default: enabled (safe degradation)
    }
    return val
}

// SetShellWebShareWarningEnabled persists the warning-enabled master switch.
// D-03: when enabling, the engine atomically resets shellWebShareWarned too.
func (a *App) SetShellWebShareWarningEnabled(v bool) error {
    if a.client == nil {
        return nil
    }
    return a.client.SetShellWebShareWarningEnabled(v)
}
```

**Layer 7 — Wails bindings (App.js + App.d.ts): MANUAL edit required**

The files `frontend/src/wailsjs/go/main/App.js` and `App.d.ts` are committed to git and manually updated per commit (verified: the "AUTO-GENERATED by Wails" comment is misleading — `wails dev` regenerates them but this project manually maintains them in source).

Add to `App.js` (after the `SetShellWebShareWarned` line at line 16):
```js
// Phase 150 SET-01 — warning-enabled master switch.
export const GetShellWebShareWarningEnabled = () => Call('main.App.GetShellWebShareWarningEnabled', [])
export const SetShellWebShareWarningEnabled = (v) => Call('main.App.SetShellWebShareWarningEnabled', [v])
```

Add to `App.d.ts` (after `SetShellWebShareWarned` line at line 40):
```ts
// Phase 150 SET-01 — warning-enabled master switch.
export function GetShellWebShareWarningEnabled(): Promise<boolean>
export function SetShellWebShareWarningEnabled(v: boolean): Promise<void>
```

**Note:** No `wails generate` command is needed. The `.js` and `.d.ts` stubs are maintained manually in this repo. Verified via git log: every prior binding addition (Phase 130, 132, 137, 139, 146) was a manual edit committed alongside the `app.go` change.

---

### Cross-Surface Interception: Share Modal Design Decision

**The gap:** `SessionShareModal.handleShareToggle` calls `ToggleWebServing` directly with no shell-warning interception. The modal's `ShareSession` interface has no `cli` field.

**Two valid approaches:**

**Option A: Add `cli` to `ShareSession`, thread `shellWebShareWarned` and `warningEnabled` as modal props**

`SessionShareModal` receives:
- `session.cli` (add to `ShareSession` interface)
- `shellWebShareWarned: boolean` prop
- `shellWebShareWarningEnabled: boolean` prop  
- `onShellWarningConfirm: (sessionId: string) => void` prop (App.tsx handles the persist + race mitigation)
- `onShellWarningCancel: () => void` prop

`handleShareToggle` adds the gate:
```ts
if (next && SHELL_CLIS.has(session.cli) && shellWebShareWarningEnabled && !shellWebShareWarned) {
    setPendingShellShare(true)  // local flag to show banner instead of calling ToggleWebServing
    return
}
```

When user confirms in the banner, calls `onShellWarningConfirm(session.id)` which is handled in App.tsx using the same race-mitigation pattern already in `handleShellWebShareConfirm`.

**Option B: Intercept at HubPanel level before opening modal**

HubPanel already has `SessionInfo` (with `cli`). App.tsx threads `shellWebShareWarned` + `warningEnabled` to HubPanel as props. HubPanel's `handleShare` checks the gate; if shell + warning needed, shows the banner before opening `SessionShareModal`.

**Recommendation for planner (Claude's discretion):** Option A is cleaner — it keeps the warning interception inside the component that owns the share toggle. Option B would require the banner to block the modal from opening, creating a two-step interaction (banner → modal) rather than banner inside the modal (the expected UX). CONTEXT.md D-10 says "reuse the interception/race-mitigation logic already in App.tsx" — this suggests the race-mitigation callback pattern stays in App.tsx regardless of which option is used.

**The `cli` availability:** In `HubPanel.tsx` line 262, `shareModalSession` is `SessionInfo | null`. `SessionInfo.cli` is available. If Option A is chosen, HubPanel passes `session={shareModalSession}` and the `ShareSession` interface extends to include `cli`.

---

### Race-Mitigation Pattern (D-10)

The exact pattern from `App.tsx:883–906` to replicate in the Share-modal confirmation path:

```ts
// 1. Set local warned flag SYNCHRONOUSLY (before any await) — prevents double-banner.
setShellWebShareWarned(true)
try {
    await Promise.all([
        SetShellWebShareWarned(true),      // persist acknowledged flag
        ToggleWebServing(sessionId, true),  // perform the actual toggle
    ])
    setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
    setPendingShellWebToggle(null)
} catch (err) {
    // On failure: roll back local warned flag so user can retry.
    setShellWebShareWarned(false)
    setPendingShellWebToggle(null)
}
```

This pattern must be applied in the Share-modal confirmation path (wherever the confirm callback lands — App.tsx or a new handler prop).

---

### Settings Toggle with Confirm-on-Disable Dialog (D-07)

Pattern for the new toggle is a hybrid: instant ON (like auto-close), confirm-required OFF (like regenerate signing key).

**State needed in SettingsTab:**
```ts
// Warning-enabled toggle (Phase 150 SET-01)
const [shellWarnEnabled, setShellWarnEnabled] = useState(true)     // default ON
const [shellWarnLoaded, setShellWarnLoaded] = useState(false)
const [shellWarnSaving, setShellWarnSaving] = useState(false)
const [shellWarnError, setShellWarnError] = useState<string | null>(null)
const [showDisableWarnConfirm, setShowDisableWarnConfirm] = useState(false)
```

**Load pattern (mirrors auto-close at lines 181–186):**
```ts
useEffect(() => {
    GetShellWebShareWarningEnabled().then(val => {
        setShellWarnEnabled(val)
        setShellWarnLoaded(true)
    }).catch(() => setShellWarnLoaded(true))
}, [])
```

**Toggle handler:**
```ts
async function handleToggleShellWarnEnabled() {
    const next = !shellWarnEnabled
    if (!next) {
        // Turning OFF is destructive — show confirm dialog (D-07)
        setShowDisableWarnConfirm(true)
        return
    }
    // Turning ON is instant (D-07)
    setShellWarnSaving(true)
    setShellWarnError(null)
    try {
        await SetShellWebShareWarningEnabled(true)
        setShellWarnEnabled(true)
        // Note: the daemon atomically resets shellWebShareWarned (re-arm D-03).
        // App.tsx must also re-hydrate shellWebShareWarned from daemon after this.
    } catch (err) {
        setShellWarnError('Could not save preference — ' + ...)
    } finally {
        setShellWarnSaving(false)
    }
}

async function handleConfirmDisableWarn() {
    setShowDisableWarnConfirm(false)
    setShellWarnSaving(true)
    setShellWarnError(null)
    try {
        await SetShellWebShareWarningEnabled(false)
        setShellWarnEnabled(false)
    } catch (err) {
        setShellWarnError('Could not save preference — ' + ...)
    } finally {
        setShellWarnSaving(false)
    }
}
```

**Re-hydration gap:** When the user turns ON (re-arm, D-03), the daemon resets `shellWebShareWarned = false` atomically. But `App.tsx` has `shellWebShareWarned` in its own React state. App.tsx must re-load `shellWebShareWarned` from the daemon after `SetShellWebShareWarningEnabled(true)` completes (or SettingsTab must notify App.tsx somehow). Options:
1. App.tsx exposes an `onWarnEnabledChange` callback prop to SettingsTab that re-calls `GetShellWebShareWarned()`.
2. SettingsTab calls `SetShellWebShareWarned(false)` directly after enabling (App.tsx's state is already a local copy — the warning gate reads it; resetting it here would keep them in sync).

**Recommendation for planner:** Option 2 is simpler — after `SetShellWebShareWarningEnabled(true)` succeeds, SettingsTab also calls `SetShellWebShareWarned(false)` and triggers a prop/callback to reset App.tsx's `shellWebShareWarned` state. Alternatively, App.tsx can listen via its own `useEffect` on the `shellWarnEnabled` setting if threaded as a prop. The planner should pick one and be explicit.

**Confirm dialog:** Reuse `RegenerateKeyModal`'s `.quit-modal*` CSS classes for the confirm-on-disable dialog. A small inline `DisableShellWarnModal` component is sufficient (same structure as `RegenerateKeyModal.tsx`).

---

### SettingsTab JSX Placement

New toggle goes inside the Session Behavior `<div className="settings-panel__field-group">` block, after the auto-close toggle block (after line 438). The current structure:

```
line 412: {/* Session Behavior section (Phase 84 D-11) */}
line 413: <h3 id="settings-session-behavior">Session Behavior</h3>
line 414: <div className="settings-panel__field-group">
line 415:   {autoCloseLoaded && (... auto-close toggle row ...)}
line 427:   <input type="checkbox" id="autoCloseSession" .../>
line 434:   <p className="description">...</p>
line 437:   {autoCloseError && ...}
line 438: </div>
line 440: {/* Appearance section */}
```

New `settings-panel__field-group` div for the warning toggle is inserted between lines 438 and 440.

---

### Updated `handleToggleWeb` Gate (StatusBar path)

After adding `shellWebShareWarningEnabled` to App.tsx state and loading it from daemon on mount, update `handleToggleWeb` (App.tsx:862–868) to add the `warningEnabled` gate:

```ts
// Current (line 864):
if (tab && SHELL_CLIS.has(tab.cli) && !shellWebShareWarned) {

// After this phase:
if (tab && SHELL_CLIS.has(tab.cli) && shellWebShareWarningEnabled && !shellWebShareWarned) {
```

The `shellWebShareWarningEnabled` state must be loaded on mount alongside `shellWebShareWarned` (line 536–541), using `GetShellWebShareWarningEnabled()`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Settings persistence | Custom file writer | `saveSettingsToDisk` + daemonSettings struct | Already handles atomic write, schema migration, mutex safety |
| Confirm dialog | Custom confirm UI from scratch | `.quit-modal*` CSS classes (RegenerateKeyModal pattern) | Established pattern, consistent look, focus management included |
| Shell type detection | Re-implement shell check | `SHELL_CLIS` Set (App.tsx:89) / `isShellSession()` (engine.go:141) | DRY — three call sites already; comment at line 85 notes to revisit extraction if a fourth appears |
| Race condition mitigation | Debounce / lock | Synchronous local state set before await (App.tsx:890 pattern) | Purpose-built for this exact problem; already tested |

---

## Common Pitfalls

### Pitfall 1: Default-ON with `omitempty` serialization trap
**What goes wrong:** Using `bool` (not `*bool`) with `omitempty` means the default-true `shellWebShareWarningEnabled` is serialized as `false` on a zero-value engine, and an absent key in old settings.json (which deserializes as `false`) silently defaults new/upgraded users to OFF.
**Why it happens:** `json:"...,omitempty"` omits zero-value bools (`false`), so a value of `true` is written but `false` is indistinguishable from "not set".
**How to avoid:** Use `*bool` pointer in `daemonSettings`, same as `AutoCloseSession`. Nil pointer → default `true` in `loadSettingsFromDisk`. Non-nil → use the stored value.
**Warning signs:** Tests that create a fresh engine see `GetShellWebShareWarningEnabled() = false` when they expect `true`.

### Pitfall 2: Re-arm requires frontend state sync
**What goes wrong:** `SetShellWebShareWarningEnabled(true)` atomically resets `shellWebShareWarned` in the daemon. But App.tsx's `shellWebShareWarned` React state is stale — it still holds the old `true` value. Next shell web-share is incorrectly suppressed.
**Why it happens:** React state is a local copy; daemon-side reset is invisible to the frontend unless re-queried.
**How to avoid:** After `SetShellWebShareWarningEnabled(true)` succeeds, also call `GetShellWebShareWarned()` (or `SetShellWebShareWarned(false)` directly) to sync App.tsx state. Document this as a required side effect.
**Warning signs:** After toggling OFF→ON in Settings, the shell warning does NOT appear on the next web-share despite re-arm semantics.

### Pitfall 3: Share modal doesn't receive `cli`
**What goes wrong:** Warning interception in `handleShareToggle` checks `SHELL_CLIS.has(session.cli)` but `ShareSession` has no `cli` field. TypeScript will reject the property access.
**Why it happens:** `ShareSession` (SessionShareModal.tsx:14–20) was defined without `cli` — the modal historically only needed `id`, `name`, `webEnabled`, `homeDir`, `browseEnabled`.
**How to avoid:** Add `cli: string` to `ShareSession` interface and thread it from HubPanel (where `SessionInfo.cli` is already available in `shareModalSession`).
**Warning signs:** TypeScript error on `session.cli` in SessionShareModal; or `cli` check silently passes for non-shell sessions because it's always `undefined`.

### Pitfall 4: Double warning on the first shell web-share (both surfaces)
**What goes wrong:** If the StatusBar path and the Share-modal path both show the banner, a user who web-shares via the Share modal then goes to the StatusBar toggle gets a second banner (or vice versa).
**Why it happens:** Both surfaces use the same `shellWebShareWarned` flag. The race-mitigation pattern sets it synchronously. If implemented correctly with the shared App.tsx state, one surface confirming the banner immediately suppresses the other.
**How to avoid:** The shared `shellWebShareWarned` state in App.tsx is the authority for both surfaces. Share-modal interception must use and update the same state (via props/callbacks), not independent local state.
**Warning signs:** Banner appears twice for the same session on the same machine across two share surfaces.

### Pitfall 5: Wails binding not updated
**What goes wrong:** `app.go` has `GetShellWebShareWarningEnabled` / `SetShellWebShareWarningEnabled` but `App.js` / `App.d.ts` are not updated. Frontend gets `undefined` when importing.
**Why it happens:** Wails does NOT auto-regenerate bindings in this project's workflow (no `wails generate` step in CI; bindings are manually maintained).
**How to avoid:** Always update `App.js` and `App.d.ts` in the same task/commit as `app.go` changes.
**Warning signs:** `GetShellWebShareWarningEnabled is not a function` runtime error; TypeScript compile error on import.

### Pitfall 6: Confirm dialog accessibility
**What goes wrong:** The confirm-on-disable dialog is implemented as a plain `<div>` without `role="dialog"` and `aria-modal`, causing screen readers to read through the background.
**Why it happens:** Quick inline implementation skips ARIA.
**How to avoid:** Mirror `RegenerateKeyModal`'s `role="dialog" aria-modal="true"` and focus the Cancel button on open (safe-action default).
**Warning signs:** `aria-modal` attribute missing; Escape key doesn't close the dialog.

---

## Runtime State Inventory

Not applicable — this is a greenfield settings addition, not a rename/refactor. No existing stored data refers to `shellWebShareWarningEnabled` (it is a new field).

---

## Environment Availability

Step 2.6: All dependencies are in-project (Go, React, existing Wails). No external tools required beyond the standard development environment already verified in prior phases.

---

## Validation Architecture

`workflow.nyquist_validation` is absent from `.planning/config.json` — treat as enabled.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.x + Go `testing` package |
| Config file | `frontend/vite.config.ts` (vitest block at line 14) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` (vitest) / `go test -race -short ./internal/daemon/...` (Go) |
| Full suite command | `go test -race -short ./...` + `cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SET-01 | `shellWebShareWarningEnabled` default ON (fresh engine = true) | Go unit | `go test -race ./internal/daemon/ -run TestShellWebShareWarningEnabled_Default` | Wave 0 — new file |
| SET-01 | `shellWebShareWarningEnabled` persists ON/OFF round-trip through settings.json | Go unit | `go test -race ./internal/daemon/ -run TestShellWebShareWarningEnabled_Persists` | Wave 0 — new file |
| SET-01 | Re-arm: `SetShellWebShareWarningEnabled(true)` resets `shellWebShareWarned` to false atomically | Go unit | `go test -race ./internal/daemon/ -run TestShellWebShareWarningEnabled_ReArm` | Wave 0 — new file |
| SET-01 | OFF never warns — gate is `warningEnabled && !warned` (both must be true) | Go unit | `go test -race ./internal/daemon/ -run TestShellWebShareWarningEnabled_OffBehavior` | Wave 0 — new file |
| SET-01 | HTTP GET `/settings/shell-web-share-warning-enabled` returns `{"value": true}` by default | Go unit | `go test -race ./internal/daemon/ -run TestAPIGetShellWebShareWarningEnabled` | Wave 0 — new file |
| SET-01 | HTTP PATCH `/settings/shell-web-share-warning-enabled` persists value | Go unit | `go test -race ./internal/daemon/ -run TestAPIPatchShellWebShareWarningEnabled` | Wave 0 — new file |
| SET-01 | `DaemonClient` round-trip get/set | Go unit | `go test -race ./internal/daemon/ -run TestDaemonClient_ShellWebShareWarningEnabled` | Wave 0 — new file |
| SET-01 | Settings toggle renders in Session Behavior section | vitest source gate | `cd frontend && pnpm test -- --run SettingsTab.shell-warn-toggle` | Wave 0 — new file |
| SET-01 | Toggle state machine: loaded/saving/error lifecycle | vitest | `cd frontend && pnpm test -- --run SettingsTab.shell-warn-toggle` | Wave 0 — new file |
| SET-01 | Confirm-on-disable dialog appears when toggling OFF; immediate ON has no dialog | vitest | `cd frontend && pnpm test -- --run SettingsTab.shell-warn-toggle` | Wave 0 — new file |
| SET-01 | Share-modal interception: shell session + warningEnabled + !warned → banner shown, no ToggleWebServing call | vitest | `cd frontend && pnpm test -- --run SessionShareModal` | Extend existing `SessionShareModal.test.tsx` |
| SET-01 | Share-modal interception: non-shell session → banner NOT shown; AI CLI proceeds immediately | vitest | Same | Extend existing |
| SET-01 | Share-modal interception: warningEnabled=false → banner NOT shown even for shell | vitest | Same | Extend existing |
| SET-01 | SettingsSearch index contains "Warn before web-sharing a shell session." entry | vitest source gate | `cd frontend && pnpm test -- --run SettingsSearch` | Wave 0 — new file or extend existing |
| SET-01 | Shell warning fires on BOTH surfaces — StatusBar (`handleToggleWeb`) and Share modal (`handleShareToggle`) | Manual | — | M-NN (new entry in TESTING.md §5) |
| SET-01 | Live persistence across restart: toggle OFF, restart app, confirm warning suppressed | Manual | — | M-NN (new entry in TESTING.md §5) |

### Sampling Rate

- **Per task commit:** `go test -race -short ./internal/daemon/ && cd frontend && pnpm test`
- **Per wave merge:** Full suite — `go test -race -short ./... && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/engine_shell_warn_test.go` — covers SET-01 Go unit tests (default, persist, re-arm, off-behavior); mirrors `engine_test.go:935–1029` structure
- [ ] `internal/daemon/api_shell_warn_test.go` OR extend `api_test.go` — covers HTTP handler tests; mirrors `TestAPIGetShellWebShareWarned_Default` etc. at api_test.go:1590+
- [ ] `frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx` — new vitest file; covers toggle state machine + confirm-on-disable dialog + SettingsSearch entry
- [ ] Extend `frontend/src/components/__tests__/SessionShareModal.test.tsx` — add shell-warning interception tests (shell vs. non-shell; warningEnabled gate; banner appearance without ToggleWebServing call)

---

## Security Domain

`security_enforcement` is not set to `false` in config.json — section required.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes | Boolean value from daemon is type-safe; PATCH body decoded with `json.Decoder` (already in use) |
| V6 Cryptography | no | — |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Settings tampering via PATCH endpoint | Tampering | Existing daemon API is localhost-only (Unix socket); no external exposure |
| Confirm-dialog bypass (UI-level) | Elevation of Privilege | Confirm dialog is client-side only — but the consequence (disabling a warning) is low severity; the warning is defense-in-depth, not access control |
| Re-arm bypass: toggling OFF→ON doesn't reset `shellWebShareWarned` | Tampering | Atomic reset in `SetShellWebShareWarningEnabled` engine method; both fields saved in one `saveSettingsToDisk` call |

**Security note:** The shell web-share warning is defense-in-depth (the daemon API already blocks auto-enable for shells per api.go:407). Disabling the warning is a user choice, not a security hole. The confirm-on-disable UX (D-07) is appropriate friction for a security guardrail.

---

## Open Questions

1. **Where does `shellWebShareWarningEnabled` state live for the Share-modal interception?**
   - What we know: `SessionShareModal` doesn't currently receive `cli` or `shellWebShareWarned`; HubPanel has both via `SessionInfo` and App.tsx props.
   - What's unclear: Whether to add props to `SessionShareModal` (Option A) or intercept at HubPanel level (Option B).
   - Recommendation: Option A (add `cli` to `ShareSession`; thread `shellWebShareWarned` + `warningEnabled` as props; confirm callback bubbles to App.tsx). Cleaner UX — banner is inside the modal, not blocking its open.

2. **Re-hydrating App.tsx `shellWebShareWarned` state after re-arm**
   - What we know: `SetShellWebShareWarningEnabled(true)` resets daemon-side `shellWebShareWarned` to false; App.tsx holds its own copy in state.
   - What's unclear: Which component triggers the re-sync (SettingsTab calls SetWarningEnabled; App.tsx owns shellWebShareWarned state).
   - Recommendation: After `SetShellWebShareWarningEnabled(true)` succeeds in SettingsTab, SettingsTab also calls `SetShellWebShareWarned(false)` (to be safe) AND App.tsx receives an `onShellWarnEnabledChange` prop from App.tsx root — when toggled ON, App.tsx re-calls `GetShellWebShareWarned()` to resync.

3. **Where does the `ShellWebShareBanner` render inside the Share modal?**
   - What we know: The banner currently renders in the `.banner-stack` at the top of App.tsx's root layout. Inside `SessionShareModal`, there is no `.banner-stack`.
   - What's unclear: Whether to render the `ShellWebShareBanner` inline inside the modal dialog (before the toggle), or use a different presentation.
   - Recommendation: Render inline inside the modal dialog's content area (before the share toggle) — this is the "modal-appropriate variant" mentioned in Claude's Discretion. The banner's `role="alert"` still works; focus moves to Cancel as designed.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No warning toggle — one-time banner only (Phase 101) | New master switch + one-time ack coexist | Phase 150 | D-02 gate: `warningEnabled && !warned` |
| Warning only on StatusBar path | Warning on both StatusBar + Hub Share modal | Phase 150 (D-09) | Closes cross-surface parity gap (release-blocking) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `*bool` pointer (not `bool`) is the correct implementation for default-ON `shellWebShareWarningEnabled` in `daemonSettings` | Architecture Patterns (Layer 1) | If planner uses `bool` + `omitempty`, new users default to OFF (breaks D-08) — must catch in test |
| A2 | App.js and App.d.ts require manual edit (no `wails generate` step) | Architecture Patterns (Layer 7) | If a future `wails generate` step was added after research, it would overwrite the manual additions — low risk given confirmed git history |
| A3 | The confirm-on-disable dialog can reuse `.quit-modal*` CSS without new CSS tokens | Architecture Patterns (Settings Toggle) | If modal CSS is being replaced in an in-flight redesign, the class names might change — low risk for this phase |

---

## Sources

### Primary (HIGH confidence — verified against current source)

- `internal/daemon/engine.go` — lines 45, 107–116, 149–214, 218–234, 993–1011 (verified)
- `internal/daemon/api.go` — lines 112–113, 725–746 (verified)
- `internal/daemon/client.go` — lines 160–181 (verified)
- `app.go` — lines 498–519, 579–619 (verified)
- `frontend/src/App.tsx` — lines 85–89, 128–141, 853–910 (verified)
- `frontend/src/components/SettingsTab.tsx` — lines 106–110, 180–186, 326–338, 412–438 (verified)
- `frontend/src/components/Hub/SessionShareModal.tsx` — lines 14–20, 183–199 (verified)
- `frontend/src/components/Hub/HubPanel.tsx` — lines 262–266 (verified)
- `frontend/src/components/ShellWebShareBanner.tsx` — full file (verified)
- `frontend/src/components/SettingsSearch.tsx` — lines 24–38 (verified)
- `frontend/src/wailsjs/go/main/App.js` — lines 1–16 (verified manual-maintenance pattern)
- `frontend/src/wailsjs/go/main/App.d.ts` — lines 1–40 (verified)
- `TESTING.md` — Sections 2, 4, 5, 6 (verified)
- `.planning/config.json` — Nyquist enabled (absent key) (verified)

### Secondary (MEDIUM confidence)

- git log `frontend/src/wailsjs/go/main/App.js` — confirms manual maintenance pattern across 5 prior phases

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; all plumbing verified line-by-line against source
- Architecture: HIGH — verified existing analogs; design decisions documented in CONTEXT.md
- Pitfalls: HIGH — derived from actual source inspection (omitempty trap, cli gap, binding gap all directly observed)
- Validation: HIGH — test infrastructure exists and is well-understood; Wave 0 gaps are specific files

**Research date:** 2026-06-23
**Valid until:** 2026-07-23 (stable codebase, no fast-moving dependencies)
