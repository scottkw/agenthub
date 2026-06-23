import React, { useEffect, useRef, useState } from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

/**
 * ShellWebShareBanner — Phase 101-03 SHELL-08.
 *
 * One-time security warning rendered at the TOP of the existing `.banner-stack`
 * when the user toggles web sharing ON for a shell session for the first time.
 * Unlike the informational PluginToggleBanner / WebGLRecoveryBanner (role=status,
 * aria-live=polite), this banner BLOCKS an action — the toggle does not complete
 * until the user confirms or cancels. Therefore role="alert" + aria-live="assertive".
 *
 * Reuses the `.webgl-recovery-banner` BEM class verbatim for visual continuity
 * (Phase 99 PUI-02 precedent), with one new modifier `--shell-warning` adding the
 * 3px destructive-red left-border accent. Banner height stays at 53px (locked
 * stack rhythm).
 *
 * Verbatim copy locked by 101-UI-SPEC §Web-share security banner copy — DO NOT
 * paraphrase:
 *   Heading:        "Web sharing this shell will expose arbitrary command execution."
 *   Body line 1:    "You are about to share '<sessionName>'. Anyone on your tailnet
 *                    who can reach the daemon will be able to type commands as your
 *                    user account."
 *   Body line 2:    "Read-only viewers cannot type, but commands you run remain
 *                    visible to them."
 *   Primary CTA:    "Enable web sharing"
 *   Confirming:     "Enabling…" (U+2026 ellipsis — single character)
 *   Secondary CTA:  "Cancel"
 *   Dismiss aria:   "Dismiss security warning"
 *
 * Focus moves to the Cancel button on mount (safe-action default per
 * QuitConfirmModal v3.0 Phase 85 precedent). Esc fires onCancel. Information
 * disclosure mitigation: sessionName is rendered as a React text node (auto-
 * escaped) inside the body line, never via dangerouslySetInnerHTML.
 */
export interface ShellWebShareBannerProps {
  sessionName: string
  onConfirm: () => void
  onCancel: () => void
  /**
   * Layout context. 'banner' (default) is the horizontal banner-stack strip
   * (StatusBar). 'block' stacks the body above a right-aligned actions row for
   * the narrower Hub Share modal, where the horizontal layout collapses the
   * text into a tall narrow column (Phase 150 SET-01 gap-closure, live UAT).
   */
  variant?: 'banner' | 'block'
}

export function ShellWebShareBanner({
  sessionName,
  onConfirm,
  onCancel,
  variant = 'banner',
}: ShellWebShareBannerProps): React.ReactElement {
  const cancelRef = useRef<HTMLButtonElement>(null)
  const [enabling, setEnabling] = useState(false)

  // Focus the safe action (Cancel) on mount — mirrors QuitConfirmModal pattern.
  useEffect(() => {
    cancelRef.current?.focus()
  }, [])

  // Esc dismisses the banner (treated as Cancel).
  // Focus gate (CR-02): only respond when keyboard focus is inside the banner
  // OR no other modal is open. This prevents the banner's Esc handler from
  // hijacking dismissal of a layered modal (kill-confirm, new-session modal,
  // find bar) that opens above the banner.
  const bannerRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      const banner = bannerRef.current
      if (!banner) return
      const active = document.activeElement
      // Focus inside the banner → handle.
      if (active && banner.contains(active)) {
        e.preventDefault()
        onCancel()
        return
      }
      // Focus elsewhere — check whether any other element with role="dialog"
      // or aria-modal="true" is on top. If so, let that owner handle Esc.
      const otherModal = document.querySelector(
        '[aria-modal="true"]:not([data-shell-web-share-banner]), [role="dialog"]:not([data-shell-web-share-banner])',
      )
      if (otherModal) return
      e.preventDefault()
      onCancel()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  function handleConfirm() {
    if (enabling) return
    setEnabling(true)
    onConfirm()
  }

  return (
    <div
      ref={bannerRef}
      className={`webgl-recovery-banner webgl-recovery-banner--shell-warning${variant === 'block' ? ' webgl-recovery-banner--block' : ''}`}
      role="alert"
      aria-live="assertive"
      aria-busy={enabling ? 'true' : undefined}
      data-shell-web-share-banner=""
    >
      <div className="webgl-recovery-banner__shell-body">
        <div className="webgl-recovery-banner__shell-heading">
          Web sharing this shell will expose arbitrary command execution.
        </div>
        <div className="webgl-recovery-banner__shell-text">
          You are about to share '{sessionName}'. Anyone on your tailnet who can reach the daemon will be able to type commands as your user account. Read-only viewers cannot type, but commands you run remain visible to them.
        </div>
      </div>
      <div className="webgl-recovery-banner__shell-actions">
        <button
          type="button"
          ref={cancelRef}
          disabled={enabling}
          onClick={onCancel}
          className="webgl-recovery-banner__shell-btn webgl-recovery-banner__shell-btn--secondary"
        >
          Cancel
        </button>
        <button
          type="button"
          disabled={enabling}
          onClick={handleConfirm}
          className="webgl-recovery-banner__shell-btn webgl-recovery-banner__shell-btn--primary-destructive"
        >
          {enabling ? 'Enabling…' : 'Enable web sharing'}
        </button>
        <button
          type="button"
          disabled={enabling}
          onClick={onCancel}
          aria-label="Dismiss security warning"
          className="webgl-recovery-banner__dismiss"
        >
          <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
        </button>
      </div>
    </div>
  )
}
