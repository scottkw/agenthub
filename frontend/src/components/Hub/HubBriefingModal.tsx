import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { GetSessionTailLines } from '../../wailsjs/go/main/App'
import { RelayClient } from '../../lib/relayClient'

export interface HubBriefingModalProps {
  session: SessionInfo
  relayPort: number
  onClose: () => void
  /** When true, the tail is derived from the WS scrollback snapshot replayed
   *  by the remote peer on connect (CR-02 fix — GetSessionTailLines is local-
   *  only and returns [] for remote session ids). The Send path routes through
   *  the daemon WS proxy. The cap is NOT passed here — the daemon looks it up
   *  by sessionID (T-134-07-01). */
  remote?: boolean
}

// Strip ANSI escape sequences to match engine.go GetSessionTailLines stripping.
// Covers CSI sequences (\x1b[...m etc.), OSC sequences (\x1b]...ST), and bare
// ESC sequences. This is client-side for the remote tail path only.
function stripAnsi(text: string): string {
  return text
    // OSC sequences: ESC ] ... (BEL or ST)
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
    // CSI sequences: ESC [ ... final byte (0x40-0x7e)
    .replace(/\x1b\[[0-9;?]*[A-Za-z]/g, '')
    // Bare ESC + single char
    .replace(/\x1b./g, '')
}

// Decode accumulated PTY output bytes to UTF-8 text, strip ANSI, and take the
// last N lines. Mirrors engine.go GetSessionTailLines without the Go side.
function extractTailLines(chunks: Uint8Array[], n: number): string[] {
  const total = chunks.reduce((sum, c) => sum + c.length, 0)
  const merged = new Uint8Array(total)
  let offset = 0
  for (const chunk of chunks) {
    merged.set(chunk, offset)
    offset += chunk.length
  }
  const text = new TextDecoder().decode(merged)
  const stripped = stripAnsi(text)
  const lines = stripped.split('\n').filter((l) => l.trim() !== '')
  return lines.slice(-n)
}

/**
 * HubBriefingModal shows the real terminal tail and a respond input for
 * sessions that need attention (waiting / errored). The Send flow constructs
 * a per-send RelayClient and delivers the response inside the onOpen callback
 * to guarantee the WebSocket is OPEN before sendInput fires (Pitfall 5 — race-safe).
 *
 * Security: maxLength={4096} bounds the single-send payload (ASVS V5).
 * Local tail lines are ANSI-stripped by GetSessionTailLines on the Go side and
 * rendered as React text content (auto-escaped — no dangerouslySetInnerHTML).
 * Remote tail (CR-02): sourced from the WS scrollback snapshot the peer replays
 * on connect; ANSI-stripped client-side; rendered as React text content.
 *
 * CR-03 fix: clientRef + settled flag + clearTimeout + useEffect unmount cleanup
 * ensure the WS and its ping interval are always torn down, and that abandoned
 * text is never written to the PTY after the 5s timeout.
 */
export function HubBriefingModal({
  session,
  relayPort,
  onClose,
  remote,
}: HubBriefingModalProps): React.ReactElement {
  const [tailLines, setTailLines] = useState<string[] | null>(null) // null = loading
  const [responseText, setResponseText] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)
  const respondInputRef = useRef<HTMLTextAreaElement>(null)

  // CR-03: track the in-flight RelayClient so the unmount cleanup can close it.
  const clientRef = useRef<RelayClient | null>(null)

  // Fetch the last 20 lines of terminal output on mount (one-shot).
  // CR-02: remote sessions read the scrollback snapshot replayed on the proxied WS
  // instead of calling GetSessionTailLines (which is local-only and returns [] for
  // remote ids — engine.go:550). A 3s timeout guards against stalled connections.
  useEffect(() => {
    if (remote) {
      // Remote path: open a short-lived proxied RelayClient, accumulate MsgOutput
      // payload bytes from the scrollback snapshot, then close.
      const chunks: Uint8Array[] = []
      let tailClient: RelayClient | null = null
      let resolved = false

      const finish = () => {
        if (resolved) return
        resolved = true
        tailClient?.close()
        tailClient = null
        setTailLines(extractTailLines(chunks, 20))
      }

      const timeoutId = setTimeout(finish, 3000)

      tailClient = new RelayClient(
        relayPort,
        session.id,
        {
          onOutput: (data) => {
            // Accumulate all MsgOutput bytes; snapshot frames come first,
            // then live PTY — finish() called on open-open (after first
            // frame) ensures we capture the snapshot before live output.
            chunks.push(data)
          },
          onOpen: () => {
            // Give the peer time to replay the full scrollback snapshot
            // (the peer sends it synchronously on subscribe, but there may
            // be multiple frames). Finish after 500ms once connected so we
            // collect the snapshot without staying connected for live PTY.
            setTimeout(finish, 500)
          },
          onClose: () => {
            clearTimeout(timeoutId)
            finish()
          },
        },
        { remote: true },
      )
    } else {
      // Local path: unchanged — GetSessionTailLines is fast, synchronous on Go side.
      GetSessionTailLines(session.id, 20)
        .then((lines) => setTailLines(lines))
        .catch(() => setTailLines([]))
    }
  }, [session.id, relayPort, remote])

  // Focus the respond textarea on mount so the user can type immediately
  useEffect(() => {
    respondInputRef.current?.focus()
  }, [])

  // CR-03: unmount cleanup — close any in-flight RelayClient (and its 30s ping
  // interval) if the modal is dismissed while a send is in flight.
  useEffect(() => () => { clientRef.current?.close() }, [])

  const sendDisabled = sending || responseText.trim() === ''

  const handleSend = useCallback(async () => {
    if (sending || responseText.trim() === '') return
    setSending(true)
    setSendError(null)
    try {
      await new Promise<void>((resolve, reject) => {
        // CR-03: settled flag prevents late onOpen sends after the 5s reject.
        let settled = false

        // CR-03: timer handle so we can clearTimeout on the happy path.
        const timer = setTimeout(() => {
          if (!settled) {
            settled = true
            // CR-03: close the WS on the reject/timeout path (leak fix).
            clientRef.current?.close()
            clientRef.current = null
            reject(new Error('timeout'))
          }
        }, 5000)

        const client = new RelayClient(
          relayPort,
          session.id,
          {
            onOutput: () => {}, // discard — sending only
            onOpen: () => {
              // CR-03: if settled (timeout already fired), tear down and bail.
              // The user abandoned; do NOT write text to the PTY post-timeout.
              if (settled) {
                client.close()
                return
              }
              // Race-safe: sendInput fires only after the WebSocket is OPEN.
              client.sendInput(responseText + '\n')
              setTimeout(() => {
                settled = true
                clearTimeout(timer)   // CR-03: clear the 5s reject timer
                client.close()
                clientRef.current = null
                resolve()
              }, 100)
            },
            onClose: () => {},
          },
          remote ? { remote: true } : undefined,
        )
        // CR-03: store the client so unmount cleanup can close it.
        clientRef.current = client
      })
      onClose()
    } catch {
      setSendError('Failed to send. Close and try again.')
      setSending(false)
    }
  }, [relayPort, session.id, responseText, sending, onClose, remote])

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
