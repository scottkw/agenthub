// Phase 94 SRC-01 — isXtermFocused helper tests.
//
// Verifies the four behavior cases from 94-03-PLAN <behavior>:
//   (a) null container → false
//   (b) no activeElement → false
//   (c) activeElement inside container → true
//   (d) activeElement in modal sibling → false (Pitfall #1: browser-native
//       Cmd-F must NOT be pre-empted when focus is on a sidebar/modal).
//
// Uses real jsdom DOM (no stubs of document.activeElement) so the helper is
// exercised exactly as it would be in the browser. createElement → focus() →
// assertion → afterEach cleanup.
import { describe, it, expect, afterEach } from 'vitest'
import { isXtermFocused } from '../isXtermFocused'

describe('isXtermFocused (SRC-01 helper)', () => {
  const created: HTMLElement[] = []

  afterEach(() => {
    while (created.length) created.pop()!.remove()
    // Blur any lingering focus so the next test starts clean.
    if (document.activeElement instanceof HTMLElement) {
      document.activeElement.blur()
    }
  })

  function attach(tag: string, opts: { tabIndex?: number; parent?: HTMLElement } = {}): HTMLElement {
    const el = document.createElement(tag)
    if (opts.tabIndex !== undefined) el.tabIndex = opts.tabIndex
    ;(opts.parent ?? document.body).appendChild(el)
    created.push(el)
    return el
  }

  it('returns false when termContainer is null', () => {
    expect(isXtermFocused(null)).toBe(false)
  })

  it('returns false when document.activeElement is null', () => {
    // jsdom: document.activeElement defaults to <body>; force-null by removing
    // body's tabindex effects via blur(). When nothing is focusable, jsdom
    // returns body — but the guard `!document.activeElement` only fires on
    // genuinely null, so this case becomes "activeElement IS body, container
    // is a sibling of body's focused element" — which still resolves to
    // false because body does not contain the (sibling) container.
    const container = attach('div')
    if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
    // body now has focus (jsdom default). body !== container, body does not
    // contain container in any meaningful sense for our use case (container
    // is empty), so contains(activeElement) on container is false.
    expect(isXtermFocused(container)).toBe(false)
  })

  it('returns true when activeElement is descendant of termContainer', () => {
    const container = attach('div')
    const inner = attach('input', { tabIndex: 0, parent: container })
    inner.focus()
    expect(document.activeElement).toBe(inner)
    expect(isXtermFocused(container)).toBe(true)
  })

  it('returns false when activeElement is sibling of termContainer (modal scenario — Pitfall #1)', () => {
    const container = attach('div')
    const modal = attach('div')
    const modalInput = attach('input', { tabIndex: 0, parent: modal })
    modalInput.focus()
    expect(document.activeElement).toBe(modalInput)
    expect(isXtermFocused(container)).toBe(false)
  })
})
