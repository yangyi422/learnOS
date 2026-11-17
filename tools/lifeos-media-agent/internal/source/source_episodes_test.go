package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type stubEpisode struct {
	ID, Name, Overview, PremiereDate string
	Index, ParentIndex              int
	Ticks                           int64
	HasImage                        bool
}

// jellyfinStub answers the endpoints the episode scanner uses. It honours the
// StartIndex/Limit contract so pagination is exercised, not just parsed.
type stubLibrary struct {
	Name, CollectionType, ItemID string
	Series                      []string
}

func jellyfinStub(t *testing.T, libraries []stubLibrary, seasons map[string]map[int][]stubEpisode, denyAuth bool) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if denyAuth || !strings.Contains(r.Header.Get("Authorization"), "Token=") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		switch {
		case path == "Library/VirtualFolders":
			fmt.Fprint(w, "["+strings.Join(mapToJSON(libraries), ",")+"]")
		case path == "Items":
			start := queryInt(r.URL, "StartIndex")
			limit := queryInt(r.URL, "Limit")
			ids := []string{}
			for _, library := range libraries {
				if library.CollectionType != "tvshows" || r.URL.Query().Get("ParentId") != library.ItemID {
					continue
				}
				ids = append(ids, library.Series...)
			}
			end := start + limit
			if end > len(ids) {
				end = len(ids)
			}
			items := []string{}
			for _, id := range ids[start:end] {
				items = append(items, `{"ID":"`+id+`","Name":"`+id+`"}`)
			}
			fmt.Fprintf(w, `{"TotalRecordCount":%d,"Items":[%s]}`, len(ids), strings.Join(items, ","))
		case strings.HasPrefix(path, "Shows/") && strings.HasSuffix(path, "/Seasons"):
			seriesID := strings.TrimSuffix(strings.TrimPrefix(path, "Shows/"), "/Seasons")
			entries := []string{}
			for number := range seasons[seriesID] {
				entries = append(entries, fmt.Sprintf(`{"ID":"season-%s-%d","Name":"Season %d","IndexNumber":%d}`, seriesID, number, number, number))
			}
			fmt.Fprintf(w, `{"Items":[%s]}`, strings.Join(entries, ","))
		case strings.HasPrefix(path, "Shows/") && strings.HasSuffix(path, "/Episodes"):
			seriesID := strings.TrimSuffix(strings.TrimPrefix(path, "Shows/"), "/Episodes")
			season := queryInt(r.URL, "season")
			rows := seasons[seriesID][season]
			start := queryInt(r.URL, "StartIndex")
			limit := queryInt(r.URL, "Limit")
			end := start + limit
			if end > len(rows) {
				end = len(rows)
			}
			entries := []string{}
			if start < len(rows) {
				for _, row := range rows[start:end] {
					images := "{}"
					if row.HasImage {
						images = `{"Primary":"tag"}`
					}
					entries = append(entries, fmt.Sprintf(`{"ID":%q,"Name":%q,"Overview":%q,"PremiereDate":%q,"IndexNumber":%d,"ParentIndexNumber":%d,"RunTimeTicks":%d,"ImageTags":%s}`,
						row.ID, row.Name, row.Overview, row.PremiereDate, row.Index, row.ParentIndex, row.Ticks, images))
				}
			}
			fmt.Fprintf(w, `{"TotalRecordCount":%d,"Items":[%s]}`, len(rows), strings.Join(entries, ","))
		case strings.HasPrefix(path, "Items/") && strings.HasSuffix(path, "/Images/Primary"):
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R'})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	return &client
}

func mapToJSON(rows []stubLibrary) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = fmt.Sprintf(`{"Name":%q,"CollectionType":%q,"ItemID":%q}`, row.Name, row.CollectionType, row.ItemID)
	}
	return out
}

func queryInt(u *url.URL, key string) int {
	value := u.Query().Get(key)
	parsed := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		parsed = parsed*10 + int(r-'0')
	}
	return parsed
}

func TestJellyfinEpisodesWalksSeasonsAndPages(t *testing.T) {
	libraries := []stubLibrary{
		{Name: "电视剧", CollectionType: "tvshows", ItemID: "tv", Series: []string{"series-a", "series-b"}},
		{Name: "电影", CollectionType: "movies", ItemID: "movies"},
		{Name: "R18", CollectionType: "tvshows", ItemID: "r18", Series: []string{"r18-series"}},
	}
	seasons := map[string]map[int][]stubEpisode{
		"series-a": {
			1: {
				{ID: "ep1", Name: "第一集", Overview: "开篇", PremiereDate: "2011-04-17T00:00:00.0000000Z", Index: 1, ParentIndex: 1, Ticks: 36000000000, HasImage: true},
				{ID: "ep2", Name: "第二集", PremiereDate: "2011-04-24T00:00:00.0000000Z", Index: 2, ParentIndex: 1, Ticks: 30000000000},
			},
			3: {},
		},
		"series-b": {2: {{ID: "ep3", Name: "", PremiereDate: "", Index: 1, ParentIndex: 2, Ticks: 0}}},
		"r18-series": {1: {{ID: "r18ep", Name: "不应出现", Index: 1, ParentIndex: 1}}},
	}
	client := jellyfinStub(t, libraries, seasons, false)
	emitted := []Item{}
	err := client.JellyfinEpisodes(context.Background(), []string{"电视剧"}, func(item Item) error {
		emitted = append(emitted, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(emitted) != 3 {
		t.Fatalf("expected 3 episodes (empty season skipped, R18 excluded), got %d: %+v", len(emitted), emitted)
	}
	first := emitted[0]
	if first.Type != "episode" || first.SeriesExternalID == "" || first.SeasonNumber != 1 || first.EpisodeNumber != 1 || first.PremiereDate != "2011-04-17" || first.Year != 2011 || first.RuntimeSeconds != 3600 {
		t.Fatalf("episode metadata wrong: %+v", first)
	}
	if first.Library != "电视剧" || first.Cover == nil {
		t.Fatalf("library/cover wrong: %+v", first)
	}
	for _, item := range emitted {
		if strings.Contains(item.Title, "不应出现") {
			t.Fatalf("R18 library leaked: %+v", item)
		}
	}
	// A missing name falls back to the SxxExx label, and a missing premiere date stays empty.
	if emitted[2].Title != "S02E01" || emitted[2].PremiereDate != "" || emitted[2].Year != 0 {
		t.Fatalf("fallback title wrong: %+v", emitted[2])
	}
}

func TestJellyfinEpisodesHandlesAuthFailure(t *testing.T) {
	client := jellyfinStub(t, []stubLibrary{{Name: "电视剧", CollectionType: "tvshows", ItemID: "tv", Series: []string{"series-a"}}}, nil, true)
	err := client.JellyfinEpisodes(context.Background(), []string{"电视剧"}, func(Item) error { return nil })
	if err == nil {
		t.Fatal("expected an authentication error")
	}
}

func TestJellyfinEpisodesIgnoresMovieLibraries(t *testing.T) {
	libraries := []stubLibrary{{Name: "电影", CollectionType: "movies", ItemID: "movies"}}
	client := jellyfinStub(t, libraries, nil, false)
	count := 0
	err := client.JellyfinEpisodes(context.Background(), []string{"电影"}, func(Item) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("movies library must not yield episodes: %d", count)
	}
}