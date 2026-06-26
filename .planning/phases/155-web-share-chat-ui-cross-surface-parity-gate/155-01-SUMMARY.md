---
phase: 155-web-share-chat-ui-cross-surface-parity-gate
plan: "01"
subsystem: chat-export
tags: [go, tdd, export, yaml-frontmatter, chat, security]
status: complete

dependency_graph:
  requires: []
  provides:
    - ChatStore.Export() YAML-frontmatter format (EXPORT-01 serializer)
    - relay loopback export route proven by Go test
    - cap-gated webserver export route proven by Go test
  affects:
    - internal/daemon/chat.go
    - internal/daemon/chat_test.go
    - internal/daemon/chat_routes_test.go
    - internal/webserver/chat_test.go
    - TESTING.md

tech_stack:
  added: []
  patterns:
    - TDD RED→GREEN (test written first, implementation follows)
    - YAML frontmatter with HMAC-safe double-quoting for alias values
    - Provider-callback pattern preserves webserver↔daemon import-cycle guard

key_files:
  modified:
    - internal/daemon/chat.go (Export() rewritten)
    - internal/daemon/chat_test.go (TestChatStore_Export added)
    - internal/daemon/chat_routes_test.go (TestChatRoutes_Export updated)
    - internal/webserver/chat_test.go (TestChatExport added)
    - TESTING.md (Section 2 note + Section 4 EXPORT-01 rows)

decisions:
  - Export() always double-quotes participant values (not just aliases with YAML-special chars) — simpler invariant, no conditional logic
  - TestChatRoutes_Export updated in-place (existing function extended) rather than adding a parallel test; single test covers the full route contract
  - TestChatExport uses a hardcoded frontmatter markdown string (no daemon import in webserver tests) — consistent with provider-callback isolation established in Phase 151

metrics:
  duration: "4 minutes"
  completed: "2026-06-26"
  tasks: 3
  files: 5
---

# Phase 155 Plan 01: ChatStore.Export() YAML Frontmatter Summary

Rewrote `ChatStore.Export()` to emit GitHub-compatible Markdown with a YAML frontmatter block (`session`, `exported_at`, `participants`), satisfying EXPORT-01 success criterion 2. Added Go unit coverage for the serializer and both export routes (relay loopback + cap-gated webserver). Registered Go test files in TESTING.md.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 RED | TestChatStore_Export (failing) | 81897bc4 | internal/daemon/chat_test.go |
| 1 GREEN | Rewrite ChatStore.Export() | a503922d | internal/daemon/chat.go |
| 2 | Export-route Go tests | 177143db | internal/daemon/chat_routes_test.go, internal/webserver/chat_test.go |
| 3 | Register Go tests in TESTING.md | e8c156bc | TESTING.md |

## What Was Built

**`ChatStore.Export()` rewrite** (`internal/daemon/chat.go`):
- Emits YAML frontmatter: `---` / `session: {id}` / `exported_at: {RFC3339}` / `participants:` (deduplicated by `AuthorID`, order of first appearance) / `---`
- Each participant value always double-quoted; embedded `"` escaped as `\"` (T-155-02 mitigation)
- Message headers: `## {alias} ({authorID}) — {RFC3339 ts}` (AuthorID now in header, not a `**Author ID:**` block)
- `_injected into terminal_` marker when `SessionInject==true`
- Closing `---` separator after each message block
- Empty thread returns frontmatter-only document + `# Chat: {id}` heading (non-empty, nil error)

**`TestChatStore_Export`** (`internal/daemon/chat_test.go`):
- 5 sub-tests: EmptyThread, SingleMessage, DeduplicatedParticipants, SessionInjectMarker, YAMLSpecialCharInAlias
- Verified RED phase (4/5 sub-tests failed before implementation)
- All 5 sub-tests GREEN after implementation

**`TestChatRoutes_Export`** (`internal/daemon/chat_routes_test.go`):
- Updated to assert body starts with `---\n` (YAML frontmatter)
- Asserts `session:`, `exported_at:`, `participants:` keys present
- Asserts exact `Content-Disposition: attachment; filename="chat-sess-export.md"`

**`TestChatExport`** (`internal/webserver/chat_test.go`):
- Valid cap → 200, frontmatter body contains `session:` and `exported_at:`, attachment Content-Disposition
- Missing cap → 401, response body contains no thread bytes (T-155-03 gate)

**TESTING.md** (Section 2 + Section 4):
- Section 2 Note: Phase 155-01 entry documenting extended files (no count change, 365 Go / 498 total)
- Section 4: Three EXPORT-01 rows (chat_test.go, chat_routes_test.go, webserver/chat_test.go)
- `bash tests/check-traceability-paths.sh` exits 0

## Verification

```
go test ./internal/daemon/... -run TestChatStore_Export -count=1      PASS
go test ./internal/daemon/... -run TestChatRoutes_Export -count=1     PASS
go test ./internal/webserver/... -run TestChatExport -count=1         PASS
go test ./... -short -count=1                                          PASS (all 14 packages)
bash tests/check-traceability-paths.sh                                 exit 0
```

## Deviations from Plan

None — plan executed exactly as written. The TDD RED→GREEN commit sequence was followed. The existing `TestChatExportFields` and `TestChatExportEmpty` tests (from Phase 151-02) continued to pass after the Export() rewrite since they check field presence (not format), and the new format still includes all required fields.

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes introduced. The Export() change is purely a serialization format update to an existing route. T-155-02 (alias YAML injection) is mitigated by always double-quoting participant values. T-155-03 (unauthorized export) is verified by `TestChatExport` (part b).

## Self-Check: PASSED

| Item | Status |
|------|--------|
| internal/daemon/chat.go | FOUND |
| internal/daemon/chat_test.go | FOUND |
| internal/daemon/chat_routes_test.go | FOUND |
| internal/webserver/chat_test.go | FOUND |
| TESTING.md | FOUND |
| 155-01-SUMMARY.md | FOUND |
| Commit 81897bc4 (RED test) | FOUND |
| Commit a503922d (GREEN impl) | FOUND |
| Commit 177143db (route tests) | FOUND |
| Commit e8c156bc (TESTING.md) | FOUND |
| check-traceability-paths.sh | exit 0 |
