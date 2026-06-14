---
phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in
plan: "03"
subsystem: api
tags: [go, http-proxy, capability, remote-files, daemon]

# Dependency graph
requires:
  - phase: 124-02
    provides: files.write constant and requireFilesWrite middleware already wired on webserver

provides:
  - proxyRemoteFiles forwards r.Body + Content-Type for PUT/POST/PATCH write verbs
  - Five remote write proxy routes registered on the daemon socket
  - Body-forwarding round-trip tests for the remote write proxy

affects:
  - 124-04 (TUI parity — remote write routes are now reachable from TUI)
  - 128 (remote write parity phase consumes these daemon-socket routes)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Body forwarding in proxy: var body io.Reader; nil for GET/HEAD, r.Body for write verbs"
    - "Request Content-Type forwarding: copy inbound header for write verbs only (not response side)"
    - "Remote write handler: thin method calling proxyRemoteFiles with fixed op string"

key-files:
  created: []
  modified:
    - internal/daemon/remote_files.go
    - internal/daemon/api.go
    - internal/daemon/remote_files_test.go

key-decisions:
  - "Forward r.Body opaquely as a byte pipe (not re-parse) to preserve multipart boundaries and JSON payloads"
  - "Daemon-socket write routes carry no auth middleware (loopback-trust WEB-01); CSRF enforcement is on the remote peer's requireFilesWrite"
  - "Body forwarding condition: PUT || POST || PATCH — DELETE carries no body by convention and is excluded"

patterns-established:
  - "Pattern: write-verb body condition — r.Method == PUT || POST || PATCH -> body = r.Body; nil otherwise"
  - "Pattern: Content-Type forwarding gated on body != nil (single check covers all write verbs)"

requirements-completed: [CAP-10]

# Metrics
duration: 5min
completed: 2026-06-14
---

# Phase 124 Plan 03: Remote Write Proxy Body Forwarding + Route Registration Summary

**Fixed nil-body bug in proxyRemoteFiles (CAP-10): five remote write proxy routes added to daemon socket with r.Body + Content-Type forwarding for PUT/POST/PATCH verbs**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-14T17:26:33Z
- **Completed:** 2026-06-14T17:31:44Z
- **Tasks:** 3 (Task 0 RED test + Task 1 body fix + Task 2 route registration)
- **Files modified:** 3

## Accomplishments

- Fixed the nil-body bug in `proxyRemoteFiles` (`remote_files.go:169`) — write verbs now forward `r.Body` verbatim to the upstream peer, unblocking remote write parity for Phase 128
- Forwarded the inbound `Content-Type` request header for write verbs so multipart boundaries (`upload`) and `application/json` (`rename`, `write`) survive transit
- Registered five remote write proxy routes on the daemon socket: `PUT /write`, `POST /upload`, `DELETE /delete`, `POST /rename`, `POST /mkdir`
- Added `TestRemoteFilesWrite_ForwardsBody`, `TestRemoteFilesWrite_CallerCapStripped`, and `TestRemoteFilesWrite_GetPassesNilBody` with full RED/GREEN TDD cycle
- Verified anti-smuggling cap-strip/force-set behavior is preserved for write verbs (T-124-10)
- All tests pass under `-race`; `go vet` and `gofmt` clean

## Task Commits

Each task was committed atomically:

1. **Task 0: Write failing remote write-proxy body-forwarding test (RED)** - `60cc563` (test)
2. **Task 1: Forward r.Body + Content-Type for write verbs** - `fd30f7c` (fix)
3. **Task 2: Register five remote write proxy routes** - `dcbf7c7` (feat)

## Files Created/Modified

- `internal/daemon/remote_files.go` - Fixed nil-body bug; added five write handler methods; gofmt applied
- `internal/daemon/api.go` - Registered five `PUT/POST/DELETE/POST/POST` remote write proxy routes
- `internal/daemon/remote_files_test.go` - Added `TestRemoteFilesWrite_ForwardsBody`, `TestRemoteFilesWrite_CallerCapStripped`, `TestRemoteFilesWrite_GetPassesNilBody`

## Decisions Made

- Forward `r.Body` opaquely as a byte pipe rather than re-parsing: preserves multipart boundaries and JSON payloads without adding parsing surface
- Body forwarding condition is `PUT || POST || PATCH` — `DELETE` carries no body per HTTP convention and `GET/HEAD` are read-only; this matches T-124-12's mitigation disposition
- Content-Type forwarding is gated on `body != nil` (single condition covers all write verbs, no duplication)
- Daemon-socket write routes carry no middleware (loopback-trust per WEB-01); CSRF + cap enforcement happens on the remote peer's `requireFilesWrite`
- Five thin handler methods added to `remote_files.go` following the existing `handleRemoteFilesList/Stat/Read` pattern (explicit named handlers, not inline closures)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] gofmt formatting fix on remote_files.go**
- **Found during:** Task 2 (after adding write handlers)
- **Issue:** `gofmt -l` reported `remote_files.go` had formatting issues after adding the write handler block (comment style differences from gofmt's doc comment normalizer)
- **Fix:** Ran `gofmt -w remote_files.go`; verified no logic changes
- **Files modified:** `internal/daemon/remote_files.go`
- **Verification:** `gofmt -l` returns no output; `go test -race ./internal/daemon/` still green
- **Committed in:** dcbf7c7 (Task 2 commit, the gofmt'd file was staged together)

---

**Total deviations:** 1 auto-fixed (1 blocking — formatting)
**Impact on plan:** Trivial formatting fix; no behavior change. No scope creep.

## Issues Encountered

- `TestRemoteFilesWrite_ForwardsBody` was RED for the expected reason (nil-body proxy) but reported `status 404` rather than "empty body" in the failure message — because the write routes weren't registered yet (Task 2 hadn't run). This is correct TDD sequencing: body fix (Task 1) + route registration (Task 2) together make the test green. The RED-OK verify command matched `FAIL` in the output as expected.

## Known Stubs

None — this plan adds no UI or data-rendering paths; all changes are proxy plumbing.

## Threat Flags

None — the threat surface additions (five write proxy routes) were explicitly modeled in the plan's threat model (T-124-10 through T-124-13). No new unmodeled surface introduced.

## Self-Check: PASSED

- `internal/daemon/remote_files.go` — FOUND
- `internal/daemon/api.go` — FOUND
- `internal/daemon/remote_files_test.go` — FOUND
- Commit `60cc563` (test RED) — FOUND
- Commit `fd30f7c` (fix body forward) — FOUND
- Commit `dcbf7c7` (route registration) — FOUND

## Next Phase Readiness

- Remote write proxy is complete for Phase 128 (remote write parity): daemon socket can now forward write verbs to any tailnet peer whose webserver has `requireFilesWrite` routes mounted
- Cap-strip anti-smuggling preserved on write verbs (T-124-10)
- Plans 124-04 and 124-05 (TUI parity, settings schema migration) are unaffected by this plan and can proceed independently

---
*Phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in*
*Completed: 2026-06-14*
