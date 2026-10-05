package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/sessions/storage"
	"arx/internal/settings"
	tool "arx/internal/tools"
	"context"
	"reflect"
	"testing"
)

func TestRuntimeChangesAppendWithoutRewritingSystem(t *testing.T) {
	chunks := []session.TranscriptChunk{{ID: "a", Role: "user", Text: "Start", Runtime: &session.RuntimeContext{Directory: "/first", Date: "2026-09-26"}}}
	first, _ := History(chunks)
	before := Request(settings.Values{Directory: "/first"}, first)
	chunks = append(chunks, session.TranscriptChunk{ID: "b", Role: "user", Text: "Continue", Runtime: &session.RuntimeContext{Directory: "/second", Date: "2026-09-27"}})
	second, _ := History(chunks)
	after := Request(settings.Values{Directory: "/second"}, second)
	if !reflect.DeepEqual(before.Messages, after.Messages[:len(before.Messages)]) || !reflect.DeepEqual(before.Tools, after.Tools) {
		t.Fatal("runtime change rewrote system or history")
	}
}

func TestStreamedToolOffsetsDoNotChangeProviderReplay(t *testing.T) {
	directory := t.TempDir()
	store, err := storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	user := session.TranscriptChunk{Conversation: "chat", ID: "u", Role: "user", Text: "Read both"}
	if err := store.Append(user); err != nil {
		t.Fatal(err)
	}
	history, _ := History([]session.TranscriptChunk{user})
	calls := []provider.Call{
		{ID: "one", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"a"}`}},
		{ID: "two", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"b"}`}},
	}
	round := 0
	var expected []provider.Message
	loop := Loop{Save: store.Append, Emit: func(session.TranscriptChunk) {}, Tool: func(context.Context, tool.Call) tool.Result { return tool.Result{Text: "result"} }, Complete: func(_ context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		round++
		if round == 1 {
			deltas := []provider.Delta{
				{Text: "Before", Reasoning: "Think"},
				{Tool: &provider.ToolDelta{Index: 0, Call: calls[0]}},
				{Text: " between", Reasoning: " more"},
				{Tool: &provider.ToolDelta{Index: 1, Call: calls[1]}},
				{Text: " after"},
			}
			for _, delta := range deltas {
				if err := emit(delta); err != nil {
					return provider.Response{}, err
				}
			}
			return provider.Response{Message: provider.Message{Role: "assistant", Content: "Before between after", Reasoning: "Think more", Calls: calls}}, nil
		}
		message := provider.Message{Role: "assistant", Content: "Done", Reasoning: "All checked"}
		expected = append(append([]provider.Message{}, request.Messages[1:]...), message)
		return provider.Response{Message: message}, nil
	}}
	if err := loop.Run(context.Background(), Input{Conversation: "chat", History: history, Model: model.Info{ContextWindow: 1048576, MaxOutputTokens: 32768}}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	reopened, err := storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	replay, err := History(reopened.Entries("chat"))
	if err != nil || !reflect.DeepEqual(replay, expected) {
		t.Fatalf("provider replay changed: %v\n%+v\n%+v", err, replay, expected)
	}
}
