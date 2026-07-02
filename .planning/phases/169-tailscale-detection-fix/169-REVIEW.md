---
phase: 169-tailscale-detection-fix
reviewed: 2026-07-02T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - internal/webserver/tailscale.go
  - internal/webserver/tailscale_test.go
  - TESTING.md
findings:
  critical: 1
  warning: 2
  info: 2
  total: 5
status: issues-found
---

# Phase 169: Code Review Report

**Reviewed:** 2026-07-02
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

The mechanical implementation is clean: `exec.CommandContext` is used with a fixed argument vector (`binary, "status", "--json"`, no shell, no `fmt.Sprintf` into args), the spawn is bounded solely by the caller's existing `ctx` (no new timeout knob, D-07), `json.Unmarshal` targets the already-imported `*ipnstate.Status` type, the fallback is additive and does not touch the SDK-success branch (verified via diff — Steps 3/4 are byte-identical), and the three new unit tests pass along with the full existing suite. `go build`, `go vet`, and `gofmt -l` are all clean.

However, cross-referencing the actual mechanism against the pinned `tailscale.com` dependency (`v1.98.3`, matching `go.mod`) surfaces a critical concern: the CLI subprocess this phase spawns authenticates to `tailscaled` through the exact same OS-level gate (`/Library/Tailscale/sameuserproof-*`, mode `0640 root:admin`) that already causes the in-process SDK call to fail for non-admin macOS accounts — because both code paths ultimately call into the same `tailscale.com/client/local` dialer logic, and a file permission is a property of the OS user/group, not of which binary reads it. This is Issue #120's entire root cause, and the fallback as implemented is very likely unable to route around it. See CR-01 for supporting evidence gathered from the vendored dependency source and a live filesystem check on this machine's actual Tailscale install.

## Critical Issues

### CR-01: CLI fallback likely hits the identical macsys permission gate it is meant to route around — Issue #120 may not actually be fixed

**File:** `internal/webserver/tailscale.go:50-61,94-113`
**Issue:**

The fallback's premise (per the plan and Issue #120) is: on non-admin macOS accounts, the SDK's `local.Client.StatusWithoutPeers` fails because it can't read `/Library/Tailscale/sameuserproof-*` (mode `0640`, owner `root:admin`) to obtain the daemon's TCP port/token. The fix spawns `tailscale status --json` as a **subprocess of AgentHub, running as the same OS user**, and expects this subprocess to succeed where the in-process SDK call failed.

Verified on this repo's pinned `tailscale.com@v1.98.3` (matches `go.mod`):

- `client/local/local.go:105-126` — `Client.dialer()` is used by **both** `local.Client.StatusWithoutPeers` (our SDK path) and by `cmd/tailscale/cli`'s `runStatus` (`cmd/tailscale/cli/status.go:80-84`, calling `localClient.StatusWithoutPeers`/`.Status`). Both call sites resolve to the identical dial logic: try `safesocket.LocalTCPPortAndToken()` (TCP+token), and only on error fall back to `safesocket.ConnectContext(ctx, lc.socket())` (plain unix socket).
- `safesocket/safesocket_darwin.go:69-84` — `LocalTCPPortAndToken` on darwin only succeeds via `SetCredentials` (only set when the process *is* the macsys/App-Store GUI app itself, detected via `XPC_SERVICE_NAME`/`$HOME` container path — properties set by macOS's app-launch mechanism, not inherited by a plain `os/exec` child process) or by reading the `sameuserproof` file (`portAndTokenFromSameUserProof`, `safesocket_darwin.go:346-360`). A subprocess spawned via `exec.CommandContext` from AgentHub is neither the GUI app nor has those env markers, so it takes the `sameuserproof` path — the exact file that's unreadable to the non-admin caller.
- Live check on this machine's real macsys install (`/Library/Tailscale/`): `sameuserproof-56771` is `-rw-r----- root:admin` and there is **no plain unix socket file** in that directory (only the sameuserproof file and an `ipnport` symlink) — confirming the plain-unix-socket fallback in `client/local/local.go:126` has nothing to connect to for a macsys install, so that fallback doesn't help either.
- The Tailscale binary itself is not setuid (`ls -la /Applications/Tailscale.app/Contents/MacOS/Tailscale` → `-rwxr-xr-x root:wheel`, no `s` bit), so spawning it grants **zero additional OS privilege** over the calling AgentHub process. A `0640 root:admin` file is unreadable by a non-admin-group process regardless of which binary performs the read.

In short: this is an OS file-permission barrier that is a property of the calling *user*, not the calling *binary*. Shelling out to the official CLI as the same unprivileged user does not change what that user is allowed to read. The three new unit tests (`TestCheckHealth_CLIFallback_Success/_NotInvokedOnSDKSuccess/_AlsoFails`) all use injected fakes and cannot exercise this — they prove the Go-side field mapping is correct, not that the real subprocess would ever return success in the target scenario. `169-01-SUMMARY.md` correctly flags M-45 (live non-admin macsys check) as unverified/`human_judgment: true`, but the plan and summary both frame it as a confirmation step rather than the make-or-break test it actually is — given the above, there's a substantial chance M-45 fails and this phase does not close Issue #120.

**Fix:**
Before treating this phase as done, run M-45 for real. If it fails as analysis suggests, this needs a different mitigation, e.g.:
```
- Confirm whether the real Tailscale.app GUI process on a non-admin macsys account
  can itself show "Connected" (i.e. whether this is fixable at all client-side, or
  whether Tailscale product behavior requires admin-group membership to use the
  local API on macsys). If it's an upstream limitation, close #120 as
  "upstream/macOS limitation" with user-facing guidance (e.g. "grant this account
  admin, or install the Homebrew tailscale CLI which uses a different socket
  path") instead of shipping a fallback that silently doesn't fix anything.
- Alternatively, investigate whether AgentHub can read the sameuserproof file via
  a privileged helper/LaunchDaemon (similar pattern to what the real macsys GUI
  uses), rather than an unprivileged subprocess spawn.
```

## Warnings

### WR-01: CLI-fallback-succeeds-but-not-Running is a new, reachable, and entirely untested state combination

**File:** `internal/webserver/tailscale.go:94-113`
**Issue:** When `cli(ctx, binary)` returns a valid, non-nil `*ipnstate.Status` whose `BackendState != "Running"` (e.g. `"Stopped"`, `"NeedsLogin"`, `"Starting"`), the code sets `h.DaemonUp = true`, `h.Installed = true`, `h.Connected = false`, and returns. This is a legitimate, reachable branch (it mirrors the SDK-success path's equivalent state) but none of the three new tests exercise it — `TestCheckHealth_CLIFallback_Success` only supplies `BackendState: "Running"`, and `TestCheckHealth_CLIFallback_AlsoFails` only supplies a CLI error, not a non-nil-but-not-Running status. This is a real code path a live daemon can produce (e.g. transient `"Starting"` during boot) with zero regression coverage.
**Fix:** Add a fourth unit test, e.g. `TestCheckHealth_CLIFallback_NotRunning`, asserting `DaemonUp=true, Installed=true, Connected=false, HasCerts=false, IP=""` when the CLI fallback succeeds with `BackendState: "Stopped"`.

### WR-02: Fallback fires unconditionally on every SDK failure, adding a subprocess spawn to a hot, indefinitely-repeating poll loop with no benefit in the common case

**File:** `internal/webserver/tailscale.go:87-116`, `internal/daemon/process.go:31-58`
**Issue:** `internal/daemon/process.go`'s `upgradeToTailscale` calls `webserver.CheckHealth` every 5 seconds, forever, for the lifetime of the daemon. Whenever the tailscale binary is present but `tailscaled` is simply not running (a common, non-error steady state for any user who has Tailscale installed but isn't currently using it, or who just rebooted before the daemon started) — not just the narrow non-admin-permission scenario this phase targets — every 5-second tick now spawns `tailscale status --json` as a subprocess (per D-03/D-04 "fire on ANY error, no classification"). Combined with CR-01, this means the added subprocess-spawn cost is paid continuously in the common case while providing no benefit in the one case it was built for.
**Fix:** Not a blocker on its own (perf is out of scope), but worth reconsidering once CR-01 is resolved — e.g. only engage the CLI fallback on `runtime.GOOS == "darwin"`, or add a short soft debounce so a permanently-down daemon doesn't spawn a process every 5s indefinitely.

## Info

### IN-01: CLI-spawn stderr is discarded; failures are silent with no log line

**File:** `internal/webserver/tailscale.go:50-61`
**Issue:** `cmd.Output()` discards stderr (only surfaced inside the `*exec.ExitError` if the caller inspects it — `checkHealth` doesn't), and no `slog` call records that the fallback fired or failed, even though the same package uses `slog.Warn`/`slog.Debug` elsewhere (`server.go:410,421,702,755,1511,1530`). Consistent with the file's pre-existing pattern of silently swallowing `pf` errors (line 138), so not a regression, but it means a live M-45 debug session (or any future investigation of Issue #120) has zero visibility into why the fallback did or didn't help.
**Fix:** Consider `slog.Debug("tailscale: CLI status fallback", "err", cliErr)` when the fallback is engaged, to aid the exact kind of live debugging this feature is meant to support.

### IN-02: TESTING.md M-45 conflates "App Store" and "macsys" distributions

**File:** `TESTING.md:722` (Category W, M-45 step 1)
**Issue:** M-45 describes the target install as "Tailscale installed via the macsys (App Store / signed installer) distribution," but per the vendored `tailscale.com` source (`version/prop.go`), App Store (`IsMacAppStoreGUI`/`readMacosSameUserProof`) and macsys (`IsMacSysGUI`/`readMacsysSameUserProof`) are distinct distributions with separate bundle IDs and separate sameuserproof-read code paths. Issue #120 and the project memory both specifically name "macsys," not App Store.
**Fix:** Tighten the M-45 wording to specify the macsys (Standalone, signed installer / `tailscale.com` direct download) distribution specifically, to avoid a tester validating against the wrong install and getting a false pass/fail.

---

_Reviewed: 2026-07-02_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
