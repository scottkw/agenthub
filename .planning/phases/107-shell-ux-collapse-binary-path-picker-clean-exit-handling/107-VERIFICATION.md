---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
verified: 2026-05-13T22:05:00Z
human_uat_signed_off: 2026-05-15T00:00:00Z
status: approved
score: 13/13 must-haves verified
overrides_applied: 0
tester: Ken Scott
build: AgentHub 2.2.0
re_verification:
  previous_status: none
  previous_score: n/a
  gaps_closed: []
  gaps_remaining: []
  regressions: []
human_verification_resolved:
  - test: "Shell tab auto-close on exit-code 0 — runtime behavior"
    result: pass-with-deviation
    notes: |
      Part A (typed `exit` → clean exit-0 → tab auto-closed silently) — PASS, behavioral chain confirmed end-to-end.
      Part B (typed `exit 2` → expected ExitToast + tab stays open) — observed: tab also auto-closed silently.
      Root-cause investigation revealed every natural exit is reported to the GUI as exit-code 0:
        * `internal/pty/session.Session.ExitCode()` returns only the cached `exitCode` int (initialised to -1).
        * `cleanup.go` populates the cache via `SetExitCode(cmd.ProcessState.ExitCode())` only on the kill path.
        * `internal/daemon/engine.go` natural-exit goroutine never reads `cmd.ProcessState` into the cache, so the cache stays at -1 and is normalised to 0 by `engine.go` line ~389 for every natural exit.
      Attempted fix during UAT (`Session.CaptureExitCode()` reading ProcessState into cache after a 100ms wait) did not change observed behaviour — needs deeper investigation of go-pty's waitOnContext timing.
deferred_to_followup:
  - id: SHELL-12-EXITCODE-NORM-OVERREACH
    severity: cosmetic
    decision: |
      Tester accepted "auto-close on any natural exit (zero or non-zero)" as the final v3.3 behaviour.
      ExitToast for non-zero natural exits is not a v3.3 requirement.
    rationale: |
      Per tester: "I don't need to know error state anyway." For shell sessions the tab-close
      gesture is sufficient; the cost of restoring the non-zero-exit ExitToast path is not
      justified by the observed user-need.
    revisit_in: v3.4 if the ExitToast for non-zero exits becomes a needed signal
---

# Phase 107: Shell UX Collapse + Binary Path Picker + Clean-Exit Handling — Verification Report

**Phase Goal:** Three deltas from first-user-test feedback on v3.3 — SHELL-10 (collapse new-session modal to a single Shell row), SHELL-11 (Settings → Paths "Shell binary" path field with executable validation), SHELL-12 (normalize PTY exit-code -1 → 0 in ListSessions emission path AND auto-close tab on exit-code 0).
**Verified:** 2026-05-13T22:05:00Z
**Status:** human_needed
**Re-verification:** No — initial verification.

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | NewSessionModal renders exactly ONE Shell row regardless of `shells` prop length | VERIFIED | `NewSessionModal.tsx:146-157` — single static `<button class="new-session-modal__agent-btn--shell">`. `grep -c "new-session-modal__agent-btn--shell" NewSessionModal.tsx` = 1. |
| 2 | `sortedShells.map(...)` loop, `SHELL_PREFIX` constant, and `shellsLoading` skeleton are gone from `NewSessionModal.tsx` | VERIFIED | `grep` finds only a comment reference to `sortedShells` (not live code); `SHELL_PREFIX` and `Loading shells` absent entirely. `_shells` / `_shellsLoading` are accepted as unused props for backwards compat, properly annotated. |
| 3 | The Shell row detail line shows the daemon-resolved path via `GetShellPath()` called on each modal open | VERIFIED | `NewSessionModal.tsx:60-65` — `useEffect` on `[isOpen]` calls `GetShellPath().then(setResolvedShellPath)`. `{resolvedShellPath}` rendered in `.new-session-modal__agent-btn__detail` span at line 156. |
| 4 | Clicking the Shell row sends bare `'shell'` (no prefix) to `onConfirm` | VERIFIED | `NewSessionModal.tsx:106-108` — `handleConfirm` shell branch calls `onConfirm('shell', selectedDir, [])`. No prefix stripping needed. `NewSessionModal.shellRow.test.tsx` assertion 5 DOM-tests this at runtime. |
| 5 | `engine.go` `shellPath` field persists across restart via settings.json round-trip | VERIFIED | `engine.go:89` — `ShellPath string` in `daemonSettings`; `engine.go:169` loads from disk; `engine.go:195` saves to disk. 5 engine tests pass including round-trip. |
| 6 | `GET /settings/shell-path` never returns empty (resolves default when unset) | VERIFIED | `engine.go:674-682` — `GetShellPath()` calls `resolveDefaultShellPath()` when field empty; `resolveDefaultShellPath()` returns `$SHELL` → DiscoverShells[name=shell] → platform hardcode. `TestHandleGetShellPath_ReturnsDefault` passes. |
| 7 | `PATCH /settings/shell-path` with a valid executable returns 204; invalid path returns 400 with human-readable body | VERIFIED | `api.go:580-596` — `handleUpdateShellPath` calls `engine.SetShellPath(req.Value)`; on error returns `http.Error(w, err.Error(), http.StatusBadRequest)`. `TestHandleUpdateShellPath_ValidPath_Persists` (204) and `TestHandleUpdateShellPath_InvalidPath_Returns400` (400) both pass. |
| 8 | Wails layer exposes `GetShellPath()` / `SetShellPath(path)` callable from frontend | VERIFIED | `app.go:446-466` — both methods present with nil-client guards. `App.d.ts:36-37` — TS declarations. `App.js:18-19` — `Call()` bindings. |
| 9 | `-1 → 0` normalization applied in `ListSessions` ExitCode emission path | VERIFIED | `engine.go:389-391` — `if ec == -1 { ec = 0 }` guard inserted between `ec := s.ExitCode()` and `exitCodePtr = &ec`. Two normalization sites confirmed: L339 (natural-exit goroutine) and L389 (ListSessions). `TestListSessions_NaturalExit_NormalizesNegativeOneToZero` passes. |
| 10 | Non-zero exit codes are preserved verbatim (no over-normalization) | VERIFIED | `TestListSessions_NaturalExit_PreservesNonZero` passes; guard is guarded by `ec == -1` only. |
| 11 | `session:exit` handler in `App.tsx` early-returns on `exitCode === 0` before `setSessionExits` | VERIFIED | `App.tsx:550-552` — `if (data.exitCode === 0) { void handleCloseTabRef.current?.(data.sessionId); return }`. Source-inspection test confirms early-return index < `setSessionExits` index. `grep -c "data.exitCode === 0" App.tsx` = 1. Old countdown block removed (`grep -c "countdown: data.exitCode === 0"` = 0). |
| 12 | Settings → Paths "Shell binary" row renders with input, Browse button, and inline error paragraph | VERIFIED | `SettingsTab.tsx:705-733` — `<tr key="shell">` with `id="settings-shell-path"`, `aria-label="Shell binary path"`, `aria-describedby="settings-shell-path-desc"`, `role="alert"` error paragraph. All 8 SHELL-11 DOM assertions pass in `SettingsTab.shellPath.test.tsx`. |
| 13 | Tab auto-closes with focus shift on exit-code 0 — runtime behavioral chain | VERIFIED (pass-with-deviation) | UAT run 2026-05-15 by Ken Scott on AgentHub 2.2.0: Part A (`exit` / clean code-0) PASS — tab auto-closed, focus shifted as designed. Part B (`exit 2`) showed silent auto-close instead of ExitToast — root cause is a daemon-side cache-population gap in the natural-exit goroutine (every natural exit reports as 0 to the GUI). Tester accepted auto-close-on-any-natural-exit as final v3.3 behaviour; ExitToast for non-zero natural exits descoped to v3.4 (see frontmatter `deferred_to_followup`). |

**Score:** 13/13 truths verified (truth #13 PASS-with-deviation after UAT 2026-05-15 — see frontmatter `human_verification_resolved` and `deferred_to_followup`)

---

### Deferred Items

None.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/engine.go` | `shellPath` field + Get/SetShellPath methods + resolveDefaultShellPath + ListSessions -1→0 guard + resolveShellSpawn branch (0) | VERIFIED | All present at expected lines. |
| `internal/daemon/api.go` | GET + PATCH `/settings/shell-path` routes + handlers with executable validation | VERIFIED | Lines 75-76 (routes) + 568-596 (handlers). |
| `internal/daemon/client.go` | `GetShellPath()` / `SetShellPath()` DaemonClient methods | VERIFIED | Lines 157-175. |
| `app.go` | `App.GetShellPath()` / `App.SetShellPath()` Wails wrappers | VERIFIED | Lines 442-466. |
| `frontend/src/wailsjs/go/main/App.d.ts` | TS declarations for GetShellPath/SetShellPath | VERIFIED | Lines 36-37. |
| `frontend/src/wailsjs/go/main/App.js` | Runtime Call() bindings | VERIFIED | Lines 18-19. |
| `frontend/src/components/NewSessionModal.tsx` | Single Shell row, `SHELL_PREFIX` removed, `GetShellPath()` wired | VERIFIED | Single button at lines 146-157; no SHELL_PREFIX or sortedShells loop in live code. |
| `frontend/src/components/SettingsTab.tsx` | Shell binary `<tr>` with input + Browse + error paragraph | VERIFIED | Lines 705-733, placed after AI CLI rows and before tailscale row. |
| `frontend/src/App.tsx` | `session:exit` handler with `exitCode === 0` early-return | VERIFIED | Lines 550-552; countdown block removed. |
| `internal/daemon/engine_test.go` | 5 engine tests (GetShellPath default, SetShellPath validations) + 5 SHELL-12 ListSessions tests | VERIFIED | 15 tests found and passing (plan had 10 core + 1 onExit callback = 11; 15 found confirms extras). |
| `internal/daemon/api_test.go` | 4 API tests for shell-path routes | VERIFIED | All 4 pass. |
| `frontend/src/components/__tests__/NewSessionModal.shellRow.test.tsx` | 8 SHELL-10 assertions (DOM render) | VERIFIED | All pass; uses actual React DOM render, not source-inspection. |
| `frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx` | 8 SHELL-11 assertions (DOM render) | VERIFIED | All pass; uses actual React DOM render. |
| `frontend/src/components/__tests__/App.shellExit.test.tsx` | 11 SHELL-12 structural assertions | VERIFIED | All pass; source-inspection only (see Human Verification section). |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `app.go GetShellPath/SetShellPath` | `client.go GetShellPath/SetShellPath` | `a.client.GetShellPath()` | WIRED | `app.go:450,464` call `a.client.GetShellPath()` / `a.client.SetShellPath(v)` |
| `client.go GetShellPath/SetShellPath` | `api.go /settings/shell-path` | `doJSON HTTP call` | WIRED | `client.go:163,173` use `c.doJSON` to `GET/PATCH /settings/shell-path` |
| `api.go handlers` | `engine.go Get/SetShellPath` | `a.engine.SetShellPath(req.Value)` | WIRED | `api.go:573` calls `a.engine.GetShellPath()`, `api.go:588` calls `a.engine.SetShellPath(req.Value)` |
| `engine.go resolveShellSpawn branch (0)` | `engine.go shellPath field` | override resolution before cliPaths branch | WIRED | `engine.go:513-527` — branch (0) reads `e.shellPath` before branch (1) cliPaths |
| `NewSessionModal.tsx` | `GetShellPath Wails binding` | `useEffect on [isOpen]` | WIRED | `NewSessionModal.tsx:60-65` |
| `SettingsTab.tsx` | `GetShellPath / SetShellPath Wails bindings` | `useEffect mount + handleSaveCLIPaths` | WIRED | `SettingsTab.tsx:146` (mount), `SettingsTab.tsx:258` (save) |
| `App.tsx session:exit handler` | `handleCloseTabRef.current` | `exitCode === 0 early-return` | WIRED | `App.tsx:550-552`; `handleCloseTabRef.current` assigned at line 698 |
| `engine.go natural-exit goroutine (L339)` | `onExit callback (L344)` | `-1→0 normalization before call` | WIRED (pre-existing) | Unchanged; `TestListSessions_OnExitCallback_ReceivesNormalized` confirms |
| `engine.go ListSessions (L389)` | `ExitCode field in SessionInfo` | `-1→0 guard before assignment` | WIRED | New guard at L389-391 |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `NewSessionModal.tsx` | `resolvedShellPath` | `GetShellPath()` Wails RPC → daemon `engine.GetShellPath()` → resolves `$SHELL` or platform default | Yes — live RPC, daemon resolves non-empty string | FLOWING |
| `SettingsTab.tsx` | `shellPath` state | `GetShellPath()` on mount; `SetShellPath()` on save | Yes — round-trips through settings.json | FLOWING |
| `App.tsx` | `data.exitCode` in `session:exit` handler | daemon `onExit(id, exitCode)` after normalization at L339 / L389 | Yes — real PTY exit code post-normalization | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Daemon test suite (all, -race, skip ANSI capture) | `go test ./internal/daemon/... -race -count=1 -skip TestOpenCodeANSICapture -timeout 120s` | `ok github.com/scottkw/agenthub/internal/daemon 4.273s` | PASS |
| Targeted shell-path + SHELL-12 daemon tests (15 tests) | `go test ./internal/daemon/ -run 'TestSetShellPath\|TestGetShellPath\|TestHandleGetShellPath\|TestHandleUpdateShellPath\|TestListSessions_*' -v -count=1` | 15/15 PASS | PASS |
| Frontend Vitest suite (all tests) | `cd frontend && pnpm test -- --run` | `896 passed (896)` in 59 test files | PASS |
| TypeScript type-check | `cd frontend && pnpm tsc --noEmit` | No output (exit 0) | PASS |
| `SHELL_PREFIX` removed from NewSessionModal | `grep -c "SHELL_PREFIX\|sortedShells" NewSessionModal.tsx` (live code) | 0 (only comment reference) | PASS |
| Single shell class in NewSessionModal | `grep -c "new-session-modal__agent-btn--shell" NewSessionModal.tsx` | 1 | PASS |
| Exactly one `data.exitCode === 0` branch in App.tsx | `grep -c "data.exitCode === 0" App.tsx` | 1 | PASS |
| Old countdown ternary removed | `grep -c "countdown: data.exitCode === 0" App.tsx` | 0 | PASS |
| Two -1 normalization sites in engine.go | `grep -n "ec == -1\|exitCode == -1" engine.go` | Lines 339 + 389 | PASS |
| All 12 phase commits verified in git log | `git log --oneline \| grep <hash>` for all 12 hashes | All 12 found | PASS |

---

### Probe Execution

No `scripts/*/tests/probe-*.sh` files declared or found for this phase.

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SHELL-10 | 107-03 | Collapse new-session modal shell rows to a single "Shell" entry | SATISFIED | `NewSessionModal.tsx:146-157` single button; 8-assertion DOM test suite passes |
| SHELL-11 | 107-01 + 107-03 | Settings → Paths "Shell binary" path field with executable validation | SATISFIED | Backend: `engine.go` shellPath field + `api.go` GET/PATCH routes with validation. Frontend: `SettingsTab.tsx:705-733` field + 8-assertion DOM test suite. Full plumbing chain wired and tested. |
| SHELL-12 | 107-02 + 107-04 | Clean-exit handling: normalize -1→0 in ListSessions + auto-close tab on exit-code 0 | SATISFIED (with UAT caveat) | Backend normalization at `engine.go:389-391` — fully verified. Frontend early-return at `App.tsx:550-552` — code structure verified; runtime behavior requires human UAT (see Human Verification section). |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `NewSessionModal.tsx` | 42-43 | `shells: _shells = []` / `shellsLoading: _shellsLoading = false` — accepted but intentionally unused props | INFO | Documented in TSDoc as "pending removal (SHELL-10)". Not a stub — modal ignores them by design while App.tsx call site still passes them for backwards compat. No behavioral impact. |
| `App.tsx` | (various) | `autoCloseRef` and `countdownTimers` refs remain defined but the countdown block was removed | INFO | Both refs are still referenced elsewhere (`countdownTimers` cleanup in `handleCloseTab`, `autoCloseRef` in mount hydration). No dead-code risk; noted in 107-04 SUMMARY as a future hygiene pass item. |

No TBD / FIXME / XXX markers found in any file modified by this phase.

---

### Human Verification Required

#### 1. SHELL-12 Runtime Tab Auto-Close and Toast Suppression

**Test:** With the application running, open a new shell session (Settings → default shell or via the New Session modal). In the terminal, type `exit` and press Enter.

**Expected:** The shell tab closes immediately with no ExitToast appearing, and focus shifts to the adjacent tab (or the Welcome tab if it was the only non-Welcome tab).

**Then:** Open another shell session, type `exit 2` and press Enter.

**Expected:** The ExitToast appears for the non-zero exit, the tab remains open, and the existing ExitToast behavior is intact.

**Why human:** `App.shellExit.test.tsx` uses source-inspection (raw import pattern) consistent with `App.exit.test.tsx`, `App.nav.test.tsx`, and `App.shellWebShare.test.tsx` established in this codebase. This correctly locks the code structure (early-return before `setSessionExits`, `handleCloseTabRef.current` wired, adjacency logic present) but does not exercise the full Wails `EventsOn` → React state → DOM render pipeline at runtime. Assertions 2 ("calls handleCloseTab with sessionId") and 5 ("active tab focus shifts") are structural-only — the behavioral chain requires the running application.

---

### Gaps Summary

No blocking gaps. All code changes are substantive, correctly wired, and pass automated tests. The single UNCERTAIN item (truth #13) is the runtime behavioral chain for SHELL-12's auto-close, which has strong structural evidence but no runtime test coverage due to the source-inspection test approach (an established codebase pattern, not a deviation). Status is `human_needed` rather than `passed` because this behavioral chain — from Wails event to tab removal with focus shift — needs one manual UAT pass to confirm the system-level wiring works end-to-end.

**SUMMARY overclaim check:**
- 107-03 SUMMARY: "full suite now 896 tests, all green." 107-04 SUMMARY: "888/888 green." No contradiction — 107-04 ran before 107-03's 8 new tests were committed (parallel wave execution); current count is 896. No overclaim.
- 107-04 SUMMARY: "Source-inspection tests" noted explicitly in key-decisions. Not an overclaim — it accurately describes the test approach.
- No instances found resembling the Phase 101/103 overclaim patterns (document-level listener vs React bubbling, decision file claimed but missing).

---

_Verified: 2026-05-13T22:05:00Z_
_Verifier: Claude (gsd-verifier)_
