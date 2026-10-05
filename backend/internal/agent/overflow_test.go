package agent

import (
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContextOverflowRetriesOnceWithoutReplayingTools(t *testing.T) {
	for _, test := range []struct {
		name          string
		failure       error
		partial       bool
		repeat        bool
		wantRequests  int
		wantSummaries int
	}{
		{name: "recover", failure: provider.ErrContextLength, wantRequests: 2, wantSummaries: 1},
		{name: "repeat", failure: provider.ErrContextLength, repeat: true, wantRequests: 2, wantSummaries: 1},
		{name: "settings", failure: errors.New("invalid settings"), wantRequests: 1},
		{name: "partial", failure: provider.ErrContextLength, partial: true, wantRequests: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests, summaries := 0, 0
			var measured *session.ContextUsage
			loop := Loop{Save: func(chunk session.TranscriptChunk) error {
				if chunk.Context != nil {
					measured = chunk.Context
				}
				return nil
			}, Emit: func(session.TranscriptChunk) {}, SaveCompaction: func(session.Compaction) error { return nil }, Complete: func(_ context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
				if len(request.Tools) == 0 {
					summaries++
					return provider.Response{Message: provider.Message{Content: "Earlier work finished."}}, nil
				}
				requests++
				if requests == 1 || test.repeat {
					if test.partial {
						emit(provider.Delta{Text: "Partial"})
					}
					return provider.Response{}, test.failure
				}
				if err := emit(provider.Delta{Text: "Done"}); err != nil {
					return provider.Response{}, err
				}
				return provider.Response{Message: provider.Message{Role: "assistant", Content: "Done"}, Usage: &session.ChunkUsage{Input: 1500, Cached: 1000}}, nil
			}}
			history := []provider.Message{{Role: "user", Content: strings.Repeat("Earlier task ", 500)}, {Role: "assistant", Content: "Done"}, {Role: "user", Content: "Continue"}}
			err := loop.Run(context.Background(), Input{Conversation: "test", Model: model.Info{ID: "test", ContextWindow: 32000, MaxOutputTokens: 4000}, History: history})
			if requests != test.wantRequests || summaries != test.wantSummaries {
				t.Fatal(requests, summaries, err)
			}
			if test.name == "recover" {
				if err != nil || measured == nil || measured.Input != 1500 {
					t.Fatal("recovery or actual usage missing", err, measured)
				}
				if contextwindow.Latest([]session.TranscriptChunk{{Context: measured}}).Input != 1500 {
					t.Fatal("cached input was subtracted")
				}
			} else if err == nil {
				t.Fatal("failure hidden")
			}
		})
	}
}
