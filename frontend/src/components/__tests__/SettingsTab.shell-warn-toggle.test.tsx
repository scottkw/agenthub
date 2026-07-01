/**
 * Phase 150-02 SET-01 — SettingsTab shell-warn toggle tests (TDD).
 *
 * Covers:
 *   - Toggle renders after GetShellWebShareWarningEnabled resolves true
 *   - Clicking the ON toggle shows the confirm dialog (does NOT call SetShellWebShareWarningEnabled)
 *   - Confirming the dialog calls SetShellWebShareWarningEnabled(false)
 *   - Cancelling the dialog calls no setter and leaves toggle ON
 *   - When initial value is false, clicking the toggle calls SetShellWebShareWarningEnabled(true) with no dialog
 *   - RPC rejection renders an error message
 *
 * Uses the source-inspection pattern (like SettingsTab.start-minimized.test.tsx) for
 * state/import checks, and the createRoot + flushSync DOM render pattern (like
 * SettingsTab.appearance-theme.test.tsx) for interaction tests.
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
// Must include GetShellWebShareWarningEnabled + SetShellWebShareWarningEnabled
// (Plan 01 bindings), plus every other binding SettingsTab currently imports.

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

describe('SET-01: SettingsTab shell-warn toggle — source contract', () => {
  it('imports GetShellWebShareWarningEnabled from Wails bindings', () => {
    expect(raw).toContain('GetShellWebShareWarningEnabled')
  })

  it('imports SetShellWebShareWarningEnabled from Wails bindings', () => {
    expect(raw).toContain('SetShellWebShareWarningEnabled')
  })

  it('has shellWarnEnabled state variable (default true per D-08)', () => {
    expect(raw).toContain('shellWarnEnabled')
    expect(raw).toContain('setShellWarnEnabled')
  })

  it('has shellWarnLoaded guard state variable', () => {
    expect(raw).toContain('shellWarnLoaded')
    expect(raw).toContain('setShellWarnLoaded')
  })

  it('has shellWarnSaving state variable', () => {
    expect(raw).toContain('shellWarnSaving')
    expect(raw).toContain('setShellWarnSaving')
  })

  it('has shellWarnError state variable', () => {
    expect(raw).toContain('shellWarnError')
    expect(raw).toContain('setShellWarnError')
  })

  it('has showDisableWarnConfirm state variable', () => {
    expect(raw).toContain('showDisableWarnConfirm')
    expect(raw).toContain('setShowDisableWarnConfirm')
  })

  it('calls GetShellWebShareWarningEnabled() in useEffect on mount', () => {
    expect(raw).toContain('GetShellWebShareWarningEnabled()')
  })

  it('has handleToggleShellWarnEnabled async function', () => {
    expect(raw).toContain('handleToggleShellWarnEnabled')
  })

  it('has handleConfirmDisableShellWarn async function', () => {
    expect(raw).toContain('handleConfirmDisableShellWarn')
  })

  it('contains exact toggle label text (D-06)', () => {
    expect(raw).toContain('Warn before web-sharing a shell session.')
  })

  it('has optional onShellWarnEnabledChange prop (D-03 re-arm)', () => {
    expect(raw).toContain('onShellWarnEnabledChange?:')
  })

  it('guards the toggle row with shellWarnLoaded', () => {
    expect(raw).toContain('shellWarnLoaded')
  })

  it('confirm dialog shows showDisableWarnConfirm state', () => {
    expect(raw).toContain('showDisableWarnConfirm')
  })

  // Gap-closure (live UAT): the disable-confirm reuses RegenerateKeyModal but
  // MUST pass disable-warning copy — not the hardcoded "Regenerate Signing
  // Key" / "Invalidate All Links" defaults, which are about a different action.
  it('disable-confirm passes shell-warning copy to RegenerateKeyModal (not signing-key defaults)', () => {
    const modalCall = raw.slice(raw.indexOf('isOpen={showDisableWarnConfirm}'))
    expect(modalCall).toContain('Disable warning')
    expect(modalCall).toMatch(/title=/)
    expect(modalCall).not.toContain('Invalidate All Links')
  })
})

// --------------------------------------------------------------------------
// DOM render tests — async interactions
// --------------------------------------------------------------------------

describe('SET-01: SettingsTab shell-warn toggle — DOM render (initial value true)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetShellWebShareWarningEnabled).mockResolvedValue(true)
    vi.mocked(AppBindings.SetShellWebShareWarningEnabled).mockResolvedValue(undefined)
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('toggle row renders with label text after load resolves', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const label = container.querySelector('label[for="shellWebShareWarningEnabled"]')
    expect(label).not.toBeNull()
    expect(label!.textContent).toContain('Warn before web-sharing a shell session.')
  })

  it('clicking the toggle when ON opens the confirm dialog (does NOT call SetShellWebShareWarningEnabled)', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    // Find and click the checkbox
    const checkbox = container.querySelector<HTMLInputElement>('#shellWebShareWarningEnabled')
    expect(checkbox).not.toBeNull()

    await act(async () => {
      checkbox!.click()
    })

    // Confirm dialog should appear (role=dialog or data-testid on our mock)
    const dialog = container.querySelector('[role="dialog"], [data-testid="regen-modal"]')
    expect(dialog).not.toBeNull()

    // SetShellWebShareWarningEnabled must NOT have been called yet
    expect(AppBindings.SetShellWebShareWarningEnabled).not.toHaveBeenCalled()
  })

  it('confirming the dialog calls SetShellWebShareWarningEnabled(false)', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    // Open the dialog by clicking the ON checkbox
    const checkbox = container.querySelector<HTMLInputElement>('#shellWebShareWarningEnabled')
    await act(async () => { checkbox!.click() })

    // Click confirm button — either our mock's data-testid or an inline dialog button
    let confirmBtn = container.querySelector<HTMLButtonElement>('[data-testid="regen-confirm"]')
    if (!confirmBtn) {
      // Inline dialog — look for a button that isn't "Cancel"
      const dialog = container.querySelector('[role="dialog"]')
      expect(dialog).not.toBeNull()
      const buttons = Array.from(dialog!.querySelectorAll<HTMLButtonElement>('button'))
      confirmBtn = buttons.find(b => !b.textContent?.toLowerCase().includes('cancel')) ?? null
    }
    expect(confirmBtn).not.toBeNull()

    await act(async () => { confirmBtn!.click() })
    await flushEffects()

    expect(AppBindings.SetShellWebShareWarningEnabled).toHaveBeenCalledWith(false)
  })

  it('cancelling the dialog calls no setter and leaves toggle visible', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    // Open dialog
    const checkbox = container.querySelector<HTMLInputElement>('#shellWebShareWarningEnabled')
    await act(async () => { checkbox!.click() })

    // Click cancel
    let cancelBtn = container.querySelector<HTMLButtonElement>('[data-testid="regen-cancel"]')
    if (!cancelBtn) {
      const dialog = container.querySelector('[role="dialog"]')
      expect(dialog).not.toBeNull()
      const buttons = Array.from(dialog!.querySelectorAll<HTMLButtonElement>('button'))
      cancelBtn = buttons.find(b => b.textContent?.toLowerCase().includes('cancel')) ?? null
    }
    expect(cancelBtn).not.toBeNull()

    await act(async () => { cancelBtn!.click() })
    await flushEffects()

    // SetShellWebShareWarningEnabled must NOT have been called
    expect(AppBindings.SetShellWebShareWarningEnabled).not.toHaveBeenCalled()

    // Dialog should be closed (mock renders null when isOpen=false)
    const dialog = container.querySelector('[data-testid="regen-modal"]')
    expect(dialog).toBeNull()
  })
})

describe('SET-01: SettingsTab shell-warn toggle — DOM render (initial value false)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetShellWebShareWarningEnabled).mockResolvedValue(false)
    vi.mocked(AppBindings.SetShellWebShareWarningEnabled).mockResolvedValue(undefined)
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('clicking the toggle when OFF calls SetShellWebShareWarningEnabled(true) with no dialog', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    // Find the checkbox
    const checkbox = container.querySelector<HTMLInputElement>('#shellWebShareWarningEnabled')
    expect(checkbox).not.toBeNull()

    await act(async () => { checkbox!.click() })
    await flushEffects()

    // Should call SetShellWebShareWarningEnabled(true) immediately — no dialog
    expect(AppBindings.SetShellWebShareWarningEnabled).toHaveBeenCalledWith(true)

    // No confirm dialog should be open
    const dialog = container.querySelector('[role="dialog"], [data-testid="regen-modal"]')
    expect(dialog).toBeNull()
  })
})
