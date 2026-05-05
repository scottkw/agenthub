---
phase: 94
plan: 01
status: complete
completed: 2026-05-05
requirements: [SRC-01, SRC-02, SRC-03, SRC-04, SRC-05]
---

# Plan 94-01 Summary — Wave 0 Vendor + RED Scaffold Sweep

## What Was Built

Wave 0 foundation for Phase 94. Two atomic commits:

1. **`feat(94-01): vendor @xterm/addon-search@0.16.0 + extend vendor-drift gate`** — `a57f41d`
2. **`test(94-01): add 13 RED scaffolds + GREEN TestAssets_AddonSearch`** — current HEAD

### Vendoring (Task 1)

| File | Change |
|------|--------|
| `frontend/package.json` | Added `"@xterm/addon-search": "^0.16.0"` |
| `frontend/pnpm-lock.yaml` | Locked addon-search@0.16.0 |
| `web/vendor/xterm/VERSION` | Appended `@xterm/addon-search@0.16.0` (manifest now 6 lines) |
| `web/vendor/xterm/addons/addon-search.js` | New — UMD bundle copy from `frontend/node_modules/@xterm/addon-search/lib/` (78 KB) |
| `web/embed.go` | Appended `vendor/xterm/addons/addon-search.js` to `//go:embed` directive |
| `internal/webserver/vendor_drift_test.go` | Bumped min-count guard 5 → 6 (T-94-01 mitigation) |

### RED Scaffolds (Task 2)

8 TS files / 12 tests, 5 Go files. Every downstream Phase 94 plan has an `<automated>` verify target waiting to turn GREEN.

**TS scaffolds** (`expect.fail()` — RED):
- `FindBar.focus.test.tsx` — SRC-01 Cmd-F focus conditioning → Plan 94-03
- `FindBar.dismiss.test.tsx` — SRC-01 Esc dismiss + xterm focus restore → Plan 94-03
- `FindBar.matchCount.test.tsx` — SRC-02 onDidChangeResults + zero-results → Plan 94-03
- `FindBar.persistence.test.tsx` — SRC-02 SearchConfig round-trip → Plan 94-03
- `FindBar.cancel.test.tsx` — SRC-03 cancel-on-close source-inspection → Plan 94-04
- `FindBar.visual.test.tsx` — SRC-04 200ms slide + selectionBackground → Plan 94-03
- `FindBar.themeMatrix.test.tsx` — SRC-04 138-theme invariant → Plan 94-05
- `isXtermFocused.test.ts` (4 tests) — SRC-01 helper covering Pitfall #1 → Plan 94-03

**Go scaffolds** (`t.Skip()` with reason — visible):
- `internal/daemon/search_config_test.go` — 3 tests for Plan 94-02 (SearchConfig defaults, JSON round-trip, defaults-merge)
- `internal/webserver/plugin_settings_search_sse_test.go` — Plan 94-02 SSE broadcast
- `internal/webserver/findbar_perf_e2e_test.go` (`//go:build e2e`) — Plan 94-04 perf
- `internal/webserver/findbar_web_e2e_test.go` (`//go:build e2e`) — Plan 94-05 web parity
- `internal/webserver/find_bar_test.go` — Plan 94-01 GREEN `TestAssets_AddonSearch` + 2 RED Plan 94-05 scaffolds

## Tests Run / Outcomes

| Test | Status |
|------|--------|
| `go test ./internal/webserver/ -run TestXtermVendorVersionsMatchPnpmLock -count=1` | ✅ PASS (lockfile + VERSION agree on 6 packages) |
| `go test ./internal/webserver/ -run TestAssets_AddonSearch -count=1` | ✅ PASS (asset served at `/assets/xterm/addons/addon-search.js`) |
| `go build ./...` | ✅ exit 0 |
| `pnpm vitest --run src/components/FindBar src/lib/__tests__/isXtermFocused` | RED as expected (8/8 files fail, 12/12 tests fail with explicit `Plan 94-NN must implement` messages) |

## Threat Model Status

| Threat | Severity | Status |
|--------|----------|--------|
| T-94-01 (vendor drift between web bundle and frontend/node_modules) | HIGH | ✅ MITIGATED — `vendor_drift_test.go` min-count guard now requires ≥ 6 @xterm/* packages |
| T-94-05 (web-mode origin/CSP regression) | LOW | ✅ ACCEPTED — vendored under `web/vendor/`, served same-origin under existing v3.1 `script-src 'self'` CSP |
| T-94-06 (RED scaffolds masking real failures) | MEDIUM | ✅ MITIGATED — every scaffold uses explicit `expect.fail()` (TS) or `t.Skip(reason)` (Go) with `Plan 94-NN must implement` marker — never silent skip |

## Key Files Created / Modified

**Created:**
- `web/vendor/xterm/addons/addon-search.js`
- `frontend/src/components/FindBar/__tests__/FindBar.focus.test.tsx`
- `frontend/src/components/FindBar/__tests__/FindBar.dismiss.test.tsx`
- `frontend/src/components/FindBar/__tests__/FindBar.matchCount.test.tsx`
- `frontend/src/components/FindBar/__tests__/FindBar.persistence.test.tsx`
- `frontend/src/components/FindBar/__tests__/FindBar.cancel.test.tsx`
- `frontend/src/components/FindBar/__tests__/FindBar.visual.test.tsx`
- `frontend/src/components/FindBar/__tests__/FindBar.themeMatrix.test.tsx`
- `frontend/src/lib/__tests__/isXtermFocused.test.ts`
- `internal/daemon/search_config_test.go`
- `internal/webserver/plugin_settings_search_sse_test.go`
- `internal/webserver/findbar_perf_e2e_test.go`
- `internal/webserver/findbar_web_e2e_test.go`
- `internal/webserver/find_bar_test.go`

**Modified:**
- `frontend/package.json`
- `frontend/pnpm-lock.yaml`
- `web/vendor/xterm/VERSION`
- `web/embed.go`
- `internal/webserver/vendor_drift_test.go`

## Notes / Deviations

- **Recovery path**: Initial executor agent timed out mid-Task-1 (Stream idle timeout). Partial work was rescued from the orphan worktree to main, Task 1 was committed, and Task 2 was completed inline by the orchestrator. All acceptance criteria met identically; no work redone.
- The orphan worktree at `.claude/worktrees/agent-a48b9ea0b0e5008be` remains locked by a stale agent process (pid 4951); cleanup deferred — does not affect main branch state.
- gopls workspace warnings ("not included in your workspace") emitted on each new Go test file are LSP-config noise, not a real issue — the files compile cleanly under `go test` from the project root.

## Self-Check

- [x] All tasks executed
- [x] Each task committed atomically (Task 1 = `a57f41d`, Task 2 = HEAD)
- [x] SUMMARY.md created
- [x] No modifications to STATE.md or ROADMAP.md (orchestrator owns those)
- [x] T-94-01 (HIGH) mitigated via vendor_drift_test.go min-count bump
- [x] Nyquist Wave 0 gate honored — every later plan has a target test ready to turn GREEN
