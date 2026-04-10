import React, { useState, useEffect } from 'react'
import {
  UpdateCLIPath,
  StartWebServer,
  StopWebServer,
  GetWebServerURL,
  IsWebServerRunning,
  HasCTDisclosure,
  AcknowledgeCTDisclosure,
  GetLocalNetworkPassword,
} from '../wailsjs/go/main/App'
import type { DetectedCLI } from '../wailsjs/go/main/App'

interface SettingsTabProps {
  clis: DetectedCLI[]
  tailscaleHealth: {
    installed: boolean
    connected: boolean
    hasCerts: boolean
    ip: string
    domain: string
  } | null
  webServerMode?: 'tailscale' | 'local' | null
  onWebServerStateChange: () => Promise<void>
}

/**
 * Inline settings tab for configuring custom CLI executable paths and web serving.
 * Lists all detected CLIs with an input field for path overrides, plus a Web Server
 * section for CT disclosure and server start/stop.
 * Renders as a sidebar tab — no modal shell.
 */
export function SettingsTab({ clis, tailscaleHealth, webServerMode, onWebServerStateChange }: SettingsTabProps): React.ReactElement {
  const [activeTab, setActiveTab] = useState<'cli-paths' | 'web-server'>('cli-paths')
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
  const [selectedPort, setSelectedPort] = useState<number>(7443)
  const [isServerRunning, setIsServerRunning] = useState(false)
  const [serverURL, setServerURL] = useState<string>('')
  const [serverError, setServerError] = useState<string | null>(null)
  const [serverLoading, setServerLoading] = useState(false)
  const [ctDisclosed, setCTDisclosed] = useState(false)
  const [ctError, setCTError] = useState<string | null>(null)
  // Local network password display
  const [localPassword, setLocalPassword] = useState('')
  const [copied, setCopied] = useState(false)

  // Load web serving state on mount.
  useEffect(() => {
    async function loadWebState() {
      try {
        const [running, ctAck] = await Promise.all([
          IsWebServerRunning(),
          HasCTDisclosure(),
        ])
        setIsServerRunning(running)
        setCTDisclosed(ctAck)

        if (running) {
          const url = await GetWebServerURL()
          setServerURL(url)
        }
      } catch (err) {
        console.error('[SettingsTab] loadWebState:', err)
      }
    }
    void loadWebState()
  }, [])

  // Fetch LAN password when in local mode and server is running.
  useEffect(() => {
    if (webServerMode === 'local' && isServerRunning) {
      GetLocalNetworkPassword().then(setLocalPassword).catch(() => setLocalPassword(''))
    } else {
      setLocalPassword('')
    }
  }, [webServerMode, isServerRunning])

  async function handleCopyPassword() {
    if (!localPassword) return
    await navigator.clipboard.writeText(localPassword)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  function tailscaleStatusClass(h: SettingsTabProps['tailscaleHealth']): string {
    if (!h) return ''
    if (h.installed && h.connected) return 'ok'
    if (h.installed) return 'warn'
    return 'error'
  }

  function tailscaleStatusText(h: SettingsTabProps['tailscaleHealth']): string {
    if (!h) return 'Checking\u2026'
    if (h.installed && h.connected) return 'Connected'
    if (h.installed) return 'Not Connected'
    return 'Not Installed'
  }

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
      await onWebServerStateChange()
    } catch (err) {
      setServerError(err instanceof Error ? err.message : String(err))
    } finally {
      setServerLoading(false)
    }
  }

  return (
    <div className="settings-tab">
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

            {/* Tailscale Status Indicator */}
            <div className="settings-panel__field-group">
              <label className="settings-panel__label">Tailscale Status</label>
              <div className="ts-status">
                {tailscaleHealth && (
                  <span className={`ts-status__dot ts-status__dot--${tailscaleStatusClass(tailscaleHealth)}`} />
                )}
                <span className="ts-status__text">{tailscaleStatusText(tailscaleHealth)}</span>
              </div>
            </div>

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
                disabled={serverLoading || (!isServerRunning && !ctDisclosed)}
                title={
                  !ctDisclosed && !isServerRunning
                    ? 'Acknowledge the Certificate Transparency disclosure first'
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

            {/* LAN Password — only shown in local network mode */}
            {webServerMode === 'local' && isServerRunning && (
              <>
                <div className="settings-web-server__mode-indicator">
                  Web server mode: Local network (self-signed TLS)
                </div>
                <div className="settings-web-server__password-label">
                  LAN Access Credentials
                </div>
                <div className="settings-web-server__credential-hint">
                  Username: <span className="settings-web-server__credential-value">leave blank or enter anything</span>
                </div>
                <div className="settings-web-server__password-label">
                  Password
                </div>
                <div
                  className="settings-web-server__password-field"
                  onClick={() => void handleCopyPassword()}
                  title="Click to copy password"
                >
                  <span>{localPassword || 'Loading\u2026'}</span>
                  <span className={`settings-web-server__copy-hint${copied ? ' settings-web-server__copy-hint--copied' : ''}`}>
                    {copied ? 'Copied!' : '(click to copy)'}
                  </span>
                </div>
              </>
            )}
          </>
        )}

      </div>
    </div>
  )
}
