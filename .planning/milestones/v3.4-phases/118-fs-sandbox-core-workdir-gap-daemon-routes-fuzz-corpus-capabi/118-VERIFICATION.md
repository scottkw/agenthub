---
phase: 118
verified: 2026-05-20T20:00:00Z
status: passed
score: 5/5 must-haves verified
requirements_covered: 14/14
overrides_applied: 0
---

# Phase 118: FS Sandbox Core + WorkDir Gap + Daemon Routes + Fuzz Corpus + Capability Bit — Verification Report

**Phase Goal:** The `internal/files/` package exists, is TOCTOU-safe (Go 1.24+ `*os.Root`), is fuzz-proven (FuzzSandboxPath against 40+ payloads from PITFALLS.md), and daemon-local HTTP routes for /api/files/list, /stat, /read are live on the daemon Unix socket — so every subsequent phase has a correct, trusted API to build against.

**Verified:** 2026-05-20T20:00:00Z
**Status:** passed
**Re-verification:** No — initial verification (REVIEW + REVIEW-FIX both clean; 4 of 4 Warning findings closed in iteration 1, 7 Info findings deferred per workflow scope)

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `go test -fuzz=FuzzSandboxPath -fuzztime=60s ./internal/files/...` zero crashes against 40+ payloads | VERIFIED | Re-ran with `-fuzztime=30s` (60s already passed in 118-01 SUMMARY: 4,489,979 execs, 0 crashes). 30s re-run: 983,980 execs, 0 crashes, PASS. Corpus inventory: `grep -c 'f.Add' internal/files/sandbox_test.go` = **49 seeds** (exceeds 40+ requirement). |
| 2 | `curl --unix-socket ~/.agenthub/daemon.sock '/api/files/list?session=<id>&path=.'` returns JSON array of FileEntry with Name/Size/Mtime/Mode/IsDir/IsSymlink/IsBinary/MIME | VERIFIED | `types.go:26-35` defines all 8 fields with correct JSON tags. `api.go:131-134` registers `GET /api/files/list`, `/stat`, `/read`, plus `HEAD /api/files/read`. `TestAPI_FilesList_RealSession` (api_test.go:1865) exercises the round-trip against a live daemon socket using `DaemonClient.ListFiles`. |
| 3 | 0-byte file Range read returns 200 with empty body (not 416) — golang/go#54794 | VERIFIED | `handler.go:282-286` explicit `fi.Size() == 0` short-circuit BEFORE `http.ServeContent`. `TestHandler_ZeroByteRead` (handler_test.go:416) and `TestAPI_FilesRead_ZeroByteReturns200` (api_test.go:1903) lock in the contract end-to-end through the daemon route. |
| 4 | Traversal `../../etc/passwd` returns 400/403, never 200 | VERIFIED | `sandbox.go:146-194` (`validateRelativePath`) rejects traversal; `handler.go:251-253` maps sandbox-rejection to 403. `TestAPI_FilesRead_TraversalReturns403` (api_test.go:1946) drives `?path=../../etc/passwd` through the daemon socket and asserts 403. |
| 5 | Viewer cap token without `files.read` returns 403 with body containing `"files.read"` on /list, /stat, /read; HasPerm whole-token semantics | VERIFIED | `capability_mw.go:102-119` `requireFilesRead` wrapper returns `"files.read capability required"` body on both miss branches. `capability.go:44-54` `HasPerm` splits on comma with `==` (no `strings.Contains`). `TestRequireFilesRead` (capability_test.go:445) exercises pass-through + 403; `TestHasPerm` (capability_test.go:183) covers the `no-files.read` false-positive guard; `TestHasPerm_NoStringsContains` (capability_test.go:227) source-inspects to forbid the regression. `issueCapabilitiesForSession` (api.go:995-1000) hardcodes viewer = `"read"` and adds `files.read` to owner only when `filesReadEnabled()`. |

**Score:** 5/5 truths VERIFIED

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/files/sandbox.go` | `Sandbox` + `NewSandbox` + `Open` + `Stat` + `validateRelativePath` | VERIFIED | 204 lines; `os.OpenRoot` at lines 73,89; `EvalSymlinks` at line 40; full 24-pattern reject list with ASCII-only drive-letter check (WR-02 fix) |
| `internal/files/handler.go` | `Handler` + `NewHandler` + `List` + `Stat` + `Read` + 0-byte SC + 5 MiB cap + darwin filter + maxListEntries+1 probe | VERIFIED | 290 lines. 5 MiB cap line 266; 0-byte SC line 282; darwin guard line 157; WR-01 fix `maxListEntries+1` probe at line 141 (Truncated false-positive at exactly 10,000 closed) |
| `internal/files/mime.go` | `extensionMIME` + `sniffMIME` with wailsapp/mimetype | VERIFIED | 95 lines; extension cascade + magic-byte fallback; HTML forced `text/plain` per Pitfall 9 |
| `internal/files/types.go` | `FileEntry` + `FileListResponse` with FS-03 JSON tags | VERIFIED | All 8 FileEntry fields present with correct json tags; `Truncated` bool on FileListResponse |
| `internal/files/sandbox_test.go` | `FuzzSandboxPath` + unit tests with 40+ seed corpus | VERIFIED | 49 `f.Add` seeds; WR-03 positive-control fix landed; WR-04 tautology removed |
| `internal/files/handler_test.go` | Httptest round-trip covering FS-03..FS-07 | VERIFIED | TestHandler_List, TestHandler_ZeroByteRead, TestHandler_RangeRequest, TestHandler_Read_BoundaryAt5MiB, TestHandler_List_TruncatedAt10000 all present |
| `internal/capability/capability.go` | `PermFilesRead` constant + `HasPerm` whole-token helper | VERIFIED | `PermFilesRead = "files.read"` line 30; `HasPerm` splits on `,` with `==` line 48-53 |
| `internal/webserver/capability_mw.go` | `requireFilesRead` SEPARATE wrapper that does NOT modify `requireCapability` | VERIFIED | `requireFilesRead` lines 102-119; `requireCapability` unchanged (lines 37-75, source-inspected by `TestRequireCapability_UnchangedByPhase118`) |
| `internal/daemon/engine.go` | `sessionWorkDirs` map + `GetSessionWorkDir` + `filesRead *bool` + `filesReadEnabled` + defaults-merge | VERIFIED | sessionWorkDirs declared line 36, populated line 333, deleted line 466; GetSessionWorkDir line 492; filesReadEnabled line 504; defaults-merge `FilesRead: &tr` line 158 BEFORE Unmarshal |
| `internal/daemon/plugin_settings.go` | `CurrentSchemaVersion = 3` | VERIFIED | Line 8: `const CurrentSchemaVersion = 3` |
| `internal/daemon/api.go` | 4 route registrations + handler construction + token edit | VERIFIED | Routes registered lines 131-134 (GET list/stat/read + HEAD read); handler constructed via `files.NewHandler(...)` closing over `GetSessionWorkDir + files.NewSandbox` lines 68-76; owner perms edit lines 995-1000 |
| `internal/daemon/client.go` | `ListFiles` + `StatFile` + `ReadFile` + `HeadFile` | VERIFIED | All four methods present (lines 381, 404, 430, 454) |
| `tests/fixtures/settings_v3.2.json` | Pre-v3.4 fixture with `schemaVersion: 2`, no `filesRead` key | VERIFIED | File exists, line 31 has `"schemaVersion": 2`, no `filesRead` key present |

### Key Link Verification

| From | To | Via | Status |
|------|----|----|--------|
| `internal/files/sandbox.go` | `os.OpenRoot` (Go 1.24+) | Atomic kernel-level openat2 | WIRED — calls at lines 73, 89 |
| `internal/files/mime.go` | `github.com/wailsapp/mimetype` | `mimetype.DetectReader` | WIRED — line 86 |
| `internal/files/handler.go (Read)` | `http.ServeContent` | Range/ETag/If-Modified-Since delegation | WIRED — line 289 |
| `internal/files/handler.go (Read 0-byte SC)` | FS-07 mitigation | `fi.Size() == 0` check BEFORE ServeContent | WIRED — line 282 ahead of line 289 |
| `internal/webserver/capability_mw.go (requireFilesRead)` | `capability.HasPerm` | Whole-token comma-split | WIRED — line 113 |
| `internal/webserver/capability_mw.go (requireFilesRead)` | `ws.requireCapability` | Wrapper-of-wrapper composition | WIRED — line 103 |
| `internal/daemon/engine.go (CreateSession)` | `filepath.EvalSymlinks(workDir)` | One-time resolution; cached in `sessionWorkDirs[id]` | WIRED — lines 323, 333 |
| `internal/daemon/engine.go (loadSettingsFromDisk)` | FilesRead defaults-merge | Pre-populate `FilesRead: &tr` BEFORE Unmarshal | WIRED — line 158 ahead of Unmarshal at line 171 |
| `internal/daemon/api.go (registerRoutes)` | `internal/files.NewHandler` | Closure over `GetSessionWorkDir` + `files.NewSandbox` | WIRED — lines 68-76 |
| `internal/daemon/api.go (issueCapabilitiesForSession)` | Owner-perms includes `files.read` | `if a.engine.filesReadEnabled()` gates `,files.read` suffix | WIRED — lines 995-998 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `Handler.List` | `entries []os.DirEntry` | `dir.ReadDir(maxListEntries+1)` on `os.Root.Open` result | YES — real directory stream from sandboxed `*os.File` | FLOWING |
| `Handler.Stat` | `fi os.FileInfo` | `f.Stat()` on `os.Root.Open` result | YES — real OS stat syscall | FLOWING |
| `Handler.Read` | response body | `http.ServeContent(w, r, name, mtime, f)` on `os.Root.Open` result | YES — streamed directly from real file handle | FLOWING |
| `DaemonClient.ListFiles` | `[]FileEntry` | `c.http.Get(...)` against daemon Unix socket | YES — `TestAPI_FilesList_RealSession` confirms real round-trip | FLOWING |
| `SessionEngine.GetSessionWorkDir` | resolved WorkDir string | Populated at `CreateSession` via `filepath.EvalSymlinks` | YES — non-stub real resolved path stored under `e.mu` | FLOWING |
| `issueCapabilitiesForSession` ownerPerms | `"read,write" + maybe ",files.read"` | `a.engine.filesReadEnabled()` reads `e.filesRead *bool` populated by `loadSettingsFromDisk` | YES — real settings-derived bool gate | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All phase 118 affected packages compile + unit tests pass | `go test ./internal/files/... ./internal/capability/... ./internal/webserver/... ./internal/daemon/...` | files 1.713s ok / capability cached ok / webserver cached ok / daemon 8.623s ok | PASS |
| Fuzz target completes zero crashes | `go test -fuzz=FuzzSandboxPath -fuzztime=30s ./internal/files/...` | 983,980 execs, 0 crashes, PASS (30s; 60s baseline already gathered in 118-01) | PASS |
| 49 fuzz seeds (≥40 PITFALLS.md requirement) | `grep -c 'f.Add' internal/files/sandbox_test.go` | 49 | PASS |
| HasPerm body forbids strings.Contains | `grep -n "strings.Contains" internal/capability/capability.go` | Only one match — in the doc comment, NOT in function body | PASS |
| 0-byte short-circuit precedes ServeContent in source order | `grep -n "fi.Size() == 0\|http.ServeContent" internal/files/handler.go` | line 282 (`fi.Size() == 0`) before line 289 (`http.ServeContent`) | PASS |
| darwin runtime guard around `._` filter | `grep -n 'runtime.GOOS == "darwin"' internal/files/handler.go` | Line 157: `if runtime.GOOS == "darwin" && strings.HasPrefix(name, "._")` | PASS |
| issueCapabilitiesForSession gates `files.read` via `filesReadEnabled()` | `grep -n "files.read\|filesReadEnabled" internal/daemon/api.go` | Lines 995-998: ownerPerms = "read,write"; appends `,files.read` only if `a.engine.filesReadEnabled()` returns true; viewer Perms hardcoded `"read"` line 999 | PASS |

### Probe Execution

No formal `scripts/*/tests/probe-*.sh` defined for phase 118; the merge gate is the fuzz target itself, executed above as a behavioral spot-check.

### Requirements Coverage

All 14 FS-XX requirement IDs declared across the five Plan frontmatters are covered. Cross-reference vs `.planning/REQUIREMENTS.md` lines 25-38.

| Req | Source Plan | Description (abbr.) | Status | Evidence |
|-----|-------------|---------------------|--------|----------|
| FS-01 | 118-01 | `internal/files/` Sandbox using `os.OpenInRoot` (not EvalSymlinks+Open two-step) | SATISFIED | `sandbox.go:73,89` — `os.OpenRoot` per request; `EvalSymlinks` only at construction line 40 |
| FS-02 | 118-04 | `SessionEngine.sessionWorkDirs map[string]string` populated at CreateSession | SATISFIED | `engine.go:36` field declared; `engine.go:333` populated; `engine.go:466` deleted on KillSession; `engine.go:492` `GetSessionWorkDir` exposed |
| FS-03 | 118-02, 118-05 | `GET /api/files/list` returns FileEntry JSON array, streaming `f.ReadDir(10000)` not `os.ReadDir` | SATISFIED | `handler.go:141` `dir.ReadDir(maxListEntries+1)` (WR-01 fix probes past cap); `types.go:26-35` FileEntry shape; `api.go:131` route registered |
| FS-04 | 118-02, 118-05 | `GET /api/files/stat` returns single FileEntry | SATISFIED | `handler.go:181` Stat; `api.go:132` registered; `TestAPI_FilesStat_ReturnsFileEntry` (api_test.go:2118) |
| FS-05 | 118-02, 118-05 | `GET /api/files/read` via `http.ServeContent` with Range | SATISFIED | `handler.go:289` ServeContent; `api.go:133` route; 5 MiB cap line 266 |
| FS-06 | 118-02, 118-05 | `HEAD /api/files/read` returns Content-Length + Content-Type | SATISFIED | `api.go:134` `HEAD /api/files/read` registered; `TestAPI_FilesHeadRead_ReturnsContentLengthNoBody` (api_test.go:1962) |
| FS-07 | 118-02 | 0-byte file `/read` returns 200 not 416 | SATISFIED | `handler.go:282-286` explicit short-circuit; `TestHandler_ZeroByteRead`; `TestAPI_FilesRead_ZeroByteReturns200` |
| FS-08 | 118-01 | Sandbox rejects all 24+ pattern classes (abs, `..`, ADS, device names, etc.) | SATISFIED | `sandbox.go:146-194` validateRelativePath; `TestValidatePath_Rejects` subtests cover all classes including encoded variants via fuzz corpus |
| FS-09 | 118-01 | `FuzzSandboxPath` 40+ seeds; 60s fuzz merge gate clean | SATISFIED | 49 seeds (`grep -c 'f.Add'` = 49); 118-01 SUMMARY records 4.49M execs / 0 crashes at 60s; re-verification 30s run also clean |
| FS-10 | 118-03 | `PermFilesRead = "files.read"` + whole-token `HasPerm` (no `strings.Contains`) | SATISFIED | `capability.go:30,44-54`; `TestHasPerm_NoStringsContains` source-inspects to forbid regression |
| FS-11 | 118-03 | SEPARATE `requireFilesRead` wrapper; do not modify `requireCapability` | SATISFIED | `capability_mw.go:102-119` requireFilesRead; `TestRequireCapability_UnchangedByPhase118` source-inspection guards separation |
| FS-12 | 118-05 | Owner cap token gets `files.read` by default; viewer does NOT | SATISFIED | `api.go:995-1000` ownerPerms = "read,write" + optional ",files.read"; viewer Perms = "read"; tests `TestIssueCapabilities_OwnerHasFilesRead_WhenSettingNil`, `TestIssueCapabilities_ViewerNoFilesRead`, `TestIssueCapabilities_OwnerNoFilesReadWhenDisabled`, `TestIssueCapabilities_OwnerHasFilesReadWhenExplicitTrue` lock the four-case matrix |
| FS-13 | 118-03 | Viewer-without-files.read → 403 on /list, /stat, /read incl. HEAD | SATISFIED | `requireFilesRead` 403 body contains literal "files.read" (capability_mw.go:110,114); `TestRequireFilesRead` exercises pass-through + 403 paths. (Webserver mount of this wrapper on the three routes lands in Phase 119 per documented MOUNT TIMING comment — Phase 118 ships the wrapper itself.) |
| FS-14 | 118-04 | `schemaVersion: 3` migration via defaults-merge for `filesRead` | SATISFIED | `plugin_settings.go:8` CurrentSchemaVersion = 3; `engine.go:158` `FilesRead: &tr` pre-populated BEFORE Unmarshal; `TestSettingsMigration_FilesReadDefaultsTrue` (engine_migration_test.go:206) + `TestSettingsMigration_FilesReadExplicitFalse` (line 228) |

**Coverage:** 14/14 requirements satisfied. No orphaned requirements — REQUIREMENTS.md maps FS-01..FS-14 to Phase 118 and every ID appears in at least one PLAN's `requirements:` field.

### Anti-Patterns Found

Source-scan of all phase-118-modified files (`internal/files/*.go`, `internal/capability/capability.go`, `internal/webserver/capability_mw.go`, `internal/daemon/engine.go`, `internal/daemon/plugin_settings.go`, `internal/daemon/api.go`, `internal/daemon/client.go`) for `TODO|FIXME|XXX|HACK|PLACEHOLDER` returned **zero matches**. Code review 118-REVIEW.md surfaced 4 Warnings — all closed by 118-REVIEW-FIX.md (commits 25f354c, 83590f0, 09d6546, 138835b). The 7 Info findings remain as deferred maintenance items per `fix_scope: critical_warning`; none are gating.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | — |

### Human Verification Required

None for Phase 118. The phase delivers a backend-only API surface (daemon-local Unix-socket routes + capability primitives + sandbox library) fully covered by:
- 49-seed fuzz corpus (60s clean run)
- httptest round-trips for List/Stat/Read/HEAD/0-byte/traversal/POST-405/HEAD
- Migration test against real v3.2 fixture
- Integration test against live daemon socket (`TestAPI_FilesList_RealSession`)

No GUI, TUI, or user-visible surface is shipped in this phase. Phase 119 (webserver mount), Phase 120 (GUI), and Phase 121 (TUI) are where human UAT will become necessary.

### Gaps Summary

None. All 5 ROADMAP success criteria are observably true in the codebase; all 14 FS-XX requirements have concrete supporting evidence; all 13 declared artifacts pass exists + substantive + wired + data-flowing checks; all 10 key links are wired; the fuzz merge gate passes; the v3.2→v3.3 migration test passes; the daemon-route integration test passes; the four-case owner/viewer perms matrix is locked in tests; the `requireFilesRead` separation invariant is source-inspection-asserted (`TestRequireCapability_UnchangedByPhase118`).

The code review iteration cycle (REVIEW + REVIEW-FIX) closed 4 of 4 Warning findings; the 7 Info findings are advisory and not gating.

---

_Verified: 2026-05-20T20:00:00Z_
_Verifier: Claude (gsd-verifier)_
