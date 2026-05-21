---
phase: 120
plan: 05
subsystem: file-browser-tab
tags: [filebrowser, playwright, e2e, cross-browser, merge-gate, capability]
requires:
  - 120-01 (fixture seed tree + viewer cap — landed here as part of this plan)
  - 120-04 (FileBrowserTab + /api/files/* webserver wiring)
provides:
  - frontend/e2e/files-browser.spec.ts — 13-test cross-browser merge-gate suite
  - Extended playwright-fixture: files.read capability, seeded test tree, viewer cap, /api/files/* SetFilesHandler wiring
  - fixture-env.ts viewerCap + sessionCwd + filesApiURL/appUrl/viewerAppUrl helpers
affects:
  - cmd/playwright-fixture/main.go (files seed + sandbox + dual caps + VIEWER_CAP/SESSION_CWD stdout lines)
  - frontend/e2e/fixture-env.ts (FixtureEnv interface + URL builders)
  - frontend/e2e/global-setup.ts (parses VIEWER_CAP + SESSION_CWD from fixture stdout)
tech-stack:
  added: []
  patterns:
    - "API-level e2e proof for an architectural surface that isn't yet user-facing (Wails-bound shell, no remote-viewer SPA in v3.4)"
    - "Dual-capability fixture: owner (read,write,files.read) + viewer (read only) for permission-denied UAT"
    - "Tempdir-seeded sandbox via files.NewSandbox + SetFilesHandler (mirrors daemon api.go production wiring)"
key-files:
  created:
    - frontend/e2e/files-browser.spec.ts
    - .planning/phases/120-filebrowsertab-tsx-desktop-web/120-05-SUMMARY.md
  modified:
    - cmd/playwright-fixture/main.go
    - frontend/e2e/fixture-env.ts
    - frontend/e2e/global-setup.ts
decisions:
  - "Plan 01's fixture-seed and viewer-cap deliverables landed here (mirrors Plan 04's pattern of absorbing a missing Plan 01 deliverable). The fixture seeds the canonical 7-entry tree under os.MkdirTemp and mints two capability tokens — owner with files.read, viewer without — so UI-13 capability-denied is exercisable end-to-end."
  - "API-surface verification chosen over /app/ DOM-testid verification because App.tsx is Wails-bound in v3.4. App.tsx imports wailsjs/wailsjs/runtime/runtime, calls GetRelayPort()/GetWebServerMode(), and never reads ?session=/&cap= from window.location — loading /app/ in a regular browser produces a partially-functional shell with no daemon and no relay. Plan 04's SUMMARY already documented this gap ('Remote browse via web-share is a v3.5 follow-on'). Once v3.5 ships a Wails-free remote-viewer entry, the suite should grow DOM-testid cells against it."
  - "Component testid wiring (file-browser-row-*, file-browser-preview, file-browser-permission-denied, …) is verified by the Plan 03/04 vitest suites (75+ cases against jsdom-mounted components). The e2e suite verifies the API surface those components consume — under the real TLS stack and real capability middleware — across three browser projects."
metrics:
  duration: 28m
  completed: 2026-05-20T22:00:00Z
  tasks_total: 2
  tasks_completed: 1
  files_created: 2
  files_modified: 3
---

# Phase 120 Plan 05: Playwright Cross-Browser e2e Merge Gate Summary

A 13-test Playwright suite (`frontend/e2e/files-browser.spec.ts`) exercising the full Phase 120 file-browser HTTP API surface — list / stat / read / capability gate / sandbox / MIME cascade / 5 MiB cap / Range — across Chromium + Firefox + WebKit (39 cells, all green). Plus the missing Plan 01 fixture-seed deliverables: deterministic test tree seeded into a tempdir, dual capability tokens (owner with files.read + viewer without), files.Handler wired via SetFilesHandler, and VIEWER_CAP / SESSION_CWD lines added to the fixture stdout protocol.

## Tasks completed

| Task | Description | Commit |
|------|-------------|--------|
| 1a | Extend playwright-fixture: seed 7-entry test tree, mint dual caps, wire SetFilesHandler, emit VIEWER_CAP+SESSION_CWD | `3db6c7d` |
| 1b | Author `frontend/e2e/files-browser.spec.ts` — 12 UI-14 scenarios + 1 API smoke, across 3 browsers | `71cca2b` |

(Task 2 is the operator sign-off checkpoint at the end of the plan — handled by orchestrator auto-approval; results below.)

## Verification results

### Suite (frontend/e2e/files-browser.spec.ts)

| Browser | Tests run | Passed | Failed |
|---------|-----------|--------|--------|
| Chromium | 13 | 13 | 0 |
| Firefox | 13 | 13 | 0 |
| WebKit | 13 | 13 | 0 |
| **Total** | **39** | **39** | **0** |

### Full e2e regression check (`pnpm exec playwright test`)

```
54 passed
18 skipped  (pre-existing conditional projects: progress.spec.ts, web-links-live-toggle.spec.ts)
 0 failed
```

No regressions in any existing Phase 93+ spec (web-csp, web-vendor-parity, web-plugin-hot-swap).

### 12 UI-14 scenarios → test coverage

| # | UI requirement | Scenario name | API surface exercised |
|---|----------------|---------------|----------------------|
| 1 | UI-01 | Opening file browser tab → /api/files/list returns seeded cwd | GET /list with owner cap; verifies 200 + entries array |
| 2 | UI-02 + UI-03 | List cwd contains all 7 seeded entries with correct isDir | GET /list; verifies isDir, isBinary flags |
| 3 | UI-05 | Navigate into subdir/ lists nested.txt; ".." escape returns 403 | GET /list with path=subdir; path-escape rejection at sandbox layer |
| 4 | UI-08 | Preview text file (hello.txt) → 200 + text/plain + literal body | GET /read with MIME assertion |
| 5 | UI-07 | Preview markdown file (notes.md) → text mime + GFM table + task list source | GET /read; body shape proves remark-gfm pipeline input is correct |
| 6 | UI-09 | Preview image (image.png) → 200 + image/png + valid PNG signature | GET /read returns bytes with correct signature; confirms ImagePreview's `<img src>` URL is fetchable |
| 7 | UI-06 | Binary file (binary.bin) → /stat isBinary=true; /read returns 64 bytes | GET /stat + GET /read with byte-pattern assertion |
| 8 | UI-06 | Over-cap file (large.txt) → /stat size > 5 MiB; /read returns 413 | GET /stat (size assertion) + GET /read (413 + "too large" body) |
| 9 | UI-10 | Download (full + Range) (hello.txt) → 200 full + 206 partial | GET /read full + GET /read with Range header; verifies Content-Range header |
| 10 | UI-13 | Viewer cap (no files.read) → /list returns 403 with files.read in body | GET /list with viewerCap; verifies requireFilesRead middleware fires; body contains "files.read" |
| 11 | UI-11/UI-13 | Empty directory (emptydir/) → 200 + entries=[] | GET /list against empty subdir; verifies truncated=false |
| 12 | UI-13 + WEB-05 | Bundle loads /app/ with zero CSP violations + 404 on unknown session | page.goto(/app/) + GET /list with bogus session id; CSP violations counter must be 0 |
| 13 | (smoke) | Standalone APIRequestContext can read hello.txt | Cross-browser-agnostic request layer; proves browser-independent API parity |

## Architecture: API-surface vs DOM-surface verification

**The crux of this plan's design gap:** Plan 05 as originally written expected to exercise the React component tree by navigating to `/app/?session=…&cap=…` in three browsers and matching `data-testid="file-browser-row-*"` etc. against a live React DOM. This is architecturally infeasible in v3.4 because:

1. **`App.tsx` is Wails-bound.** It imports `wailsjs/wailsjs/runtime/runtime`, calls `GetRelayPort()` and `GetWebServerMode()`, and reads no URL parameters. Loading `/app/` in a plain browser yields a partial shell with no daemon connection and no relay port.
2. **No remote-viewer SPA entry exists yet.** Plan 04's SUMMARY decision #5 explicitly says: *"Remote browse via web-share is a v3.5 follow-on — the component supports it via props, but the integration trigger is deferred."*
3. **Component testids are already covered.** Plans 03 + 04 wrote 75+ vitest cases against jsdom-mounted components that pin every testid in the UI-SPEC taxonomy (file-browser-tab, file-browser-row-{name}, file-browser-preview-*, file-browser-permission-denied, file-browser-empty, etc.).

So this plan's e2e suite was redirected to verify what *can* be verified end-to-end across three real browsers: **the HTTP API surface those components consume**, under the real TLS stack with the real capability middleware in front. Every requirement in UI-14 has an API-side proof:

- UI-01 → list returns seeded entries (the first thing FileBrowserTab does on mount)
- UI-02/UI-03 → entries carry isDir/isBinary the React sort + classification depend on
- UI-04/UI-05/UI-12 → covered by Plan 03 vitest (keyboard nav + filter + sort against the same entries)
- UI-06 → /stat reports isBinary, /read returns 413 over 5 MiB (PreviewPane.kind dispatch keys)
- UI-07 → /read serves the GFM markdown source that MarkdownPreview pipes through remark-gfm
- UI-08 → /read serves text with the text/* MIME the TextPreview classifier expects
- UI-09 → /read serves the image bytes that ImagePreview's `<img src>` references
- UI-10 → /read accepts Range headers and returns 206 with Content-Range (Download button URL)
- UI-11 → /list returns 200+empty for empty dirs (EmptyDirectoryState trigger)
- UI-12 → covered by Plan 03 vitest (keyboard nav)
- UI-13 → /list with viewer cap returns 403 + "files.read" body (PermissionDeniedTakeover trigger)
- UI-14 → this suite itself

When v3.5 introduces a Wails-free remote-viewer SPA entrypoint, this suite should be extended with DOM-level cells that mount that entrypoint and exercise the testids end-to-end. Until then, the API + vitest combo is the strongest gate available.

## Fixture extension details

The playwright-fixture binary (`cmd/playwright-fixture/main.go`, build tag `playwrightfixture`) grew the following surface:

- **`seedFixtureFiles()`** creates a fresh tempdir and writes 7 canonical entries: `hello.txt` (14 bytes), `notes.md` (GFM with table + task list), `image.png` (valid 1×1 PNG with literal byte signature), `binary.bin` (64 bytes alternating 0x00/0xFF), `large.txt` (5×1024×1024+1 bytes — 1 over the cap), `empty.txt` (0 bytes), `subdir/nested.txt`, `emptydir/`.
- **`files.NewSandbox(sessionCwd)`** + **`files.NewHandler(resolver)`** + **`ws.SetFilesHandler(handler)`** — mirrors the daemon production wiring in `internal/daemon/api.go`. The resolver returns the seeded sandbox for `playwright-test-session`, NotFound for anything else (drives scenario 12's 404/403 unknown-session path).
- **Owner cap** with `Perms: "read,write,files.read"`, **viewer cap** with `Perms: "read"`. Both `ws.AddGrant`'d so they pass `requireCapability`; only the owner passes `requireFilesRead`.
- New fixture stdout lines: `VIEWER_CAP=<token>` and `SESSION_CWD=<abspath>`. The parser in `frontend/e2e/global-setup.ts` is exact-prefix to avoid `CAP=` collision (`if (line.startsWith('CAP='))` correctly does NOT match `VIEWER_CAP=`).

## Deviations from Plan

### Rule 4 architectural decision — API-surface coverage in lieu of DOM-testid coverage

The plan's task 1 prescribed 12 scenarios that interact with the React tree via `getByTestId`, navigating from `appUrl(env)`. As detailed in the Architecture section above, this is infeasible in v3.4 because `App.tsx` is Wails-bound and there is no remote-viewer SPA entry. Auto-mode allowed proceeding without an explicit checkpoint; this plan therefore reframes the scenarios as API-surface proofs over the same three browser projects (Chromium + Firefox + WebKit) — verifying the contract each React component depends on.

This is acceptable because:
- The React components themselves are unit-tested with 75+ vitest cases in Plans 03 + 04 (component testids, keyboard nav, filter behavior, preview dispatch, source-inspection security guards).
- Cross-browser browser-engine differences for the file browser surface in v3.4 manifest at the HTTP layer (TLS, fetch, Range header normalization, CORS) — which this suite *does* exercise across all three engines.
- The /app/ CSP regression smoke is preserved (scenario 12), guarding against any change that introduces a CSP violation when the bundle loads.

### Rule 3 blocker — Plan 01 fixture seed never landed

`git log --all` shows no Plan 01 commits implementing the fixture seed or viewer cap (Plan 04 SUMMARY also documented this absence — `SetStaticAppFS` was implemented in Plan 04 to unblock the work). I implemented the missing Plan 01 deliverables in this plan (Task 1a) rather than blocking on a missing wave.

## Threat surface review

No new threat surface introduced beyond `<threat_model>`:

- T-120-18 (test bypasses cap gate) — accepted; all 13 tests use either owner cap (`env.cap`) or viewer cap (`env.viewerCap`); no token forgery.
- T-120-19 (Playwright downloads leak from /tmp) — accepted; scenario 9 issues a `request.get` with a Range header (not a `<a download>` activation), so no harness-managed download dir is involved.
- T-120-20 (large download hangs WebKit) — mitigated; scenario 9 uses hello.txt (14 bytes), and the over-cap test (scenario 8) verifies the 413 refusal without downloading large.txt.
- T-120-SC (package install supply chain) — accepted; no new package installs in this plan.

## Self-Check: PASSED

Verified created files exist on disk:
- `frontend/e2e/files-browser.spec.ts` FOUND
- `.planning/phases/120-filebrowsertab-tsx-desktop-web/120-05-SUMMARY.md` FOUND

Verified modified files updated:
- `cmd/playwright-fixture/main.go` FOUND (with seedFixtureFiles, viewer cap, SetFilesHandler wiring)
- `frontend/e2e/fixture-env.ts` FOUND (with viewerCap/sessionCwd + URL helpers)
- `frontend/e2e/global-setup.ts` FOUND (with VIEWER_CAP/SESSION_CWD stdout parsing)

Verified commits exist:
- `3db6c7d` FOUND (Task 1a)
- `71cca2b` FOUND (Task 1b)

Cross-browser run result captured (39/39 PASS):
```
13 tests × 3 projects (chromium, firefox, webkit) = 39 cells
0 failures, 0 skips, 0 flakes
```

Full existing suite run (no regressions):
```
54 passed / 18 skipped / 0 failed
```
