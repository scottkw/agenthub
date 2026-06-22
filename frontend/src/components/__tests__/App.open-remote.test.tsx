// Wave 0 RED tests for Phase 146 FIX-03 — out-of-band open contract (REDESIGNED).
//
// This file replaces the broadcast-era tests that validated the now-rejected
// ExchangeJoinCodeAtURL-directly / isPeerSelf / rwJoinCode flow. See CONTEXT.md
// D-02/D-04/D-09..D-12 and superseded-broadcast/README.md.
//
// The new contract:
//   - handleOpenRemoteSession: opens RemoteJoinCodeModal (via setJoinModalForSession)
//     when the viewer has no held cap — does NOT call ExchangeJoinCodeAtURL directly.
//   - handleModalExchange: has an 'open-session' branch that builds
//     /sessions/{id}?cap= URL and calls BrowserOpenURL after exchange.
//   - Dead code removed: isPeerSelf and rwJoinCode selection logic are gone (D-06/WR-01).
//   - Behavior assertion: the "Open in browser" menu item on a remote SessionCard is NOT
//     disabled and clicking it calls the open handler with the session object — exercises
//     the actual open entry point rather than inspecting source in isolation.
//
// Source-inspection tests use the App.tsx?raw import pattern (established by
// App.fileBrowserMode.test.tsx and App.wiring.test.tsx). Behavior tests render
// SessionCard using the pattern from SessionCard.share.test.tsx.
//
// RED state now: App.tsx still has isPeerSelf / rwJoinCode / direct ExchangeJoinCodeAtURL
// call in handleOpenRemoteSession; SessionCard still gates the button on roJoinCode.
// All assertions below FAIL against current code.
// They go GREEN when Plans 02 (Go) and 03 (frontend) rewrite the production code.

import { describe, it, expect, vi, afterEach } from 'vitest'
import React from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'

// Source-inspection via App.tsx?raw (established pattern — no Wails mock needed).
import raw from '../../App.tsx?raw'

// Mock Wails bindings needed transitively by SessionCard.
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  GetSessionTailLines: vi.fn().mockResolvedValue([]),
}))

import { SessionCard } from '../Hub/SessionCard'
import type { SessionInfo } from '../../wailsjs/go/main/App'

// Minimal remote session fixture — intentionally has NO roJoinCode / rwJoinCode
// (the out-of-band design carries no codes in the discovery payload).
const remoteSession: SessionInfo = {
  id: 'sess-remote-1',
  name: 'Remote Session',
  cli: 'claude',
  state: 'running',
  status: 'idle',
  createdAt: '2026-01-01T00:00:00Z',
  hostname: 'remote.host',
  webEnabled: true,
  viewerCount: 0,
  homeDir: false,
  browseEnabled: false,
  workDir: '/home/user',
}

let container: HTMLElement | undefined
let root: Root | undefined

function renderRemoteCard(opts: {
  session?: SessionInfo
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  onOpenInBrowser?: (session: any) => void
}) {
  container = document.createElement('div')
  document.body.appendChild(container)
  root = createRoot(container)
  const session = opts.session ?? remoteSession
  flushSync(() => {
    root!.render(
      React.createElement(SessionCard, {
        session,
        isRemote: true,
        onOpenInBrowser: opts.onOpenInBrowser,
      } as Parameters<typeof SessionCard>[0])
    )
  })
  return { container: container! }
}

afterEach(() => {
  if (root) {
    flushSync(() => root!.unmount())
    root = undefined
  }
  if (container) {
    container.remove()
    container = undefined
  }
  vi.clearAllMocks()
})

// ─── Source-inspection: handleOpenRemoteSession (out-of-band contract) ────────
describe('App.tsx — handleOpenRemoteSession (out-of-band, FIX-03)', () => {
  it('opens RemoteJoinCodeModal via setJoinModalForSession (not BrowserOpenURL directly)', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    expect(idx, 'handleOpenRemoteSession must be present in App.tsx').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 600)
    // Out-of-band: handler opens the modal, not the URL directly.
    expect(slice).toContain('setJoinModalForSession')
    // ExchangeJoinCodeAtURL moves to handleModalExchange — not called here.
    expect(slice).not.toContain('ExchangeJoinCodeAtURL')
  })

  it('dead code removed: isPeerSelf is gone from App.tsx (D-06/WR-01)', () => {
    expect(raw).not.toContain('isPeerSelf')
  })

  it('dead code removed: rwJoinCode selection is gone from App.tsx (D-06/WR-01)', () => {
    expect(raw).not.toContain('rwJoinCode')
  })
})

// ─── Source-inspection: handleModalExchange open-session branch ───────────────
describe('App.tsx — handleModalExchange open-session branch (FIX-03)', () => {
  it("has an 'open-session' intent branch in handleModalExchange", () => {
    const idx = raw.indexOf('handleModalExchange')
    expect(idx, 'handleModalExchange must be present in App.tsx').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 1000)
    expect(slice).toContain('open-session')
  })

  it('open-session branch builds a /sessions/{id}?cap= URL', () => {
    const idx = raw.indexOf('handleModalExchange')
    const slice = raw.slice(idx, idx + 1000)
    expect(slice).toMatch(/\/sessions\/.*\?cap=/)
  })

  it('open-session branch calls BrowserOpenURL with the cap-bearing URL', () => {
    const idx = raw.indexOf('handleModalExchange')
    const slice = raw.slice(idx, idx + 1000)
    expect(slice).toContain('BrowserOpenURL')
  })
})

// ─── Behavior: remote SessionCard "Open in browser" exercises the open entry point ─
//
// This is the cross-path assertion the prior test suite lacked. The previous suite
// only inspected source text in isolation — none of the tests exercised whether the
// button actually fires the handler. A green source-only suite cannot certify a
// working feature (it certified the dead-on-arrival broadcast approach too).
//
// Per D-03 (out-of-band model): the button must NOT be disabled — even when the
// session carries no roJoinCode — because the modal is how the viewer obtains the
// code. Gating the button on roJoinCode was the broadcast-era behavior (removed).
describe('SessionCard — "Open in browser" behavior (FIX-03 behavior-level assertion)', () => {
  it('"Open in browser" is present and enabled on a remote card with no roJoinCode (D-03: modal replaces dead-end)', () => {
    // Session fixture intentionally has NO roJoinCode — the out-of-band design
    // does not broadcast codes; the modal guides the viewer to obtain one.
    const { container: c } = renderRemoteCard({ session: remoteSession })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    expect(menuBtn).not.toBeNull()
    flushSync(() => { menuBtn.click() })
    const openBtn = Array.from(c.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Open in browser')
    ) as HTMLButtonElement | undefined
    expect(openBtn, '"Open in browser" menu item must be present on a remote card').toBeDefined()
    // KEY ASSERTION: button must NOT be disabled — the modal replaces the dead-end 401 (D-03).
    // RED: current SessionCard has disabled={!roJoinCode}; this fails because roJoinCode is absent.
    expect(openBtn!.disabled, '"Open in browser" must not be disabled when roJoinCode is absent — modal guides user').toBe(false)
  })

  it('clicking "Open in browser" calls onOpenInBrowser with the session object', () => {
    const onOpenInBrowser = vi.fn()
    const { container: c } = renderRemoteCard({ session: remoteSession, onOpenInBrowser })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    const openBtn = Array.from(c.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Open in browser')
    ) as HTMLButtonElement | undefined
    // Skip click if button is disabled (don't double-fail on the same broken state)
    if (openBtn && !openBtn.disabled) {
      flushSync(() => { openBtn.click() })
      expect(onOpenInBrowser).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'sess-remote-1' })
      )
    } else {
      // Button disabled means the D-03 gate still exists — assert it is enabled first
      expect(openBtn?.disabled, '"Open in browser" must not be disabled (D-03)').toBe(false)
    }
  })
})
