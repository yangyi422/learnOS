package pathmap

import ("errors"; "os"; "path/filepath"; "testing")

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