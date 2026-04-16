import React from 'react'
import { BrowserOpenURL } from '../wailsjs/wailsjs/runtime/runtime'

export interface UpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseURL: string
}

interface UpdateBannerProps {
  update: UpdateInfo
  onDismiss: () => void
  className?: string
}

export function UpdateBanner({ update, onDismiss, className }: UpdateBannerProps): React.ReactElement {
  return (
    <div
      className={`update-banner${className ? ' ' + className : ''}`}
      role="alert"
      aria-live="polite"
    >
      <span className="update-banner__message">
        Update available:{' '}
        <span className="update-banner__version">{update.currentVersion}</span>
        {' '}
        <span className="update-banner__arrow">&rarr;</span>
        {' '}
        <span className="update-banner__version">{update.latestVersion}</span>
      </span>
      <div className="update-banner__actions">
        <button
          type="button"
          className="update-banner__btn--download"
          onClick={() => BrowserOpenURL(update.releaseURL)}
        >
          Download Update
        </button>
        <button
          type="button"
          className="update-banner__btn--dismiss"
          aria-label="Dismiss update notification"
          onClick={onDismiss}
        >
          Dismiss
        </button>
      </div>
    </div>
  )
}
