---
phase: 97
slug: serialize-addon-save-session-ux
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-07
---

# Phase 97 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Frontend Framework** | Vitest (via `pnpm exec vitest run`) |
| **Go Framework** | `go test ./...` |
| **Frontend config file** | `frontend/vite.config.ts` |
| **Quick frontend run** | `pnpm exec vitest run src/components/__tests__/TabBar.test.tsx src/components/__tests__/TerminalPanel.test.tsx` |
| **Full frontend suite** | `cd frontend && pnpm test --run && pnpm tsc --noEmit` |
| **Go unit run (daemon)** | `go test ./internal/daemon/... -count=1` |
| **Go unit run (release SER-03)** | `go test ./internal/release/... -count=1` |
| **Go unit run (webserver vendor)** | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` |
| **Go full run** | `go test ./... -count=1` |
| **Wails build sanity** | `wails build -tags wailsassets` |
| **Estimated runtime** | ~45s per-task; ~3 min full |

---

## Sampling Rate

- **After every task commit:** `cd frontend && pnpm test --run` + `go test ./internal/{daemon,webserver,release}/... -count=1` + `go test . -run TestApp_SaveTerminalSession -count=1`
- **After every plan wave:** `go test ./... -count=1` + `cd frontend && pnpm test --run && pnpm tsc --noEmit`
- **Before `/gsd-verify-work`:** Full suite green; `wails build -tags wailsassets` succeeds; manual UAT signed off
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Req ID | Behavior | Test Type | Automated Command | File Exists | Status |
|--------|----------|-----------|-------------------|-------------|--------|
| SER-01 | TabBar context menu has "Save Terminal As…" item that fires onRequestSave with tabId | unit (vitest) | `pnpm exec vitest run src/components/__tests__/TabBar.test.tsx` | ✅ extend | ⬜ pending |
| SER-01 | TerminalPanel hot-swap arm: SerializeAddon attaches/detaches with `pluginConfig?.serialize` | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/TerminalPanel.test.tsx` | ✅ extend | ⬜ pending |
| SER-01 | App.tsx saver-registry: register → invoke → unregister round-trip | unit (vitest) | `pnpm exec vitest run src/__tests__/App.saver.test.tsx` | ❌ Wave 0 | ⬜ pending |
| SER-01 | `(*App).SaveTerminalSession` calls SaveFileDialog and writes file | unit (Go) | `go test . -run TestApp_SaveTerminalSession -count=1` | ❌ Wave 0 | ⬜ pending |
| SER-01 | `stripAnsi()` strips SGR/CUF/CUB/CUU/CUD/ECH/DEC modes | unit (vitest) | `pnpm exec vitest run src/lib/__tests__/stripAnsi.test.ts` | ❌ Wave 0 | ⬜ pending |
| SER-01 | `sanitizeFilename()` handles path traversal + Windows reserved + empty input | unit (vitest) | `pnpm exec vitest run src/lib/__tests__/sanitizeFilename.test.ts` | ❌ Wave 0 | ⬜ pending |
| SER-02 | PluginsSection shows verbatim secrets-warning caption under Serialize toggle | source-inspection (vitest) | `pnpm exec vitest run src/components/__tests__/PluginsSection.test.tsx` | ✅ extend | ⬜ pending |
| SER-02 | PluginSettings.Serialize defaults to `true` in defaultPluginSettings | unit (Go) | `go test ./internal/daemon/... -run TestDefaultPluginSettings -count=1` | ✅ extend | ⬜ pending |
| SER-03 | No `setInterval`/`setTimeout`/`BeforeQuit`/`OnShutdown` calls `serialize()` anywhere | static regex (Go) | `go test ./internal/release/... -run TestSER03_NoAutoSavePatterns -count=1` | ❌ Wave 0 | ⬜ pending |
| SER-03 | PluginSettings has no `autoSave\|autoExport\|autoCapture\|saveOnQuit` field | static regex (Go) | `go test ./internal/release/... -run TestSER03_NoAutoSettingsField -count=1` | ❌ Wave 0 | ⬜ pending |
| SER-03 | Only `(*App).SaveTerminalSession` matches `(?i)save.*(session\|terminal\|scrollback)` in app.go | static regex (Go) | (covered by TestSER03_NoAutoSavePatterns) | ❌ Wave 0 | ⬜ pending |
| WEB-01 | `web/vendor/xterm/addons/addon-serialize.js` exists; VERSION lists it; min-count == 9 | unit (Go) | `go test ./internal/webserver/... -run TestXtermVendorVersions -count=1` | ✅ extend | ⬜ pending |
| Integrated | Manual UAT — real Wails build, real Save dialog, real saved file inspection | manual | `wails build -tags wailsassets && open build/bin/AgentHub.app` per `97-HUMAN-UAT.md` | ❌ last plan | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/lib/stripAnsi.ts` — pure helper (~10 lines)
- [ ] `frontend/src/lib/sanitizeFilename.ts` — pure helper (~15 lines, handles traversal + Windows reserved + empty)
- [ ] `frontend/src/lib/__tests__/stripAnsi.test.ts` — covers SGR/CUF/CUB/CUU/CUD/ECH/DEC modes
- [ ] `frontend/src/lib/__tests__/sanitizeFilename.test.ts` — covers traversal, reserved names, empty, leading-dot
- [ ] `frontend/src/__tests__/App.saver.test.tsx` — saver-registry register/invoke/unregister round-trip
- [ ] `internal/release/no_autosave_test.go` (new file, package `release_test`) — SER-03 negative-grep tests for `setInterval/setTimeout/BeforeQuit/OnShutdown` + autoSave field
- [ ] `frontend/src/components/__tests__/TabBar.test.tsx` — extend with menu-item assertion + onRequestSave dispatch
- [ ] `frontend/src/components/__tests__/TerminalPanel.test.tsx` — extend with hot-swap arm + dep-array assertion + cleanup unregister assertion
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — extend with verbatim SER-02 caption assertion
- [ ] `app_save_terminal_test.go` (new, repo root next to `app.go`) — table-driven tests for `SaveTerminalSession`: cancel (empty path), normal write, IO error, dialog setup error
- [ ] `internal/daemon/plugin_settings_test.go` — extend `TestDefaultPluginSettings` to lock `Serialize == true`
- [ ] `internal/webserver/vendor_drift_test.go` — bump min-count from 8 to 9 + update error-message package list

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Right-click terminal tab → "Save Terminal As…" → native Save dialog opens | SER-01 | Native OS dialog cannot be exercised in headless CI; visual confirmation of menu UX | Build app via `wails build -tags wailsassets`, open, create session, emit some output, right-click tab, choose Save, observe native dialog |
| Saved file is plain text (no ANSI escapes) and contains expected scrollback | SER-01 | Real text inspection by human after real save flow | After save, open the resulting `.txt` in TextEdit/vim/cat and confirm no `\x1b[` sequences and content matches what was on screen |
| Cancel (Esc / click Cancel in dialog) writes no file | SER-01 | Real native dialog cancellation behavior | Right-click tab → Save → press Esc; confirm no file at default path |
| Toggle Serialize OFF → right-click tab → "Save…" → toast/banner appears, no dialog | SER-01, SER-02 | Live affordance behavior | In Settings, toggle Serialize OFF; right-click tab → Save; observe app-level toast "Save disabled — enable Serialize in Settings" or similar |
| Settings tooltip / italic caption under Serialize reads the verbatim secrets warning | SER-02 | Visual / text-rendered verification of warning copy | In Settings → Plugins, hover/inspect the Serialize row; confirm the caption "Saved files include any secrets, tokens, or sensitive data printed in the session." appears verbatim |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
