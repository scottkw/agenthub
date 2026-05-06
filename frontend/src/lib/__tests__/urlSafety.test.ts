import { describe, it, expect } from 'vitest';

// Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffold for src/lib/urlSafety.ts.
// Plan 95-02 implements the helpers (isAllowedScheme, hasIDN, osc8Mismatch,
// isTypoSquat, getRisk). Each `expect.fail` cites the implementing plan
// and the corresponding 95-VALIDATION row so the SUMMARY trace stays tight.
//
// Pitfall #2 (95-RESEARCH): the Cyrillic 'о' (U+043E) fixture below MUST
// survive file I/O without being normalized to Latin 'o' (U+006F). The
// metatest under `hasIDN` is the fixture-integrity check and PASSES on
// Wave 0 (it is not a feature test).

describe('urlSafety — Plan 95-02', () => {
  describe('isAllowedScheme (LNK-01)', () => {
    it('rejects javascript: scheme', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts isAllowedScheme (95-VALIDATION row 95-01-01).');
    });
    it('rejects file:// scheme', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts isAllowedScheme.');
    });
    it('rejects data: scheme', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts isAllowedScheme.');
    });
    it('allows https://, http://, mailto:', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts isAllowedScheme (95-VALIDATION row 95-01-02).');
    });
  });

  describe('hasIDN (LNK-03)', () => {
    // Cyrillic 'о' is U+043E. The string below MUST contain U+043E codepoints,
    // not Latin 'o' (U+006F). The metatest below proves the fixture survived
    // file I/O without normalization (95-RESEARCH §"Pitfall 2: Cyrillic
    // Spoof Test Fixture Encoding").
    const CYRILLIC_SPOOF = 'https://gооgle.com';
    it('fixture preserves Cyrillic codepoints (metatest — Pitfall #2)', () => {
      const host = CYRILLIC_SPOOF.match(/^https?:\/\/([^/]+)/)![1];
      const hasNonAscii = [...host].some((c) => c.charCodeAt(0) > 127);
      expect(hasNonAscii, `fixture host '${host}' lost its Cyrillic codepoints`).toBe(true);
    });
    it('Cyrillic spoof gооgle.com triggers hasIDN', () => {
      // The spoof URL passes through to the helper at GREEN time.
      void CYRILLIC_SPOOF;
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts hasIDN (95-VALIDATION row 95-04-01).');
    });
    it('Punycoded xn--google-jzd.com triggers hasIDN', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts hasIDN.');
    });
  });

  describe('osc8Mismatch (LNK-03)', () => {
    it('display "click here" + href "https://evil.example" returns true', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts osc8Mismatch (95-VALIDATION row 95-03-02).');
    });
    it('matching display + href returns false', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts osc8Mismatch.');
    });
  });

  describe('isTypoSquat (LNK-03)', () => {
    it('paypa1.com matches', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts isTypoSquat (95-VALIDATION row 95-04-02).');
    });
    it('paypal.com does NOT match', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts isTypoSquat.');
    });
  });

  describe('getRisk (LNK-03)', () => {
    it('osc8 mismatch beats idn beats typosquat (priority order)', () => {
      expect.fail('RED scaffold — Plan 95-02 implements src/lib/urlSafety.ts getRisk priority order.');
    });
  });
});
