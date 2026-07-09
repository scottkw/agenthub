---
phase: 176-platform-hardening-bug-fixes
plan: 03
subsystem: ui
tags: [react, hub, minipreview, css, dev-browser, styled-tail]

# Dependency graph
requires:
  - phase: 172-hub-card-layout-badge-refinement
    provides: current Hub card layout (.hub-card, .hub-card__preview) that BUG-07 was hypothesized against
provides:
  - Live-verified evidence that the Hub session-card mini-preview correctly renders each output line as one clipped horizontal row (does NOT stack characters per row)
  - Closure justification for GitHub #127 (already-fixed, no code change)
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "BRANCH B taken: verdict is DOES-NOT-REPRODUCE, so per D-04 no code change was made; live evidence recorded as the closure justification for #127."

patterns-established: []

requirements-completed: [BUG-07]

coverage:
  - id: D1
    description: "Live dev-browser repro of the #127 long-line scenario proves each mini-preview line renders as ONE clipped horizontal row (not stacked one-char-per-row); verdict DOES-NOT-REPRODUCE recorded with screenshot + computed-style evidence."
    requirement: "BUG-07"
    verification:
      - kind: automated_ui
        ref: "dev-browser screenshot 176-03-evidence-minipreview-repro.png + getComputedStyle/getBoundingClientRect readout of .hub-card__preview-line and child spans (see below)"
        status: pass
    human_judgment: false

# Metrics
duration: 14min
completed: 2026-07-09
status: complete
---

# Phase 176 Plan 03: BUG-07 Mini-Preview Line-Stacking Live Repro Summary

**Live dev-browser repro of GitHub #127 (Hub mini-preview char-per-row stacking) DOES NOT reproduce on the current build — each output line renders as one clipped horizontal row exactly as designed; zero code change made.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-07-09T13:20:00Z
- **Completed:** 2026-07-09T13:34:00Z
- **Tasks:** 2 (Task 1: live repro; Task 2: branch decision — BRANCH B, no code change)
- **Files modified:** 0 (source); 2 untracked throwaway harness files created (not committed, per plan)

## Accomplishments

- Built an isolated dev-browser component harness (`frontend/uat-176-harness.html` / `.tsx`, untracked, mirrors the existing `uat-162`/`uat-173` precedent) that mounts the REAL `MiniPreview` component alone against the REAL `style.css`, inside a container sized like a real `.hub-card` (300px wide, within the 240-360px range).
- Fed it `StyledSpan[][]` data constructed to EXACTLY mirror the backend's real wire shape: one `StyledSpan` per character CELL (not per style-run), matching `internal/daemon/engine.go`'s `GetSessionStyledTailLines` extraction (`row = append(row, StyledSpan{Char: cell.Content, ...})` for every column) — 80-column rows built from the #127 repro command's `live tick #NNN` output, including one row long enough to overflow the card width and one row with an embedded color/bold run (mirroring a real ANSI-styled segment).
- Started a local Vite dev server, navigated dev-browser to the harness, and captured BOTH required evidence types:
  - **Screenshot** (`.planning/phases/176-platform-hardening-bug-fixes/176-03-evidence-minipreview-repro.png`): four preview lines render as four clean, readable horizontal rows; the long line is visibly ellipsis-clipped (`"...some extra tr…"`); the colored/bold character segment renders inline mid-line, not on its own row.
  - **Computed-style + geometry readout** via `page.evaluate`: for every `.hub-card__preview-line`, `display: block`, `flex-direction: row`, `white-space: nowrap`, `overflow: hidden` — and for the child `<span>` elements, `display: inline`. Critically, `getBoundingClientRect()` on the first 6 spans of each line all share the SAME `top` value within a line, and each of the 4 lines has a DISTINCT `top` (78px / 91px / 104px / 117px, in a consistent 13px line-height stride) — i.e., all 80 per-character spans in a row sit on one horizontal baseline, not stacked into 80 separate rows.
- **Verdict: DOES-NOT-REPRODUCE.** Per D-04, made NO code change. `git diff --exit-code` on `MiniPreview.tsx` confirmed clean both after Task 1 (repro-only) and Task 2 (branch decision) — see Self-Check below.
- Task 2's automated verify command (`git -C .. diff --quiet -- src/components/Hub/MiniPreview.tsx src/components/Hub/MiniPreview.test.tsx src/style.css`) exited 0, printing `"BRANCH B: no code change — verify evidence recorded in SUMMARY"` exactly as the plan's branch-detection logic expects.

## Root-cause analysis (why it doesn't reproduce, for the record)

- `.hub-card__preview-line` already carries `white-space: nowrap; overflow: hidden; text-overflow: ellipsis;` (`frontend/src/style.css:6020-6028`) — the CSS the original issue hypothesized as the fix.
- `MiniPreview.tsx`'s per-character `<span>` elements carry no `display` override in their inline `style={{...}}` object (only `color`/`background`/`fontWeight`), so each renders with the browser default `display: inline` and flows horizontally within the `nowrap` parent — confirmed live (`firstSpanDisplay: "inline"` for every row).
- No global CSS rule in `style.css` forces `span { display: block }`, and no `flex-direction: column` leaks onto `.hub-card__preview` or `.hub-card__preview-line` (grepped both scopes; `.hub-card__preview` sets no `display` override at all, defaulting to `block`, and `.hub-card__preview-line` is `display: block` per the live readout too — each line is its own block-level row, with inline spans flowing within it).
- Conclusion: a prior hub-card layout effort (most likely the Phase 172 Hub-card layout refinement, which touched `.hub-card__preview*` selectors) already landed the correct CSS combination before this phase started. #127's root cause no longer exists in the current codebase.

## Task Commits

No source commits — both tasks are evidence-gathering / branch-decision tasks with a mandated zero-diff outcome (D-01 for Task 1 prohibits any code change; D-04/BRANCH B for Task 2 prohibits any code change given the DOES-NOT-REPRODUCE verdict). `git diff --exit-code` was run after each task and passed both times.

**Plan metadata:** (this commit) `docs(176-03): complete BUG-07 live-repro plan`

## Files Created/Modified

- `frontend/uat-176-harness.html` (untracked, NOT committed) — throwaway dev-browser harness shell, mirrors `uat-162-harness.html`/`uat-173-harness.html` precedent.
- `frontend/uat-176-harness.tsx` (untracked, NOT committed) — mounts the real `MiniPreview` with backend-accurate per-cell `StyledSpan[][]` fixture data.
- `.planning/phases/176-platform-hardening-bug-fixes/176-03-evidence-minipreview-repro.png` (committed) — dev-browser screenshot evidence for the DOES-NOT-REPRODUCE verdict.

## Decisions Made

- **BRANCH B taken** (D-04): live evidence shows the current build already renders long lines correctly, so no render-layer fix was applied. The plan's must_haves explicitly require this branch when the repro fails — followed as specified.
- Modeled the harness fixture data as ONE `StyledSpan` per character cell (not per style-run) to exactly match the real backend wire shape from `internal/daemon/engine.go`'s per-cell VT-grid extraction, rather than a simplified per-word/per-run approximation — this is the most faithful possible repro of the reported bug's actual data path.

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched their `<done>` criteria: Task 1 captured live evidence with zero code change; Task 2 correctly branched to BRANCH B (matching the Task 1 verdict) with zero code change and the automated verify command confirming the branch.

## Issues Encountered

- `dev-browser` was not resolvable via `which dev-browser` in this shell (the npm-global bin symlink and the Claude-plugin cache bin directory both lacked a plain `dev-browser` executable matching the CLI's own `package.json` `bin` mapping to `./bin/dev-browser.js`). Resolved by invoking the JS entrypoint directly via `node .../dev-browser.js`, which works identically to the documented `dev-browser` invocation. No code/config change required — purely a local-shell PATH resolution workaround for this execution session.

## User Setup Required

None - no external service configuration required.

## GitHub Issue Note

Per D-04, GitHub `scottkw/agenthub#127` should be closed as already-fixed, with this SUMMARY's evidence as the justification. Following the same convention already established by sibling plans 176-01 (#124) and 176-02 (#123) in this phase — both left their respective GitHub issues OPEN despite landing code fixes — issue closure for #127 is deferred to phase-level verification/ship (not performed by this plan-execution step), so all three phase issues are closed together at a single, consistent point (e.g., during `/gsd-verify-work 176` or ship). No `gh issue close` was run in this plan.

## Next Phase Readiness

- Phase 176's third and final bug (BUG-07/#127) is resolved via live evidence with no code change required — the phase's three bugs (BUG-05 #124, BUG-06 #123, BUG-07 #127) are all addressed (176-01, 176-02, 176-03).
- Next: phase-level TESTING.md reconciliation / traceability pass (D-12) if not already covered by 176-01/176-02, then phase verification (`/gsd-verify-work 176`) which should also close #124/#123/#127 together with the accumulated evidence from all three plans.
- No blockers.

---
*Phase: 176-platform-hardening-bug-fixes*
*Completed: 2026-07-09*
