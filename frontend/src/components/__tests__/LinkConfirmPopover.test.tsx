import { describe, it, expect } from 'vitest';

// Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffold for the
// LinkConfirmPopover component. Plan 95-03 implements the popover.
//
// XSS gate (95-RESEARCH §"Anti-Patterns"): the popover MUST render
// untrusted URL display text via React text-rendering (textContent),
// NEVER innerHTML / dangerouslySetInnerHTML.
//
// Risk types align with src/lib/urlSafety.ts getRisk return values:
//   - 'osc8'      — OSC 8 display-vs-href divergence (Plan B: scaffold
//                   only; popover surface exists but the live wiring
//                   ships in v3.3 — see 95-RESEARCH §"Wave 0 Spike Outcome")
//   - 'idn'       — non-ASCII codepoint(s) detected in hostname
//   - 'typosquat' — host within edit-distance 1 of a known brand
//
// Each risk gets bespoke copy explaining the specific risk to the user;
// generic "this might be unsafe" copy fails the verifier.

describe('LinkConfirmPopover — Plan 95-03', () => {
  it('renders risk-specific copy for risk="osc8"', () => {
    expect.fail('RED scaffold — Plan 95-03 implements src/components/LinkConfirmPopover.tsx (95-VALIDATION row 95-04-03).');
  });
  it('renders risk-specific copy for risk="idn"', () => {
    expect.fail('RED scaffold — Plan 95-03 implements LinkConfirmPopover risk copy.');
  });
  it('renders risk-specific copy for risk="typosquat"', () => {
    expect.fail('RED scaffold — Plan 95-03 implements LinkConfirmPopover risk copy.');
  });
  it('Continue button calls onContinue', () => {
    expect.fail('RED scaffold — Plan 95-03 implements LinkConfirmPopover button wiring.');
  });
  it('Cancel button calls onCancel', () => {
    expect.fail('RED scaffold — Plan 95-03 implements LinkConfirmPopover button wiring.');
  });
  it('URL is rendered via textContent (NEVER innerHTML)', () => {
    expect.fail('RED scaffold — Plan 95-03 enforces React text-rendering for untrusted display text (95-RESEARCH §"Anti-Patterns").');
  });
  it('focus is trapped inside the popover while open (a11y)', () => {
    expect.fail('RED scaffold — Plan 95-03 implements popover focus trap.');
  });
  it('Escape key calls onCancel (parity with Cancel button)', () => {
    expect.fail('RED scaffold — Plan 95-03 implements Escape-key handler.');
  });
});
