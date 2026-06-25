// Package relay provides the binary framing protocol, scrollback buffer,
// and fan-out hub for PTY session relay over WebSocket.
package relay

import (
	"encoding/json"
	"errors"
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
