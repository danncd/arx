package agent

import (
	provider "arx/internal/inference"
	attachment "arx/internal/media/attachments"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/sessions/storage"
	tool "arx/internal/tools"
	"context"
	"reflect"
	"testing"
)

func TestToolImageReachesNextRoundAndSavedHistory(t *testing.T) {
	directory := t.TempDir()
	store, err := storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	user := session.TranscriptChunk{ID: "user", Conversation: "chat", Role: "user", Text: "Inspect picture"}
	store.Append(user)
	history, _ := History([]session.TranscriptChunk{user})
	images := []attachment.Image{{ID: "image-reference", Name: "picture.png"}}
	calls := 0
	var sent []provider.Message
	loop := Loop{Save: store.Append, Emit: func(session.TranscriptChunk) {}, Tool: func(context.Context, tool.Call) tool.Result { return tool.Result{Text: "Image loaded", Images: images} }, Complete: func(_ context.Context, request provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if calls == 1 {
			return provider.Response{Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: "image-call", Type: "function", Function: provider.Function{Name: "web", Arguments: `{"operation":"image","url":"https://example.com/image.png"}`}}}}}, nil
		}
		last := request.Messages[len(request.Messages)-1]
		if last.Role != "tool" || last.CallID != "image-call" || !reflect.DeepEqual(last.Images, images) {
			t.Fatal("tool pixels missing from model request")
		}
		sent = append([]provider.Message(nil), request.Messages[1:]...)
		answer := provider.Message{Role: "assistant", Content: "The image shows a square"}
		sent = append(sent, answer)
		return provider.Response{Message: answer}, nil
	}}
	if err := loop.Run(context.Background(), Input{Model: model.Info{Vision: true, ContextWindow: 1048576, MaxOutputTokens: 32768}, Conversation: "chat", History: history}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	replay, err := History(store.Entries("chat"))
	if err != nil || !reflect.DeepEqual(sent, replay) {
		t.Fatal("image replay changed the request prefix", err)
	}
}
