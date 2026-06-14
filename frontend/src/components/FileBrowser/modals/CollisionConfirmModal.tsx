// Phase 125-04 Task 2 — 409 Collision confirm modal.
//
// Shown when a rename / create / mkdir operation returns HTTP 409 (fs.ErrExist →
// server maps to 409, write.go:244-245). The user can choose to Replace (re-issue
// the op with force semantics) or Cancel (safe default focus — T-125-12 locked).
//
// Copy strings VERBATIM from UI-SPEC §Copywriting Contract (EDIT-09/10 locked):
//   Title:  "File already exists"
//   Body:   `A file named "{name}" already exists. Replace it?`
//   Primary (destructive): "Replace"
//   Cancel (DEFAULT focus): "Cancel"
//
// Colorblind-safe: ExclamationTriangleIcon glyph + literal filename in body;
// color (amber border-left) is decoration only.
//
// QuitConfirmModal pattern: overlay click-cancel, Escape-closes, safe-default
// focus on Cancel, acting guard during async op.

import React, { useEffect, useRef, useState } from 'react'
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline'

export interface CollisionConfirmModalProps {
  isOpen: boolean
  /** The filename that already exists (shown in the body copy). */
  name: string
  /** Called when the user confirms Replace. */
  onReplace: () => Promise<void> | void
  /** Called when the user cancels (DEFAULT focus action — safe choice). */
  onCancel: () => void
}

export function CollisionConfirmModal({
  isOpen,
  name,
  onReplace,
  onCancel,
}: CollisionConfirmModalProps): React.ReactElement | null {
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

  // Focus "Cancel" (DEFAULT focus — EDIT-09/10 locked) on open.
  useEffect(() => {
    if (isOpen) cancelBtnRef.current?.focus()
  }, [isOpen])

  // Reset acting state when modal reopens.
  useEffect(() => {
    if (isOpen) setActing(false)
  }, [isOpen])

  if (!isOpen) return null

  async function handleReplace() {
    if (acting) return
    setActing(true)
    try {
      await onReplace()
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
        aria-labelledby="collision-confirm-title"
        style={{ borderLeft: '3px solid #e0af68' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          <h2 className="quit-modal__title" id="collision-confirm-title">
            <ExclamationTriangleIcon
              width={16}
              height={16}
              aria-hidden="true"
              style={{ display: 'inline', verticalAlign: 'text-bottom', marginRight: 6, color: '#e0af68' }}
            />
            {/* EDIT-09/10 locked title */}
            File already exists
          </h2>
          <button className="quit-modal__close" aria-label="Close" onClick={onCancel}>
            &times;
          </button>
        </div>
        <div className="quit-modal__body">
          {/* EDIT-09/10 locked body */}
          <p>{`A file named "${name}" already exists. Replace it?`}</p>
        </div>
        <div className="quit-modal__footer">
          {/* Cancel — DEFAULT focus (EDIT-09/10 locked safe-default invariant, T-125-12) */}
          <button
            ref={cancelBtnRef}
            type="button"
            className="file-browser__btn file-browser__btn--secondary"
            disabled={acting}
            onClick={onCancel}
          >
            Cancel
          </button>
          {/* Replace — destructive; NEVER default-focused */}
          <button
            type="button"
            className="file-browser__btn"
            style={{ background: '#f7768e', color: '#1a1b26', border: 'none' }}
            disabled={acting}
            onClick={() => { void handleReplace() }}
          >
            Replace
          </button>
        </div>
      </div>
    </div>
  )
}
