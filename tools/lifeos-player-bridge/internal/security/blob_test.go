package security

import "testing"

func TestDecodeBlobHandlesCmdkeyUTF16(t *testing.T) {
	raw := []byte{0x74, 0, 0x6f, 0, 0x6b, 0, 0x65, 0, 0x6e, 0, 0, 0}
	if got := DecodeBlob(raw); got != "token" {
		t.Fatalf("got %q", got)
	}
}

func TestDecodeBlobKeepsUTF8AndTrims(t *testing.T) {
	if got := DecodeBlob([]byte("  abc123\n")); got != "abc123" {
		t.Fatalf("got %q", got)
	}
	if got := DecodeBlob(nil); got != "" {
		t.Fatalf("got %q", got)
	}
}