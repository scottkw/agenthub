import React, { useState } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import { ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline'
import { TerminalPanel } from '../TerminalPanel'
import { ChatPanel } from './ChatPanel'
import { ChatBadge } from './ChatBadge'

type PluginSettings = daemon.PluginSettings

export interface HubInteractiveModalProps {
  session: SessionInfo
  /** True only when the modal phase is 'open' (grow animation complete).
   *  Must be false during the 220ms enter animation to prevent TerminalPanel
   *  from computing 0-column dimensions (fit-safe timing guard — Pitfall 1).
   *  The HubModal shell passes `phase === 'open'` here. */
  isOpen: boolean
  relayPort: number
  fontSize: number
  theme: ITheme
  pluginConfig?: PluginSettings | null
  onFontSizeChange?: (delta: number) => void
  /** When true, TerminalPanel routes through the daemon WS proxy at
   *  /api/relay/remote/{id}/ws (fixes CR-01 for remote sessions).
   *  The cap token is NOT passed here — the daemon looks it up by sessionID. */
  remote?: boolean
  /** NOTIF-01: lift unread state to HubPanel's unreadMap.
   *  Called with (session.id, count, hasMention) each time ChatPanel reports
   *  a change. Injecting session.id here because ChatPanel's own callback
   *  signature is (count, hasMention) without a session identifier. */
  onUnreadChange?: (sessionId: string, count: number, hasMention: boolean) => void
}

/**
 * HubInteractiveModal mounts the existing TerminalPanel for non-blocked sessions.
 * isActive is gated on the 'open' phase: false during the 220ms grow animation,
 * true once the modal is fully open — preventing TerminalPanel from computing
 * 0-column dimensions during animation (Pitfall 1).
 * Does NOT construct its own RelayClient — TerminalPanel owns it internally.
 *
 * D-02 OVERLAY MODE: ChatPanel is position:absolute over the terminal's right edge.
 * The terminal is NOT resized when the drawer opens/closes — isActive remains bound
 * to the modal-open prop (not chatOpen). The drawer covers ~360px of terminal width
 * when open, but never triggers a PTY sendResize.
 *
 * D-09: ChatPanel is always-mounted so unread messages accrue while the drawer is closed.
 */
export function HubInteractiveModal({
  session,
  isOpen: open,
  relayPort,
  fontSize,
  theme,
  pluginConfig,
  onFontSizeChange,
  remote,
  onUnreadChange,
}: HubInteractiveModalProps): React.ReactElement {
  // D-02 overlay drawer state
  const [chatOpen, setChatOpen] = useState(false)
  // NOTIF-01 / D-10: unread badge state driven by ChatPanel's onUnreadChange callback
  const [unreadCount, setUnreadCount] = useState(0)
  const [hasMention, setHasMention] = useState(false)

  function handleUnreadChange(count: number, mention: boolean) {
    setUnreadCount(count)
    setHasMention(mention)
    // NOTIF-01: inject session.id and forward to HubPanel's unreadMap handler.
    // ChatPanel's callback signature is (count, hasMention) — no sessionId.
    // This is the injection point that adds the session identifier.
    onUnreadChange?.(session.id, count, mention)
  }

  return (
    <div className="hub-modal__body hub-modal__body--interactive">
      {/* D-02: TerminalPanel is full-bleed; no column-shrink wrapper.
          isActive is bound to the modal-open prop (open), NOT chatOpen.
          Toggling the chat drawer never triggers a PTY sendResize (D-02). */}
      <TerminalPanel
        sessionId={session.id}
        isActive={open}
        relayPort={relayPort}
        fontSize={fontSize}
        onFontSizeChange={onFontSizeChange ?? (() => {})}
        theme={theme}
        pluginConfig={pluginConfig}
        remote={remote}
      />

      {/* D-09: ChatPanel is always-mounted (open prop controls visibility only).
          Unread messages accrue in ChatPanel state even while the drawer is closed. */}
      <ChatPanel
        sessionId={session.id}
        relayPort={relayPort}
        open={chatOpen}
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
