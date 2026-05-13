import { useMemo, useState } from 'react'
import { OpenDirectoryDialog } from '../wailsjs/go/main/App'
import type { daemon } from '../wailsjs/go/models'

const LAST_DIR_KEY = 'agenthub:lastWorkDir'
const ARGS_KEY = (cli: string) => `agenthub:args:${cli}`
const SHELL_ARGS_KEY = (name: string) => `agenthub:args:shell:${name}`
const SHELL_PREFIX = 'shell:'
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
   * Phase 101-02 (SHELL-01 GUI half) — shell rows rendered AFTER the AI CLI
   * list. Order: system default first, then server-provided order. Empty
   * array yields no shell rows AND no loading skeleton (silent absence per
   * UI-SPEC §Edge Cases).
   */
  shells?: daemon.DetectedShell[]
  /**
   * Phase 101-02 — when true AND `shells` is empty, a loading skeleton row
   * renders below the AI CLI list with the locked text "Loading shells…".
   */
  shellsLoading?: boolean
  onConfirm: (cli: string, workDir: string, args: string[]) => void
  onClose: () => void
}

export function NewSessionModal({
  isOpen,
  clis,
  shells = [],
  shellsLoading = false,
  onConfirm,
  onClose,
}: NewSessionModalProps) {
  // Selection state uses a prefix scheme: AI CLIs stored as plain name (e.g.
  // "claude"), shells stored as "shell:NAME" (e.g. "shell:bash"). When sending
  // to the daemon we strip the "shell:" prefix and forward bare "bash"/"zsh"/
  // "shell" — matching the daemon's cli field convention.
  const [selectedAgent, setSelectedAgent] = useState(clis[0]?.Name ?? '')
  const [selectedDir, setSelectedDir] = useState(() => localStorage.getItem(LAST_DIR_KEY) ?? '')
  const [browseLoading, setBrowseLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [argsText, setArgsText] = useState(() =>
    localStorage.getItem(ARGS_KEY(clis[0]?.Name ?? '')) ?? ''
  )

  // Sort shells client-side: name === "shell" (system default) first, then the
  // server-provided order.
  const sortedShells = useMemo(() => {
    const out = shells.slice()
    out.sort((a, b) => {
      if (a.name === 'shell' && b.name !== 'shell') return -1
      if (b.name === 'shell' && a.name !== 'shell') return 1
      return 0
    })
    return out
  }, [shells])

  const isShellSelected = selectedAgent.startsWith(SHELL_PREFIX)

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
    setSelectedAgent(SHELL_PREFIX + name)
    // Shell args memory is read from the shell-prefixed namespace. Since the
    // field is disabled when a shell is selected, the value is informational
    // only — preserved across re-selection but not editable in 101-02.
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
      // Shells: pass bare shell name to the daemon; args are intentionally
      // dropped (Phase 100 RESEARCH Anti-Pattern + Assumption A6).
      const shellName = selectedAgent.slice(SHELL_PREFIX.length)
      onConfirm(shellName, selectedDir, [])
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
              {/*
                Phase 101-02 (SHELL-01 GUI half) — shell rows AFTER AI CLI list.
                Each row has the locked "Shell — DISPLAYNAME" prefix (em-dash
                U+2014, NOT a colon) plus a mono detail line showing the
                resolved path. Selected shell uses --selected-shell (cyan
                #89ddff) instead of --selected (blue #7aa2f7).
              */}
              {sortedShells.map((s) => {
                const key = SHELL_PREFIX + s.name
                const selected = selectedAgent === key
                const cls = [
                  'new-session-modal__agent-btn',
                  'new-session-modal__agent-btn--shell',
                  selected ? 'new-session-modal__agent-btn--selected-shell' : '',
                ].filter(Boolean).join(' ')
                return (
                  <button
                    key={key}
                    className={cls}
                    aria-pressed={selected}
                    onClick={() => handleSelectShell(s.name)}
                  >
                    <span>Shell — {s.displayName}</span>
                    <span className="new-session-modal__agent-btn__detail">{s.path}</span>
                  </button>
                )
              })}
              {shellsLoading && sortedShells.length === 0 && (
                <div
                  className="new-session-modal__agent-btn new-session-modal__agent-btn--loading"
                  aria-busy="true"
                >
                  Loading shells…
                </div>
              )}
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
