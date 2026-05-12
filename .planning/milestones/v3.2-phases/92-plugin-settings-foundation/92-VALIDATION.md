---
phase: 92
slug: plugin-settings-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-04
---

# Phase 92 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (daemon), vitest (frontend) |
| **Config file** | `go.mod` (root), `frontend/vitest.config.ts` |
| **Quick run command** | `go test ./internal/daemon/... -run TestSettingsMigrationV3_1ToV3_2 -count=1` |
| **Full suite command** | `go test ./... -count=1 && (cd frontend && pnpm test --run)` |
| **Estimated runtime** | ~30s (Go) + ~15s (vitest) |

---

## Sampling Rate

- **After every task commit:** Run quick run command for the task's surface (Go or vitest)
- **After every plan wave:** Run full suite
- **Before `/gsd-verify-work`:** Full suite must be green; manual UAT (toggle persistence + event propagation) must be signed off
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

> Populated by the planner. Each task in PLAN.md must have either an `<automated>` block referencing a command below, or a Wave 0 dependency.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | PLUG-01/02/03/PUI-01 | — | n/a (Phase 92 ships no addon side effects) | unit/integration | TBD per plan | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/fixtures/settings_v3.1.json` — minimal v3.1 settings.json fixture (drives PLUG-02 migration test)
- [ ] `internal/daemon/settings_test.go` — house for `TestSettingsMigrationV3_1ToV3_2`, `TestDefaultPluginSettings`, `TestSetPluginSettingsRoundTrip` (extend if file exists; create if not)
- [ ] `frontend/src/components/__tests__/PluginsSection.test.tsx` — vitest house for the 8-row render assertion (PUI-01) and toggle interaction (PLUG-01)

*Wave 0 also installs no new test framework — `go test` and `vitest` are already in the toolchain (verified via `go.mod` and `frontend/package.json`).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Toggle persistence across GUI + daemon restart | PLUG-01 | Requires real Wails build + filesystem inspection | 1) `wails build -tags wailsassets` 2) Open app, toggle 3 plugins, Save 3) Quit app, restart daemon, reopen 4) Confirm toggles match; cat `~/.agenthub/settings.json` shows persisted values + `schemaVersion: 2` |
| `settings:plugins` event reaches every open `TerminalPanel` | PLUG-03 | Requires multi-tab desktop session and devtools subscription | 1) Open app with 3 terminal tabs 2) Toggle a plugin, Save 3) In React DevTools confirm each `TerminalPanel` receives an updated `pluginConfig` prop within 200ms 4) Confirm no app/session restart was required |
| v3.1 → v3.2 migration on a real returning-user settings.json | PLUG-02 | Fixture test covers schema; this validates real-world v3.1 file shape | 1) Copy a real v3.1 `settings.json` into test location 2) Boot v3.2 daemon 3) Confirm log shows migration event, file rewritten with `schemaVersion: 2` and 8 plugin defaults populated, no zero values |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (fixture + test files listed above)
- [ ] No watch-mode flags (use `--run` for vitest, no `-watch` for `go test`)
- [ ] Feedback latency < 30s for quick command
- [ ] `nyquist_compliant: true` set in frontmatter once planner completes Per-Task Verification Map

**Approval:** pending
