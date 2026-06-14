// Phase 125-04 Task 2 — Delete confirm modal.
//
// Two variants, selected by `isDir`:
//   - File variant: title "Delete file?" body `Delete "{name}"? This cannot be undone.`
//   - Directory variant: title "Delete folder?" body
//     `Delete "{name}" and all {N} files inside? This cannot be undone.`
//
// Destructive Delete button NEVER holds default focus (colorblind/safety contract,
// T-125-12). Cancel holds default focus (QuitConfirmModal pattern).
//
// The {N} file count is computed by a client-side listFiles walk in FileBrowserTab
// BEFORE this modal is opened — the count is passed as `fileCount` prop.
//
// Colorblind-safe: TrashIcon glyph + literal verb "Delete" in the button label;
// color (#f7768e) is decoration only.
//
// All copy strings verbatim from UI-SPEC §Copywriting Contract (EDIT-09 locked).

import React, { useEffect, useRef, useState } from 'react'
import { TrashIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'

export interface DeleteConfirmModalProps {
  isOpen: boolean
  /** Name of the file or directory being deleted. */
  name: string
  /** True when deleting a directory (recursive delete). */
  isDir: boolean
  /**
   * Number of files inside the directory (recursive count).
   * Only used when isDir=true. Computed by a client-side listFiles walk.
   */
  fileCount?: number
  /** Called when the user confirms delete. */
  onConfirm: () => Promise<void> | void
  /** Called when the user cancels (default focus action — safe choice). */
  onCancel: () => void
}

export function DeleteConfirmModal({
  isOpen,
  name,
  isDir,
  fileCount = 0,
  onConfirm,
  onCancel,
}: DeleteConfirmModalProps): React.ReactElement | null {
  const [acting, setActing] = useState(false)
  const cancelBtnRef = useRef<HTMLButtonElement>(null)

  // Close on Escape — safe/cancel action.
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  // Focus "Cancel" (safe default) on open.
  useEffect(() => {
    if (isOpen) cancelBtnRef.current?.focus()
  }, [isOpen])

  // Reset acting state when modal reopens.
  useEffect(() => {
    if (isOpen) setActing(false)
  }, [isOpen])

  if (!isOpen) return null

  const title = isDir ? 'Delete folder?' : 'Delete file?'
  const body = isDir
    // EDIT-09 locked string: "and all {N} files inside"
    ? `Delete "${name}" and all ${fileCount} files inside? This cannot be undone.`
    : `Delete "${name}"? This cannot be undone.`

  async function handleConfirm() {
    if (acting) return
    setActing(true)
    try {
      await onConfirm()
    } finally {
      setActing(false)
    }
  }

  return (
    <div className="quit-modal-overlay" onClick={onCancel}>
      <div
        className="quit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-confirm-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          <h2 className="quit-modal__title" id="delete-confirm-title">
            {isDir ? (
              <ExclamationTriangleIcon
                width={16}
                height={16}
                aria-hidden="true"
                style={{ display: 'inline', verticalAlign: 'text-bottom', marginRight: 6, color: '#f7768e' }}
              />
            ) : (
              <TrashIcon
                width={16}
                height={16}
                aria-hidden="true"
                style={{ display: 'inline', verticalAlign: 'text-bottom', marginRight: 6 }}
              />
            )}
            {title}
          </h2>
          <button className="quit-modal__close" aria-label="Close" onClick={onCancel}>
            &times;
          </button>
        </div>
        <div className="quit-modal__body">
          <p>{body}</p>
        </div>
        <div className="quit-modal__footer">
          {/* Cancel — DEFAULT focus (safe choice — T-125-12) */}
          <button
            ref={cancelBtnRef}
            type="button"
            className="file-browser__btn file-browser__btn--secondary"
            disabled={acting}
            onClick={onCancel}
          >
            Cancel
          </button>
          {/* Delete — destructive; NEVER default-focused */}
          <button
            type="button"
            className="file-browser__btn"
            style={{ background: '#f7768e', color: '#1a1b26', border: 'none' }}
            disabled={acting}
            aria-label={`Delete ${name}`}
            onClick={() => { void handleConfirm() }}
          >
            <TrashIcon width={14} height={14} aria-hidden="true" style={{ verticalAlign: 'text-bottom', marginRight: 4 }} />
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}
