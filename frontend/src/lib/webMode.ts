/**
 * Phase 120-06 — Web-vs-Desktop mode detection.
 *
 * Single source of truth for "is this React shell running in a regular browser
 * via /app/ (web-share viewer) vs inside the Wails desktop runtime?" Every
 * mode-conditional code path in App.tsx (Wails RPC suite, file-browser tab
 * baseURL, capability token sourcing) MUST consult `detectMode()` rather than
 * inlining its own `window.location.pathname.startsWith(...)` check.
 *
 * Rationale: 120-VERIFICATION.md Human Verification #2 flagged that loading
 * `/app/?session=…&cap=…` in a regular browser produced a partially-functional
 * shell because `App.tsx` was Wails-coupled (always called `GetRelayPort()` /
 * `GetWebServerMode()` and used `fbBaseURL = http://127.0.0.1:${relayPort}`).
 * This module is the canonical signal that lets App.tsx skip the Wails RPC
 * suite and drive `fbBaseURL` / `capToken` from URL params instead.
 *
 * The canonical signal is `window.location.pathname.startsWith('/app/')`
 * (or exact `/app`) — `internal/webserver/server.go:566` mounts the SPA route
 * at exactly that prefix. We deliberately do NOT consult `navigator.userAgent`
 * or Wails-only globals (PRD decision): pathname is opaque to the user and
 * mirrors the server-side route layout.
 *
 * This module is a pure-function file. No React imports, no Wails imports, no
 * side effects on import — safe to load from any code path (including
 * Wails-side code that may not yet have window.location populated).
 */

export type AppMode = 'desktop' | 'web'

export interface WebModeParams {
  /** Session id from `?session=`; null if absent, empty, or whitespace-only. */
  sessionId: string | null
  /** Capability token from `?cap=`; null if absent, empty, or whitespace-only. */
  capToken: string | null
}

/**
 * detectMode returns 'web' when running in a regular browser under the /app/
 * route (the web-share viewer entry), or 'desktop' otherwise (Wails-served
 * root, dev server, etc).
 *
 * The signal is pathname-based:
 *   - `/app`        → web
 *   - `/app/`       → web
 *   - `/app/<...>`  → web
 *   - anything else → desktop
 *
 * Note: `/apps`, `/foo/app/`, etc. are NOT web mode — the prefix match
 * respects the `/app` boundary (either an exact equal or followed by `/`).
 */
export function detectMode(loc: Location = window.location): AppMode {
  const path = loc.pathname
  if (path === '/app') return 'web'
  if (path.startsWith('/app/')) return 'web'
  return 'desktop'
}

/**
 * readWebModeParams parses `?session=` and `?cap=` from the URL search string.
 * Missing, empty, and whitespace-only values all collapse to `null` so callers
 * can use a single `value ?? fallback` idiom without distinguishing the cases.
 *
 * URLSearchParams handles percent-decoding once; we do not double-decode.
 */
export function readWebModeParams(loc: Location = window.location): WebModeParams {
  const params = new URLSearchParams(loc.search)
  const session = (params.get('session') ?? '').trim()
  const cap = (params.get('cap') ?? '').trim()
  return {
    sessionId: session === '' ? null : session,
    capToken: cap === '' ? null : cap,
  }
}
