// Phase 125-03 Task 2 — unsaved-changes navigation guard modal.
//
// Fires when the user tries to navigate away (file-switch, navigate-up,
// tab-close) while the editor buffer is dirty (EDIT-07).
//
// Pattern: QuitConfirmModal — overlay click-cancel + stopPropagation on the
// dialog, role="dialog" aria-modal, Escape-closes, safe-button default focus
// via ref+useEffect, acting guard to disable buttons during async ops.
//
// Copywriting verbatim from UI-SPEC §Unsaved-changes navigation guard (locked):
//   title:   "Unsaved changes"
//   body:    "You have unsaved changes. Save or discard?"
//   primary: "Save"
//   secondary: "Discard"
//   cancel (default focus): "Keep editing"
//
// Default focus on "Keep editing" — safe-button-first contract (not the
// destructive Discard). Discard silently drops changes, so it must NEVER
// be default-focused (colorblind/safety contract, PATTERNS §modals).
//
// NO beforeunload — Wails blocks it; this is pure React-level guarding (EDIT-07).

import React, { useEffect, useRef, useState } from 'react'

export interface UnsavedChangesModalProps {
  isOpen: boolean
  /** Called when the user clicks "Save" — the parent performs the save then proceeds. */
  onSave: () => void
  /** Called when the user explicitly discards changes and proceeds with navigation. */
  onDiscard: () => void
  /** Called when the user clicks "Keep editing" or Escape — stays in the editor. */
  onCancel: () => void
}

export function UnsavedChangesModal({
  isOpen,
  onSave,
  onDiscard,
  onCancel,
}: UnsavedChangesModalProps): React.ReactElement | null {
  const [acting, setActing] = useState(false)

  // Default focus on "Keep editing" (safe button — PATTERNS §modals)
  const keepEditingRef = useRef<HTMLButtonElement>(null)

  // Escape-closes via window.addEventListener (QuitConfirmModal pattern)
  useEffect(() => {
    if (!isOpen) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onCancel])

  // Focus "Keep editing" on open
  useEffect(() => {
    if (isOpen) keepEditingRef.current?.focus()
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
        aria-labelledby="unsaved-modal-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          {/* UI-SPEC verbatim title (locked EDIT-07) */}
          <h2 className="quit-modal__title" id="unsaved-modal-title">
            Unsaved changes
          </h2>
        </div>
        <div className="quit-modal__body">
          {/* UI-SPEC verbatim body (locked EDIT-07) */}
          <p className="quit-modal__subtitle">
            You have unsaved changes. Save or discard?
          </p>
        </div>
        <div className="quit-modal__footer">
          {/* "Keep editing" — SAFE default focus (QuitConfirmModal pattern) */}
          <button
            type="button"
            ref={keepEditingRef}
            className="quit-modal__btn--cancel"
            disabled={acting}
            onClick={onCancel}
          >
            Keep editing
          </button>

          {/* "Discard" — secondary; NOT default focus (destructive — silently drops changes) */}
          <button
            type="button"
            className="quit-modal__btn--quit-gui"
            disabled={acting}
            onClick={() => {
              setActing(true)
              onDiscard()
            }}
          >
            Discard
          </button>

          {/* "Save" — primary action */}
          <button
            type="button"
            className="quit-modal__btn--quit-all"
            disabled={acting}
            onClick={() => {
              setActing(true)
              onSave()
            }}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  )
}
