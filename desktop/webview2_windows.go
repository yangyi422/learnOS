//go:build windows

package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

// webview2RuntimeVersion reports the installed WebView2 Evergreen runtime, or ""
// when it is missing. Windows 10/11 normally ship it through Edge updates, but
// stripped or Server installs do not, and Wails cannot open a window without it.
func webview2RuntimeVersion() string {
	const clientID = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	candidates := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + clientID},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + clientID},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + clientID},
	}
	for _, candidate := range candidates {
		key, err := registry.OpenKey(candidate.root, candidate.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		version, _, err := key.GetStringValue("pv")
		_ = key.Close()
		if err == nil && strings.TrimSpace(version) != "" {
			return strings.TrimSpace(version)
		}
	}
	return ""
}
