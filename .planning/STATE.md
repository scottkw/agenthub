---
gsd_state_version: 1.0
milestone: v4.2
milestone_name: Funnel Sharing & Polish
current_phase: 167
current_phase_name: native-notifications
status: executing
stopped_at: Completed 167-07-PLAN.md (M-41 gap closure, frontend half — permission-denied hint) -- Phase 167 both gap-closure plans (06 backend, 07 frontend) done; awaiting M-41 signed-build re-run
last_updated: "2026-07-01T13:54:06.009Z"
last_activity: 2026-07-01
last_activity_desc: Completed 167-06-PLAN.md (M-41 live-delivery gap closure, instrumentation-first)
progress:
  total_phases: 4
  completed_phases: 3
  total_plans: 17
  completed_plans: 17
  percent: 75
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-30 — v4.2 milestone started)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 167 — native-notifications

## Current Position

Phase: 167 (native-notifications) — EXECUTING
Plan: 7 of 7
Status: Ready to execute
Last activity: 2026-07-01 — Completed 167-06-PLAN.md (M-41 live-delivery gap closure, instrumentation-first)

```
v4.2 Progress: [██████████░░░░░░░░░░] 50% (2/4 phases — 165 ✅ DONE (live), 166 ✅ DONE (verified 8/8))
```

## Live UAT Findings (2026-06-30, real Funnel-granted tailnet)

- **M-34 FAIL (BLOCKER, FNL-03):** Funnel URL still fails end-to-end. The bug is at the **listener's TLS cert selection (SNI)**, not the proxy-target host string — and BOTH naive target fixes are dead ends (verified live, off-tailnet device + same-host):
  - **Option A `https+insecure://<bindIP>:443` (what 165-04 shipped) → 502.** tailscaled dials the raw IP → sends **SNI=100.86.210.104** → the listener's `lc.GetCertificate` only mints a cert for the `.ts.net` hostname, none for an IP literal → TLS `internal_error` → 502. `https+insecure` disables the client's *verification*, not the SNI the *server* needs to *select* a cert. Proven: connect-to-IP + SNI=hostname → **302 served**; SNI=IP → TLS internal error; openssl SNI=hostname → real Let's-Encrypt cert, verify OK.
  - **Option B `https+insecure://<FQDN>:443` (the experiment this session) → HANG/timeout (off-tailnet too).** Root cause = **DNS resolution loop**: MagicDNS resolves the FQDN → 100.86.210.104 (local listener), but **public DNS (8.8.8.8 / 1.1.1.1) resolves it → 209.177.145.137 / 199.38.181.54 = Tailscale's Funnel INGRESS servers**. tailscaled's proxy dialer resolves the FQDN to the public ingress → Funnel→ingress→Funnel loop → hang. (The original "unproven MagicDNS hairpin" rejection of B was right, sharper reason.)
  - **CHOSEN FIX (165-05, EXECUTED) = loopback-HTTP target — NOT a server-side cert hack, NOT FQDN.** Discussion with the user clarified the cleaner architecture: tailscaled already terminates the ONLY public TLS on hop 1 (guest→ingress, real verified cert); since AgentHub and tailscaled are CO-LOCATED, hop 2 doesn't need TLS at all. Fix: AgentHub adds a plain-HTTP listener on `127.0.0.1:0` (ephemeral) serving the same mux (bound in startTailscale, closed in Stop); EnableFunnel proxy target becomes `http://127.0.0.1:<loopbackPort>`. tailscaled proxies hop 2 to a live loopback endpoint — no TLS handshake, no SNI, no cert selection. Hop 2 plaintext is safe because loopback never leaves the host (co-location assumption recorded in code + threat model; must become a WireGuard-tunneled tailnet-IP target if ever split across nodes). Commits 628cc94f→3380202f. (Earlier interim ideas — FQDN target [Option B, DNS-loop dead] and a server-side SNI-agnostic GetCertificate wrap — were superseded by this simpler approach.)
  - The 165-04 test `TestEnableFunnel_ProxyTargetReachable` was FALSE-GREEN (loopback self-signed cert answered ANY SNI — same trap as [[feedback_tests_encoding_same_wrong_assumption]]). 165-05 REWROTE it to assert the loopback-HTTP target SHAPE (scheme http, host 127.0.0.1, port==loopback listener port AND != TLS port) + dial it with a plain http.Client. NOTE: the unit test guards target shape + loopback reachability only; it CANNOT reproduce the live SNI/ingress failure — live off-tailnet M-34 is the real acceptance gate.
  - **M-34 PASS LIVE 2026-06-30 (loopback-HTTP fix verified end-to-end):** off-tailnet device `curl` to the Funnel URL → **HTTP 200** on `/app/` (followed the /sessions→/app redirect), time ~0.44s. tailscaled proxied hop 2 to `http://127.0.0.1:<loopbackPort>` and the guest reached the session. Under the dev (`wails dev`) build the same path returned 503 "app bundle not configured" — that is the DOCUMENTED `go build`/no-`wailsassets` state (server.go:252-257; daemon has no embedded SPA), NOT a Funnel bug; reproduced identically on a direct (non-Funnel) loopback + TLS-listener hit. A production build (`wails build -tags wailsassets`) embeds the SPA → 200. So the live gate REQUIRES a production build, not `wails dev`. FNL-03 is CLOSED end-to-end.
- **M-35 (a) Funnel-off, (b) web-share-off, (c) explicit-kill (DELETE), (d) daemon-stop → ALL PASS live:** serve config empties immediately on every trigger; the GAP 2 / FNL-05 kill-path fix (synchronous runSessionExitCleanup, no D-12 grace) and graceful daemon-stop teardown both validated end-to-end on real tailscaled.
- **M-36 fallback (Tailscale stopped) → PASS live:** AgentHub auto-fell-back to local-mode web-share (`https://<LAN-IP>:7443`, Basic Auth); enabling Funnel failed CLOSED (HTTP 400 `funnel: loopback listener not started` — startLocal makes no loopback listener; CheckFunnelAccess passes on cached node capability so the loopback nil-guard is what fail-closes), no serve config written, local web-share unaffected. User-facing error wording + disabling the Funnel toggle in fallback is Phase 166 (Share modal) work.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260627-d81 | Fix Welcome screen layout — long Linux install URL broke alignment (flexbox min-width:auto); install boxes off-center / mismatched sizes | 2026-06-27 | a89cf26a | [260627-d81-fix-welcome-screen-layout-linux-install-](./quick/260627-d81-fix-welcome-screen-layout-linux-install-/) |
| 260628-pt8 | Chat toggle button slides with sidebar (transition: right 220ms ease-out added to .hub-modal__chat-toggle base rule; test (d) regression gate in chatToggleOverlap.test.ts) | 2026-06-28 | d2053f7b | [260628-pt8-when-the-chat-sidebar-slides-in-and-out-](./quick/260628-pt8-when-the-chat-sidebar-slides-in-and-out-/) |
| 260628-qs5 | Add Chat help section — v4.1 session-chat docs (opening, aliases, @session, RO vs RW guests, presence/history/markdown); SECTION_META + SECTIONS + integration test count 2→3 + #help-chat | 2026-06-29 | 71890dfd | [260628-qs5-add-details-on-the-new-chat-interface-to](./quick/260628-qs5-add-details-on-the-new-chat-interface-to/) |
| 260629-w19 | Fix WinGet first submission — `submit-winget` ran `wingetcreate new` with update-only flags + no checkout + manifests lacked schema header; rewired to checkout → populate → `wingetcreate submit`, added schema headers, set `WINGET_FIRST_SUBMISSION=true`. **Live: PR [microsoft/winget-pkgs#395007](https://github.com/microsoft/winget-pkgs/pull/395007) OPEN.** Post-merge: unset var + remove continue-on-error | 2026-06-29 | 07ae6ebb | [260629-w19-fix-winget-first-submission](./quick/260629-w19-fix-winget-first-submission/) |

## Operator Next Steps (carry-forward from v4.1)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Set before triggering `distribute.yml` for INSTALL-03; unset after `microsoft/winget-pkgs` accepts the first submission PR. **Live PR: microsoft/winget-pkgs#395007 (OPEN)**. Post-merge: unset `WINGET_FIRST_SUBMISSION` + remove `continue-on-error` from distribute.yml winget step.
3. **Branch protection on `main`**: applied 2026-06-22 — 5 required checks (4 build matrix + `playwright`; `strict:false`, `enforce_admins:false`). Still valid for v4.2.
4. **INSTALL-01 / M-25 clean-box Linux install**: runs as a release-time checklist step after `main` is pushed to origin (install.sh 404s on unpushed commits). Deferred from v4.1 closeout.

## v4.2 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 165 | Funnel Backend | FNL-01, FNL-02, FNL-03, FNL-04, FNL-05, FNL-06, FNL-07 | ✅ DONE — code-verified + all live UAT pass (M-34 off-tailnet 200, M-35 a/b/c/d, M-36 fallback) |
| 166 | Funnel Frontend + Help Guide | FUI-01, FUI-02, FUI-03, FUI-04, FUI-05, FUI-06, HLP-01, HLP-02 | Not started |
| 167 | Native Notifications | NTF-01, NTF-02, NTF-03, NTF-04 | 📋 PLANNED — 4 plans/3 waves, plan-checker PASSED (macOS=native path, Win/Linux=beeep, toggle in Behavior section) |
| 168 | Bug Fix & Settings Polish | FIX-01, FIX-02, FIX-03, UX-01, UX-02 | Not started |

**Total:** 24 requirements mapped across 4 phases (100% coverage).

### Key Sequencing Constraints

- Phase 165 must precede Phase 166 (Funnel UI depends on backend atomicity — partial deployment causes silent 403s for external guests)
- Phase 165's dual-origin fix may resolve the Funnel multi-viewer kick in #117; verify in Phase 165 UAT before scoping Phase 168's #117 relay buffer work
- Phases 167 and 168 are independent of Funnel; ordering is for focus, not correctness
- Phase 165 UAT must include a machine outside the tailnet (false-parity risk, per Phase 155 precedent)

## Key Decisions (v4.2)

| Decision | Resolution |
|----------|------------|
| Funnel backend is one atomic phase | FNL-04 (Origin/BaseURL fix) must land with EnableFunnel — partial deployment causes silent 403s for all Funnel guests. Never split. |
| Risk dialog shown every enable | No "don't show again". Established in scoping: risk acknowledgment is per-enable, every time. |
| Auto-expiry enforced daemon-side | Daemon tears down Funnel at expiry independent of any connected UI (FNL-07). |
| Notifications default OFF | Avoids surprise on upgrade from v4.1; consistent with industry norm (Warp). NTF-04 default: off. |
| beeep "Script Editor" macOS attribution | Acceptable trade-off for v4.2. Title field uses "AgentHub" to aid identification. Revisit in future milestone if user feedback demands branded attribution. |
| #117 requires both Part A and Part B in same phase | Buffer 256→1024 alone reduces but does not eliminate kick risk. Viewer-disconnect UI (Part B) must ship together. |
| Colorblind-safe indicator is release-blocking | FUI-03 must use text + icon, never color alone. Verified at hex/source level, not by eye. |

## Architecture Notes (v4.2)

**New Go dependency:** `github.com/gen2brain/beeep v0.11.2` (cross-platform notifications, no CGO)

**Key files modified:**

- `internal/webserver/server.go` — promote `lc local.Client` to struct field; add `EnableFunnel`/`DisableFunnel`/`FunnelBaseURL`
- `internal/webserver/origin_mw.go` — dual-URL allowlist (tailnet + Funnel origins)
- `internal/webserver/capability_mw.go` — `originAllowedForWrite` dual check
- `internal/daemon/api.go` — `funnelSessions` map, `handleSetSessionFunnel`, funnel-aware URL builders, web-share teardown path
- `internal/daemon/types.go` — `NotifyOnWaiting bool` in Settings
- `internal/relay/hub.go` — buffer 256→1024; `KickPersonKey` method
- `app.go` — `SetSessionFunnel` + `KickSessionViewer` bound methods; `maybeNotifyWaiting` de-dup
- `notification_windows.go` (NEW), `notification_linux.go` (NEW) — beeep platform wrappers
- `frontend/src/components/Hub/SessionShareModal.tsx` — Funnel toggle + risk dialog; viewer list + Disconnect button
- `frontend/src/components/Hub/WebShareSessionView.tsx` — self-contained plugin-config fetch + SSE; `baseURL` prop
- `frontend/src/components/StatusBar.tsx` — replace Enable/Disable Web with "Share Session" → modal
- `frontend/src/components/SettingsTab.tsx` — `NotifyOnWaiting` toggle + auto-switch toggle
- `frontend/src/components/HelpTab.tsx` + `HelpSectionNav.tsx` — add Sharing Guide section
- `frontend/src/content/help/sharing-guide.md` (NEW) — Funnel + device-share help article
- `App.tsx` — `handleOpenRemoteSession` in-app tab path; StatusBar wiring; `webParams.baseURL`

## Deferred Items

| Category | Item | Status |
|----------|------|--------|
| operator_runtime | `RELEASE_PUBLISH_TOKEN` PAT | pending (one-time, before next release) |
| operator_runtime | `WINGET_FIRST_SUBMISSION=true` variable + post-merge cleanup | pending (PR microsoft/winget-pkgs#395007 open) |
| manual_uat | INSTALL-01 / M-25 clean-box Linux install | deferred from v4.1; runs at release-time after `main` pushed |
| manual_uat | Phase 125 editor on-screen render + CodeMirror Tab/Cmd-V in WebView | pending (live app required) |
| manual_uat | Phase 126 `$EDITOR` suspend-resume terminal restore | pending (live app required) |
| manual_uat | Phase 124 home-dir warning banner on-screen | pending (live app required) |
| v4.3+ | web-plugin-hot-swap (#112 SKIPPED): web guests lost /api/plugin-config + SSE after Phase 159 redirect | FIX-01 in v4.2 Phase 168 addresses this |
| v4.3+ | Device-share automation via Tailscale admin API (FUT-01) | Out of scope for v4.2 per Issue #107 |

## Session Continuity

Last session: 2026-07-01T13:54:06.000Z
Stopped at: Completed 167-07-PLAN.md (M-41 gap closure, frontend half — permission-denied hint) -- Phase 167 both gap-closure plans (06 backend, 07 frontend) done; awaiting M-41 signed-build re-run
Resume file: None
Next action: Phase 167's M-41 crash regression is now hardened (bundle-id guard + @try/@catch in the darwin native notification path). Run `/gsd-verify-work 167` to re-run M-41 on a SIGNED PRODUCTION BUILD (not `wails dev`) on macOS/Windows/Linux — confirms real toast delivery + tray-hidden behavior with the hardening in place. This is the sole remaining open item before Phase 167 can be marked complete and Phase 168 (Bug Fix & Settings Polish) begins.

## Decisions (carry-forward from v4.1 — architecture reference)

- [Phase 151]: Reject-not-trim at cap: ErrChatCapReached returned instead of trimming oldest message
- [Phase 151]: Injected baseDir: NewChatStore accepts baseDir parameter enabling constructor-level test isolation with t.TempDir()
- [Phase 152]: Relay chat routes in wrapRelayWithChat outer wrap — avoids daemon↔relay import cycle
- [Phase 152]: Webserver chat uses provider callback not daemon import — T-151-09 circular import prevention
- [Phase 152]: RWMutex over Mutex
- [Phase 152]: AliasStore rolls back in-memory map on persist failure to keep store consistent with disk
- [Phase 152]: SetIdentityProviders setter avoids relay→daemon import cycle
- [Phase 154]: Phase 154-01: HandleChatSend uses SanitizeChatContent not SanitizePTYText; silent-drop on error (no NAK)
- [Phase 154]: alias field in ChatMessage mirrors Go json:"alias" tag — not authorAlias
- [Phase 154]: D-02 overlay mode: ChatPanel position:absolute over terminal, isActive unchanged by chatOpen, no PTY resize on toggle
- [Phase 155]: Export() always double-quotes participant values — simpler YAML safety invariant
- [Phase 155]: isReadOnly defaults to true (fail-safe); RO resolved from /info ?cap= perms
- [Phase 157]: VIEW-02 origin gate is FIRST check in ResizeClient — non-local returns immediately before lock
- [Phase 157]: broadcastResize self-acquires mu, called only after hub.mu.Unlock() — prevents self-deadlock
- [Phase 163]: D-06 reconciliation: loosened ONLY HandleChatSend; HandleInject (ErrReadOnly) and MsgInput discard unchanged
- [Phase 163]: ErrChatReadOnly deleted; ErrReadOnly retained for inject gate only

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 151 P01 | 352 | 3 tasks | 4 files |
| Phase 151 P02 | 8 minutes | 2 tasks | 4 files |
| Phase 151 P03 | 15m | 3 tasks | 7 files |
| Phase 152 P03 | 15m | 2 tasks | 2 files |
| Phase 152 P04 | 5 minutes | 1 tasks | 2 files |
| Phase 152 P05 | 20 minutes | 3 tasks | 5 files |
| Phase 152 P06 | 8 minutes | 2 tasks | 4 files |
| Phase 153 P01 | 8 minutes | 2 tasks | 3 files |
| Phase 153 P02 | 9 | 3 tasks | 4 files |
| Phase 153 P03 | 7 minutes | 3 tasks | 4 files |
| Phase 154 P01 | 6 minutes | 2 tasks | 7 files |
| Phase 154 P02 | 4 minutes | 2 tasks | 4 files |
| Phase 154 P03 | 9 minutes | 2 tasks | 5 files |
| Phase 154 P05 | 40 minutes | 2 tasks | 3 files |
| Phase 154 P06 | 70 minutes | 3 tasks | 8 files |
| Phase 155 P01 | 4 minutes | 3 tasks | 5 files |
| Phase 155 P02 | 6 minutes | 3 tasks | 3 files |
| Phase 155 P03 | 5 minutes | 3 tasks | 5 files |
| Phase 155 P05 | 22 minutes | 3 tasks | 1 files |
| Phase 156 P01 | 2 minutes | 3 tasks | 3 files |
| Phase 156 P03 | ~3 minutes | - tasks | - files |
| Phase 157 P01 | 3 minutes | 2 tasks | 2 files |
| Phase 157 P02 | 8min | 3 tasks | 4 files |
| Phase 157 P04 | 9 minutes | 2 tasks | 8 files |
| Phase 157 P05 | 3min | 1 tasks | 1 files |
| Phase 158 P01 | 10 | 2 tasks | 3 files |
| Phase 158 P02 | 15 | 3 tasks | 5 files |
| Phase 159 P01 | 10 minutes | 2 tasks | 4 files |
| Phase 161 P01 | 4 minutes | 2 tasks | 8 files |
| Phase 161 P02 | 5 minutes | 2 tasks | 2 files |
| Phase 161 P03 | 6m | - tasks | - files |
| Phase phase-161 P161-04 | 25min | 3 tasks | 4 files |
| Phase 162 P01 | 3 minutes | 3 tasks | 3 files |
| Phase 163 P01 | 6min | 2 tasks | 6 files |
| Phase 163 P02 | 4min | - tasks | - files |
| Phase 163 P03 | 3min | 2 tasks | 2 files |
| Phase 164 P01 | 6 minutes | 3 tasks | 5 files |
| Phase 164 P02 | 5 | 3 tasks | 5 files |
| Phase 165 P01 | ~6m | 3 tasks | 6 files |
| Phase 165 P02 | 20 | 3 tasks | 5 files |
| Phase 165 P03 | 490 | 1 tasks | 5 files |
| Phase 165 P04 | 25 | 3 tasks | 5 files |
| Phase 165-funnel-backend P05 | 7min | 3 tasks | 3 files |
| Phase 167 P01 | 12min | 2 tasks | 5 files |
| Phase 167 P02 | 8min | 2 tasks | 6 files |
| Phase 167 P03 | 12min | 2 tasks | 6 files |
| Phase 167 P04 | 20min | 2 tasks | 7 files |
| Phase 167 P05 (gap closure) | 8min | 2 tasks | 4 files |
| Phase 167 P06 | 6min | 3 tasks | 8 files |
| Phase 167 P07 | 8min | 2 tasks | 6 files |

## Decisions

- [Phase ?]: Injectable funnelClient interface seam mirrors statusFunc/prefsFunc pattern; production ws.lc field, test fakeFunnelClient
- [Phase ?]: CheckFunnelAccess + StatusWithoutPeers before ws.mu.Lock() (blocking Unix-socket calls must not hold mutex); ws.listener accessed directly to prevent RWMutex deadlock
- [Phase ?]: requireAllowedOrigin dual-origin: tailnet URL primary, Funnel URL secondary; secondary inert when FunnelBaseURL()==empty (fail-closed, FNL-04)
- [Phase ?]: Port is always 443; CheckFunnelAccess error surfaced verbatim as 400
- [Phase ?]: Ref-count gate: ws.DisableFunnel called ONLY when len(funnelSessions)==0 to protect sibling sessions from premature teardown
- [Phase ?]: Site 4 (daemon stop / handleWebServerStop) NOT double-wired — 165-01 ws.Stop() already calls DisableFunnel
- [Phase ?]: FunnelClientForTest exported type alias enables cross-package fake injection without leaking unexported interface
- [Phase ?]: App.SetSessionFunnel mirrors ToggleWebServing/SetSessionBrowse shape exactly — nil-guard then delegate; no Funnel logic in app.go
- [Phase ?]: SessionInfo.FunnelActive json:"funnelActive" without omitempty — false must serialize so frontend poll detects expiry (mirrors BrowseEnabled rule)
- [Phase ?]: GAP 1 Option A: EnableFunnel proxy target https://localhost → https+insecure://<bindIP>:<port> (FNL-03 502 closed)
- [Phase ?]: GAP 2 kill path: handleDeleteSession calls runSessionExitCleanup synchronously — no grace period, no double-cleanup race (FNL-05 kill path closed)
- [Phase ?]: Phase 167-01: NotifyOnWaiting persisted boolean setting mirrors StartMinimized exactly — no schema bump, no defaults-merge, default OFF (NTF-04)
- [Phase ?]: [Phase 167]: Kept native macOS UNUserNotificationCenter path instead of beeep for macOS -- real AgentHub attribution beats beeep Script Editor fallback
- [Phase ?]: [Phase 167]: Windows/Linux beeep wrappers accept identifier param for signature parity but do not use it; AUMID/app_name branding deferred
- [Phase ?]: Phase 167-03: maybeNotifyWaiting requires a KNOWN previous status (not just first-run) before firing, matching RESEARCH's reference implementation exactly
- [Phase ?]: Phase 167-03: SetNotifyOnWaiting stores the atomic cache before the daemon persist call so the tray poller never reads a stale toggle mid-tick
- [Phase ?]: Phase 167-03: GetNotifyOnWaiting reads only the cached atomic.Bool (no daemon round trip) to keep the poller and Settings toggle in agreement
- [Phase ?]: Phase 167-04: notifyOnWaiting toggle mirrors handleToggleMinimized (instant, no confirm dialog) rather than the shell-warn confirm-on-disable pattern
- [Phase ?]: Phase 167-04: toggle placed under settings-behavior (Behavior section), NOT settings-session-behavior, per the LOCKED user correction overriding NTF-04's original wording
- [Phase ?]: Phase 167-05 (M-41 gap closure): guard sendNotification synchronously BEFORE dispatch_async (not only inside it) — the primary fix, preventing the crash-prone UNUserNotificationCenter call from ever being reached in an unbundled process; @try/@catch retained as defense-in-depth only
- [Phase ?]: Phase 167-05: no test invented for the async dispatch-queue crash path itself (go test never pumps the main queue, would false-pass); the honest, load-bearing assertion is hasAppBundleID() == false under an unbundled go-test binary
- [Phase ?]: Phase 167-06: SetNotifyOnWaiting(true) proactively invokes requestNotificationAuthFunc before the daemon nil-check -- surfaces the macOS permission prompt at toggle-time (leading suspected M-41 fix)
- [Phase ?]: Phase 167-06: instrumented every native-notification branch (attempt/not-granted/authorization-error/delivery) with logging; no live-delivery test invented since go test never pumps the main dispatch queue
- [Phase 167]: Phase 167-07: reused settings-panel__error CSS class for the notification-permission-denied hint (no new CSS); copy uses plain '>' separators matching the plan's action-block wording verbatim.
