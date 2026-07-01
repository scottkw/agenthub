import React, { useState } from 'react'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import { ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline'
import { TerminalPanel } from '../TerminalPanel'
import { ChatPanel } from './ChatPanel'
import { ChatBadge } from './ChatBadge'

type PluginSettings = daemon.PluginSettings

export interface WebShareSessionViewProps {
  /** Session to display. */
  sessionId: string
  /** Web-share capability token from URL ?cap= param. Never decoded client-side. */
  capToken: string
  /** Local relay port — always 0 on web-share (safe sentinel, ignored when wsURL present). */
  relayPort: number
  /** Terminal colour theme (pass terminalTheme from App.tsx). */
  theme?: ITheme
  /** Plugin configuration (pass pluginConfig from App.tsx). */
  pluginConfig?: PluginSettings | null
  /**
   * Origin to derive apiBaseURL/wsURL from (e.g. `https://peer.example.ts.net`).
   * Defaults to `window.location.origin` — the native web-share guest path.
   * FIX-03 (plan 03) supplies a different peer's origin for in-app remote-peer
   * tabs; this seam MUST stay parameterized (never hardcode window.location
   * inside URL-building logic below).
   */
  baseURL?: string
}

/**
 * WebShareSessionView mounts TerminalPanel + ChatPanel for the web-share browser
 * surface. Both panels connect via the webserver WS endpoint (`wss://`) rather
 * than the loopback relay, using the wsURL override introduced in Phase 155.
 *
 * Layout mirrors HubInteractiveModal (D-02 overlay mode): the chat drawer
 * position:absolute over the terminal; the terminal is never resized when the
 * drawer opens (isActive is always true here — no grow animation).
 *
 * CSS class names are verbatim copies of HubInteractiveModal so Playwright
 * parity selectors work identically on both surfaces (PARITY-01).
 */
export function WebShareSessionView({
  sessionId,
  capToken,
  relayPort,
  theme,
  pluginConfig,
  baseURL,
}: WebShareSessionViewProps): React.ReactElement {
  // D-02 overlay drawer state — same hooks as HubInteractiveModal
  const [chatOpen, setChatOpen] = useState(false)
  // NOTIF-01 / D-10: unread badge state driven by ChatPanel's onUnreadChange callback
  const [unreadCount, setUnreadCount] = useState(0)
  const [hasMention, setHasMention] = useState(false)

  function handleUnreadChange(count: number, mention: boolean) {
    setUnreadCount(count)
    setHasMention(mention)
  }

  // Phase 168 FIX-01/FIX-03 — baseURL seam. Defaults to window.location.origin
  // (today's native web-share guest behavior, unchanged). FIX-03 (plan 03)
  // passes a different peer's origin for in-app remote-peer tabs; apiBaseURL
  // and wsURL MUST derive from this resolved origin, never a hardcoded
  // window.location reference, so both plans share the same seam.
  const resolvedOrigin = baseURL ?? window.location.origin
  const apiBaseURL = resolvedOrigin
  // Convert the resolved origin's http/https scheme to ws/wss for the socket
  // (replaces the old hardcoded `wss://${window.location.host}`).
  const wsOrigin = resolvedOrigin.replace(/^http/, 'ws')

  // Phase 155 — webserver WS URL. Both TerminalPanel and ChatPanel connect
  // via this URL (Pitfall 6: forgetting wsURL on TerminalPanel leaves the
  // terminal on ws://127.0.0.1:0 while chat works).
  const wsURL = `${wsOrigin}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`

  return (
    <div className="hub-modal__body hub-modal__body--interactive">
      {/* D-02: TerminalPanel is full-bleed; no column-shrink wrapper.
          isActive={true} — constant on web-share (no grow animation here,
          unlike the modal's animated isActive={open}). Toggling the chat
          drawer never triggers a PTY sendResize (D-02). */}
      <TerminalPanel
        sessionId={sessionId}
        isActive={true}
        relayPort={relayPort}
        fontSize={14}
        onFontSizeChange={() => {}}
        theme={theme ?? {}}
        pluginConfig={pluginConfig}
        wsURL={wsURL}
      />

      {/* D-09: ChatPanel is always-mounted (open prop controls visibility only).
          Unread messages accrue in ChatPanel state even while the drawer is closed.
          Phase 155: wsURL, apiBaseURL, and capToken are forwarded so ChatPanel
          connects via webserver and can fetch history/export with the cap. */}
      <ChatPanel
        sessionId={sessionId}
        relayPort={relayPort}
        open={chatOpen}
        wsURL={wsURL}
        apiBaseURL={apiBaseURL}
        capToken={capToken}
        onUnreadChange={handleUnreadChange}
      />

      {/* Chat toggle button — floats over the bottom-right of the terminal body.
          COLORBLIND-SAFE: ChatBubbleLeftRightIcon (shape) + aria-label are primary signals.
          ChatBadge provides the unread count / mention glyph. */}
      <button
        type="button"
        className="hub-modal__chat-toggle"
        aria-label={chatOpen ? 'Close chat' : 'Open chat'}
        onClick={() => setChatOpen((prev) => !prev)}
      >
        <ChatBubbleLeftRightIcon className="hub-modal__chat-toggle-icon" aria-hidden="true" />
        <ChatBadge count={unreadCount} hasMention={hasMention} />
      </button>
    </div>
  )
}
