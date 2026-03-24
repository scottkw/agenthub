package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
)

// daemonSvc adapts the daemon core to the kardianos/service.Interface.
// Start launches the daemon in a goroutine; Stop cancels its context.
type daemonSvc struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (d *daemonSvc) Start(s service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.done = make(chan struct{})
	go func() {
		defer close(d.done)
		runDaemonCore(ctx)
	}()
	return nil
}

func (d *daemonSvc) Stop(s service.Service) error {
	if d.cancel != nil {
		d.cancel()
	}
	if d.done != nil {
		<-d.done
	}
	return nil
}

// newServiceConfig builds the kardianos/service config for the current binary.
// UserService=true targets user scope: ~/Library/LaunchAgents (macOS),
// ~/.config/systemd/user (Linux), per-user SCM (Windows).
// RunAtLoad=true enables auto-start on login. KeepAlive=false allows
// manual stop without automatic restart (see RESEARCH.md Pitfall 4).
func newServiceConfig() (*service.Config, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return nil, fmt.Errorf("abs executable: %w", err)
	}
	return &service.Config{
		Name:        "agenthub-daemon",
		DisplayName: "AgentHub Daemon",
		Description: "AgentHub session manager daemon",
		Executable:  exe,
		Arguments:   []string{"daemon"},
		Option: service.KeyValue{
			"UserService": true,
			"RunAtLoad":   true,
			"KeepAlive":   false,
		},
	}, nil
}

// ServiceControl performs a service management action: install, uninstall, start, or stop.
// It creates a kardianos/service instance and delegates to service.Control.
func ServiceControl(action string) error {
	cfg, err := newServiceConfig()
	if err != nil {
		return err
	}
	prg := &daemonSvc{}
	s, err := service.New(prg, cfg)
	if err != nil {
		return fmt.Errorf("service.New: %w", err)
	}
	if err := service.Control(s, action); err != nil {
		return fmt.Errorf("daemon %s: %w", action, err)
	}
	return nil
}
