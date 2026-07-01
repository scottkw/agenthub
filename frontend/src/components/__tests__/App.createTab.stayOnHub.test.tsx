/**
 * Phase 168-04 UX-01 (#116) — createTab auto-switch gating tests (TDD).
 *
 * App.tsx's createTab is not exported and App is not fully mounted in this
 * codebase's test suite (established convention: source-inspection via
 * App.tsx?raw for App-level logic — see App.open-remote.test.tsx,
 * App.wiring.test.tsx, App.fileBrowserMode.test.tsx).
 *
 * Covers:
 *   - createTab loads the stayOnHubAfterCreate preference via
 *     GetStayOnHubAfterCreate on mount.
 *   - createTab always appends the new tab via setTabs (D-10 — tab creation
 *     is never skipped).
 *   - The single setActiveId(sessionId) call is gated on
 *     stayOnHubAfterCreateRef.current — the ONLY auto-switch in the app
 *     (D-11), no fromHub flag introduced.
 *   - setTabs is called BEFORE the gated setActiveId, so the tab always
 *     exists regardless of the setting.
 */
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('UX-01 (#116): createTab auto-switch gating — source contract', () => {
  it('imports GetStayOnHubAfterCreate from Wails bindings', () => {
    expect(raw).toContain('GetStayOnHubAfterCreate')
  })

  it('loads stayOnHubAfterCreate into a ref on mount (same pattern as autoCloseRef)', () => {
    expect(raw).toContain('stayOnHubAfterCreateRef')
    expect(raw).toContain('GetStayOnHubAfterCreate().then(val => { stayOnHubAfterCreateRef.current = val })')
  })

  it('createTab still calls setTabs unconditionally (D-10: tab always created)', () => {
    const idx = raw.indexOf('const createTab = useCallback')
    expect(idx).toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 1600)
    expect(slice).toContain('setTabs((prev) => [...prev, tab])')
  })

  it('createTab gates the single setActiveId(sessionId) call on stayOnHubAfterCreateRef.current', () => {
    const idx = raw.indexOf('const createTab = useCallback')
    expect(idx).toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 1600)
    expect(slice).toContain('if (!stayOnHubAfterCreateRef.current)')
    expect(slice).toContain('setActiveId(sessionId)')

    // setTabs must run BEFORE the gate — the tab is created regardless of
    // the setting (D-10); only the switch is conditional.
    const setTabsIdx = slice.indexOf('setTabs((prev) => [...prev, tab])')
    const gateIdx = slice.indexOf('if (!stayOnHubAfterCreateRef.current)')
    const setActiveIdx = slice.indexOf('setActiveId(sessionId)')
    expect(setTabsIdx).toBeGreaterThan(-1)
    expect(gateIdx).toBeGreaterThan(setTabsIdx)
    expect(setActiveIdx).toBeGreaterThan(gateIdx)
  })

  it('does not introduce a fromHub flag — createTab has no fromHub parameter (D-11)', () => {
    const idx = raw.indexOf('const createTab = useCallback')
    expect(idx).toBeGreaterThan(-1)
    // Signature line only — createTab(cliName, workDir, args), no extra param.
    const sigSlice = raw.slice(idx, idx + 120)
    expect(sigSlice).not.toContain('fromHub')
  })
})
