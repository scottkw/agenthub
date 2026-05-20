---
phase: 119
fixed_at: 2026-05-20
review_path: .planning/phases/119-webserver-routes-files-read-capability-plumbing/119-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 119: Code Review Fix Report

**Fixed at:** 2026-05-20
**Source review:** .planning/phases/119-webserver-routes-files-read-capability-plumbing/119-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (WR-01, WR-02 — Warnings only; Info findings skipped per scope)
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: Doc comment misstates Go 1.22 mux HEAD/GET behavior

**Files modified:** `internal/webserver/server.go`
**Commit:** d0d365e
**Applied fix:** Reworded the setupRoutes comment block (around lines 458-466)
to accurately describe Go 1.22's `ServeMux` HEAD→GET auto-dispatch. The new
comment:
- States that explicit `HEAD /api/files/read` registration documents the FS-06
  intent (ServeContent path) and locks behavior against future mux semantics
  changes — NOT that HEAD/GET are distinct methods in the mux.
- Adds a sentence clarifying that `/list` and `/stat` intentionally rely on
  the GET→HEAD auto-dispatch, preempting a future maintainer adding redundant
  HEAD wrappers as "missing" registrations.

Adopted the review's recommended wording verbatim and extended it with the
/list and /stat note so the doc fully covers all three routes.

### WR-02: No test for SetFilesHandler called after Start() — race-detector exposure

**Files modified:** `internal/webserver/server.go`
**Commit:** 24c95a1
**Applied fix:** Selected Option (b) from the review (lowest-risk, matches
project conventions). Reworded the `SetFilesHandler` godoc from
"Must be called before Start()" to "Must be called before the first HTTPS
request reaches /api/files/*", with an explanatory paragraph noting:
- Production callers (`AutoStartWebServer`, `handleWebServerStart`) still
  satisfy the original before-Start() ordering.
- Tests are permitted to call SetFilesHandler after Start() because the TLS
  handshake establishes happens-before synchronization between the test
  goroutine write and the eventual handler-goroutine read.
- The setter remains single-write, no-mutex (mirrors `SetSessionResolver`,
  `SetSigningKey`).

This captures the actual safety invariant the existing tests rely on without
requiring a refactor of `testServer` or introducing an `atomic.Pointer` guard.

## Skipped Issues

None. Info findings (IN-01 through IN-04) were intentionally out of scope per
the `critical_warning` fix policy.

---

_Fixed: 2026-05-20_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
