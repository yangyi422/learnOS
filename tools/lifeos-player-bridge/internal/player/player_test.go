package player

import ("os"; "path/filepath"; "testing")

func TestDetectAcceptsExplicitExistingExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "PotPlayerMini64.exe")
	if err := os.WriteFile(path, []byte("stub"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Detect(path)
	if err != nil || got != path {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDetectRejectsMissingExecutable(t *testing.T) {
	if _, err := Detect(filepath.Join(t.TempDir(), "nope.exe")); err == nil {
		t.Fatal("expected missing executable to be rejected")
	}
}