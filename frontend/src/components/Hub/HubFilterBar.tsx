import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
// WR-01: deriveHubStatus extracted to shared util (was triplicated across SessionCard/HubFilterBar/HubPanel)
import { deriveHubStatus } from '../../lib/hubStatus'
import { PlusIcon } from '@heroicons/react/24/outline'

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
    const bucket = deriveHubStatus(s)
    // The `in` check ensures bucket is a valid HubFilter key at runtime;
    // cast is safe since HubStatus ⊇ HubFilter (minus 'all', plus 'errored').
    if (bucket in counts) {
      counts[bucket as HubFilter]++
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
    <div className="hub__filter-bar">
      {/* Filter pills */}
      {/* CR-01: hub-filter__pills child elements still match the CSS pill rules */}
      <div className="hub-filter__pills" role="group" aria-label="Session status filter">
        {FILTER_PILLS.map(({ key, label }) => (
          <button
            key={key}
            className={`hub-filter__pill${activeFilter === key ? ' hub-filter__pill--active' : ''}`}
            onClick={() => onFilterChange(key)}
            type="button"
            aria-pressed={activeFilter === key ? 'true' : 'false'}
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

      {/* New session button — POL-03: minimal comp affordance with accent PlusIcon */}
      <button
        className="hub-filter__new-session"
        onClick={onNewSession}
        type="button"
      >
        <PlusIcon className="hub-filter__new-session-icon" aria-hidden="true" />
        New session
      </button>
    </div>
  )
}
