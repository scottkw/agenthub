---
phase: 89-vendored-terminal-assets-csp
plan: "01"
subsystem: webserver/vendor
tags: [go, csp, security, vendor]
dependency_graph:
  requires: []
  provides:
    - web/vendor/xterm/xterm.js
    - web/vendor/xterm/xterm.css
    - web/vendor/xterm/addon-fit.js
    - web/vendor/xterm/VERSION
    - internal/webserver/vendor_drift_test.go
  affects:
    - internal/webserver (test suite)
    - web/vendor/xterm (new tree)
tech_stack:
  added: []
  patterns:
    - "Vendored third-party assets committed as byte-identical copies of node_modules files"
    - "Source-parse drift guard test using os.ReadFile + regexp scan across two config files"
key_files:
  created:
    - web/vendor/xterm/xterm.js
    - web/vendor/xterm/xterm.css
    - web/vendor/xterm/addon-fit.js
    - web/vendor/xterm/VERSION
    - internal/webserver/vendor_drift_test.go
  modified: []
decisions:
  - "Vendor path locked at web/vendor/xterm/ per D-01/D-02 (URL space /assets/xterm/* mapped in Plan 04)"
  - "VERSION file uses two-line LF plaintext format: @xterm/xterm@6.0.0 and @xterm/addon-fit@0.11.0"
  - "Test reads ../../frontend/pnpm-lock.yaml and ../../web/vendor/xterm/VERSION via relative path (go test cwd = internal/webserver/)"
  - "Error messages cite Phase 89 D-04 for traceability; drift-error format string exactly: version drift for %s: pnpm-lock=%s, VERSION=%s"
metrics:
  duration: "~8 minutes"
  completed: "2026-04-22"
  tasks: 2
  files: 5
---

# Phase 89 Plan 01: Vendor xterm Assets + Drift Guard Summary

Committed xterm.js, xterm.css, and addon-fit.js as byte-identical vendored copies under `web/vendor/xterm/`, with a VERSION manifest and a Go drift-guard test that fails CI if pnpm-lock.yaml and VERSION ever diverge.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Copy xterm vendor files + write VERSION manifest | c6dbfdb | web/vendor/xterm/{xterm.js,xterm.css,addon-fit.js,VERSION} |
| 2 | Write vendor-drift source-parse test (D-04/D-20) | 06bf352 | internal/webserver/vendor_drift_test.go |

## Vendored File Details

| File | Size (bytes) | Source |
|------|-------------|--------|
| web/vendor/xterm/xterm.js | 488,663 | frontend/node_modules/@xterm/xterm/lib/xterm.js |
| web/vendor/xterm/xterm.css | 7,112 | frontend/node_modules/@xterm/xterm/css/xterm.css |
| web/vendor/xterm/addon-fit.js | 1,521 | frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js |
| web/vendor/xterm/VERSION | 43 | Authored (plaintext version manifest) |
| **Total payload** | **497,296** | — |

## Resolved Versions in VERSION

```
@xterm/xterm@6.0.0
@xterm/addon-fit@0.11.0
```

Verified against `frontend/pnpm-lock.yaml` — both packages appear as top-level keys (two-space indent, quoted `'@xterm/<pkg>@<version>':` format, pnpm v7+ stable).

## Drift Guard Test

`TestXtermVendorVersionsMatchPnpmLock` in `internal/webserver/vendor_drift_test.go`:
- Reads `../../frontend/pnpm-lock.yaml` and parses `@xterm/xterm` and `@xterm/addon-fit` resolved versions using `^  '(@xterm/(?:xterm|addon-fit))@([0-9.]+)':` regex
- Reads `../../web/vendor/xterm/VERSION` and parses each line as `@<scope>/<pkg>@<version>`
- Fails with actionable error message (package name, both versions, remediation command) if versions diverge
- `go test ./internal/webserver/ -run TestXtermVendorVersionsMatchPnpmLock -count=1`: **PASS**
- `go vet ./internal/webserver/`: **PASS**

## Deviations from Plan

None — plan executed exactly as written.

Note: RESEARCH.md reference implementation used `web/assets/xterm/VERSION` path, but the PLAN's locked vendor path `web/vendor/xterm/` (D-01/D-02) was used as the authoritative source. Test file correctly reflects the plan's path, not the draft research path.

## Known Stubs

None.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. The vendored files are static assets committed to the repository; they do not execute in the Go process. No new threat surface beyond what is documented in the plan's threat model (T-89-02, T-89-05 — both mitigated by this plan).

## Self-Check: PASSED

All 5 files confirmed on disk. Commits c6dbfdb and 06bf352 verified in git log.
