//go:build windows

package main

// initTray is a no-op on Windows — system tray support is macOS-only.
func (a *App) initTray() {}

// cleanupTray is a no-op on Windows — system tray support is macOS-only.
func (a *App) cleanupTray() {}

// updateTray is a no-op on Windows — system tray support is macOS-only.
func (a *App) updateTray(sessions []SessionInfo, connected bool) {}
