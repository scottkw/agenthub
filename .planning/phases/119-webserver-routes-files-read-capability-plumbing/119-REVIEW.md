---
phase: 119
status: findings
critical_count: 0
warning_count: 2
info_count: 4
---

# Phase 119: Code Review Report

**Reviewed:** 2026-05-20
**Depth:** standard (with cross-file trace into daemon/api.go and capability_mw.go)
**Files Reviewed:** 5
**Status:** findings (no blockers; 2 warnings, 4 info-level observations)

## Summary

Phase 119 wires `*files.Handler` from the daemon into the webserver via a new
`SetFilesHandler` setter, then mounts four routes (`GET/HEAD /api/files/*`) under
`requireFilesRead` on the public TLS mux. The integration test file
(`files_routes_test.go`) covers the headline behaviors well: 200 on owner cap,
403 with literal "files.read" body on viewer cap, 401 (NOT 404) on missing cap,
405 on POST/PUT/DELETE via Go 1.22+ method-prefix mux, 503 when handler not
configured, traversal still rejected, and two WEB-05 defense-in-depth regression
guards (no CSP header, no `text/html` content type).

No correctness or security blocker was found. The closure-based nil-safety
pattern is implemented correctly; both daemon construction sites
(`AutoStartWebServer` and `handleWebServerStart`) wire `SetFilesHandler` BEFORE
`ws.Start()`, matching the documented contract.

Two warnings concern (a) a doc inaccuracy about Go 1.22 mux HEAD-vs-GET
dispatch that could mislead future maintainers, and (b) a thin coverage gap
around `SetFilesHandler(nil)` re-entry. Info items are mostly small naming/
defensive-coding observations.

## Critical Issues

None.

## Warnings

### WR-01: Doc comment misstates Go 1.22 mux HEAD/GET behavior

**File:** `internal/webserver/server.go:458-460`
**Issue:** The setupRoutes comment states *"HEAD is registered as a separate
route because the Go 1.22 mux treats HEAD and GET as distinct methods"*. This
is inaccurate. Go 1.22's `net/http.ServeMux` DOES auto-dispatch HEAD requests
to a registered GET handler when no method-specific HEAD pattern exists (this
is documented in `net/http` package docs as the "HEAD requests also match GET
handlers" rule). The explicit `HEAD /api/files/read` registration is therefore
not strictly necessary, only defensive. Worse, the same comment implicitly
suggests `HEAD /api/files/list` and `HEAD /api/files/stat` would NOT work
without explicit registration — but they DO work via the GET→HEAD auto-dispatch,
and tests for those paths do not verify HEAD at all.

A future maintainer reading this comment may believe registering only GET means
HEAD returns 405, and may add an explicit HEAD wrapper for every JSON route
"to fix the gap" — pure cargo-culting against a misleading invariant.

**Fix:** Reword the comment to reflect the actual mux semantics, or add an
explicit `HEAD /api/files/list` and `HEAD /api/files/stat` registration to
match the documented intent. Recommended wording:

```go
// HEAD /api/files/read is registered explicitly even though Go 1.22's
// ServeMux dispatches HEAD to the GET handler when no HEAD-specific
// pattern is registered — explicit registration documents the FS-06 intent
// (HEAD must hit ServeContent for Content-Length without body) and locks
// the behavior against any future mux semantics change.
```

### WR-02: No test for SetFilesHandler called after Start() — race-detector exposure

**File:** `internal/webserver/files_routes_test.go:37-58` and
`internal/webserver/server.go:142-150`
**Issue:** The `newFilesTestServer` helper calls `ws.Start()` (via `testServer`)
and THEN calls `ws.SetFilesHandler(h)`. `SetFilesHandler` writes
`ws.filesHandler` with no mutex (documented as "set once before Start()"
contract). After `Start()`, `http.Serve` runs in a background goroutine; the
test goroutine writes to `filesHandler` and a future request-handling goroutine
reads it. In production, both `AutoStartWebServer` and `handleWebServerStart`
call `SetFilesHandler` BEFORE `Start()`, satisfying the contract — but the
tests violate it.

In practice the TLS connection handshake establishes happens-before
synchronization between the test write and the eventual handler read, so the
race detector typically does not fire. However, this matches an existing
project pattern (`SetSessionResolver`, `SetSigningKey` calls in
`capability_test.go` follow the same shape) — so the new tests do not regress
behavior, they propagate an existing risk.

The narrower concern is documented contract divergence: `SetFilesHandler`'s
godoc says "Must be called before Start()." but the canonical test helper
violates this. Either the contract is wrong or the tests are wrong.

**Fix:** Either (a) tighten the test helper to call `SetFilesHandler` BEFORE
`ws.Start()` (would require refactoring `testServer` to expose a pre-Start
hook), or (b) relax the godoc to acknowledge the tested usage ("must be called
before the first HTTPS request reaches `/api/files/*`"), or (c) add a `sync.Once`
+ atomic.Pointer-style guard to `filesHandler` so the race-detector is
satisfied regardless of caller order. Option (b) is lowest-risk and matches
project conventions.

## Info

### IN-01: `filesDispatch` closure indirection adds an allocation per request

**File:** `internal/webserver/server.go:461-474`
**Issue:** `filesDispatch` takes a `func(*files.Handler) http.HandlerFunc` and
returns an `http.HandlerFunc` that invokes `op(h)(w, r)`. Each call to
`op(h)` constructs a new method-value (closure over `h`) per request. This is
allocation-on-hot-path for a route that may be called frequently as the file
browser walks a tree.

This is a quality observation, not a correctness one — and the v1 review
scope excludes performance. Listing as info so a future profiling pass can
trivially inline as e.g.:
```go
mux.HandleFunc("GET /api/files/list", ws.requireFilesRead(func(w, r) {
    h := ws.filesHandler
    if h == nil { http.Error(w, "...", 503); return }
    h.List(w, r)
}))
```

**Fix:** Acceptable as-is for v1 (clarity > performance). Revisit if
file-browser profiling shows allocator pressure.

### IN-02: Duplicate test setup — `newFilesTestServer` not reused in nil-handler test

**File:** `internal/webserver/files_routes_test.go:263-279`
**Issue:** `TestFilesRoutes_NilHandlerReturns503` re-implements the
testServer/SetSigningKey/EnableSession setup inline rather than reusing
`newFilesTestServer` (and then setting `ws.filesHandler = nil` after the fact,
or providing a variant helper that skips the SetFilesHandler call). The inline
duplication is small but every drift point increases the chance future
test-helper changes don't propagate.

**Fix:** Extract a `newFilesTestServerWithoutHandler` helper, or accept a
boolean `wireHandler bool` parameter on `newFilesTestServer`. Low priority.

### IN-03: `fileURL` builds query strings without `url.QueryEscape`

**File:** `internal/webserver/files_routes_test.go:63-69`
**Issue:** `fileURL` does raw string concatenation for `?session=` and `?path=`.
Tests using `path="../../etc/passwd"` happen to work because `..` and `/` are
URL-safe. But if any future test wants to assert on a path with spaces, `&`,
`#`, or `?`, the URL builds wrong and silently produces a different request than
intended — making test failures hard to diagnose.

**Fix:**
```go
q := url.Values{}
q.Set("session", sid)
q.Set("path", path)
if token != "" {
    q.Set("cap", token)
}
return ws.BaseURL() + route + "?" + q.Encode()
```

### IN-04: `doRequest` closes Body but returns the response struct

**File:** `internal/webserver/files_routes_test.go:74-87`
**Issue:** `doRequest` reads and closes the response body, then returns
`*http.Response` with body separately. The returned `resp.Body` is now an
already-closed `io.ReadCloser`. Any test calling `resp.Body.Read(...)` after
this point would hit `http: read on closed response body`. No current test does
so — only `resp.StatusCode`, `resp.Header`, and `resp.ContentLength` are read,
all of which are fine post-close. But the function's contract is misleading;
the comment line 71-73 even says *"The body bytes are populated into resp.Body
via a bytes.Reader so tests can re-read as needed"* — this is FALSE in the
current implementation (no `bytes.NewReader` reassignment occurs).

**Fix:** Either implement the documented behavior:
```go
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
resp.Body = io.NopCloser(bytes.NewReader(body))
return resp, body
```
or drop the misleading comment. Low priority — no current test depends on
re-reading, but the contract drift will mislead future authors.

## Structural Findings (fallow)

None provided.

---

_Reviewed: 2026-05-20_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

## REVIEW COMPLETE
