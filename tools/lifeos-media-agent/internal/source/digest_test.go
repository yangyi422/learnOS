package source

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"testing"
)

func TestCalibreDigest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/books?library_id=a%20b", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = calibreAuthorization(req, `Digest realm="calibre", nonce="nonce", algorithm="MD5", qop="auth,auth-int"`, "reader", "password")
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{}
	for _, match := range digestParameter.FindAllStringSubmatch(req.Header.Get("Authorization"), -1) {
		fields[match[1]] = match[2] + match[3]
	}
	hash := func(s string) string { return fmt.Sprintf("%x", md5.Sum([]byte(s))) }
	expected := hash(hash("reader:calibre:password") + ":nonce:00000001:" + fields["cnonce"] + ":auth:" + hash("GET:/books?library_id=a%20b"))
	if fields["uri"] != "/books?library_id=a%20b" || fields["cnonce"] == "" || fields["response"] != expected {
		t.Fatal("incorrect Digest authorization")
	}
}
