---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: 05
type: execute
wave: 3
depends_on: [160-01, 160-02, 160-03, 160-04]
files_modified:
  - internal/relay/sanitize.go
  - TESTING.md
autonomous: true
requirements: [IN-04, NOTIF-02, WR-02]
must_haves:
  truths:
    - "SanitizeChatContent's doc comment accurately states that ESC stripping removes only the 2-byte introducer; DCS/APC/PM/SOS bodies survive as plaintext."
    - "TESTING.md build-script Run Command runs both build-script.test.sh and install-sh.test.sh."
    - "TESTING.md §4 has a traceability row for NOTIF-02 (verified present) and for the new NOTIF-01 hub-card test."
    - "tests/check-traceability-paths.sh passes (all §4 path-column entries resolve to real repo files)."
  artifacts:
    - internal/relay/sanitize.go
    - TESTING.md
  key_links:
    - "TESTING.md §2 vitest count + Total reflect the one new test file (useChatUnreadListeners.test.tsx) added this phase."
    - "TESTING.md §4 NOTIF-01 row path column points at frontend/src/components/Hub/useChatUnreadListeners.test.tsx."
  prohibitions:
    - "MUST NOT change SanitizeChatContent behavior — IN-04 is a doc-comment correction ONLY."
    - "MUST NOT touch SanitizePTYText's comment — it is already accurate (only SanitizeChatContent is misleading)."
    - "MUST NOT add a duplicate NOTIF-02 row — verify the existing row (TESTING.md ~203) and only add if absent at execution time."
    - "MUST NOT put a path-column entry that is not a real .go/.ts/.tsx/.sh repo file (check-traceability-paths.sh will fail)."
---

<objective>
Final v4.1 closeout registration: correct the IN-04 doc comment, register all Phase 160 test changes in TESTING.md, and apply WR-02. This plan is the SOLE owner of TESTING.md for the phase, satisfying the repo standing convention (any phase that adds/renames tests updates TESTING.md §2/§4 and runs check-traceability-paths.sh) in one coherent pass after all test files from 160-01..160-04 exist.

Purpose: Close the remaining doc-accuracy tech debt (IN-04), confirm/record traceability (NOTIF-02), and make the documented build-script gate complete (WR-02).

Output: sanitize.go doc-comment fix (IN-04); TESTING.md updates — WR-02 Run Command cell, vitest/Total count bump + §4 NOTIF-01 row for the new hook test, IN-02 row for the new control-only inject test, NOTIF-02 row verified.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-RESEARCH.md
@.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-PATTERNS.md
@internal/relay/sanitize.go
@TESTING.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Correct the SanitizeChatContent doc comment (IN-04)</name>
  <files>internal/relay/sanitize.go</files>
  <read_first>
    - 160-RESEARCH.md lines 155-172 (IN-04: what actually happens; doc correction only; SanitizePTYText comment is fine, leave it)
    - 160-PATTERNS.md lines 319-337 (exact before/after replacement text for the comment)
    - internal/relay/sanitize.go lines 134-171 (SanitizeChatContent + its doc comment block at ~143-145)
  </read_first>
  <action>
    Replace the SanitizeChatContent doc-comment bullet (around lines 143-145) so it accurately describes the behavior: stripping ESC removes the 2-byte introducer of CSI/OSC/DCS/APC/PM/SOS sequences, but the body bytes (above U+001F) survive as printable plaintext in the output, and that surviving DCS/APC/PM/SOS body text in chat is neutralized downstream by react-markdown + rehype-sanitize before rendering. Use the replacement wording given in 160-PATTERNS.md lines 331-337. Change the comment ONLY — no behavioral change, and do NOT touch SanitizePTYText's comment.
  </action>
  <verify>
    <automated>go build ./internal/relay/... && go vet ./internal/relay/... && grep -q 'rehype-sanitize' internal/relay/sanitize.go && echo OK</automated>
  </verify>
  <acceptance_criteria>
    sanitize.go compiles and vets clean; the corrected SanitizeChatContent comment describes introducer-only stripping + surviving bodies + downstream neutralization; SanitizePTYText comment untouched; no code logic changed.
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Apply WR-02 and register Phase 160 tests in TESTING.md</name>
  <files>TESTING.md</files>
  <read_first>
    - 160-RESEARCH.md lines 174-217 (NOTIF-02 row already present ~203; WR-02 Run Command at ~31)
    - ./CLAUDE.md (AgentHub) "Regression Test Convention" — §2 manifest, §4 traceability (repo-relative path only), run check-traceability-paths.sh
    - TESTING.md line 31 (build-script Suite Manifest row), the vitest manifest row + Total row, §4 traceability map (~94+), line 203 (existing NOTIF-02 row)
  </read_first>
  <action>
    WR-02: update the build-script Suite Manifest Run Command cell (~line 31) to run both scripts: `bash tests/build-script.test.sh && bash tests/install-sh.test.sh`.
    §2 counts: the manifest vitest count is STALE at HEAD (shows 130; true HEAD count is 131 — Phase 158-02 added a file without bumping the header). FIRST re-measure live with `find frontend/src -name '*.test.ts' -o -name '*.test.tsx' | wc -l`, then set the vitest file count and manifest Total to the measured value + 1 (the single NEW test file added this phase: frontend/src/components/Hub/useChatUnreadListeners.test.tsx) — i.e. 132 if HEAD measures 131. Do NOT blindly +1 the stale printed number. Extending existing files (SessionCardGrid/HubPanel/HubInteractiveModal/server_inject_test/install-sh) does NOT change counts.
    §4 traceability: add a NOTIF-01 row whose path column is `frontend/src/components/Hub/useChatUnreadListeners.test.tsx` (describe: Phase 160-01 background unread WS listener accrues per-session unread for backgrounded sessions). Add an IN-02 row whose path column is `internal/relay/server_inject_test.go` (describe: Phase 160-03 control-only inject -> zero PTY writes). Verify the existing NOTIF-02 row (~203) is present; add it only if absent. Path columns must be repo-relative file paths only (no test names).
    Then run the traceability path check.
  </action>
  <verify>
    <automated>bash tests/check-traceability-paths.sh && grep -q 'install-sh.test.sh' TESTING.md && grep -q 'useChatUnreadListeners.test.tsx' TESTING.md && grep -q 'NOTIF-02' TESTING.md && echo OK</automated>
  </verify>
  <acceptance_criteria>
    build-script Run Command runs both test scripts; vitest count + Total set to the live-measured HEAD value + 1 (132 if HEAD measures 131), not a blind +1 on the stale printed number; §4 has NOTIF-01 (useChatUnreadListeners.test.tsx) + IN-02 (server_inject_test.go) rows with repo-relative paths; NOTIF-02 row confirmed present; `bash tests/check-traceability-paths.sh` passes.
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| chat content -> rendered DOM | DCS/APC/PM/SOS body plaintext surviving sanitization is rendered downstream (the IN-04 doc clarifies this) |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-160-IN-04 | Information Disclosure | SanitizeChatContent surviving body bytes | low | accept | Surviving plaintext is neutralized by react-markdown + rehype-sanitize before render; no HTML/script injection. This plan corrects the misleading doc comment so the residual is documented, not hidden. V5 Input Validation. |
</threat_model>

<verification>
- `go build ./internal/relay/...` and `go vet ./internal/relay/...` pass.
- `bash tests/check-traceability-paths.sh` passes.
- TESTING.md build-script Run Command includes install-sh.test.sh; vitest/Total counts bumped by 1; §4 NOTIF-01 + IN-02 rows present; NOTIF-02 row present.
</verification>

<success_criteria>
IN-04 doc accurate, WR-02 documented gate complete, NOTIF-02 traceability confirmed, and all new/changed Phase 160 tests registered with passing path-check — v4.1 chat tech debt fully closed.
</success_criteria>

<output>
Create `.planning/phases/160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c/160-05-SUMMARY.md` when done.
</output>
