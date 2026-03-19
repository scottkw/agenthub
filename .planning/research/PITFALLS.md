# Pitfalls Research

**Domain:** Wails v2 / React / xterm.js desktop app — v1.1 feature additions
**Researched:** 2026-03-19
**Confidence:** HIGH (all critical pitfalls verified against official issues, docs, and codebase inspection)

---

## Critical Pitfalls

Mistakes that force rewrites, break existing functionality, or ship broken features silently.

---

### Pitfall 1: xterm.js Font Size Change Without Subsequent fit() Leaves Terminal Garbled

**What goes wrong:**
Setting `terminal.options.fontSize = newSize` resizes each character cell, which changes the number of columns and rows that fit in the container. If `fitAddon.fit()` is NOT called immediately after, the terminal continues to think it has the old dimensions. The PTY receives incorrect `cols`/`rows` via the resize message, causing AI CLI output to wrap at the wrong column width — the terminal renders correctly visually but the underlying data model is wrong. Issue #4886 on the xterm.js GitHub documents "abnormal display" after fontSize change.

**Why it happens:**
`terminal.options.fontSize` is a pure render-side option — it changes cell height/width but does NOT trigger the FitAddon's ResizeObserver. The FitAddon only fires on container *pixel* dimension changes, not on logical character dimension changes. A 14px→18px font change with the same container pixel size still requires a re-fit because cols/rows must shrink to fit.

**How to avoid:**
After every font size change: `terminal.options.fontSize = newSize; requestAnimationFrame(() => fitAddonRef.current?.fit())`. The `requestAnimationFrame` defers until after the browser has applied the new glyph metrics. Without the defer, `fit()` reads stale cell dimensions and computes wrong cols/rows. The existing codebase already uses this pattern for other triggers (line 58-60, line 100-102 in TerminalPanel.tsx) — apply it consistently here too.

**Warning signs:**
- Terminal text wraps mid-word at incorrect column position after font size change
- `onResize` fires with the correct new pixel dimensions but wrong cols/rows
- AI CLI output re-wraps when pressing Enter after a font change (shell redraws at the old width)

**Phase to address:**
Font size shortcut implementation phase — verify fit() is called in the same handler that sets fontSize, not as a separate follow-up.

---

### Pitfall 2: SHIFT+= and SHIFT+- Key Events Are Consumed by xterm.js Before the App Can Handle Them

**What goes wrong:**
`SHIFT+=` produces `+` in most keyboard layouts. Inside an xterm.js terminal, typing `+` or `-` is valid terminal input and xterm.js passes it directly to the PTY. If the font size shortcut handler uses `onKey` (which fires after xterm.js processes the key), the character has already been sent to the shell. The user sees their font size change AND a `+` character inserted into the shell prompt simultaneously.

**Why it happens:**
xterm.js has two key event layers: `attachCustomKeyEventHandler` (runs first, can return `false` to cancel PTY delivery) and `onKey` (runs after, PTY already received the input). Font size shortcuts must use `attachCustomKeyEventHandler`, not `onKey` or `onData`, and must return `false` to suppress PTY delivery. Note: `attachCustomKeyEventHandler` only registers ONE handler at a time — registering it again overwrites the previous one, which is a pitfall if other handlers (copy/paste shortcuts) are already registered.

**How to avoid:**
Use a single `attachCustomKeyEventHandler` registration that handles all custom shortcuts in one function. Check `ev.type === 'keydown'` (not keyup/keypress), `ev.shiftKey === true`, and `ev.key === '=' || ev.key === '+'` for increase, `ev.key === '-' || ev.key === '_'` for decrease. Return `false` from the handler to prevent PTY delivery. Register this handler inside the session-creation `useEffect` (alongside `term.onData` and `term.onResize`), so it is cleaned up on session close.

**Warning signs:**
- Font size changes but `+` or `-` characters appear in the terminal prompt
- Only the last-registered custom key handler fires (previous handlers silently replaced)
- Key events fire twice on some browsers (keydown + keypress)

**Phase to address:**
Font size shortcut implementation phase — verify with manual input testing that no characters leak to the PTY.

---

### Pitfall 3: Wails OpenDirectoryDialog Panics on Windows If DefaultDirectory Does Not Exist

**What goes wrong:**
`runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{DefaultDirectory: lastPath})` panics on Windows 10 when `lastPath` points to a directory that has since been deleted or is otherwise invalid. This was reported as Issue #1052 and tracked in Issue #1381 in the Wails repo. The fix (validate with `fs.DirExists()` before passing) exists in recent Wails v2 builds, but the application still needs to guard on the Go side.

**Why it happens:**
The underlying Windows file dialog API (`SHBrowseForFolder` / `IFileOpenDialog`) does not gracefully handle invalid `DefaultDirectory` values — it passes the invalid path directly to Windows API without sanitization, causing an unrecoverable OS-level error that propagates as a Go panic.

**How to avoid:**
Before calling `OpenDirectoryDialog`, validate the `DefaultDirectory` path on the Go backend:
```go
func (a *App) BrowseFolder(defaultDir string) (string, error) {
    if defaultDir != "" {
        if _, err := os.Stat(defaultDir); err != nil {
            defaultDir = "" // fall back to OS default
        }
    }
    return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        DefaultDirectory: defaultDir,
        Title: "Choose project folder",
    })
}
```
Also guard the persisted "last folder" value on load — if the file exists but the stored path no longer exists on disk, clear it before passing to the dialog.

**Warning signs:**
- App crashes on Windows when opening the new-session modal for the first time after a folder has been moved or deleted
- No crash on macOS (different underlying dialog implementation is more tolerant)

**Phase to address:**
New-session modal / folder browser implementation phase — handle the "last folder" loading path defensively from day one.

---

### Pitfall 4: Terminal Not Filling Available Height — The Flex `min-height: 0` Trap

**What goes wrong:**
When a new per-tab status bar is added above the `TerminalPanel`, the terminal container stops filling the remaining space. The flex child with `flex: 1` stops expanding because its parent doesn't have `min-height: 0`. This is a well-known CSS flexbox bug: flex items have `min-height: auto` by default, which prevents them from shrinking below their content size even when `flex: 1` is set. The xterm.js FitAddon then measures a container with a non-zero height remainder and computes incorrect terminal dimensions.

**Why it happens:**
The existing `.terminal-wrapper` already has `display: flex; flex-direction: column; width: 100%; height: 100%`. Adding a new `<div className="status-bar">` with `flex-shrink: 0` inside it creates a new flex child above `TerminalPanel`. The `TerminalPanel`'s outer div has `flex: 1` but without `min-height: 0` on either the wrapper or the panel div, the browser calculates the available height incorrectly. This is confirmed by the existing `min-height: 0` in `.terminal-container` in style.css — the same fix is needed at every flex level in the chain.

**How to avoid:**
Every flex container in the vertical chain that uses `flex: 1` must also have `min-height: 0`. Check all ancestors of the `TerminalPanel` container:
- `.app` (already: `display: flex; flex-direction: column; height: 100%`)
- `.terminal-container` (already: `flex: 1; overflow: hidden` — add `min-height: 0` if not present)
- `.terminal-wrapper` (add `min-height: 0`)
- The `TerminalPanel` outer div (`flex: 1; min-height: 0`)

Also: the per-tab status bar must use `flex-shrink: 0` (not `flex: 1`) so only the terminal grows to fill remaining space.

**Warning signs:**
- Terminal only fills part of the window vertically, with a gray or empty strip at the bottom
- FitAddon reports fewer rows than expected (e.g., 20 rows when 40 would fit)
- Issue appears or disappears based on whether the status bar has rendered content

**Phase to address:**
Per-tab status bar implementation phase — test terminal fill explicitly after adding the status bar, before considering the feature done.

---

### Pitfall 5: Tab Rename Does Not Propagate to Web Dashboard Session Names

**What goes wrong:**
`RenameSession` in app.go updates `a.tabNames[id]` in memory and the React state in the desktop frontend. But the web dashboard (served to external browsers) fetches session names from `ListSessions()`, which reads `a.tabNames`. The problem: external browser clients have no mechanism to know when a rename happened — they don't receive Wails events (`EventsEmit` only reaches the embedded WebView, not WebSocket clients). Web dashboard session names go stale immediately after any rename.

**Why it happens:**
There are two independent display surfaces (desktop WebView, external browsers) sharing the same session state but with different update mechanisms. Wails `EventsEmit` only reaches the embedded frontend via the IPC bridge — it does not reach WebSocket-connected external clients. The web dashboard currently has no rename notification channel.

**How to avoid:**
One of two approaches:
1. **Polling (simpler):** The web dashboard polls `GET /api/sessions` on a short interval (e.g., every 3-5 seconds). The Go handler for this endpoint reads `a.tabNames` and returns current names. No new infrastructure needed, slightly stale data acceptable.
2. **SSE or WebSocket push (real-time):** Add a Server-Sent Events endpoint to the web server. On `RenameSession`, publish a `session_renamed` event. Web dashboard subscribes and updates the name. More complex but correct latency.

For v1.1, polling is sufficient. The key is to NOT consider rename "done" until the web dashboard reflects the change.

**Warning signs:**
- Renaming a tab in the desktop app does not update the session name shown on the web dashboard until page reload
- Web dashboard shows original CLI name (e.g., "claude") instead of user-assigned name (e.g., "auth-refactor") after rename

**Phase to address:**
Tab renaming phase — explicitly verify web dashboard reflects rename before marking the task complete.

---

### Pitfall 6: macOS Signing: `--deep` Flag Breaks Code Signature Integrity

**What goes wrong:**
Using `codesign --deep` to sign the Wails `.app` bundle appears to sign all nested binaries recursively, but it does NOT correctly sign them in the required bottom-up order. Go-built binaries embedded in the app bundle are signed incorrectly with `--deep`, and Apple's notarization service rejects the submission with "The signature of the binary is invalid." The `--deep` flag is documented as unreliable for production signing in Apple's own documentation and in community post-mortems.

**Why it happens:**
Correct macOS code signing requires signing all nested components first (bottom-up), then signing the outer bundle last. `--deep` attempts this automatically but fails for Go-compiled Mach-O binaries that have specific entitlement requirements or when the binary is in a non-standard location within the bundle. The Wails `.app` has the main binary at `Contents/MacOS/agenthub` — it must be signed before `Contents` is signed, before the `.app` is signed.

**How to avoid:**
Sign explicitly, bottom-up, without `--deep`:
```bash
codesign --force --options runtime \
  --entitlements build/entitlements.plist \
  --sign "Developer ID Application: <name> (<team-id>)" \
  --timestamp \
  "agenthub.app/Contents/MacOS/agenthub"

codesign --force --options runtime \
  --entitlements build/entitlements.plist \
  --sign "Developer ID Application: <name> (<team-id>)" \
  --timestamp \
  "agenthub.app"
```
Use `notarytool` (not the deprecated `altool`) to submit. Create the notarization zip with `ditto -c -k --keepParent agenthub.app agenthub.zip` (NOT `zip -r` — zip does not preserve extended attributes needed for code signing integrity).

**Warning signs:**
- `codesign --verify --deep --strict agenthub.app` exits non-zero after signing with `--deep`
- `xcrun notarytool submit` returns status "Invalid" even though `codesign` exits 0
- `spctl --assess --type execute agenthub.app` outputs "rejected"

**Phase to address:**
Build script implementation phase — test the full sign + notarize + staple pipeline end-to-end before declaring the build script complete.

---

### Pitfall 7: `notarytool` Exits 0 Even on Failure — CI Silently Ships Unsigned Binaries

**What goes wrong:**
`xcrun notarytool submit app.zip --wait` exits with status 0 even when notarization fails (e.g., "Invalid" or "Rejected" status). A CI pipeline that checks only `$?` will proceed to staple and release a binary that Gatekeeper will block. The failure is only visible by parsing stdout for the status string.

**Why it happens:**
`notarytool` considers "submission received and processed" to be a success (exit 0), regardless of whether Apple approved or rejected the submission. This is by design but is a pitfall for naive CI scripts.

**How to avoid:**
Parse notarytool output explicitly:
```bash
RESULT=$(xcrun notarytool submit agenthub.zip \
  --apple-id "$APPLE_ID" \
  --team-id "$APPLE_TEAM_ID" \
  --password "$APPLE_APP_SPECIFIC_PASSWORD" \
  --wait --output-format json)

STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" != "Accepted" ]; then
  echo "Notarization failed: $STATUS"
  echo "$RESULT" | jq .
  exit 1
fi
xcrun stapler staple agenthub.app
```
Store Apple credentials as GitHub Actions secrets. Use an app-specific password (not the account password). CI must run on a macOS runner — notarization requires network access to Apple's servers and a macOS-signed submission binary.

**Warning signs:**
- CI build passes but macOS users report "cannot be opened because Apple cannot check it for malicious software"
- notarytool output contains "status: Invalid" but CI reported success

**Phase to address:**
Build script implementation phase — the CI notarization check must be explicit from day one.

---

### Pitfall 8: UI Refactor Breaks Wails Binding Imports — TypeScript Types Silently Stale

**What goes wrong:**
Wails generates TypeScript bindings in `frontend/src/wailsjs/go/main/App.ts` by introspecting the exported Go methods on `App`. When new Go methods are added (e.g., `BrowseFolder`, `GetLastFolder`) or signatures change, running `wails build` or `wails dev` regenerates these bindings. If the frontend is modified without regenerating (e.g., editing `.tsx` files while `wails dev` is NOT running), the TypeScript types become stale. The code compiles and type-checks against the old bindings, but at runtime calls to new methods silently return undefined or throw "method not found" errors.

**Why it happens:**
The `wailsjs/` directory is auto-generated and gitignored (or included but only updated on `wails build`/`wails dev`). Frontend developers refactoring components may not realize that new Wails-bound methods require a rebuild to update the bindings file. TypeScript compiles against the local `.ts` file, which is stale.

**How to avoid:**
- Always run `wails dev` (not `pnpm dev`) when working on features that touch Go-backend interfaces
- After adding any new Go method to `App`, verify the binding is regenerated in `wailsjs/go/main/App.ts` before writing frontend code that calls it
- Add a note in the build script that regenerating bindings is a prerequisite when Go method signatures change

**Warning signs:**
- TypeScript compiles cleanly but runtime throws "method not bound" or function returns undefined
- `frontend/src/wailsjs/go/main/App.ts` does not list the new method you just added in Go
- Feature works in Wails dev mode but not in production build (or vice versa)

**Phase to address:**
Every phase that adds new Go methods — regenerate bindings as part of the definition of "done" for backend work.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Store "last folder" in React state only (not persisted) | No persistence code needed | User loses last folder on restart | Never — persistence was an explicit requirement |
| Per-terminal fontSize stored in xterm options only | Simple state | Lost on terminal re-mount / session restore | Only if per-session font size is not a v1.1 requirement |
| Web dashboard rename propagation via full-page reload only | No SSE/polling needed | Confusing UX for remote users watching a renamed session | Acceptable for v1.1 if polling is added later; not acceptable if web dashboard is primary UI |
| Build script that only signs on CI (not locally) | Simpler local dev | Cannot test notarization locally, CI failures are slow to iterate | Acceptable temporarily; add local signing path before release |
| Hardcoding `build/entitlements.plist` path in build script | Works today | Breaks if build structure changes | Acceptable; document the assumption clearly |

---

## Integration Gotchas

Common mistakes when connecting the new features to existing systems.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| xterm.js fontSize + FitAddon | Set fontSize, forget to call fit() | Always `requestAnimationFrame(() => fitAddon.fit())` after setting fontSize |
| xterm.js key handler + PTY input | Use onKey (after PTY delivery) for font shortcuts | Use `attachCustomKeyEventHandler` (before PTY delivery), return false |
| Wails OpenDirectoryDialog + last path persistence | Pass stale/deleted path as DefaultDirectory | Validate path existence before passing; fall back to "" |
| macOS codesign + notarytool | Sign with --deep; use altool; check only exit code | Sign bottom-up without --deep; use notarytool; parse output for "Accepted" |
| Tab rename + web dashboard | Only update React state | Also verify web dashboard reflects rename via ListSessions |
| Per-tab status bar + terminal fill | Status bar added but no min-height: 0 on flex ancestors | Check entire flex chain for min-height: 0 at every column flex level |
| Wails binding regeneration | Edit frontend without running wails dev | Always run wails dev when Go method signatures change |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Font size change triggers rapid SHIFT+hold | Continuous fit() calls, WebSocket resize storm | Debounce font size changes (50-100ms) before calling fit() and sending PTY resize | When user holds SHIFT+= |
| Web dashboard polling for renames at 1-second interval | Unnecessary load on Go server with many open sessions | Poll at 3-5 second interval; only poll when dashboard tab is visible | >10 sessions with aggressive polling |
| ResizeObserver firing on every status bar content change | Spurious terminal re-fits | Status bar content changes must not change the container pixel height | Status bar with variable-height content (errors, long URLs) |

---

## Security Mistakes

Domain-specific security issues for the new features.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing Apple signing certificate as plaintext in build script or repo | Certificate theft, unauthorized app distribution | Store as base64-encoded GitHub Actions secret; decode to temp file; delete after signing |
| Using account password instead of app-specific password for notarytool | Full Apple account compromise if CI secret leaks | Always use app-specific passwords from appleid.apple.com |
| Allowing BrowseFolder to return paths outside the user's home directory | Path traversal if the returned path is used to construct shell commands | Validate returned folder path before using it in session creation; use it only as working directory, not in shell command strings |

---

## UX Pitfalls

Common user experience mistakes in these specific features.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Font size shortcut changes font but doesn't update other tabs | Confusing — different tabs have different font sizes unexpectedly | Decide up front: per-tab or global font size. If per-tab, make that obvious; if global, store in app-level state and apply to all terminals |
| New-session modal requires clicking into folder browser every time | Friction for power users creating multiple sessions in same folder | Default to "last used folder"; only show browser if user clicks the folder field |
| Folder browser shows hidden files by default | Confusing on macOS where most project dirs are visible | Default ShowHiddenFiles: false; consider adding a toggle |
| Status bar added per tab but shows nothing when web server is off | Wasted vertical space; terminal area shrinks for no reason | Conditionally render status bar (only when web server is running, matching existing pattern in web-serving-bar) |
| Tab rename input doesn't trim whitespace | User creates tab named " " or "  " which looks empty | Already handled in TabBar.tsx (trimmed.length > 0 check) — maintain this in any rename refactor |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Font size shortcut:** Key handler prevents PTY delivery AND calls fit() AND sends PTY resize — verify all three, not just the visual font change
- [ ] **Folder browser:** Persists last path across app restarts (not just within a session) — verify with app restart
- [ ] **Tab rename:** Web dashboard session list reflects new name — verify in browser, not just in desktop UI
- [ ] **macOS build script:** Signed AND notarized AND stapled AND verified with `spctl --assess` — not just codesigned
- [ ] **Terminal fill fix:** Status bar added but terminal still fills remaining space — verify with both short and tall status bar content
- [ ] **Per-tab status bar:** Renders correctly on all three platforms — macOS, Linux, Windows (flexbox rendering differs slightly in WebKit vs WebView2)
- [ ] **Build script:** Works locally (not just CI) — test `./build.sh darwin` on a dev machine with signing certs available

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Garbled terminal after font size change | LOW | Add `requestAnimationFrame(() => fitAddon.fit())` after fontSize set; regression test |
| Key handler leaks + to PTY | LOW | Replace onKey with attachCustomKeyEventHandler, return false; test input |
| OpenDirectoryDialog panic on Windows | LOW | Add os.Stat validation before passing DefaultDirectory; deploy patch build |
| Terminal not filling height | LOW-MEDIUM | Audit flex chain for missing min-height: 0; CSS-only fix |
| Rename not on web dashboard | MEDIUM | Add polling to web dashboard or SSE endpoint; frontend-only if polling chosen |
| Notarization accepted by CI but rejected by Gatekeeper | MEDIUM | Re-sign from scratch bottom-up; renotarize; update CI to parse notarytool output |
| Wails bindings stale | LOW | Run wails dev to regenerate; delete wailsjs/ and rebuild |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Font size without fit() | Font size shortcuts phase | Hold SHIFT+= for 2 seconds; verify terminal wraps correctly |
| Key handler leaks to PTY | Font size shortcuts phase | Type + and - normally in terminal after implementing shortcuts; verify no regression |
| OpenDirectoryDialog panic | New-session modal phase | Test on Windows with a deleted "last folder" path |
| Terminal fill with status bar | Per-tab status bar phase | Add status bar; verify terminal occupies all remaining space |
| Tab rename not on web dashboard | Tab renaming phase | Rename a tab; reload web dashboard; verify name matches |
| codesign --deep | Build script phase | Run `codesign --verify --deep --strict` after signing |
| notarytool exit-0 on failure | Build script phase | Parse notarytool JSON output; fail CI if status != Accepted |
| Stale Wails bindings | Any phase adding Go methods | Check wailsjs/go/main/App.ts for new method before writing frontend code |

---

## Sources

- [xterm.js Issue #4886 — Set fontSize causes abnormal display](https://github.com/xtermjs/xterm.js/issues/4886) — confirmed: fontSize change requires fit() call
- [xterm.js ITerminalOptions — fontSize](https://xtermjs.org/docs/api/terminal/interfaces/iterminaloptions/) — authoritative option documentation
- [xterm.js Terminal.attachCustomKeyEventHandler](https://xtermjs.org/docs/api/terminal/classes/terminal/) — key handler API
- [xterm.js FitAddon resize issues #4841](https://github.com/xtermjs/xterm.js/issues/4841) — FitAddon resizes incorrectly when called with stale dimensions
- [Wails OpenDirectoryDialog panic Issue #1052](https://github.com/wailsapp/wails/issues/1052) — DefaultDirectory panic on Windows
- [Wails OpenDirectoryDialog invalid path Issue #1381](https://github.com/wailsapp/wails/issues/1381) — fix: validate DefaultDirectory before passing
- [Wails Dialog reference](https://wails.io/docs/reference/runtime/dialog/) — OpenDirectoryDialog API
- [Apple notarization with notarytool — Automatic CI setup (federicoterzi.com)](https://federicoterzi.com/blog/automatic-code-signing-and-notarization-for-macos-apps-using-github-actions/) — notarytool pipeline, exit-0 trap documented
- [macOS code signing gist (rsms)](https://gist.github.com/rsms/929c9c2fec231f0cf843a1a746a416f5) — bottom-up signing, --deep pitfall
- [ddev signing_tools](https://github.com/ddev/signing_tools) — reference implementation for CI signing
- [Wails cross-platform build guide](https://wails.io/docs/guides/crossplatform-build/) — macOS must build on macOS runner (CGO)
- [Wails code signing guide](https://wails.io/docs/guides/signing/) — official Wails signing documentation
- [xterm.js CSS flex min-height:0 fix — Issue #3346](https://github.com/xtermjs/xterm.js/issues/3346) — FitAddon height zero when flex container lacks min-height: 0

---
*Pitfalls research for: Wails/React/xterm.js v1.1 — build script, UI refactor, folder browser, font resize, terminal fill, tab rename*
*Researched: 2026-03-19*
