# Phase 127: Web-Share Write Security Hardening - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

The web-share write surface has been security-audited end-to-end: symlink escapes return 403, the shell-RC denylist blocks all known bypass vectors, upload abuse is covered, capability escalation is impossible, concurrent-write races leave no partial files, and a Playwright e2e confirms the full web-share write flow with and without the `files.write` cap.

**Depends on:** Phase 124 (capability model + webserver write routes live — COMPLETE) and Phase 125 (browser-facing write surface complete — COMPLETE). Dedicated security audit phase for the most-exposed surface.

**Requirements:** SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06, SEC-07

**Note:** Much of this surface is ALREADY hardened by prior phases — confirm and fill gaps:
- 123: shell-RC denylist (macOS EvalSymlinks fix), FuzzSandboxWrite, atomic write, MaxBytesReader 50 MiB cap.
- 124: requireFilesWrite (HasPerm, no strings.Contains), CSRF Origin check, per-session opt-in.
- 125: If-Match/412 + the TOCTOU re-stat-before-rename fix; canWrite probe via HEAD; cross-browser e2e.
This phase finalizes the fuzz corpus, runs the capability-escalation audit, confirms symlink-escape-on-WRITE returns 403, and documents findings in a committed SECURITY artifact under .planning/.
</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP success criteria)
- SC1: symlink-escape on write/rename → HTTP 403 (os.OpenRoot TOCTOU boundary holds on write path).
- SC2: write/rename/delete of ~/.bashrc, ~/.ssh/authorized_keys, ~/.claude/CLAUDE.md, daemon config dir → 403 Protected system file; denylist not bypassable by case variation, Unicode normalization, or path encoding.
- SC3: FuzzSandboxWrite finalized corpus (rename-dest traversal, denylist-bypass, upload-filename `../` injection) zero crashes; over-cap (>50 MiB) rejected by MaxBytesReader before ParseMultipartForm with clear error (not truncated file).
- SC4: capability escalation audit — no token lacking files.write reaches any write endpoint on any surface (daemon socket, webserver, remote proxy); files.write doesn't leak across sessions; findings in a committed SECURITY artifact.
- SC5: Playwright web-share write e2e — viewer with files.write writes OK; viewer without → 403; CSRF Origin-mismatch on POST/PUT/DELETE → 403.

### Claude's Discretion
Whether to use the gsd-security-auditor pattern; how to structure the SECURITY artifact; which additional fuzz seeds/test cases to add. Confirm existing mitigations hold; only add code where a gap is found.
</decisions>

<code_context>
## Existing Code Insights

Gathered during plan-phase research. Anchors: internal/files/ (denylist, FuzzSandboxWrite, os.OpenRoot, atomic write), internal/webserver/ (requireFilesWrite, originAllowedForWrite, route mounts), internal/daemon/ (remote proxy, cap minting), the Phase 125 e2e (files-write.spec.ts), and the v3.5 milestone PITFALLS.md.

</code_context>

<specifics>
## Specific Ideas

Refer to ROADMAP Phase 127 success criteria (precise + testable). This is an AUDIT-and-HARDEN phase — confirm the boundaries built in 123-125 hold against the full threat corpus, document the capability-escalation audit in a SECURITY artifact, and add a dedicated web-share write security e2e.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
