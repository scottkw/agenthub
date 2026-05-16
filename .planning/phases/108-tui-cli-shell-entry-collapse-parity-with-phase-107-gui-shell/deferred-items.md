# Phase 108 — Deferred items (out of plan scope)

Items discovered during execution that are out of plan scope per the SCOPE
BOUNDARY rule in the executor protocol. Logged here for visibility; NOT
fixed by Phase 108 plans.

## Pre-existing internal/daemon test failures (unrelated to Phase 108)

Discovered during Plan 108-02 final verification (`go test ./...`). Confirmed
pre-existing at commit `e8adc15` (the Phase 108 SPEC commit, before any
108-* plan touched code). Failures all in `internal/daemon/`, all related
to the `GetShellWebShareWarned` default value flipping from `false` to
`true` somewhere on the codebase trunk. Phase 108 scope explicitly
forbids modifying `internal/daemon/` (SPEC §Out-of-scope), so these are
left untouched.

| Test | File | Symptom |
|------|------|---------|
| `TestAPIGetShellWebShareWarned_Default` | `internal/daemon/api_test.go:1592` | default value: got true, want false |
| `TestDaemonClient_GetSetShellWebShareWarned_RoundTrip` | `internal/daemon/api_test.go:1638` | initial value: got true, want false |
| `TestSetShellWebShareWarned_Default` | `internal/daemon/engine_test.go:905` | default GetShellWebShareWarned: got true, want false |

Reproduction:
```bash
cd /Users/ken/dev/agenthub && git checkout e8adc15 -- internal/daemon/
go test -count=1 -run 'TestAPIGetShellWebShareWarned_Default|TestSetShellWebShareWarned_Default' ./internal/daemon/
# → FAIL on all three before any Phase 108 changes were committed.
```

Follow-up: file a separate phase or hotfix to investigate whether the
default flipped intentionally (acknowledgment-flag default change) or
whether the tests are stale. Out of scope for Phase 108.

## Plan 108-02 follow-up: SetShellPathForTest export

`TestCmdNewShell_InvalidShellPathSilentFallback` (PARITY-CLI-03
acceptance) is skipped at the harness level because both `client.SetShellPath`
and `engine.SetShellPath` validate path existence on the write path
(Phase 107 SHELL-11 hardening), and the unexported `e.shellPath` field
is not reachable from the `main` package test. Plan 108-02 scope
explicitly forbids modifying `internal/daemon/engine.go`, so no bypass
was added.

Proposed follow-up: add an unexported-field test setter in
`internal/daemon/engine.go` mirroring the existing
`(*API).SetWebServerForTest` pattern at `api.go:209-212`:

```go
// SetShellPathForTest directly assigns the shellPath override without
// the executable-existence validation that SetShellPath performs. Used
// only by tests that need to exercise the daemon's behavior when
// shellPath points to a nonexistent or non-executable binary.
func (e *SessionEngine) SetShellPathForTest(path string) {
    e.mu.Lock()
    e.shellPath = path
    e.mu.Unlock()
}
```

Once exposed, the skipped test can be unblocked:

```go
client := testSetup(t)
// Bypass SetShellPath validation to install a deliberately-broken path.
// (Engine handle exposure pattern would also need to be added to
// testSetup — return the engine alongside the client.)
engine.SetShellPathForTest("/nonexistent/shell-binary")
var buf bytes.Buffer
var callErr error
stderr := captureStderr(t, func() {
    callErr = cmdNewShell(client, nil, nil, &buf)
})
if callErr != nil { t.Fatalf("CLI must exit 0; got %v", callErr) }
if stderr != "" { t.Errorf("CLI must emit zero stderr; got %q", stderr) }
```

## Local-environment: `security-review/` directory shadows `./...`

`security-review/` is gitignored and contains Go files from an
out-of-tree security review (`internal_relay_protocol_fuzz_test.go`,
`internal_webserver_server_test.go`). When present, `go list ./...` and
`go test ./...` fail with "found packages relay (...) and webserver
(...)" because Go's wildcard expansion ignores `.gitignore`.

Workaround used during 108-02 verification:
```bash
mv security-review /tmp/agenthub-security-review-108-02-backup
# run verification
mv /tmp/agenthub-security-review-108-02-backup security-review
```

Local-only — not a regression introduced by Phase 108. If this trips
future agents, consider adding a `security-review/.goignore` or
relocating those files outside the module root.
