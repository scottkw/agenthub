import { describe, it, expect, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { TabBar } from '../TabBar'

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
