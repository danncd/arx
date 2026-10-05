package web

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageKeepsSourcePathAndValidatesContent(t *testing.T) {
	body, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/photos/diagram.png?size=small" {
			t.Error("image URL changed")
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Error("credentials forwarded")
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(body)
	}))
	defer server.Close()
	value, err := (&Client{http: server.Client()}).Image(context.Background(), server.URL+"/photos/diagram.png?size=small")
	if err != nil || value != "data:image/png;base64,"+base64.StdEncoding.EncodeToString(body) {
		t.Fatal("image fetch failed", err)
	}
}

func TestImageRejectsUnsafeAddressesFormatsAndOversizedResponses(t *testing.T) {
	for _, address := range []string{"http://127.0.0.1/a.png", "http://[::1]/a.png", "file:///tmp/a.png", "https://user:pass@example.com/a.png"} {
		if _, err := New().Image(context.Background(), address); err == nil {
			t.Fatal("unsafe URL accepted", address)
		}
	}
	for _, body := range []string{"<svg><script>alert(1)</script></svg>", "<html>not an image</html>", strings.Repeat("x", (8<<20)+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte(body))
		}))
		_, err := (&Client{http: server.Client()}).Image(context.Background(), server.URL)
		server.Close()
		if err == nil {
			t.Fatal("invalid image accepted")
		}
	}
}

func TestImageRedirectCannotReachLocalNetwork(t *testing.T) {
	client := New()
	response := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "file:///tmp/private.png", 302) }))
	defer response.Close()
	testClient := response.Client()
	testClient.CheckRedirect = client.http.CheckRedirect
	if _, err := (&Client{http: testClient}).Image(context.Background(), response.URL); err == nil {
		t.Fatal("unsafe redirect accepted")
	}
	if _, err := publicDial(context.Background(), "tcp", "127.0.0.1:80"); err == nil {
		t.Fatal("local connection accepted")
	}
}

func TestPageTextPreservesBoundedImageSources(t *testing.T) {
	html := `<p>Diagram</p><img src="../diagram.png" alt="A useful diagram"><img src="data:image/png;base64,abc" data-src="/large.jpg"><img src="javascript:alert(1)"><img src="https://user:secret@example.com/private">`
	text := pageText([]byte(html), "https://example.com/docs/page")
	if !strings.Contains(text, "https://example.com/diagram.png") || !strings.Contains(text, "A useful diagram") || !strings.Contains(text, "https://example.com/large.jpg") {
		t.Fatal("lost image source", text)
	}
	if strings.Contains(text, "javascript:") || strings.Contains(text, "secret") || strings.Contains(text, "base64") {
		t.Fatal("unsafe image source preserved")
	}
	text = pageText([]byte(strings.Repeat(`<img src="/photo.png">`, 30)), "https://example.com/")
	if strings.Count(text, "[Image:") != 12 {
		t.Fatal("image extraction was not bounded")
	}
}

func TestImageToolReturnsPixelsForTheRunner(t *testing.T) {
	body, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	client := &Client{http: server.Client()}
	prepared, err := client.Prepare([]byte(`{"operation":"image","url":"` + server.URL + `/picture.png"}`))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Action.Operation != "image" || prepared.Action.URL != server.URL+"/picture.png" {
		t.Fatal("image approval target missing")
	}
	result, err := prepared.Run(context.Background())
	if err != nil || len(result.ImageData) == 0 || len(result.Sources) != 1 {
		t.Fatal("image pixels missing", err)
	}
}
