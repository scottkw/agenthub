// Phase 120-02 Task 3 — Typed API client for the Phase 118/119 /api/files/* routes.
//
// Single point of construction for ALL browser→/api/files/* traffic. Concentrates:
//   - URL building (URLSearchParams; no manual string concat — Pitfall 7)
//   - Cap-token handling (optional; daemon-loopback omits, webserver-HTTPS passes)
//   - Error mapping (FilesApiError surfaces typed predicates the UI dispatches on)
//
// Wire shape mirrors internal/files/types.go exactly (lowercase JSON tags; see
// Phase 118 PR — `name`, `size`, `mtime`, `mode`, `isDir`, `isSymlink`, `isBinary`,
// optional `mime`). DO NOT base64-encode bytes anywhere — image previews use
// /api/files/read as a direct URL bound to <img src> (Pitfall 10).

export interface FileEntry {
  name: string
  size: number
  mtime: string
  mode: number
  isDir: boolean
  isSymlink: boolean
  isBinary: boolean
  mime?: string
}

export interface FileListResponse {
  entries: FileEntry[]
  truncated: boolean
  /** Server-reported refresh timestamp from the X-Refreshed-At response header. */
  refreshedAt: string | null
}

export interface FilesApiConfig {
  /** Base URL with no trailing slash, e.g. `http://127.0.0.1:8080` or `https://hub.example.com`. */
  baseURL: string
  /** Optional capability token. Omit on daemon-loopback (no-auth path); pass on webserver-HTTPS. */
  capToken?: string
}

/**
 * Typed error wrapping non-2xx /api/files/* responses. UI components branch on
 * the `is*()` predicates rather than raw status codes to keep dispatch readable
 * and survive future status-code changes.
 *
 * `bodyText` is the raw response body — used for the substring match in
 * `isMissingFilesReadPerm()` (case-insensitive; tolerant of JSON-wrapped messages).
 */
export class FilesApiError extends Error {
  constructor(public readonly status: number, public readonly bodyText: string) {
    super(`FilesApiError ${status}: ${bodyText.slice(0, 200)}`)
    this.name = 'FilesApiError'
  }

  /** 403 with body containing 'files.read' (case-insensitive) → viewer-cap missing perm. */
  isMissingFilesReadPerm(): boolean {
    if (this.status !== 403) return false
    return this.bodyText.toLowerCase().includes('files.read')
  }

  isUnauthorized(): boolean {
    return this.status === 401
  }

  isForbidden(): boolean {
    return this.status === 403
  }

  isNotFound(): boolean {
    return this.status === 404
  }

  /** 413 → file > 5 MiB cap (Phase 118 FS-08). */
  isOverCap(): boolean {
    return this.status === 413
  }
}

export class FilesApiClient {
  private readonly baseURL: string
  private readonly capToken: string | undefined

  constructor(cfg: FilesApiConfig) {
    // Strip trailing slash so URL building stays predictable.
    this.baseURL = cfg.baseURL.replace(/\/$/, '')
    this.capToken = cfg.capToken
  }

  /** Build a query string with session, path, and optional cap — single source of truth. */
  private buildQuery(sid: string, path: string): URLSearchParams {
    const params = new URLSearchParams()
    params.set('session', sid)
    params.set('path', path)
    if (this.capToken) params.set('cap', this.capToken)
    return params
  }

  private async fetchOrThrow(url: string, init?: RequestInit): Promise<Response> {
    const res = await fetch(url, init)
    if (!res.ok) {
      let body = ''
      try {
        body = await res.text()
      } catch {
        // ignore — body may be unavailable on HEAD/network errors
      }
      throw new FilesApiError(res.status, body)
    }
    return res
  }

  async listFiles(sid: string, path: string): Promise<FileListResponse> {
    const url = `${this.baseURL}/api/files/list?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url)
    const json = (await res.json()) as { entries: FileEntry[]; truncated: boolean }
    return {
      entries: json.entries ?? [],
      truncated: Boolean(json.truncated),
      refreshedAt: res.headers.get('X-Refreshed-At'),
    }
  }

  async statFile(sid: string, path: string): Promise<FileEntry> {
    const url = `${this.baseURL}/api/files/stat?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url)
    return (await res.json()) as FileEntry
  }

  /**
   * Read a text file. Returns the text body, the content-type, and the byte
   * size (from content-length header — preflight-cheap for callers since the
   * server has already produced the response).
   */
  async readFileText(
    sid: string,
    path: string,
  ): Promise<{ text: string; contentType: string; size: number }> {
    const url = `${this.baseURL}/api/files/read?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url)
    const contentType = res.headers.get('content-type') ?? 'application/octet-stream'
    const sizeHeader = res.headers.get('content-length')
    const size = sizeHeader ? Number.parseInt(sizeHeader, 10) : 0
    const text = await res.text()
    return { text, contentType, size }
  }

  /** HEAD probe for size and content-type without downloading bytes — used by PreviewPane size check. */
  async headFile(sid: string, path: string): Promise<{ size: number; contentType: string }> {
    const url = `${this.baseURL}/api/files/read?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url, { method: 'HEAD' })
    const contentType = res.headers.get('content-type') ?? 'application/octet-stream'
    const sizeHeader = res.headers.get('content-length')
    const size = sizeHeader ? Number.parseInt(sizeHeader, 10) : 0
    return { size, contentType }
  }

  /**
   * Pure URL builder for <img src> — bytes never pass through JS memory.
   * Image preview pattern: `<img src={client.buildImageUrl(sid, path)} />`.
   */
  buildImageUrl(sid: string, path: string): string {
    return `${this.baseURL}/api/files/read?${this.buildQuery(sid, path).toString()}`
  }

  /** Pure URL builder for the download fallback button (unsupported / over-cap states). */
  buildDownloadUrl(sid: string, path: string): string {
    return `${this.baseURL}/api/files/read?${this.buildQuery(sid, path).toString()}`
  }
}
