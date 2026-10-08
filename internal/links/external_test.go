package links

import "testing"

func TestSafeExternal(t *testing.T) {
	for _, value := range []string{"", "https://example.com/path?q=hello%20world", "http://localhost:8080", "obsidian://open?vault=YY-Wiki&file=Projects%2FLearnOS%2F%E4%BA%A7%E5%93%81.md"} {
		if !SafeExternal(value) {
			t.Errorf("rejected %q", value)
		}
	}
	for _, value := range []string{"javascript:alert(1)", "data:text/html,x", "file:///tmp/file", "//example.com", "https://", "https://u:p@example.com", "https://example.com\\@other", "https://example.com\n", "obsidian://new?vault=x&file=x", "obsidian://open?vault=x", "obsidian://open?vault=x&file=a&file=b", "obsidian://open?vault=x&file=a&append=yes", "obsidian://open?vault=x&file=%ZZ", "obsidian://open?vault=x&file=%00"} {
		if SafeExternal(value) {
			t.Errorf("accepted %q", value)
		}
	}
}
