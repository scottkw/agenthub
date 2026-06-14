# Phase 124: files.write Capability + Webserver Write Routes + Web-Share Opt-In - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

The `files.write` capability bit exists, `requireFilesWrite` middleware (with CSRF Origin check) gates all five webserver write routes, `files.write` is opt-in for every token (a per-session "Enable file writes" toggle gates the owner cap; web-share viewers require a further explicit opt-in), and `schemaVersion: 4` migration is in place — so any surface that authenticates via the webserver can exercise write operations only after writes are explicitly enabled.

**Depends on:** Phase 123 (write sandbox primitives and daemon routes frozen — COMPLETE).

**Requirements:** CAP-01, CAP-02, CAP-03, CAP-04, CAP-05, CAP-06, CAP-07, CAP-08, CAP-09, CAP-10

**Cross-surface parity (release-blocking):** SC#4 requires the home-directory write warning in BOTH GUI and TUI when `files.write` is active for a `$HOME`-cwd session.
</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP + recent decisions)
- `files.write` is OPT-IN for ALL tokens (owner and viewer) — no owner default-on. (Confirmed by commit 808767f "make files.write opt-in for all tokens".)
- CSRF protection reuses the Phase 88 `Origin`-check pattern: mismatched Origin → 403; absent Origin (desktop Wails fetch) passes vacuously.
- Permission checks MUST use the `HasPerm` whole-token comma-split helper — never `strings.Contains(perms, "files.write")`.
- Settings migrate `schemaVersion: 3 → 4` with `FilesWrite: false` default.

### Claude's Discretion
Remaining implementation details (middleware wiring, toggle component structure, migration mechanics) at Claude's discretion — use ROADMAP success criteria, Phase 123 patterns, and codebase conventions.

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research. Key anchors: Phase 88 Origin-check (CSRF precedent), the v3.4 read-side capability/cap-token model, the `HasPerm` helper, the daemon write routes from Phase 123 (`internal/files/write.go`, `internal/daemon/api.go`), and the web-share grant UI.

</code_context>

<specifics>
## Specific Ideas

Refer to ROADMAP Phase 124 success criteria — they are precise and testable (403-without-write/2xx-with-write on all five routes; Origin-check 403; static-grep HasPerm gate; viewer opt-in toggle includes "files.write" in cap; schemaVersion 4 migration test).

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
