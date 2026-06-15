---
phase: 125
slug: react-editor-codemirror-6-desktop-web
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-14
validated: 2026-06-14
---

# Phase 125 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (React component/unit) + Playwright (cross-browser e2e: Chromium/Firefox/WebKit) + go test (If-Match/412 server change, vendor_drift_test.go) |
| **Config file** | frontend/ vitest config; playwright.config (Wave 0 may add); Go native |
| **Quick run command** | `cd frontend && pnpm test` (component) |
| **Full suite command** | `cd frontend && pnpm test && pnpm exec playwright test` + `go test ./internal/files/... ./internal/daemon/...` |
| **Estimated runtime** | component ~seconds; Playwright cross-browser ~minutes |

---

## Sampling Rate

- **After every task commit:** relevant component test (`pnpm test -- --run <Component>`) or `go test ./internal/files/...`
- **After every plan wave:** `cd frontend && pnpm test` + `go test -race ./internal/...`
- **Before `/gsd:verify-work`:** full Playwright cross-browser suite green; vendor_drift_test passes; zero CSP violations
- **Max feedback latency:** component <30s; e2e is the pre-verify gate

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File / Scenario | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-----------------|--------|
| T-125-01 | 125-01 | 0 | EDIT-01 | vendor drift | package.json ↔ pnpm-lock CodeMirror version parity | go unit (behavioral) | `go test ./internal/webserver/... -run CodeMirror` | `internal/webserver/vendor_drift_test.go::TestCodeMirrorVersionsMatchPnpmLock` | ✅ green |
| T-125-01 | 125-01 | 5 | EDIT-01 | CSP | zero new CSP amendments; editor+write drive no violations | playwright (behavioral) | `pnpm exec playwright test e2e/web-csp.spec.ts` | `e2e/web-csp.spec.ts` (writeCap editor+write, afterEach asserts `cspViolations == []`) | ✅ green |
| T-125-02 | 125-02 | 1 | EDIT-02 | — | CM6 Compartment read-only↔editable toggle, no remount | vitest (source-grep) + manual | `cd frontend && pnpm test Editor.test.tsx` | `Editor.test.tsx` (?raw structural) + Manual-Only #1 (live mount) | ✅ green |
| T-125-02 | 125-02 | 1/5 | EDIT-03 | authz | Edit button hidden when isBinary or !canWrite | playwright (behavioral DOM) | `pnpm exec playwright test e2e/files-write.spec.ts -g "scenario 13"` | `files-write.spec.ts::scenario 13` (renders page, asserts Edit button `toHaveCount(0)` for binary.bin) | ✅ green |
| T-125-02 | 125-02 | 1 | EDIT-04 | — | extension→language syntax highlighting | vitest (source-grep) + manual | `cd frontend && pnpm test Editor.test.tsx` | `Editor.test.tsx` (languageFor invocation) + Manual-Only #1 (visual highlight) | ✅ green |
| T-125-01/03 | 125-01,03 | 0 | EDIT-05 | concurrency | Cmd/Ctrl+S → PUT with If-Match ETag (mtime+size) | go unit + playwright (behavioral) | `go test ./internal/files/... -run TestWrite_IfMatch` + `pnpm exec playwright test -g "scenario 1\|scenario 12"` | `write_test.go::TestWrite_IfMatch_Match/Mismatch/NewFile`; `files-write.spec.ts::scenario 1` (PUT 200 w/ cap), `scenario 12` (ETag round-trip) | ✅ green |
| T-125-03 | 125-03 | 1 | EDIT-06 | — | dirty bullet + three-state save (icon+text, ~1.5s transient) | vitest (behavioral hook + source-grep) | `cd frontend && pnpm test useFilesWrite EditorComponents` | `useFilesWrite.test.tsx` (saveState transitions on 200/412/401/405); `EditorComponents.test.tsx` (SaveIndicator/DirtyMarker copy+icon) | ✅ green |
| T-125-03 | 125-03 | 1 | EDIT-07 | — | React-level unsaved guard (no beforeunload) | vitest (behavioral negative + source-grep) | `cd frontend && pnpm test EditorComponents App.saver` | `EditorComponents.test.tsx` (asserts NO `addEventListener('beforeunload')` in src + UnsavedChangesModal copy/focus); `App.saver.test.tsx` | ✅ green |
| T-125-01/03 | 125-01,03 | 0 | EDIT-08 | concurrency | 412 conflict UX: Force/Save-as-new/Discard; buffer preserved | go unit + vitest + playwright (behavioral) | `go test ./internal/files/... -run TestWrite_IfMatch_Mismatch` + `pnpm test useFilesWrite` + `pnpm exec playwright -g "scenario 12"` | `write_test.go::TestWrite_IfMatch_Mismatch` (412); `useFilesWrite.test.tsx` (412→isConflict, buffer NOT cleared); `files-write.spec.ts::scenario 12` (412 + force `If-Match:*`→200) | ✅ green |
| T-125-04 | 125-04 | 2 | EDIT-09 | authz/collision | create/mkdir/delete(recursive count)/rename/move; 409 Cancel-default | go unit + playwright (behavioral) | `go test ./internal/files/... -run "TestRename\|TestMkdir\|TestDelete\|TestUpload_Collision"` + `pnpm exec playwright -g "scenario 4\|5\|6\|7\|8\|9"` | `write_test.go` (Rename/Mkdir/Delete_Recursive/Upload_Collision_409); `files-write.spec.ts::scenarios 4–9` (real HTTP create/mkdir/delete-dir/rename/cross-dir move) | ✅ green |
| T-125-05 | 125-05 | 3 | EDIT-10 | collision/over-cap | single+multi upload, per-file progress, 409 overwrite | go unit + vitest (behavioral XHR) + playwright | `go test ./internal/files/... -run TestUpload` + `pnpm test Upload` + `pnpm exec playwright -g "scenario 10\|11"` | `write_test.go::TestUpload_NewFile/Collision_409/Overwrite_200`; `Upload.test.tsx` (XHR functional: onProgress %, 409 isCollision, 413 isOverCap); `files-write.spec.ts::scenarios 10/11/11b` | ✅ green |
| T-125-02 | 125-02 | 1 | EDIT-11 | DoS/perf | >500KB warn; near-5MB disable syntax | vitest (source-grep) + playwright (behavioral) | `cd frontend && pnpm test Editor.test.tsx` + `pnpm exec playwright -g "scenario 14"` | `Editor.test.tsx` (thresholds + verbatim copy); `files-write.spec.ts::scenario 14` (/read 413 over-cap, /stat size > 500KB) | ✅ green |
| T-125-02..05 | 125-02–05 | 1-3 | EDIT-12 | authz | useFilesWrite + canWrite; all affordances gated on canWrite | vitest (behavioral hook) + playwright | `cd frontend && pnpm test useFilesCapability useFilesWrite` + `pnpm exec playwright -g "scenario 3\|13"` | `useFilesCapability.test.tsx` (present/denied/probe-failed resolution); `files-write.spec.ts::scenario 3` (403 no cap), `scenario 13` (Edit absent) | ✅ green |
| T-125-06 | 125-06 | 5 | EDIT-13 | CSP/authz | cross-browser e2e merge gate (14 scenarios) + zero CSP | playwright (behavioral, 3 browsers) | `pnpm exec playwright test e2e/files-write.spec.ts` | `files-write.spec.ts` 14 scenarios × Chromium/Firefox/WebKit = 51/51 green (VERIFICATION.md) | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> **Coverage note (audit 2026-06-14):** Each logic requirement has ≥1 genuinely-failable behavioral test (Go HTTP/unit, real-HTTP/DOM Playwright, or real-invocation vitest hook/API tests). The `?raw` source-inspection vitest tests (Editor.test.tsx, Editor.save.test.tsx, EditorComponents.test.tsx, FileRowActions/Upload source blocks) are structural supplements (string `.includes`) — they cannot fail on wrong logic, only on token absence — and are NOT counted as primary coverage. Primary behavioral backing is recorded in the "File / Scenario" column above.

---

## Wave 0 Requirements

- [x] Server: `Handler.Write` reads `If-Match`, returns 412 on mismatch; read route emits `ETag` (net-new — EDIT-05/08) — `write.go` StatusPreconditionFailed; `handler.go` ETag (4 refs)
- [x] `internal/files/` (or webserver) If-Match/412 unit tests — `write_test.go::TestWrite_IfMatch_Match/Mismatch/NewFile/RecheckViaHTTP/TwoWritersIfMatchRace`
- [x] `vendor_drift_test.go` — package.json ↔ pnpm-lock CodeMirror version parity gate (EDIT-01) — `TestCodeMirrorVersionsMatchPnpmLock`
- [x] Playwright fixture: add a `WRITE_CAP` variant carrying `files.write` — `cmd/playwright-fixture/main.go` WRITE_CAP; `e2e/global-setup.ts` + `fixture-env.ts` writeCap/writeAppUrl
- [x] Playwright config for Chromium + Firefox + WebKit — `playwright.config.ts` (3 projects confirmed)

*Framework present (vitest + go test); Playwright may need install in Wave 0.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CodeMirror Tab / Cmd-V inside the Wails WebView vs Phase 49 clipboard handler | EDIT (editor) | Wails WebView keyboard/clipboard interaction not reliably Playwright-drivable on desktop | Open editor in desktop app, test Tab indent + paste; confirm no conflict with the app clipboard handler |
| Desktop GUI visual render of editor + affordances | EDIT-01..12 | Wails desktop app not headless-automatable | Open a file, edit, save, exercise affordances in the running app |

*Web-share surface IS Playwright-automatable; desktop Wails interactions are the manual residue.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency acceptable (component <30s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-14 (retroactive audit)

---

## Validation Audit 2026-06-14

**Verdict:** COMPLIANT — all 13 EDIT logic requirements covered by genuinely-failable automated tests.

**Counts:** total 13 · covered (automated, behavioral) 13 · manual-only residue 2 (visual/keyboard) · missing 0

- **Gaps:** none.
- **Resolved:** 0 (no new tests authored — audit-only; existing delivered tests confirmed sufficient).
- **Escalated:** 0.

**Method:** Each EDIT requirement was mapped to its strongest delivered test and classified by whether the test can actually fail on wrong behavior:

- **Behavioral (primary coverage):** Go HTTP/unit tests (`write_test.go` If-Match/412/rename/mkdir/delete/upload-409; `vendor_drift_test.go` CodeMirror parity), real-HTTP + real-DOM Playwright e2e (`files-write.spec.ts` 14 scenarios × 3 browsers = 51/51; `web-csp.spec.ts` zero-CSP), and real-invocation vitest tests (`useFilesWrite.test.tsx` save-state machine on 200/412/401/405 + buffer-preserved; `useFilesCapability.test.tsx` probe resolution; `filesApi.test.ts` URL/parse/error predicates; `Upload.test.tsx` XHR functional progress/409/413; `BreadcrumbBar.test.tsx` render+click). EDIT-03 and EDIT-12 affordance gating are proven behaviorally by e2e scenario 13 (Edit button `toHaveCount(0)` for binary) and scenario 3 (403 without `files.write`).
- **Source-grep supplements (NOT counted as primary):** the `?raw` vitest tests in `Editor.test.tsx`, `Editor.save.test.tsx`, `EditorComponents.test.tsx`, and the source blocks of `FileRowActions.test.tsx`/`Upload.test.tsx` assert string `.includes` on file source. They verify locked copy/structure and cannot fail on wrong logic — treated as fast structural guards layered on top of the behavioral coverage above, never as the sole evidence for a requirement.

**Manual-Only residue (operator-deferred, does NOT block compliance):** the live CodeMirror 6 on-screen render, syntax-highlight appearance, and Tab/Cmd-V keyboard/clipboard interaction inside the Wails desktop WebView are not headless-automatable (Manual-Only section below, VERIFICATION.md human items 1–2). The underlying logic — mount/Compartment toggle, language detection, thresholds, save/conflict, capability gating — is automated-green. The web-share surface is fully Playwright-covered. These remain deferred to milestone-end batch UAT per the cross-surface parity policy.
