/**
 * Phase 168-04 UX-01 — SettingsTab stay-on-hub-after-create toggle tests (TDD).
 *
 * Covers:
 *   - Toggle renders inside the Session Behavior section
 *     (settings-session-behavior), NOT the Behavior section
 *     (settings-behavior) where notifyOnWaiting lives (D-08).
 *   - Default OFF (unchecked) before/after GetStayOnHubAfterCreate resolves
 *     false (D-09).
 *   - On mount, GetStayOnHubAfterCreate() resolves and sets the checked state.
 *   - Flipping the toggle calls SetStayOnHubAfterCreate(next) and reflects it
 *     in the UI.
 *   - A SetStayOnHubAfterCreate rejection surfaces an inline error and does
 *     not flip the displayed checked state.
 *
 * Mirrors the shape of SettingsTab.notify-toggle.test.tsx (source-inspection
 * + createRoot/flushSync DOM render pattern); the toggle itself is an INSTANT
 * toggle with no confirm dialog, mirroring handleToggleNotifyOnWaiting.
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
// Must include GetStayOnHubAfterCreate + SetStayOnHubAfterCreate (Plan 04
// bindings), plus every other binding SettingsTab currently imports.

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

describe('UX-01: SettingsTab stay-on-hub-after-create toggle — source contract', () => {
  it('imports GetStayOnHubAfterCreate from Wails bindings', () => {
    expect(raw).toContain('GetStayOnHubAfterCreate')
  })

  it('imports SetStayOnHubAfterCreate from Wails bindings', () => {
    expect(raw).toContain('SetStayOnHubAfterCreate')
  })

  it('has stayOnHubAfterCreate state variable defaulting to false (D-09)', () => {
    expect(raw).toContain('stayOnHubAfterCreate')
    expect(raw).toContain('setStayOnHubAfterCreate')
    expect(raw).toContain('const [stayOnHubAfterCreate, setStayOnHubAfterCreate] = useState(false)')
  })

  it('has stayOnHubAfterCreateLoaded guard state variable', () => {
    expect(raw).toContain('stayOnHubAfterCreateLoaded')
    expect(raw).toContain('setStayOnHubAfterCreateLoaded')
  })

  it('has stayOnHubAfterCreateSaving state variable', () => {
    expect(raw).toContain('stayOnHubAfterCreateSaving')
    expect(raw).toContain('setStayOnHubAfterCreateSaving')
  })

  it('has stayOnHubAfterCreateError state variable', () => {
    expect(raw).toContain('stayOnHubAfterCreateError')
    expect(raw).toContain('setStayOnHubAfterCreateError')
  })

  it('calls GetStayOnHubAfterCreate() in useEffect on mount', () => {
    expect(raw).toContain('GetStayOnHubAfterCreate()')
  })

  it('has handleToggleStayOnHub async function', () => {
    expect(raw).toContain('async function handleToggleStayOnHub')
  })

  it('contains the exact toggle label text', () => {
    expect(raw).toContain('Stay on Hub after creating a session')
  })

  it('renders the toggle inside the Session Behavior section, not the Behavior section (D-08)', () => {
    const behaviorHeadingIdx = raw.indexOf('id="settings-behavior"')
    const sessionBehaviorHeadingIdx = raw.indexOf('id="settings-session-behavior"')
    const toggleIdx = raw.indexOf('id="stayOnHubAfterCreate"')
    expect(behaviorHeadingIdx).toBeGreaterThan(-1)
    expect(sessionBehaviorHeadingIdx).toBeGreaterThan(-1)
    expect(toggleIdx).toBeGreaterThan(-1)
    // Toggle must sit after the Session Behavior heading — i.e. physically
    // inside the Session Behavior section, NOT the earlier Behavior section.
    expect(toggleIdx).toBeGreaterThan(sessionBehaviorHeadingIdx)
    expect(toggleIdx).toBeGreaterThan(behaviorHeadingIdx)
  })
})

// --------------------------------------------------------------------------
// DOM render tests — async interactions
// --------------------------------------------------------------------------

describe('UX-01: SettingsTab stay-on-hub-after-create toggle — DOM render (default OFF)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetStayOnHubAfterCreate).mockResolvedValue(false)
    vi.mocked(AppBindings.SetStayOnHubAfterCreate).mockResolvedValue(undefined)
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

    const label = container.querySelector('label[for="stayOnHubAfterCreate"]')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Stay on Hub after creating a session')

    const checkbox = container.querySelector<HTMLInputElement>('#stayOnHubAfterCreate')
    expect(checkbox).not.toBeNull()
    expect(checkbox!.checked).toBe(false)
    expect(label!.className).not.toContain('settings-panel__toggle-row--checked')
  })

  it('flipping the toggle calls SetStayOnHubAfterCreate(true) and reflects checked state', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#stayOnHubAfterCreate')
    expect(checkbox).not.toBeNull()

    await act(async () => { checkbox!.click() })
    await flushEffects()

    expect(AppBindings.SetStayOnHubAfterCreate).toHaveBeenCalledWith(true)
    expect(checkbox!.checked).toBe(true)

    const label = container.querySelector('label[for="stayOnHubAfterCreate"]')
    expect(label!.className).toContain('settings-panel__toggle-row--checked')
  })

  it('a SetStayOnHubAfterCreate rejection surfaces an inline error and does not flip the displayed state', async () => {
    vi.mocked(AppBindings.SetStayOnHubAfterCreate).mockRejectedValueOnce(new Error('daemon unreachable'))

    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#stayOnHubAfterCreate')
    expect(checkbox).not.toBeNull()

    await act(async () => { checkbox!.click() })
    await flushEffects()

    expect(AppBindings.SetStayOnHubAfterCreate).toHaveBeenCalledWith(true)
    // Displayed state must NOT flip to checked on failure.
    expect(checkbox!.checked).toBe(false)

    const errorEl = container.querySelector('.settings-panel__error')
    expect(errorEl).not.toBeNull()
    expect(errorEl!.textContent).toContain('daemon unreachable')
  })
})

describe('UX-01: SettingsTab stay-on-hub-after-create toggle — DOM render (loaded ON)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetStayOnHubAfterCreate).mockResolvedValue(true)
    vi.mocked(AppBindings.SetStayOnHubAfterCreate).mockResolvedValue(undefined)
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('on mount, GetStayOnHubAfterCreate() resolves true and sets the checked state', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#stayOnHubAfterCreate')
    expect(checkbox).not.toBeNull()
    expect(checkbox!.checked).toBe(true)
  })

  it('flipping the toggle OFF calls SetStayOnHubAfterCreate(false)', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const checkbox = container.querySelector<HTMLInputElement>('#stayOnHubAfterCreate')
    await act(async () => { checkbox!.click() })
    await flushEffects()

    expect(AppBindings.SetStayOnHubAfterCreate).toHaveBeenCalledWith(false)
    expect(checkbox!.checked).toBe(false)
  })
})
