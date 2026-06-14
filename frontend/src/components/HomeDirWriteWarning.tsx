import React from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

/**
 * HomeDirWriteWarning — Phase 124 CAP-06 GUI Surface 3.
 *
 * Colorblind-safe home-directory write warning banner. Shown in the GUI when
 * the owner has enabled file writes for a session whose cwd is $HOME
 * (server-side `homeDir: true` signal from IssueCapabilitiesResponse).
 *
 * Verbatim copy from 124-UI-SPEC §"GUI Surface 3". DO NOT paraphrase.
 *
 * Colorblind contract (RELEASE-BLOCKING):
 *   - ⚠ WARNING SIGN glyph is present alongside the amber left-border.
 *   - The word "Warning:" appears in literal text.
 *   - Color alone is NEVER the sole carrier of the warning meaning.
 *
 * Reuses `webgl-recovery-banner` base structure with a new BEM modifier
 * `--home-write-warning` (amber `#f59e0b` left-border, matching
 * `local-network-banner` — no new hex). Standing caution: NOT timer-dismissed;
 * re-shows on re-enable. Dismissed per-session-per-enable.
 */
export interface HomeDirWriteWarningProps {
  onDismiss: () => void
  className?: string
}

export function HomeDirWriteWarning({
  onDismiss,
  className,
}: HomeDirWriteWarningProps): React.ReactElement {
  const cls = [
    'webgl-recovery-banner',
    'webgl-recovery-banner--home-write-warning',
    className,
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <div className={cls} role="status" aria-live="polite">
      <div className="webgl-recovery-banner__message">
        <div className="webgl-recovery-banner__home-write-heading">
          <span className="local-network-banner__icon" aria-hidden="true">
            ⚠
          </span>{' '}
          <span>Warning: writes can affect your home directory</span>
        </div>
        <div className="webgl-recovery-banner__home-write-body">
          {
            "This session's working directory is your home folder. File writes here can modify dotfiles, SSH keys, and shell config (~/.zshrc, ~/.ssh, ~/.claude). Protected system files are always blocked."
          }
        </div>
      </div>
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
