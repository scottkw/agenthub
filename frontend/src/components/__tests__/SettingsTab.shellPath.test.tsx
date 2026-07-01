/**
 * SHELL-11 test contract — UI-SPEC §4 (eight assertions).
 * Locks the Settings → Paths "Shell binary" row introduced in Phase 107-03.
 */
import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { act } from 'react-dom/test-utils'
import { SettingsTab } from '../SettingsTab'
import * as AppMock from '../../wailsjs/go/main/App'

// Mock all Wails bindings used by SettingsTab.
vi.mock('../../wailsjs/go/main/App', () => ({
  // CLI paths
  UpdateCLIPath: vi.fn().mockResolvedValue(undefined),
  GetCLIPaths: vi.fn().mockResolvedValue({}),
  OpenFileDialog: vi.fn().mockResolvedValue(''),
  // Web server
  StartWebServer: vi.fn().mockResolvedValue(undefined),
  StopWebServer: vi.fn().mockResolvedValue(undefined),
  GetWebServerURL: vi.fn().mockResolvedValue('https://example.ts.net'),
  IsWebServerRunning: vi.fn().mockResolvedValue(false),
  // CT disclosure
  HasCTDisclosure: vi.fn().mockResolvedValue(true),
  AcknowledgeCTDisclosure: vi.fn().mockResolvedValue(undefined),
  // Local network
  GetLocalNetworkPassword: vi.fn().mockResolvedValue(''),
  GetWebServerQRCode: vi.fn().mockResolvedValue(''),
  // Toggles
  GetStartMinimized: vi.fn().mockResolvedValue(false),
  SetStartMinimized: vi.fn().mockResolvedValue(undefined),
  GetAutoCloseSession: vi.fn().mockResolvedValue(true),
  SetAutoCloseSession: vi.fn().mockResolvedValue(undefined),
  // Security
  RegenerateSigningKey: vi.fn().mockResolvedValue(undefined),
  // Plugins
  GetPluginSettings: vi.fn().mockResolvedValue({ searchConfig: {}, webLinksConfig: {}, imageConfig: {} }),
  SetPluginSettings: vi.fn().mockResolvedValue(undefined),
  SetSearchConfig: vi.fn().mockResolvedValue(undefined),
  SetWebLinksConfig: vi.fn().mockResolvedValue(undefined),
  SetImageConfig: vi.fn().mockResolvedValue(undefined),
  // Tailscale
  GetTailscaleStatus: vi.fn().mockResolvedValue(null),
  // Theme
  NotifyThemeChange: vi.fn().mockResolvedValue(undefined),
  // Shell path — SHELL-11 new bindings
  GetShellPath: vi.fn().mockResolvedValue('/bin/zsh'),
  SetShellPath: vi.fn().mockResolvedValue(undefined),
  GetShellWebShareWarningEnabled: vi.fn().mockResolvedValue(true),
  SetShellWebShareWarningEnabled: vi.fn().mockResolvedValue(undefined),
  GetNotifyOnWaiting: vi.fn().mockResolvedValue(false),
  SetNotifyOnWaiting: vi.fn().mockResolvedValue(undefined),
}))

// Mock sub-components that use their own imports.
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

const defaultProps = {
  clis: [
    { Name: 'claude', Path: '/usr/local/bin/claude' },
  ],
  tailscaleHealth: null,
  webServerMode: null as null,
  onWebServerStateChange: vi.fn().mockResolvedValue(undefined),
  selectedTheme: 'tokyonight',
  onThemeChange: vi.fn(),
  uiTheme: 'dark' as const,
  onUiThemeChange: vi.fn(),
}

function renderSettings(props = defaultProps) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(SettingsTab, props))
  })
  return { container, root }
}

describe('SHELL-11: Settings → Paths "Shell binary" row', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppMock.GetShellPath).mockResolvedValue('/bin/zsh')
    vi.mocked(AppMock.SetShellPath).mockResolvedValue(undefined)
    vi.mocked(AppMock.GetCLIPaths).mockResolvedValue({})
    vi.mocked(AppMock.IsWebServerRunning).mockResolvedValue(false)
    vi.mocked(AppMock.HasCTDisclosure).mockResolvedValue(true)
    vi.mocked(AppMock.GetStartMinimized).mockResolvedValue(false)
    vi.mocked(AppMock.GetAutoCloseSession).mockResolvedValue(true)
    vi.mocked(AppMock.OpenFileDialog).mockResolvedValue('')
  })

  afterEach(() => {
    root?.unmount()
    container?.remove()
  })

  // Assertion 1: Renders a <tr> with settings-panel__cli-name cell containing text "shell".
  it('renders a table row with cli-name cell containing text "shell"', async () => {
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })
    const cliNameCells = container.querySelectorAll('.settings-panel__cli-name')
    const shellCell = Array.from(cliNameCells).find((el) => el.textContent?.trim() === 'shell')
    expect(shellCell).not.toBeUndefined()
    expect(shellCell!.closest('tr')).not.toBeNull()
  })

  // Assertion 2: Input id="settings-shell-path" present with aria-label="Shell binary path".
  it('renders input with id="settings-shell-path" and aria-label="Shell binary path"', async () => {
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })
    const input = container.querySelector('#settings-shell-path') as HTMLInputElement | null
    expect(input).not.toBeNull()
    expect(input!.getAttribute('aria-label')).toBe('Shell binary path')
  })

  // Assertion 3: Input value initializes from GetShellPath() mock return.
  it('input value initializes from GetShellPath() (e.g. /bin/zsh)', async () => {
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })
    const input = container.querySelector('#settings-shell-path') as HTMLInputElement | null
    expect(input).not.toBeNull()
    expect(input!.value).toBe('/bin/zsh')
  })

  // Assertion 4: Typing in the input updates local state (controlled input behavior).
  it('typing in the input updates the value (controlled input)', async () => {
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })
    const input = container.querySelector('#settings-shell-path') as HTMLInputElement
    // Simulate React-controlled input change.
    const proto = Object.getPrototypeOf(input)
    const desc = Object.getOwnPropertyDescriptor(proto, 'value')
    desc?.set?.call(input, '/bin/bash')
    flushSync(() => {
      input.dispatchEvent(new Event('input', { bubbles: true }))
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(input.value).toBe('/bin/bash')
  })

  // Assertion 5: Browse button calls OpenFileDialog and updates input with return value.
  it('Browse button calls OpenFileDialog and updates input value with result', async () => {
    vi.mocked(AppMock.OpenFileDialog).mockResolvedValue('/usr/local/bin/bash')
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })
    // Find the Browse button in the shell row (the last Browse button before tailscale row).
    const shellRow = Array.from(container.querySelectorAll('tr')).find((tr) => {
      const cell = tr.querySelector('.settings-panel__cli-name')
      return cell?.textContent?.trim() === 'shell'
    })
    expect(shellRow).not.toBeUndefined()
    const browseBtn = shellRow!.querySelector('.settings-panel__browse-btn') as HTMLButtonElement
    expect(browseBtn).not.toBeNull()
    await act(async () => {
      flushSync(() => browseBtn.click())
      await Promise.resolve()
      await Promise.resolve() // extra tick for async handler
    })
    const input = container.querySelector('#settings-shell-path') as HTMLInputElement
    expect(AppMock.OpenFileDialog).toHaveBeenCalled()
    expect(input.value).toBe('/usr/local/bin/bash')
  })

  // Assertion 6: Save button calls SetShellPath with current input value.
  it('Save Paths button calls SetShellPath with the current input value', async () => {
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })

    // Change input to /bin/bash first.
    const input = container.querySelector('#settings-shell-path') as HTMLInputElement
    const proto = Object.getPrototypeOf(input)
    const desc = Object.getOwnPropertyDescriptor(proto, 'value')
    desc?.set?.call(input, '/bin/bash')
    flushSync(() => {
      input.dispatchEvent(new Event('input', { bubbles: true }))
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })

    // Click Save Paths.
    const saveBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.includes('Save Paths') || b.textContent?.includes('Saving')
    ) as HTMLButtonElement
    expect(saveBtn).not.toBeUndefined()
    await act(async () => {
      flushSync(() => saveBtn.click())
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(AppMock.SetShellPath).toHaveBeenCalledWith('/bin/bash')
  })

  // Assertion 7: On SetShellPath rejection, error paragraph with role="alert" renders.
  it('on SetShellPath rejection, renders error paragraph with role="alert"', async () => {
    vi.mocked(AppMock.SetShellPath).mockRejectedValue(new Error('path /foo does not exist or is not executable'))
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })

    const saveBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.includes('Save Paths') || b.textContent?.includes('Saving')
    ) as HTMLButtonElement
    await act(async () => {
      flushSync(() => saveBtn.click())
      await Promise.resolve()
      await Promise.resolve()
    })

    const errorPara = container.querySelector('[role="alert"]') as HTMLElement | null
    expect(errorPara).not.toBeNull()
    expect(errorPara!.textContent).toContain('path /foo does not exist or is not executable')
  })

  // Assertion 8: Error paragraph id matches input's aria-describedby.
  it('error paragraph id matches input aria-describedby (settings-shell-path-desc)', async () => {
    vi.mocked(AppMock.SetShellPath).mockRejectedValue(new Error('invalid path'))
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })

    const input = container.querySelector('#settings-shell-path') as HTMLInputElement
    expect(input.getAttribute('aria-describedby')).toBe('settings-shell-path-desc')

    const saveBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.includes('Save Paths') || b.textContent?.includes('Saving')
    ) as HTMLButtonElement
    await act(async () => {
      flushSync(() => saveBtn.click())
      await Promise.resolve()
      await Promise.resolve()
    })

    const errorPara = container.querySelector('#settings-shell-path-desc') as HTMLElement | null
    expect(errorPara).not.toBeNull()
    expect(errorPara!.getAttribute('role')).toBe('alert')
  })

  // Assertion WR-02: When SetShellPath fails, the "Saved!" indicator must NOT appear.
  // This is the regression test for the false-positive "Saved!" when shell-path
  // validation returns 400. The button must stay in its normal (not Saved!) state.
  it('WR-02: on SetShellPath failure, Saved! indicator does NOT appear', async () => {
    vi.mocked(AppMock.SetShellPath).mockRejectedValue(new Error('path /bad is a directory, not an executable'))
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })

    const saveBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.includes('Save Paths') || b.textContent?.includes('Saving')
    ) as HTMLButtonElement
    expect(saveBtn).not.toBeUndefined()

    await act(async () => {
      flushSync(() => saveBtn.click())
      await Promise.resolve()
      await Promise.resolve()
    })

    // The inline error must be visible
    const errorPara = container.querySelector('[role="alert"]') as HTMLElement | null
    expect(errorPara).not.toBeNull()
    expect(errorPara!.textContent).toContain('is a directory')

    // The save button must NOT say "Saved!" (no false success indicator)
    const allButtons = Array.from(container.querySelectorAll('button'))
    const savedBtn = allButtons.find((b) => b.textContent?.includes('Saved!'))
    expect(savedBtn).toBeUndefined()
  })

  // Assertion WR-02b: When SetShellPath succeeds, the "Saved!" indicator DOES appear.
  it('WR-02b: on SetShellPath success, Saved! indicator appears', async () => {
    vi.mocked(AppMock.SetShellPath).mockResolvedValue(undefined)
    ;({ container, root } = renderSettings())
    await act(async () => { await Promise.resolve() })

    const saveBtn = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent?.includes('Save Paths') || b.textContent?.includes('Saving')
    ) as HTMLButtonElement
    expect(saveBtn).not.toBeUndefined()

    await act(async () => {
      flushSync(() => saveBtn.click())
      await Promise.resolve()
      await Promise.resolve()
    })

    // No inline error
    const errorPara = container.querySelector('[role="alert"]')
    expect(errorPara).toBeNull()

    // The save button must show "Saved!" on success
    const allButtons = Array.from(container.querySelectorAll('button'))
    const savedBtn = allButtons.find((b) => b.textContent?.includes('Saved!'))
    expect(savedBtn).not.toBeUndefined()
  })
})
