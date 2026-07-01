/**
 * Phase 167-04 NTF-04 — SettingsTab notify-on-waiting toggle tests (TDD).
 *
 * Covers:
 *   - Toggle renders inside the Behavior section (settings-behavior), NOT
 *     Session Behavior (settings-session-behavior) — the LOCKED user correction.
 *   - Default OFF (unchecked) before/after GetNotifyOnWaiting resolves false.
 *   - On mount, GetNotifyOnWaiting() resolves and sets the checked state.
 *   - Flipping the toggle calls SetNotifyOnWaiting(next) and reflects it in the UI.
 *   - A SetNotifyOnWaiting rejection surfaces an inline error and does not flip
 *     the displayed checked state.
 *
 * Mirrors the shape of SettingsTab.shell-warn-toggle.test.tsx (source-inspection
 * + createRoot/flushSync DOM render pattern), but the notify toggle is an
 * INSTANT toggle with no confirm dialog — it mirrors handleToggleMinimized
 * (SettingsTab.start-minimized.test.tsx), not handleToggleShellWarnEnabled.
 */
import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { act } from 'react-dom/test-utils'
import { SettingsTab } from '../SettingsTab'
import raw from '../SettingsTab.tsx?raw'
import * as AppBindings from '../../wailsjs/go/main/App'

// --- Wails mock -----------------------------------------------------------
// Must include GetNotifyOnWaiting + SetNotifyOnWaiting (Plan 03 bindings),
// plus every other binding SettingsTab currently imports.

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
  RegenerateKeyModal: ({ isOpen, onConfirm, onCancel }: { isOpen: boolean; onConfirm: () => Promise<void>; onCancel: () => void }) => {
    if (!isOpen) return null
    return React.createElement('div', { 'data-testid': 'regen-modal' },
      React.createElement('button', { onClick: onCancel, 'data-testid': 'regen-cancel' }, 'Cancel'),
      React.createElement('button', { onClick: () => void onConfirm(), 'data-testid': 'regen-confirm' }, 'Confirm')
    )
  }
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

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

function makeProps(overrides: Partial<Parameters<typeof SettingsTab>[0]> = {}) {
  return {
    clis: [],
    tailscaleHealth: null,
    webServerMode: null as null,
    onWebServerStateChange: vi.fn().mockResolvedValue(undefined),
    selectedTheme: 'Tomorrow_Night',
    onThemeChange: vi.fn(),
    uiTheme: 'dark' as const,
    onUiThemeChange: vi.fn(),
    onPluginToggleSideEffect: undefined,
    ...overrides,
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

// Helper to flush all pending microtasks (for async state updates after useEffect).
async function flushEffects() {
  await act(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, 0))
  })
}

// --------------------------------------------------------------------------
// Source-inspection tests (fast, no DOM mount needed)
// --------------------------------------------------------------------------

describe('NTF-04: SettingsTab notify-on-waiting toggle — source contract', () => {
  it('imports GetNotifyOnWaiting from Wails bindings', () => {
    expect(raw).toContain('GetNotifyOnWaiting')
  })

  it('imports SetNotifyOnWaiting from Wails bindings', () => {
    expect(raw).toContain('SetNotifyOnWaiting')
  })

  it('has notifyOnWaiting state variable defaulting to false (D-04)', () => {
    expect(raw).toContain('notifyOnWaiting')
    expect(raw).toContain('setNotifyOnWaiting')
    expect(raw).toContain('const [notifyOnWaiting, setNotifyOnWaiting] = useState(false)')
  })

  it('has notifyOnWaitingLoaded guard state variable', () => {
    expect(raw).toContain('notifyOnWaitingLoaded')
    expect(raw).toContain('setNotifyOnWaitingLoaded')
  })

  it('has notifyOnWaitingSaving state variable', () => {
    expect(raw).toContain('notifyOnWaitingSaving')
    expect(raw).toContain('setNotifyOnWaitingSaving')
  })

  it('has notifyOnWaitingError state variable', () => {
    expect(raw).toContain('notifyOnWaitingError')
    expect(raw).toContain('setNotifyOnWaitingError')
  })

  it('calls GetNotifyOnWaiting() in useEffect on mount', () => {
    expect(raw).toContain('GetNotifyOnWaiting()')
  })

  it('has handleToggleNotifyOnWaiting async function', () => {
    expect(raw).toContain('async function handleToggleNotifyOnWaiting')
  })

  it('contains the exact toggle label text', () => {
    expect(raw).toContain('Notify me when a session is awaiting input')
  })

  it('renders the toggle inside the Behavior section, not Session Behavior (LOCKED correction)', () => {
    const behaviorHeadingIdx = raw.indexOf('id="settings-behavior"')
    const sessionBehaviorHeadingIdx = raw.indexOf('id="settings-session-behavior"')
    const toggleIdx = raw.indexOf('id="notifyOnWaiting"')
    expect(behaviorHeadingIdx).toBeGreaterThan(-1)
    expect(sessionBehaviorHeadingIdx).toBeGreaterThan(-1)
    expect(toggleIdx).toBeGreaterThan(-1)
    // Toggle must sit after the Behavior heading and before the Session
    // Behavior heading — i.e. physically inside the Behavior section.
    expect(toggleIdx).toBeGreaterThan(behaviorHeadingIdx)
    expect(toggleIdx).toBeLessThan(sessionBehaviorHeadingIdx)
  })
})

// --------------------------------------------------------------------------
// DOM render tests — async interactions
// --------------------------------------------------------------------------

describe('NTF-04: SettingsTab notify-on-waiting toggle — DOM render (default OFF)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetNotifyOnWaiting).mockResolvedValue(false)
    vi.mocked(AppBindings.SetNotifyOnWaiting).mockResolvedValue(undefined)
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('toggle row renders unchecked with label text after load resolves false', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const label = container.querySelector('label[for="notifyOnWaiting"]')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Notify me when a session is awaiting input')

    const checkbox = container.querySelector<HTMLInputElement>('#notifyOnWaiting')
    expect(checkbox).not.toBeNull()
    expect(checkbox!.checked).toBe(false)
    expect(label!.className).not.toContain('settings-panel__toggle-row--checked')
  })

  it('flipping the toggle calls SetNotifyOnWaiting(true) and reflects checked state', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#notifyOnWaiting')
    expect(checkbox).not.toBeNull()

    await act(async () => { checkbox!.click() })
    await flushEffects()

    expect(AppBindings.SetNotifyOnWaiting).toHaveBeenCalledWith(true)
    expect(checkbox!.checked).toBe(true)

    const label = container.querySelector('label[for="notifyOnWaiting"]')
    expect(label!.className).toContain('settings-panel__toggle-row--checked')
  })

  it('a SetNotifyOnWaiting rejection surfaces an inline error and does not flip the displayed state', async () => {
    vi.mocked(AppBindings.SetNotifyOnWaiting).mockRejectedValueOnce(new Error('daemon unreachable'))

    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#notifyOnWaiting')
    expect(checkbox).not.toBeNull()

    await act(async () => { checkbox!.click() })
    await flushEffects()

    expect(AppBindings.SetNotifyOnWaiting).toHaveBeenCalledWith(true)
    // Displayed state must NOT flip to checked on failure.
    expect(checkbox!.checked).toBe(false)

    const errorEl = container.querySelector('.settings-panel__error')
    expect(errorEl).not.toBeNull()
    expect(errorEl!.textContent).toContain('daemon unreachable')
  })
})

describe('NTF-04: SettingsTab notify-on-waiting toggle — DOM render (loaded ON)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetNotifyOnWaiting).mockResolvedValue(true)
    vi.mocked(AppBindings.SetNotifyOnWaiting).mockResolvedValue(undefined)
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('on mount, GetNotifyOnWaiting() resolves true and sets the checked state', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#notifyOnWaiting')
    expect(checkbox).not.toBeNull()
    expect(checkbox!.checked).toBe(true)
  })

  it('flipping the toggle OFF calls SetNotifyOnWaiting(false)', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#notifyOnWaiting')
    await act(async () => { checkbox!.click() })
    await flushEffects()

    expect(AppBindings.SetNotifyOnWaiting).toHaveBeenCalledWith(false)
    expect(checkbox!.checked).toBe(false)
  })
})
