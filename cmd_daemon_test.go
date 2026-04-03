package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/scottkw/agenthub/internal/daemon"
)

func TestCmdDaemon_ServiceActions(t *testing.T) {
	actions := []string{"install", "uninstall", "start", "stop"}
	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			var called string
			old := serviceControlFunc
			serviceControlFunc = func(a string) error {
				called = a
				return nil
			}
			defer func() { serviceControlFunc = old }()

			var buf bytes.Buffer
			err := cmdDaemon([]string{action}, &buf)
			if err != nil {
				t.Fatalf("cmdDaemon(%q) error: %v", action, err)
			}
			if called != action {
				t.Errorf("ServiceControl called with %q, want %q", called, action)
			}
			if !strings.Contains(buf.String(), action) {
				t.Errorf("output %q does not contain %q", buf.String(), action)
			}
		})
	}
}

func TestCmdDaemon_UnknownSubcommand(t *testing.T) {
	var buf bytes.Buffer
	err := cmdDaemon([]string{"bogus"}, &buf)
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error %q does not contain 'unknown'", err.Error())
	}
}

func TestCmdDaemon_ServiceControlError(t *testing.T) {
	old := serviceControlFunc
	serviceControlFunc = func(a string) error {
		return fmt.Errorf("mock error")
	}
	defer func() { serviceControlFunc = old }()

	var buf bytes.Buffer
	err := cmdDaemon([]string{"install"}, &buf)
	if err == nil {
		t.Fatal("expected error when ServiceControl fails")
	}
	if !strings.Contains(err.Error(), "mock error") {
		t.Errorf("error %q does not contain 'mock error'", err.Error())
	}
}

// TestCmdDaemon_Status verifies cmdDaemonStatus prints "running:" and "true" with reachable daemon.
func TestCmdDaemon_Status(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdDaemonStatus(client, nil, &buf)
	if err != nil {
		t.Fatalf("cmdDaemonStatus error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "running:") {
		t.Errorf("expected 'running:' in output, got %q", out)
	}
	if !strings.Contains(out, "true") {
		t.Errorf("expected 'true' in output, got %q", out)
	}
}

// TestCmdDaemon_Status_JSON verifies --json produces {"running":true} with reachable daemon.
func TestCmdDaemon_Status_JSON(t *testing.T) {
	client := testSetup(t)
	var buf bytes.Buffer
	err := cmdDaemonStatus(client, []string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdDaemonStatus --json error: %v", err)
	}
	var resp struct {
		Running bool `json:"running"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	if !resp.Running {
		t.Error("expected running=true for reachable daemon")
	}
}

// TestCmdDaemon_Status_Unreachable verifies cmdDaemonStatus prints "false" for unreachable daemon.
func TestCmdDaemon_Status_Unreachable(t *testing.T) {
	// Create client pointing to non-existent socket.
	client := daemon.NewDaemonClient("/tmp/nonexistent-aht-test.sock")
	var buf bytes.Buffer
	err := cmdDaemonStatus(client, nil, &buf)
	if err != nil {
		t.Fatalf("cmdDaemonStatus error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "false") {
		t.Errorf("expected 'false' for unreachable daemon, got %q", out)
	}
}

// TestCmdDaemon_Status_JSON_Unreachable verifies --json produces {"running":false} for unreachable daemon.
func TestCmdDaemon_Status_JSON_Unreachable(t *testing.T) {
	client := daemon.NewDaemonClient("/tmp/nonexistent-aht-test.sock")
	var buf bytes.Buffer
	err := cmdDaemonStatus(client, []string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdDaemonStatus --json error: %v", err)
	}
	var resp struct {
		Running bool `json:"running"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nraw: %s", err, buf.String())
	}
	if resp.Running {
		t.Error("expected running=false for unreachable daemon")
	}
}
