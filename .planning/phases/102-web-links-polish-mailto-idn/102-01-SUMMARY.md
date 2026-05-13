---
phase: 102-web-links-polish-mailto-idn
plan: 01
status: complete
requirements_satisfied: [POLISH-01, POLISH-02]
---

# Plan 102-01 SUMMARY — Web-Links Polish (mailto + IDN)

## What was built

**POLISH-01 (mailto)**: WebLinksAddon (xterm.js) now receives an explicit
`urlRegex` option that matches `http(s)` AND `mailto:` URLs. Without this,
the addon's default regex only matches http(s), and `mailto:` URLs were never
detected as clickable in terminal output.

Cmd/Ctrl-clicking `mailto:user@example.com` now:
1. WebLinksAddon detects the link via the new regex
2. Handler validates scheme through `isAllowedScheme` (already allowlists mailto)
3. Modifier-click gate fires
4. `getRisk` checks for IDN/typosquat (mailto IDN is handled inside `hasIDN`)
5. `openLink` routes to the system mail client (`window.open` on web,
   `BrowserOpenURL` on desktop — both already mailto-aware)

**POLISH-02 (IDN)**: Already implemented at the lib level prior to this phase.
`urlSafety.hasIDN()` detects non-ASCII codepoints AND `xn--` forms in
hostnames (and for mailto:, the domain after `@`). The existing handler wires
`getRisk → 'idn' → LinkConfirmPopover` so any Cyrillic-homograph URL routes
through the confirmation popover with Punycode form alongside display form.

The phase 102 contribution here is the regex change that lets the click
handler fire for mailto URLs in the first place — IDN-in-mailto now also
triggers the popover because the chain runs.

## Files modified

- `frontend/src/components/TerminalPanel.tsx` — added `urlRegex` to
  WebLinksAddon construction with POLISH-01/02 rationale comments.
- `web/assets/terminal.js` — mirrored the regex for web parity (Phase 95's
  desktop+web parity mandate).
- `frontend/src/components/__tests__/TerminalPanel.web-links.test.tsx` —
  added 3 RED→GREEN tests:
  - urlRegex option present
  - regex matches mailto + http(s), rejects javascript:
  - POLISH-02 sanity (getRisk + confirmIDN still wired)

## Verification

- vitest: **851 / 851 passing** (3 new — full sweep clean)
- `pnpm tsc --noEmit`: clean
- No regressions in TerminalPanel.web-links.test.tsx (28 / 28 passing)

## Notes / pitfalls

- The regex is permissive (matches `http(s)://...` and `mailto:user@host.tld`).
  The handler-level `isAllowedScheme` re-check is the security boundary, not
  the regex (defense-in-depth — LNK-01 pattern from Phase 95).
- The mailto regex requires `@` and a TLD; bare `mailto:` with no address
  won't match. This is intentional — bare `mailto:` has no useful destination.
- Manual UAT (real terminal output + system mail handler) should land in a
  v3.3 UAT pass; jsdom can't exercise xterm.js canvas rendering, so all tests
  here are source-grep + regex-behavior level.

## Deviations

None. Implementation was inline (no executor agent) due to autonomous-mode
budget constraints — Phase 102 scope is small (~10 lines of code change)
and the plan was authored together with the implementation, both atomic.
