import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionInfo } from '../../wailsjs/go/main/App'
import type { ITheme } from '@xterm/xterm'
import { daemon } from '../../wailsjs/go/models'
import {
  ArrowPathIcon,
  CheckCircleIcon,
  PauseCircleIcon,
  ExclamationCircleIcon,
  StopCircleIcon,
  ComputerDesktopIcon,
  GlobeAltIcon,
  BellAlertIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline'
import { deriveHubStatus, isAttentionStatus } from '../../lib/hubStatus'
import type { HubStatus } from '../../lib/hubStatus'
import { HubInteractiveModal } from './HubInteractiveModal'
import { HubBriefingModal } from './HubBriefingModal'

type PluginSettings = daemon.PluginSettings

// ---- STATUS_CONFIG ----
// COLORBLIND-SAFE: every status has unique icon shape + text label; color is reinforcement only.
const STATUS_CONFIG: Record<
  HubStatus,
  { Icon: React.ComponentType<React.SVGProps<SVGSVGElement>>; label: string; spin: boolean }
> = {
  running: { Icon: ArrowPathIcon, label: 'Running', spin: true },
  idle: { Icon: CheckCircleIcon, label: 'Idle', spin: false },
  waiting: { Icon: PauseCircleIcon, label: 'Needs input', spin: false },
  errored: { Icon: ExclamationCircleIcon, label: 'Error', spin: false },
  'stopped-ok': { Icon: StopCircleIcon, label: 'Done', spin: false },
  'stopped-err': { Icon: ExclamationCircleIcon, label: 'Exited', spin: false },
}

export interface HubModalProps {
  session: SessionInfo
  /** Card bounding rect — used to compute transformOrigin for the grow animation (MODAL-01) */
  sourceRect: DOMRect
  relayPort: number
  fontSize: number
  theme: ITheme
  pluginConfig?: PluginSettings | null
  /** WR-04: font size change callback; forwarded to HubInteractiveModal → TerminalPanel */
  onFontSizeChange?: (delta: number) => void
  /** Plan 07 seam: when true, routes through the daemon WS proxy for remote sessions (CR-01 fix). */
  remote?: boolean
  onClose: () => void
}

/**
 * HubModal — modal shell that composes HubInteractiveModal / HubBriefingModal.
 *
 * Owns:
 * - Overlay (z-index 200) + click-outside dismissal
 * - Grow/shrink animation phase machine: entering → open → exiting
 * - Escape key handler (stopImmediatePropagation — prevents Hub card's Escape from double-firing)
 * - Focus return to originating card on unmount (cardFocusRef)
 * - Header strip with status icon, session name, CLI badge, origin marker, attention badge
 * - Routing to HubInteractiveModal (non-attention) or HubBriefingModal (attention)
 * - transformOrigin set from sourceRect center (MODAL-01 grow-animation origin)
 *
 * Security: session.name / hostname rendered as React text content — no dangerouslySetInnerHTML (T-134-04-01).
 * Escape: stopImmediatePropagation guards against Hub card menu double-fire (T-134-04-02, Pitfall 6).
 */
export function HubModal({
  session,
  sourceRect,
  relayPort,
  fontSize,
  theme,
  pluginConfig,
  onFontSizeChange,
  remote,
  onClose,
}: HubModalProps): React.ReactElement {
  const hubStatus = deriveHubStatus(session)
  const isBriefing = isAttentionStatus(hubStatus)
  const { Icon: StatusIcon } = STATUS_CONFIG[hubStatus]

  // ---- Animation phase machine ----
  // entering → (animationEnd) → open; exiting → (animationEnd) → onClose()
  const [phase, setPhase] = useState<'entering' | 'open' | 'exiting'>('entering')

  const handleClose = useCallback(() => {
    setPhase('exiting')
  }, [])

  // ---- Focus return on unmount (MODAL-02) ----
  // Store the originating card (document.activeElement at mount) and return focus on unmount.
  const cardFocusRef = useRef<HTMLElement | null>(null)
  useEffect(() => {
    cardFocusRef.current = document.activeElement as HTMLElement
    return () => {
      cardFocusRef.current?.focus()
    }
  }, [])

  // ---- Escape key handler (MODAL-02, T-134-04-02) ----
  // Use a ref to handleClose so the listener stays stable without re-adding on each render.
  const handleCloseRef = useRef(handleClose)
  useEffect(() => {
    handleCloseRef.current = handleClose
  })
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent): void {
      if (e.key === 'Escape') {
        e.stopImmediatePropagation() // Prevent Hub card menu Escape from also firing (Pitfall 6)
        handleCloseRef.current()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  // ---- transform-origin — grow animation originates from card center (MODAL-01) ----
  const transformOrigin = `${sourceRect.left + sourceRect.width / 2}px ${sourceRect.top + sourceRect.height / 2}px`

  // ---- Header strip data ----
  const isLocal = !session.hostname || session.hostname === ''
  const originText = isLocal ? 'Local' : session.hostname

  // ---- ARIA label — copywriting contract ----
  const ariaLabel = isBriefing
    ? `Briefing: ${session.name} needs input`
    : `Session terminal: ${session.name}`

  return (
    <div
      className={`hub-modal-overlay hub-modal-overlay--${phase}`}
      onClick={handleClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        className={`hub-modal hub-modal--${isBriefing ? 'briefing' : 'interactive'} hub-modal--${phase}`}
        style={{ transformOrigin }}
        onClick={(e) => e.stopPropagation()}
        onAnimationEnd={() => {
          if (phase === 'entering') setPhase('open')
          if (phase === 'exiting') onClose()
        }}
      >
        {/* ---- Header strip ---- */}
        <div className="hub-modal__header">
          <StatusIcon className="hub-modal__status-icon" aria-hidden="true" />
          <span className="hub-modal__session-name">{session.name}</span>
          <span className="hub-card__badge">{session.cli}</span>
          {isLocal ? (
            <ComputerDesktopIcon className="hub-modal__origin-icon" aria-hidden="true" />
          ) : (
            <GlobeAltIcon className="hub-modal__origin-icon" aria-hidden="true" />
          )}
          <span className="hub-modal__origin-text">{originText}</span>
          {isBriefing && (
            <span className="hub-modal__attn-badge">
              <BellAlertIcon className="hub-modal__attn-icon" aria-hidden="true" />
              <span>Needs attention</span>
            </span>
          )}
          <span style={{ flex: 1 }} />
          <button
            type="button"
            className="hub-modal__close"
            aria-label="Close modal"
            onClick={handleClose}
          >
            <XMarkIcon aria-hidden="true" />
          </button>
        </div>

        {/* ---- Body: route to leaf component by attention status ----
            remote prop threads the Plan 07 isRemote discriminator to both branches:
            - HubBriefingModal: remote tail via proxied WS scrollback (CR-02) + remote send (CR-01)
            - HubInteractiveModal: routes TerminalPanel through /api/relay/remote/{id}/ws (CR-01) */}
        {isBriefing ? (
          <HubBriefingModal
            session={session}
            relayPort={relayPort}
            remote={remote}
            onClose={handleClose}
          />
        ) : (
          <HubInteractiveModal
            session={session}
            isOpen={phase === 'open'}
            relayPort={relayPort}
            fontSize={fontSize}
            theme={theme}
            pluginConfig={pluginConfig}
            remote={remote}
            onFontSizeChange={onFontSizeChange}
          />
        )}
      </div>
    </div>
  )
}
