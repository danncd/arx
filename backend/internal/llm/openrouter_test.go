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
		{"gpt-5-2025-08-07", "gpt-5"},
		{"openai/gpt-5-pro-2025-10-06", "gpt-5-pro"},
		{"GPT-5.2", "gpt-5.2"},
		{"gpt-4-0613", "gpt-4-0613"},
		// Colons are structural in fine-tune ids and must survive —
		// stripping them collapsed every ft: model onto the key "ft".
		// (":free" variant suffixes are handled in fetchORCatalog.)
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
			{"id":"deepseek/deepseek-v4-flash","context_length":1000000,
			 "supported_parameters":["reasoning","tools","temperature"],
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
	if !contains(om.SupportedParameters, "reasoning") || !contains(om.SupportedParameters, "tools") {
		t.Fatalf("supported_parameters not decoded: %+v", om.SupportedParameters)
	}
	if _, ok := index["sora-2"]; !ok {
		t.Fatal("fetch should index everything; filtering is LoadModels' job")
	}
}

// A ":free" variant shares its base model's key; the base entry's
// metadata must win no matter which the catalog lists first, because
// variants routinely differ in context length and tool support.
func TestFetchORCatalogBaseBeatsVariant(t *testing.T) {
	// Both orders: variant before base AND base before variant — a
	// last-write-wins index passes one but not both.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"deepseek/deepseek-r1:free","context_length":32000,
			 "architecture":{"modality":"text->text"}},
			{"id":"deepseek/deepseek-r1","context_length":128000,
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"vendor/basefirst","context_length":100000,
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"vendor/basefirst:free","context_length":5000,
			 "architecture":{"modality":"text->text"}},
			{"id":"vendor/lonely:free","context_length":8000,
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
	if om := index["deepseek-r1"]; om.ContextLength != 128000 || !contains(om.SupportedParameters, "tools") {
		t.Fatalf("variant-first ordering clobbered the base entry: %+v", om)
	}
	if om := index["basefirst"]; om.ContextLength != 100000 || !contains(om.SupportedParameters, "tools") {
		t.Fatalf("base-first ordering lost the base entry to its variant: %+v", om)
	}
	// A variant with no base entry still fills the key.
	if om := index["lonely"]; om.ContextLength != 8000 {
		t.Fatalf("variant-only model missing from index: %+v", om)
	}
}

// Dated snapshots collapse onto the base id's key; the undated base
// entry's metadata must win regardless of listing order.
func TestFetchORCatalogDatedSnapshotNeverBeatsBase(t *testing.T) {
	// Both orders again: snapshot-first and base-first.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"openai/gpt-4o-2024-05-13","context_length":8000,
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/gpt-4o","context_length":128000,
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/base-two","context_length":200000,
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"openai/base-two-2024-01-01","context_length":4000,
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
	if om := index["gpt-4o"]; om.ContextLength != 128000 || !contains(om.SupportedParameters, "tools") {
		t.Fatalf("dated snapshot clobbered the base entry: %+v", om)
	}
	if om := index["base-two"]; om.ContextLength != 200000 || !contains(om.SupportedParameters, "tools") {
		t.Fatalf("base-first ordering lost the base to its snapshot: %+v", om)
	}
}

// A 200 that yields no entries (an error envelope, a changed shape) is
// a failed authority and must be reported, not silently degrade.
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
