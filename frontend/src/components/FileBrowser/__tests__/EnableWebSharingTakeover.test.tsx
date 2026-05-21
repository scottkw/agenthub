/**
 * @vitest-environment jsdom
 *
 * Phase 122-03 Task 2 — EnableWebSharingTakeover tests.
 *
 * Pins the locked D-04 copy verbatim ("Remote session must be web-shared to
 * browse files. Ask the owner to enable sharing.") and the Re-enter join code
 * recovery affordance.
 */

import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { EnableWebSharingTakeover } from '../EnableWebSharingTakeover'

function render(props: { onReenterJoinCode?: () => void } = {}) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  flushSync(() => {
    root.render(
      React.createElement(EnableWebSharingTakeover, {
        onReenterJoinCode: props.onReenterJoinCode ?? (() => {}),
      }),
    )
  })
  return { container, root }
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('EnableWebSharingTakeover', () => {
  it('renders the locked D-04 copy verbatim', () => {
    const { container } = render()
    expect(container.textContent).toContain(
      'Remote session must be web-shared to browse files. Ask the owner to enable sharing.',
    )
  })

  it('has an alert-equivalent live region for screen readers', () => {
    const { container } = render()
    // The element should be either role=alert or role=region with aria-label —
    // both are accessible patterns; lock the minimum: a labelled region.
    const root = container.firstElementChild as HTMLElement
    const role = root.getAttribute('role')
    expect(role === 'alert' || role === 'region').toBe(true)
  })

  it('has a Re-enter join code button that calls the callback', () => {
    const onReenter = vi.fn()
    const { container } = render({ onReenterJoinCode: onReenter })
    const buttons = Array.from(container.querySelectorAll('button'))
    const reenter = buttons.find((b) => /re-enter join code/i.test(b.textContent || ''))
    expect(reenter).toBeTruthy()
    ;(reenter as HTMLButtonElement).click()
    expect(onReenter).toHaveBeenCalled()
  })

  it('has a recognisable takeover heading (Web sharing required)', () => {
    const { container } = render()
    expect(container.textContent).toMatch(/Web sharing required/i)
  })
})
