package chatcompletions

import (
	provider "arx/internal/inference"
	attachment "arx/internal/media/attachments"
	model "arx/internal/models"
	"testing"
)

func TestToolImagesFollowCompleteToolBatch(t *testing.T) {
	client := Client{Info: model.Info{Vision: true}, ReadImage: func(string) ([]byte, error) { return []byte("image"), nil }}
	request := provider.Request{Messages: []provider.Message{
		{Role: "assistant", Calls: []provider.Call{{ID: "a"}, {ID: "b"}}},
		{Role: "tool", CallID: "a", Content: "first", Images: []attachment.Image{{ID: "a"}}},
		{Role: "tool", CallID: "b", Content: "second", Images: []attachment.Image{{ID: "b"}}},
		{Role: "assistant", Content: "seen"},
	}}
	messages, err := client.messages(request)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"assistant", "tool", "tool", "user", "assistant"}
	if len(messages) != len(expected) {
		t.Fatal(messages)
	}
	for i, role := range expected {
		if messages[i].(map[string]any)["role"] != role {
			t.Fatalf("Message %d interrupted the tool batch", i)
		}
	}
	for i, id := range []string{"a", "b"} {
		if messages[i+1].(map[string]any)["tool_call_id"] != id {
			t.Fatal("Lost tool association")
		}
	}
	if len(messages[3].(map[string]any)["content"].([]any)) != 6 {
		t.Fatal("Missing grouped images")
	}
}
