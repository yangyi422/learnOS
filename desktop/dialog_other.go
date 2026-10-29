//go:build !windows

package main

// fatalDialog is a no-op outside Windows; stderr already carries the message.
func fatalDialog(title, message string) {}
