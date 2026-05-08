---
phase: 97-serialize-addon-save-session-ux
reviewed: 2026-05-07T00:00:00Z
depth: standard
files_reviewed: 23
files_reviewed_list:
  - app.go
  - app_save_terminal_test.go
  - frontend/src/App.tsx
  - frontend/src/__tests__/App.saver.test.tsx
  - frontend/src/components/FindBar/__tests__/FindBar.animation.test.tsx
  - frontend/src/components/PluginsSection.tsx
  - frontend/src/components/TabBar.tsx
  - frontend/src/components/TerminalPanel.tsx
  - frontend/src/components/__tests__/PluginsSection.test.tsx
  - frontend/src/components/__tests__/TabBar.test.tsx
  - frontend/src/components/__tests__/TerminalPanel.test.tsx
  - frontend/src/lib/__tests__/sanitizeFilename.test.ts
  - frontend/src/lib/__tests__/stripAnsi.test.ts
  - frontend/src/lib/sanitizeFilename.ts
  - frontend/src/lib/stripAnsi.ts
  - frontend/src/wailsjs/go/main/App.d.ts
  - frontend/src/wailsjs/go/main/App.js
  - internal/daemon/plugin_settings_test.go
  - internal/release/no_autosave_test.go
  - internal/webserver/vendor_drift_test.go
  - web/assets/terminal.js
  - web/embed.go
  - web/terminal.html
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 97: Code Review Report

**Reviewed:** 2026-05-07
**Depth:** standard
**Files Reviewed:** 23
**Status:** issues_found

## Summary

Phase 97 delivers a "Save Terminal As…" right-click action that uses `SerializeAddon.serialize()` to capture scrollback, strips ANSI sequences, sanitizes the suggested filename, and writes the result to a user-chosen path via the native Wails `SaveFileDialog`. The overall architecture is solid — the function-injection pattern for `saveFileDialogFunc`, the hot-swap useEffect placement for `SerializeAddon`, and the Pitfall #6 cleanup in the mount-effect teardown are all correctly implemented.

One critical defect exists: the Go `SaveTerminalSession` method writes the user-provided `content` string directly to a path returned by the OS dialog without any validation that the path lies within the directory the user selected. On macOS, `runtime.SaveFileDialog` normally enforces sandbox boundaries, but when the mock function is injected (or if a future Wails version changes the contract) there is no server-side guard against writing to an arbitrary absolute path — including paths outside the home directory. More concretely, the method accepts `path` from the dialog and calls `os.WriteFile(path, ...)` with no path normalization, extension enforcement, or allowlist check. The Wails dialog is the only safety net.

Three warnings round out the findings: a stale-registry leak for the `serializerRegistry` state entry when a tab is closed without the serialize plugin ever having been attached; an unhandled case in `handleRequestSave` that silently swallows the banner for non-terminal tab types other than the one guarded; and the `saveBanner` state never self-clearing — it persists until the user manually dismisses it with no timeout, diverging from the `localBanner` pattern it claims to mirror.

---

## Critical Issues

### CR-01: `SaveTerminalSession` writes to arbitrary path with no server-side validation

**File:** `app.go:867`
**Issue:** `os.WriteFile(path, []byte(content), 0o644)` is called with `path` returned directly from `saveFileDialogFunc` (the Wails `runtime.SaveFileDialog` indirection). No server-side check validates that `path` is within the directory offered to the dialog, has an expected extension, or is free of traversal components. The Wails dialog is the sole enforcement mechanism.

This is acceptable in production (Wails sandboxes the dialog return value on macOS/Windows), but the same `SaveTerminalSession` RPC is callable from any context that can reach the Wails bridge — and there is no scope guard. More critically, the test harness in `app_save_terminal_test.go` demonstrates the issue directly: the mock can return `invalidPath` (a path whose parent directory does not exist) and the code attempts the write. Any attacker or bug that controls the dialog return value (e.g., a future Wails CVE, a compromised front-end script, or a path containing a `\x00` byte on platforms that strip it) could write content to an unexpected path.

Minimum fix: after receiving `path` from the dialog, resolve it to a clean absolute path and verify it shares a prefix with `defaultDir` (or the OS home directory when `defaultDir` is empty). Additionally, reject paths whose `filepath.Ext` is not `.txt` (the only filter presented to the user). Example:

```go
path, err = a.saveFileDialogFunc(...)
if err != nil { ... }
if path == "" { return nil }

// Defense-in-depth: ensure the resolved path stays under the expected root.
cleanPath := filepath.Clean(path)
allowedRoot := defaultDir
if allowedRoot == "" {
    if home, err := os.UserHomeDir(); err == nil {
        allowedRoot = home
    }
}
if allowedRoot != "" && !strings.HasPrefix(cleanPath, allowedRoot+string(filepath.Separator)) {
    return fmt.Errorf("SaveTerminalSession: path %q is outside expected directory", cleanPath)
}
if ext := strings.ToLower(filepath.Ext(cleanPath)); ext != ".txt" && ext != "" {
    return fmt.Errorf("SaveTerminalSession: unexpected extension %q", ext)
}
if err := os.WriteFile(cleanPath, []byte(content), 0o644); err != nil { ... }
```

Note: the empty-extension branch (`ext != ""`) allows the dialog to return a path with no extension, which is legitimate on macOS when the user deletes the suggested suffix. The primary defense here is the prefix check, not the extension check.

---

## Warnings

### WR-01: `serializerRegistry` entry never cleaned up when a tab is closed before Serialize is ever enabled

**File:** `frontend/src/App.tsx:104-106`, `frontend/src/App.tsx:577-611`
**Issue:** `handleCloseTab` deletes entries from `sessionStatuses`, `fontSizes`, and `sessionExits`, but it does not clean up `serializerRegistry`. When a session tab is created and immediately closed — or when the Serialize plugin was never enabled for that session — the registry entry for that `sessionId` is never inserted (so there is nothing to clean). However, if a user enables Serialize, loads a terminal, then the hot-swap useEffect runs `onRegisterSaver(sessionId, closure)`, and then the user disables Serialize (flushing the entry to `null`) before closing the tab, the key `sessionId -> null` persists in `serializerRegistry` state indefinitely.

The mount-effect cleanup calls `onRegisterSaver?.(sessionId, null)` on unmount (TerminalPanel.tsx line 321), which does insert `null` into the registry. `handleCloseTab` does not remove this key. Over a long session with many opened and closed tabs, the registry grows with `null`-valued entries for every session that ever had Serialize attached.

Recommended fix — add a cleanup step in `handleCloseTab`:

```ts
// Clean up serializer registry for the closed session.
setSerializerRegistry((prev) => {
  const n = { ...prev }
  delete n[id]
  return n
})
```

This mirrors the `sessionStatuses` and `fontSizes` cleanup already present on lines 602-604.

### WR-02: `saveBanner` has no auto-dismiss timeout — diverges from stated "localBanner one-shot pattern"

**File:** `frontend/src/App.tsx:111`, `frontend/src/App.tsx:186-208`
**Issue:** The comment on line 866-867 of the JSX states the `saveBanner` "mirrors the localBanner one-shot pattern." The `LocalNetworkBanner` has a dismiss animation, but the `saveBanner` is a plain static `<div>` with only a manual close `×` button — no auto-dismiss. For "info" kind banners (Serialize disabled), a stuck banner that requires manual dismissal is a usability regression. More importantly, for "error" kind banners, if the user triggers multiple failed saves, the banner text is overwritten but the dismiss state is never reset, so a previous-error dismiss button click would now clear what looks like the current error.

The `saveBanner` also lacks an `id` or `aria-live` region separate from the role="status" it carries, making screen-reader behavior unpredictable.

Recommended fix: add a `setTimeout(() => setSaveBanner(null), 6000)` for the `info` kind, matching the 8-second auto-dismiss used by the WebGL recovery banner (`terminal.js:223`). For the `error` kind, keep manual dismiss only (errors should not auto-vanish before the user reads them).

### WR-03: `stripAnsi` regex does not cover OSC (Operating System Command) sequences — potential residue in output

**File:** `frontend/src/lib/stripAnsi.ts:21`
**Issue:** The regex `\x1b\[\??[0-9;]*[a-zA-Z]` covers CSI sequences (those starting with `\x1b[`) and DEC private modes (those with `?`). It does not cover:

- **OSC sequences**: `\x1b]...\x07` or `\x1b]...\x1b\\` — emitted by hyperlink addons, iTerm2 badges, window-title changes, and the progress addon (`OSC 9;4`). If the `WebLinksAddon` or a CLI emits OSC 8 hyperlinks into the buffer, `serialize({ excludeModes: true })` will include them and `stripAnsi` will not remove them.
- **SS2/SS3**: `\x1bN`/`\x1bO` (rare in practice but reachable in some terminals).
- **C1 controls** (8-bit: `\x9b`, `\x9d` etc.) — not emitted by xterm.js itself but potentially present in the raw stream.

For the Phase 97 stated scope (SerializeAddon with `{ excludeModes: true }` which suppresses mode strings), OSC sequences are the practical gap. The `WebLinksAddon` in the same terminal renders hyperlinks via OSC 8 in the stream; these may survive into the serialized output and then persist in the saved file as `\x1b]8;;https://example.com\x07text\x1b]8;;\x07`-style sequences.

The research document notes the regex was "audit-verified" against the SerializeAddon emit table — but that audit predates the `WebLinksAddon` co-existence introduced in Phase 95. A user with both `webLinks` and `serialize` enabled will get OSC 8 residue in their saved `.txt` file.

Recommended fix — extend the regex to also strip OSC sequences:

```ts
const ANSI_ESCAPE_RE = /\x1b(?:\[\??[0-9;]*[a-zA-Z]|\][^\x07\x1b]*(?:\x07|\x1b\\))/g
```

---

## Info

### IN-01: `handleRequestSave` has an invisible success path — no user feedback on successful save

**File:** `frontend/src/App.tsx:186-208`
**Issue:** When `SaveTerminalSession` succeeds (the `await` resolves without throwing), `handleRequestSave` silently returns. There is no success banner or toast. The error and "Serialize disabled" cases both set `saveBanner`, but success is silent. Users who trigger a save may not know it completed, especially if the OS Save dialog dismisses itself quickly. This is a discoverability gap rather than a correctness bug, but it diverges from the stated "info kind for Serialize disabled affordance, error kind for write/dialog failures" design (App.tsx lines 109-111) — a "success" kind was apparently omitted.

**Fix:** Add `setSaveBanner({ kind: 'info', text: 'Terminal saved.' })` after the `await SaveTerminalSession(...)` call, or rely on the OS's native "saved" affordance and add a code comment explaining the intentional silence.

### IN-02: `wailsjs/go/main/App.d.ts` declares `GetSessionQRCode` but `app.go` has no such method

**File:** `frontend/src/wailsjs/go/main/App.d.ts:46`
**Issue:** Line 46 declares `export function GetSessionQRCode(sessionID: string): Promise<string>` and line 28 of `App.js` forwards it to `Call('main.App.GetSessionQRCode', ...)`. No such method exists in `app.go`. This is a stale stub from a prior phase that was never removed. Calling this from the frontend will result in a Wails runtime error at call time (not a type error at compile time), since the TypeScript declaration satisfies the type checker.

**Fix:** Remove the `GetSessionQRCode` declaration from `App.d.ts` and the corresponding export from `App.js`. (Info severity because this is a generated-file concern and the method does not appear to be called from any reviewed source file.)

### IN-03: `no_autosave_test.go` `TestSER03_NoAutoSavePatterns` skips `.claire` and `.claude` worktree directories but not `bin/`

**File:** `internal/release/no_autosave_test.go:47-59`
**Issue:** The `skipDirs` map excludes `.git`, `node_modules`, `frontend/node_modules`, `build`, `dist`, `vendor`, `internal/release`, `.planning`, `frontend/src/wailsjs`, `screenshots`, `.claude`, and `.claire`. The repo root also contains a `bin/` directory (visible in `git status` at the top of this session). The test does not skip `bin/`. If `bin/` contains compiled Go binaries or scripts that happen to match a forbidden pattern (e.g., a shell wrapper with `auto-save` in a comment), the test would produce a false positive. Low risk in practice, but the skip-list is inconsistent: `build` and `dist` are skipped (compiled artifacts) but `bin` is not.

**Fix:** Add `"bin": true` to `skipDirs`.

---

_Reviewed: 2026-05-07_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
