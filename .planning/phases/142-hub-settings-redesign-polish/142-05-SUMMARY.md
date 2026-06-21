---
phase: 142-hub-settings-redesign-polish
plan: "05"
subsystem: frontend
unplanned: true
kind: post-uat-gap-closure
tags: [css, react, accessibility, colorblind-safe, parity, comp-fidelity]
dependency_graph:
  requires: ["142-04"]
  provides: ["new-session-accent-pill", "card-agent-type-spine", "card-rounded-corners", "card-taller-preview", "card-ia-restructure"]
  affects:
    - "frontend/src/lib/agentBadge.ts"
    - "frontend/src/components/TabBar.tsx"
    - "frontend/src/components/Hub/SessionCard.tsx"
    - "frontend/src/components/Hub/HubPanel.tsx"
    - "frontend/src/style.css"
tech_stack:
  added: []
  patterns: ["data-agent CSS attribute hook", "shared agentBadgeModifier source of truth", "var(--hub-*) token discipline", "prefers-reduced-motion two-block motion contract"]
key_files:
  created:
    - frontend/src/lib/agentBadge.ts
  modified:
    - frontend/src/components/TabBar.tsx
    - frontend/src/components/Hub/SessionCard.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/style.css
commits:
  - "25e2e7b9 fix(hub): comp-fidelity polish for cards + New Session button"
  - "5930ec2f fix(hub): restructure session card layout for clarity"
decisions:
  - "New Session button restyled from POL-03's quiet text affordance to the comp's filled, glowing accent pill (filter bar + empty state). POL-03's 'comp affordance' had been read as the sidebar text style; the comp's prominent CTA is a filled accent pill."
  - "Card left spine + type chip are colored by SESSION TYPE (agent/CLI), NOT status — the user's explicit choice. agentBadgeModifier extracted to lib/agentBadge.ts as the single source so the card spine, card chip, and tab agent-badge dot cannot drift. Palette: claude #7aa2f7, opencode #9ece6a, codex #bb9af7, gemini #2ac3de, cursor #e0af68, aider #f7768e, shell #89ddff, unknown #9aa5ce."
  - "Card mini-preview was useless on TUI agents because the tail fetch was capped at 4 lines (= the bottom footer/input). Raised to 12 lines (max 20), box 88px -> 150px, font 11px -> 10px so real session content shows above the footer."
  - "Card border-radius 10px -> 16px for comp parity."
  - "Card IA restructured for clarity (all test-referenced class names preserved): name centered on the top line between the drag-handle and overflow-menu icons; status left + colored type chip right; origin color-coded (local #9ece6a / remote #7aa2f7, icon carries meaning so colorblind-safe) with uptime+viewers right-aligned on the same line; Open + Share share one actions row; .hub-card__share was unstyled (rendered as a bare default button) and now mirrors .hub-card__open."
verification:
  - "Full frontend suite: 1766/1766 passing across 107 files"
  - "pnpm exec tsc --noEmit: exit 0"
  - "pnpm build: succeeds"
  - "Updated HubPanel.test.tsx assertions (tail fetch 4 -> 12) to match the new line count"
metrics:
  completed: "2026-06-21T23:30:00Z"
  files_changed: 6
---

## Summary

Post-completion, comp-fidelity gap closure for Phase 142, driven by hands-on UAT
against the Issue #78 hub design comp. Phase 142 had already verified PASS
(14/14 must-haves); this work corrects places where the shipped result diverged
from the comp's interaction quality and reorganizes the session card's
information architecture.

This was **not a planned plan** — it is recorded here for traceability of the
two follow-up commits (`25e2e7b9`, `5930ec2f`) that landed on `main` after the
phase was marked complete.

### What changed

1. **New Session button** — quiet borderless text → filled, glowing accent pill
   (filter bar + empty state), mirroring the primary-CTA treatment.
2. **Session-type identity** — card left spine **and** the `{cli}` chip are now
   colored by agent type, matching the tab agent-badge dot; `agentBadgeModifier`
   extracted to `lib/agentBadge.ts` as the single source of truth.
3. **Rounded corners** — `.hub-card` radius 10px → 16px.
4. **Useful preview** — tail fetch 4 → 12 lines, box 88px → 150px, font 11px →
   10px, so TUI-agent content shows above the fixed footer.
5. **Card layout restructure** — centered name on the top line; status + colored
   type chip on one row; color-coded Local/Remote origin with right-aligned
   uptime+viewers; Open + Share as a matched button pair; Share button given
   real styling (was unstyled).

### Known follow-ups (out of scope here)

- CMD +/- does not resize the terminal session font (filed as a GitHub issue;
  the Settings font-size control is the supported path). Out of scope for POL-04,
  whose font-size effect was intentionally left unchanged.
