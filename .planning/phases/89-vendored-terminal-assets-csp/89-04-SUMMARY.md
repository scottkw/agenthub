---
phase: 89-vendored-terminal-assets-csp
plan: "04"
subsystem: webserver
tags: [go, csp, security, integration, routing, embed]
dependency_graph:
  requires: [89-01, 89-02, 89-03]
  provides: [/assets/xterm/ route, /assets/ route, cspHeaders wiring on HTML handlers]
  affects: [plan-05-e2e]
tech_stack:
  added: []
  patterns: [embed.FS-two-subtrees, fs.Sub-scoped-filesystems, http.FileServerFS, http.StripPrefix, more-specific-pattern-wins, middleware-chain-csp-outermost]
key_files:
  created:
    - internal/webserver/assets_test.go
    - internal/webserver/csp_integration_test.go
    - internal/webserver/no_cdn_regression_test.go
  modified:
    - web/embed.go
    - internal/webserver/server.go
decisions:
  - "cspHeaders is outermost wrapper on /sessions/{id}: cspHeaders -> requireCapability -> handleTerminalPage. Failed auth responses still carry CSP (T-89-04 mitigation)."
  - "More-specific pattern /assets/xterm/ registered before /assets/ — Go http.ServeMux longest-prefix-wins gives xterm requests the vendored subtree, first-party requests the assets subtree."
  - "assetsNoStore helper added to apply Cache-Control: no-store to the /assets/* handlers (D-16)."
  - "Regression test TestSecurity_NoInlineScriptOrStyleInHTML parses web/*.html via regex, not just source grep, to avoid false negatives from quoted strings in code."
metrics:
  duration: "~10m"
  completed_date: "2026-04-22T22:55:00Z"
  tasks_completed: 5
  files_created: 3
  files_modified: 2
---

# Phase 89 Plan 04: Wire Assets + CSP into HTTP Mux

**One-liner:** Embed first-party and vendored trees into `webfs.WebFS`, mount `/assets/xterm/` and `/assets/` routes, wrap three HTML handlers with `cspHeaders`, and lock behavior with 16 integration/regression tests.

## Tasks Completed

| # | Name | Type | Commit | Result |
|---|------|------|--------|--------|
| 1 | Extend `web/embed.go` | feat | 3ab1767 | 13 files now embedded (3 HTML + 6 first-party + 4 vendor) |
| 2 | Extend `setupRoutes` in `server.go` | feat | 63f945b | /assets/xterm/, /assets/, three HTML handlers under cspHeaders |
| 3 | `assets_test.go` — 8 route tests | test | 15b70c4 | 8/8 GREEN |
| 4 | `csp_integration_test.go` — D-18 five-assertion suite x 3 routes | test | 0d81ea5 | 5/5 GREEN |
| 5 | `no_cdn_regression_test.go` — D-17 forbidden-strings guard | test | 76fbd99 | 3/3 GREEN |

## Artifacts

### `web/embed.go`
Explicit `//go:embed` lines per D-02 supply-chain-visibility rule:
- `assets/terminal.js assets/terminal.css assets/dashboard.js assets/dashboard.css assets/join.js assets/join.css`
- `vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION`

### `internal/webserver/server.go` (`setupRoutes`)
```go
assetsFS, _ := fs.Sub(webfs.WebFS, "assets")
xtermFS, _  := fs.Sub(webfs.WebFS, "vendor/xterm")
mux.Handle("/assets/xterm/", http.StripPrefix("/assets/xterm/", assetsNoStore(http.FileServerFS(xtermFS))))
mux.Handle("/assets/",       http.StripPrefix("/assets/",       assetsNoStore(http.FileServerFS(assetsFS))))

mux.Handle("/dashboard",         ws.cspHeaders(ws.handleDashboard))
mux.Handle("/join",              ws.cspHeaders(ws.handleJoin))
mux.Handle("/sessions/",         ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))
```

### Test files
- `assets_test.go` — 8 tests covering embed-served xterm.js/css, first-party js/css, Cache-Control, public (no-cap) access, 404, directory-listing suppression.
- `csp_integration_test.go` — TestCSPHeaderStrict_{TerminalPage, Dashboard, Join, CacheControl, OnAuthFailure} — each runs D-18's five assertions.
- `no_cdn_regression_test.go` — TestSecurity_NoCDNReferencesInWebAssets (5 forbidden CDN hosts under web/), TestSecurity_NoInlineScriptOrStyleInHTML, TestSecurity_NoAcceptAllOriginInWebserver.

## Test Results

```
=== Tests executed in worktree (go test -tags wailsassets ./internal/webserver/...) ===
PASS (16 new tests + all pre-existing tests)
ok github.com/scottkw/agenthub/internal/webserver 0.050s (filtered) / 1.086s (full package)
```

## Self-Check: PASSED

- [x] All 5 tasks executed and committed atomically
- [x] /assets/xterm/ serves vendored xterm assets with Cache-Control: no-store
- [x] /assets/ serves first-party terminal.js etc. with Cache-Control: no-store
- [x] Three HTML routes carry CSP header with all D-18 tokens
- [x] D-18 five-assertion suite passes per route
- [x] D-17 forbidden-strings regression test in place
- [x] All key-links from plan frontmatter satisfied (verified by pattern grep)
- [x] No modifications to STATE.md or ROADMAP.md (orchestrator owns them)

## Notes

Agent stream watchdog fired during the final summary step (after Task 5 commit landed). All 5 task commits (3ab1767, 63f945b, 15b70c4, 0d81ea5, 76fbd99) completed successfully; this SUMMARY.md was authored by the orchestrator post-hoc after verifying test pass in the worktree.
