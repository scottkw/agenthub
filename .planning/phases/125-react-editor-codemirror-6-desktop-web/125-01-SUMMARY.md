---
phase: 125-react-editor-codemirror-6-desktop-web
plan: 01
subsystem: files-handler, webserver-test, playwright-fixture, e2e-setup
tags: [optimistic-concurrency, etag, if-match, 412, vendor-drift, playwright, write-cap]
dependency_graph:
  requires:
    - 123-01-SUMMARY.md  # Sandbox.Stat shipped
    - 124-02-SUMMARY.md  # Handler layer and capability_mw shipped
  provides:
    - If-Match/412 precondition gate in Handler.Write (all three surfaces inherit)
    - ETag header on Handler.Read (client echo path for write conflicts)
    - TestWrite_IfMatch_Match / Mismatch / NewFile (three Go unit tests under -race)
    - TestCodeMirrorVersionsMatchPnpmLock (vendor-drift gate, trivially passes pre-Plan-02)
    - WRITE_CAP fixture cap (files.write) in playwright-fixture + global-setup parse + fixture-env interface
  affects:
    - internal/files/write.go
    - internal/files/handler.go
    - internal/files/write_test.go
    - internal/webserver/vendor_drift_test.go
    - cmd/playwright-fixture/main.go
    - frontend/e2e/global-setup.ts
    - frontend/e2e/fixture-env.ts
tech_stack:
  added: []
  patterns:
    - If-Match precondition inline in HTTP handler (NOT in writeWriteError)
    - ETag format "<UnixNano>-<size>" quoted — load-bearing validator contract
    - Sandbox.Stat as os.Root-confined validator source
    - pnpm-lock.yaml parse loop (reuse of xterm pattern; swap regex for @codemirror/)
    - Playwright cap fixture pattern (mirror viewerClaims exactly)
key_files:
  created: []
  modified:
    - internal/files/write.go
    - internal/files/handler.go
    - internal/files/write_test.go
    - internal/webserver/vendor_drift_test.go
    - cmd/playwright-fixture/main.go
    - frontend/e2e/global-setup.ts
    - frontend/e2e/fixture-env.ts
decisions:
  - "If-Match precondition is inline in Handler.Write (not in writeWriteError) — it is a precondition on the HTTP request, not a write-primitive error"
  - "ETag format is quoted <UnixNano>-<size> (RESEARCH Open Q6 resolution: server emits, client echoes verbatim — eliminates RFC3339-vs-UnixNano mismatch)"
  - "CodeMirror vendor-drift test compares package.json declared versions vs pnpm-lock.yaml resolved versions (no web/vendor/ manifest — CM6 is Vite-bundled)"
  - "WRITE_CAP fixture retains existing viewer (read-only) cap intact for 403-without-cap scenario coverage"
metrics:
  duration: "~25 minutes"
  completed: "2026-06-14T19:22:04Z"
  tasks_completed: 3
  tasks_total: 3
  files_modified: 7
  commits: 3
---

# Phase 125 Plan 01: Wave 0 Scaffolding — If-Match/ETag + Vendor-Drift Gate + WRITE_CAP Fixture

**One-liner:** Server emits ETag on Read and enforces 412 on If-Match mismatch in Write, backed by Go unit tests; CodeMirror parity gate and Playwright write-cap fixture unblock all downstream plans.

## Tasks Completed

| # | Name | Commit | Status |
|---|------|--------|--------|
| 1 | If-Match/412 precondition + ETag emission | 0894b4d | DONE |
| 2 | Go unit tests for If-Match (TestWrite_IfMatch*) | 49be062 | DONE |
| 3 | CodeMirror vendor-drift gate + Playwright WRITE_CAP fixture | 0c79c78 | DONE |

## What Was Built

### Task 1: If-Match/412 + ETag (0894b4d)

**`internal/files/write.go`** — `Handler.Write` now checks the `If-Match` header before calling `sb.WriteFileAtomic`. When `If-Match` is present and not `"*"`, the handler calls `sb.Stat(rel)` to get the on-disk `os.FileInfo`. If the target exists, it computes `cur := fmt.Sprintf("%q", fmt.Sprintf("%d-%d", fi.ModTime().UnixNano(), fi.Size()))` and returns `http.StatusPreconditionFailed` (412) if the header doesn't match. Missing target (new file) proceeds as before. Added `"fmt"` to imports.

**`internal/files/handler.go`** — `Handler.Read` now emits `ETag: "<UnixNano>-<size>"` before both the zero-byte short-circuit branch and the `http.ServeContent` call, resolving RESEARCH Open Q6: the client echoes the ETag verbatim as `If-Match`, eliminating any RFC3339-vs-UnixNano format-mismatch risk. Added `"fmt"` to imports.

### Task 2: TestWrite_IfMatch* unit tests (49be062)

Three new tests in `internal/files/write_test.go` exercising `Handler.Write` through `httptest.NewRequest`/`httptest.NewRecorder`:

- **TestWrite_IfMatch_Match**: Seeds a file, reads on-disk validator via `os.Stat`, issues PUT with correct `If-Match` → asserts 200 and content updated.
- **TestWrite_IfMatch_Mismatch**: Seeds a file, issues PUT with fabricated stale `If-Match` (`"0-0"`) → asserts 412 and original bytes unchanged on disk.
- **TestWrite_IfMatch_NewFile**: Issues PUT with no `If-Match` to a non-existent path → asserts 200 and file created.

All three pass under `-race`.

Added `invokeWrite` and `validatorFor` helpers to the test file. The check lives in the HTTP handler layer (not the Sandbox primitive), so the tests drive `h.Write` directly.

### Task 3: CodeMirror drift gate + WRITE_CAP (0c79c78)

**`internal/webserver/vendor_drift_test.go`** — New `TestCodeMirrorVersionsMatchPnpmLock` test reuses the `pnpmXtermKeyRe` parse loop pattern but with `pnpmCMKeyRe` matching `@codemirror/*` and bare `codemirror` keys. Reads `frontend/package.json` declared deps and compares stripped versions against pnpm-lock.yaml resolved versions. Passes trivially (zero CM packages declared) until Plan 02 installs the packages, at which point it becomes the T-125-SC gate. **No `web/vendor/codemirror/` directory created** — CM6 is Vite-bundled.

**`cmd/playwright-fixture/main.go`** — Added `writeClaims` (Perms: `"read,files.read,files.write"`, GrantID: `grant-playwright-fixture-write`), signed with the existing `fixedTestKey`, registered with `ws.AddGrant`. Added `fmt.Printf("WRITE_CAP=%s\n", writeToken)` in the stdout emission block. Existing viewer and owner caps unchanged.

**`frontend/e2e/global-setup.ts`** — Added `writeCap string` to `FixtureEnv` interface. Added `let writeCap = ''` variable and `if (line.startsWith('WRITE_CAP=')) writeCap = ...` parse. Added `writeCap` to the `resolveEnv` object.

**`frontend/e2e/fixture-env.ts`** — Added `writeCap: string` field to `FixtureEnv` interface with JSDoc. Added `writeAppUrl(env?)` helper function mirroring `viewerAppUrl`.

## Verification Results

```
go test ./internal/files/... -run TestWrite_IfMatch -count=1 -race -v
--- PASS: TestWrite_IfMatch_Match (0.01s)
--- PASS: TestWrite_IfMatch_Mismatch (0.00s)
--- PASS: TestWrite_IfMatch_NewFile (0.01s)
ok  github.com/scottkw/agenthub/internal/files  1.034s

go test ./internal/webserver/... -run CodeMirror -count=1 -v
--- PASS: TestCodeMirrorVersionsMatchPnpmLock (0.00s)
ok  github.com/scottkw/agenthub/internal/webserver  0.009s

go test ./internal/files/... ./internal/webserver/... -count=1 -race
ok  github.com/scottkw/agenthub/internal/files     4.883s
ok  github.com/scottkw/agenthub/internal/webserver 3.956s
```

grep -n StatusPreconditionFailed internal/files/write.go → line 67 (PASS)
grep -c 'ETag' internal/files/handler.go → 4 (PASS, >= 1 required)
grep -c 'WRITE_CAP' cmd/playwright-fixture/main.go → 1 (PASS)
grep -c 'WRITE_CAP=' frontend/e2e/global-setup.ts → 2 (PASS)
grep -c 'writeCap' frontend/e2e/fixture-env.ts → 2 (PASS)

go build -tags=playwrightfixture ./cmd/playwright-fixture/... → exits 0

TypeScript: e2e files have 3 pre-existing type errors (Node.js @types/node version mismatch on Dirent/ChildProcess types) that also appear in the main repo before these changes. No new errors introduced by the modifications.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — this plan implements backend infrastructure, no frontend stubs.

## Threat Flags

None — all new surface (If-Match validator gate) is covered by the plan's existing threat model (T-125-01).

## Self-Check: PASSED

Files verified:
- internal/files/write.go — FOUND
- internal/files/handler.go — FOUND
- internal/files/write_test.go — FOUND
- internal/webserver/vendor_drift_test.go — FOUND
- cmd/playwright-fixture/main.go — FOUND
- frontend/e2e/global-setup.ts — FOUND
- frontend/e2e/fixture-env.ts — FOUND

Commits verified:
- 0894b4d feat(125-01): If-Match/412 precondition + ETag emission
- 49be062 test(125-01): add TestWrite_IfMatch* (match/mismatch/new-file)
- 0c79c78 feat(125-01): CodeMirror vendor-drift gate + Playwright WRITE_CAP fixture
