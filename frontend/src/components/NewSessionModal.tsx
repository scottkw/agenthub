import { useState } from 'react'
import { OpenDirectoryDialog } from '../wailsjs/go/main/App'

const LAST_DIR_KEY = 'agenthub:lastWorkDir'

interface DetectedCLI {
  Name: string
  Path: string
  DisplayName?: string
}

export interface NewSessionModalProps {
  isOpen: boolean
  clis: DetectedCLI[]
  onConfirm: (cli: string, workDir: string) => void
  onClose: () => void
}

export function NewSessionModal({ isOpen, clis, onConfirm, onClose }: NewSessionModalProps) {
  const [selectedCLI, setSelectedCLI] = useState(clis[0]?.Name ?? '')
  const [selectedDir, setSelectedDir] = useState(() => localStorage.getItem(LAST_DIR_KEY) ?? '')
  const [browseLoading, setBrowseLoading] = useState(false)
  const [creating, setCreating] = useState(false)

  if (!isOpen) return null

  async function handleBrowse() {
    setBrowseLoading(true)
    try {
      const path = await OpenDirectoryDialog(selectedDir)
      if (path !== '') {
        setSelectedDir(path)
        localStorage.setItem(LAST_DIR_KEY, path)
      }
    } finally {
      setBrowseLoading(false)
    }
  }

  function handleConfirm() {
    setCreating(true)
    onConfirm(selectedCLI, selectedDir)
  }

  return (
    <div className="new-session-overlay" onClick={onClose}>
      <div className="new-session-modal" onClick={(e) => e.stopPropagation()}>
        <div className="new-session-modal__header">
          <h2>New Session</h2>
          <button className="new-session-modal__close" aria-label="Close" onClick={onClose}>&times;</button>
        </div>
        <div className="new-session-modal__body">
          <div className="new-session-modal__section">
            <label className="new-session-modal__section-label">Select Agent</label>
            <div className="new-session-modal__agent-list">
              {clis.map((cli) => (
                <button
                  key={cli.Name}
                  className={`new-session-modal__agent-btn${selectedCLI === cli.Name ? ' new-session-modal__agent-btn--selected' : ''}`}
                  onClick={() => setSelectedCLI(cli.Name)}
                >
                  {cli.DisplayName || cli.Name}
                </button>
              ))}
            </div>
          </div>
          <div className="new-session-modal__section">
            <label className="new-session-modal__section-label">Working Directory</label>
            <div className="new-session-modal__folder-row">
              <div className={`new-session-modal__folder-display${selectedDir ? ' new-session-modal__folder-display--active' : ''}`}>
                {selectedDir || 'Home directory (default)'}
              </div>
              <button
                className="new-session-modal__browse-btn"
                onClick={() => void handleBrowse()}
                disabled={browseLoading}
              >
                {browseLoading ? 'Browsing\u2026' : 'Browse\u2026'}
              </button>
            </div>
          </div>
        </div>
        <div className="new-session-modal__footer">
          <button className="new-session-modal__btn--close" onClick={onClose}>Close</button>
          <button
            className="new-session-modal__btn--create"
            onClick={handleConfirm}
            disabled={!selectedCLI || creating}
          >
            {creating ? 'Creating\u2026' : 'Create Session'}
          </button>
        </div>
      </div>
    </div>
  )
}
