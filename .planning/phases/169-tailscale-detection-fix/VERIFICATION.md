# Phase 169 — Verification (tailscale-detection-fix)

**Requirement:** FIX-05 (Issue #120)
**Verdict:** ✅ Goal achieved (re-scoped) — 1 live acceptance item deferred (human_needed)
**Verified:** 2026-07-05 · branch `v4.2-funnel-sharing`
**Method:** Goal-backward code inspection + build/test gates run live. (The spawned gsd-verifier died on an API connection error mid-run before writing this file; verification completed inline by the orchestrator.)

## Re-scoped goal

The ROADMAP's original wording ("state reported as *Connected*") is stale. Code review **CR-01** proved a non-admin macsys account *cannot* truthfully report Connected: `tailscaled` writes `/Library/Tailscale/sameuserproof-*` as `0640 root:admin` by design, and both the SDK read and a non-setuid CLI run as the same OS user → identical permission gate. The phase was re-scoped to **honest permission-aware detection**: distinguish "daemon running but status unreadable (EACCES)" from "daemon genuinely down", surface it as a distinct **Permission Limited** state with actionable guidance, and **never** report a false Connected.

## Success criteria

| # | Criterion | Result | Evidence |
|---|-----------|--------|----------|
| D-01 | Invalidated CLI-`status`-fallback fully removed | ✅ | No `cliStatusFunc`/`runTailscaleStatusCLI`/`realCLIStatusFunc`/`os/exec` in `tailscale.go` (grep clean); commit `a0e87398` |
| SC1 / D-02 / D-03 | macsys + sameuserproof EACCES + live daemon → `PermissionLimited=true`, `Connected=false` | ✅ | `tailscale.go:155-160` sets only PermissionLimited/DaemonUp/Installed; `TestCheckHealth_PermissionLimited` PASS |
| D-02 | Genuinely-down daemon NOT flagged permission-limited | ✅ | `realPermProbe` requires `macsysDaemonAlive()` TCP dial to succeed (`tailscale.go:98-105`); `TestCheckHealth_DaemonDown_NotPermissionLimited` PASS |
| D-04 | Gated to `runtime.GOOS==darwin` + macsys `sameuserproof-*` signature | ✅ | `tailscale.go:68-75` |
| Info-disclosure boundary | Probe never reads sameuserproof contents | ✅ | `realPermProbe` only `os.Open`s to observe `fs.ErrPermission`; `macsysDaemonAlive` parses `ipnport` strictly as int, never a path (`tailscale.go:79-95, 112-128`) |
| **Hard SC** | Connected NEVER set true in permission-limited path | ✅ | err-arm leaves `Connected/HasCerts/IP/Domain/AcceptDNS` at zero (`tailscale.go:160`) |
| D-06 / SC2 | SDK-success path byte-for-byte unchanged | ✅ | Steps 3-4 untouched; `TestCheckHealth_FullyHealthy` PASS |
| D-05 | UI surfaces a distinct **Permission Limited** state (never "Connected") with actionable guidance | ✅ | `SettingsTab.tsx:306,316` → `'warn'` class + `'Permission Limited'` label (checked after `connected`, before `daemonUp`); guidance copy `SettingsTab.tsx:750-751` |
| Colorblind-safe | State distinguishable by TEXT, not color alone | ✅ | Distinction carried by the `Permission Limited` label + description sentence, not the dot color |

## Gates (run live)

- `go build ./...` — OK
- `go vet ./internal/webserver/` — clean
- `go test ./internal/webserver/ -run TestCheckHealth` — **11/11** (1 pre-existing SKIP: binary-not-found guard), `ok`
- `pnpm exec tsc --noEmit` — clean (exit 0)
- vitest — new `SettingsTab.tailscale-status` suite 4/4; full suite 2333/2333 (per 169-02-SUMMARY)

## Human-needed (deferred)

- **M-45 (live macsys acceptance):** On a real non-admin macOS account with a genuine `0640 root:admin` sameuserproof file + live macsys daemon, confirm the Settings panel shows "Permission Limited" (not "Not Connected") and the guidance copy. Cannot be exercised on an admin dev box or in CI (a real root-owned file is not constructible there); the branch logic is proven via injected fakes. Registered as M-45 in TESTING.md.

## Conclusion

Both plans deliver the re-scoped FIX-05 goal: the backend honestly detects the permission-limited macsys condition without ever asserting a false Connected, and the Settings UI surfaces it as a distinct, colorblind-safe, actionable state. The only open item is the live macsys acceptance run (M-45), which is environmental and cannot be automated.
