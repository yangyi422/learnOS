package logger

import ("strings"; "testing")

func TestRedactMasksTokens(t *testing.T) {
	message := "token=0123456789abcdef0123456789abcdef path=C:\\Movies\\x.mkv"
	redacted := Redact(message, "0123456789abcdef0123456789abcdef")
	if strings.Contains(redacted, "0123456789abcdef") {
		t.Fatalf("secret leaked: %s", redacted)
	}
	if !strings.Contains(redacted, "C:\\Movies\\x.mkv") {
		t.Fatalf("path should stay readable: %s", redacted)
	}
}

func TestRedactMasksUnknownOpaqueStrings(t *testing.T) {
	redacted := Redact("header X-Emby-Token deadbeefdeadbeefdeadbeefdeadbeef")
	if strings.Contains(redacted, "deadbeefdeadbeef") {
		t.Fatalf("opaque token leaked: %s", redacted)
	}
}