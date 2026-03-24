package daemon

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kardianos/service"
)

func TestNewServiceConfig_Fields(t *testing.T) {
	cfg, err := newServiceConfig()
	if err != nil {
		t.Fatalf("newServiceConfig: %v", err)
	}
	if cfg.Name != "agenthub-daemon" {
		t.Errorf("Name = %q, want %q", cfg.Name, "agenthub-daemon")
	}
	if cfg.DisplayName != "AgentHub Daemon" {
		t.Errorf("DisplayName = %q, want %q", cfg.DisplayName, "AgentHub Daemon")
	}
	if cfg.Description != "AgentHub session manager daemon" {
		t.Errorf("Description = %q, want %q", cfg.Description, "AgentHub session manager daemon")
	}
	if len(cfg.Arguments) != 1 || cfg.Arguments[0] != "daemon" {
		t.Errorf("Arguments = %v, want [daemon]", cfg.Arguments)
	}
	// UserService, RunAtLoad, KeepAlive
	if v, ok := cfg.Option["UserService"]; !ok || v != true {
		t.Errorf("UserService = %v, want true", v)
	}
	if v, ok := cfg.Option["RunAtLoad"]; !ok || v != true {
		t.Errorf("RunAtLoad = %v, want true", v)
	}
	if v, ok := cfg.Option["KeepAlive"]; !ok || v != false {
		t.Errorf("KeepAlive = %v, want false", v)
	}
}

func TestNewServiceConfig_AbsolutePath(t *testing.T) {
	cfg, err := newServiceConfig()
	if err != nil {
		t.Fatalf("newServiceConfig: %v", err)
	}
	if !filepath.IsAbs(cfg.Executable) {
		t.Errorf("Executable = %q, want absolute path", cfg.Executable)
	}
}

func TestDaemonSvc_ImplementsInterface(t *testing.T) {
	// Compile-time check that daemonSvc satisfies service.Interface.
	var _ service.Interface = (*daemonSvc)(nil)
}

func TestDaemonSvc_StopNilCancel(t *testing.T) {
	// Stop with nil cancel should not panic.
	d := &daemonSvc{}
	if err := d.Stop(nil); err != nil {
		t.Errorf("Stop with nil cancel: %v", err)
	}
}

func TestServiceControl_Exported(t *testing.T) {
	// Verify ServiceControl is callable (compile-time check).
	// Actual call would fail without a real service manager.
	var _ func(string) error = ServiceControl
}

func TestRunDaemonCore_CancelledContext(t *testing.T) {
	// runDaemonCore with an already-cancelled context should return quickly
	// (it will fail to bind socket or return on ctx.Done).
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDaemonCore(ctx)
	}()
	select {
	case <-done:
		// OK — returned quickly
	case <-time.After(5 * time.Second):
		t.Fatal("runDaemonCore did not return within 5s on cancelled context")
	}
}
