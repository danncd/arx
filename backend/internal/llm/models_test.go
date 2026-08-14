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

// The authority filter: a model OpenRouter knows survives only with a
// text-producing modality (and gets enriched); a model OpenRouter does
// NOT know passes through unenriched — OR's slug namespace misses real
// provider-native ids (deepseek-reasoner, ft:… fine-tunes), and a join
// miss must never disappear a working model.
func TestLoadModelsAuthorityFilter(t *testing.T) {
	// Fixture ids are chosen to pin each behavior separately: compound
	// modalities match the real catalog (plain "text->text" fixtures
	// once let a broken filter pass), R/T flags differ per model so a
	// swapped assignment fails, a cased+dated id pins normalization,
	// and a known "-instruct" id pins the name veto on join HITS.
	prov := providerServing(t,
		"chat-model", "tool-model", "video-model", "img-out-model",
		"unknown-model", "whisper-native", "turbo-instruct",
		"CHAT-MODEL-2025-01-01")
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"fake/chat-model","context_length":128000,
			 "supported_parameters":["reasoning"],
			 "architecture":{"modality":"text+image->text"}},
			{"id":"fake/tool-model","context_length":64000,
			 "supported_parameters":["tools"],
			 "architecture":{"modality":"text->text"}},
			{"id":"fake/video-model",
			 "architecture":{"modality":"text->video"}},
			{"id":"fake/img-out-model",
			 "architecture":{"modality":"text->text+image"}},
			{"id":"fake/turbo-instruct","context_length":4095,
			 "architecture":{"modality":"text->text"}}
		]}`))
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)

	models, err := LoadModels(context.Background())
	if err != nil {
		t.Fatalf("LoadModels: %v", err)
	}
	byName := map[string]ModelInfo{}
	for _, m := range models {
		byName[m.Model] = m
	}
	for _, dropped := range []string{"video-model", "img-out-model", "whisper-native", "turbo-instruct"} {
		if _, kept := byName[dropped]; kept {
			t.Fatalf("%s must be dropped: %+v", dropped, models)
		}
	}
	if len(models) != 4 {
		t.Fatalf("want 4 survivors, got: %+v", models)
	}
	chat := byName["chat-model"]
	if chat.ContextWindow != 128000 || !chat.Reasoning || chat.Tools {
		t.Fatalf("chat-model: want ctx 128000, R and NOT T: %+v", chat)
	}
	tool := byName["tool-model"]
	if tool.ContextWindow != 64000 || tool.Reasoning || !tool.Tools {
		t.Fatalf("tool-model: want ctx 64000, T and NOT R: %+v", tool)
	}
	unknown := byName["unknown-model"]
	if unknown.ContextWindow != 0 || unknown.Reasoning || unknown.Tools {
		t.Fatalf("join miss must pass through UNenriched: %+v", unknown)
	}
	// A cased, dated provider id must still join its base OR entry.
	dated := byName["CHAT-MODEL-2025-01-01"]
	if dated.ContextWindow != 128000 || !dated.Reasoning {
		t.Fatalf("normalization lost the dated/cased join: %+v", dated)
	}
}

// Degrade open, tier two: when OpenRouter is down the failure is
// reported and the NAME HEURISTIC still filters — obvious non-chat ids
// are dropped, everything else survives. (An earlier version of this
// test claimed "unfiltered" with fixtures that matched no marker, so
// it passed no matter what the code did.)
func TestLoadModelsDegradesOpenWhenORDown(t *testing.T) {
	prov := providerServing(t, "chat-model", "video-model", "whisper-x")
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
		t.Fatalf("want chat-model and video-model to survive the heuristic, got: %+v", models)
	}
	for _, m := range models {
		if m.Model == "whisper-x" {
			t.Fatalf("heuristic must still filter with OR down: %+v", models)
		}
		if m.ContextWindow != 0 {
			t.Fatalf("nothing should be enriched with OR down: %+v", m)
		}
	}
}

// An empty-but-200 OpenRouter answer is a failed authority (an error
// envelope, a changed shape), not an empty universe: the failure is
// REPORTED, and models still survive via the heuristic.
func TestLoadModelsSurvivesEmptyORCatalog(t *testing.T) {
	prov := providerServing(t, "chat-model")
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)

	models, err := LoadModels(context.Background())
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("a hollow authority must be reported, got: %v", err)
	}
	if len(models) != 1 || models[0].Model != "chat-model" {
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

// Valid JSON with no "data" key is a changed response shape, not an
// empty catalog — silence would misreport a working key as keyless.
func TestListModelsRejectsMissingDataKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"object":"list","models":[{"id":"a"}]}`))
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), Provider{Name: "fake", BaseURL: srv.URL})
	if err == nil || !strings.Contains(err.Error(), "no data field") {
		t.Fatalf("missing data key must error, got: %v", err)
	}
}

// The heuristic must not care about casing in provider ids.
func TestChatCapableIsCaseInsensitive(t *testing.T) {
	for _, id := range []string{"DALL-E-3", "Whisper-1", "TTS-1-HD"} {
		if chatCapable(id) {
			t.Errorf("chatCapable(%q) = true, want false", id)
		}
	}
	if !chatCapable("GPT-5.2") {
		t.Error("chatCapable(GPT-5.2) = false, want true")
	}
}
