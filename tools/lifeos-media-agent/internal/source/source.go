package source

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Format struct {
	MIME string
	Size int64
}

type Item struct {
	Library                                                string `json:"library,omitempty"`
	Source, SourceID, ExternalID, Type, Title, Description string
	Authors, Genres, Tags                                  []string
	Formats                                                []Format
	Year                                                   int
	Rating                                                 float64
	Cover                                                  *url.URL                              `json:"-"`
	ReadCover                                              func(context.Context) ([]byte, error) `json:"-"`

	// Episode-only fields; empty for movies, series and books.
	SeriesExternalID string `json:",omitempty"`
	SeasonExternalID string `json:",omitempty"`
	SeasonNumber     int    `json:",omitempty"`
	EpisodeNumber    int    `json:",omitempty"`
	PremiereDate     string `json:",omitempty"`
	RuntimeSeconds   int    `json:",omitempty"`
}

type Client struct {
	Base                         *url.URL
	Token                        string
	BasicUsername, BasicPassword string
	HTTP                         *http.Client
}

func New(raw, token string) (Client, error) {
	base, err := url.Parse(raw)
	if err != nil || base.Host == "" || base.User != nil || (base.Scheme != "http" && base.Scheme != "https") {
		return Client{}, fmt.Errorf("invalid source URL")
	}
	return Client{Base: base, Token: token, HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c Client) get(ctx context.Context, target *url.URL, decode func([]byte) error) error {
	if target.Host != c.Base.Host || target.Scheme != c.Base.Scheme || target.User != nil {
		return fmt.Errorf("cross-origin source link")
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if err != nil {
			return fmt.Errorf("invalid request")
		}
		if c.Token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("MediaBrowser Token=%q, Client=\"LifeOS Media Agent\", Device=\"Mac mini\", DeviceId=\"lifeos-media-agent\", Version=\"1.0.0\"", c.Token))
		}
		resp, err := c.HTTP.Do(req)
		if err == nil && resp.StatusCode == http.StatusUnauthorized && c.BasicUsername != "" {
			challenge := resp.Header.Get("WWW-Authenticate")
			resp.Body.Close()
			req = req.Clone(ctx)
			if authErr := calibreAuthorization(req, challenge, c.BasicUsername, c.BasicPassword); authErr != nil {
				return authErr
			}
			resp, err = c.HTTP.Do(req)
		}
		if err == nil {
			if resp.StatusCode == 200 {
				const max = 16 << 20
				data, readErr := io.ReadAll(io.LimitReader(resp.Body, max+1))
				resp.Body.Close()
				if readErr == nil {
					if len(data) > max {
						return fmt.Errorf("response exceeds limit")
					}
					return decode(data)
				}
			} else {
				resp.Body.Close()
				if resp.StatusCode != 429 && resp.StatusCode < 500 {
					return fmt.Errorf("source returned HTTP %d", resp.StatusCode)
				}
			}
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * time.Second):
			}
		}
	}
	return fmt.Errorf("source request failed after retries")
}

type link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	MIME string `xml:"type,attr"`
	Size int64  `xml:"length,attr"`
}
type feed struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Links   []link   `xml:"link"`
	Entries []struct {
		ID         string   `xml:"id"`
		Title      string   `xml:"title"`
		Summary    string   `xml:"summary"`
		Authors    []string `xml:"author>name"`
		Links      []link   `xml:"link"`
		Categories []struct {
			Term string `xml:"term,attr"`
		} `xml:"category"`
	} `xml:"entry"`
}

func (c Client) Calibre(ctx context.Context, latest, override string, emit func(Item) error) error {
	page := c.Base
	seenPages := map[string]bool{}
	seenIDs := map[string]bool{}
	for root := true; page != nil; root = false {
		if seenPages[page.String()] || len(seenPages) >= 10000 {
			return fmt.Errorf("repeated page or page limit exceeded")
		}
		seenPages[page.String()] = true
		var f feed
		if err := c.get(ctx, page, func(b []byte) error { return xml.Unmarshal(b, &f) }); err != nil {
			return err
		}
		if root {
			href := override
			var discovered []string
			for _, entry := range f.Entries {
				for _, l := range entry.Links {
					if strings.Contains(l.MIME, "atom+xml") {
						discovered = append(discovered, entry.Title+" "+l.Href)
						if override == "" && entry.Title == latest {
							href = l.Href
						}
					}
				}
			}
			if href == "" {
				return fmt.Errorf("catalog %q missing, discovered %q", latest, discovered)
			}
			next, err := page.Parse(href)
			if err != nil {
				return fmt.Errorf("invalid catalog link")
			}
			page = next
			continue
		}
		for _, e := range f.Entries {
			item := Item{Source: "calibre", SourceID: c.Base.Host, ExternalID: e.ID, Type: "book", Title: e.Title, Description: e.Summary, Authors: e.Authors}
			for _, l := range e.Links {
				if strings.HasPrefix(l.Rel, "http://opds-spec.org/acquisition") {
					item.Formats = append(item.Formats, Format{l.MIME, l.Size})
				}
				if l.Rel == "http://opds-spec.org/image" {
					item.Cover, _ = page.Parse(l.Href)
				}
			}
			for _, category := range e.Categories {
				item.Tags = append(item.Tags, category.Term)
			}
			if len(item.Formats) == 0 || seenIDs[e.ID] {
				continue
			}
			if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Title) == "" {
				return fmt.Errorf("book is missing ID or title")
			}
			seenIDs[e.ID] = true
			if item.Cover != nil {
				item.ReadCover = func(ctx context.Context) ([]byte, error) { return c.DownloadCover(ctx, item.Cover) }
			}
			if err := emit(item); err != nil {
				return err
			}
		}
		var next *url.URL
		for _, l := range f.Links {
			if l.Rel == "next" {
				var err error
				next, err = page.Parse(l.Href)
				if err != nil {
					return fmt.Errorf("invalid pagination link")
				}
				break
			}
		}
		page = next
	}
	return nil
}

func (c Client) json(ctx context.Context, path string, query url.Values, dst any) error {
	u := c.Base.JoinPath(path)
	u.RawQuery = query.Encode()
	return c.get(ctx, u, func(b []byte) error { return json.Unmarshal(b, dst) })
}

// virtualFolder is one Jellyfin library (Movies, TV Shows, and anything else).
type virtualFolder struct {
	Name, CollectionType, ItemID string
}

func (c Client) virtualFolders(ctx context.Context) ([]virtualFolder, error) {
	var folders []virtualFolder
	if err := c.json(ctx, "Library/VirtualFolders", nil, &folders); err != nil {
		return nil, err
	}
	return folders, nil
}

func (c Client) Jellyfin(ctx context.Context, include []string, emit func(Item) error) error {
	folders, err := c.virtualFolders(ctx)
	if err != nil {
		return err
	}
	var names []string
	allowed := map[string]string{}
	for _, f := range folders {
		names = append(names, f.Name)
		if f.ItemID != "" {
			switch f.CollectionType {
			case "movies":
				allowed[f.Name] = f.ItemID
			case "tvshows":
				allowed[f.Name] = f.ItemID
			}
		}
	}
	for _, name := range include {
		if allowed[name] == "" {
			return fmt.Errorf("library %q unavailable, discovered %q", name, names)
		}
	}
	seen := map[string]bool{}
	for _, name := range include {
		for start := 0; ; {
			var page struct {
				TotalRecordCount int
				Items            []struct {
					ID, Name, Type, Overview string
					ProductionYear           int
					CommunityRating          float64
					Genres                   []string
					ImageTags                map[string]string
				}
			}
			query := url.Values{"ParentId": {allowed[name]}, "Recursive": {"true"}, "IncludeItemTypes": {"Movie,Series"}, "Fields": {"Overview,Genres"}, "Limit": {"100"}, "StartIndex": {fmt.Sprint(start)}, "EnableTotalRecordCount": {"true"}}
			if err := c.json(ctx, "Items", query, &page); err != nil {
				return err
			}
			if page.TotalRecordCount < 0 || len(page.Items) > 100 {
				return fmt.Errorf("invalid Jellyfin page")
			}
			for _, entry := range page.Items {
				if entry.ID == "" || entry.Name == "" || seen[entry.ID] || (entry.Type != "Movie" && entry.Type != "Series") {
					return fmt.Errorf("invalid or repeated Jellyfin item")
				}
				seen[entry.ID] = true
				item := Item{Library: name, Source: "jellyfin", SourceID: c.Base.Host, ExternalID: entry.ID, Type: strings.ToLower(entry.Type), Title: entry.Name, Description: entry.Overview, Genres: entry.Genres, Year: entry.ProductionYear, Rating: entry.CommunityRating}
				if entry.ImageTags["Primary"] != "" && !strings.ContainsAny(entry.ID, "/\\.") {
					item.Cover = c.Base.JoinPath("Items", entry.ID, "Images", "Primary")
					item.Cover.RawQuery = "maxWidth=600"
				}
				if item.Cover != nil {
					item.ReadCover = func(ctx context.Context) ([]byte, error) { return c.DownloadCover(ctx, item.Cover) }
				}
				if err := emit(item); err != nil {
					return err
				}
			}
			start += len(page.Items)
			if start >= page.TotalRecordCount {
				break
			}
			if len(page.Items) == 0 || start > 1000000 {
				return fmt.Errorf("Jellyfin pagination stalled")
			}
		}
	}
	return nil
}

// DownloadCover reuses source authentication and the same-origin restriction.
func (c Client) DownloadCover(ctx context.Context, target *url.URL) ([]byte, error) {
	var data []byte
	err := c.get(ctx, target, func(body []byte) error {
		if len(body) > 10<<20 {
			return fmt.Errorf("cover exceeds 10 MiB")
		}
		switch http.DetectContentType(body) {
		case "image/jpeg", "image/png", "image/webp":
			data = body
			return nil
		default:
			return fmt.Errorf("unsupported cover image type")
		}
	})
	return data, err
}
// maxEpisodePages mirrors the item scan: refuse to loop forever on a broken server.
const (
	maxSeasonCount  = 200
	maxEpisodePages = 1000
)

// JellyfinEpisodes walks every TV library, its series, the seasons of each series and
// the episodes of each season. Only metadata is read; the video file is never fetched.
func (c Client) JellyfinEpisodes(ctx context.Context, include []string, emit func(Item) error) error {
	folders, err := c.virtualFolders(ctx)
	if err != nil {
		return err
	}
	allowed := map[string]string{}
	for _, folder := range folders {
		if folder.CollectionType == "tvshows" && folder.ItemID != "" {
			allowed[folder.Name] = folder.ItemID
		}
	}
	for _, name := range include {
		parent, ok := allowed[name]
		if !ok {
			// A movies library (or an R18 library that was not requested) is not a
			// series source; skipping is the documented behaviour, not an error.
			continue
		}
		for start := 0; ; {
			var page struct {
				TotalRecordCount int
				Items            []struct{ ID, Name string }
			}
			query := url.Values{"ParentId": {parent}, "Recursive": {"true"}, "IncludeItemTypes": {"Series"},
				"Fields": {"ProductionYear"}, "Limit": {"100"}, "StartIndex": {fmt.Sprint(start)}, "EnableTotalRecordCount": {"true"}}
			if err := c.json(ctx, "Items", query, &page); err != nil {
				return err
			}
			if page.TotalRecordCount < 0 || len(page.Items) > 100 {
				return fmt.Errorf("invalid Jellyfin series page")
			}
			for _, series := range page.Items {
				if series.ID == "" || strings.ContainsAny(series.ID, "/\\.") {
					return fmt.Errorf("invalid Jellyfin series id")
				}
				if err := c.seriesEpisodes(ctx, name, series.ID, series.Name, emit); err != nil {
					return err
				}
			}
			start += len(page.Items)
			if len(page.Items) == 0 || start >= page.TotalRecordCount {
				break
			}
			if start > 1000000 {
				return fmt.Errorf("Jellyfin series pagination stalled")
			}
		}
	}
	return nil
}

func (c Client) seriesEpisodes(ctx context.Context, library, seriesID, seriesName string, emit func(Item) error) error {
	var seasons struct {
		Items []struct {
			ID, Name    string
			IndexNumber int
		}
	}
	if err := c.json(ctx, "Shows/"+seriesID+"/Seasons", nil, &seasons); err != nil {
		return err
	}
	if len(seasons.Items) > maxSeasonCount {
		return fmt.Errorf("series %q reports too many seasons", seriesName)
	}
	for _, season := range seasons.Items {
		if season.IndexNumber < 0 {
			continue
		}
		for page := 0; page < maxEpisodePages; page++ {
			var body struct {
				TotalRecordCount int
				Items            []struct {
					ID, Name, Overview, PremiereDate string
					IndexNumber, ParentIndexNumber  int
					RunTimeTicks                    int64
					ImageTags                       map[string]string
				}
			}
			query := url.Values{"season": {fmt.Sprint(season.IndexNumber)}, "Fields": {"Overview,PremiereDate,RunTimeTicks"},
				"Limit": {"100"}, "StartIndex": {fmt.Sprint(page * 100)}, "EnableTotalRecordCount": {"true"}}
			if err := c.json(ctx, "Shows/"+seriesID+"/Episodes", query, &body); err != nil {
				return err
			}
			if body.TotalRecordCount < 0 || len(body.Items) > 100 {
				return fmt.Errorf("invalid Jellyfin episode page")
			}
			for _, episode := range body.Items {
				if episode.ID == "" || strings.ContainsAny(episode.ID, "/\\.") {
					return fmt.Errorf("invalid Jellyfin episode id")
				}
				seasonNumber := episode.ParentIndexNumber
				if seasonNumber == 0 && season.IndexNumber != 0 {
					seasonNumber = season.IndexNumber
				}
				title := strings.TrimSpace(episode.Name)
				if title == "" {
					title = fmt.Sprintf("S%02dE%02d", seasonNumber, episode.IndexNumber)
				}
				item := Item{
					Library: library, Source: "jellyfin", SourceID: c.Base.Host, ExternalID: episode.ID, Type: "episode",
					Title: title, Description: episode.Overview,
					SeriesExternalID: seriesID, SeasonExternalID: season.ID, SeasonNumber: seasonNumber, EpisodeNumber: episode.IndexNumber,
					PremiereDate: premiereDate(episode.PremiereDate), RuntimeSeconds: int(episode.RunTimeTicks / 10000000),
				}
				item.Year = premiereYear(item.PremiereDate)
				if episode.ImageTags["Primary"] != "" {
					item.Cover = c.Base.JoinPath("Items", episode.ID, "Images", "Primary")
					item.Cover.RawQuery = "maxWidth=400"
					if err := emit(item); err != nil {
						return err
					}
					continue
				}
				if err := emit(item); err != nil {
					return err
				}
			}
			if len(body.Items) == 0 || (page+1)*100 >= body.TotalRecordCount {
				break
			}
		}
	}
	return nil
}

// premiereDate keeps only the calendar day: Jellyfin sends a full timestamp.
func premiereDate(value string) string {
	if len(value) >= 10 && value[4] == '-' && value[7] == '-' {
		return value[:10]
	}
	return ""
}

func premiereYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	year := 0
	for _, r := range date[:4] {
		if r < '0' || r > '9' {
			return 0
		}
		year = year*10 + int(r-'0')
	}
	return year
}