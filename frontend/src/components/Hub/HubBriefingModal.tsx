import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import { GetSessionStyledTailLines } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { Terminal } from '@xterm/xterm'
import { SerializeAddon } from '@xterm/addon-serialize'
import { RelayClient } from '../../lib/relayClient'
import { daemon } from '../../wailsjs/go/models'
import { resolveColor } from '../../lib/vtColor'

type StyledSpan = daemon.StyledSpan

export interface HubBriefingModalProps {
  session: SessionInfo
  relayPort: number
  onClose: () => void
  /** Phase 139 / CARD-05: active xterm ITheme for color resolution in the local tail. */
  theme: ITheme
  /** When true, the tail is derived from the WS scrollback snapshot replayed
   *  by the remote peer on connect (CR-02 fix — GetSessionStyledTailLines is local-
   *  only and returns [] for remote session ids). The Send path routes through
   *  the daemon WS proxy. The cap is NOT passed here — the daemon looks it up
   *  by sessionID (T-134-07-01). */
  remote?: boolean
}

/**
 * HubBriefingModal shows the real terminal tail and a respond input for
 * sessions that need attention (waiting / errored). The Send flow constructs
 * a per-send RelayClient and delivers the response inside the onOpen callback
 * to guarantee the WebSocket is OPEN before sendInput fires (Pitfall 5 — race-safe).
 *
 * Security: maxLength={4096} bounds the single-send payload (ASVS V5).
 * Local tail: StyledSpan[][] rendered as React span children (auto-escaped — T-139-07).
 *   Colors resolved through ITheme via resolveColor.
 *   NO dangerouslySetInnerHTML on the local path.
 * Remote tail (CR-02): sourced from the WS scrollback snapshot the peer replays
 *   on connect; processed by headless @xterm/xterm + serializeAsHTML (no term.open());
 *   rendered via dangerouslySetInnerHTML (T-139-08: trust level same as xterm rendering
 *   the session; scope limited to the briefing modal only).
 *
 * CR-03 fix: clientRef + settled flag + clearTimeout + useEffect unmount cleanup
 * ensure the WS and its ping interval are always torn down, and that abandoned
 * text is never written to the PTY after the 5s timeout.
 */
export function HubBriefingModal({
  session,
  relayPort,
  onClose,
  theme,
  remote,
}: HubBriefingModalProps): React.ReactElement {
  // Local path: StyledSpan[][] or null (loading)
  const [tailLines, setTailLines] = useState<StyledSpan[][] | null>(null)
  // Remote path: serialized HTML string or null (loading)
  const [remoteHtml, setRemoteHtml] = useState<string | null>(null)
  const [responseText, setResponseText] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState<string | null>(null)
  const respondInputRef = useRef<HTMLTextAreaElement>(null)

  // CR-03: track the in-flight RelayClient so the unmount cleanup can close it.
  const clientRef = useRef<RelayClient | null>(null)

  // Fetch the last 20 lines of terminal output on mount (one-shot).
  // CR-02: remote sessions read the scrollback snapshot replayed on the proxied WS
  // instead of calling GetSessionStyledTailLines (which is local-only and returns []
  // for remote ids — engine.go:550). A 3s timeout guards against stalled connections.
  //
  // WR-01: return a cleanup function so React tears down the tailClient (and its
  // 30s ping interval) if the modal unmounts during the tail collection window —
  // mirrors the CR-03 send-path fix.
  useEffect(() => {
    if (remote) {
      // Remote path: open a short-lived proxied RelayClient, accumulate MsgOutput
      // payload bytes from the scrollback snapshot, then close.
      // Phase 139 / CARD-05: replace extractTailLines regex with headless Terminal
      // + serializeAsHTML (no term.open() — Pitfall 5). A2 PASS verdict from Plan 01
      // confirms this pattern works in jsdom and production Chromium WebView.
      const chunks: Uint8Array[] = []
      let tailClient: RelayClient | null = null
      let resolved = false
      // WR-04: idle timer handle — reset on every onOutput frame so finish()
      // fires when output goes quiet rather than after a fixed 500ms window.
      let idleTimerId: ReturnType<typeof setTimeout> | null = null

      const finish = () => {
        if (resolved) return
        resolved = true
        clearTimeout(timeoutId) // IN-01: clear 3s guard uniformly on all exit paths
        if (idleTimerId !== null) clearTimeout(idleTimerId)
        tailClient?.close()
        tailClient = null

        // Headless xterm render: merge chunks → Terminal.write → serializeAsHTML → dispose.
        // MUST NOT call term.open() (Pitfall 5 — headless use skips DOM attachment).
        // T-139-08: XSS risk accepted — output is agent-owned PTY session; scope limited to briefing modal.
        // NOTE: term.write() is asynchronous — use the callback form and await via Promise
        // (see A2 verification test: xtermHeadless.verify.test.ts).
        const total = chunks.reduce((sum, c) => sum + c.length, 0)
        const merged = new Uint8Array(total)
        let offset = 0
        for (const chunk of chunks) {
          merged.set(chunk, offset)
          offset += chunk.length
        }
        const term = new Terminal({ cols: 80, rows: 50, allowProposedApi: true, theme })
        const serAddon = new SerializeAddon()
        term.loadAddon(serAddon)
        // Use callback form — term.write() is asynchronous and flushes after the callback fires.
        term.write(merged, () => {
          const html = serAddon.serializeAsHTML({ scrollback: 20, includeGlobalBackground: false })
          term.dispose()
          setRemoteHtml(html)
        })
      }

      const timeoutId = setTimeout(finish, 3000)

      tailClient = new RelayClient(
        relayPort,
        session.id,
        {
          onOutput: (data) => {
            // Accumulate all MsgOutput bytes from the scrollback snapshot.
            // WR-04: reset the idle timer on every frame — finish() fires
            // when output goes quiet for ~150ms, collecting the full snapshot
            // regardless of size (fixes the fixed 500ms truncation race).
            chunks.push(data)
            if (idleTimerId !== null) clearTimeout(idleTimerId)
            idleTimerId = setTimeout(finish, 150)
          },
          onOpen: () => {
            // WR-04: on connect, arm the idle timer immediately (in case the
            // scrollback is empty and no onOutput frames arrive at all).
            // The idle timer will fire after 150ms of quiet; if output arrives
            // first the onOutput handler will reset it.
            if (idleTimerId !== null) clearTimeout(idleTimerId)
            idleTimerId = setTimeout(finish, 150)
          },
          onClose: () => {
            finish()
          },
        },
        { remote: true },
      )

      // WR-01: unmount cleanup — closes tailClient (and its 30s ping interval)
      // if the modal is dismissed during the tail collection window.
      return () => {
        clearTimeout(timeoutId)
        if (idleTimerId !== null) clearTimeout(idleTimerId)
        tailClient?.close()
      }
    } else {
      // Local path: Phase 139 / CARD-05 — use GetSessionStyledTailLines instead of
      // GetSessionTailLines. Renders via React children (auto-escaped, T-139-07).
      GetSessionStyledTailLines(session.id, 20)
        .then((lines) => setTailLines(lines))
        .catch(() => setTailLines([]))
    }
  }, [session.id, relayPort, remote, theme])

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

  // ---- Tail display ----
  // Local path: tailLines is StyledSpan[][] | null; render as styled React spans (T-139-07).
  // Remote path: remoteHtml is string | null; render via dangerouslySetInnerHTML (T-139-08).

  // Determine loading/empty/data state for the local path
  const localLoading = !remote && tailLines === null
  const localEmpty = !remote && Array.isArray(tailLines) && tailLines.length === 0
  const remoteLoading = remote && remoteHtml === null
  const isLoading = localLoading || remoteLoading
  const isEmpty = localEmpty || (remote && remoteHtml === '')

  return (
    <div className="hub-modal__body hub-modal__body--briefing">
      {/* Terminal tail display */}
      <div
        className={[
          'hub-modal__tail',
          isLoading ? 'hub-modal__tail--loading' : '',
          isEmpty ? 'hub-modal__tail--empty' : '',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        {isLoading ? (
          <span>Loading…</span>
        ) : isEmpty ? (
          <span>No recent output available.</span>
        ) : remote ? (
          // Remote path: headless xterm serialized HTML (T-139-08 — ONLY dangerouslySetInnerHTML)
          <div
            className="hub-modal__tail-remote"
            // eslint-disable-next-line react/no-danger
            dangerouslySetInnerHTML={{ __html: remoteHtml ?? '' }}
          />
        ) : (
          // Local path: styled React children (auto-escaped, T-139-07 — no dangerouslySetInnerHTML)
          <div className="hub-modal__tail-local">
            {(tailLines ?? []).map((row, i) => (
              <div key={i} className="hub-modal__tail-line">
                {row.length === 0 ? (
                  <span>{' '}</span>
                ) : (
                  row.map((span, j) => (
                    <span
                      key={j}
                      style={{
                        color: resolveColor(span.fg, theme, true),
                        background: resolveColor(span.bg, theme, false),
                        fontWeight: span.b ? 'bold' : undefined,
                      }}
                    >
                      {span.c || ' '}
                    </span>
                  ))
                )}
              </div>
            ))}
          </div>
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
          {/* NOTE (carry-forward): For a remote session with a read-only cap, the Send button
              appears enabled but the input is silently dropped at the peer (the cap does not
              grant write access to the PTY). A non-color read-only indicator (colorblind-safe
              per user constraint) is deferred to Phase 135 (a11y). */}
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
