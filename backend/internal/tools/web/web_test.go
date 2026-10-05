package web

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalAddressesAndSchemesAreRejected(t *testing.T) {
	for _, value := range []string{"127.0.0.1", "::1", "192.168.0.1", "10.0.0.1", "172.16.1.1", "169.254.169.254", "100.64.0.1", "::ffff:127.0.0.1"} {
		if publicIP(net.ParseIP(value)) {
			t.Fatalf("private address accepted: %s", value)
		}
	}
	client := New()
	for _, address := range []string{"http://127.0.0.1:1234", "http://[::1]/", "file:///etc/passwd", "https://user:secret@example.com"} {
		if _, _, _, err := client.get(context.Background(), address); err == nil {
			t.Fatalf("unsafe URL accepted: %s", address)
		}
	}
}

func TestFetchExtractsReadableTextAndSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<html><head><title>Example</title><style>hidden</style></head><body><h1>Heading</h1><p>Useful <b>text</b>.</p><script>unwanted</script></body></html>`)
	}))
	defer server.Close()
	client := &Client{http: server.Client()}
	raw, _ := json.Marshal(map[string]string{"operation": "fetch", "url": server.URL})
	prepared, err := client.Prepare(raw)
	if err != nil {
		t.Fatal(err)
	}
	result, err := prepared.Run(context.Background())
	if err != nil || !strings.Contains(result.Text, "Source: "+server.URL) || !strings.Contains(result.Text, "Useful text") || strings.Contains(result.Text, "hidden") || strings.Contains(result.Text, "unwanted") {
		t.Fatalf("page: %+v %v", result, err)
	}
}

func TestSearchReturnsSourceLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "go docs" {
			t.Error("wrong query")
		}
		io.WriteString(w, `<rss><channel><item><title>Go docs</title><link>https://go.dev/doc/</link><description>Documentation</description></item></channel></rss>`)
	}))
	defer server.Close()
	client := &Client{http: server.Client(), searchURL: server.URL}
	result, err := client.search(context.Background(), "go docs")
	if err != nil || !strings.Contains(result.Text, "https://go.dev/doc/") {
		t.Fatalf("search: %+v %v", result, err)
	}
}

func TestPageTextPreservesLinksAndIgnoresScriptURLs(t *testing.T) {
	text := pageText([]byte(`<p><a href="../guide">Guide</a><a href="javascript:alert(1)">Action</a></p>`), "https://example.com/docs/page")
	if !strings.Contains(text, "https://example.com/guide") || strings.Contains(text, "javascript:") {
		t.Fatal(text)
	}
}
