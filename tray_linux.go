//go:build linux

package main

// initTray is a no-op on Linux — system tray support is macOS-only.
func (a *App) initTray() {}

// cleanupTray is a no-op on Linux — system tray support is macOS-only.
func (a *App) cleanupTray() {}
