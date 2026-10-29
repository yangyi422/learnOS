// Package bridge serves the LifeOS desktop shell from a loopback origin: the
// embedded Vue build plus a strict reverse proxy to the remote LifeOS API.
//
// It has no Wails dependency so its behaviour can be unit tested on any OS.
package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"
)

const APIprefix = "/api/v1/"

// proxyMethods is deliberately closed: the bridge is a client for the LifeOS
// API, never a general purpose proxy.
var proxyMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodHead:   true,
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// Bridge owns the loopback listener, the embedded frontend and the API proxy.
type Bridge struct {
	listener   net.Listener
	server     *http.Server
	target     *url.URL
	assets     fs.FS
	origin     string
	startupErr error
	proxy      *httputil.ReverseProxy
}

// Start serves dist on 127.0.0.1:<random port> and proxies /api/v1/* to the
// configured upstream. Serving the page and the API from one loopback origin
// keeps relative URLs, HttpOnly cookies, SameSite=Lax, the 401 redirect and
// history routing working exactly as they do in a browser tab.
func Start(cfg Config, dist fs.FS) (*Bridge, error) {
	target, err := url.Parse(cfg.APIBase)
	if err != nil {
		return nil, fmt.Errorf("parse api base: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}

	b := &Bridge{
		listener: listener,
		target:   target,
		assets:   dist,
		origin:   "http://" + listener.Addr().String(),
	}
	b.proxy = b.newProxy()
	b.serve()
	log.Printf("bridge listening on %s → %s (config: %s)", b.origin, b.target.Redacted(), cfg.Source)
	return b, nil
}

// StartError shows a readable page in the window when configuration is
// unusable, instead of exiting silently behind a GUI subsystem binary.
func StartError(cause error) (*Bridge, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	b := &Bridge{
		listener:   listener,
		origin:     "http://" + listener.Addr().String(),
		startupErr: cause,
	}
	b.serve()
	return b, nil
}

func (b *Bridge) serve() {
	b.server = &http.Server{
		Handler:           b.guard(http.HandlerFunc(b.route)),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
		ErrorLog:          log.New(logWriter{}, "", 0),
	}
	go func() {
		if err := b.server.Serve(b.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("bridge server stopped: %v", err)
		}
	}()
}

func (b *Bridge) URL() string { return b.origin + "/" }

func (b *Bridge) Close() {
	if b.server != nil {
		_ = b.server.Close()
	}
}

func (b *Bridge) route(w http.ResponseWriter, r *http.Request) {
	if b.startupErr != nil {
		if r.URL.Path == "/healthz" {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"status": "error",
				"error":  b.startupErr.Error(),
			})
			return
		}
		b.serveStartupError(w)
		return
	}

	switch {
	case r.URL.Path == "/healthz":
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"upstream": b.target.Redacted(),
		})
	case strings.HasPrefix(r.URL.Path, "/api/"):
		b.handleAPI(w, r)
	default:
		b.serveSPA(w, r)
	}
}

func (b *Bridge) handleAPI(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, APIprefix) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "unknown api path"})
		return
	}
	if !proxyMethods[r.Method] {
		w.Header().Set("Allow", "GET, HEAD, POST, PUT, PATCH, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	b.proxy.ServeHTTP(w, r)
}

func (b *Bridge) serveSPA(w http.ResponseWriter, r *http.Request) {
	// Vue Router uses history mode: unknown non-file paths render index.html.
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" || name == "." {
		name = "index.html"
	}
	if info, err := fs.Stat(b.assets, name); err == nil && !info.IsDir() {
		b.setAssetCache(w, name)
		http.ServeFileFS(w, r, b.assets, name)
		return
	}
	if path.Ext(name) != "" {
		http.NotFound(w, r)
		return
	}
	b.setAssetCache(w, "index.html")
	http.ServeFileFS(w, r, b.assets, "index.html")
}

func (b *Bridge) setAssetCache(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

func (b *Bridge) serveStartupError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">
<title>LifeOS 无法启动</title><style>
:root{color-scheme:light dark}body{margin:0;min-height:100vh;display:grid;place-items:center;
font:15px/1.7 "PingFang SC","Microsoft YaHei",system-ui,sans-serif;background:#eef4fc;color:#172840}
main{max-width:520px;padding:32px;border:1px solid rgba(74,96,122,.16);border-radius:16px;background:rgba(255,255,255,.7)}
h1{margin:0 0 12px;font-size:20px;font-weight:500}p{margin:0 0 8px;color:#384f69}
code{font-size:13px;color:#b42318;word-break:break-all}
</style></head><body><main><h1>LifeOS 无法连接到服务</h1>
<p>请检查 <code>desktop.config.json</code> 中的 <code>apiBase</code>，或设置环境变量 <code>LIFEOS_API_BASE</code>。</p>
<p><code>%s</code></p></main></body></html>`, html.EscapeString(b.startupErr.Error()))
}

func (b *Bridge) newProxy() *httputil.ReverseProxy {
	transport := &http.Transport{
		Proxy:                 nil,
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
	}
	return &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(b.target)
			pr.Out.Host = b.target.Host
			pr.SetXForwarded()
			pr.Out.Header.Set("X-Forwarded-Proto", b.target.Scheme)
		},
		ModifyResponse: func(res *http.Response) error {
			rewriteCookies(res.Header)
			// Client-side hangs are otherwise invisible: log every proxied call
			// with its status and duration so the desktop log shows which request
			// the page is waiting on.
			if started, ok := res.Request.Context().Value(startedKey{}).(time.Time); ok {
				log.Printf("proxy %s %s → %d (%s)", res.Request.Method, res.Request.URL.Path, res.StatusCode, time.Since(started).Round(time.Millisecond))
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy %s %s failed after %s: %v", r.Method, r.URL.Path, sinceStart(r.Context()), err)
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": map[string]any{
					"code":      "UPSTREAM_UNAVAILABLE",
					"message":   "无法连接 LifeOS 服务，请确认服务器可访问后重试。",
					"retryable": true,
				},
			})
		},
	}
}

// rewriteCookies keeps the session cookie storable on the loopback origin:
// a Domain attribute from the remote host would be rejected as a domain
// mismatch, and Secure cannot be honoured over http://127.0.0.1.
// HttpOnly, SameSite, Path, Expires and Max-Age pass through untouched.
func rewriteCookies(header http.Header) {
	values := header.Values("Set-Cookie")
	if len(values) == 0 {
		return
	}
	header.Del("Set-Cookie")
	for _, value := range values {
		parts := strings.Split(value, ";")
		kept := []string{strings.TrimSpace(parts[0])}
		for _, attribute := range parts[1:] {
			name := strings.ToLower(strings.TrimSpace(strings.SplitN(attribute, "=", 2)[0]))
			if name == "domain" || name == "secure" {
				continue
			}
			kept = append(kept, strings.TrimSpace(attribute))
		}
		header.Add("Set-Cookie", strings.Join(kept, "; "))
	}
}

// guard rejects anything that is not this loopback origin, so other sites
// cannot drive the local service (DNS rebinding / cross-site requests).
func (b *Bridge) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !b.hostAllowed(r.Host) {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !b.originAllowed(origin) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (b *Bridge) hostAllowed(host string) bool {
	name, port, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}
	if port != b.port() {
		return false
	}
	switch name {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

func (b *Bridge) originAllowed(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" {
		return false
	}
	return b.hostAllowed(parsed.Host)
}

func (b *Bridge) port() string {
	_, port, err := net.SplitHostPort(b.listener.Addr().String())
	if err != nil {
		return ""
	}
	return port
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// startedKey carries the request start time from the handler into the proxy so
// timing can be logged without another middleware layer.
type startedKey struct{}

func sinceStart(ctx context.Context) time.Duration {
	if started, ok := ctx.Value(startedKey{}).(time.Time); ok {
		return time.Since(started).Round(time.Millisecond)
	}
	return 0
}

// logWriter routes net/http's internal error log through the standard logger.
type logWriter struct{}

func (logWriter) Write(p []byte) (int, error) {
	log.Printf("http: %s", strings.TrimSpace(string(p)))
	return len(p), nil
}
