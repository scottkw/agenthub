---
phase: 134
slug: modal-interaction
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-06-17
updated: 2026-06-17
---

# Phase 134 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Covers both the original modal work (plans 134-01..05, frontend/vitest) and the
> remote-WS-proxy expansion (plans 134-06..08, Go + behavioral frontend tests).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (frontend)** | vitest (jsdom environment) |
| **Framework (Go)** | `go test` (stdlib testing + `net/http/httptest`) |
| **Config files** | `frontend/vite.config.ts` (vitest); `go.mod` (Go) |
| **Quick run (frontend)** | `cd frontend && pnpm test --run --reporter=verbose` |
| **Quick run (Go WS proxy)** | `go test ./internal/daemon/ -run RemoteSessionWS -race` |
| **Full suite** | `go test ./... -tags wailsassets` then `cd frontend && pnpm test --run` |
| **Type check** | `cd frontend && pnpm exec tsc --noEmit` |
| **Estimated runtime** | frontend ~30s (1686+ tests); Go WS-proxy subset ~5s |

---

## Sampling Rate

- **After every task commit:** `go test ./internal/daemon -run RemoteSessionWS -race` (Go tasks) and/or `cd frontend && pnpm test --run --reporter=verbose 2>&1 | tail -20` (frontend tasks)
- **After every plan wave:** `go test ./... -race` + `cd frontend && pnpm test --run`
- **Phase gate:** full suite green + `tsc --noEmit` clean, THEN the manual two-peer live UAT (below) before `/gsd:verify-work`
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

### Original modal work (plans 134-01..05 — COMPLETE)

| Requirement | Behavior | Test Type | Automated Command | Status |
|-------------|----------|-----------|-------------------|--------|
| MODAL-01 | card body onClick → onCardClick; Open/menu buttons stopPropagation | unit | `pnpm test --run SessionCard` | ✅ green |
| MODAL-02 | Escape/close/click-outside → onClose; focus returns to card | unit | `pnpm test --run HubModal` | ✅ green |
| MODAL-03 | attention routing; HubInteractiveModal mounts TerminalPanel | unit | `pnpm test --run HubModal HubInteractiveModal` | ✅ green |
| MODAL-04 | briefing renders tail; Send disabled when empty | unit | `pnpm test --run HubBriefingModal` | ✅ green |
| MODAL-05 | TerminalPanel isActive gated on open phase | unit | `pnpm test --run HubInteractiveModal` | ✅ green |
| CSS | overlay z-index 200; grow/shrink keyframes; reduced-motion guard | CSS assertion | `pnpm test --run style.hub.modal` | ✅ green |

### Remote-WS-proxy expansion (plans 134-06..08)

| Req | Behavior | Test Type | Automated Command | File Exists? |
|-----|----------|-----------|-------------------|-------------|
| MODAL-06 / WS-PROXY-01 | `/api/relay/remote/{sid}/ws` mounted on relay surface (not 404) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_MountedOnRelay` | ❌ W0 (134-06) |
| MODAL-06 / WS-PROXY-02 | No cap deposited → handler reached, "no cap registered" (not bare route-miss) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_NoCap` | ❌ W0 (134-06) |
| MODAL-06 / WS-PROXY-03 | With cap, frames copy bidirectionally to fixture peer WS | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_FrameCopy -race` | ❌ W0 (134-06) |
| MODAL-06 / WS-PROXY-04 | Proxy injects `Origin: <baseURL>` on upstream dial (fixture asserts non-empty) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_InjectsOrigin` | ❌ W0 (134-06) |
| MODAL-06 / WS-PROXY-05 | Cross-site inbound Origin rejected at Accept (mirror origin_test.go) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_RejectsCrossSiteOrigin` | ❌ W0 (134-06) |
| MODAL-06 / WS-PROXY-06 | Copy loop uses request context, survives past 10s dial timeout | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_LongLived` | ❌ W0 (134-06) |
| MODAL-03 / FE-URL-01 | RelayClient builds `/api/relay/remote/{id}/ws` when `remote` set, local path otherwise | unit (Vitest) | `pnpm test --run relayClient` | ❌ W0 (134-07) |
| MODAL-03 / FE-ROUTE-01 | HubInteractiveModal threads `remote` from `isRemote` discriminator | behavioral (Vitest) | `pnpm test --run HubInteractiveModal` | partial → behavioral (134-08) |
| MODAL-04 / CR-03-01 | Briefing send: open→sendInput→close ordering; timeout cleanup; no post-abandon send | behavioral (Vitest, mock RelayClient) | `pnpm test --run HubBriefingModal` | ❌ (WR-07 gap → 134-08) |
| MODAL-04 / TAIL-01 | Remote briefing tail rendered from WS scrollback snapshot (mock proxied WS) | behavioral (Vitest) | `pnpm test --run HubBriefingModal` | ❌ W0 (134-07/08) |

*Test-style note: the expansion's frontend tests are BEHAVIORAL (mocked `WebSocket`/`RelayClient`, asserted ordering/timeout/tail-render), NOT `?raw` source-string checks — this is the WR-07 remediation. Go WS-proxy tests model `relay_remote_files_test.go` (`newFixtureRemotePeer*` + `newDaemonAPIWithUpstreamCert`), with the fixture peer serving a cap-guarded WS echo endpoint.*

---

## Wave 0 Requirements (expansion)

- [ ] `internal/daemon/remote_ws_proxy_test.go` — WS-PROXY-01..06 (fixture peer must serve a WS `/sessions/{id}/ws` asserting `?cap=` + non-empty Origin, then echo frames)
- [ ] Fixture peer extension: a cap-guarded WS echo endpoint (e.g. `newFixtureRemotePeerWithWS`)
- [ ] `frontend/src/lib/relayClient.test.ts` — assert URL construction for both local/remote modes (FE-URL-01)
- [ ] `frontend/src/components/Hub/HubBriefingModal.test.tsx` — behavioral mock-RelayClient tests (CR-03-01, TAIL-01); replaces `?raw` checks
- [ ] Export `relay.LoopbackOriginPatterns` (or a relay-package test confirming the daemon proxy reuses it)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real two-machine tailnet interactive remote terminal | MODAL-06 | Requires two live peers with real Tailscale certs; no automated substitute | On machine A: open a remote (machine B) session card → join-code cap exchange → modal mounts terminal; type a command, confirm it executes on B; verify resize, scrollback, copy/paste |
| Remote briefing respond round-trip | MODAL-04, MODAL-06 | Requires a real remote waiting session | Open a remote waiting session's briefing modal; confirm tail shows the real prompt (from WS snapshot); type + Send; confirm B's session receives input (write-cap code only) |
| Read-only cap UX (deferred to Phase 135) | MODAL-06, A11Y | Read-only cap → Send silently drops at peer; non-color read-only indicator is Phase 135 a11y scope | With a read-only join code, confirm output renders; note Send currently presents enabled (documented gap → Phase 135) |
| Grow/shrink animation + focus return (local) | MODAL-01, MODAL-02 | xterm.js + animation need a live webview | Native `wails dev`: click card → grow; Escape → shrink + focus returns to card |

---

## Validation Sign-Off

- [x] Every task in plans 134-06..08 carries an `<automated>` verify command (Go `-race` or `pnpm test`)
- [x] No watch-mode flags; feedback latency < 30s
- [x] Wave-0 gaps enumerated (Go proxy tests + behavioral FE tests + RelayClient URL test)
- [x] Backend tier (WS proxy) covered with Go integration tests — not omitted
- [x] `?raw`-only guidance removed; expansion uses behavioral frontend tests (WR-07)
- [ ] `wave_0_complete` flips true once `remote_ws_proxy_test.go` + `relayClient.test.ts` land (plans 134-06/07)

**Approval:** approved 2026-06-17 (covers expansion plans 134-06..08)
