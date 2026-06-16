# Phase 131: Hub Foundation + Static Session Cards - Pattern Map

**Mapped:** 2026-06-16
**Files analyzed:** 14 (6 new Hub components, 4 modified existing files, 4 new test files)
**Analogs found:** 14 / 14

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/daemon/types.go` | model | — | self (add 1 field) | exact |
| `internal/daemon/engine.go` | service | CRUD | self (add WorkDir to ListSessions) | exact |
| `app.go` | controller | request-response | self (propagate 4 fields) | exact |
| `frontend/src/wailsjs/go/main/App.d.ts` | config | — | self (add workDir) | exact |
| `frontend/src/components/TabBar.tsx` | component | — | self (add `\| 'hub'` to type union) | exact |
| `frontend/src/components/Sidebar.tsx` | component | request-response | self | exact |
| `frontend/src/App.tsx` | controller | request-response | self (HUB_TAB + poll + render) | exact |
| `frontend/src/components/Hub/HubPanel.tsx` | component | request-response | `DaemonManagerPanel.tsx` | role-match |
| `frontend/src/components/Hub/SessionCard.tsx` | component | request-response | `DaemonManagerPanel.tsx` (session row) | role-match |
| `frontend/src/components/Hub/SessionCardGrid.tsx` | component | transform | `RemoteSessionsPanel.tsx` (peer groups) | role-match |
| `frontend/src/components/Hub/HubFilterBar.tsx` | component | event-driven | `TabBar.tsx` (filter-like state) | partial |
| `frontend/src/components/Hub/HubEmptyState.tsx` | component | — | `WelcomeTab.tsx` (empty state) | partial |
| `frontend/src/components/Hub/InlineSessionName.tsx` | component | request-response | `TabBar.tsx` (rename input) | exact |
| `frontend/src/style.css` | config | — | self (append Hub CSS custom properties) | exact |

---

## Pattern Assignments

### `internal/daemon/types.go` (model — add `WorkDir string`)

**Analog:** `internal/daemon/types.go` lines 19–34 (existing `SessionInfo` struct)

**Existing struct pattern** (lines 19–34):
```go
// SessionInfo is the JSON-serialisable representation of a session.
type SessionInfo struct {
    ID          string `json:"id"`
    CLI         string `json:"cli"`
    Name        string `json:"name"`
    State       string `json:"state"`
    Status      string `json:"status"` // heuristic status: running/idle/waiting/errored
    CreatedAt   string `json:"createdAt"`
    Hostname    string `json:"hostname"`
    WebEnabled  bool   `json:"webEnabled"`
    ViewerCount int    `json:"viewerCount"`        // MC-04: number of active WebSocket subscribers
    ExitCode    *int   `json:"exitCode,omitempty"` // nil while running; set when State is "stopped"
    Duration    *int   `json:"duration,omitempty"` // seconds since CreatedAt; set when State is "stopped"
    HomeDir     bool   `json:"homeDir"`
    FilesWrite  bool   `json:"filesWrite"`
}
```

**Add after `FilesWrite`:**
```go
    WorkDir     string `json:"workDir"` // Phase 131 / GRID-02: absolute working directory from sessionWorkDirs map
```

---

### `internal/daemon/engine.go` (service — populate WorkDir in ListSessions)

**Analog:** `internal/daemon/engine.go` lines 450–468 (existing `result = append(result, SessionInfo{...})` block)

**Existing append pattern** (lines 450–468):
```go
result = append(result, SessionInfo{
    ID:          s.ID,
    CLI:         s.CLI,
    Name:        name,
    State:       state,
    Status:      heuristicStatus,
    CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
    Hostname:    e.hostname,
    ViewerCount: viewerCount,
    ExitCode:    exitCodePtr,
    Duration:    durationPtr,
    HomeDir:    e.sessionCwdIsHomeUnlocked(s.ID),
    FilesWrite: e.filesWriteEnabledForUnlocked(s.ID),
})
```

**Add `WorkDir` field — safe because `e.mu.RLock` is already held via `defer` at line 410:**
```go
    WorkDir:    e.sessionWorkDirs[s.ID], // Phase 131 / GRID-02: populated from existing map inside held RLock
```

**Key constraint:** `sessionWorkDirs` is accessed under `e.mu.RLock` (line 409 `defer e.mu.RUnlock()`). Map read inside the already-held lock is safe — no new lock acquisition required.

---

### `app.go` (controller — add 4 fields to `SessionInfo` struct and `ListSessions()`)

**Analog:** `app.go` lines 29–46 (existing `SessionInfo` struct) and lines 354–369 (existing `ListSessions()` mapping)

**Existing struct** (lines 29–46) — add 4 fields after `FilesWrite`:
```go
type SessionInfo struct {
    ID         string `json:"id"`
    CLI        string `json:"cli"`
    Name       string `json:"name"`
    State      string `json:"state"`
    Status     string `json:"status"`
    CreatedAt  string `json:"createdAt"`
    Hostname   string `json:"hostname"`
    WebEnabled bool   `json:"webEnabled"`
    HomeDir    bool   `json:"homeDir"`
    FilesWrite bool   `json:"filesWrite"`
    // Phase 131 — fields required by Hub CARD-04..CARD-06, GRID-02
    ViewerCount int    `json:"viewerCount"`
    ExitCode    *int   `json:"exitCode,omitempty"`
    Duration    *int   `json:"duration,omitempty"`
    WorkDir     string `json:"workDir"`
}
```

**Existing mapping in `ListSessions()`** (lines 354–370) — add 4 propagations:
```go
result[i] = SessionInfo{
    ID:         s.ID,
    CLI:        s.CLI,
    Name:       s.Name,
    State:      s.State,
    Status:     s.Status,
    CreatedAt:  s.CreatedAt,
    Hostname:   s.Hostname,
    WebEnabled: s.WebEnabled,
    HomeDir:    s.HomeDir,
    FilesWrite: s.FilesWrite,
    // Phase 131 — propagate from daemon.SessionInfo (already populated there)
    ViewerCount: s.ViewerCount,
    ExitCode:    s.ExitCode,
    Duration:    s.Duration,
    WorkDir:     s.WorkDir,
}
```

---

### `frontend/src/wailsjs/go/main/App.d.ts` (config — add `workDir` field)

**Analog:** `App.d.ts` lines 6–22 (existing `SessionInfo` interface)

`viewerCount`, `exitCode`, and `duration` are already declared (lines 15–17). Add only `workDir`:
```typescript
export interface SessionInfo {
  // ... existing fields ...
  workDir: string  // Phase 131 / GRID-02: populated from engine.sessionWorkDirs
}
```

---

### `frontend/src/components/TabBar.tsx` (component — extend Tab type union)

**Analog:** `TabBar.tsx` line 8 (existing type union)

**Current** (line 8):
```typescript
type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings' | 'file-browser'
```

**Change to:**
```typescript
type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings' | 'file-browser' | 'hub'
```

---

### `frontend/src/components/Sidebar.tsx` (component — add `onOpenHub` prop + `activePanel`)

**Analog:** `Sidebar.tsx` lines 13–19 (existing `SidebarProps`) and lines 52–70 (existing button pattern)

**Existing props pattern** (lines 13–19):
```typescript
interface SidebarProps {
  onHome: () => void
  onOpenRemoteSessions: () => void
  onOpenDaemonManager: () => void
  onAdd: () => void
  onSettings: () => void
}
```

**Add `onOpenHub` and `activePanel`:**
```typescript
interface SidebarProps {
  onHome: () => void
  onOpenRemoteSessions: () => void
  onOpenDaemonManager: () => void
  onAdd: () => void
  onSettings: () => void
  onOpenHub: () => void           // Phase 131 / HUB-01
  activePanel?: string            // Phase 131 / Pitfall-8: active state indicator
}
```

**Existing button pattern to copy** (lines 71–79 — Sessions button):
```typescript
<button
  className="sidebar__item"
  onClick={onOpenDaemonManager}
  aria-label="Sessions"
>
  <ServerStackIcon className="sidebar__icon" />
  {!collapsed && <span className="sidebar__label">Sessions</span>}
</button>
```

**New Hub button (copy this pattern, place before "New Session"):**
```typescript
<button
  className={`sidebar__item${activePanel === '__hub__' ? ' sidebar__item--active' : ''}`}
  onClick={onOpenHub}
  aria-label="Hub"
>
  <Squares2X2Icon className="sidebar__icon" />
  {!collapsed && <span className="sidebar__label">Hub</span>}
</button>
```

**Import addition** (line 2 Heroicons block):
```typescript
import {
  Bars3Icon,
  ServerStackIcon,
  HomeIcon,
  GlobeAltIcon,
  PlusIcon,
  Cog6ToothIcon,
  Squares2X2Icon,   // Phase 131 / HUB-01
} from '@heroicons/react/24/outline'
```

---

### `frontend/src/App.tsx` (controller — HUB_TAB constant, handler, poll, render)

**Analog:** `App.tsx` lines 85–88 (existing Tab constants), lines 873–894 (daemon-manager poll), lines 1071–1079 (handleOpenRemoteSessions pattern), lines 1273–1283 (panel render)

**Tab constant** (after line 88, mirroring the existing 4 constants):
```typescript
const HUB_TAB: Tab = { id: '__hub__', name: 'Hub', sessionId: '', cli: '', type: 'hub' }
```

**Hub state** (alongside `panelSessions` at line 182):
```typescript
const [hubSessions, setHubSessions] = useState<SessionInfo[]>([])
const [hubError, setHubError] = useState(false)
```

**Handler** (mirrors `handleOpenRemoteSessions` at lines 1071–1079):
```typescript
const handleOpenHub = useCallback(() => {
  const existing = tabs.find((t) => t.type === 'hub')
  if (existing) {
    setActiveId(existing.id)
    return
  }
  setTabs((prev) => [...prev, HUB_TAB])
  setActiveId(HUB_TAB.id)
}, [tabs])
```

**Polling effect** (mirrors lines 873–894 — daemon-manager poll):
```typescript
useEffect(() => {
  if (mode === 'web') return
  if (activeId !== HUB_TAB.id) return
  let cancelled = false
  async function refresh() {
    try {
      const sessions = await ListSessions()
      if (!cancelled) { setHubSessions(sessions); setHubError(false) }
    } catch {
      if (!cancelled) setHubError(true)
    }
  }
  void refresh()
  const interval = setInterval(() => void refresh(), 3000)
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])
```

**Sidebar wiring** (lines 1242–1248 — add `onOpenHub` and `activePanel`):
```typescript
<Sidebar
  onHome={handleHome}
  onOpenRemoteSessions={handleOpenRemoteSessions}
  onOpenDaemonManager={handleOpenDaemonManager}
  onAdd={handleAddTab}
  onSettings={handleOpenSettings}
  onOpenHub={handleOpenHub}
  activePanel={activeId ?? undefined}
/>
```

**Panel render** (after line 1283 — DaemonManagerPanel block, mirroring that exact pattern):
```typescript
{activeId === HUB_TAB.id && (
  <HubPanel
    sessions={hubSessions}
    error={hubError}
    onNewSession={() => setShowNewSessionModal(true)}
    onRename={handleRenameTab}
  />
)}
```

**Terminal exclusion filter** — wherever `t.type !== 'daemon-manager'` et al. exclude non-terminals, add:
```typescript
t.type !== 'hub'
```

---

### `frontend/src/components/Hub/HubPanel.tsx` (component, request-response)

**Analog:** `frontend/src/components/DaemonManagerPanel.tsx` (top-level panel receiving `sessions` prop)

**Imports pattern** (mirrors DaemonManagerPanel.tsx lines 1–7):
```typescript
import React, { useState, useRef, useEffect, useCallback } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { HubFilterBar } from './HubFilterBar'
import { SessionCardGrid } from './SessionCardGrid'
import { HubEmptyState } from './HubEmptyState'
import { ExclamationCircleIcon } from '@heroicons/react/24/outline'
```

**Props interface pattern** (mirrors `DaemonManagerPanelProps`):
```typescript
export interface HubPanelProps {
  sessions: SessionInfo[]
  error: boolean
  onNewSession: () => void
  onRename: (id: string, name: string) => void
}
```

**Filter/search state + derived sessions** (owned by HubPanel, not lifted):
```typescript
export function HubPanel({ sessions, error, onNewSession, onRename }: HubPanelProps): React.ReactElement {
  const [activeFilter, setActiveFilter] = useState<HubFilter>('all')
  const [searchText, setSearchText] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  // "/" shortcut — focus search input (GRID-05)
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === '/' && document.activeElement?.tagName !== 'INPUT') {
        e.preventDefault()
        searchRef.current?.focus()
        searchRef.current?.select()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  const filtered = filterSessions(sessions, activeFilter, searchText)
  // ...
}
```

**Error state render** (mirrors DaemonManagerPanel error patterns):
```tsx
if (error) {
  return (
    <div className="hub__error-state">
      <ExclamationCircleIcon className="hub__error-icon" aria-hidden="true" />
      <h2 className="hub__error-heading">Couldn't load sessions</h2>
      <p className="hub__error-body">Check that the daemon is running and try again.</p>
    </div>
  )
}
```

---

### `frontend/src/components/Hub/SessionCard.tsx` (component, request-response)

**Analog:** `DaemonManagerPanel.tsx` session row rendering; `TabBar.tsx` agent badge pattern

**Imports pattern:**
```typescript
import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import {
  ArrowPathIcon, CheckCircleIcon, PauseCircleIcon,
  ExclamationCircleIcon, StopCircleIcon,
  ComputerDesktopIcon, GlobeAltIcon, EyeIcon,
} from '@heroicons/react/24/outline'
import { InlineSessionName } from './InlineSessionName'
```

**CLI badge pattern** (reuses existing `tab__agent-badge--{cli}` hex constants from style.css lines 1054–1060):
```typescript
// Reuse existing tab__agent-badge CSS classes — same hex constants
// tab__agent-badge--claude   #7aa2f7
// tab__agent-badge--opencode #9ece6a
// tab__agent-badge--codex    #bb9af7
// tab__agent-badge--gemini   #2ac3de
// tab__agent-badge--cursor   #e0af68
// tab__agent-badge--aider    #f7768e
// tab__agent-badge--shell    #89ddff
const badgeMod = agentBadgeModifier(cli)  // same function as TabBar.tsx line 18
```

**Status config pattern** (derived from UI-SPEC Status Indicator Contract):
```typescript
// COLORBLIND-SAFE: every status has unique icon shape + text label; color is reinforcement only
const STATUS_CONFIG = {
  running:      { Icon: ArrowPathIcon,        label: 'Running',     spin: true  },
  idle:         { Icon: CheckCircleIcon,      label: 'Idle',        spin: false },
  waiting:      { Icon: PauseCircleIcon,      label: 'Needs input', spin: false },
  errored:      { Icon: ExclamationCircleIcon,label: 'Error',       spin: false },
  'stopped-ok': { Icon: StopCircleIcon,       label: 'Done',        spin: false },
  'stopped-err':{ Icon: ExclamationCircleIcon,label: 'Exited',      spin: false },
} as const
```

**Dimmed card pattern** (CARD-08 — `opacity: var(--hub-dim-opacity)` on exit-0 only):
```tsx
<article
  className={`hub-card${status === 'stopped-ok' ? ' hub-card--dim' : ''}`}
  aria-label={`${name}, ${label}, ${cli}, ${isLocal ? 'Local' : hostname}`}
  tabIndex={0}
>
```

**Origin marker pattern** (CARD-04):
```tsx
{isLocal ? (
  <span className="hub-card__origin">
    <ComputerDesktopIcon className="hub-card__origin-icon" aria-hidden="true" />
    <span>Local</span>
  </span>
) : (
  <span className="hub-card__origin">
    <GlobeAltIcon className="hub-card__origin-icon" aria-hidden="true" />
    <span>{hostname}</span>
  </span>
)}
```

**Viewer count pattern** (CARD-05 — only when `viewerCount > 0`):
```tsx
{viewerCount > 0 && (
  <span className="hub-card__viewers">
    <EyeIcon className="hub-card__viewers-icon" aria-hidden="true" />
    <span>{viewerCount} {viewerCount === 1 ? 'viewer' : 'viewers'}</span>
  </span>
)}
```

---

### `frontend/src/components/Hub/SessionCardGrid.tsx` (component, transform)

**Analog:** `RemoteSessionsPanel.tsx` (groups peers with header + session list per group)

**Group-by-workDir pattern** (GRID-02):
```typescript
function groupByWorkDir(sessions: SessionInfo[]): Map<string, SessionInfo[]> {
  const groups = new Map<string, SessionInfo[]>()
  for (const s of sessions) {
    const key = s.workDir || ''
    const group = groups.get(key) ?? []
    group.push(s)
    groups.set(key, group)
  }
  return groups
}
```

**Group header pattern** (mirrors `remote-panel__peer-header` at style.css line 1577):
```tsx
// Group header: 11px / 600 / uppercase / letter-spacing 0.08em
// Mirrors .remote-panel__peer-header CSS pattern
<h2 className="hub__group-header" role="heading" aria-level={2}>
  <span title={workDir || undefined}>
    {workDir ? basename(workDir) : 'Other'}
  </span>
</h2>
```

**Grid pattern** (GRID-01 — CSS handles layout; component provides structure):
```tsx
<div role="list" className="hub__card-row">
  {group.map((s) => (
    <div role="listitem" key={s.id}>
      <SessionCard session={s} onRename={onRename} />
    </div>
  ))}
</div>
```

---

### `frontend/src/components/Hub/HubFilterBar.tsx` (component, event-driven)

**Analog:** `TabBar.tsx` (owns selection state, fires callbacks on change)

**Props and filter type:**
```typescript
export type HubFilter = 'all' | 'running' | 'waiting' | 'stopped-ok' | 'stopped-err' | 'idle'

export interface HubFilterBarProps {
  sessions: SessionInfo[]        // for computing live counts
  activeFilter: HubFilter
  searchText: string
  searchRef: React.RefObject<HTMLInputElement>
  onFilterChange: (f: HubFilter) => void
  onSearchChange: (text: string) => void
  onNewSession: () => void
}
```

**Filter pill pattern:**
```tsx
{FILTER_PILLS.map(({ key, label }) => (
  <button
    key={key}
    className={`hub-filter__pill${activeFilter === key ? ' hub-filter__pill--active' : ''}`}
    onClick={() => onFilterChange(key)}
  >
    {label}{key !== 'all' && ` (${counts[key] ?? 0})`}
  </button>
))}
```

**Search input pattern** (GRID-05):
```tsx
<input
  ref={searchRef}
  className="hub-filter__search"
  type="text"
  placeholder="Search sessions…"
  aria-label="Search sessions by name, CLI, or host"
  value={searchText}
  onChange={(e) => onSearchChange(e.target.value)}
  onKeyDown={(e) => { if (e.key === 'Escape') { onSearchChange(''); e.currentTarget.blur() } }}
/>
```

---

### `frontend/src/components/Hub/HubEmptyState.tsx` (component)

**Analog:** `WelcomeTab.tsx` (empty/onboarding state with CTA button)

**Two-variant pattern** (no sessions vs. filter matches nothing):
```typescript
export interface HubEmptyStateProps {
  variant: 'no-sessions' | 'no-matches'
  onNewSession?: () => void
  onClearFilter?: () => void
}
```

```tsx
export function HubEmptyState({ variant, onNewSession, onClearFilter }: HubEmptyStateProps) {
  if (variant === 'no-sessions') {
    return (
      <div className="hub__empty-state">
        <h2 className="hub__empty-heading">No sessions yet</h2>
        <p className="hub__empty-body">Create a session to start an AI coding agent.</p>
        <button className="hub__empty-cta" onClick={onNewSession}>New session</button>
      </div>
    )
  }
  return (
    <div className="hub__empty-state">
      <h2 className="hub__empty-heading">No matching sessions</h2>
      <p className="hub__empty-body">Clear the filter or search to see all sessions.</p>
      <button className="hub__empty-cta" onClick={onClearFilter}>Clear filter</button>
    </div>
  )
}
```

---

### `frontend/src/components/Hub/InlineSessionName.tsx` (component, request-response)

**Analog:** `TabBar.tsx` lines 140–172 (rename input pattern) and lines 203–225 (inline render)

This is the most exact analog — copy the TabBar rename pattern into a standalone component.

**Imports:**
```typescript
import React, { useState, useRef, useEffect } from 'react'
import { RenameSession } from '../../wailsjs/go/main/App'
```

**Core pattern** (mirrors TabBar.tsx lines 140–172 and 203–225):
```typescript
export function InlineSessionName({ id, name, onRenamed }: InlineSessionNameProps) {
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(name)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => { if (editing) inputRef.current?.select() }, [editing])

  async function commitEdit() {
    const trimmed = editValue.trim()
    if (trimmed.length > 0 && trimmed !== name) {
      await RenameSession(id, trimmed).catch(() => setEditValue(name))
      onRenamed?.(trimmed)
    } else {
      setEditValue(name)
    }
    setEditing(false)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') void commitEdit()
    if (e.key === 'Escape') { setEditValue(name); setEditing(false) }
  }

  return editing ? (
    <input
      ref={inputRef}
      className="tab__rename-input"   // reuse existing CSS — same visual contract
      value={editValue}
      placeholder="Session name"
      onChange={(e) => setEditValue(e.target.value)}
      onBlur={() => void commitEdit()}
      onKeyDown={handleKeyDown}
      onClick={(e) => e.stopPropagation()}
    />
  ) : (
    <span className="hub-card__name" onClick={() => setEditing(true)}>
      {name}
    </span>
  )
}
```

**Key reuse:** `className="tab__rename-input"` reuses the existing CSS rule at `style.css` line 142:
```css
.tab__rename-input {
  flex: 1;
  background: #1e2030;
  border: 1px solid #7aa2f7;   /* accent color focus ring */
  border-radius: 2px;
  color: #c0caf5;
  font-size: 13px;
  padding: 1px 4px;
  outline: none;
  font-family: inherit;
}
```

---

### `frontend/src/style.css` (config — append Hub CSS custom properties)

**Analog:** `style.css` lines 1577–1586 (`.remote-panel__peer-header` group header pattern); lines 1045–1060 (tab agent badge pattern); lines 1424–1435 (daemon-panel status dot pattern)

**CSS custom property root block** (append after all existing rules):
```css
/* ============================================================
   Phase 131 — Hub Surface
   All Hub components use var(--hub-*) tokens.
   No hardcoded hex values in .hub-* or .hub__* rules.
   ============================================================ */

:root {
  --hub-bg: #1a1b26;
  --hub-surface: #16161e;
  --hub-surface-elevated: #1e2030;
  --hub-border: #292e42;
  --hub-border-hover: #3b4261;
  --hub-text-primary: #c0caf5;
  --hub-text-secondary: #a9b1d6;
  --hub-text-muted: #9aa5ce;
  --hub-text-placeholder: #414868;
  --hub-accent: #7aa2f7;
  --hub-accent-hover: #89b4fa;
  --hub-destructive: #f7768e;
  --hub-success: #9ece6a;
  --hub-warning: #f59e0b;
  --hub-dim-opacity: 0.45;
  --hub-card-dim-bg: #13141f;
  --hub-group-header-bg: #16161e;
  --hub-filter-pill-bg: #1e2030;
  --hub-filter-pill-active-bg: rgba(122, 162, 247, 0.15);
  --hub-empty-icon-color: #3b4261;
  --hub-scrollbar: #3b4261;
  --hub-scrollbar-hover: #565f89;
}

[data-ui-theme="light"] {
  --hub-bg: #f5f5f7;
  --hub-surface: #ffffff;
  --hub-surface-elevated: #ececf0;
  --hub-border: #d1d1db;
  --hub-border-hover: #9999b0;
  --hub-text-primary: #1a1b26;
  --hub-text-secondary: #3a3b50;
  --hub-text-muted: #5c5d80;
  --hub-text-placeholder: #9999b0;
  --hub-accent: #3d6fe8; /* HUB-04 LIGHT THEME: verified WCAG AA 4.5:1 on #ffffff */
  --hub-accent-hover: #2a56cf;
  --hub-destructive: #c0394f; /* HUB-04 LIGHT THEME: verified WCAG AA 4.7:1 on #ffffff */
  --hub-success: #2e7d32;
  --hub-warning: #b45309;
  --hub-dim-opacity: 0.45;
  --hub-card-dim-bg: #ebebef;
  --hub-group-header-bg: #ebebef;
  --hub-filter-pill-bg: #ececf0;
  --hub-filter-pill-active-bg: rgba(61, 111, 232, 0.12);
  --hub-empty-icon-color: #c5c5d0;
  --hub-scrollbar: #c5c5d0;
  --hub-scrollbar-hover: #9999b0;
}
```

**Grid layout** (GRID-01):
```css
.hub__card-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 8px;
  max-width: 1440px;
}
.hub-card {
  min-width: 240px;
  max-width: 360px;
}
```

**Group header** (mirrors `.remote-panel__peer-header` at line 1577):
```css
.hub__group-header {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--hub-text-muted);
  height: 32px;
  background: var(--hub-group-header-bg);
  /* ... alignment ... */
}
```

**Dimmed card** (CARD-08):
```css
.hub-card--dim {
  opacity: var(--hub-dim-opacity);
  background: var(--hub-card-dim-bg);
}
/* Error cards are NOT dimmed — full opacity + ExclamationCircleIcon */
```

**Running spin animation — MUST be inside prefers-reduced-motion guard:**
```css
@media (prefers-reduced-motion: no-preference) {
  .hub-card__status-icon--spin {
    animation: hub-spin 0.8s linear infinite;
  }
}
@keyframes hub-spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}
```

**Sidebar active item** (Pitfall-8):
```css
.sidebar__item--active {
  background: rgba(122, 162, 247, 0.12);
  color: var(--hub-accent, #7aa2f7);
}
```

---

## Shared Patterns

### Wails RPC Mocking (all Hub test files)
**Source:** `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx` lines 11–26

```typescript
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  ListSessions: vi.fn().mockResolvedValue([]),
}))
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
}))
```

### React DOM Render Helpers (all Hub test files)
**Source:** `frontend/src/components/__tests__/Sidebar.test.tsx` lines 1–28

```typescript
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

function renderComponent(props = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => { root.render(<ComponentUnderTest {...props} />) })
  return { container, root }
}
```

### CSS Contract Test Pattern (style.hub.test.ts)
**Source:** `frontend/src/components/__tests__/style.contrast.test.ts` lines 1–7 and `Sidebar.test.tsx` line 10

```typescript
import { readFileSync } from 'fs'
import { resolve } from 'path'
const cssRaw = readFileSync(resolve(__dirname, '../../style.css'), 'utf-8')

it('declares --hub-bg custom property', () => {
  expect(cssRaw).toContain('--hub-bg')
})
it('grid uses auto-fill minmax(240px, 1fr) (GRID-01)', () => {
  expect(cssRaw).toContain('repeat(auto-fill, minmax(240px, 1fr))')
})
```

### Source-Inspection Test Pattern (App tests)
**Source:** `frontend/src/components/__tests__/App.nav.test.tsx` lines 1–3

```typescript
import raw from '../../App.tsx?raw'
// Use raw source inspection for wiring tests — avoids mounting the full Wails component tree
it('wires onOpenHub to Sidebar', () => {
  expect(raw).toContain('onOpenHub={handleOpenHub}')
})
```

### Error Handling in Async Event Handlers
**Source:** `DaemonManagerPanel.tsx` lines 85–100 (handleToggleFilesWrite pattern)

All async handlers in Hub components use try/catch, never `.catch()` chains:
```typescript
async function handleAction(): Promise<void> {
  try {
    await SomeRPC()
    // update state on success
  } catch {
    // set error state — never swallow silently
  }
}
```

---

## No Analog Found

All files have analogs. No entries in this section.

---

## Metadata

**Analog search scope:** `frontend/src/`, `frontend/src/components/`, `internal/daemon/`, `app.go`
**Files scanned:** 14 source files + 48 test files
**Key reuse highlights:**
- `tab__rename-input` CSS class reused directly in `InlineSessionName` — no new rename-input CSS
- `tab__agent-badge--{cli}` hex constants from `style.css` lines 1054–1060 reused in Hub card badges
- `daemon-panel__status` hex values (#3b82f6, #22c55e, #f59e0b, #ef4444) are the same values that go into status dot CSS custom properties
- `.remote-panel__peer-header` CSS (line 1577) is the direct template for `.hub__group-header` (11px/600/uppercase/letter-spacing 0.08em)
- DaemonManagerPanel polling pattern (lines 873–894) is the exact template for HubPanel's session poll in App.tsx
- `handleOpenRemoteSessions` (lines 1071–1079) is the exact template for `handleOpenHub`
**Pattern extraction date:** 2026-06-16
