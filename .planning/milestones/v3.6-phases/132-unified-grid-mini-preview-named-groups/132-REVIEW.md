---
phase: 132-unified-grid-mini-preview-named-groups
reviewed: 2026-06-16T00:00:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - internal/daemon/types.go
  - internal/daemon/engine.go
  - internal/daemon/api.go
  - internal/daemon/client.go
  - app.go
  - frontend/src/lib/hubGroups.ts
  - frontend/src/lib/remoteAdapter.ts
  - frontend/src/components/Hub/MiniPreview.tsx
  - frontend/src/components/Hub/GroupSidebar.tsx
  - frontend/src/components/Hub/SessionCard.tsx
  - frontend/src/components/Hub/SessionCardGrid.tsx
  - frontend/src/components/Hub/HubPanel.tsx
  - frontend/src/App.tsx
  - frontend/src/style.css
  - frontend/src/wailsjs/go/main/App.d.ts
findings:
  critical: 3
  warning: 5
  info: 3
  total: 11
status: issues_found
---

# Phase 132: Code Review Report

**Reviewed:** 2026-06-16
**Depth:** standard
**Files Reviewed:** 15
**Status:** issues_found

## Summary

Phase 132 introduces the unified Hub session grid with named groups, a mini-preview poller (CARD-07), and the GroupSidebar (GRID-03). The architecture is largely sound: MiniPreview is correctly plain-text (no xterm), usePreviewPoller is a single shared interval in HubPanel (not per-card), remote sessions are correctly excluded from tail fetches, React text escaping is used throughout (no `dangerouslySetInnerHTML`), and the SessionCard Phase 131 rows (1–5) plus the Open button are intact.

However, three blockers exist: (1) the ANSI-stripping regex is incomplete and will let OSC 8 hyperlinks with embedded `\x1b\\` terminators slip through as literal bytes in mini-preview text; (2) the daemon-side `handleGetSessionTailLines` accepts an unbounded `n` from the wire — the [1..20] clamp exists only in `app.go` (the Wails binding), leaving the HTTP route exposed to unbounded memory allocation; (3) the usePreviewPoller dep array silently drops stopped sessions from the tails Map on each poll cycle, causing the mini-preview to flicker back to "No output yet" for stopped cards.

Five warnings cover: the `assignToGroup` atomic-membership contract broken by a missing `saveGroups` call path in `handleAssignGroup`, the `hub-card--dragging` CSS class referenced in TSX but absent in style.css, the `hub-card__menu-item--header` and `hub-card__menu-item--group` modifier classes referenced in TSX but absent in style.css, the `sessionIdKey` dep including remote session IDs which cannot be fetched (wasted Promise.all round-trips for sessions that are always filtered out), and a `localStorage.getItem` call for the sidebar collapsed state that is not wrapped in a try/catch.

## Critical Issues

### CR-01: ANSI Regex Fails to Strip OSC 8 Hyperlinks with Escaped-String Terminator

**File:** `internal/daemon/engine.go:525`
**Issue:** The `ansiEscape` regex uses `\x1b\\` (ESC + backslash literal) as the OSC terminator alternate:
```
\x1b(?:\[[0-9;?]*[a-zA-Z]|\][^\x07\x1b]*(?:\x07|\x1b\\))
```
The character class `[^\x07\x1b]` stops consuming at the first `\x1b`, which is correct — but in the alternation the "ST" form is `\x1b\\`. In Go regex syntax `\\` inside a raw string literal matches a single backslash `\`. However the actual ST byte sequence emitted by terminals is `\x1b\x5c` (ESC + `\`), which *does* match. The deeper problem is the negative character class `[^\x07\x1b]*`: it stops at the first `\x1b`, so `\x1b]8;;https://foo.com\x1b\` parses the body as `8;;https://foo.com` (stops at `\x1b`) and the trailing `\` (0x5c) is left unstripped as a literal backslash that passes through to the frontend DOM. On an OSC 8 sequence with a long URL this can leave `\some-path-segment` visible in the mini-preview. Not an XSS vector because React text-escapes the content, but it is a display-corruption BLOCKER: the stripped text shown to the user is wrong.

**Fix:** Replace the OSC branch with a two-step approach that first tries BEL-terminated, then ST-terminated, consuming the embedded ESC of the ST terminator correctly. The idiomatic Go approach is to use two alternating passes or a more specific pattern:
```go
// Strip BEL-terminated OSC: ESC ] ... BEL
// Strip ST-terminated OSC:  ESC ] ... ESC \
var ansiEscape = regexp.MustCompile(
    `\x1b(?:` +
    `\[[0-9;?]*[a-zA-Z]` +          // CSI sequences
    `|\][^\x07\x1b]*\x07` +          // OSC terminated by BEL
    `|\][^\x1b]*\x1b\\` +            // OSC terminated by ST (ESC \)
    `)`,
)
```
Note `[^\x1b]*\x1b\\` — no BEL exclusion is needed in the ST branch because BEL-terminated strings would be matched by the prior branch.

---

### CR-02: `handleGetSessionTailLines` — No Upper Bound on `n` at the HTTP Layer

**File:** `internal/daemon/api.go:617-621`
**Issue:** The HTTP handler accepts `n` from the query string and passes it directly to `engine.GetSessionTailLines(id, n)` with only a `parsed > 0` guard:
```go
if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
    n = parsed
}
```
The `engine.GetSessionTailLines` function has no upper bound on `n` — it will return `lines[len(lines)-n:]` for any positive `n`, and if `n` is smaller than `len(lines)` it's fine, but the `strings.Split` on the full scrollback snapshot (up to 256 KiB) produces a slice that is then sliced by `n`. The memory cost is bounded by the 256 KiB scrollback cap, so this is not a DoS vector for the buffer read itself. However: the `[1..20]` clamp documented in the spec and implemented in `app.go` (lines 438-443) is **not enforced at the daemon HTTP route**. Any process on the same machine that can reach the Unix socket can request `?n=1000000` and get back a full 256 KiB of text in a single JSON response, repeated at whatever call rate it likes. This bypasses the intent of the spec clamp. BLOCKER because the daemon HTTP API is a defined trust boundary and the contract is violated.

**Fix:** Add an upper-bound clamp in the handler itself, matching `app.go`:
```go
func (a *API) handleGetSessionTailLines(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    n := 4
    if nStr := r.URL.Query().Get("n"); nStr != "" {
        if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
            n = parsed
        }
    }
    if n > 20 { n = 20 }  // mirror app.go clamp
    lines := a.engine.GetSessionTailLines(id, n)
    // ...
}
```

---

### CR-03: `usePreviewPoller` Overwrites Entire Tails Map — Stopped-Session Previews Flicker to "No Output Yet"

**File:** `frontend/src/components/Hub/HubPanel.tsx:80-82`
**Issue:** On every 3-second tick, the poller replaces the entire `tails` Map with a new one built only from `localSessions`:
```ts
const localSessions = sessions.filter((s) => !s.hostname || s.hostname === '')
// ...
setTails(new Map(localSessions.map((s, i) => [s.id, results[i]])))
```
When a session transitions to `state === 'stopped'`, it remains in `sessions` (the Hub keeps stopped cards visible), and it remains in `localSessions` (it is still local). However, `GetSessionTailLines` for a stopped session returns an empty slice once the hub's scrollback is cleared after session cleanup, OR may return the engine's `nil` path if the hub was removed. The engine's `GetSessionTailLines` returns `nil` when `manager.Get(id)` returns `!ok` — i.e., after `KillSession` removes the hub — which then falls through to `catch(() => [] as string[])` in the poller. So after kill+cleanup the card's preview entry becomes `[]` (empty array), rendering "No output yet" — overwriting the last-seen output that was displayed correctly before the session stopped.

More critically: if a session is killed while the Hub tab is active, the poller's next tick sets that session's entry to `[]` rather than retaining the last known value. The correct behavior is to retain the last good snapshot for stopped sessions (stop polling them, keep the value). The current design has no such retention.

**Fix:** In `setTails`, merge new results into the previous map rather than replacing it wholesale, and skip entries where the session no longer has a hub (i.e., treat `[]` from a stopped session as "keep previous"):
```ts
setTails(prev => {
  const next = new Map(prev)
  localSessions.forEach((s, i) => {
    const lines = results[i]
    // Only update if we got real lines, or if no prior value exists
    if (lines.length > 0 || !prev.has(s.id)) {
      next.set(s.id, lines)
    }
  })
  return next
})
```
Alternatively, filter `localSessions` to exclude stopped sessions (only poll running ones), retaining prior map entries for stopped sessions untouched.

---

## Warnings

### WR-01: `handleAssignGroup` Callback Does Not Enforce Single-Group Membership for Drop-on-"Other"

**File:** `frontend/src/components/Hub/HubPanel.tsx:224-228`
**Issue:** Both `handleDropOnGroup` and `handleAssignGroup` correctly call `assignToGroup` (which removes the key from all other groups before adding it to the target) or `removeFromGroup`. However `assignToGroup` calls `saveGroups` internally — and so does `removeFromGroup`. This means **two** `localStorage.setItem` calls happen per assignment: once inside the lib function, once... no, actually the lib functions call `saveGroups` themselves and the callbacks pass back the new array via `setGroupDefs`. This is correct. The actual problem is different: `assignToGroup` in `hubGroups.ts` line 41-48 correctly removes the key from all groups before adding it to the target group, enforcing single-group membership. However, when `groupId === '__other__'` in `handleAssignGroup` (line 226), it calls `removeFromGroup(prev, mKey)` — this is correct for "remove from all groups". But `removeFromGroup` does NOT add the key to any group, so "Other" is not a real group in the data model. The risk is the `assignToGroup('__other__', ...)` path in `handleDropOnGroup` at line 220: it routes to `removeFromGroup`, which is correct. This appears to be handled properly.

**Real issue:** The `handleDropOnGroup` callback (line 218-222) passes `groupId` which can be `null` from the "All" sidebar item (the drop handler in `GroupSidebarItem` returns early when `id === null` at line 106, so null is never reached here). However, the `handleAssignGroup` callback (line 224-228) is called from `SessionCard` with `groupId` from the overflow menu — where `'__other__'` maps to "Remove from group". A session can be in **zero named groups** (in "Other"), and clicking "Other (default)" from its own overflow menu when it is already in "Other" has no visible effect — that is fine. But clicking "Remove from group" (shown only when `isInNamedGroup === true`, line 269) and also clicking "Other (default)" (always shown, lines 261-265) **both** call `handleAssign('__other__')`. The "Other (default)" item is always visible even for sessions not in any named group. This means a user clicking "Other (default)" on an already-ungrouped session calls `removeFromGroup` which iterates all groups filtering the key — a no-op but also a wasted `saveGroups` write.

**Severity note:** This is a UX bug (duplicate menu item for sessions not in a group) rather than a data corruption bug. Downgraded to WARNING.

**Fix:** In `SessionCard`, hide the "Other (default)" menu item when `!isInNamedGroup`:
```tsx
{isInNamedGroup && (
  <button ... onClick={() => handleAssign('__other__')}>
    Other (default)
  </button>
)}
```
And remove the duplicate "Remove from group" section that follows — the "Other (default)" item already serves that function.

---

### WR-02: `hub-card--dragging` CSS Class Referenced in TSX but Not Defined in `style.css`

**File:** `frontend/src/components/Hub/SessionCard.tsx:211`, `frontend/src/style.css`
**Issue:** `SessionCard.tsx` applies the class `hub-card--dragging` to the article element while the card is being dragged (`isDragging === true`). A search of `style.css` finds no `.hub-card--dragging` rule. The class is applied but has no effect — the drag-state visual affordance specified in the UI-SPEC (opacity change, cursor feedback) is silently absent.

**Fix:** Add the missing rule in `style.css` near the other `.hub-card--*` modifiers:
```css
.hub-card--dragging {
  opacity: 0.6;
  cursor: grabbing;
}
```

---

### WR-03: `hub-card__menu-item--header` and `hub-card__menu-item--group` Modifier Classes Absent from `style.css`

**File:** `frontend/src/components/Hub/SessionCard.tsx:248,253,263`, `frontend/src/style.css`
**Issue:** `SessionCard.tsx` applies two modifier classes to overflow menu items:
- `hub-card__menu-item hub-card__menu-item--header` (line 248, the "Move to group" non-interactive header)
- `hub-card__menu-item hub-card__menu-item--group` (lines 253, 263, interactive group buttons)

`style.css` defines `.hub-card__menu-item` and `.hub-card__menu-item--sub` (line 4928) but neither `--header` nor `--group` variants. Without these styles, the "Move to group" header text is not visually distinguished from the clickable group items below it, and the "Other (default)" and named group items lack any differentiation from the generic menu item style. This is a visual-quality defect where the non-interactive header looks clickable.

**Fix:** Add the missing rules:
```css
.hub-card__menu-item--header {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--hub-text-muted);
  cursor: default;
  pointer-events: none;
}
.hub-card__menu-item--group {
  padding-left: 12px; /* indent under header */
}
```

---

### WR-04: `usePreviewPoller` Includes Remote Session IDs in `sessionIdKey` Dep — Wasted Poll Iterations

**File:** `frontend/src/components/Hub/HubPanel.tsx:65, 73`
**Issue:** `usePreviewPoller` receives `allSessions` (local + remote merged, line 190 of HubPanel). The stable dep key is:
```ts
const sessionIdKey = sessions.map((s) => s.id).join(',')
```
This includes remote session IDs. When remote peers are polled (every 30s), `allSessions` changes, `sessionIdKey` changes, and the preview poller effect re-runs — cancelling the current interval and starting a new one. The actual `localSessions` filter inside `poll()` correctly excludes remotes, so no incorrect tail fetch is made. However, the effect re-runs unnecessarily every time remote sessions change (every 30s), briefly resetting the 3-second interval timer. This is a robustness defect: the interval timer resets are wasteful and could cause mini-previews to skip a tick.

**Fix:** Build `sessionIdKey` only from local sessions:
```ts
const sessionIdKey = sessions
  .filter((s) => !s.hostname || s.hostname === '')
  .map((s) => s.id)
  .join(',')
```

---

### WR-05: `sidebarCollapsed` Initialization Reads `localStorage` Without a Try/Catch

**File:** `frontend/src/components/Hub/HubPanel.tsx:155-157`
**Issue:** The `sidebarCollapsed` state is initialized with a lazy initializer that calls `localStorage.getItem` directly:
```ts
const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(
  () => localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
)
```
`localStorage.getItem` can throw `SecurityError` when the browser's storage is disabled (e.g., private browsing with "block all cookies" in some browsers, or WebView storage quota exhausted). The `loadGroups()` call on line 151 wraps its `localStorage` usage in try/catch (hubGroups.ts line 17-21), but this initialization does not. A storage error during HubPanel mount would propagate as an unhandled exception and crash the Hub tab render.

**Fix:**
```ts
const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(() => {
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
  } catch {
    return false
  }
})
```

---

## Info

### IN-01: `GetSessionTailLines` Engine Method Returns `nil` When Session Has No Hub — Inconsistent with `[]string{}` Nil-Guard at API Layer

**File:** `internal/daemon/engine.go:533-535`, `internal/daemon/api.go:623-625`
**Issue:** `GetSessionTailLines` returns `nil` (not `[]string{}`) when `manager.Get(id)` returns `!ok`. The API handler then nil-guards this to `[]string{}` before JSON-encoding. The `app.go` Wails binding at line 444-447 also nil-guards. Both nil-guards are correct. However, the engine contract documents "Returns nil if the session has no hub" — this asymmetry between nil and empty slice is a gotcha for future callers who call the engine directly and forget the nil guard.

**Fix:** Return `[]string{}` directly from `GetSessionTailLines` on the no-hub path (defensive coding, matches the "never return nil to frontend" requirement in the phase spec):
```go
if !ok {
    return []string{}
}
```

---

### IN-02: `memberKey` Separator `:::` Is Not Escaped — Names or WorkDirs Containing `:::` Would Collide

**File:** `frontend/src/lib/hubGroups.ts:12-14`
**Issue:** The membership key is `"${name}:::${workDir}"`. If a session name contains the literal substring `:::`, two sessions could produce the same key for different (name, workDir) combinations. Example: session named `"foo:::bar"` with `workDir=""` produces the same key as session named `"foo"` with `workDir="::bar"` (after `__nodir__` substitution is not applied here since workDir is non-empty). In practice, session names and working-directory paths are unlikely to contain `:::` since it is not a valid path character on any platform and the name input is free-form but user-set. However the invariant is fragile.

**Fix (low urgency):** Use a more robust separator or encode the fields:
```ts
export function memberKey(name: string, workDir: string): string {
  // Use a separator that cannot appear in either field.
  // U+001F (Unit Separator) is a C0 control character that no terminal session
  // name or filesystem path would ever contain.
  return `${name}\x1f${workDir || '__nodir__'}`
}
```

---

### IN-03: `adaptRemoteSession` Always Sets `exitCode` and `duration` as `undefined` — `SessionCard` Will Show Running Duration for Remote Sessions

**File:** `frontend/src/lib/remoteAdapter.ts:8-23`
**Issue:** Remote sessions are adapted to `SessionInfo` with `createdAt: new Date().toISOString()`. Because `state` is always `'running'` and `duration` is `undefined`, `SessionCard` computes uptime as `formatUptime(createdAt)` — which uses the timestamp of the moment the adaptation ran. Since `adaptAllRemoteSessions` is called inside the 30-second remote-poll cycle, remote cards will show "0m" for 29 seconds after each poll, then briefly show "0m" again when the next poll fires. The uptime display for remote sessions is therefore always near-zero and meaningless (it reflects the time since the last remote poll, not actual session uptime).

This is cosmetic / misleading UX rather than a correctness defect. Remote sessions do not carry `createdAt` in the `RemoteSession` wire type (as seen in `App.d.ts` line 104-110), so there is no good fix without a wire-format change.

**Fix (low urgency):** Use a sentinel value or omit uptime for remote sessions by checking `session.hostname`:
```tsx
// In SessionCard.tsx
const timeText =
  hostname && hostname !== ''
    ? '' // remote sessions — no reliable createdAt
    : session.state === 'stopped' && duration !== undefined && duration !== null
    ? formatDuration(duration)
    : formatUptime(createdAt)
```

---

_Reviewed: 2026-06-16_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
