---
phase: 157-terminal-screen-share-semantics-issue-109
verified: 2026-06-27T08:55:00Z
status: human_needed
score: 5/7
behavior_unverified: 2
overrides_applied: 0
behavior_unverified_items:
  - truth: "No garble in two-surface scenario (SC-1) — with host and smaller-windowed guest, guest renders host's exact grid with no overlapping/doubled characters"
    test: "Connect two sessions to the same hub: host at standard size, web guest in a smaller browser window. Observe the guest render — no garble, no doubled characters."
    expected: "Guest screen pixel-scales cleanly; wrap points identical to host; no MC-06-style column misalignment."
    why_human: "Requires a live running daemon with two concurrent viewers at different window sizes. The xterm.js pixel render and transform:scale visual output cannot be observed by grep or vitest."
  - truth: "Web guest (terminal.js) honors server-pushed 0x02 at runtime — term.resize(cols,rows) called and recomputeScale applies CSS transform (VIEW-04/05, web surface)"
    test: "Open a web-share URL in a browser smaller than the host window. Verify the terminal renders at host grid size (no clipped/wrapped output) and is visually downscaled."
    expected: "term.cols/rows match the host PTY grid; a CSS scale transform (s < 1.0) is applied to the .xterm element; no client-driven 0x11 resize frame is sent."
    why_human: "terminal.js is a vendored asset outside the vitest suite. node --check + grep structural gates prove the code exists and is wired, but actual dispatch of the 0x02 branch at WebSocket message-time requires a live browser."
human_verification:
  - test: "Issue #109 two-surface garble check (M-27)"
    expected: "Host PTY + smaller-windowed web guest: no garble (no overlapping/doubled chars); guest grid matches host; downscale CSS transform applied (s <= 1.0). See 157-VALIDATION.md Manual-Only Verifications row 1 + row 2."
    why_human: "Live multi-window xterm.js render required; transform:scale pixel output is not unit-assertable. Web terminal.js is a vendored asset outside vitest scope."
  - test: "Desktop guest cross-surface parity check (M-28)"
    expected: "Open session as remote guest in desktop app via HubModal. Confirm guest panel shows host grid (no sendResize sent back), CSS scale fits viewport, no garble. See 157-VALIDATION.md Manual-Only Verifications row 3."
    why_human: "Requires the native Wails WebView which Playwright cannot access. The isGuest gate and sendResize suppression are unit-proven; visual render parity with web guest requires live comparison."
---

# Phase 157: Terminal Screen-Share Semantics (Issue #109) — Verification Report

**Phase Goal:** Eliminate cross-viewer PTY grid-size garble by adopting screen-share semantics (Option B): the host's terminal is the single source of truth for the PTY grid; guests render at the host's grid size and CSS-scale to fit their viewport, never driving the PTY. Full Option B scope (all six change-layers).
**Verified:** 2026-06-27T08:55:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| SC-1 | Host + smaller-windowed guest renders host's exact grid with no garble | PRESENT_BEHAVIOR_UNVERIFIED | Server arbiter correct (hub tests pass); visual render needs live test — see Human Verification |
| SC-2 | Guest window resize never changes PTY size; only host's resize does (host-authority) | VERIFIED | `TestHub_ResizeIgnoresWebOrigin` (hub -race) + `TestWebReadPump_DropsGuestResize` (webserver -race); MsgResize2 case no-ops at call site in webserver/server.go:1171 |
| SC-3 | Fresh-joined guest receives host grid before scrollback replay | VERIFIED | `TestRelayJoin_PushesResizeBeforeScrollback` (relay -race) + `TestWebJoin_PushesResizeBeforeScrollback` (webserver -race); both confirm 0x02 frame with correct dims precedes snapshot bytes |
| SC-4a | Web guest (terminal.js) honors server-pushed 0x02 → term.resize + CSS scale (VIEW-04/05) | PRESENT_BEHAVIOR_UNVERIFIED | Structural gates pass: `node --check` clean; `recomputeScale` present; `term.resize` in 0x02 branch; `fitAddon.fit()` count = 0; no client-driven 0x11 send from onopen/term.onResize; behavior requires live browser |
| SC-4b | Desktop guest (TerminalPanel) honors server-pushed 0x02 → term.resize + CSS scale (VIEW-04/05) | VERIFIED | `TerminalPanel.scale.test.tsx` 19 tests pass (guest honors onResize→term.resize, scale capped at ≤1, sendResize never called); `terminalScale.test.ts` 10 tests; `relayClient.test.ts` 6 dispatch tests; 76/76 vitest pass |
| SC-5 | Desktop TerminalPanel cross-surface parity: host path pixel-pristine, guest path honors 0x02 | VERIFIED | Host-invariance branch (3 tests): sendResize preserved, no transform applied; `isGuestRef` gate code confirmed at TerminalPanel.tsx:164; rAF dep array restored to `[isActive]` at line 779 |
| SC-6 | MC-06 max-wins tests replaced with host-authority tests; TESTING.md updated | VERIFIED | 6 new hub tests replace 3 MC-06 tests; TESTING.md VIEW-01..05 traceability rows present; `bash tests/check-traceability-paths.sh` exits 0; Category P (M-27/M-28) in Section 5 |

**Score:** 5/7 truths verified (2 present, behavior-unverified — runtime render cannot be checked without live browser/daemon)

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/hub.go` | Host-authority ResizeClient + broadcastResize + Rows() | VERIFIED | All three symbols confirmed at lines 246, 297, 744; VIEW-02 origin gate at line 249; D-02 min-among-local at line 258; D-01 freeze logic at line 273; unlock-before-IO at line 279 |
| `internal/relay/hub_test.go` | 6 host-authority tests replacing MC-06 trio | VERIFIED | All 6 functions found: MinAmongLocal, NoOpWhenDimensionsUnchanged, FreezeLastHostSize, IgnoresWebOrigin, BroadcastsToSubscribers, RowsFallback — all pass under -race |
| `internal/relay/server.go` | VIEW-03 join-push before scrollback (relay path) | VERIFIED | Lines 294-301: `if c, r := hub.Cols(), hub.Rows(); c > 0 && r > 0` → direct `conn.Write(MakeResizeFrame)` before snapshot write |
| `internal/webserver/server.go` | VIEW-03 join-push + VIEW-02 MsgResize2 no-op | VERIFIED | Lines 1133-1140: join-push block; line 1171: `case relay.MsgResize2:` body is comment-only no-op (no ResizeClient call) |
| `internal/relay/server_test.go` | TestRelayJoin_PushesResizeBeforeScrollback | VERIFIED | Function at line 407; passes under -race |
| `internal/webserver/server_test.go` | TestWebJoin_PushesResizeBeforeScrollback + TestWebReadPump_DropsGuestResize | VERIFIED | Functions at lines 861 and 913; both pass under -race |
| `web/assets/terminal.js` | 0x02 dispatch (term.resize + recomputeScale); no fitAddon.fit(); no 0x11 send | VERIFIED (structural) | MsgResize=0x02 const at line 3; 0x02 branch at line 1035; recomputeScale at line 989; fitAddon.fit() grep: 0 occurrences; wsOnopen has comment-only — no 0x11 send; window.resize → recomputeScale at line 1075 |
| `web/assets/terminal.css` | `#terminal overflow:hidden` + `#terminal .xterm transform-origin:top left` | VERIFIED | Line 56: `overflow: hidden` on `#terminal`; line 61: `#terminal .xterm { transform-origin: top left; }` |
| `frontend/src/lib/terminalScale.ts` | computeGuestScale pure helper capped at 1.0 | VERIFIED | Full implementation: zero-grid guard, min-axis, `s > 1 ? 1 : s` cap at line 20 |
| `frontend/src/lib/terminalScale.test.ts` | 10 cap/axis/zero-guard tests | VERIFIED | 10 tests pass in 76/76 vitest run |
| `frontend/src/lib/relayClient.ts` | onResize? callback + 0x02 dispatch (was dropped) | VERIFIED | `onResize?:` at line 206; dispatch at line 269: `this.callbacks.onResize?.(frame.cols, frame.rows)` |
| `frontend/src/lib/relayClient.test.ts` | +6 0x02 dispatch tests | VERIFIED | 6 new tests in 76/76 vitest run |
| `frontend/src/components/TerminalPanel.tsx` | isGuestRef gate + recomputeScale + guest/host branches | VERIFIED | isGuestRef at line 163; recomputeScale useCallback at line 172; guest onResize at line 340-343; host sendResize at line 335/356; isActive dep array `[isActive]` at line 779 |
| `frontend/src/style.css` | `.xterm { transform-origin: top left }` | VERIFIED | Line 60: `transform-origin: top left;` |
| `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` | 19 guest honor + host invariance tests | VERIFIED | 19 tests in 74/74 TerminalPanel test run (test.tsx + scale.test.tsx combined) |
| `TESTING.md` | VIEW-01..05 traceability rows + Category P (M-27/M-28) + Section 2 note | VERIFIED | VIEW-01..05 rows at lines 209-217; Category P at line 391; M-27 at line 395; M-28 at line 405; path-check exits 0 |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `Hub.ResizeClient` | `Hub.broadcastResize` | Called after `hub.mu.Unlock()` (unlock-before-IO, line 279/283) | VERIFIED | Lock released at line 279; broadcastResize at line 283 — cannot cause self-deadlock (T-157-04 mitigated) |
| `relay/server.go` join path | `Hub.Cols()/Hub.Rows()` → `MakeResizeFrame` | Direct `conn.Write` before scrollback write | VERIFIED | Lines 297-300 — direct write, not routed through sub.Msgs |
| `webserver/server.go` join path | `Hub.Cols()/Hub.Rows()` → `relay.MakeResizeFrame` | Direct `conn.Write` before scrollback write | VERIFIED | Lines 1136-1139 — same pattern as relay path |
| `webserver/server.go` MsgResize2 case | `hub.ResizeClient` | Removed — case body is no-op comment | VERIFIED | Line 1171: `case relay.MsgResize2:` with no body call (VIEW-02 / T-157-02) |
| `terminal.js` ws.onmessage | `recomputeScale` | Called from 0x02 branch after `term.resize` | VERIFIED (structural) | Line 1042: `recomputeScale()` after `term.resize(cols, rows)` at line 1041 |
| `relayClient.ts` 0x02 frame | `TerminalPanel` onResize callback | `this.callbacks.onResize?.(frame.cols, frame.rows)` at line 269 | VERIFIED | 6 relayClient tests prove the dispatch fires with correct decoded values |
| `TerminalPanel` onResize callback | `term.resize` + `recomputeScale` | Guest branch: line 341-343 | VERIFIED | TerminalPanel.scale.test.tsx behavioral tests prove term.resize fires and transform is applied |
| `TerminalPanel` container ResizeObserver (guest) | `recomputeScale` (not fitTerminal) | Line 730: `const ro = new ResizeObserver(() => { recomputeScale() })` | VERIFIED | Host branch uses fitTerminal (line 766); guest branch uses recomputeScale (line 730) — mutually exclusive |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `Hub.ResizeClient` | `minCols/minRows` | Iterates live `h.subscribers` map filtered to `Origin=="local"` | Yes — reads real subscriber dimensions | FLOWING |
| `relay/server.go` join-push | `c, r` | `hub.Cols(), hub.Rows()` — reads real `ptyCols/ptyRows` (or 220/50 fallback) | Yes — real PTY grid or meaningful fallback | FLOWING |
| `webserver/server.go` join-push | `c, r` | Same — `hub.Cols(), hub.Rows()` | Yes | FLOWING |
| `TerminalPanel` recomputeScale | `cellW, cellH` | `term._core._renderService.dimensions.css.cell` — real rendered cell dimensions | Yes — live xterm internals | FLOWING |
| `terminalScale.computeGuestScale` | Return value | Pure math on containerW/H and gridW/H | Yes — no hardcoded fallback path returns empty | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Hub host-authority tests pass under -race | `go test -race -run 'TestHub_Resize\|TestHub_Rows' ./internal/relay/` | ok (1.034s) | PASS |
| Server join-push + web-origin-drop tests pass under -race | `go test -race -run 'TestRelayJoin\|TestWebJoin\|DropsGuestResize' ./internal/relay/ ./internal/webserver/` | ok relay (1.062s), ok webserver (1.713s) | PASS |
| Full relay + webserver suites pass under -race | `go test -race -short ./internal/relay/ ./internal/webserver/` | ok relay (4.237s), ok webserver (10.849s) | PASS |
| Frontend 157-specific tests (relayClient, terminalScale, TerminalPanel.scale) | `pnpm test -- --run relayClient terminalScale TerminalPanel.scale` | 3 files, 76/76 tests pass | PASS |
| TerminalPanel regression fix (rAF dep array) | `pnpm test -- --run TerminalPanel.test TerminalPanel.scale` | 2 files, 74/74 tests pass | PASS |
| TypeScript build parity | `npx tsc --noEmit -p tsconfig.json` | Clean (no output) | PASS |
| terminal.js syntax | `node --check web/assets/terminal.js` | Clean (no output) | PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | "OK: all traceability paths exist" | PASS |
| No fitAddon.fit() remaining in terminal.js | `grep -c 'fitAddon\.fit()' web/assets/terminal.js` | 0 occurrences | PASS |
| No client-driven 0x11 from terminal.js (onopen/term.onResize paths removed) | Source inspection of ws.onopen + term.onResize absence | wsOnopen has comment-only; no term.onResize handler installed | PASS |

---

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| VIEW-01 | 157-01, 157-05 | PTY grid tracks host; server broadcasts MakeResizeFrame on host resize | VERIFIED | hub.go ResizeClient + broadcastResize; TestHub_ResizeBroadcastsToSubscribers |
| VIEW-02 | 157-01, 157-02, 157-05 | Web/remote guests never drive PTY; hub origin gate + call-site drop | VERIFIED | hub.go origin gate (line 249); webserver MsgResize2 no-op (line 1171); TestHub_ResizeIgnoresWebOrigin; TestWebReadPump_DropsGuestResize |
| VIEW-03 | 157-02, 157-05 | Guest join receives host grid before scrollback (both surfaces) | VERIFIED | relay/server.go:297-301; webserver/server.go:1136-1140; TestRelayJoin_PushesResizeBeforeScrollback; TestWebJoin_PushesResizeBeforeScrollback |
| VIEW-04 | 157-03, 157-04, 157-05 | Guests honor 0x02 → term.resize on both web and desktop surfaces | PARTIALLY PRESENT_BEHAVIOR_UNVERIFIED | Desktop: VERIFIED (76/76 tests). Web: structural gates only (vendored asset) — M-27 manual UAT required |
| VIEW-05 | 157-03, 157-04, 157-05 | Guests CSS-scale host grid to fit viewport (s ≤ 1.0, downscale-only) | PARTIALLY PRESENT_BEHAVIOR_UNVERIFIED | Desktop: VERIFIED (computeGuestScale 10 tests + TerminalPanel scale tests). Web: structural gates only — M-27 manual UAT required |

All 5 VIEW requirement IDs from REQUIREMENTS.md are covered by at least one plan. No orphaned requirements detected.

---

### Anti-Patterns Found

| File | Pattern | Severity | Assessment |
|------|---------|----------|------------|
| `web/assets/terminal.js` | `makeResizeFrame` function exists but is not called from any VIEW-04 send path | INFO | The function was pre-existing (used to send 0x11 on onopen/onResize). Those call sites are removed. The bare function definition is dead code but harmless — it is NOT a stub for the 0x02 honor path (that uses server-pushed frames, not a client-sent frame). Not a blocker. |

No TBD/FIXME/XXX markers found in any file modified by this phase. No placeholder returns or empty implementations in the new symbols.

---

### Human Verification Required

#### 1. Issue #109 Two-Surface Garble Check (M-27)

**Test:** With the daemon running and a host session active, open the web-share URL in a browser window smaller than the host terminal. Observe the guest render.
**Expected:** Guest terminal shows the host's exact column/row grid with no overlapping or doubled characters. The xterm element has a `transform: scale(s)` with s ≤ 1.0 applied (visible via browser DevTools). No 0x11 resize frame is sent from the browser to the server (verify in browser Network tab). See `157-VALIDATION.md` Manual-Only Verifications rows 1 and 2.
**Why human:** Live multi-window xterm.js render with CSS transform required. The downstream pixel result of `term.resize(cols,rows)` followed by `transform: scale(s)` is not assertable by grep or vitest. The web `terminal.js` viewer is a vendored asset outside vitest scope.

#### 2. Desktop Guest Cross-Surface Parity Check (M-28)

**Test:** Open the same session in the desktop app as a remote guest via the HubModal (not as the owner/host). Observe the guest TerminalPanel.
**Expected:** The desktop guest panel renders at the host's grid size (not self-sized by fitTerminal). No resize event is sent back to the server. CSS scale transform is applied to fit the panel. Behavior is visually equivalent to the web guest from M-27. See `157-VALIDATION.md` Manual-Only Verifications row 3.
**Why human:** Requires the native Wails WebView (not accessible to Playwright). The `isGuest = remote || !!wsURL` gate and sendResize suppression are unit-proven (TerminalPanel.scale.test.tsx); visual render parity with the web guest requires live comparison.

---

### Gaps Summary

No gaps. All automated checks pass. The two PRESENT_BEHAVIOR_UNVERIFIED truths correspond to the live-render scenarios documented in TESTING.md Category P (M-27/M-28) and are the known manual-UAT items the plans explicitly anticipated from the outset.

**Pre-existing failing test (not a Phase 157 gap):** `frontend/src/components/__tests__/style.hub.modal.test.ts` (MODAL-01 reduced-motion) was already failing at baseline commit `6a7139a8` before any Phase 157 work — confirmed by context. It concerns Phase 154 Hub-modal CSS (`prefers-reduced-motion: reduce` block missing `animation: none`), entirely unrelated to terminal screen-share semantics.

---

_Verified: 2026-06-27T08:55:00Z_
_Verifier: Claude (gsd-verifier)_
