import { describe, it, expect } from 'vitest';
import {
  isAllowedScheme,
  hasIDN,
  osc8Mismatch,
  isTypoSquat,
  getRisk,
  ALLOWED_SCHEMES,
} from '../urlSafety';

// Phase 95 Plan 95-02 — Wave 1 GREEN (was: Plan 95-01 Task 2 RED scaffold).
// Implements the helpers (isAllowedScheme, hasIDN, osc8Mismatch,
// isTypoSquat, getRisk).
//
// Pitfall #2 (95-RESEARCH): the Cyrillic 'о' (U+043E) fixture below MUST
// survive file I/O without being normalized to Latin 'o' (U+006F). The
// metatest under `hasIDN` is the fixture-integrity check and PASSES on
// Wave 0; it MUST remain GREEN at Wave 1.

// Cyrillic 'о' is U+043E. The string below MUST contain U+043E codepoints,
// not Latin 'o' (U+006F).
const CYRILLIC_SPOOF = 'https://gооgle.com';

describe('urlSafety — Plan 95-02', () => {
  describe('isAllowedScheme (LNK-01)', () => {
    it('rejects javascript: scheme', () => {
      expect(isAllowedScheme('javascript:alert(1)')).toBe(false);
    });
    it('rejects file:// scheme', () => {
      expect(isAllowedScheme('file:///etc/passwd')).toBe(false);
    });
    it('rejects data: scheme', () => {
      expect(isAllowedScheme('data:text/html,<script>')).toBe(false);
    });
    it('allows https://, http://, mailto:', () => {
      expect(isAllowedScheme('https://example.com')).toBe(true);
      expect(isAllowedScheme('http://example.com')).toBe(true);
      expect(isAllowedScheme('mailto:user@example.com')).toBe(true);
    });
    it('rejects unparseable input (URL constructor throws → caught → false)', () => {
      expect(isAllowedScheme('not-a-url-at-all')).toBe(false);
    });
    it('exports a frozen-shape allowlist with exactly https/http/mailto', () => {
      expect([...ALLOWED_SCHEMES]).toEqual(['https:', 'http:', 'mailto:']);
    });
  });

  describe('hasIDN (LNK-03)', () => {
    it('fixture preserves Cyrillic codepoints (metatest — Pitfall #2)', () => {
      const host = CYRILLIC_SPOOF.match(/^https?:\/\/([^/]+)/)![1];
      const hasNonAscii = [...host].some((c) => c.charCodeAt(0) > 127);
      expect(hasNonAscii, `fixture host '${host}' lost its Cyrillic codepoints`).toBe(true);
    });
    it('Cyrillic spoof gооgle.com triggers hasIDN', () => {
      expect(hasIDN(CYRILLIC_SPOOF)).toBe(true);
    });
    it('Punycoded xn--google-jzd.com triggers hasIDN', () => {
      expect(hasIDN('https://xn--google-jzd.com')).toBe(true);
    });
    it('plain ASCII host returns false', () => {
      expect(hasIDN('https://example.com')).toBe(false);
    });
  });

  describe('osc8Mismatch (LNK-03)', () => {
    it('display "click here" + href "https://evil.example" returns true', () => {
      expect(osc8Mismatch('click here', 'https://evil.example')).toBe(true);
    });
    it('matching display + href returns false', () => {
      expect(osc8Mismatch('https://github.com', 'https://github.com')).toBe(false);
    });
    it('different host (typo) returns true', () => {
      expect(osc8Mismatch('https://github.com', 'https://gitub.com')).toBe(true);
    });
  });

  describe('isTypoSquat (LNK-03)', () => {
    it('paypa1.com matches', () => {
      expect(isTypoSquat('https://paypa1.com')).toBe(true);
    });
    it('www.paypa1.com matches (www. stripped)', () => {
      expect(isTypoSquat('https://www.paypa1.com')).toBe(true);
    });
    it('paypal.com does NOT match', () => {
      expect(isTypoSquat('https://paypal.com')).toBe(false);
    });
  });

  describe('getRisk (LNK-03)', () => {
    it('priority osc8 > idn > typosquat — osc8 mismatch with Cyrillic href returns "osc8"', () => {
      // Cyrillic-host URL has BOTH osc8 mismatch (display=plain prose) AND idn.
      // osc8 wins.
      expect(getRisk('click here', CYRILLIC_SPOOF)).toBe('osc8');
    });
    it('priority idn > typosquat — Cyrillic host with display === href returns "idn"', () => {
      expect(getRisk(CYRILLIC_SPOOF, CYRILLIC_SPOOF)).toBe('idn');
    });
    it('priority typosquat — paypa1.com with display === href returns "typosquat"', () => {
      expect(getRisk('https://paypa1.com', 'https://paypa1.com')).toBe('typosquat');
    });
    it('returns null when no risk detected', () => {
      expect(getRisk('https://example.com', 'https://example.com')).toBe(null);
    });
  });
});
