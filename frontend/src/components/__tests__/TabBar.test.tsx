import { describe, it, expect, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { TabBar } from '../TabBar'
import type { Tab } from '../TabBar'

interface TabBarProps {
  tabs: Tab[]
  activeId: string | null
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onRename: (id: string, name: string) => void
  onAdd: () => void
  onSettings: () => void
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
        onAdd: () => {},
        onSettings: () => {},
      })
    )
  })
  return { container, root }
}

function renderTabBarWithTabs(overrides: Partial<TabBarProps> = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  const props: TabBarProps = {
    tabs: [
      { id: 'tab1', name: 'claude 1', sessionId: 'sess1', cli: 'claude' },
      { id: 'tab2', name: 'codex 1', sessionId: 'sess2', cli: 'codex' },
    ],
    activeId: 'tab1',
    onSelect: () => {},
    onClose: () => {},
    onRename: () => {},
    onAdd: () => {},
    onSettings: () => {},
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

  it('renders tab-bar__controls section', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-bar__controls')).not.toBeNull()
  })

  it('renders add button with tab-bar__btn--add class', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-bar__btn--add')).not.toBeNull()
  })

  it('renders settings button with tab-bar__btn--settings class', () => {
    ;({ container, root } = renderTabBar())
    expect(container.querySelector('.tab-bar__btn--settings')).not.toBeNull()
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
