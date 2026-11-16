package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

func lifeOSURL() (*url.URL, error) {
	return validateLifeOSURL(os.Getenv("LIFEOS_URL"))
}

func validateLifeOSURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("LIFEOS_URL is missing")
	}
	base, err := url.Parse(raw)
	if err != nil || base.Hostname() == "" || base.User != nil || base.ForceQuery || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("LIFEOS_URL must be a base URL without credentials, query or fragment")
	}
	switch base.Scheme {
	case "https":
		return base, nil
	case "http":
		switch strings.ToLower(base.Hostname()) {
		case "127.0.0.1", "localhost", "::1":
			return base, nil
		}
	}
	return nil, fmt.Errorf("LIFEOS_URL must use HTTPS or loopback HTTP")
}
