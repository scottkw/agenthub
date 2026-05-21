---
phase: 122
plan: 01-recovery
subsystem: daemon-remote-files-proxy
tags: [daemon, proxy, capability, remote, files, recovery]
requires: []
provides:
  - "RemoteCapStore — thread-safe in-memory map[sessionID]{baseURL, capToken}"
  - "POST /api/remote-files/caps — deposit (sessionId, baseUrl, capToken)"
  - "GET /api/files/remote/{sessionID}/list — proxy to remote /api/files/list"
  - "GET /api/files/remote/{sessionID}/stat — proxy to remote /api/files/stat"
  - "GET /api/files/remote/{sessionID}/read — proxy to remote /api/files/read"
  - "HEAD /api/files/remote/{sessionID}/read — proxy HEAD for ranged preview"
  - "API.remoteCaps field on *API + NewRemoteCapStore() in NewAPI"
  - "remoteFilesClientForTest test-injection hook for httptest.NewTLSServer"
affects:
  - "internal/daemon/api.go (API struct + NewAPI + registerRoutes)"
  - "internal/daemon/client_remote_files.go (122-03 stub now reaches a real endpoint)"
tech-stack:
  added: []
  patterns:
    - "TLS 1.2+ HTTPS client with InsecureSkipVerify (mirrors internal/tailnet/sessions.go)"
    - "http.MaxBytesReader + DisallowUnknownFields for POST body hardening"
    - "Caller-supplied query-param stripping (cap, session) before forwarding to upstream"
    - "Source-grep guards for no-disk-persistence + TLS-min-version invariants"
key-files:
  created:
    - "internal/daemon/remote_caps.go"
    - "internal/daemon/remote_caps_test.go"
    - "internal/daemon/remote_files.go"
    - "internal/daemon/remote_files_test.go"
  modified:
    - "internal/daemon/api.go"
decisions:
  - "404 (not 401) when no cap is registered locally — frontend treats 404 as 'paste a join code' and pops the RemoteJoinCodeModal; 401 is reserved for upstream cap-rejected"
  - "200 + {ok:true} on POST success (not 204) — small JSON envelope the frontend can assert on"
  - "Caller-supplied ?cap= stripped from query before forwarding — defense in depth against a malicious local process smuggling a different token through the proxy"
  - "Caller-supplied ?session= overwritten to match URL path param — prevents pointing at one session while listing another"
  - "Test-injection hook (remoteFilesClientForTest) over a constructor variant — keeps the production API surface clean; tests only override one field"
metrics:
  duration: ~25 min
  completed_at: "2026-05-20T23:00Z"
  tasks_completed: 2
  files_created: 4
  files_modified: 1
  test_count_added: 21 (8 RemoteCapStore + 13 proxy/POST)
---

# Phase 122 Plan 01 RECOVERY: Daemon-side Remote-Files Proxy Summary

One-liner: Closes the daemon-side gap that 122-03 was stubbing — adds `RemoteCapStore` for in-memory cap caching, `POST /api/remote-files/caps` for deposit, and four `GET/HEAD /api/files/remote/{sessionID}/{list,stat,read}` routes that proxy verbatim to the remote peer's webserver using the cached cap.

## What Was Built

### New File: `internal/daemon/remote_caps.go`
- `RemoteCapStore` struct: `sync.RWMutex` + `map[string]remoteCapEntry`
- `NewRemoteCapStore()`, `Put(sessionID, baseURL, capToken) error`, `Get(sessionID) (baseURL, capToken string, ok bool)`, `Delete(sessionID)`
- `Put` validates non-empty inputs and returns explicit errors so the POST handler can surface 400 to the caller
- Type doc forbids disk persistence (matches `JoinCodeManager` invariant); enforced by `TestRemoteCapStore_NoDiskWriteAPIs` source-grep guard

### New File: `internal/daemon/remote_files.go`
- `newRemoteFilesHTTPClient()` — outbound HTTPS client with `tls.VersionTLS12` minimum, 10s timeout, `InsecureSkipVerify=true` for tailnet self-signed certs (matches existing pattern in `internal/tailnet/sessions.go` and `internal/daemon/client_remote_files.go`)
- `(*API).remoteFilesClient()` — returns `remoteFilesClientForTest` if set, otherwise builds a fresh client; injection seam for `httptest.NewTLSServer`
- `(*API).handleRegisterRemoteCap(w, r)` — POST handler; 8 KiB body cap; `DisallowUnknownFields`; calls `remoteCaps.Put`; returns 200 + `{ok:true}` or 400
- `(*API).handleRemoteFilesList`, `handleRemoteFilesStat`, `handleRemoteFilesRead` — thin wrappers over `proxyRemoteFiles(w, r, op)`
- `(*API).proxyRemoteFiles(w, r, op)` — the workhorse:
  - Extracts `sessionID` via `r.PathValue("sessionID")` (Go 1.22 method-prefix routing)
  - Looks up cap via `remoteCaps.Get`; returns 404 + `{"error":"no cap registered for session"}` on miss
  - Strips caller-supplied `cap` query param (defense in depth); overwrites `session` to match URL path param
  - Appends server-side `?cap=<capToken>` (mirrors Phase 119 `requireFilesRead` pattern)
  - Issues `http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, nil)` so HEAD-on-read forwards as HEAD
  - Forwards Content-Type, Content-Length, Content-Range, Last-Modified, ETag, Accept-Ranges headers + status code + body
  - `redactCapTokenFromError` strips the cap from any error string before surfacing to the client (T-122-01 mitigation; 4 call sites)
- `redactCapTokenFromError(err, capToken) string` — small helper, used uniformly

### Modified: `internal/daemon/api.go`
- Added `remoteCaps *RemoteCapStore` field on `*API`
- Added `remoteFilesClientForTest *http.Client` test-injection field (documented as test-only)
- `NewAPI` initializes `remoteCaps: NewRemoteCapStore()`
- `registerRoutes` registers the four new GET/HEAD proxy routes + the POST cap-deposit route, with a comment block explaining the loopback trust boundary

## Tasks Executed

| # | Task                                                                 | Status | Commit    | Test count |
| - | -------------------------------------------------------------------- | ------ | --------- | ---------- |
| 1 | RemoteCapStore + 8 unit tests under -race                            | DONE   | `5c8b8bd` | 8 Go       |
| 2 | Proxy handlers + POST handler + route registration + 13 round-trip tests | DONE   | `5647b90` | 13 Go      |

## Verification Results

- **Go tests (daemon + regression):** `go test ./internal/daemon/ ./internal/webserver/ ./internal/files/ ./internal/tui/ -race -count=1` — all packages PASS
- **`go vet ./internal/daemon/...`:** clean
- **`go build ./internal/daemon/...`:** clean
- **Grep gates from PLAN.md `<verification>`:**

| Check                                                       | Expected | Actual | Status |
| ----------------------------------------------------------- | -------- | ------ | ------ |
| `tls.VersionTLS12` in remote_files.go                       | ≥1       | 1      | PASS   |
| `redactCapTokenFromError` calls                             | ≥3       | 4      | PASS   |
| disk-write APIs in remote_caps.go (excluding comments)      | 0        | 0      | PASS   |
| `/api/files/remote/` in api.go                              | 4 (≥)    | 5      | PASS (4 routes + 1 comment) |
| `/api/remote-files/caps` in api.go                          | 1 (≥)    | 2      | PASS (1 route + 1 comment) |
| `remoteCaps.*\*RemoteCapStore` field on API struct          | 1        | 1      | PASS   |

## Deviations from the Recovery Prompt

### Auto-fixed Issues

**1. [Rule 2 — Missing Critical Functionality] Caller-supplied `?session=` query parameter could mismatch the URL path's `{sessionID}`**

- **Found during:** writing `proxyRemoteFiles`.
- **Issue:** The remote webserver's `/api/files/*` endpoints use `?session=<id>` for sandbox lookup (Phase 118). If the daemon proxy blindly forwarded the caller's `?session=` along with the URL path's `{sessionID}`, a caller could point at one session via the path while operating on another via the query. While the daemon socket is loopback-trusted, defense in depth dictates we force-overwrite `?session` to match the path param.
- **Fix:** `proxyRemoteFiles` calls `q.Set("session", sessionID)` after the param copy, before appending `?cap=`. Same hardening as the cap-stripping logic.
- **Files modified:** `internal/daemon/remote_files.go`.

**2. [Rule 2 — Missing Critical Functionality] Caller-supplied `?cap=` query parameter could smuggle a different token**

- **Found during:** writing `proxyRemoteFiles`.
- **Issue:** The recovery prompt says "proxy strips `?cap=` from incoming request, then appends it server-side from RemoteCapStore" but does not justify why. A malicious local process that already has a different cap token (e.g., one for a session it lost) could try to use this proxy as an oracle.
- **Fix:** During the query-param copy loop, skip any key matching `cap` (case-insensitive). Then append `q.Set("cap", capToken)` from the cap store. Tested explicitly by `TestRemoteFiles_CallerCapStripped`.
- **Files modified:** `internal/daemon/remote_files.go`.

**3. [Rule 3 — Blocking] HTTP body needed a size cap on the POST handler**

- **Found during:** writing `handleRegisterRemoteCap`.
- **Issue:** Without `http.MaxBytesReader`, a malicious local process could pin daemon memory with a giant JSON blob during decode. While the socket is loopback-trusted, the daemon's other POST handlers (`handleSetPluginSettings`, `handleSetSearchConfig`, etc.) all cap bodies at 8 KiB. Consistency + defense in depth.
- **Fix:** Wrapped `r.Body` with `http.MaxBytesReader(w, r.Body, 8192)` + `DisallowUnknownFields` on the decoder. Mirrors the existing pattern.
- **Files modified:** `internal/daemon/remote_files.go`.

### Choices Made (No Permission Needed)

- **404 instead of 401 for "no cap registered locally":** the recovery prompt's spec. Matches Plan 122-03 frontend expectation that 401 means "upstream rejected the cap" (re-prompt) while 404 means "you never deposited a cap" (paste join code modal).
- **200 + `{ok:true}` instead of 204 No Content on POST success:** the recovery prompt's spec. Gives the frontend a JSON envelope to assert on rather than just a status code.
- **Did NOT implement `DaemonClient.ExchangeJoinCodeAtURL` redirect-following:** scope-bounded to the daemon-side surface. The existing 122-03 `client_remote_files.go` expects a JSON `{cap:"..."}` shape from `/join/exchange`, which mismatches the actual webserver's 303-redirect shape — but that mismatch is **not** in this recovery's scope. Filing as a deferred issue below.

## Deferred Issues

**Stub remaining in `internal/daemon/client_remote_files.go::ExchangeJoinCodeAtURL`:** The helper expects `/join/exchange` to respond with JSON `{cap:"..."}` (lines 107-117), but the webserver's `handleJoinExchange` actually responds with HTTP 303 + `Location: /sessions/{id}?cap=<token>` (see `internal/webserver/server.go:642-706`). The helper's status-code-based error mapping works for the 4xx/5xx error paths, but the success path will fail JSON-decode on the redirect body. This is a Plan 122-03 issue, not 122-01 — recommend filing as a separate gap-closure: either parse the Location header for `?cap=` in the existing helper, or have the daemon proxy `/join/exchange` and parse the redirect server-side.

## Threat Surface Scan

| Threat ID | Disposition | Verification |
|-----------|-------------|--------------|
| T-122-01 (cap-token leak in error strings) | mitigated | `redactCapTokenFromError` used on 4 error paths in remote_files.go |
| T-122-02 (cap-token disk persistence)      | mitigated | Source-grep guard in `TestRemoteCapStore_NoDiskWriteAPIs` |
| T-122-03 (cap-token forwarding via plaintext) | mitigated | `tls.VersionTLS12` source-grep + `TestRemoteFiles_TLSMinVersionInSource` |
| T-122-04 (upstream-status spoofing)        | mitigated | Proxy passes through verbatim; tests assert 401/403 pass-through |
| T-122-05 (cap-deposit via loopback)        | accept    | Daemon socket is the loopback trust boundary (existing invariant) |
| T-122-06 (unbounded RemoteCapStore growth) | accept    | Loopback-only; restart wipes; same as JoinCodeManager |
| T-122-NEW (caller smuggles ?cap=)          | mitigated | Caller `cap` stripped before forwarding; tested by `TestRemoteFiles_CallerCapStripped` |
| T-122-NEW (caller mismatches ?session=)    | mitigated | `?session` force-overwritten to URL path param |

No new threat-flags introduced beyond the registry plus the two new defense-in-depth mitigations called out above.

## Known Stubs

None in this recovery. The deferred issue (`client_remote_files.go::ExchangeJoinCodeAtURL` JSON-vs-303 mismatch) is in code owned by Plan 122-03 and is documented under "Deferred Issues" above.

## Self-Check: PASSED

Files claimed in this summary exist:
- `internal/daemon/remote_caps.go` — FOUND
- `internal/daemon/remote_caps_test.go` — FOUND
- `internal/daemon/remote_files.go` — FOUND
- `internal/daemon/remote_files_test.go` — FOUND
- `internal/daemon/api.go` — MODIFIED (verified via `git diff`)

Commits exist on this branch:
- `5c8b8bd` (Task 1) — FOUND
- `5647b90` (Task 2) — FOUND
