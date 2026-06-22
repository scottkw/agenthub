# AgentHub — Claude Instructions

## Regression Test Convention (Standing Rule)

`TESTING.md` at the repo root is the canonical regression-suite home. Every future phase that adds, renames, or removes tests must update it:

1. **Automated tests** — add new test files to the appropriate suite group in Section 2 (Suite Manifest). No build tags, no file moves.
2. **Manual checklist** — if the phase introduces a behavior that cannot be automated (native GUI, remote peer, live PTY, physical hardware), add a new M-NN item to Section 5.
3. **Traceability map** — add a row to Section 4 for every new test file that covers a v4.0 (or later milestone) requirement. Path column must be a repo-relative file path only (`.go`, `.ts`, `.tsx`, or `.sh`) — no test names in the path column. Run `bash tests/check-traceability-paths.sh` before committing.
4. **Branch protection** — if `build.yml` matrix entries change, re-apply protection with updated check context names (Section 3 of TESTING.md).

See TESTING.md Section 6 ("Standing Convention") for the full rule.
