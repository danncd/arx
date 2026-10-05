package chatcompletions

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNetworkRequestMapsThinkingAndKeepsActualModelID(t *testing.T) {
	for _, effort := range []string{"on", "none", "high"} {
		t.Run(effort, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				expected := effort
				if expected == "on" {
					expected = "medium"
				}
				if body["reasoning_effort"] != expected || body["model"] != "qwen3.5-4b" {
					t.Errorf("Wrong request: %+v", body)
				}
				if _, ok := body["cache_prompt"]; ok {
					t.Error("Local-only option sent to network server")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ready\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"))
			}))
			defer server.Close()
			client := Client{Profile: NetworkWire, URL: server.URL, RemoteModel: "qwen3.5-4b", Info: model.Info{Thinking: &model.Thinking{CanDisable: true}}}
			result, err := client.Complete(context.Background(), provider.Request{Effort: effort, MaxOutputTokens: 128, Messages: []provider.Message{{Role: "user", Content: "ready"}}}, func(provider.Delta) error { return nil })
			if err != nil || result.Message.Content != "ready" {
				t.Fatal(result, err)
			}
		})
	}
}
