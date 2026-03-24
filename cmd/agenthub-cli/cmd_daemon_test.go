package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
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
