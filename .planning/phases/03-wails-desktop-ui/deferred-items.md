# Deferred Items — Phase 03

## Pre-existing Flaky Test

**TestHub_SlowClientDisconnected** in `internal/relay/server_test.go`

- **Discovered:** Phase 03-01, Task 2
- **Status:** Pre-existing — also flaky before Phase 03 changes (verified via git stash)
- **Root cause:** Both `normalClient` and `slowClient` have 256-message buffers. The test writes 300 messages without reading from `normalClient` during the write loop. Under load, `normalClient`'s buffer can fill up, causing it to receive `StatusPolicyViolation` before the slow client does.
- **Fix:** Add goroutine to consume normalClient messages during the write flood, OR use a much smaller buffer for slowClient (0 or 1) while giving normalClient a larger buffer.
- **Impact:** Test flakiness only — no runtime behavior affected.
