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
