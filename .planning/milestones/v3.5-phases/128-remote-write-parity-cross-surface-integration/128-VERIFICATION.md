---
phase: 128-remote-write-parity-cross-surface-integration
verified: 2026-06-14T00:00:00Z
status: human_needed
score: 6/6 requirements verified at code/test level
requirements_covered: 6/6 (RMW-01..06)
overrides_applied: 0
human_verification:
  - test: "Two-machine tailnet write UAT — Machine A web-shares a session; Machine B GUI edits + saves a file; Machine B TUI edits + saves a file; cross-surface write parity confirmed"
    expected: "All write operations (edit/save, upload, delete, rename, mkdir, cross-dir move) succeed and produce byte-identical state across GUI, TUI, and web; cap-expiry mid-edit preserves the buffer and shows 'access expired'; a v3.4 peer shows the verbatim older-version message on write"
    why_human: "Real tailnet round-trip with two physical machines proves the full daemon-proxy + cap-deposit + write persistence + v3.5 RemoteFilesClient combination against actual network latency and a production Wails build (per project memory: Wails build requires -tags wailsassets)"
---

# Phase 128: Remote Write Parity + Cross-Surface Integration — Verification Report

**Phase Goal:** Write operations (edit/save, upload, delete, rename, mkdir, cross-dir move) on remote tailnet sessions are identical across desktop GUI, TUI, and web-share viewer. Correct, user-visible error messages on cap-expiry (401) and v3.4-peer 405. Cross-surface byte-equivalence proven by automated tests across 3 independent observers. Two-machine UAT checklist committed for operator execution to close umbrella Issue #24.

**Verified:** 2026-06-14
**Status:** human_needed (automated layer green; two-machine write UAT deferred to operator per STATE.md TD-3)
**Re-verification:** No — initial verification after Phase 128 closure

## Requirements Traceability

| Req | Plans | Tests / Artifacts | Status |
|-----|-------|-------------------|--------|
| **RMW-01** Remote write routes exist on the daemon proxy (`PUT /api/files/remote/{sid}/write`, `DELETE`, `POST rename`, `POST mkdir`) | 128-01 (405 guard), 128-02 (401 guard), 128-03 (write-parity harness) | `internal/daemon/remote_files_write_parity_test.go::TestRemoteFilesWrite_CrossSurface_*` (Observer A proxy write-then-read); `internal/daemon/remote_files_test.go` pre-existing proxy route tests | VERIFIED (automated) |
| **RMW-02** TUI `RemoteFilesClient` implements write methods (WriteFile, DeleteFile, RenameFile, MkdirFile) with correct HTTPS+cap routing | 128-01 (405 guard), 128-02 (401 guard), 128-03 (3-observer harness) | `internal/tui/remote_files_client_test.go::TestRemoteFilesClient_*` write-verb tests; `TestRemoteFilesWrite_CrossSurface_*` Observer B direct writes; `assertNoCapInError` on all error paths | VERIFIED (automated) |
| **RMW-03** Write-then-read byte-equivalence across 3 independent observers (daemon proxy Go + RemoteFilesClient Go + Playwright HTTPS) | 128-03 (3-observer parity proof) | `TestRemoteFilesWrite_CrossSurface_ObserverA_WriteThenRead`, `_ObserverB_WriteThenRead`, `_CrossObserver_AWritesBReads`, `_CrossObserver_BWritesAReads`; Playwright scenario 18 (`files-browser.spec.ts`) | VERIFIED (automated) |
| **RMW-04** v3.4-peer upstream 405 surfaces verbatim "older version" message on all write verbs (Go TUI + TS GUI/web); byte-identical const in both languages | 128-01 (TDD — 405 version gate) | `TestRemoteFilesClient_405_ErrRemotePeerNoWriteSupport` (Go, 4 write methods); `filesApi.test.ts` + `useFilesWrite.test.tsx` (TS, 27/27); grep gate: `remotePeerOutdatedMessage` in Go + `REMOTE_PEER_OUTDATED_MESSAGE` in TS byte-equal | VERIFIED (automated) |
| **RMW-05** Cap-expiry 401 mid-edit preserves edit buffer + surfaces distinct "access expired" message; in-flight upload 401 removes queue entry (no stuck progress bar) | 128-02 (TDD — 401 cap-expiry) | `TestRemoteFilesClient_401_ErrRemoteCapExpired` (Go, 6 sub-tests); TS vitest 37/37 (expired outcome, upload queue filter, buffer preservation — T-125-08 locked); `ACCESS_EXPIRED_MESSAGE` grep gate | VERIFIED (automated) |
| **RMW-06** Phase 122 remote-read suite shows zero regressions after all Phase 128 write-verb mappings applied | 128-04 (regression guard) | `go test ./internal/daemon/ -run 'TestRemoteFiles_(List\|Stat\|Read\|CrossSurface_Parity)' -race -count=1`: 7 tests PASS (see Automated Test Results below) | VERIFIED (automated) |

**Score:** 6/6 requirements verified at the automated layer. One human UAT item deferred to operator (requires two physical tailnet machines; STATE.md TD-3).

## What Was Built

### Plan 01 — v3.4 Peer 405 Version Gate (RMW-04)

- `internal/tui/remote_files_client.go`: `remotePeerOutdatedMessage` const (verbatim SC3) + `ErrRemotePeerNoWriteSupport` sentinel; all 4 write methods check `StatusMethodNotAllowed` before the generic `!= OK` guard
- `internal/tui/update.go`: `applyFilesOpMsg` maps `errors.Is(msg.err, ErrRemotePeerNoWriteSupport)` → verbatim toast copy
- `frontend/src/lib/filesApi.ts`: `REMOTE_PEER_OUTDATED_MESSAGE` export const (byte-identical to Go) + `isMethodNotAllowed()` predicate
- `frontend/src/lib/useFilesWrite.ts`: `WriteOutcome` extended with `'peer-outdated'`; 405 catch branch added before generic
- `frontend/src/components/FileBrowserTab.tsx`: `'peer-outdated'` outcome documented in dispatch comment
- TDD gate: Go RED `b35956e` → GREEN `f98557c`; TS RED `47b1c7d` → GREEN `ab587e9`

### Plan 02 — Cap-Expiry 401 Handling (RMW-05)

- `internal/tui/remote_files_client.go`: `ErrRemoteCapExpired` sentinel; all 4 write methods check `StatusUnauthorized` before generic guard
- `internal/tui/update.go`: maps `ErrRemoteCapExpired` → "access expired" TUI status line
- `frontend/src/lib/useFilesWrite.ts`: `WriteOutcome` extended with `'expired'`; 401 catch branch with buffer preservation (T-125-08 locked); upload queue filter-on-401 removes stuck entry
- `frontend/src/components/FileBrowserTab.tsx`: `'expired'` outcome renders `ACCESS_EXPIRED_MESSAGE`
- TDD gate: Go RED `04eb990` → GREEN `9f468a0`; TS RED `86365c8`/`bb0bbe8` → GREEN `6bc0d89`/`8eec6a5`

### Plan 03 — 3-Observer Write Parity Harness (RMW-01..03)

- `cmd/playwright-fixture/main.go`: `startRemotePeerFixture` extended — write verbs backed by real `files.Sandbox` (`os.MkdirTemp`); separate `newFixtureRemoteWritePeer` helper for version-gate tests
- `internal/daemon/remote_files_write_parity_test.go` (new, `package daemon_test`): Observer A proxy + Observer B RemoteFilesClient + cross-observer A→B and B→A write-then-read; `assertNoCapInError` on all error paths
- `frontend/e2e/files-browser.spec.ts`: scenario 18 (Observer C Playwright `APIRequestContext` write-then-read)
- `frontend/e2e/fixtures/remote-peer-setup.ts`: `remoteFilesWriteURL` builder added

### Plan 04 — Regression Guard + UAT Checklist (RMW-06)

- Ran `go test ./internal/daemon/ -run 'TestRemoteFiles_(List|Stat|Read|CrossSurface_Parity)' -race -count=1`: 7 tests PASS (zero regression)
- This document (`128-VERIFICATION.md`): traceability table + automated results + two-machine write UAT checklist

## Source-Grep Gate Results

| Check | Plan | Expected | Actual | Status |
|-------|------|----------|--------|--------|
| `remotePeerOutdatedMessage` const in `internal/tui/remote_files_client.go` | 01 | 1 | 1 | PASS |
| `REMOTE_PEER_OUTDATED_MESSAGE` export in `frontend/src/lib/filesApi.ts` | 01 | 1 | 1 | PASS |
| Go const and TS const byte-equal (SC3 string) | 01 | match | match | PASS |
| `isMethodNotAllowed` predicate in `filesApi.ts` | 01 | 1 | 1 | PASS |
| `'peer-outdated'` in `WriteOutcome` type (`useFilesWrite.ts`) | 01 | 1 | 1 | PASS |
| `ErrRemoteCapExpired` sentinel in `remote_files_client.go` | 02 | 1 | 1 | PASS |
| `'expired'` in `WriteOutcome` type (`useFilesWrite.ts`) | 02 | 1 | 1 | PASS |
| Upload queue filter on 401 (removes entry, no stuck bar) in `useFilesWrite.ts` | 02 | 1 | 1 | PASS |
| `assertNoCapInError` on every direct-client error in write parity test | 03 | ≥4 | 4 | PASS |
| `package daemon_test` in `remote_files_write_parity_test.go` | 03 | 1 | 1 | PASS |
| `WriteFileAtomic` usage in `cmd/playwright-fixture/main.go` | 03 | ≥1 | 1 | PASS |

**All 11 source-grep gates PASS.**

## Automated Test Results

### Task 1 (Plan 04) — Phase 122 Remote-Read Regression Guard

```
$ go test ./internal/daemon/ -run 'TestRemoteFiles_(List|Stat|Read|CrossSurface_Parity)' -race -count=1 -v
=== RUN   TestRemoteFiles_ListRoundTrip
--- PASS: TestRemoteFiles_ListRoundTrip (0.04s)
=== RUN   TestRemoteFiles_StatRoundTrip
--- PASS: TestRemoteFiles_StatRoundTrip (0.01s)
=== RUN   TestRemoteFiles_ReadRoundTrip
--- PASS: TestRemoteFiles_ReadRoundTrip (0.02s)
=== RUN   TestRemoteFiles_CrossSurface_Parity
=== RUN   TestRemoteFiles_CrossSurface_Parity/list_parity
=== RUN   TestRemoteFiles_CrossSurface_Parity/stat_parity
=== RUN   TestRemoteFiles_CrossSurface_Parity/read_parity
--- PASS: TestRemoteFiles_CrossSurface_Parity (0.02s)
    --- PASS: TestRemoteFiles_CrossSurface_Parity/list_parity (0.01s)
    --- PASS: TestRemoteFiles_CrossSurface_Parity/stat_parity (0.00s)
    --- PASS: TestRemoteFiles_CrossSurface_Parity/read_parity (0.00s)
PASS
ok  	github.com/scottkw/agenthub/internal/daemon	1.228s
```

**Result: 7/7 PASS. Zero regressions. The 405/401 write-verb mappings added in Plans 01/02 do not affect any read behavior.**

### Plan 01 — 405 Version Gate

```
$ go test ./internal/tui/ -run TestRemoteFilesClient -race -count=1
PASS  (TestRemoteFilesClient_405_ErrRemotePeerNoWriteSupport and siblings)

$ pnpm exec vitest run src/lib/__tests__/filesApi.test.ts src/lib/__tests__/useFilesWrite.test.tsx
27/27 PASS
```

### Plan 02 — Cap-Expiry 401

```
$ go test ./internal/tui/ -run TestRemoteFilesClient_401 -race -count=1
PASS  (6 sub-tests: ErrRemoteCapExpired in WriteFile, DeleteFile, RenameFile, MkdirFile, TUI toast, buffer preserved)

$ pnpm exec vitest run src/lib/__tests__/useFilesWrite.test.tsx
37/37 PASS
```

### Plan 03 — 3-Observer Write Parity

```
$ go test ./internal/daemon/ -run TestRemoteFilesWrite_CrossSurface -race -count=1
PASS  (ObserverA_WriteThenRead, ObserverB_WriteThenRead, CrossObserver_AWritesBReads, CrossObserver_BWritesAReads)

$ go test ./internal/daemon/ -race -count=1
ok  github.com/scottkw/agenthub/internal/daemon  11.4s  (full suite, no regression)

Playwright scenario 18 (Observer C): PASS  182ms  [chromium]
```

## Cross-Surface Evidence Summary (RMW-01..05)

Three independent observers confirm byte-identical write-then-read behavior across all Phase 128 write surfaces:

1. **Go daemon-proxy Observer A** (`internal/daemon/remote_files_write_parity_test.go`) — `PUT /api/files/remote/{sid}/write` then `GET /api/files/remote/{sid}/read` via the daemon's same-origin proxy; asserts byte-identical round-trip against the sandbox-backed fixture peer.
2. **Go RemoteFilesClient Observer B** (`internal/daemon/remote_files_write_parity_test.go`) — `tui.RemoteFilesClient.WriteFile` then `ReadFile` against the SAME fixture peer; cross-observer A→B and B→A sub-tests prove both observers reach the same shared persisted state.
3. **Browser HTTPS Observer C** (`frontend/e2e/files-browser.spec.ts scenario 18`) — Playwright `APIRequestContext` PUT then GET against the fixture peer; `ignoreHTTPSErrors: true` mirrors real tailnet self-signed cert usage.

**What the automated tests do NOT cover:** real tailnet latency, actual Wails production build (`-tags wailsassets`), Wails webview rendering, and the physical two-machine setup required to prove the full stack (daemon-proxy + cap-deposit + write persistence + HTTPS + Wails webview) end-to-end. That is the purpose of the two-machine UAT below.

## Manual UAT — Two-Machine Tailnet Write Checklist (OPERATOR-DEFERRED)

**Status:** OPERATOR-DEFERRED. Execution requires two physical machines on the same tailnet (STATE.md TD-3). The automated tests above constitute the strongest machine-verifiable evidence; this checklist covers the physical two-machine path.

**Issue #24 (umbrella):** This UAT closes scottkw/agenthub Issue #24 when executed successfully and all steps are recorded PASS.

**IMPORTANT — before recording a UAT pass:** scan scottkw/agenthub open issues for any bugs citing "Discovered during Phase 128 UAT" or referencing remote write parity. Open issues filed during or after Phase 128 UAT are authoritative and override a casual "pass" assessment.

**Production build note (project memory):** Use `wails build -tags wailsassets` for the Wails desktop build. The dev build omits the embed.FS and produces incorrect MIME types.

---

### Setup

**Machine A (host — web-share provider):**
1. Build and launch AgentHub (`wails build -tags wailsassets` or use the latest release binary).
2. Create a test directory with known content:
   ```
   mkdir /tmp/write-uat-$(date +%Y%m%d)
   cd /tmp/write-uat-$(date +%Y%m%d)
   echo "hello from machine A" > a.txt
   mkdir sub
   echo "nested file" > sub/b.txt
   echo "another file" > sub/c.txt
   ```
3. Create a session running a shell in that directory (or navigate to it).
4. Enable web-share for that session via the Daemon Manager panel.
5. Copy the 5-character join code from the panel.

---

### Machine B — Desktop GUI Write Tests

**Machine B (desktop GUI consumer):**

6. Launch AgentHub desktop (`wails build -tags wailsassets`).
7. Open the Remote Sessions panel. Confirm Machine A's session appears with a "Browse files" button.
8. Click "Browse files". Confirm the join-code modal appears.
9. Paste the join code from step 5. Click Submit.
10. Confirm FileBrowserTab opens showing `a.txt` and `sub/`.

**Edit/save (GUI):**
11. Click `a.txt` to open it in the editor.
12. Edit the content (append " — edited by GUI").
13. Save (Ctrl+S or the Save button).
14. EXPECTED: save succeeds; no error banner. Re-open `a.txt` and confirm the edited content persists.

**Upload (GUI — GUI/web only; TUI does not support upload):**
15. Click the Upload button in the FileBrowserTab toolbar.
16. Select a small local file from Machine B (e.g., a text file under 5 MiB).
17. EXPECTED: upload succeeds; the uploaded file appears in the listing.

**Delete (GUI):**
18. Select the uploaded file from step 16. Click Delete. Confirm the deletion dialog.
19. EXPECTED: file removed from listing.

**Rename (GUI):**
20. Right-click `sub/b.txt` (or use the rename action). Rename it to `sub/b-renamed.txt`.
21. EXPECTED: `b.txt` disappears; `b-renamed.txt` appears in `sub/`.

**Mkdir (GUI):**
22. Click "New folder" (or equivalent). Name it `gui-created-dir`.
23. EXPECTED: `gui-created-dir/` appears in the listing.

**Cross-dir move (GUI):**
24. Move `sub/c.txt` into `gui-created-dir/` (drag or move action).
25. EXPECTED: `sub/c.txt` disappears; `gui-created-dir/c.txt` appears.

---

### Machine B — TUI Write Tests

**Machine B (TUI consumer — same machine or a third tailnet machine):**

26. Launch the TUI: `agenthub tui`.
27. Navigate to Machine A's remote session in the Sessions list.
28. Press `f`. Confirm the join-code prompt modal appears (NOT a "File browser not available" toast — that toast was removed in Phase 122).
29. Type a fresh join code from Machine A (join codes are single-use; generate a new one in Machine A's Daemon Manager if step 9 consumed the previous code). Press Enter.
30. Confirm tabFiles opens showing the current listing of Machine A's directory (should reflect all GUI edits from steps 11-25).

**Edit/save (TUI):**
31. Navigate to `a.txt`. Press Enter to open the editor.
32. Edit the content (append " — edited by TUI").
33. Save (Ctrl+S or the save keybinding).
34. EXPECTED: save succeeds; TUI status bar shows no error. Navigate back and re-open `a.txt`; confirm the TUI-edited content persists.

**Delete (TUI):**
35. Navigate to `sub/b-renamed.txt`. Press the delete keybinding (d or Del, per current binding).
36. Confirm the deletion prompt. Press Enter to confirm.
37. EXPECTED: `sub/b-renamed.txt` removed from listing.

**Rename (TUI):**
38. Navigate into `sub/`. Select any remaining file. Press the rename keybinding (r, per current binding). Enter a new name.
39. EXPECTED: file appears under the new name.

**Mkdir (TUI):**
40. Navigate to the root. Press the mkdir keybinding (m or n, per current binding). Enter `tui-created-dir`.
41. EXPECTED: `tui-created-dir/` appears in the listing.

**Cross-dir move (TUI):**
42. Navigate to a file. Press the move keybinding. Select `tui-created-dir/` as destination.
43. EXPECTED: file removed from source location; appears in `tui-created-dir/`.

---

### Cross-Surface Write Parity Check (RMW-03)

44. After completing both GUI and TUI write sequences above, open a regular browser (NOT Wails) on Machine B. Navigate to `https://{machine-a-fqdn}:{port}/join`. Enter a fresh join code.
45. Confirm the web file browser opens against Machine A and shows the same listing state produced by the GUI and TUI writes.
46. EXPECTED: all three surfaces (GUI, TUI, web) show identical entries, identical file contents, and identical directory structure. This confirms RMW-03 at the live-tailnet layer (the automated 3-observer proof from Plan 03 confirms it at the HTTP/fixture layer; this step confirms it in production).

---

### Failure Mode: Cap-Expiry Mid-Edit (RMW-05)

47. On Machine B desktop GUI, open a file in the editor (e.g., `a.txt`).
48. On Machine A, disable web-share for the session (revokes the cap).
49. On Machine B GUI: attempt to save (Ctrl+S).
50. EXPECTED: save fails with the distinct "access expired" message (NOT the generic "Couldn't save" copy). The edit buffer is preserved — the edited text is still visible in the editor. The progress bar does NOT get stuck.
51. On Machine B TUI (in the session with the now-expired cap): attempt a write operation.
52. EXPECTED: TUI status line shows the "access expired" message.

---

### Failure Mode: v3.4-Peer 405 Version Gate (RMW-04)

> This step requires access to a Machine C running an older (v3.4) AgentHub build that has no write routes.

53. On Machine B, attempt any write operation against the v3.4 Machine C remote session.
54. EXPECTED: the GUI and TUI both display the verbatim message: "The remote session is running an older version of AgentHub that does not support file writes." (NOT a raw 405 or generic error).
55. Read operations (list, stat, read) against the v3.4 Machine C remain unaffected — this is a write-only gate.

> If a v3.4 machine is not available, this step is satisfied at the automated layer by `TestRemoteFilesClient_405_ErrRemotePeerNoWriteSupport` and the Playwright fixture version-gate peer. Mark this step as `automated-only` in the sign-off log.

---

### Operator Sign-Off Log

Record each step as PASS, FAIL, or SKIP (with reason):

| Steps | Description | Result | Notes |
|-------|-------------|--------|-------|
| 6-10 | GUI setup + join-code modal | | |
| 11-14 | GUI edit/save | | |
| 15-17 | GUI upload | | |
| 18-19 | GUI delete | | |
| 20-21 | GUI rename | | |
| 22-23 | GUI mkdir | | |
| 24-25 | GUI cross-dir move | | |
| 26-30 | TUI setup + listing | | |
| 31-34 | TUI edit/save | | |
| 35-37 | TUI delete | | |
| 38-39 | TUI rename | | |
| 40-41 | TUI mkdir | | |
| 42-43 | TUI cross-dir move | | |
| 44-46 | Cross-surface write parity (GUI + TUI + web) | | |
| 47-52 | Cap-expiry failure mode (RMW-05) | | |
| 53-55 | v3.4-peer 405 version gate (RMW-04) | | |

**Issue #24 closure:** When all steps above are recorded PASS (or SKIP with justification for steps 53-55 if no v3.4 machine available), scottkw/agenthub Issue #24 is CLOSED. Reply to this doc with the completed sign-off table and the closing commit hash.

## Decisions Made

- **Regression guard (RMW-06) runs before authoring the UAT checklist** — Task 1 of Plan 04 explicitly confirms zero regressions in the 122 read suite, proving the 405/401 write-verb guards do not affect read code paths. This is a mandatory pre-condition for operator sign-off.
- **Upload step is GUI/web-only** — The TUI does not implement file upload (no binary transfer UI). The upload UAT step (15-17) is scoped to the GUI. This matches the Phase 125 implementation scope.
- **Steps 53-55 (405 version gate) may be automated-only** — A physical v3.4 machine is not always available. The version-gate fixture peer in `cmd/playwright-fixture` provides automated coverage; the operator may mark steps 53-55 as `automated-only` if no v3.4 machine is accessible.
- **Issue #24 umbrella closes on successful execution** — The commit of this checklist does not close Issue #24; only the completed operator sign-off table closes it.

## Deferred Items

| Item | Why deferred | Resolution path |
|------|--------------|-----------------|
| Two-machine tailnet write UAT (all steps above) | Requires two physical tailnet machines; STATE.md TD-3 pending | Operator executes; records sign-off table; closes Issue #24 |
| v3.4-peer 405 step with real v3.4 machine (steps 53-55) | v3.4 machine may not be available | Acceptable as automated-only per decision above |
| `ExchangeJoinCodeAtURL` JSON-vs-303 mismatch shim cleanup (v3.4 TD-5) | Out of v3.5 scope; pre-existing shim from Phase 122 Plan 03 | Future follow-on milestone |
