/**
 * UploadDropOverlay — Phase 125-05 Task 2 (EDIT-10).
 *
 * Full-container drag-and-drop overlay shown when canWrite and the user
 * drags files over the directory listing.
 *
 * Prompt copy is verbatim from UI-SPEC §Copywriting:
 *   "Drop files to upload here"
 *
 * Colorblind: uses ArrowUpTrayIcon glyph + text — no color-only signal.
 * Border: dashed #7aa2f7 (accent, UI-SPEC).
 */

import React from 'react'
import { ArrowUpTrayIcon } from '@heroicons/react/24/outline'

export interface UploadDropOverlayProps {
  /** Called with the dropped File array when drop event fires. */
  onDrop: (files: File[]) => void
  /** Called when the drag leaves the overlay area. */
  onDragLeave: () => void
}

export function UploadDropOverlay({
  onDrop,
  onDragLeave,
}: UploadDropOverlayProps): React.ReactElement {
  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    // Must preventDefault to allow drop.
    e.preventDefault()
    e.stopPropagation()
  }

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    const dt = e.dataTransfer
    if (!dt) return
    const files = Array.from(dt.files)
    if (files.length > 0) {
      onDrop(files)
    }
  }

  const handleDragLeave = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    onDragLeave()
  }

  return (
    <div
      data-testid="upload-drop-overlay"
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      onDragLeave={handleDragLeave}
      role="region"
      aria-label="Drop zone — drop files to upload"
      style={{
        position: 'absolute',
        inset: 0,
        zIndex: 100,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 12,
        background: 'rgba(26,27,38,0.88)',
        border: '2px dashed #7aa2f7',
        borderRadius: 4,
        pointerEvents: 'all',
      }}
    >
      <ArrowUpTrayIcon
        width={32}
        height={32}
        aria-hidden="true"
        style={{ color: '#7aa2f7' }}
      />
      <span style={{ fontSize: 14, color: '#c0caf5', fontWeight: 600 }}>
        Drop files to upload here
      </span>
    </div>
  )
}
