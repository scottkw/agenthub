# Phase 119: WebServer Routes + `files.read` Capability Plumbing - Context

**Gathered:** 2026-05-20
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

The webserver exposes the same three file endpoints under `requireFilesRead` middleware — capability-gated for Tailscale-HTTPS web-share viewers — and an integration test confirms read-only viewers get 403 with an explicit message, not 404.

**Requirements:** WEB-01, WEB-02, WEB-03, WEB-04, WEB-05

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices at Claude's discretion. Use Phase 118 outputs:
- `files.NewHandler(resolver)` factory — accepts `func(sessionID string) (*Sandbox, error)`
- `capability.PermFilesRead` constant + `HasPerm` whole-token helper
- `requireFilesRead` middleware wrapper body in `internal/webserver/capability_mw.go` (Phase 118 wrote the body; Phase 119 MOUNTS it on webserver routes)
- `internal/daemon/engine.GetSessionWorkDir` for resolver implementation

</decisions>

<code_context>
## Existing Code Insights

- Phase 118 added `requireFilesRead` wrapper in `internal/webserver/capability_mw.go` — Phase 119 mounts it.
- Phase 118 `TestRequireCapability_UnchangedByPhase118` guards separation — Phase 119 should not modify `requireCapability` body either.
- Existing webserver mux registration patterns live in `internal/webserver/server.go`.
- Existing cap token issuance in `internal/webserver/` (or shared with daemon).
- CSP policy: `script-src 'self'` + `style-src 'self' 'unsafe-inline'` + `'wasm-unsafe-eval'`.

</code_context>

<specifics>
## Specific Ideas

Per ROADMAP Phase 119 success criteria:

1. Mount `requireFilesRead`-wrapped routes on webserver: `GET /api/files/list`, `GET /api/files/stat`, `GET /api/files/read`, `HEAD /api/files/read`
2. POST/PUT/DELETE on file routes → 405 (Go 1.22+ method-prefix mux)
3. Missing `?cap=` → 401, not 404 (route exists)
4. Viewer cap without `files.read` → 403 with body containing `"files.read"`
5. Owner cap with `files.read` → 200 from all 4 routes
6. CSP unchanged (no new violations under Playwright Chromium+Firefox+WebKit)
7. Integration tests in `internal/webserver/` testing all 5 success criteria with real cap tokens

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
