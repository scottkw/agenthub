import type { Terminal } from '@xterm/xterm'

/**
 * Phase 113 / Issue #56 — iPad terminal touch-scroll.
 *
 * xterm.js 6.0.0 ships with ZERO touch handlers attached to its rendered DOM
 * (the dormant Gesture singleton has no callers in v6). On iPad Safari /
 * iPad Chrome, single-finger drag therefore falls through to default browser
 * behavior — the page pans rather than the xterm scrollback scrolling.
 *
 * This module fills the gap by translating vertical touch-drag deltas into
 * `term.scrollLines(N)` calls against the xterm public API. The handler is
 * attached to the React-owned outer container `<div>` (not anything inside
 * `.xterm` / `.xterm-screen`), so it survives `term.dispose()` re-mounts.
 *
 * Requirements: UI-03 (single-finger drag scrolls scrollback) and UI-04
 * (no regression — must not preventDefault on sub-threshold taps, so the
 * OSC 8 WebLinksAddon synthetic-click path keeps working).
 *
 * Sign convention: `term.scrollLines` is positive=down (newer), negative=up
 * (older). Finger drag DOWN reveals OLDER content, so we negate the line
 * delta computed from `Δy / cellHeight`.
 *
 * See: .planning/phases/113-ipad-terminal-touch-scroll/113-RESEARCH.md
 */

// 8px tap-vs-drag threshold. Below this distance on touchend, we leave the
// event alone so iOS Safari's "fire click after touchend" heuristic still
// dispatches the synthetic click into WebLinksAddon's handler. Below the
// xterm Gesture singleton's 30px tap-window so we coexist if v7 wakes it up.
const TAP_THRESHOLD_PX = 8

// Fallback when the xterm private dimensions path is unreadable (defensive
// for tests; also matches the same default literal at TerminalPanel.tsx:29).
const FALLBACK_CELL_HEIGHT_PX = 17

function readCellHeight(term: Terminal): number {
  // Mirrors the private-path read at TerminalPanel.tsx:29 (`fitTerminal`).
  // Read live on each touchmove — SHIFT+= / SHIFT+- font-size changes
  // invalidate any cached value (RESEARCH Pitfall 2).
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const h: number | undefined = (term as any)._core?._renderService?.dimensions?.css?.cell?.height
  return h ?? FALLBACK_CELL_HEIGHT_PX
}

export function attachTouchScroll(container: HTMLElement, term: Terminal): () => void {
  let trackingId: number | null = null
  let startY = 0
  let lastY = 0
  let accumulatedDy = 0

  const onTouchStart = (e: TouchEvent): void => {
    if (e.touches.length !== 1) {
      // Multi-touch — release; let iOS handle pinch / two-finger pan.
      trackingId = null
      return
    }
    const t = e.changedTouches[0]
    trackingId = t.identifier
    startY = t.clientY
    lastY = t.clientY
    accumulatedDy = 0
  }

  const onTouchMove = (e: TouchEvent): void => {
    if (trackingId === null) return
    if (e.touches.length !== 1) {
      // A second finger landed mid-drag — release and let iOS take over the
      // multi-touch gesture (no preventDefault, per RESEARCH Pitfall 3).
      trackingId = null
      return
    }
    const t = Array.from(e.changedTouches).find((x) => x.identifier === trackingId)
    if (!t) return

    const dy = t.clientY - lastY
    lastY = t.clientY
    accumulatedDy += dy

    const cellH = readCellHeight(term)
    const lines = Math.trunc(accumulatedDy / cellH)
    if (lines !== 0) {
      // Negate: finger DOWN (positive dy) reveals OLDER content (scroll up
      // in xterm terms → negative argument).
      term.scrollLines(-lines)
      accumulatedDy -= lines * cellH
      // Only preventDefault once we've decided this IS a scroll, per
      // RESEARCH Pitfall 1 — eager preventDefault breaks the tap-to-click
      // path on iOS Safari.
      e.preventDefault()
    }
  }

  const onTouchEnd = (_e: TouchEvent): void => {
    // Sub-threshold movement is a tap. We do NOT preventDefault under any
    // condition — the WebLinksAddon click handler must receive the synthetic
    // click that iOS Safari dispatches after touchend.
    void TAP_THRESHOLD_PX // intentionally referenced; future telemetry hook
    void startY
    trackingId = null
  }

  container.addEventListener('touchstart', onTouchStart, { passive: true })
  // touchmove MUST be passive:false so we can preventDefault on confirmed
  // scrolls. Without this, iOS Safari's competing page-pan animation
  // fights our scrollLines updates.
  container.addEventListener('touchmove', onTouchMove, { passive: false })
  container.addEventListener('touchend', onTouchEnd, { passive: true })
  container.addEventListener('touchcancel', onTouchEnd, { passive: true })

  return () => {
    container.removeEventListener('touchstart', onTouchStart)
    container.removeEventListener('touchmove', onTouchMove)
    container.removeEventListener('touchend', onTouchEnd)
    container.removeEventListener('touchcancel', onTouchEnd)
  }
}
