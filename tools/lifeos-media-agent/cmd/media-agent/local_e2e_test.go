package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"lifeos-media-agent/internal/source"
)

// TestLocalMediaPipeline uses the real LifeOS server running with an isolated
// development database. Only the two NAS HTTP endpoints and Keychain are fixtures.
func TestLocalMediaPipeline(t *testing.T) {
	base := os.Getenv("MEDIA_E2E_BASE_URL")
	if base == "" {
		t.Skip("requires an isolated LifeOS development server")
	}
	ctx := context.Background()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	login, err := client.Post(base+"/api/v1/auth/login", "application/json", strings.NewReader(`{"username":"fixture","password":"fixture-local-only"}`))
	if err != nil {
		t.Fatal(err)
	}
	login.Body.Close()
	if login.StatusCode != 200 {
		t.Fatalf("fixture login failed: %d", login.StatusCode)
	}
	tokenRequest, err := client.Post(base+"/api/v1/api-tokens", "application/json", strings.NewReader(`{"name":"local media fixture","scope":"media:sync","expires_in_days":1}`))
	if err != nil {
		t.Fatal(err)
	}
	var credential struct{ Data struct{ Token string } }
	err = json.NewDecoder(tokenRequest.Body).Decode(&credential)
	tokenRequest.Body.Close()
	if err != nil || tokenRequest.StatusCode != 201 || credential.Data.Token == "" {
		t.Fatalf("cannot issue fixture token: status %d", tokenRequest.StatusCode)
	}
	previous := keychain
	keychain = func(_ context.Context, service string) (string, error) {
		if service == "lifeos-media-agent-lifeos-token" {
			return credential.Data.Token, nil
		}
		if service == "lifeos-media-agent-calibre-username" {
			return "", fmt.Errorf("Calibre fixture does not require authentication")
		}
		return "fixture", nil
	}
	t.Cleanup(func() { keychain = previous })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CALIBRE_USERNAME", "")
	t.Setenv("LIFEOS_URL", base)
	var revision, replaceCover atomic.Bool
	mixed := os.Getenv("MEDIA_E2E_MIXED_COVERS") == "1"
	webp, err := os.ReadFile("testdata/cover.webp")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/opds":
			fmt.Fprint(w, `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>按最新</title><link type="application/atom+xml" href="/books?page=1"/></entry></feed>`)
		case "/books":
			first, last := 1, 3
			fmt.Fprint(w, `<feed xmlns="http://www.w3.org/2005/Atom">`)
			if r.URL.Query().Get("page") == "2" {
				first, last = 4, 5
			} else {
				fmt.Fprint(w, `<link rel="next" href="?page=2"/>`)
			}
			for i := first; i <= last; i++ {
				title := fmt.Sprintf("中文电子书 %d", i)
				if i == 1 && revision.Load() {
					title += " 修订版"
				}
				fmt.Fprintf(w, `<entry><id>book-%d</id><title>%s</title><author><name>林一</name></author><author><name>张二</name></author><category term="阅读"/><category term="测试"/><link rel="http://opds-spec.org/acquisition" type="application/epub+zip"/><link rel="http://opds-spec.org/acquisition" type="application/pdf"/>`, i, title)
				if i != 5 {
					fmt.Fprintf(w, `<link rel="http://opds-spec.org/image" href="/cover/%d"/>`, i)
				}
				fmt.Fprint(w, "</entry>")
			}
			fmt.Fprint(w, "</feed>")
		case "/Library/VirtualFolders":
			fmt.Fprint(w, `[{"Name":"电影","CollectionType":"movies","ItemId":"movies"},{"Name":"电视剧","CollectionType":"tvshows","ItemId":"series"},{"Name":"R18","CollectionType":"movies","ItemId":"excluded"}]`)
		case "/Items":
			parent := r.URL.Query().Get("ParentId")
			kind, label, count := "Movie", "电影", 5
			switch parent {
			case "series":
				kind, label, count = "Series", "电视剧", 2
			case "movies":
			default:
				t.Errorf("unexpected library %s", parent)
				w.WriteHeader(400)
				return
			}
			start, _ := strconv.Atoi(r.URL.Query().Get("StartIndex"))
			items := []map[string]any{}
			for i := start; i < count && i < start+2; i++ {
				overview := "测试简介"
				if kind == "Movie" && i == 0 && revision.Load() {
					overview = "更新后的电影简介"
				}
				items = append(items, map[string]any{"Id": fmt.Sprintf("%s%d", kind, i), "Name": fmt.Sprintf("%s %d", label, i+1), "Type": kind, "Overview": overview, "ProductionYear": 2000 + i, "Genres": []string{"剧情"}, "ImageTags": map[string]string{"Primary": "fixture"}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"TotalRecordCount": count, "Items": items})
		default:
			value := 0
			if strings.HasPrefix(r.URL.Path, "/cover/") {
				value, _ = strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/cover/"))
			} else if strings.HasPrefix(r.URL.Path, "/Items/Movie") {
				value = 10 + int(r.URL.Path[len("/Items/Movie")]-'0')
			} else if strings.HasPrefix(r.URL.Path, "/Items/Series") {
				value = 20 + int(r.URL.Path[len("/Items/Series")]-'0')
			}
			if value == 0 {
				http.NotFound(w, r)
				return
			}
			if mixed && value == 2 {
				w.Header().Set("Content-Type", "image/webp")
				_, _ = w.Write(webp)
				return
			}
			if replaceCover.Load() && value == 1 {
				value = 31
			}
			img := image.NewRGBA(image.Rect(0, 0, 60, 90))
			for y := 0; y < 90; y++ {
				for x := 0; x < 60; x++ {
					img.SetRGBA(x, y, color.RGBA{R: uint8(value * 11), G: uint8(value*7 + y), B: uint8(x * 3), A: 255})
				}
			}
			if mixed && (value == 1 || value == 31) {
				w.Header().Set("Content-Type", "image/jpeg")
				_ = jpeg.Encode(w, img, nil)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_ = png.Encode(w, img)
		}
	}))
	defer server.Close()
	t.Setenv("CALIBRE_OPDS_URL", server.URL+"/opds")
	t.Setenv("JELLYFIN_URL", server.URL)
	counts := map[string]int{}
	if err := scanAll(ctx, "all", func(item source.Item) error { counts[item.Type]++; return nil }); err != nil {
		t.Fatal(err)
	}
	if counts["book"] != 5 || counts["movie"] != 5 || counts["series"] != 2 {
		t.Fatalf("scan counts: %v", counts)
	}
	type coverReceipt struct {
		ID   string
		Hash [32]byte
	}
	check := func(want int) map[string]coverReceipt {
		t.Helper()
		resp, err := client.Get(base + "/api/v1/library/media")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result struct {
			Data struct {
				Total int
				Items []struct {
					Title, Description string
					CoverID            string `json:"cover_id"`
				}
			}
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Data.Total != want {
			t.Fatalf("media total=%d want=%d err=%v", result.Data.Total, want, err)
		}
		covers, changed := 0, 0
		receipts := map[string]coverReceipt{}
		for _, item := range result.Data.Items {
			if strings.Contains(item.Title, "修订版") || item.Description == "更新后的电影简介" {
				changed++
			}
			if item.CoverID == "" {
				continue
			}
			cover, err := client.Get(base + "/api/v1/library/images/" + item.CoverID)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(cover.Body)
			cover.Body.Close()
			if err != nil || cover.StatusCode != 200 {
				t.Fatalf("cover download status=%d err=%v", cover.StatusCode, err)
			}
			switch {
			case mixed && strings.HasPrefix(item.Title, "中文电子书 1"):
				if cover.Header.Get("Content-Type") != "image/jpeg" {
					t.Fatal("unexpected JPEG MIME")
				}
				if _, err = jpeg.Decode(bytes.NewReader(data)); err != nil {
					t.Fatal(err)
				}
			case mixed && item.Title == "中文电子书 2":
				if cover.Header.Get("Content-Type") != "image/webp" || !bytes.Equal(data, webp) {
					t.Fatal("unexpected WebP response")
				}
			default:
				if _, err = png.Decode(bytes.NewReader(data)); err != nil {
					t.Fatal(err)
				}
			}
			receipts[item.Title] = coverReceipt{item.CoverID, sha256.Sum256(data)}
			covers++
		}
		if want == 12 && covers != 11 {
			t.Fatalf("covers=%d", covers)
		}
		if revision.Load() && changed != 2 {
			t.Fatalf("changed=%d", changed)
		}
		return receipts
	}
	check(0)
	if err := run(ctx, []string{"sync", "all", "--limit", "20"}); err != nil {
		t.Fatal(err)
	}
	check(12)
	if err := run(ctx, []string{"sync", "all"}); err != nil {
		t.Fatal(err)
	}
	check(12)
	revision.Store(true)
	if err := run(ctx, []string{"sync", "all"}); err != nil {
		t.Fatal(err)
	}
	before := check(12)
	if mixed {
		replaceCover.Store(true)
		if err := run(ctx, []string{"sync", "all"}); err != nil {
			t.Fatal(err)
		}
		after := check(12)
		for title, previous := range before {
			next := after[title]
			if title == "中文电子书 1 修订版" {
				if previous.ID == next.ID || previous.Hash == next.Hash {
					t.Fatalf("cover ID and content hash must change for %s", title)
				}
			} else if previous != next {
				t.Fatalf("unrelated cover changed: %s", title)
			}
		}
	}
}
