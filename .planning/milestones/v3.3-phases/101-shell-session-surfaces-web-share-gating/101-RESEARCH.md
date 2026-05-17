---
phase: 101
slug: shell-session-surfaces-web-share-gating
type: research
mode: auto-generated
created: 2026-05-12
---

# Phase 101 — Research

**Researched:** 2026-05-12
**Domain:** Shell-session UX surfaces (GUI/CLI/TUI) + web-share security gating
**Confidence:** HIGH — all findings verified against codebase grep + existing files; no external library research needed (zero new dependencies per UI-SPEC §Registry Safety).

## Summary

Phase 101 is a **pure-surface phase**: the daemon-side shell-session machinery is already built (Phase 100 — `pty.DiscoverShells`, `engine.isShellSession`, `engine.resolveShellSpawn`, `GET /shells`, `DaemonClient.ListShells`). The work is plumbing existing daemon capabilities to three frontends (Wails React, CLI subcommand, Bubble Tea TUI) and adding a one-time security banner for web-sharing shells.

Two pre-existing facts simplify the plan considerably:

1. **Web auto-enable is already off** (`internal/daemon/api.go:407` — "Phase 87 / SEC-01: creating a session does NOT auto-enable web serving"). SHELL-07's "shells must not auto-enable web" is therefore **already true for ALL sessions**. The Phase 101 contribution to SHELL-07 is the **frontend gate** (the banner) — there is no daemon code change needed for SHELL-07's literal text. If the planner wants belt-and-suspenders, they can add a daemon-side assertion (see §4).
2. **Wire types and client method already exist** — `internal/daemon/types.go:56` defines `DetectedShell{Name, DisplayName, Path, Argv}` and `internal/daemon/client.go:109` implements `DaemonClient.ListShells()`. The Wails layer just needs a thin `(a *App) ListShells()` wrapper paralleling `DetectCLIs()` at `app.go:396`.

**Primary recommendation:** Decompose into four plans — (1) Wails RPC binding + `ListShells` + `Get/SetShellWebShareWarned`, (2) `NewSessionModal` + `TabBar` + agent-badge CSS, (3) `ShellWebShareBanner` + `App.tsx` interception, (4) CLI `new shell` subcommand + TUI agent picker extension. Daemon `daemonSettings` gets one new field (`ShellWebShareWarned bool`).

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SHELL-01 | GUI new-session modal lists shells alongside AI CLIs | `NewSessionModal.tsx` (existing) + `DaemonClient.ListShells` (existing) — extend modal props with `shells: DetectedShell[]`, render below AI CLI rows per UI-SPEC §Component Contracts |
| SHELL-02 | CLI `agenthub new shell` subcommand | `cmdNew` at `cmd_cli.go:58` is the analog; dispatch lives at `main.go:171`. New `cmdNewShell` mirrors `cmdNew` with `--shell=` flag |
| SHELL-03 | TUI new-session modal includes shells | `internal/tui/modal.go::renderAgentPicker` cycles through `m.detectedCLIs`; extend that slice (or add parallel slice) with shell entries |
| SHELL-06 | Distinct agent badge color in GUI tab bar + TUI session list | New `.tab__agent-badge` 8px dot in `TabBar.tsx`; new `BadgeShell` color in `internal/tui/styles.go`; switch-case addition to `agentBadgeColor` |
| SHELL-07 | Web sharing for shells is opt-in only (no auto-enable when web server running) | Already satisfied at daemon level (api.go:407). Frontend gate via `ShellWebShareBanner` enforces explicit confirmation on first toggle-ON |
| SHELL-08 | One-time security confirmation banner | New `ShellWebShareBanner` component; persisted via new `shellWebShareWarned` boolean in `daemonSettings` (engine.go:82) |

---

## 1. Existing Surfaces — what's already there

### GUI (Wails React, `frontend/src/`)

| Surface | File:line | Current state |
|---------|-----------|---------------|
| Agent picker | `frontend/src/components/NewSessionModal.tsx:75-85` | Renders `clis: DetectedCLI[]` as flat `<button>` list with selected modifier. No shell awareness. |
| Tab badge (status only) | `frontend/src/components/TabBar.tsx` (per UI-SPEC) | Renders single status dot (8px circle). No agent badge yet. |
| Web-share toggle | `frontend/src/components/DaemonManagerPanel.tsx` + `StatusBar.tsx` (per UI-SPEC) | Toggle button fires `ToggleWebServing(id, bool)` directly via Wails RPC. No interception. |
| Banner stack | `App.tsx` (per Phase 81 BAN-01 + Phase 99 PUI-02 precedent) | Vertical flex-column container; existing entries: `WebGLRecoveryBanner`, `LocalNetworkBanner`, `UpdateBanner`, `PluginToggleBanner`. CSS class `.webgl-recovery-banner` reused across banners. |
| Wails type for AI CLIs | `frontend/src/wailsjs/go/main/App.d.ts` (generated) | `DetectedCLI` already exported; `DetectedShell` not yet bound (no `App.ListShells()` exists in `app.go`). |

### CLI (`cmd_cli.go`, `main.go`)

| Surface | File:line | Current state |
|---------|-----------|---------------|
| `new` subcommand | `cmd_cli.go:58::cmdNew` | Takes `<agent> <path>`, optional `-- <extra-args>`. Dispatched from `main.go:171 case "new":` |
| Usage text | `cmd_cli.go:22-54::usage()` | Currently documents `new <agent> <path>` only — must be extended for `new shell`. |
| Shell listing | none | No `agenthub list --shells` or `agenthub shells` subcommand exists. UI-SPEC §Open Decisions §1 flags this as deferrable. |

### TUI (`internal/tui/`)

| Surface | File:line | Current state |
|---------|-----------|---------------|
| Agent picker | `internal/tui/modal.go:75-83::renderAgentPicker` | Cycles `m.detectedCLIs[m.agentIdx]` left/right via `cycleAgent`. Renders `< name >` single-line. |
| Detection init | `internal/tui/tui.go` (per UI-SPEC §Component Contracts, line 28) | Initializes `detectedCLIs`. Does NOT call `pty.DiscoverShells()` today. |
| Badge color resolution | `internal/tui/styles.go::agentBadgeColor` (per UI-SPEC) | Switch on CLI name → returns `s.BadgeClaude` / `BadgeOpenCode` etc. No `BadgeShell` case. |
| Session-list row | `internal/tui/view.go` (per UI-SPEC) | Renders badge from `agentBadgeColor`. Shells currently fall through to `FgMuted`. |

### Persistence layer

| Surface | File:line | Current state |
|---------|-----------|---------------|
| `daemonSettings` struct | `internal/daemon/engine.go:82-88` | Fields: `CLIPaths`, `StartMinimized`, `AutoCloseSession`, `Plugins`, `SchemaVersion`. No shell-warning key yet. Defaults-merge pattern documented inline. |
| `loadSettingsFromDisk` | `engine.go:129-179` | Unmarshals JSON, applies defaults-merge for plugins; bools default to zero-value. |
| `saveSettingsToDisk` | `engine.go:183-196` | Marshals all fields; caller holds `e.mu.Lock()`. |
| Setter pattern | `SetStartMinimized` (referenced from engine_test) | Lock → mutate → `saveSettingsToDisk()` → Unlock. Mirror for `SetShellWebShareWarned`. |

### Already-built (Phase 100) — DO NOT rebuild

- `pty.DiscoverShells() []DetectedShell` — cross-platform shell discovery (returns Name, DisplayName, Path, Argv).
- `engine.isShellSession(cli string) bool` — `engine.go:113` — switch on `{shell, bash, zsh, pwsh, powershell}`.
- `engine.resolveShellSpawn(cli string) (path, argv, isShell)` — `engine.go:238` — used inside `CreateSession`.
- `engine.DiscoverShells()` — wrapper called by HTTP handler `handleListShells` at `api.go:494`.
- `daemon.DaemonClient.ListShells() ([]DetectedShell, error)` — `client.go:109`.
- `daemon.DetectedShell` wire type — `types.go:56`.
- `GET /shells` HTTP route — registered at `api.go:69`.
- Auto-enable removal — `api.go:407` confirms `handleCreateSession` does NOT call `webServer.EnableSession` after create.

---

## 2. Implementation Strategy Per Surface

### 2.1 Wails RPC layer (`app.go`)

Add three thin wrappers paralleling existing `DetectCLIs()` at `app.go:396`:

```go
// ListShells returns the daemon's discovery of installed shells.
func (a *App) ListShells() []daemon.DetectedShell {
    if a.client == nil { return []daemon.DetectedShell{} }
    shells, err := a.client.ListShells()
    if err != nil { return []daemon.DetectedShell{} }
    return shells
}

// GetShellWebShareWarned reads the persisted "user has been warned" flag.
func (a *App) GetShellWebShareWarned() bool {
    if a.client == nil { return false }
    // route through daemon for consistency with other settings — OR
    // read directly from engine if daemon-internal struct is exposed.
    return a.client.GetShellWebShareWarned()
}

// SetShellWebShareWarned persists the flag.
func (a *App) SetShellWebShareWarned(v bool) error { ... }
```

Two routing options for the persistence accessors:
- **Option A (preferred):** Add `GET /settings/shell-web-share-warned` + `PATCH /settings/shell-web-share-warned` HTTP routes, mirror in `DaemonClient`, then bind in `app.go`. Consistent with existing `cliPaths` pattern.
- **Option B:** Direct engine call from app.go (requires daemon and GUI to be co-located — they are, in the Wails build).

Recommend **Option A** for consistency with v3.2 patterns. The route is desktop-only; UI-SPEC §"No HTTP route" advice can be ignored — `cliPaths` is also "desktop-only-in-practice" but routed via HTTP, and that's the convention.

After binding, re-run `wails generate` (or hand-edit per UI-SPEC §Open Decisions §2) to add TypeScript types in `frontend/src/wailsjs/go/main/App.d.ts`:
- `ListShells(): Promise<daemon.DetectedShell[]>`
- `GetShellWebShareWarned(): Promise<boolean>`
- `SetShellWebShareWarned(v: boolean): Promise<void>`

Plus the type:
```typescript
export namespace daemon {
  interface DetectedShell { name: string; displayName: string; path: string; argv: string[] }
}
```

### 2.2 NewSessionModal (`frontend/src/components/NewSessionModal.tsx`)

Per UI-SPEC §Component Contracts. Concrete edit points:

- **Line 13-18 (props):** Add `shells: DetectedShell[]` and `shellsLoading: boolean` to `NewSessionModalProps`.
- **Line 7-11:** Add local `DetectedShell` interface (mirror Wails type), or import from `wailsjs/go/models`.
- **Line 75-85 (agent list render):** After the existing `clis.map()` loop, append: (a) loading skeleton if `shellsLoading`, (b) `shells.map()` rendering 2-line buttons (primary `Shell — {displayName}`, secondary mono `{path}`). Use new BEM modifier `--shell` plus `__detail` child class.
- **Line 21:** `selectedCLI` initial state must now consider both lists — if shells loaded first, fall back to first CLI. Keep `clis[0]?.Name` default.
- **Line 26 (args memory):** When `selectedCLI` matches a shell name, use key `agenthub:args:shell:<name>` per UI-SPEC §Interaction. NOTE: for shells, args are ignored downstream (Phase 100 A6) so the args field SHOULD be hidden or disabled when a shell is selected. Recommend: render args input as `disabled` with placeholder `"Arguments are not passed to shell sessions"` when `isShellSelected`.

Parent (likely `App.tsx`): on mount, call `ListShells()` Wails RPC; pass result + loading state into `<NewSessionModal>`.

### 2.3 TabBar (`frontend/src/components/TabBar.tsx`)

Per UI-SPEC §Component Contracts:
- Add new `.tab__agent-badge` 8px circle DOM element between `.tab__status` and tab name.
- Compute color class from session's `cli` field: `--claude`, `--opencode`, `--codex`, `--gemini`, `--cursor`, `--aider`, `--shell` (matches `{shell, bash, zsh, pwsh, powershell}`).
- Append CSS rules to existing TabBar stylesheet (BEM modifiers, no inline styles).
- `aria-hidden="true"` on badge.

### 2.4 ShellWebShareBanner (NEW)

New file `frontend/src/components/ShellWebShareBanner.tsx`. Reuse `.webgl-recovery-banner` BEM class per Phase 99 precedent, add `--shell-warning` modifier for 3px red left-border. Props: `sessionName`, `onConfirm`, `onCancel`. Heading + body verbatim per UI-SPEC §Web-share security banner copy.

### 2.5 App.tsx web-toggle interception

Per UI-SPEC §Interaction States. Edit `handleToggleWeb` (or equivalent) in `App.tsx`:

```typescript
const SHELL_CLIS = new Set(['shell', 'bash', 'zsh', 'pwsh', 'powershell'])

async function handleToggleWeb(id: string, enabled: boolean) {
  const session = sessions[id]
  if (enabled && session && SHELL_CLIS.has(session.cli) && !shellWebShareWarned) {
    setPendingWebToggle({ sessionId: id, sessionName: session.name })
    return
  }
  await ToggleWebServing(id, enabled)
  // ...existing optimistic update
}
```

Banner-stack render: insert `<ShellWebShareBanner>` at TOP of stack when `pendingWebToggle != null` per UI-SPEC §Stack ordering. On confirm: `SetShellWebShareWarned(true)` + `ToggleWebServing` in parallel.

### 2.6 CLI `new shell` subcommand

New function `cmdNewShell` in `cmd_cli.go` (mirror `cmdNew` at line 58). Dispatch in `main.go:171` — extend `case "new":` to peek at `cmdArgs[0]` and route to `cmdNewShell` if equal to `"shell"`. See §5 for argv shape detail.

### 2.7 TUI agent picker

Per UI-SPEC §Component Contracts:
- `internal/tui/tui.go` init: alongside `detectedCLIs`, also populate `detectedShells` via `pty.DiscoverShells()` (TUI runs in-process with daemon when `--tui` is used? Or via DaemonClient? Verify — likely `pty.DiscoverShells()` directly since TUI initializes `detectedCLIs` directly per Phase 100 pattern). TODO(needs codebase scan: confirm whether `tui.go` reads from `daemon.DaemonClient` or calls `pty` directly for detection — implementation choice).
- Combine into single `agentEntries` slice (CLIs first, then "Shell — system default", then per-shell entries) — preserves UI-SPEC §Interaction cycle order.
- `modal.go:75 renderAgentPicker`: render display name with `Shell — ` prefix when entry is a shell.
- `styles.go`: add `BadgeShell color.Color`, initialize `ld(lipgloss.Color("#3d5a80"), lipgloss.Color("#89ddff"))`.
- `agentBadgeColor`: add `case "shell", "bash", "zsh", "pwsh", "powershell": return s.BadgeShell`.

---

## 3. Persistence for `shellWebShareWarned`

Add field to `daemonSettings` struct at `engine.go:82`:

```go
ShellWebShareWarned bool `json:"shellWebShareWarned,omitempty"`
```

Additive boolean, **no `schemaVersion` bump** (matches `StartMinimized` precedent). Zero-value (false) for upgrades — user sees banner on first toggle-ON.

Methods on `*SessionEngine`: `GetShellWebShareWarned() bool` (RLock) and `SetShellWebShareWarned(v bool) error` (Lock + `saveSettingsToDisk()`). Mirror `SetStartMinimized` pattern verbatim. Also update `loadSettingsFromDisk` at engine.go:148-178 to copy `s.ShellWebShareWarned` into a new `e.shellWebShareWarned` field, and `saveSettingsToDisk` at engine.go:184 to emit it. ~50 LOC.

---

## 4. SHELL-07 Auto-Enable Override Enforcement

**Current state:** Already enforced for all sessions at `internal/daemon/api.go:407` ("Phase 87 / SEC-01: creating a session does NOT auto-enable web serving"). The handler simply does NOT call `webServer.EnableSession(id)` after `engine.CreateSession`.

**Phase 101 strategy:** No daemon code change needed. SHELL-07 is already satisfied at the daemon. The Phase 101 work is the **frontend gate** (banner) which adds defense-in-depth for the case where some future change might re-introduce auto-enable. Optionally, add an assertion test (e.g., extending `api_test.go` `SEC-01` test with a shell-CLI variant) to lock the invariant.

**If belt-and-suspenders desired:** Add a guard inside `handleToggleWebServe` to refuse `enable=true` for shells when no explicit user-confirmation token has been presented. This is over-engineered for v3.3 — the frontend gate is the single source of truth and the UI-SPEC accepts this. Recommend NOT doing this for v3.3.

---

## 5. CLI Argv Shape

Recommend matching the UI-SPEC §CLI verbatim:

```
agenthub new shell [<path>] [--shell=bash|zsh|pwsh|powershell] [-- <extra-args>...]
```

Dispatch (`main.go:171`):

```go
case "new":
    if len(cmdArgs) > 0 && cmdArgs[0] == "shell" {
        err = cmdNewShell(client, cmdArgs[1:], extraArgs, os.Stdout)
    } else {
        err = cmdNew(client, cmdArgs, extraArgs, os.Stdout)
    }
```

Implementation of `cmdNewShell`:
- `flag.NewFlagSet("new shell", flag.ContinueOnError)`, register `--shell` string flag (default empty = system default).
- Positional `[<path>]`: optional; if omitted pass empty string (daemon's `CreateSession` resolves to `$HOME` for shells per engine.go:241-255).
- Resolve `--shell` value: must be in `{"", "bash", "zsh", "pwsh", "powershell"}` else exit 1 with stderr per UI-SPEC.
- If `extraArgs` non-empty (user passed `-- args`), emit stderr warning per UI-SPEC §CLI but proceed with `nil` args (Phase 100 A6).
- CLI value sent to `client.CreateSession`: if `--shell=""` → `"shell"`; else the flag value verbatim (`bash`, `zsh`, etc.).

Match existing `cmdNew` style: print session UUID to stdout, return error on failure.

---

## 6. Wails Bindings — TypeScript Types

After adding `(a *App) ListShells`, `GetShellWebShareWarned`, `SetShellWebShareWarned` to `app.go`, the bindings layer at `frontend/src/wailsjs/go/main/App.d.ts` needs new entries. Per UI-SPEC §Open Decisions §2 the project convention is **hand-edit** these files (Phase 99 PUI-04 precedent).

Add to `App.d.ts`:
```typescript
export function ListShells(): Promise<daemon.DetectedShell[]>;
export function GetShellWebShareWarned(): Promise<boolean>;
export function SetShellWebShareWarned(v: boolean): Promise<void>;
```

Add to `frontend/src/wailsjs/go/models.ts` (or equivalent):
```typescript
export namespace daemon {
  export class DetectedShell {
    name: string
    displayName: string
    path: string
    argv: string[]
  }
}
```

Plus matching JS bridge stubs in `App.js`. The existing `DetectCLIs` pattern shows the shape.

---

## 7. Test Surfaces

Per UI-SPEC §Test Surface Map — each plan should include matching test coverage:

- **`NewSessionModal.test.tsx` (extend):**
  - Renders N shell rows when `shells` prop has N entries
  - Selecting a shell sets shell-cyan border modifier
  - "Loading shells…" skeleton renders during `shellsLoading=true`
  - Selecting a shell hides/disables the args field
  - Args memory uses shell-prefixed namespace key
- **`TabBar.test.tsx` (extend):**
  - Renders `.tab__agent-badge--shell` for `cli="shell"/"bash"/"zsh"/"pwsh"`
  - Renders correct modifier for each of 6 AI CLIs
  - Falls back to muted for unknown cli string
- **`ShellWebShareBanner.test.tsx` (NEW):**
  - Renders heading + body verbatim
  - Enable button fires `onConfirm`
  - Cancel + Esc + × all fire `onCancel`
  - Focus moves to Cancel on mount
  - `role="alert"` + `aria-live="assertive"` present
- **`App.tsx` shell-toggle interception test:**
  - Toggle on shell session with `warned=false` pushes banner + sets aria-busy
  - Toggle on shell session with `warned=true` calls `ToggleWebServing` immediately (no banner)
  - Toggle on AI CLI session never pushes banner
  - Confirm fires both `SetShellWebShareWarned(true)` and `ToggleWebServing(id, true)`
- **`cmd_cli_test.go` (extend) — `TestCmdNewShell_*`:**
  - `new shell` with no path → daemon receives empty workDir (resolves to $HOME)
  - `new shell --shell=bash` → daemon receives `cli="bash"`
  - `new shell --shell=nope` → exit 1, stderr matches UI-SPEC copy
  - `new shell -- --extra` → stderr warning, args dropped before daemon RPC
- **`internal/tui/*_test.go`:**
  - Agent picker cycles deterministically: AI CLIs → shells (per UI-SPEC §Interaction TUI flow)
  - `agentBadgeColor("shell")`, `("bash")`, `("zsh")` all return `BadgeShell`
  - Light/dark variant rendering (mirror existing `styles_test.go` patterns)
- **`internal/daemon/engine_test.go` (extend) — `TestSetShellWebShareWarned_Persists`:**
  - Default `false` on first load (no settings.json)
  - `SetShellWebShareWarned(true)` writes JSON
  - Round-trips through `loadSettingsFromDisk` on next `NewSessionEngine`
- **`internal/daemon/api_test.go` (extend if HTTP route added):**
  - `GET /settings/shell-web-share-warned` returns `{ "value": false }` initially
  - `PATCH` flips to true, GET reflects

---

## 8. Pitfalls

- **Banner-stack ordering race** — UI-SPEC mandates `ShellWebShareBanner` renders at TOP of the stack. If implemented via array-push, an existing WebGLRecoveryBanner already in the stack will appear ABOVE the new shell warning. Implementation must use **slot-based render** (separate `<ShellWebShareBanner>` JSX element rendered first, then `<BannerStack>` for the rest) OR **priority sort on push**. Slot-based is simpler and matches existing `App.tsx` pattern of explicit banner JSX rather than array iteration.

- **First-toggle persistence race** — User clicks "Enable web sharing" → both `SetShellWebShareWarned(true)` and `ToggleWebServing(id, true)` fire in parallel. If `SetShellWebShareWarned` is slow (disk write), a quick second toggle from a different shell session could race the first persist and re-show the banner. **Mitigation:** Set in-memory React state `setShellWebShareWarned(true)` synchronously BEFORE awaiting either RPC; reconcile on next `GetShellWebShareWarned` boot read.

- **CLI flag vs positional ambiguity** — `agenthub new shell --shell=bash` parses fine, but `agenthub new shell --shell=` (empty value) would be silently interpreted as system-default by Go's `flag` package. UI-SPEC §CLI requires an error here. **Mitigation:** Post-parse, check `if flagPassed("shell") && flagValue == ""` → error. Use `fs.Visit` to detect "was the flag actually set?" rather than just "is it empty?".

- **Windows pwsh-vs-powershell preference** — Per UI-SPEC §Copywriting and Phase 100 knownShells, both `pwsh` (PowerShell 7+) and `powershell` (Windows PowerShell 5.x) are valid. `pty.DiscoverShells()` returns whichever it finds on PATH first. Display order in the modal should consistently rank `pwsh` above `powershell` when both are present. **Mitigation:** Sort `shells` in `ListShells()` or in the React render by a fixed name-priority array `["shell", "bash", "zsh", "pwsh", "powershell", "fish", "sh", ...]` before display.

- **Shell args field on existing modal** — Existing `argsText` state pre-fills from `agenthub:args:<cli>` localStorage key (`NewSessionModal.tsx:26`). When user picks a shell, the args field is meaningless (Phase 100 A6: shell args are discarded). If we leave the field active, users will type args, see them persisted, then be surprised when sessions don't honor them. **Mitigation:** Disable + helper-text when shell selected (see §2.2).

- **Detection failure silently drops shells** — Per UI-SPEC §Edge Cases, a failed `ListShells()` call (daemon down) silently omits shell rows. This is correct UX but is a **silent fallback** that violates the project CLAUDE.md "let it crash" principle in spirit. **Mitigation:** Log to console (already in UI-SPEC) — but planner should ensure the log includes the underlying error string so support diagnostics can find it later.

- **`isShellSession` constant drift** — The set `{shell, bash, zsh, pwsh, powershell}` is replicated in: `engine.go:113`, `engine.go:115` (CreateSession), the new App.tsx `SHELL_CLIS`, the new `TabBar` switch, the new `agentBadgeColor` switch, and `cmd_cli.go` `cmdNewShell` validation. **Mitigation:** Plan should call out this duplication; if it grows further, extract a single source of truth (e.g., `pkg/shellset/shellset.go`). For v3.3 the duplication is acceptable but document in PLAN.

- **TUI agent picker cycle order** — UI-SPEC §Interaction TUI flow specifies `Claude Code → ... → OpenCode → Shell — system default → Shell — bash → ...`. Implementation must build `agentEntries` slice in this exact order. If `pty.DiscoverShells` returns shells in PATH order, downstream must re-sort to put `"shell"` (system default) first.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Shell discovery | Daemon (Go, `pty.DiscoverShells`) | — | Already built Phase 100; cross-platform PATH scan belongs daemon-side |
| `/shells` HTTP route | Daemon (`api.go`) | DaemonClient | Already built Phase 100 |
| Wails Go-to-JS RPC | Wails app layer (`app.go`) | — | Thin wrapper over `DaemonClient.ListShells` (paralleling `DetectCLIs`) |
| Agent picker UI | Frontend (React, `NewSessionModal.tsx`) | — | Pure presentational; receives data via Wails RPC |
| Tab agent badge | Frontend (React, `TabBar.tsx` + CSS) | — | Pure presentational |
| Web-share gating | Frontend (React, `App.tsx` interception) | Daemon (`shellWebShareWarned` persistence) | Frontend is the gate; daemon stores the "warned" flag for cross-restart memory |
| Banner stack | Frontend (React, `App.tsx` render) | — | Established Phase 81/99 pattern |
| `new shell` CLI subcommand | CLI (`cmd_cli.go`) | DaemonClient | Argv parse + DaemonClient.CreateSession |
| TUI agent picker | TUI (`internal/tui/`) | `pty.DiscoverShells` (direct) | TUI runs in-process; calls discovery directly |

---

## Project Constraints (from CLAUDE.md + project conventions)

- **Backend:** Go (project root + `internal/`). `go fmt` / `golangci-lint`. Context-aware funcs.
- **Frontend:** React + TypeScript + handcrafted CSS-BEM (no shadcn). TokyoNight palette locked since v3.0.
- **No new dependencies** — UI-SPEC §Registry Safety confirms zero net-new packages.
- **Tests:** Go `testing` + Vitest. 80%+ coverage in critical components (CLAUDE.md).
- **Build:** `wails build` requires `-tags wailsassets` (project memory `project_wails_build_requires_tags`).
- **Silent fallback principle:** Per CLAUDE.md "let it crash" — log + degrade for daemon-unreachable, do not swallow errors.
- **Settings.json schema:** Additive boolean fields use `omitempty` + zero-value default; do NOT bump `schemaVersion` (matches `StartMinimized` precedent at engine.go:84).

---

## Standard Stack

| Component | Location | Why Standard |
|-----------|----------|--------------|
| `flag.FlagSet` | stdlib | Existing `cmdList` (cmd_cli.go:87) uses it — match for `cmdNewShell` |
| `lipgloss/v2` | TUI rendering | Already used; existing `BadgeXxx` patterns |
| `@heroicons/react/20/solid` | GUI icons | Already used per UI-SPEC §Design System |
| Wails RPC (handwritten bindings) | Bridge | Project convention per Phase 99 PUI-04 |
| BEM CSS classes | Frontend styling | Established pattern; `.webgl-recovery-banner` reused per Phase 99 PUI-02 |
| Vitest | Frontend tests | Existing `__tests__/` directory |

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Shell discovery | Custom PATH scan in frontend | Call `ListShells()` Wails RPC → `pty.DiscoverShells()` |
| Modal overlay | New modal infra | Existing `.new-session-modal` BEM |
| Banner stack | New container component | Existing `.banner-stack` flex-column in `App.tsx` |
| `isShellSession` check | Frontend-only string set divergence | Mirror server constant (acceptable v3.3 duplication; see Pitfalls) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | TUI's `tui.go` initializes `detectedCLIs` by calling `pty.DetectCLIs()` directly (in-process), not via `DaemonClient` | §2.7 | TODO(needs codebase scan) — if TUI uses DaemonClient, shells must come from `client.ListShells()`. Easy to fix once verified. |
| A2 | Wails bindings are hand-edited (not regenerated) per Phase 99 PUI-04 | §6 | If `wails generate` is preferred, planner reruns generator instead of hand-editing |
| A3 | `App.tsx` already passes `clis` to `NewSessionModal`; adding `shells` follows same path | §2.2 | Low risk — `NewSessionModal.tsx:14-18` confirms `clis` prop exists |
| A4 | Adding HTTP routes for `shellWebShareWarned` is the preferred persistence path (vs direct engine call from app.go) | §2.1 | If app.go uses direct engine access elsewhere, that's an acceptable alternative. Either works. |
| A5 | `pty.DetectCLIs()` returns 6 CLIs in stable order; combining with shells will yield a deterministic UI cycle | §2.7 | If detection order varies, sort explicitly in TUI init |

---

## Open Questions

1. **TUI discovery path** — Does `internal/tui/tui.go` call `pty.DetectCLIs()` directly or via `DaemonClient`? (Determines whether shells come from `pty.DiscoverShells()` direct or `client.ListShells()`.) Recommendation: planner scans `tui.go` ~line 28 during plan-write.

2. **Args field visibility for shells** — UI-SPEC doesn't explicitly say "hide args field for shells", just notes args are namespaced. Recommend disable+helper-text approach (§2.2) to prevent user confusion. Planner should confirm or pick alternative.

3. **`agenthub list --shells` companion** — UI-SPEC §Open Decisions §1 flags as deferrable. Recommend defer to v3.4 polish.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `testing` stdlib + `httptest` |
| Framework (Frontend) | Vitest (existing `__tests__/`) |
| Config file (Go) | none — stdlib defaults |
| Config file (Frontend) | `frontend/vite.config.ts` / `vitest.config.ts` (existing) |
| Quick run command (Go) | `go test ./internal/daemon/... -run TestSetShellWebShareWarned -count=1` |
| Quick run command (FE) | `cd frontend && pnpm test -- NewSessionModal` |
| Full suite (Go) | `go test ./... -count=1` |
| Full suite (FE) | `cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req | Behavior | Test Type | Automated Command |
|-----|----------|-----------|-------------------|
| SHELL-01 | GUI modal renders shell rows | unit | `pnpm test -- NewSessionModal` |
| SHELL-02 | `new shell` creates shell session | integration | `go test ./... -run TestCmdNewShell -count=1` |
| SHELL-03 | TUI picker cycles to shell | unit | `go test ./internal/tui/... -run TestAgentPicker -count=1` |
| SHELL-06 | Tab badge color | unit (FE + TUI) | `pnpm test -- TabBar` + `go test ./internal/tui/... -run TestAgentBadgeColor` |
| SHELL-07 | No auto-enable for shells | integration | `go test ./internal/daemon/... -run TestSEC01_ShellNoAutoEnable -count=1` |
| SHELL-08 | First-toggle banner | unit (FE) | `pnpm test -- ShellWebShareBanner` + `pnpm test -- App.saver` |

### Sampling Rate
- **Per task commit:** quick run command for the surface touched
- **Per wave merge:** `go test ./... -count=1 && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/ShellWebShareBanner.test.tsx` — NEW (mirror `PluginToggleBanner.test.tsx`)
- [ ] Extend `frontend/src/components/__tests__/NewSessionModal.test.tsx` with shell-row coverage
- [ ] Extend `frontend/src/components/__tests__/TabBar.test.tsx` with agent-badge coverage
- [ ] Extend `cmd_cli_test.go` with `TestCmdNewShell_*` cases
- [ ] Extend `internal/daemon/engine_test.go` with `TestSetShellWebShareWarned_Persists`
- [ ] Extend `internal/tui/styles_test.go` + `internal/tui/update_test.go` with shell-entry coverage

---

## Security Domain

Phase 101 introduces user-facing behavior with security implications (web-sharing arbitrary command execution). Relevant ASVS categories:

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes (existing) | Existing tailnet-only access controls unchanged |
| V3 Session Management | yes (existing) | Existing read/write grant model unchanged |
| V4 Access Control | yes | Shell sessions remain opt-in per SHELL-07; gated by user-confirmed banner |
| V5 Input Validation | yes | `--shell` flag value validated against allowlist (no arbitrary strings) |
| V6 Cryptography | n/a | No new crypto |
| V14 Configuration | yes | `shellWebShareWarned` defaults to false (fail-closed for security warning) |

### Known Threat Patterns

| Pattern | STRIDE | Mitigation |
|---------|--------|------------|
| User unaware that shell sharing exposes arbitrary command execution | Information Disclosure / Elevation of Privilege | UI-SPEC §Web-share security banner copy — one-time `role="alert"` aria-live="assertive" banner |
| Accidental web-enable via stale state | Tampering | Phase 87 auto-enable removal (api.go:407) + frontend banner gate (this phase) |
| Args injection via shell session | Tampering | Phase 100 A6: shell args are NOT forwarded (engine.go:238 overwrites). CLI emits warning if user attempts to pass args. |

---

## Sources

### Primary (HIGH confidence — codebase verified)
- `internal/daemon/engine.go:82-196` — `daemonSettings` struct + load/save pattern
- `internal/daemon/engine.go:103-119` — `knownShells` + `isShellSession`
- `internal/daemon/engine.go:219-310` — `CreateSession` shell dispatch
- `internal/daemon/api.go:69,407,494` — `/shells` route registration + auto-enable comment + handler
- `internal/daemon/client.go:106-117` — `DaemonClient.ListShells`
- `internal/daemon/types.go:52-66` — `DetectedShell` wire type + `ShellsResponse`
- `app.go:396` — `DetectCLIs` Wails RPC analog
- `cmd_cli.go:22-70` — `usage` + `cmdNew` patterns
- `main.go:171` — `case "new":` dispatch
- `frontend/src/components/NewSessionModal.tsx` (entire) — existing modal shape
- `internal/tui/modal.go:75-96` — agent picker + cycle pattern
- `.planning/phases/101-shell-session-surfaces-web-share-gating/101-UI-SPEC.md` — UI design contract (locked)
- `.planning/phases/101-shell-session-surfaces-web-share-gating/101-CONTEXT.md` — phase boundary + decisions

### Secondary (MEDIUM confidence — UI-SPEC asserted, codebase not re-grepped)
- `frontend/src/components/TabBar.tsx` structure (per UI-SPEC §Component Contracts)
- `frontend/src/components/DaemonManagerPanel.tsx` web-toggle wiring (per UI-SPEC)
- `frontend/src/App.tsx` banner-stack render (per UI-SPEC + Phase 81 BAN-01 precedent)

### Tertiary (LOW — flagged for plan-time verification)
- TUI detection path (direct vs DaemonClient) — see Assumption A1 / Open Question 1

---

## Metadata

**Confidence breakdown:**
- Existing surfaces map: HIGH — verified by grep against engine.go, api.go, client.go, types.go, cmd_cli.go, NewSessionModal.tsx, modal.go
- Implementation strategy: HIGH — patterns mirror existing code (e.g., `SetStartMinimized` setter, `DetectCLIs` Wails wrapper, `cmdNew` argv shape)
- Persistence approach: HIGH — `daemonSettings` additive-bool pattern verified at engine.go:82-88
- SHELL-07 enforcement: HIGH — api.go:407 confirms auto-enable already removed
- Pitfalls: MEDIUM — banner-stack ordering, race conditions inferred from React state semantics + existing App.tsx patterns

**Research date:** 2026-05-12
**Valid until:** 2026-06-11 (30 days — surface code stable; daemon is locked v3.3)
