# Phase 128: Remote Write Parity + Cross-Surface Integration - Pattern Map

**Mapped:** 2026-06-14
**Files analyzed:** 7 (2 net-new product-code deltas across Go+TS; 1 fixture extension; 1 new Go parity test; 1 new Playwright scenario; 1 new VERIFICATION doc)
**Analogs found:** 7 / 7 (every delta has a concrete in-repo precedent — this is an integration/parity phase)

## Net-New CODE Deltas (the only two product-code changes)

Both touch **Go AND TS** for cross-surface parity (RELEASE-BLOCKING per CLAUDE.md/MEMORY):

| # | Req | Delta | Go surface | TS surface |
|---|-----|-------|-----------|-----------|
| 1 | RMW-04 | v3.4-peer upstream-405 → verbatim "older version" message | `internal/tui/remote_files_client.go` (typed sentinel on 405) + TUI write surface copy | `frontend/src/lib/filesApi.ts` `isMethodNotAllowed()` + `useFilesWrite.ts` branch + `FileBrowserTab.tsx` dispatch |
| 2 | RMW-05 | cap-expiry 401 → "access expired" (distinct from generic) + upload abort | `RemoteFilesClient` 401 detection + TUI surface copy | `useFilesWrite.ts` `isUnauthorized()` branch + `WriteOutcome` `'expired'` + verify `xhr` abort cleanup in `filesApi.ts` |

Everything else (proxy, GUI wiring, TUI write methods, TLS, server temp cleanup) is DONE — see RESEARCH "Don't Hand-Roll." Do not re-implement transport.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/lib/filesApi.ts` (modify) | utility (typed client + error) | request-response | self — existing `is*()` predicate block (`isConflict`/`isCollision`/`isOverCap`) | exact (same file, mirror sibling) |
| `frontend/src/lib/useFilesWrite.ts` (modify) | hook | request-response | self — existing `isConflict()` catch branch | exact (same file, sibling branch) |
| `frontend/src/components/FileBrowserTab.tsx` (modify) | component | request-response | self — existing conflict/save-error dispatch | exact |
| `internal/tui/remote_files_client.go` (modify) | service (HTTP client) | request-response | self — existing `WriteFile`/`DeleteFile` 5 `StatusCode != OK` blocks | exact (sibling status branch) |
| `cmd/playwright-fixture/main.go` (modify) | test fixture | file-I/O (extend to persist) | self — existing `startRemotePeerFixture` read-only mux | role-match (read→write extension) |
| `internal/daemon/remote_files_write_parity_test.go` (new) | test | request-response (write-then-read) | `internal/daemon/remote_files_parity_test.go` | exact (mirror harness) |
| `frontend/e2e/files-browser.spec.ts` (new scenario) | test (e2e) | request-response | scenarios 16+17 (same file) + `files-write.spec.ts` WRITE_CAP usage | exact |
| `.planning/phases/128-.../128-VERIFICATION.md` (new) | doc | — | `.planning/milestones/v3.4-phases/122-remote-session-file-browse-wiring/122-VERIFICATION.md` | exact (mirror) |

## Pattern Assignments

### `frontend/src/lib/filesApi.ts` (RMW-04 — add `isMethodNotAllowed()`)

**Analog:** the existing predicate block in the SAME file.

**`FilesApiError` predicate pattern** (`filesApi.ts:57-101`) — add the new predicate alongside these, mirroring `isConflict`/`isOverCap` exactly (single status compare, JSDoc with req tag):
```typescript
export class FilesApiError extends Error {
  constructor(public readonly status: number, public readonly bodyText: string) { ... }

  isUnauthorized(): boolean { return this.status === 401 }   // line 69 — REUSE for RMW-05 (already exists)

  /** 413 → file > 5 MiB cap (Phase 118 FS-08). */
  isOverCap(): boolean { return this.status === 413 }        // line 82-84

  /** 412 → If-Match precondition failed; another process changed the file (EDIT-08). */
  isConflict(): boolean { return this.status === 412 }       // line 87-89

  /** 409 → name collision on write/rename/mkdir/upload (EDIT-09/10). */
  isCollision(): boolean { return this.status === 409 }      // line 92-94
}
```
ADD (mirror shape; verbatim const lives here too — grep-gated):
```typescript
/** Verbatim SC3 string — MUST byte-match the Go const (parity contract). */
export const REMOTE_PEER_OUTDATED_MESSAGE =
  'The remote session is running an older version of AgentHub that does not support file writes.'

/** 405 → remote peer has no write routes (old AgentHub version). RMW-04. */
isMethodNotAllowed(): boolean { return this.status === 405 }
```
Scope to WRITE verbs only (Open Q2 resolved: read 405s stay generic).

**Upload-abort handlers to verify (RMW-05)** (`filesApi.ts:336-363`) — the `xhr` already rejects with typed `FilesApiError`; confirm a mid-upload 401 surfaces cleanly:
```typescript
const xhr = new XMLHttpRequest()                                    // line 336
xhr.addEventListener('load', () => {
  if (xhr.status >= 200 && xhr.status < 300) { ... }
  else { reject(new FilesApiError(xhr.status, xhr.responseText ?? '')) }  // 401 lands here → line 350
})
xhr.addEventListener('error', () => { reject(new FilesApiError(0, 'network error')) })  // 354-355
xhr.addEventListener('abort', () => { reject(new FilesApiError(0, 'upload aborted')) })  // 358-359
```
A 401 during upload rejects via the `load` branch (line 350) as `FilesApiError(401,...)` — so the `useFilesWrite`/caller 401 branch handles queue cleanup. The `abort` handler exists; verify it fires + removes the queue entry (Open Q1 — MEDIUM, may need minimal wiring).

---

### `frontend/src/lib/useFilesWrite.ts` (RMW-04 + RMW-05 — new catch branches + `WriteOutcome` extension)

**Analog:** the existing `write()` catch block in the SAME file.

**`WriteOutcome` type** (`useFilesWrite.ts:35`) — extend (callers branch synchronously, WR-02):
```typescript
export type WriteOutcome = 'saved' | 'conflict' | 'error'
// → 'saved' | 'conflict' | 'error' | 'peer-outdated' | 'expired'
```

**Catch-branch pattern to mirror** (`useFilesWrite.ts:117-132`) — insert the two new branches BEFORE the generic branch; do NOT clear the buffer (T-125-08 locked at line 121):
```typescript
} catch (err) {
  setIsSaving(false)

  if (err instanceof FilesApiError && err.isConflict()) {   // line 120 — the sibling template
    setIsConflict(true)
    setSaveState('idle')
    return 'conflict'
  }

  // INSERT (RMW-04): upstream 405 = peer too old
  // if (err instanceof FilesApiError && err.isMethodNotAllowed()) {
  //   setSaveError(REMOTE_PEER_OUTDATED_MESSAGE)
  //   setSaveState('idle'); return 'peer-outdated'
  // }
  // INSERT (RMW-05): 401 = cap expired (distinct from generic; copy at discretion per A3)
  // if (err instanceof FilesApiError && err.isUnauthorized()) {
  //   setSaveError(ACCESS_EXPIRED_MESSAGE)
  //   setSaveState('idle'); return 'expired'
  // }

  // generic — buffer still preserved (T-125-08), line 128-131
  setSaveError("Couldn't save the file. Your changes are still here — try again.")
  setSaveState('idle')
  return 'error'
}
```

---

### `frontend/src/components/FileBrowserTab.tsx` (RMW-04/05 — surface the new outcomes)

**Analog:** same file's existing conflict/save-error dispatch (path-prefix-generic; remote = `/api/files/remote/{sid}`). The new `'peer-outdated'`/`'expired'` outcomes must render their distinct messages, NOT the generic retry copy (Pitfall 4). Same dispatch shape as the existing `'conflict'` handling.

---

### `internal/tui/remote_files_client.go` (RMW-04 + RMW-05 — Go parity)

**Analog:** the 5 existing `StatusCode != http.StatusOK` blocks in the SAME file (one per verb).

**Imports** (`remote_files_client.go:1-16`): `crypto/tls`, `net/http`, `fmt`, `strings`, `internal/files`. TLS 1.2+ pinned at line 58 (`tls.Config{MinVersion: tls.VersionTLS12}`) — do NOT lower.

**Error-mapping pattern to extend** (CAP-LEAK invariant — error strings interpolate ONLY `(status, body)`, never URL/cap; comment at lines 219/247/281/315). The `WriteFile` block is the template:
```go
// remote_files_client.go:232-235 (WriteFile) — the sibling to extend:
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
    return files.FileWriteResponse{}, fmt.Errorf("remote files write: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
```
ADD, BEFORE the generic `!= OK` check, in each of the 4 write methods (`WriteFile` :221, `DeleteFile` :249, `RenameFile` :283, `MkdirFile` :317):
```go
// Package-level sentinels + verbatim const (grep-gated; MUST byte-match the TS const):
var ErrRemotePeerNoWriteSupport = errors.New(remotePeerOutdatedMessage)
const remotePeerOutdatedMessage = "The remote session is running an older version of AgentHub that does not support file writes."

if resp.StatusCode == http.StatusMethodNotAllowed { return files.FileWriteResponse{}, ErrRemotePeerNoWriteSupport }  // RMW-04
if resp.StatusCode == http.StatusUnauthorized { return files.FileWriteResponse{}, ErrRemoteCapExpired }            // RMW-05
```
TUI write surface maps `ErrRemotePeerNoWriteSupport` → verbatim copy and `ErrRemoteCapExpired` → "access expired". Pitfall 3: this is the UPSTREAM-propagated 405 (old peer), distinct from the proxy's own 405 (`TestRemoteFiles_405OnUnsupportedMethods`).

---

### `cmd/playwright-fixture/main.go` — extend `startRemotePeerFixture` to persist writes

**Analog:** the existing read-only `startRemotePeerFixture` in the SAME file (`main.go:407-496`).

**Current read-only mux pattern** (`main.go:447-492`): `http.NewServeMux()` + a `guard` closure (line 449) that checks `?cap=fixtureRemoteCap` (401 on miss) and `?session=` (404 on miss), wrapping canned `GET /api/files/{list,stat,read}` handlers returning fixed bytes (`"hello world"`, line 486). `WRITE_CAP` already emitted at `main.go:282`.

**Extension (NET-NEW, Pitfall 2 — must genuinely persist):** back the write verbs with a real `files.Sandbox` rooted at a temp dir (or mount the real `files.Handler`) so `PUT /api/files/write` then `GET /api/files/read` round-trips actual bytes — NOT canned `"hello world"`. Reuse the existing `guard` closure for cap/session checks. Add `{write,upload,delete,rename,mkdir}` routes mirroring the real webserver handler set. The byte-shape must stay consistent with the Go parity test's canonical expectations (comment at `main.go:420-422`).

**Version-gate variant (RMW-04):** a SECOND fixture peer whose write verbs return upstream `http.StatusMethodNotAllowed` (simulating v3.4 with no write routes) — the `guard` returns 405 instead of dispatching. Used by the 405 mapping tests.

---

### `internal/daemon/remote_files_write_parity_test.go` (NEW — mirror 122 read harness)

**Analog:** `internal/daemon/remote_files_parity_test.go` (455 LOC, `package daemon_test`).

**MANDATORY package + helpers** (Pattern 1; avoids the `tui`→`daemon` import cycle — Pitfall 1):
```go
// remote_files_parity_test.go:36 — external test package (NOT package daemon)
package daemon_test
```
Reuse the 4 exported `*ForTest` helpers + the two fixture/setup builders, all verified present:
```go
newFixtureRemotePeer(t)                            // :88  → httptest.NewTLSServer (:133) — EXTEND to persist writes
newDaemonAPIWithUpstreamCert(t, upstream)          // :139 → engine.ConfigDirForTest(t.TempDir()) :142
                                                   //        api.SetRemoteFilesClientForTest(upstream.Client()) :144
daemon.API.Handler()                               // wrapped by httptest.NewServer (:156)
tui.NewRemoteFilesClientForTest(url, cap, client)  // :182 — direct Observer B
assertNoCapInError(t, err)                         // :443-446 — CAP-LEAK invariant on EVERY direct-client error
```

**Write-then-read parity assertion shape** (mirror `TestRemoteFiles_CrossSurface_Parity` :151, sub-tests at :202/:246/:271 each call `assertNoCapInError`):
1. Observer A (proxy): `PUT /api/files/remote/{sid}/write?path=x.txt` body=`content-A`; then proxy `GET .../read?path=x.txt` → assert `== content-A`.
2. Observer B (direct `RemoteFilesClient`): `WriteFile(ctx, sid, "y.txt", content-B)`; `ReadFile` → assert `== content-B`.
3. Cross-observer: a write by A is read-back-identical by B against the SAME fixture sandbox (proves both hit one persisted state).
4. `assertNoCapInError(t, err)` on every error path.

A sibling test for RMW-04 points both observers at the version-gate fixture → assert each maps upstream 405 → the (cap-free) friendly error.

---

### `frontend/e2e/files-browser.spec.ts` (NEW scenario 18 — Observer C, HTTPS write-then-read)

**Analog:** scenarios 16 (`:430`) + 17 (`:501`) in the SAME file, plus `frontend/e2e/files-write.spec.ts` for `WRITE_CAP` URL building (`:37`, `:86`).

**Contract-not-DOM pattern (Anti-Pattern: no DOM modal flow)** — mirror scenario 16's `APIRequestContext` shape (`:430-480`):
```typescript
const ctx = await playwrightRequest.newContext({ ignoreHTTPSErrors: true })  // :431
// write-then-read against the fixture peer:
//   PUT  remoteFilesWriteURL({ path: 'x.txt' })  body='content-C'  → 200   (use WRITE_CAP, files-write.spec.ts:37)
//   GET  remoteFilesReadURL({ path: 'x.txt' })                     → expect(await resp.text()).toBe('content-C')
```
`global-setup.ts:153` already parses `WRITE_CAP=` from the fixture stdout. Byte-shape must agree with the Go parity test (comment precedent at `:443-446`).

---

### `128-VERIFICATION.md` (NEW DOC — mirror 122 precedent)

**Analog:** `.planning/milestones/v3.4-phases/122-remote-session-file-browse-wiring/122-VERIFICATION.md`.

Mirror its structure: RMW-01..06 traceability table, automated results, and the verbatim two-machine UAT checklist (Machine A web-share + Machine B GUI + Machine B TUI; cross-surface write parity + cap-expiry failure mode). Operator-deferred execution (Pitfall 5 — write the checklist, do NOT run it; needs a second tailnet machine, STATE.md TD-3). Note Issue #24 closes on execution. Scan `scottkw/agenthub` open issues before recording UAT pass (MEMORY).

## Shared Patterns

### Cross-surface verbatim message (single const per language, grep-gated)
**Source precedent:** `homeDirWriteWarning` grep-verifiable-const (Phase 124-05), referenced in RESEARCH Pitfall 4.
**Apply to:** RMW-04 — ONE Go `const remotePeerOutdatedMessage` (`internal/tui/remote_files_client.go`) + ONE TS `export const REMOTE_PEER_OUTDATED_MESSAGE` (`frontend/src/lib/filesApi.ts`), both byte-equal to the SC3 string. Grep-gate: the verbatim string must appear in BOTH languages or parity fails.

### CAP-LEAK invariant (cap token never in error strings)
**Source:** `assertNoCapInError` (`internal/daemon/remote_files_parity_test.go:443-446`) + `redactCapTokenFromError` (Go) + the `(status, body)`-only interpolation comments (`remote_files_client.go:219/247/281/315`).
**Apply to:** all new 405/401 messages (fixed strings, no interpolated URL/cap) and every new parity-test error assertion.

### TLS 1.2+ pinning (do not lower)
**Source:** `remote_files_client.go:58` (`tls.Config{MinVersion: tls.VersionTLS12}`); `InsecureSkipVerify` intentional for self-signed tailnet certs (`//nolint:gosec`).
**Apply to:** fixture peer (`httptest.NewTLSServer`) — unchanged; direct client — unchanged.

### Buffer preservation on save failure (T-125-08, locked anti-pattern)
**Source:** `useFilesWrite.ts:121` (412 path does NOT clear buffer); generic branch `:128-131` likewise.
**Apply to:** the new `'peer-outdated'` and `'expired'` branches — NEVER clear `editContent`.

## No Analog Found

None. Every Phase 128 delta has a concrete in-repo precedent (this is an integration + parity-proof phase, ~80% pre-built per RESEARCH).

## Metadata

**Analog search scope:** `internal/tui/`, `internal/daemon/`, `internal/files/`, `frontend/src/lib/`, `frontend/src/components/`, `frontend/e2e/`, `cmd/playwright-fixture/`, `.planning/milestones/v3.4-phases/122-*`
**Files scanned:** filesApi.ts, useFilesWrite.ts, remote_files_client.go, remote_files_parity_test.go, playwright-fixture/main.go, files-browser.spec.ts, 122-VERIFICATION.md (located)
**Pattern extraction date:** 2026-06-14
