package logger

import ("strings"; "testing")

func TestRedactMasksExplicitSecret(t *testing.T) {
	message := "token=0123456789abcdef0123456789abcdef path=C:\\Movies\\x.mkv"
	redacted := Redact(message, "0123456789abcdef0123456789abcdef")
	if strings.Contains(redacted, "0123456789abcdef") {
		t.Fatalf("secret leaked: %s", redacted)
	}
	if !strings.Contains(redacted, "C:\\Movies\\x.mkv") {
		t.Fatalf("path should stay readable: %s", redacted)
	}
}

func TestRedactMasksTokenFieldsButKeepsItemIDs(t *testing.T) {
	message := `Get "/Items/0f8fad5bd9cb469fa16570867728950e/Ancestors?UserId=9a1b2c3d4e5f60718293a4b5c6d7e8f9" with X-Emby-Token: deadbeefdeadbeefdeadbeefdeadbeef`
	redacted := Redact(message)
	if strings.Contains(redacted, "deadbeefdeadbeef") {
		t.Fatalf("token leaked: %s", redacted)
	}
	if !strings.Contains(redacted, "0f8fad5bd9cb469fa16570867728950e") {
		t.Fatalf("item id should stay readable: %s", redacted)
	}
}