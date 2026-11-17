// Package jellyfin talks to PlaybackInfo with a dedicated non-admin token and
// gates every item on an explicit library allowlist.
package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrUnauthorized = errors.New("jellyfin rejected the credential")
	ErrNotFound     = errors.New("jellyfin item not found")
	ErrUnavailable  = errors.New("jellyfin is unreachable")
	ErrLibraryDenied = errors.New("item is outside the allowed libraries")
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: timeout}}
}

type MediaSource struct {
	Path      string `json:"Path"`
	Container string `json:"Container"`
	Size      int64  `json:"Size"`
}

type PlaybackInfo struct {
	MediaSources []MediaSource `json:"MediaSources"`
}

type Ancestor struct {
	ID             string `json:"Id"`
	Name           string `json:"Name"`
	CollectionType string `json:"CollectionType"`
	Type           string `json:"Type"`
}

type Item struct {
	ID           string        `json:"Id"`
	Name         string        `json:"Name"`
	Path         string        `json:"Path"`
	Container    string        `json:"Container"`
	Type         string        `json:"Type"`
	MediaType    string        `json:"MediaType"`
	MediaSources []MediaSource `json:"MediaSources"`
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	request.Header.Set("X-Emby-Token", c.token)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(response.Body).Decode(out); err != nil {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w: HTTP %d", ErrUnavailable, response.StatusCode)
	}
}

// PlaybackInfo returns the server-side playback sources for one item.
func (c *Client) PlaybackInfo(ctx context.Context, itemID, userID string) (*PlaybackInfo, error) {
	endpoint := "/Items/" + url.PathEscape(itemID) + "/PlaybackInfo"
	if userID != "" {
		endpoint += "?UserId=" + url.QueryEscape(userID)
	}
	var info PlaybackInfo
	if err := c.get(ctx, endpoint, &info); err != nil {
		return nil, err
	}
	if len(info.MediaSources) == 0 {
		return nil, ErrNotFound
	}
	return &info, nil
}

// Item returns the item detail used for type, container and path checks.
func (c *Client) Item(ctx context.Context, itemID, userID string) (*Item, error) {
	endpoint := "/Items/" + url.PathEscape(itemID)
	if userID != "" {
		endpoint = "/Users/" + url.PathEscape(userID) + "/Items/" + url.PathEscape(itemID)
	}
	var item Item
	if err := c.get(ctx, endpoint, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// Ancestors lists the item's parent containers so the library allowlist can be
// enforced from the server, not from the URL.
func (c *Client) Ancestors(ctx context.Context, itemID, userID string) ([]Ancestor, error) {
	endpoint := "/Items/" + url.PathEscape(itemID) + "/Ancestors"
	if userID != "" {
		endpoint += "?UserId=" + url.QueryEscape(userID)
	}
	var ancestors []Ancestor
	if err := c.get(ctx, endpoint, &ancestors); err != nil {
		return nil, err
	}
	return ancestors, nil
}

// LibraryAllowed reports whether one ancestor matches the configured allowlist
// by collection type or by name (case-insensitive).
func LibraryAllowed(ancestors []Ancestor, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	for _, ancestor := range ancestors {
		collection := strings.ToLower(strings.TrimSpace(ancestor.CollectionType))
		name := strings.ToLower(strings.TrimSpace(ancestor.Name))
		for _, entry := range allowed {
			if entry == "" {
				continue
			}
			if collection == entry || name == entry {
				return true
			}
		}
	}
	return false
}