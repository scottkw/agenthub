// Phase 122-03 Task 2 — RemoteJoinCodeModal.
//
// Paste-join-code modal that exchanges a 5-character code against a remote
// tailnet peer's /join/exchange endpoint (via the Wails RPC bridge) and
// caches the resulting cap in the local daemon's RemoteCapStore. Surfaces
// per-error-substring user copy locked by 122-03-PLAN.md Task 2 behaviour
// table.
//
// Security notes (122-03 threat-model):
//   - T-122-03-03: the join code is held in React state only; never written
//     to URL, history, or localStorage. Cleared on close.
//   - T-122-03-05: session name + hostname rendered via React text content
//     (auto-escaped). No raw-HTML injection paths anywhere in this component.

import React, { useCallback, useEffect, useRef, useState } from 'react'

export interface RemoteJoinCodeModalProps {
  remoteSession: { id: string; name: string; hostname: string }
  /**
   * Exchange the entered code for a cap (typically: ExchangeJoinCodeAtURL →
   * RegisterRemoteCap two-step in App.tsx). Resolves on success; rejects with
   * an Error whose message contains one of:
   *   - 'expired'      → code TTL elapsed
   *   - 'invalid' / 'not-found' → wrong code
   *   - 'session-gone' → web-share toggled off after code was issued
   * Anything else surfaces as a raw error string (defensive fallback).
   */
  onExchange: (code: string) => Promise<void>
  onClose: () => void
}

function mapErrorMessage(raw: string): string {
  const lower = raw.toLowerCase()
  if (lower.includes('session-gone')) {
    return 'Remote session is no longer web-shared.'
  }
  if (lower.includes('expired')) {
    return 'Code expired. Ask the owner to generate a new code.'
  }
  if (lower.includes('invalid') || lower.includes('not-found')) {
    return 'Code invalid. Double-check the 5-character code.'
  }
  return raw
}

export function RemoteJoinCodeModal({
  remoteSession,
  onExchange,
  onClose,
}: RemoteJoinCodeModalProps): React.ReactElement {
  const [code, setCode] = useState<string>('')
  const [pending, setPending] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Focus the input on mount.
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  // Escape closes the modal (mirrors QuitConfirmModal).
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent): void {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const submitDisabled = pending || code.trim().length === 0

  const handleSubmit = useCallback(async () => {
    if (submitDisabled) return
    setPending(true)
    setError(null)
    try {
      await onExchange(code.trim())
      // Successful exchange — dismiss the modal.
      onClose()
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      setError(mapErrorMessage(msg))
      setPending(false)
    }
  }, [code, onClose, onExchange, submitDisabled])

  const titleId = 'remote-join-code-modal-title'

  return (
    <div
      className="modal-overlay remote-join-modal-overlay"
      onClick={onClose}
      data-testid="remote-join-modal-overlay"
    >
      <div
        className="modal-dialog remote-join-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="remote-join-modal__header">
          <h2 className="remote-join-modal__title" id={titleId}>
            Join Remote Session — Files
          </h2>
          <button
            type="button"
            className="remote-join-modal__close"
            aria-label="Close"
            onClick={onClose}
            disabled={pending}
          >
            &times;
          </button>
        </div>
        <div className="remote-join-modal__body">
          <p className="remote-join-modal__body-text">
            Ask the owner of <strong>{remoteSession.name}</strong> on{' '}
            <strong>{remoteSession.hostname}</strong> for the 5-character join
            code. (Owner generates it from the Daemon Manager panel.) Paste it
            below.
          </p>
          <input
            ref={inputRef}
            type="text"
            className="remote-join-modal__input"
            placeholder="ABCDE"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                void handleSubmit()
              }
            }}
            disabled={pending}
            aria-label="Join code"
            autoComplete="off"
            spellCheck={false}
          />
          {error !== null && (
            <p
              className="remote-join-modal__error"
              role="alert"
              data-testid="remote-join-modal-error"
            >
              {error}
            </p>
          )}
        </div>
        <div className="remote-join-modal__footer">
          <button
            type="button"
            className="remote-join-modal__btn remote-join-modal__btn--cancel"
            onClick={onClose}
            disabled={pending}
          >
            Cancel
          </button>
          <button
            type="button"
            className="remote-join-modal__btn remote-join-modal__btn--submit"
            onClick={() => void handleSubmit()}
            disabled={submitDisabled}
          >
            {pending ? 'Joining...' : 'Join'}
          </button>
        </div>
      </div>
    </div>
  )
}
