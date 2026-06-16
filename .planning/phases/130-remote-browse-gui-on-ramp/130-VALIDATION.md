---
phase: 130
slug: remote-browse-gui-on-ramp
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-06-15
---

# Phase 130 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (Go)** | `testing` (stdlib) |
| **Framework (frontend)** | vitest 4.1.0 (`frontend/vite.config.ts`) |
| **Quick run command** | `go test ./internal/tailnet/... ./internal/relay/... ./internal/daemon/... ./internal/webserver/... -count=1 && (cd frontend && pnpm test)` |
| **Full suite command** | `go test ./... && (cd frontend && pnpm test)` |
| **Estimated runtime** | Go ~60s; frontend ~30s |

---

## Sampling Rate

- **After every task commit:** quick run for the touched layer (Go package or `cd frontend && pnpm test`)
- **After every plan wave:** full suite (`go test ./...` + `cd frontend && pnpm test`)
- **Before `/gsd:verify-work`:** full suite green
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

| Task ID | Wave | Requirement | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|-------------|-----------------|-----------|-------------------|-------------|--------|
| W0 | 0 | RB-01 | `/api/sessions/meta` returns shareable metadata, no cap required | unit (Go) | `go test ./internal/webserver/... -run TestSessionsMeta` | ❌ W0 | ⬜ |
| W0 | 0 | RB-01 | `FetchAllPeerSessionsMeta` includes all probed peers (no silent drop) | unit (Go) | `go test ./internal/tailnet/... -run TestFetchAllPeerSessionsMeta` | ❌ W0 | ⬜ |
| W0 | 0 | RB-01 | `GetRemoteSessionsWithMeta` RPC returns peers w/ `Reachable` field | unit (Go) | `go test -run TestGetRemoteSessionsWithMeta` | ❌ W0 | ⬜ |
| W0 | 0 | RB-03 | `/api/sessions/meta` returns NO caps/grants/content | unit (Go) | `go test ./internal/webserver/... -run TestSessionsMeta_NoCapInResponse` | ❌ W0 | ⬜ |
| W0 | 0 | RB-04 | Panel: "Unreachable" badge / "No shareable sessions" / never false "No peers" | unit (vitest) | `cd frontend && pnpm test` | ❌ W0 (extend) | ⬜ |
| W0 | 0 | RB-05 | Relay-surface discover→pick→browse via `api.RelayHandler()` | unit (Go) | `go test ./internal/daemon/... -run TestRemoteFiles_DiscoverAndBrowse_RelaySurface` | ❌ W0 | ⬜ |
| — | 1+ | RB-01 | implementation turns the RB-01 tests green | unit | (above) | — | ⬜ |
| — | 1+ | RB-02 | "Browse Files" pick opens FileBrowser (`onBrowseFiles`) | unit (vitest) | `cd frontend && pnpm test` | ✅ extend `App.remoteFileBrowser.test.tsx` | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/sessions_meta_test.go` (new) — RB-01 + RB-03: `/api/sessions/meta` returns web-enabled session metadata without a cap; response carries NO cap/grant/content fields; empty when none enabled
- [ ] `internal/tailnet/tailnet_test.go` (extend) — RB-01: `FetchAllPeerSessionsMeta` includes unreachable peers and reachable-but-empty peers (no silent drop at the `len==0` gate, sessions.go:93)
- [ ] `app_test.go` (extend) — RB-01: `GetRemoteSessionsWithMeta` exposes a `Reachable` field per peer
- [ ] `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` (extend) — RB-04: per-peer state rendering (unreachable badge, "No shareable sessions" text, never false "No remote peers found"); **update existing assertions** `'Shows web-enabled sessions only'` → `'Shows shareable sessions'` (lines ~89, 134–135)
- [ ] `internal/daemon/relay_remote_files_test.go` (extend) — **RB-05 (release-blocking relay-surface coverage)**: `TestRemoteFiles_DiscoverAndBrowse_RelaySurface` — fixture peer with `/api/sessions/meta` handler, deposit cap via `depositCapOnSocket`, confirm the discover→pick→browse path reaches the proxy through `api.RelayHandler()` (the loopback the Wails GUI uses). Guards the v3.5-class blind spot.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Non-tailnet / unauthorized caller cannot reach `/api/sessions/meta` | RB-03 | Trust is network-layer (webserver binds to the Tailscale `100.x` IP); not unit-testable | On a non-tailnet host, confirm the metadata endpoint is unreachable (no route to the bind IP) |
| End-to-end discover→list→pick→browse on a real two-machine tailnet | RB-01/02 | Requires two live tailnet peers with shareable sessions | On machine A open Remote Sessions, see machine B's shareable sessions, pick one, browse its files |

*Automated coverage exists for all application-layer behavior (endpoint shape, no-cap/no-content guarantee, peer inclusion, panel states, relay-surface browse path). Only the network-layer trust boundary and the full two-machine flow are manual.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (incl. the RB-05 relay-surface test)
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
