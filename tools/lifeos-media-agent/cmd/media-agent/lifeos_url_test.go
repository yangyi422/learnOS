package main

import "testing"

func TestLifeOSURL(t *testing.T) {
	for _, test := range []struct {
		url     string
		allowed bool
	}{
		{"https://lifeos.example:8443/base", true},
		{"http://127.0.0.1:8888", true},
		{"http://localhost:8888", true},
		{"http://[::1]:8888", true},
		{"http://192.0.2.1:8888", false},
		{"ftp://localhost", false},
		{"https://user:secret@lifeos.example", false},
		{"https://lifeos.example?key=secret", false},
		{"", false},
	} {
		if _, err := validateLifeOSURL(test.url); (err == nil) != test.allowed {
			t.Errorf("unexpected validation result for %q", test.url)
		}
	}
}
