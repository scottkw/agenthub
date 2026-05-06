import { describe, it, expect } from 'vitest';

// Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffold for TerminalPanel
// integration with @xterm/addon-web-links. Plan 95-04 implements:
// (1) WebLinksAddon import + ref + hot-swap useEffect arm
// (2) custom click handler (NOT the addon default — see Pitfall #1)
// (3) modifier-click gate (Cmd on darwin / Ctrl elsewhere)
// (4) hot-swap on pluginConfig.webLinks toggle.
//
// Source-inspection pattern (mirrors Phase 94 FindBar.focus.test.tsx):
// most assertions read TerminalPanel.tsx?raw and regex-match for the
// expected wiring. Behavioural tests (single-click vs modifier-click)
// will be added in Plan 95-04 once the click-handler wiring lands.
//
// Why source-inspection? The hot-swap useEffect mounts xterm.js into a
// real DOM node and is awkward to drive in jsdom — Plan 92/93/94 all
// landed their addon-related tests as raw-text regex checks against
// the TSX. The popover (LinkConfirmPopover) gets the RTL-style tests
// in its own file (file 4 of this Wave 0 scaffold set).

describe('TerminalPanel web-links integration — Plan 95-04', () => {
  it('TerminalPanel.tsx imports WebLinksAddon from @xterm/addon-web-links', () => {
    expect.fail('RED scaffold — Plan 95-04 implements WebLinksAddon import + ref + hot-swap useEffect arm.');
  });
  it('WebLinksAddon constructor receives an explicit handler (not default)', () => {
    // Source-inspection: regex on TerminalPanel.tsx for `new WebLinksAddon(`
    // followed by a non-undefined first argument. RED until 95-04 lands.
    expect.fail('RED scaffold — Plan 95-04 implements custom handler (LNK-01 defense-in-depth; 95-VALIDATION row 95-01).');
  });
  it('hot-swap useEffect dep array includes pluginConfig?.webLinks', () => {
    // Plan 95-04 extends the hot-swap dep array so a daemon-driven toggle
    // disposes / reattaches the addon without a session restart.
    expect.fail('RED scaffold — Plan 95-04 extends hot-swap dep array.');
  });
  it('single-click without modifier does NOT call openLink', () => {
    expect.fail('RED scaffold — Plan 95-04 implements modifier-click gate (95-VALIDATION row 95-02-01).');
  });
  it('Cmd-click on darwin / Ctrl-click on linux+win calls openLink', () => {
    expect.fail('RED scaffold — Plan 95-04 implements isModifierPressed (95-VALIDATION row 95-02-02).');
  });
  it('modifier="none" config bypasses the click gate (still gated by allowlist)', () => {
    // 95-RESEARCH §"Pitfall 9: Modifier Configuration 'none'": even with
    // the modifier requirement disabled, the scheme allowlist + risk
    // detector still gate the actual openLink call.
    expect.fail('RED scaffold — Plan 95-04 honors webLinksConfig.modifier="none".');
  });
  it('hot-swap: pluginConfig.webLinks=false disposes addon; =true reattaches', () => {
    expect.fail('RED scaffold — Plan 95-04 implements hot-swap dep array extension (95-VALIDATION row 95-06-02).');
  });
});
