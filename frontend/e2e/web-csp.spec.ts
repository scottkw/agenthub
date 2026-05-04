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

import { test, expect } from '@playwright/test'
import { sessionURL } from './fixture-env'

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
})
