import { describe, it, expect, vi, afterEach } from 'vitest'
import { createRoot } from 'react-dom/client'
import { act } from 'react'

// CR-02 fix: InlineSessionName no longer calls RenameSession directly.
// The RPC is owned by App.handleRenameTab via the onRenamed callback chain.
// This mock is kept so that importing from wailsjs/go/main/App does not throw,
// but we assert RenameSession is NOT called from within this component.
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
}))

import { InlineSessionName } from './InlineSessionName'
import { RenameSession } from '../../wailsjs/go/main/App'

function renderName(props: { id: string; name: string; onRenamed?: (name: string) => void }) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(<InlineSessionName {...props} />)
  })
  return { container, root }
}

describe('InlineSessionName', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()
  })

  it('renders a span with the session name in display mode', () => {
    const { container } = renderName({ id: 'sess-1', name: 'My Session' })
    const span = container.querySelector('.hub-card__name')
    expect(span).not.toBeNull()
    expect(span!.textContent).toBe('My Session')
  })

  it('enters edit mode and shows input when name span is clicked', () => {
    const { container } = renderName({ id: 'sess-1', name: 'My Session' })
    const span = container.querySelector('.hub-card__name')!
    act(() => {
      span.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    const input = container.querySelector('.tab__rename-input') as HTMLInputElement
    expect(input).not.toBeNull()
    expect(input.value).toBe('My Session')
  })

  it('pressing Enter with changed value fires onRenamed and exits edit mode (CR-02: does NOT call RenameSession directly)', async () => {
    const onRenamed = vi.fn()
    const { container } = renderName({ id: 'sess-1', name: 'My Session', onRenamed })
    // Enter edit mode
    const span = container.querySelector('.hub-card__name')!
    act(() => {
      span.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    const input = container.querySelector('.tab__rename-input') as HTMLInputElement
    // Change value
    act(() => {
      Object.defineProperty(input, 'value', { writable: true, value: 'New Name' })
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    // React onChange: use React's synthetic onChange via fireEvent-like approach
    act(() => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')!.set!
      nativeInputValueSetter.call(input, 'New Name')
      input.dispatchEvent(new Event('input', { bubbles: true }))
      // Simulate React controlled input value change via direct manipulation
    })
    // Directly set the input value via React by re-rendering with a new value
    // Instead, simulate the keydown Enter after ensuring value is 'New Name'
    // We need to trigger onChange then Enter
    act(() => {
      // Fire a React-compatible change event
      const event = new Event('change', { bubbles: true })
      Object.defineProperty(event, 'target', { value: { value: 'New Name' } })
      input.dispatchEvent(event)
    })
    act(() => {
      const keyEvent = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })
      input.dispatchEvent(keyEvent)
    })
    // CR-02: InlineSessionName fires onRenamed — NOT RenameSession directly.
    // RenameSession is owned by App.handleRenameTab via the callback chain.
    expect(RenameSession).not.toHaveBeenCalled()
    expect(onRenamed).toHaveBeenCalledWith('New Name')
    // Should exit edit mode
    expect(container.querySelector('.tab__rename-input')).toBeNull()
  })

  it('pressing Escape restores original name and exits edit mode without calling onRenamed', () => {
    const onRenamed = vi.fn()
    const { container } = renderName({ id: 'sess-1', name: 'My Session', onRenamed })
    const span = container.querySelector('.hub-card__name')!
    act(() => {
      span.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    const input = container.querySelector('.tab__rename-input') as HTMLInputElement
    act(() => {
      const keyEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
      input.dispatchEvent(keyEvent)
    })
    expect(RenameSession).not.toHaveBeenCalled()
    expect(onRenamed).not.toHaveBeenCalled()
    // Should exit edit mode and show original name
    const nameSpan = container.querySelector('.hub-card__name')
    expect(nameSpan).not.toBeNull()
    expect(nameSpan!.textContent).toBe('My Session')
  })

  it('blur with unchanged value does NOT call onRenamed', async () => {
    const onRenamed = vi.fn()
    const { container } = renderName({ id: 'sess-1', name: 'My Session', onRenamed })
    const span = container.querySelector('.hub-card__name')!
    act(() => {
      span.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    const input = container.querySelector('.tab__rename-input') as HTMLInputElement
    // Blur without changing value
    act(() => {
      input.dispatchEvent(new FocusEvent('blur', { bubbles: true }))
    })
    expect(RenameSession).not.toHaveBeenCalled()
    expect(onRenamed).not.toHaveBeenCalled()
  })
})
