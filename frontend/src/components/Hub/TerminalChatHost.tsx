import React, { useState } from 'react'
import type { ITheme } from '@xterm/xterm'
import type { IProgressState } from '@xterm/addon-progress'
import { daemon } from '../../wailsjs/go/models'
import { ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline'
import { TerminalPanel } from '../TerminalPanel'
import { ChatPanel } from './ChatPanel'
import { ChatBadge } from './ChatBadge'

type PluginSettings = daemon.PluginSettings

export interface TerminalChatHostProps {
  /** Session to display. */
  sessionId: string
  /** True when this tab is the active tab (fit-safe timing guard — no 0-column resize).
   *  Forwarded directly to TerminalPanel — NOT bound to chatOpen (D-02 no PTY resize). */
  isActive: boolean
  /** Local relay port for TerminalPanel and ChatPanel. */
  relayPort: number
  /** Current font size. */
  fontSize: number
  /** Font-size change handler. */
  onFontSizeChange: (delta: number) => void
  /** Terminal colour theme. */
  theme: ITheme
  /** Plugin configuration. */
  pluginConfig?: PluginSettings | null
  /** Phase 93 WGL-02/WGL-03: fired on WebGL fallback to DOM or software-rasteriser detection. */
  onWebGLContextLost?: (reason: 'context-loss' | 'software-rasterized') => void
  /** Phase 97 SER-01: register/unregister serialise() closure with App.tsx saver registry. */
  onRegisterSaver?: (sessionId: string, fn: (() => string) | null) => void
  /** Phase 98 PRG-02: forward ProgressAddon onChange events to App.tsx for aggregation. */
  onProgressChange?: (sessionId: string, state: IProgressState) => void
  /** When true, TerminalPanel routes through the daemon WS proxy (remote sessions). */
  remote?: boolean
}

/**
 * TerminalChatHost — overlay host for the GUI terminal tab.
 *
 * Mounts TerminalPanel (full-bleed) + an always-mounted ChatPanel overlay + a
 * chat-toggle button, matching the structure of HubInteractiveModal and
 * WebShareSessionView for cross-surface parity (CHAT-PARITY-01 / PARITY-01).
 *
 * Layout (D-02 overlay mode): the chat drawer is position:absolute over the
 * terminal's right edge; the PTY is NEVER resized when the drawer opens/closes
 * (isActive is forwarded from props, not bound to chatOpen).
 *
 * D-09: ChatPanel is always-mounted so unread messages accrue while drawer is closed.
 *
 * The host wraps ONLY TerminalPanel — ExitCountdownBanner and StatusBar stay as
 * siblings in App.tsx so the absolute drawer (top:0; bottom:0) does not cover them.
 */
export function TerminalChatHost({
  sessionId,
  isActive,
  relayPort,
  fontSize,
  onFontSizeChange,
  theme,
  pluginConfig,
  onWebGLContextLost,
  onRegisterSaver,
  onProgressChange,
  remote,
}: TerminalChatHostProps): React.ReactElement {
  // D-02 overlay drawer state — mirror WebShareSessionView / HubInteractiveModal
  const [chatOpen, setChatOpen] = useState(false)
  // NOTIF-01 / D-10: unread badge state driven by ChatPanel's onUnreadChange callback
  const [unreadCount, setUnreadCount] = useState(0)
  const [hasMention, setHasMention] = useState(false)

  function handleUnreadChange(count: number, mention: boolean) {
    setUnreadCount(count)
    setHasMention(mention)
  }

  return (
    <div className="terminal-chat-host">
      {/* D-02: TerminalPanel is full-bleed; isActive forwarded from props, NOT from
          chatOpen — toggling the chat drawer never triggers a PTY sendResize (D-02). */}
      <TerminalPanel
        sessionId={sessionId}
        isActive={isActive}
        relayPort={relayPort}
        fontSize={fontSize}
        onFontSizeChange={onFontSizeChange}
        theme={theme}
        pluginConfig={pluginConfig}
        onWebGLContextLost={onWebGLContextLost}
        onRegisterSaver={onRegisterSaver}
        onProgressChange={onProgressChange}
        remote={remote}
      />

      {/* D-09: ChatPanel is always-mounted (open prop controls visibility only).
          Unread messages accrue in ChatPanel state even while the drawer is closed.
          Desktop path: no wsURL/apiBaseURL/capToken — connects via loopback relay. */}
      <ChatPanel
        sessionId={sessionId}
        relayPort={relayPort}
        open={chatOpen}
        onUnreadChange={handleUnreadChange}
      />

      {/* Chat toggle button — floats over the bottom-right of the terminal area.
          Renders AFTER ChatPanel so the .chat-panel--open ~ .hub-modal__chat-toggle
          general-sibling combinator (158-01) correctly relocates the toggle when
          the drawer is open, clearing the Send/Inject overlap (CHAT-FIX-01).
          COLORBLIND-SAFE: ChatBubbleLeftRightIcon (shape) + aria-label are primary signals. */}
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
