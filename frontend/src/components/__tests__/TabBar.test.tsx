import { describe, it, expect, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { TabBar } from '../TabBar'
import type { Tab } from '../TabBar'
import raw from '../TabBar.tsx?raw'

interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onRename: (id: string, name: string) => void
  sessionStatuses?: Record<string, string>
}

function renderTabBar() {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(TabBar, {
        tabs: [],
        activeId: null,
        onSelect: () => {},
        onClose: () => {},
        onRename: () => {},
      })
    )
  })
  return { container, root }
}

function renderTabBarWithTabs(overrides: Partial<TabBarProps> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const props = {
    tabs: [
      { id: 'tab1', name: 'claude 1', sessionId: 'sess1', cli: 'claude' },
      { id: 'tab2', name: 'codex 1', sessionId: 'sess2', cli: 'codex' },
    ],
    activeId: 'tab1',
    onSelect: () => {},
    onClose: () => {},
    onRename: () => {},
    ...overrides,
  }
  flushSync(() => {
    root.render(React.createElement(TabBar, props))
  })
  return { container, root }
}

describe('TabBar', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders tab-bar root element', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-bar')).not.toBeNull()
  })

  it('renders tab-list container', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-list')).not.toBeNull()
  })
})


describe('TabBar context menu', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
    // Clean up any leftover event listeners by dispatching a mousedown to body
  })

  it('right-clicking tab name shows context menu', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })
    expect(container.querySelector('.tab__context-menu')).not.toBeNull()
  })

  it('context menu has role="menu" and contains Rename menuitem', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })
    const menu = container.querySelector('.tab__context-menu') as HTMLElement
    expect(menu).not.toBeNull()
    expect(menu.getAttribute('role')).toBe('menu')
    const renameBtn = menu.querySelector('button[role="menuitem"]') as HTMLButtonElement
    expect(renameBtn).not.toBeNull()
    expect(renameBtn.textContent).toBe('Rename')
  })

  it('clicking Rename starts inline editing', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    flushSync(() => {
      tabName.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true }))
    })
    const renameBtn = container.querySelector('button[role="menuitem"]') as HTMLButtonElement
    expect(renameBtn).not.toBeNull()
    flushSync(() => {
      renameBtn.click()
    })
    expect(container.querySelector('.tab__rename-input')).not.toBeNull()
  })

  it('tab name title includes right-click', () => {
    ;({ container, root } = renderTabBarWithTabs())
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    expect(tabName.getAttribute('title')).toContain('right-click')
  })
})

describe('Phase 97 SER-01: TabBar Save Terminal As… menu item — Plan 97-04 implementation', () => {
  it('Save Terminal As menu-item label appears in TabBar source (with U+2026 ellipsis)', () => {
    // The literal string must contain U+2026 HORIZONTAL ELLIPSIS, not "..."
    expect(raw).toContain('Save Terminal As…')
  })

  it('TabBar.tsx props interface declares onRequestSave?: (tabId: string) => void', () => {
    expect(raw).toMatch(/onRequestSave\?:\s*\(\s*tabId:\s*string\s*\)\s*=>\s*void/)
  })

  it('Save menu-item onClick invokes onRequestSave?.(contextMenu.tabId) and closes the menu', () => {
    // Source must contain a click handler that calls both onRequestSave and setContextMenu(null).
    expect(raw).toMatch(/onRequestSave\?\.\(\s*contextMenu\.tabId\s*\)/)
    // The "Save Terminal As…" button must be near a setContextMenu(null) call.
    const saveIdx = raw.indexOf('Save Terminal As…')
    expect(saveIdx).toBeGreaterThan(-1)
    // Walk back ~300 chars from the button label and look for setContextMenu(null) — the
    // entire button JSX (handler + label) fits in this window.
    const window = raw.slice(Math.max(0, saveIdx - 300), saveIdx + 100)
    expect(window).toMatch(/setContextMenu\(\s*null\s*\)/)
  })
})

// Phase 101-02: TabBar agent badge (SHELL-06 GUI half).
// Render-based tests using createRoot + flushSync, mirroring the existing
// renderTabBarWithTabs helper.

function renderTabBarWithCLI(cli: string, tabName = 'session-1') {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(TabBar, {
        tabs: [{ id: 'tab1', name: tabName, sessionId: 'sess1', cli }],
        activeId: 'tab1',
        onSelect: () => {},
        onClose: () => {},
        onRename: () => {},
      })
    )
  })
  return { container, root }
}

describe('Phase 101-02 TabBar agent badge (SHELL-06 GUI)', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root.unmount()
    container.remove()
  })

  it('renders agent badge for claude session', () => {
    ;({ container, root } = renderTabBarWithCLI('claude'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge).not.toBeNull()
    expect(badge!.className).toContain('claude')
  })

  it('renders shell agent badge for cli=shell', () => {
    ;({ container, root } = renderTabBarWithCLI('shell'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge).not.toBeNull()
    expect(badge!.className).toContain('--shell')
  })

  it('renders shell agent badge for cli=bash', () => {
    ;({ container, root } = renderTabBarWithCLI('bash'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge).not.toBeNull()
    expect(badge!.className).toContain('--shell')
    expect(badge!.className).not.toContain('--bash')
  })

  it('renders shell agent badge for cli=zsh', () => {
    ;({ container, root } = renderTabBarWithCLI('zsh'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge!.className).toContain('--shell')
  })

  it('renders shell agent badge for cli=pwsh', () => {
    ;({ container, root } = renderTabBarWithCLI('pwsh'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge!.className).toContain('--shell')
  })

  it('renders shell agent badge for cli=powershell', () => {
    ;({ container, root } = renderTabBarWithCLI('powershell'))
    const badge = container.querySelector('.tab__agent-badge')
    expect(badge!.className).toContain('--shell')
  })

  it('falls back to muted badge for unknown cli', () => {
    ;({ container, root } = renderTabBarWithCLI('unknown-tool'))
    const badge = container.querySelector('.tab__agent-badge') as HTMLElement
    expect(badge).not.toBeNull()
    const knownModifiers = ['--claude', '--opencode', '--codex', '--gemini', '--cursor', '--aider', '--shell']
    for (const mod of knownModifiers) {
      expect(badge.className).not.toContain(mod)
    }
  })

  it('agent badge is aria-hidden', () => {
    ;({ container, root } = renderTabBarWithCLI('claude'))
    const badge = container.querySelector('.tab__agent-badge') as HTMLElement
    expect(badge.getAttribute('aria-hidden')).toBe('true')
  })

  it('tab tooltip includes agent type for shell session', () => {
    ;({ container, root } = renderTabBarWithCLI('bash', 'scratch-bash'))
    const tabName = container.querySelector('.tab__name') as HTMLElement
    expect(tabName).not.toBeNull()
    const title = tabName.getAttribute('title') || ''
    expect(title).toContain('Shell — bash')
  })

  it('tab agent badge appears between status dot and tab name', () => {
    ;({ container, root } = renderTabBarWithCLI('claude'))
    const tab = container.querySelector('.tab') as HTMLElement
    const children = Array.from(tab.children)
    const statusIdx = children.findIndex((c) => c.classList.contains('tab__status'))
    const badgeIdx = children.findIndex((c) => c.classList.contains('tab__agent-badge'))
    const nameIdx = children.findIndex((c) => c.classList.contains('tab__name'))
    expect(statusIdx).toBeGreaterThanOrEqual(0)
    expect(badgeIdx).toBeGreaterThan(statusIdx)
    expect(nameIdx).toBeGreaterThan(badgeIdx)
  })
})

// Phase 98 PRG-03 — Wave 0 RED scaffold.
// These two test cases are RED until Wave 3 (Plan 04) adds the .tab__progress
// underline element to TabBar.tsx. Tagged `progress-underline` and
// `progress-transform` for VALIDATION.md verify targeting.
// Using source-inspection (?raw) pattern consistent with the rest of this file.
describe('Phase 98 PRG-03: TabBar per-tab progress underline — Wave 0 RED scaffold (Plan 98-01 Wave 0)', () => {
  it('progress-underline: TabBarProps interface declares tabProgress?: Record<string, number>', () => {
    // RED until Wave 3 (Plan 04) extends the TabBarProps interface.
    expect(raw).toMatch(/tabProgress\?:\s*Record<string,\s*number>/)
  })

  it('progress-underline: renders .tab__progress element with data-testid keyed by tab.id', () => {
    // RED until Wave 3 adds the .tab__progress <div> inside the tab render.
    expect(raw).toContain('tab__progress')
    expect(raw).toMatch(/data-testid[^}]*tab-progress-/)
  })

  it('progress-transform: uses transform scaleX for the underline animation (not width)', () => {
    // RED until Wave 3. The production element must set transform: scaleX(...)
    // and must NOT set inline width (CSS width stays 100%; only transform changes).
    expect(raw).toMatch(/scaleX\(/)
  })
})
