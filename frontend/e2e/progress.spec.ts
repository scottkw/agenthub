// Phase 98 Plan 98-05 — progress addon web parity e2e (PRG-02 / PRG-03).
//
// Status: documented `test.skip` — the progress e2e path requires a running
// agenthub instance with a real session, OSC 9;4 write fixture, and daemon
// RPC fixture for SetTrayProgress (Plan 98-02). The chromedp-based e2e harness
// used for Phase 94 (findbar_web_e2e_test.go in internal/webserver) covers the
// deterministic source-inspection path; the runtime OSC 9;4 pipeline is
// verified manually per
// .planning/phases/98-progress-addon-p2-cuttable/98-HUMAN-UAT.md
// until Playwright is plumbed for this repo's web surface.
//
// When Playwright is wired (planned: post-v3.2 once the e2e harness adds
// a /sessions fixture + daemon RPC test stub), the bodies below describe the
// exact walk a human/agent should perform and exactly which assertions it
// must make. Keeping this here (vs. a doc) means a future engineer running
// `pnpm exec playwright test` sees the file and the documented body, not
// silence.

import { test } from '@playwright/test';

test.describe('Phase 98 — progress addon (PRG-02 / per-tab + web underline)', () => {
  test.skip(
    'OSC 9;4 sequence drives .tab__progress transform to scaleX(0.47)',
    async () => {
      // Phase 98 PRG-02 — desktop e2e walk-through (parked at test.skip until
      // Phase 99 release-gate Playwright plumbing lands; manual UAT covers
      // this scenario in the meantime via 98-HUMAN-UAT.md scenario 1).
      //
      // 1. Launch the desktop app via `wails dev` or the built binary.
      //    Settings → Plugins → toggle Progress ON; Save.
      // 2. Open a fresh terminal tab; capture its sessionId.
      // 3. In the terminal, run: bash tests/fixtures/osc94-progress-fixture.sh
      //    OR: page.evaluate(() => term.write('\x1b]9;4;1;47\x07'))
      // 4. Assert the .tab__progress element for that sessionId has
      //    style.transform matching /scaleX\(0\.47\)/.
      // 5. Emit clear: '\x1b]9;4;0\x07' — assert transform reverts to scaleX(0).
      throw new Error('test.skip — see test body for the documented walk');
    }
  );

  test.skip(
    'web-served session: #progress-underline transform scales from OSC 9;4 events',
    async () => {
      // Phase 98 PRG-02 web-half — web-served Tailscale session walk-through.
      //
      // 1. Start the daemon with web serving enabled; obtain a Tailscale
      //    session URL (see Phase 89 patterns for url construction).
      // 2. Navigate Playwright to the URL with capability token.
      // 3. Inject OSC 9;4 sequence via the WebSocket relay or via term.write.
      // 4. Assert document.getElementById('progress-underline').style.transform
      //    matches /scaleX\(0\.5\)/ for value=50.
      // 5. Toggle pluginConfig.progress=false via /api/plugin-config (or
      //    equivalent settings save); assert addon disposes; transform=scaleX(0).
      throw new Error('test.skip — see test body for the documented walk');
    }
  );

  test.skip(
    'cross-session aggregate triggers tray quartile RPC at 200ms debounce',
    async () => {
      // Phase 98 PRG-03 — tray glyph e2e is desktop-only and not directly
      // observable from Playwright (system tray is OS-level). Manual UAT
      // covers this scenario in 98-HUMAN-UAT.md scenario 2.
      //
      // Scaffold left here for future work: a mock SetTrayProgress can
      // be injected via Wails dev-mode binding and asserted on dispatch
      // counts (only quartile transitions cross the bridge after debounce).
      throw new Error('test.skip — manual UAT covers PRG-03 in scenario 2');
    }
  );
});
