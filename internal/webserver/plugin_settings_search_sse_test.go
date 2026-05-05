package webserver

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Phase 94 Plan 94-02 — SRC-02 SSE round-trip test for nested SearchConfig.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 02-daemon wave 1.
//
// The pluginSettingsProvider func() []byte indirection serializes any nested
// struct fields automatically (RESEARCH §"Pattern 3" Assumption A6). This
// test pins the contract: the SSE stream MUST carry the nested searchConfig
// JSON object end-to-end so the frontend sees per-flag toggles propagate.
func TestPluginSettingsSSE_Search(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pcs-search")
	ws.SetPluginSettingsProvider(func() []byte {
		// Phase 94 — daemon SearchConfig nested in PluginSettings JSON.
		return []byte(`{"webgl":true,"unicode11":true,"search":true,"searchConfig":{"regex":true,"caseSensitive":false,"wholeWord":true},"webLinks":true,"image":true,"serialize":true,"clipboard":true,"progress":false}`)
	})
	tok := issueCapFor(t, ws, "sess-pcs-search", "read,write")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", ws.BaseURL()+"/api/plugin-config/stream?cap="+tok, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Read the initial SSE frame.
	rdr := bufio.NewReader(resp.Body)
	var frame strings.Builder
	for {
		line, err := rdr.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString: %v (partial=%q)", err, frame.String())
		}
		frame.WriteString(line)
		if line == "\n" {
			break
		}
	}

	body := frame.String()
	// Pin the exact serialization shape — frontend depends on the camelCase
	// field names matching daemon Go struct json tags.
	if !strings.Contains(body, `"searchConfig":{"regex":true,"caseSensitive":false,"wholeWord":true}`) {
		t.Errorf("SSE frame missing nested searchConfig payload; frame=%q", body)
	}
}
