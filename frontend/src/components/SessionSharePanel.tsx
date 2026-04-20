import React, { useState } from 'react'
import { GetCapabilityQRCode } from '../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'

interface SessionSharePanelProps {
  sessionId: string
  readURL: string
  writeURL: string
  readCode: string
  writeCode: string
}

/**
 * Per-session share panel (Phase 87 UI-SPEC Surface 1).
 *
 * Renders two link rows (Read-Only Link + Full Access Link) with Copy/Open/QR
 * action buttons. The QR encodes the join-code exchange URL
 * (https://host/join?code=<code>) — NOT the raw capability token (D-09).
 *
 * Only one QR is visible per session at a time: activating one QR implicitly
 * hides the other.
 */
export function SessionSharePanel({
  sessionId,
  readURL,
  writeURL,
  readCode,
  writeCode,
}: SessionSharePanelProps): React.ReactElement {
  const [readCopied, setReadCopied] = useState(false)
  const [writeCopied, setWriteCopied] = useState(false)
  const [showReadQR, setShowReadQR] = useState(false)
  const [showWriteQR, setShowWriteQR] = useState(false)
  const [readQRb64, setReadQRb64] = useState<string | null>(null)
  const [writeQRb64, setWriteQRb64] = useState<string | null>(null)
  const [qrError, setQrError] = useState<string | null>(null)

  async function handleCopy(url: string, setter: (v: boolean) => void): Promise<void> {
    try {
      await ClipboardSetText(url)
    } catch {
      // ClipboardSetText failed — no user-visible action needed.
      return
    }
    setter(true)
    setTimeout(() => setter(false), 1500)
  }

  function joinURLFor(capURL: string, code: string): string {
    // QR encodes the join-code exchange URL (D-09), NOT the capability token.
    // Photographing/screenshotting a QR is worthless after 5 minutes or first
    // exchange — encoding the cap directly would defeat that mitigation.
    const u = new URL(capURL)
    return `${u.protocol}//${u.host}/join?code=${code}`
  }

  async function handleToggleQR(which: 'read' | 'write'): Promise<void> {
    if (which === 'read') {
      if (showReadQR) {
        setShowReadQR(false)
        return
      }
      // Only one QR visible per session at a time (UI-SPEC Surface 1).
      setShowWriteQR(false)
      setQrError(null)
      if (!readQRb64) {
        try {
          const b64 = await GetCapabilityQRCode(joinURLFor(readURL, readCode))
          setReadQRb64(b64)
        } catch {
          setQrError('QR unavailable \u2014 tap to retry')
          return
        }
      }
      setShowReadQR(true)
      return
    }

    if (showWriteQR) {
      setShowWriteQR(false)
      return
    }
    setShowReadQR(false)
    setQrError(null)
    if (!writeQRb64) {
      try {
        const b64 = await GetCapabilityQRCode(joinURLFor(writeURL, writeCode))
        setWriteQRb64(b64)
      } catch {
        setQrError('QR unavailable \u2014 tap to retry')
        return
      }
    }
    setShowWriteQR(true)
  }

  return (
    <div className="session-share-panel" data-session-id={sessionId}>
      <div className="session-share-panel__link-row">
        <span className="session-share-panel__label">Read-Only Link</span>
        <span className="session-share-panel__url" title={readURL}>{readURL}</span>
        <div className="session-share-panel__actions">
          <button
            className="daemon-panel__btn"
            onClick={() => void handleCopy(readURL, setReadCopied)}
            aria-label="Copy read-only link to clipboard"
          >
            {readCopied ? 'Copied!' : 'Copy'}
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => BrowserOpenURL(readURL)}
            aria-label="Open read-only link in browser"
          >
            Open
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => void handleToggleQR('read')}
            aria-label={showReadQR ? 'Hide read-only QR code' : 'Show read-only QR code'}
          >
            {showReadQR ? 'Hide QR' : 'QR'}
          </button>
        </div>
      </div>
      {showReadQR && readQRb64 && (
        <img
          className="session-share-panel__qr"
          src={`data:image/png;base64,${readQRb64}`}
          width={200}
          height={200}
          alt="QR code for read-only share link"
        />
      )}

      <div className="session-share-panel__link-row">
        <span className="session-share-panel__label">Full Access Link</span>
        <span className="session-share-panel__url" title={writeURL}>{writeURL}</span>
        <div className="session-share-panel__actions">
          <button
            className="daemon-panel__btn"
            onClick={() => void handleCopy(writeURL, setWriteCopied)}
            aria-label="Copy full-access link to clipboard"
          >
            {writeCopied ? 'Copied!' : 'Copy'}
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => BrowserOpenURL(writeURL)}
            aria-label="Open full-access link in browser"
          >
            Open
          </button>
          <button
            className="daemon-panel__btn"
            onClick={() => void handleToggleQR('write')}
            aria-label={showWriteQR ? 'Hide full-access QR code' : 'Show full-access QR code'}
          >
            {showWriteQR ? 'Hide QR' : 'QR'}
          </button>
        </div>
      </div>
      {showWriteQR && writeQRb64 && (
        <img
          className="session-share-panel__qr"
          src={`data:image/png;base64,${writeQRb64}`}
          width={200}
          height={200}
          alt="QR code for full-access share link"
        />
      )}

      {qrError && <p className="session-share-panel__error">{qrError}</p>}
    </div>
  )
}
