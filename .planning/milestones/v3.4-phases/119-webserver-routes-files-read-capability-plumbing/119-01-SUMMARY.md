---
phase: 119-webserver-routes-files-read-capability-plumbing
plan: 01
subsystem: webserver
tags: [webserver, capability, files, routes, integration, files.read, web-01, web-02, web-03, web-04]
requires:
  - "internal/files.Handler (Phase 118)"
  - "internal/webserver.requireFilesRead (Phase 118)"
  - "internal/daemon.API.filesHandler (Phase 118)"
provides:
  - "WebServer.SetFilesHandler(*files.Handler)"
  - "GET /api/files/list on webserver mux (cap-gated)"
  - "GET /api/files/stat on webserver mux (cap-gated)"
  - "GET /api/files/read on webserver mux (cap-gated)"
  - "HEAD /api/files/read on webserver mux (cap-gated)"
  - "503 nil-handler defense-in-depth guard"
  - "WEB-02..WEB-04 success-matrix integration tests"
affects:
  - "internal/webserver (new field, setter, four routes)"
  - "internal/daemon (two single-line wires; no behavior change for daemon-mux)"
tech_stack:
  added: []
  patterns:
    - "Closure-over-receiver for request-time field read (Pitfall 2 mitigation)"
    - "Go 1.22+ method-prefix mux with separate HEAD registration (Pitfall 1)"
    - "Same Handler instance mounted on two muxes (Phase 118 stateless design)"
key_files:
  created:
    - "internal/webserver/files_routes_test.go (15 integration subtests)"
    - ".planning/phases/119-webserver-routes-files-read-capability-plumbing/119-01-SUMMARY.md"
  modified:
    - "internal/webserver/server.go (+46 lines: import, field, setter, 4 routes)"
    - "internal/daemon/api.go (+2 lines: SetFilesHandler at two construction sites)"
decisions:
  - "Mounted file routes via filesDispatch closure helper rather than 4 inline closures (DRY + shared nil-check) — closes routes for any future Handler method without further boilerplate."
  - "Asserted JSON wire field 'name' (lowercase) instead of plan-doc 'Name' (Go field name). Implementation contract is FileEntry json tag; plan referenced the Go struct symbol."
  - "Did NOT abstract a wireWebServerCallbacks helper for the two daemon construction sites — below three-examples abstraction threshold per CLAUDE.md."
metrics:
  duration: "~6 minutes"
  completed: "2026-05-20"
  tasks: 3
  files_touched: 3
  loc_added: 346
---

# Phase 119 Plan 01: Webserver File-Route Mounting + files.read Plumbing Summary

Mounted the four read-only file routes (`GET /api/files/list`, `GET /api/files/stat`, `GET /api/files/read`, `HEAD /api/files/read`) on the public webserver mux under the Phase 118 `requireFilesRead` middleware, and wired `ws.SetFilesHandler(a.filesHandler)` at both daemon webserver construction sites. Verified end-to-end with 15 integration tests driving the real TLS surface with real signed cap tokens.

## What was built

### Task 1: Mount /api/files/* routes on webserver mux

**File:** `internal/webserver/server.go` (+46 lines)

- Added import: `"github.com/scottkw/agenthub/internal/files"` (alphabetical between `capability` and `relay`).
- Added field `filesHandler *files.Handler` on `WebServer` struct (between `pluginSettingsProvider` and `pluginConfigMu`). No mutex — set once before `Start()`, mirroring `sessionResolver` pattern.
- Added setter `SetFilesHandler(h *files.Handler)` adjacent to `SetPluginSettingsProvider`.
- Added four route registrations inside `setupRoutes()`, immediately after the plugin-config-stream block:
  - `GET /api/files/list` → `ws.requireFilesRead(filesDispatch(h.List))`
  - `GET /api/files/stat` → `ws.requireFilesRead(filesDispatch(h.Stat))`
  - `GET /api/files/read` → `ws.requireFilesRead(filesDispatch(h.Read))`
  - `HEAD /api/files/read` → `ws.requireFilesRead(filesDispatch(h.Read))`
- The `filesDispatch` helper closes over `ws` (NOT `ws.filesHandler`) so the field is dereferenced at request time. If nil → 503 with body `"files handler not configured"`. This satisfies the Pitfall 2 invariant: `setupRoutes()` runs in `NewWebServer()` BEFORE the daemon can wire the handler, so registration-time binding (`ws.filesHandler.List`) would capture nil and panic.
- Did NOT modify `requireCapability` or `requireFilesRead` bodies (T-118-14 / T-119-07 invariant).

**Commit:** `f2a3edb`

### Task 2: Integration tests against the real mux

**File:** `internal/webserver/files_routes_test.go` (new, +298 lines, 15 subtests)

Shared harness `newFilesTestServer(t)` creates a started `WebServer`, installs the test signing key, enables a session, writes a 3-byte file `"hi\n"` to a tempdir, constructs a `files.Handler` rooted there, and calls `SetFilesHandler`. Tests drive requests via the supplied `*http.Client` (trusts the in-memory CA) against `ws.BaseURL() + /api/files/...`.

| Subtest | Asserts |
|---|---|
| `TestFilesRoutes_OwnerCapReturns200_List` | Owner cap → 200 + JSON body contains `"name":` |
| `TestFilesRoutes_OwnerCapReturns200_Stat` | Owner cap → 200 + JSON FileEntry shape |
| `TestFilesRoutes_OwnerCapReturns200_Read_Get` | Owner cap → 200 + body exactly `"hi\n"` (3 bytes) |
| `TestFilesRoutes_OwnerCapReturns200_Read_Head` | Owner cap → 200 + `Content-Length: 3` + empty body |
| `TestFilesRoutes_ViewerCapReturns403_List` | Viewer cap → 403 + body contains `files.read` |
| `TestFilesRoutes_ViewerCapReturns403_Stat` | Same on `/stat` |
| `TestFilesRoutes_ViewerCapReturns403_Read_Get` | Same on `GET /read` |
| `TestFilesRoutes_ViewerCapReturns403_Read_Head` | Same on `HEAD /read` (status-only) |
| `TestFilesRoutes_MissingCapReturns401_List` | No `?cap=` → 401 |
| `TestFilesRoutes_MissingCapReturns401_Read_Head` | No `?cap=` → 401 (NOT 404) on HEAD |
| `TestFilesRoutes_PostReturns405` | POST `/list` → 405 (Go 1.22+ mux) |
| `TestFilesRoutes_PutReturns405_Read` | PUT `/read` → 405 |
| `TestFilesRoutes_DeleteReturns405_Stat` | DELETE `/stat` → 405 |
| `TestFilesRoutes_NilHandlerReturns503` | No SetFilesHandler call → 503 + `"files handler not configured"` |
| `TestFilesRoutes_TraversalRejected` | `?path=../../etc/passwd` → 403 + `"access denied"` (sandbox layer) |

All 15 subtests PASS. Full webserver suite green (`go test ./internal/webserver/ -count=1 -short` → `ok`).

**Commit:** `8817e92`

### Task 3: Wire ws.SetFilesHandler at both daemon webserver construction sites

**File:** `internal/daemon/api.go` (+2 lines total)

- Line 370 (inside `AutoStartWebServer`): inserted `ws.SetFilesHandler(a.filesHandler)` between `ws.SetSigningKey(key)` and `ws.SetJoinCodes(a.joinCodes)`.
- Line 859 (inside `handleWebServerStart`): same single-line addition at the REST toggle-on path.

Both calls reuse the existing `a.filesHandler` constructed once in `NewAPI` (line 68-77). Both placed BEFORE `ws.Start()` so the first request observes a non-nil handler (Pitfall 5).

**Commit:** `d907c0f`

## Verification

Final verification block from the plan executed cleanly:

```
go build ./internal/...                                              # OK
go vet ./internal/webserver/ ./internal/daemon/                      # OK
go test -run '^TestFilesRoutes_'        ./internal/webserver/  -count=1  # ok (15 PASS)
go test -run '^TestAPI_Files'           ./internal/daemon/     -count=1  # ok (WEB-01 regression clean)
go test -run '^TestRequireFilesRead$|^TestRequireCapability_UnchangedByPhase118$' \
                                        ./internal/webserver/  -count=1  # ok (Phase 118 invariant guards green)
go test ./internal/files/ ./internal/capability/ \
        ./internal/webserver/ ./internal/daemon/ -count=1 -short          # all four ok
```

## Success criteria

- [x] Owner cap (Perms includes files.read) → 200 on all 4 routes against the real TLS webserver (WEB-02 SC#1)
- [x] Viewer cap (Perms="read") → 403 + body contains "files.read" on all 4 routes (WEB-03 / FS-13)
- [x] Missing `?cap=` → 401, not 404 (WEB-02 SC#5 — route-existence-leak guard)
- [x] POST/PUT/DELETE on any file route → 405 (WEB-02 SC#3 — Go 1.22+ mux auto-rejection)
- [x] Nil filesHandler → 503 "files handler not configured" (Pitfall 2 mitigation)
- [x] Daemon-socket file routes remain auth-less; `TestAPI_Files*` pass unchanged (WEB-01)
- [x] Tailnet-remote sessions (WEB-04) work via the same mounted routes (verified implicitly — no peer-specific code path)
- [x] requireCapability and requireFilesRead bodies UNCHANGED (T-118-14 / T-119-07)
- [x] All four affected packages build and test clean

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test assertion correction] JSON field name is lowercase 'name', not 'Name'**

- **Found during:** Task 2 (initial test run)
- **Issue:** Plan's acceptance criteria required asserting `"Name":` in the FileEntry JSON body. The actual JSON wire format uses lowercase `"name":` (per `internal/files/types.go` json tag `json:"name"`). Plan referred to the Go struct symbol name rather than the json tag.
- **Fix:** Changed both assertion lines and the error message to `"name":` to match the actual wire shape that the Phase 118 handler emits.
- **Files modified:** `internal/webserver/files_routes_test.go` (2 lines)
- **Commit:** `8817e92` (part of the same Task 2 commit)

No other deviations. No auth gates encountered. No architectural decisions required.

## Threat Model Compliance

All 7 `mitigate` dispositions in the plan's `<threat_model>` are covered by code or tests:

| Threat | Mitigation | Verified by |
|---|---|---|
| T-119-01 (spoof) | `requireCapability` body unchanged; composed inside `requireFilesRead` | grep guard (Phase 118 source-inspection test) |
| T-119-02 (elevation) | `HasPerm(claims.Perms, PermFilesRead)` whole-token check | `TestFilesRoutes_ViewerCapReturns403_*` (4 subtests) |
| T-119-03 (traversal) | Phase 118 sandbox `os.OpenRoot` unchanged | `TestFilesRoutes_TraversalRejected` |
| T-119-04 (bypass via verb) | Go 1.22+ method-prefix mux, no fallback handler | `TestFilesRoutes_PostReturns405` + Put + Delete |
| T-119-05 (route enumeration) | `requireCapability` returns 401 for missing/bad cap | `TestFilesRoutes_MissingCapReturns401_*` (2 subtests) |
| T-119-06 (nil-Handler DoS) | Closure reads `ws.filesHandler` at request time; nil → 503 | `TestFilesRoutes_NilHandlerReturns503` |
| T-119-07 (middleware tamper) | `requireCapability` body untouched | Phase 118 `TestRequireCapability_UnchangedByPhase118` |

## TDD Gate Compliance

Plan-level tasks are `tdd="true"` but mixed-shape: Task 1 is implementation-only (verification is build/vet, no new tests), Task 2 is the full integration test suite, Task 3 is two-line wire-up (verified by Task 2 + Phase 118 regression tests). Git history shows:

- `feat(119-01): mount /api/files/* routes` — Task 1 implementation
- `test(119-01): integration tests for /api/files/*` — Task 2 RED+GREEN folded (tests written against the just-mounted routes; would fail without Task 1)
- `feat(119-01): wire ws.SetFilesHandler at both webserver construction sites` — Task 3

The integration tests in Task 2 do NOT depend on the Task 3 daemon-wire — they construct the WebServer directly via `testServer` and call `SetFilesHandler` themselves. Task 3 provides the production daemon wiring needed for the live `/api/files/*` surface; verified by the existing `TestAPI_Files*` daemon tests passing unchanged.

## Self-Check

Commits exist:
- f2a3edb FOUND in git log
- 8817e92 FOUND in git log
- d907c0f FOUND in git log

Files exist:
- internal/webserver/server.go FOUND (modified, +46 lines)
- internal/webserver/files_routes_test.go FOUND (new, 298 lines)
- internal/daemon/api.go FOUND (modified, +2 lines)

Test guarantees:
- TestFilesRoutes_* — 15 PASS
- TestAPI_Files* — PASS (WEB-01 regression clean)
- TestRequireCapability_UnchangedByPhase118 — PASS (T-119-07 invariant)
- TestRequireFilesRead — PASS (Phase 118 wrapper standalone unchanged)

## Self-Check: PASSED
