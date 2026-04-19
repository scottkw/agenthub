import React from 'react'

interface ExitCountdownBannerProps {
  countdown: number
  onKeepOpen: () => void
}

export function ExitCountdownBanner({ countdown, onKeepOpen }: ExitCountdownBannerProps): React.ReactElement {
  return (
    <div className="exit-countdown-banner" role="alert" aria-live="polite">
      <span className="exit-countdown-banner__message">
        Agent exited cleanly. Tab closes in{' '}
        <span className="exit-countdown-banner__countdown">{countdown}s</span>.
      </span>
      <button
        type="button"
        className="exit-countdown-banner__keep-open"
        onClick={onKeepOpen}
      >
        Keep Open
      </button>
    </div>
  )
}
