// Phase 125-04: FileRowActions cluster added — revealed on :hover/:focus-within.
// All action buttons (Edit/Rename/Move/Delete) gated on canWrite.
// Edit additionally requires !entry.isDir && !entry.isBinary (UI-SPEC §1).
// New optional props (canWrite, onEdit, onRename, onMove, onDelete) keep
// existing callers/tests compileable without changes.

import React from 'react'
import {
  FolderIcon,
  DocumentIcon,
  DocumentTextIcon,
  PhotoIcon,
  LinkIcon,
  ExclamationTriangleIcon,
  PencilSquareIcon,
  PencilIcon,
  ArrowsRightLeftIcon,
  TrashIcon,
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
  // ─── Optional write affordance props (Plan 04, EDIT-09/12) ───────────────
  /** True when the session has files.write perm. Gates all action buttons. */
  canWrite?: boolean
  /** Called when the Edit button is clicked (file, non-binary only). */
  onEdit?: () => void
  /** Called when the Rename button is clicked. */
  onRename?: () => void
  /** Called when the Move button is clicked. */
  onMove?: () => void
  /** Called when the Delete button is clicked. */
  onDelete?: () => void
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
  // Extract the calendar-date prefix (10 chars) cheaply. Phase 120 WR-04: any
  // string that does not match the RFC3339 calendar-date shape (YYYY-MM-DD…)
  // falls back to an em-dash rather than the raw input, so a malformed value
  // (e.g. "0001-01-01T00:00:00Z" for missing files, or a non-conforming
  // server response) cannot blow out the row's column width by dumping the
  // full string into the cell.
  if (mtime.length >= 10 && mtime[4] === '-' && mtime[7] === '-') {
    return mtime.slice(0, 10)
  }
  return '—'
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

/**
 * FileRowActions — cluster of write-affordance icon buttons.
 *
 * Revealed via CSS on `.file-browser__list-row:hover .file-browser__row-actions`
 * and `:focus-within`. All buttons gated on canWrite (caller must pass canWrite=true).
 *
 * Edit is additionally gated on !entry.isDir && !entry.isBinary (UI-SPEC §1 —
 * absence of the icon IS the colorblind-safe signal for binary/dir files).
 */
function FileRowActions({
  entry,
  onEdit,
  onRename,
  onMove,
  onDelete,
}: {
  entry: FileEntry
  onEdit?: () => void
  onRename?: () => void
  onMove?: () => void
  onDelete?: () => void
}): React.ReactElement {
  return (
    <span
      className="file-browser__row-actions"
      // Stop row-level click/dblclick from firing when the action buttons are clicked.
      onClick={(e) => e.stopPropagation()}
      onDoubleClick={(e) => e.stopPropagation()}
    >
      {/* Edit — files only, non-binary (UI-SPEC §1) */}
      {!entry.isDir && !entry.isBinary && onEdit && (
        <button
          type="button"
          className="file-browser__btn file-browser__btn--icon"
          aria-label={`Edit ${entry.name}`}
          title="Edit"
          tabIndex={-1}
          onClick={(e) => { e.stopPropagation(); onEdit() }}
        >
          <PencilSquareIcon width={14} height={14} aria-hidden="true" />
        </button>
      )}
      {/* Rename — all entries */}
      {onRename && (
        <button
          type="button"
          className="file-browser__btn file-browser__btn--icon"
          aria-label={`Rename ${entry.name}`}
          title="Rename"
          tabIndex={-1}
          onClick={(e) => { e.stopPropagation(); onRename() }}
        >
          <PencilIcon width={14} height={14} aria-hidden="true" />
        </button>
      )}
      {/* Move — all entries */}
      {onMove && (
        <button
          type="button"
          className="file-browser__btn file-browser__btn--icon"
          aria-label={`Move ${entry.name} to…`}
          title="Move to…"
          tabIndex={-1}
          onClick={(e) => { e.stopPropagation(); onMove() }}
        >
          <ArrowsRightLeftIcon width={14} height={14} aria-hidden="true" />
        </button>
      )}
      {/* Delete — all entries; destructive color (#f7768e) via CSS */}
      {onDelete && (
        <button
          type="button"
          className="file-browser__btn file-browser__btn--icon file-browser__btn--destructive"
          aria-label={`Delete ${entry.name}`}
          title="Delete"
          tabIndex={-1}
          onClick={(e) => { e.stopPropagation(); onDelete() }}
        >
          <TrashIcon width={14} height={14} aria-hidden="true" />
        </button>
      )}
    </span>
  )
}

export function FileRow({
  entry,
  isSelected,
  onClick,
  onDoubleClick,
  canWrite,
  onEdit,
  onRename,
  onMove,
  onDelete,
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
      {/* 4px slot aligning under the header's Name|Size divider. */}
      <span aria-hidden="true" />
      <span className="file-browser__row-size">{formatRowSize(entry)}</span>
      {/* 4px slot aligning under the header's Size|Modified divider. */}
      <span aria-hidden="true" />
      <span className="file-browser__row-mtime">{formatRowMtime(entry.mtime)}</span>
      {/* Phase 125-04: FileRowActions — write affordances gated on canWrite (EDIT-09/12) */}
      {canWrite && (
        <FileRowActions
          entry={entry}
          onEdit={onEdit}
          onRename={onRename}
          onMove={onMove}
          onDelete={onDelete}
        />
      )}
    </li>
  )
}
