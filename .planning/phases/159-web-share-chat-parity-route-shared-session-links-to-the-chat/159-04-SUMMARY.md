---
phase: 159-web-share-chat-parity-route-shared-session-links-to-the-chat
plan: 04
gap_closure: true
status: complete
requirements: [WEBCHAT-05]
completed: 2026-06-27
---

# Plan 159-04 Summary — Web-share tab rename suppression (WEBCHAT-05)

## Self-Check: PASSED

## What was built

Third gap-closure from Phase 159 live UAT. The tab dropdown menu let a web-share guest
"rename" the session tab (and offered "Save Terminal As…").

Investigation (systematic-debugging, root cause first): `RenameSession` / `SaveTerminalSession`
are Wails RPCs that delegate through the runtime bridge injected by the Go host. There is no
HTTP session-rename endpoint (only `/api/files/rename` for files). Verified live via the
fixture + dev-browser: in the `/app/` bundle `window.go` and `window.runtime` are both
`undefined`, and a rename fires zero network requests. So a guest rename **never reached the
host** — it failed silently (caught) and relabeled only the local browser tab. The affordance
is nonetheless removed for clarity.

## Change

- `frontend/src/components/TabBar.tsx`: new `webMode` prop. When true, gates off the
  double-click rename, the right-click `onContextMenu`, and the chevron "Session menu"
  button — removing all access to Rename / Save Terminal As / Browse files for guests. The
  × close button stays.
- `frontend/src/App.tsx`: passes `webMode={mode === 'web'}` to TabBar.
- `frontend/e2e/web-share-scope.spec.ts`: +1 test (no chevron/menu; double-click does not
  open the rename input).

## Verification

- RED→GREEN: the rename test fails on pre-fix code (chevron present, rename input opens),
  passes after the gate. Confirmed by stashing the src changes.
- Live: `window.go`/`window.runtime` undefined; rename = 0 network requests (proves no host propagation).
- Cross-browser: web-share-scope (4 tests) green on chromium + firefox + webkit; chat-parity green; tsc clean.
- Desktop app rebuilt (`wails build -tags wailsassets`).

## Files
- created: 159-04-PLAN.md, 159-04-SUMMARY.md
- modified: frontend/src/components/TabBar.tsx, frontend/src/App.tsx, frontend/e2e/web-share-scope.spec.ts, TESTING.md, .planning/REQUIREMENTS.md
