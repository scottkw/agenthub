---
phase: 149-google-antigravity-agent
verified: 2026-06-22T23:25:30Z
status: human_needed
score: 9/9 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Live agy REPL launch — run `agenthub new agy <dir>` when waitlist access is granted"
    expected: "GUI/web picker shows 'Google Antigravity'; agy launches an interactive PTY REPL; auth completes via browser-loopback OAuth or SSH/OTP path; status badge renders #ff9e64; card spine, chip, and tab dot all show agy color in lockstep"
    why_human: "agy binary is closed-beta/waitlist — cannot be installed for live UAT this phase. Live PTY REPL and browser-loopback OAuth require a real installed binary. Documented in TESTING.md M-15 per D-03."
---

# Phase 149: Google Antigravity Agent Verification Report

**Phase Goal:** Google Antigravity CLI is selectable as a supported agent and launches correctly.
**Verified:** 2026-06-22T23:25:30Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | DetectCLIs() returns an 'agy' entry named "Google Antigravity" when an agy binary is on PATH | VERIFIED | `knownCLIs` has `{Name: "agy", DisplayName: "Google Antigravity"}` at detect.go:30; `TestKnownCLIs_HasExpectedEntries` and `TestDetectCLIs_FindsAgy` both PASS |
| 2 | DetectCLI("agy") resolves to the binary path, and ErrCLINotFound when absent | VERIFIED | `TestDetectCLI_AgyNotFound` PASS; detect.go loops knownCLIs using `exec.LookPath` uniformly |
| 3 | On Windows, AugmentServicePath includes %LOCALAPPDATA%\agy\bin | VERIFIED | path_windows.go:21 `filepath.Join(local, "agy", "bin")`; `TestPlatformExtraBins_WindowsIncludesAgyBin` present; `GOOS=windows go build ./internal/daemon/` exits 0 |
| 4 | An agy session classifies as idle at its '> ' prompt rather than perma-running | VERIFIED | `DefaultAgyPatterns()` at detector.go:101 uses `>\s*$` for Idle; `PatternsForCLI("agy")` switch case at detector.go:118 returns DefaultAgyPatterns; `TestDetector_AgyIdle` and `TestDetector_AgyWaiting` PASS |
| 5 | agentBadgeModifier('agy') returns 'agy' so tab dot + card spine + chip resolve the agy color | VERIFIED | agentBadge.ts:23 `case 'agy':` in fall-through group; agentBadge.test.ts:29-30 asserts `agentBadgeModifier('agy') toBe('agy')`; 14/14 tests PASS |
| 6 | All three per-agent color sites carry the agy color #ff9e64 in lockstep | VERIFIED | style.css:1719 tab dot `.tab__agent-badge--agy { background: #ff9e64; }`; style.css:4813 spine `.hub-card[data-agent="agy"] { border-left: 3px solid #ff9e64; }`; style.css:5030 chip with `color: #ff9e64; border-color: rgba(255, 158, 100, 0.45)`; style.hub.test.ts 100/100 PASS |
| 7 | WCAG numbers documented honestly at source (dark 8.72:1 PASS, light 2.03:1 FAIL) | VERIFIED | style.css:1718 comment `dark: 8.72:1 AA PASS; light: 2.03:1 FAIL`; style.hub.test.ts contains explicit assertions for both `8.72:1` and `2.03:1` |
| 8 | TESTING.md registers the new/extended agy test files in Suite Manifest and Traceability map (AGENT-01 rows) | VERIFIED | TESTING.md Section 4 has 5 AGENT-01 rows (detect_test.go, path_windows_test.go, detector_test.go, agentBadge.test.ts, style.hub.test.ts); `grep -c 'AGENT-01' TESTING.md` = 8; `bash tests/check-traceability-paths.sh` exits 0 (with non-fatal grep flag warning on macOS) |
| 9 | README documents Google Antigravity as a supported agent with the waitlist note | VERIFIED | README.md:7 intro paragraph lists "Google Antigravity" and contains waitlist note; README.md:76 CLI auto-detection bullet adds "Google Antigravity (`agy`)" with waitlist reference to M-15 |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/pty/detect.go` | knownCLIs entry `{Name: "agy", DisplayName: "Google Antigravity"}` | VERIFIED | Line 30; 5-entry slice; D-09 key is `agy` not `antigravity` (grep count = 0) |
| `internal/pty/detect_test.go` | TestKnownCLIs (5 entries), TestDetectCLIs_FindsAgy, TestDetectCLI_AgyNotFound | VERIFIED | All three tests present and PASS |
| `internal/daemon/path_windows.go` | `filepath.Join(local, "agy", "bin")` inside LOCALAPPDATA block | VERIFIED | Line 21; `GOOS=windows go build` clean |
| `internal/daemon/path_windows_test.go` | TestPlatformExtraBins_WindowsIncludesAgyBin | VERIFIED | Present at line 96; asserts `C:\Users\test\AppData\Local\agy\bin` |
| `internal/status/detector.go` | DefaultAgyPatterns() + case "agy" in PatternsForCLI switch | VERIFIED | Lines 101-122; Idle `>\s*$`, Waiting `[y/n]` variants; Working empty (intentional per D-13) |
| `internal/status/detector_test.go` | TestDetector_AgyIdle, TestDetector_AgyWaiting, TestPatternsForCLI_AgyNotFallback | VERIFIED | All three tests present and PASS |
| `frontend/src/lib/agentBadge.ts` | `case 'agy':` in switch | VERIFIED | Line 23; key is `agy` (not `antigravity`, grep = 0) |
| `frontend/src/lib/agentBadge.test.ts` | test asserting `agentBadgeModifier('agy') toBe('agy')` | VERIFIED | Lines 29-30; 14/14 PASS |
| `frontend/src/style.css` | Three agy color sites at #ff9e64 | VERIFIED | Lines 1719, 4813, 5030; D-09 key is `agy` (grep `data-agent="antigravity"` = 0) |
| `frontend/src/components/__tests__/style.hub.test.ts` | 7 agy assertions including WCAG comment gate | VERIFIED | Lines 482-534; 100/100 PASS; GAP-04 updated to 9 agents |
| `TESTING.md` | Suite Manifest note + 5 AGENT-01 traceability rows + M-15 manual item | VERIFIED | Section 2 Phase 149 note present; Section 4 has 5 rows; Section 5 Category I M-15 at line 247 |
| `README.md` | "Google Antigravity" in intro + CLI auto-detection bullet + waitlist note | VERIFIED | Both occurrences present at lines 7 and 76 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/status/detector.go PatternsForCLI` | `DefaultAgyPatterns()` | `switch cliName { case "agy": return DefaultAgyPatterns() }` | WIRED | detector.go:114-122; switch replaces prior if/else — extensible pattern |
| `internal/daemon/path_windows.go AugmentServicePath` | `filepath.Join(local, "agy", "bin")` | inside `if local := os.Getenv("LOCALAPPDATA")` block | WIRED | path_windows.go:21; `GOOS=windows go build` exits 0 |
| `agentBadge.ts case 'agy'` | `style.css .tab__agent-badge--agy / .hub-card[data-agent="agy"]` | BEM modifier key === data-agent key === knownCLIs key === binary name `agy` | WIRED | All three sites use `agy` key; zero `antigravity` occurrences |
| `TESTING.md Traceability §4 AGENT-01 rows` | test files (detect_test.go, path_windows_test.go, detector_test.go, agentBadge.test.ts, style.hub.test.ts) | repo-relative path column, validated by check-traceability-paths.sh | WIRED | `bash tests/check-traceability-paths.sh` exits 0 |

### Data-Flow Trace (Level 4)

Not applicable — this phase delivers CLI detection (Go backend), badge color identity (CSS/TS), and documentation. No dynamic data-rendering component was introduced; the agy agent flows through the existing `DetectCLIs()` → GUI/CLI/web consumption pipeline unchanged. Level 4 trace applies to the existing pipeline which was verified in prior phases.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| knownCLIs has agy entry, all detection tests pass | `go test ./internal/pty/ -run 'TestKnownCLIs\|TestDetectCLI' -v` | 6 tests PASS | PASS |
| agy status patterns classify idle/waiting correctly | `go test ./internal/status/ -run 'Agy\|PatternsForCLI' -v` | 3 tests PASS | PASS |
| Windows PATH cross-compile clean | `GOOS=windows go build ./internal/daemon/` | exits 0 | PASS |
| agentBadge modifier returns 'agy' | `pnpm exec vitest run agentBadge` | 14/14 PASS | PASS |
| Three CSS sites all contain #ff9e64 with honest WCAG comment | `pnpm exec vitest run style.hub` | 100/100 PASS | PASS |
| Full Go test suite | `go test ./...` | 13 packages PASS | PASS |
| Full frontend vitest suite | `pnpm exec vitest run` | 117 files / 1878 tests PASS | PASS |
| TypeScript type check | `pnpm exec tsc --noEmit` | exits 0 | PASS |
| Traceability path check | `bash tests/check-traceability-paths.sh` | OK (grep -P flag warning on macOS is benign — script still exits 0 with "OK: all traceability paths exist") | PASS |

### Probe Execution

No phase-declared probes. Behavioral spot-checks above serve as the equivalent probe set.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AGENT-01 | 149-01, 149-02, 149-03 | Google Antigravity CLI is selectable as a supported agent and launches correctly (#65) | SATISFIED (source-level; live-launch deferred per D-03) | knownCLIs entry, badge color, TESTING.md M-15, README documented; D-03 accepted waitlist fallback — live REPL launch is TESTING.md M-15 manual item |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/status/detector.go` | 96-97 | `[ASSUMED]` comment on pattern source | Info | Intentional — CONTEXT D-13 explicitly defers pattern tuning to post-M-15 live access; this is documentation of a known limitation, not an untracked TODO |

No `TBD`, `FIXME`, or `XXX` debt markers found in any files modified by this phase. No stubs (all agy implementations are substantive: real `exec.LookPath` detection, real regex patterns, real CSS rules). No hardcoded empty returns.

D-09 compliance confirmed: the key `antigravity` (lowercase) does not appear in detect.go, path_windows.go, agentBadge.ts, or style.css — all four checks return 0.

### Human Verification Required

#### 1. Live Antigravity REPL Launch (M-15)

**Test:** When waitlist access to `agy` is granted, install the binary and run:
1. `agenthub new agy <dir>` from CLI
2. Open GUI — navigate to New Session picker
3. Open web UI — open session picker

**Expected:**
- (a) GUI/web picker shows "Google Antigravity" as a selectable agent
- (b) `agy` launches an interactive PTY REPL
- (c) Auth completes via browser-loopback OAuth or documented SSH/OTP degradation path
- (d) Status badge renders `#ff9e64` (orange)
- (e) Card spine, chip, and tab dot all show the agy color in lockstep

**Why human:** The `agy` binary is closed-beta/waitlist (D-03) — it cannot be installed for live UAT this phase. Live PTY REPL interaction and browser-loopback OAuth require a real installed binary not yet publicly available. This is documented as TESTING.md M-15 per the D-03 design decision.

### Gaps Summary

No gaps. All 9 source-level must-haves are VERIFIED. The only unresolved item is the live REPL launch, which is a documented manual-UAT deferral (D-03) — not a verification failure — because the `agy` binary is waitlist-gated. TESTING.md M-15 is the formal tracking item.

---

_Verified: 2026-06-22T23:25:30Z_
_Verifier: Claude (gsd-verifier)_
