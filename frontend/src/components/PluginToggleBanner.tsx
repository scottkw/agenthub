import React, { useEffect } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

/**
 * PluginToggleBanner — Phase 99 PUI-02.
 *
 * One-shot informational banner rendered inside the existing `.banner-stack`
 * (Phase 81 BAN-01/BAN-02 vocabulary) when the user toggles a plugin that
 * cannot hot-swap (Unicode 11 or Inline Images) and saves. Both variants
 * auto-dismiss after 6000ms (unconditional — differs from WebGLRecoveryBanner
 * where 'software-rasterized' is persistent).
 *
 * Verbatim copy locked by 99-RESEARCH.md "Claude's Discretion" — DO NOT
 * paraphrase. Reuses `.webgl-recovery-banner` BEM class verbatim for visual
 * continuity (same TokyoNight info palette, same 53px height, same × button).
 * Zero new CSS introduced — Phase 99 inherits the Phase 92 CSS discipline.
 */
export interface PluginToggleBannerProps {
  kind: 'unicode11' | 'image'
  onDismiss: () => void
  className?: string
}

export function PluginToggleBanner({
  kind,
  onDismiss,
  className,
}: PluginToggleBannerProps): React.ReactElement {
  useEffect(() => {
    const id = window.setTimeout(onDismiss, 6000)
    return () => window.clearTimeout(id)
  }, [onDismiss])

  const message =
    kind === 'unicode11'
      ? 'Open a new terminal session to apply the Unicode 11 change.'
      : 'Open a new terminal session to apply the Inline Images change.'

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
