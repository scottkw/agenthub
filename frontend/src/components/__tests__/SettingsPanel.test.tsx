import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SettingsTab } from '../SettingsTab'
import rawSettings from '../SettingsTab.tsx?raw'

vi.mock('../../wailsjs/go/main/App', () => ({
  UpdateCLIPath: vi.fn(),
  StartWebServer: vi.fn(),
  StopWebServer: vi.fn(),
  GetWebServerURL: vi.fn().mockResolvedValue(''),
  IsWebServerRunning: vi.fn().mockResolvedValue(false),
  HasCTDisclosure: vi.fn().mockResolvedValue(false),
  AcknowledgeCTDisclosure: vi.fn(),
}))

interface SettingsTabProps {
  clis: Array<{ Name: string; Path: string }>
  tailscaleHealth: {
    installed: boolean
    connected: boolean
    hasCerts: boolean
    ip: string
    domain: string
  } | null
  onWebServerStateChange: () => Promise<void>
}

function renderSettingsTab(props: Partial<SettingsTabProps> = {}) {
  const defaults: SettingsTabProps = {
    clis: [{ Name: 'claude', Path: '/usr/bin/claude' }],
    tailscaleHealth: null,
    onWebServerStateChange: vi.fn().mockResolvedValue(undefined),
  }
  const merged = { ...defaults, ...props }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(SettingsTab, merged as any))
  })
  return { container, root }
}

function clickTabByText(container: HTMLElement, text: string) {
  const buttons = container.querySelectorAll('.settings-panel__tab-btn')
  const btn = Array.from(buttons).find((b) => b.textContent?.trim() === text) as HTMLElement | undefined
  expect(btn).not.toBeUndefined()
  flushSync(() => {
    btn!.click()
  })
}

describe('SettingsTab', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders exactly two tab buttons: "CLI Paths" and "Web Server"', () => {
    ;({ container, root } = renderSettingsTab())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    expect(tabs.length).toBe(2)
    const tabTexts = Array.from(tabs).map((t) => t.textContent?.trim())
    expect(tabTexts).toEqual(['CLI Paths', 'Web Server'])
  })

  it('CLI Paths tab button has active class on initial render', () => {
    ;({ container, root } = renderSettingsTab())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    const cliTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'CLI Paths')
    expect(cliTab?.classList.contains('settings-panel__tab-btn--active')).toBe(true)
    expect(cliTab?.getAttribute('aria-selected')).toBe('true')
  })

  it('CLI Paths content is visible on initial render', () => {
    ;({ container, root } = renderSettingsTab())
    const table = container.querySelector('.settings-panel__table')
    expect(table).not.toBeNull()
  })

  it('Web Server content is NOT in the DOM on initial render', () => {
    ;({ container, root } = renderSettingsTab())
    const selects = container.querySelectorAll('.settings-panel__select')
    expect(selects.length).toBe(0)
  })

  it('Security tab does not exist', () => {
    ;({ container, root } = renderSettingsTab())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    const securityTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'Security')
    expect(securityTab).toBeUndefined()
  })

  it('no password input rendered (Security tab removed)', () => {
    ;({ container, root } = renderSettingsTab())
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(0)
  })

  it('clicking Web Server tab shows network interface and port, hides CLI table', () => {
    ;({ container, root } = renderSettingsTab())
    clickTabByText(container, 'Web Server')
    // Web Server content present
    const description = container.querySelector('.settings-panel__description')
    expect(description?.textContent).toContain('HTTPS access')
    // CLI Paths content gone
    const table = container.querySelector('.settings-panel__table')
    expect(table).toBeNull()
    // Password not present (Security tab removed)
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(0)
    // CA certificate section not present (Security tab removed)
    const certLabels = container.querySelectorAll('.settings-panel__label')
    const caCertLabel = Array.from(certLabels).find((l) => l.textContent?.includes('CA Certificate'))
    expect(caCertLabel).toBeUndefined()
  })

  it('Start Web Server button is disabled when CT not disclosed', () => {
    ;({ container, root } = renderSettingsTab())
    clickTabByText(container, 'Web Server')
    const buttons = container.querySelectorAll('button')
    const startBtn = Array.from(buttons).find((b) => b.textContent?.includes('Start Web Server'))
    expect(startBtn).not.toBeUndefined()
    expect((startBtn as HTMLButtonElement).disabled).toBe(true)
  })

  it('has no modal footer (tab renders inline, no close button)', () => {
    ;({ container, root } = renderSettingsTab())
    const footer = container.querySelector('.settings-panel__footer')
    expect(footer).toBeNull()
  })

  it('has no modal overlay (tab renders inline, no settings-overlay)', () => {
    ;({ container, root } = renderSettingsTab())
    const overlay = container.querySelector('.settings-overlay')
    expect(overlay).toBeNull()
  })

  it('outer wrapper has class settings-tab', () => {
    ;({ container, root } = renderSettingsTab())
    const tab = container.querySelector('.settings-tab')
    expect(tab).not.toBeNull()
  })

  it('CLI Paths tab contains a "Save Paths" button inline', () => {
    ;({ container, root } = renderSettingsTab())
    const body = container.querySelector('.settings-panel__body')
    const savePathsBtn = Array.from(body!.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Save Paths')
    expect(savePathsBtn).not.toBeUndefined()
  })

  describe('Tailscale Status Indicator', () => {
    it('SettingsTab accepts tailscaleHealth prop', () => {
      expect(rawSettings).toContain('tailscaleHealth')
    })

    it('Web Server tab contains Tailscale Status label', () => {
      expect(rawSettings).toContain('Tailscale Status')
    })

    it('renders ts-status CSS class', () => {
      expect(rawSettings).toContain('ts-status')
    })

    it('renders ts-status__dot with status class', () => {
      expect(rawSettings).toContain('ts-status__dot')
    })

    it('shows "Connected" text for healthy state', () => {
      ;({ container, root } = renderSettingsTab({
        tailscaleHealth: { installed: true, connected: true, hasCerts: true, ip: '100.64.0.1', domain: 'host.ts.net' },
      }))
      clickTabByText(container, 'Web Server')
      const statusText = container.querySelector('.ts-status__text')
      expect(statusText?.textContent).toBe('Connected')
    })

    it('shows "Checking..." when tailscaleHealth is null', () => {
      ;({ container, root } = renderSettingsTab({ tailscaleHealth: null }))
      clickTabByText(container, 'Web Server')
      const statusText = container.querySelector('.ts-status__text')
      expect(statusText?.textContent).toBe('Checking\u2026')
    })
  })
})
