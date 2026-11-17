package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `
jellyfin:
  base_url: http://192.168.31.122:8096/
  user_id: 4d1f1c1b8f9a4e2f9c3a5d6e7f809102
path_mappings:
  - source: /media/movies/
    target: Z:\Movies
  - source: /media/tvshows/
    target: Y:\TVShows\
player:
  executable: auto
playback:
  check_file_exists: true
  allowed_extensions: [mkv, .MP4]
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadNormalizesAndValidates(t *testing.T) {
	cfg, err := Load(writeConfig(t, sample))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jellyfin.BaseURL != "http://192.168.31.122:8096" {
		t.Fatalf("base url %q", cfg.Jellyfin.BaseURL)
	}
	if cfg.Jellyfin.CredentialTarget != DefaultCredentialTarget {
		t.Fatalf("credential target %q", cfg.Jellyfin.CredentialTarget)
	}
	if cfg.PathMappings[0].Source != "/media/movies/" || cfg.PathMappings[0].Target != `Z:\Movies\` {
		t.Fatalf("mapping %+v", cfg.PathMappings[0])
	}
	if cfg.PathMappings[1].Target != `Y:\TVShows\` {
		t.Fatalf("mapping keeps separators: %+v", cfg.PathMappings[1])
	}
	if !cfg.Extensions()["mp4"] || !cfg.Extensions()["mkv"] {
		t.Fatalf("extensions %v", cfg.Extensions())
	}
}

func TestLoadRejectsUnsafeConfiguration(t *testing.T) {
	cases := map[string]string{
		"missing mapping": "jellyfin:\n  base_url: http://x\n  user_id: u\n",
		"ftp base url": "jellyfin:\n  base_url: ftp://x\n  user_id: u\npath_mappings:\n  - source: /media/\n    target: \"Z:\\\\\"\n",
		"missing user id": "jellyfin:\n  base_url: http://x\npath_mappings:\n  - source: /media/\n    target: \"Z:\\\\\"\n",
		"relative source": "jellyfin:\n  base_url: http://x\n  user_id: u\npath_mappings:\n  - source: media/\n    target: \"Z:\\\\\"\n",
		"posix target": "jellyfin:\n  base_url: http://x\n  user_id: u\npath_mappings:\n  - source: /media/\n    target: /tmp\n",
		"wrong player": "jellyfin:\n  base_url: http://x\n  user_id: u\nplayer:\n  type: vlc\npath_mappings:\n  - source: /media/\n    target: \"Z:\\\\\"\n",
	}
	for name, body := range cases {
		if _, err := Load(writeConfig(t, body)); err == nil {
			t.Fatalf("%s should be rejected", name)
		}
	}
}