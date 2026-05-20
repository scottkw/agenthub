# Phase 120: FileBrowserTab.tsx (Desktop + Web) - Context

**Gathered:** 2026-05-20
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Users can open a file browser tab for any session, navigate the session's cwd tree, preview text/markdown/image files, download any file, and receive explicit error states for binary/over-cap/permission-denied cases — on both the desktop app and the web frontend.

**Requirements:** UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07, UI-08, UI-09, UI-10, UI-11, UI-12, UI-13, UI-14

</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP success criteria)
- Single-tab file browser (per session) — keyboard nav (arrow/Enter/Backspace/Tab)
- Two-pane layout: file list (left) + preview pane (right)
- Markdown: `react-markdown` + `remark-gfm`. No raw HTML. No syntax highlighting in v3.4.
- Images: `<img src="/api/files/read?...">` direct URL (no base64-in-state)
- Over-cap (>5MB) / binary: "Sorry, we can't display this file" + Download button
- Viewer without `files.read`: "files.read permission required" (not generic 403)
- Path safety: breadcrumb bar bounded to session cwd; typed/pasted/clicked paths cannot escape
- E2E: Playwright Chromium + Firefox + WebKit covering 12 scenarios — merge gate

### Claude's Discretion
- Component file location (likely `frontend/src/components/FileBrowserTab.tsx`)
- State management (likely React hooks + URL query for breadcrumb state)
- API client wrapper around /api/files/list, /stat, /read
- Capability detection (parse cap token Perms or call /api/cap/info if exists)
- Loading/skeleton states between API calls
- Tab open trigger: session context menu integration + Sessions panel button

</decisions>

<code_context>
## Existing Code Insights

- Frontend lives in `frontend/` (React + likely Vite + TypeScript)
- Existing tab system / session UI patterns — file browser opens as a tab
- Existing API client wrappers (for session terminal stream, etc.)
- CSP policy: `script-src 'self'` + `style-src 'self' 'unsafe-inline'` + `'wasm-unsafe-eval'`
- Phase 119 mounted /api/files/list /stat /read /read(HEAD) under requireFilesRead on the webserver
- Phase 118 daemon mux serves the same routes on Unix socket (desktop app uses daemon socket; web uses webserver routes)
- Desktop app is Wails-based (per memory: production builds use -tags wailsassets)

</code_context>

<specifics>
## Specific Ideas

Components likely needed:
- `FileBrowserTab` (top-level, accepts sessionID)
- `FileListPane` (left pane: virtualized list, name/size/mtime columns, keyboard nav)
- `PreviewPane` (right pane: dispatches to TextPreview, MarkdownPreview, ImagePreview, UnsupportedFile)
- `BreadcrumbBar` (path navigation bounded to session cwd)
- `FilesApiClient` (typed wrapper around /api/files/*)
- `useFilesCapability` hook (detects files.read presence)

Playwright suite at `frontend/tests/e2e/files-browser.spec.ts` or similar.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
