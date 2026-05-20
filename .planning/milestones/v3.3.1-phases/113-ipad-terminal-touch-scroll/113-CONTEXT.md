# Phase 113: iPad terminal touch-scroll - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning

<domain>
## Phase Boundary

On iPad Safari and iPad Chrome, single-finger drag on the terminal area scrolls xterm scrollback — matching desktop wheel-scroll behavior. Two-finger drag does not pan the viewport when started on the terminal area. Closes GitHub Issue #56.

</domain>

<decisions>
## Implementation Decisions

### Root cause (per v3.3.1-ROADMAP.md Phase 113)
- xterm-helper-textarea captures touch events and prevents scrollback navigation.
- Pure frontend bug. Web only — Wails desktop has no touchscreen path.

### Fix approach (LOCKED — researcher to pick best concrete shape)
- **Option A: `touch-action: pan-y` CSS opt-in** on the terminal container — delegate scroll to the browser's native touch-scroll. Simplest if it works with xterm.js.
- **Option B: explicit `touchstart`/`touchmove` handlers** on the terminal container that drive xterm.js scrollback via `scrollLines()` API. More control; more code.
- Researcher should test Option A first (lighter touch). If incompatible with xterm-helper-textarea capture, Option B.

### Must-not-regress
- **Desktop mouse-wheel scroll** must continue to work.
- **Existing iPad tap-on-link cluster (Issue #46 / UAT-04 carry-over)** — taps on OSC 8 hyperlinks must still fire click handler. This is a different surface (tap vs. drag) but the same touch-event plumbing is involved.

### Cross-surface verification (release gate)
- **iPad Safari + iPad Chrome:** physical iPad required (same hardware used for v3.3 UAT-04).
- **Desktop Chrome:** regression smoke for mouse-wheel scroll.
- **macOS executor CANNOT do iPad UAT** — items must be human_needed.

### Out of scope
- Mobile native app.
- Other iPad touch interactions beyond drag-scroll.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/TerminalPanel.tsx` — terminal container component.
- xterm.js `scrollLines(N)` API — already exists for programmatic scroll.
- xterm-helper-textarea — the offending input capture element.

### Established Patterns
- Web-link / OSC 8 tap handling (Issue #46 / UAT-04 carry-over). Researcher to find the existing tap handler so the touch-scroll path doesn't break it.
- CSS `touch-action` already used elsewhere in the frontend (researcher to confirm).

</code_context>

<specifics>
## Specific Ideas

- Issue #56 reproduction: web-share a shell session, open it on an iPad, generate enough output to scroll, try single-finger drag → currently doesn't scroll.
- Verify two-finger drag DOES pan viewport OUTSIDE the terminal but is consumed BY xterm INSIDE the terminal.

</specifics>

<deferred>
## Deferred Ideas

None.

</deferred>
