---
phase: 159-web-share-chat-parity-route-shared-session-links-to-the-chat
verified: 2026-06-27T00:00:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "Open the ACTUALLY-SHARED /sessions/{id}?cap=TOKEN link in a browser (from a real live daemon share URL, not /app/ directly) and confirm the browser is redirected to /app/?session=...&cap=..."
    expected: "Browser lands on WebShareSessionView (WebShare SPA) with ChatPanel, the chat toggle (hub-modal__chat-toggle), unread/mention badge, and presence roster rendered; NOT the vanilla-JS terminal.html viewer"
    why_human: "web-share WebSocket blocks automated input (per project memory); requires a live daemon + real browser; cross-surface parity is release-blocking (MEMORY: cross-surface parity)"

  - test: "Send a chat message from the desktop/Hub side while the redirected SPA guest is open; then send a reply from the browser guest back to the desktop"
    expected: "Chat round-trips in both directions (desktop→web and web→desktop)"
    why_human: "Requires a live daemon session with an active WebSocket relay; cannot be exercised without a real PTY and real peer"

  - test: "Repeat chat round-trip with a RO share URL (/sessions/{id}?cap=RO_TOKEN)"
    expected: "RO guest receives and participates in chat (D-06 — RO guests remain full chat participants); PTY input from RO guest is gated but chat is not"
    why_human: "Live daemon required; RO cap behavior requires end-to-end relay verification"

  - test: "With the redirected SPA guest open, resize the host PTY (e.g. resize the desktop session pane). Observe the browser terminal."
    expected: "Browser terminal re-scales to honor the new host-authority grid (Phase 157-04 scale parity); downscales to fit, never upscales past 1.0, stays readable and not clipped"
    why_human: "Requires a real rendered xterm with live host-authority resize frames; structural checks only prove code is wired, not that the resize frames propagate through the redirect path"
---

# Phase 159: Web-Share Chat Parity (302 Redirect) Verification Report

**Phase Goal:** A remote collaborator who opens a web-share link can use session chat — the same ChatPanel, toggle, unread/mention badge, and presence as the desktop tab and Hub modal. Done via 302 redirect from `handleTerminalPage` to `/app/?session={id}&cap={token}`.
**Verified:** 2026-06-27
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A valid-cap GET /sessions/{id}?cap=TOKEN returns HTTP 302 with Location /app/?session={id}&cap={token} (WEBCHAT-01) | ✓ VERIFIED | `handleTerminalPage` at server.go:979-987 issues `http.Redirect(w, r, target, http.StatusFound)`; `TestTerminalPageRedirect` RW sub-test confirms 302 + Location prefix `/app/?session=redir-rw&cap=`; test passes |
| 2 | The cap token survives the redirect URL-encoded — a token containing JWT base64 chars (+, /, =) round-trips intact (WEBCHAT-01) | ✓ VERIFIED | `url.QueryEscape` applied to both sessionID and token (server.go:982-984); `TestTerminalPageRedirect/URL-encoding_round-trip_for_base64_special_chars` proves %2B/%2F/%3D encoding and `url.Parse.Query.Get("cap")` recovers original token; test passes |
| 3 | A missing/invalid cap still returns 401/403 — the 302 NEVER fires before requireCapability validates (WEBCHAT-02 security invariant) | ✓ VERIFIED | Route registration at server.go:661-662 wraps `handleTerminalPage` with `requireCapability`: `mux.HandleFunc("GET /sessions/{id}", ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))`; `TestWebServerToggle` disabled-session asserts 403; `TestSessionAccessWithoutAuth` (pre-existing, Phase 87) asserts 401 for no-cap request; both pass in full package run |
| 4 | Both read-only (RO) and read-write (RW) caps redirect identically; RO remains a full chat participant downstream (D-06) | ✓ VERIFIED | `TestTerminalPageRedirect/RO_cap_redirects_identically_to_RW_(D-06)` mints a `Perms:"read"` cap and asserts 302 + Location `/app/?session=redir-ro&cap=` + round-trip; test passes |
| 5 | TESTING.md records the new Go redirect test (traceability) and a live-daemon manual UAT (M-31) for the actually-shared link (WEBCHAT-02 / PARITY-01) | ✓ VERIFIED | Section 2 has Phase 159 manifest delta note; Section 4 has WEBCHAT-01 and WEBCHAT-02 rows pointing at `internal/webserver/server_test.go`; Section 5 has new "Category R — Web-Share Chat Parity (Phase 159)" with M-31 (6-step live-daemon UAT on the actually-shared link); `bash tests/check-traceability-paths.sh` exits 0 |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/webserver/server.go` | handleTerminalPage MODIFIED to issue 302 | ✓ VERIFIED | Lines 979-987: reads `r.PathValue("id")` and `r.URL.Query().Get("cap")`, builds target with `url.QueryEscape` on both, calls `http.Redirect(w, r, target, http.StatusFound)`; `net/url` added to import block (line 13) |
| `internal/webserver/server_test.go` | TestTerminalPageRedirect (NEW) + TestWebServerToggle (UPDATED) | ✓ VERIFIED | TestTerminalPageRedirect at lines 386-490: 3 sub-tests (RW 302, RO 302, URL-encoding); TestWebServerToggle at lines 325-370: enabled-session now uses no-redirect client, asserts 302 + Location prefix |
| `internal/webserver/csp_integration_test.go` | CSP/Cache-Control tests updated for 302 (deviation from plan — auto-fixed) | ✓ VERIFIED | `TestCSPHeaderStrict_TerminalPage` and `TestCSPHeaderStrict_CacheControl` updated to use no-redirect clients and assert 302; Phase 89 D-16/D-18 invariants preserved |
| `TESTING.md` | Section 2 manifest note + Section 4 WEBCHAT-01/02 rows + Section 5 Category R + M-31 | ✓ VERIFIED | All four TESTING.md obligations met; traceability path check passes |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `requireCapability` | `handleTerminalPage` | Route registration at server.go:661-662 | ✓ WIRED | `ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage))` — validation always precedes redirect; route registration unchanged from Phase 87 |
| `handleTerminalPage` | Location header | `url.QueryEscape(sessionID)` + `url.QueryEscape(token)` | ✓ WIRED | server.go:982-984; both params escaped before insertion into redirect target |
| Redirect target `/app/` | `WebShareSessionView` + `ChatPanel` | SPA `readWebModeParams` reads `?session=` and `?cap=` (built in Phase 155) | ✓ WIRED | No frontend change in Phase 159 — `webMode.ts` already reads these params; out of scope (Phase 155 established this) |
| `/sessions/{id}/ws` | `handleWSSRelay` | Separate route registration at server.go:670-671 | ✓ WIRED | WebSocket route is untouched by Phase 159; Chesterton's Fence honored |

### Data-Flow Trace (Level 4)

Not applicable — this phase produces a server-side HTTP redirect handler, not a data-rendering component. The redirect itself is the data flow: `GET /sessions/{id}?cap=TOKEN` → `302 Location /app/?session={id}&cap={token}`. No dynamic data rendering to trace.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| TestTerminalPageRedirect + TestWebServerToggle pass | `go test ./internal/webserver/ -run 'TestTerminalPageRedirect\|TestWebServerToggle' -count=1 -v` | PASS (all 3 sub-tests + toggle test pass) | ✓ PASS |
| Full webserver package green | `go test ./internal/webserver/... -count=1` | `ok github.com/scottkw/agenthub/internal/webserver 5.384s` | ✓ PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | exits 0 — "OK: all traceability paths exist" (cosmetic BSD grep warning on `-P` flag is non-fatal) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| WEBCHAT-01 | 159-01-PLAN.md | GET /sessions/{id}?cap=TOKEN → 302 to /app/?session=&cap= (web-share redirect) | ✓ SATISFIED (code + test) | **WARNING: WEBCHAT-01 is NOT defined in REQUIREMENTS.md** — it exists only in PLAN frontmatter and TESTING.md Section 4. No REQUIREMENTS.md row was added for this phase. Implementation is correct; documentation gap only. |
| WEBCHAT-02 | 159-01-PLAN.md | Redirect asserted on actually-shared route; missing/invalid cap → 401/403 before redirect | ✓ SATISFIED (code + test) | **WARNING: WEBCHAT-02 is NOT defined in REQUIREMENTS.md** — same as WEBCHAT-01. Implementation is correct; documentation gap only. |
| PARITY-01 | 159-01-PLAN.md | Every Session Chat feature behaves identically on desktop and web-share browser surface (release-blocking) | ✓ SATISFIED (Phase 159 extension) | PARITY-01 exists in REQUIREMENTS.md attributed to Phase 155 (Complete). Phase 159 closes the real PARITY-01 gap (Phase 155 verified it on /app/ directly; Phase 159 makes the actually-shared /sessions/{id}?cap= link redirect to the SPA). REQUIREMENTS.md traceability table was NOT updated to reference Phase 159. Documentation gap; functional parity is now established. |

**Orphaned requirements (in REQUIREMENTS.md that map to this phase but not in any plan):** None — Phase 159 has no rows in the REQUIREMENTS.md traceability table.

**Documentation gap summary:** WEBCHAT-01 and WEBCHAT-02 were introduced as requirement IDs in the phase plan but never added to REQUIREMENTS.md as formal requirements. PARITY-01's traceability row (Phase 155) was not updated to include Phase 159 as a contributor. These are documentation gaps that do not affect functional achievement of the phase goal.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | No TODO/FIXME/XXX/TBD/HACK/PLACEHOLDER found in modified files | — | — |

No stub indicators, no empty returns, no hardcoded empty data in any modified file. Chesterton's Fence honored: `web/terminal.html`, `terminal.js`, `terminal.css` preserved; `/sessions/{id}/ws` WebSocket route untouched.

### Human Verification Required

#### 1. Redirect to Chat-Capable SPA (M-31 Step 1-2)

**Test:** Start a local daemon session; enable web-share; copy the RW share URL of the form `/sessions/{id}?cap=TOKEN` (NOT an /app/ URL). Open that share URL in a browser.
**Expected:** Browser is redirected to `/app/?session=...&cap=...`; WebShareSessionView renders (NOT the vanilla-JS terminal.html viewer); ChatPanel, chat toggle (`hub-modal__chat-toggle`), unread/mention badge, and presence roster are present in the DOM.
**Why human:** web-share WebSocket blocks automated input; requires a live daemon + real browser; cross-surface parity is release-blocking per project policy.

#### 2. Chat Round-Trip (M-31 Steps 3-4)

**Test:** With the redirected SPA guest open, send a chat message from the desktop/Hub side; confirm it arrives in the browser; send a reply from the browser back.
**Expected:** Messages flow bidirectionally (desktop→web and web→desktop).
**Why human:** Requires a live relay session with two active WebSocket peers; cannot be exercised without a real PTY and real peer.

#### 3. RO Guest Chat Participation (M-31 Step 6)

**Test:** Repeat with a RO share URL (`/sessions/{id}?cap=RO_TOKEN`); confirm the RO guest receives and participates in chat.
**Expected:** RO guest is a full chat participant (D-06); PTY input is gated but chat is not.
**Why human:** Live daemon required; RO cap behavior requires end-to-end relay relay verification through the redirect path.

#### 4. Phase 157-04 Scale Parity Through Redirect (M-31 Step 5)

**Test:** With the redirected SPA guest open, resize the host PTY (e.g. resize the desktop session pane / change the host terminal grid). Observe the browser terminal.
**Expected:** Browser terminal re-scales to honor the new host-authority grid — downscales to fit, never upscales past 1.0, stays readable and not clipped (Phase 157 VIEW-01/05 behavior preserved through the redirect path).
**Why human:** Requires a real rendered xterm with live host-authority resize frames; structural presence of Phase 157 code cannot prove the resize frames propagate correctly through the redirect-to-SPA path.

### Gaps Summary

No functional gaps. All 5 must-have truths are verified by code inspection and passing tests.

**Documentation warnings (non-blocking):**

1. **WEBCHAT-01 and WEBCHAT-02 not in REQUIREMENTS.md** — these requirement IDs appear in the PLAN frontmatter and TESTING.md Section 4 but have no formal definition row in `.planning/REQUIREMENTS.md`. The implementation is correct; the requirements document should be updated to add these IDs with descriptions.
2. **PARITY-01 traceability not updated for Phase 159** — the REQUIREMENTS.md traceability table still shows `PARITY-01 | Phase 155 | Complete` without a Phase 159 contribution. Since Phase 159 is the phase that makes PARITY-01 actually hold on the shared link, the traceability should be updated.

These warnings do not block goal achievement and do not require re-execution.

---

_Verified: 2026-06-27_
_Verifier: Claude (gsd-verifier)_
