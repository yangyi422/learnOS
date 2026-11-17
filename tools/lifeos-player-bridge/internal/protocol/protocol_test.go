package protocol

import "testing"

func TestParseAcceptsPlayOnly(t *testing.T) {
	uid := "0f8fad5b-d9cb-469f-a165-70867728950e"
	for _, raw := range []string{
		"lifeos-player://play?itemId=" + uid,
		"LIFEOS-PLAYER://play?itemId=" + uid,
		"lifeos-player://play?itemId=" + uid + "",
		"lifeos-player://play/?itemId=" + uid,
	} {
		request, err := Parse(raw)
		if err != nil || request.ItemID != uid {
			t.Fatalf("%q => %+v %v", raw, request, err)
		}
	}
}

func TestParseRejectsEverythingElse(t *testing.T) {
	cases := []string{
		"",
		"play",
		"http://play?itemId=0f8fad5bd9cb469fa16570867728950e",
		"lifeos-player://open?itemId=0f8fad5bd9cb469fa16570867728950e",
		"lifeos-player://play?path=C:\\\\Windows\\\\notepad.exe",
		"lifeos-player://play?itemId=0f8fad5bd9cb469fa16570867728950e&exec=calc.exe",
		"lifeos-player://play?itemId=0f8fad5bd9cb469fa16570867728950e&itemId=0f8fad5bd9cb469fa16570867728951e",
		"lifeos-player://play?itemId=../../../etc/passwd",
		"lifeos-player://play?itemId=0f8fad5b",
		"lifeos-player://play?itemId=zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		"lifeos-player://play?itemId=0f8fad5bd9cb469fa16570867728950e#frag",
	}
	for _, raw := range cases {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("%q should be rejected", raw)
		}
	}
}