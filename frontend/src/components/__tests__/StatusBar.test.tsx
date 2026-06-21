import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { StatusBar, StatusBarProps } from '../StatusBar'

function renderStatusBar(props: Partial<StatusBarProps> = {}) {
  const defaults: StatusBarProps = {
    sessionId: 'test-session',
    webServerRunning: false,
    webEnabled: false,
    onToggleWeb: vi.fn(),
  }
  const merged = { ...defaults, ...props }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(StatusBar, merged))
  })
  return { container, root }
}

describe('StatusBar', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders .tab-status-bar root element', () => {
    ;({ container, root } = renderStatusBar())
    expect(container.querySelector('.tab-status-bar')).not.toBeNull()
  })

  it('shows .tab-status-bar__state--inactive with text "WEB SERVER NOT RUNNING" when webServerRunning=false', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: false }))
    const badge = container.querySelector('.tab-status-bar__state--inactive')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('WEB SERVER NOT RUNNING')
  })

  it('shows .tab-status-bar__state--off with text "WEB OFF" when webServerRunning=true, webEnabled=false', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: false }))
    const badge = container.querySelector('.tab-status-bar__state--off')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('WEB OFF')
  })

  it('shows "Enable Web" button when webServerRunning=true, webEnabled=false', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: false }))
    const buttons = container.querySelectorAll('button')
    const enableBtn = Array.from(buttons).find((b) => b.textContent === 'Enable Web')
    expect(enableBtn).not.toBeUndefined()
  })

  it('shows .tab-status-bar__state--on with text "WEB ON" when webServerRunning=true, webEnabled=true', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: true }))
    const badge = container.querySelector('.tab-status-bar__state--on')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('WEB ON')
  })

  it('shows hint pointing to Hub card when web enabled (Phase 87 cleanup)', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: true }))
    const hint = container.querySelector('.tab-status-bar__hint')
    expect(hint).not.toBeNull()
    expect(hint?.textContent).toBe('Share — open the Hub card')
  })

  it('does NOT render a raw session URL or QR button when web enabled (Phase 87 cleanup)', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: true }))
    expect(container.querySelector('.tab-status-bar__url')).toBeNull()
    const buttons = container.querySelectorAll('button')
    const buttonTexts = Array.from(buttons).map((b) => b.textContent)
    expect(buttonTexts).not.toContain('QR')
    expect(buttonTexts).not.toContain('Copy Link')
    expect(buttonTexts).toContain('Disable Web')
  })

  it('calls onToggleWeb when Enable Web button clicked', () => {
    const onToggleWeb = vi.fn()
    ;({ container, root } = renderStatusBar({
      webServerRunning: true,
      webEnabled: false,
      onToggleWeb,
    }))
    const buttons = container.querySelectorAll('button')
    const enableBtn = Array.from(buttons).find((b) => b.textContent === 'Enable Web')
    expect(enableBtn).not.toBeUndefined()
    enableBtn!.click()
    expect(onToggleWeb).toHaveBeenCalledTimes(1)
  })

  it('calls onToggleWeb when Disable Web button clicked', () => {
    const onToggleWeb = vi.fn()
    ;({ container, root } = renderStatusBar({
      webServerRunning: true,
      webEnabled: true,
      onToggleWeb,
    }))
    const buttons = container.querySelectorAll('button')
    const disableBtn = Array.from(buttons).find((b) => b.textContent === 'Disable Web')
    expect(disableBtn).not.toBeUndefined()
    disableBtn!.click()
    expect(onToggleWeb).toHaveBeenCalledTimes(1)
  })
})
