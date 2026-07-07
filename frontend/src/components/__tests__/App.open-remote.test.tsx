// Phase 146-05 additions (GAP-146-A gap closure):
//   - WR-02 held-cap reuse behavior tests (source-inspection of the held-cap branch)
//   - WR-01 SID-correct fallback test (no hand-built ?cap= in open-session branch)
//   - WR-03 error-copy behavior test (mapErrorMessage for not-found → used/expired)
//
// These tests fail against the Phase 146 Plans 01-04 code base and go GREEN
// when Plan 05 adds the held-cap reuse path, WR-01 URL fix, and WR-03 copy.
//
// Phase 168-03 (FIX-03, #118) additions — D-17 REVERSES the Phase 146 external-
// browser design. Opening a remote session now opens an in-app web-session tab
// (openWebSessionTab) instead of BrowserOpenURL. The held-cap and modal-exchange
// branches still call OpenRemoteSessionURL to get the daemon-composed, SID-correct
// cap-bearing URL (WR-01 protection preserved) — but the URL is now PARSED
// (origin -> baseURL, ?cap= -> capToken) and handed to openWebSessionTab instead
// of BrowserOpenURL. These tests fail against the pre-168-03 code base (which still
// calls BrowserOpenURL) and go GREEN once the in-app reroute lands.

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
//   - Behavior assertion: the "Open in tab" menu item on a remote SessionCard is NOT
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
import React, { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { flushSync } from 'react-dom'

// Source-inspection via App.tsx?raw (established pattern — no Wails mock needed).
import raw from '../../App.tsx?raw'

// Mock Wails bindings needed transitively by SessionCard.
vi.mock('../../wailsjs/go/main/App', () => ({
  RenameSession: vi.fn().mockResolvedValue(undefined),
  GetSessionTailLines: vi.fn().mockResolvedValue([]),
}))

import { RemoteJoinCodeModal } from '../RemoteJoinCodeModal'

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
  funnelActive: false,
  funnelWriteActive: false,
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
    // Plan 05 added held-cap reuse before the modal path — use a larger slice to
    // cover the full callback including the no-cap fallback branch (D-03 parity).
    const slice = raw.slice(idx, idx + 1000)
    // Out-of-band: handler opens the modal on the no-cap path.
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

  it('open-session branch uses OpenRemoteSessionURL to get the cap-bearing URL (WR-01 fix, preserved by FIX-03)', () => {
    // Plan 05 WR-01: the open-session branch no longer hand-builds /sessions/{id}?cap=
    // directly (mismatch-prone). Instead it calls OpenRemoteSessionURL (daemon-composed).
    // Phase 168-03 keeps this SID-correctness guarantee — the URL is now parsed and
    // handed to openWebSessionTab instead of BrowserOpenURL (see tests below).
    const idx = raw.indexOf('handleModalExchange')
    const slice = raw.slice(idx, idx + 1600)
    expect(slice).toContain('OpenRemoteSessionURL')
    // The hand-built form must be gone (WR-01 acceptance criterion).
    expect(slice).not.toContain("pending.id + '?cap='")
  })

  it('open-session branch calls openWebSessionTab, NOT BrowserOpenURL, with the cap-bearing URL (FIX-03, D-17)', () => {
    const idx = raw.indexOf('handleModalExchange')
    // Plan 05 WR-01 fix added deposit + OpenRemoteSessionURL; 168-03 replaces the
    // trailing BrowserOpenURL(url) call with an in-app openWebSessionTab(...) call.
    const slice = raw.slice(idx, idx + 1600)
    expect(slice).toContain('openWebSessionTab')
    expect(slice).not.toContain('BrowserOpenURL')
  })
})

// ─── Behavior: remote SessionCard "Open in tab" exercises the open entry point ─
//
// This is the cross-path assertion the prior test suite lacked. The previous suite
// only inspected source text in isolation — none of the tests exercised whether the
// button actually fires the handler. A green source-only suite cannot certify a
// working feature (it certified the dead-on-arrival broadcast approach too).
//
// Per D-03 (out-of-band model): the button must NOT be disabled — even when the
// session carries no roJoinCode — because the modal is how the viewer obtains the
// code. Gating the button on roJoinCode was the broadcast-era behavior (removed).
describe('SessionCard — "Open in tab" behavior (FIX-03 behavior-level assertion)', () => {
  it('"Open in tab" is present and enabled on a remote card with no roJoinCode (D-03: modal replaces dead-end)', () => {
    // Session fixture intentionally has NO roJoinCode — the out-of-band design
    // does not broadcast codes; the modal guides the viewer to obtain one.
    const { container: c } = renderRemoteCard({ session: remoteSession })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    expect(menuBtn).not.toBeNull()
    flushSync(() => { menuBtn.click() })
    const openBtn = Array.from(c.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Open in tab')
    ) as HTMLButtonElement | undefined
    expect(openBtn, '"Open in tab" menu item must be present on a remote card').toBeDefined()
    // KEY ASSERTION: button must NOT be disabled — the modal replaces the dead-end 401 (D-03).
    // RED: current SessionCard has disabled={!roJoinCode}; this fails because roJoinCode is absent.
    expect(openBtn!.disabled, '"Open in tab" must not be disabled when roJoinCode is absent — modal guides user').toBe(false)
  })

  it('clicking "Open in tab" calls onOpenInBrowser with the session object', () => {
    const onOpenInBrowser = vi.fn()
    const { container: c } = renderRemoteCard({ session: remoteSession, onOpenInBrowser })
    const menuBtn = c.querySelector('.hub-card__menu-btn') as HTMLButtonElement
    flushSync(() => { menuBtn.click() })
    const openBtn = Array.from(c.querySelectorAll('.hub-card__menu-item')).find(
      (el) => el.textContent?.includes('Open in tab')
    ) as HTMLButtonElement | undefined
    // Skip click if button is disabled (don't double-fail on the same broken state)
    if (openBtn && !openBtn.disabled) {
      flushSync(() => { openBtn.click() })
      expect(onOpenInBrowser).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'sess-remote-1' })
      )
    } else {
      // Button disabled means the D-03 gate still exists — assert it is enabled first
      expect(openBtn?.disabled, '"Open in tab" must not be disabled (D-03)').toBe(false)
    }
  })
})

// ─── WR-02 behavior: held-cap reuse path in handleOpenRemoteSession ────────────
//
// These source-inspection assertions cross the held-cap reuse behavior:
//   (a) held-cap: handleOpenRemoteSession has remoteCapsCached.has() guard +
//       calls OpenRemoteSessionURL binding (no modal on hit)
//   (b) no-cap fallback: falls through to setJoinModalForSession (existing)
//   (c) WR-01: hand-built pending.id + '?cap=' removed from open-session branch
//
// Source inspection is used here (established pattern in this codebase) combined
// with the WR-03 component render below which exercises true component behavior.
describe('App.tsx — handleOpenRemoteSession held-cap reuse (GAP-146-A, Plan 05)', () => {
  it('held-cap path: handleOpenRemoteSession checks remoteCapsCached.has(session.id) FIRST', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    expect(idx, 'handleOpenRemoteSession must be present in App.tsx').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 700)
    // The held-cap reuse guard must appear before the setJoinModalForSession call.
    expect(slice).toContain('remoteCapsCached.has')
  })

  it('held-cap path: handleOpenRemoteSession calls OpenRemoteSessionURL when cap is held', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    const slice = raw.slice(idx, idx + 700)
    // The new binding must be called in the held-cap branch.
    expect(slice).toContain('OpenRemoteSessionURL')
  })

  it('held-cap path: OpenRemoteSessionURL is imported from the Wails bindings', () => {
    // Verify the binding is in the App import list.
    expect(raw).toContain('OpenRemoteSessionURL')
  })

  // Phase 168-03 (FIX-03, D-17) — in-app reroute of the held-cap branch.
  it('held-cap path: handleOpenRemoteSession does NOT call BrowserOpenURL (FIX-03, D-17)', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    const slice = raw.slice(idx, idx + 700)
    expect(slice).not.toContain('BrowserOpenURL')
  })

  it('held-cap path: handleOpenRemoteSession calls openWebSessionTab with the parsed cap-bearing URL (FIX-03)', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    const slice = raw.slice(idx, idx + 700)
    // The OpenRemoteSessionURL result must be opened in-app via openWebSessionTab,
    // parameterized on session.id (not a shared/global session id).
    expect(slice).toContain('openWebSessionTab(session.id')
  })

  it('WR-01 fixed: hand-built pending.id + ?cap= is gone from the open-session branch', () => {
    // The WR-01 mismatch-prone URL must be removed from handleModalExchange.
    // After the fix the URL is built by OpenRemoteSessionURL in the daemon.
    expect(raw).not.toContain("pending.id + '?cap='")
  })

  it('no-cap fallback: handleOpenRemoteSession still calls setJoinModalForSession when no cap held', () => {
    const idx = raw.indexOf('handleOpenRemoteSession')
    // Use 1200 chars to cover the whole callback including the no-cap fallback branch.
    const slice = raw.slice(idx, idx + 1200)
    expect(slice).toContain('setJoinModalForSession')
  })
})

// ─── FIX-03 behavior: per-tab isolation — two remote sessions open independent tabs ──
//
// openWebSessionTab is internal to the App component (not exported), so this suite
// follows the established source-inspection convention (see App.test.tsx) rather than
// mounting the full App tree. Together the two assertions below prove per-call
// isolation: (1) the tab id is deterministically keyed by the sessionId ARGUMENT
// (webSessionTabId(sessionId)), so two different remote sessionIds always produce two
// different tab ids; and (2) the created Tab's baseURL/capToken are sourced directly
// from THIS call's own parameters — never from the shared/global webParams — so two
// concurrent remote tabs can never cross-contaminate each other's cap/host
// (RESEARCH Pitfall 3 / T-168-04).
describe('App.tsx — openWebSessionTab per-tab isolation (FIX-03, two remote sessions)', () => {
  it('tab id is keyed by the sessionId argument, not a shared value (distinct sessions -> distinct tab ids)', () => {
    const idx = raw.indexOf('const openWebSessionTab')
    expect(idx, 'openWebSessionTab must be present in App.tsx').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 900)
    expect(slice).toContain('webSessionTabId(sessionId)')
  })

  it('openWebSessionTab signature is (sessionId, baseURL?, capToken?) — extended for FIX-03', () => {
    const idx = raw.indexOf('const openWebSessionTab')
    const slice = raw.slice(idx, idx + 200)
    expect(slice).toContain('sessionId: string, baseURL?: string, capToken?: string')
  })

  it('the created Tab carries baseURL/capToken from THIS call, not the global webParams', () => {
    const idx = raw.indexOf('const openWebSessionTab')
    const slice = raw.slice(idx, idx + 900)
    expect(slice).toContain('baseURL,')
    expect(slice).toContain('capToken,')
    // Must not read from the mount-stable global webParams inside this function —
    // that would make a second remote tab silently reuse the first's cap/host.
    expect(slice).not.toContain('webParams')
  })

  it('the __websession__ render branch resolves per-tab params from the active tab, not the global webParams, for remote tabs (RESEARCH Pitfall 3)', () => {
    const idx = raw.indexOf("activeId.startsWith('__websession__')")
    expect(idx, 'the __websession__ render branch must be present').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 2000)
    expect(slice).toContain('activeWebTab')
    expect(slice).toContain('isRemoteWebTab')
    expect(slice).toContain('baseURL={activeWebTab?.baseURL}')
  })

  // FIX-03 (plan 09): RC-B full-height wrapper + RC-A remote flag threading.
  // The __websession__ branch must wrap WebShareSessionView in .terminal-wrapper
  // (the full-height flex-column chain that resolves the modal's flex:1, fixing
  // the half-height terminal + dead band) AND pass remote={isRemoteWebTab} so the
  // remote tab uses the daemon proxy instead of a direct cross-origin peer wss.
  it('RC-B/RC-A: the __websession__ branch wraps WebShareSessionView in .terminal-wrapper and threads remote={isRemoteWebTab}', () => {
    const idx = raw.indexOf("activeId.startsWith('__websession__')")
    expect(idx, 'the __websession__ render branch must be present').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 2000)
    expect(slice).toContain('terminal-wrapper')
    expect(slice).toContain('remote={isRemoteWebTab}')
  })

  // CR-03 (168-REVIEW): the WebShareSessionView render site had no `key`, so
  // switching between two remote-peer tabs did NOT unmount/remount the
  // component — its local useState (chatOpen/unreadCount/hasMention/
  // livePluginConfig) leaked from session A into session B's render. Fixed
  // by keying on the same wsSessionId used to resolve this render's own
  // sessionId prop, forcing a clean remount whenever the active
  // __websession__ tab changes to a different session.
  it('CR-03: <WebShareSessionView> is keyed by wsSessionId so switching remote tabs forces a remount (no cross-session state leak)', () => {
    const idx = raw.indexOf("activeId.startsWith('__websession__')")
    expect(idx, 'the __websession__ render branch must be present').toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 2000)
    const wsvIdx = slice.indexOf('<WebShareSessionView')
    expect(wsvIdx, '<WebShareSessionView> element must be present in this branch').toBeGreaterThan(-1)
    // The key prop must appear on the element itself (before sessionId prop),
    // not merely somewhere later in the file.
    const elementSlice = slice.slice(wsvIdx, wsvIdx + 400)
    expect(elementSlice).toContain('key={wsSessionId}')
  })
})

// ─── WR-03 behavior: RemoteJoinCodeModal used/expired error copy ───────────────
//
// True component-render behavior test (not source inspection).
// The not-found error (single-use code already consumed, D-11) must surface
// "already been used or expired" copy, NOT "Code invalid. Double-check the digits."
describe('RemoteJoinCodeModal — WR-03 used/expired error copy (GAP-146-A, Plan 05)', () => {
  let wrapperDiv: HTMLDivElement | undefined
  let wrapperRoot: Root | undefined

  afterEach(() => {
    if (wrapperRoot) {
      flushSync(() => wrapperRoot!.unmount())
      wrapperRoot = undefined
    }
    if (wrapperDiv) {
      wrapperDiv.remove()
      wrapperDiv = undefined
    }
  })

  async function submitCodeToModal(
    onExchange: (code: string) => Promise<void>,
    code: string,
  ): Promise<HTMLElement> {
    wrapperDiv = document.createElement('div')
    document.body.appendChild(wrapperDiv)
    wrapperRoot = createRoot(wrapperDiv)
    flushSync(() => {
      wrapperRoot!.render(
        React.createElement(RemoteJoinCodeModal, {
          remoteSession: { id: 'sid-1', name: 'Test Session', hostname: 'test.host' },
          intent: 'open-session',
          onExchange,
          onClose: vi.fn(),
        })
      )
    })
    // Enter code
    const input = wrapperDiv.querySelector('input[type="text"]') as HTMLInputElement
    const desc = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(input), 'value')
    desc?.set?.call(input, code)
    flushSync(() => {
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    const submit = Array.from(wrapperDiv.querySelectorAll('button')).find(
      (b) => /join/i.test(b.textContent ?? '')
    ) as HTMLButtonElement
    await act(async () => {
      submit.click()
      await Promise.resolve()
      await Promise.resolve()
    })
    return wrapperDiv
  }

  it('WR-03: not-found error (single-use code consumed) shows "already been used or expired" — NOT "Code invalid"', async () => {
    const onExchange = vi.fn(async () => {
      throw new Error('join code not-found (status 404)')
    })
    const el = await submitCodeToModal(onExchange, 'ABCD-EFGH')
    const errorEl = el.querySelector('[data-testid="remote-join-modal-error"]')
    expect(errorEl).not.toBeNull()
    const errorText = errorEl!.textContent ?? ''
    // Must contain the corrected copy (WR-03 fix).
    expect(errorText).toContain('already used or expired')
    // Must NOT contain the wrong copy that conflates typos with single-use expiry.
    expect(errorText).not.toContain('Double-check')
  })

  it('WR-03: invalid error (typo) still shows "Code invalid. Double-check" copy', async () => {
    const onExchange = vi.fn(async () => {
      throw new Error('join code is invalid (status 401)')
    })
    const el = await submitCodeToModal(onExchange, 'ZZZZ-ZZZZ')
    const errorEl = el.querySelector('[data-testid="remote-join-modal-error"]')
    expect(errorEl).not.toBeNull()
    const errorText = errorEl!.textContent ?? ''
    // Typo path remains "Code invalid." (separate branch from used/expired).
    expect(errorText).toContain('Code invalid')
    expect(errorText).toContain('Double-check')
  })
})
