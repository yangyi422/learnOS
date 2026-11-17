// Package protocol parses the lifeos-player:// URL handed to the bridge by the
// browser. It accepts exactly one shape and nothing else: a bad URL must never
// reach Jellyfin, the filesystem or the player.
package protocol

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

const Scheme = "lifeos-player"

var (
	ErrInvalidURL     = errors.New("invalid lifeos-player URL")
	ErrUnsupportedAct = errors.New("unsupported action")
	ErrInvalidItemID  = errors.New("invalid item id")
)

// Jellyfin item ids are 32 hex characters, optionally dash-separated.
var itemIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$|^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Request struct {
	ItemID string
}

// Parse validates a single argument and returns the requested action.
func Parse(raw string) (Request, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Request{}, ErrInvalidURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Scheme, Scheme) {
		return Request{}, ErrInvalidURL
	}
	// lifeos-player://play?itemId=... puts the action in the host part.
	action := strings.ToLower(strings.Trim(parsed.Host, "/"))
	if action != "play" || (parsed.Path != "" && parsed.Path != "/") {
		return Request{}, ErrUnsupportedAct
	}
	if parsed.Fragment != "" || parsed.User != nil {
		return Request{}, ErrInvalidURL
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(query) != 1 {
		return Request{}, ErrInvalidURL
	}
	ids, ok := query["itemId"]
	if !ok || len(ids) != 1 {
		return Request{}, ErrInvalidURL
	}
	itemID := strings.TrimSpace(ids[0])
	if !itemIDPattern.MatchString(itemID) {
		return Request{}, ErrInvalidItemID
	}
	return Request{ItemID: strings.ToLower(itemID)}, nil
}