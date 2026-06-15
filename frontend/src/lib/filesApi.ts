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
  /**
   * Path prefix BEFORE the operation segment. Default `/api/files`.
   *
   * Phase 122-03 introduces this so the desktop GUI can proxy remote-session
   * file fetches through the local daemon's `/api/files/remote/{sid}/...`
   * routes (D-02 — no cross-origin browser fetches). For the local-loopback
   * and web-share paths the default value is correct.
   *
   * Leading slash required; trailing slash is stripped internally.
   */
  pathPrefix?: string
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

  /** 412 → If-Match precondition failed; another process changed the file (EDIT-08). */
  isConflict(): boolean {
    return this.status === 412
  }

  /** 409 → name collision on write/rename/mkdir/upload (EDIT-09/10). */
  isCollision(): boolean {
    return this.status === 409
  }

  /** 403 with body containing 'files.write' → cap is missing files.write perm. */
  isMissingFilesWritePerm(): boolean {
    if (this.status !== 403) return false
    return this.bodyText.toLowerCase().includes('files.write')
  }

  /**
   * 405 → remote peer has no write routes (old AgentHub version). RMW-04.
   * Scoped to write verbs only — read 405s remain generic (Open Q2 resolved).
   */
  isMethodNotAllowed(): boolean {
    return this.status === 405
  }
}

/**
 * Verbatim SC3 version-gate message surfaced when an upstream 405 indicates
 * the remote peer is running v3.4 (no write routes). MUST byte-match the
 * Go const `remotePeerOutdatedMessage` in
 * internal/tui/remote_files_client.go (RMW-04 cross-surface parity contract).
 */
export const REMOTE_PEER_OUTDATED_MESSAGE =
  'The remote session is running an older version of AgentHub that does not support file writes.'

/**
 * Client-side upload cap — must match the server's maxUploadBytes (50 MiB).
 * Single source of truth: import this constant everywhere the limit is
 * referenced on the frontend (WR-06). The Go test TestMaxUploadBytesMatch
 * in internal/webserver asserts the server value equals this number.
 */
export const MAX_UPLOAD_BYTES = 50 * 1024 * 1024

export class FilesApiClient {
  private readonly baseURL: string
  private readonly capToken: string | undefined
  private readonly pathPrefix: string

  constructor(cfg: FilesApiConfig) {
    // Strip trailing slash so URL building stays predictable.
    this.baseURL = cfg.baseURL.replace(/\/$/, '')
    this.capToken = cfg.capToken
    // Default to /api/files; strip trailing slash so concatenation with
    // /<op> stays unambiguous.
    this.pathPrefix = (cfg.pathPrefix ?? '/api/files').replace(/\/$/, '')
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
    const url = `${this.baseURL}${this.pathPrefix}/list?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url)
    const json = (await res.json()) as { entries: FileEntry[]; truncated: boolean }
    return {
      entries: json.entries ?? [],
      truncated: Boolean(json.truncated),
      refreshedAt: res.headers.get('X-Refreshed-At'),
    }
  }

  async statFile(sid: string, path: string): Promise<FileEntry> {
    const url = `${this.baseURL}${this.pathPrefix}/stat?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url)
    return (await res.json()) as FileEntry
  }

  /**
   * Read a text file. Returns the text body, the content-type, the byte size
   * (from content-length header), and the ETag (from ETag header — emitted by
   * Phase 125-01 handler.go; used as If-Match on subsequent writes, EDIT-05).
   */
  async readFileText(
    sid: string,
    path: string,
  ): Promise<{ text: string; contentType: string; size: number; etag: string | undefined }> {
    const url = `${this.baseURL}${this.pathPrefix}/read?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url)
    const contentType = res.headers.get('content-type') ?? 'application/octet-stream'
    const sizeHeader = res.headers.get('content-length')
    // IN-04: guard against NaN from malformed/absent header — Number.parseInt on
    // non-numeric input yields NaN, which propagates into size checks and humanSize.
    const sizeRaw = Number.parseInt(sizeHeader ?? '', 10)
    const size = Number.isFinite(sizeRaw) ? sizeRaw : 0
    const etag = res.headers.get('etag') ?? undefined
    const text = await res.text()
    return { text, contentType, size, etag }
  }

  /** HEAD probe for size and content-type without downloading bytes — used by PreviewPane size check. */
  async headFile(sid: string, path: string): Promise<{ size: number; contentType: string }> {
    const url = `${this.baseURL}${this.pathPrefix}/read?${this.buildQuery(sid, path).toString()}`
    const res = await this.fetchOrThrow(url, { method: 'HEAD' })
    const contentType = res.headers.get('content-type') ?? 'application/octet-stream'
    const sizeHeader = res.headers.get('content-length')
    // IN-04: guard against NaN from malformed/absent header.
    const sizeRaw = Number.parseInt(sizeHeader ?? '', 10)
    const size = Number.isFinite(sizeRaw) ? sizeRaw : 0
    return { size, contentType }
  }

  /**
   * Pure URL builder for <img src> — bytes never pass through JS memory.
   * Image preview pattern: `<img src={client.buildImageUrl(sid, path)} />`.
   */
  buildImageUrl(sid: string, path: string): string {
    return `${this.baseURL}${this.pathPrefix}/read?${this.buildQuery(sid, path).toString()}`
  }

  /** Pure URL builder for the download fallback button (unsupported / over-cap states). */
  buildDownloadUrl(sid: string, path: string): string {
    return `${this.baseURL}${this.pathPrefix}/read?${this.buildQuery(sid, path).toString()}`
  }

  /**
   * Write a file atomically via PUT /api/files/write.
   *
   * Sends the body as application/octet-stream. When `ifMatch` is provided,
   * includes it as the `If-Match` header for optimistic concurrency (EDIT-05).
   * The server (Phase 125-01) compares the validator against the on-disk file's
   * mtime+size and returns 412 Precondition Failed on mismatch (EDIT-08).
   *
   * Force-overwrite: pass ifMatch="*" or omit ifMatch entirely.
   * New file (no prior content): omit ifMatch.
   *
   * Throws FilesApiError(412) on conflict, FilesApiError(409) on name collision.
   */
  async writeFile(sid: string, path: string, body: BodyInit, ifMatch?: string): Promise<void> {
    const url = `${this.baseURL}${this.pathPrefix}/write?${this.buildQuery(sid, path).toString()}`
    const headers: Record<string, string> = { 'Content-Type': 'application/octet-stream' }
    if (ifMatch) headers['If-Match'] = ifMatch
    await this.fetchOrThrow(url, { method: 'PUT', headers, body })
  }

  /**
   * Delete a file or directory via DELETE /api/files/delete.
   *
   * Server performs recursive delete for directories within the os.Root-confined
   * sandbox (Phase 123 FSW-04). Throws FilesApiError(409) on name collision
   * (not expected for delete, but defensive).
   *
   * EDIT-09: client-side recursive file count walk happens BEFORE calling del
   * (in FileBrowserTab before opening DeleteConfirmModal). del itself has no
   * count semantics — the count is display-only.
   */
  async del(sid: string, path: string): Promise<void> {
    const url = `${this.baseURL}${this.pathPrefix}/delete?${this.buildQuery(sid, path).toString()}`
    await this.fetchOrThrow(url, { method: 'DELETE' })
  }

  /**
   * Rename or move a file/directory via POST /api/files/rename.
   *
   * Body is JSON {oldRel, newRel} (matches server renameRequest, write.go:171-174).
   * Cross-directory move is a rename with different directory components — the
   * server Rename method validates BOTH paths via validateAndClean (T-125-10).
   *
   * Throws FilesApiError(409) on name collision (fs.ErrExist → 409, write.go:244-245).
   */
  async rename(sid: string, oldRel: string, newRel: string): Promise<void> {
    const params = new URLSearchParams()
    params.set('session', sid)
    if (this.capToken) params.set('cap', this.capToken)
    const url = `${this.baseURL}${this.pathPrefix}/rename?${params.toString()}`
    await this.fetchOrThrow(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ oldRel, newRel }),
    })
  }

  /**
   * Create a directory via POST /api/files/mkdir.
   *
   * Throws FilesApiError(409) on name collision (directory already exists).
   */
  async mkdir(sid: string, path: string): Promise<void> {
    const url = `${this.baseURL}${this.pathPrefix}/mkdir?${this.buildQuery(sid, path).toString()}`
    await this.fetchOrThrow(url, { method: 'POST' })
  }

  /**
   * Probe the write route to determine if the cap token carries files.write perm.
   *
   * Uses a HEAD request to the write endpoint with an empty body; the server's
   * requireFilesWrite middleware runs before path resolution so a missing-perm
   * 403 fires before any path-related error. Used by useFilesCapability on the
   * web-share surface to resolve canWrite (Phase 125-02, EDIT-12).
   *
   * Throws FilesApiError(403) with body "files.write capability required" when
   * the cap lacks files.write. Throws FilesApiError(405) if HEAD is not
   * supported — callers treat any non-403-files.write error as canWrite=true
   * (server is the real authority; the UI gate is advisory).
   */
  async probeWrite(sid: string, path: string): Promise<void> {
    const url = `${this.baseURL}${this.pathPrefix}/write?${this.buildQuery(sid, path).toString()}`
    await this.fetchOrThrow(url, { method: 'HEAD' })
  }

  /**
   * Upload a single file via XMLHttpRequest (NOT fetch — XHR is required for
   * per-file upload.onprogress events; fetch has no upload-progress API).
   *
   * Server Handler.Upload (write.go) takes ONE multipart `file` part plus a
   * `dir` field. Multi-file upload = N calls, one file part each.
   * Cap token threads via URL query param (buildQuery includes it when present).
   *
   * EDIT-10 / WR-07: When `overwrite` is true, appends `overwrite=1` to the
   * form; the server skips the collision check and overwrites existing files.
   * Without it the server returns 409 when the target exists (EDIT-10 contract).
   *
   * @param sid        Session ID
   * @param dir        Target directory (relative to session cwd)
   * @param file       The File object to upload
   * @param onProgress Optional callback receiving integer 0-100 percent
   * @param overwrite  When true, send overwrite=1 to skip the 409 collision check
   * @returns Promise that resolves on 2xx; rejects with FilesApiError on
   *   409 (collision → isCollision()) or 413 (over-cap → isOverCap()).
   */
  uploadFile(
    sid: string,
    dir: string,
    file: File,
    onProgress?: (pct: number) => void,
    overwrite?: boolean,
  ): Promise<void> {
    // Build URL — session + optional cap go in query params (no auth header).
    const params = new URLSearchParams()
    params.set('session', sid)
    if (this.capToken) params.set('cap', this.capToken)
    const url = `${this.baseURL}${this.pathPrefix}/upload?${params.toString()}`

    const formData = new FormData()
    formData.append('dir', dir)
    formData.append('file', file)
    if (overwrite) formData.append('overwrite', '1')

    return new Promise<void>((resolve, reject) => {
      const xhr = new XMLHttpRequest()

      // Per-file upload progress events — the reason for XHR over fetch.
      xhr.upload.addEventListener('progress', (evt: ProgressEvent) => {
        if (evt.lengthComputable && onProgress) {
          const pct = Math.round((evt.loaded / evt.total) * 100)
          onProgress(pct)
        }
      })

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve()
        } else {
          reject(new FilesApiError(xhr.status, xhr.responseText ?? ''))
        }
      })

      xhr.addEventListener('error', () => {
        reject(new FilesApiError(0, 'network error'))
      })

      xhr.addEventListener('abort', () => {
        reject(new FilesApiError(0, 'upload aborted'))
      })

      xhr.open('POST', url)
      xhr.send(formData)
    })
  }
}
