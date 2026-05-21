---
phase: 122
verified: 2026-05-20T00:00:00Z
status: human_needed
score: 5/5 requirements verified at code/test level
requirements_covered: 5/5 (REMOTE-01..05)
overrides_applied: 0
human_verification:
  - test: "Two-machine UAT — paste join code on Machine B desktop GUI for Machine A's web-shared session; browse files; sub-dir nav; preview"
    expected: "Browse files button → modal → submit → FileBrowserTab opens with the host's seeded files; sub-dir nav; text/binary preview all match local-session UX"
    why_human: "Real tailnet round-trip cannot be simulated by the Go cross-surface parity test or Playwright fixture; only an actual two-machine setup proves the tailnet HTTPS + cap-deposit + same-origin proxy + Wails webview combination works end-to-end (Task 3 of PLAN.md)"
  - test: "Two-machine TUI UAT — press 'f' on Machine A's remote session from Machine B TUI; type join code; navigate + preview"
    expected: "joinCodePromptModel appears (NOT v3.4 toast); typing code + Enter opens tabFiles against the remote; up/down nav + filter all work; 401 on stale cap shows the verbatim 'Remote session must be web-shared…' status line"
    why_human: "TUI side of REMOTE-04 — keyboard nav + glamour markdown + filter all need a real terminal at real tailnet latency; the unit tests cover behavior but cannot perceive the latency feel"
  - test: "Cross-surface parity check — open the SAME remote session from desktop GUI, TUI, AND a regular browser (web-share viewer); confirm identical file list + preview behavior"
    expected: "All three surfaces show identical entries; identical sort order; identical preview for text/markdown/image/binary; identical 5 MiB cap refusal; identical cwd-protection at root"
    why_human: "REMOTE-05 is the codified version of project memory 'cross-surface parity is release-blocking'. Automated tests prove byte-equivalence at the HTTP/JSON layer; human verification confirms the layered UX behavior matches at the rendered-surface layer"
  - test: "Failure-mode UAT — host disables web-share AFTER consumer has cap cached; consumer presses 'f' / clicks Browse files"
    expected: "401 from remote → desktop GUI renders EnableWebSharingTakeover with verbatim copy 'Remote session must be web-shared to browse files. Ask the owner to enable sharing.' + 'Re-enter join code' button; TUI status line shows the same verbatim copy"
    why_human: "401-takeover render is unit-tested but the timing of the failure (after cap caching) requires a live host toggle"
---

# Phase 122: Remote-Session File Browse Wiring Verification Report

**Phase Goal:** Both the desktop GUI's React FileBrowserTab AND the TUI's Files view work transparently against remote tailnet sessions. Cap acquired via paste-join-code modal (GUI) or join-code prompt modal (TUI). GUI fetches through the daemon's `/api/files/remote/{sid}/...` proxy (same-origin, no CORS); TUI fetches directly via HTTPS+cap. Cross-surface parity: GUI, TUI, and web (Phase 120-06) all deliver identical observable behavior.

**Verified:** 2026-05-20
**Status:** human_needed (automated layer green; 4 human-UAT items deferred per auto-mode)
**Re-verification:** No — initial verification after Phase 122 closure
**Worktree branch:** worktree-agent-a4f0e3f6167bb8d04

## Requirements Traceability

| Req | Plans | Tests | Status |
| --- | ----- | ----- | ------ |
| **REMOTE-01** Desktop GUI's FileBrowserTab opens against a remote tailnet session via daemon proxy + cap deposit | 01-recovery (daemon proxy), 03 (GUI modal + tab branch) | `internal/daemon/remote_files_test.go::TestRemoteFiles_ListRoundTrip` + `…StatRoundTrip` + `…ReadRoundTrip` + `…HeadOnRead` + `…CallerCapStripped` (13 proxy tests); `frontend/src/components/__tests__/App.remoteFileBrowser.test.tsx` (React-DOM modal flow); `TestRemoteFiles_CrossSurface_Parity` (Plan 05 cross-surface byte-equivalence) | VERIFIED (automated); human UAT item 1 |
| **REMOTE-02** GUI shows EnableWebSharingTakeover when remote session is not web-shared | 03 (EnableWebSharingTakeover) | `frontend/src/components/FileBrowser/__tests__/EnableWebSharingTakeover.test.tsx`; `FileBrowserTab.remoteAuth.test.tsx` (401 → takeover); Playwright `scenario 17` (upstream 401 contract — trigger condition) | VERIFIED (automated); human UAT item 4 |
| **REMOTE-03** TUI Files view opens against remote tailnet sessions via HTTPS+cap; v3.4 toast removed | 04 (RemoteFilesClient + joinCodePromptModel + FilesOpen branch) | `internal/tui/remote_files_client_test.go` (13 RemoteFilesClient tests); `internal/tui/joincode_prompt_test.go` (prompt sub-model); `internal/tui/update_remote_test.go` (FilesOpen branch); v3.4 toast grep: 0 occurrences in `internal/tui/update.go` | VERIFIED (automated); human UAT item 2 |
| **REMOTE-04** TUI behavior identical local vs remote (nav + preview + filter + status line + glamour + binary/over-cap refusals) | 04 (FilesClient interface — same Update loop, two clients) | `internal/tui/files_test.go` (10/10 TUI-NN matrix unchanged); `internal/tui/files_integration_test.go` (local DaemonClient end-to-end); `TestFiles_NoSyncFSCalls` (no-sync-FS gate still passes); RemoteFilesClient implements identical FilesClient interface (4 methods, same signatures) | VERIFIED (automated); human UAT item 2 |
| **REMOTE-05** Cross-surface parity (GUI + TUI + web) — identical observable behavior | 05 (cross-surface evidence) | **`internal/daemon/remote_files_parity_test.go::TestRemoteFiles_CrossSurface_Parity`** — daemon proxy + tui.RemoteFilesClient observe byte-identical list/stat/read responses against a single fixture upstream; **`…CapAbsenceAsymmetry`** — proves proxy gate is local; **`…401Propagation`** — both surfaces observe 401; **`…CapInjectionPrevented`** — caller-cap stripping enforced; **`frontend/e2e/files-browser.spec.ts scenarios 16 + 17`** — browser-level HTTPS observer agrees on the same canonical contract | VERIFIED (automated cross-surface evidence); human UAT item 3 |

**Score:** 5/5 requirements verified at the automated layer. Human UAT items 1-4 are deferred to user per auto-mode (Task 3 of PLAN.md is a 22-step two-machine setup that cannot be executed by an agent).

## What Was Built

### Plan 01-recovery — Daemon-side Remote-Files Proxy

- `internal/daemon/remote_caps.go` — `RemoteCapStore` (thread-safe in-memory map[sessionID]{baseURL, capToken}; no disk persistence)
- `internal/daemon/remote_files.go` — `POST /api/remote-files/caps` deposit handler + `GET/HEAD /api/files/remote/{sid}/{list,stat,read}` proxy routes
- `internal/daemon/api.go` — wired `remoteCaps` field on API + 5 new routes registered
- 21 new tests (8 RemoteCapStore + 13 proxy/POST round-trip)

### Plan 02 — TUI FilesClient interface (folded into Plan 04 due to dependency-not-on-main)

- `internal/tui/files_client.go` — `FilesClient` interface (ListFiles/StatFile/ReadFile/HeadFile)
- `*daemon.DaemonClient` satisfies it via duck typing; `*RemoteFilesClient` satisfies it explicitly

### Plan 03 — Desktop GUI Remote-Session File-Browse Wiring

- `frontend/src/components/RemoteJoinCodeModal.tsx` — a11y-correct paste-join-code modal (cap never in React state; T-122-03-01 mitigation)
- `frontend/src/components/FileBrowser/EnableWebSharingTakeover.tsx` — 401-on-remote takeover with verbatim copy
- `frontend/src/lib/remoteSession.ts` — pure helpers (`findRemoteSession`, `remoteBaseURLFor`)
- `frontend/src/components/RemoteSessionsPanel.tsx` — "Browse files" button per remote session
- `frontend/src/App.tsx` — tab gate branches on `findRemoteSession`; modal mount; cap-cache state
- Wails bindings: `App.ExchangeJoinCodeAtURL` + `App.RegisterRemoteCap` (hand-regenerated since project has no `wails generate module`)
- ~50 new tests across React/TS/Go

### Plan 04 — TUI RemoteFilesClient + joinCodePromptModel

- `internal/tui/remote_files_client.go` — HTTPS+cap mirror of DaemonClient files methods; TLS 1.2+ pinned; cap-token never appears in error strings (T-122-04-01)
- `internal/tui/joincode_prompt.go` — Bubble Tea sub-model with `idle → submitting → error` state machine; `CheckRedirect=ErrUseLastResponse` to parse 303 Location header
- `internal/tui/model.go` — `modalJoinCodePrompt` + `remoteCapStore` map[sessionID]{cap}
- `internal/tui/update.go` — FilesOpen branches on local vs remote; cap-cached → fast path, cap-absent → modal
- `internal/tui/files.go` — `applyRemote401IfNeeded` shared helper (delete cap + verbatim status copy)
- v3.4 "File browser not available" toast removed (0 occurrences in update.go)

### Plan 05 — Cross-Surface Parity Evidence

- **`internal/daemon/remote_files_parity_test.go`** (455 LOC, `package daemon_test`) — Four cross-surface tests:
  - `TestRemoteFiles_CrossSurface_Parity` (3 sub-tests: list/stat/read) — daemon proxy + tui.RemoteFilesClient observe byte-identical responses against a single `httptest.NewTLSServer` fixture
  - `TestRemoteFiles_CrossSurface_CapAbsenceAsymmetry` — proxy 404 without deposit; direct RemoteFilesClient succeeds (architecture asymmetry from Plan 04)
  - `TestRemoteFiles_CrossSurface_401Propagation` — both surfaces observe 401; CAP-LEAK assertion on direct error string
  - `TestRemoteFiles_CrossSurface_CapInjectionPrevented` — smuggled `?cap=` stripped by proxy
- **Exported test helpers** (additive surface for external test packages):
  - `daemon.API.Handler() http.Handler`
  - `daemon.API.SetRemoteFilesClientForTest(c *http.Client)`
  - `daemon.SessionEngine.ConfigDirForTest(dir string)`
  - `tui.NewRemoteFilesClientForTest(baseURL, cap, *http.Client) *RemoteFilesClient`
- **`cmd/playwright-fixture/main.go`** — `startRemotePeerFixture()` adds a second TLS listener mocking the remote-peer contract; canned shape matches the Go parity test byte-for-byte
- **`frontend/e2e/fixtures/remote-peer-setup.ts`** (new) — helper module exposing `remotePeerURL()`, `remoteFilesListURL()`, etc.
- **`frontend/e2e/files-browser.spec.ts`** — Scenarios 16 + 17 (`remote-session …`) verifying the upstream contract from a third observer (browser HTTPS)
- **`frontend/e2e/global-setup.ts` + `fixture-env.ts`** — parse `REMOTE_PEER_URL=` line from fixture stdout

## Source-Grep Gate Results

| Check | Plan | Expected | Actual | Status |
| ----- | ---- | -------- | ------ | ------ |
| `tls.VersionTLS12` in `internal/daemon/remote_files.go` | 01 | ≥1 | 1 | PASS |
| `redactCapTokenFromError` calls in `internal/daemon/remote_files.go` | 01 | ≥3 | 4 | PASS |
| `/api/files/remote/` in `internal/daemon/api.go` (routes + comments) | 01 | ≥4 | 6 | PASS |
| `/api/remote-files/caps` in `internal/daemon/api.go` (route + comment) | 01 | ≥1 | 2 | PASS |
| Disk-write APIs in `internal/daemon/remote_caps.go` | 01 | 0 | 0 | PASS |
| `FilesClient` interface uses in `internal/tui/files_client.go` | 02 | ≥1 | 4 | PASS |
| `capToken` in `frontend/src/components/RemoteJoinCodeModal.tsx` (cap-token-not-in-state invariant) | 03 | 0 | 0 | PASS |
| `dangerouslySetInnerHTML` in `RemoteJoinCodeModal.tsx` | 03 | 0 | 0 | PASS |
| `pathPrefix` in `frontend/src/App.tsx` (remote-branch tab gate) | 03 | ≥1 | 1 | PASS |
| `v3.5 follow-on` in `frontend/src/App.tsx` (descope-marker removed) | 03 | 0 | 0 | PASS |
| `tls.VersionTLS12` in `internal/tui/remote_files_client.go` | 04 | ≥1 | 1 | PASS |
| `modalJoinCodePrompt` in `internal/tui/model.go` (constant + comment) | 04 | ≥1 | 2 | PASS |
| `File browser not available` in `internal/tui/update.go` (v3.4 toast removed) | 04 | 0 | 0 | PASS |
| `TestRemoteFiles_CrossSurface_Parity` in parity test (function + comment) | 05 | ≥1 | 2 | PASS |
| `scenario 16|17` in `frontend/e2e/files-browser.spec.ts` | 05 | 2 | 2 | PASS |

**All 15 source-grep gates PASS.**

## Automated Test Results

### Plan 05 Per-Task

```
$ go test ./internal/daemon/ -run TestRemoteFiles_CrossSurface -race -count=1 -v
=== RUN   TestRemoteFiles_CrossSurface_Parity
=== RUN   TestRemoteFiles_CrossSurface_Parity/list_parity
=== RUN   TestRemoteFiles_CrossSurface_Parity/stat_parity
=== RUN   TestRemoteFiles_CrossSurface_Parity/read_parity
--- PASS: TestRemoteFiles_CrossSurface_Parity (0.03s)
    --- PASS: TestRemoteFiles_CrossSurface_Parity/list_parity (0.01s)
    --- PASS: TestRemoteFiles_CrossSurface_Parity/stat_parity (0.00s)
    --- PASS: TestRemoteFiles_CrossSurface_Parity/read_parity (0.00s)
=== RUN   TestRemoteFiles_CrossSurface_CapAbsenceAsymmetry
--- PASS: TestRemoteFiles_CrossSurface_CapAbsenceAsymmetry (0.01s)
=== RUN   TestRemoteFiles_CrossSurface_401Propagation
--- PASS: TestRemoteFiles_CrossSurface_401Propagation (0.01s)
=== RUN   TestRemoteFiles_CrossSurface_CapInjectionPrevented
--- PASS: TestRemoteFiles_CrossSurface_CapInjectionPrevented (0.01s)
PASS
ok  	github.com/scottkw/agenthub/internal/daemon	1.145s
```

```
$ pnpm exec playwright test --grep "remote-session" --reporter=list
  ✓ [chromium] scenario 16: remote-session browse via owner cap — upstream contract (43ms)
  ✓ [chromium] scenario 17: remote-session 401 from upstream — no cap rejected (5ms)
  ✓ [firefox]  scenario 16: remote-session browse via owner cap — upstream contract (37ms)
  ✓ [firefox]  scenario 17: remote-session 401 from upstream — no cap rejected (5ms)
  ✓ [webkit]   scenario 16: remote-session browse via owner cap — upstream contract (38ms)
  ✓ [webkit]   scenario 17: remote-session 401 from upstream — no cap rejected (7ms)
  6 passed (3.1s)
```

### Full Files-Browser Suite (Regression)

```
$ pnpm exec playwright test files-browser --reporter=list
  51 passed (7.8s)
```

(45 pre-existing scenarios × 3 browsers + 6 new scenarios 16+17 × 3 browsers = 51; no regression.)

### Affected-Package Go Regression

```
$ go test ./internal/daemon/ ./internal/tui/ -race -count=1
ok  	github.com/scottkw/agenthub/internal/daemon	9.868s
ok  	github.com/scottkw/agenthub/internal/tui	1.428s
```

### TypeScript

```
$ pnpm --filter frontend exec tsc --noEmit
(no output — clean)
```

## Manual UAT — Two-Machine Setup (Deferred to User)

Per PLAN.md Task 3, this is a 22-step two-machine UAT that automated tests cannot execute. Auto-mode deferred this to the user. Verbatim instructions reproduced here for the operator's reference:

> **Machine A (host):**
> 1. Launch agenthub. Create a session running a shell or AI CLI in a directory with a few visible files (use `/tmp/parity-uat-{date}` with `echo hi > a.txt; mkdir sub; echo nested > sub/b.txt`).
> 2. Enable web-share for that session via the Daemon Manager panel.
> 3. Copy the 5-character join code from the panel.
>
> **Machine B (desktop GUI consumer):**
> 4. Launch agenthub (Wails desktop build with `-tags wailsassets` per project memory).
> 5. Open the Remote Sessions panel. Confirm Machine A's session appears with a "Browse files" button.
> 6. Click "Browse files". Modal appears.
> 7. Paste the join code from step 3. Click Submit.
> 8. Confirm FileBrowserTab opens showing `a.txt` and `sub/`.
> 9. Click `sub/`. Confirm `b.txt` appears.
> 10. Click `b.txt`. Confirm preview pane shows "nested".
>
> **Machine B (TUI consumer — same machine OR a third tailnet machine):**
> 11. Launch the TUI: `agenthub tui`.
> 12. Navigate to Machine A's remote session in the Sessions list.
> 13. Press `f`. Confirm the join-code prompt modal appears (NOT the "File browser not available" toast from v3.4).
> 14. Type the join code from step 3 (or generate a fresh one on Machine A — codes are single-use; if step 7 already consumed the code, generate a new code from Machine A's Daemon Manager first).
> 15. Press Enter. Confirm tabFiles opens showing the same `a.txt` and `sub/` from Machine A.
> 16. Press Down to highlight `a.txt`. Confirm preview pane shows "hi".
> 17. Press `/` and type `sub`. Confirm filter shows only `sub/`.
> 18. Press Esc. Press Backspace at the root. Confirm it's a no-op (cwd protection from Phase 121 TUI-03 still works for remote).
>
> **Cross-surface parity check (REMOTE-05):**
> 19. Open a regular browser (NOT Wails) on Machine B. Navigate to `https://{machine-a-fqdn}:{port}/join`. Enter the join code (or use a fresh one). Confirm the web file browser opens against Machine A and shows the same `a.txt` + `sub/`. (This is the Phase 120-06 web-share path — should be unaffected by Phase 122, but proving it ALL works confirms REMOTE-05.)
>
> **Failure-mode UAT:**
> 20. On Machine A, disable web-share for the session.
> 21. On Machine B desktop GUI: click "Browse files" on the (now stale) session. Confirm `EnableWebSharingTakeover` renders with the verbatim copy "Remote session must be web-shared to browse files. Ask the owner to enable sharing." plus a "Re-enter join code" button.
> 22. On Machine B TUI: press `f` on the (stale) session. Confirm the status line shows the same verbatim copy (per Plan 04's 401-handling).

**Operator sign-off log:** _(awaiting user execution; reply with PASS/FAIL per step number)_

## Cross-Surface Evidence Summary (REMOTE-05)

The merge-gate evidence for REMOTE-05 (release-blocking per project memory) comes from THREE independent observers all agreeing on the same canonical contract:

1. **Go cross-surface parity** (`internal/daemon/remote_files_parity_test.go`) — the daemon proxy (desktop GUI's transport) and `tui.RemoteFilesClient` (TUI's transport) both fetch from a shared `httptest.NewTLSServer` fixture and observe **byte-identical** list/stat/read response bodies. Asserted via `bytes.Equal` over the raw HTTP bodies.
2. **Browser HTTPS observer** (`frontend/e2e/files-browser.spec.ts scenarios 16+17`) — Playwright's `APIRequestContext` (browser-spec HTTPS stack, no Node fetch) drives the same fixture mock-remote-peer (via the playwright fixture binary's new `startRemotePeerFixture`) and observes the **same canonical shape** (`entries=[a.txt, sub/]`, `truncated=false`, `size=100`, `isDir=false/true`).
3. **Existing local-file e2e** (scenarios 1-14 in `files-browser.spec.ts`) — proves the web-share path (Phase 120-06) is unchanged by Phase 122. The web-share viewer at `/app/?session=…&cap=…` continues to fetch identical entries from the local `/api/files/list` route.

The three observers exercise three independent network stacks (`net/http` from daemon, `net/http` from tui, browser HTTPS from playwright) and all agree on the contract. This is the strongest cross-surface parity evidence the project can produce without the 22-step manual two-machine UAT (which is deferred to the user per auto-mode).

## Decisions Made

- **Parity test in `package daemon_test`, not `package daemon`** — the PLAN.md Task 1 `<action>` block proposed placing it in `package daemon`, justifying it on "daemon doesn't import tui". However, `internal/tui/files.go:17` already imports `internal/daemon`, so daemon-importing-tui would be a cycle. External test package (`package daemon_test`) sidesteps this: tests-in-test-package can import any non-test package without inducing a cycle.
- **Four new exported test helpers** (`API.Handler()`, `API.SetRemoteFilesClientForTest`, `SessionEngine.ConfigDirForTest`, `tui.NewRemoteFilesClientForTest`) — required to drive the parity test from outside the daemon package. All four have `ForTest` suffix + doc comments forbidding production use. Mirrors the existing test-only-export pattern in the project.
- **Playwright scenarios 16+17 verify the UPSTREAM CONTRACT, not the React-DOM modal flow** — the React-DOM flow (RemoteJoinCodeModal click → modal submit → FileBrowserTab mount) is covered by `frontend/src/components/__tests__/App.remoteFileBrowser.test.tsx` (Phase 122-03 Task 3); duplicating that at the Playwright layer would require either (a) extensive `window.go` mock injection to spoof Wails RPCs or (b) a special "test-mode" override of `detectMode()` to force-render the desktop tab gate. Both options dilute the test's value relative to the upstream-contract evidence the cross-surface parity test ALREADY provides at the network layer.
- **Mock remote peer reuses fixture's self-signed TLS config** — both listeners are scoped to 127.0.0.1 and the e2e specs use `ignoreHTTPSErrors: true` so the cert is moot. Avoids generating a second cert pair.

## Deferred Items

| Item | Why deferred | Phase to address |
| ---- | ------------ | ---------------- |
| 22-step two-machine manual UAT | Cannot be executed by an agent; requires two actual tailnet machines | User executes; results recorded in this doc's operator sign-off log |
| Removal of `internal/daemon/client_remote_files.go::ExchangeJoinCodeAtURL` JSON-vs-303 mismatch | Out of Plan 05 scope; this helper was added in Plan 03 as a forward-compatible shim before Plan 01 landed | Future Plan 122-06 or follow-on milestone if not closed by user UAT |

## Self-Check: PASSED

Files claimed in this report exist:
- `internal/daemon/remote_files_parity_test.go` — FOUND
- `internal/daemon/remote_caps.go` — FOUND
- `internal/daemon/remote_files.go` — FOUND
- `internal/tui/remote_files_client.go` — FOUND (with new `NewRemoteFilesClientForTest` export)
- `internal/tui/files_client.go` — FOUND
- `internal/tui/joincode_prompt.go` — FOUND
- `frontend/src/components/RemoteJoinCodeModal.tsx` — FOUND
- `frontend/src/components/FileBrowser/EnableWebSharingTakeover.tsx` — FOUND
- `frontend/e2e/files-browser.spec.ts` — FOUND (scenarios 16+17)
- `frontend/e2e/fixtures/remote-peer-setup.ts` — FOUND
- `cmd/playwright-fixture/main.go` — MODIFIED (startRemotePeerFixture added)

Commits exist on this worktree branch:
- `0b45c36` (Plan 05 Task 1 RED — parity test) — FOUND
- `9606d2c` (Plan 05 Task 1 GREEN — exported test helpers) — FOUND
- `ae0c30b` (Plan 05 Task 2 — Playwright scenarios 16+17 + fixture extension) — FOUND

Sign-off SHA: _(set on merge commit creation)_
