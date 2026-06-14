---
phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes
plan: 02
subsystem: daemon
tags: [go, http, redirect, file-browser, typescript, react]

# Dependency graph
requires:
  - phase: 122-03
    provides: ExchangeJoinCodeAtURL skeleton + remoteFilesHTTPClient; RegisterRemoteCap client
  - phase: 120-04
    provides: WR-01..05 Phase 120 file-browser hardening (confirmed all landed)
provides:
  - "303-aware ExchangeJoinCodeAtURL with dedicated http.Client (ErrUseLastResponse) parsing ?cap= from Location header"
  - "Error-substring contract preserved: expired/invalid/not-found/session-gone"
  - "TD-5 (FSW-10) closed — remote session cap acquisition unblocked for Phase 128"
  - "TD-4 (FSW-11) verified-already-satisfied: WR-01/02/03/04/05 all confirmed in tree"
affects: [128-remote-write, phase-122-remote-browse]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dedicated http.Client inside function for CheckRedirect: ErrUseLastResponse — never mutate the shared package-level client"
    - "303 Location header parsing: /join?error=<kind> error shape handled before ?cap= success shape"

key-files:
  created: []
  modified:
    - internal/daemon/client_remote_files.go
    - internal/daemon/client_test.go

key-decisions:
  - "Dedicated http.Client constructed inside ExchangeJoinCodeAtURL (not mutating remoteFilesHTTPClient) — preserves RegisterRemoteCap behavior (T-123-09)"
  - "Removed encoding/json import no longer needed after rewriting success path from JSON-decode to Location-header parse"
  - "WR-01/02/03/04/05 all confirmed already-satisfied from Phase 120 — zero TD-4 changes required"

patterns-established:
  - "TD-5 pattern: when a remote endpoint returns 303+Location (not a JSON body), use a dedicated http.Client with ErrUseLastResponse and parse the Location header"

requirements-completed: [FSW-10, FSW-11]

# Metrics
duration: 28min
completed: 2026-06-14
---

# Phase 123 Plan 02: TD-5 + TD-4 Summary

**303-aware ExchangeJoinCodeAtURL with dedicated CheckRedirect client parsing ?cap= from Location header, unblocking Phase 128 remote-write cap acquisition; WR-01..05 TD-4 hardening confirmed already-satisfied from Phase 120**

## Performance

- **Duration:** ~28 min
- **Started:** 2026-06-14T14:51:00Z
- **Completed:** 2026-06-14T15:19:42Z
- **Tasks:** 2 (Task 1 = TDD fix; Task 2 = verify-only)
- **Files modified:** 2 (Go daemon client only; no frontend changes needed)

## Accomplishments

- Fixed `ExchangeJoinCodeAtURL` in `internal/daemon/client_remote_files.go`: the function was JSON-decoding a bodyless 303 response (silent failure), now correctly parses the 303 Location `?cap=<token>` using a dedicated http.Client with `CheckRedirect: ErrUseLastResponse`
- Preserved the existing error-substring contract (`expired`/`invalid`/`not-found`/`session-gone`) the modal UI pivots on — both the `/join?error=<kind>` Location shape and the existing 4xx status-code mapping produce these substrings
- Added 5 test cases to `internal/daemon/client_test.go` verifying 303 success, no auto-follow, error-location x4 substrings, empty-cap fallback, and shared-client immutability — all pass under `-race`
- Confirmed WR-01/02/03/04/05 (TD-4 FSW-11) fully implemented from Phase 120; no further changes required

## Task Commits

1. **Task 1: TD-5 — 303-aware ExchangeJoinCodeAtURL (FSW-10)** - `f5ed52e` (fix+test TDD)
2. **Task 2: TD-4 — WR-01/02/03/04/05 verify-only** — no commit (all items already implemented; zero changes made)

## Files Created/Modified

- `internal/daemon/client_remote_files.go` — Rewrote `ExchangeJoinCodeAtURL` success path: dedicated http.Client with `ErrUseLastResponse`, detect `http.StatusSeeOther`, parse Location header (`/join?error=<kind>` → error, `/sessions/<id>?cap=<tok>` → success); removed unused `encoding/json` import
- `internal/daemon/client_test.go` — Added `TestExchangeJoinCode_303Success`, `TestExchangeJoinCode_NoAutoFollow`, `TestExchangeJoinCode_ErrorLocation` (4 subtests), `TestExchangeJoinCode_EmptyCap`, `TestExchangeJoinCode_SharedClientUnchanged`

## Decisions Made

- **Dedicated client, not shared**: Constructed a fresh `&http.Client{CheckRedirect: ErrUseLastResponse}` inside `ExchangeJoinCodeAtURL` rather than adding `CheckRedirect` to the package-level `remoteFilesHTTPClient`. The shared client is also used by `RegisterRemoteCap` (and future callers) — adding `ErrUseLastResponse` there would prevent it from following legitimate redirects. This mirrors the TUI reference implementation in `joincode_prompt.go:155-164` exactly.
- **TLS config preserved**: The dedicated client uses `InsecureSkipVerify: true` (with the same `//nolint:gosec` comment) to match the tailnet self-signed cert precedent in the original shared client. The TUI reference uses `MinVersion: tls.VersionTLS12` without `InsecureSkipVerify` — the daemon client's existing permissive TLS convention was kept for consistency with `remoteFilesHTTPClient` since both talk to the same tailnet peers.
- **WR-03 coverage confirmed complete**: `navigateInto` in `FileBrowserTab.tsx:399-422` rejects `name === ''`, `'.'`, `'..'`, and any name containing `/` or `\` — the last guard catches leading-slash names (e.g. `/etc`) that would otherwise synthesize an absolute path via `joinPath`. No missing code path found.

## WR-01/WR-02 Verified-Already-Satisfied

**WR-01** (`internal/webserver/server.go:583-598`, comment "Phase 120 WR-01"):
The `GET /app/` handler checks `rel != "" && rel[len(rel)-1] == '/'` and falls through to `serveIndex` for any trailing-slash sub-path, blocking directory-index responses. This satisfies the requirement to prevent `http.FileServerFS` from rendering a styled HTML directory listing that would leak bundle file names and mtimes. **Already done. Not re-touched (Chesterton's Fence).**

**WR-02** (`internal/webserver/server.go:556-565`, comment "Phase 120 WR-02"):
The comment documents the caching policy: `index.html` gets `Cache-Control: no-store` via `serveIndex`; hashed JS/CSS bundle assets inherit `FileServerFS` defaults (Last-Modified) with no explicit Cache-Control header — safe because Vite content-hashes every asset URL, making the hash the cache key. **Already done. Not re-touched (Chesterton's Fence).**

**WR-03** (`frontend/src/components/FileBrowserTab.tsx:399-422`, comment "Phase 120 WR-03"):
`navigateInto` rejects empty names, `'.'`/`'..'`, and any name containing `'/'` or `'\\'` before passing to `joinPath`. Leading-slash names like `/etc` are rejected by the `includes('/')` check. No other code path calls `joinPath` with server-returned file names to update the browser path — `joinPath(path, selected)` at lines 277 and 557 uses `selected` for read/download operations (not navigation), and `selected` is always a file name from the current listing. **Already done. No missing code path found. Not re-touched.**

**WR-04** (`frontend/src/components/FileBrowser/FileRow.tsx:57-68`, comment "Phase 120 WR-04"):
`formatRowMtime` returns `'—'` for falsy `mtime` (empty string / null / undefined), then validates the RFC3339 calendar-date shape (`mtime[4] === '-' && mtime[7] === '-'`) and slices to 10 chars on success, falling back to `'—'` for non-conforming values. **Already done. Not re-touched.**

**WR-05** (`frontend/src/lib/humanSize.ts:18-29`, comment "WR-05"):
The comment correctly explains that `Math.floor` truncation (not rounding) is used for display stability — not to guard a 5 MiB boundary check (which is enforced server-side in bytes). **Already done. Not re-touched.**

## Deviations from Plan

None — plan executed exactly as specified. TD-5 was a code rewrite as planned; TD-4 was verify-only as planned (all WR items already confirmed satisfied from Phase 120 by both the plan and the live code).

## Issues Encountered

None. The TUI reference implementation (`joincode_prompt.go:153-208`) ported cleanly. The `encoding/json` import was removed as a natural consequence of replacing the JSON-decode success path.

## Threat Model Verification

T-123-07 (cap token in error/log path): The new implementation never logs the cap token; it is only returned to the caller. No `redactCapTokenFromError` call needed because error paths (`/join?error=<kind>`) return the error kind string, not the token.

T-123-08 (303 auto-follow loses cap): Mitigated — `CheckRedirect: ErrUseLastResponse` on the dedicated client; `StatusSeeOther` detection is explicit.

T-123-09 (shared client mutation): Mitigated — `TestExchangeJoinCode_SharedClientUnchanged` asserts `remoteFilesHTTPClient.CheckRedirect == nil` after every call.

T-123-10 (directory-index disclosure): WR-01 verified already in place.

T-123-11 (leading-slash name alters nav path): WR-03 verified already in place.

## Known Stubs

None — implementation is complete. `ExchangeJoinCodeAtURL` now returns the actual cap token from the server's 303 Location header.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- TD-5 (FSW-10) closed: `DaemonClient.ExchangeJoinCodeAtURL` now correctly acquires a remote session cap token, which is the prerequisite for Phase 128 remote-write cap acquisition
- TD-4 (FSW-11) closed: all Phase 120 WR items confirmed in tree
- Phase 128 can proceed without needing a workaround for the silent 303 failure mode

---
*Phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes*
*Completed: 2026-06-14*
