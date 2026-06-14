---
phase: 124
slug: files-write-capability-webserver-write-routes-web-share-opt-in
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-14
---

# Phase 124 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend: capability/middleware/migration) + vitest (frontend toggle UI) |
| **Config file** | none (Go native); frontend/ vitest config |
| **Quick run command** | `go test ./internal/webserver/... ./internal/daemon/... ./internal/capability/...` |
| **Full suite command** | `go test -race ./internal/... && (cd frontend && pnpm test)` |
| **Estimated runtime** | ~30-90 seconds backend; frontend a few seconds |

---

## Sampling Rate

- **After every task commit:** Run the relevant package quick test
- **After every plan wave:** Run `go test -race ./internal/...`
- **Before `/gsd:verify-work`:** Full suite green; static-grep gate `TestHasPerm_NoStringsContains_Write` passes
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-T0 | 124-01 | 1 | CAP-01, CAP-02, CAP-09 | T-124-SC | Wave-0 RED: requireFilesWrite/Origin/static-grep gate undefined before impl | unit (RED) | `cd internal/webserver && go test -run 'TestRequireFilesWrite\|TestHasPerm_NoStringsContains_Write' -count=1 . 2>&1 \| grep -qE 'undefined\|FAIL\|build failed' && echo RED-OK` | ✅ | ⬜ pending |
| 01-T1 | 124-01 | 1 | CAP-01, CAP-03, CAP-07 | CSRF/authz | PermFilesWrite const + requireFilesWrite gate (403 without cap) + Origin-mismatch → 403 | unit | `cd internal/webserver && go build ./... && go test -run 'TestRequireFilesWrite\|TestHasPerm_NoStringsContains_Write' -count=1 . 2>&1 \| tail -3` | ✅ | ⬜ pending |
| 01-T2 | 124-01 | 1 | CAP-01, CAP-02 | authz | Five write routes mounted behind requireFilesWrite (2xx with cap) | integration | `cd internal/webserver && go test -run 'TestRequireFilesWrite' -count=1 . 2>&1 \| tail -3` | ✅ | ⬜ pending |
| 02-T0 | 124-02 | 1 | CAP-08 | — | Wave-0 RED: FilesWrite migration test fails before schema bump | unit (RED) | `cd internal/daemon && go test -run 'TestSettingsMigration_FilesWrite' -count=1 . 2>&1 \| grep -qE 'undefined\|FAIL\|build failed' && echo RED-OK` | ✅ | ⬜ pending |
| 02-T1 | 124-02 | 1 | CAP-04, CAP-08 | — | schemaVersion 4 + FilesWrite default false + per-session write map + homeDir signal | unit | `cd internal/daemon && go build ./... && go test -run 'TestSettingsMigration_FilesWrite' -count=1 . 2>&1 \| tail -3` | ✅ | ⬜ pending |
| 02-T2 | 124-02 | 1 | CAP-04, CAP-06 | authz | Cap mint appends files.write only when toggle ON; HomeDir on IssueCapabilitiesResponse | unit | `cd internal/daemon && go build ./... && go test -run 'IssueCapabilities\|TestSettingsMigration_FilesWrite' -count=1 . 2>&1 \| tail -3` | ✅ | ⬜ pending |
| 03-T0 | 124-03 | 2 | CAP-10 | — | Wave-0 RED: remote write-proxy body-forward test fails before impl | unit (RED) | `cd internal/daemon && go test -run 'TestRemoteFilesWrite' -count=1 . 2>&1 \| grep -qE 'FAIL\|empty\|undefined\|build failed' && echo RED-OK` | ✅ | ⬜ pending |
| 03-T1 | 124-03 | 2 | CAP-10 | I/T | Forward r.Body + Content-Type for write verbs; caller cap stripped | unit | `cd internal/daemon && go build ./... && go test -run 'TestRemoteFilesWrite\|TestRemoteFiles_CallerCapStripped\|TestRemoteFiles_ListRoundTrip' -count=1 . 2>&1 \| tail -4` | ✅ | ⬜ pending |
| 03-T2 | 124-03 | 2 | CAP-10 | authz | Register five remote write proxy routes | integration | `cd internal/daemon && go build ./... && go test -race . -count=1 2>&1 \| tail -3` | ✅ | ⬜ pending |
| 04-T1 | 124-04 | 3 | CAP-04 | T-124-15 | SetSessionFilesWrite binding chain (per-session, no global flag); Wails binding regenerated | integration | `go build ./... && go test -race ./internal/daemon/ -count=1 2>&1 \| tail -3` | ✅ | ⬜ pending |
| 04-T2 | 124-04 | 3 | CAP-06 | T-124-16 | HomeDirWriteWarning renders ⚠ + literal "Warning:" (colorblind-safe) | unit | `cd frontend && pnpm test -- --run HomeDirWriteWarning 2>&1 \| tail -5` | ✅ | ⬜ pending |
| 04-T3 | 124-04 | 3 | CAP-04, CAP-05 | T-124-14, T-124-15 | Owner toggle (default OFF) + viewer opt-in gated/confirmed before write cap minted | unit | `cd frontend && pnpm test -- --run SessionSharePanel DaemonManagerPanel 2>&1 \| tail -6` | ✅ | ⬜ pending |
| 05-T0 | 124-05 | 2 | CAP-06 | — | Wave-0 RED: TUI warning render test fails before impl | unit (RED) | `cd internal/tui && go test -run 'HomeDirWarning\|FilesWarning\|renderFilesTab' -count=1 . 2>&1 \| grep -qE 'FAIL\|undefined\|build failed\|Warning' && echo RED-OK` | ✅ | ⬜ pending |
| 05-T1 | 124-05 | 2 | CAP-06 | T-124-16 | TUI home-dir write warning line in renderFilesTab (⚠ + "Warning:") — parity with GUI | unit | `cd internal/tui && go test -run 'HomeDirWarning\|FilesWarning\|renderFilesTab' -count=1 . 2>&1 \| tail -4` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Webserver write-route gating tests (403 without cap / 2xx with cap on all five routes)
- [ ] CSRF Origin-check tests (mismatch → 403; absent Origin → vacuous pass)
- [ ] `TestHasPerm_NoStringsContains_Write` static-grep gate
- [ ] `TestSettingsMigration_FilesWriteDefaultsFalse` migration test
- [ ] Remote write-proxy body-forwarding test (`TestRemoteFilesWrite`)
- [ ] TUI home-dir warning render test (`HomeDirWarning`/`renderFilesTab`)
- [ ] Frontend toggle test (toggle ON → cap includes "files.write")

*Framework present (go test + vitest) — no install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Home-dir write warning visible in GUI | CAP-06 | requires running GUI + a $HOME-cwd session with writes on | Enable writes for a $HOME session, confirm amber ⚠ "Warning:" banner |
| Home-dir write warning visible in TUI | CAP-06 | requires running TUI + a $HOME-cwd session | Same, in TUI status area — parity check |
| Web-share opt-in persists across daemon restart | CAP-09/SC#5 | requires daemon restart cycle | Toggle on, restart daemon, confirm default state persists |

*Colorblind: verify warning at source level (⚠ glyph + literal "Warning:" text token in code), not by eye.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
