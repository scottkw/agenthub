---
phase: 97-serialize-addon-save-session-ux
plan: "06"
subsystem: web
status: pending-uat
tags: [phase-97, serialize, web-parity, uat, wave-3]

requires:
  - phase: 97-01
    provides: vendored addon-serialize.js at web/vendor/xterm/addons/addon-serialize.js; vendor_drift_test.go min-count=9
  - phase: 97-02
    provides: stripAnsi + sanitizeFilename helpers (desktop-side; web uses addon's own serialize() output)
  - phase: 97-03
    provides: App.tsx saver-registry pipeline
  - phase: 97-04
    provides: TerminalPanel hot-swap arm + TabBar "Save Terminal As..." menu item
  - phase: 97-05
    provides: (*App).SaveTerminalSession Wails RPC + SER-02 verbatim caption in PluginsSection

provides:
  - "web/embed.go //go:embed includes vendor/xterm/addons/addon-serialize.js (file served at /assets/xterm/addons/addon-serialize.js)"
  - "web/terminal.html <script> tag for addon-serialize.js inserted after addon-image and before terminal.js (UMD load order load-bearing)"
  - "web/assets/terminal.js initTerminal() constructs SerializeAddon via UMD global gated by pluginConfig.serialize (vendoring-discipline parity; no web save UI in v3.2)"
  - "97-HUMAN-UAT.md with 4 manual UAT scenarios: SER-OFF banner, SER-ON plain-text file, CANCEL silence, SER-02 caption visibility"

affects: [phase-98, gsd-verify-work-97, SER-FUT-WEB]

tech-stack:
  added: []
  patterns:
    - "Web parity: every @xterm/addon-* in pnpm-lock gets //go:embed entry + <script> tag + term.loadAddon() call even when no web UI consumer exists in the current release"
    - "UMD global namespace shape for @xterm/addon-serialize: window.SerializeAddon is a namespace object; class is SerializeAddon.SerializeAddon (mirrors ImageAddon.ImageAddon pattern)"
    - "Vendoring-discipline parity construction: try/catch silent swallowing per existing addon-image/addon-clipboard precedent"

key-files:
  created:
    - .planning/phases/97-serialize-addon-save-session-ux/97-HUMAN-UAT.md
  modified:
    - web/embed.go
    - web/terminal.html
    - web/assets/terminal.js

key-decisions:
  - "Web save UI (SER-FUT-WEB) deferred: Phase 97 ships desktop-only Save Terminal As... via Wails GUI right-click context menu; web client loads addon for parity but has no save affordance in v3.2 per 97-RESEARCH locked scope decision"
  - "UMD global shape new SerializeAddon.SerializeAddon() confirmed from vendored bundle IIFE preamble (globalThis.SerializeAddon assigned as namespace object with .SerializeAddon class property)"
  - "Form 1 embed.go append: addon-serialize.js appended to same //go:embed line as addon-image.js (combined 85 chars, under 120 limit)"

patterns-established:
  - "UAT runbook authoring: 4 scenarios per VALIDATION.md Manual-Only Verifications table, each with Why manual / Setup / Verify / Sign-off sections"

requirements-completed: [SER-01, SER-02, SER-03]

duration: 15min
completed: 2026-05-07
---

# Phase 97, Plan 06: Web Parity + UAT Runbook Summary

**SerializeAddon vendored, embedded in embed.FS, script-tagged in terminal.html, and loaded in initTerminal() for vendoring-discipline parity; 97-HUMAN-UAT.md authored with 4 SER-01/SER-02 scenarios; human UAT checkpoint pending sign-off**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-05-07
- **Completed:** 2026-05-07 (Tasks 1-2 done; Task 3 = human checkpoint awaiting sign-off)
- **Tasks:** 2 of 3 automated tasks complete (Task 3 is human-verify checkpoint)
- **Files modified:** 4 (web/embed.go, web/terminal.html, web/assets/terminal.js, 97-HUMAN-UAT.md)

## Accomplishments

- Extended `web/embed.go` to include `vendor/xterm/addons/addon-serialize.js` in the `//go:embed` directive (Form 1: appended to addon-image.js line; combined length 85 chars)
- Added `<script src="/assets/xterm/addons/addon-serialize.js"></script>` to `web/terminal.html` after addon-image and before terminal.js — UMD load order load-bearing (terminal.js's `initTerminal()` consumes the `window.SerializeAddon` global)
- Added SerializeAddon construction block in `web/assets/terminal.js` `initTerminal()` after ImageAddon block; gated by `pluginConfig.serialize` (already `true` in defaults); UMD global shape verified as namespace object (`new SerializeAddon.SerializeAddon()`); try/catch silent per existing pattern; detailed scope comment citing 97-RESEARCH locked decision
- Authored `97-HUMAN-UAT.md` (124 lines) with 4 manual UAT scenarios, wails build instruction, final sign-off section with platform checkboxes, tester/date/build fields, and web parity scope note

## Task Commits

Each task was committed atomically:

1. **Task 1: Web parity — embed.go + terminal.html + initTerminal SerializeAddon construction** - `784de59` (feat)
2. **Task 2: Author 97-HUMAN-UAT.md with 4 manual test scenarios** - `19bf732` (docs)
3. **Task 3: Human UAT sign-off** - awaiting human checkpoint

## Files Created/Modified

- `web/embed.go` - Added `vendor/xterm/addons/addon-serialize.js` to `//go:embed` directive; `go build ./web/...` and `go test ./internal/webserver/...` exit 0
- `web/terminal.html` - Inserted `<script src="/assets/xterm/addons/addon-serialize.js"></script>` after addon-image.js and before terminal.js
- `web/assets/terminal.js` - Added SerializeAddon construction block after ImageAddon block in `initTerminal()` with vendoring-discipline parity scope comment
- `.planning/phases/97-serialize-addon-save-session-ux/97-HUMAN-UAT.md` - 4 manual UAT scenarios per VALIDATION.md Manual-Only Verifications

## Decisions Made

- **Web save UI deferred to SER-FUT-WEB:** Phase 97 ships desktop-only "Save Terminal As..." via Wails right-click context menu. The web client loads the addon for vendoring-discipline parity (vendor_drift_test.go contract: every @xterm/addon-* in pnpm-lock must have a term.loadAddon() call) but has no save affordance. A future SER-FUT-WEB plan may add a `<a download>` blob URL approach.
- **UMD global shape:** `new SerializeAddon.SerializeAddon()` confirmed by inspecting the vendored bundle's IIFE preamble — `t.SerializeAddon=e()` where the returned object has `.SerializeAddon` as the class. Mirrors `ImageAddon.ImageAddon` pattern exactly.

## Deviations from Plan

None — plan executed exactly as written. The `serialize: true` default was already present in terminal.js (line 121 per plan's note), so C1 was a no-op. `go build ./web/...` and `go test ./internal/webserver/...` exited 0 on first attempt.

Pre-existing unrelated failure: `go test ./...` fails on the `security-review` package due to a package collision between two test files with different `package` declarations (`relay` vs `webserver`). This failure exists on `main` before this plan and is unrelated to the web parity changes.

## Issues Encountered

None — all verification checks passed on first attempt.

## Known Stubs

None — this plan is vendoring-discipline parity only. The web-side addon is loaded but intentionally has no UI consumer in v3.2. This is the locked scope decision, documented in the SUMMARY and in 97-HUMAN-UAT.md. The SER-FUT-WEB tracking item carries the future web save UI work.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes introduced. The addon-serialize.js UMD file is a pure-JS static asset served from the existing `web/embed.FS` route (same-origin, CSP `script-src 'self'` compliant, no WebAssembly/Worker/blob constructs per 97-RESEARCH §"Mandatory Pre-Phase CSP Audit").

## Next Phase Readiness

- Phase 97 automated test gates: all green (go build ./web/..., go test ./internal/webserver/..., go test ./internal/daemon/..., go test ./internal/release/...)
- Human UAT checkpoint (Task 3) is the final gate — 97-HUMAN-UAT.md is authored and ready for the tester
- After sign-off: `/gsd-verify-work 97` can flip SER-01..SER-03 success criteria to GREEN

---
*Phase: 97-serialize-addon-save-session-ux*
*Completed: 2026-05-07 (Tasks 1-2; Task 3 pending UAT)*
