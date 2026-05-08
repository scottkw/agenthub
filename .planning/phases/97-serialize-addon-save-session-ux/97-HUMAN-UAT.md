---
phase: 97
type: human-uat
created: 2026-05-07
requirements: [SER-01, SER-02, SER-03]
plans: [97-01, 97-02, 97-03, 97-04, 97-05, 97-06]
status: partial
---

# Phase 97 Human UAT — Serialize Addon + Save-Session UX

> Run these AFTER Plans 97-01 through 97-06 are complete and the standard
> test gates are green (`go test ./... -count=1`, `cd frontend && pnpm test --run`,
> `cd frontend && pnpm tsc --noEmit`).

## Setup

```bash
# Build the desktop app with vendored web assets.
wails build -tags wailsassets

# Open the built app.
open build/bin/AgentHub.app   # macOS
# or run the binary directly on Linux/Windows
```

Confirm the app launches without errors and a default session is available.

---

## Scenario 1 — SER-OFF: Save menu item shows banner when Serialize is disabled

**Why manual:** Live affordance behavior across Settings toggle + right-click context menu + banner stack rendering — automated tests cover the source-scan invariants but cannot exercise the cross-component event chain at runtime.

**Setup:** Open AgentHub. Open or create a terminal session. Open Settings → Plugins. Toggle "Save terminal as text" (Serialize) **OFF**. Click Save in Settings. Wait for "Saved" indicator.

**Verify:**
1. Right-click the terminal tab.
2. Confirm the "Save Terminal As…" menu item is **visible** (NOT hidden — discoverability per RESEARCH locked decision).
3. Click "Save Terminal As…".
4. **Expected:** A banner/toast appears with text similar to "Enable the Serialize plugin in Settings to save sessions." NO native Save dialog opens.
5. Dismiss the banner. Confirm no file was written to disk.

**Sign-off:** [ ] Verified by ____________________ on ________

---

## Scenario 2 — SER-ON: Native Save dialog opens, file is plain text, content matches scrollback

**Why manual:** Native OS Save dialog cannot be exercised in headless CI; visual confirmation of dialog UX, file write integrity, and ANSI-strip correctness require human inspection.

**Setup:** In Settings → Plugins, toggle Serialize **ON** (this is the default; if it was on before Scenario 1, simply re-enable). Click Save. Open or create a terminal session and run a command that produces colored output (e.g. `ls --color=always` or `echo -e "\033[1;31mError\033[0m: test"`).

**Verify:**
1. Right-click the tab.
2. Click "Save Terminal As…".
3. **Expected:** A native Save dialog opens. The dialog title contains "(file will include any printed secrets)" or similar warning text.
4. The default filename is the tab name + timestamp + `.txt` extension (e.g. `claude-code-2026-05-07-143022.txt`).
5. Save to `~/Desktop/agenthub-uat-test.txt`. Confirm the file appears on the Desktop.
6. Open the file in TextEdit (macOS) / Notepad / `cat`:
   - **Expected:** The file is plain UTF-8 text. NO `\x1b[` escape sequences anywhere. The colored "Error: test" output appears as plain "Error: test" without color codes.
   - **Expected:** The file contains the visible scrollback content from the session.

**Sign-off:** [ ] Verified by ____________________ on ________

---

## Scenario 3 — CANCEL: User-cancellation writes no file and shows no error

**Why manual:** Real native dialog cancellation behavior (Esc key, Cancel button) is OS-specific and cannot be exercised in headless CI.

**Setup:** Continue from Scenario 2's setup (Serialize ON, session with content).

**Verify:**
1. Right-click the tab → "Save Terminal As…".
2. When the native Save dialog opens, press **Esc** (or click Cancel).
3. **Expected:** Dialog closes. NO error toast appears. NO file is written at the default path.
4. Repeat with the Cancel button: right-click → Save → click Cancel button. Same expected outcome.

**Sign-off:** [ ] Verified by ____________________ on ________

---

## Scenario 4 — SER-02 CAPTION: Verbatim secrets warning visible in Settings

**Why manual:** Visual / text-rendered verification of warning copy — automated source-scan asserts the literal string is in the source, but only a human eye confirms the caption renders as italic text directly under the toggle (not buried elsewhere).

**Setup:** Open Settings → Plugins.

**Verify:**
1. Locate the row labeled "Save terminal as text" with the Serialize toggle.
2. **Expected:** Directly under the toggle, in italic styling, the caption reads VERBATIM:
   > Saved files include any secrets, tokens, or sensitive data printed in the session.
3. Confirm:
   - The text is italic (matches the unicode11 / image "Applies to new sessions you create" caption styling).
   - The text is positioned as a description directly under the toggle, NOT as a tooltip or hover-only.
   - The text matches the verbatim string above (no paraphrasing, no added/removed words).

**Sign-off:** [ ] Verified by ____________________ on ________

---

## Final Sign-Off

- [ ] All 4 scenarios pass on macOS
- [ ] (Optional) Verified on Linux: ___________
- [ ] (Optional) Verified on Windows: ___________
- [ ] No CSP violations observed in browser devtools (web parity check — open the web-served terminal page in Chrome, inspect Console for `Refused to load` or CSP errors; expect ZERO since addon-serialize is pure JS with no WebAssembly/Worker/blob constructs per RESEARCH §"Mandatory Pre-Phase CSP Audit")
- [ ] No regression on Phase 96 image scenarios (sanity check)

**Tester:** ____________________
**Date:** ____________________
**Build:** AgentHub `___________` (paste version from About dialog or build output)

**Notes / issues observed:**

```
(free-form notes)
```

---

## Web Parity Scope Note

Phase 97 ships "Save Terminal As…" as a **desktop-only** feature (right-click context menu in the Wails GUI shell). The SerializeAddon is loaded in the web client (`web/assets/terminal.js`) for vendoring-discipline parity — ensuring `vendor_drift_test.go` stays green and every `@xterm/addon-*` in pnpm-lock has a corresponding `term.loadAddon()` call — but there is **no web-side Save UI in v3.2**. A future `SER-FUT-WEB` plan may add a `<a download>` blob URL approach or keyboard shortcut for web users. This is a locked architectural decision per 97-RESEARCH §"Web Parity Scope".
