# Phase 131: Hub Foundation + Static Session Cards - Research

**Researched:** 2026-06-16
**Domain:** React/TypeScript frontend UI, Go Wails bindings, BEM CSS
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Claude's Discretion
All implementation choices.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HUB-01 | User can open the Hub from a "Hub" item in the left sidebar | Sidebar.tsx accepts callback props; add `onOpenHub` prop + `Squares2X2Icon` item |
| HUB-02 | Hub is a top-level surface alongside existing panels, coexisting with Sessions | Add `HUB_TAB` constant in App.tsx; render `<HubPanel>` when active; no replacement of existing panels |
| HUB-03 | When no sessions exist, Hub shows an empty state | `HubEmptyState` component; show when `sessions.length === 0` and no filter active |
| HUB-04 | Hub renders correctly in both light and dark themes | CSS custom properties on `.hub`; `[data-ui-theme="light"]` override block |
| CARD-01 | Each card shows session name, inline-editable | `InlineSessionName` component mirroring `tab__rename-input` + `RenameSession` RPC |
| CARD-02 | Each card shows CLI/agent badge using existing per-CLI color/badge mapping | Reuse existing `#7aa2f7`/`#9ece6a`/etc. hex constants; text label required for colorblind safety |
| CARD-03 | Each card shows status indicator by shape + icon + motion, not color alone | Heroicons + text label; each status has unique icon shape |
| CARD-04 | Each card shows origin marker — local vs remote + peer hostname | `s.hostname` field; local = `ComputerDesktopIcon` + "Local", remote = `GlobeAltIcon` + hostname |
| CARD-05 | Each card shows viewer count when web-shared | `s.viewerCount` field — requires Go struct fix (see Critical Gap below) |
| CARD-06 | Each card shows uptime while running, or duration + exit code once stopped | `s.createdAt` + `s.duration`/`s.exitCode` — requires Go struct fix (see Critical Gap below) |
| CARD-08 | Stopped/exit-0 cards render dimmed; error-exit cards full opacity | `opacity: var(--hub-dim-opacity)` on exit-0 only; `ExclamationCircleIcon` on non-zero |
| GRID-01 | Responsive grid reflows by viewport width | `grid-template-columns: repeat(auto-fill, minmax(240px, 1fr))` |
| GRID-02 | Cards auto-grouped by working directory | `s.workDir` field — requires Go struct fix (see Critical Gap below) |
| GRID-04 | Status filter bar (All/Working/Needs input/Complete/Error/Idle) with live counts | `HubFilterBar` component; client-side filter on `sessions` array |
| GRID-05 | Functional search field filtering by name/CLI/host; `/` shortcut | Search input in `HubFilterBar`; `keydown` on Hub surface |
| GRID-06 | "New session" on Hub opens existing create flow | `setShowNewSessionModal(true)` callback from App.tsx; no changes to `NewSessionModal` |
</phase_requirements>

---

## Summary

Phase 131 adds the Hub — a new top-level surface in the AgentHub Wails desktop app that displays all sessions as a responsive card grid. The frontend is **React/TypeScript** (not Vue — confirmed from codebase inspection), using Heroicons, vitest for testing, and a hand-rolled BEM CSS pattern in `style.css`. No new third-party libraries are needed.

The most important discovery is a **data gap in the Go Wails bindings**: the `app.go`-level `SessionInfo` struct is missing `ViewerCount`, `ExitCode`, `Duration`, and `WorkDir` fields that exist in the underlying `daemon.SessionInfo`. The TypeScript hand-maintained type already declares them, but the Go struct does not propagate them from the daemon. Phase 131 must fix this gap before the card data layer can be complete.

`WorkDir` is stored separately in the engine's `sessionWorkDirs` map and is NOT in `daemon.SessionInfo` at all — it also needs to be added to `daemon.SessionInfo`, populated in `engine.ListSessions()`, and then propagated through `app.ListSessions()`.

The sidebar navigation pattern is simple callback-prop based: add an `onOpenHub` prop to `Sidebar.tsx` and wire it from `App.tsx`. The Hub panel itself owns session polling (3s interval when active, matching the DaemonManagerPanel pattern). All session data is available from the existing `ListSessions()` RPC.

**Primary recommendation:** Fix the Go data gap first (Wave 0), then build the Hub surface components (Wave 1-2), then integrate sidebar navigation and App.tsx wiring (Wave 3), then CSS (Wave 4).

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Session data | Go backend (daemon engine) | Wails RPC binding | Sessions live in the daemon process; frontend polls via Wails-generated RPC |
| Hub navigation | Frontend (App.tsx) | Sidebar.tsx | Tab/panel routing is App.tsx state; Sidebar just fires callbacks |
| Session polling | Frontend (HubPanel.tsx) | — | Mirrors DaemonManagerPanel pattern; poll only when Hub is active |
| Card rendering | Frontend (SessionCard.tsx) | — | Pure presentational; derives all data from `SessionInfo` |
| Status filtering | Frontend (HubFilterBar.tsx) | — | Client-side filter on the fetched sessions array |
| Inline rename | Frontend (InlineSessionName.tsx) | Go RPC (RenameSession) | UI state in component; mutation via existing RenameSession binding |
| Theme variables | Frontend (style.css) | — | CSS custom properties on `.hub`; no backend involvement |
| Working-dir grouping | Frontend (SessionCardGrid.tsx) | — | Groups sessions by `workDir` field from the already-fetched array |

---

## Standard Stack

### Core (all already installed — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 18.x | Component rendering | Already the app's UI framework [VERIFIED: package.json] |
| TypeScript | 5.9.x | Type safety | Already the project language [VERIFIED: package.json] |
| @heroicons/react | installed | Icons (status, origin, viewer, search) | Already used in Sidebar.tsx; `@heroicons/react/24/outline` import confirmed [VERIFIED: Sidebar.tsx] |
| Wails v2 | installed | Desktop app RPC bridge | The whole app framework [VERIFIED: app.go] |

### Supporting (already installed)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| vitest | ^4.1.0 | Unit tests | All component tests [VERIFIED: package.json] |
| jsdom | ^29.0.0 | DOM environment for tests | Already configured in vite.config.ts [VERIFIED: vite.config.ts] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| CSS custom properties | Tailwind / Vuetify | UI-SPEC explicitly requires hand-rolled BEM + TokyoNight; no CSS framework |
| CSS Grid auto-fill | Flexbox wrap | Grid auto-fill gives exact min/max card width control (240px–360px) |
| useInterval polling | EventsOn live stream | Polling is simpler and consistent with DaemonManagerPanel pattern; live events not needed for static cards |

**Installation:** No new packages required.

---

## Package Legitimacy Audit

No new external packages are installed in Phase 131. All libraries (React, Heroicons, vitest) are already in `package.json`.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
User clicks "Hub" in Sidebar
        │
        ▼
App.tsx (activeId = '__hub__')
        │
        ├─ polls ListSessions() every 3s while Hub active
        │   └─ Wails RPC → app.go ListSessions() → DaemonClient.ListSessions()
        │       └─ HTTP GET /sessions → daemon engine.ListSessions()
        │           └─ SessionInfo{id, cli, name, state, status, hostname,
        │                         webEnabled, viewerCount, exitCode, duration,
        │                         workDir, createdAt, ...}
        │
        └─ renders <HubPanel sessions={sessions} ...>
                │
                ├─ HubFilterBar (filter pill + search state)
                │   └─ fires filter/search changes up to HubPanel
                │
                └─ SessionCardGrid (filtered sessions, grouped by workDir)
                        └─ per group: .hub__group-header + SessionCard × N
                                └─ SessionCard
                                    ├─ InlineSessionName (click-to-rename)
                                    ├─ Status icon + text (Heroicons)
                                    ├─ CLI badge chip (text + color dot)
                                    ├─ Origin marker (local / remote hostname)
                                    ├─ Viewer count (if > 0)
                                    └─ Uptime / duration + exit code
```

### Recommended Project Structure

```
frontend/src/components/
├─ Hub/
│   ├─ HubPanel.tsx          # Top-level surface; owns filter/search state + session polling
│   ├─ SessionCard.tsx        # Individual card; receives SessionInfo as prop
│   ├─ SessionCardGrid.tsx    # Responsive grid container with working-dir group headers
│   ├─ HubFilterBar.tsx       # Filter pills + search input
│   ├─ HubEmptyState.tsx      # Empty-state when no sessions (or no filter match)
│   └─ InlineSessionName.tsx  # Inline-editable session name (mirrors TabBar rename)
```

### Pattern 1: Sidebar Navigation Prop Extension

**What:** Extend Sidebar.tsx with an `onOpenHub` callback prop and `Squares2X2Icon` button. App.tsx passes `handleOpenHub` which sets `activeId` to `'__hub__'`.

**When to use:** Every new top-level surface follows this pattern (DaemonManagerPanel, RemoteSessionsPanel all use it).

**Example (from existing Sidebar.tsx):**
```typescript
// Source: frontend/src/components/Sidebar.tsx (VERIFIED: codebase)
interface SidebarProps {
  onHome: () => void
  onOpenRemoteSessions: () => void
  onOpenDaemonManager: () => void
  onAdd: () => void
  onOpenHub: () => void   // ADD THIS
  onSettings: () => void
}
```

App.tsx Tab constant to add (mirrors existing pattern):
```typescript
// Source: frontend/src/App.tsx lines 85-88 pattern (VERIFIED: codebase)
const HUB_TAB: Tab = { id: '__hub__', name: 'Hub', sessionId: '', cli: '', type: 'hub' }
```

TabBar.tsx Tab type extension needed:
```typescript
// Source: frontend/src/components/TabBar.tsx line 8 (VERIFIED: codebase)
// Current: type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions' | 'settings' | 'file-browser'
// Add: | 'hub'
```

### Pattern 2: Session Polling When Panel Active (DaemonManagerPanel pattern)

**What:** `useEffect` that polls `ListSessions()` every 3s when the Hub is the active panel. Effect cleanup cancels the interval and sets a `cancelled` flag.

**When to use:** Anytime a panel needs live session data but isn't a terminal (no event stream).

**Example (from existing DaemonManagerPanel polling in App.tsx):**
```typescript
// Source: frontend/src/App.tsx lines 873-894 (VERIFIED: codebase)
useEffect(() => {
  const isHubActive = activeId === HUB_TAB.id
  if (!isHubActive) return
  let cancelled = false
  async function refresh() {
    try {
      const sessions = await ListSessions()
      if (!cancelled) setHubSessions(sessions)
    } catch (err) {
      if (!cancelled) setHubError(true)
    }
  }
  void refresh()
  const interval = setInterval(() => void refresh(), 3000)
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])
```

### Pattern 3: Inline Rename (mirrors TabBar `tab__rename-input`)

**What:** Click on session name → replace `<span>` with `<input>` → blur/Enter commits via `RenameSession` RPC → Escape cancels.

**When to use:** Any inline-editable text field.

**Example (from existing TabBar.tsx lines 203-212):**
```typescript
// Source: frontend/src/components/TabBar.tsx (VERIFIED: codebase)
// Input uses className="tab__rename-input" (existing CSS)
// InlineSessionName reuses the same .tab__rename-input class
// so no new CSS needed for the input itself, just the hub-card wrapper
{editing ? (
  <input
    ref={inputRef}
    className="tab__rename-input"
    value={editValue}
    onChange={(e) => setEditValue(e.target.value)}
    onBlur={commitEdit}
    onKeyDown={handleKeyDown}
    onClick={(e) => e.stopPropagation()}
  />
) : (
  <span onClick={() => setEditing(true)}>{name}</span>
)}
```

### Pattern 4: Working Directory Grouping

**What:** `SessionCardGrid` uses `Array.from(new Map(...))` to group sessions by `workDir`. Sessions with empty `workDir` fall into a default "Other" group.

```typescript
// Derived from REQUIREMENTS.md GRID-02 + CARD data contract (VERIFIED: requirements)
function groupByWorkDir(sessions: SessionInfo[]): Map<string, SessionInfo[]> {
  const groups = new Map<string, SessionInfo[]>()
  for (const s of sessions) {
    const key = s.workDir || ''  // empty string = "Other" group
    const group = groups.get(key) ?? []
    group.push(s)
    groups.set(key, group)
  }
  return groups
}
```

Group header displays `path.basename(workDir)` with full path as tooltip; "Other" for empty.

### Pattern 5: Status Derivation

**What:** Map `SessionInfo.state` and `SessionInfo.status` + `exitCode` to the UI status enum.

```typescript
// Source: derived from daemon engine.go ListSessions + REQUIREMENTS.md (VERIFIED: codebase)
type HubStatus = 'running' | 'idle' | 'waiting' | 'errored' | 'stopped-ok' | 'stopped-err'

function deriveStatus(s: SessionInfo): HubStatus {
  if (s.state === 'stopped') {
    return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
  }
  // s.status is the heuristic from the detector: 'running'|'idle'|'waiting'|'errored'
  return s.status as HubStatus
}
```

### Anti-Patterns to Avoid

- **Mounting a live xterm per card:** Out of scope (CARD-07 is Phase 132). Phase 131 has NO terminal previews on cards.
- **Replacing the Sessions (DaemonManagerPanel) panel:** Hub coexists — HUB-02 explicitly requires this.
- **Color-only status differentiation:** Colorblind user requirement means every status must have a unique icon shape + visible text label.
- **Hardcoded hex in component files:** All hex values go in style.css as CSS custom properties; components reference `var(--hub-*)` tokens.
- **Polling when Hub is not active:** The `useEffect` deps must check `activeId === HUB_TAB.id` before starting the interval.

---

## Critical Data Gap — Backend Wiring Required

**This is the most important finding of this research.**

### What's Missing

The Wails-bound `app.go SessionInfo` struct is missing fields that CARD-04, CARD-05, CARD-06, and GRID-02 depend on. The TypeScript `App.d.ts` hand-maintained type declares them but the Go struct does not propagate them.

| Field | Needed By | In `daemon.SessionInfo`? | In `app.go SessionInfo`? | In `app.ListSessions()`? |
|-------|-----------|--------------------------|--------------------------|--------------------------|
| `viewerCount` | CARD-05 | YES (populated, tested) | NO | NO |
| `exitCode` | CARD-06 | YES (as `*int`) | NO | NO |
| `duration` | CARD-06 | YES (as `*int`) | NO | NO |
| `workDir` | GRID-02 | NO — stored in `sessionWorkDirs` map | NO | NO |

**Source:** [VERIFIED: codebase — app.go lines 29-46, engine.go ListSessions lines 404-471, App.d.ts lines 6-22]

### Fix Required

**Step 1:** Add `WorkDir string` to `daemon.SessionInfo` struct in `internal/daemon/types.go`.

**Step 2:** Populate `WorkDir` from `e.sessionWorkDirs[s.ID]` in `engine.ListSessions()` in `internal/daemon/engine.go`.

**Step 3:** Add `ViewerCount int`, `ExitCode *int`, `Duration *int`, `WorkDir string` to `app.go SessionInfo` struct.

**Step 4:** Propagate in `app.ListSessions()`:
```go
// In app.go ListSessions() mapping (VERIFIED: app.go lines 343-372 pattern)
result[i] = SessionInfo{
    // ... existing fields ...
    ViewerCount: s.ViewerCount,
    ExitCode:    s.ExitCode,
    Duration:    s.Duration,
    WorkDir:     s.WorkDir,
}
```

**Step 5:** Update `frontend/src/wailsjs/go/main/App.d.ts` to add `workDir: string` (the other fields are already declared).

**Risk if skipped:** CARD-05 shows 0 viewers always; CARD-06 shows no uptime/duration; GRID-02 cannot group by working directory (all sessions fall into "Other").

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Status icons | Custom SVG icon sprites | `@heroicons/react/24/outline` (already installed) | Heroicons already used in Sidebar.tsx; consistent visual language |
| Inline rename input | Custom styled input | Reuse `tab__rename-input` CSS class (existing) | Exact same visual contract; avoids introducing a parallel rename style |
| RenameSession RPC | Custom HTTP call | `RenameSession(id, name)` Wails binding (existing) | Already wired and tested |
| Session polling | WebSocket subscription | `ListSessions()` interval (DaemonManagerPanel pattern) | No live session stream exists for the Wails desktop; polling is the established pattern |
| CLI badge hex values | New color system | Existing `tab__agent-badge--{cli}` hex constants from style.css | Reuse to avoid palette divergence |

**Key insight:** The Hub is purely additive — it reuses every existing primitive (Heroicons, RenameSession, ListSessions, NewSessionModal, tab__rename-input CSS). The only truly new code is the Hub layout, card layout, filter logic, and grouping logic.

---

## Common Pitfalls

### Pitfall 1: `workDir` and `viewerCount` Always Zero
**What goes wrong:** Cards show 0 viewers and cannot group by working directory.
**Why it happens:** `app.go ListSessions()` does not propagate these fields from `daemon.SessionInfo`.
**How to avoid:** Fix the Go data gap in Wave 0 (see Critical Data Gap above) before building the card component.
**Warning signs:** `s.viewerCount === 0` even for shared sessions; all cards in "Other" group.

### Pitfall 2: `workDir` Missing From `daemon.SessionInfo`
**What goes wrong:** Even after fixing `app.go`, `workDir` is always empty because it's stored in `engine.sessionWorkDirs` map, not in the `SessionInfo` struct populated by `ListSessions()`.
**Why it happens:** `GetSessionWorkDir()` is a separate method; `ListSessions()` doesn't call it.
**How to avoid:** In `engine.ListSessions()`, add `WorkDir: e.sessionWorkDirs[s.ID]` when building the result slice (inside the already-held `e.mu.RLock()`).
**Warning signs:** All sessions have `workDir === ""` after fix.

### Pitfall 3: Hub Replaces Sessions Panel Instead of Coexisting
**What goes wrong:** Clicking Hub in the sidebar destroys the Sessions tab.
**Why it happens:** Wrong implementation of `handleOpenHub` — modifying existing panels instead of adding a new tab constant.
**How to avoid:** Follow the exact DaemonManagerPanel pattern. `HUB_TAB` is a separate const with `id: '__hub__'`. Render `<HubPanel>` when `activeId === HUB_TAB.id`. Never modify the `DAEMON_MANAGER_TAB` behavior.
**Warning signs:** Sessions panel disappears after clicking Hub.

### Pitfall 4: Tab Type Union Not Extended
**What goes wrong:** `tabs.filter((t) => t.type !== 'hub')` at App.tsx line 1438 misses Hub tab in the terminal-render exclusion.
**Why it happens:** `TabBar.tsx` Tab type doesn't include `'hub'` in its union.
**How to avoid:** Add `| 'hub'` to the TabBar Tab type union AND add `t.type !== 'hub'` to the terminal exclusion filter in App.tsx.
**Warning signs:** TypeScript error on `type: 'hub'` assignment.

### Pitfall 5: Color-Only Status Differentiation
**What goes wrong:** UAT fails because colorblind user cannot distinguish statuses.
**Why it happens:** Status rendered as colored dot without icon or text label.
**How to avoid:** Every status indicator MUST include: (1) unique Heroicon shape, (2) visible text label. Color is a reinforcing signal only.
**Warning signs:** Any `aria-label` missing from status icons; any status rendered without a text label.

### Pitfall 6: `prefers-reduced-motion` Missing on Spinner
**What goes wrong:** `ArrowPathIcon` spin animation plays even when user has reduced motion enabled.
**Why it happens:** CSS animation declared without `@media (prefers-reduced-motion: no-preference)` guard.
**How to avoid:** Wrap ALL animations in the media query per the UI-SPEC Motion Contract.
**Warning signs:** `@keyframes` or `animation:` declarations outside a `prefers-reduced-motion` block.

### Pitfall 7: Polling When Hub Is Not Active
**What goes wrong:** `ListSessions()` fires every 3s even when user is on another panel.
**Why it happens:** `useEffect` deps don't include `activeId`.
**How to avoid:** Follow the DaemonManagerPanel pattern: early-return when `activeId !== HUB_TAB.id`.

### Pitfall 8: Sidebar Active State Not Reflected
**What goes wrong:** Hub sidebar item doesn't visually indicate "current" state.
**Why it happens:** Sidebar currently has no active-state prop or CSS class — no `sidebar__item--active` exists in `style.css`.
**How to avoid:** Add `activePanel` prop to Sidebar and a `sidebar__item--active` CSS class. The UI-SPEC specifies `--hub-accent` for the active indicator (item 5 in the accent-reserved list).
**Warning signs:** All sidebar items look identical regardless of which panel is open.

---

## Code Examples

### Colorblind-Safe Status Indicator
```typescript
// Source: derived from UI-SPEC Status Indicator Contract + Heroicons (VERIFIED: UI-SPEC.md)
import {
  ArrowPathIcon, CheckCircleIcon, PauseCircleIcon,
  ExclamationCircleIcon, StopCircleIcon
} from '@heroicons/react/24/outline'

const STATUS_CONFIG = {
  running:     { Icon: ArrowPathIcon,       label: 'Running',    spin: true  },
  idle:        { Icon: CheckCircleIcon,     label: 'Idle',       spin: false },
  waiting:     { Icon: PauseCircleIcon,     label: 'Needs input',spin: false },
  errored:     { Icon: ExclamationCircleIcon,label: 'Error',     spin: false },
  'stopped-ok':{ Icon: StopCircleIcon,      label: 'Done',       spin: false },
  'stopped-err':{ Icon: ExclamationCircleIcon, label: 'Exited',  spin: false },
} as const

// EVERY status icon MUST carry aria-label and a visible text label
// (user is colorblind — color alone is NEVER sufficient)
function StatusIndicator({ status }: { status: keyof typeof STATUS_CONFIG }) {
  const { Icon, label, spin } = STATUS_CONFIG[status]
  return (
    <span className="hub-card__status-indicator">
      <Icon
        className={`hub-card__status-icon${spin ? ' hub-card__status-icon--spin' : ''}`}
        aria-label={label}
      />
      <span className="hub-card__status-label">{label}</span>
    </span>
  )
}
```

### CSS Grid Layout
```css
/* Source: UI-SPEC.md Layout Contract (VERIFIED: UI-SPEC.md) */
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

### CSS Custom Property Root Block (Dark Theme Default)
```css
/* Source: UI-SPEC.md Theme Contract (VERIFIED: UI-SPEC.md) */
/* Appended to style.css after existing rules */

/* === Hub Surface Theme Variables === */
.hub,
.hub * {
  /* All hub elements use these tokens — no hardcoded hex in Hub components */
}
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
  /* ... full list from UI-SPEC ... */
}

[data-ui-theme="light"] {
  --hub-bg: #f5f5f7;
  --hub-surface: #ffffff;
  --hub-accent: #3d6fe8; /* HUB-04 LIGHT THEME: WCAG AA 4.5:1 on #ffffff */
  --hub-destructive: #c0394f; /* HUB-04 LIGHT THEME: WCAG AA 4.7:1 on #ffffff */
  /* ... full list from UI-SPEC ... */
}

/* Running icon spin — MUST be inside prefers-reduced-motion guard */
@media (prefers-reduced-motion: no-preference) {
  .hub-card__status-icon--spin {
    animation: hub-spin 0.8s linear infinite;
  }
}
@keyframes hub-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
```

### Uptime/Duration Formatting
```typescript
// Source: derived from UI-SPEC Copywriting Contract (VERIFIED: UI-SPEC.md)
function formatUptime(createdAt: string): string {
  const seconds = Math.floor((Date.now() - new Date(createdAt).getTime()) / 1000)
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return `Ran ${h}h ${m}m`
  return `Ran ${m}m`
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-session QR / status in DaemonManagerPanel only | New Hub grid view coexists | Phase 131 | Hub is additive; Sessions panel unchanged |
| Sidebar has no active-state CSS | Add `sidebar__item--active` class | Phase 131 | Required for HUB-01 active indicator |

**Deprecated/outdated:**
- `ViewerCount`, `ExitCode`, `Duration`, `WorkDir` were aspirationally declared in the TypeScript type but never wired through the Go Wails binding. Phase 131 completes the wire-up.

---

## Project Constraints (from CLAUDE.md)

- **React/TypeScript frontend** (confirmed by codebase — NOT Vue despite CLAUDE.md listing both)
- **`pnpm` preferred** as package manager
- **TypeScript:** `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types
- **Testing:** vitest (installed at ^4.1.0); 80%+ coverage in critical components; integration tests for routes
- **No new global installs** — only project-scoped packages
- **Python not relevant** to this phase (pure frontend + Go)
- **Go:** `go fmt`, context-aware functions — relevant for the daemon SessionInfo fix

---

## Runtime State Inventory

Step 2.6: SKIPPED — Phase 131 is a frontend + Go struct fix. No renames, no migrations, no stored runtime state affected.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Frontend build | ✓ | (system) | — |
| pnpm | Package manager | ✓ | (system) | — |
| Go | Backend compile | ✓ | (system) | — |
| vitest | Frontend tests | ✓ | ^4.1.0 | — |
| @heroicons/react | Status/origin icons | ✓ | installed | — |
| Wails CLI | Dev/build | ✓ | installed | — |

All dependencies available. No missing dependencies with no fallback.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest ^4.1.0 |
| Config file | frontend/vite.config.ts (test block) |
| Quick run command | `cd frontend && pnpm test` |
| Full suite command | `cd frontend && pnpm test:coverage` |
| Go tests | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HUB-01 | Hub sidebar item present + fires callback | unit | `pnpm test` | ❌ Wave 0 — Hub/HubPanel.test.tsx |
| HUB-02 | Hub renders without replacing Sessions panel | unit | `pnpm test` | ❌ Wave 0 — App.hub.test.tsx |
| HUB-03 | Empty state shows when sessions=[] | unit | `pnpm test` | ❌ Wave 0 — Hub/HubPanel.test.tsx |
| HUB-04 | CSS custom properties declared for light+dark | unit (CSS text) | `pnpm test` | ❌ Wave 0 — style.hub.test.ts |
| CARD-01 | InlineSessionName enters edit on click, commits on Enter | unit | `pnpm test` | ❌ Wave 0 — Hub/InlineSessionName.test.tsx |
| CARD-02 | CLI badge shows text label + correct hex | unit (CSS text) | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx |
| CARD-03 | Every status has icon + text label (no color alone) | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx |
| CARD-04 | Origin marker shows "Local" vs hostname | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx |
| CARD-05 | Viewer count shown when viewerCount > 0 | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx |
| CARD-06 | Uptime shown for running; duration+exit for stopped | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx |
| CARD-08 | Stopped-ok card has opacity 0.45; stopped-err has full opacity | unit (CSS) | `pnpm test` | ❌ Wave 0 — Hub/SessionCard.test.tsx |
| GRID-01 | Grid uses auto-fill minmax(240px, 1fr) | unit (CSS text) | `pnpm test` | ❌ Wave 0 — style.hub.test.ts |
| GRID-02 | Sessions grouped by workDir; empty → "Other" | unit | `pnpm test` | ❌ Wave 0 — Hub/SessionCardGrid.test.tsx |
| GRID-04 | Filter pills filter sessions; counts accurate | unit | `pnpm test` | ❌ Wave 0 — Hub/HubFilterBar.test.tsx |
| GRID-05 | "/" shortcut focuses search; typing filters | unit | `pnpm test` | ❌ Wave 0 — Hub/HubFilterBar.test.tsx |
| GRID-06 | "New session" button fires onNewSession prop | unit | `pnpm test` | ❌ Wave 0 — Hub/HubPanel.test.tsx |
| Go data gap | engine.ListSessions includes WorkDir field | unit (Go) | `go test ./internal/daemon/...` | ❌ Wave 0 — engine_test.go addition |

### Sampling Rate
- **Per task commit:** `pnpm test` (vitest run, ~30s) + `go test ./internal/daemon/... -count=1`
- **Per wave merge:** `pnpm test:coverage` + `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/Hub/HubPanel.test.tsx` — covers HUB-01, HUB-02, HUB-03, GRID-06
- [ ] `frontend/src/components/Hub/SessionCard.test.tsx` — covers CARD-01..06, CARD-08
- [ ] `frontend/src/components/Hub/SessionCardGrid.test.tsx` — covers GRID-02
- [ ] `frontend/src/components/Hub/HubFilterBar.test.tsx` — covers GRID-04, GRID-05
- [ ] `frontend/src/components/Hub/InlineSessionName.test.tsx` — covers CARD-01 (inline edit)
- [ ] `frontend/src/components/__tests__/style.hub.test.ts` — covers HUB-04, GRID-01, CARD-08 (CSS text)
- [ ] Go test additions in `internal/daemon/engine_test.go` — WorkDir in ListSessions

---

## Security Domain

No new authentication, authorization, or capability logic is introduced. The Hub reads from `ListSessions()` which is the same data shown in the existing DaemonManagerPanel — no new security surface. The `RenameSession` RPC is already exposed and used by TabBar.

ASVS V5 input validation: inline session rename input must trim and reject empty strings (same as TabBar's existing `commitEdit` behavior).

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `session:status` EventsOn subscription in App.tsx fires for all sessions including those not in active terminal tabs | Standard Stack / Polling | If status events don't fire for sessions without open tabs, Hub cards may show stale status; polling as fallback resolves this |
| A2 | The `daemon.SessionInfo.WorkDir` field name is `WorkDir` / JSON `workDir` (matching `CreateRequest.WorkDir`) | Critical Data Gap | Field name mismatch would break JSON deserialization; trivially verified by reading types.go |

**If this table is empty for non-assumed items:** All other claims verified from codebase source files.

---

## Open Questions (RESOLVED)

1. **Sidebar active-state indicator implementation**
   - What we know: No `sidebar__item--active` CSS class exists today; no active-state prop exists on Sidebar
   - What's unclear: Should the Hub planner add a general `activePanel` prop to Sidebar (cleanest), or a specific `hubActive` boolean?
   - Recommendation: Add `activePanel?: string` prop to Sidebar so it can mark any item active; matches how the Sessions/Remote panels need it in future phases too.
   - **RESOLVED:** Plan 131-05 (Task 1) adds the general `activePanel?: string` prop to Sidebar and the `sidebar__item--active` CSS class (using `--hub-accent`), per the recommendation.

2. **Hub polling vs. using existing `panelSessions` state**
   - What we know: App.tsx polls sessions when `activeId === DAEMON_MANAGER_TAB.id` and stores in `panelSessions`; Hub has different `activeId`
   - What's unclear: Should Hub reuse `panelSessions` (shared state, needs App.tsx to change poll condition) or have its own poll?
   - Recommendation: Hub owns its own poll (simpler, more isolated). App.tsx does NOT share `panelSessions` with Hub. This mirrors how RemoteSessionsPanel has its own poll.
   - **RESOLVED:** Plan 131-05 (Task 1) gives the Hub its own `hubSessions`/`hubError` state and a dedicated 3s poll gated on `activeId === HUB_TAB.id`; `panelSessions` is left untouched, per the recommendation.

---

## Sources

### Primary (HIGH confidence)
- `frontend/src/App.tsx` — App-level routing, tab type constants, polling patterns, render patterns
- `frontend/src/components/Sidebar.tsx` — Sidebar callback props, Heroicons usage, CSS class names
- `frontend/src/wailsjs/go/main/App.d.ts` — TypeScript SessionInfo type (hand-maintained)
- `frontend/src/wailsjs/go/main/App.js` — Wails-generated bindings
- `app.go` — Go Wails-bound SessionInfo struct and ListSessions() implementation
- `internal/daemon/types.go` — daemon.SessionInfo struct (source of truth for daemon fields)
- `internal/daemon/engine.go` — ListSessions() implementation; sessionWorkDirs map
- `internal/daemon/api.go` — /sessions HTTP handler
- `frontend/src/style.css` — existing CSS classes (tab__rename-input, tab__agent-badge, daemon-panel__status, remote-panel__peer-header)
- `frontend/src/components/TabBar.tsx` — Tab type union, rename pattern
- `frontend/src/components/DaemonManagerPanel.tsx` — status rendering, SessionInfo usage
- `frontend/src/components/NewSessionModal.tsx` — modal interface (to be reused)
- `frontend/vite.config.ts` — vitest configuration
- `frontend/package.json` — dependencies and test scripts
- `.planning/phases/131-hub-foundation-static-session-cards/131-UI-SPEC.md` — locked design contract
- `.planning/REQUIREMENTS.md` — v3.6 requirements definition

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` — milestone context, key decisions
- `.planning/ROADMAP.md` — Phase 131 success criteria

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries confirmed from package.json and component imports
- Architecture: HIGH — derived directly from reading App.tsx, Sidebar.tsx, engine.go
- Critical data gap: HIGH — confirmed by cross-referencing app.go struct vs App.d.ts vs engine.go ListSessions
- Pitfalls: HIGH — derived from codebase patterns and the specific gap found
- CSS patterns: HIGH — read directly from style.css

**Research date:** 2026-06-16
**Valid until:** 2026-07-16 (stable codebase — no external dependency changes expected)
