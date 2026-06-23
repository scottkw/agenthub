# Phase 150 Deferred Items

## Pre-existing Test Failures (out-of-scope for Phase 150)

These failures existed before Phase 150-02 started (verified by running tests against HEAD before any Phase 150-02 changes).

### 1. SettingsTab.shellPath.test.tsx — 10 tests failing

All 10 DOM-render tests in SHELL-11 were failing before Phase 150. The mock in this test file was already incomplete (missing other async bindings that cause mount failures). This is not a Phase 150 regression.

File: `frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx`

**Note:** After Phase 150-02 adds `GetShellWebShareWarningEnabled` to SettingsTab, this test's mock will also be missing that binding. However since these tests were already failing, the pre-existing issue must be fixed before this can be evaluated.

### 2. SettingsTab.appearance-theme.test.tsx — 5 intentional RED tests

These 5 tests are labeled "RED — fails until POL-02 lands" and are intentionally failing, waiting for a future phase (POL-02) to implement the UI theme toggle. They were failing before Phase 150 and will continue to fail until POL-02 is delivered.

File: `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx`
