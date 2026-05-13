/**
 * Phase 107-04 SHELL-12 / CR-01 fix — App.tsx session:exit branching tests.
 *
 * UI-SPEC §4 SHELL-12 five-assertion test contract (updated by CR-01 fix):
 *
 * 1. session:exit with exitCode 0 and auto-close ON: handleCloseTabRef is called
 *    and early-return fires (auto-close path). setSessionExits NOT called on this path.
 * 2. session:exit with exitCode 0 calls handleCloseTab (inside autoCloseRef guard).
 * 3. session:exit with exitCode 1 adds entry to sessionExits state and does
 *    NOT call handleCloseTab (existing ExitToast path).
 * 4. The exitCode === 0 branch body (before inner return) does NOT call setSessionExits
 *    directly (setSessionExits is only reached when autoCloseRef.current is false).
 * 5. When the active tab closes, activeId shifts to the adjacent tab id
 *    (or 'welcome' if it was the only non-Welcome tab).
 *
 * CR-01 fix (Phase 107): autoCloseRef.current must be consulted before calling
 * handleCloseTabRef.current — the close-tab call is now INSIDE the
 * `if (autoCloseRef.current)` guard so the user's preference is honored.
 *
 * Implementation uses source-inspection (`App.tsx?raw`) to lock the structural
 * contract — the same pattern used by App.exit.test.tsx, App.shellWebShare.test.tsx,
 * and App.nav.test.tsx. This avoids mounting the full component tree (which
 * requires extensive Wails-runtime mocking) while asserting the precise code
 * shape mandated by UI-SPEC §2 and §4.
 */
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

// ── Helpers ─────────────────────────────────────────────────────────────────

/** Extract the session:exit handler body from App.tsx source. */
function getExitHandlerBody(): string {
  const start = raw.indexOf("'session:exit'")
  if (start === -1) throw new Error('session:exit subscription not found in App.tsx')
  // Grab enough chars to cover the full handler body (up to the closing paren of EventsOn)
  return raw.slice(start, start + 2000)
}

// ── SHELL-12 Test Suite ──────────────────────────────────────────────────────

describe('App.tsx SHELL-12: session:exit branching on exitCode === 0', () => {

  /**
   * Assertion 1 (UI-SPEC §4 SHELL-12 #1 — updated by CR-01 fix):
   * When auto-close is enabled (autoCloseRef.current === true), session:exit
   * with exitCode 0 calls handleCloseTabRef and does NOT reach setSessionExits.
   *
   * After the CR-01 fix, the close-tab call is inside `if (autoCloseRef.current)`.
   * We verify that the autoCloseRef guard appears before handleCloseTabRef.current
   * in the handler body (ensuring proper gating), and that handleCloseTabRef.current
   * appears before the final setSessionExits call (ensuring auto-close path
   * exits before reaching ExitToast for the auto-close=true case).
   */
  it('1: autoCloseRef guard + handleCloseTabRef appear before setSessionExits call', () => {
    const handler = getExitHandlerBody()
    const autoCloseIdx = handler.indexOf('autoCloseRef.current')
    const closeTabIdx = handler.indexOf('handleCloseTabRef.current')
    // Find setSessionExits after the `return` inside autoCloseRef block
    const returnIdx = handler.indexOf('return')
    const setExitsIdx = handler.indexOf('setSessionExits', returnIdx)
    expect(autoCloseIdx).toBeGreaterThan(-1)
    expect(closeTabIdx).toBeGreaterThan(-1)
    expect(setExitsIdx).toBeGreaterThan(-1)
    // autoCloseRef guard comes before handleCloseTabRef
    expect(autoCloseIdx).toBeLessThan(closeTabIdx)
    // handleCloseTabRef comes before setSessionExits (auto-close path exits early)
    expect(closeTabIdx).toBeLessThan(setExitsIdx)
  })

  /**
   * Assertion 2 (UI-SPEC §4 SHELL-12 #2 — updated by CR-01 fix):
   * session:exit with exitCode 0 calls handleCloseTab with the session id,
   * but only when autoCloseRef.current is true.
   *
   * The structure after CR-01:
   *   if (data.exitCode === 0) {
   *     if (autoCloseRef.current) {
   *       void handleCloseTabRef.current?.(data.sessionId)
   *       return
   *     }
   *   }
   */
  it('2: autoCloseRef guard calls handleCloseTabRef.current with data.sessionId', () => {
    const handler = getExitHandlerBody()
    expect(handler).toContain('data.exitCode === 0')
    expect(handler).toContain('autoCloseRef.current')
    // The autoCloseRef guard block must contain handleCloseTabRef.current and data.sessionId
    const guardIdx = handler.indexOf('if (autoCloseRef.current)')
    expect(guardIdx).toBeGreaterThan(-1)
    // Use a larger window to cover the full guard body
    const guardBody = handler.slice(guardIdx, guardIdx + 250)
    expect(guardBody).toContain('handleCloseTabRef.current')
    expect(guardBody).toContain('data.sessionId')
    expect(guardBody).toContain('return')
  })

  /**
   * Assertion 3 (UI-SPEC §4 SHELL-12 #3):
   * session:exit with exitCode 1 adds entry to sessionExits (existing ExitToast path).
   * The setSessionExits call is preserved for the non-zero path.
   *
   * After the early-return branch, the handler must still call setSessionExits
   * (for the non-zero exit path that shows ExitToast).
   */
  it('3: non-zero exit path still calls setSessionExits (ExitToast preserved)', () => {
    const handler = getExitHandlerBody()
    // setSessionExits must appear AFTER the early-return block in the handler
    const earlyReturnEnd = handler.indexOf('return')
    expect(earlyReturnEnd).toBeGreaterThan(-1)
    const afterReturn = handler.slice(earlyReturnEnd)
    expect(afterReturn).toContain('setSessionExits')
  })

  /**
   * Assertion 4 (UI-SPEC §4 SHELL-12 #4 — updated by CR-01 fix):
   * When auto-close is ON, ExitToast receives no entries for exit-code-0 sessions.
   * When auto-close is OFF, the session falls through to setSessionExits (ExitToast shown).
   *
   * The autoCloseRef guard block (containing handleCloseTabRef and return) must NOT
   * contain a setSessionExits call — setSessionExits is only reached outside the
   * autoCloseRef block. This is the structural invariant that preserves both behaviors.
   */
  it('4: autoCloseRef guard block does NOT call setSessionExits directly', () => {
    const handler = getExitHandlerBody()
    const guardIdx = handler.indexOf('if (autoCloseRef.current)')
    expect(guardIdx).toBeGreaterThan(-1)
    // Get the guard body (up to the matching close brace — use the return as end marker)
    const guardBody = handler.slice(guardIdx, guardIdx + 250)
    const returnIdx = guardBody.indexOf('return')
    expect(returnIdx).toBeGreaterThan(-1)
    // Inside the guard, before the return, there must be no setSessionExits
    const beforeReturn = guardBody.slice(0, returnIdx)
    expect(beforeReturn).not.toContain('setSessionExits')
  })

  /**
   * Assertion 5 (UI-SPEC §4 SHELL-12 #5):
   * When the active tab closes, activeId shifts to the adjacent tab id
   * (or null/welcome if it was the only tab).
   *
   * This invariant is provided by the existing handleCloseTab adjacency logic
   * at App.tsx ~L703-708. Assert that the adjacency logic is still present
   * (not accidentally removed) and that handleCloseTabRef is kept in sync.
   */
  it('5: handleCloseTab adjacency logic still present (focus shifts on active-tab close)', () => {
    // The adjacency pattern from App.tsx ~L703-708:
    //   if (activeId === id) {
    //     const idx = prev.findIndex(...)
    //     const next = remaining[Math.max(0, idx - 1)]
    //     setActiveId(next?.id ?? null)
    //   }
    expect(raw).toContain('activeId === id')
    expect(raw).toContain('setActiveId(next?.id ?? null)')
    // handleCloseTabRef must be kept in sync (assigned after handleCloseTab definition)
    expect(raw).toContain('handleCloseTabRef.current = handleCloseTab')
  })

  // ── CR-01 fix: autoCloseRef.current consulted before handleCloseTabRef ─────

  /**
   * CR-01 (Phase 107): the session:exit handler MUST check autoCloseRef.current
   * before calling handleCloseTabRef.current. The close-tab call must be INSIDE
   * an `if (autoCloseRef.current)` guard so the user's "Auto-close tab on exit"
   * preference is honored. When auto-close is OFF, control falls through to the
   * setSessionExits path so ExitToast is shown.
   */
  it('CR-01: autoCloseRef.current is checked before handleCloseTabRef is called', () => {
    const handler = getExitHandlerBody()
    const autoCloseIdx = handler.indexOf('autoCloseRef.current')
    const closeTabIdx = handler.indexOf('handleCloseTabRef.current')
    expect(autoCloseIdx).toBeGreaterThan(-1)
    expect(closeTabIdx).toBeGreaterThan(-1)
    // autoCloseRef.current guard must appear before the handleCloseTabRef call
    expect(autoCloseIdx).toBeLessThan(closeTabIdx)
  })

  it('CR-01: handleCloseTabRef.current call is inside an autoCloseRef guard block', () => {
    const handler = getExitHandlerBody()
    // The structural shape expected:
    //   if (autoCloseRef.current) {
    //     // Auto-close enabled: close tab immediately (SHELL-12 default path)
    //     void handleCloseTabRef.current?.(data.sessionId)
    //     return
    //   }
    const autoCloseGuardIdx = handler.indexOf('if (autoCloseRef.current)')
    expect(autoCloseGuardIdx).toBeGreaterThan(-1)
    // Use a generous window to cover the guard body including the comment line
    const guardBody = handler.slice(autoCloseGuardIdx, autoCloseGuardIdx + 300)
    expect(guardBody).toContain('handleCloseTabRef.current')
    expect(guardBody).toContain('return')
  })

  it('CR-01: when autoCloseRef is false, control falls through to setSessionExits', () => {
    const handler = getExitHandlerBody()
    // After the autoCloseRef guard block closes, the handler must fall through
    // to setSessionExits (the ExitToast path). Verify setSessionExits appears
    // AFTER the first `return` (the one inside the autoCloseRef guard).
    const firstReturn = handler.indexOf('return')
    const setExitsIdx = handler.indexOf('setSessionExits', firstReturn + 1)
    expect(firstReturn).toBeGreaterThan(-1)
    expect(setExitsIdx).toBeGreaterThan(-1)
    expect(setExitsIdx).toBeGreaterThan(firstReturn)
  })

  // ── Structural invariants that lock the countdown removal ────────────────

  /**
   * SHELL-12 invariant: the old countdown block (setInterval + countdown state)
   * must be removed from the session:exit handler. After the early-return, only
   * the non-zero path remains — no countdown logic.
   */
  it('countdown setInterval block is removed from session:exit handler', () => {
    const handler = getExitHandlerBody()
    // The old countdown block used setInterval inside the handler.
    // After the fix, that block is gone — only the countdownTimers cleanup
    // in handleCloseTab body remains (outside this handler).
    expect(handler).not.toContain('setInterval')
  })

  it('countdown: data.exitCode === 0 ? 5 : -1 pattern is removed', () => {
    // The old exitState used this ternary. After the fix, exit-code 0 never
    // reaches the exitState construction.
    expect(raw).not.toContain('countdown: data.exitCode === 0 ? 5 : -1')
  })

  // ── ExitToast component is still imported and rendered ───────────────────

  it('ExitToast component is still imported (component itself unchanged)', () => {
    expect(raw).toContain("import { ExitToast } from './components/ExitToast'")
  })

  it('ExitToast is still rendered in JSX (non-zero exits still work)', () => {
    expect(raw).toContain('<ExitToast')
  })

  // ── handleCloseTabRef infrastructure still intact ────────────────────────

  it('handleCloseTabRef is still defined and used', () => {
    expect(raw).toContain('handleCloseTabRef')
  })

  // ── countdownTimers cleanup in handleCloseTab still intact ───────────────

  it('countdownTimers cleanup still present in handleCloseTab', () => {
    // The countdownTimers.current cleanup inside handleCloseTab must remain
    // (it is a no-op for exit-code 0 paths after this fix, but must not be
    // removed so future countdown reintroduction remains safe).
    expect(raw).toContain('countdownTimers')
  })
})
