# Phase 115 Verification

**Closes:** GitHub Issue #60. Requirements WEB-04, WEB-05, WEB-06.

**Phase commit range:** `f95e61c` (RED) → `1b34dcb` (GREEN). 2 task commits on `main`.

## Automated tests

| Item | Requirement | Command | Status |
|------|-------------|---------|--------|
| handleSession integration suite (6 tests) | WEB-04, WEB-06 | `go test ./internal/relay -run 'TestRelay_handleSession' -count=1 -v` | PASS — all 6 GREEN at `1b34dcb` |
| Full relay suite under race + shuffle | WEB-06 | `go test ./internal/relay -race -count=3 -shuffle=on` | PASS — 7.94 s |
| Full webserver suite (regression check) | WEB-04 | `go test ./internal/webserver -race -count=1` | PASS — 4.22 s (existing 26 unit + 6 integration tests on the moved absorber still green) |
| Build cross-platform | — | `go build ./...` | PASS — no platform-specific code added |

## Manual cross-surface UAT (WEB-05 release gate)

**Operator:** ken@kscott
**Date:** 2026-05-19
**Build:** `wails build -tags wailsassets` against HEAD `1b34dcb` → `build/bin/agenthub.app` (21,412,162 bytes, mtime 2026-05-19 08:10)

| Surface | Probe | Pre-fix (HEAD `624279b`) | Post-fix (HEAD `1b34dcb`) |
|---------|-------|--------------------------|---------------------------|
| Web (Chrome via Playwright) | OSC 11 + DA1 sensitive probe | PASS — `web-osc-probe.png` clean | PASS (unchanged — Phase 111 wrapper still in effect) |
| Desktop (Wails GUI, macOS) | Chafa sixel render | PASS — `desktop-chafa.png` clean | PASS (unchanged — chafa probe alone doesn't trigger visible leak) |
| Desktop (Wails GUI, macOS) | OSC 11 + DA1 sensitive probe | **FAIL** — `uat-evidence/desktop-osc-probe-pre-fix.png` shows `11;rgb:1d1d/1f1f/212162;4;9;22c` typed at next prompt | **PASS** — `uat-evidence/desktop-osc-probe-fixed.png` shows `ZZZ_MARKER` followed by clean `>` prompt |

**Reproduction (post-fix):**

```bash
$ open /Users/ken/dev/agenthub/build/bin/agenthub.app
# Create Shell session via +
$ printf '\033]11;?\033\\'; printf '\033[c'; echo ZZZ_MARKER
ZZZ_MARKER
>                          ← clean — no junk typed at prompt
```

## Static checks

| Check | Status | Notes |
|-------|--------|-------|
| `internal/relay/oscabsorb.go` exists with `type InputAbsorber struct` and `func (a *InputAbsorber) Filter(in []byte) []byte` | PASS | Moved from webserver, package rename only — 99% similarity per `git mv`. |
| `internal/relay/oscabsorb_test.go` defines `TestInputAbsorber*` with 26 named subtests | PASS | Moved with the source file. |
| `internal/relay/oscabsorb_relay_test.go` defines 6 `TestRelay_handleSession_*` integration tests | PASS | New file (195 LoC). |
| `grep -nE 'absorber\s*:=\s*&InputAbsorber\{\}\|filtered\s*:=\s*absorber\.Filter' internal/relay/server.go` shows exactly two matches | PASS | Lines 121 + 136 of the post-fix file. |
| `git diff f95e61c~1..1b34dcb -- internal/relay/server.go \| grep -c '^+'` shows ≤20 net added lines | PASS | 14 added, 1 removed (net +13). |
| `git diff f95e61c~1..1b34dcb -- go.mod go.sum` is empty | PASS | No new dependencies. |
| `internal/webserver/server.go` references `relay.InputAbsorber` (not local `InputAbsorber`) | PASS | Line 741. |

## Sign-off

**WEB-04** ✅ — Desktop OSC/DA1 probe no longer leaks into shell stdin. Verified at production binary built from `1b34dcb`.

**WEB-05** ✅ — Web ↔ desktop full parity achieved. Both surfaces produce only `ZZZ_MARKER` followed by a clean prompt for the OSC 11 + DA1 sensitive probe. Retires the Phase 111 `approved with desktop follow-up: #60` resume signal.

**WEB-06** ✅ — 6 new integration tests in `internal/relay/oscabsorb_relay_test.go` exercise `handleSession` directly. Future regressions on the daemon-direct relay path fail in CI under `-race -count=3 -shuffle=on`.

**Resume signal:** `approved-both-surfaces-clean`.

---

**Date:** 2026-05-19
**Phase commit range:** `f95e61c` → `1b34dcb`
**Tests:** 6 RED → 6 GREEN; full relay suite under race+shuffle PASS; full webserver suite PASS
**Cross-surface manual UAT:** macOS desktop PASS (screenshot in `uat-evidence/desktop-osc-probe-fixed.png`)
**Closes:** Issue #60 + Phase 111 RESEARCH Open Question 1
