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
  GetCACertPath: vi.fn().mockResolvedValue(''),
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

describe('SettingsPanel', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders two tab buttons with text "CLI Paths" and "Web Serving"', () => {
    ;({ container, root } = renderSettingsPanel())
    const buttons = container.querySelectorAll('button')
    const buttonTexts = Array.from(buttons).map((b) => b.textContent?.trim())
    expect(buttonTexts).toContain('CLI Paths')
    expect(buttonTexts).toContain('Web Serving')
  })

  it('CLI Paths tab button has class settings-panel__tab-btn--active on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const buttons = container.querySelectorAll('button')
    const cliPathsBtn = Array.from(buttons).find((b) => b.textContent?.trim() === 'CLI Paths')
    expect(cliPathsBtn).not.toBeUndefined()
    expect(cliPathsBtn?.classList.contains('settings-panel__tab-btn--active')).toBe(true)
  })

  it('CLI Paths content (h3 with "CLI Paths") is visible on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const headings = container.querySelectorAll('h3')
    const cliHeading = Array.from(headings).find((h) => h.textContent?.trim() === 'CLI Paths')
    expect(cliHeading).not.toBeUndefined()
  })

  it('Web Serving content (h3 with "Web Serving") is NOT in the DOM on initial render', () => {
    ;({ container, root } = renderSettingsPanel())
    const headings = container.querySelectorAll('h3')
    const webHeading = Array.from(headings).find((h) => h.textContent?.trim() === 'Web Serving')
    expect(webHeading).toBeUndefined()
  })

  it('clicking Web Serving tab shows Web Serving h3 and hides CLI Paths h3', () => {
    ;({ container, root } = renderSettingsPanel())
    const buttons = container.querySelectorAll('button')
    const webServingBtn = Array.from(buttons).find((b) => b.textContent?.trim() === 'Web Serving')
    expect(webServingBtn).not.toBeUndefined()
    flushSync(() => {
      webServingBtn!.click()
    })
    const headings = container.querySelectorAll('h3')
    const webHeading = Array.from(headings).find((h) => h.textContent?.trim() === 'Web Serving')
    const cliHeading = Array.from(headings).find((h) => h.textContent?.trim() === 'CLI Paths')
    expect(webHeading).not.toBeUndefined()
    expect(cliHeading).toBeUndefined()
  })

  it('footer contains exactly one button with text "Close" (not "Cancel", not "Save")', () => {
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
    expect(closeBtn).not.toBeUndefined()
    expect(closeBtn?.classList.contains('settings-panel__btn--cancel')).toBe(true)
  })

  it('CLI Paths tab contains a "Save Paths" button (inline, not in footer)', () => {
    ;({ container, root } = renderSettingsPanel())
    const footer = container.querySelector('.settings-panel__footer')
    const body = container.querySelector('.settings-panel__body')
    // Save Paths should NOT be in the footer
    const footerButtons = footer!.querySelectorAll('button')
    const footerBtnTexts = Array.from(footerButtons).map((b) => b.textContent?.trim())
    expect(footerBtnTexts).not.toContain('Save Paths')
    // Save Paths SHOULD be in the body
    const bodyButtons = body!.querySelectorAll('button')
    const savePathsBtn = Array.from(bodyButtons).find((b) => b.textContent?.trim() === 'Save Paths')
    expect(savePathsBtn).not.toBeUndefined()
  })
})
