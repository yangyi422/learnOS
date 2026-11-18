package main

import (
	"os"
	"strings"
	"testing"

	"lifeos/player-bridge/internal/config"
)

// Opt-in, read-only verification with the real Windows credentials and SMB
// directory. Keep the normal unit suite independent of a particular NAS.
func TestVerifyUNCConfiguration(t *testing.T) {
	configPath := os.Getenv("PLAYER_BRIDGE_TEST_CONFIG")
	if configPath == "" {
		t.Skip("set PLAYER_BRIDGE_TEST_CONFIG to verify a real UNC configuration")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	unc := false
	for _, mapping := range cfg.PathMappings {
		unc = unc || strings.HasPrefix(mapping.Target, `\\`)
	}
	if !unc {
		t.Fatal("test configuration must include a UNC target")
	}
	log := func(string, string, ...string) {}
	if code := runVerify(configPath, log, log); code != exitOK {
		t.Fatalf("verify exit code: %d", code)
	}
}
