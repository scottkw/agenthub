// AgentHub — smoke-test binary for Phase 1 PTY Foundation.
//
// Demonstrates the full PTY session lifecycle:
//   1. Detect installed AI coding CLIs on PATH
//   2. Create a NativePTYBackend
//   3. Spawn a demo session (cat by default, or AGENTHUB_DEMO_CLI env var)
//   4. Relay PTY output → stdout and stdin → PTY (interactive)
//   5. On SIGINT/SIGTERM: kill all sessions and exit cleanly
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	pty "github.com/agenthub/agenthub/internal/pty"
)

func main() {
	fmt.Println("AgentHub v0.1.0-dev")
	fmt.Println()

	// 1. Detect AI coding CLIs installed on PATH.
	clis := pty.DetectCLIs()
	if len(clis) == 0 {
		fmt.Println("[warn] no supported AI coding CLIs detected on PATH")
		fmt.Println("       (claude, codex, gemini, or opencode not found)")
	} else {
		fmt.Printf("Detected %d CLI(s):\n", len(clis))
		for _, c := range clis {
			fmt.Printf("  %-12s  %s\n", c.Name, c.Path)
		}
	}
	fmt.Println()

	// 2. Create the PTY backend.
	backend := pty.NewNativePTYBackend()

	// 3. Determine demo CLI.
	demoCLI := os.Getenv("AGENTHUB_DEMO_CLI")
	if demoCLI == "" {
		demoCLI = "cat"
	}

	// 4. Set up context that cancels on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Spawning demo session: %q (80×24)\n", demoCLI)
	sess, err := backend.Create(ctx, pty.CreateRequest{
		CLI:  demoCLI,
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create session: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Session %s started (pid relay via PTY)\n", sess.ID[:8])
	fmt.Println("Type anything — output will be echoed. Press Ctrl-C to exit.")
	fmt.Println()

	// 5a. Drain PTY → stdout in background.
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		drainPTY(sess, os.Stdout)
	}()

	// 5b. Relay stdin → PTY in background.
	relayDone := make(chan struct{})
	go func() {
		defer close(relayDone)
		_, _ = io.Copy(sess, os.Stdin)
	}()

	// 6. Wait for signal or session exit.
	select {
	case <-ctx.Done():
		fmt.Println("\n[signal received — shutting down]")
	case <-drainDone:
		fmt.Println("\n[session exited]")
	}

	// 7. Kill the session and give goroutines a moment to finish.
	if err := backend.Kill(sess.ID); err != nil {
		fmt.Fprintf(os.Stderr, "warn: kill session: %v\n", err)
	}

	// Wait up to 2 seconds for drain goroutine to finish.
	select {
	case <-drainDone:
	case <-time.After(2 * time.Second):
	}

	fmt.Println("all sessions terminated")
	os.Exit(0)
}

// drainPTY reads output from the session until EOF or error, writing
// everything to dst.  It is meant to run in its own goroutine.
func drainPTY(sess *pty.Session, dst io.Writer) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.Read(buf)
		if n > 0 {
			_, _ = dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}
