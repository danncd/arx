package network

import (
	provider "arx/internal/inference"
	tool "arx/internal/tools"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const fixture = `{"models":[{"type":"llm","key":"qwen/test","display_name":"Qwen","max_context_length":262144,"loaded_instances":[{"id":"qwen/test","config":{"context_length":8192}}],"capabilities":{"vision":true,"trained_for_tool_use":true}},{"type":"embedding","key":"embed"}]}`

func TestDiscoveryUsesLoadedContextAndExcludesEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(fixture)) }))
	defer server.Close()
	got, err := discover(context.Background(), server.URL+"/v1", "")
	if err != nil || len(got.Models) != 1 || got.Models[0].Info.ContextWindow != 8192 || !got.Models[0].Info.Vision {
		t.Fatalf("invalid discovery: %+v %v", got, err)
	}
}

func TestDiscoveryReservesMoreOutputForLargerActiveContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Replace(fixture, `"context_length":8192`, `"context_length":20156`, 1)))
	}))
	defer server.Close()
	got, err := discover(context.Background(), server.URL, "")
	if err != nil || len(got.Models) != 1 || got.Models[0].Info.MaxOutputTokens <= 4096 {
		t.Fatalf("output limit did not reflect active context: %+v %v", got.Models, err)
	}
}
func TestServerPersistsWithoutTokenAndStreams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/models" {
			w.Write([]byte(fixture))
			return
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Error(r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()
	directory := t.TempDir()
	m, _ := Open(directory)
	connected, err := m.Connect(context.Background(), server.URL, "PC", "")
	if err != nil {
		t.Fatal(err)
	}
	m, err = Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	state := m.State(context.Background())
	if len(state) != 1 || state[0].Name != "PC" {
		t.Fatal(state)
	}
	result, err := m.Complete(context.Background(), provider.Request{Model: connected.Models[0].Info.ID, MaxOutputTokens: 10, Messages: []provider.Message{{Role: "user", Content: "Hello"}}}, nil, func(provider.Delta) error { return nil })
	if err != nil || result.Message.Content != "Hello" {
		t.Fatalf("%+v %v", result, err)
	}
	if err = m.Remove(context.Background(), connected.ID); err != nil || len(m.State(context.Background())) != 0 {
		t.Fatal(err)
	}
}
func TestDiscoveryRejectsUnrelatedServicesAndRecognizesAuthentication(t *testing.T) {
	for _, status := range []int{200, 401, 403} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"unrelated":true}`))
		}))
		got, err := discover(context.Background(), server.URL, "")
		server.Close()
		if status == 200 && err == nil {
			t.Fatal("accepted unrelated service")
		}
		if status != 200 && (err != nil || !got.RequiresToken || got.Connected) {
			t.Fatal("authentication state lost")
		}
	}
	for _, address := range []string{"file:///tmp/test", "http://user:password@host", "http://host/path", "http://host?token=secret"} {
		if _, err := normalize(address); err == nil {
			t.Fatal(address)
		}
	}
}
func TestLiveNetwork(t *testing.T) {
	address := os.Getenv("ARX_TEST_NETWORK_URL")
	if address == "" {
		t.Skip("No test server")
	}
	m, _ := Open(t.TempDir())
	server, err := m.Connect(context.Background(), address, "My PC", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range server.Models {
		if item.Loaded {
			count := 0
			result, err := m.Complete(context.Background(), provider.Request{Model: item.Info.ID, Effort: "none", MaxOutputTokens: 128, Messages: []provider.Message{{Role: "user", Content: "Reply only with: Network connection works."}}}, nil, func(d provider.Delta) error {
				if d.Text != "" {
					count++
				}
				return nil
			})
			if err != nil || count == 0 || strings.TrimSpace(result.Message.Content) == "" {
				t.Fatalf("no streamed reply: %+v %v", result, err)
			}
			t.Logf("%s: %d streamed chunks, %s", item.Info.Name, count, result.Message.Content)
			result, err = m.Complete(context.Background(), provider.Request{Model: item.Info.ID, Effort: "none", MaxOutputTokens: 256, Tools: tool.Definitions(), Messages: []provider.Message{{Role: "user", Content: "Call the files tool to list the current directory. Do not answer without calling the tool."}}}, nil, func(provider.Delta) error { return nil })
			if err != nil || len(result.Message.Calls) == 0 {
				t.Fatalf("No tool call: %+v %v", result, err)
			}
			t.Log("Tool call received:", result.Message.Calls[0].Function.Name)
			return
		}
	}
	t.Fatal("No loaded model")
}
