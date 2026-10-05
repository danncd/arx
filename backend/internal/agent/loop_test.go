package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/sessions/storage"
	tool "arx/internal/tools"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestLoopPersistsAndReplaysReasoningToolResultsAndUnicode(t *testing.T) {
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user := session.TranscriptChunk{ID: "user", Conversation: "session", Role: "user", Text: "Read notes"}
	if err := store.Append(user); err != nil {
		t.Fatal(err)
	}
	history, _ := History([]session.TranscriptChunk{user})
	calls := 0
	var expected []provider.Message
	loop := Loop{Save: store.Append, Emit: func(session.TranscriptChunk) {}, Tool: func(_ context.Context, call tool.Call) tool.Result {
		if call.Name != "files" {
			t.Fatal("wrong tool")
		}
		return tool.Result{Text: "Permission denied", Failed: true}
	}, Complete: func(_ context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		calls++
		if len(request.Tools) != len(tool.Definitions()) {
			return provider.Response{}, errors.New("tools missing")
		}
		if calls == 1 {
			for _, delta := range []provider.Delta{{Reasoning: "Thought 🌱"}, {Text: "Reading"}} {
				if err := emit(delta); err != nil {
					return provider.Response{}, err
				}
			}
			return provider.Response{Message: provider.Message{Role: "assistant", Content: "Reading", Reasoning: "Thought 🌱", Calls: []provider.Call{{ID: "call", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"notes"}`}}}}}, nil
		}
		if len(request.Messages) != 4 || request.Messages[2].Reasoning != "Thought 🌱" || !strings.Contains(request.Messages[3].Content, "Permission denied") {
			t.Fatalf("lost tool history: %+v", request.Messages)
		}
		expected = append([]provider.Message(nil), request.Messages[1:]...)
		if err := emit(provider.Delta{Text: "I could not read it."}); err != nil {
			return provider.Response{}, err
		}
		message := provider.Message{Role: "assistant", Content: "I could not read it."}
		expected = append(expected, message)
		return provider.Response{Message: message, Usage: &session.ChunkUsage{Input: 30, Output: 9}}, nil
	}}
	if err := loop.Run(context.Background(), Input{Model: model.Info{ContextWindow: 1048576, MaxOutputTokens: 32768}, Conversation: "session", History: history}); err != nil {
		t.Fatal(err)
	}
	restored, err := History(store.Entries("session"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, expected) {
		t.Fatalf("replay differs\n%+v\n%+v", restored, expected)
	}
	entries := store.Entries("session")
	last := entries[len(entries)-1]
	if last.Status != "done" || last.Usage.Output != 9 {
		t.Fatal("final status or usage lost")
	}
}

func TestCancellationRetainsPartialReplyAndNeverStartsATool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var chunks []session.TranscriptChunk
	loop := Loop{Save: func(chunk session.TranscriptChunk) error { chunks = append(chunks, chunk); return nil }, Emit: func(session.TranscriptChunk) {}, Tool: func(context.Context, tool.Call) tool.Result {
		t.Fatal("tool ran after cancellation")
		return tool.Result{}
	}, Complete: func(ctx context.Context, _ provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		if err := emit(provider.Delta{Text: "Partial 🌱", Reasoning: "Thinking"}); err != nil {
			return provider.Response{}, err
		}
		cancel()
		return provider.Response{}, ctx.Err()
	}}
	if err := loop.Run(ctx, Input{Model: model.Info{ContextWindow: 1048576, MaxOutputTokens: 32768}, Conversation: "session"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	restored, err := History(chunks)
	if err != nil || restored[0].Content != "Partial 🌱" {
		t.Fatalf("lost partial reply: %v %+v", err, restored)
	}
	if chunks[len(chunks)-1].Status != "cancelled" {
		t.Fatal("missing cancellation status")
	}
}

func TestStorageFailureStopsBeforeToolSideEffect(t *testing.T) {
	writes := 0
	loop := Loop{Save: func(session.TranscriptChunk) error {
		writes++
		if writes > 1 {
			return errors.New("disk full")
		}
		return nil
	}, Emit: func(session.TranscriptChunk) {}, Tool: func(context.Context, tool.Call) tool.Result {
		t.Fatal("tool ran without saved call")
		return tool.Result{}
	}, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		return provider.Response{Message: provider.Message{Calls: []provider.Call{{ID: "x", Function: provider.Function{Name: "bash", Arguments: `{"command":"true"}`}}}}}, nil
	}}
	if err := loop.Run(context.Background(), Input{Model: model.Info{ContextWindow: 1048576, MaxOutputTokens: 32768}, Conversation: "session"}); err == nil {
		t.Fatal("storage failure ignored")
	}
}

func TestLegacyToolBoundariesAndInterruptedTools(t *testing.T) {
	chunks := []session.TranscriptChunk{{ID: "a", Role: "assistant", Text: "BeforeAfter", Reasoning: "ThinkMore", Tools: []session.ToolRecord{{ID: "t", Name: "files", Arguments: `{}`, Offset: 6, ThoughtAt: 5, Status: "running"}}}}
	messages, err := History(chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 || messages[0].Content != "Before" || messages[0].Reasoning != "Think" || messages[2].Content != "After" || !strings.Contains(messages[1].Content, "unknown") {
		t.Fatalf("incorrect reconstruction: %+v", messages)
	}
}
