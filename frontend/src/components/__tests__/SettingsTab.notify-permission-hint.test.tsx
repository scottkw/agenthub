/**
 * Phase 167-07 (M-41 gap closure, frontend half) — SettingsTab notification
 * permission-denied hint tests (TDD).
 *
 * Covers:
 *   - The hint is NOT rendered before any `notification:permission-denied`
 *     event fires (hidden by default).
 *   - The hint IS rendered in the Behavior section once the backend emits
 *     `notification:permission-denied` (167-06's onNotificationAuthResult
 *     callback), directing the user to System Settings > Notifications.
 *   - The EventsOn subscription's returned unsubscribe function is called on
 *     unmount (T-167-13 — no leaked subscription / duplicate handlers).
 *
 * Mirrors SettingsTab.notify-toggle.test.tsx for the render harness + wails
 * mocks, but exercises the EventsOn('notification:permission-denied', ...)
 * subscription instead of the toggle's Get/Set round trip.
 */
import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { act } from 'react-dom/test-utils'
import { SettingsTab } from '../SettingsTab'
import * as AppBindings from '../../wailsjs/go/main/App'
import * as WailsRuntime from '../../wailsjs/wailsjs/runtime/runtime'

// --- Wails mock -----------------------------------------------------------
// Every binding SettingsTab currently imports, mirroring notify-toggle.test.tsx.

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

// Captures the handler registered for 'notification:permission-denied' so the
// test can invoke it directly, and returns an unsubscribe spy to assert
// cleanup-on-unmount (T-167-13).
const unsubscribeSpy = vi.fn()
let capturedPermissionDeniedHandler: (() => void) | null = null
let capturedPermissionGrantedHandler: (() => void) | null = null

vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
  ClipboardSetText: vi.fn().mockResolvedValue(undefined),
  EventsOn: vi.fn((eventName: string, handler: (...args: unknown[]) => void) => {
    if (eventName === 'notification:permission-denied') {
      capturedPermissionDeniedHandler = handler as () => void
    }
    if (eventName === 'notification:permission-granted') {
      capturedPermissionGrantedHandler = handler as () => void
    }
    return unsubscribeSpy
  }),
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

const HINT_TEXT_FRAGMENT = 'System Settings'

describe('NTF-04/M-41 gap closure: SettingsTab notification-permission-denied hint', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  beforeEach(() => {
    vi.mocked(AppBindings.GetNotifyOnWaiting).mockResolvedValue(false)
    vi.mocked(AppBindings.SetNotifyOnWaiting).mockResolvedValue(undefined)
    capturedPermissionDeniedHandler = null
    capturedPermissionGrantedHandler = null
    unsubscribeSpy.mockClear()
  })

  afterEach(() => {
    root.unmount()
    container.remove()
    vi.clearAllMocks()
  })

  it('subscribes to the notification:permission-denied event on mount', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    expect(WailsRuntime.EventsOn).toHaveBeenCalledWith(
      'notification:permission-denied',
      expect.any(Function)
    )
    expect(capturedPermissionDeniedHandler).not.toBeNull()
  })

  it('does NOT render the hint before any permission-denied event fires', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    expect(container.textContent).not.toContain(HINT_TEXT_FRAGMENT)
  })

  it('renders the hint in the Behavior section once the event fires', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    expect(capturedPermissionDeniedHandler).not.toBeNull()
    await act(async () => {
      capturedPermissionDeniedHandler!()
    })
    await flushEffects()

    expect(container.textContent).toContain(HINT_TEXT_FRAGMENT)
    expect(container.textContent).toContain('Notifications')

    // Must live inside the Behavior section (between the Behavior heading and
    // the Session Behavior heading), alongside the notifyOnWaiting toggle.
    const behaviorHeading = container.querySelector('#settings-behavior')
    const sessionBehaviorHeading = container.querySelector('#settings-session-behavior')
    expect(behaviorHeading).not.toBeNull()
    expect(sessionBehaviorHeading).not.toBeNull()

    const html = container.innerHTML
    const behaviorIdx = html.indexOf('id="settings-behavior"')
    const sessionBehaviorIdx = html.indexOf('id="settings-session-behavior"')
    const hintIdx = html.indexOf(HINT_TEXT_FRAGMENT)
    expect(hintIdx).toBeGreaterThan(behaviorIdx)
    expect(hintIdx).toBeLessThan(sessionBehaviorIdx)
  })

  it('calls the EventsOn-returned unsubscribe function for both subscriptions on unmount', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    expect(unsubscribeSpy).not.toHaveBeenCalled()
    root.unmount()
    // Two subscriptions (denied + granted, WR-01) → two unsubscribe calls.
    expect(unsubscribeSpy).toHaveBeenCalledTimes(2)
  })

  // WR-01 (code review, Phase 167): the hint must self-heal instead of staying
  // stuck for the whole session once the user actually fixes permissions.
  it('subscribes to the notification:permission-granted event on mount', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    expect(WailsRuntime.EventsOn).toHaveBeenCalledWith(
      'notification:permission-granted',
      expect.any(Function)
    )
    expect(capturedPermissionGrantedHandler).not.toBeNull()
  })

  it('clears the hint when a permission-granted event fires after a denial', async () => {
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    // Denial surfaces the hint...
    await act(async () => { capturedPermissionDeniedHandler!() })
    await flushEffects()
    expect(container.textContent).toContain(HINT_TEXT_FRAGMENT)

    // ...and a subsequent grant (re-auth after fixing System Settings) clears it.
    await act(async () => { capturedPermissionGrantedHandler!() })
    await flushEffects()
    expect(container.textContent).not.toContain(HINT_TEXT_FRAGMENT)
  })

  it('clears a stale hint when the toggle is turned back on', async () => {
    // Toggle starts OFF; the hint is showing from an earlier denial.
    vi.mocked(AppBindings.GetNotifyOnWaiting).mockResolvedValue(false)
    const props = makeProps()
    ;({ container, root } = renderSettingsTab(props))
    await flushEffects()

    await act(async () => { capturedPermissionDeniedHandler!() })
    await flushEffects()
    expect(container.textContent).toContain(HINT_TEXT_FRAGMENT)

    // Turning the toggle ON optimistically clears the stale hint before the
    // backend re-requests authorization (WR-01). The checkbox has id="notifyOnWaiting".
    const toggle = container.querySelector<HTMLInputElement>('#notifyOnWaiting')
    expect(toggle).not.toBeNull()
    await act(async () => { toggle!.click() })
    await flushEffects()

    expect(container.textContent).not.toContain(HINT_TEXT_FRAGMENT)
  })
})
