import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App', () => {
  // UILAY-02: App.tsx imports and renders <StatusBar> inside terminal-wrapper
  describe('StatusBar integration (UILAY-02)', () => {
    it('imports StatusBar from components/StatusBar', () => {
      expect(raw).toContain("import { StatusBar } from './components/StatusBar'")
    })

    it('renders <StatusBar inside terminal-wrapper', () => {
      // Both TerminalPanel and StatusBar must appear inside the terminal-wrapper block
      const wrapperBlock = raw.slice(raw.indexOf('terminal-wrapper'))
      expect(wrapperBlock).toContain('<StatusBar')
    })

    it('passes required props to StatusBar', () => {
      expect(raw).toContain('webServerRunning={webServerRunning}')
      expect(raw).toContain('webEnabled={!!webEnabled[tab.sessionId]}')
      expect(raw).toContain('sessionURL={sessionURLs[tab.sessionId]}')
    })
  })

  // UILAY-03: Old web-serving-bar overlay has been removed from App.tsx
  describe('old overlay removed (UILAY-03)', () => {
    it('does not contain web-serving-bar class', () => {
      expect(raw).not.toContain('web-serving-bar')
    })

    it('does not contain web-toggle-btn class', () => {
      expect(raw).not.toContain('web-toggle-btn')
    })

    it('does not contain web-session-url class', () => {
      expect(raw).not.toContain('web-session-url')
    })

    it('does not contain copy-token-btn class', () => {
      expect(raw).not.toContain('copy-token-btn')
    })
  })

  describe('per-tab font size state', () => {
    it('declares fontSizes state as Record<string, number>', () => {
      expect(raw).toContain('Record<string, number>')
      expect(raw).toContain('fontSizes')
    })

    it('defines DEFAULT_FONT_SIZE = 14', () => {
      expect(raw).toContain('DEFAULT_FONT_SIZE = 14')
    })

    it('defines handleFontSizeChange callback', () => {
      expect(raw).toContain('handleFontSizeChange')
    })

    it('clamps font size between 6 and 32', () => {
      expect(raw).toContain('Math.max(6')
      expect(raw).toContain('Math.min(32')
    })

    it('passes fontSize prop to TerminalPanel', () => {
      expect(raw).toContain('fontSize=')
    })

    it('passes onFontSizeChange prop to TerminalPanel', () => {
      expect(raw).toContain('onFontSizeChange=')
    })

    it('cleans up fontSizes entry on tab close', () => {
      expect(raw).toContain('setFontSizes')
      // Cleanup pattern: delete n[id] in handleCloseTab
      const closeSection = raw.slice(raw.indexOf('handleCloseTab'))
      expect(closeSection).toContain('setFontSizes')
    })
  })

  describe('Tailscale health and local network integration', () => {
    it('imports GetTailscaleStatus from wailsjs bindings', () => {
      expect(raw).toContain('GetTailscaleStatus')
    })

    it('imports LocalNetworkBanner component', () => {
      expect(raw).toContain("import { LocalNetworkBanner } from './components/LocalNetworkBanner'")
    })

    it('calls GetTailscaleStatus in Promise.all init', () => {
      expect(raw).toContain('GetTailscaleStatus()')
    })

    it('subscribes to tailscale:health event', () => {
      expect(raw).toContain("EventsOn('tailscale:health'")
    })

    it('cleans up health event listener', () => {
      expect(raw).toContain('offHealth()')
    })

    it('renders LocalNetworkBanner in JSX', () => {
      expect(raw).toContain('<LocalNetworkBanner')
    })

    it('passes tailscaleHealth prop to SettingsTab', () => {
      expect(raw).toContain('tailscaleHealth={tailscaleHealth}')
    })

    it('passes tailscaleBinaryFound prop to LocalNetworkBanner', () => {
      expect(raw).toContain('tailscaleBinaryFound={')
    })

    it('passes tailscaleDaemonUp prop to LocalNetworkBanner', () => {
      expect(raw).toContain('tailscaleDaemonUp={')
    })

    it('passes platformHint prop to LocalNetworkBanner', () => {
      expect(raw).toContain('platformHint={')
    })

    it('tailscaleHealth state includes binaryFound', () => {
      expect(raw).toContain('binaryFound: boolean')
    })

    it('tailscaleHealth state includes daemonUp', () => {
      expect(raw).toContain('daemonUp: boolean')
    })
  })

  describe('THM-02/03: terminal theme state', () => {
    it('imports xterm-theme library', () => {
      expect(raw).toContain("from 'xterm-theme'")
    })

    it('imports ITheme type from @xterm/xterm', () => {
      expect(raw).toContain('ITheme')
    })

    it('defines THEME_STORAGE_KEY constant', () => {
      expect(raw).toContain("THEME_STORAGE_KEY = 'agenthub:terminalTheme'")
    })

    it('defines DEFAULT_THEME_NAME constant as Tomorrow_Night', () => {
      expect(raw).toContain("DEFAULT_THEME_NAME = 'Tomorrow_Night'")
    })

    it('initializes terminalThemeName from localStorage', () => {
      expect(raw).toContain('localStorage.getItem(THEME_STORAGE_KEY)')
    })

    it('derives terminalTheme ITheme object from theme name', () => {
      expect(raw).toContain('terminalTheme')
      expect(raw).toContain('xtermThemes')
    })

    it('handleThemeChange writes to localStorage and updates state', () => {
      expect(raw).toContain('localStorage.setItem(THEME_STORAGE_KEY, name)')
      expect(raw).toContain('setTerminalThemeName(name)')
    })

    it('passes theme prop to TerminalPanel', () => {
      expect(raw).toContain('theme={terminalTheme}')
    })

    it('passes selectedTheme prop to SettingsTab', () => {
      expect(raw).toContain('selectedTheme={terminalThemeName}')
    })

    it('passes onThemeChange prop to SettingsTab', () => {
      expect(raw).toContain('onThemeChange={handleThemeChange}')
    })
  })

  describe('daemon error handling (Phase 26)', () => {
    it('imports RetryDaemon from wailsjs bindings', () => {
      expect(raw).toContain('RetryDaemon')
      expect(raw).toContain("from './wailsjs/go/main/App'")
    })

    it('subscribes to daemon:error event on mount', () => {
      expect(raw).toContain("EventsOn('daemon:error'")
    })

    it('unsubscribes from daemon:error in cleanup', () => {
      expect(raw).toContain('offDaemonError()')
    })

    it('retryInit calls RetryDaemon before other methods', () => {
      // RetryDaemon must appear before Promise.all in retryInit
      const retryBlock = raw.slice(raw.indexOf('const retryInit'))
      const retryDaemonPos = retryBlock.indexOf('await RetryDaemon()')
      const promiseAllPos = retryBlock.indexOf('Promise.all')
      expect(retryDaemonPos).toBeGreaterThan(-1)
      expect(promiseAllPos).toBeGreaterThan(-1)
      expect(retryDaemonPos).toBeLessThan(promiseAllPos)
    })

    it('renders daemonError directly in banner (not hardcoded message)', () => {
      // The banner body should use {daemonError} not a static string
      expect(raw).toContain('{daemonError}')
      expect(raw).not.toContain('The background daemon did not start in time')
    })

    it('imports GetDaemonError from wailsjs bindings', () => {
      expect(raw).toContain('GetDaemonError')
    })

    it('calls GetDaemonError before Promise.all in init', () => {
      const initBlock = raw.slice(raw.indexOf('async function init()'))
      const getDaemonPos = initBlock.indexOf('GetDaemonError()')
      const promiseAllPos = initBlock.indexOf('Promise.all')
      expect(getDaemonPos).toBeGreaterThan(-1)
      expect(promiseAllPos).toBeGreaterThan(-1)
      expect(getDaemonPos).toBeLessThan(promiseAllPos)
    })
  })
})

describe('THM-05: NotifyThemeChange wiring', () => {
  it('imports NotifyThemeChange from wailsjs bindings', () => {
    expect(raw).toContain('NotifyThemeChange')
  })

  it('handleThemeChange calls NotifyThemeChange', () => {
    const themeHandler = raw.slice(raw.indexOf('handleThemeChange'))
    expect(themeHandler).toContain('NotifyThemeChange()')
  })

  it('NotifyThemeChange call has .catch for error handling', () => {
    expect(raw).toContain("NotifyThemeChange().catch")
  })
})

describe('BannerStack integration (BAN-01, BAN-02)', () => {
  it('imports UpdateBanner component', () => {
    expect(raw).toContain("import { UpdateBanner } from './components/UpdateBanner'")
  })

  it('imports UpdateInfo type', () => {
    expect(raw).toContain("import type { UpdateInfo } from './components/UpdateBanner'")
  })

  it('imports GetLastUpdateInfo from wailsjs bindings', () => {
    expect(raw).toContain('GetLastUpdateInfo')
  })

  it('declares update state at App level', () => {
    expect(raw).toContain('useState<UpdateInfo | null>(null)')
  })

  it('declares localBannerDismissed state', () => {
    expect(raw).toContain('localBannerDismissed')
  })

  it('declares localBannerExiting state', () => {
    expect(raw).toContain('localBannerExiting')
  })

  it('declares updateExiting state', () => {
    expect(raw).toContain('updateExiting')
  })

  it('defines handleDismissLocalBanner callback', () => {
    expect(raw).toContain('handleDismissLocalBanner')
  })

  it('defines handleDismissUpdate callback', () => {
    expect(raw).toContain('handleDismissUpdate')
  })

  it('renders banner-stack div', () => {
    expect(raw).toContain('banner-stack')
  })

  it('renders LocalNetworkBanner inside banner-stack', () => {
    const stackBlock = raw.slice(raw.indexOf('banner-stack'))
    expect(stackBlock).toContain('<LocalNetworkBanner')
  })

  it('renders UpdateBanner inside banner-stack', () => {
    const stackBlock = raw.slice(raw.indexOf('banner-stack'))
    expect(stackBlock).toContain('<UpdateBanner')
  })

  it('subscribes to update:available event', () => {
    expect(raw).toContain("EventsOn('update:available'")
  })

  it('passes onDismiss to LocalNetworkBanner', () => {
    expect(raw).toContain('onDismiss={handleDismissLocalBanner}')
  })

  it('passes onDismiss to UpdateBanner', () => {
    expect(raw).toContain('onDismiss={handleDismissUpdate}')
  })

  it('passes className with banner-exit for exit animation', () => {
    expect(raw).toContain("localBannerExiting ? 'banner-exit' : undefined")
  })

  it('resets localBannerDismissed when webServerMode changes', () => {
    expect(raw).toContain('setLocalBannerDismissed(false)')
  })
})

// ARGS-02: Args threading from modal to CreateSession
describe('ARGS-02: args threading', () => {
  it('onConfirm passes args to createTab', () => {
    expect(raw).toContain('createTab(cli, workDir, args)')
  })
  it('createTab passes args to CreateSession', () => {
    expect(raw).toContain('CreateSession(cliName, defaultName, workDir, args, cols, rows)')
  })
})

// BRND-02: Welcome tab integration
describe('BRND-02: welcome tab integration', () => {
  it('imports WelcomeTab component', () => {
    expect(raw).toContain("import { WelcomeTab } from './components/WelcomeTab'")
  })

  it('defines a WELCOME_TAB constant with type welcome', () => {
    expect(raw).toContain("type: 'welcome'")
    expect(raw).toContain('WELCOME_TAB')
  })

  it('initializes tabs with welcome tab', () => {
    expect(raw).toContain('useState<Tab[]>([WELCOME_TAB])')
  })

  it('initializes activeId with welcome tab id', () => {
    expect(raw).toContain('useState<string | null>(WELCOME_TAB.id)')
  })

  it('renders WelcomeTab when active tab is welcome', () => {
    expect(raw).toContain('<WelcomeTab')
  })

  it('hides static HTML splash on mount', () => {
    expect(raw).toContain("getElementById('splash-static')")
  })
})
