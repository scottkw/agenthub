# Phase 132: Unified Grid + Mini Preview + Named Groups - Research

**Researched:** 2026-06-16
**Domain:** React/TypeScript frontend UI, Go Wails bindings, BEM CSS, HTML5 drag-and-drop, localStorage
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Mini preview = throttled snapshot of the session's recent output tail. NEVER a live xterm per card. Performance constraint is non-negotiable (CARD-07).
- Group membership key = session name + working directory. Survives session-id churn across restarts; unmatched sessions fall to a default lane (GROUP-04).
- Remote sessions reuse the locked Phase 122 design (daemon proxy + join-code exchange). No new remote-access architecture.

### Claude's Discretion
All implementation choices — discuss phase was skipped per user setting.

### Deferred Ideas (OUT OF SCOPE)
Structured "agent suggests options" multi-select (#78) deferred to #93.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CARD-07 | Each card shows a mini terminal preview of the session's recent output tail | Wave-0 Go backend gap — `GetSessionTailLines` RPC must be added; scrollback buffer is in place |
| GRID-03 | Collapsible group sidebar with per-group running/total counts and needs-input badge | New `GroupSidebar.tsx` + `hubGroups.ts`; `GroupSidebarItem` sub-component |
| GRID-07 | Grid includes both local and remote tailnet/web-shared peer sessions in one unified view | `GetRemoteSessionsWithMeta` already exists; `adaptRemoteSession` helper needed |
| GROUP-01 | User can create named groups | Inline create flow in group sidebar; `hubGroups.ts` CRUD |
| GROUP-02 | User can assign cards to a group via drag-and-drop or per-card "Move to group" menu | HTML5 drag-and-drop; card overflow menu; `hubGroups.ts` membership update |
| GROUP-03 | Group definitions and membership persist locally (localStorage) | `agenthub:hubGroups:v1` key; same pattern as sidebar collapsed state |
| GROUP-04 | Group membership keys off session name + workDir; unmatched sessions fall to default lane | `${name}:::${workDir}` key in `HubGroupDef.memberKeys`; "Other" group as built-in |
</phase_requirements>

---

## Summary

Phase 132 extends the Phase 131 Hub with three interlocking capabilities: throttled terminal output snapshots on every card (CARD-07), remote tailnet sessions merged into the local grid (GRID-07), and a collapsible named-group sidebar with drag-and-drop card assignment (GROUP-01..04).

**The most important discovery is a Wave-0 Go backend gap:** there is no `GetSessionTailLines` (or equivalent) RPC anywhere in `app.go`, `internal/daemon/api.go`, `internal/daemon/client.go`, or `App.d.ts`. The data infrastructure to support it exists — every live session has an in-memory `relay.Hub` with a `Scrollback` ring buffer (256 KiB) that already accumulates raw PTY output frames — but no HTTP endpoint, no daemon client method, and no Wails-bound Go function exposes that buffer as plain text lines. Wave 0 must add this before the `MiniPreview` component can work.

The remote session plumbing is already wired: `GetRemoteSessionsWithMeta()` exists as a Wails RPC, `remotePeers` state is already managed in `App.tsx`, and the `RemoteSession` / `RemotePeerSessions` types are defined. Phase 132 needs only an `adaptRemoteSession` helper that maps the `RemoteSession` shape to `SessionInfo` for the unified grid — no new backend work.

The localStorage pattern for group persistence is established by `Sidebar.tsx` (key `sidebar-collapsed`) and `NewSessionModal.tsx` (keys `last-dir`, `agent-args-*`). The test setup in `test-setup.ts` already polyfills `localStorage` for vitest. No new test infrastructure is needed for localStorage-based tests.

HTML5 native drag-and-drop is the correct approach. The codebase already uses React-native `onDragOver`/`onDrop`/`onDragLeave` event handlers in `FileBrowserTab.tsx` (for file upload drops) — this is the established drag pattern. No third-party DnD library is used or needed.

**Primary recommendation:** Add `GetSessionTailLines` Go RPC in Wave 0 (3 layers: daemon API route, daemon client, app.go binding + App.d.ts update), then build frontend components in Waves 1-3, then CSS tokens + integration in Wave 4.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Terminal output tail snapshot | Go backend (relay.Hub scrollback) | Wails RPC binding (app.go) | PTY output lives in `relay.Hub.scrollback` (256 KiB ring buffer) — only the Go layer can access it; must be exposed via new RPC |
| ANSI stripping for preview text | Go backend (new helper in app.go or engine) | — | Stripping is cheapest close to source; the frontend `stripAnsi` util exists but requiring the JS layer to strip every poll tick is wasteful |
| Mini preview polling | Frontend (HubPanel / usePreviewPoller hook) | — | One shared 3-second interval drives all cards; interval is gated on Hub being the active tab |
| Remote session adaptation | Frontend (new `adaptRemoteSession` helper) | — | Shape mapping from `RemoteSession` to `SessionInfo` is pure data transformation; no backend work needed |
| Remote session discovery | Go backend (existing `GetRemoteSessionsWithMeta`) | App.tsx 30s poll | Already implemented; HubPanel needs to receive merged sessions as a prop |
| Named group CRUD + persistence | Frontend (`hubGroups.ts` + localStorage) | — | localStorage is the established pattern for layout state; no backend required |
| Group membership key derivation | Frontend (`hubGroups.ts`) | — | Key = `${name}:::${workDir}` is a pure string derivation |
| Drag-and-drop group assignment | Frontend (SessionCard drag source, GroupSidebar drop target) | — | HTML5 native DnD; no backend round-trip needed |
| Group sidebar collapsed state | Frontend (HubPanel state → CSS class) | — | Mirrors Sidebar.tsx collapsed-state pattern; localStorage if needed |

---

## Standard Stack

### Core (all already installed — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 18.x | Component rendering | Already the app's UI framework [VERIFIED: frontend/package.json] |
| TypeScript | 5.9.x | Type safety | Already the project language [VERIFIED: frontend/package.json] |
| @heroicons/react | installed | Icons (Bars3Icon drag handle, EllipsisHorizontalIcon menu, ChevronLeft/RightIcon sidebar toggle, PauseCircleIcon needs-input badge) | Already used throughout Hub [VERIFIED: SessionCard.tsx, Sidebar.tsx] |
| Wails v2 | installed | Desktop RPC bridge | App framework [VERIFIED: app.go] |

### Supporting (already installed)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| vitest | ^4.1.0 | Unit tests | All component and lib tests [VERIFIED: frontend/package.json] |
| jsdom | ^29.0.0 | DOM environment | Already configured [VERIFIED: vite.config.ts] |

**No new packages required.** All new capabilities use existing infrastructure.

---

## Package Legitimacy Audit

No new external packages are installed in Phase 132. All libraries are already in `package.json`.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
App.tsx (Hub active)
    │
    ├─ polls ListSessions() every 3s        → hubSessions (local)
    ├─ polls GetRemoteSessionsWithMeta() every 30s → remotePeers
    │
    └─ renders <HubPanel sessions={merged} remoteSessions={remotePeers} ...>
            │
            ├─ merges local + adapted remote sessions
            │   └─ adaptRemoteSession(peer, session) → SessionInfo[]
            │
            ├─ HubFilterBar (unchanged from Phase 131)
            │
            ├─ GroupSidebar (new)
            │   ├─ reads groupDefs from hubGroups.ts (localStorage)
            │   ├─ computes per-group running/total/needs-input counts
            │   ├─ fires onGroupSelect → HubPanel activeGroup state
            │   └─ fires onCreateGroup, onAssignGroup → hubGroups.ts CRUD
            │
            ├─ usePreviewPoller (new hook — single shared 3s interval)
            │   └─ batch-calls GetSessionTailLines for all visible sessions
            │       └─ Wails RPC → app.go GetSessionTailLines(ids, n)
            │           └─ engine.Manager().Get(id).ScrollbackSnapshot()
            │               → strip ANSI + split on \n → last N lines
            │
            └─ SessionCardGrid (extended with groupDefs prop)
                    └─ groups sessions by named group (override workDir grouping)
                           └─ SessionCard (extended)
                               ├─ drag handle (onDragStart)
                               ├─ overflow menu (Move to group)
                               └─ ROW 6: MiniPreview (plain text, no xterm)
```

### Recommended Project Structure

```
frontend/src/
├─ components/Hub/
│   ├─ HubPanel.tsx          (modified: add remoteSessions prop, group sidebar, usePreviewPoller)
│   ├─ SessionCard.tsx       (modified: add ROW 6 MiniPreview, drag handle, overflow menu)
│   ├─ SessionCardGrid.tsx   (modified: accept groupDefs + onAssignGroup props)
│   ├─ MiniPreview.tsx       (new: plain text 56px preview pane — CARD-07)
│   ├─ GroupSidebar.tsx      (new: collapsible sidebar + GroupSidebarItem — GRID-03)
│   ├─ HubFilterBar.tsx      (unchanged)
│   ├─ HubEmptyState.tsx     (unchanged)
│   └─ InlineSessionName.tsx (unchanged)
└─ lib/
    ├─ hubGroups.ts          (new: HubGroupDef CRUD + localStorage + membership key)
    ├─ hubStatus.ts          (unchanged)
    └─ remoteSession.ts      (unchanged — adaptRemoteSession is a new function here or in a new remoteAdapter.ts)
```

### Anti-Patterns to Avoid

- **Per-card `setInterval` for preview polling:** Forbidden. One shared timer in `HubPanel` (or `usePreviewPoller` hook). Each card receives tail lines as a prop.
- **Mounting an xterm instance per card:** Absolutely forbidden (CARD-07 non-negotiable). `MiniPreview` renders plain `<span>` text lines only.
- **Loading remote sessions in a new separate poll:** Reuse the existing `remotePeers` state from App.tsx. HubPanel receives it as a prop; App.tsx already polls every 30s when the remote tab is active. For Phase 132, the Hub also needs remote sessions — the simplest approach is to make App.tsx poll remote sessions when EITHER remote or Hub is active (or always at a low cadence).
- **Polling tail lines for remote sessions:** Remote sessions have no local scrollback buffer. The preview for remote sessions must always show "No output yet" immediately.
- **`node:path` import in frontend code:** Already called out in `SessionCardGrid.tsx` — use the inline `basename` helper pattern.

---

## Wave-0 Backend Gap: `GetSessionTailLines` RPC

**This is the primary Wave-0 blocker for CARD-07.**

### What exists (confirmed by codebase inspection)

| Layer | Status | Detail |
|-------|--------|--------|
| `relay.Scrollback` ring buffer | EXISTS [VERIFIED: internal/relay/scrollback.go] | 256 KiB max, `Snapshot()` returns raw bytes |
| `relay.Hub.ScrollbackSnapshot()` | EXISTS [VERIFIED: internal/relay/hub.go line 204] | Returns the full scrollback byte slice |
| `relay.HubManager.Get(id)` | EXISTS [VERIFIED: internal/relay/manager.go line 41] | Returns `(*Hub, bool)` for a session ID |
| `engine.Manager()` | EXISTS [VERIFIED: internal/daemon/engine.go line 1034] | Returns `*relay.HubManager` |
| `relay.MsgOutput` byte | EXISTS, = `0x01` [VERIFIED: internal/relay/protocol.go line 12] | Framing prefix byte; must be stripped when parsing scrollback |

### What does NOT exist (confirmed by exhaustive search)

- No `/sessions/{id}/tail` HTTP route in `internal/daemon/api.go` [VERIFIED]
- No `GetSessionTailLines` or equivalent method in `internal/daemon/client.go` [VERIFIED]
- No `GetSessionTailLines` binding in `app.go` [VERIFIED]
- No declaration in `frontend/src/wailsjs/go/main/App.d.ts` [VERIFIED]

### Three-layer addition required

**Layer 1 — daemon HTTP route** (`internal/daemon/api.go`):

Add `GET /sessions/{id}/tail?n=4` route. Handler:
1. Calls `a.engine.GetSessionTailLines(id, n)`
2. Returns `{"lines": ["...", "...", ...]}` JSON

**Layer 2 — engine method** (`internal/daemon/engine.go`):

```go
// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer. ANSI escape sequences are stripped before splitting.
// Returns nil if the session has no hub (not yet started or already stopped).
func (e *SessionEngine) GetSessionTailLines(id string, n int) []string {
    hub, ok := e.manager.Get(id)
    if !ok {
        return nil
    }
    raw := hub.ScrollbackSnapshot()
    // Strip MsgOutput framing bytes (0x01 prefix per frame boundary)
    // and ANSI escape sequences, then split on newlines, return last n.
    text := stripANSI(stripFramingBytes(raw))
    lines := splitLines(text)
    if len(lines) > n {
        lines = lines[len(lines)-n:]
    }
    return lines
}
```

The `stripANSI` helper uses a regexp: `\x1b\[??[0-9;]*[a-zA-Z]` (same pattern as the frontend `stripAnsi.ts`). Note: the scrollback stores raw PTY output including ANSI escape sequences — the terminal agents (Claude Code, opencode, etc.) emit heavily ANSI-laden text. Stripping is mandatory for a readable plain-text preview.

The framing byte strip: scrollback stores relay frames (each frame = `[0x01 | payload bytes]`). Strip all `0x01` bytes before parsing as text. This matches the pattern used in `engine_test.go` line 467 where tests strip `relay.MsgOutput` from scrollback snapshots. [VERIFIED: internal/daemon/engine_test.go line 464-468]

**Layer 3 — app.go binding** (`app.go`):

```go
// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences stripped.
// Returns an empty slice if the session has no scrollback (e.g. remote sessions).
// Used by the Hub mini-preview poller (CARD-07). n is clamped to [1..20].
func (a *App) GetSessionTailLines(id string, n int) []string {
    if a.client == nil {
        return []string{}
    }
    if n < 1 { n = 1 }
    if n > 20 { n = 20 }
    lines, err := a.client.GetSessionTailLines(id, n)
    if err != nil || lines == nil {
        return []string{}
    }
    return lines
}
```

**Layer 4 — App.d.ts update**:

```typescript
export function GetSessionTailLines(id: string, n: number): Promise<string[]>
```

### Alternative: batch endpoint

The UI-SPEC says "planner decides" between `GetAllSessionTailLines` (one call for all sessions) and N individual calls. Given the 3-second poll cadence and typically small session counts (< 20), N individual calls batched via `Promise.all` is simpler and avoids a new batch-response type. The planner should choose based on performance profile, but N individual calls should work fine for typical use.

---

## Remote Sessions in the Unified Grid (GRID-07)

### Existing plumbing (all VERIFIED from codebase)

| Component | Status | Location |
|-----------|--------|----------|
| `GetRemoteSessionsWithMeta()` Wails RPC | EXISTS | `app.go` line 1144 |
| `remotePeers` state in App.tsx | EXISTS | `App.tsx` line 190 |
| Remote poll (30s, when remote tab active) | EXISTS | `App.tsx` line 933-960 |
| `RemotePeerSessions` / `RemoteSession` types | EXISTS | `RemoteSessionsPanel.tsx` lines 1-16 |
| `GlobeAltIcon` + hostname origin marker in `SessionCard.tsx` | EXISTS | `SessionCard.tsx` lines 179-183 |

### What needs to be added

**`adaptRemoteSession` helper** (new function in `frontend/src/lib/remoteSession.ts` or a new `remoteAdapter.ts`):

```typescript
// Source: derived from UI-SPEC §4 Remote Sessions table (VERIFIED: 132-UI-SPEC.md)
import type { RemotePeerSessions, RemoteSession } from '../components/RemoteSessionsPanel'
import type { SessionInfo } from '../wailsjs/go/main/App'

export function adaptRemoteSession(
  peer: RemotePeerSessions,
  session: RemoteSession,
): SessionInfo {
  return {
    id: session.id,
    name: session.name,
    cli: session.cliType,
    state: 'running', // remote sessions with no status → conservative 'running'
    status: session.status || 'running',
    createdAt: new Date().toISOString(),
    hostname: peer.hostname,       // non-empty → GlobeAltIcon + hostname in SessionCard
    webEnabled: true,              // remote implies web-share active
    viewerCount: 0,
    exitCode: undefined,
    duration: undefined,
    workDir: '',                   // remote sessions have no local workDir → fall into "Other"
    homeDir: false,
    filesWrite: false,
  }
}
```

The `SessionInfo` type in `App.d.ts` declares `exitCode?: number` and `duration?: number` as optional — `undefined` is valid. [VERIFIED: frontend/src/wailsjs/go/main/App.d.ts]

**Remote sessions polling when Hub is active:** App.tsx currently polls `GetRemoteSessionsWithMeta` only when `activeId === REMOTE_SESSIONS_TAB.id`. Phase 132 needs remote sessions when `activeId === HUB_TAB.id` too. The simplest fix: extend the existing `useEffect` deps to include `HUB_TAB.id` in the active-check condition. The 30-second cadence is appropriate for remote sessions (tailnet probe is expensive).

**Merge in HubPanel:** `HubPanel` receives a new `remoteSessions` prop (the already-adapted `SessionInfo[]`). The merged list is `[...sessions, ...remoteSessions]`. This merged list is passed to `filterSessions` and then to `SessionCardGrid`. Mini preview for remote sessions: the `usePreviewPoller` skips remote sessions (identified by `session.hostname !== ''`) and the `MiniPreview` component renders "No output yet" state immediately.

---

## Named Groups Architecture (GROUP-01..04)

### `hubGroups.ts` — data model and localStorage persistence

```typescript
// Source: 132-UI-SPEC.md §3 (VERIFIED: 132-UI-SPEC.md)

export interface HubGroupDef {
  id: string           // random uuid — stable across restarts
  name: string         // user-chosen display name
  memberKeys: string[] // "${session.name}:::${session.workDir}" strings
}

const STORAGE_KEY = 'agenthub:hubGroups:v1'

// GROUP-04: membership key survives session-id churn
export function memberKey(name: string, workDir: string): string {
  /* GROUP-04: membership key = "${session.name}:::${session.workDir}" — survives session-id churn */
  return `${name}:::${workDir || '__nodir__'}`
}

export function loadGroups(): HubGroupDef[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as HubGroupDef[]
  } catch {
    return []
  }
}

export function saveGroups(groups: HubGroupDef[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(groups))
}
```

The `loadGroups` / `saveGroups` pattern follows `NewSessionModal.tsx` and `Sidebar.tsx` exactly. [VERIFIED: Sidebar.tsx line 34, NewSessionModal.tsx line 48]

The `test-setup.ts` localStorage polyfill already handles the jsdom environment. No additional test setup is needed. [VERIFIED: frontend/src/test-setup.ts]

### Grouping logic in SessionCardGrid

Phase 131's `groupByWorkDir` groups by `workDir`. Phase 132 overrides this when named groups exist. The new grouping logic:

1. For each session, compute `memberKey(session.name, session.workDir)`.
2. Check each `HubGroupDef.memberKeys` array.
3. First matching named group wins; if no match, session falls into built-in "Other" group.
4. Named groups appear in the sidebar and grid in definition order; "Other" is always last.

The `SessionCardGrid` receives `groupDefs: HubGroupDef[]` as a new prop. When `groupDefs.length === 0` (no named groups), it falls back to `groupByWorkDir` behavior (Phase 131 compatibility).

---

## HTML5 Drag-and-Drop Pattern

The codebase uses **native React HTML5 drag-and-drop** (not a third-party library). The pattern is established in `FileBrowserTab.tsx` using `onDragOver`, `onDragLeave`, `onDrop` React event handlers. [VERIFIED: FileBrowserTab.tsx lines 879-899]

For Phase 132, the drag flow is:

**Drag source (`SessionCard`):**
```typescript
// Source: 132-UI-SPEC.md §3 Drag-and-Drop (VERIFIED)
<article
  draggable="true"
  onDragStart={(e) => {
    e.dataTransfer.setData('text/plain', memberKeyForSession)
    e.dataTransfer.effectAllowed = 'move'
  }}
  onDragEnd={() => setIsDragging(false)}
  ...
>
```

**Drop target (`GroupSidebarItem`):**
```typescript
<li
  onDragOver={(e) => { e.preventDefault(); setIsDragOver(true) }}
  onDragLeave={() => setIsDragOver(false)}
  onDrop={(e) => {
    e.preventDefault()
    setIsDragOver(false)
    const key = e.dataTransfer.getData('text/plain')
    onDrop(groupDef.id, key)
  }}
  className={`hub__group-sidebar-item${isActive ? ' hub__group-sidebar-item--active' : ''}${isDragOver ? ' hub__group-sidebar-item--drag-over' : ''}`}
>
```

`e.preventDefault()` on `onDragOver` is mandatory to allow drops. This is the same pattern used in `FileBrowserTab.tsx` line 882. [VERIFIED: FileBrowserTab.tsx]

The drag transfer uses `text/plain` with the session's membership key (`${name}:::${workDir}`). This is sufficient since drag targets are within the same page — no cross-origin considerations.

---

## `usePreviewPoller` Hook Design

```typescript
// Source: 132-UI-SPEC.md §1 Mini Preview Interaction Contract (VERIFIED)
// Located in HubPanel.tsx (or extracted to lib/usePreviewPoller.ts)

function usePreviewPoller(
  sessions: SessionInfo[],
  isActive: boolean,
): Map<string, string[]> {
  const [tails, setTails] = useState<Map<string, string[]>>(new Map())

  useEffect(() => {
    if (!isActive || sessions.length === 0) return
    let cancelled = false

    async function poll() {
      // Only fetch for local sessions — remote sessions have no tail API
      const localIds = sessions
        .filter((s) => !s.hostname || s.hostname === '')
        .map((s) => s.id)

      if (localIds.length === 0) return

      // Individual calls via Promise.all — simpler than a batch endpoint
      // for typical session counts (< 20)
      const results = await Promise.all(
        localIds.map((id) =>
          GetSessionTailLines(id, 4).catch(() => [] as string[])
        )
      )
      if (!cancelled) {
        setTails(new Map(localIds.map((id, i) => [id, results[i]])))
      }
    }

    void poll()
    const interval = setInterval(() => void poll(), 3000)
    return () => { cancelled = true; clearInterval(interval) }
  }, [sessions, isActive])  // note: sessions dep — re-binds when session list changes

  return tails
}
```

`isActive` is derived from `activeId === HUB_TAB.id` in `HubPanel`'s props or a prop passed down from App.tsx. When the Hub tab is not active, the interval is not started (effect returns early). [VERIFIED requirement: 132-UI-SPEC.md §1]

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Scrollback ring buffer | Custom ring buffer | `relay.Scrollback` (256 KiB, already running) [VERIFIED: scrollback.go] | Already accumulating all PTY output |
| ANSI escape stripping | Custom regex | Pattern from `frontend/src/lib/stripAnsi.ts` (`/\x1b\[??[0-9;]*[a-zA-Z]/g`) [VERIFIED: stripAnsi.ts line 21] | Already tested and correct for this codebase's ANSI output |
| Remote session discovery | New tailnet probe | `GetRemoteSessionsWithMeta()` + `remotePeers` state [VERIFIED: App.tsx line 940] | Already implemented; just needs Hub to receive it |
| UUID generation for group IDs | Custom ID scheme | `crypto.randomUUID()` (browser built-in) | Available in Wails WebView context (Chromium) |
| localStorage serialization | Custom format | `JSON.stringify/parse` (same as `NewSessionModal.tsx`) [VERIFIED: NewSessionModal.tsx] | Already established pattern |
| Drag-and-drop library | react-dnd or dnd-kit | HTML5 native DnD via React event handlers [VERIFIED: FileBrowserTab.tsx] | Already the established pattern; no new dependency |

**Key insight:** All three capabilities (scrollback, remote sessions, localStorage) have their infrastructure already built. Phase 132 is almost entirely additive frontend work plus one Go backend surface.

---

## Common Pitfalls

### Pitfall 1: Scrollback Contains Relay Framing Bytes
**What goes wrong:** Parsed tail lines contain garbage characters (`\x01`) or split at wrong boundaries.
**Why it happens:** The `relay.Hub` scrollback stores framed bytes. Each PTY output chunk is stored as `[0x01 | payload...]` (see `hub.go` line 145: `frame := MakeOutputFrame(buf[:n])` → `h.scrollback.Append(frame)`). The `0x01` byte is `MsgOutput`.
**How to avoid:** Strip `0x01` bytes before parsing as text. Pattern from `engine_test.go` lines 464-468: `for _, b := range hub.ScrollbackSnapshot() { if b != relay.MsgOutput { collected.WriteByte(b) } }` [VERIFIED: engine_test.go]
**Warning signs:** Preview lines show `\x01` or control character artifacts.

### Pitfall 2: ANSI Sequences in Preview Text
**What goes wrong:** Terminal agents (Claude Code, opencode, Gemini CLI) emit heavily ANSI-laden output. Preview text shows garbled color codes.
**Why it happens:** PTY output is stored raw including ANSI escape sequences.
**How to avoid:** Strip ANSI after stripping framing bytes. Use Go regexp `\x1b\[??[0-9;]*[a-zA-Z]` (matches the frontend `stripAnsi.ts` pattern). Also strip OSC sequences (`\x1b]...\x07` or `\x1b]...\x1b\\`) — these appear in the scrollback from session title-setting and OSC 8 hyperlinks.
**Warning signs:** Preview shows `[32m` or `]0;` literals in text.

### Pitfall 3: usePreviewPoller Rebinds on Every Render
**What goes wrong:** The `sessions` dep in `usePreviewPoller` causes the effect to re-run on every 3-second `hubSessions` update, creating a polling storm.
**Why it happens:** `sessions` is a new array reference on every render. `useEffect` sees it as changed.
**How to avoid:** Depend on a stable derived value (e.g. session IDs joined as a string) rather than the sessions array directly. Or memoize the `localIds` array. The `DaemonManagerPanel` pattern uses the `activeId` dep which is stable.
**Warning signs:** `GetSessionTailLines` calls spike instead of settling at one batch per 3 seconds.

### Pitfall 4: Remote Sessions Polled Only When Remote Tab Is Active
**What goes wrong:** Hub shows no remote sessions because `remotePeers` is always empty when the user opens Hub directly.
**Why it happens:** App.tsx currently polls `GetRemoteSessionsWithMeta` only when `activeId === REMOTE_SESSIONS_TAB.id`. [VERIFIED: App.tsx line 935]
**How to avoid:** Extend the poll condition to `(activeId === REMOTE_SESSIONS_TAB.id || activeId === HUB_TAB.id)`. The 30-second cadence is fine.
**Warning signs:** Hub grid shows zero remote sessions even when tailnet peers are known to be running.

### Pitfall 5: Named Groups Overwrite WorkDir Groups Without Fallback
**What goes wrong:** When named groups are first created (empty `memberKeys`), all sessions fall into "Other" and the workDir grouping disappears.
**Why it happens:** The new grouping logic replaces `groupByWorkDir` entirely when any named group exists.
**How to avoid:** The "Other" group in the sidebar corresponds to the built-in unassigned bucket. Sessions not in any named group appear in "Other" — this is the correct behavior. But the sidebar should still show workDir-derived sub-buckets within "Other" (or collapse them all to "Other"). The UI-SPEC says "Other" is a built-in group (cannot rename/delete). WorkDir grouping in the grid is superseded by named-group display once any group exists.
**Warning signs:** User creates a group, and all other sessions disappear from the grid.

### Pitfall 6: Drag Handle Absolute Position Breaks Card Layout
**What goes wrong:** Drag handle and overflow menu button push card content down or overlap other rows.
**Why it happens:** `position: absolute` on `.hub-card__drag-handle` and `.hub-card__menu-btn` requires `.hub-card` to be `position: relative` (already set in Phase 131) and the inner content rows must not overlap the absolute-positioned elements.
**How to avoid:** Confirm `.hub-card` has `position: relative` (it does [VERIFIED: SessionCard.tsx line 141 renders `class="hub-card"` and Phase 131 CSS sets this]). The drag handle is `top: 8px; left: 8px` and menu btn is `top: 8px; right: 8px` — they float over ROW 1. Card content already has `padding: 12px 16px` from Phase 131, so the icons (16px) fit within the padding without overlap.
**Warning signs:** Card content shifts when hover state activates.

### Pitfall 7: `prefers-reduced-motion` Not Applied to Group Sidebar Transition
**What goes wrong:** Sidebar width transition plays even for users with reduced motion enabled.
**Why it happens:** CSS `transition: width 150ms ease` declared outside `@media (prefers-reduced-motion: no-preference)`.
**How to avoid:** Wrap ALL new transitions in the media query per the UI-SPEC Motion Contract.
**Warning signs:** `transition:` on `.hub__group-sidebar` outside a media query block.

### Pitfall 8: Needs-Input Badge Is Color-Only
**What goes wrong:** UAT failure — colorblind user cannot identify needs-input groups.
**Why it happens:** Amber badge rendered without the `PauseCircleIcon`.
**How to avoid:** Badge MUST include `PauseCircleIcon` (16px) alongside the count text. The icon carries the state; amber color is reinforcement only. [VERIFIED requirement: 132-UI-SPEC.md §Colorblind Mandate]
**Warning signs:** Badge renders as `<span className="hub__group-sidebar-item__needs-input-badge">1</span>` without an icon.

---

## Code Examples

### Scrollback Tail Lines — Go Implementation Pattern

```go
// Source: pattern from internal/daemon/engine_test.go lines 464-468 + internal/relay/protocol.go
// (VERIFIED: codebase)

// stripFramingBytes removes relay.MsgOutput (0x01) framing bytes from raw scrollback data.
func stripFramingBytes(data []byte) []byte {
    out := make([]byte, 0, len(data))
    for _, b := range data {
        if b != relay.MsgOutput { // relay.MsgOutput == 0x01
            out = append(out, b)
        }
    }
    return out
}

// ansiEscape matches common ANSI/VT100 sequences (CSI + OSC).
var ansiEscape = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[a-zA-Z]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)

func stripANSI(data []byte) string {
    return ansiEscape.ReplaceAllString(string(data), "")
}

// GetSessionTailLines returns the last n plain-text lines from the session's scrollback.
func (e *SessionEngine) GetSessionTailLines(id string, n int) []string {
    hub, ok := e.manager.Get(id)
    if !ok {
        return nil
    }
    raw := stripANSI(string(stripFramingBytes(hub.ScrollbackSnapshot())))
    lines := strings.Split(raw, "\n")
    // Remove empty trailing lines
    for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
        lines = lines[:len(lines)-1]
    }
    if len(lines) > n {
        lines = lines[len(lines)-n:]
    }
    return lines
}
```

### adaptRemoteSession

```typescript
// Source: 132-UI-SPEC.md §4 Remote Sessions (VERIFIED)
// In frontend/src/lib/remoteAdapter.ts (new file) or appended to remoteSession.ts

import type { RemotePeerSessions, RemoteSession } from '../components/RemoteSessionsPanel'
import type { SessionInfo } from '../wailsjs/go/main/App'

/* GRID-07: remote sessions adapted via adaptRemoteSession(); hostname != '' routes to GlobeAltIcon + hostname */
export function adaptRemoteSession(
  peer: RemotePeerSessions,
  session: RemoteSession,
): SessionInfo {
  return {
    id: session.id,
    name: session.name,
    cli: session.cliType,
    state: 'running',
    status: session.status || 'running',
    createdAt: new Date().toISOString(),
    hostname: peer.hostname,
    webEnabled: true,
    viewerCount: 0,
    workDir: '',
    homeDir: false,
    filesWrite: false,
  }
}

export function adaptAllRemoteSessions(peers: RemotePeerSessions[]): SessionInfo[] {
  return peers
    .filter((p) => p.reachable)
    .flatMap((p) => p.sessions.map((s) => adaptRemoteSession(p, s)))
}
```

### HubGroupDef CRUD (hubGroups.ts)

```typescript
// Source: 132-UI-SPEC.md §3 Named Groups (VERIFIED)
/* HUB-GROUPS-V1: localStorage key "agenthub:hubGroups:v1" — JSON array of HubGroupDef */
/* GROUP-04: membership key = "${session.name}:::${session.workDir}" — survives session-id churn */

const STORAGE_KEY = 'agenthub:hubGroups:v1'

export interface HubGroupDef {
  id: string
  name: string
  memberKeys: string[]
}

export function memberKey(name: string, workDir: string): string {
  return `${name}:::${workDir || '__nodir__'}`
}

export function loadGroups(): HubGroupDef[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as HubGroupDef[]) : []
  } catch {
    return []
  }
}

export function saveGroups(groups: HubGroupDef[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(groups))
}

export function createGroup(groups: HubGroupDef[], name: string): HubGroupDef[] {
  const def: HubGroupDef = { id: crypto.randomUUID(), name, memberKeys: [] }
  const updated = [...groups, def]
  saveGroups(updated)
  return updated
}

export function assignToGroup(
  groups: HubGroupDef[],
  groupId: string,
  key: string,
): HubGroupDef[] {
  // Remove from all groups first, then add to target
  const updated = groups.map((g) => ({
    ...g,
    memberKeys:
      g.id === groupId
        ? [...g.memberKeys.filter((k) => k !== key), key]
        : g.memberKeys.filter((k) => k !== key),
  }))
  saveGroups(updated)
  return updated
}

export function removeFromGroup(groups: HubGroupDef[], key: string): HubGroupDef[] {
  const updated = groups.map((g) => ({
    ...g,
    memberKeys: g.memberKeys.filter((k) => k !== key),
  }))
  saveGroups(updated)
  return updated
}
```

### New CSS Tokens (append to existing `:root` and `[data-ui-theme="light"]` blocks)

```css
/* Source: 132-UI-SPEC.md Design System tokens (VERIFIED) */
/* Append to existing :root block — do NOT duplicate existing tokens */
:root {
  /* CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */
  --hub-preview-bg: #0d0e17;
  --hub-preview-text: #8b92b3;
  --hub-preview-border: #1e2130;
  /* GRID-03: group sidebar */
  --hub-sidebar-bg: #13141f;
  --hub-sidebar-width: 200px;
  --hub-sidebar-collapsed-width: 32px;
  --hub-sidebar-item-active-bg: rgba(122,162,247,0.12);
  --hub-sidebar-item-hover-bg: rgba(122,162,247,0.07);
  /* COLORBLIND-SAFE: needs-input badge dark hex #f59e0b — reinforcement only; PauseCircleIcon carries the state */
  --hub-needs-input-badge-bg: rgba(245,158,11,0.18);
  --hub-needs-input-badge-text: #f59e0b;
  /* COLORBLIND-SAFE: drag-over border dark hex #7aa2f7 — border change (shape) is primary non-color cue */
  --hub-drag-over-border: #7aa2f7;
  --hub-drag-over-bg: rgba(122,162,247,0.08);
}

[data-ui-theme="light"] {
  --hub-preview-bg: #e8e8f0;
  --hub-preview-text: #6b6f8e;
  --hub-preview-border: #d1d1db;
  --hub-sidebar-bg: #eaeaef;
  --hub-sidebar-width: 200px;
  --hub-sidebar-collapsed-width: 32px;
  --hub-sidebar-item-active-bg: rgba(61,111,232,0.10);
  --hub-sidebar-item-hover-bg: rgba(61,111,232,0.05);
  /* COLORBLIND-SAFE: needs-input badge light hex #b45309 — WCAG AA 7.6:1 on white; icon carries state */
  --hub-needs-input-badge-bg: rgba(180,83,9,0.15);
  --hub-needs-input-badge-text: #b45309;
  /* COLORBLIND-SAFE: drag-over border light hex #3d6fe8 — border change (shape) is primary non-color cue */
  --hub-drag-over-border: #3d6fe8;
  --hub-drag-over-bg: rgba(61,111,232,0.06);
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hub grid shows only local sessions | Unified grid with remote sessions via adapter | Phase 132 | Remote sessions render via existing `GlobeAltIcon` origin marker path — no new card variant |
| Hub cards show no output preview | Throttled 4-line text snapshot (56px pane) | Phase 132 | MiniPreview is a new ROW 6 on SessionCard; never an xterm instance |
| Sessions organized by workDir only | Named groups (user-defined) + workDir fallback | Phase 132 | Group membership keyed on name+workDir survives session-id churn |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `crypto.randomUUID()` is available in the Wails WebView (Chromium) context without polyfill | hubGroups.ts code example | If unavailable, use a simple timestamp+random ID instead; trivially fixable |
| A2 | Stripping `0x01` bytes plus the ANSI regexp is sufficient to produce readable plain-text lines from the scrollback | GetSessionTailLines Go implementation | Some terminal output may use non-CSI ANSI sequences (OSC sequences not fully stripped); preview may show minor artifacts; non-blocking |
| A3 | Remote sessions poll should extend to Hub-active state (same 30s interval) | Remote Sessions pitfall | If 30s is too slow for users who want fresh remote sessions in the Hub, a shorter interval may be needed; design decision for planner |

---

## Open Questions

1. **Batch endpoint vs. N individual calls for `GetSessionTailLines`**
   - What we know: UI-SPEC says "planner decides." Individual calls via `Promise.all` are simpler. A batch endpoint `GetAllSessionTailLines(ids []string, n int) map[string][]string` avoids N round trips.
   - What's unclear: Session counts in practice. For < 20 sessions, N calls is fine; for > 50, a batch endpoint is better.
   - Recommendation: Implement as N individual calls (simpler Wave 0); if perf is a concern during UAT, add batch endpoint in the same phase.

2. **Remote sessions polling cadence when Hub is active**
   - What we know: Current remote poll is 30s gated on remote tab active.
   - What's unclear: Whether 30s is acceptable for Hub unified grid. A remote session going offline would take 30s to disappear from the Hub.
   - Recommendation: Keep 30s cadence (same as remote panel); extend the gate condition to `|| activeId === HUB_TAB.id`.

---

## Environment Availability

Step 2.6: SKIPPED — This phase is a frontend + Go RPC addition. No new external tools, services, or runtimes are required beyond what Phase 131 established (Node.js, pnpm, Go, Wails CLI — all confirmed available).

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0 |
| Config file | `frontend/vite.config.ts` (test block) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test:coverage` |
| Go tests | `go test ./internal/daemon/... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CARD-07 | MiniPreview renders plain text; no xterm; shows Loading/Empty states | unit | `pnpm test` | ❌ Wave 0 — Hub/MiniPreview.test.tsx |
| CARD-07 | `GetSessionTailLines` Go RPC returns plain-text lines (ANSI stripped, framing stripped) | unit (Go) | `go test ./internal/daemon/... -run TestGetSessionTailLines` | ❌ Wave 0 — engine_test.go addition |
| CARD-07 | usePreviewPoller: single interval, pauses when inactive, excludes remote sessions | unit | `pnpm test` | ❌ Wave 0 — Hub/HubPanel.test.tsx addition |
| GRID-03 | GroupSidebar renders group list with counts; collapsed state (32px); expanded (200px) | unit | `pnpm test` | ❌ Wave 0 — Hub/GroupSidebar.test.tsx |
| GRID-03 | Needs-input badge renders PauseCircleIcon + count (colorblind mandate) | unit | `pnpm test` | ❌ Wave 0 — Hub/GroupSidebar.test.tsx |
| GRID-07 | adaptRemoteSession maps RemoteSession + hostname → SessionInfo correctly | unit | `pnpm test` | ❌ Wave 0 — lib/remoteAdapter.test.ts |
| GRID-07 | Remote sessions in Hub grid: hostname non-empty → GlobeAltIcon origin marker | unit | `pnpm test` | ❌ Wave 0 — via SessionCard.test.tsx (already exists) |
| GROUP-01 | Create named group: inline input, Enter confirms, Escape cancels | unit | `pnpm test` | ❌ Wave 0 — Hub/GroupSidebar.test.tsx |
| GROUP-02 | Drag-and-drop: drop on group sidebar item assigns membership key | unit | `pnpm test` | ❌ Wave 0 — Hub/GroupSidebar.test.tsx |
| GROUP-02 | Per-card menu "Move to group" assigns membership key | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx addition |
| GROUP-03 | hubGroups.ts: loadGroups reads `agenthub:hubGroups:v1`; saveGroups persists | unit | `pnpm test` | ❌ Wave 0 — lib/hubGroups.test.ts |
| GROUP-03 | Groups survive Hub unmount + remount (localStorage round-trip) | unit | `pnpm test` | ❌ Wave 0 — lib/hubGroups.test.ts |
| GROUP-04 | memberKey: `${name}:::${workDir}` format; empty workDir → `__nodir__` | unit | `pnpm test` | ❌ Wave 0 — lib/hubGroups.test.ts |
| GROUP-04 | Sessions not in any group appear in "Other" | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCardGrid.test.tsx addition |

### Wave 0 Gaps

- [ ] `frontend/src/components/Hub/MiniPreview.test.tsx` — covers CARD-07 rendering states
- [ ] `frontend/src/components/Hub/GroupSidebar.test.tsx` — covers GRID-03, GROUP-01, GROUP-02 (drop)
- [ ] `frontend/src/lib/hubGroups.test.ts` — covers GROUP-03, GROUP-04
- [ ] `frontend/src/lib/remoteAdapter.test.ts` — covers GRID-07 adaptation
- [ ] Addition to `frontend/src/components/Hub/SessionCard.test.tsx` — covers GROUP-02 (menu), drag handle visibility
- [ ] Addition to `frontend/src/components/Hub/SessionCardGrid.test.tsx` — covers GROUP-04 "Other" fallback
- [ ] Addition to `internal/daemon/engine_test.go` — covers `GetSessionTailLines` ANSI stripping + framing strip

*(Existing Hub test files from Phase 131 are in place — only additions needed, not new infrastructure)*

---

## Security Domain

No new authentication, authorization, or capability logic is introduced. All new surfaces read from existing RPCs (`ListSessions`, `GetRemoteSessionsWithMeta`, new `GetSessionTailLines`). The terminal tail lines are read-only, non-interactive. localStorage data is user-owned with no secrets.

ASVS V5 input validation: the group name input must trim whitespace and reject empty strings before creating a group (same pattern as the inline session rename in `InlineSessionName.tsx`).

---

## Project Constraints (from CLAUDE.md)

- **React/TypeScript frontend** (confirmed from codebase — the Hub is React, not Vue)
- **`pnpm` preferred** as package manager
- **TypeScript:** `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types
- **Go:** `go fmt`, context-aware — new daemon methods follow existing engine patterns
- **Testing:** vitest (installed); 80%+ coverage in critical components
- **No new global installs** — Phase 132 adds zero new npm or Go dependencies
- **Python not relevant** — pure frontend + Go RPC addition

---

## Sources

### Primary (HIGH confidence)
- `internal/relay/scrollback.go` — Scrollback ring buffer (256 KiB, `Snapshot()` method) [VERIFIED: codebase]
- `internal/relay/hub.go` — `Hub.ScrollbackSnapshot()`, framing byte `MsgOutput = 0x01`, `Run()` accumulation loop [VERIFIED: codebase]
- `internal/relay/manager.go` — `HubManager.Get(id)` accessor [VERIFIED: codebase]
- `internal/daemon/engine.go` — `GetSessionTailLines` gap confirmed; `Manager()` accessor; `ListSessions()` pattern [VERIFIED: codebase]
- `internal/daemon/engine_test.go` lines 464-468 — Scrollback parsing pattern (strip MsgOutput, strip ANSI) [VERIFIED: codebase]
- `internal/daemon/api.go` — All HTTP routes (confirmed no `/sessions/{id}/tail`) [VERIFIED: codebase]
- `internal/daemon/client.go` — All DaemonClient methods (confirmed no tail method) [VERIFIED: codebase]
- `app.go` — All public Wails-bound methods (confirmed no `GetSessionTailLines`); `GetRemoteSessionsWithMeta()` exists [VERIFIED: codebase]
- `frontend/src/wailsjs/go/main/App.d.ts` — All exported TS bindings (confirmed no tail declaration) [VERIFIED: codebase]
- `frontend/src/components/Hub/HubPanel.tsx` — Current HubPanel shape, props, polling pattern [VERIFIED: codebase]
- `frontend/src/components/Hub/SessionCard.tsx` — Current card structure ROW 1-5, origin marker path [VERIFIED: codebase]
- `frontend/src/components/Hub/SessionCardGrid.tsx` — `groupByWorkDir`, `basename` helper [VERIFIED: codebase]
- `frontend/src/components/RemoteSessionsPanel.tsx` — `RemoteSession`, `RemotePeerSessions` types [VERIFIED: codebase]
- `frontend/src/lib/stripAnsi.ts` — ANSI regex pattern (`/\x1b\[??[0-9;]*[a-zA-Z]/g`) [VERIFIED: codebase]
- `frontend/src/lib/hubStatus.ts` — `deriveHubStatus`, `HubStatus` type [VERIFIED: codebase]
- `frontend/src/components/FileBrowserTab.tsx` lines 879-899 — Native HTML5 DnD pattern (`onDragOver`/`onDrop`) [VERIFIED: codebase]
- `frontend/src/components/Sidebar.tsx` — localStorage pattern (`STORAGE_KEY`, `useState` initializer) [VERIFIED: codebase]
- `frontend/src/test-setup.ts` — localStorage polyfill for vitest [VERIFIED: codebase]
- `frontend/src/App.tsx` lines 190, 909-923, 933-960 — `remotePeers` state, Hub poll, Remote poll [VERIFIED: codebase]
- `.planning/phases/132-unified-grid-mini-preview-named-groups/132-UI-SPEC.md` — Locked design contract [VERIFIED: planning docs]
- `.planning/phases/132-unified-grid-mini-preview-named-groups/132-CONTEXT.md` — Locked decisions [VERIFIED: planning docs]

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` — CARD-07, GRID-03, GRID-07, GROUP-01..04 definitions
- `.planning/STATE.md` — v3.6 Key Decisions

---

## Metadata

**Confidence breakdown:**
- Wave-0 backend gap (GetSessionTailLines): HIGH — exhaustively verified against all layers (api.go, client.go, app.go, App.d.ts)
- Scrollback buffer infrastructure: HIGH — read scrollback.go, hub.go, manager.go, engine_test.go directly
- Remote sessions plumbing: HIGH — read App.tsx, RemoteSessionsPanel.tsx, app.go GetRemoteSessionsWithMeta
- localStorage pattern: HIGH — read Sidebar.tsx, NewSessionModal.tsx, test-setup.ts
- HTML5 DnD pattern: HIGH — read FileBrowserTab.tsx directly
- Named group architecture: HIGH — derived from UI-SPEC + codebase conventions
- CSS tokens: HIGH — read existing style.css patterns + UI-SPEC token tables

**Research date:** 2026-06-16
**Valid until:** 2026-07-16 (stable codebase — no external dependency changes expected)
