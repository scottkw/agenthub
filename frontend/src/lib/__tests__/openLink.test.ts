import { describe, it, expect, vi, beforeEach } from 'vitest';

// Phase 95 Plan 95-02 — Wave 1 GREEN (was: Plan 95-01 Task 2 RED scaffold).
// Plan 95-02 implements src/lib/openLink.ts. The literal options string
// '_blank', 'noopener,noreferrer' MUST be asserted character-for-character —
// `noopener` alone is insufficient (Pitfall #1: tab-isolation requires
// BOTH flags + the explicit `_blank` target).
//
// The helper has TWO call paths gated on `window.runtime?.BrowserOpenURL`:
//   - Desktop (Wails): hand off to Go via BrowserOpenURL.
//   - Web: window.open(url, '_blank', 'noopener,noreferrer').
//
// Defense-in-depth: openLink itself MUST re-validate the scheme so a buggy
// upstream caller cannot punch through the allowlist.

// vi.mock is hoisted to the top of the file by Vitest, so the mock applies
// before openLink imports BrowserOpenURL.
vi.mock('../../wailsjs/wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

import { openLink, isModifierPressed } from '../openLink';
import { BrowserOpenURL } from '../../wailsjs/wailsjs/runtime/runtime';

describe('openLink — Plan 95-02 (LNK-04)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset window.runtime / window.open between tests.
    (window as unknown as { runtime?: unknown }).runtime = undefined;
  });

  it('calls Wails BrowserOpenURL when window.runtime.BrowserOpenURL is present', () => {
    (window as unknown as { runtime: { BrowserOpenURL: () => void } }).runtime = {
      BrowserOpenURL: vi.fn(),
    };
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    openLink('https://example.com');
    expect(BrowserOpenURL).toHaveBeenCalledTimes(1);
    expect(BrowserOpenURL).toHaveBeenCalledWith('https://example.com');
    expect(openSpy).not.toHaveBeenCalled();
    openSpy.mockRestore();
  });

  it('calls window.open(url, "_blank", "noopener,noreferrer") when no Wails runtime', () => {
    // Acceptance grep target: the literal options string must appear in this file:
    // window.open(url, '_blank', 'noopener,noreferrer')
    (window as unknown as { runtime?: unknown }).runtime = undefined;
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    openLink('https://example.com');
    expect(openSpy).toHaveBeenCalledTimes(1);
    expect(openSpy).toHaveBeenCalledWith('https://example.com', '_blank', 'noopener,noreferrer');
    expect(BrowserOpenURL).not.toHaveBeenCalled();
    openSpy.mockRestore();
  });

  it('passes the literal third argument unchanged (no spaces, no extra flags)', () => {
    // Regression guard: a typo like 'noopen,noreferrer' silently degrades to
    // a leaky tab. Assert character-for-character match.
    (window as unknown as { runtime?: unknown }).runtime = undefined;
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    openLink('https://example.com');
    const call = openSpy.mock.calls[0];
    expect(call[1]).toBe('_blank');
    expect(call[2]).toBe('noopener,noreferrer');
    openSpy.mockRestore();
  });

  it('silently rejects non-allowlisted schemes (defense-in-depth)', () => {
    (window as unknown as { runtime: { BrowserOpenURL: () => void } }).runtime = {
      BrowserOpenURL: vi.fn(),
    };
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    openLink('javascript:alert(1)');
    expect(BrowserOpenURL).not.toHaveBeenCalled();
    expect(openSpy).not.toHaveBeenCalled();
    openSpy.mockRestore();
  });

  it('silently rejects file:// scheme (defense-in-depth)', () => {
    (window as unknown as { runtime?: unknown }).runtime = undefined;
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    openLink('file:///etc/passwd');
    expect(openSpy).not.toHaveBeenCalled();
    openSpy.mockRestore();
  });

  it('does not throw when window.runtime is undefined (web context guard)', () => {
    (window as unknown as { runtime?: unknown }).runtime = undefined;
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    expect(() => openLink('https://example.com')).not.toThrow();
    expect(openSpy).toHaveBeenCalledTimes(1);
    openSpy.mockRestore();
  });

  it('opens mailto: URLs on web via window.open with the same options', () => {
    (window as unknown as { runtime?: unknown }).runtime = undefined;
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
    openLink('mailto:user@example.com');
    expect(openSpy).toHaveBeenCalledWith('mailto:user@example.com', '_blank', 'noopener,noreferrer');
    openSpy.mockRestore();
  });
});

describe('isModifierPressed — Plan 95-02 (LNK-02)', () => {
  const setPlatform = (p: string) => {
    Object.defineProperty(navigator, 'platform', { value: p, configurable: true });
  };

  it("'none' mode always returns true regardless of modifier state", () => {
    expect(isModifierPressed({ metaKey: false, ctrlKey: false } as MouseEvent, 'none')).toBe(true);
    expect(isModifierPressed({ metaKey: true, ctrlKey: true } as MouseEvent, 'none')).toBe(true);
  });

  it("'platform' on macOS requires metaKey (Cmd)", () => {
    setPlatform('MacIntel');
    expect(isModifierPressed({ metaKey: true, ctrlKey: false } as MouseEvent, 'platform')).toBe(true);
    expect(isModifierPressed({ metaKey: false, ctrlKey: true } as MouseEvent, 'platform')).toBe(false);
  });

  it("'platform' on linux requires ctrlKey (Pitfall #3 — metaKey is Super on Linux)", () => {
    setPlatform('Linux x86_64');
    expect(isModifierPressed({ metaKey: false, ctrlKey: true } as MouseEvent, 'platform')).toBe(true);
    expect(isModifierPressed({ metaKey: true, ctrlKey: false } as MouseEvent, 'platform')).toBe(false);
  });

  it("'cmd' always checks metaKey regardless of platform", () => {
    setPlatform('Linux x86_64');
    expect(isModifierPressed({ metaKey: true } as MouseEvent, 'cmd')).toBe(true);
    expect(isModifierPressed({ metaKey: false, ctrlKey: true } as MouseEvent, 'cmd')).toBe(false);
  });

  it("'ctrl' always checks ctrlKey regardless of platform", () => {
    setPlatform('MacIntel');
    expect(isModifierPressed({ ctrlKey: true } as MouseEvent, 'ctrl')).toBe(true);
    expect(isModifierPressed({ metaKey: true, ctrlKey: false } as MouseEvent, 'ctrl')).toBe(false);
  });
});
