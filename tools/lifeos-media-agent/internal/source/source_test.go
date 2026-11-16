package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCalibreNavigation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var content string
		switch r.URL.RequestURI() {
		case "/opds":
			content = `<entry><title>按最新</title><link type="application/atom+xml" href="books?library_id=a%20b"/></entry>`
		case "/books?library_id=a%20b":
			content = `<link rel="next" href="?library_id=a%20b&amp;page=2"/><entry><id>one</id><title>中文</title><author><name>甲</name></author><author><name>乙</name></author><link rel="http://opds-spec.org/acquisition" type="application/epub+zip" href="/never-download"/></entry>`
		case "/books?library_id=a%20b&page=2":
			content = `<entry><id>one</id><title>重复</title><link rel="http://opds-spec.org/acquisition"/></entry><entry><id>two</id><title>续页</title><link rel="http://opds-spec.org/acquisition"/></entry>`
		default:
			t.Errorf("unexpected request %s", r.URL)
			w.WriteHeader(404)
			return
		}
		fmt.Fprintf(w, `<feed xmlns="http://www.w3.org/2005/Atom">%s</feed>`, content)
	}))
	defer server.Close()
	client, err := New(server.URL+"/opds", "")
	if err != nil {
		t.Fatal(err)
	}
	var items []Item
	err = client.Calibre(context.Background(), "按最新", "", func(item Item) error { items = append(items, item); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].ExternalID != "two" || !reflect.DeepEqual(items[0].Authors, []string{"甲", "乙"}) {
		t.Fatalf("unexpected books: %+v", items)
	}
}
