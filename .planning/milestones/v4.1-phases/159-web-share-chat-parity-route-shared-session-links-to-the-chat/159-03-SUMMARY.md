---
phase: 159-web-share-chat-parity-route-shared-session-links-to-the-chat
plan: 03
gap_closure: true
status: complete
requirements: [WEBCHAT-04]
completed: 2026-06-27
---

# Plan 159-03 Summary — Web-share file-tab gating (WEBCHAT-04)

## Self-Check: PASSED

## What was built

Second gap-closure from Phase 159 live UAT. After 159-02 hid the desktop sidebar, the
guest still saw a second tab that rendered "files.read permission required" — the web
bootstrap opened the file-browser tab unconditionally. Root cause: the bootstrap assumed
the cap was opaque client-side, but `GET /api/sessions/{id}/info` returns server-verified
perms (ChatPanel already uses it for read-only state).

## Change

- `frontend/src/App.tsx`:
  - Web bootstrap now opens the WebShareSessionView tab immediately (active), then
    probes `/api/sessions/{id}/info?cap=` and opens the file-browser tab **only** when
    `perms` includes `files.read`. Fail-safe: on fetch error or missing perm, no file tab.
  - `handleOpenFileBrowser` gained an `activate = true` param; the bootstrap passes
    `false` so the file tab opens in the background while the session view stays active.
- `frontend/e2e/web-share-scope.spec.ts`: +2 tests — owner cap → "— Files" tab present
  (background) with the terminal still active; viewer cap → no file tab AND no
  "files.read permission required" takeover.

## Verification

- RED→GREEN: the viewer-cap test fails on pre-fix code (file tab + takeover present),
  passes after the gate. Confirmed by stashing the App.tsx change.
- Cross-browser: web-share-scope (3 tests) green on chromium + firefox + webkit.
- No regression: chat-parity green on all 3 browsers; `pnpm exec tsc` clean.
- Desktop app rebuilt (`wails build -tags wailsassets`) with the fix embedded.

## Files
- created: 159-03-PLAN.md, 159-03-SUMMARY.md
- modified: frontend/src/App.tsx, frontend/e2e/web-share-scope.spec.ts, TESTING.md, .planning/REQUIREMENTS.md
