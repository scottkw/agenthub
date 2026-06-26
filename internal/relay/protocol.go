// Package relay provides the binary framing protocol, scrollback buffer,
// and fan-out hub for PTY session relay over WebSocket.
package relay

import (
	"encoding/json"
	"errors"
	"strings"
)

// Message type bytes — single-byte prefix for every framed message.
const (
	MsgOutput  byte = 0x01 // PTY stdout/stderr → client
	MsgResize  byte = 0x02 // Terminal resize (cols, rows as big-endian uint16)
	MsgTitle   byte = 0x03 // Window title update
	MsgInput   byte = 0x10 // Client keyboard input → PTY stdin
	MsgResize2 byte = 0x11 // Alternative resize format (reserved)
	MsgPing    byte = 0x12 // Keep-alive ping

	// MsgMeta is the server-to-client metadata push channel (JSON payload).
	// Reserved range 0x20-0x2F for future server-push frame types.
	MsgMeta byte = 0x20
)

// MakeOutputFrame prepends the MsgOutput type byte to data.
func MakeOutputFrame(data []byte) []byte {
	frame := make([]byte, 1+len(data))
	frame[0] = MsgOutput
	copy(frame[1:], data)
	return frame
}

// MakeInputFrame prepends the MsgInput type byte to data.
func MakeInputFrame(data []byte) []byte {
	frame := make([]byte, 1+len(data))
	frame[0] = MsgInput
	copy(frame[1:], data)
	return frame
}

// MakeResizeFrame encodes cols and rows as a 5-byte big-endian frame.
//
//	[MsgResize, cols_hi, cols_lo, rows_hi, rows_lo]
func MakeResizeFrame(cols, rows uint16) []byte {
	return []byte{
		MsgResize,
		byte(cols >> 8), byte(cols),
		byte(rows >> 8), byte(rows),
	}
}

// ParseFrame splits a raw frame into its type byte and payload.
// Returns an error if the frame is empty.
func ParseFrame(frame []byte) (msgType byte, payload []byte, err error) {
	if len(frame) == 0 {
		return 0, nil, errors.New("relay: empty frame")
	}
	return frame[0], frame[1:], nil
}

// MetaPayload is the extensible JSON payload for MsgMeta frames.
// All fields are pointers so omitempty works correctly for partial updates.
type MetaPayload struct {
	ViewerCount *int `json:"viewerCount,omitempty"`
}

// MakeMeta encodes a MetaPayload as a MsgMeta frame.
func MakeMeta(p MetaPayload) []byte {
	b, _ := json.Marshal(p) // MetaPayload is always serialisable
	frame := make([]byte, 1+len(b))
	frame[0] = MsgMeta
	copy(frame[1:], b)
	return frame
}

// Chat and presence frame types — 0x30-0x3F reserved for chat/presence.
// MsgChat (0x30) and MsgChatSend (0x31) are Phase 154 dispatch stubs; only
// 0x32-0x34 are dispatched in Phase 152.
const (
	MsgChat     byte = 0x30 // server → client: deliver chat message (JSON ChatMessage)     [Phase 154 dispatch]
	MsgChatSend byte = 0x31 // client → server: send chat message (JSON content)            [Phase 154 dispatch]
	MsgPresence byte = 0x32 // server → client: full presence roster (JSON PresencePayload) [Phase 152]
	MsgTyping   byte = 0x33 // bidirectional: typing-start/stop (JSON TypingPayload)        [Phase 152]
	MsgAliasSet byte = 0x34 // client → server: set/update alias (JSON AliasPayload)        [Phase 152]
)

// PresenceEntry describes one participant in the presence roster.
// PersonKey = TailnetID + ":" + origin — the stable collapse key (D-04).
type PresenceEntry struct {
	PersonKey string `json:"personKey"`
	TailnetID string `json:"tailnetID"` // "local" for desktop owner, node pubkey for web
	Origin    string `json:"origin"`    // "local" (relay loopback) or "web" (webserver Tailscale)
	Alias     string `json:"alias"`
	ConnCount int    `json:"connCount"` // active connections for this person key
}

// PresencePayload is the JSON body of MsgPresence frames (server → client).
// Full roster on every change — clients replace, not patch.
type PresencePayload struct {
	Participants []PresenceEntry `json:"participants"`
}

// TypingPayload is the JSON body of MsgTyping frames (bidirectional).
// Client → server: PersonKey and Alias are left empty (server fills them from Subscriber).
// Server → client: PersonKey and Alias are populated before broadcast.
type TypingPayload struct {
	PersonKey string `json:"personKey,omitempty"`
	Alias     string `json:"alias,omitempty"`
	Typing    bool   `json:"typing"`
}

// AliasPayload is the JSON body of MsgAliasSet frames (client → server).
type AliasPayload struct {
	Alias string `json:"alias"`
}

// MakePresenceFrame encodes a PresencePayload as a MsgPresence frame.
func MakePresenceFrame(p PresencePayload) []byte {
	b, _ := json.Marshal(p) // PresencePayload is always serialisable
	frame := make([]byte, 1+len(b))
	frame[0] = MsgPresence
	copy(frame[1:], b)
	return frame
}

// MakeTypingFrame encodes a TypingPayload as a MsgTyping frame.
func MakeTypingFrame(p TypingPayload) []byte {
	b, _ := json.Marshal(p) // TypingPayload is always serialisable
	frame := make([]byte, 1+len(b))
	frame[0] = MsgTyping
	copy(frame[1:], b)
	return frame
}

// MakeAliasSetFrame encodes an AliasPayload as a MsgAliasSet frame.
func MakeAliasSetFrame(p AliasPayload) []byte {
	b, _ := json.Marshal(p) // AliasPayload is always serialisable
	frame := make([]byte, 1+len(b))
	frame[0] = MsgAliasSet
	copy(frame[1:], b)
	return frame
}

// ValidateAlias returns the alias if valid and non-empty after trimming,
// or "" if the alias should be rejected. It is exported so both the relay
// and webserver read pumps validate alias input identically (single source
// of truth — Pitfall 3). Mitigates T-152-01 (control-char/injection).
//
// Rules:
//   - Leading/trailing whitespace is stripped with strings.TrimSpace.
//   - Empty result after trim is rejected (returns "").
//   - Input longer than 32 runes is rejected — not truncated; the caller
//     should inform the sender so they can choose a shorter alias.
//   - Any rune in the C0 range (U+0000–U+001F) or C1 range
//     (U+007F–U+009F) is rejected; these could interfere with terminal
//     rendering on the web-share surface.
func ValidateAlias(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > 32 {
		return "" // reject — do not truncate silently
	}
	for _, r := range runes {
		if r < 0x0020 || (r >= 0x007F && r <= 0x009F) {
			return "" // C0 or C1 control character
		}
	}
	return trimmed
}

// ChatSchemaVersion is the current version of the ChatMessage wire format.
// Forward-compat: readers can branch on this field when later phases add keys.
// Go's encoding/json ignores unknown fields on Unmarshal, so schema evolution
// is backward-compatible for free — new keys in the JSON are silently skipped
// by older readers.
const ChatSchemaVersion = 1

// ChatMessage is the canonical wire/serialization type for a single chat
// message. It is the stable contract that every client surface and every
// storage layer serializes to — changes to field names or JSON tags are
// backward-incompatible and require a schema migration.
//
// Stability semantics (do not change without a migration plan):
//
//   - AuthorID is the STABLE identity of the sender: the Tailscale node public
//     key (string form) for remote participants, or the literal "local" for the
//     daemon owner. It does NOT change across daemon restarts and is the ONLY
//     authoritative author field for routing or access-control decisions.
//
//   - AuthorAlias is a per-message SNAPSHOT of the author's chosen display name
//     at send time. It is display-only and is NEVER authoritative for routing
//     or access control. Alias changes after a message is stored do not
//     retroactively update stored messages.
//
//   - TimestampMs is UNIX milliseconds in UTC (time.Now().UnixMilli()).
//
//   - SchemaVersion is a forward-compat marker. Set to ChatSchemaVersion (1)
//     by the writer. Readers that understand only schema version 1 may branch
//     on this field to handle future schemas gracefully.
//
//   - Mentions lists AuthorIDs (not aliases) of mentioned participants. Omitted
//     when empty.
//
//   - SessionInject is true when this message triggered an @session PTY write.
//     Omitted (false) for normal chat messages.
//
// JSON encoding: Go's encoding/json silently ignores unknown fields on
// Unmarshal, providing forward-compatibility as later phases add new keys.
type ChatMessage struct {
	SchemaVersion int      `json:"v"`
	ID            string   `json:"id"`
	SessionID     string   `json:"sessionID"`
	AuthorID      string   `json:"authorID"`
	AuthorAlias   string   `json:"alias"`
	Content       string   `json:"content"`
	Mentions      []string `json:"mentions,omitempty"`
	SessionInject bool     `json:"sessionInject,omitempty"`
	TimestampMs   int64    `json:"ts"`
}
