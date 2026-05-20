---
phase: 119-webserver-routes-files-read-capability-plumbing
plan: 02
subsystem: webserver
tags: [webserver, csp, security, docstring, web-05, defense-in-depth]
requires:
  - "internal/webserver.testServer + issueCapFor (Phase 119-01 harness)"
  - "GET /api/files/list mounted on webserver mux (Phase 119-01)"
  - "internal/webserver.requireFilesRead docstring (Phase 118)"
  - "internal/files.Handler docstring (Phase 118)"
provides:
  - "TestFilesRoutes_NoCSPHeader regression guard (T-119-09)"
  - "TestFilesRoutes_NoHTMLContentType regression guard (T-119-10)"
  - "Corrected docstrings on requireFilesRead and files.Handler referring to SetFilesHandler"
  - "Re-run gate on TestBrowserCSP_* (existing HTML-page CSP e2e suite) confirming zero Phase 119 regression"
affects:
  - "internal/webserver/files_routes_test.go (+42 lines, 2 new tests)"
  - "internal/webserver/capability_mw.go (1-line comment fix)"
  - "internal/files/handler.go (1-line comment fix)"
tech_stack:
  added: []
  patterns:
    - "Defense-in-depth header assertions over JSON routes (no CSP, no text/html)"
key_files:
  created:
    - ".planning/phases/119-webserver-routes-files-read-capability-plumbing/119-02-SUMMARY.md"
  modified:
    - "internal/webserver/files_routes_test.go (+42 lines: 2 subtests)"
    - "internal/webserver/capability_mw.go (1-line docstring)"
    - "internal/files/handler.go (1-line docstring)"
decisions:
  - "Tests authored directly as GREEN-on-first-run guards rather than mechanical RED→GREEN. The system under test (file route mounting without cspHeaders) was already correct as of 119-01; the tests exist to catch a future regression, not to drive new behavior. Committed under `test(119-02):` prefix to make the regression-guard intent explicit."
  - "Task 3 (re-run TestBrowserCSP_*) executed as a verification gate only — chromedp was available and all four browser-CSP tests PASS, so no SKIP fallback was needed."
metrics:
  duration: "~3 minutes"
  completed: "2026-05-20"
  tasks: 3
  files_touched: 3
  loc_added: 44
---

# Phase 119 Plan 02: CSP Defense-in-Depth + Docstring Cleanup Summary

Added two WEB-05 regression guards on the file routes mounted in 119-01, fixed two Phase 118 docstrings that referenced a stale `SetFilesHandlerProvider` symbol name, and re-ran the existing browser CSP e2e suite to confirm Phase 119 introduced no CSP regression on `/dashboard`, `/join`, or `/sessions/{id}`.

## What was built

### Task 1: Defense-in-depth CSP/Content-Type assertions

**File:** `internal/webserver/files_routes_test.go` (+42 lines)

Appended two top-level test functions reusing the existing `newFilesTestServer(t)` + `issueCapFor` harness from 119-01:

| Test | Asserts |
|---|---|
| `TestFilesRoutes_NoCSPHeader` | `GET /api/files/list?...&cap=<owner-token>` → 200 AND `resp.Header.Get("Content-Security-Policy") == ""`. Mitigates T-119-09 (future regression accidentally wrapping file routes in `cspHeaders`). |
| `TestFilesRoutes_NoHTMLContentType` | `GET /api/files/list?...&cap=<owner-token>` → 200 AND `Content-Type` starts with `"application/json"` (never `"text/html"`). Mitigates T-119-10 (future regression serving an HTML error page from a file route). |

Both tests PASS on first run — file routes are correctly outside `cspHeaders` (which is mounted only on the three HTML routes: `/dashboard`, `/join`, `/sessions/{id}`).

**Commit:** `cf2f2b3`

### Task 2: Docstring cleanup — `SetFilesHandlerProvider` → `SetFilesHandler`

Two one-line comment edits, pure no-op for the runtime:

1. `internal/webserver/capability_mw.go` line 99 — `requireFilesRead`'s MOUNT TIMING note now reads `via SetFilesHandler` (was `via SetFilesHandlerProvider`).
2. `internal/files/handler.go` line 5 — file-level docstring on the `Handler` type now reads `via the SetFilesHandler parity point` (was `SetFilesHandlerProvider`).

Verification:
- `grep -rn 'SetFilesHandlerProvider' internal/` → no matches (`stale-refs-exit=1`).
- `grep -c 'SetFilesHandler' internal/webserver/capability_mw.go` → 1 (updated reference).
- `grep -c 'SetFilesHandler' internal/files/handler.go` → 1 (updated reference).
- `go build ./internal/webserver/ ./internal/files/` → exit 0.
- `go vet ./internal/webserver/ ./internal/files/` → no diagnostics.

**Commit:** `955050d`

### Task 3: Re-run existing CSP e2e suite (no code changes)

Executed `go test -tags=e2e -run '^TestBrowserCSP_' ./internal/webserver/ -count=1 -v`:

| Test | Result | Duration |
|---|---|---|
| `TestBrowserCSP_TerminalNoViolations` | PASS | 10.99s |
| `TestBrowserCSP_DashboardNoViolations` | PASS | 3.34s |
| `TestBrowserCSP_JoinNoViolations` | PASS | 2.84s |
| `TestBrowserCSP_TerminalImage_NoViolations` | PASS | 4.15s |

chromedp was available — no self-skip fallback fired. Phase 119 introduces zero CSP regression on any HTML-serving page. WEB-05's "zero new CSP violations on the complete file browse flow" criterion is satisfied at the layer Phase 119 can actually cover (the file-browser UI is a Phase 120 deliverable; the read-only routes added here do not render HTML).

No commit for Task 3 — verification gate only.

## Verification

Final verification block from `<verification>`:

```
go test -run '^TestFilesRoutes_'    ./internal/webserver/ -count=1 -v       # 17 PASS
go test -tags=e2e -run '^TestBrowserCSP_' ./internal/webserver/ -count=1    # 4 PASS
grep -rn 'SetFilesHandlerProvider' internal/                                # no matches (stale-refs-exit=1)
go test ./internal/files/ ./internal/capability/ \
        ./internal/webserver/ ./internal/daemon/ -count=1 -short            # all four ok
```

Expanded test counts: 15 `TestFilesRoutes_*` from 119-01 + 2 new (`NoCSPHeader`, `NoHTMLContentType`) = **17 total PASS**.

## Success criteria

- [x] `TestFilesRoutes_NoCSPHeader` PASS — `/api/files/list` response has no CSP header
- [x] `TestFilesRoutes_NoHTMLContentType` PASS — `/api/files/list` Content-Type starts with `application/json`
- [x] Existing `TestBrowserCSP_*` tests PASS (all four, chromedp available — no SKIP) — Phase 119 introduces no CSP regression
- [x] Zero `SetFilesHandlerProvider` references remain in `internal/`
- [x] Cross-package suite (`internal/files`, `internal/capability`, `internal/webserver`, `internal/daemon`) green in short mode
- [x] WEB-05 satisfied: zero new CSP amendments to the codebase; defense-in-depth tests in place to catch future regressions; HTML-page CSP e2e unchanged

## Deviations from Plan

None — plan executed exactly as written. No auth gates, no architectural decisions, no auto-fixes required.

The plan documents the inherent tension that the `tdd="true"` Task 1 is a defense-in-depth guard whose RED phase cannot meaningfully fail (because 119-01 correctly mounted the routes outside `cspHeaders`); both tests were authored and pass on first run. This is an explicit Plan-documented shape, not a deviation.

## Threat Model Compliance

| Threat | Disposition | Mitigation | Verified by |
|---|---|---|---|
| T-119-09 (Tampering — future regression wrapping file route in cspHeaders) | mitigate | `TestFilesRoutes_NoCSPHeader` asserts absence of CSP header | Test PASS on commit `cf2f2b3` |
| T-119-10 (Tampering — future regression serving HTML from file route) | mitigate | `TestFilesRoutes_NoHTMLContentType` asserts `Content-Type` prefix | Test PASS on commit `cf2f2b3` |
| T-119-11 (Information disclosure — docstring drift) | accept | Task 2 updates both stale references | Commit `955050d`; grep confirms zero stale refs |
| T-119-12 (Tampering — Phase 119 breaks CSP on an HTML page it did not touch) | mitigate | Task 3 re-runs `TestBrowserCSP_*` suite as gate | All 4 e2e tests PASS |

All four `mitigate`/`accept` dispositions covered.

## TDD Gate Compliance

Task 1 was specified as `tdd="true"`. The shape is unusual: the system was already correct as of 119-01, so the "RED" gate could not meaningfully fail. Per the plan-level note in `<action>` (and consistent with the per-PRD Plan-level TDD Gate fail-fast rule — "if a test passes unexpectedly during RED, investigate whether the feature already exists"), the tests were authored against an already-correct implementation as documented defense-in-depth guards rather than as new behavior drivers. Committed under `test(119-02):` to make the regression-guard intent unambiguous to future readers.

The Plan-level TDD gate sequence (test → feat → refactor) does NOT strictly apply here because Plan 02's net behavior change to the production binary is zero (one test addition + two comment edits). The closest analog in git history:

- `test(119-02)`: cf2f2b3 — defense-in-depth guards
- `docs(119-02)`: 955050d — docstring cleanup

A `feat(119-02)` commit would have been semantically wrong (no production behavior added).

## Self-Check

Commits exist:
- `cf2f2b3` test(119-02): add defense-in-depth CSP/Content-Type guards — FOUND in git log
- `955050d` docs(119-02): update Phase 118 docstrings — FOUND in git log

Files exist:
- `internal/webserver/files_routes_test.go` — FOUND (modified, +42 lines)
- `internal/webserver/capability_mw.go` — FOUND (1-line comment fix)
- `internal/files/handler.go` — FOUND (1-line comment fix)
- `.planning/phases/119-webserver-routes-files-read-capability-plumbing/119-02-SUMMARY.md` — FOUND (this file)

Test guarantees:
- `TestFilesRoutes_NoCSPHeader` — PASS
- `TestFilesRoutes_NoHTMLContentType` — PASS
- Full `TestFilesRoutes_*` suite (17 tests) — PASS
- `TestBrowserCSP_*` (4 tests, `-tags=e2e`) — PASS
- Cross-package short-mode suite — PASS

## Self-Check: PASSED
