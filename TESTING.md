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
| Go unit/integration | **346** `*_test.go` files | `internal/`, repo root | `go test -race -short ./...` | Daemon API, relay wire framing, capability model, PTY, webserver, files, status, tailnet |
| vitest (frontend) | **108** `*.test.ts/tsx` files | `frontend/src/` | `cd frontend && pnpm test` | React component render contracts, UI state, CSS token source gates, lib adapters (relay, remote, hub, status) |
| Playwright e2e | **7** `*.spec.ts` files | `frontend/e2e/` | `cd frontend && pnpm exec playwright test` | Web surface: file browser cap gate, file write/upload/delete, CSP, web-links toggle, plugin hot-swap, vendored xterm addons |
| build-script | **1** `build-script.test.sh` | `tests/` | `bash tests/build-script.test.sh` | Go build + Wails asset embedding |
| **Total** | **462** | — | — | — |

> Note: CONTEXT.md references "459 test files" and "115 vitest files". The authoritative counts above come from a live filesystem scan on 2026-06-21. The correct vitest count is 108 (7 vitest files were removed during Phase 136 TUI deletion).

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

- **M-03** Remote card "Open in browser" shows a real peer URL (not a blank page); `BrowserOpenURL` forwards the URL correctly to the system browser.
  - _Why not automatable:_ Requires a live reachable Tailscale remote peer on a separate machine.
  - _Source:_ 138-HUMAN-UAT.md item 3 (postponed 2026-06-22 — needs office second machine)

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

---

## 6. Standing Convention

**Every future phase that adds, renames, or removes tests must:**

1. **Automated tests** — add the new test files to the appropriate suite group (Go, vitest, Playwright, or build-script). No build tags, no file moves — the group delineation lives in this manifest only.

2. **Manual checklist** — if the phase introduces a behavior that cannot be automated (native GUI interaction, remote peer, live PTY, physical hardware), add a new M-NN item to Section 5 of this file with: behavior description, why it cannot be automated, and source phase/UAT log.

3. **Traceability map** — add a row to Section 4 for every new test file that covers a v4.0 (or later milestone) requirement. The path column must be a repo-relative file path ending in `.go`, `.ts`, `.tsx`, or `.sh` — no test names, no describe-block suffixes in the path column. Test names go in the Notes column. Run `bash tests/check-traceability-paths.sh` locally before committing to confirm the script exits 0.

4. **Branch protection maintenance** — if `build.yml` matrix entries change (platform added, OS label changed), re-apply branch protection with the updated check context names (see Section 3).

This convention is also referenced in the repo-level `CLAUDE.md` so it surfaces to every Claude session.
