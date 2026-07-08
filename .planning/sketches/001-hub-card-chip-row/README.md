---
sketch: 001
name: hub-card-chip-row
question: "How should the consolidated agent · origin · exposure chip row look, and how should it handle a long remote hostname + both exposure badges coexisting?"
winner: "B"
tags: [hub-card, chips, badges, exposure, colorblind, phase-172]
---

# Sketch 001: Hub Card Chip Row

## Design Question

Phase 172 consolidates the Hub session card's three inconsistent metadata treatments
(status `Running` icon+text · `/bin/zsh` outlined pill · `Local` colored text · `INTERNET`
filled pill on its own row) into ONE consistent chip row — `agent · origin · exposure` —
with tighter vertical rhythm. The chip-STYLE direction is already locked (outlined ghost
pills, per 172-CONTEXT.md D-01/D-07). These variants pin the remaining discretionary
decisions:

- **Radius:** full pill vs rounded-rect
- **Long remote hostname:** wrap the row vs a dedicated exposure line vs truncate/ellipsis
- **Origin color:** fully muted vs color-coded (green local / blue remote) reinforcement
- **Coexist wrap:** how INTERNET + FULL ACCESS behave together on a narrow (240–260px) card

Locked & carried forward, NOT re-litigated here: status stays the primary top line (D-03);
exposure badges are the only FILLED chips and pop by contrast (D-01); INTERNET + FULL ACCESS
coexist, read-many/write-one (D-05); FULL ACCESS keeps its load-bearing notched clip-path +
heavier weight (Phase-171 colorblind guarantee); muted meta line below the chip row (D-06).

## How to View

```
open .planning/sketches/001-hub-card-chip-row/index.html
```

Use the top tabs to switch variants; the bottom-right toolbar toggles Dark/Light (verify the
exposure badges stay unmistakable in both — colorblind constraint, [[user_colorblind]]).
Five real card states render per variant: shell+INTERNET (the critique image), claude+INTERNET+FULL
ACCESS (coexist), a long remote hostname, a needs-input baseline, and an error-exit card.

## Variants

- **A: Pill · wrap-as-unit · muted origin** — Full-pill chips (reuses today's `.hub-card__badge`
  geometry exactly = path of least resistance). Origin fully muted. Row wraps; the filled exposure
  cluster stays glued together (`margin-left:auto`) and wraps as one unit to line 2 when narrow.
- **B: Rounded-rect · dedicated exposure line · roomier** — ★ **WINNER**. 7px rounded-rect chips, more
  padding, 8px gap. Exposure cluster is forced onto its own right-aligned line so `agent · origin` always
  share one clean line. Most predictable with long hostnames / both badges; costs one extra line of
  height. Chosen for its guaranteed-clean quiet-chip line and stable behavior when INTERNET + FULL ACCESS
  coexist — no wrap surprises, no clipping.
- **C: Pill · single line, never wraps · color-coded origin** — Full-pill chips on a `nowrap` row;
  the origin pill truncates with an ellipsis so everything stays one line. Origin keeps color-coded
  text as reinforcement. Most compact / most stable height — BUT reveals a failure mode: when INTERNET
  + FULL ACCESS coexist on a narrow card, the exposure cluster gets crushed and FULL ACCESS clips
  (see the oauth-rework card). Truncating the origin doesn't recover enough width for two filled badges.

## What to Look For

1. **oauth-rework card (INTERNET + FULL ACCESS):** which wrap behavior reads cleanest — A's
   cluster-wraps-as-unit, B's dedicated line, or C's clipping? This is the decisive case.
2. **db-migration card (long `.ts.net` hostname):** truncate (C) vs wrap (A) vs own-line (B).
3. **Radius feel:** pill (A/C) vs rounded-rect (B) against the card's 16px corners.
4. **Origin treatment:** muted (A/B) vs color-coded (C) — does the color add useful signal or noise?
5. **Vertical rhythm:** does the quiet chip row + muted meta line feel tighter than today's loose stack?
6. **Exposure prominence:** in every variant, do the filled INTERNET/FULL ACCESS chips still pop as
   the loudest thing on the card (the whole point — "quieter neighbors make INTERNET pop MORE")?
