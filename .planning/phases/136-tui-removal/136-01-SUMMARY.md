---
phase: 136-tui-removal
plan: "01"
subsystem: tui-removal
tags: [go, tui, deletion, dependency-cleanup, charm]
dependency_graph:
  requires: []
  provides: [tui-package-deleted, charm-deps-pruned, go-build-clean]
  affects: [go.mod, go.sum, main.go, cmd_cli.go]
tech_stack:
  added: []
  patterns: [go-mod-tidy, git-rm-atomic]
key_files:
  created:
    - internal/daemon/remote_files_test_helpers_test.go
  modified:
    - main.go
    - cmd_cli.go
    - go.mod
    - go.sum
  deleted:
    - internal/tui/ (32 files — 18 source, 14 test)
    - cmd_tui.go
    - internal/daemon/remote_files_parity_test.go
    - internal/daemon/remote_files_write_parity_test.go
decisions:
  - "Delete parity test helpers atomically alongside parity test files — relay_remote_files_test.go shared them; extracted to remote_files_test_helpers_test.go (Rule 1 auto-fix)"
  - "Comment-only references to internal/tui in internal/daemon/client.go and internal/attach/attach.go left for plan 02 polish pass per research notes"
  - "Pre-existing TestSER03_NoAutoSavePatterns failure in internal/release confirmed pre-existing (fails before our commits); logged as deferred, not caused by this plan"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-19"
  tasks_completed: 2
  files_changed: 40
---

# Phase 136 Plan 01: TUI Package Deletion Summary

**One-liner:** Deleted 32-file `internal/tui/` package plus all import consumers (`cmd_tui.go`, 2 daemon parity test files), stripped 5 direct charm.land/charmbracelet deps plus ~15 indirect orphans via `go mod tidy`, build and surviving tests green.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Atomic deletion of TUI package and all import consumers | 5a771762 | internal/tui/ (32 files), cmd_tui.go, 2 daemon parity tests, main.go, cmd_cli.go |
| Auto-fix | Extract shared daemon test helpers | fc27ef2c | internal/daemon/remote_files_test_helpers_test.go |
| 2 | Prune unused charm dependencies with go mod tidy | 04902169 | go.mod, go.sum |

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `! ls internal/tui` | PASS — directory gone |
| No `"internal/tui"` import statements | PASS — zero matches |
| `charm.land/(bubbletea|bubbles|lipgloss)` removed from go.mod | PASS |
| `charmbracelet/glamour` removed from go.mod | PASS |
| `charmbracelet/x/ansi` removed from go.mod | PASS |
| `golang.org/x/term` preserved in go.mod | PASS |
| `internal/attach/attach.go` survives | PASS |
| `internal/daemon/engine.go` unmodified | PASS |
| `go test -race -short ./internal/daemon/...` | PASS |
| `go test -race -short ./...` (excluding pre-existing) | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Shared daemon test helpers orphaned by parity file deletion**
- **Found during:** Task 1 verification — `go test -race -short ./...` showed `internal/daemon [build failed]`
- **Issue:** `relay_remote_files_test.go` depends on `newFixtureRemotePeer`, `newDaemonAPIWithUpstreamCert`, `fixtureCap`, and `canonicalListResponse` which were defined in `remote_files_parity_test.go` (deleted). Research missed that these helpers were shared across test files.
- **Fix:** Extracted all four non-TUI helpers into a new `internal/daemon/remote_files_test_helpers_test.go` with no `internal/tui` import. The parity-test-specific TUI oracle functions were not extracted (they are moot after TUI removal).
- **Files modified:** internal/daemon/remote_files_test_helpers_test.go (created)
- **Commit:** fc27ef2c

## Known Stubs

None — this is a pure deletion plan; no data flows or UI components were created.

## Deferred Items

| Item | Reason |
|------|--------|
| Comment-only TUI references in `internal/daemon/client.go:516` and `internal/attach/attach.go:1-3` | Research noted these as cosmetic cleanup for plan 02 polish pass; no import or code change needed |
| Comment-only TUI references in `frontend/src/lib/filesApi.ts:115` and `frontend/src/components/DaemonManagerPanel.tsx:389` | DaemonManagerPanel.tsx is scheduled for deletion in Phase 138; filesApi.ts comment is cosmetic, plan 02 |
| `TestSER03_NoAutoSavePatterns` failure in `internal/release` | Pre-existing failure confirmed by testing at commit `fd3df73c` (docs-only commit before our changes). File `cmd/playwright-fixture/dist/assets/index-Dklc5ak1.js` matches forbidden auto-save vocabulary. Not caused by Phase 136. |

## Threat Flags

None — pure deletion phase. Attack surface uniformly reduced: `RemoteFilesClient` direct HTTPS + cap-token path removed. No new network endpoints, auth paths, or schema changes introduced.

## Self-Check: PASSED

- `internal/daemon/remote_files_test_helpers_test.go` — EXISTS
- Commit 5a771762 — EXISTS (`feat(136-01): atomic deletion of TUI package and all import consumers`)
- Commit 04902169 — EXISTS (`chore(136-01): prune unused charm dependencies with go mod tidy`)
- Commit fc27ef2c — EXISTS (`fix(136-01): extract shared daemon test helpers broken by parity file deletion`)
