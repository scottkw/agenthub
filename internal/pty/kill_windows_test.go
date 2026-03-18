//go:build windows

package pty

import "errors"

// killSignal0 is not meaningful on Windows — always returns an error to
// indicate unsupported.
func killSignal0(pid int) error {
	return errors.New("killSignal0 not supported on Windows")
}
