---
phase: 123-td-cleanup-write-sandbox-primitives-daemon-routes
verified: 2026-06-14T17:00:00Z
status: passed
score: 5/5
overrides_applied: 0
human_verification_completed: 2026-06-14
human_verification:
  - test: "Live daemon smoke: start agenthub daemon with an active session, then run: curl --unix-socket <daemon.sock> -X PUT 'http://localhost/api/files/write?session=<id>&path=hello.txt' --data-binary 'hello' and confirm 200 + FileWriteResponse; follow up with a GET read-back to verify byte-identical content."
    expected: "PUT returns 200 with JSON FileWriteResponse; subsequent GET returns identical bytes."
    result: "PASSED 2026-06-14 — fresh build, daemon run, shell session in /tmp/uat123-work. PUT /api/files/write → HTTP 200 {\"path\":\"hello.txt\",\"size\":11}; GET /api/files/read → byte-identical 'hello world'; on-disk content confirmed. Also verified live: denylist write to ~/.bashrc in a home-rooted session → HTTP 403 'Protected system file' (SC#3); wrong verb on write route → HTTP 405 (SC#5). Socket on macOS at ~/Library/Application Support/agenthub/daemon.sock."
---

# Phase 123: TD Cleanup + Write Sandbox Primitives + Daemon Routes Verification Report

**Phase Goal:** The internal/files/ sandbox has all write primitives (atomic write, rename, delete, mkdir, upload), the shell-RC denylist is enforced on all write paths, the two carried tech-debts (TD-4 and TD-5) are closed, and the daemon local-socket write routes are live — so every subsequent phase has a correct, trusted, fuzz-proven write API to build against.

**Verified:** 2026-06-14T17:00:00Z
**Status:** passed (live human verification completed 2026-06-14)
**HEAD:** ce9659fc0925b16724e8071d5d64fa5aa3892116
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | FuzzSandboxWrite 60s run reports zero crashes | VERIFIED | `go test -run '^$' -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` — PASS, ~65k execs, 0 crashes |
| 2 | WriteFileAtomic uses sibling temp + f.Sync + root.Rename; concurrent reader never sees partial file | VERIFIED | sandbox.go:188-215: `O_CREATE|O_EXCL` temp, `f.Sync()` at :199, `writeAtomicRename` at :211; TestWriteFileAtomic_ConcurrentReadNeverPartial passes under -race (200 concurrent writers) |
| 3 | Denylist returns 403 "Protected system file" for protected $HOME files across all five write methods (sandbox layer AND HTTP layer) | VERIFIED | TestDenylist_HomeRooted: 9 targets x 4 methods pass; TestHandlerWrite_DenylistForbidden and TestHandlerUpload_DenylistForbidden both pass; HTTP body asserts "Protected system file" |
| 4 | ExchangeJoinCodeAtURL parses 303 Location ?cap=<token> (TD-5 closed); shared remoteFilesHTTPClient unmutated | VERIFIED | client_remote_files.go:84-92: dedicated http.Client with ErrUseLastResponse; StatusSeeOther check at :114; TestExchangeJoinCode_303Success, _NoAutoFollow, _ErrorLocation (4 subtests), _EmptyCap, _SharedClientUnchanged — all pass under -race |
| 5 | Five daemon write routes registered method-prefixed; go test -race ./internal/files/... ./internal/daemon/... green | VERIFIED | api.go:149-153: PUT write, POST upload, DELETE delete, POST rename, POST mkdir all wired to a.filesHandler.*; full race suite: 5.368s + 10.799s, both OK |

**Score:** 5/5 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/files/sandbox.go` | WriteFileAtomic, Rename, Mkdir, MkdirAll, Delete, denylistCheck, ErrProtectedSystemFile | VERIFIED | All six methods + sentinel present at lines 63-337; per-op fresh os.OpenRoot; validateAndClean on all paths |
| `internal/files/types.go` | FileWriteResponse, FileOpResponse wire types | VERIFIED | FileWriteResponse at :53 (`path`, `size`); FileOpResponse at :64 (`path`, `ok`); camelCase JSON tags |
| `internal/files/write_test.go` | Unit tests for atomic write, rename, mkdir, delete, denylist | VERIFIED | 17 named test functions including TestWriteFileAtomic_ConcurrentReadNeverPartial, TestRename_DestinationTraversalRejected, TestDenylist_HomeRooted (9 targets x 4 methods) |
| `internal/files/sandbox_test.go` | FuzzSandboxWrite harness | VERIFIED | FuzzSandboxWrite at :367 immediately after FuzzSandboxPath; 45-seed corpus + 6 write seeds |
| `internal/files/write.go` | Handler.Write/Upload/Delete/Rename/Mkdir + writeWriteError | VERIFIED | All 5 handler methods at :38-182; writeWriteError at :190; maxUploadBytes = 50<<20 at :32 |
| `internal/daemon/api.go` | Five method-prefixed write-route registrations | VERIFIED | Lines 149-153: PUT/POST/DELETE/POST/POST wired to a.filesHandler.* with no auth block |
| `internal/daemon/client_remote_files.go` | 303-aware ExchangeJoinCodeAtURL with dedicated CheckRedirect client | VERIFIED | Lines 84-92: dedicated client; ErrUseLastResponse; StatusSeeOther check at :114 |
| `internal/daemon/client.go` | WriteFile, UploadFile, DeleteFile, RenameFile, MkdirFile on *DaemonClient | VERIFIED | Five methods at :505, :531, :574, :604, :634; each takes ctx context.Context first |
| `internal/tui/files_client.go` | FilesClient interface still 4 methods (scope guard) | VERIFIED | Interface has exactly 4 read methods; no write methods added; T-123-19 scope guard intact |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| sandbox.go write methods | validateAndClean + denylistCheck + os.OpenRoot | per-op fresh os.OpenRoot, validate first, denylist second | WIRED | All 5 write methods: validateAndClean → denylistCheck → os.OpenRoot (new per call) |
| daemon api.go mux | files Handler write methods | a.mux.HandleFunc with explicit verb prefix | WIRED | api.go:149-153 — a.filesHandler.Write/Upload/Delete/Rename/Mkdir |
| Handler.Upload | Sandbox.WriteFileAtomic | MaxBytesReader then ParseMultipartForm then filepath.Base then WriteFileAtomic | WIRED | write.go:77-78: MaxBytesReader at :77 BEFORE ParseMultipartForm at :78; filepath.Base at :95 |
| DaemonClient.ExchangeJoinCodeAtURL | Location header ?cap= token | dedicated http.Client with ErrUseLastResponse, detect 303 | WIRED | client_remote_files.go:84-155 |
| DaemonClient write methods | daemon write routes | http.NewRequestWithContext to filesURL("write"/"upload"/etc.) | WIRED | client.go:505-660; filesURL called with write op names |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| sandbox.go WriteFileAtomic | content []byte (caller-supplied) | caller → Sandbox → os.Root write | Yes — real fs write confirmed by TestWriteFileAtomic round-trip | FLOWING |
| write.go Handler.Write | body bytes from http.Request.Body | PUT request body | Yes — TestHandlerWrite_RoundTrip confirms byte-identical read-back | FLOWING |
| write.go Handler.Upload | multipart part data | POST multipart/form-data body, bounded by MaxBytesReader(50MiB) | Yes — TestHandlerUpload_DenylistForbidden and _FilenameSanitized confirm real write | FLOWING |
| client_remote_files.go ExchangeJoinCodeAtURL | cap token | 303 Location header from remote webserver | Yes — TestExchangeJoinCode_303Success confirms token extraction | FLOWING |
| client.go DaemonClient.WriteFile | FileWriteResponse | daemon PUT /api/files/write → sandbox → filesystem | Yes — TestDaemonClientWrite_RoundTrip confirms byte-identical read-back | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| FuzzSandboxWrite 60s zero crashes | `go test -run '^$' -fuzz=FuzzSandboxWrite -fuzztime=60s ./internal/files/` | PASS, ~65.8k execs, 0 crashes | PASS |
| Full race suite clean | `go test -race -count=1 ./internal/files/... ./internal/daemon/...` | files: 5.368s OK; daemon: 10.799s OK | PASS |
| Denylist tests | `go test -race ./internal/files/ -run 'TestDenylist'` | All subtests pass (home-rooted + non-home) | PASS |
| TD-5 ExchangeJoinCode tests | `go test -race ./internal/daemon/ -run 'TestExchangeJoinCode'` | 6/6 pass (303 success, no auto-follow, 4 error subtypes, empty cap, shared client unchanged) | PASS |
| Handler write tests | `go test -race ./internal/files/ -run 'TestHandlerWrite\|TestHandlerUpload\|TestHandlerRename\|TestHandlerMkdir\|TestHandlerDelete'` | 9/9 pass | PASS |
| Daemon route tests | `go test -race ./internal/daemon/ -run 'TestFilesWriteRoutes\|TestDaemonClientWrite\|TestDaemonClientUpload\|TestDaemonClientDelete\|TestDaemonClientRename\|TestDaemonClientMkdir'` | 13/13 pass | PASS |
| No O_TRUNC in sandbox.go | `grep -c 'O_TRUNC' internal/files/sandbox.go` | 0 | PASS |
| No os.TempDir in sandbox.go | `grep -c 'os.TempDir' internal/files/sandbox.go` | 0 | PASS |
| MaxBytesReader before ParseMultipartForm | `grep -n 'MaxBytesReader\|ParseMultipartForm' internal/files/write.go` | MaxBytesReader at :77, ParseMultipartForm at :78 (correct order) | PASS |
| FilesClient still 4 methods | `grep -c 'func\|interface' internal/tui/files_client.go` | Interface has 4 methods, no write methods | PASS |
| Live daemon curl smoke | requires running daemon | not run — manual only per VALIDATION.md | SKIP (human_needed) |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| FSW-01 | 123-01 | Sandbox.WriteFileAtomic atomic temp+Sync+rename; no O_TRUNC | SATISFIED | sandbox.go:172-216; TestWriteFileAtomic_ConcurrentReadNeverPartial pass |
| FSW-02 | 123-01 | Sandbox.Rename validates BOTH paths; destination traversal rejected | SATISFIED | sandbox.go:251-271; TestRename_DestinationTraversalRejected pass; uses native root.Rename |
| FSW-03 | 123-01 | Sandbox.Mkdir/MkdirAll; traversal rejected | SATISFIED | sandbox.go:278-316; TestMkdir, TestMkdirAll, TestMkdir_TraversalRejected pass |
| FSW-04 | 123-01 | Sandbox.Delete; sandbox-confined recursive remove | SATISFIED | sandbox.go:317-337; TestDelete_File, _RecursiveSubtree, _TraversalRejected pass |
| FSW-05 | 123-03 | Upload sanitizes via filepath.Base + validateAndClean | SATISFIED | write.go:95 filepath.Base; TestHandlerUpload_FilenameSanitized pass |
| FSW-06 | 123-01/03 | Denylist enforced in ALL write methods AND HTTP upload | SATISFIED | sandbox.go denylistCheck called in all 5 methods; TestDenylist_HomeRooted; TestHandlerUpload_DenylistForbidden pass |
| FSW-07 | 123-01 | FuzzSandboxWrite 60s 0-crash merge gate | SATISFIED | Ran live: 65.8k execs, 0 crashes, PASS |
| FSW-08 | 123-03 | Five auth-less method-prefixed daemon write routes | SATISFIED | api.go:149-153; TestFilesWriteRoutes_WrongVerb405, _Registered, _WriteRoundTrip pass |
| FSW-09 | 123-04 | DaemonClient write methods (WriteFile/UploadFile/DeleteFile/RenameFile/MkdirFile) | SATISFIED | client.go:505-660; all 7 client write tests pass under -race |
| FSW-10 | 123-02 | TD-5: ExchangeJoinCodeAtURL parses 303 Location ?cap=<token> | SATISFIED | client_remote_files.go:84-155; all 6 TestExchangeJoinCode_* pass |
| FSW-11 | 123-02 | TD-4: WR-01..05 verified/applied; WR-03/04/05 applied, WR-01/02 verified-already-satisfied | SATISFIED | server.go:556,583 (WR-01/02 comments); FileBrowserTab.tsx:399 (WR-03); FileRow.tsx:57-66 (WR-04); humanSize.ts:18-22 (WR-05) |
| FSW-12 | 123-03 | 50 MiB upload cap via MaxBytesReader BEFORE ParseMultipartForm | SATISFIED | write.go:32,77-78; TestHandlerUpload_OverCap413 returns 413, no truncated file |

**All 12 FSW requirements SATISFIED.**

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No TBD/FIXME/XXX/TODO debt markers found in any phase-modified file | — | None |

No anti-patterns detected. No unresolved debt markers. No placeholder returns or stub implementations found.

---

### Human Verification Required

#### 1. Live daemon socket write smoke test

**Test:** Start the agenthub daemon and create a file-browser session. Then run:
```
curl --unix-socket ~/.agenthub/daemon.sock \
  -X PUT 'http://localhost/api/files/write?session=<session-id>&path=hello.txt' \
  --data-binary 'hello'
```
**Expected:** HTTP 200 with JSON `{"path":"hello.txt","size":5}`; a subsequent `curl --unix-socket ~/.agenthub/daemon.sock 'http://localhost/api/files/read?session=<session-id>&path=hello.txt'` returns the exact bytes `hello`.

**Why human:** Requires a live running daemon process and a real active session. VALIDATION.md explicitly documents this as "Manual-Only Verification." All automated evidence (TestFilesWriteRoutes_WriteRoundTrip, TestDaemonClientWrite_RoundTrip, TestHandlerWrite_RoundTrip) proves the round-trip logic is correct at the handler, route, and client layers. Only the actual live Unix socket wire is unverifiable without a running daemon.

---

### Gaps Summary

No blocker gaps found. All five success criteria verified:

1. **FuzzSandboxWrite 60s gate** — VERIFIED. Ran live: ~65.8k executions, 0 crashes, PASS exit code.
2. **Atomic write semantics** — VERIFIED. sandbox.go uses `O_CREATE|O_EXCL`, `f.Sync()`, `root.Rename`; ConcurrentReadNeverPartial test passes under -race. Live curl smoke is documented as manual-only and does not constitute a phase failure.
3. **Denylist 403 enforcement** — VERIFIED at both sandbox layer (TestDenylist_HomeRooted: 9 targets x 4+ write methods) and HTTP layer (TestHandlerWrite_DenylistForbidden + TestHandlerUpload_DenylistForbidden both assert 403 "Protected system file").
4. **TD-5 ExchangeJoinCodeAtURL** — VERIFIED. Dedicated client with ErrUseLastResponse; StatusSeeOther detection; all 6 test cases pass under -race; shared client confirmed unmutated.
5. **Five daemon write routes + race gate** — VERIFIED. api.go:149-153; `go test -race -count=1 ./internal/files/... ./internal/daemon/...` = OK in 5.4s + 10.8s.

The only pending item is the live-daemon curl smoke, which VALIDATION.md explicitly designates as manual-only and which does not represent a code defect.

---

_Verified: 2026-06-14T17:00:00Z_
_Verifier: Claude (gsd-verifier)_
