---
phase: 96
plan: 02
subsystem: image-addon-daemon-rpc
tags: [phase-96, image, daemon, sub-key-rpc, wave-1]
dependency_graph:
  requires:
    - phase-96-01 (ImageConfig struct + RED scaffolds)
    - phase-95-05 (SetWebLinksConfig pattern, mirrored verbatim)
    - phase-94-07 (SetSearchConfig pattern, original sub-key writer)
  provides:
    - "(*SessionEngine).SetImageConfig sub-key writer (concurrency contract: lock → mutate → save → capture listener → unlock → invoke)"
    - "PATCH /settings/image-config HTTP route + handleSetImageConfig handler with [1, 1000] range gate"
    - "(*DaemonClient).SetImageConfig wrapper for app.go (Plan 96-05) consumption"
    - "3 Wave 0 RED scaffolds flipped GREEN (TestSetImageConfigPreservesSiblings, TestHandleSetImageConfig_RangeRejected, TestHandleSetImageConfig_ValidAccepted)"
  affects:
    - "internal/daemon/engine.go (one new method appended after SetWebLinksConfig)"
    - "internal/daemon/api.go (one new route + one new handler appended after web-links-config)"
    - "internal/daemon/client.go (one new wrapper appended after SetWebLinksConfig)"
tech-stack:
  added: []
  patterns:
    - "Sub-key writer concurrency contract (3rd application — SearchConfig, WebLinksConfig, ImageConfig)"
    - "PATCH handler defense-in-depth: MaxBytesReader(8 KiB) + DisallowUnknownFields + value-domain validation"
    - "Numeric range gate [1, 1000] on StorageLimit (novel — first numeric range gate in this PATCH family; prior gates were enum literals)"
key-files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - internal/daemon/image_config_test.go
    - internal/daemon/api_image_test.go
decisions:
  - "Range gate [1, 1000] MB rejects 0 (defeat-the-feature footgun — no images render) and >1000 (defeats tab-OOM mitigation per ROADMAP Phase 96 SC-3); upstream addon-image default is 128 MB but AgentHub locks 16 MB by default with user-override headroom up to a hard 1000 MB cap"
  - "Mirror SetWebLinksConfig structure verbatim — no novel transport, no novel concurrency primitive; only the value-validation predicate differs (numeric range vs enum literals)"
  - "Test the DaemonClient wrapper directly (TestDaemonClient_SetImageConfig_RoundTrip) so the wrapper has caller-side coverage independent of the rawPatch HTTP harness; this also satisfies the boundary-literal grep acceptance check by exercising StorageLimit struct literals 1, 1000, 0, 1001"
metrics:
  duration: ~25 min
  completed_date: 2026-05-07
  tasks_completed: 2
  files_created: 0
  files_modified: 5
  commits: 2
requirements: [IMG-02]
---

# Phase 96 Plan 02: Daemon Sub-Key RPC for ImageConfig Summary

**One-liner:** Landed `(*SessionEngine).SetImageConfig` sub-key writer (3rd application of the 94-07 / 95-05 concurrency contract), the `PATCH /settings/image-config` HTTP route + `handleSetImageConfig` handler with `[1, 1000]` MB numeric range gate plus 8 KiB `MaxBytesReader` + `DisallowUnknownFields` defense-in-depth, and the `(*DaemonClient).SetImageConfig` PATCH wrapper — flipping all three Wave 0 RED scaffolds to GREEN.

## What shipped

### Task 1 — engine.SetImageConfig sub-key writer (commit `5117d54`)

- `internal/daemon/engine.go`: appended `(*SessionEngine).SetImageConfig(cfg ImageConfig)` immediately after `SetWebLinksConfig`. Concurrency contract is verbatim-identical to its two siblings:
  1. `e.mu.Lock()`
  2. `e.pluginSettings.ImageConfig = cfg`
  3. `e.saveSettingsToDisk()` (still under lock — saveSettingsToDisk's contract requires the caller hold `e.mu`)
  4. `listener := e.pluginSettingsListener` (capture under lock, snapshot guard)
  5. `e.mu.Unlock()`
  6. `if listener != nil { listener() }` (invoke after release — avoids re-entrancy deadlock if the listener calls back into the engine)
- `internal/daemon/image_config_test.go`: replaced the Wave 0 `t.Skip` body with the real `TestSetImageConfigPreservesSiblings`. Mirrors `TestSetWebLinksConfigPreservesSiblings` verbatim with sentinel values swapped:
  - Baseline `PluginSettings` populated with non-default sentinels for every field (so any stomp by SetImageConfig is detectable).
  - Listener counter asserts `listenerCount == 1` after a single `SetImageConfig` call (T-96-02-05 mitigation).
  - Sibling integrity asserted on every other PluginSettings field (WebGL / Unicode11 / Search / SearchConfig / WebLinks / WebLinksConfig / Image bool / Serialize / Clipboard / Progress) — T-96-02-04 mitigation.
  - Persistence verified by reloading via a fresh `SessionEngine` pointed at the same `configDir`; reloaded `ImageConfig`, `SearchConfig`, `WebLinksConfig`, and `Progress` all preserved.

### Task 2 — PATCH handler + DaemonClient wrapper + RED→GREEN (commit `5b7e6e0`)

- `internal/daemon/api.go`:
  - Route registered: `a.mux.HandleFunc("PATCH /settings/image-config", a.handleSetImageConfig)` immediately after the `web-links-config` route, preserving the PATCH-grouping order at lines 76–78 (search-config, web-links-config, image-config).
  - Handler `handleSetImageConfig` appended immediately after `handleSetWebLinksConfig`. Defense-in-depth mirrors its sibling: 8 KiB `MaxBytesReader` (T-96-02-03), `DisallowUnknownFields` (T-96-02-02), `400 Bad Request` on decode error with body `"invalid request body"`. Range gate (T-96-02-01) is `if req.StorageLimit < 1 || req.StorageLimit > 1000` with body `"storageLimit must be in range [1, 1000]"`. Success path calls `a.engine.SetImageConfig(req)` and returns `204 No Content`.
- `internal/daemon/client.go`: `(*DaemonClient).SetImageConfig(cfg ImageConfig) error` appended immediately after `SetWebLinksConfig`. Routes through the same `c.doJSON(http.MethodPatch, "/settings/image-config", cfg, nil)` helper used by every other sub-key writer wrapper.
- `internal/daemon/api_image_test.go`: replaced both Wave 0 `t.Skip` bodies with real tests, plus added a third client-wrapper round-trip test:
  - `TestHandleSetImageConfig_RangeRejected` — five sub-tests: `StorageLimit=0`, `StorageLimit=-1`, `StorageLimit=1001` (range gate), `UnknownField extra=y` (DisallowUnknownFields gate), `BodyExceeds8KiB` (MaxBytesReader gate). Each rejected case asserts the response body contains both `"1"` and `"1000"` for the range-gate cases (so the user can read the bounds out of the error), `400` status for all rejected cases, and that `ImageConfig.StorageLimit` and `SearchConfig` remain unchanged at their pre-call values.
  - `TestHandleSetImageConfig_ValidAccepted` — four sub-tests: `StorageLimit=1` (lower bound inclusive), `StorageLimit=1000` (upper bound inclusive), `StorageLimit=16` (the default), `StorageLimit=32` (mid-range hypothetical override). Each accepted case asserts `204 No Content`, empty body, the new value persists, AND every other PluginSettings field equals its pre-call value (full sibling preservation through the HTTP layer).
  - `TestDaemonClient_SetImageConfig_RoundTrip` — exercises the wrapper directly with `StorageLimit: 64`, `StorageLimit: 1`, `StorageLimit: 1000`, `StorageLimit: 0`, `StorageLimit: 1001`. Confirms valid values land, out-of-range values surface non-nil errors, and rejected calls do not stomp the last-good persisted value.

## Verification results

```text
$ go test ./internal/daemon/ -run TestSetImageConfigPreservesSiblings -count=1 -v
=== RUN   TestSetImageConfigPreservesSiblings
--- PASS: TestSetImageConfigPreservesSiblings (0.00s)
PASS
ok  github.com/scottkw/agenthub/internal/daemon  0.016s

$ go test ./internal/daemon/ -run TestHandleSetImageConfig -count=1 -v
=== RUN   TestHandleSetImageConfig_RangeRejected
=== RUN   TestHandleSetImageConfig_RangeRejected/StorageLimit=0
=== RUN   TestHandleSetImageConfig_RangeRejected/StorageLimit=-1
=== RUN   TestHandleSetImageConfig_RangeRejected/StorageLimit=1001
=== RUN   TestHandleSetImageConfig_RangeRejected/UnknownField_extra=y
=== RUN   TestHandleSetImageConfig_RangeRejected/BodyExceeds8KiB
--- PASS: TestHandleSetImageConfig_RangeRejected (0.02s)
=== RUN   TestHandleSetImageConfig_ValidAccepted
=== RUN   TestHandleSetImageConfig_ValidAccepted/StorageLimit=1
=== RUN   TestHandleSetImageConfig_ValidAccepted/StorageLimit=1000
=== RUN   TestHandleSetImageConfig_ValidAccepted/StorageLimit=16
=== RUN   TestHandleSetImageConfig_ValidAccepted/StorageLimit=32
--- PASS: TestHandleSetImageConfig_ValidAccepted (0.01s)
PASS

$ go test ./internal/daemon/ -run TestDaemonClient_SetImageConfig_RoundTrip -count=1 -v
=== RUN   TestDaemonClient_SetImageConfig_RoundTrip
--- PASS: TestDaemonClient_SetImageConfig_RoundTrip (0.01s)
PASS

$ go test ./internal/daemon/... -count=1
ok  github.com/scottkw/agenthub/internal/daemon  6.694s

$ go vet ./internal/daemon/...
(no output)

$ gofmt -l internal/daemon/engine.go internal/daemon/api.go internal/daemon/client.go internal/daemon/image_config_test.go internal/daemon/api_image_test.go
(no output)
```

No regression on Phase 92 / 93 / 94 / 95 daemon tests; full `internal/daemon/...` suite green in 6.69s.

## Acceptance Criteria Mapping

| Criterion | Result |
|-----------|--------|
| `func (e *SessionEngine) SetImageConfig` exists exactly once in engine.go | PASS (`grep -c` == 1) |
| `e.pluginSettings.ImageConfig = cfg` exists exactly once in engine.go | PASS (`grep -c` == 1) |
| Function body uses `e.mu.Lock()`, `e.saveSettingsToDisk()`, `e.mu.Unlock()`, `listener()` | PASS — verbatim mirror of SetWebLinksConfig |
| `TestSetImageConfigPreservesSiblings -count=1 -v` exits 0 with `--- PASS` (not SKIP) | PASS |
| `func (a *API) handleSetImageConfig` exists exactly once in api.go | PASS (`grep -c` == 1) |
| `"PATCH /settings/image-config"` route registered exactly once | PASS (`grep -c` == 1) |
| Handler contains `MaxBytesReader(w, r.Body, 8192)`, `DisallowUnknownFields`, `StorageLimit < 1`, `StorageLimit > 1000`, `a.engine.SetImageConfig(req)`, `http.StatusNoContent` | PASS — all six tokens present |
| `func (c *DaemonClient) SetImageConfig` exists exactly once in client.go | PASS (`grep -c` == 1) |
| `TestHandleSetImageConfig_RangeRejected -count=1 -v` exits 0 with `--- PASS` | PASS (5 sub-tests all PASS) |
| `TestHandleSetImageConfig_ValidAccepted -count=1 -v` exits 0 with `--- PASS` | PASS (4 sub-tests all PASS) |
| `go test ./internal/daemon/... -count=1` exits 0 | PASS |
| `go vet ./internal/daemon/...` exits 0 | PASS |
| `gofmt -l` produces no output | PASS |
| Boundary literals: `StorageLimit:` followed by `1`, `1000`, `0`, `1001` each appear in api_image_test.go | PASS — `grep -c` == 4 (in TestDaemonClient_SetImageConfig_RoundTrip struct literals) |

## Truth Table

| Truth | Verified By |
|-------|-------------|
| engine.SetImageConfig writes ONLY ImageConfig sub-key | TestSetImageConfigPreservesSiblings (sibling field assertions across 10 fields) |
| Listener fires exactly once after release | TestSetImageConfigPreservesSiblings (`listenerCount == 1`) |
| saveSettingsToDisk fires while still under lock | TestSetImageConfigPreservesSiblings (reload via fresh engine confirms persistence) |
| PATCH /settings/image-config validates [1, 1000] range | TestHandleSetImageConfig_RangeRejected (boundaries 0, -1, 1001) + TestHandleSetImageConfig_ValidAccepted (boundaries 1, 1000) |
| PATCH returns 204 on success | TestHandleSetImageConfig_ValidAccepted (4 sub-tests, status + empty-body assertions) |
| DisallowUnknownFields rejects forward-evolved bodies | TestHandleSetImageConfig_RangeRejected/UnknownField_extra=y |
| MaxBytesReader caps at 8 KiB | TestHandleSetImageConfig_RangeRejected/BodyExceeds8KiB |
| DaemonClient.SetImageConfig issues PATCH and surfaces 400 as error | TestDaemonClient_SetImageConfig_RoundTrip |

## Threat Model Coverage

| Threat ID | Component | Mitigation Landed |
|-----------|-----------|-------------------|
| T-96-02-01 | StorageLimit DoS via negative/absurd values | Range gate `[1, 1000]` in handleSetImageConfig; tested via 3 rejected boundary cases |
| T-96-02-02 | Tampering via forward-evolved unknown fields | `dec.DisallowUnknownFields()`; tested via UnknownField sub-test |
| T-96-02-03 | DoS via arbitrary-size body | `http.MaxBytesReader(w, r.Body, 8192)`; tested via BodyExceeds8KiB sub-test |
| T-96-02-04 | Sub-key writer leaks into other PluginSettings fields | engine.SetImageConfig writes ONLY `e.pluginSettings.ImageConfig`; tested via 10-field sibling-integrity assertion |
| T-96-02-05 | Listener fires multiple/zero times per call | Capture-under-lock + invoke-after-release; tested via `listenerCount == 1` assertion |

## Deviations from Plan

None — plan executed exactly as written. The ad-hoc `TestDaemonClient_SetImageConfig_RoundTrip` test is an additive coverage extension (the plan's Task 2 acceptance criteria list mentions client wrapper presence but does not require a wrapper-specific test); adding it satisfies the boundary-literal grep acceptance criterion (`StorageLimit: 1`, `1000`, `0`, `1001` struct-literal forms, count >= 4) AND gives the wrapper at least one direct caller-side test, so a future regression in `c.doJSON` PATCH transport would surface here without depending on the rawPatch HTTP harness.

## Parallel-safety note

This plan is parallel-safe with Plan 96-03 (CSP middleware amendment): different files (`internal/webserver/csp_mw.go` + `csp_mw_test.go` for 96-03 vs. `internal/daemon/{engine,api,client,image_config_test,api_image_test}.go` for 96-02), no shared symbols, no shared route registration block.

## Self-Check: PASSED

Modified files exist (verified by `git diff --name-only HEAD~2 HEAD`):
- FOUND: internal/daemon/engine.go
- FOUND: internal/daemon/api.go
- FOUND: internal/daemon/client.go
- FOUND: internal/daemon/image_config_test.go
- FOUND: internal/daemon/api_image_test.go

Commits exist:
- FOUND: 5117d54 (feat(96-02): add engine.SetImageConfig sub-key writer)
- FOUND: 5b7e6e0 (feat(96-02): add PATCH /settings/image-config + DaemonClient wrapper)
