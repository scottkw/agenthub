package webserver

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

// TailscaleHealth is the result of a health check against the local tailscaled daemon.
// Serialised to JSON and returned to the Wails frontend via GetTailscaleStatus().
type TailscaleHealth struct {
	Installed    bool   `json:"installed"`    // daemon socket reachable (legacy compat: true when DaemonUp)
	Connected    bool   `json:"connected"`    // BackendState == "Running"
	HasCerts     bool   `json:"hasCerts"`     // len(CertDomains) > 0
	IP           string `json:"ip"`           // first TailscaleIP as string, empty if not connected
	Domain       string `json:"domain"`       // first CertDomain e.g. hostname.ts.net, empty if none
	BinaryFound  bool   `json:"binaryFound"`  // tailscale binary exists on disk
	DaemonUp     bool   `json:"daemonUp"`     // tailscaled daemon socket reachable
	PlatformHint string `json:"platformHint"` // runtime.GOOS value for frontend platform-specific instructions
	// AcceptDNS is true when the local Tailscale node has accept-dns enabled (the default).
	// false means MagicDNS resolution will fail for remote peers, blocking remote file browse;
	// also false when daemon is unreachable or prefs are unavailable (safe zero value). (DNS-03)
	AcceptDNS bool `json:"acceptDns"`
}

// statusFunc is the injectable status function type for testability.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// prefsFunc is the injectable prefs-probe function type for testability.
// It returns the CorpDNS (accept-dns) boolean and an error. Mirrors the
// statusFunc injection idiom so tests can pass a fake without a live daemon.
type prefsFunc func(ctx context.Context) (bool, error)

// cliStatusFunc is the injectable CLI-fallback function type for testability.
// It mirrors the statusFunc/prefsFunc test-seam idiom: tests inject a fake so
// they can exercise the fallback without spawning a real tailscale binary or
// live daemon. binary is the already-resolved path from detectTailscaleBinary
// (D-06), so a custom Tailscale binary path is honored by the fallback too.
type cliStatusFunc func(ctx context.Context, binary string) (*ipnstate.Status, error)

// runTailscaleStatusCLI runs `tailscale status --json` via exec.CommandContext
// with a FIXED argument vector (no shell, no string interpolation, no
// user-controlled args — T-169-01) and unmarshals stdout into the same
// *ipnstate.Status type the SDK path already uses. The spawn is bounded
// solely by ctx's existing deadline (D-07) — no separate timeout is added.
func runTailscaleStatusCLI(ctx context.Context, binary string) (*ipnstate.Status, error) {
	cmd := exec.CommandContext(ctx, binary, "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var status ipnstate.Status
	if err := json.Unmarshal(out, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// realCLIStatusFunc returns the real cliStatusFunc implementation, wired
// through both public entrypoints (CheckHealth, CheckHealthWithCustomPath).
func realCLIStatusFunc() cliStatusFunc {
	return runTailscaleStatusCLI
}

// checkHealth is the internal testable health check function.
// It accepts an injected statusFn so tests can pass a fake without a live daemon.
// It accepts an injected prefsFn (nullable) to probe the accept-dns setting when Connected.
// It accepts an injected cli (nullable) CLI-fallback func, engaged only when the SDK
// read fails (D-03), on all platforms (D-04) — see the err != nil arm below.
// The 4-state cascade: binary detection -> daemon probe -> connection state -> certs.
func checkHealth(ctx context.Context, fn statusFunc, customPath string, pf prefsFunc, cli cliStatusFunc) TailscaleHealth {
	h := TailscaleHealth{PlatformHint: runtime.GOOS}

	// Step 1: Binary detection — custom path -> well-known -> PATH
	binary := detectTailscaleBinary(customPath)
	if binary == "" {
		return h // BinaryFound=false, DaemonUp=false, Installed=false, Connected=false
	}
	h.BinaryFound = true

	// Step 2: Daemon socket probe
	status, err := fn(ctx)
	if err != nil {
		// CLI fallback (FIX-05, Issue #120): the SDK read can fail on
		// non-admin macOS accounts where the macsys sameuserproof file is
		// root:admin 0640 and unreadable. Fire on ANY error, on ALL
		// platforms — no error-string classification, no runtime.GOOS gate
		// (D-03/D-04). This branch is purely additive: it can only improve
		// on the existing not-connected fallthrough below.
		if cli != nil {
			cliStatus, cliErr := cli(ctx, binary)
			if cliErr == nil && cliStatus != nil {
				h.DaemonUp = true
				h.Installed = true
				h.Connected = cliStatus.BackendState == "Running"
				if h.Connected {
					h.HasCerts = len(cliStatus.CertDomains) > 0
					if len(cliStatus.TailscaleIPs) > 0 {
						h.IP = cliStatus.TailscaleIPs[0].String()
					}
					if len(cliStatus.CertDomains) > 0 {
						h.Domain = cliStatus.CertDomains[0]
					}
				}
				// AcceptDNS stays at its false zero value (D-02) — no CLI
				// prefs read is added in the fallback path.
				return h
			}
		}
		// CLI fallback unavailable, or also failed/returned nil (D-05):
		// fall through to today's not-connected state.
		return h // BinaryFound=true, DaemonUp=false, Installed=false, Connected=false
	}
	h.DaemonUp = true
	h.Installed = true // legacy compat: Installed now means daemon reachable

	// Step 3: Connection state
	h.Connected = status.BackendState == "Running"

	// Step 4: Certs and accept-dns state (only if connected)
	if h.Connected {
		h.HasCerts = len(status.CertDomains) > 0
		if len(status.TailscaleIPs) > 0 {
			h.IP = status.TailscaleIPs[0].String()
		}
		if len(status.CertDomains) > 0 {
			h.Domain = status.CertDomains[0]
		}

		// Step 4b: DNS accept state (DNS-03 proactive probe).
		// Failure (daemon not responding) is silently swallowed — safe degradation:
		// AcceptDNS stays false (unknown / assume disabled for warning). RESEARCH Pitfall 4.
		if pf != nil {
			if corpDNS, prefsErr := pf(ctx); prefsErr == nil {
				h.AcceptDNS = corpDNS
			}
		}
	}
	return h
}

// realPrefsFunc returns a prefsFunc that calls lc.GetPrefs and extracts CorpDNS.
func realPrefsFunc(lc *local.Client) prefsFunc {
	return func(ctx context.Context) (bool, error) {
		prefs, err := lc.GetPrefs(ctx)
		if err != nil {
			return false, err
		}
		return prefs.CorpDNS, nil
	}
}

// CheckHealth queries tailscaled via local.Client and returns TailscaleHealth.
// ctx should carry a short timeout (3-5 seconds) to prevent UI hangs.
func CheckHealth(ctx context.Context) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers, "", realPrefsFunc(&lc), realCLIStatusFunc())
}

// CheckHealthWithCustomPath queries tailscaled and returns TailscaleHealth,
// checking customPath first for the tailscale binary before well-known paths.
func CheckHealthWithCustomPath(ctx context.Context, customPath string) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers, customPath, realPrefsFunc(&lc), realCLIStatusFunc())
}
