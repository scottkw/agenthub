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
import { appUrl, viewerAppUrl, loadFixtureEnv } from './fixture-env'

// Close-button aria-label for the auto-opened file-browser tab. The web
// bootstrap opens it as `${sessionId} — Files` (App.tsx handleOpenFileBrowser).
const FILES_TAB_CLOSE = 'Close playwright-test-session — Files'

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

  // WEBCHAT-04: the file-browser tab auto-opens ONLY when the cap grants
  // files.read. A guest whose share lacks file access must not be left with a
  // dead "files.read permission required" tab.
  test('owner cap (files.read) → file-browser tab auto-opens in the background', async ({ browser }) => {
    const env = loadFixtureEnv()
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true })
    try {
      const page = await ctx.newPage()
      await page.goto(appUrl(env), { waitUntil: 'domcontentloaded' })
      await expect(page.locator('.hub-modal__chat-toggle')).toBeVisible({ timeout: 15_000 })
      // The owner cap includes files.read, so /info gates the tab open.
      await expect(page.getByRole('button', { name: FILES_TAB_CLOSE })).toHaveCount(1, { timeout: 15_000 })
      // The session view (not the file tab) remains the active surface.
      await expect(page.locator('.xterm').first()).toBeVisible()
    } finally {
      await ctx.close()
    }
  })

  test('viewer cap (no files.read) → NO file-browser tab (no dead permission-denied tab)', async ({ browser }) => {
    const env = loadFixtureEnv()
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true })
    try {
      const page = await ctx.newPage()
      await page.goto(viewerAppUrl(env), { waitUntil: 'domcontentloaded' })
      // Boot completes (chat surface renders) — proves the bootstrap ran.
      await expect(page.locator('.hub-modal__chat-toggle')).toBeVisible({ timeout: 15_000 })
      // Give the /info perms probe time to resolve; it must NOT open a file tab.
      await page.waitForTimeout(2_000)
      await expect(page.getByRole('button', { name: FILES_TAB_CLOSE })).toHaveCount(0)
      // The permission-denied takeover must never render for this guest.
      await expect(page.getByText('files.read permission required')).toHaveCount(0)
    } finally {
      await ctx.close()
    }
  })

  // WEBCHAT-05: a web-share guest cannot rename the session tab. RenameSession is
  // a Wails RPC with no bridge in a browser (it would fail silently and only
  // relabel the local tab), so the desktop-only tab menu + double-click rename
  // are suppressed in web mode.
  test('web-share guest cannot rename the tab (no session menu, no rename-on-double-click)', async ({ browser }) => {
    const env = loadFixtureEnv()
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true })
    try {
      const page = await ctx.newPage()
      await page.goto(appUrl(env), { waitUntil: 'domcontentloaded' })
      await expect(page.locator('.hub-modal__chat-toggle')).toBeVisible({ timeout: 15_000 })

      // No session-menu chevron in web mode.
      await expect(page.locator('[data-testid="tab-chevron"]')).toHaveCount(0)
      await expect(page.getByRole('button', { name: 'Session menu' })).toHaveCount(0)

      // Double-clicking the active tab name must NOT open the rename input.
      const sessionTab = page.locator('.tab__name', { hasText: 'Session' }).first()
      await sessionTab.dblclick()
      await page.waitForTimeout(300)
      await expect(page.locator('.tab__rename-input')).toHaveCount(0)
    } finally {
      await ctx.close()
    }
  })
})
