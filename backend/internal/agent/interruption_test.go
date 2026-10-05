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

func TestOutputLimitRetainsUnfinishedToolAsContextOnly(t *testing.T) {
	var chunks []session.TranscriptChunk
	loop := Loop{
		Complete: func(ctx context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
			emit(provider.Delta{Text: "Creating the game."})
			emit(provider.Delta{Tool: &provider.ToolDelta{Index: 0, Call: provider.Call{ID: "write", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"write","path":"game.html","content":"<html>`}}}})
			return provider.Response{}, provider.ErrOutputLimit
		},
		Save: func(chunk session.TranscriptChunk) error { chunks = append(chunks, chunk); return nil },
		Emit: func(session.TranscriptChunk) {},
	}
	err := loop.Run(context.Background(), Input{Model: model.Info{ContextWindow: 32000, MaxOutputTokens: 8000}, Settings: settings.Values{}, Conversation: "test"})
	if err == nil {
		t.Fatal("missing output limit")
	}
	history, err := History(chunks)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, message := range history {
		if len(message.Calls) != 0 {
			t.Fatal("replayed unfinished tool")
		}
		text.WriteString(message.Content)
	}
	if !strings.Contains(text.String(), "game.html") || !strings.Contains(text.String(), "not executed") {
		t.Fatalf("lost interruption: %s", text.String())
	}
}
