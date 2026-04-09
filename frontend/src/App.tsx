import React, { useEffect, useState, useCallback } from 'react'
import { TabBar, type Tab } from './components/TabBar'
import { Sidebar } from './components/Sidebar'
import { TerminalPanel } from './components/TerminalPanel'
import { SettingsTab } from './components/SettingsTab'
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
  RetryDaemon,
  GetDaemonError,
  GetRemoteSessions,
  AutoInstallTailscale,
} from './wailsjs/go/main/App'
import type { DetectedCLI, SessionInfo, RemotePeerSessions } from './wailsjs/go/main/App'
import { EventsOn, Environment, BrowserOpenURL } from './wailsjs/wailsjs/runtime/runtime'
import { QRModal } from './components/QRModal'
import { StatusBar } from './components/StatusBar'
import { NewSessionModal } from './components/NewSessionModal'
import { HealthModal } from './components/HealthModal'
import { WelcomeTab } from './components/WelcomeTab'
import { DaemonManagerPanel } from './components/DaemonManagerPanel'
import { RemoteSessionsPanel } from './components/RemoteSessionsPanel'

const DEFAULT_FONT_SIZE = 14

/**
 * App is the root component — it owns all tab state and wires
 * the Wails-generated TypeScript bindings to the child components.
 */
function App(): React.ReactElement {
  const WELCOME_TAB: Tab = { id: '__welcome__', name: 'Welcome', sessionId: '', cli: '', type: 'welcome' }
  const DAEMON_MANAGER_TAB: Tab = { id: '__daemon_manager__', name: 'Sessions', sessionId: '', cli: '', type: 'daemon-manager' }
  const REMOTE_SESSIONS_TAB: Tab = { id: '__remote_sessions__', name: 'Remote', sessionId: '', cli: '', type: 'remote-sessions' }
  const SETTINGS_TAB: Tab = { id: '__settings__', name: 'Settings', sessionId: '', cli: '', type: 'settings' }
  const [tabs, setTabs] = useState<Tab[]>([WELCOME_TAB])
  const [activeId, setActiveId] = useState<string | null>(WELCOME_TAB.id)
  const [relayPort, setRelayPort] = useState<number | null>(null)
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
  // Auto-install Tailscale state
  const [installProgress, setInstallProgress] = useState<string[]>([])
  const [installStatus, setInstallStatus] = useState<'idle' | 'running' | 'success' | 'error'>('idle')
  const [installError, setInstallError] = useState<string | undefined>(undefined)
  // Sessions list for the DaemonManagerPanel (polled when the panel tab is active)
  const [panelSessions, setPanelSessions] = useState<SessionInfo[]>([])
  // Remote peers for RemoteSessionsPanel (polled when the tab is active)
  const [remotePeers, setRemotePeers] = useState<RemotePeerSessions[]>([])
  const [remoteLoading, setRemoteLoading] = useState(false)

  // On mount: hide static HTML splash and initialize.
  useEffect(() => {
    const splashEl = document.getElementById('splash-static')
    if (splashEl) splashEl.style.display = 'none'

    async function init() {
      // Check if startup() failed before calling methods that need a.client.
      const startupErr = await GetDaemonError()
      if (startupErr) {
        setDaemonError(startupErr)
        return
      }
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

          // Seed webEnabled state from daemon's SessionInfo.webEnabled field (SERVE-02 restore).
          if (running) {
            const enabledMap: Record<string, boolean> = {}
            const urlMap: Record<string, string> = {}
            let serverURL: string | undefined
            try {
              serverURL = await GetWebServerURL()
            } catch (_) { /* ignore */ }

            sessions.forEach((s) => {
              if (s.webEnabled) {
                enabledMap[s.id] = true
                if (serverURL) {
                  urlMap[s.id] = `${serverURL}/sessions/${s.id}`
                }
              }
            })
            if (Object.keys(enabledMap).length > 0) {
              setWebEnabled(enabledMap)
              setSessionURLs(urlMap)
            }
          }
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

    const offDaemonError = EventsOn('daemon:error', (msg: string) => {
      setDaemonError(msg)
    })

    const cancelTrayFocus = EventsOn('tray:focus-session', (sessionId: string) => {
      setTabs(prev => {
        const tab = prev.find(t => t.sessionId === sessionId)
        if (tab) {
          setActiveId(tab.id)
        }
        return prev
      })
    })

    const cancelInstallProgress = EventsOn('tailscale:install:progress', (line: string) => {
      setInstallProgress(prev => [...prev, line])
    })
    const cancelInstallDone = EventsOn('tailscale:install:done', (result: { success: boolean; error?: string }) => {
      if (result.success) {
        setInstallStatus('success')
      } else {
        setInstallStatus('error')
        setInstallError(result.error ?? 'Unknown error')
      }
    })

    return () => {
      offStatus()
      offHealth()
      offDaemonError()
      cancelTrayFocus()
      cancelInstallProgress()
      cancelInstallDone()
    }
  }, [])

  const createTab = useCallback(async (cliName: string, workDir: string, args: string[]) => {
    const defaultName = `${cliName} ${tabCounter}`
    setTabCounter((n) => n + 1)

    // Estimate initial PTY dimensions from the terminal container.
    // These are approximations — the double-rAF fit sends exact resize after mount.
    const container = document.querySelector('.terminal-container') as HTMLElement | null
    let cols = 220, rows = 50  // Reasonable fallback for large screens
    if (container && container.clientWidth > 0 && container.clientHeight > 0) {
      const statusBarHeight = 32  // .tab-status-bar fixed height
      cols = Math.max(80, Math.floor(container.clientWidth / 8))
      rows = Math.max(24, Math.floor((container.clientHeight - statusBarHeight) / 17))
    }

    try {
      const sessionId = await CreateSession(cliName, defaultName, workDir, args, cols, rows)
      const tab: Tab = {
        id: sessionId,
        name: defaultName,
        sessionId,
        cli: cliName,
      }
      setTabs((prev) => [...prev, tab])
      setActiveId(sessionId)

      // Auto-seed webEnabled state for new sessions when web server is running (SERVE-02).
      if (webServerRunning) {
        setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
        try {
          const url = await GetWebServerURL()
          if (url) {
            setSessionURLs((prev) => ({ ...prev, [sessionId]: `${url}/sessions/${sessionId}` }))
          }
        } catch (_) { /* URL fetch failure is non-fatal */ }
      }
    } catch (err) {
      console.error('[App] CreateSession failed:', err)
    }
  }, [tabCounter, webServerRunning])

  const handleOpenSettings = useCallback(() => {
    const existing = tabs.find((t) => t.type === 'settings')
    if (existing) {
      setActiveId(existing.id)
      return
    }
    setTabs((prev) => [...prev, SETTINGS_TAB])
    setActiveId(SETTINGS_TAB.id)
  }, [tabs])

  const handleAddTab = useCallback(() => {
    if (detectedCLIs.length === 0) {
      // No CLIs found — open settings so the user can configure a path.
      handleOpenSettings()
      return
    }
    setShowNewSessionModal(true)
  }, [detectedCLIs, handleOpenSettings])

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

  const handleAutoInstallTailscale = useCallback(async () => {
    setInstallProgress([])
    setInstallStatus('running')
    setInstallError(undefined)
    try {
      await AutoInstallTailscale()
    } catch (err) {
      setInstallStatus('error')
      setInstallError(String(err))
    }
  }, [])

  // Poll sessions when the daemon-manager panel tab is active.
  useEffect(() => {
    const isDaemonManagerActive = activeId === DAEMON_MANAGER_TAB.id
    if (!isDaemonManagerActive) return

    let cancelled = false
    async function refresh() {
      try {
        const sessions = await ListSessions()
        if (!cancelled) setPanelSessions(sessions)
      } catch (err) {
        console.warn('[App] ListSessions poll failed:', err)
      }
    }
    void refresh()
    const interval = setInterval(() => void refresh(), 3000)
    return () => {
      cancelled = true
      clearInterval(interval)
    }
  }, [activeId])

  // Poll remote sessions when the remote-sessions tab is active.
  useEffect(() => {
    if (activeId !== REMOTE_SESSIONS_TAB.id) return
    let cancelled = false
    async function refresh() {
      if (remotePeers.length === 0) setRemoteLoading(true)
      try {
        const peers = await GetRemoteSessions()
        if (!cancelled) {
          setRemotePeers(peers ?? [])
          setRemoteLoading(false)
        }
      } catch {
        if (!cancelled) setRemoteLoading(false)
      }
    }
    void refresh()
    const interval = setInterval(() => void refresh(), 30_000)
    return () => {
      cancelled = true
      clearInterval(interval)
    }
  }, [activeId])

  const handleOpenDaemonManager = useCallback(() => {
    // If daemon-manager tab already exists, just focus it.
    const existing = tabs.find((t) => t.type === 'daemon-manager')
    if (existing) {
      setActiveId(existing.id)
      return
    }
    // Otherwise, add it and focus.
    setTabs((prev) => [...prev, DAEMON_MANAGER_TAB])
    setActiveId(DAEMON_MANAGER_TAB.id)
  }, [tabs])

  const handleOpenRemoteSession = useCallback((url: string) => {
    BrowserOpenURL(url)
  }, [])

  const handleOpenRemoteSessions = useCallback(() => {
    const existing = tabs.find((t) => t.type === 'remote-sessions')
    if (existing) {
      setActiveId(existing.id)
      return
    }
    setTabs((prev) => [...prev, REMOTE_SESSIONS_TAB])
    setActiveId(REMOTE_SESSIONS_TAB.id)
  }, [tabs])

  const handleHome = useCallback(() => {
    const existing = tabs.find((t) => t.type === 'welcome')
    if (existing) {
      setActiveId(existing.id)
      return
    }
    setTabs((prev) => [...prev, WELCOME_TAB])
    setActiveId(WELCOME_TAB.id)
  }, [tabs])

  const retryInit = useCallback(async () => {
    setDaemonError(null)
    try {
      await RetryDaemon()
    } catch (err) {
      setDaemonError(String(err))
      return
    }
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
      <Sidebar
        onHome={handleHome}
        onOpenRemoteSessions={handleOpenRemoteSessions}
        onOpenDaemonManager={handleOpenDaemonManager}
        onAdd={handleAddTab}
        onSettings={handleOpenSettings}
      />
      <div className="app__content">
        <TabBar
          tabs={tabs}
          activeId={activeId}
          onSelect={setActiveId}
          onClose={handleCloseTab}
          onRename={handleRenameTab}
          sessionStatuses={sessionStatuses}
        />

        <div className="terminal-container">
        {activeId === WELCOME_TAB.id && (
          <WelcomeTab />
        )}
        {activeId === DAEMON_MANAGER_TAB.id && (
          <DaemonManagerPanel
            sessions={panelSessions}
            sessionStatuses={sessionStatuses}
            webServerRunning={webServerRunning}
            webEnabled={webEnabled}
            onKill={(id) => void handleCloseTab(id)}
            onToggleWeb={(id) => void handleToggleWeb(id)}
          />
        )}
        {activeId === REMOTE_SESSIONS_TAB.id && (
          <RemoteSessionsPanel
            peers={remotePeers}
            loading={remoteLoading}
            onOpen={handleOpenRemoteSession}
          />
        )}
        {activeId === SETTINGS_TAB.id && (
          <SettingsTab
            clis={detectedCLIs}
            tailscaleHealth={tailscaleHealth}
            onWebServerStateChange={async () => {
              const running = await IsWebServerRunning()
              setWebServerRunning(running)
            }}
          />
        )}
        {daemonError && tabs.filter((t) => t.type !== 'welcome' && t.type !== 'daemon-manager' && t.type !== 'remote-sessions' && t.type !== 'settings').length === 0 && (
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
              {daemonError}
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
            if (tab.type === 'welcome' || tab.type === 'daemon-manager' || tab.type === 'remote-sessions' || tab.type === 'settings') return null
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
      </div>

      {showNewSessionModal && (
        <NewSessionModal
          isOpen={showNewSessionModal}
          clis={detectedCLIs}
          onConfirm={(cli, workDir, args) => {
            setShowNewSessionModal(false)
            void createTab(cli, workDir, args)
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

      <HealthModal
        health={tailscaleHealth}
        platform={platform}
        onCheckAgain={handleCheckHealthAgain}
        onOpenURL={BrowserOpenURL}
        onAutoInstall={handleAutoInstallTailscale}
        installProgress={installProgress}
        installStatus={installStatus}
        installError={installError}
      />
    </div>
  )
}

export default App
