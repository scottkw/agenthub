# Phase 169: Tailscale Detection Fix - Context

**Gathered:** 2026-07-02 (re-scoped 2026-07-02 — see supersession note)
**Status:** Ready for planning

> **Re-scope note (2026-07-02).** The original CONTEXT (CLI `tailscale status --json`
> fallback) was **invalidated** by Phase 169 code review CR-01 (see `169-REVIEW.md`): a
> non-setuid `tailscale` CLI spawned by AgentHub runs as the **same OS user** as the SDK
> read and hits the identical `0640 root:admin` `sameuserproof` permission gate that
> `tailscaled` sets **by design**; only the macsys GUI bypasses it via in-process
> `SetCredentials`. A same-user shell-out grants no extra privilege, so it cannot report
> true Connected status. FIX-05 was re-scoped to **honest permission-aware detection**.
> The old CLI-fallback decisions (previously D-01..D-07) are **superseded** by the
> decisions below; the old alternatives remain in `169-DISCUSSION-LOG.md` for audit.

<domain>
## Phase Boundary

Fix Tailscale connection detection on non-admin macOS `macsys` accounts (Issue #120). On
those accounts the `macsys` `sameuserproof` file is `0640 root:admin` and unreadable by the
non-admin user, so the SDK read `local.Client.StatusWithoutPeers(ctx)` fails with a
permission error. Today `checkHealth` (`internal/webserver/tailscale.go`) treats that error
identically to "daemon down" (`DaemonUp=false`), producing a **false** "installed but not
connected" / "Daemon Stopped" report even though `tailscaled` is actually running.

This phase makes detection **honest**:
1. **Revert** the invalidated 169-01 CLI-fallback code.
2. **Detect** the "daemon running but status unreadable due to OS permissions" condition as
   a state **distinct** from daemon-down.
3. **Surface** it accurately in the Settings Tailscale status UI, with **actionable
   guidance** (grant this account admin, or install the Homebrew `tailscale` build, which
   uses a different socket path).

**Not in scope:** a privileged helper / LaunchDaemon; any change to the SDK-success path; a
false `Connected=true`; the Funnel-path `StatusWithoutPeers` call in `server.go:620`; CLI
prefs reads for AcceptDNS.

</domain>

<decisions>
## Implementation Decisions

### Revert the invalidated approach
- **D-01:** Revert 169-01's CLI-fallback code in full — remove the `cliStatusFunc` type,
  `runTailscaleStatusCLI`, `realCLIStatusFunc`, the `cli` parameter on `checkHealth`, the
  fallback branch inside the `err != nil` arm, and the three fallback unit tests
  (`TestCheckHealth_CLIFallback_Success/_NotInvokedOnSDKSuccess/_AlsoFails`). Restore
  `checkHealth`'s pre-169-01 signature and the `CheckHealth` / `CheckHealthWithCustomPath`
  wrappers. Rationale: CR-01 proves the fallback can report a **false** Connected on the
  exact case it targets — it is misleading dead code. Reverting is a prerequisite, not a
  separate cleanup.

### Honest detection of the permission-limited state
- **D-02:** When the SDK read fails, distinguish **"permission-limited"** (daemon running,
  status unreadable) from **"daemon genuinely down."** Detection mechanism is
  research-gated (see Research Needs): the leading candidate is to check the macsys
  `sameuserproof` file — glob `/Library/Tailscale/sameuserproof-*`; if it **exists** but
  `open()` returns `EACCES`, `tailscaled` is running and we merely lack permission →
  permission-limited. If absent / connection-refused → daemon down. The researcher must
  confirm whether the SDK error itself is reliably classifiable (errno/type) before
  committing to the file-stat approach.
- **D-03:** Represent the new condition with a dedicated, additive field on
  `TailscaleHealth` (e.g. `PermissionLimited bool`, exact name/shape planner's call). **Do
  not** set `Connected=true` in this state (hard constraint — SC: no false Connected). The
  `DaemonUp` semantics in this state (true = "daemon is up, honestly" vs. staying false
  with only the new flag driving the UI) is a planner decision — note that
  `internal/daemon/process.go` gates web-server restart on `Connected && HasCerts &&
  IP != ""`, so a permission-limited node (Connected=false) never triggers a restart
  regardless.

### Platform scope
- **D-04:** The permission-limited detection is **macOS-macsys-specific** — that is the only
  distribution with the `/Library/Tailscale/sameuserproof-*` gate. Guard it behind
  `runtime.GOOS == "darwin"` plus the file check. Windows/Linux and the macOS App-Store
  distribution keep today's behavior. (Contrast with the invalidated D-04, which ran the CLI
  fallback on all platforms — no longer applicable.)

### UI surface (SC3)
- **D-05:** Surface the permission-limited state in the `SettingsTab` Tailscale status block
  as a **distinct** message/row — not the existing "Daemon Stopped" / "Not Connected"
  copy — with actionable guidance: grant this account admin, or install the Homebrew
  `tailscale` build. Exact copy and visual treatment are Claude's discretion but must be
  accurate (do not imply Connected). The existing 4-step cascade (Binary → Daemon →
  Connected → Certs) at `SettingsTab.tsx:737+` is the surface to extend.

### SDK-success path
- **D-06:** The SDK-success path (Steps 3–4 of `checkHealth`) is **byte-for-byte unchanged**
  (SC2). The fix only touches the `err != nil` arm and the struct/UI.

### Claude's Discretion
- Exact detection mechanism (file-stat + EACCES vs. SDK-error classification), pending
  research.
- Exact new field name/shape on `TailscaleHealth` and the `DaemonUp` semantics in the
  permission-limited state.
- Exact UI copy and placement of the guidance.
- Whether detection lives in a small injectable helper (mirroring the `statusFunc` /
  `prefsFunc` test-seam idiom) so it can be unit-tested without a live macsys install.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirement (RE-SCOPED — read the re-scoped text, not the original)
- `.planning/REQUIREMENTS.md` — FIX-05 (line 54, re-scoped 2026-07-02): honest,
  permission-aware detection with guidance. The bullet beneath it documents why the CLI
  fallback was invalidated.
- `.planning/ROADMAP.md` §"Phase 169": Goal + 3 Success Criteria (accurate state, unchanged
  SDK-success behavior, actionable UI guidance) + the re-plan note.

### Invalidation evidence (read to understand WHY the approach changed)
- `.planning/phases/169-tailscale-detection-fix/169-REVIEW.md` — CR-01 (vendored
  `tailscale.com@v1.98.3` source + live `/Library/Tailscale/` check). Also WR-01/WR-02/IN-01
  worth reading for related gaps.

### Implementation targets
- `internal/webserver/tailscale.go` — `checkHealth` (line 75); the `err != nil` arm
  (lines 87–117) currently holds the CLI fallback to revert (D-01); `TailscaleHealth` struct
  (lines 15–28) gets the new field (D-03).
- `internal/webserver/tailscale_test.go` — the three fallback tests to remove (D-01) and the
  `statusFunc`/`prefsFunc` injection idiom to mirror for a testable detection helper.
- `frontend/src/components/SettingsTab.tsx` — the Tailscale status cascade (status label at
  lines 297–307; rendered rows at lines 737–782) is the SC3 UI surface (D-05). The
  serialized `TailscaleHealth` shape is mirrored at lines 52–59.

### Related SDK call (NOT modified this phase — awareness only)
- `internal/webserver/server.go:620` — the Funnel path's own `StatusWithoutPeers` call. Same
  non-admin failure mode, but out of scope (deferred).

</canonical_refs>

<research_needs>
## Research Needs (drives the RESEARCH.md before planning)

1. **Is the SDK read error classifiable?** Does `local.Client.StatusWithoutPeers` surface a
   distinguishable error (errno `EACCES`, a typed error, a stable message) on the non-admin
   macsys `sameuserproof` case vs. a genuinely-down daemon (connection refused / no socket)?
   If yes, error-classification may be cleaner than a file stat.
2. **The `sameuserproof` file-stat approach.** Confirm the glob path
   (`/Library/Tailscale/sameuserproof-*`), that its presence reliably means `tailscaled` is
   running, and that a non-admin `open()` yields `EACCES` (not `ENOENT`). Confirm there is no
   plain unix socket to fall back on for macsys (per CR-01's live check).
3. **App-Store vs. macsys distinction** (per IN-02): different bundle IDs / sameuserproof
   read paths (`readMacosSameUserProof` vs. `readMacsysSameUserProof`, `version/prop.go`).
   Scope the detection to macsys specifically to avoid false positives.
4. **Testability.** How to unit-test the detection helper without a live macsys install —
   inject the file-stat / error source the way `statusFunc`/`prefsFunc` are injected.

</research_needs>

<specifics>
## Specific Ideas

- Root cause (memory `reference_tailscale_connection_detection`): macsys `sameuserproof` is
  `root:admin 0640`, set by `tailscaled` design (`safesocket_darwin.go` `Fchown(...,0,80)`),
  so non-admin accounts get an SDK read error and fall to the "installed but not connected"
  branch. The honest fix reports this accurately rather than pretending Connected.
- CR-01's own recommended framing for guidance: "grant this account admin, or install the
  Homebrew tailscale CLI which uses a different socket path."

</specifics>

<deferred>
## Deferred Ideas

- Privileged helper / LaunchDaemon to read `sameuserproof` (the mechanism the real macsys
  GUI uses) — explicitly out of scope for this fix; would be a much larger security-sensitive
  change. Note for a future phase if #120 follow-up demands true Connected on non-admin.
- Applying honest detection / guidance to the Funnel-path `StatusWithoutPeers` call in
  `server.go:620`.
- CLI prefs fallback for `AcceptDNS` on non-admin accounts.

</deferred>

---

*Phase: 169-Tailscale Detection Fix*
*Context re-scoped: 2026-07-02 — honest permission-aware detection (supersedes CLI fallback)*
