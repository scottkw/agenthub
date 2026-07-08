---
phase: 172-hub-card-layout-badge-refinement
verified: 2026-07-08T02:21:25Z
status: passed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 172: Hub-card layout & badge refinement Verification Report

**Phase Goal:** Consolidate the Hub session card's three inconsistent metadata treatments
(`Running`/`Local` icon+text, `/bin/zsh` outlined pill, `INTERNET` filled green pill on its
own row) into ONE consistent chip row (agent · origin · exposure) with tighter vertical
rhythm — while deliberately KEEPING the INTERNET/FULL ACCESS exposure badges the one
prominent colored/filled treatment (a security-exposure signal that must stay unmissable +
colorblind-safe). Frontend-only. Built to Sketch 001 Variant B.

**Verified:** 2026-07-08T02:21:25Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Status stays the primary top-line signal above the chip row, keeps spin + attention pulse, NOT chipified (D-03) | VERIFIED | `SessionCard.tsx:488-505` — `.hub-card__row1` unchanged, `STATUS_CONFIG` icon+label+spin logic untouched, rendered above the new `.hub-card__chiprow` (line 520). Confirmed by existing (unbroken) spin/attention-icon unit tests. |
| 2 | Agent + origin render as one consistent row of outlined quiet chips: 7px rounded-rect, 1px border, transparent bg, muted text | VERIFIED | `style.css:5032-5043` `.hub-card__chip { border-radius:7px; border:1px solid var(--hub-border); background:transparent; color:var(--hub-text-muted); font-size:11px; padding:2px 8px; }` — matches Sketch 001 `.v-b .chip` exactly (`index.html:192`). `SessionCard.tsx:524,529` renders both chips with these classes. |
| 3 | Origin chip fully muted — no green-local/blue-remote color coding | VERIFIED | `style.css:5075-5078` `.hub-card__chip--origin { color: var(--hub-text-muted); border-color: var(--hub-border); }` — no per-origin color modifier exists anywhere in style.css or SessionCard.tsx (grep for `hub-card__origin--local`/`--remote` returns zero hits). |
| 4 | INTERNET and FULL ACCESS are the only FILLED chips, on their own dedicated right-aligned exposure line below agent·origin | VERIFIED | `style.css:5085-5092` `.hub-card__exposure { flex-basis:100%; justify-content:flex-end; }` matches sketch `.v-b .exposure` (`index.html:194`). `.hub-internet-badge`/`.hub-fullaccess-badge` (style.css:7266-7273, 7581-7590) keep filled `background` + clip-path — the only filled treatments; `.hub-card__chip` is `background:transparent`. |
| 5 | Both badges coexist when funnelActive AND funnelWriteActive are both true — no supersede | VERIFIED | `SessionCard.tsx:543-570` — outer gate `(session.funnelActive \|\| session.funnelWriteActive)`, each badge gated independently on its own flag (lines 548, 563) — no `else`/mutual exclusion. Test `SessionCard.test.tsx:898-911` "D-05: ...renders BOTH exposure badges" passes (ran live, see Behavioral Spot-Checks). |
| 6 | FULL ACCESS keeps its notched clip-path, 700 weight, LockOpenIcon | VERIFIED | `style.css:7588` `clip-path: polygon(6px 0, 100% 0, 100% 100%, 0 100%, 0 6px)`; `style.css:7598` `font-weight: 700`; `SessionCard.tsx:565` `<LockOpenIcon className="hub-fullaccess-badge__icon">`. All unchanged from Phase 171 (badge CSS block untouched by this phase, only its container moved). |
| 7 | Uptime, viewer count, remote Connected/Available render together on one muted meta line at 11px --hub-text-muted below the chip row | VERIFIED | `style.css:5096-5105` `.hub-card__meta { font-size:11px; color:var(--hub-text-muted); }`; `SessionCard.tsx:317-343` builds `metaItems` array (timeText, viewers, conn) rendered together at `SessionCard.tsx:574-581`, below `.hub-card__chiprow`. Test `SessionCard.test.tsx:913-932` (viewer count + Connected item) passes. |
| 8 | Local and remote branches still work: isLocal renders 'Local'+ComputerDesktopIcon; remote renders peer hostname+GlobeAltIcon; isRemote drives Connected/Available meta item | VERIFIED | `SessionCard.tsx:529-541` (origin chip branch), `SessionCard.tsx:331-343` (conn meta item gated on `isRemote` prop). Tests: "local session renders...'Local'" and "remote session renders...peer hostname" (`SessionCard.test.tsx:868-886`) pass; `SessionCard.share.test.tsx` CARD-02 origin-indicator assertions updated + passing. |
| 9 | Card renders correctly in BOTH dark (:root) and light ([data-ui-theme="light"]) themes — new rules resolve through --hub-* tokens | VERIFIED | All new rules (`.hub-card__chip`, `--origin`, `--exposure`, `--meta`) reference only `--hub-border`, `--hub-text-muted`, `--hub-font-mono` — confirmed present with distinct values in both `:root` (style.css ~4649-4746) and `[data-ui-theme="light"]` (~4730-4780) blocks. Exposure-badge fill tokens (`--hub-internet-badge-bg/text`, `--hub-fullaccess-badge-bg/text`) also have both dark (4707-4715) and light (4773-4780) values, unchanged by this phase. The one intentional per-theme exception (agent-chip tint hexes, no light override) is a carried-forward pattern flagged as WR-02 in 172-REVIEW.md — advisory, not a must-have violation (must-have only requires the *new* rules to resolve through tokens, which they do). |

**Score:** 9/9 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/style.css` | New chip/exposure/meta CSS, both-theme token resolution | VERIFIED | All 6+ new classes present, substantive, correct values; dead `.hub-card__row2`/`row2-meta`/`row2b`/`origin*` rules removed (grep confirms zero remaining references in CSS or TSX) |
| `frontend/src/components/Hub/SessionCard.tsx` | Restructured render: status → chiprow → exposure → meta | VERIFIED | Exact structure present at lines 488-581; `pnpm build` (tsc --noEmit + vite build) passes clean |
| `frontend/src/components/Hub/SessionCard.test.tsx` | Extended structure coverage incl. both-badges-coexist | VERIFIED | New `describe('SessionCard chip row (Phase 172)')` block present (7 tests), all pass live (94/94 across both affected test files) |
| `TESTING.md` | Reconciled Suite Manifest note, no count drift | VERIFIED | Dated Phase-172 note present (`TESTING.md:34`) confirming in-place extension, counts unchanged (376 Go / 142 vitest / 529 total) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| Agent chip tint | `data-agent` attribute on `.hub-card` | CSS attribute selector `.hub-card[data-agent="..."] .hub-card__chip--agent` | WIRED | `SessionCard.tsx:355` sets `data-agent={agentBadgeModifier(cli) ?? 'unknown'}` on the article; `style.css:5061-5068` defines matching per-agent tint rules — same hex palette as the left spine/tab dot (coupling preserved) |
| Exposure badges | `session.funnelActive` / `session.funnelWriteActive` | Conditional JSX render | WIRED | `SessionCard.tsx:548,563` — gating expressions byte-identical to pre-phase logic; only DOM container (`.hub-card__exposure`) changed |
| `pnpm build` | tsc --noEmit + vite build | Build pipeline | WIRED | Ran live: build completed clean, no type errors, dist assets emitted |

### Data-Flow Trace (Level 4)

Not applicable — this phase moves/restyles existing derived values (`cli`, `hostname`, `timeText`, `viewerCount`, `isConnected`) that were already flowing correctly pre-phase; no new data source was introduced. Verified the derivation logic (`isLocal`, `originText`, `timeText`, `metaItems`) is untouched (SUMMARY explicitly notes derived-value logic was out of scope for Task 2, confirmed by reading the diff region — only JSX container/class names changed).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `pnpm build` succeeds (tsc + vite) | `cd frontend && pnpm build` | `✓ built in 539ms`, no type errors | PASS |
| `bash tests/check-traceability-paths.sh` exits 0 | `bash tests/check-traceability-paths.sh` | `OK: all traceability paths exist` | PASS |
| SessionCard.test.tsx + SessionCard.share.test.tsx pass, incl. new Phase-172 chip-row block | `pnpm vitest run src/components/Hub/SessionCard.test.tsx src/components/__tests__/SessionCard.share.test.tsx` | `2 passed (2)`, `94 passed (94)` | PASS |
| D-05 both-badges-coexist test specifically | included in above run — `D-05: funnelActive AND funnelWriteActive both true renders BOTH exposure badges` | passed (part of the 94) | PASS |

### Requirements Coverage

No formal REQUIREMENTS.md REQ-IDs apply to this phase (ROADMAP states "Requirements: TBD";
design-polish phase). Plan-local decision IDs D-01 through D-07 (from 172-CONTEXT.md) map to
the 9 observable truths above — all VERIFIED. No orphaned requirements.

### Anti-Patterns Found

None in the phase's modified files. Grep for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` across
`style.css`, `SessionCard.tsx`, `SessionCard.test.tsx` returns only unrelated pre-existing
`::placeholder` CSS pseudo-selectors and `--hub-text-placeholder` token names outside the
phase's new rules — no debt markers in the new/changed code.

**Advisory findings (from 172-REVIEW.md, not must_have violations, noted per instructions):**
- WR-01: `.hub-card__meta` renders unconditionally and could reserve an empty ~24px strip on
  a remote card if a caller omits the `isRemote` prop (production `HubPanel` always supplies
  it, so this is latent, not currently observable in the app). Polish-tier, not blocking.
- WR-02: Agent-chip tint hexes have no light-theme override, carried forward from the
  pre-existing `.hub-card__badge` palette (same issue existed before this phase). The origin
  chip was actually *improved* to token-based muted color in this phase; the agent chip is
  the lone carried-forward holdout. Polish-tier, not blocking — does not violate the D-02
  must-have, which only requires the phase's *new* rules to resolve through tokens (they do;
  the one intentional exception is explicitly documented in-file).
- IN-01/IN-02/IN-03: dead `.hub-card__row3` selector, redundant `margin-left:0`, index-based
  React keys — cosmetic, non-blocking.

### Human Verification Required

None. This phase's load-bearing invariant (exposure-badge prominence/colorblind-safety) is
fully verifiable at source per project convention (user is colorblind — verify hex constants
in code, not by eye). Sketch-parity (7px radius, 8px gap, flex-basis:100% exposure line) was
confirmed by direct comparison of the ported CSS values against `.v-b` in
`.planning/sketches/001-hub-card-chip-row/index.html`, not by rendering. Dark/light theme
token resolution (D-02) was verified by confirming all referenced `--hub-*` custom properties
have concrete values in both `:root` and `[data-ui-theme="light"]` blocks — a deterministic,
source-level check, not requiring a browser render.

### Gaps Summary

None. All 9 must-have truths verified against the actual codebase (not SUMMARY claims):
CSS classes exist with the exact Sketch 001 Variant B values, JSX renders them in the correct
order with unchanged gating logic, `pnpm build` and the targeted vitest run both pass live,
and `check-traceability-paths.sh` exits 0. The two advisory warnings from the code review
(WR-01 empty meta strip on an unsupported caller path, WR-02 carried-forward agent-chip
contrast) are real but do not violate any must_have and were explicitly scoped as
non-blocking by the phase brief.

---

_Verified: 2026-07-08T02:21:25Z_
_Verifier: Claude (gsd-verifier)_
