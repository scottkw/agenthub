# Phase 71: OpenCode Theming Fix - Research

**Researched:** 2026-04-13
**Domain:** OpenCode TUI theming system (Bubble Tea/Lip Gloss) + xterm.js ANSI color remapping + PTY env injection
**Confidence:** HIGH

## Summary

OpenCode ignores the globally selected xterm.js theme because it renders using 24-bit true color (RGB) escape sequences (`\033[38;2;R;G;Bm`) from its own built-in theme system, rather than using ANSI palette indices (0-15) that xterm.js can remap. Claude Code, Codex, and Gemini CLI all use ANSI escape codes that reference palette indices, so xterm.js theme changes work for them automatically.

The fix is to force OpenCode to use its built-in `system` theme, which uses ANSI colors 0-15 for syntax highlighting and `"none"` for text/background (inheriting terminal defaults). When the `system` theme is active, OpenCode's output uses the same ANSI palette indices that xterm.js controls, making xterm.js theme selection effective for OpenCode sessions just like the other three agents.

The implementation requires: (1) a managed `tui.json` file with `{"theme": "system"}` written to AgentHub's config directory, and (2) injecting `OPENCODE_TUI_CONFIG=/path/to/tui.json` into the PTY environment when spawning OpenCode sessions. The existing `CreateRequest.Env` field and `mergeEnv()` function in `native.go` support this without architectural changes. No frontend changes are needed -- the existing xterm.js theme propagation (`useEffect([theme])` with `clearTextureAtlas()` + `refresh()`) already handles live repainting of ANSI-palette text.

**Primary recommendation:** Write a static `~/.config/agenthub/opencode-tui.json` at engine startup. In `engine.CreateSession()`, detect `cli == "opencode"` and add `OPENCODE_TUI_CONFIG=<path>` to `CreateRequest.Env`. The system theme makes OpenCode respect xterm.js palette colors, matching Claude Code/Codex/Gemini CLI behavior.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| THM-05 | The theme selected in Settings > Appearance is applied to OpenCode terminal sessions, matching the behavior for Claude Code, Codex, and Gemini CLI sessions | Force OpenCode's `system` theme via `OPENCODE_TUI_CONFIG` env var, which makes OpenCode use ANSI 0-15 palette colors that xterm.js controls via the theme object |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/xterm` | 6.0.0 | Terminal emulator with theme palette remapping | Already installed; `options.theme` setter remaps ANSI 0-15 colors [VERIFIED: frontend/package.json] |
| `xterm-theme` | 1.1.0 | 246 theme palettes with ANSI 0-15 color definitions | Already installed; provides ITheme objects with all 16 ANSI colors [VERIFIED: frontend/package.json] |
| `go-pty` | (current) | PTY backend with env injection | Already used; `CreateRequest.Env` + `mergeEnv()` support per-session env vars [VERIFIED: internal/pty/native.go] |

### Supporting

None. No additional libraries needed.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `OPENCODE_TUI_CONFIG` env var pointing to managed file | Write to `~/.config/opencode/tui.json` directly | Invasive -- modifies user's personal OpenCode config; breaks if user has custom settings |
| Force `system` theme via env var | Custom OpenCode theme JSON with ANSI colors matching AgentHub theme | Massive complexity; would need to generate 246 OpenCode theme JSONs to match each xterm-theme palette |
| Backend env injection | Frontend-side approach (sending theme data over PTY) | Not feasible; OpenCode reads theme config at startup, not from stdin |
| Single managed tui.json | Generate per-session temp files | Unnecessary complexity; all sessions need the same `system` theme |

## Architecture Patterns

### Recommended Project Structure

Changes touch:

```
internal/
  daemon/
    engine.go          # Inject OPENCODE_TUI_CONFIG env var for opencode sessions
    engine_test.go     # Test env var injection for opencode CLI
  pty/
    native.go          # No changes (mergeEnv already handles req.Env)
app.go                 # Write managed opencode-tui.json on startup (or in configDir helper)
```

No frontend changes needed. The existing theme propagation pipeline (Phase 65) works correctly once OpenCode outputs ANSI palette colors instead of 24-bit RGB.

### Pattern 1: Per-Agent Environment Injection

**What:** When creating a session, check the CLI name and inject agent-specific environment variables into `CreateRequest.Env`. The existing `mergeEnv()` in `native.go` handles the merge with inherited env and required vars (`TERM`, `COLORTERM`).

**When to use:** Any per-agent configuration that must be set via environment variables at process start.

**Example:**
```go
// In engine.go CreateSession(), before calling backend.Create():
var env []string
if cli == "opencode" {
    tuiConfigPath := filepath.Join(configDir(), "opencode-tui.json")
    env = append(env, "OPENCODE_TUI_CONFIG="+tuiConfigPath)
}

sess, err := e.backend.Create(ctx, pty.CreateRequest{
    CLI:     cliPath,
    Args:    args,
    Env:     env,
    Cols:    cols,
    Rows:    rows,
    WorkDir: workDir,
})
```
[VERIFIED: CreateRequest.Env field exists in internal/pty/backend.go; mergeEnv() processes it in internal/pty/native.go line 46]

### Pattern 2: Managed Config File

**What:** Write a static JSON config file to AgentHub's config directory at startup. The file content is fixed (`{"theme": "system"}`) and does not change at runtime.

**When to use:** When an external tool needs a config file that AgentHub controls.

**Example:**
```go
// Write managed OpenCode TUI config that forces the "system" theme.
// The system theme uses ANSI palette colors (0-15), making OpenCode
// respect the xterm.js theme selected in AgentHub settings.
func ensureOpenCodeTUIConfig() string {
    dir := configDir() // ~/.config/agenthub/
    path := filepath.Join(dir, "opencode-tui.json")
    content := []byte(`{"$schema":"https://opencode.ai/tui.json","theme":"system"}` + "\n")
    _ = os.WriteFile(path, content, 0644)
    return path
}
```
[VERIFIED: configDir() exists in app.go line 316, returns ~/.config/agenthub/ and creates it if needed]

### Pattern 3: CLI Name Detection

**What:** The `engine.CreateSession()` receives the CLI name (e.g., "opencode", "claude") as a plain string. Use exact string matching to apply per-agent configuration.

**When to use:** Agent-specific behavior in session creation.

**Example:**
```go
// Existing pattern in codebase: PatternsForCLI() in internal/status/detector.go
// uses cli name string matching for per-agent behavior.
if cliName == "claude" {
    return DefaultClaudePatterns()
}
```
[VERIFIED: internal/status/detector.go line 86]

### Anti-Patterns to Avoid

- **Modifying the user's OpenCode config (`~/.config/opencode/tui.json`):** AgentHub must not touch the user's personal OpenCode configuration. Use a separate managed file and `OPENCODE_TUI_CONFIG` env var.
- **Generating per-session OpenCode theme JSONs:** Each xterm-theme has 16 ANSI colors; mapping them to OpenCode's 62-color theme format would be fragile and unmaintainable. The `system` theme solves this generically.
- **Frontend-only fix (e.g., filtering escape codes):** The problem is at the CLI output level. Intercepting and rewriting 24-bit color escape codes in the PTY stream is fragile and error-prone.
- **Restarting OpenCode sessions on theme change:** Not needed. Once OpenCode uses ANSI palette colors, xterm.js repaints them on theme change via the existing `clearTextureAtlas()` + `refresh()` mechanism.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mapping xterm-theme palettes to OpenCode theme format | Custom theme generator that converts 246 xterm themes to OpenCode JSON format | OpenCode's built-in `system` theme | The system theme generically adapts to any terminal palette; no per-theme mapping needed |
| PTY escape code rewriting | ANSI parser that converts 24-bit RGB to ANSI palette indices | `OPENCODE_TUI_CONFIG` env var with `system` theme | Escape code rewriting is extremely fragile; the env var approach is clean and supported |
| Config file management | Custom config file lifecycle management | Simple `os.WriteFile` in existing `configDir()` | File content is static; no complex lifecycle needed |

**Key insight:** The `system` theme is the single leverage point that converts OpenCode from "uses hardcoded 24-bit RGB" to "uses ANSI palette colors." All the complexity of mapping 246 xterm themes to OpenCode's format is eliminated.

## Common Pitfalls

### Pitfall 1: OpenCode's `/theme` command overrides managed config

**What goes wrong:** If the user types `/theme` inside an OpenCode session in AgentHub and selects a different theme, OpenCode may cache that preference and override our `OPENCODE_TUI_CONFIG` on subsequent launches.
**Why it happens:** OpenCode stores the last-used theme in `~/.local/share/opencode/` which may take precedence over config files.
**How to avoid:** The `OPENCODE_TUI_CONFIG` env var should take precedence as it's an explicit environment override. In v1.4.0, this was confirmed working (the earlier precedence bug was in older versions). [CITED: github.com/anomalyco/opencode/issues/18381 -- closed as outdated version issue]
**Warning signs:** OpenCode sessions render with non-system theme colors despite env var being set.

### Pitfall 2: OpenCode OSC background detection fails in PTY

**What goes wrong:** The `system` theme tries to detect the terminal background color via OSC escape sequences. In a PTY context, this detection may fail or return incorrect results.
**Why it happens:** OSC responses require the terminal emulator to reply, but the PTY is connected to xterm.js which may not respond to OSC queries.
**How to avoid:** The `system` theme uses `"none"` for background/foreground (inheriting terminal defaults) regardless of OSC detection. ANSI 0-15 colors work without OSC detection. The OSC detection only affects whether "dark" or "light" variant is chosen for adaptive colors. Since AgentHub themes are predominantly dark, this should produce acceptable results. If needed, we could also set `COLORFGBG=0;15` env var to hint at a dark background.
**Warning signs:** System theme picks "light" variant in an obviously dark terminal.

### Pitfall 3: mergeEnv ordering -- OPENCODE_TUI_CONFIG must survive

**What goes wrong:** If `OPENCODE_TUI_CONFIG` is already set in the parent process environment, the injection might be overridden or duplicated.
**Why it happens:** `mergeEnv` processes `base` (os.Environ) first, then `extra` (req.Env), then `required`. Later entries win.
**How to avoid:** `req.Env` is processed after `os.Environ()`, so our injection wins over any inherited value. This is correct behavior. [VERIFIED: internal/pty/native.go lines 150-178, mergeEnv implementation]
**Warning signs:** OpenCode ignores the managed tui.json and uses a different theme.

### Pitfall 4: Pre-existing `opencode-tui.json` gets overwritten on every launch

**What goes wrong:** If the user manually edits `~/.config/agenthub/opencode-tui.json`, it gets overwritten.
**Why it happens:** The managed file write happens at startup unconditionally.
**How to avoid:** This is acceptable behavior. The file is documented as managed by AgentHub. Users who want to customize OpenCode's theme for AgentHub sessions would need to modify the code. However, this file only affects OpenCode sessions launched from AgentHub (because `OPENCODE_TUI_CONFIG` is only set in those PTY processes); it does NOT affect standalone OpenCode usage.
**Warning signs:** None -- this is by design.

### Pitfall 5: configDir import from app.go not accessible in daemon package

**What goes wrong:** `configDir()` is defined in `app.go` (main package). `engine.go` is in `internal/daemon` and cannot import it.
**Why it happens:** Go package boundaries prevent importing from `main`.
**How to avoid:** Either (a) duplicate the `configDir()` helper in the daemon package, (b) pass the config path as a parameter to `NewSessionEngine()`, or (c) write the file in `app.go` at startup and pass the path through to the engine. Option (c) is cleanest -- write the file in App startup, compute the path once, and inject it.
**Warning signs:** Compilation error: cannot import "main" from "internal/daemon".

## Code Examples

### How xterm.js theme remapping works (why system theme fixes OpenCode)

```
ANSI escape code flow:

Claude Code output:  \033[31m  (ANSI color 1 = "red")
                          |
                     xterm.js looks up theme.red
                          |
                     renders with theme palette color

OpenCode (default):  \033[38;2;255;85;85m  (24-bit RGB #ff5555)
                          |
                     xterm.js uses RGB directly (cannot remap!)
                          |
                     renders with hardcoded #ff5555 regardless of theme

OpenCode (system):   \033[31m  (ANSI color 1 = "red")
                          |
                     xterm.js looks up theme.red
                          |
                     renders with theme palette color  <-- now matches Claude Code!
```
[VERIFIED: xterm.js ITheme interface defines `red`, `brightRed`, etc. for ANSI color remapping]

### Managed tui.json content

```json
{"$schema":"https://opencode.ai/tui.json","theme":"system"}
```
[CITED: opencode.ai/docs/tui/ -- tui.json format with theme field]
[CITED: opencode.ai/docs/themes/ -- "system" theme uses ANSI colors and "none" for defaults]

### Engine env injection pattern

```go
// Source: based on existing mergeEnv pattern in internal/pty/native.go line 46
// and CreateRequest.Env field in internal/pty/backend.go line 14

func (e *SessionEngine) CreateSession(ctx context.Context, cli, name, workDir string, args []string, cols, rows int, onStatus func(string, status.SessionStatus)) (string, error) {
    cliPath := e.ResolveCLI(cli)

    // ... cols/rows defaults ...

    // Per-agent environment configuration.
    var env []string
    if cli == "opencode" && e.opencodeTUIConfig != "" {
        env = append(env, "OPENCODE_TUI_CONFIG="+e.opencodeTUIConfig)
    }

    sess, err := e.backend.Create(ctx, pty.CreateRequest{
        CLI:     cliPath,
        Args:    args,
        Env:     env,
        Cols:    cols,
        Rows:    rows,
        WorkDir: workDir,
    })
    // ...
}
```

### How existing theme propagation handles ANSI palette repainting

```typescript
// Source: [VERIFIED: frontend/src/components/TerminalPanel.tsx lines 188-196]
// When the user changes theme in Settings, this effect fires:
useEffect(() => {
    if (!termRef.current) return
    termRef.current.options.theme = theme       // Updates ANSI palette mapping
    termRef.current.clearTextureAtlas()          // Forces WebGL glyph cache rebuild
    termRef.current.refresh(0, termRef.current.rows - 1)  // Repaints visible buffer
}, [theme])
// For text rendered with ANSI indices (like system theme output),
// refresh() re-resolves each ANSI index against the new palette.
// For text rendered with 24-bit RGB (OpenCode default theme), refresh()
// uses the original RGB value unchanged -- hence the theming failure.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No OpenCode support | OpenCode added as 4th supported CLI | Phase 68 (v1.13) | OpenCode sessions work but ignore theme |
| Hard-coded terminal theme | Global theme selector with 246 options | Phase 65 (v1.12) | Users can choose themes -- but OpenCode ignores them |
| OpenCode uses built-in theme (tokyonight default) | Force `system` theme via env var | This phase (71) | OpenCode sessions honor xterm.js theme |

**Deprecated/outdated:**
- Nothing deprecated in this phase. The approach adds new behavior without removing existing functionality.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | OpenCode's `system` theme outputs ANSI palette indices (0-15) rather than 24-bit RGB for syntax/UI colors | Architecture Patterns, Code Examples | If system theme still uses 24-bit RGB, the fix won't work. Mitigation: test immediately after implementation. Alternative: create a custom OpenCode theme with explicit ANSI 0-15 references. |
| A2 | `OPENCODE_TUI_CONFIG` env var takes precedence over cached `/theme` selection in OpenCode v1.4.0 | Pitfalls #1 | If cached selection wins, users who ran `/theme` in OpenCode would not see the fix. Mitigation: document as known limitation, or clear OpenCode's cache at session start. |
| A3 | OpenCode's OSC background detection produces acceptable results in AgentHub's PTY context | Pitfalls #2 | If OSC detection fails and system theme picks wrong light/dark variant, colors may look off. Mitigation: set `COLORFGBG=0;15` env var to hint dark background. |

## Open Questions

1. **Does `system` theme actually use ANSI 0-15 indices in escape output?**
   - What we know: Documentation says it "uses ANSI colors (0-15)" and "none" for defaults. Multiple GitHub issues confirm this intent.
   - What's unclear: The actual escape sequences emitted have not been captured and verified.
   - Recommendation: First implementation task should include a manual verification step: launch OpenCode with system theme, capture PTY output, grep for `\033[38;2;` (24-bit) vs `\033[3Xm` (ANSI palette). If 24-bit codes are found, the approach needs adjustment.

2. **How does OpenCode's system theme handle grayscale generation in a PTY?**
   - What we know: System theme detects background via OSC and generates grayscale based on luminance.
   - What's unclear: Whether OSC detection works when xterm.js is the terminal (the WebSocket relay doesn't forward OSC responses back to the PTY).
   - Recommendation: If grayscale colors look wrong, add `COLORFGBG=0;15` to the env to hint dark background. This is a well-known env var that many TUI frameworks respect.

3. **Pre-existing Go test failure in detect_test.go**
   - What we know: `TestKnownCLIs_HasExpectedEntries` expects 4 CLIs but `knownCLIs` has 5 (tailscale was added).
   - What's unclear: Whether this should be fixed in this phase or is tracked elsewhere.
   - Recommendation: Fix the test assertion as a minor cleanup in this phase (change expected from 4 to 5 and add "tailscale" to the expected list).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| OpenCode CLI | Testing the fix | Yes | 1.4.0 | -- |
| Go toolchain | Backend code changes | Yes | (project standard) | -- |
| pnpm / Vitest | Frontend test verification | Yes | (project standard) | -- |

**Missing dependencies with no fallback:** None.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework (Go) | `testing` (stdlib) |
| Framework (JS) | Vitest ^4.1.0 |
| Go test command | `go test ./internal/daemon/ -count=1 -run TestOpenCode` |
| JS test command | `cd /Users/ken/dev/agenthub/frontend && pnpm test` |
| Full suite | `go test ./... -count=1 -short && cd /Users/ken/dev/agenthub/frontend && pnpm test` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| THM-05 | engine.CreateSession injects OPENCODE_TUI_CONFIG for opencode CLI | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestOpenCode` | Needs new test in engine_test.go |
| THM-05 | engine.CreateSession does NOT inject env for non-opencode CLIs | unit (Go) | `go test ./internal/daemon/ -count=1 -run TestOpenCode` | Needs new test in engine_test.go |
| THM-05 | Managed opencode-tui.json exists at configDir path with system theme | unit (Go) | `go test ./... -count=1 -run TestOpenCodeTUI` | Needs new test |
| THM-05 | TerminalPanel theme useEffect includes clearTextureAtlas + refresh | unit (source-text) | `cd frontend && pnpm test` | Exists in TerminalPanel.test.tsx (THM-03 block) |
| THM-05 | Visual verification: OpenCode session changes colors on theme switch | manual (UAT) | Human verifies in app | -- |

### Sampling Rate

- **Per task commit:** `go test ./internal/daemon/ -count=1 -short && cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Per wave merge:** `go test ./... -count=1 -short && cd /Users/ken/dev/agenthub/frontend && pnpm test`
- **Phase gate:** Full suite green + manual visual verification before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/engine_test.go` -- new test: `TestCreateSession_OpenCodeEnv` verifying OPENCODE_TUI_CONFIG injection
- [ ] Test for managed tui.json file creation
- [ ] Fix pre-existing `TestKnownCLIs_HasExpectedEntries` (expects 4, has 5)

## Security Domain

This phase injects one environment variable (`OPENCODE_TUI_CONFIG`) into a child process and writes one managed JSON file to the user's config directory. No credentials, no network changes, no user input handling.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | -- |
| V3 Session Management | no | -- |
| V4 Access Control | no | -- |
| V5 Input Validation | no | File content is a hardcoded constant string; env var value is a computed filesystem path |
| V6 Cryptography | no | -- |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path injection via configDir | Tampering | `configDir()` uses `os.UserConfigDir()` which returns OS-standard paths; no user input in path computation |
| File permissions | Information Disclosure | Write with 0644 (world-readable) -- acceptable since content is just `{"theme":"system"}` with no secrets |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: internal/pty/native.go] -- `mergeEnv(os.Environ(), req.Env, ...)` on line 46; CreateRequest.Env is processed and merged into child process environment
- [VERIFIED: internal/pty/backend.go] -- `CreateRequest.Env []string` field on line 14; available for per-session env var injection
- [VERIFIED: internal/daemon/engine.go] -- `CreateSession()` builds CreateRequest without Env on line 60; modification point identified
- [VERIFIED: app.go line 316] -- `configDir()` returns `~/.config/agenthub/` and creates directory
- [VERIFIED: frontend/src/components/TerminalPanel.tsx lines 188-196] -- Theme effect with `clearTextureAtlas()` + `refresh()` already handles ANSI palette repainting
- [VERIFIED: internal/status/detector.go line 86] -- Precedent for per-CLI name detection (`if cliName == "claude"`)
- [VERIFIED: opencode v1.4.0 installed at /opt/homebrew/bin/opencode]
- [VERIFIED: OPENCODE_TUI_CONFIG env var accepted by opencode --version without errors]

### Secondary (MEDIUM confidence)
- [CITED: opencode.ai/docs/themes/] -- System theme uses ANSI colors 0-15, "none" for text/background, generates grayscale from terminal background
- [CITED: opencode.ai/docs/tui/] -- tui.json format with theme field; OPENCODE_TUI_CONFIG env var
- [CITED: opencode.ai/docs/config/] -- Configuration merge behavior and precedence
- [CITED: github.com/anomalyco/opencode/issues/101] -- System theme feature request + PR #419 merged
- [CITED: github.com/anomalyco/opencode/issues/4429] -- ANSI palette index crash fixed in v1.0.68+; v1.4.0 is well beyond this
- [CITED: github.com/anomalyco/opencode/issues/18381] -- tui.json precedence works correctly in current versions (closed as outdated version)

### Tertiary (LOW confidence)
- [ASSUMED: A1] -- System theme actually emits ANSI palette indices in escape sequences
- [ASSUMED: A2] -- OPENCODE_TUI_CONFIG wins over cached /theme selection
- [ASSUMED: A3] -- OSC background detection works acceptably in PTY context

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already installed; env injection path verified in source
- Architecture: HIGH -- uses existing patterns (CreateRequest.Env, mergeEnv, configDir, per-CLI detection)
- Fix mechanism: MEDIUM -- system theme's ANSI output is documented but not empirically verified in this context (A1)
- Pitfalls: MEDIUM -- OSC detection and theme precedence are documented but need runtime verification

**Research date:** 2026-04-13
**Valid until:** Stable -- OpenCode v1.4.0 is current; xterm.js 6.0.0 locked; patterns are architectural
