package web

import (
	tool "arx/internal/tools"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchRejectsChallengeHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><title>Verify you are human</title></head><body>Challenge</body></html>`))
	}))
	defer server.Close()
	_, err := (&Client{http: server.Client(), searchURL: server.URL}).search(context.Background(), "go docs")
	if err == nil {
		t.Fatal("challenge interpreted as empty results")
	}
}

func TestSearchFiltersUnsafeDuplicateAndOffsiteResults(t *testing.T) {
	result := searchResult("site:go.dev guide", []tool.Source{
		{URL: "javascript:alert(1)", Title: "Guide"},
		{URL: "https://go.dev/doc/#one", Title: "Guide"},
		{URL: "https://go.dev/doc/#two", Title: "Guide duplicate"},
		{URL: "https://notgo.dev/", Title: "Wrong host"},
		{URL: "https://go.dev.evil.example/", Title: "Wrong suffix"},
		{URL: "https://pkg.go.dev/", Title: "Package guide"},
	})
	if len(result.Sources) != 2 || result.Sources[0].URL != "https://go.dev/doc/" {
		t.Fatalf("sources: %+v", result.Sources)
	}
}

func TestSearchWarnsAboutUnrelatedResultsWithoutInventingRanking(t *testing.T) {
	result := searchResult("Qwen3.5 model vision", []tool.Source{{URL: "https://example.com/dictionary", Title: "Dictionary definition"}})
	if !strings.Contains(result.Text, "may not match") || len(result.Sources) != 1 {
		t.Fatal(result.Text)
	}
}

func TestSearchLimitCountsUsableResults(t *testing.T) {
	candidates := make([]tool.Source, 12)
	candidates = append(candidates, tool.Source{URL: "https://go.dev/", Title: "Go"})
	if len(searchResult("Go", candidates).Sources) != 1 {
		t.Fatal("discarded usable result after invalid entries")
	}
}
