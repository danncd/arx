package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMainContentOmitsChromeAndPreservesEvidenceLinks(t *testing.T) {
	body := `<header>Account menus</header><main><h1>Guide</h1><p>Useful facts <a href="/source">source</a></p><div hidden>Hidden promotion</div><img src="/diagram.png" alt="Diagram"></main><footer>Footer</footer>`
	text := pageText([]byte(body), "https://example.com/guide")
	for _, expected := range []string{"Useful facts", "https://example.com/source", "https://example.com/diagram.png"} {
		if !strings.Contains(text, expected) {
			t.Fatal(text)
		}
	}
	for _, omitted := range []string{"Account menus", "Hidden promotion", "Footer"} {
		if strings.Contains(text, omitted) {
			t.Fatal(text)
		}
	}
}

func TestMultipleArticlesRemainVisible(t *testing.T) {
	text := pageText([]byte(`<article>First result</article><article>Second result</article>`), "https://example.com")
	if !strings.Contains(text, "First result") || !strings.Contains(text, "Second result") {
		t.Fatal(text)
	}
}

func TestFetchReportsChallengeAndEmptyShell(t *testing.T) {
	for _, body := range []string{`<title>Just a moment...</title><p>Checking your browser</p>`, `<html><head><title>App</title></head><body><script>render()</script></body></html>`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(body))
		}))
		_, err := (&Client{http: server.Client()}).fetch(context.Background(), server.URL)
		server.Close()
		if err == nil {
			t.Fatal("unreadable page accepted")
		}
	}
}
