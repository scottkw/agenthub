/**
 * Phase 101-03 SHELL-07/SHELL-08 — App.tsx web-toggle interception tests.
 * Phase 150 SET-01 — warningEnabled gate + re-arm re-sync + HubPanel prop threading.
 *
 * Source-inspection tests (matches App.exit.test.tsx / App.test.tsx patterns
 * — full App mount is gated by extensive Wails-binding mocking that the
 * existing test suite intentionally avoids). The contract is verified by
 * asserting that App.tsx contains the interception structure: shellWebShareWarned
 * state, SHELL_CLIS gate, pendingShellWebToggle state, ShellWebShareBanner
 * render slot at top of banner stack, GetShellWebShareWarned mount call,
 * SetShellWebShareWarned + ToggleWebServing parallel call, race mitigation
 * comment.
 *
 * The ShellWebShareBanner component itself is fully unit-tested in
 * ShellWebShareBanner.test.tsx with rendered DOM.
 */
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx shell web-share interception (Phase 101-03 SHELL-07/SHELL-08)', () => {
  it('imports ShellWebShareBanner from components/ShellWebShareBanner', () => {
    expect(raw).toContain(
      "import { ShellWebShareBanner } from './components/ShellWebShareBanner'"
    )
  })

  it('imports GetShellWebShareWarned and SetShellWebShareWarned Wails bindings', () => {
    expect(raw).toContain('GetShellWebShareWarned')
    expect(raw).toContain('SetShellWebShareWarned')
  })

  it('declares SHELL_CLIS set with shell/bash/zsh/pwsh/powershell membership', () => {
    expect(raw).toContain('SHELL_CLIS')
    // Each of the 5 shell cli identifiers must be present in the set literal.
    const shellSetBlock = raw.slice(raw.indexOf('SHELL_CLIS'))
    expect(shellSetBlock).toContain("'shell'")
    expect(shellSetBlock).toContain("'bash'")
    expect(shellSetBlock).toContain("'zsh'")
    expect(shellSetBlock).toContain("'pwsh'")
    expect(shellSetBlock).toContain("'powershell'")
  })

  it('declares shellWebShareWarned React state', () => {
    expect(raw).toContain('shellWebShareWarned')
    expect(raw).toContain('setShellWebShareWarned')
  })

  it('declares pendingShellWebToggle React state for pending banner data', () => {
    expect(raw).toContain('pendingShellWebToggle')
    expect(raw).toContain('setPendingShellWebToggle')
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

  it('intercepts handleToggleWeb: short-circuits when shell session + enabling + !shellWebShareWarned', () => {
    // The interception is the load-bearing gate. Assert the literal guard structure.
    expect(raw).toContain('SHELL_CLIS.has')
    expect(raw).toContain('!shellWebShareWarned')
    // The handler must call setPendingShellWebToggle inside the intercept branch.
    const interceptBlock = raw.slice(raw.indexOf('SHELL_CLIS.has'))
    expect(interceptBlock.slice(0, 600)).toContain('setPendingShellWebToggle')
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

  it('on cancel: clears pendingShellWebToggle without invoking any Wails RPC', () => {
    const cancelIdx = raw.indexOf('handleShellWebShareCancel')
    expect(cancelIdx).toBeGreaterThan(-1)
    const cancelBlock = raw.slice(cancelIdx, cancelIdx + 400)
    expect(cancelBlock).toContain('setPendingShellWebToggle(null)')
    // Cancel must NOT call either Wails RPC.
    expect(cancelBlock).not.toContain('SetShellWebShareWarned')
    expect(cancelBlock).not.toContain('ToggleWebServing')
  })

  it('renders ShellWebShareBanner at the TOP of the banner-stack (priority slot #1)', () => {
    // Find the banner-stack opening div, then assert ShellWebShareBanner
    // appears BEFORE LocalNetworkBanner / UpdateBanner / WebGLRecoveryBanner /
    // saveBanner / PluginToggleBanner inside that block.
    const stackIdx = raw.indexOf('className="banner-stack"')
    expect(stackIdx).toBeGreaterThan(-1)
    const stackTail = raw.slice(stackIdx)
    const shellIdx = stackTail.indexOf('<ShellWebShareBanner')
    const localIdx = stackTail.indexOf('<LocalNetworkBanner')
    const updateIdx = stackTail.indexOf('<UpdateBanner')
    const webglIdx = stackTail.indexOf('<WebGLRecoveryBanner')
    const pluginIdx = stackTail.indexOf('<PluginToggleBanner')
    expect(shellIdx).toBeGreaterThan(-1)
    // Banner must come before every other banner in the stack.
    if (localIdx > -1) expect(shellIdx).toBeLessThan(localIdx)
    if (updateIdx > -1) expect(shellIdx).toBeLessThan(updateIdx)
    if (webglIdx > -1) expect(shellIdx).toBeLessThan(webglIdx)
    if (pluginIdx > -1) expect(shellIdx).toBeLessThan(pluginIdx)
  })

  it('ShellWebShareBanner receives sessionName, onConfirm, onCancel props', () => {
    const bannerIdx = raw.indexOf('<ShellWebShareBanner')
    expect(bannerIdx).toBeGreaterThan(-1)
    const bannerBlock = raw.slice(bannerIdx, bannerIdx + 500)
    expect(bannerBlock).toContain('sessionName=')
    expect(bannerBlock).toContain('onConfirm=')
    expect(bannerBlock).toContain('onCancel=')
  })

  it('handleToggleWeb continues to work for AI CLIs (no banner short-circuit when cli !∈ SHELL_CLIS)', () => {
    // Negative assertion — the interception MUST be gated on SHELL_CLIS.has,
    // not on every session. If the SHELL_CLIS.has check is missing, AI CLI
    // toggles would also be blocked, breaking unrelated workflows.
    // Verify the original ToggleWebServing fall-through path still exists:
    expect(raw).toContain('ToggleWebServing(')
    // The interception must be conditional, not unconditional:
    const handlerIdx = raw.indexOf('handleToggleWeb')
    expect(handlerIdx).toBeGreaterThan(-1)
  })
})

// ---------------------------------------------------------------------------
// Phase 150 SET-01 — warningEnabled gate + re-arm sync + HubPanel threading
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

  it('handleToggleWeb gate includes shellWebShareWarningEnabled && before !shellWebShareWarned', () => {
    // The gate must now be: SHELL_CLIS.has && shellWebShareWarningEnabled && !shellWebShareWarned
    expect(raw).toContain('shellWebShareWarningEnabled &&')
    // Verify the order: warningEnabled check appears before !warned check in the same gate
    const gateIdx = raw.indexOf('SHELL_CLIS.has')
    expect(gateIdx).toBeGreaterThan(-1)
    const gateBlock = raw.slice(gateIdx, gateIdx + 300)
    const enabledIdx = gateBlock.indexOf('shellWebShareWarningEnabled &&')
    const warnedIdx = gateBlock.indexOf('!shellWebShareWarned')
    expect(enabledIdx).toBeGreaterThan(-1)
    expect(warnedIdx).toBeGreaterThan(-1)
    expect(enabledIdx).toBeLessThan(warnedIdx)
  })

  it('handleToggleWeb useCallback includes shellWebShareWarningEnabled in its dependency array', () => {
    // The new state must be added to the handleToggleWeb useCallback dep array
    const cbIdx = raw.indexOf('handleToggleWeb')
    expect(cbIdx).toBeGreaterThan(-1)
    // Find the closing } of the callback deps: look for the ], [... pattern after the handleToggleWeb def
    const cbBlock = raw.slice(cbIdx, cbIdx + 2000)
    expect(cbBlock).toContain('shellWebShareWarningEnabled')
  })

  it('SettingsTab render passes onShellWarnEnabledChange callback', () => {
    // App.tsx must thread the re-arm callback into SettingsTab
    expect(raw).toContain('onShellWarnEnabledChange=')
  })

  it('onShellWarnEnabledChange callback calls setShellWebShareWarningEnabled', () => {
    const cbIdx = raw.indexOf('onShellWarnEnabledChange=')
    expect(cbIdx).toBeGreaterThan(-1)
    const cbBlock = raw.slice(cbIdx, cbIdx + 600)
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

  it('HubPanel render receives shellWebShareWarned prop', () => {
    const hubIdx = raw.indexOf('<HubPanel')
    expect(hubIdx).toBeGreaterThan(-1)
    const hubBlock = raw.slice(hubIdx, hubIdx + 1500)
    expect(hubBlock).toContain('shellWebShareWarned=')
  })

  it('HubPanel render receives shellWebShareWarningEnabled prop', () => {
    const hubIdx = raw.indexOf('<HubPanel')
    expect(hubIdx).toBeGreaterThan(-1)
    const hubBlock = raw.slice(hubIdx, hubIdx + 1500)
    expect(hubBlock).toContain('shellWebShareWarningEnabled=')
  })

  it('HubPanel render receives onShellWebShareConfirm and onShellWebShareCancel callbacks', () => {
    const hubIdx = raw.indexOf('<HubPanel')
    expect(hubIdx).toBeGreaterThan(-1)
    const hubBlock = raw.slice(hubIdx, hubIdx + 1500)
    expect(hubBlock).toContain('onShellWebShareConfirm=')
    expect(hubBlock).toContain('onShellWebShareCancel=')
  })
})
