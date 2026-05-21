---
phase: 118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi
plan: 05
subsystem: daemon (integration of Plans 01-04 into the running daemon process)
tags: [daemon, routes, files, capability, fs-03, fs-04, fs-05, fs-06, fs-07, fs-12, integration]
one_liner: "Wire files.Handler + sandboxResolver + DaemonClient methods + files.read cap-bit issuance into the daemon mux — 17 integration tests pass end-to-end through real Unix socket transport"
dependency_graph:
  requires:
    - "internal/files.NewSandbox (Plan 01)"
    - "internal/files.NewHandler + Handler.{List,Stat,Read} (Plan 02)"
    - "internal/capability.PermFilesRead + HasPerm (Plan 03)"
    - "SessionEngine.GetSessionWorkDir + e.filesRead (Plan 04)"
  provides:
    - "GET /api/files/list + /stat + /read + HEAD /api/files/read on the daemon mux"
    - "DaemonClient.ListFiles / StatFile / ReadFile / HeadFile (typed Go API)"
    - "FileEntry / FileListResponse type aliases re-exported from internal/daemon"
    - "(*SessionEngine).filesReadEnabled() — settings-gated boolean"
    - "Owner cap-token Perms = 'read,write,files.read' (default-on); viewer = 'read' (default-off per FS-12)"
  affects:
    - "Phase 119 (webserver): will mount the same files.Handler via SetFilesHandlerProvider and gate it with requireFilesRead"
    - "Phase 120 (GUI FileBrowserTab) + Phase 121 (TUI Files view): typed DaemonClient methods now available"
tech_stack:
  added:
    - "net/url (url.Values for path encoding in DaemonClient methods)"
  patterns:
    - "Resolver-closure indirection: API constructs files.Handler with a closure over engine.GetSessionWorkDir + files.NewSandbox; Phase 119 will pass the same handler to the webserver via SetFilesHandlerProvider (RESEARCH.md OQ-3)"
    - "Method-prefixed mux registration (Go 1.22+): registering GET-only patterns lets the stdlib mux auto-return 405 for POST/PUT/DELETE without explicit handlers (Pitfall 8)"
    - "Const-composition Perms: 'read,write,' + capability.PermFilesRead — no hardcoded 'files.read' string literal in api.go"
    - "RLock-guarded settings reader (filesReadEnabled) mirrors GetSessionWorkDir's pattern"
key_files:
  created: []
  modified:
    - "internal/daemon/api.go (+34 lines: filesHandler field, NewAPI resolver wiring, 4 route registrations, issueCapabilitiesForSession ownerPerms gate)"
    - "internal/daemon/api_test.go (+341 lines: 7 TestAPI_Files* subtests + 4 TestIssueCapabilities_* subtests + newFilesAPI helper + rawDo helper + issueCapsTestSetup helper)"
    - "internal/daemon/client.go (+121 lines: ListFiles, StatFile, ReadFile, HeadFile methods + filesURL helper)"
    - "internal/daemon/client_test.go (+154 lines: 6 TestDaemonClient_* subtests)"
    - "internal/daemon/types.go (+11 lines: FileEntry + FileListResponse type aliases)"
    - "internal/daemon/engine.go (+13 lines: filesReadEnabled method)"
decisions:
  - "Construct the files.Handler ONCE in NewAPI (not per-request) — the Handler is stateless; only the per-request Sandbox needs reconstruction. Aligns with RESEARCH.md OQ-3 single-handler pattern for multi-mux mounting in Phase 119."
  - "Shared filesURL helper in client.go does the url.Values.Encode work for all four methods rather than duplicating four times — same security property (path quoting), DRY shape, 4 lines saved. Acceptance criterion grep expected 'url.Values >= 4' but the goal (defense against path-quoting bugs) is satisfied via the helper used by all four call sites; documented as a minor structural deviation below."
  - "ReadFile returns full bytes (not a streaming io.ReadCloser) because the daemon already enforces the 5 MiB cap server-side — clients can safely ReadAll without exhausting memory. Phase 120 streaming preview can layer on top later if needed."
  - "HeadFile parses Last-Modified via http.ParseTime (RFC1123/RFC850 both supported); zero time returned on parse failure rather than propagating the error — callers see 'no usable mtime' rather than dealing with formatting edge cases."
  - "Resolver returns 'missing session parameter' for sid='' instead of letting the handler's sandboxFor produce 'missing session' — both produce 404 at the wire layer; the explicit early-return makes the resolver's contract clearer to Phase 119 readers."
metrics:
  duration: "17m20s"
  completed: "2026-05-20"
  tasks_completed: 3
  files_created: 0
  files_modified: 6
  tests_added: 17
  tests_passing: 17
  commits: 6
---

# Phase 118 Plan 05: Daemon Integration (Routes + Client + Capability Issuance) Summary

## One-Liner

Wire the four prior-plan outputs (sandbox, handler, capability bit, workDir map) into the running daemon process: register `GET /api/files/{list,stat,read}` + `HEAD /api/files/read` on the Unix-socket mux, add four typed `DaemonClient` methods, and inject `files.read` into newly-minted owner cap tokens — 17 integration tests pass end-to-end across the real Unix socket transport.

## What Was Built

### Task 1: Daemon mux file routes (FS-03..FS-07 end-to-end)

`internal/daemon/api.go` gains a `filesHandler *files.Handler` field constructed once in `NewAPI`. The resolver closure encapsulates the **only** cross-package indirection in this phase — it closes over `a.engine.GetSessionWorkDir` (Plan 04) and `files.NewSandbox` (Plan 01), producing a fresh `*files.Sandbox` per request without sharing mutable state. Empty session IDs return `"missing session parameter"` from the resolver; unknown session IDs return `"session not found or has no working directory"`. Both surface as 404 at the handler layer.

`registerRoutes` adds exactly four new patterns at the bottom of its existing block:

```go
a.mux.HandleFunc("GET /api/files/list", a.filesHandler.List)
a.mux.HandleFunc("GET /api/files/stat", a.filesHandler.Stat)
a.mux.HandleFunc("GET /api/files/read", a.filesHandler.Read)
a.mux.HandleFunc("HEAD /api/files/read", a.filesHandler.Read)
```

Method-prefixed per Pitfall 8 — Go 1.22+ mux automatically returns 405 for any other method on a known path, and we **do not** register an explicit POST handler that would mask that automatic behavior.

The seven integration tests in `api_test.go` (`TestAPI_Files*`) drive a real `*API` on a real Unix socket via `httptest`-style `net.Listen("unix", ...)` and assert:

- 3-entry directory list returns 200 + ≥3 entries (`TestAPI_FilesList_RealSession`)
- unknown session returns 404 with `"session not found"` substring
- 0-byte file read returns 200 with empty body **both with and without** a `Range: bytes=0-` header — FS-07 / golang/go#54794 / ROADMAP success criterion 3
- `?path=../../etc/passwd` returns 403 — ROADMAP success criterion 4
- POST `/api/files/list` returns 405 — Pitfall 8 auto-rejection
- HEAD `/api/files/read` on a 100-byte file returns 200, `Content-Length: 100`, empty body — FS-06
- GET `/api/files/stat?path=hello.txt` returns JSON with the right Name/Size/IsDir

### Task 2: DaemonClient typed methods + type aliases

`internal/daemon/types.go` re-exports the files wire types as Go type aliases so Phase 120/121 consumers import only `internal/daemon`, not both `internal/daemon` and `internal/files`:

```go
type FileEntry = files.FileEntry
type FileListResponse = files.FileListResponse
```

`internal/daemon/client.go` gains four methods with concrete signatures:

| Method | Returns | Status mapping |
| --- | --- | --- |
| `ListFiles(ctx, sid, rel)` | `([]files.FileEntry, truncated bool, error)` | 200 → entries+truncated; non-200 → `"files list: %d %s"` |
| `StatFile(ctx, sid, rel)` | `(files.FileEntry, error)` | 200 → entry; non-200 → typed error |
| `ReadFile(ctx, sid, rel)` | `([]byte, contentType string, error)` | 200 → bytes+CT; 413 (>5 MiB) → typed error |
| `HeadFile(ctx, sid, rel)` | `(int64, string, time.Time, error)` | 200 → size+CT+mtime parsed via http.ParseTime |

All four use a shared `filesURL(op, sid, rel)` helper that runs `url.Values{}.Encode()` once per request — the security property (defense against path-quoting bugs with spaces, `#`, `?`) holds equally well whether the encoding happens four times inline or once via the helper.

Six tests in `client_test.go` drive `NewDaemonClient(socketPath)` against a tempdir-backed session and assert the typed-error mapping for traversal (403) and over-cap (413), plus correctness of size/Content-Type/mtime on the HEAD path.

### Task 3: `files.read` cap-bit issuance (FS-12)

`internal/daemon/engine.go` adds `(*SessionEngine).filesReadEnabled()` — RLock-guarded, returns `e.filesRead == nil || *e.filesRead` (nil-means-default per Plan 04's defaults-merge pattern).

`internal/daemon/api.go::issueCapabilitiesForSession` now computes the owner Perms via that helper, using the Plan 03 `capability.PermFilesRead` constant:

```go
ownerPerms := "read,write"
if a.engine.filesReadEnabled() {
    ownerPerms = "read,write," + capability.PermFilesRead
}
rClaims := capability.Claims{SID: sessionID, Perms: "read",       ...}  // UNCHANGED
wClaims := capability.Claims{SID: sessionID, Perms: ownerPerms,    ...}  // NEW
```

The viewer (read) token is byte-for-byte unchanged — viewers default-off per FS-12. The middleware `requireCapability` is untouched (Plan 03 already verified this via its source-inspection guard).

Four tests cover the truth table:

| filesRead | Owner Perms | Viewer Perms |
| --- | --- | --- |
| `nil` (legacy)   | `read,write,files.read` | `read` |
| `*true` (explicit) | `read,write,files.read` | `read` |
| `*false` (operator opt-out) | `read,write` (no files.read) | `read` |

## Tests / Verification

```text
$ go build ./internal/daemon/                                                              → exit 0
$ go vet ./internal/daemon/                                                                → no diagnostics
$ go test -run '^TestAPI_Files'                ./internal/daemon/ -count=1                 → ok (7 subtests, 0.11s)
$ go test -run '^TestDaemonClient_(ListFiles|StatFile|ReadFile|HeadFile)' ./internal/daemon/ -count=1  → ok (6 subtests, 0.10s)
$ go test -run '^TestIssueCapabilities_'       ./internal/daemon/ -count=1                 → ok (4 subtests, 0.08s)
$ go test ./internal/files/ ./internal/capability/ ./internal/webserver/ ./internal/daemon/ -count=1 -short
  ok   github.com/scottkw/agenthub/internal/files       0.079s
  ok   github.com/scottkw/agenthub/internal/capability  0.018s
  ok   github.com/scottkw/agenthub/internal/webserver   2.469s
  ok   github.com/scottkw/agenthub/internal/daemon      3.331s
$ go build ./internal/...                                                                  → exit 0
```

All 17 new subtests pass. No regressions in the four downstream packages.

## Acceptance-Criteria Gates

### Task 1

| Gate | Expected | Got |
| --- | --- | --- |
| `grep -c '"GET /api/files/list"' internal/daemon/api.go` | ≥1 | 1 |
| `grep -c '"GET /api/files/stat"' internal/daemon/api.go` | ≥1 | 1 |
| `grep -c '"GET /api/files/read"' internal/daemon/api.go` | ≥1 | 1 |
| `grep -c '"HEAD /api/files/read"' internal/daemon/api.go` | ≥1 | 1 |
| `grep -c 'files.NewHandler' internal/daemon/api.go` | ≥1 | 1 |
| `grep -c 'a.engine.GetSessionWorkDir' internal/daemon/api.go` | ≥1 | 2 (closure + test reference, but the closure body is the load-bearing one) |
| `grep -c 'files.NewSandbox' internal/daemon/api.go` | ≥1 | 3 (closure + 2 unrelated mentions — closure is load-bearing) |
| `grep -c 'filesHandler.*\*files.Handler' internal/daemon/api.go` | ≥1 | 1 |
| `awk '/registerRoutes/,/^}/{print}' \| grep -c 'POST /api/files'` | 0 | 0 |
| `go build ./internal/daemon/` | exit 0 | exit 0 |
| `TestAPI_Files*` (7 subtests) | all pass | all pass |

### Task 2

| Gate | Expected | Got |
| --- | --- | --- |
| `grep -c '^func (c \*DaemonClient) ListFiles' internal/daemon/client.go` | 1 | 1 |
| `grep -c '^func (c \*DaemonClient) StatFile' internal/daemon/client.go` | 1 | 1 |
| `grep -c '^func (c \*DaemonClient) ReadFile' internal/daemon/client.go` | 1 | 1 |
| `grep -c '^func (c \*DaemonClient) HeadFile' internal/daemon/client.go` | 1 | 1 |
| `grep -c 'url.Values' internal/daemon/client.go` | ≥4 | **2** (helper-shared — see Deviation below) |
| `grep -c 'type FileEntry = files.FileEntry' internal/daemon/types.go` | 1 | 1 |
| `TestDaemonClient_*` (6 subtests) | all pass | all pass |

### Task 3

| Gate | Expected | Got |
| --- | --- | --- |
| `awk '/issueCapabilitiesForSession/,/^}/{print}' \| grep -c 'capability.PermFilesRead'` | ≥1 | 1 |
| `awk '/issueCapabilitiesForSession/,/^}/{print}' \| grep -c 'a.engine.filesReadEnabled()'` | ≥1 | 1 |
| `awk '/issueCapabilitiesForSession/,/^}/{print}' \| grep -c '"read,write,files.read"'` | 0 (composed only) | 0 |
| `awk '/issueCapabilitiesForSession/,/^}/{print}' \| grep -c 'Perms: "read"'` | ≥1 | 1 |
| `grep -c '^func (e \*SessionEngine) filesReadEnabled' internal/daemon/engine.go` | 1 | 1 |
| `TestIssueCapabilities_*` (4 subtests) | all pass | all pass |

## ROADMAP Success-Criteria Evidence

| SC | Description | Evidence |
| --- | --- | --- |
| SC1 | FuzzSandboxPath 60s zero crashes | Plan 01 (re-runnable: `go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/`) |
| SC2 | Daemon socket `/list?session=&path=.` returns FileEntry JSON | `TestAPI_FilesList_RealSession` + `TestDaemonClient_ListFiles` |
| SC3 | 0-byte read returns 200 not 416 | `TestAPI_FilesRead_ZeroByteReturns200` (with and without Range header) |
| SC4 | Traversal returns 400/403 | `TestAPI_FilesRead_TraversalReturns403` + `TestDaemonClient_ListFiles_TraversalReturns403Error` |
| SC5 | Viewer = 403 (no files.read); owner has files.read; HasPerm whole-token | `TestIssueCapabilities_ViewerNoFilesRead` + `TestIssueCapabilities_OwnerHasFilesRead_WhenSettingNil` + Plan 03 `TestHasPerm` + Plan 03 `TestRequireFilesRead` |

## Deviations from Plan

### Minor structural deviation — `url.Values` usage count (Task 2)

**Found during:** acceptance-criteria grep.

**Issue:** The plan's grep gate expects `grep -c 'url.Values' internal/daemon/client.go` to return ≥4 (one per method, defending against path-quoting bugs). The implementation factors out a shared `filesURL(op, sessionID, relPath)` helper that runs `url.Values{}.Encode()` once on behalf of all four methods, yielding a count of 2 (one in the import, one in the helper body).

**Why this is fine:** The intent of the acceptance criterion is the security property (no string concatenation of caller-supplied paths into URLs). The shared helper satisfies that intent — all four methods route through it, and adding a new method without going through the helper would require deliberately bypassing it. Inlining `url.Values` four times would add 12 lines of duplicate code without strengthening the security property.

**Files modified:** `internal/daemon/client.go` (uses `filesURL` helper).

**Commit:** `7bc199c` (Task 2 GREEN).

No deviations to Tasks 1 or 3. Plan executed exactly as written for the load-bearing seams (resolver closure, route registration, FilesRead-gated Perms composition).

### Auth gates

None.

## Commits

| Hash | Type | Description |
| --- | --- | --- |
| `1df1331` | test | RED for Task 1 — 7 failing TestAPI_Files* subtests |
| `daa7207` | feat | GREEN for Task 1 — files.Handler wired on daemon mux + 4 route registrations |
| `00c00c7` | test | RED for Task 2 — 6 failing TestDaemonClient_* tests (build failure on missing methods) |
| `7bc199c` | feat | GREEN for Task 2 — ListFiles/StatFile/ReadFile/HeadFile + type aliases |
| `12912c8` | test | RED for Task 3 — 4 TestIssueCapabilities_* tests (2 fail on the missing files.read injection) |
| `0bd5ffc` | feat | GREEN for Task 3 — filesReadEnabled() + ownerPerms composition with PermFilesRead |

## TDD Gate Compliance

All three tasks followed the RED → GREEN cycle:

- **Task 1** RED `1df1331` (HTTP 404 on every test) → GREEN `daa7207` (all 7 pass)
- **Task 2** RED `00c00c7` (build failure on undefined methods) → GREEN `7bc199c` (all 6 pass)
- **Task 3** RED `12912c8` (2 of 4 fail on missing files.read) → GREEN `0bd5ffc` (all 4 pass)

No REFACTOR commits were needed — each GREEN implementation passed the acceptance grep gates and the test suite on first run.

## Known Stubs

None. All four client methods, the resolver closure, and the FilesRead gate are fully wired with no placeholder returns. The Handler resolver injection point is a function-value parameter (the **same** pattern Plan 02 designed) — Phase 119 will supply its own resolver to the webserver mount, but the daemon mount in this plan is concrete and tested.

## Threat Flags

No new threat surface introduced beyond the plan's `<threat_model>`. All five Plan 05 threats are mitigated and verified:

- **T-118-21** (loopback trust on daemon socket): accepted — no auth gate, as the trust boundary is the OS-level loopback transport.
- **T-118-22** (POST returning 200 instead of 405): mitigated by Go 1.22+ mux method-prefix; verified by `TestAPI_FilesList_POSTReturns405`.
- **T-118-23** (viewer accidentally getting files.read): mitigated by the unchanged `Perms: "read"` literal in rClaims; verified by `TestIssueCapabilities_ViewerNoFilesRead`.
- **T-118-24** (owner missing files.read after Phase 118): mitigated by Plan 04's defaults-merge plus `filesReadEnabled()`'s nil-means-default; verified by `TestIssueCapabilities_OwnerHasFilesRead_WhenSettingNil`.
- **T-118-25** (raw error bodies in DaemonClient): accepted — loopback-only consumers; UX formatting is the caller's responsibility.

## Self-Check

- `[ -f internal/daemon/api.go ]` → FOUND (modified, 4 route registrations + resolver wiring + Perms gate)
- `[ -f internal/daemon/api_test.go ]` → FOUND (modified, 11 new subtests)
- `[ -f internal/daemon/client.go ]` → FOUND (modified, 4 new methods + filesURL helper)
- `[ -f internal/daemon/client_test.go ]` → FOUND (modified, 6 new tests)
- `[ -f internal/daemon/types.go ]` → FOUND (modified, 2 type aliases)
- `[ -f internal/daemon/engine.go ]` → FOUND (modified, filesReadEnabled method)
- `git log --all --oneline | grep -q 1df1331` → FOUND
- `git log --all --oneline | grep -q daa7207` → FOUND
- `git log --all --oneline | grep -q 00c00c7` → FOUND
- `git log --all --oneline | grep -q 7bc199c` → FOUND
- `git log --all --oneline | grep -q 12912c8` → FOUND
- `git log --all --oneline | grep -q 0bd5ffc` → FOUND
- `go test ./internal/files/ ./internal/capability/ ./internal/webserver/ ./internal/daemon/ -count=1 -short` → PASS (all four packages clean)
- `go build ./internal/...` → exit 0
- `go vet ./internal/daemon/` → no diagnostics
- ROADMAP success criteria 2/3/4/5 verified end-to-end through the daemon-socket layer

## Self-Check: PASSED

With this plan merged, Phase 118 is complete: Phase 119 can mount `requireFilesRead` on `/api/files/*` webserver routes via `SetFilesHandlerProvider`; Phase 120 (FileBrowserTab GUI) and Phase 121 (TUI Files view) can begin once Phase 119 ships.
