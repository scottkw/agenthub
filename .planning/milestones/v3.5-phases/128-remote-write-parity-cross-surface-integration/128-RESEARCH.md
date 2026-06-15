# Phase 128: Remote Write Parity + Cross-Surface Integration - Research

**Researched:** 2026-06-14
**Domain:** Go HTTP-proxy integration, cross-surface parity proof (Go + Playwright), tailnet remote-write end-to-end, failure-mode UX (405 version-gate, cap-expiry mid-edit), two-machine UAT
**Confidence:** HIGH (all findings verified against current source in `internal/daemon`, `internal/tui`, `internal/files`, `frontend/src`)

## Summary

Phase 128 is the FINAL v3.5 phase and is **overwhelmingly integration + parity-proof + a committed UAT checklist**, with exactly **two genuinely net-new code gaps**. The remote-write plumbing built in 123-127 is real and verified in source:

- **All 5 remote write proxy verbs exist and forward bodies correctly** (CAP-10, `internal/daemon/remote_files.go` + `api.go`): `PUT write`, `POST upload`, `DELETE delete`, `POST rename`, `POST mkdir`, with `r.Body` forwarded for PUT/POST/PATCH and `Content-Type` + conditional headers (`If-Match` etc.) carried through. Round-trip tested (`TestRemoteFilesWrite_ForwardsBody`, `TestRemoteFilesWrite_CallerCapStripped`, `TestRemoteFilesWrite_GetPassesNilBody`).
- **`tui.RemoteFilesClient` has all 4 write methods** (TUIW-01, no upload by design): `WriteFile/DeleteFile/RenameFile/MkdirFile` over HTTPS with TLS 1.2+ pinned (`tls.Config{MinVersion: tls.VersionTLS12}`) and the CAP-LEAK invariant enforced (error strings interpolate only `(status, body)`, never the URL/cap). Unit-tested via `httptest.TLSServer`.
- **GUI remote write is fully wired for all 6 ops** (RMW-02): `FileBrowserTab` is path-prefix-generic — App.tsx passes `pathPrefix=/api/files/remote/{sessionId}` so every write op (write/upload/delete/rename/move/mkdir) routes through the daemon proxy automatically. Cross-dir move is a `rename` with a different destination. No per-op remote wiring is missing.
- **The Phase 122 read-parity 3-observer harness exists and is the exact template to mirror**: `internal/daemon/remote_files_parity_test.go` (`package daemon_test`, 455 LOC) proves daemon-proxy + `tui.RemoteFilesClient` are byte-identical against one `httptest.NewTLSServer` fixture; Playwright scenarios 16+17 + `startRemotePeerFixture` in `cmd/playwright-fixture/main.go` add the browser-HTTPS third observer. The fixture peer currently serves only READ endpoints with canned content.
- **Buffer preservation on save failure is already locked** (T-125-08): `useFilesWrite.write()` preserves the CM6 buffer on *every* non-success path and shows "Couldn't save the file. Your changes are still here — try again." So **no silent buffer loss exists today** — but the message is generic.
- **Server-side temp-file cleanup is already correct**: `Sandbox.WriteFileAtomic` calls `root.Remove(tmp)` on every failure path. Orphaned partial uploads on the *server* are already handled.

**The two genuine GAPS (net-new code):**

1. **RMW-04 — v3.4-peer 405 → friendly version message.** Neither the daemon proxy nor `RemoteFilesClient` nor `FilesApiError` special-cases a 405 from an old peer. The proxy passes 405 through verbatim; `RemoteFilesClient` returns `"remote files write: 405 <body>"`; `FilesApiError` has predicates for 401/403/404/409/412/413 but **no `isMethodNotAllowed()`** and no "older version of AgentHub" copy anywhere in the codebase (verified: zero matches for "older version" / "does not support file writes"). This needs the 405→friendly-message mapping in **both** Go (`RemoteFilesClient` + TUI surface) and TS (`FilesApiError` predicate + editor/FileBrowserTab dispatch).

2. **RMW-05 — cap-expiry-mid-edit "access expired" message + client upload abort.** The buffer is already preserved on 401, but there is no *distinct* "access expired" message — 401 currently falls into the generic save-error branch. Needs a `isUnauthorized()`-based branch in the save flow surfacing "access expired" (distinct from the generic retry copy). The server already cleans its own temp files; the only client-side concern is aborting an in-flight `XMLHttpRequest` upload (the upload path uses `xhr` with an abort handler — verify it fires on cap-expiry and surfaces cleanly).

**Primary recommendation:** Decompose into 4 plans — (1) the 405 version-gate mapping across Go+TS surfaces, (2) the cap-expiry "access expired" UX + upload-abort cleanup, (3) the 3-observer WRITE parity harness (extend the fixture peer to persist writes; write-then-read byte-equivalence; Playwright HTTPS observer), (4) the regression guard + committed two-machine UAT checklist. Plans 1+2 are net-new code (TDD); plans 3+4 are test/proof/doc. The two-machine UAT *execution* is operator-deferred (auto-mode) — the deliverable is the committed checklist.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Remote write transport (5 verbs) | API/Backend (daemon proxy) | — | DONE — `proxyRemoteFiles` forwards body + headers to remote peer's webserver |
| TUI remote write (4 verbs) | API/Backend (`tui.RemoteFilesClient` → remote webserver, direct HTTPS) | — | DONE — TUI bypasses daemon proxy, talks direct to peer (122-04 asymmetry) |
| GUI remote write (6 ops) | Browser/Client (`FileBrowserTab` + `FilesApiClient`) → daemon proxy | API/Backend (proxy) | DONE — path-prefix-generic; all ops route through `/api/files/remote/{sid}` |
| 405 version-gate detection | API/Backend (Go client) + Browser/Client (TS `FilesApiError`) | — | **GAP** — neither surface maps 405→friendly message |
| Cap-expiry "access expired" UX | Browser/Client (editor save flow) + TUI | — | **GAP** — buffer preserved but no distinct 401 message |
| Orphaned-upload cleanup (server) | API/Backend (`Sandbox.WriteFileAtomic`) | — | DONE — `root.Remove(tmp)` on every failure path |
| Orphaned-upload abort (client) | Browser/Client (`xhr.abort`) | — | Mostly done — verify abort fires + surfaces on cap-expiry |
| 3-observer write parity proof | Test (Go `daemon_test` + Playwright) | — | **NET-NEW TEST** — mirror 122 harness, extend fixture peer to persist |
| Two-machine tailnet UAT | Operator (manual) | Doc (committed checklist) | Checklist is the deliverable; execution operator-deferred |

## Standard Stack

No new dependencies. This phase is integration/test/UX over the existing v3.5 stack.

### Core (already present, verified in source)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http/httptest` | Go 1.26 | Fixture remote peer (`NewTLSServer`) for parity tests | Already the 122 harness backbone |
| Go stdlib `crypto/tls` | Go 1.26 | TLS 1.2+ pinning (`tls.Config.MinVersion`) | Already used in `RemoteFilesClient` + proxy |
| Playwright | 1.59.1 `[VERIFIED: pnpm-lock]` | Browser-HTTPS third observer via `APIRequestContext` | Already the 122 scenario-16/17 mechanism |
| Bubble Tea (`tea.Cmd`) | existing | TUI write dispatch (no sync I/O in Update) | `TestFiles_NoSyncFSCalls` gate enforces |
| CodeMirror 6 | existing (vendored) | Editor buffer that must survive cap-expiry | Buffer-preservation locked at T-125-08 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Extending the existing `httptest.NewTLSServer` fixture peer to persist writes | Spawn a real second `webserver` process | A real process is closer to prod but heavier and flakier in CI; the 122 precedent deliberately used an in-process fixture. **Recommend: extend the fixture to persist into a temp dir** (or back it with a real `files.Sandbox` over `t.TempDir()`) so write-then-read is genuine. |
| In-process fixture peer for the Go parity test | `httptest` + real `files.Handler` mounted on the fixture | Mounting the real `files.Handler` (the same handler the webserver uses) makes the write-then-read round trip exercise the actual sandbox write path, not a mock — strictly higher evidentiary value. **Recommend this** for the write fixture. |

**Installation:** None.

## Package Legitimacy Audit

Not applicable — this phase installs zero external packages. All work is over the existing verified v3.5 stack (Go stdlib, existing Playwright 1.59.1, existing CodeMirror 6 vendored). No `npm install` / `go get` / `pip install` in scope.

## Architecture Patterns

### System Architecture Diagram (write parity, 3 observers)

```
                         ┌─────────────────────────────────────────┐
                         │  Fixture remote peer (httptest.TLSServer)│
                         │  mounts the REAL files.Handler over a    │
                         │  files.Sandbox(t.TempDir())              │
                         │  serves: write/upload/delete/rename/     │
                         │          mkdir + list/stat/read          │
                         └───────────────▲──────────────▲──────────┘
                                         │ HTTPS+cap    │ HTTPS+cap
            ┌────────────────────────────┤              ├───────────────────────┐
            │ Observer A                  │              │ Observer B            │
            │ daemon proxy                │              │ tui.RemoteFilesClient │
   write─►  │ /api/files/remote/{sid}/    │              │ (direct, no proxy)    │
            │ {write,delete,rename,mkdir} │              │ WriteFile/Delete/...  │
            └──────────────┬──────────────┘              └───────────┬───────────┘
                           │ write-then-read              write-then-read
                           ▼                                          ▼
                   ┌───────────────────────────────────────────────────────┐
                   │  ASSERT byte-equivalence of the read-back content      │
                   │  across A and B (and the canonical bytes written)      │
                   └───────────────────────────────────────────────────────┘

   Observer C (Playwright HTTPS): browser APIRequestContext PUTs to the
   fixture peer's /api/files/write, then GETs /api/files/read — asserts the
   read-back bytes equal what it wrote. (Mirrors scenarios 16+17.)

   Version gate (RMW-04): a SECOND fixture peer that has NO write routes
   (404/405 on write verbs) → each observer maps 405 → friendly message.
```

### Component Responsibilities

| Component | File | Responsibility in Phase 128 |
|-----------|------|------------------------------|
| `proxyRemoteFiles` | `internal/daemon/remote_files.go:175` | DONE — already forwards all write verbs; add 405 pass-through is already verbatim. No change unless we choose to map 405 at proxy (NOT recommended — keep mapping client-side). |
| `RemoteFilesClient.WriteFile/...` | `internal/tui/remote_files_client.go:~220+` | **GAP** — add 405 detection → return a typed/sentinel error the TUI surface maps to the friendly message; add 401 detection for "access expired". |
| `FilesApiError` | `frontend/src/lib/filesApi.ts:57` | **GAP** — add `isMethodNotAllowed()` (405) predicate; 401 `isUnauthorized()` already exists. |
| `useFilesWrite.write` | `frontend/src/lib/useFilesWrite.ts:93` | **GAP** — add 405 → version-gate outcome and 401 → "access expired" outcome; buffer already preserved on all non-success paths. |
| `FileBrowserTab` save/dispatch | `frontend/src/components/FileBrowserTab.tsx` | **GAP** — surface the new 405/401 messages distinctly (not the generic retry copy). |
| Fixture remote peer | `cmd/playwright-fixture/main.go` (`startRemotePeerFixture`) | **NET-NEW** — extend to persist writes (back with a real `files.Sandbox` or temp dir) for write-then-read. |
| Go write-parity test | new `internal/daemon/remote_files_write_parity_test.go` (`package daemon_test`) | **NET-NEW** — mirror `TestRemoteFiles_CrossSurface_Parity` with write-then-read. |
| Two-machine UAT checklist | new `128-VERIFICATION.md` (mirror `122-VERIFICATION.md`) | **NET-NEW DOC** — operator-executed; closes Issue #24. |

### Pattern 1: External-test-package parity harness (MANDATORY)
**What:** The parity test MUST live in `package daemon_test`, not `package daemon`.
**Why:** `internal/tui/files.go:17` imports `internal/daemon` → `daemon` cannot import `tui` (cycle). The 122 plan got this wrong and had to recover. The external-test package sidesteps the cycle.
**Reuse the existing exported test helpers** (already added in 122):
```go
// Source: internal/daemon/remote_files_parity_test.go (verified)
daemon.API.Handler() http.Handler                 // wrap with httptest.NewServer
daemon.API.SetRemoteFilesClientForTest(*http.Client) // inject fixture's self-signed cert
daemon.SessionEngine.ConfigDirForTest(string)
tui.NewRemoteFilesClientForTest(baseURL, cap, *http.Client) *RemoteFilesClient
```

### Pattern 2: Client-side 405 mapping (the RMW-04 fix shape)
**What:** Detect `StatusCode == 405` at the client boundary and surface a fixed message; do NOT map at the proxy.
**Why:** The proxy is a dumb byte-pipe (correctly — it forwards verbatim). Both the TUI Go client AND the browser/GUI consume status codes; mapping must happen in each consumer so the message is identical cross-surface (parity contract).
```go
// Go (RemoteFilesClient) — sketch
if resp.StatusCode == http.StatusMethodNotAllowed {
    return files.FileWriteResponse{}, ErrRemotePeerNoWriteSupport // typed sentinel
}
// TUI surface maps ErrRemotePeerNoWriteSupport → the verbatim copy.
```
```ts
// TS (filesApi.ts) — sketch
isMethodNotAllowed(): boolean { return this.status === 405 }
// useFilesWrite: if (err.isMethodNotAllowed()) return 'peer-outdated'
```
**Verbatim message (locked by SC3, must match in BOTH surfaces):**
`"The remote session is running an older version of AgentHub that does not support file writes."`

### Pattern 3: Cap-expiry "access expired" (the RMW-05 fix shape)
**What:** Add a distinct 401 branch in `useFilesWrite.write` (and the TUI write surface) BEFORE the generic error branch.
```ts
// useFilesWrite.ts — insert before the generic setSaveError branch
if (err instanceof FilesApiError && err.isUnauthorized()) {
  setSaveError("Your access to this remote session has expired. Your changes are still here.")
  setSaveState('idle')
  return 'expired'   // new WriteOutcome member
}
```
**Buffer is already preserved** (T-125-08) — do NOT clear `editContent`. The new outcome member `'expired'` extends `WriteOutcome = 'saved' | 'conflict' | 'error'`.

### Anti-Patterns to Avoid
- **Mapping 405/401 at the daemon proxy.** The proxy must stay a verbatim byte-pipe; surface mapping belongs in each consumer (parity). Verified the proxy currently does this correctly.
- **Spawning a real second webserver process for the Go parity test.** The 122 precedent is in-process `httptest`; follow it. (A real second machine is the *manual* UAT, not the automated proof.)
- **Re-deriving the friendly 405 string per surface.** It is a locked verbatim string — extract a single constant per language (Go const + TS const) and grep-gate it, mirroring how `homeDirWriteWarning` was made grep-verifiable.
- **Clearing the editor buffer on any save failure.** Locked anti-pattern (T-125-08).
- **Playwright DOM-flow for remote write.** The fixture serves `/app/` web mode where Wails RPCs don't exist; the 122 decision was to test the UPSTREAM CONTRACT via `APIRequestContext`, not the DOM modal. Mirror that — write-then-read via the browser's HTTP client against the fixture peer.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Remote write transport | New proxy verbs | Existing `proxyRemoteFiles` (CAP-10) | All 5 verbs already forward body+headers; tested |
| TUI remote write | New HTTP client | Existing `RemoteFilesClient` 4 write methods | TLS-pinned, CAP-LEAK-safe, tested |
| GUI remote write per-op wiring | Per-op remote branches | Existing `FilesApiClient` + `pathPrefix` | Path-prefix-generic; all 6 ops already route through proxy |
| Parity test scaffolding | New test infra | Existing `daemon_test` helpers + fixture peer | 4 exported `*ForTest` helpers + `startRemotePeerFixture` already exist |
| Server temp-file cleanup | Cleanup goroutine / sweeper | Existing `Sandbox.WriteFileAtomic` `root.Remove(tmp)` | Already removes temp on every failure path |
| TLS config | New transport | Existing `tls.Config{MinVersion: VersionTLS12}` | Already pinned in both client and proxy |

**Key insight:** ~80% of this phase is already built and tested. The temptation is to "re-prove" remote write by re-implementing transport — don't. The two real code deltas are tiny status-code→message mappings (405, 401). The bulk of the *effort* is the write-parity harness (extend fixture to persist) and the committed UAT doc.

## Runtime State Inventory

This phase adds no renames/migrations. It is integration + two small UX mappings + tests + a doc. Explicitly:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by grep; no datastore keys touched | none |
| Live service config | None — no n8n/Datadog/Tailscale config carries phase strings | none |
| OS-registered state | None — no Task Scheduler / launchd / systemd state | none |
| Secrets/env vars | None new. The cap token is already redacted from errors (CAP-LEAK invariant, verified) | none |
| Build artifacts | None — no `pyproject.toml`/egg-info; Go + vendored CodeMirror unchanged | none |

## Common Pitfalls

### Pitfall 1: Package import cycle in the write-parity test
**What goes wrong:** Placing the test in `package daemon` fails to compile because `tui` imports `daemon`.
**Why it happens:** The 122 plan mis-analyzed the cycle as one-directional and had to recover.
**How to avoid:** Use `package daemon_test` (external test package) — proven in `remote_files_parity_test.go`. Reuse the 4 existing `*ForTest` helpers.
**Warning signs:** `import cycle not allowed` at `go test`.

### Pitfall 2: Fixture peer that doesn't actually persist writes
**What goes wrong:** A canned mock returns 200 on write but read-back returns the OLD canned bytes → the "byte-equivalence" assertion is vacuous (always passes, proves nothing).
**Why it happens:** The current `startRemotePeerFixture` serves a fixed contract (read-only canned files).
**How to avoid:** Back the fixture's write endpoints with a real `files.Sandbox` rooted at `t.TempDir()` (or mount the real `files.Handler`). Then write-then-read genuinely round-trips through the sandbox.
**Warning signs:** Read-back bytes equal the canned `"hello world"` regardless of what was written.

### Pitfall 3: 405 vs the *proxy's own* 405
**What goes wrong:** Conflating "remote peer has no write routes (405 from upstream)" with "daemon proxy rejects an unsupported HTTP method (405 from proxy)". `TestRemoteFiles_405OnUnsupportedMethods` already tests the latter.
**Why it happens:** Both surface as 405.
**How to avoid:** RMW-04 is specifically the UPSTREAM 405 (old peer). The version-gate fixture peer must return 405 *from the upstream* on write verbs (simulating v3.4 which has no write routes). The client maps the upstream-propagated 405 → friendly message. Test with a dedicated "old peer" fixture.
**Warning signs:** The friendly message fires for local proxy method errors, or never fires because the proxy swallows the upstream 405.

### Pitfall 4: Cross-surface message drift
**What goes wrong:** The Go TUI shows one 405 string, the browser shows a slightly different one → parity violation (release-blocking per CLAUDE.md + MEMORY).
**Why it happens:** Two independent string literals.
**How to avoid:** Single Go const + single TS const, both grep-gated to the exact SC3 verbatim text. Mirror the `homeDirWriteWarning` grep-verifiable-const pattern (124-05).
**Warning signs:** A grep for the verbatim string finds it in only one language.

### Pitfall 5: Treating the two-machine UAT as automatable
**What goes wrong:** Attempting to run the real tailnet UAT in CI/agent mode.
**Why it happens:** The phase goal mentions "two-machine UAT."
**How to avoid:** The 122 precedent deferred the 22-step manual UAT to the operator (auto-mode). The Phase 128 deliverable is the COMMITTED CHECKLIST in `128-VERIFICATION.md`; execution is operator-run (requires a second tailnet machine — see STATE.md TD-3 still pending). Mark it deferred-to-user.
**Warning signs:** A plan task says "run the two-machine UAT" instead of "write the two-machine UAT checklist."

## Code Examples

### Existing write-parity assertion shape to mirror (write-then-read)
```go
// Source: internal/daemon/remote_files_parity_test.go (READ parity, verified) — mirror for WRITE:
// 1. Observer A (proxy): PUT /api/files/remote/sid1/write?path=x.txt  (body="content-A")
// 2. read back via proxy GET /api/files/remote/sid1/read?path=x.txt → bytesA
// 3. Observer B (direct RemoteFilesClient): WriteFile(ctx,"sid1","y.txt","content-B"); ReadFile → bytesB
// 4. assert bytesA == []byte("content-A") && bytesB == []byte("content-B")
//    AND a write-by-A is read-back-identical by B against the SAME fixture sandbox
//    (proves both surfaces hit the same persisted state).
directClient := tui.NewRemoteFilesClientForTest(upstream.URL, fixtureCap, upstream.Client())
assertNoCapInError(t, err) // CAP-LEAK invariant on every direct-client error path
```

### Existing FilesApiError predicate pattern (extend with 405)
```ts
// Source: frontend/src/lib/filesApi.ts:86 (verified) — add alongside isConflict():
/** 405 → remote peer has no write routes (old AgentHub version). RMW-04. */
isMethodNotAllowed(): boolean {
  return this.status === 405
}
```

### Existing save-failure branch to extend (401 + 405)
```ts
// Source: frontend/src/lib/useFilesWrite.ts:117 (verified) — insert BEFORE the generic branch:
} catch (err) {
  setIsSaving(false)
  if (err instanceof FilesApiError && err.isConflict()) { /* 412 → existing */ }
  if (err instanceof FilesApiError && err.isMethodNotAllowed()) {
    setSaveError(REMOTE_PEER_OUTDATED_MESSAGE) // verbatim SC3 const
    setSaveState('idle'); return 'peer-outdated'
  }
  if (err instanceof FilesApiError && err.isUnauthorized()) {
    setSaveError(ACCESS_EXPIRED_MESSAGE)
    setSaveState('idle'); return 'expired'
  }
  // generic (buffer still preserved — T-125-08)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Read-only remote browse (v3.4 Phase 122) | Full remote write parity (v3.5) | 123-128 | This phase proves write parity at the same 3-observer bar as read |
| `proxyRemoteFiles` passed `nil` body | Forwards `r.Body` for PUT/POST/PATCH (CAP-10) | Phase 124-03 | Remote write transport works end-to-end |
| `ExchangeJoinCodeAtURL` JSON-decoded a 303 (broken) | Parses `?cap=` from `Location` header (FSW-10/TD-5) | Phase 123 | Desktop GUI can acquire a remote cap — prerequisite for all RMW |

**Deprecated/outdated:** None introduced.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 405 friendly message must be byte-identical across Go-TUI and browser surfaces to satisfy the cross-surface parity contract | Pitfall 4 | If parity allows per-surface phrasing, the dual-const grep gate is over-strict (low risk — parity is the project's release-blocking norm per CLAUDE.md/MEMORY) |
| A2 | Backing the fixture peer's write endpoints with a real `files.Sandbox(t.TempDir())` is acceptable for the automated proof (vs a real second process) | Standard Stack / Pitfall 2 | If the planner wants process-level fidelity, the harness is heavier; but 122 precedent is in-process — low risk |
| A3 | The 401 "access expired" copy wording is at Claude's discretion (CONTEXT.md grants discretion on cap-expiry handling); only the SC3 405 string is verbatim-locked | Pattern 3 | If a specific 401 string is required, copy must be confirmed — low risk, discretion granted |
| A4 | Client-side upload abort already fires via the existing `xhr` abort handler on cap-expiry; only verification (not new code) is needed | Architectural Map | If the xhr abort doesn't propagate cap-expiry cleanly, a small abort-wiring task is needed — MEDIUM; verify during planning |

## Open Questions (RESOLVED)

1. **Does the in-flight upload `xhr` surface cap-expiry as an abortable error that cleans up the upload queue entry?**
   - What we know: `filesApi.ts` upload uses `XMLHttpRequest` with `onabort`/`onerror` handlers rejecting `FilesApiError(0, ...)` / `FilesApiError(status,...)`.
   - What's unclear: whether a mid-upload 401 produces a clean queue-entry removal + "access expired" vs a stuck progress bar.
   - Recommendation: a Plan-2 task verifies (and minimally wires) upload-abort-on-401; server-side temp cleanup is already correct.

2. **Should RMW-04's 405 mapping also cover the *list/read* (read-side) surface, or only write verbs?**
   - What we know: SC3 is specifically about *write* against a v3.4 peer.
   - What's unclear: a v3.4 peer DOES support read, so a 405 on read would be anomalous.
   - Recommendation: scope the 405→friendly-message mapping to write verbs only (write/upload/delete/rename/mkdir). Read 405s remain generic.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All Go tests + build | ✓ (assumed — project builds) | 1.26 | — |
| pnpm + Playwright | Browser-HTTPS observer e2e | ✓ | Playwright 1.59.1 | — |
| `wails build -tags wailsassets` | GUI production verify (if any) | ✓ (per MEMORY) | — | `wails dev` for DevTools UATs |
| Second tailnet machine | Two-machine UAT *execution* | ✗ | — | Operator-deferred; deliverable is the committed checklist (mirrors 122 TD-3, still pending) |

**Missing dependencies with no fallback:** None for the automated scope.
**Missing dependencies with fallback:** Second tailnet machine — the automated 3-observer proof + committed checklist is the in-scope deliverable; physical two-machine run is operator-deferred (Issue #24 closes when executed).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (`package daemon_test`) + Vitest (frontend units) + Playwright 1.59.1 (e2e HTTPS observer) |
| Config file | `frontend/playwright.config.ts`; Go uses default `go test` |
| Quick run command | `go test ./internal/daemon/ -run TestRemoteFiles -race -count=1` |
| Full suite command | `go test ./internal/daemon/ ./internal/tui/ ./internal/webserver/ ./internal/files/ -race && pnpm --filter frontend exec vitest run && pnpm exec playwright test files-browser` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RMW-01 | 3-observer write-then-read byte-equivalence | integration | `go test ./internal/daemon/ -run TestRemoteFilesWrite_CrossSurface -race` | ❌ Wave 0 (new `remote_files_write_parity_test.go`) |
| RMW-02 | GUI remote write 6 ops via proxy | integration/e2e | existing proxy tests + Playwright write-then-read scenario | ⚠️ proxy DONE; e2e write observer ❌ Wave 0 |
| RMW-03 | TUI remote write 4 ops via HTTPS (TLS pinned, cap redacted) | unit | `go test ./internal/tui/ -run TestRemoteFilesClient_Write -race` | ✅ (126-01) |
| RMW-04 | v3.4-peer 405 → friendly version message (both surfaces) | unit | `go test ./internal/tui/ -run ...405...` + `vitest run filesApi` | ❌ Wave 0 (new mapping + tests) |
| RMW-05 | cap-expiry mid-edit → buffer preserved + "access expired" | unit | `vitest run useFilesWrite` (+ TUI) | ❌ Wave 0 (buffer-preserve ✅; distinct message new) |
| RMW-06 | zero regression on 122 read suite + committed UAT checklist | regression + doc | `go test ./internal/daemon/ -run 'TestRemoteFiles_(List\|Stat\|Read\|CrossSurface_Parity)'` | ✅ regression exists; checklist doc ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon/ -run TestRemoteFiles -race -count=1` (+ `vitest run` for TS tasks)
- **Per wave merge:** full Go affected-package suite + `vitest run` + `playwright test files-browser`
- **Phase gate:** full suite green + 122 read suite zero regressions before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/daemon/remote_files_write_parity_test.go` (`package daemon_test`) — covers RMW-01 write-then-read 3-observer
- [ ] Extend `cmd/playwright-fixture/main.go::startRemotePeerFixture` to persist writes (back with real sandbox/temp dir) — required by RMW-01/RMW-02 e2e observer
- [ ] New 405 version-gate fixture peer (write verbs → upstream 405) — required by RMW-04 tests
- [ ] `frontend/src/lib/filesApi.ts` `isMethodNotAllowed()` + verbatim-const + tests — RMW-04
- [ ] `frontend/src/lib/useFilesWrite.ts` 405/401 branches + `WriteOutcome` extension + tests — RMW-04/RMW-05
- [ ] Go `RemoteFilesClient` 405/401 typed-error mapping + TUI surface copy + tests — RMW-04/RMW-05 (TUI parity)
- [ ] `128-VERIFICATION.md` two-machine UAT checklist (mirror `122-VERIFICATION.md`) — RMW-06

## Security Domain

`security_enforcement` is not explicitly false; included. This phase's security posture is largely inherited (the dedicated write-security audit was Phase 127/SEC). Phase 128 must not regress it.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Cap-token over HTTPS; cap-expiry (401) is the RMW-05 trigger |
| V3 Session Management | yes | Remote cap deposited in in-memory `RemoteCapStore`; expiry handling |
| V4 Access Control | yes | `requireFilesWrite` on the remote peer (Phase 124/127); proxy is loopback-trusted |
| V5 Input Validation | yes (inherited) | `validateAndClean` in sandbox; proxy strips caller-supplied `?cap=` |
| V6 Cryptography | yes | TLS 1.2+ pinned (`tls.Config.MinVersion`); never hand-roll — Go stdlib |
| V7 Error Handling/Logging | yes | **CAP-LEAK invariant** — error strings never contain the cap token (verified `redactCapTokenFromError` + `assertNoCapInError`). The new 405/401 messages MUST also be cap-free. |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cap token leaking into a client-visible 405/401 error message | Information Disclosure | New messages are fixed strings (no interpolated URL/cap); assert with the existing `assertNoCapInError` discipline |
| Caller smuggling a different `?cap=` through the proxy | Elevation of Privilege | Proxy already strips caller `cap` and force-sets session (verified `TestRemoteFiles_CallerCapStripped` / `TestRemoteFilesWrite_CallerCapStripped`) |
| MITM downgrade on the tailnet HTTPS hop | Tampering | TLS 1.2+ pinned MinVersion (verified); `InsecureSkipVerify` is intentional for self-signed tailnet peer certs (documented `//nolint:gosec`) — do not change |
| Orphaned temp file after interrupted remote write | DoS (disk) | `WriteFileAtomic` `root.Remove(tmp)` on every failure path (verified) |

## Project Constraints (from CLAUDE.md + MEMORY.md)

- **Go:** `go fmt`, `golangci-lint`, context-aware functions. JS/TS: ESLint + Prettier, TypeScript types.
- **TLS 1.2+ pinned** on remote hops — do not lower `MinVersion`. `InsecureSkipVerify` is intentional for self-signed tailnet certs (keep the `//nolint:gosec` justification).
- **Cap-token redaction** from all error messages (CAP-LEAK invariant) — the new 405/401 messages must be cap-free.
- **Cross-surface parity is RELEASE-BLOCKING** — the 405 and "access expired" behaviors MUST be equivalent across GUI / web-share / TUI; never defer a parity gap without explicit user sign-off (MEMORY).
- **NEVER `kill node.exe`** (Claude Code runs on Node).
- **Wails production build requires `-tags wailsassets`** (MEMORY) if any GUI build/verify is needed.
- **Wails DevTools disabled in production** (MEMORY) — use `wails dev` or web-share to Chrome for any DevTools-dependent UAT.
- **Colorblind user** (MEMORY) — verify any color-based UAT at the hex/source level, not by eye. (Low relevance here — messages are text, not color-coded.)
- **Cross-check open GitHub issues during UAT** (MEMORY) — scan `scottkw/agenthub` open issues before recording UAT pass; this phase closes umbrella **Issue #24**.
- **Don't delete test artifacts early** (MEMORY) — wait for user confirmation before cleanup.
- **Let it crash / no silent fallbacks** (CLAUDE.md) — the 405/401 mappings must be explicit branches, not `or {}` swallows.

## Recommended Plan Decomposition

Four plans. Plans 1+2 are the only net-new *product* code (small, TDD). Plans 3+4 are proof + doc.

- **Plan 128-01 — v3.4-peer 405 version-gate mapping (RMW-04).** Net-new. Add `isMethodNotAllowed()` to `FilesApiError`; add the verbatim SC3 const (Go + TS, grep-gated); branch in `useFilesWrite` + `FileBrowserTab` (browser/GUI) and in `RemoteFilesClient` + TUI write surface (Go) so both surfaces show the identical message on an upstream 405. TDD: a "no-write-routes" fixture peer returning 405. Cross-surface parity gate.
- **Plan 128-02 — cap-expiry mid-edit "access expired" + upload abort (RMW-05).** Net-new (small). Add the 401 → "access expired" branch (buffer already preserved — assert it stays). Extend `WriteOutcome` with `'expired'`. Verify/wire `xhr` upload abort-on-401 cleans the queue entry; assert server temp cleanup is already correct (no new server code expected). TUI parity branch.
- **Plan 128-03 — 3-observer write-parity harness (RMW-01/RMW-02).** Net-new test. Extend `startRemotePeerFixture` to persist writes (real `files.Sandbox`/temp dir). New `internal/daemon/remote_files_write_parity_test.go` (`package daemon_test`) asserting write-then-read byte-equivalence across daemon-proxy + `tui.RemoteFilesClient`. New Playwright write-then-read scenario (HTTPS observer) on the fixture peer. Reuse the 4 existing `*ForTest` helpers.
- **Plan 128-04 — regression guard + committed two-machine UAT checklist + VERIFICATION (RMW-06).** Confirm the 122 read suite passes zero regressions (`TestRemoteFiles_List/Stat/Read/CrossSurface_Parity`). Author `128-VERIFICATION.md` (mirror `122-VERIFICATION.md`): RMW-01..06 traceability, automated results, and the verbatim two-machine UAT checklist (Machine A web-share + Machine B GUI + Machine B TUI; cross-surface write parity + cap-expiry failure mode), operator-deferred. Note Issue #24 closes on execution.

Dependency: 128-03 depends on 128-01+02 (so the parity/e2e suite exercises the new failure-mode messages too); 128-04 depends on all.

## Sources

### Primary (HIGH confidence — current source in this repo)
- `internal/daemon/remote_files.go` (`proxyRemoteFiles`, 5 write handlers, `redactCapTokenFromError`) — verified body+header forwarding, no 405 mapping
- `internal/daemon/api.go:164-175` — 5 remote write proxy routes registered
- `internal/daemon/remote_files_parity_test.go` — the 122 3-observer read-parity harness (template) + 4 exported `*ForTest` helpers
- `internal/daemon/remote_files_test.go` — `TestRemoteFilesWrite_ForwardsBody/CallerCapStripped/GetPassesNilBody`, `TestRemoteFiles_405OnUnsupportedMethods`, read round-trip regression tests
- `internal/tui/remote_files_client.go` — 4 write methods, TLS 1.2+ MinVersion, CAP-LEAK-safe error strings (generic 405 — gap)
- `internal/files/sandbox.go:250` (`WriteFileAtomic`) — temp-file `root.Remove(tmp)` on every failure path
- `internal/files/write.go` — upload `MaxBytesReader` before `ParseMultipartForm`
- `frontend/src/lib/filesApi.ts:57` (`FilesApiError`) — 401/403/404/409/412/413 predicates; NO 405
- `frontend/src/lib/useFilesWrite.ts:93` — `write()` buffer-preserving error flow (T-125-08); `WriteOutcome` type
- `frontend/src/components/FileBrowserTab.tsx` — path-prefix-generic `FilesApiClient`; remote = `/api/files/remote/{sid}`
- `cmd/playwright-fixture/main.go` — `startRemotePeerFixture` (read-only canned contract), `WRITE_CAP`
- `.planning/milestones/v3.4-phases/122-remote-session-file-browse-wiring/122-05-SUMMARY.md` + `122-VERIFICATION.md` — UAT checklist precedent
- Phase 124-03/124-04/124-05 + 126-01 SUMMARYs — CAP-10 proxy, GUI write opt-in, TUIW-01 client

### Secondary (MEDIUM)
- `.planning/REQUIREMENTS.md` (RMW-01..06), `.planning/STATE.md`, `128-CONTEXT.md` — locked SCs and discretion areas

## Metadata

**Confidence breakdown:**
- Already-covered plumbing (proxy/client/GUI/TLS/cleanup): HIGH — read directly from current source
- Gaps (405 mapping, 401 "access expired" message): HIGH — verified absent (zero grep matches for the verbatim strings / `isMethodNotAllowed`)
- Harness approach (extend fixture to persist): HIGH — mirrors proven 122 pattern; fixture-persistence is the one design choice (A2)
- Upload-abort-on-401 client behavior: MEDIUM — handlers exist; exact cap-expiry behavior needs a verification task (Open Q1)

**Research date:** 2026-06-14
**Valid until:** 2026-07-14 (stable — internal integration phase, no fast-moving external deps)
