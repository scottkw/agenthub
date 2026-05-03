---
created: 2026-05-02
updated: 2026-05-03
session_topic: v3.1 milestone UAT continuation
status: paused_blocker_b1_native_pane_blank
last_commit: 6ad9bef
---

# Handoff — v3.1 UAT Continuation (round 2)

## What was completed this session

### Pre-flight resolved (Phase 90 Plan 06 steps 1-5 PASS)

All three blockers from the previous handoff cleared:

1. **Working-tree drift** → resolved. `go mod tidy` was non-idempotent against the bloated HEAD; tidied state committed in `f163ef5`. Direct dep anchors (nfpm/v2 v2.33.1, wails/v2 v2.12.0) preserved; only unreachable indirect deps removed.

2. **121 unpushed commits / build.yml red on origin** → resolved. Pushed to `release/v3.1` first as a green-gate (branch-first strategy). Build.yml on origin had been broken since 4-19 due to runner-image drift (windows-latest PowerShell parsing, ubuntu-latest webkit-4.1) + Phase 87 source-grep test rot. Required 4 fix commits before build.yml went green:
   - `a59d7fc` — delete stale Phase-61 source-grep test file (`App.serve02.test.tsx`)
   - `144b320` — Windows skip guards on capability/daemon Go tests (Unix-only assertions)
   - `1fae495` — update App.test/SettingsTab.test source-grep assertions for Phase 87 refactor
   - `91227fb` — `shell: bash` on tool-install steps for Windows compatibility (build.yml + 4 in release.yml)

   Once green, fast-forwarded `main` to `91227fb` (127 commits). Pre-flight UAT 1-5 PASS recorded in `90-06-HUMAN-UAT.md` (`abe4513`).

3. **Release env had zero protection rules** → resolved. Added required-reviewer rule (scottkw, prevent_self_review=false) via `gh api -X PUT`. sign-macos now pauses for approval before notarization.

### Release pipeline E2E validated through 3 rc cycles

- **rc1** (`abe4513`) — failed at sign-macos step "Verify internal attestation". Root cause: artifact-zip nesting bug. attest-build-provenance's bundle-path is absolute; mixing absolute + relative paths in upload-artifact's `path:` lines made the LCA `/`, so the downloaded artifact landed nested instead of flat. Fixed in `706c74f` (stage attestation bundle alongside artifact in build/bin/, then upload that dir).
- **rc2** (`706c74f`) — verify gate passed, untar passed, then failed at codesign with `build/entitlements.plist: cannot read entitlement data`. Root cause: Plan 04 split build-macos → build-macos + sign-macos but didn't add a Checkout step to sign-macos, so source files were absent. Fixed in `2729ba6` (added actions/checkout to sign-macos).
- **rc3** (`2729ba6`) — **all 6 jobs green**. Draft release `v3.1.0-rc3` exists with all 6 expected assets + 1 leaked extra (`AgentHub.app.tar.gz` — see P-7 in v3.1-RELEASE-BLOCKERS.md). Glob fix landed in `6ad9bef` for next tag.

### Phase 88 UAT signed off

Both SC-2 manual-only items PASS against the v3.1.0-rc3 signed dmg:

- **Item 1 — local-HTTPS-fallback**: Safari opened share link at `https://192.168.1.186:7443/sessions/<id>?cap=...`, accepted self-signed cert, terminal page rendered with live PTY echo. WS upgrade verified: Origin `https://192.168.1.186:7443`, response 101 Switching Protocols.
- **Item 2 — tailscale-mode**: iPhone scanned QR, accessed `https://kens-personal-macbook-air.tail46d69a.ts.net:7443/sessions/<id>?cap=...`, no cert warning (Let's Encrypt via Tailscale), bidirectional input verified. WS upgrade Origin header confirmed via Web Inspector.

`88-VERIFICATION.md` flipped `human_needed` → `passed` (4/4 must-haves verified). Phase 88 is done.

### Phase 90 Plan 06 — partial completion

Steps 1-9 UAT recorded. Steps 10-15 (external attestation verify, distribute.yml, install testing, sign-off) deferred until B-1 is fixed and a fresh rc is cut.

## Why we stopped — release blocker discovered

**B-1 (Critical): Native session pane renders blank in AgentHub GUI.** Sessions in the desktop window show empty terminal panes. The session is alive (web-served version renders fine), but the in-app xterm.js never paints. This was originally filed as a Phase 87 follow-up scoped to "LAN mode + web-enabled" but user observed it more broadly — needs reproduction matrix to pin down exact triggers.

Cannot ship v3.1.0 until B-1 is fixed. See `.planning/v3.1-RELEASE-BLOCKERS.md` for the full bug list (1 blocker + 7 polish items).

## State summary

| Phase | UAT status |
|---|---|
| 87 (capability auth) | ✅ Signed off 2026-04-21 |
| 88 (WebSocket Origin) | ✅ Signed off this session |
| 89 (CSP + vendored xterm) | ✅ Signed off this session (commit `6dd5c5b`) |
| 90 (release pipeline) | 🚧 Steps 1-9 PASS; 10-15 deferred until next rc |

| Pipeline state | Value |
|---|---|
| origin/main HEAD | `6ad9bef` (Phase 88 docs + glob fix) |
| Latest green build.yml on main | `91227fb` (also `6ad9bef` via the docs commit run, presumably) |
| Latest rc tag | `v3.1.0-rc3` (draft release exists) |
| release/v3.1 branch | merged to main; can be deleted, but leave for now as safety net |
| release env | required-reviewer rule active (scottkw) |
| Tailscale state | restored (tailscaled loaded + tailscale up) |
| AgentHub installed | rc3 dmg installed via Applications drag (brew cask uninstalled before install) |

## Files to read first on next session

- This handoff
- `.planning/v3.1-RELEASE-BLOCKERS.md` — bug list with severity classifications
- `.planning/STATE.md` — milestone progress
- `.planning/phases/90-release-pipeline-hardening/90-06-HUMAN-UAT.md` — release pipeline UAT progress (steps 10-15 still pending)

## Where to resume

1. **Reproduce B-1 (native pane blank).** Pin down the trigger matrix — does it affect all sessions? Only after web-enable? Only after Tailscale-mode switch? Only on rc3 dmg install vs. fresh-build? Worth running both `git checkout main && bash build.sh --platform macos` (fresh local build) AND the rc3 dmg side-by-side to see if local builds reproduce.

2. **Open Phase 91** for the native-pane fix (and decide which P-* items to bundle in). The Phase 87 plan dir is archived under `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/` so consult its plans 05-06 for the frontend refactor surface area.

3. **Add a real behavioral test** for native session render — source-grep tests only catch string presence, not actual mount + render. Vitest with React Testing Library or Playwright would catch this regression in CI next time.

4. After B-1 is fixed: tag a fresh rc, validate, then continue Phase 90 Plan 06 steps 10-15 (external attestation verify, distribute.yml dry-run via workflow_dispatch on the rc tag, install verification).

5. Once all green: tag `v3.1.0` (no `-rc` suffix → release.yml `draft: false` → ship). distribute.yml will fire on release published, updating the Homebrew tap and submitting the WinGet PR.

## Don't forget

- The rc3 draft release at https://github.com/scottkw/agenthub/releases is fine to leave drafted; we'll obsolete it with the next rc once B-1 is fixed.
- The `release-90-test` branch on `scottkw/homebrew-agenthub` (created during step 2 of Plan 06) is still there for distribute.yml dry-run. Don't delete until step 11 is verified.
- v3.0 brew cask was uninstalled to install rc3. The Homebrew tap won't auto-install rc3 (it's a draft); you're running the dmg-installed version.
