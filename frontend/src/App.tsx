import React, { useEffect, useState, useCallback, useRef, useMemo } from 'react'
import type { IProgressState } from '@xterm/addon-progress'
import * as xtermThemes from 'xterm-theme'
import type { ITheme } from '@xterm/xterm'
import { TabBar, type Tab } from './components/TabBar'
import { Sidebar } from './components/Sidebar'
import { SettingsTab } from './components/SettingsTab'
import { stripAnsi } from './lib/stripAnsi'
import { sanitizeFilename } from './lib/sanitizeFilename'
import { detectMode, readWebModeParams } from './lib/webMode'
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
  GetRemoteSessionsWithMeta,
  GetWebServerMode,
  NotifyThemeChange,
  GetLastUpdateInfo,
  GetAutoCloseSession,
  GetStayOnHubAfterCreate,
  GetPluginSettings,
  GetShellWebShareWarned,
  SetShellWebShareWarned,
  GetShellWebShareWarningEnabled,
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
import { FileBrowserTab, fileBrowserTabId } from './components/FileBrowserTab'
import { RemoteJoinCodeModal } from './components/RemoteJoinCodeModal'
import { HubPanel } from './components/Hub/HubPanel'
import { SessionShareModal } from './components/Hub/SessionShareModal'
import { HelpTab } from './components/HelpTab'
import { EnableWebSharingTakeover } from './components/FileBrowser/EnableWebSharingTakeover'
import { findRemoteSession, remoteBaseURLFor } from './lib/remoteSession'
import { adaptAllRemoteSessions } from './lib/remoteAdapter'
import type { AdaptedRemoteSessionInfo } from './lib/remoteAdapter'
import { ExchangeJoinCodeAtURL, RegisterRemoteCap, OpenRemoteSessionURL } from './wailsjs/go/main/App'
import { LocalNetworkBanner } from './components/LocalNetworkBanner'
import { WebShareSessionView } from './components/Hub/WebShareSessionView'
import { TerminalChatHost } from './components/Hub/TerminalChatHost'
import { RemoteBrowseDNSWarning } from './components/RemoteBrowseDNSWarning'
import { UpdateBanner } from './components/UpdateBanner'
import type { UpdateInfo } from './components/UpdateBanner'
import { WebGLRecoveryBanner } from './components/WebGLRecoveryBanner'
import { PluginToggleBanner } from './components/PluginToggleBanner'
import { ExitToast } from './components/ExitToast'
import type { ExitState } from './components/ExitToast'
import { ExitCountdownBanner } from './components/ExitCountdownBanner'
import { QuitConfirmModal } from './components/QuitConfirmModal'
import { ALLOWED_THEMES } from './themes'
import {
  loadGroups,
  createGroup,
  assignToGroup,
  removeFromGroup,
  type HubGroupDef,
} from './lib/hubGroups'
import type { GroupCounts } from './lib/hubGroupCounts'

const DEFAULT_FONT_SIZE = 14
const THEME_STORAGE_KEY = 'agenthub:terminalTheme'
const DEFAULT_THEME_NAME = 'Tomorrow_Night'
const UI_THEME_STORAGE_KEY = 'agenthub:uiTheme'

// Phase 101-03 (SHELL-07/SHELL-08) — shell cli identifiers that route through
// the one-time security-warning banner now live in lib/shellCli (imported at
// top). Phase 150 SET-01 gap-closure: extracted to that shared module (the
// predicted "third call-site" — Hub SessionShareModal — now exists) and made
// path-aware, because a shell session's cli is its full path ('/bin/zsh'), not
// a bare name. Must stay in sync with the daemon's isShellSession check
// (internal/daemon/engine.go).

/**
 * App is the root component — it owns all tab state and wires
 * the Wails-generated TypeScript bindings to the child components.
 */
function App(): React.ReactElement {
  const WELCOME_TAB: Tab = { id: '__welcome__', name: 'Welcome', sessionId: '', cli: '', type: 'welcome' }
const SETTINGS_TAB: Tab = { id: '__settings__', name: 'Settings', sessionId: '', cli: '', type: 'settings' }
  // Phase 131 — Hub top-level surface tab.
  const HUB_TAB: Tab = { id: '__hub__', name: 'Hub', sessionId: '', cli: '', type: 'hub' }
  // Phase 147 — In-app Help page tab.
  const HELP_TAB: Tab = { id: '__help__', name: 'Help', sessionId: '', cli: '', type: 'help' }
  // Phase 120-06 — single source of truth for "is this React shell running in
  // a regular browser under /app/ vs inside the Wails desktop runtime?" Used
  // to gate the Wails RPC suite (init, retryInit, sessions polls) so the SPA
  // can be served to web-share viewers without crashing on Wails-only calls.
  // See 120-VERIFICATION.md Human Verification #2 + lib/webMode.ts.
  const mode = detectMode()
  // webParams is captured once on first mount — URL params don't change for
  // the lifetime of the SPA. useMemo keeps the reference stable so dependency
  // arrays of downstream effects never see false re-evaluations.
  const webParams = useMemo(() => readWebModeParams(), [])
  // Phase 159-04 (WEBCHAT-05) — in web mode, whether the guest's cap grants
  // files.read (resolved from the /api/sessions/{id}/info probe in the web
  // bootstrap). Gates the TabBar session-menu chevron so a file-enabled guest
  // can re-open the file browser, while a no-files guest gets no menu at all.
  const [webFilesEnabled, setWebFilesEnabled] = useState(false)
  // Phase 120-06 — web-mode initial state: no Welcome tab, no Sidebar focus.
  // The auto-open effect below will populate the file-browser tab from the
  // ?session= URL param. Starting empty keeps WelcomeTab (which calls
  // GetVersion + references the absolute /agenthub-title-logo.png path that
  // 404s under /app/) from briefly mounting before the auto-open fires.
  const initialTabs: Tab[] = mode === 'web' ? [] : [WELCOME_TAB]
  const [tabs, setTabs] = useState<Tab[]>(initialTabs)
  const [activeId, setActiveId] = useState<string | null>(
    mode === 'web' ? null : WELCOME_TAB.id,
  )
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
  // Phase 101-03 SHELL-07/SHELL-08 — one-time security-warning gate for shell
  // sessions. `shellWebShareWarned` is hydrated from the daemon settings via
  // GetShellWebShareWarned() on mount; once the user confirms the banner, both
  // local state and the daemon-persisted flag flip to true permanently (per-
  // machine, not per-session). Phase 168-05: the in-flight pending state now
  // lives inside SessionShareModal (pendingShellShare, local) — the App-level
  // pendingShellWebToggle used only by the retired footer-direct-toggle path.
  const [shellWebShareWarned, setShellWebShareWarned] = useState(false)
  // Phase 150 SET-01 — master warning-enabled switch (default true = ON per D-08).
  // Hydrated from daemon on mount; safe-degradation default is true (warning shows)
  // so that a daemon disconnect never silently disables the security guardrail.
  const [shellWebShareWarningEnabled, setShellWebShareWarningEnabled] = useState(true)
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
    acceptDns?: boolean
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
  // Phase 131 — Hub sessions and error state (polled when the Hub tab is active)
  const [hubSessions, setHubSessions] = useState<SessionInfo[]>([])
  const [hubError, setHubError] = useState(false)
  // Phase 168-05 (UX-02) — Share modal open-state, lifted from HubPanel so both the Hub
  // card click AND the footer "Share Session" button drive a single SessionShareModal
  // instance (RESEARCH Pattern 4). null = closed; non-null = open for this session.
  const [shareModalSession, setShareModalSession] = useState<SessionInfo | null>(null)
  // Remote peers for Hub remote cards (polled when the Hub tab is active — Phase 138 T-138-08)
  const [remotePeers, setRemotePeers] = useState<RemotePeerSessions[]>([])
  // WR-01: tracks whether the remote-sessions poll has completed at least
  // once. A ref (not a closed-over remotePeers snapshot) so the spinner gate
  // reflects the current fetch rather than a stale render-time value frozen
  // across the 30s interval's lifetime.
  const remoteHasLoadedRef = useRef(false)

  // Phase 122-03 — remote-session file-browse state.
  //   remoteCapsCached: sessionIds whose caps are already deposited in the
  //     local daemon's RemoteCapStore. Per locked decision D-03, the modal is
  //     skipped for cached sessions.
  //   joinModalForSession: the session the paste-join-code modal is open for,
  //     or null when no modal is showing.
  const [remoteCapsCached, setRemoteCapsCached] = useState<Set<string>>(
    () => new Set<string>(),
  )
  const [joinModalForSession, setJoinModalForSession] = useState<{
    id: string
    name: string
    hostname: string
    intent?: 'files' | 'hub-modal' | 'open-session'
    /** Pre-computed for open-session intent; avoids re-deriving in the modal exchange handler. */
    baseURL?: string
  } | null>(null)

  // Phase 134 — holds the HubPanel's cap-acquired callback (registered via onRegisterCapAcquired).
  // Using a ref avoids stale closure issues in the modal exchange callback (which has handleOpenFileBrowser dep).
  const capAcquiredRef = useRef<((sessionId: string) => void) | null>(null)

  // Phase 134 — WR-01: holds HubPanel's cap-cancelled callback (registered via onRegisterCapCancelled).
  // Invoked from RemoteJoinCodeModal onClose (hub-modal intent path only) so a dismissed modal
  // resets HubPanel's pendingModalSessionId / pendingSourceRectRef and does not strand pending state.
  const capCancelledRef = useRef<(() => void) | null>(null)

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
  // Stay-on-hub-after-create setting (Phase 168 UX-01): loaded on mount,
  // default false (auto-switch, today's behavior). Stored as a ref (not
  // state) — same rationale as autoCloseRef, avoids stale closures inside
  // createTab without needing to add it to the useCallback deps array.
  const stayOnHubAfterCreateRef = useRef(false)
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

  // UI theme (light/dark whole-app appearance) — distinct from terminal color theme.
  // Default is dark (attribute absent). Light is opt-in via localStorage.
  const [uiTheme, setUiTheme] = useState<'dark' | 'light'>(() =>
    localStorage.getItem(UI_THEME_STORAGE_KEY) === 'light' ? 'light' : 'dark'
  )
  useEffect(() => {
    if (uiTheme === 'light') {
      document.documentElement.setAttribute('data-ui-theme', 'light')
      document.documentElement.style.colorScheme = 'light'
    } else {
      document.documentElement.removeAttribute('data-ui-theme')
      document.documentElement.style.colorScheme = 'dark'
    }
  }, [uiTheme])
  const handleUiThemeChange = useCallback((t: 'dark' | 'light') => {
    localStorage.setItem(UI_THEME_STORAGE_KEY, t)
    setUiTheme(t)
  }, [])

  // POL-05: Group state lifted from HubPanel to App.tsx
  // groupDefs + activeGroupId drive both Sidebar (render+select+create+drag-drop)
  // and HubPanel (filtering). Counts flow UP from HubPanel via onGroupCountsChange callback
  // (allSessions is NOT lifted — HubPanel keeps its own polling).
  const [groupDefs, setGroupDefs] = useState<HubGroupDef[]>(() => loadGroups())
  const [activeGroupId, setActiveGroupId] = useState<string | null>(null)
  const [groupCounts, setGroupCounts] = useState<Record<string, GroupCounts>>({})
  const [globalGroupCounts, setGlobalGroupCounts] = useState<GroupCounts>(
    { running: 0, total: 0, attention: 0, waiting: 0 }
  )

  const handleGroupSelect = useCallback((id: string | null) => {
    setActiveGroupId(id)
  }, [])

  const handleCreateGroup = useCallback((name: string) => {
    setGroupDefs((prev) => createGroup(prev, name))
  }, [])

  const handleDropOnGroup = useCallback((groupId: string, key: string) => {
    setGroupDefs((prev) =>
      groupId === '__other__' ? removeFromGroup(prev, key) : assignToGroup(prev, groupId, key)
    )
  }, [])

  const handleGroupCountsChange = useCallback(
    (counts: Record<string, GroupCounts>, global: GroupCounts) => {
      setGroupCounts(counts)
      setGlobalGroupCounts(global)
    },
    []
  )

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
      // Phase 120-06 — skip the entire Wails RPC suite when running in web
      // mode (regular browser under /app/?session=…&cap=…). Wails RPCs are
      // unreachable from a browser; session+cap come from URL params and the
      // file-browser tab is auto-opened by a separate effect below that runs
      // AFTER handleOpenFileBrowser is declared. We return early here before
      // any Wails-bound call to avoid the partially-functional shell flagged
      // in 120-VERIFICATION.md Human Verification #2.
      if (mode === 'web') {
        return
      }
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
        // Phase 168-05 (UX-02): seed hubSessions from the same ListSessions() call used
        // for tab restoration, so openShareModalForActiveSession works even before the
        // user ever visits the Hub tab (the 3s Hub poll below only runs while Hub is
        // active — SESS-02 restore can land the user directly on a session tab).
        setHubSessions(sessions)

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

        // Load stay-on-hub-after-create preference (Phase 168 UX-01)
        GetStayOnHubAfterCreate().then(val => { stayOnHubAfterCreateRef.current = val }).catch(() => {})

        // Phase 101-03 SHELL-08 — hydrate the one-time shell-web-share-warning
        // flag from daemon-persisted settings. On failure default to false so
        // the banner re-shows on the next toggle attempt (safe degradation —
        // worst case the user re-confirms once).
        GetShellWebShareWarned()
          .then((v) => setShellWebShareWarned(v))
          .catch((err) => {
            console.warn('[App] GetShellWebShareWarned failed:', err)
            setShellWebShareWarned(false)
          })

        // Phase 150 SET-01 — hydrate the warning-enabled master switch from daemon.
        // On failure default to true (safe degradation: warning stays ON per D-08).
        GetShellWebShareWarningEnabled()
          .then((v) => setShellWebShareWarningEnabled(v))
          .catch((err) => {
            console.warn('[App] GetShellWebShareWarningEnabled failed:', err)
            setShellWebShareWarningEnabled(true) // default ON (safe degradation per D-08)
          })

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
      acceptDns?: boolean
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
        // SHELL-12: on clean exit (code 0), respect the user's "Auto-close tab
        // on exit" preference (autoCloseRef, loaded from GetAutoCloseSession()).
        // When auto-close is ON (default): close tab immediately — no toast, no
        // countdown. When OFF: fall through to setSessionExits so the ExitToast
        // appears, honoring the user's explicit preference.
        // The daemon (107-02) normalizes PTY -1 → 0, so this branch covers all
        // natural-exit cases including shell EOF.
        if (data.exitCode === 0) {
          if (autoCloseRef.current) {
            // Auto-close enabled: close tab immediately (SHELL-12 default path)
            void handleCloseTabRef.current?.(data.sessionId)
            return
          }
          // Auto-close disabled: fall through to show ExitToast (no countdown)
        }
        // Non-zero exit (or zero-exit with auto-close OFF): record exit state
        // and show ExitToast (existing behavior).
        const exitState: ExitState = {
          sessionId: data.sessionId,
          sessionName: data.sessionName,
          cli: data.cli,
          exitCode: data.exitCode,
          duration: data.duration,
          finalStatus: data.finalStatus,
          countdown: -1,
          cancelled: false,
        }
        setSessionExits(prev => ({ ...prev, [data.sessionId]: exitState }))
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
      // Phase 168 UX-01: when stayOnHubAfterCreate is ON, skip the
      // auto-switch so the user stays on the Hub after creating a session.
      // The tab is still created unconditionally (D-10) — only this single
      // setActiveId call (the only auto-switch in the app, D-11) is gated.
      if (!stayOnHubAfterCreateRef.current) {
        setActiveId(sessionId)
      }

      // SEC-01 / D-06: new sessions start with web-sharing OFF. The user must
      // explicitly toggle web on to share. The daemon enforces this at the
      // handleCreateSession layer (TestHandleCreateSession_NoAutoEnable); the
      // previous auto-seed here created a UI-daemon state mismatch that made
      // the removed Sessions page show bogus "WEB ON" state with broken URLs.
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

  // Phase 147 — find-or-add help tab (mirrors handleOpenSettings exactly).
  // Phase 166 FUI-06 — optional sectionId scrolls the Help tab to a specific section
  // (e.g. 'help-sharing' from the Funnel risk-panel cross-link). HelpTab is mounted with
  // a display-toggle (not conditional), so the section element exists in the DOM; defer
  // the scroll until the tab is visible.
  const handleOpenHelp = useCallback((sectionId?: string) => {
    const existing = tabs.find((t) => t.type === 'help')
    if (existing) {
      setActiveId(existing.id)
    } else {
      setTabs((prev) => [...prev, HELP_TAB])
      setActiveId(HELP_TAB.id)
    }
    if (sectionId) {
      setTimeout(() => {
        document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }, 50)
    }
  }, [tabs])

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

  // Phase 168-05 (UX-02 / D-14): the footer no longer toggles web sharing
  // directly — it opens the (lifted) Share modal for the active session.
  // Toggling now happens exclusively inside SessionShareModal, which owns
  // its own self-contained shell-warning gate (SET-01, D-10 cross-surface
  // parity) — the App-level pendingShellWebToggle/ShellWebShareBanner path
  // that used to back the footer's direct toggle is retired below.
  const openShareModalForActiveSession = useCallback(() => {
    const session = hubSessions.find((s) => s.id === activeId)
    if (session) setShareModalSession(session)
  }, [hubSessions, activeId])

  // Phase 101-03 — banner confirm: per RESEARCH §8 race mitigation, set the
  // local shellWebShareWarned flag SYNCHRONOUSLY (before awaiting either RPC).
  // If a second shell session's toggle fires while the SetShellWebShareWarned
  // disk write is still in flight, the second toggle's interception check
  // will already see the updated local state and skip the banner.
  // Phase 168-05: sessionId now derives from shareModalSession (the single
  // lifted modal-open state, D-14) instead of the retired pendingShellWebToggle
  // — the modal is the only surface that can trigger this confirm.
  const handleShellWebShareConfirm = useCallback(async () => {
    if (!shareModalSession) return
    const sessionId = shareModalSession.id
    // Race mitigation (RESEARCH §8): set local "warned" flag synchronously
    // before any await so a fast double-toggle doesn't re-show the banner.
    // The banner itself stays mounted during the await so its "Enabling…"
    // transient state + aria-busy can render (CR-01 fix).
    setShellWebShareWarned(true)
    try {
      await Promise.all([
        SetShellWebShareWarned(true),
        ToggleWebServing(sessionId, true),
      ])
      setWebEnabled((prev) => ({ ...prev, [sessionId]: true }))
    } catch (err) {
      console.warn('[App] shell web-share confirm failed:', err)
      // Best-effort rollback: clear the banner and let the user retry. Avoid
      // leaving local "warned" true if the persist call failed — the next
      // toggle will re-prompt rather than silently downgrade.
      setShellWebShareWarned(false)
    }
  }, [shareModalSession])

  // Phase 168-05: the modal owns its own pendingShellShare local state and
  // already resets it before calling this — no App-level state to clear.
  const handleShellWebShareCancel = useCallback(() => {}, [])

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

  // Phase 131 — Poll sessions when the Hub tab is active (mirrors daemon-manager poll pattern).
  // T-131-10: early-return when not active prevents DoS from polling while Hub is inactive.
  // WR-02: reset hubError immediately on Hub activation so a stale error banner from a
  // prior visit does not persist when the user returns to a healthy Hub. hubSessions is NOT
  // reset here to avoid a flash-to-empty — the first refresh() call populates it promptly.
  useEffect(() => {
    if (mode === 'web') return // Phase 120-06: no Wails RPC in browser mode.
    if (activeId !== HUB_TAB.id) return
    setHubError(false) // WR-02: clear any stale error state from the previous visit
    let cancelled = false
    async function refresh() {
      try {
        const sessions = await ListSessions()
        if (!cancelled) { setHubSessions(sessions); setHubError(false) }
      } catch {
        if (!cancelled) setHubError(true)
      }
    }
    void refresh()
    const interval = setInterval(() => void refresh(), 3000)
    return () => { cancelled = true; clearInterval(interval) }
  }, [activeId])

  // Phase 168-05 (UX-02, RESEARCH Pitfall 3 / Pattern 4): keep the lifted Share modal's
  // session prop in sync with hubSessions. shareModalSession is a snapshot taken at
  // open-time (either from a Hub card click or the footer's openShareModalForActiveSession);
  // without this effect a server-side flip (e.g. funnelActive completing, viewerCount
  // changing) never reaches an already-open modal. This is the single sync instance —
  // SessionShareModal itself is now rendered once, here, at the App.tsx level (not inside
  // HubPanel, which unmounts on non-Hub tabs — the footer button must work from any local
  // session tab). Keyed on shareModalSession?.id (not the whole object) to avoid re-running
  // from its own setState.
  useEffect(() => {
    if (!shareModalSession) return
    const updated = hubSessions.find((s) => s.id === shareModalSession.id)
    if (updated && updated !== shareModalSession) {
      setShareModalSession(updated)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hubSessions, shareModalSession?.id])

  // Poll remote sessions when the Hub tab is active (Phase 138 — only the
  // HUB_TAB guard is retained; Hub still needs remote data — T-138-08).
  // WR-01: the spinner gate uses remoteHasLoadedRef (a ref) rather than the
  // closed-over remotePeers snapshot, so setRemoteLoading(true) fires only
  // on the genuine first load and not on every 30s poll. mode and remotePeers
  // are intentionally omitted from deps (mode is mount-stable; the loaded
  // gate is now ref-based), matching the eslint-disable on the sibling
  // mount-effect below.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (mode === 'web') return // Phase 120-06: no Wails RPC in browser mode.
    if (activeId !== HUB_TAB.id) return
    let cancelled = false
    async function refresh() {
      try {
        const peers = await GetRemoteSessionsWithMeta()
        if (!cancelled) {
          setRemotePeers(peers ?? [])
          remoteHasLoadedRef.current = true
        }
      } catch {
        // Remote poll errors are silently absorbed — Hub cards simply remain
        // stale until the next 30s poll succeeds. Error display is handled by
        // the Hub's existing empty-state messaging.
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
    if (mode === 'web') return // Phase 120-06: update poller is Wails-only.
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

  // Phase 155-03 — stable tab id for the web-session view. One view per
  // session (find-or-focus semantics, same pattern as fileBrowserTabId).
  const webSessionTabId = (sessionId: string) => `__websession__${sessionId}`

  // Phase 155-03 — open (or focus) the WebShareSessionView tab for a web-share
  // session. Mirrors handleOpenFileBrowser in structure.
  //
  // Phase 168-03 (FIX-03) — extended to (sessionId, baseURL?, capToken?). The
  // app's own web-share bootstrap tab (mode==='web') still omits baseURL/
  // capToken and is resolved from the mount-stable webParams at render time
  // (unchanged behavior). Remote-peer tabs (opened via the remote-open action)
  // pass a baseURL + capToken that are carried PER-TAB on the Tab object, so
  // two different remote sessions never share params (RESEARCH Pitfall 3).
  const openWebSessionTab = useCallback((sessionId: string, baseURL?: string, capToken?: string) => {
    const tabId = webSessionTabId(sessionId)
    const existing = tabs.find((t) => t.id === tabId)
    if (existing) {
      setActiveId(existing.id)
      return
    }
    const newTab: Tab = {
      id: tabId,
      name: 'Session',
      sessionId,
      cli: '',
      type: 'web-session',
      baseURL,
      capToken,
    }
    setTabs((prev) => [...prev, newTab])
    setActiveId(newTab.id)
  }, [tabs])

  // Phase 120-04 UI-01 — per-session FileBrowserTab find-or-add. Opens the
  // file browser for a session, either focusing the existing tab if one is
  // open or creating a new one keyed by fileBrowserTabId(sessionId).
  // `activate` defaults true (focus the tab). Phase 159-03 (WEBCHAT-04) passes
  // false so the web-share bootstrap can open the file-browser tab in the
  // BACKGROUND — the WebShareSessionView (terminal + chat) stays the active tab.
  const handleOpenFileBrowser = useCallback((sessionId: string, sessionName: string, activate = true) => {
    const tabId = fileBrowserTabId(sessionId)
    const existing = tabs.find((t) => t.id === tabId)
    if (existing) {
      if (activate) setActiveId(existing.id)
      return
    }
    const newTab: Tab = {
      id: tabId,
      name: `${sessionName} — Files`,
      sessionId,
      cli: '',
      type: 'file-browser',
    }
    setTabs((prev) => [...prev, newTab])
    if (activate) setActiveId(newTab.id)
  }, [tabs])

  // Phase 131 UAT follow-up — re-attach to an already-running session.
  // Opens (or focuses) a terminal tab for an existing session id. Mirrors the
  // startup-restore path (SESS-02): a terminal tab is { id: sessionId, name,
  // sessionId, cli } with no `type`, and the Terminal component attaches to the
  // live PTY by sessionId. Wired to both the Hub cards and the Sessions panel so
  // a session created in another window (or whose tab was closed) can be reopened.
  // NOTE: the Hub surfaces this via an explicit "Open" button rather than a
  // whole-card click — card-click is reserved for the Phase 134 modal gesture.
  const handleOpenSessionTab = useCallback((sessionId: string, name: string, cli: string) => {
    const existing = tabs.find((t) => t.id === sessionId)
    if (existing) {
      setActiveId(existing.id)
      return
    }
    const newTab: Tab = {
      id: sessionId,
      name: name || cli,
      sessionId,
      cli,
    }
    setTabs((prev) => [...prev, newTab])
    setActiveId(newTab.id)
  }, [tabs])

  // Phase 120-06 / 155-03 — web-mode bootstrap: when the SPA loads under /app/
  // with a ?session=<id> param, open both the session view and the file-browser
  // tab. The session view (WebShareSessionView) is the active/primary tab.
  // The file-browser tab is opened first (background) so the session tab ends
  // up as the last setActiveId call and is therefore active on mount.
  // The cap cannot be decoded client-side, so the file tab is always opened and
  // PermissionDeniedTakeover handles a missing files.read perm (RESEARCH Pattern 5a).
  // Effect runs exactly once because mode + webParams are mount-stable.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (mode === 'web' && webParams.sessionId) {
      const sid = webParams.sessionId
      // The WebShareSessionView (terminal + chat) is the primary surface —
      // open it immediately as the active tab.
      openWebSessionTab(sid)
      // Phase 159-03 (WEBCHAT-04) — the file-browser tab is opened in the
      // BACKGROUND only when the cap actually grants files.read. The cap is
      // opaque client-side, but GET /api/sessions/{id}/info returns the
      // server-verified perms (same endpoint ChatPanel uses for read-only
      // state). Previously the tab was always opened, leaving a guest whose
      // share lacks file access staring at a dead "files.read permission
      // required" takeover. Fail-safe: on any error or missing perm, no file tab.
      const cap = webParams.capToken ?? ''
      const base = window.location.origin
      const ctrl = new AbortController()
      fetch(`${base}/api/sessions/${encodeURIComponent(sid)}/info?cap=${encodeURIComponent(cap)}`, {
        signal: ctrl.signal,
      })
        .then((r) => (r.ok ? r.json() : null))
        .then((info: { perms?: string } | null) => {
          const perms = (info?.perms ?? '').split(',').map((s) => s.trim())
          if (perms.includes('files.read')) {
            setWebFilesEnabled(true)
            handleOpenFileBrowser(sid, sid, false)
          }
        })
        .catch(() => {
          /* fail-safe: do not open a file-browser tab the guest can't use */
        })
      return () => ctrl.abort()
    }
  }, [])

  // Phase 146 FIX-03 (out-of-band redesign) — held-cap reuse (remoteCapsCached) +
  // modal fallback for depositing a cap.
  //
  // Phase 168-03 (D-17) — REVERSES the Phase 146 external-browser design: opening
  // a remote session now opens an in-app web-session tab (openWebSessionTab)
  // instead of BrowserOpenURL. OpenRemoteSessionURL is still called to get the
  // daemon-composed, SID-correct cap-bearing URL (WR-01 protection preserved —
  // the daemon builds the URL from its own RemoteCapStore entry, so the path SID
  // always matches the deposited cap); the URL is parsed (origin -> baseURL,
  // ?cap= -> capToken) and handed to openWebSessionTab, which carries those
  // values PER-TAB so a second remote session never reuses the first's cap/host.
  const handleOpenRemoteSession = useCallback(
    async (session: AdaptedRemoteSessionInfo): Promise<void> => {
      // Held-cap reuse: cap already held → build the open URL from the daemon
      // store and open it in-app (no modal).
      if (remoteCapsCached.has(session.id)) {
        try {
          const url = await OpenRemoteSessionURL(session.id)
          const parsed = new URL(url)
          openWebSessionTab(session.id, parsed.origin, parsed.searchParams.get('cap') ?? '')
          return
        } catch {
          // Stale/evicted cap — fall through to modal for self-heal.
        }
      }
      // No cap (or stale): open join-code modal for out-of-band exchange.
      const baseURL = remoteBaseURLFor(session)
      if (!baseURL) {
        setSaveBanner({ kind: 'error', text: 'Cannot open session — the remote peer URL is unavailable.' })
        return
      }
      setJoinModalForSession({
        id: session.id,
        name: session.name,
        hostname: session.hostname,
        intent: 'open-session',
        baseURL,
      })
    },
    [remoteCapsCached, setSaveBanner, openWebSessionTab],
  )

  // Phase 122-03 — remote-session file-browse entry point. If the cap is
  // already cached (D-03), open the file-browser tab immediately. Otherwise
  // open the join-code modal for the chosen session.
  const handleBrowseFilesRemote = useCallback(
    async (sessionId: string, sessionName: string) => {
      if (remoteCapsCached.has(sessionId)) {
        handleOpenFileBrowser(sessionId, sessionName)
        return
      }
      let remote = findRemoteSession(sessionId, remotePeers)
      // WR-02: a peer can drop out of remotePeers between the 30s poll that
      // populated the panel and the user clicking "Browse files". Rather than
      // silently no-op'ing (a reachable-looking button that does nothing),
      // re-poll once to self-heal a transient drop, then surface a banner if
      // the session is genuinely gone.
      if (!remote) {
        try {
          const peers = (await GetRemoteSessionsWithMeta()) ?? []
          setRemotePeers(peers)
          remote = findRemoteSession(sessionId, peers)
        } catch {
          /* fall through to the not-available banner below */
        }
      }
      if (!remote) {
        setSaveBanner({
          kind: 'error',
          text: 'Remote session is no longer available — refresh peers and try again.',
        })
        return
      }
      setJoinModalForSession({
        id: sessionId,
        name: sessionName,
        hostname: remote.hostname,
      })
    },
    [remoteCapsCached, remotePeers, handleOpenFileBrowser, setRemotePeers, setSaveBanner],
  )

  // Phase 122-03 — modal-exchange handler. Two-step: exchange join code for a
  // cap against the REMOTE peer's /join/exchange endpoint, then deposit the
  // cap into the local daemon's RemoteCapStore. On success, mark the session
  // as cap-cached and open the file-browser tab. The cap token never enters
  // React state (T-122-03-01).
  //
  // Phase 146 FIX-03 (out-of-band): 'open-session' branch runs BEFORE hub-modal/files.
  // For open-session: skip RegisterRemoteCap's file-browser path (T-146-04: cap goes
  // straight into the open URL, not into a stored-then-browsed file session).
  //
  // Phase 168-03 (D-17) — the open-session branch now opens the in-app web-session
  // tab (openWebSessionTab) rather than an external browser window.
  const handleModalExchange = useCallback(
    async (code: string): Promise<void> => {
      const pending = joinModalForSession
      if (!pending) throw new Error('no session pending')

      // Phase 146: open-session intent — use pre-computed baseURL, open cap-bearing URL.
      // GAP-146-A WR-01 fix (Plan 05): deposit the cap so RemoteCapStore holds the
      // SID-correct entry, then build the open URL via OpenRemoteSessionURL (daemon-
      // composed) — eliminating the mismatch-prone hand-built URL that could produce
      // /sessions/A with the cap for B when the clicked-session id mismatches cap SID.
      if (pending.intent === 'open-session') {
        const baseURL = pending.baseURL ?? ''
        if (!baseURL) throw new Error('session-gone')
        const cap = await ExchangeJoinCodeAtURL(baseURL, code)
        // Deposit the cap so the daemon's RemoteCapStore holds the keyed entry.
        await RegisterRemoteCap(pending.id, baseURL, cap)
        // Mark session as cap-cached so future opens reuse it (held-cap branch above).
        setRemoteCapsCached((prev) => {
          const next = new Set(prev)
          next.add(pending.id)
          return next
        })
        // Build the SID-correct cap-bearing URL from the daemon's stored entry
        // (keyed by the cap just deposited — daemon ensures path SID = lookup key),
        // then open it in-app (D-17).
        const url = await OpenRemoteSessionURL(pending.id)
        const parsed = new URL(url)
        openWebSessionTab(pending.id, parsed.origin, parsed.searchParams.get('cap') ?? '')
        return
      }

      const remote = findRemoteSession(pending.id, remotePeers)
      if (!remote) throw new Error('session-gone')
      const baseURL = remoteBaseURLFor(remote)
      // WR-05: remoteBaseURLFor returns '' when the peer-supplied url is
      // malformed/empty. Treat that as a recoverable session-gone error so
      // the modal renders user-facing copy instead of attempting an exchange
      // against an empty base URL.
      if (!baseURL) throw new Error('session-gone')
      const cap = await ExchangeJoinCodeAtURL(baseURL, code)
      await RegisterRemoteCap(pending.id, baseURL, cap)
      setRemoteCapsCached((prev) => {
        const next = new Set(prev)
        next.add(pending.id)
        return next
      })
      // Phase 134 — MODAL-06 intent discriminator: route to hub-modal callback or file browser
      if (pending.intent === 'hub-modal') {
        // Signal HubPanel to open the modal for the pending session
        capAcquiredRef.current?.(pending.id)
      } else {
        handleOpenFileBrowser(pending.id, pending.name)
      }
    },
    [joinModalForSession, remotePeers, handleOpenFileBrowser, openWebSessionTab],
  )

  // Phase 131 — open the Hub tab. HUB-02: coexists
  // with the Sessions panel — adds its own tab rather than replacing any existing tab.
  const handleOpenHub = useCallback(() => {
    const existing = tabs.find((t) => t.type === 'hub')
    if (existing) {
      setActiveId(existing.id)
      return
    }
    setTabs((prev) => [...prev, HUB_TAB])
    setActiveId(HUB_TAB.id)
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
    // Phase 120-06 — web mode has no daemon to retry against. The web-share
    // shell never sets daemonError (init early-returns before any RPC), so
    // this is defensive — but keeping the guard documents the contract that
    // every Wails-bound effect must consult `mode` first.
    if (mode === 'web') return
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
      // Phase 168-05 (UX-02): mirror the mount-time hubSessions seed (see init above).
      setHubSessions(sessions)

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
      {/* INVARIANT: this gating expression must include the trigger of EVERY banner
          rendered inside .banner-stack below. A banner whose own render condition is
          true but whose trigger is missing here will never mount (the whole stack is
          conditionally rendered). Phase 129 DNS-03 regressed exactly this way. */}
      {((webServerMode === 'local' && !localBannerDismissed) ||
        (tailscaleHealth?.connected === true && tailscaleHealth?.acceptDns === false) ||
        update ||
        ((webglContextLost || webglSoftwareDetected) && !webglBannerDismissed) ||
        saveBanner !== null ||
        pluginToggleBanners.length > 0) && (
        <div className="banner-stack">
          {/* Phase 101-03 SHELL-08 — the action-blocking shell web-share security
              banner used to render here (top priority) for the footer's direct
              toggle. Phase 168-05 (D-14): the footer no longer toggles directly —
              the equivalent banner (ShellWebShareBanner, same shared warned/
              warningEnabled authority) now lives inside SessionShareModal only. */}
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
          <RemoteBrowseDNSWarning
            connected={!!(tailscaleHealth?.connected)}
            acceptDns={tailscaleHealth?.acceptDns}
          />
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
      {/* Phase 159-02 (WEBCHAT-03) — web-share guest scope. In web mode the SPA
          is served to a remote guest holding a capability for ONE session; the
          desktop navigation chrome (Home / Hub / Settings / session groups) must
          NOT be rendered, or the guest could navigate away from the scoped
          session and reach the open /api/sessions/meta enumeration surface.
          Mirrors the existing `mode !== 'web'` gates on the Settings surface. */}
      {mode !== 'web' && (
      <Sidebar
        onHome={handleHome}
        onSettings={handleOpenSettings}
        onOpenHub={handleOpenHub}
        onOpenHelp={handleOpenHelp}
        activePanel={activeId ?? undefined}
        groupDefs={groupDefs}
        activeGroupId={activeGroupId}
        onGroupSelect={handleGroupSelect}
        onCreateGroup={handleCreateGroup}
        onDropOnGroup={handleDropOnGroup}
        groupCounts={groupCounts}
        globalGroupCounts={globalGroupCounts}
      />
      )}
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
          onBrowseFiles={handleOpenFileBrowser}
          webMode={mode === 'web'}
          webFilesEnabled={webFilesEnabled}
          funnelActiveSessions={/* Phase 166 / FUI-03: derived from hubSessions 3s poll; no new interval */
            hubSessions.reduce<Record<string, boolean>>(
              (acc, s) => ({ ...acc, [s.id]: s.funnelActive }),
              {}
            )
          }
        />

        <div className="terminal-container">
        {activeId === WELCOME_TAB.id && (
          <WelcomeTab />
        )}
        {/* Phase 131 — Hub surface. Phase 138 — wired with onKill/onOpenInBrowser/onBrowseFiles/remotePeers. */}
        {activeId === HUB_TAB.id && (
          <HubPanel
            sessions={hubSessions}
            error={hubError}
            onNewSession={() => setShowNewSessionModal(true)}
            onRename={handleRenameTab}
            onOpenSession={handleOpenSessionTab}
            remoteSessions={adaptAllRemoteSessions(remotePeers)}
            isActive={activeId === HUB_TAB.id}
            relayPort={relayPort ?? undefined}
            terminalTheme={terminalTheme}
            pluginConfig={pluginConfig}
            remoteCapsCached={remoteCapsCached}
            onRequestRemoteCap={(s) => {
              // WR-02: do NOT overwrite an in-flight joinModalForSession (e.g. a file-browse
              // intent already open for a different session). Preserves the in-flight request
              // rather than silently redirecting the cap exchange to a different consumer.
              // Fire the cancel ref so HubPanel resets pendingModalSessionId immediately —
              // without this, the card click appears to do nothing and the pending id is
              // stranded until the next modal close.
              if (joinModalForSession) { capCancelledRef.current?.(); return }
              setJoinModalForSession({ id: s.id, name: s.name, hostname: s.hostname, intent: 'hub-modal' })
            }}
            onRegisterCapAcquired={(fn) => { capAcquiredRef.current = fn }}
            onRegisterCapCancelled={(fn) => { capCancelledRef.current = fn }}
            fontSizes={fontSizes}
            onFontSizeChange={handleFontSizeChange}
            onKill={(id) => void handleCloseTab(id)}
            onOpenInBrowser={handleOpenRemoteSession}
            onBrowseFiles={handleBrowseFilesRemote}
            remotePeers={remotePeers}
            activeGroupId={activeGroupId}
            groupDefs={groupDefs}
            onDropOnGroup={handleDropOnGroup}
            onGroupCountsChange={handleGroupCountsChange}
            setShareModalSession={setShareModalSession}
          />
        )}
        {/* Phase 155-03 — WebShareSessionView render branch. Activated when
            the active tab id starts with __websession__. This is the primary
            surface for web-share viewers: TerminalPanel + ChatPanel overlay
            connected to the webserver WS (not the loopback relay). relayPort
            is 0 on web-share; wsURL inside WebShareSessionView overrides the
            loopback construction so the sentinel 0 is never used for socket
            connects.

            Phase 168-03 (FIX-03) — per-tab param resolution. A remote-peer
            tab (opened via handleOpenRemoteSession) carries its OWN
            baseURL/capToken on the Tab object; those are read here instead
            of the single mount-stable webParams (RESEARCH Pitfall 3 — reading
            the global webParams for a remote tab would silently reuse the
            FIRST remote session's cap/host for a SECOND one). The app's own
            web-share bootstrap tab (mode==='web', no per-tab baseURL) still
            falls back to webParams, preserving prior behavior. pluginConfig
            is likewise only meaningful for the local bootstrap tab — App's
            own pluginConfig state describes THIS daemon's plugins, which is
            irrelevant to a different peer's session; passing it through for
            a remote tab would suppress WebShareSessionView's web-guest
            self-fetch (isWebGuest) and apply the wrong peer's config. */}
        {activeId !== null && activeId.startsWith('__websession__') && (() => {
          const activeWebTab = tabs.find((t) => t.id === activeId)
          const isRemoteWebTab = activeWebTab?.baseURL !== undefined
          const wsSessionId = activeWebTab?.sessionId ?? webParams.sessionId ?? activeId.slice('__websession__'.length)
          const wsCapToken = isRemoteWebTab ? (activeWebTab?.capToken ?? '') : (webParams.capToken ?? '')
          return (
            <WebShareSessionView
              sessionId={wsSessionId}
              capToken={wsCapToken}
              baseURL={activeWebTab?.baseURL}
              relayPort={relayPort ?? 0}
              theme={terminalTheme}
              pluginConfig={isRemoteWebTab ? undefined : (pluginConfig ?? undefined)}
            />
          )
        })()}

        {/* Phase 120-04 — per-session FileBrowserTab. Activated when activeId
            begins with the __files__ prefix; the tab id encodes the sessionId
            after the prefix so we can resolve which session to browse.

            Phase 120-06 — mode-aware fbBaseURL + capToken selection for the
            local + web-share paths.

            Phase 122-03 — remote-on-desktop branch: when the sessionId is a
            REMOTE (tailnet-peer) session, route through the local-daemon
            proxy at /api/files/remote/{sid}/... (D-02 — no cross-origin
            browser fetches). The cap lives in the daemon's RemoteCapStore,
            NOT in React state (T-122-03-01); FilesApiClient sends no cap on
            the same-origin proxy URL. */}
        {activeId !== null && activeId.startsWith('__files__') && (() => {
          const fbSessionId = activeId.slice('__files__'.length)
          const remote = findRemoteSession(fbSessionId, remotePeers)

          if (remote) {
            const reopenJoinModal = () => {
              setRemoteCapsCached((prev) => {
                const next = new Set(prev)
                next.delete(fbSessionId)
                return next
              })
              setJoinModalForSession({
                id: fbSessionId,
                name: remote.name,
                hostname: remote.hostname,
              })
            }
            // Defensive guard: if the user landed on the file-browser tab
            // for a remote session without going through the modal (e.g.
            // they reloaded after closing the modal), show the takeover so
            // they can re-enter the join code.
            if (!remoteCapsCached.has(fbSessionId)) {
              return <EnableWebSharingTakeover onReenterJoinCode={reopenJoinModal} />
            }
            return (
              <FileBrowserTab
                sessionId={fbSessionId}
                sessionName={remote.name}
                isActive={true}
                isRemote={true}
                baseURL={`http://127.0.0.1:${relayPort ?? 0}`}
                pathPrefix={`/api/files/remote/${fbSessionId}`}
                onReenterJoinCode={reopenJoinModal}
              />
            )
          }

          // Local-session path: UNCHANGED from Phase 120-06.
          // Phase 138: use hubSessions (was panelSessions for the deleted Sessions panel).
          const fbSession = hubSessions.find((s) => s.id === fbSessionId)
          const fbName =
            fbSession?.name ||
            tabs.find((t) => t.id === activeId)?.name?.replace(/ — Files$/, '') ||
            fbSessionId ||
            'Session'
          const isWeb = mode === 'web'
          const fbBaseURL = isWeb
            ? window.location.origin
            : `http://127.0.0.1:${relayPort ?? 0}`
          const fbCapToken: string | undefined = isWeb
            ? (webParams.capToken ?? undefined)
            : undefined
          return (
            <FileBrowserTab
              sessionId={fbSessionId}
              sessionName={fbName}
              isActive={true}
              isRemote={isWeb}
              baseURL={fbBaseURL}
              capToken={fbCapToken}
            />
          )
        })()}
        {/* Phase 120-06 — SettingsTab is Wails-bound (calls IsWebServerRunning,
            HasCTDisclosure, GetCLIPaths, etc. unconditionally on mount). In
            web mode none of those RPCs are reachable; rather than gate each
            call inside the component, we skip mounting it entirely. The
            Settings surface has no meaning for a web-share viewer (no daemon
            to configure). */}
        {mode !== 'web' && (
        <div style={{ display: activeId === SETTINGS_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
          <SettingsTab
            clis={detectedCLIs}
            tailscaleHealth={tailscaleHealth}
            webServerMode={webServerMode}
            selectedTheme={terminalThemeName}
            onThemeChange={handleThemeChange}
            uiTheme={uiTheme}
            onUiThemeChange={handleUiThemeChange}
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
            onShellWarnEnabledChange={(enabled: boolean) => {
              // Phase 150 SET-01 — keep App.tsx warningEnabled in sync with
              // the SettingsTab toggle. When the user re-enables the warning
              // (OFF→ON), also re-fetch shellWebShareWarned from the daemon
              // because SetShellWebShareWarningEnabled(true) atomically reset
              // shellWebShareWarned to false on the daemon side (D-03 re-arm).
              // Without this re-sync, local state would still show warned=true
              // and the banner would never re-appear.
              setShellWebShareWarningEnabled(enabled)
              if (enabled) {
                GetShellWebShareWarned()
                  .then(setShellWebShareWarned)
                  .catch(() => setShellWebShareWarned(false))
              }
            }}
          />
        </div>
        )}
        {/* Phase 147 — HelpTab mounted with display-toggle (not conditional) to
            preserve scroll position and activeSection state across tab switches.
            Mirrors the SettingsTab display-toggle mount pattern above. */}
        {mode !== 'web' && (
        <div style={{ display: activeId === HELP_TAB.id ? 'flex' : 'none', flexDirection: 'column', height: '100%' }}>
          <HelpTab />
        </div>
        )}
        {daemonError && tabs.filter((t) => t.type !== 'welcome' && t.type !== 'settings' && t.type !== 'hub').length === 0 && (
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
            if (tab.type === 'welcome' || tab.type === 'settings' || tab.type === 'file-browser' || tab.type === 'hub' || tab.type === 'help' || tab.type === 'web-session') return null
            const isActive = tab.id === activeId
            return (
              <div
                key={tab.sessionId}
                className="terminal-wrapper"
                style={{ display: isActive ? 'flex' : 'none' }}
              >
                <TerminalChatHost
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
                  onShareSession={openShareModalForActiveSession}
                />
              </div>
            )
          })}
        </div>
      </div>
      </div>{/* end app__row */}

      {/* Phase 168-05 (UX-02) — Share modal: rendered here (App.tsx top level, always
          mounted regardless of active tab) rather than inside HubPanel (which fully
          unmounts on non-Hub tabs). This is the SINGLE SessionShareModal instance —
          both the Hub card click (HubPanel's handleShare, via the setShareModalSession
          prop) and the footer "Share Session" button (openShareModalForActiveSession)
          drive this one shareModalSession state (RESEARCH Pattern 4). Do NOT also
          render the SessionShareModal component from HubPanel.tsx — a second
          instance reproduces the exact button-modal state-drift bug (#115) this
          plan exists to fix. */}
      {shareModalSession && (
        <SessionShareModal
          session={shareModalSession}
          webServerMode={webServerMode}
          webServerRunning={webServerRunning}
          onClose={() => setShareModalSession(null)}
          shellWebShareWarned={shellWebShareWarned}
          shellWebShareWarningEnabled={shellWebShareWarningEnabled}
          onShellWebShareConfirm={handleShellWebShareConfirm}
          onShellWebShareCancel={handleShellWebShareCancel}
          onOpenHelp={() => handleOpenHelp('help-sharing')}
        />
      )}

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

      {/* Phase 122-03 — paste-join-code modal for remote-session file browse.
          WR-01: when the intent is 'hub-modal', also invoke capCancelledRef to reset
          HubPanel's pending state (pendingModalSessionId / pendingSourceRectRef).
          File-browse intent does not need a reset (no HubPanel pending state). */}
      {joinModalForSession && (
        <RemoteJoinCodeModal
          remoteSession={joinModalForSession}
          intent={joinModalForSession.intent}
          onExchange={handleModalExchange}
          onClose={() => {
            if (joinModalForSession?.intent === 'hub-modal') {
              capCancelledRef.current?.()
            }
            setJoinModalForSession(null)
          }}
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
