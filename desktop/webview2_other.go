//go:build !windows

package main

// webview2RuntimeVersion is only meaningful on Windows.
func webview2RuntimeVersion() string { return "not-applicable" }
