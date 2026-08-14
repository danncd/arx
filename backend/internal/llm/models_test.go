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
	p := Provider{Name: "fake", BaseURL: srv.URL + "/", KeyEnv: "FAKE_KEY", Dialect: OpenAI}

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

/*
	Swaps provider and OpenRouter for tests.
*/

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

/*
	Known models use modality; unknown models pass through.
*/

func TestLoadModelsAuthorityFilter(t *testing.T) {
	// Cover metadata, normalization and name filters.
	prov := providerServing(t,
		"chat-model", "tool-model", "video-model", "img-out-model",
		"unknown-model", "sparse-model", "whisper-native", "turbo-instruct",
		"CHAT-MODEL-2025-01-01", "drift-model")
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
			 "architecture":{"modality":"text->text"}},
			{"id":"fake/drift-model","context_length":99000,
			 "supported_parameters":["tools"],"architecture":null},
			{"id":"fake/sparse-model","context_length":32000,
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
	if len(models) != 6 {
		t.Fatalf("want 6 survivors, got: %+v", models)
	}
	// Missing architecture keeps other metadata.
	drift := byName["drift-model"]
	if drift.ContextWindow != 99000 || !drift.Tools || !drift.ToolsKnown {
		t.Fatalf("drifted entry lost its enrichment: %+v", drift)
	}
	chat := byName["chat-model"]
	if chat.ContextWindow != 128000 || !chat.Reasoning || chat.Tools || !chat.ToolsKnown {
		t.Fatalf("chat-model: want ctx 128000, R and NOT T: %+v", chat)
	}
	tool := byName["tool-model"]
	if tool.ContextWindow != 64000 || tool.Reasoning || !tool.Tools || !tool.ToolsKnown {
		t.Fatalf("tool-model: want ctx 64000, T and NOT R: %+v", tool)
	}
	unknown := byName["unknown-model"]
	if unknown.ContextWindow != 0 || unknown.Reasoning || unknown.Tools || unknown.ToolsKnown {
		t.Fatalf("join miss must pass through UNenriched: %+v", unknown)
	}
	sparse := byName["sparse-model"]
	if sparse.ContextWindow != 32000 || sparse.Tools || sparse.ToolsKnown {
		t.Fatalf("missing capability metadata must stay unknown: %+v", sparse)
	}
	// Provider snapshots join the base model.
	dated := byName["CHAT-MODEL-2025-01-01"]
	if dated.ContextWindow != 128000 || !dated.Reasoning {
		t.Fatalf("normalization lost the dated/cased join: %+v", dated)
	}
}

/*
	OpenRouter failure keeps heuristic results.
*/

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

/*
	Empty OpenRouter data reports an error without hiding models.
*/

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

/*
	Missing data is a bad response.
*/

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

func TestListModelsSkipsEmptyIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{}, {"id":"valid"}]}`))
	}))
	defer srv.Close()

	models, err := ListModels(context.Background(), Provider{Name: "fake", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 1 || models[0].ID != "valid" {
		t.Fatalf("empty ids must be skipped: %+v", models)
	}
}

func TestListModelsRejectsOnlyEmptyIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{}, {"id":"   "}]}`))
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), Provider{Name: "fake", BaseURL: srv.URL})
	if err == nil || !strings.Contains(err.Error(), "missing an id") {
		t.Fatalf("hollow model list must error, got: %v", err)
	}
}

func TestListModelsRejectsTrailingJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{"id":"accepted"}]}{"error":"upstream failed"}`))
	}))
	defer srv.Close()

	_, err := ListModels(context.Background(), Provider{Name: "fake", BaseURL: srv.URL})
	if err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("trailing JSON must error, got: %v", err)
	}
}

func TestLoadModelsRejectsNativeNonChatIDs(t *testing.T) {
	prov := providerServing(t,
		"gpt-5-codex", "gpt-5.1-codex", "gpt-5.1-codex-max",
		"gpt-5.2-codex", "gpt-5.3-codex")
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"openai/gpt-5-codex","architecture":{"modality":"text+image->text"}},
			{"id":"openai/gpt-5.1-codex","architecture":{"modality":"text+image->text"}},
			{"id":"openai/gpt-5.1-codex-max","architecture":{"modality":"text+image->text"}},
			{"id":"openai/gpt-5.2-codex","architecture":{"modality":"text+image->text"}},
			{"id":"openai/gpt-5.3-codex","architecture":{"modality":"text->text"}}
		]}`))
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)
	Providers = map[string]Provider{
		"openai": {Name: "openai", BaseURL: prov.URL, KeyEnv: "FAKE_KEY", Dialect: OpenAI},
	}

	models, err := LoadModels(context.Background())
	if err != nil {
		t.Fatalf("LoadModels: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("native non-chat models must be dropped: %+v", models)
	}
}

/*
	Providers without keys are skipped.
*/

func TestLoadModelsSkipsKeylessProviders(t *testing.T) {
	prov := providerServing(t, "chat-model")
	nokey := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("keyless provider must not be queried")
	}))
	t.Cleanup(nokey.Close)
	or := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":[{"id":"fake/chat-model","context_length":1000,
			"architecture":{"modality":"text->text"}}]}`))
	}))
	t.Cleanup(or.Close)
	withFakeWorld(t, prov.URL, or.URL)
	Providers["nokey"] = Provider{Name: "nokey", BaseURL: nokey.URL, KeyEnv: "ARX_TEST_UNSET_KEY"}
	t.Setenv("ARX_TEST_UNSET_KEY", "")

	models, err := LoadModels(context.Background())
	if err != nil {
		t.Fatalf("a skipped provider is not an error: %v", err)
	}
	if len(models) != 1 || models[0].Model != "chat-model" {
		t.Fatalf("want only the keyed provider's model: %+v", models)
	}
}

/*
	Model filters ignore case.
*/

func TestChatCapableIsCaseInsensitive(t *testing.T) {
	for _, id := range []string{"DALL-E-3", "Whisper-1", "TTS-1-HD"} {
		if chatCapable(id) {
			t.Errorf("chatCapable(%q) = true, want false", id)
		}
	}
	if !chatCapable("GPT-5.2") {
		t.Error("chatCapable(GPT-5.2) = false, want true")
	}
	if providerChatCapable(Provider{Name: "openai"}, "GPT-5.3-CODEX-2026-02-24") {
		t.Error("native Codex snapshot must be rejected")
	}
	if !providerChatCapable(Provider{Name: "fake"}, "vendor-codex-chat") {
		t.Error("Codex names from other providers must not be rejected")
	}
}
