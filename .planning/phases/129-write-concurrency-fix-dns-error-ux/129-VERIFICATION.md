---
phase: 129-write-concurrency-fix-dns-error-ux
verified: 2026-06-15T00:00:00Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
human_verification_resolved: 2026-06-15
human_verification:
  - test: "With a real tailnet peer configured with accept-dns=false, open the desktop GUI"
    expected: "The RemoteBrowseDNSWarning banner renders on screen — showing 'Enable Tailscale DNS (accept-dns) to browse remote sessions' — BEFORE attempting any remote browse"
    why_human: "Live tailnet with accept-dns=false peer required; no live daemon available in automated test environment. DNS-03 on-screen render is explicitly declared manual-only in 129-VALIDATION.md."
    result: "PASS (2026-06-15, live UAT). Initially failed to render — UAT uncovered that RemoteBrowseDNSWarning was inside .banner-stack but its trigger was absent from the stack's mount-gating expression, so the stack stayed unmounted when no other banner was active. Fixed (App.tsx gating + invariant comment). Re-verified: banner shows at accept-dns=false and hides at accept-dns=true."
---

# Phase 129: Write Concurrency Fix + DNS Error UX — Verification Report

**Phase Goal:** The release gate passes deterministically and users see actionable errors when Tailscale DNS prerequisites are missing.
**Verified:** 2026-06-15
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `TestWrite_TwoWritersIfMatchRace` passes 100/100 runs — scheduling-dependent outcomes eliminated | VERIFIED | `go test ./internal/files/ -run TestWrite_TwoWritersIfMatchRace -count=100 -race` → PASS (2.157s); package-level `pathLocks sync.Map` + `perPathLock` in `sandbox.go:92-99` acquired after denylistCheck, before `os.OpenRoot` at lines 292-295 with `defer mu.Unlock()` |
| 2 | Code, comments, and proxy all agree on single-winner guarantee — no mismatch | VERIFIED | `grep -c "last-writer-wins" internal/daemon/remote_files.go` → 0; lines 166 and 245 assert single-winner; WriteFileAtomic doc comment at line 264 states "TRUE single-winner guarantee — not last-writer-wins" |
| 3 | Final file content is coherent (all-A or all-B, never interleaved); no `.agenthub-tmp-*` leftovers | VERIFIED | `TestWrite_TwoWritersIfMatchRace` (100/100) and `TestRemoteFiles_TwoWriterRace_RelaySurface` both PASS — both assert exactly-one-200/exactly-one-412 and no leftover temp files |
| 4 | accept-dns=false failure produces actionable "Enable Tailscale DNS (accept-dns)..." message, not opaque 502 | VERIFIED | `TestProxyRemoteFiles_AcceptDNSMessage` sub-case A PASS; `remote_files.go:261` emits exactly "Enable Tailscale DNS (accept-dns) to browse remote sessions" via `isUnresolvableMagicDNS` |
| 5 | Actionable DNS message appears only for MagicDNS failures, not for connection-refused/TLS | VERIFIED | `TestProxyRemoteFiles_AcceptDNSMessage` sub-case B PASS; `isUnresolvableMagicDNS` requires BOTH `*net.DNSError` AND `.ts.net`/`.tailscale.net` host — connection-refused takes the generic "remote unreachable" path |
| 6 | accept-dns probed proactively; user warned before hitting failure path | VERIFIED (automated); HUMAN for on-screen render | `TestCheckHealth_AcceptDNS` (3 table cases) PASS; `TailscaleHealth.AcceptDNS` field exists; `RemoteBrowseDNSWarning.tsx` renders only when `connected===true && acceptDns===false`; wired in `App.tsx:1149-1152`; `tsc --noEmit` clean. Live banner render requires live tailnet (manual UAT) |

**Score:** 6/6 truths verified (automated); 1 sub-truth requires human validation (SC6 on-screen render)

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/files/sandbox.go` | Package-level `pathLocks sync.Map` + `perPathLock` helper; lock held across stat→rename window | VERIFIED | Lines 92-99: `var pathLocks sync.Map`; `perPathLock` uses `LoadOrStore`; lock acquired at line 293 after `denylistCheck`, before `os.OpenRoot` (line 297); `defer mu.Unlock()` at line 295 |
| `internal/daemon/remote_files.go` | `isUnresolvableMagicDNS` classifier + DNS-specific 502 branch; zero "last-writer-wins" | VERIFIED | `isUnresolvableMagicDNS` at lines 298-306; DNS branch at lines 258-263; `grep -c "last-writer-wins"` → 0 |
| `internal/webserver/tailscale.go` | `AcceptDNS bool` field with json tag `acceptDns`; injectable `prefsFunc`; populated from CorpDNS | VERIFIED | Line 25: `AcceptDNS bool \`json:"acceptDns"\``; `prefsFunc` type at line 34; `checkHealth` accepts `pf prefsFunc`; `realPrefsFunc` wraps `lc.GetPrefs` |
| `frontend/src/wailsjs/wailsjs/go/models.ts` | `acceptDns: boolean` on TailscaleHealth class | VERIFIED | Line 282: `acceptDns: boolean;`; constructor assigns it at line 298 |
| `frontend/src/components/RemoteBrowseDNSWarning.tsx` | Functional component; renders only when `connected===true && acceptDns===false`; exact message wording; min 20 lines | VERIFIED | 50 lines; conditional guard at line 30; renders exact string "Enable Tailscale DNS (accept-dns) to browse remote sessions" at line 41; `role="status"` at line 37 |
| `frontend/src/App.tsx` | Both inline types include `acceptDns?`; component imported and rendered in banner region | VERIFIED | `acceptDns?: boolean` at lines 148 and 518; import at line 55; rendered at lines 1149-1152 with `connected={!!(tailscaleHealth?.connected)}` and `acceptDns={tailscaleHealth?.acceptDns}` |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `sandbox.go WriteFileAtomic` | package-level `pathLocks sync.Map` keyed on absolute path | `perPathLock(lockKey).Lock(); defer Unlock()` across stat-check→rename | WIRED | Lock key is `filepath.Join(s.rootPath, cleaned)` at line 292; acquired before `os.OpenRoot`; deferred unlock covers all error paths |
| `remote_files.go proxyRemoteFiles` client.Do error path | `isUnresolvableMagicDNS(err, baseURL)` → DNS-specific 502 | `errors.As(err, &dnsErr)` + `.ts.net` check | WIRED | Lines 258-265; DNS branch has no information disclosure; non-DNS path falls through to generic `redactCapTokenFromError` |
| `tailscale.go checkHealth` | `TailscaleHealth.AcceptDNS` = `prefs.CorpDNS` | injectable `prefsFunc` called when `Connected` | WIRED | Lines 74-77; called only when `h.Connected`; prefs error silently leaves AcceptDNS=false (graceful degradation) |
| `App.tsx tailscaleHealth` state | `RemoteBrowseDNSWarning` banner | `tailscaleHealth.acceptDns` prop via `tailscale:health` event | WIRED | Both inline type literals include `acceptDns?`; `setTailscaleHealth(h)` at line 520 passes event payload directly; `acceptDns` flows automatically from health poller |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `RemoteBrowseDNSWarning.tsx` | `acceptDns` prop | `tailscaleHealth.acceptDns` state in App.tsx, populated by `tailscale:health` event from `startHealthPoller` (app.go), which calls `CheckHealth` → `realPrefsFunc` → `lc.GetPrefs` → `prefs.CorpDNS` | Yes — real `GetPrefs` call to live tailscaled | FLOWING |
| `proxyRemoteFiles` DNS 502 branch | `err` from `client.Do(req)` | Real HTTP client performing actual network dial to upstream; `isUnresolvableMagicDNS` unwraps real `*net.DNSError` | Yes — real network error unwrapping | FLOWING |
| `WriteFileAtomic` single-winner guarantee | `pathLocks sync.Map` per-path mutex | Package-level `sync.Map`; `perPathLock` uses `LoadOrStore` to ensure canonical mutex per absolute path | Yes — live mutex state | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Race test passes 100/100 with -race | `go test ./internal/files/ -run TestWrite_TwoWritersIfMatchRace -count=100 -race` | ok (2.157s) | PASS |
| Relay-surface two-writer race test passes | `go test ./internal/daemon/ -run TestRemoteFiles_TwoWriterRace_RelaySurface -v` | PASS (0.04s) | PASS |
| DNS message test — both sub-cases pass | `go test ./internal/daemon/ -run TestProxyRemoteFiles_AcceptDNSMessage -v` | PASS sub-case A (MagicDNS) + PASS sub-case B (non-DNS) | PASS |
| AcceptDNS health check — 3 table cases | `go test ./internal/webserver/ -run TestCheckHealth_AcceptDNS -v` | PASS all 3 cases | PASS |
| Zero "last-writer-wins" in proxy | `grep -c "last-writer-wins" internal/daemon/remote_files.go` | 0 | PASS |
| TypeScript type-check clean | `cd frontend && npx tsc --noEmit` | No output (clean) | PASS |
| All phase-129 packages green | `go test ./internal/files/... ./internal/daemon/... ./internal/webserver/...` | ok (all 3 cached) | PASS |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| RACE-01 | 129-02 | Release gate passes deterministically; TestWrite_TwoWritersIfMatchRace no longer flakes | SATISFIED | Test passes 100/100 -count=100 -race; relay-surface test also passes |
| RACE-02 | 129-02 | WriteFileAtomic concurrency contract consistent — no doc/code mismatch | SATISFIED | `grep -c "last-writer-wins" remote_files.go` = 0; both comment blocks assert single-winner |
| RACE-03 | 129-02 | Final file content never interleaved; no leftover .agenthub-tmp-* files | SATISFIED | Asserted in both TestWrite_TwoWritersIfMatchRace and TestRemoteFiles_TwoWriterRace_RelaySurface; both PASS |
| DNS-01 | 129-03 | accept-dns=false failure shows actionable message | SATISFIED | `isUnresolvableMagicDNS` → exact string "Enable Tailscale DNS (accept-dns) to browse remote sessions"; TestProxyRemoteFiles_AcceptDNSMessage sub-case A PASS |
| DNS-02 | 129-03 | Actionable message only for MagicDNS failures, not blanket catch-all | SATISFIED | Requires BOTH `*net.DNSError` AND `.ts.net`/`.tailscale.net`; TestProxyRemoteFiles_AcceptDNSMessage sub-case B PASS |
| DNS-03 | 129-03 | accept-dns probed proactively; user warned before failure | SATISFIED (automated); HUMAN for on-screen render | TestCheckHealth_AcceptDNS PASS; RemoteBrowseDNSWarning component wired; tsc clean; live render is manual UAT |

All 6 phase-129 requirement IDs accounted for. No orphaned requirements.

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `RemoteBrowseDNSWarning.tsx` | 31 | `return null` | Info | This is the correct conditional guard (`connected===true && acceptDns===false`), NOT a stub. The component renders real content when the condition is met. |

No TBD/FIXME/XXX/unresolved debt markers found in phase-129-modified files.

---

### Human Verification Required

#### 1. DNS-03 On-Screen Banner Render (Live Tailnet)

**Test:** Set a Tailscale client to `accept-dns=false` on a peer. Open the AgentHub desktop GUI and navigate to the Remote Sessions panel.
**Expected:** The `RemoteBrowseDNSWarning` banner renders on screen — displaying "Enable Tailscale DNS (accept-dns) to browse remote sessions" — BEFORE any remote browse attempt is made.
**Why human:** Requires a live tailnet with a real peer configured with `accept-dns=false`. No live daemon available in automated test environment. This item is explicitly designated manual-only in `129-VALIDATION.md` Manual-Only Verifications.

---

### Gaps Summary

No automated gaps found. All 6 ROADMAP success criteria are verified by automated tests. The single human verification item (DNS-03 on-screen banner render with a live `accept-dns=false` peer) is the only remaining item before full phase sign-off.

**Pre-existing environment note:** `go test ./internal/release/ -run TestSER03_NoAutoSavePatterns` fails due to an untracked, gitignored stray build artifact at `cmd/playwright-fixture/dist/assets/index-Dklc5ak1.js` left by a prior dev-browser session. This failure is NOT caused by any phase 129 commit, does not exist in CI, and does not appear in any phase-129-modified package. It is environmental contamination, not a phase 129 finding.

---

_Verified: 2026-06-15_
_Verifier: Claude (gsd-verifier)_
