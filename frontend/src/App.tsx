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
  GetWebServerURL,
  IsWebServerRunning,
  GetSessionStatus,
  GetTailscaleStatus,
} from './wailsjs/go/main/App'
import type { DetectedCLI } from './wailsjs/go/main/App'
import { EventsOn, Environment } from './wailsjs/wailsjs/runtime/runtime'
import { QRModal } from './components/QRModal'
import { StatusBar } from './components/StatusBar'
import { NewSessionModal } from './components/NewSessionModal'
import { HealthModal } from './components/HealthModal'

const DEFAULT_FONT_SIZE = 14

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
  const [showNewSessionModal, setShowNewSessionModal] = useState(false)
  // Track web serving state per session: sessionId -> enabled
  const [webEnabled, setWebEnabled] = useState<Record<string, boolean>>({})
  // Track per-session web URLs (populated when web serving is enabled)
  const [sessionURLs, setSessionURLs] = useState<Record<string, string>>({})
  // Track web server running state
  const [webServerRunning, setWebServerRunning] = useState(false)
  // Track live status per session: sessionId -> status string
  const [sessionStatuses, setSessionStatuses] = useState<Record<string, string>>({})
  // Track which session's QR modal is open (null = none)
  const [qrSessionId, setQrSessionId] = useState<string | null>(null)
  // Track font size per session: sessionId -> fontSize (pixels)
  const [fontSizes, setFontSizes] = useState<Record<string, number>>({})
  // Tailscale health state
  const [tailscaleHealth, setTailscaleHealth] = useState<{
    installed: boolean
    connected: boolean
    hasCerts: boolean
    ip: string
    domain: string
  } | null>(null)
  const [platform, setPlatform] = useState<string>('linux')
  const [daemonError, setDaemonError] = useState<string | null>(null)

  // On mount: get relay port, detect CLIs, restore any existing sessions.
  useEffect(() => {
    async function init() {
      try {
        const [port, clis, sessions, running, health, env] = await Promise.all([
          GetRelayPort(),
          DetectCLIs(),
          ListSessions(),
          IsWebServerRunning(),
          GetTailscaleStatus(),
          Environment(),
        ])
        setRelayPort(port)
        setDetectedCLIs(clis)
        setWebServerRunning(running)
        setTailscaleHealth(health)
        setPlatform(env.platform)

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

          // Seed initial status for each restored session.
          sessions.forEach((s) => {
            GetSessionStatus(s.id)
              .then((st) => {
                setSessionStatuses((prev) => ({ ...prev, [s.id]: st }))
              })
              .catch(() => { /* status unavailable — leave unset */ })
          })
        }
      } catch (err) {
        console.error('[App] init failed:', err)
        setDaemonError(String(err))
      }
    }
    void init()

    // Subscribe to live session:status events from the Go backend.
    const offStatus = EventsOn(
      'session:status',
      (data: { sessionId: string; status: string }) => {
        setSessionStatuses((prev) => ({ ...prev, [data.sessionId]: data.status }))
      },
    )

    const offHealth = EventsOn('tailscale:health', (h: {
      installed: boolean
      connected: boolean
      hasCerts: boolean
      ip: string
      domain: string
    }) => {
      setTailscaleHealth(h)
    })

    return () => {
      offStatus()
      offHealth()
    }
  }, [])

  const createTab = useCallback(async (cliName: string, workDir: string) => {
    const defaultName = `${cliName} ${tabCounter}`
    setTabCounter((n) => n + 1)
    try {
      const sessionId = await CreateSession(cliName, defaultName, workDir)
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
    setShowNewSessionModal(true)
  }, [detectedCLIs])

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
    // Clean up session status for the closed session.
    setSessionStatuses((prev) => { const n = { ...prev }; delete n[id]; return n })
    // Clean up font size for the closed session.
    setFontSizes((prev) => { const n = { ...prev }; delete n[id]; return n })
    // Close QR modal if it was open for this session.
    setQrSessionId((prev) => (prev === id ? null : prev))
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

  // Re-check server running state when settings panel closes (user may have started/stopped server).
  const handleSettingsClose = useCallback(async () => {
    setShowSettings(false)
    try {
      const running = await IsWebServerRunning()
      setWebServerRunning(running)
    } catch (_) { /* ignore */ }
  }, [])

  const handleFontSizeChange = useCallback((sessionId: string, delta: number) => {
    setFontSizes((prev) => {
      const current = prev[sessionId] ?? DEFAULT_FONT_SIZE
      const next = Math.max(6, Math.min(32, current + delta))
      return { ...prev, [sessionId]: next }
    })
  }, [])

  const handleCheckHealthAgain = useCallback(async () => {
    try {
      const health = await GetTailscaleStatus()
      setTailscaleHealth(health)
    } catch (err) {
      console.error('[App] GetTailscaleStatus failed:', err)
    }
  }, [])

  const retryInit = useCallback(async () => {
    setDaemonError(null)
    try {
      const [port, clis, sessions, running, health, env] = await Promise.all([
        GetRelayPort(),
        DetectCLIs(),
        ListSessions(),
        IsWebServerRunning(),
        GetTailscaleStatus(),
        Environment(),
      ])
      setRelayPort(port)
      setDetectedCLIs(clis)
      setWebServerRunning(running)
      setTailscaleHealth(health)
      setPlatform(env.platform)
      if (sessions.length > 0) {
        const restoredTabs: Tab[] = sessions.map((s) => ({
          id: s.id,
          name: s.name || s.cli,
          sessionId: s.id,
          cli: s.cli,
        }))
        setTabs(restoredTabs)
        setActiveId(restoredTabs[0].id)
        sessions.forEach((s) => {
          GetSessionStatus(s.id)
            .then((st) => setSessionStatuses((prev) => ({ ...prev, [s.id]: st })))
            .catch(() => {})
        })
      }
    } catch (err) {
      console.error('[App] retry init failed:', err)
      setDaemonError(String(err))
    }
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
        sessionStatuses={sessionStatuses}
      />

      <div className="terminal-container">
        {daemonError && tabs.length === 0 && (
          <div style={{
            background: '#16161e',
            borderLeft: '3px solid #f7768e',
            border: '1px solid #292e42',
            borderLeftWidth: '3px',
            borderLeftColor: '#f7768e',
            borderRadius: '4px',
            padding: '12px 16px',
            margin: '16px',
            color: '#a9b1d6',
            fontSize: '13px',
          }}>
            <div style={{ fontWeight: 600, marginBottom: '8px', color: '#c0caf5' }}>
              Unable to connect to session daemon
            </div>
            <div style={{ marginBottom: '12px' }}>
              The background daemon did not start in time. Your sessions are not accessible. Check the system log or restart AgentHub.
            </div>
            <button
              onClick={retryInit}
              style={{
                border: '1px solid #292e42',
                background: 'transparent',
                color: '#a9b1d6',
                padding: '4px 12px',
                borderRadius: '4px',
                cursor: 'pointer',
                fontSize: '13px',
              }}
              onMouseEnter={(e) => { (e.target as HTMLElement).style.background = '#1e2030' }}
              onMouseLeave={(e) => { (e.target as HTMLElement).style.background = 'transparent' }}
            >
              Retry Connection
            </button>
          </div>
        )}
        {relayPort != null && relayPort > 0 &&
          tabs.map((tab) => {
            const isActive = tab.id === activeId
            return (
              <div
                key={tab.sessionId}
                className="terminal-wrapper"
                style={{ display: isActive ? 'flex' : 'none' }}
              >
                <TerminalPanel
                  sessionId={tab.sessionId}
                  isActive={isActive}
                  relayPort={relayPort}
                  fontSize={fontSizes[tab.sessionId] ?? DEFAULT_FONT_SIZE}
                  onFontSizeChange={(delta) => handleFontSizeChange(tab.sessionId, delta)}
                />
                <StatusBar
                  sessionId={tab.sessionId}
                  webServerRunning={webServerRunning}
                  webEnabled={!!webEnabled[tab.sessionId]}
                  sessionURL={sessionURLs[tab.sessionId]}
                  onToggleWeb={() => void handleToggleWeb(tab.sessionId)}
                  onShowQR={() => setQrSessionId(tab.sessionId)}
                />
              </div>
            )
          })}
      </div>

      {showNewSessionModal && (
        <NewSessionModal
          isOpen={showNewSessionModal}
          clis={detectedCLIs}
          onConfirm={(cli, workDir) => {
            setShowNewSessionModal(false)
            void createTab(cli, workDir)
          }}
          onClose={() => setShowNewSessionModal(false)}
        />
      )}

      {qrSessionId !== null && (
        <QRModal
          sessionId={qrSessionId}
          sessionURL={sessionURLs[qrSessionId]}
          onClose={() => setQrSessionId(null)}
        />
      )}

      <SettingsPanel
        isOpen={showSettings}
        onClose={handleSettingsClose}
        clis={detectedCLIs}
        tailscaleHealth={tailscaleHealth}
      />

      <HealthModal
        health={tailscaleHealth}
        platform={platform}
        onCheckAgain={handleCheckHealthAgain}
      />
    </div>
  )
}

export default App
