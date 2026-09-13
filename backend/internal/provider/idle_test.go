package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStreamIdleTimeout(t *testing.T) {
	old := idleTimeout
	idleTimeout = 100 * time.Millisecond
	t.Cleanup(func() { idleTimeout = old })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"hi"}}]}` + "\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(2 * time.Second)
	}))
	t.Cleanup(srv.Close)

	prof := Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m"}
	start := time.Now()
	_, err := (OpenAIDialect{}).Stream(context.Background(), prof, []Message{{Role: "user", Content: "x"}}, nil, Options{}, func(string, bool) {})
	if err == nil || !strings.Contains(err.Error(), "idle timeout") {
		t.Fatalf("want idle timeout, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("idle timeout did not fire promptly: %s", time.Since(start))
	}
}

func TestChatIdleTimeout(t *testing.T) {
	old := idleTimeout
	idleTimeout = 100 * time.Millisecond
	t.Cleanup(func() { idleTimeout = old })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[`))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(2 * time.Second)
	}))
	t.Cleanup(srv.Close)

	prof := Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m"}
	_, err := (OpenAIDialect{}).Chat(context.Background(), prof, []Message{{Role: "user", Content: "x"}}, nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "idle timeout") {
		t.Fatalf("want idle timeout, got %v", err)
	}
}
