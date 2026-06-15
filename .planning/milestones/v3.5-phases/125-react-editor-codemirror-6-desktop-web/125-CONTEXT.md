# Phase 125: React Editor (CodeMirror 6) — Desktop + Web - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Users can open any text file in a CodeMirror 6 editor with syntax highlighting, save changes atomically via Cmd/Ctrl+S with conflict detection, and perform all write operations (create file, mkdir, delete, rename, cross-directory move, single and multi-file upload) from the `FileBrowserTab` — on both the desktop app and the web-share surface.

**Depends on:** Phase 123 (write API frozen — COMPLETE) and Phase 124 (capability model live, webserver write routes accessible — COMPLETE). Milestone centrepiece.

**Requirements:** EDIT-01..EDIT-13

**Cross-surface parity (release-blocking):** the editor + write affordances must work on BOTH desktop app and web-share. All write affordances gated on `canWrite` (files.write capability).
</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP success criteria)
- Editor: CodeMirror 6, syntax highlighting by extension (Go, TS, Python, JSON, YAML, Markdown, Bash, HTML, CSS, + common langs).
- Edit button absent for binary files and callers without `files.write`.
- Files > 500 KB → large-file warning before edit; files approaching 5 MB cap → disable syntax highlighting with in-editor notice.
- Save: Cmd/Ctrl+S → atomic write (temp+sync+rename) with `If-Match: <etag>` header. Three-state save indicator (idle / saving… / saved ~1.5s). Dirty-state bullet/asterisk. Unsaved-changes guard is React-level only — NO `beforeunload` (Wails blocks it).
- Conflict: If-Match mismatch (HTTP 412) → "This file was modified by another process" with [Force overwrite] / [Save as new file] / [Discard my changes]. Buffer NEVER silently discarded.
- Write affordances (create file, mkdir, delete, rename, cross-dir move via "Move to…" picker, single + multi-file upload w/ per-file progress, drag-and-drop) visible/operable only when canWrite. 409 collision → "A file named X already exists. Replace it?" with Cancel as default.
- Recursive directory delete → confirm with file count.
- Testing: Playwright cross-browser e2e (Chromium + Firefox + WebKit). Zero CSP violations. `vendor_drift_test.go` keeps CodeMirror packages version-matched.

### Claude's Discretion
Component decomposition, CodeMirror extension wiring, state management, vendoring/bundling approach — at Claude's discretion guided by success criteria, the UI-SPEC, and existing FileBrowserTab patterns.
</decisions>

<code_context>
## Existing Code Insights

Gathered during plan-phase research. Anchors: existing `FileBrowserTab.tsx` / `FileBrowser/` components (v3.4 read-side), the Phase 124 `canWrite` signal + `files.write` cap, the daemon/webserver write routes (PUT write w/ If-Match? — confirm ETag support), the read route's ETag/Last-Modified headers, frontend bundling (Wails embed + CSP), and existing Playwright/e2e setup if any.

</code_context>

<specifics>
## Specific Ideas

Refer to ROADMAP Phase 125 success criteria (precise + testable). CodeMirror 6 is the mandated editor. ETag-based optimistic concurrency (If-Match → 412) is the conflict model.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
