# Sketch Manifest

## Design Direction

Phase 172 — **Hub session card layout & badge refinement** (frontend-only, AgentHub v4.2).
Consolidate the card's three inconsistent metadata treatments into ONE consistent chip row
`agent · origin · exposure` with tighter vertical rhythm. Chip style is **locked**: quiet
identity chips are **outlined ghost pills**; the exposure badges (INTERNET / FULL ACCESS) are
the only **filled** chips, so they pop by structural contrast (fill+shape vs outline) — which
also serves the colorblind requirement ([[user_colorblind]]: verify hex at source, never by eye).
Status stays the primary top line; a muted meta line (uptime · viewers · Connected) sits below
the chip row. Mockups reuse the real app's tokens, class names, and badge geometry so a winner
translates straight to `SessionCard.tsx` + `style.css`.

## Reference Points

- The running AgentHub Hub (`frontend/src/components/Hub/SessionCard.tsx`, `frontend/src/style.css`).
- The critique image: today's card stacks `Running` (icon+text), `Local` (colored text), `/bin/zsh`
  (outlined pill), and `INTERNET` (filled green pill) on separate loosely-spaced rows.
- Phase 171 FULL ACCESS badge — notched clip-path + heavier weight, must be preserved.

## Sketches

| # | Name | Design Question | Winner | Tags |
|---|------|----------------|--------|------|
| 001 | hub-card-chip-row | How should the `agent · origin · exposure` chip row look, and how should it handle a long remote hostname + coexisting exposure badges? | **B** — rounded-rect chips (7px), muted origin, exposure forced onto its own right-aligned line | hub-card, chips, badges, exposure, colorblind, phase-172 |
