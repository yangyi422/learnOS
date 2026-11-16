package source

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var digestParameter = regexp.MustCompile(`([A-Za-z]+)\s*=\s*(?:"([^"]*)"|([^,\s]+))`)

func calibreAuthorization(req *http.Request, challenge, username, password string) error {
	scheme, parameters, _ := strings.Cut(challenge, " ")
	if strings.EqualFold(scheme, "Basic") {
		req.SetBasicAuth(username, password)
		return nil
	}
	if !strings.EqualFold(scheme, "Digest") {
		return fmt.Errorf("unsupported Calibre authentication challenge")
	}
	fields := map[string]string{}
	for _, match := range digestParameter.FindAllStringSubmatch(parameters, -1) {
		fields[strings.ToLower(match[1])] = match[2] + match[3]
	}
	algorithm := fields["algorithm"]
	if fields["nonce"] == "" || (algorithm != "" && !strings.EqualFold(algorithm, "MD5")) {
		return fmt.Errorf("unsupported Calibre Digest parameters")
	}
	auth := false
	for _, qop := range strings.Split(fields["qop"], ",") {
		auth = auth || strings.TrimSpace(qop) == "auth"
	}
	if !auth {
		return fmt.Errorf("Calibre Digest must support qop=auth")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	cnonce := hex.EncodeToString(random[:])
	const nc = "00000001"
	uri := req.URL.RequestURI()
	hash := func(value string) string { return fmt.Sprintf("%x", md5.Sum([]byte(value))) }
	ha1 := hash(username + ":" + fields["realm"] + ":" + password)
	ha2 := hash(req.Method + ":" + uri)
	response := hash(strings.Join([]string{ha1, fields["nonce"], nc, cnonce, "auth", ha2}, ":"))
	header := fmt.Sprintf("Digest username=%q, realm=%q, nonce=%q, uri=%q, response=%q, algorithm=MD5, qop=auth, nc=%s, cnonce=%q",
		username, fields["realm"], fields["nonce"], uri, response, nc, cnonce)
	if opaque := fields["opaque"]; opaque != "" {
		header += fmt.Sprintf(", opaque=%q", opaque)
	}
	req.Header.Set("Authorization", header)
	return nil
}
