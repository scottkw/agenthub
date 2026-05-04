# Roadmap: AgentHub

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
- ✅ **v1.1 Polish & Build** — Phases 7-13 (shipped 2026-03-20)
- ✅ **v1.2 Tailscale-Only Networking** — Phases 14-18 (shipped 2026-03-23)
- ✅ **v1.3 CLI + Daemon** — Phases 19-26 (shipped 2026-03-25)
- ✅ **v1.4 Unified Binary** — Phases 27-29 (shipped 2026-03-25)
- ✅ **v1.5 Bug Fixes & CLI Args** — Phases 30-34 (shipped 2026-03-26)
- ✅ **v1.6 Terminal Fill Fix v2** — Phase 35 (shipped 2026-03-31)
- ✅ **v1.7 Daemon UX & Branding** — Phases 36-43 (shipped 2026-04-03)
- ✅ **v1.8 GitHub Distribution & CI/CD** — Phases 44-48 (shipped 2026-04-06)
- ✅ **v1.9 Remote Sessions & App Polish** — Phases 49-54 (shipped 2026-04-08)
- ✅ **v1.10 Collapsible Sidebar Navigation** — Phases 55-56 (shipped 2026-04-08)
- ✅ **v1.11 Local Network & UX Polish** — Phases 57-62 (shipped 2026-04-10)
- ✅ **v1.12 UI/UX Polish** — Phases 63-66 (shipped 2026-04-11)
- ✅ **v1.13 Cross-Platform Fixes & UX** — Phases 67-69 (shipped 2026-04-12)
- ✅ **v1.14 UI Polish** — Phases 70-73 (shipped 2026-04-14)
- ✅ **v2.0 Multi-Client, CLI UX & TUI Mode** — Phases 74-78 (shipped 2026-04-16)
- ✅ **v2.1 Bug Fixes & UX** — Phases 79-82 (shipped 2026-04-17)
- ✅ **v3.0 Session Lifecycle & TUI Polish** — Phases 83-86 (shipped 2026-04-19)
- ✅ **v3.1 Security Hardening** — Phases 87-90 (shipped 2026-05-03, closes Issue #35)
- 🚧 **v3.2 Plugin Suite** — Phases 92-99 (in progress, closes Issue #36; Phase 91 deferred to a future milestone — see `.planning/deferred/91-distribution-pipeline-followups/`)

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

<details>
<summary>✅ v1.2 Tailscale-Only Networking (Phases 14-18) — SHIPPED 2026-03-23</summary>

- [x] Phase 14: Tailscale Health Check Infrastructure (2/2 plans) — completed 2026-03-20
- [x] Phase 15: Tailscale TLS + Interface Binding (2/2 plans) — completed 2026-03-20
- [x] Phase 16: Auth Layer Removal (2/2 plans) — completed 2026-03-20
- [x] Phase 17: Dead Code Cleanup (2/2 plans) — completed 2026-03-20
- [x] Phase 18: Frontend Health Modal + Status UI (2/2 plans) — completed 2026-03-22

</details>

<details>
<summary>✅ v1.3 CLI + Daemon (Phases 19-26) — SHIPPED 2026-03-25</summary>

- [x] Phase 19: Daemon Core / Engine + IPC (2/2 plans) — completed 2026-03-23
- [x] Phase 20: Process Separation (2/2 plans) — completed 2026-03-23
- [x] Phase 21: CLI Session + Web Commands (2/2 plans) — completed 2026-03-24
- [x] Phase 22: CLI Attach (2/2 plans) — completed 2026-03-24
- [x] Phase 23: Service Manager Integration (2/2 plans) — completed 2026-03-24
- [x] Phase 24: CLI Polish (2/2 plans) — completed 2026-03-24
- [x] Phase 25: Windows Named Pipe Dial Fix (1/1 plans) — completed 2026-03-24
- [x] Phase 26: Graceful GUI Startup Failure (2/2 plans) — completed 2026-03-24

</details>

<details>
<summary>✅ v1.4 Unified Binary (Phases 27-29) — SHIPPED 2026-03-25</summary>

- [x] Phase 27: Unified Entrypoint (1/1 plans) — completed 2026-03-25
- [x] Phase 28: CLI Package Removal (1/1 plans) — completed 2026-03-25
- [x] Phase 29: Build System & Verification (1/1 plans) — completed 2026-03-25

</details>

<details>
<summary>✅ v1.5 Bug Fixes & CLI Args (Phases 30-34) — SHIPPED 2026-03-26</summary>

- [x] Phase 30: Backend Args Wiring (1/1 plans) — completed 2026-03-26
- [x] Phase 31: CLI Arg Passthrough (1/1 plans) — completed 2026-03-26
- [x] Phase 32: Daemon Startup Performance (2/2 plans) — completed 2026-03-26
- [x] Phase 33: GUI Args Field (1/1 plans) — completed 2026-03-26
- [x] Phase 34: PATH Augmentation (1/1 plans) — completed 2026-03-26

</details>

<details>
<summary>✅ v1.6 Terminal Fill Fix v2 (Phase 35) — SHIPPED 2026-03-31</summary>

- [x] Phase 35: Terminal Fill Retry Loop (1/1 plans) — completed 2026-03-31

</details>

<details>
<summary>✅ v1.7 Daemon UX & Branding (Phases 36-43) — SHIPPED 2026-04-03</summary>

- [x] Phase 36: App Icon (1/1 plans) — completed 2026-04-02
- [x] Phase 37: Splash Screen (1/1 plans) — completed 2026-04-02
- [x] Phase 38: Hostname in Session API (1/1 plans) — completed 2026-04-02
- [x] Phase 39: Web Terminal Status Bar (1/1 plans) — completed 2026-04-02
- [x] Phase 40: CLI Attach Banner (1/1 plans) — completed 2026-04-02
- [x] Phase 41: Daemon Manager Panel (1/1 plans) — completed 2026-04-02
- [x] Phase 42: Daemon Shutdown + Tray Icons (2/2 plans) — completed 2026-04-02
- [x] Phase 43: Tray Dynamic Session Menu (2/2 plans) — completed 2026-04-03

</details>

<details>
<summary>✅ v1.8 GitHub Distribution & CI/CD (Phases 44-48) — SHIPPED 2026-04-06</summary>

- [x] Phase 44: Go Module Path Rewrite (1/1 plans) — completed 2026-04-04
- [x] Phase 45: Release Please Setup (1/1 plans) — completed 2026-04-04
- [x] Phase 46: Release Artifacts & One-Liner (2/2 plans) — completed 2026-04-05
- [x] Phase 47: Homebrew + WinGet Manifests (2/2 plans) — completed 2026-04-05
- [x] Phase 48: Distribution Workflow (3/3 plans) — completed 2026-04-05

</details>

<details>
<summary>✅ v1.9 Remote Sessions & App Polish (Phases 49-54) — SHIPPED 2026-04-08</summary>

- [x] Phase 49: macOS App Menus (2/2 plans) — completed 2026-04-06
- [x] Phase 50: Tailscale Peer Discovery (3/3 plans) — completed 2026-04-06
- [x] Phase 51: Auto-Update Checker (2/2 plans) — completed 2026-04-06
- [x] Phase 52: Remote Sessions GUI (3/3 plans) — completed 2026-04-07
- [x] Phase 53: CLI Remote Sessions (2/2 plans) — completed 2026-04-07
- [x] Phase 54: Tailscale Onboarding (2/2 plans) — completed 2026-04-07

</details>

<details>
<summary>✅ v1.10 Collapsible Sidebar Navigation (Phases 55-56) — SHIPPED 2026-04-08</summary>

- [x] Phase 55: Sidebar Icons + Layout (2/2 plans) — completed 2026-04-08
- [x] Phase 56: Tab Bar Cleanup (1/1 plans) — completed 2026-04-08

</details>

<details>
<summary>✅ v1.11 Local Network & UX Polish (Phases 57-62) — SHIPPED 2026-04-10</summary>

- [x] Phase 57: Agent Discovery Paths (1/1 plans) — completed 2026-04-08
- [x] Phase 58: Settings Tab Conversion (1/1 plans) — completed 2026-04-08
- [x] Phase 59: Web Server Auto-Start (1/1 plans) — completed 2026-04-09
- [x] Phase 60: Local Network Fallback (3/3 plans) — completed 2026-04-09
- [x] Phase 61: Frontend webEnabled Seeding (2/2 plans) — completed 2026-04-09
- [x] Phase 62: Tech Debt Cleanup (1/1 plans) — completed 2026-04-10

</details>

<details>
<summary>✅ v1.12 UI/UX Polish (Phases 63-66) — SHIPPED 2026-04-11</summary>

- [x] Phase 63: Sidebar Icon Centering (1/1 plans) — completed 2026-04-10
- [x] Phase 64: Terminal Padding (1/1 plans) — completed 2026-04-11
- [x] Phase 65: Terminal Theming (1/1 plans) — completed 2026-04-11
- [x] Phase 66: Web Server Link UX (1/1 plans) — completed 2026-04-11

</details>

<details>
<summary>✅ v1.13 Cross-Platform Fixes & UX (Phases 67-69) — SHIPPED 2026-04-12</summary>

- [x] Phase 67: Cross-Platform System Tray (2/2 plans) — completed 2026-04-12
- [x] Phase 68: Agent & Tailscale Discovery + Install Instructions (2/2 plans) — completed 2026-04-12
- [x] Phase 69: Settings Scrollable Layout (1/1 plan) — completed 2026-04-12

</details>

<details>
<summary>✅ v1.14 UI Polish (Phases 70-73) — SHIPPED 2026-04-14</summary>

- [x] Phase 70: Sidebar Icon Position Stability (1/1 plans) — completed 2026-04-13
- [x] Phase 71: OpenCode Theming Fix (5/5 plans) — completed 2026-04-13
- [x] Phase 72: UI Contrast Improvement (2/2 plans) — completed 2026-04-14
- [x] Phase 73: Theme Usability Audit (1/1 plan) — completed 2026-04-14

</details>

<details>
<summary>✅ v2.0 Multi-Client, CLI UX & TUI Mode (Phases 74-78) — SHIPPED 2026-04-16</summary>

- [x] Phase 74: Multi-Client Fan-Out (3/3 plans) — completed 2026-04-15
- [x] Phase 75: CLI Status Bar (3/3 plans) — completed 2026-04-15
- [x] Phase 76: TUI Foundation (3/3 plans) — completed 2026-04-15
- [x] Phase 77: TUI Session Operations (4/4 plans) — completed 2026-04-15
- [x] Phase 78: TUI Remote & QR (3/3 plans) — completed 2026-04-15

</details>

<details>
<summary>✅ v2.1 Bug Fixes & UX (Phases 79-82) — SHIPPED 2026-04-17</summary>

- [x] Phase 79: Settings Persistence & Path Browsing (2/2 plans) — completed 2026-04-16
- [x] Phase 80: Tailscale Detection (2/2 plans) — completed 2026-04-16
- [x] Phase 81: Banner Notifications (2/2 plans) — completed 2026-04-16
- [x] Phase 82: Minimize to Tray (2/2 plans) — completed 2026-04-17

</details>

<details>
<summary>✅ v3.0 Session Lifecycle & TUI Polish (Phases 83-86) — SHIPPED 2026-04-19</summary>

- [x] Phase 83: Settings UI Alignment (1/1 plans) — completed 2026-04-19
- [x] Phase 84: Session Auto-Close (3/3 plans) — completed 2026-04-19
- [x] Phase 85: Quit Confirmation Modal (2/2 plans) — completed 2026-04-19
- [x] Phase 86: TUI Visual Polish (3/3 plans) — completed 2026-04-19

</details>

<details>
<summary>✅ v3.1 Security Hardening (Phases 87-90) — SHIPPED 2026-05-03 (v3.1.0, closes Issue #35)</summary>

- [x] **Phase 87: Capability-Based Session Authorization** — Server-issued, signed capability tokens gate session listing, metadata, WebSocket access, and write permission. Tailnet membership is no longer sufficient. (UAT complete 2026-04-21)
- [x] **Phase 88: WebSocket Handshake Security** — Strict Origin allowlist (Tailscale FQDN, local-mode host, same-origin); cross-site upgrades return 403. Includes Wails desktop webview origin (`wails://wails`) — both production-darwin and dev-mode patterns. (UAT complete 2026-05-02 against rc3 + rc5 dmg)
- [x] **Phase 89: Vendored Terminal Assets + CSP** — xterm JS/CSS embedded in binary; strict CSP on all three HTML routes (`script-src 'self'`, `connect-src 'self' wss://<host>`, `style-src 'self' 'unsafe-inline'` per D-09). Zero CDN fetches verified across 1779 requests. (UAT complete 2026-05-02)
- [x] **Phase 90: Release Pipeline Hardening** — All third-party Actions SHA-pinned; build tools pinned via `tools.go`; release.yml split into validate → build-{macos,windows,linux} → sign-macos (gated by required-reviewer rule) → publish; SLSA L2 attestations verified before codesigning. (Pipeline E2E proven through 5 rc cycles before v3.1.0 final tag)

Distribution follow-ups deferred to a future milestone (see `.planning/deferred/91-distribution-pipeline-followups/91-CONTEXT.md`):
  - 91-A: Switch `release.yml` from `GITHUB_TOKEN` to PAT so `release.published` auto-triggers `distribute.yml`
  - 91-B: Fix `submit-winget` step's templating to handle `workflow_dispatch` events (use `RELEASE_TAG` env var instead of `github.event.release.tag_name`)
  - 91-C: One-time WinGet first-submission to `microsoft/winget-pkgs` (use `wingetcreate new`, not `update`)

</details>

<details open>
<summary>🚧 v3.2 Plugin Suite (Phases 92-99) — IN PROGRESS (closes Issue #36)</summary>

> Phase 91 is reserved for the deferred v3.1 distribution-pipeline follow-ups (`.planning/deferred/91-distribution-pipeline-followups/`); v3.2 starts at Phase 92.

- [ ] **Phase 92: Plugin Settings Foundation** — Daemon `PluginSettings` struct, `Get/SetPluginSettings` Wails RPC, `settings:plugins` runtime event, `PluginsSection.tsx` shell with disabled toggles, v3.1→v3.2 settings.json migration test. No addon-loading work.
- [ ] **Phase 93: Vendoring Discipline + Web Parity for Already-Shipping Addons** — Migrate webgl/unicode11/clipboard onto reconcile pattern; vendor all three for the web page (none vendored today); generalize `vendor_drift_test.go` regex to enforce versions for every `@xterm/addon-*` package; capability-gated `/api/plugin-config` endpoint.
- [ ] **Phase 94: Search Addon + Find Bar (Desktop + Web)** — Vendored `@xterm/addon-search`; floating find bar matching BannerStack vocabulary; Cmd-F (focus-conditioned) / Esc / Enter / Shift-Enter / Cmd-G keybindings; match count; per-flag persisted defaults; web parity for find bar.
- [ ] **Phase 95: Web-Links Addon + Security Hardening** — v3.1-rigor security gate: vendored `@xterm/addon-web-links`; strict scheme allowlist (`https`, `http`, `mailto`); platform-aware Cmd/Ctrl-click activation; OSC 8 hover href display + spoof warning; IDN/Punycode click confirmation; `BrowserOpenURL` (desktop) / `noopener,noreferrer` `window.open` (web).
- [ ] **Phase 96: Image Addon + CSP Audit** — Pre-phase research subtask audits `addon-image.js` for `URL.createObjectURL`/`new Worker(`/`blob:` usage BEFORE wiring; vendored addon with `storageLimit: 16` MB override; Settings advanced reveal for storage cap; multi-client byte-fidelity replay regression test.
- [ ] **Phase 97: Serialize Addon + Save-Session UX** — Vendored `@xterm/addon-serialize`; "Save Terminal As…" tab right-click action via Wails `SaveFileDialog`; text-only output in v3.2 (HTML deferred); explicit secrets-warning tooltip; no auto-save / no on-disk capture without explicit gesture.
- [ ] **Phase 98: Progress Addon (P2 — cuttable)** — Vendored `@xterm/addon-progress` with default OFF; OSC 9;4 progress events route to per-tab progress underline + tray aggregate quartile glyph. Explicitly cuttable if Phases 95 or 96 over-run.
- [ ] **Phase 99: Settings UI Polish + Migration + Final CSP Audit (Release Gate)** — Polished per-plugin captions ("Applies to new sessions you create" + post-toggle BannerStack on unicode11/image); per-plugin advanced disclosures; three-state Save reuse; `schemaVersion: 2` migration test green; cross-browser CSP e2e (Chromium + Safari + Firefox); iPad Safari Tailscale UAT.

</details>

## Phase Details

### Phase 87: Capability-Based Session Authorization
**Goal**: Tailnet reachability no longer grants session access; only explicitly granted, capability-token-bearing clients can list, view, or drive a session, and write permission is a server-controlled property of that capability.
**Depends on**: Nothing (first v3.1 phase; builds on shipped v3.0 relay + web server)
**Requirements**: SEC-01, SEC-02, SEC-03, SEC-04, SEC-05
**Success Criteria** (what must be TRUE):
  1. A user on the tailnet who has not been granted a specific session cannot enumerate sessions via `GET /api/sessions` — the response is rejected without a valid capability token, even though the request reaches the server over Tailscale.
  2. A user with a valid capability for session A cannot open session B's WebSocket or metadata endpoint with that same capability — the server rejects the request because the capability is bound to a specific session ID.
  3. Creating a new session while the web server is running does not automatically expose it; the session is only reachable after the user explicitly grants share access, at which point the daemon returns a signed capability-bearing URL.
  4. A read-only capability rejects `MsgInput` frames at the relay even if the client omits `?readonly=1` or reconnects without it — write permission is determined by the capability, not by the client or query string.
  5. Capability tokens survive daemon restart (signing key persisted alongside existing `settings.json`) so that previously-shared links remain valid without regenerating URLs.
**Plans**: 6 plans (6 waves)
  - [x] 87-01-test-infrastructure-PLAN.md — Wave 0 test scaffolds for SEC-01..SEC-05 (Complete 2026-04-20)
  - [x] 87-02-capability-core-PLAN.md — internal/capability package (sign/verify/keystore/joincode) (Complete 2026-04-20)
  - [x] 87-03-webserver-enforcement-PLAN.md — requireCapability middleware + route wiring + relay readonly source (Complete 2026-04-20)
  - [x] 87-04-daemon-api-PLAN.md — SEC-01 auto-enable removal + IPC + Wails bindings (Complete 2026-04-20)
  - [x] 87-05-frontend-ui-PLAN.md — Share panel + Regenerate key modal + Settings Security section (Complete 2026-04-20)
  - [x] 87-06-web-pages-integration-PLAN.md — Dashboard landing + Join page + terminal caret suppression (Complete 2026-04-20)

### Phase 88: WebSocket Handshake Security
**Goal**: Cross-site WebSocket hijacking is blocked at the handshake; only browsers whose `Origin` matches the server's own serving origin can complete the upgrade.
**Depends on**: Phase 87 (capability check runs after Origin check; both must pass)
**Requirements**: SEC-06
**Success Criteria** (what must be TRUE):
  1. A WebSocket upgrade request arriving with an `Origin` header outside the server's allowlist (Tailscale FQDN serving URL, local-mode host URL, and configured same-origin) is rejected at handshake with a 403/close before any capability check runs.
  2. The terminal page served by the app itself completes the WebSocket upgrade successfully in both Tailscale mode (FQDN origin) and local-network-fallback mode (self-signed HTTPS host origin) without user-visible regressions.
  3. A WebSocket upgrade request with no `Origin` header (non-browser client) follows a documented, explicit policy (allowed only with a valid capability token, or rejected) rather than the previous accept-all default.
  4. The `OriginPatterns: ["*"]` / `InsecureSkipVerify: true` accept-all configuration is gone from the code path — a regression test fails if it is reintroduced.
**Plans**: 2 plans (1 wave — parallel; Plan 01 webserver + Plan 02 relay share no files)
  - [x] 88-01-PLAN.md — Webserver Origin middleware + route wiring + library-layer allowlist + regression guard (Wave 1)
  - [x] 88-02-PLAN.md — Relay loopback-only OriginPatterns + InsecureSkipVerify removal + regression guard (Wave 1, parallel with 01)
**UI hint**: no (backend-only — SC-2 manual UAT validates existing terminal page surface)

### Phase 89: Vendored Terminal Assets + CSP
**Goal**: The interactive terminal page — the one with command-execution consequences — loads only from the embedded binary and is protected by a Content-Security-Policy that blocks inline/remote script injection.
**Depends on**: Nothing (can run in parallel with Phases 87/88; independent subsystem)
**Requirements**: SEC-07, SEC-08
**Success Criteria** (what must be TRUE):
  1. The terminal page renders correctly with network access to `cdn.jsdelivr.net` fully blocked — xterm JS and CSS are served from the app binary itself, not from a third-party CDN at runtime.
  2. The HTML response for the terminal page carries a `Content-Security-Policy` header that restricts `script-src` and `style-src` to `'self'` and restricts `connect-src` to `'self'` plus the explicit WebSocket origin; no `unsafe-inline` or `*` wildcards remain.
  3. Browser devtools shows zero requests to `cdn.jsdelivr.net` (or any third-party origin) during normal terminal session use — attach, resize, scrollback, detach.
  4. The web dashboard and terminal page pass the CSP without console violations on all supported browsers (Chromium-based + Safari) in both Tailscale-mode FQDN serving and local-network-fallback HTTPS serving.
**Plans**: 5 plans (3 waves)
  - [x] 89-01-PLAN.md — Vendor xterm files (xterm.js, xterm.css, addon-fit.js) under web/assets/xterm/ + VERSION manifest + pnpm-lock.yaml drift test (Wave 1)
  - [x] 89-02-PLAN.md — Extract inline <script>/<style> from terminal.html, dashboard.html, join.html into web/assets/*.{js,css} + swap CDN URLs (Wave 1, parallel with 01)
  - [x] 89-03-PLAN.md — cspHeaders middleware on *WebServer (per-request BaseURL composition, fail-closed) + 8 unit tests (Wave 1, parallel with 01 and 02)
  - [x] 89-04-PLAN.md — Extend web/embed.go + wire /assets/ FileServerFS route + wrap HTML handlers with cspHeaders + integration tests (D-18's 5 assertions × 3 routes) + source-grep regression guard (D-17) (Wave 2)
  - [x] 89-05-PLAN.md — chromedp e2e CSP-violation test (//go:build e2e) + 89-HUMAN-UAT.md (Safari + local-network-fallback + live-session network audit) + blocking human checkpoint (Wave 3)
**UI hint**: yes

### Phase 90: Release Pipeline Hardening
**Goal**: Untrusted third-party code can no longer execute during a job that holds macOS signing, notarization, or publish credentials; build tools are reproducible; a compromised floating tag cannot silently ship malicious artifacts.
**Depends on**: Nothing (independent from runtime phases 87-89; CI/CD surface only)
**Requirements**: SEC-09, SEC-10, SEC-11
**Success Criteria** (what must be TRUE):
  1. Every third-party GitHub Action referenced in `.github/workflows/` resolves to an immutable commit SHA; a grep for `@main`, `@master`, or unpinned branch refs across workflow files returns zero results.
  2. Every Go build tool the workflows or `build.sh` install (including `wails` and `nfpm`) is pinned to an exact version; `go install tool@latest` does not appear in any workflow or build script.
  3. The release pipeline has two separate jobs: an unsigned build job that produces artifacts and has no access to signing, notarization, or publish secrets; and a signing/publish job that consumes those artifacts and is the only job that can read `MACOS_CERT_P12`, `APPLE_ID_APP_PASSWORD`, `WINGET_TOKEN`, `TAP_DEPLOY_TOKEN`, and `RELEASE_PLEASE_TOKEN`.
  4. A dry-run release (or a test tag) successfully produces signed, notarized macOS artifacts and publishes to GitHub releases + Homebrew tap through the new split pipeline — proving the restructure is functionally equivalent to the v3.0 pipeline, not just theoretically safer.
**Plans**: 6 plans (6 waves)
  - [x] 90-01-scaffolding-PLAN.md — Wave 0 scaffolding: grep-gate script + build-script.test.sh Section 12 + tap-branch setup runbook
  - [x] 90-02-tools-go-dependabot-PLAN.md — tools.go + go.mod (nfpm, wails v2.12.0) + .github/dependabot.yml
  - [x] 90-03-build-yml-sha-pins-PLAN.md — SHA-pin build.yml + release-please.yml + replace wails@latest in build.sh with go-list pattern (Complete 2026-04-24)
  - [x] 90-04-release-yml-split-PLAN.md — release.yml three-stage split (build → sign-macos → publish) + tar-before-upload + internal/release attestations + TAP_DEPLOY_TOKEN fix + rc draft
  - [x] 90-05-distribute-yml-wingetcreate-PLAN.md — distribute.yml SHA-pin + tap rc-branch routing + swap winget-releaser for wingetcreate on windows-latest + rc winget skip
  - [ ] 90-06-e2e-rc-verification-PLAN.md — human-checkpoint: cut v3.1.0-rc1 tag + observe pipeline + external gh attestation verify + distribute rc-branch + UAT sign-off (autonomous: false)

### Phase 92: Plugin Settings Foundation
**Goal**: A returning v3.1 user opens v3.2, finds a Plugins section in Settings, sees plugin defaults populated correctly, and the daemon→Wails→React→TerminalPanel propagation pipeline is fully wired and exercised — with no addon-loading work behind any toggle yet.
**Depends on**: Nothing (foundation phase; v3.1 shipped)
**Requirements**: PLUG-01, PLUG-02, PLUG-03, PUI-01
**Success Criteria** (what must be TRUE):
  1. A v3.1 `settings.json` fixture upgraded into v3.2 lands with sensible plugin defaults populated (no zero-value addons-disabled state, no zero-value `storageLimit`) and `schemaVersion: 2` written; a fixture-based migration test asserts this and is green in CI.
  2. User opens Settings → sees a new Plugins section listing all v3.2 plugins (WebGL, Unicode 11, Search, Web Links, Inline Images, Serialize, Clipboard, Progress) with name, short description, and a (currently inert) enable/disable toggle each.
  3. Toggling any plugin and pressing Save persists the choice via the existing daemon `settings.json` mechanism and survives both GUI restart and daemon restart (verified by reading the settings.json file on disk and reopening the app).
  4. A plugin-state change emits a `settings:plugins` Wails runtime event observed by `App.tsx`, which threads `pluginConfig` as a prop into every open `TerminalPanel` — no app restart required for the propagation pipeline (addons not yet wired; the pipeline exists end-to-end).
**Plans**: 3 plans (3 waves)
  - [x] 92-01-PLAN.md — Daemon PluginSettings struct + defaults-merge load + engine RPC + HTTP routes + DaemonClient + fixture migration test (Wave 1) — covers PLUG-01, PLUG-02
  - [x] 92-02-PLAN.md — Wails (*App).GetPluginSettings + (*App).SetPluginSettings + settings:plugins EventsEmit + regenerated TS bindings (Wave 2) — covers PLUG-03 RPC half
  - [x] 92-03-PLAN.md — PluginsSection 8-toggle UI + SettingsTab insertion + App.tsx EventsOn subscription + TerminalPanel inert-prop wiring (Wave 3) — covers PUI-01 + PLUG-03 subscription half

### Phase 93: Vendoring Discipline + Web Parity for Already-Shipping Addons
**Goal**: The three already-shipping desktop addons (webgl, unicode11, clipboard) are migrated under the new reconcile pattern AND vendored same-origin for the web-served terminal page (where none are vendored today), with `vendor_drift_test.go` extended into a load-bearing CI gate that enforces version parity for every `@xterm/addon-*` package.
**Depends on**: Phase 92 (foundation pipeline + Settings UI shell required)
**Requirements**: PLUG-04, WGL-01, WGL-02, WGL-03, WGL-04, U11-01, U11-02, CLIP-01, CLIP-02, WEB-01, WEB-02, WEB-03
**Success Criteria** (what must be TRUE):
  1. User toggles WebGL in Settings and the change applies live to all open desktop terminals (hot-swap both directions) without a session restart; toggling Unicode 11 displays an inline italic caption "Applies to new sessions you create" and is honored at next-session create time only.
  2. Web-served Tailscale terminal page renders with WebGL renderer, Unicode 11 width tables, and OSC 52 clipboard support all loaded from same-origin vendored assets under `web/vendor/xterm/addons/`; browser devtools shows zero CDN requests during a full attach/resize/scrollback session, and the strict v3.1 CSP (`script-src 'self'`) reports zero violations.
  3. WebGL context loss (induced via `WEBGL_lose_context.loseContext()` in DevTools, system sleep/wake, or an iPad Safari background/foreground) automatically falls back to the DOM renderer with scrollback intact, no auto-retry loop, and a one-shot BannerStack toast informing the user; software-rasterized WebGL contexts (SwiftShader, llvmpipe, ANGLE-software) are detected at startup and the DOM renderer is used preemptively.
  4. A web-served plugin-config change applies to all connected web clients without a manual page reload (for hot-swappable plugins) via the new `/api/plugin-config` endpoint, which is gated by the same v3.1 SEC-* capability-token model that protects every other web-served route.
  5. CI fails (red, blocking) if `frontend/package.json` and `web/vendor/xterm/VERSION` disagree on the version of any `@xterm/addon-*` package — the generalized `vendor_drift_test.go` regex covers every addon, not just `addon-fit`.
**Plans**: 5 plans (3 waves)
  - [x] 93-01-PLAN.md — Generalize vendor_drift_test.go regex + min-count guard (Wave 1, WEB-02)
  - [x] 93-02-PLAN.md — Vendor 3 addon UMD bundles + VERSION + embed.go + terminal.html script tags (Wave 1, WEB-01/WGL-04/U11-02/CLIP-01)
  - [ ] 93-03-PLAN.md — Lift Phase 92 inert-prop invariant; TerminalPanel hot-swap + WebGLRecoveryBanner + isSoftwareWebGL + italic caption (Wave 2, WGL-01/02/03 + U11-01 + CLIP-02 desktop)
  - [ ] 93-04-PLAN.md — /api/plugin-config endpoint + capability gate + web terminal.js conditional addon loading + context-loss banner (Wave 2, PLUG-04/WEB-03/U11-02/CLIP-02)
  - [ ] 93-05-PLAN.md — Playwright e2e specs (vendor parity, CSP, plugin hot-swap) + iPad Safari UAT script + VALIDATION sign-off (Wave 3, WEB-02 e2e)
**UI hint**: yes

### Phase 94: Search Addon + Find Bar (Desktop + Web)
**Goal**: User can open a polished find bar with Cmd-F in any desktop or web terminal, search a 10,000-line scrollback without UI lockup, and the find bar visual treatment matches AgentHub's BannerStack vocabulary.
**Depends on**: Phase 93 (reconcile pattern + vendoring discipline + web pipeline)
**Requirements**: SRC-01, SRC-02, SRC-03, SRC-04, SRC-05
**Success Criteria** (what must be TRUE):
  1. User can open a find bar with Cmd-F (focus-conditioned: only when the xterm DOM is `document.activeElement`, so browser find still works for non-terminal page text) and dismiss with Esc; behavior is identical on desktop and on web-served Tailscale terminal pages.
  2. Find bar supports next-match (Enter / Cmd-G), previous-match (Shift-Enter / Cmd-Shift-G), match count display ("3 of 12"), and toggleable regex / case-sensitive / whole-word options; per-flag defaults persist across sessions via the daemon settings (`SearchConfig`).
  3. A search across a 10,000-line scrollback fixture completes without UI lockup (no "page unresponsive" dialog, no >1s frame budget breach measured in DevTools Performance); long-running regex searches can be cancelled by closing the find bar.
  4. The find bar visual treatment matches AgentHub's BannerStack vocabulary: TokyoNight palette, 200ms slide-in/out animation, and theme-aware match highlight via `theme.selectionBackground` (works across all 138 curated themes).
**Plans**: TBD
**UI hint**: yes

### Phase 95: Web-Links Addon + Security Hardening
**Goal**: Clickable URLs ship with v3.1-WS-Origin-allowlist rigor: no scheme outside an explicit allowlist becomes clickable, no link can be activated by accidental single-click, and OSC 8 / IDN / typosquat phishing primitives are detected and surfaced before navigation.
**Depends on**: Phase 93 (reconcile pattern + web vendoring pipeline)
**Requirements**: LNK-01, LNK-02, LNK-03, LNK-04, LNK-05, LNK-06
**Success Criteria** (what must be TRUE):
  1. Plain `https://`, `http://`, and `mailto:` URLs in terminal output are detected and made clickable; `file://`, `javascript:`, and any other scheme is never made clickable by default. A scheme-allowlist regression test fails (red) if an attacker-supplied `javascript:` URL becomes clickable.
  2. Activating a link requires Cmd-click on macOS / Ctrl-click on Linux/Windows by default (configurable in Settings); single-click never activates a link by default; a hover tooltip displays the actual resolved href in real-time on every link, including OSC 8 hyperlinks where display text differs from href.
  3. OSC 8 hyperlinks where display text differs from href, IDN/Punycode URLs, and known typosquat patterns trigger an explicit click-confirmation popover showing the full resolved URL before navigation; a fixture test exercising `https://gооgle.com` (Cyrillic) and an OSC 8 with mismatched display-vs-href fails (red) if the popover does not appear.
  4. On desktop, link activation routes through Wails `BrowserOpenURL` (links open in the user's default browser, never inside the WebView); on web-served sessions, links open via `window.open(url, '_blank', 'noopener,noreferrer')` (no current-tab navigation, ever — verified by a regression test).
  5. User can enable/disable web-links in Settings; toggling applies live to all open terminals (already-rendered links update on next refresh) without a session restart.
**Plans**: TBD
**UI hint**: yes

### Phase 96: Image Addon + CSP Audit
**Goal**: Inline sixel + iTerm2 IIP rendering ships with the heaviest addon and the only one that might require CSP amendment — gated by a mandatory pre-phase audit of `addon-image.js` source for `URL.createObjectURL` / `new Worker(` / `blob:` usage, with a dedicated multi-client byte-fidelity replay regression test and a tab-OOM guard via a 16 MB storage cap.
**Depends on**: Phase 93 (vendoring pipeline + web parity)
**Requirements**: IMG-01, IMG-02, IMG-03, IMG-04
**Success Criteria** (what must be TRUE):
  1. Pre-phase research subtask reads `frontend/node_modules/@xterm/addon-image/lib/addon-image.js` source and produces a written finding (committed to phase RESEARCH.md) on whether `URL.createObjectURL`, `new Worker(`, `blob:`, or `data:` script construction is present; CSP is amended (matching v3.1 D-09 documentation rigor) only if the finding requires it, and the e2e CSP zero-violation suite is green on Chromium + Safari + Firefox after any amendment.
  2. User enables inline image support in Settings (default ON), the toggle is clearly marked as "applies to new sessions you create", and a new session emitting a sixel or iTerm2 IIP escape sequence (e.g. `chafa --format=iterm2 chart.png`) renders the image inline both on desktop and on web-served Tailscale terminal pages.
  3. Per-terminal sixel/IIP storage is hard-capped at 16 MB of decoded RGBA by default (overriding upstream's 100 MB default); user can adjust the cap via an Advanced disclosure in Settings; a regression test loading a 50 MB sixel fixture confirms FIFO eviction at the cap and no tab OOM.
  4. A second client joining a session mid-stream after the first client has rendered an image receives a correctly-rendered image during scrollback replay (multi-client byte-fidelity audit of `internal/relay/` confirms no line-based buffering or escape filtering corrupts sixel bytes).
**Plans**: TBD
**UI hint**: yes

### Phase 97: Serialize Addon + Save-Session UX
**Goal**: User can right-click a terminal tab, choose "Save Terminal As…", and export the visible scrollback as a `.txt` file via Wails save dialog — with explicit secrets-warning copy and zero auto-save / zero on-disk capture without an explicit user gesture.
**Depends on**: Phase 92 (Settings persistence pipeline; otherwise independent)
**Requirements**: SER-01, SER-02, SER-03
**Success Criteria** (what must be TRUE):
  1. User right-clicks a terminal tab → chooses "Save Terminal As…" → Wails `SaveFileDialog` opens → confirms a path → a `.txt` file is written containing the full visible scrollback (text-only output; HTML output is explicitly out of scope for v3.2 and tracked as SER-FUT-01).
  2. Settings tooltip on the Serialize toggle reads (verbatim or near-verbatim): "Saved files include any secrets, tokens, or sensitive data printed in the session." Toggle defaults to ON for the addon-as-library; serialize never auto-saves or auto-runs.
  3. No on-disk capture of session state occurs without an explicit user action in v3.2 — a regression test (or scope-discipline review checklist item) confirms there is no timer-driven serialization, no graceful-shutdown serialization, and no settings option that enables auto-save.
**Plans**: TBD
**UI hint**: yes

### Phase 98: Progress Addon (P2 — Cuttable)
**Goal**: OSC 9;4 progress reporting from running CLIs surfaces as a per-tab progress underline and a tray-icon aggregate quartile glyph — shipped default OFF in v3.2 and explicitly cuttable if Phases 95 or 96 over-run.
**Depends on**: Phase 92 (Settings persistence pipeline)
**Requirements**: PRG-01, PRG-02, PRG-03
**Success Criteria** (what must be TRUE):
  1. Phase is explicitly cuttable: if Phases 95 (web-links security) or 96 (image + CSP) over-run their scope, this entire phase can be deferred to v3.3 with no impact on v3.2 release readiness — its absence does not block any other phase.
  2. User enables OSC 9;4 progress support in Settings (default OFF in v3.2; the toggle copy notes the default flips to ON in v3.3 after field validation); a CLI emitting OSC 9;4 progress sequences (e.g. `pip install`, an AI CLI reporting long-running task percent) shows a subtle progress underline on its tab in the tab strip.
  3. With progress enabled, the system tray icon reflects an aggregate progress glyph (quartile indicator) summarizing across all sessions emitting progress; updates do not cause tray icon flicker or excessive system-tray-API churn.
**Plans**: TBD
**UI hint**: yes

### Phase 99: Settings UI Polish + Migration + Final CSP Audit (Release Gate)
**Goal**: v3.2 ships with a polished Plugins section in Settings, the v3.1→v3.2 settings.json migration verified green on a real returning-user fixture, and the CSP zero-violation e2e suite green on Chromium + Safari + Firefox + iPad Safari Tailscale UAT — i.e. v3.2 is ready to release.
**Depends on**: Phases 92, 93, 94, 95, 96, 97, and (if shipped) 98 — release-gate phase, sequential.
**Requirements**: PUI-02, PUI-03, PUI-04
**Success Criteria** (what must be TRUE):
  1. Toggles for plugins that cannot hot-swap (Unicode 11, Inline Images) display an inline italic caption "Applies to new sessions you create" directly under the toggle; toggling them surfaces a one-shot BannerStack confirmation telling the user to open a new session to see the change. A user-facing UI review (3+ test users or a structured walkthrough) confirms the affordance is unambiguous.
  2. Plugins with meaningful runtime configuration — Search (defaults regex/case/word), Web-Links (Cmd-vs-Ctrl click modifier and confirmation policy), Inline Images (`storageLimit`) — expose those options via an inline `<details>` disclosure under their toggle; the Plugins section reuses the existing three-state Save button (idle/saving/saved) and the existing `daemonSettings` persistence mechanism — no new save infrastructure is introduced.
  3. The settings.json migration test loads a real v3.1 fixture (`tests/fixtures/settings_v3.1.json`), upgrades it through the v3.2 daemon, and asserts that all plugin defaults are populated (no zero values), `schemaVersion: 2` is written, and the migration is idempotent on a second run.
  4. The CSP zero-violation e2e suite is green on Chromium + Safari + Firefox; iPad Safari Tailscale UAT (real device, not emulator) reports zero CSP violations and zero CDN requests during a full attach/render/scrollback/detach session with all v3.2 plugins enabled.
**Plans**: TBD
**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1-6 | v1.0 | 19/19 | Complete | 2026-03-19 |
| 7-13 | v1.1 | 13/13 | Complete | 2026-03-20 |
| 14-18 | v1.2 | 10/10 | Complete | 2026-03-23 |
| 19-26 | v1.3 | 15/15 | Complete | 2026-03-25 |
| 27-29 | v1.4 | 3/3 | Complete | 2026-03-25 |
| 30-34 | v1.5 | 6/6 | Complete | 2026-03-26 |
| 35 | v1.6 | 1/1 | Complete | 2026-03-31 |
| 36-43 | v1.7 | 10/10 | Complete | 2026-04-03 |
| 44-48 | v1.8 | 9/9 | Complete | 2026-04-06 |
| 49-54 | v1.9 | 14/14 | Complete | 2026-04-08 |
| 55-56 | v1.10 | 3/3 | Complete | 2026-04-08 |
| 57-62 | v1.11 | 9/9 | Complete | 2026-04-10 |
| 63-66 | v1.12 | 4/4 | Complete | 2026-04-11 |
| 67-69 | v1.13 | 5/5 | Complete | 2026-04-12 |
| 70-73 | v1.14 | 9/9 | Complete | 2026-04-14 |
| 74-78 | v2.0 | 16/16 | Complete | 2026-04-16 |
| 79-82 | v2.1 | 8/8 | Complete | 2026-04-17 |
| 83 | v3.0 | 1/1 | Complete    | 2026-04-19 |
| 84 | v3.0 | 3/3 | Complete    | 2026-04-19 |
| 85 | v3.0 | 2/2 | Complete    | 2026-04-19 |
| 86 | v3.0 | 3/3 | Complete    | 2026-04-19 |
| 87 | v3.1 | 6/6 | Complete | 2026-04-20 |
| 88 | v3.1 | 2/2 | Complete | 2026-04-22 |
| 89 | v3.1 | 5/5 | Complete    | 2026-04-23 |
| 90 | v3.1 | 5/6 | In Progress|  |
| 92 | v3.2 | 3/3 | Complete (pending verify) | 2026-05-04 |
| 93 | v3.2 | 2/5 | In Progress|  |
| 94 | v3.2 | 0/0 | Not started | — |
| 95 | v3.2 | 0/0 | Not started | — |
| 96 | v3.2 | 0/0 | Not started | — |
| 97 | v3.2 | 0/0 | Not started | — |
| 98 | v3.2 | 0/0 | Not started | — |
| 99 | v3.2 | 0/0 | Not started | — |

---
*Full v1.0 details: .planning/milestones/v1.0-ROADMAP.md*
*Full v1.1 details: .planning/milestones/v1.1-ROADMAP.md*
*Full v1.2 details: .planning/milestones/v1.2-ROADMAP.md*
*Full v1.3 details: .planning/milestones/v1.3-ROADMAP.md*
*Full v1.4 details: .planning/milestones/v1.4-ROADMAP.md*
*Full v1.5 details: .planning/milestones/v1.5-ROADMAP.md*
*Full v1.6 details: .planning/milestones/v1.6-ROADMAP.md*
*Full v1.7 details: .planning/milestones/v1.7-ROADMAP.md*
*Full v1.8 details: .planning/milestones/v1.8-ROADMAP.md*
*Full v1.9 details: .planning/milestones/v1.9-ROADMAP.md*
*Full v1.10 details: .planning/milestones/v1.10-ROADMAP.md*
*Full v1.11 details: .planning/milestones/v1.11-ROADMAP.md*
*Full v1.12 details: .planning/milestones/v1.12-ROADMAP.md*
*Full v1.13 details: .planning/milestones/v1.13-ROADMAP.md*
*Full v1.14 details: .planning/milestones/v1.14-ROADMAP.md*
*Full v2.0 details: .planning/milestones/v2.0-ROADMAP.md*
*Full v2.1 details: .planning/milestones/v2.1-ROADMAP.md*
*Full v3.0 details: .planning/milestones/v3.0-ROADMAP.md*
