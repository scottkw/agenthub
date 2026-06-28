---
phase: 159-web-share-chat-parity-route-shared-session-links-to-the-chat
plan: 02
gap_closure: true
status: complete
requirements: [WEBCHAT-03]
completed: 2026-06-27
---

# Plan 159-02 Summary — Web-share guest scope (WEBCHAT-03)

## Self-Check: PASSED

## What was built

Gap-closure for a scope defect found during Phase 159 live UAT: the 159-01 redirect
correctly sends remote guests to `/app/`, but `/app/` rendered the **full desktop app
shell** — the `<Sidebar>` (Home / Hub / Settings / session groups) was rendered
unconditionally in `App.tsx`, with no `mode === 'web'` gate. A guest holding a
capability for ONE session could navigate to Home / Hub / Settings and reach the open
`/api/sessions/meta` enumeration surface (lists all web-enabled sessions).

Root cause: unfinished web-mode scoping from Phase 120/155 — web mode suppressed Wails
RPCs, the Settings tab *content*, and the Welcome tab, but never the navigation chrome.
Never exposed because nothing routed a real guest to `/app/` until 159-01.

## Change

- `frontend/src/App.tsx` — wrapped `<Sidebar ... />` in `{mode !== 'web' && ( ... )}`
  (mirrors the existing `mode !== 'web'` gates on the Settings surface). The guest keeps
  the scoped surface: WebShareSessionView (terminal + chat) and the TabBar's session +
  file-browser tabs. TabBar has no new-session control, so keeping it grants no extra power.
- `frontend/e2e/web-share-scope.spec.ts` (NEW) — on `/app/?session=&cap=`, asserts the
  Sidebar `nav[aria-label="Main navigation"]` is absent while the chat toggle + xterm
  terminal render, and Hub/Settings nav buttons are unreachable.

## Verification

- RED→GREEN: new spec failed on base code (Sidebar present, count 1), passes after the gate.
- Cross-browser: `web-share-scope.spec.ts` green on chromium + firefox + webkit.
- No regression: `chat-parity.spec.ts` (release-blocking PARITY-01 gate) green; `pnpm exec tsc` clean.
- Live visual (playwright-fixture + dev-browser): sidebar=0, Home=0, Hub=0, chat-toggle=1 on the redirected `/app/` view.
- Desktop app rebuilt (`wails build -tags wailsassets`) with the fix embedded.

## Deviations / notes

- Pre-existing, NOT from this phase: `files-browser.spec.ts` scenario 13 ("file-browser
  tab mounts on web load") fails on the base commit too — Phase 155-03 made the
  WebShareSessionView the active tab, so the file-browser tab content doesn't render until
  clicked. Stale assertion; logged in TESTING.md note. Out of scope for this gap-closure.

## Files
- created: frontend/e2e/web-share-scope.spec.ts, 159-02-PLAN.md, 159-02-SUMMARY.md
- modified: frontend/src/App.tsx, TESTING.md, .planning/REQUIREMENTS.md
