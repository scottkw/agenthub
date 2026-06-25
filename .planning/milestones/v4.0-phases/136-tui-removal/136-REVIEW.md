---
phase: 136-tui-removal
reviewed: 2026-06-19T00:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - README.md
  - cmd_cli.go
  - frontend/src/lib/filesApi.ts
  - go.mod
  - internal/attach/attach.go
  - internal/daemon/client.go
  - internal/daemon/remote_files_test_helpers_test.go
  - main.go
findings:
  critical: 0
  warning: 3
  info: 3
  total: 6
status: issues_found
---

# Phase 136: Code Review Report

**Reviewed:** 2026-06-19
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Phase 136 is a pure deletion phase: 32 TUI source files and two daemon parity test files were removed, charm/charmbracelet dependencies were pruned, and a new test-helpers file was extracted to keep the surviving relay tests buildable. The changes are structurally sound — build passes, tests pass (excluding a pre-existing `internal/release` failure), and no TUI imports survive in production code.

Three warning-level issues were found: a context-unaware `http.NewRequest` in the high-traffic `doJSON` helper that predates this phase but remains a correctness risk; stale first-person TUI references in the live README introductory text and architecture section that contradict the current two-surface product; and a misleading "three surfaces" claim in the live README description. Three info-level items cover stale doc comments, historical release notes carrying TUI detail (acceptable as changelog), and a forward-reference call pattern in the new test helper file.

No critical issues were found. No charm/charmbracelet dependencies remain in go.mod. `golang.org/x/term` is correctly preserved (still imported by `cmd_attach.go`, `internal/statusbar/bar.go`, and `internal/attach/attach_unix.go`). The extracted test helper file is correct and complete: all four functions (`newFixtureRemotePeer`, `newDaemonAPIWithUpstreamCert`, `fixtureCap`, `canonicalListResponse`) that `relay_remote_files_test.go` depends on are present; the additionally extracted `canonicalStatResponse` function (also used by tests in that file) is correct. The package declaration `daemon_test` is correct.

---

## Warnings

### WR-01: `doJSON` Uses `http.NewRequest` Without Context — All Control-Plane Calls Are Unleak-able

**File:** `internal/daemon/client.go:686`
**Issue:** The `doJSON` helper — which backs every non-file-browser method on `DaemonClient` (session create/kill/rename, capabilities, settings, web server, relay port, etc.) — calls `http.NewRequest` with no context. This means none of those calls can be cancelled by the caller. In contrast, the five file-browser methods (added later, lines 401–671) all correctly use `http.NewRequestWithContext`. The inconsistency means a hung daemon will block a CLI or GUI call forever with no timeout escape hatch, and callers that pass a `context.Context` (e.g., the GUI's `app.shutdown`) cannot signal cancellation through to `doJSON`-backed requests.

This is a pre-existing defect, not introduced by Phase 136. However, Phase 136 is the first phase touching `client.go`, making it the correct moment to flag it.

**Fix:**
```go
// Change the doJSON signature to accept a context:
func (c *DaemonClient) doJSON(ctx context.Context, method, path string, body, result any) error {
    // ...existing marshal logic...
    req, err := http.NewRequestWithContext(ctx, method, c.base+path, reqBody)
    // ...rest unchanged...
}
```
All callers of `doJSON` would then pass `context.Background()` (or a real context where one is available). Alternatively, as a minimal fix within the current signature, use `http.NewRequestWithContext(context.Background(), ...)` to document intent and allow future threading.

---

### WR-02: Live README Intro Claims "Discoverable From All Three Surfaces" — TUI Is Now Gone

**File:** `README.md:9`
**Issue:** Line 9 of the README (the live product description visible to users on GitHub, not historical release notes) states: "Remote sessions on other tailnet machines are discoverable from all three surfaces." After Phase 136, there are only two surfaces (GUI and CLI). This is a factually incorrect product claim in the active section of the README that a user or prospective installer will read today.

The historical release notes (v3.3–v3.5 entries) also contain TUI references (lines 27, 32, 45, 50, 51, 54, 60, 66), but those are immutable changelog records and are acceptable as-is. Line 9 is the live description.

**Fix:**
```diff
-Remote sessions on other tailnet machines are discoverable from all three surfaces.
+Remote sessions on other tailnet machines are discoverable from both surfaces.
```
Or, if the intent is to enumerate: "...discoverable from the GUI and CLI."

---

### WR-03: Live README Architecture Section Describes "Bubble Tea TUI" as Current Feature (v3.4 Entry Active Text)

**File:** `README.md:45`
**Issue:** The v3.4 release entry headline on line 45 reads "**v3.4 — File Browser (Read-Only) + TUI Parity**" and the body at lines 50–51 describes the Bubble Tea TUI as a working feature across all three surfaces. While this is in the "Latest Release" section (which is historical), the entry is not clearly dated-past — it appears inline with v3.5.1 and v3.5 release notes without a deprecation notice. Any reader scanning the release notes will encounter "full Files view in the Bubble Tea TUI" (line 50) as an apparently-current claim.

More critically, the v3.5 entry on line 27 reads: "...across desktop, web-share, and TUI" and line 32 lists "TUI write parity" as a delivered feature bullet — presented as current, not archived.

The combined effect: the README currently advertises the TUI as a shipping feature in three places in the release notes section, contradicting the actual binary behavior (`agenthub tui` exits 1 with "unknown command").

**Fix:** Add a deprecation/removal notice above the v3.5 entry in the Latest Release section, or add a parenthetical "(TUI removed in v4.0)" to affected lines. At minimum the v3.5 and v3.4 TUI bullets should carry a note that the TUI surface was retired in Phase 136 / v4.0. Example:

```markdown
> **Note (v4.0):** The TUI surface was removed in Phase 136. The features listed below that reference "TUI" are historical; `agenthub tui` now exits with "unknown command".
```

---

## Info

### IN-01: Stale Doc Comment in `client.go` — `IssueCapabilities` Still Names TUI as a Caller

**File:** `internal/daemon/client.go:333`
**Issue:** The `IssueCapabilities` function comment reads: "Called by the GUI/CLI/TUI after toggle-on." The TUI surface no longer exists. This is a cosmetic stale comment — the function itself is correct.

**Fix:**
```go
// IssueCapabilities mints the read + read,write capability pair for a
// web-enabled session (D-07). Returns the URLs and single-use join codes
// (D-09) for each. Called by the GUI and CLI after toggle-on.
```

---

### IN-02: `canonicalStatResponse` Referenced Before Its Definition in Test Helper File

**File:** `internal/daemon/remote_files_test_helpers_test.go:63` (defined at line 104)
**Issue:** `newFixtureRemotePeer` calls `canonicalStatResponse()` at line 63, but the function is defined at line 104, below its call site in the same file. Go resolves package-level declarations regardless of order, so this is not a bug — the code compiles and tests pass. However, the established pattern in the same file places `canonicalListResponse` (line 29) before its use (line 62). The inconsistency is a minor readability issue in a file that reviewers will consult when debugging test failures.

**Fix:** Move `canonicalStatResponse` definition (lines 104–111) to immediately after `canonicalListResponse` (after line 42), before `newFixtureRemotePeer`.

---

### IN-03: `rename` Method in `filesApi.ts` Does Not Include `path` in Query Params (Inconsistency With `buildQuery`)

**File:** `frontend/src/lib/filesApi.ts:282-292`
**Issue:** The `rename` method builds its query string manually (lines 283–285), setting only `session` and optionally `cap`, deliberately omitting `path`. All other write/read methods use `buildQuery(sid, path)` which always sets a `path` param. This inconsistency is intentional (rename doesn't use a path query param — it uses `{oldRel, newRel}` in the JSON body), but the deviation from the `buildQuery` helper and the absence of a comment explaining why makes this look like a bug during code review. The Go client (`client.go:629`) shows the same pattern — `filesURL("rename", sessionID, ".")` passes `"."` as a dummy path. The inconsistency is not a bug, but the lack of explanation is a maintenance hazard.

**Fix:** Add an inline comment in the `rename` method explaining the deviation:
```typescript
// rename passes paths in the JSON body (oldRel/newRel), not as a query
// param — unlike other operations that use buildQuery. Only session and
// cap go in the URL.
const params = new URLSearchParams()
```

---

_Reviewed: 2026-06-19_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
