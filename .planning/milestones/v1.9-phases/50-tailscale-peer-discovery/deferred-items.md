# Deferred Items - Phase 50

## Pre-existing Issues (Out of Scope)

### Flaky Test: TestHub_SlowClientDisconnected
- **File:** `internal/relay/server_test.go:244`
- **Symptom:** Fails intermittently (~1/3 runs) with: `normalClient stopped receiving at frame N: failed to get reader: received close frame: status = StatusPolicyViolation and reason = "too slow"`
- **Cause:** Timing-sensitive test affected by system load; unrelated to tailnet package
- **Discovered during:** Plan 50-01 full test suite run
- **Action:** Log only — do NOT fix in this phase
