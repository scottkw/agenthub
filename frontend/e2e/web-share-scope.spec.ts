// Phase 159 Plan 02 (gap-closure) — web-share-scope.spec.ts
//
// WEBCHAT-03: a remote web-share guest who lands on /app/?session=&cap= (via the
// 159-01 redirect) must see ONLY the scoped session surface — the terminal + chat
// (+ file-browser tab) — and NOT the desktop app chrome (the Sidebar with
// Home / Hub / Settings / session groups).
//
// Discovered during 159 live UAT: /app/ rendered the full desktop shell because
// <Sidebar> was rendered unconditionally in App.tsx (no `mode === 'web'` gate).
// A guest given a capability for one session could navigate to Home / Hub /
// Settings and reach the open /api/sessions/meta enumeration surface.
//
// Selector contract:
//   nav[aria-label="Main navigation"]  — the desktop Sidebar (MUST be absent in web mode)
//   .hub-modal__chat-toggle            — chat toggle (scoped surface present)
//   .xterm                             — xterm terminal (session surface present)

import { test, expect } from '@playwright/test'
import { appUrl, loadFixtureEnv } from './fixture-env'

test.describe('Phase 159-02 — web-share guest scope (WEBCHAT-03)', () => {
  test('web-share /app/ guest sees the scoped session surface, NOT the desktop Sidebar', async ({ browser }) => {
    const env = loadFixtureEnv()
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true })
    try {
      const page = await ctx.newPage()
      await page.goto(appUrl(env), { waitUntil: 'domcontentloaded' })

      // Scoped surface IS present: chat toggle + terminal mount.
      await expect(page.locator('.hub-modal__chat-toggle')).toBeVisible({ timeout: 15_000 })
      await expect(page.locator('.xterm').first()).toBeVisible({ timeout: 15_000 })

      // Desktop chrome is NOT present: the Sidebar nav must not render in web mode.
      await expect(page.locator('nav[aria-label="Main navigation"]')).toHaveCount(0)

      // Belt-and-suspenders: the Sidebar's nav entry points must be unreachable.
      await expect(page.getByRole('button', { name: 'Hub' })).toHaveCount(0)
      await expect(page.getByRole('button', { name: 'Settings' })).toHaveCount(0)
    } finally {
      await ctx.close()
    }
  })
})
