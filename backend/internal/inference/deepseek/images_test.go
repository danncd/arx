package deepseek

import (
	provider "arx/internal/inference"
	attachment "arx/internal/media/attachments"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestVisionWireRequestAndReplay(t *testing.T) {
	var previous json.RawMessage
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []json.RawMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		var user struct {
			Role    string `json:"role"`
			Content []struct {
				Type  string `json:"type"`
				Text  string `json:"text"`
				Image struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		}
		if err := json.Unmarshal(body.Messages[0], &user); err != nil {
			t.Error(err)
		}
		if user.Role != "user" || len(user.Content) != 3 || user.Content[0].Text != "Describe it" || !strings.HasPrefix(user.Content[2].Image.URL, "data:image/png;base64,") {
			t.Error("image did not reach wire payload")
		}
		if calls > 0 && !reflect.DeepEqual(previous, body.Messages[0]) {
			t.Error("replay changed image prefix")
		}
		previous = append(json.RawMessage(nil), body.Messages[0]...)
		calls++
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"A square\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	client := &Client{client: server.Client(), baseURL: server.URL, ReadImage: func(id string) ([]byte, error) {
		if id != "saved-image" {
			t.Error("wrong image requested")
		}
		return []byte("fixture"), nil
	}}
	request := provider.Request{Model: "deepseek-flash", Messages: []provider.Message{{Role: "user", Content: "Describe it", Images: []attachment.Image{{ID: "saved-image", Name: "square.png"}}}}}
	for range 2 {
		if _, err := client.Complete(context.Background(), "test", request, func(provider.Delta) error { return nil }); err != nil {
			t.Fatal(err)
		}
		request.Messages = append(request.Messages, provider.Message{Role: "assistant", Content: "A square"})
	}
	request.Model = "deepseek-v4-pro"
	if _, err := client.messages(request); err == nil {
		t.Fatal("accepted images for a text-only model")
	}
}

func TestToolImageKeepsItsCallIDOnTheWire(t *testing.T) {
	client := &Client{ReadImage: func(string) ([]byte, error) { return []byte("image"), nil }}
	request := provider.Request{Model: "deepseek-flash", Messages: []provider.Message{{Role: "tool", CallID: "call-image", Content: "Loaded image", Images: []attachment.Image{{ID: "saved"}}}}}
	messages, err := client.messages(request)
	if err != nil {
		t.Fatal(err)
	}
	encoded := messages[0].(map[string]any)
	if encoded["role"] != "tool" || encoded["tool_call_id"] != "call-image" || len(encoded["content"].([]any)) != 3 {
		t.Fatal("tool image was serialized incorrectly")
	}
}
