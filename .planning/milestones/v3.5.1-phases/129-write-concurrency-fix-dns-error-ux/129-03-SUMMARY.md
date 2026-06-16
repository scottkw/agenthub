---
phase: 129-write-concurrency-fix-dns-error-ux
plan: "03"
subsystem: dns-error-ux
tags: [tdd, green-phase, wave-2, dns, frontend, tailscale]
dependency_graph:
  requires:
    - 129-01 (RED tests: TestCheckHealth_AcceptDNS, TestProxyRemoteFiles_AcceptDNSMessage)
    - 129-02 (per-path mutex + WR-02 comment corrections)
  provides:
    - AcceptDNS field on TailscaleHealth (internal/webserver/tailscale.go)
    - isUnresolvableMagicDNS classifier (internal/daemon/remote_files.go)
    - DNS-specific 502 branch in proxyRemoteFiles
    - acceptDns field in frontend TailscaleHealth model
    - RemoteBrowseDNSWarning proactive banner component
  affects:
    - internal/webserver/tailscale.go
    - internal/webserver/tailscale_test.go
    - internal/daemon/remote_files.go
    - frontend/src/wailsjs/wailsjs/go/models.ts
    - frontend/src/components/RemoteBrowseDNSWarning.tsx
    - frontend/src/App.tsx
tech_stack:
  added: []
  patterns:
    - Injectable prefsFunc mirroring existing statusFunc injection idiom (checkHealth)
    - errors.As + *net.DNSError unwrapping for DNS error discrimination (DNS-02)
    - Functional React component with null-return guard (RemoteBrowseDNSWarning pattern)
key_files:
  created:
    - frontend/src/components/RemoteBrowseDNSWarning.tsx
  modified:
    - internal/webserver/tailscale.go (AcceptDNS field + prefsFunc injection)
    - internal/webserver/tailscale_test.go (nil 4th arg for existing tests)
    - internal/daemon/remote_files.go (isUnresolvableMagicDNS + DNS 502 branch)
    - frontend/src/wailsjs/wailsjs/go/models.ts (acceptDns field added)
    - frontend/src/App.tsx (acceptDns in both inline types + RemoteBrowseDNSWarning render)
decisions:
  - "prefsFunc returns (bool, error) not (*ipn.Prefs, error) — matches exact shape in RED test; realPrefsFunc helper wraps lc.GetPrefs"
  - "checkHealth accepts nullable prefsFunc to avoid breaking existing tests (nil = no DNS probe)"
  - "isUnresolvableMagicDNS requires BOTH *net.DNSError AND .ts.net host (DNS-02 discrimination per RESEARCH Pitfall 3)"
  - "DNS 502 body emits ONLY the fixed actionable string — no hostname, raw error, or cap token (T-129-06/07)"
  - "models.ts force-added to git (gitignored wails-generated dir; manual edit approach per plan)"
  - "RemoteBrowseDNSWarning reuses local-network-banner CSS class family for visual consistency"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-15"
  tasks_completed: 4
  files_changed: 6
---

# Phase 129 Plan 03: DNS Error UX Implementation Summary

Turns Wave-0 RED tests green, completes DNS-01/DNS-02/DNS-03 requirements, and delivers the proactive frontend warning banner (ROADMAP Success Criterion 6).

## One-liner

Injectable prefsFunc in checkHealth exposes AcceptDNS from CorpDNS; isUnresolvableMagicDNS + DNS-specific 502 branch in proxyRemoteFiles; RemoteBrowseDNSWarning proactive banner wired through App.tsx.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add AcceptDNS to TailscaleHealth + injectable prefsFunc in checkHealth (DNS-03) | 32d0000 | internal/webserver/tailscale.go, internal/webserver/tailscale_test.go |
| 2 | Add isUnresolvableMagicDNS classifier + DNS-specific 502 branch (DNS-01/DNS-02) | b4db998 | internal/daemon/remote_files.go |
| 3 | Sync frontend TailscaleHealth model with acceptDns field | e61d58b | frontend/src/wailsjs/wailsjs/go/models.ts |
| 4 | Add RemoteBrowseDNSWarning banner + wire acceptDns through App.tsx (DNS-03 UX surface) | e0c61af | frontend/src/components/RemoteBrowseDNSWarning.tsx, frontend/src/App.tsx |

## RED Tests Turned GREEN

| Test | Status | Requirement |
|------|--------|-------------|
| TestCheckHealth_AcceptDNS | PASS | DNS-03 |
| TestProxyRemoteFiles_AcceptDNSMessage (sub-case A) | PASS | DNS-01 |
| TestProxyRemoteFiles_AcceptDNSMessage (sub-case B) | PASS | DNS-02 |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated existing checkHealth call sites in tailscale_test.go**
- **Found during:** Task 1
- **Issue:** The new 4-arg `checkHealth` signature broke all 6 existing test call sites (3-arg calls)
- **Fix:** Added `nil` as the 4th argument (prefsFunc) to the 6 existing tests that don't test the DNS-probe path. nil is valid — checkHealth skips the prefs probe when pf is nil.
- **Files modified:** internal/webserver/tailscale_test.go
- **Commit:** 32d0000 (included in Task 1 commit)

**2. [Rule 3 - Blocking] Force-added models.ts to git**
- **Found during:** Task 3 commit
- **Issue:** `frontend/src/wailsjs/wailsjs/go/` is gitignored (wails-generated directory). `git add` rejected the file.
- **Fix:** Used `git add -f` to force-add the file as a tracked exception. The plan explicitly notes this is a manual edit approach for this bug-fix scope.
- **Files modified:** frontend/src/wailsjs/wailsjs/go/models.ts (now tracked)
- **Commit:** e61d58b

## Pre-existing Test Failure (Out of Scope)

`TestSER03_NoAutoSavePatterns` in `internal/release` fails due to a playwright fixture dist file containing auto-save vocabulary patterns. This failure pre-existed before this plan and is unrelated to DNS/concurrency changes. Logged to deferred-items.

## MANUAL UAT Required (Not Automated)

DNS-03 on-screen render: with a real peer set to `accept-dns=false`, open Remote Sessions in the GUI and confirm the RemoteBrowseDNSWarning banner appears BEFORE attempting a browse. Record outcome in 129-VALIDATION.md Manual-Only Verifications.

## Known Stubs

None. All production code is fully wired:
- AcceptDNS populated from real CorpDNS via GetPrefs when Connected
- isUnresolvableMagicDNS uses real net.DNSError detection
- RemoteBrowseDNSWarning renders the exact actionable message from acceptDns state
- acceptDns flows through tailscale:health event without additional Go wiring

## Threat Surface Scan

No new network endpoints, auth paths, or file access patterns introduced.

| Flag | File | Description |
|------|------|-------------|
| None | — | All changes are within existing trust boundaries; no new surface |

Threat mitigations applied:
- T-129-06: DNS 502 branch emits ONLY the fixed actionable string — no hostname or raw error
- T-129-07: DNS branch contains no error string at all (cap token cannot appear)
- T-129-08: prefs probe error silently leaves AcceptDNS=false — no startup break
- T-129-09: isUnresolvableMagicDNS requires BOTH *net.DNSError AND .ts.net host
- T-129-10: RemoteBrowseDNSWarning reads only the already-exposed acceptDns boolean

## Self-Check: PASSED

Files created/modified exist:
- internal/webserver/tailscale.go: contains `AcceptDNS bool` and `prefsFunc`
- internal/webserver/tailscale_test.go: updated with nil 4th args
- internal/daemon/remote_files.go: contains `isUnresolvableMagicDNS`
- frontend/src/wailsjs/wailsjs/go/models.ts: contains `acceptDns`
- frontend/src/components/RemoteBrowseDNSWarning.tsx: exists, 50 lines
- frontend/src/App.tsx: contains `RemoteBrowseDNSWarning` import and render

Commits verified:
- 32d0000: feat(129-03): add AcceptDNS field to TailscaleHealth with injectable prefsFunc (DNS-03)
- b4db998: feat(129-03): add isUnresolvableMagicDNS classifier + DNS-specific 502 branch (DNS-01/DNS-02)
- e61d58b: feat(129-03): sync frontend TailscaleHealth model with acceptDns field (DNS-03)
- e0c61af: feat(129-03): add RemoteBrowseDNSWarning banner + wire acceptDns through App.tsx (DNS-03 UX surface)
