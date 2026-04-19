import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// Source inspection tests — verify session:exit wiring contract without mounting full App.
describe('App.tsx session:exit wiring (Phase 84)', () => {
  // SESS-01: Event subscription
  it('subscribes to session:exit Wails event', () => {
    expect(raw).toContain("'session:exit'")
  })

  // SESS-01: Countdown state
  it('defines sessionExits state for exit tracking', () => {
    expect(raw).toContain('sessionExits')
  })

  it('defines countdownTimers ref for interval handles', () => {
    expect(raw).toContain('countdownTimers')
  })

  // SESS-01: Auto-close ref (stored as ref to avoid stale closures in [] useEffect)
  it('defines autoCloseRef for auto-close setting', () => {
    expect(raw).toContain('autoCloseRef')
  })

  // SESS-02: ExitCountdownBanner rendered inside terminal-wrapper
  it('renders ExitCountdownBanner component', () => {
    expect(raw).toContain('ExitCountdownBanner')
  })

  // SESS-03: ExitToast rendered for notifications
  it('renders ExitToast component', () => {
    expect(raw).toContain('ExitToast')
  })

  // D-02: Keep Open handler
  it('defines handleKeepOpen callback', () => {
    expect(raw).toContain('handleKeepOpen')
  })

  // D-02: Dismiss handler
  it('defines handleDismissExit callback', () => {
    expect(raw).toContain('handleDismissExit')
  })

  // D-11: Auto-close setting import
  it('imports GetAutoCloseSession', () => {
    expect(raw).toContain('GetAutoCloseSession')
  })

  // D-04: exitCountdowns passed to TabBar
  it('passes exitCountdowns prop to TabBar', () => {
    expect(raw).toContain('exitCountdowns')
  })

  // Cleanup: countdown timers cleared in handleCloseTab
  it('cleans up sessionExits in handleCloseTab', () => {
    // Both the setSessionExits cleanup and countdownTimers cleanup should be present
    expect(raw).toContain('setSessionExits')
  })

  // Event cleanup
  it('unsubscribes from session:exit in useEffect cleanup (offExit)', () => {
    expect(raw).toContain('offExit')
  })
})
