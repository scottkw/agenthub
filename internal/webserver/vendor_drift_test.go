// Package webserver D-04/D-20 vendor-drift guard (Phase 89).
//
// TestXtermVendorVersionsMatchPnpmLock fails if the resolved @xterm/xterm and
// @xterm/addon-fit versions in frontend/pnpm-lock.yaml ever diverge from the
// version manifest at web/vendor/xterm/VERSION.  This catches the case where
// a developer runs `pnpm update` and bumps the lockfile without re-copying the
// vendored files (or vice-versa).
package webserver

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-fit))@([0-9.]+)':`)

func TestXtermVendorVersionsMatchPnpmLock(t *testing.T) {
	// Step 1: read pnpm-lock.yaml (source of truth for resolved versions).
	lockData, err := os.ReadFile("../../frontend/pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("ReadFile pnpm-lock.yaml: %v", err)
	}

	// Step 2: scan for top-level '@xterm/...' package keys.
	pnpmVersions := map[string]string{}
	for _, line := range strings.Split(string(lockData), "\n") {
		if m := pnpmXtermKeyRe.FindStringSubmatch(line); m != nil {
			pnpmVersions[m[1]] = m[2]
		}
	}
	if len(pnpmVersions) < 2 {
		t.Fatalf("failed to parse @xterm/xterm and @xterm/addon-fit from pnpm-lock.yaml: found %v (Phase 89 D-04 — the lockfile format may have changed; see 89-RESEARCH.md Q3)", pnpmVersions)
	}

	// Step 3: read web/vendor/xterm/VERSION (the vendored manifest).
	versionData, err := os.ReadFile("../../web/vendor/xterm/VERSION")
	if err != nil {
		t.Fatalf("ReadFile web/vendor/xterm/VERSION: %v", err)
	}

	// Step 4: parse each non-empty, non-comment line as @<scope>/<pkg>@<version>.
	vendorVersions := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(versionData)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		// "@xterm/xterm@6.0.0" → SplitN on "@" gives ["", "xterm/xterm", "6.0.0"]
		parts := strings.SplitN(line, "@", 3)
		if len(parts) == 3 {
			vendorVersions["@"+parts[1]] = parts[2]
		}
	}

	// Step 5: for each package resolved in pnpm-lock.yaml, check VERSION agrees.
	for pkg, wantV := range pnpmVersions {
		if _, ok := vendorVersions[pkg]; !ok {
			t.Errorf(
				"web/vendor/xterm/VERSION missing entry for %s (Phase 89 D-04: re-copy files from frontend/node_modules/%s/ and update VERSION to include this line)",
				pkg, pkg,
			)
			continue
		}
		if gotV := vendorVersions[pkg]; gotV != wantV {
			t.Errorf(
				"version drift for %s: pnpm-lock=%s, VERSION=%s — update VERSION AND re-copy files from frontend/node_modules/%s/ per Phase 89 D-05 (pnpm update + cp). Do NOT edit VERSION alone without re-copying the files.",
				pkg, wantV, gotV, pkg,
			)
		}
	}
}
