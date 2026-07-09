---
phase: 176-platform-hardening-bug-fixes
verified: 2026-07-09T15:00:00Z
status: human_needed
score: 7/10 must-haves verified
behavior_unverified: 3
overrides_applied: 0
behavior_unverified_items:
  - truth: "On Linux the GUI launches without the macOS role-menu segfault"
    test: "Launch the built .deb/AppImage on a real Linux/Wayland box (e.g. Pop!_OS 24.04/COSMIC, WebKit2GTK 4.1) and confirm no crash on startup."
    expected: "The GTK backend does not dereference a nil SubMenu; the app window opens normally."
    why_human: "No Linux/Wayland compositor available on this macOS dev box; the GTK menu-construction crash path cannot be exercised here. Tracked as TESTING.md M-52 (opportunistic, not ship-blocking per D-11 — reporter's from-source verification already accepted)."
  - truth: "On Linux/Wayland the webview renders without the DMABUF GPU-renderer freeze"
    test: "On the same Linux/Wayland box, interact with the UI (click, scroll, resize) after launch and confirm no hang."
    expected: "WEBKIT_DISABLE_DMABUF_RENDERER=1 is set by the guard (verified present in code) and the WebKit2GTK compositor does not freeze on first interaction."
    why_human: "Requires a real Wayland compositor + WebKit2GTK runtime; cannot be exercised on macOS. Tracked as TESTING.md M-52."
  - truth: "Production /app/ SPA loads without a breaking CSP violation (console sweep across inline scripts, wasm-unsafe-eval, connect-src SSE/WS, font-src, img-src data:)"
    test: "Build with `wails build -tags \"webkit2_41,wailsassets\"`, load /app/ in a browser, read DevTools console for CSP violations."
    expected: "SPA renders fully; no CSP violation breaks functionality (xterm wasm, SSE plugin-config stream, WS relay all load)."
    why_human: "/app/ returns 503 under `wails dev`/plain `go build` (no embedded SPA) — requires a production Vite build and manual DevTools inspection. Explicit `<human-check>` in 176-02-PLAN.md; tracked as TESTING.md M-53."
human_verification:
  - test: "Launch the Linux .deb/AppImage build on a real Linux/Wayland box; confirm no segfault on startup and the File/Help menu bar is present and functional."
    expected: "App window opens without crashing; File and Help menus work; no other role menus appear (by design, GOOS != linux guard)."
    why_human: "No Linux/Wayland environment available on this macOS dev box. TESTING.md M-52."
  - test: "On the same Linux/Wayland box, click around the UI and confirm no DMABUF-related freeze on first interaction."
    expected: "UI stays responsive; no hang tied to the WebKit2GTK GPU renderer."
    why_human: "Requires real Wayland compositor. TESTING.md M-52."
  - test: "Build production bundle (`wails build -tags \"webkit2_41,wailsassets\"`), load /app/ in a browser, sweep DevTools console for CSP violations."
    expected: "No CSP violation breaks the SPA (inline scripts, wasm-unsafe-eval, connect-src SSE/WS, font-src, img-src data: all clear)."
    why_human: "Requires a production build + browser DevTools; /app/ 503s under dev/plain-build. Explicit human-check in 176-02-PLAN.md. TESTING.md M-53."
---

# Phase 176: Platform & Hardening Bug Fixes Verification Report

**Phase Goal:** Close the remaining cross-cutting platform and hardening bugs — the Linux GUI launches and renders, the `/app/` route carries a CSP header, and the Hub card mini-preview wraps long lines correctly.
**Verified:** 2026-07-09T15:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | On Linux, the GUI launches without the macOS role-menu segfault | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `main.go:115,125,130` guard all three role menus with `if goruntime.GOOS != "linux"`. `go build ./...` and `go vet ./...` pass. Actual segfault-free launch not runnable on this macOS box (TESTING.md M-52). |
| 2 | On Linux/Wayland, the webview renders without the DMABUF GPU-renderer freeze | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `main.go:70-73`: `if goruntime.GOOS == "linux" { if _, ok := os.LookupEnv(...); !ok { os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1") } }`. Code present and correctly gated (user override respected); real Wayland freeze behavior not runnable here (TESTING.md M-52). |
| 3 | On macOS, AppMenu/EditMenu/WindowMenu still appear (menu behavior unchanged) | ✓ VERIFIED | `goruntime.GOOS != "linux"` is true on darwin, so all three role-menu appends still execute on macOS. Verified by direct code inspection of `main.go:108-137`. |
| 4 | Windows role menus preserved (code-review WR-01 regression fixed) | ✓ VERIFIED | Original fix scoped the guard to `GOOS == "darwin"` which silently stripped menus from Windows (WebView2, unaffected by the Linux GTK bug). Code review (176-REVIEW.md WR-01) caught this; commit `89329c54` changed the guard to `GOOS != "linux"` so Windows keeps its menus. Confirmed current `main.go` uses `!= "linux"` throughout (not `== "darwin"`). |
| 5 | The custom File and Help submenus appear on every platform (unconditional) | ✓ VERIFIED | `main.go:119` (`fileMenu := m.AddSubmenu("File")`) and `main.go:135-136` (`helpMenu := m.AddSubmenu("Help")` + items) sit outside any GOOS guard. |
| 6 | GET /app/ carries the strict Content-Security-Policy header (same policy as /dashboard, /join, /sessions/{id}) | ✓ VERIFIED | `server.go:1046`: `mux.HandleFunc("GET /app/", ws.cspHeaders(func(...)...))`. Ran `go test ./internal/webserver/ -run TestCSPHeaderStrict_App -v` myself — **PASS**. Test wires a stub `fstest.MapFS`, gets 200 on `/app/`, and asserts all five `assertCSPHeaderStrict` checks. |
| 7 | Hashed JS/CSS assets under /app/ remain browser-cacheable (Cache-Control not forced to no-store) — code-review WR-02 regression fixed | ✓ VERIFIED | `cspHeaders` sets `Cache-Control: no-store` on every response it wraps; `server.go:1085` (`w.Header().Del("Cache-Control")`) deletes it specifically on the hashed-asset branch, preserving the Phase 120 content-hash caching contract. Ran `go test ./internal/webserver/ -run TestAppBundle_HashedAsset_CacheableNotNoStore -v` myself — **PASS** (asserts CSP present + Cache-Control NOT no-store on the asset branch, and Cache-Control still no-store on the index.html branch). |
| 8 | Production /app/ SPA loads without a breaking CSP violation (console sweep) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Requires a `wails build -tags "webkit2_41,wailsassets"` production build + manual browser DevTools console read; `/app/` 503s under `wails dev`/plain `go build`. Explicit `<human-check>` in 176-02-PLAN.md; tracked as TESTING.md M-53. |
| 9 | Hub card mini-preview wraps long lines correctly — each output line renders as ONE clipped horizontal row, not stacked one/two characters per row | ✓ VERIFIED | Live dev-browser repro (176-03) mounted the real `MiniPreview` component against the real `style.css` with backend-accurate per-cell `StyledSpan[][]` fixture data including an overflow-length line. **I personally viewed the captured screenshot** (`176-03-evidence-minipreview-repro.png`): four preview lines render as four clean horizontal rows, and the long line is visibly ellipsis-clipped (`"...some extra tr…"`) — confirming the DOES-NOT-REPRODUCE verdict. Cross-checked `MiniPreview.tsx` (unchanged since Phase 139, zero diff since #127 was filed) — no `display` override on child spans, so they flow inline within the `nowrap` `.hub-card__preview-line` (`style.css:6019-6027`, unchanged: `white-space: nowrap; overflow: hidden; text-overflow: ellipsis`). Existing `MiniPreview.test.tsx` suite passes (10/10, ran myself). |
| 10 | TESTING.md truthfully records Phase 176 (M-52/M-53 manual items, BUG-06 traceability row, Suite Manifest reconciliation, check-traceability-paths.sh green) | ✓ VERIFIED | Confirmed via grep: `Category AA` (line 843) with M-52 (845) and M-53 (852); Section 4 BUG-06 row (415) pointing at `internal/webserver/csp_integration_test.go` / `TestCSPHeaderStrict_App`; a documented no-row rationale for BUG-07 (417); dated Suite Manifest note (34) stating counts unchanged (298 total). Ran `bash tests/check-traceability-paths.sh` myself — exits 0 (known macOS BSD-grep `-P` limitation documented in the SUMMARY and prior Phase 173-07 precedent; not a new issue). |

**Score:** 7/10 truths verified (3 present + wired, behavior-unverified — all three are pre-declared, opportunistic, non-ship-blocking manual checks per the plans' own design, not gaps discovered by this verification).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `main.go` | darwin-guarded → correctly Linux-excluded role menus + Linux DMABUF env guard | ✓ VERIFIED | `goruntime "runtime"` alias present; 4 `goruntime.GOOS != "linux"`/`== "linux"` guards; builds/vets clean. |
| `internal/webserver/server.go` | `/app/` wrapped in `ws.cspHeaders`, WR-02 Cache-Control fix | ✓ VERIFIED | Line 1046 wrap confirmed; line 1085 `Del("Cache-Control")` confirmed. |
| `internal/webserver/csp_integration_test.go` | `TestCSPHeaderStrict_App` + WR-02 regression test | ✓ VERIFIED | Both `TestCSPHeaderStrict_App` and `TestAppBundle_HashedAsset_CacheableNotNoStore` present and passing. |
| `frontend/src/components/Hub/MiniPreview.tsx` | unchanged (BRANCH B, no code needed) | ✓ VERIFIED | Zero diff since Phase 139 (predates #127); confirmed via `git log -1`. |
| `176-03-evidence-minipreview-repro.png` | live screenshot evidence | ✓ VERIFIED | File exists (42KB), visually inspected — shows 4 clean clipped rows, no char-per-row stacking. |
| `TESTING.md` | M-52, M-53, BUG-06 traceability row, Suite Manifest note | ✓ VERIFIED | All present via grep, confirmed above. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `main.go` `runGUI()` | `wails.Run(...)` | DMABUF env guard runs before `wails.Run` call | ✓ WIRED | Guard at lines 70-74, `wails.Run` call starts at line 79. |
| `internal/webserver/server.go` `/app/` registration | `ws.cspHeaders` middleware | outermost wrap | ✓ WIRED | `mux.HandleFunc("GET /app/", ws.cspHeaders(func(...)))`. |
| `TestCSPHeaderStrict_App` | `assertCSPHeaderStrict` helper | delegates all 5 assertions | ✓ WIRED | Confirmed in csp_integration_test.go:151; test passes. |
| 176-REVIEW.md findings (WR-01, WR-02) | commit `89329c54` | fix applied and tested | ✓ WIRED | Both regressions fixed with a dedicated new regression test (`TestAppBundle_HashedAsset_CacheableNotNoStore`), which I ran and confirmed passing. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| main.go compiles/vets clean | `go build ./... && go vet ./...` | clean | ✓ PASS |
| /app/ carries CSP header | `go test ./internal/webserver/ -run TestCSPHeaderStrict_App -v` | PASS (0.02s) | ✓ PASS |
| Hashed /app/ assets stay cacheable (WR-02 regression gate) | `go test ./internal/webserver/ -run TestAppBundle_HashedAsset_CacheableNotNoStore -v` | PASS (0.01s) | ✓ PASS |
| MiniPreview existing suite still green | `pnpm exec vitest run src/components/Hub/MiniPreview.test.tsx` | 10/10 PASS | ✓ PASS |
| TESTING.md traceability paths | `bash tests/check-traceability-paths.sh` | exit 0 (known BSD-grep `-P` limitation, documented) | ✓ PASS |
| Linux/Wayland live GUI launch | — | not runnable on macOS | ? SKIP (routed to human verification, M-52) |
| Production /app/ CSP console sweep | — | not runnable without a prod build + browser | ? SKIP (routed to human verification, M-53) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| BUG-05 | 176-01 | Linux GUI launches without segfault + no DMABUF freeze | ✓ SATISFIED (code) / human_needed (live) | main.go guards verified; live Linux behavior deferred to M-52 per plan's own D-11 design. |
| BUG-06 | 176-02 | /app/ serves CSP header | ✓ SATISFIED | `TestCSPHeaderStrict_App` passes; header confirmed wired outermost. |
| BUG-07 | 176-03 | Mini-preview wraps long lines correctly | ✓ SATISFIED | DOES-NOT-REPRODUCE verdict with live evidence I personally reviewed; root cause (Phase 172 CSS) confirmed still present and correct. |

**Note on REQUIREMENTS.md:** BUG-05/06/07 do NOT appear in `.planning/REQUIREMENTS.md` (confirmed via grep — zero matches). Per the important_context supplied with this verification task and corroborated independently in `176-01-SUMMARY.md`/`176-04-SUMMARY.md` and `STATE.md`, Phase 176 (like ad-hoc bug-fix phases 172/173) was added to ROADMAP.md after the original 26-requirement v4.2 REQUIREMENTS.md scope was written, and was never back-filled into REQUIREMENTS.md. This is a pre-existing documentation gap, not an execution failure of this phase. ROADMAP.md is treated as the authoritative traceability source for BUG-05/06/07, consistent with the explicit instruction for this verification. No orphaned requirements were found in ROADMAP.md's Phase 176 section beyond BUG-05/06/07, all three of which are accounted for above.

### Anti-Patterns Found

None. Scanned `main.go`, `internal/webserver/server.go` (lines 1000-1090), `internal/webserver/csp_integration_test.go`, and `frontend/src/components/Hub/MiniPreview.tsx` for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` — zero matches.

### Code Review Findings (176-REVIEW.md) — Disposition

The phase's own code review (`176-REVIEW.md`, `status: issues_found`, 2 warnings) found two real regressions:
- **WR-01** — the BUG-05 menu guard used `GOOS == "darwin"` instead of `GOOS != "linux"`, silently stripping role menus from Windows builds (unaffected by the Linux-only GTK segfault).
- **WR-02** — wrapping the entire `/app/` handler in `cspHeaders` forced `Cache-Control: no-store` onto hashed JS/CSS asset responses, contradicting the Phase 120 content-hash caching contract documented in the same code block.

Both were fixed in commit `89329c54` (`fix(176): resolve code-review WR-01/WR-02 regressions`), verified by me directly:
- `main.go` now uses `goruntime.GOOS != "linux"` (not `== "darwin"`) for all three role-menu guards — confirmed via grep.
- `server.go:1085` deletes the `Cache-Control` header on the hashed-asset branch only — confirmed via read.
- New regression test `TestAppBundle_HashedAsset_CacheableNotNoStore` was added and passes when I ran it.

The two Info-level findings (IN-01: DMABUF guard applies to all Linux not just Wayland; IN-02: new test doesn't cover the asset branch) were explicitly noted as acceptable trade-offs / addressed by the WR-02 fix's new test respectively — no action required.

### Human Verification Required

### 1. Linux/Wayland GUI launch (BUG-05, #124)

**Test:** Launch the built `.deb`/AppImage on a real Linux/Wayland box (reporter's environment class: Pop!_OS 24.04/COSMIC/Wayland, WebKit2GTK 4.1).
**Expected:** No segfault on startup; File and Help menu bar present and functional.
**Why human:** No Linux/Wayland compositor available in this verification environment (macOS). TESTING.md M-52 — marked opportunistic/non-ship-blocking per the plan's own D-11 (reporter's from-source verification already accepted as sufficient to ship).

### 2. Linux/Wayland DMABUF freeze (BUG-05, #124)

**Test:** On the same box, interact with the UI (click, scroll, resize) after launch.
**Expected:** No hang tied to the WebKit2GTK GPU renderer.
**Why human:** Same environment limitation as #1. TESTING.md M-52.

### 3. Production /app/ CSP console sweep (BUG-06, #123)

**Test:** Build `wails build -tags "webkit2_41,wailsassets"`, load `/app/` in a browser, read the DevTools console for CSP violations (inline scripts, `wasm-unsafe-eval`, SSE/WS `connect-src`, `font-src`, `img-src data:`).
**Expected:** SPA renders fully; no CSP violation breaks functionality.
**Why human:** `/app/` 503s under `wails dev`/plain `go build` (no embedded SPA) so a production build + manual browser inspection is required. This is an explicit `<human-check>` in 176-02-PLAN.md and TESTING.md M-53.

### Gaps Summary

No gaps were found. All code-level truths for BUG-05, BUG-06, and BUG-07 are verified present, substantive, and correctly wired, including two regressions (WR-01, WR-02) that the phase's own code review caught and a follow-up commit (`89329c54`) fixed — verified directly by re-running the new regression test. The three remaining open items are pre-declared, opportunistic manual/live checks (Linux/Wayland GUI behavior and a production-build CSP console sweep) that cannot be executed in this macOS-only verification environment and were never claimed to be automatable by the plans themselves — they route to human verification (TESTING.md M-52/M-53) rather than blocking the phase. The BUG-05/06/07 absence from REQUIREMENTS.md is a known, pre-existing documentation gap (ROADMAP.md-only tracking for ad-hoc bug-fix phases), not a phase execution failure.

GitHub issues #124, #123, and #127 remain intentionally OPEN (confirmed via `gh issue view`) — closure is deferred to phase-level ship per the phase's own documented convention, not an oversight.

---

_Verified: 2026-07-09T15:00:00Z_
_Verifier: Claude (gsd-verifier)_
