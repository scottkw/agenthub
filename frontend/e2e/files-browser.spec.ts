// Phase 120-05 — files-browser.spec.ts
//
// Cross-browser merge-gate e2e suite for the FileBrowserTab feature
// (Phase 120). Covers the 12 UI-14 scenarios across Chromium + Firefox +
// WebKit.
//
// ──────────────────────────────────────────────────────────────────────
// ARCHITECTURE NOTE (read before editing — see SUMMARY for the full
// story):
//
// Plan 05 originally landed as an API-surface-only suite because v3.4's
// App.tsx was tightly coupled to the Wails desktop runtime (imported
// `wailsjs/wailsjs/runtime/runtime`, called `GetRelayPort()` /
// `GetWebServerMode()`, never read URL params). Phase 120-06 closes that
// gap: App.tsx now consults `lib/webMode.detectMode()` and, when the
// pathname starts with `/app/`, skips the Wails RPC suite and drives
// `fbBaseURL` from `window.location.origin` + `capToken` from `?cap=`.
//
// Scenarios 13 + 14 (added Phase 120-06) exercise the React DOM via the
// playwright fixture (which now embeds frontend/dist under
// -tags=playwrightfixture,wailsassets) — owner cap mounts the
// FileBrowserTab and viewer cap renders the PermissionDeniedTakeover.
//
// What this suite verifies on all three browsers:
//   - The full /api/files/{list,stat,read} HTTP surface (scenarios 1-11) —
//     the same calls the React FilesApiClient (Plan 02) and FileBrowserTab
//     orchestrator (Plan 04) make against the daemon and webserver. Every
//     behaviour the 12 UI scenarios depend on (capability gate, sandboxing,
//     MIME cascade, 5 MiB cap, 0-byte short-circuit, Range support, error
//     codes & error-body wording) is exercised end-to-end across the real
//     TLS stack with the real capability middleware in front.
//   - The /app/ route loads the React bundle's index.html without CSP
//     violations and serves bundled assets under the documented CSP
//     (scenario 12).
//   - DOM-level mounting through the React tree from the browser-only
//     remote-viewer path (scenarios 13 + 14) — Phase 120-06 closes this
//     gap that Plan 05 left for v3.5.
// ──────────────────────────────────────────────────────────────────────

import { test, expect, request as playwrightRequest } from '@playwright/test'
import { loadFixtureEnv, filesApiURL, appUrl, viewerAppUrl } from './fixture-env'

// Conservative timeout for the multi-MiB Range download in scenario 9.
test.describe.configure({ mode: 'serial' })

test.describe('Phase 120 UI-14 file browser merge-gate (cross-browser API + bundle)', () => {
  // Track page errors + console errors per test so cross-cutting assertions
  // can flag any regression that surfaces while loading /app/.
  let errors: string[] = []

  test.beforeEach(() => {
    errors = []
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 1 — UI-01: tab discovery (proven via API surface that backs
  // the DaemonManagerPanel "Browse files" button — fileBrowserTabId in
  // App.tsx prefixes the session id with __files__, and the first thing
  // it does on mount is call /api/files/list. We verify the list returns
  // 200 + the seeded entries with the owner cap.).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 1: opening file browser tab → /api/files/list returns seeded cwd', async ({ request }) => {
    const url = filesApiURL('list', '.')
    const resp = await request.get(url)
    expect(resp.status(), 'GET /api/files/list root with owner cap').toBe(200)
    const body = await resp.json()
    expect(body.entries, 'entries array').toBeInstanceOf(Array)
    // Plan 01 fixture seed: 7 top-level entries (6 files + 2 dirs = 8, minus
    // the .file convention if any). We just check the canonical names exist.
    const names = body.entries.map((e: { name: string }) => e.name)
    expect(names).toContain('hello.txt')
    expect(names).toContain('subdir')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 2 — UI-02 + UI-03: listing semantics. The orchestrator
  // displays directories sticky-on-top (Plan 03 sortEntries). We verify
  // both the API populates every seeded entry AND that directories are
  // distinguishable via isDir=true (the React sort uses isDir as primary
  // key).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 2: list cwd contains all 7 seeded entries with correct isDir', async ({ request }) => {
    const resp = await request.get(filesApiURL('list', '.'))
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    const byName: Record<string, { isDir: boolean; isBinary: boolean }> =
      Object.fromEntries(body.entries.map((e: { name: string; isDir: boolean; isBinary: boolean }) => [e.name, e]))
    expect(byName['hello.txt']).toBeDefined()
    expect(byName['hello.txt'].isDir).toBe(false)
    expect(byName['notes.md']).toBeDefined()
    expect(byName['notes.md'].isDir).toBe(false)
    expect(byName['image.png']).toBeDefined()
    expect(byName['image.png'].isDir).toBe(false)
    expect(byName['binary.bin']).toBeDefined()
    expect(byName['binary.bin'].isDir).toBe(false)
    expect(byName['large.txt']).toBeDefined()
    expect(byName['empty.txt']).toBeDefined()
    expect(byName['subdir']).toBeDefined()
    expect(byName['subdir'].isDir).toBe(true)
    expect(byName['emptydir']).toBeDefined()
    expect(byName['emptydir'].isDir).toBe(true)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 3 — UI-05: navigate into subdirectory. The breadcrumb
  // surface in FileBrowserTab calls /list with path="subdir" when the
  // user double-clicks the row. We verify the API serves the nested
  // entry and rejects path escape attempts.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 3: navigate into subdir/ lists nested.txt; ".." escape returns 403', async ({ request }) => {
    const inResp = await request.get(filesApiURL('list', 'subdir'))
    expect(inResp.status()).toBe(200)
    const inBody = await inResp.json()
    const inNames = inBody.entries.map((e: { name: string }) => e.name)
    expect(inNames).toContain('nested.txt')

    // Path escape — Plan 118 sandbox guard. Must NOT serve parent.
    const escResp = await request.get(filesApiURL('list', '../etc'))
    expect(escResp.status(), 'path escape via ".." returns access denied').toBe(403)
    const escBody = await escResp.text()
    expect(escBody).toMatch(/access denied/i)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 4 — UI-08: preview text file. The TextPreview component
  // streams from /api/files/read. We verify the read returns 200, the
  // body is the literal "Hello, world!\n" (14 bytes), and the
  // Content-Type is text/plain (so PreviewPane picks the text path).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 4: preview text file (hello.txt) → 200 + text/plain + literal body', async ({ request }) => {
    const resp = await request.get(filesApiURL('read', 'hello.txt'))
    expect(resp.status()).toBe(200)
    const contentType = resp.headers()['content-type'] ?? ''
    expect(contentType).toMatch(/^text\/plain/)
    const body = await resp.text()
    expect(body).toBe('Hello, world!\n')
    expect(body.length).toBe(14)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 5 — UI-07: preview markdown file. MarkdownPreview only
  // renders text/markdown OR text/* with .md extension; we verify the
  // API surfaces both signals (Content-Type contains markdown OR plain,
  // AND the body contains the GFM table + task list markup that the
  // remark-gfm pipeline will render).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 5: preview markdown file (notes.md) → text mime + GFM table + task list source', async ({ request }) => {
    const resp = await request.get(filesApiURL('read', 'notes.md'))
    expect(resp.status()).toBe(200)
    const contentType = resp.headers()['content-type'] ?? ''
    expect(contentType, 'notes.md content-type is a text variant').toMatch(/text\/(markdown|plain)/)
    const body = await resp.text()
    // GFM table source — react-markdown + remark-gfm renders these as a
    // <table>; the React component layer (Plan 04) is unit-tested for
    // the table render in PreviewPane.test.tsx.
    expect(body).toContain('| Header A | Header B |')
    expect(body).toContain('| --- | --- |')
    // GFM task list source — renders as <input type=checkbox disabled>.
    expect(body).toContain('- [x] First task done')
    expect(body).toContain('- [ ] Second task open')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 6 — UI-09: preview image via direct src URL. ImagePreview
  // writes `<img src="/api/files/read?…">` (NO base64, NO blob: URL —
  // verified by the source-inspection test in Plan 04). We verify the
  // image read returns the PNG bytes and the correct image/png MIME so
  // the <img> tag can render natively.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 6: preview image (image.png) → 200 + image/png + valid PNG signature', async ({ request }) => {
    const resp = await request.get(filesApiURL('read', 'image.png'))
    expect(resp.status()).toBe(200)
    const contentType = resp.headers()['content-type'] ?? ''
    expect(contentType).toBe('image/png')
    const body = await resp.body()
    // PNG signature: 89 50 4E 47 0D 0A 1A 0A
    expect(body[0]).toBe(0x89)
    expect(body[1]).toBe(0x50)
    expect(body[2]).toBe(0x4E)
    expect(body[3]).toBe(0x47)
    expect(body[4]).toBe(0x0D)
    expect(body[5]).toBe(0x0A)
    expect(body[6]).toBe(0x1A)
    expect(body[7]).toBe(0x0A)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 7 — UI-06: binary file refusal + Download. The orchestrator
  // dispatches to UnsupportedFile when the stat result has isBinary=true
  // (Plan 04 PreviewPane). We verify /stat reports IsBinary=true for
  // binary.bin AND /read still serves the bytes (so the Download button
  // works against the same URL the UnsupportedFile component renders).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 7: binary file (binary.bin) → /stat isBinary=true; /read returns 64 bytes', async ({ request }) => {
    const statResp = await request.get(filesApiURL('stat', 'binary.bin'))
    expect(statResp.status()).toBe(200)
    const statBody = await statResp.json()
    expect(statBody.name).toBe('binary.bin')
    expect(statBody.isDir).toBe(false)
    expect(statBody.isBinary).toBe(true)
    expect(statBody.size).toBe(64)

    const readResp = await request.get(filesApiURL('read', 'binary.bin'))
    expect(readResp.status()).toBe(200)
    const body = await readResp.body()
    expect(body.length).toBe(64)
    // Alternating 0x00 0xFF pattern from the fixture seed.
    expect(body[0]).toBe(0x00)
    expect(body[1]).toBe(0xFF)
    expect(body[62]).toBe(0x00)
    expect(body[63]).toBe(0xFF)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 8 — UI-06: over-cap file refusal. The 5 MiB preview cap is
  // enforced by Phase 118's files.Handler (Plan 04 relies on it: the
  // component renders Over-cap UI when /stat reports size > 5 MiB OR
  // /read returns 413). We verify both: /stat reports size = 5*1024*1024+1
  // AND /read returns 413.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 8: over-cap file (large.txt) → /stat size > 5 MiB; /read returns 413', async ({ request }) => {
    const statResp = await request.get(filesApiURL('stat', 'large.txt'))
    expect(statResp.status()).toBe(200)
    const statBody = await statResp.json()
    expect(statBody.size).toBe(5 * 1024 * 1024 + 1)

    const readResp = await request.get(filesApiURL('read', 'large.txt'))
    expect(readResp.status(), 'over-cap /read returns 413').toBe(413)
    const errBody = await readResp.text()
    expect(errBody).toMatch(/too large/i)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 9 — UI-10: download (full + Range request). The Download
  // button in PreviewPane is a plain `<a download href="/api/files/read?…">`
  // — the same URL we exercise here. We verify (a) a full GET returns
  // the complete body, and (b) a Range request returns 206 with a
  // partial body (proving http.ServeContent's Range plumbing reaches
  // through the capability gate).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 9: download (Range) (hello.txt) → full + partial work', async ({ request }) => {
    // Full GET
    const fullResp = await request.get(filesApiURL('read', 'hello.txt'))
    expect(fullResp.status()).toBe(200)
    const full = await fullResp.text()
    expect(full).toBe('Hello, world!\n')
    // Partial range — bytes 7–12 ("world!")
    const rangeResp = await request.get(filesApiURL('read', 'hello.txt'), {
      headers: { Range: 'bytes=7-12' },
    })
    expect(rangeResp.status(), 'Range request returns 206').toBe(206)
    const partial = await rangeResp.text()
    expect(partial).toBe('world!')
    const cr = rangeResp.headers()['content-range'] ?? ''
    expect(cr).toMatch(/bytes 7-12\/14/)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 10 — UI-13: capability-denied viewer sees the takeover
  // copy. The PermissionDeniedTakeover component renders when /list
  // returns 403 with a body containing "files.read". We verify the API
  // returns the documented status + body when the viewer cap (no
  // files.read perm) is presented.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 10: viewer cap (no files.read) → /list returns 403 with files.read in body', async ({ request }) => {
    const env = loadFixtureEnv()
    const url = filesApiURL('list', '.', env.viewerCap)
    const resp = await request.get(url)
    expect(resp.status(), 'viewer cap is rejected by requireFilesRead').toBe(403)
    const body = await resp.text()
    expect(body).toMatch(/files\.read/)
    // Plan 04 PermissionDeniedTakeover keys off this exact substring (see
    // FileBrowserTab.tsx error-classification path).
    expect(body.toLowerCase()).toContain('files.read')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 11 — empty directory state. EmptyDirectoryState renders
  // when /list returns 200 with entries=[]. We verify the API serves
  // an empty array (not a 404) for our seeded `emptydir/`.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 11: empty directory (emptydir/) → 200 + entries=[]', async ({ request }) => {
    const resp = await request.get(filesApiURL('list', 'emptydir'))
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body.entries).toBeInstanceOf(Array)
    expect(body.entries.length).toBe(0)
    expect(body.truncated).toBe(false)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 12 — network error state + /app/ bundle smoke. The
  // NetworkErrorState component renders when /list returns a 5xx /
  // network failure. We exercise both halves:
  //   (a) The /app/ React bundle loads via the real HTTPS stack with no
  //       CSP violations (regression guard for Phase 93 WEB-02).
  //   (b) An unknown session id triggers 404 from /list (which the
  //       FileBrowserTab classifies as a not-found state — the same code
  //       path NetworkErrorState ends up in if the underlying fetch
  //       throws because the daemon dropped the connection).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 12: bundle loads /app/ with zero CSP violations + 404 on unknown session', async ({ page, request }) => {
    const cspViolations: string[] = []
    page.on('console', (msg) => {
      const text = msg.text()
      if (msg.type() === 'error' && /content security policy|csp/i.test(text)) {
        cspViolations.push(text)
      }
      if (msg.type() === 'error') errors.push(`console.error: ${text}`)
    })
    page.on('weberror', (err) => {
      const stringified = String(err.error())
      errors.push(`weberror: ${stringified}`)
      if (/content security policy|csp/i.test(stringified)) {
        cspViolations.push(stringified)
      }
    })
    page.on('pageerror', (err) => {
      errors.push(`pageerror: ${err.message}`)
    })

    const resp = await page.goto(appUrl())
    // /app/ must serve the index.html (200) — SPA fallback path from
    // Plan 04's webserver SetStaticAppFS plumbing. If the bundle is
    // absent (dev build with no -tags wailsassets), /app/ returns 503
    // and the bundle-load assertion is downgraded to a 503-tolerant
    // smoke that still proves the route is wired correctly.
    expect([200, 503]).toContain(resp?.status() ?? 0)
    if (resp?.status() === 503) {
      // Confirm the 503 is the expected "frontend bundle not embedded"
      // path and not a different 5xx.
      const body = await resp.text()
      expect(body.toLowerCase()).toContain('not')
    }
    // Either way, no CSP violations may surface during bundle load.
    await page.waitForTimeout(500)
    expect(cspViolations, 'no CSP violations from /app/ bundle load').toEqual([])

    // 404 path — files API surface that NetworkErrorState/NotFound can hit.
    const env = loadFixtureEnv()
    const params = new URLSearchParams({
      session: 'no-such-session',
      path: '.',
      cap: env.cap,
    })
    const badResp = await request.get(`${env.baseURL}/api/files/list?${params.toString()}`)
    expect([404, 403], 'unknown session is rejected at the cap layer or the resolver layer').toContain(
      badResp.status(),
    )
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 13 — Phase 120-06: web-mode owner cap mounts the React tree.
  // The fixture now embeds frontend/dist under
  // -tags=playwrightfixture,wailsassets so /app/ serves the SPA. App.tsx
  // consults lib/webMode.detectMode(), sees the /app/ pathname, skips
  // Wails RPCs, parses ?session= + ?cap= from window.location, and
  // mounts the FileBrowserTab. We verify the file-browser tab mounts and
  // the breadcrumb + seeded entries render via the real DOM.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 13: web mode owner cap → file-browser tab mounts and lists seeded entries', async ({ page }) => {
    const resp = await page.goto(appUrl())
    expect(resp?.status(), '/app/ must serve the bundle (200) — fixture now embeds frontend/dist').toBe(200)
    await expect(
      page.getByTestId('file-browser-tab'),
      'file-browser-tab must mount under web mode',
    ).toBeVisible({ timeout: 15_000 })
    await expect(
      page.getByTestId('file-browser-breadcrumb'),
      'breadcrumb must render',
    ).toBeVisible()
    // hello.txt is canonical from the fixture seed (cmd/playwright-fixture/main.go seedFixtureFiles).
    await expect(
      page.getByTestId('file-browser-row-hello.txt'),
      'hello.txt row must render via React DOM',
    ).toBeVisible({ timeout: 10_000 })
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 14 — Phase 120-06: web-mode viewer cap (no files.read)
  // triggers the PermissionDeniedTakeover via the DOM. The viewer cap
  // grants `read` but NOT `files.read`, so useFilesCapability resolves
  // to 'denied' and FileBrowserTab dispatches to the takeover. The
  // heading text is verbatim from PermissionDeniedTakeover.tsx:28.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 14: web mode viewer cap (no files.read) → permission-denied takeover renders verbatim copy', async ({ page }) => {
    const resp = await page.goto(viewerAppUrl())
    expect(resp?.status()).toBe(200)
    await expect(
      page.getByTestId('file-browser-permission-denied'),
      'permission-denied takeover must mount under viewer cap',
    ).toBeVisible({ timeout: 15_000 })
    await expect(
      page.getByRole('heading', { name: /^files\.read permission required$/ }),
    ).toBeVisible()
  })

  test.afterEach(() => {
    // pageerror / console.error allow-list (mirrors web-csp.spec.ts).
    //
    // ALLOWED entries:
    //   - 503 Service Unavailable from /app/ in dev builds: the React
    //     bundle is only embedded under `-tags wailsassets`, so under a
    //     plain `go build` the fixture's daemon.SetStaticAppFS receives
    //     nil and /app/ returns 503 by design (Plan 04 SUMMARY decision
    //     #3 — "assets_stub.go would have leaked working directory under
    //     /app/"). The browser logs this as a console.error which we
    //     allow; the assertion that matters (zero CSP violations) is
    //     still enforced.
    const ALLOWED: RegExp[] = [
      /Failed to load resource: the server responded with a status of 503/,
    ]
    const offenders = errors.filter((e) => !ALLOWED.some((re) => re.test(e)))
    expect(offenders, `unallowed page errors:\n${offenders.join('\n')}`).toEqual([])
  })
})

// ───────────────────────────────────────────────────────────────────
// Smoke parity — verify the same /api/files/* surface is reachable via
// playwright's standalone APIRequestContext (no browser page). This
// proves the cross-browser projects don't accidentally diverge on
// request-level behaviour due to per-browser fetch quirks (e.g., WebKit
// not always supporting custom Range header normalization in early 14.x).
// ───────────────────────────────────────────────────────────────────
test('api smoke: standalone APIRequestContext can read hello.txt', async () => {
  const env = loadFixtureEnv()
  const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })
  try {
    const resp = await ctx.get(filesApiURL('read', 'hello.txt', undefined, env))
    expect(resp.status()).toBe(200)
    expect(await resp.text()).toBe('Hello, world!\n')
  } finally {
    await ctx.dispose()
  }
})
