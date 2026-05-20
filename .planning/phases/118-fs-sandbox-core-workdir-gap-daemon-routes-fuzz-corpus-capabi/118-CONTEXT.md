# Phase 118: FS Sandbox Core + WorkDir Gap + Daemon Routes + Fuzz Corpus + Capability Bit - Context

**Gathered:** 2026-05-20
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

The `internal/files/` package exists, is TOCTOU-safe, is fuzz-proven, and daemon-local HTTP routes for list/stat/read are live — so every subsequent phase has a correct, trusted API to build against.

**Requirements:** FS-01, FS-02, FS-03, FS-04, FS-05, FS-06, FS-07, FS-08, FS-09, FS-10, FS-11, FS-12, FS-13, FS-14

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, REQUIREMENTS.md, research docs in `.planning/research/` (STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md, SUMMARY.md), and codebase conventions to guide decisions.

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research. Reference research docs in `.planning/research/` (especially PITFALLS.md for the 40+ fuzz payload corpus, ARCHITECTURE.md for daemon route patterns).

</code_context>

<specifics>
## Specific Ideas

Per ROADMAP Phase 118 success criteria:

1. `internal/files/` package with TOCTOU-safe `*os.Root` (Go 1.24+) sandbox
2. `FuzzSandboxPath` fuzz test covering 40+ payloads from PITFALLS.md (path traversal, Unicode tricks, Windows reserved names, NTFS streams, UNC, etc.)
3. Daemon-local HTTP routes: `/api/files/list`, `/api/files/stat`, `/api/files/read` on the Unix socket
4. `FileEntry` JSON shape: Name/Size/Mtime/Mode/IsDir/IsSymlink/IsBinary/MIME
5. 0-byte file Range read returns 200 with empty body (golang/go#54794)
6. `HasPerm` whole-token match (comma-split, not substring) — `files.read` capability bit added
7. WorkDir gap fix (session cwd tracked correctly)

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
