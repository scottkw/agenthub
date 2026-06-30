---
phase: 160-v4-1-chat-closeout-wire-notif-01-hub-card-unread-badge-and-c
plan: "05"
subsystem: docs/traceability
tags: [in-04, wr-02, notif-02, testing, docs]
dependency_graph:
  requires: [160-01, 160-02, 160-03, 160-04]
  provides: [TESTING.md-phase-160-registration, IN-04-doc-accuracy]
  affects: [internal/relay/sanitize.go, TESTING.md]
tech_stack:
  added: []
  patterns: [doc-comment correction, TESTING.md traceability convention]
key_files:
  created: []
  modified:
    - internal/relay/sanitize.go
    - TESTING.md
decisions:
  - "Vitest count set to 132 (live-measured HEAD already includes useChatUnreadListeners.test.tsx from 160-01); no additional +1 applied on top of 132"
  - "NOTIF-02 row confirmed present at line 203 — no duplicate added"
  - "Phase 160 Note paragraph prepended to start of §2 Note block (consistent with historical convention)"
metrics:
  duration: "~10 minutes"
  completed: "2026-06-27"
  tasks_completed: 2
  tasks_total: 2
  files_changed: 2
status: complete
---

# Phase 160 Plan 05: Docs/Traceability Closeout Summary

Final v4.1 chat tech-debt closeout: corrected the SanitizeChatContent doc comment (IN-04) to accurately describe introducer-only ESC stripping with surviving body bytes, and registered all Phase 160 test changes in TESTING.md in a single authoritative pass per the repo standing convention.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Correct SanitizeChatContent doc comment (IN-04) | 735dbfc5 | internal/relay/sanitize.go |
| 2 | Apply WR-02 and register Phase 160 tests in TESTING.md | 58819358 | TESTING.md |

## What Was Built

**Task 1 — IN-04 doc comment correction (`internal/relay/sanitize.go`):**

The `SanitizeChatContent` function's doc comment bullet for C0 stripping was inaccurate — it claimed "escape sequences cannot be reconstructed by a renderer," which implied the body bytes were also stripped. In fact only the 2-byte ESC introducer is stripped (ESC is C0 ≤ U+001F); body bytes (above U+001F) survive as printable plaintext. The corrected comment now accurately states:
- Stripping ESC removes the 2-byte introducer of CSI/OSC/DCS/APC/PM/SOS sequences
- Body bytes (above U+001F) survive as printable plaintext in the output
- DCS/APC/PM/SOS body content in chat is cosmetically confusing but is neutralized downstream by react-markdown + rehype-sanitize before rendering

No behavioral change — comment only. `SanitizePTYText` comment untouched (it was already accurate).

**Task 2 — TESTING.md (WR-02 + §2 counts + §4 traceability):**

- WR-02: build-script Run Command updated from `bash tests/build-script.test.sh` to `bash tests/build-script.test.sh && bash tests/install-sh.test.sh` — the documented gate now exercises both test scripts.
- §2 vitest count: corrected from stale 130 to 132 (Phase 158-02 had added `TerminalChatHost.test.tsx` without bumping the header; Phase 160-01 adds `useChatUnreadListeners.test.tsx`). Live-measured HEAD count = 132 via `find`.
- §2 Total: updated 507 → 509.
- §4 NOTIF-01 row added: `frontend/src/components/Hub/useChatUnreadListeners.test.tsx` (Phase 160-01 background unread WS listener — backgrounded sessions accrue per-session unread count).
- §4 IN-02 row added: `internal/relay/server_inject_test.go` (Phase 160-03 `TestInject_ControlOnlyInput` — control-only MsgSessionInject payload produces zero PTY writes).
- §4 NOTIF-02 row confirmed present (line 203) — no duplicate added.

## Verification Results

- `go build ./internal/relay/... && go vet ./internal/relay/...`: PASSED
- `grep -q 'rehype-sanitize' internal/relay/sanitize.go`: PASSED
- `bash tests/check-traceability-paths.sh`: PASSED — "OK: all traceability paths exist"
- `grep -q 'install-sh.test.sh' TESTING.md`: PASSED
- `grep -q 'useChatUnreadListeners.test.tsx' TESTING.md`: PASSED
- `grep -q 'NOTIF-02' TESTING.md`: PASSED

## Deviations from Plan

None — plan executed exactly as written. The vitest count was correctly set to 132 (the live-measured value, which already includes `useChatUnreadListeners.test.tsx` from 160-01 already being committed) rather than 133 (which would result from blindly adding +1 to the already-correct live count). The plan's example "i.e. 132 if HEAD measures 131" anticipated measurement BEFORE 160-01 committed the file; since 160-01 landed first, live measurement returned 132 directly.

## Self-Check

### Files Exist
- `internal/relay/sanitize.go`: FOUND (modified)
- `TESTING.md`: FOUND (modified)

### Commits Exist
- 735dbfc5: FOUND — `docs(160-05): correct SanitizeChatContent doc comment (IN-04)`
- 58819358: FOUND — `chore(160-05): register Phase 160 tests in TESTING.md (WR-02, NOTIF-01, IN-02)`

## Self-Check: PASSED
