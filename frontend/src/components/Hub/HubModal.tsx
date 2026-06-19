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
 * - Escape key handler (dialog-scoped onKeyDown with stopPropagation — WR-05)
 * - Focus return to originating card on unmount (cardFocusRef)
 * - Header strip with status icon, session name, CLI badge, origin marker, attention badge
 * - Routing to HubInteractiveModal (non-attention) or HubBriefingModal (attention)
 * - transformOrigin set from sourceRect center (MODAL-01 grow-animation origin)
 *
 * Security: session.name / hostname rendered as React text content — no dangerouslySetInnerHTML (T-134-04-01).
 * Escape: the dialog onKeyDown calls preventDefault + stopPropagation. Isolation from the
 * originating Hub card's own Escape handler (SessionCard onKeyDown) is provided by the inert
 * background trap (the card is `inert` while the modal is open, so it cannot receive key
 * events) — NOT by stopPropagation, which only stops bubbling within the dialog's own React
 * subtree and has no effect on the card's sibling subtree under .hub (WR-02 / WR-05).
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
  const { Icon: StatusIcon, label: statusLabel } = STATUS_CONFIG[hubStatus]

  // ---- Animation phase machine ----
  // entering → (animationEnd) → open; exiting → (animationEnd) → onClose()
  //
  // GAP-134-D: under prefers-reduced-motion the grow/shrink animations are disabled
  // (CSS `animation: none`), so onAnimationEnd NEVER fires. Driving the phase machine
  // purely off onAnimationEnd then breaks both transitions: the modal stays stuck in
  // 'entering' (interactive terminal never activates) and — worse — can never close
  // (exiting → onClose never runs). When reduced motion is requested we therefore skip
  // the animated phases entirely: open immediately and close synchronously.
  // Guarded for jsdom/SSR where window.matchMedia is undefined (→ animated path).
  const prefersReducedMotion =
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches

  const [phase, setPhase] = useState<'entering' | 'open' | 'exiting'>(
    prefersReducedMotion ? 'open' : 'entering',
  )

  const handleClose = useCallback(() => {
    if (prefersReducedMotion) {
      onClose() // no exit animation → onAnimationEnd won't fire; close synchronously
      return
    }
    setPhase('exiting')
  }, [prefersReducedMotion, onClose])

  // ---- Focus return on unmount (MODAL-02) ----
  // Store the originating card (document.activeElement at mount) and return focus on unmount.
  const cardFocusRef = useRef<HTMLElement | null>(null)
  useEffect(() => {
    cardFocusRef.current = document.activeElement as HTMLElement
    return () => {
      cardFocusRef.current?.focus()
    }
  }, [])

  // ---- A11Y-04: Focus trap via inert + initial focus ----
  // Mark the .hub background inert so Tab cannot reach background cards while the modal
  // is mounted. The trap must persist through BOTH 'open' AND 'exiting' — dropping it
  // during the exit animation (WR-01) re-exposes background cards while the modal is
  // still mounted with aria-modal="true". Only the 'entering' grow animation is excluded
  // (Pitfall 3). Cleanup removes inert on every phase change AND on unmount — because the
  // effect re-runs and re-applies inert on each non-entering phase, there is never a gap
  // where the background is left interactive while the modal is open (keyboard-lock guard:
  // cleanup ALWAYS runs `inert = false`, so no path leaves .hub permanently inert).
  // Move initial focus to the close button only once the modal is fully 'open' (WCAG 2.4.3).
  // The cardFocusRef cleanup (above) handles focus-return to the originating card — do NOT duplicate it here.
  const closeBtnRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (phase === 'entering') return // Pitfall 3: do not trap during 'entering' grow animation

    const hubEl = document.querySelector('.hub') as HTMLElement | null
    if (hubEl) hubEl.inert = true // trap stays applied through 'open' AND 'exiting'

    if (phase === 'open') closeBtnRef.current?.focus()

    return () => {
      if (hubEl) hubEl.inert = false // Pitfall 1: MUST remove inert or Hub keyboard-locks
    }
  }, [phase])

  // WR-05: Escape is handled via onKeyDown on the role="dialog" element (see JSX below).
  // This replaces the previous document-level Escape listener (Phase 134, now removed).
  // The dialog-scoped handler uses stopPropagation which stops bubbling within the component
  // tree without globally suppressing other document Escape handlers.

  // ---- transform-origin — grow animation originates from card center (MODAL-01) ----
  const transformOrigin = `${sourceRect.left + sourceRect.width / 2}px ${sourceRect.top + sourceRect.height / 2}px`

  // ---- Header strip data ----
  // GAP-134-C: local vs remote is provenance (the `remote` prop, derived in HubPanel from
  // the remoteSessions list), NOT hostname — local sessions carry the machine os.Hostname(),
  // so a hostname check mislabels every local session with the globe icon + machine name.
  const isLocal = !remote
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
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault()
            e.stopPropagation()
            handleClose()
          }
        }}
        onAnimationEnd={() => {
          if (phase === 'entering') setPhase('open')
          if (phase === 'exiting') onClose()
        }}
      >
        {/* ---- Header strip ---- */}
        <div className="hub-modal__header">
          {/* A11Y-01: the heroicon is decorative (Heroicons hard-code aria-hidden),
              so status is conveyed to AT by a visually-hidden text label — mirrors
              SessionCard's visible status-label span. Colorblind-safe: text, not color. */}
          <StatusIcon className="hub-modal__status-icon" aria-hidden="true" />
          <span className="sr-only">{statusLabel}</span>
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
            ref={closeBtnRef}
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
