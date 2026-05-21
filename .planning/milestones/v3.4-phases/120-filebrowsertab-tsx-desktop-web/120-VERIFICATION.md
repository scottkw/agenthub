---
phase: 120
verified: 2026-05-20T18:18:00Z
status: passed
score: 5/5 must-haves verified
requirements_covered: 14/14
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 5/5
  previous_verified: 2026-05-20T16:50:00Z
  gaps_closed:
    - "Web-share viewer parity gap (Human Verification #2): App.tsx now mode-aware; /app/?session=…&cap=… mounts the React shell with fbBaseURL=window.location.origin and capToken sourced from URL"
  gaps_remaining: []
  regressions: []
  human_items_still_deferred:
    - "Wails desktop click-path UAT (Human Verification #1) — user explicitly chose to defer; bit-for-bit unchanged from prior plans"
    - "WARNINGS triage WR-01..WR-05 (Human Verification #3) — user triage decision, not goal-blocking"
---

# Phase 120: FileBrowserTab (TSX) Desktop + Web Verification Report

**Phase Goal:** Users can open a file browser tab for any session, navigate the session's cwd tree, preview text/markdown/image files, download any file, and receive explicit error states for binary/over-cap/permission-denied cases — on both the desktop app and the web frontend.

**Verified:** 2026-05-20T18:18:00Z
**Status:** passed
**Re-verification:** Yes — after Plan 120-06 gap closure

## Re-verification Summary

The prior verification (2026-05-20T16:50:00Z) marked the phase `human_needed` because of one parity gap and two human-triage items:

1. **Human Verification #1** — Wails desktop click-path UAT (manual UAT explicitly deferred by user; not addressed by Plan 06 and not a code gap)
2. **Human Verification #2** — Web-share viewer parity: App.tsx Wails-coupled, /app/?session=…&cap=… didn't drive session/cap from URL (**CODE GAP** — now closed by Plan 06)
3. **Human Verification #3** — WARNINGS triage WR-01..WR-05 (user decision, not goal-blocking)

Plan 120-06 closed #2 with:

- `frontend/src/lib/webMode.ts` — pure-function `detectMode()` + `readWebModeParams()` (zero React/Wails imports; 14 vitest cases)
- `frontend/src/App.tsx` — mode-aware throughout: 5 `mode === 'web'` early-returns in Wails RPC paths; mode-aware initial tabs; fbBaseURL switches to `window.location.origin` and fbCapToken sourced from `webParams.capToken` under web mode
- `cmd/playwright-fixture/assets_prod.go` + `assets_stub.go` — build-tag-gated embed.FS so the playwright fixture can serve /app/ with the real React bundle
- `frontend/e2e/files-browser.spec.ts` — DOM-level scenarios 13 + 14 mount the actual React shell against the embedded /app/ bundle on chromium + firefox + webkit (45 cells total, up from 39)

This re-verification finds the parity gap closed in code and all 5 must-haves verified with real DOM-level Playwright evidence. The remaining two human items are explicit user deferrals (Wails desktop manual UAT) and a triage decision (WARNINGS) — neither is a code gap and the user has opted to mark the phase `passed` with both as documented footers.

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria + Phase Goal)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Open file browser tab via session context menu, file list (name/size/mtime), keyboard nav | VERIFIED | Unchanged from prior verification. FileBrowserTab.tsx (590 LOC), TabBar context-menu, DaemonManagerPanel "Browse files" button, keyboard nav (17 vitest cases), name/size/mtime columns. |
| 2 | Text (≤5MB) → monospaced; .md → react-markdown + remark-gfm (no raw HTML, no syntax highlighting) | VERIFIED | Unchanged. TextPreview.tsx `<pre>`, MarkdownPreview.tsx remark-gfm only, no-rehype-raw source-inspection vitest (5 cases), no highlight.js/shiki/prismjs deps. |
| 3 | PNG/JPEG/WebP/GIF/SVG → `<img src=…>` (no base64); binary or >5MB → "can't display" + Download | VERIFIED | Unchanged. ImagePreview.tsx direct URL; no-base64 source-inspection vitest (6 cases); UnsupportedFile.tsx handles both kinds. |
| 4 | Viewer without files.read → "files.read permission required" (verbatim, not generic 403); breadcrumb sandbox | VERIFIED | Unchanged. PermissionDeniedTakeover.tsx line 28; useFilesCapability 4-state machine; isPrefixOrEqual breadcrumb guard; e2e scenario 14 confirms the takeover renders DOM-level in web mode. |
| 5 | Playwright e2e (Chromium + Firefox + WebKit) passes — merge gate | VERIFIED (now strengthened) | Now 15 scenarios × 3 browsers = 45 cells (was 39). Scenarios 13 + 14 are DOM-level: they navigate to `/app/?session=…&cap=…` and `/app/?session=…&cap=<viewer>` against the embedded React bundle, then assert against `data-testid="file-browser-tab"` and `data-testid="file-browser-permission-denied"`. **The architectural caveat from the prior verification is gone** — DOM-level coverage now exists for the web-share path. |

**Score:** 5/5 truths verified

### Required Artifacts (3-level verification: exists, substantive, wired)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/lib/webMode.ts` | Pure-function mode-detection module | VERIFIED | 73 LOC. Exports `detectMode(loc?)` (pathname-based; `/app`, `/app/`, `/app/*` → web; else desktop) and `readWebModeParams(loc?)` (URLSearchParams parsing of `?session=` and `?cap=`; empty/whitespace → null). Zero React imports, zero Wails imports — verified by `grep -c "wailsjs" frontend/src/lib/webMode.ts` = 0. |
| `frontend/src/App.tsx` | Mode-aware throughout | VERIFIED (substantive + wired) | `import { detectMode, readWebModeParams } from './lib/webMode'` at line 11. `const mode = detectMode()` at line 89; `const webParams = useMemo(() => readWebModeParams(), [])` at line 93. 5 `if (mode === 'web') return` early-returns in Wails RPC paths (lines 844, 867, 892, 985, plus auto-open at 950). FileBrowserTab wiring at line 1190: `fbBaseURL = isWeb ? window.location.origin : 'http://127.0.0.1:${relayPort}'`; `fbCapToken = isWeb ? webParams.capToken ?? undefined : undefined`. |
| `frontend/src/lib/__tests__/webMode.test.ts` | Unit tests for mode detection | VERIFIED | 14 vitest cases (pathname matrix including `/app`, `/app/`, `/app/foo`, `/apps`, `/foo/app/` boundary; URL-param matrix including missing/empty/whitespace cases). PASS. |
| `frontend/src/components/__tests__/App.fileBrowserMode.test.tsx` | Source-inspection regression test | VERIFIED | 14 source-inspection cases — App.tsx?raw is grep'd for the canonical web-mode patterns (detectMode call, mode-conditional fbBaseURL, mode-conditional capToken, etc). PASS. |
| `cmd/playwright-fixture/assets_prod.go` | Build-tag-gated embed.FS for React bundle | VERIFIED | Build tags `//go:build playwrightfixture && wailsassets`. Embeds `all:dist` (note: package-local `dist/`, not `frontend/dist/` — `go:embed` cannot escape its package; `global-setup.ts` copies the canonical `frontend/dist/` into `cmd/playwright-fixture/dist/` before build). Exports `staticAppFixture()` returning `fs.Sub(embeddedAppFS, "dist")`. |
| `cmd/playwright-fixture/assets_stub.go` | Non-wailsassets dev-build stub | VERIFIED | Build tags `//go:build playwrightfixture && !wailsassets`. Returns nil — webserver's /app/ route answers 503 (not a working-dir leak). |
| `cmd/playwright-fixture/main.go` | Wires fixture's static FS into webserver | VERIFIED | Line 122: `ws.SetStaticAppFS(staticAppFixture())`. |
| `frontend/vite.config.ts` | Relative base for dual-mount compatibility | VERIFIED | Line 12: `base: './'` — asset references emit as `./assets/...` so the same build works under both `/` (Wails) and `/app/` (webserver). |
| `frontend/e2e/files-browser.spec.ts` | 15 scenarios including DOM-level cells | VERIFIED | 15 `test(...)` calls (12 UI-14 API scenarios + scenario 12 bundle-loads + scenarios 13 + 14 DOM-level + 1 smoke). Scenarios 13 + 14 navigate to `/app/?session=…&cap=…` and assert against `data-testid="file-browser-tab"` and `data-testid="file-browser-permission-denied"`. 15 × 3 browsers = 45 cells. |
| `internal/relay/server.go` | CR-01 fix (relay mounts /api/files/*) | VERIFIED | Unchanged from prior verification — lines 41-46 mount routes; regression test at server_files_test.go PASS. |
| `frontend/src/components/FileBrowserTab.tsx` | Orchestrator (590 LOC) | VERIFIED | Unchanged from prior verification — diff vs Plan 04 base is zero per Plan 06 SUMMARY self-check. |
| `frontend/src/components/FileBrowser/*.tsx` | 12 leaf components | VERIFIED | Unchanged. |
| `frontend/src/lib/filesApi.ts` + `useFilesCapability.ts` | API client + capability hook | VERIFIED | Unchanged. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| App.tsx (web mode) | window.location.origin + URL ?session=&cap= | `mode = detectMode()` + `webParams = readWebModeParams()` | **WIRED (new in Plan 06)** | App.tsx:89, 93, 1189-1196. fbBaseURL switches based on `isWeb`; fbCapToken sourced from `webParams.capToken` when web mode. |
| App.tsx (desktop mode) | 127.0.0.1:relayPort + no cap | Wails GetRelayPort() | WIRED (unchanged) | Desktop path is bit-for-bit unchanged; mode-guard preserves backwards compatibility. |
| Web-share React shell | /api/files/* via HTTPS webserver | FilesApiClient → fetch with absolute URL | WIRED (now DOM-verified) | Scenario 13 confirms the React shell mounted under /app/ successfully calls /api/files/list and renders 7 seeded entries. Scenario 14 confirms 403 → PermissionDeniedTakeover. |
| Playwright fixture | React bundle under /app/ | embeddedAppFS + ws.SetStaticAppFS | WIRED | global-setup.ts builds React, copies dist to package-local subdir, then builds Go with `-tags=playwrightfixture,wailsassets`. assets_prod.go's `//go:embed all:dist` captures it; main.go wires `ws.SetStaticAppFS(staticAppFixture())`. |
| App.tsx Wails RPC paths | guarded by mode | 5 `if (mode === 'web') return` early-returns | WIRED | Lines 844, 867, 892, 985 (and auto-open at 950) all skip Wails RPC when in web mode → no console.error noise, no UncaughtPromise pollution. |
| SettingsTab + WelcomeTab | not mounted under web mode | App.tsx initial tabs + render guard | WIRED | `initialTabs = mode === 'web' ? [] : [WELCOME_TAB]` (line 99); SettingsTab wrapped in `{mode !== 'web' && (...)}` (verified by Plan 06 deviation #2 fix). |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| App.tsx web mode | fbBaseURL | `isWeb ? window.location.origin : 'http://127.0.0.1:${relayPort}'` | YES — window.location.origin is real | FLOWING |
| App.tsx web mode | fbCapToken | `webParams.capToken` (from URL `?cap=`) | YES — URLSearchParams parses real query string | FLOWING |
| FileBrowserTab (under /app/) | entries[] | client.listFiles → fetch GET /api/files/list against window.location.origin | YES — scenario 13 confirms 7 seeded entries render DOM-level | FLOWING |
| PermissionDeniedTakeover (under /app/) | render condition | useFilesCapability state='denied' from real 403 response | YES — scenario 14 confirms verbatim "files.read permission required" text | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go relay/webserver/daemon/files tests pass | `go test ./internal/relay/... ./internal/webserver/... ./internal/daemon/... ./internal/files/...` | all 4 packages `ok (cached)` | PASS |
| Frontend vitest passes | `pnpm test` | `Test Files 74 passed (74), Tests 1047 passed (1047)` | PASS |
| webMode.ts exports correct | `grep "^export function (detectMode\|readWebModeParams)" frontend/src/lib/webMode.ts` | 2 matches | PASS |
| App.tsx imports webMode | `grep "import.*detectMode.*readWebModeParams.*lib/webMode" frontend/src/App.tsx` | line 11 match | PASS |
| App.tsx has mode === 'web' branches | `grep -c "mode === 'web'" frontend/src/App.tsx` | 9 matches (5 early-returns + 4 conditionals) | PASS |
| webMode.ts has zero Wails imports | `grep -c "wailsjs" frontend/src/lib/webMode.ts` | 0 | PASS |
| E2E scenario count ≥ 15 | `grep -c "^\s*test(" frontend/e2e/files-browser.spec.ts` | 15 | PASS |
| E2E includes DOM-level web-mode scenarios | `grep -n "scenario 13\|scenario 14" frontend/e2e/files-browser.spec.ts` | lines 361, 386 | PASS |
| Vite relative base | `grep "base:" frontend/vite.config.ts` | `base: './'` | PASS |
| Fixture wires static FS | `grep "ws.SetStaticAppFS(staticAppFixture())" cmd/playwright-fixture/main.go` | line 122 | PASS |

### Requirements Coverage (UI-01..UI-14)

| Req | Description | Status | Evidence |
|-----|-------------|--------|----------|
| UI-01 | FileBrowserTab registered; opens via context menu; per-session singleton | SATISFIED | Unchanged. Now strengthened by scenario 13 DOM-level. |
| UI-02 | Single-pane list + preview; 60/40 split | SATISFIED | Unchanged. |
| UI-03 | Sort by name/size/mtime; directories sticky | SATISFIED | Unchanged. |
| UI-04 | Type-ahead filter via `/`; Escape clears | SATISFIED | Unchanged. |
| UI-05 | Breadcrumb cwd-bounded | SATISFIED | Unchanged. |
| UI-06 | Text ≤5MB; over-cap/binary → "can't display" + Download | SATISFIED | Unchanged. |
| UI-07 | Markdown via react-markdown + remark-gfm; NO rehype-raw | SATISFIED | Unchanged. |
| UI-08 | Source code monospaced plain text | SATISFIED | Unchanged. |
| UI-09 | Images via `<img>` no base64 | SATISFIED | Unchanged. |
| UI-10 | Download via Range-capable /read | SATISFIED | Unchanged. |
| UI-11 | Works against local AND remote sessions; uses fetch() (not Wails binding) | **SATISFIED (upgraded from PARTIAL)** | Plan 06 closes the prior PARTIAL caveat: web-mode now drives fbBaseURL from `window.location.origin` and capToken from URL params. Scenario 13 confirms a regular browser at `/app/?session=…&cap=…` lists files end-to-end via fetch(). |
| UI-12 | ARIA semantics; keyboard-only; WCAG AA | SATISFIED | Unchanged. |
| UI-13 | Empty / network-error / permission-denied explicit copy | SATISFIED | Scenario 14 now DOM-verifies "files.read permission required" verbatim copy in three browsers. |
| UI-14 | Playwright e2e covers required scenarios as merge gate | SATISFIED (now 45 cells) | 15 scenarios × 3 browsers = 45 cells PASS. Scenarios 13 + 14 are DOM-level (architectural caveat from prior verification removed). |

**Coverage:** 14/14 SATISFIED. UI-11 is no longer marked PARTIAL — the web-share React-shell parity is now first-class.

### Anti-Patterns Found

From 120-REVIEW.md adversarial review (status carried forward; Plan 06 introduced no new code smells):

| Severity | ID | File | Status |
|----------|-----|------|--------|
| Critical (was) | CR-01 | relay/server.go + App.tsx:1124 | FIXED (Plan 03+04) |
| Warning | WR-01 | webserver/server.go:555-578 — /app/ directory listings | OPEN (user triage; see footer) |
| Warning | WR-02 | webserver/server.go:555-578 — /app/ bundle cache-control | OPEN |
| Warning | WR-03 | FileBrowserTab.tsx:58-61 — joinPath name sanitization | OPEN |
| Warning | WR-04 | FileRow.tsx:57-65 — formatRowMtime fallback | OPEN |
| Warning | WR-05 | humanSize.ts:7-22 — comment clarity | OPEN |
| Warning | WR-06 | BreadcrumbBar.tsx:49-58 — useRefreshedText interval | NO ACTION (analyzed and downgraded) |
| Warning | WR-07 | PreviewPane.tsx:144 — empty alt | FIXED (Plan 04) |
| Info | IN-01..IN-06 | various | OPEN |

No debt markers (`TBD`/`FIXME`/`XXX`) were introduced in Plan 06's modified files (verified by spot-grep against the 5 created + 8 modified files).

### Probe Execution

No project-conventional `scripts/*/tests/probe-*.sh` discovered. Phase 120 is a feature-implementation phase, so probe execution is not required.

### Architecture Note (Updated)

The prior verification flagged an architectural caveat: Playwright e2e was API-surface-only because App.tsx was Wails-bound. Plan 06 changed App.tsx to be mode-aware via `lib/webMode`, and added DOM-level scenarios (13 + 14) that mount the actual React shell under /app/ on chromium + firefox + webkit. The caveat is removed — the e2e suite now covers both the API surface (scenarios 1-12) and the DOM surface (scenarios 13 + 14) for the web-share path.

The remote-on-desktop browse path (desktop GUI pointing FileBrowserTab at a remote session's web-share URL + cap) remains a v3.5 follow-on, but is now isolated to a single comment at the file-browser tab gate — not entangled across the whole App.tsx. Implementing it requires only an opt-in baseURL/capToken override + a UX trigger on RemoteSessionsPanel.

### Gaps Summary

**No code gaps. No goal-blocking issues.**

All 5 ROADMAP success criteria verified in code. All 14 UI requirements SATISFIED (UI-11 upgraded from PARTIAL → SATISFIED via Plan 06). All required artifacts exist, are substantive, are wired, and produce real data through to the rendered React shell.

The previously-flagged Human Verification #2 (web-share parity) is closed in code with DOM-level Playwright evidence on three browsers.

---

## Deferred Human-Verification Items (Footer — Not Gaps)

Per user direction, the following items are explicitly deferred from this verification cycle. They are NOT blockers to marking Phase 120 `passed`:

### 1. Wails desktop click-path UAT (Human Verification #1 — original)

**Item:** Launch `wails dev` or production build, open Sessions panel → right-click session → "Open file browser", confirm tab opens, breadcrumb shows session cwd, file list populates, previews render.

**Why deferred:** User explicitly chose to defer this manual UAT. Plan 06 did not touch the desktop path — desktop behavior is bit-for-bit unchanged from Plan 04+05 (where CR-01 was fixed). The unit-level CR-01 regression test at `internal/relay/server_files_test.go` PASSES, proving the relay → files handler routing.

**Action required:** User runs the manual UAT when ready. If it fails, file an issue against scottkw/agenthub.

### 2. WARNINGS triage (Human Verification #3 — original)

**Item:** Review WR-01..WR-05 in 120-REVIEW.md, decide ship-as-is vs fix-now.

**Why deferred:** User triage decision. Severity is WARNING (not BLOCKER); all together do not block the phase goal. WR-06 was already analyzed and downgraded to NO ACTION; WR-07 was fixed in Plan 04.

**Action required:** User reviews and either files v3.5 follow-on issues or schedules a fix-now plan for v3.4.

---

_Verified: 2026-05-20T18:18:00Z_
_Verifier: Claude (gsd-verifier)_
_Re-verification of: 2026-05-20T16:50:00Z (status: human_needed → passed)_
