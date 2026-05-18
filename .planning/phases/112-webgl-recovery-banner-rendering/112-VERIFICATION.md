# Phase 112 — WebGL Recovery Banner Rendering: Verification

**Phase:** 112 (webgl-recovery-banner-rendering)
**Plan:** 01
**Date:** 2026-05-18
**Requirements:** UI-01, UI-02
**Source:** [112-RESEARCH.md](./112-RESEARCH.md) — root cause (call ordering inside `onContextLoss`)
**Context:** [112-CONTEXT.md](./112-CONTEXT.md) — original hypothesis (closure rot) REFUTED by research
**GitHub Issue:** [#55](https://github.com/scottkw/agenthub/issues/55) — WebGL recovery banner does not render

---

## 1. Test-Environment Snapshot

| Item              | Value                                                  |
| ----------------- | ------------------------------------------------------ |
| OS                | macOS 26.5 (Build 25F71)                               |
| Node              | v24.14.1                                               |
| Package Manager   | pnpm 9.15.9                                            |
| Wails             | v2.10.2 (`/Users/ken/go/bin/wails`)                    |
| Frontend SHA      | `a4cdc2e` (post-fix HEAD on `main`)                    |
| Vitest            | 4.1.0                                                  |
| jsdom             | 29.x                                                   |
| Chrome (web UAT)  | _executor environment has no Chrome installed — UAT-2 needs operator-supplied Chrome_ |

---

## 2. Automated Test Result Summary

### Targeted suite (Task 1 RED, Task 2 GREEN gates)

Command: `cd frontend && pnpm test -- TerminalPanel.hot-swap TerminalPanel.contextLoss`

**Pre-fix (RED):**

```
 Test Files  2 failed (2)
      Tests  4 failed | 12 passed (16)

  × Phase 112 UI-01 > onContextLoss callback invokes onWebGLContextLost(...) BEFORE webglAddon.dispose()
    AssertionError: expected 93 to be less than 15
  × Phase 112 UI-01 > onContextLoss callback uses queueMicrotask to defer the dispose work
    AssertionError: expected '\n              webglAddon.dispose()\…' to match /queueMicrotask\s*\(/
  × Phase 112 UI-01 > onContextLoss callback wraps webglAddon.dispose() in try/catch
    AssertionError: expected a `try` keyword before webglAddon.dispose()
  × TerminalPanel.contextLoss > records notify BEFORE dispose in the shared call-order timeline
    AssertionError: notify must precede dispose — Issue #55 root cause: expected 1 to be less than 0
```

Captured at commit `b889c63` (RED test commit, before fix).

**Post-fix (GREEN):**

```
 Test Files  2 passed (2)
      Tests  16 passed (16)
   Duration  1.21s
```

Captured at commit `a4cdc2e`.

### Full frontend suite

Command: `cd frontend && pnpm test`

```
 Test Files  60 passed (60)
      Tests  907 passed (907)
   Duration  16.47s
```

No regressions. All 907 tests green.

### TypeScript typecheck

Command: `cd frontend && npx tsc --noEmit`

```
(no output — clean)
```

### Lint

The frontend has no `lint` script and no top-level ESLint config; CLAUDE.md
ESLint convention is satisfied by `tsc --noEmit` + the type-aware test suite.
No new ESLint violations are possible because the diff is local to one
five-line block in TerminalPanel.tsx and the new test file follows the
established patterns of TerminalPanel.search.seedAndPersist.test.tsx /
WebGLRecoveryBanner.test.tsx.

---

## 3. UAT-1 — Desktop (Wails dev mode)

**Status:** [ ] PASS / [ ] FAIL / [x] **deferred (operator-driven)**
**Why deferred:** Wails dev mode launches an OS GUI window (a real
`webview2_loader` / `wkwebview` surface) — not driveable from this
non-interactive executor session. The fix is verified by automated tests
above; manual UAT confirms end-to-end behavior in the real WebKit shell.

### Preconditions
- `wails dev` running locally (this is the ONLY way to get DevTools on the
  desktop surface — see `project_wails_devtools_disabled_in_prod`).
- WebGL plugin enabled in Settings (default ON; verify the plugin toggle in
  Settings → Plugins).
- A shell session created in the GUI.

### Procedure

1. **Start the app**
   ```sh
   cd /Users/ken/dev/agenthub
   wails dev
   ```
   Wait for the AgentHub GUI window to open and "Vite ready" in the terminal.

2. **Create a session** in the GUI: click the `+` to create a new shell
   session on any agent / path (e.g., `bash` on `~`).

3. **Open DevTools** — right-click in the terminal area → Inspect.

4. **Trigger context loss** (DevTools Console):
   ```js
   (() => { for (const c of document.querySelectorAll('canvas')) { const gl = c.getContext('webgl2') || c.getContext('webgl'); if (gl) { const e = gl.getExtension('WEBGL_lose_context'); if (e) { e.loseContext(); return 'triggered — wait 3.5s'; } } } return 'no WebGL canvas found'; })()
   ```
   Expected return value: `'triggered — wait 3.5s'`.
   If `'no WebGL canvas found'`: WebGL plugin is OFF — enable in Settings and retry.

5. **Wait 4 seconds** (xterm's internal wait-for-restore window is 3000ms;
   the banner appears after that — see `node_modules/@xterm/addon-webgl/src/WebglRenderer.ts:114-120`).

6. **Verify banner present** (DevTools Console):
   ```js
   ({ hasBanner: !!document.querySelector('.webgl-recovery-banner'), hasStack: !!document.querySelector('.banner-stack'), text: document.querySelector('.webgl-recovery-banner__message')?.textContent })
   ```
   **Expected:**
   ```
   {
     hasBanner: true,
     hasStack: true,
     text: "Hardware-accelerated rendering recovered — your terminal is now using the standard renderer. Scrollback is intact."
   }
   ```
   Verbatim copy locked at `WebGLRecoveryBanner.tsx:42-45`.
   Per user memory `user_colorblind`: assertion is by DOM presence + text content, not color/visual eyeballing.

7. **Screenshot** the GUI window showing the banner → save to
   `screenshots/112-uat-desktop-banner.png`.

8. **Verify UI-02 (DOM fallback)** — click in the terminal area and type:
   ```sh
   echo banner-fallback-test
   ```
   Expected: output renders in the terminal (DOM renderer took over via
   `WebglAddon.dispose()` → `renderService.setRenderer(_createRenderer())`).
   Screenshot → `screenshots/112-uat-desktop-fallback.png`.

9. **Wait 8 more seconds**, then verify auto-dismiss (DevTools Console):
   ```js
   ({ hasBanner: !!document.querySelector('.webgl-recovery-banner') })
   ```
   Expected: `{ hasBanner: false }`.
   Screenshot → `screenshots/112-uat-desktop-dismissed.png`.

### Sign-off

- [ ] Step 4 returned `'triggered — wait 3.5s'`
- [ ] Step 6 returned `hasBanner: true` and exact verbatim text
- [ ] Step 7 screenshot captured: `screenshots/112-uat-desktop-banner.png`
- [ ] Step 8 `echo` output rendered (UI-02 fallback works)
- [ ] Step 8 screenshot captured: `screenshots/112-uat-desktop-fallback.png`
- [ ] Step 9 returned `hasBanner: false`
- [ ] Step 9 screenshot captured: `screenshots/112-uat-desktop-dismissed.png`

---

## 4. UAT-2 — Web (Chrome via web-share)

**Status:** [ ] PASS / [ ] FAIL / [x] **deferred (operator-driven)**
**Why deferred:** Same as UAT-1 plus this executor environment has no Chrome
installed.

### Preconditions
- `wails dev` running (or production app).
- A web-shared session URL (use the AgentHub web-share feature on a session,
  copy the URL).
- Chrome (or any modern Chromium-based browser with DevTools) opened to the
  web-share URL.

### Procedure

Repeat steps 4–9 from UAT-1, executed in Chrome connected to the web-shared
session. Same DevTools Console snippets, same expected results.

Screenshots:
- `screenshots/112-uat-web-banner.png`
- `screenshots/112-uat-web-fallback.png`
- `screenshots/112-uat-web-dismissed.png`

### Sign-off

- [ ] Step 4 returned `'triggered — wait 3.5s'` (Chrome)
- [ ] Step 6 returned `hasBanner: true` and exact verbatim text
- [ ] Step 7 screenshot captured: `screenshots/112-uat-web-banner.png`
- [ ] Step 8 `echo` output rendered (UI-02 fallback works)
- [ ] Step 8 screenshot captured: `screenshots/112-uat-web-fallback.png`
- [ ] Step 9 returned `hasBanner: false`
- [ ] Step 9 screenshot captured: `screenshots/112-uat-web-dismissed.png`

---

## 5. Regression Smoke

- [ ] No `WebglAddon` / `_renderer` console errors during the 8s banner
      window. The new `try { webglAddon.dispose() } catch { /* ignore */ }`
      should absorb any post-loss throws silently.
- [x] `cd frontend && pnpm test` — full suite GREEN (60 files / 907 tests).
- [x] `cd frontend && npx tsc --noEmit` — clean.

---

## 6. Screenshot Index

| Path                                              | Caption                                                                 |
| ------------------------------------------------- | ----------------------------------------------------------------------- |
| `screenshots/112-uat-desktop-banner.png`          | UAT-1 step 7 — banner visible in `.banner-stack`, Wails GUI window     |
| `screenshots/112-uat-desktop-fallback.png`        | UAT-1 step 8 — `echo banner-fallback-test` output via DOM renderer     |
| `screenshots/112-uat-desktop-dismissed.png`       | UAT-1 step 9 — banner removed after 8s auto-dismiss                    |
| `screenshots/112-uat-web-banner.png`              | UAT-2 step 7 — banner visible in `.banner-stack`, Chrome web-share view |
| `screenshots/112-uat-web-fallback.png`            | UAT-2 step 8 — DOM fallback in Chrome                                   |
| `screenshots/112-uat-web-dismissed.png`           | UAT-2 step 9 — banner auto-dismissed in Chrome                          |

Per project memory `feedback_dont_delete_test_artifacts_early`: KEEP all six
screenshots until the user confirms phase complete.

---

## 7. GitHub Issue #55 Status

Run before declaring UAT pass (per `feedback_check_github_issues_during_uat`):

```sh
gh issue view 55 --comments
```

- [x] No new comments since plan creation that cite repro variants this fix
      doesn't cover. **Checked 2026-05-18 — Issue #55 has 0 comments, state
      OPEN, no new symptoms reported.**
- [x] No new symptom reports that contradict the notify-before-dispose root
      cause.
- [ ] Ready to close on milestone v3.3.1 release tag (or comment with fix
      reference + commit SHA `a4cdc2e`).

---

## 8. Sign-off Block

| Requirement | Status            | Date | Verifier |
| ----------- | ----------------- | ---- | -------- |
| UI-01       | [ ] PASS [ ] FAIL |      |          |
| UI-02       | [ ] PASS [ ] FAIL |      |          |

**Executor note (2026-05-18, commit `a4cdc2e`):**
Automated gates GREEN. Manual UATs deferred to operator — the executor
session has no GUI display and no Chrome. The fix is small (24 changed
lines, single block) and the behavioral mock + source-inspection tests cover
the exact root cause (call ordering + microtask defer + try/catch) verified
in `112-RESEARCH.md` §1 / §Pattern 2 / §5. UAT confirms behavior end-to-end
in the real WebKit shell.

---

## 9. Deviations / Open Questions

- **Deviation:** UAT execution deferred. Reason: this session is
  non-interactive; `wails dev` requires a GUI display and operator-driven
  DevTools interaction. Plan task 4 (`checkpoint:human-verify`) explicitly
  allows this fallback ("If running Wails dev mode is non-trivial in this
  session, scaffold VERIFICATION.md and mark as human_needed"). No
  blocker — the automated gates above cover the source-inspection +
  behavioral invariants.
- **No code deviations** from RESEARCH §5 recommended pattern. The fix
  applied is exactly the handler reorder + queueMicrotask + try/catch
  pattern. CONTEXT's `useRef`-for-setter alternative was correctly rejected
  per RESEARCH §Pattern 1.
