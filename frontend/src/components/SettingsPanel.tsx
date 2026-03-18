import React, { useState } from 'react'
import { UpdateCLIPath } from '../wailsjs/go/main/App'
import type { DetectedCLI } from '../wailsjs/go/main/App'

interface SettingsPanelProps {
  isOpen: boolean
  onClose: () => void
  clis: DetectedCLI[]
}

/**
 * Modal settings panel for configuring custom CLI executable paths.
 * Lists all detected CLIs with an input field for path overrides.
 */
export function SettingsPanel({ isOpen, onClose, clis }: SettingsPanelProps): React.ReactElement | null {
  // Track custom path overrides keyed by CLI name.
  const [customPaths, setCustomPaths] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {}
    for (const cli of clis) {
      initial[cli.Name] = cli.Path
    }
    return initial
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isOpen) return null

  async function handleSave() {
    setSaving(true)
    setError(null)
    try {
      for (const cli of clis) {
        const path = customPaths[cli.Name] ?? ''
        if (path !== cli.Path && path.trim() !== '') {
          await UpdateCLIPath(cli.Name, path.trim())
        }
      }
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="settings-overlay" onClick={onClose}>
      <div className="settings-panel" onClick={(e) => e.stopPropagation()}>
        <div className="settings-panel__header">
          <h2>Settings</h2>
          <button className="settings-panel__close" onClick={onClose} aria-label="Close settings">
            ×
          </button>
        </div>

        <div className="settings-panel__body">
          <h3>CLI Paths</h3>
          {clis.length === 0 ? (
            <p className="settings-panel__empty">No CLIs detected. Install an AI coding CLI and restart the app.</p>
          ) : (
            <table className="settings-panel__table">
              <thead>
                <tr>
                  <th>CLI</th>
                  <th>Path</th>
                </tr>
              </thead>
              <tbody>
                {clis.map((cli) => (
                  <tr key={cli.Name}>
                    <td className="settings-panel__cli-name">{cli.Name}</td>
                    <td>
                      <input
                        className="settings-panel__path-input"
                        type="text"
                        value={customPaths[cli.Name] ?? cli.Path}
                        onChange={(e) =>
                          setCustomPaths((prev) => ({ ...prev, [cli.Name]: e.target.value }))
                        }
                        placeholder={cli.Path || `Path to ${cli.Name}`}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {error && <p className="settings-panel__error">{error}</p>}
        </div>

        <div className="settings-panel__footer">
          <button className="settings-panel__btn settings-panel__btn--cancel" onClick={onClose}>
            Cancel
          </button>
          <button
            className="settings-panel__btn settings-panel__btn--save"
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
