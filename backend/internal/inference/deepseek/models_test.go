package deepseek

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestModelsUsesProviderMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("incorrect provider request")
		}
		io.WriteString(w, `{"data":[{"id":"deepseek-flash","name":"DeepSeek Flash","context_window":1048576,"max_output_tokens":393216,"effort":{"supported_levels":["low","high","max"],"default_level":"high"}},{"id":"unknown"},{"id":"unknown"}]}`)
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL = server.URL
	models, err := client.Models(context.Background(), "test-key")
	if err != nil || len(models) != 2 {
		t.Fatalf("models: %+v %v", models, err)
	}
	first := models[0]
	if first.Name != "DeepSeek Flash" || first.ContextWindow != 1048576 || first.MaxOutputTokens != 393216 || first.Thinking.DefaultEffort != "high" || !first.Thinking.CanDisable {
		t.Fatalf("metadata: %+v", first)
	}
	if models[1].Thinking != nil || models[1].ContextWindow != 0 || models[1].Name != "unknown" {
		t.Fatalf("invented metadata: %+v", models[1])
	}
}

func TestProviderErrorsNeverEchoResponseBodies(t *testing.T) {
	for _, status := range []int{401, 403, 402, 429, 500} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			io.WriteString(w, r.Header.Get("Authorization"))
		}))
		client := NewClient()
		client.baseURL = server.URL
		_, err := client.Models(context.Background(), "sensitive-test-key")
		server.Close()
		if err == nil || strings.Contains(err.Error(), "sensitive-test-key") {
			t.Fatalf("unsafe error: %v", err)
		}
	}
}

func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	followed := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := NewClient()
	client.baseURL = server.URL
	if _, err := client.Models(context.Background(), "test-key"); err == nil {
		t.Fatal("redirect accepted")
	}
	if followed {
		t.Fatal("credentials followed redirect")
	}
}

func TestModelResponseIsBoundedAndCancellable(t *testing.T) {
	for _, body := range []string{"not json", `{"data":[]}`, strings.Repeat("x", (1<<20)+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
		client := NewClient()
		client.baseURL = server.URL
		_, err := client.Models(context.Background(), "key")
		server.Close()
		if err == nil {
			t.Fatal("invalid response accepted")
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client := NewClient()
	client.baseURL = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := client.Models(ctx, "key"); err == nil {
		t.Fatal("cancelled request succeeded")
	}
}
