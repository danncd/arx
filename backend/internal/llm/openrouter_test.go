package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeModelID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"deepseek/deepseek-chat", "deepseek-chat"},
		{"deepseek-v4-flash", "deepseek-v4-flash"},
		{"meta-llama/llama-3-70b:free", "llama-3-70b"},
		{"gpt-5-2025-08-07", "gpt-5"},
		{"openai/gpt-5-pro-2025-10-06", "gpt-5-pro"},
		{"GPT-5.2", "gpt-5.2"},
		{"gpt-4-0613", "gpt-4-0613"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := normalizeModelID(c.in); got != c.want {
				t.Errorf("normalizeModelID(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestFetchORCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"deepseek/deepseek-v4-flash","context_length":1000000,
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/sora-2","context_length":0,
			 "architecture":{"modality":"text->video"}}
		]}`))
	}))
	defer srv.Close()

	old := orModelsURL
	orModelsURL = srv.URL
	defer func() { orModelsURL = old }()

	index, err := fetchORCatalog(context.Background())
	if err != nil {
		t.Fatalf("fetchORCatalog: %v", err)
	}
	om, ok := index["deepseek-v4-flash"]
	if !ok {
		t.Fatalf("catalog not indexed by normalized name: %v", index)
	}
	if om.ContextLength != 1000000 || om.Architecture.Modality != "text->text" {
		t.Fatalf("wrong entry: %+v", om)
	}
	if _, ok := index["sora-2"]; !ok {
		t.Fatal("fetch should index everything; filtering is LoadModels' job")
	}
}

func TestFetchORCatalogErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := orModelsURL
	orModelsURL = srv.URL
	defer func() { orModelsURL = old }()

	if _, err := fetchORCatalog(context.Background()); err == nil {
		t.Fatal("HTTP 500 must surface as an error")
	}
}
