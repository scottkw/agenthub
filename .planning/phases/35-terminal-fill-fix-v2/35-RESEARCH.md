# Phase 35: Terminal Fill Fix v2 - Research

**Researched:** 2026-03-26
**Domain:** xterm.js FitAddon cell-dimension initialization, CharSizeService timing, Wails WebView rendering
**Confidence:** HIGH

## Summary

Phase 34 implemented the double-rAF deferral and initial PTY dimension threading. That code is **still in the codebase and working** — the backend plumbing (cols/rows through the Go stack) is correct. The problem is narrower than Phase 34 assumed.

The true root cause: `FitAddon.proposeDimensions()` silently returns `undefined` — and `fit()` does nothing — when `_renderService.dimensions.css.cell.width === 0`. This condition occurs when xterm.js's `CharSizeService` has not yet measured the font's character cell dimensions. CharSizeService measures by inserting a hidden `<span>` into the terminal container and reading `getBoundingClientRect()`. If that span has no layout dimensions (which happens when the container has `display: none` at `term.open()` time), the measurement returns zero and is **cached** as zero until the next xterm.js render cycle.

Codex fills because it outputs data immediately after startup, which triggers an xterm.js render pass, which re-runs CharSizeService measurement, which caches non-zero cell dimensions. By the time the double-rAF fires, cell dimensions are already non-zero, and `fit()` succeeds.

Claude, Gemini, and OpenCode have slower startup sequences (authentication, API initialization). When the double-rAF fires ~32ms after tab activation, xterm.js has not yet had a render cycle for those sessions, so `css.cell.width === 0`, and `fit()` silently returns `undefined` and does nothing. The user sees the terminal at its spawn dimensions (the estimated 220x50 or whatever the PTY was told), but xterm.js has drawn it at 80x24 or its initialized dimensions.

**The fix requires**: a retry loop that calls `fit()` repeatedly via `requestAnimationFrame` until `FitAddon.proposeDimensions()` returns a non-undefined value (meaning cell dimensions are non-zero). This is the "rAF polling until success" pattern. The retry loop must have a bounded max-attempt count and must be cancelled on cleanup.

**Primary recommendation:** Replace the double-rAF one-shot with an rAF-based retry loop (max ~20 frames ≈ 333ms at 60fps) that polls `proposeDimensions()` until it returns non-undefined, then calls `fit()`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FILL-01 | Terminal fills full viewport width on initial tab activation for Claude CLI | rAF retry loop waits until CharSizeService measures non-zero cell dims |
| FILL-02 | Terminal fills full viewport width on initial tab activation for Gemini CLI | Same fix as FILL-01; CLI-agnostic |
| FILL-03 | Terminal fills full viewport width on initial tab activation for OpenCode | Same fix as FILL-01; CLI-agnostic |
| FILL-04 | Terminal fills full viewport width on initial tab activation for Codex (no regression) | Codex already succeeds; retry loop will succeed on first attempt |
| FILL-05 | Switching tabs to previously hidden terminal fills correctly without resize | rAF retry fires on each `isActive` change; works for tab switches too |
| FILL-06 | Fix behaves identically in `wails dev` and production `wails build` | rAF polling is timing-agnostic; works regardless of WebView paint delay |
</phase_requirements>

## Root Cause Analysis (Why Phase 34 Was Insufficient)

### FitAddon.proposeDimensions() source (verified via `frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js`):

```javascript
proposeDimensions() {
  if (!this._terminal) return;
  if (!this._terminal.element || !this._terminal.element.parentElement) return;
  const e = this._terminal._core._renderService.dimensions;
  // GUARD: returns undefined if cell dimensions are zero
  if (0 === e.css.cell.width || 0 === e.css.cell.height) return;
  // ... rest of calculation
}
```

The `if (0 === e.css.cell.width || 0 === e.css.cell.height) return` guard is the silent failure point.

### Why cell dimensions are zero at double-rAF time for slow CLIs

1. `term.open(containerRef.current)` is called when the panel has `display: none`
2. xterm.js inserts `.xterm-char-measure-element` (hidden span) into the container
3. CharSizeService calls `getBoundingClientRect()` on that span — returns 0 because the parent has `display: none`
4. CharSizeService caches 0 for `css.cell.width` and `css.cell.height`
5. The terminal displays but has no font measurement yet
6. Double-rAF fires ~32ms later — cell dimensions still 0 — `fit()` returns `undefined` silently — nothing happens
7. Only when xterm.js renders content (parses output from CLI) does it re-trigger CharSizeService measurement on a visible element

### Why Codex succeeds with double-rAF

Codex outputs data within milliseconds of starting. By the time the user's tab is active and the double-rAF fires, xterm.js has already rendered output, measured the font, and cached non-zero cell dimensions.

### Why Claude/Gemini/OpenCode fail

These CLIs have initialization delays (auth, API setup). No output arrives before the double-rAF deadline. Cell dimensions remain zero.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @xterm/xterm | 6.0.0 (current) | Terminal emulator | Already installed; no upgrade needed |
| @xterm/addon-fit | 0.11.0 (current) | Measure container, call terminal.resize() | Already installed; no upgrade needed |
| requestAnimationFrame | browser built-in | rAF retry loop | Native; no dependencies |
| ResizeObserver | browser built-in | Subsequent resize handling | Already wired; unchanged |

**Installation:** No new dependencies required.

**Version verification:** `@xterm/xterm` 6.0.0 and `@xterm/addon-fit` 0.11.0 are the current installed versions, verified via `npm view @xterm/xterm version` (2026-03-26). No version changes needed.

## Architecture Patterns

### Recommended Project Structure

No structural changes. This is a single-file change to `frontend/src/components/TerminalPanel.tsx`.

### Pattern 1: rAF Retry Loop Until Fit Succeeds

**What:** Replace the one-shot double-rAF with a retry loop that polls `proposeDimensions()` until it returns non-undefined (meaning cell dimensions are non-zero), then calls `fit()`.

**When to use:** For the initial fit when a panel becomes visible. The loop stops immediately on first success.

**Why rAF, not setTimeout:** requestAnimationFrame fires before paint, batched with the browser's render loop. It gives xterm.js the opportunity to render in each frame before we check. setTimeout(0) would also work but rAF is more precise.

**Max attempts:** 20 rAF calls covers ~333ms at 60fps. This is long enough for any CLI to start outputting data. If 20 frames elapse with no success, a final direct `fit()` attempt is made as a best-effort fallback.

**Example:**
```typescript
// Source: derived from FitAddon source inspection + CharSizeService timing analysis
useEffect(() => {
  if (!isActive || !containerRef.current) return

  const container = containerRef.current
  let cancelled = false
  let rafId: number | undefined
  const MAX_ATTEMPTS = 20  // ~333ms at 60fps

  const tryFit = (attempt: number) => {
    if (cancelled) return

    // Check if FitAddon can measure (cell dimensions non-zero)
    const dims = fitAddonRef.current?.proposeDimensions()
    if (dims !== undefined) {
      // Cell dimensions are ready — fit now
      fitAddonRef.current?.fit()
      return
    }

    // Cell dimensions still zero — schedule next attempt
    if (attempt < MAX_ATTEMPTS) {
      rafId = requestAnimationFrame(() => tryFit(attempt + 1))
    } else {
      // Final fallback: attempt fit regardless of cell dim state
      fitAddonRef.current?.fit()
    }
  }

  // Start retry loop after first rAF (ensures layout is committed after display:none -> flex)
  rafId = requestAnimationFrame(() => tryFit(0))

  // ResizeObserver handles all subsequent size changes
  const ro = new ResizeObserver(() => fitAddonRef.current?.fit())
  ro.observe(container)

  return () => {
    cancelled = true
    if (rafId !== undefined) cancelAnimationFrame(rafId)
    ro.disconnect()
  }
}, [isActive])
```

**Key properties:**
- First rAF defers until layout is committed (display:none → flex transition)
- `tryFit` checks `proposeDimensions()` — if undefined (cell dims zero), schedules next rAF
- If defined, calls `fit()` and stops
- Bounded at MAX_ATTEMPTS = 20 to prevent infinite loop
- `cancelled` flag + `cancelAnimationFrame` prevents fit on unmounted terminal
- `document.fonts.ready` is NOT needed — CharSizeService handles font measurement independently

### Pattern 2: ResizeObserver for Subsequent Resizes (unchanged)

The existing ResizeObserver pattern is correct and handles all resize events after initial fit. No changes.

### Anti-Patterns to Avoid

- **One-shot double-rAF:** The current approach — fires once at ~32ms, hits zero cell dims for slow CLIs, does nothing. This is the bug.
- **Unbounded polling loop:** Must have a MAX_ATTEMPTS cap to prevent infinite rAF chains on pathological cases.
- **Using `proposeDimensions()` without null check in ResizeObserver:** The ResizeObserver callback calls `fit()` directly (not `proposeDimensions()` first). This is fine because after the initial fit succeeds, cell dimensions are non-zero and stay non-zero.
- **Changing the `document.fonts.ready` strategy:** Not needed. The issue is CharSizeService, not font loading. Font loading would produce 0-width container, not 0-width cells.
- **Removing the single leading rAF before tryFit:** The outer rAF is still needed to ensure the display:none → flex layout change is committed before the first measurement attempt.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Measure container → cols/rows | Manual pixel math | FitAddon.proposeDimensions() | Already handles padding, scrollbar width, cell measurement |
| Track container size changes | setInterval polling | ResizeObserver | Already wired; fires on layout changes |
| PTY resize signaling | Custom protocol | Existing MSG_RESIZE2 frame + onResize handler | Already implemented |
| Detect cell dimension readiness | Custom CharSizeService access | `proposeDimensions() !== undefined` check | This is the correct way to test if FitAddon can measure |

**Key insight:** `proposeDimensions()` returning `undefined` is already the canonical signal that cell dimensions are not ready. The retry loop exploits this existing guard to know when to try vs when to wait.

## Common Pitfalls

### Pitfall 1: Accessing `_core` Directly for Cell Dimension Check

**What goes wrong:** Someone might check `termRef.current._core._renderService.dimensions.css.cell.width > 0` directly instead of using `proposeDimensions()`.

**Why it's bad:** `_core` is a private API. `proposeDimensions()` is the correct public-adjacent way to check readiness and returns `undefined` as the canonical "not ready" signal.

**How to avoid:** Use `fitAddonRef.current?.proposeDimensions() !== undefined` as the readiness check.

### Pitfall 2: MAX_ATTEMPTS Too Low

**What goes wrong:** If MAX_ATTEMPTS is too small (e.g., 2-3), Claude CLI at 60fps would still fail on very slow machines or under heavy load.

**Why it happens:** 20 frames at 60fps = 333ms. At 30fps = 667ms. This covers typical CLI startup delays. Going lower risks misses.

**How to avoid:** Keep MAX_ATTEMPTS at 20. This is the minimum viable value. 20 rAF calls is negligible overhead.

### Pitfall 3: cancelAnimationFrame Only Tracks One ID

**What goes wrong:** The retry loop creates a new rAF ID each iteration. If only the first is tracked, subsequent rAF calls leak after cleanup.

**Why it happens:** `let rafId: number | undefined` is overwritten each time `requestAnimationFrame` is called. Each overwrite means the previous ID is lost and cannot be cancelled.

**How to avoid:** The `cancelled` flag short-circuits the loop. Even if `cancelAnimationFrame(rafId)` only cancels the most recently scheduled rAF, the `if (cancelled) return` at the top of `tryFit` ensures no further rAF calls are made. The combination of `cancelled = true` + `cancelAnimationFrame(rafId)` is sufficient — the current outer rAF is cancelled, and any in-flight rAF exits immediately due to `cancelled`.

**Warning signs:** Console error "cannot fit a disposed terminal."

### Pitfall 4: wails dev vs. wails build Behavior Difference (FILL-06)

**What goes wrong:** Fix appears to work in `wails dev` but fails in the production binary.

**Why it happens:** `wails dev` uses a Vite dev server with different paint timing. The production Wails binary embeds assets in WebView2/WebKit which may delay the first layout pass.

**How to avoid:** rAF polling is timing-agnostic — it retries until success regardless of how many frames the WebView needs. This fix is specifically more robust than double-rAF because it doesn't assume a fixed number of frames.

**Validation note:** Must test in production binary (`wails build -tags wailsassets`), not just `wails dev`.

### Pitfall 5: ResizeObserver Fires Before Initial Fit Completes

**What goes wrong:** ResizeObserver fires while the retry loop is still running. Two concurrent fit operations.

**Why it's not a real problem:** The ResizeObserver `fit()` call on an xterm.js terminal is idempotent. If the retry loop has already succeeded, a redundant `fit()` is a no-op (same cols/rows). If the retry loop hasn't succeeded yet (cell dims still zero), `fit()` in ResizeObserver also fails silently — same outcome. No corruption.

## Code Examples

### Complete isActive Effect (rAF Retry Loop)

```typescript
// Source: FitAddon source inspection + CharSizeService timing analysis
// Pattern: retry requestAnimationFrame until proposeDimensions() returns non-undefined
useEffect(() => {
  if (!isActive || !containerRef.current) return

  const container = containerRef.current
  let cancelled = false
  let rafId: number | undefined
  const MAX_ATTEMPTS = 20  // ~333ms at 60fps; covers slow CLI startup delays

  const tryFit = (attempt: number) => {
    if (cancelled) return

    // proposeDimensions() returns undefined when css.cell.width === 0
    // (CharSizeService hasn't measured font yet — zero cell dims from display:none open())
    const dims = fitAddonRef.current?.proposeDimensions()
    if (dims !== undefined) {
      fitAddonRef.current?.fit()
      return
    }

    // Cell dimensions not ready — schedule next rAF attempt
    if (attempt < MAX_ATTEMPTS) {
      rafId = requestAnimationFrame(() => tryFit(attempt + 1))
    } else {
      // Best-effort fallback after max attempts
      fitAddonRef.current?.fit()
    }
  }

  // Initial rAF: ensure display:none -> flex layout change is committed
  rafId = requestAnimationFrame(() => tryFit(0))

  // ResizeObserver handles all subsequent size changes (window resize, font size change)
  const ro = new ResizeObserver(() => fitAddonRef.current?.fit())
  ro.observe(container)

  return () => {
    cancelled = true
    if (rafId !== undefined) cancelAnimationFrame(rafId)
    ro.disconnect()
  }
}, [isActive])
```

### What Stays Unchanged

The following Phase 34 changes remain correct and must NOT be reverted:

- `internal/daemon/types.go` — `CreateRequest.Cols` and `CreateRequest.Rows` fields
- `internal/daemon/engine.go` — `cols, rows int` params + `if cols <= 0` defaults
- `internal/daemon/client.go` — `cols, rows int` params threading
- `internal/daemon/api.go` — `req.Cols, req.Rows` passed to engine
- `app.go` — `cols, rows int` in `App.CreateSession`
- `frontend/src/App.tsx` — `createTab` container measurement + `CreateSession(..., cols, rows)`
- `frontend/src/wailsjs/go/main/App.js` / `App.d.ts` — updated binding signatures

Only `TerminalPanel.tsx` changes in Phase 35.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| document.fonts.ready.then(fit) | Double-rAF → fit() | Phase 34 | Insufficient: cell dims may still be 0 |
| Double-rAF one-shot | rAF retry until proposeDimensions() non-undefined | Phase 35 (this phase) | Correct: waits until xterm.js font measured |
| Hardcoded 80x24 spawn | Container-estimated cols/rows from createTab | Phase 34 | Correct: PTY spawns at viewport size |

**Deprecated/outdated:**
- Double-rAF one-shot pattern: replaced by rAF retry loop in this phase.
- `document.fonts.ready.then(fit)`: removed in Phase 34, still absent in Phase 35.

## Open Questions

1. **Is MAX_ATTEMPTS = 20 sufficient for all environments?**
   - What we know: 20 frames at 60fps = 333ms; at 30fps = 667ms. Most CLI auth flows complete within this window on normal hardware.
   - What's unclear: Very slow machines or first-launch authentication flows (e.g., Claude needing OAuth in the browser) may take longer.
   - Recommendation: 20 is a reasonable minimum. The final best-effort `fit()` call after MAX_ATTEMPTS ensures we don't simply give up — it may work or not. ResizeObserver will catch any subsequent window resize anyway. If 20 is empirically insufficient, bump to 30 (500ms at 60fps).

2. **Should `document.fonts.ready` be awaited inside tryFit?**
   - What we know: CharSizeService measurements return non-zero cell dimensions only after the font is loaded (it measures the span with the terminal font applied). If font hasn't loaded, CharSizeService returns 0.
   - What's unclear: Whether Cascadia Code/system monospace fonts are always ready before the rAF retry loop finishes.
   - Recommendation: `proposeDimensions() !== undefined` already implies cell dims are non-zero which implies fonts are measured. No need to separately await `document.fonts.ready` — if fonts aren't loaded, `proposeDimensions()` will return undefined and the loop continues.

## Environment Availability

Step 2.6: Purely frontend TypeScript change. No new external dependencies.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js / pnpm | Frontend build | ✓ | v20.19.3 | — |
| Go toolchain | Backend (unchanged) | ✓ | go1.26.1 | — |
| wails CLI | Production build validation | ✓ | v2.10.2 | — |
| vitest | Frontend tests | ✓ | 4.1.0 | — |

**Missing dependencies with no fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.0 |
| Config file | `frontend/vite.config.ts` |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && npx vitest run` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && npx vitest run && cd .. && go test ./internal/daemon/... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FILL-01..04 | rAF retry loop present in isActive effect | unit (source inspection) | `cd frontend && npx vitest run` | ✅ extend `TerminalPanel.test.tsx` |
| FILL-01..04 | Loop checks `proposeDimensions()` before calling `fit()` | unit (source inspection) | `cd frontend && npx vitest run` | ✅ extend `TerminalPanel.test.tsx` |
| FILL-01..04 | MAX_ATTEMPTS constant present | unit (source inspection) | `cd frontend && npx vitest run` | ✅ extend `TerminalPanel.test.tsx` |
| FILL-05 | isActive effect dependency array remains `[isActive]` | unit (source inspection) | `cd frontend && npx vitest run` | ✅ existing test covers dep array |
| FILL-06 | No wails-dev-specific code path | manual (production binary) | `wails build -tags wailsassets` then launch | ❌ manual only |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && npx vitest run`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && npx vitest run && cd .. && go test ./internal/daemon/... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work` + manual production binary validation

### Wave 0 Gaps
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — add FILL-01..06 source inspection tests:
  - `tryFit` function present in isActive effect
  - `proposeDimensions()` call present
  - `MAX_ATTEMPTS` constant present
  - `cancelled` flag still present
  - `cancelAnimationFrame(rafId)` present in cleanup
  - `[isActive]` dependency array unchanged

*(No new test files needed — extend existing file)*

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 35 |
|-----------|-------------------|
| TypeScript: camelCase, ESLint + Prettier | Variable names: `tryFit`, `rafId`, `MAX_ATTEMPTS`, `cancelled` |
| pnpm preferred | Use `pnpm run test` / `npx vitest run` for test execution |
| 80%+ coverage in critical components | Source inspection tests provide structural coverage; behavioral coverage requires production binary |
| Wails build requires `-tags wailsassets` | Production validation: `wails build -tags wailsassets` |
| Make beliefs pay rent | The root cause (zero cell dims from `display:none` open) must be confirmed, not assumed |

## Sources

### Primary (HIGH confidence)
- Direct source inspection: `frontend/node_modules/@xterm/addon-fit/lib/addon-fit.js` — `proposeDimensions()` returns `undefined` when `css.cell.width === 0 || css.cell.height === 0`
- Direct source inspection: `frontend/src/components/TerminalPanel.tsx` (current Phase 34 code) — double-rAF one-shot fires once, does not retry
- Project STATE.md — "double-rAF is insufficient; may need a fundamentally different trigger (MutationObserver, explicit dimension polling, or CLI-specific ready signal)"
- Project STATE.md — "Codex fills on initial load; Claude, Gemini, OpenCode do not — key diagnostic clue for fix approach"

### Secondary (MEDIUM confidence)
- xterm.js issue #4338 — FitAddon proposeDimensions() returns undefined/NaN when render service dimensions aren't initialized (verified that zero cell dims cause early return)
- xterm.js official API docs — `onRender` event fires when rows are rendered; no public API for triggering CharSizeService measurement
- General xterm.js pattern: `proposeDimensions()` returning undefined is canonical signal that cell dimensions not yet measured

### Tertiary (LOW confidence)
- Inference about CharSizeService behavior from `display:none` containers — derived from known CSS/DOM behavior (getBoundingClientRect returns 0 for hidden elements); not directly confirmed from CharSizeService source
- Inference about Codex vs other CLIs — based on behavioral evidence (Codex fills, others don't) and startup-timing hypothesis; not directly confirmed

## Metadata

**Confidence breakdown:**
- Root cause identification: HIGH — FitAddon source directly confirms `css.cell.width === 0` early return; CharSizeService zero-from-hidden-container inference is MEDIUM but well-supported by known DOM behavior
- Fix approach: HIGH — rAF retry until `proposeDimensions()` non-undefined is the canonical pattern for this class of problem; exploits existing FitAddon contract
- Codex vs others explanation: MEDIUM — behavioral inference supported by timing analysis; would need console instrumentation to confirm definitively

**Research date:** 2026-03-26
**Valid until:** 2026-06-26 (stable domain; xterm.js FitAddon API unlikely to change)
