---
phase: 125-react-editor-codemirror-6-desktop-web
plan: 02
subsystem: frontend-editor, files-api, webserver-test
tags: [codemirror, cm6, editor, languageFor, canWrite, etag, vendor-drift, tdd]
dependency_graph:
  requires:
    - 125-01-SUMMARY.md  # vendor_drift_test + ETag + WRITE_CAP fixture
  provides:
    - 17 @codemirror/* packages installed at pinned versions
    - languageFor.ts: extension → CM6 LanguageSupport (Bash via legacy-modes)
    - filesApi.ts: readFileText returns etag from ETag header
    - filesApi.ts: probeWrite method for canWrite web-share probe
    - useFilesCapability.ts: canWrite resolution (desktop signal / web-share probe)
    - Editor.tsx: CM6 mount, Compartment toggle, large-file/binary guards
    - PreviewPane.tsx: Edit button gated on canWrite + text/markdown + !isBinary
    - Editor.test.tsx: source-inspection unit tests (16 assertions)
    - vendor_drift_test.go: regex fix for bare codemirror package format
  affects:
    - frontend/package.json
    - frontend/pnpm-lock.yaml
    - frontend/src/lib/languageFor.ts
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesCapability.ts
    - frontend/src/components/Editor.tsx
    - frontend/src/components/FileBrowser/PreviewPane.tsx
    - frontend/src/components/__tests__/Editor.test.tsx
    - internal/webserver/vendor_drift_test.go
tech_stack:
  added:
    - codemirror@6.0.2
    - "@codemirror/state@6.6.0"
    - "@codemirror/view@6.43.1"
    - "@codemirror/commands@6.10.3"
    - "@codemirror/language@6.12.3"
    - "@codemirror/language-data@6.5.2"
    - "@codemirror/lang-go@6.0.1"
    - "@codemirror/lang-python@6.2.1"
    - "@codemirror/lang-javascript@6.2.5"
    - "@codemirror/lang-json@6.0.2"
    - "@codemirror/lang-yaml@6.1.3"
    - "@codemirror/lang-markdown@6.5.0"
    - "@codemirror/lang-html@6.4.11"
    - "@codemirror/lang-css@6.3.1"
    - "@codemirror/lang-rust@6.0.2"
    - "@codemirror/lang-cpp@6.0.3"
    - "@codemirror/legacy-modes@6.5.3"
  patterns:
    - CM6 imperative mount via useEffect (no @uiw/react-codemirror wrapper)
    - Compartment.reconfigure() for read-only/editable toggle without remount
    - Hand-rolled TokyoNight EditorView.theme (zero new hexes)
    - languageFor: language-data registry + legacy-modes StreamLanguage for shell
    - canWrite: desktop derives from filesWriteSignal; web-share probes write route
    - ETag echo pattern: readFileText returns etag for If-Match on next write
key_files:
  created:
    - frontend/src/lib/languageFor.ts
    - frontend/src/components/Editor.tsx
    - frontend/src/components/__tests__/Editor.test.tsx
  modified:
    - frontend/package.json
    - frontend/pnpm-lock.yaml
    - frontend/src/lib/filesApi.ts
    - frontend/src/lib/useFilesCapability.ts
    - frontend/src/components/FileBrowser/PreviewPane.tsx
    - internal/webserver/vendor_drift_test.go
decisions:
  - "@codemirror/theme-one-dark NOT installed — hand-rolled EditorView.theme with TokyoNight hexes from style.css per UI-SPEC zero-new-hexes mandate and colorblind contract"
  - "Worktree merged main before beginning — worktree fork was at v3.4.2 patch, missing 125-01 backend/vendor-drift changes; fast-forward merge resolved"
  - "vendor_drift_test.go pnpmCMKeyRe regex fixed: pnpm v9 omits quotes around bare package names (codemirror@N.N.N) but quotes scoped names (@codemirror/state@N.N.N); regex updated to handle both forms"
  - "useFilesCapability gains optional filesWriteSignal param: desktop passes SessionInfo.filesWrite; web-share passes undefined triggering write-route HEAD probe"
  - "filesApi.ts gains probeWrite(sid, path) method for canWrite web-share probe via HEAD request to write route"
metrics:
  duration: "~7 minutes"
  completed: "2026-06-14T19:35:23Z"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 9
  commits: 4
---

# Phase 125 Plan 02: CodeMirror 6 Install + Editor Core Mount + canWrite Summary

**One-liner:** CM6 17-package install with vendor-drift gate, languageFor extension mapper (Bash via legacy-modes), ETag-aware readFileText, canWrite per-surface resolution, imperative Editor.tsx with Compartment toggle and TokyoNight theme, and PreviewPane Edit button gated on canWrite/!binary.

## Tasks Completed

| # | Name | Commit | Status |
|---|------|--------|--------|
| 1 | Install CM6 + languageFor + extend readFileText/useFilesCapability | 93e2e8e | DONE |
| RED | Editor source-inspection tests | 1e215e7 | DONE |
| 2 | Editor.tsx CM6 mount + Compartment toggle + PreviewPane Edit button | 430f1ec | DONE |

## What Was Built

### Task 1: Install + languageFor + canWrite (93e2e8e)

**`frontend/package.json` + `pnpm-lock.yaml`** — 17 `@codemirror/*` packages installed at the research-verified exact pinned versions. `@codemirror/theme-one-dark` was intentionally NOT installed (UI-SPEC mandates hand-rolled TokyoNight theme, zero new hexes).

**`frontend/src/lib/languageFor.ts`** (NEW) — `async languageFor(filename): Promise<Extension>` using `@codemirror/language-data` registry `.find()` + `.load()` for common languages. Bash/sh/zsh use `StreamLanguage.define(shell)` from `@codemirror/legacy-modes/mode/shell` (`@codemirror/lang-shell` does not exist on npm — RESEARCH Open Q2 resolution). Unknown extension returns `[]` (plain text).

**`frontend/src/lib/filesApi.ts`** — `readFileText` extended to also return `etag: string | undefined` from the `ETag` response header (emitted by 125-01 `handler.go`; client echoes as `If-Match` on next write, EDIT-05). Added `probeWrite(sid, path): Promise<void>` HEAD method for canWrite web-share probe.

**`frontend/src/lib/useFilesCapability.ts`** — Extended return type now includes `canWrite: boolean`. New optional `filesWriteSignal?: boolean | null` param controls resolution strategy:
- Desktop (`filesWriteSignal = true/false`): derived directly from `SessionInfo.filesWrite` (daemon socket is auth-less — probe always succeeds, RESEARCH Pitfall 2).
- Web-share (`filesWriteSignal = undefined`): probes `HEAD /api/files/write`; maps 403-with-"files.write" body → `canWrite=false`.
- `filesWriteSignal = null`: `canWrite=false` (capability not relevant).

**`internal/webserver/vendor_drift_test.go`** — Bug fix: `pnpmCMKeyRe` regex updated to handle both quoted scoped names (`'@codemirror/state@6.6.0':`) and unquoted bare names (`codemirror@6.0.2:`). pnpm v9 omits quotes for bare (non-scoped) package names. The test passes with all 17 packages resolved in the lockfile.

### Task 2 (TDD): Editor.tsx + PreviewPane Edit button (1e215e7 RED, 430f1ec GREEN)

**`frontend/src/components/__tests__/Editor.test.tsx`** (RED, NEW) — 16 source-inspection assertions:
- Compartment + reconfigure usage (EDIT-02)
- languageFor invocation (EDIT-04)
- 500KB/5MB threshold constants (EDIT-11)
- Verbatim copy strings: "Syntax highlighting disabled for large files." and "Edits may be slow."
- `dangerouslySetInnerHTML` absent (T-125-04 XSS gate)
- `theme-one-dark` absent in non-comment lines
- `useEffect` + `EditorView` + `.destroy()` (imperative mount lifecycle)
- PreviewPane: `canWrite`, `isBinary`, `PencilSquareIcon` present

**`frontend/src/components/Editor.tsx`** (GREEN, NEW) — CM6 editor component:
- Mounts `EditorView` imperatively in `useEffect`; cleanup calls `.destroy()`
- `editable` + `language` Compartments; starts read-only, flips to editable on mount
- Hand-rolled `EditorView.theme()` using TokyoNight hexes: `#1a1b26`/`#16161e`/`#1e2030`/`#292e42`/`#c0caf5`/`#9aa5ce`/`#7aa2f7` — all from existing `style.css`
- `languageFor(filename)` applied lazily to `language` Compartment
- `LARGE_FILE_WARN_THRESHOLD = 500 * 1024` → `LargeFileNotice` warn-then-proceed banner
- `PLAIN_TEXT_THRESHOLD = 5 * 1024 * 1024` → plain-text mode + "Syntax highlighting disabled for large files." caption
- `Cmd/Ctrl+S` keymap wired to `onSave` prop (no-op default; save implementation in Plan 03)
- `data-testid="file-browser-preview"` preserved so Playwright targets work unchanged
- Zero `dangerouslySetInnerHTML` (T-125-04 XSS gate)

**`frontend/src/components/FileBrowser/PreviewPane.tsx`** (MODIFIED) — Added three optional props (`canWrite`, `isBinary`, `onEdit`) and the Pencil Edit button:
- Gated: `canWrite && !isBinary && (state.kind === 'text' || state.kind === 'markdown') && onEdit !== undefined`
- `PencilSquareIcon` at 14px from `@heroicons/react/24/outline`
- `aria-label="Edit {filename}"`; `title="Edit"`
- Absence is the colorblind-safe signal for binary/no-permission (no greyed ghost)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] Worktree behind main (missing 125-01 changes)**
- **Found during:** Start of execution
- **Issue:** Worktree was forked from `d725107` (v3.4.2 patch release), missing all 125-01 commits (If-Match/412, ETag, vendor-drift test, WRITE_CAP fixture) that were merged into `main` as `4938930`.
- **Fix:** `git merge main --no-edit` (fast-forward) to bring in 125-01 changes before starting 125-02 work.
- **Files modified:** 101 files (via fast-forward merge of 125-01 + all phase 123/124 changes)

**2. [Rule 1 - Bug] vendor_drift_test.go pnpmCMKeyRe regex didn't match bare package format**
- **Found during:** Task 1 verification (`TestCodeMirrorVersionsMatchPnpmLock` failed)
- **Issue:** The regex `^  '(@codemirror/[\w-]+|codemirror)@([0-9][^']+)':` required single quotes around package names, but pnpm v9 only quotes scoped packages (`@codemirror/state@6.6.0`) — bare packages are unquoted (`codemirror@6.0.2:`).
- **Fix:** Updated regex to `^  '?(@codemirror/[\w-]+|codemirror)@([0-9][^':]+)'?:` (optional quotes, character class `[^':]` avoids matching across colon).
- **Commit:** included in 93e2e8e
- **Files modified:** `internal/webserver/vendor_drift_test.go`

## Known Stubs

- `Editor.tsx` `onSave` prop: wired to keymap (Cmd/Ctrl+S calls `onSave`) but the actual HTTP PUT + If-Match header + 412 handling lands in Plan 03. Callers that don't pass `onSave` get a no-op. This is intentional and documented in the component.
- `Editor.tsx` `onDirty` prop: `EditorView.updateListener` calls `onDirty` when doc changes vs savedSnapshot, but dirty-state UI (dirty marker `●`, save indicator) lands in Plan 03.
- `useFilesCapability` canWrite on web-share: `probeWrite` uses HEAD which may return 405 (Method Not Allowed) from the write route if it doesn't support HEAD. The hook handles this as `canWrite=true` (server is the real authority). If needed, Plan 03 can refine to use OPTIONS or a body-less PUT with a dummy path.

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes introduced. The `probeWrite` method issues a HEAD request to the existing `/api/files/write` route (already guarded by `requireFilesWrite`). No new surface.

## Self-Check

| Claim | Status |
|-------|--------|
| `frontend/src/lib/languageFor.ts` exists | FOUND |
| `frontend/src/components/Editor.tsx` exists | FOUND |
| `frontend/src/components/__tests__/Editor.test.tsx` exists | FOUND |
| Commit 93e2e8e exists | FOUND |
| Commit 1e215e7 exists | FOUND |
| Commit 430f1ec exists | FOUND |
| pnpm test: 1152 tests pass | PASS |
| `go test -run TestCodeMirrorVersionsMatchPnpmLock`: PASS | PASS |
| grep dangerouslySetInnerHTML in Editor.tsx == 0 | PASS |
| grep theme-one-dark (non-comment) in Editor.tsx == 0 | PASS |
| grep legacy-modes in languageFor.ts >= 1 | PASS (3) |
| grep files.write in useFilesCapability.ts >= 1 | PASS (5) |
| grep etag in filesApi.ts >= 1 | PASS (3) |
| tsc --noEmit: clean | PASS |

## Self-Check: PASSED
