# Milestones

## v4.2 Funnel Sharing & Polish (Shipped: 2026-07-09)

**Phases completed:** 13 phases, 61 plans, 125 tasks

**Key accomplishments:**

- Injectable `funnelClient` interface seam with `EnableFunnel`/`DisableFunnel`/`FunnelBaseURL`/`ClearLingeringFunnel` methods and dual-origin WebSocket allowlist for Tailscale Funnel guests.
- Per-session `funnelSessions` reference-count map, `POST /sessions/{id}/funnel` endpoint with auto-expiry timers, all five teardown triggers wired through `disableFunnelForSession`, Funnel-aware URL builders in `issueCapabilitiesForSession` and `handleExchangeJoinCode`, and daemon-restart `ClearLingeringFunnel` on startup.
- `App.SetSessionFunnel` Wails bound method + `DaemonClient.SetSessionFunnel` client delegation + `SessionInfo.FunnelActive bool` mirror field with copy-loop propagation in `App.ListSessions`, completing the FNL-01 enable path end-to-end.
- Fixed two live-UAT defects: Funnel proxy target changed from `https://localhost:<port>` to `https+insecure://<bindIP>:<port>` (closes HTTP 502 for all external Funnel guests), and handleDeleteSession now synchronously tears down Funnel on the explicit-kill path via runSessionExitCleanup (closes stale ref-count leak)
- FNL-03 proxy-target layer closed: AgentHub adds a plain-HTTP loopback listener (127.0.0.1, ephemeral) as the Funnel hop-2 target, replacing the proven-dead https+insecure://<bindIP>:<tlsPort> target from 165-04
- 1. [Rule 1 - Bug] Fixed 12 test fixtures missing funnelActive field
- 1. Inline execution instead of subagent
- SessionCard.tsx
- 1. Inline execution + panel-before-modal order
- Persisted, socket-reachable `NotifyOnWaiting` boolean setting (engine + REST + client) mirroring `StartMinimized` exactly, defaulting OFF with zero schema-version impact.
- Identifier-aware `sendNotification(identifier, title, body)` on darwin/windows/linux — native UNUserNotificationCenter retained on macOS, new beeep v0.11.2 wrappers added for Windows/Linux
- `maybeNotifyWaiting` edge-detects the non-waiting->waiting transition inside the existing 5s tray poller and fires through the injected `sendNotificationFunc` seam; `GetNotifyOnWaiting`/`SetNotifyOnWaiting` Wails-bound and wired end-to-end into all 4 wailsjs binding files.
- Notify-on-waiting toggle lands in the Behavior section (default OFF, mirrors startMinimized end-to-end), and TESTING.md gains the Phase-167 Suite Manifest note, four NTF traceability rows, and Category U's M-41 manual delivery item — closing the user-visible half of NTF-04.
- Guarded `sendNotification` in `tray_objc_darwin.m`/`notification_darwin.go` with a bundle-id check (fail before dispatch) + `@try/@catch` defense-in-depth, closing the always-on tray-poller crash that killed the toast/tab-dot/Hub-glow/auto-close together under `wails dev`.
- Every native macOS notification branch (attempt, not-granted, authorization error, delivery-accepted/failed) now logs, and enabling the NotifyOnWaiting toggle proactively requests UNUserNotificationCenter authorization + registers a foreground-presentation delegate — closing the observability gap that blocked root-causing the M-41 live-delivery blocker.
- SettingsTab now subscribes to the backend's `notification:permission-denied` Wails event and renders a "System Settings > Notifications > AgentHub" remediation hint in the Behavior section — closing the frontend half of the M-41 gap where a proactively-requested, silently-denied permission had no way for the user to discover or fix it.
- Hub session cards now read 0 viewers for never-shared local sessions — `Hub.RemoteViewerCount()` filters on `Origin=="web"`, replacing the raw `SubscriberCount()` (which included the app's own internal TerminalPanel/ChatPanel/status-watcher/preview WebSocket connections) as the source for `SessionInfo.ViewerCount`.
- WebShareSessionView self-fetches /api/plugin-config and subscribes to its SSE stream when acting as a web guest, restoring live plugin hot-swap lost after the Phase 159 /sessions->/app redirect (#112), and gains a baseURL prop seam for FIX-03's remote-peer reuse.
- Opening a remote tailnet session from the Hub now opens an in-app terminal/chat tab (openWebSessionTab, per-tab baseURL/capToken) instead of launching an external browser window, reversing the Phase 146 out-of-band design (D-17, #118).
- A persisted `stayOnHubAfterCreate` daemon setting (5-hop chain mirroring Phase 167's NotifyOnWaiting) backs a new Settings → Session Behavior toggle that, when ON, skips `App.tsx`'s single `setActiveId(sessionId)` auto-switch in `createTab` so the user stays on the Hub after creating a session.
- Lifted `shareModalSession` state and its single `<SessionShareModal>` render from HubPanel to App.tsx (RESEARCH Pattern 4) so the renamed footer "Share Session" button — which now only opens the modal instead of calling `ToggleWebServing` directly — works from any local session tab, not just while the Hub tab happens to be mounted.
- `Hub.DisconnectWebViewers()` force-closes only `Origin=="web"` relay subscribers (reusing `CloseSlow` with collect-then-unlock-then-close lock discipline), wired end-to-end through a daemon-local-only `POST /sessions/{id}/disconnect-viewers` RPC to a new "Disconnect all viewers" button in `SessionShareModal`, plus a regression test proving #117's "second viewer kicks first" bug does not reproduce in current code.
- TESTING.md is now the complete, accurate canonical regression record for Phase 168: all six requirements (FIX-01..04, UX-01, UX-02) have traceability rows, M-13 reflects the shipped in-app-tab behavior, two new live-only manual checks (M-42, M-43) cover what code alone can't prove, and a stale Suite Manifest total was corrected.
- SessionShareModal gains an onShareEnabledChange(sessionId, enabled) callback, wired in App.tsx to setWebEnabled, so the footer pill tracks server-truth web-share state on every modal toggle — closing the last recorded gap in the UX-02 (#115) "no more button↔modal↔pill drift" goal.
- Remote in-app web-session tab now streams the peer PTY through the local daemon proxy (dropping the doomed direct cross-origin wss), fills full pane height via .terminal-wrapper, hides chat, and its card menu reads "Open in tab" with an in-app glyph
- Reverted the invalidated CLI-status-fallback and replaced it with a darwin-only, liveness-confirmed macsys permission probe that reports `PermissionLimited=true` instead of ever risking a false `Connected`
- SettingsTab now renders a distinct, colorblind-safe "Permission Limited" state (grant-admin / Homebrew-build guidance, never "Connected") for the honest macsys detection added in 169-01, with TESTING.md reconciled and REVIEW IN-02's macsys/App-Store/Homebrew distinction corrected
- JoinCodeManager gains IssueReusable/Revoke + a reusable-conditional Exchange delete, proven at both the unit layer and the public /join/exchange HTTP boundary — closing the FNL-08 UAT dead-end where a public share code resolved only once.
- issueCapabilitiesForSession mints a reusable public-share join code from the read-only token only, caches it per session so it never rotates on re-issue, and disableFunnelForSession revokes it on every one of the four in-process Funnel teardown triggers — closing FNL-08 at the daemon layer.
- The Share modal's Internet (public) section now shows a reusable, read-only "Public join code (reusable):" row via the existing `<CodeDisplay>` component, sourced from the same warm-up `IssueCapabilities` round-trip that already resolves the Funnel URL — closing the FNL-08 UAT dead-end where a recipient without QR/URL access had no code to type at `/join`.
- Final reconciliation pass over TESTING.md for the whole Phase 170 test surface — verified the Suite Manifest counts, phase notes, and all 5 FNL-08 traceability rows left by 170-01/02/03's incremental self-registration were already internally consistent (no drift found), and added the one missing piece: Section 5 Category X / M-46, the live off-tailnet reusable-code-join + share-lifetime-teardown manual checklist item.
- Two enforcement primitives (grant-registration gating at the real WS upgrade + Funnel-origin RW-gate check) and a custom-TTL single-use join code, closing the internet-write RCE surface at the layer that actually reaches the PTY-write path.
- Gate-minted, terminal-only public write capability (single-use code, RW consent gate, unconditional 1h expiry clamp) with a surgical teardown that never disturbs the reusable public read share, plus the D-04 fix that stops the accidental tailnet-write-cap-to-Funnel-base rebasing.
- A real `<button>` hold-to-confirm gesture (pointer + keyboard, 3000ms, timer-authoritative completion) gates `SetSessionFunnelWrite`, with a colorblind-safe FULL ACCESS badge (LockOpenIcon + notched clip-path, distinct from the read INTERNET pill) that coexists with and clears independently of the existing read share.
- Reconciled TESTING.md (corrected a 1-file Go-count omission from 171-01, added FNL-09 traceability for 171-03's frontend tests, and added the M-47 live off-tailnet public-write acceptance gate), a risk-forward Sharing Guide section for public write access, and 171-SECURITY.md — the SPEC R8 threat model synthesizing all four plans' STRIDE registers and asserting no public write path exists except the hard consent gate.
- Consolidated the Hub session card's three inconsistent metadata treatments into one `agent · origin · exposure` chip row (Sketch 001 Variant B): 7px rounded-rect quiet outlined chips for agent/origin, INTERNET/FULL ACCESS as the sole filled badges on their own right-aligned line, and a muted uptime/viewers/Connected meta line below.
- Hoisted CodeDisplay/HoldToConfirmButton into an exported `SessionShare/shared.tsx` module (zero behavior change) and added an additive `prefers-reduced-motion` plain-confirm fallback to the hold button, both proven against the full existing test suite (2365 tests green).
- Hand-rolled `role="tablist"` component (`ShareSegmentedControl`) with roving tabindex, arrow-key navigation that skips disabled segments, and a component-owned colorblind-safe danger glyph — this app's first tablist implementation.
- One reusable `ShareLinkCard` component (title · CSS-truncated URL · Copy/Open/QR · join code · scope description directly beneath) replacing the four ad-hoc hand-laid link rows in `SessionSharePanel.tsx`, with the QR always encoding the join-code exchange URL rather than the raw capability token.
- Decomposed `SessionSharePanel.tsx` into `TailnetTab`/`InternetReadOnlyTab`/`InternetFullAccessTab`, walling the public-write/command-execution flow off inside one component with a new Idle → Gate-open → Armed local state machine, plus 25 new vitest tests proving the SM-04 negative wall-off.
- SessionShareModal now a fixed control strip + swappable three-tab panel (ShareSegmentedControl + TailnetTab/InternetReadOnlyTab/InternetFullAccessTab) with a transient FunnelRiskPanel confirm view and On/Off/N-A toggle labels, replacing the old single-column SessionSharePanel stack.
- Reconciled TESTING.md's Suite Manifest (142→147 vitest files) and added the phase's first 15 SM-01..SM-08 Traceability rows, then proved the entire frontend suite (147 files / 2384 tests), `tsc --noEmit`, and `vite build` all green — closing out Phase 173's regression bookkeeping ahead of `/gsd-verify-work`.
- Closed the two source-confirmed SM-07 accessibility gaps (roving-tabindex focus-follow + colorblind-safe pending toggle label) plus the cheap same-file SM-05/WR-01 funnelOn server-truth resync, surgically, with zero changes to the passing 173-01..07 output.
- Four SHA-pinned CI-action bumps (setup-go 6.5.0, pnpm/action-setup 6.0.9, attest-build-provenance 4.1.1, action-gh-release 3.0.1) applied across build.yml/e2e.yml/release.yml on v4.2-funnel-sharing, with the corresponding Dependabot PRs #114/#113/#103/#85 closed citing this phase.
- Bumped coder/websocket 1.8.15, golang.org/x/term 0.44.0, and goreleaser/nfpm/v2 2.47.0 on v4.2-funnel-sharing — each isolated behind its own build/test gate — with go-webview2 confirmed to stay pinned at v1.0.19; PR-closing (Task 3) blocked by a runtime permission gate pending user action.
- Added 3 surgical `.github/dependabot.yml` ignore entries (wails/v2 >=2.11.0, tailscale.com >=1.100.0, actions/checkout semver-major) and closed Dependabot PRs #104/#88/#102 citing Phase 174 rationale — all three closures succeeded with no permission-classifier block.
- Live timed reproduction DISPROVED the exit-poll-deadline hypothesis — a >5-minute shared session still auto-closed its tab; BUG-03 did not reproduce, steering 175-05 to a diagnostic-logging pass instead of removing the deadline.
- Extracted the exit-poll deadline into a pure injectable-clock helper (`shouldContinuePolling`) and laid down skip-guarded RED Go tests for BUG-02 (WS close reason) and BUG-04 (alt-screen reconnect replay) that Waves 1–2 must turn GREEN.
- Added a floor-aware guest-viewport scale helper (`computeGuestViewport`) and wired `TerminalPanel.recomputeScale` to clamp at a readability floor (0.7) with a `.terminal-guest--scroll-x` horizontal-scroll CSS fallback, fixing BUG-01 (#128) where narrow-phone web-share guests saw the 80-col grid shrink to unreadable text with no floor.
- Lazy, continuously-fed VT emulator per Hub (charmbracelet/x/vt) whose `RenderSnapshot()` reconstructs alt-screen mode + current screen content for reconnecting/late-joining viewers, replacing the raw 256 KiB scrollback replay at both WS handler sites — closes BUG-04's residual blank-window bug (#119, Problem 2).
- Implemented the 175-01 DISPROVED branch: kept the fixed 300s exit-poll deadline unchanged and added structured slog diagnostics around the suspect daemon-client stall path plus every non-exit-emitting terminal branch of `pollSessionStatus`, closing the silent-give-up failure mode without a behavior change.
- Both WS write pumps now emit a fixed `conn.Close(StatusNormalClosure, "session ended")` on `hub.Done()` (closing BUG-02/#125's silent-dead-terminal gap), and the guest client renders a colorblind-safe `SessionEndedBanner` (text + `role=status` + `aria-live=polite`, never the raw `CloseEvent.reason`) instead of a frozen terminal.
- Reconciled TESTING.md's Suite Manifest to the actual working-tree test-file counts (correcting a stale, worktree-inflated 376 Go down to the real 139), registered all of Phase 175's new/extended test files in the Traceability map, added four new manual live-UAT items (M-48..M-51) for BUG-01..04 — with M-50 correctly framed as a standing regression watch reflecting BUG-03's DISPROVED live-diagnosis verdict rather than a fresh repro script — and reconciled Category P's stale "the web viewer" wording to the post-Phase-159 `/app/` React SPA architecture.
- Darwin-guarded the three Wails role menus and added a Linux-only WEBKIT_DISABLE_DMABUF_RENDERER env guard in main.go, fixing the Linux desktop GUI dead-on-arrival bug (#124/BUG-05).
- Wrapped the public guest SPA route `/app/` in the existing `ws.cspHeaders` middleware, closing the BUG-06 (#123) gap where the sole unauthenticated Funnel-exposed surface served zero Content-Security-Policy protection.
- Live dev-browser repro of GitHub #127 (Hub mini-preview char-per-row stacking) DOES NOT reproduce on the current build — each output line renders as one clipped horizontal row exactly as designed; zero code change made.
- Reconciled TESTING.md for Phase 176's three bug fixes — added manual items M-52 (BUG-05 Linux/Wayland GUI) and M-53 (BUG-06 /app/ CSP console sweep), a Section 4 traceability row for BUG-06, and a dated Suite Manifest note confirming zero net-new test files, as the single serialized editor of TESTING.md for the phase.
- Added `FunnelWriteActive` to app.go's `SessionInfo` struct and `ListSessions()` conversion loop — the sole load-bearing fix that lets the already-built FULL ACCESS badge/tab-icon/modal-resync consumers receive a real boolean.
- Added a genuine Go regression test suite (reflection-based field parity + a real ListSessions/json.Marshal round trip) that fails if app.go ever again drops a daemon funnel-exposure field — proven RED against a temporarily reintroduced pre-fix app.go and GREEN on the fixed code — then reconciled TESTING.md's Suite Manifest and FNL-09 traceability map.

---

## v4.1 Session Chat (Shipped: 2026-06-29)

**Delivered:** A per-session human-to-human chat side-channel inside AgentHub — tailnet identity + self-chosen alias, one-way `@session`→PTY-prompt bridge, daemon-persisted thread, Markdown export — working identically on the desktop GUI and the web-share browser surface. Closes Issues #79, #108, #109.

**Phases completed:** 14 phases (151–164), 54 plans, 57 tasks
**Stats:** 381 commits, 532 files changed, +51,309 / −2,306 LOC over 2026-06-25 → 2026-06-28 (4 days); git range `440aafef..fa99782b`
**Closeout type:** override_closeout — 1 accepted deferral (INSTALL-01 / M-25 clean-box Linux install runs as a release-time step after `main` is pushed to origin). Audit `passed` (45/45 reqs, 14/14 phases, 14/14 integration, 6/6 flows).

**Key accomplishments:**

- **Daemon chat store + wire contract** — JSONL-backed `ChatStore` (`~/.config/agenthub/chats/<id>.jsonl`) with 10k-cap append, restart-survival replay, path-traversal-hardened sessionID validation, Markdown export, and session-lifecycle teardown (zero orphaned files); a stable `ChatMessage`/frame contract (0x30–0x37) in `relay/protocol.go` that all surfaces serialize to.
- **Identity, presence & the `@session` bridge** — tailnet-ID + self-chosen alias attribution (`AliasStore`/`MsgAliasSet`, set live from the shared `ChatPanel` sidebar across all three surfaces), connected/typing presence, and a sanitized one-way `@session` PTY injection gated to write-capability holders with a deliberate confirm gesture.
- **Cross-surface chat UI (GUI + web, parity by construction)** — a virtualized slide-over `ChatPanel` (own relay WS, HTTP scrollback, sticky day separators, mentions, unread/mention badges on toggle + Hub card, safe `remark-gfm` Markdown — no raw HTML/XSS) shared verbatim by the GUI session tab, the Hub interactive modal, and the web-share guest.
- **Real web-share chat parity** — the actually-shared `/sessions/{id}?cap=` link now 302-redirects (after capability check) to the chat-capable `/app/` SPA, closing the false-parity gap where Phase 155 verified parity only on a surface no remote guest was ever sent to; scoped web guests get terminal + chat + cap-gated file tab and nothing else.
- **Read-only guests as full chat participants (D-06)** — reversed the SEC-01 RO chat-send gate so RO guests can post chat on every surface, while `@session` inject and raw PTY input stay byte-for-byte RO-gated (security re-reviewed, `threats_open: 0`).
- **Issue #109 screen-share semantics** — host-authority PTY arbiter replacing max-wins: guests honor a single server-pushed grid size and CSS scale-to-fit (web + desktop parity), fixing cross-viewer terminal garble.
- **Install/distribution + Settings fixes** — working Linux `scripts/install.sh` (arch-detect → release tarball → SHA256 verify), corrected winget id (`scottkw.agenthub`) and repo link, winget catalog first-submission automation (#108 Terminal-Plugins jump-link fix folded in).

---

## v4.0 Hub-First Consolidation & UI/UX Overhaul (Shipped: 2026-06-23)

**Phases completed:** 15 phases (136–150; 151 cancelled), 52 plans, 61 tasks
**Git tag:** v4.0 · **Timeline:** 2026-06-19 → 2026-06-23 (5 days)
**Closes:** #51, #65, #68, #69, #96, #97, #98, #100, #101 (completed) · #99 (closed not-planned — zoom already shipped) · #82, #95 (closed at scoping)

**Delivered:** AgentHub collapsed onto a single Hub-centric surface. The TUI and the Sessions/Remote sidebar pages are gone; their controls live in per-card Share modals and colorblind-safe indicators. The sidebar is now Home / Hub / Help / Settings. Ships the v4.0 visual redesign (Plus Jakarta Sans + JetBrains Mono, comp palette, persisted Light/Dark theme), a Chrome-style tab strip, an in-app Help page, a Google Antigravity agent, and a formal regression-test program with a CI branch-protection gate.

**Key accomplishments:**

- **Hub-first consolidation (136, 138):** removed the `agenthub tui` command + Bubble Tea infra (cross-surface parity contract narrows to GUI/CLI/web), deleted the Sessions (DaemonManagerPanel) and Remote (RemoteSessionsPanel) pages and the sidebar New-Session item; session creation now flows only through the Hub's HubFilterBar.
- **Unified Share/capability model (137, 146):** one per-card Share modal mints RO + RW cap-tokens; "Enable remote file browsing" derives permission from the presented share code; remote "Open in browser" reuses the held web-share cap (App.OpenRemoteSessionURL + daemon open-url endpoint) instead of minting a second single-use code.
- **v4.0 redesign + theming (139–142):** headless-VT styled-tail mini-previews (#96), browser-style shrink-then-scroll tab strip, the comp visual language (fonts/palette/radii/type-scale) adopted after catching and fixing a token-only false pass, a persisted Light/Dark toggle, and WebGL repaint hardening (no terminal garble on theme/tab switch). GroupSidebar ARIA rewritten to plain buttons (#97).
- **Regression-test program (143):** TESTING.md suite manifest + traceability map + path-check CI script + 17-item manual checklist, plus branch protection on `main` (4 build jobs + playwright required) — applied once CI went green.
- **Issue burndown (144–148, 150):** daemon styled-tail data race (#100), Windows files test failures (#101), the Open-Session capability bug (#98), an in-app Help page with search + external links (#69), a session-tab discoverability chevron (#68), and a Settings toggle for the shell web-share warning with path-aware shell detection (#51).
- **Google Antigravity agent (149):** `agy` wired across all surfaces — knownCLIs detection, Windows `%LOCALAPPDATA%\agy\bin` PATH, status detector, and a `#ff9e64` badge identity on card spine/chip/tab-dot — verified by live REPL UAT (#65).

---

## v3.6 Hub (Session Grid / Control Room) (Shipped: 2026-06-19)

**Phases completed:** 5 phases, 26 plans, 29 tasks
**Git tag:** v3.6 · **Closes:** Issue #78

**Delivered:** A "control room" Hub — a unified, live session grid that shows every local and remote AI-coding session as a data-accurate card, with filter/search, named groups, attention surfacing, and a card→modal interaction, all colorblind-safe and keyboard-operable.

**Key accomplishments:**

- **Hub surface + live cards (Phase 131):** new top-level Hub tab (coexists with Sessions) rendering all sessions as cards (name, CLI badge, colorblind-safe status icon, origin, viewer count, uptime/exit-code), auto-grouped by working directory, with a status filter bar (live counts) and `/`-focused search. Required a Wave-0 Go data-gap fix (WorkDir/ViewerCount/ExitCode/Duration propagated through daemon + app SessionInfo).
- **Unified grid + mini-preview + named groups (Phase 132):** throttled output-snapshot mini-previews on every card (never a live xterm — single shared poll interval), remote tailnet-peer sessions merged into the same grid, and user-defined named groups with drag-and-drop assignment (membership keyed by name+workDir, localStorage-persisted).
- **Attention system (Phase 133):** waiting/errored/non-zero-exit sessions float to the top of their group and pulse (amber `#e0af68`, BellAlertIcon + aria — color is never the sole cue), with a 1s-debounced FLIP reorder and a collapsed-group attention badge; all motion respects prefers-reduced-motion.
- **Card → modal interaction (Phase 134):** clicking a card opens a shared-element grow/shrink modal — an interactive terminal modal for normal sessions and a briefing modal for attention sessions — including a cap-gated relay WebSocket proxy for remote-session terminals (MODAL-06), with clean focus return on close.
- **Accessibility hardening (Phase 135):** full keyboard operability, prefers-reduced-motion compliance, and colorblind-safe status/attention cues verified across the surface (colorblind-safe is a release-blocking constraint, verified at the hex-constant level, not by eye).
- **Close-time daemon bugfix (commit 245414c2):** fixed macOS natural-exit exit-code capture (`Session.ReapNaturalExit()`) so stopped-err cards are reachable; TDD regression test, daemon+pty suites green with `-race`. All Phase 131/132/133 live UATs verified this session (incl. remote tailnet peer across two real machines).

**Known deferred (release-eligible, see milestones/v3.6-MILESTONE-AUDIT.md):** TUI Hub parity (#82, signed-off deferral); mini-preview issues one tail RPC per session per tick (scale-perf nuance, candidate issue); minor advisory A11Y items (WR-04 → #97); carry-forward v3.5 live UATs + operator pre-release tokens (RELEASE_PUBLISH_TOKEN, WINGET_FIRST_SUBMISSION).

---

## v3.5.1 Remote Browse Completion + Release-Gate Fix (Shipped: 2026-06-16)

**Phases completed:** 2 phases, 7 plans, 11 tasks

**Key accomplishments:**

- Confirmed RED with `-race` flag.
- Package-level per-path sync.Map mutex in WriteFileAtomic closes the stat→rename TOCTOU window — deterministic single-winner guarantee (RACE-01/RACE-03) — and zero "last-writer-wins" language remains in the remote-write proxy (RACE-02).
- 1. [Rule 1 - Bug] Updated existing checkHealth call sites in tailscale_test.go
- Wave 0 RED tests locking the discover→list→pick behavioral contract: four test files encode RB-01 no-silent-drop, RB-03 metadata-only security whitelist, and RB-05 relay-surface discover→browse path through api.RelayHandler()
- Vitest assertions locking per-peer honest states (Unreachable badge, No shareable sessions text, never false 'No remote peers found') for the colorblind-safe contract; copy updated to UI-SPEC 'Shows shareable sessions'; suite intentionally RED pending plan 04 implementation
- Open GET /api/sessions/meta webserver endpoint (tailnet-trusted, metadata-only), FetchAllPeerSessionsMeta no-silent-drop path, and GetRemoteSessionsWithMeta Wails RPC with Reachable field — all plan-01 RED tests GREEN including the RB-05 relay-surface release gate
- RemoteSessionsPanel renders honest per-peer states (unreachable badge, zero-sessions block, populated rows) per UI-SPEC colorblind contract; GetRemoteSessionsWithMeta wired in App.tsx + App.d.ts; all 87 vitest files green

---

## v3.5 File Browser — Write Operations & Editor (Shipped: 2026-06-15)

**Phases completed:** 6 phases (123-128), 27/27 plans
**Requirements:** 55/55 satisfied (FSW-01..12, CAP-01..10, EDIT-01..13, TUIW-01..07, SEC-01..07, RMW-01..06)
**Commits:** 238 | **Timeline:** 2 days (2026-06-14 → 2026-06-15)
**Source changes:** ~14.9K source LOC added across 86 files (excluding `.planning/`)
**Closes:** GitHub Issues #63 (editing + upload) + #64 (TUI edit parity). Umbrella epic #24 retires on the operator two-machine tailnet write UAT (committed checklist, RMW-06).
**Audit status:** `tech_debt` (accepted) — 55/55 reqs satisfied at code/test level, integration 98/100 PASS, no critical blockers at close.

**Key accomplishments:**

- TOCTOU-free write primitives on the v3.4 `os.OpenRoot` sandbox: atomic temp+sync+rename (no `O_TRUNC`), rename validates BOTH source AND destination, recursive delete, mkdir, and multipart upload — all guarded by a shell-RC denylist enforced on every write path and a 60s `FuzzSandboxWrite` merge gate (0 crashes); daemon Unix-socket write routes live (auth-less by design — local trust boundary); folds in carried tech-debt TD-4 (Phase 120 WR-01..05) and TD-5 (Phase 122 `ExchangeJoinCode` shim) — Phase 123 (FSW-01..12)
- New opt-in `files.write` capability (never default-on — owner enables per session, web-share viewers require a further explicit grant) gating webserver write routes behind `requireFilesWrite` + a CSRF Origin check, with `schemaVersion: 4` migration via the established defaults-merge constructor — Phase 124 (CAP-01..10)
- Milestone centrepiece: vendored CodeMirror 6 editor (zero new CSP amendments — Monaco rejected for requiring `worker-src blob:`) replacing the v3.4 plain-text preview — syntax highlighting by extension, atomic Cmd/Ctrl+S with `If-Match`/412 conflict detection, dirty-state + unsaved-changes guard, and the full create/mkdir/delete/rename/cross-directory-move/single+multi-file-upload (drag-and-drop, XHR progress, 409/413 handling) affordance suite; 51/51 cross-browser write e2e with zero CSP violations — Phase 125 (EDIT-01..13)
- TUI write parity via `$EDITOR` shell-out (`e` key, suspend/`tea.ClearScreen`/resume + unconditional refresh) plus `d`/`r`/`m` keyboard ops; the `FilesClient` interface grew 4 → 8 methods so ONE pipeline drives local AND remote writes; TUI upload formally descoped with on-screen message + follow-up Issue #82 — Phase 126 (TUIW-01..07)
- Dedicated web-share write security-hardening phase: denylist hardening + symlink-escape / privilege-escalation / CSRF / concurrent-write audits end-to-end on the most-exposed surface; `SECURITY` artifact produced; fully automated (audit/test/doc) — Phase 127 (SEC-01..07)
- Remote tailnet peer write parity proven byte-identical by 3 independent observers (daemon-proxy Go + `tui.RemoteFilesClient` Go + Playwright HTTPS browser), with 405 peer-outdated / 401 cap-expired mappings and a Phase 122 read-regression guard — Phase 128 (RMW-01..06)
- Live two-machine tailnet UAT (2026-06-15) executed: remote read+**write data path proven live** (v3.4.2 peer 405 version-gate; v3.5 peer write/mkdir/rename/delete all 200, read-back confirmed). Uncovered a 4-layer desktop-GUI remote-browse breakage invisible to the 98/100 automated suite (which never drove the relay loopback the Wails GUI uses): relay routes not mounted (**FIXED `58af6d6`** + relay-surface regression gate), discovery probe rejecting cap-protected peers (**FIXED `3508bd7`/#84`), accept-dns prerequisite (**#83**, deferred), session-enumeration-vs-cap-gating (**#86**, deferred). Share-link scope relabel (`e45ccba`/#24 UX).

**Known deferred items (carried to v3.5.1):**

- #86 — Remote Sessions panel can't enumerate a peer's sessions (`/api/sessions` is cap-gated by design D-18); blocks the desktop GUI remote-browse on-ramp. Remote read/write data path works; the discover→list→pick UX is not yet shippable. Target v3.5.1 / dedicated design pass.
- #87 — Flaky CI `TestWrite_TwoWritersIfMatchRace` (`WriteFileAtomic` If-Match re-check TOCTOU window) blocks the release gate.
- #83 — Remote file browse fails with opaque 502 when the client has Tailscale DNS disabled (`accept-dns=false`).
- #82 — TUI Files upload parity gap (formally descoped in Phase 126).
- Umbrella epic #24 remains open until the GUI remote-browse feature is usable (on-ramp lands in v3.5.1).
- Operator one-time (carry-forward): `RELEASE_PUBLISH_TOKEN` PAT + `WINGET_FIRST_SUBMISSION=true` variable (first WinGet submission).
- Live desktop/TUI visual UATs: Phase 125 editor on-screen render + CodeMirror Tab/Cmd-V; Phase 126 `$EDITOR` suspend-resume terminal restore; Phase 124 home-dir warning banner. Logic + colorblind tokens source-verified; live render is manual residue.
- Nyquist frontmatter bookkeeping: Phases 123/125/126/127 `VALIDATION.md` `nyquist_compliant:false` despite green automated coverage (advisory).
- Known deferred items at close: see STATE.md Deferred Items.

---

## v3.4 File Browser (Read-Only) + TUI Parity (Shipped: 2026-05-21)

**Phases completed:** 5 phases (118-122, including Phase 122 inserted mid-milestone), 20/21 plans
**Requirements:** 48/48 satisfied (FS-01..14, WEB-01..05, UI-01..14, TUI-01..10, REMOTE-01..05)
**Commits:** 176 | **Timeline:** 2 days (2026-05-20 → 2026-05-21)
**Source changes:** 197 files, +42,159 / -1,963 lines (includes `.planning/` documentation)
**Closes:** GitHub Issues #62 (read-only file browser) + v3.4 slice of #64 (TUI browse+preview parity). Umbrella epic #24 stays open across v3.4 + v3.5.

**Key accomplishments:**

- TOCTOU-safe sandboxed filesystem API in new `internal/files/` package: `os.OpenRoot`-based `Sandbox` (Go 1.24+ kernel-level path resolution, NOT the legacy `filepath.EvalSymlinks` + `os.Open` two-step that's exploitable by CVE-2026-27976/43998); HTTP `Handler` for `GET/HEAD /api/files/{list,stat,read}` with Range support + 5 MB read cap + 0-byte short-circuit (golang/go#54794 special case) + darwin-filter; `FuzzSandboxPath` merge gate against 40+ payload corpus (path traversal, `%2e%2e%2f`, `%252e%252e%252f`, U+FF0F fullwidth slash, U+2024 one-dot-leader, null bytes, Windows reserved device names, alt data streams, 8.3 short names, UNC variants) — Phase 118 (FS-01..14)
- `files.read` capability bit + `HasPerm` whole-token comma-split (NOT `strings.Contains`) + `requireFilesRead` middleware separated from `requireCapability` switch (avoids breaking existing relay routes); session-owner cap includes `files.read` by default, web-share viewer cap excludes by default; `SessionEngine.sessionWorkDirs` map closes the WorkDir gap; `daemonSettings.FilesRead` + `schemaVersion: 3` migration via established defaults-merge constructor — Phase 118 (FS-02, FS-10..14)
- Webserver mounts `/api/files/*` under `requireFilesRead` (cap-gated for Tailscale-HTTPS web-share viewers); `SetFilesHandler` plumbed at both `AutoStartWebServer` and `handleWebServerStart` daemon construction sites — daemon socket and webserver share the same `*files.Handler` instance with no duplication; viewer without `files.read` gets 403 with body containing `"files.read"` (NOT 404); missing cap → 401; non-GET/HEAD → 405. Cross-browser Playwright (Chromium + Firefox + WebKit) reports zero CSP violations — existing `script-src 'self'` + `style-src 'self' 'unsafe-inline'` + `'wasm-unsafe-eval'` policy unchanged — Phase 119 (WEB-01..05)
- React `FileBrowserTab` (desktop + web-share): single-pane file list + side-by-side preview (NOT tree+list — collides with sidebar); `BreadcrumbBar` bounded at session cwd (cannot navigate above root via typed/pasted/clicked path); `react-markdown@10.1.0` + `remark-gfm@^4` for GFM tables/task lists (NO `rehype-raw` — XSS risk); image previews via `<img src="/api/files/read?...">` (NOT base64-in-state); type-ahead filter activated by `/` (parity with TUI; NOT Cmd-F — keeps xterm.js scrollback search); over-cap/binary refusal copy + Range-capable Download button; ARIA semantics (`role="grid"`, `role="region"`, `role="navigation"`); 4-state `useFilesCapability` hook surfaces `PermissionDeniedTakeover` with verbatim `"files.read permission required"`; web-mode detection module (`webMode.detectMode`) reads URL-param-driven `session+cap` for `/app/...` web-share mount; Playwright cross-browser e2e merge gate (45 cells across Chromium + Firefox + WebKit including DOM-level scenarios 13 owner + 14 viewer 403) — Phase 120 (UI-01..14)
- TUI Files view: lipgloss two-pane file list + viewport preview pane joined via `lipgloss.JoinHorizontal`, TokyoNight palette consistent with existing tabs; ALL filesystem I/O dispatched via `tea.Cmd` returning `tea.Msg` (no synchronous `os.ReadDir` in `Update` — would freeze render loop); static-grep gate `TestFiles_NoSyncFSCalls` enforces; key-dispatch priority slot 5.5 (above main view, below kill-confirm/new-session/QR overlay/help); `charmbracelet/glamour` promoted from indirect to direct dep for markdown rendering; `truncateLeft` status line (`…/utils/helper.ts`) preserves high-information leaf-end; `/` filter + Escape dismiss; `?` help overlay updated with Files keybindings; Backspace at cwd root is a no-op (never traverses above session root) — Phase 121 (TUI-01..10 local half; TUI-08 remote half delivered by Phase 122)
- Cross-surface remote-session file browse parity (audit-driven Phase 122 insert after Phase 121 scoped local-only with toast): daemon proxy route `/api/files/remote/{sid}/{op}` + in-memory `RemoteCapStore` (sid → {baseURL, cap}); `ExchangeJoinCodeAtURL` Wails helper exchanges a join code for a session cap via remote daemon 303 redirect; desktop GUI `RemoteJoinCodeModal` + `EnableWebSharingTakeover` (verbatim copy "Enable web sharing to browse this session's files") + App.tsx remote tab gate routes `FileBrowserTab` through `pathPrefix='/api/files/remote/<sid>'`; TUI `RemoteFilesClient` (HTTPS+cap; TLS 1.2+ pinned; cap redacted from errors) implements same `FilesClient` interface as `*daemon.DaemonClient` → same `handleFilesKey` + `applyFilesListMsg` pipeline drives local AND remote; `joinCodePromptModel` Bubble Tea modal; previous v3.4 toast "File browser not available for remote sessions" removed (grep = 0). Cross-surface byte-equivalence proven by 3 independent observers: daemon-proxy Go + tui.RemoteFilesClient Go + Playwright HTTPS browser against shared fixture — Phase 122 (REMOTE-01..05)
- Audit-driven mid-milestone phase insertion pattern reaffirmed: Phase 122 inserted after Phase 121 surfaced cross-surface remote-browse as a release-blocking parity gap. Same pattern as v3.3 Phases 107/108. Cross-surface (GUI/TUI/CLI/web) parity remains a release-blocking contract.

**Known deferred items (carried to v3.5):**

- TD-1: Phase 120 — Wails desktop click-path UAT deferred to manual milestone audit (Sessions panel → right-click → Open file browser → verify tab + breadcrumb + file list + previews)
- TD-2: Phase 121 — Visual TokyoNight + lipgloss border + remote-session perceptual UAT (5 items, user is colorblind — requires sighted spot-check)
- TD-3: Phase 122 — 22-step two-machine tailnet UAT (Machine A web-share + Machine B GUI + Machine B TUI; cross-surface parity check; failure-mode takeover)
- TD-4: Phase 120 WARNINGS WR-01..WR-05 open (`/app/` dir listings, cache-control, joinPath sanitization, mtime fallback, comment clarity) — non-blocking
- TD-5: Phase 122 `ExchangeJoinCodeAtURL` JSON-vs-303 mismatch shim cleanup deferred to v3.5
- Operator one-time (carry-forward from v3.3): `RELEASE_PUBLISH_TOKEN` PAT + `WINGET_FIRST_SUBMISSION=true` variable (one-time, first WinGet submission)
- All write operations (upload/delete/rename/mkdir/edit), CodeMirror 6 vs Monaco decision, TUI shell-out to `$EDITOR`, syntax highlighting for code files — all v3.5
- Known deferred items at close: 5 user-acknowledged TDs (see STATE.md Deferred Items)

---

## v3.3 Shell Sessions & Polish (Shipped: 2026-05-17)

**Phases completed:** 9 phases (100-108), 18 plans
**Requirements:** 35/35 satisfied (SHELL-01..12, PARITY-01..04, SETUI-01..03, POLISH-01..06, UAT-01..07, DIST-01..03)
**Commits:** 133 | **Timeline:** 5 days (2026-05-12 → 2026-05-17)
**Source changes:** 57 files, +5,784 / -156 lines (excluding `.planning/`)
**Closes:** GitHub Issues #44 (shell agent) + #45 (Settings hyperlinked index); absorbs v3.1 Phase 91 distribution-pipeline followups and v3.2 polish/UAT carry-over

**Key accomplishments:**

- Shell sessions as a first-class agent type across all three surfaces (GUI/TUI/CLI): daemon-side cross-platform shell discovery (`internal/pty/shells.go` — bash/zsh/pwsh/powershell + `$SHELL` + `/etc/shells` + Windows PowerShell paths), interactive (non-login) PTY spawn with WorkDir honored, status-heuristic exclusion (only `running`/`stopped`, never `waiting`/`error`), one-time web-share confirmation banner explaining arbitrary-command-execution risk, slate-cyan (#89ddff) agent badge — Phases 100/101 (SHELL-01..09)
- Mid-milestone shell-UX scope flip: collapsed multi-row picker to a single "Shell" entry across GUI (Phase 107) and TUI/CLI (Phase 108); Settings → Paths "Shell binary" field with daemon-resolved `shellPath` (fallback chain `$SHELL` → `DiscoverShells` → platform default), clean-exit `-1 → 0` PTY-wait normalization with `autoCloseRef`-gated tab auto-close — SHELL-10/11/12 + PARITY-01..04
- Settings hyperlinked index (Issue #45): sticky jump-link bar with anchor links to each section header, autocomplete search filtering settings by label (static `SEARCH_INDEX`, scroll-margin-top anchors, native `a href=#...` smooth scroll) — Phase 104 (SETUI-01..03)
- v3.2 polish closure: Cmd/Ctrl-click `mailto:` URLs route through `LinkConfirmPopover` IDN chain (Phase 102, POLISH-01/02); find-bar Esc/close dismiss after case-sensitive toggle, 200ms exit animation parity, `Sidebar.test.tsx` localStorage polyfill, sixel-only IIP decision (Phase 103, POLISH-03..06)
- Phase 91 v3.1 distribution-pipeline backfill: `release.yml` PAT fallback so `release.published` auto-triggers `distribute.yml` (DIST-01); `distribute.yml` reads `$env:RELEASE_TAG` env block for both `release.published` and `workflow_dispatch` (DIST-02); `wingetcreate new`/`update` branching on `WINGET_FIRST_SUBMISSION` (DIST-03) — Phase 106 (code-complete; operator-pending)
- Deferred v3.2 UAT re-run executed end-to-end (Phase 105, 7 scenarios): 3 pass + 2 verified-in-code + 2 with bugs filed to GitHub (#54 chafa OSC leak, #55 WebGLRecoveryBanner missing, #56 iPad tap-on-link captured by xterm-helper-textarea) — all 4 deferred to v3.4 as pre-existing (non-regression) tech debt
- Audit-driven phase insertion pattern proven: Phase 107 inserted 2026-05-13 after code-complete audit surfaced shell-UX feedback + clean-exit bug; Phase 108 inserted 2026-05-16 after 101-UAT Test 3 declared multi-shell rows on TUI/CLI a release-blocking parity gap. Cross-surface parity (GUI/TUI/CLI) now treated as release-blocking contract.

**Known deferred items (carried to v3.4):**

- Operator: Phase 106 `RELEASE_PUBLISH_TOKEN` PAT creation + `WINGET_FIRST_SUBMISSION=true` variable (one-time, before next release)
- `scottkw/agenthub#54` — chafa OSC 10/11 + DA1 response leak into shell stdin (web surface only; pre-existing)
- `scottkw/agenthub#55` — WebGLRecoveryBanner not rendering despite functional DOM fallback (pre-existing Phase 93 bug)
- `scottkw/agenthub#56` — iPad tap-on-link captured by xterm-helper-textarea (pre-existing iPad-touch polish cluster)
- Phase 101 visual-fidelity UAT (5 cosmetic items, non-gating)
- Phase 108 WR-01/WR-02 + IN-01..04 (documentation/dead-code cleanups)
- Phase 107 IN-01/02/03 + Browse-button aria-label pattern + SettingsSearch SEARCH_INDEX missing "Shell binary"
- Phase 101 advisory WR-01..09 + IN-01..06 (15 advisory tech-debt items in `101-REVIEW.md`)
- Phase 103 process debt (no `103-SUMMARY.md`, no `103-IIP-DECISION.md`, no `103-VERIFICATION.md`)
- TestOpenCodeANSICapture data race (pre-existing, skipped)
- Pre-existing `TestShellWebShareWarned_Default`-family failures (3 internal/daemon tests; SPEC §Out-of-scope for Phase 108)
- Nyquist `*-VALIDATION.md` missing for Phases 101–108 (process debt; not a blocker)
- Known deferred items at close: 12+ (see STATE.md Deferred Items)

---

## v3.2 Plugin Suite (Shipped: 2026-05-12)

**Phases completed:** 8 phases (92-99), 44 plans, 55 tasks
**Requirements:** 40/40 satisfied (PLUG-01..04, WGL-01..04, U11-01..02, SRC-01..05, LNK-01..06, IMG-01..04, SER-01..03, CLIP-01..02, PRG-01..03, PUI-01..04, WEB-01..03)
**Commits:** 269 | **Timeline:** 9 days (2026-05-03 → 2026-05-12)
**Source changes:** 117 files, +22,697 lines
**Closes:** GitHub Issue #36 ("Extend xterm.js functionality with select plugins")

**Key accomplishments:**

- Plugin Settings Foundation: daemon `PluginSettings` source of truth with defaults-merge constructor, Wails RPC + `settings:plugins` runtime event hot-swap pipeline, 8-toggle PluginsSection in Settings tab, v3.1 → v3.2 `schemaVersion: 2` migration with per-field assertions (Phase 92, PLUG-01..03, PUI-01)
- Vendoring discipline + web parity: generalized `vendor_drift_test.go` into a load-bearing CI gate enforcing `@xterm/addon-*` version parity for every addon; vendored 3 already-shipping addons (webgl/unicode11/clipboard) for the web page; capability-gated `/api/plugin-config` REST + SSE stream; two-useEffect TerminalPanel pattern + WebGLRecoveryBanner with context-loss / software-rasterized variants (Phase 93, PLUG-04, WGL-01..04, U11-01..02, CLIP-01..02, WEB-01..03)
- Scrollback search (Cmd-F find bar): focus-conditioned find bar on desktop + web with regex/case/word toggles, persisted defaults, 200ms slide-in/out animation matching BannerStack; `SetSearchConfig` sub-key RPC + `seededRef` one-shot useEffect pattern; 10,000-line scrollback perf gate + cancel-on-close contract as automated regression tests (Phase 94, SRC-01..05)
- Web-Links security hardening (v3.1-allowlist rigor): strict scheme allowlist (`https`/`http`/`mailto`), Cmd-click on macOS / Ctrl-click elsewhere, OSC 8 hover-href display, IDN/Punycode + 30-entry typosquat list with portal-rendered ARIA `LinkConfirmPopover`, Wails `BrowserOpenURL` desktop / `_blank`+`noopener,noreferrer` web split (Phase 95, LNK-01..06)
- Inline images + CSP audit: sixel via `@xterm/addon-image`, `'wasm-unsafe-eval'` CSP amendment 2 (audited and minimal), 16 MB per-tab `storageLimit` (down from upstream 100 MB), `SetImageConfig` sub-key RPC, byte-fidelity multi-client relay test (Phase 96, IMG-01..04)
- Save Terminal As + OSC 9;4 progress: SerializeAddon → TabBar "Save Terminal As…" with secrets warning and mockable `saveFileDialogFunc`; ProgressAddon → per-tab `.tab__progress` underline + atomic `SetTrayProgress(quartile)` (200ms debounced) (Phases 97 + 98, SER-01..03, PRG-01..03)
- Release gate (settings UI polish + final CSP audit): `PluginToggleBanner` one-shot toasts for non-hot-swappable plugins, inline `<details>` disclosures persisting via sub-key RPCs immediately without "Save Plugins", cross-browser Playwright e2e (Chromium + Firefox + WebKit) with zero CSP violations, GitHub Actions e2e workflow committed (Phase 99, PUI-02..04)

**Known deferred items (carried to v3.3):**

- 6 polish-grade tech-debt items tracked in `.planning/v3.2-RELEASE-BLOCKERS.md`:
  - P-1: `mailto:` URLs not detected as clickable despite documented allowlist (Phase 95)
  - P-2: Cyrillic IDN spoof URLs silently inert (defensive-by-accident; documented confirmation flow unreachable) (Phase 95)
  - P-3: Find bar will not dismiss after clicking case-sensitive toggle (Esc + close both stop working) (Phase 94)
  - P-4: Find bar slide-OUT animation missing on Esc / close (slide-IN works) (Phase 94)
  - P-5: iTerm2 IIP protocol does not render (sixel works); confirm sixel-only intent vs `iipSupport` audit (Phase 96)
  - P-6: 20 `Sidebar.test.tsx` tests fail under Vitest 4 + jsdom 29 (localStorage global) — pre-existing test-env debt, no production impact
- 9 UAT items deferred to v3.3 (require physical iPad, real Tailscale tailnet, or raw shell PTY): 93 UAT-1/2/5; 94 UAT-3/4; 95 web-noopener + iPad LNK-01..05; 96 Scenarios 1+2; 99 Test 11
- Backlog: raw shell session type — unblocker for all of the above UAT items and the v3.3+ headline feature
- Known deferred items at close: 15 (see STATE.md Deferred Items)

---

## v3.0 Session Lifecycle & TUI Polish (Shipped: 2026-04-19)

**Phases completed:** 4 phases, 9 plans, 13 tasks
**Requirements:** 12/12 satisfied (SET-01..SET-02, SESS-01..SESS-03, APP-01..APP-03, TUI-01..TUI-04)
**Commits:** ~40 | **Timeline:** 2 days (2026-04-18 → 2026-04-19)
**Source changes:** 97 files, +17,049 / -301 lines

**Key accomplishments:**

- Settings UI alignment: unified Paths section into single table with shared column headers for CLI and Tailscale rows; removed inline style overrides for consistent 12px description text; all four section headers use shared CSS rule (Phase 83, SET-01..SET-02)
- Session auto-close: exit detection pipeline via `hub.Done()` PTY EOF signal, `pollSessionStatus` detects `State=="stopped"`, emits `session:exit` Wails event; frontend 5-second countdown with ExitToast (fixed-position) and ExitCountdownBanner (inline); Keep Open cancel; auto-close toggle in Settings > Session Behavior; 10-second web-serving grace period (Phase 84, SESS-01..SESS-03)
- Quit confirmation modal: `beforeClose` and tray Quit both emit `app:quit-requested` event; QuitConfirmModal shows session count with colored status dots; "Quit GUI Only" hides window and sends macOS native notification via UNUserNotificationCenter; "Quit Everything" shuts daemon and exits (Phase 85, APP-01..APP-03)
- TUI visual polish: TokyoNight hex palette with 22+ lipgloss LightDark adaptive tokens; two-pane layout (sidebar + tabbed content) with focus-aware key routing; bordered session frames via `injectBorderTitle()`; per-agent colored badges for 6 CLIs; Home/Sessions/Remote/Settings navigation mirroring GUI sidebar (Phase 86, TUI-01..TUI-04)

---

## v2.1 Bug Fixes & UX (Shipped: 2026-04-17)

**Phases completed:** 4 phases, 8 plans, 19 tasks
**Requirements:** 12/12 satisfied (SET-01..SET-05, TS-01..TS-02, BAN-01..BAN-02, TRAY-01..TRAY-03)
**Commits:** 91 | **Timeline:** 2 days (2026-04-16 → 2026-04-17)
**Source changes:** 82 files, +14,301 / -300 lines

**Key accomplishments:**

- Settings persistence: CLI paths saved to daemon `settings.json` with JSON file storage, three-state Save button (idle/saving/saved), and native file/folder picker via Wails `OpenFileDialog` bindings (Phase 79, SET-01..SET-05)
- Tailscale detection: 4-state health check cascade (Not Installed → Daemon Stopped → Not Connected → Connected) with platform-specific binary detection for Homebrew, Snap, Flatpak, and Windows default install paths; diagnostics checklist in Settings (Phase 80, TS-01..TS-02)
- Banner notifications: vertical stacking with BannerStack container, independent dismiss handlers with 200ms exit animation, lifted update state from WelcomeTab to App.tsx, dismissed-state reset on webServerMode change (Phase 81, BAN-01..BAN-02)
- Minimize to tray: `startMinimized` toggle in Settings > Behavior section, persisted via daemon settings.json, conditional `WindowShow` in `domReady` gate, non-optimistic toggle state, `toggleLoaded` flash-gate prevents off→on flash on load (Phase 82, TRAY-01..TRAY-03)

---

## v2.0 Multi-Client, CLI UX & TUI Mode (Shipped: 2026-04-16)

**Phases completed:** 5 phases, 16 plans, 22 tasks
**Requirements:** 23/23 satisfied (MC-01..MC-06, SB-01..SB-07, TUI-01..TUI-10)
**Commits:** 116 | **Timeline:** 3 days (2026-04-14 → 2026-04-16)
**Source changes:** 158 files, +30,146 / -8,674 lines

**Key accomplishments:**

- Multi-client fan-out: relay Hub supports simultaneous WebSocket clients with per-subscriber metadata, independent scrollback, read-only mode (`--readonly` flag), max-wins PTY resize arbitration, and viewer count API (Phase 74, MC-01..MC-06)
- CLI status bar: persistent DECSTBM scroll-region bar with session name, agent type, hostname, detach hint, elapsed time, live viewer count via MsgMeta push frames, connection state tracking, `--status-top` flag, and clean teardown (Phase 75, SB-01..SB-07)
- TUI foundation: `agenthub tui` launches full-screen Bubble Tea v2 interface with session list, heuristic status glyphs, web server status footer, `?` help overlay, adaptive color, and 2s tick refresh (Phase 76, TUI-01, TUI-02, TUI-08, TUI-09)
- TUI session operations: attach (tea.Exec suspend/resume raw PTY), create (agent picker modal), kill (confirmation dialog), rename (inline edit) — shared `internal/attach/` package used by both CLI and TUI (Phase 77, TUI-03..TUI-06)
- TUI remote & QR: unified local+remote session list with tailnet peer discovery, grouped by hostname with divider rows, ASCII QR code overlay for session web URLs via go-qrcode (Phase 78, TUI-07, TUI-10)

**Known tech debt at close:** 21 items (metadata debt only — SUMMARY frontmatter missing requirements_completed across 4 phases, REQUIREMENTS.md checkboxes unchecked for 19/23 items; 0 code gaps)

---

## v1.14 UI Polish (Shipped: 2026-04-14)

**Phases completed:** 4 phases, 9 plans
**Requirements:** 4/4 satisfied (SBR-02, THM-05, UI-01, THM-04)
**Commits:** 64 | **Timeline:** 2 days (2026-04-13 → 2026-04-14)
**Source changes:** 69 files, +9,551 / -94 lines

**Key accomplishments:**

- Sidebar icons stay in fixed horizontal position during collapse/expand — fixed 48px icon slot via margin: 0 14px (Phase 70, SBR-02)
- OpenCode honors selected terminal theme — managed tui.json with OPENCODE_TUI_CONFIG env injection, SIGUSR2 broadcast to active sessions for live theme switching (Phase 71, THM-05)
- All GUI text meets WCAG AA 4.5:1 contrast — replaced all 32 `#565f89` text declarations with `#9aa5ce` across tab bar, settings, welcome, modals, and panels (Phase 72, UI-01)
- Theme picker curated from 157 to 138 readable themes with WCAG-derived readability filtering and localStorage fallback guard for stale theme names (Phase 73, THM-04)

---

## v1.13 Cross-Platform Fixes & UX (Shipped: 2026-04-12)

**Phases completed:** 3 phases, 5 plans
**Requirements:** 13/13 satisfied
**Commits:** 11 | **Timeline:** 2 days (2026-04-11 → 2026-04-12)
**Source changes:** 68 files, +3,604 / -6,569 lines

**Key accomplishments:**

- Cross-platform system tray: Linux D-Bus StatusNotifierItem (GNOME/KDE/XFCE) and Windows Shell_NotifyIcon Win32 API with shared menu helpers (tray_common.go), dynamic session list, status icons, hide-on-close, and Quit lifecycle
- Comprehensive agent CLI discovery: daemon PATH augmentation extended with snap, flatpak, cargo, npm, pnpm, and Windows native installer paths via build-tagged platform files (path_windows.go, path_other.go)
- Tailscale detection across platforms: Homebrew, system package manager, and Windows default install location (`C:\Program Files\Tailscale\tailscale.exe`)
- macOS Homebrew install command updated to single copyable `brew tap scottkw/agenthub && brew install --cask agenthub` on Welcome screen
- Settings tab refactored from sub-tab navigation to single scrollable page with h3 section headers (Appearance, Web Server, Paths) and CSS dividers

**Known tech debt (non-blocking):**

- 9 human UAT items: Linux/Windows tray testing requires live desktop environments; Settings visual verification requires running app
- SUMMARY frontmatter missing `requirements_completed` field across 4/5 plan summaries

---

## v1.12 UI/UX Polish (Shipped: 2026-04-11)

**Phases completed:** 4 phases, 4 plans
**Requirements:** 8/8 satisfied
**Commits:** 48 | **Timeline:** 2 days (2026-04-10 → 2026-04-11)
**Source changes:** 43 files, +4,667 / -82 lines

**Key accomplishments:**

- CSS flexbox centering for sidebar icons in collapsed 48px rail (SBR-01)
- 8px symmetric terminal padding with dynamic background matching terminal theme (PAD-01)
- 157 xterm-theme color schemes with live apply, localStorage persistence, and Settings > Appearance tab (THM-01/02/03)
- Web server URL actions: open in browser, copy to clipboard, and inline QR code in Settings (WEB-01/02/03)

---

## v1.11 Local Network & UX Polish (Shipped: 2026-04-10)

**Phases completed:** 6 phases, 9 plans, 6 tasks

**Requirements:** 10/10 satisfied
**Commits:** 57 | **Timeline:** 2 days (2026-04-08 → 2026-04-10)
**Source changes:** 31 files, +1,651 / -980 lines

**Key accomplishments:**

- Claude Code native installer detection: `~/.local/bin` added as first AugmentServicePath candidate for Anthropic native installer discovery
- Settings converted from modal overlay to singleton sidebar tab (SettingsTab.tsx), consistent with Home/Remote/Sessions panels
- Web server auto-starts on daemon launch; new sessions auto-enabled for web serving in both Tailscale and local modes
- Local network fallback: self-signed TLS (P256 + IP SAN), HTTP Basic Auth with generated password, LAN IP selection, persistent nudge banner, password display in Settings
- Frontend webEnabled seeding chain restored across all 5 IPC layers (Go bindings → TypeScript types → React state) for correct StatusBar display
- Tech debt cleanup: deleted orphaned SettingsPanel/HealthModal components, fixed 11 stale test assertions

**Known tech debt (7 items, all non-blocking):**

- SUMMARY frontmatter missing `requirements_completed` for phases 59, 60, 61
- App.tsx `retryInit()` missing `webServerRunning` override (narrow race, no data loss)
- App.tsx stale closure risk in `createTab` useCallback (WR-01)
- App.tsx init/retryInit code duplication (WR-02)

---

## v1.10 Collapsible Sidebar Navigation (Shipped: 2026-04-08)

**Phases completed:** 2 phases, 3 plans, 4 tasks
**Requirements:** 11/11 satisfied
**Commits:** 21 | **Timeline:** 1 day (2026-04-08)
**Source changes:** 26 files, +3,243 / -205 lines

**Key accomplishments:**

- Collapsible left sidebar with Heroicons SVG icons (@heroicons/react 2.2.0) replacing toolbar action buttons
- App layout restructured to flex-row with Sidebar + app__content, collapsed (48px) and expanded (200px) modes
- All 5 navigation items wired: Home, Remote, Sessions, New Tab, Settings — each opens corresponding tab/panel
- Tab bar cleaned up: action buttons removed, retains session tabs only; dead CSS and obsolete tests removed
- Sidebar collapsed/expanded state persists via localStorage across app restarts

---

## v1.9 Remote Sessions & App Polish (Shipped: 2026-04-08)

**Phases completed:** 6 phases, 14 plans, 17 tasks
**Requirements:** 17/17 satisfied
**Commits:** 111 | **Timeline:** 2 days (2026-04-06 → 2026-04-07)
**Source changes:** 36 files, +3,499 / -97 lines

**Key accomplishments:**

- Standard macOS app menus (File, Edit, Window, Help) with Cmd+C/V clipboard in terminal tabs and build-time version injection via ldflags
- Tailscale peer discovery (`internal/tailnet`) with injectable deps, concurrent probe pool (cap 5), and daemon `GET /tailnet/peers` with 30s thundering-herd-safe cache
- Auto-update checker polling GitHub releases on startup + hourly, notification banner in WelcomeTab, and Help menu "Check for Updates" item
- Remote Sessions GUI panel with tailnet peer grouping, loading states, 30s auto-refresh, and one-click browser open via BrowserOpenURL
- CLI remote sessions: unified `agenthub list` with HOST column grouping, `agenthub attach hostname:session-id` via WSS relay with hostname banner
- Tailscale onboarding enhancement: platform-specific install commands with copy-to-clipboard, macOS auto-install via Homebrew with streaming progress, and numbered HTTPS cert setup guide

---

## v1.8 GitHub Distribution & CI/CD (Shipped: 2026-04-06)

**Phases completed:** 5 phases, 9 plans, 11 tasks

**Key accomplishments:**

- Go module path rewritten from `github.com/agenthub/agenthub` to `github.com/scottkw/agenthub` across go.mod and all 30 import sites in 16 .go files; race detector clean
- RELEASE_PLEASE_TOKEN configured and release-please created PR #1 (chore(main): release 1.8.0) on first push to main
- One-liner:
- Homebrew cask template and three-file WinGet manifests (schema 1.12.0) with sed-replaceable {{VERSION}} tokens matching Phase 46 artifact naming exactly
- distribute.yml GitHub Actions workflow that auto-updates scottkw/homebrew-agenthub cask formula on each release:published event using checksums.txt SHA256 extraction, nick-fields/retry, and TAP_DEPLOY_TOKEN PAT
- winget-releaser job added to distribute.yml with restrictive installer regex, plus populate-manifests.sh helper for one-time manual WinGet first submission
- WINGET_TOKEN PAT created with public_repo scope, stored as repo secret; scottkw/winget-pkgs fork verified; manifest submission deferred pending first release

---

## v1.7 Daemon UX & Branding (Shipped: 2026-04-03)

**Phases completed:** 8 phases, 10 plans, 20 tasks

**Key accomplishments:**

- 1024x1024 branded A logomark extracted from title logo, compiled into full 10-entry macOS ICNS (590KB), 6-frame Windows ICO, and 6 Linux PNGs via sips+iconutil+ImageMagick pipeline with post-build bundle injection
- Branded splash screen with StartHidden + OnDomReady lifecycle, static HTML bridge div, React SplashScreen overlay, and triple-path init dismissal with 3s fallback
- Machine hostname added to daemon session API — SessionInfo includes Hostname field populated at engine startup via os.Hostname()
- Web terminal status bar with session name, agent type, hostname display and 3-second REST-polled connection state indicator using TokyoNight theme
- Connection banner and detach message for CLI attach — shows session name, CLI type, hostname, and Ctrl-\ hint on stderr before raw mode
- DaemonManagerPanel tab in TabBar showing live session list with status dots, kill buttons, and web-serve toggles via ☰ button — zero new Go bindings
- Daemon POST /shutdown endpoint, two 18x18 monochrome tray icon PNGs embedded in tray.go, and LSUIElement=true in production Info.plist
- AgentHubMenuDelegate NSMenuDelegate for dynamic session menu, updateTray() for icon/tooltip state, startTrayPoller() 5s background refresh, and tray:focus-session frontend event handler
- Split refreshTrayState nil-client guard so tray shows error icon (trayIconErrorBytes) and updated tooltip when daemon fails to start
- Hostname field forwarded from daemon SessionInfo through App.go Wails binding, displayed as pill badge in DaemonManagerPanel with em dash fallback for empty values

---

## v1.6 Terminal Fill Fix v2 (Shipped: 2026-03-31)

**Phases completed:** 1 phases, 1 plans, 2 tasks

**Key accomplishments:**

- Replaced double-rAF one-shot fit with bounded rAF retry loop polling FitAddon.proposeDimensions() until cell dimensions are non-zero, fixing initial-load terminal fill for Claude, Gemini, and OpenCode CLIs

---

## v1.5 Bug Fixes & CLI Args (Shipped: 2026-03-26)

**Phases completed:** 5 phases, 6 plans, 8 tasks

**Key accomplishments:**

- Args wiring: `args []string` threaded through all 5 daemon IPC layers with integration tests proving args survive the full HTTP round-trip from JSON to PTY
- CLI passthrough: `agenthub new <agent> <path> -- <extra-args>` via `splitDashDash` helper
- Eliminated 2-second session status startup delay by restructuring pollSessionStatus to poll immediately then sleep 500ms between iterations
- TDD implementation of runtime PATH augmentation at daemon startup so service-mode agents (nvm, Volta, Homebrew) resolve via exec.LookPath without shell init files
- Thread cols/rows from React frontend through Wails/Go stack to PTY spawn with double-rAF initial fit timing

---

## v1.4 Unified Binary (Shipped: 2026-03-25)

**Phases completed:** 3 phases, 3 plans, 6 tasks

**Key accomplishments:**

- Merged cmd/agenthub-cli/ into root package: single agenthub binary dispatches no-args→GUI, flags→GUI, --help→usage(), commands→runCLI() with full migrated+new test suite
- Deleted cmd/agenthub-cli/ dead package (8 files, 1559 lines) and scrubbed its README.md row — repo now has zero references to the old standalone CLI binary
- Portable BASH_SOURCE path resolution in build-script tests and race-enabled CI workflow with ubuntu-latest build-script verification

---

## v1.3 CLI + Daemon (Shipped: 2026-03-25)

**Phases completed:** 8 phases, 15 plans, 23 tasks

**Key accomplishments:**

- Daemon architecture: SessionEngine extracted into `internal/daemon` with HTTP/JSON API over Unix socket, typed DaemonClient, 28 tests with -race
- Process separation: Sessions survive GUI close/reopen; RunDaemon/EnsureDaemon lifecycle; App reduced to thin DaemonClient shell
- Full CLI: 13 commands (new, list, kill, rename, attach, web start/stop/status, serve/unserve, health, qr, settings) with `--json` output
- Interactive attach: Full PTY proxy with raw I/O, detach key (Ctrl-\), SIGWINCH resize, Ctrl-C passthrough, scrollback replay, signal-safe terminal restore
- Service manager: `agenthub daemon install/uninstall/start/stop` via kardianos/service for launchd/systemd/Windows SCM
- Robustness: Windows named pipe dial fix + graceful GUI startup failure with error banner and retry

**Stats:**

- 101 commits, 38 files changed, +4,197/-295 lines, ~12,619 LOC (9,068 Go + 2,619 TS/TSX + 932 CSS)
- Timeline: 8 days (2026-03-17 → 2026-03-25)

**Tech debt (accepted):**

- ROADMAP.md plan checkboxes unchecked for 22-02, 23-02, 26-01 (code implemented and working)
- SUMMARY.md frontmatter missing requirements_completed field across all plan summaries
- All 6 Nyquist VALIDATION.md files are draft status
- Service manager live start/stop round-trip needs human verification on macOS

---

## v1.2 Tailscale-Only Networking (Shipped: 2026-03-23)

**Phases completed:** 5 phases, 10 plans

**Key accomplishments:**

- Tailscale health check infrastructure — detects installation, connection, and cert readiness with background polling via `local.Client{}`
- Let's Encrypt TLS via Tailscale daemon — replaced self-signed cert system with `GetCertificate` hook, FQDN-based URLs, CT disclosure flow
- Auth layer removal — deleted password auth, per-session tokens, and all auth middleware; tailnet membership = access control
- Dead code cleanup — removed generic VPN interface picker (`network.go`), `GetNetworkInterfaces`, and all orphaned frontend bindings
- Health modal with platform-specific guidance — three-state instructional UI (not installed / not connected / no certs) with Check Again button and auto-dismiss
- Tailscale status indicator in Settings panel replacing removed VPN interface picker

**Stats:**

- 64 commits, 74 files changed, ~8,846 LOC (5,364 Go + 2,550 TS/TSX + 932 CSS)
- Timeline: 6 days (2026-03-17 → 2026-03-23)
- Git range: `4e84151..6894faf`

**Tech debt (info-level only):**

- TailscaleHealth type defined inline in App.tsx and App.d.ts rather than imported from models.ts — manual sync needed if fields change
- NoCertsPanel `_platform` unused param — intentional, no platform-specific content in that panel

---

## v1.1 Polish & Build (Shipped: 2026-03-20)

**Phases completed:** 7 phases, 13 plans

**Key accomplishments:**

- Terminal layout baseline — CSS flex chain fix, enlarged toolbar buttons (38x38px), 42px tab bar
- Per-tab status bar replacing floating web-serving overlay with 3-state strip (inactive/off/on)
- Tabbed settings modal with inline Save Paths and single Close footer
- Per-tab font size adjustment via SHIFT+=/- shortcuts with per-tab state isolation
- New-session modal with agent picker, native OS folder browser, and last-folder memory
- Tab rename (double-click + right-click context menu) with name propagation to web dashboard
- Web dashboard visual redesign: card layout, status dots, CLI badges, TokyoNight palette
- Cross-platform build script (`build.sh`) with macOS signing/notarization pipeline

**Stats:**

- 88 files changed, ~9,956 LOC (6,541 Go + 2,622 TS/TSX + 793 CSS)
- Timeline: 4 days (2026-03-17 → 2026-03-20)
- Git range: `feat(07-01)` → `feat(13-02)`

**Known Gaps (accepted as tech debt):**

- TERM-01 partial: Initial-paint terminal fill timing race (xterm.js FitAddon) — fills after resize, tabled by user
- BUILD-01..04: Missing from 13-01-SUMMARY.md requirements_completed frontmatter (code verified working)
- DetectedCLI.DisplayName missing from TypeScript Wails stub (works at runtime)
- build_linux() uses go build instead of wails build inside Docker (binary produced, may lack wails metadata)
- macOS notarization untested end-to-end (codesign verified, notarytool needs real app-specific password)

---

## v1.0 MVP (Shipped: 2026-03-19)

**Phases completed:** 6 phases, 19 plans, 6 tasks

**Key accomplishments:**

- Cross-platform PTY process management with CLI auto-detection (Claude Code, Codex, Gemini CLI, OpenCode)
- WebSocket fan-out relay with binary framing protocol and bounded scrollback replay
- Wails desktop UI with tabbed xterm.js terminals, session naming, system tray persistence
- Embedded HTTPS web server with self-signed TLS (CA+leaf), bcrypt auth, per-session token links, VPN/Tailscale binding
- QR code generation for web-served sessions (desktop modal + web dashboard) with live status badges
- GitHub Actions CI matrix for macOS (signed/notarized), Linux (WebKitGTK 4.0/4.1), and Windows (NSIS)

**Stats:**

- 107 commits, 149 files, ~8,100 LOC (6,400 Go + 1,700 JS/TS/CSS)
- Timeline: 3 days (2026-03-17 → 2026-03-19)
- Git range: `feat(01-01)` → `feat(06-02)`

**Known Gaps:**

- STAT-02 partial: Status heuristics only for Claude CLI; codex/gemini/opencode always show "running"
- TERM-05 partial: @xterm/addon-clipboard installed but unused; native browser copy/paste works
- ParseWin32Input exported but not wired into relay/webserver input path (Windows only)
- 9 items require human verification (TLS flows, visual items, CI execution)

---
