package bridge

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// newTestBridge builds a bridge pointed at the given upstream without touching
// the embedded frontend, so proxy and guard behaviour can be asserted directly.
func newTestBridge(t *testing.T, upstream string) *Bridge {
	t.Helper()
	target, err := url.Parse(upstream)
	if err != nil {
		t.Fatalf("parse upstream: %v", err)
	}
	// A real loopback listener gives the bridge an address whose host:port the
	// guard accepts; the handler is invoked directly, so nothing is served.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	b := &Bridge{listener: ln, target: target, origin: "http://" + ln.Addr().String()}
	b.proxy = b.newProxy()
	b.server = &http.Server{Handler: b.guard(http.HandlerFunc(b.route))}
	return b
}

func TestGuardRejectsForeignHostAndOrigin(t *testing.T) {
	b := newTestBridge(t, "http://upstream.invalid:8888")
	handler := b.server.Handler
	origin := b.origin

	forbidden := []*http.Request{
		httptest.NewRequest(http.MethodGet, "http://evil.example/healthz", nil),
		httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/healthz", nil),
	}
	req := httptest.NewRequest(http.MethodPost, origin+"/api/v1/auth/login", strings.NewReader("{}"))
	req.Header.Set("Origin", "http://evil.example")
	forbidden = append(forbidden, req)

	for i, request := range forbidden {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("case %d: expected 403, got %d", i, recorder.Code)
		}
	}

	allowed := httptest.NewRequest(http.MethodGet, origin+"/healthz", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, allowed)
	if recorder.Code != http.StatusOK {
		t.Fatalf("loopback healthz should pass, got %d", recorder.Code)
	}
}

func TestProxyRejectsPathsOutsideAPIAndUnknownMethods(t *testing.T) {
	b := newTestBridge(t, "http://upstream.invalid:8888")
	handler := b.server.Handler

	outside := httptest.NewRequest(http.MethodGet, b.origin+"/api/other/thing", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, outside)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("non-v1 API path must not be proxied, got %d", recorder.Code)
	}

	forbiddenMethod := httptest.NewRequest(http.MethodTrace, b.origin+"/api/v1/auth/me", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, forbiddenMethod)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("TRACE must be rejected, got %d", recorder.Code)
	}
}

func TestProxyForwardsBodyQueryAndRewritesSessionCookie(t *testing.T) {
	var (
		gotPath  string
		gotQuery string
		gotBody  string
		gotAuth  string
	)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Cookie")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Add("Set-Cookie", "learnos_session=abc123; Path=/; Domain=43.138.202.56; HttpOnly; Secure; SameSite=Lax; Max-Age=604800")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer upstream.Close()

	b := newTestBridge(t, upstream.URL)
	body := `{"username":"demo"}`
	req := httptest.NewRequest(http.MethodPost, b.origin+"/api/v1/auth/login?next=%2F", strings.NewReader(body))
	req.Header.Set("Cookie", "learnos_session=previous")
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	b.server.Handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status not forwarded: %d", recorder.Code)
	}
	if gotPath != "/api/v1/auth/login" || gotQuery != "next=%2F" || gotBody != body {
		t.Fatalf("request not forwarded intact: path=%q query=%q body=%q", gotPath, gotQuery, gotBody)
	}
	if !strings.Contains(gotAuth, "learnos_session=previous") {
		t.Fatalf("cookie not forwarded: %q", gotAuth)
	}

	cookie := recorder.Header().Get("Set-Cookie")
	for _, banned := range []string{"Domain=", "Secure"} {
		if strings.Contains(cookie, banned) {
			t.Fatalf("cookie kept %s, which the loopback origin cannot honour: %q", banned, cookie)
		}
	}
	for _, kept := range []string{"learnos_session=abc123", "HttpOnly", "SameSite=Lax", "Path=/", "Max-Age=604800"} {
		if !strings.Contains(cookie, kept) {
			t.Fatalf("cookie lost %s: %q", kept, cookie)
		}
	}
	if recorder.Body.String() != `{"data":{"ok":true}}` {
		t.Fatalf("body not forwarded: %q", recorder.Body.String())
	}
}

func TestConfigNormalization(t *testing.T) {
	valid := map[string]string{
		"http://43.138.202.56:8888":  "http://43.138.202.56:8888",
		" https://learnos.example/ ": "https://learnos.example",
	}
	for input, want := range valid {
		got, err := NormalizeAPIBase(input)
		if err != nil || got != want {
			t.Fatalf("normalize(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	invalid := []string{"", "ftp://host", "http://", "http://host/api", "http://user:pass@host", "http://host/?a=1"}
	for _, input := range invalid {
		if got, err := NormalizeAPIBase(input); err == nil {
			t.Fatalf("normalize(%q) should fail, got %q", input, got)
		}
	}
}

func TestLoadConfigUsesEmbeddedDefaultAndEnvOverride(t *testing.T) {
	embedded := []byte(`{"apiBase":"http://embedded.example:8888"}`)

	// No file next to the test binary or in the package dir, so the embedded
	// default is the effective source: this is what makes a single .exe enough.
	t.Setenv("LIFEOS_API_BASE", "")
	cfg, err := LoadConfig(embedded)
	if err != nil {
		t.Fatalf("embedded config: %v", err)
	}
	if cfg.APIBase != "http://embedded.example:8888" || cfg.Source != "embedded default" {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	// The environment still wins, so another machine can point elsewhere.
	t.Setenv("LIFEOS_API_BASE", "https://other.example")
	cfg, err = LoadConfig(embedded)
	if err != nil || cfg.APIBase != "https://other.example" {
		t.Fatalf("env override: %+v %v", cfg, err)
	}
}
