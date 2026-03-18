import React, { useEffect, useState } from 'react'
import { GetSessionQRCode } from '../wailsjs/go/main/App'

interface QRModalProps {
  sessionId: string
  sessionURL?: string
  onClose: () => void
}

/**
 * Modal overlay that fetches and displays the QR code for a session.
 * Close via overlay click or Escape key.
 */
export function QRModal({ sessionId, sessionURL, onClose }: QRModalProps): React.ReactElement {
  const [b64, setB64] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Fetch QR code on mount.
  useEffect(() => {
    GetSessionQRCode(sessionId)
      .then((code) => setB64(code))
      .catch((err) => {
        console.error('[QRModal] GetSessionQRCode failed:', err)
        setError('Failed to load QR code')
      })
  }, [sessionId])

  // Close on Escape key.
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  return (
    <div
      className="qr-modal-overlay"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label="QR code"
    >
      <div className="qr-modal" onClick={(e) => e.stopPropagation()}>
        {b64 === null && error === null && (
          <p className="qr-modal__loading">Loading QR code...</p>
        )}
        {error !== null && (
          <p className="qr-modal__error">{error}</p>
        )}
        {b64 !== null && (
          <img
            src={`data:image/png;base64,${b64}`}
            width={256}
            height={256}
            alt="QR code"
          />
        )}
        {sessionURL && (
          <p className="qr-modal__url">{sessionURL}</p>
        )}
        <button className="qr-modal__close" onClick={onClose}>
          Close
        </button>
      </div>
    </div>
  )
}
