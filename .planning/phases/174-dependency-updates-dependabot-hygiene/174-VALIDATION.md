---
phase: 174
slug: dependency-updates-dependabot-hygiene
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-08
---

# Phase 174 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> This is a dependency-hygiene phase: "validation" is the existing build+test gate
> re-run against each bumped dependency, not new unit tests. No new test files are
> authored. The gate proves each bump is non-breaking; the flaky CI badges are NOT
> trusted (see RESEARCH.md — CI-YAML-only bumps show the same "failures").

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (backend) + vitest / `tsc && vite build` (frontend) |
| **Config file** | Existing — `go.mod`, `frontend/package.json`, `.github/workflows/build.yml` |
| **Quick run command** | `go build ./... && go vet ./...` |
| **Full suite command** | `go test ./... && (cd frontend && pnpm tsc && pnpm vite build)` |
| **Estimated runtime** | ~120–240 seconds (backend test suite dominates) |

---

## Sampling Rate

- **After every task commit (a bump applied):** Run `go build ./... && go vet ./...` (or, for CI-YAML-only bumps, `actionlint`/YAML parse — no Go impact).
- **After every plan wave:** Run the full suite `go test ./...` + `tsc && vite build`; for the coder/websocket bump specifically run `go test ./internal/webserver/... ./internal/relay/...`.
- **Before `/gsd-verify-work`:** Full suite green locally + deb packaging still builds (goreleaser/nfpm; pipeline produces deb only, no rpm — RESEARCH.md).
- **Max feedback latency:** ~240 seconds.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 174-01-01 | 01 | A | DEP-01 (#114 attest-build-provenance) | — | CI action pin updated, workflow still parses | build | `actionlint .github/workflows/*.yml` | ✅ | ⬜ pending |
| 174-01-02 | 01 | A | DEP-01 (#113 setup-go, 3 files) | — | workflows parse, go setup unchanged | build | `actionlint .github/workflows/*.yml` | ✅ | ⬜ pending |
| 174-01-03 | 01 | A | DEP-01 (#103 action-gh-release) | — | release workflow parses | build | `actionlint .github/workflows/*.yml` | ✅ | ⬜ pending |
| 174-01-04 | 01 | A | DEP-01 (#85 pnpm/action-setup, 3 files) | — | workflows parse | build | `actionlint .github/workflows/*.yml` | ✅ | ⬜ pending |
| 174-02-01 | 02 | B | DEP-01 (#89 coder/websocket 1.8.15) | — | webserver+relay behavior unchanged | unit | `go test ./internal/webserver/... ./internal/relay/...` | ✅ | ⬜ pending |
| 174-02-02 | 02 | B | DEP-01 (#106 x/term 0.44.0) | — | build+full test green | unit | `go build ./... && go test ./...` | ✅ | ⬜ pending |
| 174-02-03 | 02 | B | DEP-01 (#105 nfpm/v2 2.47.0) | — | deb packaging still builds | build | `goreleaser build --snapshot --clean` (or CI package job) | ✅ | ⬜ pending |
| 174-03-01 | 03 | C | DEP-02 (#104 wails defer) | — | ignore entry blocks 2.12.0 bump only | build | `yq '.updates[].ignore' .github/dependabot.yml` | ✅ | ⬜ pending |
| 174-03-02 | 03 | C | DEP-02 (#88 tailscale defer) | — | ignore entry blocks 1.100.0 bump only | build | `yq '.updates[].ignore' .github/dependabot.yml` | ✅ | ⬜ pending |
| 174-03-03 | 03 | C | DEP-02 (#102 checkout v7 defer) | — | ignore entry blocks v7 major only | build | `yq '.updates[].ignore' .github/dependabot.yml` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements.* No new test files, no framework install. `actionlint`/`yq` are optional lint aids — plans must degrade to a manual YAML read if they are not installed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Windows build not broken by any bump | DEP-01 | No local Windows toolchain; go-webview2 v1.0.19 pin is the guard | Confirm no bump touches `go-webview2` version; CI Windows leg (or post-ship release build) is the true check |
| Deferred Dependabot PRs stop re-opening | DEP-02 | Dependabot re-open behavior only observable over time on GitHub | After ignore entries merge to main, confirm #104/#88/#102 closed and do not re-appear on next Dependabot run |
| deb package installs & runs | DEP-01 (#105 nfpm) | Requires a Debian/Ubuntu host | Post-ship: `dpkg -i` the built .deb and launch daemon |

---

## Validation Sign-Off

- [ ] All tasks have an automated verify command (build/test/lint) or a documented manual-only reason
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (none — existing infra)
- [ ] No watch-mode flags
- [ ] Feedback latency < 240s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
