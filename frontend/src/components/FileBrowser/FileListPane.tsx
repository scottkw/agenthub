import React, { useMemo } from 'react'
import { ChevronUpIcon, ChevronDownIcon } from '@heroicons/react/24/outline'
import type { FileEntry } from '../../lib/filesApi'
import type { SortKey, SortDir } from '../../lib/filesTypes'
import { sortEntries } from './sortEntries'
import { FileRow } from './FileRow'

export interface FileListPaneProps {
  entries: FileEntry[]
  /** Currently-selected entry name, or null if nothing selected. */
  selectedName: string | null
  sortKey: SortKey
  sortDir: SortDir
  /** Substring filter — empty string disables filtering. */
  filter: string
  /** Whether the daemon truncated the listing (banner shown if true). */
  truncated: boolean
  /** Whether the parent tab is currently active (gates '/' filter activation). */
  isActive: boolean
  /** Single-click row — parent treats as selection change. */
  onSelect: (name: string) => void
  /** Double-click dir OR Enter on dir-selected row. */
  onNavigateInto: (name: string) => void
  /** Backspace — parent handles root-no-op. */
  onNavigateUp: () => void
  /** Click on a column header — parent recomputes sortKey/sortDir. */
  onSortChange: (key: SortKey) => void
  /** '/' key while list has focus — parent reveals filter input. */
  onFilterActivate: () => void
}

// Page size for PgUp/PgDn keys. Static value — UI-SPEC says viewport÷row-height
// but a container ref isn't reliable in tests; 10 is the canonical fallback.
const PAGE_SIZE = 10

/**
 * Map a SortKey to its aria-sort attribute value given the active sort state.
 */
function ariaSortFor(
  col: SortKey,
  activeKey: SortKey,
  activeDir: SortDir,
): 'ascending' | 'descending' | 'none' {
  if (col !== activeKey) return 'none'
  return activeDir === 'asc' ? 'ascending' : 'descending'
}

interface ColumnHeaderProps {
  col: SortKey
  label: string
  activeKey: SortKey
  activeDir: SortDir
  onClick: () => void
}

function ColumnHeader({
  col,
  label,
  activeKey,
  activeDir,
  onClick,
}: ColumnHeaderProps): React.ReactElement {
  const isActive = col === activeKey
  const aria = ariaSortFor(col, activeKey, activeDir)
  return (
    <button
      type="button"
      role="columnheader"
      aria-sort={aria}
      className={`file-browser__col file-browser__col--${col}${isActive ? ' file-browser__col--active' : ''}`}
      data-testid={`file-browser-col-${col}`}
      onClick={onClick}
    >
      <span>{label}</span>
      {isActive && activeDir === 'asc' && (
        <ChevronUpIcon
          className="file-browser__col-chevron"
          width={10}
          height={10}
          aria-hidden="true"
        />
      )}
      {isActive && activeDir === 'desc' && (
        <ChevronDownIcon
          className="file-browser__col-chevron"
          width={10}
          height={10}
          aria-hidden="true"
        />
      )}
    </button>
  )
}

export function FileListPane({
  entries,
  selectedName,
  sortKey,
  sortDir,
  filter,
  truncated,
  isActive: _isActive,
  onSelect,
  onNavigateInto,
  onNavigateUp,
  onSortChange,
  onFilterActivate,
}: FileListPaneProps): React.ReactElement {
  // Compute the display list: filter, then sort.
  const displayEntries: FileEntry[] = useMemo(() => {
    const filtered = filter.length > 0
      ? entries.filter((e) =>
          e.name.toLowerCase().includes(filter.toLowerCase()),
        )
      : entries
    return sortEntries(filtered, sortKey, sortDir)
  }, [entries, filter, sortKey, sortDir])

  const selectedIndex = useMemo(() => {
    if (selectedName === null) return -1
    return displayEntries.findIndex((e) => e.name === selectedName)
  }, [displayEntries, selectedName])

  function selectAt(index: number): void {
    const clamped = Math.max(0, Math.min(displayEntries.length - 1, index))
    const next = displayEntries[clamped]
    if (next) onSelect(next.name)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLDivElement>): void {
    // Determine effective selection index — fall back to 0 if nothing selected.
    const base = selectedIndex < 0 ? 0 : selectedIndex
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        selectAt(base + 1)
        return
      case 'ArrowUp':
        e.preventDefault()
        selectAt(base - 1)
        return
      case 'Home':
        e.preventDefault()
        selectAt(0)
        return
      case 'End':
        e.preventDefault()
        selectAt(displayEntries.length - 1)
        return
      case 'PageDown':
        e.preventDefault()
        selectAt(base + PAGE_SIZE)
        return
      case 'PageUp':
        e.preventDefault()
        selectAt(base - PAGE_SIZE)
        return
      case 'Enter': {
        e.preventDefault()
        const selected = selectedIndex >= 0 ? displayEntries[selectedIndex] : undefined
        if (selected && selected.isDir) {
          onNavigateInto(selected.name)
        }
        // For files: Plan 04 owns preview-focus orchestration via onSelect;
        // here we no-op so the parent can wire focus management.
        return
      }
      case 'Backspace':
        e.preventDefault()
        onNavigateUp()
        return
      case '/':
        e.preventDefault()
        onFilterActivate()
        return
      default:
        // Tab + other keys fall through to native handling.
        return
    }
  }

  const filterActiveCount = filter.length > 0 ? displayEntries.length : entries.length
  const isFilterEmpty = filter.length > 0 && displayEntries.length === 0

  return (
    <div
      className="file-browser__list"
      data-testid="file-browser-list"
      tabIndex={0}
      onKeyDown={handleKeyDown}
    >
      {truncated && (
        <div
          className="file-browser__truncated-banner"
          data-testid="file-browser-truncated"
          role="status"
        >
          <strong>Showing first 10,000 entries.</strong>
          <span> Use the terminal to inspect deeper directories.</span>
        </div>
      )}
      <div role="row" className="file-browser__list-header">
        <ColumnHeader
          col="name"
          label="NAME"
          activeKey={sortKey}
          activeDir={sortDir}
          onClick={() => onSortChange('name')}
        />
        <ColumnHeader
          col="size"
          label="SIZE"
          activeKey={sortKey}
          activeDir={sortDir}
          onClick={() => onSortChange('size')}
        />
        <ColumnHeader
          col="modified"
          label="MODIFIED"
          activeKey={sortKey}
          activeDir={sortDir}
          onClick={() => onSortChange('modified')}
        />
      </div>
      <ul
        role="listbox"
        aria-label="Directory contents"
        aria-multiselectable="false"
        aria-busy={false}
        className="file-browser__list-scroll"
        data-testid="file-browser-list-scroll"
        data-filter-count={filterActiveCount}
      >
        {isFilterEmpty ? (
          <li
            className="file-browser__list-empty"
            data-testid="file-browser-no-match"
          >
            No files match "{filter}"
          </li>
        ) : (
          displayEntries.map((entry) => (
            <FileRow
              key={entry.name}
              entry={entry}
              isSelected={entry.name === selectedName}
              onClick={() => onSelect(entry.name)}
              onDoubleClick={() => {
                if (entry.isDir) onNavigateInto(entry.name)
                else onSelect(entry.name)
              }}
            />
          ))
        )}
      </ul>
    </div>
  )
}
