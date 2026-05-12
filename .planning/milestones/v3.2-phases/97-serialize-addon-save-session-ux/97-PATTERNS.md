# Phase 97: Serialize Addon + Save-Session UX — Pattern Map

**Mapped:** 2026-05-07
**Files analyzed:** 22 (8 new, 14 modified)
**Analogs found:** 22 / 22 — every Phase 97 file has a near-verbatim analog from Phases 92, 93, 95, or 96.

> **Phase context (load-bearing):**
> - Phase 96 already shipped: `Serialize bool` is in `PluginSettings` at line 81 with `Serialize: true` default at line 109 of `internal/daemon/plugin_settings.go`. **Phase 97 makes NO changes to the daemon struct or defaults.**
> - Phase 96 already shipped: `addon-image.js` is vendored, `web/embed.go` line 10 embeds it, `web/vendor/xterm/VERSION` has 8 entries, `web/terminal.html` line 50 loads it, `vendor_drift_test.go` line 34 enforces ≥ 8.
> - Phase 96 also already shipped `(*App).SetImageConfig` Wails binding (`App.d.ts` line 143, `App.js` line 86) — these are **the immediately-prior verbatim hand-edit precedent** for Phase 97's new `SaveTerminalSession` binding.

---

## File Classification

### NEW files

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `frontend/src/lib/stripAnsi.ts` | utility | transform | `frontend/src/lib/urlSafety.ts` (Phase 95) | role-match (pure helper) |
| `frontend/src/lib/sanitizeFilename.ts` | utility | transform | `frontend/src/lib/urlSafety.ts` `isAllowedScheme` (Phase 95) | role-match (pure validator) |
| `frontend/src/lib/__tests__/stripAnsi.test.ts` | test | static | `frontend/src/lib/relayClient.test.ts` (existing) | role-match (lib unit test) |
| `frontend/src/lib/__tests__/sanitizeFilename.test.ts` | test | static | `frontend/src/lib/relayClient.test.ts` | role-match |
| `frontend/src/__tests__/App.saver.test.tsx` | test | static-source-scan | `frontend/src/__tests__/App.plugin-event.test.tsx` (Phase 92/95/96) | exact (App-level event/registry) |
| `app_save_terminal_test.go` (repo root) | test | request-response | `app_test.go` `TestUpdateCLIPath` (lines 234-259) — `testApp(t)` harness pattern | exact |
| `internal/release/no_autosave_test.go` | test | static-grep | `internal/webserver/vendor_drift_test.go` (Phase 93) | exact (filepath.Walk + regex) |
| `web/vendor/xterm/addons/addon-serialize.js` | vendored-asset | static-file | `web/vendor/xterm/addons/addon-image.js` (Phase 96) | exact (byte-identical copy) |
| `.planning/phases/97-…/97-HUMAN-UAT.md` | doc | runbook | `.planning/phases/96-image-addon-csp-audit/96-HUMAN-UAT.md` (Phase 96) | exact |

### MODIFIED files

| Modified File | Role | Data Flow | Closest Analog (self-as-analog) | Match Quality |
|---------------|------|-----------|----------------------------------|---------------|
| `frontend/package.json` | config | dependency-list | existing `@xterm/addon-image` entry (Phase 96) | exact |
| `frontend/pnpm-lock.yaml` | lockfile | dependency-list | (mechanical regen) | n/a |
| `frontend/src/components/TerminalPanel.tsx` | component | event-driven | self — Clipboard hot-swap arm (lines 367-379) and webgl/search arms in same useEffect | exact (hot-swap, NOT mount) |
| `frontend/src/components/TabBar.tsx` | component | event-driven | self — `Rename` `<button role="menuitem">` (lines 159-168) | exact |
| `frontend/src/components/PluginsSection.tsx` | component | request-response | self — `image` `renderRow` 4th-arg italic caption (lines 135-137) | exact |
| `frontend/src/App.tsx` | controller | event-driven | self — `pluginConfig` state + `EventsOn('settings:plugins')` listener (Phase 92/95/96 wiring) | role-match (registry pattern is new shape) |
| `frontend/src/components/__tests__/TabBar.test.tsx` | test | static-source-scan | self — existing source-scan tests | exact |
| `frontend/src/components/__tests__/TerminalPanel.test.tsx` | test | static-source-scan | self — existing source-scan tests (Phase 96 IMG-01/02 pattern) | exact |
| `frontend/src/components/__tests__/PluginsSection.test.tsx` | test | static-source-scan | self — existing italic-caption assertions (Phase 96) | exact |
| `frontend/src/wailsjs/go/main/App.d.ts` | type-defs | hand-edit | self — `SetImageConfig` line 143 (Phase 96) | exact |
| `frontend/src/wailsjs/go/main/App.js` | binding | hand-edit | self — `SetImageConfig` line 86 (Phase 96) | exact |
| `app.go` | controller | Wails-RPC | self — `OpenFileDialog` (lines 815-829) + `SetImageConfig` (Phase 96) | exact |
| `internal/daemon/plugin_settings_test.go` | test | static | self — `WebLinksConfig`/`ImageConfig` default-assertion blocks (Phase 95/96) | exact (lock existing default) |
| `internal/webserver/vendor_drift_test.go` | test | static | self — min-count guard line 34 | exact (constant bump 8 → 9) |
| `web/embed.go` | config | static-embed | self — Phase 96 line 10 (`addon-image.js`) | exact |
| `web/vendor/xterm/VERSION` | manifest | static | self — existing 8-entry manifest | exact (line append) |
| `web/terminal.html` | template | static-asset-list | self — `addon-image.js` `<script>` tag (line 50) | exact |
| `web/assets/terminal.js` | controller | event-driven | self — Image construction in `initTerminal()` (lines 248-257) | exact |

---

## Pattern Assignments

### `frontend/src/lib/stripAnsi.ts` (NEW — utility, transform)

**Analog:** `frontend/src/lib/urlSafety.ts` (Phase 95) — single-file pure helper module exporting named functions, no DOM/network/runtime side-effects.

**File-header pattern** (from `urlSafety.ts:1-8`):
```typescript
/**
 * urlSafety — Phase 95 (LNK-01, LNK-03) — pure security helpers for the
 * web-links addon click pipeline. NO network calls. NO logging. NO DOM.
 */
```

**Mirror exactly:**
- File-header doc-comment citing Phase 97 SER-01 + ANSI-Output-Audit (RESEARCH §"ANSI Output Audit")
- "NO network. NO logging. NO DOM. NO console output." rigor matching Phase 95 prose
- Single named export `export function stripAnsi(input: string): string`
- Implementation per RESEARCH Pattern 3: regex `/\[\??[0-9;]*[a-zA-Z]/g` replacement
- The regex covers SGR / ECH / CUF/CUB/CUU/CUD / DEC private modes (audit verified upstream of `serialize({ excludeModes: true })`)

**Adapt — escape character:** RESEARCH Pattern 3's example regex omits the leading ESC (`\x1b`). The actual ANSI sequences emit `\x1b[<…>` — verify the regex catches the ESC byte too. Recommended pattern: `/\x1b\[[?]?[0-9;]*[a-zA-Z]/g`. Plan-checker MUST verify the regex against the addon-source ANSI vocabulary captured in 97-RESEARCH §"ANSI Output Audit" (lines 139-167).

---

### `frontend/src/lib/sanitizeFilename.ts` (NEW — utility, validator)

**Analog:** `frontend/src/lib/urlSafety.ts` `isAllowedScheme` (lines 20-27) — pure-function validator with no side-effects, returning a string (or boolean) without throwing.

**Pattern (from urlSafety.ts:20-27):**
```typescript
export function isAllowedScheme(href: string): boolean {
  try {
    const u = new URL(href);
    return (ALLOWED_SCHEMES as readonly string[]).includes(u.protocol);
  } catch {
    return false;
  }
}
```

**Mirror exactly:**
- Pure named export `export function sanitizeFilename(name: string): string`
- File-header comment citing Pitfall #4 (path traversal, reserved Windows names, leading-dot hidden files)
- Implementation per RESEARCH Pattern 4 (lines 575-583):
  ```typescript
  export function sanitizeFilename(name: string): string {
    const collapsed = name.trim().replace(/\s+/g, '_')
    const sanitized = collapsed.replace(/[^\w\-.]/g, '_')
    if (sanitized === '' || sanitized.startsWith('.')) return 'session'
    if (/^(con|prn|aux|nul|com[1-9]|lpt[1-9])$/i.test(sanitized)) return 'session'
    return sanitized
  }
  ```
- "Defense-in-depth — the OS Save dialog also rejects bad names" rigor in the doc-comment

**Adapt:** Nothing. RESEARCH Pattern 4 is verbatim-ready.

---

### `frontend/src/lib/__tests__/stripAnsi.test.ts` (NEW — vitest unit tests)

**Analog:** `frontend/src/lib/relayClient.test.ts` (existing — same `lib/__tests__/` directory shape).

**Mirror exactly:**
- `import { describe, it, expect } from 'vitest'`
- `import { stripAnsi } from '../stripAnsi'`
- Coverage matrix per RESEARCH §"Wave 0 Gaps" line 1031: SGR (`\x1b[1;31m`), ECH (`\x1b[5X`), CUF/CUB/CUU/CUD (`\x1b[3C`), DEC private modes (`\x1b[?7l`, `\x1b[?25h`).
- Round-trip test: feed a fixture string captured from a real `serialize({ excludeModes: true })` invocation; assert all `\x1b[…]` sequences are removed AND printable text preserved.
- Negative test: assert `stripAnsi('plain text')` returns `'plain text'` unchanged.

---

### `frontend/src/lib/__tests__/sanitizeFilename.test.ts` (NEW — vitest unit tests)

**Analog:** `frontend/src/lib/relayClient.test.ts` (existing).

**Mirror exactly:**
- Coverage matrix per RESEARCH §"Wave 0 Gaps" line 1032:
  - Path traversal: `../../etc/passwd` → `_.._.._etc_passwd` (or similar)
  - Empty string → `'session'`
  - Leading-dot (`.bashrc`) → `'session'`
  - Reserved Windows names: `'CON'`, `'NUL'`, `'COM1'`, `'LPT1'` → `'session'`
  - Whitespace collapse: `'my session  name'` → `'my_session_name'`
  - Allowed characters preserved: `'agent-1.txt'` → `'agent-1.txt'`
- One test per row (table-driven `it.each` style preferred).

---

### `frontend/src/__tests__/App.saver.test.tsx` (NEW — App-level integration test)

**Analog:** `frontend/src/__tests__/App.plugin-event.test.tsx` (Phase 92/95/96) — App-level static-source-scan tests for cross-component wiring.

**Pattern (App.plugin-event.test.tsx:1-22):**
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../App.tsx?raw'
import terminalPanelRaw from '../components/TerminalPanel.tsx?raw'

describe('PLUG-03: App.tsx Wails event subscription', () => {
  it("registers EventsOn('settings:plugins', ...)", () => {
    expect(raw).toMatch(/EventsOn\(\s*['"]settings:plugins['"]/)
  })
  it('declares pluginConfig state', () => {
    expect(raw).toMatch(/pluginConfig/)
    expect(raw).toMatch(/setPluginConfig/)
  })
  // ...
})
```

**Mirror exactly:**
- `import raw from '../App.tsx?raw'` (and `tabBarRaw`, `terminalPanelRaw` as needed)
- `describe('Phase 97 SER-01: saver registry round-trip', () => { ... })`
- Source-scan assertions (per project convention — vitest cannot stand up Wails RPC):
  1. App.tsx declares `serializerRegistry` state (or equivalent map name)
  2. App.tsx declares `handleRegisterSaver` callback (or equivalent)
  3. App.tsx declares `handleRequestSave` callback (or equivalent)
  4. App.tsx imports `stripAnsi` from `'./lib/stripAnsi'`
  5. App.tsx imports `sanitizeFilename` from `'./lib/sanitizeFilename'`
  6. App.tsx imports `SaveTerminalSession` from the wailsjs binding
  7. TerminalPanel.tsx accepts `onRegisterSaver` optional prop
  8. TerminalPanel.tsx calls `onRegisterSaver?.(sessionId, ...)` inside the hot-swap useEffect (positive arm with closure)
  9. TerminalPanel.tsx calls `onRegisterSaver?.(sessionId, null)` in the negative arm (unregister on toggle-off — Pitfall #6 prevention)
  10. TabBar.tsx accepts `onRequestSave` optional prop
  11. TabBar.tsx menu item calls `onRequestSave?.(contextMenu.tabId)`

**Adapt:** This is the **only Phase 97 file with a structurally-new shape** (saver registry). The closest test-pattern analog is App.plugin-event.test.tsx; the closest runtime-pattern is React's "child registers callback with parent" (App.tsx already uses this for `onWebGLContextLost`).

---

### `app_save_terminal_test.go` (NEW — Go test file at repo root)

**Analog:** `app_test.go` `TestUpdateCLIPath` (lines 234-259) and `testApp(t)` harness (lines 28-67).

**`testApp(t)` harness pattern (app_test.go:28-67):**
```go
func testApp(t *testing.T) *App {
    t.Helper()
    if goruntime.GOOS == "windows" {
        t.Skip("testApp uses Unix domain sockets")
    }
    if goruntime.GOOS == "darwin" {
        t.Setenv("HOME", t.TempDir())
    } else {
        t.Setenv("XDG_CONFIG_HOME", t.TempDir())
    }
    engine := daemon.NewSessionEngine()
    api := daemon.NewAPI(engine)
    // ... in-process daemon API on tempdir socket ...
}
```

**Test pattern (app_test.go:234-259):**
```go
func TestUpdateCLIPath(t *testing.T) {
    app := testApp(t)
    if err := app.UpdateCLIPath("claude", "/bin/cat"); err != nil {
        t.Fatalf("UpdateCLIPath: %v", err)
    }
    // ... assertions ...
}
```

**Mirror exactly:**
- `package main` + same imports as `app_test.go` (uses `testApp(t)` from sibling file)
- New file in repo root next to `app.go` (NOT in `internal/`)
- Table-driven test `TestSaveTerminalSession` with subcases:
  1. **Cancel path:** mock `runtime.SaveFileDialog` returning `("", nil)` — assert no file written, no error
  2. **Normal write:** mock dialog returning a tempdir path — assert file contents match `content` argument
  3. **Write to non-existent dir:** assert wrapped error containing "SaveTerminalSession: write:"
  4. **Dialog setup error:** mock dialog returning `("", err)` — assert wrapped error containing "SaveTerminalSession: dialog:"

**Adapt — dialog mocking:** `runtime.SaveFileDialog` cannot be mocked directly (it's a free function from `wails/v2/pkg/runtime`). The pragmatic adaptation is to extract the dialog call behind a function-injection point in `app.go` (e.g., `a.saveFileDialogFunc` defaulting to the Wails runtime). This mirrors the `serviceControlFunc` and `statusFunc` patterns already established in the codebase (see PROJECT.md "Key Decisions" — "Function injection for service control" and "Function injection for health checks"). **Plan-checker MUST validate this — Plan author should propose either (a) function-injection pattern, or (b) split testable inner write logic from outer dialog call. Recommendation: (a), to match prior `*Func` precedent.**

---

### `internal/release/no_autosave_test.go` (NEW — SER-03 negative-grep regression test)

**Analog:** `internal/webserver/vendor_drift_test.go` (Phase 93/89) — static file/source content scan via `os.ReadFile` + regex; same "forever-defense at minimal cost" model.

**Pattern (vendor_drift_test.go:11-36):**
```go
package webserver

import (
    "os"
    "regexp"
    "strings"
    "testing"
)

var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`)

func TestXtermVendorVersionsMatchPnpmLock(t *testing.T) {
    lockData, err := os.ReadFile("../../frontend/pnpm-lock.yaml")
    if err != nil {
        t.Fatalf("ReadFile pnpm-lock.yaml: %v", err)
    }
    // ... scan + assertions ...
}
```

**Mirror exactly:**
- New `internal/release/` package directory (does NOT yet exist — confirmed via `ls`)
- `package release`
- `func TestSER03_NoAutoSavePatterns(t *testing.T)` walks BOTH the Go tree (`../../`) AND the frontend `src/` tree, applying a slice of forbidden regex patterns:
  - `(?i)\bauto.?save\b` (autosave/auto-save/autoSave variants)
  - `(?i)\bsave.?on.?(quit|exit|close)\b`
  - `(?i)on(quit|exit|close).*\.(write|save)`
  - Companion test `TestSER03_NoAutoSettingsField` asserts that `internal/daemon/plugin_settings.go` does NOT contain a field name matching `(?i)autosave` in `PluginSettings` (per RESEARCH Wave 0 Gap line 1035).
- `filepath.Walk` over the repo root from `../../` skipping `node_modules`, `.git`, `vendor/` (the addon UMD copies do not get scanned), and the `internal/release/` directory itself (so the regex literals in this test file don't false-positive)
- `t.Errorf` with file:line citation when a forbidden pattern matches

**Adapt:** The Phase 88 "OriginPatterns: ["*"] reintroduction" test (cited in RESEARCH §"Cross-tier note for SER-03") is the closest precedent for a `filepath.Walk`-based negative-grep across the whole tree. Locate that exact test (likely `internal/relay/origin_test.go` or similar) before implementation and mirror its skip-list verbatim.

---

### `web/vendor/xterm/addons/addon-serialize.js` (NEW — vendored UMD bundle)

**Analog:** `web/vendor/xterm/addons/addon-image.js` (Phase 96) — byte-identical copy from `frontend/node_modules/@xterm/addon-image/lib/addon-image.js`.

**Mechanical recipe (Phase 96 verbatim):**
1. `cd frontend && pnpm add @xterm/addon-serialize@^0.14.0`
2. `cp frontend/node_modules/@xterm/addon-serialize/lib/addon-serialize.js web/vendor/xterm/addons/addon-serialize.js`
3. Verify the file is the UMD build (assigns `global.SerializeAddon` in the IIFE preamble): `head -3 web/vendor/xterm/addons/addon-serialize.js` should show `(function (global, factory) { ... })` UMD wrapper
4. Verify same-origin (no remote `<script>` references — required for CSP `script-src 'self'`)
5. Verify no `URL.createObjectURL`, `new Worker(`, `blob:` script construction (Phase 96 audit precedent — RESEARCH §"Mandatory Pre-Phase CSP Audit" already confirmed addon-serialize is clean per `Findings Table` lines 109-133)

---

### `.planning/phases/97-…/97-HUMAN-UAT.md` (NEW — UAT runbook)

**Analog:** `.planning/phases/96-image-addon-csp-audit/96-HUMAN-UAT.md` (Phase 96, just landed).

**Frontmatter pattern (96-HUMAN-UAT.md:1-7):**
```yaml
---
phase: 96
type: human-uat
created: 2026-05-07
requirements: [IMG-01, IMG-02, IMG-03, IMG-04]
plans: [96-01, 96-02, 96-03, 96-04, 96-05, 96-06]
---
```

**Mirror exactly:**
- Same frontmatter shape (phase: 97, requirements list, plans list)
- "Run these AFTER Plans … are all complete and the standard test gates are green" preamble
- Per-scenario layout: **Why manual** / **Setup** / **Verify** / **Sign-off line**
- Phase 97 scenarios (per RESEARCH Wave 0 Gap line 1042):
  1. SER-OFF: open new session → toggle Serialize OFF in Settings → right-click tab → "Save Terminal As…" → verify toast appears (no native dialog)
  2. SER-ON: toggle Serialize ON → right-click → "Save…" → native Save dialog appears → save to `~/Desktop/test.txt` → verify file contents are plain text (no `\x1b[…]`) AND contain expected scrollback
  3. CANCEL: toggle ON → right-click → "Save…" → click Cancel → verify NO file written, NO error toast
  4. (Optional) Cross-platform: repeat scenario 2 on Linux + Windows (Plan-author's discretion based on Phase 99 release-gate pairing)

---

### `frontend/package.json` (MODIFY — add `@xterm/addon-serialize` dependency)

**Analog:** existing `@xterm/addon-image: "^0.9.0"` entry (Phase 96).

**Mirror exactly:** Insert `"@xterm/addon-serialize": "^0.14.0"` in alphabetical order under `dependencies`. Position: AFTER `@xterm/addon-search` and BEFORE `@xterm/addon-unicode11` (alphabetical: image < search < serialize < unicode11 < web-links < webgl).

---

### `frontend/pnpm-lock.yaml` (MODIFY — auto-regenerated)

**Mechanical:** `pnpm install` after the package.json edit. Verify resulting `'@xterm/addon-serialize@0.14.0': {}` block has zero transitive deps (RESEARCH §"Standard Stack" line 203).

---

### `frontend/src/components/TerminalPanel.tsx` (MODIFY — hot-swap arm + saver registration)

**Analog:** self, Clipboard hot-swap arm (lines 367-379) and the broader hot-swap useEffect (lines 329-495).

**Pattern (TerminalPanel.tsx:367-379):**
```typescript
// Clipboard hot-swap (CLIP-01)
if (pluginConfig?.clipboard) {
  if (!clipboardAddonRef.current) {
    const clipAddon = new ClipboardAddon()
    term.loadAddon(clipAddon)
    clipboardAddonRef.current = clipAddon
  }
} else {
  if (clipboardAddonRef.current) {
    clipboardAddonRef.current.dispose()
    clipboardAddonRef.current = null
  }
}
```

**Existing dep array (TerminalPanel.tsx:495):**
```typescript
}, [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search, pluginConfig?.webLinks, onWebGLContextLost, sessionId])
```

**Mirror exactly:**
- Import: `import { SerializeAddon } from '@xterm/addon-serialize'` alongside the existing addon imports (group with line 8 `ClipboardAddon` import)
- Add `const serializeAddonRef = useRef<SerializeAddon | null>(null)` alongside other addon refs (line 86-87 grouping)
- Add a NEW prop `onRegisterSaver?: (sessionId: string, fn: (() => string) | null) => void` to the props interface (alongside `onWebGLContextLost`)
- Add a NEW arm INSIDE the hot-swap useEffect (after the WebLinks arm at line 494, BEFORE the closing `}, […]`) per RESEARCH Pattern 1 (lines 483-499):
  ```typescript
  // Phase 97 SER-01 hot-swap arm. Mirrors clipboard/webgl shape.
  if (pluginConfig?.serialize) {
    if (!serializeAddonRef.current) {
      const serializeAddon = new SerializeAddon()
      term.loadAddon(serializeAddon)
      serializeAddonRef.current = serializeAddon
      onRegisterSaver?.(sessionId, () =>
        serializeAddon.serialize({ excludeModes: true }))
    }
  } else {
    if (serializeAddonRef.current) {
      serializeAddonRef.current.dispose()
      serializeAddonRef.current = null
      onRegisterSaver?.(sessionId, null)  // Pitfall #6 — unregister stale closure
    }
  }
  ```
- Extend the dep array at line 495: append `pluginConfig?.serialize` and `onRegisterSaver` (BUT NOT `serializeAddonRef.current` — refs are stable). The full new array: `[pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search, pluginConfig?.webLinks, pluginConfig?.serialize, onWebGLContextLost, onRegisterSaver, sessionId]`
- Cleanup in the MOUNT useEffect's RETURN (lines 265-280): dispose `serializeAddonRef.current` AND call `onRegisterSaver?.(sessionId, null)` to flush the registry on unmount (Pitfall #6 — saver registry memory leak). Mirror the `webglAddonRef` cleanup style (lines 269-271).

**Adapt — critical distinction from Phase 96:**
- Image (Phase 96) is **mount-only / next-session-only** because storage allocation has buffer-state implications.
- Serialize (Phase 97) is **hot-swap-friendly** because the addon is purely a buffer-walker; loading/unloading has no buffer-state effect (RESEARCH §"Architectural Responsibility Map" row 3).
- Therefore Serialize lives in the **hot-swap useEffect** (lines 329-495) alongside Clipboard, NOT in the **mount useEffect** alongside Image and Unicode 11.
- **Plan-author MUST NOT copy the Phase 96 mount-block placement.** Use the Clipboard hot-swap arm at lines 367-379 as the structural template.

---

### `frontend/src/components/TabBar.tsx` (MODIFY — extend context menu with Save item)

**Analog:** self, existing `Rename` menu item (lines 159-168) inside the `contextMenu` block (lines 152-170).

**Pattern (TabBar.tsx:152-170):**
```tsx
{contextMenu && tabs.some(t => t.id === contextMenu.tabId) && (
  <div
    className="tab__context-menu"
    role="menu"
    style={{ position: 'fixed', top: contextMenu.y, left: contextMenu.x }}
    onMouseDown={(e) => e.stopPropagation()}
  >
    <button
      role="menuitem"
      className="tab__context-menu__item"
      onClick={() => {
        startEditById(contextMenu.tabId)
        setContextMenu(null)
      }}
    >
      Rename
    </button>
  </div>
)}
```

**Mirror exactly:**
- Add a new prop `onRequestSave?: (tabId: string) => void` to the props interface (alongside `onRename` at line 16)
- Destructure `onRequestSave` in the component signature (line 30 area)
- Add a SECOND `<button role="menuitem">` AFTER the existing `Rename` button (line 168) but INSIDE the same `<div className="tab__context-menu">`:
  ```tsx
  <button
    role="menuitem"
    className="tab__context-menu__item"
    onClick={() => {
      onRequestSave?.(contextMenu.tabId)
      setContextMenu(null)
    }}
  >
    Save Terminal As…
  </button>
  ```
- The button label is "Save Terminal As…" with **U+2026 HORIZONTAL ELLIPSIS** (the literal three-dot character matching Phase 96's italic-caption convention and macOS/Apple HIG menu conventions). Do NOT use three ASCII dots.
- The `setContextMenu(null)` call closes the menu after invocation, matching the Rename item's behavior

**Adapt:** Nothing structural. Pure additive — no existing behavior changes.

---

### `frontend/src/components/PluginsSection.tsx` (MODIFY — add italic caption to Serialize row)

**Analog:** self, image `renderRow` (lines 135-137) — Phase 96's just-landed italic-caption.

**Current `serialize` row (PluginsSection.tsx:138-139):**
```tsx
{renderRow('serialize', 'Save terminal as text',
  'Right-click a tab to export the visible scrollback as a text file.')}
```

**Mirror exactly — extend with 4th argument** per RESEARCH §"Code Examples" lines 891-893:
```tsx
{renderRow('serialize', 'Save terminal as text',
  'Right-click a tab to export the visible scrollback as a text file.',
  'Saved files include any secrets, tokens, or sensitive data printed in the session.')}
```

The `renderRow` function signature already supports the 4th argument (lines 100-114) — this is a single-line extension to the existing call. The italic styling comes from the existing `settings-panel__description--italic` class at line 108.

**Adapt — wording:** This is the **SER-02 secrets warning**, NOT the unicode11/image "Applies to new sessions you create" affordance. The Serialize plugin IS hot-swap-friendly, so the next-session-only caption would be incorrect. Use the secrets-warning prose verbatim from RESEARCH line 893.

---

### `frontend/src/App.tsx` (MODIFY — add saver registry + handleRequestSave + Wails RPC call)

**Analog:** self — existing `pluginConfig` state + `EventsOn('settings:plugins')` listener (Phase 92/95/96 wiring) + existing `onWebGLContextLost` callback prop pattern (App.tsx line ~168 area).

**Pattern (handcrafted; mirrors React idioms already in App.tsx):**

Per RESEARCH §"Pattern 2: Saver Registry (TerminalPanel → App.tsx)" lines 510-543:
```typescript
// In App.tsx:
const [serializerRegistry, setSerializerRegistry] = useState<
  Record<string, (() => string) | null>
>({})

const handleRegisterSaver = useCallback(
  (sessionId: string, fn: (() => string) | null) => {
    setSerializerRegistry((prev) => ({ ...prev, [sessionId]: fn }))
  },
  []
)

const handleRequestSave = useCallback(async (tabId: string) => {
  const tab = tabs.find((t) => t.id === tabId)
  if (!tab) return
  const fn = serializerRegistry[tab.sessionId]
  if (!fn) {
    setBanner({ kind: 'info', text: 'Enable the Serialize plugin in Settings to save sessions' })
    return
  }
  const plainText = stripAnsi(fn())
  const stamp = new Date().toISOString().replace(/[:T]/g, '-').replace(/\..+/, '')
  const fname = sanitizeFilename(tab.name) + '-' + stamp + '.txt'
  try {
    await SaveTerminalSession('', fname, plainText)
  } catch (err) {
    setBanner({ kind: 'error', text: 'Could not save terminal: ' + String(err) })
  }
}, [tabs, serializerRegistry])
```

**Mirror exactly:**
- Imports at top of App.tsx: `import { stripAnsi } from './lib/stripAnsi'`, `import { sanitizeFilename } from './lib/sanitizeFilename'`, `import { SaveTerminalSession } from './wailsjs/go/main/App'`
- `serializerRegistry` state declared near other plugin-related state
- `handleRegisterSaver` callback memoized via `useCallback` with empty deps (the setter is stable)
- `handleRequestSave` callback memoized via `useCallback` with `[tabs, serializerRegistry]` deps
- Pass `onRegisterSaver={handleRegisterSaver}` prop to `<TerminalPanel ... />` (alongside existing `onWebGLContextLost`)
- Pass `onRequestSave={handleRequestSave}` prop to `<TabBar ... />` (alongside existing `onRename`)
- Banner pattern: use whatever existing banner/toast mechanism App.tsx already uses (Phase 95 `LinkConfirmPopover`, Phase 95 banner stack — **plan-author MUST inspect App.tsx to identify the existing banner state hook before implementation**; the pseudo-code `setBanner` above is illustrative, not literal)

**Adapt — banner mechanism:** The exact `setBanner` call shape will depend on App.tsx's existing banner state. Phase 81's `BannerStack` component is the most likely consumer (PROJECT.md "Banner vertical stacking" decision). Plan-author identifies the precise hook and adapts the pseudo-code to fit.

---

### `frontend/src/components/__tests__/TabBar.test.tsx` (MODIFY — assert Save menu item)

**Analog:** self, existing source-scan tests in this file.

**Mirror exactly:**
- Add a new `it('Phase 97 SER-01: Save Terminal As… menu item present', () => { ... })` test
- Source-scan: `expect(raw).toContain('Save Terminal As')` (matches both ASCII and U+2026 ellipsis variants)
- Source-scan: `expect(raw).toContain('onRequestSave?.(contextMenu.tabId)')` — verifies the click handler invokes the prop with the correct argument
- Source-scan: `expect(raw).toMatch(/onRequestSave\?\s*:/)` — verifies the prop is declared optional in the props interface

---

### `frontend/src/components/__tests__/TerminalPanel.test.tsx` (MODIFY — assert hot-swap arm + saver register/unregister)

**Analog:** self, Phase 96 IMG-01/02 test patterns (recently added).

**Mirror exactly:**
- New `describe('Phase 97 SER-01: SerializeAddon hot-swap arm', () => { ... })` block. Source-scan assertions:
  1. `expect(raw).toContain("import { SerializeAddon } from '@xterm/addon-serialize'")`
  2. `expect(raw).toContain('serializeAddonRef')`
  3. `expect(raw).toContain('new SerializeAddon()')`
  4. `expect(raw).toContain("serialize({ excludeModes: true })")` — Pitfall #1 regression guard (must pass `excludeModes: true`)
  5. `expect(raw).toMatch(/pluginConfig\?\.serialize/)` — verifies the toggle is read
  6. `expect(raw).toContain('onRegisterSaver')` — verifies the register prop is consumed
  7. **Hot-swap-vs-mount placement check (CRITICAL — Phase 96 contrast):** assert that `'SerializeAddon'` appears WITHIN the hot-swap useEffect range AND does NOT appear in the mount useEffect range. Practical implementation: split the file source on the `useEffect(...` markers, scan only the hot-swap section. Mirror the testing style Phase 96 used for the `ImageAddon` placement check (which scans ONLY the mount block).
  8. Unregister-on-toggle-off: `expect(raw).toContain('onRegisterSaver?.(sessionId, null)')`
  9. Dep-array assertion: `expect(raw).toMatch(/\[\s*pluginConfig\?\.webgl[\s\S]*?pluginConfig\?\.serialize/)` — verifies `serialize` is in the same dep array as `webgl`/`clipboard`

---

### `frontend/src/components/__tests__/PluginsSection.test.tsx` (MODIFY — assert SER-02 caption)

**Analog:** self, existing italic-caption assertions (Phase 96 IMG-02).

**Mirror exactly:** Add a new `it('Phase 97 SER-02: Serialize row carries secrets-warning italic caption', () => { ... })`. Source-scan assertion: `expect(raw).toContain('Saved files include any secrets, tokens, or sensitive data printed in the session.')`. For tighter regression guarding, slice the source at the `'Save terminal as text'` index and assert the caption appears within ~200 characters AFTER that point (proves it's the 4th argument to the serialize `renderRow`, not somewhere else).

---

### `frontend/src/wailsjs/go/main/App.d.ts` (MODIFY — add SaveTerminalSession signature)

**Analog:** self — `SetImageConfig` line 143 (Phase 96, just landed) and `OpenFileDialog` line 57.

**Pattern (App.d.ts:54-57):**
```typescript
export function OpenDirectoryDialog(defaultDir: string): Promise<string>

// File dialog bound method (SET-04)
export function OpenFileDialog(defaultDir: string): Promise<string>
```

**Pattern (App.d.ts:140-143):**
```typescript
// Web-links sub-key writer (Phase 95 LNK-05 / LNK-06).
export function SetWebLinksConfig(arg1: daemon.WebLinksConfig): Promise<void>

// Image-addon sub-key writer (Phase 96 IMG-02).
export function SetImageConfig(arg1: daemon.ImageConfig): Promise<void>
```

**Mirror exactly:**
- Add a new comment block `// Save terminal session (Phase 97 SER-01)`
- Add `export function SaveTerminalSession(defaultDir: string, defaultName: string, content: string): Promise<void>` immediately after the SetImageConfig line (line 143-144 area)
- Use named parameters (Wails generated style varies between named params and `arg1`/`arg2`/`arg3` — match the Wails generator output for THREE string args; OpenFileDialog uses named, SetImageConfig uses `arg1` — this is **a real divergence in the existing file**; recommend named params per OpenFileDialog precedent)

---

### `frontend/src/wailsjs/go/main/App.js` (MODIFY — add SaveTerminalSession Call() stub)

**Analog:** self — `SetImageConfig` line 86 (Phase 96) and `OpenFileDialog` line 39.

**Pattern (App.js:36-39):**
```javascript
export const OpenDirectoryDialog  = (defaultDir)            => Call('main.App.OpenDirectoryDialog', [defaultDir])

// File dialog bound method (SET-04)
export const OpenFileDialog       = (defaultDir)            => Call('main.App.OpenFileDialog', [defaultDir])
```

**Pattern (App.js:85-86):**
```javascript
// Image-addon sub-key writer (Phase 96 IMG-02).
export const SetImageConfig          = (cfg)                => Call('main.App.SetImageConfig', [cfg])
```

**Mirror exactly:**
- Add a new comment block `// Save terminal session (Phase 97 SER-01).` (matches Phase 96 caption style)
- Add `export const SaveTerminalSession    = (defaultDir, defaultName, content) => Call('main.App.SaveTerminalSession', [defaultDir, defaultName, content])` immediately after the SetImageConfig line
- Match the column-aligned arrow style used throughout this file (whitespace alignment is preserved by Phase 92's hand-edit-pin convention — RESEARCH §"State of the Art" notes this is a hand-maintained file, NOT regenerated)

---

### `app.go` (MODIFY — add `(*App).SaveTerminalSession` Wails method)

**Analog:** self — `OpenFileDialog` (lines 815-829) for the dialog-options/cancellation-empty-string pattern + `SetImageConfig` (Phase 96, lines 589-611) for the daemon-not-connected/error-wrap rigor.

**OpenFileDialog pattern (app.go:815-829) — the structural template:**
```go
// OpenFileDialog opens a native OS file picker and returns the selected path.
// Returns "" if the user cancels. Falls back to the user's home directory when
// defaultDir is empty. Used by Settings > Paths browse buttons (SET-04).
func (a *App) OpenFileDialog(defaultDir string) (string, error) {
    if defaultDir == "" {
        if home, err := os.UserHomeDir(); err == nil {
            defaultDir = home
        }
    }
    return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
        Title:            "Select Executable",
        DefaultDirectory: defaultDir,
        ShowHiddenFiles:  true,
    })
}
```

**Mirror exactly — full method per RESEARCH §"Phase 97 call shape" (lines 327-355):**
```go
// SaveTerminalSession opens a native Save File dialog and writes the supplied
// terminal scrollback content to the user-chosen path. Cancellation is silent
// success. Returns wrapped errors for dialog setup or write failures.
//
// Phase 97 SER-01. Mirrors OpenFileDialog (lines 815-829) for the cancel=""
// pattern and SetImageConfig (Phase 96) for daemon-not-connected/error-wrap
// rigor.
func (a *App) SaveTerminalSession(defaultDir, defaultName, content string) error {
    if defaultDir == "" {
        if home, err := os.UserHomeDir(); err == nil {
            defaultDir = home
        }
    }
    path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
        Title:                "Save Terminal As… (file will include any printed secrets)",
        DefaultDirectory:     defaultDir,
        DefaultFilename:      defaultName,
        CanCreateDirectories: true,
        Filters: []runtime.FileFilter{
            {DisplayName: "Text File (*.txt)", Pattern: "*.txt"},
        },
    })
    if err != nil {
        return fmt.Errorf("SaveTerminalSession: dialog: %w", err)
    }
    if path == "" {
        return nil // user cancelled — silent success (mirror OpenFileDialog cancel=="")
    }
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        return fmt.Errorf("SaveTerminalSession: write: %w", err)
    }
    return nil
}
```

**Mirror exactly:**
- Position: insert AFTER `OpenFileDialog` (line 829) and BEFORE `GetTailscaleStatus` (line 831). This keeps dialog methods grouped together.
- Comment header citing Phase 97 SER-01 + reference to OpenFileDialog as the structural sibling
- Empty-path = silent success (NOT an error) — mirror OpenFileDialog's behavior precisely
- Error wrapping with `"SaveTerminalSession: dialog:"` and `"SaveTerminalSession: write:"` prefixes for triage
- File mode `0o644` (per RESEARCH §"Security Domain" line 1074 — owner rw, others r; tighter `0o600` rejected as surprising default)

**Adapt — testability:** Per the `app_save_terminal_test.go` analog above, plan-author MUST decide whether to extract a `saveFileDialogFunc` injection point or split inner write logic. The pure form above does not yet support unit testing the cancel/write paths without mocking `runtime.SaveFileDialog`. Recommendation: introduce `a.saveFileDialogFunc` field defaulting to `runtime.SaveFileDialog`, mirror the `serviceControlFunc`/`statusFunc` precedent.

---

### `internal/daemon/plugin_settings_test.go` (MODIFY — lock Serialize default assertion)

**Analog:** self, `WebLinksConfig` and `ImageConfig` default-assertion blocks (Phase 95/96).

**Mirror exactly:** Add a single explicit assertion to `TestDefaultPluginSettings`:
```go
if !s.Serialize {
    t.Error("Serialize should default true (Phase 97 SER-01 lock-in — Phase 96 set the default; Phase 97 freezes the assertion)")
}
```

**Adapt:** This is a **PURE ASSERTION ADDITION** — no struct change is required. The `Serialize bool \`json:"serialize"\`` field already exists (line 81 of `plugin_settings.go`) with `Serialize: true` in `defaultPluginSettings()` at line 109. Phase 97 only freezes the assertion to defend against accidental regression.

---

### `internal/webserver/vendor_drift_test.go` (MODIFY — bump min-count 8 → 9)

**Analog:** self, line 34 (Phase 95: 7→8 verbatim precedent in Phase 96 PATTERNS.md).

**Current pattern (line 33-36):**
```go
if len(pnpmVersions) < 8 {
    t.Fatalf("failed to parse at least 8 @xterm/* packages (xterm, addon-fit, addon-webgl, addon-unicode11, addon-clipboard, addon-search, addon-web-links, addon-image) from pnpm-lock.yaml: found %v (Phase 95 SRC-95-06 — addon-web-links joined the manifest; Phase 96 IMG-03 — addon-image joined the manifest; T-96-06-01 mitigation)", pnpmVersions)
}
```

**Mirror exactly:**
- Change `< 8` → `< 9`
- Change `at least 8` → `at least 9`
- Append `addon-serialize` to the inline package list inside the parenthetical
- Update the citation suffix: append `; Phase 97 SER-03 — addon-serialize joined the manifest; T-97-XX-XX mitigation` to the existing chain

**Adapt:** Nothing structural. Mechanical bump.

---

### `web/embed.go` (MODIFY — add addon-serialize to `//go:embed`)

**Analog:** self, line 10 (Phase 96 — `addon-image.js` was added there as a continuation line).

**Current pattern (web/embed.go:5-11):**
```go
//go:embed dashboard.html terminal.html join.html
//go:embed assets/terminal.js assets/terminal.css
//go:embed assets/dashboard.js assets/dashboard.css
//go:embed assets/join.js assets/join.css
//go:embed vendor/xterm/xterm.js vendor/xterm/xterm.css vendor/xterm/addon-fit.js vendor/xterm/VERSION
//go:embed vendor/xterm/addons/addon-webgl.js vendor/xterm/addons/addon-unicode11.js vendor/xterm/addons/addon-clipboard.js vendor/xterm/addons/addon-search.js vendor/xterm/addons/addon-web-links.js
//go:embed vendor/xterm/addons/addon-image.js
var WebFS embed.FS
```

**Mirror exactly:** Append `vendor/xterm/addons/addon-serialize.js` to the existing line 11 (the same line as `addon-image.js`):
```go
//go:embed vendor/xterm/addons/addon-image.js vendor/xterm/addons/addon-serialize.js
```
OR add a new continuation line if line-length exceeds 120 cols. Both forms are accepted by the Go `//go:embed` directive parser.

---

### `web/vendor/xterm/VERSION` (MODIFY — append version line)

**Analog:** self — existing 8-entry manifest.

**Current contents (verified via cat):**
```
@xterm/xterm@6.0.0
@xterm/addon-fit@0.11.0
@xterm/addon-webgl@0.19.0
@xterm/addon-unicode11@0.9.0
@xterm/addon-clipboard@0.2.0
@xterm/addon-search@0.16.0
@xterm/addon-web-links@0.12.0
@xterm/addon-image@0.9.0
```

**Mirror exactly:** Append `@xterm/addon-serialize@0.14.0` as the 9th line, preserving the trailing newline. The drift test parses each non-comment, non-empty line as `@<scope>/<pkg>@<version>` (vendor_drift_test.go lines 50-55).

---

### `web/terminal.html` (MODIFY — add `<script>` tag for addon-serialize)

**Analog:** self, lines 43-50 (Phase 96 — `addon-image.js` script tag at line 50).

**Current pattern (web/terminal.html:43-50):**
```html
<script src="/assets/xterm/xterm.js"></script>
<script src="/assets/xterm/addon-fit.js"></script>
<script src="/assets/xterm/addons/addon-webgl.js"></script>
<script src="/assets/xterm/addons/addon-unicode11.js"></script>
<script src="/assets/xterm/addons/addon-clipboard.js"></script>
<script src="/assets/xterm/addons/addon-search.js"></script>
<script src="/assets/xterm/addons/addon-web-links.js"></script>
<script src="/assets/xterm/addons/addon-image.js"></script>
```

**Mirror exactly:** Insert `<script src="/assets/xterm/addons/addon-serialize.js"></script>` AFTER the addon-image line (line 50) and BEFORE `<script src="/assets/terminal.js"></script>` (line 64). Order matters: terminal.js depends on UMD globals being defined.

---

### `web/assets/terminal.js` (MODIFY — construct SerializeAddon in `initTerminal()`)

**Analog:** self, ImageAddon construction block (lines 248-257) — Phase 96, just landed.

**Current pattern (web/assets/terminal.js:248-257):**
```javascript
if (pluginConfig.image) {
  try {
    var storageLimit = (pluginConfig.imageConfig && pluginConfig.imageConfig.storageLimit) || 16;
    var imageAddon = new ImageAddon.ImageAddon({
      storageLimit: storageLimit,
      enableSizeReports: false
    });
    term.loadAddon(imageAddon);
  } catch (e) { /* addon UMD may not be present, or WASM bootstrap failed — silent */ }
}
```

**Mirror exactly:**
- Add `serialize: true` to the `pluginConfig` defaults at line 118-130 (alphabetical position alongside `image`/`progress` — already present at line 122 per RESEARCH; verify and add if missing)
- Add a new construction block IMMEDIATELY AFTER the ImageAddon block (after line 257), structurally parallel:
  ```javascript
  // Phase 97 SER-01: SerializeAddon construction — hot-swap-friendly on
  // desktop (TerminalPanel.tsx) but the web client has no save UI in v3.2,
  // so we just register the addon for parity. Future v3.3 work may add a
  // web-side save button consuming this addon's serialize() method.
  if (pluginConfig.serialize) {
    try {
      var serializeAddon = new SerializeAddon.SerializeAddon();
      term.loadAddon(serializeAddon);
    } catch (e) { /* addon UMD may not be present — silent */ }
  }
  ```
- UMD global namespace pitfall #11 (Pitfall #4 in RESEARCH): `new SerializeAddon.SerializeAddon()` (NOT `new SerializeAddon()`) — verify the actual UMD shape during implementation by inspecting the vendored file's IIFE preamble (mirror Phase 96 verification approach)
- `try { ... } catch (e) { /* … silent */ }` wrapping is mandatory (matches addon-image / addon-clipboard / addon-search precedent in this file)

**Adapt — web has no save UI in v3.2:** Per RESEARCH §"Architectural Responsibility Map" line 180: "Vendored addon serving (...) WEB-01 vendoring discipline; **no runtime web consumer in v3.2**". The web-side construction is **vendoring-discipline parity only** — the addon is loaded but no UI invokes it. This is acceptable per Phase 93's vendor-drift-test contract (every `@xterm/addon-*` in pnpm-lock must have a `<script>` tag and a `term.loadAddon()` call in terminal.js, even if no UI consumes it yet).

---

## Shared Patterns

### Hot-swap addon arm contract (vs. mount-only)

**Source:** `frontend/src/components/TerminalPanel.tsx` Clipboard arm (lines 367-379) and the dep array at line 495.

**Apply to:** SerializeAddon arm

**Why exactly:**
- Hot-swap arms live INSIDE the second useEffect (lines 329-495). Toggling the plugin in Settings causes an immediate `dispose()` / `loadAddon()` cycle on the active terminal.
- Mount-only arms (Image, Unicode 11) live INSIDE the FIRST useEffect (lines 158-280). Toggling the plugin requires a new session because the addon's load has buffer-state implications (storage allocation, width-table swap).
- **Serialize is hot-swap-friendly** (RESEARCH §"Architectural Responsibility Map" row 3) because the addon is purely a buffer-walker. Place it alongside Clipboard/WebGL/Search/WebLinks, NOT alongside Image/Unicode11.
- Plan-checker MUST verify the arm is in the correct useEffect — the source-scan test (TerminalPanel.test.tsx assertion #7 above) is the regression guard.

---

### Saver-registry callback-prop pattern (TerminalPanel → App.tsx)

**Source:** Existing `onWebGLContextLost` callback prop at App.tsx (line ~168 area) and TerminalPanel.tsx prop interface (line 60 area).

**Apply to:** New `onRegisterSaver` prop on TerminalPanel + new `serializerRegistry` state on App.tsx + new `onRequestSave` prop on TabBar.

**Why exactly:** TabBar is structurally above TerminalPanel in the React tree; TerminalPanel can't directly handle right-click events on tabs. The closure-registry pattern lifts the addon's `serialize()` method into App.tsx via callback-prop registration, where TabBar's right-click handler can reach it. This is React-idiomatic; the antipattern alternative (`useImperativeHandle` + ref-forwarding) would require cross-tree imperative refs.

---

### Wails dialog cancellation = silent success

**Source:** `app.go` `OpenFileDialog` (lines 815-829) and `OpenDirectoryDialog` (lines 803-813).

**Apply to:** `SaveTerminalSession`

**Why exactly:** The Wails `runtime.SaveFileDialog` returns `("", nil)` on user cancellation — empty string + NIL error. Treating the empty path as an error would surface a spurious "save failed" toast on every cancel click, which is hostile UX. Mirror the existing OpenFileDialog/OpenDirectoryDialog cancellation path: empty string returns immediately as nil error. Only non-empty paths trigger `os.WriteFile`. RESEARCH §"Wails SaveFileDialog API Contract" lines 317-323 documents this contract explicitly.

---

### Vendoring + drift-test five-file lockstep (Phase 93/94/95/96 verbatim)

**Sources:** `web/embed.go`, `web/vendor/xterm/VERSION`, `internal/webserver/vendor_drift_test.go`, `web/terminal.html`, `web/assets/terminal.js`

**Apply to:** Any new `@xterm/addon-*` vendored package — Phase 97 is the FIFTH application of this pattern (after Phase 89 xterm-core, Phase 93 webgl/unicode11/clipboard, Phase 94 search, Phase 95 web-links, Phase 96 image).

**Pattern:** Every addon-vendor introduction touches FIVE files in lockstep:
1. `web/vendor/xterm/addons/addon-X.js` — the file itself (copied from `frontend/node_modules`)
2. `web/embed.go` — `//go:embed` directive includes the new file
3. `web/vendor/xterm/VERSION` — `@xterm/addon-X@<version>` line appended
4. `web/terminal.html` — `<script src="/assets/xterm/addons/addon-X.js"></script>` tag added before terminal.js
5. `internal/webserver/vendor_drift_test.go` — bump min-count guard
6. `web/assets/terminal.js` — `term.loadAddon(new XAddon.XAddon())` construction in initTerminal()

**Skipping any one of these breaks vendor_drift_test, breaks `script-src 'self'` CSP, or breaks the UMD load in the browser.**

---

### Source-scan test convention

**Source:** `frontend/src/components/__tests__/PluginsSection.test.tsx` lines 4-35; `frontend/src/components/__tests__/TerminalPanel.test.tsx` lines 11-22; `frontend/src/__tests__/App.plugin-event.test.tsx` lines 1-50.

**Apply to:** All new vitest assertions for TerminalPanel + TabBar + PluginsSection + App in Phase 97.

**Why exactly:** Tests in this codebase favor `expect(raw).toContain(...)` source-scan assertions over React Testing Library renders, because the components depend on browser APIs (Terminal, WebGL, Wails runtime) that vitest cannot stand up cleanly. New SER-01/02 tests must follow this convention.

---

### Wails-method daemon-not-connected guard pattern

**Source:** `app.go` `SetWebLinksConfig` and `SetImageConfig` — `if a.client == nil { return fmt.Errorf("daemon not connected") }`.

**Apply to:** N/A — `SaveTerminalSession` does NOT touch the daemon. It is a pure file-I/O method on `*App`. The daemon-not-connected guard is for daemon-RPC methods only.

**Why this is called out:** Plan-author should NOT copy the daemon guard into `SaveTerminalSession` — the dialog and write paths are runtime-only operations.

---

### Function-injection for testability (project precedent)

**Source:** `serviceControlFunc` (kardianos service control), `statusFunc` (Tailscale health), per PROJECT.md "Key Decisions".

**Apply to:** `saveFileDialogFunc` (proposed) for `SaveTerminalSession`'s dialog mocking in `app_save_terminal_test.go`.

**Why exactly:** Wails runtime functions cannot be mocked in unit tests directly (they are free functions, not interfaces). The project's established pattern is to introduce a function-typed field on `*App` defaulting to the runtime function; tests inject a stub. This enables testing the cancel/write/error paths without spawning a real Wails dialog.

---

## No Analog Found

**None.** Every Phase 97 file has a clear analog already in the codebase. Phase 97 is structurally a Phase-96 mirror (vendoring + caption + binding hand-edit) plus a Phase-95-style new-pure-helper module pair (stripAnsi, sanitizeFilename) plus a Phase-93-style negative-grep regression test (no_autosave_test.go).

The only **structurally novel** shape is the **saver-registry callback chain** (TabBar → App → TerminalPanel via callback props). Even this has a conceptual analog: App.tsx's existing `onWebGLContextLost` callback prop is the same React idiom (child registers callback with parent for cross-tree event routing). The "register at addon-attach / unregister at addon-detach" closure lifecycle is new but well-precedented in React.

---

## Metadata

**Analog search scope:**
- `internal/daemon/` (plugin_settings.go, plugin_settings_test.go)
- `app.go`, `app_test.go` (root)
- `internal/webserver/` (vendor_drift_test.go)
- `internal/release/` (NEW directory — does not yet exist)
- `web/` (embed.go, terminal.html, assets/terminal.js, vendor/xterm/VERSION, vendor/xterm/addons/)
- `frontend/src/components/` (TabBar.tsx, TerminalPanel.tsx, PluginsSection.tsx, __tests__/)
- `frontend/src/lib/` (openLink.ts, urlSafety.ts, __tests__/)
- `frontend/src/wailsjs/go/main/` (App.d.ts, App.js)
- `frontend/src/__tests__/App.plugin-event.test.tsx`
- `.planning/phases/96-image-addon-csp-audit/` (96-PATTERNS.md, 96-HUMAN-UAT.md as cross-reference)

**Files scanned (non-exhaustive):** 16 source files + 6 test files + 2 cross-reference Phase 96 docs

**Pattern extraction date:** 2026-05-07

**Confidence:** HIGH — every analog was read directly with file:line citations verified by Read/Grep; every "mirror exactly" instruction maps to a verified excerpt above; every "adapt" callout names a specific divergence with rationale.

**Phase precedent richness:** Phase 97 has the **strongest precedent of any v3.2 phase** — Phases 92, 93, 95, and 96 collectively pre-establish the vendoring pipeline, the hot-swap arm shape, the italic-caption pattern, the hand-edited Wails binding stub pattern, the function-injection testability pattern, and the negative-grep regression test pattern. The only file requiring genuine design effort is `App.tsx`'s saver registry — every other file is a near-verbatim copy of an existing analog.
