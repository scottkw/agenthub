import React, { useEffect, useRef } from 'react'

export interface StatusLineProps {
  /** Number of items visible after filter is applied. */
  itemCount: number
  /** Number of items before filter is applied. */
  totalCount: number
  /** Whether the filter input is currently active (showing). */
  filterActive: boolean
  /** Current filter text. */
  filterValue: string
  /** Filter input onChange. */
  onFilterChange: (v: string) => void
  /** Escape pressed inside the filter input — parent should clear+dismiss. */
  onFilterDismiss: () => void
}

/**
 * Pluralized item count copy.
 * - 1 → "1 item"
 * - n → "n items"
 */
function itemsCopy(n: number): string {
  return `${n} ${n === 1 ? 'item' : 'items'}`
}

function matchesCopy(n: number): string {
  return `${n} ${n === 1 ? 'match' : 'matches'}`
}

export function StatusLine({
  itemCount,
  totalCount,
  filterActive,
  filterValue,
  onFilterChange,
  onFilterDismiss,
}: StatusLineProps): React.ReactElement {
  const inputRef = useRef<HTMLInputElement | null>(null)

  // Auto-focus the filter input when activated so the user can type
  // immediately after pressing '/'.
  useEffect(() => {
    if (filterActive && inputRef.current) {
      inputRef.current.focus()
    }
  }, [filterActive])

  const showFilterCount = filterActive && filterValue.length > 0

  return (
    <div
      role="status"
      aria-live="polite"
      className="file-browser__status"
      data-testid="file-browser-status"
    >
      <span className="file-browser__status-count">
        {showFilterCount
          ? `${itemCount} of ${itemsCopy(totalCount)}`
          : itemsCopy(totalCount)}
      </span>
      {filterActive && (
        <input
          ref={inputRef}
          type="search"
          role="searchbox"
          aria-label="Filter files in current directory"
          placeholder="Type to filter…"
          className="file-browser__status-filter"
          data-testid="file-browser-filter"
          value={filterValue}
          onChange={(e) => onFilterChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              e.preventDefault()
              onFilterDismiss()
            }
          }}
        />
      )}
      {showFilterCount && (
        <span className="file-browser__status-filter-info">
          Filtering: "{filterValue}" — {matchesCopy(itemCount)}
        </span>
      )}
    </div>
  )
}
