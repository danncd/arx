package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDuckResultsDecodeLinksAndRejectChallenges(t *testing.T) {
	body := `<div class="result"><h2><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2Fdoc%2F">Go <b>documentation</b></a></h2><a class="result__snippet">Official documentation.</a></div>`
	results, err := duckResults([]byte(body))
	if err != nil || len(results) != 1 || results[0].URL != "https://go.dev/doc/" || results[0].Title != "Go documentation" {
		t.Fatalf("%+v %v", results, err)
	}
	for _, body := range []string{`<form id="challenge-form"></form>`, `<html>unexpected provider response</html>`} {
		if _, err := duckResults([]byte(body)); err == nil {
			t.Fatal("provider failure accepted")
		}
	}
	if duckLink("//duckduckgo.com/l/?uddg=javascript%3Aalert(1)") != "" {
		t.Fatal("unsafe redirect accepted")
	}
}

func TestProviderFallbackAndConcurrentCache(t *testing.T) {
	var duckCalls, bingCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "Go documentation" {
			t.Error("query changed")
		}
		if r.URL.Path == "/duck" {
			duckCalls.Add(1)
			w.Write([]byte(`<form id="challenge-form"></form>`))
		} else {
			bingCalls.Add(1)
			w.Write([]byte(`<rss><channel><item><title>Go documentation</title><link>https://go.dev/doc/</link><description>Official documentation</description></item></channel></rss>`))
		}
	}))
	defer server.Close()
	client := &Client{http: server.Client(), duckURL: server.URL + "/duck", searchURL: server.URL + "/bing"}
	var group sync.WaitGroup
	for range 12 {
		group.Go(func() {
			result, err := client.search(context.Background(), "Go documentation")
			if err != nil || len(result.Sources) != 1 || !strings.Contains(result.Text, "Search provider: Bing") {
				t.Errorf("%+v %v", result, err)
			}
		})
	}
	group.Wait()
	if duckCalls.Load() != 1 || bingCalls.Load() != 1 {
		t.Fatalf("repeated lookups: %d %d", duckCalls.Load(), bingCalls.Load())
	}
}

func TestSpecificNamesAndVersionsDoNotAcceptBroadMatches(t *testing.T) {
	for _, pair := range [][2]string{
		{"danncd github", "Battery discussion forum"},
		{"Qwen3.5 4B vision model", "Qwen3 models vision model"},
		{"Queens College schedule fall 2026", "Queens puzzle game"},
	} {
		if matchesQuery(pair[0], pair[1]) {
			t.Fatalf("accepted unrelated result: %v", pair)
		}
	}
	if !matchesQuery("Go documentation", "Official Go documentation") {
		t.Fatal("relevant result rejected")
	}
}

func TestUnrelatedResultsBecomeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<rss><channel><item><title>Battery discussion</title><link>https://example.com/battery</link></item></channel></rss>`))
	}))
	defer server.Close()
	result, err := (&Client{http: server.Client(), searchURL: server.URL}).search(context.Background(), "danncd github")
	if err == nil || len(result.Sources) != 0 || !strings.Contains(err.Error(), "no sufficiently matching results") {
		t.Fatalf("%+v %v", result, err)
	}
}
