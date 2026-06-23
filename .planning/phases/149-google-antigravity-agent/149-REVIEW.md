---
phase: 149-google-antigravity-agent
reviewed: 2026-06-22T23:25:00Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - frontend/src/components/__tests__/style.hub.test.ts
  - frontend/src/lib/agentBadge.test.ts
  - frontend/src/lib/agentBadge.ts
  - frontend/src/style.css
  - internal/daemon/path_windows_test.go
  - internal/daemon/path_windows.go
  - internal/pty/detect_test.go
  - internal/pty/detect.go
  - internal/release/no_autosave_test.go
  - internal/status/detector_test.go
  - internal/status/detector.go
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: resolved
resolution:
  WR-01: "Fixed in 8bfc40d1 — agy idle pattern anchored to (?m)^>$ + negative regression test"
  WR-02: "Fixed in e6223e5c — WCAG comment corrected to actual chip surface (dark 8.16:1 / light 1.73:1) + source-gate tests updated"
  IN-01: "Folded into WR-01 — no positive working marker remains a documented post-M-15 tuning item"
  IN-02: "Addressed by WR-01 negative test (TestDetector_AgyIdleNotBroadAngleBracket)"
---

# Phase 149: Code Review Report

> **Resolution (2026-06-23):** Both warnings fixed and committed. WR-01 → `8bfc40d1`,
> WR-02 → `e6223e5c`. Full suite re-verified green (go test ./..., vitest 1878/1878, tsc clean).

**Reviewed:** 2026-06-22T23:25:00Z
**Depth:** standard
**Files Reviewed:** 11
**Status:** issues_found

## Summary

Phase 149 adds the Google Antigravity (`agy`) CLI as a first-class agent across
the stack: PATH discovery (`detect.go`, `path_windows.go`), status detection
(`detector.go`), and frontend badge/spine/chip color identity (`agentBadge.ts`,
`style.css`). The change is small, additive, and well-tested — all 11 files
build and their tests pass (`go test ./internal/status ./internal/pty` OK;
`vitest` 96/96 green).

No security defects and no crash/data-loss risks were found. The change set
correctly avoids touching unrelated code paths and keeps the three CSS color
sites (tab dot, card spine, card chip) in lockstep, satisfying the
single-source-of-truth invariant the existing tests guard.

Two WARNING-level correctness/honesty issues stand out, both in the
heuristic/documentation layer rather than the wiring:

1. The `agy` idle pattern `>\s*$` is far too broad and will misclassify
   ordinary output as "idle."
2. The WCAG contrast numbers documented in the new CSS comment do not match
   the surface the text actually renders on — a concern that matters because
   the project relies on source-level (not by-eye) color verification.

The `[ASSUMED]`/post-M-15 tuning caveat in `DefaultAgyPatterns` is honestly
documented, which is why the pattern issue is a WARNING and not a BLOCKER.

## Warnings

### WR-01: `agy` idle pattern `>\s*$` is over-broad and will false-positive to Idle

**File:** `internal/status/detector.go:104` (`DefaultAgyPatterns`)
**Issue:**
The `agy` Idle pattern is `regexp.MustCompile(`>\s*$`)`. Unlike Claude's idle
marker `❯` (line 69), which is a distinctive non-ASCII prompt glyph almost
never seen in normal output, a bare `>` followed by optional trailing
whitespace is extremely common in real CLI output: shell-style prompts,
redirection examples (`foo > bar`), markdown blockquotes, HTML/XML/JSX
fragments (`...</div>`), diff/arrow tokens (`=>`, `->`), and quoted email/log
lines all end a line with `>`. Because `classify()` checks Idle against the
last 256 bytes (`tailSuffix`) and `agy`'s `Working` set is empty, any chunk
whose suffix happens to end in `>` will flip a genuinely-working session to
`StatusIdle`. On the Hub this clears the "running" affordance prematurely and
can suppress the "Needs input"/attention signal.

This is the same class of detector-tuning bug the project has hit before
(MEMORY: "Hub status comes from a daemon heuristic detector" / "stuck Running"
tuning, and the #95 collapsed-footer fix). The pattern is explicitly marked
`[ASSUMED]` and slated for post-M-15 tuning, so this is a WARNING, but it
should not ship as the long-term marker: a bare `>` is the weakest possible
idle signal.

**Fix:** Anchor the idle marker to the real agy prompt once captured live; in
the interim, tighten it so stray `>` in mid-output does not match — e.g.
require the prompt to be at the start of the final line and be the whole line:

```go
Idle: []*regexp.Regexp{
    // Match a prompt-only final line ("> " with nothing after it), not any
    // line that merely ends in '>'. Tune to the real agy glyph post-M-15.
    regexp.MustCompile(`(?m)^>\s*$`),
},
```

Add a negative test (e.g. `Feed([]byte("rendered <div>\n"))` must NOT be Idle)
to lock the narrowed contract.

### WR-02: Documented WCAG contrast ratios for `agy` do not match the rendering surface

**File:** `frontend/src/style.css:1718` (and the chip rule at line 5030)
**Issue:**
The new comment claims `dark: 8.72:1 AA PASS; light: 2.03:1 FAIL`. The project
verifies color at the source level (the user is colorblind), so these numbers
are a load-bearing honesty gate (D-07) — but neither figure matches the surface
the `#ff9e64` text actually renders against. The colored chip text
(`.hub-card[data-agent="agy"] .hub-card__badge`) sits on
`--hub-surface-elevated` (`#1c1e28` dark / `#ececf0` light, style.css:4540/4610),
which gives **8.16:1 dark / 1.73:1 light** — not 8.72 / 2.03. The documented
`2.03:1` only reproduces against pure `#ffffff`, which is not the light theme
background (`--hub-bg: #f5f5f7`, or the elevated chip surface `#ececf0`), and I
could find no surface in the file that yields `8.72:1` (computed values:
8.96 vs `#14151b`, 8.16 vs `#1c1e28`). The comment is also attached to the tab
**dot** rule, where `#ff9e64` is a *background* fill, so a text-contrast ratio
does not even describe that selector.

The visual outcome is fine (shape + text carry identity; color is
reinforcement), so this is not a BLOCKER — but the precise ratios are the thing
the convention exists to make trustworthy, and they are wrong for the context.

**Fix:** Recompute against the actual chip surface and state which surface the
ratio is measured on, or drop the false-precision numbers:

```css
/* agy — orange #ff9e64. Chip text on --hub-surface-elevated:
   dark #1c1e28 = 8.16:1 (AA PASS); light #ececf0 = 1.73:1 (FAIL,
   same gap as all existing agents — text + left-spine shape carry identity). */
.tab__agent-badge--agy      { background: #ff9e64; }
```

## Info

### IN-01: `agy` has no positive "working" signal, so active sessions rely on the default

**File:** `internal/status/detector.go:99-101, 108`
**Issue:** `DefaultAgyPatterns` leaves `Working` empty by design (documented),
so a busy `agy` session only reads as running via the conservative default in
`classify()`. Combined with WR-01's broad idle pattern, a working session that
emits any `>`-terminated line will transition Idle → and stay there until the
next non-matching chunk. The honest `[ASSUMED]` comment covers the intent, but
the empty `Working` set + weak `Idle` marker compound each other; tuning WR-01
mitigates most of the risk. Track alongside the post-M-15 verification
(RESEARCH Open Question 1 / CONTEXT D-13).
**Fix:** When live access is available, add at least one `Working` marker (the
agy equivalent of "ctrl+c to interrupt"/spinner) so active work is positively
classified rather than inferred only by the absence of other matches.

### IN-02: `agy` Idle test asserts the over-broad pattern it should constrain

**File:** `internal/status/detector_test.go:388-395` (`TestDetector_AgyIdle`)
**Issue:** The test feeds `"Output done\n> "` and asserts `StatusIdle`, which
passes precisely because `>\s*$` is broad. There is no negative case proving a
mid-output `>` (e.g. rendered markup or a `foo > bar` example) is *not*
misread as idle. As written the suite cannot catch the WR-01 regression class.
**Fix:** Add a negative assertion, e.g.:

```go
func TestDetector_AgyNotIdleOnTrailingAngle(t *testing.T) {
    var got status.SessionStatus
    d := newAgyDetector("s1", func(_ string, s status.SessionStatus) { got = s })
    d.Feed([]byte("writing <Component>\n"))
    if got == status.StatusIdle {
        t.Errorf("trailing '>' in mid-output must not classify as Idle")
    }
}
```
Update TESTING.md (Suite Manifest + traceability) per the standing rule if a
new test file/case is added.

---

_Reviewed: 2026-06-22T23:25:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
