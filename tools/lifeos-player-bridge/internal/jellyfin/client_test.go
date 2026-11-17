package jellyfin

import ("context"; "errors"; "net/http"; "net/http/httptest"; "testing"; "time")

func TestPlaybackInfoSendsTokenAndParsesSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Emby-Token"); got != "secret-token" {
			t.Errorf("token header %q", got)
		}
		if r.URL.Path != "/Items/0f8fad5bd9cb469fa16570867728950e/PlaybackInfo" {
			t.Errorf("path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"MediaSources":[{"Path":"/media/movies/Dune.mkv","Container":"mkv","Size":1024}]}`))
	}))
	defer server.Close()
	client := New(server.URL, "secret-token", time.Second)
	info, err := client.PlaybackInfo(context.Background(), "0f8fad5bd9cb469fa16570867728950e", "user-1")
	if err != nil || len(info.MediaSources) != 1 || info.MediaSources[0].Path != "/media/movies/Dune.mkv" {
		t.Fatalf("%+v %v", info, err)
	}
}

func TestStatusCodesMapToTypedErrors(t *testing.T) {
	for status, want := range map[int]error{401: ErrUnauthorized, 403: ErrUnauthorized, 404: ErrNotFound, 500: ErrUnavailable, 503: ErrUnavailable} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		client := New(server.URL, "t", time.Second)
		_, err := client.PlaybackInfo(context.Background(), "0f8fad5bd9cb469fa16570867728950e", "")
		server.Close()
		if !errors.Is(err, want) {
			t.Fatalf("status %d => %v, want %v", status, err, want)
		}
	}
}

func TestLibraryAllowedRequiresAnExplicitMatch(t *testing.T) {
	ancestors := []Ancestor{{Name: "Movies", CollectionType: "movies"}, {Name: "NAS"}}
	if !LibraryAllowed(ancestors, []string{"movies"}) {
		t.Fatal("collection type should match")
	}
	if !LibraryAllowed(ancestors, []string{"nas"}) {
		t.Fatal("library name should match case-insensitively")
	}
	if LibraryAllowed(ancestors, []string{"tvshows"}) {
		t.Fatal("unlisted library must be denied")
	}
	if LibraryAllowed(ancestors, nil) {
		t.Fatal("empty allowlist must deny")
	}
}