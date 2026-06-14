// Phase 125-03 Task 2 — HTTP 412 conflict modal.
//
// Opened when the save PUT returns 412 Precondition Failed — another process
// modified the file between the read and the write (EDIT-08).
//
// Pattern: QuitConfirmModal — overlay click-cancel + stopPropagation,
// role="dialog" aria-modal, Escape-closes, safe-button default focus,
// acting guard.
//
// Copywriting verbatim from UI-SPEC §Conflict modal (locked EDIT-08):
//   heading:  "This file was modified by another process"
//   body:     "Saving now will overwrite the other change. Choose how to
//              continue — your edits are preserved either way."
//   action 1 (destructive): "Force overwrite"
//   action 2 (primary):     "Save as new file"
//   action 3 (default focus, safe): "Discard my changes"
//
// Default focus on "Discard my changes" — safe button first (PATTERNS §modals).
// "Force overwrite" is NEVER default-focused (destroys concurrent changes).
//
// Buffer is NEVER silently cleared on any path (T-125-08 locked decision):
//   - Force overwrite: caller re-PUTs with If-Match="*" (server skip-check)
//   - Save as new file: caller PUTs a new path {basename}-copy{ext}, no If-Match
//   - Discard: caller re-fetches server content + replaces buffer

import React, { useEffect, useRef, useState } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

export interface ConflictModalProps {
  isOpen: boolean
  /** Called when the user chooses "Force overwrite" — re-PUT with If-Match="*". */
  onForceOverwrite: () => void
  /**
   * Called when the user chooses "Save as new file".
   * The parent prompts for/derives the new path ({basename}-copy{ext}) and PUTs
   * to it with no If-Match.
   */
  onSaveAsNew: () => void
  /**
   * Called when the user chooses "Discard my changes".
   * The parent re-fetches server content and replaces the editor buffer.
   */
  onDiscard: () => void
  /** Called when the user cancels (Escape / overlay click) — stays in the editor. */
  onCancel: () => void
}

export function ConflictModal({
  isOpen,
  onForceOverwrite,
  onSaveAsNew,
  onDiscard,
  onCancel,
}: ConflictModalProps): React.ReactElement | null {
  const [acting, setActing] = useState(false)

  // Default focus on "Discard my changes" (safe button — PATTERNS §modals)
  const discardRef = useRef<HTMLButtonElement>(null)

  // Escape-closes (QuitConfirmModal pattern)
  useEffect(() => {
    if (!isOpen) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onCancel])

  // Focus "Discard my changes" on open
  useEffect(() => {
    if (isOpen) discardRef.current?.focus()
  }, [isOpen])

  // Reset acting guard on reopen
  useEffect(() => {
    if (isOpen) setActing(false)
  }, [isOpen])

  if (!isOpen) return null

  return (
    <div className="quit-modal-overlay" onClick={onCancel}>
      <div
        className="quit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="conflict-modal-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          {/* ExclamationTriangleIcon + heading text — BOTH carry meaning (colorblind contract) */}
          <ExclamationTriangleIcon
            width={16}
            height={16}
            aria-hidden="true"
            className="quit-modal__conflict-icon"
          />
          {/* UI-SPEC verbatim heading (locked EDIT-08) */}
          <h2 className="quit-modal__title" id="conflict-modal-title">
            This file was modified by another process
          </h2>
        </div>
        <div className="quit-modal__body">
          {/* UI-SPEC verbatim body (locked EDIT-08) — "your edits are preserved either way" */}
          <p className="quit-modal__subtitle">
            Saving now will overwrite the other change. Choose how to continue —
            your edits are preserved either way.
          </p>
        </div>
        <div className="quit-modal__footer">
          {/* "Discard my changes" — SAFE default focus (buffer stays until user re-fetches) */}
          <button
            type="button"
            ref={discardRef}
            className="quit-modal__btn--cancel"
            disabled={acting}
            onClick={() => {
              setActing(true)
              onDiscard()
            }}
          >
            Discard my changes
          </button>

          {/* "Save as new file" — primary (no If-Match; new path) */}
          <button
            type="button"
            className="quit-modal__btn--quit-gui"
            disabled={acting}
            onClick={() => {
              setActing(true)
              onSaveAsNew()
            }}
          >
            Save as new file
          </button>

          {/* "Force overwrite" — destructive; re-PUT with If-Match="*" (server skip-check).
              NOT default-focused — safety contract (PATTERNS §modals). */}
          <button
            type="button"
            className="quit-modal__btn--quit-all quit-modal__btn--destructive"
            disabled={acting}
            onClick={() => {
              setActing(true)
              // Caller will re-PUT with If-Match: "*" to skip the precondition check.
              // The "*" value signals to the server: "I know a conflict exists; overwrite."
              onForceOverwrite()
            }}
          >
            Force overwrite
          </button>
        </div>
      </div>
    </div>
  )
}
