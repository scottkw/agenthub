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

  describe('Tailscale health integration (HEALTH-04)', () => {
    it('imports GetTailscaleStatus from wailsjs bindings', () => {
      expect(raw).toContain('GetTailscaleStatus')
    })

    it('imports Environment from wailsjs runtime', () => {
      expect(raw).toContain('Environment')
    })

    it('imports HealthModal component', () => {
      expect(raw).toContain("import { HealthModal } from './components/HealthModal'")
    })

    it('calls GetTailscaleStatus in Promise.all init', () => {
      expect(raw).toContain('GetTailscaleStatus()')
    })

    it('calls Environment() in Promise.all init', () => {
      expect(raw).toContain('Environment()')
    })

    it('subscribes to tailscale:health event', () => {
      expect(raw).toContain("EventsOn('tailscale:health'")
    })

    it('cleans up health event listener', () => {
      expect(raw).toContain('offHealth()')
    })

    it('renders HealthModal in JSX', () => {
      expect(raw).toContain('<HealthModal')
    })

    it('passes health prop to HealthModal', () => {
      expect(raw).toContain('health={tailscaleHealth}')
    })

    it('passes platform prop to HealthModal', () => {
      expect(raw).toContain('platform={platform}')
    })

    it('passes onCheckAgain prop to HealthModal', () => {
      expect(raw).toContain('onCheckAgain={handleCheckHealthAgain}')
    })

    it('passes tailscaleHealth prop to SettingsTab', () => {
      expect(raw).toContain('tailscaleHealth={tailscaleHealth}')
    })

    it('stores platform from Environment().platform', () => {
      expect(raw).toContain('env.platform')
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
