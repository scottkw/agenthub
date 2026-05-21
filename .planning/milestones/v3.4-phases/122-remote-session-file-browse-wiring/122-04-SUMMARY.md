---
phase: 122
plan: 04
subsystem: tui
tags: [tui, files, remote, cap, joincode, prompt, modal]
requires: ["122-01 daemon RemoteCapStore + ExchangeJoinCodeAtURL", "122-02 tui FilesClient interface"]
provides:
  - "RemoteFilesClient — HTTPS+cap implementation of FilesClient"
  - "joinCodePromptModel — Bubble Tea modal sub-model for cap acquisition"
  - "Model.remoteCapStore — in-memory cap cache keyed by sessionID"
  - "FilesOpen remote branch — fast path (cached cap) + slow path (prompt)"
  - "401 → cap-forget + re-prompt flow"
affects:
  - "internal/tui/files.go"
  - "internal/tui/files_cmds.go"
  - "internal/tui/update.go"
  - "internal/tui/view.go"
  - "internal/tui/modal.go"
  - "internal/tui/model.go"
tech-stack:
  added: []
  patterns:
    - "FilesClient interface decouples local vs remote transports — same Update loop, two clients"
    - "Bubble Tea sub-model with state machine (idle → submitting → error) for modal flow"
    - "CheckRedirect=ErrUseLastResponse to parse the 303 Location header instead of following"
key-files:
  created:
    - "internal/tui/files_client.go"
    - "internal/tui/remote_files_client.go"
    - "internal/tui/remote_files_client_test.go"
    - "internal/tui/joincode_prompt.go"
    - "internal/tui/joincode_prompt_test.go"
    - "internal/tui/update_remote_test.go"
  modified:
    - "internal/tui/files.go"
    - "internal/tui/files_cmds.go"
    - "internal/tui/files_test.go"
    - "internal/tui/update.go"
    - "internal/tui/view.go"
    - "internal/tui/modal.go"
    - "internal/tui/model.go"
decisions:
  - "Folded Plan 02 (FilesClient interface) into this plan because the dependency was not yet merged when execution started — see Deviations."
  - "Cap-token redaction uses (status, body) interpolation pattern instead of URL — redactCapFromURL helper is defense-in-depth only."
  - "exchangeJoinCodeCmd has its OWN HTTP client distinct from RemoteFilesClient (different timeouts + CheckRedirect requirement)."
  - "join-code typing clears any stale errMsg so the user can edit + retry without a confusing prior-attempt error."
metrics:
  duration: "single session"
  completed: "2026-05-20"
  tasks: 3
  commits: 4
---

# Phase 122 Plan 04: TUI RemoteFilesClient + joinCodePromptModel + FilesOpen Remote Branch Summary

Wired the TUI's `f`-keypress flow against remote tailnet sessions to fetch
files over Tailscale HTTPS with a session-scoped cap token, replacing the
v3.4 placeholder toast with a Bubble Tea join-code prompt modal that mirrors
the desktop GUI's paste-join-code flow. Closes REMOTE-03, REMOTE-04, and the
TUI half of REMOTE-05.

## One-Liner

TUI Files view now opens against remote tailnet sessions via HTTPS+cap with a
modal join-code prompt for cap acquisition; 401 forgets the cap and re-prompts.

## Tasks Completed

| Task | Name                                                                              | Commit  |
| ---- | --------------------------------------------------------------------------------- | ------- |
| 0    | FilesClient interface + filesModel.client wiring (folded-in Plan 02 prerequisite) | e819722 |
| 1    | RemoteFilesClient — HTTPS+cap implementation of FilesClient                       | d6ba740 |
| 2    | joinCodePromptModel + Model.remoteCapStore                                        | af09bf0 |
| 3    | FilesOpen remote branch + 401 cap-forget + remove v3.4 toast                      | c2c429e |

## What Was Built

**internal/tui/files_client.go** — `FilesClient` interface declaring the four
methods (`ListFiles`, `StatFile`, `ReadFile`, `HeadFile`) that the TUI's Cmd
factories dispatch against. `*daemon.DaemonClient` satisfies it via existing
methods (duck typing); `*RemoteFilesClient` satisfies it explicitly.

**internal/tui/remote_files_client.go** — HTTPS-and-cap mirror of
`*daemon.DaemonClient`'s files methods. TLS 1.2+ pinned, 15-second timeout,
URL construction via `net/url.Values.Encode()`, no cap-token interpolation
into error strings (T-122-04-01 CAP-LEAK invariant). `redactCapFromURL`
helper is provided as defense-in-depth for any future code path that needs to
interpolate a URL.

**internal/tui/joincode_prompt.go** — Bubble Tea sub-model
`joinCodePromptModel` capturing a 5-character join code via `textinput.Model`.
State machine: `idle → submitting → error`. `exchangeJoinCodeCmd` POSTs to the
remote `/join/exchange` endpoint with `CheckRedirect=ErrUseLastResponse` so
the 303 `Location: /sessions/<sid>?cap=<token>` header is observed and the
cap extracted. Four documented exchange-error kinds (`expired`, `invalid`,
`not-found`, `session-gone`) are translated by `friendlyJoinCodeError` into
user-facing copy.

**internal/tui/model.go** — Added `modalJoinCodePrompt` constant, the
`remoteCapEntry` helper type, `Model.remoteCapStore` (in-memory cap cache,
never persisted), and `Model.joinCodePrompt` sub-model field.

**internal/tui/update.go** — FilesOpen handler now branches on `entryLocal`
vs `entryRemote`. Cap-cached remote → fast path (open tabFiles immediately
against a `RemoteFilesClient`). Cap-absent remote → open
`modalJoinCodePrompt`. `handleKey` Priority 3.5 routes keypresses into the
prompt sub-model. `applyJoinCodeResultMsg` caches the cap, closes the modal,
opens tabFiles, dispatches `loadDirCmd` on success; on error transitions to
`joinCodePromptError` with the friendly message. `cancelJoinCodeMsg` clears
the modal.

**internal/tui/files.go** — `applyRemote401IfNeeded` shared helper detects a
401 from a `*RemoteFilesClient` and (a) deletes the cached cap so the next
`f` re-prompts and (b) replaces the status line with the verbatim D-04 copy
"Remote session must be web-shared to browse files. Ask the owner to enable
sharing." Wired into all three apply-message handlers (list/head/read) so a
401 from any of them triggers the same recovery.

**internal/tui/view.go + modal.go** — `renderJoinCodePromptModal` wraps the
prompt sub-model's `View()` in the standard centered bordered overlay frame
consistent with the kill-confirm and new-session modals.

## How It Works

```
Sessions view, cursor on a remote-session entry
                       │
                press 'f'
                       │
                       ▼
            ┌──────────────────────┐
            │ remoteCapStore[sid]? │
            └──────┬───────────┬───┘
                   │ YES       │ NO
                   ▼           ▼
       NewRemoteFilesClient  modalJoinCodePrompt opens
       opens tabFiles        ↓
       loadDirCmd            user types 5-char code, Enter
                             ↓
                             exchangeJoinCodeCmd POST /join/exchange
                             ↓ (303 Location)
                             joinCodeResultMsg{sessionID, cap, err}
                             ↓
            ┌──────────────────────┐
            │  err != nil ?         │
            └──────┬───────────┬───┘
                   │ NO        │ YES
                   ▼           ▼
        cache cap, open      joinCodePromptError +
        tabFiles, loadDir    friendlyJoinCodeError;
                             modal stays open for retry
```

If the loadDir (or head/read) round-trip later returns a 401:

```
filesListMsg{err: "remote files list: 401 unauthorized"}
                       │
                       ▼
            applyRemote401IfNeeded
                       │
        delete(remoteCapStore[sid])
        files.err = "Remote session must be web-shared…"
                       │
                       ▼
            (next `f` will re-prompt)
```

## Threat Mitigations

| ID          | Status | Implementation                                                                                         |
| ----------- | ------ | ------------------------------------------------------------------------------------------------------ |
| T-122-04-01 | done   | RemoteFilesClient error paths interpolate only (status, body) — cap never appears. Unit test verifies. |
| T-122-04-02 | done   | remoteCapStore lives only on Model; no os.WriteFile / ioutil.WriteFile in the new files (grep gate).   |
| T-122-04-03 | done   | TLS 1.2+ pinned on both RemoteFilesClient and exchangeJoinCodeCmd HTTP clients.                        |
| T-122-04-04 | accept | Join codes displayed in plain text in the textinput — parity with desktop GUI / Phase 87 threat model. |
| T-122-04-05 | done   | Modal captures sessionID at open time; joinCodeResultMsg echoes it; staleness guard rejects mismatch.  |
| T-122-04-06 | done   | RemoteFilesClient timeout=15s; per-request ctx timeouts (5s list/head, 10s read) in Cmd factories.     |
| T-122-04-07 | done   | 401 → delete cap → re-prompt on next `f`. Already mitigated upstream via Phase 119 capability middleware. |
| T-122-SC    | done   | Zero new Go packages introduced.                                                                       |

## Deviations from Plan

### Rule 3 — Auto-fixed blocking issue: Plan 02 dependency was not on main

**Found during:** Plan startup (load_plan step)
**Issue:** The execution prompt declared "Plans 01+02 already on main: use
daemon.RemoteCapStore + ExchangeJoinCodeAtURL + tui.FilesClient interface."
A grep + git-log check confirmed neither the daemon-side `RemoteCapStore` /
`ExchangeJoinCodeAtURL` nor the `internal/tui/FilesClient` interface existed
in the worktree's base (760b56b) or any other branch on the remote. Wave 1
appears to have been documented (the "Wave 1 progress update" commit
exists) but never landed as code.

**Fix:** Folded the Plan 02 prerequisite (FilesClient interface +
filesModel.client wiring + isNilFilesClient typed-nil guard) into this
plan's Task 0 (commit e819722). Plan 04 was scoped around the interface
being available; without it the FilesOpen handler could not produce a
RemoteFilesClient that the Cmd factories would accept.

The Plan 01 daemon-side `RemoteCapStore` / `ExchangeJoinCodeAtURL` were NOT
needed for this plan in the end — the TUI talks DIRECTLY to the remote
webserver's `/join/exchange` endpoint, with the cap cached on the Model
struct in-process. RESEARCH §Architectural Responsibility Map line "TUI
cross-network file fetch | API / Backend (Go HTTP client) | Standard
net/http over Tailscale TLS — no CORS, no browser involved" applies — the
daemon-side proxy that Plan 01 was scoped to deliver is a desktop-GUI
concern, not a TUI concern. The plan's `<interfaces>` block over-specified
its dependency on Plan 01.

**Files modified:** internal/tui/files_client.go (new),
internal/tui/files_cmds.go (signature changes),
internal/tui/files.go (filesModel.client field + helper methods)
**Commit:** e819722

### Rule 2 — Auto-added missing critical functionality: typed-nil interface guard

**Found during:** Task 0 (refactor files_cmds.go to take FilesClient)
**Issue:** The existing tests pass `nil` of the concrete `*daemon.DaemonClient`
type to assert the `errNilClient` sentinel propagates. After the refactor,
those tests pass an untyped `nil` literal that becomes the zero `FilesClient`
interface — `c == nil` would catch that. But a future caller wrapping a
typed-nil `(*RemoteFilesClient)(nil)` in the interface would silently get
past the guard and panic on the first method dispatch.

**Fix:** Added `isNilFilesClient` helper that handles both untyped-nil and
typed-nil cases for the two concrete implementations. Defense-in-depth that
covers the existing test invariant AND the new remote path.

**Files modified:** internal/tui/files_cmds.go
**Commit:** e819722 (rolled into Task 0)

### Rule 2 — Help-text update skipped per plan's own deliberation

The plan's Task 3 action block notes: "actually — `f` is already taken to
open the tab from Sessions; in tabFiles mode it doesn't currently do
anything. … Skip the help addition (delete this paragraph if the executor
agrees) — the `errMsg` in the status line tells the user what to do." We
agree and have not added a help-bar entry. The 401 → "Remote session must
be web-shared…" message is self-documenting; the user closes the tab and
presses `f` again from Sessions to re-prompt.

## Self-Check: PASSED

Verified:
- `internal/tui/files_client.go` — FOUND
- `internal/tui/remote_files_client.go` — FOUND
- `internal/tui/remote_files_client_test.go` — FOUND
- `internal/tui/joincode_prompt.go` — FOUND
- `internal/tui/joincode_prompt_test.go` — FOUND
- `internal/tui/update_remote_test.go` — FOUND
- Commits e819722, d6ba740, af09bf0, c2c429e — all reachable from HEAD
- `go test ./internal/tui/ ./internal/daemon/ ./internal/webserver/ ./internal/files/ -race` — PASS
- All 10 plan grep gates — match expected counts (modalJoinCodePrompt is 2 because
  the constant declaration is followed by an inline comment using the same
  identifier; the plan's "expect 1" was the declaration count)
- v3.4 toast — REMOVED (0 occurrences in update.go)

## Known Stubs

None.

## Threat Flags

None — no new attack surface beyond the threat register above.
