---
phase: 169
slug: tailscale-detection-fix
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-02
---

# Phase 169 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from 169-RESEARCH.md "Validation Architecture".

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (backend); vitest (frontend — SettingsTab copy only) |
| **Config file** | none (Go); `frontend/vitest.config.*` (frontend) |
| **Quick run command** | `go test ./internal/webserver/ -run TestCheckHealth -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds (backend package); ~2 min full |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/ -run TestCheckHealth -count=1 && gofmt -l internal/webserver/ && go vet ./internal/webserver/`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full backend suite green + `golangci-lint run` clean + frontend `tsc && vite build` (per memory `feedback_post_merge_gate_run_tsc`) + M-45 live acceptance
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 169-01-01 | 01 | 1 | FIX-05 | — | CLI fallback fully removed; SDK-success path byte-identical | build/regression | `go build ./... && go vet ./... && go test ./internal/webserver/ -run TestCheckHealth -count=1` | ✅ (revert) | ⬜ pending |
| 169-01-02 | 01 | 1 | FIX-05 | T-169-01 (symlink→int parse) | Permission-limited detected, Connected stays false; daemon-down NOT flagged | unit (`permProbeFunc` fake) | `go test ./internal/webserver/ -run TestCheckHealth_PermissionLimited -count=1` | ❌ W0 (new) | ⬜ pending |
| 169-01-03 | 01 | 2 | FIX-05 | — | SettingsTab shows distinct guidance row, never "Connected" | unit (vitest) + manual | `cd frontend && pnpm test SettingsTab` | ❌ W0 (new) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/tailscale_test.go` — add `TestCheckHealth_PermissionLimited` and `TestCheckHealth_DaemonDown_NotPermissionLimited`; remove the 3 CLI-fallback tests (`TestCheckHealth_CLIFallback_*`); update all existing `checkHealth(...)` call sites to the reverted-plus-`permProbeFunc` signature.
- [ ] `TESTING.md` — update Section 4 traceability for `internal/webserver/tailscale_test.go`; tighten M-45 (Section 5, Category W) wording to the **macsys Standalone / signed-installer** distribution per REVIEW IN-02. Run `bash tests/check-traceability-paths.sh` before commit (repo standing rule).
- [ ] No framework install needed — existing Go + vitest infra covers all automated requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| UI shows distinct "daemon running, status unreadable (permission-limited)" state + guidance, never a false "Connected", on a real non-admin macsys account | FIX-05 (SC1, SC3) | Requires a genuine non-admin macOS user on a macsys (signed-installer) Tailscale install to reproduce the `sameuserproof` EACCES; cannot repro on this dev box (`ken` ∈ `admin`) or in CI | M-45 (TESTING.md Category W): on a non-admin macsys account, open Settings → Tailscale; confirm the status reads permission-limited (not "Daemon Stopped"/"Not Connected"), shows grant-admin / Homebrew-build guidance, and does NOT show Connected |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter (planner sets when tasks map cleanly)

**Approval:** pending
