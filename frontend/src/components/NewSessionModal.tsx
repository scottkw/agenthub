import { useEffect, useState } from 'react'
import { GetShellPath, OpenDirectoryDialog } from '../wailsjs/go/main/App'
import type { daemon } from '../wailsjs/go/models'

const LAST_DIR_KEY = 'agenthub:lastWorkDir'
const ARGS_KEY = (cli: string) => `agenthub:args:${cli}`
const SHELL_ARGS_KEY = (name: string) => `agenthub:args:shell:${name}`
const SHELL_ARGS_PLACEHOLDER = 'Arguments are not passed to shell sessions'

interface DetectedCLI {
  Name: string
  Path: string
  DisplayName?: string
}

export interface NewSessionModalProps {
  isOpen: boolean
  clis: DetectedCLI[]
  /**
   * Phase 101-02 (SHELL-01 GUI half) — shell list formerly used to render
   * per-binary rows. Kept for backwards compat with App.tsx call site
   * (App.tsx:1217 passes `shells={detectedShells}`); the modal no longer
   * renders multiple shell rows. Pending removal in a future cleanup (SHELL-10).
   */
  shells?: daemon.DetectedShell[]
  /**
   * Phase 101-02 — formerly triggered a loading skeleton when true AND shells
   * was empty. Kept for backwards compat with App.tsx call site; no skeleton
   * is rendered in the collapsed single-row design. Pending removal (SHELL-10).
   */
  shellsLoading?: boolean
  onConfirm: (cli: string, workDir: string, args: string[]) => void
  onClose: () => void
}

export function NewSessionModal({
  isOpen,
  clis,
  // shells / shellsLoading are kept as accepted (but unused) props for
  // backwards compat with the App.tsx call site. They will be removed once
  // App.tsx is updated to stop passing them in a future cleanup.
  shells: _shells = [],
  shellsLoading: _shellsLoading = false,
  onConfirm,
  onClose,
}: NewSessionModalProps) {
  const [selectedAgent, setSelectedAgent] = useState(clis[0]?.Name ?? '')
  const [selectedDir, setSelectedDir] = useState(() => localStorage.getItem(LAST_DIR_KEY) ?? '')
  const [browseLoading, setBrowseLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [argsText, setArgsText] = useState(() =>
    localStorage.getItem(ARGS_KEY(clis[0]?.Name ?? '')) ?? ''
  )

  // SHELL-10: Resolved shell binary path shown in the single static Shell row.
  // Fetched from daemon via GetShellPath() on every modal open so that changes
  // made in Settings → Paths are reflected immediately without a full reload.
  const [resolvedShellPath, setResolvedShellPath] = useState('')

  useEffect(() => {
    if (!isOpen) return
    GetShellPath()
      .then(setResolvedShellPath)
      .catch(() => setResolvedShellPath(''))
  }, [isOpen])

  // SHELL-10: Shell selection is now bare 'shell' — no "shell:" prefix scheme.
  const isShellSelected = selectedAgent === 'shell'

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

  function handleSelectCLI(name: string) {
    setSelectedAgent(name)
    setArgsText(localStorage.getItem(ARGS_KEY(name)) ?? '')
  }

  function handleSelectShell(name: string) {
    // SHELL-10: agent id is bare 'shell' — no prefix.
    setSelectedAgent(name)
    setArgsText(localStorage.getItem(SHELL_ARGS_KEY(name)) ?? '')
  }

  function handleClearArgs() {
    setArgsText('')
    if (!isShellSelected) {
      localStorage.removeItem(ARGS_KEY(selectedAgent))
    }
  }

  function handleConfirm() {
    setCreating(true)
    if (isShellSelected) {
      // SHELL-10: Agent id is already bare 'shell' — pass directly.
      onConfirm('shell', selectedDir, [])
      return
    }
    if (argsText.trim()) {
      localStorage.setItem(ARGS_KEY(selectedAgent), argsText)
    } else {
      localStorage.removeItem(ARGS_KEY(selectedAgent))
    }
    const args = argsText.trim().split(/\s+/).filter(Boolean)
    onConfirm(selectedAgent, selectedDir, args)
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
              {clis.map((cli) => {
                const selected = selectedAgent === cli.Name
                return (
                  <button
                    key={cli.Name}
                    className={`new-session-modal__agent-btn${selected ? ' new-session-modal__agent-btn--selected' : ''}`}
                    aria-pressed={selected}
                    onClick={() => handleSelectCLI(cli.Name)}
                  >
                    {cli.DisplayName || cli.Name}
                  </button>
                )
              })}
              {/* SHELL-10: Single static Shell row — replaces the Phase 101 sortedShells.map loop.
                  Agent id is bare 'shell'. Detail line shows daemon-resolved path from GetShellPath().
                  Selected state uses --selected-shell (cyan #89ddff) per UI-SPEC §2. */}
              <button
                className={[
                  'new-session-modal__agent-btn',
                  'new-session-modal__agent-btn--shell',
                  selectedAgent === 'shell' ? 'new-session-modal__agent-btn--selected-shell' : '',
                ].filter(Boolean).join(' ')}
                aria-pressed={selectedAgent === 'shell'}
                onClick={() => handleSelectShell('shell')}
              >
                <span>Shell</span>
                <span className="new-session-modal__agent-btn__detail">{resolvedShellPath}</span>
              </button>
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
                {browseLoading ? 'Browsing…' : 'Browse…'}
              </button>
            </div>
          </div>
          <div className="new-session-modal__section">
            <label className="new-session-modal__section-label">Extra Arguments</label>
            <div className="new-session-modal__args-row">
              <input
                className="new-session-modal__args-input"
                type="text"
                value={isShellSelected ? '' : argsText}
                onChange={(e) => setArgsText(e.target.value)}
                placeholder={isShellSelected ? SHELL_ARGS_PLACEHOLDER : 'e.g. --model claude-opus-4-5'}
                disabled={isShellSelected}
              />
              {argsText && !isShellSelected && (
                <button
                  className="new-session-modal__args-clear"
                  onClick={handleClearArgs}
                  aria-label="Clear arguments"
                >
                  Clear Args
                </button>
              )}
            </div>
          </div>
        </div>
        <div className="new-session-modal__footer">
          <button className="new-session-modal__btn--close" onClick={onClose}>Close</button>
          <button
            className="new-session-modal__btn--create"
            onClick={handleConfirm}
            disabled={!selectedAgent || creating}
          >
            {creating ? 'Creating…' : 'Create Session'}
          </button>
        </div>
      </div>
    </div>
  )
}
