/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { act } from 'react'
import { FileListPane } from '../FileListPane'
import type { FileEntry } from '../../../lib/filesApi'

let container: HTMLDivElement
let root: Root

beforeEach(() => {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
})

afterEach(() => {
  act(() => {
    root.unmount()
  })
  container.remove()
})

function render(node: React.ReactElement): void {
  act(() => {
    root.render(node)
  })
}

function ent(name: string, opts: Partial<FileEntry> = {}): FileEntry {
  return {
    name,
    size: 0,
    mtime: '',
    mode: 0,
    isDir: false,
    isSymlink: false,
    isBinary: false,
    ...opts,
  }
}

const defaultEntries: FileEntry[] = [
  ent('subdir', { isDir: true }),
  ent('apple.txt', { size: 100, mtime: '2026-05-18T10:00:00Z' }),
  ent('banana.md', { size: 0, mtime: '2026-05-19T10:00:00Z' }),
  ent('component.tsx', { size: 3072, mtime: '2026-05-20T10:00:00Z' }),
]

interface RenderOpts {
  entries?: FileEntry[]
  selectedName?: string | null
  sortKey?: 'name' | 'size' | 'modified'
  sortDir?: 'asc' | 'desc'
  filter?: string
  truncated?: boolean
  onSelect?: (n: string) => void
  onNavigateInto?: (n: string) => void
  onNavigateUp?: () => void
  onSortChange?: (k: 'name' | 'size' | 'modified') => void
  onFilterActivate?: () => void
}

function renderPane(opts: RenderOpts = {}): void {
  render(
    <FileListPane
      entries={opts.entries ?? defaultEntries}
      selectedName={opts.selectedName ?? null}
      sortKey={opts.sortKey ?? 'name'}
      sortDir={opts.sortDir ?? 'asc'}
      filter={opts.filter ?? ''}
      truncated={opts.truncated ?? false}
      isActive={true}
      onSelect={opts.onSelect ?? vi.fn()}
      onNavigateInto={opts.onNavigateInto ?? vi.fn()}
      onNavigateUp={opts.onNavigateUp ?? vi.fn()}
      onSortChange={opts.onSortChange ?? vi.fn()}
      onFilterActivate={opts.onFilterActivate ?? vi.fn()}
    />,
  )
}

function getListContainer(): HTMLElement {
  return container.querySelector('[data-testid="file-browser-list"]') as HTMLElement
}

function getRows(): HTMLElement[] {
  return Array.from(
    container.querySelectorAll('[role="option"]'),
  ) as HTMLElement[]
}

function pressKey(key: string): void {
  const listEl = getListContainer()
  act(() => {
    listEl.dispatchEvent(
      new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }),
    )
  })
}

describe('FileListPane — rendering', () => {
  it('renders one role=option per entry', () => {
    renderPane()
    expect(getRows().length).toBe(4)
  })

  it('selected entry has aria-selected=true; others false', () => {
    renderPane({ selectedName: 'apple.txt' })
    const rows = getRows()
    const selected = rows.find((r) => r.dataset.testid === 'file-browser-row-apple.txt')
    expect(selected!.getAttribute('aria-selected')).toBe('true')
    const other = rows.find((r) => r.dataset.testid === 'file-browser-row-subdir')
    expect(other!.getAttribute('aria-selected')).toBe('false')
  })

  it('renders directories before files (directories-sticky sort rule)', () => {
    renderPane()
    const rows = getRows()
    // subdir must come before any non-dir entry under name-asc default.
    expect(rows[0].dataset.testid).toBe('file-browser-row-subdir')
  })

  it('row size column shows "—" for directories AND files with size=0', () => {
    renderPane()
    const subdirRow = container.querySelector(
      '[data-testid="file-browser-row-subdir"]',
    )!
    const bananaRow = container.querySelector(
      '[data-testid="file-browser-row-banana.md"]',
    )!
    expect(subdirRow.querySelector('.file-browser__row-size')!.textContent).toBe('—')
    expect(bananaRow.querySelector('.file-browser__row-size')!.textContent).toBe('—')
  })

  it('aria-sort attribute reflects sortKey + sortDir', () => {
    renderPane({ sortKey: 'size', sortDir: 'desc' })
    const sizeCol = container.querySelector('[data-testid="file-browser-col-size"]')!
    const nameCol = container.querySelector('[data-testid="file-browser-col-name"]')!
    const modCol = container.querySelector('[data-testid="file-browser-col-modified"]')!
    expect(sizeCol.getAttribute('aria-sort')).toBe('descending')
    expect(nameCol.getAttribute('aria-sort')).toBe('none')
    expect(modCol.getAttribute('aria-sort')).toBe('none')
  })

  it('truncated=true renders the truncation banner', () => {
    renderPane({ truncated: true })
    expect(
      container.querySelector('[data-testid="file-browser-truncated"]'),
    ).not.toBeNull()
  })

  it('filter narrows the listbox to entries containing the filter (case-insensitive)', () => {
    renderPane({ filter: 'COMP' })
    const rows = getRows()
    expect(rows.length).toBe(1)
    expect(rows[0].dataset.testid).toBe('file-browser-row-component.tsx')
  })
})

describe('FileListPane — pointer interactions', () => {
  it('clicking a row fires onSelect with that name', () => {
    const onSelect = vi.fn()
    renderPane({ onSelect })
    const row = container.querySelector(
      '[data-testid="file-browser-row-apple.txt"]',
    ) as HTMLElement
    act(() => {
      row.click()
    })
    expect(onSelect).toHaveBeenCalledWith('apple.txt')
  })

  it('double-clicking a directory row fires onNavigateInto', () => {
    const onNavigateInto = vi.fn()
    renderPane({ onNavigateInto })
    const row = container.querySelector(
      '[data-testid="file-browser-row-subdir"]',
    ) as HTMLElement
    act(() => {
      row.dispatchEvent(
        new MouseEvent('dblclick', { bubbles: true, cancelable: true }),
      )
    })
    expect(onNavigateInto).toHaveBeenCalledWith('subdir')
  })

  it('clicking a column header fires onSortChange with its key', () => {
    const onSortChange = vi.fn()
    renderPane({ onSortChange })
    const sizeCol = container.querySelector(
      '[data-testid="file-browser-col-size"]',
    ) as HTMLElement
    act(() => {
      sizeCol.click()
    })
    expect(onSortChange).toHaveBeenCalledWith('size')
  })
})

describe('FileListPane — keyboard navigation (UI-12)', () => {
  it('ArrowDown from row 0 selects row 1', () => {
    const onSelect = vi.fn()
    renderPane({ onSelect, selectedName: 'subdir' })
    pressKey('ArrowDown')
    // Display order under name-asc: subdir, apple.txt, banana.md, component.tsx
    expect(onSelect).toHaveBeenCalledWith('apple.txt')
  })

  it('ArrowDown at last row stays at last row (no wrap)', () => {
    const onSelect = vi.fn()
    renderPane({ onSelect, selectedName: 'component.tsx' })
    pressKey('ArrowDown')
    // Clamped to last row → still component.tsx
    expect(onSelect).toHaveBeenCalledWith('component.tsx')
  })

  it('Home selects first row', () => {
    const onSelect = vi.fn()
    renderPane({ onSelect, selectedName: 'component.tsx' })
    pressKey('Home')
    expect(onSelect).toHaveBeenCalledWith('subdir')
  })

  it('End selects last row', () => {
    const onSelect = vi.fn()
    renderPane({ onSelect, selectedName: 'subdir' })
    pressKey('End')
    expect(onSelect).toHaveBeenCalledWith('component.tsx')
  })

  it('Enter on a selected directory row fires onNavigateInto', () => {
    const onNavigateInto = vi.fn()
    renderPane({ onNavigateInto, selectedName: 'subdir' })
    pressKey('Enter')
    expect(onNavigateInto).toHaveBeenCalledWith('subdir')
  })

  it('Backspace fires onNavigateUp', () => {
    const onNavigateUp = vi.fn()
    renderPane({ onNavigateUp })
    pressKey('Backspace')
    expect(onNavigateUp).toHaveBeenCalledTimes(1)
  })

  it('"/" key fires onFilterActivate AND preventDefault', () => {
    const onFilterActivate = vi.fn()
    renderPane({ onFilterActivate })
    const listEl = getListContainer()
    const evt = new KeyboardEvent('keydown', {
      key: '/',
      bubbles: true,
      cancelable: true,
    })
    act(() => {
      listEl.dispatchEvent(evt)
    })
    expect(onFilterActivate).toHaveBeenCalledTimes(1)
    expect(evt.defaultPrevented).toBe(true)
  })
})
