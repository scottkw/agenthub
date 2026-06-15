# Requirements: AgentHub — v3.5.1 Remote Browse Completion + Release-Gate Fix

**Defined:** 2026-06-15
**Core Value:** One app to launch, manage, and share AI coding terminal sessions across local and remote access — with zero manual setup for web serving, TLS, or session persistence.

**Milestone Goal:** Close the desktop-GUI remote-browse on-ramp (discover→list→pick a tailnet peer's sessions) and clear the release-gate blocker, retiring umbrella epic #24. The remote read/write *data path* is already proven live; this milestone fixes the GUI on-ramp and the flaky write-concurrency test that gates releases.

**Closes GitHub Issues:** #86 (session enumeration vs cap-gating), #83 (accept-dns=false opaque 502), #87 (WriteFileAtomic If-Match TOCTOU release-gate flake). Retires umbrella epic **#24**.

**Cross-surface parity** remains a release-blocking contract. **Relay-surface coverage** is mandatory: all remote work must exercise the relay loopback the Wails GUI uses (`internal/relay/server_files_test.go`, `internal/daemon/relay_remote_files_test.go`) — the v3.5 audit's 98/100 score was blind to a 4-layer breakage precisely because tests only hit the webserver/fixture surface.

---

## v1 Requirements

### Remote Browse On-Ramp (#86)

The desktop GUI must complete discover→list→pick for a tailnet peer's sessions. The architecture (options a/b/c in #86) is **resolved in a design pass** before the implementation phase; these requirements are written to hold regardless of which option is chosen.

- [ ] **RB-01**: User can see a discovered, reachable tailnet peer's shareable sessions in the Remote Sessions panel — a peer is no longer silently dropped because its `/api/sessions` list isn't enumerable without a session-scoped cap
- [ ] **RB-02**: User can select a remote peer session from the panel and open it in the File Browser, completing discover→list→pick end-to-end over the relay loopback the GUI uses
- [ ] **RB-03**: The chosen approach preserves the Phase 87 no-enumeration security guarantee — an unauthorized / non-tailnet caller still cannot enumerate session content or obtain a cap without the intended grant
- [ ] **RB-04**: Remote panel states are honest — a reachable peer with shareable sessions is never shown as "No remote peers found"; genuinely empty/unreachable peers still surface a correct empty/error state
- [ ] **RB-05**: A relay-surface regression test covers the discover→list→pick path (not just the webserver/fixture surface), guarding against the v3.5-class blind spot

### Tailscale DNS Prerequisite (#83)

- [ ] **DNS-01**: When a remote browse fails because the client has `accept-dns=false` (unresolvable MagicDNS), the user sees an actionable message naming the fix (e.g. *"Enable Tailscale DNS (accept-dns) to browse remote sessions"*) instead of an opaque `502 no such host`
- [ ] **DNS-02**: The daemon distinguishes the unresolvable-MagicDNS / `accept-dns=false` condition from other remote-unreachable failures, so the actionable message is shown only when correct (not as a blanket catch-all)
- [ ] **DNS-03**: `accept-dns` state is probed proactively (at startup or before the first remote browse) and the user is warned before hitting the failure

### Write Concurrency / Release Gate (#87)

The `WriteFileAtomic` If-Match contract under same-process concurrency is **decided in the design pass** — option (a) per-path serialization for a true single-winner guarantee, or (b) documented last-writer-wins with an invariants-only test. These requirements hold either way.

- [ ] **RACE-01**: The release `validate` gate passes deterministically — `TestWrite_TwoWritersIfMatchRace` no longer flakes (outcome is not goroutine-scheduling-dependent)
- [ ] **RACE-02**: The `WriteFileAtomic` If-Match concurrency contract is implemented and documented consistently across the code, comments, and the remote-write proxy — no mismatch between an asserted single-winner guarantee and "last-writer-wins (WR-02)" comments
- [ ] **RACE-03**: After a concurrent-write conflict, the final file content is never interleaved (all-A or all-B) and no leftover `.agenthub-tmp-*` temp files remain — regardless of the chosen contract

## v2 Requirements

Deferred to a future release. Tracked but not in this roadmap.

### TUI File Operations

- **TUIW-UP-01** (#82): TUI Files view supports file upload (parity with GUI/web upload). Formally descoped during v3.5 Phase 126; on-screen message + follow-up issue already in place.

## Out of Scope

| Feature | Reason |
|---------|--------|
| #82 TUI Files upload parity | Deferred to a later milestone; not part of the remote-browse on-ramp cluster |
| Relay remote-route mounting | Already FIXED `58af6d6` (v3.5) — proven live |
| Discovery probe accepting cap-protected peers (#84) | Already FIXED `3508bd7` (v3.5) |
| Auto-enabling `accept-dns` on the client's behalf | AgentHub assists/guides Tailscale setup but does not manage ongoing Tailscale config (existing project boundary) |
| Broad rework of the capability/enumeration model beyond what #86 requires | Scope-limited to making the remote-browse on-ramp usable without weakening the Phase 87/88 security model |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| RB-01 | Phase 130 | Pending |
| RB-02 | Phase 130 | Pending |
| RB-03 | Phase 130 | Pending |
| RB-04 | Phase 130 | Pending |
| RB-05 | Phase 130 | Pending |
| DNS-01 | Phase 129 | Pending |
| DNS-02 | Phase 129 | Pending |
| DNS-03 | Phase 129 | Pending |
| RACE-01 | Phase 129 | Pending |
| RACE-02 | Phase 129 | Pending |
| RACE-03 | Phase 129 | Pending |

**Coverage:**
- v1 requirements: 11 total
- Mapped to phases: 11 (roadmap complete)
- Unmapped: 0 ✓

## Open Design Decisions (resolved before/at implementation phase)

| Decision | Options | Where resolved |
|----------|---------|----------------|
| #86 remote-browse architecture | (a) tailnet-trusted metadata-only discovery endpoint; (b) list peers + join-code/URL per session, stop dropping empty-list peers; (c) keep enumeration locked, reframe as paste-join-code-only | `/gsd:discuss-phase` or `/gsd:plan-phase` for Phase 130 |
| #87 If-Match concurrency contract | (a) per-path lock → true single-winner; (b) accept last-writer-wins + invariants-only test | `/gsd:discuss-phase` or `/gsd:plan-phase` for Phase 129 |

---
*Requirements defined: 2026-06-15*
*Last updated: 2026-06-15 after v3.5.1 roadmap creation — traceability table filled, coverage 11/11*
