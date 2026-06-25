# Phase 149: Google Antigravity Agent - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-22
**Phase:** 149-google-antigravity-agent
**Areas discussed:** Availability & acceptance gate, Gemini relationship, Badge color identity, Scope compression

---

## Availability & acceptance gate

### Q1 — How to handle the risk that the CLI may not be installable/standalone?

| Option | Description | Selected |
|--------|-------------|----------|
| Research spike gates planning | Researcher must confirm standalone PTY-capable binary exists before any code; pause + re-scope if not | ✓ |
| Ship entry, defer live verify | Build full integration, accept on source/unit + docs, verify live later | |
| I'll install it first | User obtains access now; live UAT realistic this phase | |

**User's choice:** Research spike gates planning.
**Notes:** Plan-phase blocked until RESEARCH.md answers: (1) binary name + install, (2) runs standalone (no IDE daemon), (3) interactive PTY REPL when launched bare, (4) auth degrades inside PTY. Any "no" → pause, re-scope, comment on #65.

### Q2 — If the CLI exists but is closed-beta/waitlist (no live UAT possible)?

| Option | Description | Selected |
|--------|-------------|----------|
| Proceed, source-level accept | Build full integration; accept on source/unit + docs; live launch becomes TESTING.md M-NN manual item | ✓ |
| Block until installable | No live binary = no ship; pause until publicly installable | |

**User's choice:** Proceed, source-level accept.
**Notes:** Existence-confirmed is enough to build. Live REPL launch deferred to a TESTING.md M-NN manual checklist item ("verify when waitlist access granted") + README waitlist note.

---

## Gemini relationship

### Q1 — How should Antigravity relate to the existing Gemini CLI agent?

| Option | Description | Selected |
|--------|-------------|----------|
| Always separate entry | Own top-level agent regardless of backend | ✓ |
| Separate IF distinct binary | Separate only if real binary; fold into Gemini if a wrapper | |

**User's choice:** Always separate entry.
**Notes:** "Google Antigravity" is always first-class, even if it wraps `gemini`. Overrides #65's "treat as Gemini variant if wrapper" fallback.

### Q2 — Picker presentation for the two Google/Gemini-backed agents?

| Option | Description | Selected |
|--------|-------------|----------|
| Flat list, distinct names | Append to existing flat list; name/desc/badge carry distinction | ✓ |
| Group under 'Google' | New 'Google' subsection grouping Gemini CLI + Antigravity | |

**User's choice:** Flat list, distinct names.
**Notes:** No grouping UI in v1; least GUI/web churn.

---

## Badge color identity

### Q1 — Approach for the Antigravity badge color?

| Option | Description | Selected |
|--------|-------------|----------|
| Lock TokyoNight orange | Use #ff9e64 (main unused accent); planner verifies WCAG-AA at hex | ✓ |
| Research picks + verifies | Delegate exact hex to research with TokyoNight/WCAG-AA constraint | |

**User's choice:** Lock TokyoNight orange (`#ff9e64`).
**Notes:** Colorblind project — WCAG-AA verified at hex (source), not by eye. Existing hues: claude #7aa2f7, opencode #9ece6a, codex #bb9af7, gemini #2ac3de, cursor #e0af68, aider #f7768e, shell #89ddff. Update agentBadge.ts + 3 style.css blocks. Badge key must match the real binary name from the spike.

---

## Scope compression

### Q1 — If the spike surfaces quirks (auth modal, config shim, classifier tuning)?

| Option | Description | Selected |
|--------|-------------|----------|
| Absorb if small, else defer | Mechanical quirks in 149; heavy auth-UX → follow-up phase | |
| Everything in 149 | Handle all surfaced quirks in this one phase, however large | ✓ |
| MVP only, defer all quirks | Clean path only; any quirk → follow-up phase | |

**User's choice:** Everything in 149.
**Notes:** Phase 149 = a fully working Antigravity agent including any config shim, auth modal, or classifier tuning. No follow-up splitting.

---

## Claude's Discretion

- PATH-augmentation install locations (derived from spike findings).
- Per-agent argument-memory wiring and Settings → Paths override field (mirror existing agents).
- Engine changes expected to be none for the clean PTY case; add a shim only if D-13 quirks require it.

## Deferred Ideas

- "Google" picker subsection grouping (declined for v1).
- Gemini launch-mode-variant treatment (rejected; recorded as considered-and-declined from #65).

## Carried Forward (not re-asked)

- TUI dropped in v4.0 — ignore #65's TUI sections; parity = GUI/CLI/web.
