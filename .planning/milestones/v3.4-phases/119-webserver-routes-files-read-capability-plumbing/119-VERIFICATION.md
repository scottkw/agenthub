---
phase: 119
verified: 2026-05-20T00:00:00Z
status: passed
score: 5/5 must-haves verified
requirements_covered: 5/5
overrides_applied: 0
---

# Phase 119: Webserver Routes — `files.read` Capability Plumbing — Verification Report

**Phase Goal:** The webserver exposes the same three file endpoints under `requireFilesRead` middleware — capability-gated for Tailscale-HTTPS web-share viewers — and an integration test confirms read-only viewers get 403 with an explicit message, not 404.

**Verified:** 2026-05-20
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| #   | Truth                                                                                                                                                | Status      | Evidence                                                                                                                                                              |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Owner cap (with `files.read`) → 200 from GET /api/files/list, /stat, /read AND HEAD /api/files/read                                                  | VERIFIED    | 4× `TestFilesRoutes_OwnerCapReturns200_{List,Stat,Read_Get,Read_Head}` PASS                                                                                           |
| 2   | Viewer cap (without `files.read`) → 403 (not 404) with body containing literal "files.read"                                                          | VERIFIED    | 4× `TestFilesRoutes_ViewerCapReturns403_{List,Stat,Read_Get,Read_Head}` PASS — body assertion uses `strings.Contains(string(body), "files.read")`                     |
| 3   | POST /api/files/list → 405 (and PUT/DELETE on file routes → 405)                                                                                     | VERIFIED    | `TestFilesRoutes_PostReturns405`, `TestFilesRoutes_PutReturns405_Read`, `TestFilesRoutes_DeleteReturns405_Stat` all PASS (Go 1.22+ method-prefix mux auto-rejection)   |
| 4   | Zero new CSP violations — Playwright not in stack; defense-in-depth chromedp CSP tests + JSON-route header guards in place                           | VERIFIED    | `TestFilesRoutes_NoCSPHeader` + `TestFilesRoutes_NoHTMLContentType` PASS; `TestBrowserCSP_{Terminal,Dashboard,Join,TerminalImage}NoViolations` PASS (4/4 under -tags=e2e) |
| 5   | Missing cap (no `?cap=`) → 401 (not 404)                                                                                                             | VERIFIED    | `TestFilesRoutes_MissingCapReturns401_List` + `TestFilesRoutes_MissingCapReturns401_Read_Head` PASS                                                                   |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact                                          | Expected                                                                                              | Status     | Details                                                                                                                                                |
| ------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/webserver/server.go`                    | `filesHandler` field, `SetFilesHandler` setter, four mounted routes under `requireFilesRead`, 503 nil-safety | VERIFIED | Line 101 (`filesHandler *files.Handler`), line 153 (`SetFilesHandler`), lines 481-484 (4 routes), line 475 (`"files handler not configured"` / 503)    |
| `internal/webserver/files_routes_test.go`         | Integration test suite covering all 5 SCs                                                              | VERIFIED   | 17 `TestFilesRoutes_*` functions (15 from Plan 01 + 2 CSP guards from Plan 02), all PASS                                                                |
| `internal/webserver/capability_mw.go`             | `requireFilesRead` body unchanged (T-118-14/T-119-07 invariant); docstring updated                    | VERIFIED   | `awk` extract of `requireCapability` body contains 0 occurrences of `files.read` (invariant); `SetFilesHandlerProvider` references gone from `internal/` |
| `internal/daemon/api.go`                          | `ws.SetFilesHandler(a.filesHandler)` wired at BOTH `AutoStartWebServer` AND `handleWebServerStart`, BEFORE `ws.Start()` | VERIFIED | Line 370 (before line 372 Start), Line 859 (before line 862 Start) — both call sites verified                                                          |

### Key Link Verification

| From                                                          | To                                                                | Via                                                                | Status   | Details                                                                                                              |
| ------------------------------------------------------------- | ----------------------------------------------------------------- | ------------------------------------------------------------------ | -------- | -------------------------------------------------------------------------------------------------------------------- |
| `internal/daemon/api.go::AutoStartWebServer`                  | `internal/webserver.(*WebServer).SetFilesHandler`                  | `ws.SetFilesHandler(a.filesHandler)` BEFORE `ws.Start()`           | WIRED    | Line 370 ws.SetFilesHandler precedes line 372 ws.Start() — Pitfall 5 satisfied                                       |
| `internal/daemon/api.go::handleWebServerStart`                | `internal/webserver.(*WebServer).SetFilesHandler`                  | REST toggle-on path                                                | WIRED    | Line 859 ws.SetFilesHandler precedes line 862 ws.Start() — same ordering                                             |
| `internal/webserver.(*WebServer).setupRoutes`                 | `internal/webserver.(*WebServer).requireFilesRead`                 | 4× `mux.HandleFunc` wrapping closure that reads `ws.filesHandler` at request time | WIRED    | All four `ws.requireFilesRead(filesDispatch(...))` registrations present at lines 481-484                            |

### Data-Flow Trace (Level 4)

| Artifact                                  | Data Variable      | Source                                                                              | Produces Real Data | Status   |
| ----------------------------------------- | ------------------ | ----------------------------------------------------------------------------------- | ------------------ | -------- |
| `server.go` route closures                | `ws.filesHandler`  | Daemon `NewAPI` (Phase 118) constructs `a.filesHandler` with real sandbox resolver; wired via `ws.SetFilesHandler` | Yes | FLOWING  |
| `TestFilesRoutes_OwnerCapReturns200_List` | JSON response body | real `files.Handler.List` reading real tempdir `hi\n` file via `files.NewSandbox` | Yes — `"name":` in body | FLOWING  |
| `TestFilesRoutes_OwnerCapReturns200_Read_Get` | response body  | real handler reads 3-byte file from sandbox tempdir                                  | Yes — exact bytes match | FLOWING  |

### Behavioral Spot-Checks

| Behavior                                              | Command                                                                                                              | Result                                                                                                                                              | Status |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Full file-routes integration suite                    | `go test -run '^TestFilesRoutes_' ./internal/webserver/ -count=1 -v`                                                | 17 PASS, 0 FAIL                                                                                                                                     | PASS   |
| Cross-package suite (files, capability, webserver, daemon) | `go test ./internal/files/ ./internal/capability/ ./internal/webserver/ ./internal/daemon/ -count=1 -short`     | all 4 packages `ok`                                                                                                                                 | PASS   |
| Phase 118 invariant guards                            | `go test -run '^TestRequireFilesRead$\|^TestRequireCapability_UnchangedByPhase118$' ./internal/webserver/ -count=1` | PASS — middleware bodies untouched                                                                                                                  | PASS   |
| WEB-01 regression: daemon-mux file routes still work  | `go test -run '^TestAPI_Files' ./internal/daemon/ -count=1`                                                          | 7 PASS — daemon-socket auth-less surface preserved                                                                                                  | PASS   |
| Browser CSP e2e regression guard                      | `go test -tags=e2e -run '^TestBrowserCSP_' ./internal/webserver/ -count=1`                                          | 4 PASS (`Terminal`, `Dashboard`, `Join`, `TerminalImage` — `NoViolations`)                                                                          | PASS   |
| `requireCapability` body unchanged                    | `awk '/func \(ws \*WebServer\) requireCapability/,/^}/' .../capability_mw.go \| grep -c 'files.read'`               | `0` — Phase 118 T-118-14 invariant holds                                                                                                            | PASS   |
| No stale `SetFilesHandlerProvider` references         | `grep -rn 'SetFilesHandlerProvider' internal/`                                                                       | no matches (exit 1) — docstrings cleaned up in Plan 02                                                                                              | PASS   |

### Probe Execution

No probe scripts declared for Phase 119 (Go test suites act as the integration probe; covered under Behavioral Spot-Checks).

### Requirements Coverage

| Requirement | Source Plan       | Description                                                                                                                                | Status     | Evidence                                                                                                                                                              |
| ----------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| WEB-01      | 119-01            | Daemon's local-socket HTTP API exposes file routes auth-less for in-process consumers                                                       | SATISFIED  | `TestAPI_Files*` (7 PASS) — daemon-mux file routes still work without cap-token middleware after Phase 119                                                            |
| WEB-02      | 119-01            | Webserver mux exposes same three endpoints wrapped by `requireFilesRead`; mounted via `SetFilesHandler` (note: REQUIREMENTS.md still says `SetFilesHandlerProvider` — actual symbol per Phase 119 simplification is `SetFilesHandler`; verified via grep) | SATISFIED  | 4 routes mounted lines 481-484 server.go with `ws.requireFilesRead`; `TestFilesRoutes_OwnerCapReturns200_*` (4 PASS); `TestFilesRoutes_MissingCapReturns401_*` (2 PASS) |
| WEB-03      | 119-01            | Read-only viewer cannot use file endpoints; integration test asserts 403 across all 3 endpoints + both methods on /read                     | SATISFIED  | 4× `TestFilesRoutes_ViewerCapReturns403_{List,Stat,Read_Get,Read_Head}` PASS, all assert 403 + body contains "files.read"                                              |
| WEB-04      | 119-01            | Tailnet-remote sessions work via Tailscale HTTPS (peer-relative base URL)                                                                   | SATISFIED  | Same `WebServer` mux on every peer's TLS surface; no peer-specific code path; verified implicitly by integration tests driving the real TLS surface                    |
| WEB-05      | 119-02            | Zero new CSP amendments; cross-browser CSP e2e reports zero violations from file browser flows                                              | SATISFIED  | `TestFilesRoutes_NoCSPHeader` + `TestFilesRoutes_NoHTMLContentType` PASS (defense-in-depth on JSON routes); `TestBrowserCSP_*` 4/4 PASS (HTML pages unregressed); no new CSP middleware added |

All 5 requirements covered; no orphans.

**Note on WEB-05 / Playwright:** ROADMAP says "Playwright (Chromium + Firefox + WebKit)". The repository's actual browser-CSP harness is chromedp (Chromium-only) — Playwright is not in `go.mod`. Per the phase plan (119-02 `<objective>` and 119-RESEARCH §"CSP Impact Analysis") this is a known stack mismatch in the ROADMAP wording, not a Phase 119 gap: the file-browser UI is a Phase 120 deliverable, so there is no "file browse flow" to drive in a browser yet. Phase 119 satisfies the spirit of WEB-05 via (a) defense-in-depth Go unit assertions that file routes emit no CSP header and no HTML content type, AND (b) the existing chromedp HTML-page CSP suite passing unchanged. The cross-browser Playwright suite is properly scoped to Phase 120 when the UI exists. Not flagged as a gap.

### Anti-Patterns Found

| File                                         | Line   | Pattern                                              | Severity | Impact                                                                                          |
| -------------------------------------------- | ------ | ---------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------- |
| (none)                                       | —      | —                                                    | —        | No `TBD`/`FIXME`/`XXX` markers, no placeholder/stub patterns, no empty handlers in Phase 119 files |

Phase 119 modified `internal/webserver/server.go` (+46 lines), `internal/webserver/capability_mw.go` (1-line docstring), `internal/files/handler.go` (1-line docstring), `internal/daemon/api.go` (+2 lines), and added `internal/webserver/files_routes_test.go` (340+ lines). All changes are substantive code with real wiring; no stubs, debt markers, or empty returns introduced.

### Human Verification Required

None. All ROADMAP success criteria are observable in automated tests against the real TLS-mounted mux with real signed cap tokens and a real file handler backed by a tempdir sandbox. Browser-level CSP behavior is covered by the chromedp e2e suite which passes. The Phase 120 file-browser UI will drive cross-surface UAT — that is a Phase 120 deliverable.

### Gaps Summary

No gaps. Phase 119 goal achieved:

- Webserver mounts `GET /api/files/list`, `GET /api/files/stat`, `GET /api/files/read`, and `HEAD /api/files/read` under `requireFilesRead`.
- Daemon wires `SetFilesHandler(a.filesHandler)` at both webserver construction sites (`AutoStartWebServer` line 370, `handleWebServerStart` line 859), both before `ws.Start()`.
- Read-only viewer caps return 403 with body containing the literal `"files.read"` substring on all 4 file endpoints — verified by 4 dedicated tests.
- Missing-cap requests return 401 (not 404) — verified by 2 tests covering both GET and HEAD to confirm no route-existence leak via status-code distinction.
- POST/PUT/DELETE on file routes return 405 via Go 1.22+ method-prefix mux auto-rejection — verified by 3 tests.
- Nil-handler defense-in-depth: 503 "files handler not configured" — verified by `TestFilesRoutes_NilHandlerReturns503`.
- Phase 118 sandbox traversal protection still active through the new routes — verified by `TestFilesRoutes_TraversalRejected`.
- CSP regression posture: no CSP header and no `text/html` content type on JSON file routes (defense-in-depth); existing HTML-page browser CSP e2e suite (4 tests) still passes; no new CSP amendments to the codebase.
- Phase 118 invariants preserved: `requireCapability` body unchanged (T-118-14/T-119-07), `requireFilesRead` body unchanged, daemon-socket auth-less surface unchanged (WEB-01).

---

_Verified: 2026-05-20_
_Verifier: Claude (gsd-verifier)_
