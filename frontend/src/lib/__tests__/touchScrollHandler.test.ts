import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Terminal } from '@xterm/xterm'
import { attachTouchScroll } from '../touchScrollHandler'

// Phase 113 Plan 01 — Tests for the pure attachTouchScroll function.
// Per RESEARCH Open Q3: jsdom's TouchEvent is incomplete (jsdom#1508).
// We mock the container so we can capture the registered handlers off
// `addEventListener` spy calls and invoke them directly with synthesized
// event objects of the shape:
//   { type, touches: [{ identifier, clientY }], changedTouches: [...],
//     preventDefault: vi.fn() }

type Listener = (e: unknown) => void

interface MockTerm {
  scrollLines: ReturnType<typeof vi.fn>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  _core: any
}

function makeContainer() {
  const handlers: Record<string, Listener> = {}
  const addEventListener = vi.fn(
    (type: string, fn: Listener) => {
      handlers[type] = fn
    },
  )
  const removeEventListener = vi.fn()
  return {
    container: {
      addEventListener,
      removeEventListener,
    } as unknown as HTMLElement,
    handlers,
    addEventListener,
    removeEventListener,
  }
}

function makeTerm(cellHeight: number | undefined = 17): MockTerm {
  return {
    scrollLines: vi.fn(),
    _core: {
      _renderService: {
        dimensions: {
          css: {
            cell: cellHeight === undefined ? {} : { height: cellHeight },
          },
        },
      },
    },
  }
}

function touchEvent(
  type: string,
  touches: Array<{ identifier: number; clientY: number }>,
  changed?: Array<{ identifier: number; clientY: number }>,
) {
  return {
    type,
    touches,
    changedTouches: changed ?? touches,
    preventDefault: vi.fn(),
  }
}

describe('attachTouchScroll', () => {
  let m: ReturnType<typeof makeContainer>
  let term: MockTerm

  beforeEach(() => {
    m = makeContainer()
    term = makeTerm(17)
  })

  // Test 1
  it('is exported and returns a cleanup function', () => {
    const cleanup = attachTouchScroll(m.container, term as unknown as Terminal)
    expect(typeof cleanup).toBe('function')
  })

  // Test 2: finger drag DOWN (positive Δy) → scrollLines NEGATIVE → reveal older
  it('translates Δy = 2*cellHeight to scrollLines(-2) (drag down → older content)', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    const cellH = 17
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 100 }]))
    m.handlers.touchmove(touchEvent('touchmove', [{ identifier: 1, clientY: 100 + 2 * cellH }]))
    expect(term.scrollLines).toHaveBeenCalledWith(-2)
  })

  // Test 3: finger drag UP (negative Δy) → scrollLines POSITIVE → reveal newer
  it('translates Δy = -3*cellHeight to scrollLines(3) (drag up → newer content)', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    const cellH = 17
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 200 }]))
    m.handlers.touchmove(touchEvent('touchmove', [{ identifier: 1, clientY: 200 - 3 * cellH }]))
    expect(term.scrollLines).toHaveBeenCalledWith(3)
  })

  // Test 4: sub-cell-height drag → no scrollLines, no preventDefault
  it('sub-threshold drag (|Δy| < cellHeight) does not scroll or preventDefault', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 100 }]))
    const moveEv = touchEvent('touchmove', [{ identifier: 1, clientY: 104 }]) // Δy = 4 < 17
    m.handlers.touchmove(moveEv)
    expect(term.scrollLines).not.toHaveBeenCalled()
    expect(moveEv.preventDefault).not.toHaveBeenCalled()
  })

  // Test 5: sub-threshold tap on touchend → no preventDefault (preserves OSC 8)
  it('sub-threshold tap (<8px total) on touchend does not preventDefault', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 100 }]))
    // tiny movement of 3px (well below 8px tap threshold)
    m.handlers.touchmove(touchEvent('touchmove', [{ identifier: 1, clientY: 103 }]))
    const endEv = touchEvent('touchend', [], [{ identifier: 1, clientY: 103 }])
    m.handlers.touchend(endEv)
    expect(endEv.preventDefault).not.toHaveBeenCalled()
    expect(term.scrollLines).not.toHaveBeenCalled()
  })

  // Test 6: multi-touch (pinch) → no scrollLines, no preventDefault on touchmove
  it('multi-touch (2 touches) does not trigger scroll or preventDefault', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    m.handlers.touchstart(
      touchEvent('touchstart', [
        { identifier: 1, clientY: 100 },
        { identifier: 2, clientY: 200 },
      ]),
    )
    const moveEv = touchEvent('touchmove', [
      { identifier: 1, clientY: 150 },
      { identifier: 2, clientY: 250 },
    ])
    m.handlers.touchmove(moveEv)
    expect(term.scrollLines).not.toHaveBeenCalled()
    expect(moveEv.preventDefault).not.toHaveBeenCalled()
  })

  // Test 7: confirmed scroll (Δy >= cellHeight) → preventDefault called
  it('preventDefault is called once scroll is confirmed (Δy >= cellHeight)', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    const cellH = 17
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 100 }]))
    const moveEv = touchEvent('touchmove', [{ identifier: 1, clientY: 100 + cellH }])
    m.handlers.touchmove(moveEv)
    expect(term.scrollLines).toHaveBeenCalledWith(-1)
    expect(moveEv.preventDefault).toHaveBeenCalled()
  })

  // Test 8: cleanup removes all four listeners
  it('cleanup removes touchstart, touchmove, touchend, and touchcancel listeners', () => {
    const cleanup = attachTouchScroll(m.container, term as unknown as Terminal)
    cleanup()
    const removed = m.removeEventListener.mock.calls.map((c) => c[0])
    expect(removed).toContain('touchstart')
    expect(removed).toContain('touchmove')
    expect(removed).toContain('touchend')
    expect(removed).toContain('touchcancel')
    expect(removed.length).toBe(4)
  })

  // Test 9: cell height read LIVE on each touchmove
  it('reads cell height live from term._core path on each touchmove', () => {
    attachTouchScroll(m.container, term as unknown as Terminal)
    // First move with cellH = 17 → drag 34px should yield scrollLines(-2)
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 100 }]))
    m.handlers.touchmove(touchEvent('touchmove', [{ identifier: 1, clientY: 134 }]))
    expect(term.scrollLines).toHaveBeenLastCalledWith(-2)
    // Now mutate the private path to simulate a font-size change → cellH = 34
    term._core._renderService.dimensions.css.cell.height = 34
    // From baseline 134, drag another 34px (down to 168) → should yield ONE more line (-1)
    m.handlers.touchmove(touchEvent('touchmove', [{ identifier: 1, clientY: 168 }]))
    expect(term.scrollLines).toHaveBeenLastCalledWith(-1)
  })

  // Test 10: fallback cell height of 17 when private path is undefined
  it('falls back to cellHeight = 17 when private path is undefined', () => {
    const termNoCore = {
      scrollLines: vi.fn(),
      _core: undefined,
    }
    attachTouchScroll(m.container, termNoCore as unknown as Terminal)
    m.handlers.touchstart(touchEvent('touchstart', [{ identifier: 1, clientY: 100 }]))
    // Δy = 34 = 2 * 17 → scrollLines(-2)
    m.handlers.touchmove(touchEvent('touchmove', [{ identifier: 1, clientY: 134 }]))
    expect(termNoCore.scrollLines).toHaveBeenCalledWith(-2)
  })
})
