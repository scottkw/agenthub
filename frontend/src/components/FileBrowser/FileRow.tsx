import React from 'react'
import {
  FolderIcon,
  DocumentIcon,
  DocumentTextIcon,
  PhotoIcon,
  LinkIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline'
import type { FileEntry } from '../../lib/filesApi'
import { humanSize } from '../../lib/humanSize'

export interface FileRowProps {
  entry: FileEntry
  isSelected: boolean
  /** Single-click — parent treats as selection change. */
  onClick: () => void
  /** Double-click — open dir / focus preview for file. */
  onDoubleClick: () => void
}

/**
 * Select the Heroicon to render for a given file entry.
 *
 * UI-SPEC §Color colorblind contract: file type is signalled by an icon
 * glyph, never by color alone. Symlinks that look broken (mime would
 * carry the marker — not yet plumbed in v3.4) show a warning glyph;
 * for now LinkIcon covers all symlinks.
 */
function iconFor(entry: FileEntry): React.ComponentType<React.SVGProps<SVGSVGElement>> {
  if (entry.isDir) return FolderIcon
  if (entry.isSymlink) return LinkIcon
  if (entry.mime?.startsWith('image/')) return PhotoIcon
  if (entry.mime?.startsWith('text/')) return DocumentTextIcon
  if (entry.mime === 'text/markdown') return DocumentTextIcon
  // Unknown mime + name ends in .md → still mark as markdown for icon.
  if (entry.name.toLowerCase().endsWith('.md')) return DocumentTextIcon
  return DocumentIcon
}

/**
 * Format the size column for a row.
 * Per UI-SPEC: directories AND files where size==0 (List endpoint default
 * before lazy-stat fires) both render as an em-dash. Other files render
 * via humanSize.
 */
function formatRowSize(entry: FileEntry): string {
  if (entry.isDir) return '—'
  if (entry.size === 0) return '—'
  return humanSize(entry.size)
}

/**
 * Render the date portion of an RFC3339 mtime string (YYYY-MM-DD).
 * Empty mtime → em-dash placeholder.
 */
function formatRowMtime(mtime: string): string {
  if (!mtime) return '—'
  // Extract the calendar-date prefix (10 chars) cheaply. If the string is
  // shorter than 10 chars (malformed), fall back to the raw string.
  if (mtime.length >= 10 && mtime[4] === '-' && mtime[7] === '-') {
    return mtime.slice(0, 10)
  }
  return mtime
}

/**
 * Compose an aria-label for the row that conveys all the column data
 * audibly to a screen reader (per UI-SPEC §Accessibility Contract).
 *
 * Used as the SR-only announcement when the row receives focus or selection.
 */
function rowAriaLabel(entry: FileEntry): string {
  const kind = entry.isDir
    ? 'directory'
    : entry.isSymlink
      ? 'symlink'
      : 'file'
  const size = entry.isDir
    ? ''
    : `, ${entry.size === 0 ? 'size unknown' : humanSize(entry.size)}`
  const mtime = entry.mtime ? `, modified ${formatRowMtime(entry.mtime)}` : ''
  return `${entry.name}, ${kind}${size}${mtime}`
}

export function FileRow({
  entry,
  isSelected,
  onClick,
  onDoubleClick,
}: FileRowProps): React.ReactElement {
  const Icon = iconFor(entry)
  const className = isSelected
    ? 'file-browser__list-row file-browser__list-row--selected'
    : 'file-browser__list-row'
  return (
    <li
      role="option"
      aria-selected={isSelected}
      aria-label={rowAriaLabel(entry)}
      data-testid={`file-browser-row-${entry.name}`}
      className={className}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
    >
      <span className="file-browser__row-icon" aria-hidden="true">
        <Icon width={14} height={14} />
        {entry.isSymlink && entry.name.endsWith('!broken') && (
          <ExclamationTriangleIcon
            className="file-browser__row-icon-overlay"
            width={10}
            height={10}
          />
        )}
      </span>
      <span className="file-browser__row-name">{entry.name}</span>
      <span className="file-browser__row-size">{formatRowSize(entry)}</span>
      <span className="file-browser__row-mtime">{formatRowMtime(entry.mtime)}</span>
    </li>
  )
}
