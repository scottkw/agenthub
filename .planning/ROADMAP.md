# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- 🚧 **v1.2 Tailscale-Only Networking** — Phases 14-18 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: PTY Foundation (2/2 plans) — completed 2026-03-18
- [x] Phase 2: Session Registry + WebSocket Relay (2/2 plans) — completed 2026-03-18
- [x] Phase 3: Wails Desktop UI (3/3 plans) — completed 2026-03-18
- [x] Phase 4: Web Serving + TLS + Auth (4/4 plans) — completed 2026-03-18
- [x] Phase 5: QR Codes + Status Indicators (6/6 plans) — completed 2026-03-18
- [x] Phase 6: Distribution + Cross-Platform (2/2 plans) — completed 2026-03-19

</details>

<details>
<summary>✅ v1.1 Polish & Build (Phases 7-13) — SHIPPED 2026-03-20</summary>

- [x] Phase 7: Layout Baseline (1/1 plans) — completed 2026-03-19
- [x] Phase 8: Per-Tab Status Bar (2/2 plans) — completed 2026-03-19
- [x] Phase 9: Settings Modal Overhaul (1/1 plans) — completed 2026-03-19
- [x] Phase 10: Per-Tab Font Size (1/1 plans) — completed 2026-03-19
- [x] Phase 11: New-Session Modal (3/3 plans) — completed 2026-03-19
- [x] Phase 12: Tab Rename + Web Dashboard (3/3 plans) — completed 2026-03-20
- [x] Phase 13: Build Script (2/2 plans) — completed 2026-03-20

</details>

### v1.2 Tailscale-Only Networking (In Progress)

**Milestone Goal:** Simplify networking to Tailscale-only — Let's Encrypt certs from the Tailscale daemon, Tailscale IP binding, health checks with instructional modal, and complete removal of self-signed TLS, password auth, and generic VPN interface support.

**Phase Summary:**

- [x] **Phase 14: Tailscale Health Check Infrastructure** — Backend health check logic and Wails binding; zero regression risk; unblocks every subsequent phase (completed 2026-03-20)
- [x] **Phase 15: Tailscale TLS + Interface Binding** — Replace self-signed cert infrastructure with Let's Encrypt via Tailscale daemon; bind web server to Tailscale IP (completed 2026-03-20)
- [x] **Phase 16: Auth Layer Removal** — Remove password auth, per-session tokens, and all auth middleware; safe only after Phase 15 confirms Tailscale IP binding (completed 2026-03-20)
- [ ] **Phase 17: Dead Code Cleanup** — Delete obsolete files and simplify Config struct; compiler-guided; unblocks clean codebase
- [ ] **Phase 18: Frontend Health Modal + Status UI** — Instructional health modal with three-state guidance, CT disclosure, and Tailscale status indicator replacing interface picker

## Phase Details

### Phase 14: Tailscale Health Check Infrastructure
**Goal**: The app can detect Tailscale installation, connection, and cert readiness, and exposes that state to the frontend via a Wails method
**Depends on**: Phase 13 (v1.1 complete)
**Requirements**: HEALTH-01, HEALTH-02, HEALTH-03, HEALTH-06
**Success Criteria** (what must be TRUE):
  1. App correctly detects when Tailscale daemon is not reachable (not installed or not running)
  2. App correctly detects when Tailscale is installed but not connected to a tailnet
  3. App correctly detects when Tailscale is connected but HTTPS certs are not enabled
  4. App correctly detects when all three health conditions are satisfied
  5. Health checks run on a background goroutine and the frontend receives updated state automatically without a restart
**Plans:** 2/2 plans complete
Plans:
- [ ] 14-01-PLAN.md — TailscaleHealth struct + CheckHealth function with TDD tests
- [ ] 14-02-PLAN.md — Wails GetTailscaleStatus binding + background health poller

### Phase 15: Tailscale TLS + Interface Binding
**Goal**: The web server uses Let's Encrypt certificates from the Tailscale daemon and binds exclusively to the Tailscale interface IP, with the machine FQDN auto-derived and a CT log disclosure surfaced before first cert use
**Depends on**: Phase 14
**Requirements**: TLS-01, TLS-02, TLS-03, TLS-04, TLS-05
**Success Criteria** (what must be TRUE):
  1. Browser can open the web dashboard via the `.ts.net` HTTPS URL without a certificate warning
  2. Web server refuses to start if the Tailscale interface IP is not available (health check gates startup)
  3. The FQDN shown in the app and QR codes is derived from the Tailscale daemon, not any hardcoded value
  4. User sees a Certificate Transparency disclosure before the first cert is provisioned
  5. Self-signed certificate generation code is gone and no cert files are written to disk
**Plans:** 2/2 plans complete
Plans:
- [ ] 15-01-PLAN.md — WebServer TLS swap: GetCertificate hook, FQDN BaseURL, delete tls.go, update tests
- [ ] 15-02-PLAN.md — App layer + frontend: StartWebServer(port), CT disclosure, SettingsPanel update

### Phase 16: Auth Layer Removal
**Goal**: The web dashboard and session streams are accessible to any tailnet member without a password or token; all auth infrastructure is deleted from the codebase
**Depends on**: Phase 15
**Requirements**: AUTH-01, AUTH-02, AUTH-03
**Success Criteria** (what must be TRUE):
  1. A tailnet member can open the web dashboard URL directly without being prompted for a password
  2. Session stream URLs no longer contain tokens and open directly for any tailnet member
  3. No login route, no token-generation route, and no auth middleware exists in the running server
**Plans:** 2/2 plans complete
Plans:
- [ ] 16-01-PLAN.md — Backend auth removal: delete auth.go/tokens.go, strip middleware, update app.go and Go tests
- [ ] 16-02-PLAN.md — Frontend auth removal: SettingsPanel Security tab, StatusBar Copy Link, dashboard.html login/CA sections, Wails bindings

### Phase 17: Dead Code Cleanup
**Goal**: All code that existed solely to support generic VPN interface selection, auth middleware, or token infrastructure is deleted; the codebase builds cleanly with no dead paths
**Depends on**: Phase 16
**Requirements**: CLEAN-01, CLEAN-02, CLEAN-03
**Success Criteria** (what must be TRUE):
  1. `go build ./...` passes with zero errors after the deletions
  2. No generic VPN interface picker, password field, or token UI remains in the settings panel
  3. Auth middleware, token generation routes, and VPN interface code are absent from the compiled binary (confirmed by source inspection)
**Plans:** 1/2 plans executed
Plans:
- [ ] 17-01-PLAN.md — Backend Go dead code deletion (network.go, GetNetworkInterfaces)
- [ ] 17-02-PLAN.md — Frontend Wails binding cleanup (GetNetworkInterfaces exports, NetworkInterface type)

### Phase 18: Frontend Health Modal + Status UI
**Goal**: Users see clear, platform-specific instructional guidance when any Tailscale health check fails, and the settings panel shows Tailscale connection status in place of the removed interface picker
**Depends on**: Phase 14 (GetTailscaleStatus available), Phase 17 (settings UI cleaned up)
**Requirements**: HEALTH-04, HEALTH-05
**Success Criteria** (what must be TRUE):
  1. When Tailscale is not installed, the modal shows platform-specific installation instructions (macOS menu bar, Linux CLI, Windows tray)
  2. When Tailscale is installed but not connected, the modal shows tailnet connection instructions distinct from the "not installed" message
  3. When Tailscale is connected but certs are not enabled, the modal shows cert-enablement instructions with a "Check Again" button
  4. The Certificate Transparency disclosure is visible in the modal before any cert provisioning attempt
  5. Settings panel shows a Tailscale connection status indicator where the VPN interface picker used to be
**Plans:** 2 plans
Plans:
- [ ] 17-01-PLAN.md — Backend Go dead code deletion (network.go, GetNetworkInterfaces)
- [ ] 17-02-PLAN.md — Frontend Wails binding cleanup (GetNetworkInterfaces exports, NetworkInterface type)

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. PTY Foundation | v1.0 | 2/2 | Complete | 2026-03-18 |
| 2. Session Registry + WebSocket Relay | v1.0 | 2/2 | Complete | 2026-03-18 |
| 3. Wails Desktop UI | v1.0 | 3/3 | Complete | 2026-03-18 |
| 4. Web Serving + TLS + Auth | v1.0 | 4/4 | Complete | 2026-03-18 |
| 5. QR Codes + Status Indicators | v1.0 | 6/6 | Complete | 2026-03-18 |
| 6. Distribution + Cross-Platform | v1.0 | 2/2 | Complete | 2026-03-19 |
| 7. Layout Baseline | v1.1 | 1/1 | Complete | 2026-03-19 |
| 8. Per-Tab Status Bar | v1.1 | 2/2 | Complete | 2026-03-19 |
| 9. Settings Modal Overhaul | v1.1 | 1/1 | Complete | 2026-03-19 |
| 10. Per-Tab Font Size | v1.1 | 1/1 | Complete | 2026-03-19 |
| 11. New-Session Modal | v1.1 | 3/3 | Complete | 2026-03-19 |
| 12. Tab Rename + Web Dashboard | v1.1 | 3/3 | Complete | 2026-03-20 |
| 13. Build Script | v1.1 | 2/2 | Complete | 2026-03-20 |
| 14. Tailscale Health Check Infrastructure | 2/2 | Complete    | 2026-03-20 | - |
| 15. Tailscale TLS + Interface Binding | 2/2 | Complete    | 2026-03-20 | - |
| 16. Auth Layer Removal | 2/2 | Complete    | 2026-03-20 | - |
| 17. Dead Code Cleanup | 1/2 | In Progress|  | - |
| 18. Frontend Health Modal + Status UI | v1.2 | 0/? | Not started | - |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
