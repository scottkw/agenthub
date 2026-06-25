package relay

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestChatSchemaVersion verifies the constant is defined and equal to 1.
func TestChatSchemaVersion(t *testing.T) {
	if ChatSchemaVersion != 1 {
		t.Errorf("expected ChatSchemaVersion=1, got %d", ChatSchemaVersion)
	}
}

// TestChatMessageRoundTrip verifies that marshalling then unmarshalling a fully
// populated ChatMessage produces a value identical to the original.
func TestChatMessageRoundTrip(t *testing.T) {
	original := ChatMessage{
		SchemaVersion: ChatSchemaVersion,
		ID:            "msg-001",
		SessionID:     "sess-abc",
		AuthorID:      "node-xyz",
		AuthorAlias:   "alice",
		Content:       "Hello, world!",
		Mentions:      []string{"bob", "carol"},
		SessionInject: true,
		TimestampMs:   1700000000000,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ChatMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip mismatch:\n  original: %+v\n  decoded:  %+v", original, decoded)
	}
}

// TestChatMessageJSONKeys verifies the exact JSON key names in the wire format.
func TestChatMessageJSONKeys(t *testing.T) {
	msg := ChatMessage{
		SchemaVersion: 1,
		ID:            "id1",
		SessionID:     "s1",
		AuthorID:      "author1",
		AuthorAlias:   "Bob",
		Content:       "hi",
		Mentions:      []string{"alice"},
		SessionInject: true,
		TimestampMs:   12345,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}

	// Verify exact key names are present.
	expectedKeys := map[string]bool{
		"v":             true,
		"id":            true,
		"sessionID":     true,
		"authorID":      true,
		"alias":         true,
		"content":       true,
		"mentions":      true,
		"sessionInject": true,
		"ts":            true,
	}
	for key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q to be present but it was absent", key)
		}
	}

	// No unexpected keys should appear.
	for key := range raw {
		if !expectedKeys[key] {
			t.Errorf("unexpected JSON key %q present in output", key)
		}
	}
}

// TestChatMessageOmitempty verifies that Mentions and SessionInject are absent
// from the JSON when they are zero values.
func TestChatMessageOmitempty(t *testing.T) {
	msg := ChatMessage{
		SchemaVersion: 1,
		ID:            "id1",
		SessionID:     "s1",
		AuthorID:      "author1",
		AuthorAlias:   "Bob",
		Content:       "hi",
		// Mentions is nil (zero value)
		// SessionInject is false (zero value)
		TimestampMs: 12345,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}

	if _, ok := raw["mentions"]; ok {
		t.Error("expected 'mentions' to be absent (omitempty) when nil, but it was present")
	}
	if _, ok := raw["sessionInject"]; ok {
		t.Error("expected 'sessionInject' to be absent (omitempty) when false, but it was present")
	}
}

// TestChatMessageUnknownFieldTolerance verifies that unmarshalling a JSON object
// that contains an unknown extra key succeeds and ignores the extra key.
// This locks Go's default forward-compatibility behavior.
func TestChatMessageUnknownFieldTolerance(t *testing.T) {
	jsonWithExtra := `{
		"v": 1,
		"id": "id1",
		"sessionID": "s1",
		"authorID": "author1",
		"alias": "Bob",
		"content": "hi",
		"ts": 12345,
		"futureField": "some-future-value",
		"anotherNewField": 42
	}`

	var msg ChatMessage
	if err := json.Unmarshal([]byte(jsonWithExtra), &msg); err != nil {
		t.Fatalf("unmarshal with unknown fields failed (expected success): %v", err)
	}

	if msg.ID != "id1" {
		t.Errorf("expected ID=id1, got %q", msg.ID)
	}
	if msg.AuthorAlias != "Bob" {
		t.Errorf("expected alias=Bob, got %q", msg.AuthorAlias)
	}
}
