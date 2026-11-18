package pathmap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapWindowsTargets(t *testing.T) {
	for _, tc := range []struct{ target, want string }{
		{`Z:\Movies\`, `Z:\Movies\sub dir\中文 名称 🎬.MP4`},
		{`Z:/Movies//`, `Z:\Movies\sub dir\中文 名称 🎬.MP4`},
		{`\\192.168.31.122\share\Movies`, `\\192.168.31.122\share\Movies\sub dir\中文 名称 🎬.MP4`},
		{`\\nas\共享\`, `\\nas\共享\sub dir\中文 名称 🎬.MP4`},
		{`//nas/share/电影 目录/`, `\\nas\share\电影 目录\sub dir\中文 名称 🎬.MP4`},
	} {
		t.Run(tc.target, func(t *testing.T) {
			m := New([]Mapping{{Source: "/media/movies", Target: tc.target}}, map[string]bool{"mp4": true}, false)
			got, err := m.Map("/media/movies//sub dir/中文 名称 🎬.MP4")
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestMapRejectsTraversalBeforeCleaning(t *testing.T) {
	for _, target := range []string{`Z:\Movies`, `\\nas\share\Movies`} {
		m := New([]Mapping{{Source: "/media/movies/", Target: target}}, nil, false)
		for _, input := range []string{
			`/media/movies/sub/../film.mkv`, `/media/movies/../movies/film.mkv`,
			`/other/../media/movies/film.mkv`, `/media/movies/sub\..\film.mkv`,
			`/media/movies/.. /film.mkv`, `/media/movies/film.mkv:stream`,
			"/media/movies/bad\x00.mkv",
		} {
			if _, err := m.Map(input); !errors.Is(err, ErrTraversal) {
				t.Fatalf("%q -> %q: expected traversal rejection, got %v", input, target, err)
			}
		}
		for _, input := range []string{`/media/movies2/a.mkv`, `/media/movies`, `/media/movies/`, `/media/movies-other/a.mkv`} {
			if _, err := m.Map(input); !errors.Is(err, ErrNoMapping) {
				t.Fatalf("%q: expected no mapping, got %v", input, err)
			}
		}
	}
}

func TestMapCaseInsensitiveRootAndUnicodeSource(t *testing.T) {
	m := New([]Mapping{{Source: "/MEDIA/电影", Target: `\\nas\share`}}, nil, false)
	got, err := m.Map("/media/电影/影片.mkv")
	if err != nil || got != `\\nas\share\影片.mkv` {
		t.Fatalf("%q %v", got, err)
	}
}

func TestMapperRejectsInvalidConfiguration(t *testing.T) {
	for _, mapping := range []Mapping{
		{Source: "/media/../movies", Target: `Z:\Movies`},
		{Source: "/media/movies", Target: `\\nas`},
		{Source: "/media/movies", Target: `\\nas\share\..\Other`},
	} {
		if _, err := New([]Mapping{mapping}, nil, false).Map("/media/movies/a.mkv"); err == nil {
			t.Fatalf("invalid mapping accepted: %+v", mapping)
		}
	}
}

func TestCheckTargetDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := CheckTargetDirectory(dir); err != nil {
		t.Fatalf("empty directory: %v", err)
	}
	file := filepath.Join(dir, "中文 文件.mkv")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckTargetDirectory(dir); err != nil {
		t.Fatalf("readable directory: %v", err)
	}
	for _, target := range []string{file, filepath.Join(dir, "missing")} {
		if err := CheckTargetDirectory(target); err == nil {
			t.Fatalf("non-directory accepted: %q", target)
		}
	}
}

// Optional read-only integration test against a real authenticated SMB share.
func TestUNCShareAccessible(t *testing.T) {
	target := os.Getenv("PLAYER_BRIDGE_TEST_UNC")
	if target == "" {
		t.Skip("set PLAYER_BRIDGE_TEST_UNC to test a real SMB directory")
	}
	root, err := NormalizeTarget(target)
	if err != nil || !strings.HasPrefix(root, `\\`) {
		t.Fatalf("expected UNC target: %q, %v", root, err)
	}
	if err := CheckTargetDirectory(root); err != nil {
		t.Fatal(err)
	}
}

func mapper(checkExists bool) *Mapper {
	return New(
		[]Mapping{{Source: "/media/movies/", Target: "Z:\\Movies\\"}, {Source: "/media/tvshows/", Target: "Z:\\TVShows\\"}},
		map[string]bool{"mkv": true, "mp4": true},
		checkExists,
	)
}

func TestMapUsesPrefixBoundaryAndCaseInsensitiveRoots(t *testing.T) {
	got, err := mapper(false).Map("/media/movies/Dune.2021.2160p.mkv")
	if err != nil || got != "Z:\\Movies\\Dune.2021.2160p.mkv" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = mapper(false).Map("/media/movies/sub dir/中文 名称.MP4")
	if err != nil || got != "Z:\\Movies\\sub dir\\中文 名称.MP4" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestMapRejectsEscapesAndUnknownRoots(t *testing.T) {
	for _, containerPath := range []string{
		"/media/movies2/a.mkv",
		"/media/movies/../../windows/system32/calc.exe",
		"/etc/passwd",
		"Z:\\\\Movies\\\\a.mkv",
		"",
	} {
		if _, err := mapper(false).Map(containerPath); err == nil {
			t.Fatalf("%q should be rejected", containerPath)
		}
	}
}

func TestMapRejectsDisallowedExtension(t *testing.T) {
	if _, err := mapper(false).Map("/media/movies/setup.exe"); !errors.Is(err, ErrExtension) {
		t.Fatalf("expected extension rejection, got %v", err)
	}
}

func TestMapReportsMissingFileAndDrive(t *testing.T) {
	if _, err := mapper(true).Map("/media/movies/absent.mkv"); !errors.Is(err, ErrDriveUnavailable) && !errors.Is(err, ErrFileMissing) {
		t.Fatalf("expected a clear failure, got %v", err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "Dune.mkv")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	local := New([]Mapping{{Source: "/m/", Target: dir + "\\"}}, map[string]bool{"mkv": true}, true)
	if _, err := local.Map("/m/Dune.mkv"); err != nil {
		t.Fatalf("existing file should map: %v", err)
	}
}
