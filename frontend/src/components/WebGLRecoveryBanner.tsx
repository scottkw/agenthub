import React, { useEffect } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

/**
 * WebGLRecoveryBanner — Phase 93 WGL-02 / WGL-03.
 *
 * One-shot informational banner rendered inside the existing `.banner-stack`
 * (Phase 81 BAN-01/BAN-02 vocabulary) when the WebGL renderer is unavailable
 * or has fallen back to DOM. Two variants:
 *
 * - reason='context-loss'        — Hardware GPU context was lost; DOM renderer
 *                                  has taken over; scrollback is intact. Tone:
 *                                  info (recovery already happened). Auto-
 *                                  dismisses after 8000ms.
 * - reason='software-rasterized' — Software WebGL detected at startup
 *                                  (SwiftShader/llvmpipe/ANGLE-software);
 *                                  DOM renderer used preemptively for
 *                                  performance. Persistent (rare event,
 *                                  worth keeping visible until acknowledged).
 *
 * Verbatim copy locked by 93-UI-SPEC §"Copywriting Contract" — DO NOT
 * paraphrase. Information-disclosure mitigation: messages never include
 * "SwiftShader", "llvmpipe", "ANGLE", or any internal-detection vocabulary.
 */
export interface WebGLRecoveryBannerProps {
  reason: 'context-loss' | 'software-rasterized'
  onDismiss: () => void
  className?: string
}

export function WebGLRecoveryBanner({
  reason,
  onDismiss,
  className,
}: WebGLRecoveryBannerProps): React.ReactElement {
  useEffect(() => {
    if (reason !== 'context-loss') return
    const id = window.setTimeout(onDismiss, 8000)
    return () => window.clearTimeout(id)
  }, [reason, onDismiss])

  const message =
    reason === 'context-loss'
      ? 'Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact.'
      : 'Hardware acceleration is unavailable on this device. Your terminal is using the standard renderer for the best experience.'

  const cls = ['webgl-recovery-banner', className].filter(Boolean).join(' ')

  return (
    <div className={cls} role="status" aria-live="polite">
      <span className="webgl-recovery-banner__message">{message}</span>
      <button
        type="button"
        className="webgl-recovery-banner__dismiss"
        aria-label="Dismiss notification"
        onClick={onDismiss}
      >
        <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
      </button>
    </div>
  )
}
