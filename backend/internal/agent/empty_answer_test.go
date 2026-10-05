package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/settings"
	"context"
	"strings"
	"testing"
)

func TestReasoningOnlyReplyRetriesWithoutThinking(t *testing.T) {
	var chunks []session.TranscriptChunk
	calls := 0
	loop := Loop{
		Save: func(chunk session.TranscriptChunk) error {
			chunks = append(chunks, chunk)
			return nil
		},
		Emit: func(session.TranscriptChunk) {},
		Complete: func(_ context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
			calls++
			if calls == 1 {
				if err := emit(provider.Delta{Reasoning: "I should answer."}); err != nil {
					return provider.Response{}, err
				}
				return provider.Response{Message: provider.Message{Role: "assistant", Reasoning: "I should answer."}}, nil
			}
			if request.Effort != "none" || request.Messages[len(request.Messages)-1].Role != "user" || !strings.Contains(request.Messages[len(request.Messages)-1].Content, "direct answer") {
				t.Fatalf("retry request = %+v", request)
			}
			return provider.Response{Message: provider.Message{Role: "assistant", Content: "Here is the answer."}}, nil
		},
	}
	input := Input{
		Conversation: "chat",
		Model:        model.Info{ContextWindow: 16384, MaxOutputTokens: 4096, Thinking: &model.Thinking{CanDisable: true}},
		Settings:     settings.Values{Run: settings.Run{Provider: "local", Model: "local:test", Effort: "default"}},
		History:      []provider.Message{{Role: "user", Content: "Answer me"}},
	}
	if err := loop.Run(context.Background(), input); err != nil || calls != 2 {
		t.Fatalf("calls = %d, error = %v", calls, err)
	}
	history, err := History(chunks)
	if err != nil || len(history) != 2 || history[1].Content != "Here is the answer." {
		t.Fatalf("history = %+v, error = %v", history, err)
	}
	if last := chunks[len(chunks)-1]; last.Status != "done" {
		t.Fatalf("last chunk = %+v", last)
	}
}

func TestRepeatedReasoningOnlyReplyFailsClearly(t *testing.T) {
	var chunks []session.TranscriptChunk
	calls := 0
	loop := Loop{
		Save: func(chunk session.TranscriptChunk) error {
			chunks = append(chunks, chunk)
			return nil
		},
		Emit: func(session.TranscriptChunk) {},
		Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
			calls++
			return provider.Response{Message: provider.Message{Role: "assistant", Reasoning: "Still thinking."}}, nil
		},
	}
	input := Input{Conversation: "chat", Model: model.Info{ContextWindow: 16384, MaxOutputTokens: 4096}, History: []provider.Message{{Role: "user", Content: "Answer me"}}}
	err := loop.Run(context.Background(), input)
	if calls != 2 || err == nil || !strings.Contains(err.Error(), "without giving an answer") {
		t.Fatalf("calls = %d, error = %v", calls, err)
	}
	if last := chunks[len(chunks)-1]; last.Status != "failed" || !last.Failed {
		t.Fatalf("last chunk = %+v", last)
	}
}
