---
phase: 74-multi-client-fan-out
verified: 2026-04-15T03:10:00Z
status: human_needed
score: 5/5
overrides_applied: 0
human_verification:
  - test: "Open two browser tabs to the same session's web URL and type in one tab"
    expected: "Both tabs show identical live PTY output in real-time; input from the active tab appears in both"
    why_human: "Verifying real WebSocket fan-out across browser tabs requires a running server and visual confirmation of no dropped bytes"
  - test: "CLI attach with --readonly flag and attempt typing"
    expected: "Output streams normally; keystrokes produce no effect on the PTY process; detach key (Ctrl-backslash) still works"
    why_human: "End-to-end CLI UX with raw terminal mode cannot be verified programmatically"
  - test: "Resize one of two connected terminals while both are attached"
    expected: "PTY dimensions stabilize to the largest terminal; no continuous resize flicker; the smaller terminal sees truncated lines but no crash"
    why_human: "Visual resize behavior and absence of flicker requires human observation"
---

# Phase 74: Multi-Client Fan-Out Verification Report

**Phase Goal:** Multiple WebSocket clients can connect to the same session simultaneously, with independent control over scrollback, read-only access, visible identity, and stable PTY dimensions
**Verified:** 2026-04-15T03:10:00Z
**Status:** human_needed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Two browser tabs (or CLI attaches) connected to the same session both receive live PTY output without either dropping bytes | VERIFIED | Hub fan-out broadcasts to all subscribers via `broadcast()` in hub.go:156-167; `TestHub_TwoClientsFanOut` in server_test.go:190-218 verifies two WebSocket clients receive identical output |
| 2 | Each client can independently scroll its own scrollback without affecting what other clients see | VERIFIED | Each client gets independent `ScrollbackSnapshot()` on connect (server.go:88); each has own `Msgs` channel; `TestHub_ReconnectScrollback` in server_test.go:222-284 verifies scrollback replay is independent |
| 3 | A client connected with `--readonly` flag receives output but keystrokes are discarded -- the PTY process is not disturbed | VERIFIED | `ReadOnly` field on Subscriber (hub.go:17); `if !sub.ReadOnly` guard in relay/server.go:109 and webserver/server.go:447; `--readonly` CLI flag in cmd_attach.go:35; `TestServer_ReadOnlyClientInputDiscarded` in server_test.go:91-146 proves input discarded and output received |
| 4 | `agenthub list` (or session metadata API) shows the current viewer count for each session | VERIFIED | `ViewerCount int` on SessionInfo (types.go:12); `hub.SubscriberCount()` called in engine.go:152-153; `TestAPI_ListSessionsViewerCount` in api_test.go:658-743 verifies delta-based viewer count through subscribe/unsubscribe lifecycle |
| 5 | When clients have different terminal sizes, PTY dimensions stabilize to the largest active client -- no continuous resize loop occurs | VERIFIED | `ResizeClient()` in hub.go:100-126 implements max-wins policy; `h.mu.Unlock()` at line 120 before `resizeFn` call; both servers call `hub.ResizeClient()` (relay/server.go:116, webserver/server.go:454); old `hub.Resize()` calls fully replaced; `TestHub_ResizeMaxWinsPolicy`, `TestHub_ResizeClientNoOpWhenDimensionsUnchanged`, `TestHub_ResizeClientUnsubscribeDoesNotShrink` provide comprehensive coverage |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/hub.go` | Subscriber metadata fields, Hub dimension tracking, SubscriberCount, ResizeClient | VERIFIED | ReadOnly, Name, Cols, Rows on Subscriber; ptyCols, ptyRows on Hub; SubscriberCount() at line 88; ResizeClient() at line 100 |
| `internal/relay/hub_test.go` | Tests for SubscriberCount, ReadOnly storage, Name storage, ResizeClient max-wins | VERIFIED | 6 new tests (lines 293-490): SubscriberCount lifecycle, ReadOnly flag, Name storage, MaxWins policy, NoOp on unchanged, Unsubscribe recompute |
| `internal/relay/server.go` | Query param parsing, read-only gating, ResizeClient call | VERIFIED | ?readonly= and ?client= parsed at line 47-51; ReadOnly/Name set on Subscriber at lines 73-75; MsgInput gated at line 109; ResizeClient at line 116 |
| `internal/webserver/server.go` | Same changes mirrored for browser WebSocket path | VERIFIED | Identical pattern at lines 385-389 (params), 411-414 (fields), 447 (gating), 454 (ResizeClient) |
| `internal/relay/server_test.go` | Test for read-only enforcement at relay layer | VERIFIED | TestServer_ReadOnlyClientInputDiscarded (line 91), TestServer_ReadOnlyClientReceivesOutput (line 150), TestServer_ClientNameQueryParam (line 171), plus TestHub_TwoClientsFanOut and TestHub_ReconnectScrollback |
| `internal/daemon/types.go` | ViewerCount int field on SessionInfo | VERIFIED | Line 12: `ViewerCount int json:"viewerCount"` |
| `internal/daemon/engine.go` | ListSessions populates ViewerCount from hub.SubscriberCount() | VERIFIED | Lines 148-154: manager.Get + hub.SubscriberCount(); ViewerCount in SessionInfo literal at line 162 |
| `cmd_attach.go` | --readonly and --client= flag parsing, URL construction with query params | VERIFIED | Lines 32-50: flag parsing; lines 87-102: url.URL builder with q.Set("readonly","1") and q.Set("client", clientName) |
| `internal/daemon/api_test.go` | Test verifying viewerCount appears in session list API | VERIFIED | TestAPI_ListSessionsViewerCount at line 658: baseline-delta approach, verifies subscribe +1 and unsubscribe back to baseline |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| internal/relay/hub.go | Hub.resizeFn | ResizeClient calls resizeFn after unlock | WIRED | hub.go:120 `h.mu.Unlock()` followed by hub.go:122-123 `if needResize && h.resizeFn != nil { return h.resizeFn(maxCols, maxRows) }` |
| internal/relay/server.go | internal/relay/hub.go | hub.ResizeClient(sub, int(cols), int(rows)) | WIRED | server.go:116 calls hub.ResizeClient; old hub.Resize() call completely removed |
| internal/webserver/server.go | internal/relay/hub.go | hub.ResizeClient(sub, int(cols), int(rows)) | WIRED | webserver/server.go:454 calls hub.ResizeClient; old hub.Resize() call completely removed |
| internal/daemon/engine.go | internal/relay/hub.go | e.manager.Get(s.ID) then hub.SubscriberCount() | WIRED | engine.go:152-153 calls manager.Get then hub.SubscriberCount() |
| cmd_attach.go | internal/relay/server.go | WebSocket URL with ?readonly=1&client=name query params | WIRED | cmd_attach.go:93-98 uses q.Set("readonly","1") and q.Set("client",clientName) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| internal/daemon/types.go (SessionInfo.ViewerCount) | viewerCount | hub.SubscriberCount() via engine.go:152 | Yes -- returns len(h.subscribers) under mutex | FLOWING |
| internal/relay/server.go (Subscriber.ReadOnly) | readonly | r.URL.Query().Get("readonly") at line 47 | Yes -- parsed from real HTTP request query params | FLOWING |
| internal/relay/server.go (Subscriber.Name) | clientName | r.URL.Query().Get("client") at line 48 | Yes -- parsed from real HTTP request query params | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All relay tests pass (hub + server) | `go test ./internal/relay/... -run "TestHub_SubscriberCount\|TestHub_ReadOnly\|TestHub_ClientName\|TestHub_ResizeMax\|TestHub_ResizeClient\|TestServer_ReadOnly\|TestServer_ClientName\|TestHub_TwoClientsFanOut\|TestHub_ReconnectScrollback" -count=1 -timeout 30s -short` | ok (0.523s) | PASS |
| Daemon API ViewerCount test passes | `go test ./internal/daemon/... -run "ViewerCount" -count=1 -timeout 30s` | ok (0.029s) | PASS |
| Full project builds | `go build ./...` | exit 0, no errors | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-----------|-------------|--------|----------|
| MC-01 | 74-01 | Multiple WebSocket clients receive live output simultaneously | SATISFIED | Hub fan-out in hub.go broadcast(); TestHub_TwoClientsFanOut |
| MC-02 | 74-01 | Independent scrollback position per client | SATISFIED | Per-client ScrollbackSnapshot() on connect; independent Msgs channels; TestHub_ReconnectScrollback |
| MC-03 | 74-01, 74-02, 74-03 | Read-only mode via --readonly flag | SATISFIED | Subscriber.ReadOnly field, server gating, CLI flag, TestServer_ReadOnlyClientInputDiscarded |
| MC-04 | 74-01, 74-03 | Viewer count in session metadata API | SATISFIED | SessionInfo.ViewerCount, hub.SubscriberCount(), TestAPI_ListSessionsViewerCount |
| MC-05 | 74-01, 74-02, 74-03 | Client identity name at connection | SATISFIED | Subscriber.Name field stored, ?client= parsed in both servers, 64-char cap, --client= CLI flag, TestServer_ClientNameQueryParam |
| MC-06 | 74-01, 74-02 | PTY resize arbitration (max-wins) | SATISFIED | Hub.ResizeClient with max-wins policy, ptyCols/ptyRows tracking, resizeFn after unlock, three comprehensive tests |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | -- | No TODO, FIXME, HACK, placeholder, or stub patterns found | -- | -- |

No anti-patterns detected in any of the 7 modified files.

### Human Verification Required

### 1. Multi-Client Browser Fan-Out

**Test:** Open two browser tabs to the same session's web URL. Type commands in one tab.
**Expected:** Both tabs show identical live PTY output in real-time. Input from one tab appears in both. Neither tab drops bytes or freezes.
**Why human:** Verifying real WebSocket fan-out across browser tabs requires a running server, web UI, and visual confirmation of no dropped bytes or race conditions.

### 2. CLI Read-Only Attach

**Test:** Run `agenthub attach <session-id> --readonly` from CLI. Attempt typing.
**Expected:** Terminal output streams normally from the session. Keystrokes produce no effect on the PTY process. Detach key (Ctrl-backslash) still works to exit.
**Why human:** End-to-end CLI UX with raw terminal mode, signal handling, and detach key behavior cannot be verified programmatically.

### 3. Multi-Client Resize Stability

**Test:** Attach two terminals of different sizes to the same session. Resize one terminal while both are connected.
**Expected:** PTY dimensions stabilize to the largest terminal. No continuous resize flicker. The smaller terminal sees truncated or wrapped lines but no crash or corruption.
**Why human:** Visual resize behavior, absence of flicker, and terminal rendering correctness require human observation.

### Gaps Summary

No gaps found. All 5 roadmap success criteria are verified with implementation evidence, test coverage, and passing behavioral spot-checks. All 6 requirements (MC-01 through MC-06) are satisfied with concrete implementation artifacts.

All 8 commit hashes from the 3 plan summaries are verified in the git log:
- 72a91c7 (RED), beb8e3a (GREEN) -- Plan 01
- b0114a8, df71a4b, 1affe11 -- Plan 02
- f38fbf7, 2768b94, 73940cf -- Plan 03

3 items require human verification to confirm the UX behaviors match expectations in a live environment.

---

_Verified: 2026-04-15T03:10:00Z_
_Verifier: Claude (gsd-verifier)_
