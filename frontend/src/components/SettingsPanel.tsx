import React, { useState, useEffect } from 'react'
import {
  UpdateCLIPath,
  SetWebPassword,
  IsWebPasswordSet,
  StartWebServer,
  StopWebServer,
  GetWebServerURL,
  IsWebServerRunning,
  HasCTDisclosure,
  AcknowledgeCTDisclosure,
} from '../wailsjs/go/main/App'
import type { DetectedCLI } from '../wailsjs/go/main/App'

interface SettingsPanelProps {
  isOpen: boolean
  onClose: () => void
  clis: DetectedCLI[]
}

/**
 * Modal settings panel for configuring custom CLI executable paths and web serving.
 * Lists all detected CLIs with an input field for path overrides, plus a Web Server
 * section for CT disclosure and server start/stop, and a Security section for
 * password setup.
 */
export function SettingsPanel({ isOpen, onClose, clis }: SettingsPanelProps): React.ReactElement | null {
  const [activeTab, setActiveTab] = useState<'cli-paths' | 'web-server' | 'security'>('cli-paths')
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

  // Web serving state
  const [webPassword, setWebPassword] = useState('')
  const [isPasswordSet, setIsPasswordSet] = useState(false)
  const [passwordSaving, setPasswordSaving] = useState(false)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [selectedPort, setSelectedPort] = useState<number>(7443)
  const [isServerRunning, setIsServerRunning] = useState(false)
  const [serverURL, setServerURL] = useState<string>('')
  const [serverError, setServerError] = useState<string | null>(null)
  const [serverLoading, setServerLoading] = useState(false)
  const [ctDisclosed, setCTDisclosed] = useState(false)
  const [ctError, setCTError] = useState<string | null>(null)

  // Load web serving state on panel open.
  useEffect(() => {
    if (!isOpen) return
    async function loadWebState() {
      try {
        const [pwSet, running, ctAck] = await Promise.all([
          IsWebPasswordSet(),
          IsWebServerRunning(),
          HasCTDisclosure(),
        ])
        setIsPasswordSet(pwSet)
        setIsServerRunning(running)
        setCTDisclosed(ctAck)

        if (running) {
          const url = await GetWebServerURL()
          setServerURL(url)
        }
      } catch (err) {
        console.error('[SettingsPanel] loadWebState:', err)
      }
    }
    void loadWebState()
  }, [isOpen])

  if (!isOpen) return null

  async function handleSaveCLIPaths() {
    setSaving(true)
    setError(null)
    try {
      for (const cli of clis) {
        const path = customPaths[cli.Name] ?? ''
        if (path !== cli.Path && path.trim() !== '') {
          await UpdateCLIPath(cli.Name, path.trim())
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  async function handleSetPassword() {
    if (!webPassword.trim()) {
      setPasswordError('Password cannot be empty')
      return
    }
    setPasswordSaving(true)
    setPasswordError(null)
    try {
      await SetWebPassword(webPassword.trim())
      setIsPasswordSet(true)
      setWebPassword('')
    } catch (err) {
      setPasswordError(err instanceof Error ? err.message : String(err))
    } finally {
      setPasswordSaving(false)
    }
  }

  async function handleAcknowledgeCT() {
    setCTError(null)
    try {
      await AcknowledgeCTDisclosure()
      setCTDisclosed(true)
    } catch (err) {
      setCTError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleToggleServer() {
    setServerError(null)
    setServerLoading(true)
    try {
      if (isServerRunning) {
        await StopWebServer()
        setIsServerRunning(false)
        setServerURL('')
      } else {
        await StartWebServer(selectedPort)
        const url = await GetWebServerURL()
        setServerURL(url)
        setIsServerRunning(true)
      }
    } catch (err) {
      setServerError(err instanceof Error ? err.message : String(err))
    } finally {
      setServerLoading(false)
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

        <div className="settings-panel__tabs" role="tablist">
          <button
            className={`settings-panel__tab-btn ${activeTab === 'cli-paths' ? 'settings-panel__tab-btn--active' : ''}`}
            onClick={() => setActiveTab('cli-paths')}
            role="tab"
            aria-selected={activeTab === 'cli-paths'}
          >
            CLI Paths
          </button>
          <button
            className={`settings-panel__tab-btn ${activeTab === 'web-server' ? 'settings-panel__tab-btn--active' : ''}`}
            onClick={() => setActiveTab('web-server')}
            role="tab"
            aria-selected={activeTab === 'web-server'}
          >
            Web Server
          </button>
          <button
            className={`settings-panel__tab-btn ${activeTab === 'security' ? 'settings-panel__tab-btn--active' : ''}`}
            onClick={() => setActiveTab('security')}
            role="tab"
            aria-selected={activeTab === 'security'}
          >
            Security
          </button>
        </div>

        <div className="settings-panel__body">
          {activeTab === 'cli-paths' && (
            <>
              {/* CLI Paths Section */}
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

              <div className="settings-panel__save-paths-row">
                <button
                  className="settings-panel__btn settings-panel__btn--save"
                  onClick={handleSaveCLIPaths}
                  disabled={saving}
                >
                  {saving ? 'Saving\u2026' : 'Save Paths'}
                </button>
              </div>
            </>
          )}

          {activeTab === 'web-server' && (
            <>
              <p className="settings-panel__description">
                Enable HTTPS access to terminal sessions from your Tailscale network.
              </p>

              {/* CT Disclosure Banner */}
              <div className={`ct-disclosure ${ctDisclosed ? 'ct-disclosure--acknowledged' : ''}`}>
                <label className="settings-panel__label">Certificate Transparency</label>
                {ctDisclosed ? (
                  <p className="ct-disclosure__text">
                    <span style={{ color: '#9ece6a' }}>&#10003;</span> Certificate Transparency acknowledged
                  </p>
                ) : (
                  <>
                    <p className="ct-disclosure__text">
                      When you start the web server, Tailscale will provision a Let&apos;s Encrypt TLS certificate
                      for your device&apos;s hostname (e.g., <code className="settings-panel__code">hostname.ts.net</code>).
                      This hostname will be permanently visible in public Certificate Transparency logs.
                      This is normal and expected for any Let&apos;s Encrypt certificate.
                    </p>
                    <button
                      className="ct-disclosure__btn settings-panel__btn settings-panel__btn--save"
                      onClick={handleAcknowledgeCT}
                    >
                      I Understand
                    </button>
                    {ctError && <p className="settings-panel__error">{ctError}</p>}
                  </>
                )}
              </div>

              {/* Port */}
              <div className="settings-panel__field-group">
                <label className="settings-panel__label">Port</label>
                <input
                  className="settings-panel__path-input settings-panel__port-input"
                  type="number"
                  value={selectedPort}
                  onChange={(e) => setSelectedPort(Number(e.target.value))}
                  disabled={isServerRunning}
                  min={1}
                  max={65535}
                />
              </div>

              {/* Start/Stop Server */}
              <div className="settings-panel__field-group">
                <button
                  className={`settings-panel__btn ${isServerRunning ? 'settings-panel__btn--cancel' : 'settings-panel__btn--save'}`}
                  onClick={handleToggleServer}
                  disabled={serverLoading || (!isServerRunning && (!isPasswordSet || !ctDisclosed))}
                  title={
                    !ctDisclosed && !isServerRunning
                      ? 'Acknowledge the Certificate Transparency disclosure first'
                      : !isPasswordSet && !isServerRunning
                        ? 'Set a password in the Security tab first'
                        : undefined
                  }
                >
                  {serverLoading
                    ? (isServerRunning ? 'Stopping\u2026' : 'Starting\u2026')
                    : (isServerRunning ? 'Stop Web Server' : 'Start Web Server')}
                </button>
                {serverError && <p className="settings-panel__error">{serverError}</p>}
                {isServerRunning && serverURL && (
                  <p className="settings-panel__url">
                    Server running at: <a href={serverURL} target="_blank" rel="noreferrer">{serverURL}</a>
                  </p>
                )}
              </div>
            </>
          )}

          {activeTab === 'security' && (
            <>
              {/* Password Setup */}
              <div className="settings-panel__field-group">
                <label className="settings-panel__label">
                  Dashboard Password
                  {isPasswordSet && (
                    <span className="settings-panel__check" title="Password is set"> ✓</span>
                  )}
                </label>
                <div className="settings-panel__row">
                  <input
                    className="settings-panel__path-input"
                    type="password"
                    value={webPassword}
                    onChange={(e) => setWebPassword(e.target.value)}
                    placeholder={isPasswordSet ? 'Change password\u2026' : 'Set a password to enable web serving'}
                    onKeyDown={(e) => { if (e.key === 'Enter') void handleSetPassword() }}
                  />
                  <button
                    className="settings-panel__btn settings-panel__btn--save"
                    onClick={handleSetPassword}
                    disabled={passwordSaving || !webPassword.trim()}
                  >
                    {passwordSaving ? 'Saving\u2026' : 'Set Password'}
                  </button>
                </div>
                {passwordError && <p className="settings-panel__error">{passwordError}</p>}
              </div>
            </>
          )}
        </div>

        <div className="settings-panel__footer">
          <button className="settings-panel__btn settings-panel__btn--cancel" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
