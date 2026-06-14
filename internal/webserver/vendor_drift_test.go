// Package webserver D-04/D-20 vendor-drift guard (Phase 89; generalized in Phase 93 WEB-02).
//
// TestXtermVendorVersionsMatchPnpmLock fails if the resolved @xterm/xterm or
// any @xterm/addon-* version in frontend/pnpm-lock.yaml ever diverges from
// the version manifest at web/vendor/xterm/VERSION. This catches the case where
// a developer runs `pnpm update` and bumps the lockfile without re-copying the
// vendored files (or vice-versa). Phase 93 generalized the regex from
// `addon-fit` only to every `@xterm/addon-*` package.
//
// TestCodeMirrorVersionsMatchPnpmLock (Phase 125-01, T-125-SC) asserts that
// every @codemirror/* and the bare `codemirror` package declared in
// frontend/package.json matches the resolved version in frontend/pnpm-lock.yaml.
// CodeMirror is Vite-bundled (not web-served), so there is no web/vendor/
// directory to compare — we compare package.json declared versions directly
// against pnpm-lock.yaml resolved versions. This gate will be trivially empty
// until Plan 02 installs the packages, at which point it becomes load-bearing.
package webserver

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

var pnpmXtermKeyRe = regexp.MustCompile(`^  '(@xterm/(?:xterm|addon-[\w-]+))@([0-9.]+)':`)

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
	if len(pnpmVersions) < 10 {
		t.Fatalf("failed to parse at least 10 @xterm/* packages (xterm, addon-fit, addon-webgl, addon-unicode11, addon-clipboard, addon-search, addon-web-links, addon-image, addon-serialize, addon-progress) from pnpm-lock.yaml: found %v (Phase 95 SRC-95-06 — addon-web-links joined the manifest; Phase 96 IMG-03 — addon-image joined the manifest; Phase 97 SER-03 — addon-serialize joined the manifest; Phase 98 PRG-04 — addon-progress joined the manifest)", pnpmVersions)
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

// pnpmCMKeyRe matches top-level pnpm-lock.yaml package keys for @codemirror/* and the
// bare "codemirror" meta-package. pnpm v9 quotes scoped packages but NOT bare names:
//
//	  '@codemirror/state@6.4.1':    ← scoped, quoted
//	  codemirror@6.0.2:             ← bare, no quotes
var pnpmCMKeyRe = regexp.MustCompile(`^  '?(@codemirror/[\w-]+|codemirror)@([0-9][^':]+)'?:`)

// TestCodeMirrorVersionsMatchPnpmLock asserts that every @codemirror/* package and
// the bare "codemirror" package declared in frontend/package.json have the same
// resolved version in frontend/pnpm-lock.yaml. This catches the case where a
// developer runs `pnpm update` and bumps the lockfile without updating package.json,
// or vice-versa.
//
// NOTE: CodeMirror is Vite-bundled (no web/vendor/ directory). This test compares
// package.json declared semver ranges against pnpm-lock.yaml resolved versions
// directly — there is no VERSION manifest to check (RESEARCH Open Q1).
//
// The test is trivially empty (zero packages; passes) until Plan 02 installs
// @codemirror/* packages, at which point it becomes a load-bearing gate (T-125-SC).
func TestCodeMirrorVersionsMatchPnpmLock(t *testing.T) {
	// Step 1: read pnpm-lock.yaml (source of truth for resolved versions).
	lockData, err := os.ReadFile("../../frontend/pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("ReadFile pnpm-lock.yaml: %v", err)
	}

	// Step 2: scan for top-level '@codemirror/...' and 'codemirror@...' package keys.
	pnpmVersions := map[string]string{}
	for _, line := range strings.Split(string(lockData), "\n") {
		if m := pnpmCMKeyRe.FindStringSubmatch(line); m != nil {
			pnpmVersions[m[1]] = m[2]
		}
	}

	// Step 3: read frontend/package.json (declared dependencies).
	pkgData, err := os.ReadFile("../../frontend/package.json")
	if err != nil {
		t.Fatalf("ReadFile frontend/package.json: %v", err)
	}
	var pkgJSON struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(pkgData, &pkgJSON); err != nil {
		t.Fatalf("unmarshal package.json: %v", err)
	}

	// Merge dependencies + devDependencies for a complete declared set.
	declaredDeps := make(map[string]string)
	for k, v := range pkgJSON.Dependencies {
		declaredDeps[k] = v
	}
	for k, v := range pkgJSON.DevDependencies {
		declaredDeps[k] = v
	}

	// Step 4: collect every @codemirror/* and 'codemirror' from declared deps.
	declaredCM := map[string]string{}
	for pkg, ver := range declaredDeps {
		if pkg == "codemirror" || strings.HasPrefix(pkg, "@codemirror/") {
			declaredCM[pkg] = ver
		}
	}

	// Step 5: if no CodeMirror packages are declared yet, the test passes trivially.
	// This is the expected state before Plan 02 installs the packages.
	if len(declaredCM) == 0 {
		t.Log("No @codemirror/* packages declared in package.json yet (pre-Plan-02); gate trivially passes.")
		return
	}

	// Step 6: for each declared CodeMirror package, verify the lockfile has a
	// resolved version and that they agree (semver range vs. resolved exact version).
	for pkg, declaredRange := range declaredCM {
		resolvedV, ok := pnpmVersions[pkg]
		if !ok {
			t.Errorf(
				"pnpm-lock.yaml has no resolved entry for %s (declared in package.json as %s) — run `pnpm install` to update the lockfile",
				pkg, declaredRange,
			)
			continue
		}
		// The declared range (e.g. "^6.4.1") may differ from the resolved exact version
		// (e.g. "6.4.1"). Strip leading semver range operators for the comparison.
		stripped := strings.TrimLeft(declaredRange, "^~>=<")
		if resolvedV != stripped {
			t.Errorf(
				"version drift for %s: package.json=%s (stripped=%s), pnpm-lock=%s — run `pnpm install` or update package.json to match",
				pkg, declaredRange, stripped, resolvedV,
			)
		}
	}
}
