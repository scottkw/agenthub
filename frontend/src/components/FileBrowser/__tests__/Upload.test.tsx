/**
 * Upload.test.tsx — Phase 125-05 TDD tests for:
 *   Task 1: filesApi.ts uploadFile() — XHR, FormData(file+dir), 409/413 errors
 *   Task 2: UploadQueuePanel + UploadDropOverlay + BreadcrumbBar Upload button
 *
 * Pattern: vitest source-inspection (?raw) + XHR mock for functional upload tests.
 * No @testing-library/react needed — matches existing test patterns in this project.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// ─── Source-inspection imports ────────────────────────────────────────────────
import filesApiRaw from '../../../lib/filesApi.ts?raw'
import useFilesWriteRaw from '../../../lib/useFilesWrite.ts?raw'
import uploadQueuePanelRaw from '../UploadQueuePanel.tsx?raw'
import uploadDropOverlayRaw from '../UploadDropOverlay.tsx?raw'
import breadcrumbBarRaw from '../BreadcrumbBar.tsx?raw'

// ─── Task 1: filesApi.uploadFile() — source-level assertions ─────────────────

describe('filesApi.ts — uploadFile() source assertions (EDIT-10)', () => {
  it('contains XMLHttpRequest (not fetch) for upload', () => {
    expect(filesApiRaw).toContain('XMLHttpRequest')
  })

  it('uses onprogress for per-file N% events', () => {
    expect(filesApiRaw).toContain('onprogress')
  })

  it('sends FormData with file + dir fields', () => {
    expect(filesApiRaw).toContain('FormData')
    expect(filesApiRaw).toContain("'dir'")
    expect(filesApiRaw).toContain("'file'")
  })

  it('has uploadFile method that takes (sid, dir, file, onProgress)', () => {
    expect(filesApiRaw).toContain('uploadFile(')
    expect(filesApiRaw).toContain('onProgress')
  })

  it('maps status 409 to a collision-distinguishable error', () => {
    expect(filesApiRaw).toContain('409')
    expect(filesApiRaw).toContain('isCollision')
  })

  it('maps status 413 to an over-cap-distinguishable error', () => {
    expect(filesApiRaw).toContain('413')
    expect(filesApiRaw).toContain('isOverCap')
  })

  it('threads cap token via URL (query param, not header)', () => {
    // uploadFile should use buildQuery or capToken in the URL it constructs.
    expect(filesApiRaw).toContain('uploadFile')
    expect(filesApiRaw).toContain('capToken')
  })
})

// ─── Task 1: useFilesWrite.upload real implementation ─────────────────────────

describe('useFilesWrite.ts — upload real implementation (Plan 05)', () => {
  it('upload is no longer a stub (no "not implemented" throw)', () => {
    expect(useFilesWriteRaw).not.toContain("throw new Error('upload not implemented")
  })

  it('calls client.uploadFile in the upload implementation', () => {
    expect(useFilesWriteRaw).toContain('uploadFile')
  })

  it('exposes per-file progress via onProgress callback through to uploadFile', () => {
    // The hook's upload function should wire onProgress through to the API call.
    expect(useFilesWriteRaw).toContain('onProgress')
  })
})

// ─── Task 1: XHR functional test — uploadFile sends correct request ───────────

describe('filesApi.uploadFile() — XHR functional test', () => {
  // Shared mutable state captured from XHR instance construction.
  let xhrOpenCalls: Array<[string, string]> = []
  let xhrSendArg: unknown = null
  // Pointer to the ACTUAL XHR instance created by uploadFile() — lets us
  // mutate `status`/`responseText` on the real object before firing 'load'.
  let xhrInstanceRef: Record<string, unknown> | null = null

  beforeEach(() => {
    xhrOpenCalls = []
    xhrSendArg = null
    xhrInstanceRef = null

    vi.stubGlobal('XMLHttpRequest', class {
      status = 200
      responseText = ''
      _listeners: Record<string, Function> = {}
      _uploadListeners: Record<string, Function> = {}
      upload = {
        addEventListener: (ev: string, fn: Function) => {
          this._uploadListeners[ev] = fn
        },
      }
      constructor() {
        // Store a reference to THIS instance so tests can mutate it.
        // eslint-disable-next-line @typescript-eslint/no-this-alias
        xhrInstanceRef = this as unknown as Record<string, unknown>
      }
      open(method: string, url: string) {
        xhrOpenCalls.push([method, url])
      }
      send(arg: unknown) {
        xhrSendArg = arg
      }
      setRequestHeader() {}
      addEventListener(ev: string, fn: Function) {
        this._listeners[ev] = fn
      }
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  /** Helper: set status on the captured XHR instance and fire its 'load' listener. */
  function fireLoad(status: number, responseText = '') {
    if (!xhrInstanceRef) throw new Error('XHR instance not captured')
    xhrInstanceRef['status'] = status
    xhrInstanceRef['responseText'] = responseText
    const listeners = xhrInstanceRef['_listeners'] as Record<string, Function>
    const fn = listeners['load']
    if (fn) fn.call(xhrInstanceRef, new Event('load'))
  }

  /** Helper: fire the upload 'progress' listener. */
  function fireProgress(loaded: number, total: number) {
    if (!xhrInstanceRef) throw new Error('XHR instance not captured')
    const uploadListeners = xhrInstanceRef['_uploadListeners'] as Record<string, Function>
    const fn = uploadListeners['progress']
    if (fn) fn.call(null, { lengthComputable: true, loaded, total })
  }

  it('uses POST for the upload request', async () => {
    const { FilesApiClient } = await import('../../../lib/filesApi')
    const client = new FilesApiClient({ baseURL: 'http://localhost:8080' })
    const file = new File(['hello'], 'test.txt', { type: 'text/plain' })

    const promise = client.uploadFile('sid1', 'docs', file)
    fireLoad(200)
    await promise

    expect(xhrOpenCalls.length).toBe(1)
    expect(xhrOpenCalls[0][0]).toBe('POST')
  })

  it('URL contains session, optional cap, and /upload path', async () => {
    const { FilesApiClient } = await import('../../../lib/filesApi')
    const client = new FilesApiClient({
      baseURL: 'http://localhost:8080',
      capToken: 'tok123',
    })
    const file = new File(['data'], 'file.txt')

    const promise = client.uploadFile('sess99', 'images', file)
    fireLoad(200)
    await promise

    const url = xhrOpenCalls[0][1]
    expect(url).toContain('session=sess99')
    expect(url).toContain('cap=tok123')
    expect(url).toContain('/upload')
  })

  it('sends FormData with file part and dir field', async () => {
    const { FilesApiClient } = await import('../../../lib/filesApi')
    const client = new FilesApiClient({ baseURL: 'http://localhost:8080' })
    const file = new File(['abc'], 'notes.txt')

    const promise = client.uploadFile('s1', 'my/dir', file)
    fireLoad(200)
    await promise

    expect(xhrSendArg).toBeInstanceOf(FormData)
    const fd = xhrSendArg as FormData
    expect(fd.get('dir')).toBe('my/dir')
    const filePart = fd.get('file')
    expect(filePart).toBeInstanceOf(File)
    expect((filePart as File).name).toBe('notes.txt')
  })

  it('fires onProgress with percentage from upload.onprogress', async () => {
    const { FilesApiClient } = await import('../../../lib/filesApi')
    const client = new FilesApiClient({ baseURL: 'http://localhost:8080' })
    const file = new File(['data'], 'big.bin')
    const captured: number[] = []

    const promise = client.uploadFile('s1', '.', file, (pct) => captured.push(pct))

    fireProgress(50, 100)
    fireLoad(200)
    await promise

    expect(captured).toContain(50)
  })

  it('rejects with FilesApiError(isCollision) on 409', async () => {
    const mod = await import('../../../lib/filesApi')
    const client = new mod.FilesApiClient({ baseURL: 'http://localhost:8080' })
    const file = new File(['x'], 'dup.txt')

    const promise = client.uploadFile('s1', '.', file)
    fireLoad(409, 'file exists')

    const err = await promise.then(() => null).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(mod.FilesApiError)
    expect((err as InstanceType<typeof mod.FilesApiError>).isCollision()).toBe(true)
  })

  it('rejects with FilesApiError(isOverCap) on 413', async () => {
    const mod = await import('../../../lib/filesApi')
    const client = new mod.FilesApiClient({ baseURL: 'http://localhost:8080' })
    const file = new File(['x'], 'huge.zip')

    const promise = client.uploadFile('s1', '.', file)
    fireLoad(413, 'too large')

    const err = await promise.then(() => null).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(mod.FilesApiError)
    expect((err as InstanceType<typeof mod.FilesApiError>).isOverCap()).toBe(true)
  })
})

// ─── Task 2: UploadQueuePanel — source-level assertions ──────────────────────

describe('UploadQueuePanel.tsx — source assertions', () => {
  it('contains "Uploading" for the queue title', () => {
    expect(uploadQueuePanelRaw).toContain('Uploading')
  })

  it('shows N% progress (% symbol in progress text)', () => {
    // The component renders something like `{item.progress}%`
    expect(uploadQueuePanelRaw).toContain('%')
  })

  it('contains "Done" text for done status (colorblind contract)', () => {
    expect(uploadQueuePanelRaw).toContain('Done')
  })

  it('contains "Failed — try again" text for failed status (colorblind contract)', () => {
    expect(uploadQueuePanelRaw).toContain('Failed — try again')
  })

  it('contains over-cap skip message verbatim', () => {
    expect(uploadQueuePanelRaw).toContain('is too large (max 50 MB) and was skipped')
  })

  it('contains Replace? / Replace button for 409 collision per-row', () => {
    expect(uploadQueuePanelRaw).toContain('Replace')
  })

  it('has role="status" + aria-live for progress announcements', () => {
    expect(uploadQueuePanelRaw).toContain('role="status"')
    expect(uploadQueuePanelRaw).toContain('aria-live')
  })

  it('uses CheckCircleIcon for done state (colorblind contract)', () => {
    expect(uploadQueuePanelRaw).toContain('CheckCircleIcon')
  })

  it('uses ExclamationTriangleIcon for failed state (colorblind contract)', () => {
    expect(uploadQueuePanelRaw).toContain('ExclamationTriangleIcon')
  })
})

// ─── Task 2: UploadDropOverlay — source-level assertions ─────────────────────

describe('UploadDropOverlay.tsx — source assertions', () => {
  it('contains drop prompt verbatim: "Drop files to upload here"', () => {
    expect(uploadDropOverlayRaw).toContain('Drop files to upload here')
  })

  it('has onDrop + onDragLeave props', () => {
    expect(uploadDropOverlayRaw).toContain('onDrop')
    expect(uploadDropOverlayRaw).toContain('onDragLeave')
  })

  it('uses ArrowUpTrayIcon', () => {
    expect(uploadDropOverlayRaw).toContain('ArrowUpTrayIcon')
  })

  it('has a data-testid for the overlay', () => {
    expect(uploadDropOverlayRaw).toContain('upload-drop-overlay')
  })
})

// ─── Task 2: BreadcrumbBar Upload button — source assertions ──────────────────

describe('BreadcrumbBar.tsx — Upload button (EDIT-10)', () => {
  it('accepts an onUpload prop', () => {
    expect(breadcrumbBarRaw).toContain('onUpload')
  })

  it('renders Upload button inside canWrite guard', () => {
    // The button must be gated on canWrite
    expect(breadcrumbBarRaw).toContain('ArrowUpTrayIcon')
    expect(breadcrumbBarRaw).toContain('canWrite')
  })

  it('Upload button has aria-label containing "Upload"', () => {
    expect(breadcrumbBarRaw).toMatch(/aria-label="[^"]*Upload[^"]*"/)
  })
})
