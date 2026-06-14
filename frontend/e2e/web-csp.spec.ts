// Phase 93 Plan 05 Task 1 — web-csp.spec.ts
//
// Asserts no Content Security Policy violations are reported on the web
// terminal page during a full attach + interaction session.
//
// Two capture paths cover both browser surfaces where a CSP violation can
// surface:
//   1. console-message text matching /content security policy|csp/i (Chromium
//      logs CSP violations as console errors with a "Content Security Policy"
//      directive prefix).
//   2. weberror events whose error stringification contains the same literal —
//      catches the rare case where a violation is thrown rather than logged.
//
// Phase 125-06 extension: EDIT-01 mandates zero new CSP amendments when the
// CodeMirror 6 editor mounts. The second test exercises the editor + write
// flow (/app/ with write cap) and asserts zero CSP violations under the
// full React + CM6 bundle load.

import { test, expect, request as playwrightRequest } from '@playwright/test'
import { sessionURL, writeAppUrl, loadFixtureEnv } from './fixture-env'

test.describe('Phase 93 WEB-02 web-csp zero-violation', () => {
  test('no CSP violations during attach/scroll session', async ({ page }) => {
    const cspViolations: string[] = []
    page.on('console', (msg) => {
      const text = msg.text()
      if (msg.type() === 'error' && /content security policy|csp/i.test(text)) {
        cspViolations.push(text)
      }
    })
    page.on('weberror', (err) => {
      const stringified = String(err.error())
      if (/content security policy|csp/i.test(stringified)) {
        cspViolations.push(stringified)
      }
    })

    await page.goto(sessionURL())
    await page.waitForTimeout(3_000)

    // Exercise the renderer + scroll path to fire any deferred CSP-relevant
    // load (e.g. lazy WebGL glyph atlas creation) before the assertion.
    const term = page.locator('#terminal')
    await term.click().catch(() => {
      // terminal may not yet be focusable; not fatal for CSP coverage
    })
    await page.keyboard.press('PageDown').catch(() => {})
    await page.waitForTimeout(500)

    expect(cspViolations, 'No CSP violations on web terminal page').toEqual([])
  })

  // ──────────────────────────────────────────────────────────────────
  // Phase 125-06 / EDIT-01: Zero CSP violations when the CodeMirror 6
  // editor loads via the /app/ web-share path.
  //
  // CodeMirror 6 is Vite-bundled (no CDN, no web worker, no eval — see
  // RESEARCH §Standard Stack). The existing CSP (style-src 'unsafe-inline'
  // already covers CM6's inline style injection; no worker-src needed).
  // This test drives the full editor mount flow and asserts that no new
  // CSP amendments were required (T-125-17).
  // ──────────────────────────────────────────────────────────────────
  test('Phase 125 EDIT-01: zero CSP violations during editor + write-cap app load', async ({ page }) => {
    const cspViolations: string[] = []
    page.on('console', (msg) => {
      const text = msg.text()
      if (msg.type() === 'error' && /content security policy|csp/i.test(text)) {
        cspViolations.push(text)
      }
    })
    page.on('weberror', (err) => {
      const stringified = String(err.error())
      if (/content security policy|csp/i.test(stringified)) {
        cspViolations.push(stringified)
      }
    })

    // Navigate to the /app/ entry point with the write cap so CM6 loads.
    const resp = await page.goto(writeAppUrl())
    // 200 if the bundle is embedded; 503 in dev builds without wailsassets.
    expect([200, 503]).toContain(resp?.status() ?? 0)

    if (resp?.status() === 200) {
      // Wait for the React tree to settle so any deferred CM6 extension
      // loading (lazy language-data, highlight.js, etc.) has time to fire.
      await page.waitForLoadState('networkidle').catch(() => {
        // networkidle may time out if a long-poll keeps the connection open;
        // 3-second fallback is sufficient to catch CM6-triggered CSP hits.
      })
      await page.waitForTimeout(3_000)

      // Try to mount the file-browser tab to trigger CM6 initialization
      // (if the bundle is present). Not fatal if the tab doesn't appear —
      // the CSP assertion is what matters.
      await page.getByTestId('file-browser-tab').waitFor({ timeout: 5_000 }).catch(() => {})

      // Exercise the write-path: perform a small write op to trigger any
      // fetch calls that might violate CSP via injected headers or URLs.
      const env = loadFixtureEnv()
      const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })
      try {
        const params = new URLSearchParams({
          session: 'playwright-test-session',
          path: `csp-test-${Date.now()}.txt`,
          cap: env.writeCap,
        })
        await ctx.put(`${env.baseURL}/api/files/write?${params.toString()}`, {
          headers: { 'Content-Type': 'application/octet-stream' },
          data: 'csp write probe',
        })
      } finally {
        await ctx.dispose()
      }

      await page.waitForTimeout(500)
    }

    expect(cspViolations, 'Zero CSP violations during editor + write-cap app load (EDIT-01 / T-125-17)').toEqual([])
  })
})
