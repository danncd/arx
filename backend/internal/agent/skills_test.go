package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	tool "arx/internal/tools"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestLoadedSkillGuidanceSurvivesCompaction(t *testing.T) {
	loaded := ""
	calls := 0
	summaries := 0
	skill := strings.Repeat("Follow the user's requested workflow. ", 30)
	loop := Loop{Guidance: func() string { return loaded }, Save: func(session.TranscriptChunk) error { return nil }, Emit: func(session.TranscriptChunk) {}, SaveCompaction: func(session.Compaction) error { return nil }, Tool: func(context.Context, tool.Call) tool.Result {
		loaded = "\nLoaded skill:\n" + skill
		return tool.Output(strings.Repeat("Large tool result. ", 500))
	}, Complete: func(_ context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		if len(request.Tools) == 0 {
			summaries++
			return provider.Response{Message: provider.Message{Role: "assistant", Content: "The user asked to use the skill."}}, nil
		}
		calls++
		if calls == 1 {
			return provider.Response{Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: "load", Type: "function", Function: provider.Function{Name: "skills", Arguments: `{"operation":"load","id":"test"}`}}}}}, nil
		}
		if !strings.Contains(request.Messages[0].Content, skill) {
			t.Fatal("loaded instructions lost during compaction")
		}
		emit(provider.Delta{Text: "Finished"})
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Finished"}}, nil
	}}
	history := []provider.Message{}
	for i := 0; i < 20; i++ {
		for _, role := range []string{"user", "assistant"} {
			history = append(history, provider.Message{Role: role, Content: fmt.Sprint(i) + strings.Repeat("Earlier project context. ", 100)})
		}
	}
	history = append(history, provider.Message{Role: "user", Content: "Use the workflow."})
	if err := loop.Run(context.Background(), Input{Model: model.Info{ID: "test", ContextWindow: 16000, MaxOutputTokens: 4000}, History: history}); err != nil {
		t.Fatal(err)
	}
	if summaries < 2 {
		t.Fatal("test did not compact")
	}
}
