# Phase 172: Hub-card layout & badge refinement - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-07
**Phase:** 172-hub-card-layout-badge-refinement
**Areas discussed:** Quiet-chip style, Status placement, Exposure badges, Rhythm & meta, Mockup process

---

## Quiet-chip style

| Option | Description | Selected |
|--------|-------------|----------|
| Outlined ghost pills | Agent + origin become bordered/transparent pills (origin pillified to match today's /bin/zsh badge). INTERNET the only filled chip → pops. | ✓ |
| Text + middot | Agent/origin plain muted inline text separated by ·; only exposure badge is a pill. | |
| Subtle-filled pills | Agent/origin get soft surface-elevated fill (no border); closer to the exposure fill, reduces contrast gap. | |

**User's choice:** Outlined ghost pills
**Notes:** Keeps a consistent "chip family" with the existing agent badge; contrast against INTERNET/FULL ACCESS is structural (outline vs fill), which also serves the colorblind requirement.

---

## Status placement

| Option | Description | Selected |
|--------|-------------|----------|
| Stays primary, above chip row | Status keeps its own prominent top line with spin + attention pulse; chip row sits below as secondary metadata. | ✓ |
| Status becomes a chip in the row | Status joins the unified row as first chip (tightest rhythm, but status loses prominence). | |

**User's choice:** Stays primary, above chip row
**Notes:** Status is the #1 state signal and carries the spin animation + attention pulse; keep it dominant.

---

## Exposure badges (INTERNET + FULL ACCESS)

| Option | Description | Selected |
|--------|-------------|----------|
| Join chip row, both coexist | Exposure badges become the prominent right end of the chip row; FULL ACCESS shows alongside INTERNET (read-many/write-one). | ✓ |
| Keep on own line below | Consolidate only agent/origin/status; leave exposure badges on a dedicated line. | |
| FULL ACCESS supersedes INTERNET | Show only FULL ACCESS when write active; INTERNET alone when read-only. | |

**User's choice:** Join chip row, both coexist
**Notes:** Matches current logic; FULL ACCESS's notched-shape/heavier-weight distinction from INTERNET is a Phase-171 colorblind guarantee and must be preserved.

---

## Rhythm & meta

| Option | Description | Selected |
|--------|-------------|----------|
| Muted meta line under chip row | uptime + viewers + remote conn on one muted line below chip row (tightened row2-meta). | ✓ |
| Fold stats into the chip row | uptime/viewers as trailing muted items on the chip row (most compact, mixes identity with live numbers). | |
| Leave meta rows unchanged | Only restyle chips; keep meta rows as today. | |

**User's choice:** Muted meta line under chip row
**Notes:** Clean separation of identity chips vs live stats; tighten vertical rhythm to address the "loosely stacked" critique.

---

## Mockup process

| Option | Description | Selected |
|--------|-------------|----------|
| Variations within outlined direction | Mock 2-3 refinements of the chosen direction (border weight/radius, gap, meta density, long-hostname + both-badges wrapping). | ✓ |
| Still compare all 3 chip styles | Render outlined vs text+middot vs subtle-filled side by side to re-check the direction. | |
| Skip mockups, go to plan | Implement outlined pills directly, no visual pre-check. | |

**User's choice:** Variations within outlined direction
**Notes:** Direction is locked; mockups pin the final polish (spacing, wrap behavior) before code. Also flagged the "Ready for context" gate — user selected Ready for context.

---

## Claude's Discretion

- Exact border weight, corner radius, gap, and padding for the outlined pills — pinned via mockups.
- Long remote-hostname handling in its origin pill (truncate/ellipsis vs wrap).
- Whether origin pill keeps color-coded text (green local / blue remote) or goes fully muted — resolve visually, color reinforcement-only.

## Deferred Ideas

None — discussion stayed within phase scope (frontend card polish only).
