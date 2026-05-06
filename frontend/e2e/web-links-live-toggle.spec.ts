// Phase 95 Plan 95-06 — web parity live-toggle e2e (LNK-05/SC-5).
//
// Status: documented `test.skip` — the live-toggle path requires a running
// agenthub instance with a real session, a Tailscale-served terminal page,
// and a daemon RPC fixture for SetWebLinksConfig (Plan 95-05). The
// chromedp-based e2e harness used for Phase 94 (findbar_web_e2e_test.go in
// internal/webserver) covers the deterministic source-inspection path; the
// runtime click-pipeline path is verified manually per
// .planning/phases/95-web-links-addon-security-hardening/95-WEB-UAT.md
// and 95-DESKTOP-UAT.md until Playwright is plumbed for this repo's web
// surface.
//
// When Playwright is wired (planned: post-v3.2 once the e2e harness adds
// a /sessions fixture + daemon RPC test stub), the body below describes the
// exact walk a human/agent should perform and exactly which assertions it
// must make. Keeping this here (vs. a doc) means a future engineer running
// `pnpm exec playwright test` sees the file and the documented body, not
// silence.

import { test } from '@playwright/test';

test.describe('Phase 95 — web-links live toggle (LNK-05/SC-5)', () => {
  test.skip(
    'toggle webLinks=false disposes addon; toggling back on reattaches; window.open is called with _blank+noopener,noreferrer',
    async () => {
      // 1. Navigate to a Tailscale-served session URL with webLinks=true.
      // 2. Echo `https://example.com\r` into the terminal via term.write().
      // 3. Spy on window.open via page.exposeFunction OR page.on('popup').
      // 4. Cmd-click (mac CI) / Ctrl-click (linux/win CI) on the rendered URL.
      // 5. Assert spy fired with exactly (url, '_blank', 'noopener,noreferrer').
      // 6. POST /api/plugin-config { webLinks: false } via test fixture
      //    (Plan 95-05 SetWebLinksConfig path) OR toggle via Settings UI in
      //    a desktop window paired to this session.
      // 7. Wait for SSE settings:plugins event (poll DOM for absence of the
      //    addon's underline class on the next URL hover, < 2s).
      // 8. Hover URL — assert NO clickable underline / addon link decoration.
      // 9. Toggle webLinks=true; verify clickable URL returns; NO PAGE RELOAD.
      throw new Error('test.skip — see test body for the documented walk');
    }
  );

  test.skip(
    'Cyrillic spoof URL triggers popover before window.open; Continue dispatches; Cancel dismisses',
    async () => {
      // 1. With webLinks=true, echo `https://gооgle.com\r` (the two `о` chars
      //    are Cyrillic U+043E — verify codepoints survived file I/O via
      //    .charCodeAt() in a fixture loader).
      // 2. Cmd-click / Ctrl-click on the URL. Assert that
      //    document.getElementById('link-confirm-popover').hidden === false
      //    AND #link-confirm-url textContent === the Cyrillic URL (NOT
      //    Punycode-normalized).
      // 3. Click #link-confirm-cancel. Assert popover.hidden === true; assert
      //    window.open spy was NOT called.
      // 4. Cmd-click again. Click #link-confirm-continue. Assert popover hides
      //    AND window.open spy fires with (url, '_blank', 'noopener,noreferrer').
      throw new Error('test.skip — see test body for the documented walk');
    }
  );

  test.skip(
    'typosquat URL triggers popover; non-modifier-click is suppressed; scheme allowlist blocks javascript:',
    async () => {
      // 1. Echo `https://paypa1.com\r`. Cmd-click → popover with typosquat copy.
      // 2. Echo `https://example.com\r`. SINGLE-click (no modifier) →
      //    nothing happens (no popover, no window.open call).
      // 3. Echo `javascript:alert(1)\r`. Hover → addon does NOT decorate as
      //    a clickable link (default urlRegex blocks at the regex layer;
      //    isAllowedScheme blocks at the handler layer — defense in depth).
      // 4. Echo `mailto:test@example.com\r`. Cmd-click → openLink dispatches
      //    (mailto is in ALLOWED_SCHEMES).
      throw new Error('test.skip — see test body for the documented walk');
    }
  );
});
