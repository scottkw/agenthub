---
phase: 107
plan: "107-02"
type: execute
status: pending
wave: 0
depends_on: []
requirements: [SHELL-12]
files_modified:
  - internal/daemon/engine.go
  - internal/daemon/engine_test.go
autonomous: true
must_haves:
  truths:
    - "A shell session whose PTY exit code is -1 (natural EOF) is normalized to 0 BEFORE the onExit callback fires (currently the natural-exit path normalizes; this plan extends normalization into the ListSessions ExitCode emission path AND the onExit emission path)."
    - "After a shell session exits cleanly (exit-code 0 OR -1-normalized-to-0), ListSessions returns `state: \"stopped\"` for that session (not \"running\")."
    - "After a shell session exits cleanly, the session's ExitCode field in ListSessions reports 0 (never -1)."
    - "Non-zero exit codes are preserved verbatim (no normalization)."
    - "SHELL-09 status.Watch bypass for shell sessions is NOT regressed — sessions still fall through to the conservative \"running\" status default while alive."
  artifacts:
    - path: internal/daemon/engine.go
      provides: "Normalized exit-code propagation in onExit callback + ListSessions ExitCode field"
      contains: "exitCode == -1"
  key_links:
    - from: engine.go natural-exit goroutine (~L323-341)
      to: onExit callback receivers (api.go web-serve grace period + GUI session:exit event)
      via: "onExit(id, exitCode) with exitCode already normalized"
    - from: engine.go ListSessions (~L377-387) ExitCode read
      to: pty.Session.ExitCode()
      via: "if ec == -1 { ec = 0 } guard before assignment to exitCodePtr"
---

<objective>
SHELL-12 backend: normalize PTY exit code `-1 → 0` everywhere the daemon emits it (both the onExit callback path AND ListSessions ExitCode field), so consumers never see -1 for a naturally-exited session. Paired with 107-04 (frontend exit-code branching) in wave 1.

Purpose: First-user test reported "exited with error" toast firing on natural shell exits (typing `exit` in zsh). Inspection found the natural-exit goroutine already normalizes -1→0 BEFORE calling onExit (engine.go:333-336), but the ListSessions emission path at line 383 reads `s.ExitCode()` directly without normalization. The bug surfaces when the GUI calls ListSessions to render session state — the daemon reports exitCode=-1 there, which the frontend interprets as "non-zero, show toast". This plan closes the gap.

Output: One small but precise behavioral fix in engine.go. State assertions: clean exit → state=stopped + exitCode=0; non-zero exit → state=stopped + exitCode preserved. Independent of 107-01 (different file regions, different concerns). Wave 0.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-CONTEXT.md

@internal/daemon/engine.go

<interfaces>
From internal/daemon/engine.go (~L323-341) — the natural-exit goroutine that ALREADY normalizes correctly. No change needed here; this is the reference for the correct pattern:

```go
go func() {
    <-hub.Done()
    if sess.IsKilled() { return }
    sess.CancelContext()
    time.Sleep(100 * time.Millisecond)
    exitCode := sess.ExitCode()
    if exitCode == -1 {
        exitCode = 0 // conservative default per D-10
    }
    sess.SetState(pty.StateStopped)
    if onExit != nil {
        onExit(id, exitCode)
    }
}()
```

From internal/daemon/engine.go (~L377-387) — the ListSessions branch that MISSES the normalization. THIS is what we are fixing:

```go
var exitCodePtr *int
var durationPtr *int
if state == "stopped" && !s.IsKilled() {
    // Only read exit code for natural exits.
    ec := s.ExitCode()
    exitCodePtr = &ec
    dur := int(time.Since(s.CreatedAt).Seconds())
    durationPtr = &dur
}
```

The fix: insert `if ec == -1 { ec = 0 }` between `ec := s.ExitCode()` and `exitCodePtr = &ec`. This is line-for-line the same conservative normalization the natural-exit goroutine applies. Per CONTEXT.md the bug location is "internal/daemon/engine.go:377-398 reads s.ExitCode() directly without that normalization. Apply the same `if ec == -1 { ec = 0 }` guard at line 383-384."

Note re: SHELL-12 §"state to stopped on clean exit" — re-reading the natural-exit goroutine confirms `sess.SetState(pty.StateStopped)` already fires unconditionally before onExit is called (line 337). The "state stays running" symptom in the user screenshot was a derived consequence of ListSessions returning exitCode=-1, which the FRONTEND uses to decide whether to render the toast over a still-attached terminal — not a backend state bug. No change to the SetState call is needed. We verify this with a regression test asserting state=="stopped" after a 0-exit and after a -1-normalized exit.
</interfaces>

</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Normalize -1→0 in ListSessions ExitCode emission + regression tests</name>
  <files>internal/daemon/engine.go, internal/daemon/engine_test.go</files>
  <behavior>
    - After a shell session exits naturally with PTY-reported -1, ListSessions returns ExitCode=0 (not -1).
    - After a shell session exits naturally with PTY-reported 0, ListSessions returns ExitCode=0 (unchanged, regression coverage).
    - After a shell session exits with non-zero (e.g., 2), ListSessions returns ExitCode=2 (preserved, regression coverage).
    - After any natural exit (regardless of code), ListSessions returns state="stopped".
    - Killed sessions still skip exit-code emission (existing `!s.IsKilled()` guard preserved).
    - onExit callback continues to receive normalized exitCode (existing behavior unchanged; regression coverage proves it).
  </behavior>
  <action>
    Open internal/daemon/engine.go. In the ListSessions for-loop at the natural-exit emission block (~L379-386), insert the normalization guard:

    Before (current):
    ```
    if state == "stopped" && !s.IsKilled() {
        ec := s.ExitCode()
        exitCodePtr = &ec
        dur := int(time.Since(s.CreatedAt).Seconds())
        durationPtr = &dur
    }
    ```

    After (with fix):
    ```
    if state == "stopped" && !s.IsKilled() {
        ec := s.ExitCode()
        if ec == -1 {
            ec = 0 // SHELL-12: mirror the natural-exit goroutine's -1→0 normalization so GUI consumers never see PTY-EOF as an error code.
        }
        exitCodePtr = &ec
        dur := int(time.Since(s.CreatedAt).Seconds())
        durationPtr = &dur
    }
    ```

    Then write tests in internal/daemon/engine_test.go. Use the existing shell-spawn helper pattern from TestCreateSession_ShellArgv_Interactive (~L581) and TestCreateSession_ShellSkipsStatusWatch (~L700) — both already wire a shell session through CreateSession and observe the natural-exit path. The new tests:

      * TestListSessions_NaturalExit_NormalizesNegativeOneToZero — Create a shell session that exits with EOF (close stdin or use `printf 'exit\n' | shell`). Wait for hub.Done. Call ListSessions. Assert: len==1, state=="stopped", ExitCode != nil, *ExitCode == 0. The fact that PTY reports -1 vs 0 is platform-dependent; the test asserts the post-normalization invariant either way.
      * TestListSessions_NaturalExit_PreservesNonZero — Use a shell command that exits non-zero (e.g., spawn `sh -c "exit 2"` via the regular CLI path, NOT the shell-session path, because shell-session argv is hardcoded `-i`). If non-zero shells are hard to fixture deterministically, use a non-shell CLI (the existing `claude` mock pattern from TestCreateSession_OpenCodeEnv) configured to exit 2. Assert *ExitCode == 2.
      * TestListSessions_OnExitCallback_ReceivesNormalized — Pass a captured onExit callback to CreateSession; assert the callback's `code` argument is 0 for natural-EOF shell exit. (This is regression coverage — the natural-exit goroutine ALREADY does this, but the test prevents future refactors from breaking the contract that we are now also relying on in ListSessions.)
      * TestListSessions_State_StoppedAfterNaturalExit — Spawn shell, force EOF, assert ListSessions reports state=="stopped" within a 2-second wait window (poll every 50ms; existing pattern in TestCreateSession_ShellSkipsStatusWatch).
      * TestListSessions_KilledSession_ExitCodeNil — Existing-behavior regression: kill a session via KillSession, assert ExitCode field is nil (the `!s.IsKilled()` guard still works).

    Place tests near the existing shell-session tests (~L700) for locality. Use the same fixture/temp-dir + cleanup pattern.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/daemon/ -run 'TestListSessions_NaturalExit|TestListSessions_OnExitCallback|TestListSessions_State_StoppedAfterNaturalExit|TestListSessions_KilledSession' -v -count=1 -timeout 30s</automated>
  </verify>
  <done>
    Five tests pass. Existing daemon suite (`go test ./internal/daemon/ -count=1`) green — confirms no regression of SHELL-01..09 or PLUG / SET tests. `grep -c "ec == -1" internal/daemon/engine.go` returns exactly 2 (one in natural-exit goroutine, one new in ListSessions).
  </done>
</task>

</tasks>

<verification>
- `go test ./internal/daemon/ -count=1 -timeout 60s` — full daemon suite green.
- `grep -n "ec == -1\|exitCode == -1" /Users/ken/dev/agenthub/internal/daemon/engine.go` — exactly 2 sites (line ~334 and the new one ~L383). Filter comments: `grep -v '^[[:space:]]*//' internal/daemon/engine.go | grep -c "== -1"` returns 2.
- Manual: with daemon running, spawn a shell session via the GUI, type `exit`, and confirm the daemon log shows `exitCode=0` for the session:exit event (not -1). The 107-04 frontend plan will then close the loop.
</verification>

<success_criteria>
- The exact bug from CONTEXT.md §SHELL-12 ("the ExitToast emission path at engine.go:377-398 reads s.ExitCode() directly without that normalization") is closed.
- "Critical invariant: Exit-code 0 ≠ exited with error" is satisfied at the daemon boundary; the frontend (107-04) can now trust that exitCode in the session:exit event payload is the user-meaningful exit code.
- No regression of SHELL-09 status.Watch bypass — shells still report state="running" while alive, "stopped" after exit. Tested explicitly via TestListSessions_State_StoppedAfterNaturalExit.
</success_criteria>

<output>
After completion, create `.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-02-SUMMARY.md` covering: -1→0 normalization at the ListSessions emission site, the 5 regression tests, and confirmation that the natural-exit goroutine's existing normalization still fires for the onExit callback path. Note for 107-04 executor: the daemon now guarantees exitCode is the user-meaningful value (0 for clean, non-zero for errors); the frontend can branch purely on `data.exitCode === 0`.
</output>
