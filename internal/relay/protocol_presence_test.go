package relay

import (
	"encoding/json"
	"testing"
)

// TestPresencePayloadRoundTrip verifies that MakePresenceFrame encodes a
// PresencePayload with the correct type byte and that the JSON body
// round-trips back to an equal value.
func TestPresencePayloadRoundTrip(t *testing.T) {
	entry := PresenceEntry{
		PersonKey: "local:local",
		TailnetID: "local",
		Origin:    "local",
		Alias:     "ken",
		ConnCount: 1,
	}
	p := PresencePayload{Participants: []PresenceEntry{entry}}
	frame := MakePresenceFrame(p)

	if frame[0] != MsgPresence {
		t.Fatalf("wrong type byte: got 0x%02x, want 0x%02x (MsgPresence)", frame[0], MsgPresence)
	}

	var decoded PresencePayload
	if err := json.Unmarshal(frame[1:], &decoded); err != nil {
		t.Fatalf("failed to unmarshal PresencePayload: %v", err)
	}
	if len(decoded.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(decoded.Participants))
	}
	got := decoded.Participants[0]
	if got.PersonKey != entry.PersonKey {
		t.Errorf("PersonKey: got %q, want %q", got.PersonKey, entry.PersonKey)
	}
	if got.TailnetID != entry.TailnetID {
		t.Errorf("TailnetID: got %q, want %q", got.TailnetID, entry.TailnetID)
	}
	if got.Origin != entry.Origin {
		t.Errorf("Origin: got %q, want %q", got.Origin, entry.Origin)
	}
	if got.Alias != entry.Alias {
		t.Errorf("Alias: got %q, want %q", got.Alias, entry.Alias)
	}
	if got.ConnCount != entry.ConnCount {
		t.Errorf("ConnCount: got %d, want %d", got.ConnCount, entry.ConnCount)
	}
}

// TestTypingPayloadRoundTrip verifies that MakeTypingFrame encodes a
// TypingPayload with the correct type byte and that the JSON body
// round-trips back to an equal value.
func TestTypingPayloadRoundTrip(t *testing.T) {
	p := TypingPayload{
		PersonKey: "k:web",
		Alias:     "sam",
		Typing:    true,
	}
	frame := MakeTypingFrame(p)

	if frame[0] != MsgTyping {
		t.Fatalf("wrong type byte: got 0x%02x, want 0x%02x (MsgTyping)", frame[0], MsgTyping)
	}

	var decoded TypingPayload
	if err := json.Unmarshal(frame[1:], &decoded); err != nil {
		t.Fatalf("failed to unmarshal TypingPayload: %v", err)
	}
	if decoded.PersonKey != p.PersonKey {
		t.Errorf("PersonKey: got %q, want %q", decoded.PersonKey, p.PersonKey)
	}
	if decoded.Alias != p.Alias {
		t.Errorf("Alias: got %q, want %q", decoded.Alias, p.Alias)
	}
	if decoded.Typing != p.Typing {
		t.Errorf("Typing: got %v, want %v", decoded.Typing, p.Typing)
	}
}

// TestAliasPayloadRoundTrip verifies that MakeAliasSetFrame encodes an
// AliasPayload with the correct type byte and that the JSON body
// round-trips back to an equal value.
func TestAliasPayloadRoundTrip(t *testing.T) {
	p := AliasPayload{Alias: "ken"}
	frame := MakeAliasSetFrame(p)

	if frame[0] != MsgAliasSet {
		t.Fatalf("wrong type byte: got 0x%02x, want 0x%02x (MsgAliasSet)", frame[0], MsgAliasSet)
	}

	var decoded AliasPayload
	if err := json.Unmarshal(frame[1:], &decoded); err != nil {
		t.Fatalf("failed to unmarshal AliasPayload: %v", err)
	}
	if decoded.Alias != p.Alias {
		t.Errorf("Alias: got %q, want %q", decoded.Alias, p.Alias)
	}
}

// TestMakeSelfFrame verifies that MakeSelfFrame encodes a SelfPayload with the
// correct type byte (0x37 / MsgSelf) and that the JSON body round-trips back
// to an equal value with the expected lowerCamel JSON tags.
func TestMakeSelfFrame(t *testing.T) {
	p := SelfPayload{PersonKey: "local:local", Alias: "ken"}
	frame := MakeSelfFrame(p)

	if len(frame) == 0 {
		t.Fatal("MakeSelfFrame returned empty frame")
	}
	if frame[0] != MsgSelf {
		t.Fatalf("wrong type byte: got 0x%02x, want 0x%02x (MsgSelf)", frame[0], MsgSelf)
	}

	var decoded SelfPayload
	if err := json.Unmarshal(frame[1:], &decoded); err != nil {
		t.Fatalf("failed to unmarshal SelfPayload: %v", err)
	}
	if decoded.PersonKey != p.PersonKey {
		t.Errorf("PersonKey: got %q, want %q", decoded.PersonKey, p.PersonKey)
	}
	if decoded.Alias != p.Alias {
		t.Errorf("Alias: got %q, want %q", decoded.Alias, p.Alias)
	}

	// Verify the JSON tags are lowerCamel (personKey, alias) — not PascalCase.
	// This is the wire contract Plan 161-02 TS parser must match.
	var raw map[string]interface{}
	if err := json.Unmarshal(frame[1:], &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}
	if _, ok := raw["personKey"]; !ok {
		t.Errorf("JSON body missing 'personKey' key (got keys: %v)", raw)
	}
	if _, ok := raw["alias"]; !ok {
		t.Errorf("JSON body missing 'alias' key (got keys: %v)", raw)
	}
}

// TestTypingPayload_TypingFalse verifies that Typing=false is preserved
// (it is non-omitempty in the wire format).
func TestTypingPayload_TypingFalse(t *testing.T) {
	p := TypingPayload{Typing: false}
	frame := MakeTypingFrame(p)
	var decoded TypingPayload
	if err := json.Unmarshal(frame[1:], &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Typing != false {
		t.Errorf("expected Typing=false, got %v", decoded.Typing)
	}
	if decoded.PersonKey != "" {
		t.Errorf("expected empty PersonKey (omitempty), got %q", decoded.PersonKey)
	}
}

// TestValidateAlias covers all documented validation cases.
func TestValidateAlias(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// Trimming
		{"trim leading/trailing spaces", "  ken  ", "ken"},
		{"no trim needed", "ken", "ken"},

		// Empty/whitespace reject
		{"empty string", "", ""},
		{"all whitespace", "   ", ""},

		// Length boundary
		{"exactly 32 runes accepted", "12345678901234567890123456789012", "12345678901234567890123456789012"},
		{"33 runes rejected", "123456789012345678901234567890123", ""},

		// C0 control characters (U+0000–U+001F)
		{"C0 BEL in middle", "a\x07b", ""},
		{"C0 NUL at start", "\x00ab", ""},
		{"C0 TAB (U+0009)", "a\tb", ""},
		{"C0 newline", "a\nb", ""},

		// C1 controls (U+007F–U+009F)
		{"DEL U+007F", "a\x7fb", ""},
		{"C1 U+0080", "ab", ""},
		{"C1 U+009F", "ab", ""},

		// Printable Unicode accepted
		{"ASCII printable", "Ken", "Ken"},
		{"multibyte printable", "José 日本語", "José 日本語"},
		{"emoji (printable codepoints)", "🔥", "🔥"},

		// Space (U+0020) is the boundary — must be accepted
		{"space U+0020 accepted", "hello world", "hello world"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateAlias(tc.input)
			if got != tc.want {
				t.Errorf("ValidateAlias(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
