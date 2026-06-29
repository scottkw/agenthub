// Phase 93 Plan 05 Task 1 — web-plugin-hot-swap.spec.ts
//
// SKIPPED (v4.1, Phase 159 WEBCHAT-01): this suite tests the raw /sessions/{id}
// terminal.js viewer, which fetched /api/plugin-config + subscribed to the SSE
// stream for live plugin hot-swap (PLUG-04). Phase 159 retired that viewer —
// every share link now redirects to the /app/ React SPA, which acquires plugin
// settings via the Wails RPC GetPluginSettings() + EventsOn('settings:plugins')
// (desktop-only). In web mode (/app/ in a browser) there is no Wails bridge, so
// the SPA does NOT consume /api/plugin-config or the SSE stream at all — web
// guests no longer get live plugin-config hot-swap. The instrumentation here
// (hooking the UMD `window.WebglAddon` global) also cannot fire against the
// Vite-bundled SPA. This is a known, chat-orthogonal cross-surface gap tracked
// for a follow-up that wires /api/plugin-config + SSE into the /app/ web mode.
// Re-enable after that work lands. See: scottkw/agenthub#112 (web-guest
// plugin-config parity).
//
// Asserts the web terminal page honors plugin-config:
//   1. Initial /api/plugin-config webgl=true → WebGL addon is fetched from
//      /assets/xterm/addons/addon-webgl.js.
//   2. Initial /api/plugin-config webgl=false (route-mocked) → addon-webgl.js
//      is NOT fetched.
//   3. SSE push frame from /api/plugin-config/stream with webgl=false →
//      WebglAddon disposed without a page reload (PLUG-04 push channel).
//
// The third spec drives the fixture's POST /__test__/plugin-config endpoint
// which (a) updates the server's pluginSettingsProvider source-of-truth and
// (b) calls WebServer.BroadcastPluginConfig, fanning out to the open
// EventSource.

import { test, expect, request as pwRequest, type Page } from '@playwright/test'
import { loadFixtureEnv, sessionURL } from './fixture-env'

// installWebglAddonHook installs an init script that:
//   1. Spoofs WebGLRenderingContext.getParameter so RENDERER returns a string
//      that does NOT match the SwiftShader/llvmpipe/ANGLE-software regex used
//      by terminal.js' isSoftwareWebGL() probe. Without this, headless Chromium
//      (which renders WebGL via SwiftShader) is detected as software-rasterized
//      and the WebglAddon is intentionally never constructed — the test would
//      pass for the wrong reason.
//   2. Wraps the WebglAddon constructor to increment window.__phase93WebglCtorCount
//      every time the page constructs an addon. The setter pattern fires on
//      the UMD bundle's `globalThis.WebglAddon = factory()` assignment.
async function installWebglAddonHook(page: Page): Promise<void> {
  await page.addInitScript(() => {
    // Spoof renderer to a hardware-looking string.
    try {
      const proto = (window as unknown as { WebGLRenderingContext?: { prototype: { getParameter: (p: number) => unknown } } })
        .WebGLRenderingContext?.prototype
      if (proto) {
        const orig = proto.getParameter
        proto.getParameter = function (this: WebGLRenderingContext, pname: number): unknown {
          const RENDERER = 0x1f01
          if (pname === RENDERER) return 'NVIDIA GeForce RTX 4090'
          return orig.call(this, pname)
        } as typeof proto.getParameter
      }
      const proto2 = (window as unknown as { WebGL2RenderingContext?: { prototype: { getParameter: (p: number) => unknown } } })
        .WebGL2RenderingContext?.prototype
      if (proto2) {
        const orig2 = proto2.getParameter
        proto2.getParameter = function (this: WebGL2RenderingContext, pname: number): unknown {
          const RENDERER = 0x1f01
          if (pname === RENDERER) return 'NVIDIA GeForce RTX 4090'
          return orig2.call(this, pname)
        } as typeof proto2.getParameter
      }
    } catch {
      // best-effort
    }

    // Initialize counter.
    ;(window as unknown as { __phase93WebglCtorCount: number }).__phase93WebglCtorCount = 0

    // Hook WebglAddon assignment via the UMD's `root.WebglAddon = factory()`.
    Object.defineProperty(window, 'WebglAddon', {
      configurable: true,
      set(v: { WebglAddon: new (...args: unknown[]) => unknown }) {
        const Original = v.WebglAddon
        const Wrapped = function (this: unknown, ...args: unknown[]) {
          ;(window as unknown as { __phase93WebglCtorCount: number }).__phase93WebglCtorCount++
          return new (Original as new (...a: unknown[]) => unknown)(...args)
        } as unknown as typeof Original
        ;(Wrapped as unknown as { prototype: unknown }).prototype = (Original as unknown as { prototype: unknown }).prototype
        Object.defineProperty(window, 'WebglAddon', {
          configurable: true,
          writable: true,
          value: { ...v, WebglAddon: Wrapped },
        })
      },
    })
  })
}

test.describe.skip('Phase 93 WEB-03 + PLUG-04 push web-plugin-hot-swap', () => {
  test('initial /api/plugin-config webgl=false → no WebglAddon construction on web', async ({ page }) => {
    // Hook the WebglAddon constructor to count calls, AND override the page's
    // WebGL context to look hardware-accelerated so isSoftwareWebGL() returns
    // false. Without this, headless Chromium (SwiftShader) would correctly
    // skip construction regardless of pluginConfig.webgl, and the test would
    // pass for the wrong reason.
    await installWebglAddonHook(page)

    // Seed the fixture with webgl=false so BOTH the initial GET /api/plugin-config
    // AND the SSE first-frame return webgl=false. Otherwise the SSE stream
    // (which page.route() does not intercept by path) would push the default
    // webgl=true and trigger construction asynchronously.
    const env = loadFixtureEnv()
    {
      const ctx = await pwRequest.newContext({ ignoreHTTPSErrors: true })
      const seedResp = await ctx.post(`${env.adminURL}/__test__/plugin-config`, {
        data: {
          webgl: false,
          unicode11: true,
          clipboard: true,
          search: true,
          webLinks: true,
          image: true,
          serialize: true,
          progress: false,
        },
      })
      expect(seedResp.ok(), 'seed admin POST must succeed').toBeTruthy()
      await ctx.dispose()
    }

    await page.route('**/api/plugin-config', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          webgl: false,
          unicode11: true,
          clipboard: true,
          search: true,
          webLinks: true,
          image: true,
          serialize: true,
          progress: false,
        }),
      })
    })

    await page.goto(sessionURL())
    await page.waitForTimeout(2_500)

    const ctorCount = await page.evaluate(
      () => (window as unknown as { __phase93WebglCtorCount?: number }).__phase93WebglCtorCount ?? -1
    )
    expect(ctorCount, 'WebglAddon constructor MUST NOT be called when pluginConfig.webgl=false').toBe(0)
  })

  test('initial /api/plugin-config webgl=true → WebglAddon loads from vendor', async ({ page }) => {
    // Reset the fixture's source-of-truth to webgl=true so the SSE first
    // frame and the page's own GET /api/plugin-config both return webgl=true
    // (tests share the fixture process; prior test may have flipped state).
    const env = loadFixtureEnv()
    {
      const ctx = await pwRequest.newContext({ ignoreHTTPSErrors: true })
      const seedResp = await ctx.post(`${env.adminURL}/__test__/plugin-config`, {
        data: {
          webgl: true,
          unicode11: true,
          clipboard: true,
          search: true,
          webLinks: true,
          image: true,
          serialize: true,
          progress: false,
        },
      })
      expect(seedResp.ok(), 'seed admin POST must succeed').toBeTruthy()
      await ctx.dispose()
    }

    await installWebglAddonHook(page)
    await page.goto(sessionURL())
    await expect
      .poll(
        async () =>
          page.evaluate(() => (window as unknown as { __phase93WebglCtorCount?: number }).__phase93WebglCtorCount ?? 0),
        {
          message: 'WebglAddon constructor must be called when pluginConfig.webgl=true',
          timeout: 10_000,
        }
      )
      .toBeGreaterThanOrEqual(1)
  })

  test('SSE push from /api/plugin-config/stream hot-swaps WebGL OFF without page reload (PLUG-04)', async ({ page }) => {
    const env = loadFixtureEnv()

    // Reset the fixture's server-side plugin settings to webgl=true before
    // navigating so the page boots with WebGL active.
    {
      const ctx = await pwRequest.newContext({ ignoreHTTPSErrors: true })
      const seedResp = await ctx.post(`${env.adminURL}/__test__/plugin-config`, {
        data: {
          webgl: true,
          unicode11: true,
          clipboard: true,
          search: true,
          webLinks: true,
          image: true,
          serialize: true,
          progress: false,
        },
      })
      expect(seedResp.ok(), 'seed admin POST must succeed').toBeTruthy()
      await ctx.dispose()
    }

    // Hook constructor + spoof renderer string so isSoftwareWebGL() returns false.
    await installWebglAddonHook(page)

    let mainFrameNavigations = 0
    page.on('framenavigated', (frame) => {
      if (frame === page.mainFrame()) mainFrameNavigations++
    })

    await page.goto(sessionURL())
    // Allow the page to apply initial pluginConfig (which is webgl=true at this
    // point since we seeded the fixture above). Wait for the constructor hook
    // to register at least one call, confirming the WebglAddon was loaded.
    await expect
      .poll(
        async () =>
          page.evaluate(() => (window as unknown as { __phase93WebglCtorCount?: number }).__phase93WebglCtorCount ?? 0),
        {
          message: 'WebglAddon constructor must be called during initial load (webgl=true)',
          timeout: 10_000,
        }
      )
      .toBeGreaterThanOrEqual(1)
    // Allow the EventSource to open after initial load.
    await page.waitForTimeout(1_000)
    const navAfterInitialLoad = mainFrameNavigations

    // Now flip plugin settings server-side via the fixture admin endpoint,
    // which calls BroadcastPluginConfig() and sends the SSE frame.
    const ctx = await pwRequest.newContext({ ignoreHTTPSErrors: true })
    const flipResp = await ctx.post(`${env.adminURL}/__test__/plugin-config`, {
      data: {
        webgl: false,
        unicode11: true,
        clipboard: true,
        search: true,
        webLinks: true,
        image: true,
        serialize: true,
        progress: false,
      },
    })
    expect(flipResp.ok(), 'admin flip POST must succeed').toBeTruthy()
    await ctx.dispose()

    // Allow up to 3s for the SSE-driven hot-swap to complete. We assert:
    //   (a) The page DID NOT reload (frame navigation count stays the same).
    //   (b) The WebglAddon constructor was NOT called again after the flip
    //       (SSE push with webgl=false → applyPluginConfig only DISPOSES, it
    //       does not re-construct).
    const ctorCountBeforeFlipSnapshot = await page.evaluate(
      () => (window as unknown as { __phase93WebglCtorCount?: number }).__phase93WebglCtorCount ?? 0
    )
    await page.waitForTimeout(2_000)

    expect(mainFrameNavigations, 'no page reload during SSE-driven hot-swap (PLUG-04)').toBe(navAfterInitialLoad)

    const ctorCountAfter = await page.evaluate(
      () => (window as unknown as { __phase93WebglCtorCount?: number }).__phase93WebglCtorCount ?? 0
    )
    expect(
      ctorCountAfter,
      'WebglAddon constructor MUST NOT be called again after SSE webgl=false push (dispose-only path)'
    ).toBe(ctorCountBeforeFlipSnapshot)
  })
})
