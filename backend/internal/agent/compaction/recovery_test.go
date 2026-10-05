package compaction

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOversizedNewestToolExchangeRecoversWithoutChangingSavedDataOrUserRequest(t *testing.T) {
	user := "Read the file and identify errors; do not change it."
	large := strings.Repeat("large source output ", 20000)
	history := []provider.Message{{Role: "user", Content: user}, {Role: "assistant", Calls: []provider.Call{{ID: "read", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"source"}`}}}}, {Role: "tool", CallID: "read", Content: large}}
	for _, enabled := range []bool{false, true} {
		attempts := 0
		manager := Compactor{Model: model.Info{ID: "test", ContextWindow: 32000, MaxOutputTokens: 4000}, AutoRecover: enabled, Recovery: func(kind string, n int) {
			if kind != "context" || n > 3 {
				t.Fatal(kind, n)
			}
			attempts++
		}, Save: func(state session.Compaction) error {
			if !Valid(state, history) {
				t.Fatal("invalid saved checkpoint")
			}
			return nil
		}, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
			return provider.Response{Message: provider.Message{Content: "Earlier context summarized."}}, nil
		}}
		result, err := manager.Prepare(context.Background(), request(history))
		if !enabled {
			if err == nil {
				t.Fatal("disabled recovery unexpectedly succeeded")
			}
			continue
		}
		if err != nil || attempts < 1 || attempts > 3 {
			t.Fatal(err, attempts)
		}
		foundUser, foundTool, foundCall := false, false, false
		for _, m := range result.Messages {
			if m.Role == "user" && m.Content == user {
				foundUser = true
			}
			if m.Role == "tool" && m.CallID == "read" && len(m.Content) < len(large) && strings.Contains(m.Content, "context_recovery") {
				foundTool = true
			}
			if len(m.Calls) > 0 && m.Calls[0].Function.Arguments == history[1].Calls[0].Function.Arguments {
				foundCall = true
			}
		}
		if !foundUser || !foundTool || !foundCall || history[2].Content != large {
			t.Fatal("changed user instructions, saved results, tool arguments, or missing recovery")
		}
	}
}
func TestOversizedUserRequestIsPreservedAndRecoveryStopsAfterThreeAttempts(t *testing.T) {
	user := strings.Repeat("User instruction ", 20000)
	attempts := 0
	manager := Compactor{Model: model.Info{ID: "test", ContextWindow: 32000, MaxOutputTokens: 4000}, AutoRecover: true, Recovery: func(_ string, _ int) { attempts++ }}
	_, err := manager.Prepare(context.Background(), request([]provider.Message{{Role: "user", Content: user}}))
	if !errors.Is(err, provider.ErrContextLength) || attempts != 3 {
		t.Fatal(err, attempts)
	}
}
