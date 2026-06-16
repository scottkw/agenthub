# Phase 132: Unified Grid + Mini Preview + Named Groups - Pattern Map

**Mapped:** 2026-06-16
**Files analyzed:** 13 new/modified files
**Analogs found:** 13 / 13

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/types.go` | model | request-response | `internal/daemon/types.go` (StatusResponse) | exact |
| `internal/daemon/api.go` | route | request-response | `internal/daemon/api.go` handleGetSessionStatus | exact |
| `internal/daemon/engine.go` | service | CRUD | `internal/daemon/engine.go` GetSessionStatus | exact |
| `internal/daemon/client.go` | service | request-response | `internal/daemon/client.go` GetSessionStatus | exact |
| `app.go` | controller | request-response | `app.go` GetSessionStatus | exact |
| `frontend/src/wailsjs/go/main/App.d.ts` | config | request-response | `App.d.ts` GetSessionStatus declaration | exact |
| `frontend/src/lib/hubGroups.ts` | utility | CRUD | `frontend/src/components/Sidebar.tsx` (localStorage pattern) | role-match |
| `frontend/src/lib/remoteAdapter.ts` | utility | transform | `frontend/src/components/RemoteSessionsPanel.tsx` (RemoteSession types) | role-match |
| `frontend/src/components/Hub/MiniPreview.tsx` | component | request-response | `frontend/src/components/Hub/SessionCard.tsx` (ROW pattern) | role-match |
| `frontend/src/components/Hub/GroupSidebar.tsx` | component | event-driven | `frontend/src/components/Sidebar.tsx` (collapsed state + toggle) | role-match |
| `frontend/src/components/Hub/SessionCard.tsx` | component | event-driven | `frontend/src/components/Hub/SessionCard.tsx` (existing) | exact |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | component | CRUD | `frontend/src/components/Hub/SessionCardGrid.tsx` (existing) | exact |
| `frontend/src/components/Hub/HubPanel.tsx` | component | request-response | `frontend/src/components/Hub/HubPanel.tsx` (existing) | exact |
| `frontend/src/App.tsx` | controller | event-driven | `frontend/src/App.tsx` Hub poll useEffect (lines 907-923) | exact |
| `frontend/src/style.css` | config | n/a | `frontend/src/style.css` existing `:root` + `[data-ui-theme="light"]` blocks | exact |

---

## Pattern Assignments

### `internal/daemon/types.go` — new TailLinesResponse struct

**Analog:** `internal/daemon/types.go` StatusResponse (lines 57-60)

**Core type pattern** (lines 57-60):
```go
// StatusResponse is the response body for GET /sessions/{id}/status.
type StatusResponse struct {
	Status string `json:"status"`
}
```

**New type to add** — append after StatusResponse:
```go
// TailLinesResponse is the response body for GET /sessions/{id}/tail.
type TailLinesResponse struct {
	Lines []string `json:"lines"`
}
```

---

### `internal/daemon/api.go` — new GET /sessions/{id}/tail route

**Analog:** `internal/daemon/api.go` handleGetSessionStatus (lines 103-107, 604-608)

**Route registration pattern** (lines 103, 604-608):
```go
// In setupRoutes (line 103 area):
a.mux.HandleFunc("GET /sessions/{id}/status", a.handleGetSessionStatus)
// ADD:
a.mux.HandleFunc("GET /sessions/{id}/tail", a.handleGetSessionTailLines)

// Handler (mirrors handleGetSessionStatus):
func (a *API) handleGetSessionStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := a.engine.GetSessionStatus(id)
	writeJSON(w, http.StatusOK, StatusResponse{Status: s})
}
```

**writeJSON helper** (lines 459-464):
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
```

**New handler to add:**
```go
func (a *API) handleGetSessionTailLines(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n := 4 // default; parse ?n= query param if present
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}
	lines := a.engine.GetSessionTailLines(id, n)
	if lines == nil {
		lines = []string{}
	}
	writeJSON(w, http.StatusOK, TailLinesResponse{Lines: lines})
}
```

---

### `internal/daemon/engine.go` — new GetSessionTailLines method

**Analog:** `internal/daemon/engine.go` GetSessionStatus (lines 508-518) + engine_test.go scrollback strip pattern (lines 463-471)

**GetSessionStatus pattern** (lines 508-518):
```go
func (e *SessionEngine) GetSessionStatus(sessionID string) string {
	e.statusMu.RLock()
	s, ok := e.sessionStatuses[sessionID]
	e.statusMu.RUnlock()
	if !ok {
		return string(status.StatusRunning)
	}
	return string(s)
}
```

**Scrollback framing-byte strip pattern from engine_test.go** (lines 463-471):
```go
// Strip all 0x01 bytes to recover the payload text.
var collected strings.Builder
for _, b := range hub.ScrollbackSnapshot() {
	if b != relay.MsgOutput {
		collected.WriteByte(b)
	}
}
```

**Manager accessor** (engine.go line 1034):
```go
func (e *SessionEngine) Manager() *relay.HubManager {
```

**New method to add** (uses `e.manager.Get(id)` not `e.Manager()` — `manager` is the unexported field; see engine.go line 423):
```go
// ansiEscape matches CSI sequences and OSC sequences (title-setting, hyperlinks).
// Covers the full ANSI vocabulary emitted by Claude Code, opencode, Gemini CLI.
var ansiEscape = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[a-zA-Z]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)

// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer. ANSI escape sequences and relay framing bytes (0x01)
// are stripped. Returns nil if the session has no hub.
func (e *SessionEngine) GetSessionTailLines(id string, n int) []string {
	hub, ok := e.manager.Get(id)
	if !ok {
		return nil
	}
	raw := hub.ScrollbackSnapshot()
	// Strip relay.MsgOutput (0x01) framing bytes — pattern from engine_test.go lines 463-471.
	stripped := make([]byte, 0, len(raw))
	for _, b := range raw {
		if b != relay.MsgOutput {
			stripped = append(stripped, b)
		}
	}
	// Strip ANSI escape sequences.
	text := ansiEscape.ReplaceAllString(string(stripped), "")
	lines := strings.Split(text, "\n")
	// Remove empty trailing lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
```

---

### `internal/daemon/client.go` — new GetSessionTailLines client method

**Analog:** `internal/daemon/client.go` GetSessionStatus (lines 92-99)

**GetSessionStatus pattern** (lines 92-99):
```go
// GetSessionStatus returns the current status string for the given session.
func (c *DaemonClient) GetSessionStatus(id string) (string, error) {
	var resp StatusResponse
	if err := c.doJSON(http.MethodGet, "/sessions/"+id+"/status", nil, &resp); err != nil {
		return "", err
	}
	return resp.Status, nil
}
```

**New method to add:**
```go
// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences and framing bytes stripped.
func (c *DaemonClient) GetSessionTailLines(id string, n int) ([]string, error) {
	var resp TailLinesResponse
	if err := c.doJSON(http.MethodGet, fmt.Sprintf("/sessions/%s/tail?n=%d", id, n), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Lines, nil
}
```

---

### `app.go` — new GetSessionTailLines Wails binding

**Analog:** `app.go` GetSessionStatus (lines 418-427)

**GetSessionStatus pattern** (lines 418-427):
```go
func (a *App) GetSessionStatus(sessionID string) string {
	if a.client == nil {
		return string(status.StatusRunning)
	}
	s, err := a.client.GetSessionStatus(sessionID)
	if err != nil {
		return string(status.StatusRunning) // conservative default
	}
	return s
}
```

**New binding to add:**
```go
// GetSessionTailLines returns the last n plain-text lines from the session's
// scrollback buffer, with ANSI escape sequences stripped.
// Returns an empty slice if the session has no scrollback (e.g. remote sessions).
// Used by the Hub mini-preview poller (CARD-07). n is clamped to [1..20].
func (a *App) GetSessionTailLines(id string, n int) []string {
	if a.client == nil {
		return []string{}
	}
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	lines, err := a.client.GetSessionTailLines(id, n)
	if err != nil || lines == nil {
		return []string{}
	}
	return lines
}
```

---

### `frontend/src/wailsjs/go/main/App.d.ts` — new GetSessionTailLines declaration

**Analog:** Existing GetSessionStatus declaration pattern in App.d.ts (line 32 area):
```typescript
export function ListSessions(): Promise<SessionInfo[]>
// Phase 101-01 — shell discovery via daemon /shells route.
export function ListShells(): Promise<daemon.DetectedShell[]>
```

**Line to append** (after GetSessionStatus or near related session RPCs):
```typescript
// Phase 132 / CARD-07 — throttled tail snapshot for Hub mini-preview. n clamped to [1..20].
export function GetSessionTailLines(id: string, n: number): Promise<string[]>
```

---

### `frontend/src/lib/hubGroups.ts` — new file (group CRUD + localStorage)

**Analog:** `frontend/src/components/Sidebar.tsx` localStorage pattern (lines 12-43) + `frontend/src/components/NewSessionModal.tsx` localStorage initializer (lines 48-52)

**localStorage pattern from Sidebar.tsx** (lines 12-43):
```typescript
const STORAGE_KEY = 'sidebar-collapsed'

const [collapsed, setCollapsed] = useState<boolean>(
  () => localStorage.getItem(STORAGE_KEY) === 'true'
)

const toggle = () => {
  setCollapsed((prev) => {
    const next = !prev
    localStorage.setItem(STORAGE_KEY, String(next))
    return next
  })
}
```

**localStorage initializer from NewSessionModal.tsx** (lines 48-52):
```typescript
const [selectedDir, setSelectedDir] = useState(() => localStorage.getItem(LAST_DIR_KEY) ?? '')
const [argsText, setArgsText] = useState(() =>
  localStorage.getItem(ARGS_KEY(clis[0]?.Name ?? '')) ?? ''
)
```

**Full file to create** at `frontend/src/lib/hubGroups.ts`:
```typescript
/* HUB-GROUPS-V1: localStorage key "agenthub:hubGroups:v1" — JSON array of HubGroupDef */

export interface HubGroupDef {
  id: string           // random uuid — stable across restarts
  name: string         // user-chosen display name
  memberKeys: string[] // "${session.name}:::${session.workDir}" strings
}

const STORAGE_KEY = 'agenthub:hubGroups:v1'

/* GROUP-04: membership key = "${session.name}:::${session.workDir}" — survives session-id churn */
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

export function deleteGroup(groups: HubGroupDef[], groupId: string): HubGroupDef[] {
  const updated = groups.filter((g) => g.id !== groupId)
  saveGroups(updated)
  return updated
}
```

---

### `frontend/src/lib/remoteAdapter.ts` — new file (adaptRemoteSession)

**Analog:** `frontend/src/components/RemoteSessionsPanel.tsx` (RemoteSession + RemotePeerSessions types, lines 1-16) + `frontend/src/wailsjs/go/main/App.d.ts` SessionInfo shape (lines 6-24)

**RemoteSession/RemotePeerSessions types** (RemoteSessionsPanel.tsx lines 1-16):
```typescript
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
}

export interface RemotePeerSessions {
  hostname: string
  reachable: boolean
  sessions: RemoteSession[]
}
```

**SessionInfo shape** (App.d.ts lines 6-24):
```typescript
export interface SessionInfo {
  id: string; cli: string; name: string; state: string; status: string;
  createdAt: string; hostname: string; webEnabled: boolean; viewerCount: number;
  exitCode?: number; duration?: number; homeDir: boolean; filesWrite: boolean; workDir: string
}
```

**Full file to create** at `frontend/src/lib/remoteAdapter.ts`:
```typescript
/* GRID-07: remote sessions adapted via adaptRemoteSession(); hostname != '' routes to GlobeAltIcon + hostname */
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
    state: 'running',          // conservative default — remote status is not granular
    status: session.status || 'running',
    createdAt: new Date().toISOString(),
    hostname: peer.hostname,   // non-empty → GlobeAltIcon + hostname in SessionCard
    webEnabled: true,
    viewerCount: 0,
    workDir: '',               // remote sessions have no local workDir → fall into "Other"
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

---

### `frontend/src/components/Hub/MiniPreview.tsx` — new component

**Analog:** `frontend/src/components/Hub/SessionCard.tsx` ROW rendering pattern (lines 1-16, 139-227) — same import style, same BEM class pattern, same aria conventions

**Import pattern from SessionCard.tsx** (lines 1-16):
```typescript
import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
// ... heroicons as needed
```

**ROW rendering pattern from SessionCard.tsx** (lines 139-227):
Each ROW is a `<div className="hub-card__rowN">` containing spans. States guarded by conditionals.

**New component to create** at `frontend/src/components/Hub/MiniPreview.tsx`:
```typescript
/* CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */
import React from 'react'

export interface MiniPreviewProps {
  /** Lines from usePreviewPoller. undefined = not yet fetched (loading). Empty array = no output. */
  lines: string[] | undefined
  /** When true (stopped-ok), preview gets dim opacity via parent card's CSS. */
  dimmed?: boolean
}

/**
 * MiniPreview — ROW 6 of SessionCard. Plain text snapshot of last 4 lines.
 * NEVER mounts an xterm instance. aria-hidden — decorative only.
 */
export function MiniPreview({ lines }: MiniPreviewProps): React.ReactElement {
  if (lines === undefined) {
    return (
      <div className="hub-card__preview hub-card__preview--loading" aria-hidden="true">
        <span className="hub-card__preview-line">Loading…</span>
      </div>
    )
  }
  if (lines.length === 0) {
    return (
      <div className="hub-card__preview hub-card__preview--empty" aria-hidden="true">
        <span className="hub-card__preview-line">No output yet</span>
      </div>
    )
  }
  return (
    <div className="hub-card__preview" aria-hidden="true">
      {lines.map((line, i) => (
        // key by index — order is stable within a snapshot
        <div key={i} className="hub-card__preview-line">{line || ' '}</div>
      ))}
    </div>
  )
}
```

---

### `frontend/src/components/Hub/GroupSidebar.tsx` — new component

**Analog:** `frontend/src/components/Sidebar.tsx` collapsed state + toggle pattern (lines 24-79) + `frontend/src/components/Hub/SessionCard.tsx` STATUS_CONFIG BEM pattern

**Collapsed toggle pattern from Sidebar.tsx** (lines 24-79):
```typescript
const [collapsed, setCollapsed] = useState<boolean>(
  () => localStorage.getItem(STORAGE_KEY) === 'true'
)

const toggle = () => {
  setCollapsed((prev) => {
    const next = !prev
    localStorage.setItem(STORAGE_KEY, String(next))
    return next
  })
}

return (
  <nav className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`} aria-label="...">
    <button className="sidebar__toggle" onClick={toggle} aria-label="Toggle sidebar">
      <Bars3Icon className="sidebar__icon" />
    </button>
    {!collapsed && <span className="sidebar__label">Groups</span>}
    ...
  </nav>
)
```

**Drag-drop target pattern from FileBrowserTab.tsx** (lines 879-899):
```typescript
const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
  e.preventDefault()
  e.stopPropagation()
  setIsDragOver(true)
}, [canWrite])

const handleDragLeave = useCallback(() => {
  setIsDragOver(false)
}, [])
```

**Component structure to create** at `frontend/src/components/Hub/GroupSidebar.tsx`:
- Named exports: `GroupSidebar` (outer) + `GroupSidebarItem` (sub-component, same file)
- Props: `groupDefs: HubGroupDef[]`, `sessions: SessionInfo[]`, `activeGroupId: string | null`, `collapsed: boolean`, `onToggle: () => void`, `onGroupSelect: (id: string | null) => void`, `onCreateGroup: (name: string) => void`, `onDropOnGroup: (groupId: string, memberKey: string) => void`
- BEM classes: `.hub__group-sidebar`, `.hub__group-sidebar--collapsed`, `.hub__group-sidebar-toggle`, `.hub__group-sidebar-heading`, `.hub__group-sidebar-list`, `.hub__group-sidebar-item`, `.hub__group-sidebar-item--active`, `.hub__group-sidebar-item__name`, `.hub__group-sidebar-item__count`, `.hub__group-sidebar-item__needs-input-badge`, `.hub__group-sidebar-new`
- Needs-input badge MUST include `PauseCircleIcon` alongside amber color per Pitfall 8 / colorblind mandate

---

### `frontend/src/components/Hub/SessionCard.tsx` — modified

**Analog:** Self (existing file, lines 1-228)

**What to add to the existing card:**

1. **Drag handle** — position absolute top-left, visible on hover:
```typescript
// Add to imports:
import { Bars3Icon, EllipsisHorizontalIcon } from '@heroicons/react/24/outline'

// Add draggable attribute and drag handlers to <article>:
<article
  className={`hub-card${hubStatus === 'stopped-ok' ? ' hub-card--dim' : ''}`}
  draggable="true"
  onDragStart={(e) => {
    e.dataTransfer.setData('text/plain', memberKeyForSession)
    e.dataTransfer.effectAllowed = 'move'
    setIsDragging(true)
  }}
  onDragEnd={() => setIsDragging(false)}
  aria-label={cardAriaLabel}
  tabIndex={0}
>
  {/* Drag handle — visible on hover (CSS opacity transition) */}
  <span
    className="hub-card__drag-handle"
    aria-label="Drag to reorder"
    aria-hidden="true"
  >
    <Bars3Icon className="w-4 h-4" />
  </span>

  {/* Overflow menu button — visible on hover */}
  <button
    type="button"
    className="hub-card__menu-btn"
    aria-label={`Card options for ${name}`}
    aria-expanded={menuOpen}
    aria-haspopup="menu"
    onClick={() => setMenuOpen((p) => !p)}
  >
    <EllipsisHorizontalIcon className="w-4 h-4" />
  </button>
```

2. **ROW 6: MiniPreview** — add after ROW 5:
```typescript
{/* ROW 6: MiniPreview — CARD-07: plain text snapshot; NO xterm */}
<MiniPreview lines={previewLines} />
```

3. **Props additions:**
```typescript
export interface SessionCardProps {
  session: SessionInfo
  onRename?: (id: string, name: string) => void
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** CARD-07: tail lines from usePreviewPoller; undefined = loading */
  previewLines?: string[]
  /** GROUP-02: group definitions for the "Move to group" overflow menu */
  groupDefs?: HubGroupDef[]
  /** GROUP-02: fires when user assigns this card to a group via menu */
  onAssignGroup?: (memberKey: string, groupId: string) => void
}
```

**memberKey derivation** (add near top of component):
```typescript
import { memberKey } from '../../lib/hubGroups'
// ...
const memberKeyForSession = memberKey(name, session.workDir)
```

---

### `frontend/src/components/Hub/SessionCardGrid.tsx` — modified

**Analog:** Self (existing file, lines 1-103), particularly `groupByWorkDir` (lines 13-22) and the map/render loop (lines 79-101)

**Grouping logic to replace/extend** (lines 13-22 + 75-101):
```typescript
// Existing groupByWorkDir (lines 13-22):
export function groupByWorkDir(sessions: SessionInfo[]): Map<string, SessionInfo[]> {
  const groups = new Map<string, SessionInfo[]>()
  for (const s of sessions) {
    const key = s.workDir || ''
    const group = groups.get(key) ?? []
    group.push(s)
    groups.set(key, group)
  }
  return groups
}

// Existing render loop (lines 79-101):
const groups = groupByWorkDir(sessions)
return (
  <>
    {Array.from(groups.entries()).map(([workDir, groupSessions]) => {
      const headerLabel = workDir ? basename(workDir) : 'Other'
      return (
        <div key={workDir || '__other__'} className="hub__group">
          <h2 className="hub__group-header" ...>
          <div role="list" className="hub__card-row">
            {groupSessions.map((s) => (
              <div role="listitem" key={s.id}>
                <SessionCard session={s} onRename={onRename} onOpenSession={onOpenSession} />
              </div>
            ))}
          </div>
        </div>
      )
    })}
  </>
)
```

**New props to add:**
```typescript
export interface SessionCardGridProps {
  sessions: SessionInfo[]
  onRename: (id: string, name: string) => void
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** Phase 132 — named group definitions; when non-empty overrides workDir grouping */
  groupDefs?: HubGroupDef[]
  /** Phase 132 — tail lines map from usePreviewPoller, keyed by session ID */
  previewTails?: Map<string, string[]>
  /** Phase 132 — fires when user assigns via card overflow menu or DnD */
  onAssignGroup?: (memberKey: string, groupId: string) => void
}
```

**New groupByNamedGroups helper** (add alongside groupByWorkDir):
```typescript
import { memberKey, type HubGroupDef } from '../../lib/hubGroups'

export function groupByNamedGroups(
  sessions: SessionInfo[],
  groupDefs: HubGroupDef[],
): Map<string, { label: string; sessions: SessionInfo[] }> {
  // Build result map: named groups in definition order, then "Other"
  const result = new Map<string, { label: string; sessions: SessionInfo[] }>()
  for (const g of groupDefs) {
    result.set(g.id, { label: g.name, sessions: [] })
  }
  result.set('__other__', { label: 'Other', sessions: [] })

  for (const s of sessions) {
    const key = memberKey(s.name, s.workDir)
    const matchingGroup = groupDefs.find((g) => g.memberKeys.includes(key))
    if (matchingGroup) {
      result.get(matchingGroup.id)!.sessions.push(s)
    } else {
      result.get('__other__')!.sessions.push(s)
    }
  }
  return result
}
```

---

### `frontend/src/components/Hub/HubPanel.tsx` — modified

**Analog:** Self (existing file, lines 1-158), particularly the useEffect polling pattern (lines 91-108) and the filterSessions composition (line 109)

**useEffect poll pattern to mirror** (lines 91-108):
```typescript
// '/' shortcut — focus search input when no input is focused (GRID-05)
useEffect(() => {
  function onKeyDown(e: KeyboardEvent) { ... }
  window.addEventListener('keydown', onKeyDown)
  return () => window.removeEventListener('keydown', onKeyDown)
}, [])
```

**usePreviewPoller hook** (new hook, defined in HubPanel.tsx or extracted to lib/):
```typescript
import { GetSessionTailLines } from '../../wailsjs/go/main/App'

function usePreviewPoller(
  sessions: SessionInfo[],
  isActive: boolean,
): Map<string, string[]> {
  const [tails, setTails] = useState<Map<string, string[]>>(new Map())

  // Stable dep: join session IDs to avoid rebinding on every array reference change (Pitfall 3)
  const sessionIdKey = sessions.map((s) => s.id).join(',')

  useEffect(() => {
    if (!isActive || sessions.length === 0) return
    let cancelled = false

    async function poll() {
      // Only fetch for local sessions — remote sessions have no tail API (Pitfall 4 avoidance)
      const localSessions = sessions.filter((s) => !s.hostname || s.hostname === '')
      if (localSessions.length === 0) return
      const results = await Promise.all(
        localSessions.map((s) =>
          GetSessionTailLines(s.id, 4).catch(() => [] as string[])
        )
      )
      if (!cancelled) {
        setTails(new Map(localSessions.map((s, i) => [s.id, results[i]])))
      }
    }

    void poll()
    const interval = setInterval(() => void poll(), 3000)
    return () => { cancelled = true; clearInterval(interval) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionIdKey, isActive])

  return tails
}
```

**Props additions to HubPanel:**
```typescript
export interface HubPanelProps {
  sessions: SessionInfo[]
  error: boolean
  onNewSession: () => void
  onRename: (id: string, name: string) => void
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** Phase 132 / GRID-07 — remote sessions from App.tsx (already adapted to SessionInfo[]) */
  remoteSessions?: SessionInfo[]
  /** Phase 132 / CARD-07 — true when Hub tab is active (gates usePreviewPoller interval) */
  isActive?: boolean
}
```

**Hub body layout change** — wrap existing `hub__grid-scroll` in new `hub__body` flex container:
```typescript
{/* Phase 132: hub__body is a flex row wrapping GroupSidebar + hub__grid-scroll */}
<div className="hub__body">
  <GroupSidebar
    groupDefs={groupDefs}
    sessions={allSessions}  // merged local + remote
    activeGroupId={activeGroupId}
    collapsed={sidebarCollapsed}
    onToggle={() => setSidebarCollapsed((p) => !p)}
    onGroupSelect={setActiveGroupId}
    onCreateGroup={(name) => setGroupDefs((prev) => createGroup(prev, name))}
    onDropOnGroup={(groupId, key) => setGroupDefs((prev) => assignToGroup(prev, groupId, key))}
  />
  <div className="hub__grid-scroll">
    {body}
  </div>
</div>
```

---

### `frontend/src/App.tsx` — modified (remote poll extension)

**Analog:** Self (existing), remote poll useEffect (lines 933-960), Hub poll useEffect (lines 907-923)

**Remote poll gating pattern** (lines 933-936):
```typescript
useEffect(() => {
  if (mode === 'web') return
  if (activeId !== REMOTE_SESSIONS_TAB.id) return  // ← EXTEND THIS
  ...
}, [activeId])
```

**Hub rendering** (lines 1355-1363):
```typescript
{activeId === HUB_TAB.id && (
  <HubPanel
    sessions={hubSessions}
    error={hubError}
    onNewSession={() => setShowNewSessionModal(true)}
    onRename={handleRenameTab}
    onOpenSession={handleOpenSessionTab}
  />
)}
```

**Changes required:**
1. Extend remote poll gate: `if (activeId !== REMOTE_SESSIONS_TAB.id && activeId !== HUB_TAB.id) return`
2. Add `remoteSessions` and `isActive` props to `<HubPanel>`:
```typescript
{activeId === HUB_TAB.id && (
  <HubPanel
    sessions={hubSessions}
    error={hubError}
    onNewSession={() => setShowNewSessionModal(true)}
    onRename={handleRenameTab}
    onOpenSession={handleOpenSessionTab}
    remoteSessions={adaptAllRemoteSessions(remotePeers)}
    isActive={activeId === HUB_TAB.id}
  />
)}
```

---

### `frontend/src/style.css` — append new CSS tokens + Hub body/preview/sidebar rules

**Analog:** Existing `:root` and `[data-ui-theme="light"]` blocks + `.hub-*` BEM class definitions (Phase 131)

**Token append pattern** — append to existing `:root` block (do NOT create a new `:root`):
```css
/* Phase 132 / CARD-07: mini preview is plain text snapshot — NO xterm instance; polling interval 3s shared across all cards */
/* Phase 132 new --hub-* tokens */
:root {
  /* ... existing Phase 131 tokens ... */
  --hub-preview-bg: #0d0e17;
  --hub-preview-text: #8b92b3;
  --hub-preview-border: #1e2130;
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
  /* ... existing Phase 131 tokens ... */
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

**New BEM rules to add** (after existing `.hub-*` rules):
```css
/* Phase 132 / GRID-03: hub__body two-column flex layout */
.hub__body {
  display: flex;
  flex-direction: row;
  flex: 1;
  min-height: 0;
}

/* Phase 132 / GRID-03: group sidebar */
.hub__group-sidebar {
  width: var(--hub-sidebar-width);
  background: var(--hub-sidebar-bg);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}

.hub__group-sidebar--collapsed {
  width: var(--hub-sidebar-collapsed-width);
}

/* Motion contract: width transition only when user has no reduced-motion preference */
@media (prefers-reduced-motion: no-preference) {
  .hub__group-sidebar {
    transition: width 150ms ease;
  }
}

/* Drag handle and menu button — visible on card hover/focus-within */
.hub-card__drag-handle,
.hub-card__menu-btn {
  position: absolute;
  opacity: 0;
}

@media (prefers-reduced-motion: no-preference) {
  .hub-card__drag-handle,
  .hub-card__menu-btn {
    transition: opacity 100ms ease;
  }
}

.hub-card:hover .hub-card__drag-handle,
.hub-card:focus-within .hub-card__drag-handle,
.hub-card:hover .hub-card__menu-btn,
.hub-card:focus-within .hub-card__menu-btn {
  opacity: 1;
}

.hub-card__drag-handle {
  top: 8px;
  left: 8px;
  cursor: grab;
  color: var(--hub-text-muted);
}

.hub-card__menu-btn {
  top: 8px;
  right: 8px;
  cursor: pointer;
  color: var(--hub-text-muted);
  background: transparent;
  border: none;
  padding: 0;
}

/* CARD-07: mini preview pane */
.hub-card__preview {
  height: 56px;
  overflow: hidden;
  border-top: 1px solid var(--hub-preview-border);
  background: var(--hub-preview-bg);
  padding: 4px 8px;
}

.hub-card__preview-line {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.3;
  color: var(--hub-preview-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hub-card__preview--empty .hub-card__preview-line,
.hub-card__preview--loading .hub-card__preview-line {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--hub-text-muted);
  font-family: system-ui;
}
```

---

## Shared Patterns

### Go response type pattern
**Source:** `internal/daemon/types.go` lines 57-60
**Apply to:** `TailLinesResponse` new struct
```go
type TailLinesResponse struct {
	Lines []string `json:"lines"`
}
```

### Go handler pattern
**Source:** `internal/daemon/api.go` lines 604-608
**Apply to:** `handleGetSessionTailLines`
```go
func (a *API) handleGetSessionStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := a.engine.GetSessionStatus(id)
	writeJSON(w, http.StatusOK, StatusResponse{Status: s})
}
```

### App.go nil-guard delegation
**Source:** `app.go` lines 418-427
**Apply to:** `GetSessionTailLines` Wails binding
```go
func (a *App) GetSessionStatus(sessionID string) string {
	if a.client == nil {
		return string(status.StatusRunning)
	}
	s, err := a.client.GetSessionStatus(sessionID)
	if err != nil {
		return string(status.StatusRunning)
	}
	return s
}
```

### localStorage CRUD pattern
**Source:** `frontend/src/components/Sidebar.tsx` lines 12-43
**Apply to:** `hubGroups.ts` loadGroups/saveGroups
```typescript
const STORAGE_KEY = 'sidebar-collapsed'
const [collapsed, setCollapsed] = useState<boolean>(
  () => localStorage.getItem(STORAGE_KEY) === 'true'
)
localStorage.setItem(STORAGE_KEY, String(next))
```

### HTML5 DnD handlers
**Source:** `frontend/src/components/FileBrowserTab.tsx` lines 879-899
**Apply to:** `GroupSidebarItem` drop target, `SessionCard` drag source
```typescript
const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
  e.preventDefault()   // mandatory to allow drop
  e.stopPropagation()
  setIsDragOver(true)
}, [canWrite])

const handleDragLeave = useCallback(() => { setIsDragOver(false) }, [])
```

### Hub polling useEffect
**Source:** `frontend/src/App.tsx` lines 907-923
**Apply to:** `usePreviewPoller` hook interval pattern, remote poll extension
```typescript
useEffect(() => {
  if (mode === 'web') return
  if (activeId !== HUB_TAB.id) return
  let cancelled = false
  async function refresh() {
    try {
      const sessions = await ListSessions()
      if (!cancelled) { setHubSessions(sessions); setHubError(false) }
    } catch { if (!cancelled) setHubError(true) }
  }
  void refresh()
  const interval = setInterval(() => void refresh(), 3000)
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])
```

### ANSI strip regex
**Source:** `frontend/src/lib/stripAnsi.ts` line 21
**Apply to:** Go `ansiEscape` regexp in engine.go (extended to also cover OSC sequences)
```typescript
const ANSI_ESCAPE_RE = /\x1b\[\??[0-9;]*[a-zA-Z]/g
```
Go equivalent (extended):
```go
var ansiEscape = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[a-zA-Z]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)
```

### BEM modifier + conditional class pattern
**Source:** `frontend/src/components/Hub/SessionCard.tsx` line 141
**Apply to:** GroupSidebar collapsed modifier, MiniPreview state modifiers
```typescript
<article className={`hub-card${hubStatus === 'stopped-ok' ? ' hub-card--dim' : ''}`}>
```

---

## No Analog Found

All files have close analogs. No files require falling back to RESEARCH.md patterns alone.

---

## Critical Implementation Notes

### Framing byte strip (Go layer)
The scrollback stores relay frames with a leading `0x01` (`relay.MsgOutput`) byte. This MUST be stripped before treating the buffer as text. Pattern is from `engine_test.go` lines 463-471:
```go
for _, b := range hub.ScrollbackSnapshot() {
    if b != relay.MsgOutput {
        collected.WriteByte(b)
    }
}
```

### usePreviewPoller stable dep (Pitfall 3)
Do NOT depend on the `sessions` array directly — it is a new reference every poll tick, causing a polling storm. Derive a stable string dep:
```typescript
const sessionIdKey = sessions.map((s) => s.id).join(',')
// use sessionIdKey in useEffect deps instead of sessions
```

### Remote poll gate extension (Pitfall 4)
App.tsx line 935 currently reads `if (activeId !== REMOTE_SESSIONS_TAB.id) return`. Must become:
```typescript
if (activeId !== REMOTE_SESSIONS_TAB.id && activeId !== HUB_TAB.id) return
```

### prefers-reduced-motion (Pitfall 7)
ALL new CSS transitions must be inside `@media (prefers-reduced-motion: no-preference)`. Specifically:
- `transition: width 150ms ease` on `.hub__group-sidebar`
- `transition: opacity 100ms ease` on `.hub-card__drag-handle` and `.hub-card__menu-btn`

### Needs-input badge icon (Pitfall 8 / colorblind mandate)
```typescript
import { PauseCircleIcon } from '@heroicons/react/24/outline'
// Badge must render:
<span className="hub__group-sidebar-item__needs-input-badge" aria-label={`${count} session${count === 1 ? '' : 's'} need${count === 1 ? 's' : ''} input`}>
  {/* COLORBLIND-SAFE: needs-input badge dark hex #f59e0b — reinforcement only; PauseCircleIcon carries the state */}
  <PauseCircleIcon className="w-3 h-3" aria-hidden="true" />
  <span>{count}</span>
</span>
```

---

## Metadata

**Analog search scope:** `internal/daemon/`, `app.go`, `frontend/src/components/Hub/`, `frontend/src/components/`, `frontend/src/lib/`, `frontend/src/App.tsx`, `frontend/src/wailsjs/`
**Files scanned:** 18
**Pattern extraction date:** 2026-06-16
