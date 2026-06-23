import React, { useEffect, useRef, useState } from 'react'

interface RegenerateKeyModalProps {
  isOpen: boolean
  onConfirm: () => Promise<void>
  onCancel: () => void
  /**
   * Optional copy overrides so this confirm modal can be reused for actions
   * other than signing-key rotation (Phase 150: shell web-share warning
   * disable-confirm). All default to the signing-key copy, so existing call
   * sites that omit them are unchanged.
   */
  title?: string
  body?: string
  confirmLabel?: string
  /** Label shown while onConfirm is in flight (defaults to "Invalidating…"). */
  actingLabel?: string
  cancelLabel?: string
  titleId?: string
}

/**
 * Confirmation modal — originally the signing-key rotation prompt (Phase 87
 * UI-SPEC Surface 2, D-16), generalized in Phase 150 to accept copy overrides
 * so other destructive/guarded actions can reuse the same .quit-modal* shell.
 *
 * Structurally mirrors QuitConfirmModal: reuses the .quit-modal* CSS classes.
 * Defaults preserve the original copy: "Invalidate All Links" is the
 * destructive confirm — rotating the signing key immediately invalidates every
 * outstanding capability across every session (the v3.1 panic button).
 */
function RegenerateKeyModal({
  isOpen,
  onConfirm,
  onCancel,
  title = 'Regenerate Signing Key?',
  body = 'This immediately invalidates ALL shared links across ALL sessions. Anyone currently using a shared terminal will be disconnected. You will need to re-share sessions to give access again.',
  confirmLabel = 'Invalidate All Links',
  actingLabel = 'Invalidating…',
  cancelLabel = 'Keep Links',
  titleId = 'regen-key-title',
}: RegenerateKeyModalProps): React.ReactElement | null {
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
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          <h2 className="quit-modal__title" id={titleId}>{title}</h2>
          <button className="quit-modal__close" aria-label="Close" onClick={onCancel}>&times;</button>
        </div>
        <div className="quit-modal__body">
          <p className="quit-modal__subtitle">{body}</p>
          {error && <p className="settings-panel__error">{error}</p>}
        </div>
        <div className="quit-modal__footer">
          <button
            className="quit-modal__btn--cancel"
            ref={cancelBtnRef}
            disabled={acting}
            onClick={onCancel}
          >
            {cancelLabel}
          </button>
          <button
            className="quit-modal__btn--quit-all"
            disabled={acting}
            onClick={() => void handleConfirm()}
          >
            {acting ? actingLabel : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

export { RegenerateKeyModal }
