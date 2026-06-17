import React, { useState, useRef, useEffect, useCallback } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { GetSessionTailLines } from '../../wailsjs/go/main/App'
import { ExclamationCircleIcon } from '@heroicons/react/24/outline'
import { HubFilterBar } from './HubFilterBar'
import type { HubFilter } from './HubFilterBar'
import { SessionCardGrid } from './SessionCardGrid'
import { HubEmptyState } from './HubEmptyState'
import { GroupSidebar } from './GroupSidebar'
// WR-01: deriveHubStatus extracted to shared util (was triplicated across SessionCard/HubFilterBar/HubPanel)
// ATTN-01/04: isAttentionStatus is the single canonical attention predicate; used for live set + debounced sort key
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'
import {
  loadGroups,
  createGroup,
  assignToGroup,
  removeFromGroup,
  memberKey,
  type HubGroupDef,
} from '../../lib/hubGroups'

/**
 * filterSessions — apply status filter + case-insensitive substring search.
 *
 * @param sessions  - full session list (from prop)
 * @param filter    - active HubFilter key ('all' means no filter)
 * @param search    - trimmed search string ('' means no search)
 * @returns filtered subset, preserving input order
 */
export function filterSessions(
  sessions: SessionInfo[],
  filter: HubFilter,
  search: string,
): SessionInfo[] {
  const needle = search.toLowerCase()
  return sessions.filter((s) => {
    // Status filter
    if (filter !== 'all') {
      const status = deriveHubStatus(s)
      if (status !== filter) return false
    }

    // Search filter — case-insensitive substring on name, cli, and hostname
    if (needle) {
      const inName = s.name.toLowerCase().includes(needle)
      const inCli = s.cli.toLowerCase().includes(needle)
      const inHost = s.hostname.toLowerCase().includes(needle)
      if (!inName && !inCli && !inHost) return false
    }

    return true
  })
}

// ---- usePreviewPoller ----

// CARD-07: single shared 3s poller drives all card previews — NOT per-card intervals
// T-132-12: stable `sessionIdKey` dep prevents polling storm on array reference churn
function usePreviewPoller(
  sessions: SessionInfo[],
  isActive: boolean,
): Map<string, string[]> {
  const [tails, setTails] = useState<Map<string, string[]>>(new Map())

  // Caller passes LOCAL sessions only (see call site). Local sessions carry the
  // machine's own hostname (os.Hostname()), so a hostname check is NOT a valid
  // local/remote discriminator — provenance (which prop) is. Dep key = all ids passed.
  const sessionIdKey = sessions.map((s) => s.id).join(',')

  useEffect(() => {
    if (!isActive || sessions.length === 0) return
    let cancelled = false

    async function poll() {
      // All passed sessions are local (caller filters by provenance). Remote sessions
      // have no tail API and are excluded by the caller, not by hostname.
      const localSessions = sessions
      if (localSessions.length === 0) return
      const results = await Promise.all(
        localSessions.map((s) =>
          GetSessionTailLines(s.id, 4).catch(() => [] as string[])
        )
      )
      if (!cancelled) {
        // CR-03: merge results into the previous map instead of replacing it wholesale.
        // When a session is stopped/killed, GetSessionTailLines returns [] (the hub has
        // been removed). Replacing the whole map would flip the preview back to "No output
        // yet". Instead, only overwrite an entry when we received real lines OR when there
        // is no prior value for that session (first fetch). This preserves the last-seen
        // snapshot for sessions that return empty due to error or cleanup.
        setTails((prev) => {
          const next = new Map(prev)
          localSessions.forEach((s, i) => {
            const lines = results[i]
            if (lines.length > 0 || !prev.has(s.id)) {
              next.set(s.id, lines)
            }
          })
          return next
        })
      }
    }

    void poll()
    const interval = setInterval(() => void poll(), 3000)
    return () => { cancelled = true; clearInterval(interval) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionIdKey, isActive])

  return tails
}

/* ATTN-04: debounce hook — controls reorder ORDER only, never card content */
/* Uses useRef + setTimeout — does NOT add a second periodic timer; the single interval is in usePreviewPoller */
function useDebouncedValue<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = React.useState<T>(value)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (timerRef.current !== null) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => setDebouncedValue(value), delay)
    return () => { if (timerRef.current !== null) clearTimeout(timerRef.current) }
  }, [value, delay])

  return debouncedValue
}

// ---- Props ----

export interface HubPanelProps {
  /** Session list polled by App.tsx (Plan 05 wires the polling). */
  sessions: SessionInfo[]
  /** True when the last ListSessions() call threw — triggers error state. */
  error: boolean
  /** Opens the NewSessionModal (handled by App.tsx). */
  onNewSession: () => void
  /** Fired when the user commits an inline rename. */
  onRename: (id: string, name: string) => void
  /** Opens (or focuses) the session's terminal tab. Phase 131 UAT follow-up. */
  onOpenSession?: (sessionId: string, name: string, cli: string) => void
  /** Phase 132 / GRID-07 — remote sessions from App.tsx (already adapted to SessionInfo[]) */
  remoteSessions?: SessionInfo[]
  /** Phase 132 / CARD-07 — true when Hub tab is active (gates usePreviewPoller interval) */
  isActive?: boolean
}

const SIDEBAR_COLLAPSED_KEY = 'hub-group-sidebar-collapsed'

// ---- Component ----

/**
 * HubPanel — top-level Hub surface.
 *
 * Owns filter + search state; applies filtering; composes:
 *   - .hub__header  (title + New session button)
 *   - HubFilterBar  (sticky; owns searchRef; passes state + callbacks)
 *   - .hub__body (flex row)
 *       → GroupSidebar  (named groups, collapsed/expanded, localStorage)
 *       → .hub__grid-scroll
 *           → error state     when error=true
 *           → no-sessions     when sessions.length === 0
 *           → no-matches      when filtered.length === 0 but sessions.length > 0
 *           → SessionCardGrid otherwise
 *
 * Keyboard shortcut (GRID-05): '/' focuses the search input when no input is active.
 *
 * Error-state copy (UI-SPEC Copywriting Contract):
 *   Heading: "Couldn't load sessions"
 *   Body:    "Check that the daemon is running and try again."
 */
export function HubPanel({
  sessions,
  error,
  onNewSession,
  onRename,
  onOpenSession,
  remoteSessions,
  isActive,
}: HubPanelProps): React.ReactElement {
  const [activeFilter, setActiveFilter] = useState<HubFilter>('all')
  const [searchText, setSearchText] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  // Phase 132 — named group state (localStorage-persisted)
  const [groupDefs, setGroupDefs] = useState<HubGroupDef[]>(() => loadGroups())
  const [activeGroupId, setActiveGroupId] = useState<string | null>(null)

  // Phase 132 — sidebar collapsed state (localStorage-persisted, mirrors Sidebar.tsx pattern)
  // WR-05: wrap in try/catch — localStorage.getItem throws SecurityError in private browsing
  // with "block all cookies" or when WebView storage quota is exhausted. loadGroups() already
  // guards its own localStorage access (hubGroups.ts line 17-21); this matches that pattern.
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(() => {
    try {
      return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true'
    } catch {
      return false
    }
  })

  const handleSidebarToggle = useCallback(() => {
    setSidebarCollapsed((prev) => {
      const next = !prev
      try {
        localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(next))
      } catch {
        // SecurityError (private browsing / "block all cookies") or QuotaExceededError
        // — collapse state lives only in memory for this session; not fatal.
      }
      return next
    })
  }, [])

  // '/' shortcut — focus search input when no input is focused (GRID-05)
  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === '/' && (document.activeElement as HTMLElement)?.tagName !== 'INPUT') {
        e.preventDefault()
        searchRef.current?.focus()
        searchRef.current?.select()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [])

  // Reset both filter and search (used by Clear-filter CTA)
  const handleClearFilter = useCallback(() => {
    setActiveFilter('all')
    setSearchText('')
  }, [])

  // Phase 132 / GRID-07: merge local + remote sessions into one unified grid
  const allSessions = [...sessions, ...(remoteSessions ?? [])]

  // Phase 132 / CARD-07: single shared 3s poller — LOCAL sessions only (the `sessions`
  // prop). Local sessions carry the machine hostname, so we must NOT use hostname to
  // decide local-vs-remote — provenance (this prop vs remoteSessions) is the discriminator.
  const localPreviewTails = usePreviewPoller(sessions, isActive ?? false)
  // Remote sessions have no tail API: seed them as empty ([] → "No output yet", not a
  // perpetual "Loading…" placeholder).
  const previewTails = React.useMemo(() => {
    const m = new Map(localPreviewTails)
    for (const r of remoteSessions ?? []) {
      if (!m.has(r.id)) m.set(r.id, [])
    }
    return m
  }, [localPreviewTails, remoteSessions])

  /* ATTN-02: float-to-top reorder is within-group; debounce window 1000ms */
  /* Only POSITION is debounced — card content (isAttention) uses the LIVE set above */
  const attentionSortKey = allSessions
    .map((s) => `${s.id}:${isAttentionStatus(deriveHubStatus(s)) ? '1' : '0'}`)
    .join(',')
  const debouncedSortKey = useDebouncedValue(attentionSortKey, 1000)

  /* ATTN-01: live attention set — NOT debounced; card border/icon update immediately */
  /* IN-02: memoized on attentionSortKey (encodes all id:bit changes) so the Set keeps
     referential identity across preview-poll renders where attention membership is stable.
     This avoids unnecessary SessionCardGrid re-renders from always-new Set references. */
  const attentionIds = React.useMemo(
    () => new Set(allSessions.filter((s) => isAttentionStatus(deriveHubStatus(s))).map((s) => s.id)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [attentionSortKey], // attentionSortKey encodes all id:bit changes
  )

  // Apply status filter + search to allSessions
  const filtered = filterSessions(allSessions, activeFilter, searchText)

  // Phase 132 / GRID-03: apply named-group filter when a group is selected
  // When activeGroupId is '__other__', show sessions not in any named group.
  // When activeGroupId is a group id, show sessions whose memberKey is in that group.
  // When activeGroupId is null, show all filtered sessions.
  let visibleSessions = filtered
  if (activeGroupId !== null && groupDefs.length > 0) {
    if (activeGroupId === '__other__') {
      const allMemberKeys = new Set(groupDefs.flatMap((g) => g.memberKeys))
      visibleSessions = filtered.filter((s) => !allMemberKeys.has(memberKey(s.name, s.workDir)))
    } else {
      const activeDef = groupDefs.find((g) => g.id === activeGroupId)
      if (activeDef) {
        const memberKeySet = new Set(activeDef.memberKeys)
        visibleSessions = filtered.filter((s) => memberKeySet.has(memberKey(s.name, s.workDir)))
      }
    }
  }

  // Group CRUD callbacks
  const handleCreateGroup = useCallback((name: string) => {
    setGroupDefs((prev) => createGroup(prev, name))
  }, [])

  const handleDropOnGroup = useCallback((groupId: string, key: string) => {
    setGroupDefs((prev) =>
      groupId === '__other__' ? removeFromGroup(prev, key) : assignToGroup(prev, groupId, key)
    )
  }, [])

  const handleAssignGroup = useCallback((mKey: string, groupId: string) => {
    setGroupDefs((prev) =>
      groupId === '__other__' ? removeFromGroup(prev, mKey) : assignToGroup(prev, groupId, mKey)
    )
  }, [])

  // ---- Determine which body to render ----
  let body: React.ReactNode

  if (error) {
    body = (
      <div className="hub__error-state">
        <ExclamationCircleIcon className="hub__error-icon" aria-hidden="true" />
        {/* UI-SPEC Copywriting Contract — error-state copy */}
        <h2 className="hub__error-heading">{"Couldn't load sessions"}</h2>
        <p className="hub__error-body">Check that the daemon is running and try again.</p>
      </div>
    )
  } else if (allSessions.length === 0) {
    body = <HubEmptyState variant="no-sessions" onNewSession={onNewSession} />
  } else if (visibleSessions.length === 0) {
    body = <HubEmptyState variant="no-matches" onClearFilter={handleClearFilter} />
  } else {
    body = (
      <SessionCardGrid
        sessions={visibleSessions}
        onRename={onRename}
        onOpenSession={onOpenSession}
        groupDefs={groupDefs.length > 0 ? groupDefs : undefined}
        previewTails={previewTails}
        onAssignGroup={handleAssignGroup}
        attentionIds={attentionIds}
        debouncedSortKey={debouncedSortKey}
      />
    )
  }

  return (
    <div className="hub">
      {/* Header strip — UI-SPEC Layout Contract */}
      <div className="hub__header">
        <span className="hub__title">Hub</span>
        <button className="hub__new-session-btn" type="button" onClick={onNewSession}>
          New session
        </button>
      </div>

      {/* Filter bar — sticky; owns searchRef; passes live session list for counts */}
      <HubFilterBar
        sessions={allSessions}
        activeFilter={activeFilter}
        searchText={searchText}
        searchRef={searchRef}
        onFilterChange={setActiveFilter}
        onSearchChange={setSearchText}
        onNewSession={onNewSession}
      />

      {/* Phase 132: hub__body is a flex row wrapping GroupSidebar + hub__grid-scroll */}
      <div className="hub__body">
        <GroupSidebar
          groupDefs={groupDefs}
          sessions={allSessions}
          activeGroupId={activeGroupId}
          collapsed={sidebarCollapsed}
          onToggle={handleSidebarToggle}
          onGroupSelect={setActiveGroupId}
          onCreateGroup={handleCreateGroup}
          onDropOnGroup={handleDropOnGroup}
        />

        {/* Scrollable grid area */}
        <div className="hub__grid-scroll">
          {body}
        </div>
      </div>
    </div>
  )
}
