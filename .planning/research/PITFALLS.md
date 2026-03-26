# Pitfalls Research

**Domain:** Desktop app — terminal fit fix, daemon performance, CLI arg passthrough (AgentHub v1.5)
**Researched:** 2026-03-25
**Confidence:** HIGH — codebase read directly, pitfalls verified against xterm.js GitHub issues and Go exec docs; Gemini CLI startup issue verified against upstream issues

---

## Critical Pitfalls

### Pitfall 1: FitAddon Called While Container Has Zero Dimensions

**What goes wrong:**
`fitAddon.fit()` is called when the TerminalPanel container is hidden (`display: none`) or not yet laid out. FitAddon calls `proposeDimensions()` internally, which reads `getBoundingClientRect()` on the parent. Hidden elements return zero. The result is a 1-column or 0-row terminal that stays broken until the next manual resize — exactly the bug described in PROJECT.md: "CSS flex chain fixed, fills after resize — initial-paint timing gap remains."

**Why it happens:**
The current `TerminalPanel.tsx` runs `fit()` inside the `isActive` useEffect, which fires when `isActive` becomes true. In Wails/WebView there is a non-zero gap between React setting `display: flex` on the container and the browser completing layout. The ResizeObserver fires immediately on first observation — before the browser has flushed the layout pass — so the first `fit()` call can read zero dimensions.

The existing `document.fonts.ready` guard only protects against font-measurement errors. It does not wait for the layout-flush-after-display-change. Both conditions must be true before `fit()` is safe: fonts loaded AND container has non-zero layout dimensions.

Confirmed upstream: xterm.js issue #3029 ("FitAddon and display 'none'") was closed as "designed" behavior — FitAddon requires the container to have measurable dimensions; it is the caller's responsibility to guard against zero-dimension calls. Issue #5320 documents `width=1` results from layout not being settled.

**How to avoid:**
Wrap every `fit()` call in a `safeFit()` guard that checks `proposeDimensions()` first:
```typescript
function safeFit(fitAddon: FitAddon): void {
  const dims = fitAddon.proposeDimensions()
  if (dims && dims.cols > 0 && dims.rows > 0) {
    fitAddon.fit()
  }
}
```
Additionally, for the initial activation (first time `isActive` becomes true), schedule the fit call inside `requestAnimationFrame` so at least one layout frame has committed before measuring. ResizeObserver fires after layout passes, so subsequent calls do not need the rAF deferral — only the first observation at activation.

**Warning signs:**
- Terminal appears as a thin strip (1–2 rows) or incorrect width on first tab activation.
- Terminal fills correctly after any window resize or browser zoom change.
- The bug reproduces on all CLI types but is most visible with CLIs that print a splash screen immediately (Claude, Gemini) because the broken dimensions are visible before the user has a chance to resize.

**Phase to address:** Phase 1 — Terminal fill fix.

---

### Pitfall 2: WebGL Context Lost on Hidden-Tab Reactivation, No Renderer Fallback

**What goes wrong:**
Browsers budget WebGL contexts per page. When multiple terminals are open and inactive panels are hidden via `display: none`, the browser may silently drop a WebGL context. When that tab is reactivated, the WebGL addon fires `onContextLoss` — the current handler disposes the addon — but does not attach a fallback renderer. The terminal accepts data but renders blank.

**Why it happens:**
The current `TerminalPanel.tsx` disposes the WebGL addon on context loss but takes no further action:
```typescript
webglAddon.onContextLoss(() => {
  webglAddon.dispose()
  // Nothing here — terminal is now renderer-less
})
```
Without an active renderer, xterm.js has no way to paint to the canvas.

**How to avoid:**
After disposing the WebGL addon, explicitly force canvas rendering via xterm.js options. Canvas is the default fallback and performs adequately for CLI output volumes:
```typescript
webglAddon.onContextLoss(() => {
  webglAddon.dispose()
  term.options.allowTransparency = false  // required before canvas mode on some xterm versions
  // xterm.js v5 falls back to canvas automatically after WebGL addon is disposed
  // but calling fit() after ensures dimensions are recalculated for the new renderer
  safeFit(fitAddonRef.current)
})
```

**Warning signs:**
- Terminal goes blank when switching back to a tab that was idle for a long time.
- No JavaScript console errors (context loss is silent by default without the handler).
- Only affects tabs with WebGL renderer — if WebGL failed at creation and canvas was used from the start, this pitfall doesn't trigger.

**Phase to address:** Phase 1 — Terminal fill fix (fix context loss recovery alongside fit fix).

---

### Pitfall 3: Daemon Starts With Minimal System PATH — Agent Resolution Fails or Resolves Wrong Binary

**What goes wrong:**
The daemon is a background service started by launchd/systemd/SCM. Its environment contains only the minimal system PATH (`/usr/bin:/bin:/usr/sbin:/sbin`). When `CreateSession` calls `engine.ResolveCLI("claude")` and falls through to using the name as-is, `exec.LookPath("claude")` or the PTY spawn resolves against the daemon's PATH — not the user's PATH.

For agents installed via npm global (`gemini`), nvm/volta-managed Node paths, or Homebrew (`/opt/homebrew/bin`), the binary is simply not found or the wrong version is used. This is the most likely root cause of the slow-startup regression introduced in v1.3 when sessions moved to daemon mode — the daemon's PATH may miss the binary entirely and fall back to a slow retry path, or the wrong binary (older system-level install) is used.

**Why it happens:**
When sessions were created in-process (pre-v1.3), they inherited the full user shell environment. When moved to the daemon, the daemon process was spawned via `startDetachedDaemon` which inherits only the minimal GUI launch environment — not the shell profile environment. Users with nvm, volta, pyenv, or Homebrew have their binaries in paths that only appear after shell profile scripts run (`~/.zshrc`, `~/.profile`).

Gemini CLI additionally has documented startup regressions of 8–60 seconds caused by synchronous MCP server initialization (GitHub issues #4544, #21853, #17774). These are CLI-side issues unrelated to daemon mode but may be conflated with the daemon regression.

**How to avoid:**
Profile first — add a `time.Now()` delta log around `b.backend.Create()` in `engine.go` to measure actual PTY spawn time separately from agent initialization time. This distinguishes daemon-side from CLI-side latency.

If daemon-side PATH is the issue, two options:
1. **Login shell spawn wrapper:** Spawn agents via a login shell: `cmd = /bin/zsh -l -c "<agent> <args>"`. This adds one shell process but ensures the full user environment including nvm/volta/Homebrew paths.
2. **PATH expansion at daemon startup:** At `runDaemonCore`, expand PATH by reading `/etc/paths`, `/etc/paths.d/*`, and common tool manager paths (`~/.nvm/alias/default` version resolution, `/opt/homebrew/bin`), then set `PATH` in the daemon's environment explicitly. Cache the expanded PATH.

Log the resolved binary path for each session creation so misresolution is diagnosable.

**Warning signs:**
- Session takes 2–5+ seconds to show first output where it previously appeared in under 1 second.
- `agenthub list` shows session stuck in `running` state for several seconds with no terminal output.
- `which gemini` in a user terminal returns `~/.nvm/versions/node/.../bin/gemini` but daemon logs resolve it differently.
- Bug only manifests when daemon is running as a service (`agenthub daemon install`), not when launched manually from a terminal.

**Phase to address:** Phase 2 — Daemon performance fix.

---

### Pitfall 4: CLI Args Word-Splitting on User Input String

**What goes wrong:**
The user types extra args in a text field: `--model claude-opus-4-5 --no-auto-updates`. The frontend sends this as a single string to the Go backend. If the backend splits naively with `strings.Fields()`, arguments with embedded spaces or quotes (`--config "/path/with spaces/config.json"`) are split incorrectly. If the backend passes the raw string as one element to `exec.Command`, the agent receives the entire string as a single token and ignores it. Either failure is silent — no error, the agent just doesn't see the flags.

**Why it happens:**
String splitting looks trivially simple until quoted arguments appear. `strings.Fields()` splits on whitespace only and does not understand POSIX quoting conventions (`"..."`, `'...'`, `\` escaping). Shell-style parsing requires a proper lexer.

**How to avoid:**
Tokenize the args string using a shlex-equivalent before building the `[]string` slice passed to `pty.CreateRequest.Args`. Options:
- `github.com/google/shlex` — MIT licensed, minimal, no dependencies
- `mvdan.cc/sh/v3/syntax` — full POSIX shell parser (heavier, overkill for arg splitting)

Pass the tokenized `[]string` to `exec.Command` / `gopty.CommandContext` — never concatenate back into a shell string. Go's `exec.Command` does not invoke a shell, so array-based passing is injection-safe by construction once the string is correctly tokenized.

Edge cases to test: `--key "value with spaces"`, `--key='value with spaces'`, `--key value`, multiple consecutive spaces, empty string input.

**Warning signs:**
- `--model` flag is ignored when combined with other flags in the same text field.
- Args containing quoted paths with spaces produce "file not found" errors from the agent.
- Test: create a session with `--version` as extra args; verify the agent prints its version and exits (single flag, no quoting complexity).

**Phase to address:** Phase 3 — CLI args passthrough.

---

### Pitfall 5: Args Field Missing From daemon.CreateRequest — Silent Discard at Daemon Boundary

**What goes wrong:**
`pty.CreateRequest` already has `Args []string` and `NativePTYBackend.Create` uses it correctly. However, `daemon.CreateRequest` (the HTTP JSON type in `daemon/types.go`) does not have an `Args` field. If args are wired into the frontend and `app.go` but the daemon type layer is not updated, args are silently dropped at the HTTP serialization boundary — the daemon creates the session without them and returns success.

**Why it happens:**
The call chain has three distinct type layers: `daemon.CreateRequest` (HTTP JSON) → `engine.CreateSession()` parameters → `pty.CreateRequest`. The pty layer is already done. The daemon layer requires surgery at every level:
- `daemon/types.go` — struct definition
- `daemon/api.go` — handler reads `req.Args`
- `daemon/engine.go` — `CreateSession` accepts `args []string` and passes to `pty.CreateRequest{Args: args}`
- `daemon/client.go` — `CreateSession` method accepts and sends `args`
- `app.go` — `CreateSession` Wails method accepts and forwards `args`
- Wails binding regeneration — required after any method signature change

**How to avoid:**
Update all six layers in sequence. Write an integration test that creates a session with `Args: []string{"--version"}` via the daemon API and verifies the spawned process received the flag (check `ps aux` output or capture stderr). A unit test that only checks `daemon.CreateRequest` serialization is not sufficient — it must exercise the full IPC chain.

Also update the `cmd_cli.go` `new` command to accept `--` suffix args: `agenthub new --cli claude --workdir /tmp -- --model claude-opus-4-5`.

**Warning signs:**
- Extra args appear in the UI modal but the agent behaves identically with and without them.
- No error is returned from `CreateSession` — the session is created successfully, just without the args.

**Phase to address:** Phase 3 — CLI args passthrough.

---

### Pitfall 6: Per-Agent Arg Memory Uses Non-Namespaced localStorage Keys

**What goes wrong:**
If per-agent args are stored using a bare key like `"args"` or `"claude-args"`, two problems arise:
1. Key collisions with any other feature that uses similar keys.
2. If an agent is renamed or a new agent with a similar name is added, the wrong defaults are pre-filled.

The existing codebase uses the pattern `'agenthub:lastWorkDir'` (in `NewSessionModal.tsx`). Per-agent keys must follow this namespace pattern consistently.

A secondary risk: stored args persist across app upgrades in the Wails WebView data directory. If a deprecated flag is stored (e.g., `--old-flag` that the agent no longer accepts), every new session silently gets a broken default.

**How to avoid:**
Use the key pattern `agenthub:args:<cliName>` (e.g., `agenthub:args:claude`, `agenthub:args:gemini`). Store args as the raw text field string — not a parsed array. Parsing happens at session creation, not at storage time. Provide a clear button that calls `localStorage.removeItem('agenthub:args:' + cli)` (not just `setState('')`). On load, validate the stored value is a string type; clear it if malformed.

**Warning signs:**
- Switching the selected CLI in the modal pre-fills the wrong agent's saved args.
- Clearing args in the modal shows an empty field but reopening the modal shows the args again.
- Two agents with similar name prefixes ("claude", "claude-opus") share defaults incorrectly.

**Phase to address:** Phase 3 — CLI args passthrough.

---

### Pitfall 7: Wails TypeScript Bindings Stale After Go Method Signature Change

**What goes wrong:**
When `App.CreateSession` is changed to accept an `args string` parameter, Wails's auto-generated TypeScript bindings in `wailsjs/go/main/App.js` and `App.d.ts` must be regenerated. If `wails generate` is not run, the frontend calls the old signature (no `args` parameter). Wails silently ignores extra arguments and omits missing ones at the IPC boundary — no runtime error, just missing data.

**Why it happens:**
Wails generates TypeScript bindings at build time, not automatically on every save. Developers who test with `wails dev` may see the right behavior if dev mode regenerates on restart, but production builds with stale bindings silently drop the new parameter.

**How to avoid:**
Make `wails generate` a required step in the build runbook for any phase that changes Wails-bound method signatures. Do not commit `wailsjs/` directory changes without verifying they match the current Go source. Check TypeScript binding parameters match the Go signature exactly before marking the feature complete.

**Warning signs:**
- Frontend compiles without TypeScript errors but args are not passed to the backend.
- TypeScript call site shows correct parameter count but the Go handler receives zero/empty value.
- `wailsjs/go/main/App.d.ts` shows the old signature after a Go method was changed.

**Phase to address:** Phase 3 — CLI args passthrough (as a build step requirement, not a code change).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `strings.Fields()` for arg splitting | Simple, no dependency | Breaks on quoted paths/spaces silently | Never — use shlex, 3-line change |
| Direct `fit()` without `safeFit()` guard | Less code | Terminal broken on every cold start and hidden panel | Never — guard is 4 lines |
| Hardcoded `80x24` initial PTY dimensions in `engine.go` | Simple | PTY and xterm.js are out of sync until first resize event | Acceptable for v1.5 if fit fix resolves UX; address in v1.6 |
| Storing args string only in localStorage (not settings file) | Simple | Lost if user clears WebView storage or migrates machines | Acceptable for v1.5; persist to settings file in v1.6 |
| Login shell spawn wrapper for daemon PATH fix | Simple workaround | Adds one extra process per session; slower startup | Acceptable as interim fix; replace with proper PATH expansion later |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| xterm.js FitAddon | Call `fit()` synchronously in `useEffect` after `isActive` changes | Defer first call with `requestAnimationFrame`; guard all calls with `safeFit()` that checks `proposeDimensions()` returns non-zero |
| xterm.js WebGL addon | Dispose on context loss but leave terminal without renderer | After dispose, re-fit (triggers canvas fallback) or explicitly set canvas renderer option |
| go-pty `CommandContext` | Pass raw user input string as `req.Args[0]` | Tokenize with shlex-equivalent first; pass `[]string` to `CommandContext(ctx, cli, args...)` |
| Wails TypeScript bindings | Manually edit `wailsjs/go/main/App.d.ts` | Run `wails generate` after any bound method signature change; manual edits are overwritten on next build |
| `daemon.CreateRequest` JSON type | Add args to `pty.CreateRequest` only | Update all three layers: `daemon/types.go` (JSON struct), `daemon/engine.go` (function parameters), `daemon/api.go` (handler) |
| Gemini CLI startup | Attribute all startup slowness to daemon mode regression | Gemini CLI has documented 8–60s startup regressions from MCP initialization; profile to separate daemon-side vs. CLI-side latency before fixing |
| `kardianos/service` launchd plist | Service inherits minimal system PATH | Explicitly expand PATH at daemon startup or in the plist `EnvironmentVariables` key; test with `agenthub daemon install && agenthub daemon start` (not just in-terminal launch) |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Daemon spawned with minimal system PATH | Agent not found or wrong binary version resolves; slow startup | Log resolved binary path on every session creation; expand PATH at daemon startup | First cold start after `agenthub daemon install` on any machine with nvm/volta/Homebrew |
| nvm/volta-managed agents invisible to daemon | Correct binary in user shell, missing or wrong binary in daemon | Use login shell spawn (`/bin/zsh -l -c`) or set PATH in plist `EnvironmentVariables` | Any user with version-manager-managed Node.js |
| `exec.LookPath` called in daemon context without logging | Resolves to system binary silently, not user's preferred version | Log resolved path at startup; compare against expected in tests | Silent — no error, wrong version used |
| FitAddon fit on `ResizeObserver` first observation before layout | Terminal renders at 1col or wrong size until resize | `requestAnimationFrame` deferral on first activation only | Every cold start of the app |
| Gemini CLI MCP startup blocking PTY ready signal | Session appears slow regardless of daemon state | Distinguish via timing logs; this is CLI-side, not daemon-side | Gemini CLI with any MCP server configured |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Splitting user args with shell=true or via `/bin/sh -c` | Shell injection — user inputs `; rm -rf ~` | Always use `exec.Command(cli, args...)` with args as separate `[]string` tokens; never concatenate into a shell string |
| Passing raw unsplit args string as a single `argv[1]` element | No injection risk but args silently ignored | Tokenize with shlex before passing to `exec.Command` |
| Logging user-supplied args to stderr without redaction | Args may contain API keys or tokens (`--api-key sk-...`) | Omit or redact args from daemon log output; use structured logging with explicit field allowlist |
| No validation that tokenized args don't include `--exec` or similar for agent-specific RCE flags | Low risk for Claude/Gemini current versions; risk increases as agents add shell-execution flags | Add a per-agent flag denylist for known RCE-capable flags; review when agents add new flag sets |

Note: Go's `exec.Command` does NOT invoke a shell — passing args as a `[]string` is injection-safe by construction. The only risk is in the tokenization step (shlex parsing) and in agents that accept flags which internally invoke a shell.

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Args text field accepts anything; bad args discovered only after session creation with no visible feedback | Session creates, CLI exits or misbehaves silently, user confused | No pre-validation needed (CLIs validate their own args), but ensure stderr is visible in the terminal so agent-side flag errors are immediately readable |
| "Saved defaults" not communicated to user | User doesn't know pre-fill came from storage; unexpected behavior when switching machines | Show a subtle "Saved" label or visual indicator next to pre-filled args; make clear button prominent |
| Clear button clears UI state but not localStorage | User clears args, closes modal, reopens — args are back | Clear button must call `localStorage.removeItem(key)` AND `setState('')` |
| Terminal fill fix only tested in `wails dev`, not `wails build` | Fix works in development but broken in production binary | Always test terminal fill with a `wails build` production binary; dev mode and production have different asset loading timing |

---

## "Looks Done But Isn't" Checklist

- [ ] **Terminal fit fix:** `fit()` works on first load in production binary (`wails build`) — not just in `wails dev`. Open the app fresh with no prior sessions; no resize needed.
- [ ] **Terminal fit fix:** All CLI types (Claude, Gemini, OpenCode, Codex) show correct dimensions on first activation.
- [ ] **Terminal fit fix:** Switching tabs multiple times with terminals at different font sizes produces no blank or incorrectly-sized terminals.
- [ ] **Context loss recovery:** Open 3+ sessions, let them idle for several minutes, switch tabs — no blank terminals.
- [ ] **Args passthrough:** Spawned agent process has args as separate tokens in `ps aux` / Task Manager — not as a single concatenated string.
- [ ] **Args passthrough:** Multi-word quoted arg `--config "/path with spaces/file"` passes as one `argv` element.
- [ ] **Args passthrough:** `agenthub new --cli claude --workdir /tmp -- --model claude-opus-4-5` CLI path works end-to-end.
- [ ] **Args persistence:** Switching CLI in the modal shows the correct saved args for that CLI (not another agent's args).
- [ ] **Args persistence:** Clear button actually removes the stored value; reopening the modal shows an empty field.
- [ ] **Daemon performance:** Profiling confirms slow path is daemon-side (PATH mismatch) vs. CLI-side (Gemini MCP) before applying a fix.
- [ ] **Daemon performance:** Fix tested with `agenthub daemon install && agenthub daemon start` (service mode), not just `./agenthub daemon` from a terminal.
- [ ] **Wails bindings:** `wails generate` run after `App.CreateSession` signature change; TypeScript types match Go types.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| FitAddon wrong dimensions on first load | LOW | Add `safeFit()` guard + `requestAnimationFrame` deferral; no architecture change; 10-line patch |
| WebGL context loss — blank terminal | LOW | Add canvas fallback after `webglAddon.dispose()`; one line change |
| Args silently dropped (missing daemon type layer) | LOW | Add `Args []string` to `daemon.CreateRequest`; update 6 call sites; run `wails generate` |
| Args word-split incorrectly | LOW | Add shlex dependency; swap `strings.Fields()` call at one location |
| Daemon slow due to PATH mismatch | MEDIUM | Profile first; if PATH, add login shell spawn or PATH expansion at daemon startup |
| Daemon slow due to Gemini MCP startup | NONE (external) | Cannot fix in AgentHub; document known issue; recommend user disable unused MCP servers in Gemini config |
| localStorage key naming collision | LOW | Rename keys to `agenthub:args:<cliName>` pattern; no migration needed (stale keys auto-orphan) |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| FitAddon zero-dimension on initial activation | Phase 1: Terminal fill fix | Open app fresh, create Claude session — fills viewport without resize event |
| WebGL context loss — blank terminal on tab switch | Phase 1: Terminal fill fix | Open 3+ sessions, idle, switch tabs — no blank terminals |
| Daemon PATH mismatch / agent not found in service mode | Phase 2: Daemon performance | Time session creation with `agenthub daemon install && agenthub daemon start`; < 2s for Claude |
| CLI-side startup latency (Gemini MCP) attributed to daemon | Phase 2: Daemon performance | Profile shows daemon-side PTY spawn time separate from agent init time |
| Args word-splitting with quoted paths | Phase 3: CLI args | Create session with `--config "/path with spaces/x"` — single `argv` element in `ps` output |
| Args silently dropped (incomplete type chain) | Phase 3: CLI args | Integration test creates session with `Args: []string{"--version"}`, verifies output contains version string |
| Per-agent args localStorage key collisions | Phase 3: CLI args | Switch between Claude and Gemini in modal; each shows own saved args, not shared |
| Args clear button not persisting the clear | Phase 3: CLI args | Clear args, close modal, reopen — field is empty |
| Wails binding out of sync after signature change | Phase 3: CLI args | `wails generate` run as part of phase build; TypeScript types verified against Go signature |

---

## Sources

- xterm.js FitAddon display:none designed behavior: https://github.com/xtermjs/xterm.js/issues/3029
- xterm.js FitAddon width=1 from unsettled layout: https://github.com/xtermjs/xterm.js/issues/5320
- xterm.js FitAddon incorrect resize (v5.3.0): https://github.com/xtermjs/xterm.js/issues/4841
- Gemini CLI slow startup from synchronous MCP initialization: https://github.com/google-gemini/gemini-cli/issues/4544
- Gemini CLI 20–50s startup regression on Windows: https://github.com/google-gemini/gemini-cli/issues/21853
- Gemini CLI startup slower than Claude: https://github.com/google-gemini/gemini-cli/issues/17774
- nvm shell startup overhead: https://github.com/nvm-sh/nvm/issues/2724
- Go exec.Command argument handling (no shell invocation): https://pkg.go.dev/os/exec
- go-pty library (Args field in Cmd struct): https://pkg.go.dev/github.com/aymanbagabas/go-pty
- localStorage namespace collision: https://medium.com/@emadalam/namespace-localstorage-e2d1d2e68b20
- AgentHub codebase (direct read): `pty/backend.go` (Args field exists in CreateRequest), `pty/native.go` (Args used in CommandContext), `daemon/types.go` (Args field absent from CreateRequest), `daemon/engine.go` (does not pass args to pty), `frontend/src/components/TerminalPanel.tsx` (safeFit not yet present), `frontend/src/components/NewSessionModal.tsx` (no args field)

---
*Pitfalls research for: AgentHub v1.5 — terminal fill fix, daemon performance, CLI args passthrough*
*Researched: 2026-03-25*
