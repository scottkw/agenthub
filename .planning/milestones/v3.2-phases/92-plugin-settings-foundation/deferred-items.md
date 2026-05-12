# Phase 92 Deferred Items

Out-of-scope discoveries during phase execution that did NOT cause work to block on this phase.

## Discovered during 92-02 (Wave 2)

### Pre-existing: Sidebar.test.tsx — 20 test failures

- **File:** `frontend/src/components/__tests__/Sidebar.test.tsx`
- **Symptom:** All 20 tests in this file fail with `TypeError: Cannot read properties of undefined (reading 'unmount')` in `afterEach`. Root cause appears to be a React 19 / test setup issue with `createRoot` / `root.unmount()`.
- **Verified pre-existing:** `git stash -u && pnpm test` on `main@028f0b4` (baseline before 92-02 changes) reports the same 20 failing Sidebar tests. Not caused by 92-02.
- **Action:** Not fixed (out of scope for 92-02 per scope-boundary rule). All other 485 frontend tests pass; `tsc --noEmit` is clean for new code.

### Pre-existing: tsc warnings on `wailsjs/wailsjs/runtime/runtime` import path

- **Files:** `App.tsx`, `DaemonManagerPanel.tsx`, `SessionSharePanel.tsx`, `SettingsTab.tsx`, `UpdateBanner.tsx`, `__tests__/UpdateBanner.test.tsx`
- **Symptom:** `tsc --noEmit` prints `TS2307: Cannot find module './wailsjs/wailsjs/runtime/runtime'` (note doubled `wailsjs/wailsjs`). These imports are resolved by the project's vite alias at runtime/test time (`vite.config.ts` lines 13-16), so they don't break dev/build/test — only `tsc --noEmit` flags them. Exit code is 0.
- **Verified pre-existing:** baseline (`main@028f0b4`) shows identical warnings.
- **Action:** Not fixed (out of scope). Likely resolved when the project switches to single-`wailsjs` import paths matching the in-repo directory layout.

### Pre-existing: `security-review/` package mixed-package directory

- **File:** `security-review/{internal_relay_protocol_fuzz_test.go, internal_webserver_server_test.go}`
- **Symptom:** `go build ./...` fails with `found packages relay (...) and webserver (...) in security-review`.
- **Verified pre-existing:** documented in `92-01-SUMMARY.md` "Verification" section.
- **Action:** Not fixed (out of scope). `go build .` and `go vet .` for the root package pass cleanly.
