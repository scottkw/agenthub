//go:build !windows

package pty

import (
	"os"
	"syscall"
	"testing"
)

func TestSession_Signal_NilCmd(t *testing.T) {
	s := &Session{ID: "test-nil-cmd", cmd: nil}
	err := s.Signal(syscall.SIGUSR2)
	if err == nil {
		t.Error("Signal on nil cmd: want error, got nil")
	}
}

func TestSession_Signal_NilProcess(t *testing.T) {
	s := &Session{ID: "test-nil"}
	err := s.Signal(os.Kill)
	if err == nil {
		t.Error("Signal on nil process: want error, got nil")
	}
}
