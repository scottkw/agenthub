/**
 * funnelBinding.contract.test.tsx
 *
 * Import-contract test for the hand-authored Wails stubs — guards against
 * RESEARCH Pitfalls 1 and 2:
 *   Pitfall 1: updating only the generated wailsjs/wailsjs/go tree (not imported by the app)
 *   Pitfall 2: adding funnelActive only to models.ts, not to the SessionInfo interface in App.d.ts
 *
 * Uses vitest `?raw` source-string imports so the assertions fire at test time,
 * not at compile time — this means the contract survives future wails regenerations
 * that would normally clobber these files.
 */
import { describe, it, expect } from 'vitest'
import appDts from '../../wailsjs/go/main/App.d.ts?raw'
import appJs from '../../wailsjs/go/main/App.js?raw'

describe('Phase 166 / FNL-01 — hand-authored stub import contract', () => {
  it('App.d.ts SessionInfo interface contains funnelActive: boolean', () => {
    // Guards Pitfall 2: funnelActive must be on the interface in the stub,
    // not only in the generated models.ts.
    expect(appDts).toContain('funnelActive: boolean')
  })

  it('App.d.ts SessionInfo interface contains funnelWriteActive: boolean', () => {
    // Phase 177 / FNL-09 (D-05b, optional): mirrors the funnelActive assertion
    // above for the write-share sibling field. Guards the ACTUALLY-imported
    // stub only — the load-bearing runtime guard is the Go-side
    // TestListSessions_PropagatesFunnelWriteActive in app_test.go, which
    // asserts the real serialized JSON rather than this stub text.
    expect(appDts).toContain('funnelWriteActive: boolean')
  })

  it('App.d.ts exports SetSessionFunnel with correct 3-arg signature', () => {
    // Guards Pitfall 1: must be in the hand-authored stub, not the generated copy.
    // Also verifies expiresIn: number is present (FNL-07 no-expiry sentinel = 0).
    expect(appDts).toContain(
      'SetSessionFunnel(sessionID: string, enabled: boolean, expiresIn: number): Promise<void>'
    )
  })

  it('App.js registers main.App.SetSessionFunnel Call wrapper', () => {
    // Guards Pitfall 1: must be in the hand-authored stub.
    expect(appJs).toContain("Call('main.App.SetSessionFunnel', [sessionID, enabled, expiresIn])")
  })
})
