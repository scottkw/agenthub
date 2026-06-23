# Phase 149: Google Antigravity Agent - Research

**Researched:** 2026-06-22
**Domain:** CLI agent integration (Go backend + TypeScript frontend badge)
**Confidence:** HIGH (spike facts), MEDIUM (status-classifier tuning — output patterns not yet observable without live install)

---

## CRITICAL GATE RESULT — READ FIRST

**D-01 Verdict: D-03 (proceed with source-level acceptance)**

The Antigravity CLI (`agy`) is a confirmed standalone, PTY-capable binary. However, it is still
in limited rollout / waitlist for many users and cannot be confirmed installable for live UAT in
this phase. Build the full integration. Live REPL launch becomes manual checklist item M-15.

See `## Verification Spike (D-01)` below for the four-fact evidence.

---

## Verification Spike (D-01)

> Hard gate mandated by CONTEXT.md. Every fact below must be answered before any integration
> code is written. Evidence is cited per the source hierarchy.

### Fact 1 — Binary name and install method per platform

**Binary name: `agy`** — this is the executable invoked from a terminal on all platforms.
Not `antigravity` (that is the desktop app cask name). [CITED: dev.to hands-on guide, pasqualepillitteri.it, medium.com/google-cloud tutorial]

**Install methods:**

| Platform | Method | Command | Install path |
|----------|--------|---------|--------------|
| macOS / Linux | Official installer script | `curl -fsSL https://antigravity.google/cli/install.sh \| bash` | `~/.local/bin/agy` |
| macOS (Homebrew) | `brew install antigravity-cli` (cask, formula name `antigravity-cli`) | — | `/opt/homebrew/bin/agy` (ARM) or `/usr/local/bin/agy` (Intel) |
| Windows | PowerShell installer | `irm https://antigravity.google/cli/install.ps1 \| iex` | `%LOCALAPPDATA%\agy\bin\agy.exe` (canonical: `C:\Users\<user>\AppData\Local\agy\bin\agy.exe`) |
| Windows (alt) | CMD | `curl -fsSL https://antigravity.google/cli/install.cmd -o install.cmd && install.cmd && del install.cmd` | same as PowerShell path |

[CITED: github.com/google-antigravity/antigravity-cli README, codelabs.developers.google.com/antigravity-cli-hands-on, pasqualepillitteri.it]

**PATH augmentation implications:**
- macOS/Linux `~/.local/bin` is already in `AugmentServicePath()` (path.go line 23: "Anthropic native installer").
- `/opt/homebrew/bin` and `/usr/local/bin` are already covered (path.go lines 25-26).
- Windows `%LOCALAPPDATA%\agy\bin` is NOT yet in `platformExtraBins()` and must be added.
  The existing Windows entries cover `%LOCALAPPDATA%\pnpm`, `%LOCALAPPDATA%\Programs\nodejs`, etc., but not `%LOCALAPPDATA%\agy\bin`.

**Homebrew naming conflict:** the `antigravity` cask (desktop GUI app) may also write an `agy`
binary, causing a conflict message when `antigravity-cli` is subsequently installed. This is a
known user-land issue — it does not affect AgentHub's PATH augmentation because we probe for the
`agy` binary by name, not by cask. [CITED: blog.davep.org/2026/05/21/antigravity-cli-now-on-homebrew.html]

### Fact 2 — Runs standalone (no IDE/desktop daemon required)

**YES — confirmed standalone.**

Multiple authoritative sources agree: "no desktop app required." The GitHub README and Google
Codelabs documentation explicitly state the CLI makes the "full power of that platform available
directly from your terminal, with no desktop app required." Antigravity 2.0 is five separate
product surfaces launched simultaneously (desktop app, `agy` CLI, Python/TS/Go SDK, Managed
Agents API, Gemini Enterprise Agent Platform) — the CLI is architecturally independent.

The optional `/export` command syncs a CLI session to the desktop app, but this is supplementary
functionality, not a prerequisite. [CITED: github.com/google-antigravity/antigravity-cli,
dev.to/arindam_1729/antigravity-cli-a-hands-on-guide, medium.com/google-cloud tutorial series]

NOTE: One third-party setup guide (itecsonline.com) implied the desktop app must be "running" on
macOS, but this contradicts four other sources including the official GitHub repository README and
Google Codelabs. That guide appears to describe the desktop-app installation path (DMG), not the
CLI. The official source takes precedence. [ASSUMED] risk is LOW.

### Fact 3 — Interactive PTY REPL when launched bare

**YES — `agy` with no arguments launches an interactive TUI.**

Running `agy` from any project directory launches a full Terminal User Interface (TUI) consisting
of: a scrollable conversation pane, a `>` prompt (changes to `!` in shell passthrough mode), and a
status bar showing active model, token usage, workspace path, and user email.

Behavior description: "Running `agy` from any project directory launches the full TUI: a
scrollable conversation pane, a `>` prompt, and a status bar showing the active model, token
usage, and any running operations." Quitting: `/quit` command or Ctrl+D twice.
[CITED: codelabs.developers.google.com/antigravity-cli-hands-on, dev.to hands-on guide,
medium.com/google-cloud/antigravity-cli-tutorial-series]

The TUI is implemented in Go (rewritten from the Node.js Gemini CLI codebase for lower memory
footprint and faster startup). [CITED: agentpedia.codes/blog/antigravity-cli-deep-dive]

This is a full PTY REPL — equivalent in behavior to how `claude`, `codex`, and `gemini` behave.

### Fact 4 — Auth flow degrades inside a PTY

**YES — explicit SSH/PTY degradation path is documented and supported.**

Auth flow on first run:
- **Desktop / local machine:** Browser opens automatically for Google OAuth consent, token cached
  in `~/.gemini/antigravity-cli/credentials.enc` (macOS Keychain on macOS, Credential Manager on
  Windows, libsecret on Linux).
- **SSH / PTY / headless:** CLI detects the SSH/remote context and prints an authorization URL
  plus a one-time code. User opens the URL on their local machine, completes OAuth, token pinned
  back to the SSH session. This was "explicitly identified as addressing a pain point in the
  earlier Gemini CLI." Works "behind double SSH bouncing or tmux."
  [CITED: agentpedia.codes/blog/antigravity-cli-deep-dive, pasqualepillitteri.it, medium.com/google-cloud/getting-started-antigravity-2-0]

**Alternative: `ANTIGRAVITY_API_KEY` env var** — API key authentication is available as an
alternative to OAuth. Set the env var before running. [CITED: dev.to hands-on guide]
Note: As of the research date, full API-key support was a feature request in some sources
(github.com/google-antigravity/antigravity-cli/issues/78) but confirmed working in others.
[ASSUMED] the env-var path works for key holders but may not be GA for all account tiers.

**Known PTY quirk:** The `agy auth login` OAuth URL is long enough to hard-wrap in some terminal
emulators when the terminal column width is narrow. The CLI queries console width and injects
physical newlines, breaking Ctrl+Click in those terminals. This is a cosmetic UX issue (auth
itself completes), not a functional blocker for AgentHub's relay.
[CITED: github.com/google-antigravity/antigravity-cli/issues/43, issues/315]
The auth-flow guidance modal (D-13) should document: "run `agy auth login` in a wide terminal
before first use in AgentHub if Ctrl+Click is needed."

### D-01 Classification

**D-03: Proceed with source-level acceptance.**

All four facts confirm a standalone, PTY-capable CLI with a documented headless auth degradation
path. However, `agy` remains in a limited rollout / waitlist stage — it is not universally
installable today for live UAT on this machine. Per D-03:

- Build the full integration.
- Source-level acceptance: `knownCLIs` entry, unit tests (found/not-found/stored-path/stale-path),
  badge hex WCAG-AA verified at source, picker shows "Google Antigravity", `agenthub new agy`
  works, README waitlist note present.
- Live REPL launch: manual checklist item **M-15** in TESTING.md Section 5.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01** (gate): Research spike must confirm four facts before any integration code. Done — D-03 outcome.
- **D-02** (pause): Not triggered — all four facts passed.
- **D-03** (proceed, source-level): Full integration with source/unit acceptance; live launch is M-15 manual item.
- **D-04** (always a separate top-level agent): "Google Antigravity" is its own first-class entry regardless of binary relationship to `gemini`. Never fold into Gemini CLI.
- **D-05** (flat picker, distinct names): Append "Google Antigravity" to the flat agent picker. No "Google" grouping subsection. Display name "Google Antigravity" with distinct description.
- **D-06** (lock `#ff9e64`): Badge color is exactly `#ff9e64` (TokyoNight orange).
- **D-07** (WCAG-AA at hex, not by eye): Must verify hex contrast. See `## WCAG-AA Contrast Analysis` below.
- **D-08** (update all color sites in lockstep): `agentBadge.ts` switch, `.tab__agent-badge--{key}`, `.hub-card[data-agent="{key}"]` spine, `.hub-card[data-agent="{key}"] .hub-card__badge`.
- **D-09** (badge key = real binary name): Key must equal `agy` (the confirmed binary name), not `antigravity`. CSS comments use accurate hue label.
- **D-13** (everything in Phase 149): Config shim (if needed), auth modal (if needed), status-classifier tuning (if needed) — all in this phase.

### Claude's Discretion

- PATH augmentation install locations — derived from spike findings; planner/executor decides exact paths (see Fact 1 above — Windows path confirmed as `%LOCALAPPDATA%\agy\bin`).
- Per-agent argument-memory wiring and Settings → Paths override — mirror existing mechanism.
- Engine changes expected to be none for the clean PTY case; add shim only if D-13 quirks demand it.

### Deferred Ideas (OUT OF SCOPE)

- "Google" picker subsection grouping Gemini CLI + Antigravity — declined for v1 (D-05).
- Gemini launch-mode-variant treatment — rejected (D-04).
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AGENT-01 | Google Antigravity CLI is selectable as a supported agent and launches correctly (#65) | D-03 outcome: full integration via `knownCLIs` + PATH + badge. Source-level acceptance for this phase; live launch is M-15. |
</phase_requirements>

---

## Summary

Google Antigravity CLI (`agy`) is a standalone, PTY-capable terminal coding agent that satisfies all four D-01 verification criteria. The D-03 outcome applies: build the full integration, accept at the source/unit level, and add a manual M-15 checklist item for live REPL launch once the waitlist clears.

The integration is mechanically straightforward. One `CLISpec` row in `knownCLIs`, one Windows-specific PATH entry in `path_windows.go`, three CSS rules in `style.css`, one case in `agentBadge.ts`. The status-classifier architecture (`PatternsForCLI`) already has a natural extension point for `agy`-specific patterns once output samples are available — use the FallbackPatterns default for now and tune if needed.

**Critical finding:** `#ff9e64` fails WCAG AA contrast against the light-theme `--hub-surface: #ffffff` (2.03:1, threshold 3:1). However, every existing agent badge color also fails on white light theme (ranging 1.52–2.65:1) with no per-agent light-theme overrides in the codebase. This is a pre-existing systemic gap. D-07 requires hex-level verification and honest reporting — see `## WCAG-AA Contrast Analysis` for the full picture and the correct planner response.

**Primary recommendation:** Proceed with D-03 integration. Add `agy` to `knownCLIs`, add `%LOCALAPPDATA%\agy\bin` to Windows PATH augmentation, add badge to all three CSS sites and `agentBadge.ts`, extend unit tests with the `agy` entry, add TESTING.md M-15 item and traceability row.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI detection (`agy` on PATH) | Backend / daemon | — | `knownCLIs` + `exec.LookPath` lives in `internal/pty/detect.go`; same for all agents |
| PATH augmentation (Windows `%LOCALAPPDATA%\agy\bin`) | Backend / daemon | — | `internal/daemon/path_windows.go` `platformExtraBins()` |
| Agent session launch | Backend / daemon | — | Engine spawns `cliPath + args + WorkDir`; no change for clean PTY case |
| Badge color identity | Frontend (source of truth) | CSS (consumers) | `agentBadge.ts` switch is single source; `style.css` consumes via modifier key |
| Picker entry ("Google Antigravity") | Frontend GUI + web | CLI | New-session modal picker; `agenthub new agy` CLI; both flow from `knownCLIs` detection |
| Settings → Paths override | Backend + Frontend | — | `CLIPaths` in `daemonSettings`, mirrored by existing per-agent UI |
| Status classification | Backend / daemon | — | `PatternsForCLI("agy")` in `internal/status/detector.go` |
| Auth guidance (if modal needed) | Frontend | — | First-run UX only; D-13 conditional |

---

## Standard Stack

This phase adds no new external packages. It reuses existing project stack entirely.

### Core (reused)
| Component | File(s) | Role |
|-----------|---------|------|
| `knownCLIs` CLISpec registry | `internal/pty/detect.go` | Single insertion point for new agent |
| `AugmentServicePath` / `platformExtraBins` | `internal/daemon/path.go`, `path_windows.go` | PATH augmentation for daemon service launches |
| `agentBadgeModifier` | `frontend/src/lib/agentBadge.ts` | Badge modifier key (single source of truth) |
| Three CSS badge blocks | `frontend/src/style.css` ~1711, ~4803, ~5020 | Tab dot, card spine, card chip |
| `daemonSettings.CLIPaths` | `internal/daemon/engine.go` | Per-agent stored path override |
| `PatternsForCLI` / `FallbackPatterns` | `internal/status/detector.go` | Status heuristic dispatch |

### No new npm/Go packages required.

---

## Package Legitimacy Audit

No external packages are introduced in this phase. Section not applicable.

---

## Architecture Patterns

### System Architecture Diagram

```
User terminal
     │
     ▼
AgentHub daemon ──────────────────────────────────────────────────────┐
  AugmentServicePath() adds to PATH:                                   │
    ~/.local/bin (macOS/Linux agy install)                             │
    /opt/homebrew/bin  /usr/local/bin (macOS Homebrew)                │
    %LOCALAPPDATA%\agy\bin  ← NEW (Windows)                           │
     │                                                                 │
     ▼                                                                 │
  DetectCLIs() → exec.LookPath("agy")   ◄── knownCLIs entry NEW      │
     │                                                                 │
     ▼                                                                 │
  Frontend picker: "Google Antigravity"                               │
     │                                                                 │
     ▼                                                                 │
  CreateSession(cli="agy", workDir=...)                                │
     │                                                                 │
     ├── ensureOpenCodeTUIConfig? → NO (no shim needed for clean PTY) │
     │                                                                 │
     ▼                                                                 │
  Spawn PTY: agy + args → relay hub                                   │
     │                                                                 │
     ├── status.Watch(cli="agy") → PatternsForCLI("agy")              │
     │     currently → FallbackPatterns (conservative running default)│
     │     may add agy-specific patterns after live sampling           │
     │                                                                 │
     ▼                                                                 │
  GUI tab: badge dot (.tab__agent-badge--agy, bg #ff9e64)             │
  Hub card: spine (.hub-card[data-agent="agy"], border-left #ff9e64)  │
  Hub card: chip (.hub-card[data-agent="agy"] .hub-card__badge)       │
```

### Recommended Project Structure Changes

```
internal/pty/
  detect.go           add {Name: "agy", DisplayName: "Google Antigravity"} to knownCLIs
  detect_test.go      add "agy" to TestKnownCLIs_HasExpectedEntries expected list

internal/daemon/
  path_windows.go     add filepath.Join(local, "agy", "bin") to platformExtraBins

frontend/src/lib/
  agentBadge.ts       add 'agy' case to switch → return 'agy'
  agentBadge.test.ts  add test: returns 'agy' for cli='agy'

frontend/src/style.css
  ~1711 block         add .tab__agent-badge--agy   { background: #ff9e64; } /* agy — orange */
  ~4803 block         add .hub-card[data-agent="agy"] { border-left: 3px solid #ff9e64; }
  ~5020 block         add .hub-card[data-agent="agy"] .hub-card__badge { color: #ff9e64; border-color: rgba(255,158,100,0.45); }

frontend/src/components/__tests__/
  style.hub.test.ts   extend data-agent presence list to include "agy"
```

### Pattern 1: CLISpec Row Addition (single insertion point)

```go
// Source: internal/pty/detect.go — existing knownCLIs slice
var knownCLIs = []CLISpec{
    {Name: "claude",      DisplayName: "Claude Code"},
    {Name: "codex",       DisplayName: "OpenAI Codex"},
    {Name: "gemini",      DisplayName: "Gemini CLI"},
    {Name: "opencode",    DisplayName: "OpenCode"},
    {Name: "agy",         DisplayName: "Google Antigravity"}, // NEW — D-04: separate entry, not Gemini variant
}
```

**Key:** `agy` (binary name, not `antigravity`). Per D-09: badge key = real binary name.

### Pattern 2: Windows PATH Entry (platformExtraBins)

```go
// Source: internal/daemon/path_windows.go — existing platformExtraBins func
if local := os.Getenv("LOCALAPPDATA"); local != "" {
    paths = append(paths, filepath.Join(local, "agy", "bin")) // agy CLI installer
    // ... existing entries ...
}
```

### Pattern 3: Badge Modifier (agentBadge.ts)

```typescript
// Source: frontend/src/lib/agentBadge.ts — existing switch
case 'agy':
  return 'agy'
```

### Pattern 4: Status Pattern Registration (detector.go)

Current `PatternsForCLI` only has a Claude case. For `agy`, use `FallbackPatterns()` until
live output can be sampled:

```go
// Source: internal/status/detector.go — PatternsForCLI
func PatternsForCLI(cliName string) PatternSet {
    switch cliName {
    case "claude":
        return DefaultClaudePatterns()
    case "agy":
        return DefaultAgyPatterns() // or FallbackPatterns() if not yet sampled
    default:
        return FallbackPatterns()
    }
}
```

The `agy` TUI prompt is `> ` (idle) and `! ` (shell mode). Thinking state shows
`"▸ Thought for Xs, Y.Zk tokens"`. These are the candidate patterns for a
`DefaultAgyPatterns()` function — confirmed from Medium tutorial. [CITED]

### Anti-Patterns to Avoid

- **Using `antigravity` as the agent key:** The CLI binary is `agy`. Use `agy` as the key everywhere.
- **Adding `antigravity` to PATH entries:** The Windows installer puts the binary at `%LOCALAPPDATA%\agy\bin\agy.exe`, not `%LOCALAPPDATA%\Antigravity\`. The parent dir is `agy`, not `Antigravity`.
- **Folding into Gemini CLI:** D-04 explicitly forbids treating `agy` as a Gemini variant even though both are Google/Gemini-backed. Separate `CLISpec` row, separate badge.
- **Adding engine shim preemptively:** Expect no engine changes for the clean PTY case. Add a shim only if live testing (post-waitlist) surfaces a concrete quirk.

---

## WCAG-AA Contrast Analysis

**Required by D-07: must verify at hex level, never by eye.**

Badge color: `#ff9e64` (RGB 255, 158, 100). Relative luminance: 0.4663.

| Background | Hex | Luminance | Contrast ratio | Threshold | Result |
|------------|-----|-----------|----------------|-----------|--------|
| Dark hub-surface | `#16181f` | 0.0107 | **8.72:1** | 3:1 (UI component) | PASS |
| Dark hub-surface-elevated | `#1c1e28` | 0.0135 | **8.16:1** | 3:1 | PASS |
| Light hub-surface | `#ffffff` | 1.0000 | **2.03:1** | 3:1 | FAIL |
| Light hub-surface-elevated | `#ececf0` | 0.8612 | **1.73:1** | 3:1 | FAIL |

**Dark theme: PASS. Light theme: FAIL.**

**Systemic context (critical for the planner):** Every existing agent badge hex also fails light
theme. Computed on white `#ffffff`:

| Agent | Hex | Contrast on white |
|-------|-----|-------------------|
| claude | `#7aa2f7` | 2.52:1 FAIL |
| opencode | `#9ece6a` | 1.83:1 FAIL |
| codex | `#bb9af7` | 2.31:1 FAIL |
| gemini | `#2ac3de` | 2.11:1 FAIL |
| cursor | `#e0af68` | 2.00:1 FAIL |
| aider | `#f7768e` | 2.65:1 FAIL |
| shell | `#89ddff` | 1.52:1 FAIL |
| **agy** | **`#ff9e64`** | **2.03:1 FAIL** |

The codebase has no light-theme per-agent badge overrides. The COLORBLIND-SAFE comments in
`style.css` note that the spine / dot is a **secondary identity cue** — the agent is also shown
as a text chip (`{cli}` label) in the card header, and by display name in the picker. Color is
never the sole identifier.

**D-07 compliance posture (what the planner must do):**
1. Record the hex-verified contrast numbers in a source comment (as existing COLORBLIND-SAFE
   comments do). Do not assert "WCAG AA" when the light-theme fails.
2. The CSS comment for the `agy` badge should read:
   `/* agy — orange; dark: 8.72:1 AA PASS; light: 2.03:1 FAIL (same gap as all existing agents — text chip carries identity) */`
3. This is the D-07 deliverable: honest hex-level documentation. The planner should NOT choose a
   different color (D-06 locks `#ff9e64`). The planner should NOT claim AA compliance in light
   theme — instead, document the gap accurately.
4. A future phase can address the systemic light-theme badge-color gap for all agents together.
   It is out of scope for Phase 149 per D-05 (no redesign of existing structure).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Agent binary discovery | Custom file-system scanner | `exec.LookPath` via existing `DetectCLI` pattern | Already correct; `knownCLIs` row is the only change |
| PATH discovery for daemon service | Custom shell-init sourcing | `AugmentServicePath()` in `path.go` | Already handles all platforms; add one Windows entry |
| Badge color dispatch | Inline hex in components | `agentBadge.ts` switch | Single source of truth; components consume modifier class |
| Status classification | Polling agent process | `status.Watch` + `PatternSet` | Already wired for all agents |
| Agent argument passthrough | Custom CLI parsing per agent | Existing engine `args` field + Settings per-agent memory | Already wired |

---

## Runtime State Inventory

This is not a rename/refactor phase. Section not applicable.

---

## Common Pitfalls

### Pitfall 1: Using `antigravity` as the binary/key name
**What goes wrong:** `exec.LookPath("antigravity")` finds nothing (or finds the desktop app symlink, not the CLI). Tests fail. Badge CSS never matches.
**Why it happens:** The product is called "Google Antigravity" but the binary is `agy`.
**How to avoid:** Key is `agy` everywhere — `CLISpec.Name`, `agentBadgeModifier` case, CSS `data-agent` attribute, TESTING.md references.
**Warning signs:** `TestKnownCLIs_HasExpectedEntries` will fail if you add the wrong name.

### Pitfall 2: Forgetting the Windows PATH entry
**What goes wrong:** `agy` is not found when AgentHub runs as a Windows service/startup app because `%LOCALAPPDATA%\agy\bin` is not in the daemon's PATH.
**Why it happens:** `platformExtraBins` is Windows-only; it's easy to test only on macOS where `~/.local/bin` is already covered.
**How to avoid:** Add `filepath.Join(local, "agy", "bin")` to `platformExtraBins()` in `path_windows.go`. Write a unit test (mirror existing `path_windows_test.go` pattern).
**Warning signs:** Windows CI passing all Go tests but users report `agy not found` on Windows.

### Pitfall 3: Forgetting to update `TestKnownCLIs_HasExpectedEntries`
**What goes wrong:** The test asserts `len(knownCLIs) == 4` and lists `["claude", "codex", "gemini", "opencode"]`. Adding `agy` without updating the test will fail CI.
**How to avoid:** Update the `expected` slice to 5 entries and add `"agy"`.

### Pitfall 4: Missing one of the three CSS badge sites
**What goes wrong:** The tab dot shows the badge color but the hub card spine doesn't (or vice versa), creating a drift between surfaces. D-08 requires all three sites updated in lockstep.
**How to avoid:** Search for all three patterns before committing: `.tab__agent-badge--agy`, `.hub-card[data-agent="agy"]`, `.hub-card[data-agent="agy"] .hub-card__badge`.

### Pitfall 5: Status classifier stays on FallbackPatterns indefinitely
**What goes wrong:** The card always shows "running" even when `agy` is idle at its `>` prompt.
**Why it happens:** `PatternsForCLI` defaults to `FallbackPatterns()` for unknown CLIs.
**How to avoid:** Add a `case "agy"` to `PatternsForCLI` with at least the idle pattern
(`> ` or `>\s*$` at end of tail) derived from the confirmed TUI prompt. Full tuning deferred
to post-waitlist live testing, but a minimal idle pattern can be included now.

### Pitfall 6: Auth-flow URL wrapping in narrow terminals (cosmetic but confusing)
**What goes wrong:** First-time users running `agy auth login` inside AgentHub see a broken/wrapped URL and think auth failed.
**Why it happens:** `agy` hard-wraps long OAuth URLs to the detected console column width.
**How to avoid:** The auth guidance modal (if implemented per D-13) should say:
"Run `agy auth login` in a separate wide terminal before using in AgentHub." This is a known upstream issue (#43, #315 in the antigravity-cli repo).

---

## Code Examples

### Adding to knownCLIs (detect.go)
```go
// Source: internal/pty/detect.go line ~25
var knownCLIs = []CLISpec{
    {Name: "claude",   DisplayName: "Claude Code"},
    {Name: "codex",    DisplayName: "OpenAI Codex"},
    {Name: "gemini",   DisplayName: "Gemini CLI"},
    {Name: "opencode", DisplayName: "OpenCode"},
    {Name: "agy",      DisplayName: "Google Antigravity"}, // Phase 149
}
```

### Windows PATH entry (path_windows.go)
```go
// Source: internal/daemon/path_windows.go — platformExtraBins
if local := os.Getenv("LOCALAPPDATA"); local != "" {
    paths = append(paths, filepath.Join(local, "agy", "bin")) // agy CLI — %LOCALAPPDATA%\agy\bin\agy.exe
    paths = append(paths, filepath.Join(local, "pnpm"))
    // ... existing entries ...
}
```

### CSS additions (style.css — three sites)
```css
/* Site 1: tab dot ~line 1717 */
.tab__agent-badge--agy    { background: #ff9e64; } /* agy — orange; dark: 8.72:1 AA PASS; light: 2.03:1 FAIL (same gap as all existing agents — text chip carries identity) */

/* Site 2: hub card spine ~line 4810 */
.hub-card[data-agent="agy"]    { border-left: 3px solid #ff9e64; } /* agy — orange */

/* Site 3: hub card badge chip ~line 5026 */
.hub-card[data-agent="agy"]    .hub-card__badge { color: #ff9e64; border-color: rgba(255, 158, 100, 0.45); }
```

### agentBadge.ts addition
```typescript
// Source: frontend/src/lib/agentBadge.ts
case 'agy':
  return 'agy'
```

### Minimal status patterns for agy (detector.go)
```go
// Source: internal/status/detector.go — new function
func DefaultAgyPatterns() PatternSet {
    return PatternSet{
        Idle: []*regexp.Regexp{
            // agy idle prompt is "> " at the start of a new input line
            regexp.MustCompile(`>\s*$`),
        },
        Waiting: []*regexp.Regexp{
            // agy uses y/n confirmation for destructive operations (same as claude)
            regexp.MustCompile(`\[y/n\]|\[Y/n\]|\[y/N\]`),
        },
        // Working: no known reliable indicator yet; FallbackPatterns default (running) applies
    }
}
```

**NOTE:** These patterns are [ASSUMED] from TUI description. Verify and tune after live access.

### OpenCode shim precedent (for reference — D-13, only if needed)
```go
// Source: internal/daemon/engine.go line ~80 — only apply if verification surfaces a quirk
func ensureOpenCodeTUIConfig(dir string) string {
    path := filepath.Join(dir, "opencode-tui.json")
    content := []byte("{\"$schema\":\"https://opencode.ai/tui.json\",\"theme\":\"system\"}\n")
    _ = os.WriteFile(path, content, 0644)
    return path
}
```
Currently no evidence that `agy` requires a managed config shim. The clean PTY path applies.

---

## State of the Art

| Old Approach | Current Approach | Impact for Phase 149 |
|--------------|------------------|---------------------|
| Gemini CLI (`gemini`) — the last Google-adjacent agent added | `agy` — successor to Gemini CLI, Go-based, adds subagents + slash commands | Separate agent entry per D-04; `gemini` stays unchanged |
| Node.js Gemini CLI internals | `agy` rebuilt in Go | Faster startup, smaller memory footprint; no integration impact |
| OAuth-only auth (Gemini CLI era) | OAuth + `ANTIGRAVITY_API_KEY` env var | Future auth-flow modal can offer both paths |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `agy` desktop-app-independent (one source contradicts four) | Spike Fact 2 | If IDE daemon required: D-02 outcome; phase pauses |
| A2 | `%LOCALAPPDATA%\agy\bin` is the canonical Windows install path (multiple sources agree but none are the official Windows installer documentation page) | Standard Stack, Pitfall 2 | Wrong Windows PATH entry; `agy` not found on Windows until user manually adds to PATH |
| A3 | `ANTIGRAVITY_API_KEY` env var auth is GA for all account tiers | Spike Fact 4 | Some users may still need OAuth-only; guidance modal text needs adjustment |
| A4 | `> ` prompt pattern is reliable for idle detection across agy versions | Code Examples (status patterns) | Status classifier may misread idle as running; tune after live access |
| A5 | No managed config shim required (clean PTY path) | Don't Hand-Roll | If agy needs a theme or config file: add engine shim per D-13 playbook |

---

## Open Questions

1. **Live output pattern sampling for status classifier**
   - What we know: `agy` TUI shows `>` prompt idle, `! ` shell mode, `"▸ Thought for Xs"` during reasoning.
   - What's unclear: exact regex-safe representation after ANSI strip, whether spinner frames appear before the thought summary.
   - Recommendation: Ship minimal `DefaultAgyPatterns` with `> \s*$` idle rule. Add TESTING.md M-15 note to tune after live access.

2. **Homebrew cask naming (`antigravity-cli` vs `antigravity`)**
   - What we know: `brew install antigravity-cli` installs the CLI as `agy`; `brew install --cask antigravity` installs the desktop app (may also write an `agy` binary, causing conflict).
   - What's unclear: Whether `brew install antigravity-cli` is the formula or cask (sources disagree on formula vs cask classification).
   - Recommendation: PATH augmentation probes for `~/.local/bin/agy`, `/opt/homebrew/bin/agy`, `/usr/local/bin/agy` by file system existence — it doesn't care which install method was used. The planner need not resolve the formula-vs-cask question.

3. **Auth guidance modal scope**
   - What we know: OAuth URL wrapping is a cosmetic PTY issue (issues #43, #315); auth works functionally.
   - What's unclear: Whether the first-run experience inside AgentHub is smooth enough without a modal, or whether users will be confused.
   - Recommendation: Defer the auth modal to post-waitlist live testing. Add a README note. If post-live UAT (M-15) surfaces confusion, add the modal as a gap-closure plan.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `agy` binary | Live REPL launch (M-15 only) | Unknown (waitlist) | — | Source-level acceptance (D-03) |
| Go toolchain | Backend unit tests | ✓ | (project standard) | — |
| vitest | Frontend unit tests | ✓ | (project standard) | — |
| Homebrew (macOS) | PATH augmentation testing | ✓ | (dev machine) | — |

**Missing dependencies with no fallback for D-03 source acceptance:** None — all source/unit tests run without `agy` installed.

**Missing dependencies blocking live UAT:** `agy` binary (waitlist). Tracked as M-15 manual checklist item.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go framework | `testing` (stdlib) |
| Go config | `go test ./...` |
| TS framework | vitest |
| TS config | `frontend/vitest.config.ts` |
| Quick run (Go) | `go test ./internal/pty/... ./internal/daemon/... ./internal/status/...` |
| Quick run (TS) | `cd frontend && pnpm test --run` |
| Full suite | `go test ./... && cd frontend && pnpm test --run && tsc --noEmit && vite build` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File |
|--------|----------|-----------|-------------------|------|
| AGENT-01 | `knownCLIs` contains `agy` entry | unit (Go) | `go test ./internal/pty/ -run TestKnownCLIs` | `internal/pty/detect_test.go` |
| AGENT-01 | DetectCLIs finds `agy` stub on PATH | unit (Go) | `go test ./internal/pty/ -run TestDetectCLIs` | `internal/pty/detect_test.go` |
| AGENT-01 | `agentBadgeModifier('agy') === 'agy'` | unit (TS) | `pnpm test --run agentBadge` | `frontend/src/lib/agentBadge.test.ts` |
| AGENT-01 | `.tab__agent-badge--agy` CSS rule present | unit (TS) | `pnpm test --run style.hub` | `frontend/src/components/__tests__/style.hub.test.ts` |
| AGENT-01 | `.hub-card[data-agent="agy"]` spine rule present | unit (TS) | `pnpm test --run style.hub` | `frontend/src/components/__tests__/style.hub.test.ts` |
| AGENT-01 | Windows PATH augmentation includes `%LOCALAPPDATA%\agy\bin` | unit (Go, Windows) | CI Windows matrix | `internal/daemon/path_windows_test.go` |
| AGENT-01 | Live REPL launch (waitlist gate) | manual | M-15 checklist | `TESTING.md Section 5` |

### Sampling Rate
- **Per task commit:** `go test ./internal/pty/ && cd frontend && pnpm test --run agentBadge`
- **Per wave merge:** `go test ./... && cd frontend && pnpm test --run && tsc --noEmit`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/pty/detect_test.go` — extend `TestKnownCLIs_HasExpectedEntries` to include `"agy"` (5 entries)
- [ ] `frontend/src/lib/agentBadge.test.ts` — add `returns 'agy' for cli='agy'` test
- [ ] `frontend/src/components/__tests__/style.hub.test.ts` — add `"agy"` to data-agent presence assertion
- [ ] `internal/daemon/path_windows_test.go` — add test that `platformExtraBins` includes `agy\bin` path

---

## Security Domain

This phase has no new authentication flows, no new network endpoints, and no new input validation
surfaces. The existing security model (capability-based session auth, PTY relay, CORS proxy)
applies to `agy` identically to all existing agents — no new ASVS categories are introduced.

The only security-adjacent note: if the auth-guidance modal (D-13) is implemented, it must not
display or store the user's Google credentials — only direct the user to run `agy auth login`
externally. This is trivially satisfied by the proposed UX (text instructions, no credential
input fields).

---

## Sources

### Primary (HIGH confidence)
- [github.com/google-antigravity/antigravity-cli](https://github.com/google-antigravity/antigravity-cli) — official GitHub repository; binary name, install script, standalone confirmation, auth flow
- [codelabs.developers.google.com/antigravity-cli-hands-on](https://codelabs.developers.google.com/antigravity-cli-hands-on) — official Google Codelabs; interactive TUI behavior, bare `agy` invocation, install paths
- `internal/pty/detect.go` — current knownCLIs registry (codebase read)
- `internal/daemon/path.go`, `path_windows.go` — PATH augmentation architecture (codebase read)
- `frontend/src/lib/agentBadge.ts` — badge modifier single source of truth (codebase read)
- `frontend/src/style.css` — all three badge color sites (codebase read)
- `internal/status/detector.go` — status classifier architecture (codebase read)

### Secondary (MEDIUM confidence)
- [dev.to/arindam_1729/antigravity-cli-a-hands-on-guide](https://dev.to/arindam_1729/antigravity-cli-a-hands-on-guide-to-googles-terminal-coding-agent-5bc7) — hands-on guide; install paths, interactive behavior, auth flow
- [medium.com/google-cloud/antigravity-cli-tutorial-series](https://medium.com/google-cloud/antigravity-cli-tutorial-series-12b46cfe3bf2) — tutorial series; TUI prompt strings, thinking state output, install paths
- [pasqualepillitteri.it/en/news/3422/antigravity-cli-agy-install-migrate-gemini-cli](https://pasqualepillitteri.it/en/news/3422/antigravity-cli-agy-install-migrate-gemini-cli) — install guide; Homebrew conflict analysis, auth token path
- [agentpedia.codes/blog/antigravity-cli-deep-dive](https://agentpedia.codes/blog/antigravity-cli-deep-dive) — deep dive; Go implementation, SSH auth details, standalone confirmation
- [blog.davep.org/2026/05/21/antigravity-cli-now-on-homebrew.html](https://blog.davep.org/2026/05/21/antigravity-cli-now-on-homebrew.html) — Homebrew install analysis; conflict with desktop cask
- [formulae.brew.sh/cask/antigravity-cli](https://formulae.brew.sh/cask/antigravity-cli) — Homebrew cask record; version 1.0.10

### Tertiary (LOW confidence — verify at first live access)
- [github.com/google-antigravity/antigravity-cli/issues/43](https://github.com/google-antigravity/antigravity-cli/issues/43) — OAuth URL wrapping PTY issue; confirms auth works functionally
- [github.com/google-antigravity/antigravity-cli/issues/78](https://github.com/google-antigravity/antigravity-cli/issues/78) — API key feature request; confirms OAuth is primary auth method
- Output pattern claims (`> ` idle prompt, `▸ Thought for Xs` thinking state) — derived from tutorial screenshots; treat as [ASSUMED] until verified with live binary

---

## Metadata

**Confidence breakdown:**
- D-01 spike facts: HIGH — multiple independent sources agree on binary name, standalone operation, PTY REPL, and headless auth degradation
- Standard stack: HIGH — all changes are in existing, well-understood files
- Architecture: HIGH — established pattern; one new CLISpec row and three CSS rules
- Status patterns: LOW — pattern strings derived from screenshots, not live binary testing
- Windows PATH: MEDIUM — path `%LOCALAPPDATA%\agy\bin` confirmed by multiple sources but not from the official installer documentation page

**Research date:** 2026-06-22
**Valid until:** 2026-09-22 (stable; fast-moving only if Google releases a major `agy` v2+ with different binary name)
