/**
 * Phase 168-05 (UX-02, #115) — StatusBar footer "Share Session" button.
 *
 * D-13: label is "Share Session" in both the OFF and ON web-share states (no more
 *       "Enable Web"/"Disable Web" two-branch toggle text).
 * D-14: clicking the button always calls onShareSession (→ App.tsx's
 *       openShareModalForActiveSession, which opens the lifted Share modal) — it
 *       NEVER calls ToggleWebServing directly. The direct-toggle behavior tested by
 *       the old StatusBar.test.tsx ("Enable Web"/"Disable Web" + onToggleWeb) is
 *       retired; this file supersedes it (see StatusBar.tsx doc comment).
 * D-15 (SECURITY, STRIDE-EoP): the button/StatusBar must never render on a
 *       non-shareable surface (Hub, Settings, Help, Welcome, web-session,
 *       file-browser tabs). StatusBar itself has no tab-type awareness — the gate
 *       lives in App.tsx's tab.map() filter, which is asserted here via source
 *       inspection (App.tsx is not fully mounted in this codebase's established
 *       test convention — see App.shellWebShare.test.tsx / App.createTab.stayOnHub.test.tsx).
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { StatusBar, StatusBarProps } from '../StatusBar'
import statusBarRaw from '../StatusBar.tsx?raw'
import raw from '../../App.tsx?raw'

function renderStatusBar(props: Partial<StatusBarProps> = {}) {
  const defaults: StatusBarProps = {
    sessionId: 'test-session',
    webServerRunning: false,
    webEnabled: false,
    onShareSession: vi.fn(),
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

describe('StatusBar — "Share Session" button (Phase 168-05 UX-02)', () => {
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

  it('shows .tab-status-bar__state--inactive with text "WEB SERVER NOT RUNNING" when webServerRunning=false, no button', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: false }))
    const badge = container.querySelector('.tab-status-bar__state--inactive')
    expect(badge).not.toBeNull()
    expect(badge?.textContent).toBe('WEB SERVER NOT RUNNING')
    expect(container.querySelector('button')).toBeNull()
  })

  it('shows "Share Session" button (not "Enable Web") when webServerRunning=true, webEnabled=false', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: false }))
    const badge = container.querySelector('.tab-status-bar__state--off')
    expect(badge?.textContent).toBe('WEB OFF')
    const buttons = container.querySelectorAll('button')
    const texts = Array.from(buttons).map((b) => b.textContent)
    expect(texts).toContain('Share Session')
    expect(texts).not.toContain('Enable Web')
  })

  it('shows "Share Session" button (not "Disable Web") when webServerRunning=true, webEnabled=true', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: true }))
    const badge = container.querySelector('.tab-status-bar__state--on')
    expect(badge?.textContent).toBe('WEB ON')
    const buttons = container.querySelectorAll('button')
    const texts = Array.from(buttons).map((b) => b.textContent)
    expect(texts).toContain('Share Session')
    expect(texts).not.toContain('Disable Web')
  })

  it('renders exactly one "Share Session" button in both OFF and ON states (single label, no two-branch toggle)', () => {
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: false }))
    let buttons = Array.from(container.querySelectorAll('button')).filter((b) => b.textContent === 'Share Session')
    expect(buttons.length).toBe(1)
    root.unmount()
    ;({ container, root } = renderStatusBar({ webServerRunning: true, webEnabled: true }))
    buttons = Array.from(container.querySelectorAll('button')).filter((b) => b.textContent === 'Share Session')
    expect(buttons.length).toBe(1)
  })

  it('calls onShareSession (not onToggleWeb) when the button is clicked (webEnabled=false)', () => {
    const onShareSession = vi.fn()
    ;({ container, root } = renderStatusBar({
      webServerRunning: true,
      webEnabled: false,
      onShareSession,
    }))
    const btn = Array.from(container.querySelectorAll('button')).find((b) => b.textContent === 'Share Session')
    expect(btn).not.toBeUndefined()
    btn!.click()
    expect(onShareSession).toHaveBeenCalledTimes(1)
  })

  it('calls onShareSession (not onToggleWeb) when the button is clicked (webEnabled=true)', () => {
    const onShareSession = vi.fn()
    ;({ container, root } = renderStatusBar({
      webServerRunning: true,
      webEnabled: true,
      onShareSession,
    }))
    const btn = Array.from(container.querySelectorAll('button')).find((b) => b.textContent === 'Share Session')
    expect(btn).not.toBeUndefined()
    btn!.click()
    expect(onShareSession).toHaveBeenCalledTimes(1)
  })

})

describe('StatusBar.tsx source — direct-toggle path fully removed (D-14)', () => {
  it('StatusBar.tsx does not reference onToggleWeb or ToggleWebServing', () => {
    expect(statusBarRaw).not.toContain('onToggleWeb')
    expect(statusBarRaw).not.toContain('ToggleWebServing')
  })

  it('StatusBarProps declares onShareSession', () => {
    expect(statusBarRaw).toContain('onShareSession')
  })
})

describe('App.tsx wiring — footer "Share Session" opens the lifted Share modal (D-14)', () => {
  it('StatusBar render site passes onShareSession={openShareModalForActiveSession}, not onToggleWeb', () => {
    const idx = raw.indexOf('<StatusBar')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 300)
    expect(block).toContain('onShareSession={openShareModalForActiveSession}')
    expect(block).not.toContain('onToggleWeb')
  })

  it('declares openShareModalForActiveSession, deriving the session from hubSessions + activeId', () => {
    expect(raw).toContain('openShareModalForActiveSession')
    const idx = raw.indexOf('const openShareModalForActiveSession')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 400)
    expect(block).toContain('hubSessions.find(')
    expect(block).toContain('activeId')
    expect(block).toContain('setShareModalSession(')
  })

  it('shareModalSession state is lifted to App.tsx (not local to HubPanel)', () => {
    expect(raw).toContain('const [shareModalSession, setShareModalSession] = useState')
  })

  it('renders exactly one <SessionShareModal> instance', () => {
    const matches = raw.match(/<SessionShareModal/g) ?? []
    expect(matches.length).toBe(1)
  })

  it('the SessionShareModal instance is gated on shareModalSession and receives session={shareModalSession}', () => {
    const idx = raw.indexOf('<SessionShareModal')
    expect(idx).toBeGreaterThan(-1)
    const before = raw.slice(Math.max(0, idx - 40), idx)
    expect(before).toContain('shareModalSession &&')
    const block = raw.slice(idx, idx + 400)
    expect(block).toContain('session={shareModalSession}')
  })

  // Phase 168-08 (gap): the modal's own toggle must notify App so webEnabled —
  // and therefore the footer pill — tracks the modal toggle for every path
  // (footer-opened + Hub-card-opened, warned + un-warned). App.tsx is not
  // fully mountable in this codebase's test convention (see file header), so
  // this is a source-inspection proof that the wiring exists at the single
  // <SessionShareModal> render site, matching the D3 rationale in
  // 168-05-SUMMARY.md.
  it('wires onShareEnabledChange on the <SessionShareModal> render to setWebEnabled, closing the UX-02 / #115 footer pill drift', () => {
    const idx = raw.indexOf('<SessionShareModal')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 600)
    expect(block).toContain('onShareEnabledChange')
    expect(block).toContain('setWebEnabled')
  })
})

describe('App.tsx tab-type gate — footer Share affordance hidden on non-shareable tabs (D-15, T-168-06)', () => {
  it('excludes welcome/settings/file-browser/hub/help/web-session tab types from the StatusBar-bearing wrapper', () => {
    const idx = raw.indexOf("tabs.map((tab) => {")
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 300)
    // Gate must check tab TYPE, not just sessionId presence (STRIDE-EoP).
    for (const t of ['welcome', 'settings', 'file-browser', 'hub', 'help', 'web-session']) {
      expect(block).toContain(`tab.type === '${t}'`)
    }
  })

  it('StatusBar is rendered inside the same type-gated tab wrapper (structural: after the filter, before the closing map)', () => {
    const mapIdx = raw.indexOf('tabs.map((tab) => {')
    const nextMapCloseIdx = raw.indexOf('})}', mapIdx)
    expect(mapIdx).toBeGreaterThan(-1)
    expect(nextMapCloseIdx).toBeGreaterThan(mapIdx)
    const block = raw.slice(mapIdx, nextMapCloseIdx)
    expect(block).toContain('<StatusBar')
  })
})
