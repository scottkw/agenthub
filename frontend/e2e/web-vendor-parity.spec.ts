// Phase 93 Plan 05 Task 1 — web-vendor-parity.spec.ts
//
// Asserts the web terminal page loads webgl/unicode11/clipboard addons from
// the same-origin /assets/xterm/addons/ path AND that no CDN host is reached
// during a full attach session (must_haves.truths zero-CDN claim).
//
// Pinned URLs:
//   /assets/xterm/addons/addon-webgl.js
//   /assets/xterm/addons/addon-unicode11.js
//   /assets/xterm/addons/addon-clipboard.js
//
// CDN hosts checked:
//   cdn.jsdelivr.net, unpkg.com, esm.sh, cdnjs.cloudflare.com

import { test, expect } from '@playwright/test'
import { sessionURL } from './fixture-env'

test.describe('Phase 93 WEB-01 web-vendor-parity', () => {
  test('loads webgl/unicode11/clipboard addons from /assets/xterm/addons/ same-origin', async ({ page }) => {
    const cdnHosts = ['cdn.jsdelivr.net', 'unpkg.com', 'esm.sh', 'cdnjs.cloudflare.com']
    const cdnHits: string[] = []
    page.on('request', (req) => {
      try {
        const url = new URL(req.url())
        if (cdnHosts.some((h) => url.hostname.endsWith(h))) cdnHits.push(req.url())
      } catch {
        // ignore non-URL request objects
      }
    })

    const webglResp = page.waitForResponse(
      (r) => r.url().includes('/assets/xterm/addons/addon-webgl.js') && r.status() === 200,
      { timeout: 15_000 }
    )
    const u11Resp = page.waitForResponse(
      (r) => r.url().includes('/assets/xterm/addons/addon-unicode11.js') && r.status() === 200,
      { timeout: 15_000 }
    )
    const clipResp = page.waitForResponse(
      (r) => r.url().includes('/assets/xterm/addons/addon-clipboard.js') && r.status() === 200,
      { timeout: 15_000 }
    )

    await page.goto(sessionURL())
    await Promise.all([webglResp, u11Resp, clipResp])

    // Exercise the page so any late vendor pulls would fire.
    await page.waitForTimeout(2_000)

    expect(cdnHits, 'No CDN requests during web terminal session').toEqual([])
  })
})
