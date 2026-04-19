import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { ExitToast } from '../ExitToast'
import type { ExitState } from '../ExitToast'

function makeExitState(overrides: Partial<ExitState> = {}): ExitState {
  return {
    sessionId: 'sess-1',
    sessionName: 'claude 1',
    cli: 'claude',
    exitCode: 0,
    duration: 120,
    finalStatus: 'idle',
    countdown: 5,
    cancelled: false,
    ...overrides,
  }
}

function renderToast(
  exits: Record<string, ExitState>,
  onKeepOpen: (id: string) => void = vi.fn(),
  onDismiss: (id: string) => void = vi.fn(),
) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(ExitToast, { exits, onKeepOpen, onDismiss }))
  })
  return { container, root }
}

describe('ExitToast', () => {
  let container: HTMLElement
  let root: ReturnType<typeof createRoot>

  afterEach(() => {
    root?.unmount()
    container?.remove()
    vi.clearAllMocks()
  })

  it('returns null when exits is empty', () => {
    ;({ container, root } = renderToast({}))
    expect(container.querySelector('.exit-toast')).toBeNull()
  })

  it('renders clean exit toast with green border class', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 0 }) }
    ;({ container, root } = renderToast(exits))
    const item = container.querySelector('.exit-toast__item--clean')
    expect(item).not.toBeNull()
  })

  it('renders error exit toast with red border class', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 1 }) }
    ;({ container, root } = renderToast(exits))
    const item = container.querySelector('.exit-toast__item--error')
    expect(item).not.toBeNull()
  })

  it('displays session name in toast header', () => {
    const exits = { 'sess-1': makeExitState({ sessionName: 'my-session' }) }
    ;({ container, root } = renderToast(exits))
    expect(container.textContent).toContain('my-session')
  })

  it('displays CLI, exit status, and final status in meta line', () => {
    const exits = { 'sess-1': makeExitState({ cli: 'opencode', finalStatus: 'idle' }) }
    ;({ container, root } = renderToast(exits))
    const meta = container.querySelector('.exit-toast__meta')
    expect(meta?.textContent).toContain('opencode')
    expect(meta?.textContent).toContain('exited')
    expect(meta?.textContent).toContain('idle')
  })

  it('displays exit code and duration', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 0, duration: 300 }) }
    ;({ container, root } = renderToast(exits))
    expect(container.textContent).toContain('Exit code: 0')
    expect(container.textContent).toContain('Duration: 300s')
  })

  it('shows countdown and Keep Open button for clean exit with active countdown', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 0, countdown: 3, cancelled: false }) }
    ;({ container, root } = renderToast(exits))
    expect(container.textContent).toContain('Closing in 3s')
    expect(container.querySelector('.exit-toast__keep-open')).not.toBeNull()
  })

  it('hides countdown for error exits', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 1, countdown: -1 }) }
    ;({ container, root } = renderToast(exits))
    expect(container.querySelector('.exit-toast__actions')).toBeNull()
  })

  it('hides countdown when cancelled', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 0, countdown: 3, cancelled: true }) }
    ;({ container, root } = renderToast(exits))
    expect(container.querySelector('.exit-toast__actions')).toBeNull()
  })

  it('calls onKeepOpen when Keep Open button clicked', () => {
    const onKeepOpen = vi.fn()
    const exits = { 'sess-1': makeExitState({ countdown: 3 }) }
    ;({ container, root } = renderToast(exits, onKeepOpen))
    const btn = container.querySelector('.exit-toast__keep-open') as HTMLButtonElement
    flushSync(() => { btn.click() })
    expect(onKeepOpen).toHaveBeenCalledWith('sess-1')
  })

  it('calls onDismiss when dismiss button clicked', () => {
    const onDismiss = vi.fn()
    const exits = { 'sess-1': makeExitState() }
    ;({ container, root } = renderToast(exits, vi.fn(), onDismiss))
    const btn = container.querySelector('.exit-toast__dismiss') as HTMLButtonElement
    flushSync(() => { btn.click() })
    expect(onDismiss).toHaveBeenCalledWith('sess-1')
  })

  it('renders multiple toast items', () => {
    const exits = {
      'sess-1': makeExitState({ sessionId: 'sess-1', sessionName: 'session A' }),
      'sess-2': makeExitState({ sessionId: 'sess-2', sessionName: 'session B', exitCode: 1 }),
    }
    ;({ container, root } = renderToast(exits))
    const items = container.querySelectorAll('.exit-toast__item')
    expect(items.length).toBe(2)
  })

  it('has correct aria-label on dismiss button', () => {
    const exits = { 'sess-1': makeExitState({ sessionName: 'test-sess' }) }
    ;({ container, root } = renderToast(exits))
    const btn = container.querySelector('.exit-toast__dismiss')
    expect(btn?.getAttribute('aria-label')).toBe('Dismiss exit notification for test-sess')
  })

  it('shows "exited with error" for non-zero exit code meta line', () => {
    const exits = { 'sess-1': makeExitState({ exitCode: 2 }) }
    ;({ container, root } = renderToast(exits))
    const meta = container.querySelector('.exit-toast__meta')
    expect(meta?.textContent).toContain('exited with error')
  })
})
