# Phase 114: CI test stability — webserver capability test flake - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning

<domain>
## Phase Boundary

`TestPluginConfigStream_ExpiredCap_Returns401` passes deterministically (100/100 runs) on Linux CI under `-race -shuffle=on`, returning 401 (not 403), with the root cause identified and fixed (NOT papered over). Closes GitHub Issue #58.

</domain>

<decisions>
## Implementation Decisions

### Investigation-first (LOCKED)
- **Root cause MUST be identified in writing before any fix lands.** No `t.Skip`, no retry loop, no `-shuffle=off` workaround.
- **Likely candidates per Issue #58 triage:**
  1. **Test-state pollution across `internal/webserver` tests** — `testServer` / `EnableSession` / `SetSigningKey` leaking between tests under specific shuffle orderings.
  2. **Base64 strict-decode variance** — capability token decoding.
  3. **HMAC implementation** — signing key handling under concurrent access.
- **Hypothesis 1 (test-state pollution) is leading** per Issue #58 + standard flake patterns.

### Approach
1. **Investigation phase:** reproduce the flake locally; identify root cause; document in writing (commit message + plan SUMMARY).
2. **Fix phase:** address the root cause — e.g. move `testServer` resets to `t.Cleanup`, isolate `EnableSession` state per test, etc.
3. **Verification:** `go test -race -shuffle=on -count=100 ./internal/webserver/` returns 100/100 on Linux CI. macOS CI continues to pass.

### Out of scope
- Other flaky tests (file separately).
- Refactoring webserver test infrastructure beyond what the fix needs.

### Cross-surface verification
- N/A user surface (CI-only fix).
- Validation: Linux CI green across 100 consecutive runs at merge commit. macOS CI continues to pass.

### macOS executor caveat
- Test passes deterministically on macOS in author's experience (per ROADMAP Phase 114). Reproduction may require Linux. If we can't repro locally, document the root cause from code analysis + apply the fix + verify via cross-compile + careful code review.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/webserver/plugin_config_stream_test.go::TestPluginConfigStream_ExpiredCap_Returns401`
- `internal/webserver/` test helpers — `testServer`, `EnableSession`, `SetSigningKey`, `testServerWithHub` (used by Phase 111).

### Established Patterns
- `t.Cleanup` for per-test teardown (idiomatic Go).
- Test-helper isolation patterns elsewhere in the codebase.

</code_context>

<specifics>
## Specific Ideas

- Issue #58 repro: `go test -race -shuffle=on -count=200 ./internal/webserver/` on Linux — observe at least one 403 result.
- 403 vs 401: capability auth flow has two failure modes. 403 = capability mismatch (signing key wrong, payload mutated). 401 = expired cap (genuine "expired" path). Test expects 401; sometimes gets 403 — strongly suggests cross-test state pollution rotating the signing key.

</specifics>

<deferred>
## Deferred Ideas

None.

</deferred>
