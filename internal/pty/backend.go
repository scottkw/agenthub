package pty

import (
	"context"
	"errors"
)

// ErrSessionNotFound is returned when a session with the given ID does not exist.
var ErrSessionNotFound = errors.New("session not found")

// CreateRequest describes what CLI to launch and the initial terminal dimensions.
type CreateRequest struct {
	CLI  string
	Args []string
	Env  []string
	Cols int
	Rows int
}

// SessionBackend is the interface that a PTY backend must implement.
// It is the primary contract between the orchestration layer and the
// platform-specific PTY implementation.
type SessionBackend interface {
	// Create starts a new CLI process in a PTY and returns a Session handle.
	Create(ctx context.Context, req CreateRequest) (*Session, error)

	// Resize changes the terminal dimensions for the given session.
	Resize(id string, cols, rows int) error

	// Kill terminates the process for the given session.
	Kill(id string) error

	// List returns all active sessions managed by this backend.
	List() []*Session
}
