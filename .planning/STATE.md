---
gsd_state_version: 1.0
milestone: v4.2
milestone_name: Funnel Sharing & Polish
current_phase: 174
current_phase_name: dependency-updates-dependabot-hygiene
status: executing
stopped_at: Completed 174-01-PLAN.md (4 low-risk CI-action Dependabot bumps applied + PRs closed); next = 174-02
last_updated: "2026-07-08T16:26:02.416Z"
last_activity: 2026-07-08
progress:
  total_phases: 12
  completed_phases: 9
  total_plans: 48
  completed_plans: 47
  percent: 75
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-30 — v4.2 milestone started)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 174 — dependency-updates-dependabot-hygiene

## Current Position

Phase: 174 (dependency-updates-dependabot-hygiene) — EXECUTING
Plan: 3 of 3
Status: Ready to execute

### Roadmap Evolution

- 2026-07-08 — Phase 174 added: Dependency Updates & Dependabot Hygiene (merge 7 low-risk Dependabot PRs; defer #104 Wails / #88 Tailscale / #102 checkout-v7 with dependabot.yml ignore rules + PR-close rationale).
- 2026-07-08 — Phase 175 added: Web-share, Remote-viewer & Windowing Bug Fixes (#128 mobile scaling, #125 disconnect notice, #126 exited-session tab, #119 host/guest empty-window — re-verify #119 vs 168-03 first).
- 2026-07-08 — Phase 176 added: Platform & Hardening Bug Fixes (#124 Linux GUI segfault/DMABUF, #123 /app/ CSP header, #127 Hub preview char-per-row wrapping).
- 2026-07-08 — Issue #120 closed (Phase 169 shipped the feasible honest-detection fix; true auto-connect needs a privileged helper, out of scope).

- **170 Public Share Access Codes (read)** — reusable share-lifetime join code for the INTERNET (PUBLIC) link (FNL-08). UAT found the public link shows no join code (unlike RO/Full) → typing the URL dead-ends on a code page with no code. Root cause: join codes are 5-min single-use (`api.go:259`), wrong for a public share → needs per-code TTL + reusable semantics tied to funnel expiry, read-only only. **170-01 EXECUTED 2026-07-05** (commits 70f2f8ad/fa35b928/1618255b/cc864779/eb119e4c): `JoinCodeManager.IssueReusable`/`Revoke` + reusable-conditional `Exchange` delete, proven at unit layer (4 new tests, 6 pre-existing untouched) AND at the public `/join/exchange` HTTP boundary (new `internal/webserver/join_test.go` — reusable read code resolves twice, single-use code still fails on 2nd try). **170-02 EXECUTED 2026-07-05** (commits 3cb05dea/5f0bb981/d38a9689/835bbaa4, RED/GREEN TDD): `issueCapabilitiesForSession` mints the reusable public read code from the read-only `rTok` ONLY (never `wTok`), caches it per session (idempotent — no rotation on re-issue), and surfaces it on `IssueCapabilitiesResponse.PublicReadCode` (`""` for non-Funnel sessions); `handleSetSessionFunnel` captures `min(ExpiresIn, 8h)` as the per-code TTL; `disableFunnelForSession` (the single teardown chokepoint) revokes the cached code on all 4 in-process triggers (toggle-off, web-share-off, session-exit, auto-expiry timer — daemon-stop intentionally excluded, bypasses this chokepoint by design). Wails TS binding (`models.ts`) regenerated for Wave 3. **170-03 EXECUTED 2026-07-06** (commits b05e22ce/3c93ec57/e10b6067/2ec5c1c5/7b5cfa54): Share modal INTERNET (PUBLIC) section renders `<CodeDisplay label="Public join code (reusable):" code={publicReadCode} />`; `SessionShareModal` threads `resp.publicReadCode` from the existing warm-up round-trip and clears it on disable. Deviation: the type field went into the *actually-imported* hand-authored stub `frontend/src/wailsjs/go/main/App.d.ts` (not just the unused generated `models.ts`) — required for `tsc --noEmit` to pass. Frontend gate: `tsc --noEmit` clean + `vite build` OK + 2335 vitest pass. **170-04 EXECUTED 2026-07-06** (commits 805dbf87/d1c8ddda): TESTING.md reconciled — counts already correct (528 total: 375 Go/142 vitest/9 PW/2 build), added Section-5 category with **M-46** live off-tailnet reusable-code-join + teardown item; `check-traceability-paths.sh` exits 0. **PHASE VERIFIED (code) 2026-07-06** — gsd-verifier: 14/14 must-haves VERIFIED including the security-critical read-only-only mint (`IssueReusable(rTok, …)`, never `wTok`); status `human_needed`, sole open item = M-46 (needs real Funnel tailnet + 2 off-tailnet devices). ROADMAP/REQUIREMENTS completion markers reverted to pending per user (keep-pending-UAT). NEXT = `/gsd-verify-work 170` (record live M-46 pass → phase complete).
- **171 Public Full-Access (RW) Sharing** — opt-in public read-write behind a hard consent gate + single-use write code (FNL-09). Supersedes today's ACCIDENTAL public write (issueCapabilitiesForSession rebases both read+write caps to the funnel base by timing; funnel exposes whole mux, no read-only downgrade). **SPEC-FIRST** (internet RCE): `/gsd-spec-phase 171` → discuss → `/gsd-secure-phase` → plan.

**Live UAT of 165-169 (2026-07-05, prod build on real Funnel tailnet) — Phase 166 Funnel UAT NOW COMPLETE, all 4 PASS (166-UAT.md status→passed):** M-37 (off-tailnet phone QR loads read-only session; public URL 200 off-host), M-38 (daemon 60s auto-expiry → torn down, URL HTTP 000, badge cleared, no manual disable), M-39 (globe + INTERNET badge appears/clears), M-40 (stop tailscale + restart daemon → local mode → toggle greyed + "requires Tailscale" + backend 400 fail-closed; reconnect auto-upgrades → toggle re-enables). **2 Share-modal layout bugs found + fixed live (commit 27d398e7):** (1) `.hub-share-modal` had no height bound → overflowed viewport/clipped; fixed with max-height. (2) that max-height made `__body` a constrained flex column and the risk panel's `overflow:hidden` let flexbox shrink it 202px→39px, clipping the Auto-expire select + Enable CTA → Funnel uncommittable via UI; fixed with `flex-shrink:0`. Both root-caused via dev-browser CSS harness, verified live. OBSERVATION (not blocking, maybe file): daemon does NOT live-downgrade to local mode on a mid-session tailscale drop (stays tailscale-mode w/ stale IP) — local fallback is startup-only; auto-upgrade on reconnect works. REMAINING deferrals: M-41 (notif delivery win/linux, needs those platforms), M-45 (non-admin macsys, env-only). **Phase 172 (Hub-card layout & badge refinement) CREATED 2026-07-05** (`/gsd-phase add`; frontend-only, Depends on: None — independent of Funnel 170/171). NEXT = user wants 2-3 throwaway HTML mockups (frontend-design / /gsd-sketch) BEFORE `/gsd-plan-phase 172`. Design critique captured: the Hub session card (image ref) uses THREE inconsistent metadata treatments — `Running`/`Local` = icon+plain-text, `/bin/zsh` = outlined pill, `INTERNET` = filled green pill on its own row — loosely stacked with no grouping. Direction: consolidate into ONE consistent chip row (agent · origin · exposure) with tighter vertical rhythm, while KEEPING the INTERNET chip the one deliberately-prominent colored/filled chip (it's a security-exposure signal that must stay unmissable + colorblind-safe per [[user_colorblind]]; making others quieter/outlined makes INTERNET pop MORE by contrast). Frontend-only (Hub card component + style.css); no backend. User wants 2-3 throwaway HTML mockups (frontend-design skill) BEFORE touching code. This is a v4.2 phase (milestone reopened). To add: `/gsd-phase add ...` (will become 172).
Last activity: 2026-07-08

```
v4.2 Progress: [█████████████████░░░] 86% (6/7 phases — 165 ✅, 166 ✅ (+2 modal fixes 2026-07-05), 167 ✅ (M-41 deferred), 168 ✅, 169 ✅ (M-45 deferred), 170 ✅ (M-46 live UAT PASSED 2026-07-06, fix 5a92ddae); 171 ⬜ spec-first)
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
| 166 | Funnel Frontend + Help Guide | FUI-01, FUI-02, FUI-03, FUI-04, FUI-05, FUI-06, HLP-01, HLP-02 | ✅ DONE — verified 8/8 (2026-06-30); human UAT fixed 4 CSS defects (f3c8d848); M-37–M-40 pending prod-build UAT |
| 167 | Native Notifications | NTF-01, NTF-02, NTF-03, NTF-04 | ✅ DONE — 7/7 plans, code-verified 11/11; **M-41 live delivery DEFERRED** to release-time UAT (signed builds, unautomatable) |
| 168 | Bug Fix & Settings Polish | FIX-01, FIX-02, FIX-03, FIX-04, UX-01, UX-02 | ✅ DONE — 9/9 plans (incl. 168-08/09 gap-closures); verified 6/6, live UAT 4/4 PASS (#112/#115/#116/#117/#118/#121 code-fixed) |
| 169 | Tailscale Detection Fix | FIX-05 | ⬜ Not started — next: /gsd-plan-phase 169 (#120; orthogonal, non-admin macOS test env) |
| 170 | Public Share Access Codes (read) | FNL-08 | ✅ DONE — 4/4 + code-verified 14/14 + M-46 live UAT PASSED (2026-07-06); live blocker fixed (5a92ddae: public URL → /join?code=) |
| 171 | Public Full-Access (RW) Sharing | FNL-09 | ⬜ Not started — reopen phase; SPEC-FIRST: /gsd-spec-phase 171 |
| 172 | Hub-card Layout & Badge Refinement | D-01..D-07 | ✅ DONE — 1/1 plan; verified passed; UAT 6/6 PASS (2026-07-08, D5 theme parity confirmed; UAT-06 Open/Share↔preview spacing found + fixed inline c7a5b69b) |
| 173 | Share Modal Three-Tab Segmented Redesign | SM-01..SM-08 | ⬜ Not started — GitHub #129 (frontend-only UX/IA; spec already detailed); next: /gsd-plan-phase 173 |

**Total:** 26 requirements mapped across 5 phases (100% coverage). *(2026-07-01: +FIX-04 #121 phantom viewer count into Phase 168; +FIX-05 #120 Tailscale detection split into new Phase 169.)*

### Roadmap Evolution

- Phase 173 added (2026-07-08): Share modal three-tab segmented redesign — GitHub #129; frontend-only UX/IA reorg (fixed control strip + Tailnet/Internet-RO/Internet-Full-access segmented tabs, public-write walled off, reusable ShareLinkCard). First of a planned series to work through outstanding GitHub bug issues one at a time.

### Key Sequencing Constraints

- Phase 165 must precede Phase 166 (Funnel UI depends on backend atomicity — partial deployment causes silent 403s for external guests)
- Phase 165's dual-origin fix may resolve the Funnel multi-viewer kick in #117; verify in Phase 165 UAT before scoping Phase 168's #117 relay buffer work
- Phases 167 and 168 are independent of Funnel; ordering is for focus, not correctness
- Phase 169 (Tailscale detection, #120) is independent of everything else — orthogonal subsystem; can be planned/executed any time. Split from 168 (2026-07-01) to isolate its non-admin-macOS test environment from the web-share/Hub verification surface.
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
| manual_uat | **Phase 167 M-41** — native notification live on-screen delivery (macOS/Windows/Linux) on a SIGNED PRODUCTION BUILD; toggle-on shows exactly one AgentHub-attributed toast on non-waiting→waiting while tray-hidden, toggle-off shows none | deferred (release-time; inherently unautomatable — `wails dev` returns documented no-bundle state; code-verified 11/11 + M-41 crash-hardened 167-05) |
| manual_uat | **Phase 166 M-37–M-40** — Funnel Share modal / help-guide prod-build UAT (carry-forward from partial 4/8 human UAT) | deferred (release-time, prod build) |
| manual_uat | Phase 125 editor on-screen render + CodeMirror Tab/Cmd-V in WebView | pending (live app required) |
| manual_uat | Phase 126 `$EDITOR` suspend-resume terminal restore | pending (live app required) |
| manual_uat | Phase 124 home-dir warning banner on-screen | pending (live app required) |
| v4.3+ | web-plugin-hot-swap (#112 SKIPPED): web guests lost /api/plugin-config + SSE after Phase 159 redirect | FIX-01 in v4.2 Phase 168 addresses this |
| v4.3+ | Device-share automation via Tailscale admin API (FUT-01) | Out of scope for v4.2 per Issue #107 |

## Session Continuity

Last session: 2026-07-08T16:26:02.406Z
Stopped at: Completed 174-01-PLAN.md (4 low-risk CI-action Dependabot bumps applied + PRs closed); next = 174-02
Resume file: None
Next action: Phase 169 (Tailscale Detection Fix, #120) is the last open v4.2 phase — FIX-05: non-admin macOS accounts report Tailscale "installed but not Connected" because `macsys` `sameuserproof` is unreadable; add a CLI `status` fallback. Run `/gsd-plan-phase 169` to begin. Deferred release-time UATs (Phase 167 M-41, Phase 166 M-37–M-40) are tracked in Deferred Items and run on signed production builds at release time.

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
| Phase 168 P01 | 5min | 2 tasks | 5 files |
| Phase 168 P02 | 11min | 2 tasks | 3 files |
| Phase 168 P03 | 8min | 2 tasks | 4 files |
| Phase 168 P04 | 12min | 3 tasks | 8 files |
| Phase 168 P05 | 29min | 2 tasks | 7 files |
| Phase 168 P06 | 9min | 3 tasks | 10 files |
| Phase 168 P07 | 15min | 2 tasks | 1 files |
| Phase 169 P01 | 13min | 2 tasks | 3 files |
| Phase 170 P01 | 5min | 3 tasks | 3 files |
| Phase 170 P02 | 9min | 2 tasks | 6 files |
| Phase 170 P03 | 4min | 3 tasks | 5 files |
| Phase 170 P04 | 8min | 2 tasks | 1 files |
| Phase 171 P01 | 9min | 3 tasks | 6 files |
| Phase 171 P02 | 26min | 3 tasks | 20 files |
| Phase 171 P03 | 21min | 3 tasks | 10 files |
| Phase 171 P04 | 11min | 3 tasks | 3 files |
| Phase 172 P01 | 6min | 3 tasks | 5 files |
| Phase 173 P01 | 5min | 3 tasks | 1 files |
| Phase 173 P02 | 3min | 2 tasks | 3 files |
| Phase 173 P03 | 5min | 2 tasks | 2 files |
| Phase 173 P05 | 13min | 3 tasks | 6 files |
| Phase 173 P06 | 55min | 3 tasks | 5 files |
| Phase 173 P07 | 12min | 2 tasks | 1 files |
| Phase 173 P08 | 4min | 3 tasks | 5 files |
| Phase 174 P01 | 5min | 3 tasks | 3 files |
| Phase 174 P02 | 8min | 3 tasks | 2 files |

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
- [Phase ?]: RemoteViewerCount is a new, separate Hub method — SubscriberCount left completely unchanged (still consumed by relay/server.go's NotifyViewerCount MsgMeta frame).
- [Phase ?]: Raw per-connection count (no PersonKey collapse) for RemoteViewerCount, matching D-01/D-02.
- [Phase 168]: Phase 168-02: WebShareSessionView baseURL seam — apiBaseURL/wsURL derive from resolved origin (baseURL ?? window.location.origin), not a hardcoded window.location reference, so FIX-03 (168-03) can reuse it for remote-peer tabs
- [Phase 168]: Phase 168-02: isWebGuest = pluginConfig === undefined (not == null); effective plugin config forwarded to TerminalPanel only (ChatPanel has no pluginConfig prop)
- [Phase 168]: Phase 168-03: pluginConfig is suppressed (undefined) for remote-peer web-session tabs so WebShareSessionView's web-guest self-fetch (168-02) fires instead of applying this daemon's own plugin config to a different peer's session
- [Phase 168]: Phase 168-03: OpenRemoteSessionURL's daemon-composed URL is parsed client-side (origin -> baseURL, cap param -> capToken) and handed to openWebSessionTab instead of BrowserOpenURL, preserving the WR-01 SID-correctness guarantee
- [Phase 168]: Phase 168-04: App.GetStayOnHubAfterCreate/SetStayOnHubAfterCreate is a plain client passthrough (no atomic cache), unlike NotifyOnWaiting, since there is no background tray-poller reading it
- [Phase 168]: Phase 168-04: createTab reads stayOnHubAfterCreate via a useRef (mirrors autoCloseRef), not React state, to avoid useCallback dep churn and stale closures
- [Phase 168]: Phase 168-04: no fromHub flag introduced - createTab's single setActiveId(sessionId) call is the only auto-switch in the app (D-11), so gating that one call site is sufficient
- [Phase 168]: shareModalSession lifted to App.tsx; single <SessionShareModal> render (not inside HubPanel, which unmounts off the Hub tab); footer 'Share Session' calls onShareSession, never ToggleWebServing directly (D-14)
- [Phase 168]: handleShellWebShareConfirm now sources sessionId from shareModalSession instead of the retired pendingShellWebToggle, fixing a latent no-op bug in the modal's own shell-warn confirm flow
- [Phase 168-06]: DisconnectWebViewers reuses Subscriber.CloseSlow (the same close-on-full mechanism broadcastResize uses) instead of a new termination path
- [Phase 168-06]: Disconnect RPC registered on the daemon-local api.go mux (same trust boundary as ToggleWebServing/SetSessionFunnel), never a guest-reachable /api/... route (T-168-07)
- [Phase 168-06]: No eviction-on-subscribe logic added; TestHub_TwoWebOriginSubscribers_NoEviction proves #117 Part A does not reproduce in current code
- [Phase 168-06]: Disconnect drops connections only (D-06) -- never calls ToggleWebServing(false) or revokes the capability
- [Phase ?]: Phase 168-07: fixed pre-existing Suite Manifest Total off-by-one (524->525)
- [Phase ?]: Phase 168-07: plans 168-01..06 self-registered TESTING.md changes inline per-plan rather than deferring to this isolated final plan; 168-07 closed remaining gaps (FIX-04 traceability, M-13 reword, M-42/M-43 manual items)
- [Phase ?]: Phase 169-01: reused tailscale.com/ipn/ipnstate.Status for CLI JSON unmarshal instead of a new local struct — mirrors the SDK-success field-mapping exactly
- [Phase ?]: Phase 169-01: cliStatusFunc fires on ANY SDK error, on ALL platforms — no error-string classification, no runtime.GOOS gate (D-03/D-04)
- [Phase 170-01]: IssueReusable reuses Issue's exact crypto/rand + joinCodeEncoding path (no new RNG surface); Exchange's success-path delete gated on entry.reusable while expiry-path delete stays unconditional for both classes
- [Phase ?]: Phase 170-02: a.mu held across IssueReusable (no blocking I/O) closes the concurrent-mint TOCTOU race for the Funnel public read code
- [Phase ?]: Phase 170-02: PublicReadCode lives only on IssueCapabilitiesResponse (not SetSessionFunnelResponse) — rides the frontend's existing warm-up re-issue
- [Phase ?]: Phase 170-02: daemon-stop (site 4) intentionally excluded from public-read-code revocation assertions — ws.Stop() bypasses disableFunnelForSession by design (165-02 precedent)
- [Phase ?]: Phase 170-03: App.d.ts hand-authored IssueCapabilitiesResponse interface (distinct from daemon.IssueCapabilitiesResponse in models.ts) needed publicReadCode added too -- two separate wails TS binding files exist; 170-02 synced one, 170-03 synced the other (precedent 887975df)
- [Phase ?]: Phase 170-03: publicReadCode kept as Funnel-scoped local state in SessionShareModal, separate from CachedShare (single-use RO/Full-Access codes)
- [Phase ?]: Phase 170-04: TESTING.md Suite Manifest/traceability rows were already reconciled by 170-01/02/03's incremental docs commits (no drift); added Section 5 Category X / M-46 (live off-tailnet reusable-code join + share-lifetime teardown)
- [Phase ?]: [Phase 171-01]: IssueSingleUseWithTTL reuses IssueReusable's exact RNG/encoding path; reusable stays false (zero value) so Exchange's atomic delete-on-first-redeem is reused verbatim
- [Phase ?]: [Phase 171-01]: RemoveGrant is a new surgical sibling to AddGrant/ClearGrants (not a ClearGrants reuse) — ClearGrants would wipe a session's unrelated tailnet read/write grants alongside the gate-minted write grant
- [Phase ?]: [Phase 171-01]: originAllowedForWrite gained a sessionID param (from claims.SID at its sole call site in requireFilesWrite) so the Funnel-origin branch can consult isRWGated; tailnet-origin branch and fail-closed empty-BaseURL behavior unchanged
- [Phase ?]: [Phase 171-02]: D-04 split issueCapabilitiesForSession's single base var into readBase/writeBase -- WriteURL (tailnet Full Access Link) never rebases to FunnelBaseURL, closing the accidental public-write gap (T-171-07)
- [Phase ?]: [Phase 171-02]: revokeFunnelWriteLocked is the single shared write-cap teardown -- called from the RW-only DELETE disable path AND appended inside disableFunnelForSession's cascade (D-03), and re-invoked at mint time to prevent a re-mint from leaking a stale grant/code/timer
- [Phase ?]: [Phase 171-02]: ExpiresIn on the gate-minted write cap is clamped unconditionally to (0, 3600] server-side -- deliberately NOT the read Funnel handler's ExpiresIn==0-means-unbounded semantics (R5/D-11/Pitfall 6)
- [Phase ?]: [Phase 171-03]: writeGateUsed left as an undriven controlled prop this phase -- no backend redemption signal exists (SessionInfo.funnelWriteActive stays true across redemption); 171-04's M-47 live UAT is the real verification
- [Phase ?]: [Phase 171-03]: Hold-to-confirm completion is timer-authoritative (3000ms setTimeout), not pointerup-authoritative -- release is only ever a cancellation path
- [Phase 171-04]: Suite Manifest correction pattern: add a NEW dated note stating old->new counts rather than editing a prior plan's historical note
- [Phase 171-04]: 171-SECURITY.md separates D-01 grant-registration gating (primary terminal-write barrier) from D-02 originAllowedForWrite (defense-in-depth for files.write only, does not reach MsgInput/MsgSessionInject)
- [Phase 171-04]: M-47 is one end-to-end live UAT script (hold-gate, off-tailnet redeem+execute, second-redemption-fails-closed, read-spectator-coexists, disable-revokes-writer-read-survives) mirroring the M-46 precedent
- [Phase 172-01]: Kept .hub-card__badge CSS unchanged in style.css since HubModal.tsx's session-picker chip still consumes it (confirmed via grep before removal)
- [Phase 172-01]: Origin chip resolves to --hub-text-muted only, dropping green-local/blue-remote color coding entirely (Sketch 001 Variant B D-01 pin)
- [Phase 172-01]: Exposure cluster (.hub-card__exposure) renders only when funnelActive || funnelWriteActive, avoiding an empty wrapper on non-exposed cards
- [Phase 173-01]: New token --hub-share-seg-active-bg mirrors --hub-sidebar-item-active-bg's existing accent-tint pattern rather than inventing a new visual language
- [Phase 173-01]: Did not add a --hub-danger-line token; reused --hub-destructive for the danger ring per RESEARCH guidance
- [Phase 173-01]: .hub-share-modal__body keeps overflow-y:auto as an outer safety bound; .hub-share-modal__tabpanel (new) is the region documented to actually scroll
- [Phase 173]: D-07 vs D-09 resolved: HoldToConfirmButton reduced-motion fallback is additive (plain single-click confirm), 3s hold safety-gate contract unchanged
- [Phase 173]: formatCountdown stays in SessionSharePanel.tsx (not moved to shared.tsx) — only used by the panel's own JSX, not by either hoisted component
- [Phase ?]: Phase 173-03: ShareSegmentedControl prefixes the warning glyph onto the danger tab's sub label itself (belt-and-suspenders with the is-danger CSS ring), independent of what the shell passes
- [Phase ?]: Phase 173-03: disabled segment sub text is always 'N/A' per plan must_haves, overriding the DESIGN sketch's illustrative 'Off'
- [Phase ?]: Phase 173-05: InternetFullAccessTab implements the Idle -> Gate-open -> Armed 3-state flow per the plan's explicit must_haves/behavior spec, an additive local-state change over the simpler always-visible-hold flow in the pre-existing SessionSharePanel.tsx
- [Phase ?]: Phase 173-05: InternetReadOnlyTab's ShareLinkCard receives the pre-computed /join?code=... exchange URL (never the raw funnelUrl cap link) as its url prop, so Copy/Open/QR can never leak the ephemeral capability token
- [Phase 173]: Phase 173-06: funnelError renders in both the transient confirm view and inside .hub-share-modal__tabpanel — the full-access gate-confirm failure sets the same slot but fires once past the confirm view
- [Phase 173]: Phase 173-06: React.act() wrapping (not bare flushSync+setTimeout(0)) required for async-RPC-then-passive-effect test sequences — verified flake-free across 20+ repeated runs
- [Phase ?]: Phase 173-07: Suite Manifest correction pattern reused (171-04 precedent) — new dated note stating old->new counts, not editing a prior plan's note
- [Phase ?]: Phase 173-07: check-traceability-paths.sh's grep -oP is unreliable on macOS BSD grep (false-OK, zero paths checked); validated path correctness manually via equivalent Python regex instead
- [Phase ?]: Phase 173-08: funnelOn resync effect keyed only on session.funnelActive (not funnelOn) to avoid stomping handleFunnelEnable's optimistic setFunnelOn(true) during warm-up
- [Phase ?]: Phase 173-08: Internet toggle checked/aria-checked stays funnelOn || riskPanelOpen; only the TEXT state label is gated strictly on funnelOn (pending window renders 'Confirm…')
- [Phase 174]: Phase 174-01: Applied 4 low-risk CI-action Dependabot bumps (setup-go, pnpm/action-setup, attest-build-provenance, action-gh-release) directly on v4.2-funnel-sharing rather than merging Dependabot PRs into main; closed PRs #114/#113/#103/#85 citing Phase 174.
- [Phase ?]: Phase 174-02: Committed Task 1 (coder/websocket) and Task 2 (x/term + nfpm) as two separate atomic commits rather than the plan's suggested single Task-3 combined commit, for independent bisectability.
- [Phase ?]: Phase 174-02: Left the transitive go.mod toolchain directive bump (1.26.3 -> 1.26.4) in place from go mod tidy; local toolchain 1.26.5 already satisfies both, no observable effect.

### Blockers

- Phase 174-02: Dependabot PRs #89, #106, #105 remain OPEN — gh pr close blocked by runtime's auto-mode permission classifier (external-system-write guardrail). Go module bumps themselves done/verified/committed. Needs user to explicitly authorize closing these PRs or close manually (see 174-02-PLAN.md Task 3 for exact comment text).
