import React, { useEffect, useState, useCallback } from 'react'
import { TabBar, type Tab } from './components/TabBar'
import { TerminalPanel } from './components/TerminalPanel'
import { SettingsPanel } from './components/SettingsPanel'
import {
  CreateSession,
  ListSessions,
  KillSession,
  RenameSession,
  DetectCLIs,
  GetRelayPort,
  ToggleWebServing,
  GenerateSessionToken,
  GetWebServerURL,
  IsWebServerRunning,
} from './wailsjs/go/main/App'
import type { DetectedCLI } from './wailsjs/go/main/App'

/**
 * App is the root component — it owns all tab state and wires
 * the Wails-generated TypeScript bindings to the child components.
 */
function App(): React.ReactElement {
  const [tabs, setTabs] = useState<Tab[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [relayPort, setRelayPort] = useState<number | null>(null)
  const [showSettings, setShowSettings] = useState(false)
  const [detectedCLIs, setDetectedCLIs] = useState<DetectedCLI[]>([])
  const [tabCounter, setTabCounter] = useState(1)
  // Track if CLI picker dropdown is open (multiple CLIs)
  const [showCLIPicker, setShowCLIPicker] = useState(false)
  // Track web serving state per session: sessionId -> enabled
  const [webEnabled, setWebEnabled] = useState<Record<string, boolean>>({})
  // Track per-session web URLs (populated when web serving is enabled)
  const [sessionURLs, setSessionURLs] = useState<Record<string, string>>({})
  // Track web server running state
  const [webServerRunning, setWebServerRunning] = useState(false)

  // On mount: get relay port, detect CLIs, restore any existing sessions.
  useEffect(() => {
    async function init() {
      try {
        const [port, clis, sessions, running] = await Promise.all([
          GetRelayPort(),
          DetectCLIs(),
          ListSessions(),
          IsWebServerRunning(),
        ])
        setRelayPort(port)
        setDetectedCLIs(clis)
        setWebServerRunning(running)

        // Restore existing sessions as tabs (SESS-02 reattachment after window re-show).
        if (sessions.length > 0) {
          const restoredTabs: Tab[] = sessions.map((s) => ({
            id: s.id,
            name: s.name || s.cli,
            sessionId: s.id,
            cli: s.cli,
          }))
          setTabs(restoredTabs)
          setActiveId(restoredTabs[0].id)
        }
      } catch (err) {
        console.error('[App] init failed:', err)
      }
    }
    void init()
  }, [])

  const createTab = useCallback(async (cliName: string) => {
    const defaultName = `${cliName} ${tabCounter}`
    setTabCounter((n) => n + 1)
    try {
      const sessionId = await CreateSession(cliName, defaultName)
      const tab: Tab = {
        id: sessionId,
        name: defaultName,
        sessionId,
        cli: cliName,
      }
      setTabs((prev) => [...prev, tab])
      setActiveId(sessionId)
    } catch (err) {
      console.error('[App] CreateSession failed:', err)
    }
  }, [tabCounter])

  const handleAddTab = useCallback(() => {
    if (detectedCLIs.length === 0) {
      // No CLIs found — open settings so the user can configure a path.
      setShowSettings(true)
      return
    }
    if (detectedCLIs.length === 1) {
      // Only one CLI available: create immediately.
      void createTab(detectedCLIs[0].Name)
    } else {
      // Multiple CLIs: show a picker.
      setShowCLIPicker(true)
    }
  }, [detectedCLIs, createTab])

  const handleSelectCLI = useCallback((cliName: string) => {
    setShowCLIPicker(false)
    void createTab(cliName)
  }, [createTab])

  const handleCloseTab = useCallback(async (id: string) => {
    // Disable web serving for this session before closing.
    if (webEnabled[id]) {
      try { await ToggleWebServing(id, false) } catch (_) { /* ignore */ }
      setWebEnabled((prev) => { const n = { ...prev }; delete n[id]; return n })
      setSessionURLs((prev) => { const n = { ...prev }; delete n[id]; return n })
    }
    try {
      await KillSession(id)
    } catch (err) {
      console.warn('[App] KillSession failed:', err)
    }
    setTabs((prev) => {
      const remaining = prev.filter((t) => t.id !== id)
      // Activate an adjacent tab if the closed one was active.
      if (activeId === id) {
        const idx = prev.findIndex((t) => t.id === id)
        const next = remaining[Math.max(0, idx - 1)]
        setActiveId(next?.id ?? null)
      }
      return remaining
    })
  }, [activeId, webEnabled])

  const handleRenameTab = useCallback(async (id: string, name: string) => {
    try {
      await RenameSession(id, name)
    } catch (err) {
      console.warn('[App] RenameSession failed:', err)
    }
    setTabs((prev) =>
      prev.map((t) => (t.id === id ? { ...t, name } : t))
    )
  }, [])

  const handleToggleWeb = useCallback(async (sessionId: string) => {
    const nowEnabled = !webEnabled[sessionId]
    try {
      await ToggleWebServing(sessionId, nowEnabled)
      setWebEnabled((prev) => ({ ...prev, [sessionId]: nowEnabled }))
      if (nowEnabled) {
        const url = await GetWebServerURL()
        if (url) {
          setSessionURLs((prev) => ({
            ...prev,
            [sessionId]: `${url}/sessions/${sessionId}`,
          }))
        }
      } else {
        setSessionURLs((prev) => { const n = { ...prev }; delete n[sessionId]; return n })
      }
    } catch (err) {
      console.warn('[App] ToggleWebServing failed:', err)
    }
  }, [webEnabled])

  const handleCopyTokenLink = useCallback(async (sessionId: string) => {
    try {
      const tokenURL = await GenerateSessionToken(sessionId)
      await navigator.clipboard.writeText(tokenURL)
    } catch (err) {
      console.warn('[App] GenerateSessionToken/clipboard failed:', err)
    }
  }, [])

  // Re-check server running state when settings panel closes (user may have started/stopped server).
  const handleSettingsClose = useCallback(async () => {
    setShowSettings(false)
    try {
      const running = await IsWebServerRunning()
      setWebServerRunning(running)
    } catch (_) { /* ignore */ }
  }, [])

  return (
    <div className="app">
      <TabBar
        tabs={tabs}
        activeId={activeId}
        onSelect={setActiveId}
        onClose={handleCloseTab}
        onRename={handleRenameTab}
        onAdd={handleAddTab}
        onSettings={() => setShowSettings(true)}
      />

      <div className="terminal-container">
        {relayPort !== null &&
          tabs.map((tab) => (
            <div key={tab.sessionId} className="terminal-wrapper">
              {/* Per-tab web serving controls */}
              <div className="web-serving-bar">
                <button
                  className={`web-toggle-btn${webEnabled[tab.sessionId] ? ' web-toggle-btn--active' : ''}`}
                  onClick={() => void handleToggleWeb(tab.sessionId)}
                  disabled={!webServerRunning}
                  title={webServerRunning ? (webEnabled[tab.sessionId] ? 'Disable web serving' : 'Enable web serving') : 'Start web server in Settings first'}
                >
                  {webEnabled[tab.sessionId] ? 'Web On' : 'Web Off'}
                </button>
                {webEnabled[tab.sessionId] && sessionURLs[tab.sessionId] && (
                  <>
                    <a
                      className="web-session-url"
                      href={sessionURLs[tab.sessionId]}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {sessionURLs[tab.sessionId]}
                    </a>
                    <button
                      className="copy-token-btn"
                      onClick={() => void handleCopyTokenLink(tab.sessionId)}
                      title="Copy shareable token link"
                    >
                      Copy Token Link
                    </button>
                  </>
                )}
              </div>
              <TerminalPanel
                sessionId={tab.sessionId}
                isActive={tab.id === activeId}
                relayPort={relayPort}
              />
            </div>
          ))}
      </div>

      {/* CLI picker dropdown — shown when multiple CLIs are detected */}
      {showCLIPicker && (
        <div className="cli-picker-overlay" onClick={() => setShowCLIPicker(false)}>
          <div className="cli-picker" onClick={(e) => e.stopPropagation()}>
            <p className="cli-picker__label">Choose a CLI:</p>
            {detectedCLIs.map((cli) => (
              <button
                key={cli.Name}
                className="cli-picker__btn"
                onClick={() => handleSelectCLI(cli.Name)}
              >
                {cli.Name}
              </button>
            ))}
          </div>
        </div>
      )}

      <SettingsPanel
        isOpen={showSettings}
        onClose={handleSettingsClose}
        clis={detectedCLIs}
      />
    </div>
  )
}

export default App
