# Phase 31: CLI Arg Passthrough - Research

**Researched:** 2026-03-25
**Domain:** Go CLI argument parsing — `--` separator convention, `os.Args` slicing, `cmdNew` dispatch
**Confidence:** HIGH

## Summary

Phase 30 wired `args []string` through all five daemon layers: `daemon.CreateRequest`, `SessionEngine.CreateSession`, `handleCreateSession`, `DaemonClient.CreateSession`, and `App.CreateSession`. Every layer now accepts `args []string` and passes `nil` from existing callers. The entire backend is ready.

Phase 31 is the final CLI-surface piece: teach `runCLI` in `main.go` to detect the `--` separator in `os.Args`, split the slice at that boundary, and hand the right-hand tokens to `cmdNew` as the `args` parameter. `cmdNew` already calls `client.CreateSession(agent, name, workDir, nil)` — the only change is passing the extracted args slice instead of `nil`.

This is a narrow, low-risk change with exactly two concerns: (1) correct `--` detection in the raw `os.Args` slice before `runCLI` strips the command name, and (2) tests for `cmdNew` with/without the separator.

**Primary recommendation:** Split `os.Args` at `--` in `runCLI` before dispatching. Extract trailing args into a local `extraArgs []string` variable and thread it through to `cmdNew`. Do not use Go's `flag` package for `cmdNew` — it does not support `--` as a passthrough terminator.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ARGS-01 | User can pass extra arguments to an agent via `agenthub new <agent> -- --flag value` | `--` separator detection in `os.Args`; `cmdNew` calls `client.CreateSession` which is already wired (Phase 30) |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os` (stdlib) | stdlib | `os.Args` slice access | The raw args slice is the only reliable source before `flag.Parse` |
| `strings` (stdlib) | stdlib | Already imported in `main.go` | No new import needed |

### Supporting
None — this is a slice-manipulation change in `main.go` and `cmd_cli.go`. No new dependencies.

**Installation:** No new packages required.

## Architecture Patterns

### Where the `--` split happens

The split must happen in `runCLI` before the `switch cmd` block, using the full `args []string` parameter (which is `os.Args[1:]`). The resulting `extraArgs` must be passed down to `cmdNew`.

**Current flow:**
```
os.Args = ["agenthub", "new", "claude", "/path", "--", "--model", "claude-opus-4-5"]
main() → runCLI(os.Args[1:])
runCLI(args):
    cmd = args[0]  → "new"
    cmdArgs = args[1:]  → ["claude", "/path", "--", "--model", "claude-opus-4-5"]
    switch "new": cmdNew(client, cmdArgs, os.Stdout)
cmdNew(client, cmdArgs, out):
    agent = cmdArgs[0]  → "claude"
    workDir = cmdArgs[1]  → "/path"
    // cmdArgs[2:] contains ["--", "--model", "claude-opus-4-5"] — currently ignored
    client.CreateSession(agent, name, workDir, nil)  // nil drops the extra args
```

**Target flow:**
```
runCLI(args):
    // 1. Split at "--" before the switch
    extraArgs := splitAtDashDash(args)  // ["--model", "claude-opus-4-5"] or nil
    args = trimAtDashDash(args)         // ["new", "claude", "/path"]
    cmd = args[0]  → "new"
    cmdArgs = args[1:]  → ["claude", "/path"]
    switch "new": cmdNew(client, cmdArgs, extraArgs, os.Stdout)
cmdNew(client, cmdArgs, extraArgs []string, out):
    agent = cmdArgs[0]  → "claude"
    workDir = cmdArgs[1]  → "/path"
    client.CreateSession(agent, name, workDir, extraArgs)
```

### Pattern: `--` separator detection

The Go standard library `flag` package stops parsing at `--` and puts remaining args in `flag.Args()`, but `cmdNew` does not use `flag.NewFlagSet` — it reads positional args directly. The correct approach is a manual scan of the args slice.

**Example implementation (in `runCLI` or as a helper):**
```go
// splitDashDash partitions args at the first "--" sentinel.
// Returns (before, after) where after is nil if "--" is not present.
// Source: standard Go CLI convention; verified by reading os/exec documentation.
func splitDashDash(args []string) (before, after []string) {
    for i, a := range args {
        if a == "--" {
            return args[:i], args[i+1:]
        }
    }
    return args, nil
}
```

Usage in `runCLI`:
```go
func runCLI(args []string) {
    before, extraArgs := splitDashDash(args)
    // before is ["new", "claude", "/path"] — use for command dispatch
    // extraArgs is ["--model", "claude-opus-4-5"] or nil
    cmd := before[0]
    cmdArgs := before[1:]

    // ... EnsureDaemon, client setup ...

    switch cmd {
    case "new":
        err = cmdNew(client, cmdArgs, extraArgs, os.Stdout)
    // other cases pass nil for extraArgs (only "new" uses them)
    // ...
    }
}
```

### `cmdNew` signature change

```go
// Before (current)
func cmdNew(client *daemon.DaemonClient, args []string, out io.Writer) error

// After
func cmdNew(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error
```

The only change inside `cmdNew` is: `client.CreateSession(agent, name, workDir, extraArgs)` instead of `nil`.

### Callers of `cmdNew` that must be updated

`cmdNew` is called in two places:
1. `main.go:runCLI` (the switch case) — update to pass `extraArgs`
2. `cmd_cli_test.go` — all calls to `cmdNew(client, ...)` must add a `nil` third arg before `out`, or `extraArgs` in the test cases that test the `--` behavior

The test helper `testSetup` in `cmd_cli_test.go` creates a real daemon API — it is the correct foundation for the new tests.

### `--` only applies to `new`

Other commands (`list`, `kill`, `rename`, etc.) ignore `extraArgs`. The `--` split happens once in `runCLI`, so `extraArgs` is cleanly separated before the switch. Non-`new` commands do not receive or use `extraArgs`.

### Anti-Patterns to Avoid

- **Splitting `--` inside `cmdNew`:** The `args` parameter passed to `cmdNew` is `cmdArgs` (already `before[1:]`). If the split happens inside `cmdNew` instead of in `runCLI`, the outer `runCLI` still receives the full unmodified `args` and `flag` parsing for other commands could break. Split once, at the `runCLI` level.
- **Using `flag.FlagSet` for `cmdNew`:** `flag.FlagSet.Parse` stops at the first non-flag argument. `cmdNew` takes positional args (`agent`, `path`) then `--` then flags. A `FlagSet` would treat `claude` as a flag and error. Manual positional parsing is correct here.
- **Shell-splitting the args string:** ARGS-06 (deferred). Phase 31 receives an already-split `[]string` from the OS; no shell parsing is needed or wanted.
- **Passing `extraArgs` to every command:** Only `new` uses extra args. Passing `extraArgs` to `kill`, `list`, etc. would be dead code and add noise. Pass `nil` to all non-`new` cases.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Safe `[]string` nil/empty handling | Nil guards everywhere | Pass `nil` naturally; `gopty` already handles nil args slice | PTY layer already spreads `req.Args...` correctly for nil |
| Shell word-splitting | shlex parser | OS already splits at shell level before `os.Args` | User types `-- --model foo`; OS delivers `["--model","foo"]` |

## Common Pitfalls

### Pitfall 1: `cmdNew` test breakage from signature change
**What goes wrong:** `cmdNew` gains a new parameter (`extraArgs []string`) between `args` and `out`. All existing `cmdNew(client, ...)` calls in `cmd_cli_test.go` fail to compile.
**Why it happens:** Go has no default parameters.
**How to avoid:** Update all existing `cmdNew` calls in `cmd_cli_test.go` to pass `nil` for `extraArgs`. Grep for `cmdNew(` before finalising the change.
**Warning signs:** `go build ./...` fails immediately after signature change.

### Pitfall 2: `--` appearing in wrong position in `os.Args`
**What goes wrong:** `agenthub new claude /path --dir /foo -- --model bar` — the `--dir` flag is currently unused (cmdNew ignores flags), but a future test might accidentally put `--` before the two positional args.
**Why it happens:** Users might put `--` anywhere. The spec says `agenthub new <agent> <path> -- <extra>`.
**How to avoid:** The split-at-`--` in `runCLI` happens on the raw `args` slice before positional extraction. `cmdNew` receives only `before[1:]` (agent, path) and `extraArgs` separately. Position of `--` relative to positional args is preserved correctly.
**Warning signs:** `cmdNew` sees fewer than 2 positional args when `--` appears before `<path>`.

### Pitfall 3: Empty `extraArgs` treated differently from `nil`
**What goes wrong:** `agenthub new claude /path --` (trailing `--` with nothing after) produces `extraArgs = []string{}` (empty non-nil). The downstream PTY receives an empty spread — identical to `nil`.
**Why it happens:** `args[i+1:]` on a slice where `i` is the last index returns `[]string{}`, not `nil`.
**How to avoid:** No special handling needed. `gopty.Cmd(req.CLI, req.Args...)` spreads zero elements for both nil and empty slice. Document and leave as-is.
**Warning signs:** Test explicitly checking for `nil` vs `[]string{}` — use `len(args) == 0` rather than `args == nil`.

### Pitfall 4: `runCLI` `len(before) == 0` panic
**What goes wrong:** `agenthub -- --model foo` (just `--` with no command). `before` is empty, `before[0]` panics.
**Why it happens:** The `--` split runs before the `cmd := before[0]` assignment.
**How to avoid:** Guard with `if len(before) == 0 { usage(); return }` or `if len(before) == 0 { before = args }`. In practice this is a degenerate case — the existing check `if len(os.Args) == 1 || strings.HasPrefix(os.Args[1], "-")` in `main()` means bare `--` (which starts with `-`) routes to GUI, not CLI. Verify this edge case in tests.

## Code Examples

### `splitDashDash` helper

```go
// splitDashDash partitions a command-line args slice at the first "--" element.
// Returns (before, nil) if "--" is not present.
// Returns (before, after) where after may be empty if "--" is the last element.
func splitDashDash(args []string) (before, after []string) {
    for i, a := range args {
        if a == "--" {
            return args[:i], args[i+1:]
        }
    }
    return args, nil
}
```

### Updated `runCLI` dispatch (condensed diff)

```go
func runCLI(args []string) {
    before, extraArgs := splitDashDash(args)
    if len(before) == 0 {
        usage()
        return
    }
    cmd := before[0]
    cmdArgs := before[1:]

    // daemon short-circuit unchanged (uses before/cmd)...

    // switch
    switch cmd {
    case "new":
        err = cmdNew(client, cmdArgs, extraArgs, os.Stdout)
    // all other cases: extraArgs not passed
    case "list":
        err = cmdList(client, cmdArgs, os.Stdout)
    // ...
    }
}
```

### Updated `cmdNew` signature

```go
// cmdNew creates a new session. extraArgs are passed directly to the agent process.
func cmdNew(client *daemon.DaemonClient, args []string, extraArgs []string, out io.Writer) error {
    if len(args) < 2 {
        return fmt.Errorf("usage: agenthub new <agent> <path>")
    }
    agent, workDir := args[0], args[1]
    name := filepath.Base(workDir)
    id, err := client.CreateSession(agent, name, workDir, extraArgs)
    if err != nil {
        return fmt.Errorf("agenthub new: %w", err)
    }
    fmt.Fprintln(out, id)
    return nil
}
```

### Test pattern for `--` separator (using existing `testSetup`)

```go
// TestCmdNew_WithExtraArgs verifies that args after "--" are forwarded via CreateSession.
func TestCmdNew_WithExtraArgs(t *testing.T) {
    client := testSetup(t)
    var buf bytes.Buffer
    err := cmdNew(client, []string{"cat", "/tmp"}, []string{"--model", "opus"}, &buf)
    if err != nil {
        t.Fatalf("cmdNew with extraArgs: %v", err)
    }
    out := strings.TrimSpace(buf.String())
    if len(out) != 32 {
        t.Errorf("expected 32-char hex session ID, got %q", out)
    }
}

// TestCmdNew_NoSeparator verifies that no "--" produces nil extraArgs (backward compat).
func TestCmdNew_NoSeparator(t *testing.T) {
    client := testSetup(t)
    var buf bytes.Buffer
    err := cmdNew(client, []string{"cat", "/tmp"}, nil, &buf)
    if err != nil {
        t.Fatalf("cmdNew without extraArgs: %v", err)
    }
    out := strings.TrimSpace(buf.String())
    if len(out) != 32 {
        t.Errorf("expected 32-char hex session ID, got %q", out)
    }
}
```

### `splitDashDash` unit tests

```go
func TestSplitDashDash(t *testing.T) {
    cases := []struct {
        input  []string
        before []string
        after  []string
    }{
        {[]string{"new", "cat", "/tmp"}, []string{"new", "cat", "/tmp"}, nil},
        {[]string{"new", "cat", "/tmp", "--", "--model", "foo"}, []string{"new", "cat", "/tmp"}, []string{"--model", "foo"}},
        {[]string{"new", "cat", "/tmp", "--"}, []string{"new", "cat", "/tmp"}, []string{}},
        {[]string{"--", "--model", "foo"}, []string{}, []string{"--model", "foo"}},
    }
    for _, tc := range cases {
        b, a := splitDashDash(tc.input)
        // compare b vs tc.before, a vs tc.after
    }
}
```

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test . -run TestCmdNew -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ARGS-01 | `cmdNew` with `extraArgs` returns a session ID | unit | `go test . -run TestCmdNew_WithExtraArgs -v` | No — Wave 0 |
| ARGS-01 | `cmdNew` with `nil` extraArgs still works (no regression) | regression | `go test . -run TestCmdNew_NoSeparator -v` | No — Wave 0 |
| ARGS-01 | `splitDashDash` correctly handles all boundary cases | unit | `go test . -run TestSplitDashDash -v` | No — Wave 0 |
| ARGS-01 | Full suite regression (all existing tests pass after signature change) | regression | `go test ./...` | Existing tests |

### Sampling Rate
- **Per task commit:** `go test . -run "TestCmdNew|TestSplitDashDash" -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd_cli_test.go` — add `TestCmdNew_WithExtraArgs` (non-nil extraArgs, expects session ID)
- [ ] `cmd_cli_test.go` — add `TestCmdNew_NoSeparator` (nil extraArgs, backward compat)
- [ ] `dispatch_test.go` or `cmd_cli_test.go` — add `TestSplitDashDash` (boundary conditions: no `--`, `--` mid-slice, trailing `--`, leading `--`)

*(No new test files required — add functions to existing files)*

## Environment Availability

Step 2.6: SKIPPED — pure Go source code changes with no external dependencies beyond what is already in go.mod.

## Sources

### Primary (HIGH confidence)
- `/Users/ken/dev/agenthub/cmd_cli.go` — current `cmdNew` signature and body confirmed
- `/Users/ken/dev/agenthub/main.go` — `runCLI` dispatch logic confirmed; `os.Args[1:]` is passed raw
- `/Users/ken/dev/agenthub/internal/daemon/client.go` — `CreateSession(cli, name, workDir string, args []string)` confirmed present (Phase 30 complete)
- `/Users/ken/dev/agenthub/internal/daemon/types.go` — `CreateRequest.Args []string` confirmed present
- `/Users/ken/dev/agenthub/cmd_cli_test.go` — `testSetup` helper and existing `TestCmdNew_*` patterns confirmed
- `/Users/ken/dev/agenthub/.planning/phases/30-backend-args-wiring/30-01-SUMMARY.md` — Phase 30 completion confirmed; all callers already pass `nil`

### Secondary (MEDIUM confidence)
- Go language specification: `--` as args terminator is a POSIX/getopt convention; Go's `os.Args` delivers it as a literal `"--"` string element — confirmed by standard library behavior (no lookup needed; this is definitional)

## Metadata

**Confidence breakdown:**
- `--` detection approach: HIGH — manual slice scan; no library ambiguity
- `cmdNew` signature change: HIGH — read source directly; one parameter insertion
- Backward compatibility: HIGH — existing callers already pass `nil`; new nil path unchanged
- Test strategy: HIGH — `testSetup` pattern is proven; `cat` is the existing CLI stand-in

**Research date:** 2026-03-25
**Valid until:** 2026-06-25 (stable — pure Go stdlib patterns, no third-party library decisions)
