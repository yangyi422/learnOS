package bridge

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestBridgeAgainstRealUpstream exercises the whole loopback path against a
// real LifeOS deployment: SPA delivery, history fallback and the API proxy with
// its 401 passthrough. It is opt-in because it needs network access:
//
//	LIFEOS_E2E_UPSTREAM=http://43.138.202.56:8888 go test ./internal/bridge -run RealUpstream -v
func TestBridgeAgainstRealUpstream(t *testing.T) {
	upstream := os.Getenv("LIFEOS_E2E_UPSTREAM")
	if upstream == "" {
		t.Skip("set LIFEOS_E2E_UPSTREAM to run the live proxy check")
	}
	base, err := NormalizeAPIBase(upstream)
	if err != nil {
		t.Fatalf("upstream: %v", err)
	}

	dist := fstest.MapFS{
		"index.html":     {Data: []byte("<!doctype html><title>LifeOS shell</title>")},
		"assets/app.css": {Data: []byte("body{}")},
	}
	b, err := Start(Config{APIBase: base, Source: "test"}, dist)
	if err != nil {
		t.Fatalf("start bridge: %v", err)
	}
	defer b.Close()

	client := &http.Client{Timeout: 20 * time.Second}
	get := func(path string) (*http.Response, string) {
		t.Helper()
		res, err := client.Get(b.origin + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return res, string(body)
	}

	if res, body := get("/healthz"); res.StatusCode != http.StatusOK || !strings.Contains(body, `"ok"`) {
		t.Fatalf("healthz: %d %s", res.StatusCode, body)
	}
	if res, body := get("/"); res.StatusCode != http.StatusOK || !strings.Contains(body, "LifeOS shell") {
		t.Fatalf("index: %d %s", res.StatusCode, body)
	}
	if res, body := get("/projects"); res.StatusCode != http.StatusOK || !strings.Contains(body, "LifeOS shell") {
		t.Fatalf("history fallback: %d %s", res.StatusCode, body)
	}
	if res, _ := get("/assets/app.css"); res.StatusCode != http.StatusOK ||
		!strings.Contains(res.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("asset caching: %d %q", res.StatusCode, res.Header.Get("Cache-Control"))
	}

	// Unauthenticated API access must reach the real server and come back as 401
	// JSON, not as a local error page.
	res, body := get("/api/v1/auth/me")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("api proxy status: %d %s", res.StatusCode, body)
	}
	if !strings.Contains(res.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("api proxy content-type: %q", res.Header.Get("Content-Type"))
	}
	if !strings.Contains(body, "error") {
		t.Fatalf("api proxy body: %s", body)
	}

	// A blocked path must never be proxied.
	if res, _ := get("/api/internal/secret"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected proxy of non-v1 path: %d", res.StatusCode)
	}
}
