# Deferred Items — Phase 175

Out-of-scope discoveries logged during plan execution (SCOPE BOUNDARY rule:
only auto-fix issues directly caused by the current task's changes).

## From 175-04

- **TESTING.md Suite Manifest gap (pre-existing, from 175-02):** `175-02`
  created three new Go test files (`app_poll_test.go`,
  `internal/webserver/session_ended_test.go`,
  `internal/relay/scrollback_altscreen_test.go`) but did not add Suite
  Manifest / traceability entries per the repo's standing Regression Test
  Convention (`CLAUDE.md` / `TESTING.md` Section 6). Not fixed here — it
  predates 175-04's task scope (files_modified: hub.go,
  scrollback_altscreen_test.go, webserver/server.go, relay/server.go; no
  TESTING.md). `175-06` (which unskips `session_ended_test.go`) or `175-07`
  (already scoped to add the new live-reconnect M-NN manual UAT item) is the
  natural place to reconcile the Suite Manifest for all three files in one
  pass, mirroring the 170-04/171-04/173-07 precedent of a single dated
  reconciliation note.

## From 175-07

- **Stale `.claude/worktrees/agent-*` directories inflate ad hoc `find`/`grep`
  sweeps of the repo (out of scope — this plan is documentation-only, no
  runtime code):** while recomputing the TESTING.md Suite Manifest counts,
  the previously-documented Go count (376) was found to be wildly off from
  the actual working-tree count (`find internal -name '*_test.go' | wc -l` +
  repo-root `*_test.go` = 139). Root cause: three leftover parallel-execution
  worktree checkouts under `.claude/worktrees/agent-*` (~348MB total), each
  containing a full duplicate copy of `internal/` with its own `_test.go`
  files. An unscoped `find . -name '*_test.go'` from the repo root (rather
  than the scoped commands this plan's Task 1 uses) picks those up and
  returns 379 — within rounding of the stale 376, strongly suggesting past
  manual reconciliation passes were unknowingly counting worktree
  duplicates. Not fixed here (this plan's `files_modified` is `TESTING.md`
  only; deleting `.claude/worktrees/*` is a workspace-hygiene action with no
  test/requirement coverage of its own). Recommend a future quick task or
  workspace-cleanup pass to remove `.claude/worktrees/agent-*` once confirmed
  they hold no in-flight work, and to periodically re-verify the Suite
  Manifest counts against the scoped `find` commands (not a bare `find .`)
  to prevent this drift from recurring.
