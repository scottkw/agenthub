# Project Research Summary

**Project:** AgentHub v1.5 — Terminal Fill Fix, Daemon Startup Performance, CLI Arg Passthrough
**Domain:** Desktop app (Wails/Go + React) — three targeted bug fixes and one quality-of-life feature
**Researched:** 2026-03-25
**Confidence:** HIGH

## Executive Summary

AgentHub v1.5 is a focused maintenance and enhancement release with zero new dependencies. All three feature areas (terminal fill fix, daemon startup performance, CLI argument passthrough) are implementable entirely within the existing technology stack. The PTY layer already supports argument arrays; the terminal fit infrastructure already uses the right mechanisms; the daemon polling logic already has the correct shape — all three just need targeted refinements to eliminate timing races and close propagation gaps.

The recommended approach is a strict build order driven by code dependencies: wire the args data model through the Go backend first (purely additive, all existing callers continue to work), then add CLI parsing, then fix the status polling latency, then add the GUI modal fields, and finally apply the terminal fit timing fix. This order ensures each phase is independently testable and that frontend changes are never blocked by incomplete backend wiring. The most critical architectural insight is that `pty.CreateRequest.Args` already exists and is already forwarded to the PTY process — the entire args feature is a propagation change through six layers above it.

The primary risks are subtle rather than complex: (1) the terminal fit fix requires understanding why `requestAnimationFrame` is the correct deferral and `setTimeout` is not; (2) the daemon performance root cause is actually a 2-second sleep-before-first-poll in `pollSessionStatus` in `app.go`, not the `EnsureDaemon` polling loop in `process.go`; and (3) args silently disappearing at the `daemon.CreateRequest` JSON boundary if that struct is not updated. All three risks are well-understood and have clear, low-cost fixes.

## Key Findings

### Recommended Stack

No new dependencies are required for any v1.5 feature. The fix surfaces are all within existing packages: `frontend/src/components/TerminalPanel.tsx` (fit timing), `app.go` (status polling), `internal/daemon/` (args propagation), `cmd_cli.go` (CLI parsing), and `frontend/src/components/NewSessionModal.tsx` (GUI args field).

**Core technologies in scope:**
- `@xterm/addon-fit` v0.11.0: FitAddon timing fix — wrap `fit()` calls in `requestAnimationFrame` inside the `document.fonts.ready` callback; use double-rAF to guarantee both layout and WebGL canvas first-frame are complete before measuring dimensions
- `github.com/aymanbagabas/go-pty` v0.2.2: `CreateRequest.Args []string` already defined and wired to `CommandContext(ctx, req.CLI, req.Args...)` in `native.go` — no change needed in PTY layer
- Go stdlib `flag.NewFlagSet`: `--` terminator behavior is stable since Go 1.0; split remaining args on `"--"` in `cmdNew` to extract passthrough tokens
- `wailsapp/wails/v2` v2.10.2: `App.CreateSession` Wails binding must gain an `args []string` parameter; `wails dev` or `wails build` auto-regenerates TypeScript bindings

### Expected Features

**Must have (v1.5 table stakes):**
- Terminal fills viewport on first tab activation without requiring a window resize — currently broken for Claude and Gemini which render splash screens before the user can resize
- Status indicator appears within 1 second of session creation — currently delayed 2 full seconds by a sleep-before-first-poll pattern
- `agenthub new -- --flag value` passes extra flags to the agent CLI — currently impossible; `CreateRequest.Args` is always nil at every layer above PTY
- GUI new-session modal accepts and persists per-agent extra args — per-CLI localStorage memory keyed as `agenthub:args:<cliName>`

**Should have (quality):**
- `safeFit()` guard that checks `proposeDimensions()` returns non-zero before calling `fit()` — prevents broken 1-column terminal on hidden panel activation
- WebGL context loss recovery that calls `safeFit()` after disposing the addon — prevents blank terminal on tab reactivation after idle
- Args clear button that persists the clear to localStorage (not just UI state)
- `wails generate` as an explicit build step requirement after `App.CreateSession` signature change

**Defer (v2+):**
- Shell-word splitting (shlex) for quoted arguments in the GUI text field — simple whitespace split is sufficient for v1.5; document the limitation in placeholder text
- Combined `/ready` daemon endpoint (health + relay port in one round-trip) — exponential backoff on the existing two-call pattern resolves startup detection without API changes
- Persist args to a settings file — localStorage is sufficient for v1.5; settings file migration is a v1.6 concern

### Architecture Approach

All v1.5 changes are modifications to existing files — no new files are needed. The architecture is a six-layer call chain from GUI to PTY: `NewSessionModal` → `App.CreateSession` (Wails) → `DaemonClient.CreateSession` (Unix socket) → `daemon.API` (HTTP) → `SessionEngine` → `pty.NativePTYBackend`. Args must be threaded through every layer from the top down; the PTY layer at the bottom already supports them.

**Major components and v1.5 changes:**

1. `internal/daemon/types.go` — add `Args []string` to `daemon.CreateRequest` (JSON wire type); this is the most commonly missed layer and the silent-discard risk
2. `internal/daemon/{engine,api,client}.go` — thread `args []string` through all three; purely additive changes
3. `app.go` — two independent changes: (a) add `args []string` to `App.CreateSession` Wails binding; (b) fix `pollSessionStatus` to check immediately then sleep 500ms (not sleep 2s then check)
4. `cmd_cli.go` — parse `--` separator in `cmdNew`; pass trailing tokens to `CreateSession`
5. `NewSessionModal.tsx` — add args text field, per-agent localStorage memory keyed on `agenthub:args:<cli>`, clear button
6. `TerminalPanel.tsx` — replace direct `fit()` in `document.fonts.ready` callback with double-`requestAnimationFrame` deferral; add `safeFit()` guard

### Critical Pitfalls

1. **FitAddon called while container has zero dimensions** — `document.fonts.ready` only gates font load, not CSS layout. Resolved promises run as microtasks before the browser paints; the container may still report zero dimensions. Wrap every `fit()` call in a `safeFit()` dimension guard, and defer the initial fit call with double-`requestAnimationFrame`. Confirmed upstream in xterm.js issue #3029 (designed behavior — caller's responsibility to guard).

2. **Status polling starts with a 2-second sleep** — The real daemon "startup latency" is not `EnsureDaemon` polling but `pollSessionStatus` in `app.go` sleeping 2 seconds before its first HTTP call. Fix: move `time.Sleep` to the end of the loop, reduce to 500ms. This is the highest-value single-line change in the release.

3. **Args silently dropped at `daemon.CreateRequest` boundary** — `pty.CreateRequest.Args` exists and works; `daemon.CreateRequest` (the JSON HTTP type) does not have `Args`. If only the PTY layer is verified, args appear to work in unit tests but are always nil in production. Update all six layers in sequence; write an integration test that exercises the full IPC chain with a non-empty args slice.

4. **Daemon PATH mismatch when running as a service** — Agents installed via nvm, volta, or Homebrew are invisible to the daemon's minimal system PATH when started by launchd/systemd. Log the resolved binary path on every session creation; if PATH is the cause, use a login shell spawn wrapper (`/bin/zsh -l -c`). Profile before fixing — Gemini CLI has documented 8–60s MCP initialization regressions that are CLI-side, not daemon-side.

5. **Wails TypeScript bindings stale after Go signature change** — Wails silently uses the old signature if `wails generate` is not run after `App.CreateSession` changes. Make `wails generate` a required build step in the phase runbook; verify `wailsjs/go/main/App.d.ts` parameters match the Go source before marking the phase complete.

## Implications for Roadmap

Based on the dependency analysis in ARCHITECTURE.md, the build order is driven by two constraints: (1) backend must be wired before frontend can be tested end-to-end; (2) each phase must leave the codebase in a passing-tests state.

### Phase 1: Backend Args Wiring (Go only)

**Rationale:** The entire args feature depends on this foundation. All changes are additive and backward-compatible — callers that pass no args are equivalent to passing `nil`. No UI changes means no Wails binding regeneration in this phase.
**Delivers:** `daemon.CreateRequest`, `engine.CreateSession`, `api.handleCreateSession`, and `DaemonClient.CreateSession` all accept and forward `args []string`.
**Addresses:** CLI arg passthrough (Go backend half); closes the silent-discard-at-boundary pitfall.
**Avoids:** Pitfall — args dropped at `daemon.CreateRequest` JSON boundary.

### Phase 2: CLI Arg Passthrough

**Rationale:** The CLI path (`cmdNew`) is pure Go with no UI dependencies — testable with existing Go test infrastructure. Depends on Phase 1 backend wiring being in place.
**Delivers:** `agenthub new claude /path -- --model claude-opus-4-5 --verbose` works end-to-end from CLI to PTY.
**Addresses:** CLI-side arg parsing with `flag.NewFlagSet` `--` separator.
**Avoids:** Word-splitting pitfall; uses `[]string` tokens passed to `exec.Command`, never a raw shell string.

### Phase 3: Daemon Startup Latency Fix

**Rationale:** This is an independent change in `app.go` with no dependencies on Phase 1 or 2. It delivers immediately visible UX improvement (status indicator in less than 1 second vs. 2+ seconds). Worth shipping as a standalone fix and can be developed in parallel with Phase 1.
**Delivers:** `pollSessionStatus` polls immediately on first iteration then at 500ms intervals. First status event appears in less than 100ms of session creation.
**Addresses:** Daemon startup performance feature from PROJECT.md.
**Avoids:** Conflating daemon spawn latency (`EnsureDaemon`) with status reporting latency (`pollSessionStatus`).

### Phase 4: GUI Args Field + Wails Binding

**Rationale:** Depends on Phase 1 (backend wiring) and requires Wails binding regeneration. GUI changes are the most manually-tested phase; place after backend is proven.
**Delivers:** New-session modal includes args text field with per-agent localStorage memory and clear button. `App.CreateSession` Wails binding updated with `args []string` parameter.
**Addresses:** GUI half of the CLI arg passthrough feature; per-agent args memory (`agenthub:args:<cli>` localStorage keys).
**Avoids:** Non-namespaced localStorage key collisions; stale Wails TypeScript bindings.

### Phase 5: Terminal Fill Fix

**Rationale:** Self-contained change in `TerminalPanel.tsx`. Placed last because it requires manual testing with a production build (`wails build`) — the behavior differs between `wails dev` and the production binary due to asset loading timing differences.
**Delivers:** Terminal fills the viewport on first activation for all CLIs (Claude, Gemini, OpenCode, Codex) without requiring a window resize. WebGL context loss recovery also addressed.
**Addresses:** Terminal fill fix feature from PROJECT.md.
**Avoids:** FitAddon zero-dimension on hidden panel activation; WebGL context loss with no renderer fallback.

### Phase Ordering Rationale

- Phases 1 and 2 must be sequential (Phase 2 calls Phase 1 APIs).
- Phase 3 is independent and can be developed in parallel with Phase 1 or shipped as a standalone PR.
- Phase 4 depends on Phase 1 (Wails binding calls backend that must have the args param).
- Phase 5 is fully independent of Phases 1-4 and can be developed in parallel or applied at any point.
- The dependency-free ordering (3, 1, 2, 4, 5) matches ascending implementation risk: Phase 3 is the highest-value/lowest-risk change in the entire release.

### Research Flags

Phases with well-documented patterns (no additional research needed):
- **Phase 1:** Additive struct field propagation in Go — standard pattern; no research required.
- **Phase 2:** Go stdlib `flag.NewFlagSet` `--` terminator — documented behavior, verified in STACK.md.
- **Phase 3:** Loop restructuring — trivial refactor; no research required.
- **Phase 4:** React controlled input with localStorage — standard React pattern.
- **Phase 5:** `requestAnimationFrame` deferral for FitAddon — confirmed fix in xterm.js issue #4841; no further research required.

Phases that may need targeted investigation during implementation:
- **Phase 4 (Wails binding regeneration):** Confirm whether `wails dev` auto-regenerates on method signature change vs. requiring explicit `wails generate`. Build tooling question, not a code question.
- **Phase 5 (double-rAF vs. single-rAF):** STACK.md recommends single-rAF; ARCHITECTURE.md recommends double-rAF. The correct choice depends on whether WebGL canvas first-frame timing matters. Test both in a production build; double-rAF is the safer default.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies verified against existing `go.mod` and `package.json`; no new deps; all code paths confirmed by direct file reads |
| Features | HIGH | v1.5 features directly derived from PROJECT.md scope; implementation surfaces confirmed in v1.4 codebase |
| Architecture | HIGH | All integration points verified against v1.4 HEAD; `CreateRequest.Args` existence confirmed in `pty/backend.go:13` and `pty/native.go:41`; `pollSessionStatus` 2s sleep confirmed in `app.go:144` |
| Pitfalls | HIGH | Terminal fit pitfalls confirmed via xterm.js upstream issues; daemon PATH pitfall confirmed by direct daemon spawn analysis; Gemini MCP regression confirmed via upstream issues |

**Overall confidence:** HIGH

### Gaps to Address

- **Single-rAF vs. double-rAF for initial fit:** STACK.md and ARCHITECTURE.md give slightly different recommendations. ARCHITECTURE.md's double-rAF is the safer choice for production (matches xterm.js internal pattern); verify in Phase 5 implementation with a production binary.
- **Daemon PATH mismatch vs. Gemini MCP:** The daemon performance improvement may be partially obscured by Gemini CLI's own 8–60s MCP initialization regression. Profile session creation time with `time.Now()` deltas before and after the Phase 3 fix to distinguish daemon-side from CLI-side latency; communicate this distinction to users if Gemini remains slow.
- **Shlex dependency decision:** Simple whitespace-split is documented as a v1.5 limitation for quoted arguments. If any of the target CLIs (Claude Code, Gemini, OpenCode, Codex) require quoted flag values in practice before v1.6, `github.com/google/shlex` can be added at any point — it has no transitive dependencies and is MIT licensed.
- **Wails binding regeneration flow:** Confirm whether `wails dev` watches for Go method signature changes and regenerates TypeScript bindings automatically, or whether `wails generate` must be run explicitly. This affects Phase 4 validation procedure.

## Sources

### Primary (HIGH confidence)

- `/Users/ken/dev/agenthub/internal/pty/backend.go` — `CreateRequest.Args []string` already defined
- `/Users/ken/dev/agenthub/internal/pty/native.go` — `p.CommandContext(childCtx, req.CLI, req.Args...)` confirms args are forwarded
- `/Users/ken/dev/agenthub/app.go:144` — `time.Sleep(2 * time.Second)` before first poll is the status latency source
- `/Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx` — `document.fonts.ready.then(() => fit())` is the fit timing issue
- `/Users/ken/dev/agenthub/frontend/src/components/NewSessionModal.tsx` — `LAST_DIR_KEY` localStorage pattern to extend for per-agent args
- `pkg.go.dev/flag` — `--` terminator behavior for stdlib flag package; stable since Go 1.0

### Secondary (MEDIUM confidence)

- xterm.js issue #4841 — FitAddon resizes incorrectly; `requestAnimationFrame` confirmed as correct fix by maintainer (Tyriar)
- xterm.js issue #5320 — `width=1` result from CSS layout race; confirms zero-dimension pitfall
- xterm.js issue #5298 — layout timing patterns; confirms ResizeObserver fires after layout
- xterm.js issue #3029 — FitAddon with `display:none` is designed behavior; caller's responsibility to guard

### Tertiary (confirmed upstream issues, external fix not in-scope)

- Gemini CLI issue #4544 — synchronous MCP server initialization causing 8–60s startup regression; not fixable in AgentHub
- Gemini CLI issues #21853, #17774 — startup regressions on Windows and vs. Claude

---
*Research completed: 2026-03-25*
*Ready for roadmap: yes*
