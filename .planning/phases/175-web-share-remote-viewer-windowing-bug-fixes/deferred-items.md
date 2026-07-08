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
