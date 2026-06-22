// Phase 122-03 Task 2 — RemoteJoinCodeModal.
//
// Paste-join-code modal that exchanges an 8-character code (format XXXX-XXXX)
// against a remote tailnet peer's /join/exchange endpoint (via the Wails RPC
// bridge) and caches the resulting cap in the local daemon's RemoteCapStore.
// Surfaces per-error-substring user copy locked by 122-03-PLAN.md Task 2
// behaviour table.
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
   * GAP-134-E: what the cap is being acquired for. Drives the title so a Phase 134
   * hub-modal cap request is not mislabelled "Files". Defaults to the generic title.
   * Phase 146 FIX-03: 'open-session' intent is for opening a remote session in the browser.
   */
  intent?: 'files' | 'hub-modal' | 'open-session'
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
  // WR-03 (GAP-146-A Plan 05): split the used/consumed single-use code case (D-11)
  // from the genuinely-wrong-digits case. 'not-found' means the code was already
  // consumed by the in-app connect (join codes are single-use — the exchange endpoint
  // returns 404 after first use). 'already used'/'already-used' covers the explicit
  // error text some exchange paths return. Both are distinct from a typo ('invalid').
  if (lower.includes('not-found') || lower.includes('already used') || lower.includes('already-used')) {
    return 'Code already used or expired — ask the owner for a fresh code or use the share link.'
  }
  if (lower.includes('invalid')) {
    return 'Code invalid. Double-check the 8-character code (XXXX-XXXX).'
  }
  return raw
}

export function RemoteJoinCodeModal({
  remoteSession,
  intent,
  onExchange,
  onClose,
}: RemoteJoinCodeModalProps): React.ReactElement {
  // GAP-134-E: only the file-browse flow is about "Files"; the Phase 134 hub-modal
  // flow opens the interactive/briefing terminal, so don't mislabel it.
  // Phase 146 FIX-03: 'open-session' has its own title.
  const title =
    intent === 'files' ? 'Join Remote Session — Files' :
    intent === 'open-session' ? 'Open Remote Session' :
    'Join Remote Session'
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
            {title}
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
          {intent === 'open-session' ? (
            <p className="remote-join-modal__body-text">
              To open <strong>{remoteSession.name}</strong> on{' '}
              <strong>{remoteSession.hostname}</strong> in your browser, ask
              the owner to share the session and send you the join code or share
              link. Paste the join code (format XXXX-XXXX) below.
            </p>
          ) : (
            <p className="remote-join-modal__body-text">
              Ask the owner of <strong>{remoteSession.name}</strong> on{' '}
              <strong>{remoteSession.hostname}</strong> for the 8-character join
              code (format XXXX-XXXX). (Owner generates it from the Daemon Manager
              panel.) Paste it below.
            </p>
          )}
          <input
            ref={inputRef}
            type="text"
            className="remote-join-modal__input"
            placeholder="XXXX-XXXX"
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
