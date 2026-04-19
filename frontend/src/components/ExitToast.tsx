import React from 'react'
import { XMarkIcon } from '@heroicons/react/20/solid'

export interface ExitState {
  sessionId: string
  sessionName: string
  cli: string
  exitCode: number
  duration: number
  finalStatus: string
  countdown: number    // seconds remaining (5 -> 0), -1 if no countdown
  cancelled: boolean   // true = user clicked "Keep Open"
}

interface ExitToastProps {
  exits: Record<string, ExitState>
  onKeepOpen: (sessionId: string) => void
  onDismiss: (sessionId: string) => void
}

export function ExitToast({ exits, onKeepOpen, onDismiss }: ExitToastProps): React.ReactElement | null {
  const entries = Object.values(exits)
  if (entries.length === 0) return null

  return (
    <div className="exit-toast">
      {entries.map((exit) => (
        <div
          key={exit.sessionId}
          className={`exit-toast__item ${exit.exitCode === 0 ? 'exit-toast__item--clean' : 'exit-toast__item--error'}`}
          role="alert"
          aria-live="polite"
        >
          <div className="exit-toast__header">
            <span className="exit-toast__session-name">{exit.sessionName}</span>
            <button
              type="button"
              className="exit-toast__dismiss"
              aria-label={`Dismiss exit notification for ${exit.sessionName}`}
              onClick={() => onDismiss(exit.sessionId)}
            >
              <XMarkIcon style={{ width: 16, height: 16 }} />
            </button>
          </div>
          <span className="exit-toast__meta">
            {exit.cli} &middot; {exit.exitCode === 0 ? 'exited' : 'exited with error'} &middot; {exit.finalStatus}
          </span>
          <span className="exit-toast__codes">
            <span className={exit.exitCode === 0 ? 'exit-toast__exit-code--ok' : 'exit-toast__exit-code--err'}>
              Exit code: {exit.exitCode}
            </span>
            {' \u00b7 '}Duration: {exit.duration}s
          </span>
          {exit.exitCode === 0 && !exit.cancelled && exit.countdown > 0 && (
            <div className="exit-toast__actions">
              <span className="exit-toast__countdown">Closing in {exit.countdown}s</span>
              <button
                type="button"
                className="exit-toast__keep-open"
                onClick={() => onKeepOpen(exit.sessionId)}
              >
                Keep Open
              </button>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
