// Phase 125-06 — files-write.spec.ts
//
// Cross-browser merge-gate e2e suite for the FileBrowser write surface
// (Phase 125). Covers all 14 EDIT-13 scenarios across Chromium + Firefox +
// WebKit on the web-share surface.
//
// ──────────────────────────────────────────────────────────────────────
// ARCHITECTURE NOTE:
//
// All write routes require the files.write capability perm. The suite uses
// two cap tokens:
//
//   WRITE_CAP (read,files.read,files.write) — for all write-success and
//     412-conflict scenarios. Built into writeAppUrl(env) and used directly
//     for API-surface tests below.
//
//   viewerCap (read only, no files.read or files.write) — for the
//     403-without-cap scenario (EDIT-13 scenario 3 / T-125-16).
//
// Routes under test (all gated by requireFilesWrite on web-share):
//   PUT    /api/files/write    — write file (If-Match + 412 concurrency)
//   POST   /api/files/mkdir    — create directory
//   DELETE /api/files/delete   — delete file or directory (recursive)
//   POST   /api/files/rename   — rename or cross-dir move
//   POST   /api/files/upload   — multipart single/multi-file upload
//
// CSP assertion: every test listens for console CSP errors and
//   securitypolicyviolation events; the afterEach hook fails if any fire.
// ──────────────────────────────────────────────────────────────────────

import { test, expect, request as playwrightRequest } from '@playwright/test'
import { loadFixtureEnv, filesApiURL, appUrl, writeAppUrl } from './fixture-env'

test.describe.configure({ mode: 'serial' })

// ──────────────────────────────────────────────────────────────────────
// Helper: build a write-API URL for a given operation using the WRITE_CAP.
// ──────────────────────────────────────────────────────────────────────
function filesWriteApiURL(
  op: 'write' | 'mkdir' | 'delete' | 'rename' | 'upload',
  path: string = '.',
  cap?: string,
): string {
  const env = loadFixtureEnv()
  const useCap = cap ?? env.writeCap
  const params = new URLSearchParams({
    session: 'playwright-test-session',
    path,
    cap: useCap,
  })
  return `${env.baseURL}/api/files/${op}?${params.toString()}`
}

test.describe('Phase 125 EDIT-13 write surface merge-gate (cross-browser)', () => {
  // Collect CSP violations and page errors per test for afterEach assertion.
  let cspViolations: string[] = []
  let errors: string[] = []

  test.beforeEach(async ({ page }) => {
    cspViolations = []
    errors = []

    page.on('console', (msg) => {
      const text = msg.text()
      if (msg.type() === 'error' && /content security policy|csp/i.test(text)) {
        cspViolations.push(text)
      }
      if (msg.type() === 'error') {
        errors.push(`console.error: ${text}`)
      }
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
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 1 — EDIT-13 / EDIT-05: web-share write-and-save with
  // files.write cap succeeds (200). Confirms the WRITE_CAP routes
  // through requireFilesWrite to Handler.Write without rejection.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 1: local write-and-save — PUT /api/files/write returns 200', async ({ request }) => {
    const url = filesWriteApiURL('write', 'write-test.txt')
    const resp = await request.put(url, {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'hello from write test',
    })
    expect(resp.status(), 'PUT write with files.write cap must return 200').toBe(200)
    const body = await resp.json()
    expect(body.path, 'response path matches').toBe('write-test.txt')
    expect(body.size, 'response size matches').toBeGreaterThan(0)

    // Verify the file is readable after write.
    const readResp = await request.get(filesApiURL('read', 'write-test.txt'))
    expect(readResp.status()).toBe(200)
    expect(await readResp.text()).toBe('hello from write test')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 2 — EDIT-13 / EDIT-12: web-share write with files.write cap.
  // Exercises the web-share URL + cap flow end-to-end via the DOM.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 2: web-share write with files.write cap — app loads + file-browser tab mounts', async ({ page }) => {
    const resp = await page.goto(writeAppUrl())
    // The React bundle must load (200) — fixture embeds frontend/dist.
    expect(resp?.status(), 'writeAppUrl must serve bundle (200)').toBe(200)
    // Phase 159-03 (WEBCHAT-04): the file-browser tab auto-opens in the
    // BACKGROUND while the session/chat surface stays active. Activate it
    // before asserting its content (a write cap includes files.read).
    await page
      .getByRole('button', { name: 'Close playwright-test-session — Files' })
      .waitFor({ timeout: 15_000 })
    await page.locator('.tab__name').filter({ hasText: 'playwright-test-session — Files' }).click()
    await expect(
      page.getByTestId('file-browser-tab'),
      'file-browser-tab must mount under write cap',
    ).toBeVisible({ timeout: 15_000 })
    await expect(
      page.getByTestId('file-browser-breadcrumb'),
      'breadcrumb must render',
    ).toBeVisible()
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 3 — EDIT-13 / T-125-16: 403 without files.write cap.
  // The viewer cap has no files.write perm; requireFilesWrite returns 403.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 3: 403 without files.write cap — viewer cap rejected', async ({ request }) => {
    const env = loadFixtureEnv()
    const url = filesWriteApiURL('write', 'denied-test.txt', env.viewerCap)
    const resp = await request.put(url, {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'should be rejected',
    })
    expect(resp.status(), 'viewer cap (no files.write) must return 403').toBe(403)
    const body = await resp.text()
    // requireFilesWrite emits "files.write" in the body (verified capability_mw.go:147).
    expect(body.toLowerCase()).toContain('files.write')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 4 — EDIT-13 / EDIT-09: create file.
  // PUT /api/files/write with no If-Match header creates a new file.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 4: create file — PUT write to new path succeeds', async ({ request }) => {
    const newFilePath = `created-${Date.now()}.txt`
    const url = filesWriteApiURL('write', newFilePath)
    const resp = await request.put(url, {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'new file content',
    })
    expect(resp.status(), 'create new file returns 200').toBe(200)
    const body = await resp.json()
    expect(body.path).toBe(newFilePath)

    // Verify file appears in listing.
    const listResp = await request.get(filesApiURL('list', '.'))
    const listBody = await listResp.json()
    const names = listBody.entries.map((e: { name: string }) => e.name)
    expect(names, 'newly created file appears in list').toContain(newFilePath)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 5 — EDIT-13 / EDIT-09: mkdir.
  // POST /api/files/mkdir creates a directory.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 5: mkdir — POST /api/files/mkdir creates directory', async ({ request }) => {
    const newDir = `newdir-${Date.now()}`
    const url = filesWriteApiURL('mkdir', newDir)
    const resp = await request.post(url)
    expect(resp.status(), 'mkdir returns 200 or 201').toBeGreaterThanOrEqual(200)
    expect(resp.status()).toBeLessThan(300)

    // Verify directory appears in listing.
    const listResp = await request.get(filesApiURL('list', '.'))
    const listBody = await listResp.json()
    const dirs = listBody.entries.filter((e: { isDir: boolean; name: string }) => e.isDir).map((e: { name: string }) => e.name)
    expect(dirs, 'newly created directory appears in list').toContain(newDir)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 6 — EDIT-13 / EDIT-09: delete file.
  // DELETE /api/files/delete removes a file from the sandbox.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 6: delete file — DELETE /api/files/delete removes file', async ({ request }) => {
    // First create a file to delete.
    const targetPath = `to-delete-${Date.now()}.txt`
    await request.put(filesWriteApiURL('write', targetPath), {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'delete me',
    })

    // Now delete it.
    const delUrl = filesWriteApiURL('delete', targetPath)
    const delResp = await request.delete(delUrl)
    expect(delResp.status(), 'delete returns 200').toBe(200)

    // Verify file no longer in listing.
    const listResp = await request.get(filesApiURL('list', '.'))
    const listBody = await listResp.json()
    const names = listBody.entries.map((e: { name: string }) => e.name)
    expect(names, 'deleted file no longer in list').not.toContain(targetPath)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 7 — EDIT-13 / EDIT-09: delete directory (recursive confirm
  // with file count). Creates a dir with files, deletes recursively.
  // The client shows a file-count confirm; the server deletes recursively.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 7: delete dir (recursive confirm with count) — DELETE removes dir + contents', async ({ request }) => {
    // Create a directory.
    const dirPath = `recurse-dir-${Date.now()}`
    await request.post(filesWriteApiURL('mkdir', dirPath))

    // Seed two files inside it (simulate "N files inside" for the modal count).
    for (const name of ['file1.txt', 'file2.txt']) {
      await request.put(filesWriteApiURL('write', `${dirPath}/${name}`), {
        headers: { 'Content-Type': 'application/octet-stream' },
        data: `content of ${name}`,
      })
    }

    // Client-side: count files inside the dir (emulates the modal count walk).
    const preList = await request.get(filesApiURL('list', dirPath))
    const preBody = await preList.json()
    const fileCount = preBody.entries.filter((e: { isDir: boolean }) => !e.isDir).length
    expect(fileCount, 'two files inside the dir before recursive delete').toBe(2)

    // Now recursively delete the directory.
    const delUrl = filesWriteApiURL('delete', dirPath)
    const delResp = await request.delete(delUrl)
    expect(delResp.status(), 'recursive dir delete returns 200').toBe(200)

    // Verify dir no longer in listing.
    const listResp = await request.get(filesApiURL('list', '.'))
    const listBody = await listResp.json()
    const names = listBody.entries.map((e: { name: string }) => e.name)
    expect(names, 'recursively deleted dir no longer in list').not.toContain(dirPath)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 8 — EDIT-13 / EDIT-09: rename.
  // POST /api/files/rename renames a file (or moves within the same dir).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 8: rename — POST /api/files/rename changes filename', async ({ request }) => {
    const env = loadFixtureEnv()
    const oldName = `rename-src-${Date.now()}.txt`
    const newName = `rename-dst-${Date.now()}.txt`

    // Create source file.
    await request.put(filesWriteApiURL('write', oldName), {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'rename me',
    })

    // Rename it.
    const params = new URLSearchParams({ session: 'playwright-test-session', cap: env.writeCap })
    const renameUrl = `${env.baseURL}/api/files/rename?${params.toString()}`
    const renameResp = await request.post(renameUrl, {
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ oldRel: oldName, newRel: newName }),
    })
    expect(renameResp.status(), 'rename returns 200').toBe(200)

    // Verify old name gone, new name present.
    const listResp = await request.get(filesApiURL('list', '.'))
    const listBody = await listResp.json()
    const names = listBody.entries.map((e: { name: string }) => e.name)
    expect(names, 'old name gone after rename').not.toContain(oldName)
    expect(names, 'new name present after rename').toContain(newName)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 9 — EDIT-13 / EDIT-09: cross-directory move.
  // POST /api/files/rename with a different directory component is the
  // move primitive (rename validates both oldRel and newRel paths).
  // ───────────────────────────────────────────────────────────────────
  test('scenario 9: cross-dir move — rename with different dir component', async ({ request }) => {
    const env = loadFixtureEnv()
    const srcFile = `move-src-${Date.now()}.txt`
    const dstDir = `move-dst-dir-${Date.now()}`
    const dstFile = `${dstDir}/moved.txt`

    // Create source file and destination directory.
    await request.put(filesWriteApiURL('write', srcFile), {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'move me cross-dir',
    })
    await request.post(filesWriteApiURL('mkdir', dstDir))

    // Cross-dir move: rename oldRel=srcFile newRel=dstDir/moved.txt
    const params = new URLSearchParams({ session: 'playwright-test-session', cap: env.writeCap })
    const moveUrl = `${env.baseURL}/api/files/rename?${params.toString()}`
    const moveResp = await request.post(moveUrl, {
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ oldRel: srcFile, newRel: dstFile }),
    })
    expect(moveResp.status(), 'cross-dir move returns 200').toBe(200)

    // Source gone from root.
    const rootList = await request.get(filesApiURL('list', '.'))
    const rootNames = (await rootList.json()).entries.map((e: { name: string }) => e.name)
    expect(rootNames, 'source file gone from root after cross-dir move').not.toContain(srcFile)

    // Destination present in dstDir.
    const dirList = await request.get(filesApiURL('list', dstDir))
    const dirNames = (await dirList.json()).entries.map((e: { name: string }) => e.name)
    expect(dirNames, 'moved file present in destination dir').toContain('moved.txt')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 10 — EDIT-13 / EDIT-10: single file upload.
  // POST /api/files/upload with a multipart form and a single file part.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 10: single upload — POST /api/files/upload places file', async ({ request }) => {
    const env = loadFixtureEnv()
    const uploadDir = '.'
    const fileName = `upload-single-${Date.now()}.txt`

    const params = new URLSearchParams({ session: 'playwright-test-session', cap: env.writeCap })
    const uploadUrl = `${env.baseURL}/api/files/upload?${params.toString()}`

    // Playwright request.post with multipart form data.
    const resp = await request.post(uploadUrl, {
      multipart: {
        dir: uploadDir,
        file: {
          name: fileName,
          mimeType: 'text/plain',
          buffer: Buffer.from('single upload content'),
        },
      },
    })
    expect(resp.status(), 'single upload returns 200').toBe(200)

    // Verify uploaded file is readable.
    const readResp = await request.get(filesApiURL('read', fileName))
    expect(readResp.status()).toBe(200)
    expect(await readResp.text()).toBe('single upload content')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 11 — EDIT-13 / EDIT-10: multi-file upload.
  // Multi-file upload = N sequential XHR calls (one file part each).
  // We verify two sequential uploads succeed and both files land.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 11: multi-file upload — two sequential uploads land', async ({ request }) => {
    const env = loadFixtureEnv()
    const uploadDir = '.'
    const params = new URLSearchParams({ session: 'playwright-test-session', cap: env.writeCap })
    const uploadUrl = `${env.baseURL}/api/files/upload?${params.toString()}`

    const files = [
      { name: `multi-upload-a-${Date.now()}.txt`, content: 'file a content' },
      { name: `multi-upload-b-${Date.now()}.txt`, content: 'file b content' },
    ]

    for (const f of files) {
      const resp = await request.post(uploadUrl, {
        multipart: {
          dir: uploadDir,
          file: {
            name: f.name,
            mimeType: 'text/plain',
            buffer: Buffer.from(f.content),
          },
        },
      })
      expect(resp.status(), `multi-upload ${f.name} returns 200`).toBe(200)
    }

    // Verify both files are readable.
    for (const f of files) {
      const readResp = await request.get(filesApiURL('read', f.name))
      expect(readResp.status(), `multi-upload ${f.name} readable`).toBe(200)
      expect(await readResp.text()).toBe(f.content)
    }
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 11b — EDIT-13 / EDIT-10 / WR-07: upload-collision 409 + overwrite.
  // The new contract (WR-07): upload 409s when the target exists UNLESS
  // overwrite=1 is present. This makes the Replace affordance reachable.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 11b: upload-collision 409 and overwrite=1 path', async ({ request }) => {
    const env = loadFixtureEnv()
    const uploadDir = '.'
    const params = new URLSearchParams({ session: 'playwright-test-session', cap: env.writeCap })
    const uploadUrl = `${env.baseURL}/api/files/upload?${params.toString()}`
    const fileName = `collision-upload-${Date.now()}.txt`

    // First upload — new file, expect 200.
    const first = await request.post(uploadUrl, {
      multipart: {
        dir: uploadDir,
        file: { name: fileName, mimeType: 'text/plain', buffer: Buffer.from('v1') },
      },
    })
    expect(first.status(), 'first upload (new file) returns 200').toBe(200)

    // Second upload of the same name WITHOUT overwrite=1 — expect 409 (collision).
    const second = await request.post(uploadUrl, {
      multipart: {
        dir: uploadDir,
        file: { name: fileName, mimeType: 'text/plain', buffer: Buffer.from('v2') },
      },
    })
    expect(second.status(), 'second upload (same name, no overwrite) returns 409').toBe(409)

    // Third upload WITH overwrite=1 — expect 200 (Replace semantics).
    const overwriteParams = new URLSearchParams({ session: 'playwright-test-session', cap: env.writeCap })
    const overwriteUrl = `${env.baseURL}/api/files/upload?${overwriteParams.toString()}`
    const third = await request.post(overwriteUrl, {
      multipart: {
        dir: uploadDir,
        overwrite: '1',
        file: { name: fileName, mimeType: 'text/plain', buffer: Buffer.from('v3-overwrite') },
      },
    })
    expect(third.status(), 'third upload with overwrite=1 returns 200').toBe(200)

    // Verify v3 content is on disk.
    const readResp = await request.get(filesApiURL('read', fileName))
    expect(await readResp.text(), 'overwrite=1 lands v3 content').toBe('v3-overwrite')
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 12 — EDIT-13 / EDIT-08: 412 conflict flow (stale If-Match).
  // Write a file, capture its ETag, then stale it by writing again, then
  // send a PUT with the STALE If-Match. Server must return 412.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 12: 412 conflict flow — stale If-Match returns 412', async ({ request }) => {
    const conflictPath = `conflict-test-${Date.now()}.txt`

    // Initial write — no If-Match (new file).
    const initResp = await request.put(filesWriteApiURL('write', conflictPath), {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'initial content',
    })
    expect(initResp.status(), 'initial write returns 200').toBe(200)

    // Read to capture the ETag.
    const readResp = await request.get(filesApiURL('read', conflictPath))
    expect(readResp.status()).toBe(200)
    const etag = readResp.headers()['etag']
    // ETag is emitted by Phase 125-01 Handler.Read as "<UnixNano>-<size>".
    expect(etag, 'ETag header must be present after write').toBeTruthy()

    // Second write changes the file (simulates another process writing it).
    await request.put(filesWriteApiURL('write', conflictPath), {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'updated by another process',
    })

    // Now attempt to write with the STALE etag — must return 412.
    const staleResp = await request.put(filesWriteApiURL('write', conflictPath), {
      headers: {
        'Content-Type': 'application/octet-stream',
        'If-Match': etag!, // stale — the second write changed the mtime+size validator
      },
      data: 'user edit that arrived after the stale read',
    })
    expect(staleResp.status(), '412 stale If-Match conflict flow').toBe(412)
    const body = await staleResp.text()
    expect(body.toLowerCase()).toContain('modified')

    // Force-overwrite path: If-Match="*" bypasses the 412 check.
    const forceResp = await request.put(filesWriteApiURL('write', conflictPath), {
      headers: {
        'Content-Type': 'application/octet-stream',
        'If-Match': '*',
      },
      data: 'force-overwrote',
    })
    expect(forceResp.status(), 'force-overwrite with If-Match=* returns 200').toBe(200)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 13 — EDIT-13 / EDIT-03: binary file no-edit.
  // The binary.bin fixture file has isBinary=true; the Edit button must
  // be absent in the DOM. We verify this at the API level: /stat returns
  // isBinary=true, confirming the component can correctly suppress the
  // Edit affordance.
  // ───────────────────────────────────────────────────────────────────
  test('scenario 13: binary-file no-edit — /stat reports isBinary=true + Edit affordance absent', async ({ page }) => {
    // API: stat returns isBinary=true for binary.bin.
    const env = loadFixtureEnv()
    const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })
    try {
      const statResp = await ctx.get(filesApiURL('stat', 'binary.bin'))
      expect(statResp.status()).toBe(200)
      const stat = await statResp.json()
      expect(stat.isBinary, 'binary.bin /stat returns isBinary=true').toBe(true)
    } finally {
      await ctx.dispose()
    }

    // DOM: navigate to the write-cap app and verify the binary file row
    // does NOT have an Edit (pencil) button affordance.
    const resp = await page.goto(writeAppUrl())
    expect(resp?.status()).toBe(200)
    // Phase 159-03 (WEBCHAT-04): the file-browser tab auto-opens in the
    // background; activate it before asserting its content.
    await page
      .getByRole('button', { name: 'Close playwright-test-session — Files' })
      .waitFor({ timeout: 15_000 })
    await page.locator('.tab__name').filter({ hasText: 'playwright-test-session — Files' }).click()
    await expect(page.getByTestId('file-browser-tab')).toBeVisible({ timeout: 15_000 })

    // Wait for the file list to render.
    await expect(
      page.getByTestId('file-browser-row-binary.bin'),
      'binary.bin row must render',
    ).toBeVisible({ timeout: 10_000 })

    // The Edit button for a binary file must NOT be present.
    // It carries aria-label="Edit binary.bin" per PreviewPane pattern.
    const editBtn = page.locator('[aria-label="Edit binary.bin"]')
    await expect(editBtn, 'Edit button absent for binary file').toHaveCount(0)
  })

  // ───────────────────────────────────────────────────────────────────
  // Scenario 14 — EDIT-13 / EDIT-11: large-file guard.
  // large.txt is > 5 MiB; the read route returns 413. Files > 500 KB
  // show a large-file notice before the edit affordance. We verify:
  // (a) /read returns 413 (the backend large-file cap is present)
  // (b) /stat reports size > 500 KB (the frontend guard threshold)
  // ───────────────────────────────────────────────────────────────────
  test('scenario 14: large-file guard — /read 413; /stat size > 500 KB', async ({ request }) => {
    // large.txt is seeded at 5 MiB + 1 byte by the fixture.
    const readResp = await request.get(filesApiURL('read', 'large.txt'))
    expect(readResp.status(), '/read for large.txt returns 413 (over-cap)').toBe(413)
    const errBody = await readResp.text()
    expect(errBody).toMatch(/too large/i)

    // /stat reports size > 500 KB (the EDIT-11 guard threshold).
    const statResp = await request.get(filesApiURL('stat', 'large.txt'))
    expect(statResp.status()).toBe(200)
    const stat = await statResp.json()
    expect(stat.size, 'large.txt size exceeds 500 KB guard threshold').toBeGreaterThan(500 * 1024)

    // Write large.txt via the WRITE_CAP route → still 200 (the PUT route
    // has its own maxBytesReader; this test writes a small payload to a
    // different path to confirm the route is live and gated correctly).
    const largePath = `large-write-guard-${Date.now()}.txt`
    const writeResp = await request.put(filesWriteApiURL('write', largePath), {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'small content confirming write route is live',
    })
    expect(writeResp.status(), 'write route is live for write cap').toBe(200)
  })

  test.afterEach(() => {
    // Fail if any CSP violations were observed in this test.
    expect(cspViolations, `zero CSP violations:\n${cspViolations.join('\n')}`).toEqual([])

    // Allowed console errors (pre-existing app behaviors that are not regressions):
    //   1. 503 — dev builds without -tags wailsassets don't embed the bundle; /app/ returns 503.
    //   2. 404 — other resource-not-found errors from app initialization.
    //   Note: after CR-02 fix, HEAD /api/files/write now returns 200 (with files.write cap)
    //   or 403 (without), never 405. The 404 allowance covers other non-write routes.
    const ALLOWED: RegExp[] = [
      /Failed to load resource: the server responded with a status of 503/,
      /Failed to load resource: the server responded with a status of 404/,
    ]
    const offenders = errors.filter((e) => !ALLOWED.some((re) => re.test(e)))
    expect(offenders, `unallowed page errors:\n${offenders.join('\n')}`).toEqual([])
  })
})

// ──────────────────────────────────────────────────────────────────────
// Standalone write-API smoke — one APIRequestContext call per browser to
// confirm the write-cap token is valid and the PUT route serves from all
// three browser engines without any browser-specific quirks.
// ──────────────────────────────────────────────────────────────────────
test('write-api smoke: standalone APIRequestContext can PUT with write cap', async () => {
  const env = loadFixtureEnv()
  const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })
  try {
    const params = new URLSearchParams({
      session: 'playwright-test-session',
      path: `smoke-write-${Date.now()}.txt`,
      cap: env.writeCap,
    })
    const url = `${env.baseURL}/api/files/write?${params.toString()}`
    const resp = await ctx.put(url, {
      headers: { 'Content-Type': 'application/octet-stream' },
      data: 'smoke write via standalone ctx',
    })
    expect(resp.status(), 'write-cap PUT via standalone ctx returns 200').toBe(200)
  } finally {
    await ctx.dispose()
  }
})

// ──────────────────────────────────────────────────────────────────────
// SEC-07: CSRF Origin-mismatch — a PUT with a VALID files.write cap but a
// mismatched Origin header (https://evil.example.com) is rejected with
// HTTP 403.
//
// Behavior anchor: originAllowedForWrite (capability_mw.go:187-198) returns
// false when a present Origin != ws.BaseURL(). requireFilesWrite runs the
// Origin check AFTER the cap check (capability_mw.go:160-167), so the 403
// here proves CSRF rejection on a fully-capable token — NOT a cap failure.
// ──────────────────────────────────────────────────────────────────────
test('SEC-07: CSRF Origin-mismatch — valid write cap with evil.example.com Origin must 403', async () => {
  const env = loadFixtureEnv()
  const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })
  try {
    const params = new URLSearchParams({
      session: 'playwright-test-session',
      path: `origin-test-${Date.now()}.txt`,
      cap: env.writeCap, // VALID files.write cap — 403 proves Origin check, not cap failure
    })
    const url = `${env.baseURL}/api/files/write?${params.toString()}`
    const resp = await ctx.put(url, {
      headers: {
        'Content-Type': 'application/octet-stream',
        Origin: 'https://evil.example.com',
      },
      data: 'csrf attempt',
    })
    expect(resp.status(), 'mismatched Origin must 403 even with valid write cap').toBe(403)
  } finally {
    await ctx.dispose()
  }
})
