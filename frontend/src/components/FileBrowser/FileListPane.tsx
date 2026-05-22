import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { ChevronUpIcon, ChevronDownIcon } from '@heroicons/react/24/outline'
import type { FileEntry } from '../../lib/filesApi'
import type { SortKey, SortDir } from '../../lib/filesTypes'
import { sortEntries } from './sortEntries'
import { FileRow } from './FileRow'

// Phase 120 UAT-1: persisted column widths for Size + Modified.
// Name absorbs remaining space via `1fr`.
const COL_WIDTH_STORAGE_KEY = 'agenthub.fileBrowser.colWidths.v1'
const SIZE_COL_DEFAULT = 80
const MTIME_COL_DEFAULT = 110
const COL_MIN_PX = 50
const COL_MAX_PX = 400

interface PersistedColWidths {
  size: number
  mtime: number
}

function clampColPx(px: number): number {
  if (!Number.isFinite(px)) return SIZE_COL_DEFAULT
  return Math.max(COL_MIN_PX, Math.min(COL_MAX_PX, Math.round(px)))
}

function loadPersistedColWidths(): PersistedColWidths {
  if (typeof window === 'undefined') return { size: SIZE_COL_DEFAULT, mtime: MTIME_COL_DEFAULT }
  try {
    const raw = window.localStorage?.getItem(COL_WIDTH_STORAGE_KEY)
    if (!raw) return { size: SIZE_COL_DEFAULT, mtime: MTIME_COL_DEFAULT }
    const parsed = JSON.parse(raw) as Partial<PersistedColWidths>
    return {
      size: clampColPx(parsed.size ?? SIZE_COL_DEFAULT),
      mtime: clampColPx(parsed.mtime ?? MTIME_COL_DEFAULT),
    }
  } catch {
    return { size: SIZE_COL_DEFAULT, mtime: MTIME_COL_DEFAULT }
  }
}

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

  // Phase 120 UAT-1: column widths (Size + Modified) — Name absorbs the rest.
  const [colWidths, setColWidths] = useState<PersistedColWidths>(loadPersistedColWidths)
  useEffect(() => {
    if (typeof window === 'undefined') return
    try {
      window.localStorage?.setItem(COL_WIDTH_STORAGE_KEY, JSON.stringify(colWidths))
    } catch {
      // localStorage may be unavailable (private mode, quota); ignore.
    }
  }, [colWidths])

  const startColDrag = useCallback(
    (target: 'size' | 'mtime', e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      const startX = e.clientX
      const startWidth = target === 'size' ? colWidths.size : colWidths.mtime
      const prevCursor = document.body.style.cursor
      const prevSelect = document.body.style.userSelect
      document.body.style.cursor = 'col-resize'
      document.body.style.userSelect = 'none'
      const onMove = (mv: MouseEvent) => {
        const delta = mv.clientX - startX
        // Drag east (+delta) = right-side boundary moves east = THIS column
        // gets smaller (the column gives space to its left neighbor).
        const next = clampColPx(startWidth - delta)
        setColWidths((prev) =>
          target === 'size'
            ? { ...prev, size: next }
            : { ...prev, mtime: next },
        )
      }
      const onUp = () => {
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseup', onUp)
        document.body.style.cursor = prevCursor
        document.body.style.userSelect = prevSelect
      }
      window.addEventListener('mousemove', onMove)
      window.addEventListener('mouseup', onUp)
    },
    [colWidths.size, colWidths.mtime],
  )

  // CSS custom properties so .file-browser__list-header and
  // .file-browser__list-row share the same column widths (grid-template-columns
  // uses var(--fb-size-col, 80px) + var(--fb-mtime-col, 110px)).
  const gridStyle = {
    '--fb-size-col': `${colWidths.size}px`,
    '--fb-mtime-col': `${colWidths.mtime}px`,
  } as React.CSSProperties

  return (
    <div
      className="file-browser__list"
      data-testid="file-browser-list"
      tabIndex={0}
      onKeyDown={handleKeyDown}
      style={gridStyle}
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
        <div
          className="file-browser__col-divider"
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize size column"
          data-testid="file-browser-col-divider-size"
          onMouseDown={(e) => startColDrag('size', e)}
        />
        <ColumnHeader
          col="size"
          label="SIZE"
          activeKey={sortKey}
          activeDir={sortDir}
          onClick={() => onSortChange('size')}
        />
        <div
          className="file-browser__col-divider"
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize modified column"
          data-testid="file-browser-col-divider-mtime"
          onMouseDown={(e) => startColDrag('mtime', e)}
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
