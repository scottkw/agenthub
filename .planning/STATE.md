---
gsd_state_version: 1.0
milestone: v4.1
milestone_name: Session Chat
current_phase: 159
current_phase_name: web-share-chat-parity-route-shared-session-links-to-the-chat
status: needs-uat
stopped_at: "Phase 159 EXECUTED + automated-verified (5/5 must-haves, gsd-verifier status human_needed). 159-01 redirect: handleTerminalPage now 302-redirects /sessions/{id}?cap= → /app/?session=&cap= after requireCapability; RO/RW identical; CSP/Cache-Control invariants preserved; webserver tests green. REQUIREMENTS.md: added WEBCHAT-01/02, credited Phase 159 in PARITY-01 traceability. AWAITING live-daemon UAT M-31 (4 tests: redirect→SPA, chat round-trip, RO participation, scale parity). Pre-existing flake TestRelay_MixedReplyAndKeystrokes fails identically on base — NOT a 159 regression. Next: run M-31 live UAT, then /gsd-execute-phase 160. Milestone close-out still owes: push main→origin + re-run verify-work 156 Test 1"
last_updated: "2026-06-27T18:40:45.454Z"
last_activity: 2026-06-27
last_activity_desc: Phase 159 executed + automated-verified; awaiting live UAT M-31
progress:
  total_phases: 12
  completed_phases: 8
  total_plans: 36
  completed_plans: 36
  percent: 67
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-25 — v4.1 milestone started)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 159 — web-share-chat-parity-route-shared-session-links-to-the-chat

## Current Position

Phase: 159 (web-share-chat-parity-route-shared-session-links-to-the-chat) — AWAITING UAT
Plan: 5 of 5 complete (159-01 redirect + 159-02 hide-sidebar + 159-03 file-tab-gating + 159-04 tab-rename-suppression + 159-05 chat-name-truncation)
Status: Automated verification PASSED; three live-UAT gap-closures landed (159-02 hid the desktop sidebar; 159-03 stops the dead files.read tab; 159-04 removes the desktop-only tab menu/rename from web guests — rename never reached the host, verified, but affordance removed). Fresh app rebuilt (20:09). Awaiting remaining two-peer live checks (bidirectional chat, RO chat, resize scale).
Last activity: 2026-06-27 — Phase 159 executed + automated-verified; awaiting live UAT M-31

```
Progress: [█████████████░░░░░░░] 8/12 phases complete (67%); Phase 159 code done, UAT pending
```

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260627-d81 | Fix Welcome screen layout — long Linux install URL broke alignment (flexbox min-width:auto); install boxes off-center / mismatched sizes | 2026-06-27 | a89cf26a | [260627-d81-fix-welcome-screen-layout-linux-install-](./quick/260627-d81-fix-welcome-screen-layout-linux-install-/) |

## Operator Next Steps (carry-forward from v4.0)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time, first WinGet submission): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Set before triggering `distribute.yml` for INSTALL-03; unset after `microsoft/winget-pkgs` accepts the first submission PR.
3. **Branch protection on `main`**: applied 2026-06-22 — 5 required checks (4 build matrix + `playwright`; `strict:false`, `enforce_admins:false`). Still valid for v4.1 CI.

## v4.1 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 151 | Message Schema + ChatStore | PERSIST-01, PERSIST-02, PERSIST-03 | Complete |
| 152 | Relay Protocol + Identity + Presence | IDENT-01, IDENT-02, PRESENCE-01, PRESENCE-02 | Complete |
| 153 | @session PTY Bridge | MENTION-02, MENTION-03, SEC-01, SEC-02 | Complete |
| 154 | Desktop Chat UI | CHAT-01, CHAT-02, CHAT-03, CHAT-04, MENTION-01, NOTIF-01, NOTIF-02, SEC-03 | Complete |
| 155 | Web-Share Chat UI + Cross-Surface Parity Gate | EXPORT-01, PARITY-01 | Complete |
| 156 | Install Links & Distribution | INSTALL-01, INSTALL-02, INSTALL-03 | Complete |
| 157 | Terminal Screen-Share Semantics (Issue #109) | VIEW-01, VIEW-02, VIEW-03, VIEW-04, VIEW-05 | Complete |
| 158 | Chat affordance polish (toggle/Send overlap + chat on terminal tab) | CHAT-FIX-01, CHAT-PARITY-01 | Complete |
| 159 | Web-Share Chat Parity (remote web guests get chat) | WEBCHAT-01, WEBCHAT-02, PARITY-01 (upstream) | Complete (UAT pending — M-31) |
| 160 | v4.1 Chat Closeout (NOTIF-01 + 153/154/156 tech debt) | NOTIF-01, tech-debt closeout | Not planned |
| 161 | Chat-Sidebar Alias Control (set display name from shared ChatPanel) | ALIAS-UI-01, ALIAS-UI-02 | Not planned |
| 162 | Settings Polish — Terminal Plugins jump link (#108) | SETTINGS-UI-01 | Not planned |

**Total:** 29 requirements mapped across phases 151–157 (100% coverage); 158 complete (CHAT-FIX-01, CHAT-PARITY-01); 159–162 requirements set at plan time. v4.1 = 12 phases (closeout audit runs after 162; see ROADMAP closeout-ordering note).

### Roadmap Evolution

- Phases 161 + 162 added (2026-06-27): v4.1 expanded to 12 phases. **161 Chat-Sidebar Alias Control** — the Phase 152 alias backend (`MsgAliasSet` 0x34, `AliasStore`/aliases.json, `ValidateAlias`, `ChatMessage.alias`/`PresenceEntry.alias`) shipped with NO UI; the desktop owner gets a default alias (host/computed name) but can't change it, and `encodeAliasSetFrame()` is called only in tests. User first suggested a Settings → Profile section, then (correctly) moved it to the **shared `ChatPanel` sidebar** — because the web-share surface has no Settings page, so a sidebar control gives GUI tab + Hub modal + web-share guest (via the 159 redirect) the control from ONE shared component = cross-surface parity by construction ([[feedback_cross_surface_parity]]). Reuses the existing wire path; may avoid new Wails bindings. **162 Settings Polish (#108)** — move the "Plugins" Settings jump link to last + rename to "Terminal Plugins" (anchor id stable); independent of chat. Decoupled from 161 once the alias work left the Settings page. Closeout (160) keeps its number to avoid breaking committed Phase-160 refs in the 159 plan/research; milestone-close audit runs after 162.
- Phases 159 + 160 added (2026-06-27): milestone reopened 8/10. **159 Web-Share Chat Parity** — remote web guests can't chat: the share flow hands out `/sessions/{id}?cap=` (raw `terminal.js` viewer, no chat, discards frames 0x30–0x34), while the chat-capable `/app/` React SPA is never linked. Phase 155 verified PARITY-01 on `/app/` — a surface no remote guest is ever sent to (false-parity, [[feedback_tests_encoding_same_wrong_assumption]]). DECIDED approach (user, 2026-06-27): redirect `/sessions/{id}?cap=` → `/app/?session=&cap=` (server-side, `internal/webserver/server.go`); also unblocks the upcoming Tailscale Funnel session-sharing milestone. **160 v4.1 Chat Closeout** — NOTIF-01 Hub-card unread-badge dead-wiring (audit BLOCKER) + minor 153/154/156 tech debt. Split from web-chat per user (159 = feature/parity, 160 = cleanup).
- Phase 158 added (2026-06-27): chat affordance polish — found during v4.1 UAT. (1) BUG: `.hub-modal__chat-toggle` (bottom-right, z-index 6) covers the chat composer Send button when the drawer is open (drawer z-index 5, 360px, right:0) — fix: shift toggle left of the drawer (`right:372px`) while `chat-panel--open` (CHAT-FIX-01). (2) PARITY: chat toggle + ChatPanel existed only in `HubInteractiveModal` and web-share, not the raw session terminal tab — added via TerminalChatHost (CHAT-PARITY-01). COMPLETE 2026-06-27, UAT 2/2.

## Key Decisions (v4.1)

| Decision | Resolution |
|----------|------------|
| Phase ordering | Store before transport before UI: JSONL schema is the wire contract everything serializes to; late schema changes cascade through all layers |
| Identity in Phase 152 not 154 | `TailnetID` must be stamped on Subscriber before any message is stored; retrofitting after storage is wired requires a migration |
| `@session` as dedicated Phase 153 | Highest-security-risk new capability; isolated phase gives full test focus to sanitizer and cap-check without rushing under UI phase deadline |
| SEC-01/02 with Phase 153 | Security requirements are embedded in the phase that introduces their risk, not deferred to a trailing hardening phase |
| SEC-03 with Phase 154 | XSS risk lives in the Markdown-rendering UI; embed the test there, not in a separate phase |
| Web UI last (Phase 155) | Shares ChatPanel.tsx built in Phase 154; parity gate belongs here as a wholistic cross-surface pass |
| INSTALL as Phase 156, independent | No dependency on chat phases; orthogonal; can be planned/executed in parallel with any chat phase |
| PARITY-01 is release-blocking | Per standing rule: GUI/CLI/web cross-surface parity is never deferrable; Phase 155 is the release gate |
| Phase 154 chat drawer: push → overlay (2026-06-26) | D-02 revised from push mode (terminal shrinks → PTY resize) to overlay (drawer floats over terminal, no resize). Push mode fights the host-authority PTY model adopted for Issue #109. Tradeoff: drawer covers ~360px of terminal while open |
| Phase 157 added for Issue #109 (2026-06-26) | Terminal screen-share semantics (Option B): host is single source of truth for PTY grid, guests conform + CSS scale-to-fit. Appended to v4.1; orthogonal to chat. Full Option B scope |

## Architecture Notes (v4.1)

- **New files:** `internal/daemon/chat.go` (ChatStore, JSONL), `frontend/src/components/Hub/ChatPanel.tsx`
- **Modified files:** `internal/relay/protocol.go` (frame constants 0x30–0x34), `internal/relay/hub.go` (BroadcastChat, BroadcastPresence), `internal/relay/server.go` + `internal/webserver/server.go` (MsgChatSend dispatch, WhoIs at upgrade), `internal/daemon/engine.go` (chatStores map, KillSession teardown), `frontend/src/components/Hub/HubInteractiveModal.tsx`
- **New npm packages:** `@tanstack/react-virtual` v3.14.3, `react-textarea-autosize` v8.5.9
- **No new Go modules** — all server-side capabilities use existing `go.mod` entries

## Deferred Items (carry-forward from v4.0)

| Category | Item | Status |
|----------|------|--------|
| operator_runtime | `RELEASE_PUBLISH_TOKEN` PAT | pending (one-time, before next release) |
| operator_runtime | `WINGET_FIRST_SUBMISSION=true` variable | pending (needed for INSTALL-03) |
| manual_uat | Phase 125 editor on-screen render + CodeMirror Tab/Cmd-V in WebView | pending (live app required) |
| manual_uat | Phase 126 `$EDITOR` suspend-resume terminal restore | pending (live app required) |
| manual_uat | Phase 124 home-dir warning banner on-screen | pending (live app required) |

## Session Continuity

Last session: 2026-06-27T18:40:45.445Z
Stopped at: Phase 157 Plan 01 complete — host-authority ResizeClient + broadcastResize + Rows() in hub.go, MC-06 tests replaced
Resume file: None
Next action: `/gsd:plan-phase 151`

## Decisions

- [Phase ?]: Reject-not-trim at cap: ErrChatCapReached returned instead of trimming oldest message
- [Phase ?]: Injected baseDir: NewChatStore accepts baseDir parameter enabling constructor-level test isolation with t.TempDir()
- [Phase ?]: Relay chat routes in wrapRelayWithChat outer wrap — avoids daemon↔relay import cycle
- [Phase ?]: Webserver chat uses provider callback not daemon import — T-151-09 circular import prevention
- [Phase ?]: RWMutex over Mutex
- [Phase ?]: AliasStore rolls back in-memory map on persist failure to keep store consistent with disk
- [Phase ?]: SetIdentityProviders setter avoids relay→daemon import cycle
- [Phase ?]: Phase 152-06
- [Phase ?]: Phase 153-01
- [Phase ?]: Phase 153-01
- [Phase ?]: Web inject case structurally identical to relay case — shares hub.HandleInject, no direct WriteInput
- [Phase ?]: assertNoFrameType removed from server_inject_test.go (unused, lint-clean)
- [Phase ?]: Phase 154-01: HandleChatSend uses SanitizeChatContent not SanitizePTYText; silent-drop on error (no NAK)
- [Phase ?]: alias field in ChatMessage mirrors Go json:"alias" tag — not authorAlias (RESEARCH Pitfall 4)
- [Phase ?]: all new RelayClientCallbacks members optional (?) for TerminalPanel backward compat (RESEARCH Pitfall 2)
- [Phase ?]: Pitfall 7: Enter always routes to sendChat, never to inject — strictly separate code paths
- [Phase ?]: draftRef liveRef pattern: draft mirrored to ref inline during render so 600ms inject timer reads non-stale draft
- [Phase ?]: D-02 overlay mode: ChatPanel position:absolute over terminal, isActive unchanged by chatOpen, no PTY resize on toggle
- [Phase 155-01]: Export() always double-quotes participant values — simpler YAML safety invariant, no conditional logic needed for special-char detection
- [Phase ?]: Phase 155-02: isReadOnly defaults to true (fail-safe); RO resolved from /info ?cap= perms; wsURL short-circuits before port=0 loopback
- [Phase ?]: Phase 155-03: isActive=true constant on WebShareSessionView (no grow animation, unlike modal)
- [Phase ?]: Phase 155-03: openWebSessionTab called after handleOpenFileBrowser so session tab wins active focus
- [Phase ?]: Phase 155-06: SC-3 WS-ready gate uses .first() not text-filter — virtualizer scrolls seeded message out of DOM after broadcast tests
- [Phase ?]: python3 yaml.safe_load for portable YAML validation in winget dry-run
- [Phase ?]: WINGET_TOKEN public_repo scope only (T-156-07 least-privilege, distribute.yml line 79)
- [Phase ?]: Phase 157-01: VIEW-02 origin gate is FIRST check in ResizeClient — non-local returns immediately before lock (T-157-01 single enforcement point)
- [Phase ?]: Phase 157-01: broadcastResize self-acquires mu, called only after hub.mu.Unlock() — prevents T-157-04 self-deadlock
- [Phase ?]: Phase 157-01: Hub.Rows() fallback is 50 (engine.go emuRows), mirrors Cols() 220 fallback
- [Phase ?]: Added VIEW-01..03 rows to TESTING.md Section 4; extended Section 2 Phase 157 delta note; added M-27/M-28 to Section 5 Category P
- [Phase ?]: TerminalChatHost wraps only TerminalPanel so StatusBar is not covered
- [Phase ?]: D-02 invariant: overlay toggle never triggers PTY sendResize
- [Phase ?]: WEBCHAT-01 redirect to /app/
- [Phase ?]: url.QueryEscape for cap token round-trip; terminal.html preserved

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
