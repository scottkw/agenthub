import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { GetSessionTailLines } from '../../wailsjs/go/main/App'
import { RelayClient } from '../../lib/relayClient'

export interface HubBriefingModalProps {
  session: SessionInfo
  relayPort: number
  onClose: () => void
}

/**
 * HubBriefingModal shows the real terminal tail and a respond input for
 * sessions that need attention (waiting / errored). The Send flow constructs
 * a per-send RelayClient and delivers the response inside the onOpen callback
 * to guarantee the WebSocket is OPEN before sendInput fires (Pitfall 5 — race-safe).
 *
 * Security: maxLength={4096} bounds the single-send payload (ASVS V5).
 * Tail lines are ANSI-stripped by GetSessionTailLines on the Go side and
 * rendered as React text content (auto-escaped — no dangerouslySetInnerHTML).
 */
export function HubBriefingModal({
  session,
  relayPort,
  onClose,
}: HubBriefingModalProps): React.ReactElement {
  const [tailLines, setTailLines] = useState<string[] | null>(null) // null = loading
  const [responseText, setResponseText] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)
  const respondInputRef = useRef<HTMLTextAreaElement>(null)

  // Fetch the last 20 lines of terminal output on mount (one-shot)
  useEffect(() => {
    GetSessionTailLines(session.id, 20)
      .then((lines) => setTailLines(lines))
      .catch(() => setTailLines([]))
  }, [session.id])

  // Focus the respond textarea on mount so the user can type immediately
  useEffect(() => {
    respondInputRef.current?.focus()
  }, [])

  const sendDisabled = sending || responseText.trim() === ''

  const handleSend = useCallback(async () => {
    if (sending || responseText.trim() === '') return
    setSending(true)
    setSendError(null)
    try {
      await new Promise<void>((resolve, reject) => {
        const client = new RelayClient(relayPort, session.id, {
          onOutput: () => {}, // discard — sending only
          onOpen: () => {
            // Race-safe: sendInput fires only after the WebSocket is OPEN (Pitfall 5)
            client.sendInput(responseText + '\n')
            setTimeout(() => {
              client.close()
              resolve()
            }, 100)
          },
          onClose: () => {},
        })
        // Guard: reject after 5s if the WS never opens
        setTimeout(() => reject(new Error('timeout')), 5000)
      })
      onClose()
    } catch {
      setSendError('Failed to send. Close and try again.')
      setSending(false)
    }
  }, [relayPort, session.id, responseText, sending, onClose])

  return (
    <div className="hub-modal__body hub-modal__body--briefing">
      {/* Terminal tail display */}
      <div
        className={[
          'hub-modal__tail',
          tailLines === null ? 'hub-modal__tail--loading' : '',
          Array.isArray(tailLines) && tailLines.length === 0
            ? 'hub-modal__tail--empty'
            : '',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        {tailLines === null ? (
          <span>Loading…</span>
        ) : tailLines.length === 0 ? (
          <span>No recent output available.</span>
        ) : (
          <pre>{tailLines.join('\n')}</pre>
        )}
      </div>

      {/* Respond section */}
      <div className="hub-modal__respond">
        <span className="hub-modal__respond-label">RESPOND</span>
        <textarea
          ref={respondInputRef}
          className="hub-modal__respond-input"
          placeholder="Type a response…"
          value={responseText}
          onChange={(e) => setResponseText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
              e.preventDefault()
              void handleSend()
            }
          }}
          disabled={sending}
          maxLength={4096}
          rows={3}
        />
        {sendError !== null && (
          <p className="hub-modal__error-banner" role="alert">
            {sendError}
          </p>
        )}
        <div className="hub-modal__respond-footer">
          <button
            type="button"
            className="hub-modal__close-btn"
            onClick={onClose}
            disabled={sending}
          >
            Close
          </button>
          <button
            type="button"
            className="hub-modal__send-btn"
            onClick={() => void handleSend()}
            disabled={sendDisabled}
          >
            {sending ? 'Sending…' : 'Send Response'}
          </button>
        </div>
      </div>
    </div>
  )
}
