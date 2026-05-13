/**
 * Phase 107-04 SHELL-12 — App.tsx session:exit branching tests.
 *
 * UI-SPEC §4 SHELL-12 five-assertion test contract:
 *
 * 1. session:exit with exitCode 0 does NOT add entry to sessionExits state
 *    (setSessionExits is never called for clean exits).
 * 2. session:exit with exitCode 0 calls handleCloseTab with the session id
 *    (via handleCloseTabRef.current — early-return branch).
 * 3. session:exit with exitCode 1 adds entry to sessionExits state and does
 *    NOT call handleCloseTab (existing ExitToast path).
 * 4. ExitToast receives no entries for exit-code-0 sessions
 *    (implicit from #1; asserted structurally).
 * 5. When the active tab closes, activeId shifts to the adjacent tab id
 *    (or 'welcome' if it was the only non-Welcome tab).
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
   * Assertion 1 (UI-SPEC §4 SHELL-12 #1):
   * session:exit with exitCode 0 does NOT add entry to sessionExits state.
   *
   * The early-return branch MUST appear BEFORE any setSessionExits call.
   * We verify this by asserting that the early-return (handleCloseTabRef.current)
   * comes before the first setSessionExits call in the handler body.
   */
  it('1: exit-code-0 early-return branch appears before setSessionExits call', () => {
    const handler = getExitHandlerBody()
    const earlyReturnIdx = handler.indexOf('handleCloseTabRef.current')
    const setExitsIdx = handler.indexOf('setSessionExits')
    expect(earlyReturnIdx).toBeGreaterThan(-1)
    expect(setExitsIdx).toBeGreaterThan(-1)
    expect(earlyReturnIdx).toBeLessThan(setExitsIdx)
  })

  /**
   * Assertion 2 (UI-SPEC §4 SHELL-12 #2):
   * session:exit with exitCode 0 calls handleCloseTab with the session id.
   *
   * The early-return block must reference handleCloseTabRef.current and
   * pass data.sessionId to it, then return immediately without falling
   * through to the ExitToast path.
   */
  it('2: early-return block calls handleCloseTabRef.current with data.sessionId', () => {
    const handler = getExitHandlerBody()
    // The branch shape from UI-SPEC §2:
    //   if (data.exitCode === 0) {
    //     void handleCloseTabRef.current?.(data.sessionId)
    //     return
    //   }
    expect(handler).toContain('data.exitCode === 0')
    const branchIdx = handler.indexOf('data.exitCode === 0')
    const branchBody = handler.slice(branchIdx, branchIdx + 200)
    expect(branchBody).toContain('handleCloseTabRef.current')
    expect(branchBody).toContain('data.sessionId')
    expect(branchBody).toContain('return')
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
   * Assertion 4 (UI-SPEC §4 SHELL-12 #4):
   * ExitToast receives no entries for exit-code-0 sessions.
   *
   * This is structurally guaranteed by the early-return: setSessionExits is
   * never called for exit-code 0, so sessionExits never contains an entry
   * for clean-exit sessions. Assert that no setSessionExits call exists inside
   * the exitCode === 0 branch (i.e., before the `return` statement).
   */
  it('4: exitCode === 0 branch does NOT call setSessionExits (ExitToast gets no entries)', () => {
    const handler = getExitHandlerBody()
    const branchStart = handler.indexOf('data.exitCode === 0')
    expect(branchStart).toBeGreaterThan(-1)
    // The return statement closes the branch; grab only the branch body
    const branchBody = handler.slice(branchStart, branchStart + 200)
    const returnIdx = branchBody.indexOf('return')
    expect(returnIdx).toBeGreaterThan(-1)
    const onlyBranchBody = branchBody.slice(0, returnIdx)
    expect(onlyBranchBody).not.toContain('setSessionExits')
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
