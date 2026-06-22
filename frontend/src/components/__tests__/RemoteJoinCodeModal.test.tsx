/**
 * @vitest-environment jsdom
 *
 * Phase 122-03 Task 2 — RemoteJoinCodeModal tests.
 *
 * Behaviour pinned (locked by 122-03-PLAN.md Task 2 Tests 1-9):
 *   - Role + aria attributes for screen-reader correctness.
 *   - Single text input + Submit + Cancel; Submit disabled while empty.
 *   - onExchange wired to Submit; "Joining..." pending state.
 *   - Error mapping: 'expired' / 'invalid' / 'not-found' / 'session-gone' →
 *     specific user-facing strings; anything else → raw message.
 *   - Cancel + Escape both call onClose; successful submit auto-closes.
 */

import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { act } from 'react'
import { flushSync } from 'react-dom'
import { RemoteJoinCodeModal } from '../RemoteJoinCodeModal'

type ExchangeFn = (code: string) => Promise<void>

function renderModal(
  overrides: Partial<{
    onExchange: ExchangeFn
    onClose: () => void
    remoteSession: { id: string; name: string; hostname: string }
  }> = {},
) {
  const defaults = {
    remoteSession: { id: 'sid-1', name: 'claude 1', hostname: 'hub-a' },
    onExchange: vi.fn(async () => undefined),
    onClose: vi.fn(),
  }
  const props = { ...defaults, ...overrides }
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(React.createElement(RemoteJoinCodeModal, props))
  })
  return { container, root, props }
}

afterEach(() => {
  document.body.innerHTML = ''
})

/**
 * Set the value of a controlled React input. Uses the native HTMLInputElement
 * value setter so React's synthetic onChange fires correctly under React 19.
 */
function setControlledInputValue(input: HTMLInputElement, value: string): void {
  const desc = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(input), 'value')
  desc?.set?.call(input, value)
  flushSync(() => {
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
}

describe('RemoteJoinCodeModal — a11y + structure', () => {
  it('renders a dialog with role="dialog" + aria-modal="true" + aria-labelledby', () => {
    const { container } = renderModal()
    const dialog = container.querySelector('[role="dialog"]')
    expect(dialog).toBeTruthy()
    expect(dialog!.getAttribute('aria-modal')).toBe('true')
    const labelledBy = dialog!.getAttribute('aria-labelledby')
    expect(labelledBy).toBeTruthy()
    // The id must point at a real element inside the dialog
    expect(container.querySelector(`#${labelledBy}`)).toBeTruthy()
  })

  it('has a single text input, a Submit button, and a Cancel button', () => {
    const { container } = renderModal()
    const inputs = container.querySelectorAll('input[type="text"]')
    expect(inputs.length).toBe(1)
    const buttons = Array.from(container.querySelectorAll('button'))
    const submit = buttons.find((b) => /join/i.test(b.textContent || ''))
    const cancel = buttons.find((b) => /cancel/i.test(b.textContent || ''))
    expect(submit).toBeTruthy()
    expect(cancel).toBeTruthy()
  })

  it('mentions the remote session name and hostname in the body copy', () => {
    const { container } = renderModal({
      remoteSession: { id: 'x', name: 'codex-A', hostname: 'hub-zeta' },
    })
    expect(container.textContent).toContain('codex-A')
    expect(container.textContent).toContain('hub-zeta')
  })
})

describe('RemoteJoinCodeModal — submit gating', () => {
  it('Submit button is disabled while input is empty / whitespace-only', () => {
    const { container } = renderModal()
    const buttons = Array.from(container.querySelectorAll('button'))
    const submit = buttons.find((b) => /join/i.test(b.textContent || '')) as HTMLButtonElement
    expect(submit.disabled).toBe(true)

    const input = container.querySelector('input[type="text"]') as HTMLInputElement
    setControlledInputValue(input, '   ')
    expect(submit.disabled).toBe(true)
  })

  it('Submit becomes enabled when input has non-whitespace text', () => {
    const { container } = renderModal()
    const input = container.querySelector('input[type="text"]') as HTMLInputElement
    setControlledInputValue(input, 'ABC12')
    const submit = Array.from(container.querySelectorAll('button')).find((b) =>
      /join/i.test(b.textContent || ''),
    ) as HTMLButtonElement
    expect(submit.disabled).toBe(false)
  })
})

describe('RemoteJoinCodeModal — submit flow', () => {
  it('calls onExchange(code) on Submit', async () => {
    const onExchange = vi.fn(async () => undefined)
    const { container } = renderModal({ onExchange })
    const input = container.querySelector('input[type="text"]') as HTMLInputElement
    setControlledInputValue(input, 'ABC12')
    const submit = Array.from(container.querySelectorAll('button')).find((b) =>
      /join/i.test(b.textContent || ''),
    ) as HTMLButtonElement
    await act(async () => {
      submit.click()
    })
    expect(onExchange).toHaveBeenCalledWith('ABC12')
  })

  it('Submit shows "Joining..." while the exchange promise is pending', async () => {
    let resolve: () => void = () => {}
    const onExchange = vi.fn(
      () =>
        new Promise<void>((res) => {
          resolve = res
        }),
    )
    const { container } = renderModal({ onExchange })
    const input = container.querySelector('input[type="text"]') as HTMLInputElement
    setControlledInputValue(input, 'ABCDE')
    const submit = Array.from(container.querySelectorAll('button')).find((b) =>
      /join/i.test(b.textContent || ''),
    ) as HTMLButtonElement
    await act(async () => {
      submit.click()
      // microtask flush for the setState before the await
      await Promise.resolve()
    })
    expect(submit.textContent).toMatch(/Joining/i)
    expect(submit.disabled).toBe(true)
    await act(async () => {
      resolve()
    })
  })

  it('calls onClose after a successful exchange', async () => {
    const onExchange = vi.fn(async () => undefined)
    const onClose = vi.fn()
    const { container } = renderModal({ onExchange, onClose })
    const input = container.querySelector('input[type="text"]') as HTMLInputElement
    setControlledInputValue(input, 'ABC12')
    const submit = Array.from(container.querySelectorAll('button')).find((b) =>
      /join/i.test(b.textContent || ''),
    ) as HTMLButtonElement
    await act(async () => {
      submit.click()
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(onClose).toHaveBeenCalled()
  })
})

describe('RemoteJoinCodeModal — error mapping', () => {
  async function submitCode(container: HTMLElement, code: string): Promise<void> {
    const input = container.querySelector('input[type="text"]') as HTMLInputElement
    setControlledInputValue(input, code)
    const submit = Array.from(container.querySelectorAll('button')).find((b) =>
      /join/i.test(b.textContent || ''),
    ) as HTMLButtonElement
    await act(async () => {
      submit.click()
      // give the promise rejection two microtask flushes
      await Promise.resolve()
      await Promise.resolve()
    })
  }

  it("'expired' substring → 'Code expired. Ask the owner to generate a new code.'", async () => {
    const onExchange = vi.fn(async () => {
      throw new Error('join code expired (status 408)')
    })
    const { container } = renderModal({ onExchange })
    await submitCode(container, 'ABC12')
    expect(container.textContent).toContain('Code expired')
    expect(container.textContent).toContain('Ask the owner to generate a new code.')
  })

  it("'invalid' substring → 'Code invalid. Double-check the 8-character code (XXXX-XXXX).'", async () => {
    const onExchange = vi.fn(async () => {
      throw new Error('join code is invalid (status 401)')
    })
    const { container } = renderModal({ onExchange })
    await submitCode(container, 'ABCD-EFGH')
    expect(container.textContent).toContain('Code invalid')
    expect(container.textContent).toContain('Double-check the 8-character code (XXXX-XXXX).')
  })

  it("'not-found' substring → 'Code already used or expired' (WR-03: single-use code consumed, GAP-146-A Plan 05)", async () => {
    // WR-03 fix: 'not-found' now maps to the used/expired message (single-use codes),
    // NOT 'Code invalid' (typo). A 404 from the exchange endpoint means the code was
    // already consumed by an earlier exchange (D-11 in CONTEXT.md).
    const onExchange = vi.fn(async () => {
      throw new Error('join code not-found (status 404)')
    })
    const { container } = renderModal({ onExchange })
    await submitCode(container, 'ZZZZ-ZZZZ')
    expect(container.textContent).toContain('already used or expired')
    expect(container.textContent).not.toContain('Double-check')
  })

  it("'session-gone' substring → 'Remote session is no longer web-shared.'", async () => {
    const onExchange = vi.fn(async () => {
      throw new Error('join code is session-gone (status 410)')
    })
    const { container } = renderModal({ onExchange })
    await submitCode(container, 'WWWWW')
    expect(container.textContent).toContain('Remote session is no longer web-shared')
  })

  it('falls through to raw error message on unknown error', async () => {
    const onExchange = vi.fn(async () => {
      throw new Error('transport refused unexpectedly')
    })
    const { container } = renderModal({ onExchange })
    await submitCode(container, 'AAAAA')
    expect(container.textContent).toContain('transport refused unexpectedly')
  })
})

describe('RemoteJoinCodeModal — dismissal', () => {
  it('calls onClose on Cancel button click', () => {
    const onClose = vi.fn()
    const { container } = renderModal({ onClose })
    const cancel = Array.from(container.querySelectorAll('button')).find((b) =>
      /cancel/i.test(b.textContent || ''),
    ) as HTMLButtonElement
    cancel.click()
    expect(onClose).toHaveBeenCalled()
  })

  it('calls onClose on Escape key', () => {
    const onClose = vi.fn()
    renderModal({ onClose })
    flushSync(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    })
    expect(onClose).toHaveBeenCalled()
  })
})
