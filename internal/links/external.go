package links

import (
	"net/url"
	"strings"
	"unicode"
)

// SafeExternal permits web navigation and only Obsidian's read/open action.
// It returns the original URI, preserving encoded vault and file names.
func SafeExternal(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 4096 || strings.ContainsAny(value, "\\") || strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil || u.Opaque != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Hostname() != ""
	case "obsidian":
		if u.Host != "open" || (u.Path != "" && u.Path != "/") || u.Fragment != "" {
			return false
		}
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil || q.Get("vault") == "" || q.Get("file") == "" {
			return false
		}
		for key, values := range q {
			if (key != "vault" && key != "file") || len(values) != 1 || strings.IndexFunc(values[0], unicode.IsControl) >= 0 {
				return false
			}
		}
		return true
	}
	return false
}
