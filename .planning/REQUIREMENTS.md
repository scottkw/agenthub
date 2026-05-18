# Milestone v3.3.1 Requirements — Bug Sweep

**Defined:** 2026-05-18
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

**Milestone Goal:** Close all open GitHub bug-labeled issues against v3.3 baseline as a patch release (v3.3.1). Bugs only — no enhancements, no advisory tech-debt drains, no process-debt retroactive fills. Cross-surface (GUI/TUI/CLI) parity is a release-blocking contract per v3.3 Phase 108.

**Closes GitHub Issues:** #52, #54, #55, #56, #57, #58.

**Carry-forward from v3.3 (operator one-time, before release):**

- `RELEASE_PUBLISH_TOKEN` PAT (`Contents: read/write` on `scottkw/agenthub`) — `gh secret set RELEASE_PUBLISH_TOKEN`
- `WINGET_FIRST_SUBMISSION=true` (one-time, first submission only) — `gh variable set WINGET_FIRST_SUBMISSION --body "true"`

---

## v3.3.1 Requirements

### IPC — Windows daemon named-pipe IPC (Issue #52, third-party PR #53)

Third-party report by `im-alexandre` with attached PR #53 based on v3.2 (commit `032a6e9`, 140 commits behind v3.3 tip). PR is small (7 files, +214/-13): adds `ipc_{windows,nonwindows}.go` abstraction, threads through `api.go` + `client.go` + `tray_windows.go`. Must be rebased / re-applied against v3.3 with author attribution preserved (`Co-Authored-By: im-alexandre <…>`). Five v3.3 commits since PR base touch `internal/daemon/api.go` and `client.go` (handleListShells, handleUpdateShellPath, ShellWebShareWarned) — likely merge conflicts in those two files; the two new `ipc_*.go` files drop in clean.

- [x] **IPC-01** — On Windows, the daemon listens on `\\.\pipe\agenthub-daemon` using `winio.ListenPipe` (not `net.Listen("unix", path)`); on macOS/Linux it continues to listen on a Unix socket. Verified by: `agenthub.exe daemon run` succeeds without `bind: A socket operation encountered a dead network` on Windows 11.
- [x] **IPC-02** — On Windows, `DaemonClient` dials `\\.\pipe\agenthub-daemon` using `winio.DialPipeContext`; CLI subcommands (`list`, `new`, `daemon status`, `tui`) connect to the running daemon without `EnsureDaemon` timeout.
- [x] **IPC-03** — `API.Stop()` does NOT attempt filesystem removal on named-pipe paths; existing `CleanupStaleSocket` named-pipe probing remains functional.
- [x] **IPC-04** — Windows regression tests exercise `API.Start` + `DaemonClient.Health()` over a named pipe end-to-end (not just `CleanupStaleSocket`), and `API.Stop` named-pipe path.
- [ ] **IPC-05** — All three surfaces (GUI / CLI / TUI) tested working on Windows 11 — daemon auto-start, session create, session list, attach/detach, web-share toggle.
- [x] **IPC-06** — PR #53 author (`im-alexandre`) credited via `Co-Authored-By` trailer on the merged/cherry-picked commits, or via dedicated commit message attribution if re-applied from scratch.

### WEB — Web-served terminal correctness (Issue #54)

Web surface only — desktop unaffected. Web frontend's session bridge does not consume OSC color-query / Device Attributes responses, leaking them into shell stdin. Pre-existing (predates v3.3); blocking parity for any sixel-using or capability-probing program (chafa, vim, neovim, mc) on the web surface.

- [ ] **WEB-01** — On the web-served terminal, OSC 10 (FG color query), OSC 11 (BG color query), and Device Attributes (`CSI c`) responses are consumed by the requesting program and do NOT appear in shell stdin. Reproducible with `chafa --format=sixel /tmp/<png>` in a web-shared shell session.
- [ ] **WEB-02** — Web ↔ desktop parity holds for chafa sixel rendering — the same `chafa --format=sixel` produces clean prompts on both surfaces (no leaked `10;rgb:…`, `11;rgb:…`, `62;4;9;22c` after image render).
- [ ] **WEB-03** — Regression test (Go or e2e) covers OSC response consumption on the web bridge; future regressions in the response path fail in CI.

### UI — Frontend bugs (Issues #55, #56)

- [ ] **UI-01** (#55) — When the terminal's WebGL context is lost (`WEBGL_lose_context.loseContext()` or natural loss), `WebGLRecoveryBanner` renders inside `.banner-stack` and auto-dismisses after 8s, per Phase 93 contract. Verified via DevTools console: `document.querySelector('.webgl-recovery-banner')` is non-null after context-loss event.
- [ ] **UI-02** (#55) — DOM fallback continues to work after WebGL loss (terminal content remains readable) — fallback path is not regressed by the banner-rendering fix.
- [ ] **UI-03** (#56) — On iPad Safari and iPad Chrome, single-finger drag on the terminal area scrolls xterm scrollback (matching desktop wheel-scroll behavior). Two-finger drag does not pan the viewport when started on the terminal area.
- [ ] **UI-04** (#56) — iPad touch-scroll does not regress mouse-wheel scrolling on desktop browsers, and does not break the existing iPad tap-on-link cluster (UAT-04 carry-over, separate from this milestone but must not be regressed).

### PTY — Linux PTY natural-exit detection (Issue #57)

v3.3 regression candidate — SHELL-12 auto-close was UAT-verified on macOS only per `107-VERIFICATION.md`; Linux PTY EOF semantics differ (`pty.Read()` blocks indefinitely after clean child exit on Linux amd64 with go-pty v0.2.2 + v0.2.3). The `TestListSessions_OnExitCallback_ReceivesNormalized` test was added in Phase 107-02 and silently skipped on Linux via `t.Skip()` pending this fix.

- [ ] **PTY-01** — On Linux, a shell session whose child process exits cleanly (`exit 0` at the prompt) causes the GUI tab / TUI list entry to auto-close — same behavior as macOS. Verified manually on a Linux desktop build and via CI test.
- [ ] **PTY-02** — Natural-exit detection is platform-aware: on Linux, a separate exit-detector goroutine polls `syscall.Wait4(pid, &status, WNOHANG)` (or equivalent) and explicitly closes the PTY to unblock the read loop, coordinating with go-pty's `waitOnContext` to avoid double-`Wait` race.
- [ ] **PTY-03** — `TestListSessions_OnExitCallback_ReceivesNormalized` no longer needs `t.Skip()` on Linux — runs and passes deterministically in `linux/amd64` CI under `-race -shuffle=on`.
- [ ] **PTY-04** — Cross-surface parity holds: TUI Linux and CLI Linux benefit from the same daemon-side fix (TUI uses the same daemon path; CLI attach-detach is independent but daemon-side cleanup is fixed). No regression on macOS or Windows.

### TEST — Test-suite stability (Issue #58)

- [ ] **TEST-01** — `internal/webserver/plugin_config_stream_test.go::TestPluginConfigStream_ExpiredCap_Returns401` passes deterministically (100/100 runs) on Linux CI under `-race -shuffle=on`, returning 401 (not 403). Root cause documented in the fix commit.
- [ ] **TEST-02** — Underlying cause investigated and stated in writing — not a rerun-pass hack. Likely candidates per issue triage: (a) shared-state pollution across tests in `internal/webserver` test setup (testServer / EnableSession / SetSigningKey leaks under specific orderings), (b) base64 strict-decode variance, (c) HMAC implementation. Fix addresses the root, not the symptom.

---

## Future Requirements (deferred, not v3.3.1)

These open enhancement issues remain backlog for v3.4 or later:

- `#51` Settings flag to enable/disable shell-session sharing warning
- `#50` Local agent running Gemma 3 1b
- `#49` Split-window functionality
- `#47` Admin control to restrict functionality
- `#42` Let's Encrypt certs when not using Tailscale
- `#30` iOS/Android mobile app versions
- `#24` File browser tab with remote capability
- `#10` Intersession communication / orchestration
- `#9` Networking enhancement

Internal v3.3 carry-forward (also deferred to v3.4):

- Phase 101 visual-fidelity UAT cosmetic items (5)
- Phase 101 advisory WR-01..09 + IN-01..06 (15 items)
- Phase 107 IN-01/02/03 + Browse-button aria-label + SettingsSearch `SEARCH_INDEX` missing "Shell binary"
- Phase 108 WR-01/WR-02 + IN-01..04 (docs/dead-code)
- Phase 103 process debt — `103-SUMMARY.md` + `103-IIP-DECISION.md` + `103-VERIFICATION.md`
- Nyquist `*-VALIDATION.md` missing for Phases 101–108
- `TestOpenCodeANSICapture` data race (pre-existing, skipped)
- `TestShellWebShareWarned_Default`-family failures (3 internal/daemon tests; Phase 108 SPEC §Out-of-scope)
- Phase 108 PARITY-CLI-03 harness limitation (test skip with `SetShellPathForTest` follow-up sketched)

---

## Out of Scope (v3.3.1)

- **Any new feature work** — patch release semantics enforced
- **Refactors not required by a bug fix** — Chesterton's Fence applies; surgical changes only
- **Process-debt fills (Phase 103, Nyquist)** — separate v3.4 deliverable
- **Cosmetic / advisory tech-debt drains** — deferred per scope discipline
- **Mobile native app** — desktop + web only (general project Out of Scope, reaffirmed)
- **Plugin system for adding new CLIs** — general project Out of Scope, reaffirmed

---

## Traceability

(Populated by roadmapper — REQ-ID → Phase mapping below)

| REQ-ID  | Phase | Status |
|---------|-------|--------|
| IPC-01  | Phase 109 | Complete |
| IPC-02  | Phase 109 | Complete |
| IPC-03  | Phase 109 | Complete |
| IPC-04  | Phase 109 | Complete |
| IPC-05  | Phase 109 | pending |
| IPC-06  | Phase 109 | Complete |
| WEB-01  | Phase 111 | pending |
| WEB-02  | Phase 111 | pending |
| WEB-03  | Phase 111 | pending |
| UI-01   | Phase 112 | pending |
| UI-02   | Phase 112 | pending |
| UI-03   | Phase 113 | pending |
| UI-04   | Phase 113 | pending |
| PTY-01  | Phase 110 | pending |
| PTY-02  | Phase 110 | pending |
| PTY-03  | Phase 110 | pending |
| PTY-04  | Phase 110 | pending |
| TEST-01 | Phase 114 | pending |
| TEST-02 | Phase 114 | pending |
