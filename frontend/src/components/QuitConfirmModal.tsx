import React, { useEffect, useRef, useState } from 'react'

interface QuitConfirmModalProps {
  isOpen: boolean
  sessions: Array<{ id: string; name: string; status: string }>
  onQuitGUI: () => void
  onQuitAll: () => void
  onCancel: () => void
}

function dotColor(status: string): string {
  switch (status) {
    case 'running': return '#9ece6a'
    case 'errored': return '#f7768e'
    case 'idle': return '#e0af68'
    case 'waiting': return '#7aa2f7'
    default: return '#9aa5ce'
  }
}

function QuitConfirmModal({ isOpen, sessions, onQuitGUI, onQuitAll, onCancel }: QuitConfirmModalProps): React.ReactElement | null {
  const [acting, setActing] = useState(false)
  const cancelBtnRef = useRef<HTMLButtonElement>(null)

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  // Focus "Keep Running" button on open
  useEffect(() => {
    if (isOpen) cancelBtnRef.current?.focus()
  }, [isOpen])

  // Reset acting state when modal reopens
  useEffect(() => {
    if (isOpen) setActing(false)
  }, [isOpen])

  if (!isOpen) return null

  const visible = sessions.slice(0, 5)
  const overflow = sessions.length - 5

  const subtitleText =
    sessions.length === 0
      ? 'No active sessions.'
      : sessions.length === 1
        ? 'You have 1 active session running.'
        : `You have ${sessions.length} active sessions running.`

  return (
    <div className="quit-modal-overlay" onClick={onCancel}>
      <div
        className="quit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="quit-modal-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="quit-modal__header">
          <h2 className="quit-modal__title" id="quit-modal-title">Quit AgentHub?</h2>
          <button className="quit-modal__close" aria-label="Close" onClick={onCancel}>&times;</button>
        </div>
        <div className="quit-modal__body">
          <p className="quit-modal__subtitle">{subtitleText}</p>
          {sessions.length > 0 && (
            <div className="quit-modal__session-list">
              {visible.map((s) => (
                <div className="quit-modal__session-item" key={s.id}>
                  <span
                    className="quit-modal__session-dot"
                    aria-hidden="true"
                    style={{ backgroundColor: dotColor(s.status) }}
                  />
                  <span className="quit-modal__session-name">{s.name}</span>
                  <span className="quit-modal__session-status">({s.status})</span>
                </div>
              ))}
              {overflow > 0 && (
                <span className="quit-modal__overflow">...and {overflow} more</span>
              )}
            </div>
          )}
          {sessions.length === 0 && (
            <p className="quit-modal__no-sessions">No active sessions.</p>
          )}
        </div>
        <div className="quit-modal__footer">
          <button
            className="quit-modal__btn--cancel"
            ref={cancelBtnRef}
            disabled={acting}
            onClick={onCancel}
          >
            Keep Running
          </button>
          <button
            className="quit-modal__btn--quit-gui"
            disabled={acting}
            onClick={() => { setActing(true); onQuitGUI() }}
          >
            Quit GUI Only
          </button>
          <button
            className="quit-modal__btn--quit-all"
            disabled={acting}
            onClick={() => { setActing(true); onQuitAll() }}
          >
            Quit Everything
          </button>
        </div>
      </div>
    </div>
  )
}

export { QuitConfirmModal }
