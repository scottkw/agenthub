/**
 * Phase 169-02 (FIX-05, #120, v4.2) — SettingsTab permission-limited Tailscale
 * status tests.
 *
 * Covers (per 169-02-PLAN.md Task 1 <behavior>):
 *   - Rendering with tailscaleHealth.permissionLimited=true (connected=false):
 *     the status text reads a distinct label ("Permission Limited"), NOT
 *     "Connected" and NOT "Not Connected"; the description contains the
 *     grant-admin / Homebrew-build guidance; the status dot does NOT carry
 *     the 'ok' modifier class.
 *   - Rendering with connected=true (permissionLimited absent/false): the
 *     status text still reads "Connected" (regression — healthy path intact).
 *   - The word "Connected" is not present anywhere in the permission-limited
 *     render output (guards against a false-positive impression — SC3).
 *
 * Mirrors SettingsTab.notify-permission-hint.test.tsx for the render harness
 * + wails-mock idiom (GetTailscaleStatus is unused here — the health object
 * is passed directly via the tailscaleHealth prop, not fetched).
 */
import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { act } from 'react-dom/test-utils'
import { SettingsTab } from '../SettingsTab'

// --- Wails mock -----------------------------------------------------------
// Every binding SettingsTab currently imports, mirroring
// SettingsTab.notify-permission-hint.test.tsx.

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

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  EventsOn: vi.fn(() => vi.fn()),
}))

vi.mock('../RegenerateKeyModal', () => ({
  RegenerateKeyModal: ({ isOpen }: { isOpen: boolean }) => {
    if (!isOpen) return null
    return React.createElement('div', { 'data-testid': 'regen-modal' })
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

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

type TailscaleHealth = NonNullable<Parameters<typeof SettingsTab>[0]['tailscaleHealth']>

function makeHealth(overrides: Partial<TailscaleHealth> = {}): TailscaleHealth {
  return {
    installed: true,
    connected: false,
    hasCerts: false,
    ip: '',
    domain: '',
    binaryFound: true,
    daemonUp: false,
    platformHint: 'darwin',
    ...overrides,
  }
}

function makeProps(overrides: Partial<Parameters<typeof SettingsTab>[0]> = {}) {
  return {
    clis: [],
    tailscaleHealth: null as TailscaleHealth | null,
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

async function flushEffects() {
  await act(async () => {
    await new Promise<void>((resolve) => setTimeout(resolve, 0))
  })
}

describe('FIX-05/169-02: SettingsTab permission-limited Tailscale status', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('shows a distinct "Permission Limited" label with actionable guidance, and never the ok dot', async () => {
    const props = makeProps({
      tailscaleHealth: makeHealth({ daemonUp: true, permissionLimited: true }),
    })
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const statusText = container.querySelector('.ts-status__text')
    expect(statusText).not.toBeNull()
    expect(statusText!.textContent).toBe('Permission Limited')

    const dot = container.querySelector('.ts-status__dot')
    expect(dot).not.toBeNull()
    expect(dot!.className).not.toContain('ts-status__dot--ok')
    expect(dot!.className).toContain('ts-status__dot--warn')

    // Actionable guidance: grant admin, or the Homebrew build remedy.
    expect(container.textContent).toContain('admin')
    expect(container.textContent).toContain('Homebrew')
  })

  it('never renders the word "Connected" in the permission-limited status/description copy (SC3)', async () => {
    // Scoped to the Tailscale status field-group's status text + description
    // (the pre-existing "Show diagnostics" step-cascade below it unconditionally
    // labels its 3rd step "Connected to Tailscale" for every non-connected state
    // — that generic step label predates this plan and is not "the
    // permission-limited copy" the plan's hard truth is guarding).
    const props = makeProps({
      tailscaleHealth: makeHealth({ daemonUp: true, permissionLimited: true }),
    })
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const fieldGroup = container.querySelector('.ts-status')!.closest('.settings-panel__field-group') as HTMLElement
    expect(fieldGroup).not.toBeNull()
    const clone = fieldGroup.cloneNode(true) as HTMLElement
    clone.querySelector('details')?.remove()

    expect(clone.textContent).not.toContain('Connected')
  })

  it('still renders "Connected" on the healthy path (regression, permissionLimited absent)', async () => {
    const props = makeProps({
      tailscaleHealth: makeHealth({
        connected: true,
        hasCerts: true,
        ip: '100.64.0.1',
        domain: 'my-node.ts.net',
        daemonUp: true,
      }),
    })
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const statusText = container.querySelector('.ts-status__text')
    expect(statusText).not.toBeNull()
    expect(statusText!.textContent).toBe('Connected')

    const dot = container.querySelector('.ts-status__dot')
    expect(dot).not.toBeNull()
    expect(dot!.className).toContain('ts-status__dot--ok')
  })

  it('still renders "Connected" when permissionLimited is explicitly false (regression)', async () => {
    const props = makeProps({
      tailscaleHealth: makeHealth({
        connected: true,
        hasCerts: true,
        ip: '100.64.0.1',
        domain: 'my-node.ts.net',
        daemonUp: true,
        permissionLimited: false,
      }),
    })
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    const statusText = container.querySelector('.ts-status__text')
    expect(statusText!.textContent).toBe('Connected')
  })
})
