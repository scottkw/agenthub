package pty

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
)

// NativePTYBackend implements SessionBackend using the host OS PTY facilities.
// On POSIX systems it uses Unix PTYs via go-pty; on Windows it uses ConPTY.
type NativePTYBackend struct {
	registry *SessionRegistry
}

// NewNativePTYBackend returns a NativePTYBackend backed by a fresh SessionRegistry.
func NewNativePTYBackend() *NativePTYBackend {
	return &NativePTYBackend{
		registry: NewSessionRegistry(),
	}
}

// Create starts a new CLI process in a PTY and returns a Session handle.
// The process receives TERM=xterm-256color and COLORTERM=truecolor in addition
// to the caller's Env slice and the current process environment.
// On POSIX the process is placed in its own process group (Setpgid: true).
func (b *NativePTYBackend) Create(ctx context.Context, req CreateRequest) (*Session, error) {
	p, err := gopty.New()
	if err != nil {
		return nil, fmt.Errorf("open pty: %w", err)
	}

	childCtx, cancel := context.WithCancel(ctx)
	cmd := p.CommandContext(childCtx, req.CLI, req.Args...)

	// Merge environment: inherit from current process, apply caller extras, enforce
	// required terminal vars.
	cmd.Env = mergeEnv(os.Environ(), req.Env, "TERM=xterm-256color", "COLORTERM=truecolor")

	// On POSIX, go-pty's start() already sets Setsid: true which creates a new
	// session — the child's process group ID equals its PID. We do NOT set
	// Setpgid here because combining Setpgid+Setsid causes EPERM on macOS.
	// killSession uses -pid (negative) to kill the entire process group.

	if err := cmd.Start(); err != nil {
		cancel()
		_ = p.Close()
		return nil, fmt.Errorf("start process: %w", err)
	}

	// Set initial terminal dimensions. On Windows this may silently fail if
	// ConPTY is not fully initialised yet; log but do not abort.
	if err := p.Resize(req.Cols, req.Rows); err != nil {
		if runtime.GOOS == "windows" {
			log.Printf("[warn] pty resize on Windows after Start: %v", err)
		} else {
			cancel()
			_ = p.Close()
			return nil, fmt.Errorf("initial resize: %w", err)
		}
	}

	id, err := generateID()
	if err != nil {
		cancel()
		_ = p.Close()
		return nil, fmt.Errorf("generate session ID: %w", err)
	}

	sess := &Session{
		ID:        id,
		CLI:       req.CLI,
		State:     StateRunning,
		CreatedAt: time.Now(),
		pty:       p,
		cmd:       cmd,
		cancel:    cancel,
	}

	// On Windows, assign the process to a Job Object for reliable cleanup.
	// On POSIX this is a no-op; process groups handle cleanup instead.
	if cmd.Process != nil {
		assignJobObject(sess, cmd.Process)
	}

	b.registry.Add(sess)
	return sess, nil
}

// Resize changes the terminal dimensions for the given session.
func (b *NativePTYBackend) Resize(id string, cols, rows int) error {
	sess, ok := b.registry.Get(id)
	if !ok {
		return ErrSessionNotFound
	}
	sess.mu.Lock()
	p := sess.pty
	sess.mu.Unlock()
	if p == nil {
		return fmt.Errorf("session %s: PTY not initialised", id)
	}
	return p.Resize(cols, rows)
}

// Kill terminates the process for the given session and removes it from the
// registry.
func (b *NativePTYBackend) Kill(id string) error {
	sess, ok := b.registry.Get(id)
	if !ok {
		return ErrSessionNotFound
	}

	sess.mu.Lock()
	sess.State = StateStopped
	sess.mu.Unlock()

	b.registry.Remove(id)
	return killSession(sess)
}

// List returns all sessions currently tracked by this backend.
func (b *NativePTYBackend) List() []*Session {
	return b.registry.List()
}

// generateID returns a cryptographically random 16-byte hex string suitable
// for use as a session identifier.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// mergeEnv merges base and extra environment slices, then overrides/adds the
// required key=value pairs. Later entries win: extra overrides base, required
// overrides both.
//
// Algorithm: build a map keyed by variable name, iterate in order so later
// assignments win, then serialise back to []string.
func mergeEnv(base []string, extra []string, required ...string) []string {
	// Preserve insertion order by tracking key order.
	order := make([]string, 0, len(base)+len(extra)+len(required))
	vals := make(map[string]string, len(base)+len(extra)+len(required))

	add := func(entry string) {
		k, v, _ := strings.Cut(entry, "=")
		if _, exists := vals[k]; !exists {
			order = append(order, k)
		}
		vals[k] = v
	}

	for _, e := range base {
		add(e)
	}
	for _, e := range extra {
		add(e)
	}
	for _, e := range required {
		add(e)
	}

	out := make([]string, 0, len(order))
	for _, k := range order {
		out = append(out, k+"="+vals[k])
	}
	return out
}
