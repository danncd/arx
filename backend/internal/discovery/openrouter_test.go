package discovery

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
		{"gpt-5-2025-08-07", "gpt-5"},
		{"openai/gpt-5-pro-2025-10-06", "gpt-5-pro"},
		{"GPT-5.2", "gpt-5.2"},
		{"gpt-4-0613", "gpt-4-0613"},
		{"ft:gpt-4o-mini-2024-07-18:acme::9abc", "ft:gpt-4o-mini-2024-07-18:acme::9abc"},
		{"meta-llama/llama-3-70b:free", "llama-3-70b:free"},
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
			{"id":"deepseek/deepseek-v4-flash",
			 "supported_parameters":["reasoning","tools","temperature"],
			 "architecture":{"modality":"text->text"},
			 "top_provider":{"max_completion_tokens":65536}},
			{"id":"openai/sora-2",
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
	if om.Architecture.Modality != "text->text" {
		t.Fatalf("wrong entry: %+v", om)
	}
	if om.TopProvider.MaxCompletionTokens != 65536 {
		t.Fatalf("top_provider.max_completion_tokens not decoded: %+v", om)
	}
	if !contains(om.SupportedParameters, "reasoning") || !contains(om.SupportedParameters, "tools") {
		t.Fatalf("supported_parameters not decoded: %+v", om.SupportedParameters)
	}
	if _, ok := index["sora-2"]; !ok {
		t.Fatal("fetch should index everything; filtering is LoadModels' job")
	}
}

func TestFetchORCatalogBaseBeatsVariant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"deepseek/deepseek-r1:free",
			 "architecture":{"modality":"text->text"}},
			{"id":"deepseek/deepseek-r1",
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"vendor/basefirst",
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"vendor/basefirst:free",
			 "architecture":{"modality":"text->text"}},
			{"id":"vendor/lonely:free",
			 "architecture":{"modality":"text->text"}}
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
	if om := index["deepseek-r1"]; !contains(om.SupportedParameters, "tools") {
		t.Fatalf("variant-first ordering clobbered the base entry: %+v", om)
	}
	if om := index["basefirst"]; !contains(om.SupportedParameters, "tools") {
		t.Fatalf("base-first ordering lost the base entry to its variant: %+v", om)
	}
	if _, ok := index["lonely"]; !ok {
		t.Fatal("variant-only model missing from index")
	}
}

func TestFetchORCatalogDatedSnapshotNeverBeatsBase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"openai/gpt-4o-2024-05-13",
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/gpt-4o",
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/base-two",
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/base-two-2024-01-01",
			 "architecture":{"modality":"text->text"}}
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
	if om := index["gpt-4o"]; !contains(om.SupportedParameters, "tools") {
		t.Fatalf("dated snapshot clobbered the base entry: %+v", om)
	}
	if om := index["base-two"]; !contains(om.SupportedParameters, "tools") {
		t.Fatalf("base-first ordering lost the base to its snapshot: %+v", om)
	}
}

func TestFetchORCatalogEmptyIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"error":{"code":429,"message":"rate limited"}}`))
	}))
	defer srv.Close()

	old := orModelsURL
	orModelsURL = srv.URL
	defer func() { orModelsURL = old }()

	if _, err := fetchORCatalog(context.Background()); err == nil {
		t.Fatal("empty catalog must be reported as an error")
	}
}

func TestFetchORCatalogHollowIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{}]}`))
	}))
	defer srv.Close()

	old := orModelsURL
	orModelsURL = srv.URL
	defer func() { orModelsURL = old }()

	if _, err := fetchORCatalog(context.Background()); err == nil {
		t.Fatal("hollow catalog must be reported as an error")
	}
}

func TestFetchORCatalogSkipsHollowEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{},
			{"id":"openai/gpt-4o",
			 "architecture":{"modality":"text->text"}}
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
	if len(index) != 1 {
		t.Fatalf("wrong index: %+v", index)
	}
	if _, ok := index["gpt-4o"]; !ok {
		t.Fatalf("hollow entry clobbered a real one: %+v", index)
	}
	if _, ok := index[""]; ok {
		t.Fatal("hollow entry reached the index")
	}
}

func TestFetchORCatalogRejectsTrailingJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{"id":"openai/gpt-4o"}]}{"error":"upstream failed"}`))
	}))
	defer srv.Close()

	old := orModelsURL
	orModelsURL = srv.URL
	defer func() { orModelsURL = old }()

	if _, err := fetchORCatalog(context.Background()); err == nil {
		t.Fatal("trailing JSON must error")
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
