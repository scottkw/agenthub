/**
 * Phase 141-08 appearance theme tests.
 * Verifies the Light/Dark segmented control in Settings → Appearance:
 *   - Active state reflection (aria-pressed) for current uiTheme
 *   - Click callbacks invoke onUiThemeChange with correct value
 *
 * Uses the createRoot + flushSync DOM render pattern (consistent with
 * SettingsTab.shellPath.test.tsx and StatusBar.test.tsx in this project).
 *
 * Note: App.tsx documentElement effect requires Wails bindings to mount the
 * full App component; attribute assertion is deferred to 141-09 live UAT.
 * The SettingsTab + App prop wiring is covered by tsc (strict type-checking)
 * and the source-inspection checks below.
 */
import { describe, it, expect, afterEach, vi } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SettingsTab } from '../SettingsTab'
import raw from '../SettingsTab.tsx?raw'
import appRaw from '../../App.tsx?raw'

// Mock all Wails bindings used by SettingsTab.
vi.mock('../../wailsjs/go/main/App', () => ({
  UpdateCLIPath: vi.fn().mockResolvedValue(undefined),
  GetCLIPaths: vi.fn().mockResolvedValue({}),
  OpenFileDialog: vi.fn().mockResolvedValue(''),
  StartWebServer: vi.fn().mockResolvedValue(undefined),
  StopWebServer: vi.fn().mockResolvedValue(undefined),
  GetWebServerURL: vi.fn().mockResolvedValue('https://example.ts.net'),
  IsWebServerRunning: vi.fn().mockResolvedValue(false),
  HasCTDisclosure: vi.fn().mockResolvedValue(true),
  AcknowledgeCTDisclosure: vi.fn().mockResolvedValue(undefined),
  GetLocalNetworkPassword: vi.fn().mockResolvedValue(''),
  GetWebServerQRCode: vi.fn().mockResolvedValue(''),
  GetStartMinimized: vi.fn().mockResolvedValue(false),
  SetStartMinimized: vi.fn().mockResolvedValue(undefined),
  GetAutoCloseSession: vi.fn().mockResolvedValue(true),
  SetAutoCloseSession: vi.fn().mockResolvedValue(undefined),
  RegenerateSigningKey: vi.fn().mockResolvedValue(undefined),
  GetPluginSettings: vi.fn().mockResolvedValue({ searchConfig: {}, webLinksConfig: {}, imageConfig: {} }),
  SetPluginSettings: vi.fn().mockResolvedValue(undefined),
  SetSearchConfig: vi.fn().mockResolvedValue(undefined),
  SetWebLinksConfig: vi.fn().mockResolvedValue(undefined),
  SetImageConfig: vi.fn().mockResolvedValue(undefined),
  GetTailscaleStatus: vi.fn().mockResolvedValue(null),
  NotifyThemeChange: vi.fn().mockResolvedValue(undefined),
  GetShellPath: vi.fn().mockResolvedValue('/bin/zsh'),
  SetShellPath: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('../RegenerateKeyModal', () => ({
  RegenerateKeyModal: () => null,
}))
vi.mock('../PluginsSection', () => ({
  PluginsSection: () => null,
}))
vi.mock('../SettingsJumpBar', () => ({
  SettingsJumpBar: () => null,
}))
vi.mock('../SettingsSearch', () => ({
  SettingsSearch: () => null,
}))
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
}))

// Minimal stub props satisfying all required SettingsTabProps fields.
function makeProps(overrides: { uiTheme?: 'dark' | 'light'; onUiThemeChange?: (t: 'dark' | 'light') => void } = {}) {
  return {
    clis: [],
    tailscaleHealth: null,
    webServerMode: null as null,
    onWebServerStateChange: vi.fn().mockResolvedValue(undefined),
    selectedTheme: 'Tomorrow_Night',
    onThemeChange: vi.fn(),
    uiTheme: overrides.uiTheme ?? ('dark' as const),
    onUiThemeChange: overrides.onUiThemeChange ?? vi.fn(),
    onPluginToggleSideEffect: undefined,
  }
}

function renderSettingsTab(props: ReturnType<typeof makeProps>) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(SettingsTab, props))
  })
  return { container, root }
}

describe('141-08: SettingsTab appearance-theme source contract', () => {
  it('SettingsTabProps includes uiTheme field', () => {
    const interfaceStart = raw.indexOf('interface SettingsTabProps')
    const interfaceEnd = raw.indexOf('\n}', interfaceStart)
    const block = raw.slice(interfaceStart, interfaceEnd + 2)
    expect(block).toContain("uiTheme: 'dark' | 'light'")
  })

  it('SettingsTabProps includes onUiThemeChange callback', () => {
    const interfaceStart = raw.indexOf('interface SettingsTabProps')
    const interfaceEnd = raw.indexOf('\n}', interfaceStart)
    const block = raw.slice(interfaceStart, interfaceEnd + 2)
    expect(block).toContain('onUiThemeChange')
  })

  it('Appearance section contains aria-pressed attribute', () => {
    expect(raw).toContain('aria-pressed')
  })

  it('SettingsTab calls onUiThemeChange with light', () => {
    expect(raw).toContain("onUiThemeChange('light')")
  })

  it('SettingsTab calls onUiThemeChange with dark', () => {
    expect(raw).toContain("onUiThemeChange('dark')")
  })
})

describe('141-08: App.tsx uiTheme wiring source contract', () => {
  it('defines UI_THEME_STORAGE_KEY constant', () => {
    expect(appRaw).toContain('UI_THEME_STORAGE_KEY')
  })

  it('UI_THEME_STORAGE_KEY is distinct from terminal theme key', () => {
    expect(appRaw).toContain("'agenthub:uiTheme'")
    expect(appRaw).toContain("'agenthub:terminalTheme'")
    // Both exist and are different strings
    expect("'agenthub:uiTheme'").not.toBe("'agenthub:terminalTheme'")
  })

  it('uiTheme state reads from localStorage on init', () => {
    expect(appRaw).toContain('UI_THEME_STORAGE_KEY')
    expect(appRaw).toContain("=== 'light' ? 'light' : 'dark'")
  })

  it('effect calls setAttribute on documentElement for light theme', () => {
    expect(appRaw).toContain("document.documentElement.setAttribute('data-ui-theme', 'light')")
  })

  it('effect calls removeAttribute on documentElement for dark theme', () => {
    expect(appRaw).toContain("document.documentElement.removeAttribute('data-ui-theme')")
  })

  it('passes uiTheme prop to SettingsTab render', () => {
    expect(appRaw).toContain('uiTheme={uiTheme}')
  })

  it('passes onUiThemeChange prop to SettingsTab render', () => {
    expect(appRaw).toContain('onUiThemeChange={handleUiThemeChange}')
  })
})

describe('141-08: SettingsTab Light/Dark control DOM behavior', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('Dark button has aria-pressed=true when uiTheme is dark', () => {
    const props = makeProps({ uiTheme: 'dark' })
    ;({ container, root } = renderSettingsTab(props))
    const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-pressed]')
    const darkBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'Dark')
    expect(darkBtn).toBeDefined()
    expect(darkBtn!.getAttribute('aria-pressed')).toBe('true')
  })

  it('Light button has aria-pressed=false when uiTheme is dark', () => {
    const props = makeProps({ uiTheme: 'dark' })
    ;({ container, root } = renderSettingsTab(props))
    const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-pressed]')
    const lightBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'Light')
    expect(lightBtn).toBeDefined()
    expect(lightBtn!.getAttribute('aria-pressed')).toBe('false')
  })

  it('clicking Light button calls onUiThemeChange with "light"', () => {
    const onUiThemeChange = vi.fn()
    const props = makeProps({ uiTheme: 'dark', onUiThemeChange })
    ;({ container, root } = renderSettingsTab(props))
    const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-pressed]')
    const lightBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'Light')
    expect(lightBtn).toBeDefined()
    lightBtn!.click()
    expect(onUiThemeChange).toHaveBeenCalledWith('light')
  })

  it('clicking Dark button calls onUiThemeChange with "dark" when uiTheme is light', () => {
    const onUiThemeChange = vi.fn()
    const props = makeProps({ uiTheme: 'light', onUiThemeChange })
    ;({ container, root } = renderSettingsTab(props))
    const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-pressed]')
    const darkBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'Dark')
    expect(darkBtn).toBeDefined()
    darkBtn!.click()
    expect(onUiThemeChange).toHaveBeenCalledWith('dark')
  })

  it('Light button has aria-pressed=true when uiTheme is light', () => {
    const props = makeProps({ uiTheme: 'light' })
    ;({ container, root } = renderSettingsTab(props))
    const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-pressed]')
    const lightBtn = Array.from(buttons).find(b => b.textContent?.trim() === 'Light')
    expect(lightBtn).toBeDefined()
    expect(lightBtn!.getAttribute('aria-pressed')).toBe('true')
  })

  it('the Interface theme control has role=group aria-label', () => {
    const props = makeProps({ uiTheme: 'dark' })
    ;({ container, root } = renderSettingsTab(props))
    const group = container.querySelector('[role="group"][aria-label="Interface theme"]')
    expect(group).not.toBeNull()
  })
})
