---
phase: 124
slug: files-write-capability-webserver-write-routes-web-share-opt-in
status: draft
nyquist_compliant: false
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
| (planner fills) | — | — | CAP-01..10 | CSRF/authz | 403 without files.write; 2xx with | unit/integration | `go test -race ./internal/webserver/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Webserver write-route gating tests (403 without cap / 2xx with cap on all five routes)
- [ ] CSRF Origin-check tests (mismatch → 403; absent Origin → vacuous pass)
- [ ] `TestHasPerm_NoStringsContains_Write` static-grep gate
- [ ] `TestSettingsMigration_FilesWriteDefaultsFalse` migration test
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
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
