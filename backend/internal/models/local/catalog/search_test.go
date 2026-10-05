package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchPaginationKeepsSortingAndMetadata(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if q.Get("sort") != "likes" || q.Get("search") != "Qwen" || q.Get("filter") != "gguf" {
			t.Error(q)
		}
		if q.Get("cursor") == "" {
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/api/models?cursor=next%%2Bpage>; rel="next"`, r.Host))
			fmt.Fprint(w, `[{"id":"org/Qwen","pipeline_tag":"image-text-to-text","tags":["reasoning"],"likes":80},{"id":"org/tts-helper","pipeline_tag":"text-to-speech"}]`)
		} else {
			if q.Get("cursor") != "next+page" {
				t.Error(q)
			}
			fmt.Fprint(w, `[{"id":"org/Qwen2","pipeline_tag":"text-generation"}]`)
		}
	}))
	defer server.Close()
	client := New()
	client.BaseURL = server.URL
	page, err := client.SearchPage(context.Background(), SearchOptions{Query: "Qwen", Sort: "likes"})
	if err != nil || len(page.Models) != 1 || page.Next != "next+page" {
		t.Fatal(page, err)
	}
	if len(page.Models[0].Tags) != 1 || page.Models[0].Likes != 80 {
		t.Fatal(page)
	}
	page, err = client.SearchPage(context.Background(), SearchOptions{Query: "Qwen", Sort: "likes", Cursor: page.Next})
	if err != nil || len(page.Models) != 1 || page.Next != "" || calls != 2 {
		t.Fatal(page, err, calls)
	}
	if _, err = client.SearchPage(context.Background(), SearchOptions{Sort: "invalid"}); err == nil || calls != 2 {
		t.Fatal("Invalid sort reached network")
	}
}
