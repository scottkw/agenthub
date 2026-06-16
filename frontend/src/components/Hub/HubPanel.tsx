import React, { useState, useRef, useEffect, useCallback } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { ExclamationCircleIcon } from '@heroicons/react/24/outline'
import { HubFilterBar } from './HubFilterBar'
import type { HubFilter } from './HubFilterBar'
import { SessionCardGrid } from './SessionCardGrid'
import { HubEmptyState } from './HubEmptyState'
// WR-01: deriveHubStatus extracted to shared util (was triplicated across SessionCard/HubFilterBar/HubPanel)
import { deriveHubStatus } from '../../lib/hubStatus'

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
}

// ---- Component ----

/**
 * HubPanel — top-level Hub surface.
 *
 * Owns filter + search state; applies filtering; composes:
 *   - .hub__header  (title + New session button)
 *   - HubFilterBar  (sticky; owns searchRef; passes state + callbacks)
 *   - .hub__grid-scroll
 *       → error state     when error=true
 *       → no-sessions     when sessions.length === 0
 *       → no-matches      when filtered.length === 0 but sessions.length > 0
 *       → SessionCardGrid otherwise
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
}: HubPanelProps): React.ReactElement {
  const [activeFilter, setActiveFilter] = useState<HubFilter>('all')
  const [searchText, setSearchText] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

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

  const filtered = filterSessions(sessions, activeFilter, searchText)

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
  } else if (sessions.length === 0) {
    body = <HubEmptyState variant="no-sessions" onNewSession={onNewSession} />
  } else if (filtered.length === 0) {
    body = <HubEmptyState variant="no-matches" onClearFilter={handleClearFilter} />
  } else {
    body = <SessionCardGrid sessions={filtered} onRename={onRename} />
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
        sessions={sessions}
        activeFilter={activeFilter}
        searchText={searchText}
        searchRef={searchRef}
        onFilterChange={setActiveFilter}
        onSearchChange={setSearchText}
        onNewSession={onNewSession}
      />

      {/* Scrollable grid area */}
      <div className="hub__grid-scroll">
        {body}
      </div>
    </div>
  )
}
