---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: 03
type: tdd
wave: 1
depends_on: []
files_modified:
  - internal/relay/server_inject_test.go
autonomous: true
requirements: [IN-02]
must_haves:
  truths:
    - "A MsgSessionInject frame carrying control-only text results in zero PTY writes (no spurious Enter)."
  artifacts:
    - internal/relay/server_inject_test.go
  key_links:
    - "Test drives the live read-pump -> HandleInject path so the hub.go:608 TrimSpace guard is exercised end-to-end (not unit-mocked)."
  prohibitions:
    - "MUST NOT modify hub.go or any production code — the IN-02 guard is already correct at hub.go:608; this plan adds COVERAGE only."
    - "MUST NOT add new test helpers — reuse setupInjectTestServer and dialInjectWS already in the file."
---

<objective>
Close the IN-02 tech-debt item from the v4.1 milestone audit: control-only inject text (e.g. a clear-screen escape) sanitizes to whitespace and must NOT press Enter at the PTY. RESEARCH verified the behavioral guard is ALREADY present at hub.go:608 (`strings.TrimSpace(sanitized) == ""`); the gap is missing regression coverage. This plan adds a single Go test — no production change.

Purpose: Lock in the IN-02 guard with an automated regression so a future refactor cannot silently reintroduce the spurious-Enter behavior.

Output: `TestInject_ControlOnlyInput` in internal/relay/server_inject_test.go.

Note: server_inject_test.go is an existing (already-registered) test file; per the repo standing convention, the IN-02 §4 traceability row is added centrally in plan 160-05 (sole TESTING.md owner).
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-RESEARCH.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-PATTERNS.md
@internal/relay/server_inject_test.go
@internal/relay/hub.go
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add TestInject_ControlOnlyInput (IN-02 regression)</name>
  <files>internal/relay/server_inject_test.go</files>
  <read_first>
    - 160-RESEARCH.md lines 128-153 (IN-02 status: code already correct at hub.go:608; test missing) and 338-340 (Pitfall 6: add a TEST, not a code change)
    - 160-PATTERNS.md lines 282-315 (exact test function to add; helper reuse: setupInjectTestServer, dialInjectWS)
    - internal/relay/server_inject_test.go (existing TestInject_RWCap_WritesToPTY, TestInject_OnlyDedicatedFrame, TestInject_ROCap_RelayPath — copy structure and the ptyWriteCount assertion)
    - internal/relay/hub.go lines 587-646 (HandleInject + the TrimSpace guard at ~608 being exercised)
  </read_first>
  <behavior>
    - Given a session with an RW-capable inject connection, sending a MsgSessionInject frame whose Text is a control-only clear-screen escape results in ptyWriteCount == 0.
    - The frame is non-empty at the raw level (passes the read-pump ip.Text != "" guard) but SanitizePTYText collapses it and TrimSpace yields "", triggering the hub.go:608 early-return.
  </behavior>
  <action>
    Add `TestInject_ControlOnlyInput` to server_inject_test.go following the structure in 160-PATTERNS.md lines 286-313. Stand up the server with the existing `setupInjectTestServer(t)` (which returns ptyWriteCount), dial an RW inject WS via `dialInjectWS`, then write a single MsgSessionInject frame whose InjectPayload.Text is a control-only clear-screen escape sequence. After a short settle, assert `ptyWriteCount.Load() == 0` with a message naming the IN-02 guard. Do not add helpers; do not touch production code. Keep the function golangci-lint clean (ctx-aware, gofmt).
  </action>
  <verify>
    <automated>go test -race -run TestInject_ControlOnly ./internal/relay/...</automated>
  </verify>
  <acceptance_criteria>
    `go test -race -run TestInject_ControlOnly ./internal/relay/...` passes; the test asserts ptyWriteCount==0 for control-only inject; no production file changed; gofmt/golangci-lint clean.
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| inject WS client -> PTY | Untrusted inject text crosses into the PTY write path (the IN-02 surface) |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-160-IN-02 | Tampering | HandleInject control-only text | medium | mitigate | Guard already at hub.go:608 (TrimSpace empty -> early return, no PTY write); this plan adds the regression test that pins it. V5 Input Validation. |
</threat_model>

<verification>
- `go test -race -run TestInject_ControlOnly ./internal/relay/...` passes.
- `git diff --stat` shows only server_inject_test.go changed (no production code).
</verification>

<success_criteria>
IN-02 control-only inject behavior is locked by an automated regression proving zero PTY writes; the audit tech-debt item is closeable.
</success_criteria>

<output>
Create `.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-03-SUMMARY.md` when done.
</output>
