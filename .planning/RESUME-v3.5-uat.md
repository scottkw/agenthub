# RESUME — v3.5 UAT + Tech-Debt Session

**Paused:** 2026-06-14 night. **Resume:** next morning.
**Context:** Post-archive work on the **v3.5** milestone (File Browser — Write Ops & Editor).
Goal of session: do the deferred UATs (#24 two-machine + live GUI/TUI visual) and
resolve documented tech debt from `v3.5-MILESTONE-AUDIT.md`.

---

## DONE this session (9 commits on `main`, tree clean)

### Tech debt (all 3 audit items resolved)
- `0a9bbd9` — gofmt: `gofmt -l ./internal` now clean; Go suite green.
- `92aa919` — renamed gitignored `security-review/` → `.security-review/`; `go build ./...`
  and `go vet ./...` now pass (were exit 1). No artifacts deleted.
- `38ab6c3` — flipped Nyquist `nyquist_compliant: true` on phases 123/125/126/127
  (audit-only; agents re-ran scoped tests as evidence; VALIDATION.md maps filled).

### Live GUI/TUI visual UATs — Bucket A (COMPLETE, all operator-confirmed)
- **Phase 125 editor**: render ✓, paste ✓, dirty marker ✓. **Tab was broken** (moved focus) →
  fixed `7708efc` (added `indentWithTab`). Re-confirmed ✓.
- **Desktop local writes were 100% broken (404)** → fixed `64dad31` (relay loopback never
  mounted v3.5 write routes). Save/rename/upload/mkdir+delete all confirmed ✓ after fix.
- **CSS**: inline name input ~1 char wide + row-action icons overlapping next row →
  fixed `000b613`. Confirmed ✓.
- **Deselect** enhancement (click empty list space) → `e6cb2ba`. Confirmed ✓.
- **Phase 124 home-dir banner** never appeared → root cause `app.go ListSessions` dropped
  `HomeDir`/`FilesWrite` → fixed `b5cbba2`. Banner now renders ✓.
- **Phase 126 TUI $EDITOR suspend-resume**: confirmed ✓ (press Ctrl-\ to detach to Sessions
  list, highlight session WITHOUT Enter, press `f`, select file, `e` → nano → restores clean).

### Source-verified (colorblind contract) — authoritative, no eyes needed
Phase 124 banner + Phase 125 editor: every state = icon + text + ARIA; color is decoration.
Colorblind-safe at source. (See agent reports in session history.)

---

## RESOLVED 2026-06-15 — the big finding (commit `58af6d6`, on `main`, unpushed)

**FIXED.** `StartRelay` now wraps the relay server in a parent mux
(`wrapRelayWithRemoteFiles`, `internal/daemon/relay_remote_files.go`) that mounts
the 9 remote routes via the new `RelayHandler()` + relay CORS
(`relay.FilesCORS`/`relay.FilesPreflight`, newly exported) and falls through to
the relay server. Regression test `internal/daemon/relay_remote_files_test.go`
(routes mounted/200, CORS preflight 204+PUT+If-Match, no-cap reaches proxy not
route-miss). TDD: RED confirmed the bare-`404 page not found` route-miss first.
Full Go suite green; gofmt/vet/build clean. User corroborated remote browse
**never worked** before — confirms the finding. The real two-machine tailnet
UAT (#24) still needs two physical machines (see REMAINING).

<details><summary>Original finding (kept for the record)</summary>

**Remote file operations are broken on the desktop GUI — every `/api/files/remote/{sid}/...`
call (browse/read AND write/delete/rename/mkdir) 404s.**

- **Root cause:** The desktop GUI reaches files via the **relay loopback TCP server**
  (`internal/relay/server.go`, started in `internal/daemon/api.go:StartRelay` ~line 247,
  served on `relayPort`). Phase 120 added LOCAL read routes to BOTH the relay and the unix
  socket. **Phase 122 (remote read) and Phase 128 (remote write) added their routes ONLY to
  the unix-socket `a.mux` (`api.go:164-175`), never to the relay.** The webview can't reach
  the unix socket (api.go:242-245 comment), so remote routes are unreachable from the GUI.
- **Evidence:** live probe on the relay port → `GET /api/files/remote/{sid}/list` and
  `PUT .../write` both return **"404 page not found"** (Go ServeMux route-miss), while LOCAL
  routes return real data. `App.tsx:1270` sets remote `baseURL=http://127.0.0.1:${relayPort}`
  + `pathPrefix=/api/files/remote/{sid}`.
- **Scope:** Includes v3.4 Phase 122 remote *browse* — remote file access likely NEVER worked
  on the desktop GUI. Two-machine UAT #24 cannot pass until this is fixed.
- **Fix (well-understood, bigger than the others):** mount the 9 remote routes on the relay
  surface. Handlers exist as `(a *API) handleRemoteFiles*` in `internal/daemon/remote_files.go`
  (List/Stat/Read/Write/Upload/Delete/Rename/Mkdir), reachable from `StartRelay`. Cleanest:
  in `StartRelay`, wrap the relay server in a parent mux that registers the 9 remote routes
  (→ `a.handleRemoteFiles*`) and falls through (`parent.Handle("/", relayServer)`).
  **Must add CORS + OPTIONS preflight** for these (relay surface is cross-origin from the
  webview; local routes there use `withCORS`/`handleFilesPreflight`). Remote handlers
  currently emit NO CORS (they were unix-socket-only by design — see remote_files.go:7).
  Add a regression test mirroring `internal/relay/server_files_test.go::TestServer_FilesWriteAPI_MountedOnRelay`.

### Two pending decisions — ANSWERED 2026-06-15:
1. **Corroborate:** "No / never tried" → confirms the finding.
2. **Proceed:** "Fix it now" → done (commit `58af6d6`).

</details>

---

## LIVE UAT PROGRESS (2026-06-15, two-machine)
Client = this Mac (new build w/ fix, driven via `wails dev` + dev-browser at localhost:34115).
Host = `kens-macbook-air-1574` (v3.4.2), tailnet HTTPS :7443.
- **Relay fix PROVEN end-to-end live:** client relay → daemon proxy → v3.4.2 peer:
  LIST `200` (real listing), READ `a.txt` `200`, WRITE(PUT) `405` → v3.4 version-gate copy
  (`filesApi.ts:118`, source-verified). = v3.4.2 405-gate test (runbook 53–55) PASSED.
- Cap was deposited directly from the host's Full-Access capability URL (v3.4.2 shows the
  `?cap=` URL, not a 5-char code) via `POST /api/remote-files/caps`, then driven on the relay port.
- Two adjacent blockers found + filed: **#83** (client needs `accept-dns=true`; else opaque 502)
  and **#84** (discovery probe drops cap-protected peers — `/api/sessions` 401). Both noted in
  128-VERIFICATION.md prerequisites.
- **New-vs-new WRITE PROVEN:** host upgraded to v3.5; via relay, WRITE/MKDIR/RENAME/DELETE all
  `200`, read-back confirmed, dir restored. files.write cap present.
- **Remote browse had 4 stacked breakages** — see below.
- **accept-dns left ON.** Host web-share still ON. dev-browser at `~/.nvm/.../bin/dev-browser`
  (PATH not default). `wails dev` running (recompiled w/ #84; /tmp/wails-dev2.log).

## REMOTE-BROWSE BREAKAGE STACK (the full "why it never worked")
1. Relay routes not mounted → 404. FIXED `58af6d6` (proven live).
2. Client needs `accept-dns=true` or opaque 502. → **#83** (open).
3. Discovery probe accepted only HTTP 200; shared peers `/api/sessions`→401. FIXED `3508bd7`
   (**#84**) — daemon `/tailnet/peers` now lists the peer.
4. Remote panel can't LIST a peer's sessions: `FetchAllPeerSessions` calls cap-less
   `/api/sessions`, but it's intentionally cap-gated (D-18, returns only the capped session;
   server.go:456-458) and peers with 0 sessions are dropped (sessions.go:93-94). Architectural
   conflict → deferred v3.5.1 as **#86** (options a/b/c). Data path works; GUI on-ramp blocked.

Also: share-link scope relabel committed `e45ccba` (#24 UX).

## REMAINING work
- [x] ~~Relay gap~~ DONE `58af6d6` (proven). ~~v3.4.2 405 gate~~ PASSED. ~~new-vs-new write~~ PROVEN.
- [x] ~~#84 discovery~~ FIXED `3508bd7`. ~~scope relabel~~ `e45ccba`. ~~file #83/#84/#86~~ DONE.
- [ ] **Release scoping decision:** which commits land in v3.5 vs v3.5.1; remote-browse GUI is
  NOT shippable (blocked by #86 + #83) — treat as v3.5.1 / dedicated design pass.
- [ ] Update `v3.5-MILESTONE-AUDIT.md` (desktop remote path was broken in multiple layers).
- [ ] Cleanup when user confirms done: host Web Off, `accept-dns=false` (if desired), stop
  `wails dev`, optionally reship the rebuilt app.
- [ ] **Two-machine tailnet write UAT (#24)** — full runbook in
  `.planning/milestones/v3.5-phases/128-.../128-VERIFICATION.md`. Closes umbrella Issue #24.
- [ ] **Recommended:** file GitHub issues for the 5 UAT-discovered bugs (traceability;
  bugs cite "Discovered during Phase 12x UAT" per project convention) and **update
  `v3.5-MILESTONE-AUDIT.md`** — the desktop GUI write path was broken, which materially
  changes the milestone's "tech_debt/passed" status.
- [ ] Open child issues #62/#63/#64 may be closeable alongside #24.

## Environment notes
- Fresh prod GUI: `build/bin/agenthub.app` (rebuild: `wails build -tags wailsassets`, ~20-60s).
- Fresh CLI/TUI: `/tmp/agenthub-current` (rebuild: `go build -o /tmp/agenthub-current .`).
- Relaunch GUI: `osascript -e 'quit app "AgentHub"'; pkill -f "agenthub.app/Contents/MacOS/AgentHub"; open build/bin/agenthub.app`
  (NOTE: each GUI relaunch restarts the daemon → new session ids + new relayPort).
- Daemon socket: `~/Library/Application Support/agenthub/daemon.sock` (curl recipe used in session).
- `$EDITOR=nano`. Test dir `/tmp/v35-visual-uat` (recreate if gone). Issue #82 (TUI upload) is a
  sanctioned descope, NOT a bug.
- Only `.security-review/`, `bin/`, `build/`, `node_modules/` etc. are untracked — all fixes committed.
