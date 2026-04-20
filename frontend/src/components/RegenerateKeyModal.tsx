import React, { useEffect, useRef, useState } from 'react'

interface RegenerateKeyModalProps {
  isOpen: boolean
  onConfirm: () => Promise<void>
  onCancel: () => void
}

/**
 * Signing-key rotation confirmation modal (Phase 87 UI-SPEC Surface 2, D-16).
 *
 * Structurally mirrors QuitConfirmModal: reuses the .quit-modal* CSS classes.
 * "Invalidate All Links" is the destructive confirm — rotating the signing
 * key immediately invalidates every outstanding capability across every
 * session, which is the intended blast radius of the v3.1 panic button.
 */
function RegenerateKeyModal({ isOpen, onConfirm, onCancel }: RegenerateKeyModalProps): React.ReactElement | null {
  const [acting, setActing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cancelBtnRef = useRef<HTMLButtonElement>(null)

  // Close on Escape (matches QuitConfirmModal convention).
  useEffect(() => {
    if (!isOpen) return
    function handleKeyDown(e: KeyboardEvent): void {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onCancel])

  // Focus the cancel button on open (safer default for a destructive action).
  useEffect(() => {
    if (isOpen) cancelBtnRef.current?.focus()
  }, [isOpen])

  // Reset transient state when modal (re)opens or closes.
  useEffect(() => {
    if (!isOpen) {
      setError(null)
      setActing(false)
    }
  }, [isOpen])

  if (!isOpen) return null

  async function handleConfirm(): Promise<void> {
    setError(null)
    setActing(true)
    try {
      await onConfirm()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      setActing(false)
      return
    }
    setActing(false)
  }

  return (
    <div className="quit-modal-overlay" onClick={onCancel}>
      <div
        className="quit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="regen-key-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          <h2 className="quit-modal__title" id="regen-key-title">Regenerate Signing Key?</h2>
          <button className="quit-modal__close" aria-label="Close" onClick={onCancel}>&times;</button>
        </div>
        <div className="quit-modal__body">
          <p className="quit-modal__subtitle">
            This immediately invalidates ALL shared links across ALL sessions. Anyone currently using a shared terminal will be disconnected. You will need to re-share sessions to give access again.
          </p>
          {error && <p className="settings-panel__error">{error}</p>}
        </div>
        <div className="quit-modal__footer">
          <button
            className="quit-modal__btn--cancel"
            ref={cancelBtnRef}
            disabled={acting}
            onClick={onCancel}
          >
            Keep Links
          </button>
          <button
            className="quit-modal__btn--quit-all"
            disabled={acting}
            onClick={() => void handleConfirm()}
          >
            {acting ? 'Invalidating\u2026' : 'Invalidate All Links'}
          </button>
        </div>
      </div>
    </div>
  )
}

export { RegenerateKeyModal }
