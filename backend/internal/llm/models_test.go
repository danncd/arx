package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("path = %q, want /models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("auth header = %q, want Bearer sk-test", got)
		}
		w.Write([]byte(`{"object":"list","data":[
			{"id":"deepseek-v4-flash","owned_by":"deepseek"},
			{"id":"deepseek-v4-pro","owned_by":"deepseek"}]}`))
	}))
	defer srv.Close()

	t.Setenv("FAKE_KEY", "sk-test")
	p := Provider{Name: "fake", BaseURL: srv.URL, KeyEnv: "FAKE_KEY", Dialect: OpenAI}

	models, err := ListModels(context.Background(), p)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 || models[0].ID != "deepseek-v4-flash" || models[1].ID != "deepseek-v4-pro" {
		t.Fatalf("wrong models: %+v", models)
	}
}

func TestListModelsSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), Provider{Name: "fake", BaseURL: srv.URL})
	if err == nil {
		t.Fatal("a 401 must come back as an error")
	}
	if !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error should carry status and body, got: %v", err)
	}
}

// withFakeWorld points the package's two global knobs — the provider
// registry and the OpenRouter URL — at test servers, restoring both on
// cleanup. Tests using it cannot run in parallel.
func withFakeWorld(t *testing.T, providerURL, orURL string) {
	t.Helper()
	oldProviders := Providers
	Providers = map[string]Provider{
		"fake": {Name: "fake", BaseURL: providerURL, KeyEnv: "FAKE_KEY", Dialect: OpenAI},
	}
	oldOR := orModelsURL
	orModelsURL = orURL
	t.Cleanup(func() {
		Providers = oldProviders
		orModelsURL = oldOR
	})
	t.Setenv("FAKE_KEY", "sk-test")
}

func providerServing(t *testing.T, ids ...string) *httptest.Server {
	t.Helper()
	body := `{"data":[`
	for i, id := range ids {
		if i > 0 {
			body += ","
		}
		body += `{"id":"` + id + `","owned_by":"fake"}`
	}
	body += `]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The intersection policy: a model survives only if OpenRouter lists it
// with a text-producing modality, and survivors get the context window.
func TestLoadModelsIntersection(t *testing.T) {
	prov := providerServing(t, "chat-model", "video-model", "unknown-model")
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"fake/chat-model","context_length":128000,
			 "architecture":{"modality":"text->text"}},
			{"id":"fake/video-model","context_length":0,
			 "architecture":{"modality":"text->video"}}
		]}`))
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)

	models, err := LoadModels(context.Background())
	if err != nil {
		t.Fatalf("LoadModels: %v", err)
	}
	if len(models) != 1 || models[0].Model != "chat-model" {
		t.Fatalf("want only chat-model to survive, got: %+v", models)
	}
	if models[0].ContextWindow != 128000 {
		t.Fatalf("context window not stamped: %+v", models[0])
	}
}

// Degrade open: when OpenRouter is down, nothing is filtered and the
// failure surfaces as an error alongside the full raw list.
func TestLoadModelsDegradesOpenWhenORDown(t *testing.T) {
	prov := providerServing(t, "chat-model", "video-model")
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)

	models, err := LoadModels(context.Background())
	if err == nil {
		t.Fatal("an unreachable authority must be reported")
	}
	if len(models) != 2 {
		t.Fatalf("OR down must pass models through unfiltered, got: %+v", models)
	}
}

// An EMPTY-but-successful OpenRouter answer must also degrade open —
// intersecting with a hollow authority would blank the whole catalog.
func TestLoadModelsSurvivesEmptyORCatalog(t *testing.T) {
	prov := providerServing(t, "chat-model")
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)

	models, err := LoadModels(context.Background())
	if err != nil {
		t.Fatalf("empty catalog is not an error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("empty OR catalog must not blank the list, got: %+v", models)
	}
}

func TestListModelsRejectsGarbage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("<html>this is not json</html>"))
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), Provider{Name: "fake", BaseURL: srv.URL})
	if err == nil {
		t.Fatal("non-JSON body must error, not return empty models")
	}
}
