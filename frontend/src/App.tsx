import React, { useEffect, useState, useCallback, useRef } from 'react'
import type { IProgressState } from '@xterm/addon-progress'
import * as xtermThemes from 'xterm-theme'
import type { ITheme } from '@xterm/xterm'
import { TabBar, type Tab } from './components/TabBar'
import { Sidebar } from './components/Sidebar'
import { TerminalPanel } from './components/TerminalPanel'
import { SettingsTab } from './components/SettingsTab'
import { stripAnsi } from './lib/stripAnsi'
import { sanitizeFilename } from './lib/sanitizeFilename'
import {
  CreateSession,
  ListSessions,
  KillSession,
  RenameSession,
  DetectCLIs,
  ListShells,
  GetRelayPort,
  ToggleWebServing,
  IsWebServerRunning,
  GetSessionStatus,
  GetTailscaleStatus,
  RetryDaemon,
  GetDaemonError,
  GetRemoteSessions,
  GetWebServerMode,
  NotifyThemeChange,
  GetLastUpdateInfo,
  GetAutoCloseSession,
  GetPluginSettings,
  QuitGUIOnly,
  QuitAll,
  SaveTerminalSession,
  SetTrayProgress,
} from './wailsjs/go/main/App'
import { aggregateProgress } from './lib/aggregateProgress'
import type { DetectedCLI, SessionInfo, RemotePeerSessions } from './wailsjs/go/main/App'
import type { daemon } from './wailsjs/go/models'
type PluginSettings = daemon.PluginSettings
import { EventsOn, BrowserOpenURL } from './wailsjs/wailsjs/runtime/runtime'
import { StatusBar } from './components/StatusBar'
import { NewSessionModal } from './components/NewSessionModal'
import { WelcomeTab } from './components/WelcomeTab'
import { DaemonManagerPanel } from './components/DaemonManagerPanel'
import { RemoteSessionsPanel } from './components/RemoteSessionsPanel'
import { LocalNetworkBanner } from './components/LocalNetworkBanner'
import { UpdateBanner } from './components/UpdateBanner'
import type { UpdateInfo } from './components/UpdateBanner'
import { WebGLRecoveryBanner } from './components/WebGLRecoveryBanner'
import { PluginToggleBanner } from './components/PluginToggleBanner'
import { ExitToast } from './components/ExitToast'
import type { ExitState } from './components/ExitToast'
import { ExitCountdownBanner } from './components/ExitCountdownBanner'
import { QuitConfirmModal } from './components/QuitConfirmModal'
import { ALLOWED_THEMES } from './themes'

const DEFAULT_FONT_SIZE = 14
const THEME_STORAGE_KEY = 'agenthub:terminalTheme'
const DEFAULT_THEME_NAME = 'Tomorrow_Night'

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
  // Phase 101-02 (SHELL-01 GUI half) — discovered shells from the daemon.
  // Threaded into NewSessionModal so the user can pick a shell session.
  const [detectedShells, setDetectedShells] = useState<daemon.DetectedShell[]>([])
  const [shellsLoading, setShellsLoading] = useState(true)
  const [tabCounter, setTabCounter] = useState(1)
  const [showNewSessionModal, setShowNewSessionModal] = useState(false)
  // Track web serving state per session: sessionId -> enabled
  const [webEnabled, setWebEnabled] = useState<Record<string, boolean>>({})
  // Track web server running state
  const [webServerRunning, setWebServerRunning] = useState(false)
  // Track web server mode: 'tailscale' | 'local' | null
  const [webServerMode, setWebServerMode] = useState<'tailscale' | 'local' | null>(null)
  // Track live status per session: sessionId -> status string
  const [sessionStatuses, setSessionStatuses] = useState<Record<string, string>>({})
  // Track which session's QR modal is open (null = none)
  // Track font size per session: sessionId -> fontSize (pixels)
  const [fontSizes, setFontSizes] = useState<Record<string, number>>({})
  // Tailscale health state
  const [tailscaleHealth, setTailscaleHealth] = useState<{
    installed: boolean
    connected: boolean
    hasCerts: boolean
    ip: string
    domain: string
    binaryFound: boolean
    daemonUp: boolean
    platformHint: string
  } | null>(null)
  const [daemonError, setDaemonError] = useState<string | null>(null)
  // Plugin settings state (PLUG-03): updated by GetPluginSettings on mount
  // and by the EventsOn('settings:plugins') subscription. Threaded into every
  // open TerminalPanel via prop. Phase 92 contract: TerminalPanel accepts the
  // prop but does not consume it; Phase 93 wires consumption.
  const [pluginConfig, setPluginConfig] = useState<PluginSettings | null>(null)

  // Phase 97 SER-01: saver registry. TerminalPanel registers a closure
  // that returns the addon's serialize() output keyed by sessionId; App
  // uses it when handleRequestSave fires from the TabBar context menu.
  // Cleared on unmount via TerminalPanel's useEffect cleanup (Pitfall #6).
  const [serializerRegistry, setSerializerRegistry] = useState<
    Record<string, (() => string) | null>
  >({})

  // Phase 97 SER-01: one-shot save-feedback banner. Mirrors the
  // localBanner pattern — info kind for "Serialize disabled"
  // affordance, error kind for write/dialog failures.
  const [saveBanner, setSaveBanner] = useState<{ kind: 'info' | 'error'; text: string } | null>(null)

  // Phase 98 PRG-02/PRG-03 — progress registry (mirrors Phase 97 saver registry shape).
  // useRef Map is the aggregation source: no re-render on .set/.delete; the actual
  // per-tab UI prop lives in tabProgress (useState Record). The Wails-RPC dispatch
  // is debounced via trayDebounceRef + idempotency-guarded by lastDispatchedQuartileRef.
  const progressRegistry = useRef(new Map<string, IProgressState>())
  const [tabProgress, setTabProgress] = useState<Record<string, number>>({})
  const trayDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null) // 200ms debounce window
  const lastDispatchedQuartileRef = useRef<number>(-1)

  // Ref for the background upgrade poller (local -> tailscale mode transition)
  const upgradePollerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  // Sessions list for the DaemonManagerPanel (polled when the panel tab is active)
  const [panelSessions, setPanelSessions] = useState<SessionInfo[]>([])
  // Remote peers for RemoteSessionsPanel (polled when the tab is active)
  const [remotePeers, setRemotePeers] = useState<RemotePeerSessions[]>([])
  const [remoteLoading, setRemoteLoading] = useState(false)

  // Update notification state (lifted from WelcomeTab — Phase 81 D-06)
  const [update, setUpdate] = useState<UpdateInfo | null>(null)
  // Local network banner dismiss state (session-only, D-04)
  const [localBannerDismissed, setLocalBannerDismissed] = useState(false)
  const [localBannerExiting, setLocalBannerExiting] = useState(false)
  const [updateExiting, setUpdateExiting] = useState(false)

  // Phase 93 WGL-02 / WGL-03: WebGL recovery banner state. One-shot per
  // app session — webglBannerDismissed gates rendering even if the
  // underlying webglContextLost / webglSoftwareDetected event fires
  // multiple times (e.g., user toggles WebGL OFF/ON while context lost).
  const [webglContextLost, setWebglContextLost] = useState(false)
  const [webglSoftwareDetected, setWebglSoftwareDetected] = useState(false)
  const [webglBannerDismissed, setWebglBannerDismissed] = useState(false)

  // Phase 99 PUI-02: one-shot post-save Unicode 11 / Image toggle banners.
  // Each kind appears at most once at any moment; dismissing removes that kind
  // from the set; auto-dismiss fires from the PluginToggleBanner component.
  // Set deduplication via Array.from(new Set(...)) caps the array at 2 entries max.
  type PluginToggleKindLocal = 'unicode11' | 'image'
  const [pluginToggleBanners, setPluginToggleBanners] = useState<PluginToggleKindLocal[]>([])

  const handlePluginToggleSideEffect = useCallback((kinds: PluginToggleKindLocal[]) => {
    setPluginToggleBanners((prev) => Array.from(new Set([...prev, ...kinds])))
  }, [])

  // Session exit state: per-session exit info for toast/banner/countdown (Phase 84)
  const [sessionExits, setSessionExits] = useState<Record<string, ExitState>>({})
  // Quit confirmation modal state (Phase 85)
  const [showQuitModal, setShowQuitModal] = useState(false)
  const [quitSessions, setQuitSessions] = useState<Array<{ id: string; name: string; status: string }>>([])
  const countdownTimers = useRef<Record<string, ReturnType<typeof setInterval>>>({})
  // Auto-close setting (D-11): loaded on mount, default true
  // Stored as a ref (not state) because it's only read inside the EventsOn callback
  // which runs in a [] deps useEffect — a ref avoids stale closures without the
  // setAutoCloseEnabled(current => ...) trick that triggers TS6133.
  const autoCloseRef = useRef(true)
  // Ref to handleCloseTab so the session:exit event handler (with [] deps) can call
  // the latest version without a stale closure
  const handleCloseTabRef = useRef<((id: string) => Promise<void>) | undefined>(undefined)

  // Terminal theme (global — same theme for all sessions)
  const [terminalThemeName, setTerminalThemeName] = useState<string>(() => {
    const stored = localStorage.getItem(THEME_STORAGE_KEY) ?? DEFAULT_THEME_NAME
    if (ALLOWED_THEMES.includes(stored)) return stored
    if (ALLOWED_THEMES.includes(DEFAULT_THEME_NAME)) return DEFAULT_THEME_NAME
    return ALLOWED_THEMES[0] ?? DEFAULT_THEME_NAME
  })
  const terminalTheme: ITheme = (xtermThemes as Record<string, ITheme>)[terminalThemeName]
    ?? (xtermThemes as Record<string, ITheme>)[DEFAULT_THEME_NAME]

  const handleThemeChange = useCallback((name: string) => {
    localStorage.setItem(THEME_STORAGE_KEY, name)
    setTerminalThemeName(name)
    // Signal active OpenCode sessions to re-query terminal palette (SIGUSR2).
    // Fire-and-forget: errors logged to console, never block UI.
    NotifyThemeChange().catch(err => console.warn('NotifyThemeChange failed:', err))
  }, [])

  // Phase 97 SER-01: TerminalPanel calls this on attach/detach to
  // register/unregister its serialize() closure. Empty deps because
  // setSerializerRegistry is stable.
  const handleRegisterSaver = useCallback(
    (sessionId: string, fn: (() => string) | null) => {
      setSerializerRegistry((prev) => ({ ...prev, [sessionId]: fn }))
    },
    []
  )

  // Phase 98 PRG-02/PRG-03 — fired by every TerminalPanel via the onProgressChange prop.
  // state:1 (set) updates registry+tabProgress; everything else (state:0/2/3/4) clears the entry
  // (v3.2 ships state:1+state:0 in the UI; 2/3/4 deferred to v3.3 per RESEARCH "Cuttable Inside Cuttable").
  const handleProgressChange = useCallback(
    (sessionId: string, state: IProgressState) => {
      if (state.state === 1) {
        progressRegistry.current.set(sessionId, state)
        setTabProgress((prev) => ({ ...prev, [sessionId]: state.value }))
      } else {
        progressRegistry.current.delete(sessionId)
        setTabProgress((prev) => {
          // Strip the key without mutating the previous object.
          // eslint-disable-next-line @typescript-eslint/no-unused-vars
          const { [sessionId]: _drop, ...rest } = prev
          return rest
        })
      }
      // Recompute aggregate quartile + schedule debounced RPC.
      const quartile = aggregateProgress(progressRegistry.current)
      if (trayDebounceRef.current) {
        clearTimeout(trayDebounceRef.current)
      }
      trayDebounceRef.current = setTimeout(() => {
        if (lastDispatchedQuartileRef.current === quartile) return  // idempotent
        lastDispatchedQuartileRef.current = quartile
        void SetTrayProgress(quartile)
      }, 200)
    },
    []
  )

  // Phase 97 SER-01: TabBar calls this when the user picks "Save Terminal As…"
  // from the right-click context menu. Looks up the saver closure for the
  // tab's session, strips ANSI, sanitizes the filename, and invokes the
  // SaveTerminalSession Wails RPC. Shows a banner if no saver is registered
  // (Serialize toggled OFF) — per RESEARCH §"User Constraints / Whether
  // the Save menu item appears when Serialize is toggled OFF": always show
  // the menu item; show toast on click if disabled (discoverability).
  const handleRequestSave = useCallback(
    async (tabId: string) => {
      const tab = tabs.find((t) => t.id === tabId)
      if (!tab || !tab.sessionId) {
        setSaveBanner({ kind: 'info', text: 'Open a terminal session to save its scrollback.' })
        return
      }
      const fn = serializerRegistry[tab.sessionId]
      if (!fn) {
        setSaveBanner({ kind: 'info', text: 'Enable the Serialize plugin in Settings to save sessions.' })
        return
      }
      const plainText = stripAnsi(fn())
      const stamp = new Date().toISOString().replace(/[:T]/g, '-').replace(/\..+/, '')
      const fname = sanitizeFilename(tab.name) + '-' + stamp + '.txt'
      try {
        await SaveTerminalSession('', fname, plainText)
      } catch (err) {
        setSaveBanner({ kind: 'error', text: 'Could not save terminal: ' + String(err) })
      }
    },
    [tabs, serializerRegistry]
  )

  const handleDismissLocalBanner = useCallback(() => {
    setLocalBannerExiting(true)
    setTimeout(() => {
      setLocalBannerDismissed(true)
      setLocalBannerExiting(false)
    }, 200)
  }, [])

  const handleDismissUpdate = useCallback(() => {
    setUpdateExiting(true)
    setTimeout(() => {
      setUpdate(null)
      setUpdateExiting(false)
    }, 200)
  }, [])

  // Phase 93 WGL-02 / WGL-03: stable callback for TerminalPanel's hot-swap
  // useEffect dep array. useCallback with [] keeps the identity stable so
  // the hot-swap effect doesn't re-run when App re-renders for unrelated
  // reasons.
  const handleWebGLContextLost = useCallback((reason: 'context-loss' | 'software-rasterized') => {
    if (reason === 'software-rasterized') {
      setWebglSoftwareDetected(true)
    } else {
      setWebglContextLost(true)
    }
  }, [])

  // Phase 98 PRG-03 — clear the debounce timer on App unmount so a pending
  // tray-RPC doesn't fire after the React tree is torn down (cosmetic but
  // matches the established cleanup pattern from Phase 94 BannerStack).
  useEffect(() => {
    return () => {
      if (trayDebounceRef.current) {
        clearTimeout(trayDebounceRef.current)
        trayDebounceRef.current = null
      }
    }
  }, [])

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
        const [port, clis, sessions, running, health] = await Promise.all([
          GetRelayPort(),
          DetectCLIs(),
          ListSessions(),
          IsWebServerRunning(),
          GetTailscaleStatus(),
        ])
        setRelayPort(port)
        setDetectedCLIs(clis)
        setWebServerRunning(running)
        setTailscaleHealth(health)

        // Phase 101-02 (SHELL-01 GUI half) — call ListShells() on mount.
        // Loaded in parallel via a separate promise so a slow daemon response
        // doesn't delay the rest of the init flow. Failures fall through to
        // an empty shell list (silent absence per UI-SPEC §Edge Cases).
        ListShells()
          .then((s) => setDetectedShells(s ?? []))
          .catch(() => setDetectedShells([]))
          .finally(() => setShellsLoading(false))

        // Fetch web server mode for local-network-fallback UI
        GetWebServerMode().then(mode => {
          const validMode = mode === 'tailscale' || mode === 'local' ? mode : null
          setWebServerMode(validMode)
          if (validMode) setWebServerRunning(true)
        }).catch(() => setWebServerMode(null))

        // If web server isn't running yet, poll briefly — the daemon may still
        // be starting it (local-mode fallback takes a moment after init).
        if (!running) {
          let attempts = 0
          const poll = setInterval(async () => {
            attempts++
            try {
              const nowRunning = await IsWebServerRunning()
              if (nowRunning) {
                setWebServerRunning(true)
                const mode = await GetWebServerMode()
                const validMode = mode === 'tailscale' || mode === 'local' ? mode : null
                setWebServerMode(validMode)
                clearInterval(poll)
              } else if (attempts >= 10) {
                clearInterval(poll)
              }
            } catch {
              clearInterval(poll)
            }
          }, 500)
        }

        // Load auto-close preference (Phase 84 D-11)
        GetAutoCloseSession().then(val => { autoCloseRef.current = val }).catch(() => {})

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
            sessions.forEach((s) => {
              if (s.webEnabled) {
                enabledMap[s.id] = true
              }
            })
            if (Object.keys(enabledMap).length > 0) {
              setWebEnabled(enabledMap)
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
      binaryFound: boolean
      daemonUp: boolean
      platformHint: string
    }) => {
      setTailscaleHealth(h)
      // If Tailscale just became fully healthy, poll for the backend to upgrade
      // from local mode to tailscale mode (daemon's upgradeToTailscale goroutine).
      if (h.connected && h.hasCerts && h.ip) {
        setWebServerMode(prev => {
          if (prev === 'local') {
            // Clear any existing poller before starting a new one.
            if (upgradePollerRef.current !== null) {
              clearInterval(upgradePollerRef.current)
            }
            let attempts = 0
            upgradePollerRef.current = setInterval(async () => {
              attempts++
              try {
                const mode = await GetWebServerMode()
                if (mode === 'tailscale') {
                  setWebServerMode('tailscale')
                  if (upgradePollerRef.current !== null) {
                    clearInterval(upgradePollerRef.current)
                    upgradePollerRef.current = null
                  }
                } else if (attempts >= 10) {
                  if (upgradePollerRef.current !== null) {
                    clearInterval(upgradePollerRef.current)
                    upgradePollerRef.current = null
                  }
                }
              } catch {
                if (attempts >= 10) {
                  if (upgradePollerRef.current !== null) {
                    clearInterval(upgradePollerRef.current)
                    upgradePollerRef.current = null
                  }
                }
              }
            }, 3000)
          }
          return prev
        })
      }
    })

    const offDaemonError = EventsOn('daemon:error', (msg: string) => {
      setDaemonError(msg)
    })

    // Plugin settings (PLUG-03): initial fetch + change subscription. The
    // PluginsSection component does its own fetch on mount; this top-level
    // fetch ensures App-state has the value for the prop drill before any
    // TerminalPanel mounts. Failure leaves pluginConfig at null (toggles in
    // PluginsSection surface the error via its own loadError state).
    GetPluginSettings()
      .then((s) => setPluginConfig(s))
      .catch(() => { /* keep null; PluginsSection surfaces its own error */ })

    const offPlugins = EventsOn('settings:plugins', (s: PluginSettings) => {
      setPluginConfig(s)
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

    const offExit = EventsOn(
      'session:exit',
      (data: {
        sessionId: string
        exitCode: number
        sessionName: string
        cli: string
        duration: number
        finalStatus: string
      }) => {
        // Always record the exit (shows toast)
        const exitState: ExitState = {
          sessionId: data.sessionId,
          sessionName: data.sessionName,
          cli: data.cli,
          exitCode: data.exitCode,
          duration: data.duration,
          finalStatus: data.finalStatus,
          countdown: data.exitCode === 0 ? 5 : -1,
          cancelled: false,
        }
        setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))

        // Only start auto-close countdown for clean exits (D-10) when enabled (D-11)
        if (data.exitCode === 0) {
          if (autoCloseRef.current) {
            const timer = setInterval(() => {
              setSessionExits(prev => {
                const entry = prev[data.sessionId]
                if (!entry || entry.cancelled) {
                  clearInterval(timer)
                  delete countdownTimers.current[data.sessionId]
                  return prev
                }
                if (entry.countdown <= 1) {
                  clearInterval(timer)
                  delete countdownTimers.current[data.sessionId]
                  // Auto-close the tab
                  void handleCloseTabRef.current?.(data.sessionId)
                  const { [data.sessionId]: _, ...rest } = prev
                  return rest
                }
                return {
                  ...prev,
                  [data.sessionId]: { ...entry, countdown: entry.countdown - 1 },
                }
              })
            }, 1000)
            countdownTimers.current[data.sessionId] = timer
          } else {
            // Auto-close disabled: show toast without countdown
            setSessionExits(prev => ({
              ...prev,
              [data.sessionId]: { ...prev[data.sessionId], countdown: -1 },
            }))
          }
        }
      }
    )

    // Subscribe to quit-requested event from Go backend (Phase 85)
    const offQuit = EventsOn('app:quit-requested', () => {
      setShowQuitModal(prev => {
        if (prev) return prev  // Ignore if modal already showing (double-fire guard)
        ListSessions().then(sessions => {
          const activeSessions = sessions
            .filter((s: SessionInfo) => s.state !== 'stopped')
            .map((s: SessionInfo) => ({ id: s.id, name: s.name || s.cli, status: s.state || 'running' }))
          setQuitSessions(activeSessions)
          setShowQuitModal(true)
        }).catch(() => {
          setQuitSessions([])
          setShowQuitModal(true)
        })
        return prev
      })
    })

    return () => {
      offStatus()
      offHealth()
      offDaemonError()
      offPlugins()
      cancelTrayFocus()
      offExit()
      offQuit()
      // Clear all countdown timers
      Object.values(countdownTimers.current).forEach(clearInterval)
      countdownTimers.current = {}
      if (upgradePollerRef.current !== null) {
        clearInterval(upgradePollerRef.current)
        upgradePollerRef.current = null
      }
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

      // SEC-01 / D-06: new sessions start with web-sharing OFF. The user must
      // explicitly toggle web on to share. The daemon enforces this at the
      // handleCreateSession layer (TestHandleCreateSession_NoAutoEnable); the
      // previous auto-seed here created a UI-daemon state mismatch that made
      // the Sessions tab show bogus "WEB ON" state with broken URLs.
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
    // For naturally-exited sessions (in sessionExits), skip immediate disable —
    // the daemon's exit watcher already started a 10-second grace period (D-12)
    // so web viewers can see the final output before serving stops.
    if (webEnabled[id] && !sessionExits[id]) {
      try { await ToggleWebServing(id, false) } catch (_) { /* ignore */ }
      setWebEnabled((prev) => { const n = { ...prev }; delete n[id]; return n })
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
    // Clean up exit state and countdown timer (Phase 84)
    if (countdownTimers.current[id]) {
      clearInterval(countdownTimers.current[id])
      delete countdownTimers.current[id]
    }
    setSessionExits(prev => { const n = { ...prev }; delete n[id]; return n })
  }, [activeId, webEnabled, sessionExits])
  // Keep the ref in sync so the session:exit event handler ([] deps) always calls the latest version
  handleCloseTabRef.current = handleCloseTab

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

  const handleKeepOpen = useCallback((sessionId: string) => {
    // Cancel countdown (D-02)
    if (countdownTimers.current[sessionId]) {
      clearInterval(countdownTimers.current[sessionId])
      delete countdownTimers.current[sessionId]
    }
    setSessionExits(prev => {
      const entry = prev[sessionId]
      if (!entry) return prev
      return { ...prev, [sessionId]: { ...entry, cancelled: true, countdown: -1 } }
    })
  }, [])

  const handleDismissExit = useCallback((sessionId: string) => {
    // Remove toast entry
    if (countdownTimers.current[sessionId]) {
      clearInterval(countdownTimers.current[sessionId])
      delete countdownTimers.current[sessionId]
    }
    setSessionExits(prev => {
      const { [sessionId]: _, ...rest } = prev
      return rest
    })
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

  // Lift update:available subscription from WelcomeTab (Phase 81 D-06)
  useEffect(() => {
    GetLastUpdateInfo()
      .then((info) => { if (info) setUpdate(info) })
      .catch(() => {})
    const offUpdate = EventsOn('update:available', (info: UpdateInfo) => {
      setUpdate(info)
    })
    return () => { offUpdate() }
  }, [])

  // Reset dismissed state when entering local mode (D-04: reappear if conditions change)
  useEffect(() => {
    if (webServerMode === 'local') {
      setLocalBannerDismissed(false)
    }
  }, [webServerMode])

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
      const [port, clis, sessions, running, health] = await Promise.all([
        GetRelayPort(),
        DetectCLIs(),
        ListSessions(),
        IsWebServerRunning(),
        GetTailscaleStatus(),
      ])
      setRelayPort(port)
      setDetectedCLIs(clis)
      setWebServerRunning(running)
      setTailscaleHealth(health)

      // Phase 101-02 (SHELL-01 GUI half) — re-discover shells on daemon retry.
      setShellsLoading(true)
      ListShells()
        .then((s) => setDetectedShells(s ?? []))
        .catch(() => setDetectedShells([]))
        .finally(() => setShellsLoading(false))

      // Fetch web server mode for local-network-fallback UI
      GetWebServerMode().then(mode => {
        setWebServerMode(mode === 'tailscale' || mode === 'local' ? mode : null)
      }).catch(() => setWebServerMode(null))

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

        // Seed webEnabled state from daemon's SessionInfo.webEnabled field (SERVE-02 restore).
        if (running) {
          const enabledMap: Record<string, boolean> = {}
          sessions.forEach((s) => {
            if (s.webEnabled) {
              enabledMap[s.id] = true
            }
          })
          if (Object.keys(enabledMap).length > 0) {
            setWebEnabled(enabledMap)
          }
        }
      }
    } catch (err) {
      console.error('[App] retry init failed:', err)
      setDaemonError(String(err))
    }
  }, [])

  return (
    <div className="app">
      {((webServerMode === 'local' && !localBannerDismissed) ||
        update ||
        ((webglContextLost || webglSoftwareDetected) && !webglBannerDismissed) ||
        saveBanner !== null ||
        pluginToggleBanners.length > 0) && (
        <div className="banner-stack">
          {webServerMode === 'local' && !localBannerDismissed && (
            <LocalNetworkBanner
              visible={true}
              tailscaleConnected={!!(tailscaleHealth?.connected && tailscaleHealth?.hasCerts && tailscaleHealth?.ip)}
              tailscaleInstalled={!!(tailscaleHealth?.installed || detectedCLIs.some(c => c.Name === 'tailscale'))}
              tailscaleBinaryFound={!!(tailscaleHealth?.binaryFound)}
              tailscaleDaemonUp={!!(tailscaleHealth?.daemonUp)}
              platformHint={tailscaleHealth?.platformHint || ''}
              onOpenURL={BrowserOpenURL}
              onDismiss={handleDismissLocalBanner}
              className={localBannerExiting ? 'banner-exit' : undefined}
            />
          )}
          {update && (
            <UpdateBanner
              update={update}
              onDismiss={handleDismissUpdate}
              className={updateExiting ? 'banner-exit' : undefined}
            />
          )}
          {(webglContextLost || webglSoftwareDetected) && !webglBannerDismissed && (
            <WebGLRecoveryBanner
              reason={webglSoftwareDetected ? 'software-rasterized' : 'context-loss'}
              onDismiss={() => setWebglBannerDismissed(true)}
            />
          )}
          {/* Phase 97 SER-01: save-feedback banner (info = Serialize disabled,
              error = write/dialog failure). Mirrors localBanner one-shot pattern. */}
          {saveBanner && (
            <div
              className={saveBanner.kind === 'error' ? 'banner banner--error' : 'banner banner--info'}
              role="status"
            >
              <span>{saveBanner.text}</span>
              <button onClick={() => setSaveBanner(null)} aria-label="Dismiss">×</button>
            </div>
          )}
          {/* Phase 99 PUI-02: one-shot toggle-change banners for unicode11 / image.
              Both auto-dismiss after 6000ms; user can also dismiss via × button. */}
          {pluginToggleBanners.map((kind) => (
            <PluginToggleBanner
              key={kind}
              kind={kind}
              onDismiss={() =>
                setPluginToggleBanners((prev) => prev.filter((k) => k !== kind))
              }
            />
          ))}
        </div>
      )}
      <div className="app__row">
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
          exitCountdowns={
            Object.fromEntries(
              Object.entries(sessionExits)
                .filter(([, e]) => e.exitCode === 0 && !e.cancelled && e.countdown > 0)
                .map(([id, e]) => [id, e.countdown])
            )
          }
          onRequestSave={handleRequestSave}
          tabProgress={tabProgress}
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
            webServerMode={webServerMode}
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
        <div style={{ display: activeId === SETTINGS_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
          <SettingsTab
            clis={detectedCLIs}
            tailscaleHealth={tailscaleHealth}
            webServerMode={webServerMode}
            selectedTheme={terminalThemeName}
            onThemeChange={handleThemeChange}
            onPluginToggleSideEffect={handlePluginToggleSideEffect}
            onWebServerStateChange={async () => {
              try {
                const running = await IsWebServerRunning()
                setWebServerRunning(running)
                if (!running) {
                  setWebServerMode(null)
                } else {
                  const mode = await GetWebServerMode()
                  setWebServerMode(mode === 'tailscale' || mode === 'local' ? mode : null)
                }
              } catch (_) { /* ignore */ }
            }}
          />
        </div>
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
                  theme={terminalTheme}
                  pluginConfig={pluginConfig}
                  onWebGLContextLost={handleWebGLContextLost}
                  onRegisterSaver={handleRegisterSaver}
                  onProgressChange={handleProgressChange}
                />
                {sessionExits[tab.sessionId] && sessionExits[tab.sessionId].exitCode === 0 && !sessionExits[tab.sessionId].cancelled && sessionExits[tab.sessionId].countdown > 0 && (
                  <ExitCountdownBanner
                    countdown={sessionExits[tab.sessionId].countdown}
                    onKeepOpen={() => handleKeepOpen(tab.sessionId)}
                  />
                )}
                <StatusBar
                  sessionId={tab.sessionId}
                  webServerRunning={webServerRunning}
                  webEnabled={!!webEnabled[tab.sessionId]}
                  onToggleWeb={() => void handleToggleWeb(tab.sessionId)}
                />
              </div>
            )
          })}
        </div>
      </div>
      </div>{/* end app__row */}

      {showNewSessionModal && (
        <NewSessionModal
          isOpen={showNewSessionModal}
          clis={detectedCLIs}
          shells={detectedShells}
          shellsLoading={shellsLoading}
          onConfirm={(cli, workDir, args) => {
            setShowNewSessionModal(false)
            void createTab(cli, workDir, args)
          }}
          onClose={() => setShowNewSessionModal(false)}
        />
      )}

      {showQuitModal && (
        <QuitConfirmModal
          isOpen={showQuitModal}
          sessions={quitSessions}
          onQuitGUI={() => { setShowQuitModal(false); void QuitGUIOnly() }}
          onQuitAll={() => { setShowQuitModal(false); void QuitAll() }}
          onCancel={() => setShowQuitModal(false)}
        />
      )}
      <ExitToast
        exits={sessionExits}
        onKeepOpen={handleKeepOpen}
        onDismiss={handleDismissExit}
      />
    </div>
  )
}

export default App
