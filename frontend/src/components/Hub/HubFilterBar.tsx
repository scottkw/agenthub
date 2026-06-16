import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// ---- Types ----

/**
 * HubFilter — the set of status buckets the filter bar can select.
 * Exported so parent components (HubPanel) can own the active filter state.
 */
export type HubFilter = 'all' | 'running' | 'waiting' | 'stopped-ok' | 'stopped-err' | 'idle'

// ---- Filter pill definitions (UI-SPEC Copywriting Contract) ----

interface FilterPillDef {
  key: HubFilter
  label: string
}

const FILTER_PILLS: FilterPillDef[] = [
  { key: 'all', label: 'All' },
  { key: 'running', label: 'Working' },
  { key: 'waiting', label: 'Needs input' },
  { key: 'stopped-ok', label: 'Complete' },
  { key: 'stopped-err', label: 'Error' },
  { key: 'idle', label: 'Idle' },
]

// ---- Helpers ----

/**
 * Derive the Hub display status from a SessionInfo — mirrors SessionCard.tsx deriveStatus.
 * - state === 'stopped' + exitCode non-zero → 'stopped-err'
 * - state === 'stopped' + exitCode 0 (or undefined) → 'stopped-ok'
 * - otherwise: use session.status as HubFilter key
 */
function deriveFilterStatus(s: SessionInfo): HubFilter {
  if (s.state === 'stopped') {
    return (s.exitCode ?? 0) !== 0 ? 'stopped-err' : 'stopped-ok'
  }
  return s.status as HubFilter
}

/**
 * Compute per-bucket counts from a sessions array.
 * "All" pill omits its count; all other pills show " (N)".
 */
function computeCounts(sessions: SessionInfo[]): Record<HubFilter, number> {
  const counts: Record<HubFilter, number> = {
    all: sessions.length,
    running: 0,
    waiting: 0,
    'stopped-ok': 0,
    'stopped-err': 0,
    idle: 0,
  }
  for (const s of sessions) {
    const bucket = deriveFilterStatus(s)
    if (bucket in counts) {
      counts[bucket]++
    }
  }
  return counts
}

// ---- Props ----

export interface HubFilterBarProps {
  /** Full session list — used to compute live counts per filter bucket. */
  sessions: SessionInfo[]
  /** The currently active filter key. */
  activeFilter: HubFilter
  /** The current search text (controlled by parent). */
  searchText: string
  /** Ref forwarded to the search input — parent uses this for the "/" shortcut. */
  searchRef: React.RefObject<HTMLInputElement | null>
  /** Fired when the user clicks a filter pill. */
  onFilterChange: (filter: HubFilter) => void
  /** Fired when the search text changes (including Escape → ''). */
  onSearchChange: (text: string) => void
  /** Fired when the user clicks "New session". */
  onNewSession: () => void
}

// ---- Component ----

/**
 * HubFilterBar — status filter pills with live counts, a search input,
 * and the "New session" button.
 *
 * UI-SPEC Interaction Contract — Filter Bar:
 *   - Filter pill click → activates pill, fires onFilterChange
 *   - Typing in search → fires onSearchChange with current value
 *   - Escape in search → fires onSearchChange(''), blurs the input
 *   - "New session" click → fires onNewSession
 *   - searchRef is forwarded to the input so the parent's "/" shortcut can focus it
 */
export function HubFilterBar({
  sessions,
  activeFilter,
  searchText,
  searchRef,
  onFilterChange,
  onSearchChange,
  onNewSession,
}: HubFilterBarProps): React.ReactElement {
  const counts = computeCounts(sessions)

  return (
    <div className="hub-filter">
      {/* Filter pills */}
      <div className="hub-filter__pills" role="group" aria-label="Session status filter">
        {FILTER_PILLS.map(({ key, label }) => (
          <button
            key={key}
            className={`hub-filter__pill${activeFilter === key ? ' hub-filter__pill--active' : ''}`}
            onClick={() => onFilterChange(key)}
            type="button"
          >
            {key === 'all' ? label : `${label} (${counts[key] ?? 0})`}
          </button>
        ))}
      </div>

      {/* Search input — aria-label per Accessibility Contract (UI-SPEC §Accessibility) */}
      <input
        ref={searchRef}
        className="hub-filter__search"
        type="text"
        placeholder="Search sessions…"
        aria-label="Search sessions by name, CLI, or host"
        value={searchText}
        onChange={(e) => onSearchChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            onSearchChange('')
            e.currentTarget.blur()
          }
        }}
      />

      {/* New session button */}
      <button
        className="hub-filter__new-session"
        onClick={onNewSession}
        type="button"
      >
        New session
      </button>
    </div>
  )
}
