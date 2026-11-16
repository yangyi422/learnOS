package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchAgentPlist(t *testing.T) {
	t.Setenv("LIFEOS_URL", "https://fixture.example")
	t.Setenv("CALIBRE_ALL_BOOKS_HREF", "/books?a=b&c=d")
	plist := launchPlist("/Users/fixture/Media Agent/bin/media-agent", "/Users/fixture/Logs/sync.log")
	for _, want := range []string{"<integer>3</integer>", "<integer>0</integer>", "https://fixture.example", "a=b&amp;c=d", "/Users/fixture/Media Agent/bin/media-agent"} {
		if !strings.Contains(plist, want) {
			t.Fatalf("plist missing %q", want)
		}
	}
	path := filepath.Join(t.TempDir(), "agent.plist")
	if err := os.WriteFile(path, []byte(plist), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("/usr/bin/plutil", "-lint", path).CombinedOutput(); err != nil {
		t.Fatalf("invalid plist: %s", output)
	}
}
