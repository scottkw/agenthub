package daemon

import (
	"fmt"
	"strings"
	"testing"
)

// TestHandleSetImageConfig_RangeRejected — Phase 96 IMG-02.
// PATCH /settings/image-config with StorageLimit out of [1, 1000] OR with
// unknown JSON fields must return 400 Bad Request; sibling PluginSettings
// fields and ImageConfig.StorageLimit MUST remain unchanged.
//
// Per 96-PATTERNS.md §`internal/daemon/api.go` Adapt block:
//   - StorageLimit < 1 or > 1000  → 400 (T-96-02-01)
//   - DisallowUnknownFields gate  → 400 (T-96-02-02)
//   - body > 8 KiB MaxBytesReader → 400 (T-96-02-03)
//
// Defense-in-depth mirrors handleSetWebLinksConfig.
func TestHandleSetImageConfig_RangeRejected(t *testing.T) {
	_, _, socketPath := testDaemon(t)

	// Capture the daemon's pre-call ImageConfig.StorageLimit (the default
	// 16 from defaultPluginSettings) so we can assert it remains unchanged
	// after each rejected request. We use rawGet on /settings/plugins which
	// returns the full PluginSettings JSON.
	getStorageLimit := func() int {
		t.Helper()
		client := NewDaemonClient(socketPath)
		ps, err := client.GetPluginSettings()
		if err != nil {
			t.Fatalf("GetPluginSettings: %v", err)
		}
		return ps.ImageConfig.StorageLimit
	}
	getSearchConfig := func() SearchConfig {
		t.Helper()
		client := NewDaemonClient(socketPath)
		ps, err := client.GetPluginSettings()
		if err != nil {
			t.Fatalf("GetPluginSettings: %v", err)
		}
		return ps.SearchConfig
	}

	preLimit := getStorageLimit()
	preSearch := getSearchConfig()

	type rejectedCase struct {
		name           string
		body           string
		wantBodyHasOne []string // body must contain at least one of these substrings
	}

	cases := []rejectedCase{
		{
			name:           "StorageLimit=0",
			body:           `{"storageLimit": 0}`,
			wantBodyHasOne: []string{"1", "1000"},
		},
		{
			name:           "StorageLimit=-1",
			body:           `{"storageLimit": -1}`,
			wantBodyHasOne: []string{"1", "1000"},
		},
		{
			name:           "StorageLimit=1001",
			body:           `{"storageLimit": 1001}`,
			wantBodyHasOne: []string{"1", "1000"},
		},
		{
			name: "UnknownField extra=y",
			body: `{"storageLimit": 16, "extra": "y"}`,
			// http.Error response from json decode failure — exact wording
			// is "invalid request body" but we don't pin it; pin only that
			// 400 was returned.
			wantBodyHasOne: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := rawPatch(t, socketPath, "/settings/image-config", tc.body)
			if status != 400 {
				t.Errorf("PATCH /settings/image-config %s: status got %d, want 400 (body=%q)", tc.name, status, string(body))
			}
			if len(tc.wantBodyHasOne) > 0 {
				bodyStr := string(body)
				foundAll := true
				for _, want := range tc.wantBodyHasOne {
					if !strings.Contains(bodyStr, want) {
						foundAll = false
						break
					}
				}
				if !foundAll {
					t.Errorf("PATCH /settings/image-config %s: body=%q missing one of %v", tc.name, bodyStr, tc.wantBodyHasOne)
				}
			}
			// Sibling integrity: rejected PATCH must not have mutated the
			// daemon's persisted ImageConfig.StorageLimit.
			if got := getStorageLimit(); got != preLimit {
				t.Errorf("after rejected %s: ImageConfig.StorageLimit changed: got %d, want %d", tc.name, got, preLimit)
			}
			if got := getSearchConfig(); got != preSearch {
				t.Errorf("after rejected %s: SearchConfig changed: got %+v, want %+v", tc.name, got, preSearch)
			}
		})
	}

	// Body > 8 KiB MaxBytesReader gate (T-96-02-03). Construct a payload
	// whose JSON length exceeds 8192 bytes; the decoder must error before
	// the range check.
	t.Run("BodyExceeds8KiB", func(t *testing.T) {
		// Build {"storageLimit": 16}<padding> by injecting a long unknown
		// field name. The MaxBytesReader cap is 8192 bytes; 9 KiB is well
		// over.
		filler := strings.Repeat("a", 9*1024)
		oversized := fmt.Sprintf(`{"storageLimit": 16, "%s": "x"}`, filler)
		status, _ := rawPatch(t, socketPath, "/settings/image-config", oversized)
		if status != 400 {
			t.Errorf("PATCH /settings/image-config oversized body: status got %d, want 400", status)
		}
		if got := getStorageLimit(); got != preLimit {
			t.Errorf("after oversized body: ImageConfig.StorageLimit changed: got %d, want %d", got, preLimit)
		}
	})
}

// TestHandleSetImageConfig_ValidAccepted — Phase 96 IMG-02.
// PATCH /settings/image-config with valid StorageLimit values returns
// 204 No Content; the persisted struct reflects the new value;
// sibling PluginSettings fields remain unchanged.
//
// Boundary cases: StorageLimit=1 (lower bound inclusive),
// StorageLimit=1000 (upper bound inclusive), StorageLimit=32 (mid range,
// representative of a hypothetical user override above the 16 MB default).
func TestHandleSetImageConfig_ValidAccepted(t *testing.T) {
	_, _, socketPath := testDaemon(t)

	getPluginSettings := func() PluginSettings {
		t.Helper()
		client := NewDaemonClient(socketPath)
		ps, err := client.GetPluginSettings()
		if err != nil {
			t.Fatalf("GetPluginSettings: %v", err)
		}
		return ps
	}

	// Capture pre-call siblings so accepted PATCH calls can be asserted to
	// not mutate them.
	preSiblings := getPluginSettings()

	type acceptedCase struct {
		name string
		body string
		want int
	}

	cases := []acceptedCase{
		{name: "StorageLimit=1", body: `{"storageLimit": 1}`, want: 1},
		{name: "StorageLimit=1000", body: `{"storageLimit": 1000}`, want: 1000},
		{name: "StorageLimit=16", body: `{"storageLimit": 16}`, want: 16},
		{name: "StorageLimit=32", body: `{"storageLimit": 32}`, want: 32},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := rawPatch(t, socketPath, "/settings/image-config", tc.body)
			if status != 204 {
				t.Errorf("PATCH /settings/image-config %s: status got %d, want 204 (body=%q)", tc.name, status, string(body))
			}
			if len(body) != 0 {
				t.Errorf("PATCH /settings/image-config %s: body got %q, want empty (204 No Content)", tc.name, string(body))
			}

			ps := getPluginSettings()
			if ps.ImageConfig.StorageLimit != tc.want {
				t.Errorf("after %s: ImageConfig.StorageLimit got %d, want %d", tc.name, ps.ImageConfig.StorageLimit, tc.want)
			}

			// Siblings unchanged: every other PluginSettings field equals
			// its pre-call value.
			if ps.WebGL != preSiblings.WebGL {
				t.Errorf("after %s: WebGL changed", tc.name)
			}
			if ps.Unicode11 != preSiblings.Unicode11 {
				t.Errorf("after %s: Unicode11 changed", tc.name)
			}
			if ps.Search != preSiblings.Search {
				t.Errorf("after %s: Search changed", tc.name)
			}
			if ps.SearchConfig != preSiblings.SearchConfig {
				t.Errorf("after %s: SearchConfig changed: got %+v, want %+v", tc.name, ps.SearchConfig, preSiblings.SearchConfig)
			}
			if ps.WebLinks != preSiblings.WebLinks {
				t.Errorf("after %s: WebLinks changed", tc.name)
			}
			if ps.WebLinksConfig != preSiblings.WebLinksConfig {
				t.Errorf("after %s: WebLinksConfig changed: got %+v, want %+v", tc.name, ps.WebLinksConfig, preSiblings.WebLinksConfig)
			}
			if ps.Image != preSiblings.Image {
				t.Errorf("after %s: Image (boolean) changed", tc.name)
			}
			if ps.Serialize != preSiblings.Serialize {
				t.Errorf("after %s: Serialize changed", tc.name)
			}
			if ps.Clipboard != preSiblings.Clipboard {
				t.Errorf("after %s: Clipboard changed", tc.name)
			}
			if ps.Progress != preSiblings.Progress {
				t.Errorf("after %s: Progress changed", tc.name)
			}
		})
	}
}

// TestDaemonClient_SetImageConfig_RoundTrip — Phase 96 IMG-02.
// Exercises (*DaemonClient).SetImageConfig over the Unix socket; valid
// values return nil and persist; rejected values return non-nil error.
//
// This test exists so the DaemonClient wrapper has at least one direct
// caller-side test (the rest of the GREEN coverage runs through rawPatch).
func TestDaemonClient_SetImageConfig_RoundTrip(t *testing.T) {
	_, client, _ := testDaemon(t)

	// Valid mid-range value lands.
	if err := client.SetImageConfig(ImageConfig{StorageLimit: 64}); err != nil {
		t.Fatalf("SetImageConfig(64): want nil, got %v", err)
	}
	ps, err := client.GetPluginSettings()
	if err != nil {
		t.Fatalf("GetPluginSettings: %v", err)
	}
	if ps.ImageConfig.StorageLimit != 64 {
		t.Errorf("after SetImageConfig(64): StorageLimit got %d, want 64", ps.ImageConfig.StorageLimit)
	}

	// Lower-bound inclusive: StorageLimit=1 accepted.
	if err := client.SetImageConfig(ImageConfig{StorageLimit: 1}); err != nil {
		t.Errorf("SetImageConfig(1) lower bound: want nil, got %v", err)
	}
	ps, _ = client.GetPluginSettings()
	if ps.ImageConfig.StorageLimit != 1 {
		t.Errorf("after SetImageConfig(1): StorageLimit got %d, want 1", ps.ImageConfig.StorageLimit)
	}

	// Upper-bound inclusive: StorageLimit=1000 accepted.
	if err := client.SetImageConfig(ImageConfig{StorageLimit: 1000}); err != nil {
		t.Errorf("SetImageConfig(1000) upper bound: want nil, got %v", err)
	}
	ps, _ = client.GetPluginSettings()
	if ps.ImageConfig.StorageLimit != 1000 {
		t.Errorf("after SetImageConfig(1000): StorageLimit got %d, want 1000", ps.ImageConfig.StorageLimit)
	}

	// Out-of-range values return non-nil error.
	if err := client.SetImageConfig(ImageConfig{StorageLimit: 0}); err == nil {
		t.Errorf("SetImageConfig(0): want non-nil error, got nil")
	}
	if err := client.SetImageConfig(ImageConfig{StorageLimit: 1001}); err == nil {
		t.Errorf("SetImageConfig(1001): want non-nil error, got nil")
	}

	// Persisted value still 1000 after rejected calls (sibling integrity at
	// the persistence layer — rejected calls must not stomp last-good value).
	ps2, err := client.GetPluginSettings()
	if err != nil {
		t.Fatalf("GetPluginSettings: %v", err)
	}
	if ps2.ImageConfig.StorageLimit != 1000 {
		t.Errorf("after rejected SetImageConfig calls: StorageLimit got %d, want 1000 (last good)", ps2.ImageConfig.StorageLimit)
	}
}
