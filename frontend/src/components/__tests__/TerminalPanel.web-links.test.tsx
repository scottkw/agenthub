/**
 * Phase 95 Plan 95-04 — TerminalPanel WebLinksAddon integration.
 *
 * Source-inspection invariants (mirrors the established Phase 93/94 pattern —
 * see TerminalPanel.hot-swap.test.tsx and TerminalPanel.search.test.tsx).
 * Runtime mounting of xterm requires <canvas> + WebGL which jsdom does not
 * provide; source-inspection is the deterministic gate. Manual UAT (Cmd-click
 * on https / cyrillic / typosquat URL) lives in 95-DESKTOP-UAT.md, finalized
 * in Plan 95-06.
 *
 * Wave 0 RED scaffolds (Plan 95-01) are flipped GREEN here. The plan's
 * <action> Step C suggested @testing-library/react render harness, but the
 * project does not depend on @testing-library/react and every existing
 * TerminalPanel test (including Phase 94 SearchAddon) uses raw-text regex
 * against `../TerminalPanel.tsx?raw`. Plan 95-03 documented the same
 * convention deviation.
 *
 * Plan B (Wave 0 spike outcome): OSC 8 mismatch detection is deferred to
 * v3.3 because IBufferCell.getHyperlinkId is absent from @xterm/xterm@6.0.0
 * public typings. v3.2 ships IDN + typosquat detectors only; the popover's
 * 'osc8' branch ships dormant in LinkConfirmPopover so the v3.3 wiring slice
 * can land without re-touching presentation.
 */
import { describe, it, expect } from 'vitest'
import src from '../TerminalPanel.tsx?raw'

describe('Phase 95 LNK-01..04 — TerminalPanel WebLinksAddon integration', () => {
  it('TerminalPanel.tsx imports WebLinksAddon from @xterm/addon-web-links', () => {
    expect(src).toMatch(/import\s+\{\s*WebLinksAddon\s*\}\s+from\s+['"]@xterm\/addon-web-links['"]/)
  })

  it('imports isAllowedScheme + getRisk + RiskKind from ../lib/urlSafety', () => {
    expect(src).toMatch(/import\s+\{[^}]*\bisAllowedScheme\b[^}]*\bgetRisk\b[^}]*\}\s+from\s+['"]\.\.\/lib\/urlSafety['"]/)
  })

  it('imports openLink + isModifierPressed from ../lib/openLink', () => {
    expect(src).toMatch(/import\s+\{[^}]*\bopenLink\b[^}]*\bisModifierPressed\b[^}]*\}\s+from\s+['"]\.\.\/lib\/openLink['"]/)
  })

  it('imports LinkConfirmPopover from ./LinkConfirmPopover', () => {
    expect(src).toMatch(/import\s+\{\s*LinkConfirmPopover\s*\}\s+from\s+['"]\.\/LinkConfirmPopover['"]/)
  })

  it('declares webLinksAddonRef (≥4 references — declaration, attach, dispose, cleanup)', () => {
    const matches = src.match(/webLinksAddonRef/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(4)
  })

  it('declares webLinksConfigRef for sub-config read-at-click-time (Pitfall #8)', () => {
    expect(src).toContain('webLinksConfigRef')
  })

  it('declares linkConfirmState React state for popover render gating', () => {
    expect(src).toMatch(/setLinkConfirmState/)
    expect(src).toMatch(/linkConfirmState/)
  })

  it('WebLinksAddon constructor receives an explicit handler (not bare default — Pitfall #1)', () => {
    // Pitfall #1: bare `new WebLinksAddon()` would use upstream default which
    // calls window.open() then n.location.href = t (worse than _blank, no
    // noopener,noreferrer). MUST construct with explicit handler argument.
    expect(src).toMatch(/new\s+WebLinksAddon\s*\(\s*\w+/)
    expect(src).not.toMatch(/new\s+WebLinksAddon\s*\(\s*\)/)
  })

  it('handler enforces scheme allowlist (LNK-01 defense-in-depth — calls isAllowedScheme(uri))', () => {
    expect(src).toContain('isAllowedScheme(uri)')
  })

  it('handler enforces modifier-click gate (LNK-02 — calls isModifierPressed(event, ...))', () => {
    expect(src).toMatch(/isModifierPressed\(event,/)
  })

  it('handler reads sub-config via webLinksConfigRef.current (Pitfall #8 — sub-config NOT a useEffect dep)', () => {
    expect(src).toContain('webLinksConfigRef.current')
  })

  it('handler runs risk detection via getRisk (LNK-03)', () => {
    expect(src).toMatch(/getRisk\(/)
  })

  it('handler routes through openLink for non-risky URLs (LNK-04)', () => {
    expect(src).toContain('openLink(uri)')
  })

  it('hover callback sets the link DOM element title attribute (LNK-02 hover tooltip)', () => {
    expect(src).toContain("setAttribute('title'")
  })

  it('leave callback removes the title attribute (Pitfall #10 — both hover AND leave required)', () => {
    expect(src).toContain("removeAttribute('title')")
  })

  it('hot-swap useEffect dep array includes pluginConfig?.webLinks (boolean — main toggle)', () => {
    expect(src).toMatch(/\[\s*pluginConfig\?\.webgl\s*,\s*pluginConfig\?\.clipboard\s*,\s*pluginConfig\?\.search\s*,\s*pluginConfig\?\.webLinks/)
  })

  it('hot-swap useEffect dep array does NOT include pluginConfig?.webLinksConfig (sub-config flows via ref — Pitfall #8)', () => {
    // Extract the hot-swap useEffect dep array (the one that contains
    // pluginConfig?.webgl). Plan 95-04: webLinksConfig MUST be excluded.
    const match = src.match(/\[\s*pluginConfig\?\.webgl[^\]]*\]/)
    expect(match).not.toBeNull()
    expect(match![0]).not.toContain('pluginConfig?.webLinksConfig')
  })

  it('cleanup useEffect disposes webLinksAddonRef.current on unmount', () => {
    expect(src).toContain('webLinksAddonRef.current.dispose()')
  })

  it('hot-swap useEffect disposes the addon when pluginConfig?.webLinks toggles to false', () => {
    // Both the unmount cleanup AND the hot-swap toggle-off path call dispose.
    // Count must be ≥ 2 (one for unmount, one for toggle-off).
    const matches = src.match(/webLinksAddonRef\.current\.dispose\(\)/g) ?? []
    expect(matches.length).toBeGreaterThanOrEqual(2)
  })

  it('renders <LinkConfirmPopover> conditionally when linkConfirmState !== null', () => {
    expect(src).toMatch(/linkConfirmState\s*&&\s*\(?\s*<LinkConfirmPopover/)
  })

  it('LinkConfirmPopover Continue handler invokes openLink + clears linkConfirmState', () => {
    expect(src).toContain('openLink(linkConfirmState.url)')
    expect(src).toContain('setLinkConfirmState(null)')
  })

  it('Plan B (Wave 0 spike outcome): no OSC 8 secondary registerLinkProvider call', () => {
    // Plan B selected per 95-RESEARCH §"Wave 0 Spike Outcome": OSC 8 mismatch
    // detection deferred to v3.3 because IBufferCell.getHyperlinkId absent
    // from @xterm/xterm@6.0.0 public typings. The osc8 branch in
    // LinkConfirmPopover ships dormant; TerminalPanel does NOT register a
    // secondary provider for OSC 8 walking.
    expect(src).not.toContain('registerLinkProvider')
    expect(src).not.toContain('getHyperlinkId')
  })

  it('handler honors confirmIDN flag from webLinksConfig (LNK-03)', () => {
    expect(src).toContain('confirmIDN')
  })

  it('handler honors confirmTyposquat flag from webLinksConfig (LNK-03)', () => {
    expect(src).toContain('confirmTyposquat')
  })

  it('no telemetry / network calls in handler body (T-95-04-07 mitigation)', () => {
    // Source-level invariant — the handler MUST NOT log or fetch URLs.
    // (FindBar.search persistence already uses SetSearchConfig; no new
    // analytics surface should land in the click path.)
    expect(src).not.toMatch(/console\.log\([^)]*uri/)
    expect(src).not.toMatch(/fetch\([^)]*uri/)
  })
})
