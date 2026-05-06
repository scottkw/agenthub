package webserver

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	webfs "github.com/scottkw/agenthub/web"
)

// Phase 95 Plan 95-06 — web parity gates for the web-links addon.
// Mirrors Phase 94 find_bar_test.go pattern (asset reachability +
// source-inspection invariants). All three tests were skip-marked RED
// scaffolds authored in Plan 95-01 Task 2 and turned GREEN here.

// TestAssets_AddonWebLinks — Phase 95 LNK-01..05 web parity.
// GET /assets/xterm/addons/addon-web-links.js returns 200 with a
// content-type containing "javascript". Mirrors Phase 94
// TestAssets_AddonSearch verbatim. T-95-06-01 mitigation gate (vendor
// drift fail-loud is the upstream gate; this asserts the asset is
// actually served at the expected URL once embed.go is extended).
func TestAssets_AddonWebLinks(t *testing.T) {
	ws, client := testServer(t)
	const path = "/assets/xterm/addons/addon-web-links.js"
	resp, err := client.Get(ws.BaseURL() + path)
	if err != nil {
		t.Fatalf("client.Get %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Errorf("GET %s: expected javascript content-type, got %q", path, ct)
	}
}

// TestSecurity_NoCurrentTabNavigation — Phase 95 LNK-04 / T-95-06-02 mitigation.
//
// Source-inspect web/assets/terminal.js AND frontend/src/lib/openLink.ts for
// any of the canonical current-tab-navigation patterns. ZERO matches required.
// Defends against a future commit accidentally introducing
// `location.href = url` (etc.) in either file — which would silently downgrade
// the noopener,noreferrer guarantee and re-open the window.opener pivot
// surface (RESEARCH §"Pattern 5: Single-Helper Opener" + §"Pitfall — comment
// strings can become uncommented code in a refactor"; we scan the whole file,
// comments included, deliberately).
func TestSecurity_NoCurrentTabNavigation(t *testing.T) {
	bannedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`location\.href\s*=`),
		regexp.MustCompile(`window\.location\s*=`),
		regexp.MustCompile(`document\.location\s*=`),
	}
	files := []string{
		"../../web/assets/terminal.js",
		"../../frontend/src/lib/openLink.ts",
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, pat := range bannedPatterns {
			if pat.Match(data) {
				t.Errorf(
					"%s contains banned current-tab navigation pattern %q (T-95-06-02 mitigation: openLink MUST go through window.open with noopener,noreferrer; never assign to location.href / window.location / document.location). Comment strings count — they could become uncommented in a refactor.",
					path, pat.String(),
				)
			}
		}
	}
}

// TestTerminalJS_WebLinksOpener — Phase 95 LNK-04 / T-95-06-03..04 mitigation.
//
// Source-inspect web/assets/terminal.js: must define an openLink function;
// must contain the EXACT '_blank', 'noopener,noreferrer' options literal
// (so a refactor that drops noopener / replaces it with 'noopener noreferrer'
// or similar fails CI); must NOT reference BrowserOpenURL because the web
// session is never inside the Wails runtime — that is a desktop-only
// opener path (frontend/src/lib/openLink.ts).
func TestTerminalJS_WebLinksOpener(t *testing.T) {
	data, err := webfs.WebFS.ReadFile("assets/terminal.js")
	if err != nil {
		t.Fatalf("ReadFile assets/terminal.js: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "function openLink") {
		t.Error("terminal.js missing `function openLink` definition (T-95-06-03 mitigation)")
	}
	if !strings.Contains(s, `'_blank', 'noopener,noreferrer'`) {
		t.Error("terminal.js openLink does not pass the EXACT '_blank', 'noopener,noreferrer' options string (T-95-06-03 mitigation: a future commit dropping noopener,noreferrer must fail CI)")
	}
	if strings.Contains(s, "BrowserOpenURL") {
		t.Error("terminal.js MUST NOT reference BrowserOpenURL — web is never inside Wails (T-95-06-04 mitigation)")
	}
	if !strings.Contains(s, "WebLinksAddon.WebLinksAddon") {
		t.Error("terminal.js must construct via the namespace global form `new WebLinksAddon.WebLinksAddon(...)` (Pitfall #7 — UMD root assignment is a namespace object, not the constructor itself)")
	}
}
