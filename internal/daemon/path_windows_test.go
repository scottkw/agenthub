//go:build windows

package daemon

import (
	"strings"
	"testing"
)

// containsString reports whether slice contains an element equal to want.
// Local helper to avoid depending on slices.Contains generics — keeps this
// test file flat and stdlib-only.
func containsString(slice []string, want string) bool {
	for _, s := range slice {
		if s == want {
			return true
		}
	}
	return false
}

// TestPlatformExtraBins_WindowsIncludesPowerShell verifies that both canonical
// PowerShell 7 install locations are returned by platformExtraBins on Windows:
//   - C:\Program Files\PowerShell\7 (hardcoded absolute, system-wide install)
//   - %LOCALAPPDATA%\Microsoft\WindowsApps (Microsoft Store install location)
//
// Without these entries, a service-mode daemon cannot discover pwsh.exe via
// exec.LookPath because the user's PATH is not inherited into the service
// process. See RESEARCH.md Pitfall 1 and Assumption A5 (SHELL-04).
func TestPlatformExtraBins_WindowsIncludesPowerShell(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\test\AppData\Local`)

	got := platformExtraBins()

	if !containsString(got, `C:\Program Files\PowerShell\7`) {
		t.Errorf("platformExtraBins() missing PowerShell 7 install path; got %v", got)
	}
	if !containsString(got, `C:\Users\test\AppData\Local\Microsoft\WindowsApps`) {
		t.Errorf("platformExtraBins() missing Microsoft\\WindowsApps path; got %v", got)
	}
}

// TestPlatformExtraBins_PreservesExistingEntries is a regression guard: it
// proves the four pre-existing entries (APPDATA\npm, LOCALAPPDATA\pnpm,
// LOCALAPPDATA\Programs\nodejs, C:\Program Files\Tailscale) survive the
// PowerShell extension and remain discoverable.
func TestPlatformExtraBins_PreservesExistingEntries(t *testing.T) {
	t.Setenv("APPDATA", `C:\AppData`)
	t.Setenv("LOCALAPPDATA", `C:\Local`)

	got := platformExtraBins()

	wants := []string{
		`C:\AppData\npm`,
		`C:\Local\pnpm`,
		`C:\Local\Programs\nodejs`,
		`C:\Program Files\Tailscale`,
	}
	for _, want := range wants {
		if !containsString(got, want) {
			t.Errorf("platformExtraBins() missing existing entry %q; got %v", want, got)
		}
	}
}

// TestPlatformExtraBins_LocalAppDataEmpty verifies graceful degradation when
// %LOCALAPPDATA% is unset:
//   - The Microsoft\WindowsApps entry is silently skipped (no entry ending
//     in "Microsoft\WindowsApps" appears in the result).
//   - The agy\bin entry is silently skipped (no entry ending in "agy\bin"
//     appears in the result).
//   - The hardcoded C:\Program Files\PowerShell\7 entry is unaffected (it is
//     not env-dependent and must still be returned).
func TestPlatformExtraBins_LocalAppDataEmpty(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")

	got := platformExtraBins()

	for _, p := range got {
		if strings.HasSuffix(p, `Microsoft\WindowsApps`) {
			t.Errorf("platformExtraBins() returned WindowsApps entry %q despite empty LOCALAPPDATA; got %v", p, got)
		}
		if strings.HasSuffix(p, `agy\bin`) {
			t.Errorf("platformExtraBins() returned agy\\bin entry %q despite empty LOCALAPPDATA; got %v", p, got)
		}
	}
	if !containsString(got, `C:\Program Files\PowerShell\7`) {
		t.Errorf("platformExtraBins() missing hardcoded PowerShell 7 path when LOCALAPPDATA empty; got %v", got)
	}
}

// TestPlatformExtraBins_WindowsIncludesAgyBin verifies that the agy CLI
// install path (%LOCALAPPDATA%\agy\bin\agy.exe) is included in platformExtraBins
// when LOCALAPPDATA is set. Without this entry, a service-mode daemon cannot
// discover agy.exe via exec.LookPath. See Phase 149 RESEARCH Fact 1.
func TestPlatformExtraBins_WindowsIncludesAgyBin(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\test\AppData\Local`)

	got := platformExtraBins()

	if !containsString(got, `C:\Users\test\AppData\Local\agy\bin`) {
		t.Errorf("platformExtraBins() missing agy\\bin path; got %v", got)
	}
}
