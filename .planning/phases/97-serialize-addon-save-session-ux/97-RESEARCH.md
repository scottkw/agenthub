# Phase 97: Serialize Addon + Save-Session UX — Research

**Researched:** 2026-05-07
**Domain:** xterm.js `@xterm/addon-serialize`, Wails v2 `runtime.SaveFileDialog`, tab right-click "Save Terminal As…" UX, terminal scrollback export to `.txt` file with secrets-warning copy and zero-auto-save guarantee
**Confidence:** HIGH

---

## Summary

Phase 97 ships "Save Terminal As…" — a deliberate, user-gestured export of the visible scrollback to a plain-text `.txt` file. The mechanical work is small but the architecture has three loud surprises:

1. **`@xterm/addon-serialize@0.14.0`'s `serialize()` returns ANSI-escape-laden text, NOT plain text.** Direct read of `package/src/SerializeAddon.ts` (the upstream source on npm) confirms the output is dense with `[...m` SGR color codes, `[...C/D/A/B` cursor moves, and `[...X` erase sequences — designed for *replaying into a terminal*, not for human reading. SER-01 mandates "text-only output." The plan MUST therefore include an ANSI-strip step between `serialize()` and the file write. This is THE load-bearing finding for Phase 97. [VERIFIED: `/tmp/serialize-inspect/package/src/SerializeAddon.ts` lines 146, 215-223, 245-282 — `_currentRow += '[...'` patterns throughout `_serializeString()`.]

2. **The "Serialize" PluginSettings boolean is acting as the addon-as-library toggle, NOT a feature gate for the save action.** ROADMAP SC-2: "Toggle defaults to ON for the addon-as-library; serialize never auto-saves or auto-runs." The toggle controls whether the SerializeAddon is *attached to the terminal* (so `.serialize()` is callable). The right-click "Save Terminal As…" menu item appears regardless of the toggle, but its CALLBACK no-ops (or shows a banner) when the addon isn't loaded. This avoids the trap of conflating "addon enabled" with "save feature enabled" — they are two different things and SER-03 forbids any auto-save path.

3. **Wails v2.12.0 has `runtime.SaveFileDialog` available — this codebase has never called it before.** `runtime.SaveDialogOptions` is a type alias to `frontend.SaveDialogOptions` exposing `Title, DefaultDirectory, DefaultFilename, Filters []FileFilter, ShowHiddenFiles, CanCreateDirectories, TreatPackagesAsDirectories`. Cancellation returns `("", nil)` (empty path, no error — same convention as the existing `OpenFileDialog` at `app.go:818-829`). The pattern to add to `app.go` is fully prescribed: a new `(*App).SaveTerminalSession(sessionID, defaultName, content string) error` method that delegates to `runtime.SaveFileDialog` then `os.WriteFile` — no native code, no platform branching.

The remaining work is mechanical: install `@xterm/addon-serialize@0.14.0`, vendor `lib/addon-serialize.js` to `web/vendor/xterm/addons/`, append to `web/vendor/xterm/VERSION`, bump `vendor_drift_test.go` min-count from 8 to 9 (Phase 96 closed at 8), construct the addon in TerminalPanel's hot-swap useEffect (NOT mount — see "Hot-swap vs Mount" decision below — toggling the Serialize boolean attaches/detaches the addon live, mirror of the WebGL/Clipboard arms), add the addon ref + `getSerializedText` callback prop pattern so App.tsx can reach the addon when the right-click handler fires, extend `TabBar` with a "Save Terminal As…" menu item that calls a new App-level handler, and add a `SaveTerminalSession` Wails RPC pair in `app.go` + `App.{d.ts,js}` hand-edits (Phase 92 STATE.md pin pattern).

Web parity in v3.2 is **scoped down to "text export via download blob"** (no native dialog on the browser side). This phase is fundamentally desktop-first because the right-click context menu lives in the GUI shell; web-served pages have a different keyboard/menu vocabulary and the natural web equivalent is `<a download>` anchor with a generated blob URL — viable but parallel work. **Recommendation: ship desktop-only in Phase 97 and let Phase 99 / SER-FUT-* track web parity if real users ask for it.** SER-01..SER-03 read as desktop-flow requirements (right-click menu, Wails dialog) and the ROADMAP's "blessed v3.2 plugin suite" framing for vendoring (WEB-01) means we still vendor the addon under `web/vendor/xterm/addons/` for completeness even though no web UI consumes it yet — the file's presence keeps `vendor_drift_test.go` green and lets Phase 99 wire the web side without re-running the vendor step.

The secrets-warning copy is settled: SER-02 mandates the verbatim/near-verbatim text *"Saved files include any secrets, tokens, or sensitive data printed in the session."* This belongs as an italic caption directly under the Serialize toggle row in `PluginsSection.tsx` — same `settings-panel__description settings-panel__description--italic` class already used by Phase 93/96 for "Applies to new sessions you create.". A second copy of the warning belongs in the SaveFileDialog's title or default-filename context (e.g., the dialog Title `"Save Terminal As… — file will include secrets"`) so the user sees it again at the moment of decision — but this is Claude's discretion (see User Constraints).

The no-auto-save invariant (SER-03) is enforced by **negative grep regression tests**: assert the codebase has zero matches for the patterns `setInterval.*serialize`, `setTimeout.*serialize.*[0-9]{3,}` (long-delay schedule), `BeforeQuit.*[Ss]eriali[zs]e`, and zero settings keys named `autoSave|autoExport|autoCapture` in `PluginSettings`. This is grep-based, fast, and source-inspection-style consistent with Phase 87's negative regression patterns and Phase 88's `OriginPatterns: ["*"]` reintroduction guard.

**Primary recommendation:** Install `@xterm/addon-serialize@^0.14.0` (run `cd frontend && pnpm add @xterm/addon-serialize@^0.14.0`), vendor `lib/addon-serialize.js` to `web/vendor/xterm/addons/addon-serialize.js`, append `@xterm/addon-serialize@0.14.0` to `web/vendor/xterm/VERSION`, bump `vendor_drift_test.go` min-count guard from 8 to 9, construct the addon in TerminalPanel's hot-swap useEffect (specific-key dep `pluginConfig?.serialize` — mirror of webgl/clipboard/search/webLinks arms), add a `serializeAddonRef` and an `onRequestSave` callback prop that App.tsx wires to TabBar's new "Save Terminal As…" context menu item, add `(*App).SaveTerminalSession` Wails RPC in `app.go` that calls `runtime.SaveFileDialog` then `os.WriteFile` (UTF-8, LF line endings, no BOM), strip ANSI before writing the file using a small pure helper in `frontend/src/lib/stripAnsi.ts` (or accept the ANSI-laden text and strip server-side — tradeoff documented), add the SER-02 italic secrets-warning caption under the Serialize toggle row in `PluginsSection.tsx`, and lock SER-03 with three negative regression tests.

**No CSP amendment needed.** The pre-phase audit of `addon-serialize` source (this RESEARCH §"Mandatory Pre-Phase CSP Audit") found 0 occurrences of `WebAssembly.*`, `new Worker(`, `URL.createObjectURL(`, `new Blob(`, `eval(`, `new Function(`, `importScripts(`, `blob:`, or `data:text/javascript`. The addon is pure JS string concatenation over xterm Buffer cells. CSP changes from Phase 96 (`'wasm-unsafe-eval'` for addon-image) carry forward unchanged.

---

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase` vars, `PascalCase` components, ESLint + Prettier, TypeScript types — applies to TerminalPanel hot-swap arm, TabBar context-menu extension, `stripAnsi.ts` helper.
- **Node:** `pnpm` (project default). Add `@xterm/addon-serialize@^0.14.0` as a regular dependency (not devDep), matching the runtime-dep pattern of all other `@xterm/addon-*` packages — including Phase 96's IMG-01 promotion.
- **Go:** `go fmt`, context-aware functions. Applies to new `(*App).SaveTerminalSession` method (it should accept `defaultDir, defaultName, content string` parameters and use `os.WriteFile` with `0o644` perms).
- **No global npm installs.**
- **NEVER kill node.exe** — Claude Code runs as Node.
- **LSP first** for code navigation — applies to discovering existing `runtime.SaveFileDialog` consumers (verified: zero existing call sites in the project; this is the first).
- **UAT via dev-browser skill** for browser-based verifications — only relevant for the optional web-parity check (Claude's discretion). Desktop UAT is native (real Wails build + macOS Save dialog) and uses the project memory `wails build -tags wailsassets` pattern.
- **Wails build requires `-tags wailsassets`** for production builds (project memory feedback).
- **Don't delete test artifacts early** — applies to the `.txt` files generated by the manual UAT script; preserve until user confirms verification complete.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SER-01 | User right-clicks a terminal tab → chooses "Save Terminal As…" → Wails `SaveFileDialog` opens → confirms a path → a `.txt` file is written containing the full visible scrollback (text-only; HTML is SER-FUT-01). | Three new pieces wire together: (1) `TabBar.tsx` adds a "Save Terminal As…" `<button role="menuitem">` next to the existing "Rename" item in the existing context menu; (2) App.tsx accepts an `onRequestSave(tabId)` callback from TabBar, looks up the corresponding TerminalPanel via a per-tab `Record<string, () => string>` callback registry, calls the registered `getSerializedText()`, strips ANSI via `frontend/src/lib/stripAnsi.ts`, and invokes the new `(*App).SaveTerminalSession` Wails RPC; (3) `(*App).SaveTerminalSession` calls `runtime.SaveFileDialog({ Title, DefaultFilename: <session-name>.txt, Filters: [{ DisplayName: "Text File", Pattern: "*.txt" }] })`, and on a non-empty path returned, writes the content via `os.WriteFile(path, []byte(content), 0o644)`. Cancellation (empty path) is a no-op success. The "full visible scrollback" is what `SerializeAddon.serialize()` returns when no `range` or `scrollback` option is passed (verified — typings/addon-serialize.d.ts §ISerializeOptions: "When not specified, all available rows in the scrollback buffer will be serialized"). |
| SER-02 | Settings tooltip on the Serialize toggle reads (verbatim or near-verbatim): "Saved files include any secrets, tokens, or sensitive data printed in the session." Toggle defaults to ON for the addon-as-library; serialize never auto-saves or auto-runs. | Italic caption pattern lives directly under the Serialize toggle row in `PluginsSection.tsx`, using the existing `settings-panel__description settings-panel__description--italic` class (Phase 93/96 precedent). The caption argument is the 4th parameter of `renderRow()`. PluginSettings.Serialize defaults `true` already (`internal/daemon/plugin_settings.go:81,109` — confirmed via grep). The "addon-as-library" framing means: toggle OFF → SerializeAddon not loaded → save button still appears in context menu but invokes nothing meaningful (recommendation: show a one-shot BannerStack toast "Enable the Serialize plugin in Settings to save sessions" when the user clicks Save with the addon disabled). |
| SER-03 | No on-disk capture of session state occurs without an explicit user action in v3.2 — a regression test (or scope-discipline review checklist item) confirms there is no timer-driven serialization, no graceful-shutdown serialization, and no settings option that enables auto-save. | Three negative grep regression tests (mirror Phase 87's negative regression patterns and Phase 88's `OriginPatterns: ["*"]` reintroduction guard): (a) `internal/release/no_autosave_test.go` (or `frontend/src/__tests__/no_autosave.test.tsx` source-inspection style) asserts zero matches for `setInterval\(.*[Ss]eriali[zs]e\(`, `setTimeout\(.*[Ss]eriali[zs]e\([^,]+,\s*[0-9]{4,}` (long-delay scheduled), `BeforeQuit\b.*[Ss]eriali[zs]e`, `OnShutdown.*[Ss]eriali[zs]e` across `frontend/src/**/*.{ts,tsx}` and `**/*.go`; (b) the daemon `PluginSettings` struct has no fields whose JSON name matches `auto[Ss]ave|auto[Ee]xport|auto[Cc]apture|saveOn[A-Z]\w+`; (c) `(*App).SaveTerminalSession` is the ONLY method whose name matches `(?i)save.*(session|terminal|scrollback)` in app.go. All three are source-inspection regex-grep tests, fast, and provide forever-defense against regression. |
</phase_requirements>

---

<user_constraints>
## User Constraints

> No `97-CONTEXT.md` will be authored. Per [skip-discuss-when-research-complete] memory: when ROADMAP/REQUIREMENTS/research already pre-answer the gray areas, skip `/gsd-discuss-phase` and proceed to `/gsd-plan-phase`. ROADMAP success criteria + REQUIREMENTS SER-01..SER-03 leave only mechanical and small-discretion questions, all answered below.

### Locked Decisions

- **Phase 97 owns Save Terminal As… for desktop ONLY in v3.2.** [ROADMAP Phase 97 SC-1: "User right-clicks a terminal tab → chooses "Save Terminal As…" → Wails `SaveFileDialog` opens"; the `SaveFileDialog` framing is desktop-specific.] The vendored UMD copy of `addon-serialize.js` lands under `web/vendor/xterm/addons/` to keep the WEB-01 vendoring discipline consistent and `vendor_drift_test.go` green; web-served pages neither construct the addon nor expose a save action in v3.2. Web parity tracked as a future `SER-FUT-WEB` item if real users request it. (See "Web parity scope" question below.)
- **Text-only output in v3.2.** [ROADMAP Phase 97 SC-1; REQUIREMENTS SER-01: "text-only output; HTML output is explicitly out of scope for v3.2 and tracked as SER-FUT-01"] The addon's `serializeAsHTML()` method is NOT called. The output of `serialize()` is run through an ANSI-strip step before write.
- **Default ON for the addon-as-library.** [ROADMAP Phase 97 SC-2 verbatim: "Toggle defaults to ON for the addon-as-library"] PluginSettings.Serialize already defaults `true` (Phase 92). No daemon change needed for defaults.
- **Verbatim-or-near-verbatim secrets warning.** [REQUIREMENTS SER-02 verbatim: "Saved files include any secrets, tokens, or sensitive data printed in the session."] Italic caption under the Serialize toggle row, exactly that string. Recommendation: also include a short "(includes secrets — review before sharing)" suffix in the SaveFileDialog Title, so the warning is visible at the moment of decision (not just buried in Settings). Final wording at planner's discretion.
- **No auto-save / no on-disk capture without explicit user action.** [REQUIREMENTS SER-03] Enforced via three negative regression tests (see SER-03 row). No `BeforeQuit`/`OnShutdown` hook calls `serialize()`. No timer schedules `serialize()`. No `autoSave*` field exists in `PluginSettings`.
- **`@xterm/addon-serialize@0.14.0` (latest stable, not 0.15.0-beta).** [Mirror of Phase 96 IMG-01 decision — stable wins for v3.2; betas in security-sensitive code paths are a non-starter.] [VERIFIED: npm registry, 2026-05-07 — `pnpm view @xterm/addon-serialize` returned `latest: 0.14.0, beta: 0.15.0-beta.216, deps: none, main: lib/addon-serialize.js, types: typings/addon-serialize.d.ts`.]
- **Vendored same-origin under `web/vendor/xterm/addons/addon-serialize.js`.** [STATE.md ROADMAP Phase 93 vendoring discipline] Phase 93/94/95/96 pattern applies verbatim — copy `frontend/node_modules/@xterm/addon-serialize/lib/addon-serialize.js` (CJS UMD, NOT the `.mjs`) to `web/vendor/xterm/addons/addon-serialize.js`. The file's presence is what `vendor_drift_test.go` (generalized in Phase 93 WEB-02) checks; the `<script>` tag in `web/terminal.html` may be added in Phase 97 (defensible as part of WEB-01 completeness) OR deferred to Phase 99 release-gate / a future SER-FUT-WEB. **Recommendation:** add the `<script>` tag in Phase 97 — keeps the v3.2 vendoring story whole and costs ~1 line — but skip the `web/assets/terminal.js` construction (no web UI consumer; the addon class becomes available on `window.SerializeAddon` but is unused).
- **Addon construction is HOT-SWAPPABLE (not next-session-only).** [Mirror of WebGL/Clipboard hot-swap arms] Toggling Serialize in Settings while a terminal is open should attach/detach the addon live — there is no buffer-state implication (unlike Unicode 11 or Image which re-flow / re-allocate). The italic caption under the Serialize toggle is the SECRETS warning, NOT a "next-session-only" caption. (See "Hot-swap vs mount" question below.)
- **No image copy / save / extract gestures bleed into Phase 97.** [REQUIREMENTS `## Out of Scope`] Phase 97 ships text-only saves of the entire scrollback. Selection-only export, image-cell extraction, regex-filtered export, etc. are all SER-FUT-* candidates.
- **Cancellation is a silent no-op.** Wails `runtime.SaveFileDialog` returns `("", nil)` on cancel (verified — `runtime/dialog.go:65-74`); the App-level handler treats empty path as success-with-no-action and emits no toast or banner.

### Claude's Discretion

- **Where ANSI stripping happens (frontend vs backend).** **Recommendation: strip in the frontend** via a small pure helper at `frontend/src/lib/stripAnsi.ts`. Rationale: (1) the SerializeAddon already runs in the renderer, so the bytes never leave the renderer in their ANSI form except via the Wails RPC payload; (2) keeping the strip in TS lets us unit-test it with vitest against the exact set of escape sequences the addon emits (verified: `[...m`, `[...A`, `[...B`, `[...C`, `[...D`, `[...X`, with one or more numeric parameters); (3) backend-side stripping would require shipping a Go `regexp.MustCompile(...)` and re-implementing the same logic; (4) the strip is a one-liner — a focused regex `/\[[?]?[0-9;]*[a-zA-Z]/g` with a `replace(..., '')` covers SerializeAddon's full output (no OSC sequences emitted by `serialize()` per source review). Alternative: skip the strip entirely and document that v3.2 saves "raw terminal serialization (includes color codes)" — but this directly contradicts SER-01's "text-only output" wording and the v3.2 secrets-warning intent.
- **Default filename format.** **Recommendation: `<session-name>-<YYYY-MM-DD-HHmmss>.txt`** (e.g., `claude-code-2026-05-07-143022.txt`). The session name is already user-meaningful (renamed via the existing tab right-click); the timestamp prevents accidental overwrites of an earlier save. Sanitize the session name to filesystem-safe characters via `String.replace(/[^\w\-.]/g, '_')` to prevent edge cases (Pitfall #4). Alternative: just `<session-name>.txt` (no timestamp). Slight UX preference for the timestamped form because v3.2's "save same session multiple times during a long-running task" workflow is plausible.
- **Whether the SaveFileDialog Title includes the secrets warning.** **Recommendation: yes — Title = `"Save Terminal As… (file will include any printed secrets)"`** so the user sees the warning at the moment-of-decision, not just once in Settings days earlier. The string is platform-rendered consistently (macOS, Windows, GNOME). Alternative: plain `"Save Terminal As…"` and rely solely on the Settings caption — viable but weaker; users routinely skim Settings text.
- **Whether the Save menu item appears when Serialize is toggled OFF.** **Recommendation: ALWAYS show the menu item; the click handler shows a one-shot BannerStack toast `"Enable the Serialize plugin in Settings to save sessions"` when the addon isn't loaded.** Rationale: hiding context menu items based on settings state is a discoverability anti-pattern (the user wonders "where's the save action?"). Showing-with-toast is the established v3.x pattern (see Phase 87 capability-token errors, Phase 95 link-confirm). Alternative: hide the item entirely when Serialize is OFF — viable but worse for discoverability.
- **Whether to ship `frontend/src/lib/stripAnsi.ts` or use an existing dependency.** **Recommendation: write our own — it's ~10 lines.** Rationale: zero new deps to audit; the regex `/\[[?]?[0-9;]*[a-zA-Z]/g` covers the entire CSI/SGR vocabulary SerializeAddon emits (verified via `package/src/SerializeAddon.ts` line-by-line audit — see "ANSI Output Audit" below). Adding `strip-ansi@7.x` would import a 7-package transitive tree (strip-ansi → ansi-regex → ...). Alternative: import the npm package — lower implementation cost, higher supply-chain cost. Project posture (Phase 89 vendoring discipline, Phase 93 vendor_drift_test gate) leans heavily toward "small custom > broad transitive."
- **Whether to add a keyboard shortcut for Save (e.g., Cmd-S when xterm is focused).** **Recommendation: NO — context-menu-only in v3.2.** Cmd-S in a terminal context is ambiguous (some users expect it to send Ctrl-S to the running CLI, others expect "save"); safer to ship the explicit gesture only and consider Cmd-S in Phase 99 / future based on user feedback. Alternative: add Cmd-S with focus-conditioned guard (mirror of Phase 94's Cmd-F pattern) — viable, but pile-on risk; defer.
- **Whether to write the SER-03 negative-grep test in Go (`internal/release/`) or TS (`frontend/src/__tests__/`).** **Recommendation: Go-side at `internal/release/no_autosave_test.go`** because Phase 90 already has `internal/release/` for SHA-pin / pipeline-hardening regression tests; SER-03 fits that "release-gate negative regression" shape. Bonus: a single Go test can `filepath.Walk` both `frontend/src/` and the Go tree, applying the regex to all sources. Alternative: a vitest source-inspection test that reads files via `fs.readFileSync` (Phase 89 D-17 pattern). Both are equivalent effort.

### Deferred Ideas (OUT OF SCOPE)

- **HTML output (`serializeAsHTML()`).** [REQUIREMENTS SER-FUT-01] Theme-aware HTML serialization with color preservation. Defer to v3.3.
- **Web-served Save Terminal As…** Browser download via `<a download>` blob URL would be the natural web equivalent; defer to a future `SER-FUT-WEB` if requested. The vendored `addon-serialize.js` file ships in v3.2 to keep `vendor_drift_test.go` green and to avoid a re-vendor step when web parity lands.
- **Selection-only export.** SerializeAddon supports `range` (IMarker | line numbers) and `onlySelection` (HTML-only); exposing these requires UI surface. Defer.
- **Auto-save / scheduled save / save-on-quit.** Forbidden by SER-03. Permanent non-goal.
- **Per-session "save format" preference.** v3.2 ships text-only; preferences live in v3.3+ when there's a second format.
- **Cmd-S keyboard shortcut.** See "Claude's Discretion" — defer to Phase 99 / user feedback.
- **Save to clipboard / share via system shareSheet.** Out of scope; Phase 97 ships file-to-disk only.
- **Session "history" / "recent saves" surface.** Files written to user-chosen paths are managed by the OS file explorer, not by AgentHub.
- **Encryption / password-protection of saved files.** Plain `.txt` write only. Future consideration only if a real privacy concern emerges.

</user_constraints>

---

## Mandatory Pre-Phase CSP Audit

**Audit target:** Upstream `@xterm/addon-serialize@0.14.0` source — both `package/lib/addon-serialize.js` (CJS UMD, used for web vendoring) and `package/lib/addon-serialize.mjs` (ESM, used by frontend bundler). Audit performed on the upstream tarball at `/tmp/serialize-inspect/package/` (downloaded via `npm pack @xterm/addon-serialize@0.14.0`, 2026-05-07).

**Audit method:** `grep -cE "WebAssembly|new Worker|URL.createObjectURL|new Blob|importScripts|eval\(|new Function|blob:|data:text/javascript"` against both bundles. Cross-checked against the human-readable upstream source `package/src/SerializeAddon.ts` (695 lines) for any patterns the minifier might have transformed.

### Findings Table

| Pattern Searched | Count in `.mjs` | Count in `.js` | Count in `src/SerializeAddon.ts` | CSP Directive Affected | Mitigation Required |
|------------------|----------------|---------------|---------------------------------|------------------------|---------------------|
| `WebAssembly.*` | 0 | 0 | 0 | `script-src 'wasm-unsafe-eval'` | None |
| `new Worker(` | 0 | 0 | 0 | `worker-src` | None |
| `URL.createObjectURL(` | 0 | 0 | 0 | `img-src` / `connect-src` (depending on use) | None |
| `new Blob(` | 0 | 0 | 0 | n/a | None |
| literal `blob:` URL strings | 0 | 0 | 0 | n/a | None |
| `data:text/javascript` | 0 | 0 | 0 | `script-src` | None |
| `data:application/...` script construction | 0 | 0 | 0 | `script-src` | None |
| `importScripts(` | 0 | 0 | 0 | n/a (no Worker) | None |
| `eval(` | 0 | 0 | 0 | `script-src 'unsafe-eval'` | None |
| `new Function(` | 0 | 0 | 0 | `script-src 'unsafe-eval'` | None |
| `setInterval(` | 0 | 0 | 0 | n/a | None — also confirms zero auto-save timer (SER-03) |
| `setTimeout(` | 0 | 0 | 0 | n/a | None |
| `requestAnimationFrame(` | 0 | 0 | 0 | n/a | None |

**Audit Conclusion:** addon-serialize is pure JS string concatenation over the xterm.js Buffer cell API. It has zero CSP-relevant constructs, zero timers, zero workers, zero WASM. No CSP changes are required for Phase 97; the Phase 96 IMG-03 amendment (`script-src 'self' 'wasm-unsafe-eval'`) carries forward unchanged.

**Confidence:** HIGH (verified via direct source read of the upstream npm tarball; cross-checked against `package/src/SerializeAddon.ts` 695-line TypeScript source; no new attack surface introduced).

---

## ANSI Output Audit (load-bearing for SER-01 "text-only" requirement)

**Audit target:** The exact set of escape sequences `SerializeAddon.serialize()` emits, sourced from the upstream TypeScript at `package/src/SerializeAddon.ts` (the same code that gets compiled into `lib/addon-serialize.js`).

**Why this matters:** SER-01 says "text-only output." `serialize()` returns text *intermixed with* ANSI escape sequences, which a human reader perceives as garbage. The plan must include a strip step. To strip correctly, we must know exactly what to strip — false negatives (missed sequences = garbage in saved file) and false positives (over-stripping = mangled human text) are the failure modes.

### Sequences `serialize()` emits (verified by source-reading lines 146, 215-223, 245-323 of SerializeAddon.ts)

| Sequence | Pattern | Source line(s) | Purpose |
|----------|---------|---------------|---------|
| SGR (Select Graphic Rendition) | `[<n>(;<n>)*m` | 146, 245-282 (the `_diffStyle` method emits the full SGR vocabulary: `0`, `38;2;<r>;<g>;<b>` (RGB fg), `38;5;<n>` (256 fg), `30-37/90-97` (16 fg), `48;2;<r>;<g>;<b>` (RGB bg), `48;5;<n>` (256 bg), `40-47/100-107` (16 bg), `1/22` (bold), `2/22` (dim), `3/23` (italic), `4/24` (underline), `5/25` (blink), `7/27` (inverse), `8/28` (invisible), `9/29` (strikethrough), `53/55` (overline)) | Color/style transitions between cells |
| ECH (erase character) | `[<n>X` | 146 | Encode null cells (cells with no content but with a background color) |
| CUF (cursor forward) | `[<n>C` | 220, 296, 314 | Skip over null-foreground cells |
| CUB (cursor backward) | `[<n>D` | 215, 222, 316 | Reposition for the alt-buffer cursor finalization |
| CUU (cursor up) | `[<n>A` | 219, 318 | Multi-line cursor positioning |
| CUD (cursor down) | `[<n>B` | 223, 317 | Multi-line cursor positioning |

**Crucially:** there are NO OSC sequences (`]...\\`), NO DCS sequences (`P...\\`), NO C1 controls, NO 8-bit CSI, NO mode-set escapes (`[?...l/h`) emitted by `_serializeString()` itself. (The addon has an `excludeModes: false` option that wraps the output in `[?<n>l/h` mode-restore sequences — but that is appended AFTER `_serializeString()` in `BufferDecoder.serialize()`, and is filterable via the `excludeModes: true` option — see Pitfall #2.) The DEC private-mode escape uses the `?` prefix; our regex must therefore handle the optional `?`.

### Strip regex

```typescript
// Source: handcrafted from the audit above; covers SGR + cursor moves + ECH + DEC private modes.
// Allows optional ? after [ (DEC private modes), 0+ digits/semicolons, then a single-letter terminator.
function stripAnsi(input: string): string {
  return input.replace(/\[\??[0-9;]*[a-zA-Z]/g, '')
}
```

This regex is verified to cover every sequence the audit identified. It does NOT cover OSC/DCS — but those are not emitted by `_serializeString()`. Adding `excludeModes: true` to the `serialize({ excludeModes: true })` call belt-and-braces removes the only mode-restore sequence the addon would otherwise wrap around the output.

**Confidence:** HIGH (verified by direct source read of all 695 lines of `SerializeAddon.ts`; the strip regex matches the addon's emit patterns exactly; the `excludeModes: true` option neutralizes the only edge case).

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Buffer scrollback → text serialization | Browser (xterm.js core + addon-serialize, in-process JS) | — | `SerializeAddon.serialize()` walks `term.buffer.active` cells, no native involvement |
| ANSI escape stripping | Browser (`frontend/src/lib/stripAnsi.ts`) | — | One-line pure helper; runs after `serialize()` returns; output is then passed to the Wails RPC |
| Addon attach/detach (hot-swap on toggle) | Browser (`TerminalPanel.tsx` hot-swap useEffect, `pluginConfig?.serialize` dep-array key) | — | Mirror of Phase 93 WebGL/Clipboard arms; toggling Serialize in Settings adds/removes the addon live without re-mounting the terminal. No buffer-state implications (unlike Unicode 11 or Image addon — those are next-session-only). |
| Save-action callback registry (TerminalPanel → App.tsx) | Browser (`App.tsx` owns a `Record<sessionId, () => string>` keyed by `sessionId`) | — | TerminalPanel exposes `getSerializedText()` via the existing `onWebGLContextLost`-style callback prop pattern (e.g., a new `onRegisterSaver` prop the parent uses to capture the closure). The closure reads `serializeAddonRef.current?.serialize({ excludeModes: true })`. App.tsx looks up the closure for the requested session and invokes it. |
| Right-click "Save Terminal As…" menu item | Browser (`TabBar.tsx` extends the existing context menu with a second `<button role="menuitem">` next to "Rename") | — | The context menu and `tab__name onContextMenu` handler already exist (`TabBar.tsx:124-128`). Only the menu's items list needs extension; no new menu infrastructure. |
| Wails RPC: SaveTerminalSession | Browser → Wails bridge → Go (`(*App).SaveTerminalSession` in `app.go`) | — | Mirror of `(*App).OpenFileDialog` at `app.go:818-829`. Accepts `defaultDir, defaultName, content string`; returns `error`. |
| Native Save File Dialog | Wails runtime (`runtime.SaveFileDialog`) → platform code (`internal/frontend/desktop/{darwin,windows,linux}/dialog.go`) | — | Wails v2.12.0 abstracts macOS NSSavePanel, Windows GetSaveFileNameW, Linux GTK FileChooserDialog behind a single Go API. Cancellation = `("", nil)`. |
| File write to disk | Go (`os.WriteFile(path, []byte(content), 0o644)`) | — | Standard library; no streaming needed (10K-line scrollback × ~80 cols × ~5 bytes/cell after strip = ~4 MB worst case, well within atomic-write thresholds) |
| Vendored addon serving | CDN / Static (`web/vendor/xterm/addons/addon-serialize.js` served via Go embed.FS at `/assets/xterm/addons/addon-serialize.js`) | — | WEB-01 vendoring discipline; no runtime web consumer in v3.2 |
| `vendor_drift_test.go` CI gate | CI / Go test | — | Phase 93's generalized regex `(@xterm/(?:xterm|addon-[\w-]+))` matches `@xterm/addon-serialize` automatically; bump min-count guard from 8 to 9 |
| Secrets-warning caption | Browser (`PluginsSection.tsx`, italic `settings-panel__description--italic` class on the 4th renderRow argument) | — | Pure markup; no logic |
| SER-03 no-autosave regression test | CI / Go test (`internal/release/no_autosave_test.go`) | — | filepath.Walk + regex; covers both Go and frontend trees in one test |

**Cross-tier note for SER-03:** The "no auto-save" guarantee crosses two trees (Go + TS). A single Go test using `filepath.Walk` + `regexp.MatchString` against a small set of forbidden patterns provides forever-defense at minimal cost. This is the pattern Phase 88 established for "OriginPatterns: ["*"] reintroduction."

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/addon-serialize` | `^0.14.0` | Walks `term.buffer.active` cells, emits a string of ANSI-escape-laden text reproducing the visible scrollback (and optionally the alt buffer) | First-party `@xterm` scoped addon; drop-in compatible with the project's `@xterm/xterm@^6.0.0` core; same family as already-shipped `@xterm/addon-fit`, `@xterm/addon-webgl`, `@xterm/addon-search`, `@xterm/addon-web-links`, `@xterm/addon-image`, `@xterm/addon-unicode11`, `@xterm/addon-clipboard`. Latest non-beta release. |

**Verified:** `pnpm view @xterm/addon-serialize` returned `0.14.0` (latest), `0.15.0-beta.216` (beta), `deps: none`, `main: lib/addon-serialize.js`, `module: lib/addon-serialize.mjs`, `types: typings/addon-serialize.d.ts`, `unpackedSize: 205.8 kB` (verified 2026-05-07). [VERIFIED: npm registry, 2026-05-07 via `pnpm view @xterm/addon-serialize`]

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — addon-serialize has zero runtime dependencies) | — | — | Verified by `pnpm view @xterm/addon-serialize peerDependencies` returning empty and inspection of `package.json` showing no `dependencies` entry. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `@xterm/addon-serialize@0.14.0` | `0.15.0-beta.216` | Beta tag implies API instability; `serializeAsHTML` API surface changes are common in pre-stable; v3.2 ships stable, v3.3 reconsiders. |
| `@xterm/addon-serialize` | Custom buffer-walker (`for row in term.buffer.active.viewportY ... term.buffer.active.cursorY: row.translateToString()`) | Re-implements ~700 lines of carefully-tested cell-diff and SGR emission. The `translateToString()` API is line-only (no buffer-wide), and skipping color codes loses theme/style information regardless. **Don't hand-roll.** |
| `@xterm/addon-serialize` (text via strip) | `@xterm/addon-serialize` calling `serializeAsHTML()` | HTML serialization is theme-aware but explicitly out of scope for v3.2 (SER-FUT-01); the v3.2 contract is text-only. |
| Wails `runtime.SaveFileDialog` | Browser-side `<a download>` blob URL | The latter works in Wails too (the WebView is a real browser), but it bypasses the OS native Save dialog (no Default Directory persistence, no Filters, no platform-consistent UX). Choose the native dialog for desktop. |
| Custom strip-ansi (`frontend/src/lib/stripAnsi.ts`) | `strip-ansi@7.x` from npm | npm package adds a 7-package transitive dep tree (strip-ansi → ansi-regex → ...). For ~10 lines of code with a verified regex, custom is the project posture. |

**Installation:**

```bash
cd frontend && pnpm add @xterm/addon-serialize@^0.14.0
```

After install, copy `frontend/node_modules/@xterm/addon-serialize/lib/addon-serialize.js` to `web/vendor/xterm/addons/addon-serialize.js` (Phase 93/94/95/96 pattern; byte-identical to source). Append `@xterm/addon-serialize@0.14.0` to `web/vendor/xterm/VERSION`. Bump `internal/webserver/vendor_drift_test.go` min-count guard from 8 to 9.

**Version verification:**

```bash
pnpm view @xterm/addon-serialize version  # confirmed 0.14.0 on 2026-05-07
```

[VERIFIED: npm registry, 2026-05-07]

---

## SerializeAddon API Contract

### Constructor

```typescript
// [VERIFIED: /tmp/serialize-inspect/package/typings/addon-serialize.d.ts]
new SerializeAddon()  // no constructor args
```

### Methods

```typescript
// [VERIFIED: typings/addon-serialize.d.ts]

// Activate the addon (called automatically by term.loadAddon).
public activate(terminal: Terminal): void;

// Serialize buffer into ANSI-escape-laden string.
public serialize(options?: ISerializeOptions): string;

// HTML serialization (out of scope for v3.2 — SER-FUT-01).
public serializeAsHTML(options?: Partial<IHTMLSerializeOptions>): string;

// Disposes the addon.
public dispose(): void;
```

### Options shape (relevant for Phase 97)

```typescript
// [VERIFIED: typings/addon-serialize.d.ts §ISerializeOptions]
interface ISerializeOptions {
  range?: ISerializeRange       // { start: IMarker | number, end: IMarker | number }
  scrollback?: number           // # rows from BOTTOM of scrollback. UNSPECIFIED = all rows.
  excludeModes?: boolean        // default false; true skips the trailing mode-restore wrap (Pitfall #2)
  excludeAltBuffer?: boolean    // default false; true skips the alt buffer (vim/htop screen)
}
```

**Phase 97 call shape:**

```typescript
// Output: ANSI-escape-laden text covering the FULL scrollback + main buffer
// (no range, no scrollback cap; default behavior — verified per typings comment:
// "When not specified, all available rows in the scrollback buffer will be serialized")
const raw = serializeAddon.serialize({ excludeModes: true })
// Strip ANSI to satisfy SER-01's "text-only output" requirement.
const plainText = stripAnsi(raw)
```

[VERIFIED: typings/addon-serialize.d.ts source-read 2026-05-07]

---

## Wails SaveFileDialog API Contract

### Method signature

```go
// [VERIFIED: /Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.12.0/pkg/runtime/dialog.go:65-74]
func SaveFileDialog(ctx context.Context, dialogOptions SaveDialogOptions) (string, error)
```

### Options shape

```go
// [VERIFIED: github.com/wailsapp/wails/v2/internal/frontend/options.go — SaveDialogOptions struct;
// type alias forwarded via runtime/dialog.go line 18]
type SaveDialogOptions struct {
    DefaultDirectory           string       // Initial directory (must exist or empty)
    DefaultFilename            string       // Initial filename in the dialog
    Title                      string       // Dialog title
    Filters                    []FileFilter // {DisplayName, Pattern} pairs (e.g., "*.txt;*.log")
    ShowHiddenFiles            bool
    CanCreateDirectories       bool
    TreatPackagesAsDirectories bool          // macOS only; expand .app/.bundle as directories
}

type FileFilter struct {
    DisplayName string  // e.g., "Text Files (*.txt)"
    Pattern     string  // e.g., "*.txt;*.log" (semicolon-separated)
}
```

### Return values

- **User confirmed a path:** non-empty string + nil error. The path is the absolute filesystem path the user chose; the dialog's "Save" button has been clicked.
- **User cancelled:** empty string + nil error. (NO error is returned — cancellation is a normal flow, not a failure.)
- **Dialog setup failure (rare):** empty string + non-nil error (e.g., `defaultDirectory does not exist`).

**Critical pattern:** the App-side caller MUST treat empty path as success-with-no-action, NOT as an error. Mirror the existing `OpenFileDialog` pattern at `app.go:818-829` and `OpenDirectoryDialog` at `app.go:803-813`.

### Phase 97 call shape

```go
// New method in app.go — mirror of OpenFileDialog signature pattern.
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
        return nil // user cancelled — silent success
    }
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        return fmt.Errorf("SaveTerminalSession: write: %w", err)
    }
    return nil
}
```

### Cross-platform notes

| Platform | Native dialog | Notes |
|----------|--------------|-------|
| macOS | NSSavePanel | TreatPackagesAsDirectories matters here (default false; `.app` bundles appear as files). The DefaultFilename's extension auto-applies if user doesn't type one. |
| Windows | GetSaveFileNameW | Filters use `*.ext` Pattern (semicolon-separated for multi). Default filename auto-extension behavior depends on Filter being non-empty. |
| Linux (GTK) | GtkFileChooserDialog | Filters apply post-typing; less aggressive auto-extension than macOS/Windows. Recommendation: always set DefaultFilename with `.txt` suffix. |

**Confidence:** HIGH (verified via direct source-read of `runtime/dialog.go` v2.12.0; verified via grep of existing `OpenFileDialog` use at `app.go:818-829` for the cancellation-returns-empty-path pattern).

---

## Architecture Patterns

### System Architecture Diagram

```
                    ┌──────────────────────────────────────────────────┐
                    │              User right-clicks tab               │
                    └────────────────┬─────────────────────────────────┘
                                     │ MouseEvent type=contextmenu
                                     ▼
       ┌────────────────────────────────────────────────────┐
       │ TabBar.tsx                                         │
       │  - existing onContextMenu sets contextMenu state   │
       │  - menu has [Rename, Save Terminal As…]            │
       │  - "Save…" calls onRequestSave(tabId)              │  ◄── NEW Phase 97
       └─────────────────────┬──────────────────────────────┘
                             │ onRequestSave(tabId)
                             ▼
       ┌────────────────────────────────────────────────────┐
       │ App.tsx                                            │
       │  - serializerRegistry: Record<sessionId,           │
       │       () => string>                                │  ◄── NEW Phase 97
       │  - registered by TerminalPanel via                 │
       │       onRegisterSaver(sessionId, fn)               │
       │  - handleRequestSave(tabId):                       │
       │      const fn = serializerRegistry[sessionId]      │
       │      if !fn → toast "Enable Serialize plugin…"     │
       │      const raw = fn()                              │
       │      const plain = stripAnsi(raw)  ◄── frontend    │
       │      const fname = sanitize(name)+stamp+'.txt'     │
       │      await SaveTerminalSession('',fname,plain)     │
       └─────────────────────┬──────────────────────────────┘
                             │ Wails Call('main.App.SaveTerminalSession',…)
                             ▼
       ┌────────────────────────────────────────────────────┐
       │ app.go (Go)                                        │
       │  (*App).SaveTerminalSession(dir, name, content)    │  ◄── NEW Phase 97
       │  ├─ runtime.SaveFileDialog(...)                    │
       │  │    → path string, err error                     │
       │  │    → cancel = ("", nil)                         │
       │  └─ os.WriteFile(path, []byte(content), 0o644)     │
       └────────────────────────────────────────────────────┘

  Parallel TerminalPanel side:

       ┌────────────────────────────────────────────────────┐
       │ TerminalPanel.tsx                                  │
       │  hot-swap useEffect dep-array adds                 │
       │    pluginConfig?.serialize                         │  ◄── NEW Phase 97
       │  ┌─ on attach:                                     │
       │  │    serializeAddon = new SerializeAddon()        │
       │  │    term.loadAddon(serializeAddon)               │
       │  │    serializeAddonRef.current = serializeAddon   │
       │  │    onRegisterSaver(sessionId, () =>             │
       │  │       serializeAddon.serialize({excludeModes}))│
       │  └─ on detach:                                     │
       │       serializeAddon.dispose()                     │
       │       onRegisterSaver(sessionId, null)             │
       └────────────────────────────────────────────────────┘
```

### Recommended Project Structure (additions only)

```
frontend/src/
├── components/
│   ├── TabBar.tsx                # EXTEND: add "Save Terminal As…" menu item
│   ├── TerminalPanel.tsx         # EXTEND: hot-swap arm + serializeAddonRef + onRegisterSaver
│   ├── PluginsSection.tsx        # EXTEND: italic SER-02 secrets caption
│   └── __tests__/
│       ├── TabBar.test.tsx                # EXTEND: new menu item present
│       ├── TerminalPanel.test.tsx         # EXTEND: addon attach/detach + register/unregister
│       └── PluginsSection.test.tsx        # EXTEND: SER-02 caption present
└── lib/
    ├── stripAnsi.ts              # NEW: pure helper (~10 lines)
    └── __tests__/
        └── stripAnsi.test.ts     # NEW: vitest cases for SGR/CUF/CUB/CUU/CUD/ECH

internal/release/
└── no_autosave_test.go           # NEW: SER-03 negative-grep regression test

web/vendor/xterm/
├── addons/
│   └── addon-serialize.js        # NEW: vendored UMD copy
└── VERSION                       # EXTEND: append @xterm/addon-serialize@0.14.0

internal/webserver/
└── vendor_drift_test.go          # EXTEND: bump min-count from 8 to 9

frontend/src/wailsjs/go/main/
├── App.d.ts                      # EXTEND: SaveTerminalSession signature
└── App.js                        # EXTEND: SaveTerminalSession Call() stub

app.go                            # EXTEND: (*App).SaveTerminalSession method

web/terminal.html                 # EXTEND (optional): <script src=…/addon-serialize.js> tag
```

### Pattern 1: Hot-Swap Addon Arm (mirror of WebGL/Clipboard/Search/WebLinks)

**What:** SerializeAddon attaches/detaches live based on `pluginConfig?.serialize`; no terminal re-mount required.

**When to use:** Any plugin whose load/unload has no buffer-state implications (the SerializeAddon doesn't render anything; it only walks the buffer when invoked).

**Example:**

```typescript
// Source: extends TerminalPanel.tsx hot-swap useEffect (lines 329-495)
// Pattern verified against existing webgl/clipboard/search/webLinks arms.

// ADD to dep array (line 495):
//   [pluginConfig?.webgl, pluginConfig?.clipboard, pluginConfig?.search,
//    pluginConfig?.webLinks, pluginConfig?.serialize, onWebGLContextLost, sessionId]

if (pluginConfig?.serialize) {
  if (!serializeAddonRef.current) {
    const serializeAddon = new SerializeAddon()
    term.loadAddon(serializeAddon)
    serializeAddonRef.current = serializeAddon
    // Register the saver closure with App.tsx so the right-click handler
    // can reach this terminal's serializer.
    onRegisterSaver?.(sessionId, () =>
      serializeAddon.serialize({ excludeModes: true }))
  }
} else {
  if (serializeAddonRef.current) {
    serializeAddonRef.current.dispose()
    serializeAddonRef.current = null
    onRegisterSaver?.(sessionId, null)  // unregister
  }
}
```

### Pattern 2: Saver Registry (TerminalPanel → App.tsx)

**What:** App.tsx holds a `Record<sessionId, () => string>` map. Each TerminalPanel registers a closure when its SerializeAddon attaches and unregisters on detach/unmount. The TabBar's Save action calls App.tsx, which looks up the closure for the active tab.

**Why:** TabBar is structurally above TerminalPanel; TerminalPanel can't directly handle the right-click. The closure pattern is identical to React's "child registers callback with parent" idiom and avoids the antipatterns of `useImperativeHandle` (which would require imperative refs across the component tree).

**Example:**

```typescript
// Source: handcrafted; mirrors React useState + useCallback patterns already
// in App.tsx (handleWebGLContextLost at App.tsx:168 uses the same callback-prop shape).

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

### Pattern 3: ANSI-Strip Helper (custom, ~10 lines)

**What:** A single regex replacement that removes all CSI/SGR/cursor sequences emitted by `SerializeAddon.serialize()`. Verified against the addon source.

**Example:**

```typescript
// Source: handcrafted; regex verified against /tmp/serialize-inspect/package/src/SerializeAddon.ts
// (full audit in §"ANSI Output Audit" above).
//
// Coverage: SGR ([<n>(;<n>)*m), ECH ([<n>X), CUF/CUB/CUU/CUD,
// DEC private modes ([?<n>l|h — only emitted when excludeModes:false).
//
// We pass excludeModes:true at the call site, but the regex covers DEC modes
// belt-and-braces in case a future addon update adds them outside that flag.

export function stripAnsi(input: string): string {
  return input.replace(/\[\??[0-9;]*[a-zA-Z]/g, '')
}
```

### Pattern 4: Filename Sanitization (Pitfall #4 mitigation)

**What:** Sanitize the session name before putting it in `DefaultFilename` to avoid path-traversal characters and platform-incompatible characters (`/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`).

**Example:**

```typescript
// Replace any non-[word, hyphen, dot] character with underscore.
// Word covers [A-Za-z0-9_], so this is intentionally restrictive.
export function sanitizeFilename(name: string): string {
  // Trim, collapse whitespace, then replace forbidden characters.
  const collapsed = name.trim().replace(/\s+/g, '_')
  const sanitized = collapsed.replace(/[^\w\-.]/g, '_')
  // Avoid empty names + leading dots (hidden files) + reserved Windows names.
  if (sanitized === '' || sanitized.startsWith('.')) return 'session'
  if (/^(con|prn|aux|nul|com[1-9]|lpt[1-9])$/i.test(sanitized)) return 'session'
  return sanitized
}
```

### Pattern 5: Negative Regression Test (SER-03)

**What:** A Go test using `filepath.Walk` + `regexp.MatchString` that fails if any source file in the project contains a forbidden auto-save pattern.

**Example:**

```go
// Source: handcrafted; mirrors Phase 88 OriginPatterns: ["*"] reintroduction guard
// pattern (https://github.com/scottkw/agenthub/blob/main/internal/webserver/origin_mw_test.go).
package release

import (
    "io/fs"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
)

// SER-03 — three forbidden patterns:
//   1. setInterval(...serialize(...)
//   2. setTimeout(...serialize(...), N) for any N >= 1000ms (long-delay schedule)
//   3. BeforeQuit / OnShutdown / OnBeforeClose calling serialize
//
// And one forbidden settings shape:
//   4. PluginSettings.AutoSave / AutoExport / AutoCapture / SaveOnQuit
var ser03Forbidden = []*regexp.Regexp{
    regexp.MustCompile(`setInterval\([^)]*[Ss]eriali[zs]e\(`),
    regexp.MustCompile(`setTimeout\([^,]*[Ss]eriali[zs]e\([^,]+,\s*[0-9]{4,}`),
    regexp.MustCompile(`(?i)\bBeforeQuit\b[\s\S]{0,200}?[Ss]eriali[zs]e`),
    regexp.MustCompile(`(?i)\bOn(Shutdown|BeforeClose)\b[\s\S]{0,200}?[Ss]eriali[zs]e`),
    regexp.MustCompile(`(?i)\bauto(Save|Export|Capture)\b\s+bool`),
    regexp.MustCompile(`(?i)"auto(Save|Export|Capture)"`),
}

var ser03Roots = []string{
    "../../frontend/src",
    "../../",  // Go tree, excluding vendor + node_modules in the walk
}

func TestSER03_NoAutoSavePatterns(t *testing.T) {
    for _, root := range ser03Roots {
        err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
            if err != nil { return err }
            if info.IsDir() {
                if strings.Contains(path, "node_modules") ||
                   strings.Contains(path, "vendor/") ||
                   strings.Contains(path, ".git") ||
                   strings.HasSuffix(path, "/release") /* skip self */ {
                    return filepath.SkipDir
                }
                return nil
            }
            if !strings.HasSuffix(path, ".go") &&
               !strings.HasSuffix(path, ".ts") &&
               !strings.HasSuffix(path, ".tsx") {
                return nil
            }
            data, err := os.ReadFile(path)
            if err != nil { return err }
            for _, re := range ser03Forbidden {
                if re.Match(data) {
                    t.Errorf("SER-03 violation in %s: matched forbidden pattern %s", path, re.String())
                }
            }
            return nil
        })
        if err != nil { t.Fatalf("Walk(%s): %v", root, err) }
    }
}
```

### Anti-Patterns to Avoid

- **Don't conflate "Serialize plugin enabled" with "save feature enabled."** The toggle controls addon load. The save action is always-visible in the menu and shows a toast if the addon isn't loaded. (See SER-02 row + Claude's Discretion #4.)
- **Don't call `serialize()` from any timer or shutdown hook.** SER-03 forbids it. Negative regression test catches this forever.
- **Don't pass the ANSI-laden text to `os.WriteFile` directly.** SER-01 says "text-only output." The strip step is non-negotiable.
- **Don't use `useImperativeHandle` to call `serialize()` from TabBar.** It violates React's data-flow principle and complicates testing. Use the saver-registry callback pattern (Pattern 2).
- **Don't set `runtime.SaveDialogOptions.DefaultDirectory` to a path that doesn't exist.** Wails returns an error before showing the dialog (`runtime/dialog.go:68-71`). Mirror the existing OpenFileDialog pattern: empty string → fall back to `os.UserHomeDir()`.
- **Don't strip ANSI on the Go side.** It would require shipping a Go regex and double-implementing the same logic. Frontend strip keeps the ANSI bytes from leaving the renderer process (other than via the Wails RPC payload).
- **Don't add `excludeAltBuffer: true` reflexively.** v3.2 ships full default behavior — including the alt buffer if the user is in vim/htop/less when they save. Document this; users who don't want it can scroll back to a non-alt-buffer state before saving (or future SER-FUT-* exposes the option).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Buffer cell → text conversion | Custom buffer-walker using `term.buffer.active.getLine(y).translateToString()` for every row | `@xterm/addon-serialize.serialize()` | The addon handles SGR diff state, null-cell padding, multi-cell wide chars, and (optionally) alt-buffer correctly; reinventing is ~700 lines and silently buggy on edge cases |
| Native Save dialog | `<a download>` blob URL OR cgo NSSavePanel call | `runtime.SaveFileDialog` | Wails abstracts macOS/Windows/Linux behind one API; no platform branching needed |
| ANSI escape detection | Walk each character checking for `` and parsing CSI parameters by hand | A single regex `/\[\??[0-9;]*[a-zA-Z]/g` | The audit confirms SerializeAddon's emit vocabulary is bounded to that regex |
| Filename timestamp | `Date.now() / 60000` math | `new Date().toISOString().replace(...)` | Stdlib gives ISO-8601; minimal transform yields filesystem-safe form |
| File-system-safe character sanitization | Custom switch over forbidden chars | `String.replace(/[^\w\-.]/g, '_')` regex | One-liner; covers Windows reserved chars + path traversal |
| Auto-save regression detection | Code-review checklists + memory | `regexp.MustCompile + filepath.Walk` test | Forever-defense; runs in every CI build |

**Key insight:** Phase 97 is a small phase whose risk is *temptation* — the temptation to add features (HTML output, auto-save, web parity, Cmd-S) that aren't in scope, and the temptation to hand-roll glue (buffer walker, native dialog, ANSI parser) that already exists. Stick to the prescribed pattern. Each of the 7 don't-hand-rolls above has a published-and-tested replacement; every "but it's just a few lines" inversion has a debt cost downstream.

---

## Common Pitfalls

### Pitfall 1: Calling `serialize()` Without `excludeModes: true`

**What goes wrong:** The output gets wrapped in DEC private mode set/restore sequences (`[?<n>l/h`), which the strip regex catches but cleanly avoids if disabled.

**Why it happens:** `excludeModes` defaults to `false` to support "replay into a terminal" use cases. We're not replaying; we're saving to disk for human reading.

**How to avoid:** Always pass `{ excludeModes: true }` in the `serialize()` call. Use `excludeAltBuffer: false` (default) so users in vim/htop see what they were looking at.

**Warning signs:** A test asserting the saved file contains no `` characters fails when the strip regex misses an edge case (e.g., a future addon update adds a new sequence type).

### Pitfall 2: Including the Alt Buffer Without Warning

**What goes wrong:** A user in vim hits "Save Terminal As…" expecting the file to contain their PROMPT history; instead they get the vim screen contents.

**Why it happens:** `excludeAltBuffer` defaults to `false` — vim/htop/less use the alt buffer to render their UI; SerializeAddon includes it by default.

**How to avoid:** Document the behavior in SER-02's caption (or a hover tooltip): "Saves the visible terminal state — including the alt buffer if you're in vim/htop." Phase 99 may expose `excludeAltBuffer` as an Advanced disclosure if users complain. Phase 97 ships the default.

**Warning signs:** UAT confusion when a tester saves while in vim and the file contains vim's UI characters instead of shell history.

### Pitfall 3: Empty Path Returned by SaveFileDialog Treated as Error

**What goes wrong:** A naive caller treats the empty-path return as a failure and shows an error toast — but the user simply hit Cancel.

**Why it happens:** Many file-dialog APIs return null/error on cancel; Wails returns `("", nil)` to distinguish cancel-success from setup-failure.

**How to avoid:** In `(*App).SaveTerminalSession`, after calling `runtime.SaveFileDialog`, check `if path == "" { return nil }` BEFORE attempting `os.WriteFile`. Mirror the existing `OpenFileDialog` pattern at `app.go:818-829`.

**Warning signs:** Users see "Could not save: invalid path" after clicking Cancel.

### Pitfall 4: Filename Path Traversal / Cross-Platform Incompatibility

**What goes wrong:** A user names a session `../../etc/passwd` (or `con` on Windows, or includes `:` on macOS pre-2017); the filename in the dialog inherits the dangerous form.

**Why it happens:** Tab names are user-controlled strings. If passed verbatim into `DefaultFilename`, the OS dialog might render confusingly or even let through traversal characters.

**How to avoid:** Sanitize via `String.replace(/[^\w\-.]/g, '_')` BEFORE setting DefaultFilename. The OS Save dialog itself prevents the user from clicking Save on a path containing `/` or `\` (those are path separators), but defense-in-depth on our side means the dialog opens with a clean name.

**Warning signs:** Filenames showing up with `_` substitutions on systems where the user expected the original name.

### Pitfall 5: ANSI Strip False-Positive on Legitimate Terminal Text

**What goes wrong:** A user has output that contains literal `\x1b[` characters (e.g., a debug log of escape sequences); the strip regex removes them.

**Why it happens:** The regex `/\[\??[0-9;]*[a-zA-Z]/g` matches any `ESC [ ...` — there's no way to distinguish "real ANSI emitted by the terminal" from "literal characters that happened to spell ESC [".

**How to avoid:** This is intentionally accepted. SerializeAddon's `serialize()` ALWAYS emits the addon's own SGR/cursor sequences (it's not a passthrough); literal `\x1b[` characters in the source PTY stream get rendered into the buffer cells and then re-emitted as the corresponding SGR (lossless on color, lossy on the literal escape). For Phase 97 v3.2, "what you see in the saved text-only file is what you saw in the terminal as glyphs" is the correct contract. Document.

**Warning signs:** None in normal use; only matters for niche debug-the-terminal-with-the-terminal use cases.

### Pitfall 6: Saver Registry Memory Leak (Stale Closure on Session Close)

**What goes wrong:** A TerminalPanel unmounts but its registered saver closure stays in App.tsx's `serializerRegistry`, holding a reference to the disposed Terminal/SerializeAddon — which keeps the entire buffer in memory until App.tsx re-renders.

**Why it happens:** TerminalPanel's hot-swap useEffect cleanup correctly nulls `serializeAddonRef.current`, but the closure in App.tsx's registry still captures `serializeAddon` from the time it was registered.

**How to avoid:** TerminalPanel's MOUNT useEffect cleanup function (the one with `[sessionId]` dep array) MUST call `onRegisterSaver(sessionId, null)` to evict the entry. The hot-swap useEffect's else branch does this naturally for "user toggles Serialize off"; the mount cleanup adds it for "user closes the tab."

**Warning signs:** Long-running test suites accumulate memory; killing-and-recreating tabs leaks an addon ref per cycle.

### Pitfall 7: Hot-Swap Re-Registration Overwrites a Stale Disabled Closure

**What goes wrong:** User toggles Serialize off (closure unregistered), then on (new closure registered). If the unregister-on-toggle-off skips the explicit `onRegisterSaver(sessionId, null)` because we're "racing" with re-attach, we get a stale closure pointing at a disposed addon.

**Why it happens:** React useEffect cleanup runs synchronously; the effect body runs after. If we return early on `pluginConfig?.serialize === false` without unregistering, then re-render with `serialize === true` constructs a new addon and overwrites the registry entry — but if the user is FAST on the toggle, we could in principle have a brief window where an old closure is callable.

**How to avoid:** ALWAYS unregister in the `else` branch. The pattern in "Pattern 1: Hot-Swap Addon Arm" above is correct — every detach is paired with `onRegisterSaver(sessionId, null)`. Verified by the test "toggling Serialize off-then-on does not leak old closure."

**Warning signs:** Save action returns garbage from a previous session's buffer.

### Pitfall 8: SerializeAddon Loaded Before Buffer Has Content

**What goes wrong:** A new tab opens; user immediately right-clicks → Save Terminal As… → file is empty (or contains only the prompt that hasn't been typed yet).

**Why it happens:** Edge case is fine — empty buffer is a valid serialization. But if the user expected "save what's there" they may be surprised.

**How to avoid:** Document. Phase 97 saves what's visible at the moment the user clicks Save. No PTY drain, no wait-for-prompt logic.

**Warning signs:** UAT testers report "save right after opening a tab gives me an empty file" — confirm that's the expected behavior.

### Pitfall 9: UTF-8 BOM / Line-Ending Inconsistency

**What goes wrong:** Saved `.txt` files open in some Windows editors with garbled characters or wrong line counts.

**Why it happens:** Some Windows editors auto-detect encoding via BOM; a missing BOM might trigger CP1252 fallback and garble UTF-8 multi-byte chars. Conversely, adding a BOM breaks Unix tools that don't expect it. Line endings are similarly contested: Notepad pre-Windows-10 doesn't render LF-only files correctly.

**How to avoid:** Phase 97 ships UTF-8 WITHOUT BOM, with LF line endings (the addon's output uses `\n`). Modern Windows Notepad (1809+) handles LF-only files correctly. macOS / Linux are fine. Document the choice; if a real user reports a problem, Phase 99 can add an Advanced disclosure for "Add CRLF line endings (Windows)."

**Warning signs:** Tester on old Windows reports "the file is one giant line."

### Pitfall 10: Concurrent SaveFileDialog Calls Block the Wails Main Thread

**What goes wrong:** User somehow triggers two Save dialogs simultaneously (rapid right-click + keyboard shortcut?). The second call may queue or block.

**Why it happens:** Native Save dialogs are modal on the Wails main thread; you can't open two at once on macOS (the first is dismissed before the second appears) or Windows (race-y).

**How to avoid:** Add a guard in App.tsx's `handleRequestSave`: a `savingRef` boolean that's set true on entry and cleared in the finally block of the await. Reject re-entry while saving is true. Pattern: similar to the "saving" three-state save button in PluginsSection.

**Warning signs:** Two stacked dialogs in UAT; second one blocks until first is dismissed; user confused about which one's path is being written to.

### Pitfall 11: Vendored UMD Drift After pnpm Update

**What goes wrong:** A developer runs `pnpm update`, bumping `@xterm/addon-serialize` to 0.14.1; `web/vendor/xterm/addons/addon-serialize.js` is stale; CI fails on `vendor_drift_test.go`.

**Why it happens:** The drift gate compares `pnpm-lock.yaml` to `web/vendor/xterm/VERSION`; both files must be re-touched on every package update.

**How to avoid:** Phase 89 D-05 already documents this: "Don't bump the lockfile without re-copying the vendored files." Same procedure applies to addon-serialize.

**Warning signs:** CI fails immediately after a pnpm update.

### Pitfall 12: Phase 96 IMG Test Min-Count Pinning

**What goes wrong:** Phase 96 set the `vendor_drift_test.go` min-count guard to 8 (tracking 8 packages: xterm, addon-fit, addon-webgl, addon-unicode11, addon-clipboard, addon-search, addon-web-links, addon-image). Phase 97 adds a 9th (addon-serialize). If the bump from 8 → 9 is forgotten, the test passes but allows a future regression where someone removes addon-serialize from the lockfile without re-checking.

**Why it happens:** Easy to forget; the test still passes when min-count is 8 even though 9 packages exist (the test asserts ≥ min, not exactly min).

**How to avoid:** Bump in the same commit as the VERSION-line addition. Plan must explicitly call out `vendor_drift_test.go` line 35: `len(pnpmVersions) < 8` → `< 9`.

**Warning signs:** Add test to assert `len(pnpmVersions) == 9` (or ≥) at start of TestXtermVendorVersionsMatchPnpmLock for clearer test failure on missing entry.

---

## Code Examples

Verified patterns from existing source:

### Existing Wails dialog pattern (the template for SaveTerminalSession)

```go
// Source: /Users/ken/dev/agenthub/app.go:815-829
// OpenFileDialog opens a native OS file picker and returns the selected path.
// Returns "" if the user cancels.
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

### Existing TabBar context-menu pattern (the template for "Save Terminal As…" menu item)

```typescript
// Source: /Users/ken/dev/agenthub/frontend/src/components/TabBar.tsx:152-170
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
    {/* Phase 97 SER-01 — NEW menu item below */}
    {/* <button role="menuitem" onClick={() => { onRequestSave?.(contextMenu.tabId); setContextMenu(null) }}>Save Terminal As…</button> */}
  </div>
)}
```

### Existing hot-swap pattern (the template for serialize arm)

```typescript
// Source: /Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx:367-379
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
// Phase 97 SER-01 — NEW hot-swap arm follows the same shape; see Pattern 1 above.
```

### Existing PluginsSection caption pattern (the template for SER-02)

```tsx
// Source: /Users/ken/dev/agenthub/frontend/src/components/PluginsSection.tsx:107-111
// Caption is the 4th argument of renderRow.
{renderRow('image', 'Inline images',
  'Render images sent via sixel or the iTerm2 inline image protocol directly inside the terminal.',
  'Applies to new sessions you create.')}

// Phase 97 SER-02 (extend at line 138):
{renderRow('serialize', 'Save terminal as text',
  'Right-click a tab to export the visible scrollback as a text file.',
  'Saved files include any secrets, tokens, or sensitive data printed in the session.')}
```

### Existing sub-key RPC pattern (NOT used by Phase 97 — Serialize is a bare bool)

```go
// Source: /Users/ken/dev/agenthub/app.go:589-611 — SetImageConfig
// Phase 97 does NOT need a SerializeConfig sub-key — the boolean toggle is the only state.
// PluginsSection's existing SetPluginSettings call covers the full PluginSettings save.
// (If a future SER-FUT adds Advanced options, they'd ship as a new SerializeConfig
// nested struct using this exact pattern.)
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Browser `<a download>` for Wails save flows | Wails `runtime.SaveFileDialog` | Wails v2.0+ (2022) | Native OS dialog with platform-consistent UX; preserves DefaultDirectory across sessions |
| Custom buffer-walker | `@xterm/addon-serialize.serialize()` | xterm.js 4.0+ (2020) | Correct handling of SGR diff state, null cells, multi-cell wide chars |
| `strip-ansi` npm package | Custom regex (~10 lines) | This phase / project posture | Zero new dependencies; verified coverage matches addon emit vocabulary exactly |
| Cmd-S keyboard shortcut as default | Context-menu-only invocation | This phase | Avoids ambiguity with terminal-passed Ctrl-S; safer default for v3.2 |
| Auto-save / save-on-quit | Explicit-gesture-only | This phase / privacy posture | Aligns with v3.1's "tailnet membership ≠ auto-trust" mental model |

**Deprecated/outdated:**
- `xterm.js` versions <4.0 do not have `@xterm/addon-serialize` (the addon predates the `@xterm` scope rename in 2024). v3.2 ships `@xterm/xterm@6.0.0`, well past that boundary.
- `addon-serialize@0.10.x` and earlier did not expose `excludeAltBuffer` or `excludeModes`. v3.2 ships 0.14.0.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The default-filename timestamp format `<name>-<YYYY-MM-DD-HHmmss>.txt` is acceptable to users. | "Claude's Discretion / Default filename format" | Low — single value with easy reversal; if users hate it, plain `<name>.txt` is one regex change |
| A2 | macOS Notarization does not require special handling for the "Save File" gesture (no entitlements changes). | "Wails SaveFileDialog API Contract" | Low — `runtime.SaveFileDialog` is in widespread Wails v2 use; if there's an entitlements gap it would have surfaced years ago |
| A3 | Strip regex `/\[\??[0-9;]*[a-zA-Z]/g` covers the entire emit vocabulary of `serialize({ excludeModes: true })` for addon-serialize 0.14.0. | "ANSI Output Audit" | Low — verified by direct source-read of all 695 lines of `SerializeAddon.ts`; future addon updates that add new sequence types would need a regex extension (caught by a vitest unit test asserting the saved file has zero `` characters on a synthetic full-vocabulary fixture) |
| A4 | Users prefer a context-menu invocation over a Cmd-S shortcut. | "Claude's Discretion / Cmd-S keyboard shortcut" | Low-medium — easy to add Cmd-S in Phase 99 / future if requested |
| A5 | The native Save dialog title text "(file will include any printed secrets)" is rendered consistently across macOS/Windows/Linux without truncation. | "User Constraints / Locked Decisions" + Wails contract | Low — title bars accept ~100+ chars on all three platforms; the 60-char string fits comfortably |
| A6 | Web parity (browser-side save via `<a download>`) is acceptable to defer to a future phase. | "User Constraints / Locked Decisions / web parity" | Medium — if user explicitly wants browser-side save in v3.2, plan must be revised. **PLANNER: confirm with user.** |
| A7 | The addon's `serialize({ excludeModes: true })` does not throw or hang on a 10000-line scrollback. | "API contract" + Pitfall #8 | Low — the addon is well-tested upstream; performance benchmarks in upstream's `npm run benchmark` script. UAT validates with real 10K-line content. |

**If any of these need user confirmation before locking:** A6 (web parity scope) is the highest-leverage. A4 (Cmd-S) is a small UX call. The rest are mechanical.

---

## Open Questions

> All questions answered inline based on ROADMAP, REQUIREMENTS, and existing-pattern reasoning. Listed here for transparency in case the planner wants to re-litigate any.

1. **Web parity scope: ship in Phase 97 or defer to SER-FUT?**
   - What we know: The vendored addon-serialize.js file ships in Phase 97 (vendoring discipline + `vendor_drift_test.go` requires it). The `<script>` tag in `web/terminal.html` is optional. There's no natural web equivalent to the Wails native Save dialog except `<a download>` blob URL.
   - What's unclear: Whether users care about "right-click a tab in the browser-served terminal page → save."
   - Recommendation: Ship vendor file + `<script>` tag; SKIP browser save UI in v3.2. Track as SER-FUT-WEB. Phase 99 can revisit.

2. **Hot-swap vs mount-only attach for SerializeAddon?**
   - What we know: SerializeAddon has no rendering side effects (it doesn't allocate canvases, attach DOM, register parsers, or hook PTY events). It's a "library function" that walks the buffer when invoked.
   - Options: (a) Hot-swap arm — addon attaches/detaches with toggle; mirror of WebGL/Clipboard. (b) Mount-only — attached at session init, no detach until tab close.
   - Recommendation: **(a) Hot-swap.** Toggling Serialize OFF should release the addon's dispose-able resources immediately (it's almost nothing — but consistency with the other hot-swap arms is the defensible default). The italic caption is the SECRETS warning, NOT a "next-session-only" caption.

3. **Should the Save menu item be hidden when Serialize is OFF?**
   - What we know: Hidden = simpler menu, but discoverability suffers ("where's save?"). Always-shown + toast = consistent menu, slight noise on click-when-disabled.
   - Recommendation: Always show the menu item. Click-when-disabled shows a one-shot BannerStack info toast `"Enable the Serialize plugin in Settings to save sessions"`. Mirrors Phase 87 capability-token error UX.

4. **Where does ANSI strip happen — frontend TS or backend Go?**
   - Recommendation: Frontend (`frontend/src/lib/stripAnsi.ts`). Reasons in §"Claude's Discretion / Where ANSI stripping happens."

5. **Default filename includes a timestamp?**
   - Recommendation: Yes, `<sanitized-name>-YYYY-MM-DD-HHmmss.txt`. Reasons in §"Claude's Discretion / Default filename format."

6. **SaveFileDialog Title contains the secrets warning?**
   - Recommendation: Yes, `"Save Terminal As… (file will include any printed secrets)"`. Reasons in §"Claude's Discretion."

7. **Do we need a SerializeConfig nested struct (mirror of ImageConfig/WebLinksConfig)?**
   - What we know: PluginSettings.Serialize is a bare bool; no per-plugin runtime config exists for v3.2.
   - Recommendation: NO — Phase 97 ships only the bool. If v3.3+ adds Advanced options (excludeAltBuffer, format, default-dir), they'd ship as a new SerializeConfig nested struct using the Phase 95/96 sub-key RPC pattern. Phase 97 explicitly does not pre-build that infrastructure.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node / pnpm | Frontend dep install + vitest | ✓ | pnpm 9.x | — |
| Go 1.22+ | Daemon + app.go + tests | ✓ | go.mod project requirement | — |
| Wails v2.12.0 | `runtime.SaveFileDialog` | ✓ | go.mod | — |
| `@xterm/addon-serialize@0.14.0` | This phase | NOT YET INSTALLED | 0.14.0 (latest) | Plan task installs it |
| chafa (for IMG-related tests) | Phase 96 only | ✓ optional | — | n/a for Phase 97 |
| macOS signing cert | Final UAT (signed build) | ✓ | per project memory `reference_macos_signing_cert.md` | — for code-only verification |

**Missing dependencies with no fallback:** None.

**Missing dependencies that the phase must install:** `@xterm/addon-serialize@0.14.0` (handled by the first plan task; mirror of Phase 96 IMG-01 promotion).

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework (Go) | `testing` (stdlib) — `internal/release/no_autosave_test.go`, `internal/webserver/vendor_drift_test.go`, `internal/daemon/plugin_settings_test.go` (already present, needs no changes for SER) |
| Framework (TS) | `vitest` 1.x — `frontend/src/components/__tests__/`, `frontend/src/lib/__tests__/` |
| Config files | `frontend/vitest.config.ts`, `frontend/tsconfig.json`, root `go.mod` |
| Quick run command (Go) | `go test ./internal/... -count=1 -short` |
| Quick run command (TS) | `cd frontend && pnpm test --run` (vitest) |
| Full suite (Go) | `go test ./... -count=1 -race` |
| Full suite (TS) | `cd frontend && pnpm test --run && pnpm tsc --noEmit && pnpm lint` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SER-01 | TabBar context menu has "Save Terminal As…" item | unit | `cd frontend && pnpm test src/components/__tests__/TabBar.test.tsx` | EXTEND existing |
| SER-01 | TerminalPanel hot-swap arm: addon attaches/detaches with `pluginConfig?.serialize` | unit (source-inspection) | `cd frontend && pnpm test src/components/__tests__/TerminalPanel.test.tsx` | EXTEND existing |
| SER-01 | App.tsx saver-registry round-trip (register → invoke → unregister) | unit | `cd frontend && pnpm test src/__tests__/App.saver.test.tsx` | NEW (Wave 0) |
| SER-01 | `(*App).SaveTerminalSession` calls SaveFileDialog and writes file | unit / integration | `go test ./. -run TestApp_SaveTerminalSession -count=1` | NEW (Wave 0) |
| SER-01 | `stripAnsi()` correctly strips SGR/CUF/CUB/CUU/CUD/ECH/DEC modes | unit | `cd frontend && pnpm test src/lib/__tests__/stripAnsi.test.ts` | NEW (Wave 0) |
| SER-01 | `sanitizeFilename()` handles path-traversal + Windows reserved + empty | unit | `cd frontend && pnpm test src/lib/__tests__/sanitizeFilename.test.ts` | NEW (Wave 0) |
| SER-02 | PluginsSection shows the verbatim secrets-warning caption under Serialize toggle | unit (source-inspection) | `cd frontend && pnpm test src/components/__tests__/PluginsSection.test.tsx` | EXTEND existing |
| SER-03 | No `setInterval`/`setTimeout`/`BeforeQuit`/`OnShutdown` calls `serialize()` anywhere | static (regex grep) | `go test ./internal/release/... -run TestSER03_NoAutoSavePatterns -count=1` | NEW (Wave 0) |
| SER-03 | PluginSettings has no `autoSave|autoExport|autoCapture|saveOnQuit` field | static (regex grep) | (covered by TestSER03_NoAutoSavePatterns above) | NEW (Wave 0) |
| SER-03 | Only `(*App).SaveTerminalSession` matches `(?i)save.*(session|terminal|scrollback)` in app.go | static | (covered) | NEW (Wave 0) |
| WEB-01 (vendoring) | `web/vendor/xterm/addons/addon-serialize.js` exists; VERSION lists it; min-count == 9 | unit | `go test ./internal/webserver/... -run TestXtermVendorVersionsMatchPnpmLock -count=1` | EXTEND existing |
| Integrated | Manual UAT — real Wails build, real Save dialog, real saved file inspection | manual | `wails build -tags wailsassets && open build/bin/AgentHub.app` then `.planning/phases/97-.../97-HUMAN-UAT.md` checklist | NEW (last plan) |

### Sampling Rate

- **Per task commit:** `cd frontend && pnpm test --run` (vitest) + `go test ./internal/{daemon,webserver,release}/... -count=1` (focused Go)
- **Per wave merge:** `go test ./... -count=1` + `cd frontend && pnpm test --run && pnpm tsc --noEmit && pnpm lint`
- **Phase gate:** Full suite green + `wails build -tags wailsassets` succeeds + 97-HUMAN-UAT.md checklist signed off before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `frontend/src/lib/stripAnsi.ts` — new helper (~10 lines) + tests
- [ ] `frontend/src/lib/sanitizeFilename.ts` — new helper (~15 lines) + tests
- [ ] `frontend/src/lib/__tests__/stripAnsi.test.ts` — covers SGR/CUF/CUB/CUU/CUD/ECH/DEC modes
- [ ] `frontend/src/lib/__tests__/sanitizeFilename.test.ts` — covers traversal, reserved names, empty, leading-dot
- [ ] `frontend/src/__tests__/App.saver.test.tsx` — saver-registry register/invoke/unregister round-trip
- [ ] `internal/release/no_autosave_test.go` — SER-03 negative-grep test
- [ ] `internal/release/no_autosave_test.go` companion — `TestSER03_NoAutoSettingsField` asserting PluginSettings has no autoSave field
- [ ] `internal/daemon/plugin_settings_test.go` — extend `TestDefaultPluginSettings` to assert `Serialize == true` (already true; lock the assertion explicitly)
- [ ] `app_save_terminal_test.go` (new, in repo root next to `app.go`) — table-driven tests for `SaveTerminalSession` covering: cancel (empty path), normal write, write to non-existent dir error, dialog setup error
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — assert "Save Terminal As…" menu item is present and calls onRequestSave with tabId
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — assert hot-swap addon attach + dep-array contains `pluginConfig?.serialize` + cleanup unregisters saver
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — assert SER-02 caption present verbatim
- [ ] `internal/webserver/vendor_drift_test.go` — bump min-count from 8 to 9 + update error message to mention addon-serialize
- [ ] `.planning/phases/97-.../97-HUMAN-UAT.md` — checklist for: open new session → toggle Serialize OFF → right-click → "Save…" → see toast (no dialog); toggle ON → repeat → dialog appears → save to ~/Desktop → verify file contents are plain text (no escape sequences) and contain expected scrollback; cancel test (clicks Cancel → no file written)

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture | yes | Threat-model the new write-to-disk path; document trust boundaries (untrusted PTY output → user-confirmed-path → user-owned file) |
| V2 Authentication | no | No auth surface added |
| V3 Session Management | no | No session token added |
| V4 Access Control | yes (filesystem) | OS file permissions on `os.WriteFile(path, []byte, 0o644)` — owner read/write, others read-only. Path is user-chosen via native dialog (OS enforces user can write to the destination). |
| V5 Input Validation | yes | Frontend `sanitizeFilename` strips path-traversal + reserved Windows names from DefaultFilename before passing to dialog (defense-in-depth; the OS dialog itself prevents path-separator characters in filenames) |
| V6 Cryptography | no | No crypto operations |
| V7 Error Handling | yes | Cancel = silent success (no error toast); IO errors surface via App-level BannerStack with the system error message |
| V8 Data Protection | yes | SER-02 secrets warning is the user-facing data-protection control; user is informed before saving that secrets/tokens may be in the output |
| V9 Communication | n/a | All-local; no network IO in this phase |
| V10 Malicious Code | yes | Negative-grep regression tests (SER-03) prevent reintroduction of auto-save patterns; supply-chain controlled via vendored UMD + drift test |
| V11 Business Logic | yes | "No on-disk capture without explicit user gesture" is the load-bearing business rule; SER-03 enforces |
| V12 File and Resource | yes | File path is user-chosen (no app-determined location); content is user-derived from their own terminal buffer |
| V13 API and Web Service | n/a | No new HTTP routes |
| V14 Configuration | yes | No new env vars or config flags; PluginSettings.Serialize is the only knob and is bool-only |

### Known Threat Patterns for {Wails desktop + xterm.js + Go save-to-disk}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via session name in DefaultFilename | Tampering | `sanitizeFilename` strips `/`, `\`, `..` etc. before the dialog opens. The OS native dialog provides a second layer of defense (it disallows the user from clicking Save with `/` in the filename). |
| Symlink TOCTOU (user picks a path that's a symlink to a sensitive file) | Tampering | Out of scope — the user chose the path explicitly; the OS Save dialog gives a "Replace?" prompt for existing files. We trust the OS dialog's overwrite-confirmation. |
| Auto-saving secrets without user knowledge | Information Disclosure | SER-03 negative-grep regression test prevents auto-save reintroduction |
| Saved file readable by other users on a shared system | Information Disclosure | `os.WriteFile(path, []byte, 0o644)` — `0o644` = owner rw + others r. **Tradeoff:** `0o600` would be tighter (owner-only) but is non-default and surprising for "I just saved a text file." Recommendation: ship `0o644` as the friendly default; if a user reports concern, Phase 99 / SER-FUT can add a "private files" toggle for `0o600`. |
| TempFile leak (writing to a temp file that's accessible) | Information Disclosure | We do NOT write to a temp file; `os.WriteFile` is direct write to the user-chosen path |
| Memory pressure from giant scrollback save (10K rows × 80 cols × ~5 bytes = ~4 MB) | Denial of Service | Acceptable; a 4 MB string in JS heap and a 4 MB file write are both within normal app behavior. Document. If a real user reports a freeze, Phase 99 / SER-FUT can add streaming via a chunked Wails RPC. |
| User accidentally saving over an existing important file | Tampering (user-induced) | Native Save dialog provides "Replace?" confirmation prompt. Trust the OS. |
| Race between Save dialog open + tab close | Confused state | App.tsx saver-registry returns null if the session has been unregistered (tab closed mid-dialog); the App-level handler shows a "could not save: terminal closed" banner. Pitfall #6 + #7 prevention. |

---

## Sources

### Primary (HIGH confidence)
- npm registry `@xterm/addon-serialize@0.14.0` — package.json (main, types, deps), versions, file list. Verified 2026-05-07 via `pnpm view`.
- `/tmp/serialize-inspect/package/typings/addon-serialize.d.ts` — full TypeScript type signatures
- `/tmp/serialize-inspect/package/src/SerializeAddon.ts` — full upstream source (695 lines), including the `_serializeString()` method that emits ANSI escapes
- `/tmp/serialize-inspect/package/lib/addon-serialize.{js,mjs}` — production bundles, audited for CSP-relevant patterns
- `/Users/ken/go/pkg/mod/github.com/wailsapp/wails/v2@v2.12.0/pkg/runtime/dialog.go` — `SaveFileDialog`, `SaveDialogOptions`, cancellation contract
- `/Users/ken/dev/agenthub/app.go:803-829` — existing `OpenDirectoryDialog`/`OpenFileDialog` pattern (cancellation handling, default-dir fallback to home)
- `/Users/ken/dev/agenthub/frontend/src/components/TabBar.tsx` — existing context-menu infrastructure (right-click handler, ESC dismiss, outside-click dismiss, role=menu pattern)
- `/Users/ken/dev/agenthub/frontend/src/components/TerminalPanel.tsx` — hot-swap useEffect pattern (lines 329-495)
- `/Users/ken/dev/agenthub/frontend/src/components/PluginsSection.tsx` — italic caption pattern (lines 107-111)
- `/Users/ken/dev/agenthub/internal/daemon/plugin_settings.go` — `Serialize bool` field at line 81; default `true` at line 109
- `/Users/ken/dev/agenthub/internal/webserver/vendor_drift_test.go` — generalized regex + min-count guard
- `/Users/ken/dev/agenthub/.planning/phases/96-image-addon-csp-audit/96-RESEARCH.md` — Phase 96 RESEARCH structure (template for this document)
- `/Users/ken/dev/agenthub/.planning/phases/96-image-addon-csp-audit/96-01-SUMMARY.md` — Phase 96 Plan 01 summary (Wave 0 scaffolds pattern reference)
- `/Users/ken/dev/agenthub/.planning/REQUIREMENTS.md` — SER-01, SER-02, SER-03 (lines 70-72) + SER-FUT-01 (line 112)
- `/Users/ken/dev/agenthub/.planning/ROADMAP.md` — Phase 97 goal + 3 success criteria (lines 410-416)
- `/Users/ken/dev/agenthub/.planning/STATE.md` — locked decisions + Phase 92 hand-edit pin pattern + Phase 94/95 sub-key RPC pattern + Phase 96 italic caption pattern

### Secondary (MEDIUM confidence)
- xterm.js GitHub addon-serialize README — installation + usage example (verified by direct fetch)
- Wails v2 docs (https://wails.io/docs/reference/runtime/dialog) — `SaveFileDialog` API documentation matching the source

### Tertiary (LOW confidence)
- (none — every claim above has at least a primary source)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified via npm registry + direct source-read of upstream tarball
- Architecture: HIGH — every pattern has an existing-code precedent (Phase 92-96)
- Pitfalls: HIGH — derived from direct source-read of the addon, the Wails dialog API, and the existing TabBar/TerminalPanel/PluginsSection code
- ANSI strip regex: HIGH — verified by full source-read of all 695 lines of SerializeAddon.ts; covers the entire emit vocabulary; `excludeModes: true` neutralizes the only edge case
- Web parity scope: MEDIUM — relies on assumption A6 (web parity acceptable to defer); planner should confirm with user if there's any doubt
- SER-03 regression test: HIGH — Go regex-walk pattern is well-established (Phase 88 `OriginPatterns: ["*"]` guard); minimal risk of false positives or negatives

**Research date:** 2026-05-07
**Valid until:** 2026-06-06 (30 days for stable upstream addon-serialize@0.14.0; Wails v2.12.0 is stable; xterm.js 6.0.0 is stable)
