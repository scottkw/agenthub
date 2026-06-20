---
phase: 137
slug: share-modal-cap-model
status: verified
threats_open: 0
asvs_level: 1
created: 2026-06-20
---

# SECURITY.md — Phase 137: share-modal-cap-model

**Audit date:** 2026-06-20
**Audit type:** State B (register authored at plan time; verify mitigations in implemented code)
**ASVS Level:** L1 (default)
**Block on:** default (open)
**Result:** SECURED — 11/11 threats resolved (9 mitigate CLOSED, 2 accept CLOSED)

Implementation files were not modified. Each `mitigate` threat was verified by locating
the mitigation in the cited source file (file:line) and, where a pinning test exists,
running it green. Each `accept` threat was verified by confirming the accept rationale
still holds against the implemented code.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| owner GUI → daemon socket | Browse toggle + IssueCapabilities cross here; loopback-trusted (daemon binds 127.0.0.1), no auth gate | session id, browse-enable flag, cap-perm set |
| web visitor → webserver routes | Untrusted; presents a cap token; perms enforced by requireFilesRead/requireFilesWrite | cap token (frozen perms), file read/write requests |
| issuance time vs enforcement time | Perms frozen into the token at issuance (api.go); enforcement reads token content only — an old RO token issued pre-browse carries only "read" and correctly 403s on file routes | cap token perm string |
| card ownership check | isLocal gates the Share affordance; remote peers cannot reach the share path (UI gate reinforcing backend ownership truth) | session ownership flag |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-137-01 | Elevation of Privilege | issueCapabilitiesForSession (browse w/o consent) | mitigate | `browseEnabledForUnlocked` returns `e.sessionBrowse[sessionID]` = false for absent key (default OFF, D-06) — internal/daemon/engine.go:608-610; map init engine.go:257; pinned by `TestIssueCapabilities_BrowseOff_NoFilesPerms` (PASS) | closed |
| T-137-01b | Elevation of Privilege | stale RO URL after browse toggle | mitigate | Browse toggle calls `SetSessionBrowse` then `IssueCapabilities` when sharing ON — SessionShareModal.tsx:209,214 (Pitfall 1); backend also clears grants — api.go:1303 | closed |
| T-137-02 | Elevation of Privilege | RO token perm injection | mitigate | RO perms set to exactly `"read," + capability.PermFilesRead`; files.write NEVER added to rPerms — api.go:1116-1122; pinned by `TestIssueCapabilities_BrowseOn_ROPermsExact` (PASS) | closed |
| T-137-03 | Elevation of Privilege | RO holder writing files | mitigate | files.write injected only into wPerms (RW) — api.go:1120; `requireFilesWrite` 403s on missing files.write via whole-token `HasPerm` — capability_mw.go:156; all 5 write routes gated — server.go:587-596; pinned by `TestFilesRoutes_RO_BrowseOn_WriteRoute403` (PASS) | closed |
| T-137-04 | Information Disclosure | CAP-05 two-gate removal in UI | mitigate | CAP-05 two-gate symbols removed from panel — SessionSharePanel.tsx:62-67; panel calls NO toggle APIs (renders only); backend perms remain the authority | closed |
| T-137-05 | Elevation of Privilege | global filesRead removal (D-07) | mitigate | Global `filesReadEnabled()` kill-switch removed; per-session default OFF replaces it — api.go:1099-1115 (Reversal 3 audit comment) + engine.go:90-94; no live `filesReadEnabled`/`e.filesRead` refs remain | closed |
| T-137-06 | Tampering | new perm-injection code | mitigate | `HasPerm` splits on `,` and compares whole tokens with `==` — capability.go:55-60; zero `strings.Contains` in api.go/engine.go; pinned by `TestHasPerm_NoStringsContains_Browse` (PASS) | closed |
| T-137-SHARE06 | Spoofing / EoP | remote peer re-share | mitigate | Share button `disabled={!isLocal}` with LockClosedIcon + tooltip (colorblind-safe, D-13) — SessionCard.tsx:164,420-426; click guarded by stopPropagation + `.hub-card__share` closest() guard — SessionCard.tsx:252,421 | closed |
| stale-cap | Repudiation | browse toggle-off | mitigate | `handleSetSessionBrowse` calls `ws.ClearGrants(id)` on every browse toggle (parity with handleWebServe) — api.go:1295-1304; route POST /sessions/{id}/browse — api.go:140; KillSession also clears stale entry — engine.go:506 | closed |
| T-137-07 | Tampering (CSRF) | CSRF on web write routes | accept (unchanged) | `originAllowedForWrite` untouched and still invoked after the cap check — capability_mw.go:164,172-187; browse is a daemon-socket POST (loopback), not a webserver write route. `TestRequireFilesWrite` PASS. | closed |
| T-137-SC | Tampering (supply chain) | npm/go installs | accept | No new packages. `git diff --stat 3fde1411..73de8dcd` for go.mod / go.sum / frontend/package.json / pnpm-lock.yaml returns zero changes. LockClosedIcon from already-present @heroicons/react. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| T-137-07 | CSRF on web write routes | Phase does not touch `originAllowedForWrite`; browse toggle is a loopback daemon-socket call, not an Origin-bearing webserver route. No new CSRF surface introduced. (Verified: capability_mw.go:172-187 unchanged; TestRequireFilesWrite green) | gsd-security-auditor | 2026-06-20 |
| T-137-SC | New npm/go dependency introduced | No install tasks in any of the 3 plans; manifest diff across the full phase range is empty. (Verified: go.mod/go.sum/package.json/pnpm-lock.yaml diff empty) | gsd-security-auditor | 2026-06-20 |

### Intentional design acceptances (documented in PLAN truths, not threats requiring mitigation)

- **D-05:** RO-code holders gaining read-only filesystem access when browse is ON is intended and accepted. The RO token carries `files.read` by design when the owner enables browse; this is the feature, not a leak.
- **D-08:** Browse state is ephemeral in-memory (`sessionBrowse` map), reset on daemon restart — mirrors the sessionWrites/web-serve lifecycle.

---

## Unregistered Flags

The executor `## Threat Flags` sections across all three SUMMARYs surfaced one flag:

- **stale-cap-on-browse-on** (137-02-SUMMARY, internal/daemon/api.go) — `ClearGrants` fires on browse toggle-ON as well as toggle-OFF. This maps to the existing `stale-cap` threat and is strictly safer than the plan's toggle-off-only requirement (old `files.read` caps issued before browse-ON are invalidated). **Informational — maps to existing threat `stale-cap`; not a new attack surface.**

Plans 01 and 03 reported "No new network endpoints, auth paths, or schema changes" / "None." Plan 02's one new endpoint (POST /sessions/{id}/browse) is a loopback daemon-socket route mapped to threat `stale-cap` (ClearGrants) and T-137-07 (no CSRF surface — not a webserver write route).

**No unregistered flags requiring a new threat mapping.**

---

## Runtime Verification

Security-delta pinning tests executed green during this audit:

```
go test ./internal/daemon/...    -run TestIssueCapabilities_Browse                                  ok
go test ./internal/webserver/... -run TestFilesRoutes_RO_BrowseOn_WriteRoute403                     ok
go test ./internal/webserver/... -run TestFilesRoutes_RW_BrowseOn_WriteRoute200                     ok
go test ./internal/webserver/... -run TestHasPerm_NoStringsContains_Browse                          ok
go test ./internal/webserver/... -run TestRequireFilesWrite                                         ok
```

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-20 | 11 | 11 | 0 | gsd-security-auditor (State B, register_authored_at_plan_time: true) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-20
