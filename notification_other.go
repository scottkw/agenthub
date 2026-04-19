//go:build !darwin

package main

// sendNotification is a no-op on non-macOS platforms.
func sendNotification(title, body string) {}
