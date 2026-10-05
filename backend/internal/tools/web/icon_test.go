package web

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSiteIconUsesOnlyTheOriginAndReturnsRasterData(t *testing.T) {
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/favicon.ico" || r.URL.RawQuery != "" {
			t.Error("icon request retained page path or query")
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
	}))
	defer server.Close()
	client := &Client{http: server.Client()}
	value, err := client.Icon(context.Background(), server.URL+"/private/path?query=hidden")
	if err != nil || !strings.HasPrefix(value, "data:image/png;base64,") {
		t.Fatalf("icon: %s %v", value, err)
	}
}

func TestSiteIconRejectsLocalAddressesScriptsAndOversizedData(t *testing.T) {
	if _, err := New().Icon(context.Background(), "http://127.0.0.1/favicon.ico"); err == nil {
		t.Fatal("local address allowed")
	}
	for _, body := range []string{"<svg xmlns=\"http://www.w3.org/2000/svg\"><script>alert(1)</script></svg>", strings.Repeat("x", (256<<10)+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		_, err := (&Client{http: server.Client()}).Icon(context.Background(), server.URL)
		server.Close()
		if err == nil {
			t.Fatal("unsupported icon accepted")
		}
	}
}
