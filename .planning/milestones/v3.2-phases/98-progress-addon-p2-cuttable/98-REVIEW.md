---
phase: 98-progress-addon-p2-cuttable
reviewed: 2026-05-08T00:00:00Z
depth: standard
files_reviewed: 33
files_reviewed_list:
  - app_set_tray_progress_test.go
  - app.go
  - assets/tray_icon_progress_100.png
  - assets/tray_icon_progress_25.png
  - assets/tray_icon_progress_50.png
  - assets/tray_icon_progress_75.png
  - build/gen_progress_icons.go
  - frontend/e2e/progress.spec.ts
  - frontend/package.json
  - frontend/pnpm-lock.yaml
  - frontend/src/App.tsx
  - frontend/src/components/__tests__/App.test.tsx
  - frontend/src/components/__tests__/PluginsSection.test.tsx
  - frontend/src/components/__tests__/TabBar.test.tsx
  - frontend/src/components/__tests__/TerminalPanel.test.tsx
  - frontend/src/components/PluginsSection.tsx
  - frontend/src/components/TabBar.tsx
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/lib/__tests__/aggregateProgress.test.ts
  - frontend/src/lib/aggregateProgress.ts
  - frontend/src/style.css
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - internal/release/no_progress_when_off_test.go
  - internal/webserver/vendor_drift_test.go
  - tests/fixtures/osc94-progress-fixture.sh
  - tray_linux.go
  - tray_windows.go
  - tray.go
  - web/assets/terminal.css
  - web/assets/terminal.js
  - web/embed.go
  - web/vendor/xterm/addons/addon-progress.js
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 98: Code Review Report

**Reviewed:** 2026-05-08
**Depth:** standard
**Files Reviewed:** 33
**Status:** issues_found

## Summary

Phase 98 adds OSC 9;4 progress support: a ProgressAddon hot-swap arm in `TerminalPanel.tsx`, cross-session mean aggregation in `aggregateProgress.ts`, a debounced `SetTrayProgress` Wails RPC, and per-quartile tray icon glyphs on macOS/Linux/Windows. The overall architecture is sound and the test scaffold is well-structured. One confirmed data race on `App.lastTrayQuartile` affects all three platforms; three quality issues follow.

## Critical Issues

### CR-01: Data race on `App.lastTrayQuartile` — read/write from concurrent goroutines without synchronization

**File:** `app.go:59,899-902` / `tray.go:115` / `tray_linux.go:429` / `tray_windows.go:616`

**Issue:** `SetTrayProgress` is called by the Wails RPC dispatcher goroutine (one goroutine per inbound call). It reads and writes `a.lastTrayQuartile` at lines 899 and 902 without holding any lock. Concurrently, `startTrayPoller` calls `refreshTrayState()` every 5 seconds from a separate goroutine, which calls `updateTray` → `trayIconBytesForState`, which reads `a.lastTrayQuartile` (all three platform files). These two goroutines operate on the same field with no mutex or atomic, which is an unsynchronized concurrent read/write — a data race per the Go memory model.

The race window is narrow in practice (the field is a single `int` and the scheduler rarely interleaves at exactly the right instruction), but `go test -race` will detect it and it is technically undefined behavior.

**Fix:** Add a mutex to `App` protecting `lastTrayQuartile`, or use `sync/atomic`:

```go
// In App struct (app.go):
lastTrayQuartiले int32 // use atomic ops; -1 = unset

// In SetTrayProgress:
cur := atomic.LoadInt32(&a.lastTrayQuartile)
if cur == int32(quartile) {
    return nil
}
atomic.StoreInt32(&a.lastTrayQuartile, int32(quartile))

// In trayIconBytesForState (each platform file):
q := int(atomic.LoadInt32(&a.lastTrayQuartile))
switch q {
...
```

Alternatively, guard both sites with a dedicated `sync.Mutex` (e.g. `trayMu sync.Mutex`) acquired before the idempotency check in `SetTrayProgress` and before the `switch` in `trayIconBytesForState`.

## Warnings

### WR-01: Linux `updateTray` decodes progress PNG on every 5-second poll cycle, inside `tray.mu.Lock()`

**File:** `tray_linux.go:451-454`

**Issue:** On Linux, `updateTray` calls `makePixmap(bytes)` — which decodes a PNG and converts it to ARGB32 pixel data — while holding `tray.mu.Lock()`. The four progress-quartile pixmaps are NOT pre-decoded at `initTray` time (unlike Windows, which pre-caches `HICON` handles). Every 5-second tray refresh therefore does a full PNG decode under the menu lock, blocking D-Bus menu responses (e.g., `GetLayout`, `Event`) for the duration of the decode. While not catastrophic, it is inconsistent with the Windows path's stated "pre-cached" design and can cause observable menu jitter on slower systems.

**Fix:** Pre-decode the four progress pixmaps at `initTray` time, parallel to the Windows approach:

```go
// In linuxTray struct:
progressPixmaps [5][][3]interface{} // index 1..4 = quartile 1..4

// In initTray:
tray.progressPixmaps[1] = makePixmap(trayIconProgress25Bytes)
tray.progressPixmaps[2] = makePixmap(trayIconProgress50Bytes)
tray.progressPixmaps[3] = makePixmap(trayIconProgress75Bytes)
tray.progressPixmaps[4] = makePixmap(trayIconProgress100Bytes)

// In updateTray: select from pre-decoded slice instead of calling makePixmap.
```

### WR-02: `aggregateProgress` silently treats `state:1` with `value=0` as "no active progress", masking the tray glyph for a legitimately-active session

**File:** `frontend/src/lib/aggregateProgress.ts:28`

**Issue:** When the mean of all `state:1` values is exactly `0` (e.g., a tool emits `OSC 9;4;1;0` to announce it has started but not yet made progress), `aggregateProgress` returns `0`, which causes `SetTrayProgress(0)` to revert the tray icon to the base state. The registry still holds the entry (so the tab underline is at `scaleX(0)`, invisible), but the tray gives no indication that progress is active. A user watching the tray would see no signal until the value exceeds 0.

The OSC 9;4 spec allows value=0 with state=1 (e.g., "task started"). The ProgressAddon clamps value to `[0, 100]` and fires `onChange` with `{state:1, value:0}` for this case.

**Fix:** The simplest correction is to return `1` (minimum glyph) whenever there are active `state:1` entries regardless of their mean value:

```typescript
const mean = values.reduce((a, b) => a + b, 0) / values.length
if (mean <= 0) return values.length > 0 ? 1 : 0  // at least quartile 1 if any active session
```

Alternatively, document the "value=0 is invisible" behaviour as an explicit constraint in the function JSDoc, and add a test case for it so future readers understand the invariant.

### WR-03: `build/gen_progress_icons.go` uses `bounds.Max.Y` (not corrected height) as the image height when computing bar position

**File:** `build/gen_progress_icons.go:68-72`

**Issue:** Line 69 computes `imgH := bounds.Max.Y`. If the decoded image has a non-zero `bounds.Min.Y` (e.g., a sub-image or a PNG with a non-standard origin), the bottom bar would be drawn at the wrong y-coordinates. The correct height is `bounds.Max.Y - bounds.Min.Y`, and the loop should start at `bounds.Min.Y + (height - barH)`.

In practice, `png.Decode` always returns images with `Min = {0,0}`, so the generated assets are correct. The bug is latent. Since this is a one-shot code-generation script, the risk is low, but the script is brittle against any future use with sub-images or non-standard bounds.

**Fix:**

```go
h := bounds.Max.Y - bounds.Min.Y
for y := bounds.Min.Y + h - barH; y < bounds.Max.Y; y++ {
```

## Info

### IN-01: Misleading comment in `TerminalPanel.tsx` — "App.tsx maps state:2/3/4 → state:0 by convention" is not what the code does

**File:** `frontend/src/components/TerminalPanel.tsx:574-576`

**Issue:** The comment inside the `progressAddon.onChange` subscription says "App.tsx maps state:2/3/4 → state:0 by convention". The actual `handleProgressChange` in `App.tsx` (lines 196-207) does not perform any mapping — it simply treats any `state !== 1` as a delete from the registry. The comment describes an intended convention that never materialised in code. Future contributors maintaining the state:2/3/4 deferral may assume a mapping exists and skip writing one.

**Fix:** Replace the comment with an accurate description:
```typescript
// state !== 1 (cleared, error, indeterminate, paused): remove this session
// from the registry. The tray/tab will revert once all active entries are gone.
// state:2/3/4 handling deferred to v3.3 per RESEARCH "Cuttable Inside Cuttable".
```

### IN-02: Web `#progress-underline` overlaps the top 2px of `#web-status-bar` at `position: fixed; top: 0`

**File:** `web/assets/terminal.css:235-247` / `web/terminal.html:19`

**Issue:** The `#progress-underline` element is declared `position: fixed; top: 0; height: 2px`. The `#web-status-bar` is 32px tall and rendered first in the flex column. Because the progress bar is fixed-positioned at `top: 0`, it overlays the top edge of the status bar rather than appearing below it. With `scaleX(0)` (the initial/idle state) this is invisible; when active and `z-index: 1000`, it occludes the top 2px of the status-bar text/dot.

This appears intentional (the comment says "top of the viewport regardless of page scroll"), but it conflicts with the comment "Mirrors desktop `.tab__progress`" — on desktop the underline is at the bottom of the tab strip, not over other UI chrome. If the intent is a progress indicator visible during scroll, the current placement is correct. If the intent is parity with the desktop, it should be `top: 32px` (below the status bar).

**Fix:** Clarify intent in comments. If the bar should not cover the status bar, change to:
```css
#progress-underline {
  position: fixed;
  top: 32px; /* below #web-status-bar */
  ...
}
```

### IN-03: `osc94-progress-fixture.sh` has no sleep between the 100% emit and the clear, making the 100% state nearly unobservable in manual UAT

**File:** `tests/fixtures/osc94-progress-fixture.sh:8-9`

**Issue:** Lines 8-9 are:
```bash
printf '\x1b]9;4;1;100\x07'; sleep 1
printf '\x1b]9;4;0\x07'  # clear
```

There is a `sleep 1` after the 100% emit (line 8), so the 100% state IS visible for 1 second. The `# clear` on line 9 fires immediately after. This is fine for automated testing but the comment in the e2e file (progress.spec.ts) says the UAT tester should observe 100%. The fixture is correct; this is an informational note only.

The actual concern: the script runs `set -euo pipefail` but `printf` failures (e.g., if stdout is closed) would abort the script mid-sequence, leaving the progress bar stuck. This is a fragile fixture for a CI context where stdout may be redirected.

**Fix:** Acceptable as-is for manual UAT. For CI robustness, add `|| true` after each `printf`:
```bash
printf '\x1b]9;4;1;25\x07' || true; sleep 1
```

---

_Reviewed: 2026-05-08_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
