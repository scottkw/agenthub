---
gsd_state_version: 1.0
milestone: v4.1
milestone_name: Session Chat
current_phase: 154
current_phase_name: desktop-chat-ui
status: executing
stopped_at: Phase 154 UI-SPEC approved
last_updated: "2026-06-26T19:03:03.894Z"
last_activity: 2026-06-26
last_activity_desc: Phase 154 execution started
progress:
  total_phases: 5
  completed_phases: 3
  total_plans: 18
  completed_plans: 15
  percent: 60
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-25 — v4.1 milestone started)

**Core value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.
**Current focus:** Phase 154 — desktop-chat-ui

## Current Position

Phase: 154 (desktop-chat-ui) — EXECUTING
Plan: 4 of 6
Status: Ready to execute
Last activity: 2026-06-26 — Phase 154 execution started

```
Progress: [██████████░░░░░░░░░░] 50% — 3/6 phases complete
```

## Operator Next Steps (carry-forward from v4.0)

**Pre-next-release operator follow-ups (no coding, required before next tagged release):**

1. **`RELEASE_PUBLISH_TOKEN`** (one-time): create fine-grained PAT scoped to `Contents: read/write` on `scottkw/agenthub`, then `gh secret set RELEASE_PUBLISH_TOKEN`. Without this, `release.published` will not auto-trigger `distribute.yml`.
2. **`WINGET_FIRST_SUBMISSION=true`** (one-time, first WinGet submission): `gh variable set WINGET_FIRST_SUBMISSION --body "true"`. Set before triggering `distribute.yml` for INSTALL-03; unset after `microsoft/winget-pkgs` accepts the first submission PR.
3. **Branch protection on `main`**: applied 2026-06-22 — 5 required checks (4 build matrix + `playwright`; `strict:false`, `enforce_admins:false`). Still valid for v4.1 CI.

## v4.1 Phase Plan

| Phase | Name | Requirements | Status |
|-------|------|--------------|--------|
| 151 | Message Schema + ChatStore | PERSIST-01, PERSIST-02, PERSIST-03 | Not started |
| 152 | Relay Protocol + Identity + Presence | IDENT-01, IDENT-02, PRESENCE-01, PRESENCE-02 | Not started |
| 153 | @session PTY Bridge | MENTION-02, MENTION-03, SEC-01, SEC-02 | Not started |
| 154 | Desktop Chat UI | CHAT-01, CHAT-02, CHAT-03, CHAT-04, MENTION-01, NOTIF-01, NOTIF-02, SEC-03 | Not started |
| 155 | Web-Share Chat UI + Cross-Surface Parity Gate | EXPORT-01, PARITY-01 | Not started |
| 156 | Install Links & Distribution | INSTALL-01, INSTALL-02, INSTALL-03 | Not started |
| 157 | Terminal Screen-Share Semantics (Issue #109) | VIEW-01, VIEW-02, VIEW-03, VIEW-04, VIEW-05 | Not started |

**Total:** 29 requirements mapped across 7 phases (100% coverage).

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

Last session: 2026-06-26T19:03:03.888Z
Stopped at: Phase 154 UI-SPEC approved
Resume file: .planning/phases/154-desktop-chat-ui/154-UI-SPEC.md
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
