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
