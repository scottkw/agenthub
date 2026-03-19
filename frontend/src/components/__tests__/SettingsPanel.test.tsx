import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { SettingsPanel } from '../SettingsPanel'

vi.mock('../../wailsjs/go/main/App', () => ({
  UpdateCLIPath: vi.fn(),
  SetWebPassword: vi.fn(),
  IsWebPasswordSet: vi.fn().mockResolvedValue(false),
  GetNetworkInterfaces: vi.fn().mockResolvedValue([]),
  StartWebServer: vi.fn(),
  StopWebServer: vi.fn(),
  GetWebServerURL: vi.fn().mockResolvedValue(''),
  GetCACertPath: vi.fn().mockResolvedValue('/fake/ca.crt'),
  IsWebServerRunning: vi.fn().mockResolvedValue(false),
}))

interface SettingsPanelProps {
  isOpen: boolean
  onClose: () => void
  clis: Array<{ Name: string; Path: string }>
}

function renderSettingsPanel(props: Partial<SettingsPanelProps> = {}) {
  const defaults: SettingsPanelProps = {
    isOpen: true,
    onClose: vi.fn(),
    clis: [{ Name: 'claude', Path: '/usr/bin/claude' }],
  }
  const merged = { ...defaults, ...props }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(SettingsPanel, merged as any))
  })
  return { container, root }
}

function clickTabByText(container: HTMLElement, text: string) {
  const buttons = container.querySelectorAll('.settings-panel__tab-btn')
  const btn = Array.from(buttons).find((b) => b.textContent?.trim() === text)
  expect(btn).not.toBeUndefined()
  flushSync(() => {
    btn!.click()
  })
}

describe('SettingsPanel', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders three tab buttons: "CLI Paths", "Web Server", "Security"', () => {
    ;({ container, root } = renderSettingsPanel())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    expect(tabs.length).toBe(3)
    const tabTexts = Array.from(tabs).map((t) => t.textContent?.trim())
    expect(tabTexts).toEqual(['CLI Paths', 'Web Server', 'Security'])
  })

  it('CLI Paths tab button has active class on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    const cliTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'CLI Paths')
    expect(cliTab?.classList.contains('settings-panel__tab-btn--active')).toBe(true)
    expect(cliTab?.getAttribute('aria-selected')).toBe('true')
  })

  it('CLI Paths content is visible on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const table = container.querySelector('.settings-panel__table')
    expect(table).not.toBeNull()
  })

  it('Web Server content is NOT in the DOM on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const selects = container.querySelectorAll('.settings-panel__select')
    expect(selects.length).toBe(0)
  })

  it('Security content is NOT in the DOM on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(0)
  })

  it('clicking Web Server tab shows network interface and port, hides CLI table', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Web Server')
    // Web Server content present
    const description = container.querySelector('.settings-panel__description')
    expect(description?.textContent).toContain('HTTPS access')
    // CLI Paths content gone
    const table = container.querySelector('.settings-panel__table')
    expect(table).toBeNull()
    // Password not present (that's Security tab)
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(0)
    // CA certificate section not present (that's Security tab)
    const certLabels = container.querySelectorAll('.settings-panel__label')
    const caCertLabel = Array.from(certLabels).find((l) => l.textContent?.includes('CA Certificate'))
    expect(caCertLabel).toBeUndefined()
  })

  it('clicking Security tab shows password field and hides Web Server and CLI content', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Security')
    // Password field present
    const passwordInputs = container.querySelectorAll('input[type="password"]')
    expect(passwordInputs.length).toBe(1)
    // CLI table gone
    const table = container.querySelector('.settings-panel__table')
    expect(table).toBeNull()
    // Network select gone
    const selects = container.querySelectorAll('.settings-panel__select')
    expect(selects.length).toBe(0)
  })

  it('Security tab button has aria-selected="true" when active', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Security')
    const tabs = container.querySelectorAll('.settings-panel__tab-btn')
    const securityTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'Security')
    expect(securityTab?.getAttribute('aria-selected')).toBe('true')
    // Other tabs should not be selected
    const cliTab = Array.from(tabs).find((t) => t.textContent?.trim() === 'CLI Paths')
    expect(cliTab?.getAttribute('aria-selected')).toBe('false')
  })

  it('footer contains exactly one button with text "Close"', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    expect(footer).not.toBeNull()
    const footerButtons = footer!.querySelectorAll('button')
    expect(footerButtons.length).toBe(1)
    expect(footerButtons[0].textContent?.trim()).toBe('Close')
  })

  it('Close button has class settings-panel__btn--cancel (secondary style)', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    const closeBtn = footer?.querySelector('button')
    expect(closeBtn?.classList.contains('settings-panel__btn--cancel')).toBe(true)
  })

  it('CLI Paths tab contains a "Save Paths" button inline, not in footer', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    const body = container.querySelector('.settings-panel__body')
    const footerBtnTexts = Array.from(footer!.querySelectorAll('button')).map((b) => b.textContent?.trim())
    expect(footerBtnTexts).not.toContain('Save Paths')
    const savePathsBtn = Array.from(body!.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Save Paths')
    expect(savePathsBtn).not.toBeUndefined()
  })

  it('Security tab shows CA certificate path', () => {
    ;({ container, root } = renderSettingsPanel())
    clickTabByText(container, 'Security')
    const codeBlocks = container.querySelectorAll('.settings-panel__code')
    const certCode = Array.from(codeBlocks).find((c) => c.textContent?.includes('ca.crt'))
    expect(certCode).not.toBeUndefined()
  })
})
