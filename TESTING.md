# Testing — AgentHub Regression Suite

**Maintained by:** the team (update this file as part of every phase that adds or removes tests)
**Canonical home:** this file at the repo root (`TESTING.md`) supersedes the scattered per-phase UAT logs as the living document going forward. The per-phase logs (`137-HUMAN-UAT.md`, `138-HUMAN-UAT.md`, `141-HUMAN-UAT.md`, and per-phase `*-VERIFICATION.md` files) remain in place as historical record and are not deleted.

---

## 1. Overview

This document is the single source of truth for AgentHub's regression test program. It holds:

1. **Suite Manifest** — the four test groups, their file counts, run commands, and what each guards
2. **Merge Gate** — the exact `gh api` command to apply (and roll back) branch protection on `main`
3. **Requirement→Test Traceability Map** — v4.0-scoped, hand-maintained table mapping v4.0 requirement IDs to specific test files
4. **Manual Regression Checklist** — human-intervention behaviors that cannot be automated
5. **Standing Convention** — the per-phase rule every executor must follow

The traceability table is machine-validated by `tests/check-traceability-paths.sh`, which runs in CI as part of the `build (agenthub, linux/amd64, ubuntu-latest, ...)` job and exits 1 if any mapped path no longer exists on disk.

---

## 2. Suite Manifest

The entire CI suite IS the regression suite. No build tags, no relocated files — the suite is delineated here by manifest only.

| Group | Count | Location | Run Command | Guards |
|-------|-------|----------|-------------|--------|
| Go unit/integration | **365** `*_test.go` files | `internal/`, repo root | `go test -race -short ./...` | Daemon API, relay wire framing, capability model, PTY, webserver, files, status, tailnet |
| vitest (frontend) | **125** `*.test.ts/tsx` files | `frontend/src/` | `cd frontend && pnpm test` | React component render contracts, UI state, CSS token source gates, lib adapters (relay, remote, hub, status) |
| Playwright e2e | **7** `*.spec.ts` files | `frontend/e2e/` | `cd frontend && pnpm exec playwright test` | Web surface: file browser cap gate, file write/upload/delete, CSP, web-links toggle, plugin hot-swap, vendored xterm addons |
| build-script | **1** `build-script.test.sh` | `tests/` | `bash tests/build-script.test.sh` | Go build + Wails asset embedding |
| **Total** | **498** | — | — | — |

> Note: Phase 150 gap-closure #3 (live UAT): no file-count delta — EXTENDED `ShellWebShareBanner.test.tsx` (+2 variant tests) for the `variant="block"` modal layout fix; the banner reused in the Hub Share modal collapsed its text into a narrow column under the default horizontal banner-strip layout. Counts updated Phase 150 gap-closure #2 (live UAT, clean daemon): vitest +1 (`lib/shellCli.test.ts` — isShellCli() path-aware shell detection; the banner never fired because a shell session's cli is its full path `/bin/zsh`, but the gate matched a bare-name Set, and every prior test used bare names). `App.shellWebShare.test.tsx` + `SessionShareModal.test.tsx` updated to the shared isShellCli helper + a `/bin/zsh` path case. Known divergence (not fixed here): daemon `isShellSession` (engine.go:143) has the same bare-name limitation but gates SHELL-09 status/SHELL-11 spawn, not the SET-01 warning. Counts updated Phase 150 gap-closure (live UAT): vitest +1 (`RegenerateKeyModal.test.tsx` — 3 tests pinning default signing-key copy + parameterized copy overrides; the disable-warning reuse was rendering "Regenerate Signing Key? / Invalidate All Links" copy because the modal had hardcoded strings and the SettingsTab test mocked it). Phase 150-03: no new files — EXTENDED existing files only: `SessionShareModal.test.tsx` (+6 SET-01 shell-warning interception tests for D-09/D-10 cross-surface parity) and `App.shellWebShare.test.tsx` (+11 SET-01 warningEnabled gate + re-arm sync + HubPanel threading tests). Phase 150-02: vitest +1 (SettingsTab.shell-warn-toggle.test.tsx — 19 tests covering SET-01 toggle render/load, ON→confirm-dialog (D-07), confirm→SetShellWebShareWarningEnabled(false), cancel→noop, OFF→immediate true). Phase 150-01: Go +2 (engine_shell_warn_test.go + api_shell_warn_test.go — ShellWebShareWarningEnabled default/persist/rearm/off/API-GET/API-PATCH/client-roundtrip tests for SET-01). Phase 146: Go -1 (broadcast-only test deleted in Plan 02 — mintSessionJoinCodes wiring removed); vitest +3 (Phase 146 added `App.open-remote.test.tsx` + `__tests__/remoteAdapter.test.ts` in 146-00, reaching 112 live files). Live scan 2026-06-22 (347 Go, 112 vitest). Plan 05 (gap closure) Go +1 (open_remote_session_url_test.go — held-cap open-url read path). Phase 147-01: vitest +4 (HelpTab.test.tsx, HelpSearch.test.tsx, HelpSectionNav.test.tsx, HelpContent.test.tsx — RED stubs for Wave 0; turn GREEN in Plans 02/03). Phase 147-02: HelpContent + HelpSearch + HelpSectionNav + HelpTab implemented — 24 tests GREEN; 4 App.tsx + 7 Sidebar gates remain RED (Wave 3 wiring in Plan 03). Phase 147 code-review fix: vitest +1 (HelpTab.integration.test.tsx — real render-based integration test added for CR-01 dead-navigation fix; the prior source-gate-only HelpTab.test.tsx passed green against broken nav), reaching 117 files / 473 total. Phase 149 (AGENT-01): no file-count delta — EXTENDED existing files only: internal/pty/detect_test.go (agy detection tests), internal/daemon/path_windows_test.go (agy Windows PATH test), internal/status/detector_test.go (agy idle/waiting patterns), frontend/src/lib/agentBadge.test.ts (agy badge modifier), frontend/src/components/__tests__/style.hub.test.ts (agy three CSS color sites + WCAG comment). Phase 151 (PERSIST-01/02/03, v4.1 Session Chat): Go +4 (internal/relay/protocol_chat_test.go — ChatMessage schema round-trip + JSON marshal/unmarshal; internal/daemon/chat_test.go — ChatStore JSONL append + restart-survival load + concurrent -race + 10k cap reject at AppendMessage; internal/daemon/chat_routes_test.go — GET /api/chat/{id}/history+export relay loopback routes, restart-survival PERSIST-02; internal/webserver/chat_test.go — cap-gated web chat history/export, 401 on no/invalid cap, 403 on wrong-session cap); reaching 354 Go / 482 total. Phase 152-02 (IDENT-02, AliasStore): Go +1 (internal/daemon/alias_store_test.go — NewAliasStore first-run empty map, Get/Set/GetOrDefault round-trip, reload-persistence D-02, composite personKey isolation, invalid alias rejection T-152-01, 0600 file perms, fixed basename T-152-06); reaching 355 Go / 483 total. Phase 152-01 (relay protocol wire layer): Go +1 (internal/relay/protocol_presence_test.go — MakePresenceFrame/MakeTypingFrame/MakeAliasSetFrame round-trips, ValidateAlias accept/reject cases); reaching 356 Go / 484 total. Phase 152-03 (hub presence/typing layer): Go +1 (internal/relay/hub_presence_test.go — Subscribe/Unsubscribe ConnCount, UpdateTyping TTL auto-clear, UpdateAlias, CurrentPresence roster, Unsubscribe returns presenceChanged bool); reaching 357 Go / 485 total. Phase 152-05 (relay-path identity stamping): Go +1 (internal/relay/server_identity_test.go — identity stamped local:local on relay-path subscribe, WhoIs-fallback criterion-5 proof, MsgAliasSet propagates presence, MsgTyping outside ReadOnly gate); reaching 358 Go / 486 total. Phase 152-06 (web-share WhoIs identity + parity): Go +1 (internal/webserver/identity_test.go — TestWebIdentity_WhoIsFailureFallback non-local web personKey, TestWebAliasPropagation alias propagates via MsgPresence, TestWebReadOnlyCanChat D-06 RO alias-set); reaching 359 Go / 487 total. Phase 153-01 (PTY sanitizer): Go +1 (internal/relay/sanitize_test.go — SanitizePTYText strips C0/C1/CSI/OSC/bidi-overrides, output = printable text + single trailing \n, SEC-02). Phase 153-02 (core inject machinery): Go +1 (internal/relay/server_inject_test.go — TestInject_RWCap_WritesToPTY RW write + MsgChat broadcast MENTION-02; TestInject_OnlyDedicatedFrame MsgChatSend/stray never writes PTY MENTION-03/D-02; TestInject_ROCap_RelayPath RO client hand-crafted frame receives MsgInjectError NAK + zero PTY writes SEC-01 relay path). Phase 153-03 (web-share inject parity): Go +1 (internal/webserver/inject_test.go — TestInjectRO_WebPath RO-JWT web client hand-crafted MsgSessionInject frame receives MsgInjectError NAK + zero PTY writes, SEC-01 web path — distinct derivation from claims.Perms==\"read\" not URL param, Pitfall 5); reaching 362 Go / 490 total. Phase 154-01 (MsgChatSend dispatch): Go +3 (internal/relay/hub_chatsend_test.go — RO subscriber → ErrChatReadOnly no-persist/no-broadcast, empty-after-sanitize silent no-op, RW subscriber → chatAppendFn+BroadcastChat, HandleChatSend NEVER writes PTY; internal/relay/server_chatsend_test.go — relay read-pump RW broadcast, RO drop, malformed/empty silent ignore; internal/webserver/server_chatsend_test.go — webserver RW MsgChat broadcast, RO JWT silent drop, malformed ignore; SEC-01 web-path chat gate). Phase 154-03 (ChatMessage + ChatDaySeparator): vitest +2 (ChatMessage.test.tsx — 26 tests: safe Markdown render, rehype-sanitize strips script (SEC-03), inject-origin visual states (D-05/D-06), @mention text highlight (NOTIF-02), SessionInject=true badge; ChatDaySeparator.test.tsx — 12 tests: today/yesterday/date formatting, aria-label, no visible text duplicate). Phase 154-04 (ChatBadge + MentionPopover): vitest +2 (ChatBadge.test.tsx — count=0 null, count badge count text, hasMention=true shows @ glyph (D-10), singular/plural aria-label (NOTIF-01); MentionPopover.test.tsx — @session always shown, filter matches alias prefix, ArrowDown/Up active-index cycle, Enter selects, Escape closes, closes-on-click-outside (MENTION-01)). Phase 154-05 (ChatPanel subscription): vitest +1 (ChatPanel.test.tsx — WebSocket lifecycle, RelayClient callbacks, buildItems ordering + day-separator injection, unread accrual while closed (D-09), unread resets on open, SEC-03 script payload). Phase 154-06 (composer + overlay integration): no new files — EXTENDED ChatPanel.test.tsx (+5 tests: inject tap<600ms no-op, hold≥600ms fires inject (D-08), Enter+@session fires sendChat not inject (Pitfall 7), Enter sends+clears, Shift+Enter newline, @ opens popover, popover-Enter selects) and HubInteractiveModal.test.tsx (+7 tests: toggle exists/aria-label, ChatPanel always-mounted open=false (D-09), toggle→open=true, isActive unchanged by chatOpen (D-02), no column wrapper, onUnreadChange badge (NOTIF-01), mention glyph (D-10)) and SessionCard.test.tsx (+3 tests: NOTIF-01/D-10 ChatBadge props); reaching 365 Go / 498 total.

### CI Workflow Mapping

- `build.yml` (runs on push + PR): Go tests (all matrix platforms), vitest + build-script tests (ubuntu-latest only), and the traceability path-check step
- `e2e.yml` (runs on push/PR to `main`): Playwright cross-browser (chromium, firefox, webkit)

---

## 3. Merge Gate: How to Apply Branch Protection

Branch protection requires all five CI check contexts to pass before a PR can merge. Admin direct push to `main` is allowed (preserves the GSD `.planning/` doc-commit flow). No PR review requirement.

### Apply Protection

```bash
gh api repos/scottkw/agenthub/branches/main/protection \
  --method PUT \
  --header "Accept: application/vnd.github+json" \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": false,
    "checks": [
      {"context": "build (agenthub, linux/amd64, ubuntu-latest, webkit2_41, libwebkit2gtk-4.1-dev)", "app_id": 15368},
      {"context": "build (agenthub, linux/amd64, ubuntu-22.04, libwebkit2gtk-4.0-dev)", "app_id": 15368},
      {"context": "build (agenthub, darwin/universal, macos-latest)", "app_id": 15368},
      {"context": "build (agenthub, windows/amd64, windows-latest)", "app_id": 15368},
      {"context": "playwright", "app_id": 15368}
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null
}
EOF
```

**Key field rationale:**
- `checks[]` with `app_id` — current GitHub format (`contexts[]` is deprecated)
- `strict: false` — branches do not need to be up-to-date with `main` before merge
- `enforce_admins: false` — admin (scottkw) can push directly to `main` without CI; this preserves the GSD `.planning/` doc-commit flow
- `required_pull_request_reviews: null` — no PR review requirement
- `restrictions: null` — no user/team push restrictions
- GitHub Actions `app_id: 15368` — verified 2026-06-21 from live check-runs API

### Roll Back Protection

```bash
gh api repos/scottkw/agenthub/branches/main/protection --method DELETE
```

### Maintenance Note (Pitfall-1)

**If `build.yml` matrix entries change** (platform added, OS version bumped, label renamed), the status check context names change and the branch protection rule silently stops enforcing the renamed check. After any matrix change: roll back with DELETE, then re-apply with PUT using the new check context names from `gh api repos/scottkw/agenthub/commits/main/check-runs`.

**Warning sign:** A new PR passes CI but the "required checks" status badge shows an unknown check name.

---

## 4. Requirement→Test Traceability Map

Scope: v4.0 release-critical behaviors only (NAV/SHARE/CARD/TAB/RDS/POL/CARRY/TEST requirement IDs). Pre-v4.0 history is not traced here.

The path column must contain a repo-relative file path ending in `.go`, `.ts`, `.tsx`, or `.sh`. Test/describe names go in the Notes column. This format is required by `tests/check-traceability-paths.sh`.

| Requirement | Test File | Suite Group | Notes |
|-------------|-----------|-------------|-------|
| NAV-01 | internal/daemon/engine_test.go | Go | TUI removal: `go build ./...` passes post-Phase-136; Go test suite runs without TUI packages |
| NAV-02 | frontend/src/components/__tests__/App.nav.test.tsx | vitest | "NAV-02: Remote sidebar item is removed" — asserts no `onOpenRemoteSessions` prop |
| NAV-03 | frontend/src/components/__tests__/App.nav.test.tsx | vitest | "NAV-03: Sessions sidebar item is removed" — asserts no `onOpenDaemonManager` prop |
| NAV-04 | frontend/src/components/__tests__/App.nav.test.tsx | vitest | "NAV-04: Remote page is removed" — asserts absence of remote page routing |
| NAV-05 | frontend/src/components/__tests__/App.nav.test.tsx | vitest | "NAV-05: sidebar has no standalone New Session item" — asserts no `onAdd={handleAddTab}` prop |
| NAV-05 | frontend/src/components/__tests__/Sidebar.test.tsx | vitest | GAP-03: 3-item positive render contract with groupDefs present; `button.sidebar__item` count === 3 |
| SHARE-01 | frontend/src/components/__tests__/SessionShareModal.test.tsx | vitest | Share modal per-session render |
| SHARE-02 | frontend/src/components/__tests__/SessionShareModal.test.tsx | vitest | "Share the session" toggle reveals RO + RW rows |
| SHARE-03 | internal/daemon/api_test.go | Go | Browse-matrix test: RO code → read-only browse; RW code → read/write browse |
| SHARE-04 | frontend/src/components/__tests__/SessionShareModal.test.tsx | vitest | Copyable links, QR, LAN Basic Auth password surface |
| SHARE-05 | frontend/src/components/__tests__/SessionShareModal.test.tsx | vitest | Cap/URL/QR lifecycle (off→on cache-clear, stale-URL cleanup) |
| SHARE-06 | frontend/src/components/__tests__/SessionCard.share.test.tsx | vitest | Remote peer card: `isRemote` prop disables Share affordance (LockClosedIcon) |
| CARD-01 | frontend/src/components/__tests__/App.hub.test.tsx | vitest | Asserts `HUB_TAB` wiring; no `.hub__header` rendered |
| CARD-01 | frontend/src/components/__tests__/Sidebar.test.tsx | vitest | Asserts no `hub__header` inside Sidebar render |
| CARD-02 | frontend/src/components/__tests__/SessionCard.share.test.tsx | vitest | Local/remote origin indicator rendered per `isRemote` prop |
| CARD-03 | frontend/src/components/__tests__/SessionCard.share.test.tsx | vitest | Remote Available (GlobeAltIcon) vs Connected (LinkIcon) indicator |
| CARD-04 | frontend/src/lib/remoteAdapter.test.ts | vitest | `adaptRemoteSession` hostname/URL mapping; card layout carries all fields |
| CARD-05 | frontend/src/components/Hub/MiniPreview.test.tsx | vitest | StyledSpan headless VT render; correct column spacing |
| CARD-05 | internal/daemon/engine_test.go | Go | `TestGetSessionStyledTailLines_ColorBold` — daemon-side scrollback render |
| TAB-01 | frontend/src/components/__tests__/TabBar.test.tsx | vitest | Shrink floor: tabs do not collapse below minimum width |
| TAB-02 | frontend/src/components/__tests__/TabBar.test.tsx | vitest | Overflow: scroll chevron affordance renders when tabs overflow |
| TAB-03 | frontend/src/components/__tests__/TabBar.test.tsx | vitest | Tab close/rename/progress-underline functional at min width |
| TAB-04 | frontend/src/components/__tests__/TabBar.test.tsx | vitest | Session-tab chevron: present on session tabs, absent on no-sessionId tabs, click opens context menu, right-click path preserved (D-02), getBoundingClientRect rect-anchoring at source level |
| RDS-01 | frontend/src/components/__tests__/style.redesign.test.ts | vitest | Design direction documented; redesign token source gate |
| RDS-02 | frontend/src/components/__tests__/style.hub.test.ts | vitest | Hub-page redesign tokens: hex/var source gates |
| RDS-02 | frontend/src/components/__tests__/style.hub.modal.test.ts | vitest | Share modal redesign tokens |
| RDS-02 | frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx | vitest | Settings redesign: hex/var source gates |
| RDS-02 | frontend/src/__tests__/themeTokens.test.ts | vitest | Theme token palette source gates |
| RDS-04 | frontend/src/components/__tests__/style.redesign.test.ts | vitest | Colorblind-safe semantics: `prefers-reduced-motion` |
| POL-02 | frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx | vitest | Settings toggle: `role="switch"`, `aria-checked` |
| POL-03 | frontend/src/components/__tests__/style.hub.test.ts | vitest | "New session" button token source gate |
| POL-04 | frontend/src/components/__tests__/TerminalPanel.contextLoss.test.tsx | vitest | Terminal repaint after theme switch / tab switch |
| POL-05 | frontend/src/components/__tests__/Sidebar.test.tsx | vitest | Group sub-list ARIA structure; Hub group navigation in sidebar |
| CARRY-01 | frontend/src/components/__tests__/Sidebar.test.tsx | vitest | ARIA fix: `listbox`/`option` roles or roving-tabindex pattern |
| CARRY-02 | frontend/src/components/__tests__/style.hub.test.ts | vitest | Hub card layout tokens post-#93 triage |
| TEST-01 | tests/check-traceability-paths.sh | build-script | Path-check script: verifies every mapped path exists on disk |
| TEST-03 | frontend/src/lib/hubGroupCounts.test.ts | vitest | GAP-01: `computeCounts` / `computeGlobalCounts` — group count badges and "All" group totals |
| TEST-03 | frontend/src/lib/agentBadge.test.ts | vitest | GAP-02: `agentBadgeModifier` — session-type classification for color-coded origin spine/chip |
| TEST-03 | frontend/src/components/__tests__/Sidebar.test.tsx | vitest | GAP-03: 3-item positive render contract (see NAV-05 row above) |
| TEST-03 | frontend/src/components/__tests__/style.hub.test.ts | vitest | GAP-04: Phase 142 comp-fidelity CSS tokens (spine, chip, border-radius, preview height) |
| Cross-surface relay contract | frontend/src/lib/relayClient.test.ts | vitest | Wire framing: `encodeInputFrame`, `encodeResizeFrame`, `parseServerFrame` |
| Cross-surface relay contract | internal/relay/hub_test.go | Go | Relay hub-side wire protocol |
| Cross-surface relay contract | internal/relay/oscabsorb_relay_test.go | Go | OSC absorb filter in relay path |
| Hub group persistence | frontend/src/lib/hubGroups.test.ts | vitest | `loadGroups`/`saveGroups`/`createGroup`/`assignToGroup`/`removeFromGroup` round-trip |
| Hub status classification | frontend/src/lib/hubStatus.test.ts | vitest | `isAttentionStatus`: all six session states |
| FIX-01 | internal/daemon/engine_test.go | Go | Daemon styled-tail race fix (#100): `TestGetSessionStyledTailLines_*` — no data race, strip covers all query verbs + mode-2048 |
| FIX-02 | internal/files/write_test.go | Go | Windows concurrent-read fix (#101): `TestWriteFileAtomic_ConcurrentReadNeverPartial` — reader uses `readFilePlatformSafe` (FILE_SHARE_DELETE on Windows) so POSIX-semantics rename succeeds |
| FIX-02 | internal/files/concurrent_read_windows_test.go | Go | Windows build-tagged `readFilePlatformSafe` via `syscall.CreateFile` with FILE_SHARE_DELETE |
| FIX-02 | internal/files/concurrent_read_unix_test.go | Go | Non-Windows build-tagged `readFilePlatformSafe` delegating to `os.ReadFile` |
| FIX-03 | internal/webserver/sessions_meta_embed_test.go | Go | `TestSessionsMeta_NoJoinCodesInResponse` — RB-03 restored: ro_join_code/rw_join_code must NOT appear in /api/sessions/meta (cap-free discovery; broadcast wiring removed in Plan 02) |
| FIX-03 | internal/daemon/open_remote_session_url_test.go | Go | held-cap open-url read path — RemoteCapStore.Get → baseURL+/sessions/{id}?cap=TOKEN; absent → 404 (GAP-146-A reuse fix) |
| FIX-03 | frontend/src/components/__tests__/App.open-remote.test.tsx | vitest | Out-of-band open contract: `handleOpenRemoteSession` opens modal (not direct URL); `handleModalExchange` open-session branch builds `/sessions/{id}?cap=` URL + calls `BrowserOpenURL`; dead code (broadcast symbols) gone; behavior-level: "Open in browser" not disabled on remote card with no code (D-03). Plan 05 gap closure: held-cap reuse path (`remoteCapsCached.has` → `OpenRemoteSessionURL`; no modal on hit); WR-03 error-copy: not-found → "already used or expired" (not "Code invalid"). |
| HELP-01 | frontend/src/components/__tests__/HelpTab.test.tsx | vitest | Phase 147: HELP_TAB constant in App.tsx + --hub-search-highlight-bg CSS token source gates (RED in Plan 01; GREEN in Plans 02/03) |
| HELP-01 | frontend/src/components/__tests__/HelpSearch.test.tsx | vitest | Phase 147: search label, clear button, empty-state, and mark highlight assertions (RED in Plan 01; GREEN in Plan 03) |
| HELP-01 | frontend/src/components/__tests__/HelpSectionNav.test.tsx | vitest | Phase 147: section nav renders buttons per section; aria-current + active class + onSectionChange on click (RED in Plan 01; GREEN in Plan 03) |
| HELP-01 | frontend/src/components/__tests__/HelpContent.test.tsx | vitest | Phase 147: react-markdown import gate; BrowserOpenURL called on link click; no raw `<a href` in output (RED in Plan 01; GREEN in Plan 03) |
| HELP-01 | frontend/src/components/__tests__/HelpTab.integration.test.tsx | vitest | Phase 147 code-review fix (CR-01): real render-based test — section anchor ids (`#help-getting-started`/`#help-faq`) exist, nav click calls scrollIntoView + sets aria-current, search jump resolves a non-null section (fails against pre-fix concatenated render) |
| AGENT-01 | internal/pty/detect_test.go | Go | TestKnownCLIs has agy/"Google Antigravity"; TestDetectCLIs_FindsAgy finds agy stub on PATH; TestDetectCLI_AgyNotFound returns ErrCLINotFound when absent |
| AGENT-01 | internal/daemon/path_windows_test.go | Go | TestPlatformExtraBins_WindowsIncludesAgyBin: platformExtraBins includes %LOCALAPPDATA%\agy\bin (Windows matrix) |
| AGENT-01 | internal/status/detector_test.go | Go | TestDetector_AgyIdle (idle pattern `> `), TestDetector_AgyWaiting (`[y/n]`); PatternsForCLI("agy") not Fallback |
| AGENT-01 | frontend/src/lib/agentBadge.test.ts | vitest | agentBadgeModifier('agy') === 'agy' |
| AGENT-01 | frontend/src/components/__tests__/style.hub.test.ts | vitest | agy three CSS color sites all contain #ff9e64; WCAG comment present (dark 8.72:1 / light 2.03:1) |
| SET-01 | internal/daemon/engine_shell_warn_test.go | Go | TestShellWebShareWarningEnabled_Default (nil ptr → true), _Persists (Set(false)/reload, Set(true)/reload), _ReArm (D-03: Set(true) resets shellWebShareWarned), _OffBehavior (Set(false) does not reset warned) |
| SET-01 | internal/daemon/api_shell_warn_test.go | Go | TestAPIGetShellWebShareWarningEnabled_Default (GET → true), TestAPIPatchShellWebShareWarningEnabled_FlipsFalse (PATCH false → 204 + GET confirms), _BadBody (400), TestDaemonClient_GetSetShellWebShareWarningEnabled_RoundTrip |
| SET-01 | frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx | vitest | Source contract: imports, state quartet (shellWarnEnabled/Loaded/Saving/Error), showDisableWarnConfirm, handlers, exact label, optional onShellWarnEnabledChange prop. DOM: toggle renders after load; ON→confirm dialog (no RPC, D-07); confirm→SetShellWebShareWarningEnabled(false); cancel→noop; OFF→immediate SetShellWebShareWarningEnabled(true) with no dialog. |
| SET-01 | frontend/src/lib/shellCli.test.ts | vitest | Gap-closure #2 (live UAT): isShellCli() matches bare shell names AND full paths (`/bin/zsh`, `/usr/local/bin/bash`), Windows paths + `.exe` strip, case-insensitive basename; rejects non-shell CLIs (`claude`, `/usr/bin/claude`), out-of-set shells (`/bin/sh`, `fish`), and empty/undefined. Shared authority for both share surfaces (App StatusBar + Hub SessionShareModal). |
| SET-01 | frontend/src/components/__tests__/RegenerateKeyModal.test.tsx | vitest | Gap-closure (live UAT): default render keeps signing-key copy (Security surface regression guard); copy-override props (title/body/confirmLabel/cancelLabel) render correctly and do NOT leak "Regenerate Signing Key"/"Invalidate All Links" into the disable-warning reuse; confirm button invokes onConfirm regardless of label. |
| SET-01 | frontend/src/components/__tests__/SessionShareModal.test.tsx | vitest | Phase 150-03 D-09/D-10 cross-surface parity: shell+warningEnabled+!warned→banner+ToggleWebServing-blocked; warningEnabled=false→no banner+ToggleWebServing-called; non-shell→no banner+ToggleWebServing-called; already-warned→no banner+ToggleWebServing-called; cancel→onShellWebShareCancel+share-stays-OFF; confirm→onShellWebShareConfirm-called. |
| SET-01 | frontend/src/components/__tests__/App.shellWebShare.test.tsx | vitest | Phase 150-03: shellWebShareWarningEnabled state (default true), GetShellWebShareWarningEnabled hydration, gate includes warningEnabled&& before !shellWebShareWarned, useCallback dep array, onShellWarnEnabledChange→SettingsTab with re-arm GetShellWebShareWarned re-fetch, HubPanel receives shellWebShareWarned/Enabled + confirm/cancel props (D-09/D-10 single-authority threading). |
| PERSIST-01 | internal/daemon/chat_test.go | Go | Phase 151: ChatStore JSONL append + restart-survival load (pre-populated JSONL read back by fresh store) + concurrent AppendMessage -race + 10k cap reject (ErrChatCapReached at MaxChatMessages). |
| PERSIST-02 | internal/daemon/chat_routes_test.go | Go | Phase 151: GET /api/chat/{id}/history through RelayHandler returns full thread in order; restart-survival (engine2 with same chats dir returns same messages); webserver cap-gated path in internal/webserver/chat_test.go (401/403 gates). |
| PERSIST-03 | internal/daemon/engine_test.go | Go | Phase 151: KillSession calls store.Delete() — JSONL file absent after kill, ChatStoreFor ok==false (no orphan). 10k AppendMessage cap reject also covered in internal/daemon/chat_test.go. |
| IDENT-01 | internal/relay/server_identity_test.go | Go | Phase 152-05: relay-path identity stamping — TailnetID="local"/Origin="local"/PersonKey="local:local" on relay subscribe; WhoIs-fallback produces non-"local" personKey (criterion 5); NotifyPresence on join/leave; relay server builds and start cleanly with SetIdentityProviders nil. |
| IDENT-01 | internal/webserver/identity_test.go | Go | Phase 152-06: TestWebIdentity_WhoIsFailureFallback — WhoIs failure (no live tailnet) stamps Origin="web" and personKey ending ":web" that is NOT "local:local" (criterion 5 / D-04). |
| IDENT-02 | internal/daemon/alias_store_test.go | Go | Phase 152-02: AliasStore JSON persistence — NewAliasStore first-run empty map, Get/Set/GetOrDefault round-trip, reload-persistence (D-02 restart survival), composite personKey isolation (owner vs same-machine-browser), invalid alias rejection without persisting (T-152-01 defense in depth), aliases.json written at 0600, fixed basename (T-152-06 path traversal mitigation). |
| IDENT-02 | internal/webserver/identity_test.go | Go | Phase 152-06: TestWebAliasPropagation — MsgAliasSet over web path validates+persists+broadcasts a MsgPresence with the new alias; TestWebReadOnlyCanChat — RO-cap client can set alias (D-06: only MsgInput remains ReadOnly-gated). |
| PRESENCE-01 | internal/relay/hub_presence_test.go | Go | Phase 152-03: Subscribe increments ConnCount in presenceRoster; Unsubscribe decrements and returns presenceChanged=true on last connection drop; CurrentPresence roster snapshot. |
| PRESENCE-01 | internal/relay/server_identity_test.go | Go | Phase 152-05: NotifyPresence called on relay-path connect and disconnect; MsgPresence frame arrives on the client's channel. |
| PRESENCE-02 | internal/relay/hub_presence_test.go | Go | Phase 152-03: UpdateTyping sets typing=true and broadcasts MsgTyping; TTL timer auto-clears typing state and broadcasts typing=false; UpdateAlias persists new alias and updates presenceRoster. |
| MENTION-02 | internal/relay/server_inject_test.go | Go | Phase 153-02: TestInject_RWCap_WritesToPTY — RW client MsgSessionInject causes ptyWriteCount > 0 and MsgChat broadcast with SessionInject:true received (MENTION-02: @session inject writes PTY + broadcasts chat). |
| MENTION-03 | internal/relay/server_inject_test.go | Go | Phase 153-02: TestInject_OnlyDedicatedFrame — MsgChatSend (0x31) and stray frames leave ptyWriteCount == 0; only MsgSessionInject (0x35) can write PTY (MENTION-03/D-02: dedicated verb guard). |
| SEC-01 | internal/relay/server_inject_test.go | Go | Phase 153-02: TestInject_ROCap_RelayPath — RO client (?readonly=1) sends hand-crafted MsgSessionInject; receives MsgInjectError NAK within timeout AND ptyWriteCount == 0 (SEC-01 relay-path adversarial proof). |
| SEC-01 | internal/webserver/inject_test.go | Go | Phase 153-03: TestInjectRO_WebPath — RO-JWT web client (claims.Perms=="read") sends hand-crafted MsgSessionInject; receives relay.MsgInjectError NAK AND ptyWriteCount == 0 (SEC-01 web-path adversarial proof; Pitfall 5 cross-surface parity). |
| SEC-02 | internal/relay/sanitize_test.go | Go | Phase 153-01: SanitizePTYText strips C0 control bytes (except \n), C1 escapes, CSI/OSC sequences, bidi-override chars; output is printable text with a single trailing newline (SEC-02 injection-sanitizer correctness). |
| CHAT-01 | internal/relay/hub_chatsend_test.go | Go | Phase 154-01: Hub.HandleChatSend — RO subscriber returns ErrChatReadOnly with no persist/broadcast; empty-after-sanitize is silent no-op; RW subscriber calls chatAppendFn once + BroadcastChat once; HandleChatSend NEVER calls WriteInput (no PTY write, CHAT-01 server-side gate). |
| CHAT-01 | internal/relay/server_chatsend_test.go | Go | Phase 154-01: relay read-pump MsgChatSend dispatch — RW client MsgChatSend (0x31) causes MsgChat (0x30) broadcast; RO client frame silently dropped (no broadcast, no NAK); malformed/empty-content JSON silently ignored. |
| CHAT-01 | internal/webserver/server_chatsend_test.go | Go | Phase 154-01: webserver read-pump MsgChatSend dispatch — RW JWT client causes MsgChat broadcast; RO JWT client (claims.Perms=="read") silently dropped (SEC-01 web-path chat gate); malformed/empty JSON ignored. |
| CHAT-01 | frontend/src/components/Hub/ChatPanel.test.tsx | vitest | Phase 154-05: ChatPanel WebSocket lifecycle (subscribe/cleanup), RelayClient onChat callback, buildItems orders messages and injects day separators; Phase 154-06: Enter sends via sendChat + clears draft, @session present but hold<600ms fires nothing, @session Enter always routes to sendChat (Pitfall 7 guard). |
| CHAT-02 | frontend/src/components/Hub/ChatMessage.test.tsx | vitest | Phase 154-03: ChatMessage renders sender alias, content via react-markdown (no raw HTML), SessionInject=true shows "Injected" badge (D-05), "me" alias applies is-me CSS modifier, @mention text highlight (NOTIF-02). |
| CHAT-03 | frontend/src/components/Hub/ChatPanel.test.tsx | vitest | Phase 154-06: composer auto-grow — TextareaAutosize present, Enter/send path clears draft, Shift+Enter inserts newline without sending. |
| CHAT-04 | frontend/src/components/Hub/ChatDaySeparator.test.tsx | vitest | Phase 154-03: ChatDaySeparator renders "Today" / "Yesterday" / locale date string; aria-label equals displayed text; no duplicate visible content node (12 tests). |
| MENTION-01 | frontend/src/components/Hub/MentionPopover.test.tsx | vitest | Phase 154-04: @session always shown; alias prefix filter; ArrowDown/Up active-index cycle (wraps); Enter triggers onSelect with alias; Escape triggers onClose; click-outside triggers onClose; colorblind-safe — active item uses aria-selected not color alone. |
| MENTION-01 | frontend/src/components/Hub/ChatPanel.test.tsx | vitest | Phase 154-06: typing "@" opens MentionPopover; Enter in popover selects alias and inserts "@alias" into draft (MENTION-01 end-to-end wiring in composer). |
| NOTIF-01 | frontend/src/components/Hub/ChatBadge.test.tsx | vitest | Phase 154-04: count=0 renders null (no DOM node); count=3 hasMention=false shows "3" text; count=1 singular aria-label; count=3 hasMention=true shows "@" glyph (D-10 colorblind-safe shape signal) with mention aria-label. |
| NOTIF-01 | frontend/src/components/Hub/HubInteractiveModal.test.tsx | vitest | Phase 154-06: onUnreadChange(3, false) from ChatPanel → .chat-badge with text "3" appears on toggle; onUnreadChange(1, true) → .chat-badge--mention with "@" (D-10). |
| SEC-03 | frontend/src/components/Hub/ChatMessage.test.tsx | vitest | Phase 154-03: ChatMessage renders content via rehype-sanitize — no <script> element in DOM after passing script-tag content; no rehype-raw import in source (SEC-03 XSS gate). |
| SEC-03 | frontend/src/components/Hub/ChatPanel.test.tsx | vitest | Phase 154-05/06: onChat callback with <script>-containing payload — empty-state disappears (message received) and no <script> element in DOM (SEC-03 belt-and-suspenders over ChatMessage unit proof). |

---

## 5. Manual Regression Checklist

Human-intervention items that cannot be automated. Run before each tagged release.

### Category A — Share Modal (SHARE-01..06)

- **M-01** Share modal opens with RO + RW link rows in the live native webview. Home-dir write warning banner renders before the browse toggle when the session's workDir is the home directory.
  - _Why not automatable:_ Wails native webview is not accessible to Playwright or headless browser automation.
  - _Source:_ 137-HUMAN-UAT.md item 4

- **M-02** Remote peer card Share button is disabled with a lock icon and tooltip in the live native webview (user cannot re-share a session they do not own).
  - _Why not automatable:_ Same as M-01 — native webview only.
  - _Source:_ 137-HUMAN-UAT.md item 5

### Category B — Hub Navigation / Remote Peer (NAV, CARD)

- **M-03** Remote card "Open in browser" opens the `RemoteJoinCodeModal` (not a raw 401 page); after the owner shares and the viewer pastes the join code, the session opens in the browser at `baseURL/sessions/{id}?cap=TOKEN` (FIX-03 out-of-band flow, Phase 146).
  - _Why not automatable:_ Requires a live reachable Tailscale remote peer on a separate machine; cross-machine join-code exchange cannot be simulated in headless vitest.
  - _Source:_ 138-HUMAN-UAT.md item 3 (updated Phase 146 — out-of-band flow; needs office second machine)

- **M-04** Remote card overflow menu shows only "Open in browser" and "Browse files" — no Kill option on remote sessions.
  - _Why not automatable:_ Requires live reachable Tailscale peer.
  - _Source:_ 138-HUMAN-UAT.md item 5, remote half

- **M-05** Remote Connected chip (LinkIcon) and remote Available chip (GlobeAltIcon) render with icon + text on a live remote card.
  - _Why not automatable:_ Requires live reachable Tailscale peer.
  - _Source:_ 138-HUMAN-UAT.md item 6, remote half

- **M-06** Kill two-step confirm on a live local session: first click reveals "Confirm kill" button; second click terminates the session.
  - _Why not automatable:_ PTY session requires real daemon; PTY interaction has no TTY in the wails-dev `:34115` bridge.
  - _Source:_ 138-HUMAN-UAT.md (PASS recorded 2026-06-20; re-verify after Hub restructure)

### Category C — Terminal / Theme (POL-04)

- **M-07** Terminal repaints cleanly after a theme switch on an active session; after a tab switch away and back; CMD+/- font resize does not garble the output.
  - _Why not automatable:_ Wails native webview required for the GPU atlas paint path; Playwright cannot exercise the native rendering path.
  - _Source:_ 142-VERIFICATION.md (PASS recorded 2026-06-21; standing check for each release)

### Category D — Signed Build / Distribution

- **M-08** AirDrop'd signed macOS build passes Gatekeeper (`spctl --assess --verbose /Applications/AgentHub.app`) on a separate machine without quarantine prompts.
  - _Why not automatable:_ Requires code signing and two physical machines.
  - _Source:_ General release protocol; macOS signing cert shared across repos (re-exported 2026-04-08)

### Category E — File Browser / Editor (Deferred Live UATs)

- **M-09** Phase 125 editor on-screen render: CodeMirror file opens, Tab key indents, and Cmd-V pastes in the native WebView.
  - _Why not automatable:_ Wails native webview; CodeMirror keyboard events not reproducible in a headless browser.
  - _Source:_ STATE.md deferred items (pending live app)

- **M-10** Phase 126 `$EDITOR` shell-out: suspend-resume terminal restore works — the session terminal restores correctly after the spawned editor exits.
  - _Why not automatable:_ PTY interaction requires a real shell and editor launch.
  - _Source:_ STATE.md deferred items (pending live app)

- **M-11** Phase 124 home-dir warning banner on-screen in the live native WebView: the file write warning renders before the browse toggle when the session's workDir is the home directory.
  - _Why not automatable:_ Wails native webview required; the banner render path is not exercised by the headless browser.
  - _Source:_ STATE.md deferred items (pending live app)

### Category F — CI-Gated Go Portability (Windows)

- **M-12** FIX-02 (#101): `TestHandlerUpload_FilenameSanitized`, `TestDenylist_NonHomeRootedUnaffected`, and `TestWriteFileAtomic_ConcurrentReadNeverPartial` pass on Windows — verify the `build (agenthub, windows/amd64, windows-latest)` job is green in GitHub Actions after pushing (no local Windows env available).
  - _Why not automatable:_ Development is on macOS; Windows file-share semantics (FILE_SHARE_DELETE requirement for POSIX-semantics rename) cannot be observed locally. The CI Windows runner is the only ground truth.
  - _Source:_ Phase 145 windows-files-test-fixes; FIX-02 (#101)

### Category G — Live Tailnet Remote Session Open (FIX-03)

- **M-13** FIX-03 (#98, out-of-band flow + held-cap reuse, GAP-146-A): Two sub-scenarios on a live two-Mac tailnet:
  - **First open (no held cap):** "Open in browser" on a remote session card opens `RemoteJoinCodeModal` (not a raw 401). After the owner shares and delivers a join code out of band, the viewer pastes the code; the session opens in the browser at `baseURL/sessions/{id}?cap=TOKEN`. RO code → RO open; RW code → RW open. No "capability required" page.
  - **Second open (held-cap reuse, Plan 05):** Once the viewer has already connected in-app (which deposited the cap into RemoteCapStore), clicking "Open in browser" a SECOND time on the same session must open WITHOUT prompting for a fresh join code. The app reuses the held cap directly (the single-use code is already consumed — D-11). Steps: (1) On Mac A, start a session, enable Share, deliver join code to Mac B out of band. (2) On Mac B, connect in-app (cap deposited). (3) On Mac B, click "Open in browser" — expect it opens directly WITHOUT a modal. (4) Verify browser opens at `baseURL/sessions/{id}?cap=TOKEN`. No join-code prompt on the second open.
  - _Why not automatable:_ Requires two real Macs on the same tailnet; the `:34115` wails-dev bridge has no real tailnet peer; web-share WebSocket blocks automated terminal input (see live-UAT-daemon-gotchas memory).
  - _Source:_ 146-VALIDATION.md Manual-Only Verifications table; FIX-03 (#98) — Phase 146 out-of-band redesign (D-02/D-04); GAP-146-A Plan 05 held-cap reuse

### Category H — In-App Help Page (HELP-01)

- **M-14** Help page opens in the live native WebView via the sidebar Help button: search input, left section-nav ("Getting Started", "FAQ"), and Markdown content all render. Clicking an FAQ answer's GitHub Issues link opens the system browser (not in-app webview). Search for "DevTools" shows at least one highlighted result with a `<mark>` span.
  - _Why not automatable:_ Wails native webview required for the BrowserOpenURL external-browser path and IntersectionObserver scroll-spy active-section tracking.
  - _Source:_ Phase 147 HELP-01 (#69); add when Plans 02/03 complete

### Category I — Live Agent Launch (AGENT-01)

- **M-15** ✅ PASSED 2026-06-23 — Live Antigravity launch: ran the live UAT with `agy` (Antigravity CLI 1.0.10) in the native build and confirmed (a) the new-session picker shows "Google Antigravity"; (b) `agy` launches an interactive PTY REPL (bidirectional round-trip — typed prompt → live agent response); (c) auth complete (authenticated session, Antigravity Starter Quota); (d) `#ff9e64` badge source-validated (style.hub.test.ts) + tab dot amber in live build; (e) color lockstep source-validated. Owner is colorblind — color confirmed at source, not by eye.
  - _Why it was deferred:_ `agy` was in closed-beta/waitlist (D-03) — the binary could not be installed for live UAT during Phase 149. Resolved once the maintainer obtained access.
  - _Source:_ CONTEXT D-03; 149-VERIFICATION.md (M-15 resolved 2026-06-23); AGENT-01 (#65)

### Category J — Shell Web-Share Warning Toggle (SET-01)

- **M-16** Shell web-share warning banner fires on both share surfaces — live PTY web-share verification:
  1. Start a shell session (bash, zsh, or shell cli). Confirm `shellWebShareWarned` is not yet set (fresh install or cleared).
  2. **Hub Share modal path:** Click the Share button on the shell session card → Share modal opens → click "Share the session" toggle ON → confirm the ShellWebShareBanner appears ("Web sharing this shell will expose arbitrary command execution.") and ToggleWebServing has NOT yet fired (sharing is still OFF). Click "Enable web sharing" → banner dismisses, sharing enables, share links appear.
  3. **StatusBar path:** Open the shell session tab → click the StatusBar web-share toggle → confirm the same ShellWebShareBanner appears. Click Cancel → sharing stays OFF.
  4. **Disabled warning:** Go to Settings > Session Behavior → disable "Warn before web-sharing a shell session." → confirm the modal (D-07). Then repeat step 2 — banner must NOT appear; sharing enables immediately.
  5. **Re-enable:** Re-enable the warning in Settings → both share surfaces must show the banner again on the next unacknowledged shell toggle.
  - _Why not automatable:_ Live PTY session requires a real daemon; the `:34115` wails-dev browser bridge has no PTY (see reference_wails_dev_browser_pty_limit memory). Web-share WebSocket blocks automated input. Real shell session required to exercise the SHELL_CLIS gate.
  - _Source:_ 150-VALIDATION.md Manual-Only Verifications; SET-01 (#51)
  - ✅ **VERIFIED 2026-06-23 (live UAT, wails dev).** Both surfaces fire the banner; disable suppresses; re-enable re-arms. Surfaced 3 bugs (all fixed this session): wrong disable-confirm copy, `/bin/zsh` full-path gate miss (banner never fired for real shells), modal banner layout.

- **M-17** Shell web-share warning restart-persistence check:
  1. Open Settings > Session Behavior → disable "Warn before web-sharing a shell session." (confirm modal). Quit the app and restart.
  2. Open a shell session Share modal → confirm banner does NOT appear (warning suppressed persisted across restart via daemon settings.json).
  3. Re-enable the warning in Settings. Quit and restart.
  4. Open a shell session Share modal → confirm banner DOES appear again (re-arm persisted).
  - _Why not automatable:_ Requires full daemon restart + disk settings.json read-back, which cannot be simulated in headless vitest without a running daemon process.
  - _Source:_ 150-VALIDATION.md Manual-Only Verifications (restart-persistence); SET-01 (#51)
  - ✅ **VERIFIED 2026-06-23 (live UAT).** Both directions confirmed across genuine cold daemon restarts (daemon PID changed each cycle, proving fresh settings.json read — the daemon detaches and survives `wails dev` Ctrl+C, so the daemon process itself had to be killed to force a cold start). Disabled→no banner; re-enabled→banner fires.

### Category K — Relay/Web-Share Identity & Presence (IDENT-01, PRESENCE-01, PRESENCE-02)

- **M-18** Over a live tailnet with two real clients (desktop owner + tailnet peer's browser or a second machine's relay attach), the presence roster shows TWO DISTINCT entries — the owner stamped as `local:local` and the web/relay peer stamped as `<tailnetNodeKey>:web` (or `local:local` for a second relay client on the same machine if they happen to share a key, but distinct from the web entry). Neither entry silently aliases to the other.
  - _Why not automatable:_ Criterion 5 (owner vs same-machine browser disambiguation) requires a live tailnet with an actual WhoIs response. In automated tests `lc.WhoIs` always fails (no tailscaled running) so `tailnetID` stays `"unknown"`; the only proof of real WhoIs stamping is a live multi-client scenario.
  - _Status:_ ✅ VERIFIED 2026-06-25 — live `lc.WhoIs` against running tailscaled (the exact `webserver/server.go` identity path) resolved real, distinct node keys: this Mac `nodekey:456d9361…:web` (alias `kens-personal-macbook-air`) and peer kens-inspiron `nodekey:ec65d9ee…:web`, both distinct from owner `local:local`; loopback fell back to `unknown:web` (non-`local`). Hub roster distinctness covered by `TestCompositePersonKey`. No silent merge.
  - _Source:_ Phase 152 IDENT-01 (D-04 / criterion 5); 152-06-PLAN.md must_haves

- **M-19** Typing indicator timing in a live browser: when a web-share viewer or relay-attaching peer starts typing (sends `MsgTyping{typing:true}`), the typing indicator appears in any observer's presence view within ≤500ms. When the user stops typing (no further `MsgTyping{typing:true}` within 5 seconds), the indicator automatically clears within ≤500ms of the 5-second TTL expiry.
  - _Why not automatable:_ Wall-clock timing (5-second TTL auto-clear) cannot be validated in unit tests without injecting an accelerated clock; the unit tests use a 1ms TTL override that proves functional correctness but not the live 5-second UX contract. A real browser is needed to confirm the visual indicator renders and clears on the correct schedule.
  - _Status:_ Wire-level VERIFIED (Phase 152): TTL auto-clear, timer reset, explicit stop, sender exclusion, 500ms rate-limit (`hub.go:367`), 5s TTL (`hub.go:86`) all covered by `TestTyping*`. **Browser-visual observation DEFERRED to Phases 154/155** — the chat UI that renders the typing indicator does not exist in Phase 152 (152-04 parses frames only; ChatPanel ships in 154/155). Run the live-browser timing check as part of 154/155 UAT.
  - _Source:_ Phase 152 PRESENCE-02 (relay.Hub UpdateTyping TTL); 152-RESEARCH.md Validation Architecture lines 779-799; browser-visual → Phases 154/155

### Category L — Desktop Chat UI (Phase 154)

- **M-20** Inject fill animation renders in native WebView: open the Hub chat panel on a live session that has `@session` in the composer draft, then press-and-hold the Inject button. The button's `::before` pseudo-element scaleX fill must animate left→right over exactly 600ms, then the inject fires (PTY receives the text). Confirm no animation plays for a tap (pointer-up before 600ms). The `prefers-reduced-motion` guard may suppress the animation but the 600ms inject must still fire.
  - _Why not automatable:_ CSS pseudo-element animation (`::before { transition: transform 600ms linear }`) cannot be observed in jsdom (no paint), and the Wails native WebView is not accessible to Playwright. The press-and-hold timer fires in jsdom tests (D-08 verified), but the visual fill is visual-only.
  - _Source:_ Phase 154-06 D-08 / 154-UI-SPEC.md §7; RESEARCH Pitfall 7

- **M-21** Overlay no-resize proof on live PTY: open the Hub interactive modal on a running session, record the PTY column count (`tput cols` in the PTY), then click the chat toggle to open the overlay drawer. Confirm `tput cols` returns the same value after the drawer opens. No PTY resize must fire from toggling chat.
  - _Why not automatable:_ PTY column count requires a live PTY process; jsdom has no layout engine, so the terminal has no column width. The unit test (HubInteractiveModal.test.tsx) proves `isActive` is unchanged, but cannot measure the actual PTY dimensions.
  - _Source:_ Phase 154-06 D-02; the overlay must NOT call `client.sendResize` on chat toggle

- **M-22** Day separators scroll with message history: in a live chat session, send messages on different calendar days (or fast-forward system clock across midnight). Verify "Today", "Yesterday", and date-string separators appear correctly in the chat stream with no duplicate visible text and that they do not interfere with scroll behavior (the list stays scrolled to bottom on new messages).
  - _Why not automatable:_ ChatDaySeparator unit tests cover the text formatting, but jsdom reports zero container height for virtualized lists, so the `@tanstack/react-virtual` scroll-to-bottom path is not exercised. Calendar-day cross-boundary requires a live session with controlled time.
  - _Source:_ Phase 154-03 CHAT-04; ChatDaySeparator.test.tsx covers text; scroll behavior is live-only

- **M-23** Accidental-Enter guard on live PTY: with `@session` in the composer draft, press Enter (no modifier, no hold). Confirm the message is sent as a chat message (appears in the chat stream) and the PTY does NOT receive the text (no visible echo in the terminal). Then press-and-hold the Inject button for ≥600ms and confirm the PTY DOES receive the text.
  - _Why not automatable:_ The separation of chat-send vs. PTY-inject paths is unit-tested (Pitfall 7 in ChatPanel.test.tsx), but observing that no PTY echo appears requires a live running process where PTY output is visible in the terminal pane.
  - _Source:_ Phase 154-06 D-08 / RESEARCH Pitfall 7; Enter-never-injects invariant

---

## 6. Standing Convention

**Every future phase that adds, renames, or removes tests must:**

1. **Automated tests** — add the new test files to the appropriate suite group (Go, vitest, Playwright, or build-script). No build tags, no file moves — the group delineation lives in this manifest only.

2. **Manual checklist** — if the phase introduces a behavior that cannot be automated (native GUI interaction, remote peer, live PTY, physical hardware), add a new M-NN item to Section 5 of this file with: behavior description, why it cannot be automated, and source phase/UAT log.

3. **Traceability map** — add a row to Section 4 for every new test file that covers a v4.0 (or later milestone) requirement. The path column must be a repo-relative file path ending in `.go`, `.ts`, `.tsx`, or `.sh` — no test names, no describe-block suffixes in the path column. Test names go in the Notes column. Run `bash tests/check-traceability-paths.sh` locally before committing to confirm the script exits 0.

4. **Branch protection maintenance** — if `build.yml` matrix entries change (platform added, OS label changed), re-apply branch protection with the updated check context names (see Section 3).

This convention is also referenced in the repo-level `CLAUDE.md` so it surfaces to every Claude session.
