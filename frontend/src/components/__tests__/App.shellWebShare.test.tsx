/**
 * Phase 101-03 SHELL-07/SHELL-08 — App.tsx shell web-share warning state.
 * Phase 150 SET-01 — warningEnabled gate + re-arm re-sync.
 * Phase 168-05 (UX-02, #115) — RETIREMENT of the footer's direct-toggle interception.
 *
 * D-14 removed the footer's direct `ToggleWebServing` call entirely — the footer
 * "Share Session" button only opens the (lifted) Share modal now. That modal already
 * had its OWN complete, independent shell-warning gate (Phase 150 SET-01,
 * `SessionShareModal.tsx` — `isShellCli`/`pendingShellShare`/`ShellWebShareBanner`,
 * covered by `SessionShareModal.test.tsx`). With the footer no longer toggling
 * directly, App.tsx's footer-only interception mechanism — `handleToggleWeb`,
 * `pendingShellWebToggle` state, and the App-level `<ShellWebShareBanner>` slot in
 * `.banner-stack` — is retired as dead code (nothing sets `pendingShellWebToggle`
 * anymore). This file's assertions that previously targeted that retired mechanism
 * are removed/rewritten below; the parts that remain true (shellWebShareWarned
 * hydration, the confirm/cancel handlers now sourced from `shareModalSession`, and
 * threading into the App-level `<SessionShareModal>` render) are kept.
 *
 * Source-inspection tests (matches App.exit.test.tsx / App.test.tsx patterns —
 * full App mount is gated by extensive Wails-binding mocking that the existing
 * test suite intentionally avoids).
 */
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx shell web-share warned state (Phase 101-03 SHELL-07/SHELL-08)', () => {
  it('imports GetShellWebShareWarned and SetShellWebShareWarned Wails bindings', () => {
    expect(raw).toContain('GetShellWebShareWarned')
    expect(raw).toContain('SetShellWebShareWarned')
  })

  it('declares shellWebShareWarned React state', () => {
    expect(raw).toContain('shellWebShareWarned')
    expect(raw).toContain('setShellWebShareWarned')
  })

  it('does NOT declare pendingShellWebToggle as React state (retired — Phase 168-05 D-14)', () => {
    // The footer-only pending-toggle state is dead now that the footer never
    // toggles directly; the modal has its own independent pendingShellShare.
    // (Explanatory comments may still reference the retired name for history —
    // only the actual useState declaration is asserted absent here.)
    expect(raw).not.toContain('const [pendingShellWebToggle')
    expect(raw).not.toContain('setPendingShellWebToggle(')
  })

  it('does NOT import ShellWebShareBanner or isShellCli at the App.tsx level (retired — Phase 168-05 D-14)', () => {
    // Both were used only by the retired footer-direct-toggle path. The equivalent
    // gate now lives entirely inside SessionShareModal.tsx (imports there, tested
    // by SessionShareModal.test.tsx).
    expect(raw).not.toContain("from './components/ShellWebShareBanner'")
    expect(raw).not.toContain("from './lib/shellCli'")
  })

  it('calls GetShellWebShareWarned in a mount useEffect to seed shellWebShareWarned', () => {
    // The mount-time hydration must invoke GetShellWebShareWarned() and feed
    // the result into setShellWebShareWarned via a .then() chain (matching
    // the Phase 81 D-11 GetAutoCloseSession precedent).
    expect(raw).toContain('GetShellWebShareWarned()')
    // The call-site (NOT the doc comment) must be followed within ~250 chars
    // by a .then(...) that invokes setShellWebShareWarned.
    expect(raw).toMatch(
      /GetShellWebShareWarned\(\)[\s\S]{0,250}\.then\([\s\S]{0,200}setShellWebShareWarned/
    )
  })

  it('on confirm: calls SetShellWebShareWarned(true) and ToggleWebServing in parallel', () => {
    // Body must reference both RPCs in the confirm handler.
    expect(raw).toContain('SetShellWebShareWarned(true)')
    // The Promise.all (or equivalent parallel structure) keeps them concurrent.
    expect(raw).toMatch(/Promise\.all\([\s\S]{0,200}SetShellWebShareWarned/)
  })

  it("on confirm: sets shellWebShareWarned synchronously BEFORE await (race mitigation per RESEARCH §8)", () => {
    // The race comment is part of the locked contract — drift would break the
    // mitigation and make the second-shell-toggle test scenario regress.
    expect(raw).toMatch(/race|RESEARCH §8|synchronous(ly)?/i)
    // setShellWebShareWarned(true) must appear before Promise.all in the
    // confirm handler. Locate the confirm function block first.
    const confirmIdx = raw.indexOf('handleShellWebShareConfirm')
    expect(confirmIdx).toBeGreaterThan(-1)
    const confirmBlock = raw.slice(confirmIdx, confirmIdx + 1500)
    const syncSetIdx = confirmBlock.indexOf('setShellWebShareWarned(true)')
    const promiseAllIdx = confirmBlock.indexOf('Promise.all')
    expect(syncSetIdx).toBeGreaterThan(-1)
    expect(promiseAllIdx).toBeGreaterThan(syncSetIdx)
  })

  it('Phase 168-05: handleShellWebShareConfirm sources sessionId from shareModalSession (not the retired pendingShellWebToggle)', () => {
    const confirmIdx = raw.indexOf('const handleShellWebShareConfirm')
    expect(confirmIdx).toBeGreaterThan(-1)
    const confirmBlock = raw.slice(confirmIdx, confirmIdx + 1500)
    expect(confirmBlock).toContain('shareModalSession')
    expect(confirmBlock).not.toContain('pendingShellWebToggle')
  })

  it('handleShellWebShareCancel exists and calls no Wails RPC (the modal owns its own pendingShellShare reset)', () => {
    const cancelIdx = raw.indexOf('const handleShellWebShareCancel')
    expect(cancelIdx).toBeGreaterThan(-1)
    const cancelBlock = raw.slice(cancelIdx, cancelIdx + 300)
    expect(cancelBlock).not.toContain('SetShellWebShareWarned')
    expect(cancelBlock).not.toContain('ToggleWebServing')
  })

  it('the retired App-level ShellWebShareBanner banner-stack slot is gone', () => {
    // Phase 101-03 SHELL-08 previously rendered <ShellWebShareBanner> at the top of
    // .banner-stack for the footer's direct toggle. The equivalent banner now lives
    // only inside SessionShareModal.tsx (Phase 150 SET-01, tested separately).
    expect(raw).not.toContain('<ShellWebShareBanner')
  })
})

// ---------------------------------------------------------------------------
// Phase 150 SET-01 — warningEnabled gate + re-arm sync
// ---------------------------------------------------------------------------
describe('App.tsx shell web-share warning-enabled gate (Phase 150 SET-01)', () => {
  it('imports GetShellWebShareWarningEnabled from Wails bindings', () => {
    expect(raw).toContain('GetShellWebShareWarningEnabled')
  })

  it('declares shellWebShareWarningEnabled React state (default true)', () => {
    expect(raw).toContain('shellWebShareWarningEnabled')
    expect(raw).toContain('setShellWebShareWarningEnabled')
    // Default must be true (default-ON per D-08)
    expect(raw).toMatch(/shellWebShareWarningEnabled[^)]*useState\(true\)/)
  })

  it('hydrates shellWebShareWarningEnabled from GetShellWebShareWarningEnabled on mount', () => {
    expect(raw).toContain('GetShellWebShareWarningEnabled()')
    // Must be followed by setShellWebShareWarningEnabled in a .then() chain
    expect(raw).toMatch(
      /GetShellWebShareWarningEnabled\(\)[\s\S]{0,300}\.then\([\s\S]{0,200}setShellWebShareWarningEnabled/
    )
  })

  it('SettingsTab render passes onShellWarnEnabledChange callback', () => {
    // App.tsx must thread the re-arm callback into SettingsTab
    expect(raw).toContain('onShellWarnEnabledChange=')
  })

  it('onShellWarnEnabledChange callback calls setShellWebShareWarningEnabled', () => {
    const cbIdx = raw.indexOf('onShellWarnEnabledChange=')
    expect(cbIdx).toBeGreaterThan(-1)
    const cbBlock = raw.slice(cbIdx, cbIdx + 1200)
    expect(cbBlock).toContain('setShellWebShareWarningEnabled')
  })

  it('onShellWarnEnabledChange callback re-fetches GetShellWebShareWarned when enabled=true (re-arm re-sync D-03)', () => {
    // When the user re-enables the warning, App.tsx must re-fetch shellWebShareWarned
    // so the re-armed daemon state is synced back to the frontend.
    const cbIdx = raw.indexOf('onShellWarnEnabledChange=')
    expect(cbIdx).toBeGreaterThan(-1)
    const cbBlock = raw.slice(cbIdx, cbIdx + 800)
    // Must call GetShellWebShareWarned() inside this callback
    expect(cbBlock).toContain('GetShellWebShareWarned()')
    expect(cbBlock).toContain('setShellWebShareWarned')
  })
})

// ---------------------------------------------------------------------------
// Phase 168-05 (UX-02) — shellWebShareWarned/-WarningEnabled + confirm/cancel now
// thread into the single App-level <SessionShareModal> render, not HubPanel.
// ---------------------------------------------------------------------------
describe('App.tsx SessionShareModal render site receives the shell-warn authority props (Phase 168-05)', () => {
  it('the <SessionShareModal> render receives shellWebShareWarned and shellWebShareWarningEnabled', () => {
    const idx = raw.indexOf('<SessionShareModal')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 500)
    expect(block).toContain('shellWebShareWarned={shellWebShareWarned}')
    expect(block).toContain('shellWebShareWarningEnabled={shellWebShareWarningEnabled}')
  })

  it('the <SessionShareModal> render receives onShellWebShareConfirm and onShellWebShareCancel', () => {
    const idx = raw.indexOf('<SessionShareModal')
    expect(idx).toBeGreaterThan(-1)
    const block = raw.slice(idx, idx + 500)
    expect(block).toContain('onShellWebShareConfirm={handleShellWebShareConfirm}')
    expect(block).toContain('onShellWebShareCancel={handleShellWebShareCancel}')
  })

  it('HubPanel no longer receives the shell-warn / SessionShareModal props (moved off HubPanel)', () => {
    const hubIdx = raw.indexOf('<HubPanel')
    expect(hubIdx).toBeGreaterThan(-1)
    const nextCloseIdx = raw.indexOf('/>', hubIdx)
    const hubBlock = raw.slice(hubIdx, nextCloseIdx)
    expect(hubBlock).not.toContain('shellWebShareWarned=')
    expect(hubBlock).not.toContain('shellWebShareWarningEnabled=')
    expect(hubBlock).not.toContain('onShellWebShareConfirm=')
    expect(hubBlock).not.toContain('onShellWebShareCancel=')
    expect(hubBlock).not.toContain('webServerMode=')
    expect(hubBlock).not.toContain('webServerRunning=')
    expect(hubBlock).not.toContain('onOpenHelp=')
    // HubPanel now drives the lifted state via setShareModalSession only.
    expect(hubBlock).toContain('setShareModalSession=')
  })
})
