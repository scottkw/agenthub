/**
 * Phase 142-01 / POL-02 appearance theme tests — updated for single role=switch toggle.
 * Wave 0 RED tests: assertions expect the new single-toggle contract. They WILL FAIL
 * against the current two-button implementation — that is correct for Wave 0.
 *
 * Verifies the Light/Dark toggle in Settings → Appearance:
 *   - Toggle is a single [role="switch"] element (not two button[aria-pressed])
 *   - aria-checked reflects current uiTheme ('true' for light, 'false' for dark)
 *   - Click calls onUiThemeChange with the OPPOSITE of the current theme
 *   - Source contains role="switch", aria-checked, SunIcon/MoonIcon, Light/Dark text
 *
 * Uses the createRoot + flushSync DOM render pattern (consistent with
 * SettingsTab.shellPath.test.tsx and StatusBar.test.tsx in this project).
 *
 * Note: App.tsx documentElement effect requires Wails bindings to mount the
 * full App component; attribute assertion is deferred to live UAT.
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
  GetShellWebShareWarningEnabled: vi.fn().mockResolvedValue(true),
  SetShellWebShareWarningEnabled: vi.fn().mockResolvedValue(undefined),
  GetNotifyOnWaiting: vi.fn().mockResolvedValue(false),
  SetNotifyOnWaiting: vi.fn().mockResolvedValue(undefined),
  GetStayOnHubAfterCreate: vi.fn().mockResolvedValue(false),
  SetStayOnHubAfterCreate: vi.fn().mockResolvedValue(undefined),
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
  // Phase 167-07: SettingsTab now also subscribes to
  // notification:permission-denied on mount; stub returns a no-op unsubscribe.
  EventsOn: vi.fn().mockReturnValue(vi.fn()),
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

  // POL-02 RED: replaces old aria-pressed assertion
  it('Appearance section contains role="switch" toggle (POL-02 RED)', () => {
    expect(raw).toContain('role="switch"')
  })

  // POL-02 RED: replaces old aria-pressed assertion
  it('Appearance section contains aria-checked attribute (POL-02 RED)', () => {
    expect(raw).toContain('aria-checked')
  })

  it('SettingsTab calls onUiThemeChange with opposite theme on click (toggle idiom)', () => {
    // The new toggle-call source: onUiThemeChange(uiTheme === 'light' ? 'dark' : 'light')
    // This expression contains both 'dark' and 'light' — assert the toggle pattern
    expect(raw).toContain("uiTheme === 'light' ? 'dark' : 'light'")
  })

  // POL-02 / D-06 RED: colorblind-safe — icon+text knob must be present in source
  it('Appearance section uses SunIcon or MoonIcon (colorblind-safe D-06, POL-02 RED)', () => {
    expect(raw).toMatch(/SunIcon|MoonIcon/)
  })

  // POL-02 / D-06 RED: Light and Dark text labels on the knob
  it('Appearance section uses Light and Dark text labels (colorblind-safe D-06, POL-02 RED)', () => {
    // These are the text labels on the knob — both must be present in source
    expect(raw).toMatch(/['"]Light['"]/)
    expect(raw).toMatch(/['"]Dark['"]/)
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

describe('142-01 / POL-02: SettingsTab Light/Dark toggle DOM behavior (RED — fails until POL-02 lands)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  // POL-02 RED: replaces two-button assertion
  it('the Interface theme control is a role=switch element (POL-02 RED)', () => {
    const props = makeProps({ uiTheme: 'dark' })
    ;({ container, root } = renderSettingsTab(props))
    const toggle = container.querySelector('[role="switch"]')
    expect(toggle).not.toBeNull()
  })

  // POL-02 RED: aria-checked=false for dark mode (switch is "off" for dark = light not selected)
  it('toggle has aria-checked=false when uiTheme is dark (POL-02 RED)', () => {
    const props = makeProps({ uiTheme: 'dark' })
    ;({ container, root } = renderSettingsTab(props))
    const toggle = container.querySelector<HTMLElement>('[role="switch"]')
    expect(toggle).not.toBeNull()
    expect(toggle!.getAttribute('aria-checked')).toBe('false')
  })

  // POL-02 RED: aria-checked=true for light mode
  it('toggle has aria-checked=true when uiTheme is light (POL-02 RED)', () => {
    const props = makeProps({ uiTheme: 'light' })
    ;({ container, root } = renderSettingsTab(props))
    const toggle = container.querySelector<HTMLElement>('[role="switch"]')
    expect(toggle).not.toBeNull()
    expect(toggle!.getAttribute('aria-checked')).toBe('true')
  })

  // POL-02 RED: clicking when dark calls onUiThemeChange('light') — opposite of current
  it('clicking toggle when dark calls onUiThemeChange("light") (POL-02 RED)', () => {
    const onUiThemeChange = vi.fn()
    const props = makeProps({ uiTheme: 'dark', onUiThemeChange })
    ;({ container, root } = renderSettingsTab(props))
    const toggle = container.querySelector<HTMLElement>('[role="switch"]')
    expect(toggle).not.toBeNull()
    toggle!.click()
    expect(onUiThemeChange).toHaveBeenCalledWith('light')
  })

  // POL-02 RED: clicking when light calls onUiThemeChange('dark') — opposite of current
  it('clicking toggle when light calls onUiThemeChange("dark") (POL-02 RED)', () => {
    const onUiThemeChange = vi.fn()
    const props = makeProps({ uiTheme: 'light', onUiThemeChange })
    ;({ container, root } = renderSettingsTab(props))
    const toggle = container.querySelector<HTMLElement>('[role="switch"]')
    expect(toggle).not.toBeNull()
    toggle!.click()
    expect(onUiThemeChange).toHaveBeenCalledWith('dark')
  })
})
