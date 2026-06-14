// Phase 125-03 Task 2 — three-state save indicator.
//
// Displays the current save state inline in the editor header.
// Three states (EDIT-06 + UI-SPEC §Color colorblind contract):
//   idle    — renders nothing (absence = clean/no-pending-save)
//   saving  — ArrowPathIcon + "Saving…" (amber reinforcement color, not the signal)
//   saved   — CheckCircleIcon + "Saved" (~1.5s transient before parent resets to idle)
//
// Every state carries BOTH an icon AND literal text — color is decoration only
// (RELEASE-BLOCKING colorblind contract: user is colorblind).
//
// Wrapped in role="status" aria-live="polite" so state transitions are announced
// to assistive technology without interrupting (UI-SPEC §2, PATTERNS §SaveIndicator).

import React from 'react'
import {
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline'
import type { SaveState } from '../../lib/useFilesWrite'

export interface SaveIndicatorProps {
  /** Current save state from useFilesWrite. */
  saveState: SaveState
  /** True when a non-412, non-success save error is present. */
  hasError?: boolean
}

export function SaveIndicator({ saveState, hasError = false }: SaveIndicatorProps): React.ReactElement {
  return (
    <span
      role="status"
      aria-live="polite"
      className="file-browser__save-indicator"
      aria-atomic="true"
    >
      {saveState === 'saving' && (
        <>
          {/* Saving: spinner icon + literal text — both carry meaning (colorblind contract) */}
          <ArrowPathIcon
            width={14}
            height={14}
            aria-hidden="true"
            className="file-browser__save-indicator-icon file-browser__save-indicator-icon--saving"
          />
          <span className="file-browser__save-indicator-text">Saving…</span>
        </>
      )}
      {saveState === 'saved' && !hasError && (
        <>
          {/* Saved: check icon + literal text — green reinforcement color is decoration */}
          <CheckCircleIcon
            width={14}
            height={14}
            aria-hidden="true"
            className="file-browser__save-indicator-icon file-browser__save-indicator-icon--saved"
          />
          <span className="file-browser__save-indicator-text">Saved</span>
        </>
      )}
      {saveState === 'idle' && hasError && (
        <>
          {/* Error: warning icon — literal error text is in EditorErrorBar (non-takeover) */}
          <ExclamationTriangleIcon
            width={14}
            height={14}
            aria-hidden="true"
            className="file-browser__save-indicator-icon file-browser__save-indicator-icon--error"
          />
        </>
      )}
    </span>
  )
}
