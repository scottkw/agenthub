// Phase 93 Plan 05 Task 1 — web-vendor-parity.spec.ts
//
// Asserts the web terminal loads its xterm engine + addons same-origin with
// ZERO CDN requests during a full attach session (must_haves.truths zero-CDN
// supply-chain claim).
//
// Phase 159 (WEBCHAT-01) retired the raw /sessions/{id} terminal.js viewer —
// every share link now 302-redirects to the /app/ React SPA, which is the
// surface remote guests actually reach. The SPA Vite-bundles the xterm addons
// (ESM imports) into its own same-origin chunks rather than loading them as
// separate /assets/xterm/addons/*.js scripts, so the assertion adapts from
// "pinned addon URLs respond 200" to "the bundle is same-origin + zero CDN +
// no cross-origin script/style" — the same supply-chain guarantee on the live
// surface.
//
// CDN hosts checked:
//   cdn.jsdelivr.net, unpkg.com, esm.sh, cdnjs.cloudflare.com

import { test, expect } from '@playwright/test'
import { appUrl, loadFixtureEnv } from './fixture-env'

test.describe('Phase 93 WEB-01 web-vendor-parity', () => {
  test('/app/ loads the terminal + addons same-origin with zero CDN requests', async ({ page }) => {
    const env = loadFixtureEnv()
    const appOrigin = new URL(env.baseURL).host

    const cdnHosts = ['cdn.jsdelivr.net', 'unpkg.com', 'esm.sh', 'cdnjs.cloudflare.com']
    const cdnHits: string[] = []
    const crossOriginScripts: string[] = []
    let sameOriginScriptLoaded = false

    page.on('request', (req) => {
      try {
        const url = new URL(req.url())
        if (cdnHosts.some((h) => url.host.endsWith(h))) cdnHits.push(req.url())

        const type = req.resourceType()
        if (type === 'script' || type === 'stylesheet') {
          if ((url.protocol === 'http:' || url.protocol === 'https:') && url.host !== appOrigin) {
            crossOriginScripts.push(req.url())
          }
          if (url.host === appOrigin) sameOriginScriptLoaded = true
        }
      } catch {
        // ignore non-URL request objects
      }
    })

    await page.goto(appUrl(env))
    // The terminal must render — proves the bundled xterm engine + addons loaded
    // and a session attached, so the zero-CDN assertion is meaningful (not vacuous).
    await expect(page.locator('.xterm').first()).toBeVisible({ timeout: 15_000 })

    // Exercise the page so any late vendor pulls would fire.
    await page.waitForTimeout(2_000)

    expect(sameOriginScriptLoaded, 'the /app/ SPA bundle must load same-origin').toBeTruthy()
    expect(crossOriginScripts, 'no cross-origin script/style during web terminal session').toEqual([])
    expect(cdnHits, 'No CDN requests during web terminal session').toEqual([])
  })
})
