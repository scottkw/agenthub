import React from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import { TerminalPanel } from '../TerminalPanel'

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
}

/**
 * HubInteractiveModal mounts the existing TerminalPanel for non-blocked sessions.
 * isActive is gated on the 'open' phase: false during the 220ms grow animation,
 * true once the modal is fully open — preventing TerminalPanel from computing
 * 0-column dimensions during animation (Pitfall 1).
 * Does NOT construct its own RelayClient — TerminalPanel owns it internally.
 */
export function HubInteractiveModal({
  session,
  isOpen: open,
  relayPort,
  fontSize,
  theme,
  pluginConfig,
  onFontSizeChange,
}: HubInteractiveModalProps): React.ReactElement {
  return (
    <div className="hub-modal__body hub-modal__body--interactive">
      <TerminalPanel
        sessionId={session.id}
        isActive={open}
        relayPort={relayPort}
        fontSize={fontSize}
        onFontSizeChange={onFontSizeChange ?? (() => {})}
        theme={theme}
        pluginConfig={pluginConfig}
      />
    </div>
  )
}
