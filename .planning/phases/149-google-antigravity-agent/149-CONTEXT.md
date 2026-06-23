# Phase 149: Google Antigravity Agent - Context

**Gathered:** 2026-06-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Make the **Google Antigravity CLI a selectable, launchable agent** in AgentHub —
a first-class peer of Claude Code, OpenAI Codex, Gemini CLI, and OpenCode.
Delivers requirement **AGENT-01** (#65).

In scope:
- Detection: add Antigravity to `knownCLIs` (`internal/pty/detect.go`) + PATH
  augmentation for its install locations.
- Cross-surface entry across **GUI / CLI / web** (TUI is dropped in v4.0 — see
  carry-forward below).
- Distinct status badge color, WCAG-AA verified at hex.
- Settings → Paths binary-override field, consistent with existing agents.
- Whatever quirk-handling the verification spike surfaces — config shim, auth
  modal, status-classifier tuning — is **all handled inside this one phase**
  (see D-13).
- Documentation: README agent list, CHANGELOG, release notes.

Out of scope:
- TUI surface (v4.0 dropped the TUI; parity = GUI/CLI/web only).
- A "Google" grouping/subsection in the agent picker (flat list in v1).
- New top-level capabilities unrelated to launching Antigravity.

**Hard gate:** This phase does not proceed to a normal build until a research
spike confirms a standalone, PTY-capable Antigravity CLI actually exists. If it
doesn't, the phase pauses and re-scopes (see D-01..D-03).
</domain>

<decisions>
## Implementation Decisions

### Availability & acceptance gate
- **D-01 (gate plan-phase on a verification spike):** Before any integration
  code, the researcher MUST confirm four facts. The plan is BLOCKED until
  `RESEARCH.md` answers all of them:
  1. **Binary name + install method** per platform (macOS / Linux / Windows).
  2. **Runs standalone** — no Antigravity IDE daemon required as a backend.
  3. **Interactive PTY REPL** when launched bare (like `claude` / `codex` /
     `gemini`), not IDE-only or requiring a non-degradable subcommand.
  4. **Auth flow degrades inside a PTY** (browser-loopback OAuth completes, or a
     documented "authenticate in a standalone terminal first" path exists).
- **D-02 (any "no" → pause + re-scope + comment on #65):** If the CLI is
  IDE-companion-only, doesn't exist as a distinct binary, or can't run in a PTY,
  STOP. Do not build an agent entry for a binary that can't launch. Post the
  finding on GitHub issue #65 and re-evaluate the milestone shape with the user.
- **D-03 (exists-but-waitlisted → proceed with source-level acceptance):** If
  the spike confirms a standalone PTY-capable CLI EXISTS but it is
  closed-beta/waitlist so it can't be installed for live UAT this phase:
  - Build the full integration anyway.
  - Acceptance falls back to **source/unit-level**: `knownCLIs` entry + unit
    tests (found/not-found, stored-path override, stale-path filter), badge hex
    WCAG-AA verified at source, picker shows "Google Antigravity" (GUI/web),
    `agenthub new antigravity …` works, README waitlist note present.
  - The **live REPL launch** becomes a manual checklist item (**M-NN in
    TESTING.md Section 5**): "Live Antigravity launch — verify when waitlist
    access is granted." Per the standing Regression Test Convention, add this
    M-NN item and traceability rows in TESTING.md.

### Gemini relationship & picker presentation
- **D-04 (always a separate top-level agent):** Antigravity is its own
  first-class agent entry ("Google Antigravity") with a distinct display name,
  key, and badge — **regardless** of whether the spike finds a distinct binary
  or a `gemini` wrapper. We do NOT fold it into the Gemini CLI agent as a
  launch-mode variant. (This overrides the "treat as Gemini variant if it's a
  wrapper" fallback floated in #65.)
- **D-05 (flat picker list, distinct names):** Append "Google Antigravity" to
  the existing flat agent picker. The display name + description + badge color
  carry the Gemini-vs-Antigravity distinction. **No "Google" grouping
  subsection** in v1 — keep current picker layout, least GUI/web churn.

### Badge color identity
- **D-06 (lock `#ff9e64` — TokyoNight orange):** Antigravity's status-badge
  color is `#ff9e64`, the main TokyoNight accent not yet consumed by an existing
  agent. Clearly separable from the existing blue/cyan/green/purple/gold/red set.
- **D-07 (verify WCAG-AA at hex, not by eye):** This is a colorblind project.
  The planner/executor MUST verify `#ff9e64` meets WCAG-AA contrast against the
  badge backgrounds at the hex level (source check), never by visual judgment.
- **D-08 (update all color sites in lockstep):** Add the new color to every site
  that carries per-agent identity so the tab dot and card spine can't drift:
  - `frontend/src/lib/agentBadge.ts` — add the new case to the `switch`.
  - `frontend/src/style.css` — `.tab__agent-badge--{key}` (~line 1711 block),
    `.hub-card[data-agent="{key}"]` border-left (~4803 block), and
    `.hub-card[data-agent="{key}"] .hub-card__badge` color/border (~5020 block).
- **D-09 (badge key tracks the real binary name):** The `agentBadge.ts` modifier
  and CSS `data-agent` key MUST equal the agent key used in `knownCLIs`, which
  equals the actual binary name confirmed by the spike. If the binary is not
  literally `antigravity`, use the real name as the key (precedent: `opencode`
  key ≠ "OpenCode" display name). Keep the CSS comment hue label accurate.

### Scope of quirk-handling
- **D-13 (everything in Phase 149 — no follow-up splitting):** Whatever the
  verification spike surfaces is handled within this single phase, however large:
  - Managed-config-file shim (analog to OpenCode's `opencode-tui.json`; see
    `internal/daemon/engine.go:73` precedent) if Antigravity needs one.
  - First-run auth-flow guidance modal if OAuth needs special UX.
  - Status-classifier (running/waiting/idle/errored) tuning if Antigravity's
    output patterns confuse the existing heuristic detector.
  - Theme (138-theme smoke) + v3.2 plugin (find bar, OSC 52, image, OSC 9;4
    progress) compatibility verification.
  Phase 149 is "a fully working Antigravity agent," not just the clean path.

### Claude's Discretion
- PATH-augmentation install locations (Homebrew tap, Windows MSI default, Linux
  package paths) — derived from spike findings, planner/executor decides exact
  paths.
- Per-agent argument-memory wiring and Settings → Paths override field — mirror
  the exact mechanism already used for the four existing agents; no novel design.
- Engine changes are expected to be **none** for the clean PTY case (engine
  spawns `cliPath` + args + WorkDir uniformly); only add an engine shim if D-13
  quirks demand it.

### Carry-forward (decided in prior context — do NOT re-ask)
- **TUI is dropped in v4.0.** Issue #65 predates v4.0 and describes a TUI
  new-session modal + session-list badge. **Ignore all TUI sections of #65.**
  Cross-surface parity for this phase is **GUI / CLI / web** only (per ROADMAP
  Success Criterion 3 and the cross-surface-parity standing rule).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirement & scope source
- `.planning/ROADMAP.md` — Phase 149 entry (Goal, Depends-on 143, AGENT-01,
  three success criteria: appears in picker / launches correctly / GUI-CLI-web
  parity).
- `.planning/REQUIREMENTS.md` §AGENT-01 (line ~87) — the single requirement.
- **GitHub issue #65** (scottkw/agenthub) — the authoritative design doc:
  feasibility analysis, the four "Verify Before Planning" facts, proposed
  architecture, acceptance criteria (1–12), and the risk register. **Note the
  TUI references are stale (see carry-forward).** Re-read before planning.

### Integration points (code)
- `internal/pty/detect.go:25` — `knownCLIs` slice; the single insertion point
  for a new agent + the PATH-augmentation logic.
- `internal/daemon/engine.go:73` — OpenCode `opencode-tui.json` managed-config
  precedent; the playbook if Antigravity needs a config shim (D-13).
- `frontend/src/lib/agentBadge.ts` — per-CLI badge color identity (single
  source of truth for the `switch`).
- `frontend/src/style.css` — badge color palette: `.tab__agent-badge--*`
  (~1711), `.hub-card[data-agent="*"]` spine (~4803), `.hub-card__badge`
  text/border (~5020).
- `frontend/src/lib/agentBadge.test.ts`, `frontend/src/components/__tests__/style.hub.test.ts`
  — existing badge tests to extend.

### Testing convention (MANDATORY for this phase)
- `TESTING.md` — Section 2 (Suite Manifest — register new Go/TS test files),
  Section 4 (Traceability map — add AGENT-01 rows; run
  `bash tests/check-traceability-paths.sh` before commit), Section 5 (Manual
  Checklist — add the M-NN live-launch item from D-03), Section 6 (Standing
  Convention). Per repo CLAUDE.md, this phase MUST update TESTING.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `knownCLIs` (`internal/pty/detect.go:25`): authoritative agent list. Adding one
  `CLISpec{Name, DisplayName}` row wires detection (`DetectCLIs`/`DetectCLI`) and
  flows to every surface that reads the detected-CLI list.
- `agentBadge.ts` + the three `style.css` color blocks: per-agent color identity,
  already structured for exactly this kind of additive change (`cursor`/`aider`
  are pre-stubbed here but absent from `knownCLIs` — antigravity is the first
  NEW real consumer, so it must be added to BOTH).
- `opencode-tui.json` shim at `engine.go:73`: ready-made template if a managed
  config file is needed.
- Stale-path filter (v3.3 SHELL-11 precedent, the `claude → /bin/sh` aliasing
  guard) + stored-path override: reuse the same protective unit-test shape.

### Established Patterns
- Engine spawns every agent uniformly (`cliPath` + `args` + `WorkDir` → relay
  hub). Expect **no engine changes** for the clean PTY case.
- Agent key ≠ display name is an accepted precedent (`opencode` → "OpenCode").
- 138 themes apply via standard ANSI and the 8 v3.2 plugins are OSC-driven —
  expected to "just work"; verify via existing matrices, no per-agent code.

### Integration Points
- New agent surfaces wherever the detected-CLI list is consumed: GUI new-session
  modal picker + badge, web picker, CLI (`agenthub new antigravity`,
  `agenthub list`, `agenthub attach`), Settings → Paths override.

</code_context>

<specifics>
## Specific Ideas

- Badge color is pinned exactly: `#ff9e64` (TokyoNight orange).
- Picker presentation pinned: flat list, display name "Google Antigravity",
  distinct description differentiating it from "Gemini CLI".
- Acceptance for the waitlist case is explicitly source-level + a TESTING.md
  M-NN manual item — not a hand-wave; downstream must add that item.

</specifics>

<deferred>
## Deferred Ideas

- **"Google" picker subsection** grouping Gemini CLI + Antigravity — explicitly
  declined for v1 (D-05). Revisit only if the picker gains grouping for other
  reasons later.
- **Gemini launch-mode-variant treatment** — rejected (D-04); noted only so a
  future maintainer knows it was a considered-and-declined path from #65.

None of these block this phase.

</deferred>

---

*Phase: 149-google-antigravity-agent*
*Context gathered: 2026-06-22*
