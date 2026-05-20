// Phase 120-04 Task 1 — network-error state with Retry.
//
// Used in two distinct contexts:
//   - scope='directory' — /api/files/list rejected with a non-permission error
//                         (network, 5xx, timeout). Replaces the list pane body.
//   - scope='preview'   — /api/files/read rejected on a specific file. Renders
//                         inside the preview pane.
//
// Headings differ by scope (UI-SPEC §"Error copy" — "Could not load directory"
// vs "Could not read this file") but body + action are identical.
//
// Retry is owned by the caller (FileBrowserTab passes a closure that either
// re-issues the directory fetch or re-issues the file head/read).

import React from 'react'

export interface NetworkErrorStateProps {
  scope: 'directory' | 'preview'
  onRetry: () => void
}

export function NetworkErrorState({
  scope,
  onRetry,
}: NetworkErrorStateProps): React.ReactElement {
  const heading =
    scope === 'directory' ? 'Could not load directory' : 'Could not read this file'
  return (
    <div
      className={`file-browser__error file-browser__error--${scope}`}
      data-testid="file-browser-network-error"
      role="alert"
      aria-live="assertive"
    >
      <h3 className="file-browser__error-heading">{heading}</h3>
      <p className="file-browser__error-body">
        Network error. Check your connection and try again.
      </p>
      <button
        type="button"
        className="file-browser__btn file-browser__btn--secondary"
        data-testid="file-browser-network-error-retry"
        onClick={onRetry}
      >
        Retry
      </button>
    </div>
  )
}
