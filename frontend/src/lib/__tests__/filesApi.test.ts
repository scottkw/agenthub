import { describe, it, expect, beforeEach, vi } from 'vitest'
import { FilesApiClient, FilesApiError } from '../filesApi'

// Phase 120-02 Task 3 — FilesApiClient unit tests.
// Mock global fetch with vi.stubGlobal. Asserts URL construction, header parsing,
// error mapping per HTTP status, and that no base64/data-URL encoding of bytes
// ever happens (Pitfall 10).

function mockFetchOk(body: unknown, headers: Record<string, string> = {}): void {
  const headerObj = new Headers(headers)
  const isString = typeof body === 'string'
  const responseBody = isString ? (body as string) : JSON.stringify(body)
  const contentType = headerObj.get('content-type') ?? (isString ? 'text/plain' : 'application/json')
  if (!headerObj.has('content-type')) headerObj.set('content-type', contentType)

  vi.stubGlobal(
    'fetch',
    vi.fn(async () =>
      ({
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: headerObj,
        text: async () => responseBody,
        json: async () => (isString ? JSON.parse(responseBody) : body),
      } as unknown as Response),
    ),
  )
}

function mockFetchError(status: number, bodyText: string): void {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () =>
      ({
        ok: false,
        status,
        statusText: 'error',
        headers: new Headers({ 'content-type': 'text/plain' }),
        text: async () => bodyText,
        json: async () => {
          throw new Error('not json')
        },
      } as unknown as Response),
    ),
  )
}

beforeEach(() => {
  vi.unstubAllGlobals()
})

describe('FilesApiClient — URL construction', () => {
  it('constructs list URL with session and path encoded (no cap)', async () => {
    mockFetchOk({ entries: [], truncated: false })
    const client = new FilesApiClient({ baseURL: 'http://host' })
    await client.listFiles('sid', 'src/a.ts')
    const fetchMock = global.fetch as unknown as ReturnType<typeof vi.fn>
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toBe('http://host/api/files/list?session=sid&path=src%2Fa.ts')
  })

  it('appends cap when provided', async () => {
    mockFetchOk({ entries: [], truncated: false })
    const client = new FilesApiClient({ baseURL: 'http://host', capToken: 'tok' })
    await client.listFiles('sid', '.')
    const fetchMock = global.fetch as unknown as ReturnType<typeof vi.fn>
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain('&cap=tok')
  })

  it('URL encodes paths with + & = and space', async () => {
    mockFetchOk({ entries: [], truncated: false })
    const client = new FilesApiClient({ baseURL: 'http://host' })
    await client.listFiles('sid', '1 + 1 & 2 = 3.txt')
    const fetchMock = global.fetch as unknown as ReturnType<typeof vi.fn>
    const url = fetchMock.mock.calls[0][0] as string
    // URLSearchParams encodes space as '+', '+' as '%2B', '&' as '%26', '=' as '%3D'
    const params = new URL(url).searchParams
    expect(params.get('path')).toBe('1 + 1 & 2 = 3.txt')
  })
})

describe('FilesApiClient — response parsing', () => {
  it('listFiles parses JSON entries and truncated flag and X-Refreshed-At header', async () => {
    mockFetchOk(
      {
        entries: [
          { name: 'a.ts', size: 0, mtime: '', mode: 0o644, isDir: false, isSymlink: false, isBinary: false },
        ],
        truncated: true,
      },
      { 'X-Refreshed-At': '2026-05-20T10:00:00Z', 'content-type': 'application/json' },
    )
    const client = new FilesApiClient({ baseURL: 'http://host' })
    const res = await client.listFiles('sid', '.')
    expect(res.entries.length).toBe(1)
    expect(res.entries[0].name).toBe('a.ts')
    expect(res.truncated).toBe(true)
    expect(res.refreshedAt).toBe('2026-05-20T10:00:00Z')
  })

  it('statFile returns single FileEntry', async () => {
    mockFetchOk(
      { name: 'b.md', size: 100, mtime: '2026-05-20T10:00:00Z', mode: 0o644, isDir: false, isSymlink: false, isBinary: false, mime: 'text/markdown' },
      { 'content-type': 'application/json' },
    )
    const client = new FilesApiClient({ baseURL: 'http://host' })
    const entry = await client.statFile('sid', 'b.md')
    expect(entry.name).toBe('b.md')
    expect(entry.size).toBe(100)
    expect(entry.mime).toBe('text/markdown')
  })

  it('readFileText returns text + content-type + size', async () => {
    mockFetchOk('hello world', { 'content-type': 'text/plain', 'content-length': '11' })
    const client = new FilesApiClient({ baseURL: 'http://host' })
    const result = await client.readFileText('sid', 'a.txt')
    expect(result.text).toBe('hello world')
    expect(result.contentType).toBe('text/plain')
    expect(result.size).toBe(11)
  })

  it('headFile uses HEAD method', async () => {
    mockFetchOk('', { 'content-type': 'image/png', 'content-length': '2048' })
    const client = new FilesApiClient({ baseURL: 'http://host' })
    const result = await client.headFile('sid', 'a.png')
    const fetchMock = global.fetch as unknown as ReturnType<typeof vi.fn>
    const opts = fetchMock.mock.calls[0][1] as RequestInit
    expect(opts.method).toBe('HEAD')
    expect(result.size).toBe(2048)
    expect(result.contentType).toBe('image/png')
  })

  it('buildImageUrl + buildDownloadUrl are pure URL builders (no fetch)', () => {
    mockFetchOk('') // ensure fetch is mocked but should NOT be called
    const client = new FilesApiClient({ baseURL: 'http://host', capToken: 'tok' })
    const imgUrl = client.buildImageUrl('sid', 'img/p.png')
    const dlUrl = client.buildDownloadUrl('sid', 'doc.pdf')
    expect(imgUrl).toBe('http://host/api/files/read?session=sid&path=img%2Fp.png&cap=tok')
    expect(dlUrl).toBe('http://host/api/files/read?session=sid&path=doc.pdf&cap=tok')
    const fetchMock = global.fetch as unknown as ReturnType<typeof vi.fn>
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('FilesApiError', () => {
  it('throws FilesApiError on non-2xx with correct status and body', async () => {
    mockFetchError(404, 'session not found')
    const client = new FilesApiClient({ baseURL: 'http://host' })
    let err: unknown = null
    try {
      await client.listFiles('sid', '.')
    } catch (e) {
      err = e
    }
    expect(err).toBeInstanceOf(FilesApiError)
    expect((err as FilesApiError).status).toBe(404)
    expect((err as FilesApiError).bodyText).toBe('session not found')
  })

  it('isMissingFilesReadPerm — 403 with body containing files.read (case-insensitive) → true', () => {
    expect(new FilesApiError(403, 'cap missing files.read perm').isMissingFilesReadPerm()).toBe(true)
    expect(new FilesApiError(403, 'CAP MISSING FILES.READ PERM').isMissingFilesReadPerm()).toBe(true)
    expect(new FilesApiError(403, '{"error":"missing perm files.read"}').isMissingFilesReadPerm()).toBe(true)
    expect(new FilesApiError(403, 'permission denied for path').isMissingFilesReadPerm()).toBe(false)
    expect(new FilesApiError(401, 'missing files.read').isMissingFilesReadPerm()).toBe(false)
  })

  it('isUnauthorized — 401 → true; 403 → false', () => {
    expect(new FilesApiError(401, '').isUnauthorized()).toBe(true)
    expect(new FilesApiError(403, '').isUnauthorized()).toBe(false)
  })

  it('isForbidden — 403 → true; 401 → false', () => {
    expect(new FilesApiError(403, '').isForbidden()).toBe(true)
    expect(new FilesApiError(401, '').isForbidden()).toBe(false)
  })

  it('isOverCap — 413 → true; others → false', () => {
    expect(new FilesApiError(413, 'too large').isOverCap()).toBe(true)
    expect(new FilesApiError(403, '').isOverCap()).toBe(false)
    expect(new FilesApiError(404, '').isOverCap()).toBe(false)
  })

  it('isNotFound — 404 → true', () => {
    expect(new FilesApiError(404, '').isNotFound()).toBe(true)
    expect(new FilesApiError(403, '').isNotFound()).toBe(false)
  })
})
