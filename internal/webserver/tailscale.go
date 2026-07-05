package webserver

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

// macsysSharedDir is the fixed, hardcoded shared-state directory used by the
// macOS "System Extension" (macsys) Tailscale install. No user input ever
// reaches this path (T-169-04).
const macsysSharedDir = "/Library/Tailscale"

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
	// PermissionLimited is true when tailscaled is running (macsys install, confirmed
	// live via the ipnport TCP liveness dial) but its status is unreadable to this OS
	// account due to permissions (macsys sameuserproof is root:admin 0640, unreadable
	// by non-admin accounts). Connected stays false in this state (FIX-05, Issue #120,
	// D-02/D-03) — this field lets the frontend distinguish "permission-limited" from
	// a genuinely down daemon without ever reporting a false Connected.
	PermissionLimited bool `json:"permissionLimited"`
}

// statusFunc is the injectable status function type for testability.
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

// prefsFunc is the injectable prefs-probe function type for testability.
// It returns the CorpDNS (accept-dns) boolean and an error. Mirrors the
// statusFunc injection idiom so tests can pass a fake without a live daemon.
type prefsFunc func(ctx context.Context) (bool, error)

// permProbeFunc is the injectable permission-probe function type for
// testability, mirroring the statusFunc/prefsFunc test-seam idiom. It
// returns true only when tailscaled is a macsys install that is running
// (liveness-confirmed) but whose status is unreadable to this OS account.
type permProbeFunc func(ctx context.Context) bool

// realPermProbe returns a permProbeFunc implementing the research-verified,
// darwin-only macsys permission-limited detection (D-02/D-03/D-04). It never
// reads the sameuserproof token contents — it only observes whether os.Open
// returns fs.ErrPermission (T-169-02, information-disclosure boundary).
func realPermProbe() permProbeFunc {
	return func(ctx context.Context) bool {
		if runtime.GOOS != "darwin" {
			return false
		}

		matches, err := filepath.Glob(filepath.Join(macsysSharedDir, "sameuserproof-*"))
		if err != nil || len(matches) == 0 {
			return false
		}

		sawPermission := false
		for _, m := range matches {
			f, openErr := os.Open(m)
			if openErr == nil {
				// Readable — the SDK would have worked, so this is not the
				// permission-limited condition. Not permission-limited.
				f.Close()
				return false
			}
			if errors.Is(openErr, fs.ErrPermission) {
				sawPermission = true
				continue
			}
			if errors.Is(openErr, fs.ErrNotExist) {
				continue
			}
			// Any other error is unknown — fail safe to "not permission-limited"
			// (T-169-04) but log for visibility (REVIEW IN-01).
			slog.Debug("realPermProbe: unexpected error opening sameuserproof file", "file", m, "error", openErr)
		}

		if !sawPermission {
			return false
		}

		// EACCES observed on at least one sameuserproof file — confirm the
		// daemon is actually alive (not a stale file from a dead daemon)
		// before reporting permission-limited.
		return macsysDaemonAlive(ctx)
	}
}

// macsysDaemonAlive confirms tailscaled liveness by reading the ipnport
// symlink (parsed strictly as an integer port — never used as a filesystem
// path, T-169-01) and dialing 127.0.0.1:<port> with a bounded timeout.
func macsysDaemonAlive(ctx context.Context) bool {
	portStr, err := os.Readlink(filepath.Join(macsysSharedDir, "ipnport"))
	if err != nil {
		return false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}
	d := net.Dialer{Timeout: time.Second}
	conn, err := d.DialContext(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkHealth is the internal testable health check function.
// It accepts an injected statusFn so tests can pass a fake without a live daemon.
// It accepts an injected prefsFn (nullable) to probe the accept-dns setting when Connected.
// It accepts an injected perm (nullable) permission probe, consulted only when the SDK
// status read fails (D-03) — see the err != nil arm below.
// The 4-state cascade: binary detection -> daemon probe -> connection state -> certs.
func checkHealth(ctx context.Context, fn statusFunc, customPath string, pf prefsFunc, perm permProbeFunc) TailscaleHealth {
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
		// Permission-limited detection (FIX-05, Issue #120): the SDK read can
		// fail on non-admin macOS accounts where the macsys sameuserproof file
		// is root:admin 0640 and unreadable. This branch is purely additive
		// and NEVER sets Connected=true (D-03, hard SC) — it only reports
		// that the daemon is running-but-unreadable, distinct from genuinely
		// down.
		if perm != nil && perm(ctx) {
			h.PermissionLimited = true
			h.DaemonUp = true
			h.Installed = true
		}
		return h // Connected/HasCerts/IP/Domain/AcceptDNS stay at zero values
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
	return checkHealth(ctx, lc.StatusWithoutPeers, "", realPrefsFunc(&lc), realPermProbe())
}

// CheckHealthWithCustomPath queries tailscaled and returns TailscaleHealth,
// checking customPath first for the tailscale binary before well-known paths.
func CheckHealthWithCustomPath(ctx context.Context, customPath string) TailscaleHealth {
	var lc local.Client
	return checkHealth(ctx, lc.StatusWithoutPeers, customPath, realPrefsFunc(&lc), realPermProbe())
}
